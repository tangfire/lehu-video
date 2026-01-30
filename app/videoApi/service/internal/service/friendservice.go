package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	v1 "lehu-video/api/videoApi/service/v1"
	"lehu-video/app/videoApi/service/internal/biz"
)

type FriendServiceService struct {
	v1.UnimplementedFriendServiceServer
	uc  *biz.FriendUsecase
	log *log.Helper
}

func NewFriendServiceService(uc *biz.FriendUsecase, logger log.Logger) *FriendServiceService {
	return &FriendServiceService{
		uc:  uc,
		log: log.NewHelper(logger),
	}
}

func (s *FriendServiceService) SearchUsers(ctx context.Context, req *v1.SearchUsersReq) (*v1.SearchUsersResp, error) {
	input := &biz.SearchUsersInput{
		Keyword:  req.Keyword,
		Page:     req.PageStats.Page,
		PageSize: req.PageStats.Size,
	}

	output, err := s.uc.SearchUsers(ctx, input)
	if err != nil {
		return nil, err
	}

	// 转换用户信息
	users := make([]*v1.User, 0, len(output.Users))
	for _, user := range output.Users {
		users = append(users, &v1.User{
			Id:             user.Id,
			Name:           user.Name,
			Avatar:         user.Avatar,
			Nickname:       user.Nickname,
			Signature:      user.Signature,
			Gender:         user.Gender,
			OnlineStatus:   user.OnlineStatus,
			LastOnlineTime: user.LastOnlineTime.Format("2006-01-02 15:04:05"),
		})
	}

	return &v1.SearchUsersResp{
		Users: users,
		PageStats: &v1.PageStatsResp{
			Total: int32(output.Total),
		},
	}, nil
}

func (s *FriendServiceService) SendFriendApply(ctx context.Context, req *v1.SendFriendApplyReq) (*v1.SendFriendApplyResp, error) {
	input := &biz.SendFriendApplyInput{
		ReceiverID:  req.ReceiverId,
		ApplyReason: req.ApplyReason,
	}

	output, err := s.uc.SendFriendApply(ctx, input)
	if err != nil {
		return nil, err
	}

	return &v1.SendFriendApplyResp{
		ApplyId: output.ApplyID,
	}, nil
}

func (s *FriendServiceService) HandleFriendApply(ctx context.Context, req *v1.HandleFriendApplyReq) (*v1.HandleFriendApplyResp, error) {
	input := &biz.HandleFriendApplyInput{
		ApplyID: req.ApplyId,
		Accept:  req.Accept,
	}

	err := s.uc.HandleFriendApply(ctx, input)
	if err != nil {
		return nil, err
	}

	return &v1.HandleFriendApplyResp{}, nil
}

func (s *FriendServiceService) ListFriendApplies(ctx context.Context, req *v1.ListFriendAppliesReq) (*v1.ListFriendAppliesResp, error) {
	input := &biz.ListFriendAppliesInput{
		Page:     req.PageStats.Page,
		PageSize: req.PageStats.Size,
		Status:   req.Status,
	}

	output, err := s.uc.ListFriendApplies(ctx, input)
	if err != nil {
		return nil, err
	}

	// 转换申请信息
	applies := make([]*v1.FriendApplyInfo, 0, len(output.Applies))
	for _, apply := range output.Applies {
		// 注意：这里需要获取申请人和接收人的详细信息
		// 为了简化，这里只返回基础信息，实际项目中需要从core服务获取详细信息
		friendApplyInfo := &v1.FriendApplyInfo{
			Id:          apply.ID,
			ApplyReason: apply.ApplyReason,
			Status:      apply.Status,
			CreatedAt:   apply.CreatedAt.Format("2006-01-02 15:04:05"),
		}

		if apply.HandledAt != nil {
			handledAt := apply.HandledAt.Format("2006-01-02 15:04:05")
			friendApplyInfo.HandledAt = handledAt
		}

		// 申请人信息
		friendApplyInfo.Applicant = &v1.User{
			Id: apply.ApplicantID,
		}

		// 接收人信息
		friendApplyInfo.Receiver = &v1.User{
			Id: apply.ReceiverID,
		}

		applies = append(applies, friendApplyInfo)
	}

	return &v1.ListFriendAppliesResp{
		Applies: applies,
		PageStats: &v1.PageStatsResp{
			Total: int32(output.Total),
		},
	}, nil
}

