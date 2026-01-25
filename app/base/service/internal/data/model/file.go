package model

import "time"

const TableNameFile = "file"

type File struct {
	Id         int64     `gorm:"column:id;primary_key"`
	DomainName string    `gorm:"column:domain_name;NOT NULL"`
	BizName    string    `gorm:"column:biz_name;NOT NULL"`
	Hash       string    `gorm:"column:hash;NOT NULL"`
	FileSize   int64     `gorm:"column:file_size;default:0;NOT NULL"`
	FileType   string    `gorm:"column:file_type;NOT NULL"`
	Uploaded   bool      `gorm:"column:uploaded;default:0;NOT NULL"`
	IsDeleted  bool      `gorm:"column:is_deleted;default:0;NOT NULL"`
	CreatedAt  time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP;NOT NULL"`
	UpdatedAt  time.Time `gorm:"column:updated_at;default:CURRENT_TIMESTAMP;NOT NULL"`
}
