package biz

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/go-kratos/kratos/v2/log"
	"lehu-video/pkg/apperror"
	sharedauth "lehu-video/pkg/auth"
)

const (
	CampusAuditStatusPending  int32 = 0
	CampusAuditStatusVisible  int32 = 1
	CampusAuditStatusRejected int32 = 2
	CampusAuditStatusDeleted  int32 = 3

	CampusFeedbackStatusPending    int32 = 0
	CampusFeedbackStatusProcessing int32 = 1
	CampusFeedbackStatusResolved   int32 = 2

	CampusIPBlockStatusActive   int32 = 1
	CampusIPBlockStatusInactive int32 = 0

	CampusAuthStatusUnverified int32 = 0
	CampusAuthStatusVerified   int32 = 1

	CampusPostMediaText  = "text"
	CampusPostMediaImage = "image"
	CampusPostMediaVideo = "video"

	CampusPostTypeNote     = "note"
	CampusPostTypeLost     = "lost"
	CampusPostTypeQuestion = "question"
	CampusPostTypeGuide    = "guide"
	CampusPostTypeClub     = "club"

	CampusPostSortRecommend = "recommend"
	CampusPostSortHot       = "hot"
	CampusPostSortNew       = "new"

	CampusNotificationTypeComment     = "comment"
	CampusNotificationTypeReply       = "reply"
	CampusNotificationTypePostLike    = "post_like"
	CampusNotificationTypePostCollect = "post_collect"
	CampusNotificationTypeCommentLike = "comment_like"
	CampusNotificationTypeSystem      = "system"

	CampusNotificationGroupAll         = "all"
	CampusNotificationGroupReply       = "reply"
	CampusNotificationGroupInteraction = "interaction"
	CampusNotificationGroupSystem      = "system"

	CampusNotificationOutboxStatusPending    = "pending"
	CampusNotificationOutboxStatusProcessing = "processing"
	CampusNotificationOutboxStatusDone       = "done"
	CampusNotificationOutboxStatusFailed     = "failed"

	campusNotificationOutboxMaxRetry = 5

	CampusAIReplyTaskStatusPending    = "pending"
	CampusAIReplyTaskStatusProcessing = "processing"
	CampusAIReplyTaskStatusDone       = "done"
	CampusAIReplyTaskStatusFailed     = "failed"

	campusAIReplyTaskMaxRetry = 3

	CampusPostAuditModeOff    = "off"
	CampusPostAuditModeManual = "manual"
	CampusPostAuditModeAI     = "ai"

	CampusAIContentAuditTaskStatusPending    = "pending"
	CampusAIContentAuditTaskStatusProcessing = "processing"
	CampusAIContentAuditTaskStatusDone       = "done"
	CampusAIContentAuditTaskStatusFailed     = "failed"

	CampusAIContentAuditDecisionPass   = "pass"
	CampusAIContentAuditDecisionReview = "review"
	CampusAIContentAuditDecisionReject = "reject"

	campusAIContentAuditTaskMaxRetry = 3

	CampusKnowledgeDocumentStatusDraft    = "draft"
	CampusKnowledgeDocumentStatusIndexing = "indexing"
	CampusKnowledgeDocumentStatusActive   = "active"
	CampusKnowledgeDocumentStatusDisabled = "disabled"
	CampusKnowledgeDocumentStatusFailed   = "failed"

	CampusKnowledgeChunkStatusActive   = "active"
	CampusKnowledgeChunkStatusDisabled = "disabled"
	CampusKnowledgeChunkStatusFailed   = "failed"

	CampusKnowledgeContentTypeFile = "file"
	CampusKnowledgeContentTypeText = "text"
)

type CampusIDGenerator interface {
	NextID() int64
}

type campusIDGenerator struct {
	node *snowflake.Node
}

func NewCampusIDGenerator() (CampusIDGenerator, error) {
	node, err := snowflake.NewNode(21)
	if err != nil {
		return nil, fmt.Errorf("create campus id generator: %w", err)
	}
	return &campusIDGenerator{node: node}, nil
}

func (g *campusIDGenerator) NextID() int64 {
	return g.node.Generate().Int64()
}

type CampusForumCategory struct {
	ID          int64
	Code        string
	Name        string
	Description string
	SortOrder   int32
}

