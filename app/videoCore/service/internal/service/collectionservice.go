package service

import (
	"context"

	pb "lehu-video/api/videoCore/service/v1"
)

type CollectionServiceService struct {
	pb.UnimplementedCollectionServiceServer
}

func NewCollectionServiceService() *CollectionServiceService {
	return &CollectionServiceService{}
}

func (s *CollectionServiceService) CreateCollection(ctx context.Context, req *pb.CreateCollectionReq) (*pb.CreateCollectionResp, error) {
	return &pb.CreateCollectionResp{}, nil
}
func (s *CollectionServiceService) UpdateCollection(ctx context.Context, req *pb.UpdateCollectionReq) (*pb.UpdateCollectionResp, error) {
	return &pb.UpdateCollectionResp{}, nil
}
func (s *CollectionServiceService) RemoveCollection(ctx context.Context, req *pb.RemoveCollectionReq) (*pb.RemoveCollectionResp, error) {
	return &pb.RemoveCollectionResp{}, nil
}
func (s *CollectionServiceService) GetCollectionById(ctx context.Context, req *pb.GetCollectionByIdReq) (*pb.GetCollectionByIdResp, error) {
	return &pb.GetCollectionByIdResp{}, nil
}
func (s *CollectionServiceService) ListCollection(ctx context.Context, req *pb.ListCollectionReq) (*pb.ListCollectionResp, error) {
	return &pb.ListCollectionResp{}, nil
}
func (s *CollectionServiceService) AddVideo2Collection(ctx context.Context, req *pb.AddVideo2CollectionReq) (*pb.AddVideo2CollectionResp, error) {
	return &pb.AddVideo2CollectionResp{}, nil
}
func (s *CollectionServiceService) RemoveVideoFromCollection(ctx context.Context, req *pb.RemoveVideoFromCollectionReq) (*pb.RemoveVideoFromCollectionResp, error) {
	return &pb.RemoveVideoFromCollectionResp{}, nil
}
func (s *CollectionServiceService) ListVideo2Collection(ctx context.Context, req *pb.ListVideoFromCollectionReq) (*pb.ListVideoFromCollectionResp, error) {
	return &pb.ListVideoFromCollectionResp{}, nil
}
func (s *CollectionServiceService) IsCollected(ctx context.Context, req *pb.IsCollectedReq) (*pb.IsCollectedResp, error) {
	return &pb.IsCollectedResp{}, nil
}
func (s *CollectionServiceService) CountCollect4Video(ctx context.Context, req *pb.CountCollect4VideoReq) (*pb.CountCollect4VideoResp, error) {
	return &pb.CountCollect4VideoResp{}, nil
}
