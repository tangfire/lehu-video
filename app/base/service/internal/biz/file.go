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
	// 确保有文件类型
	fileType := f.FileType

	// 如果file_type为空，尝试从FileName提取
	if fileType == "" && f.FileName != "" {
		// 从原始文件名提取后缀
		if idx := strings.LastIndex(f.FileName, "."); idx != -1 {
			fileType = strings.ToLower(f.FileName[idx+1:])
		}
	}

	// 默认后缀
	if fileType == "" {
		// 根据业务类型设置默认后缀
		if f.BizName == "video" {
			fileType = "mp4"
		} else if f.BizName == "cover" {
			fileType = "jpg"
		} else {
			fileType = "bin"
		}
	}

	return fmt.Sprintf("%s/%d.%s", f.BizName, f.Id, fileType)
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

// ✅ 查询操作使用Query/Result
type CheckFileQuery struct {
	File *File
}

type CheckFileResult struct {
	Exists bool
	FileId int64
}

// ✅ 命令操作使用Command/Result
type PreSignGetQuery struct {
	File *File
}

type PreSignGetResult struct {
	Url string
}

type PreSignPutCommand struct {
	File *File
}

type PreSignPutResult struct {
	Url    string
	FileId int64
}

type ReportUploadedCommand struct {
	File *File
}

type ReportUploadedResult struct {
	Url string
}

type PreSignSlicingPutCommand struct {
	File *File
}

type PreSignSlicingPutResult struct {
	SlicingFile *SlicingFile
	Exists      bool
	FileId      int64
}

type GetProgressRate4SlicingPutQuery struct {
	UploadId string
	File     *File
}

type GetProgressRate4SlicingPutResult struct {
	Progress map[string]bool
}

type MergeFilePartsCommand struct {
	UploadId string
	File     *File
}

type MergeFilePartsResult struct{}

type RemoveFileCommand struct {
	File *File
}

type RemoveFileResult struct{}

type GetFileInfoByIdQuery struct {
	DomainName string
	BizName    string
	FileId     int64
}

