package biz

import (
	"context"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"lehu-video/pkg/apperror"
)

type CampusRecommendPool struct {
	mu        sync.RWMutex
	recommend []int64
	hot       []int64
	updatedAt time.Time
	log       *log.Helper
}

func NewCampusRecommendPool(logger log.Logger) *CampusRecommendPool {
	return &CampusRecommendPool{log: log.NewHelper(logger)}
}

func (p *CampusRecommendPool) Set(recommend, hot []int64) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.recommend = append([]int64(nil), recommend...)
	p.hot = append([]int64(nil), hot...)
	p.updatedAt = time.Now()
}

func (p *CampusRecommendPool) Get(sort string, offset, limit int) ([]int64, bool) {
	if p == nil || offset < 0 || limit <= 0 {
		return nil, false
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	var ids []int64
	switch sort {
	case CampusPostSortHot:
		ids = p.hot
	case CampusPostSortRecommend:
		ids = p.recommend
	default:
		return nil, false
	}
	if len(ids) == 0 || offset >= len(ids) {
		return nil, false
	}
	end := offset + limit
	if end > len(ids) {
		end = len(ids)
	}
	return append([]int64(nil), ids[offset:end]...), true
}

func (p *CampusRecommendPool) Total(sort string) int64 {
	if p == nil {
		return 0
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	switch sort {
	case CampusPostSortHot:
		return int64(len(p.hot))
	case CampusPostSortRecommend:
		return int64(len(p.recommend))
	default:
		return 0
	}
}

func (uc *CampusUsecase) RefreshCampusRecommendPool(ctx context.Context) error {
	if uc.recommendPool == nil {
		return nil
	}
	recommend, _, err := uc.repo.ListPosts(ctx, ListCampusPostQuery{
		Sort:     CampusPostSortRecommend,
		Statuses: []int32{CampusAuditStatusVisible},
		Offset:   0,
		Limit:    200,
	})
	if err != nil {
		return apperror.Internal(err, "刷新推荐池失败")
	}
	hot, _, err := uc.repo.ListPosts(ctx, ListCampusPostQuery{
		Sort:     CampusPostSortHot,
		Statuses: []int32{CampusAuditStatusVisible},
		Offset:   0,
		Limit:    200,
	})
	if err != nil {
		return apperror.Internal(err, "刷新热门池失败")
	}
	uc.recommendPool.Set(postIDs(recommend), postIDs(hot))
	return nil
}

func (uc *CampusUsecase) listPostsFromPool(ctx context.Context, query ListCampusPostQuery) ([]*CampusForumPost, int64, bool, error) {
	if uc.recommendPool == nil || !query.EligibleForRecommendPool() {
		return nil, 0, false, nil
	}
	ids, ok := uc.recommendPool.Get(query.Sort, query.Offset, query.Limit)
	if !ok || len(ids) == 0 {
		return nil, 0, false, nil
	}
	posts, err := uc.repo.ListPostsByIDs(ctx, ids, query.Statuses)
	if err != nil {
		return nil, 0, true, err
	}
	return posts, uc.recommendPool.Total(query.Sort), true, nil
}

func (q ListCampusPostQuery) EligibleForRecommendPool() bool {
	if q.Sort != CampusPostSortRecommend && q.Sort != CampusPostSortHot {
		return false
	}
	return q.CategoryCode == "" &&
		q.PostType == "" &&
		q.Keyword == "" &&
		q.AuthorID == "" &&
		q.CollectedByUserID == "" &&
		!q.IncludeDeleted &&
		q.OnlyOfficial == nil &&
		q.OnlyFeatured == nil &&
		q.OnlyPinned == nil &&
		!q.OnlyReported
}

func postIDs(posts []*CampusForumPost) []int64 {
	ids := make([]int64, 0, len(posts))
	for _, post := range posts {
		if post != nil && post.ID > 0 {
			ids = append(ids, post.ID)
		}
	}
	return ids
}
