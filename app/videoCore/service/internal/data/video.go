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

func (r *videoRepo) PublishVideo(ctx context.Context, in biz.Video) (int64, error) {
	// todo 改成雪花算法
	uid, err := uuid.NewUUID()
	if err != nil {
		return 0, err
	}
	video := model.Video{
		Id:          int64(uid.ID()),
		UserId:      in.Author.Id,
		Title:       in.Title,
		Description: in.Description,
		VideoUrl:    in.VideoUrl,
		CoverUrl:    in.CoverUrl,
	}

	err = r.data.db.Table(model.Video{}.TableName()).Create(&video).Error
	if err != nil {
		return 0, err
	}
	return video.Id, nil
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
		return false, nil, nil
	}
	if err != nil {
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
	user := model.User{}
	err := r.data.db.Table(model.User{}.TableName()).Where("id = ?", uid).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var videoList []*model.Video
	offset := (pageStats.Page - 1) * pageStats.PageSize
	err = r.data.db.Table(model.Video{}.TableName()).
		Where("user_id = ?", uid).
		Where("create_at <= ?", latestTime).
		Limit(int(pageStats.PageSize)).
		Offset(int(offset)).
		Order("id desc").
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
			Author:       &biz.Author{Id: user.Id, Name: user.Name, Avatar: user.Avatar},
			UploadTime:   video.CreatedAt,
		})
	}
	return videoBizList, nil
}
