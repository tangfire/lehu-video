package biz

import "time"

// PageStats 分页参数
type PageStats struct {
	Page     int
	PageSize int
	Sort     string
}

// UserInfo 用户信息
type UserInfo struct {
	Id              string
	Name            string
	Nickname        string
	Avatar          string
	BackgroundImage string
	Signature       string
	Mobile          string
	Email           string
	Gender          int32
	OnlineStatus    int32
	LastOnlineTime  time.Time
}

// Video 视频
type Video struct {
	ID             string
	Author         *VideoAuthor
	PlayURL        string
	CoverURL       string
	FavoriteCount  int64
	CommentCount   int64
	IsFavorite     bool
	Title          string
	IsCollected    bool
	CollectedCount int64
}

// VideoAuthor 视频作者
type VideoAuthor struct {
	ID          string
	Name        string
	Avatar      string
	IsFollowing bool
}

type Comment struct {
	Id         string
	VideoId    string
	ParentId   string
	User       *CommentUser
	ReplyUser  *CommentUser
	Content    string
	Date       string
	LikeCount  int64
	ReplyCount int64
	Comments   []*Comment
}

// CommentUser 评论用户
type CommentUser struct {
	Id          string
	Name        string
	Avatar      string
	IsFollowing bool
}

// Collection 收藏夹
type Collection struct {
	Id          string
	UserId      string
	Name        string
	Description string
}

var (
	VIDEO   FavoriteTarget = 0
	COMMENT FavoriteTarget = 1
)

var (
	FAVORITE FavoriteType = 0
	UNLIKE   FavoriteType = 1
)

type FollowType int64

var (
	FOLLOWING FollowType = 0
	Follower  FollowType = 1
	BOTH      FollowType = 2
)

// FileInfo 文件信息
type FileInfo struct {
	ObjectName string `json:"object_name"`
	Hash       string `json:"hash"`
}
