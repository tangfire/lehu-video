package data

import (
	"context"
	"github.com/go-kratos/kratos/v2/log"
	"lehu-video/app/videoCore/service/internal/biz"
	"lehu-video/app/videoCore/service/internal/data/model"
	"lehu-video/app/videoCore/service/internal/pkg/utils"
	"time"
)

type commentRepo struct {
	data *Data
	log  *log.Helper
}

func NewCommentRepo(data *Data, logger log.Logger) biz.CommentRepo {
	return &commentRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *commentRepo) CreateComment(ctx context.Context, in *biz.Comment) error {
	comment := model.Comment{
		Id:       in.Id,
		VideoId:  in.VideoId,
		UserId:   in.UserId,
		Content:  in.Content,
		ParentId: in.ParentId,
		ToUserId: in.ToUserId,
	}

	return r.data.db.WithContext(ctx).Create(&comment).Error
}

func (r *commentRepo) RemoveComment(ctx context.Context, in *biz.Comment) error {
	comment := model.Comment{
		Id:       in.Id,
		VideoId:  in.VideoId,
		UserId:   in.UserId,
		ParentId: in.ParentId,
		ToUserId: in.ToUserId,
		Content:  in.Content,
	}
	err := r.data.db.WithContext(ctx).Table(model.Comment{}.TableName()).
		Where("id = ? and user_id = ?", comment.Id, comment.UserId).
		Delete(&comment).Error
	if err != nil {
		return err
	}
	return nil
}

func (r *commentRepo) ListCommentByVideoId(ctx context.Context, videoId int64, page int32, size int32) (int64, []*biz.Comment, error) {
	var commentList []model.Comment
	db := r.data.db.WithContext(ctx).Table(model.Comment{}.TableName()).
		Where("parent_id = ?", 0).
		Where("video_id = ?", videoId).
		Where("is_deleted = ?", false)

	var total int64
	err := db.Count(&total).Error
	if err != nil {
		return 0, nil, err
	}
	err = db.Offset(int((page - 1) * size)).Limit(int(size)).Find(&commentList).Error
	if err != nil {
		return 0, nil, err
	}
	commentIdList := utils.Slice2Slice(commentList, func(comment model.Comment) int64 {
		return comment.Id
	})

	var commentItemList []struct {
		ParentId int64 `json:"parent_id"`
		Count    int64 `json:"count"`
	}

	err = r.data.db.WithContext(ctx).Table(model.Comment{}.TableName()).
		Select("parent_id,count(parent_id) as count").
		Where("parent_id in (?)", commentIdList).
		Where("is_deleted = ?", false).
		Group("parent_id").
		Find(&commentItemList).Error
	if err != nil {
		return 0, nil, err
	}
	commentCountMap := make(map[int64]int64)
	for _, item := range commentItemList {
		commentCountMap[item.ParentId] = item.Count
	}

	var childCommentList []model.Comment
	err = r.data.db.WithContext(ctx).Table(model.Comment{}.TableName()).
		Where("parent_id in (?)", commentIdList).
		Where("is_deleted = ?", false).
		Find(&childCommentList).Error
	if err != nil {
		return 0, nil, err
	}
	childCommentMap := make(map[int64][]*biz.Comment)
	for _, comment := range childCommentList {
		tmp := &biz.Comment{
			Id:            comment.Id,
			VideoId:       comment.VideoId,
			UserId:        comment.UserId,
			ParentId:      comment.ParentId,
			ToUserId:      comment.ToUserId,
			Content:       comment.Content,
			Date:          comment.CreatedAt.Format(time.DateTime),
			CreateTime:    comment.CreatedAt,
			Comments:      nil,
			ChildNumbers:  0,
			FirstComments: nil,
		}
		childCommentMap[comment.ParentId] = append(childCommentMap[comment.ParentId], tmp)
	}
	var retCommentList []*biz.Comment
	for _, comment := range commentList {
		tmp := &biz.Comment{
			Id:            comment.Id,
			VideoId:       comment.VideoId,
			UserId:        comment.UserId,
			ParentId:      comment.ParentId,
			ToUserId:      comment.ToUserId,
			Content:       comment.Content,
			Date:          comment.CreatedAt.Format(time.DateTime),
			CreateTime:    comment.CreatedAt,
			Comments:      childCommentMap[comment.Id],
			ChildNumbers:  commentCountMap[comment.Id],
			FirstComments: nil,
		}
		retCommentList = append(retCommentList, tmp)
	}
	for _, comment := range retCommentList {
		if comment.Comments != nil && len(comment.Comments) > 0 {
			commentLen := len(comment.Comments)
			_len := min(commentLen, 5)
			comment.FirstComments = comment.Comments[:_len]
		}
	}
	return total, retCommentList, nil
}

func (r *commentRepo) ListChildCommentById(ctx context.Context, commentId int64, page int32, size int32) (int64, []*biz.Comment, error) {
	var commentList []model.Comment
	db := r.data.db.WithContext(ctx).Table(model.Comment{}.TableName()).
		Where("parent_id = ?", commentId).
		Where("is_deleted = ?", false)
	var total int64
	err := db.Count(&total).Error
	if err != nil {
		return 0, nil, err
	}

	err = db.Offset(int((page - 1) * size)).Limit(int(size)).Find(&commentList).Error
	if err != nil {
		return 0, nil, err
	}
	childCommentList := utils.Slice2Slice(commentList, func(comment model.Comment) *biz.Comment {
		return &biz.Comment{
			Id:            comment.Id,
			VideoId:       comment.VideoId,
			UserId:        comment.UserId,
			ParentId:      comment.ParentId,
			ToUserId:      comment.ToUserId,
			Content:       comment.Content,
			Date:          comment.CreatedAt.Format(time.DateTime),
			CreateTime:    comment.CreatedAt,
			Comments:      nil,
			ChildNumbers:  0,
			FirstComments: nil,
		}
	})
	return total, childCommentList, nil
}
