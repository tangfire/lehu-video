package biz

import (
	"context"
	"errors"
	"github.com/go-kratos/kratos/v2/log"
	"lehu-video/app/videoApi/service/internal/pkg/utils/claims"
)

type CreateCommentInput struct {
	VideoId     int64
	Content     string
	ParentId    int64
	ReplyUserId int64
}

type CreateCommentOutput struct {
	comment *Comment
}

type RemoveCommentInput struct {
	Id int64
}

type ListChildCommentInput struct {
	CommentId int64
	PageStats *PageStats
}

type ListChildCommentsOutput struct {
	comments []*Comment
	Total    int64
}

type ListComment4VideoInput struct {
	VideoId   int64
	PageStats *PageStats
}

type ListComment4VideoOutput struct {
	comments []*Comment
	Total    int64
}

type Comment struct {
	Id         int64
	VideoId    int64
	ParentId   int64
	User       *CommentUser
	ReplyUser  *CommentUser
	Content    string
	Date       string
	LikeCount  int64
	ReplyCount int64
	Comments   []*Comment
}

type CommentUser struct {
	Id          int64
	Name        string
	Avatar      string
	IsFollowing bool
}

type CommentUsecase struct {
	core CoreAdapter
	log  *log.Helper
}

func NewCommentUsecase(core CoreAdapter, logger log.Logger) *CommentUsecase {
	return &CommentUsecase{
		core: core,
		log:  log.NewHelper(logger),
	}
}

func (uc *CommentUsecase) CreateComment(ctx context.Context, input *CreateCommentInput) (*CreateCommentOutput, error) {
	userId, err := claims.GetUserId(ctx)
	if err != nil {
		return nil, errors.New("获取用户信息失败")
	}

	comment, err := uc.core.CreateComment(ctx, userId, input.Content, input.ParentId, input.ReplyUserId, input.ReplyUserId)
	if err != nil {
		return nil, err
	}
	userInfo, err := uc.core.GetUserInfo(ctx, userId, 0)
	if err != nil {
		return nil, err
	}

	userResp := uc.generateCommentUserInfo(userInfo)
	var replyUserResp *CommentUser
	if comment.ReplyUser.Id != 0 {
		userInfo, err := uc.core.GetUserInfo(ctx, comment.ReplyUser.Id, 0)
		if err != nil {
			// 弱依赖
			log.Context(ctx).Warnf("failed to get user info: %v", err)
		} else {
			replyUserResp = uc.generateCommentUserInfo(userInfo)
		}
	}
	comment.User = userResp
	comment.ReplyUser = replyUserResp

	return &CreateCommentOutput{comment: comment}, nil

}

func (uc *CommentUsecase) generateCommentUserInfo(userInfo *UserInfo) (commentUser *CommentUser) {
	if userInfo == nil {
		return commentUser
	}
	commentUser = &CommentUser{
		Id:     userInfo.Id,
		Name:   userInfo.Name,
		Avatar: userInfo.Avatar,
		// todo 增加是否已关注
		IsFollowing: true,
	}
	return commentUser
}

func (uc *CommentUsecase) RemoveComment(ctx context.Context, input *RemoveCommentInput) error {
	userId, err := claims.GetUserId(ctx)
	if err != nil {
		return errors.New("获取用户信息失败")
	}

	commentInfo, err := uc.core.GetCommentById(ctx, input.Id)
	if err != nil {
		return errors.New("评论不存在")
	}

	if commentInfo.User.Id != userId {
		return errors.New("无权删除评论")
	}

	err = uc.core.RemoveComment(ctx, input.Id, userId)
	if err != nil {
		return errors.New("删除评论失败")
	}
	return nil
}
