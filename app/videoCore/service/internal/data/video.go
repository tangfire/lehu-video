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

func (r *videoRepo) db(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value("db").(*gorm.DB); ok {
		return tx
	}
	return r.data.db.WithContext(ctx)
}

func (r *videoRepo) PublishVideo(ctx context.Context, video *biz.Video) (int64, error) {
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
	err = r.db(ctx).Table(model.Video{}.TableName()).Create(&dbVideo).Error
	if err != nil {
		return 0, err
	}
	return dbVideo.Id, nil
}

func (r *videoRepo) GetVideoById(ctx context.Context, id int64) (bool, *biz.Video, error) {
	var video model.Video
	err := r.db(ctx).Table(model.Video{}.TableName()).Where("id = ?", id).First(&video).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, err
	}

	var user model.User
	err = r.db(ctx).Table(model.User{}.TableName()).Where("id = ?", video.UserId).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		user = model.User{
			Id:   video.UserId,
			Name: "用户已注销",
		}
	} else if err != nil {
		return false, nil, err
	}

	return true, &biz.Video{
		Id:              video.Id,
		Title:           video.Title,
		Description:     video.Description,
		VideoUrl:        video.VideoUrl,
		CoverUrl:        video.CoverUrl,
		LikeCount:       video.LikeCount,
		CommentCount:    video.CommentCount,
		CollectionCount: video.CollectionCount,
		ViewCount:       video.ViewCount,
		Author: &biz.Author{
			Id:     user.Id,
			Name:   user.Name,
			Avatar: user.Avatar,
		},
		UploadTime: video.CreatedAt,
	}, nil
}

func (r *videoRepo) GetVideoListByUid(ctx context.Context, uid int64, latestTime time.Time, pageStats biz.PageStats) (int64, []*biz.Video, error) {
	var user model.User
	err := r.db(ctx).Table(model.User{}.TableName()).Where("id = ?", uid).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, []*biz.Video{}, nil
	}
	if err != nil {
		return 0, nil, err
	}

	var total int64
	query := r.db(ctx).Table(model.Video{}.TableName()).
		Where("user_id = ?", uid).
		Where("created_at <= ?", latestTime)

	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	if total == 0 {
		return 0, []*biz.Video{}, nil
	}

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

	bizVideos, err := r.convertDBVideosToBiz(ctx, videoList)
	if err != nil {
		return 0, nil, err
	}
	return total, bizVideos, nil
}

func (r *videoRepo) GetVideoByIdList(ctx context.Context, idList []int64) ([]*biz.Video, error) {
	if len(idList) == 0 {
		return []*biz.Video{}, nil
	}

	var videoList []*model.Video
	err := r.db(ctx).Table(model.Video{}.TableName()).
		Where("id IN (?)", idList).
		Find(&videoList).Error
	if err != nil {
		return nil, err
	}
	return r.convertDBVideosToBiz(ctx, videoList)
}

func (r *videoRepo) GetFeedVideos(ctx context.Context, latestTime time.Time, pageStats biz.PageStats) ([]*biz.Video, error) {
	var videoList []*model.Video
	offset := (pageStats.Page - 1) * pageStats.PageSize

	err := r.db(ctx).Table(model.Video{}.TableName()).
		Where("created_at <= ?", latestTime).
		Limit(int(pageStats.PageSize)).
		Offset(int(offset)).
		Order("created_at desc").
		Find(&videoList).Error
	if err != nil {
		return nil, err
	}
	return r.convertDBVideosToBiz(ctx, videoList)
}

// GetHotVideos 获取近7天视频，用于热门池计算（返回ID、作者ID、上传时间、互动数、播放量）
func (r *videoRepo) GetHotVideos(ctx context.Context, limit int) ([]*biz.Video, error) {
	var videos []*model.Video
	sevenDaysAgo := time.Now().AddDate(0, 0, -7)
	err := r.db(ctx).
		Table(model.Video{}.TableName()).
		Select("id", "user_id", "created_at", "like_count", "comment_count", "collection_count", "view_count").
		Where("created_at > ?", sevenDaysAgo).
		Limit(limit).
		Find(&videos).Error
	if err != nil {
		return nil, err
	}
	bizVideos := make([]*biz.Video, 0, len(videos))
	for _, v := range videos {
		bizVideos = append(bizVideos, &biz.Video{
			Id:              v.Id,
			LikeCount:       v.LikeCount,
			CommentCount:    v.CommentCount,
			CollectionCount: v.CollectionCount,
			ViewCount:       v.ViewCount,
			UploadTime:      v.CreatedAt,
			Author:          &biz.Author{Id: v.UserId},
		})
	}
	return bizVideos, nil
}

func (r *videoRepo) GetVideosByAuthors(ctx context.Context, authorIDs []string, latestTime int64, limit int) ([]*biz.Video, error) {
	return r.GetVideosByAuthorsExclude(ctx, authorIDs, latestTime, limit, nil)
}

