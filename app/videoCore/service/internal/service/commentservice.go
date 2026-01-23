package service

import (
	"context"
	"lehu-video/app/videoCore/service/internal/biz"

	pb "lehu-video/api/videoCore/service/v1"
)

type CommentServiceService struct {
	pb.UnimplementedCommentServiceServer

	uc *biz.CommentUsecase
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
func (s *CommentServiceService) ListChildComment4Comment(ctx context.Context, req *pb.ListChildComment4CommentReq) (*pb.ListChildComment4CommentResp, error) {
	return &pb.ListChildComment4CommentResp{}, nil
}
func (s *CommentServiceService) GetCommentById(ctx context.Context, req *pb.GetCommentByIdReq) (*pb.GetCommentByIdResp, error) {
	return &pb.GetCommentByIdResp{}, nil
}
func (s *CommentServiceService) CountComment4Video(ctx context.Context, req *pb.CountComment4VideoReq) (*pb.CountComment4VideoResp, error) {
	return &pb.CountComment4VideoResp{}, nil
}
func (s *CommentServiceService) CountComment4User(ctx context.Context, req *pb.CountComment4UserReq) (*pb.CountComment4UserResp, error) {
	return &pb.CountComment4UserResp{}, nil
}
