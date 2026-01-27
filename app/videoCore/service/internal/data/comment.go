package data

import (
	"context"
	"errors"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"

	"lehu-video/app/videoCore/service/internal/biz"
	"lehu-video/app/videoCore/service/internal/data/model"
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

func (r *commentRepo) Create(ctx context.Context, comment *biz.Comment) error {
	dbComment := &model.Comment{
		Id:       comment.ID,
		VideoId:  comment.VideoID,
		UserId:   comment.UserID,
		ParentId: comment.ParentID,
		ToUserId: comment.ReplyUserID,
		Content:  comment.Content,
	}

	return r.data.db.WithContext(ctx).Create(dbComment).Error
}

func (r *commentRepo) GetByID(ctx context.Context, id int64) (*biz.Comment, error) {
	var dbComment model.Comment
	err := r.data.db.WithContext(ctx).Where("id = ?", id).First(&dbComment).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return r.toBizComment(&dbComment), nil
}

func (r *commentRepo) Update(ctx context.Context, comment *biz.Comment) error {
	dbComment := &model.Comment{
		Id:       comment.ID,
		VideoId:  comment.VideoID,
		UserId:   comment.UserID,
		ParentId: comment.ParentID,
		ToUserId: comment.ReplyUserID,
		Content:  comment.Content,
	}

	return r.data.db.WithContext(ctx).Save(dbComment).Error
}

func (r *commentRepo) Delete(ctx context.Context, id int64, userID int64) error {
	return r.data.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&model.Comment{}).Error
}

func (r *commentRepo) SoftDelete(ctx context.Context, id int64, userID int64) error {
	// 这里假设有is_deleted字段，如果没有需要修改
	return r.data.db.WithContext(ctx).
		Model(&model.Comment{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("is_deleted", true).Error
}

func (r *commentRepo) FindByCondition(ctx context.Context, condition map[string]interface{}) ([]*biz.Comment, error) {
	db := r.data.db.WithContext(ctx).Model(&model.Comment{})
	db = r.applyConditions(db, condition)

	var dbComments []*model.Comment
	err := db.Find(&dbComments).Error
	if err != nil {
		return nil, err
	}

	// 转换为业务层结构
	comments := make([]*biz.Comment, 0, len(dbComments))
	for _, dbComment := range dbComments {
		comments = append(comments, r.toBizComment(dbComment))
	}

	return comments, nil
}

func (r *commentRepo) CountByCondition(ctx context.Context, condition map[string]interface{}) (int64, error) {
	db := r.data.db.WithContext(ctx).Model(&model.Comment{})
	db = r.applyConditions(db, condition)

	var count int64
	err := db.Count(&count).Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (r *commentRepo) FindByIDs(ctx context.Context, ids []int64) ([]*biz.Comment, error) {
	if len(ids) == 0 {
		return []*biz.Comment{}, nil
	}

	var dbComments []*model.Comment
	err := r.data.db.WithContext(ctx).
		Where("id IN (?)", ids).
		Find(&dbComments).Error
	if err != nil {
		return nil, err
	}

	comments := make([]*biz.Comment, 0, len(dbComments))
	for _, dbComment := range dbComments {
		comments = append(comments, r.toBizComment(dbComment))
	}

	return comments, nil
}

func (r *commentRepo) CountByVideoIDs(ctx context.Context, videoIDs []int64) (map[int64]int64, error) {
	if len(videoIDs) == 0 {
		return map[int64]int64{}, nil
	}

	type Result struct {
		VideoID int64
		Count   int64
	}

	var results []Result
	err := r.data.db.WithContext(ctx).
		Model(&model.Comment{}).
		Select("video_id, COUNT(*) as count").
		Where("video_id IN (?) AND is_deleted = ?", videoIDs, false).
		Group("video_id").
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	counts := make(map[int64]int64)
	for _, result := range results {
		counts[result.VideoID] = result.Count
	}

	// 确保所有videoID都有值
	for _, videoID := range videoIDs {
		if _, exists := counts[videoID]; !exists {
			counts[videoID] = 0
		}
	}

	return counts, nil
}

func (r *commentRepo) CountByUserIDs(ctx context.Context, userIDs []int64) (map[int64]int64, error) {
	if len(userIDs) == 0 {
		return map[int64]int64{}, nil
	}

	type Result struct {
		UserID int64
		Count  int64
	}

	var results []Result
	err := r.data.db.WithContext(ctx).
		Model(&model.Comment{}).
		Select("user_id, COUNT(*) as count").
		Where("user_id IN (?) AND is_deleted = ?", userIDs, false).
		Group("user_id").
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	counts := make(map[int64]int64)
	for _, result := range results {
		counts[result.UserID] = result.Count
	}

	// 确保所有userID都有值
	for _, userID := range userIDs {
		if _, exists := counts[userID]; !exists {
			counts[userID] = 0
		}
	}

	return counts, nil
}

// CountGroupByParentID 新增方法：按父评论ID分组计数
func (r *commentRepo) CountGroupByParentID(ctx context.Context, parentIDs []int64) (map[int64]int64, error) {
	if len(parentIDs) == 0 {
		return map[int64]int64{}, nil
	}

	type Result struct {
		ParentID int64
		Count    int64
	}

	var results []Result
	err := r.data.db.WithContext(ctx).
		Model(&model.Comment{}).
		Select("parent_id, COUNT(*) as count").
		Where("parent_id IN (?) AND is_deleted = ?", parentIDs, false).
		Group("parent_id").
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	counts := make(map[int64]int64)
	for _, result := range results {
		counts[result.ParentID] = result.Count
	}

	// 确保所有parentID都有值（即使为0）
	for _, parentID := range parentIDs {
		if _, exists := counts[parentID]; !exists {
			counts[parentID] = 0
		}
	}

	return counts, nil
}

// applyConditions 应用查询条件
func (r *commentRepo) applyConditions(db *gorm.DB, condition map[string]interface{}) *gorm.DB {
	for key, value := range condition {
		switch key {
		case "video_id":
			db = db.Where("video_id = ?", value)
		case "user_id":
			db = db.Where("user_id = ?", value)
		case "parent_id":
			if ids, ok := value.([]int64); ok {
				db = db.Where("parent_id IN (?)", ids)
			} else {
				db = db.Where("parent_id = ?", value)
			}
		case "is_deleted":
			db = db.Where("is_deleted = ?", value)
		case "limit":
			db = db.Limit(int(value.(int64)))
		case "offset":
			db = db.Offset(int(value.(int64)))
		case "order_by":
			db = db.Order(value.(string))
		case "group_by":
			db = db.Group(value.(string))
		default:
			r.log.Warnf("Unknown condition key: %s", key)
		}
	}
	return db
}

// toBizComment 将数据库模型转换为业务层模型
func (r *commentRepo) toBizComment(dbComment *model.Comment) *biz.Comment {
	return &biz.Comment{
		ID:          dbComment.Id,
		VideoID:     dbComment.VideoId,
		UserID:      dbComment.UserId,
		ParentID:    dbComment.ParentId,
		ReplyUserID: dbComment.ToUserId,
		Content:     dbComment.Content,
		CreateTime:  dbComment.CreatedAt,
		IsDeleted:   false, // 根据实际表结构调整
	}
}
