// data/video.go - 完整的VideoRepo和FeedRepo实现
package data

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"lehu-video/app/videoCore/service/internal/biz"
	"lehu-video/app/videoCore/service/internal/data/model"
)

type videoRepo struct {
	data *Data
	log  *log.Helper
}

func NewVideoRepo(data *Data, logger log.Logger) biz.VideoRepo {
	return &videoRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

// VideoRepo接口方法

func (r *videoRepo) PublishVideo(ctx context.Context, video *biz.Video) (int64, error) {
	// 使用uuid生成ID，后续可以改为雪花算法
	uid, err := uuid.NewUUID()
	if err != nil {
		return 0, err
	}

	dbVideo := model.Video{
		Id:          int64(uid.ID()),
		UserId:      video.Author.Id,
		Title:       video.Title,
		Description: video.Description,
		VideoUrl:    video.VideoUrl,
		CoverUrl:    video.CoverUrl,
		CreatedAt:   video.UploadTime,
	}

	err = r.data.db.Table(model.Video{}.TableName()).Create(&dbVideo).Error
	if err != nil {
		return 0, err
	}
	return dbVideo.Id, nil
}

func (r *videoRepo) GetVideoById(ctx context.Context, id int64) (bool, *biz.Video, error) {
	video := model.Video{}
	err := r.data.db.Table(model.Video{}.TableName()).Where("id = ?", id).First(&video).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, err
	}

	author := model.User{}
	err = r.data.db.Table(model.User{}.TableName()).Where("id = ?", video.UserId).First(&author).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 作者不存在，但视频存在，这种情况可能发生在用户被删除时
		// 返回一个空的作者信息
		author = model.User{
			Id:   video.UserId,
			Name: "用户已注销",
		}
	} else if err != nil {
		return false, nil, err
	}

	return true, &biz.Video{
		Id:           video.Id,
		Title:        video.Title,
		Description:  video.Description,
		VideoUrl:     video.VideoUrl,
		CoverUrl:     video.CoverUrl,
		LikeCount:    video.LikeCount,
		CommentCount: video.CommentCount,
		Author: &biz.Author{
			Id:     author.Id,
			Name:   author.Name,
			Avatar: author.Avatar,
		},
		UploadTime: video.CreatedAt,
	}, nil
}

func (r *videoRepo) GetVideoListByUid(ctx context.Context, uid int64, latestTime time.Time, pageStats biz.PageStats) (int64, []*biz.Video, error) {
	// 首先检查用户是否存在
	user := model.User{}
	err := r.data.db.Table(model.User{}.TableName()).Where("id = ?", uid).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 用户不存在，返回0和空列表
		return 0, []*biz.Video{}, nil
	}
	if err != nil {
		return 0, nil, err
	}

	// 1. 先查询总数
	var total int64
	query := r.data.db.Table(model.Video{}.TableName()).
		Where("user_id = ?", uid).
		Where("created_at <= ?", latestTime)

	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	// 如果总数为0，直接返回
	if total == 0 {
		return 0, []*biz.Video{}, nil
	}

	// 2. 查询分页数据
	var videoList []*model.Video
	offset := (pageStats.Page - 1) * pageStats.PageSize
	err = query.
		Limit(int(pageStats.PageSize)).
		Offset(int(offset)).
		Order("created_at desc").
		Find(&videoList).Error
	if err != nil {
		return 0, nil, err
	}

	var videoBizList []*biz.Video
	for _, video := range videoList {
		videoBizList = append(videoBizList, &biz.Video{
			Id:           video.Id,
			Title:        video.Title,
			Description:  video.Description,
			VideoUrl:     video.VideoUrl,
			CoverUrl:     video.CoverUrl,
			LikeCount:    video.LikeCount,
			CommentCount: video.CommentCount,
			Author: &biz.Author{
				Id:     user.Id,
				Name:   user.Name,
				Avatar: user.Avatar,
			},
			UploadTime: video.CreatedAt,
		})
	}

	return total, videoBizList, nil
}

