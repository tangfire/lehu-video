package data

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"lehu-video/app/videoApi/service/internal/biz"
)

type campusRepo struct {
	data *Data
	log  *log.Helper
}

func NewCampusRepo(data *Data, logger log.Logger) biz.CampusRepo {
	return &campusRepo{data: data, log: log.NewHelper(logger)}
}

func campusPostOrder(sort string, collectedByUser bool) string {
	if collectedByUser {
		return "c.updated_at DESC, c.id DESC"
	}
	ageHours := "GREATEST(TIMESTAMPDIFF(HOUR, campus_forum_post.created_at, NOW()), 0)"
	interactionScore := "(campus_forum_post.like_count * 2 + campus_forum_post.comment_count * 4 + campus_forum_post.collected_count * 5)"
	decayedInteractionScore := "(" + interactionScore + " / POW(" + ageHours + " + 2, 1.2))"
	switch sort {
	case biz.CampusPostSortRecommend:
		recommendScore := "(campus_forum_post.sort_weight * 10 + IF(campus_forum_post.is_featured, 80, 0) + IF(campus_forum_post.is_official, 30, 0) + " + decayedInteractionScore + " + (24 / (" + ageHours + " + 2)))"
		return "campus_forum_post.is_pinned DESC, " + recommendScore + " DESC, campus_forum_post.created_at DESC, campus_forum_post.id DESC"
	case biz.CampusPostSortHot:
		hotScore := "(" + interactionScore + " / POW(" + ageHours + " + 2, 1.15))"
		return "campus_forum_post.is_pinned DESC, campus_forum_post.is_featured DESC, campus_forum_post.sort_weight DESC, " + hotScore + " DESC, campus_forum_post.created_at DESC, campus_forum_post.id DESC"
	case biz.CampusPostSortNew:
		return "campus_forum_post.created_at DESC, campus_forum_post.id DESC"
	}
	return "campus_forum_post.created_at DESC, campus_forum_post.id DESC"
}

type campusWechatIdentityModel struct {
	ID        int64     `gorm:"column:id"`
	Provider  string    `gorm:"column:provider"`
	OpenID    string    `gorm:"column:open_id"`
	UnionID   string    `gorm:"column:union_id"`
	UserID    int64     `gorm:"column:user_id"`
	AccountID int64     `gorm:"column:account_id"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (campusWechatIdentityModel) TableName() string { return "campus_wechat_identity" }

func (r *campusRepo) GetAccountIDByEmail(ctx context.Context, email string) (bool, string, error) {
	var row struct {
		ID int64 `gorm:"column:id"`
	}
	err := r.data.db.WithContext(ctx).
		Table("account").
		Select("id").
		Where("email = ? AND is_deleted = ?", email, false).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, "", nil
	}
	if err != nil {
		return false, "", err
	}
	return true, strconv.FormatInt(row.ID, 10), nil
}

type campusProfileModel struct {
	ID           int64     `gorm:"column:id"`
	UserID       int64     `gorm:"column:user_id"`
	AccountID    int64     `gorm:"column:account_id"`
	OpenID       string    `gorm:"column:open_id"`
	UnionID      string    `gorm:"column:union_id"`
	SchoolName   string    `gorm:"column:school_name"`
	StudentNo    string    `gorm:"column:student_no"`
	RealName     string    `gorm:"column:real_name"`
	ClassName    string    `gorm:"column:class_name"`
	DormBuilding string    `gorm:"column:dorm_building"`
	RoomNo       string    `gorm:"column:room_no"`
	Mobile       string    `gorm:"column:mobile"`
	AuthStatus   int32     `gorm:"column:auth_status"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

func (campusProfileModel) TableName() string { return "campus_profile" }

type campusTimetableCourseModel struct {
	ID             int64     `gorm:"column:id"`
	UserID         int64     `gorm:"column:user_id"`
	Term           string    `gorm:"column:term"`
	CourseName     string    `gorm:"column:course_name"`
	Teacher        string    `gorm:"column:teacher"`
	Classroom      string    `gorm:"column:classroom"`
	Weekday        int32     `gorm:"column:weekday"`
	StartSection   int32     `gorm:"column:start_section"`
	EndSection     int32     `gorm:"column:end_section"`
	StartWeek      int32     `gorm:"column:start_week"`
	EndWeek        int32     `gorm:"column:end_week"`
	WeekParity     int32     `gorm:"column:week_parity"`
	Source         string    `gorm:"column:source"`
	SourceCourseID string    `gorm:"column:source_course_id"`
	CreatedAt      time.Time `gorm:"column:created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at"`
}

func (campusTimetableCourseModel) TableName() string { return "campus_timetable_course" }

type campusForumCategoryModel struct {
	ID          int64     `gorm:"column:id"`
	Code        string    `gorm:"column:code"`
	Name        string    `gorm:"column:name"`
	Description string    `gorm:"column:description"`
	SortOrder   int32     `gorm:"column:sort_order"`
	IsDeleted   bool      `gorm:"column:is_deleted"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

func (campusForumCategoryModel) TableName() string { return "campus_forum_category" }

type campusForumPostModel struct {
	ID             int64           `gorm:"column:id"`
	CategoryCode   string          `gorm:"column:category_code"`
	AuthorID       int64           `gorm:"column:author_id"`
	Title          string          `gorm:"column:title"`
	Content        string          `gorm:"column:content"`
	Images         json.RawMessage `gorm:"column:images"`
	MediaType      string          `gorm:"column:media_type"`
	PostType       string          `gorm:"column:post_type"`
	Extra          json.RawMessage `gorm:"column:extra"`
	CoverURL       string          `gorm:"column:cover_url"`
	VideoURL       string          `gorm:"column:video_url"`
	IsOfficial     bool            `gorm:"column:is_official"`
	IsFeatured     bool            `gorm:"column:is_featured"`
	IsPinned       bool            `gorm:"column:is_pinned"`
	SortWeight     int32           `gorm:"column:sort_weight"`
	Status         int32           `gorm:"column:status"`
	AuditReason    string          `gorm:"column:audit_reason"`
	LikeCount      int64           `gorm:"column:like_count"`
	CommentCount   int64           `gorm:"column:comment_count"`
	CollectedCount int64           `gorm:"column:collected_count"`
	IsDeleted      bool            `gorm:"column:is_deleted"`
	CreatedAt      time.Time       `gorm:"column:created_at"`
	UpdatedAt      time.Time       `gorm:"column:updated_at"`
}

func (campusForumPostModel) TableName() string { return "campus_forum_post" }

type campusForumCommentModel struct {
	ID               int64           `gorm:"column:id"`
	PostID           int64           `gorm:"column:post_id"`
	ParentID         int64           `gorm:"column:parent_id"`
	ReplyToCommentID int64           `gorm:"column:reply_to_comment_id"`
	ReplyToUserID    int64           `gorm:"column:reply_to_user_id"`
	AuthorID         int64           `gorm:"column:author_id"`
	Content          string          `gorm:"column:content"`
	Images           json.RawMessage `gorm:"column:images"`
	Status           int32           `gorm:"column:status"`
	AuditReason      string          `gorm:"column:audit_reason"`
	LikeCount        int64           `gorm:"column:like_count"`
	ReplyCount       int64           `gorm:"column:reply_count"`
	IsDeleted        bool            `gorm:"column:is_deleted"`
	CreatedAt        time.Time       `gorm:"column:created_at"`
	UpdatedAt        time.Time       `gorm:"column:updated_at"`
}

func (campusForumCommentModel) TableName() string { return "campus_forum_comment" }

type campusForumCommentLikeModel struct {
	ID        int64     `gorm:"column:id"`
	CommentID int64     `gorm:"column:comment_id"`
	UserID    int64     `gorm:"column:user_id"`
	IsDeleted bool      `gorm:"column:is_deleted"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (campusForumCommentLikeModel) TableName() string { return "campus_forum_comment_like" }

type campusForumPostLikeModel struct {
	ID        int64     `gorm:"column:id"`
	PostID    int64     `gorm:"column:post_id"`
	UserID    int64     `gorm:"column:user_id"`
	IsDeleted bool      `gorm:"column:is_deleted"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (campusForumPostLikeModel) TableName() string { return "campus_forum_post_like" }

type campusForumPostCollectionModel struct {
	ID        int64     `gorm:"column:id"`
	PostID    int64     `gorm:"column:post_id"`
	UserID    int64     `gorm:"column:user_id"`
	IsDeleted bool      `gorm:"column:is_deleted"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (campusForumPostCollectionModel) TableName() string { return "campus_forum_post_collection" }

type campusForumReportModel struct {
	ID         int64     `gorm:"column:id"`
	TargetType string    `gorm:"column:target_type"`
	TargetID   int64     `gorm:"column:target_id"`
	ReporterID int64     `gorm:"column:reporter_id"`
	Reason     string    `gorm:"column:reason"`
	Detail     string    `gorm:"column:detail"`
	Status     int32     `gorm:"column:status"`
	CreatedAt  time.Time `gorm:"column:created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at"`
}

func (campusForumReportModel) TableName() string { return "campus_forum_report" }

type campusFeedbackModel struct {
	ID           int64           `gorm:"column:id"`
	UserID       int64           `gorm:"column:user_id"`
	FeedbackType string          `gorm:"column:feedback_type"`
	Content      string          `gorm:"column:content"`
	Contact      string          `gorm:"column:contact"`
	Images       json.RawMessage `gorm:"column:images"`
	Status       int32           `gorm:"column:status"`
	OperatorNote string          `gorm:"column:operator_note"`
	CreatedAt    time.Time       `gorm:"column:created_at"`
	UpdatedAt    time.Time       `gorm:"column:updated_at"`
}

func (campusFeedbackModel) TableName() string { return "campus_feedback" }

type campusNotificationModel struct {
	ID          int64           `gorm:"column:id"`
	RecipientID int64           `gorm:"column:recipient_id"`
	ActorID     int64           `gorm:"column:actor_id"`
	EventType   string          `gorm:"column:event_type"`
	TargetType  string          `gorm:"column:target_type"`
	TargetID    int64           `gorm:"column:target_id"`
	DedupeKey   *string         `gorm:"column:dedupe_key"`
	Title       string          `gorm:"column:title"`
	Content     string          `gorm:"column:content"`
	LinkPage    string          `gorm:"column:link_page"`
	LinkParams  json.RawMessage `gorm:"column:link_params"`
	ReadAt      *time.Time      `gorm:"column:read_at"`
	IsDeleted   bool            `gorm:"column:is_deleted"`
	CreatedAt   time.Time       `gorm:"column:created_at"`
	UpdatedAt   time.Time       `gorm:"column:updated_at"`
}

func (campusNotificationModel) TableName() string { return "campus_notification" }

type campusNotificationOutboxModel struct {
	ID          int64           `gorm:"column:id"`
	RecipientID int64           `gorm:"column:recipient_id"`
	ActorID     int64           `gorm:"column:actor_id"`
	EventType   string          `gorm:"column:event_type"`
	TargetType  string          `gorm:"column:target_type"`
	TargetID    int64           `gorm:"column:target_id"`
	DedupeKey   *string         `gorm:"column:dedupe_key"`
	Title       string          `gorm:"column:title"`
	Content     string          `gorm:"column:content"`
	LinkPage    string          `gorm:"column:link_page"`
	LinkParams  json.RawMessage `gorm:"column:link_params"`
	Audience    string          `gorm:"column:audience"`
	Status      string          `gorm:"column:status"`
	RetryCount  int32           `gorm:"column:retry_count"`
	NextRetryAt *time.Time      `gorm:"column:next_retry_at"`
	LockedUntil *time.Time      `gorm:"column:locked_until"`
	LastError   string          `gorm:"column:last_error"`
	CreatedAt   time.Time       `gorm:"column:created_at"`
	UpdatedAt   time.Time       `gorm:"column:updated_at"`
	ProcessedAt *time.Time      `gorm:"column:processed_at"`
}

func (campusNotificationOutboxModel) TableName() string { return "campus_notification_outbox" }

type campusAIReplyTaskModel struct {
	ID               int64      `gorm:"column:id"`
	PostID           int64      `gorm:"column:post_id"`
	RootCommentID    int64      `gorm:"column:root_comment_id"`
	TriggerCommentID int64      `gorm:"column:trigger_comment_id"`
	AskerID          int64      `gorm:"column:asker_id"`
	BotUserID        int64      `gorm:"column:bot_user_id"`
	Prompt           string     `gorm:"column:prompt"`
	Status           string     `gorm:"column:status"`
	RetryCount       int32      `gorm:"column:retry_count"`
	NextRetryAt      *time.Time `gorm:"column:next_retry_at"`
	LockedUntil      *time.Time `gorm:"column:locked_until"`
	AnswerCommentID  int64      `gorm:"column:answer_comment_id"`
	LastError        string     `gorm:"column:last_error"`
	CreatedAt        time.Time  `gorm:"column:created_at"`
	UpdatedAt        time.Time  `gorm:"column:updated_at"`
	ProcessedAt      *time.Time `gorm:"column:processed_at"`
}

func (campusAIReplyTaskModel) TableName() string { return "campus_ai_reply_task" }

type campusOpsSettingModel struct {
	ID        int64     `gorm:"column:id"`
	Key       string    `gorm:"column:setting_key"`
	Value     string    `gorm:"column:setting_value"`
	UpdatedBy int64     `gorm:"column:updated_by"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (campusOpsSettingModel) TableName() string { return "campus_ops_setting" }

type campusAIContentAuditTaskModel struct {
	ID          int64      `gorm:"column:id"`
	TargetType  string     `gorm:"column:target_type"`
	TargetID    int64      `gorm:"column:target_id"`
	Status      string     `gorm:"column:status"`
	RiskLevel   string     `gorm:"column:risk_level"`
	Decision    string     `gorm:"column:decision"`
	Reason      string     `gorm:"column:reason"`
	RawResult   string     `gorm:"column:raw_result"`
	RetryCount  int32      `gorm:"column:retry_count"`
	NextRetryAt *time.Time `gorm:"column:next_retry_at"`
	LockedUntil *time.Time `gorm:"column:locked_until"`
	LastError   string     `gorm:"column:last_error"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
	UpdatedAt   time.Time  `gorm:"column:updated_at"`
	ProcessedAt *time.Time `gorm:"column:processed_at"`
}

func (campusAIContentAuditTaskModel) TableName() string { return "campus_ai_audit_task" }

type campusKnowledgeDocumentModel struct {
	ID           int64      `gorm:"column:id"`
	Title        string     `gorm:"column:title"`
	Source       string     `gorm:"column:source"`
	Category     string     `gorm:"column:category"`
	ContentType  string     `gorm:"column:content_type"`
	FileURL      string     `gorm:"column:file_url"`
	FileID       int64      `gorm:"column:file_id"`
	FileType     string     `gorm:"column:file_type"`
	RawContent   string     `gorm:"column:raw_content"`
	Status       string     `gorm:"column:status"`
	ParseStatus  string     `gorm:"column:parse_status"`
	ErrorMessage string     `gorm:"column:error_message"`
	UploadedBy   int64      `gorm:"column:uploaded_by"`
	EffectiveAt  *time.Time `gorm:"column:effective_at"`
	ExpiredAt    *time.Time `gorm:"column:expired_at"`
	ChunkCount   int64      `gorm:"column:chunk_count"`
	IsDeleted    bool       `gorm:"column:is_deleted"`
	CreatedAt    time.Time  `gorm:"column:created_at"`
	UpdatedAt    time.Time  `gorm:"column:updated_at"`
}

func (campusKnowledgeDocumentModel) TableName() string { return "campus_knowledge_document" }

type campusKnowledgeChunkModel struct {
	ID              int64           `gorm:"column:id"`
	DocumentID      int64           `gorm:"column:document_id"`
	ChunkIndex      int32           `gorm:"column:chunk_index"`
	Title           string          `gorm:"column:title"`
	Content         string          `gorm:"column:content"`
	Summary         string          `gorm:"column:summary"`
	Category        string          `gorm:"column:category"`
	Keywords        json.RawMessage `gorm:"column:keywords"`
	Source          string          `gorm:"column:source"`
	Status          string          `gorm:"column:status"`
	QdrantPointID   string          `gorm:"column:qdrant_point_id"`
	EmbeddingStatus string          `gorm:"column:embedding_status"`
	IsDeleted       bool            `gorm:"column:is_deleted"`
	CreatedAt       time.Time       `gorm:"column:created_at"`
	UpdatedAt       time.Time       `gorm:"column:updated_at"`
}

func (campusKnowledgeChunkModel) TableName() string { return "campus_knowledge_chunk" }

type campusRAGQueryLogModel struct {
	ID               int64           `gorm:"column:id"`
	UserID           int64           `gorm:"column:user_id"`
	PostID           int64           `gorm:"column:post_id"`
	TriggerCommentID int64           `gorm:"column:trigger_comment_id"`
	Query            string          `gorm:"column:query"`
	NeedKnowledge    bool            `gorm:"column:need_knowledge"`
	Confidence       float64         `gorm:"column:confidence"`
	HitChunks        json.RawMessage `gorm:"column:hit_chunks"`
	Answer           string          `gorm:"column:answer"`
	Model            string          `gorm:"column:model"`
	DurationMs       int64           `gorm:"column:duration_ms"`
	ErrorMessage     string          `gorm:"column:error_message"`
	CreatedAt        time.Time       `gorm:"column:created_at"`
}

func (campusRAGQueryLogModel) TableName() string { return "campus_rag_query_log" }

type campusAccessLogModel struct {
	ID          int64     `gorm:"column:id"`
	UserID      int64     `gorm:"column:user_id"`
	IP          string    `gorm:"column:ip"`
	Method      string    `gorm:"column:method"`
	Path        string    `gorm:"column:path"`
	StatusCode  int32     `gorm:"column:status_code"`
	DurationMs  int64     `gorm:"column:duration_ms"`
	UserAgent   string    `gorm:"column:user_agent"`
	RateLimited bool      `gorm:"column:rate_limited"`
	Blocked     bool      `gorm:"column:blocked"`
	CreatedAt   time.Time `gorm:"column:created_at"`
}

func (campusAccessLogModel) TableName() string { return "campus_access_log" }

type campusIPBlockModel struct {
	ID        int64     `gorm:"column:id"`
	IP        string    `gorm:"column:ip"`
	Reason    string    `gorm:"column:reason"`
	Status    int32     `gorm:"column:status"`
	CreatedBy int64     `gorm:"column:created_by"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (campusIPBlockModel) TableName() string { return "campus_ip_block" }

type campusAuditLogModel struct {
	ID         int64     `gorm:"column:id"`
	TargetType string    `gorm:"column:target_type"`
	TargetID   int64     `gorm:"column:target_id"`
	UserID     int64     `gorm:"column:user_id"`
	Provider   string    `gorm:"column:provider"`
	Result     string    `gorm:"column:result"`
	Reason     string    `gorm:"column:reason"`
	CreatedAt  time.Time `gorm:"column:created_at"`
}

func (campusAuditLogModel) TableName() string { return "campus_audit_log" }

type campusOperatorModel struct {
	UserID    int64     `gorm:"column:user_id"`
	Role      string    `gorm:"column:role"`
	IsDeleted bool      `gorm:"column:is_deleted"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (campusOperatorModel) TableName() string { return "campus_operator" }

type campusEventModel struct {
	ID         int64           `gorm:"column:id"`
	UserID     int64           `gorm:"column:user_id"`
	EventType  string          `gorm:"column:event_type"`
	Page       string          `gorm:"column:page"`
	TargetType string          `gorm:"column:target_type"`
	TargetID   int64           `gorm:"column:target_id"`
	Channel    string          `gorm:"column:channel"`
	Extra      json.RawMessage `gorm:"column:extra"`
	UserAgent  string          `gorm:"column:user_agent"`
	IP         string          `gorm:"column:ip"`
	CreatedAt  time.Time       `gorm:"column:created_at"`
}

func (campusEventModel) TableName() string { return "campus_event" }

type campusUserRow struct {
	UserID           int64        `gorm:"column:user_id"`
	AccountID        int64        `gorm:"column:account_id"`
	Mobile           string       `gorm:"column:mobile"`
	Email            string       `gorm:"column:email"`
	Name             string       `gorm:"column:name"`
	Nickname         string       `gorm:"column:nickname"`
	Avatar           string       `gorm:"column:avatar"`
	SchoolName       string       `gorm:"column:school_name"`
	StudentNo        string       `gorm:"column:student_no"`
	RealName         string       `gorm:"column:real_name"`
	ClassName        string       `gorm:"column:class_name"`
	DormBuilding     string       `gorm:"column:dorm_building"`
	RoomNo           string       `gorm:"column:room_no"`
	AuthStatus       int32        `gorm:"column:auth_status"`
	Role             string       `gorm:"column:role"`
	PostCount        int64        `gorm:"column:post_count"`
	CommentCount     int64        `gorm:"column:comment_count"`
	LikeCount        int64        `gorm:"column:like_count"`
	CollectionCount  int64        `gorm:"column:collection_count"`
	FeedbackCount    int64        `gorm:"column:feedback_count"`
	ReportCount      int64        `gorm:"column:report_count"`
	LoginCount       int64        `gorm:"column:login_count"`
	VisitCount       int64        `gorm:"column:visit_count"`
	LastLoginAt      sql.NullTime `gorm:"column:last_login_at"`
	LastActiveAt     sql.NullTime `gorm:"column:last_active_at"`
	LastActiveIP     string       `gorm:"column:last_active_ip"`
	LastActivePath   string       `gorm:"column:last_active_path"`
	LastActiveStatus int32        `gorm:"column:last_active_status"`
	CreatedAt        time.Time    `gorm:"column:created_at"`
	UpdatedAt        time.Time    `gorm:"column:updated_at"`
}

func (r *campusRepo) GetWechatIdentity(ctx context.Context, provider, openID string) (bool, *biz.CampusWechatIdentity, error) {
	var row campusWechatIdentityModel
	err := r.data.db.WithContext(ctx).
		Where("provider = ? AND open_id = ?", provider, openID).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, err
	}
	return true, toBizWechatIdentity(&row), nil
}

func (r *campusRepo) SaveWechatIdentity(ctx context.Context, identity *biz.CampusWechatIdentity) error {
	row := campusWechatIdentityModel{
		ID:        identity.ID,
		Provider:  identity.Provider,
		OpenID:    identity.OpenID,
		UnionID:   identity.UnionID,
		UserID:    parseID(identity.UserID),
		AccountID: parseID(identity.AccountID),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	return r.data.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "provider"}, {Name: "open_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"union_id", "user_id", "account_id", "updated_at"}),
		}).
		Create(&row).Error
}

func (r *campusRepo) GetProfileByUserID(ctx context.Context, userID string) (bool, *biz.CampusProfile, error) {
	var row campusProfileModel
	err := r.data.db.WithContext(ctx).
		Where("user_id = ?", parseID(userID)).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, err
	}
	return true, toBizProfile(&row), nil
}

func (r *campusRepo) SaveProfile(ctx context.Context, profile *biz.CampusProfile) error {
	row := campusProfileModel{
		ID:           profile.ID,
		UserID:       parseID(profile.UserID),
		AccountID:    parseID(profile.AccountID),
		OpenID:       profile.OpenID,
		UnionID:      profile.UnionID,
		SchoolName:   profile.SchoolName,
		StudentNo:    profile.StudentNo,
		RealName:     profile.RealName,
		ClassName:    profile.ClassName,
		DormBuilding: profile.DormBuilding,
		RoomNo:       profile.RoomNo,
		Mobile:       profile.Mobile,
		AuthStatus:   profile.AuthStatus,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	return r.data.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"account_id", "open_id", "union_id", "school_name", "mobile", "updated_at",
			}),
		}).
		Create(&row).Error
}

