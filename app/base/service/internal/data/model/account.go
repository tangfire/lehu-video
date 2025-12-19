package model

import "time"

type Account struct {
	Id        int64     `gorm:"column:id" db:"id" json:"id" form:"id"`
	Mobile    string    `gorm:"column:mobile" db:"mobile" json:"mobile" form:"mobile"`
	Email     string    `gorm:"column:email" db:"email" json:"email" form:"email"`
	Password  string    `gorm:"column:password" db:"password" json:"password" form:"password"`
	Salt      string    `gorm:"column:salt" db:"salt" json:"salt" form:"salt"`
	CreatedAt time.Time `gorm:"column:create_at" db:"create_at" json:"create_at" form:"create_at"`
	UpdatedAt time.Time `gorm:"column:update_at" db:"update_at" json:"update_at" form:"update_at"`
}

func (Account) TableName() string {
	return "account"
}