type CampusProfile struct {
	ID           int64
	UserID       string
	AccountID    string
	OpenID       string
	UnionID      string
	SchoolName   string
	StudentNo    string
	RealName     string
	ClassName    string
	DormBuilding string
	RoomNo       string
	Mobile       string
	AuthStatus   int32
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type CampusWechatIdentity struct {
	ID        int64
	Provider  string
	OpenID    string
	UnionID   string
	UserID    string
	AccountID string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CampusTimetableCourse struct {
	ID             int64
	UserID         string
	Term           string
	CourseName     string
	Teacher        string
	Classroom      string
	Weekday        int32
	StartSection   int32
	EndSection     int32
	StartWeek      int32
	EndWeek        int32
	WeekParity     int32
	Source         string
	SourceCourseID string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type CampusTimetableProvider interface {
	Fetch(ctx context.Context, studentNo, password, term string) ([]*CampusTimetableCourse, error)
}

type MockCampusTimetableProvider struct{}

func NewMockCampusTimetableProvider() CampusTimetableProvider {
	return &MockCampusTimetableProvider{}
}

func (p *MockCampusTimetableProvider) Fetch(ctx context.Context, studentNo, password, term string) ([]*CampusTimetableCourse, error) {
	_ = ctx
	if strings.TrimSpace(password) == "" {
		return nil, apperror.InvalidArgument("教务系统密码不能为空")
	}
	seed := shortHash(studentNo+term, 8)
	return []*CampusTimetableCourse{
		{
			CourseName:     "高等数学 A",
			Teacher:        "李老师",
			Classroom:      "教学楼 A203",
			Weekday:        1,
			StartSection:   1,
			EndSection:     2,
			StartWeek:      1,
			EndWeek:        16,
			WeekParity:     0,
			SourceCourseID: "mock-math-" + seed,
		},
		{
			CourseName:     "大学英语",
			Teacher:        "陈老师",
			Classroom:      "教学楼 B105",
			Weekday:        2,
			StartSection:   3,
			EndSection:     4,
			StartWeek:      1,
			EndWeek:        16,
			WeekParity:     0,
			SourceCourseID: "mock-english-" + seed,
		},
		{
			CourseName:     "程序设计基础",
			Teacher:        "王老师",
			Classroom:      "实训楼 C301",
			Weekday:        3,
			StartSection:   5,
			EndSection:     6,
			StartWeek:      2,
			EndWeek:        15,
			WeekParity:     0,
			SourceCourseID: "mock-code-" + seed,
		},
		{
			CourseName:     "体育",
			Teacher:        "周老师",
			Classroom:      "运动场",
			Weekday:        5,
			StartSection:   7,
			EndSection:     8,
			StartWeek:      1,
			EndWeek:        12,
			WeekParity:     0,
			SourceCourseID: "mock-pe-" + seed,
		},
	}, nil
}

type CampusForumAuthor struct {
	UserID     string
	Name       string
	Nickname   string
	Avatar     string
	SchoolName string
	AuthStatus int32
}

type CampusPublicUserStats struct {
	PostCount       int64
	LikeCount       int64
	CollectedCount  int64
	HasOfficialPost bool
}

type CampusPublicUserProfile struct {
	UserID     string
	Name       string
	Nickname   string
	Avatar     string
	SchoolName string
	AuthStatus int32
	IsOfficial bool
	Bio        string
	Stats      *CampusPublicUserStats
}

type CampusForumPost struct {
	ID              int64
	CategoryCode    string
	CategoryName    string
	AuthorID        string
	Author          *CampusForumAuthor
	Title           string
	Content         string
	Images          []string
	MediaType       string
	PostType        string
	Extra           map[string]string
	CoverURL        string
	VideoURL        string
	IsOfficial      bool
	IsFeatured      bool
	IsPinned        bool
	SortWeight      int32
	Status          int32
	AuditReason     string
	AIAuditStatus   string
	AIAuditRisk     string
	AIAuditDecision string
	AIAuditReason   string
	AIAuditError    string
	LikeCount       int64
	CommentCount    int64
	CollectedCount  int64
	IsLiked         bool
	IsCollected     bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type CampusForumComment struct {
	ID               int64
	PostID           int64
	Post             *CampusForumPost
	ParentID         int64
	ReplyToCommentID int64
	ReplyToUserID    string
	ReplyToUser      *CampusForumAuthor
	AuthorID         string
	Author           *CampusForumAuthor
	Content          string
	Images           []string
	Status           int32
	AuditReason      string
	LikeCount        int64
	ReplyCount       int64
	IsLiked          bool
	PreviewReplies   []*CampusForumComment
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type CampusForumReport struct {
	ID         int64
	TargetType string
	TargetID   int64
	Target     *CampusForumPost
	Comment    *CampusForumComment
	ReporterID string
	Reporter   *CampusForumAuthor
	Reason     string
	Detail     string
	Status     int32
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type CampusFeedback struct {
	ID           int64
	UserID       string
	Author       *CampusForumAuthor
	FeedbackType string
	Content      string
	Contact      string
	Images       []string
	Status       int32
	OperatorNote string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type CampusNotification struct {
	ID          int64
	RecipientID string
	ActorID     string
	Actor       *CampusForumAuthor
	EventType   string
	TargetType  string
	TargetID    int64
	DedupeKey   string
	Title       string
	Content     string
	LinkPage    string
	LinkParams  map[string]string
	ReadAt      *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CampusNotificationOutbox struct {
	ID          int64
	RecipientID string
	ActorID     string
	EventType   string
	TargetType  string
	TargetID    int64
	DedupeKey   string
	Title       string
	Content     string
	LinkPage    string
	LinkParams  map[string]string
	Audience    string
	Status      string
	RetryCount  int32
	NextRetryAt *time.Time
	LockedUntil *time.Time
	LastError   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ProcessedAt *time.Time
}

type CampusAIReplyTask struct {
	ID               int64
	PostID           int64
	RootCommentID    int64
	TriggerCommentID int64
	AskerID          string
	BotUserID        string
	Prompt           string
	Status           string
	RetryCount       int32
	NextRetryAt      *time.Time
	LockedUntil      *time.Time
	AnswerCommentID  int64
	LastError        string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	ProcessedAt      *time.Time
}

type CampusOpsAuditSettings struct {
	PostAuditMode string
	AIEnabled     bool
	UpdatedBy     string
	UpdatedAt     time.Time
}

type CampusAIContentAuditTask struct {
	ID          int64
	TargetType  string
	TargetID    int64
	Status      string
	RiskLevel   string
	Decision    string
	Reason      string
	RawResult   string
	RetryCount  int32
	NextRetryAt *time.Time
	LockedUntil *time.Time
	LastError   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ProcessedAt *time.Time
}

type CampusAIReplyOverview struct {
	Enabled    bool
	BotUserID  string
	BotReady   bool
	BotName    string
	BotAvatar  string
	Model      string
	BaseURL    string
	RAGHealth  *CampusRAGHealth
	DailyLimit int64
	TodayUsed  int64
	Pending    int64
	Processing int64
	Done       int64
	Failed     int64
	Recent     []*CampusAIReplyTask
}

type CampusKnowledgeDocument struct {
	ID           int64
	Title        string
	Source       string
	Category     string
	ContentType  string
	FileURL      string
	FileID       string
	FileType     string
	RawContent   string
	Status       string
	ParseStatus  string
	ErrorMessage string
	UploadedBy   string
	EffectiveAt  *time.Time
	ExpiredAt    *time.Time
	ChunkCount   int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type CampusKnowledgeChunk struct {
	ID              int64
	DocumentID      int64
	ChunkIndex      int32
	Title           string
	Content         string
	Summary         string
	Category        string
	Keywords        []string
	Source          string
	Status          string
	QdrantPointID   string
	EmbeddingStatus string
	Score           float64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type CampusRAGQueryLog struct {
	ID               int64
	UserID           string
	PostID           int64
	TriggerCommentID int64
	Query            string
	NeedKnowledge    bool
	Confidence       float64
	HitChunks        []*CampusRAGQueryChunk
	Answer           string
	Model            string
	DurationMs       int64
	ErrorMessage     string
	CreatedAt        time.Time
}

type CampusAccessLog struct {
	ID          int64
	UserID      string
	IP          string
	Method      string
	Path        string
	StatusCode  int32
	DurationMs  int64
	UserAgent   string
	RateLimited bool
	Blocked     bool
	CreatedAt   time.Time
}

type CampusIPBlock struct {
	ID        int64
	IP        string
	Reason    string
	Status    int32
	CreatedBy string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CampusAuditLog struct {
	ID         int64
	TargetType string
	TargetID   int64
	UserID     string
	Provider   string
	Result     string
	Reason     string
	CreatedAt  time.Time
}

type WechatLoginInput struct {
	Code     string
	Nickname string
	Avatar   string
}

type WechatLoginOutput struct {
	Token   string
	Profile *CampusProfile
	User    *UserBaseInfo
}

type UpdateCampusProfileInput struct {
	UserID       string
	SchoolName   string
	StudentNo    string
	RealName     string
	ClassName    string
	DormBuilding string
	RoomNo       string
	Mobile       string
}

type ImportCampusTimetableInput struct {
	UserID    string
	StudentNo string
	Password  string
	Term      string
}

type ImportCampusTimetableOutput struct {
	Term    string
	Courses []*CampusTimetableCourse
	Count   int32
}

type ListCampusTimetableInput struct {
	UserID string
	Term   string
}

type ListCampusTimetableOutput struct {
	Term    string
	Courses []*CampusTimetableCourse
}

type CreateCampusPostInput struct {
	UserID       string
	CategoryCode string
	Title        string
	Content      string
	Images       []string
	MediaType    string
	PostType     string
	Extra        map[string]string
	CoverURL     string
	VideoURL     string
	IsOfficial   bool
	IsFeatured   bool
	IsPinned     bool
	SortWeight   int32
}

type ListCampusPostsInput struct {
	CurrentUserID string
	AuthorID      string
	CategoryCode  string
	PostType      string
	Sort          string
	Keyword       string
	Page          int32
	Size          int32
}

type ListCampusPostsOutput struct {
	Posts []*CampusForumPost
	Total int64
}

type ListCampusPostQuery struct {
	CategoryCode      string
	PostType          string
	Sort              string
	Keyword           string
	AuthorID          string
	CollectedByUserID string
	Statuses          []int32
	IncludeDeleted    bool
	OnlyOfficial      *bool
	OnlyFeatured      *bool
	OnlyPinned        *bool
	OnlyReported      bool
	Offset            int
	Limit             int
}

type GetCampusPostInput struct {
	CurrentUserID string
	PostID        int64
}

type CreateCampusCommentInput struct {
	UserID           string
	PostID           int64
	ParentID         int64
	ReplyToCommentID int64
	Content          string
	Images           []string
}

type ListCampusCommentsInput struct {
	PostID        int64
	UserID        string
	CommentID     int64
	CurrentUserID string
	Page          int32
	Size          int32
}

type ListCampusCommentsOutput struct {
	Comments []*CampusForumComment
	Total    int64
}

type ListCampusCommentQuery struct {
	PostID         int64
	ParentID       *int64
	AuthorID       string
	Statuses       []int32
	IncludeDeleted bool
	Offset         int
	Limit          int
}

type ReportCampusContentInput struct {
	UserID     string
	TargetType string
	TargetID   int64
	Reason     string
	Detail     string
}

type ListCampusModerationInput struct {
	UserID string
	Status int32
	Page   int32
	Size   int32
}

type CampusAdminSummary struct {
	TotalUsers       int64
	TodayUsers       int64
	TotalLogins      int64
	TodayLogins      int64
	TotalVisits      int64
	TodayVisits      int64
	TotalShares      int64
	TodayShares      int64
	TodayPublishOpen int64
	TodayPublishDone int64
	TodayDetailViews int64
	TodayFeedback    int64
	TodayReports     int64
	TotalPosts       int64
	TodayPosts       int64
	TotalComments    int64
	TodayComments    int64
	TotalLikes       int64
	TodayLikes       int64
	TotalCollections int64
	TodayCollections int64
	TotalReports     int64
	PendingReports   int64
	PendingFeedback  int64
	PendingPosts     int64
	PendingComments  int64
	PendingAIAudits  int64
	FeaturedPosts    int64
	OfficialPosts    int64
	Trends           []*CampusAdminTrend
}

type CampusAdminTrend struct {
	Date        string
	Users       int64
	Logins      int64
	Visits      int64
	Shares      int64
	Posts       int64
	Comments    int64
	Likes       int64
	Collections int64
	Reports     int64
}

type ListCampusAdminPostsInput struct {
	UserID       string
	CategoryCode string
	PostType     string
	OpsFilter    string
	Keyword      string
	Status       int32
	Sort         string
	Page         int32
	Size         int32
}

type UpdateCampusAdminPostInput struct {
	UserID       string
	PostID       int64
	CategoryCode string
	Title        string
	Content      string
	Images       []string
	MediaType    string
	PostType     string
	Extra        map[string]string
	CoverURL     string
	VideoURL     string
	Status       int32
	AuditReason  string
	IsOfficial   bool
	IsFeatured   bool
	IsPinned     bool
	SortWeight   int32
}

type BatchCampusAdminPostsInput struct {
	UserID     string
	PostIDs    []int64
	Action     string
	SortWeight int32
}

type BatchCampusAdminPostsOutput struct {
	UpdatedCount int32
}

type ListCampusAdminCommentsInput struct {
	UserID string
	Status int32
	PostID int64
	Page   int32
	Size   int32
}

type ListCampusAIReplyTasksInput struct {
	UserID string
	Status string
	Page   int32
	Size   int32
}

type ListCampusAIReplyTasksOutput struct {
	Tasks []*CampusAIReplyTask
	Total int64
}

type RetryCampusAIReplyTaskInput struct {
	UserID string
	TaskID int64
}

type GetCampusAuditSettingsInput struct {
	UserID string
}

type UpdateCampusAuditSettingsInput struct {
	UserID        string
	PostAuditMode string
}

type ListCampusKnowledgeDocumentsInput struct {
	UserID   string
	Keyword  string
	Category string
	Status   string
	Page     int32
	Size     int32
}

type ListCampusKnowledgeDocumentsOutput struct {
	Documents []*CampusKnowledgeDocument
	Total     int64
}

type CreateCampusKnowledgeDocumentInput struct {
	UserID      string
	Title       string
	Source      string
	Category    string
	ContentType string
	FileURL     string
	FileID      string
	FileType    string
	RawContent  string
	Status      string
	EffectiveAt *time.Time
	ExpiredAt   *time.Time
}

type UpdateCampusKnowledgeDocumentInput struct {
	UserID      string
	DocumentID  int64
	Title       string
	Source      string
	Category    string
	Status      string
	EffectiveAt *time.Time
	ExpiredAt   *time.Time
}

type ListCampusKnowledgeChunksInput struct {
	UserID     string
	DocumentID int64
	Page       int32
	Size       int32
}

type ListCampusKnowledgeChunksOutput struct {
	Chunks []*CampusKnowledgeChunk
	Total  int64
}

type TestCampusKnowledgeQueryInput struct {
	UserID string
	Query  string
	TopK   int32
}

type ListCampusRAGQueryLogsInput struct {
	UserID string
	Page   int32
	Size   int32
}

type ListCampusRAGQueryLogsOutput struct {
	Logs  []*CampusRAGQueryLog
	Total int64
}

type ListCampusReportsInput struct {
	UserID string
	Status int32
	Page   int32
	Size   int32
}

type ListCampusReportsOutput struct {
	Reports []*CampusForumReport
	Total   int64
}

type ReviewCampusReportInput struct {
	UserID   string
	ReportID int64
	Action   string
	Reason   string
}

type CampusAdminUser struct {
	User             *UserBaseInfo
	Profile          *CampusProfile
	Role             string
	PostCount        int64
	CommentCount     int64
	LikeCount        int64
	CollectionCount  int64
	FeedbackCount    int64
	ReportCount      int64
	LoginCount       int64
	VisitCount       int64
	LastLoginAt      time.Time
	LastActiveAt     time.Time
	LastActiveIP     string
	LastActivePath   string
	LastActiveStatus int32
}

type ListCampusAdminUsersInput struct {
	UserID     string
	Keyword    string
	Role       string
	AuthStatus int32
	Page       int32
	Size       int32
}

type ListCampusAdminUsersOutput struct {
	Users []*CampusAdminUser
	Total int64
}

type UpdateCampusUserRoleInput struct {
	UserID       string
	TargetUserID string
	Role         string
}

type CreateCampusFeedbackInput struct {
	UserID       string
	FeedbackType string
	Content      string
	Contact      string
	Images       []string
}

type ListCampusFeedbackInput struct {
	UserID string
	Status int32
	Page   int32
	Size   int32
}

type ListCampusFeedbackOutput struct {
	Feedbacks []*CampusFeedback
	Total     int64
}

type ReviewCampusFeedbackInput struct {
	UserID       string
	FeedbackID   int64
	Status       int32
	OperatorNote string
}

type ListCampusNotificationsInput struct {
	UserID string
	Type   string
	Page   int32
	Size   int32
}

type ListCampusNotificationsOutput struct {
	Notifications []*CampusNotification
	Total         int64
}

type CreateCampusAdminNotificationInput struct {
	UserID     string
	Title      string
	Content    string
	LinkPage   string
	LinkParams map[string]string
	Audience   string
}

type CampusUnreadNotificationCount struct {
	Total       int64
	Reply       int64
	Interaction int64
	System      int64
}

type CampusRateLimitInput struct {
	UserID   string
	IP       string
	Method   string
	Path     string
	Category string
}

type CampusAccessLogInput struct {
	UserID      string
	IP          string
	Method      string
	Path        string
	StatusCode  int32
	DurationMs  int64
	UserAgent   string
	RateLimited bool
	Blocked     bool
}

type CampusSecurityOverview struct {
	TodayRequests    int64
	TodayUniqueIPs   int64
	TodayRateLimited int64
	TodayBlocked     int64
	TodayErrors      int64
	ActiveBlockedIPs int64
	TopIPs           []*CampusSecurityIPStat
	TopPaths         []*CampusSecurityPathStat
	RecentAccessLogs []*CampusAccessLog
	BlockedIPs       []*CampusIPBlock
}

type CampusStatsReconcileResult struct {
	CheckedAt       time.Time
	UpdatedPosts    int64
	UpdatedComments int64
}

type CampusUploadPresignInput struct {
	MediaType string
	Hash      string
	FileType  string
	Filename  string
	Size      int64
}

type CampusUploadPresignOutput struct {
	FileID    string
	UploadURL string
	Method    string
	Headers   map[string]string
	ExpiresIn int64
}

type CampusUploadCompleteInput struct {
	MediaType string
	FileID    string
}

type CampusUploadCompleteOutput struct {
	FileID     string
	URL        string
	ObjectName string
}

type CampusSecurityIPStat struct {
	IP           string
	RequestCount int64
	ErrorCount   int64
	LastSeen     time.Time
}

type CampusSecurityPathStat struct {
	Path         string
	RequestCount int64
	ErrorCount   int64
}

type ListCampusSecurityInput struct {
	UserID string
}

type BlockCampusIPInput struct {
	UserID string
	IP     string
	Reason string
}

type TrackCampusEventInput struct {
	UserID     string
	EventType  string
	Page       string
	TargetType string
	TargetID   int64
	Channel    string
	Extra      map[string]string
	UserAgent  string
	IP         string
}

type ReviewCampusContentInput struct {
	UserID     string
	TargetType string
	TargetID   int64
	Action     string
	Reason     string
}

type CampusRepo interface {
	GetWechatIdentity(ctx context.Context, provider, openID string) (bool, *CampusWechatIdentity, error)
	GetAccountIDByEmail(ctx context.Context, email string) (bool, string, error)
	SaveWechatIdentity(ctx context.Context, identity *CampusWechatIdentity) error
	GetProfileByUserID(ctx context.Context, userID string) (bool, *CampusProfile, error)
	SaveProfile(ctx context.Context, profile *CampusProfile) error
	UpdateProfile(ctx context.Context, profile *CampusProfile) error
	ReplaceTimetableCourses(ctx context.Context, userID, term, source string, courses []*CampusTimetableCourse) error
	ListTimetableCourses(ctx context.Context, userID, term string) ([]*CampusTimetableCourse, error)
	ListCategories(ctx context.Context) ([]*CampusForumCategory, error)
	GetCategoryByCode(ctx context.Context, code string) (bool, *CampusForumCategory, error)
	CreatePost(ctx context.Context, post *CampusForumPost) error
	ListPosts(ctx context.Context, query ListCampusPostQuery) ([]*CampusForumPost, int64, error)
	GetPublicUserPostStats(ctx context.Context, userID string) (*CampusPublicUserStats, error)
	ListPostsByIDs(ctx context.Context, postIDs []int64, statuses []int32) ([]*CampusForumPost, error)
	GetPostByID(ctx context.Context, postID int64) (bool, *CampusForumPost, error)
	GetAnyPostByID(ctx context.Context, postID int64) (bool, *CampusForumPost, error)
	DeletePost(ctx context.Context, postID int64) error
	UpdatePostStatus(ctx context.Context, postID int64, status int32, reason string) error
	UpdatePostByAdmin(ctx context.Context, post *CampusForumPost) error
	CreateComment(ctx context.Context, comment *CampusForumComment) error
	CreateCommentWithOutbox(ctx context.Context, comment *CampusForumComment, outbox *CampusNotificationOutbox) error
	ListComments(ctx context.Context, query ListCampusCommentQuery) ([]*CampusForumComment, int64, error)
	FillCommentPosts(ctx context.Context, comments []*CampusForumComment) error
	GetCommentByID(ctx context.Context, commentID int64) (bool, *CampusForumComment, error)
	GetAnyCommentByID(ctx context.Context, commentID int64) (bool, *CampusForumComment, error)
	DeleteComment(ctx context.Context, commentID int64) error
	UpdateCommentStatus(ctx context.Context, commentID int64, status int32, reason string) error
	GetCommentLikeStatus(ctx context.Context, userID string, commentIDs []int64) (map[int64]bool, error)
	AddCommentLike(ctx context.Context, id int64, userID string, commentID int64) error
	AddCommentLikeWithOutbox(ctx context.Context, id int64, userID string, commentID int64, outbox *CampusNotificationOutbox) error
	RemoveCommentLike(ctx context.Context, userID string, commentID int64) error
	GetPostLikeStatus(ctx context.Context, userID string, postIDs []int64) (map[int64]bool, error)
	AddPostLike(ctx context.Context, id int64, userID string, postID int64) error
	AddPostLikeWithOutbox(ctx context.Context, id int64, userID string, postID int64, outbox *CampusNotificationOutbox) error
	RemovePostLike(ctx context.Context, userID string, postID int64) error
	GetPostCollectionStatus(ctx context.Context, userID string, postIDs []int64) (map[int64]bool, error)
	AddPostCollection(ctx context.Context, id int64, userID string, postID int64) error
	AddPostCollectionWithOutbox(ctx context.Context, id int64, userID string, postID int64, outbox *CampusNotificationOutbox) error
	RemovePostCollection(ctx context.Context, userID string, postID int64) error
	CreateReport(ctx context.Context, report *CampusForumReport) error
	ListReports(ctx context.Context, status int32, offset, limit int) ([]*CampusForumReport, int64, error)
	UpdateReportStatus(ctx context.Context, reportID int64, status int32) error
	CreateFeedback(ctx context.Context, feedback *CampusFeedback) error
	ListFeedback(ctx context.Context, status int32, offset, limit int) ([]*CampusFeedback, int64, error)
	UpdateFeedbackStatus(ctx context.Context, feedbackID int64, status int32, note string) error
	CreateNotification(ctx context.Context, notification *CampusNotification, unique bool) error
	BulkCreateNotifications(ctx context.Context, notifications []*CampusNotification) error
	CreateNotificationOutbox(ctx context.Context, outbox *CampusNotificationOutbox) error
	ClaimNotificationOutbox(ctx context.Context, limit int, lockFor time.Duration) ([]*CampusNotificationOutbox, error)
	MarkNotificationOutboxDone(ctx context.Context, id int64) error
	MarkNotificationOutboxRetry(ctx context.Context, id int64, retryCount int32, nextRetryAt *time.Time, lastError string, final bool) error
	CreateAIReplyTask(ctx context.Context, task *CampusAIReplyTask) error
	ClaimAIReplyTasks(ctx context.Context, limit int, lockFor time.Duration) ([]*CampusAIReplyTask, error)
	MarkAIReplyTaskDone(ctx context.Context, id int64, answerCommentID int64) error
	MarkAIReplyTaskRetry(ctx context.Context, id int64, retryCount int32, nextRetryAt *time.Time, lastError string, final bool) error
	CountAIRepliesToday(ctx context.Context, botUserID string) (int64, error)
	GetAIReplyOverview(ctx context.Context, botUserID string, limit int) (*CampusAIReplyOverview, error)
	ListAIReplyTasks(ctx context.Context, status string, offset, limit int) ([]*CampusAIReplyTask, int64, error)
	ResetAIReplyTask(ctx context.Context, id int64) error
	GetOpsSetting(ctx context.Context, key string) (bool, string, string, time.Time, error)
	SetOpsSetting(ctx context.Context, key, value, updatedBy string) error
	CreateAIContentAuditTask(ctx context.Context, task *CampusAIContentAuditTask) error
	ClaimAIContentAuditTasks(ctx context.Context, limit int, lockFor time.Duration) ([]*CampusAIContentAuditTask, error)
	MarkAIContentAuditTaskDone(ctx context.Context, id int64, decision, riskLevel, reason, rawResult string) error
	MarkAIContentAuditTaskRetry(ctx context.Context, id int64, retryCount int32, nextRetryAt *time.Time, lastError string, final bool) error
	GetLatestAIContentAuditTask(ctx context.Context, targetType string, targetID int64) (bool, *CampusAIContentAuditTask, error)
	GetLatestAIContentAuditTasks(ctx context.Context, targetType string, targetIDs []int64) (map[int64]*CampusAIContentAuditTask, error)
	CountPendingAIContentAuditTasks(ctx context.Context) (int64, error)
	CreateKnowledgeDocument(ctx context.Context, doc *CampusKnowledgeDocument) error
	UpdateKnowledgeDocument(ctx context.Context, doc *CampusKnowledgeDocument) error
	GetKnowledgeDocumentByID(ctx context.Context, id int64) (bool, *CampusKnowledgeDocument, error)
	ListKnowledgeDocuments(ctx context.Context, keyword, category, status string, offset, limit int) ([]*CampusKnowledgeDocument, int64, error)
	ReplaceKnowledgeChunks(ctx context.Context, documentID int64, chunks []*CampusKnowledgeChunk) error
	ListKnowledgeChunks(ctx context.Context, documentID int64, offset, limit int) ([]*CampusKnowledgeChunk, int64, error)
	CreateRAGQueryLog(ctx context.Context, item *CampusRAGQueryLog) error
	ListRAGQueryLogs(ctx context.Context, offset, limit int) ([]*CampusRAGQueryLog, int64, error)
	ListNotifications(ctx context.Context, userID, group string, offset, limit int) ([]*CampusNotification, int64, error)
	CountUnreadNotifications(ctx context.Context, userID string) (*CampusUnreadNotificationCount, error)
	MarkNotificationRead(ctx context.Context, userID string, notificationID int64) error
	MarkAllNotificationsRead(ctx context.Context, userID string) error
	ListNotificationRecipients(ctx context.Context) ([]string, error)
	IsIPBlocked(ctx context.Context, ip string) (bool, error)
	AllowCampusRequest(ctx context.Context, key string, limit int64, window time.Duration) (bool, error)
	CreateAccessLog(ctx context.Context, log *CampusAccessLog) error
	CreateAccessLogs(ctx context.Context, logs []*CampusAccessLog) error
	GetSecurityOverview(ctx context.Context) (*CampusSecurityOverview, error)
	BlockIP(ctx context.Context, block *CampusIPBlock) error
	UnblockIP(ctx context.Context, ip string) error
	CreateAuditLog(ctx context.Context, log *CampusAuditLog) error
	TrackEvent(ctx context.Context, event *TrackCampusEventInput) error
	TrackEvents(ctx context.Context, events []*TrackCampusEventInput) error
	GetAdminSummary(ctx context.Context) (*CampusAdminSummary, error)
	ReconcileCampusStats(ctx context.Context) (*CampusStatsReconcileResult, error)
	ListCampusUsers(ctx context.Context, keyword, role string, authStatus int32, offset, limit int) ([]*CampusAdminUser, int64, error)
	GetCampusOperatorRole(ctx context.Context, userID string) (string, error)
	UpsertCampusOperator(ctx context.Context, userID, role string) error
	RemoveCampusOperator(ctx context.Context, userID string) error
}

type CampusUsecase struct {
	repo              CampusRepo
	base              BaseAdapter
	core              CoreAdapter
	timetableProvider CampusTimetableProvider
	idGen             CampusIDGenerator
	authSecret        string
	assembler         *CampusPostAssembler
	recommendPool     *CampusRecommendPool
	eventBatcher      *CampusBatchProcessor[*TrackCampusEventInput]
	accessLogBatcher  *CampusBatchProcessor[*CampusAccessLog]
	knowledgeIndexer  *CampusBatchProcessor[*CampusKnowledgeDocument]
	aiReplyConfig     CampusAIReplyConfig
	aiAuditConfig     CampusAIContentAuditConfig
	rag               CampusRAGClient
	log               *log.Helper
}

type CampusAIReplyConfig struct {
	Enabled         bool
	BotUserID       string
	APIKey          string
	BaseURL         string
	Model           string
	DailyLimit      int64
	MaxOutputTokens int
	Temperature     float64
	Timeout         time.Duration
}

type CampusAIContentAuditConfig struct {
	Enabled         bool
	APIKey          string
	BaseURL         string
	Model           string
	MaxOutputTokens int
	Temperature     float64
	Timeout         time.Duration
}

func NewCampusUsecase(repo CampusRepo, base BaseAdapter, core CoreAdapter, timetableProvider CampusTimetableProvider, idGen CampusIDGenerator, rag CampusRAGClient, authSecret string, logger log.Logger) *CampusUsecase {
	if rag == nil {
		rag = &noopCampusRAGClient{}
	}
	assembler := NewCampusPostAssembler(repo, core, logger)
	recommendPool := NewCampusRecommendPool(logger)
	uc := &CampusUsecase{
		repo:              repo,
		base:              base,
		core:              core,
		timetableProvider: timetableProvider,
		idGen:             idGen,
		authSecret:        authSecret,
		assembler:         assembler,
		recommendPool:     recommendPool,
		aiReplyConfig:     loadCampusAIReplyConfig(),
		aiAuditConfig:     loadCampusAIContentAuditConfig(),
		rag:               rag,
		log:               log.NewHelper(logger),
	}
	uc.eventBatcher = NewCampusBatchProcessor("campus_event", 100, 2*time.Second, uc.persistCampusEvents, logger)
	uc.accessLogBatcher = NewCampusBatchProcessor("campus_access_log", 100, 2*time.Second, uc.persistCampusAccessLogs, logger)
	uc.knowledgeIndexer = NewCampusBatchProcessor("campus_knowledge_index", 100, time.Second, uc.processKnowledgeIndexBatch, logger)
	uc.knowledgeIndexer.timeout = 90 * time.Second
	return uc
}

func loadCampusAIContentAuditConfig() CampusAIContentAuditConfig {
	apiKey := firstNonEmpty(os.Getenv("CAMPUS_AI_AUDIT_API_KEY"), os.Getenv("CAMPUS_AI_API_KEY"), os.Getenv("DEEPSEEK_API_KEY"))
	baseURL := firstNonEmpty(os.Getenv("CAMPUS_AI_AUDIT_BASE_URL"), os.Getenv("CAMPUS_AI_BASE_URL"), "https://api.deepseek.com/chat/completions")
	model := firstNonEmpty(os.Getenv("CAMPUS_AI_AUDIT_MODEL"), os.Getenv("CAMPUS_AI_MODEL"), "deepseek-chat")
	enabled := strings.TrimSpace(apiKey) != "" && !envBoolFalse(os.Getenv("CAMPUS_AI_AUDIT_ENABLED"))
	return CampusAIContentAuditConfig{
		Enabled:         enabled,
		APIKey:          strings.TrimSpace(apiKey),
		BaseURL:         strings.TrimSpace(baseURL),
		Model:           strings.TrimSpace(model),
		MaxOutputTokens: int(envInt64("CAMPUS_AI_AUDIT_MAX_OUTPUT_TOKENS", 180)),
		Temperature:     0.1,
		Timeout:         10 * time.Second,
	}
}

func loadCampusAIReplyConfig() CampusAIReplyConfig {
	apiKey := firstNonEmpty(os.Getenv("CAMPUS_AI_API_KEY"), os.Getenv("DEEPSEEK_API_KEY"))
	botUserID := firstNonEmpty(os.Getenv("CAMPUS_EZAI_BOT_USER_ID"), os.Getenv("CAMPUS_EZAI_USER_ID"))
	baseURL := firstNonEmpty(os.Getenv("CAMPUS_AI_BASE_URL"), "https://api.deepseek.com/chat/completions")
	model := firstNonEmpty(os.Getenv("CAMPUS_AI_MODEL"), "deepseek-chat")
	enabled := strings.TrimSpace(apiKey) != "" && strings.TrimSpace(botUserID) != "" && !envBoolFalse(os.Getenv("CAMPUS_AI_EZAI_ENABLED"))
	return CampusAIReplyConfig{
		Enabled:         enabled,
		BotUserID:       strings.TrimSpace(botUserID),
		APIKey:          strings.TrimSpace(apiKey),
		BaseURL:         strings.TrimSpace(baseURL),
		Model:           strings.TrimSpace(model),
		DailyLimit:      envInt64("CAMPUS_AI_DAILY_LIMIT", 200),
		MaxOutputTokens: int(envInt64("CAMPUS_AI_MAX_OUTPUT_TOKENS", 220)),
		Temperature:     0.35,
		Timeout:         12 * time.Second,
	}
}

func envBoolFalse(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "0", "false", "off", "no", "disabled":
		return true
	default:
		return false
	}
}

func envInt64(key string, fallback int64) int64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func (uc *CampusUsecase) WechatLogin(ctx context.Context, input *WechatLoginInput) (*WechatLoginOutput, error) {
	code := strings.TrimSpace(input.Code)
	if code == "" {
		return nil, apperror.InvalidArgument("微信登录 code 不能为空")
	}

	session, err := resolveWechatSession(ctx, code)
	if err != nil {
		return nil, err
	}

	exists, identity, err := uc.repo.GetWechatIdentity(ctx, "wechat", session.OpenID)
	if err != nil {
		return nil, apperror.Internal(err, "查询微信身份失败")
	}

	var user *UserBaseInfo
	if exists {
		user, err = uc.core.GetUserBaseInfo(ctx, identity.UserID, "")
		if err != nil {
			return nil, apperror.Internal(err, "获取用户信息失败")
		}
	} else {
		accountID, userID, err := uc.createWechatAccountAndUser(ctx, session.OpenID, input.Nickname, input.Avatar)
		if err != nil {
			return nil, err
		}
		identity = &CampusWechatIdentity{
			ID:        uc.idGen.NextID(),
			Provider:  "wechat",
			OpenID:    session.OpenID,
			UnionID:   session.UnionID,
			UserID:    userID,
			AccountID: accountID,
		}
		if err := uc.repo.SaveWechatIdentity(ctx, identity); err != nil {
			return nil, apperror.Internal(err, "保存微信身份失败")
		}
		user, err = uc.core.GetUserBaseInfo(ctx, userID, "")
		if err != nil {
			return nil, apperror.Internal(err, "获取用户信息失败")
		}
	}

	profile, err := uc.ensureProfile(ctx, identity, user)
	if err != nil {
		return nil, err
	}

	token, err := sharedauth.GenerateToken(uc.authSecret, sharedauth.NewClaims(user.ID, sharedauth.DefaultTokenTTL))
	if err != nil {
		return nil, apperror.Internal(err, "生成登录态失败")
	}

	uc.trackEvent(ctx, &TrackCampusEventInput{
		UserID:    user.ID,
		EventType: "login",
		Page:      "mine",
		Channel:   "wechat",
	})

	return &WechatLoginOutput{Token: token, Profile: profile, User: user}, nil
}

func (uc *CampusUsecase) createWechatAccountAndUser(ctx context.Context, openID, nickname, avatar string) (string, string, error) {
	email := "wx_" + shortHash(openID, 24) + "@wechat.local"
	password := "Wx#" + shortHash(openID+uc.authSecret, 40)

	accountID, err := uc.base.Register(ctx, "", email, password)
	if err != nil {
		accountID, err = uc.base.CheckAccount(ctx, "", email, password)
		if err != nil {
			ok, existingAccountID, lookupErr := uc.repo.GetAccountIDByEmail(ctx, email)
			if lookupErr != nil {
				return "", "", apperror.Internal(lookupErr, "查询微信账号失败")
			}
			if !ok {
				return "", "", apperror.Internal(err, "创建微信账号失败")
			}
			accountID = existingAccountID
		}
	}

	baseUser, err := uc.core.GetUserBaseInfo(ctx, "0", accountID)
	if err == nil && baseUser != nil && baseUser.ID != "" {
		return accountID, baseUser.ID, nil
	}

	displayName := strings.TrimSpace(nickname)
	if displayName == "" {
		displayName = "微信用户"
	}
	userID, err := uc.core.CreateUser(ctx, "", email, accountID)
	if err != nil {
		return "", "", apperror.Internal(err, "创建微信用户失败")
	}
	if displayName != "微信用户" || avatar != "" {
		if err := uc.core.UpdateUserInfo(ctx, userID, displayName, displayName, avatar, "", "", 0); err != nil {
			uc.log.WithContext(ctx).Warnf("update wechat user profile failed: user_id=%s err=%v", userID, err)
		}
	}
	return accountID, userID, nil
}

func (uc *CampusUsecase) ensureProfile(ctx context.Context, identity *CampusWechatIdentity, user *UserBaseInfo) (*CampusProfile, error) {
	exists, profile, err := uc.repo.GetProfileByUserID(ctx, identity.UserID)
	if err != nil {
		return nil, apperror.Internal(err, "查询校园资料失败")
	}
	if exists {
		return profile, nil
	}
	profile = &CampusProfile{
		ID:         uc.idGen.NextID(),
		UserID:     identity.UserID,
		AccountID:  identity.AccountID,
		OpenID:     identity.OpenID,
		UnionID:    identity.UnionID,
		SchoolName: "深圳职业技术大学深汕校区",
		Mobile:     user.Mobile,
		AuthStatus: CampusAuthStatusUnverified,
	}
	if err := uc.repo.SaveProfile(ctx, profile); err != nil {
		return nil, apperror.Internal(err, "创建校园资料失败")
	}
	return profile, nil
}

func (uc *CampusUsecase) UpdateProfile(ctx context.Context, input *UpdateCampusProfileInput) (*CampusProfile, error) {
	if strings.TrimSpace(input.UserID) == "" {
		return nil, apperror.Unauthorized("请先登录")
	}
	exists, profile, err := uc.repo.GetProfileByUserID(ctx, input.UserID)
	if err != nil {
		return nil, apperror.Internal(err, "查询校园资料失败")
	}
	if !exists {
		return nil, apperror.NotFound("校园资料不存在")
	}
	profile.SchoolName = firstNonEmpty(input.SchoolName, profile.SchoolName)
	profile.StudentNo = strings.TrimSpace(input.StudentNo)
	profile.RealName = strings.TrimSpace(input.RealName)
	profile.ClassName = strings.TrimSpace(input.ClassName)
	profile.DormBuilding = strings.TrimSpace(input.DormBuilding)
	profile.RoomNo = strings.TrimSpace(input.RoomNo)
	profile.Mobile = strings.TrimSpace(input.Mobile)
	if err := uc.repo.UpdateProfile(ctx, profile); err != nil {
		return nil, apperror.Internal(err, "更新校园资料失败")
	}
	return profile, nil
}

func (uc *CampusUsecase) UpdateAvatar(ctx context.Context, userID, avatar string) (*UserBaseInfo, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, apperror.Unauthorized("请先登录")
	}
	avatar = strings.TrimSpace(avatar)
	if avatar == "" {
		return nil, apperror.InvalidArgument("头像不能为空")
	}
	if len(avatar) > 2048 {
		return nil, apperror.InvalidArgument("头像地址过长")
	}
	if err := uc.core.UpdateUserInfo(ctx, userID, "", "", avatar, "", "", 0); err != nil {
		return nil, apperror.Internal(err, "更新头像失败")
	}
	user, err := uc.core.GetUserBaseInfo(ctx, userID, "")
	if err != nil {
		return nil, apperror.Internal(err, "获取用户信息失败")
	}
	return user, nil
}

func (uc *CampusUsecase) GetProfile(ctx context.Context, userID string) (*CampusProfile, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, apperror.Unauthorized("请先登录")
	}
	exists, profile, err := uc.repo.GetProfileByUserID(ctx, userID)
	if err != nil {
		return nil, apperror.Internal(err, "查询校园资料失败")
	}
	if !exists {
		return nil, apperror.NotFound("校园资料不存在")
	}
	return profile, nil
}

func (uc *CampusUsecase) ImportTimetable(ctx context.Context, input *ImportCampusTimetableInput) (*ImportCampusTimetableOutput, error) {
	if strings.TrimSpace(input.UserID) == "" {
		return nil, apperror.Unauthorized("请先登录")
	}
	studentNo := strings.TrimSpace(input.StudentNo)
	if len([]rune(studentNo)) < 4 || len([]rune(studentNo)) > 32 {
		return nil, apperror.InvalidArgument("请输入正确的学号")
	}
	password := strings.TrimSpace(input.Password)
	if len([]rune(password)) < 4 || len([]rune(password)) > 128 {
		return nil, apperror.InvalidArgument("请输入教务系统密码")
	}
	term := normalizeCampusTerm(input.Term)
	courses, err := uc.timetableProvider.Fetch(ctx, studentNo, password, term)
	if err != nil {
		return nil, err
	}
	for _, course := range courses {
		course.ID = uc.idGen.NextID()
		course.UserID = input.UserID
		course.Term = term
		course.Source = "demo"
	}
	if err := uc.repo.ReplaceTimetableCourses(ctx, input.UserID, term, "demo", courses); err != nil {
		return nil, apperror.Internal(err, "保存课表失败")
	}
	return &ImportCampusTimetableOutput{Term: term, Courses: courses, Count: int32(len(courses))}, nil
}

func (uc *CampusUsecase) ListTimetable(ctx context.Context, input *ListCampusTimetableInput) (*ListCampusTimetableOutput, error) {
	if strings.TrimSpace(input.UserID) == "" {
		return nil, apperror.Unauthorized("请先登录")
	}
	term := normalizeCampusTerm(input.Term)
	courses, err := uc.repo.ListTimetableCourses(ctx, input.UserID, term)
	if err != nil {
		return nil, apperror.Internal(err, "获取课表失败")
	}
	return &ListCampusTimetableOutput{Term: term, Courses: courses}, nil
}

func (uc *CampusUsecase) ListCategories(ctx context.Context) ([]*CampusForumCategory, error) {
	categories, err := uc.repo.ListCategories(ctx)
	if err != nil {
		return nil, apperror.Internal(err, "获取论坛版块失败")
	}
	return categories, nil
}

func (uc *CampusUsecase) PreSignPublicImage(ctx context.Context, hash, fileType, filename string, size int64) (string, string, error) {
	hash = strings.TrimSpace(hash)
	fileType = strings.Trim(strings.ToLower(strings.TrimSpace(fileType)), ".")
	filename = strings.TrimSpace(filename)
	if hash == "" {
		return "", "", apperror.InvalidArgument("图片 hash 不能为空")
	}
	if fileType != "jpg" && fileType != "jpeg" && fileType != "png" && fileType != "webp" {
		return "", "", apperror.InvalidArgument("仅支持 jpg、png、webp 图片")
	}
	if size <= 0 || size > 5<<20 {
		return "", "", apperror.InvalidArgument("图片不能超过 5MB")
	}
	fileID, url, err := uc.base.PreSign4PublicUpload(ctx, hash, fileType, filename, size, 3600)
	if err != nil {
		return "", "", apperror.Internal(err, "创建图片上传地址失败")
	}
	return fileID, url, nil
}

func (uc *CampusUsecase) PreSignPublicVideo(ctx context.Context, hash, fileType, filename string, size int64) (string, string, error) {
	hash = strings.TrimSpace(hash)
	fileType = strings.Trim(strings.ToLower(strings.TrimSpace(fileType)), ".")
	filename = strings.TrimSpace(filename)
	if hash == "" {
		return "", "", apperror.InvalidArgument("视频 hash 不能为空")
	}
	if fileType != "mp4" && fileType != "mov" {
		return "", "", apperror.InvalidArgument("仅支持 mp4、mov 视频")
	}
	if size <= 0 || size > 20<<20 {
		return "", "", apperror.InvalidArgument("视频不能超过 20MB")
	}
	fileID, url, err := uc.base.PreSign4PublicUpload(ctx, hash, fileType, filename, size, 3600)
	if err != nil {
		return "", "", apperror.Internal(err, "创建视频上传地址失败")
	}
	return fileID, url, nil
}

func (uc *CampusUsecase) PreSignPublicKnowledgeFile(ctx context.Context, hash, fileType, filename string, size int64) (string, string, error) {
	hash = strings.TrimSpace(hash)
	fileType = normalizeKnowledgeFileType(fileType)
	filename = strings.TrimSpace(filename)
	if hash == "" {
		return "", "", apperror.InvalidArgument("文档 hash 不能为空")
	}
	switch fileType {
	case "pdf", "docx", "txt", "md":
	default:
		return "", "", apperror.InvalidArgument("仅支持 PDF、DOCX、TXT、MD 文档")
	}
	if size <= 0 || size > 20<<20 {
		return "", "", apperror.InvalidArgument("知识库文档不能超过 20MB")
	}
	fileID, url, err := uc.base.PreSign4PublicUpload(ctx, hash, fileType, filename, size, 3600)
	if err != nil {
		return "", "", apperror.Internal(err, "创建文档上传地址失败")
	}
	return fileID, url, nil
}

func (uc *CampusUsecase) PreSignCampusUpload(ctx context.Context, input *CampusUploadPresignInput) (*CampusUploadPresignOutput, error) {
	if input == nil {
		return nil, apperror.InvalidArgument("上传参数不能为空")
	}
	mediaType := strings.TrimSpace(strings.ToLower(input.MediaType))
	var fileID, uploadURL string
	var err error
	switch mediaType {
	case CampusPostMediaImage:
		fileID, uploadURL, err = uc.PreSignPublicImage(ctx, input.Hash, input.FileType, input.Filename, input.Size)
	case CampusPostMediaVideo:
		fileID, uploadURL, err = uc.PreSignPublicVideo(ctx, input.Hash, input.FileType, input.Filename, input.Size)
	default:
		return nil, apperror.InvalidArgument("上传类型无效")
	}
	if err != nil {
		return nil, err
	}
	return &CampusUploadPresignOutput{
		FileID:    fileID,
		UploadURL: uploadURL,
		Method:    http.MethodPut,
		Headers:   map[string]string{"Content-Type": campusUploadContentType(mediaType, input.FileType)},
		ExpiresIn: 3600,
	}, nil
}

func (uc *CampusUsecase) CompleteCampusUpload(ctx context.Context, input *CampusUploadCompleteInput) (*CampusUploadCompleteOutput, error) {
	if input == nil {
		return nil, apperror.InvalidArgument("上传确认参数不能为空")
	}
	mediaType := strings.TrimSpace(strings.ToLower(input.MediaType))
	fileID := strings.TrimSpace(input.FileID)
	if fileID == "" || fileID == "0" {
		return nil, apperror.InvalidArgument("file_id 无效")
	}
	var finalURL string
	var err error
	switch mediaType {
	case CampusPostMediaImage:
		finalURL, err = uc.ReportPublicImageUploaded(ctx, fileID)
	case CampusPostMediaVideo:
		finalURL, err = uc.ReportPublicVideoUploaded(ctx, fileID)
	default:
		return nil, apperror.InvalidArgument("上传类型无效")
	}
	if err != nil {
		return nil, err
	}
	objectName := ""
	if uc.base != nil {
		if info, infoErr := uc.base.GetFileInfoById(ctx, fileID); infoErr == nil && info != nil {
			objectName = info.ObjectName
		} else if infoErr != nil {
			uc.log.WithContext(ctx).Warnf("get uploaded file info failed: file_id=%s err=%v", fileID, infoErr)
		}
	}
	return &CampusUploadCompleteOutput{FileID: fileID, URL: finalURL, ObjectName: objectName}, nil
}

func (uc *CampusUsecase) ReportPublicImageUploaded(ctx context.Context, fileID string) (string, error) {
	fileID = strings.TrimSpace(fileID)
	if fileID == "" || fileID == "0" {
		return "", apperror.InvalidArgument("图片 file_id 无效")
	}
	url, err := uc.base.ReportPublicUploaded(ctx, fileID)
	if err != nil {
		return "", apperror.Internal(err, "确认图片上传失败")
	}
	return url, nil
}

func campusUploadContentType(mediaType, fileType string) string {
	fileType = strings.Trim(strings.ToLower(strings.TrimSpace(fileType)), ".")
	switch strings.TrimSpace(strings.ToLower(mediaType)) {
	case CampusPostMediaImage:
		switch fileType {
		case "jpg", "jpeg":
			return "image/jpeg"
		case "png":
			return "image/png"
		case "webp":
			return "image/webp"
		}
	case CampusPostMediaVideo:
		switch fileType {
		case "mov":
			return "video/quicktime"
		case "mp4":
			return "video/mp4"
		}
	}
	return "application/octet-stream"
}

func (uc *CampusUsecase) ReportPublicKnowledgeFileUploaded(ctx context.Context, fileID string) (string, error) {
	fileID = strings.TrimSpace(fileID)
	if fileID == "" || fileID == "0" {
		return "", apperror.InvalidArgument("文档 file_id 无效")
	}
	url, err := uc.base.ReportPublicUploaded(ctx, fileID)
	if err != nil {
		return "", apperror.Internal(err, "确认文档上传失败")
	}
	return url, nil
}

func (uc *CampusUsecase) ReportPublicVideoUploaded(ctx context.Context, fileID string) (string, error) {
	fileID = strings.TrimSpace(fileID)
	if fileID == "" || fileID == "0" {
		return "", apperror.InvalidArgument("视频 file_id 无效")
	}
	url, err := uc.base.ReportPublicUploaded(ctx, fileID)
	if err != nil {
		return "", apperror.Internal(err, "确认视频上传失败")
	}
	return url, nil
}

func (uc *CampusUsecase) CreatePost(ctx context.Context, input *CreateCampusPostInput) (*CampusForumPost, error) {
	if strings.TrimSpace(input.UserID) == "" {
		return nil, apperror.Unauthorized("请先登录")
	}
	input.CategoryCode = strings.TrimSpace(input.CategoryCode)
	if input.CategoryCode == "" {
		return nil, apperror.InvalidArgument("请选择版块")
	}
	ok, category, err := uc.repo.GetCategoryByCode(ctx, input.CategoryCode)
	if err != nil {
		return nil, apperror.Internal(err, "查询版块失败")
	}
	if !ok {
		return nil, apperror.InvalidArgument("版块不存在")
	}
	title := strings.TrimSpace(input.Title)
	content := strings.TrimSpace(input.Content)
	if len([]rune(title)) < 2 || len([]rune(title)) > 60 {
		return nil, apperror.InvalidArgument("标题需要 2-60 个字")
	}
	if len([]rune(content)) < 2 || len([]rune(content)) > 2000 {
		return nil, apperror.InvalidArgument("正文需要 2-2000 个字")
	}
	images := sanitizeImages(input.Images, 9)
	mediaType, coverURL, videoURL, err := normalizeCampusPostMedia(input.MediaType, images, input.CoverURL, input.VideoURL)
	if err != nil {
		return nil, err
	}
	if mediaType != CampusPostMediaImage {
		images = []string{}
	}
	postType := normalizeCampusPostType(input.PostType)
	extra := sanitizeCampusPostExtra(input.Extra)
	isOperator := uc.isCampusOperator(ctx, input.UserID)
	isOfficial := input.IsOfficial && isOperator
	isFeatured := input.IsFeatured && isOperator
	isPinned := input.IsPinned && isOperator
	sortWeight := int32(0)
	if isOperator {
		sortWeight = clampSortWeight(input.SortWeight)
	}
	status := CampusAuditStatusVisible
	auditReason := ""
	var auditSettings *CampusOpsAuditSettings
	if !isOperator {
		settings, err := uc.getCampusAuditSettings(ctx)
		if err != nil {
			return nil, err
		}
		auditSettings = settings
		switch settings.PostAuditMode {
		case CampusPostAuditModeManual:
			status = CampusAuditStatusPending
			auditReason = "等待人工审核"
		case CampusPostAuditModeAI:
			status = CampusAuditStatusPending
			if uc.aiAuditConfig.Enabled {
				auditReason = "等待 AI 审核"
			} else {
				auditReason = "AI 审核未启用，等待人工复核"
			}
		}
	}
	post := &CampusForumPost{
		ID:           uc.idGen.NextID(),
		CategoryCode: category.Code,
		CategoryName: category.Name,
		AuthorID:     input.UserID,
		Title:        title,
		Content:      content,
		Images:       images,
		MediaType:    mediaType,
		PostType:     postType,
		Extra:        extra,
		CoverURL:     coverURL,
		VideoURL:     videoURL,
		IsOfficial:   isOfficial,
		IsFeatured:   isFeatured,
		IsPinned:     isPinned,
		SortWeight:   sortWeight,
		Status:       status,
		AuditReason:  auditReason,
	}
	if err := uc.repo.CreatePost(ctx, post); err != nil {
		return nil, apperror.Internal(err, "发布帖子失败")
	}
	if !isOperator && status == CampusAuditStatusPending {
		if auditSettings != nil && auditSettings.PostAuditMode == CampusPostAuditModeAI && uc.aiAuditConfig.Enabled {
			if err := uc.enqueuePostAIContentAudit(ctx, post); err != nil {
				uc.log.WithContext(ctx).Warnf("queue campus post ai audit failed: post_id=%d err=%v", post.ID, err)
			}
		}
	}
	uc.trackEvent(ctx, &TrackCampusEventInput{
		UserID:     input.UserID,
		EventType:  "post_create",
		Page:       "publish",
		TargetType: "post",
		TargetID:   post.ID,
	})
	_ = uc.assembler.HydratePosts(ctx, []*CampusForumPost{post}, input.UserID)
	return post, nil
}

func (uc *CampusUsecase) ListPosts(ctx context.Context, input *ListCampusPostsInput) (*ListCampusPostsOutput, error) {
	page, size := normalizePage(input.Page, input.Size)
	query := ListCampusPostQuery{
		CategoryCode: strings.TrimSpace(input.CategoryCode),
		PostType:     strings.TrimSpace(input.PostType),
		Sort:         normalizeCampusPostSort(input.Sort, CampusPostSortRecommend),
		Keyword:      strings.TrimSpace(input.Keyword),
		Statuses:     []int32{CampusAuditStatusVisible},
		Offset:       int((page - 1) * size),
		Limit:        int(size),
	}
	posts, total, usedPool, err := uc.listPostsFromPool(ctx, query)
	if err != nil {
		uc.log.WithContext(ctx).Warnf("list campus posts from pool failed: %v", err)
	}
	if !usedPool || err != nil {
		posts, total, err = uc.repo.ListPosts(ctx, query)
	}
	if err != nil {
		return nil, apperror.Internal(err, "获取帖子列表失败")
	}
	if err := uc.assembler.HydratePosts(ctx, posts, input.CurrentUserID); err != nil {
		uc.log.WithContext(ctx).Warnf("hydrate campus posts failed: %v", err)
	}
	return &ListCampusPostsOutput{Posts: posts, Total: total}, nil
}

func (uc *CampusUsecase) ListMyPosts(ctx context.Context, input *ListCampusPostsInput) (*ListCampusPostsOutput, error) {
	if strings.TrimSpace(input.CurrentUserID) == "" {
		return nil, apperror.Unauthorized("请先登录")
	}
	page, size := normalizePage(input.Page, input.Size)
	posts, total, err := uc.repo.ListPosts(ctx, ListCampusPostQuery{
		CategoryCode:   strings.TrimSpace(input.CategoryCode),
		PostType:       strings.TrimSpace(input.PostType),
		Sort:           normalizeCampusPostSort(input.Sort, CampusPostSortNew),
		Keyword:        strings.TrimSpace(input.Keyword),
		AuthorID:       input.CurrentUserID,
		IncludeDeleted: false,
		Offset:         int((page - 1) * size),
		Limit:          int(size),
	})
	if err != nil {
		return nil, apperror.Internal(err, "获取我的帖子失败")
	}
	if err := uc.assembler.HydratePosts(ctx, posts, input.CurrentUserID); err != nil {
		uc.log.WithContext(ctx).Warnf("hydrate my campus posts failed: %v", err)
	}
	return &ListCampusPostsOutput{Posts: posts, Total: total}, nil
}

func (uc *CampusUsecase) ListMyCollections(ctx context.Context, input *ListCampusPostsInput) (*ListCampusPostsOutput, error) {
	if strings.TrimSpace(input.CurrentUserID) == "" {
		return nil, apperror.Unauthorized("请先登录")
	}
	page, size := normalizePage(input.Page, input.Size)
	posts, total, err := uc.repo.ListPosts(ctx, ListCampusPostQuery{
		CategoryCode:      strings.TrimSpace(input.CategoryCode),
		PostType:          strings.TrimSpace(input.PostType),
		Sort:              normalizeCampusPostSort(input.Sort, CampusPostSortNew),
		Keyword:           strings.TrimSpace(input.Keyword),
		CollectedByUserID: input.CurrentUserID,
		Statuses:          []int32{CampusAuditStatusVisible},
		IncludeDeleted:    false,
		Offset:            int((page - 1) * size),
		Limit:             int(size),
	})
	if err != nil {
		return nil, apperror.Internal(err, "获取我的收藏失败")
	}
	if err := uc.assembler.HydratePosts(ctx, posts, input.CurrentUserID); err != nil {
		uc.log.WithContext(ctx).Warnf("hydrate collected campus posts failed: %v", err)
	}
	return &ListCampusPostsOutput{Posts: posts, Total: total}, nil
}

func (uc *CampusUsecase) GetPublicCampusUserProfile(ctx context.Context, userID string) (*CampusPublicUserProfile, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" || userID == "0" {
		return nil, apperror.InvalidArgument("用户 ID 无效")
	}
	user, err := uc.core.GetUserBaseInfo(ctx, userID, "")
	if err != nil {
		return nil, apperror.Internal(err, "获取用户信息失败")
	}
	if user == nil || strings.TrimSpace(user.ID) == "" || strings.TrimSpace(user.ID) == "0" {
		return nil, apperror.NotFound("用户不存在")
	}
	stats, err := uc.repo.GetPublicUserPostStats(ctx, userID)
	if err != nil {
		return nil, apperror.Internal(err, "获取用户统计失败")
	}
	if stats == nil {
		stats = &CampusPublicUserStats{}
	}
	profile := &CampusPublicUserProfile{
		UserID:     user.ID,
		Name:       firstNonEmpty(user.Nickname, user.Name, "深汕同学"),
		Nickname:   user.Nickname,
		Avatar:     user.Avatar,
		IsOfficial: stats.HasOfficialPost || uc.isCampusOperator(ctx, userID),
		Stats:      stats,
	}
	if ok, campusProfile, err := uc.repo.GetProfileByUserID(ctx, userID); err == nil && ok {
		profile.SchoolName = campusProfile.SchoolName
		profile.AuthStatus = campusProfile.AuthStatus
	} else if err != nil {
		uc.log.WithContext(ctx).Warnf("load public campus profile failed: user_id=%s err=%v", userID, err)
	}
	if profile.IsOfficial {
		profile.Bio = "深汕校园e站官方账号，整理报到攻略、校园问答和重要提醒。"
	} else if profile.AuthStatus == CampusAuthStatusVerified {
		profile.Bio = "已认证的深汕校园同学"
	} else {
		profile.Bio = "深汕校园社区同学"
	}
	return profile, nil
}

func (uc *CampusUsecase) ListPublicUserPosts(ctx context.Context, input *ListCampusPostsInput) (*ListCampusPostsOutput, error) {
	authorID := strings.TrimSpace(input.AuthorID)
	if authorID == "" || authorID == "0" {
		return nil, apperror.InvalidArgument("用户 ID 无效")
	}
	page, size := normalizePage(input.Page, input.Size)
	sort := normalizeCampusPostSort(input.Sort, CampusPostSortNew)
	if sort == CampusPostSortRecommend {
		sort = CampusPostSortNew
	}
	posts, total, err := uc.repo.ListPosts(ctx, ListCampusPostQuery{
		PostType:       strings.TrimSpace(input.PostType),
		Sort:           sort,
		AuthorID:       authorID,
		Statuses:       []int32{CampusAuditStatusVisible},
		IncludeDeleted: false,
		Offset:         int((page - 1) * size),
		Limit:          int(size),
	})
	if err != nil {
		return nil, apperror.Internal(err, "获取用户帖子失败")
	}
	if err := uc.assembler.HydratePosts(ctx, posts, input.CurrentUserID); err != nil {
		uc.log.WithContext(ctx).Warnf("hydrate public user posts failed: %v", err)
	}
	return &ListCampusPostsOutput{Posts: posts, Total: total}, nil
}

func (uc *CampusUsecase) GetPost(ctx context.Context, input *GetCampusPostInput) (*CampusForumPost, error) {
	if input.PostID <= 0 {
		return nil, apperror.InvalidArgument("帖子 ID 无效")
	}
	ok, post, err := uc.repo.GetPostByID(ctx, input.PostID)
	if err != nil {
		return nil, apperror.Internal(err, "获取帖子详情失败")
	}
	if !ok {
		return nil, apperror.NotFound("帖子不存在")
	}
	if err := uc.assembler.HydratePosts(ctx, []*CampusForumPost{post}, input.CurrentUserID); err != nil {
		uc.log.WithContext(ctx).Warnf("hydrate campus post failed: %v", err)
	}
	return post, nil
}

func (uc *CampusUsecase) DeletePost(ctx context.Context, userID string, postID int64) error {
	if strings.TrimSpace(userID) == "" {
		return apperror.Unauthorized("请先登录")
	}
	if postID <= 0 {
		return apperror.InvalidArgument("帖子 ID 无效")
	}
	ok, post, err := uc.repo.GetAnyPostByID(ctx, postID)
	if err != nil {
		return apperror.Internal(err, "查询帖子失败")
	}
	if !ok {
		return apperror.NotFound("帖子不存在")
	}
	if post.AuthorID != userID && !uc.isCampusAdmin(ctx, userID) {
		return apperror.Forbidden("只能撤回自己的帖子")
	}
	if err := uc.repo.DeletePost(ctx, postID); err != nil {
		return apperror.Internal(err, "撤回帖子失败")
	}
	return nil
}

func (uc *CampusUsecase) CreateComment(ctx context.Context, input *CreateCampusCommentInput) (*CampusForumComment, error) {
	if strings.TrimSpace(input.UserID) == "" {
		return nil, apperror.Unauthorized("请先登录")
	}
	if input.PostID <= 0 {
		return nil, apperror.InvalidArgument("帖子 ID 无效")
	}
	ok, post, err := uc.repo.GetPostByID(ctx, input.PostID)
	if err != nil {
		return nil, apperror.Internal(err, "查询帖子失败")
	}
	if !ok {
		return nil, apperror.NotFound("帖子不存在")
	}
	parentID := int64(0)
	replyToCommentID := int64(0)
	replyToUserID := ""
	if input.ParentID > 0 || input.ReplyToCommentID > 0 {
		targetID := input.ReplyToCommentID
		if targetID <= 0 {
			targetID = input.ParentID
		}
		ok, target, err := uc.repo.GetCommentByID(ctx, targetID)
		if err != nil {
			return nil, apperror.Internal(err, "查询被回复评论失败")
		}
		if !ok || target.PostID != input.PostID || target.Status != CampusAuditStatusVisible {
			return nil, apperror.NotFound("被回复评论不存在")
		}
		if target.ParentID > 0 {
			parentID = target.ParentID
			replyToCommentID = target.ID
			replyToUserID = target.AuthorID
		} else {
			parentID = target.ID
			replyToCommentID = target.ID
			replyToUserID = target.AuthorID
		}
	}
	content := strings.TrimSpace(input.Content)
	if len([]rune(content)) < 1 || len([]rune(content)) > 500 {
		return nil, apperror.InvalidArgument("评论需要 1-500 个字")
	}
	comment := &CampusForumComment{
		ID:               uc.idGen.NextID(),
		PostID:           input.PostID,
		ParentID:         parentID,
		ReplyToCommentID: replyToCommentID,
		ReplyToUserID:    replyToUserID,
		AuthorID:         input.UserID,
		Content:          content,
		Images:           sanitizeImages(input.Images, 3),
		Status:           CampusAuditStatusVisible,
	}
	var notification *CampusNotification
	if parentID > 0 && replyToUserID != "" {
		notification = &CampusNotification{
			RecipientID: replyToUserID,
			ActorID:     input.UserID,
			EventType:   CampusNotificationTypeReply,
			TargetType:  "post",
			TargetID:    input.PostID,
			DedupeKey:   fmt.Sprintf("campus:reply:%d", comment.ID),
			Title:       "有人回复了你的评论",
			Content:     trimLimit(content, 80),
			LinkPage:    "post-detail",
			LinkParams:  map[string]string{"id": fmt.Sprintf("%d", input.PostID)},
		}
	} else if post != nil {
		notification = &CampusNotification{
			RecipientID: post.AuthorID,
			ActorID:     input.UserID,
			EventType:   CampusNotificationTypeComment,
			TargetType:  "post",
			TargetID:    input.PostID,
			DedupeKey:   fmt.Sprintf("campus:comment:%d", comment.ID),
			Title:       "有人评论了你的帖子",
			Content:     trimLimit(content, 80),
			LinkPage:    "post-detail",
			LinkParams:  map[string]string{"id": fmt.Sprintf("%d", input.PostID)},
		}
	}
	if err := uc.repo.CreateCommentWithOutbox(ctx, comment, uc.buildNotificationOutbox(notification, false)); err != nil {
		return nil, apperror.Internal(err, "发表评论失败")
	}
	uc.enqueueEzaiReplyTask(ctx, comment)
	uc.trackEvent(ctx, &TrackCampusEventInput{
		UserID:     input.UserID,
		EventType:  "comment_create",
		Page:       "post-detail",
		TargetType: "post",
		TargetID:   input.PostID,
	})
	_ = uc.assembler.HydrateComments(ctx, []*CampusForumComment{comment}, input.UserID)
	return comment, nil
}

func (uc *CampusUsecase) enqueueEzaiReplyTask(ctx context.Context, comment *CampusForumComment) {
	if comment == nil || !uc.aiReplyConfig.Enabled {
		return
	}
	if comment.AuthorID == uc.aiReplyConfig.BotUserID || !containsEzaiMention(comment.Content) {
		return
	}
	prompt := stripEzaiMention(comment.Content)
	if strings.TrimSpace(prompt) == "" {
		prompt = comment.Content
	}
	rootCommentID := comment.ID
	if comment.ParentID > 0 {
		rootCommentID = comment.ParentID
	}
	task := &CampusAIReplyTask{
		ID:               uc.idGen.NextID(),
		PostID:           comment.PostID,
		RootCommentID:    rootCommentID,
		TriggerCommentID: comment.ID,
		AskerID:          comment.AuthorID,
		BotUserID:        uc.aiReplyConfig.BotUserID,
		Prompt:           trimLimit(prompt, 500),
		Status:           CampusAIReplyTaskStatusPending,
	}
	if err := uc.repo.CreateAIReplyTask(ctx, task); err != nil {
		uc.log.WithContext(ctx).Warnf("queue ezai ai reply task failed: post_id=%d comment_id=%d err=%v", comment.PostID, comment.ID, err)
	}
}

func containsEzaiMention(content string) bool {
	text := strings.ToLower(strings.TrimSpace(content))
	return strings.Contains(text, "@深汕e仔") ||
		strings.Contains(text, "＠深汕e仔") ||
		strings.Contains(text, "@e仔") ||
		strings.Contains(text, "＠e仔")
}

func stripEzaiMention(content string) string {
	replacer := strings.NewReplacer("@深汕e仔", "", "＠深汕e仔", "", "@e仔", "", "＠e仔", "")
	return strings.TrimSpace(replacer.Replace(content))
}

func (uc *CampusUsecase) ListComments(ctx context.Context, input *ListCampusCommentsInput) (*ListCampusCommentsOutput, error) {
	if input.PostID <= 0 {
		return nil, apperror.InvalidArgument("帖子 ID 无效")
	}
	page, size := normalizePage(input.Page, input.Size)
	rootParentID := int64(0)
	comments, total, err := uc.repo.ListComments(ctx, ListCampusCommentQuery{
		PostID:   input.PostID,
		ParentID: &rootParentID,
		Statuses: []int32{CampusAuditStatusVisible},
		Offset:   int((page - 1) * size),
		Limit:    int(size),
	})
	if err != nil {
		return nil, apperror.Internal(err, "获取评论失败")
	}
	if err := uc.assembler.HydrateComments(ctx, comments, input.CurrentUserID); err != nil {
		uc.log.WithContext(ctx).Warnf("hydrate campus comments failed: %v", err)
	}
	if err := uc.assembler.FillPreviewReplies(ctx, comments, input.CurrentUserID); err != nil {
		uc.log.WithContext(ctx).Warnf("fill campus comment replies failed: %v", err)
	}
	return &ListCampusCommentsOutput{Comments: comments, Total: total}, nil
}

func (uc *CampusUsecase) ListCommentReplies(ctx context.Context, input *ListCampusCommentsInput) (*ListCampusCommentsOutput, error) {
	if input.CommentID <= 0 {
		return nil, apperror.InvalidArgument("评论 ID 无效")
	}
	ok, comment, err := uc.repo.GetCommentByID(ctx, input.CommentID)
	if err != nil {
		return nil, apperror.Internal(err, "查询评论失败")
	}
	if !ok || comment.Status != CampusAuditStatusVisible {
		return nil, apperror.NotFound("评论不存在")
	}
	rootID := comment.ID
	if comment.ParentID > 0 {
		rootID = comment.ParentID
	}
	page, size := normalizePage(input.Page, input.Size)
	replies, total, err := uc.repo.ListComments(ctx, ListCampusCommentQuery{
		PostID:   comment.PostID,
		ParentID: &rootID,
		Statuses: []int32{CampusAuditStatusVisible},
		Offset:   int((page - 1) * size),
		Limit:    int(size),
	})
	if err != nil {
		return nil, apperror.Internal(err, "获取回复失败")
	}
	if err := uc.assembler.HydrateComments(ctx, replies, input.CurrentUserID); err != nil {
		uc.log.WithContext(ctx).Warnf("hydrate campus replies failed: %v", err)
	}
	return &ListCampusCommentsOutput{Comments: replies, Total: total}, nil
}

func (uc *CampusUsecase) ListMyComments(ctx context.Context, input *ListCampusCommentsInput) (*ListCampusCommentsOutput, error) {
	if strings.TrimSpace(input.UserID) == "" {
		return nil, apperror.Unauthorized("请先登录")
	}
	page, size := normalizePage(input.Page, input.Size)
	comments, total, err := uc.repo.ListComments(ctx, ListCampusCommentQuery{
		AuthorID: input.UserID,
		Statuses: []int32{CampusAuditStatusVisible},
		Offset:   int((page - 1) * size),
		Limit:    int(size),
	})
	if err != nil {
		return nil, apperror.Internal(err, "获取我的评论失败")
	}
	if err := uc.assembler.HydrateComments(ctx, comments, input.UserID); err != nil {
		uc.log.WithContext(ctx).Warnf("hydrate my campus comments failed: %v", err)
	}
	if err := uc.repo.FillCommentPosts(ctx, comments); err != nil {
		uc.log.WithContext(ctx).Warnf("fill my comment posts failed: %v", err)
	}
	return &ListCampusCommentsOutput{Comments: comments, Total: total}, nil
}

func (uc *CampusUsecase) DeleteComment(ctx context.Context, userID string, commentID int64) error {
	if strings.TrimSpace(userID) == "" {
		return apperror.Unauthorized("请先登录")
	}
	if commentID <= 0 {
		return apperror.InvalidArgument("评论 ID 无效")
	}
	ok, comment, err := uc.repo.GetCommentByID(ctx, commentID)
	if err != nil {
		return apperror.Internal(err, "查询评论失败")
	}
	if !ok {
		return apperror.NotFound("评论不存在")
	}
	if comment.AuthorID != userID && !uc.isCampusAdmin(ctx, userID) {
		return apperror.Forbidden("只能撤回自己的评论")
	}
	if err := uc.repo.DeleteComment(ctx, commentID); err != nil {
		return apperror.Internal(err, "撤回评论失败")
	}
	return nil
}

func (uc *CampusUsecase) LikeComment(ctx context.Context, userID string, commentID int64) error {
	if strings.TrimSpace(userID) == "" {
		return apperror.Unauthorized("请先登录")
	}
	if commentID <= 0 {
		return apperror.InvalidArgument("评论 ID 无效")
	}
	ok, comment, err := uc.repo.GetCommentByID(ctx, commentID)
	if err != nil {
		return apperror.Internal(err, "查询评论失败")
	}
	if !ok || comment.Status != CampusAuditStatusVisible {
		return apperror.NotFound("评论不存在")
	}
	notification := uc.buildNotificationOutbox(&CampusNotification{
		RecipientID: comment.AuthorID,
		ActorID:     userID,
		EventType:   CampusNotificationTypeCommentLike,
		TargetType:  "comment",
		TargetID:    commentID,
		Title:       "有人赞了你的评论",
		Content:     trimLimit(comment.Content, 80),
		LinkPage:    "post-detail",
		LinkParams:  map[string]string{"id": fmt.Sprintf("%d", comment.PostID)},
	}, true)
	if err := uc.repo.AddCommentLikeWithOutbox(ctx, uc.idGen.NextID(), userID, commentID, notification); err != nil {
		return apperror.Internal(err, "评论点赞失败")
	}
	uc.trackEvent(ctx, &TrackCampusEventInput{
		UserID:     userID,
		EventType:  "comment_like",
		Page:       "post-detail",
		TargetType: "comment",
		TargetID:   commentID,
	})
	return nil
}

func (uc *CampusUsecase) UnlikeComment(ctx context.Context, userID string, commentID int64) error {
	if strings.TrimSpace(userID) == "" {
		return apperror.Unauthorized("请先登录")
	}
	if commentID <= 0 {
		return apperror.InvalidArgument("评论 ID 无效")
	}
	if err := uc.repo.RemoveCommentLike(ctx, userID, commentID); err != nil {
		return apperror.Internal(err, "取消评论点赞失败")
	}
	return nil
}

func (uc *CampusUsecase) LikePost(ctx context.Context, userID string, postID int64) error {
	if strings.TrimSpace(userID) == "" {
		return apperror.Unauthorized("请先登录")
	}
	if postID <= 0 {
		return apperror.InvalidArgument("帖子 ID 无效")
	}
	ok, post, err := uc.repo.GetPostByID(ctx, postID)
	if err != nil {
		return apperror.Internal(err, "查询帖子失败")
	}
	if !ok {
		return apperror.NotFound("帖子不存在")
	}
	var notification *CampusNotificationOutbox
	if post != nil {
		notification = uc.buildNotificationOutbox(&CampusNotification{
			RecipientID: post.AuthorID,
			ActorID:     userID,
			EventType:   CampusNotificationTypePostLike,
			TargetType:  "post",
			TargetID:    postID,
			Title:       "有人赞了你的帖子",
			Content:     trimLimit(post.Title, 80),
			LinkPage:    "post-detail",
			LinkParams:  map[string]string{"id": fmt.Sprintf("%d", postID)},
		}, true)
	}
	if err := uc.repo.AddPostLikeWithOutbox(ctx, uc.idGen.NextID(), userID, postID, notification); err != nil {
		return apperror.Internal(err, "点赞失败")
	}
	uc.trackEvent(ctx, &TrackCampusEventInput{
		UserID:     userID,
		EventType:  "like",
		Page:       "post-detail",
		TargetType: "post",
		TargetID:   postID,
	})
	return nil
}

func (uc *CampusUsecase) UnlikePost(ctx context.Context, userID string, postID int64) error {
	if strings.TrimSpace(userID) == "" {
		return apperror.Unauthorized("请先登录")
	}
	if postID <= 0 {
		return apperror.InvalidArgument("帖子 ID 无效")
	}
	if err := uc.repo.RemovePostLike(ctx, userID, postID); err != nil {
		return apperror.Internal(err, "取消点赞失败")
	}
	return nil
}

func (uc *CampusUsecase) CollectPost(ctx context.Context, userID string, postID int64) error {
	if strings.TrimSpace(userID) == "" {
		return apperror.Unauthorized("请先登录")
	}
	if postID <= 0 {
		return apperror.InvalidArgument("帖子 ID 无效")
	}
	ok, post, err := uc.repo.GetPostByID(ctx, postID)
	if err != nil {
		return apperror.Internal(err, "查询帖子失败")
	}
	if !ok {
		return apperror.NotFound("帖子不存在")
	}
	var notification *CampusNotificationOutbox
	if post != nil {
		notification = uc.buildNotificationOutbox(&CampusNotification{
			RecipientID: post.AuthorID,
			ActorID:     userID,
			EventType:   CampusNotificationTypePostCollect,
			TargetType:  "post",
			TargetID:    postID,
			Title:       "有人收藏了你的帖子",
			Content:     trimLimit(post.Title, 80),
			LinkPage:    "post-detail",
			LinkParams:  map[string]string{"id": fmt.Sprintf("%d", postID)},
		}, true)
	}
	if err := uc.repo.AddPostCollectionWithOutbox(ctx, uc.idGen.NextID(), userID, postID, notification); err != nil {
		return apperror.Internal(err, "收藏失败")
	}
	uc.trackEvent(ctx, &TrackCampusEventInput{
		UserID:     userID,
		EventType:  "collect",
		Page:       "post-detail",
		TargetType: "post",
		TargetID:   postID,
	})
	return nil
}

func (uc *CampusUsecase) UncollectPost(ctx context.Context, userID string, postID int64) error {
	if strings.TrimSpace(userID) == "" {
		return apperror.Unauthorized("请先登录")
	}
	if postID <= 0 {
		return apperror.InvalidArgument("帖子 ID 无效")
	}
	if err := uc.repo.RemovePostCollection(ctx, userID, postID); err != nil {
		return apperror.Internal(err, "取消收藏失败")
	}
	return nil
}

func (uc *CampusUsecase) ListNotifications(ctx context.Context, input *ListCampusNotificationsInput) (*ListCampusNotificationsOutput, error) {
	if strings.TrimSpace(input.UserID) == "" {
		return nil, apperror.Unauthorized("请先登录")
	}
	page, size := normalizePage(input.Page, input.Size)
	group := normalizeCampusNotificationGroup(input.Type)
	notifications, total, err := uc.repo.ListNotifications(ctx, input.UserID, group, int((page-1)*size), int(size))
	if err != nil {
		return nil, apperror.Internal(err, "获取消息失败")
	}
	return &ListCampusNotificationsOutput{Notifications: notifications, Total: total}, nil
}

func (uc *CampusUsecase) CountUnreadNotifications(ctx context.Context, userID string) (*CampusUnreadNotificationCount, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, apperror.Unauthorized("请先登录")
	}
	count, err := uc.repo.CountUnreadNotifications(ctx, userID)
	if err != nil {
		return nil, apperror.Internal(err, "获取未读消息失败")
	}
	return count, nil
}

func (uc *CampusUsecase) MarkNotificationRead(ctx context.Context, userID string, notificationID int64) error {
	if strings.TrimSpace(userID) == "" {
		return apperror.Unauthorized("请先登录")
	}
	if notificationID <= 0 {
		return apperror.InvalidArgument("消息 ID 无效")
	}
	if err := uc.repo.MarkNotificationRead(ctx, userID, notificationID); err != nil {
		return apperror.Internal(err, "标记消息已读失败")
	}
	return nil
}

func (uc *CampusUsecase) MarkAllNotificationsRead(ctx context.Context, userID string) error {
	if strings.TrimSpace(userID) == "" {
		return apperror.Unauthorized("请先登录")
	}
	if err := uc.repo.MarkAllNotificationsRead(ctx, userID); err != nil {
		return apperror.Internal(err, "标记全部已读失败")
	}
	return nil
}

func (uc *CampusUsecase) AdminCreateSystemNotification(ctx context.Context, input *CreateCampusAdminNotificationInput) (int64, error) {
	if !uc.isCampusOperator(ctx, input.UserID) {
		return 0, apperror.Forbidden("没有后台权限")
	}
	title := trimLimit(input.Title, 80)
	content := trimLimit(input.Content, 500)
	if len([]rune(title)) < 2 {
		return 0, apperror.InvalidArgument("通知标题至少 2 个字")
	}
	if len([]rune(content)) < 2 {
		return 0, apperror.InvalidArgument("通知内容至少 2 个字")
	}
	if strings.TrimSpace(input.Audience) != "" && strings.TrimSpace(input.Audience) != "all_users" {
		return 0, apperror.InvalidArgument("通知范围暂只支持全体用户")
	}
	taskID := uc.idGen.NextID()
	linkPage := firstNonEmpty(input.LinkPage, "community")
	outbox := &CampusNotificationOutbox{
		ID:         taskID,
		ActorID:    input.UserID,
		EventType:  CampusNotificationTypeSystem,
		TargetType: "system",
		TargetID:   taskID,
		DedupeKey:  fmt.Sprintf("campus:system-task:%d", taskID),
		Title:      title,
		Content:    content,
		LinkPage:   trimLimit(linkPage, 64),
		LinkParams: sanitizeTrackExtra(input.LinkParams),
		Audience:   "all_users",
		Status:     CampusNotificationOutboxStatusPending,
	}
	if err := uc.repo.CreateNotificationOutbox(ctx, outbox); err != nil {
		return 0, apperror.Internal(err, "创建系统通知任务失败")
	}
	return taskID, nil
}

func (uc *CampusUsecase) buildNotificationOutbox(notification *CampusNotification, unique bool) *CampusNotificationOutbox {
	if notification == nil {
		return nil
	}
	if strings.TrimSpace(notification.RecipientID) == "" || notification.RecipientID == "0" || notification.RecipientID == notification.ActorID {
		return nil
	}
	outboxID := uc.idGen.NextID()
	dedupeKey := strings.TrimSpace(notification.DedupeKey)
	if dedupeKey == "" && unique {
		dedupeKey = campusNotificationDedupeKey(notification)
	}
	if dedupeKey == "" {
		dedupeKey = fmt.Sprintf("campus:notification-outbox:%d", outboxID)
	}
	return &CampusNotificationOutbox{
		ID:          outboxID,
		RecipientID: strings.TrimSpace(notification.RecipientID),
		ActorID:     strings.TrimSpace(notification.ActorID),
		EventType:   trimLimit(notification.EventType, 32),
		TargetType:  trimLimit(notification.TargetType, 32),
		TargetID:    notification.TargetID,
		DedupeKey:   dedupeKey,
		Title:       trimLimit(notification.Title, 80),
		Content:     trimLimit(notification.Content, 500),
		LinkPage:    trimLimit(firstNonEmpty(notification.LinkPage, "post-detail"), 64),
		LinkParams:  sanitizeTrackExtra(notification.LinkParams),
		Status:      CampusNotificationOutboxStatusPending,
	}
}

func campusNotificationDedupeKey(notification *CampusNotification) string {
	if notification == nil {
		return ""
	}
	recipientID := strings.TrimSpace(notification.RecipientID)
	actorID := strings.TrimSpace(notification.ActorID)
	eventType := strings.TrimSpace(notification.EventType)
	targetType := strings.TrimSpace(notification.TargetType)
	if recipientID == "" || recipientID == "0" || actorID == "" || actorID == "0" || eventType == "" || targetType == "" || notification.TargetID == 0 {
		return ""
	}
	return fmt.Sprintf("campus:%s:%s:%s:%s:%d", recipientID, actorID, eventType, targetType, notification.TargetID)
}

func (uc *CampusUsecase) ProcessPendingNotificationOutbox(ctx context.Context, limit int) error {
	if limit <= 0 {
		limit = 100
	}
	items, err := uc.repo.ClaimNotificationOutbox(ctx, limit, 30*time.Second)
	if err != nil {
		return apperror.Internal(err, "领取通知任务失败")
	}
	var firstErr error
	for _, item := range items {
		if item == nil {
			continue
		}
		if err := uc.processNotificationOutboxItem(ctx, item); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			uc.markNotificationOutboxRetry(ctx, item, err)
			continue
		}
		if err := uc.repo.MarkNotificationOutboxDone(ctx, item.ID); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			uc.log.WithContext(ctx).Warnf("mark campus notification outbox done failed: id=%d err=%v", item.ID, err)
		}
	}
	return firstErr
}

func (uc *CampusUsecase) processNotificationOutboxItem(ctx context.Context, item *CampusNotificationOutbox) error {
	if item.EventType == CampusNotificationTypeSystem {
		return uc.deliverSystemNotificationOutbox(ctx, item)
	}
	return uc.deliverInteractionNotificationOutbox(ctx, item)
}

func (uc *CampusUsecase) deliverSystemNotificationOutbox(ctx context.Context, item *CampusNotificationOutbox) error {
	if strings.TrimSpace(item.Audience) != "" && item.Audience != "all_users" {
		return fmt.Errorf("unsupported notification audience: %s", item.Audience)
	}
	recipients, err := uc.repo.ListNotificationRecipients(ctx)
	if err != nil {
		return err
	}
	for _, recipientID := range recipients {
		recipientID = strings.TrimSpace(recipientID)
		if recipientID == "" || recipientID == "0" {
			continue
		}
		notification := &CampusNotification{
			ID:          uc.idGen.NextID(),
			RecipientID: recipientID,
			ActorID:     item.ActorID,
			EventType:   CampusNotificationTypeSystem,
			TargetType:  firstNonEmpty(item.TargetType, "system"),
			TargetID:    item.ID,
			DedupeKey:   fmt.Sprintf("campus:system:%d:%s", item.ID, recipientID),
			Title:       item.Title,
			Content:     item.Content,
			LinkPage:    firstNonEmpty(item.LinkPage, "community"),
			LinkParams:  sanitizeTrackExtra(item.LinkParams),
		}
		if err := uc.repo.CreateNotification(ctx, notification, true); err != nil {
			return err
		}
	}
	return nil
}

func (uc *CampusUsecase) deliverInteractionNotificationOutbox(ctx context.Context, item *CampusNotificationOutbox) error {
	if strings.TrimSpace(item.RecipientID) == "" || item.RecipientID == "0" || item.RecipientID == item.ActorID {
		return nil
	}
	dedupeKey := strings.TrimSpace(item.DedupeKey)
	if dedupeKey == "" {
		dedupeKey = fmt.Sprintf("campus:notification-outbox:%d", item.ID)
	}
	notification := &CampusNotification{
		ID:          uc.idGen.NextID(),
		RecipientID: item.RecipientID,
		ActorID:     item.ActorID,
		EventType:   item.EventType,
		TargetType:  item.TargetType,
		TargetID:    item.TargetID,
		DedupeKey:   dedupeKey,
		Title:       item.Title,
		Content:     item.Content,
		LinkPage:    firstNonEmpty(item.LinkPage, "post-detail"),
		LinkParams:  sanitizeTrackExtra(item.LinkParams),
	}
	return uc.repo.CreateNotification(ctx, notification, true)
}

func (uc *CampusUsecase) markNotificationOutboxRetry(ctx context.Context, item *CampusNotificationOutbox, processErr error) {
	retryCount := item.RetryCount + 1
	final := retryCount >= campusNotificationOutboxMaxRetry
	var nextRetryAt *time.Time
	if !final {
		next := time.Now().Add(campusNotificationOutboxBackoff(retryCount))
		nextRetryAt = &next
	}
	if err := uc.repo.MarkNotificationOutboxRetry(ctx, item.ID, retryCount, nextRetryAt, trimLimit(processErr.Error(), 500), final); err != nil {
		uc.log.WithContext(ctx).Warnf("mark campus notification outbox retry failed: id=%d err=%v", item.ID, err)
	}
}

func campusNotificationOutboxBackoff(retryCount int32) time.Duration {
	switch retryCount {
	case 1:
		return 10 * time.Second
	case 2:
		return 30 * time.Second
	case 3:
		return 2 * time.Minute
	case 4:
		return 10 * time.Minute
	default:
		return 30 * time.Minute
	}
}

func (uc *CampusUsecase) ProcessPendingAIReplyTasks(ctx context.Context, limit int) error {
	if !uc.aiReplyConfig.Enabled {
		return nil
	}
	if limit <= 0 {
		limit = 20
	}
	items, err := uc.repo.ClaimAIReplyTasks(ctx, limit, 45*time.Second)
	if err != nil {
		return apperror.Internal(err, "领取 e仔回复任务失败")
	}
	for _, item := range items {
		if item == nil {
			continue
		}
		if err := uc.processAIReplyTask(ctx, item); err != nil {
			uc.log.WithContext(ctx).Warnf("process ezai ai reply task failed: id=%d err=%v", item.ID, err)
			uc.markAIReplyTaskRetry(ctx, item, err)
		}
	}
	return nil
}

func (uc *CampusUsecase) enqueuePostAIContentAudit(ctx context.Context, post *CampusForumPost) error {
	if post == nil || post.ID <= 0 {
		return nil
	}
	return uc.repo.CreateAIContentAuditTask(ctx, &CampusAIContentAuditTask{
		ID:         uc.idGen.NextID(),
		TargetType: "post",
		TargetID:   post.ID,
		Status:     CampusAIContentAuditTaskStatusPending,
	})
}

func (uc *CampusUsecase) ProcessPendingAIContentAuditTasks(ctx context.Context, limit int) error {
	if !uc.aiAuditConfig.Enabled {
		return nil
	}
	if limit <= 0 {
		limit = 10
	}
	tasks, err := uc.repo.ClaimAIContentAuditTasks(ctx, limit, 45*time.Second)
	if err != nil {
		return apperror.Internal(err, "领取 AI 审核任务失败")
	}
	var firstErr error
	for _, task := range tasks {
		if task == nil {
			continue
		}
		if err := uc.processAIContentAuditTask(ctx, task); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			uc.log.WithContext(ctx).Warnf("process campus ai audit task failed: id=%d err=%v", task.ID, err)
			uc.markAIContentAuditTaskRetry(ctx, task, err)
		}
	}
	return firstErr
}

func (uc *CampusUsecase) processAIContentAuditTask(ctx context.Context, task *CampusAIContentAuditTask) error {
	if task == nil {
		return nil
	}
	if task.TargetType != "post" {
		return fmt.Errorf("unsupported ai audit target type: %s", task.TargetType)
	}
	ok, post, err := uc.repo.GetAnyPostByID(ctx, task.TargetID)
	if err != nil {
		return err
	}
	if !ok || post == nil {
		return fmt.Errorf("post not found")
	}
	if post.Status != CampusAuditStatusPending {
		return uc.repo.MarkAIContentAuditTaskDone(ctx, task.ID, CampusAIContentAuditDecisionReview, "none", "内容已不处于待审核状态", "")
	}
	result, raw, err := uc.auditPostWithAI(ctx, post)
	if err != nil {
		return err
	}
	decision := normalizeAIContentAuditDecision(result.Decision)
	if decision == "" {
		decision = CampusAIContentAuditDecisionReview
	}
	riskLevel := normalizeAIContentAuditRiskLevel(result.RiskLevel)
	reason := trimLimit(firstNonEmpty(result.Reason, "AI 审核建议人工复核"), 240)
	switch decision {
	case CampusAIContentAuditDecisionPass:
		if err := uc.repo.UpdatePostStatus(ctx, post.ID, CampusAuditStatusVisible, "AI 审核通过"); err != nil {
			return err
		}
		uc.notifyPostAuditResult(ctx, post, true, "你的帖子已通过审核")
	case CampusAIContentAuditDecisionReject:
		decision = CampusAIContentAuditDecisionReview
		if err := uc.repo.UpdatePostStatus(ctx, post.ID, CampusAuditStatusPending, reason); err != nil {
			return err
		}
	default:
		if err := uc.repo.UpdatePostStatus(ctx, post.ID, CampusAuditStatusPending, reason); err != nil {
			return err
		}
	}
	_ = uc.repo.CreateAuditLog(ctx, &CampusAuditLog{
		ID:         uc.idGen.NextID(),
		TargetType: "post",
		TargetID:   post.ID,
		UserID:     post.AuthorID,
		Provider:   "ai",
		Result:     decision,
		Reason:     reason,
	})
	return uc.repo.MarkAIContentAuditTaskDone(ctx, task.ID, decision, riskLevel, reason, raw)
}

type aiContentAuditResult struct {
	Decision  string `json:"decision"`
	RiskLevel string `json:"risk_level"`
	Reason    string `json:"reason"`
}

func (uc *CampusUsecase) auditPostWithAI(ctx context.Context, post *CampusForumPost) (*aiContentAuditResult, string, error) {
	cfg := uc.aiAuditConfig
	taskCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()
	systemPrompt := "你是校园社区内容安全审核员。请审核同学发布的帖子是否适合在校园社区展示。重点识别违法违规、辱骂人身攻击、隐私泄露、广告诈骗、色情暴力、代考代课、危险物品、骚扰引战和虚假信息。只输出 JSON，不要输出多余文字。decision 只能是 pass/review/reject；risk_level 只能是 low/medium/high；reason 用中文 40 字以内。低风险正常校园分享给 pass；不确定或可能违规给 review；明显严重违规才给 reject。"
	userPrompt := fmt.Sprintf("帖子类型：%s\n标题：%s\n正文：%s\n图片数量：%d\n视频：%t",
		post.PostType,
		trimLimit(post.Title, 120),
		trimLimit(post.Content, 1800),
		len(post.Images),
		strings.TrimSpace(post.VideoURL) != "",
	)
	body, _ := json.Marshal(map[string]interface{}{
		"model": cfg.Model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"max_tokens":  cfg.MaxOutputTokens,
		"temperature": cfg.Temperature,
	})
	req, err := http.NewRequestWithContext(taskCtx, http.MethodPost, cfg.BaseURL, bytes.NewReader(body))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, string(raw), fmt.Errorf("ai audit api status=%d body=%s", resp.StatusCode, trimLimit(string(raw), 300))
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, string(raw), err
	}
	if len(out.Choices) == 0 {
		return nil, string(raw), fmt.Errorf("ai audit api returned empty choices")
	}
	content := strings.TrimSpace(out.Choices[0].Message.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)
	var result aiContentAuditResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return &aiContentAuditResult{Decision: CampusAIContentAuditDecisionReview, RiskLevel: "medium", Reason: "AI 返回格式异常，需人工复核"}, string(raw), nil
	}
	return &result, string(raw), nil
}

func (uc *CampusUsecase) markAIContentAuditTaskRetry(ctx context.Context, item *CampusAIContentAuditTask, processErr error) {
	if item == nil {
		return
	}
	nextRetryCount := item.RetryCount + 1
	final := nextRetryCount >= campusAIContentAuditTaskMaxRetry
	var nextRetryAt *time.Time
	if !final {
		next := time.Now().Add(time.Duration(nextRetryCount*nextRetryCount) * time.Minute)
		nextRetryAt = &next
	}
	if err := uc.repo.MarkAIContentAuditTaskRetry(ctx, item.ID, nextRetryCount, nextRetryAt, processErr.Error(), final); err != nil {
		uc.log.WithContext(ctx).Warnf("mark campus ai audit task retry failed: id=%d err=%v", item.ID, err)
	}
	if final && item.TargetType == "post" {
		_ = uc.repo.UpdatePostStatus(ctx, item.TargetID, CampusAuditStatusPending, "AI 审核失败，等待人工复核")
	}
}

func normalizeAIContentAuditDecision(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case CampusAIContentAuditDecisionPass:
		return CampusAIContentAuditDecisionPass
	case CampusAIContentAuditDecisionReview:
		return CampusAIContentAuditDecisionReview
	case CampusAIContentAuditDecisionReject:
		return CampusAIContentAuditDecisionReject
	default:
		return ""
	}
}

