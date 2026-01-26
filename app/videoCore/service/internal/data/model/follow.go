package model

import "time"

type Follow struct {
	Id           int64     `gorm:"column:id" db:"id" json:"id" form:"id"`
	UserId       int64     `gorm:"column:user_id" db:"user_id" json:"user_id" form:"user_id"`
	TargetUserId int64     `gorm:"column:target_user_id" db:"target_user_id" json:"target_user_id" form:"target_user_id"` // 被关注的用户id
	IsDeleted    bool      `gorm:"column:is_deleted" db:"is_deleted" json:"is_deleted" form:"is_deleted"`
	CreatedAt    time.Time `gorm:"column:created_at" db:"created_at" json:"created_at" form:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at" db:"updated_at" json:"updated_at" form:"updated_at"`
}

func (Follow) TableName() string {
	return "follow"
}
