package model

import (
	"time"
)

type FriendRelation struct {
	ID        int64     `gorm:"column:id;primaryKey;comment:ID"`
	UserID    int64     `gorm:"column:user_id;not null;index:idx_user_id;comment:用户ID"`
	FriendID  int64     `gorm:"column:friend_id;not null;index:idx_friend_id;comment:好友ID"`
	Status    int8      `gorm:"column:status;default:0;comment:状态：0=待处理，1=已同意，2=已拒绝，3=已拉黑"`
	Remark    string    `gorm:"column:remark;type:varchar(50);comment:备注"`
	GroupName string    `gorm:"column:group_name;type:varchar(50);comment:分组名称"`
	CreatedAt time.Time `gorm:"column:created_at;type:datetime;not null;comment:创建时间"`
	UpdatedAt time.Time `gorm:"column:updated_at;type:datetime;not null;comment:更新时间"`
	IsDeleted bool      `gorm:"column:is_deleted;default:0;NOT NULL"`
}

func (FriendRelation) TableName() string {
	return "friend_relation"
}

type FriendApply struct {
	ID          int64      `gorm:"column:id;primaryKey;comment:ID"`
	ApplicantID int64      `gorm:"column:applicant_id;not null;index:idx_applicant;comment:申请人ID"`
	ReceiverID  int64      `gorm:"column:receiver_id;not null;index:idx_receiver;comment:接收人ID"`
	ApplyReason string     `gorm:"column:apply_reason;type:varchar(200);comment:申请理由"`
	Status      int8       `gorm:"column:status;default:0;comment:状态：0=待处理，1=已同意，2=已拒绝"`
	HandledAt   *time.Time `gorm:"column:handled_at;type:datetime;comment:处理时间"`
	CreatedAt   time.Time  `gorm:"column:created_at;type:datetime;not null;comment:创建时间"`
	UpdatedAt   time.Time  `gorm:"column:updated_at;type:datetime;not null;comment:更新时间"`
	IsDeleted   bool       `gorm:"column:is_deleted;default:0;NOT NULL"`
}

func (FriendApply) TableName() string {
	return "friend_apply"
}

type UserOnlineStatus struct {
	ID             int64     `gorm:"column:id;primaryKey;comment:ID"`
	UserID         int64     `gorm:"column:user_id;not null;uniqueIndex:uk_user_id;comment:用户ID"`
	Status         int8      `gorm:"column:status;default:0;comment:状态：0=离线，1=在线，2=忙碌，3=离开"`
	DeviceType     string    `gorm:"column:device_type;type:varchar(20);comment:设备类型：web/ios/android"`
	LastOnlineTime time.Time `gorm:"column:last_online_time;type:datetime;comment:最后在线时间"`
	CreatedAt      time.Time `gorm:"column:created_at;type:datetime;not null;comment:创建时间"`
	UpdatedAt      time.Time `gorm:"column:updated_at;type:datetime;not null;comment:更新时间"`
}

func (UserOnlineStatus) TableName() string {
	return "user_online_status"
}