func (r *videoRepo) GetVideoByIdList(ctx context.Context, idList []int64) ([]*biz.Video, error) {
	if len(idList) == 0 {
		return []*biz.Video{}, nil
	}

	// 批量查询视频
	var videoList []*model.Video
	err := r.data.db.Table(model.Video{}.TableName()).
		Where("id IN (?)", idList).
		Find(&videoList).Error
	if err != nil {
		return nil, err
	}

	// 收集用户ID
	userIds := make([]int64, 0)
	userVideoMap := make(map[int64][]*model.Video)
	for _, video := range videoList {
		userIds = append(userIds, video.UserId)
		userVideoMap[video.UserId] = append(userVideoMap[video.UserId], video)
	}

	// 批量查询用户信息
	var users []*model.User
	if len(userIds) > 0 {
		err = r.data.db.Table(model.User{}.TableName()).
			Where("id IN (?)", userIds).
			Find(&users).Error
		if err != nil {
			return nil, err
		}
	}

	// 创建用户ID到用户信息的映射
	userMap := make(map[int64]*model.User)
	for _, user := range users {
		userMap[user.Id] = user
	}

	// 组装结果
	var videoBizList []*biz.Video
	for userId, videos := range userVideoMap {
		user, exists := userMap[userId]
		var author *biz.Author
		if exists {
			author = &biz.Author{
				Id:     user.Id,
				Name:   user.Name,
				Avatar: user.Avatar,
			}
		} else {
			// 用户不存在，创建默认作者信息
			author = &biz.Author{
				Id:   userId,
				Name: "用户已注销",
			}
		}

		for _, video := range videos {
			videoBizList = append(videoBizList, &biz.Video{
				Id:           video.Id,
				Title:        video.Title,
				Description:  video.Description,
				VideoUrl:     video.VideoUrl,
				CoverUrl:     video.CoverUrl,
				LikeCount:    video.LikeCount,
				CommentCount: video.CommentCount,
				Author:       author,
				UploadTime:   video.CreatedAt,
			})
		}
	}

	return videoBizList, nil
}

func (r *videoRepo) GetFeedVideos(ctx context.Context, latestTime time.Time, pageStats biz.PageStats) ([]*biz.Video, error) {
	var videoList []*model.Video
	offset := (pageStats.Page - 1) * pageStats.PageSize

	// 按时间倒序获取视频，用于Feed流
	err := r.data.db.Table(model.Video{}.TableName()).
		Where("created_at <= ?", latestTime).
		Limit(int(pageStats.PageSize)).
		Offset(int(offset)).
		Order("created_at desc").
		Find(&videoList).Error
	if err != nil {
		return nil, err
	}

	// 收集用户ID
	userIds := make([]int64, 0)
	userVideoMap := make(map[int64][]*model.Video)
	for _, video := range videoList {
		userIds = append(userIds, video.UserId)
		userVideoMap[video.UserId] = append(userVideoMap[video.UserId], video)
	}

	// 批量查询用户信息
	var users []*model.User
	if len(userIds) > 0 {
		err = r.data.db.Table(model.User{}.TableName()).
			Where("id IN (?)", userIds).
			Find(&users).Error
		if err != nil {
			return nil, err
		}
	}

	// 创建用户ID到用户信息的映射
	userMap := make(map[int64]*model.User)
	for _, user := range users {
		userMap[user.Id] = user
	}

	// 组装结果，保持原有的顺序
	var videoBizList []*biz.Video
	for _, video := range videoList {
		user, exists := userMap[video.UserId]
		var author *biz.Author
		if exists {
			author = &biz.Author{
				Id:     user.Id,
				Name:   user.Name,
				Avatar: user.Avatar,
			}
		} else {
			// 用户不存在，创建默认作者信息
			author = &biz.Author{
				Id:   video.UserId,
				Name: "用户已注销",
			}
		}

		videoBizList = append(videoBizList, &biz.Video{
			Id:           video.Id,
			Title:        video.Title,
			Description:  video.Description,
			VideoUrl:     video.VideoUrl,
			CoverUrl:     video.CoverUrl,
			LikeCount:    video.LikeCount,
			CommentCount: video.CommentCount,
			Author:       author,
			UploadTime:   video.CreatedAt,
		})
	}

	return videoBizList, nil
}

// FeedRepo接口方法

// data/video.go - GetHotVideos 简化，不关联用户表
// data/video.go - GetHotVideos 必须填充 Author.Id
func (r *videoRepo) GetHotVideos(ctx context.Context, limit int) ([]*biz.Video, error) {
	var videos []*model.Video
	sevenDaysAgo := time.Now().AddDate(0, 0, -7)
	err := r.data.db.WithContext(ctx).
		Table(model.Video{}.TableName()).
		Where("created_at > ?", sevenDaysAgo).
		Order("(like_count * 2 + comment_count) DESC").
		Limit(limit).
		Find(&videos).Error
	if err != nil {
		return nil, err
	}

	bizVideos := make([]*biz.Video, 0, len(videos))
	for _, v := range videos {
		bizVideos = append(bizVideos, &biz.Video{
			Id:           v.Id,
			LikeCount:    v.LikeCount,
			CommentCount: v.CommentCount,
			UploadTime:   v.CreatedAt,
			Author: &biz.Author{ // ✅ 必须设置 Author.Id
				Id: v.UserId,
			},
		})
	}
	return bizVideos, nil
}

