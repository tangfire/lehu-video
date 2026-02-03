package biz

import (
	"context"
	"github.com/go-kratos/kratos/v2/log"
)

// FileInfo 文件信息
type FileInfo struct {
	ObjectName string `json:"object_name"`
	Hash       string `json:"hash"`
}

type PreSignUploadPublicReq struct {
	Hash     string
	FileType string
	FileName string
	Size     int64
}

type PreSignUploadPublicResp struct {
	Url    string
	FileId string
}

type ReportPublicFileUploadedReq struct {
	FileId string
}

type ReportPublicFileUploadedResp struct {
	ObjectName string
}

type FileUsecase struct {
	base BaseAdapter
	log  *log.Helper
}

func NewFileUsecase(base BaseAdapter, logger log.Logger) *FileUsecase {
	return &FileUsecase{
		base: base,
		log:  log.NewHelper(logger),
	}
}

func (uc *FileUsecase) PreSignUploadingPublicFile(ctx context.Context, req *PreSignUploadPublicReq) (*PreSignUploadPublicResp, error) {
	fileId, url, err := uc.base.PreSign4PublicUpload(ctx, req.Hash, req.FileType, req.FileName, req.Size, 3600)
	if err != nil {
		return nil, err
	}
	return &PreSignUploadPublicResp{
		Url:    url,
		FileId: fileId,
	}, nil
}

func (uc *FileUsecase) ReportPublicFileUploaded(ctx context.Context, req *ReportPublicFileUploadedReq) (*ReportPublicFileUploadedResp, error) {
	_, err := uc.base.ReportPublicUploaded(ctx, req.FileId)
	if err != nil {
		return nil, err
	}
	info, err := uc.base.GetFileInfoById(ctx, req.FileId)
	if err != nil {
		return nil, err
	}
	return &ReportPublicFileUploadedResp{ObjectName: info.ObjectName}, nil
}
