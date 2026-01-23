package data

import (
	"context"
	"fmt"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"lehu-video/app/base/service/internal/biz"
	"lehu-video/app/base/service/internal/pkg/utils"
	"time"
)

type authRepo struct {
	data *Data
	log  *log.Helper
}

func NewAuthRepo(data *Data, logger log.Logger) biz.AuthRepo {
	return &authRepo{data: data, log: log.NewHelper(logger)}
}

func (r *authRepo) GetVerificationKey(id int64) string {
	return fmt.Sprintf("verification_code_id_%d", id)
}

func (r *authRepo) CreateVerificationCode(ctx context.Context, bits, expireTime int64) (*biz.VerificationCode, error) {
	code := utils.UuCode(bits)
	id := int64(uuid.New().ID())
	err := r.data.rds.Set(ctx, r.GetVerificationKey(id), code, time.Duration(expireTime)*time.Second).Err()
	if err != nil {
		return nil, err
	}
	return biz.NewVerificationCode(id, code), nil
}

func (r *authRepo) GetVerificationCode(ctx context.Context, id int64) (*biz.VerificationCode, error) {
	code := r.data.rds.Get(ctx, r.GetVerificationKey(id))
	return biz.NewVerificationCode(id, code.Val()), nil
}

func (r *authRepo) DelVerificationCode(ctx context.Context, id int64) error {
	err := r.data.rds.Del(ctx, r.GetVerificationKey(id)).Err()
	if err != nil {
		return err
	}
	return nil
}
