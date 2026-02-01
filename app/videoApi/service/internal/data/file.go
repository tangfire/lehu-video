package data

import (
	"context"
	base "lehu-video/api/base/service/v1"
	"lehu-video/app/videoApi/service/internal/biz"
	"lehu-video/app/videoApi/service/internal/pkg/utils/respcheck"
)

func (r *baseAdapterImpl) PreSign4PublicUpload(ctx context.Context, hash, fileType, fileName string, size, expireSeconds int64) (string, string, error) {
	fileCtx := &base.FileContext{
		Domain:        DomainName,
		BizName:       Public,
		Hash:          hash,
		FileType:      fileType,
		Size:          size,
		ExpireSeconds: expireSeconds,
		Filename:      fileName,
	}
	resp, err := r.file.PreSignPut(ctx, &base.PreSignPutReq{
		FileContext: fileCtx,
	})
	if err != nil {
		return "0", "", err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return "0", "", err
	}
	return resp.FileId, resp.Url, nil
}

func (r *baseAdapterImpl) PreSign4Upload(ctx context.Context, hash, fileType, fileName string, size, expireSeconds int64) (string, string, error) {
	fileCtx := &base.FileContext{
		Domain:        DomainName,
		BizName:       BizName,
		Hash:          hash,
		FileType:      fileType,
		Size:          size,
		ExpireSeconds: expireSeconds,
		Filename:      fileName,
	}
	resp, err := r.file.PreSignPut(ctx, &base.PreSignPutReq{
		FileContext: fileCtx,
	})
	if err != nil {
		return "0", "", err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return "0", "", err
	}
	return resp.FileId, resp.Url, nil
}

func (r *baseAdapterImpl) ReportPublicUploaded(ctx context.Context, fileId string) (string, error) {
	fileCtx := &base.FileContext{
		Domain:        DomainName,
		BizName:       Public,
		FileId:        fileId,
		ExpireSeconds: 7200,
	}
	resp, err := r.file.ReportUploaded(ctx, &base.ReportUploadedReq{
		FileContext: fileCtx,
	})

	if err != nil {
		return "", err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return "", err
	}
	return resp.Url, nil
}

func (r *baseAdapterImpl) ReportUploaded(ctx context.Context, fileId string) (string, error) {
	fileCtx := &base.FileContext{
		Domain:        DomainName,
		BizName:       BizName,
		FileId:        fileId,
		ExpireSeconds: 7200,
	}
	resp, err := r.file.ReportUploaded(ctx, &base.ReportUploadedReq{
		FileContext: fileCtx,
	})

	if err != nil {
		return "", err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return "", err
	}
	return resp.Url, nil
}

func (r *baseAdapterImpl) GetFileInfoById(ctx context.Context, fileId string) (*biz.FileInfo, error) {
	resp, err := r.file.GetFileInfoById(ctx, &base.GetFileInfoByIdReq{
		FileId:     fileId,
		DomainName: DomainName,
		BizName:    Public,
	})
	if err != nil {
		return nil, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return nil, err
	}
	ret := &biz.FileInfo{
		ObjectName: resp.ObjectName,
		Hash:       resp.Hash,
	}
	return ret, nil
}
