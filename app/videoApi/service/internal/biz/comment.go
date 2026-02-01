package biz

import (
	"context"
	"errors"
	"github.com/spf13/cast"

	"github.com/go-kratos/kratos/v2/log"
	"lehu-video/app/videoApi/service/internal/pkg/utils/claims"
)

type CreateCommentInput struct {
	VideoId     string
	Content     string
	ParentId    string
	ReplyUserId string
}

type CreateCommentOutput struct {
	Comment *Comment
}

type RemoveCommentInput struct {
	Id string
}

type ListChildCommentInput struct {
	CommentId string
	PageStats *PageStats
}

type ListChildCommentOutput struct {
	Comments []*Comment
	Total    int64
}

type ListComment4VideoInput struct {
	VideoId   string
	PageStats *PageStats
}

type ListComment4VideoOutput struct {
	Comments []*Comment
	Total    int64
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

	// 调用core服务创建评论
	comment, err := uc.core.CreateComment(ctx, userId, input.Content, input.VideoId, input.ParentId, input.ReplyUserId)
	if err != nil {
		return nil, err
	}

	// 获取用户信息
	userInfo, err := uc.core.GetUserInfo(ctx, userId, "0")
	if err != nil {
		// 弱依赖
		log.Context(ctx).Warnf("failed to get user info: %v", err)
	} else {
		comment.User = uc.generateCommentUserInfo(userInfo)
	}

	// 获取回复用户信息
	if comment.ReplyUser != nil && cast.ToInt64(comment.ReplyUser.Id) != 0 {
		replyUserInfo, err := uc.core.GetUserInfo(ctx, comment.ReplyUser.Id, "0")
		if err != nil {
			// 弱依赖
			log.Context(ctx).Warnf("failed to get reply user info: %v", err)
		} else {
			comment.ReplyUser = uc.generateCommentUserInfo(replyUserInfo)
		}
	}

	// 获取子评论的用户信息
	if len(comment.Comments) > 0 {
		uc.enrichCommentsWithUserInfo(ctx, comment.Comments)
	}

	return &CreateCommentOutput{Comment: comment}, nil
}

func (uc *CommentUsecase) generateCommentUserInfo(userInfo *UserInfo) *CommentUser {
	if userInfo == nil {
		return &CommentUser{}
	}

	// TODO: 需要调用follow服务获取是否已关注
	// 这里暂时设置为false，需要根据实际情况实现
	isFollowing := false

	return &CommentUser{
		Id:          userInfo.Id,
		Name:        userInfo.Name,
		Avatar:      userInfo.Avatar,
		IsFollowing: isFollowing,
	}
}

func (uc *CommentUsecase) enrichCommentsWithUserInfo(ctx context.Context, comments []*Comment) {
	if len(comments) == 0 {
		return
	}

	// 收集所有需要查询的用户ID
	var userIds []string
	for _, comment := range comments {
		if comment.User != nil {
			userIds = append(userIds, comment.User.Id)
		}
		if comment.ReplyUser != nil && cast.ToInt64(comment.ReplyUser.Id) != 0 {
			userIds = append(userIds, comment.ReplyUser.Id)
		}
		// 递归处理子评论
		if len(comment.Comments) > 0 {
			uc.enrichCommentsWithUserInfo(ctx, comment.Comments)
		}
	}

	if len(userIds) == 0 {
		return
	}

	// 批量获取用户信息
	userInfos, err := uc.core.GetUserInfoByIdList(ctx, userIds)
	if err != nil {
		// 弱依赖
		log.Context(ctx).Warnf("failed to get user info list: %v", err)
		return
	}

	// 创建用户信息映射
	userInfoMap := make(map[string]*UserInfo)
	for _, user := range userInfos {
		userInfoMap[user.Id] = user
	}

	// 填充用户信息
	for _, comment := range comments {
		if comment.User != nil {
			if userInfo, ok := userInfoMap[comment.User.Id]; ok {
				comment.User = uc.generateCommentUserInfo(userInfo)
			}
		}
		if comment.ReplyUser != nil && cast.ToInt64(comment.ReplyUser.Id) != 0 {
			if userInfo, ok := userInfoMap[comment.ReplyUser.Id]; ok {
				comment.ReplyUser = uc.generateCommentUserInfo(userInfo)
			}
		}
	}
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

	if commentInfo == nil || commentInfo.User == nil || commentInfo.User.Id != userId {
		return errors.New("无权删除评论")
	}

	err = uc.core.RemoveComment(ctx, input.Id, userId)
	if err != nil {
		return errors.New("删除评论失败")
	}
	return nil
}

