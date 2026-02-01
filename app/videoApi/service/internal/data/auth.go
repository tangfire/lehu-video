package data

import (
	"context"
	base "lehu-video/api/base/service/v1"
	"lehu-video/app/videoApi/service/internal/pkg/utils/respcheck"
)

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