func normalizeAIContentAuditRiskLevel(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "low", "medium", "high":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "medium"
	}
}

func (uc *CampusUsecase) notifyPostAuditResult(ctx context.Context, post *CampusForumPost, passed bool, title string) {
	if post == nil || strings.TrimSpace(post.AuthorID) == "" {
		return
	}
	content := "审核通过后，其他同学就能在首页和主页看到这条内容。"
	linkPage := "post-detail"
	linkParams := map[string]string{"id": fmt.Sprintf("%d", post.ID)}
	if !passed {
		content = firstNonEmpty(post.AuditReason, "你的帖子未通过审核，可以修改后重新发布。")
		linkPage = "my-posts"
		linkParams = map[string]string{}
	}
	outbox := &CampusNotificationOutbox{
		ID:          uc.idGen.NextID(),
		RecipientID: post.AuthorID,
		ActorID:     "0",
		EventType:   CampusNotificationTypeSystem,
		TargetType:  "post",
		TargetID:    post.ID,
		DedupeKey:   fmt.Sprintf("campus:post-audit:%d:%t", post.ID, passed),
		Title:       title,
		Content:     trimLimit(content, 80),
		LinkPage:    linkPage,
		LinkParams:  linkParams,
		Status:      CampusNotificationOutboxStatusPending,
	}
	if err := uc.repo.CreateNotificationOutbox(ctx, outbox); err != nil {
		uc.log.WithContext(ctx).Warnf("queue post audit notification failed: post_id=%d err=%v", post.ID, err)
	}
}

