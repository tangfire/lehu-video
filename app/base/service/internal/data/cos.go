package data

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/minio/minio-go/v7"
	"github.com/tencentyun/cos-go-sdk-v5"
	"lehu-video/app/base/service/internal/biz"
)

type cosConfig struct {
	SecretID      string
	SecretKey     string
	Region        string
	Bucket        string
	PublicBaseURL string
}

type cosRepo struct {
	client        *cos.Client
	secretID      string
	secretKey     string
	publicBaseURL string
}

func NewCOSRepoFromEnv(logger log.Logger) (biz.MinioRepo, error) {
	cfg, err := newCOSConfigFromEnv()
	if err != nil {
		return nil, err
	}
	bucketURL, err := url.Parse(fmt.Sprintf("https://%s.cos.%s.myqcloud.com", cfg.Bucket, cfg.Region))
	if err != nil {
		return nil, fmt.Errorf("build COS bucket URL: %w", err)
	}
	client := cos.NewClient(&cos.BaseURL{BucketURL: bucketURL}, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  cfg.SecretID,
			SecretKey: cfg.SecretKey,
		},
	})
	return &cosRepo{
		client:        client,
		secretID:      cfg.SecretID,
		secretKey:     cfg.SecretKey,
		publicBaseURL: cfg.PublicBaseURL,
	}, nil
}

func newCOSConfigFromEnv() (*cosConfig, error) {
	cfg := &cosConfig{
		SecretID:      strings.TrimSpace(os.Getenv("COS_SECRET_ID")),
		SecretKey:     strings.TrimSpace(os.Getenv("COS_SECRET_KEY")),
		Region:        strings.TrimSpace(os.Getenv("COS_REGION")),
		Bucket:        strings.TrimSpace(os.Getenv("COS_BUCKET")),
		PublicBaseURL: strings.TrimRight(strings.TrimSpace(os.Getenv("COS_PUBLIC_CDN_BASE_URL")), "/"),
	}
	missing := make([]string, 0, 5)
	if cfg.SecretID == "" {
		missing = append(missing, "COS_SECRET_ID")
	}
	if cfg.SecretKey == "" {
		missing = append(missing, "COS_SECRET_KEY")
	}
	if cfg.Region == "" {
		missing = append(missing, "COS_REGION")
	}
	if cfg.Bucket == "" {
		missing = append(missing, "COS_BUCKET")
	}
	if cfg.PublicBaseURL == "" {
		missing = append(missing, "COS_PUBLIC_CDN_BASE_URL")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("LEHU_STORAGE_PROVIDER=cos requires %s", strings.Join(missing, ", "))
	}
	parsed, err := url.Parse(cfg.PublicBaseURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return nil, fmt.Errorf("COS_PUBLIC_CDN_BASE_URL must be an https URL, got %q", cfg.PublicBaseURL)
	}
	return cfg, nil
}

func (r *cosRepo) PreSignGetUrl(ctx context.Context, bucketName, objectName, fileName string, expireSeconds int64) (string, error) {
	return r.GetPublicUrl(ctx, bucketName, objectName)
}

func (r *cosRepo) PreSignPutUrl(ctx context.Context, bucketName, objectName string, expireSeconds int64) (string, error) {
	expire := time.Duration(expireSeconds) * time.Second
	if expire <= 0 {
		expire = time.Hour
	}
	u, err := r.client.Object.GetPresignedURL(ctx, http.MethodPut, objectName, r.secretID, r.secretKey, expire, nil)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (r *cosRepo) CreateSlicingUpload(ctx context.Context, bucketName, objectName string, options minio.PutObjectOptions) (string, error) {
	return "", errors.New("cos provider does not support legacy multipart upload")
}

func (r *cosRepo) ListSlicingFileParts(ctx context.Context, bucketName, objectName, uploadId string, partsNum int64) (minio.ListObjectPartsResult, error) {
	return minio.ListObjectPartsResult{}, errors.New("cos provider does not support legacy multipart upload")
}

func (r *cosRepo) PreSignSlicingPutUrl(ctx context.Context, bucketName, objectName, uploadId string, parts int64) (string, error) {
	return "", errors.New("cos provider does not support legacy multipart upload")
}

func (r *cosRepo) MergeSlices(ctx context.Context, bucketName, objectName, uploadId string, parts []minio.CompletePart) error {
	return errors.New("cos provider does not support legacy multipart upload")
}

func (r *cosRepo) GetObjectHash(ctx context.Context, bucketName, objectName string) (string, error) {
	resp, err := r.client.Object.Head(ctx, objectName, nil)
	if err != nil {
		return "", err
	}
	etag := strings.Trim(resp.Header.Get("ETag"), "\"")
	if etag == "" {
		return "", errors.New("COS object ETag is empty")
	}
	return strings.ToUpper(etag), nil
}

func (r *cosRepo) GetPublicUrl(ctx context.Context, bucketName, objectName string) (string, error) {
	return joinPublicURL(r.publicBaseURL, objectName), nil
}

func joinPublicURL(baseURL, objectName string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	objectName = strings.TrimLeft(strings.TrimSpace(objectName), "/")
	if objectName == "" {
		return baseURL
	}
	parts := strings.Split(objectName, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return baseURL + "/" + strings.Join(parts, "/")
}