func (r *campusRepo) UpdateProfile(ctx context.Context, profile *biz.CampusProfile) error {
	return r.data.db.WithContext(ctx).Model(&campusProfileModel{}).
		Where("user_id = ?", parseID(profile.UserID)).
		Updates(map[string]interface{}{
			"school_name":   profile.SchoolName,
			"student_no":    nullString(profile.StudentNo),
			"real_name":     nullString(profile.RealName),
			"class_name":    nullString(profile.ClassName),
			"dorm_building": nullString(profile.DormBuilding),
			"room_no":       nullString(profile.RoomNo),
			"mobile":        nullString(profile.Mobile),
			"auth_status":   profile.AuthStatus,
			"updated_at":    time.Now(),
		}).Error
}

func (r *campusRepo) ReplaceTimetableCourses(ctx context.Context, userID, term, source string, courses []*biz.CampusTimetableCourse) error {
	parsedUserID := parseID(userID)
	rows := make([]campusTimetableCourseModel, 0, len(courses))
	now := time.Now()
	for _, course := range courses {
		if course == nil {
			continue
		}
		rows = append(rows, campusTimetableCourseModel{
			ID:             course.ID,
			UserID:         parsedUserID,
			Term:           term,
			CourseName:     course.CourseName,
			Teacher:        course.Teacher,
			Classroom:      course.Classroom,
			Weekday:        course.Weekday,
			StartSection:   course.StartSection,
			EndSection:     course.EndSection,
			StartWeek:      course.StartWeek,
			EndWeek:        course.EndWeek,
			WeekParity:     course.WeekParity,
			Source:         source,
			SourceCourseID: course.SourceCourseID,
			CreatedAt:      now,
			UpdatedAt:      now,
		})
	}
	return r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ? AND term = ?", parsedUserID, term).
			Delete(&campusTimetableCourseModel{}).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		return tx.Create(&rows).Error
	})
}

func (r *campusRepo) ListTimetableCourses(ctx context.Context, userID, term string) ([]*biz.CampusTimetableCourse, error) {
	var rows []campusTimetableCourseModel
	if err := r.data.db.WithContext(ctx).
		Where("user_id = ? AND term = ?", parseID(userID), term).
		Order("weekday ASC, start_section ASC, id ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*biz.CampusTimetableCourse, 0, len(rows))
	for i := range rows {
		out = append(out, toBizTimetableCourse(&rows[i]))
	}
	return out, nil
}

func (r *campusRepo) ListCategories(ctx context.Context) ([]*biz.CampusForumCategory, error) {
	var rows []campusForumCategoryModel
	if err := r.data.db.WithContext(ctx).
		Where("is_deleted = ?", false).
		Order("sort_order ASC, id ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*biz.CampusForumCategory, 0, len(rows))
	for i := range rows {
		out = append(out, toBizCategory(&rows[i]))
	}
	return out, nil
}

func (r *campusRepo) GetCategoryByCode(ctx context.Context, code string) (bool, *biz.CampusForumCategory, error) {
	var row campusForumCategoryModel
	err := r.data.db.WithContext(ctx).
		Where("code = ? AND is_deleted = ?", code, false).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, err
	}
	return true, toBizCategory(&row), nil
}

func (r *campusRepo) CreatePost(ctx context.Context, post *biz.CampusForumPost) error {
	images, _ := json.Marshal(post.Images)
	extra, _ := json.Marshal(post.Extra)
	row := campusForumPostModel{
		ID:           post.ID,
		CategoryCode: post.CategoryCode,
		AuthorID:     parseID(post.AuthorID),
		Title:        post.Title,
		Content:      post.Content,
		Images:       images,
		MediaType:    post.MediaType,
		PostType:     post.PostType,
		Extra:        extra,
		CoverURL:     post.CoverURL,
		VideoURL:     post.VideoURL,
		IsOfficial:   post.IsOfficial,
		IsFeatured:   post.IsFeatured,
		IsPinned:     post.IsPinned,
		SortWeight:   post.SortWeight,
		Status:       post.Status,
		AuditReason:  post.AuditReason,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	return r.data.db.WithContext(ctx).Create(&row).Error
}

func (r *campusRepo) ListPosts(ctx context.Context, query biz.ListCampusPostQuery) ([]*biz.CampusForumPost, int64, error) {
	db := r.data.db.WithContext(ctx).Model(&campusForumPostModel{})
	if !query.IncludeDeleted {
		db = db.Where("campus_forum_post.is_deleted = ?", false)
	}
	if len(query.Statuses) > 0 {
		db = db.Where("campus_forum_post.status IN ?", query.Statuses)
	}
	if query.CategoryCode != "" {
		db = db.Where("campus_forum_post.category_code = ?", query.CategoryCode)
	}
	if query.PostType != "" {
		db = db.Where("campus_forum_post.post_type = ?", query.PostType)
	}
	if query.AuthorID != "" {
		db = db.Where("campus_forum_post.author_id = ?", parseID(query.AuthorID))
	}
	if query.CollectedByUserID != "" {
		db = db.Joins("JOIN campus_forum_post_collection c ON c.post_id = campus_forum_post.id AND c.user_id = ? AND c.is_deleted = ?", parseID(query.CollectedByUserID), false)
	}
	if query.OnlyReported {
		db = db.Where("EXISTS (SELECT 1 FROM campus_forum_report r WHERE r.target_type = ? AND r.target_id = campus_forum_post.id AND r.status = ?)", "post", biz.CampusAuditStatusPending)
	}
	if query.Keyword != "" {
		keyword := "%" + query.Keyword + "%"
		if postID, err := strconv.ParseInt(strings.TrimSpace(query.Keyword), 10, 64); err == nil && postID > 0 {
			db = db.Where("(campus_forum_post.id = ? OR campus_forum_post.title LIKE ? OR campus_forum_post.content LIKE ?)", postID, keyword, keyword)
		} else {
			db = db.Where("(campus_forum_post.title LIKE ? OR campus_forum_post.content LIKE ?)", keyword, keyword)
		}
	}
	if query.OnlyOfficial != nil {
		db = db.Where("campus_forum_post.is_official = ?", *query.OnlyOfficial)
	}
	if query.OnlyFeatured != nil {
		db = db.Where("campus_forum_post.is_featured = ?", *query.OnlyFeatured)
	}
	if query.OnlyPinned != nil {
		db = db.Where("campus_forum_post.is_pinned = ?", *query.OnlyPinned)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	order := campusPostOrder(query.Sort, query.CollectedByUserID != "")
	var rows []campusForumPostModel
	if err := db.Order(order).Offset(query.Offset).Limit(query.Limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	posts := make([]*biz.CampusForumPost, 0, len(rows))
	for i := range rows {
		post := toBizPost(&rows[i])
		posts = append(posts, post)
	}
	if err := r.fillPostCategoryNames(ctx, posts); err != nil {
		return nil, 0, err
	}
	return posts, total, nil
}

func (r *campusRepo) GetPublicUserPostStats(ctx context.Context, userID string) (*biz.CampusPublicUserStats, error) {
	var row struct {
		PostCount       int64
		LikeCount       sql.NullInt64
		CollectedCount  sql.NullInt64
		OfficialPostCnt int64
	}
	err := r.data.db.WithContext(ctx).Model(&campusForumPostModel{}).
		Select(`
			COUNT(*) AS post_count,
			COALESCE(SUM(like_count), 0) AS like_count,
			COALESCE(SUM(collected_count), 0) AS collected_count,
			COALESCE(SUM(CASE WHEN is_official THEN 1 ELSE 0 END), 0) AS official_post_cnt
		`).
		Where("author_id = ? AND is_deleted = ? AND status = ?", parseID(userID), false, biz.CampusAuditStatusVisible).
		Scan(&row).Error
	if err != nil {
		return nil, err
	}
	stats := &biz.CampusPublicUserStats{
		PostCount:       row.PostCount,
		LikeCount:       row.LikeCount.Int64,
		CollectedCount:  row.CollectedCount.Int64,
		HasOfficialPost: row.OfficialPostCnt > 0,
	}
	return stats, nil
}

func (r *campusRepo) ListPostsByIDs(ctx context.Context, postIDs []int64, statuses []int32) ([]*biz.CampusForumPost, error) {
	if len(postIDs) == 0 {
		return []*biz.CampusForumPost{}, nil
	}
	var rows []campusForumPostModel
	db := r.data.db.WithContext(ctx).Model(&campusForumPostModel{}).
		Where("id IN ? AND is_deleted = ?", postIDs, false)
	if len(statuses) > 0 {
		db = db.Where("status IN ?", statuses)
	}
	if err := db.Find(&rows).Error; err != nil {
		return nil, err
	}
	postMap := make(map[int64]*biz.CampusForumPost, len(rows))
	postsForCategory := make([]*biz.CampusForumPost, 0, len(rows))
	for i := range rows {
		post := toBizPost(&rows[i])
		postMap[post.ID] = post
		postsForCategory = append(postsForCategory, post)
	}
	if err := r.fillPostCategoryNames(ctx, postsForCategory); err != nil {
		return nil, err
	}
	ordered := make([]*biz.CampusForumPost, 0, len(postIDs))
	for _, id := range postIDs {
		if post := postMap[id]; post != nil {
			ordered = append(ordered, post)
		}
	}
	return ordered, nil
}

func (r *campusRepo) GetPostByID(ctx context.Context, postID int64) (bool, *biz.CampusForumPost, error) {
	var row campusForumPostModel
	err := r.data.db.WithContext(ctx).Model(&campusForumPostModel{}).
		Where("id = ? AND is_deleted = ? AND status = ?", postID, false, biz.CampusAuditStatusVisible).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, err
	}
	post := toBizPost(&row)
	if err := r.fillPostCategoryNames(ctx, []*biz.CampusForumPost{post}); err != nil {
		return false, nil, err
	}
	return true, post, nil
}

func (r *campusRepo) GetAnyPostByID(ctx context.Context, postID int64) (bool, *biz.CampusForumPost, error) {
	var row campusForumPostModel
	err := r.data.db.WithContext(ctx).Model(&campusForumPostModel{}).
		Where("id = ? AND is_deleted = ?", postID, false).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, err
	}
	post := toBizPost(&row)
	if err := r.fillPostCategoryNames(ctx, []*biz.CampusForumPost{post}); err != nil {
		return false, nil, err
	}
	return true, post, nil
}

func (r *campusRepo) DeletePost(ctx context.Context, postID int64) error {
	return r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&campusForumPostModel{}).
			Where("id = ? AND is_deleted = ?", postID, false).
			Updates(map[string]interface{}{
				"is_deleted":   true,
				"status":       biz.CampusAuditStatusDeleted,
				"audit_reason": "用户删除",
				"updated_at":   time.Now(),
			}).Error; err != nil {
			return err
		}
		return tx.Model(&campusForumCommentModel{}).
			Where("post_id = ? AND is_deleted = ?", postID, false).
			Updates(map[string]interface{}{
				"is_deleted": true,
				"status":     biz.CampusAuditStatusDeleted,
				"updated_at": time.Now(),
			}).Error
	})
}

