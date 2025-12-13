package model

import "time"

type Video struct {
	Id           int64     `json:"id" gorm:"column:id"`
	UserId       int64     `json:"user_id" gorm:"column:user_id"`
	Title        string    `json:"title" gorm:"column:title"`
	Description  string    `json:"description" gorm:"column:description"`
	VideoUrl     string    `json:"video_url" gorm:"column:video_url"`
	CoverUrl     string    `json:"cover_url" gorm:"column:cover_url"`
	LikeCount    int64     `json:"like_count" gorm:"column:like_count"`
	CommentCount int64     `json:"comment_count" gorm:"column:comment_count"`
	CreatedAt    time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"column:updated_at"`
}

func (m Video) TableName() string {
	return "video"
}
