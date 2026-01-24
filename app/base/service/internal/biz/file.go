package biz

import (
	"context"
	"github.com/go-kratos/kratos/v2/log"
	pb "lehu-video/api/base/service/v1"
	"time"
)

type File struct {
	Id         int64
	DomainName string
	BizName    string
	Hash       string
	FileType   string
	FileSize   int64
	Uploaded   bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type FileRepo interface {
	CheckFileExistedAndGetFile(ctx context.Context, fileCtx *File) (int64, bool, error)
}

type FileUsecase struct {
	repo  FileRepo
	minio MinioRepo
	log   *log.Helper
}

func NewFileUsecase(repo FileRepo, logger log.Logger) *FileUsecase {
	return &FileUsecase{repo: repo, log: log.NewHelper(logger)}
}

func (uc *FileUsecase) PreSignGet(ctx context.Context, req *pb.PreSignGetReq) (*pb.PreSignGetResp, error) {
	return &pb.PreSignGetResp{}, nil
}
func (uc *FileUsecase) PreSignPut(ctx context.Context, req *pb.PreSignPutReq) (*pb.PreSignPutResp, error) {
	return &pb.PreSignPutResp{}, nil
}
func (uc *FileUsecase) ReportUploaded(ctx context.Context, req *pb.ReportUploadedReq) (*pb.ReportUploadedResp, error) {
	return &pb.ReportUploadedResp{}, nil
}
func (uc *FileUsecase) PreSignSlicingPut(ctx context.Context, req *pb.PreSignSlicingPutReq) (*pb.PreSignSlicingPutResp, error) {
	return &pb.PreSignSlicingPutResp{}, nil
}
func (uc *FileUsecase) GetProgressRate4SlicingPut(ctx context.Context, req *pb.GetProgressRate4SlicingPutReq) (*pb.GetProgressRate4SlicingPutResp, error) {
	return &pb.GetProgressRate4SlicingPutResp{}, nil
}
func (uc *FileUsecase) MergeFileParts(ctx context.Context, req *pb.MergeFilePartsReq) (*pb.MergeFilePartsResp, error) {
	return &pb.MergeFilePartsResp{}, nil
}
func (uc *FileUsecase) RemoveFile(ctx context.Context, req *pb.RemoveFileReq) (*pb.RemoveFileResp, error) {
	return &pb.RemoveFileResp{}, nil
}
func (uc *FileUsecase) GetFileInfoById(ctx context.Context, req *pb.GetFileInfoByIdReq) (*pb.GetFileInfoByIdResp, error) {
	return &pb.GetFileInfoByIdResp{}, nil
}
