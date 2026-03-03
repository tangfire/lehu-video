// biz/video.go - 加入播放量统计
package biz

import (
	"context"
	"strconv"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

type Author struct {
	Id          int64
	Name        string
	Avatar      string
	IsFollowing int64
}

type Video struct {
	Id              int64
	Title           string
	Description     string
	VideoUrl        string
	CoverUrl        string
	LikeCount       int64
	CommentCount    int64
	CollectionCount int64
	ViewCount       int64 // 新增播放量字段
	Author          *Author
	UploadTime      time.Time
}

type PublishVideoCommand struct {
	UserId      int64
	Title       string
	Description string
	PlayUrl     string
	CoverUrl    string
}

type PublishVideoResult struct {
	VideoId int64
}

type GetVideoByIdQuery struct {
	VideoId int64
	UserId  int64
}

type GetVideoByIdResult struct {
	Video *Video
}

type ListPublishedVideoQuery struct {
	UserId     int64
	LatestTime int64
	PageStats  PageStats
}

type ListPublishedVideoResult struct {
	Videos []*Video
	Total  int64
}

type FeedShortVideoQuery struct {
	LatestTime int64
	PageStats  PageStats
}

type FeedShortVideoResult struct {
	Videos []*Video
}

type GetVideoByIdListQuery struct {
	VideoIdList []int64
}

type GetVideoByIdListResult struct {
	Videos []*Video
}

type VideoStats struct {
	VideoID      string
	LikeCount    int64
	CommentCount int64
	ShareCount   int64
	ViewCount    int64
	HotScore     float64
}

type VideoRepo interface {
	PublishVideo(ctx context.Context, video *Video) (int64, error)
	GetVideoById(ctx context.Context, id int64) (bool, *Video, error)
	GetVideoListByUid(ctx context.Context, uid int64, latestTime time.Time, pageStats PageStats) (int64, []*Video, error)
	GetVideoByIdList(ctx context.Context, idList []int64) ([]*Video, error)
	GetFeedVideos(ctx context.Context, latestTime time.Time, pageStats PageStats) ([]*Video, error)
	GetHotVideos(ctx context.Context, limit int) ([]*Video, error) // 需要返回 view_count
	GetVideosByAuthors(ctx context.Context, authorIDs []string, latestTime int64, limit int) ([]*Video, error)
	GetVideosByAuthorsExclude(ctx context.Context, authorIDs []string, latestTime int64, limit int, excludeIDs []string) ([]*Video, error)
	GetAuthorInfo(ctx context.Context, authorID string) (*Author, error)
	GetVideoListByTime(ctx context.Context, latestTime time.Time, limit int) ([]*Video, error)
	GetVideoStats(ctx context.Context, videoID string) (*VideoStats, error)
	GetAllVideoIDs(ctx context.Context, offset int64, limit int) ([]string, error)
}

type VideoUsecase struct {
	repo            VideoRepo
	userCounterRepo UserCounterRepo
	videoCounter    VideoCounterRepo
	feedUsecase     *FeedUsecase
	recentViewed    *RecentViewedManager
	globalBloom     *GlobalVideoBloomFilter
	log             *log.Helper
}

func NewVideoUsecase(
	repo VideoRepo,
	userCounterRepo UserCounterRepo,
	videoCounter VideoCounterRepo,
	feedUsecase *FeedUsecase,
	recentViewed *RecentViewedManager,
	logger log.Logger,
) *VideoUsecase {
	globalBloom := NewGlobalVideoBloomFilter(repo, logger, 10_000_000, 0.001)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if err := globalBloom.Init(ctx); err != nil {
			log.NewHelper(logger).Warnf("全局视频布隆过滤器初始化失败: %v", err)
		}
	}()
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			<-ticker.C
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day(), 3, 0, 0, 0, now.Location())
			if now.After(next) {
				next = next.Add(24 * time.Hour)
			}
			time.Sleep(time.Until(next))

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
			if err := globalBloom.Rebuild(ctx); err != nil {
				log.NewHelper(logger).Errorf("重建全局布隆过滤器失败: %v", err)
			}
			cancel()
		}
	}()

	return &VideoUsecase{
		repo:            repo,
		userCounterRepo: userCounterRepo,
		videoCounter:    videoCounter,
		feedUsecase:     feedUsecase,
		recentViewed:    recentViewed,
		globalBloom:     globalBloom,
		log:             log.NewHelper(logger),
	}
}

