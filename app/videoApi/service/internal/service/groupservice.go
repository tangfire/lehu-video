package service

import (
	"context"
	"github.com/go-kratos/kratos/v2/log"
	v1 "lehu-video/api/videoApi/service/v1"
	"lehu-video/app/videoApi/service/internal/biz"
)

type GroupServiceService struct {
	v1.UnimplementedGroupServiceServer
	uc  *biz.GroupUsecase
	log *log.Helper
}

func NewGroupServiceService(uc *biz.GroupUsecase, logger log.Logger) *GroupServiceService {
	return &GroupServiceService{
		uc:  uc,
		log: log.NewHelper(logger),
	}
}

func (s *GroupServiceService) CreateGroup(ctx context.Context, req *v1.CreateGroupReq) (*v1.CreateGroupResp, error) {
	input := &biz.CreateGroupInput{
		Name:    req.Name,
		Notice:  req.Notice,
		AddMode: req.AddMode,
		Avatar:  req.Avatar,
	}

	groupId, err := s.uc.CreateGroup(ctx, input)
	if err != nil {
		return &v1.CreateGroupResp{}, err
	}

	return &v1.CreateGroupResp{
		GroupId: groupId,
	}, nil
}

func (s *GroupServiceService) LoadMyGroup(ctx context.Context, req *v1.LoadMyGroupReq) (*v1.LoadMyGroupResp, error) {
	input := &biz.LoadMyGroupInput{
		Page:     req.PageStats.Page,
		PageSize: req.PageStats.Size,
	}

	output, err := s.uc.LoadMyGroup(ctx, input)
	if err != nil {
		return &v1.LoadMyGroupResp{}, err
	}

	// 转换群聊信息
	var groups []*v1.GroupInfo
	for _, group := range output.Groups {
		groups = append(groups, &v1.GroupInfo{
			Id:        group.ID,
			Name:      group.Name,
			Notice:    group.Notice,
			MemberCnt: int32(group.MemberCnt),
			OwnerId:   group.OwnerID,
			AddMode:   group.AddMode,
			Avatar:    group.Avatar,
			Status:    group.Status,
			CreatedAt: group.CreatedAt,
			UpdatedAt: group.UpdatedAt,
		})
	}

	return &v1.LoadMyGroupResp{
		Groups: groups,
		PageStats: &v1.PageStatsResp{
			Total: int32(output.Total),
		},
	}, nil
}

func (s *GroupServiceService) CheckGroupAddMode(ctx context.Context, req *v1.CheckGroupAddModeReq) (*v1.CheckGroupAddModeResp, error) {
	input := &biz.CheckGroupAddModeInput{
		GroupID: req.GroupId,
	}

	output, err := s.uc.CheckGroupAddMode(ctx, input)
	if err != nil {
		return &v1.CheckGroupAddModeResp{}, err
	}

	return &v1.CheckGroupAddModeResp{
		AddMode: output.AddMode,
	}, nil
}

func (s *GroupServiceService) EnterGroupDirectly(ctx context.Context, req *v1.EnterGroupDirectlyReq) (*v1.EnterGroupDirectlyResp, error) {
	input := &biz.EnterGroupDirectlyInput{
		GroupID: req.GroupId,
	}

	err := s.uc.EnterGroupDirectly(ctx, input)
	if err != nil {
		return &v1.EnterGroupDirectlyResp{}, err
	}

	return &v1.EnterGroupDirectlyResp{}, nil
}

func (s *GroupServiceService) ApplyJoinGroup(ctx context.Context, req *v1.ApplyJoinGroupReq) (*v1.ApplyJoinGroupResp, error) {
	input := &biz.ApplyJoinGroupInput{
		GroupID:     req.GroupId,
		ApplyReason: req.ApplyReason,
	}

	err := s.uc.ApplyJoinGroup(ctx, input)
	if err != nil {
		return &v1.ApplyJoinGroupResp{}, err
	}

	return &v1.ApplyJoinGroupResp{}, nil
}

