package biz

import (
	"context"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	"github.com/go-kratos/kratos/v2/transport"
	jwtv5 "github.com/golang-jwt/jwt/v5"
	pb "lehu-video/api/videoApi/service/v1"
	"lehu-video/app/videoApi/service/internal/pkg/utils/claims"
)

type UserUsecase struct {
	base *BaseAdapter
	core *CoreAdapter
	log  *log.Helper
}

func NewUserUsecase(base *BaseAdapter, core *CoreAdapter, logger log.Logger) *UserUsecase {
	return &UserUsecase{base: base, core: core, log: log.NewHelper(logger)}
}

func (uc *UserUsecase) GetVerificationCode(ctx context.Context, req *pb.GetVerificationCodeReq) (*pb.GetVerificationCodeResp, error) {
	codeId, err := uc.base.repo.CreateVerificationCode(ctx, 6, 60*10)
	if err != nil {
		return nil, err
	}
	return &pb.GetVerificationCodeResp{
		CodeId: codeId,
	}, nil
}
func (uc *UserUsecase) Register(ctx context.Context, req *pb.RegisterReq) (*pb.RegisterResp, error) {
	err := uc.base.repo.ValidateVerificationCode(ctx, req.CodeId, req.Code)
	if err != nil {
		return nil, err
	}

	accountId, err := uc.base.repo.Register(ctx, req.Mobile, req.Email, req.Password)
	if err != nil {
		return nil, err
	}

	// todo 这里好像有个分布式事务？？？ 调用core服务创建基本用户信息, 需要处理 register 成功，但是创建用户信息失败
	userId, err := uc.core.repo.CreateUser(ctx, req.Mobile, req.Email, accountId)
	if err != nil {
		return nil, err
	}
	return &pb.RegisterResp{
		UserId: userId,
	}, nil
}
func (uc *UserUsecase) Login(ctx context.Context, req *pb.LoginReq) (*pb.LoginResp, error) {
	accountId, err := uc.base.repo.CheckAccount(ctx, req.Mobile, req.Email, req.Password)
	if err != nil {
		return nil, err
	}
	user, err := uc.core.repo.GetUserInfo(ctx, 0, accountId)
	if err != nil {
		return nil, err
	}
	token, err := uc.setToken2Header(ctx, claims.New(user.Id))
	if err != nil {
		return nil, err
	}

	retUser := &pb.User{
		Id:              user.Id,
		Name:            user.Name,
		Avatar:          user.Avatar,
		BackgroundImage: user.BackgroundImage,
		Mobile:          user.Mobile,
		Email:           user.Email,
	}

	return &pb.LoginResp{
		Token: token,
		User:  retUser,
	}, nil
}

func (uc *UserUsecase) setToken2Header(ctx context.Context, claim *claims.Claims) (string, error) {
	token := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claim)
	tokenString, err := token.SignedString([]byte("token"))
	if err != nil {
		return "", err
	}

	if header, ok := transport.FromServerContext(ctx); ok {
		header.ReplyHeader().Set("Authorization", "Bearer "+tokenString)
		return tokenString, nil
	}

	return "", jwt.ErrWrongContext
}

func (uc *UserUsecase) GetUserInfo(ctx context.Context, req *pb.GetUserInfoReq) (*pb.GetUserInfoResp, error) {
	return &pb.GetUserInfoResp{}, nil
}
func (uc *UserUsecase) UpdateUserInfo(ctx context.Context, req *pb.UpdateUserInfoReq) (*pb.UpdateUserInfoResp, error) {
	return &pb.UpdateUserInfoResp{}, nil
}
func (uc *UserUsecase) BindUserVoucher(ctx context.Context, req *pb.BindUserVoucherReq) (*pb.BindUserVoucherResp, error) {
	return &pb.BindUserVoucherResp{}, nil
}
func (uc *UserUsecase) UnbindUserVoucher(ctx context.Context, req *pb.UnbindUserVoucherReq) (*pb.UnbindUserVoucherResp, error) {
	return &pb.UnbindUserVoucherResp{}, nil
}
