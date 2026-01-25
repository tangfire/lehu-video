package data

import (
	"context"
	"fmt"
	"gorm.io/gorm"
	"lehu-video/app/base/service/internal/biz"
	"lehu-video/app/base/service/internal/conf"
	"lehu-video/app/base/service/internal/data/model"
)

type fileRepoHelperImpl struct {
	fr  *fileRepo
	cfg *FileShardingConfig
}

func NewFileRepoHelper(cfg *FileShardingConfig, fr *fileRepo) biz.FileRepoHelper {
	return &fileRepoHelperImpl{
		cfg: cfg,
		fr:  fr,
	}
}

type FileShardingConfig struct {
	DbShardingConfig map[string]*conf.DataSetting_DBShardingConfigItem
	DbShardingTables map[string]*conf.DataSetting_DBShardingTable
}

func NewFileShardingConfig(conf *conf.DataSetting) *FileShardingConfig {
	return &FileShardingConfig{
		DbShardingConfig: conf.DbShardingConfig,
		DbShardingTables: conf.DbShardingTables,
	}
}

func (cfg *FileShardingConfig) GetShardingNumber(fileName, domainName, bizName string) int64 {
	key := fmt.Sprintf("%s_%s_%s", fileName, domainName, bizName)
	if num, ok := cfg.DbShardingConfig[key]; ok {
		return int64(num.ShardingNumber)
	}

	return 1
}

func (r *fileRepoHelperImpl) GetTableNameById(file *biz.File) string {
	shardingNum := r.cfg.GetShardingNumber(model.TableNameFile, file.DomainName, file.BizName)

	if shardingNum > 0 {
		return fmt.Sprintf("%s_%s_%s_id_%d", model.TableNameFile, file.DomainName, file.BizName, file.Id%shardingNum)
	}

	return fmt.Sprintf("%s_%s", file.DomainName, file.BizName)
}

func (r *fileRepoHelperImpl) GetTableNameByHash(file *biz.File) string {
	shardingNum := r.cfg.GetShardingNumber(model.TableNameFile, file.DomainName, file.BizName)

	if shardingNum > 0 {
		return fmt.Sprintf("%s_%s_%s_hash_%d", model.TableNameFile, file.DomainName, file.BizName, int64([]byte(file.Hash)[0])%shardingNum)
	}

	return fmt.Sprintf("%s_%s", file.DomainName, file.BizName)
}

func (r *fileRepoHelperImpl) doWithTx(
	ctx context.Context,
	file *biz.File,
	opName string,
	fn func(ctx context.Context, tx *gorm.DB, tableName string, file *biz.File) error,
) (err error) {

	if file.Id == 0 || file.Hash == "" {
		return fmt.Errorf("file can't %s without id or hash", opName)
	}

	tx := r.fr.data.Begin()
	if tx == nil {
		return fmt.Errorf("failed to begin tx when %s", opName)
	}

	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
		if commitErr := tx.Commit().Error; commitErr != nil {
			err = commitErr
		}
	}()

	tableNames := []string{
		r.GetTableNameById(file),
		r.GetTableNameByHash(file),
	}

	for _, tableName := range tableNames {
		if err = fn(ctx, tx, tableName, file); err != nil {
			return err
		}
	}

	return nil
}

func (r *fileRepoHelperImpl) AddFile(ctx context.Context, file *biz.File) error {
	return r.doWithTx(ctx, file, "add file",
		func(ctx context.Context, tx *gorm.DB, tableName string, file *biz.File) error {
			return r.fr.CreateFile(ctx, tx, tableName, file)
		})
}

func (r *fileRepoHelperImpl) RemoveFile(ctx context.Context, file *biz.File) error {
	return r.doWithTx(ctx, file, "remove file",
		func(ctx context.Context, tx *gorm.DB, tableName string, file *biz.File) error {
			return r.fr.DeleteFile(ctx, tx, tableName, file.Id)
		})
}

func (r *fileRepoHelperImpl) UpdateFile(ctx context.Context, file *biz.File) error {
	return r.doWithTx(ctx, file, "update file",
		func(ctx context.Context, tx *gorm.DB, tableName string, file *biz.File) error {
			return r.fr.UpdateFile(ctx, tx, tableName, file)
		})
}
