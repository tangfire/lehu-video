package data

import (
	"context"
	core "lehu-video/api/videoCore/service/v1"
	"lehu-video/app/videoApi/service/internal/biz"
	"lehu-video/app/videoApi/service/internal/pkg/utils/respcheck"
)

func (r *CoreAdapterImpl) CountComments4Video(ctx context.Context, videoIdList []string) (map[string]int64, error) {
	resp, err := r.comment.CountComment4Video(ctx, &core.CountComment4VideoReq{
		VideoId: videoIdList,
	})
	if err != nil {
		return nil, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return nil, err
	}
	result := make(map[string]int64)
	for _, item := range resp.Results {
		result[item.Id] = item.Count
	}

	return result, nil
}

func (r *CoreAdapterImpl) CreateComment(ctx context.Context, userId string, content string, videoId string, parentId string, replyUserId string) (*biz.Comment, error) {
	resp, err := r.comment.CreateComment(ctx, &core.CreateCommentReq{
		VideoId:     videoId,
		UserId:      userId,
		Content:     content,
		ParentId:    parentId,
		ReplyUserId: replyUserId,
	})
	if err != nil {
		return nil, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return nil, err
	}
	comment := resp.Comment
	childComments := comment.Comments
	var retChildComments []*biz.Comment
	for _, childComment := range childComments {
		retChildComments = append(retChildComments, &biz.Comment{
			Id:         childComment.Id,
			VideoId:    childComment.VideoId,
			ParentId:   childComment.ParentId,
			User:       nil,
			ReplyUser:  nil,
			Content:    childComment.Content,
			Date:       childComment.Date,
			LikeCount:  childComment.LikeCount,
			ReplyCount: childComment.ReplyCount,
			Comments:   nil,
		})
	}
	retComment := &biz.Comment{
		Id:       comment.Id,
		VideoId:  comment.VideoId,
		ParentId: comment.ParentId,
		User: &biz.CommentUser{
			Id:          comment.UserId,
			Name:        "",
			Avatar:      "",
			IsFollowing: false,
		},
		ReplyUser: &biz.CommentUser{
			Id:          comment.ReplyUserId,
			Name:        "",
			Avatar:      "",
			IsFollowing: false,
		},
		Content:    comment.Content,
		Date:       comment.Date,
		LikeCount:  comment.LikeCount,
		ReplyCount: comment.ReplyCount,
		Comments:   retChildComments,
	}
	return retComment, nil
}

func (r *CoreAdapterImpl) GetCommentById(ctx context.Context, commentId string) (*biz.Comment, error) {
	resp, err := r.comment.GetCommentById(ctx, &core.GetCommentByIdReq{
		CommentId: commentId,
	})
	if err != nil {
		return nil, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return nil, err
	}
	comment := resp.Comment
	retComment := &biz.Comment{
		Id:         comment.Id,
		VideoId:    comment.VideoId,
		ParentId:   comment.ParentId,
		Content:    comment.Content,
		Date:       comment.Date,
		LikeCount:  comment.LikeCount,
		ReplyCount: comment.ReplyCount,
		Comments:   nil,
		User: &biz.CommentUser{
			Id: comment.UserId,
		},
	}
	return retComment, nil
}

func (r *CoreAdapterImpl) RemoveComment(ctx context.Context, commentId, userId string) error {
	resp, err := r.comment.RemoveComment(ctx, &core.RemoveCommentReq{
		CommentId: commentId,
		UserId:    userId,
	})
	if err != nil {
		return err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return err
	}
	return nil
}

func (r *CoreAdapterImpl) ListChildComment(ctx context.Context, commentId string, pageStats *biz.PageStats) (int64, []*biz.Comment, error) {
	resp, err := r.comment.ListChildComment4Comment(ctx, &core.ListChildComment4CommentReq{
		CommentId: commentId,
		PageStats: &core.PageStatsReq{
			Page: int32(pageStats.Page),
			Size: int32(pageStats.PageSize),
		},
	})
	if err != nil {
		return 0, nil, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return 0, nil, err
	}
	var childComments []*biz.Comment
	for _, comment := range resp.CommentList {
		childComments = append(childComments, &biz.Comment{
			Id:         comment.Id,
			VideoId:    comment.VideoId,
			ParentId:   comment.ParentId,
			User:       &biz.CommentUser{Id: comment.UserId},
			ReplyUser:  &biz.CommentUser{Id: comment.ReplyUserId},
			Content:    comment.Content,
			Date:       comment.Date,
			LikeCount:  comment.LikeCount,
			ReplyCount: comment.ReplyCount,
			Comments:   nil,
		})
	}
	return int64(resp.PageStats.Total), childComments, nil
}

func (r *CoreAdapterImpl) ListComment4Video(ctx context.Context, videoId string, pageStats *biz.PageStats) (int64, []*biz.Comment, error) {
	resp, err := r.comment.ListComment4Video(ctx, &core.ListComment4VideoReq{
		VideoId: videoId,
		PageStats: &core.PageStatsReq{
			Page: int32(pageStats.Page),
			Size: int32(pageStats.PageSize),
		},
	})
	if err != nil {
		return 0, nil, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return 0, nil, err
	}
	var Comments []*biz.Comment
	for _, comment := range resp.CommentList {
		childComments := comment.Comments
		var retChildComments []*biz.Comment
		for _, childComment := range childComments {
			retChildComments = append(retChildComments, &biz.Comment{
				Id:         childComment.Id,
				VideoId:    childComment.VideoId,
				ParentId:   childComment.ParentId,
				User:       &biz.CommentUser{Id: childComment.Id},
				ReplyUser:  &biz.CommentUser{Id: childComment.ReplyUserId},
				Content:    childComment.Content,
				Date:       childComment.Date,
				LikeCount:  childComment.LikeCount,
				ReplyCount: childComment.ReplyCount,
				Comments:   nil,
			})
		}
		Comments = append(Comments, &biz.Comment{
			Id:         comment.Id,
			VideoId:    comment.VideoId,
			ParentId:   comment.ParentId,
			User:       &biz.CommentUser{Id: comment.UserId},
			ReplyUser:  &biz.CommentUser{Id: comment.ReplyUserId},
			Content:    comment.Content,
			Date:       comment.Date,
			LikeCount:  comment.LikeCount,
			ReplyCount: comment.ReplyCount,
			Comments:   retChildComments,
		})
	}
	return int64(resp.PageStats.Total), Comments, nil
}