func (s *GroupServiceService) LeaveGroup(ctx context.Context, req *v1.LeaveGroupReq) (*v1.LeaveGroupResp, error) {
	input := &biz.LeaveGroupInput{
		GroupID: req.GroupId,
	}

	err := s.uc.LeaveGroup(ctx, input)
	if err != nil {
		return &v1.LeaveGroupResp{}, err
	}

	return &v1.LeaveGroupResp{}, nil
}

func (s *GroupServiceService) DismissGroup(ctx context.Context, req *v1.DismissGroupReq) (*v1.DismissGroupResp, error) {
	input := &biz.DismissGroupInput{
		GroupID: req.GroupId,
	}

	err := s.uc.DismissGroup(ctx, input)
	if err != nil {
		return &v1.DismissGroupResp{}, err
	}

	return &v1.DismissGroupResp{}, nil
}

func (s *GroupServiceService) GetGroupInfo(ctx context.Context, req *v1.GetGroupInfoReq) (*v1.GetGroupInfoResp, error) {
	input := &biz.GetGroupInfoInput{
		GroupID: req.GroupId,
	}

	output, err := s.uc.GetGroupInfo(ctx, input)
	if err != nil {
		return &v1.GetGroupInfoResp{}, err
	}

	if output.Group == nil {
		return &v1.GetGroupInfoResp{}, nil
	}

	group := output.Group
	return &v1.GetGroupInfoResp{
		Group: &v1.GroupInfo{
			Id:        group.ID,
			Name:      group.Name,
			Notice:    group.Notice,
			MemberCnt: int32(group.MemberCnt),
			OwnerId:   group.OwnerID,
			AddMode:   group.AddMode,
			Avatar:    group.Avatar,
			Status:    group.Status,
			CreatedAt: group.CreatedAt,
			UpdatedAt: group.UpdatedAt,
		},
	}, nil
}

func (s *GroupServiceService) ListMyJoinedGroups(ctx context.Context, req *v1.ListMyJoinedGroupsReq) (*v1.ListMyJoinedGroupsResp, error) {
	input := &biz.ListMyJoinedGroupsInput{
		Page:     req.PageStats.Page,
		PageSize: req.PageStats.Size,
	}

	output, err := s.uc.ListMyJoinedGroups(ctx, input)
	if err != nil {
		return &v1.ListMyJoinedGroupsResp{}, err
	}

	var groups []*v1.GroupInfo
	for _, group := range output.Groups {
		groups = append(groups, &v1.GroupInfo{
			Id:        group.ID,
			Name:      group.Name,
			Notice:    group.Notice,
			MemberCnt: int32(group.MemberCnt),
			OwnerId:   group.OwnerID,
			AddMode:   group.AddMode,
			Avatar:    group.Avatar,
			Status:    group.Status,
			CreatedAt: group.CreatedAt,
			UpdatedAt: group.UpdatedAt,
		})
	}

	return &v1.ListMyJoinedGroupsResp{
		Groups: groups,
		PageStats: &v1.PageStatsResp{
			Total: int32(output.Total),
		},
	}, nil
}

func (s *GroupServiceService) HandleGroupApply(ctx context.Context, req *v1.HandleGroupApplyReq) (*v1.HandleGroupApplyResp, error) {
	input := &biz.HandleGroupApplyInput{
		ApplyID:  req.ApplyId,
		Accept:   req.Accept,
		ReplyMsg: req.ReplyMsg,
	}
	err := s.uc.HandleGroupApply(ctx, input)
	if err != nil {
		return nil, err
	}
	return &v1.HandleGroupApplyResp{}, nil
}

func (s *GroupServiceService) GetGroupMembers(ctx context.Context, req *v1.GetGroupMembersReq) (*v1.GetGroupMembersResp, error) {
	input := &biz.GetGroupMembersInput{
		GroupID: req.GroupId,
	}
	output, err := s.uc.GetGroupMembers(ctx, input)
	if err != nil {
		return nil, err
	}
	return &v1.GetGroupMembersResp{
		MemberIds: output.MemberIDs,
	}, nil
}
