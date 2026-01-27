package service

import (
	"context"

	pb "lehu-video/api/videoApi/service/v1"
	"lehu-video/app/videoApi/service/internal/biz"
)

type CommentServiceService struct {
	pb.UnimplementedCommentServiceServer
	uc *biz.CommentUsecase
}

func NewCommentServiceService(uc *biz.CommentUsecase) *CommentServiceService {
	return &CommentServiceService{uc: uc}
}

func (s *CommentServiceService) CreateComment(ctx context.Context, req *pb.CreateCommentReq) (*pb.CreateCommentResp, error) {
	input := &biz.CreateCommentInput{
		VideoId:     req.VideoId,
		Content:     req.Content,
		ParentId:    req.ParentId,
		ReplyUserId: req.ReplyUserId,
	}

	output, err := s.uc.CreateComment(ctx, input)
	if err != nil {
		return nil, err
	}

	pbComment := s.toPbComment(output.Comment)
	return &pb.CreateCommentResp{Comment: pbComment}, nil
}

func (s *CommentServiceService) RemoveComment(ctx context.Context, req *pb.RemoveCommentReq) (*pb.RemoveCommentResp, error) {
	input := &biz.RemoveCommentInput{
		Id: req.Id,
	}

	err := s.uc.RemoveComment(ctx, input)
	if err != nil {
		return nil, err
	}

	return &pb.RemoveCommentResp{}, nil
}

func (s *CommentServiceService) ListComment4Video(ctx context.Context, req *pb.ListComment4VideoReq) (*pb.ListComment4VideoResp, error) {
	pageStats := &biz.PageStats{
		Page:     req.Pagination.Page,
		PageSize: req.Pagination.Size,
	}

	input := &biz.ListComment4VideoInput{
		VideoId:   req.VideoId,
		PageStats: pageStats,
	}

	output, err := s.uc.ListComment4Video(ctx, input)
	if err != nil {
		return nil, err
	}

	pbComments := make([]*pb.Comment, 0, len(output.Comments))
	for _, comment := range output.Comments {
		pbComments = append(pbComments, s.toPbComment(comment))
	}

	return &pb.ListComment4VideoResp{
		Comments: pbComments,
		Pagination: &pb.PageStatsResp{
			Total: int32(output.Total),
		},
	}, nil
}

func (s *CommentServiceService) ListChildComment(ctx context.Context, req *pb.ListChildCommentReq) (*pb.ListChildCommentResp, error) {
	pageStats := &biz.PageStats{
		Page:     req.Pagination.Page,
		PageSize: req.Pagination.Size,
	}

	input := &biz.ListChildCommentInput{
		CommentId: req.CommentId,
		PageStats: pageStats,
	}

	output, err := s.uc.ListChildComment(ctx, input)
	if err != nil {
		return nil, err
	}

	pbComments := make([]*pb.Comment, 0, len(output.Comments))
	for _, comment := range output.Comments {
		pbComments = append(pbComments, s.toPbComment(comment))
	}

	return &pb.ListChildCommentResp{
		Comments: pbComments,
		Pagination: &pb.PageStatsResp{
			Total: int32(output.Total),
		},
	}, nil
}

func (s *CommentServiceService) toPbComment(comment *biz.Comment) *pb.Comment {
	if comment == nil {
		return nil
	}

	// 转换子评论
	var pbComments []*pb.Comment
	if len(comment.Comments) > 0 {
		pbComments = make([]*pb.Comment, 0, len(comment.Comments))
		for _, child := range comment.Comments {
			pbComments = append(pbComments, s.toPbComment(child))
		}
	}

	// 转换用户信息
	var pbUser *pb.CommentUser
	if comment.User != nil {
		pbUser = &pb.CommentUser{
			Id:          comment.User.Id,
			Name:        comment.User.Name,
			Avatar:      comment.User.Avatar,
			IsFollowing: comment.User.IsFollowing,
		}
	}

	// 转换回复用户信息
	var pbReplyUser *pb.CommentUser
	if comment.ReplyUser != nil {
		pbReplyUser = &pb.CommentUser{
			Id:          comment.ReplyUser.Id,
			Name:        comment.ReplyUser.Name,
			Avatar:      comment.ReplyUser.Avatar,
			IsFollowing: comment.ReplyUser.IsFollowing,
		}
	}

	return &pb.Comment{
		Id:         comment.Id,
		VideoId:    comment.VideoId,
		ParentId:   comment.ParentId,
		User:       pbUser,
		ReplyUser:  pbReplyUser,
		Content:    comment.Content,
		Date:       comment.Date,
		LikeCount:  comment.LikeCount,  // 直接使用int64，无需转换
		ReplyCount: comment.ReplyCount, // 直接使用int64，无需转换
		Comments:   pbComments,
	}
}
