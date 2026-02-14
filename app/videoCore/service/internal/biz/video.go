package biz

import (
	"context"
	"github.com/go-kratos/kratos/v2/log"
	"strconv"
	"time"
)

type Author struct {
	Id          int64
	Name        string
	Avatar      string
	IsFollowing int64
}

type Video struct {
	Id           int64
	Title        string
	Description  string
	VideoUrl     string
	CoverUrl     string
	LikeCount    int64
	CommentCount int64
	Author       *Author
	UploadTime   time.Time
}

// ✅ 使用Command/Query/Result模式
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
	LatestTime int64 // Unix时间戳
	PageStats  PageStats
}

type ListPublishedVideoResult struct {
	Videos []*Video
	Total  int64
}

// Feed流查询
type FeedShortVideoQuery struct {
	LatestTime int64 // Unix时间戳
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

// VideoStats 视频统计信息
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

	GetHotVideos(ctx context.Context, limit int) ([]*Video, error)
	GetVideosByAuthors(ctx context.Context, authorIDs []string, latestTime int64, limit int) ([]*Video, error)
	GetAuthorInfo(ctx context.Context, authorID string) (*Author, error)
	GetVideoListByTime(ctx context.Context, latestTime time.Time, limit int) ([]*Video, error)
	GetVideoStats(ctx context.Context, videoID string) (*VideoStats, error)

	// 新增：分页获取所有视频ID（用于布隆过滤器初始化）
	GetAllVideoIDs(ctx context.Context, offset int64, limit int) ([]string, error)
}

type VideoUsecase struct {
	repo         VideoRepo
	counterRepo  CounterRepo // 新增
	feedUsecase  *FeedUsecase
	recentViewed *RecentViewedManager    // 新增
	globalBloom  *GlobalVideoBloomFilter // 新增全局布隆过滤器
	log          *log.Helper
}

func NewVideoUsecase(
	repo VideoRepo,
	counterRepo CounterRepo,
	feedUsecase *FeedUsecase,
	recentViewed *RecentViewedManager,
	logger log.Logger,
) *VideoUsecase {
	// 创建全局布隆过滤器：预计1000万视频，误判率0.1%
	globalBloom := NewGlobalVideoBloomFilter(repo, logger, 10_000_000, 0.001)

	// 异步初始化布隆过滤器（避免阻塞服务启动）
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if err := globalBloom.Init(ctx); err != nil {
			log.NewHelper(logger).Warnf("全局视频布隆过滤器初始化失败: %v", err)
		}
	}()

	// 定时重建布隆过滤器（每天凌晨3点）
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			<-ticker.C
			// 可以选择在低峰期重建，这里简单固定延迟到凌晨3点执行
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
		repo:         repo,
		counterRepo:  counterRepo,
		feedUsecase:  feedUsecase,
		recentViewed: recentViewed,
		globalBloom:  globalBloom,
		log:          log.NewHelper(logger),
	}
}

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

	// 增加用户作品计数
	if _, err := uc.counterRepo.IncrUserCounter(ctx, cmd.UserId, "work_count", 1); err != nil {
		uc.log.Warnf("增加用户 work_count 失败: userId=%d, err=%v", cmd.UserId, err)
	}

	// 将新视频ID加入布隆过滤器
	uc.globalBloom.Add(strconv.FormatInt(videoId, 10))

	// 触发 Feed 事件（异步）
	if uc.feedUsecase != nil {
		go uc.feedUsecase.VideoPublishedHandler(
			context.Background(),
			strconv.FormatInt(videoId, 10),
			strconv.FormatInt(cmd.UserId, 10),
		)
	}

	return &PublishVideoResult{
		VideoId: videoId,
	}, nil
}

func (uc *VideoUsecase) GetVideoById(ctx context.Context, query *GetVideoByIdQuery) (*GetVideoByIdResult, error) {
	videoIDStr := strconv.FormatInt(query.VideoId, 10)

	// 布隆过滤器拦截：如果ID一定不存在，直接返回nil，避免查询数据库
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

	// 异步记录最近观看（仅当用户ID不为0）
	if query.UserId != 0 {
		go func() {
			bgCtx := context.Background()
			if err := uc.recentViewed.Add(bgCtx, strconv.FormatInt(query.UserId, 10), videoIDStr); err != nil {
				uc.log.Warnf("记录最近观看失败: %v", err)
			}
		}()
	}

	return &GetVideoByIdResult{
		Video: video,
	}, nil
}

func (uc *VideoUsecase) ListPublishedVideo(ctx context.Context, query *ListPublishedVideoQuery) (*ListPublishedVideoResult, error) {
	latestTime := time.Now()
	if query.LatestTime > 0 {
		latestTime = time.Unix(query.LatestTime, 0)
	}

	total, videos, err := uc.repo.GetVideoListByUid(ctx, query.UserId, latestTime, query.PageStats)
	if err != nil {
		return nil, err
	}

	return &ListPublishedVideoResult{
		Videos: videos,
		Total:  total,
	}, nil
}

func (uc *VideoUsecase) FeedShortVideo(ctx context.Context, query *FeedShortVideoQuery) (*FeedShortVideoResult, error) {
	latestTime := time.Now()
	if query.LatestTime > 0 {
		latestTime = time.Unix(query.LatestTime, 0)
	}

	videos, err := uc.repo.GetFeedVideos(ctx, latestTime, query.PageStats)
	if err != nil {
		return nil, err
	}

	return &FeedShortVideoResult{
		Videos: videos,
	}, nil
}

func (uc *VideoUsecase) GetVideoByIdList(ctx context.Context, query *GetVideoByIdListQuery) (*GetVideoByIdListResult, error) {
	videos, err := uc.repo.GetVideoByIdList(ctx, query.VideoIdList)
	if err != nil {
		return nil, err
	}

	return &GetVideoByIdListResult{
		Videos: videos,
	}, nil
}
