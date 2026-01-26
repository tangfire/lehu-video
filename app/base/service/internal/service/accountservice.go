package service

import (
	"context"
	"errors"
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
	// ✅ 改为Command
	cmd := &biz.RegisterCommand{
		Mobile:   req.Mobile,
		Email:    req.Email,
		Password: req.Password,
	}

	result, err := s.uc.Register(ctx, cmd)
	if err != nil {
		return &pb.RegisterResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.RegisterResp{
		Meta:      utils.GetSuccessMeta(),
		AccountId: result.AccountId,
	}, nil
}

func (s *AccountServiceService) CheckAccount(ctx context.Context, req *pb.CheckAccountReq) (*pb.CheckAccountResp, error) {
	// ✅ 改为Query
	query := &biz.CheckAccountQuery{
		AccountId: req.AccountId,
		Mobile:    req.Mobile,
		Email:     req.Email,
		Password:  req.Password,
	}

	result, err := s.uc.CheckAccount(ctx, query)
	if err != nil {
		return &pb.CheckAccountResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.CheckAccountResp{
		Meta:      utils.GetSuccessMeta(),
		AccountId: result.AccountId,
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
		return &pb.BindResp{
			Meta: utils.GetMetaWithError(errors.New("invalid voucher type")),
		}, nil
	}

	// ✅ 改为Command
	cmd := &biz.BindCommand{
		AccountId:   req.AccountId,
		VoucherType: voucherType,
		Voucher:     req.Voucher,
	}

	_, err := s.uc.Bind(ctx, cmd)
	if err != nil {
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
		return &pb.UnbindResp{
			Meta: utils.GetMetaWithError(errors.New("invalid voucher type")),
		}, nil
	}

	// ✅ 改为Command
	cmd := &biz.UnbindCommand{
		AccountId:   req.AccountId,
		VoucherType: voucherType,
	}

	_, err := s.uc.Unbind(ctx, cmd)
	if err != nil {
		return &pb.UnbindResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.UnbindResp{
		Meta: utils.GetSuccessMeta(),
	}, nil
}
