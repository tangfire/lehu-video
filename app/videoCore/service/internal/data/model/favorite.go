package model

import "time"

type Favorite struct {
	Id           int64     `gorm:"column:id" json:"id"`
	UserId       int64     `gorm:"column:user_id" json:"user_id"`
	TargetType   int64     `gorm:"column:target_type" json:"target_type"` // 0-视频, 1-评论
	TargetId     int64     `gorm:"column:target_id" json:"target_id"`
	FavoriteType int64     `gorm:"column:favorite_type" json:"favorite_type"`   // 0-点赞, 1-踩
	DeleteAt     int64     `gorm:"column:delete_at;default:0" json:"delete_at"` // 0 表示有效，非0为删除时间戳
	CreatedAt    time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (Favorite) TableName() string {
	return "favorite"
}
