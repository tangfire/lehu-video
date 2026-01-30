package model

import (
	"time"
)

type GroupInfo struct {
	Id        int64     `gorm:"column:id;primaryKey;comment:id"`
	Name      string    `gorm:"column:name;type:varchar(20);not null;comment:群名称"`
	Notice    string    `gorm:"column:notice;type:varchar(500);comment:群公告"`
	MemberCnt int       `gorm:"column:member_cnt;default:1;comment:群人数"`
	OwnerId   int64     `gorm:"column:owner_id;not null;comment:群主ID"`
	AddMode   int8      `gorm:"column:add_mode;default:0;comment:加群方式，0.直接，1.审核"`
	Avatar    string    `gorm:"column:avatar;type:varchar(255);comment:头像"`
	Status    int8      `gorm:"column:status;default:0;comment:状态，0.正常，1.禁用，2.解散"`
	CreatedAt time.Time `gorm:"column:created_at;index;type:datetime;not null;comment:创建时间"`
	UpdatedAt time.Time `gorm:"column:updated_at;type:datetime;not null;comment:更新时间"`
	IsDeleted bool      `gorm:"column:is_deleted;default:0;NOT NULL"`
}

func (GroupInfo) TableName() string {
	return "group_info"
}

type GroupMember struct {
	Id        int64     `gorm:"column:id;primaryKey;comment:id"`
	UserId    int64     `gorm:"column:user_id;not null;comment:用户ID"`
	GroupId   int64     `gorm:"column:group_id;not null;comment:群聊ID"`
	Role      int8      `gorm:"column:role;default:0;comment:角色，0.普通成员，1.管理员，2.群主"`
	JoinTime  time.Time `gorm:"column:join_time;not null;comment:加入时间"`
	IsDeleted bool      `gorm:"column:is_deleted;default:0;NOT NULL"`
}

func (GroupMember) TableName() string {
	return "group_member"
}

type GroupApply struct {
	Id          int64     `gorm:"column:id;primaryKey;comment:id"`
	UserId      int64     `gorm:"column:user_id;not null;comment:申请用户ID"`
	GroupId     int64     `gorm:"column:group_id;not null;comment:群聊ID"`
	ApplyReason string    `gorm:"column:apply_reason;type:varchar(200);comment:申请理由"`
	Status      int8      `gorm:"column:status;default:0;comment:状态，0.待处理，1.已通过，2.已拒绝"`
	HandlerId   int64     `gorm:"column:handler_id;comment:处理人ID"`
	ReplyMsg    string    `gorm:"column:reply_msg;type:varchar(200);comment:回复消息"`
	CreatedAt   time.Time `gorm:"column:created_at;not null;comment:创建时间"`
	UpdatedAt   time.Time `gorm:"column:updated_at;not null;comment:更新时间"`
	IsDeleted   bool      `gorm:"column:is_deleted;default:0;NOT NULL"`
}

func (GroupApply) TableName() string {
	return "group_apply"
}