// PublishVideo 发布视频
func (uc *VideoUsecase) PublishVideo(ctx context.Context, cmd *PublishVideoCommand) (*PublishVideoResult, error) {
	video := &Video{
		Title:       cmd.Title,
		Description: cmd.Description,
		VideoUrl:    cmd.PlayUrl,
		CoverUrl:    cmd.CoverUrl,
		Author:      &Author{Id: cmd.UserId},
		UploadTime:  time.Now(),
	}
	videoId, err := uc.repo.PublishVideo(ctx, video)
	if err != nil {
		return nil, err
	}
	if _, err := uc.userCounterRepo.IncrUserCounter(ctx, cmd.UserId, "work_count", 1); err != nil {
		uc.log.Warnf("增加用户 work_count 失败: userId=%d, err=%v", cmd.UserId, err)
	}
	uc.globalBloom.Add(strconv.FormatInt(videoId, 10))
	if uc.feedUsecase != nil {
		go uc.feedUsecase.VideoPublishedHandler(
			context.Background(),
			strconv.FormatInt(videoId, 10),
			strconv.FormatInt(cmd.UserId, 10),
		)
	}
	return &PublishVideoResult{VideoId: videoId}, nil
}

// GetVideoById 获取视频详情（填充计数器），同时增加播放量计数
func (uc *VideoUsecase) GetVideoById(ctx context.Context, query *GetVideoByIdQuery) (*GetVideoByIdResult, error) {
	videoIDStr := strconv.FormatInt(query.VideoId, 10)
	if !uc.globalBloom.Exists(videoIDStr) {
		uc.log.Debugf("布隆过滤器拦截不存在视频ID: %d", query.VideoId)
		return nil, nil
	}
	exist, video, err := uc.repo.GetVideoById(ctx, query.VideoId)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, nil
	}
	// 填充计数（从 Redis 获取）
	if err := uc.fillVideoCounters(ctx, video); err != nil {
		uc.log.Warnf("填充视频计数失败: %v", err)
	}
	// 增加播放量计数
	if uc.videoCounter != nil {
		go func() {
			// 异步增加播放量，不影响主流程
			if err := uc.videoCounter.IncrVideoCounter(context.Background(), query.VideoId, "view_count", 1); err != nil {
				uc.log.Warnf("增加视频播放量失败: videoId=%d, err=%v", query.VideoId, err)
			}
		}()
	}
	if query.UserId != 0 {
		go func() {
			bgCtx := context.Background()
			if err := uc.recentViewed.Add(bgCtx, strconv.FormatInt(query.UserId, 10), videoIDStr); err != nil {
				uc.log.Warnf("记录最近观看失败: %v", err)
			}
		}()
	}
	return &GetVideoByIdResult{Video: video}, nil
}

// ListPublishedVideo 列出用户发布的视频
func (uc *VideoUsecase) ListPublishedVideo(ctx context.Context, query *ListPublishedVideoQuery) (*ListPublishedVideoResult, error) {
	latestTime := time.Now()
	if query.LatestTime > 0 {
		latestTime = time.Unix(query.LatestTime, 0)
	}
	total, videos, err := uc.repo.GetVideoListByUid(ctx, query.UserId, latestTime, query.PageStats)
	if err != nil {
		return nil, err
	}
	if err := uc.batchFillVideoCounters(ctx, videos); err != nil {
		uc.log.Warnf("批量填充视频计数失败: %v", err)
	}
	return &ListPublishedVideoResult{
		Videos: videos,
		Total:  total,
	}, nil
}

// FeedShortVideo 视频流
func (uc *VideoUsecase) FeedShortVideo(ctx context.Context, query *FeedShortVideoQuery) (*FeedShortVideoResult, error) {
	latestTime := time.Now()
	if query.LatestTime > 0 {
		latestTime = time.Unix(query.LatestTime, 0)
	}
	videos, err := uc.repo.GetFeedVideos(ctx, latestTime, query.PageStats)
	if err != nil {
		return nil, err
	}
	if err := uc.batchFillVideoCounters(ctx, videos); err != nil {
		uc.log.Warnf("批量填充视频计数失败: %v", err)
	}
	return &FeedShortVideoResult{Videos: videos}, nil
}

