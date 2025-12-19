package model

import "time"

type Follow struct {
	Id           int64     `gorm:"column:id" db:"id" json:"id" form:"id"`
	UserId       int64     `gorm:"column:user_id" db:"user_id" json:"user_id" form:"user_id"`
	TargetUserId int64     `gorm:"column:target_user_id" db:"target_user_id" json:"target_user_id" form:"target_user_id"` //  被关注的用户id
	CreateAt     time.Time `gorm:"column:create_at" db:"create_at" json:"create_at" form:"create_at"`
	UpdateAt     time.Time `gorm:"column:update_at" db:"update_at" json:"update_at" form:"update_at"`
}

func (Follow) TableName() string {
	return "follow"
}
