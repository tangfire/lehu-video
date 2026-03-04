// biz/video.go - 使用 BatchProcessor 聚合播放量
package biz

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
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
	ViewCount       int64
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
	GetHotVideos(ctx context.Context, limit int) ([]*Video, error)
	GetVideosByAuthors(ctx context.Context, authorIDs []string, latestTime int64, limit int) ([]*Video, error)
	GetVideosByAuthorsExclude(ctx context.Context, authorIDs []string, latestTime int64, limit int, excludeIDs []string) ([]*Video, error)
	GetAuthorInfo(ctx context.Context, authorID string) (*Author, error)
	GetVideoListByTime(ctx context.Context, latestTime time.Time, limit int) ([]*Video, error)
	GetVideoStats(ctx context.Context, videoID string) (*VideoStats, error)
	GetAllVideoIDs(ctx context.Context, offset int64, limit int) ([]string, error)
}

// viewCountCmd 播放量增加命令
type viewCountCmd struct {
	videoID int64
	delta   int64
}

type VideoUsecase struct {
	repo            VideoRepo
	userCounterRepo UserCounterRepo
	videoCounter    VideoCounterRepo
	feedUsecase     *FeedUsecase
	recentViewed    *RecentViewedManager
	redisClient     *redis.Client
	sfg             singleflight.Group
	log             *log.Helper
	viewBatchProc   *BatchProcessor[*viewCountCmd] // 批量处理器
}

func NewVideoUsecase(
	repo VideoRepo,
	userCounterRepo UserCounterRepo,
	videoCounter VideoCounterRepo,
	feedUsecase *FeedUsecase,
	recentViewed *RecentViewedManager,
	redisClient *redis.Client,
	logger log.Logger,
) *VideoUsecase {
	uc := &VideoUsecase{
		repo:            repo,
		userCounterRepo: userCounterRepo,
		videoCounter:    videoCounter,
		feedUsecase:     feedUsecase,
		recentViewed:    recentViewed,
		redisClient:     redisClient,
		log:             log.NewHelper(logger),
	}
	// 初始化批量处理器：积攒 500 条或每 2 秒刷新一次
	uc.viewBatchProc = NewBatchProcessor[*viewCountCmd](
		500,
		2*time.Second,
		uc.batchProcessViewCount,
		logger,
	)
	return uc
}

func (uc *VideoUsecase) Stop() {
	if uc.viewBatchProc != nil {
		uc.viewBatchProc.Stop()
	}
}

// batchProcessViewCount 批量处理播放量
func (uc *VideoUsecase) batchProcessViewCount(cmds []*viewCountCmd) error {
	if len(cmds) == 0 {
		return nil
	}
	agg := make(map[int64]int64)
	for _, cmd := range cmds {
		agg[cmd.videoID] += cmd.delta
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := uc.videoCounter.BatchIncrVideoCounters(ctx, agg); err != nil {
		uc.log.Warnf("批量增加播放量失败: %v", err)
		return err
	}
	return nil
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
	if uc.feedUsecase != nil {
		go uc.feedUsecase.VideoPublishedHandler(
			context.Background(),
			strconv.FormatInt(videoId, 10),
			strconv.FormatInt(cmd.UserId, 10),
		)
	}
	return &PublishVideoResult{VideoId: videoId}, nil
}

// GetVideoById 获取视频详情（SingleFlight + 缓存空值）
func (uc *VideoUsecase) GetVideoById(ctx context.Context, query *GetVideoByIdQuery) (*GetVideoByIdResult, error) {
	videoIDStr := strconv.FormatInt(query.VideoId, 10)
	cacheKey := "video:" + videoIDStr
	emptyMark := "EMPTY"

	// 1. 尝试从 Redis 缓存读取
	cached, err := uc.redisClient.Get(ctx, cacheKey).Result()
	if err == nil {
		if cached == emptyMark {
			uc.log.Debugf("缓存空值命中，视频ID不存在: %s", videoIDStr)
			return nil, nil
		}
		var video Video
		if err := json.Unmarshal([]byte(cached), &video); err == nil {
			// 填充计数（从 Redis 获取最新的计数）
			if err := uc.fillVideoCounters(ctx, &video); err != nil {
				uc.log.Warnf("填充视频计数失败: %v", err)
			}
			// 播放量通过批量处理器增加
			uc.viewBatchProc.Add(&viewCountCmd{videoID: query.VideoId, delta: 1})
			uc.recordRecentViewAsync(query.UserId, videoIDStr)
			return &GetVideoByIdResult{Video: &video}, nil
		}
		uc.log.Warnf("缓存反序列化失败，key=%s, err=%v", cacheKey, err)
	} else if !errors.Is(err, redis.Nil) {
		uc.log.Warnf("读取缓存失败: %v", err)
	}

	// 2. 使用 SingleFlight 保护数据库查询
	val, err, _ := uc.sfg.Do(videoIDStr, func() (interface{}, error) {
		// 双重检查：再次查询缓存
		cached, err := uc.redisClient.Get(ctx, cacheKey).Result()
		if err == nil {
			if cached == emptyMark {
				return nil, nil
			}
			var video Video
			if err := json.Unmarshal([]byte(cached), &video); err == nil {
				return &video, nil
			}
		}

		// 查询数据库
		exist, video, err := uc.repo.GetVideoById(ctx, query.VideoId)
		if err != nil {
			return nil, err
		}
		if !exist {
			// 缓存空值
			if err := uc.redisClient.Set(ctx, cacheKey, emptyMark, 5*time.Minute).Err(); err != nil {
				uc.log.Warnf("缓存空值失败: %v", err)
			}
			return nil, nil
		}

		// 视频存在，序列化后写入缓存
		data, err := json.Marshal(video)
		if err != nil {
			uc.log.Warnf("序列化视频失败: %v", err)
		} else {
			if err := uc.redisClient.Set(ctx, cacheKey, data, 24*time.Hour).Err(); err != nil {
				uc.log.Warnf("缓存视频失败: %v", err)
			}
		}
		return video, nil
	})

	if err != nil {
		return nil, err
	}
	video, _ := val.(*Video)

	// 3. 填充计数
	if video != nil {
		if err := uc.fillVideoCounters(ctx, video); err != nil {
			uc.log.Warnf("填充视频计数失败: %v", err)
		}
		uc.viewBatchProc.Add(&viewCountCmd{videoID: query.VideoId, delta: 1})
		uc.recordRecentViewAsync(query.UserId, videoIDStr)
	}

	return &GetVideoByIdResult{Video: video}, nil
}

// recordRecentViewAsync 异步记录最近观看
func (uc *VideoUsecase) recordRecentViewAsync(userId int64, videoIDStr string) {
	if userId == 0 || uc.recentViewed == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := uc.recentViewed.Add(ctx, strconv.FormatInt(userId, 10), videoIDStr); err != nil {
			uc.log.Warnf("记录最近观看失败: %v", err)
		}
	}()
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

// fillVideoCounters 填充单个视频的计数（从Redis获取）
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
		// 异步回填 Redis 计数（避免阻塞）
		go func() {
			bgCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			if err := uc.videoCounter.SetVideoCounters(bgCtx, video.Id, setCounts); err != nil {
				uc.log.Warnf("回填视频计数器失败: videoID=%d, err=%v", video.Id, err)
			}
		}()
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
