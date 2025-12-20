package service

import (
	"context"
	"lehu-video/app/videoCore/service/internal/biz"

	pb "lehu-video/api/videoCore/service/v1"
)

type FavoriteServiceService struct {
	pb.UnimplementedFavoriteServiceServer
	uc *biz.FavoriteUsecase
}

func NewFavoriteServiceService(uc *biz.FavoriteUsecase) *FavoriteServiceService {
	return &FavoriteServiceService{uc: uc}
}

func (s *FavoriteServiceService) AddFavorite(ctx context.Context, req *pb.AddFavoriteReq) (*pb.AddFavoriteResp, error) {
	return s.uc.AddFavorite(ctx, req)
}
func (s *FavoriteServiceService) RemoveFavorite(ctx context.Context, req *pb.RemoveFavoriteReq) (*pb.RemoveFavoriteResp, error) {
	return s.uc.RemoveFavorite(ctx, req)
}
func (s *FavoriteServiceService) ListFavorite(ctx context.Context, req *pb.ListFavoriteReq) (*pb.ListFavoriteResp, error) {
	return s.uc.ListFavorite(ctx, req)
}
func (s *FavoriteServiceService) CountFavorite(ctx context.Context, req *pb.CountFavoriteReq) (*pb.CountFavoriteResp, error) {
	return s.uc.CountFavorite(ctx, req)
}
func (s *FavoriteServiceService) IsFavorite(ctx context.Context, req *pb.IsFavoriteReq) (*pb.IsFavoriteResp, error) {
	return s.uc.IsFavorite(ctx, req)
}