func (r *campusRepo) UpdatePostStatus(ctx context.Context, postID int64, status int32, reason string) error {
	return r.data.db.WithContext(ctx).Model(&campusForumPostModel{}).
		Where("id = ?", postID).
		Updates(map[string]interface{}{
			"status":       status,
			"audit_reason": reason,
			"is_deleted":   status == biz.CampusAuditStatusDeleted,
			"updated_at":   time.Now(),
		}).Error
}

func (r *campusRepo) UpdatePostByAdmin(ctx context.Context, post *biz.CampusForumPost) error {
	images, _ := json.Marshal(post.Images)
	extra, _ := json.Marshal(post.Extra)
	return r.data.db.WithContext(ctx).Model(&campusForumPostModel{}).
		Where("id = ?", post.ID).
		Updates(map[string]interface{}{
			"category_code": post.CategoryCode,
			"title":         post.Title,
			"content":       post.Content,
			"images":        images,
			"media_type":    post.MediaType,
			"post_type":     post.PostType,
			"extra":         extra,
			"cover_url":     post.CoverURL,
			"video_url":     post.VideoURL,
			"status":        post.Status,
			"audit_reason":  post.AuditReason,
			"is_official":   post.IsOfficial,
			"is_featured":   post.IsFeatured,
			"is_pinned":     post.IsPinned,
			"sort_weight":   post.SortWeight,
			"is_deleted":    post.Status == biz.CampusAuditStatusDeleted,
			"updated_at":    time.Now(),
		}).Error
}

func (r *campusRepo) CreateComment(ctx context.Context, comment *biz.CampusForumComment) error {
	return r.CreateCommentWithOutbox(ctx, comment, nil)
}

func (r *campusRepo) CreateCommentWithOutbox(ctx context.Context, comment *biz.CampusForumComment, outbox *biz.CampusNotificationOutbox) error {
	images, _ := json.Marshal(comment.Images)
	row := campusForumCommentModel{
		ID:               comment.ID,
		PostID:           comment.PostID,
		ParentID:         comment.ParentID,
		ReplyToCommentID: comment.ReplyToCommentID,
		ReplyToUserID:    parseID(comment.ReplyToUserID),
		AuthorID:         parseID(comment.AuthorID),
		Content:          comment.Content,
		Images:           images,
		Status:           comment.Status,
		AuditReason:      comment.AuditReason,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	return r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		if comment.Status == biz.CampusAuditStatusVisible {
			if err := tx.Model(&campusForumPostModel{}).
				Where("id = ?", comment.PostID).
				UpdateColumn("comment_count", gorm.Expr("comment_count + ?", 1)).Error; err != nil {
				return err
			}
			if comment.ParentID > 0 {
				if err := tx.Model(&campusForumCommentModel{}).
					Where("id = ?", comment.ParentID).
					UpdateColumn("reply_count", gorm.Expr("reply_count + ?", 1)).Error; err != nil {
					return err
				}
			}
		}
		if err := createNotificationOutboxWithTx(tx, outbox); err != nil {
			return err
		}
		return nil
	})
}

func (r *campusRepo) ListComments(ctx context.Context, query biz.ListCampusCommentQuery) ([]*biz.CampusForumComment, int64, error) {
	db := r.data.db.WithContext(ctx).Model(&campusForumCommentModel{})
	if !query.IncludeDeleted {
		db = db.Where("is_deleted = ?", false)
	}
	if query.PostID > 0 {
		db = db.Where("post_id = ?", query.PostID)
	}
	if query.ParentID != nil {
		db = db.Where("parent_id = ?", *query.ParentID)
	}
	if query.AuthorID != "" {
		db = db.Where("author_id = ?", parseID(query.AuthorID))
	}
	if len(query.Statuses) > 0 {
		db = db.Where("status IN ?", query.Statuses)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []campusForumCommentModel
	order := "created_at ASC, id ASC"
	if query.AuthorID != "" {
		order = "created_at DESC, id DESC"
	}
	if err := db.Order(order).Offset(query.Offset).Limit(query.Limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	comments := make([]*biz.CampusForumComment, 0, len(rows))
	for i := range rows {
		comments = append(comments, toBizComment(&rows[i]))
	}
	return comments, total, nil
}

func (r *campusRepo) FillCommentPosts(ctx context.Context, comments []*biz.CampusForumComment) error {
	if len(comments) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(comments))
	seen := map[int64]struct{}{}
	for _, comment := range comments {
		if comment == nil || comment.PostID <= 0 {
			continue
		}
		if _, ok := seen[comment.PostID]; ok {
			continue
		}
		seen[comment.PostID] = struct{}{}
		ids = append(ids, comment.PostID)
	}
	if len(ids) == 0 {
		return nil
	}
	var rows []campusForumPostModel
	if err := r.data.db.WithContext(ctx).
		Where("id IN ? AND is_deleted = ?", ids, false).
		Find(&rows).Error; err != nil {
		return err
	}
	posts := make([]*biz.CampusForumPost, 0, len(rows))
	postMap := make(map[int64]*biz.CampusForumPost, len(rows))
	for i := range rows {
		post := toBizPost(&rows[i])
		posts = append(posts, post)
		postMap[post.ID] = post
	}
	if err := r.fillPostCategoryNames(ctx, posts); err != nil {
		return err
	}
	for _, comment := range comments {
		if comment != nil {
			comment.Post = postMap[comment.PostID]
		}
	}
	return nil
}

func (r *campusRepo) fillReports(ctx context.Context, reports []*biz.CampusForumReport) error {
	if len(reports) == 0 {
		return nil
	}
	postIDs := make([]int64, 0)
	commentIDs := make([]int64, 0)
	reporterIDs := make([]string, 0, len(reports))
	seenPost := map[int64]struct{}{}
	seenComment := map[int64]struct{}{}
	seenReporter := map[string]struct{}{}
	for _, report := range reports {
		if report == nil {
			continue
		}
		switch report.TargetType {
		case "post":
			if _, ok := seenPost[report.TargetID]; !ok {
				seenPost[report.TargetID] = struct{}{}
				postIDs = append(postIDs, report.TargetID)
			}
		case "comment":
			if _, ok := seenComment[report.TargetID]; !ok {
				seenComment[report.TargetID] = struct{}{}
				commentIDs = append(commentIDs, report.TargetID)
			}
		}
		if report.ReporterID != "" {
			if _, ok := seenReporter[report.ReporterID]; !ok {
				seenReporter[report.ReporterID] = struct{}{}
				reporterIDs = append(reporterIDs, report.ReporterID)
			}
		}
	}
	postMap := make(map[int64]*biz.CampusForumPost)
	if len(postIDs) > 0 {
		var rows []campusForumPostModel
		if err := r.data.db.WithContext(ctx).Where("id IN ?", postIDs).Find(&rows).Error; err != nil {
			return err
		}
		posts := make([]*biz.CampusForumPost, 0, len(rows))
		for i := range rows {
			post := toBizPost(&rows[i])
			posts = append(posts, post)
			postMap[post.ID] = post
		}
		if err := r.fillPostCategoryNames(ctx, posts); err != nil {
			return err
		}
	}
	commentMap := make(map[int64]*biz.CampusForumComment)
	if len(commentIDs) > 0 {
		var rows []campusForumCommentModel
		if err := r.data.db.WithContext(ctx).Where("id IN ?", commentIDs).Find(&rows).Error; err != nil {
			return err
		}
		comments := make([]*biz.CampusForumComment, 0, len(rows))
		for i := range rows {
			comment := toBizComment(&rows[i])
			comments = append(comments, comment)
			commentMap[comment.ID] = comment
		}
		if err := r.FillCommentPosts(ctx, comments); err != nil {
			return err
		}
	}
	reporterMap := make(map[string]*biz.CampusForumAuthor)
	if len(reporterIDs) > 0 {
		var rows []struct {
			ID       int64  `gorm:"column:id"`
			Name     string `gorm:"column:name"`
			Nickname string `gorm:"column:nickname"`
			Avatar   string `gorm:"column:avatar"`
		}
		ids := make([]int64, 0, len(reporterIDs))
		for _, id := range reporterIDs {
			ids = append(ids, parseID(id))
		}
		if err := r.data.db.WithContext(ctx).Table("user").
			Select("id, name, nickname, avatar").
			Where("id IN ?", ids).
			Find(&rows).Error; err != nil {
			return err
		}
		for _, row := range rows {
			id := fmt.Sprintf("%d", row.ID)
			reporterMap[id] = &biz.CampusForumAuthor{
				UserID:   id,
				Name:     firstNonEmptyData(row.Nickname, row.Name, "同学"),
				Nickname: row.Nickname,
				Avatar:   row.Avatar,
			}
		}
	}
	for _, report := range reports {
		if report == nil {
			continue
		}
		report.Reporter = reporterMap[report.ReporterID]
		if report.TargetType == "post" {
			report.Target = postMap[report.TargetID]
		}
		if report.TargetType == "comment" {
			report.Comment = commentMap[report.TargetID]
		}
	}
	return nil
}

func (r *campusRepo) GetCommentByID(ctx context.Context, commentID int64) (bool, *biz.CampusForumComment, error) {
	var row campusForumCommentModel
	err := r.data.db.WithContext(ctx).
		Where("id = ? AND is_deleted = ?", commentID, false).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, err
	}
	return true, toBizComment(&row), nil
}

func (r *campusRepo) GetAnyCommentByID(ctx context.Context, commentID int64) (bool, *biz.CampusForumComment, error) {
	var row campusForumCommentModel
	err := r.data.db.WithContext(ctx).
		Where("id = ?", commentID).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, err
	}
	return true, toBizComment(&row), nil
}

func (r *campusRepo) DeleteComment(ctx context.Context, commentID int64) error {
	return r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var comment campusForumCommentModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND is_deleted = ?", commentID, false).
			First(&comment).Error; err != nil {
			return err
		}
		decrement := int64(1)
		if comment.ParentID == 0 && comment.Status == biz.CampusAuditStatusVisible {
			var replyCount int64
			if err := tx.Model(&campusForumCommentModel{}).
				Where("parent_id = ? AND status = ? AND is_deleted = ?", comment.ID, biz.CampusAuditStatusVisible, false).
				Count(&replyCount).Error; err != nil {
				return err
			}
			decrement += replyCount
		}
		if err := tx.Model(&campusForumCommentModel{}).
			Where("id = ?", commentID).
			Updates(map[string]interface{}{
				"is_deleted":   true,
				"status":       biz.CampusAuditStatusDeleted,
				"audit_reason": "用户删除",
				"updated_at":   time.Now(),
			}).Error; err != nil {
			return err
		}
		if comment.ParentID == 0 {
			if err := tx.Model(&campusForumCommentModel{}).
				Where("parent_id = ? AND is_deleted = ?", comment.ID, false).
				Updates(map[string]interface{}{
					"is_deleted":   true,
					"status":       biz.CampusAuditStatusDeleted,
					"audit_reason": "父评论已撤回",
					"updated_at":   time.Now(),
				}).Error; err != nil {
				return err
			}
		}
		if comment.Status == biz.CampusAuditStatusVisible {
			if err := tx.Model(&campusForumPostModel{}).
				Where("id = ?", comment.PostID).
				UpdateColumn("comment_count", gorm.Expr("GREATEST(comment_count - ?, 0)", decrement)).Error; err != nil {
				return err
			}
			if comment.ParentID > 0 {
				return tx.Model(&campusForumCommentModel{}).
					Where("id = ?", comment.ParentID).
					UpdateColumn("reply_count", gorm.Expr("GREATEST(reply_count - ?, 0)", 1)).Error
			}
		}
		return nil
	})
}

func (r *campusRepo) UpdateCommentStatus(ctx context.Context, commentID int64, status int32, reason string) error {
	return r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var comment campusForumCommentModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", commentID).
			First(&comment).Error; err != nil {
			return err
		}
		wasVisible := comment.Status == biz.CampusAuditStatusVisible && !comment.IsDeleted
		willVisible := status == biz.CampusAuditStatusVisible
		if err := tx.Model(&campusForumCommentModel{}).
			Where("id = ?", commentID).
			Updates(map[string]interface{}{
				"status":       status,
				"audit_reason": reason,
				"is_deleted":   status == biz.CampusAuditStatusDeleted,
				"updated_at":   time.Now(),
			}).Error; err != nil {
			return err
		}
		if wasVisible == willVisible {
			return nil
		}
		delta := -1
		if willVisible {
			delta = 1
		}
		if err := tx.Model(&campusForumPostModel{}).
			Where("id = ?", comment.PostID).
			UpdateColumn("comment_count", gorm.Expr("GREATEST(comment_count + ?, 0)", delta)).Error; err != nil {
			return err
		}
		if comment.ParentID > 0 {
			return tx.Model(&campusForumCommentModel{}).
				Where("id = ?", comment.ParentID).
				UpdateColumn("reply_count", gorm.Expr("GREATEST(reply_count + ?, 0)", delta)).Error
		}
		return nil
	})
}

func (r *campusRepo) GetCommentLikeStatus(ctx context.Context, userID string, commentIDs []int64) (map[int64]bool, error) {
	result := make(map[int64]bool, len(commentIDs))
	if userID == "" || len(commentIDs) == 0 {
		return result, nil
	}
	var rows []campusForumCommentLikeModel
	if err := r.data.db.WithContext(ctx).
		Where("user_id = ? AND comment_id IN ? AND is_deleted = ?", parseID(userID), commentIDs, false).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.CommentID] = true
	}
	return result, nil
}

func (r *campusRepo) AddCommentLike(ctx context.Context, id int64, userID string, commentID int64) error {
	return r.AddCommentLikeWithOutbox(ctx, id, userID, commentID, nil)
}

func (r *campusRepo) AddCommentLikeWithOutbox(ctx context.Context, id int64, userID string, commentID int64, outbox *biz.CampusNotificationOutbox) error {
	parsedUserID := parseID(userID)
	return r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing campusForumCommentLikeModel
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("comment_id = ? AND user_id = ?", commentID, parsedUserID).
			First(&existing).Error
		if err == nil {
			if !existing.IsDeleted {
				return nil
			}
			if err := tx.Model(&campusForumCommentLikeModel{}).
				Where("id = ?", existing.ID).
				Updates(map[string]interface{}{"is_deleted": false, "updated_at": time.Now()}).Error; err != nil {
				return err
			}
			if err := tx.Model(&campusForumCommentModel{}).
				Where("id = ?", commentID).
				UpdateColumn("like_count", gorm.Expr("GREATEST(like_count + ?, 0)", 1)).Error; err != nil {
				return err
			}
			return createNotificationOutboxWithTx(tx, outbox)
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		row := campusForumCommentLikeModel{
			ID:        id,
			UserID:    parsedUserID,
			CommentID: commentID,
			IsDeleted: false,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		if err := tx.Model(&campusForumCommentModel{}).
			Where("id = ?", commentID).
			UpdateColumn("like_count", gorm.Expr("GREATEST(like_count + ?, 0)", 1)).Error; err != nil {
			return err
		}
		return createNotificationOutboxWithTx(tx, outbox)
	})
}

