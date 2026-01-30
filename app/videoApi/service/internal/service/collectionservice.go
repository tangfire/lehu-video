package service

import (
	"context"
	"github.com/go-kratos/kratos/v2/log"
	"lehu-video/app/videoApi/service/internal/biz"

	pb "lehu-video/api/videoApi/service/v1"
)

type CollectionServiceService struct {
	pb.UnimplementedCollectionServiceServer

	log *log.Helper

	uc *biz.CollectionUsecase
}

func NewCollectionServiceService(uc *biz.CollectionUsecase, logger log.Logger) *CollectionServiceService {
	return &CollectionServiceService{uc: uc, log: log.NewHelper(logger)}
}

func (s *CollectionServiceService) CreateCollection(ctx context.Context, req *pb.CreateCollectionReq) (*pb.CreateCollectionResp, error) {
	input := &biz.CreateCollectionInput{
		Name:        req.Name,
		Description: req.Description,
	}
	err := s.uc.CreateCollection(ctx, input)
	if err != nil {
		return nil, err
	}
	return &pb.CreateCollectionResp{}, nil
}
func (s *CollectionServiceService) RemoveCollection(ctx context.Context, req *pb.RemoveCollectionReq) (*pb.RemoveCollectionResp, error) {
	err := s.uc.RemoveCollection(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return &pb.RemoveCollectionResp{}, nil
}
func (s *CollectionServiceService) ListCollection(ctx context.Context, req *pb.ListCollectionReq) (*pb.ListCollectionResp, error) {
	input := &biz.ListCollectionInput{
		PageStats: &biz.PageStats{
			Page:     int(req.PageStats.Page),
			PageSize: int(req.PageStats.Size),
		},
	}
	output, err := s.uc.ListCollection(ctx, input)
	if err != nil {
		return nil, err
	}
	collections := output.Collections
	var retCollections []*pb.Collection
	for _, collection := range collections {
		retCollections = append(retCollections, &pb.Collection{
			Id:          collection.Id,
			UserId:      collection.UserId,
			Name:        collection.Name,
			Description: collection.Description,
		})
	}
	return &pb.ListCollectionResp{
		Collections: retCollections,
		PageStats:   &pb.PageStatsResp{Total: int32(output.Total)},
	}, nil
}
func (s *CollectionServiceService) UpdateCollection(ctx context.Context, req *pb.UpdateCollectionReq) (*pb.UpdateCollectionResp, error) {
	input := &biz.UpdateCollectionInput{
		Id:          req.Id,
		Name:        req.Name,
		Description: req.Description,
	}
	err := s.uc.UpdateCollection(ctx, input)
	if err != nil {
		return nil, err
	}
	return &pb.UpdateCollectionResp{}, nil
}
func (s *CollectionServiceService) AddVideo2Collection(ctx context.Context, req *pb.AddVideo2CollectionReq) (*pb.AddVideo2CollectionResp, error) {
	input := &biz.AddVideo2CollectionInput{
		CollectionId: req.CollectionId,
		VideoId:      req.VideoId,
	}
	err := s.uc.AddVideo2Collection(ctx, input)
	if err != nil {
		return nil, err
	}
	return &pb.AddVideo2CollectionResp{}, nil
}
func (s *CollectionServiceService) RemoveVideoFromCollection(ctx context.Context, req *pb.RemoveVideoFromCollectionReq) (*pb.RemoveVideoFromCollectionResp, error) {
	input := &biz.RemoveVideo4CollectionInput{
		CollectionId: req.CollectionId,
		VideoId:      req.VideoId,
	}
	err := s.uc.RemoveVideo4Collection(ctx, input)
	if err != nil {
		return nil, err
	}
	return &pb.RemoveVideoFromCollectionResp{}, nil
}
func (s *CollectionServiceService) ListVideo4Collection(ctx context.Context, req *pb.ListVideo4CollectionReq) (*pb.ListVideo4CollectionResp, error) {
	input := &biz.ListVideo4CollectionsInput{
		CollectionId: req.CollectionId,
		PageStats: &biz.PageStats{
			Page:     int(req.PageStats.Page),
			PageSize: int(req.PageStats.Size),
		},
	}
	output, err := s.uc.ListVideo4Collection(ctx, input)
	if err != nil {
		return nil, err
	}

	videos := output.Videos
	var retVideos []*pb.Video
	for _, video := range videos {
		retVideos = append(retVideos, &pb.Video{
			Id: video.ID,
			Author: &pb.VideoAuthor{
				Id:          video.Author.ID,
				Name:        video.Author.Name,
				Avatar:      video.Author.Avatar,
				IsFollowing: video.Author.IsFollowing,
			},
			PlayUrl:        video.PlayURL,
			CoverUrl:       video.CoverURL,
			FavoriteCount:  video.FavoriteCount,
			CommentCount:   video.CommentCount,
			IsFavorite:     video.IsFavorite,
			Title:          video.Title,
			IsCollected:    video.IsCollected,
			CollectedCount: video.CollectedCount,
		})
	}

	return &pb.ListVideo4CollectionResp{
		Videos: retVideos,
		PageStats: &pb.PageStatsResp{
			Total: int32(output.Total),
		},
	}, nil
}
