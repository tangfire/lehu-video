package data

import (
	"context"
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
	core *minio.Core
}

func NewMinioRepo(core *minio.Core) biz.MinioRepo {
	return &minioRepo{
		core: core,
	}
}

/*
=====================
 MinIO Client 初始化
=====================
*/

//func NewMinioClient(conf *conf.Data) *minio.Client {
//	endPoint := fmt.Sprintf("%s:%s", conf.Minio.Host, conf.Minio.Port)
//
//	// 添加时间差容忍配置
//	transport := &http.Transport{
//		TLSClientConfig: &tls.Config{
//			InsecureSkipVerify: true,
//		},
//	}
//
//	client, err := minio.New(endPoint, &minio.Options{
//		Creds:     credentials.NewStaticV4(conf.Minio.AccessKey, conf.Minio.SecretKey, ""),
//		Secure:    false,
//		Transport: transport,
//	})
//	if err != nil {
//		fmt.Println(err)
//		os.Exit(1)
//	}
//
//	return client
//}

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

	// 获取扩展名
	ext := r.getFileExtension(objectName)

	// 设置 Content-Type
	if ext != "" {
		contentType := r.getContentType(ext)
		reqParams.Set("response-content-type", contentType)
	}

	// 设置 Content-Disposition
	if fileName != "" {
		// 如果传入了 fileName，使用传入的
		reqParams.Set(
			"response-content-disposition",
			"inline; filename=\""+fileName+"\"",
		)
	} else if ext != "" {
		// 如果没有传入 fileName，但能获取扩展名，则使用 objectName 作为文件名
		// 从 objectName 中提取文件名部分
		filename := r.extractFilenameFromObjectName(objectName)
		if filename != "" {
			reqParams.Set(
				"response-content-disposition",
				"inline; filename=\""+filename+"\"",
			)
		}
	}

	u, err := r.core.PresignedGetObject(
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

// 从对象名中提取文件名
func (r *minioRepo) extractFilenameFromObjectName(objectName string) string {
	if objectName == "" {
		return ""
	}

	// 找到最后一个斜杠
	idx := strings.LastIndex(objectName, "/")
	if idx == -1 {
		// 没有路径，直接返回
		return objectName
	}

	// 返回路径后的部分
	return objectName[idx+1:]
}

// 从对象名获取扩展名
func (r *minioRepo) getFileExtension(objectName string) string {
	if objectName == "" {
		return ""
	}
	// 找到最后一个点
	idx := strings.LastIndex(objectName, ".")
	if idx == -1 || idx == len(objectName)-1 {
		return ""
	}
	// 转换为小写
	return strings.ToLower(objectName[idx+1:])
}

// 根据扩展名获取 Content-Type
func (r *minioRepo) getContentType(ext string) string {
	switch ext {
	case "mp4", "m4v":
		return "video/mp4"
	case "mov":
		return "video/quicktime"
	case "avi":
		return "video/x-msvideo"
	case "webm":
		return "video/webm"
	case "mp3":
		return "audio/mpeg"
	case "wav":
		return "audio/wav"
	case "ogg":
		return "audio/ogg"
	case "jpg", "jpeg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "gif":
		return "image/gif"
	case "webp":
		return "image/webp"
	case "svg":
		return "image/svg+xml"
	case "pdf":
		return "application/pdf"
	default:
		return "application/octet-stream"
	}
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

	u, err := r.core.PresignedPutObject(
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

	u, err := r.core.Presign(
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

	stat, err := r.core.StatObject(
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
