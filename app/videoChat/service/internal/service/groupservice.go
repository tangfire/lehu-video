package service

import (
	"context"

	pb "lehu-video/api/videoChat/service/v1"
	"lehu-video/app/videoChat/service/internal/biz"
	"lehu-video/app/videoChat/service/internal/pkg/utils"
)

type GroupServiceService struct {
	pb.UnimplementedGroupServiceServer

	uc *biz.GroupUsecase
}

func NewGroupServiceService(uc *biz.GroupUsecase) *GroupServiceService {
	return &GroupServiceService{uc: uc}
}

func (s *GroupServiceService) CreateGroup(ctx context.Context, req *pb.CreateGroupReq) (*pb.CreateGroupResp, error) {
	cmd := &biz.CreateGroupCommand{
		OwnerID: req.OwnerId,
		Name:    req.Name,
		Notice:  req.Notice,
		AddMode: req.AddMode,
		Avatar:  req.Avatar,
	}

	result, err := s.uc.CreateGroup(ctx, cmd)
	if err != nil {
		return &pb.CreateGroupResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.CreateGroupResp{
		GroupId: result.GroupID,
		Meta:    utils.GetSuccessMeta(),
	}, nil
}

func (s *GroupServiceService) LoadMyGroup(ctx context.Context, req *pb.LoadMyGroupReq) (*pb.LoadMyGroupResp, error) {
	query := &biz.LoadMyGroupQuery{
		OwnerID: req.OwnerId,
		PageStats: biz.PageStats{
			Page:     req.PageStats.Page,
			PageSize: req.PageStats.Size,
		},
	}

	result, err := s.uc.LoadMyGroup(ctx, query)
	if err != nil {
		return &pb.LoadMyGroupResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	var groups []*pb.GroupInfo
	for _, g := range result.Groups {
		groups = append(groups, &pb.GroupInfo{
			Id:        g.ID,
			Name:      g.Name,
			Notice:    g.Notice,
			Members:   g.Members,
			MemberCnt: int32(g.MemberCnt),
			OwnerId:   g.OwnerID,
			AddMode:   g.AddMode,
			Avatar:    g.Avatar,
			Status:    g.Status,
			CreatedAt: g.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt: g.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &pb.LoadMyGroupResp{
		Groups: groups,
		Meta:   utils.GetSuccessMeta(),
		PageStats: &pb.PageStatsResp{
			Total: int32(result.Total),
		},
	}, nil
}

func (s *GroupServiceService) CheckGroupAddMode(ctx context.Context, req *pb.CheckGroupAddModeReq) (*pb.CheckGroupAddModeResp, error) {
	query := &biz.CheckGroupAddModeQuery{
		GroupID: req.GroupId,
	}

	result, err := s.uc.CheckGroupAddMode(ctx, query)
	if err != nil {
		return &pb.CheckGroupAddModeResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.CheckGroupAddModeResp{
		AddMode: result.AddMode,
		Meta:    utils.GetSuccessMeta(),
	}, nil
}

func (s *GroupServiceService) EnterGroupDirectly(ctx context.Context, req *pb.EnterGroupDirectlyReq) (*pb.EnterGroupDirectlyResp, error) {
	cmd := &biz.EnterGroupDirectlyCommand{
		UserID:  req.UserId,
		GroupID: req.GroupId,
	}

	_, err := s.uc.EnterGroupDirectly(ctx, cmd)
	if err != nil {
		return &pb.EnterGroupDirectlyResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.EnterGroupDirectlyResp{
		Meta: utils.GetSuccessMeta(),
	}, nil
}

func (s *GroupServiceService) ApplyJoinGroup(ctx context.Context, req *pb.ApplyJoinGroupReq) (*pb.ApplyJoinGroupResp, error) {
	cmd := &biz.ApplyJoinGroupCommand{
		UserID:      req.UserId,
		GroupID:     req.GroupId,
		ApplyReason: req.ApplyReason,
	}

	_, err := s.uc.ApplyJoinGroup(ctx, cmd)
	if err != nil {
		return &pb.ApplyJoinGroupResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.ApplyJoinGroupResp{
		Meta: utils.GetSuccessMeta(),
	}, nil
}

func (s *GroupServiceService) HandleJoinApply(ctx context.Context, req *pb.HandleJoinApplyReq) (*pb.HandleJoinApplyResp, error) {
	cmd := &biz.HandleJoinApplyCommand{
		ApplyID:   req.ApplyId,
		HandlerID: req.HandlerId,
		Accept:    req.Accept,
		ReplyMsg:  req.ReplyMsg,
	}

	_, err := s.uc.HandleJoinApply(ctx, cmd)
	if err != nil {
		return &pb.HandleJoinApplyResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.HandleJoinApplyResp{
		Meta: utils.GetSuccessMeta(),
	}, nil
}

func (s *GroupServiceService) LeaveGroup(ctx context.Context, req *pb.LeaveGroupReq) (*pb.LeaveGroupResp, error) {
	cmd := &biz.LeaveGroupCommand{
		UserID:  req.UserId,
		GroupID: req.GroupId,
	}

	_, err := s.uc.LeaveGroup(ctx, cmd)
	if err != nil {
		return &pb.LeaveGroupResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.LeaveGroupResp{
		Meta: utils.GetSuccessMeta(),
	}, nil
}

func (s *GroupServiceService) DismissGroup(ctx context.Context, req *pb.DismissGroupReq) (*pb.DismissGroupResp, error) {
	cmd := &biz.DismissGroupCommand{
		OwnerID: req.OwnerId,
		GroupID: req.GroupId,
	}

	_, err := s.uc.DismissGroup(ctx, cmd)
	if err != nil {
		return &pb.DismissGroupResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.DismissGroupResp{
		Meta: utils.GetSuccessMeta(),
	}, nil
}

func (s *GroupServiceService) GetGroupInfo(ctx context.Context, req *pb.GetGroupInfoReq) (*pb.GetGroupInfoResp, error) {
	query := &biz.GetGroupInfoQuery{
		GroupID: req.GroupId,
	}

	result, err := s.uc.GetGroupInfo(ctx, query)
	if err != nil {
		return &pb.GetGroupInfoResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	if result.Group == nil {
		return &pb.GetGroupInfoResp{
			Meta: utils.GetMetaWithError(nil),
		}, nil
	}

	group := result.Group
	return &pb.GetGroupInfoResp{
		Group: &pb.GroupInfo{
			Id:        group.ID,
			Name:      group.Name,
			Notice:    group.Notice,
			Members:   group.Members,
			MemberCnt: int32(group.MemberCnt),
			OwnerId:   group.OwnerID,
			AddMode:   group.AddMode,
			Avatar:    group.Avatar,
			Status:    group.Status,
			CreatedAt: group.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt: group.UpdatedAt.Format("2006-01-02 15:04:05"),
		},
		Meta: utils.GetSuccessMeta(),
	}, nil
}

func (s *GroupServiceService) UpdateGroupInfo(ctx context.Context, req *pb.UpdateGroupInfoReq) (*pb.UpdateGroupInfoResp, error) {
	cmd := &biz.UpdateGroupInfoCommand{
		GroupID:    req.GroupId,
		OperatorID: req.OperatorId,
		Name:       req.Name,
		Notice:     req.Notice,
		AddMode:    req.AddMode,
		Avatar:     req.Avatar,
	}

	_, err := s.uc.UpdateGroupInfo(ctx, cmd)
	if err != nil {
		return &pb.UpdateGroupInfoResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.UpdateGroupInfoResp{
		Meta: utils.GetSuccessMeta(),
	}, nil
}

func (s *GroupServiceService) ListGroupMembers(ctx context.Context, req *pb.ListGroupMembersReq) (*pb.ListGroupMembersResp, error) {
	query := &biz.ListGroupMembersQuery{
		GroupID: req.GroupId,
		PageStats: biz.PageStats{
			Page:     req.PageStats.Page,
			PageSize: req.PageStats.Size,
		},
	}

	result, err := s.uc.ListGroupMembers(ctx, query)
	if err != nil {
		return &pb.ListGroupMembersResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	var members []*pb.GroupMember
	for _, m := range result.Members {
		members = append(members, &pb.GroupMember{
			UserId:   m.UserID,
			GroupId:  m.GroupID,
			Role:     m.Role,
			JoinTime: m.JoinTime.Format("2006-01-02 15:04:05"),
		})
	}

	return &pb.ListGroupMembersResp{
		Members: members,
		Meta:    utils.GetSuccessMeta(),
		PageStats: &pb.PageStatsResp{
			Total: int32(result.Total),
		},
	}, nil
}

func (s *GroupServiceService) RemoveGroupMember(ctx context.Context, req *pb.RemoveGroupMemberReq) (*pb.RemoveGroupMemberResp, error) {
	cmd := &biz.RemoveGroupMemberCommand{
		GroupID:      req.GroupId,
		OperatorID:   req.OperatorId,
		TargetUserID: req.TargetUserId,
	}

	_, err := s.uc.RemoveGroupMember(ctx, cmd)
	if err != nil {
		return &pb.RemoveGroupMemberResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.RemoveGroupMemberResp{
		Meta: utils.GetSuccessMeta(),
	}, nil
}

func (s *GroupServiceService) TransferGroupOwner(ctx context.Context, req *pb.TransferGroupOwnerReq) (*pb.TransferGroupOwnerResp, error) {
	cmd := &biz.TransferGroupOwnerCommand{
		GroupID:    req.GroupId,
		FromUserID: req.FromUserId,
		ToUserID:   req.ToUserId,
	}

	_, err := s.uc.TransferGroupOwner(ctx, cmd)
	if err != nil {
		return &pb.TransferGroupOwnerResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.TransferGroupOwnerResp{
		Meta: utils.GetSuccessMeta(),
	}, nil
}

func (s *GroupServiceService) SetGroupAdmin(ctx context.Context, req *pb.SetGroupAdminReq) (*pb.SetGroupAdminResp, error) {
	cmd := &biz.SetGroupAdminCommand{
		GroupID:      req.GroupId,
		OperatorID:   req.OperatorId,
		TargetUserID: req.TargetUserId,
		SetAsAdmin:   req.SetAsAdmin,
	}

	_, err := s.uc.SetGroupAdmin(ctx, cmd)
	if err != nil {
		return &pb.SetGroupAdminResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.SetGroupAdminResp{
		Meta: utils.GetSuccessMeta(),
	}, nil
}

func (s *GroupServiceService) ListMyJoinedGroups(ctx context.Context, req *pb.ListMyJoinedGroupsReq) (*pb.ListMyJoinedGroupsResp, error) {
	query := &biz.ListMyJoinedGroupsQuery{
		UserID: req.UserId,
		PageStats: biz.PageStats{
			Page:     req.PageStats.Page,
			PageSize: req.PageStats.Size,
		},
	}

	result, err := s.uc.ListMyJoinedGroups(ctx, query)
	if err != nil {
		return &pb.ListMyJoinedGroupsResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	var groups []*pb.GroupInfo
	for _, g := range result.Groups {
		groups = append(groups, &pb.GroupInfo{
			Id:        g.ID,
			Name:      g.Name,
			Notice:    g.Notice,
			Members:   g.Members,
			MemberCnt: int32(g.MemberCnt),
			OwnerId:   g.OwnerID,
			AddMode:   g.AddMode,
			Avatar:    g.Avatar,
			Status:    g.Status,
			CreatedAt: g.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt: g.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &pb.ListMyJoinedGroupsResp{
		Groups: groups,
		Meta:   utils.GetSuccessMeta(),
		PageStats: &pb.PageStatsResp{
			Total: int32(result.Total),
		},
	}, nil
}
