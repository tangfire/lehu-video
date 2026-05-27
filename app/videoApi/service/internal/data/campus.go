package data

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
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
	ID          int64           `gorm:"column:id"`
	PostID      int64           `gorm:"column:post_id"`
	AuthorID    int64           `gorm:"column:author_id"`
	Content     string          `gorm:"column:content"`
	Images      json.RawMessage `gorm:"column:images"`
	Status      int32           `gorm:"column:status"`
	AuditReason string          `gorm:"column:audit_reason"`
	LikeCount   int64           `gorm:"column:like_count"`
	IsDeleted   bool            `gorm:"column:is_deleted"`
	CreatedAt   time.Time       `gorm:"column:created_at"`
	UpdatedAt   time.Time       `gorm:"column:updated_at"`
}

func (campusForumCommentModel) TableName() string { return "campus_forum_comment" }

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

type campusUserRow struct {
	UserID       int64     `gorm:"column:user_id"`
	AccountID    int64     `gorm:"column:account_id"`
	Mobile       string    `gorm:"column:mobile"`
	Email        string    `gorm:"column:email"`
	Name         string    `gorm:"column:name"`
	Nickname     string    `gorm:"column:nickname"`
	Avatar       string    `gorm:"column:avatar"`
	SchoolName   string    `gorm:"column:school_name"`
	StudentNo    string    `gorm:"column:student_no"`
	RealName     string    `gorm:"column:real_name"`
	ClassName    string    `gorm:"column:class_name"`
	DormBuilding string    `gorm:"column:dorm_building"`
	RoomNo       string    `gorm:"column:room_no"`
	AuthStatus   int32     `gorm:"column:auth_status"`
	Role         string    `gorm:"column:role"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
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
	if query.Keyword != "" {
		keyword := "%" + query.Keyword + "%"
		db = db.Where("(campus_forum_post.title LIKE ? OR campus_forum_post.content LIKE ?)", keyword, keyword)
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
	order := "campus_forum_post.is_pinned DESC, campus_forum_post.is_featured DESC, campus_forum_post.sort_weight DESC, campus_forum_post.created_at DESC, campus_forum_post.id DESC"
	if query.CollectedByUserID != "" {
		order = "c.updated_at DESC, c.id DESC"
	}
	if query.Sort == "hot" {
		order = "campus_forum_post.is_pinned DESC, campus_forum_post.is_featured DESC, campus_forum_post.sort_weight DESC, (campus_forum_post.like_count * 3 + campus_forum_post.comment_count * 5 + campus_forum_post.collected_count * 4) DESC, campus_forum_post.created_at DESC"
	}
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
	images, _ := json.Marshal(comment.Images)
	row := campusForumCommentModel{
		ID:          comment.ID,
		PostID:      comment.PostID,
		AuthorID:    parseID(comment.AuthorID),
		Content:     comment.Content,
		Images:      images,
		Status:      comment.Status,
		AuditReason: comment.AuditReason,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
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

func (r *campusRepo) DeleteComment(ctx context.Context, commentID int64) error {
	return r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var comment campusForumCommentModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND is_deleted = ?", commentID, false).
			First(&comment).Error; err != nil {
			return err
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
		if comment.Status == biz.CampusAuditStatusVisible {
			return tx.Model(&campusForumPostModel{}).
				Where("id = ?", comment.PostID).
				UpdateColumn("comment_count", gorm.Expr("GREATEST(comment_count - ?, 0)", 1)).Error
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
		return tx.Model(&campusForumPostModel{}).
			Where("id = ?", comment.PostID).
			UpdateColumn("comment_count", gorm.Expr("GREATEST(comment_count + ?, 0)", delta)).Error
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
			return tx.Model(&campusForumPostModel{}).
				Where("id = ?", postID).
				UpdateColumn("like_count", gorm.Expr("GREATEST(like_count + ?, 0)", 1)).Error
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
		return tx.Model(&campusForumPostModel{}).
			Where("id = ?", postID).
			UpdateColumn("like_count", gorm.Expr("GREATEST(like_count + ?, 0)", 1)).Error
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
			return tx.Model(&campusForumPostModel{}).
				Where("id = ?", postID).
				UpdateColumn("collected_count", gorm.Expr("GREATEST(collected_count + ?, 0)", 1)).Error
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
		return tx.Model(&campusForumPostModel{}).
			Where("id = ?", postID).
			UpdateColumn("collected_count", gorm.Expr("GREATEST(collected_count + ?, 0)", 1)).Error
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
		{"campus_forum_post", "status = 0 AND is_deleted = 0", &summary.PendingPosts},
		{"campus_forum_comment", "status = 0 AND is_deleted = 0", &summary.PendingComments},
		{"campus_forum_post", "is_featured = 1 AND is_deleted = 0", &summary.FeaturedPosts},
		{"campus_forum_post", "is_official = 1 AND is_deleted = 0", &summary.OfficialPosts},
	}
	for _, item := range counts {
		db := r.data.db.WithContext(ctx).Table(item.table).Where(item.where)
		if item.where == "created_at >= ?" || item.where == "is_deleted = 0 AND created_at >= ?" {
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

func (r *campusRepo) ListCampusUsers(ctx context.Context, keyword string, offset, limit int) ([]*biz.CampusAdminUser, int64, error) {
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
			u.created_at, u.updated_at`).
		Joins("LEFT JOIN campus_profile p ON p.user_id = u.id").
		Joins("LEFT JOIN campus_operator o ON o.user_id = u.id AND o.is_deleted = 0")
	if keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where("u.nickname LIKE ? OR u.name LIKE ? OR u.mobile LIKE ? OR u.email LIKE ? OR p.student_no LIKE ? OR p.real_name LIKE ?", like, like, like, like, like, like)
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

func toBizAdminUser(row *campusUserRow) *biz.CampusAdminUser {
	userID := fmt.Sprintf("%d", row.UserID)
	accountID := fmt.Sprintf("%d", row.AccountID)
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
		Role: row.Role,
	}
}

func toBizComment(row *campusForumCommentModel) *biz.CampusForumComment {
	images := make([]string, 0)
	_ = json.Unmarshal(row.Images, &images)
	return &biz.CampusForumComment{
		ID:          row.ID,
		PostID:      row.PostID,
		AuthorID:    fmt.Sprintf("%d", row.AuthorID),
		Content:     row.Content,
		Images:      images,
		Status:      row.Status,
		AuditReason: row.AuditReason,
		LikeCount:   row.LikeCount,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
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

func parseID(value string) int64 {
	id, _ := strconv.ParseInt(value, 10, 64)
	return id
}
