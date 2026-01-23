package model

import "time"

type Account struct {
	Id        int64     `gorm:"column:id" db:"id" json:"id" form:"id"`
	Mobile    string    `gorm:"column:mobile" db:"mobile" json:"mobile" form:"mobile"`
	Email     string    `gorm:"column:email" db:"email" json:"email" form:"email"`
	Password  string    `gorm:"column:password" db:"password" json:"password" form:"password"`
	Salt      string    `gorm:"column:salt" db:"salt" json:"salt" form:"salt"`
	CreatedAt time.Time `gorm:"column:created_at" db:"created_at" json:"created_at" form:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" db:"updated_at" json:"updated_at" form:"updated_at"`
}

func (Account) TableName() string {
	return "account"
}