func (s *FriendServiceService) ListFriends(ctx context.Context, req *v1.ListFriendsReq) (*v1.ListFriendsResp, error) {
	input := &biz.ListFriendsInput{
		Page:      req.PageStats.Page,
		PageSize:  req.PageStats.Size,
		GroupName: req.GroupName,
	}

	output, err := s.uc.ListFriends(ctx, input)
	if err != nil {
		return nil, err
	}

	// 转换好友信息
	friends := make([]*v1.FriendInfo, 0, len(output.Friends))
	for _, friend := range output.Friends {
		friendInfo := &v1.FriendInfo{
			Id:        friend.ID,
			Remark:    friend.Remark,
			GroupName: friend.GroupName,
			Status:    friend.Status,
			CreatedAt: friend.CreatedAt.Format("2006-01-02 15:04:05"),
		}

		if friend.Friend != nil {
			friendInfo.Friend = &v1.User{
				Id:             friend.Friend.Id,
				Name:           friend.Friend.Name,
				Avatar:         friend.Friend.Avatar,
				Nickname:       friend.Friend.Nickname,
				Signature:      friend.Friend.Signature,
				Gender:         friend.Friend.Gender,
				OnlineStatus:   friend.Friend.OnlineStatus,
				LastOnlineTime: friend.Friend.LastOnlineTime.Format("2006-01-02 15:04:05"),
			}
		}

		friends = append(friends, friendInfo)
	}

	return &v1.ListFriendsResp{
		Friends: friends,
		PageStats: &v1.PageStatsResp{
			Total: int32(output.Total),
		},
	}, nil
}

func (s *FriendServiceService) DeleteFriend(ctx context.Context, req *v1.DeleteFriendReq) (*v1.DeleteFriendResp, error) {
	input := &biz.DeleteFriendInput{
		FriendID: req.FriendId,
	}

	err := s.uc.DeleteFriend(ctx, input)
	if err != nil {
		return nil, err
	}

	return &v1.DeleteFriendResp{}, nil
}

func (s *FriendServiceService) UpdateFriendRemark(ctx context.Context, req *v1.UpdateFriendRemarkReq) (*v1.UpdateFriendRemarkResp, error) {
	input := &biz.UpdateFriendRemarkInput{
		FriendID: req.FriendId,
		Remark:   req.Remark,
	}

	err := s.uc.UpdateFriendRemark(ctx, input)
	if err != nil {
		return nil, err
	}

	return &v1.UpdateFriendRemarkResp{}, nil
}

func (s *FriendServiceService) SetFriendGroup(ctx context.Context, req *v1.SetFriendGroupReq) (*v1.SetFriendGroupResp, error) {
	input := &biz.SetFriendGroupInput{
		FriendID:  req.FriendId,
		GroupName: req.GroupName,
	}

	err := s.uc.SetFriendGroup(ctx, input)
	if err != nil {
		return nil, err
	}

	return &v1.SetFriendGroupResp{}, nil
}

func (s *FriendServiceService) CheckFriendRelation(ctx context.Context, req *v1.CheckFriendRelationReq) (*v1.CheckFriendRelationResp, error) {
	input := &biz.CheckFriendRelationInput{
		TargetID: req.TargetId,
	}

	output, err := s.uc.CheckFriendRelation(ctx, input)
	if err != nil {
		return nil, err
	}

	return &v1.CheckFriendRelationResp{
		IsFriend: output.IsFriend,
		Status:   output.Status,
	}, nil
}

func (s *FriendServiceService) GetUserOnlineStatus(ctx context.Context, req *v1.GetUserOnlineStatusReq) (*v1.GetUserOnlineStatusResp, error) {
	input := &biz.GetUserOnlineStatusInput{
		UserID: req.UserId,
	}

	output, err := s.uc.GetUserOnlineStatus(ctx, input)
	if err != nil {
		return nil, err
	}

	return &v1.GetUserOnlineStatusResp{
		OnlineStatus:   output.OnlineStatus,
		LastOnlineTime: output.LastOnlineTime,
	}, nil
}

func (s *FriendServiceService) BatchGetUserOnlineStatus(ctx context.Context, req *v1.BatchGetUserOnlineStatusReq) (*v1.BatchGetUserOnlineStatusResp, error) {
	input := &biz.BatchGetUserOnlineStatusInput{
		UserIDs: req.UserIds,
	}

	output, err := s.uc.BatchGetUserOnlineStatus(ctx, input)
	if err != nil {
		return nil, err
	}

	return &v1.BatchGetUserOnlineStatusResp{
		OnlineStatus: output.OnlineStatus,
	}, nil
}
