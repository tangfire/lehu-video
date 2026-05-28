package biz

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
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

type CampusForumPost struct {
	ID             int64
	CategoryCode   string
	CategoryName   string
	AuthorID       string
	Author         *CampusForumAuthor
	Title          string
	Content        string
	Images         []string
	MediaType      string
	PostType       string
	Extra          map[string]string
	CoverURL       string
	VideoURL       string
	IsOfficial     bool
	IsFeatured     bool
	IsPinned       bool
	SortWeight     int32
	Status         int32
	AuditReason    string
	LikeCount      int64
	CommentCount   int64
	CollectedCount int64
	IsLiked        bool
	IsCollected    bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
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
	User    *UserBaseInfo
	Profile *CampusProfile
	Role    string
}

type ListCampusAdminUsersInput struct {
	UserID  string
	Keyword string
	Page    int32
	Size    int32
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
	ListPostsByIDs(ctx context.Context, postIDs []int64, statuses []int32) ([]*CampusForumPost, error)
	GetPostByID(ctx context.Context, postID int64) (bool, *CampusForumPost, error)
	GetAnyPostByID(ctx context.Context, postID int64) (bool, *CampusForumPost, error)
	DeletePost(ctx context.Context, postID int64) error
	UpdatePostStatus(ctx context.Context, postID int64, status int32, reason string) error
	UpdatePostByAdmin(ctx context.Context, post *CampusForumPost) error
	CreateComment(ctx context.Context, comment *CampusForumComment) error
	ListComments(ctx context.Context, query ListCampusCommentQuery) ([]*CampusForumComment, int64, error)
	FillCommentPosts(ctx context.Context, comments []*CampusForumComment) error
	GetCommentByID(ctx context.Context, commentID int64) (bool, *CampusForumComment, error)
	GetAnyCommentByID(ctx context.Context, commentID int64) (bool, *CampusForumComment, error)
	DeleteComment(ctx context.Context, commentID int64) error
	UpdateCommentStatus(ctx context.Context, commentID int64, status int32, reason string) error
	GetCommentLikeStatus(ctx context.Context, userID string, commentIDs []int64) (map[int64]bool, error)
	AddCommentLike(ctx context.Context, id int64, userID string, commentID int64) error
	RemoveCommentLike(ctx context.Context, userID string, commentID int64) error
	GetPostLikeStatus(ctx context.Context, userID string, postIDs []int64) (map[int64]bool, error)
	AddPostLike(ctx context.Context, id int64, userID string, postID int64) error
	RemovePostLike(ctx context.Context, userID string, postID int64) error
	GetPostCollectionStatus(ctx context.Context, userID string, postIDs []int64) (map[int64]bool, error)
	AddPostCollection(ctx context.Context, id int64, userID string, postID int64) error
	RemovePostCollection(ctx context.Context, userID string, postID int64) error
	CreateReport(ctx context.Context, report *CampusForumReport) error
	ListReports(ctx context.Context, status int32, offset, limit int) ([]*CampusForumReport, int64, error)
	UpdateReportStatus(ctx context.Context, reportID int64, status int32) error
	CreateFeedback(ctx context.Context, feedback *CampusFeedback) error
	ListFeedback(ctx context.Context, status int32, offset, limit int) ([]*CampusFeedback, int64, error)
	UpdateFeedbackStatus(ctx context.Context, feedbackID int64, status int32, note string) error
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
	ListCampusUsers(ctx context.Context, keyword string, offset, limit int) ([]*CampusAdminUser, int64, error)
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
	log               *log.Helper
}

func NewCampusUsecase(repo CampusRepo, base BaseAdapter, core CoreAdapter, timetableProvider CampusTimetableProvider, idGen CampusIDGenerator, authSecret string, logger log.Logger) *CampusUsecase {
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
		log:               log.NewHelper(logger),
	}
	uc.eventBatcher = NewCampusBatchProcessor("campus_event", 100, 2*time.Second, uc.persistCampusEvents, logger)
	uc.accessLogBatcher = NewCampusBatchProcessor("campus_access_log", 100, 2*time.Second, uc.persistCampusAccessLogs, logger)
	return uc
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
		course.Source = "educational_system"
	}
	if err := uc.repo.ReplaceTimetableCourses(ctx, input.UserID, term, "educational_system", courses); err != nil {
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
		Status:       CampusAuditStatusVisible,
	}
	if err := uc.repo.CreatePost(ctx, post); err != nil {
		return nil, apperror.Internal(err, "发布帖子失败")
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
	ok, _, err := uc.repo.GetPostByID(ctx, input.PostID)
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
	if err := uc.repo.CreateComment(ctx, comment); err != nil {
		return nil, apperror.Internal(err, "发表评论失败")
	}
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
	if err := uc.repo.AddCommentLike(ctx, uc.idGen.NextID(), userID, commentID); err != nil {
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
	ok, _, err := uc.repo.GetPostByID(ctx, postID)
	if err != nil {
		return apperror.Internal(err, "查询帖子失败")
	}
	if !ok {
		return apperror.NotFound("帖子不存在")
	}
	if err := uc.repo.AddPostLike(ctx, uc.idGen.NextID(), userID, postID); err != nil {
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
	ok, _, err := uc.repo.GetPostByID(ctx, postID)
	if err != nil {
		return apperror.Internal(err, "查询帖子失败")
	}
	if !ok {
		return apperror.NotFound("帖子不存在")
	}
	if err := uc.repo.AddPostCollection(ctx, uc.idGen.NextID(), userID, postID); err != nil {
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
		ok, _, err := uc.repo.GetAnyPostByID(ctx, input.TargetID)
		if err != nil {
			return apperror.Internal(err, "查询帖子失败")
		}
		if !ok {
			return apperror.NotFound("帖子不存在")
		}
		if err := uc.repo.UpdatePostStatus(ctx, input.TargetID, status, reason); err != nil {
			return apperror.Internal(err, "审核帖子失败")
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
	return summary, nil
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
	return &ListCampusPostsOutput{Posts: posts, Total: total}, nil
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
	users, total, err := uc.repo.ListCampusUsers(ctx, strings.TrimSpace(input.Keyword), int((page-1)*size), int(size))
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
	if strings.HasPrefix(code, "mock-") {
		return &wechatSession{OpenID: "mock_" + strings.TrimPrefix(code, "mock-")}, nil
	}
	if openID := strings.TrimSpace(os.Getenv("LEHU_WECHAT_DEV_OPENID")); openID != "" {
		return &wechatSession{OpenID: openID}, nil
	}
	appID := strings.TrimSpace(os.Getenv("WECHAT_APP_ID"))
	secret := strings.TrimSpace(os.Getenv("WECHAT_APP_SECRET"))
	if appID == "" || secret == "" {
		return &wechatSession{OpenID: "dev_" + shortHash(code, 32)}, nil
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
