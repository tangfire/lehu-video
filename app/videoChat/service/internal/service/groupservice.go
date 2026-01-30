package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	pb "lehu-video/api/videoChat/service/v1"
	"lehu-video/app/videoChat/service/internal/biz"
	"lehu-video/app/videoChat/service/internal/pkg/utils"
)

type GroupServiceService struct {
	pb.UnimplementedGroupServiceServer
	uc  *biz.GroupUsecase
	log *log.Helper
}

func NewGroupServiceService(uc *biz.GroupUsecase, logger log.Logger) *GroupServiceService {
	return &GroupServiceService{
		uc:  uc,
		log: log.NewHelper(logger),
	}
}

func (s *GroupServiceService) CreateGroup(ctx context.Context, req *pb.CreateGroupReq) (*pb.CreateGroupResp, error) {
	// ✅ 构建Command
	cmd := &biz.CreateGroupCommand{
		OwnerID: req.OwnerId,
		Name:    req.Name,
		Notice:  req.Notice,
		AddMode: req.AddMode,
		Avatar:  req.Avatar,
	}

	// ✅ 调用业务层
	result, err := s.uc.CreateGroup(ctx, cmd)
	if err != nil {
		return &pb.CreateGroupResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.CreateGroupResp{
		Meta:    utils.GetSuccessMeta(),
		GroupId: result.GroupID,
	}, nil
}

func (s *GroupServiceService) LoadMyGroup(ctx context.Context, req *pb.LoadMyGroupReq) (*pb.LoadMyGroupResp, error) {
	// ✅ 构建Query
	query := &biz.LoadMyGroupQuery{
		OwnerID: req.OwnerId,
		PageStats: biz.PageStats{
			Page:     int(req.PageStats.Page),
			PageSize: int(req.PageStats.Size),
		},
	}

	// ✅ 调用业务层
	result, err := s.uc.LoadMyGroup(ctx, query)
	if err != nil {
		return &pb.LoadMyGroupResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	// ✅ 转换为proto结构
	groups := make([]*pb.GroupInfo, 0, len(result.Groups))
	for _, group := range result.Groups {
		groups = append(groups, &pb.GroupInfo{
			Id:        group.ID,
			Name:      group.Name,
			Notice:    group.Notice,
			MemberCnt: int32(group.MemberCnt),
			OwnerId:   group.OwnerID,
			AddMode:   group.AddMode,
			Avatar:    group.Avatar,
			Status:    group.Status,
			CreatedAt: group.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt: group.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &pb.LoadMyGroupResp{
		Meta:   utils.GetSuccessMeta(),
		Groups: groups,
		PageStats: &pb.PageStatsResp{
			Total: int32(result.Total),
		},
	}, nil
}

func (s *GroupServiceService) CheckGroupAddMode(ctx context.Context, req *pb.CheckGroupAddModeReq) (*pb.CheckGroupAddModeResp, error) {
	// ✅ 构建Query
	query := &biz.CheckGroupAddModeQuery{
		GroupID: req.GroupId,
	}

	// ✅ 调用业务层
	result, err := s.uc.CheckGroupAddMode(ctx, query)
	if err != nil {
		return &pb.CheckGroupAddModeResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.CheckGroupAddModeResp{
		Meta:    utils.GetSuccessMeta(),
		AddMode: result.AddMode,
	}, nil
}

func (s *GroupServiceService) EnterGroupDirectly(ctx context.Context, req *pb.EnterGroupDirectlyReq) (*pb.EnterGroupDirectlyResp, error) {
	// ✅ 构建Command
	cmd := &biz.EnterGroupDirectlyCommand{
		UserID:  req.UserId,
		GroupID: req.GroupId,
	}

	// ✅ 调用业务层
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
	// ✅ 构建Command
	cmd := &biz.ApplyJoinGroupCommand{
		UserID:      req.UserId,
		GroupID:     req.GroupId,
		ApplyReason: req.ApplyReason,
	}

	// ✅ 调用业务层
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

func (s *GroupServiceService) LeaveGroup(ctx context.Context, req *pb.LeaveGroupReq) (*pb.LeaveGroupResp, error) {
	// ✅ 构建Command
	cmd := &biz.LeaveGroupCommand{
		UserID:  req.UserId,
		GroupID: req.GroupId,
	}

	// ✅ 调用业务层
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
	// ✅ 构建Command
	cmd := &biz.DismissGroupCommand{
		OwnerID: req.OwnerId,
		GroupID: req.GroupId,
	}

	// ✅ 调用业务层
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
	// ✅ 构建Query
	query := &biz.GetGroupInfoQuery{
		GroupID: req.GroupId,
	}

	// ✅ 调用业务层
	result, err := s.uc.GetGroupInfo(ctx, query)
	if err != nil {
		return &pb.GetGroupInfoResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	if result.Group == nil {
		return &pb.GetGroupInfoResp{
			Meta: &pb.Metadata{
				Code:    404,
				Message: "群聊不存在",
			},
		}, nil
	}

	return &pb.GetGroupInfoResp{
		Meta: utils.GetSuccessMeta(),
		Group: &pb.GroupInfo{
			Id:        result.Group.ID,
			Name:      result.Group.Name,
			Notice:    result.Group.Notice,
			MemberCnt: int32(result.Group.MemberCnt),
			OwnerId:   result.Group.OwnerID,
			AddMode:   result.Group.AddMode,
			Avatar:    result.Group.Avatar,
			Status:    result.Group.Status,
			CreatedAt: result.Group.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt: result.Group.UpdatedAt.Format("2006-01-02 15:04:05"),
		},
	}, nil
}

func (s *GroupServiceService) ListMyJoinedGroups(ctx context.Context, req *pb.ListMyJoinedGroupsReq) (*pb.ListMyJoinedGroupsResp, error) {
	// ✅ 构建Query
	query := &biz.ListMyJoinedGroupsQuery{
		UserID: req.UserId,
		PageStats: biz.PageStats{
			Page:     int(req.PageStats.Page),
			PageSize: int(req.PageStats.Size),
		},
	}

	// ✅ 调用业务层
	result, err := s.uc.ListMyJoinedGroups(ctx, query)
	if err != nil {
		return &pb.ListMyJoinedGroupsResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	// ✅ 转换为proto结构
	groups := make([]*pb.GroupInfo, 0, len(result.Groups))
	for _, group := range result.Groups {
		groups = append(groups, &pb.GroupInfo{
			Id:        group.ID,
			Name:      group.Name,
			Notice:    group.Notice,
			MemberCnt: int32(group.MemberCnt),
			OwnerId:   group.OwnerID,
			AddMode:   group.AddMode,
			Avatar:    group.Avatar,
			Status:    group.Status,
			CreatedAt: group.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt: group.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &pb.ListMyJoinedGroupsResp{
		Meta:   utils.GetSuccessMeta(),
		Groups: groups,
		PageStats: &pb.PageStatsResp{
			Total: int32(result.Total),
		},
	}, nil
}

// 新增：获取群成员列表
func (s *GroupServiceService) GetGroupMembers(ctx context.Context, req *pb.GetGroupMembersReq) (*pb.GetGroupMembersResp, error) {
	// ✅ 构建Query
	query := &biz.GetGroupMembersQuery{
		GroupID: req.GroupId,
	}

	// ✅ 调用业务层
	result, err := s.uc.GetGroupMembers(ctx, query)
	if err != nil {
		return &pb.GetGroupMembersResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.GetGroupMembersResp{
		Meta:      utils.GetSuccessMeta(),
		MemberIds: result.MemberIDs,
	}, nil
}

// 新增：检查是否为群成员
func (s *GroupServiceService) IsGroupMember(ctx context.Context, req *pb.IsGroupMemberReq) (*pb.IsGroupMemberResp, error) {
	// ✅ 构建Query
	query := &biz.IsGroupMemberQuery{
		GroupID: req.GroupId,
		UserID:  req.UserId,
	}

	// ✅ 调用业务层
	result, err := s.uc.IsGroupMember(ctx, query)
	if err != nil {
		return &pb.IsGroupMemberResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.IsGroupMemberResp{
		Meta:     utils.GetSuccessMeta(),
		IsMember: result.IsMember,
	}, nil
}
