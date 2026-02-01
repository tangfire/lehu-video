package data

import (
	"context"
	base "lehu-video/api/base/service/v1"
	"lehu-video/app/videoApi/service/internal/pkg/utils/respcheck"
)

func (r *baseAdapterImpl) Register(ctx context.Context, mobile, email, password string) (string, error) {
	resp, err := r.account.Register(ctx, &base.RegisterReq{
		Mobile:   mobile,
		Email:    email,
		Password: password,
	})
	if err != nil {
		return "0", err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return "0", err
	}
	return resp.AccountId, nil
}

func (r *baseAdapterImpl) CheckAccount(ctx context.Context, mobile, email, password string) (string, error) {
	resp, err := r.account.CheckAccount(ctx, &base.CheckAccountReq{
		Mobile:   mobile,
		Email:    email,
		Password: password,
	})
	if err != nil {
		return "0", err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return "0", err
	}
	return resp.AccountId, nil
}