func (uc *CommentUsecase) ListChildComment(ctx context.Context, input *ListChildCommentInput) (*ListChildCommentOutput, error) {
	total, childComments, err := uc.core.ListChildComment(ctx, input.CommentId, input.PageStats)
	if err != nil {
		return nil, err
	}

	// 获取用户信息并组装结果
	result := uc.assembleCommentListResult(ctx, childComments)

	return &ListChildCommentOutput{
		Comments: result,
		Total:    total,
	}, nil
}

func (uc *CommentUsecase) ListComment4Video(ctx context.Context, input *ListComment4VideoInput) (*ListComment4VideoOutput, error) {
	total, comments, err := uc.core.ListComment4Video(ctx, input.VideoId, input.PageStats)
	if err != nil {
		return nil, errors.New("获取评论失败")
	}

	// 获取用户信息并组装结果
	result := uc.assembleCommentListResult(ctx, comments)

	return &ListComment4VideoOutput{
		Comments: result,
		Total:    total,
	}, nil
}

func (uc *CommentUsecase) assembleCommentListResult(ctx context.Context, commentList []*Comment) []*Comment {
	if len(commentList) == 0 {
		return []*Comment{}
	}

	// 收集所有需要查询的用户ID
	var userIds []string
	for _, comment := range commentList {
		if comment.User != nil {
			userIds = append(userIds, comment.User.Id)
		}
		if comment.ReplyUser != nil && cast.ToInt64(comment.ReplyUser.Id) != 0 {
			userIds = append(userIds, comment.ReplyUser.Id)
		}
		// 递归处理子评论
		if len(comment.Comments) > 0 {
			// 递归收集子评论的用户ID
			for _, child := range comment.Comments {
				if child.User != nil {
					userIds = append(userIds, child.User.Id)
				}
				if child.ReplyUser != nil && cast.ToInt64(child.ReplyUser.Id) != 0 {
					userIds = append(userIds, child.ReplyUser.Id)
				}
			}
		}
	}

	// 获取用户信息
	var userInfoMap map[string]*UserInfo
	if len(userIds) > 0 {
		userInfos, err := uc.core.GetUserInfoByIdList(ctx, userIds)
		if err != nil {
			// 弱依赖
			log.Context(ctx).Warnf("failed to get user info list: %v", err)
		} else {
			userInfoMap = make(map[string]*UserInfo)
			for _, user := range userInfos {
				userInfoMap[user.Id] = user
			}
		}
	}

	// 组装结果
	var result []*Comment
	for _, comment := range commentList {
		var userResp *CommentUser
		if comment.User != nil && userInfoMap != nil {
			if userInfo, ok := userInfoMap[comment.User.Id]; ok {
				userResp = uc.generateCommentUserInfo(userInfo)
			}
		}

		var replyUserResp *CommentUser
		if comment.ReplyUser != nil && cast.ToInt64(comment.ReplyUser.Id) != 0 && userInfoMap != nil {
			if userInfo, ok := userInfoMap[comment.ReplyUser.Id]; ok {
				replyUserResp = uc.generateCommentUserInfo(userInfo)
			}
		}

		// 递归处理子评论
		var childComments []*Comment
		if len(comment.Comments) > 0 {
			childComments = uc.assembleCommentListResult(ctx, comment.Comments)
		}

		result = append(result, &Comment{
			Id:         comment.Id,
			VideoId:    comment.VideoId,
			ParentId:   comment.ParentId,
			User:       userResp,
			ReplyUser:  replyUserResp,
			Content:    comment.Content,
			Date:       comment.Date,
			LikeCount:  comment.LikeCount,
			ReplyCount: comment.ReplyCount,
			Comments:   childComments,
		})
	}

	return result
}
