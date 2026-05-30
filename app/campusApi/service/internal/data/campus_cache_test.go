package data

import (
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"lehu-video/app/campusApi/service/internal/biz"
)

func TestCampusCacheTTLParsesDurationAndSeconds(t *testing.T) {
	t.Setenv("LEHU_TEST_CACHE_TTL", "2m")
	if got := campusCacheTTL("LEHU_TEST_CACHE_TTL", time.Second); got != 2*time.Minute {
		t.Fatalf("duration ttl = %s, want 2m", got)
	}

	t.Setenv("LEHU_TEST_CACHE_TTL", "15")
	if got := campusCacheTTL("LEHU_TEST_CACHE_TTL", time.Second); got != 15*time.Second {
		t.Fatalf("seconds ttl = %s, want 15s", got)
	}

	t.Setenv("LEHU_TEST_CACHE_TTL", "bad")
	if got := campusCacheTTL("LEHU_TEST_CACHE_TTL", time.Second); got != time.Second {
		t.Fatalf("fallback ttl = %s, want 1s", got)
	}
}

func TestShouldCachePostListOnlyAllowsPublicEarlyPages(t *testing.T) {
	repo := &campusRepo{}
	t.Setenv("LEHU_REDIS_CACHE_ENABLED", "true")

	cacheable := biz.ListCampusPostQuery{
		Statuses: []int32{biz.CampusAuditStatusVisible},
		Sort:     biz.CampusPostSortRecommend,
		Offset:   0,
		Limit:    20,
	}
	if repo.shouldCachePostList(cacheable) {
		t.Fatalf("query without redis client should not be cacheable")
	}

	repo = &campusRepo{data: &Data{}}
	if repo.shouldCachePostList(cacheable) {
		t.Fatalf("query without redis client should not be cacheable")
	}

	repo = &campusRepo{data: &Data{rds: redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})}}
	defer repo.data.rds.Close()
	if !repo.shouldCachePostList(cacheable) {
		t.Fatalf("public early page should be cacheable")
	}
	withKeyword := cacheable
	withKeyword.Keyword = "食堂"
	if repo.shouldCachePostList(withKeyword) {
		t.Fatalf("keyword query should not be cacheable")
	}
	latePage := cacheable
	latePage.Offset = 60
	if repo.shouldCachePostList(latePage) {
		t.Fatalf("page after third should not be cacheable")
	}

	repo = &campusRepo{data: &Data{rds: nil}}
	if repo.shouldCachePostList(cacheable) {
		t.Fatalf("query with nil redis client should not be cacheable")
	}

	repo = &campusRepo{data: &Data{rds: redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})}}
	defer repo.data.rds.Close()
	t.Setenv("LEHU_REDIS_CACHE_ENABLED", "false")
	if repo.shouldCachePostList(cacheable) {
		t.Fatalf("disabled cache should not be cacheable")
	}
}

func TestEnvBoolFalseData(t *testing.T) {
	for _, value := range []string{"0", "false", "no", "off", "disabled", " FALSE "} {
		if !envBoolFalseData(value) {
			t.Fatalf("envBoolFalseData(%q) = false, want true", value)
		}
	}
	for _, value := range []string{"", "1", "true", "yes"} {
		if envBoolFalseData(value) {
			t.Fatalf("envBoolFalseData(%q) = true, want false", value)
		}
	}
}
