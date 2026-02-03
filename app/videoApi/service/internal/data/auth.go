package data

import (
	"context"
	"fmt"
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

func (r *baseAdapterImpl) BindVoucher(ctx context.Context, userID, voucherType, voucher string) error {
	var vt base.VoucherType
	switch voucherType {
	case "phone":
		vt = base.VoucherType_VOUCHER_PHONE
	case "email":
		vt = base.VoucherType_VOUCHER_EMAIL
	default:
		return fmt.Errorf("invalid voucher type: %s", voucherType)
	}

	resp, err := r.account.Bind(ctx, &base.BindReq{
		AccountId:   userID,
		VoucherType: vt,
		Voucher:     voucher,
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

func (r *baseAdapterImpl) UnbindVoucher(ctx context.Context, userID, voucherType string) error {
	var vt base.VoucherType
	switch voucherType {
	case "phone":
		vt = base.VoucherType_VOUCHER_PHONE
	case "email":
		vt = base.VoucherType_VOUCHER_EMAIL
	default:
		return fmt.Errorf("invalid voucher type: %s", voucherType)
	}

	resp, err := r.account.Unbind(ctx, &base.UnbindReq{
		AccountId:   userID,
		VoucherType: vt,
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
