package model

import (
	"time"
)

// UserOnlineStatus 用户在线状态表
type UserOnlineStatus struct {
	ID             int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`                 // 主键ID
	UserID         int64     `gorm:"column:user_id;not null;uniqueIndex" json:"user_id"`           // 用户ID
	OnlineStatus   int32     `gorm:"column:online_status;not null;default:0" json:"online_status"` // 在线状态：0=离线，1=在线，2=忙碌，3=离开
	DeviceType     string    `gorm:"column:device_type;size:20" json:"device_type"`                // 设备类型：web/ios/android
	LastOnlineTime time.Time `gorm:"column:last_online_time;not null" json:"last_online_time"`     // 最后在线时间
	CreatedAt      time.Time `gorm:"column:created_at;not null;autoCreateTime" json:"created_at"`  // 创建时间
	UpdatedAt      time.Time `gorm:"column:updated_at;not null;autoUpdateTime" json:"updated_at"`  // 更新时间
}

// TableName 指定表名
func (UserOnlineStatus) TableName() string {
	return "user_online_status"
}

// FriendRelation 好友关系表
type FriendRelation struct {
	ID          int64     `gorm:"column:id;primaryKey" json:"id"`                              // 主键ID
	UserID      int64     `gorm:"column:user_id;not null;index" json:"user_id"`                // 用户ID
	FriendID    int64     `gorm:"column:friend_id;not null;index" json:"friend_id"`            // 好友ID
	Status      int32     `gorm:"column:status;not null;default:1;index" json:"status"`        // 状态：1=好友，2=已删除，3=拉黑
	Remark      string    `gorm:"column:remark;size:100" json:"remark"`                        // 备注
	GroupName   string    `gorm:"column:group_name;size:50" json:"group_name"`                 // 分组名称
	IsFollowing bool      `gorm:"column:is_following;not null;default:1" json:"is_following"`  // 是否关注好友
	IsFollower  bool      `gorm:"column:is_follower;not null;default:1" json:"is_follower"`    // 是否被好友关注
	CreatedAt   time.Time `gorm:"column:created_at;not null;autoCreateTime" json:"created_at"` // 创建时间
	UpdatedAt   time.Time `gorm:"column:updated_at;not null;autoUpdateTime" json:"updated_at"` // 更新时间
}

// TableName 指定表名
func (FriendRelation) TableName() string {
	return "friend_relation"
}

// FriendApply 好友申请表
type FriendApply struct {
	ID          int64      `gorm:"column:id;primaryKey" json:"id"`                              // ID
	ApplicantID int64      `gorm:"column:applicant_id;not null" json:"applicant_id"`            // 申请人ID
	ReceiverID  int64      `gorm:"column:receiver_id;not null;index" json:"receiver_id"`        // 接收人ID
	ApplyReason string     `gorm:"column:apply_reason;size:200" json:"apply_reason"`            // 申请理由
	Status      int32      `gorm:"column:status;not null;default:0" json:"status"`              // 状态：0=待处理，1=已同意，2=已拒绝
	HandledAt   *time.Time `gorm:"column:handled_at" json:"handled_at"`                         // 处理时间
	CreatedAt   time.Time  `gorm:"column:created_at;not null;autoCreateTime" json:"created_at"` // 创建时间
	UpdatedAt   time.Time  `gorm:"column:updated_at;not null;autoUpdateTime" json:"updated_at"` // 更新时间
}

// TableName 指定表名
func (FriendApply) TableName() string {
	return "friend_apply"
}