func (r *campusRepo) RemoveCommentLike(ctx context.Context, userID string, commentID int64) error {
	return r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&campusForumCommentLikeModel{}).
			Where("comment_id = ? AND user_id = ? AND is_deleted = ?", commentID, parseID(userID), false).
			Updates(map[string]interface{}{"is_deleted": true, "updated_at": time.Now()})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected > 0 {
			return tx.Model(&campusForumCommentModel{}).
				Where("id = ?", commentID).
				UpdateColumn("like_count", gorm.Expr("GREATEST(like_count - ?, 0)", 1)).Error
		}
		return nil
	})
}

func (r *campusRepo) GetPostLikeStatus(ctx context.Context, userID string, postIDs []int64) (map[int64]bool, error) {
	result := make(map[int64]bool, len(postIDs))
	if userID == "" || len(postIDs) == 0 {
		return result, nil
	}
	var rows []campusForumPostLikeModel
	if err := r.data.db.WithContext(ctx).
		Where("user_id = ? AND post_id IN ? AND is_deleted = ?", parseID(userID), postIDs, false).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.PostID] = true
	}
	return result, nil
}

func (r *campusRepo) AddPostLike(ctx context.Context, id int64, userID string, postID int64) error {
	return r.AddPostLikeWithOutbox(ctx, id, userID, postID, nil)
}

func (r *campusRepo) AddPostLikeWithOutbox(ctx context.Context, id int64, userID string, postID int64, outbox *biz.CampusNotificationOutbox) error {
	parsedUserID := parseID(userID)
	return r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing campusForumPostLikeModel
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("post_id = ? AND user_id = ?", postID, parsedUserID).
			First(&existing).Error
		if err == nil {
			if !existing.IsDeleted {
				return nil
			}
			if err := tx.Model(&campusForumPostLikeModel{}).
				Where("id = ?", existing.ID).
				Updates(map[string]interface{}{"is_deleted": false, "updated_at": time.Now()}).Error; err != nil {
				return err
			}
			if err := tx.Model(&campusForumPostModel{}).
				Where("id = ?", postID).
				UpdateColumn("like_count", gorm.Expr("GREATEST(like_count + ?, 0)", 1)).Error; err != nil {
				return err
			}
			return createNotificationOutboxWithTx(tx, outbox)
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		row := campusForumPostLikeModel{
			ID:        id,
			UserID:    parsedUserID,
			PostID:    postID,
			IsDeleted: false,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		if err := tx.Model(&campusForumPostModel{}).
			Where("id = ?", postID).
			UpdateColumn("like_count", gorm.Expr("GREATEST(like_count + ?, 0)", 1)).Error; err != nil {
			return err
		}
		return createNotificationOutboxWithTx(tx, outbox)
	})
}

func (r *campusRepo) RemovePostLike(ctx context.Context, userID string, postID int64) error {
	return r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&campusForumPostLikeModel{}).
			Where("post_id = ? AND user_id = ? AND is_deleted = ?", postID, parseID(userID), false).
			Updates(map[string]interface{}{"is_deleted": true, "updated_at": time.Now()})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected > 0 {
			return tx.Model(&campusForumPostModel{}).
				Where("id = ?", postID).
				UpdateColumn("like_count", gorm.Expr("GREATEST(like_count - ?, 0)", 1)).Error
		}
		return nil
	})
}

func (r *campusRepo) GetPostCollectionStatus(ctx context.Context, userID string, postIDs []int64) (map[int64]bool, error) {
	result := make(map[int64]bool, len(postIDs))
	if userID == "" || len(postIDs) == 0 {
		return result, nil
	}
	var rows []campusForumPostCollectionModel
	if err := r.data.db.WithContext(ctx).
		Where("user_id = ? AND post_id IN ? AND is_deleted = ?", parseID(userID), postIDs, false).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.PostID] = true
	}
	return result, nil
}

func (r *campusRepo) AddPostCollection(ctx context.Context, id int64, userID string, postID int64) error {
	return r.AddPostCollectionWithOutbox(ctx, id, userID, postID, nil)
}

func (r *campusRepo) AddPostCollectionWithOutbox(ctx context.Context, id int64, userID string, postID int64, outbox *biz.CampusNotificationOutbox) error {
	parsedUserID := parseID(userID)
	return r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing campusForumPostCollectionModel
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("post_id = ? AND user_id = ?", postID, parsedUserID).
			First(&existing).Error
		if err == nil {
			if !existing.IsDeleted {
				return nil
			}
			if err := tx.Model(&campusForumPostCollectionModel{}).
				Where("id = ?", existing.ID).
				Updates(map[string]interface{}{"is_deleted": false, "updated_at": time.Now()}).Error; err != nil {
				return err
			}
			if err := tx.Model(&campusForumPostModel{}).
				Where("id = ?", postID).
				UpdateColumn("collected_count", gorm.Expr("GREATEST(collected_count + ?, 0)", 1)).Error; err != nil {
				return err
			}
			return createNotificationOutboxWithTx(tx, outbox)
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		row := campusForumPostCollectionModel{
			ID:        id,
			UserID:    parsedUserID,
			PostID:    postID,
			IsDeleted: false,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		if err := tx.Model(&campusForumPostModel{}).
			Where("id = ?", postID).
			UpdateColumn("collected_count", gorm.Expr("GREATEST(collected_count + ?, 0)", 1)).Error; err != nil {
			return err
		}
		return createNotificationOutboxWithTx(tx, outbox)
	})
}

func (r *campusRepo) RemovePostCollection(ctx context.Context, userID string, postID int64) error {
	return r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&campusForumPostCollectionModel{}).
			Where("post_id = ? AND user_id = ? AND is_deleted = ?", postID, parseID(userID), false).
			Updates(map[string]interface{}{"is_deleted": true, "updated_at": time.Now()})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected > 0 {
			return tx.Model(&campusForumPostModel{}).
				Where("id = ?", postID).
				UpdateColumn("collected_count", gorm.Expr("GREATEST(collected_count - ?, 0)", 1)).Error
		}
		return nil
	})
}

func (r *campusRepo) CreateReport(ctx context.Context, in *biz.CampusForumReport) error {
	row := campusForumReportModel{
		ID:         in.ID,
		TargetType: in.TargetType,
		TargetID:   in.TargetID,
		ReporterID: parseID(in.ReporterID),
		Reason:     in.Reason,
		Detail:     in.Detail,
		Status:     in.Status,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	return r.data.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "target_type"}, {Name: "target_id"}, {Name: "reporter_id"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"reason":     in.Reason,
				"detail":     in.Detail,
				"status":     in.Status,
				"updated_at": time.Now(),
			}),
		}).
		Create(&row).Error
}

func (r *campusRepo) ListReports(ctx context.Context, status int32, offset, limit int) ([]*biz.CampusForumReport, int64, error) {
	db := r.data.db.WithContext(ctx).Model(&campusForumReportModel{})
	if status >= 0 {
		db = db.Where("status = ?", status)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []campusForumReportModel
	if err := db.Order("created_at DESC, id DESC").Offset(offset).Limit(limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	reports := make([]*biz.CampusForumReport, 0, len(rows))
	for i := range rows {
		reports = append(reports, toBizReport(&rows[i]))
	}
	if err := r.fillReports(ctx, reports); err != nil {
		return nil, 0, err
	}
	return reports, total, nil
}

func (r *campusRepo) UpdateReportStatus(ctx context.Context, reportID int64, status int32) error {
	return r.data.db.WithContext(ctx).Model(&campusForumReportModel{}).
		Where("id = ?", reportID).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		}).Error
}

func (r *campusRepo) CreateFeedback(ctx context.Context, in *biz.CampusFeedback) error {
	images, _ := json.Marshal(in.Images)
	row := campusFeedbackModel{
		ID:           in.ID,
		UserID:       parseID(in.UserID),
		FeedbackType: in.FeedbackType,
		Content:      in.Content,
		Contact:      in.Contact,
		Images:       images,
		Status:       in.Status,
		OperatorNote: in.OperatorNote,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	return r.data.db.WithContext(ctx).Create(&row).Error
}

func (r *campusRepo) ListFeedback(ctx context.Context, status int32, offset, limit int) ([]*biz.CampusFeedback, int64, error) {
	db := r.data.db.WithContext(ctx).Model(&campusFeedbackModel{})
	if status >= 0 {
		db = db.Where("status = ?", status)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []campusFeedbackModel
	if err := db.Order("created_at DESC, id DESC").Offset(offset).Limit(limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	feedbacks := make([]*biz.CampusFeedback, 0, len(rows))
	for i := range rows {
		feedbacks = append(feedbacks, toBizFeedback(&rows[i]))
	}
	return feedbacks, total, nil
}

func (r *campusRepo) UpdateFeedbackStatus(ctx context.Context, feedbackID int64, status int32, note string) error {
	return r.data.db.WithContext(ctx).Model(&campusFeedbackModel{}).
		Where("id = ?", feedbackID).
		Updates(map[string]interface{}{
			"status":        status,
			"operator_note": note,
			"updated_at":    time.Now(),
		}).Error
}

func (r *campusRepo) CreateNotification(ctx context.Context, in *biz.CampusNotification, unique bool) error {
	if in == nil {
		return nil
	}
	row := toNotificationModel(in)
	db := r.data.db.WithContext(ctx)
	if unique {
		if row.DedupeKey == nil || strings.TrimSpace(*row.DedupeKey) == "" {
			dedupeKey := notificationDedupeKey(row.RecipientID, row.ActorID, row.EventType, row.TargetType, row.TargetID)
			row.DedupeKey = &dedupeKey
		}
		db = db.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "dedupe_key"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"title":       row.Title,
				"content":     row.Content,
				"link_page":   row.LinkPage,
				"link_params": row.LinkParams,
				"is_deleted":  false,
				"updated_at":  time.Now(),
			}),
		})
	}
	return db.Create(&row).Error
}

func (r *campusRepo) BulkCreateNotifications(ctx context.Context, notifications []*biz.CampusNotification) error {
	if len(notifications) == 0 {
		return nil
	}
	rows := make([]campusNotificationModel, 0, len(notifications))
	for _, notification := range notifications {
		if notification == nil {
			continue
		}
		rows = append(rows, toNotificationModel(notification))
	}
	if len(rows) == 0 {
		return nil
	}
	return r.data.db.WithContext(ctx).CreateInBatches(rows, 100).Error
}

func (r *campusRepo) CreateNotificationOutbox(ctx context.Context, outbox *biz.CampusNotificationOutbox) error {
	return createNotificationOutboxWithTx(r.data.db.WithContext(ctx), outbox)
}

func (r *campusRepo) ClaimNotificationOutbox(ctx context.Context, limit int, lockFor time.Duration) ([]*biz.CampusNotificationOutbox, error) {
	if limit <= 0 {
		limit = 100
	}
	if lockFor <= 0 {
		lockFor = 30 * time.Second
	}
	now := time.Now()
	lockedUntil := now.Add(lockFor)
	var rows []campusNotificationOutboxModel
	err := r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("((status = ? AND (next_retry_at IS NULL OR next_retry_at <= ?)) OR (status = ? AND (locked_until IS NULL OR locked_until < ?)))",
				biz.CampusNotificationOutboxStatusPending, now, biz.CampusNotificationOutboxStatusProcessing, now).
			Order("created_at ASC, id ASC").
			Limit(limit).
			Find(&rows).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		ids := make([]int64, 0, len(rows))
		for _, row := range rows {
			ids = append(ids, row.ID)
		}
		return tx.Model(&campusNotificationOutboxModel{}).
			Where("id IN ?", ids).
			Updates(map[string]interface{}{
				"status":       biz.CampusNotificationOutboxStatusProcessing,
				"locked_until": lockedUntil,
				"updated_at":   now,
			}).Error
	})
	if err != nil {
		return nil, err
	}
	out := make([]*biz.CampusNotificationOutbox, 0, len(rows))
	for i := range rows {
		rows[i].Status = biz.CampusNotificationOutboxStatusProcessing
		rows[i].LockedUntil = &lockedUntil
		out = append(out, toBizNotificationOutbox(&rows[i]))
	}
	return out, nil
}

func (r *campusRepo) MarkNotificationOutboxDone(ctx context.Context, id int64) error {
	now := time.Now()
	return r.data.db.WithContext(ctx).Model(&campusNotificationOutboxModel{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":        biz.CampusNotificationOutboxStatusDone,
			"locked_until":  nil,
			"next_retry_at": nil,
			"last_error":    "",
			"processed_at":  now,
			"updated_at":    now,
		}).Error
}

func (r *campusRepo) MarkNotificationOutboxRetry(ctx context.Context, id int64, retryCount int32, nextRetryAt *time.Time, lastError string, final bool) error {
	status := biz.CampusNotificationOutboxStatusPending
	if final {
		status = biz.CampusNotificationOutboxStatusFailed
	}
	values := map[string]interface{}{
		"status":        status,
		"retry_count":   retryCount,
		"next_retry_at": nextRetryAt,
		"locked_until":  nil,
		"last_error":    trimLimitData(lastError, 600),
		"updated_at":    time.Now(),
	}
	return r.data.db.WithContext(ctx).Model(&campusNotificationOutboxModel{}).
		Where("id = ?", id).
		Updates(values).Error
}

func (r *campusRepo) CreateAIReplyTask(ctx context.Context, task *biz.CampusAIReplyTask) error {
	if task == nil {
		return nil
	}
	row := toAIReplyTaskModel(task)
	return r.data.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "trigger_comment_id"}},
			DoNothing: true,
		}).
		Create(&row).Error
}

func (r *campusRepo) ClaimAIReplyTasks(ctx context.Context, limit int, lockFor time.Duration) ([]*biz.CampusAIReplyTask, error) {
	if limit <= 0 {
		limit = 20
	}
	if lockFor <= 0 {
		lockFor = 45 * time.Second
	}
	now := time.Now()
	lockedUntil := now.Add(lockFor)
	var rows []campusAIReplyTaskModel
	err := r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("((status = ? AND (next_retry_at IS NULL OR next_retry_at <= ?)) OR (status = ? AND (locked_until IS NULL OR locked_until < ?)))",
				biz.CampusAIReplyTaskStatusPending, now, biz.CampusAIReplyTaskStatusProcessing, now).
			Order("created_at ASC, id ASC").
			Limit(limit).
			Find(&rows).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		ids := make([]int64, 0, len(rows))
		for _, row := range rows {
			ids = append(ids, row.ID)
		}
		return tx.Model(&campusAIReplyTaskModel{}).
			Where("id IN ?", ids).
			Updates(map[string]interface{}{
				"status":       biz.CampusAIReplyTaskStatusProcessing,
				"locked_until": lockedUntil,
				"updated_at":   now,
			}).Error
	})
	if err != nil {
		return nil, err
	}
	out := make([]*biz.CampusAIReplyTask, 0, len(rows))
	for i := range rows {
		rows[i].Status = biz.CampusAIReplyTaskStatusProcessing
		rows[i].LockedUntil = &lockedUntil
		out = append(out, toBizAIReplyTask(&rows[i]))
	}
	return out, nil
}

func (r *campusRepo) MarkAIReplyTaskDone(ctx context.Context, id int64, answerCommentID int64) error {
	now := time.Now()
	return r.data.db.WithContext(ctx).Model(&campusAIReplyTaskModel{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":            biz.CampusAIReplyTaskStatusDone,
			"answer_comment_id": answerCommentID,
			"locked_until":      nil,
			"next_retry_at":     nil,
			"last_error":        "",
			"processed_at":      now,
			"updated_at":        now,
		}).Error
}

func (r *campusRepo) MarkAIReplyTaskRetry(ctx context.Context, id int64, retryCount int32, nextRetryAt *time.Time, lastError string, final bool) error {
	status := biz.CampusAIReplyTaskStatusPending
	if final {
		status = biz.CampusAIReplyTaskStatusFailed
	}
	return r.data.db.WithContext(ctx).Model(&campusAIReplyTaskModel{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":        status,
			"retry_count":   retryCount,
			"next_retry_at": nextRetryAt,
			"locked_until":  nil,
			"last_error":    trimLimitData(lastError, 600),
			"updated_at":    time.Now(),
		}).Error
}