func (r *videoRepo) GetVideosByAuthors(ctx context.Context, authorIDs []string, latestTime int64, limit int) ([]*biz.Video, error) {
	if len(authorIDs) == 0 {
		return []*biz.Video{}, nil
	}

	var videos []*model.Video

	// 将string类型的authorIDs转换为int64
	var ids []int64
	for _, id := range authorIDs {
		if intID, err := strconv.ParseInt(id, 10, 64); err == nil {
			ids = append(ids, intID)
		}
	}

	if len(ids) == 0 {
		return []*biz.Video{}, nil
	}

	query := r.data.db.WithContext(ctx).
		Where("user_id IN (?)", ids).
		Order("created_at DESC").
		Limit(limit)

	if latestTime > 0 {
		query = query.Where("created_at < ?", time.Unix(latestTime, 0))
	}

	err := query.Find(&videos).Error
	if err != nil {
		return nil, err
	}
	res, err := r.convertDBVideosToBiz(videos)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (r *videoRepo) GetVideoListByTime(ctx context.Context, latestTime time.Time, limit int) ([]*biz.Video, error) {
	var videos []*model.Video

	err := r.data.db.WithContext(ctx).
		Where("created_at < ?", latestTime).
		Order("created_at DESC").
		Limit(limit).
		Find(&videos).Error

	if err != nil {
		return nil, err
	}

	res, err := r.convertDBVideosToBiz(videos)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (r *videoRepo) GetAuthorInfo(ctx context.Context, authorID string) (*biz.Author, error) {
	authorId, err := strconv.ParseInt(authorID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("作者ID格式错误: %v", err)
	}

	var user model.User
	err = r.data.db.WithContext(ctx).
		Table(model.User{}.TableName()).
		Where("id = ?", authorId).
		First(&user).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 用户不存在
		return &biz.Author{
			Id:   authorId,
			Name: "用户已注销",
		}, nil
	}

	if err != nil {
		return nil, err
	}

	return &biz.Author{
		Id:     user.Id,
		Name:   user.Name,
		Avatar: user.Avatar,
		// IsFollowing需要在业务层根据当前用户判断，这里不设置
	}, nil
}

func (r *videoRepo) GetVideoStats(ctx context.Context, videoID string) (*biz.VideoStats, error) {
	videoIDInt, err := strconv.ParseInt(videoID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("视频ID格式错误: %v", err)
	}

	var video model.Video
	err = r.data.db.WithContext(ctx).
		Table(model.Video{}.TableName()).
		Where("id = ?", videoIDInt).
		First(&video).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &biz.VideoStats{
			VideoID: videoID,
		}, nil
	}

	if err != nil {
		return nil, err
	}

	// 计算热度分数（简单示例）
	hotScore := float64(video.LikeCount)*0.4 + float64(video.CommentCount)*0.3

	return &biz.VideoStats{
		VideoID:      videoID,
		LikeCount:    video.LikeCount,
		CommentCount: video.CommentCount,
		ShareCount:   0, // 需要从其他表获取
		ViewCount:    0, // 需要从其他表获取
		HotScore:     hotScore,
	}, nil
}

// 辅助方法：转换数据库视频列表为业务层视频列表
func (r *videoRepo) convertDBVideosToBiz(dbVideos []*model.Video) ([]*biz.Video, error) {
	if len(dbVideos) == 0 {
		return []*biz.Video{}, nil
	}

	// 收集用户ID
	userIds := make([]int64, 0, len(dbVideos))
	for _, video := range dbVideos {
		userIds = append(userIds, video.UserId)
	}

	// 批量查询用户信息
	var users []*model.User
	err := r.data.db.Table(model.User{}.TableName()).
		Where("id IN (?)", userIds).
		Find(&users).Error
	if err != nil {
		return nil, err
	}

	// 创建用户映射
	userMap := make(map[int64]*model.User)
	for _, user := range users {
		userMap[user.Id] = user
	}

	// 转换视频
	bizVideos := make([]*biz.Video, 0, len(dbVideos))
	for _, dbVideo := range dbVideos {
		author := &biz.Author{
			Id:   dbVideo.UserId,
			Name: "用户已注销",
		}

		if user, exists := userMap[dbVideo.UserId]; exists {
			author.Name = user.Name
			author.Avatar = user.Avatar
		}

		bizVideos = append(bizVideos, &biz.Video{
			Id:           dbVideo.Id,
			Title:        dbVideo.Title,
			Description:  dbVideo.Description,
			VideoUrl:     dbVideo.VideoUrl,
			CoverUrl:     dbVideo.CoverUrl,
			LikeCount:    dbVideo.LikeCount,
			CommentCount: dbVideo.CommentCount,
			Author:       author,
			UploadTime:   dbVideo.CreatedAt,
		})
	}

	return bizVideos, nil
}
