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

// ExecTx 执行事务
func (r *commentRepo) ExecTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 将 tx 存入 context
		txCtx := context.WithValue(ctx, "db", tx)
		return fn(txCtx)
	})
}

// db 获取当前 context 中的 DB 对象（支持事务）
func (r *commentRepo) db(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value("db").(*gorm.DB); ok {
		return tx
	}
	return r.data.db.WithContext(ctx)
}

func (r *commentRepo) Create(ctx context.Context, comment *biz.Comment) (int64, error) {
	dbComment := &model.Comment{
		Id:         comment.ID,
		VideoId:    comment.VideoID,
		UserId:     comment.UserID,
		ParentId:   comment.ParentID,
		ToUserId:   comment.ReplyUserID,
		Content:    comment.Content,
		LikeCount:  0,
		ReplyCount: 0,
		IsDeleted:  false,
		CreatedAt:  comment.CreateTime,
	}
	err := r.db(ctx).Create(dbComment).Error
	if err != nil {
		return 0, err
	}
	return dbComment.Id, nil
}

func (r *commentRepo) GetByID(ctx context.Context, id int64) (*biz.Comment, error) {
	var dbComment model.Comment
	err := r.db(ctx).Where("id = ?", id).First(&dbComment).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return r.toBizComment(&dbComment), nil
}

func (r *commentRepo) Update(ctx context.Context, comment *biz.Comment) error {
	dbComment := r.toDBComment(comment)
	return r.db(ctx).Save(dbComment).Error
}

func (r *commentRepo) Delete(ctx context.Context, id int64, userID int64) error {
	return r.db(ctx).Where("id = ? AND user_id = ?", id, userID).Delete(&model.Comment{}).Error
}

func (r *commentRepo) SoftDelete(ctx context.Context, id int64, userID int64) error {
	return r.db(ctx).Model(&model.Comment{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("is_deleted", true).Error
}

func (r *commentRepo) FindByIDs(ctx context.Context, ids []int64) ([]*biz.Comment, error) {
	if len(ids) == 0 {
		return []*biz.Comment{}, nil
	}
	var dbComments []*model.Comment
	err := r.db(ctx).Where("id IN (?) AND is_deleted = ?", ids, false).Find(&dbComments).Error
	if err != nil {
		return nil, err
	}
	result := make([]*biz.Comment, 0, len(dbComments))
	for _, dbc := range dbComments {
		result = append(result, r.toBizComment(dbc))
	}
	return result, nil
}

func (r *commentRepo) FindByCondition(ctx context.Context, condition map[string]interface{}) ([]*biz.Comment, error) {
	db := r.db(ctx).Model(&model.Comment{})
	db = r.applyConditions(db, condition)
	var dbComments []*model.Comment
	err := db.Find(&dbComments).Error
	if err != nil {
		return nil, err
	}
	result := make([]*biz.Comment, 0, len(dbComments))
	for _, dbc := range dbComments {
		result = append(result, r.toBizComment(dbc))
	}
	return result, nil
}

func (r *commentRepo) CountByCondition(ctx context.Context, condition map[string]interface{}) (int64, error) {
	db := r.db(ctx).Model(&model.Comment{})
	db = r.applyConditions(db, condition)
	var count int64
	err := db.Count(&count).Error
	return count, err
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
	err := r.db(ctx).Model(&model.Comment{}).
		Select("video_id, COUNT(*) as count").
		Where("video_id IN (?) AND is_deleted = ?", videoIDs, false).
		Group("video_id").
		Find(&results).Error
	if err != nil {
		return nil, err
	}
	counts := make(map[int64]int64)
	for _, res := range results {
		counts[res.VideoID] = res.Count
	}
	for _, vid := range videoIDs {
		if _, ok := counts[vid]; !ok {
			counts[vid] = 0
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
	err := r.db(ctx).Model(&model.Comment{}).
		Select("user_id, COUNT(*) as count").
		Where("user_id IN (?) AND is_deleted = ?", userIDs, false).
		Group("user_id").
		Find(&results).Error
	if err != nil {
		return nil, err
	}
	counts := make(map[int64]int64)
	for _, res := range results {
		counts[res.UserID] = res.Count
	}
	for _, uid := range userIDs {
		if _, ok := counts[uid]; !ok {
			counts[uid] = 0
		}
	}
	return counts, nil
}

// IncrReplyCount 原子增加父评论的回复数（用于事务内）
func (r *commentRepo) IncrReplyCount(ctx context.Context, parentID int64, delta int) error {
	return r.db(ctx).Model(&model.Comment{}).
		Where("id = ?", parentID).
		Update("reply_count", gorm.Expr("reply_count + ?", delta)).Error
}

func (r *commentRepo) GetReplyCount(ctx context.Context, commentID int64) (int64, error) {
	var c model.Comment
	err := r.db(ctx).Select("reply_count").Where("id = ?", commentID).First(&c).Error
	if err != nil {
		return 0, err
	}
	return c.ReplyCount, nil
}

// applyConditions 条件构建（与之前相同，略作调整）
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
					db = db.Where("1 = 0")
				} else {
					db = db.Where("parent_id IN (?)", ids)
				}
			} else {
				db = db.Where("parent_id = ?", value)
			}
		case "is_deleted":
			db = db.Where("is_deleted = ?", value)
		case "limit":
			if l, ok := value.(int64); ok {
				db = db.Limit(int(l))
			} else if l, ok := value.(int); ok {
				db = db.Limit(l)
			}
		case "offset":
			if o, ok := value.(int64); ok {
				db = db.Offset(int(o))
			} else if o, ok := value.(int); ok {
				db = db.Offset(o)
			}
		case "order_by":
			db = db.Order(value.(string))
		default:
			r.log.Warnf("Unknown condition key: %s", key)
		}
	}
	if _, has := condition["is_deleted"]; !has {
		db = db.Where("is_deleted = ?", false)
	}
	return db
}

func (r *commentRepo) BatchSoftDelete(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db(ctx).Model(&model.Comment{}).
		Where("id IN (?)", ids).
		Update("is_deleted", true).Error
}

func (r *commentRepo) toBizComment(dbComment *model.Comment) *biz.Comment {
	return &biz.Comment{
		ID:          dbComment.Id,
		VideoID:     dbComment.VideoId,
		UserID:      dbComment.UserId,
		ParentID:    dbComment.ParentId,
		ReplyUserID: dbComment.ToUserId,
		Content:     dbComment.Content,
		CreateTime:  dbComment.CreatedAt,
		LikeCount:   dbComment.LikeCount,
		ReplyCount:  dbComment.ReplyCount,
		IsDeleted:   dbComment.IsDeleted,
	}
}

func (r *commentRepo) toDBComment(bizComment *biz.Comment) *model.Comment {
	return &model.Comment{
		Id:         bizComment.ID,
		VideoId:    bizComment.VideoID,
		UserId:     bizComment.UserID,
		ParentId:   bizComment.ParentID,
		ToUserId:   bizComment.ReplyUserID,
		Content:    bizComment.Content,
		LikeCount:  bizComment.LikeCount,
		ReplyCount: bizComment.ReplyCount,
		IsDeleted:  bizComment.IsDeleted,
		CreatedAt:  bizComment.CreateTime,
	}
}