func (r *campusRepo) CountAIRepliesToday(ctx context.Context, botUserID string) (int64, error) {
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	var count int64
	err := r.data.db.WithContext(ctx).Model(&campusAIReplyTaskModel{}).
		Where("bot_user_id = ? AND status = ? AND processed_at >= ?", parseID(botUserID), biz.CampusAIReplyTaskStatusDone, start).
		Count(&count).Error
	return count, err
}

func (r *campusRepo) GetAIReplyOverview(ctx context.Context, botUserID string, limit int) (*biz.CampusAIReplyOverview, error) {
	overview := &biz.CampusAIReplyOverview{}
	db := r.data.db.WithContext(ctx).Model(&campusAIReplyTaskModel{})
	if strings.TrimSpace(botUserID) != "" {
		db = db.Where("bot_user_id = ?", parseID(botUserID))
	}
	var rows []struct {
		Status string
		Count  int64
	}
	if err := db.Select("status, COUNT(*) AS count").Group("status").Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		switch row.Status {
		case biz.CampusAIReplyTaskStatusPending:
			overview.Pending = row.Count
		case biz.CampusAIReplyTaskStatusProcessing:
			overview.Processing = row.Count
		case biz.CampusAIReplyTaskStatusDone:
			overview.Done = row.Count
		case biz.CampusAIReplyTaskStatusFailed:
			overview.Failed = row.Count
		}
	}
	if limit <= 0 {
		limit = 5
	}
	var recent []campusAIReplyTaskModel
	query := r.data.db.WithContext(ctx).Model(&campusAIReplyTaskModel{})
	if strings.TrimSpace(botUserID) != "" {
		query = query.Where("bot_user_id = ?", parseID(botUserID))
	}
	if err := query.Order("updated_at DESC, created_at DESC, id DESC").Limit(limit).Find(&recent).Error; err != nil {
		return nil, err
	}
	overview.Recent = make([]*biz.CampusAIReplyTask, 0, len(recent))
	for i := range recent {
		overview.Recent = append(overview.Recent, toBizAIReplyTask(&recent[i]))
	}
	return overview, nil
}

func (r *campusRepo) ListAIReplyTasks(ctx context.Context, status string, offset, limit int) ([]*biz.CampusAIReplyTask, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	db := r.data.db.WithContext(ctx).Model(&campusAIReplyTaskModel{})
	if strings.TrimSpace(status) != "" {
		db = db.Where("status = ?", status)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []campusAIReplyTaskModel
	if err := db.Order("updated_at DESC, created_at DESC, id DESC").Offset(offset).Limit(limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	tasks := make([]*biz.CampusAIReplyTask, 0, len(rows))
	for i := range rows {
		tasks = append(tasks, toBizAIReplyTask(&rows[i]))
	}
	return tasks, total, nil
}

func (r *campusRepo) ResetAIReplyTask(ctx context.Context, id int64) error {
	return r.data.db.WithContext(ctx).Model(&campusAIReplyTaskModel{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":        biz.CampusAIReplyTaskStatusPending,
			"retry_count":   0,
			"next_retry_at": nil,
			"locked_until":  nil,
			"last_error":    "",
			"updated_at":    time.Now(),
		}).Error
}

func (r *campusRepo) GetOpsSetting(ctx context.Context, key string) (bool, string, string, time.Time, error) {
	var row campusOpsSettingModel
	err := r.data.db.WithContext(ctx).Model(&campusOpsSettingModel{}).
		Where("setting_key = ?", strings.TrimSpace(key)).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, "", "", time.Time{}, nil
	}
	if err != nil {
		return false, "", "", time.Time{}, err
	}
	updatedBy := ""
	if row.UpdatedBy > 0 {
		updatedBy = fmt.Sprintf("%d", row.UpdatedBy)
	}
	return true, row.Value, updatedBy, row.UpdatedAt, nil
}

func (r *campusRepo) SetOpsSetting(ctx context.Context, key, value, updatedBy string) error {
	now := time.Now()
	row := campusOpsSettingModel{
		Key:       strings.TrimSpace(key),
		Value:     strings.TrimSpace(value),
		UpdatedBy: parseID(updatedBy),
		CreatedAt: now,
		UpdatedAt: now,
	}
	return r.data.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "setting_key"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"setting_value": row.Value,
			"updated_by":    row.UpdatedBy,
			"updated_at":    now,
		}),
	}).Create(&row).Error
}

func (r *campusRepo) CreateAIContentAuditTask(ctx context.Context, task *biz.CampusAIContentAuditTask) error {
	if task == nil {
		return nil
	}
	row := toAIContentAuditTaskModel(task)
	return r.data.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "target_type"}, {Name: "target_id"}},
		DoNothing: true,
	}).Create(&row).Error
}

func (r *campusRepo) ClaimAIContentAuditTasks(ctx context.Context, limit int, lockFor time.Duration) ([]*biz.CampusAIContentAuditTask, error) {
	if limit <= 0 {
		limit = 10
	}
	if lockFor <= 0 {
		lockFor = 45 * time.Second
	}
	now := time.Now()
	lockedUntil := now.Add(lockFor)
	var rows []campusAIContentAuditTaskModel
	err := r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("((status = ? AND (next_retry_at IS NULL OR next_retry_at <= ?)) OR (status = ? AND (locked_until IS NULL OR locked_until < ?)))",
				biz.CampusAIContentAuditTaskStatusPending, now, biz.CampusAIContentAuditTaskStatusProcessing, now).
			Order("created_at ASC, id ASC").
			Limit(limit).
			Find(&rows).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		ids := make([]int64, 0, len(rows))
		for _, row := range rows {
			ids = append(ids, row.ID)
		}
		return tx.Model(&campusAIContentAuditTaskModel{}).
			Where("id IN ?", ids).
			Updates(map[string]interface{}{
				"status":       biz.CampusAIContentAuditTaskStatusProcessing,
				"locked_until": lockedUntil,
				"updated_at":   now,
			}).Error
	})
	if err != nil {
		return nil, err
	}
	out := make([]*biz.CampusAIContentAuditTask, 0, len(rows))
	for i := range rows {
		rows[i].Status = biz.CampusAIContentAuditTaskStatusProcessing
		rows[i].LockedUntil = &lockedUntil
		out = append(out, toBizAIContentAuditTask(&rows[i]))
	}
	return out, nil
}

func (r *campusRepo) MarkAIContentAuditTaskDone(ctx context.Context, id int64, decision, riskLevel, reason, rawResult string) error {
	now := time.Now()
	return r.data.db.WithContext(ctx).Model(&campusAIContentAuditTaskModel{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":        biz.CampusAIContentAuditTaskStatusDone,
			"decision":      trimLimitData(decision, 24),
			"risk_level":    trimLimitData(riskLevel, 24),
			"reason":        trimLimitData(reason, 255),
			"raw_result":    trimLimitData(rawResult, 4000),
			"locked_until":  nil,
			"next_retry_at": nil,
			"last_error":    "",
			"processed_at":  now,
			"updated_at":    now,
		}).Error
}

func (r *campusRepo) MarkAIContentAuditTaskRetry(ctx context.Context, id int64, retryCount int32, nextRetryAt *time.Time, lastError string, final bool) error {
	status := biz.CampusAIContentAuditTaskStatusPending
	if final {
		status = biz.CampusAIContentAuditTaskStatusFailed
	}
	return r.data.db.WithContext(ctx).Model(&campusAIContentAuditTaskModel{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":        status,
			"retry_count":   retryCount,
			"next_retry_at": nextRetryAt,
			"locked_until":  nil,
			"last_error":    trimLimitData(lastError, 600),
			"updated_at":    time.Now(),
		}).Error
}

func (r *campusRepo) GetLatestAIContentAuditTask(ctx context.Context, targetType string, targetID int64) (bool, *biz.CampusAIContentAuditTask, error) {
	var row campusAIContentAuditTaskModel
	err := r.data.db.WithContext(ctx).Model(&campusAIContentAuditTaskModel{}).
		Where("target_type = ? AND target_id = ?", targetType, targetID).
		Order("created_at DESC, id DESC").
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, err
	}
	return true, toBizAIContentAuditTask(&row), nil
}

func (r *campusRepo) GetLatestAIContentAuditTasks(ctx context.Context, targetType string, targetIDs []int64) (map[int64]*biz.CampusAIContentAuditTask, error) {
	out := map[int64]*biz.CampusAIContentAuditTask{}
	if len(targetIDs) == 0 {
		return out, nil
	}
	var rows []campusAIContentAuditTaskModel
	if err := r.data.db.WithContext(ctx).Model(&campusAIContentAuditTaskModel{}).
		Where("target_type = ? AND target_id IN ?", targetType, targetIDs).
		Order("created_at DESC, id DESC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for i := range rows {
		if _, exists := out[rows[i].TargetID]; exists {
			continue
		}
		out[rows[i].TargetID] = toBizAIContentAuditTask(&rows[i])
	}
	return out, nil
}

func (r *campusRepo) CountPendingAIContentAuditTasks(ctx context.Context) (int64, error) {
	var count int64
	err := r.data.db.WithContext(ctx).Model(&campusAIContentAuditTaskModel{}).
		Where("status IN ?", []string{biz.CampusAIContentAuditTaskStatusPending, biz.CampusAIContentAuditTaskStatusProcessing}).
		Count(&count).Error
	return count, err
}

func (r *campusRepo) CreateKnowledgeDocument(ctx context.Context, doc *biz.CampusKnowledgeDocument) error {
	if doc == nil {
		return nil
	}
	row := toKnowledgeDocumentModel(doc)
	return r.data.db.WithContext(ctx).Create(&row).Error
}

func (r *campusRepo) UpdateKnowledgeDocument(ctx context.Context, doc *biz.CampusKnowledgeDocument) error {
	if doc == nil {
		return nil
	}
	values := map[string]interface{}{
		"title":         doc.Title,
		"source":        doc.Source,
		"category":      doc.Category,
		"status":        doc.Status,
		"parse_status":  doc.ParseStatus,
		"error_message": trimLimitData(doc.ErrorMessage, 1000),
		"effective_at":  doc.EffectiveAt,
		"expired_at":    doc.ExpiredAt,
		"chunk_count":   doc.ChunkCount,
		"updated_at":    time.Now(),
	}
	return r.data.db.WithContext(ctx).Model(&campusKnowledgeDocumentModel{}).
		Where("id = ? AND is_deleted = ?", doc.ID, false).
		Updates(values).Error
}

func (r *campusRepo) GetKnowledgeDocumentByID(ctx context.Context, id int64) (bool, *biz.CampusKnowledgeDocument, error) {
	var row campusKnowledgeDocumentModel
	err := r.data.db.WithContext(ctx).
		Where("id = ? AND is_deleted = ?", id, false).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, err
	}
	return true, toBizKnowledgeDocument(&row), nil
}

func (r *campusRepo) ListKnowledgeDocuments(ctx context.Context, keyword, category, status string, offset, limit int) ([]*biz.CampusKnowledgeDocument, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	db := r.data.db.WithContext(ctx).Model(&campusKnowledgeDocumentModel{}).Where("is_deleted = ?", false)
	if strings.TrimSpace(keyword) != "" {
		like := "%" + strings.TrimSpace(keyword) + "%"
		db = db.Where("title LIKE ? OR source LIKE ? OR raw_content LIKE ?", like, like, like)
	}
	if strings.TrimSpace(category) != "" {
		db = db.Where("category = ?", strings.TrimSpace(category))
	}
	if strings.TrimSpace(status) != "" {
		db = db.Where("status = ?", strings.TrimSpace(status))
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []campusKnowledgeDocumentModel
	if err := db.Order("updated_at DESC, id DESC").Offset(offset).Limit(limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]*biz.CampusKnowledgeDocument, 0, len(rows))
	for i := range rows {
		out = append(out, toBizKnowledgeDocument(&rows[i]))
	}
	return out, total, nil
}

func (r *campusRepo) ReplaceKnowledgeChunks(ctx context.Context, documentID int64, chunks []*biz.CampusKnowledgeChunk) error {
	return r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&campusKnowledgeChunkModel{}).
			Where("document_id = ?", documentID).
			Updates(map[string]interface{}{"is_deleted": true, "status": biz.CampusKnowledgeChunkStatusDisabled, "updated_at": time.Now()}).Error; err != nil {
			return err
		}
		rows := make([]campusKnowledgeChunkModel, 0, len(chunks))
		for _, chunk := range chunks {
			if chunk == nil {
				continue
			}
			rows = append(rows, toKnowledgeChunkModel(chunk))
		}
		if len(rows) > 0 {
			if err := tx.CreateInBatches(rows, 100).Error; err != nil {
				return err
			}
		}
		return tx.Model(&campusKnowledgeDocumentModel{}).
			Where("id = ?", documentID).
			Updates(map[string]interface{}{"chunk_count": len(rows), "updated_at": time.Now()}).Error
	})
}

func (r *campusRepo) ListKnowledgeChunks(ctx context.Context, documentID int64, offset, limit int) ([]*biz.CampusKnowledgeChunk, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	db := r.data.db.WithContext(ctx).Model(&campusKnowledgeChunkModel{}).
		Where("document_id = ? AND is_deleted = ?", documentID, false)
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []campusKnowledgeChunkModel
	if err := db.Order("chunk_index ASC, id ASC").Offset(offset).Limit(limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]*biz.CampusKnowledgeChunk, 0, len(rows))
	for i := range rows {
		out = append(out, toBizKnowledgeChunk(&rows[i]))
	}
	return out, total, nil
}

func (r *campusRepo) CreateRAGQueryLog(ctx context.Context, item *biz.CampusRAGQueryLog) error {
	if item == nil {
		return nil
	}
	row := toRAGQueryLogModel(item)
	return r.data.db.WithContext(ctx).Create(&row).Error
}

func (r *campusRepo) ListRAGQueryLogs(ctx context.Context, offset, limit int) ([]*biz.CampusRAGQueryLog, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	db := r.data.db.WithContext(ctx).Model(&campusRAGQueryLogModel{})
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []campusRAGQueryLogModel
	if err := db.Order("created_at DESC, id DESC").Offset(offset).Limit(limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]*biz.CampusRAGQueryLog, 0, len(rows))
	for i := range rows {
		out = append(out, toBizRAGQueryLog(&rows[i]))
	}
	return out, total, nil
}

