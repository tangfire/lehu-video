package service

import (
	"context"
	"github.com/spf13/cast"
	"time"

	pb "lehu-video/api/videoChat/service/v1"
	"lehu-video/app/videoChat/service/internal/biz"
	"lehu-video/app/videoChat/service/internal/pkg/utils"
)

type FriendServiceService struct {
	pb.UnimplementedFriendServiceServer
	uc *biz.FriendUsecase
}

func NewFriendServiceService(uc *biz.FriendUsecase) *FriendServiceService {
	return &FriendServiceService{uc: uc}
}

func (s *FriendServiceService) SendFriendApply(ctx context.Context, req *pb.SendFriendApplyReq) (*pb.SendFriendApplyResp, error) {
	cmd := &biz.SendFriendApplyCommand{
		ApplicantID: cast.ToInt64(req.ApplicantId),
		ReceiverID:  cast.ToInt64(req.ReceiverId),
		ApplyReason: req.ApplyReason,
	}

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
	cmd := &biz.HandleFriendApplyCommand{
		ApplyID:   cast.ToInt64(req.ApplyId),
		HandlerID: cast.ToInt64(req.HandlerId),
		Accept:    req.Accept,
	}

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
	query := &biz.ListFriendAppliesQuery{
		UserID: cast.ToInt64(req.UserId),
		Page:   int(req.PageStats.Page),
		Limit:  int(req.PageStats.Size),
	}

	if req.Status != nil {
		status := *req.Status
		query.Status = &status
	}

	result, err := s.uc.ListFriendApplies(ctx, query)
	if err != nil {
		return &pb.ListFriendAppliesResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	// 转换为proto格式
	applies := make([]*pb.FriendApplyInfo, 0, len(result.Applies))
	for _, apply := range result.Applies {
		var applicant, receiver *pb.User
		var handledAt string

		if apply.Applicant != nil {
			applicant = &pb.User{
				Id:             cast.ToString(apply.Applicant.ID),
				Name:           apply.Applicant.Name,
				Nickname:       apply.Applicant.Nickname,
				Avatar:         apply.Applicant.Avatar,
				Signature:      apply.Applicant.Signature,
				Gender:         apply.Applicant.Gender,
				OnlineStatus:   apply.Applicant.OnlineStatus,
				LastOnlineTime: formatTime(apply.Applicant.LastOnlineTime),
			}
		}

		if apply.Receiver != nil {
			receiver = &pb.User{
				Id:             cast.ToString(apply.Receiver.ID),
				Name:           apply.Receiver.Name,
				Nickname:       apply.Receiver.Nickname,
				Avatar:         apply.Receiver.Avatar,
				Signature:      apply.Receiver.Signature,
				Gender:         apply.Receiver.Gender,
				OnlineStatus:   apply.Receiver.OnlineStatus,
				LastOnlineTime: formatTime(apply.Receiver.LastOnlineTime),
			}
		}

		if apply.HandledAt != nil {
			handledAt = formatTime(*apply.HandledAt)
		}

		applies = append(applies, &pb.FriendApplyInfo{
			Id:          cast.ToString(apply.ID),
			Applicant:   applicant,
			Receiver:    receiver,
			ApplyReason: apply.ApplyReason,
			Status:      apply.Status,
			HandledAt:   handledAt,
			CreatedAt:   formatTime(apply.CreatedAt),
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
	query := &biz.ListFriendsQuery{
		UserID: cast.ToInt64(req.UserId),
		Page:   int(req.PageStats.Page),
		Limit:  int(req.PageStats.Size),
	}

	if req.GroupName != nil {
		query.GroupName = req.GroupName
	}

	result, err := s.uc.ListFriends(ctx, query)
	if err != nil {
		return &pb.ListFriendsResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	// 转换为proto格式
	friends := make([]*pb.FriendInfo, 0, len(result.Friends))
	for _, friend := range result.Friends {
		var pbFriend *pb.User
		if friend.Friend != nil {
			pbFriend = &pb.User{
				Id:             cast.ToString(friend.Friend.ID),
				Name:           friend.Friend.Name,
				Nickname:       friend.Friend.Nickname,
				Avatar:         friend.Friend.Avatar,
				Signature:      friend.Friend.Signature,
				Gender:         friend.Friend.Gender,
				OnlineStatus:   friend.Friend.OnlineStatus,
				LastOnlineTime: formatTime(friend.Friend.LastOnlineTime),
			}
		}

		friends = append(friends, &pb.FriendInfo{
			Id:        cast.ToString(friend.ID),
			Friend:    pbFriend,
			Remark:    friend.Remark,
			GroupName: friend.GroupName,
			Status:    friend.Status,
			CreatedAt: formatTime(friend.CreatedAt),
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
	cmd := &biz.DeleteFriendCommand{
		UserID:   cast.ToInt64(req.UserId),
		FriendID: cast.ToInt64(req.FriendId),
	}

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
	cmd := &biz.UpdateFriendRemarkCommand{
		UserID:   cast.ToInt64(req.UserId),
		FriendID: cast.ToInt64(req.FriendId),
		Remark:   req.Remark,
	}

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
	cmd := &biz.SetFriendGroupCommand{
		UserID:    cast.ToInt64(req.UserId),
		FriendID:  cast.ToInt64(req.FriendId),
		GroupName: req.GroupName,
	}

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
	query := &biz.CheckFriendRelationQuery{
		UserID:   cast.ToInt64(req.UserId),
		TargetID: cast.ToInt64(req.TargetId),
	}

	result, err := s.uc.CheckFriendRelation(ctx, query)
	if err != nil {
		return &pb.CheckFriendRelationResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.CheckFriendRelationResp{
		Meta:     utils.GetSuccessMeta(),
		IsFriend: result.IsFriend,
		Status:   result.Status,
	}, nil
}

func (s *FriendServiceService) GetUserOnlineStatus(ctx context.Context, req *pb.GetUserOnlineStatusReq) (*pb.GetUserOnlineStatusResp, error) {
	query := &biz.GetUserOnlineStatusQuery{
		UserID: cast.ToInt64(req.UserId),
	}

	result, err := s.uc.GetUserOnlineStatus(ctx, query)
	if err != nil {
		return &pb.GetUserOnlineStatusResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.GetUserOnlineStatusResp{
		Meta:           utils.GetSuccessMeta(),
		OnlineStatus:   result.Status,
		LastOnlineTime: formatTime(result.LastOnlineTime),
	}, nil
}

func (s *FriendServiceService) BatchGetUserOnlineStatus(ctx context.Context, req *pb.BatchGetUserOnlineStatusReq) (*pb.BatchGetUserOnlineStatusResp, error) {
	query := &biz.BatchGetUserOnlineStatusQuery{
		UserIDs: cast.ToInt64Slice(req.UserIds),
	}

	result, err := s.uc.BatchGetUserOnlineStatus(ctx, query)
	if err != nil {
		return &pb.BatchGetUserOnlineStatusResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}
	ret := make(map[string]int32)
	for k, v := range result.Statuses {
		ret[cast.ToString(k)] = v
	}

	return &pb.BatchGetUserOnlineStatusResp{
		Meta:         utils.GetSuccessMeta(),
		OnlineStatus: ret,
	}, nil
}

// 辅助函数
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}