// GetVideosByAuthorsExclude 根据作者ID列表获取视频，并排除指定ID
func (r *videoRepo) GetVideosByAuthorsExclude(ctx context.Context, authorIDs []string, latestTime int64, limit int, excludeIDs []string) ([]*biz.Video, error) {
	if len(authorIDs) == 0 {
		return []*biz.Video{}, nil
	}
	var ids []int64
	for _, id := range authorIDs {
		if intID, err := strconv.ParseInt(id, 10, 64); err == nil {
			ids = append(ids, intID)
		}
	}
	if len(ids) == 0 {
		return []*biz.Video{}, nil
	}
	query := r.db(ctx).
		Table(model.Video{}.TableName()).
		Where("user_id IN (?)", ids).
		Order("created_at DESC").
		Limit(limit)
	if latestTime > 0 {
		query = query.Where("created_at < ?", time.Unix(latestTime, 0))
	}
	if len(excludeIDs) > 0 {
		var excludeInts []int64
		for _, id := range excludeIDs {
			if intID, err := strconv.ParseInt(id, 10, 64); err == nil {
				excludeInts = append(excludeInts, intID)
			}
		}
		if len(excludeInts) > 0 {
			query = query.Where("id NOT IN (?)", excludeInts)
		}
	}
	var videos []*model.Video
	err := query.Find(&videos).Error
	if err != nil {
		return nil, err
	}
	return r.convertDBVideosToBiz(ctx, videos)
}

func (r *videoRepo) GetVideoListByTime(ctx context.Context, latestTime time.Time, limit int) ([]*biz.Video, error) {
	var videos []*model.Video
	err := r.db(ctx).
		Table(model.Video{}.TableName()).
		Where("created_at < ?", latestTime).
		Order("created_at DESC").
		Limit(limit).
		Find(&videos).Error
	if err != nil {
		return nil, err
	}
	return r.convertDBVideosToBiz(ctx, videos)
}

func (r *videoRepo) GetAuthorInfo(ctx context.Context, authorID string) (*biz.Author, error) {
	authorId, err := strconv.ParseInt(authorID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("作者ID格式错误: %v", err)
	}
	var user model.User
	err = r.db(ctx).
		Table(model.User{}.TableName()).
		Where("id = ?", authorId).
		First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
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
	}, nil
}

func (r *videoRepo) GetVideoStats(ctx context.Context, videoID string) (*biz.VideoStats, error) {
	videoIDInt, err := strconv.ParseInt(videoID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("视频ID格式错误: %v", err)
	}
	var video model.Video
	err = r.db(ctx).
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
	hotScore := float64(video.LikeCount)*0.4 + float64(video.CommentCount)*0.3 + float64(video.ViewCount)*0.1
	return &biz.VideoStats{
		VideoID:      videoID,
		LikeCount:    video.LikeCount,
		CommentCount: video.CommentCount,
		ShareCount:   0,
		ViewCount:    video.ViewCount,
		HotScore:     hotScore,
	}, nil
}

func (r *videoRepo) GetAllVideoIDs(ctx context.Context, offset int64, limit int) ([]string, error) {
	var ids []int64
	err := r.db(ctx).
		Table(model.Video{}.TableName()).
		Select("id").
		Order("id ASC").
		Offset(int(offset)).
		Limit(limit).
		Pluck("id", &ids).Error
	if err != nil {
		return nil, err
	}
	strIDs := make([]string, len(ids))
	for i, id := range ids {
		strIDs[i] = strconv.FormatInt(id, 10)
	}
	return strIDs, nil
}

// convertDBVideosToBiz 通用的数据库视频列表转业务视频列表（包括作者信息）
func (r *videoRepo) convertDBVideosToBiz(ctx context.Context, dbVideos []*model.Video) ([]*biz.Video, error) {
	if len(dbVideos) == 0 {
		return []*biz.Video{}, nil
	}
	userIds := make([]int64, 0, len(dbVideos))
	for _, video := range dbVideos {
		userIds = append(userIds, video.UserId)
	}
	var users []*model.User
	err := r.db(ctx).Table(model.User{}.TableName()).
		Where("id IN (?)", userIds).
		Find(&users).Error
	if err != nil {
		return nil, err
	}
	userMap := make(map[int64]*model.User)
	for _, user := range users {
		userMap[user.Id] = user
	}
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
			Id:              dbVideo.Id,
			Title:           dbVideo.Title,
			Description:     dbVideo.Description,
			VideoUrl:        dbVideo.VideoUrl,
			CoverUrl:        dbVideo.CoverUrl,
			LikeCount:       dbVideo.LikeCount,
			CommentCount:    dbVideo.CommentCount,
			CollectionCount: dbVideo.CollectionCount,
			ViewCount:       dbVideo.ViewCount,
			Author:          author,
			UploadTime:      dbVideo.CreatedAt,
		})
	}
	return bizVideos, nil
}

// BatchGetVideoAuthors 批量获取视频的作者ID
func (r *videoRepo) BatchGetVideoAuthors(ctx context.Context, videoIDs []int64) (map[int64]int64, error) {
	if len(videoIDs) == 0 {
		return make(map[int64]int64), nil
	}
	type result struct {
		ID     int64
		UserID int64
	}
	var rows []result
	err := r.db(ctx).Table("video").
		Select("id, user_id").
		Where("id IN (?)", videoIDs).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	authorMap := make(map[int64]int64, len(rows))
	for _, row := range rows {
		authorMap[row.ID] = row.UserID
	}
	return authorMap, nil
}
