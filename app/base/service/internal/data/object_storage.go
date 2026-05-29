package data

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
	"lehu-video/app/base/service/internal/biz"
	"lehu-video/app/base/service/internal/conf"
)

const (
	storageProviderMinio = "minio"
	storageProviderCOS   = "cos"
)

func currentStorageProvider() string {
	provider := strings.TrimSpace(strings.ToLower(os.Getenv("LEHU_STORAGE_PROVIDER")))
	if provider == "" {
		return storageProviderMinio
	}
	return provider
}

func NewObjectStorageRepo(conf *conf.Data, logger log.Logger) (biz.MinioRepo, error) {
	provider := currentStorageProvider()
	helper := log.NewHelper(logger)
	switch provider {
	case storageProviderMinio:
		helper.Infof("object storage provider: %s", provider)
		core := NewMinioCore(conf)
		return NewMinioRepo(conf, core), nil
	case storageProviderCOS:
		helper.Infof("object storage provider: %s public-media-only=true", provider)
		cosRepo, err := NewCOSRepoFromEnv(logger)
		if err != nil {
			return nil, err
		}
		core := NewMinioCore(conf)
		minioRepo := NewMinioRepo(conf, core)
		return NewPublicMediaStorageRepo(cosRepo, minioRepo), nil
	default:
		return nil, fmt.Errorf("unsupported LEHU_STORAGE_PROVIDER %q, expected minio or cos", provider)
	}
}