func (uc *CampusUsecase) processAIReplyTask(ctx context.Context, task *CampusAIReplyTask) error {
	if task == nil {
		return nil
	}
	count, err := uc.repo.CountAIRepliesToday(ctx, uc.aiReplyConfig.BotUserID)
	if err != nil {
		return err
	}
	if count >= uc.aiReplyConfig.DailyLimit {
		next := nextLocalDayStart(time.Now()).Add(5 * time.Minute)
		return uc.repo.MarkAIReplyTaskRetry(ctx, task.ID, task.RetryCount, &next, "daily ai reply limit reached", false)
	}
	if ok, existing, err := uc.repo.GetAnyCommentByID(ctx, task.ID); err != nil {
		return err
	} else if ok && existing != nil {
		return uc.repo.MarkAIReplyTaskDone(ctx, task.ID, existing.ID)
	}
	ok, post, err := uc.repo.GetPostByID(ctx, task.PostID)
	if err != nil {
		return err
	}
	if !ok || post == nil {
		return fmt.Errorf("post not found")
	}
	ok, trigger, err := uc.repo.GetCommentByID(ctx, task.TriggerCommentID)
	if err != nil {
		return err
	}
	if !ok || trigger == nil || trigger.Status != CampusAuditStatusVisible {
		return fmt.Errorf("trigger comment not visible")
	}
	answer, err := uc.generateEzaiAnswer(ctx, task, post, trigger, task.Prompt)
	if err != nil {
		return err
	}
	answer = sanitizeEzaiAnswer(answer)
	if answer == "" {
		answer = "这个问题 e仔暂时不能确定，建议先以学校官方渠道为准；如果你愿意，也可以在评论区补充更多信息。"
	}
	parentID := trigger.ID
	if trigger.ParentID > 0 {
		parentID = trigger.ParentID
	}
	comment := &CampusForumComment{
		ID:               task.ID,
		PostID:           task.PostID,
		ParentID:         parentID,
		ReplyToCommentID: trigger.ID,
		ReplyToUserID:    task.AskerID,
		AuthorID:         uc.aiReplyConfig.BotUserID,
		Content:          answer,
		Status:           CampusAuditStatusVisible,
	}
	notification := uc.buildNotificationOutbox(&CampusNotification{
		RecipientID: task.AskerID,
		ActorID:     uc.aiReplyConfig.BotUserID,
		EventType:   CampusNotificationTypeReply,
		TargetType:  "post",
		TargetID:    task.PostID,
		DedupeKey:   fmt.Sprintf("campus:ezai-reply:%d", task.ID),
		Title:       "深汕e仔回复了你",
		Content:     trimLimit(answer, 80),
		LinkPage:    "post-detail",
		LinkParams:  map[string]string{"id": fmt.Sprintf("%d", task.PostID)},
	}, false)
	if err := uc.repo.CreateCommentWithOutbox(ctx, comment, notification); err != nil {
		return err
	}
	return uc.repo.MarkAIReplyTaskDone(ctx, task.ID, comment.ID)
}

