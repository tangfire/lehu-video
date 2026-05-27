package service

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/gorilla/mux"
	"lehu-video/app/videoApi/service/internal/biz"
	"lehu-video/app/videoApi/service/internal/pkg/resp"
	"lehu-video/app/videoApi/service/internal/pkg/utils/claims"
	"lehu-video/pkg/apperror"
	sharedauth "lehu-video/pkg/auth"
)

type CampusService struct {
	uc         *biz.CampusUsecase
	authSecret string
}

func NewCampusService(uc *biz.CampusUsecase, authSecret string) *CampusService {
	return &CampusService{uc: uc, authSecret: authSecret}
}

func (s *CampusService) RegisterRoutes(srv *khttp.Server) {
	r := srv.Route("/")
	r.POST("/v1/auth/wechat-login", s.wrap(s.handleWechatLogin))
	r.GET("/v1/campus/profile", s.wrap(s.authRequired(s.handleGetProfile)))
	r.PUT("/v1/campus/profile", s.wrap(s.authRequired(s.handleUpdateProfile)))
	r.GET("/v1/campus/timetable", s.wrap(s.authRequired(s.handleListTimetable)))
	r.POST("/v1/campus/timetable/import", s.wrap(s.authRequired(s.handleImportTimetable)))
	r.POST("/v1/campus/upload/image", s.wrap(s.authRequired(s.handleUploadImage)))
	r.POST("/v1/campus/upload/video", s.wrap(s.authRequired(s.handleUploadVideo)))
	r.GET("/v1/campus/forum/categories", s.wrap(s.handleListCategories))
	r.GET("/v1/campus/forum/posts", s.wrap(s.handleListPosts))
	r.POST("/v1/campus/forum/posts", s.wrap(s.authRequired(s.handleCreatePost)))
	r.GET("/v1/campus/forum/my-posts", s.wrap(s.authRequired(s.handleListMyPosts)))
	r.GET("/v1/campus/forum/my-collections", s.wrap(s.authRequired(s.handleListMyCollections)))
	r.GET("/v1/campus/forum/posts/{id}", s.wrap(s.handleGetPost))
	r.DELETE("/v1/campus/forum/posts/{id}", s.wrap(s.authRequired(s.handleDeletePost)))
	r.GET("/v1/campus/forum/posts/{id}/comments", s.wrap(s.handleListComments))
	r.POST("/v1/campus/forum/posts/{id}/comments", s.wrap(s.authRequired(s.handleCreateComment)))
	r.POST("/v1/campus/forum/posts/{id}/like", s.wrap(s.authRequired(s.handleLikePost)))
	r.DELETE("/v1/campus/forum/posts/{id}/like", s.wrap(s.authRequired(s.handleUnlikePost)))
	r.POST("/v1/campus/forum/posts/{id}/collection", s.wrap(s.authRequired(s.handleCollectPost)))
	r.DELETE("/v1/campus/forum/posts/{id}/collection", s.wrap(s.authRequired(s.handleUncollectPost)))
	r.POST("/v1/campus/forum/posts/{id}/report", s.wrap(s.authRequired(s.handleReportPost)))
	r.DELETE("/v1/campus/forum/comments/{id}", s.wrap(s.authRequired(s.handleDeleteComment)))
	r.POST("/v1/campus/forum/comments/{id}/report", s.wrap(s.authRequired(s.handleReportComment)))
	r.GET("/v1/campus/moderation/posts", s.wrap(s.authRequired(s.handleListModerationPosts)))
	r.GET("/v1/campus/moderation/comments", s.wrap(s.authRequired(s.handleListModerationComments)))
	r.POST("/v1/campus/moderation/posts/{id}/review", s.wrap(s.authRequired(s.handleReviewPost)))
	r.POST("/v1/campus/moderation/comments/{id}/review", s.wrap(s.authRequired(s.handleReviewComment)))
}

func (s *CampusService) wrap(handler http.HandlerFunc) khttp.HandlerFunc {
	return func(ctx khttp.Context) error {
		handler(ctx.Response(), ctx.Request())
		return nil
	}
}

