package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	pb "lehu-video/api/videoApi/service/v1"
	"lehu-video/app/videoApi/service/internal/biz"
)

type GroupServiceService struct {
	pb.UnimplementedGroupServiceServer

	log *log.Helper
	uc  *biz.GroupUsecase
}

func NewGroupServiceService(uc *biz.GroupUsecase, logger log.Logger) *GroupServiceService {
	return &GroupServiceService{
		uc:  uc,
		log: log.NewHelper(logger),
	}
}

func (s *GroupServiceService) CreateGroup(ctx context.Context, req *pb.CreateGroupReq) (*pb.CreateGroupResp, error) {
	input := &biz.CreateGroupInput{
		Name:    req.Name,
		Notice:  req.Notice,
		AddMode: req.AddMode,
		Avatar:  req.Avatar,
	}

	groupID, err := s.uc.CreateGroup(ctx, input)
	if err != nil {
		return nil, err
	}

	return &pb.CreateGroupResp{
		GroupId: groupID,
	}, nil
}

func (s *GroupServiceService) LoadMyGroup(ctx context.Context, req *pb.LoadMyGroupReq) (*pb.LoadMyGroupResp, error) {
	input := &biz.LoadMyGroupInput{
		PageStats: &biz.PageStats{
			Page:     req.PageStats.Page,
			PageSize: req.PageStats.Size,
		},
	}

	output, err := s.uc.LoadMyGroup(ctx, input)
	if err != nil {
		return nil, err
	}

	var groups []*pb.GroupInfo
	for _, g := range output.Groups {
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
			CreatedAt: g.CreatedAt,
			UpdatedAt: g.UpdatedAt,
		})
	}

	return &pb.LoadMyGroupResp{
		Groups: groups,
		PageStats: &pb.PageStatsResp{
			Total: int32(output.Total),
		},
	}, nil
}

func (s *GroupServiceService) CheckGroupAddMode(ctx context.Context, req *pb.CheckGroupAddModeReq) (*pb.CheckGroupAddModeResp, error) {
	input := &biz.CheckGroupAddModeInput{
		GroupID: req.GroupId,
	}

	output, err := s.uc.CheckGroupAddMode(ctx, input)
	if err != nil {
		return nil, err
	}

	return &pb.CheckGroupAddModeResp{
		AddMode: output.AddMode,
	}, nil
}

func (s *GroupServiceService) EnterGroupDirectly(ctx context.Context, req *pb.EnterGroupDirectlyReq) (*pb.EnterGroupDirectlyResp, error) {
	input := &biz.EnterGroupDirectlyInput{
		GroupID: req.GroupId,
	}

	err := s.uc.EnterGroupDirectly(ctx, input)
	if err != nil {
		return nil, err
	}

	return &pb.EnterGroupDirectlyResp{}, nil
}

func (s *GroupServiceService) ApplyJoinGroup(ctx context.Context, req *pb.ApplyJoinGroupReq) (*pb.ApplyJoinGroupResp, error) {
	input := &biz.ApplyJoinGroupInput{
		GroupID:     req.GroupId,
		ApplyReason: req.ApplyReason,
	}

	err := s.uc.ApplyJoinGroup(ctx, input)
	if err != nil {
		return nil, err
	}

	return &pb.ApplyJoinGroupResp{}, nil
}

func (s *GroupServiceService) LeaveGroup(ctx context.Context, req *pb.LeaveGroupReq) (*pb.LeaveGroupResp, error) {
	input := &biz.LeaveGroupInput{
		GroupID: req.GroupId,
	}

	err := s.uc.LeaveGroup(ctx, input)
	if err != nil {
		return nil, err
	}

	return &pb.LeaveGroupResp{}, nil
}

func (s *GroupServiceService) DismissGroup(ctx context.Context, req *pb.DismissGroupReq) (*pb.DismissGroupResp, error) {
	input := &biz.DismissGroupInput{
		GroupID: req.GroupId,
	}

	err := s.uc.DismissGroup(ctx, input)
	if err != nil {
		return nil, err
	}

	return &pb.DismissGroupResp{}, nil
}

func (s *GroupServiceService) GetGroupInfo(ctx context.Context, req *pb.GetGroupInfoReq) (*pb.GetGroupInfoResp, error) {
	input := &biz.GetGroupInfoInput{
		GroupID: req.GroupId,
	}

	output, err := s.uc.GetGroupInfo(ctx, input)
	if err != nil {
		return nil, err
	}

	group := output.Group
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
			CreatedAt: group.CreatedAt,
			UpdatedAt: group.UpdatedAt,
		},
	}, nil
}

func (s *GroupServiceService) ListMyJoinedGroups(ctx context.Context, req *pb.ListMyJoinedGroupsReq) (*pb.ListMyJoinedGroupsResp, error) {
	input := &biz.ListMyJoinedGroupsInput{
		PageStats: &biz.PageStats{
			Page:     req.PageStats.Page,
			PageSize: req.PageStats.Size,
		},
	}

	output, err := s.uc.ListMyJoinedGroups(ctx, input)
	if err != nil {
		return nil, err
	}

	var groups []*pb.GroupInfo
	for _, g := range output.Groups {
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
			CreatedAt: g.CreatedAt,
			UpdatedAt: g.UpdatedAt,
		})
	}

	return &pb.ListMyJoinedGroupsResp{
		Groups: groups,
		PageStats: &pb.PageStatsResp{
			Total: int32(output.Total),
		},
	}, nil
}
