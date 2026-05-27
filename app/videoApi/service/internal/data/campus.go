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
	ID           int64           `gorm:"column:id"`
	CategoryCode string          `gorm:"column:category_code"`
	AuthorID     int64           `gorm:"column:author_id"`
	Title        string          `gorm:"column:title"`
	Content      string          `gorm:"column:content"`
	Images       json.RawMessage `gorm:"column:images"`
	MediaType    string          `gorm:"column:media_type"`
	CoverURL     string          `gorm:"column:cover_url"`
	VideoURL     string          `gorm:"column:video_url"`
	Status       int32           `gorm:"column:status"`
	AuditReason  string          `gorm:"column:audit_reason"`
	LikeCount    int64           `gorm:"column:like_count"`
	CommentCount int64           `gorm:"column:comment_count"`
	IsDeleted    bool            `gorm:"column:is_deleted"`
	CreatedAt    time.Time       `gorm:"column:created_at"`
	UpdatedAt    time.Time       `gorm:"column:updated_at"`
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
	row := campusForumPostModel{
		ID:           post.ID,
		CategoryCode: post.CategoryCode,
		AuthorID:     parseID(post.AuthorID),
		Title:        post.Title,
		Content:      post.Content,
		Images:       images,
		MediaType:    post.MediaType,
		CoverURL:     post.CoverURL,
		VideoURL:     post.VideoURL,
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
		db = db.Where("is_deleted = ?", false)
	}
	if len(query.Statuses) > 0 {
		db = db.Where("status IN ?", query.Statuses)
	}
	if query.CategoryCode != "" {
		db = db.Where("category_code = ?", query.CategoryCode)
	}
	if query.AuthorID != "" {
		db = db.Where("author_id = ?", parseID(query.AuthorID))
	}
	if query.Keyword != "" {
		keyword := "%" + query.Keyword + "%"
		db = db.Where("(title LIKE ? OR content LIKE ?)", keyword, keyword)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	order := "created_at DESC, id DESC"
	if query.Sort == "hot" {
		order = "(like_count * 3 + comment_count * 5) DESC, created_at DESC"
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
	if len(query.Statuses) > 0 {
		db = db.Where("status IN ?", query.Statuses)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []campusForumCommentModel
	if err := db.Order("created_at ASC, id ASC").Offset(query.Offset).Limit(query.Limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	comments := make([]*biz.CampusForumComment, 0, len(rows))
	for i := range rows {
		comments = append(comments, toBizComment(&rows[i]))
	}
	return comments, total, nil
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

func toBizPost(row *campusForumPostModel) *biz.CampusForumPost {
	images := make([]string, 0)
	_ = json.Unmarshal(row.Images, &images)
	return &biz.CampusForumPost{
		ID:           row.ID,
		CategoryCode: row.CategoryCode,
		AuthorID:     fmt.Sprintf("%d", row.AuthorID),
		Title:        row.Title,
		Content:      row.Content,
		Images:       images,
		MediaType:    row.MediaType,
		CoverURL:     row.CoverURL,
		VideoURL:     row.VideoURL,
		Status:       row.Status,
		AuditReason:  row.AuditReason,
		LikeCount:    row.LikeCount,
		CommentCount: row.CommentCount,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
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

func parseID(value string) int64 {
	id, _ := strconv.ParseInt(value, 10, 64)
	return id
}
