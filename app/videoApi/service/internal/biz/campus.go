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

	CampusAuthStatusUnverified int32 = 0
	CampusAuthStatusVerified   int32 = 1

	CampusPostMediaText  = "text"
	CampusPostMediaImage = "image"
	CampusPostMediaVideo = "video"
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
	CoverURL       string
	VideoURL       string
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
	ID          int64
	PostID      int64
	AuthorID    string
	Author      *CampusForumAuthor
	Content     string
	Images      []string
	Status      int32
	AuditReason string
	LikeCount   int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CampusForumReport struct {
	ID         int64
	TargetType string
	TargetID   int64
	ReporterID string
	Reason     string
	Detail     string
	Status     int32
	CreatedAt  time.Time
	UpdatedAt  time.Time
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

type CreateCampusPostInput struct {
	UserID       string
	CategoryCode string
	Title        string
	Content      string
	Images       []string
	MediaType    string
	CoverURL     string
	VideoURL     string
}

type ListCampusPostsInput struct {
	CurrentUserID string
	CategoryCode  string
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
	Sort              string
	Keyword           string
	AuthorID          string
	CollectedByUserID string
	Statuses          []int32
	IncludeDeleted    bool
	Offset            int
	Limit             int
}

type GetCampusPostInput struct {
	CurrentUserID string
	PostID        int64
}

type CreateCampusCommentInput struct {
	UserID  string
	PostID  int64
	Content string
	Images  []string
}

type ListCampusCommentsInput struct {
	PostID int64
	Page   int32
	Size   int32
}

type ListCampusCommentsOutput struct {
	Comments []*CampusForumComment
	Total    int64
}

type ListCampusCommentQuery struct {
	PostID         int64
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

type ReviewCampusContentInput struct {
	UserID     string
	TargetType string
	TargetID   int64
	Action     string
	Reason     string
}

type CampusRepo interface {
	GetWechatIdentity(ctx context.Context, provider, openID string) (bool, *CampusWechatIdentity, error)
	SaveWechatIdentity(ctx context.Context, identity *CampusWechatIdentity) error
	GetProfileByUserID(ctx context.Context, userID string) (bool, *CampusProfile, error)
	SaveProfile(ctx context.Context, profile *CampusProfile) error
	UpdateProfile(ctx context.Context, profile *CampusProfile) error
	ListCategories(ctx context.Context) ([]*CampusForumCategory, error)
	GetCategoryByCode(ctx context.Context, code string) (bool, *CampusForumCategory, error)
	CreatePost(ctx context.Context, post *CampusForumPost) error
	ListPosts(ctx context.Context, query ListCampusPostQuery) ([]*CampusForumPost, int64, error)
	GetPostByID(ctx context.Context, postID int64) (bool, *CampusForumPost, error)
	GetAnyPostByID(ctx context.Context, postID int64) (bool, *CampusForumPost, error)
	DeletePost(ctx context.Context, postID int64) error
	UpdatePostStatus(ctx context.Context, postID int64, status int32, reason string) error
	CreateComment(ctx context.Context, comment *CampusForumComment) error
	ListComments(ctx context.Context, query ListCampusCommentQuery) ([]*CampusForumComment, int64, error)
	GetCommentByID(ctx context.Context, commentID int64) (bool, *CampusForumComment, error)
	DeleteComment(ctx context.Context, commentID int64) error
	UpdateCommentStatus(ctx context.Context, commentID int64, status int32, reason string) error
	GetPostLikeStatus(ctx context.Context, userID string, postIDs []int64) (map[int64]bool, error)
	AddPostLike(ctx context.Context, id int64, userID string, postID int64) error
	RemovePostLike(ctx context.Context, userID string, postID int64) error
	GetPostCollectionStatus(ctx context.Context, userID string, postIDs []int64) (map[int64]bool, error)
	AddPostCollection(ctx context.Context, id int64, userID string, postID int64) error
	RemovePostCollection(ctx context.Context, userID string, postID int64) error
	CreateReport(ctx context.Context, report *CampusForumReport) error
	CreateAuditLog(ctx context.Context, log *CampusAuditLog) error
}

type CampusUsecase struct {
	repo       CampusRepo
	base       BaseAdapter
	core       CoreAdapter
	idGen      CampusIDGenerator
	authSecret string
	log        *log.Helper
}

func NewCampusUsecase(repo CampusRepo, base BaseAdapter, core CoreAdapter, idGen CampusIDGenerator, authSecret string, logger log.Logger) *CampusUsecase {
	return &CampusUsecase{
		repo:       repo,
		base:       base,
		core:       core,
		idGen:      idGen,
		authSecret: authSecret,
		log:        log.NewHelper(logger),
	}
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

	return &WechatLoginOutput{Token: token, Profile: profile, User: user}, nil
}

func (uc *CampusUsecase) createWechatAccountAndUser(ctx context.Context, openID, nickname, avatar string) (string, string, error) {
	email := "wx_" + shortHash(openID, 24) + "@wechat.local"
	password := "Wx#" + shortHash(openID+uc.authSecret, 40)

	accountID, err := uc.base.Register(ctx, "", email, password)
	if err != nil {
		accountID, err = uc.base.CheckAccount(ctx, "", email, password)
		if err != nil {
			return "", "", apperror.Internal(err, "创建微信账号失败")
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
	if size <= 0 || size > 80<<20 {
		return "", "", apperror.InvalidArgument("视频不能超过 80MB")
	}
	fileID, url, err := uc.base.PreSign4PublicUpload(ctx, hash, fileType, filename, size, 3600)
	if err != nil {
		return "", "", apperror.Internal(err, "创建视频上传地址失败")
	}
	return fileID, url, nil
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
	audit := auditText(ctx, "post", title+"\n"+content)
	if audit.Blocked {
		return nil, apperror.InvalidArgument(audit.Reason)
	}
	status := CampusAuditStatusVisible
	if audit.Pending {
		status = CampusAuditStatusPending
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
		CoverURL:     coverURL,
		VideoURL:     videoURL,
		Status:       status,
		AuditReason:  audit.Reason,
	}
	if err := uc.repo.CreatePost(ctx, post); err != nil {
		return nil, apperror.Internal(err, "发布帖子失败")
	}
	_ = uc.repo.CreateAuditLog(ctx, &CampusAuditLog{
		ID:         uc.idGen.NextID(),
		TargetType: "post",
		TargetID:   post.ID,
		UserID:     input.UserID,
		Provider:   audit.Provider,
		Result:     audit.Result,
		Reason:     audit.Reason,
	})
	_ = uc.hydratePosts(ctx, []*CampusForumPost{post}, input.UserID)
	return post, nil
}

func (uc *CampusUsecase) ListPosts(ctx context.Context, input *ListCampusPostsInput) (*ListCampusPostsOutput, error) {
	page, size := normalizePage(input.Page, input.Size)
	posts, total, err := uc.repo.ListPosts(ctx, ListCampusPostQuery{
		CategoryCode: strings.TrimSpace(input.CategoryCode),
		Sort:         strings.TrimSpace(input.Sort),
		Keyword:      strings.TrimSpace(input.Keyword),
		Statuses:     []int32{CampusAuditStatusVisible},
		Offset:       int((page - 1) * size),
		Limit:        int(size),
	})
	if err != nil {
		return nil, apperror.Internal(err, "获取帖子列表失败")
	}
	if err := uc.hydratePosts(ctx, posts, input.CurrentUserID); err != nil {
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
		Sort:           strings.TrimSpace(input.Sort),
		Keyword:        strings.TrimSpace(input.Keyword),
		AuthorID:       input.CurrentUserID,
		IncludeDeleted: false,
		Offset:         int((page - 1) * size),
		Limit:          int(size),
	})
	if err != nil {
		return nil, apperror.Internal(err, "获取我的帖子失败")
	}
	if err := uc.hydratePosts(ctx, posts, input.CurrentUserID); err != nil {
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
		Sort:              strings.TrimSpace(input.Sort),
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
	if err := uc.hydratePosts(ctx, posts, input.CurrentUserID); err != nil {
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
	if err := uc.hydratePosts(ctx, []*CampusForumPost{post}, input.CurrentUserID); err != nil {
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
	if post.AuthorID != userID && !uc.isCampusAdmin(userID) {
		return apperror.Forbidden("只能删除自己的帖子")
	}
	if err := uc.repo.DeletePost(ctx, postID); err != nil {
		return apperror.Internal(err, "删除帖子失败")
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
	content := strings.TrimSpace(input.Content)
	if len([]rune(content)) < 1 || len([]rune(content)) > 500 {
		return nil, apperror.InvalidArgument("评论需要 1-500 个字")
	}
	audit := auditText(ctx, "comment", content)
	if audit.Blocked {
		return nil, apperror.InvalidArgument(audit.Reason)
	}
	status := CampusAuditStatusVisible
	if audit.Pending {
		status = CampusAuditStatusPending
	}
	comment := &CampusForumComment{
		ID:          uc.idGen.NextID(),
		PostID:      input.PostID,
		AuthorID:    input.UserID,
		Content:     content,
		Images:      sanitizeImages(input.Images, 3),
		Status:      status,
		AuditReason: audit.Reason,
	}
	if err := uc.repo.CreateComment(ctx, comment); err != nil {
		return nil, apperror.Internal(err, "发表评论失败")
	}
	_ = uc.repo.CreateAuditLog(ctx, &CampusAuditLog{
		ID:         uc.idGen.NextID(),
		TargetType: "comment",
		TargetID:   comment.ID,
		UserID:     input.UserID,
		Provider:   audit.Provider,
		Result:     audit.Result,
		Reason:     audit.Reason,
	})
	_ = uc.hydrateComments(ctx, []*CampusForumComment{comment})
	return comment, nil
}

func (uc *CampusUsecase) ListComments(ctx context.Context, input *ListCampusCommentsInput) (*ListCampusCommentsOutput, error) {
	if input.PostID <= 0 {
		return nil, apperror.InvalidArgument("帖子 ID 无效")
	}
	page, size := normalizePage(input.Page, input.Size)
	comments, total, err := uc.repo.ListComments(ctx, ListCampusCommentQuery{
		PostID:   input.PostID,
		Statuses: []int32{CampusAuditStatusVisible},
		Offset:   int((page - 1) * size),
		Limit:    int(size),
	})
	if err != nil {
		return nil, apperror.Internal(err, "获取评论失败")
	}
	if err := uc.hydrateComments(ctx, comments); err != nil {
		uc.log.WithContext(ctx).Warnf("hydrate campus comments failed: %v", err)
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
	if comment.AuthorID != userID && !uc.isCampusAdmin(userID) {
		return apperror.Forbidden("只能删除自己的评论")
	}
	if err := uc.repo.DeleteComment(ctx, commentID); err != nil {
		return apperror.Internal(err, "删除评论失败")
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
	if !uc.isCampusAdmin(input.UserID) {
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
	if err := uc.hydratePosts(ctx, posts, input.UserID); err != nil {
		uc.log.WithContext(ctx).Warnf("hydrate moderation posts failed: %v", err)
	}
	return &ListCampusPostsOutput{Posts: posts, Total: total}, nil
}

func (uc *CampusUsecase) ListModerationComments(ctx context.Context, input *ListCampusModerationInput) (*ListCampusCommentsOutput, error) {
	if !uc.isCampusAdmin(input.UserID) {
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
	if err := uc.hydrateComments(ctx, comments); err != nil {
		uc.log.WithContext(ctx).Warnf("hydrate moderation comments failed: %v", err)
	}
	return &ListCampusCommentsOutput{Comments: comments, Total: total}, nil
}

func (uc *CampusUsecase) ReviewContent(ctx context.Context, input *ReviewCampusContentInput) error {
	if !uc.isCampusAdmin(input.UserID) {
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
		ok, _, err := uc.repo.GetCommentByID(ctx, input.TargetID)
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

func (uc *CampusUsecase) hydratePosts(ctx context.Context, posts []*CampusForumPost, currentUserID string) error {
	if len(posts) == 0 {
		return nil
	}
	userIDs := make([]string, 0, len(posts))
	postIDs := make([]int64, 0, len(posts))
	seen := map[string]struct{}{}
	for _, post := range posts {
		postIDs = append(postIDs, post.ID)
		if _, ok := seen[post.AuthorID]; !ok && post.AuthorID != "" {
			seen[post.AuthorID] = struct{}{}
			userIDs = append(userIDs, post.AuthorID)
		}
	}
	authors, err := uc.loadAuthors(ctx, userIDs)
	if err != nil {
		return err
	}
	likeStatus := map[int64]bool{}
	collectionStatus := map[int64]bool{}
	if currentUserID != "" && currentUserID != "0" {
		likeStatus, _ = uc.repo.GetPostLikeStatus(ctx, currentUserID, postIDs)
		collectionStatus, _ = uc.repo.GetPostCollectionStatus(ctx, currentUserID, postIDs)
	}
	for _, post := range posts {
		post.Author = authors[post.AuthorID]
		post.IsLiked = likeStatus[post.ID]
		post.IsCollected = collectionStatus[post.ID]
	}
	return nil
}

func (uc *CampusUsecase) hydrateComments(ctx context.Context, comments []*CampusForumComment) error {
	if len(comments) == 0 {
		return nil
	}
	userIDs := make([]string, 0, len(comments))
	seen := map[string]struct{}{}
	for _, comment := range comments {
		if _, ok := seen[comment.AuthorID]; !ok && comment.AuthorID != "" {
			seen[comment.AuthorID] = struct{}{}
			userIDs = append(userIDs, comment.AuthorID)
		}
	}
	authors, err := uc.loadAuthors(ctx, userIDs)
	if err != nil {
		return err
	}
	for _, comment := range comments {
		comment.Author = authors[comment.AuthorID]
	}
	return nil
}

func (uc *CampusUsecase) loadAuthors(ctx context.Context, userIDs []string) (map[string]*CampusForumAuthor, error) {
	authors := make(map[string]*CampusForumAuthor, len(userIDs))
	if len(userIDs) == 0 {
		return authors, nil
	}
	users, err := uc.core.BatchGetUserBaseInfo(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	for _, user := range users {
		if user == nil {
			continue
		}
		author := &CampusForumAuthor{
			UserID:   user.ID,
			Name:     firstNonEmpty(user.Nickname, user.Name, "同学"),
			Nickname: user.Nickname,
			Avatar:   user.Avatar,
		}
		if ok, profile, err := uc.repo.GetProfileByUserID(ctx, user.ID); err == nil && ok {
			author.SchoolName = profile.SchoolName
			author.AuthStatus = profile.AuthStatus
		}
		authors[user.ID] = author
	}
	return authors, nil
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

func (uc *CampusUsecase) isCampusAdmin(userID string) bool {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return false
	}
	allowList := strings.TrimSpace(os.Getenv("LEHU_CAMPUS_ADMIN_USER_IDS"))
	if allowList == "" {
		return true
	}
	for _, item := range strings.Split(allowList, ",") {
		if strings.TrimSpace(item) == userID {
			return true
		}
	}
	return false
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
