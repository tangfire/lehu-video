package data

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"lehu-video/app/base/service/internal/biz"
	"lehu-video/app/base/service/internal/conf"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type minioRepo struct {
	core           *minio.Core
	publicCore     *minio.Core
	publicEndpoint string
}

func NewMinioRepo(conf *conf.Data, core *minio.Core) biz.MinioRepo {
	internalEndpoint := fmt.Sprintf("%s:%s", conf.Minio.Host, conf.Minio.Port)
	publicEndpoint := normalizeMinioEndpoint(os.Getenv("LEHU_PUBLIC_MINIO_ENDPOINT"))
	if publicEndpoint == "" {
		publicEndpoint = internalEndpoint
	}

	publicCore := core
	if publicEndpoint != internalEndpoint {
		var err error
		publicCore, err = minio.NewCore(publicEndpoint, &minio.Options{
			Creds:     credentials.NewStaticV4(conf.Minio.AccessKey, conf.Minio.SecretKey, ""),
			Secure:    false,
			Transport: newMinioHostRewriteTransport(publicEndpoint, internalEndpoint),
		})
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	return &minioRepo{
		core:           core,
		publicCore:     publicCore,
		publicEndpoint: publicEndpoint,
	}
}

func newMinioHostRewriteTransport(publicEndpoint, internalEndpoint string) http.RoundTripper {
	baseTransport := http.DefaultTransport.(*http.Transport).Clone()
	baseTransport.DialContext = (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext
	baseTransport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}

	if publicEndpoint == "" || internalEndpoint == "" || publicEndpoint == internalEndpoint {
		return baseTransport
	}

	baseDial := baseTransport.DialContext
	baseTransport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		if address == publicEndpoint {
			address = internalEndpoint
		}
		return baseDial(ctx, network, address)
	}
	return baseTransport
}

func normalizeMinioEndpoint(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	return strings.TrimRight(endpoint, "/")
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

	u, err := r.publicCore.PresignedGetObject(
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

	u, err := r.publicCore.PresignedPutObject(
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

	u, err := r.publicCore.Presign(
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

// GetPublicUrl 返回对象的公共访问 URL（需要桶已设置为 public）
func (r *minioRepo) GetPublicUrl(ctx context.Context, bucketName, objectName string) (string, error) {
	// 拼接公共地址，确保末尾没有多余的斜杠
	return fmt.Sprintf("http://%s/%s/%s", r.publicEndpoint, bucketName, objectName), nil
}
