package service

import (
	"context"

	pb "lehu-video/api/videoCore/service/v1"
)

type FavoriteServiceService struct {
	pb.UnimplementedFavoriteServiceServer
}

func NewFavoriteServiceService() *FavoriteServiceService {
	return &FavoriteServiceService{}
}

func (s *FavoriteServiceService) AddFavorite(ctx context.Context, req *pb.AddFavoriteReq) (*pb.AddFavoriteResp, error) {
	return &pb.AddFavoriteResp{}, nil
}
func (s *FavoriteServiceService) RemoveFavorite(ctx context.Context, req *pb.RemoveFavoriteReq) (*pb.RemoveFavoriteResp, error) {
	return &pb.RemoveFavoriteResp{}, nil
}
func (s *FavoriteServiceService) ListFavorite(ctx context.Context, req *pb.ListFavoriteReq) (*pb.ListFavoriteResp, error) {
	return &pb.ListFavoriteResp{}, nil
}
func (s *FavoriteServiceService) CountFavorite(ctx context.Context, req *pb.CountFavoriteReq) (*pb.CountFavoriteResp, error) {
	return &pb.CountFavoriteResp{}, nil
}
func (s *FavoriteServiceService) IsFavorite(ctx context.Context, req *pb.IsFavoriteReq) (*pb.IsFavoriteResp, error) {
	return &pb.IsFavoriteResp{}, nil
}
