package biz

import (
	"context"
)

type BaseAdapterRepo interface {
	CreateVerificationCode(ctx context.Context, bits, expiredSeconds int64) (int64, error)
	ValidateVerificationCode(ctx context.Context, codeId int64, code string) error
	Register(ctx context.Context, mobile, email, password string) (int64, error)
	CheckAccount(ctx context.Context, mobile, email, password string) (int64, error)
}

type BaseAdapter struct {
	repo BaseAdapterRepo
}

func NewBaseAdapter(repo BaseAdapterRepo) *BaseAdapter {
	return &BaseAdapter{repo: repo}
}
