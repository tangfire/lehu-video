package biz

import (
	"context"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	pb "lehu-video/api/videoCore/service/v1"
	"lehu-video/app/videoCore/service/internal/pkg/utils"
	"time"
)

type Comment struct {
	Id            int64
	VideoId       int64
	UserId        int64
	ParentId      int64
	ToUserId      int64
	Content       string
	Date          string
	CreateTime    time.Time
	Comments      []*Comment // 子评论
	ChildNumbers  int64      // 子评论个数
	FirstComments []*Comment // 最初的x条子评论
}

type CommentRepo interface {
	CreateComment(ctx context.Context, comment *Comment) error
	RemoveComment(ctx context.Context, comment *Comment) error
	ListCommentByVideoId(ctx context.Context, videoId int64) ([]*Comment, error)
	ListChildCommentById(ctx context.Context, commentId int64) ([]*Comment, error)
}

type CommentUsecase struct {
	repo CommentRepo
	log  *log.Helper
}

func NewCommentUsecase(repo CommentRepo, logger log.Logger) *CommentUsecase {
	return &CommentUsecase{
		repo: repo,
		log:  log.NewHelper(logger),
	}
}

func (uc *CommentUsecase) CreateComment(ctx context.Context, req *pb.CreateCommentReq) (*pb.CreateCommentResp, error) {
	comment := &Comment{
		Id:            int64(uuid.New().ID()),
		VideoId:       req.VideoId,
		UserId:        req.UserId,
		ParentId:      req.ParentId,
		ToUserId:      0,
		Content:       req.Content,
		Date:          time.Now().Format(time.DateTime),
		CreateTime:    time.Now(),
		Comments:      nil,
		ChildNumbers:  0,
		FirstComments: nil,
	}
	err := uc.repo.CreateComment(ctx, comment)
	if err != nil {
		return nil, err
	}
	return &pb.CreateCommentResp{
		Meta: utils.GetSuccessMeta(),
	}, nil
}

func (uc *CommentUsecase) RemoveComment(ctx context.Context, req *pb.RemoveCommentReq) (*pb.RemoveCommentResp, error) {

}

func (uc *CommentUsecase) ListComment4Video(ctx context.Context, req *pb.ListComment4VideoReq) (*pb.ListComment4VideoResp, error) {

}
