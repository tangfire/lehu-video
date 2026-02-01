package service

import (
	"context"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/spf13/cast"
	pb "lehu-video/api/videoChat/service/v1"
	"lehu-video/app/videoChat/service/internal/biz"
	"lehu-video/app/videoChat/service/internal/pkg/utils"
)

type FriendServiceService struct {
	pb.UnimplementedFriendServiceServer
	uc  *biz.FriendUsecase
	log *log.Helper
}

func NewFriendServiceService(uc *biz.FriendUsecase, logger log.Logger) *FriendServiceService {
	return &FriendServiceService{
		uc:  uc,
		log: log.NewHelper(logger),
	}
}

func (s *FriendServiceService) SearchUsers(ctx context.Context, req *pb.SearchUsersReq) (*pb.SearchUsersResp, error) {
	// ✅ 构建Query
	query := &biz.SearchUsersCommand{
		Keyword: req.Keyword,
		PageStats: &biz.PageStats{
			Page:     int(req.PageStats.Page),
			PageSize: int(req.PageStats.Size),
		},
	}

	// ✅ 调用业务层
	result, err := s.uc.SearchUsers(ctx, query)
	if err != nil {
		return &pb.SearchUsersResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	// ✅ 转换为proto结构
	users := make([]*pb.User, 0, len(result.Users))
	for _, user := range result.Users {
		users = append(users, &pb.User{
			Id:             cast.ToString(user.ID),
			Name:           user.Username,
			Nickname:       user.Nickname,
			Avatar:         user.Avatar,
			Signature:      user.Signature,
			Gender:         user.Gender,
			OnlineStatus:   user.OnlineStatus,
			LastOnlineTime: user.LastOnlineTime.Format("2006-01-02 15:04:05"),
		})
	}

	return &pb.SearchUsersResp{
		Meta:  utils.GetSuccessMeta(),
		Users: users,
		PageStats: &pb.PageStatsResp{
			Total: int32(result.Total),
		},
	}, nil
}

func (s *FriendServiceService) SendFriendApply(ctx context.Context, req *pb.SendFriendApplyReq) (*pb.SendFriendApplyResp, error) {
	// ✅ 构建Command
	cmd := &biz.SendFriendApplyCommand{
		ApplicantID: cast.ToInt64(req.ApplicantId),
		ReceiverID:  cast.ToInt64(req.ReceiverId),
		ApplyReason: req.ApplyReason,
	}

	// ✅ 调用业务层
	result, err := s.uc.SendFriendApply(ctx, cmd)
	if err != nil {
		return &pb.SendFriendApplyResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.SendFriendApplyResp{
		Meta:    utils.GetSuccessMeta(),
		ApplyId: cast.ToString(result.ApplyID),
	}, nil
}

func (s *FriendServiceService) HandleFriendApply(ctx context.Context, req *pb.HandleFriendApplyReq) (*pb.HandleFriendApplyResp, error) {
	// ✅ 构建Command
	cmd := &biz.HandleFriendApplyCommand{
		ApplyID:   cast.ToInt64(req.ApplyId),
		HandlerID: cast.ToInt64(req.HandlerId),
		Accept:    req.Accept,
	}

	// ✅ 调用业务层
	_, err := s.uc.HandleFriendApply(ctx, cmd)
	if err != nil {
		return &pb.HandleFriendApplyResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.HandleFriendApplyResp{
		Meta: utils.GetSuccessMeta(),
	}, nil
}

func (s *FriendServiceService) ListFriendApplies(ctx context.Context, req *pb.ListFriendAppliesReq) (*pb.ListFriendAppliesResp, error) {
	// ✅ 构建Query
	query := &biz.ListFriendAppliesQuery{
		UserID: cast.ToInt64(req.UserId),
		PageStats: &biz.PageStats{
			Page:     int(req.PageStats.Page),
			PageSize: int(req.PageStats.Size),
		},
	}

	// 处理可选参数
	if req.Status != nil {
		status := *req.Status
		query.Status = &status
	}

	// ✅ 调用业务层
	result, err := s.uc.ListFriendApplies(ctx, query)
	if err != nil {
		return &pb.ListFriendAppliesResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	// ✅ 转换为proto结构
	applies := make([]*pb.FriendApplyInfo, 0, len(result.Applies))
	for _, apply := range result.Applies {
		// 获取申请人信息
		applicantInfo, _ := s.uc.GetUserInfo(ctx, apply.ApplicantID)
		// 获取接收人信息
		receiverInfo, _ := s.uc.GetUserInfo(ctx, apply.ReceiverID)

		var handledAt string
		if apply.HandledAt != nil {
			handledAt = apply.HandledAt.Format("2006-01-02 15:04:05")
		}

		applies = append(applies, &pb.FriendApplyInfo{
			Id: cast.ToString(apply.ID),
			Applicant: &pb.User{
				Id:       cast.ToString(apply.ID),
				Name:     applicantInfo.Username,
				Nickname: applicantInfo.Nickname,
				Avatar:   applicantInfo.Avatar,
			},
			Receiver: &pb.User{
				Id:       cast.ToString(apply.ReceiverID),
				Name:     receiverInfo.Username,
				Nickname: receiverInfo.Nickname,
				Avatar:   receiverInfo.Avatar,
			},
			ApplyReason: apply.ApplyReason,
			Status:      apply.Status,
			HandledAt:   handledAt,
			CreatedAt:   apply.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &pb.ListFriendAppliesResp{
		Meta:    utils.GetSuccessMeta(),
		Applies: applies,
		PageStats: &pb.PageStatsResp{
			Total: int32(result.Total),
		},
	}, nil
}

func (s *FriendServiceService) ListFriends(ctx context.Context, req *pb.ListFriendsReq) (*pb.ListFriendsResp, error) {
	// ✅ 构建Query
	query := &biz.ListFriendsQuery{
		UserID: cast.ToInt64(req.UserId),
		PageStats: &biz.PageStats{
			Page:     int(req.PageStats.Page),
			PageSize: int(req.PageStats.Size),
		},
	}

	// 处理可选参数
	if req.GroupName != nil {
		query.GroupName = req.GroupName
	}

	// ✅ 调用业务层
	result, err := s.uc.ListFriends(ctx, query)
	if err != nil {
		return &pb.ListFriendsResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	// ✅ 转换为proto结构
	friends := make([]*pb.FriendInfo, 0, len(result.Friends))
	for _, friend := range result.Friends {
		friends = append(friends, &pb.FriendInfo{
			Id: cast.ToString(friend.ID),
			Friend: &pb.User{
				Id:             cast.ToString(friend.ID),
				Name:           friend.Friend.Username,
				Nickname:       friend.Friend.Nickname,
				Avatar:         friend.Friend.Avatar,
				Signature:      friend.Friend.Signature,
				Gender:         friend.Friend.Gender,
				OnlineStatus:   friend.Friend.OnlineStatus,
				LastOnlineTime: friend.Friend.LastOnlineTime.Format("2006-01-02 15:04:05"),
			},
			Remark:    friend.Remark,
			GroupName: friend.GroupName,
			Status:    friend.Status,
			CreatedAt: friend.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &pb.ListFriendsResp{
		Meta:    utils.GetSuccessMeta(),
		Friends: friends,
		PageStats: &pb.PageStatsResp{
			Total: int32(result.Total),
		},
	}, nil
}

func (s *FriendServiceService) DeleteFriend(ctx context.Context, req *pb.DeleteFriendReq) (*pb.DeleteFriendResp, error) {
	// ✅ 构建Command
	cmd := &biz.DeleteFriendCommand{
		UserID:   cast.ToInt64(req.UserId),
		FriendID: cast.ToInt64(req.FriendId),
	}

	// ✅ 调用业务层
	_, err := s.uc.DeleteFriend(ctx, cmd)
	if err != nil {
		return &pb.DeleteFriendResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.DeleteFriendResp{
		Meta: utils.GetSuccessMeta(),
	}, nil
}

func (s *FriendServiceService) UpdateFriendRemark(ctx context.Context, req *pb.UpdateFriendRemarkReq) (*pb.UpdateFriendRemarkResp, error) {
	// ✅ 构建Command
	cmd := &biz.UpdateFriendRemarkCommand{
		UserID:   cast.ToInt64(req.UserId),
		FriendID: cast.ToInt64(req.FriendId),
		Remark:   req.Remark,
	}

	// ✅ 调用业务层
	_, err := s.uc.UpdateFriendRemark(ctx, cmd)
	if err != nil {
		return &pb.UpdateFriendRemarkResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.UpdateFriendRemarkResp{
		Meta: utils.GetSuccessMeta(),
	}, nil
}

func (s *FriendServiceService) SetFriendGroup(ctx context.Context, req *pb.SetFriendGroupReq) (*pb.SetFriendGroupResp, error) {
	// ✅ 构建Command
	cmd := &biz.SetFriendGroupCommand{
		UserID:    cast.ToInt64(req.UserId),
		FriendID:  cast.ToInt64(req.FriendId),
		GroupName: req.GroupName,
	}

	// ✅ 调用业务层
	_, err := s.uc.SetFriendGroup(ctx, cmd)
	if err != nil {
		return &pb.SetFriendGroupResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.SetFriendGroupResp{
		Meta: utils.GetSuccessMeta(),
	}, nil
}

func (s *FriendServiceService) CheckFriendRelation(ctx context.Context, req *pb.CheckFriendRelationReq) (*pb.CheckFriendRelationResp, error) {
	// ✅ 调用业务层
	isFriend, status, err := s.uc.CheckFriendRelation(ctx, cast.ToInt64(req.UserId), cast.ToInt64(req.TargetId))
	if err != nil {
		return &pb.CheckFriendRelationResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.CheckFriendRelationResp{
		Meta:     utils.GetSuccessMeta(),
		IsFriend: isFriend,
		Status:   status,
	}, nil
}

func (s *FriendServiceService) GetUserOnlineStatus(ctx context.Context, req *pb.GetUserOnlineStatusReq) (*pb.GetUserOnlineStatusResp, error) {
	// ✅ 构建Query
	query := &biz.GetUserOnlineStatusQuery{
		UserID: cast.ToInt64(req.UserId),
	}

	// ✅ 调用业务层
	result, err := s.uc.GetUserOnlineStatus(ctx, query)
	if err != nil {
		return &pb.GetUserOnlineStatusResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.GetUserOnlineStatusResp{
		Meta:           utils.GetSuccessMeta(),
		OnlineStatus:   result.Status,
		LastOnlineTime: result.LastOnlineTime.Format("2006-01-02 15:04:05"),
	}, nil
}

func (s *FriendServiceService) BatchGetUserOnlineStatus(ctx context.Context, req *pb.BatchGetUserOnlineStatusReq) (*pb.BatchGetUserOnlineStatusResp, error) {
	// ✅ 构建Query
	ids := make([]int64, 0, len(req.UserIds))
	for _, userID := range req.UserIds {
		ids = append(ids, cast.ToInt64(userID))
	}

	query := &biz.BatchGetUserOnlineStatusQuery{
		UserIDs: ids,
	}

	// ✅ 调用业务层
	result, err := s.uc.BatchGetUserOnlineStatus(ctx, query)
	if err != nil {
		return &pb.BatchGetUserOnlineStatusResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}
	os := make(map[string]int32)
	for k, v := range result.OnlineStatus {
		os[cast.ToString(k)] = v
	}

	return &pb.BatchGetUserOnlineStatusResp{
		Meta:         utils.GetSuccessMeta(),
		OnlineStatus: os,
	}, nil
}