func (uc *CampusUsecase) generateEzaiAnswer(ctx context.Context, task *CampusAIReplyTask, post *CampusForumPost, trigger *CampusForumComment, prompt string) (string, error) {
	cfg := uc.aiReplyConfig
	taskCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()
	query := trimLimit(firstNonEmpty(prompt, trigger.Content), 500)
	postContext := buildEzaiPostContext(post)
	ragResp, ragDuration, ragErr := uc.queryKnowledgeForEzai(taskCtx, query, postContext)
	knowledgeContext := buildEzaiKnowledgeContext(ragResp)
	userPrompt := fmt.Sprintf("帖子上下文：\n%s\n\n同学在评论区说：%s\n同学真正想问：%s",
		postContext,
		trimLimit(trigger.Content, 500),
		query,
	)
	if knowledgeContext != "" {
		userPrompt += "\n\n可参考的校园资料：\n" + knowledgeContext
	} else if ragResp != nil && ragResp.NeedKnowledge {
		userPrompt += "\n\n知识库检索结果：当前资料里没有高置信度命中。若问题涉及报到、宿舍、交通、校园网、军训等学校事实，请不要编造。"
	}
	systemPrompt := "你是“深汕e仔”，深汕校园e站的官方内容小伙伴。用户是在某个帖子评论区 @ 你，所以很多问题里的“这个帖子、楼主、上面、图里、这是什么意思”都指向帖子上下文。请先读帖子标题和正文，能基于帖子解释、总结、提醒时，就直接围绕帖子回答；只有涉及报到、宿舍、交通、校园网、军训等学校事实时，才结合知识库资料。请用温和、简洁、像校园学长学姐一样的语气回复。不要冒充学校官方，不确定时明确说以学校官方渠道为准。不要输出联系方式、广告、敏感隐私，不要编造政策。回复控制在120字以内。"
	if knowledgeContext != "" {
		systemPrompt += " 若提供了校园资料，优先依据资料回答；可以自然提到“资料里写到/目前资料显示”，但不要生硬罗列引用。"
	}
	body, _ := json.Marshal(map[string]interface{}{
		"model": cfg.Model,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": systemPrompt,
			},
			{"role": "user", "content": userPrompt},
		},
		"max_tokens":  cfg.MaxOutputTokens,
		"temperature": cfg.Temperature,
	})
	req, err := http.NewRequestWithContext(taskCtx, http.MethodPost, cfg.BaseURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("ai api status=%d body=%s", resp.StatusCode, trimLimit(string(raw), 300))
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("ai api returned empty choices")
	}
	answer := out.Choices[0].Message.Content
	uc.recordRAGQueryLog(ctx, task, post, query, ragResp, answer, ragDuration, ragErr)
	return answer, nil
}

func buildEzaiPostContext(post *CampusForumPost) string {
	if post == nil {
		return "无帖子上下文"
	}
	var builder strings.Builder
	title := trimLimit(strings.TrimSpace(post.Title), 120)
	if title == "" {
		title = "未填写标题"
	}
	builder.WriteString("标题：" + title)
	postType := strings.TrimSpace(post.PostType)
	if postType != "" {
		builder.WriteString("\n类型：" + postType)
	}
	category := firstNonEmpty(strings.TrimSpace(post.CategoryName), strings.TrimSpace(post.CategoryCode))
	if category != "" {
		builder.WriteString("\n版块：" + category)
	}
	content := strings.TrimSpace(post.Content)
	if content != "" {
		builder.WriteString("\n正文：" + trimLimit(content, 900))
	} else {
		builder.WriteString("\n正文：无文字内容")
	}
	if len(post.Images) > 0 {
		builder.WriteString(fmt.Sprintf("\n图片：%d 张。你不能直接看图片细节，只能根据标题和正文判断。", len(post.Images)))
	}
	if post.MediaType == CampusPostMediaVideo || strings.TrimSpace(post.VideoURL) != "" {
		builder.WriteString("\n视频：有视频内容。你不能直接观看视频，只能根据标题和正文判断。")
	}
	return builder.String()
}

func (uc *CampusUsecase) queryKnowledgeForEzai(ctx context.Context, query, postContext string) (*CampusRAGQueryResponse, int64, error) {
	if uc.rag == nil || strings.TrimSpace(query) == "" {
		return nil, 0, nil
	}
	start := time.Now()
	resp, err := uc.rag.Query(ctx, &CampusRAGQueryRequest{
		Query:   query,
		Context: trimLimit(postContext, 1000),
		TopK:    5,
	})
	duration := time.Since(start).Milliseconds()
	if err != nil {
		uc.log.WithContext(ctx).Warnf("ezai rag query failed: %v", err)
		return nil, duration, err
	}
	return resp, duration, nil
}

func (uc *CampusUsecase) recordRAGQueryLog(ctx context.Context, task *CampusAIReplyTask, post *CampusForumPost, query string, ragResp *CampusRAGQueryResponse, answer string, durationMs int64, ragErr error) {
	if task == nil {
		return
	}
	item := &CampusRAGQueryLog{
		ID:               uc.idGen.NextID(),
		UserID:           task.AskerID,
		PostID:           task.PostID,
		TriggerCommentID: task.TriggerCommentID,
		Query:            query,
		Answer:           trimLimit(answer, 1000),
		Model:            uc.aiReplyConfig.Model,
		DurationMs:       durationMs,
		CreatedAt:        time.Now(),
	}
	if post != nil {
		item.PostID = post.ID
	}
	if ragResp != nil {
		item.NeedKnowledge = ragResp.NeedKnowledge
		item.Confidence = ragResp.Confidence
		item.HitChunks = ragResp.Chunks
	}
	if ragErr != nil {
		item.ErrorMessage = trimLimit(ragErr.Error(), 1000)
	}
	if err := uc.repo.CreateRAGQueryLog(ctx, item); err != nil {
		uc.log.WithContext(ctx).Warnf("create rag query log failed: %v", err)
	}
}

func buildEzaiKnowledgeContext(resp *CampusRAGQueryResponse) string {
	if resp == nil || !resp.NeedKnowledge || resp.Confidence < 0.52 || len(resp.Chunks) == 0 {
		return ""
	}
	var builder strings.Builder
	count := 0
	for _, chunk := range resp.Chunks {
		if chunk == nil || strings.TrimSpace(chunk.Content) == "" {
			continue
		}
		count++
		builder.WriteString(fmt.Sprintf("[%d] 标题：%s；来源：%s；内容：%s\n", count, trimLimit(chunk.Title, 80), trimLimit(chunk.Source, 80), trimLimit(chunk.Content, 420)))
		if count >= 4 {
			break
		}
	}
	return strings.TrimSpace(builder.String())
}

func (uc *CampusUsecase) markAIReplyTaskRetry(ctx context.Context, item *CampusAIReplyTask, processErr error) {
	retryCount := item.RetryCount + 1
	final := retryCount >= campusAIReplyTaskMaxRetry
	var nextRetryAt *time.Time
	if !final {
		next := time.Now().Add(time.Duration(retryCount*retryCount) * 5 * time.Second)
		nextRetryAt = &next
	}
	if err := uc.repo.MarkAIReplyTaskRetry(ctx, item.ID, retryCount, nextRetryAt, trimLimit(processErr.Error(), 500), final); err != nil {
		uc.log.WithContext(ctx).Warnf("mark ezai ai reply task retry failed: id=%d err=%v", item.ID, err)
	}
}

func sanitizeEzaiAnswer(answer string) string {
	text := strings.TrimSpace(answer)
	text = strings.Trim(text, "\"'` \n\t")
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\n\n", "\n")
	return trimLimit(text, 220)
}

func normalizeAIReplyTaskStatus(status string) string {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case CampusAIReplyTaskStatusPending, CampusAIReplyTaskStatusProcessing, CampusAIReplyTaskStatusDone, CampusAIReplyTaskStatusFailed:
		return strings.TrimSpace(strings.ToLower(status))
	default:
		return ""
	}
}

