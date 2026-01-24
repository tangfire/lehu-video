package model

import "time"

const TableNameFile = "file"

type File struct {
	Id         int64     `gorm:"column:id" db:"id" json:"id" form:"id"`
	DomainName string    `gorm:"column:domain_name" db:"domain_name" json:"domain_name" form:"domain_name"`
	BizName    string    `gorm:"column:biz_name" db:"biz_name" json:"biz_name" form:"biz_name"`
	Hash       string    `gorm:"column:hash" db:"hash" json:"hash" form:"hash"`
	FileSize   int64     `gorm:"column:file_size" db:"file_size" json:"file_size" form:"file_size"`
	FileType   string    `gorm:"column:file_type" db:"file_type" json:"file_type" form:"file_type"`
	CreatedAt  time.Time `gorm:"column:created_at" db:"created_at" json:"created_at" form:"created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at" db:"updated_at" json:"updated_at" form:"updated_at"`
}

func (File) TableName() string {
	return "file"
}
