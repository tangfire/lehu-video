package service

import (
	"context"
	"lehu-video/app/videoApi/service/internal/biz"

	pb "lehu-video/api/videoApi/service/v1"
)

type FileServiceService struct {
	pb.UnimplementedFileServiceServer

	uc *biz.FileUsecase
}

func NewFileServiceService(uc *biz.FileUsecase) *FileServiceService {
	return &FileServiceService{uc: uc}
}

func (s *FileServiceService) PreSignUploadingPublicFile(ctx context.Context, req *pb.PreSignUploadPublicFileReq) (*pb.PreSignUploadPublicFileResp, error) {
	bizReq := &biz.PreSignUploadPublicReq{
		Hash:     req.Hash,
		FileType: req.FileType,
		FileName: req.Filename,
		Size:     req.Size,
	}
	resp, err := s.uc.PreSignUploadingPublicFile(ctx, bizReq)
	if err != nil {
		return nil, err
	}
	return &pb.PreSignUploadPublicFileResp{
		Url:    resp.Url,
		FileId: resp.FileId,
	}, nil
}
func (s *FileServiceService) ReportPublicFileUploaded(ctx context.Context, req *pb.ReportPublicFileUploadedReq) (*pb.ReportPublicFileUploadedResp, error) {
	bizReq := &biz.ReportPublicFileUploadedReq{
		FileId: req.FileId,
	}
	resp, err := s.uc.ReportPublicFileUploaded(ctx, bizReq)
	if err != nil {
		return nil, err
	}
	return &pb.ReportPublicFileUploadedResp{
		ObjectName: resp.ObjectName,
	}, nil
}
