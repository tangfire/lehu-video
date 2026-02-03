package biz

import (
	"context"
	"errors"
	"github.com/go-kratos/kratos/v2/log"
	"lehu-video/app/videoApi/service/internal/pkg/utils/claims"
)

type Comment struct {
	Id         string
	VideoId    string
	ParentId   string
	User       *CommentUser
	ReplyUser  *CommentUser
	Content    string
	Date       string
	LikeCount  int64
	ReplyCount int64
	Comments   []*Comment
}

// CommentUser 评论用户
type CommentUser struct {
	Id          string
	Name        string
	Avatar      string
	IsFollowing bool
}

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
	core        CoreAdapter
	userUsecase *UserUsecase // 注入 UserUsecase 来获取聚合的用户信息
	log         *log.Helper
}

func NewCommentUsecase(core CoreAdapter, userUsecase *UserUsecase, logger log.Logger) *CommentUsecase {
	return &CommentUsecase{
		core:        core,
		userUsecase: userUsecase,
		log:         log.NewHelper(logger),
	}
}

func (uc *CommentUsecase) CreateComment(ctx context.Context, input *CreateCommentInput) (*CreateCommentOutput, error) {
	currentUserID, err := claims.GetUserId(ctx)
	if err != nil {
		return nil, errors.New("获取用户信息失败")
	}

	// 调用core服务创建评论
	comment, err := uc.core.CreateComment(ctx, currentUserID, input.Content, input.VideoId, input.ParentId, input.ReplyUserId)
	if err != nil {
		return nil, err
	}

	// 获取当前用户信息
	userInput := &GetUserInfoInput{
		UserID:         currentUserID,
		IncludePrivate: false,
	}

	userOutput, err := uc.userUsecase.GetCompleteUserInfo(ctx, userInput)
	if err != nil {
		// 弱依赖
		log.Context(ctx).Warnf("failed to get user info: %v", err)
	} else if userOutput != nil && userOutput.User != nil {
		comment.User = uc.generateCommentUserInfo(userOutput.User)
	}

	// 获取回复用户信息
	if input.ReplyUserId != "" && input.ReplyUserId != "0" {
		replyUserInput := &GetUserInfoInput{
			UserID:         input.ReplyUserId,
			IncludePrivate: false,
		}

		replyUserOutput, err := uc.userUsecase.GetCompleteUserInfo(ctx, replyUserInput)
		if err != nil {
			// 弱依赖
			log.Context(ctx).Warnf("failed to get reply user info: %v", err)
		} else if replyUserOutput != nil && replyUserOutput.User != nil {
			comment.ReplyUser = uc.generateCommentUserInfo(replyUserOutput.User)
		}
	}

	// 获取子评论的用户信息
	if len(comment.Comments) > 0 {
		uc.enrichCommentsWithUserInfo(ctx, comment.Comments, currentUserID)
	}

	return &CreateCommentOutput{Comment: comment}, nil
}

func (uc *CommentUsecase) generateCommentUserInfo(user *UserInfo) *CommentUser {
	if user == nil {
		return &CommentUser{}
	}

	return &CommentUser{
		Id:          user.ID,
		Name:        user.Name,
		Avatar:      user.Avatar,
		IsFollowing: user.IsFollowing,
	}
}

