package model

import "time"

type Comment struct {
	Id        int64     `gorm:"column:id;primaryKey" db:"id" json:"id" form:"id"`
	VideoId   int64     `gorm:"column:video_id" db:"video_id" json:"video_id" form:"video_id"`
	UserId    int64     `gorm:"column:user_id" db:"user_id" json:"user_id" form:"user_id"`             //  发表评论的用户id
	ParentId  int64     `gorm:"column:parent_id" db:"parent_id" json:"parent_id" form:"parent_id"`     //  父评论id
	ToUserId  int64     `gorm:"column:to_user_id" db:"to_user_id" json:"to_user_id" form:"to_user_id"` //  评论所回复的用户id
	Content   string    `gorm:"column:content" db:"content" json:"content" form:"content"`             //  评论内容
	IsDeleted bool      `gorm:"column:is_deleted" db:"is_deleted" json:"is_deleted" form:"is_deleted"`
	CreatedAt time.Time `gorm:"column:created_at" db:"created_at" json:"created_at" form:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" db:"updated_at" json:"updated_at" form:"updated_at"`
}

func (Comment) TableName() string {
	return "comment"
}
