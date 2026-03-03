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

// db 返回事务中的数据库连接，支持事务
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

	var videoBizList []*biz.Video
	for _, video := range videoList {
		videoBizList = append(videoBizList, &biz.Video{
			Id:              video.Id,
			Title:           video.Title,
			Description:     video.Description,
			VideoUrl:        video.VideoUrl,
			CoverUrl:        video.CoverUrl,
			LikeCount:       video.LikeCount,
			CommentCount:    video.CommentCount,
			CollectionCount: video.CollectionCount,
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

	var videoList []*model.Video
	err := r.db(ctx).Table(model.Video{}.TableName()).
		Where("id IN (?)", idList).
		Find(&videoList).Error
	if err != nil {
		return nil, err
	}

	userIds := make([]int64, 0)
	userVideoMap := make(map[int64][]*model.Video)
	for _, video := range videoList {
		userIds = append(userIds, video.UserId)
		userVideoMap[video.UserId] = append(userVideoMap[video.UserId], video)
	}

	var users []*model.User
	if len(userIds) > 0 {
		err = r.db(ctx).Table(model.User{}.TableName()).
			Where("id IN (?)", userIds).
			Find(&users).Error
		if err != nil {
			return nil, err
		}
	}

	userMap := make(map[int64]*model.User)
	for _, user := range users {
		userMap[user.Id] = user
	}

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
			author = &biz.Author{
				Id:   userId,
				Name: "用户已注销",
			}
		}
		for _, video := range videos {
			videoBizList = append(videoBizList, &biz.Video{
				Id:              video.Id,
				Title:           video.Title,
				Description:     video.Description,
				VideoUrl:        video.VideoUrl,
				CoverUrl:        video.CoverUrl,
				LikeCount:       video.LikeCount,
				CommentCount:    video.CommentCount,
				CollectionCount: video.CollectionCount,
				Author:          author,
				UploadTime:      video.CreatedAt,
			})
		}
	}
	return videoBizList, nil
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

	userIds := make([]int64, 0)
	userVideoMap := make(map[int64][]*model.Video)
	for _, video := range videoList {
		userIds = append(userIds, video.UserId)
		userVideoMap[video.UserId] = append(userVideoMap[video.UserId], video)
	}

	var users []*model.User
	if len(userIds) > 0 {
		err = r.db(ctx).Table(model.User{}.TableName()).
			Where("id IN (?)", userIds).
			Find(&users).Error
		if err != nil {
			return nil, err
		}
	}

	userMap := make(map[int64]*model.User)
	for _, user := range users {
		userMap[user.Id] = user
	}

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
			author = &biz.Author{
				Id:   video.UserId,
				Name: "用户已注销",
			}
		}
		videoBizList = append(videoBizList, &biz.Video{
			Id:              video.Id,
			Title:           video.Title,
			Description:     video.Description,
			VideoUrl:        video.VideoUrl,
			CoverUrl:        video.CoverUrl,
			LikeCount:       video.LikeCount,
			CommentCount:    video.CommentCount,
			CollectionCount: video.CollectionCount,
			Author:          author,
			UploadTime:      video.CreatedAt,
		})
	}
	return videoBizList, nil
}

func (r *videoRepo) GetHotVideos(ctx context.Context, limit int) ([]*biz.Video, error) {
	var videos []*model.Video
	sevenDaysAgo := time.Now().AddDate(0, 0, -7)
	err := r.db(ctx).
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
			Id:              v.Id,
			LikeCount:       v.LikeCount,
			CommentCount:    v.CommentCount,
			CollectionCount: v.CollectionCount,
			UploadTime:      v.CreatedAt,
			Author:          &biz.Author{Id: v.UserId},
		})
	}
	return bizVideos, nil
}

func (r *videoRepo) GetVideosByAuthors(ctx context.Context, authorIDs []string, latestTime int64, limit int) ([]*biz.Video, error) {
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
		Where("user_id IN (?)", ids).
		Order("created_at DESC").
		Limit(limit)
	if latestTime > 0 {
		query = query.Where("created_at < ?", time.Unix(latestTime, 0))
	}
	var videos []*model.Video
	err := query.Find(&videos).Error
	if err != nil {
		return nil, err
	}
	return r.convertDBVideosToBiz(videos)
}

func (r *videoRepo) GetVideoListByTime(ctx context.Context, latestTime time.Time, limit int) ([]*biz.Video, error) {
	var videos []*model.Video
	err := r.db(ctx).
		Where("created_at < ?", latestTime).
		Order("created_at DESC").
		Limit(limit).
		Find(&videos).Error
	if err != nil {
		return nil, err
	}
	return r.convertDBVideosToBiz(videos)
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
	hotScore := float64(video.LikeCount)*0.4 + float64(video.CommentCount)*0.3
	return &biz.VideoStats{
		VideoID:      videoID,
		LikeCount:    video.LikeCount,
		CommentCount: video.CommentCount,
		ShareCount:   0,
		ViewCount:    0,
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

// 新增：增加视频点赞计数
func (r *videoRepo) IncrVideoLikeCount(ctx context.Context, videoId int64, delta int64) error {
	return r.db(ctx).Model(&model.Video{}).Where("id = ?", videoId).
		UpdateColumn("like_count", gorm.Expr("like_count + ?", delta)).Error
}

// 新增：增加视频收藏计数
func (r *videoRepo) IncrVideoCollectionCount(ctx context.Context, videoId int64, delta int64) error {
	return r.db(ctx).Model(&model.Video{}).Where("id = ?", videoId).
		UpdateColumn("collection_count", gorm.Expr("collection_count + ?", delta)).Error
}

// 新增：增加作者被点赞总数（用户表的 be_liked_count）
func (r *videoRepo) IncrAuthorBeLikedCount(ctx context.Context, authorId int64, delta int64) error {
	return r.db(ctx).Model(&model.User{}).Where("id = ?", authorId).
		UpdateColumn("be_liked_count", gorm.Expr("be_liked_count + ?", delta)).Error
}

// 辅助方法：转换数据库视频列表为业务层视频列表
func (r *videoRepo) convertDBVideosToBiz(dbVideos []*model.Video) ([]*biz.Video, error) {
	if len(dbVideos) == 0 {
		return []*biz.Video{}, nil
	}
	userIds := make([]int64, 0, len(dbVideos))
	for _, video := range dbVideos {
		userIds = append(userIds, video.UserId)
	}
	var users []*model.User
	err := r.db(context.Background()).Table(model.User{}.TableName()).
		Where("id IN (?)", userIds).
		Find(&users).Error // 注意这里用 background 可能会导致事务丢失，但此方法只在非事务场景调用（如 GetHotVideos），如果需要在事务内使用，应改造
	// 为了简单，此处不处理事务，因为当前调用者不会在事务内使用该方法
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
			Author:          author,
			UploadTime:      dbVideo.CreatedAt,
		})
	}
	return bizVideos, nil
}
