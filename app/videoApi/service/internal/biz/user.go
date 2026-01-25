package biz

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	"github.com/go-kratos/kratos/v2/transport"
	jwtv5 "github.com/golang-jwt/jwt/v5"
	"lehu-video/app/videoApi/service/internal/pkg/utils/claims"
)

type RegisterReq struct {
	Mobile   string
	Email    string
	Password string
	CodeId   int64
	Code     string
}

type RegisterResp struct {
	UserId int64
}

type LoginReq struct {
	Mobile   string
	Email    string
	Password string
}

type LoginResp struct {
	Token string
	User  *UserInfo
}

type GetUserInfoReq struct {
	UserId int64
}

type GetUserInfoResp struct {
	User *UserInfo
}

type UpdateUserInfoReq struct {
	UserId          int64
	Name            string
	Avatar          string
	BackgroundImage string
	Signature       string
}

type UpdateUserInfoResp struct {
	// 更新成功不需要额外数据
}

type BindUserVoucherReq struct {
	UserId      int64
	VoucherType string // email or phone
	Voucher     string // 具体的邮箱或手机号
	CodeId      int64
	Code        string
}

type BindUserVoucherResp struct{}

type UnbindUserVoucherReq struct {
	UserId      int64
	VoucherType string // email or phone
}

type UnbindUserVoucherResp struct{}

// 用户信息结构体
type UserInfo struct {
	Id              int64
	Name            string
	Avatar          string
	BackgroundImage string
	Mobile          string
	Email           string
}

type UserUsecase struct {
	base BaseAdapter
	core CoreAdapter
	log  *log.Helper
}

func NewUserUsecase(base BaseAdapter, core CoreAdapter, logger log.Logger) *UserUsecase {
	return &UserUsecase{base: base, core: core, log: log.NewHelper(logger)}
}

func (uc *UserUsecase) GetVerificationCode(ctx context.Context) (int64, error) {
	// 默认生成6位数字验证码，10分钟过期
	codeId, err := uc.base.CreateVerificationCode(ctx, 6, 60*10)
	if err != nil {
		return 0, err
	}
	return codeId, nil
}

func (uc *UserUsecase) Register(ctx context.Context, req *RegisterReq) (*RegisterResp, error) {
	// 1. 验证验证码
	err := uc.base.ValidateVerificationCode(ctx, req.CodeId, req.Code)
	if err != nil {
		return nil, err
	}

	// 2. 注册账户
	accountId, err := uc.base.Register(ctx, req.Mobile, req.Email, req.Password)
	if err != nil {
		return nil, err
	}

	// 3. 创建用户信息
	// TODO: 这里存在分布式事务问题，需要处理注册成功但创建用户失败的情况
	// 可以考虑引入Saga模式或本地消息表等方式解决
	userId, err := uc.core.CreateUser(ctx, req.Mobile, req.Email, accountId)
	if err != nil {
		// 如果创建用户失败，可能需要回滚账户注册
		// 这里需要根据具体业务需求处理
		uc.log.Error("注册成功但创建用户失败",
			"accountId", accountId,
			"error", err,
			"mobile", req.Mobile,
			"email", req.Email)
		return nil, err
	}

	return &RegisterResp{
		UserId: userId,
	}, nil
}

func (uc *UserUsecase) Login(ctx context.Context, req *LoginReq) (*LoginResp, error) {
	// 1. 验证账户
	accountId, err := uc.base.CheckAccount(ctx, req.Mobile, req.Email, req.Password)
	if err != nil {
		return nil, err
	}

	// 2. 获取用户信息
	user, err := uc.core.GetUserInfo(ctx, 0, accountId)
	if err != nil {
		return nil, err
	}

	// 3. 生成token
	token, err := uc.setToken2Header(ctx, claims.New(user.Id))
	fmt.Println("token = " + token)
	if err != nil {
		return nil, err
	}

	return &LoginResp{
		Token: token,
		User:  user,
	}, nil
}

func (uc *UserUsecase) setToken2Header(ctx context.Context, claim *claims.Claims) (string, error) {
	token := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claim)
	tokenString, err := token.SignedString([]byte("fireshine"))
	if err != nil {
		return "", err
	}

	if header, ok := transport.FromServerContext(ctx); ok {
		header.ReplyHeader().Set("Authorization", "Bearer "+tokenString)
		return tokenString, nil
	}

	return "", jwt.ErrWrongContext
}

func (uc *UserUsecase) GetUserInfo(ctx context.Context, req *GetUserInfoReq) (*GetUserInfoResp, error) {
	// TODO: 这里需要获取当前登录用户的ID
	// 通常从context中获取jwt claims
	// 暂时返回未实现错误
	return nil, errors.New("not implemented")
}

func (uc *UserUsecase) UpdateUserInfo(ctx context.Context, req *UpdateUserInfoReq) (*UpdateUserInfoResp, error) {
	// 调用核心服务更新用户信息
	err := uc.core.UpdateUserInfo(ctx, req.UserId, req.Name, req.Avatar, req.BackgroundImage, req.Signature)
	if err != nil {
		return nil, err
	}

	return &UpdateUserInfoResp{}, nil
}

func (uc *UserUsecase) BindUserVoucher(ctx context.Context, req *BindUserVoucherReq) (*BindUserVoucherResp, error) {
	// 1. 验证验证码
	err := uc.base.ValidateVerificationCode(ctx, req.CodeId, req.Code)
	if err != nil {
		return nil, err
	}

	// 2. 调用基础服务绑定凭证
	// TODO: 需要实现基础服务的绑定接口
	return nil, errors.New("not implemented")
}

func (uc *UserUsecase) UnbindUserVoucher(ctx context.Context, req *UnbindUserVoucherReq) (*UnbindUserVoucherResp, error) {
	// 调用基础服务解绑凭证
	// TODO: 需要实现基础服务的解绑接口
	return nil, errors.New("not implemented")
}
