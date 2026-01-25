package data

import (
	"context"
	"errors"
	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
	"lehu-video/app/base/service/internal/biz"
	"lehu-video/app/base/service/internal/data/model"
	"time"
)

type fileRepo struct {
	data *Data
	log  *log.Helper
}

func NewFileRepo(data *Data, logger log.Logger) *fileRepo {
	return &fileRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func NewBizFileRepo(data *Data, logger log.Logger) biz.FileRepo {
	return &fileRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *fileRepo) GetUploadedFileById(ctx context.Context, tableName string, id int64) (bool, *biz.File, error) {
	file := model.File{}
	err := r.data.db.Table(tableName).Where("id = ? and uploaded = ?", id, true).First(&file).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, err
	}
	retFile := &biz.File{
		Id:         file.Id,
		DomainName: file.DomainName,
		BizName:    file.BizName,
		Hash:       file.Hash,
		FileType:   file.FileType,
		FileSize:   file.FileSize,
		Uploaded:   file.Uploaded,
		CreatedAt:  file.CreatedAt,
		UpdatedAt:  file.UpdatedAt,
	}
	return true, retFile, nil
}

func (r *fileRepo) GetUploadedFileByHash(ctx context.Context, tableName string, hash string) (bool, *biz.File, error) {
	file := model.File{}
	err := r.data.db.Table(tableName).Where("hash = ? and uploaded = ?", hash, true).First(&file).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, err
	}
	retFile := &biz.File{
		Id:         file.Id,
		DomainName: file.DomainName,
		BizName:    file.BizName,
		Hash:       file.Hash,
		FileType:   file.FileType,
		FileSize:   file.FileSize,
		Uploaded:   file.Uploaded,
		CreatedAt:  file.CreatedAt,
		UpdatedAt:  file.UpdatedAt,
	}
	return true, retFile, nil

}
func (r *fileRepo) GetFileById(ctx context.Context, tableName string, id int64) (bool, *biz.File, error) {
	file := model.File{}
	err := r.data.db.Table(tableName).Where("id = ?", id).First(&file).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, err
	}
	retFile := &biz.File{
		Id:         file.Id,
		DomainName: file.DomainName,
		BizName:    file.BizName,
		Hash:       file.Hash,
		FileType:   file.FileType,
		FileSize:   file.FileSize,
		Uploaded:   file.Uploaded,
		CreatedAt:  file.CreatedAt,
		UpdatedAt:  file.UpdatedAt,
	}
	return true, retFile, nil

}
func (r *fileRepo) CreateFile(ctx context.Context, tx *gorm.DB, tableName string, retFile *biz.File) error {
	file := model.File{
		Id:         retFile.Id,
		DomainName: retFile.DomainName,
		BizName:    retFile.BizName,
		Hash:       retFile.Hash,
		FileSize:   retFile.FileSize,
		FileType:   retFile.FileType,
		Uploaded:   retFile.Uploaded,
		IsDeleted:  false,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	err := tx.Table(tableName).Create(&file).Error
	if err != nil {
		return err
	}
	return nil
}
func (r *fileRepo) UpdateFile(ctx context.Context, tx *gorm.DB, tableName string, retFile *biz.File) error {
	file := model.File{
		Id:         retFile.Id,
		DomainName: retFile.DomainName,
		BizName:    retFile.BizName,
		Hash:       retFile.Hash,
		FileSize:   retFile.FileSize,
		FileType:   retFile.FileType,
		Uploaded:   retFile.Uploaded,
		CreatedAt:  retFile.CreatedAt,
		UpdatedAt:  time.Now(),
	}
	err := tx.Table(tableName).Where("id = ? ", file.Id).Updates(&file).Error
	if err != nil {
		return err
	}
	return nil
}
func (r *fileRepo) DeleteFile(ctx context.Context, tx *gorm.DB, tableName string, id int64) error {
	err := tx.Table(tableName).Where("id = ? ", id).UpdateColumns(map[string]interface{}{
		"is_deleted": true,
	}).Error
	if err != nil {
		return err
	}
	return nil
}