func (r *campusRepo) ListNotifications(ctx context.Context, userID, group string, offset, limit int) ([]*biz.CampusNotification, int64, error) {
	db := r.data.db.WithContext(ctx).Model(&campusNotificationModel{}).
		Where("recipient_id = ? AND is_deleted = ?", parseID(userID), false)
	switch group {
	case biz.CampusNotificationGroupReply:
		db = db.Where("event_type IN ?", []string{biz.CampusNotificationTypeComment, biz.CampusNotificationTypeReply})
	case biz.CampusNotificationGroupInteraction:
		db = db.Where("event_type IN ?", []string{biz.CampusNotificationTypePostLike, biz.CampusNotificationTypePostCollect, biz.CampusNotificationTypeCommentLike})
	case biz.CampusNotificationGroupSystem:
		db = db.Where("event_type = ?", biz.CampusNotificationTypeSystem)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []campusNotificationModel
	if err := db.Order("created_at DESC, id DESC").Offset(offset).Limit(limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	notifications := make([]*biz.CampusNotification, 0, len(rows))
	actorIDs := make([]int64, 0, len(rows))
	seenActors := map[int64]struct{}{}
	for i := range rows {
		notification := toBizNotification(&rows[i])
		notifications = append(notifications, notification)
		if rows[i].ActorID > 0 {
			if _, ok := seenActors[rows[i].ActorID]; !ok {
				seenActors[rows[i].ActorID] = struct{}{}
				actorIDs = append(actorIDs, rows[i].ActorID)
			}
		}
	}
	if len(actorIDs) > 0 {
		actors, err := r.loadCampusAuthors(ctx, actorIDs)
		if err != nil {
			return nil, 0, err
		}
		for _, notification := range notifications {
			if notification != nil {
				notification.Actor = actors[parseID(notification.ActorID)]
			}
		}
	}
	return notifications, total, nil
}

func (r *campusRepo) CountUnreadNotifications(ctx context.Context, userID string) (*biz.CampusUnreadNotificationCount, error) {
	base := func() *gorm.DB {
		return r.data.db.WithContext(ctx).Model(&campusNotificationModel{}).
			Where("recipient_id = ? AND read_at IS NULL AND is_deleted = ?", parseID(userID), false)
	}
	result := &biz.CampusUnreadNotificationCount{}
	if err := base().Count(&result.Total).Error; err != nil {
		return nil, err
	}
	if err := base().Where("event_type IN ?", []string{biz.CampusNotificationTypeComment, biz.CampusNotificationTypeReply}).Count(&result.Reply).Error; err != nil {
		return nil, err
	}
	if err := base().Where("event_type IN ?", []string{biz.CampusNotificationTypePostLike, biz.CampusNotificationTypePostCollect, biz.CampusNotificationTypeCommentLike}).Count(&result.Interaction).Error; err != nil {
		return nil, err
	}
	if err := base().Where("event_type = ?", biz.CampusNotificationTypeSystem).Count(&result.System).Error; err != nil {
		return nil, err
	}
	return result, nil
}

func (r *campusRepo) MarkNotificationRead(ctx context.Context, userID string, notificationID int64) error {
	return r.data.db.WithContext(ctx).Model(&campusNotificationModel{}).
		Where("id = ? AND recipient_id = ? AND is_deleted = ?", notificationID, parseID(userID), false).
		Updates(map[string]interface{}{"read_at": time.Now(), "updated_at": time.Now()}).Error
}

func (r *campusRepo) MarkAllNotificationsRead(ctx context.Context, userID string) error {
	return r.data.db.WithContext(ctx).Model(&campusNotificationModel{}).
		Where("recipient_id = ? AND read_at IS NULL AND is_deleted = ?", parseID(userID), false).
		Updates(map[string]interface{}{"read_at": time.Now(), "updated_at": time.Now()}).Error
}

func (r *campusRepo) ListNotificationRecipients(ctx context.Context) ([]string, error) {
	var rows []struct {
		ID int64 `gorm:"column:id"`
	}
	if err := r.data.db.WithContext(ctx).Table("user").
		Select("id").
		Order("id ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, fmt.Sprintf("%d", row.ID))
	}
	return out, nil
}

func (r *campusRepo) IsIPBlocked(ctx context.Context, ip string) (bool, error) {
	var count int64
	err := r.data.db.WithContext(ctx).Model(&campusIPBlockModel{}).
		Where("ip = ? AND status = ?", ip, biz.CampusIPBlockStatusActive).
		Count(&count).Error
	return count > 0, err
}

func (r *campusRepo) AllowCampusRequest(ctx context.Context, key string, limit int64, window time.Duration) (bool, error) {
	if r.data.rds == nil {
		return true, nil
	}
	count, err := r.data.rds.Incr(ctx, key).Result()
	if err != nil {
		return true, err
	}
	if count == 1 {
		_ = r.data.rds.Expire(ctx, key, window).Err()
	}
	return count <= limit, nil
}

func (r *campusRepo) CreateAccessLog(ctx context.Context, in *biz.CampusAccessLog) error {
	if in == nil {
		return nil
	}
	row := campusAccessLogModel{
		ID:          in.ID,
		UserID:      parseID(in.UserID),
		IP:          in.IP,
		Method:      in.Method,
		Path:        in.Path,
		StatusCode:  in.StatusCode,
		DurationMs:  in.DurationMs,
		UserAgent:   in.UserAgent,
		RateLimited: in.RateLimited,
		Blocked:     in.Blocked,
		CreatedAt:   time.Now(),
	}
	return r.data.db.WithContext(ctx).Create(&row).Error
}

func (r *campusRepo) CreateAccessLogs(ctx context.Context, logs []*biz.CampusAccessLog) error {
	if len(logs) == 0 {
		return nil
	}
	now := time.Now()
	rows := make([]campusAccessLogModel, 0, len(logs))
	for _, in := range logs {
		if in == nil {
			continue
		}
		createdAt := in.CreatedAt
		if createdAt.IsZero() {
			createdAt = now
		}
		rows = append(rows, campusAccessLogModel{
			ID:          in.ID,
			UserID:      parseID(in.UserID),
			IP:          in.IP,
			Method:      in.Method,
			Path:        in.Path,
			StatusCode:  in.StatusCode,
			DurationMs:  in.DurationMs,
			UserAgent:   in.UserAgent,
			RateLimited: in.RateLimited,
			Blocked:     in.Blocked,
			CreatedAt:   createdAt,
		})
	}
	if len(rows) == 0 {
		return nil
	}
	return r.data.db.WithContext(ctx).CreateInBatches(rows, 100).Error
}

func (r *campusRepo) GetSecurityOverview(ctx context.Context) (*biz.CampusSecurityOverview, error) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	overview := &biz.CampusSecurityOverview{}
	if err := r.data.db.WithContext(ctx).Table("campus_access_log").Where("created_at >= ?", today).Count(&overview.TodayRequests).Error; err != nil {
		return nil, err
	}
	if err := r.data.db.WithContext(ctx).Table("campus_access_log").Where("created_at >= ?", today).Distinct("ip").Count(&overview.TodayUniqueIPs).Error; err != nil {
		return nil, err
	}
	if err := r.data.db.WithContext(ctx).Table("campus_access_log").Where("created_at >= ? AND rate_limited = 1", today).Count(&overview.TodayRateLimited).Error; err != nil {
		return nil, err
	}
	if err := r.data.db.WithContext(ctx).Table("campus_access_log").Where("created_at >= ? AND blocked = 1", today).Count(&overview.TodayBlocked).Error; err != nil {
		return nil, err
	}
	if err := r.data.db.WithContext(ctx).Table("campus_access_log").Where("created_at >= ? AND status_code >= 400", today).Count(&overview.TodayErrors).Error; err != nil {
		return nil, err
	}
	if err := r.data.db.WithContext(ctx).Table("campus_ip_block").Where("status = ?", biz.CampusIPBlockStatusActive).Count(&overview.ActiveBlockedIPs).Error; err != nil {
		return nil, err
	}

	var ipRows []struct {
		IP           string    `gorm:"column:ip"`
		RequestCount int64     `gorm:"column:request_count"`
		ErrorCount   int64     `gorm:"column:error_count"`
		LastSeen     time.Time `gorm:"column:last_seen"`
	}
	if err := r.data.db.WithContext(ctx).Table("campus_access_log").
		Select("ip, COUNT(*) AS request_count, SUM(CASE WHEN status_code >= 400 THEN 1 ELSE 0 END) AS error_count, MAX(created_at) AS last_seen").
		Where("created_at >= ?", today).
		Group("ip").
		Order("request_count DESC").
		Limit(10).
		Find(&ipRows).Error; err != nil {
		return nil, err
	}
	overview.TopIPs = make([]*biz.CampusSecurityIPStat, 0, len(ipRows))
	for _, row := range ipRows {
		overview.TopIPs = append(overview.TopIPs, &biz.CampusSecurityIPStat{
			IP:           row.IP,
			RequestCount: row.RequestCount,
			ErrorCount:   row.ErrorCount,
			LastSeen:     row.LastSeen,
		})
	}

	var pathRows []struct {
		Path         string `gorm:"column:path"`
		RequestCount int64  `gorm:"column:request_count"`
		ErrorCount   int64  `gorm:"column:error_count"`
	}
	if err := r.data.db.WithContext(ctx).Table("campus_access_log").
		Select("path, COUNT(*) AS request_count, SUM(CASE WHEN status_code >= 400 THEN 1 ELSE 0 END) AS error_count").
		Where("created_at >= ?", today).
		Group("path").
		Order("request_count DESC").
		Limit(10).
		Find(&pathRows).Error; err != nil {
		return nil, err
	}
	overview.TopPaths = make([]*biz.CampusSecurityPathStat, 0, len(pathRows))
	for _, row := range pathRows {
		overview.TopPaths = append(overview.TopPaths, &biz.CampusSecurityPathStat{
			Path:         row.Path,
			RequestCount: row.RequestCount,
			ErrorCount:   row.ErrorCount,
		})
	}

	var logRows []campusAccessLogModel
	if err := r.data.db.WithContext(ctx).Order("created_at DESC, id DESC").Limit(30).Find(&logRows).Error; err != nil {
		return nil, err
	}
	overview.RecentAccessLogs = make([]*biz.CampusAccessLog, 0, len(logRows))
	for i := range logRows {
		overview.RecentAccessLogs = append(overview.RecentAccessLogs, toBizAccessLog(&logRows[i]))
	}
	var blockRows []campusIPBlockModel
	if err := r.data.db.WithContext(ctx).Where("status = ?", biz.CampusIPBlockStatusActive).Order("updated_at DESC").Limit(50).Find(&blockRows).Error; err != nil {
		return nil, err
	}
	overview.BlockedIPs = make([]*biz.CampusIPBlock, 0, len(blockRows))
	for i := range blockRows {
		overview.BlockedIPs = append(overview.BlockedIPs, toBizIPBlock(&blockRows[i]))
	}
	return overview, nil
}

func (r *campusRepo) BlockIP(ctx context.Context, block *biz.CampusIPBlock) error {
	row := campusIPBlockModel{
		ID:        block.ID,
		IP:        block.IP,
		Reason:    block.Reason,
		Status:    block.Status,
		CreatedBy: parseID(block.CreatedBy),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	return r.data.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "ip"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"reason":     block.Reason,
			"status":     biz.CampusIPBlockStatusActive,
			"created_by": parseID(block.CreatedBy),
			"updated_at": time.Now(),
		}),
	}).Create(&row).Error
}

func (r *campusRepo) UnblockIP(ctx context.Context, ip string) error {
	return r.data.db.WithContext(ctx).Model(&campusIPBlockModel{}).
		Where("ip = ?", ip).
		Updates(map[string]interface{}{
			"status":     biz.CampusIPBlockStatusInactive,
			"updated_at": time.Now(),
		}).Error
}

func (r *campusRepo) CreateAuditLog(ctx context.Context, in *biz.CampusAuditLog) error {
	row := campusAuditLogModel{
		ID:         in.ID,
		TargetType: in.TargetType,
		TargetID:   in.TargetID,
		UserID:     parseID(in.UserID),
		Provider:   in.Provider,
		Result:     in.Result,
		Reason:     in.Reason,
		CreatedAt:  time.Now(),
	}
	return r.data.db.WithContext(ctx).Create(&row).Error
}

func (r *campusRepo) TrackEvent(ctx context.Context, in *biz.TrackCampusEventInput) error {
	if in == nil {
		return nil
	}
	extra, _ := json.Marshal(in.Extra)
	row := campusEventModel{
		ID:         time.Now().UnixNano(),
		UserID:     parseID(in.UserID),
		EventType:  in.EventType,
		Page:       in.Page,
		TargetType: in.TargetType,
		TargetID:   in.TargetID,
		Channel:    in.Channel,
		Extra:      extra,
		UserAgent:  in.UserAgent,
		IP:         in.IP,
		CreatedAt:  time.Now(),
	}
	return r.data.db.WithContext(ctx).Create(&row).Error
}

func (r *campusRepo) TrackEvents(ctx context.Context, events []*biz.TrackCampusEventInput) error {
	if len(events) == 0 {
		return nil
	}
	now := time.Now()
	rows := make([]campusEventModel, 0, len(events))
	for _, in := range events {
		if in == nil {
			continue
		}
		extra, _ := json.Marshal(in.Extra)
		rows = append(rows, campusEventModel{
			ID:         time.Now().UnixNano() + int64(len(rows)),
			UserID:     parseID(in.UserID),
			EventType:  in.EventType,
			Page:       in.Page,
			TargetType: in.TargetType,
			TargetID:   in.TargetID,
			Channel:    in.Channel,
			Extra:      extra,
			UserAgent:  in.UserAgent,
			IP:         in.IP,
			CreatedAt:  now,
		})
	}
	if len(rows) == 0 {
		return nil
	}
	return r.data.db.WithContext(ctx).CreateInBatches(rows, 100).Error
}

func (r *campusRepo) GetAdminSummary(ctx context.Context) (*biz.CampusAdminSummary, error) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	summary := &biz.CampusAdminSummary{}
	counts := []struct {
		table string
		where string
		dest  *int64
	}{
		{"user", "1 = 1", &summary.TotalUsers},
		{"user", "created_at >= ?", &summary.TodayUsers},
		{"campus_event", "event_type = 'login'", &summary.TotalLogins},
		{"campus_event", "event_type = 'login' AND created_at >= ?", &summary.TodayLogins},
		{"campus_event", "event_type = 'visit'", &summary.TotalVisits},
		{"campus_event", "event_type = 'visit' AND created_at >= ?", &summary.TodayVisits},
		{"campus_event", "event_type = 'share'", &summary.TotalShares},
		{"campus_event", "event_type = 'share' AND created_at >= ?", &summary.TodayShares},
		{"campus_event", "event_type = 'publish_open' AND created_at >= ?", &summary.TodayPublishOpen},
		{"campus_event", "event_type = 'publish_success' AND created_at >= ?", &summary.TodayPublishDone},
		{"campus_event", "event_type = 'post_detail_visit' AND created_at >= ?", &summary.TodayDetailViews},
		{"campus_event", "event_type = 'feedback_create' AND created_at >= ?", &summary.TodayFeedback},
		{"campus_event", "event_type = 'report_create' AND created_at >= ?", &summary.TodayReports},
		{"campus_forum_post", "is_deleted = 0", &summary.TotalPosts},
		{"campus_forum_post", "is_deleted = 0 AND created_at >= ?", &summary.TodayPosts},
		{"campus_forum_comment", "is_deleted = 0", &summary.TotalComments},
		{"campus_forum_comment", "is_deleted = 0 AND created_at >= ?", &summary.TodayComments},
		{"campus_forum_post_like", "is_deleted = 0", &summary.TotalLikes},
		{"campus_forum_post_like", "is_deleted = 0 AND created_at >= ?", &summary.TodayLikes},
		{"campus_forum_post_collection", "is_deleted = 0", &summary.TotalCollections},
		{"campus_forum_post_collection", "is_deleted = 0 AND created_at >= ?", &summary.TodayCollections},
		{"campus_forum_report", "1 = 1", &summary.TotalReports},
		{"campus_forum_report", "status = 0", &summary.PendingReports},
		{"campus_feedback", "status = 0", &summary.PendingFeedback},
		{"campus_forum_post", "status = 0 AND is_deleted = 0", &summary.PendingPosts},
		{"campus_forum_comment", "status = 0 AND is_deleted = 0", &summary.PendingComments},
		{"campus_forum_post", "is_featured = 1 AND is_deleted = 0", &summary.FeaturedPosts},
		{"campus_forum_post", "is_official = 1 AND is_deleted = 0", &summary.OfficialPosts},
	}
	for _, item := range counts {
		db := r.data.db.WithContext(ctx).Table(item.table).Where(item.where)
		if strings.Contains(item.where, "created_at >= ?") {
			db = r.data.db.WithContext(ctx).Table(item.table).Where(item.where, today)
		}
		if err := db.Count(item.dest).Error; err != nil {
			return nil, err
		}
	}
	trends := make([]*biz.CampusAdminTrend, 0, 7)
	for i := 6; i >= 0; i-- {
		day := today.AddDate(0, 0, -i)
		next := day.AddDate(0, 0, 1)
		trend := &biz.CampusAdminTrend{Date: day.Format("01-02")}
		if err := r.data.db.WithContext(ctx).Table("user").Where("created_at >= ? AND created_at < ?", day, next).Count(&trend.Users).Error; err != nil {
			return nil, err
		}
		if err := r.data.db.WithContext(ctx).Table("campus_event").Where("event_type = ? AND created_at >= ? AND created_at < ?", "login", day, next).Count(&trend.Logins).Error; err != nil {
			return nil, err
		}
		if err := r.data.db.WithContext(ctx).Table("campus_event").Where("event_type = ? AND created_at >= ? AND created_at < ?", "visit", day, next).Count(&trend.Visits).Error; err != nil {
			return nil, err
		}
		if err := r.data.db.WithContext(ctx).Table("campus_event").Where("event_type = ? AND created_at >= ? AND created_at < ?", "share", day, next).Count(&trend.Shares).Error; err != nil {
			return nil, err
		}
		if err := r.data.db.WithContext(ctx).Table("campus_forum_post").Where("is_deleted = 0 AND created_at >= ? AND created_at < ?", day, next).Count(&trend.Posts).Error; err != nil {
			return nil, err
		}
		if err := r.data.db.WithContext(ctx).Table("campus_forum_comment").Where("is_deleted = 0 AND created_at >= ? AND created_at < ?", day, next).Count(&trend.Comments).Error; err != nil {
			return nil, err
		}
		if err := r.data.db.WithContext(ctx).Table("campus_forum_post_like").Where("is_deleted = 0 AND created_at >= ? AND created_at < ?", day, next).Count(&trend.Likes).Error; err != nil {
			return nil, err
		}
		if err := r.data.db.WithContext(ctx).Table("campus_forum_post_collection").Where("is_deleted = 0 AND created_at >= ? AND created_at < ?", day, next).Count(&trend.Collections).Error; err != nil {
			return nil, err
		}
		if err := r.data.db.WithContext(ctx).Table("campus_forum_report").Where("created_at >= ? AND created_at < ?", day, next).Count(&trend.Reports).Error; err != nil {
			return nil, err
		}
		trends = append(trends, trend)
	}
	summary.Trends = trends
	return summary, nil
}