func normalizeKnowledgeDocumentStatus(status string) string {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case CampusKnowledgeDocumentStatusDraft:
		return CampusKnowledgeDocumentStatusDraft
	case CampusKnowledgeDocumentStatusIndexing:
		return CampusKnowledgeDocumentStatusIndexing
	case CampusKnowledgeDocumentStatusActive:
		return CampusKnowledgeDocumentStatusActive
	case CampusKnowledgeDocumentStatusDisabled:
		return CampusKnowledgeDocumentStatusDisabled
	case CampusKnowledgeDocumentStatusFailed:
		return CampusKnowledgeDocumentStatusFailed
	default:
		return ""
	}
}

func formatRAGTime(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func sameOptionalTime(a, b *time.Time) bool {
	if a == nil || a.IsZero() {
		return b == nil || b.IsZero()
	}
	if b == nil || b.IsZero() {
		return false
	}
	return a.Equal(*b)
}

func normalizeKnowledgeContentType(contentType string) string {
	switch strings.TrimSpace(strings.ToLower(contentType)) {
	case CampusKnowledgeContentTypeFile:
		return CampusKnowledgeContentTypeFile
	case CampusKnowledgeContentTypeText:
		return CampusKnowledgeContentTypeText
	default:
		return CampusKnowledgeContentTypeText
	}
}

func normalizeKnowledgeCategory(category string) string {
	category = strings.TrimSpace(strings.ToLower(category))
	switch category {
	case "registration", "dorm", "traffic", "timetable", "network", "express", "military", "club", "lost", "platform", "policy", "life", "study":
		return category
	case "":
		return "general"
	default:
		return trimLimit(category, 32)
	}
}

func normalizeKnowledgeFileType(fileType string) string {
	fileType = strings.Trim(strings.ToLower(strings.TrimSpace(fileType)), ".")
	switch fileType {
	case "pdf", "docx", "txt", "md", "markdown":
		if fileType == "markdown" {
			return "md"
		}
		return fileType
	default:
		return fileType
	}
}

func nextLocalDayStart(now time.Time) time.Time {
	y, m, d := now.Local().Date()
	return time.Date(y, m, d+1, 0, 0, 0, 0, now.Local().Location())
}

func (uc *CampusUsecase) ReportContent(ctx context.Context, input *ReportCampusContentInput) error {
	if strings.TrimSpace(input.UserID) == "" {
		return apperror.Unauthorized("请先登录")
	}
	targetType := normalizeCampusTargetType(input.TargetType)
	if targetType == "" {
		return apperror.InvalidArgument("举报对象无效")
	}
	if input.TargetID <= 0 {
		return apperror.InvalidArgument("举报对象 ID 无效")
	}
	if targetType == "post" {
		ok, _, err := uc.repo.GetPostByID(ctx, input.TargetID)
		if err != nil {
			return apperror.Internal(err, "查询帖子失败")
		}
		if !ok {
			return apperror.NotFound("帖子不存在")
		}
	} else {
		ok, comment, err := uc.repo.GetCommentByID(ctx, input.TargetID)
		if err != nil {
			return apperror.Internal(err, "查询评论失败")
		}
		if !ok || comment.Status != CampusAuditStatusVisible {
			return apperror.NotFound("评论不存在")
		}
	}
	reason := firstNonEmpty(input.Reason, "其他")
	detail := strings.TrimSpace(input.Detail)
	if len([]rune(reason)) > 60 {
		return apperror.InvalidArgument("举报原因不能超过 60 个字")
	}
	if len([]rune(detail)) > 300 {
		return apperror.InvalidArgument("举报说明不能超过 300 个字")
	}
	if err := uc.repo.CreateReport(ctx, &CampusForumReport{
		ID:         uc.idGen.NextID(),
		TargetType: targetType,
		TargetID:   input.TargetID,
		ReporterID: input.UserID,
		Reason:     reason,
		Detail:     detail,
		Status:     CampusAuditStatusPending,
	}); err != nil {
		return apperror.Internal(err, "提交举报失败")
	}
	return nil
}

func (uc *CampusUsecase) ListModerationPosts(ctx context.Context, input *ListCampusModerationInput) (*ListCampusPostsOutput, error) {
	if !uc.isCampusOperator(ctx, input.UserID) {
		return nil, apperror.Forbidden("没有审核权限")
	}
	page, size := normalizePage(input.Page, input.Size)
	status := input.Status
	if status < CampusAuditStatusPending || status > CampusAuditStatusDeleted {
		status = CampusAuditStatusPending
	}
	posts, total, err := uc.repo.ListPosts(ctx, ListCampusPostQuery{
		Statuses: []int32{status},
		Sort:     "new",
		Offset:   int((page - 1) * size),
		Limit:    int(size),
	})
	if err != nil {
		return nil, apperror.Internal(err, "获取审核帖子失败")
	}
	if err := uc.assembler.HydratePosts(ctx, posts, input.UserID); err != nil {
		uc.log.WithContext(ctx).Warnf("hydrate moderation posts failed: %v", err)
	}
	return &ListCampusPostsOutput{Posts: posts, Total: total}, nil
}

func (uc *CampusUsecase) ListModerationComments(ctx context.Context, input *ListCampusModerationInput) (*ListCampusCommentsOutput, error) {
	if !uc.isCampusOperator(ctx, input.UserID) {
		return nil, apperror.Forbidden("没有审核权限")
	}
	page, size := normalizePage(input.Page, input.Size)
	status := input.Status
	if status < CampusAuditStatusPending || status > CampusAuditStatusDeleted {
		status = CampusAuditStatusPending
	}
	comments, total, err := uc.repo.ListComments(ctx, ListCampusCommentQuery{
		Statuses: []int32{status},
		Offset:   int((page - 1) * size),
		Limit:    int(size),
	})
	if err != nil {
		return nil, apperror.Internal(err, "获取审核评论失败")
	}
	if err := uc.assembler.HydrateComments(ctx, comments, input.UserID); err != nil {
		uc.log.WithContext(ctx).Warnf("hydrate moderation comments failed: %v", err)
	}
	return &ListCampusCommentsOutput{Comments: comments, Total: total}, nil
}

func (uc *CampusUsecase) ReviewContent(ctx context.Context, input *ReviewCampusContentInput) error {
	if !uc.isCampusOperator(ctx, input.UserID) {
		return apperror.Forbidden("没有审核权限")
	}
	targetType := normalizeCampusTargetType(input.TargetType)
	if targetType == "" {
		return apperror.InvalidArgument("审核对象无效")
	}
	status := CampusAuditStatusVisible
	action := strings.TrimSpace(strings.ToLower(input.Action))
	switch action {
	case "approve", "pass", "visible":
		status = CampusAuditStatusVisible
	case "reject":
		status = CampusAuditStatusRejected
	case "delete":
		status = CampusAuditStatusDeleted
	default:
		return apperror.InvalidArgument("审核动作无效")
	}
	reason := strings.TrimSpace(input.Reason)
	if targetType == "post" {
		ok, post, err := uc.repo.GetAnyPostByID(ctx, input.TargetID)
		if err != nil {
			return apperror.Internal(err, "查询帖子失败")
		}
		if !ok {
			return apperror.NotFound("帖子不存在")
		}
		if err := uc.repo.UpdatePostStatus(ctx, input.TargetID, status, reason); err != nil {
			return apperror.Internal(err, "审核帖子失败")
		}
		if post != nil {
			post.Status = status
			post.AuditReason = reason
			if status == CampusAuditStatusVisible {
				uc.notifyPostAuditResult(ctx, post, true, "你的帖子已通过审核")
			} else if status == CampusAuditStatusRejected {
				uc.notifyPostAuditResult(ctx, post, false, "你的帖子未通过审核")
			}
		}
	} else {
		ok, _, err := uc.repo.GetAnyCommentByID(ctx, input.TargetID)
		if err != nil {
			return apperror.Internal(err, "查询评论失败")
		}
		if !ok {
			return apperror.NotFound("评论不存在")
		}
		if err := uc.repo.UpdateCommentStatus(ctx, input.TargetID, status, reason); err != nil {
			return apperror.Internal(err, "审核评论失败")
		}
	}
	_ = uc.repo.CreateAuditLog(ctx, &CampusAuditLog{
		ID:         uc.idGen.NextID(),
		TargetType: targetType,
		TargetID:   input.TargetID,
		UserID:     input.UserID,
		Provider:   "manual",
		Result:     action,
		Reason:     reason,
	})
	return nil
}

func (uc *CampusUsecase) AdminSummary(ctx context.Context, userID string) (*CampusAdminSummary, error) {
	if !uc.isCampusOperator(ctx, userID) {
		return nil, apperror.Forbidden("没有后台权限")
	}
	summary, err := uc.repo.GetAdminSummary(ctx)
	if err != nil {
		return nil, apperror.Internal(err, "获取数据总览失败")
	}
	if pending, err := uc.repo.CountPendingAIContentAuditTasks(ctx); err == nil {
		summary.PendingAIAudits = pending
	}
	return summary, nil
}

func (uc *CampusUsecase) AdminGetAuditSettings(ctx context.Context, input *GetCampusAuditSettingsInput) (*CampusOpsAuditSettings, error) {
	if !uc.isCampusOperator(ctx, input.UserID) {
		return nil, apperror.Forbidden("没有后台权限")
	}
	return uc.getCampusAuditSettings(ctx)
}

func (uc *CampusUsecase) AdminUpdateAuditSettings(ctx context.Context, input *UpdateCampusAuditSettingsInput) (*CampusOpsAuditSettings, error) {
	if !uc.isCampusOperator(ctx, input.UserID) {
		return nil, apperror.Forbidden("没有后台权限")
	}
	mode := normalizeCampusPostAuditMode(input.PostAuditMode)
	if mode == "" {
		return nil, apperror.InvalidArgument("审核模式无效")
	}
	if err := uc.repo.SetOpsSetting(ctx, "post_audit_mode", mode, input.UserID); err != nil {
		return nil, apperror.Internal(err, "保存审核设置失败")
	}
	return uc.getCampusAuditSettings(ctx)
}

func (uc *CampusUsecase) getCampusAuditSettings(ctx context.Context) (*CampusOpsAuditSettings, error) {
	ok, value, updatedBy, updatedAt, err := uc.repo.GetOpsSetting(ctx, "post_audit_mode")
	if err != nil {
		return nil, apperror.Internal(err, "读取审核设置失败")
	}
	mode := CampusPostAuditModeOff
	if ok {
		mode = normalizeCampusPostAuditMode(value)
		if mode == "" {
			mode = CampusPostAuditModeOff
		}
	}
	return &CampusOpsAuditSettings{
		PostAuditMode: mode,
		AIEnabled:     uc.aiAuditConfig.Enabled,
		UpdatedBy:     updatedBy,
		UpdatedAt:     updatedAt,
	}, nil
}

func normalizeCampusPostAuditMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", CampusPostAuditModeOff:
		return CampusPostAuditModeOff
	case CampusPostAuditModeManual:
		return CampusPostAuditModeManual
	case CampusPostAuditModeAI:
		return CampusPostAuditModeAI
	default:
		return ""
	}
}

func (uc *CampusUsecase) AdminReconcileCampusStats(ctx context.Context, userID string) (*CampusStatsReconcileResult, error) {
	if !uc.isCampusOperator(ctx, userID) {
		return nil, apperror.Forbidden("没有后台权限")
	}
	result, err := uc.repo.ReconcileCampusStats(ctx)
	if err != nil {
		return nil, apperror.Internal(err, "计数对账失败")
	}
	_ = uc.repo.CreateAuditLog(ctx, &CampusAuditLog{
		ID:         uc.idGen.NextID(),
		TargetType: "campus_stats",
		TargetID:   0,
		UserID:     userID,
		Provider:   "manual",
		Result:     "reconcile",
		Reason:     fmt.Sprintf("updated_posts=%d updated_comments=%d", result.UpdatedPosts, result.UpdatedComments),
	})
	return result, nil
}

func (uc *CampusUsecase) RunCampusStatsReconcile(ctx context.Context) (*CampusStatsReconcileResult, error) {
	result, err := uc.repo.ReconcileCampusStats(ctx)
	if err != nil {
		return nil, apperror.Internal(err, "计数对账失败")
	}
	return result, nil
}

func (uc *CampusUsecase) AdminListPosts(ctx context.Context, input *ListCampusAdminPostsInput) (*ListCampusPostsOutput, error) {
	if !uc.isCampusOperator(ctx, input.UserID) {
		return nil, apperror.Forbidden("没有后台权限")
	}
	page, size := normalizePage(input.Page, input.Size)
	statuses := []int32{}
	if input.Status >= CampusAuditStatusPending && input.Status <= CampusAuditStatusDeleted {
		statuses = []int32{input.Status}
	}
	onlyOfficial, onlyFeatured, onlyPinned, onlyReported := parseOpsFilter(input.OpsFilter)
	posts, total, err := uc.repo.ListPosts(ctx, ListCampusPostQuery{
		CategoryCode:   strings.TrimSpace(input.CategoryCode),
		PostType:       strings.TrimSpace(input.PostType),
		Sort:           normalizeCampusPostSort(input.Sort, CampusPostSortNew),
		Keyword:        strings.TrimSpace(input.Keyword),
		Statuses:       statuses,
		IncludeDeleted: true,
		OnlyOfficial:   onlyOfficial,
		OnlyFeatured:   onlyFeatured,
		OnlyPinned:     onlyPinned,
		OnlyReported:   onlyReported,
		Offset:         int((page - 1) * size),
		Limit:          int(size),
	})
	if err != nil {
		return nil, apperror.Internal(err, "获取后台帖子失败")
	}
	if err := uc.assembler.HydratePosts(ctx, posts, input.UserID); err != nil {
		uc.log.WithContext(ctx).Warnf("hydrate admin posts failed: %v", err)
	}
	uc.attachLatestAIAuditTasks(ctx, posts)
	return &ListCampusPostsOutput{Posts: posts, Total: total}, nil
}

func (uc *CampusUsecase) attachLatestAIAuditTasks(ctx context.Context, posts []*CampusForumPost) {
	if len(posts) == 0 {
		return
	}
	ids := make([]int64, 0, len(posts))
	for _, post := range posts {
		if post != nil && post.ID > 0 {
			ids = append(ids, post.ID)
		}
	}
	if len(ids) == 0 {
		return
	}
	tasks, err := uc.repo.GetLatestAIContentAuditTasks(ctx, "post", ids)
	if err != nil {
		uc.log.WithContext(ctx).Warnf("load latest ai content audit tasks failed: %v", err)
		return
	}
	for _, post := range posts {
		if post == nil {
			continue
		}
		task := tasks[post.ID]
		if task == nil {
			continue
		}
		post.AIAuditStatus = task.Status
		post.AIAuditRisk = task.RiskLevel
		post.AIAuditDecision = task.Decision
		post.AIAuditReason = task.Reason
		post.AIAuditError = task.LastError
	}
}

func (uc *CampusUsecase) AdminBatchPosts(ctx context.Context, input *BatchCampusAdminPostsInput) (*BatchCampusAdminPostsOutput, error) {
	if !uc.isCampusOperator(ctx, input.UserID) {
		return nil, apperror.Forbidden("没有后台权限")
	}
	action := strings.TrimSpace(strings.ToLower(input.Action))
	if action == "" {
		return nil, apperror.InvalidArgument("请选择批量操作")
	}
	seen := map[int64]struct{}{}
	postIDs := make([]int64, 0, len(input.PostIDs))
	for _, id := range input.PostIDs {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		postIDs = append(postIDs, id)
		if len(postIDs) >= 100 {
			break
		}
	}
	if len(postIDs) == 0 {
		return nil, apperror.InvalidArgument("请选择要操作的内容")
	}
	var updated int32
	for _, postID := range postIDs {
		ok, existing, err := uc.repo.GetAnyPostByID(ctx, postID)
		if err != nil {
			return nil, apperror.Internal(err, "查询帖子失败")
		}
		if !ok {
			continue
		}
		next := *existing
		next.AuditReason = existing.AuditReason
		switch action {
		case "pin":
			next.IsPinned = true
		case "unpin":
			next.IsPinned = false
		case "feature":
			next.IsFeatured = true
		case "unfeature":
			next.IsFeatured = false
		case "official":
			next.IsOfficial = true
		case "unofficial":
			next.IsOfficial = false
		case "visible":
			next.Status = CampusAuditStatusVisible
			next.AuditReason = ""
		case "delete":
			next.Status = CampusAuditStatusDeleted
			next.AuditReason = "运营下架"
		case "set_weight":
			next.SortWeight = clampSortWeight(input.SortWeight)
		default:
			return nil, apperror.InvalidArgument("批量操作无效")
		}
		if err := uc.repo.UpdatePostByAdmin(ctx, &next); err != nil {
			return nil, apperror.Internal(err, "批量更新内容失败")
		}
		if action == "visible" && existing.Status != CampusAuditStatusVisible {
			uc.notifyPostAuditResult(ctx, &next, true, "你的帖子已通过审核")
		}
		updated++
	}
	return &BatchCampusAdminPostsOutput{UpdatedCount: updated}, nil
}

func (uc *CampusUsecase) AdminCreatePost(ctx context.Context, input *CreateCampusPostInput) (*CampusForumPost, error) {
	if !uc.isCampusOperator(ctx, input.UserID) {
		return nil, apperror.Forbidden("没有运营发帖权限")
	}
	return uc.CreatePost(ctx, input)
}

func (uc *CampusUsecase) AdminUpdatePost(ctx context.Context, input *UpdateCampusAdminPostInput) (*CampusForumPost, error) {
	if !uc.isCampusOperator(ctx, input.UserID) {
		return nil, apperror.Forbidden("没有后台权限")
	}
	ok, existing, err := uc.repo.GetAnyPostByID(ctx, input.PostID)
	if err != nil {
		return nil, apperror.Internal(err, "查询帖子失败")
	}
	if !ok {
		return nil, apperror.NotFound("帖子不存在")
	}
	categoryCode := strings.TrimSpace(input.CategoryCode)
	if categoryCode == "" {
		categoryCode = existing.CategoryCode
	}
	ok, category, err := uc.repo.GetCategoryByCode(ctx, categoryCode)
	if err != nil {
		return nil, apperror.Internal(err, "查询版块失败")
	}
	if !ok {
		return nil, apperror.InvalidArgument("版块不存在")
	}
	title := firstNonEmpty(input.Title, existing.Title)
	content := firstNonEmpty(input.Content, existing.Content)
	if len([]rune(title)) < 2 || len([]rune(title)) > 60 {
		return nil, apperror.InvalidArgument("标题需要 2-60 个字")
	}
	if len([]rune(content)) < 2 || len([]rune(content)) > 2000 {
		return nil, apperror.InvalidArgument("正文需要 2-2000 个字")
	}
	images := input.Images
	if images == nil {
		images = existing.Images
	}
	images = sanitizeImages(images, 9)
	mediaType, coverURL, videoURL, err := normalizeCampusPostMedia(firstNonEmpty(input.MediaType, existing.MediaType), images, firstNonEmpty(input.CoverURL, existing.CoverURL), firstNonEmpty(input.VideoURL, existing.VideoURL))
	if err != nil {
		return nil, err
	}
	if mediaType != CampusPostMediaImage {
		images = []string{}
	}
	status := input.Status
	if status < CampusAuditStatusPending || status > CampusAuditStatusDeleted {
		status = existing.Status
	}
	post := &CampusForumPost{
		ID:             existing.ID,
		CategoryCode:   category.Code,
		CategoryName:   category.Name,
		AuthorID:       existing.AuthorID,
		Title:          title,
		Content:        content,
		Images:         images,
		MediaType:      mediaType,
		PostType:       normalizeCampusPostType(firstNonEmpty(input.PostType, existing.PostType)),
		Extra:          mergeCampusPostExtra(existing.Extra, input.Extra),
		CoverURL:       coverURL,
		VideoURL:       videoURL,
		IsOfficial:     input.IsOfficial,
		IsFeatured:     input.IsFeatured,
		IsPinned:       input.IsPinned,
		SortWeight:     clampSortWeight(input.SortWeight),
		Status:         status,
		AuditReason:    strings.TrimSpace(input.AuditReason),
		LikeCount:      existing.LikeCount,
		CommentCount:   existing.CommentCount,
		CollectedCount: existing.CollectedCount,
		CreatedAt:      existing.CreatedAt,
	}
	if err := uc.repo.UpdatePostByAdmin(ctx, post); err != nil {
		return nil, apperror.Internal(err, "更新帖子失败")
	}
	if existing.Status != post.Status {
		if post.Status == CampusAuditStatusVisible {
			uc.notifyPostAuditResult(ctx, post, true, "你的帖子已通过审核")
		} else if post.Status == CampusAuditStatusRejected {
			uc.notifyPostAuditResult(ctx, post, false, "你的帖子未通过审核")
		}
	}
	_ = uc.assembler.HydratePosts(ctx, []*CampusForumPost{post}, input.UserID)
	return post, nil
}

func (uc *CampusUsecase) AdminDeletePost(ctx context.Context, userID string, postID int64) error {
	if !uc.isCampusOperator(ctx, userID) {
		return apperror.Forbidden("没有后台权限")
	}
	if postID <= 0 {
		return apperror.InvalidArgument("帖子 ID 无效")
	}
	if err := uc.repo.DeletePost(ctx, postID); err != nil {
		return apperror.Internal(err, "删除帖子失败")
	}
	return nil
}

func (uc *CampusUsecase) AdminListComments(ctx context.Context, input *ListCampusAdminCommentsInput) (*ListCampusCommentsOutput, error) {
	if !uc.isCampusOperator(ctx, input.UserID) {
		return nil, apperror.Forbidden("没有后台权限")
	}
	page, size := normalizePage(input.Page, input.Size)
	statuses := []int32{}
	if input.Status >= CampusAuditStatusPending && input.Status <= CampusAuditStatusDeleted {
		statuses = []int32{input.Status}
	}
	comments, total, err := uc.repo.ListComments(ctx, ListCampusCommentQuery{
		PostID:         input.PostID,
		Statuses:       statuses,
		IncludeDeleted: true,
		Offset:         int((page - 1) * size),
		Limit:          int(size),
	})
	if err != nil {
		return nil, apperror.Internal(err, "获取后台评论失败")
	}
	if err := uc.assembler.HydrateComments(ctx, comments, input.UserID); err != nil {
		uc.log.WithContext(ctx).Warnf("hydrate admin comments failed: %v", err)
	}
	if err := uc.repo.FillCommentPosts(ctx, comments); err != nil {
		uc.log.WithContext(ctx).Warnf("fill admin comment posts failed: %v", err)
	}
	return &ListCampusCommentsOutput{Comments: comments, Total: total}, nil
}

