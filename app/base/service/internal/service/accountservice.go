package service

import (
	"context"
	"errors" // ✅ 补充缺失的errors导入
	pb "lehu-video/api/base/service/v1"
	"lehu-video/app/base/service/internal/biz"
	"lehu-video/app/base/service/internal/pkg/utils"
)

type AccountServiceService struct {
	pb.UnimplementedAccountServiceServer
	uc *biz.AccountUsecase
}

func NewAccountServiceService(uc *biz.AccountUsecase) *AccountServiceService {
	return &AccountServiceService{uc: uc}
}

func (s *AccountServiceService) Register(ctx context.Context, req *pb.RegisterReq) (*pb.RegisterResp, error) {
	bizReq := &biz.RegisterRequest{
		Mobile:   req.Mobile,
		Email:    req.Email,
		Password: req.Password,
	}

	resp, err := s.uc.Register(ctx, bizReq)
	if err != nil {
		// ✅ 重要：返回带有错误信息的Meta，而不是直接返回err
		return &pb.RegisterResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.RegisterResp{
		Meta:      utils.GetSuccessMeta(),
		AccountId: resp.AccountId,
	}, nil
}

func (s *AccountServiceService) CheckAccount(ctx context.Context, req *pb.CheckAccountReq) (*pb.CheckAccountResp, error) {
	bizReq := &biz.CheckAccountRequest{
		AccountId: req.AccountId,
		Mobile:    req.Mobile,
		Email:     req.Email,
		Password:  req.Password,
	}

	resp, err := s.uc.CheckAccount(ctx, bizReq)
	if err != nil {
		// ✅ 重要：返回带有错误信息的Meta，而不是直接返回err
		return &pb.CheckAccountResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.CheckAccountResp{
		Meta:      utils.GetSuccessMeta(),
		AccountId: resp.AccountId,
	}, nil
}

func (s *AccountServiceService) Bind(ctx context.Context, req *pb.BindReq) (*pb.BindResp, error) {
	var voucherType biz.VoucherType
	switch req.VoucherType {
	case pb.VoucherType_VOUCHER_EMAIL:
		voucherType = biz.VoucherTypeEmail
	case pb.VoucherType_VOUCHER_PHONE:
		voucherType = biz.VoucherTypePhone
	default:
		// ✅ 返回错误Meta而不是直接返回error
		return &pb.BindResp{
			Meta: utils.GetMetaWithError(errors.New("invalid voucher type")),
		}, nil
	}

	bizReq := &biz.BindRequest{
		AccountId:   req.AccountId,
		VoucherType: voucherType,
		Voucher:     req.Voucher,
	}

	_, err := s.uc.Bind(ctx, bizReq)
	if err != nil {
		// ✅ 返回错误Meta而不是直接返回error
		return &pb.BindResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.BindResp{
		Meta: utils.GetSuccessMeta(),
	}, nil
}

func (s *AccountServiceService) Unbind(ctx context.Context, req *pb.UnbindReq) (*pb.UnbindResp, error) {
	var voucherType biz.VoucherType
	switch req.VoucherType {
	case pb.VoucherType_VOUCHER_EMAIL:
		voucherType = biz.VoucherTypeEmail
	case pb.VoucherType_VOUCHER_PHONE:
		voucherType = biz.VoucherTypePhone
	default:
		// ✅ 返回错误Meta而不是直接返回error
		return &pb.UnbindResp{
			Meta: utils.GetMetaWithError(errors.New("invalid voucher type")),
		}, nil
	}

	bizReq := &biz.UnbindRequest{
		AccountId:   req.AccountId,
		VoucherType: voucherType,
	}

	_, err := s.uc.Unbind(ctx, bizReq)
	if err != nil {
		// ✅ 返回错误Meta而不是直接返回error
		return &pb.UnbindResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.UnbindResp{
		Meta: utils.GetSuccessMeta(),
	}, nil
}
