package model

import "time"

type User struct {
	Id              int64     `json:"id" gorm:"column:id;primary_key"`
	AccountId       int64     `json:"account_id" gorm:"column:account_id;index"`
	Mobile          string    `json:"mobile" gorm:"column:mobile;size:20;index"`
	Email           string    `json:"email" gorm:"column:email;size:100;index"`
	Name            string    `json:"name" gorm:"column:name;size:100"`
	Nickname        string    `json:"nickname" gorm:"column:nickname;size:100;index"`
	Avatar          string    `json:"avatar" gorm:"column:avatar;size:500"`
	BackgroundImage string    `json:"background_image" gorm:"column:background_image;size:500"`
	Signature       string    `json:"signature" gorm:"column:signature;size:500"`
	Gender          int32     `json:"gender" gorm:"column:gender;default:0"`
	FollowCount     int64     `json:"follow_count" gorm:"column:follow_count;default:0"`
	FollowerCount   int64     `json:"follower_count" gorm:"column:follower_count;default:0"`
	TotalFavorited  int64     `json:"total_favorited" gorm:"column:total_favorited;default:0"`
	WorkCount       int64     `json:"work_count" gorm:"column:work_count;default:0"`
	FavoriteCount   int64     `json:"favorite_count" gorm:"column:favorite_count;default:0"`
	CreatedAt       time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt       time.Time `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
}

func (m User) TableName() string {
	return "user"
}