func (uc *CampusUsecase) AdminAIReplyOverview(ctx context.Context, userID string) (*CampusAIReplyOverview, error) {
	if !uc.isCampusOperator(ctx, userID) {
		return nil, apperror.Forbidden("没有后台权限")
	}
	overview, err := uc.repo.GetAIReplyOverview(ctx, uc.aiReplyConfig.BotUserID, 5)
	if err != nil {
		return nil, apperror.Internal(err, "获取 e仔回复状态失败")
	}
	if overview == nil {
		overview = &CampusAIReplyOverview{}
	}
	overview.Enabled = uc.aiReplyConfig.Enabled
	overview.BotUserID = uc.aiReplyConfig.BotUserID
	overview.Model = uc.aiReplyConfig.Model
	overview.BaseURL = uc.aiReplyConfig.BaseURL
	overview.DailyLimit = uc.aiReplyConfig.DailyLimit
	if uc.aiReplyConfig.BotUserID != "" {
		if user, err := uc.core.GetUserBaseInfo(ctx, uc.aiReplyConfig.BotUserID, ""); err == nil && user != nil {
			overview.BotReady = true
			overview.BotName = firstNonEmpty(user.Nickname, user.Name, "深汕e仔")
			overview.BotAvatar = user.Avatar
		} else if err != nil {
			uc.log.WithContext(ctx).Warnf("load ezai bot user failed: user_id=%s err=%v", uc.aiReplyConfig.BotUserID, err)
		}
		if used, err := uc.repo.CountAIRepliesToday(ctx, uc.aiReplyConfig.BotUserID); err == nil {
			overview.TodayUsed = used
		} else {
			uc.log.WithContext(ctx).Warnf("count ezai replies today failed: %v", err)
		}
	}
	if uc.rag != nil {
		if health, err := uc.rag.Health(ctx); err == nil {
			overview.RAGHealth = health
		} else {
			overview.RAGHealth = &CampusRAGHealth{Status: "unavailable", Qdrant: "unknown", LastError: trimLimit(err.Error(), 200)}
		}
	}
	return overview, nil
}

func (uc *CampusUsecase) AdminListAIReplyTasks(ctx context.Context, input *ListCampusAIReplyTasksInput) (*ListCampusAIReplyTasksOutput, error) {
	if !uc.isCampusOperator(ctx, input.UserID) {
		return nil, apperror.Forbidden("没有后台权限")
	}
	page, size := normalizePage(input.Page, input.Size)
	status := normalizeAIReplyTaskStatus(input.Status)
	tasks, total, err := uc.repo.ListAIReplyTasks(ctx, status, int((page-1)*size), int(size))
	if err != nil {
		return nil, apperror.Internal(err, "获取 e仔回复任务失败")
	}
	return &ListCampusAIReplyTasksOutput{Tasks: tasks, Total: total}, nil
}

func (uc *CampusUsecase) AdminRetryAIReplyTask(ctx context.Context, input *RetryCampusAIReplyTaskInput) error {
	if !uc.isCampusOperator(ctx, input.UserID) {
		return apperror.Forbidden("没有后台权限")
	}
	if input.TaskID <= 0 {
		return apperror.InvalidArgument("任务 ID 无效")
	}
	if err := uc.repo.ResetAIReplyTask(ctx, input.TaskID); err != nil {
		return apperror.Internal(err, "重试 e仔回复任务失败")
	}
	return nil
}

func (uc *CampusUsecase) AdminListKnowledgeDocuments(ctx context.Context, input *ListCampusKnowledgeDocumentsInput) (*ListCampusKnowledgeDocumentsOutput, error) {
	if !uc.isCampusOperator(ctx, input.UserID) {
		return nil, apperror.Forbidden("没有后台权限")
	}
	page, size := normalizePage(input.Page, input.Size)
	status := normalizeKnowledgeDocumentStatus(input.Status)
	docs, total, err := uc.repo.ListKnowledgeDocuments(ctx, strings.TrimSpace(input.Keyword), normalizeKnowledgeCategory(input.Category), status, int((page-1)*size), int(size))
	if err != nil {
		return nil, apperror.Internal(err, "获取知识库文档失败")
	}
	return &ListCampusKnowledgeDocumentsOutput{Documents: docs, Total: total}, nil
}

func (uc *CampusUsecase) AdminCreateKnowledgeDocument(ctx context.Context, input *CreateCampusKnowledgeDocumentInput) (*CampusKnowledgeDocument, error) {
	if !uc.isCampusOperator(ctx, input.UserID) {
		return nil, apperror.Forbidden("没有后台权限")
	}
	if input == nil {
		return nil, apperror.InvalidArgument("知识库文档不能为空")
	}
	contentType := normalizeKnowledgeContentType(input.ContentType)
	title := trimLimit(input.Title, 120)
	source := trimLimit(firstNonEmpty(input.Source, "运营录入"), 120)
	category := normalizeKnowledgeCategory(input.Category)
	if len([]rune(title)) < 2 {
		return nil, apperror.InvalidArgument("标题至少 2 个字")
	}
	if contentType == CampusKnowledgeContentTypeText && len([]rune(strings.TrimSpace(input.RawContent))) < 10 {
		return nil, apperror.InvalidArgument("手动录入内容至少 10 个字")
	}
	if contentType == CampusKnowledgeContentTypeFile && strings.TrimSpace(input.FileURL) == "" {
		return nil, apperror.InvalidArgument("请先上传知识库文档")
	}
	doc := &CampusKnowledgeDocument{
		ID:           uc.idGen.NextID(),
		Title:        title,
		Source:       source,
		Category:     category,
		ContentType:  contentType,
		FileURL:      trimLimit(input.FileURL, 1024),
		FileID:       trimLimit(input.FileID, 64),
		FileType:     normalizeKnowledgeFileType(input.FileType),
		RawContent:   trimLimit(input.RawContent, 20000),
		Status:       CampusKnowledgeDocumentStatusIndexing,
		ParseStatus:  "indexing",
		UploadedBy:   strings.TrimSpace(input.UserID),
		EffectiveAt:  input.EffectiveAt,
		ExpiredAt:    input.ExpiredAt,
		ErrorMessage: "",
	}
	if strings.TrimSpace(input.Status) == CampusKnowledgeDocumentStatusDraft {
		doc.Status = CampusKnowledgeDocumentStatusDraft
		doc.ParseStatus = "draft"
	}
	if err := uc.repo.CreateKnowledgeDocument(ctx, doc); err != nil {
		return nil, apperror.Internal(err, "创建知识库文档失败")
	}
	if doc.Status != CampusKnowledgeDocumentStatusDraft {
		uc.enqueueKnowledgeIndex(ctx, doc)
	}
	return doc, nil
}

func (uc *CampusUsecase) AdminUpdateKnowledgeDocument(ctx context.Context, input *UpdateCampusKnowledgeDocumentInput) (*CampusKnowledgeDocument, error) {
	if !uc.isCampusOperator(ctx, input.UserID) {
		return nil, apperror.Forbidden("没有后台权限")
	}
	ok, doc, err := uc.repo.GetKnowledgeDocumentByID(ctx, input.DocumentID)
	if err != nil {
		return nil, apperror.Internal(err, "查询知识库文档失败")
	}
	if !ok || doc == nil {
		return nil, apperror.NotFound("知识库文档不存在")
	}
	wasActive := doc.Status == CampusKnowledgeDocumentStatusActive
	needsReindex := false
	if strings.TrimSpace(input.Title) != "" {
		next := trimLimit(input.Title, 120)
		if next != doc.Title {
			doc.Title = next
			needsReindex = true
		}
	}
	if strings.TrimSpace(input.Source) != "" {
		next := trimLimit(input.Source, 120)
		if next != doc.Source {
			doc.Source = next
			needsReindex = true
		}
	}
	if strings.TrimSpace(input.Category) != "" {
		next := normalizeKnowledgeCategory(input.Category)
		if next != doc.Category {
			doc.Category = next
			needsReindex = true
		}
	}
	if !sameOptionalTime(doc.EffectiveAt, input.EffectiveAt) {
		needsReindex = true
	}
	if !sameOptionalTime(doc.ExpiredAt, input.ExpiredAt) {
		needsReindex = true
	}
	doc.EffectiveAt = input.EffectiveAt
	doc.ExpiredAt = input.ExpiredAt
	status := normalizeKnowledgeDocumentStatus(input.Status)
	if status != "" && status != doc.Status {
		switch status {
		case CampusKnowledgeDocumentStatusActive:
			doc.Status = CampusKnowledgeDocumentStatusIndexing
			doc.ParseStatus = "indexing"
			doc.ErrorMessage = ""
			if err := uc.repo.UpdateKnowledgeDocument(ctx, doc); err != nil {
				return nil, apperror.Internal(err, "更新知识库文档失败")
			}
			uc.enqueueKnowledgeIndex(ctx, doc)
			return doc, nil
		case CampusKnowledgeDocumentStatusDisabled:
			doc.Status = CampusKnowledgeDocumentStatusDisabled
			doc.ParseStatus = "disabled"
			doc.ErrorMessage = ""
			doc.ChunkCount = 0
			_ = uc.rag.DeleteDocument(ctx, doc.ID)
			if err := uc.repo.ReplaceKnowledgeChunks(ctx, doc.ID, nil); err != nil {
				return nil, apperror.Internal(err, "下架知识片段失败")
			}
		case CampusKnowledgeDocumentStatusDraft:
			doc.Status = CampusKnowledgeDocumentStatusDraft
			doc.ParseStatus = "draft"
			doc.ErrorMessage = ""
			doc.ChunkCount = 0
			_ = uc.rag.DeleteDocument(ctx, doc.ID)
			if err := uc.repo.ReplaceKnowledgeChunks(ctx, doc.ID, nil); err != nil {
				return nil, apperror.Internal(err, "下架知识片段失败")
			}
		default:
			return nil, apperror.InvalidArgument("知识库文档状态无效")
		}
	} else if wasActive && needsReindex {
		doc.Status = CampusKnowledgeDocumentStatusIndexing
		doc.ParseStatus = "indexing"
		doc.ErrorMessage = ""
		if err := uc.repo.UpdateKnowledgeDocument(ctx, doc); err != nil {
			return nil, apperror.Internal(err, "更新知识库文档失败")
		}
		uc.enqueueKnowledgeIndex(ctx, doc)
		return doc, nil
	}
	if err := uc.repo.UpdateKnowledgeDocument(ctx, doc); err != nil {
		return nil, apperror.Internal(err, "更新知识库文档失败")
	}
	return doc, nil
}

func (uc *CampusUsecase) AdminReindexKnowledgeDocument(ctx context.Context, userID string, documentID int64) (*CampusKnowledgeDocument, error) {
	if !uc.isCampusOperator(ctx, userID) {
		return nil, apperror.Forbidden("没有后台权限")
	}
	ok, doc, err := uc.repo.GetKnowledgeDocumentByID(ctx, documentID)
	if err != nil {
		return nil, apperror.Internal(err, "查询知识库文档失败")
	}
	if !ok || doc == nil {
		return nil, apperror.NotFound("知识库文档不存在")
	}
	doc.Status = CampusKnowledgeDocumentStatusIndexing
	doc.ParseStatus = "indexing"
	doc.ErrorMessage = ""
	if err := uc.repo.UpdateKnowledgeDocument(ctx, doc); err != nil {
		return nil, apperror.Internal(err, "更新知识库文档状态失败")
	}
	uc.enqueueKnowledgeIndex(ctx, doc)
	return doc, nil
}

func (uc *CampusUsecase) AdminListKnowledgeChunks(ctx context.Context, input *ListCampusKnowledgeChunksInput) (*ListCampusKnowledgeChunksOutput, error) {
	if !uc.isCampusOperator(ctx, input.UserID) {
		return nil, apperror.Forbidden("没有后台权限")
	}
	if input.DocumentID <= 0 {
		return nil, apperror.InvalidArgument("文档 ID 无效")
	}
	page, size := normalizePage(input.Page, input.Size)
	chunks, total, err := uc.repo.ListKnowledgeChunks(ctx, input.DocumentID, int((page-1)*size), int(size))
	if err != nil {
		return nil, apperror.Internal(err, "获取知识片段失败")
	}
	return &ListCampusKnowledgeChunksOutput{Chunks: chunks, Total: total}, nil
}

func (uc *CampusUsecase) AdminTestKnowledgeQuery(ctx context.Context, input *TestCampusKnowledgeQueryInput) (*CampusRAGQueryResponse, error) {
	if !uc.isCampusOperator(ctx, input.UserID) {
		return nil, apperror.Forbidden("没有后台权限")
	}
	query := strings.TrimSpace(input.Query)
	if len([]rune(query)) < 2 {
		return nil, apperror.InvalidArgument("请输入要测试的问题")
	}
	topK := int(input.TopK)
	if topK <= 0 || topK > 10 {
		topK = 5
	}
	out, err := uc.rag.Query(ctx, &CampusRAGQueryRequest{Query: query, TopK: topK})
	if err != nil {
		return nil, apperror.DependencyUnavailable(err, "RAG 服务暂不可用")
	}
	return out, nil
}

func (uc *CampusUsecase) AdminListRAGQueryLogs(ctx context.Context, input *ListCampusRAGQueryLogsInput) (*ListCampusRAGQueryLogsOutput, error) {
	if !uc.isCampusOperator(ctx, input.UserID) {
		return nil, apperror.Forbidden("没有后台权限")
	}
	page, size := normalizePage(input.Page, input.Size)
	logs, total, err := uc.repo.ListRAGQueryLogs(ctx, int((page-1)*size), int(size))
	if err != nil {
		return nil, apperror.Internal(err, "获取 RAG 查询日志失败")
	}
	return &ListCampusRAGQueryLogsOutput{Logs: logs, Total: total}, nil
}

func (uc *CampusUsecase) enqueueKnowledgeIndex(ctx context.Context, doc *CampusKnowledgeDocument) {
	if doc == nil {
		return
	}
	copyDoc := *doc
	if uc.knowledgeIndexer != nil {
		if err := uc.knowledgeIndexer.Add(ctx, &copyDoc); err == nil {
			return
		}
		uc.log.WithContext(ctx).Warnf("queue knowledge index failed: document_id=%d", doc.ID)
	}
	go func() {
		taskCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		if err := uc.indexKnowledgeDocument(taskCtx, &copyDoc); err != nil {
			uc.log.WithContext(taskCtx).Warnf("async knowledge index failed: document_id=%d err=%v", copyDoc.ID, err)
		}
	}()
}

func (uc *CampusUsecase) processKnowledgeIndexBatch(ctx context.Context, docs []*CampusKnowledgeDocument) error {
	for _, doc := range docs {
		if doc == nil {
			continue
		}
		if err := uc.indexKnowledgeDocument(ctx, doc); err != nil {
			uc.log.WithContext(ctx).Warnf("knowledge index failed: document_id=%d err=%v", doc.ID, err)
		}
	}
	return nil
}

func (uc *CampusUsecase) indexKnowledgeDocument(ctx context.Context, doc *CampusKnowledgeDocument) error {
	if doc == nil {
		return nil
	}
	effectiveAt := formatRAGTime(doc.EffectiveAt)
	expiredAt := formatRAGTime(doc.ExpiredAt)
	req := &CampusRAGIndexRequest{
		DocumentID:  doc.ID,
		Title:       doc.Title,
		Category:    doc.Category,
		Source:      doc.Source,
		FileURL:     doc.FileURL,
		FileType:    doc.FileType,
		Content:     doc.RawContent,
		EffectiveAt: effectiveAt,
		ExpiredAt:   expiredAt,
		Metadata: map[string]string{
			"content_type": doc.ContentType,
			"file_id":      doc.FileID,
			"effective_at": effectiveAt,
			"expired_at":   expiredAt,
		},
	}
	var resp *CampusRAGIndexResponse
	var err error
	if doc.ContentType == CampusKnowledgeContentTypeText {
		resp, err = uc.rag.IndexText(ctx, req)
	} else {
		resp, err = uc.rag.IndexDocument(ctx, req)
	}
	if err != nil {
		doc.Status = CampusKnowledgeDocumentStatusFailed
		doc.ParseStatus = "failed"
		doc.ErrorMessage = trimLimit(err.Error(), 1000)
		_ = uc.repo.UpdateKnowledgeDocument(ctx, doc)
		return apperror.DependencyUnavailable(err, "知识库索引失败")
	}
	chunks := make([]*CampusKnowledgeChunk, 0, len(resp.Chunks))
	for idx, chunk := range resp.Chunks {
		if chunk == nil {
			continue
		}
		chunk.ID = uc.idGen.NextID()
		chunk.DocumentID = doc.ID
		chunk.ChunkIndex = int32(idx)
		chunk.Title = firstNonEmpty(chunk.Title, doc.Title)
		chunk.Category = firstNonEmpty(chunk.Category, doc.Category)
		chunk.Source = firstNonEmpty(chunk.Source, doc.Source)
		chunk.Status = firstNonEmpty(chunk.Status, CampusKnowledgeChunkStatusActive)
		chunk.EmbeddingStatus = firstNonEmpty(chunk.EmbeddingStatus, "done")
		chunks = append(chunks, chunk)
	}
	if err := uc.repo.ReplaceKnowledgeChunks(ctx, doc.ID, chunks); err != nil {
		doc.Status = CampusKnowledgeDocumentStatusFailed
		doc.ParseStatus = "failed"
		doc.ErrorMessage = trimLimit(err.Error(), 1000)
		_ = uc.repo.UpdateKnowledgeDocument(ctx, doc)
		return apperror.Internal(err, "保存知识片段失败")
	}
	doc.Status = CampusKnowledgeDocumentStatusActive
	doc.ParseStatus = "done"
	doc.ErrorMessage = ""
	doc.ChunkCount = int64(len(chunks))
	if err := uc.repo.UpdateKnowledgeDocument(ctx, doc); err != nil {
		return apperror.Internal(err, "更新知识库文档状态失败")
	}
	return nil
}

func (uc *CampusUsecase) AdminDeleteComment(ctx context.Context, userID string, commentID int64) error {
	if !uc.isCampusOperator(ctx, userID) {
		return apperror.Forbidden("没有后台权限")
	}
	if commentID <= 0 {
		return apperror.InvalidArgument("评论 ID 无效")
	}
	if err := uc.repo.DeleteComment(ctx, commentID); err != nil {
		return apperror.Internal(err, "删除评论失败")
	}
	return nil
}

func (uc *CampusUsecase) AdminListReports(ctx context.Context, input *ListCampusReportsInput) (*ListCampusReportsOutput, error) {
	if !uc.isCampusOperator(ctx, input.UserID) {
		return nil, apperror.Forbidden("没有后台权限")
	}
	page, size := normalizePage(input.Page, input.Size)
	status := input.Status
	if status < 0 || status > 2 {
		status = -1
	}
	reports, total, err := uc.repo.ListReports(ctx, status, int((page-1)*size), int(size))
	if err != nil {
		return nil, apperror.Internal(err, "获取举报列表失败")
	}
	return &ListCampusReportsOutput{Reports: reports, Total: total}, nil
}

func (uc *CampusUsecase) AdminReviewReport(ctx context.Context, input *ReviewCampusReportInput) error {
	if !uc.isCampusOperator(ctx, input.UserID) {
		return apperror.Forbidden("没有后台权限")
	}
	if input.ReportID <= 0 {
		return apperror.InvalidArgument("举报 ID 无效")
	}
	status := int32(1)
	switch strings.TrimSpace(strings.ToLower(input.Action)) {
	case "resolve", "handled", "approve", "pass":
		status = 1
	case "reject", "dismiss":
		status = 2
	default:
		return apperror.InvalidArgument("举报处理动作无效")
	}
	if err := uc.repo.UpdateReportStatus(ctx, input.ReportID, status); err != nil {
		return apperror.Internal(err, "处理举报失败")
	}
	_ = uc.repo.CreateAuditLog(ctx, &CampusAuditLog{
		ID:         uc.idGen.NextID(),
		TargetType: "report",
		TargetID:   input.ReportID,
		UserID:     input.UserID,
		Provider:   "manual",
		Result:     input.Action,
		Reason:     strings.TrimSpace(input.Reason),
	})
	return nil
}

func (uc *CampusUsecase) CreateFeedback(ctx context.Context, input *CreateCampusFeedbackInput) (*CampusFeedback, error) {
	if strings.TrimSpace(input.UserID) == "" {
		return nil, apperror.Unauthorized("请先登录")
	}
	feedbackType := normalizeCampusFeedbackType(input.FeedbackType)
	content := strings.TrimSpace(input.Content)
	if len([]rune(content)) < 5 {
		return nil, apperror.InvalidArgument("请至少写 5 个字，方便我们理解问题")
	}
	if len([]rune(content)) > 800 {
		return nil, apperror.InvalidArgument("反馈内容不能超过 800 个字")
	}
	contact := strings.TrimSpace(input.Contact)
	if len([]rune(contact)) > 80 {
		return nil, apperror.InvalidArgument("联系方式不能超过 80 个字")
	}
	feedback := &CampusFeedback{
		ID:           uc.idGen.NextID(),
		UserID:       input.UserID,
		FeedbackType: feedbackType,
		Content:      content,
		Contact:      contact,
		Images:       sanitizeImages(input.Images, 3),
		Status:       CampusFeedbackStatusPending,
	}
	if err := uc.repo.CreateFeedback(ctx, feedback); err != nil {
		return nil, apperror.Internal(err, "提交反馈失败")
	}
	uc.trackEvent(ctx, &TrackCampusEventInput{
		UserID:     input.UserID,
		EventType:  "feedback_create",
		Page:       "feedback",
		TargetType: "feedback",
		TargetID:   feedback.ID,
		Channel:    feedbackType,
	})
	return feedback, nil
}

func (uc *CampusUsecase) AdminListFeedback(ctx context.Context, input *ListCampusFeedbackInput) (*ListCampusFeedbackOutput, error) {
	if !uc.isCampusOperator(ctx, input.UserID) {
		return nil, apperror.Forbidden("没有后台权限")
	}
	page, size := normalizePage(input.Page, input.Size)
	status := input.Status
	if status < -1 || status > CampusFeedbackStatusResolved {
		status = -1
	}
	feedbacks, total, err := uc.repo.ListFeedback(ctx, status, int((page-1)*size), int(size))
	if err != nil {
		return nil, apperror.Internal(err, "获取反馈列表失败")
	}
	if err := uc.assembler.HydrateFeedbackAuthors(ctx, feedbacks); err != nil {
		uc.log.WithContext(ctx).Warnf("hydrate feedback authors failed: %v", err)
	}
	return &ListCampusFeedbackOutput{Feedbacks: feedbacks, Total: total}, nil
}

