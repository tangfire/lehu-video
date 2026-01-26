package data

import (
	"context"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"lehu-video/app/videoCore/service/internal/biz"
	"lehu-video/app/videoCore/service/internal/data/model"
	"time"
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

func (r *videoRepo) GetVideoListByUid(ctx context.Context, uid int64, latestTime time.Time, pageStats biz.PageStats) ([]*biz.Video, error) {
	// 首先检查用户是否存在
	user := model.User{}
	err := r.data.db.Table(model.User{}.TableName()).Where("id = ?", uid).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 用户不存在，返回空列表
		return []*biz.Video{}, nil
	}
	if err != nil {
		return nil, err
	}

	var videoList []*model.Video
	offset := (pageStats.Page - 1) * pageStats.PageSize
	err = r.data.db.Table(model.Video{}.TableName()).
		Where("user_id = ?", uid).
		Where("created_at <= ?", latestTime).
		Limit(int(pageStats.PageSize)).
		Offset(int(offset)).
		Order("created_at desc").
		Find(&videoList).Error
	if err != nil {
		return nil, err
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
	return videoBizList, nil
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
