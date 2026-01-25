package biz

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"gorm.io/gorm"
	"math"
	"strconv"
	"strings"
	"time"
)

type File struct {
	Id            int64
	DomainName    string
	BizName       string
	Hash          string
	FileType      string
	FileSize      int64
	Uploaded      bool
	FileName      string
	ExpireSeconds int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (f *File) SetId() {
	f.Id = int64(uuid.New().ID())
}

func (f *File) GetObjectName() string {
	return fmt.Sprintf("%s/%d", f.BizName, f.Id)
}

func (f *File) CheckHash(hash string) bool {
	// 将两个哈希值都转换为小写
	fHashLower := strings.ToLower(f.Hash)
	hashLower := strings.ToLower(hash)

	// 比较小写哈希值
	return fHashLower == hashLower
}

func (f *File) SetUploaded() *File {
	f.Uploaded = true
	return f
}

const SizePerChunk float64 = 5 * 1024 * 1024

type SlicingFile struct {
	File       *File
	TotalParts int64
	UploadId   string
	UploadUrl  []string
}

func NewSlicingFile(f *File) *SlicingFile {
	return &SlicingFile{
		File: f,
	}
}

func (f *SlicingFile) SetUploadId(uploadId string) *SlicingFile {
	f.UploadId = uploadId
	return f
}

func (f *SlicingFile) SetTotalParts() *SlicingFile {
	f.TotalParts = int64(math.Ceil(float64(f.File.FileSize) / SizePerChunk))
	return f
}

type FileRepo interface {
	GetUploadedFileById(ctx context.Context, tableName string, id int64) (bool, *File, error)
	GetUploadedFileByHash(ctx context.Context, tableName string, hash string) (bool, *File, error)
	GetFileById(ctx context.Context, tableName string, id int64) (bool, *File, error)
	CreateFile(ctx context.Context, tx *gorm.DB, tableName string, file *File) error
	UpdateFile(ctx context.Context, tx *gorm.DB, tableName string, file *File) error
	DeleteFile(ctx context.Context, tx *gorm.DB, tableName string, id int64) error
}

type FileUsecase struct {
	repo  FileRepo
	minio MinioRepo
	frh   FileRepoHelper
	log   *log.Helper
}

func NewFileUsecase(repo FileRepo, logger log.Logger, frh FileRepoHelper, minio MinioRepo) *FileUsecase {
	return &FileUsecase{repo: repo, frh: frh, minio: minio, log: log.NewHelper(logger)}
}

func (uc *FileUsecase) CheckFileExistedAndGetFile(ctx context.Context, file *File) (bool, int64, error) {
	tableName := uc.frh.GetTableNameByHash(file)
	exist, retFile, err := uc.repo.GetUploadedFileByHash(ctx, tableName, file.Hash)
	if err != nil {
		return false, 0, err
	}
	if !exist {
		return false, 0, nil
	}
	return true, retFile.Id, nil
}

func (uc *FileUsecase) PreSignGet(ctx context.Context, file *File) (string, error) {
	tableName := uc.frh.GetTableNameById(file)
	exist, retFile, err := uc.repo.GetUploadedFileById(ctx, tableName, file.Id)
	if err != nil {
		return "", err
	}
	if !exist {
		return "", nil
	}
	url, err := uc.minio.PreSignGetUrl(ctx, retFile.DomainName, retFile.GetObjectName(), file.FileName, file.ExpireSeconds)
	if err != nil {
		return "", err
	}
	return url, nil
}
func (uc *FileUsecase) PreSignPut(ctx context.Context, file *File) (string, int64, error) {
	var err error
	exist, fileId, err := uc.CheckFileExistedAndGetFile(ctx, file)
	if err != nil {
		return "", 0, err
	}
	if exist {
		return "", fileId, nil
	}

	file.SetId()
	err = uc.frh.AddFile(ctx, file)
	if err != nil {
		return "", 0, err
	}

	url, err := uc.minio.PreSignPutUrl(ctx, file.DomainName, file.GetObjectName(), file.ExpireSeconds)
	if err != nil {
		return "", 0, err
	}

	return url, file.Id, nil
}
func (uc *FileUsecase) ReportUploaded(ctx context.Context, file *File) error {

	tableName := uc.frh.GetTableNameById(file)
	exist, retFile, err := uc.repo.GetFileById(ctx, tableName, file.Id)
	if err != nil {
		return err
	}
	if !exist {
		return nil
	}

	hash, err := uc.minio.GetObjectHash(ctx, retFile.DomainName, retFile.GetObjectName())
	if err != nil {
		return err
	}

	equals := retFile.CheckHash(hash)
	if !equals && !strings.Contains(hash, "-") {
		log.Context(ctx).Errorf("failed to validate hash of uploaded file, hash: %s, expected: %s", hash, retFile.Hash)
		return errors.New("failed to validate hash of uploaded file")
	}
	retFile.SetUploaded()
	err = uc.frh.UpdateFile(ctx, retFile)
	if err != nil {
		return err
	}
	return nil
}
func (uc *FileUsecase) PreSignSlicingPut(ctx context.Context, file *File) (*SlicingFile, error) {
	exist, fileId, err := uc.CheckFileExistedAndGetFile(ctx, file)
	if err != nil {
		return nil, err
	}
	if exist {
		return &SlicingFile{
			File: &File{Id: fileId},
		}, nil
	}

	file.SetId()
	err = uc.frh.AddFile(ctx, file)
	if err != nil {
		return nil, err
	}

	uploadId, err := uc.minio.CreateSlicingUpload(ctx, file.DomainName, file.GetObjectName(), minio.PutObjectOptions{})
	if err != nil {
		return nil, err
	}
	slicingFile := NewSlicingFile(file)
	slicingFile.UploadId = uploadId
	slicingFile.SetTotalParts()
	urlList := make([]string, slicingFile.TotalParts)
	for i := 1; i <= int(slicingFile.TotalParts); i++ {
		url, err := uc.minio.PreSignSlicingPutUrl(ctx, file.DomainName, file.GetObjectName(), uploadId, int64(i))
		if err != nil {
			return nil, err
		}
		urlList[i-1] = url
	}
	slicingFile.UploadUrl = urlList
	return slicingFile, nil
}
func (uc *FileUsecase) GetProgressRate4SlicingPut(ctx context.Context, uploadId string, file *File) (map[string]bool, error) {
	tableName := uc.frh.GetTableNameById(file)
	exist, retFile, err := uc.repo.GetFileById(ctx, tableName, file.Id)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, nil
	}
	sf := NewSlicingFile(retFile).SetTotalParts()
	result, err := uc.minio.ListSlicingFileParts(ctx, retFile.DomainName, retFile.GetObjectName(), uploadId, sf.TotalParts)
	if err != nil {
		return nil, err
	}
	res := make(map[string]bool)
	parts := result.ObjectParts
	for i := 0; i < int(sf.TotalParts); i++ {
		if len(parts[i].ETag) > 0 {
			res[strconv.FormatInt(int64(i+1), 10)] = true
		} else {
			res[strconv.FormatInt(int64(i+1), 10)] = false
		}
	}

	return res, nil
}
func (uc *FileUsecase) MergeFileParts(ctx context.Context, uploadId string, file *File) error {
	uploadResult, err := uc.GetProgressRate4SlicingPut(ctx, uploadId, file)
	if err != nil {
		return err
	}
	if ok, _ := uc.checkSlicingFileUploaded(uploadResult); !ok {
		return errors.New("not all parts uploaded")
	}

	tableName := uc.frh.GetTableNameById(file)
	exist, retFile, err := uc.repo.GetFileById(ctx, tableName, file.Id)
	if err != nil {
		return err
	}
	if !exist {
		return nil
	}
	sf := NewSlicingFile(retFile).SetTotalParts()
	result, err := uc.minio.ListSlicingFileParts(ctx, retFile.DomainName, retFile.GetObjectName(), uploadId, sf.TotalParts)
	if err != nil {
		return err
	}

	parts := make([]minio.CompletePart, 0)
	for i := 0; i < len(result.ObjectParts); i++ {
		parts = append(parts, minio.CompletePart{
			PartNumber: i + 1,
			ETag:       result.ObjectParts[i].ETag,
		})
	}
	err = uc.minio.MergeSlices(ctx, retFile.DomainName, retFile.GetObjectName(), uploadId, parts)
	if err != nil {
		return err
	}
	return nil
}

func (uc *FileUsecase) checkSlicingFileUploaded(res map[string]bool) (bool, string) {
	total := 0
	finished := 0
	for _, uploaded := range res {
		if uploaded {
			finished++
		}

		total++
	}

	rate := fmt.Sprintf("%d/%d", finished, total)
	return total == finished, rate

}

func (uc *FileUsecase) RemoveFile(ctx context.Context, file *File) error {
	err := uc.frh.RemoveFile(ctx, file)
	if err != nil {
		return err
	}
	return nil
}

func (uc *FileUsecase) GetFileInfoById(ctx context.Context, domainName, bizName string, fileId int64) (*File, error) {
	file := &File{
		Id:         fileId,
		DomainName: domainName,
		BizName:    bizName,
	}
	tableName := uc.frh.GetTableNameById(file)
	exist, retFile, err := uc.repo.GetFileById(ctx, tableName, file.Id)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, nil
	}
	return retFile, nil
}
