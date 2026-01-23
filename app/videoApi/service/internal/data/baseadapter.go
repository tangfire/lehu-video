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

type baseAdapterRepo struct {
	account base.AccountServiceClient
	auth    base.AuthServiceClient
}

func NewBaseAdapterRepo(account base.AccountServiceClient, auth base.AuthServiceClient) biz.BaseAdapterRepo {
	return &baseAdapterRepo{
		account: account,
		auth:    auth,
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

func (r *baseAdapterRepo) CreateVerificationCode(ctx context.Context, bits, expiredSeconds int64) (int64, error) {
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

func (r *baseAdapterRepo) ValidateVerificationCode(ctx context.Context, codeId int64, code string) error {
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

func (r *baseAdapterRepo) Register(ctx context.Context, mobile, email, password string) (int64, error) {
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

func (r *baseAdapterRepo) CheckAccount(ctx context.Context, mobile, email, password string) (int64, error) {
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
