package service

import (
	"context"

	pb "lehu-video/api/videoApi/service/v1"
)

type CommentServiceService struct {
	pb.UnimplementedCommentServiceServer
}

func NewCommentServiceService() *CommentServiceService {
	return &CommentServiceService{}
}

func (s *CommentServiceService) CreateComment(ctx context.Context, req *pb.CreateCommentReq) (*pb.CreateCommentResp, error) {
	return &pb.CreateCommentResp{}, nil
}
func (s *CommentServiceService) RemoveComment(ctx context.Context, req *pb.RemoveCommentReq) (*pb.RemoveCommentResp, error) {
	return &pb.RemoveCommentResp{}, nil
}
func (s *CommentServiceService) ListComment4Video(ctx context.Context, req *pb.ListComment4VideoReq) (*pb.ListComment4VideoResp, error) {
	return &pb.ListComment4VideoResp{}, nil
}
func (s *CommentServiceService) ListChildComment(ctx context.Context, req *pb.ListChildCommentReq) (*pb.ListChildCommentResp, error) {
	return &pb.ListChildCommentResp{}, nil
}
