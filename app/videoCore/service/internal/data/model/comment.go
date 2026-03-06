package model

import "time"

type Comment struct {
	Id         int64     `gorm:"column:id;primaryKey"`
	VideoId    int64     `gorm:"column:video_id"`
	UserId     int64     `gorm:"column:user_id"`
	ParentId   int64     `gorm:"column:parent_id"`
	ToUserId   int64     `gorm:"column:to_user_id"`
	Content    string    `gorm:"column:content"`
	LikeCount  int64     `gorm:"column:like_count;default:0"`  // 点赞数冗余
	ReplyCount int64     `gorm:"column:reply_count;default:0"` // 直接子评论数冗余
	IsDeleted  bool      `gorm:"column:is_deleted"`
	CreatedAt  time.Time `gorm:"column:created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at"`
}

func (Comment) TableName() string {
	return "comment"
}