// GetVideoByIdList 根据ID列表获取视频
func (uc *VideoUsecase) GetVideoByIdList(ctx context.Context, query *GetVideoByIdListQuery) (*GetVideoByIdListResult, error) {
	videos, err := uc.repo.GetVideoByIdList(ctx, query.VideoIdList)
	if err != nil {
		return nil, err
	}
	if err := uc.batchFillVideoCounters(ctx, videos); err != nil {
		uc.log.Warnf("批量填充视频计数失败: %v", err)
	}
	return &GetVideoByIdListResult{Videos: videos}, nil
}

// fillVideoCounters 填充单个视频的计数（从Redis获取，若无则用MySQL值并同步回填）
func (uc *VideoUsecase) fillVideoCounters(ctx context.Context, video *Video) error {
	if video == nil {
		return nil
	}
	counters, err := uc.videoCounter.GetVideoCounters(ctx, video.Id, "like_count", "comment_count", "collection_count", "view_count")
	if err != nil {
		return err
	}
	needSet := false
	setCounts := make(map[string]int64)
	if val, ok := counters["like_count"]; ok {
		video.LikeCount = val
	} else {
		setCounts["like_count"] = video.LikeCount
		needSet = true
	}
	if val, ok := counters["comment_count"]; ok {
		video.CommentCount = val
	} else {
		setCounts["comment_count"] = video.CommentCount
		needSet = true
	}
	if val, ok := counters["collection_count"]; ok {
		video.CollectionCount = val
	} else {
		setCounts["collection_count"] = video.CollectionCount
		needSet = true
	}
	if val, ok := counters["view_count"]; ok {
		video.ViewCount = val
	} else {
		setCounts["view_count"] = video.ViewCount
		needSet = true
	}
	if needSet {
		setCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		if err := uc.videoCounter.SetVideoCounters(setCtx, video.Id, setCounts); err != nil {
			uc.log.Warnf("回填视频计数器失败: videoID=%d, err=%v", video.Id, err)
		}
	}
	return nil
}

// batchFillVideoCounters 批量填充视频计数
func (uc *VideoUsecase) batchFillVideoCounters(ctx context.Context, videos []*Video) error {
	if len(videos) == 0 {
		return nil
	}
	videoIds := make([]int64, len(videos))
	for i, v := range videos {
		videoIds[i] = v.Id
	}
	countersMap, err := uc.videoCounter.BatchGetVideoCounters(ctx, videoIds, "like_count", "comment_count", "collection_count", "view_count")
	if err != nil {
		return err
	}
	needSet := make(map[int64]map[string]int64)
	for _, v := range videos {
		if counters, ok := countersMap[v.Id]; ok {
			if val, ok := counters["like_count"]; ok {
				v.LikeCount = val
			} else {
				if needSet[v.Id] == nil {
					needSet[v.Id] = make(map[string]int64)
				}
				needSet[v.Id]["like_count"] = v.LikeCount
			}
			if val, ok := counters["comment_count"]; ok {
				v.CommentCount = val
			} else {
				if needSet[v.Id] == nil {
					needSet[v.Id] = make(map[string]int64)
				}
				needSet[v.Id]["comment_count"] = v.CommentCount
			}
			if val, ok := counters["collection_count"]; ok {
				v.CollectionCount = val
			} else {
				if needSet[v.Id] == nil {
					needSet[v.Id] = make(map[string]int64)
				}
				needSet[v.Id]["collection_count"] = v.CollectionCount
			}
			if val, ok := counters["view_count"]; ok {
				v.ViewCount = val
			} else {
				if needSet[v.Id] == nil {
					needSet[v.Id] = make(map[string]int64)
				}
				needSet[v.Id]["view_count"] = v.ViewCount
			}
		} else {
			needSet[v.Id] = map[string]int64{
				"like_count":       v.LikeCount,
				"comment_count":    v.CommentCount,
				"collection_count": v.CollectionCount,
				"view_count":       v.ViewCount,
			}
		}
	}
	if len(needSet) > 0 {
		go func() {
			bgCtx := context.Background()
			for vid, counts := range needSet {
				if err := uc.videoCounter.SetVideoCounters(bgCtx, vid, counts); err != nil {
					uc.log.Warnf("异步回填视频计数器失败: videoID=%d, err=%v", vid, err)
				}
			}
		}()
	}
	return nil
}
