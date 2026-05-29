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

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
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
	log        *log.Helper
}

const (
	campusMaxImageBytes       = 5 << 20
	campusMaxVideoBytes       = 20 << 20
	campusMaxKnowledgeBytes   = 20 << 20
	campusMultipartExtraBytes = 1 << 20
)

func NewCampusService(uc *biz.CampusUsecase, authSecret string, logger log.Logger) *CampusService {
	return &CampusService{uc: uc, authSecret: authSecret, log: log.NewHelper(logger)}
}

func (s *CampusService) RegisterRoutes(srv *khttp.Server) {
	r := srv.Route("/")
	r.POST("/v1/auth/wechat-login", s.wrap(s.handleWechatLogin))
	r.GET("/v1/campus/profile", s.wrap(s.authRequired(s.handleGetProfile)))
	r.PUT("/v1/campus/profile", s.wrap(s.authRequired(s.handleUpdateProfile)))
	r.PUT("/v1/campus/me/avatar", s.wrap(s.authRequired(s.handleUpdateAvatar)))
	r.GET("/v1/campus/timetable", s.wrap(s.authRequired(s.handleListTimetable)))
	r.POST("/v1/campus/timetable/import", s.wrap(s.authRequired(s.handleImportTimetable)))
	r.POST("/v1/campus/analytics/track", s.wrap(s.handleTrackEvent))
	r.POST("/v1/campus/upload/presign", s.wrap(s.authRequired(s.handleUploadPresign)))
	r.POST("/v1/campus/upload/complete", s.wrap(s.authRequired(s.handleUploadComplete)))
	r.POST("/v1/campus/upload/image", s.wrap(s.authRequired(s.handleUploadImage)))
	r.POST("/v1/campus/upload/video", s.wrap(s.authRequired(s.handleUploadVideo)))
	r.GET("/v1/campus/forum/categories", s.wrap(s.handleListCategories))
	r.GET("/v1/campus/forum/posts", s.wrap(s.handleListPosts))
	r.GET("/v1/campus/users/{id}", s.wrap(s.handleGetPublicUserProfile))
	r.GET("/v1/campus/users/{id}/posts", s.wrap(s.handleListPublicUserPosts))
	r.POST("/v1/campus/forum/posts", s.wrap(s.authRequired(s.handleCreatePost)))
	r.GET("/v1/campus/forum/my-posts", s.wrap(s.authRequired(s.handleListMyPosts)))
	r.GET("/v1/campus/forum/my-collections", s.wrap(s.authRequired(s.handleListMyCollections)))
	r.GET("/v1/campus/forum/my-comments", s.wrap(s.authRequired(s.handleListMyComments)))
	r.GET("/v1/campus/forum/posts/{id}", s.wrap(s.handleGetPost))
	r.DELETE("/v1/campus/forum/posts/{id}", s.wrap(s.authRequired(s.handleDeletePost)))
	r.GET("/v1/campus/forum/posts/{id}/comments", s.wrap(s.handleListComments))
	r.POST("/v1/campus/forum/posts/{id}/comments", s.wrap(s.authRequired(s.handleCreateComment)))
	r.POST("/v1/campus/forum/posts/{id}/like", s.wrap(s.authRequired(s.handleLikePost)))
	r.DELETE("/v1/campus/forum/posts/{id}/like", s.wrap(s.authRequired(s.handleUnlikePost)))
	r.POST("/v1/campus/forum/posts/{id}/collection", s.wrap(s.authRequired(s.handleCollectPost)))
	r.DELETE("/v1/campus/forum/posts/{id}/collection", s.wrap(s.authRequired(s.handleUncollectPost)))
	r.POST("/v1/campus/forum/posts/{id}/report", s.wrap(s.authRequired(s.handleReportPost)))
	r.GET("/v1/campus/forum/comments/{id}/replies", s.wrap(s.handleListCommentReplies))
	r.POST("/v1/campus/forum/comments/{id}/like", s.wrap(s.authRequired(s.handleLikeComment)))
	r.DELETE("/v1/campus/forum/comments/{id}/like", s.wrap(s.authRequired(s.handleUnlikeComment)))
	r.DELETE("/v1/campus/forum/comments/{id}", s.wrap(s.authRequired(s.handleDeleteComment)))
	r.POST("/v1/campus/forum/comments/{id}/report", s.wrap(s.authRequired(s.handleReportComment)))
	r.POST("/v1/campus/feedback", s.wrap(s.authRequired(s.handleCreateFeedback)))
	r.GET("/v1/campus/notifications", s.wrap(s.authRequired(s.handleListNotifications)))
	r.GET("/v1/campus/notifications/unread-count", s.wrap(s.authRequired(s.handleUnreadNotificationCount)))
	r.POST("/v1/campus/notifications/read-all", s.wrap(s.authRequired(s.handleMarkAllNotificationsRead)))
	r.POST("/v1/campus/notifications/{id}/read", s.wrap(s.authRequired(s.handleMarkNotificationRead)))
	r.GET("/v1/campus/moderation/posts", s.wrap(s.authRequired(s.handleListModerationPosts)))
	r.GET("/v1/campus/moderation/comments", s.wrap(s.authRequired(s.handleListModerationComments)))
	r.POST("/v1/campus/moderation/posts/{id}/review", s.wrap(s.authRequired(s.handleReviewPost)))
	r.POST("/v1/campus/moderation/comments/{id}/review", s.wrap(s.authRequired(s.handleReviewComment)))
	r.GET("/v1/campus/admin/summary", s.wrap(s.authRequired(s.handleAdminSummary)))
	r.GET("/v1/campus/admin/settings/audit", s.wrap(s.authRequired(s.handleAdminGetAuditSettings)))
	r.PUT("/v1/campus/admin/settings/audit", s.wrap(s.authRequired(s.handleAdminUpdateAuditSettings)))
	r.POST("/v1/campus/admin/stats/reconcile", s.wrap(s.authRequired(s.handleAdminReconcileStats)))
	r.GET("/v1/campus/admin/posts", s.wrap(s.authRequired(s.handleAdminListPosts)))
	r.POST("/v1/campus/admin/posts", s.wrap(s.authRequired(s.handleAdminCreatePost)))
	r.POST("/v1/campus/admin/posts/batch", s.wrap(s.authRequired(s.handleAdminBatchPosts)))
	r.PUT("/v1/campus/admin/posts/{id}", s.wrap(s.authRequired(s.handleAdminUpdatePost)))
	r.DELETE("/v1/campus/admin/posts/{id}", s.wrap(s.authRequired(s.handleAdminDeletePost)))
	r.GET("/v1/campus/admin/comments", s.wrap(s.authRequired(s.handleAdminListComments)))
	r.DELETE("/v1/campus/admin/comments/{id}", s.wrap(s.authRequired(s.handleAdminDeleteComment)))
	r.GET("/v1/campus/admin/ai-replies/summary", s.wrap(s.authRequired(s.handleAdminAIReplySummary)))
	r.GET("/v1/campus/admin/ai-replies/tasks", s.wrap(s.authRequired(s.handleAdminListAIReplyTasks)))
	r.POST("/v1/campus/admin/ai-replies/tasks/{id}/retry", s.wrap(s.authRequired(s.handleAdminRetryAIReplyTask)))
	r.GET("/v1/campus/admin/knowledge/documents", s.wrap(s.authRequired(s.handleAdminListKnowledgeDocuments)))
	r.POST("/v1/campus/admin/knowledge/documents", s.wrap(s.authRequired(s.handleAdminCreateKnowledgeDocument)))
	r.PUT("/v1/campus/admin/knowledge/documents/{id}", s.wrap(s.authRequired(s.handleAdminUpdateKnowledgeDocument)))
	r.POST("/v1/campus/admin/knowledge/documents/{id}/reindex", s.wrap(s.authRequired(s.handleAdminReindexKnowledgeDocument)))
	r.GET("/v1/campus/admin/knowledge/documents/{id}/chunks", s.wrap(s.authRequired(s.handleAdminListKnowledgeChunks)))
	r.POST("/v1/campus/admin/knowledge/test-query", s.wrap(s.authRequired(s.handleAdminTestKnowledgeQuery)))
	r.GET("/v1/campus/admin/knowledge/query-logs", s.wrap(s.authRequired(s.handleAdminListRAGQueryLogs)))
	r.POST("/v1/campus/admin/knowledge/upload", s.wrap(s.authRequired(s.handleAdminUploadKnowledgeFile)))
	r.GET("/v1/campus/admin/reports", s.wrap(s.authRequired(s.handleAdminListReports)))
	r.POST("/v1/campus/admin/reports/{id}/review", s.wrap(s.authRequired(s.handleAdminReviewReport)))
	r.GET("/v1/campus/admin/feedback", s.wrap(s.authRequired(s.handleAdminListFeedback)))
	r.POST("/v1/campus/admin/feedback/{id}/review", s.wrap(s.authRequired(s.handleAdminReviewFeedback)))
	r.GET("/v1/campus/admin/security", s.wrap(s.authRequired(s.handleAdminSecurityOverview)))
	r.POST("/v1/campus/admin/security/ip-blocks", s.wrap(s.authRequired(s.handleAdminBlockIP)))
	r.DELETE("/v1/campus/admin/security/ip-blocks/{id}", s.wrap(s.authRequired(s.handleAdminUnblockIP)))
	r.GET("/v1/campus/admin/users", s.wrap(s.authRequired(s.handleAdminListUsers)))
	r.PUT("/v1/campus/admin/users/{id}/role", s.wrap(s.authRequired(s.handleAdminUpdateUserRole)))
	r.POST("/v1/campus/admin/notifications", s.wrap(s.authRequired(s.handleAdminCreateNotification)))
}