type GetFileInfoByIdResult struct {
	File *File
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

func (uc *FileUsecase) CheckFileExistedAndGetFile(ctx context.Context, query *CheckFileQuery) (*CheckFileResult, error) {
	tableName := uc.frh.GetTableNameByHash(query.File)
	exist, retFile, err := uc.repo.GetUploadedFileByHash(ctx, tableName, query.File.Hash)
	if err != nil {
		return nil, err
	}
	if !exist {
		return &CheckFileResult{Exists: false, FileId: 0}, nil
	}
	return &CheckFileResult{Exists: true, FileId: retFile.Id}, nil
}

func (uc *FileUsecase) PreSignGet(ctx context.Context, query *PreSignGetQuery) (*PreSignGetResult, error) {
	tableName := uc.frh.GetTableNameById(query.File)
	exist, retFile, err := uc.repo.GetUploadedFileById(ctx, tableName, query.File.Id)
	if err != nil {
		return nil, err
	}
	if !exist {
		return &PreSignGetResult{Url: ""}, nil
	}
	url, err := uc.minio.PreSignGetUrl(ctx, retFile.DomainName, retFile.GetObjectName(), query.File.FileName, query.File.ExpireSeconds)
	if err != nil {
		return nil, err
	}
	return &PreSignGetResult{Url: url}, nil
}

func (uc *FileUsecase) PreSignPut(ctx context.Context, cmd *PreSignPutCommand) (*PreSignPutResult, error) {
	checkQuery := &CheckFileQuery{File: cmd.File}
	checkResult, err := uc.CheckFileExistedAndGetFile(ctx, checkQuery)
	if err != nil {
		return nil, err
	}
	if checkResult.Exists {
		return &PreSignPutResult{Url: "", FileId: checkResult.FileId}, nil
	}

	cmd.File.SetId()
	err = uc.frh.AddFile(ctx, cmd.File)
	if err != nil {
		return nil, err
	}

	url, err := uc.minio.PreSignPutUrl(ctx, cmd.File.DomainName, cmd.File.GetObjectName(), cmd.File.ExpireSeconds)
	if err != nil {
		return nil, err
	}

	return &PreSignPutResult{Url: url, FileId: cmd.File.Id}, nil
}

func (uc *FileUsecase) ReportUploaded(ctx context.Context, cmd *ReportUploadedCommand) (*ReportUploadedResult, error) {
	tableName := uc.frh.GetTableNameById(cmd.File)
	exist, retFile, err := uc.repo.GetFileById(ctx, tableName, cmd.File.Id)
	if err != nil {
		return nil, err
	}
	if !exist {
		return &ReportUploadedResult{Url: ""}, nil
	}

	hash, err := uc.minio.GetObjectHash(ctx, retFile.DomainName, retFile.GetObjectName())
	if err != nil {
		return nil, err
	}

	equals := retFile.CheckHash(hash)
	if !equals && !strings.Contains(hash, "-") {
		log.Context(ctx).Errorf("failed to validate hash of uploaded file, hash: %s, expected: %s", hash, retFile.Hash)
		return nil, errors.New("failed to validate hash of uploaded file")
	}
	retFile.SetUploaded()
	err = uc.frh.UpdateFile(ctx, retFile)
	if err != nil {
		return nil, err
	}

	// 改用公共 URL
	publicUrl, err := uc.minio.GetPublicUrl(ctx, retFile.DomainName, retFile.GetObjectName())
	if err != nil {
		return nil, err
	}

	return &ReportUploadedResult{Url: publicUrl}, nil
}

func (uc *FileUsecase) PreSignSlicingPut(ctx context.Context, cmd *PreSignSlicingPutCommand) (*PreSignSlicingPutResult, error) {
	checkQuery := &CheckFileQuery{File: cmd.File}
	checkResult, err := uc.CheckFileExistedAndGetFile(ctx, checkQuery)
	if err != nil {
		return nil, err
	}
	if checkResult.Exists {
		return &PreSignSlicingPutResult{
			Exists: true,
			FileId: checkResult.FileId,
		}, nil
	}

	cmd.File.SetId()
	err = uc.frh.AddFile(ctx, cmd.File)
	if err != nil {
		return nil, err
	}

	uploadId, err := uc.minio.CreateSlicingUpload(ctx, cmd.File.DomainName, cmd.File.GetObjectName(), minio.PutObjectOptions{})
	if err != nil {
		return nil, err
	}
	slicingFile := NewSlicingFile(cmd.File)
	slicingFile.UploadId = uploadId
	slicingFile.SetTotalParts()
	urlList := make([]string, slicingFile.TotalParts)
	for i := 1; i <= int(slicingFile.TotalParts); i++ {
		url, err := uc.minio.PreSignSlicingPutUrl(ctx, cmd.File.DomainName, cmd.File.GetObjectName(), uploadId, int64(i))
		if err != nil {
			return nil, err
		}
		urlList[i-1] = url
	}
	slicingFile.UploadUrl = urlList

	return &PreSignSlicingPutResult{
		SlicingFile: slicingFile,
		Exists:      false,
		FileId:      cmd.File.Id,
	}, nil
}

func (uc *FileUsecase) GetProgressRate4SlicingPut(ctx context.Context, query *GetProgressRate4SlicingPutQuery) (*GetProgressRate4SlicingPutResult, error) {
	tableName := uc.frh.GetTableNameById(query.File)
	exist, retFile, err := uc.repo.GetFileById(ctx, tableName, query.File.Id)
	if err != nil {
		return nil, err
	}
	if !exist {
		return &GetProgressRate4SlicingPutResult{Progress: nil}, nil
	}
	sf := NewSlicingFile(retFile).SetTotalParts()
	result, err := uc.minio.ListSlicingFileParts(ctx, retFile.DomainName, retFile.GetObjectName(), query.UploadId, sf.TotalParts)
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

	return &GetProgressRate4SlicingPutResult{Progress: res}, nil
}

func (uc *FileUsecase) MergeFileParts(ctx context.Context, cmd *MergeFilePartsCommand) (*MergeFilePartsResult, error) {
	progressQuery := &GetProgressRate4SlicingPutQuery{
		UploadId: cmd.UploadId,
		File:     cmd.File,
	}
	progressResult, err := uc.GetProgressRate4SlicingPut(ctx, progressQuery)
	if err != nil {
		return nil, err
	}
	if ok, _ := uc.checkSlicingFileUploaded(progressResult.Progress); !ok {
		return nil, errors.New("not all parts uploaded")
	}

	tableName := uc.frh.GetTableNameById(cmd.File)
	exist, retFile, err := uc.repo.GetFileById(ctx, tableName, cmd.File.Id)
	if err != nil {
		return nil, err
	}
	if !exist {
		return &MergeFilePartsResult{}, nil
	}
	sf := NewSlicingFile(retFile).SetTotalParts()
	result, err := uc.minio.ListSlicingFileParts(ctx, retFile.DomainName, retFile.GetObjectName(), cmd.UploadId, sf.TotalParts)
	if err != nil {
		return nil, err
	}

	parts := make([]minio.CompletePart, 0)
	for i := 0; i < len(result.ObjectParts); i++ {
		parts = append(parts, minio.CompletePart{
			PartNumber: i + 1,
			ETag:       result.ObjectParts[i].ETag,
		})
	}
	err = uc.minio.MergeSlices(ctx, retFile.DomainName, retFile.GetObjectName(), cmd.UploadId, parts)
	if err != nil {
		return nil, err
	}

	// 报告上传完成
	reportCmd := &ReportUploadedCommand{File: cmd.File}
	_, err = uc.ReportUploaded(ctx, reportCmd)
	if err != nil {
		return nil, err
	}

	return &MergeFilePartsResult{}, nil
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

func (uc *FileUsecase) RemoveFile(ctx context.Context, cmd *RemoveFileCommand) (*RemoveFileResult, error) {
	err := uc.frh.RemoveFile(ctx, cmd.File)
	if err != nil {
		return nil, err
	}
	return &RemoveFileResult{}, nil
}

func (uc *FileUsecase) GetFileInfoById(ctx context.Context, query *GetFileInfoByIdQuery) (*GetFileInfoByIdResult, error) {
	file := &File{
		Id:         query.FileId,
		DomainName: query.DomainName,
		BizName:    query.BizName,
	}
	tableName := uc.frh.GetTableNameById(file)
	exist, retFile, err := uc.repo.GetFileById(ctx, tableName, file.Id)
	if err != nil {
		return nil, err
	}
	if !exist {
		return &GetFileInfoByIdResult{File: nil}, nil
	}
	return &GetFileInfoByIdResult{File: retFile}, nil
}
