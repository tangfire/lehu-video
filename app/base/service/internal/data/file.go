package data

import (
	"context"
	"errors"
	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
	"lehu-video/app/base/service/internal/biz"
	"lehu-video/app/base/service/internal/data/model"
)

type fileRepo struct {
	data *Data
	fph  *FileRepoHelper
	log  *log.Helper
}

func NewFileRepo(data *Data, fph *FileRepoHelper, logger log.Logger) biz.FileRepo {
	return &fileRepo{
		data: data,
		fph:  fph,
		log:  log.NewHelper(logger),
	}
}

func (r *fileRepo) CheckFileExistedAndGetFile(ctx context.Context, fileCtx *biz.File) (int64, bool, error) {
	hash := r.fph.GetTableNameByHash(fileCtx)
	file := model.File{}
	err := r.data.db.Table(model.File{}.TableName()).
		Where("hash = ? and uploaded = ?", hash, true).First(&file).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return file.Id, true, nil
}
