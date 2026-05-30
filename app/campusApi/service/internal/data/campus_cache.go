package data

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"lehu-video/app/campusApi/service/internal/biz"
)

const (
	campusCachePrefix        = "campus:cache:v1"
	campusPostFeedVersionKey = campusCachePrefix + ":postfeed:version"
	campusMomentsVersionKey  = campusCachePrefix + ":moments:version"
)

type campusPostListCache struct {
	IDs   []int64 `json:"ids"`
	Total int64   `json:"total"`
}

func (r *campusRepo) cacheEnabled() bool {
	if r == nil || r.data == nil || r.data.rds == nil {
		return false
	}
	return !envBoolFalseData(os.Getenv("LEHU_REDIS_CACHE_ENABLED"))
}

func (r *campusRepo) getCacheJSON(ctx context.Context, key string, out any) bool {
	if !r.cacheEnabled() {
		return false
	}
	raw, err := r.data.rds.Get(ctx, key).Bytes()
	if err != nil {
		if err != redis.Nil {
			r.log.WithContext(ctx).Warnf("redis cache get failed: key=%s err=%v", key, err)
		}
		return false
	}
	if err := json.Unmarshal(raw, out); err != nil {
		r.log.WithContext(ctx).Warnf("redis cache decode failed: key=%s err=%v", key, err)
		return false
	}
	return true
}

func (r *campusRepo) setCacheJSON(ctx context.Context, key string, value any, ttl time.Duration) {
	if !r.cacheEnabled() || ttl <= 0 {
		return
	}
	raw, err := json.Marshal(value)
	if err != nil {
		r.log.WithContext(ctx).Warnf("redis cache encode failed: key=%s err=%v", key, err)
		return
	}
	if err := r.data.rds.Set(ctx, key, raw, ttl).Err(); err != nil {
		r.log.WithContext(ctx).Warnf("redis cache set failed: key=%s err=%v", key, err)
	}
}

func (r *campusRepo) deleteCacheKeys(ctx context.Context, keys ...string) {
	if !r.cacheEnabled() || len(keys) == 0 {
		return
	}
	if err := r.data.rds.Del(ctx, keys...).Err(); err != nil {
		r.log.WithContext(ctx).Warnf("redis cache delete failed: keys=%v err=%v", keys, err)
	}
}

func (r *campusRepo) bumpPostFeedCacheVersion(ctx context.Context) {
	if !r.cacheEnabled() {
		return
	}
	if err := r.data.rds.Incr(ctx, campusPostFeedVersionKey).Err(); err != nil {
		r.log.WithContext(ctx).Warnf("redis post feed version bump failed: err=%v", err)
	}
}

func (r *campusRepo) bumpMomentsCacheVersion(ctx context.Context) {
	if !r.cacheEnabled() {
		return
	}
	if err := r.data.rds.Incr(ctx, campusMomentsVersionKey).Err(); err != nil {
		r.log.WithContext(ctx).Warnf("redis moments version bump failed: err=%v", err)
	}
}

func (r *campusRepo) postFeedCacheVersion(ctx context.Context) int64 {
	if !r.cacheEnabled() {
		return 0
	}
	version, err := r.data.rds.Get(ctx, campusPostFeedVersionKey).Int64()
	if err == nil {
		return version
	}
	if err != redis.Nil {
		r.log.WithContext(ctx).Warnf("redis post feed version get failed: err=%v", err)
	}
	return 0
}

func (r *campusRepo) momentsCacheVersion(ctx context.Context) int64 {
	if !r.cacheEnabled() {
		return 0
	}
	version, err := r.data.rds.Get(ctx, campusMomentsVersionKey).Int64()
	if err == nil {
		return version
	}
	if err != redis.Nil {
		r.log.WithContext(ctx).Warnf("redis moments version get failed: err=%v", err)
	}
	return 0
}

func (r *campusRepo) invalidatePostDetailCache(ctx context.Context, postID int64) {
	if postID <= 0 {
		return
	}
	r.deleteCacheKeys(ctx, campusPostDetailCacheKey(postID))
}

