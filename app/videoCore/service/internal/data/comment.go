package data

import (
	"context"
	"github.com/go-kratos/kratos/v2/log"
	"lehu-video/app/videoCore/service/internal/biz"
	"lehu-video/app/videoCore/service/internal/data/model"
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

func (r *commentRepo) ListCommentByVideoId(ctx context.Context, videoId int64) ([]*biz.Comment, error) {
	// 1. 获取视频下的所有顶级评论（parent_id = 0）
	var topComments []model.Comment
	err := r.data.db.WithContext(ctx).
		Where("video_id = ? AND parent_id = 0 AND is_deleted = ?", videoId, false).
		Order("created_at DESC").
		Find(&topComments).Error
	if err != nil {
		return nil, err
	}

	// 如果没有评论，返回空切片
	if len(topComments) == 0 {
		return []*biz.Comment{}, nil
	}

	// 2. 批量获取每个顶级评论的子评论数量
	commentCountMap, err := r.getCommentChildCounts(ctx, topComments)
	if err != nil {
		r.log.Warnf("failed to get comment counts: %v", err)
		// 继续执行，不返回错误
	}

	// 3. 批量获取每个顶级评论的前5条子评论
	childCommentMap, err := r.getFirstChildComments(ctx, topComments, 5)
	if err != nil {
		r.log.Warnf("failed to get first child comments: %v", err)
		// 继续执行，不返回错误
	}

	// 4. 转换为业务对象
	result := make([]*biz.Comment, len(topComments))
	for i, comment := range topComments {
		result[i] = r.convertToBizCommentWithChildren(&comment, commentCountMap, childCommentMap)
	}

	return result, nil
}

// 批量获取评论的子评论数量
func (r *commentRepo) getCommentChildCounts(ctx context.Context, comments []model.Comment) (map[int64]int64, error) {
	if len(comments) == 0 {
		return make(map[int64]int64), nil
	}

	var commentIds []int64
	for _, comment := range comments {
		commentIds = append(commentIds, comment.Id)
	}

	var countResults []struct {
		ParentID int64
		Count    int64
	}

	err := r.data.db.WithContext(ctx).
		Model(&model.Comment{}).
		Select("parent_id, COUNT(*) as count").
		Where("parent_id IN (?) AND is_deleted = ?", commentIds, false).
		Group("parent_id").
		Scan(&countResults).Error

	if err != nil {
		return nil, err
	}

	countMap := make(map[int64]int64)
	for _, result := range countResults {
		countMap[result.ParentID] = result.Count
	}

	return countMap, nil
}

// 批量获取每个评论的前N条子评论
func (r *commentRepo) getFirstChildComments(ctx context.Context, comments []model.Comment, limit int) (map[int64][]*biz.Comment, error) {
	if len(comments) == 0 {
		return make(map[int64][]*biz.Comment), nil
	}

	var commentIds []int64
	for _, comment := range comments {
		commentIds = append(commentIds, comment.Id)
	}

	// 查询所有相关子评论
	var childComments []model.Comment
	err := r.data.db.WithContext(ctx).
		Where("parent_id IN (?) AND is_deleted = ?", commentIds, false).
		Order("parent_id, created_at DESC").
		Find(&childComments).Error
	if err != nil {
		return nil, err
	}

	// 按父评论ID分组，并限制每个父评论的数量
	childMap := make(map[int64][]*biz.Comment)
	countMap := make(map[int64]int)

	for _, child := range childComments {
		parentID := child.ParentId

		// 如果已经达到限制数量，跳过
		if countMap[parentID] >= limit {
			continue
		}

		if childMap[parentID] == nil {
			childMap[parentID] = make([]*biz.Comment, 0, limit)
		}

		childMap[parentID] = append(childMap[parentID], r.convertModelToBiz(&child, 0))
		countMap[parentID]++
	}

	return childMap, nil
}

// 转换评论对象（包含子评论信息）
func (r *commentRepo) convertToBizCommentWithChildren(comment *model.Comment, countMap map[int64]int64, childMap map[int64][]*biz.Comment) *biz.Comment {
	bizComment := &biz.Comment{
		Id:           comment.Id,
		VideoId:      comment.VideoId,
		UserId:       comment.UserId,
		Content:      comment.Content,
		Date:         comment.CreatedAt.Format(time.DateTime),
		CreateTime:   comment.CreatedAt,
		ChildNumbers: countMap[comment.Id], // 子评论总数
		ParentId:     comment.ParentId,
		ToUserId:     comment.ToUserId,
	}

	// 设置前几条子评论
	if firstComments, exists := childMap[comment.Id]; exists {
		bizComment.FirstComments = firstComments
		// 如果子评论数量小于等于5，Comments和FirstComments相同
		if len(firstComments) <= 5 {
			bizComment.Comments = firstComments
		}
	}
	return bizComment
}

// 基础转换方法
func (r *commentRepo) convertModelToBiz(comment *model.Comment, childCount int64) *biz.Comment {
	bizComment := &biz.Comment{
		Id:           comment.Id,
		VideoId:      comment.VideoId,
		UserId:       comment.UserId,
		Content:      comment.Content,
		Date:         comment.CreatedAt.Format(time.DateTime),
		CreateTime:   comment.CreatedAt,
		ChildNumbers: childCount,
		ParentId:     comment.ParentId,
		ToUserId:     comment.ToUserId,
	}
	return bizComment
}

func (r *commentRepo) ListChildCommentById(ctx context.Context, commentId int64) ([]*biz.Comment, error) {

}
