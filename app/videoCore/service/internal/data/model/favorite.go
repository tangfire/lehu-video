package model

import "time"

type Favorite struct {
	Id           int64     `gorm:"column:id" db:"id" json:"id" form:"id"`
	UserId       int64     `gorm:"column:user_id" db:"user_id" json:"user_id" form:"user_id"`
	TargetType   int64     `gorm:"column:target_type" db:"target_type" json:"target_type" form:"target_type"`         //  点赞对象类型 1-视频 2-评论
	TargetId     int64     `gorm:"column:target_id" db:"target_id" json:"target_id" form:"target_id"`                 //  点赞对象id
	FavoriteType int64     `gorm:"column:favorite_type" db:"favorite_type" json:"favorite_type" form:"favorite_type"` //  点赞类型 1-点赞 2-踩
	IsDeleted    bool      `gorm:"column:is_deleted" db:"is_deleted" json:"is_deleted" form:"is_deleted"`
	CreatedAt    time.Time `gorm:"column:created_at" db:"created_at" json:"created_at" form:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at" db:"updated_at" json:"updated_at" form:"updated_at"`
}

func (Favorite) TableName() string {
	return "favorite"
}
