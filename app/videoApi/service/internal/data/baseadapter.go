package data

import (
	"context"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	base "lehu-video/api/base/service/v1"
	"lehu-video/app/videoApi/service/internal/biz"
	"lehu-video/app/videoApi/service/internal/pkg/utils/respcheck"
)

const (
	DomainName = "shortvideo"
	BizName    = "short_video"
	Public     = "public"
)

type baseAdapterImpl struct {
	account base.AccountServiceClient
	auth    base.AuthServiceClient
	file    base.FileServiceClient
}

func NewBaseAdapter(account base.AccountServiceClient, auth base.AuthServiceClient, file base.FileServiceClient) biz.BaseAdapter {
	return &baseAdapterImpl{
		account: account,
		auth:    auth,
		file:    file,
	}
}

func NewAccountServiceClient(r registry.Discovery) base.AccountServiceClient {
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint("discovery:///lehu-video.base.service"),
		grpc.WithDiscovery(r),
		grpc.WithMiddleware(
			recovery.Recovery(),
		),
	)
	if err != nil {
		panic(err)
	}
	return base.NewAccountServiceClient(conn)
}

func NewAuthServiceClient(r registry.Discovery) base.AuthServiceClient {
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint("discovery:///lehu-video.base.service"),
		grpc.WithDiscovery(r),
		grpc.WithMiddleware(
			recovery.Recovery(),
		),
	)
	if err != nil {
		panic(err)
	}
	return base.NewAuthServiceClient(conn)
}

func NewFileServiceClient(r registry.Discovery) base.FileServiceClient {
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint("discovery:///lehu-video.base.service"),
		grpc.WithDiscovery(r),
		grpc.WithMiddleware(
			recovery.Recovery(),
		),
	)
	if err != nil {
		panic(err)
	}
	return base.NewFileServiceClient(conn)
}

func (r *baseAdapterImpl) CreateVerificationCode(ctx context.Context, bits, expiredSeconds int64) (int64, error) {
	resp, err := r.auth.CreateVerificationCode(ctx, &base.CreateVerificationCodeReq{
		Bits:       bits,
		ExpireTime: expiredSeconds,
	})
	if err != nil {
		return 0, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return 0, err
	}
	return resp.VerificationCodeId, nil
}

func (r *baseAdapterImpl) ValidateVerificationCode(ctx context.Context, codeId int64, code string) error {
	resp, err := r.auth.ValidateVerificationCode(ctx, &base.ValidateVerificationCodeReq{
		VerificationCodeId: codeId,
		Code:               code,
	})
	if err != nil {
		return err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return err
	}
	return nil
}

func (r *baseAdapterImpl) Register(ctx context.Context, mobile, email, password string) (int64, error) {
	resp, err := r.account.Register(ctx, &base.RegisterReq{
		Mobile:   mobile,
		Email:    email,
		Password: password,
	})
	if err != nil {
		return 0, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return 0, err
	}
	return resp.AccountId, nil
}

func (r *baseAdapterImpl) CheckAccount(ctx context.Context, mobile, email, password string) (int64, error) {
	resp, err := r.account.CheckAccount(ctx, &base.CheckAccountReq{
		Mobile:   mobile,
		Email:    email,
		Password: password,
	})
	if err != nil {
		return 0, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return 0, err
	}
	return resp.AccountId, nil
}

func (r *baseAdapterImpl) PreSign4PublicUpload(ctx context.Context, hash, fileType, fileName string, size, expireSeconds int64) (int64, string, error) {
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
		return 0, "", err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return 0, "", err
	}
	return resp.FileId, resp.Url, nil
}

func (r *baseAdapterImpl) ReportPublicUploaded(ctx context.Context, fileId int64) (string, error) {
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
func (r *baseAdapterImpl) GetFileInfoById(ctx context.Context, fileId int64) (*biz.FileInfo, error) {
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
