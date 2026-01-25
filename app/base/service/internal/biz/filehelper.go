package biz

import "context"

type FileRepoHelper interface {
	AddFile(ctx context.Context, file *File) error
	RemoveFile(ctx context.Context, file *File) error
	UpdateFile(ctx context.Context, file *File) error
	GetTableNameById(file *File) string
	GetTableNameByHash(file *File) string
}
