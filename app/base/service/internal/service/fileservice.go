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
	query := &biz.PreSignGetQuery{File: file}
	result, err := s.uc.PreSignGet(ctx, query)
	if err != nil {
		return &pb.PreSignGetResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}
	return &pb.PreSignGetResp{
		Meta: utils.GetSuccessMeta(),
		Url:  result.Url,
	}, nil
}

func (s *FileServiceService) PreSignPut(ctx context.Context, req *pb.PreSignPutReq) (*pb.PreSignPutResp, error) {
	file := PbToBiz(req.FileContext)

	// 检查文件是否已存在
	checkQuery := &biz.CheckFileQuery{File: file}
	checkResult, err := s.uc.CheckFileExistedAndGetFile(ctx, checkQuery)
	if err != nil {
		return &pb.PreSignPutResp{
			Meta:   utils.GetMetaWithError(err),
			Url:    "",
			FileId: 0,
		}, nil
	}
	if checkResult.Exists {
		return &pb.PreSignPutResp{
			Meta:   utils.GetSuccessMeta(),
			Url:    "",
			FileId: checkResult.FileId,
		}, nil
	}

	// 生成预签名URL
	cmd := &biz.PreSignPutCommand{File: file}
	result, err := s.uc.PreSignPut(ctx, cmd)
	if err != nil {
		return &pb.PreSignPutResp{
			Meta:   utils.GetMetaWithError(err),
			Url:    "",
			FileId: 0,
		}, nil
	}
	return &pb.PreSignPutResp{
		Meta:   utils.GetSuccessMeta(),
		Url:    result.Url,
		FileId: result.FileId,
	}, nil
}

func (s *FileServiceService) ReportUploaded(ctx context.Context, req *pb.ReportUploadedReq) (*pb.ReportUploadedResp, error) {
	file := PbToBiz(req.FileContext)
	cmd := &biz.ReportUploadedCommand{File: file}
	result, err := s.uc.ReportUploaded(ctx, cmd)
	if err != nil {
		return &pb.ReportUploadedResp{
			Meta: utils.GetMetaWithError(err),
			Url:  "",
		}, nil
	}
	return &pb.ReportUploadedResp{
		Meta: utils.GetSuccessMeta(),
		Url:  result.Url,
	}, nil
}

func (s *FileServiceService) PreSignSlicingPut(ctx context.Context, req *pb.PreSignSlicingPutReq) (*pb.PreSignSlicingPutResp, error) {
	file := PbToBiz(req.FileContext)

	// 检查文件是否已存在
	checkQuery := &biz.CheckFileQuery{File: file}
	checkResult, err := s.uc.CheckFileExistedAndGetFile(ctx, checkQuery)
	if err != nil {
		return &pb.PreSignSlicingPutResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}
	if checkResult.Exists {
		return &pb.PreSignSlicingPutResp{
			Meta:     utils.GetSuccessMeta(),
			FileId:   checkResult.FileId,
			Uploaded: true,
		}, nil
	}

	cmd := &biz.PreSignSlicingPutCommand{File: file}
	result, err := s.uc.PreSignSlicingPut(ctx, cmd)
	if err != nil {
		return &pb.PreSignSlicingPutResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.PreSignSlicingPutResp{
		Meta:     utils.GetSuccessMeta(),
		Urls:     result.SlicingFile.UploadUrl,
		UploadId: result.SlicingFile.UploadId,
		Parts:    result.SlicingFile.TotalParts,
		FileId:   result.FileId,
	}, nil
}

func (s *FileServiceService) GetProgressRate4SlicingPut(ctx context.Context, req *pb.GetProgressRate4SlicingPutReq) (*pb.GetProgressRate4SlicingPutResp, error) {
	file := PbToBiz(req.FileContext)
	query := &biz.GetProgressRate4SlicingPutQuery{
		UploadId: req.UploadId,
		File:     file,
	}
	result, err := s.uc.GetProgressRate4SlicingPut(ctx, query)
	if err != nil {
		return &pb.GetProgressRate4SlicingPutResp{
			Meta: utils.GetSuccessMeta(),
		}, nil
	}

	if result.Progress == nil {
		return &pb.GetProgressRate4SlicingPutResp{
			Meta: utils.GetSuccessMeta(),
		}, nil
	}

	total := 0
	finished := 0
	for _, uploaded := range result.Progress {
		if uploaded {
			finished++
		}
		total++
	}

	var progressRate float32 = 0
	if total > 0 {
		progressRate = float32(finished*100) / float32(total)
	}

	return &pb.GetProgressRate4SlicingPutResp{
		Meta:         utils.GetSuccessMeta(),
		Parts:        result.Progress,
		ProgressRate: progressRate,
	}, nil
}

func (s *FileServiceService) MergeFileParts(ctx context.Context, req *pb.MergeFilePartsReq) (*pb.MergeFilePartsResp, error) {
	file := PbToBiz(req.FileContext)
	cmd := &biz.MergeFilePartsCommand{
		UploadId: req.UploadId,
		File:     file,
	}
	_, err := s.uc.MergeFileParts(ctx, cmd)
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
	cmd := &biz.RemoveFileCommand{File: file}
	_, err := s.uc.RemoveFile(ctx, cmd)
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
	query := &biz.GetFileInfoByIdQuery{
		DomainName: req.DomainName,
		BizName:    req.BizName,
		FileId:     req.FileId,
	}
	result, err := s.uc.GetFileInfoById(ctx, query)
	if err != nil {
		return &pb.GetFileInfoByIdResp{
			Meta: utils.GetMetaWithError(err),
		}, err
	}

	if result.File == nil {
		return &pb.GetFileInfoByIdResp{
			Meta: utils.GetSuccessMeta(),
		}, nil
	}

	return &pb.GetFileInfoByIdResp{
		Meta:       utils.GetSuccessMeta(),
		ObjectName: result.File.GetObjectName(),
		Hash:       result.File.Hash,
	}, nil
}
