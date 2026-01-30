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
	Id              int64
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
	ID             int64
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
	ID          int64
	Name        string
	Avatar      string
	IsFollowing bool
}

type Comment struct {
	Id         int64
	VideoId    int64
	ParentId   int64
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
	Id          int64
	Name        string
	Avatar      string
	IsFollowing bool
}

// Collection 收藏夹
type Collection struct {
	Id          int64
	UserId      int64
	Name        string
	Description string
}

type FavoriteTarget int64

var (
	VIDEO   FavoriteTarget = 0
	COMMENT FavoriteTarget = 1
)

type FavoriteType int64

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