func (s *CampusService) wrap(handler http.HandlerFunc) khttp.HandlerFunc {
	return func(ctx khttp.Context) error {
		s.secure(handler)(ctx.Response(), ctx.Request())
		return nil
	}
}

func (s *CampusService) secure(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestID := r.Header.Get("X-Request-ID")
		if strings.TrimSpace(requestID) == "" {
			requestID = fmt.Sprintf("%d", time.Now().UnixNano())
			r.Header.Set("X-Request-ID", requestID)
		}
		w.Header().Set("X-Request-ID", requestID)
		rw := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		ip := clientIP(r)
		userID, _ := optionalUserIDFromRequest(r, s.authSecret)
		category := campusRequestCategory(r)
		blocked, allowed, err := s.uc.CheckCampusRequest(r.Context(), &biz.CampusRateLimitInput{
			UserID:   userID,
			IP:       ip,
			Method:   r.Method,
			Path:     r.URL.Path,
			Category: category,
		})
		if err != nil {
			writeError(rw, r, err)
		} else if blocked {
			writeError(rw, r, apperror.Forbidden("当前网络访问异常，已被暂时限制"))
		} else if !allowed {
			writeError(rw, r, apperror.TooManyRequests("操作太频繁，请稍后再试"))
		} else {
			next(rw, r)
		}
		statusCode := int32(rw.statusCode)
		errorText := ""
		if rw.statusCode >= http.StatusBadRequest {
			errorText = http.StatusText(rw.statusCode)
		}
		s.uc.RecordAccessLog(r.Context(), &biz.CampusAccessLogInput{
			UserID:      userID,
			IP:          ip,
			Method:      r.Method,
			Path:        r.URL.Path,
			StatusCode:  statusCode,
			DurationMs:  time.Since(start).Milliseconds(),
			UserAgent:   r.UserAgent(),
			RateLimited: !allowed && !blocked,
			Blocked:     blocked,
		})
		duration := time.Since(start)
		s.log.WithContext(r.Context()).Infow(
			"request_id", requestID,
			"trace_id", tracing.TraceID()(r.Context()),
			"span_id", tracing.SpanID()(r.Context()),
			"user_id", userID,
			"ip", ip,
			"method", r.Method,
			"path", r.URL.Path,
			"status", statusCode,
			"duration_ms", duration.Milliseconds(),
			"error", errorText,
			"rate_limited", !allowed && !blocked,
			"blocked", blocked,
		)
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

type avatarRequest struct {
	Avatar string `json:"avatar"`
}

type trackEventRequest struct {
	EventType  string            `json:"event_type"`
	Page       string            `json:"page"`
	TargetType string            `json:"target_type"`
	TargetID   int64             `json:"target_id"`
	Channel    string            `json:"channel"`
	Extra      map[string]string `json:"extra"`
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

func (s *CampusService) handleUpdateAvatar(w http.ResponseWriter, r *http.Request) {
	var req avatarRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	userID, _ := s.userIDFromRequest(r)
	user, err := s.uc.UpdateAvatar(r.Context(), userID, req.Avatar)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{"user": userToMap(user)})
}

func (s *CampusService) handleTrackEvent(w http.ResponseWriter, r *http.Request) {
	var req trackEventRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	userID, _ := optionalUserIDFromRequest(r, s.authSecret)
	if err := s.uc.TrackEvent(r.Context(), &biz.TrackCampusEventInput{
		UserID:     userID,
		EventType:  req.EventType,
		Page:       req.Page,
		TargetType: req.TargetType,
		TargetID:   req.TargetID,
		Channel:    req.Channel,
		Extra:      req.Extra,
		UserAgent:  r.UserAgent(),
		IP:         clientIP(r),
	}); err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{})
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
	r.Body = http.MaxBytesReader(w, r.Body, campusMaxImageBytes+campusMultipartExtraBytes)
	if err := r.ParseMultipartForm(campusMaxImageBytes); err != nil {
		writeError(w, r, apperror.InvalidArgument("图片上传请求无效"))
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, r, apperror.InvalidArgument("请选择图片"))
		return
	}
	defer file.Close()
	if header.Size > campusMaxImageBytes {
		writeError(w, r, apperror.InvalidArgument("图片不能超过 5MB"))
		return
	}
	data, err := io.ReadAll(io.LimitReader(file, campusMaxImageBytes+1))
	if err != nil {
		writeError(w, r, apperror.Internal(err, "读取图片失败"))
		return
	}
	if len(data) > campusMaxImageBytes {
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

type uploadPresignRequest struct {
	MediaType string `json:"media_type"`
	Hash      string `json:"hash"`
	FileType  string `json:"file_type"`
	Filename  string `json:"filename"`
	Size      int64  `json:"size"`
}

type uploadCompleteRequest struct {
	MediaType string `json:"media_type"`
	FileID    string `json:"file_id"`
}

func (s *CampusService) handleUploadPresign(w http.ResponseWriter, r *http.Request) {
	var req uploadPresignRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	out, err := s.uc.PreSignCampusUpload(r.Context(), &biz.CampusUploadPresignInput{
		MediaType: req.MediaType,
		Hash:      req.Hash,
		FileType:  req.FileType,
		Filename:  req.Filename,
		Size:      req.Size,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{
		"file_id":    out.FileID,
		"upload_url": out.UploadURL,
		"method":     out.Method,
		"headers":    out.Headers,
		"expires_in": out.ExpiresIn,
	})
}

func (s *CampusService) handleUploadComplete(w http.ResponseWriter, r *http.Request) {
	var req uploadCompleteRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	out, err := s.uc.CompleteCampusUpload(r.Context(), &biz.CampusUploadCompleteInput{
		MediaType: req.MediaType,
		FileID:    req.FileID,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{
		"file_id":     out.FileID,
		"url":         out.URL,
		"object_name": out.ObjectName,
	})
}

func (s *CampusService) handleUploadVideo(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, campusMaxVideoBytes+campusMultipartExtraBytes)
	if err := r.ParseMultipartForm(campusMaxVideoBytes); err != nil {
		writeError(w, r, apperror.InvalidArgument("视频上传请求无效"))
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, r, apperror.InvalidArgument("请选择视频"))
		return
	}
	defer file.Close()
	if header.Size > campusMaxVideoBytes {
		writeError(w, r, apperror.InvalidArgument("视频不能超过 20MB"))
		return
	}
	data, err := io.ReadAll(io.LimitReader(file, campusMaxVideoBytes+1))
	if err != nil {
		writeError(w, r, apperror.Internal(err, "读取视频失败"))
		return
	}
	if len(data) > campusMaxVideoBytes {
		writeError(w, r, apperror.InvalidArgument("视频不能超过 20MB"))
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

func (s *CampusService) handleAdminUploadKnowledgeFile(w http.ResponseWriter, r *http.Request) {
	userID, _ := s.userIDFromRequest(r)
	if _, err := s.uc.AdminListKnowledgeDocuments(r.Context(), &biz.ListCampusKnowledgeDocumentsInput{UserID: userID, Page: 1, Size: 1}); err != nil {
		writeError(w, r, err)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, campusMaxKnowledgeBytes+campusMultipartExtraBytes)
	if err := r.ParseMultipartForm(campusMaxKnowledgeBytes); err != nil {
		writeError(w, r, apperror.InvalidArgument("知识库文档上传请求无效"))
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, r, apperror.InvalidArgument("请选择知识库文档"))
		return
	}
	defer file.Close()
	if header.Size > campusMaxKnowledgeBytes {
		writeError(w, r, apperror.InvalidArgument("知识库文档不能超过 20MB"))
		return
	}
	data, err := io.ReadAll(io.LimitReader(file, campusMaxKnowledgeBytes+1))
	if err != nil {
		writeError(w, r, apperror.Internal(err, "读取知识库文档失败"))
		return
	}
	if len(data) > campusMaxKnowledgeBytes {
		writeError(w, r, apperror.InvalidArgument("知识库文档不能超过 20MB"))
		return
	}
	fileType := knowledgeFileType(header.Filename)
	if fileType == "" {
		writeError(w, r, apperror.InvalidArgument("仅支持 PDF、DOCX、TXT、MD 文档"))
		return
	}
	sum := md5.Sum(data)
	fileID, putURL, err := s.uc.PreSignPublicKnowledgeFile(r.Context(), fmt.Sprintf("%x", sum), fileType, header.Filename, int64(len(data)))
	if err != nil {
		writeError(w, r, err)
		return
	}
	if putURL != "" {
		uploadURL, signedHost := rewritePresignedURLForServerUpload(putURL)
		req, err := http.NewRequestWithContext(r.Context(), http.MethodPut, uploadURL, bytes.NewReader(data))
		if err != nil {
			writeError(w, r, apperror.Internal(err, "创建知识库文档上传请求失败"))
			return
		}
		if signedHost != "" {
			req.Host = signedHost
		}
		req.Header.Set("Content-Type", contentTypeFromKnowledgeType(fileType))
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			writeError(w, r, apperror.DependencyUnavailable(err, "上传知识库文档失败"))
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			writeError(w, r, apperror.DependencyUnavailable(fmt.Errorf("minio status %d", resp.StatusCode), "上传知识库文档失败"))
			return
		}
	}
	url, err := s.uc.ReportPublicKnowledgeFileUploaded(r.Context(), fileID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{"url": url, "file_id": fileID, "file_type": fileType, "filename": header.Filename})
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
		PostType:      q.Get("post_type"),
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

func (s *CampusService) handleGetPublicUserProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := pathStringID(w, r)
	if !ok {
		return
	}
	profile, err := s.uc.GetPublicCampusUserProfile(r.Context(), userID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{"user": publicUserProfileToMap(profile)})
}

func (s *CampusService) handleListPublicUserPosts(w http.ResponseWriter, r *http.Request) {
	userID, ok := pathStringID(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	currentUserID, _ := optionalUserIDFromRequest(r, s.authSecret)
	out, err := s.uc.ListPublicUserPosts(r.Context(), &biz.ListCampusPostsInput{
		CurrentUserID: currentUserID,
		AuthorID:      userID,
		PostType:      q.Get("post_type"),
		Sort:          q.Get("sort"),
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
		"posts":      posts,
		"page_stats": map[string]interface{}{"total": out.Total},
	})
}

func (s *CampusService) handleListMyPosts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	userID, _ := s.userIDFromRequest(r)
	out, err := s.uc.ListMyPosts(r.Context(), &biz.ListCampusPostsInput{
		CurrentUserID: userID,
		CategoryCode:  q.Get("category_code"),
		PostType:      q.Get("post_type"),
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
		PostType:      q.Get("post_type"),
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
	CategoryCode string            `json:"category_code"`
	Title        string            `json:"title"`
	Content      string            `json:"content"`
	Images       []string          `json:"images"`
	MediaType    string            `json:"media_type"`
	PostType     string            `json:"post_type"`
	Extra        map[string]string `json:"extra"`
	CoverURL     string            `json:"cover_url"`
	VideoURL     string            `json:"video_url"`
	IsOfficial   bool              `json:"is_official"`
	IsFeatured   bool              `json:"is_featured"`
	IsPinned     bool              `json:"is_pinned"`
	SortWeight   int32             `json:"sort_weight"`
}

type batchPostsRequest struct {
	IDs        []int64 `json:"ids"`
	Action     string  `json:"action"`
	SortWeight int32   `json:"sort_weight"`
}

type feedbackRequest struct {
	FeedbackType string   `json:"feedback_type"`
	Content      string   `json:"content"`
	Contact      string   `json:"contact"`
	Images       []string `json:"images"`
}

type reviewFeedbackRequest struct {
	Status       int32  `json:"status"`
	OperatorNote string `json:"operator_note"`
}

type adminNotificationRequest struct {
	Title      string            `json:"title"`
	Content    string            `json:"content"`
	LinkPage   string            `json:"link_page"`
	LinkParams map[string]string `json:"link_params"`
	Audience   string            `json:"audience"`
}

type knowledgeDocumentRequest struct {
	Title       string `json:"title"`
	Source      string `json:"source"`
	Category    string `json:"category"`
	ContentType string `json:"content_type"`
	FileURL     string `json:"file_url"`
	FileID      string `json:"file_id"`
	FileType    string `json:"file_type"`
	RawContent  string `json:"raw_content"`
	Status      string `json:"status"`
	EffectiveAt string `json:"effective_at"`
	ExpiredAt   string `json:"expired_at"`
}

type knowledgeTestQueryRequest struct {
	Query string `json:"query"`
	TopK  int32  `json:"top_k"`
}

type blockIPRequest struct {
	IP     string `json:"ip"`
	Reason string `json:"reason"`
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
		PostType:     req.PostType,
		Extra:        req.Extra,
		CoverURL:     req.CoverURL,
		VideoURL:     req.VideoURL,
		IsOfficial:   req.IsOfficial,
		IsFeatured:   req.IsFeatured,
		IsPinned:     req.IsPinned,
		SortWeight:   req.SortWeight,
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
	currentUserID, _ := optionalUserIDFromRequest(r, s.authSecret)
	out, err := s.uc.ListComments(r.Context(), &biz.ListCampusCommentsInput{
		PostID:        postID,
		CurrentUserID: currentUserID,
		Page:          int32(queryInt(q.Get("page"), 1)),
		Size:          int32(queryInt(q.Get("size"), 20)),
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

func (s *CampusService) handleListCommentReplies(w http.ResponseWriter, r *http.Request) {
	commentID, ok := pathID(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	currentUserID, _ := optionalUserIDFromRequest(r, s.authSecret)
	out, err := s.uc.ListCommentReplies(r.Context(), &biz.ListCampusCommentsInput{
		CommentID:     commentID,
		CurrentUserID: currentUserID,
		Page:          int32(queryInt(q.Get("page"), 1)),
		Size:          int32(queryInt(q.Get("size"), 20)),
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	replies := make([]map[string]interface{}, 0, len(out.Comments))
	for _, comment := range out.Comments {
		replies = append(replies, commentToMap(comment))
	}
	writeJSON(w, r, map[string]interface{}{
		"comments": replies,
		"page_stats": map[string]interface{}{
			"total": out.Total,
		},
	})
}

func (s *CampusService) handleListMyComments(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	userID, _ := s.userIDFromRequest(r)
	out, err := s.uc.ListMyComments(r.Context(), &biz.ListCampusCommentsInput{
		UserID: userID,
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
	Content          string   `json:"content"`
	ParentID         int64    `json:"parent_id"`
	ReplyToCommentID int64    `json:"reply_to_comment_id"`
	Images           []string `json:"images"`
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
		UserID:           userID,
		PostID:           postID,
		ParentID:         req.ParentID,
		ReplyToCommentID: req.ReplyToCommentID,
		Content:          req.Content,
		Images:           req.Images,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{"comment": commentToMap(comment)})
}

func (s *CampusService) handleLikeComment(w http.ResponseWriter, r *http.Request) {
	commentID, ok := pathID(w, r)
	if !ok {
		return
	}
	userID, _ := s.userIDFromRequest(r)
	if err := s.uc.LikeComment(r.Context(), userID, commentID); err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{})
}

func (s *CampusService) handleUnlikeComment(w http.ResponseWriter, r *http.Request) {
	commentID, ok := pathID(w, r)
	if !ok {
		return
	}
	userID, _ := s.userIDFromRequest(r)
	if err := s.uc.UnlikeComment(r.Context(), userID, commentID); err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{})
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

func (s *CampusService) handleListNotifications(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	userID, _ := s.userIDFromRequest(r)
	out, err := s.uc.ListNotifications(r.Context(), &biz.ListCampusNotificationsInput{
		UserID: userID,
		Type:   q.Get("type"),
		Page:   int32(queryInt(q.Get("page"), 1)),
		Size:   int32(queryInt(q.Get("size"), 20)),
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	items := make([]map[string]interface{}, 0, len(out.Notifications))
	for _, item := range out.Notifications {
		items = append(items, notificationToMap(item))
	}
	writeJSON(w, r, map[string]interface{}{
		"notifications": items,
		"page_stats":    map[string]interface{}{"total": out.Total},
	})
}

func (s *CampusService) handleUnreadNotificationCount(w http.ResponseWriter, r *http.Request) {
	userID, _ := s.userIDFromRequest(r)
	count, err := s.uc.CountUnreadNotifications(r.Context(), userID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{
		"total":       count.Total,
		"reply":       count.Reply,
		"interaction": count.Interaction,
		"system":      count.System,
	})
}

func (s *CampusService) handleMarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	notificationID, ok := pathID(w, r)
	if !ok {
		return
	}
	userID, _ := s.userIDFromRequest(r)
	if err := s.uc.MarkNotificationRead(r.Context(), userID, notificationID); err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{})
}

func (s *CampusService) handleMarkAllNotificationsRead(w http.ResponseWriter, r *http.Request) {
	userID, _ := s.userIDFromRequest(r)
	if err := s.uc.MarkAllNotificationsRead(r.Context(), userID); err != nil {
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

func (s *CampusService) handleCreateFeedback(w http.ResponseWriter, r *http.Request) {
	var req feedbackRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	userID, _ := s.userIDFromRequest(r)
	feedback, err := s.uc.CreateFeedback(r.Context(), &biz.CreateCampusFeedbackInput{
		UserID:       userID,
		FeedbackType: req.FeedbackType,
		Content:      req.Content,
		Contact:      req.Contact,
		Images:       req.Images,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{"feedback": feedbackToMap(feedback)})
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

func (s *CampusService) handleAdminSummary(w http.ResponseWriter, r *http.Request) {
	userID, _ := s.userIDFromRequest(r)
	summary, err := s.uc.AdminSummary(r.Context(), userID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{"summary": adminSummaryToMap(summary)})
}

type auditSettingsRequest struct {
	PostAuditMode string `json:"post_audit_mode"`
}

func (s *CampusService) handleAdminGetAuditSettings(w http.ResponseWriter, r *http.Request) {
	userID, _ := s.userIDFromRequest(r)
	settings, err := s.uc.AdminGetAuditSettings(r.Context(), &biz.GetCampusAuditSettingsInput{UserID: userID})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{"settings": auditSettingsToMap(settings)})
}

func (s *CampusService) handleAdminUpdateAuditSettings(w http.ResponseWriter, r *http.Request) {
	var req auditSettingsRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	userID, _ := s.userIDFromRequest(r)
	settings, err := s.uc.AdminUpdateAuditSettings(r.Context(), &biz.UpdateCampusAuditSettingsInput{
		UserID:        userID,
		PostAuditMode: req.PostAuditMode,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{"settings": auditSettingsToMap(settings)})
}

func (s *CampusService) handleAdminReconcileStats(w http.ResponseWriter, r *http.Request) {
	userID, _ := s.userIDFromRequest(r)
	result, err := s.uc.AdminReconcileCampusStats(r.Context(), userID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{"result": statsReconcileResultToMap(result)})
}

func (s *CampusService) handleAdminListPosts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	userID, _ := s.userIDFromRequest(r)
	out, err := s.uc.AdminListPosts(r.Context(), &biz.ListCampusAdminPostsInput{
		UserID:       userID,
		CategoryCode: q.Get("category_code"),
		PostType:     q.Get("post_type"),
		OpsFilter:    q.Get("ops_filter"),
		Keyword:      q.Get("keyword"),
		Status:       int32(queryInt(q.Get("status"), -1)),
		Sort:         q.Get("sort"),
		Page:         int32(queryInt(q.Get("page"), 1)),
		Size:         int32(queryInt(q.Get("size"), 20)),
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
		"posts":      posts,
		"page_stats": map[string]interface{}{"total": out.Total},
	})
}

func (s *CampusService) handleAdminCreatePost(w http.ResponseWriter, r *http.Request) {
	var req postRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	userID, _ := s.userIDFromRequest(r)
	post, err := s.uc.AdminCreatePost(r.Context(), &biz.CreateCampusPostInput{
		UserID:       userID,
		CategoryCode: req.CategoryCode,
		Title:        req.Title,
		Content:      req.Content,
		Images:       req.Images,
		MediaType:    req.MediaType,
		PostType:     req.PostType,
		Extra:        req.Extra,
		CoverURL:     req.CoverURL,
		VideoURL:     req.VideoURL,
		IsOfficial:   req.IsOfficial,
		IsFeatured:   req.IsFeatured,
		IsPinned:     req.IsPinned,
		SortWeight:   req.SortWeight,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{"post": postToMap(post)})
}

func (s *CampusService) handleAdminBatchPosts(w http.ResponseWriter, r *http.Request) {
	var req batchPostsRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	userID, _ := s.userIDFromRequest(r)
	out, err := s.uc.AdminBatchPosts(r.Context(), &biz.BatchCampusAdminPostsInput{
		UserID:     userID,
		PostIDs:    req.IDs,
		Action:     req.Action,
		SortWeight: req.SortWeight,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{"updated_count": out.UpdatedCount})
}

func (s *CampusService) handleAdminUpdatePost(w http.ResponseWriter, r *http.Request) {
	postID, ok := pathID(w, r)
	if !ok {
		return
	}
	var req struct {
		postRequest
		Status      int32  `json:"status"`
		AuditReason string `json:"audit_reason"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	userID, _ := s.userIDFromRequest(r)
	post, err := s.uc.AdminUpdatePost(r.Context(), &biz.UpdateCampusAdminPostInput{
		UserID:       userID,
		PostID:       postID,
		CategoryCode: req.CategoryCode,
		Title:        req.Title,
		Content:      req.Content,
		Images:       req.Images,
		MediaType:    req.MediaType,
		PostType:     req.PostType,
		Extra:        req.Extra,
		CoverURL:     req.CoverURL,
		VideoURL:     req.VideoURL,
		Status:       req.Status,
		AuditReason:  req.AuditReason,
		IsOfficial:   req.IsOfficial,
		IsFeatured:   req.IsFeatured,
		IsPinned:     req.IsPinned,
		SortWeight:   req.SortWeight,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{"post": postToMap(post)})
}

func (s *CampusService) handleAdminDeletePost(w http.ResponseWriter, r *http.Request) {
	postID, ok := pathID(w, r)
	if !ok {
		return
	}
	userID, _ := s.userIDFromRequest(r)
	if err := s.uc.AdminDeletePost(r.Context(), userID, postID); err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{})
}

func (s *CampusService) handleAdminListComments(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	userID, _ := s.userIDFromRequest(r)
	out, err := s.uc.AdminListComments(r.Context(), &biz.ListCampusAdminCommentsInput{
		UserID: userID,
		Status: int32(queryInt(q.Get("status"), -1)),
		PostID: int64(queryInt(q.Get("post_id"), 0)),
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
		"comments":   comments,
		"page_stats": map[string]interface{}{"total": out.Total},
	})
}

func (s *CampusService) handleAdminDeleteComment(w http.ResponseWriter, r *http.Request) {
	commentID, ok := pathID(w, r)
	if !ok {
		return
	}
	userID, _ := s.userIDFromRequest(r)
	if err := s.uc.AdminDeleteComment(r.Context(), userID, commentID); err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{})
}

func (s *CampusService) handleAdminAIReplySummary(w http.ResponseWriter, r *http.Request) {
	userID, _ := s.userIDFromRequest(r)
	overview, err := s.uc.AdminAIReplyOverview(r.Context(), userID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{"summary": aiReplyOverviewToMap(overview)})
}

func (s *CampusService) handleAdminListAIReplyTasks(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	userID, _ := s.userIDFromRequest(r)
	out, err := s.uc.AdminListAIReplyTasks(r.Context(), &biz.ListCampusAIReplyTasksInput{
		UserID: userID,
		Status: q.Get("status"),
		Page:   int32(queryInt(q.Get("page"), 1)),
		Size:   int32(queryInt(q.Get("size"), 20)),
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	tasks := make([]map[string]interface{}, 0, len(out.Tasks))
	for _, task := range out.Tasks {
		tasks = append(tasks, aiReplyTaskToMap(task))
	}
	writeJSON(w, r, map[string]interface{}{
		"tasks":      tasks,
		"page_stats": map[string]interface{}{"total": out.Total},
	})
}

func (s *CampusService) handleAdminRetryAIReplyTask(w http.ResponseWriter, r *http.Request) {
	taskID, ok := pathID(w, r)
	if !ok {
		return
	}
	userID, _ := s.userIDFromRequest(r)
	if err := s.uc.AdminRetryAIReplyTask(r.Context(), &biz.RetryCampusAIReplyTaskInput{
		UserID: userID,
		TaskID: taskID,
	}); err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{})
}

func (s *CampusService) handleAdminListKnowledgeDocuments(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	userID, _ := s.userIDFromRequest(r)
	out, err := s.uc.AdminListKnowledgeDocuments(r.Context(), &biz.ListCampusKnowledgeDocumentsInput{
		UserID:   userID,
		Keyword:  q.Get("keyword"),
		Category: q.Get("category"),
		Status:   q.Get("status"),
		Page:     int32(queryInt(q.Get("page"), 1)),
		Size:     int32(queryInt(q.Get("size"), 20)),
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	items := make([]map[string]interface{}, 0, len(out.Documents))
	for _, doc := range out.Documents {
		items = append(items, knowledgeDocumentToMap(doc))
	}
	writeJSON(w, r, map[string]interface{}{"documents": items, "page_stats": map[string]interface{}{"total": out.Total}})
}

func (s *CampusService) handleAdminCreateKnowledgeDocument(w http.ResponseWriter, r *http.Request) {
	var req knowledgeDocumentRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	userID, _ := s.userIDFromRequest(r)
	doc, err := s.uc.AdminCreateKnowledgeDocument(r.Context(), &biz.CreateCampusKnowledgeDocumentInput{
		UserID:      userID,
		Title:       req.Title,
		Source:      req.Source,
		Category:    req.Category,
		ContentType: req.ContentType,
		FileURL:     req.FileURL,
		FileID:      req.FileID,
		FileType:    req.FileType,
		RawContent:  req.RawContent,
		Status:      req.Status,
		EffectiveAt: parseOptionalRequestTime(req.EffectiveAt),
		ExpiredAt:   parseOptionalRequestTime(req.ExpiredAt),
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{"document": knowledgeDocumentToMap(doc)})
}

func (s *CampusService) handleAdminUpdateKnowledgeDocument(w http.ResponseWriter, r *http.Request) {
	documentID, ok := pathID(w, r)
	if !ok {
		return
	}
	var req knowledgeDocumentRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	userID, _ := s.userIDFromRequest(r)
	doc, err := s.uc.AdminUpdateKnowledgeDocument(r.Context(), &biz.UpdateCampusKnowledgeDocumentInput{
		UserID:      userID,
		DocumentID:  documentID,
		Title:       req.Title,
		Source:      req.Source,
		Category:    req.Category,
		Status:      req.Status,
		EffectiveAt: parseOptionalRequestTime(req.EffectiveAt),
		ExpiredAt:   parseOptionalRequestTime(req.ExpiredAt),
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{"document": knowledgeDocumentToMap(doc)})
}

func (s *CampusService) handleAdminReindexKnowledgeDocument(w http.ResponseWriter, r *http.Request) {
	documentID, ok := pathID(w, r)
	if !ok {
		return
	}
	userID, _ := s.userIDFromRequest(r)
	doc, err := s.uc.AdminReindexKnowledgeDocument(r.Context(), userID, documentID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{"document": knowledgeDocumentToMap(doc)})
}

func (s *CampusService) handleAdminListKnowledgeChunks(w http.ResponseWriter, r *http.Request) {
	documentID, ok := pathID(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	userID, _ := s.userIDFromRequest(r)
	out, err := s.uc.AdminListKnowledgeChunks(r.Context(), &biz.ListCampusKnowledgeChunksInput{
		UserID:     userID,
		DocumentID: documentID,
		Page:       int32(queryInt(q.Get("page"), 1)),
		Size:       int32(queryInt(q.Get("size"), 20)),
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	items := make([]map[string]interface{}, 0, len(out.Chunks))
	for _, chunk := range out.Chunks {
		items = append(items, knowledgeChunkToMap(chunk))
	}
	writeJSON(w, r, map[string]interface{}{"chunks": items, "page_stats": map[string]interface{}{"total": out.Total}})
}

func (s *CampusService) handleAdminTestKnowledgeQuery(w http.ResponseWriter, r *http.Request) {
	var req knowledgeTestQueryRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	userID, _ := s.userIDFromRequest(r)
	out, err := s.uc.AdminTestKnowledgeQuery(r.Context(), &biz.TestCampusKnowledgeQueryInput{
		UserID: userID,
		Query:  req.Query,
		TopK:   req.TopK,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{"result": ragQueryResponseToMap(out)})
}

func (s *CampusService) handleAdminListRAGQueryLogs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	userID, _ := s.userIDFromRequest(r)
	out, err := s.uc.AdminListRAGQueryLogs(r.Context(), &biz.ListCampusRAGQueryLogsInput{
		UserID: userID,
		Page:   int32(queryInt(q.Get("page"), 1)),
		Size:   int32(queryInt(q.Get("size"), 20)),
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	items := make([]map[string]interface{}, 0, len(out.Logs))
	for _, item := range out.Logs {
		items = append(items, ragQueryLogToMap(item))
	}
	writeJSON(w, r, map[string]interface{}{"logs": items, "page_stats": map[string]interface{}{"total": out.Total}})
}

func (s *CampusService) handleAdminListReports(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	userID, _ := s.userIDFromRequest(r)
	out, err := s.uc.AdminListReports(r.Context(), &biz.ListCampusReportsInput{
		UserID: userID,
		Status: int32(queryInt(q.Get("status"), -1)),
		Page:   int32(queryInt(q.Get("page"), 1)),
		Size:   int32(queryInt(q.Get("size"), 20)),
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	reports := make([]map[string]interface{}, 0, len(out.Reports))
	for _, report := range out.Reports {
		reports = append(reports, reportToMap(report))
	}
	writeJSON(w, r, map[string]interface{}{
		"reports":    reports,
		"page_stats": map[string]interface{}{"total": out.Total},
	})
}

func (s *CampusService) handleAdminReviewReport(w http.ResponseWriter, r *http.Request) {
	reportID, ok := pathID(w, r)
	if !ok {
		return
	}
	var req reviewRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	userID, _ := s.userIDFromRequest(r)
	if err := s.uc.AdminReviewReport(r.Context(), &biz.ReviewCampusReportInput{
		UserID:   userID,
		ReportID: reportID,
		Action:   req.Action,
		Reason:   req.Reason,
	}); err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{})
}

func (s *CampusService) handleAdminListFeedback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	userID, _ := s.userIDFromRequest(r)
	out, err := s.uc.AdminListFeedback(r.Context(), &biz.ListCampusFeedbackInput{
		UserID: userID,
		Status: int32(queryInt(q.Get("status"), -1)),
		Page:   int32(queryInt(q.Get("page"), 1)),
		Size:   int32(queryInt(q.Get("size"), 20)),
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	items := make([]map[string]interface{}, 0, len(out.Feedbacks))
	for _, feedback := range out.Feedbacks {
		items = append(items, feedbackToMap(feedback))
	}
	writeJSON(w, r, map[string]interface{}{
		"feedbacks":  items,
		"page_stats": map[string]interface{}{"total": out.Total},
	})
}

func (s *CampusService) handleAdminReviewFeedback(w http.ResponseWriter, r *http.Request) {
	feedbackID, ok := pathID(w, r)
	if !ok {
		return
	}
	var req reviewFeedbackRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	userID, _ := s.userIDFromRequest(r)
	if err := s.uc.AdminReviewFeedback(r.Context(), &biz.ReviewCampusFeedbackInput{
		UserID:       userID,
		FeedbackID:   feedbackID,
		Status:       req.Status,
		OperatorNote: req.OperatorNote,
	}); err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{})
}

func (s *CampusService) handleAdminCreateNotification(w http.ResponseWriter, r *http.Request) {
	var req adminNotificationRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	userID, _ := s.userIDFromRequest(r)
	taskID, err := s.uc.AdminCreateSystemNotification(r.Context(), &biz.CreateCampusAdminNotificationInput{
		UserID:     userID,
		Title:      req.Title,
		Content:    req.Content,
		LinkPage:   req.LinkPage,
		LinkParams: req.LinkParams,
		Audience:   req.Audience,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{"queued": true, "task_id": fmt.Sprintf("%d", taskID)})
}

func (s *CampusService) handleAdminSecurityOverview(w http.ResponseWriter, r *http.Request) {
	userID, _ := s.userIDFromRequest(r)
	overview, err := s.uc.AdminSecurityOverview(r.Context(), &biz.ListCampusSecurityInput{UserID: userID})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{"security": securityOverviewToMap(overview)})
}

func (s *CampusService) handleAdminBlockIP(w http.ResponseWriter, r *http.Request) {
	var req blockIPRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	userID, _ := s.userIDFromRequest(r)
	if err := s.uc.AdminBlockIP(r.Context(), &biz.BlockCampusIPInput{
		UserID: userID,
		IP:     req.IP,
		Reason: req.Reason,
	}); err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{})
}

func (s *CampusService) handleAdminUnblockIP(w http.ResponseWriter, r *http.Request) {
	ip := strings.TrimSpace(mux.Vars(r)["id"])
	if ip == "" {
		writeError(w, r, apperror.InvalidArgument("IP 无效"))
		return
	}
	userID, _ := s.userIDFromRequest(r)
	if err := s.uc.AdminUnblockIP(r.Context(), &biz.BlockCampusIPInput{UserID: userID, IP: ip}); err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]interface{}{})
}

func (s *CampusService) handleAdminListUsers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	userID, _ := s.userIDFromRequest(r)
	out, err := s.uc.AdminListUsers(r.Context(), &biz.ListCampusAdminUsersInput{
		UserID:     userID,
		Keyword:    q.Get("keyword"),
		Role:       q.Get("role"),
		AuthStatus: int32(queryInt(q.Get("auth_status"), -1)),
		Page:       int32(queryInt(q.Get("page"), 1)),
		Size:       int32(queryInt(q.Get("size"), 20)),
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	users := make([]map[string]interface{}, 0, len(out.Users))
	for _, user := range out.Users {
		users = append(users, adminUserToMap(user))
	}
	writeJSON(w, r, map[string]interface{}{
		"users":      users,
		"page_stats": map[string]interface{}{"total": out.Total},
	})
}

func (s *CampusService) handleAdminUpdateUserRole(w http.ResponseWriter, r *http.Request) {
	targetUserID, ok := pathStringID(w, r)
	if !ok {
		return
	}
	var req struct {
		Role string `json:"role"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	userID, _ := s.userIDFromRequest(r)
	if err := s.uc.AdminUpdateUserRole(r.Context(), &biz.UpdateCampusUserRoleInput{
		UserID:       userID,
		TargetUserID: targetUserID,
		Role:         req.Role,
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

func clientIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}
	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}
	host := r.RemoteAddr
	if idx := strings.LastIndex(host, ":"); idx > -1 {
		return host[:idx]
	}
	return host
}

func campusRequestCategory(r *http.Request) string {
	path := r.URL.Path
	switch {
	case path == "/v1/auth/wechat-login":
		return "auth"
	case strings.HasPrefix(path, "/v1/campus/admin/"):
		return "admin"
	case strings.HasPrefix(path, "/v1/campus/upload/"):
		return "upload"
	case path == "/v1/campus/feedback":
		return "feedback"
	case r.Method != http.MethodGet:
		return "write"
	default:
		return "read"
	}
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := mux.Vars(r)["id"]
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, r, apperror.InvalidArgument("ID 无效"))
		return 0, false
	}
	return id, true
}

func pathStringID(w http.ResponseWriter, r *http.Request) (string, bool) {
	raw := strings.TrimSpace(mux.Vars(r)["id"])
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, r, apperror.InvalidArgument("ID 无效"))
		return "", false
	}
	return raw, true
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

func knowledgeFileType(filename string) string {
	name := strings.ToLower(strings.TrimSpace(filename))
	switch {
	case strings.HasSuffix(name, ".pdf"):
		return "pdf"
	case strings.HasSuffix(name, ".docx"):
		return "docx"
	case strings.HasSuffix(name, ".txt"):
		return "txt"
	case strings.HasSuffix(name, ".md"), strings.HasSuffix(name, ".markdown"):
		return "md"
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

func contentTypeFromKnowledgeType(fileType string) string {
	switch fileType {
	case "pdf":
		return "application/pdf"
	case "docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case "txt":
		return "text/plain; charset=utf-8"
	case "md":
		return "text/markdown; charset=utf-8"
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
		"id":                strconv.FormatInt(post.ID, 10),
		"category_code":     post.CategoryCode,
		"category_name":     post.CategoryName,
		"author":            authorToMap(post.Author),
		"title":             post.Title,
		"content":           post.Content,
		"images":            post.Images,
		"media_type":        post.MediaType,
		"post_type":         post.PostType,
		"extra":             post.Extra,
		"cover_url":         post.CoverURL,
		"video_url":         post.VideoURL,
		"is_official":       post.IsOfficial,
		"is_featured":       post.IsFeatured,
		"is_pinned":         post.IsPinned,
		"sort_weight":       post.SortWeight,
		"status":            post.Status,
		"audit_reason":      post.AuditReason,
		"ai_audit_status":   post.AIAuditStatus,
		"ai_audit_risk":     post.AIAuditRisk,
		"ai_audit_decision": post.AIAuditDecision,
		"ai_audit_reason":   post.AIAuditReason,
		"ai_audit_error":    post.AIAuditError,
		"like_count":        post.LikeCount,
		"comment_count":     post.CommentCount,
		"collected_count":   post.CollectedCount,
		"is_liked":          post.IsLiked,
		"is_collected":      post.IsCollected,
		"created_at":        formatTime(post.CreatedAt),
		"updated_at":        formatTime(post.UpdatedAt),
	}
}

func auditSettingsToMap(settings *biz.CampusOpsAuditSettings) map[string]interface{} {
	if settings == nil {
		return map[string]interface{}{
			"post_audit_mode": biz.CampusPostAuditModeOff,
			"ai_enabled":      false,
		}
	}
	return map[string]interface{}{
		"post_audit_mode": settings.PostAuditMode,
		"ai_enabled":      settings.AIEnabled,
		"updated_by":      settings.UpdatedBy,
		"updated_at":      formatTime(settings.UpdatedAt),
	}
}

func publicUserProfileToMap(user *biz.CampusPublicUserProfile) map[string]interface{} {
	if user == nil {
		return nil
	}
	stats := user.Stats
	if stats == nil {
		stats = &biz.CampusPublicUserStats{}
	}
	return map[string]interface{}{
		"user_id":     user.UserID,
		"name":        user.Name,
		"nickname":    user.Nickname,
		"avatar":      user.Avatar,
		"school_name": user.SchoolName,
		"auth_status": user.AuthStatus,
		"is_official": user.IsOfficial,
		"bio":         user.Bio,
		"stats": map[string]interface{}{
			"post_count":      stats.PostCount,
			"like_count":      stats.LikeCount,
			"collected_count": stats.CollectedCount,
		},
	}
}

func reportToMap(report *biz.CampusForumReport) map[string]interface{} {
	if report == nil {
		return nil
	}
	return map[string]interface{}{
		"id":          strconv.FormatInt(report.ID, 10),
		"target_type": report.TargetType,
		"target_id":   strconv.FormatInt(report.TargetID, 10),
		"target":      postToMap(report.Target),
		"comment":     commentToMap(report.Comment),
		"reporter":    authorToMap(report.Reporter),
		"reason":      report.Reason,
		"detail":      report.Detail,
		"status":      report.Status,
		"created_at":  formatTime(report.CreatedAt),
		"updated_at":  formatTime(report.UpdatedAt),
	}
}

func feedbackToMap(feedback *biz.CampusFeedback) map[string]interface{} {
	if feedback == nil {
		return nil
	}
	return map[string]interface{}{
		"id":            strconv.FormatInt(feedback.ID, 10),
		"user_id":       feedback.UserID,
		"author":        authorToMap(feedback.Author),
		"feedback_type": feedback.FeedbackType,
		"content":       feedback.Content,
		"contact":       feedback.Contact,
		"images":        feedback.Images,
		"status":        feedback.Status,
		"operator_note": feedback.OperatorNote,
		"created_at":    formatTime(feedback.CreatedAt),
		"updated_at":    formatTime(feedback.UpdatedAt),
	}
}

func notificationToMap(notification *biz.CampusNotification) map[string]interface{} {
	if notification == nil {
		return nil
	}
	readAt := ""
	if notification.ReadAt != nil {
		readAt = formatTime(*notification.ReadAt)
	}
	return map[string]interface{}{
		"id":           strconv.FormatInt(notification.ID, 10),
		"recipient_id": notification.RecipientID,
		"actor_id":     notification.ActorID,
		"actor":        authorToMap(notification.Actor),
		"event_type":   notification.EventType,
		"target_type":  notification.TargetType,
		"target_id":    strconv.FormatInt(notification.TargetID, 10),
		"dedupe_key":   notification.DedupeKey,
		"title":        notification.Title,
		"content":      notification.Content,
		"link_page":    notification.LinkPage,
		"link_params":  notification.LinkParams,
		"is_read":      notification.ReadAt != nil,
		"read_at":      readAt,
		"created_at":   formatTime(notification.CreatedAt),
		"updated_at":   formatTime(notification.UpdatedAt),
	}
}

func securityOverviewToMap(overview *biz.CampusSecurityOverview) map[string]interface{} {
	if overview == nil {
		return nil
	}
	topIPs := make([]map[string]interface{}, 0, len(overview.TopIPs))
	for _, item := range overview.TopIPs {
		topIPs = append(topIPs, map[string]interface{}{
			"ip":            item.IP,
			"request_count": item.RequestCount,
			"error_count":   item.ErrorCount,
			"last_seen":     formatTime(item.LastSeen),
		})
	}
	topPaths := make([]map[string]interface{}, 0, len(overview.TopPaths))
	for _, item := range overview.TopPaths {
		topPaths = append(topPaths, map[string]interface{}{
			"path":          item.Path,
			"request_count": item.RequestCount,
			"error_count":   item.ErrorCount,
		})
	}
	recentLogs := make([]map[string]interface{}, 0, len(overview.RecentAccessLogs))
	for _, item := range overview.RecentAccessLogs {
		recentLogs = append(recentLogs, accessLogToMap(item))
	}
	blockedIPs := make([]map[string]interface{}, 0, len(overview.BlockedIPs))
	for _, item := range overview.BlockedIPs {
		blockedIPs = append(blockedIPs, ipBlockToMap(item))
	}
	return map[string]interface{}{
		"today_requests":     overview.TodayRequests,
		"today_unique_ips":   overview.TodayUniqueIPs,
		"today_rate_limited": overview.TodayRateLimited,
		"today_blocked":      overview.TodayBlocked,
		"today_errors":       overview.TodayErrors,
		"active_blocked_ips": overview.ActiveBlockedIPs,
		"top_ips":            topIPs,
		"top_paths":          topPaths,
		"recent_logs":        recentLogs,
		"blocked_ips":        blockedIPs,
	}
}

func accessLogToMap(log *biz.CampusAccessLog) map[string]interface{} {
	if log == nil {
		return nil
	}
	return map[string]interface{}{
		"id":           strconv.FormatInt(log.ID, 10),
		"user_id":      log.UserID,
		"ip":           log.IP,
		"method":       log.Method,
		"path":         log.Path,
		"status_code":  log.StatusCode,
		"duration_ms":  log.DurationMs,
		"user_agent":   log.UserAgent,
		"rate_limited": log.RateLimited,
		"blocked":      log.Blocked,
		"created_at":   formatTime(log.CreatedAt),
	}
}

func ipBlockToMap(block *biz.CampusIPBlock) map[string]interface{} {
	if block == nil {
		return nil
	}
	return map[string]interface{}{
		"id":         strconv.FormatInt(block.ID, 10),
		"ip":         block.IP,
		"reason":     block.Reason,
		"status":     block.Status,
		"created_by": block.CreatedBy,
		"created_at": formatTime(block.CreatedAt),
		"updated_at": formatTime(block.UpdatedAt),
	}
}

func statsReconcileResultToMap(result *biz.CampusStatsReconcileResult) map[string]interface{} {
	if result == nil {
		return nil
	}
	return map[string]interface{}{
		"checked_at":       formatTime(result.CheckedAt),
		"updated_posts":    result.UpdatedPosts,
		"updated_comments": result.UpdatedComments,
	}
}

func aiReplyOverviewToMap(overview *biz.CampusAIReplyOverview) map[string]interface{} {
	if overview == nil {
		return nil
	}
	recent := make([]map[string]interface{}, 0, len(overview.Recent))
	for _, task := range overview.Recent {
		recent = append(recent, aiReplyTaskToMap(task))
	}
	return map[string]interface{}{
		"enabled":     overview.Enabled,
		"bot_user_id": overview.BotUserID,
		"bot_ready":   overview.BotReady,
		"bot_name":    overview.BotName,
		"bot_avatar":  overview.BotAvatar,
		"model":       overview.Model,
		"base_url":    overview.BaseURL,
		"rag_health":  ragHealthToMap(overview.RAGHealth),
		"daily_limit": overview.DailyLimit,
		"today_used":  overview.TodayUsed,
		"pending":     overview.Pending,
		"processing":  overview.Processing,
		"done":        overview.Done,
		"failed":      overview.Failed,
		"recent":      recent,
	}
}

func ragHealthToMap(health *biz.CampusRAGHealth) map[string]interface{} {
	if health == nil {
		return nil
	}
	return map[string]interface{}{
		"status":       health.Status,
		"qdrant":       health.Qdrant,
		"chunk_count":  health.ChunkCount,
		"failed_count": health.FailedCount,
		"last_error":   health.LastError,
	}
}

func aiReplyTaskToMap(task *biz.CampusAIReplyTask) map[string]interface{} {
	if task == nil {
		return nil
	}
	return map[string]interface{}{
		"id":                 strconv.FormatInt(task.ID, 10),
		"post_id":            strconv.FormatInt(task.PostID, 10),
		"root_comment_id":    strconv.FormatInt(task.RootCommentID, 10),
		"trigger_comment_id": strconv.FormatInt(task.TriggerCommentID, 10),
		"asker_id":           task.AskerID,
		"bot_user_id":        task.BotUserID,
		"prompt":             task.Prompt,
		"status":             task.Status,
		"retry_count":        task.RetryCount,
		"next_retry_at":      formatOptionalTime(task.NextRetryAt),
		"locked_until":       formatOptionalTime(task.LockedUntil),
		"answer_comment_id":  strconv.FormatInt(task.AnswerCommentID, 10),
		"last_error":         task.LastError,
		"created_at":         formatTime(task.CreatedAt),
		"updated_at":         formatTime(task.UpdatedAt),
		"processed_at":       formatOptionalTime(task.ProcessedAt),
	}
}

func knowledgeDocumentToMap(doc *biz.CampusKnowledgeDocument) map[string]interface{} {
	if doc == nil {
		return nil
	}
	return map[string]interface{}{
		"id":            strconv.FormatInt(doc.ID, 10),
		"title":         doc.Title,
		"source":        doc.Source,
		"category":      doc.Category,
		"content_type":  doc.ContentType,
		"file_url":      doc.FileURL,
		"file_id":       doc.FileID,
		"file_type":     doc.FileType,
		"raw_content":   doc.RawContent,
		"status":        doc.Status,
		"parse_status":  doc.ParseStatus,
		"error_message": doc.ErrorMessage,
		"uploaded_by":   doc.UploadedBy,
		"effective_at":  formatOptionalTime(doc.EffectiveAt),
		"expired_at":    formatOptionalTime(doc.ExpiredAt),
		"chunk_count":   doc.ChunkCount,
		"created_at":    formatTime(doc.CreatedAt),
		"updated_at":    formatTime(doc.UpdatedAt),
	}
}

func knowledgeChunkToMap(chunk *biz.CampusKnowledgeChunk) map[string]interface{} {
	if chunk == nil {
		return nil
	}
	return map[string]interface{}{
		"id":               strconv.FormatInt(chunk.ID, 10),
		"document_id":      strconv.FormatInt(chunk.DocumentID, 10),
		"chunk_index":      chunk.ChunkIndex,
		"title":            chunk.Title,
		"content":          chunk.Content,
		"summary":          chunk.Summary,
		"category":         chunk.Category,
		"keywords":         chunk.Keywords,
		"source":           chunk.Source,
		"status":           chunk.Status,
		"qdrant_point_id":  chunk.QdrantPointID,
		"embedding_status": chunk.EmbeddingStatus,
		"score":            chunk.Score,
		"created_at":       formatTime(chunk.CreatedAt),
		"updated_at":       formatTime(chunk.UpdatedAt),
	}
}

func ragQueryResponseToMap(resp *biz.CampusRAGQueryResponse) map[string]interface{} {
	if resp == nil {
		return nil
	}
	chunks := make([]map[string]interface{}, 0, len(resp.Chunks))
	for _, chunk := range resp.Chunks {
		chunks = append(chunks, ragQueryChunkToMap(chunk))
	}
	return map[string]interface{}{
		"need_knowledge": resp.NeedKnowledge,
		"confidence":     resp.Confidence,
		"chunks":         chunks,
	}
}

func ragQueryChunkToMap(chunk *biz.CampusRAGQueryChunk) map[string]interface{} {
	if chunk == nil {
		return nil
	}
	return map[string]interface{}{
		"chunk_id":    chunk.ChunkID,
		"document_id": chunk.DocumentID,
		"title":       chunk.Title,
		"category":    chunk.Category,
		"content":     chunk.Content,
		"source":      chunk.Source,
		"score":       chunk.Score,
	}
}

func ragQueryLogToMap(item *biz.CampusRAGQueryLog) map[string]interface{} {
	if item == nil {
		return nil
	}
	chunks := make([]map[string]interface{}, 0, len(item.HitChunks))
	for _, chunk := range item.HitChunks {
		chunks = append(chunks, ragQueryChunkToMap(chunk))
	}
	return map[string]interface{}{
		"id":                 strconv.FormatInt(item.ID, 10),
		"user_id":            item.UserID,
		"post_id":            strconv.FormatInt(item.PostID, 10),
		"trigger_comment_id": strconv.FormatInt(item.TriggerCommentID, 10),
		"query":              item.Query,
		"need_knowledge":     item.NeedKnowledge,
		"confidence":         item.Confidence,
		"hit_chunks":         chunks,
		"answer":             item.Answer,
		"model":              item.Model,
		"duration_ms":        item.DurationMs,
		"error_message":      item.ErrorMessage,
		"created_at":         formatTime(item.CreatedAt),
	}
}

func adminSummaryToMap(summary *biz.CampusAdminSummary) map[string]interface{} {
	if summary == nil {
		return nil
	}
	trends := make([]map[string]interface{}, 0, len(summary.Trends))
	for _, trend := range summary.Trends {
		trends = append(trends, map[string]interface{}{
			"date":        trend.Date,
			"users":       trend.Users,
			"logins":      trend.Logins,
			"visits":      trend.Visits,
			"shares":      trend.Shares,
			"posts":       trend.Posts,
			"comments":    trend.Comments,
			"likes":       trend.Likes,
			"collections": trend.Collections,
			"reports":     trend.Reports,
		})
	}
	return map[string]interface{}{
		"total_users":        summary.TotalUsers,
		"today_users":        summary.TodayUsers,
		"total_logins":       summary.TotalLogins,
		"today_logins":       summary.TodayLogins,
		"total_visits":       summary.TotalVisits,
		"today_visits":       summary.TodayVisits,
		"total_shares":       summary.TotalShares,
		"today_shares":       summary.TodayShares,
		"today_publish_open": summary.TodayPublishOpen,
		"today_publish_done": summary.TodayPublishDone,
		"today_detail_views": summary.TodayDetailViews,
		"today_feedback":     summary.TodayFeedback,
		"today_reports":      summary.TodayReports,
		"total_posts":        summary.TotalPosts,
		"today_posts":        summary.TodayPosts,
		"total_comments":     summary.TotalComments,
		"today_comments":     summary.TodayComments,
		"total_likes":        summary.TotalLikes,
		"today_likes":        summary.TodayLikes,
		"total_collections":  summary.TotalCollections,
		"today_collections":  summary.TodayCollections,
		"total_reports":      summary.TotalReports,
		"pending_reports":    summary.PendingReports,
		"pending_feedback":   summary.PendingFeedback,
		"pending_posts":      summary.PendingPosts,
		"pending_comments":   summary.PendingComments,
		"pending_ai_audits":  summary.PendingAIAudits,
		"featured_posts":     summary.FeaturedPosts,
		"official_posts":     summary.OfficialPosts,
		"trends":             trends,
	}
}

func adminUserToMap(user *biz.CampusAdminUser) map[string]interface{} {
	if user == nil {
		return nil
	}
	return map[string]interface{}{
		"user":               userToMap(user.User),
		"profile":            profileToMap(user.Profile),
		"role":               user.Role,
		"post_count":         user.PostCount,
		"comment_count":      user.CommentCount,
		"like_count":         user.LikeCount,
		"collection_count":   user.CollectionCount,
		"feedback_count":     user.FeedbackCount,
		"report_count":       user.ReportCount,
		"login_count":        user.LoginCount,
		"visit_count":        user.VisitCount,
		"last_login_at":      formatTime(user.LastLoginAt),
		"last_active_at":     formatTime(user.LastActiveAt),
		"last_active_ip":     user.LastActiveIP,
		"last_active_path":   user.LastActivePath,
		"last_active_status": user.LastActiveStatus,
	}
}

func commentToMap(comment *biz.CampusForumComment) map[string]interface{} {
	if comment == nil {
		return nil
	}
	previewReplies := make([]map[string]interface{}, 0, len(comment.PreviewReplies))
	for _, reply := range comment.PreviewReplies {
		previewReplies = append(previewReplies, commentToMap(reply))
	}
	return map[string]interface{}{
		"id":                  strconv.FormatInt(comment.ID, 10),
		"post_id":             strconv.FormatInt(comment.PostID, 10),
		"post":                postToMap(comment.Post),
		"parent_id":           strconv.FormatInt(comment.ParentID, 10),
		"reply_to_comment_id": strconv.FormatInt(comment.ReplyToCommentID, 10),
		"reply_to_user_id":    comment.ReplyToUserID,
		"reply_to_user":       authorToMap(comment.ReplyToUser),
		"author":              authorToMap(comment.Author),
		"content":             comment.Content,
		"images":              comment.Images,
		"status":              comment.Status,
		"audit_reason":        comment.AuditReason,
		"like_count":          comment.LikeCount,
		"reply_count":         comment.ReplyCount,
		"is_liked":            comment.IsLiked,
		"preview_replies":     previewReplies,
		"created_at":          formatTime(comment.CreatedAt),
		"updated_at":          formatTime(comment.UpdatedAt),
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

func formatOptionalTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return formatTime(*t)
}

func parseOptionalRequestTime(value string) *time.Time {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return nil
	}
	if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
		return &parsed
	}
	layouts := []string{time.DateTime, "2006-01-02T15:04", "2006-01-02 15:04", time.DateOnly}
	for _, layout := range layouts {
		parsed, err := time.ParseInLocation(layout, raw, time.Local)
		if err == nil {
			return &parsed
		}
	}
	return nil
}