func (uc *CommentUsecase) enrichCommentsWithUserInfo(ctx context.Context, comments []*Comment, currentUserID string) {
	if len(comments) == 0 {
		return
	}

	// 收集所有需要查询的用户ID
	userIDs := make([]string, 0)
	for _, comment := range comments {
		if comment.User != nil && comment.User.Id != "" {
			userIDs = append(userIDs, comment.User.Id)
		}
		if comment.ReplyUser != nil && comment.ReplyUser.Id != "" {
			userIDs = append(userIDs, comment.ReplyUser.Id)
		}
		// 递归处理子评论
		if len(comment.Comments) > 0 {
			uc.enrichCommentsWithUserInfo(ctx, comment.Comments, currentUserID)
		}
	}

	if len(userIDs) == 0 {
		return
	}

	// 批量获取用户信息
	batchInput := &BatchGetUserInfoInput{
		UserIDs:         userIDs,
		CurrentUserID:   currentUserID,
		IncludePrivate:  false,
		IncludeRelation: true,
	}

	batchOutput, err := uc.userUsecase.BatchGetUserInfo(ctx, batchInput)
	if err != nil {
		// 弱依赖
		log.Context(ctx).Warnf("failed to get user info list: %v", err)
		return
	}

	// 填充用户信息
	for _, comment := range comments {
		if comment.User != nil && comment.User.Id != "" {
			if user, ok := batchOutput.Users[comment.User.Id]; ok {
				comment.User = uc.generateCommentUserInfo(user)
			}
		}
		if comment.ReplyUser != nil && comment.ReplyUser.Id != "" {
			if user, ok := batchOutput.Users[comment.ReplyUser.Id]; ok {
				comment.ReplyUser = uc.generateCommentUserInfo(user)
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
	currentUserID, _ := claims.GetUserId(ctx)

	total, childComments, err := uc.core.ListChildComment(ctx, input.CommentId, input.PageStats)
	if err != nil {
		return nil, err
	}

	// 获取用户信息并组装结果
	result := uc.assembleCommentListResult(ctx, childComments, currentUserID)

	return &ListChildCommentOutput{
		Comments: result,
		Total:    total,
	}, nil
}

func (uc *CommentUsecase) ListComment4Video(ctx context.Context, input *ListComment4VideoInput) (*ListComment4VideoOutput, error) {
	currentUserID, _ := claims.GetUserId(ctx)

	total, comments, err := uc.core.ListComment4Video(ctx, input.VideoId, input.PageStats)
	if err != nil {
		return nil, errors.New("获取评论失败")
	}

	// 获取用户信息并组装结果
	result := uc.assembleCommentListResult(ctx, comments, currentUserID)

	return &ListComment4VideoOutput{
		Comments: result,
		Total:    total,
	}, nil
}

func (uc *CommentUsecase) assembleCommentListResult(ctx context.Context, commentList []*Comment, currentUserID string) []*Comment {
	if len(commentList) == 0 {
		return []*Comment{}
	}

	// 收集所有需要查询的用户ID
	userIDs := make([]string, 0)
	for _, comment := range commentList {
		if comment.User != nil && comment.User.Id != "" {
			userIDs = append(userIDs, comment.User.Id)
		}
		if comment.ReplyUser != nil && comment.ReplyUser.Id != "" && comment.ReplyUser.Id != "0" {
			userIDs = append(userIDs, comment.ReplyUser.Id)
		}
		// 递归处理子评论
		if len(comment.Comments) > 0 {
			for _, child := range comment.Comments {
				if child.User != nil && child.User.Id != "" {
					userIDs = append(userIDs, child.User.Id)
				}
				if child.ReplyUser != nil && child.ReplyUser.Id != "" && child.ReplyUser.Id != "0" {
					userIDs = append(userIDs, child.ReplyUser.Id)
				}
			}
		}
	}

	// 批量获取用户信息
	var userInfoMap map[string]*UserInfo
	if len(userIDs) > 0 {
		batchInput := &BatchGetUserInfoInput{
			UserIDs:         userIDs,
			CurrentUserID:   currentUserID,
			IncludePrivate:  false,
			IncludeRelation: true,
		}

		batchOutput, err := uc.userUsecase.BatchGetUserInfo(ctx, batchInput)
		if err != nil {
			// 弱依赖
			log.Context(ctx).Warnf("failed to get user info list: %v", err)
		} else {
			userInfoMap = batchOutput.Users
		}
	}

	// 组装结果
	result := make([]*Comment, 0, len(commentList))
	for _, comment := range commentList {
		var userResp *CommentUser
		if comment.User != nil && comment.User.Id != "" && userInfoMap != nil {
			if userInfo, ok := userInfoMap[comment.User.Id]; ok {
				userResp = uc.generateCommentUserInfo(userInfo)
			}
		}

		var replyUserResp *CommentUser
		if comment.ReplyUser != nil && comment.ReplyUser.Id != "" && comment.ReplyUser.Id != "0" && userInfoMap != nil {
			if userInfo, ok := userInfoMap[comment.ReplyUser.Id]; ok {
				replyUserResp = uc.generateCommentUserInfo(userInfo)
			}
		}

		// 递归处理子评论
		var childComments []*Comment
		if len(comment.Comments) > 0 {
			childComments = uc.assembleCommentListResult(ctx, comment.Comments, currentUserID)
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
