package service

import (
	"context"

	pb "lehu-video/api/base/service/v1"
)

type FileServiceService struct {
	pb.UnimplementedFileServiceServer
}

func NewFileServiceService() *FileServiceService {
	return &FileServiceService{}
}

func (s *FileServiceService) PreSignGet(ctx context.Context, req *pb.PreSignGetReq) (*pb.PreSignGetResp, error) {
	return &pb.PreSignGetResp{}, nil
}
func (s *FileServiceService) PreSignPut(ctx context.Context, req *pb.PreSignPutReq) (*pb.PreSignPutResp, error) {
	return &pb.PreSignPutResp{}, nil
}
func (s *FileServiceService) ReportUploaded(ctx context.Context, req *pb.ReportUploadedReq) (*pb.ReportUploadedResp, error) {
	return &pb.ReportUploadedResp{}, nil
}
func (s *FileServiceService) PreSignSlicingPut(ctx context.Context, req *pb.PreSignSlicingPutReq) (*pb.PreSignSlicingPutResp, error) {
	return &pb.PreSignSlicingPutResp{}, nil
}
func (s *FileServiceService) GetProgressRate4SlicingPut(ctx context.Context, req *pb.GetProgressRate4SlicingPutReq) (*pb.GetProgressRate4SlicingPutResp, error) {
	return &pb.GetProgressRate4SlicingPutResp{}, nil
}
func (s *FileServiceService) MergeFileParts(ctx context.Context, req *pb.MergeFilePartsReq) (*pb.MergeFilePartsResp, error) {
	return &pb.MergeFilePartsResp{}, nil
}
func (s *FileServiceService) RemoveFile(ctx context.Context, req *pb.RemoveFileReq) (*pb.RemoveFileResp, error) {
	return &pb.RemoveFileResp{}, nil
}
func (s *FileServiceService) GetFileInfoById(ctx context.Context, req *pb.GetFileInfoByIdReq) (*pb.GetFileInfoByIdResp, error) {
	return &pb.GetFileInfoByIdResp{}, nil
}
