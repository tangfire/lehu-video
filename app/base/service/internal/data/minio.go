package data

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"lehu-video/app/base/service/internal/biz"
	"lehu-video/app/base/service/internal/conf"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type minioRepo struct {
	client *minio.Client
	core   *minio.Core
}

func NewMinioRepo(client *minio.Client, core *minio.Core) biz.MinioRepo {
	return &minioRepo{
		client: client,
		core:   core,
	}
}

/*
=====================
 MinIO Client 初始化
=====================
*/

func NewMinioClient(conf *conf.Data) *minio.Client {
	endPoint := fmt.Sprintf("%s:%s", conf.Minio.Host, conf.Minio.Port)

	// 添加时间差容忍配置
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	client, err := minio.New(endPoint, &minio.Options{
		Creds:     credentials.NewStaticV4(conf.Minio.AccessKey, conf.Minio.SecretKey, ""),
		Secure:    false,
		Transport: transport,
	})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	return client
}

/*
=====================
 MinIO Core 初始化
=====================
*/

func NewMinioCore(conf *conf.Data) *minio.Core {
	endPoint := fmt.Sprintf("%s:%s", conf.Minio.Host, conf.Minio.Port)

	core, err := minio.NewCore(endPoint, &minio.Options{
		Creds:  credentials.NewStaticV4(conf.Minio.AccessKey, conf.Minio.SecretKey, ""),
		Secure: false,
	})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	return core
}

/*
=====================
 预签名下载
=====================
*/

func (r *minioRepo) PreSignGetUrl(
	ctx context.Context,
	bucketName,
	objectName,
	fileName string,
	expireSeconds int64,
) (string, error) {

	reqParams := make(url.Values)

	if fileName != "" {
		reqParams.Set(
			"response-content-disposition",
			"attachment; filename="+fileName,
		)
	}

	u, err := r.client.PresignedGetObject(
		ctx,
		bucketName,
		objectName,
		time.Duration(expireSeconds)*time.Second,
		reqParams,
	)
	if err != nil {
		return "", err
	}

	return u.String(), nil
}

/*
=====================
 预签名普通上传
=====================
*/

func (r *minioRepo) PreSignPutUrl(
	ctx context.Context,
	bucketName,
	objectName string,
	expireSeconds int64,
) (string, error) {

	u, err := r.client.PresignedPutObject(
		ctx,
		bucketName,
		objectName,
		time.Duration(expireSeconds)*time.Second,
	)
	if err != nil {
		return "", err
	}

	return u.String(), nil
}

/*
=====================
 分片上传 - 创建
=====================
*/

func (r *minioRepo) CreateSlicingUpload(
	ctx context.Context,
	bucketName,
	objectName string,
	options minio.PutObjectOptions,
) (uploadId string, err error) {

	return r.core.NewMultipartUpload(ctx, bucketName, objectName, options)
}

/*
=====================
 分片上传 - 查询已上传分片
=====================
*/

func (r *minioRepo) ListSlicingFileParts(
	ctx context.Context,
	bucketName,
	objectName,
	uploadId string,
	partsNum int64,
) (minio.ListObjectPartsResult, error) {

	var nextPartNumberMarker int

	return r.core.ListObjectParts(
		ctx,
		bucketName,
		objectName,
		uploadId,
		nextPartNumberMarker,
		int(partsNum)+1,
	)
}

/*
=====================
 分片上传 - 预签名某个分片
=====================
*/

func (r *minioRepo) PreSignSlicingPutUrl(
	ctx context.Context,
	bucketName,
	objectName,
	uploadId string,
	partNumber int64,
) (string, error) {

	params := url.Values{
		"uploadId":   {uploadId},
		"partNumber": {strconv.FormatInt(partNumber, 10)},
	}

	u, err := r.client.Presign(
		ctx,
		http.MethodPut,
		bucketName,
		objectName,
		time.Hour,
		params,
	)
	if err != nil {
		return "", err
	}

	return u.String(), nil
}

/*
=====================
 分片上传 - 合并
=====================
*/

func (r *minioRepo) MergeSlices(
	ctx context.Context,
	bucketName,
	objectName,
	uploadId string,
	parts []minio.CompletePart,
) error {

	_, err := r.core.CompleteMultipartUpload(
		ctx,
		bucketName,
		objectName,
		uploadId,
		parts,
		minio.PutObjectOptions{},
	)
	return err
}

/*
=====================
 获取对象 ETag
=====================
*/

func (r *minioRepo) GetObjectHash(
	ctx context.Context,
	bucketName,
	objectName string,
) (string, error) {

	stat, err := r.client.StatObject(
		ctx,
		bucketName,
		objectName,
		minio.StatObjectOptions{},
	)
	if err != nil {
		return "", err
	}

	return strings.ToUpper(stat.ETag), nil
}