func (uc *CampusUsecase) AdminReviewFeedback(ctx context.Context, input *ReviewCampusFeedbackInput) error {
	if !uc.isCampusOperator(ctx, input.UserID) {
		return apperror.Forbidden("没有后台权限")
	}
	if input.FeedbackID <= 0 {
		return apperror.InvalidArgument("反馈 ID 无效")
	}
	if input.Status < CampusFeedbackStatusPending || input.Status > CampusFeedbackStatusResolved {
		return apperror.InvalidArgument("反馈状态无效")
	}
	note := strings.TrimSpace(input.OperatorNote)
	if len([]rune(note)) > 300 {
		return apperror.InvalidArgument("处理备注不能超过 300 个字")
	}
	if err := uc.repo.UpdateFeedbackStatus(ctx, input.FeedbackID, input.Status, note); err != nil {
		return apperror.Internal(err, "更新反馈状态失败")
	}
	return nil
}

func (uc *CampusUsecase) CheckCampusRequest(ctx context.Context, input *CampusRateLimitInput) (bool, bool, error) {
	ip := strings.TrimSpace(input.IP)
	if ip == "" {
		ip = "unknown"
	}
	blocked, err := uc.repo.IsIPBlocked(ctx, ip)
	if err != nil {
		return false, false, apperror.Internal(err, "检查 IP 状态失败")
	}
	if blocked {
		return true, false, nil
	}
	limit, window := campusRateLimitRule(input.Category, input.Method, input.Path)
	if limit <= 0 {
		return false, true, nil
	}
	userKey := strings.TrimSpace(input.UserID)
	if userKey == "" {
		userKey = "guest"
	}
	key := fmt.Sprintf("campus:rl:%s:%s:%s", input.Category, ip, userKey)
	allowed, err := uc.repo.AllowCampusRequest(ctx, key, limit, window)
	if err != nil {
		uc.log.WithContext(ctx).Warnf("campus rate limit failed: %v", err)
		return false, true, nil
	}
	return false, allowed, nil
}

func (uc *CampusUsecase) RecordAccessLog(ctx context.Context, input *CampusAccessLogInput) {
	if input == nil {
		return
	}
	log := &CampusAccessLog{
		ID:          uc.idGen.NextID(),
		UserID:      strings.TrimSpace(input.UserID),
		IP:          trimLimit(input.IP, 64),
		Method:      trimLimit(strings.ToUpper(input.Method), 12),
		Path:        trimLimit(input.Path, 255),
		StatusCode:  input.StatusCode,
		DurationMs:  input.DurationMs,
		UserAgent:   trimLimit(input.UserAgent, 512),
		RateLimited: input.RateLimited,
		Blocked:     input.Blocked,
		CreatedAt:   time.Now(),
	}
	if uc.accessLogBatcher != nil {
		if err := uc.accessLogBatcher.Add(ctx, log); err != nil {
			uc.log.WithContext(ctx).Warnf("record campus access log batch failed: %v", err)
		}
		return
	}
	if err := uc.repo.CreateAccessLog(ctx, log); err != nil {
		uc.log.WithContext(ctx).Warnf("record campus access log failed: %v", err)
	}
}

func (uc *CampusUsecase) AdminSecurityOverview(ctx context.Context, input *ListCampusSecurityInput) (*CampusSecurityOverview, error) {
	if !uc.isCampusOperator(ctx, input.UserID) {
		return nil, apperror.Forbidden("没有后台权限")
	}
	overview, err := uc.repo.GetSecurityOverview(ctx)
	if err != nil {
		return nil, apperror.Internal(err, "获取安全数据失败")
	}
	return overview, nil
}

func (uc *CampusUsecase) AdminBlockIP(ctx context.Context, input *BlockCampusIPInput) error {
	if !uc.isCampusOperator(ctx, input.UserID) {
		return apperror.Forbidden("没有后台权限")
	}
	ip := strings.TrimSpace(input.IP)
	if ip == "" || len(ip) > 64 {
		return apperror.InvalidArgument("IP 无效")
	}
	reason := strings.TrimSpace(input.Reason)
	if reason == "" {
		reason = "后台手动封禁"
	}
	if len([]rune(reason)) > 120 {
		return apperror.InvalidArgument("封禁原因不能超过 120 个字")
	}
	return uc.repo.BlockIP(ctx, &CampusIPBlock{
		ID:        uc.idGen.NextID(),
		IP:        ip,
		Reason:    reason,
		Status:    CampusIPBlockStatusActive,
		CreatedBy: input.UserID,
	})
}

func (uc *CampusUsecase) AdminUnblockIP(ctx context.Context, input *BlockCampusIPInput) error {
	if !uc.isCampusOperator(ctx, input.UserID) {
		return apperror.Forbidden("没有后台权限")
	}
	ip := strings.TrimSpace(input.IP)
	if ip == "" || len(ip) > 64 {
		return apperror.InvalidArgument("IP 无效")
	}
	return uc.repo.UnblockIP(ctx, ip)
}

func (uc *CampusUsecase) AdminListUsers(ctx context.Context, input *ListCampusAdminUsersInput) (*ListCampusAdminUsersOutput, error) {
	if !uc.isCampusOperator(ctx, input.UserID) {
		return nil, apperror.Forbidden("没有后台权限")
	}
	page, size := normalizePage(input.Page, input.Size)
	role := strings.TrimSpace(strings.ToLower(input.Role))
	switch role {
	case "", "all", "user", "operator", "admin":
	default:
		return nil, apperror.InvalidArgument("角色筛选无效")
	}
	users, total, err := uc.repo.ListCampusUsers(ctx, strings.TrimSpace(input.Keyword), role, input.AuthStatus, int((page-1)*size), int(size))
	if err != nil {
		return nil, apperror.Internal(err, "获取用户列表失败")
	}
	for _, user := range users {
		if user.Role == "" {
			switch {
			case uc.isEnvListed("LEHU_CAMPUS_ADMIN_USER_IDS", user.User.ID):
				user.Role = "admin"
			case uc.isEnvListed("LEHU_CAMPUS_OPERATOR_USER_IDS", user.User.ID):
				user.Role = "operator"
			default:
				user.Role = "user"
			}
		}
	}
	return &ListCampusAdminUsersOutput{Users: users, Total: total}, nil
}

func (uc *CampusUsecase) AdminUpdateUserRole(ctx context.Context, input *UpdateCampusUserRoleInput) error {
	if !uc.isCampusAdmin(ctx, input.UserID) {
		return apperror.Forbidden("只有管理员可以调整运营权限")
	}
	targetUserID := strings.TrimSpace(input.TargetUserID)
	if targetUserID == "" {
		return apperror.InvalidArgument("用户 ID 无效")
	}
	role := strings.TrimSpace(strings.ToLower(input.Role))
	switch role {
	case "admin", "operator":
		if err := uc.repo.UpsertCampusOperator(ctx, targetUserID, role); err != nil {
			return apperror.Internal(err, "更新用户权限失败")
		}
	case "user", "":
		if err := uc.repo.RemoveCampusOperator(ctx, targetUserID); err != nil {
			return apperror.Internal(err, "移除用户权限失败")
		}
	default:
		return apperror.InvalidArgument("角色无效")
	}
	return nil
}

func flattenComments(comments []*CampusForumComment) []*CampusForumComment {
	flat := make([]*CampusForumComment, 0, len(comments))
	var walk func(items []*CampusForumComment)
	walk = func(items []*CampusForumComment) {
		for _, item := range items {
			if item == nil {
				continue
			}
			flat = append(flat, item)
			if len(item.PreviewReplies) > 0 {
				walk(item.PreviewReplies)
			}
		}
	}
	walk(comments)
	return flat
}

type wechatSession struct {
	OpenID  string
	UnionID string
}

func resolveWechatSession(ctx context.Context, code string) (*wechatSession, error) {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("LEHU_WECHAT_MOCK_LOGIN")), "true") && strings.HasPrefix(code, "mock-") {
		return &wechatSession{OpenID: "mock_" + strings.TrimPrefix(code, "mock-")}, nil
	}
	if openID := strings.TrimSpace(os.Getenv("LEHU_WECHAT_DEV_OPENID")); openID != "" {
		return &wechatSession{OpenID: openID}, nil
	}
	appID := strings.TrimSpace(os.Getenv("WECHAT_APP_ID"))
	secret := strings.TrimSpace(os.Getenv("WECHAT_APP_SECRET"))
	if appID == "" || secret == "" {
		return nil, apperror.Internal(fmt.Errorf("wechat app credentials missing"), "微信登录未配置")
	}
	endpoint := "https://api.weixin.qq.com/sns/jscode2session"
	reqURL := endpoint + "?appid=" + url.QueryEscape(appID) +
		"&secret=" + url.QueryEscape(secret) +
		"&js_code=" + url.QueryEscape(code) +
		"&grant_type=authorization_code"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, apperror.Internal(err, "创建微信登录请求失败")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, apperror.DependencyUnavailable(err, "微信登录服务暂不可用")
	}
	defer resp.Body.Close()
	var out struct {
		OpenID  string `json:"openid"`
		UnionID string `json:"unionid"`
		ErrCode int64  `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := jsonDecode(resp.Body, &out); err != nil {
		return nil, apperror.Internal(err, "解析微信登录响应失败")
	}
	if out.ErrCode != 0 || out.OpenID == "" {
		return nil, apperror.Unauthorized(firstNonEmpty(out.ErrMsg, "微信登录失败"))
	}
	return &wechatSession{OpenID: out.OpenID, UnionID: out.UnionID}, nil
}

type auditDecision struct {
	Provider string
	Result   string
	Reason   string
	Blocked  bool
	Pending  bool
}

func auditText(ctx context.Context, targetType, text string) auditDecision {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return auditDecision{Provider: "local", Result: "pass"}
	}
	lower := strings.ToLower(trimmed)
	for _, word := range []string{"广告代刷", "赌博", "诈骗", "裸聊"} {
		if strings.Contains(lower, word) {
			return auditDecision{Provider: "local", Result: "reject", Reason: "内容包含违规信息", Blocked: true}
		}
	}
	token := strings.TrimSpace(os.Getenv("WECHAT_CONTENT_SECURITY_TOKEN"))
	if token == "" {
		return auditDecision{Provider: "local", Result: "pass"}
	}
	if err := callWechatTextSecurity(ctx, token, trimmed); err != nil {
		return auditDecision{Provider: "wechat", Result: "pending", Reason: "内容安全服务暂不可用，已进入人工审核", Pending: true}
	}
	return auditDecision{Provider: "wechat", Result: "pass"}
}

func callWechatTextSecurity(ctx context.Context, token, content string) error {
	reqBody := strings.NewReader(fmt.Sprintf(`{"content":%q}`, content))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.weixin.qq.com/wxa/msg_sec_check?access_token="+url.QueryEscape(token), reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var out struct {
		ErrCode int64  `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := jsonDecode(resp.Body, &out); err != nil {
		return err
	}
	if out.ErrCode != 0 {
		return fmt.Errorf("wechat content security rejected: %d %s", out.ErrCode, out.ErrMsg)
	}
	return nil
}

func normalizePage(page, size int32) (int32, int32) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	if size > 50 {
		size = 50
	}
	return page, size
}

func sanitizeImages(images []string, limit int) []string {
	out := make([]string, 0, len(images))
	for _, image := range images {
		image = strings.TrimSpace(image)
		if image == "" {
			continue
		}
		out = append(out, image)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func normalizeCampusPostMedia(mediaType string, images []string, coverURL, videoURL string) (string, string, string, error) {
	mediaType = strings.TrimSpace(strings.ToLower(mediaType))
	coverURL = strings.TrimSpace(coverURL)
	videoURL = strings.TrimSpace(videoURL)
	if mediaType == "" {
		switch {
		case videoURL != "":
			mediaType = CampusPostMediaVideo
		case len(images) > 0:
			mediaType = CampusPostMediaImage
		default:
			mediaType = CampusPostMediaText
		}
	}
	switch mediaType {
	case CampusPostMediaText:
		return mediaType, "", "", nil
	case CampusPostMediaImage:
		if len(images) == 0 {
			return "", "", "", apperror.InvalidArgument("图文笔记至少需要 1 张图片")
		}
		if coverURL == "" {
			coverURL = images[0]
		}
		return mediaType, coverURL, "", nil
	case CampusPostMediaVideo:
		if videoURL == "" {
			return "", "", "", apperror.InvalidArgument("视频笔记需要上传视频")
		}
		if coverURL == "" {
			return "", "", "", apperror.InvalidArgument("视频笔记需要上传封面")
		}
		return mediaType, coverURL, videoURL, nil
	default:
		return "", "", "", apperror.InvalidArgument("笔记类型无效")
	}
}

func normalizeCampusPostType(postType string) string {
	switch strings.TrimSpace(strings.ToLower(postType)) {
	case "", CampusPostTypeNote:
		return CampusPostTypeNote
	case CampusPostTypeLost:
		return CampusPostTypeLost
	case CampusPostTypeQuestion:
		return CampusPostTypeQuestion
	case CampusPostTypeGuide:
		return CampusPostTypeGuide
	case CampusPostTypeClub:
		return CampusPostTypeClub
	default:
		return CampusPostTypeNote
	}
}

func normalizeCampusPostSort(sort string, fallback string) string {
	switch strings.TrimSpace(strings.ToLower(sort)) {
	case CampusPostSortRecommend:
		return CampusPostSortRecommend
	case CampusPostSortHot:
		return CampusPostSortHot
	case CampusPostSortNew:
		return CampusPostSortNew
	}
	if fallback == CampusPostSortHot || fallback == CampusPostSortNew {
		return fallback
	}
	return CampusPostSortRecommend
}

func normalizeCampusFeedbackType(feedbackType string) string {
	switch strings.TrimSpace(strings.ToLower(feedbackType)) {
	case "bug":
		return "bug"
	case "suggestion":
		return "suggestion"
	case "content":
		return "content"
	case "cooperation":
		return "cooperation"
	case "contact":
		return "contact"
	default:
		return "suggestion"
	}
}

func normalizeCampusNotificationGroup(group string) string {
	switch strings.TrimSpace(strings.ToLower(group)) {
	case CampusNotificationGroupReply:
		return CampusNotificationGroupReply
	case CampusNotificationGroupInteraction:
		return CampusNotificationGroupInteraction
	case CampusNotificationGroupSystem:
		return CampusNotificationGroupSystem
	default:
		return CampusNotificationGroupAll
	}
}

func campusRateLimitRule(category, method, path string) (int64, time.Duration) {
	category = strings.TrimSpace(strings.ToLower(category))
	method = strings.ToUpper(strings.TrimSpace(method))
	switch category {
	case "auth":
		return 12, time.Minute
	case "upload":
		return 12, time.Minute
	case "write":
		return 30, time.Minute
	case "feedback":
		return 6, time.Minute
	case "admin":
		return 180, time.Minute
	default:
		if method == http.MethodGet {
			return 240, time.Minute
		}
		_ = path
		return 60, time.Minute
	}
}

func sanitizeCampusPostExtra(extra map[string]string) map[string]string {
	allowed := map[string]int{
		"lost_kind":      16,
		"location":       80,
		"event_time":     80,
		"contact":        80,
		"club_name":      60,
		"activity_time":  80,
		"activity_place": 80,
	}
	out := make(map[string]string)
	for key, limit := range allowed {
		value := strings.TrimSpace(extra[key])
		if value == "" {
			continue
		}
		runes := []rune(value)
		if len(runes) > limit {
			value = string(runes[:limit])
		}
		out[key] = value
	}
	return out
}

func mergeCampusPostExtra(base map[string]string, next map[string]string) map[string]string {
	if next == nil {
		return sanitizeCampusPostExtra(base)
	}
	return sanitizeCampusPostExtra(next)
}

func clampSortWeight(weight int32) int32 {
	if weight < 0 {
		return 0
	}
	if weight > 10000 {
		return 10000
	}
	return weight
}

func normalizeCampusTerm(term string) string {
	term = strings.TrimSpace(term)
	if term != "" {
		return term
	}
	now := time.Now()
	year := now.Year()
	if now.Month() < 8 {
		return fmt.Sprintf("%d-%d-2", year-1, year)
	}
	return fmt.Sprintf("%d-%d-1", year, year+1)
}

func normalizeCampusTargetType(targetType string) string {
	switch strings.TrimSpace(strings.ToLower(targetType)) {
	case "post", "posts":
		return "post"
	case "comment", "comments":
		return "comment"
	default:
		return ""
	}
}

func parseOpsFilter(filter string) (*bool, *bool, *bool, bool) {
	value := true
	switch strings.TrimSpace(strings.ToLower(filter)) {
	case "official":
		return &value, nil, nil, false
	case "featured":
		return nil, &value, nil, false
	case "pinned":
		return nil, nil, &value, false
	case "reported":
		return nil, nil, nil, true
	default:
		return nil, nil, nil, false
	}
}

func (uc *CampusUsecase) isCampusOperator(ctx context.Context, userID string) bool {
	if uc.isCampusAdmin(ctx, userID) {
		return true
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return false
	}
	if uc.isEnvListed("LEHU_CAMPUS_OPERATOR_USER_IDS", userID) {
		return true
	}
	role, err := uc.repo.GetCampusOperatorRole(ctx, userID)
	if err != nil {
		uc.log.WithContext(ctx).Warnf("query campus operator role failed: user_id=%s err=%v", userID, err)
		return false
	}
	return role == "operator" || role == "admin"
}

func (uc *CampusUsecase) isCampusAdmin(ctx context.Context, userID string) bool {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return false
	}
	if strings.TrimSpace(os.Getenv("LEHU_CAMPUS_ADMIN_ALLOW_ALL")) == "true" {
		return true
	}
	if uc.isEnvListed("LEHU_CAMPUS_ADMIN_USER_IDS", userID) {
		return true
	}
	role, err := uc.repo.GetCampusOperatorRole(ctx, userID)
	if err != nil {
		uc.log.WithContext(ctx).Warnf("query campus admin role failed: user_id=%s err=%v", userID, err)
		return false
	}
	return role == "admin"
}

func (uc *CampusUsecase) isEnvListed(envName, userID string) bool {
	allowList := strings.TrimSpace(os.Getenv(envName))
	for _, item := range strings.Split(allowList, ",") {
		if strings.TrimSpace(item) == userID {
			return true
		}
	}
	return false
}

func (uc *CampusUsecase) TrackEvent(ctx context.Context, input *TrackCampusEventInput) error {
	if input == nil {
		return apperror.InvalidArgument("埋点事件不能为空")
	}
	event := normalizeCampusEvent(input.EventType)
	if event == "" {
		return apperror.InvalidArgument("埋点事件类型无效")
	}
	tracked := &TrackCampusEventInput{
		UserID:     strings.TrimSpace(input.UserID),
		EventType:  event,
		Page:       trimLimit(input.Page, 64),
		TargetType: trimLimit(input.TargetType, 32),
		TargetID:   input.TargetID,
		Channel:    trimLimit(input.Channel, 64),
		Extra:      sanitizeTrackExtra(input.Extra),
		UserAgent:  trimLimit(input.UserAgent, 512),
		IP:         trimLimit(input.IP, 64),
	}
	if uc.eventBatcher != nil {
		if err := uc.eventBatcher.Add(ctx, tracked); err != nil {
			return apperror.Internal(err, "记录埋点失败")
		}
		return nil
	}
	if err := uc.repo.TrackEvent(ctx, tracked); err != nil {
		return apperror.Internal(err, "记录埋点失败")
	}
	return nil
}

func (uc *CampusUsecase) FlushCampusBatches(ctx context.Context) error {
	var firstErr error
	if uc.eventBatcher != nil {
		if err := uc.eventBatcher.Flush(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if uc.accessLogBatcher != nil {
		if err := uc.accessLogBatcher.Flush(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if uc.knowledgeIndexer != nil {
		if err := uc.knowledgeIndexer.Flush(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (uc *CampusUsecase) StopCampusBatches(ctx context.Context) error {
	var firstErr error
	if uc.eventBatcher != nil {
		if err := uc.eventBatcher.Stop(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if uc.accessLogBatcher != nil {
		if err := uc.accessLogBatcher.Stop(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if uc.knowledgeIndexer != nil {
		if err := uc.knowledgeIndexer.Stop(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (uc *CampusUsecase) persistCampusEvents(ctx context.Context, events []*TrackCampusEventInput) error {
	if err := uc.repo.TrackEvents(ctx, events); err != nil {
		for _, event := range events {
			if singleErr := uc.repo.TrackEvent(ctx, event); singleErr != nil {
				uc.log.WithContext(ctx).Warnf("fallback track campus event failed: event=%s err=%v", event.EventType, singleErr)
			}
		}
		return err
	}
	return nil
}

func (uc *CampusUsecase) persistCampusAccessLogs(ctx context.Context, logs []*CampusAccessLog) error {
	if err := uc.repo.CreateAccessLogs(ctx, logs); err != nil {
		for _, item := range logs {
			if singleErr := uc.repo.CreateAccessLog(ctx, item); singleErr != nil {
				uc.log.WithContext(ctx).Warnf("fallback create campus access log failed: path=%s err=%v", item.Path, singleErr)
			}
		}
		return err
	}
	return nil
}

func (uc *CampusUsecase) trackEvent(ctx context.Context, input *TrackCampusEventInput) {
	if input == nil {
		return
	}
	if err := uc.TrackEvent(ctx, input); err != nil {
		uc.log.WithContext(ctx).Warnf("track campus event failed: event=%s err=%v", input.EventType, err)
	}
}

func normalizeCampusEvent(event string) string {
	switch strings.TrimSpace(strings.ToLower(event)) {
	case "visit", "share", "login", "post_create", "publish_open", "publish_success", "post_detail_visit", "comment_create", "comment_like", "like", "collect", "feedback_create", "report_create":
		return strings.TrimSpace(strings.ToLower(event))
	default:
		return ""
	}
}

func sanitizeTrackExtra(extra map[string]string) map[string]string {
	if len(extra) == 0 {
		return nil
	}
	out := make(map[string]string, len(extra))
	for key, value := range extra {
		key = trimLimit(key, 40)
		if key == "" {
			continue
		}
		out[key] = trimLimit(value, 200)
		if len(out) >= 16 {
			break
		}
	}
	return out
}

func trimLimit(value string, max int) string {
	value = strings.TrimSpace(value)
	if max <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max])
}

func shortHash(input string, n int) string {
	sum := sha256.Sum256([]byte(input))
	raw := hex.EncodeToString(sum[:])
	if n > len(raw) {
		n = len(raw)
	}
	return raw[:n]
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func jsonDecode(r io.Reader, v interface{}) error {
	return json.NewDecoder(r).Decode(v)
}