func (r *campusRepo) ReconcileCampusStats(ctx context.Context) (*biz.CampusStatsReconcileResult, error) {
	result := &biz.CampusStatsReconcileResult{CheckedAt: time.Now()}
	err := r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		postSQL := `
			UPDATE campus_forum_post p
			LEFT JOIN (
				SELECT post_id, COUNT(*) AS real_likes
				FROM campus_forum_post_like
				WHERE is_deleted = 0
				GROUP BY post_id
			) likes ON p.id = likes.post_id
			LEFT JOIN (
				SELECT post_id, COUNT(*) AS real_collections
				FROM campus_forum_post_collection
				WHERE is_deleted = 0
				GROUP BY post_id
			) collections ON p.id = collections.post_id
			LEFT JOIN (
				SELECT post_id, COUNT(*) AS real_comments
				FROM campus_forum_comment
				WHERE is_deleted = 0 AND status = ?
				GROUP BY post_id
			) comments ON p.id = comments.post_id
			SET
				p.like_count = COALESCE(likes.real_likes, 0),
				p.collected_count = COALESCE(collections.real_collections, 0),
				p.comment_count = COALESCE(comments.real_comments, 0),
				p.updated_at = NOW(3)
			WHERE p.is_deleted = 0
			  AND (
				p.like_count != COALESCE(likes.real_likes, 0)
				OR p.collected_count != COALESCE(collections.real_collections, 0)
				OR p.comment_count != COALESCE(comments.real_comments, 0)
			  )
		`
		postRes := tx.Exec(postSQL, biz.CampusAuditStatusVisible)
		if postRes.Error != nil {
			return postRes.Error
		}
		result.UpdatedPosts = postRes.RowsAffected

		commentSQL := `
			UPDATE campus_forum_comment c
			LEFT JOIN (
				SELECT comment_id, COUNT(*) AS real_likes
				FROM campus_forum_comment_like
				WHERE is_deleted = 0
				GROUP BY comment_id
			) likes ON c.id = likes.comment_id
			LEFT JOIN (
				SELECT parent_id, COUNT(*) AS real_replies
				FROM campus_forum_comment
				WHERE is_deleted = 0 AND status = ? AND parent_id > 0
				GROUP BY parent_id
			) replies ON c.id = replies.parent_id
			SET
				c.like_count = COALESCE(likes.real_likes, 0),
				c.reply_count = COALESCE(replies.real_replies, 0),
				c.updated_at = NOW(3)
			WHERE c.is_deleted = 0
			  AND (
				c.like_count != COALESCE(likes.real_likes, 0)
				OR c.reply_count != COALESCE(replies.real_replies, 0)
			  )
		`
		commentRes := tx.Exec(commentSQL, biz.CampusAuditStatusVisible)
		if commentRes.Error != nil {
			return commentRes.Error
		}
		result.UpdatedComments = commentRes.RowsAffected
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *campusRepo) ListCampusUsers(ctx context.Context, keyword, role string, authStatus int32, offset, limit int) ([]*biz.CampusAdminUser, int64, error) {
	db := r.data.db.WithContext(ctx).Table("user u").
		Select(`u.id AS user_id, u.account_id, u.mobile, u.email, u.name, u.nickname, u.avatar,
			COALESCE(p.school_name, '') AS school_name,
			COALESCE(p.student_no, '') AS student_no,
			COALESCE(p.real_name, '') AS real_name,
			COALESCE(p.class_name, '') AS class_name,
			COALESCE(p.dorm_building, '') AS dorm_building,
			COALESCE(p.room_no, '') AS room_no,
			COALESCE(p.auth_status, 0) AS auth_status,
			COALESCE(o.role, '') AS role,
			COALESCE(post_stats.post_count, 0) AS post_count,
			COALESCE(comment_stats.comment_count, 0) AS comment_count,
			COALESCE(like_stats.like_count, 0) AS like_count,
			COALESCE(collection_stats.collection_count, 0) AS collection_count,
			COALESCE(feedback_stats.feedback_count, 0) AS feedback_count,
			COALESCE(report_stats.report_count, 0) AS report_count,
			COALESCE(login_stats.login_count, 0) AS login_count,
			COALESCE(visit_stats.visit_count, 0) AS visit_count,
			login_stats.last_login_at AS last_login_at,
			access_stats.last_active_at AS last_active_at,
			COALESCE(access_stats.last_active_ip, '') AS last_active_ip,
			COALESCE(access_stats.last_active_path, '') AS last_active_path,
			COALESCE(access_stats.last_active_status, 0) AS last_active_status,
			u.created_at, u.updated_at`).
		Joins("LEFT JOIN campus_profile p ON p.user_id = u.id").
		Joins("LEFT JOIN campus_operator o ON o.user_id = u.id AND o.is_deleted = 0").
		Joins(`LEFT JOIN (
			SELECT author_id AS user_id, COUNT(*) AS post_count
			FROM campus_forum_post
			WHERE is_deleted = 0
			GROUP BY author_id
		) post_stats ON post_stats.user_id = u.id`).
		Joins(`LEFT JOIN (
			SELECT author_id AS user_id, COUNT(*) AS comment_count
			FROM campus_forum_comment
			WHERE is_deleted = 0
			GROUP BY author_id
		) comment_stats ON comment_stats.user_id = u.id`).
		Joins(`LEFT JOIN (
			SELECT user_id, COUNT(*) AS like_count
			FROM campus_forum_post_like
			WHERE is_deleted = 0
			GROUP BY user_id
		) like_stats ON like_stats.user_id = u.id`).
		Joins(`LEFT JOIN (
			SELECT user_id, COUNT(*) AS collection_count
			FROM campus_forum_post_collection
			WHERE is_deleted = 0
			GROUP BY user_id
		) collection_stats ON collection_stats.user_id = u.id`).
		Joins(`LEFT JOIN (
			SELECT user_id, COUNT(*) AS feedback_count
			FROM campus_feedback
			GROUP BY user_id
		) feedback_stats ON feedback_stats.user_id = u.id`).
		Joins(`LEFT JOIN (
			SELECT reporter_id AS user_id, COUNT(*) AS report_count
			FROM campus_forum_report
			GROUP BY reporter_id
		) report_stats ON report_stats.user_id = u.id`).
		Joins(`LEFT JOIN (
			SELECT user_id, COUNT(*) AS login_count, MAX(created_at) AS last_login_at
			FROM campus_event
			WHERE event_type = 'login'
			GROUP BY user_id
		) login_stats ON login_stats.user_id = u.id`).
		Joins(`LEFT JOIN (
			SELECT user_id, COUNT(*) AS visit_count
			FROM campus_event
			WHERE event_type = 'visit'
			GROUP BY user_id
		) visit_stats ON visit_stats.user_id = u.id`).
		Joins(`LEFT JOIN (
			SELECT l.user_id, l.created_at AS last_active_at, l.ip AS last_active_ip, l.path AS last_active_path, l.status_code AS last_active_status
			FROM campus_access_log l
			INNER JOIN (
				SELECT user_id, MAX(id) AS max_id
				FROM campus_access_log
				WHERE user_id > 0
				GROUP BY user_id
			) latest ON latest.max_id = l.id
		) access_stats ON access_stats.user_id = u.id`)
	if keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where("u.nickname LIKE ? OR u.name LIKE ? OR u.mobile LIKE ? OR u.email LIKE ? OR p.student_no LIKE ? OR p.real_name LIKE ?", like, like, like, like, like, like)
	}
	if authStatus >= 0 {
		db = db.Where("COALESCE(p.auth_status, 0) = ?", authStatus)
	}
	switch role {
	case "admin", "operator":
		db = db.Where("o.role = ?", role)
	case "user":
		db = db.Where("o.role IS NULL OR o.role = ''")
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []campusUserRow
	if err := db.Order("u.created_at DESC, u.id DESC").Offset(offset).Limit(limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	users := make([]*biz.CampusAdminUser, 0, len(rows))
	for i := range rows {
		users = append(users, toBizAdminUser(&rows[i]))
	}
	return users, total, nil
}

func (r *campusRepo) GetCampusOperatorRole(ctx context.Context, userID string) (string, error) {
	var row campusOperatorModel
	err := r.data.db.WithContext(ctx).
		Where("user_id = ? AND is_deleted = ?", parseID(userID), false).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return row.Role, nil
}

func (r *campusRepo) UpsertCampusOperator(ctx context.Context, userID, role string) error {
	row := campusOperatorModel{
		UserID:    parseID(userID),
		Role:      role,
		IsDeleted: false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	return r.data.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"role":       role,
				"is_deleted": false,
				"updated_at": time.Now(),
			}),
		}).
		Create(&row).Error
}

func (r *campusRepo) RemoveCampusOperator(ctx context.Context, userID string) error {
	return r.data.db.WithContext(ctx).Model(&campusOperatorModel{}).
		Where("user_id = ?", parseID(userID)).
		Updates(map[string]interface{}{
			"is_deleted": true,
			"updated_at": time.Now(),
		}).Error
}

