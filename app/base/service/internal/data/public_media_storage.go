package data

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/minio/minio-go/v7"
	"lehu-video/app/base/service/internal/biz"
)

type publicMediaStorageRepo struct {
	publicMedia biz.MinioRepo
	fallback    biz.MinioRepo
}

func NewPublicMediaStorageRepo(publicMedia, fallback biz.MinioRepo) biz.MinioRepo {
	return &publicMediaStorageRepo{publicMedia: publicMedia, fallback: fallback}
}

func (r *publicMediaStorageRepo) target(objectName string) biz.MinioRepo {
	if isPublicMediaObject(objectName) {
		return r.publicMedia
	}
	return r.fallback
}

func isPublicMediaObject(objectName string) bool {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(objectName)), ".")
	switch ext {
	case "jpg", "jpeg", "png", "webp", "gif", "mp4", "mov":
		return true
	default:
		return false
	}
}

func (r *publicMediaStorageRepo) PreSignGetUrl(ctx context.Context, bucketName, objectName, fileName string, expireSeconds int64) (string, error) {
	return r.target(objectName).PreSignGetUrl(ctx, bucketName, objectName, fileName, expireSeconds)
}

func (r *publicMediaStorageRepo) PreSignPutUrl(ctx context.Context, bucketName, objectName string, expireSeconds int64) (string, error) {
	return r.target(objectName).PreSignPutUrl(ctx, bucketName, objectName, expireSeconds)
}

func (r *publicMediaStorageRepo) CreateSlicingUpload(ctx context.Context, bucketName, objectName string, options minio.PutObjectOptions) (string, error) {
	return r.target(objectName).CreateSlicingUpload(ctx, bucketName, objectName, options)
}

func (r *publicMediaStorageRepo) ListSlicingFileParts(ctx context.Context, bucketName, objectName, uploadId string, partsNum int64) (minio.ListObjectPartsResult, error) {
	return r.target(objectName).ListSlicingFileParts(ctx, bucketName, objectName, uploadId, partsNum)
}

func (r *publicMediaStorageRepo) PreSignSlicingPutUrl(ctx context.Context, bucketName, objectName, uploadId string, parts int64) (string, error) {
	return r.target(objectName).PreSignSlicingPutUrl(ctx, bucketName, objectName, uploadId, parts)
}

func (r *publicMediaStorageRepo) MergeSlices(ctx context.Context, bucketName, objectName, uploadId string, parts []minio.CompletePart) error {
	return r.target(objectName).MergeSlices(ctx, bucketName, objectName, uploadId, parts)
}

func (r *publicMediaStorageRepo) GetObjectHash(ctx context.Context, bucketName, objectName string) (string, error) {
	return r.target(objectName).GetObjectHash(ctx, bucketName, objectName)
}

func (r *publicMediaStorageRepo) GetPublicUrl(ctx context.Context, bucketName, objectName string) (string, error) {
	return r.target(objectName).GetPublicUrl(ctx, bucketName, objectName)
}
