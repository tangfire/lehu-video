package service

import (
	"context"
	pb "lehu-video/api/base/service/v1"
	"lehu-video/app/base/service/internal/biz"
	"lehu-video/app/base/service/internal/pkg/utils"
)

type FileServiceService struct {
	pb.UnimplementedFileServiceServer

	uc *biz.FileUsecase
}

func NewFileServiceService(uc *biz.FileUsecase) *FileServiceService {
	return &FileServiceService{uc: uc}
}

func PbToBiz(fileCtx *pb.FileContext) *biz.File {
	return &biz.File{
		Id:            fileCtx.FileId,
		DomainName:    fileCtx.Domain,
		BizName:       fileCtx.BizName,
		Hash:          fileCtx.Hash,
		FileType:      fileCtx.FileType,
		FileSize:      fileCtx.Size,
		FileName:      fileCtx.Filename,
		ExpireSeconds: fileCtx.ExpireSeconds,
	}
}

func (s *FileServiceService) PreSignGet(ctx context.Context, req *pb.PreSignGetReq) (*pb.PreSignGetResp, error) {
	file := PbToBiz(req.FileContext)
	url, err := s.uc.PreSignGet(ctx, file)
	if err != nil {
		return &pb.PreSignGetResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}
	return &pb.PreSignGetResp{
		Meta: utils.GetSuccessMeta(),
		Url:  url,
	}, nil
}
func (s *FileServiceService) PreSignPut(ctx context.Context, req *pb.PreSignPutReq) (*pb.PreSignPutResp, error) {
	file := PbToBiz(req.FileContext)
	exist, fileId, err := s.uc.CheckFileExistedAndGetFile(ctx, file)
	if err != nil {
		return &pb.PreSignPutResp{
			Meta:   utils.GetMetaWithError(err),
			Url:    "",
			FileId: 0,
		}, nil
	}
	if exist {
		return &pb.PreSignPutResp{
			Meta:   utils.GetSuccessMeta(),
			Url:    "",
			FileId: fileId,
		}, nil
	}
	url, fileId, err := s.uc.PreSignPut(ctx, file)
	if err != nil {
		return &pb.PreSignPutResp{
			Meta:   utils.GetMetaWithError(err),
			Url:    "",
			FileId: 0,
		}, nil
	}
	return &pb.PreSignPutResp{
		Meta:   utils.GetSuccessMeta(),
		Url:    url,
		FileId: fileId,
	}, nil
}
func (s *FileServiceService) ReportUploaded(ctx context.Context, req *pb.ReportUploadedReq) (*pb.ReportUploadedResp, error) {
	file := PbToBiz(req.FileContext)
	err := s.uc.ReportUploaded(ctx, file)
	if err != nil {
		return &pb.ReportUploadedResp{
			Meta: utils.GetMetaWithError(err),
			Url:  "",
		}, nil
	}
	url, err := s.uc.PreSignGet(ctx, file)
	if err != nil {
		return &pb.ReportUploadedResp{
			Meta: utils.GetMetaWithError(err),
			Url:  "",
		}, nil
	}
	return &pb.ReportUploadedResp{
		Meta: utils.GetSuccessMeta(),
		Url:  url,
	}, nil
}

func (s *FileServiceService) PreSignSlicingPut(ctx context.Context, req *pb.PreSignSlicingPutReq) (*pb.PreSignSlicingPutResp, error) {
	file := PbToBiz(req.FileContext)
	exist, fileId, err := s.uc.CheckFileExistedAndGetFile(ctx, file)
	if err != nil {
		return &pb.PreSignSlicingPutResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}
	if exist {
		return &pb.PreSignSlicingPutResp{
			Meta:     utils.GetSuccessMeta(),
			FileId:   fileId,
			Uploaded: true,
		}, nil
	}

	sf, err := s.uc.PreSignSlicingPut(ctx, file)
	if err != nil {
		return &pb.PreSignSlicingPutResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.PreSignSlicingPutResp{
		Meta:     utils.GetSuccessMeta(),
		Urls:     sf.UploadUrl,
		UploadId: sf.UploadId,
		Parts:    sf.TotalParts,
		FileId:   sf.File.Id,
	}, nil
}
func (s *FileServiceService) GetProgressRate4SlicingPut(ctx context.Context, req *pb.GetProgressRate4SlicingPutReq) (*pb.GetProgressRate4SlicingPutResp, error) {
	file := PbToBiz(req.FileContext)
	result, err := s.uc.GetProgressRate4SlicingPut(ctx, req.UploadId, file)
	if err != nil {
		return &pb.GetProgressRate4SlicingPutResp{
			Meta: utils.GetSuccessMeta(),
		}, nil
	}
	total := 0
	finished := 0
	for _, uploaded := range result {
		if uploaded {
			finished++
		}

		total++
	}
	return &pb.GetProgressRate4SlicingPutResp{
		Meta:         utils.GetSuccessMeta(),
		Parts:        result,
		ProgressRate: float32(finished*100) / float32(total),
	}, nil
}
func (s *FileServiceService) MergeFileParts(ctx context.Context, req *pb.MergeFilePartsReq) (*pb.MergeFilePartsResp, error) {
	file := PbToBiz(req.FileContext)
	err := s.uc.MergeFileParts(ctx, req.UploadId, file)
	if err != nil {
		return &pb.MergeFilePartsResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}
	err = s.uc.ReportUploaded(ctx, file)
	if err != nil {
		return &pb.MergeFilePartsResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}
	return &pb.MergeFilePartsResp{
		Meta: utils.GetSuccessMeta(),
	}, nil
}
func (s *FileServiceService) RemoveFile(ctx context.Context, req *pb.RemoveFileReq) (*pb.RemoveFileResp, error) {
	file := PbToBiz(req.FileContext)
	err := s.uc.RemoveFile(ctx, file)
	if err != nil {
		return &pb.RemoveFileResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}
	return &pb.RemoveFileResp{
		Meta: utils.GetSuccessMeta(),
	}, nil
}
func (s *FileServiceService) GetFileInfoById(ctx context.Context, req *pb.GetFileInfoByIdReq) (*pb.GetFileInfoByIdResp, error) {
	file, err := s.uc.GetFileInfoById(ctx, req.DomainName, req.BizName, req.FileId)
	if err != nil {
		return &pb.GetFileInfoByIdResp{
			Meta: utils.GetMetaWithError(err),
		}, err
	}
	return &pb.GetFileInfoByIdResp{
		Meta:       utils.GetSuccessMeta(),
		ObjectName: file.GetObjectName(),
		Hash:       file.Hash,
	}, nil
}