func toBizWechatIdentity(row *campusWechatIdentityModel) *biz.CampusWechatIdentity {
	return &biz.CampusWechatIdentity{
		ID:        row.ID,
		Provider:  row.Provider,
		OpenID:    row.OpenID,
		UnionID:   row.UnionID,
		UserID:    fmt.Sprintf("%d", row.UserID),
		AccountID: fmt.Sprintf("%d", row.AccountID),
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func toBizProfile(row *campusProfileModel) *biz.CampusProfile {
	return &biz.CampusProfile{
		ID:           row.ID,
		UserID:       fmt.Sprintf("%d", row.UserID),
		AccountID:    fmt.Sprintf("%d", row.AccountID),
		OpenID:       row.OpenID,
		UnionID:      row.UnionID,
		SchoolName:   row.SchoolName,
		StudentNo:    row.StudentNo,
		RealName:     row.RealName,
		ClassName:    row.ClassName,
		DormBuilding: row.DormBuilding,
		RoomNo:       row.RoomNo,
		Mobile:       row.Mobile,
		AuthStatus:   row.AuthStatus,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
}

func toBizCategory(row *campusForumCategoryModel) *biz.CampusForumCategory {
	return &biz.CampusForumCategory{
		ID:          row.ID,
		Code:        row.Code,
		Name:        row.Name,
		Description: row.Description,
		SortOrder:   row.SortOrder,
	}
}

func toBizTimetableCourse(row *campusTimetableCourseModel) *biz.CampusTimetableCourse {
	return &biz.CampusTimetableCourse{
		ID:             row.ID,
		UserID:         fmt.Sprintf("%d", row.UserID),
		Term:           row.Term,
		CourseName:     row.CourseName,
		Teacher:        row.Teacher,
		Classroom:      row.Classroom,
		Weekday:        row.Weekday,
		StartSection:   row.StartSection,
		EndSection:     row.EndSection,
		StartWeek:      row.StartWeek,
		EndWeek:        row.EndWeek,
		WeekParity:     row.WeekParity,
		Source:         row.Source,
		SourceCourseID: row.SourceCourseID,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}

func toBizPost(row *campusForumPostModel) *biz.CampusForumPost {
	images := make([]string, 0)
	_ = json.Unmarshal(row.Images, &images)
	extra := make(map[string]string)
	_ = json.Unmarshal(row.Extra, &extra)
	postType := row.PostType
	if postType == "" {
		postType = biz.CampusPostTypeNote
	}
	return &biz.CampusForumPost{
		ID:             row.ID,
		CategoryCode:   row.CategoryCode,
		AuthorID:       fmt.Sprintf("%d", row.AuthorID),
		Title:          row.Title,
		Content:        row.Content,
		Images:         images,
		MediaType:      row.MediaType,
		PostType:       postType,
		Extra:          extra,
		CoverURL:       row.CoverURL,
		VideoURL:       row.VideoURL,
		IsOfficial:     row.IsOfficial,
		IsFeatured:     row.IsFeatured,
		IsPinned:       row.IsPinned,
		SortWeight:     row.SortWeight,
		Status:         row.Status,
		AuditReason:    row.AuditReason,
		LikeCount:      row.LikeCount,
		CommentCount:   row.CommentCount,
		CollectedCount: row.CollectedCount,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}

func toBizReport(row *campusForumReportModel) *biz.CampusForumReport {
	return &biz.CampusForumReport{
		ID:         row.ID,
		TargetType: row.TargetType,
		TargetID:   row.TargetID,
		ReporterID: fmt.Sprintf("%d", row.ReporterID),
		Reason:     row.Reason,
		Detail:     row.Detail,
		Status:     row.Status,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}
}

func toBizFeedback(row *campusFeedbackModel) *biz.CampusFeedback {
	images := make([]string, 0)
	_ = json.Unmarshal(row.Images, &images)
	return &biz.CampusFeedback{
		ID:           row.ID,
		UserID:       fmt.Sprintf("%d", row.UserID),
		FeedbackType: row.FeedbackType,
		Content:      row.Content,
		Contact:      row.Contact,
		Images:       images,
		Status:       row.Status,
		OperatorNote: row.OperatorNote,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
}

func toKnowledgeDocumentModel(in *biz.CampusKnowledgeDocument) campusKnowledgeDocumentModel {
	now := time.Now()
	status := in.Status
	if status == "" {
		status = biz.CampusKnowledgeDocumentStatusDraft
	}
	parseStatus := in.ParseStatus
	if parseStatus == "" {
		parseStatus = status
	}
	return campusKnowledgeDocumentModel{
		ID:           in.ID,
		Title:        trimLimitData(in.Title, 120),
		Source:       trimLimitData(in.Source, 120),
		Category:     trimLimitData(in.Category, 32),
		ContentType:  trimLimitData(in.ContentType, 16),
		FileURL:      trimLimitData(in.FileURL, 1024),
		FileID:       parseID(in.FileID),
		FileType:     trimLimitData(in.FileType, 16),
		RawContent:   in.RawContent,
		Status:       status,
		ParseStatus:  parseStatus,
		ErrorMessage: trimLimitData(in.ErrorMessage, 1000),
		UploadedBy:   parseID(in.UploadedBy),
		EffectiveAt:  in.EffectiveAt,
		ExpiredAt:    in.ExpiredAt,
		ChunkCount:   in.ChunkCount,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func toBizKnowledgeDocument(row *campusKnowledgeDocumentModel) *biz.CampusKnowledgeDocument {
	if row == nil {
		return nil
	}
	return &biz.CampusKnowledgeDocument{
		ID:           row.ID,
		Title:        row.Title,
		Source:       row.Source,
		Category:     row.Category,
		ContentType:  row.ContentType,
		FileURL:      row.FileURL,
		FileID:       fmt.Sprintf("%d", row.FileID),
		FileType:     row.FileType,
		RawContent:   row.RawContent,
		Status:       row.Status,
		ParseStatus:  row.ParseStatus,
		ErrorMessage: row.ErrorMessage,
		UploadedBy:   fmt.Sprintf("%d", row.UploadedBy),
		EffectiveAt:  row.EffectiveAt,
		ExpiredAt:    row.ExpiredAt,
		ChunkCount:   row.ChunkCount,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
}

func toKnowledgeChunkModel(in *biz.CampusKnowledgeChunk) campusKnowledgeChunkModel {
	now := time.Now()
	keywords, _ := json.Marshal(in.Keywords)
	status := in.Status
	if status == "" {
		status = biz.CampusKnowledgeChunkStatusActive
	}
	embeddingStatus := in.EmbeddingStatus
	if embeddingStatus == "" {
		embeddingStatus = "done"
	}
	return campusKnowledgeChunkModel{
		ID:              in.ID,
		DocumentID:      in.DocumentID,
		ChunkIndex:      in.ChunkIndex,
		Title:           trimLimitData(in.Title, 120),
		Content:         in.Content,
		Summary:         trimLimitData(in.Summary, 500),
		Category:        trimLimitData(in.Category, 32),
		Keywords:        keywords,
		Source:          trimLimitData(in.Source, 120),
		Status:          status,
		QdrantPointID:   trimLimitData(in.QdrantPointID, 128),
		EmbeddingStatus: trimLimitData(embeddingStatus, 24),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

func toBizKnowledgeChunk(row *campusKnowledgeChunkModel) *biz.CampusKnowledgeChunk {
	if row == nil {
		return nil
	}
	keywords := make([]string, 0)
	_ = json.Unmarshal(row.Keywords, &keywords)
	return &biz.CampusKnowledgeChunk{
		ID:              row.ID,
		DocumentID:      row.DocumentID,
		ChunkIndex:      row.ChunkIndex,
		Title:           row.Title,
		Content:         row.Content,
		Summary:         row.Summary,
		Category:        row.Category,
		Keywords:        keywords,
		Source:          row.Source,
		Status:          row.Status,
		QdrantPointID:   row.QdrantPointID,
		EmbeddingStatus: row.EmbeddingStatus,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}

func toRAGQueryLogModel(in *biz.CampusRAGQueryLog) campusRAGQueryLogModel {
	hitChunks, _ := json.Marshal(in.HitChunks)
	now := time.Now()
	if !in.CreatedAt.IsZero() {
		now = in.CreatedAt
	}
	return campusRAGQueryLogModel{
		ID:               in.ID,
		UserID:           parseID(in.UserID),
		PostID:           in.PostID,
		TriggerCommentID: in.TriggerCommentID,
		Query:            trimLimitData(in.Query, 1000),
		NeedKnowledge:    in.NeedKnowledge,
		Confidence:       in.Confidence,
		HitChunks:        hitChunks,
		Answer:           trimLimitData(in.Answer, 1000),
		Model:            trimLimitData(in.Model, 64),
		DurationMs:       in.DurationMs,
		ErrorMessage:     trimLimitData(in.ErrorMessage, 1000),
		CreatedAt:        now,
	}
}

func toBizRAGQueryLog(row *campusRAGQueryLogModel) *biz.CampusRAGQueryLog {
	if row == nil {
		return nil
	}
	chunks := make([]*biz.CampusRAGQueryChunk, 0)
	_ = json.Unmarshal(row.HitChunks, &chunks)
	return &biz.CampusRAGQueryLog{
		ID:               row.ID,
		UserID:           fmt.Sprintf("%d", row.UserID),
		PostID:           row.PostID,
		TriggerCommentID: row.TriggerCommentID,
		Query:            row.Query,
		NeedKnowledge:    row.NeedKnowledge,
		Confidence:       row.Confidence,
		HitChunks:        chunks,
		Answer:           row.Answer,
		Model:            row.Model,
		DurationMs:       row.DurationMs,
		ErrorMessage:     row.ErrorMessage,
		CreatedAt:        row.CreatedAt,
	}
}

func toNotificationModel(in *biz.CampusNotification) campusNotificationModel {
	linkParams, _ := json.Marshal(in.LinkParams)
	now := time.Now()
	var dedupeKey *string
	if strings.TrimSpace(in.DedupeKey) != "" {
		key := strings.TrimSpace(in.DedupeKey)
		dedupeKey = &key
	}
	return campusNotificationModel{
		ID:          in.ID,
		RecipientID: parseID(in.RecipientID),
		ActorID:     parseID(in.ActorID),
		EventType:   in.EventType,
		TargetType:  in.TargetType,
		TargetID:    in.TargetID,
		DedupeKey:   dedupeKey,
		Title:       in.Title,
		Content:     in.Content,
		LinkPage:    in.LinkPage,
		LinkParams:  linkParams,
		ReadAt:      in.ReadAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func toNotificationOutboxModel(in *biz.CampusNotificationOutbox) campusNotificationOutboxModel {
	linkParams, _ := json.Marshal(in.LinkParams)
	now := time.Now()
	var dedupeKey *string
	if strings.TrimSpace(in.DedupeKey) != "" {
		key := strings.TrimSpace(in.DedupeKey)
		dedupeKey = &key
	}
	status := in.Status
	if status == "" {
		status = biz.CampusNotificationOutboxStatusPending
	}
	return campusNotificationOutboxModel{
		ID:          in.ID,
		RecipientID: parseID(in.RecipientID),
		ActorID:     parseID(in.ActorID),
		EventType:   in.EventType,
		TargetType:  in.TargetType,
		TargetID:    in.TargetID,
		DedupeKey:   dedupeKey,
		Title:       in.Title,
		Content:     in.Content,
		LinkPage:    in.LinkPage,
		LinkParams:  linkParams,
		Audience:    in.Audience,
		Status:      status,
		RetryCount:  in.RetryCount,
		NextRetryAt: in.NextRetryAt,
		LockedUntil: in.LockedUntil,
		LastError:   in.LastError,
		CreatedAt:   now,
		UpdatedAt:   now,
		ProcessedAt: in.ProcessedAt,
	}
}

func toBizNotificationOutbox(row *campusNotificationOutboxModel) *biz.CampusNotificationOutbox {
	linkParams := make(map[string]string)
	_ = json.Unmarshal(row.LinkParams, &linkParams)
	dedupeKey := ""
	if row.DedupeKey != nil {
		dedupeKey = *row.DedupeKey
	}
	return &biz.CampusNotificationOutbox{
		ID:          row.ID,
		RecipientID: fmt.Sprintf("%d", row.RecipientID),
		ActorID:     fmt.Sprintf("%d", row.ActorID),
		EventType:   row.EventType,
		TargetType:  row.TargetType,
		TargetID:    row.TargetID,
		DedupeKey:   dedupeKey,
		Title:       row.Title,
		Content:     row.Content,
		LinkPage:    row.LinkPage,
		LinkParams:  linkParams,
		Audience:    row.Audience,
		Status:      row.Status,
		RetryCount:  row.RetryCount,
		NextRetryAt: row.NextRetryAt,
		LockedUntil: row.LockedUntil,
		LastError:   row.LastError,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
		ProcessedAt: row.ProcessedAt,
	}
}

func toAIReplyTaskModel(in *biz.CampusAIReplyTask) campusAIReplyTaskModel {
	now := time.Now()
	status := in.Status
	if status == "" {
		status = biz.CampusAIReplyTaskStatusPending
	}
	return campusAIReplyTaskModel{
		ID:               in.ID,
		PostID:           in.PostID,
		RootCommentID:    in.RootCommentID,
		TriggerCommentID: in.TriggerCommentID,
		AskerID:          parseID(in.AskerID),
		BotUserID:        parseID(in.BotUserID),
		Prompt:           in.Prompt,
		Status:           status,
		RetryCount:       in.RetryCount,
		NextRetryAt:      in.NextRetryAt,
		LockedUntil:      in.LockedUntil,
		AnswerCommentID:  in.AnswerCommentID,
		LastError:        in.LastError,
		CreatedAt:        now,
		UpdatedAt:        now,
		ProcessedAt:      in.ProcessedAt,
	}
}

func toBizAIReplyTask(row *campusAIReplyTaskModel) *biz.CampusAIReplyTask {
	if row == nil {
		return nil
	}
	return &biz.CampusAIReplyTask{
		ID:               row.ID,
		PostID:           row.PostID,
		RootCommentID:    row.RootCommentID,
		TriggerCommentID: row.TriggerCommentID,
		AskerID:          fmt.Sprintf("%d", row.AskerID),
		BotUserID:        fmt.Sprintf("%d", row.BotUserID),
		Prompt:           row.Prompt,
		Status:           row.Status,
		RetryCount:       row.RetryCount,
		NextRetryAt:      row.NextRetryAt,
		LockedUntil:      row.LockedUntil,
		AnswerCommentID:  row.AnswerCommentID,
		LastError:        row.LastError,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
		ProcessedAt:      row.ProcessedAt,
	}
}

func toAIContentAuditTaskModel(in *biz.CampusAIContentAuditTask) campusAIContentAuditTaskModel {
	now := time.Now()
	status := in.Status
	if status == "" {
		status = biz.CampusAIContentAuditTaskStatusPending
	}
	return campusAIContentAuditTaskModel{
		ID:          in.ID,
		TargetType:  in.TargetType,
		TargetID:    in.TargetID,
		Status:      status,
		RiskLevel:   in.RiskLevel,
		Decision:    in.Decision,
		Reason:      in.Reason,
		RawResult:   in.RawResult,
		RetryCount:  in.RetryCount,
		NextRetryAt: in.NextRetryAt,
		LockedUntil: in.LockedUntil,
		LastError:   in.LastError,
		CreatedAt:   now,
		UpdatedAt:   now,
		ProcessedAt: in.ProcessedAt,
	}
}

func toBizAIContentAuditTask(row *campusAIContentAuditTaskModel) *biz.CampusAIContentAuditTask {
	if row == nil {
		return nil
	}
	return &biz.CampusAIContentAuditTask{
		ID:          row.ID,
		TargetType:  row.TargetType,
		TargetID:    row.TargetID,
		Status:      row.Status,
		RiskLevel:   row.RiskLevel,
		Decision:    row.Decision,
		Reason:      row.Reason,
		RawResult:   row.RawResult,
		RetryCount:  row.RetryCount,
		NextRetryAt: row.NextRetryAt,
		LockedUntil: row.LockedUntil,
		LastError:   row.LastError,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
		ProcessedAt: row.ProcessedAt,
	}
}

func toBizNotification(row *campusNotificationModel) *biz.CampusNotification {
	linkParams := make(map[string]string)
	_ = json.Unmarshal(row.LinkParams, &linkParams)
	dedupeKey := ""
	if row.DedupeKey != nil {
		dedupeKey = *row.DedupeKey
	}
	return &biz.CampusNotification{
		ID:          row.ID,
		RecipientID: fmt.Sprintf("%d", row.RecipientID),
		ActorID:     fmt.Sprintf("%d", row.ActorID),
		EventType:   row.EventType,
		TargetType:  row.TargetType,
		TargetID:    row.TargetID,
		DedupeKey:   dedupeKey,
		Title:       row.Title,
		Content:     row.Content,
		LinkPage:    row.LinkPage,
		LinkParams:  linkParams,
		ReadAt:      row.ReadAt,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}

func notificationDedupeKey(recipientID, actorID int64, eventType, targetType string, targetID int64) string {
	if recipientID == 0 || actorID == 0 || strings.TrimSpace(eventType) == "" || strings.TrimSpace(targetType) == "" || targetID == 0 {
		return ""
	}
	return fmt.Sprintf("%d:%d:%s:%s:%d", recipientID, actorID, eventType, targetType, targetID)
}

func createNotificationOutboxWithTx(tx *gorm.DB, outbox *biz.CampusNotificationOutbox) error {
	if outbox == nil {
		return nil
	}
	row := toNotificationOutboxModel(outbox)
	db := tx
	if row.DedupeKey != nil && strings.TrimSpace(*row.DedupeKey) != "" {
		db = db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "dedupe_key"}},
			DoNothing: true,
		})
	}
	return db.Create(&row).Error
}

func toBizAccessLog(row *campusAccessLogModel) *biz.CampusAccessLog {
	return &biz.CampusAccessLog{
		ID:          row.ID,
		UserID:      fmt.Sprintf("%d", row.UserID),
		IP:          row.IP,
		Method:      row.Method,
		Path:        row.Path,
		StatusCode:  row.StatusCode,
		DurationMs:  row.DurationMs,
		UserAgent:   row.UserAgent,
		RateLimited: row.RateLimited,
		Blocked:     row.Blocked,
		CreatedAt:   row.CreatedAt,
	}
}

func toBizIPBlock(row *campusIPBlockModel) *biz.CampusIPBlock {
	return &biz.CampusIPBlock{
		ID:        row.ID,
		IP:        row.IP,
		Reason:    row.Reason,
		Status:    row.Status,
		CreatedBy: fmt.Sprintf("%d", row.CreatedBy),
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func (r *campusRepo) loadCampusAuthors(ctx context.Context, ids []int64) (map[int64]*biz.CampusForumAuthor, error) {
	result := make(map[int64]*biz.CampusForumAuthor, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	var rows []struct {
		ID       int64  `gorm:"column:id"`
		Name     string `gorm:"column:name"`
		Nickname string `gorm:"column:nickname"`
		Avatar   string `gorm:"column:avatar"`
	}
	if err := r.data.db.WithContext(ctx).Table("user").
		Select("id, name, nickname, avatar").
		Where("id IN ?", ids).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.ID] = &biz.CampusForumAuthor{
			UserID:   fmt.Sprintf("%d", row.ID),
			Name:     firstNonEmptyData(row.Nickname, row.Name, "同学"),
			Nickname: row.Nickname,
			Avatar:   row.Avatar,
		}
	}
	return result, nil
}

func toBizAdminUser(row *campusUserRow) *biz.CampusAdminUser {
	userID := fmt.Sprintf("%d", row.UserID)
	accountID := fmt.Sprintf("%d", row.AccountID)
	var lastLoginAt time.Time
	if row.LastLoginAt.Valid {
		lastLoginAt = row.LastLoginAt.Time
	}
	var lastActiveAt time.Time
	if row.LastActiveAt.Valid {
		lastActiveAt = row.LastActiveAt.Time
	}
	return &biz.CampusAdminUser{
		User: &biz.UserBaseInfo{
			ID:        userID,
			Name:      row.Name,
			Nickname:  row.Nickname,
			Avatar:    row.Avatar,
			Mobile:    row.Mobile,
			Email:     row.Email,
			CreatedAt: row.CreatedAt.Format(time.DateTime),
			UpdatedAt: row.UpdatedAt.Format(time.DateTime),
		},
		Profile: &biz.CampusProfile{
			UserID:       userID,
			AccountID:    accountID,
			SchoolName:   row.SchoolName,
			StudentNo:    row.StudentNo,
			RealName:     row.RealName,
			ClassName:    row.ClassName,
			DormBuilding: row.DormBuilding,
			RoomNo:       row.RoomNo,
			Mobile:       row.Mobile,
			AuthStatus:   row.AuthStatus,
		},
		Role:             row.Role,
		PostCount:        row.PostCount,
		CommentCount:     row.CommentCount,
		LikeCount:        row.LikeCount,
		CollectionCount:  row.CollectionCount,
		FeedbackCount:    row.FeedbackCount,
		ReportCount:      row.ReportCount,
		LoginCount:       row.LoginCount,
		VisitCount:       row.VisitCount,
		LastLoginAt:      lastLoginAt,
		LastActiveAt:     lastActiveAt,
		LastActiveIP:     row.LastActiveIP,
		LastActivePath:   row.LastActivePath,
		LastActiveStatus: row.LastActiveStatus,
	}
}

func toBizComment(row *campusForumCommentModel) *biz.CampusForumComment {
	images := make([]string, 0)
	_ = json.Unmarshal(row.Images, &images)
	return &biz.CampusForumComment{
		ID:               row.ID,
		PostID:           row.PostID,
		ParentID:         row.ParentID,
		ReplyToCommentID: row.ReplyToCommentID,
		ReplyToUserID:    fmt.Sprintf("%d", row.ReplyToUserID),
		AuthorID:         fmt.Sprintf("%d", row.AuthorID),
		Content:          row.Content,
		Images:           images,
		Status:           row.Status,
		AuditReason:      row.AuditReason,
		LikeCount:        row.LikeCount,
		ReplyCount:       row.ReplyCount,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}
}

func (r *campusRepo) fillPostCategoryNames(ctx context.Context, posts []*biz.CampusForumPost) error {
	if len(posts) == 0 {
		return nil
	}
	codes := make([]string, 0, len(posts))
	seen := map[string]struct{}{}
	for _, post := range posts {
		if post.CategoryCode == "" {
			continue
		}
		if _, ok := seen[post.CategoryCode]; ok {
			continue
		}
		seen[post.CategoryCode] = struct{}{}
		codes = append(codes, post.CategoryCode)
	}
	if len(codes) == 0 {
		return nil
	}
	var rows []campusForumCategoryModel
	if err := r.data.db.WithContext(ctx).
		Where("code IN ?", codes).
		Find(&rows).Error; err != nil {
		return err
	}
	names := make(map[string]string, len(rows))
	for _, row := range rows {
		names[row.Code] = row.Name
	}
	for _, post := range posts {
		post.CategoryName = names[post.CategoryCode]
	}
	return nil
}

func nullString(value string) interface{} {
	if value == "" {
		return sql.NullString{}
	}
	return value
}

func firstNonEmptyData(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func trimLimitData(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

func parseID(value string) int64 {
	id, _ := strconv.ParseInt(value, 10, 64)
	return id
}
