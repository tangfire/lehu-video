package model

import "time"

type User struct {
	Id              int64     `json:"id" gorm:"column:id"`
	AccountId       int64     `json:"account_id" gorm:"column:account_id"`
	Mobile          string    `json:"mobile" gorm:"column:mobile"`
	Email           string    `json:"email" gorm:"column:email"`
	Name            string    `json:"name" gorm:"column:name"`
	Avatar          string    `json:"avatar" gorm:"column:avatar"`
	BackgroundImage string    `json:"background_image" gorm:"column:background_image"`
	Signature       string    `json:"signature" gorm:"column:signature"`
	CreatedAt       time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt       time.Time `json:"updated_at" gorm:"column:updated_at"`
}

func (m User) TableName() string {
	return "user"
}
