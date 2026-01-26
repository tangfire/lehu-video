package service

import (
	"context"
	"lehu-video/app/videoCore/service/internal/biz"
	"lehu-video/app/videoCore/service/internal/pkg/utils"

	pb "lehu-video/api/videoCore/service/v1"
)

type CollectionServiceService struct {
	pb.UnimplementedCollectionServiceServer

	uc *biz.CollectionUsecase
}

func NewCollectionServiceService(uc *biz.CollectionUsecase) *CollectionServiceService {
	return &CollectionServiceService{uc: uc}
}

func (s *CollectionServiceService) CreateCollection(ctx context.Context, req *pb.CreateCollectionReq) (*pb.CreateCollectionResp, error) {
	// ✅ 改为Command
	cmd := &biz.CreateCollectionCommand{
		UserId:      req.UserId,
		Name:        req.Name,
		Description: req.Description,
	}

	_, err := s.uc.CreateCollection(ctx, cmd)
	if err != nil {
		return &pb.CreateCollectionResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.CreateCollectionResp{
		Meta: utils.GetSuccessMeta(),
	}, nil
}

func (s *CollectionServiceService) UpdateCollection(ctx context.Context, req *pb.UpdateCollectionReq) (*pb.UpdateCollectionResp, error) {
	// ✅ 改为Command
	cmd := &biz.UpdateCollectionCommand{
		CollectionId: req.Id,
		Name:         req.Name,
		Description:  req.Description,
	}

	_, err := s.uc.UpdateCollection(ctx, cmd)
	if err != nil {
		return &pb.UpdateCollectionResp{Meta: utils.GetMetaWithError(err)}, nil
	}

	return &pb.UpdateCollectionResp{
		Meta: utils.GetSuccessMeta(),
	}, nil
}

func (s *CollectionServiceService) RemoveCollection(ctx context.Context, req *pb.RemoveCollectionReq) (*pb.RemoveCollectionResp, error) {
	// ✅ 改为Command
	cmd := &biz.RemoveCollectionCommand{
		CollectionId: req.Id,
	}

	_, err := s.uc.RemoveCollection(ctx, cmd)
	if err != nil {
		return &pb.RemoveCollectionResp{Meta: utils.GetMetaWithError(err)}, nil
	}

	return &pb.RemoveCollectionResp{
		Meta: utils.GetSuccessMeta(),
	}, nil
}

func (s *CollectionServiceService) GetCollectionById(ctx context.Context, req *pb.GetCollectionByIdReq) (*pb.GetCollectionByIdResp, error) {
	// ✅ 改为Query
	query := &biz.GetCollectionByIdQuery{
		CollectionId: req.Id,
	}

	result, err := s.uc.GetCollectionById(ctx, query)
	if err != nil {
		return &pb.GetCollectionByIdResp{Meta: utils.GetMetaWithError(err)}, nil
	}

	if result.Collection == nil {
		return &pb.GetCollectionByIdResp{
			Meta: utils.GetMetaWithError(nil),
		}, nil
	}

	return &pb.GetCollectionByIdResp{
		Collection: &pb.Collection{
			Id:          result.Collection.Id,
			UserId:      result.Collection.UserId,
			Name:        result.Collection.Title,
			Description: result.Collection.Description,
		},
		Meta: utils.GetSuccessMeta(),
	}, nil
}

func (s *CollectionServiceService) ListCollection(ctx context.Context, req *pb.ListCollectionReq) (*pb.ListCollectionResp, error) {
	// ✅ 改为Query
	query := &biz.ListCollectionQuery{
		UserId: req.UserId,
		PageStats: biz.PageStats{
			Page:     req.PageStats.Page,
			PageSize: req.PageStats.Size,
		},
	}

	result, err := s.uc.ListCollection(ctx, query)
	if err != nil {
		return &pb.ListCollectionResp{Meta: utils.GetMetaWithError(err)}, nil
	}

	var collections []*pb.Collection
	for _, c := range result.Collections {
		collections = append(collections, &pb.Collection{
			Id:          c.Id,
			UserId:      c.UserId,
			Name:        c.Title,
			Description: c.Description,
		})
	}

	return &pb.ListCollectionResp{
		Meta:        utils.GetSuccessMeta(),
		Collections: collections,
		PageStats:   &pb.PageStatsResp{Total: int32(result.Total)},
	}, nil
}

func (s *CollectionServiceService) AddVideo2Collection(ctx context.Context, req *pb.AddVideo2CollectionReq) (*pb.AddVideo2CollectionResp, error) {
	// ✅ 改为Command
	cmd := &biz.AddVideoToCollectionCommand{
		UserId:       req.UserId,
		CollectionId: req.CollectionId,
		VideoId:      req.VideoId,
	}

	_, err := s.uc.AddVideoToCollection(ctx, cmd)
	if err != nil {
		return &pb.AddVideo2CollectionResp{Meta: utils.GetMetaWithError(err)}, nil
	}

	return &pb.AddVideo2CollectionResp{
		Meta: utils.GetSuccessMeta(),
	}, nil
}

func (s *CollectionServiceService) RemoveVideoFromCollection(ctx context.Context, req *pb.RemoveVideoFromCollectionReq) (*pb.RemoveVideoFromCollectionResp, error) {
	// ✅ 改为Command
	cmd := &biz.RemoveVideoFromCollectionCommand{
		UserId:       req.UserId,
		CollectionId: req.CollectionId,
		VideoId:      req.VideoId,
	}

	_, err := s.uc.RemoveVideoFromCollection(ctx, cmd)
	if err != nil {
		return &pb.RemoveVideoFromCollectionResp{Meta: utils.GetMetaWithError(err)}, nil
	}

	return &pb.RemoveVideoFromCollectionResp{
		Meta: utils.GetSuccessMeta(),
	}, nil
}

func (s *CollectionServiceService) ListVideo4Collection(ctx context.Context, req *pb.ListVideo4CollectionReq) (*pb.ListVideo4CollectionResp, error) {
	// ✅ 改为Query
	query := &biz.ListVideo4CollectionQuery{
		CollectionId: req.CollectionId,
		PageStats: biz.PageStats{
			Page:     req.PageStats.Page,
			PageSize: req.PageStats.Size,
		},
	}

	result, err := s.uc.ListVideo4Collection(ctx, query)
	if err != nil {
		return &pb.ListVideo4CollectionResp{Meta: utils.GetMetaWithError(err)}, nil
	}

	return &pb.ListVideo4CollectionResp{
		Meta:        utils.GetSuccessMeta(),
		VideoIdList: result.VideoIds,
		PageStats:   &pb.PageStatsResp{Total: int32(result.Total)},
	}, nil
}

func (s *CollectionServiceService) IsCollected(ctx context.Context, req *pb.IsCollectedReq) (*pb.IsCollectedResp, error) {
	// ✅ 改为Query
	query := &biz.IsCollectedQuery{
		UserId:   req.UserId,
		VideoIds: req.VideoIdList,
	}

	result, err := s.uc.IsCollected(ctx, query)
	if err != nil {
		return &pb.IsCollectedResp{Meta: utils.GetMetaWithError(err)}, nil
	}

	return &pb.IsCollectedResp{
		Meta:        utils.GetSuccessMeta(),
		VideoIdList: result.CollectedVideoIds,
	}, nil
}

func (s *CollectionServiceService) CountCollect4Video(ctx context.Context, req *pb.CountCollect4VideoReq) (*pb.CountCollect4VideoResp, error) {
	// ✅ 改为Query
	query := &biz.CountCollect4VideoQuery{
		VideoIds: req.VideoIdList,
	}

	result, err := s.uc.CountCollectedNumber4Video(ctx, query)
	if err != nil {
		return &pb.CountCollect4VideoResp{Meta: utils.GetMetaWithError(err)}, nil
	}

	var results []*pb.CountCollect4VideoResult
	for _, item := range result.Counts {
		results = append(results, &pb.CountCollect4VideoResult{
			Id:    item.Id,
			Count: item.Count,
		})
	}

	return &pb.CountCollect4VideoResp{
		Meta:        utils.GetSuccessMeta(),
		CountResult: results,
	}, nil
}