func (r *campusRepo) invalidatePostReadCaches(ctx context.Context, postID int64, bumpFeed bool) {
	r.invalidatePostDetailCache(ctx, postID)
	if bumpFeed {
		r.bumpPostFeedCacheVersion(ctx)
		r.bumpMomentsCacheVersion(ctx)
	}
	r.deleteCacheKeys(ctx, campusAdminSummaryCacheKey())
}

func (r *campusRepo) shouldCachePostList(query biz.ListCampusPostQuery) bool {
	if !r.cacheEnabled() {
		return false
	}
	if query.IncludeDeleted || query.Keyword != "" || query.AuthorID != "" || query.CollectedByUserID != "" || query.OnlyReported {
		return false
	}
	if query.OnlyOfficial != nil || query.OnlyFeatured != nil || query.OnlyPinned != nil {
		return false
	}
	if len(query.Statuses) != 1 || query.Statuses[0] != biz.CampusAuditStatusVisible {
		return false
	}
	if query.Limit <= 0 || query.Offset < 0 {
		return false
	}
	page := query.Offset/query.Limit + 1
	return page >= 1 && page <= 3
}

func (r *campusRepo) postListCacheKey(ctx context.Context, query biz.ListCampusPostQuery) string {
	version := r.postFeedCacheVersion(ctx)
	payload := fmt.Sprintf("v=%d|category=%s|post_type=%s|sort=%s|offset=%d|limit=%d",
		version,
		strings.TrimSpace(query.CategoryCode),
		strings.TrimSpace(query.PostType),
		strings.TrimSpace(query.Sort),
		query.Offset,
		query.Limit,
	)
	sum := sha1.Sum([]byte(payload))
	return campusCachePrefix + ":postfeed:" + hex.EncodeToString(sum[:])
}

func campusPostDetailCacheKey(postID int64) string {
	return fmt.Sprintf("%s:post:%d", campusCachePrefix, postID)
}

func campusCategoriesCacheKey() string {
	return campusCachePrefix + ":categories"
}

func campusAdminSummaryCacheKey() string {
	return campusCachePrefix + ":admin:summary"
}

func campusSecurityOverviewCacheKey() string {
	return campusCachePrefix + ":admin:security-overview"
}

func (r *campusRepo) momentsCandidatesCacheKey(ctx context.Context, start, end time.Time, limit int) string {
	return fmt.Sprintf("%s:moments:candidates:%d:%s:%s:%d",
		campusCachePrefix,
		r.momentsCacheVersion(ctx),
		start.UTC().Format("20060102T150405Z"),
		end.UTC().Format("20060102T150405Z"),
		limit,
	)
}

func campusCacheTTL(envName string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(envName))
	if value == "" {
		return fallback
	}
	ttl, err := time.ParseDuration(value)
	if err == nil && ttl > 0 {
		return ttl
	}
	if seconds, err := strconv.Atoi(value); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	return fallback
}

func campusPostListCacheTTL() time.Duration {
	return campusCacheTTL("LEHU_CACHE_POST_LIST_TTL", 10*time.Second)
}

func campusPostDetailCacheTTL() time.Duration {
	return campusCacheTTL("LEHU_CACHE_POST_DETAIL_TTL", 30*time.Second)
}

func campusAdminSummaryCacheTTL() time.Duration {
	return campusCacheTTL("LEHU_CACHE_ADMIN_SUMMARY_TTL", 60*time.Second)
}

func campusSecurityOverviewCacheTTL() time.Duration {
	return campusCacheTTL("LEHU_CACHE_SECURITY_OVERVIEW_TTL", 60*time.Second)
}

func campusCategoriesCacheTTL() time.Duration {
	return campusCacheTTL("LEHU_CACHE_CATEGORIES_TTL", 30*time.Minute)
}

func campusMomentsCandidatesCacheTTL() time.Duration {
	return campusCacheTTL("LEHU_CACHE_MOMENTS_CANDIDATES_TTL", 3*time.Minute)
}

func envBoolFalseData(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "0", "false", "no", "off", "disabled":
		return true
	default:
		return false
	}
}
