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

func (r *commentRepo) Create(ctx context.Context, comment *biz.Comment) (int64, error) {
	dbComment := &model.Comment{
		Id:        comment.ID,
		VideoId:   comment.VideoID,
		UserId:    comment.UserID,
		ParentId:  comment.ParentID,
		ToUserId:  comment.ReplyUserID,
		Content:   comment.Content,
		IsDeleted: false,
	}

	err := r.data.db.WithContext(ctx).Create(dbComment).Error
	if err != nil {
		return 0, err
	}

	return dbComment.Id, nil
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
		Id:        comment.ID,
		VideoId:   comment.VideoID,
		UserId:    comment.UserID,
		ParentId:  comment.ParentID,
		ToUserId:  comment.ReplyUserID,
		Content:   comment.Content,
		IsDeleted: comment.IsDeleted,
	}

	return r.data.db.WithContext(ctx).Save(dbComment).Error
}

func (r *commentRepo) Delete(ctx context.Context, id int64, userID int64) error {
	return r.data.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&model.Comment{}).Error
}

func (r *commentRepo) SoftDelete(ctx context.Context, id int64, userID int64) error {
	// 更新is_deleted字段为true
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
		Where("id IN (?) AND is_deleted = ?", ids, false).
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
	// 注意：这里要去除已删除的评论，is_deleted = false
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

// GetLikeCounts 获取评论点赞数统计
func (r *commentRepo) GetLikeCounts(ctx context.Context, commentIDs []int64) (map[int64]int64, error) {
	if len(commentIDs) == 0 {
		return map[int64]int64{}, nil
	}

	// 这里需要根据实际的点赞表结构来查询
	// 假设有一个comment_likes表，包含comment_id字段
	type LikeResult struct {
		CommentID int64
		Count     int64
	}

	var results []LikeResult
	// 示例查询，需要根据实际表结构调整
	err := r.data.db.WithContext(ctx).
		Table("comment_likes").
		Select("comment_id, COUNT(*) as count").
		Where("comment_id IN (?)", commentIDs).
		Group("comment_id").
		Find(&results).Error

	if err != nil {
		// 如果表不存在或其他错误，返回空map
		r.log.Warnf("查询点赞数失败: %v", err)
		return map[int64]int64{}, nil
	}

	counts := make(map[int64]int64)
	for _, result := range results {
		counts[result.CommentID] = result.Count
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
				if len(ids) == 0 {
					db = db.Where("1 = 0") // 如果传入空数组，返回空结果
				} else {
					db = db.Where("parent_id IN (?)", ids)
				}
			} else {
				db = db.Where("parent_id = ?", value)
			}
		case "is_deleted":
			db = db.Where("is_deleted = ?", value)
		case "limit":
			if limit, ok := value.(int64); ok {
				db = db.Limit(int(limit))
			} else if limit, ok := value.(int32); ok {
				db = db.Limit(int(limit))
			} else if limit, ok := value.(int); ok {
				db = db.Limit(limit)
			}
		case "offset":
			if offset, ok := value.(int64); ok {
				db = db.Offset(int(offset))
			} else if offset, ok := value.(int32); ok {
				db = db.Offset(int(offset))
			} else if offset, ok := value.(int); ok {
				db = db.Offset(offset)
			}
		case "order_by":
			db = db.Order(value.(string))
		case "group_by":
			db = db.Group(value.(string))
		default:
			r.log.Warnf("Unknown condition key: %s", key)
		}
	}

	// 默认只查询未删除的记录，除非显式指定
	if _, hasIsDeleted := condition["is_deleted"]; !hasIsDeleted {
		db = db.Where("is_deleted = ?", false)
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
		LikeCount:   0, // 默认值，需要单独查询
		ReplyCount:  0, // 默认值，需要单独查询
		IsDeleted:   dbComment.IsDeleted,
	}
}