type wechatLoginRequest struct {
	Code     string `json:"code"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

func (s *CampusService) handleWechatLogin(w http.ResponseWriter, r *http.Request) {
	var req wechatLoginRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	out, err := s.uc.WechatLogin(r.Context(), &biz.WechatLoginInput{
		Code:     req.Code,
		Nickname: req.Nickname,
		Avatar:   req.Avatar,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{
		"token":   out.Token,
		"profile": profileToMap(out.Profile),
		"user":    userToMap(out.User),
	})
}

type profileRequest struct {
	SchoolName   string `json:"school_name"`
	StudentNo    string `json:"student_no"`
	RealName     string `json:"real_name"`
	ClassName    string `json:"class_name"`
	DormBuilding string `json:"dorm_building"`
	RoomNo       string `json:"room_no"`
	Mobile       string `json:"mobile"`
}

type importTimetableRequest struct {
	StudentNo string `json:"student_no"`
	Password  string `json:"password"`
	Term      string `json:"term"`
}

func (s *CampusService) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	userID, _ := s.userIDFromRequest(r)
	profile, err := s.uc.GetProfile(r.Context(), userID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{"profile": profileToMap(profile)})
}

func (s *CampusService) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	var req profileRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	userID, _ := s.userIDFromRequest(r)
	profile, err := s.uc.UpdateProfile(r.Context(), &biz.UpdateCampusProfileInput{
		UserID:       userID,
		SchoolName:   req.SchoolName,
		StudentNo:    req.StudentNo,
		RealName:     req.RealName,
		ClassName:    req.ClassName,
		DormBuilding: req.DormBuilding,
		RoomNo:       req.RoomNo,
		Mobile:       req.Mobile,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{"profile": profileToMap(profile)})
}

func (s *CampusService) handleListTimetable(w http.ResponseWriter, r *http.Request) {
	userID, _ := s.userIDFromRequest(r)
	out, err := s.uc.ListTimetable(r.Context(), &biz.ListCampusTimetableInput{
		UserID: userID,
		Term:   r.URL.Query().Get("term"),
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	courses := make([]map[string]interface{}, 0, len(out.Courses))
	for _, course := range out.Courses {
		courses = append(courses, timetableCourseToMap(course))
	}
	writeJSON(w, r, map[string]interface{}{
		"term":    out.Term,
		"courses": courses,
	})
}

func (s *CampusService) handleImportTimetable(w http.ResponseWriter, r *http.Request) {
	var req importTimetableRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	userID, _ := s.userIDFromRequest(r)
	out, err := s.uc.ImportTimetable(r.Context(), &biz.ImportCampusTimetableInput{
		UserID:    userID,
		StudentNo: req.StudentNo,
		Password:  req.Password,
		Term:      req.Term,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	courses := make([]map[string]interface{}, 0, len(out.Courses))
	for _, course := range out.Courses {
		courses = append(courses, timetableCourseToMap(course))
	}
	writeJSON(w, r, map[string]interface{}{
		"term":    out.Term,
		"count":   out.Count,
		"courses": courses,
	})
}

func (s *CampusService) handleUploadImage(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		writeError(w, r, apperror.InvalidArgument("图片上传请求无效"))
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, r, apperror.InvalidArgument("请选择图片"))
		return
	}
	defer file.Close()
	if header.Size > 5<<20 {
		writeError(w, r, apperror.InvalidArgument("图片不能超过 5MB"))
		return
	}
	data, err := io.ReadAll(io.LimitReader(file, 5<<20+1))
	if err != nil {
		writeError(w, r, apperror.Internal(err, "读取图片失败"))
		return
	}
	if len(data) > 5<<20 {
		writeError(w, r, apperror.InvalidArgument("图片不能超过 5MB"))
		return
	}
	fileType := imageFileType(header.Filename, http.DetectContentType(data))
	if fileType == "" {
		writeError(w, r, apperror.InvalidArgument("仅支持 jpg、png、webp 图片"))
		return
	}
	sum := md5.Sum(data)
	fileID, putURL, err := s.uc.PreSignPublicImage(r.Context(), fmt.Sprintf("%x", sum), fileType, header.Filename, int64(len(data)))
	if err != nil {
		writeError(w, r, err)
		return
	}
	if putURL != "" {
		uploadURL, signedHost := rewritePresignedURLForServerUpload(putURL)
		req, err := http.NewRequestWithContext(r.Context(), http.MethodPut, uploadURL, bytes.NewReader(data))
		if err != nil {
			writeError(w, r, apperror.Internal(err, "创建图片上传请求失败"))
			return
		}
		if signedHost != "" {
			req.Host = signedHost
		}
		req.Header.Set("Content-Type", contentTypeFromImageType(fileType))
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			writeError(w, r, apperror.DependencyUnavailable(err, "上传图片失败"))
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			writeError(w, r, apperror.DependencyUnavailable(fmt.Errorf("minio status %d", resp.StatusCode), "上传图片失败"))
			return
		}
	}
	url, err := s.uc.ReportPublicImageUploaded(r.Context(), fileID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{"url": url, "file_id": fileID})
}

func (s *CampusService) handleUploadVideo(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(90 << 20); err != nil {
		writeError(w, r, apperror.InvalidArgument("视频上传请求无效"))
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, r, apperror.InvalidArgument("请选择视频"))
		return
	}
	defer file.Close()
	if header.Size > 80<<20 {
		writeError(w, r, apperror.InvalidArgument("视频不能超过 80MB"))
		return
	}
	data, err := io.ReadAll(io.LimitReader(file, 80<<20+1))
	if err != nil {
		writeError(w, r, apperror.Internal(err, "读取视频失败"))
		return
	}
	if len(data) > 80<<20 {
		writeError(w, r, apperror.InvalidArgument("视频不能超过 80MB"))
		return
	}
	fileType := videoFileType(header.Filename, http.DetectContentType(data))
	if fileType == "" {
		writeError(w, r, apperror.InvalidArgument("仅支持 mp4、mov 视频"))
		return
	}
	sum := md5.Sum(data)
	fileID, putURL, err := s.uc.PreSignPublicVideo(r.Context(), fmt.Sprintf("%x", sum), fileType, header.Filename, int64(len(data)))
	if err != nil {
		writeError(w, r, err)
		return
	}
	if putURL != "" {
		uploadURL, signedHost := rewritePresignedURLForServerUpload(putURL)
		req, err := http.NewRequestWithContext(r.Context(), http.MethodPut, uploadURL, bytes.NewReader(data))
		if err != nil {
			writeError(w, r, apperror.Internal(err, "创建视频上传请求失败"))
			return
		}
		if signedHost != "" {
			req.Host = signedHost
		}
		req.Header.Set("Content-Type", contentTypeFromVideoType(fileType))
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			writeError(w, r, apperror.DependencyUnavailable(err, "上传视频失败"))
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			writeError(w, r, apperror.DependencyUnavailable(fmt.Errorf("minio status %d", resp.StatusCode), "上传视频失败"))
			return
		}
	}
	url, err := s.uc.ReportPublicVideoUploaded(r.Context(), fileID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{"url": url, "file_id": fileID})
}

func (s *CampusService) handleListCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := s.uc.ListCategories(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	items := make([]map[string]interface{}, 0, len(categories))
	for _, category := range categories {
		items = append(items, categoryToMap(category))
	}
	writeJSON(w, r, map[string]interface{}{"categories": items})
}

func (s *CampusService) handleListPosts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	currentUserID, _ := optionalUserIDFromRequest(r, s.authSecret)
	out, err := s.uc.ListPosts(r.Context(), &biz.ListCampusPostsInput{
		CurrentUserID: currentUserID,
		CategoryCode:  q.Get("category_code"),
		Sort:          q.Get("sort"),
		Keyword:       q.Get("keyword"),
		Page:          int32(queryInt(q.Get("page"), 1)),
		Size:          int32(queryInt(q.Get("size"), 20)),
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	posts := make([]map[string]interface{}, 0, len(out.Posts))
	for _, post := range out.Posts {
		posts = append(posts, postToMap(post))
	}
	writeJSON(w, r, map[string]interface{}{
		"posts": posts,
		"page_stats": map[string]interface{}{
			"total": out.Total,
		},
	})
}

func (s *CampusService) handleListMyPosts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	userID, _ := s.userIDFromRequest(r)
	out, err := s.uc.ListMyPosts(r.Context(), &biz.ListCampusPostsInput{
		CurrentUserID: userID,
		CategoryCode:  q.Get("category_code"),
		Sort:          q.Get("sort"),
		Keyword:       q.Get("keyword"),
		Page:          int32(queryInt(q.Get("page"), 1)),
		Size:          int32(queryInt(q.Get("size"), 20)),
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	posts := make([]map[string]interface{}, 0, len(out.Posts))
	for _, post := range out.Posts {
		posts = append(posts, postToMap(post))
	}
	writeJSON(w, r, map[string]interface{}{
		"posts": posts,
		"page_stats": map[string]interface{}{
			"total": out.Total,
		},
	})
}

func (s *CampusService) handleListMyCollections(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	userID, _ := s.userIDFromRequest(r)
	out, err := s.uc.ListMyCollections(r.Context(), &biz.ListCampusPostsInput{
		CurrentUserID: userID,
		CategoryCode:  q.Get("category_code"),
		Sort:          q.Get("sort"),
		Keyword:       q.Get("keyword"),
		Page:          int32(queryInt(q.Get("page"), 1)),
		Size:          int32(queryInt(q.Get("size"), 20)),
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	posts := make([]map[string]interface{}, 0, len(out.Posts))
	for _, post := range out.Posts {
		posts = append(posts, postToMap(post))
	}
	writeJSON(w, r, map[string]interface{}{
		"posts": posts,
		"page_stats": map[string]interface{}{
			"total": out.Total,
		},
	})
}

type postRequest struct {
	CategoryCode string   `json:"category_code"`
	Title        string   `json:"title"`
	Content      string   `json:"content"`
	Images       []string `json:"images"`
	MediaType    string   `json:"media_type"`
	CoverURL     string   `json:"cover_url"`
	VideoURL     string   `json:"video_url"`
}

func (s *CampusService) handleCreatePost(w http.ResponseWriter, r *http.Request) {
	var req postRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	userID, _ := s.userIDFromRequest(r)
	post, err := s.uc.CreatePost(r.Context(), &biz.CreateCampusPostInput{
		UserID:       userID,
		CategoryCode: req.CategoryCode,
		Title:        req.Title,
		Content:      req.Content,
		Images:       req.Images,
		MediaType:    req.MediaType,
		CoverURL:     req.CoverURL,
		VideoURL:     req.VideoURL,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{"post": postToMap(post)})
}

func (s *CampusService) handleGetPost(w http.ResponseWriter, r *http.Request) {
	postID, ok := pathID(w, r)
	if !ok {
		return
	}
	currentUserID, _ := optionalUserIDFromRequest(r, s.authSecret)
	post, err := s.uc.GetPost(r.Context(), &biz.GetCampusPostInput{PostID: postID, CurrentUserID: currentUserID})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{"post": postToMap(post)})
}

func (s *CampusService) handleDeletePost(w http.ResponseWriter, r *http.Request) {
	postID, ok := pathID(w, r)
	if !ok {
		return
	}
	userID, _ := s.userIDFromRequest(r)
	if err := s.uc.DeletePost(r.Context(), userID, postID); err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{})
}

func (s *CampusService) handleListComments(w http.ResponseWriter, r *http.Request) {
	postID, ok := pathID(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	out, err := s.uc.ListComments(r.Context(), &biz.ListCampusCommentsInput{
		PostID: postID,
		Page:   int32(queryInt(q.Get("page"), 1)),
		Size:   int32(queryInt(q.Get("size"), 20)),
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	comments := make([]map[string]interface{}, 0, len(out.Comments))
	for _, comment := range out.Comments {
		comments = append(comments, commentToMap(comment))
	}
	writeJSON(w, r, map[string]interface{}{
		"comments": comments,
		"page_stats": map[string]interface{}{
			"total": out.Total,
		},
	})
}

type commentRequest struct {
	Content string   `json:"content"`
	Images  []string `json:"images"`
}

func (s *CampusService) handleCreateComment(w http.ResponseWriter, r *http.Request) {
	postID, ok := pathID(w, r)
	if !ok {
		return
	}
	var req commentRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	userID, _ := s.userIDFromRequest(r)
	comment, err := s.uc.CreateComment(r.Context(), &biz.CreateCampusCommentInput{
		UserID:  userID,
		PostID:  postID,
		Content: req.Content,
		Images:  req.Images,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{"comment": commentToMap(comment)})
}

func (s *CampusService) handleLikePost(w http.ResponseWriter, r *http.Request) {
	postID, ok := pathID(w, r)
	if !ok {
		return
	}
	userID, _ := s.userIDFromRequest(r)
	if err := s.uc.LikePost(r.Context(), userID, postID); err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{})
}

func (s *CampusService) handleUnlikePost(w http.ResponseWriter, r *http.Request) {
	postID, ok := pathID(w, r)
	if !ok {
		return
	}
	userID, _ := s.userIDFromRequest(r)
	if err := s.uc.UnlikePost(r.Context(), userID, postID); err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{})
}

func (s *CampusService) handleCollectPost(w http.ResponseWriter, r *http.Request) {
	postID, ok := pathID(w, r)
	if !ok {
		return
	}
	userID, _ := s.userIDFromRequest(r)
	if err := s.uc.CollectPost(r.Context(), userID, postID); err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{})
}

func (s *CampusService) handleUncollectPost(w http.ResponseWriter, r *http.Request) {
	postID, ok := pathID(w, r)
	if !ok {
		return
	}
	userID, _ := s.userIDFromRequest(r)
	if err := s.uc.UncollectPost(r.Context(), userID, postID); err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{})
}

func (s *CampusService) handleDeleteComment(w http.ResponseWriter, r *http.Request) {
	commentID, ok := pathID(w, r)
	if !ok {
		return
	}
	userID, _ := s.userIDFromRequest(r)
	if err := s.uc.DeleteComment(r.Context(), userID, commentID); err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{})
}

type reportRequest struct {
	Reason string `json:"reason"`
	Detail string `json:"detail"`
}

func (s *CampusService) handleReportPost(w http.ResponseWriter, r *http.Request) {
	postID, ok := pathID(w, r)
	if !ok {
		return
	}
	var req reportRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	userID, _ := s.userIDFromRequest(r)
	if err := s.uc.ReportContent(r.Context(), &biz.ReportCampusContentInput{
		UserID:     userID,
		TargetType: "post",
		TargetID:   postID,
		Reason:     req.Reason,
		Detail:     req.Detail,
	}); err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{})
}

func (s *CampusService) handleReportComment(w http.ResponseWriter, r *http.Request) {
	commentID, ok := pathID(w, r)
	if !ok {
		return
	}
	var req reportRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	userID, _ := s.userIDFromRequest(r)
	if err := s.uc.ReportContent(r.Context(), &biz.ReportCampusContentInput{
		UserID:     userID,
		TargetType: "comment",
		TargetID:   commentID,
		Reason:     req.Reason,
		Detail:     req.Detail,
	}); err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{})
}

func (s *CampusService) handleListModerationPosts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	userID, _ := s.userIDFromRequest(r)
	out, err := s.uc.ListModerationPosts(r.Context(), &biz.ListCampusModerationInput{
		UserID: userID,
		Status: int32(queryInt(q.Get("status"), int(biz.CampusAuditStatusPending))),
		Page:   int32(queryInt(q.Get("page"), 1)),
		Size:   int32(queryInt(q.Get("size"), 20)),
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	posts := make([]map[string]interface{}, 0, len(out.Posts))
	for _, post := range out.Posts {
		posts = append(posts, postToMap(post))
	}
	writeJSON(w, r, map[string]interface{}{
		"posts": posts,
		"page_stats": map[string]interface{}{
			"total": out.Total,
		},
	})
}

func (s *CampusService) handleListModerationComments(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	userID, _ := s.userIDFromRequest(r)
	out, err := s.uc.ListModerationComments(r.Context(), &biz.ListCampusModerationInput{
		UserID: userID,
		Status: int32(queryInt(q.Get("status"), int(biz.CampusAuditStatusPending))),
		Page:   int32(queryInt(q.Get("page"), 1)),
		Size:   int32(queryInt(q.Get("size"), 20)),
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	comments := make([]map[string]interface{}, 0, len(out.Comments))
	for _, comment := range out.Comments {
		comments = append(comments, commentToMap(comment))
	}
	writeJSON(w, r, map[string]interface{}{
		"comments": comments,
		"page_stats": map[string]interface{}{
			"total": out.Total,
		},
	})
}

type reviewRequest struct {
	Action string `json:"action"`
	Reason string `json:"reason"`
}

func (s *CampusService) handleReviewPost(w http.ResponseWriter, r *http.Request) {
	postID, ok := pathID(w, r)
	if !ok {
		return
	}
	var req reviewRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	userID, _ := s.userIDFromRequest(r)
	if err := s.uc.ReviewContent(r.Context(), &biz.ReviewCampusContentInput{
		UserID:     userID,
		TargetType: "post",
		TargetID:   postID,
		Action:     req.Action,
		Reason:     req.Reason,
	}); err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{})
}

func (s *CampusService) handleReviewComment(w http.ResponseWriter, r *http.Request) {
	commentID, ok := pathID(w, r)
	if !ok {
		return
	}
	var req reviewRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	userID, _ := s.userIDFromRequest(r)
	if err := s.uc.ReviewContent(r.Context(), &biz.ReviewCampusContentInput{
		UserID:     userID,
		TargetType: "comment",
		TargetID:   commentID,
		Action:     req.Action,
		Reason:     req.Reason,
	}); err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{})
}

func (s *CampusService) authRequired(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_, err := s.userIDFromRequest(r)
		if err != nil {
			writeError(w, r, err)
			return
		}
		next(w, r)
	}
}

func (s *CampusService) userIDFromRequest(r *http.Request) (string, error) {
	if userID, err := claims.GetUserId(r.Context()); err == nil && userID != "" && userID != "0" {
		return userID, nil
	}
	if userID, err := optionalUserIDFromRequest(r, s.authSecret); err == nil && userID != "" && userID != "0" {
		return userID, nil
	}
	return "", apperror.Unauthorized("请先登录")
}

func optionalUserIDFromRequest(r *http.Request, secret string) (string, error) {
	if userID, err := claims.GetUserId(r.Context()); err == nil && userID != "" && userID != "0" {
		return userID, nil
	}
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if header == "" {
		return "", nil
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer"))
	parsed, err := sharedauth.ParseToken(token, secret)
	if err != nil {
		return "", err
	}
	return parsed.UserId, nil
}

func pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil || id <= 0 {
		writeError(w, r, apperror.InvalidArgument("ID 无效"))
		return 0, false
	}
	return id, true
}

func decodeJSON(w http.ResponseWriter, r *http.Request, out interface{}) bool {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return false
	}
	if err := json.NewDecoder(r.Body).Decode(out); err != nil {
		writeError(w, r, apperror.InvalidArgument("请求体不是合法 JSON"))
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, r *http.Request, data interface{}) {
	_ = resp.ResponseEncoder(w, r, data)
}

func writeError(w http.ResponseWriter, r *http.Request, err error) {
	resp.ErrorEncoder(w, r, err)
}

func queryInt(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return n
}

func imageFileType(filename, detected string) string {
	detected = strings.ToLower(strings.TrimSpace(detected))
	switch detected {
	case "image/jpeg":
		return "jpg"
	case "image/png":
		return "png"
	case "image/webp":
		return "webp"
	}
	name := strings.ToLower(strings.TrimSpace(filename))
	switch {
	case strings.HasSuffix(name, ".jpg"), strings.HasSuffix(name, ".jpeg"):
		return "jpg"
	case strings.HasSuffix(name, ".png"):
		return "png"
	case strings.HasSuffix(name, ".webp"):
		return "webp"
	default:
		return ""
	}
}

func videoFileType(filename, detected string) string {
	detected = strings.ToLower(strings.TrimSpace(detected))
	switch detected {
	case "video/mp4":
		return "mp4"
	case "video/quicktime":
		return "mov"
	}
	name := strings.ToLower(strings.TrimSpace(filename))
	switch {
	case strings.HasSuffix(name, ".mp4"):
		return "mp4"
	case strings.HasSuffix(name, ".mov"):
		return "mov"
	default:
		return ""
	}
}

func contentTypeFromImageType(fileType string) string {
	switch fileType {
	case "jpg", "jpeg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "webp":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}

func contentTypeFromVideoType(fileType string) string {
	switch fileType {
	case "mp4":
		return "video/mp4"
	case "mov":
		return "video/quicktime"
	default:
		return "application/octet-stream"
	}
}

func rewritePresignedURLForServerUpload(rawURL string) (string, string) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL, ""
	}
	signedHost := parsed.Host
	internalEndpoint := strings.TrimSpace(os.Getenv("LEHU_INTERNAL_MINIO_ENDPOINT"))
	if internalEndpoint == "" {
		internalEndpoint = "minio:9000"
	}
	parsed.Host = internalEndpoint
	return parsed.String(), signedHost
}

func categoryToMap(category *biz.CampusForumCategory) map[string]interface{} {
	return map[string]interface{}{
		"id":          strconv.FormatInt(category.ID, 10),
		"code":        category.Code,
		"name":        category.Name,
		"description": category.Description,
		"sort_order":  category.SortOrder,
	}
}

func profileToMap(profile *biz.CampusProfile) map[string]interface{} {
	if profile == nil {
		return nil
	}
	return map[string]interface{}{
		"id":            strconv.FormatInt(profile.ID, 10),
		"user_id":       profile.UserID,
		"school_name":   profile.SchoolName,
		"student_no":    profile.StudentNo,
		"real_name":     profile.RealName,
		"class_name":    profile.ClassName,
		"dorm_building": profile.DormBuilding,
		"room_no":       profile.RoomNo,
		"mobile":        profile.Mobile,
		"auth_status":   profile.AuthStatus,
		"created_at":    formatTime(profile.CreatedAt),
		"updated_at":    formatTime(profile.UpdatedAt),
	}
}

func userToMap(user *biz.UserBaseInfo) map[string]interface{} {
	if user == nil {
		return nil
	}
	return map[string]interface{}{
		"id":       user.ID,
		"name":     user.Name,
		"nickname": user.Nickname,
		"avatar":   user.Avatar,
		"mobile":   user.Mobile,
		"email":    user.Email,
	}
}

func timetableCourseToMap(course *biz.CampusTimetableCourse) map[string]interface{} {
	if course == nil {
		return nil
	}
	return map[string]interface{}{
		"id":               strconv.FormatInt(course.ID, 10),
		"term":             course.Term,
		"course_name":      course.CourseName,
		"teacher":          course.Teacher,
		"classroom":        course.Classroom,
		"weekday":          course.Weekday,
		"start_section":    course.StartSection,
		"end_section":      course.EndSection,
		"start_week":       course.StartWeek,
		"end_week":         course.EndWeek,
		"week_parity":      course.WeekParity,
		"source":           course.Source,
		"source_course_id": course.SourceCourseID,
		"created_at":       formatTime(course.CreatedAt),
		"updated_at":       formatTime(course.UpdatedAt),
	}
}

func postToMap(post *biz.CampusForumPost) map[string]interface{} {
	if post == nil {
		return nil
	}
	return map[string]interface{}{
		"id":              strconv.FormatInt(post.ID, 10),
		"category_code":   post.CategoryCode,
		"category_name":   post.CategoryName,
		"author":          authorToMap(post.Author),
		"title":           post.Title,
		"content":         post.Content,
		"images":          post.Images,
		"media_type":      post.MediaType,
		"cover_url":       post.CoverURL,
		"video_url":       post.VideoURL,
		"status":          post.Status,
		"audit_reason":    post.AuditReason,
		"like_count":      post.LikeCount,
		"comment_count":   post.CommentCount,
		"collected_count": post.CollectedCount,
		"is_liked":        post.IsLiked,
		"is_collected":    post.IsCollected,
		"created_at":      formatTime(post.CreatedAt),
		"updated_at":      formatTime(post.UpdatedAt),
	}
}

func commentToMap(comment *biz.CampusForumComment) map[string]interface{} {
	if comment == nil {
		return nil
	}
	return map[string]interface{}{
		"id":           strconv.FormatInt(comment.ID, 10),
		"post_id":      strconv.FormatInt(comment.PostID, 10),
		"author":       authorToMap(comment.Author),
		"content":      comment.Content,
		"images":       comment.Images,
		"status":       comment.Status,
		"audit_reason": comment.AuditReason,
		"like_count":   comment.LikeCount,
		"created_at":   formatTime(comment.CreatedAt),
		"updated_at":   formatTime(comment.UpdatedAt),
	}
}

func authorToMap(author *biz.CampusForumAuthor) map[string]interface{} {
	if author == nil {
		return nil
	}
	return map[string]interface{}{
		"user_id":     author.UserID,
		"name":        author.Name,
		"nickname":    author.Nickname,
		"avatar":      author.Avatar,
		"school_name": author.SchoolName,
		"auth_status": author.AuthStatus,
	}
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.DateTime)
}
