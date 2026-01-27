package biz

import (
	"context"
)

// BaseAdapter 基础服务适配器接口
type BaseAdapter interface {
	CreateVerificationCode(ctx context.Context, bits, expireTime int64) (int64, error)
	ValidateVerificationCode(ctx context.Context, codeId int64, code string) error
	Register(ctx context.Context, mobile, email, password string) (int64, error)
	CheckAccount(ctx context.Context, mobile, email, password string) (int64, error)

	PreSign4PublicUpload(ctx context.Context, hash, fileType, fileName string, size, expireSeconds int64) (int64, string, error)
	PreSign4Upload(ctx context.Context, hash, fileType, fileName string, size, expireSeconds int64) (int64, string, error)
	ReportPublicUploaded(ctx context.Context, fileId int64) (string, error)
	ReportUploaded(ctx context.Context, fileId int64) (string, error)
	GetFileInfoById(ctx context.Context, fileId int64) (*FileInfo, error)
}
