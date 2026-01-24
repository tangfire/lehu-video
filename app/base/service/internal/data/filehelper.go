package data

import (
	"fmt"
	"lehu-video/app/base/service/internal/biz"
	"lehu-video/app/base/service/internal/conf"
	"lehu-video/app/base/service/internal/data/model"
)

type FileRepoHelper struct {
	cfg *FileShardingConfig
}

func NewFileRepoHelper(cfg *FileShardingConfig) *FileRepoHelper {
	return &FileRepoHelper{cfg: cfg}
}

func (h *FileRepoHelper) GetTableNameById(file *biz.File) string {
	shardingNum := h.cfg.GetShardingNumber(model.TableNameFile, file.DomainName, file.BizName)

	if shardingNum > 0 {
		return fmt.Sprintf("%s_%s_%s_id_%d", model.TableNameFile, file.DomainName, file.BizName, file.Id%shardingNum)
	}

	return fmt.Sprintf("%s_%s", file.DomainName, file.BizName)
}

func (h *FileRepoHelper) GetTableNameByHash(file *biz.File) string {
	shardingNum := h.cfg.GetShardingNumber(model.TableNameFile, file.DomainName, file.BizName)

	if shardingNum > 0 {
		return fmt.Sprintf("%s_%s_%s_hash_%d", model.TableNameFile, file.DomainName, file.BizName, int64([]byte(file.Hash)[0])%shardingNum)
	}

	return fmt.Sprintf("%s_%s", file.DomainName, file.BizName)
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
