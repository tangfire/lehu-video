package service

import (
	"context"
	"github.com/spf13/cast"
	"time"

	pb "lehu-video/api/videoCore/service/v1"
	"lehu-video/app/videoCore/service/internal/biz"
	"lehu-video/app/videoCore/service/internal/pkg/utils"
)

type CommentServiceService struct {
	pb.UnimplementedCommentServiceServer
	uc *biz.CommentUsecase
}

func NewCommentServiceService(uc *biz.CommentUsecase) *CommentServiceService {
	return &CommentServiceService{
		uc: uc,
	}
}

func (s *CommentServiceService) CreateComment(ctx context.Context, req *pb.CreateCommentReq) (*pb.CreateCommentResp, error) {
	// 构建Command
	cmd := &biz.CreateCommentCommand{
		VideoID:     cast.ToInt64(req.VideoId),
		UserID:      cast.ToInt64(req.UserId),
		ParentID:    cast.ToInt64(req.ParentId),
		ReplyUserID: cast.ToInt64(req.ReplyUserId),
		Content:     req.Content,
	}

	// 调用业务层
	res, err := s.uc.CreateComment(ctx, cmd)
	if err != nil {
		return &pb.CreateCommentResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	comment := res.Comment

	return &pb.CreateCommentResp{
		Meta:    utils.GetSuccessMeta(),
		Comment: s.toPBComment(comment),
	}, nil
}

func (s *CommentServiceService) RemoveComment(ctx context.Context, req *pb.RemoveCommentReq) (*pb.RemoveCommentResp, error) {
	// 构建Command
	cmd := &biz.RemoveCommentCommand{
		CommentID: cast.ToInt64(req.CommentId),
		UserID:    cast.ToInt64(req.UserId),
	}

	// 调用业务层
	_, err := s.uc.RemoveComment(ctx, cmd)
	if err != nil {
		return &pb.RemoveCommentResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.RemoveCommentResp{
		Meta: utils.GetSuccessMeta(),
	}, nil
}

func (s *CommentServiceService) ListComment4Video(ctx context.Context, req *pb.ListComment4VideoReq) (*pb.ListComment4VideoResp, error) {
	// 构建Query
	query := &biz.ListVideoCommentsQuery{
		VideoID: cast.ToInt64(req.VideoId),
		PageStats: biz.PageStats{
			Page:     req.PageStats.Page,
			PageSize: req.PageStats.Size,
		},
		WithChildren: true, // 默认加载子评论
	}

	// 调用业务层
	result, err := s.uc.ListVideoComments(ctx, query)
	if err != nil {
		return &pb.ListComment4VideoResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	// 转换为proto结构
	pbComments := make([]*pb.Comment, 0, len(result.Comments))
	for _, comment := range result.Comments {
		pbComment := s.toPBComment(comment)
		pbComments = append(pbComments, pbComment)
	}

	return &pb.ListComment4VideoResp{
		Meta:        utils.GetSuccessMeta(),
		CommentList: pbComments,
		PageStats: &pb.PageStatsResp{
			Total: int32(result.Total),
		},
	}, nil
}

func (s *CommentServiceService) ListChildComment4Comment(ctx context.Context, req *pb.ListChildComment4CommentReq) (*pb.ListChildComment4CommentResp, error) {
	// 构建Query
	query := &biz.ListChildCommentsQuery{
		ParentID: cast.ToInt64(req.CommentId),
		PageStats: biz.PageStats{
			Page:     req.PageStats.Page,
			PageSize: req.PageStats.Size,
		},
	}

	// 调用业务层
	result, err := s.uc.ListChildComments(ctx, query)
	if err != nil {
		return &pb.ListChildComment4CommentResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	pbComments := make([]*pb.Comment, 0, len(result.Comments))
	for _, comment := range result.Comments {
		pbComments = append(pbComments, s.toPBComment(comment))
	}

	return &pb.ListChildComment4CommentResp{
		Meta:        utils.GetSuccessMeta(),
		CommentList: pbComments,
		PageStats: &pb.PageStatsResp{
			Total: int32(result.Total),
		},
	}, nil
}

func (s *CommentServiceService) GetCommentById(ctx context.Context, req *pb.GetCommentByIdReq) (*pb.GetCommentByIdResp, error) {
	// 构建Query
	query := &biz.GetCommentQuery{
		CommentID: cast.ToInt64(req.CommentId),
	}

	// 调用业务层
	result, err := s.uc.GetCommentByID(ctx, query)
	if err != nil {
		return &pb.GetCommentByIdResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	if result.Comment == nil {
		return &pb.GetCommentByIdResp{
			Meta: utils.GetMetaWithError(biz.ErrCommentNotFound),
		}, nil
	}

	return &pb.GetCommentByIdResp{
		Meta:    utils.GetSuccessMeta(),
		Comment: s.toPBComment(result.Comment),
	}, nil
}

func (s *CommentServiceService) CountComment4Video(ctx context.Context, req *pb.CountComment4VideoReq) (*pb.CountComment4VideoResp, error) {

	ids := cast.ToInt64Slice(req.VideoId)
	// 构建Query
	query := &biz.CountVideoCommentsQuery{
		VideoIDs: ids,
	}

	// 调用业务层
	result, err := s.uc.CountVideoComments(ctx, query)
	if err != nil {
		return &pb.CountComment4VideoResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	// 转换为proto结构
	results := make([]*pb.CountResult, 0, len(result.Counts))
	for videoID, count := range result.Counts {
		results = append(results, &pb.CountResult{
			Id:    cast.ToString(videoID),
			Count: count,
		})
	}

	return &pb.CountComment4VideoResp{
		Meta:    utils.GetSuccessMeta(),
		Results: results,
	}, nil
}

func (s *CommentServiceService) CountComment4User(ctx context.Context, req *pb.CountComment4UserReq) (*pb.CountComment4UserResp, error) {
	// 构建Query
	query := &biz.CountUserCommentsQuery{
		UserIDs: cast.ToInt64Slice(req.UserId),
	}

	// 调用业务层
	result, err := s.uc.CountUserComments(ctx, query)
	if err != nil {
		return &pb.CountComment4UserResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	// 转换为proto结构
	results := make([]*pb.CountResult, 0, len(result.Counts))
	for userID, count := range result.Counts {
		results = append(results, &pb.CountResult{
			Id:    cast.ToString(userID),
			Count: count,
		})
	}

	return &pb.CountComment4UserResp{
		Meta:    utils.GetSuccessMeta(),
		Results: results,
	}, nil
}

// toPBComment 将业务层Comment转换为proto Comment
func (s *CommentServiceService) toPBComment(comment *biz.Comment) *pb.Comment {
	if comment == nil {
		return nil
	}

	pbComment := &pb.Comment{
		Id:          cast.ToString(comment.ID),
		VideoId:     cast.ToString(comment.VideoID),
		Content:     comment.Content,
		Date:        comment.CreateTime.Format(time.DateTime),
		LikeCount:   comment.LikeCount,
		ReplyCount:  comment.ReplyCount,
		UserId:      cast.ToString(comment.UserID),
		ParentId:    cast.ToString(comment.ParentID),
		ReplyUserId: cast.ToString(comment.ReplyUserID),
		Comments:    []*pb.Comment{},
	}

	// 如果有子评论，转换子评论
	if len(comment.ChildComments) > 0 {
		for _, child := range comment.ChildComments {
			pbComment.Comments = append(pbComment.Comments, s.toPBComment(child))
		}
	}

	return pbComment
}
