package model

import "time"

type Collection struct {
	Id          int64     `gorm:"column:id" db:"id" json:"id" form:"id"`
	UserId      int64     `gorm:"column:user_id" db:"user_id" json:"user_id" form:"user_id"`
	Title       string    `gorm:"column:title" db:"title" json:"title" form:"title"`
	Description string    `gorm:"column:description" db:"description" json:"description" form:"description"`
	IsDeleted   bool      `gorm:"column:is_deleted" db:"is_deleted" json:"is_deleted" form:"is_deleted"`
	CreatedAt   time.Time `gorm:"column:created_at" db:"created_at" json:"created_at" form:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at" db:"updated_at" json:"updated_at" form:"updated_at"`
}

func (Collection) TableName() string {
	return "collection"
}

type CollectionVideo struct {
	Id           int64     `gorm:"column:id" db:"id" json:"id" form:"id"`
	CollectionId int64     `gorm:"column:collection_id" db:"collection_id" json:"collection_id" form:"collection_id"`
	UserId       int64     `gorm:"column:user_id" db:"user_id" json:"user_id" form:"user_id"`
	VideoId      int64     `gorm:"column:video_id" db:"video_id" json:"video_id" form:"video_id"`
	IsDeleted    bool      `gorm:"column:is_deleted" db:"is_deleted" json:"is_deleted" form:"is_deleted"`
	CreatedAt    time.Time `gorm:"column:created_at" db:"created_at" json:"created_at" form:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at" db:"updated_at" json:"updated_at" form:"updated_at"`
}

func (CollectionVideo) TableName() string {
	return "collection_video"
}
