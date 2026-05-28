package biz

import (
	"context"
	"testing"
	"time"

	core "lehu-video/api/videoCore/service/v1"

	"github.com/go-kratos/kratos/v2/log"
)

func TestNormalizeCampusPostMedia(t *testing.T) {
	tests := []struct {
		name      string
		mediaType string
		images    []string
		coverURL  string
		videoURL  string
		wantType  string
		wantCover string
		wantVideo string
		wantErr   bool
	}{
		{
			name:     "default text",
			wantType: CampusPostMediaText,
		},
		{
			name:      "image defaults cover",
			mediaType: CampusPostMediaImage,
			images:    []string{"https://example.com/1.jpg"},
			wantType:  CampusPostMediaImage,
			wantCover: "https://example.com/1.jpg",
		},
		{
			name:      "video requires cover and url",
			mediaType: CampusPostMediaVideo,
			coverURL:  "https://example.com/cover.jpg",
			videoURL:  "https://example.com/video.mp4",
			wantType:  CampusPostMediaVideo,
			wantCover: "https://example.com/cover.jpg",
			wantVideo: "https://example.com/video.mp4",
		},
		{
			name:      "image requires images",
			mediaType: CampusPostMediaImage,
			wantErr:   true,
		},
		{
			name:      "video requires cover",
			mediaType: CampusPostMediaVideo,
			videoURL:  "https://example.com/video.mp4",
			wantErr:   true,
		},
		{
			name:      "invalid type",
			mediaType: "mixed",
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, gotCover, gotVideo, err := normalizeCampusPostMedia(tt.mediaType, tt.images, tt.coverURL, tt.videoURL)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotType != tt.wantType || gotCover != tt.wantCover || gotVideo != tt.wantVideo {
				t.Fatalf("got (%q, %q, %q), want (%q, %q, %q)", gotType, gotCover, gotVideo, tt.wantType, tt.wantCover, tt.wantVideo)
			}
		})
	}
}

func TestCreatePostIgnoresOpsFlagsForNormalUser(t *testing.T) {
	users := map[string]*UserBaseInfo{"10": {ID: "10", Nickname: "同学"}}
	repo := &campusRepoStub{
		category: &CampusForumCategory{Code: "guide", Name: "校园攻略"},
		roles:    map[string]string{},
	}
	core := &campusCoreStub{users: users}
	uc := NewCampusUsecase(repo, nil, core, nil, fixedCampusIDGenerator(1001), "secret", log.NewStdLogger(ioDiscard{}))

	post, err := uc.CreatePost(context.Background(), &CreateCampusPostInput{
		UserID:       "10",
		CategoryCode: "guide",
		Title:        "报到攻略",
		Content:      "这里是报到攻略内容",
		MediaType:    CampusPostMediaText,
		PostType:     CampusPostTypeGuide,
		IsOfficial:   true,
		IsFeatured:   true,
		IsPinned:     true,
		SortWeight:   100,
	})
	if err != nil {
		t.Fatalf("CreatePost() error = %v", err)
	}
	if post.IsOfficial || post.IsFeatured || post.IsPinned || post.SortWeight != 0 {
		t.Fatalf("normal user ops flags leaked: official=%v featured=%v pinned=%v weight=%d", post.IsOfficial, post.IsFeatured, post.IsPinned, post.SortWeight)
	}
}

func TestCreatePostAllowsOpsFlagsForOperator(t *testing.T) {
	users := map[string]*UserBaseInfo{"10": {ID: "10", Nickname: "深汕e仔"}}
	repo := &campusRepoStub{
		category: &CampusForumCategory{Code: "guide", Name: "校园攻略"},
		roles:    map[string]string{"10": "operator"},
	}
	core := &campusCoreStub{users: users}
	uc := NewCampusUsecase(repo, nil, core, nil, fixedCampusIDGenerator(1001), "secret", log.NewStdLogger(ioDiscard{}))

	post, err := uc.CreatePost(context.Background(), &CreateCampusPostInput{
		UserID:       "10",
		CategoryCode: "guide",
		Title:        "报到攻略",
		Content:      "这里是报到攻略内容",
		MediaType:    CampusPostMediaText,
		PostType:     CampusPostTypeGuide,
		IsOfficial:   true,
		IsFeatured:   true,
		IsPinned:     true,
		SortWeight:   100,
	})
	if err != nil {
		t.Fatalf("CreatePost() error = %v", err)
	}
	if !post.IsOfficial || !post.IsFeatured || !post.IsPinned || post.SortWeight != 100 {
		t.Fatalf("operator ops flags not applied: official=%v featured=%v pinned=%v weight=%d", post.IsOfficial, post.IsFeatured, post.IsPinned, post.SortWeight)
	}
}

func TestListPostsPassesPostTypeQuery(t *testing.T) {
	repo := &campusRepoStub{roles: map[string]string{}}
	uc := NewCampusUsecase(repo, nil, &campusCoreStub{}, nil, fixedCampusIDGenerator(1001), "secret", log.NewStdLogger(ioDiscard{}))

	if _, err := uc.ListPosts(context.Background(), &ListCampusPostsInput{PostType: CampusPostTypeQuestion, Page: 1, Size: 20}); err != nil {
		t.Fatalf("ListPosts() error = %v", err)
	}
	if repo.lastListQuery.PostType != CampusPostTypeQuestion {
		t.Fatalf("PostType query = %q, want %q", repo.lastListQuery.PostType, CampusPostTypeQuestion)
	}
}

func TestListPostsDefaultsToRecommendSort(t *testing.T) {
	repo := &campusRepoStub{roles: map[string]string{}}
	uc := NewCampusUsecase(repo, nil, &campusCoreStub{}, nil, fixedCampusIDGenerator(1001), "secret", log.NewStdLogger(ioDiscard{}))

	if _, err := uc.ListPosts(context.Background(), &ListCampusPostsInput{Sort: "unknown", Page: 1, Size: 20}); err != nil {
		t.Fatalf("ListPosts() error = %v", err)
	}
	if repo.lastListQuery.Sort != CampusPostSortRecommend {
		t.Fatalf("Sort = %q, want %q", repo.lastListQuery.Sort, CampusPostSortRecommend)
	}
}

func TestAdminListPostsDefaultsToNewSort(t *testing.T) {
	repo := &campusRepoStub{roles: map[string]string{"10": "operator"}}
	uc := NewCampusUsecase(repo, nil, &campusCoreStub{}, nil, fixedCampusIDGenerator(1001), "secret", log.NewStdLogger(ioDiscard{}))

	if _, err := uc.AdminListPosts(context.Background(), &ListCampusAdminPostsInput{
		UserID: "10",
		Sort:   "unknown",
		Page:   1,
		Size:   20,
	}); err != nil {
		t.Fatalf("AdminListPosts() error = %v", err)
	}
	if repo.lastListQuery.Sort != CampusPostSortNew {
		t.Fatalf("Sort = %q, want %q", repo.lastListQuery.Sort, CampusPostSortNew)
	}
}

func TestAdminListPostsPassesOpsFilters(t *testing.T) {
	repo := &campusRepoStub{roles: map[string]string{"10": "operator"}}
	uc := NewCampusUsecase(repo, nil, &campusCoreStub{}, nil, fixedCampusIDGenerator(1001), "secret", log.NewStdLogger(ioDiscard{}))

	if _, err := uc.AdminListPosts(context.Background(), &ListCampusAdminPostsInput{
		UserID:       "10",
		PostType:     CampusPostTypeGuide,
		CategoryCode: "life",
		OpsFilter:    "pinned",
		Status:       CampusAuditStatusVisible,
		Page:         1,
		Size:         20,
	}); err != nil {
		t.Fatalf("AdminListPosts() error = %v", err)
	}
	if repo.lastListQuery.PostType != CampusPostTypeGuide || repo.lastListQuery.CategoryCode != "life" {
		t.Fatalf("query mismatch: %+v", repo.lastListQuery)
	}
	if repo.lastListQuery.OnlyPinned == nil || !*repo.lastListQuery.OnlyPinned {
		t.Fatalf("OnlyPinned query not set: %+v", repo.lastListQuery)
	}
	if len(repo.lastListQuery.Statuses) != 1 || repo.lastListQuery.Statuses[0] != CampusAuditStatusVisible {
		t.Fatalf("Statuses = %+v, want visible", repo.lastListQuery.Statuses)
	}
}

func TestAdminBatchPostsRequiresOperator(t *testing.T) {
	repo := &campusRepoStub{roles: map[string]string{}}
	uc := NewCampusUsecase(repo, nil, &campusCoreStub{}, nil, fixedCampusIDGenerator(1001), "secret", log.NewStdLogger(ioDiscard{}))

	if _, err := uc.AdminBatchPosts(context.Background(), &BatchCampusAdminPostsInput{
		UserID:  "10",
		PostIDs: []int64{1},
		Action:  "pin",
	}); err == nil {
		t.Fatalf("AdminBatchPosts() expected forbidden error")
	}
}

func TestAdminBatchPostsUpdatesContentFlags(t *testing.T) {
	repo := &campusRepoStub{
		roles: map[string]string{"10": "operator"},
		posts: map[int64]*CampusForumPost{
			1: {
				ID:           1,
				CategoryCode: "guide",
				AuthorID:     "10",
				Title:        "报到攻略",
				Content:      "内容",
				MediaType:    CampusPostMediaText,
				PostType:     CampusPostTypeGuide,
				Status:       CampusAuditStatusPending,
			},
			2: {
				ID:           2,
				CategoryCode: "life",
				AuthorID:     "10",
				Title:        "宿舍 FAQ",
				Content:      "内容",
				MediaType:    CampusPostMediaText,
				PostType:     CampusPostTypeGuide,
				Status:       CampusAuditStatusPending,
			},
		},
	}
	uc := NewCampusUsecase(repo, nil, &campusCoreStub{}, nil, fixedCampusIDGenerator(1001), "secret", log.NewStdLogger(ioDiscard{}))

	out, err := uc.AdminBatchPosts(context.Background(), &BatchCampusAdminPostsInput{
		UserID:  "10",
		PostIDs: []int64{1, 2, 2},
		Action:  "pin",
	})
	if err != nil {
		t.Fatalf("AdminBatchPosts(pin) error = %v", err)
	}
	if out.UpdatedCount != 2 {
		t.Fatalf("updated count = %d, want 2", out.UpdatedCount)
	}
	if !repo.posts[1].IsPinned || !repo.posts[2].IsPinned {
		t.Fatalf("posts not pinned: %+v %+v", repo.posts[1], repo.posts[2])
	}

	if _, err := uc.AdminBatchPosts(context.Background(), &BatchCampusAdminPostsInput{
		UserID:     "10",
		PostIDs:    []int64{1},
		Action:     "set_weight",
		SortWeight: 120,
	}); err != nil {
		t.Fatalf("AdminBatchPosts(set_weight) error = %v", err)
	}
	if repo.posts[1].SortWeight != 120 {
		t.Fatalf("sort weight = %d, want 120", repo.posts[1].SortWeight)
	}

	if _, err := uc.AdminBatchPosts(context.Background(), &BatchCampusAdminPostsInput{
		UserID:  "10",
		PostIDs: []int64{1},
		Action:  "visible",
	}); err != nil {
		t.Fatalf("AdminBatchPosts(visible) error = %v", err)
	}
	if repo.posts[1].Status != CampusAuditStatusVisible {
		t.Fatalf("status = %d, want visible", repo.posts[1].Status)
	}
}

func TestCreateFeedbackSanitizesInput(t *testing.T) {
	repo := &campusRepoStub{roles: map[string]string{}}
	uc := NewCampusUsecase(repo, nil, &campusCoreStub{}, nil, fixedCampusIDGenerator(2001), "secret", log.NewStdLogger(ioDiscard{}))

	feedback, err := uc.CreateFeedback(context.Background(), &CreateCampusFeedbackInput{
		UserID:       "10",
		FeedbackType: "bug",
		Content:      "发布页草稿恢复有点问题",
		Contact:      "微信 test",
		Images:       []string{"https://example.com/1.jpg", "", "https://example.com/2.jpg", "https://example.com/3.jpg", "https://example.com/4.jpg"},
	})
	if err != nil {
		t.Fatalf("CreateFeedback() error = %v", err)
	}
	if feedback.ID != 2001 || feedback.FeedbackType != "bug" || feedback.Status != CampusFeedbackStatusPending {
		t.Fatalf("feedback mismatch: %+v", feedback)
	}
	if len(feedback.Images) != 3 {
		t.Fatalf("images len = %d, want 3", len(feedback.Images))
	}
	if repo.lastFeedback == nil || repo.lastFeedback.Content != "发布页草稿恢复有点问题" {
		t.Fatalf("feedback not persisted: %+v", repo.lastFeedback)
	}
}

func TestAdminListFeedbackRequiresOperator(t *testing.T) {
	repo := &campusRepoStub{roles: map[string]string{}}
	uc := NewCampusUsecase(repo, nil, &campusCoreStub{}, nil, fixedCampusIDGenerator(1001), "secret", log.NewStdLogger(ioDiscard{}))

	if _, err := uc.AdminListFeedback(context.Background(), &ListCampusFeedbackInput{UserID: "10"}); err == nil {
		t.Fatalf("AdminListFeedback() expected forbidden error")
	}
}

func TestCheckCampusRequestBlocksIP(t *testing.T) {
	repo := &campusRepoStub{roles: map[string]string{}, blockedIPs: map[string]bool{"1.2.3.4": true}}
	uc := NewCampusUsecase(repo, nil, &campusCoreStub{}, nil, fixedCampusIDGenerator(1001), "secret", log.NewStdLogger(ioDiscard{}))

	blocked, allowed, err := uc.CheckCampusRequest(context.Background(), &CampusRateLimitInput{
		IP:       "1.2.3.4",
		Method:   "GET",
		Path:     "/v1/campus/forum/posts",
		Category: "read",
	})
	if err != nil {
		t.Fatalf("CheckCampusRequest() error = %v", err)
	}
	if !blocked || allowed {
		t.Fatalf("blocked=%v allowed=%v, want blocked only", blocked, allowed)
	}
}

func TestAdminBlockIPRequiresOperator(t *testing.T) {
	repo := &campusRepoStub{roles: map[string]string{}}
	uc := NewCampusUsecase(repo, nil, &campusCoreStub{}, nil, fixedCampusIDGenerator(1001), "secret", log.NewStdLogger(ioDiscard{}))

	if err := uc.AdminBlockIP(context.Background(), &BlockCampusIPInput{UserID: "10", IP: "1.2.3.4"}); err == nil {
		t.Fatalf("AdminBlockIP() expected forbidden error")
	}
}

type fixedCampusIDGenerator int64

func (g fixedCampusIDGenerator) NextID() int64 { return int64(g) }

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }

type campusRepoStub struct {
	category      *CampusForumCategory
	roles         map[string]string
	posts         map[int64]*CampusForumPost
	blockedIPs    map[string]bool
	lastPost      *CampusForumPost
	lastFeedback  *CampusFeedback
	lastListQuery ListCampusPostQuery
}

func (r *campusRepoStub) GetCategoryByCode(ctx context.Context, code string) (bool, *CampusForumCategory, error) {
	if r.category == nil || r.category.Code != code {
		return false, nil, nil
	}
	return true, r.category, nil
}

func (r *campusRepoStub) CreatePost(ctx context.Context, post *CampusForumPost) error {
	r.lastPost = post
	return nil
}

func (r *campusRepoStub) ListPosts(ctx context.Context, query ListCampusPostQuery) ([]*CampusForumPost, int64, error) {
	r.lastListQuery = query
	return []*CampusForumPost{}, 0, nil
}
func (r *campusRepoStub) ListPostsByIDs(context.Context, []int64, []int32) ([]*CampusForumPost, error) {
	return []*CampusForumPost{}, nil
}

func (r *campusRepoStub) GetCampusOperatorRole(ctx context.Context, userID string) (string, error) {
	return r.roles[userID], nil
}

func (r *campusRepoStub) GetWechatIdentity(context.Context, string, string) (bool, *CampusWechatIdentity, error) {
	return false, nil, nil
}
func (r *campusRepoStub) GetAccountIDByEmail(context.Context, string) (bool, string, error) {
	return false, "", nil
}
func (r *campusRepoStub) SaveWechatIdentity(context.Context, *CampusWechatIdentity) error { return nil }
func (r *campusRepoStub) GetProfileByUserID(context.Context, string) (bool, *CampusProfile, error) {
	return false, nil, nil
}
func (r *campusRepoStub) SaveProfile(context.Context, *CampusProfile) error   { return nil }
func (r *campusRepoStub) UpdateProfile(context.Context, *CampusProfile) error { return nil }
func (r *campusRepoStub) ReplaceTimetableCourses(context.Context, string, string, string, []*CampusTimetableCourse) error {
	return nil
}
func (r *campusRepoStub) ListTimetableCourses(context.Context, string, string) ([]*CampusTimetableCourse, error) {
	return nil, nil
}
func (r *campusRepoStub) ListCategories(context.Context) ([]*CampusForumCategory, error) {
	return nil, nil
}
func (r *campusRepoStub) GetPostByID(context.Context, int64) (bool, *CampusForumPost, error) {
	return false, nil, nil
}
func (r *campusRepoStub) GetAnyPostByID(ctx context.Context, postID int64) (bool, *CampusForumPost, error) {
	_ = ctx
	post := r.posts[postID]
	if post == nil {
		return false, nil, nil
	}
	copyPost := *post
	return true, &copyPost, nil
}
func (r *campusRepoStub) DeletePost(context.Context, int64) error { return nil }
func (r *campusRepoStub) UpdatePostStatus(context.Context, int64, int32, string) error {
	return nil
}
func (r *campusRepoStub) UpdatePostByAdmin(ctx context.Context, post *CampusForumPost) error {
	_ = ctx
	copyPost := *post
	if r.posts == nil {
		r.posts = map[int64]*CampusForumPost{}
	}
	r.posts[post.ID] = &copyPost
	return nil
}
func (r *campusRepoStub) CreateComment(context.Context, *CampusForumComment) error { return nil }
func (r *campusRepoStub) ListComments(context.Context, ListCampusCommentQuery) ([]*CampusForumComment, int64, error) {
	return nil, 0, nil
}
func (r *campusRepoStub) FillCommentPosts(context.Context, []*CampusForumComment) error {
	return nil
}
func (r *campusRepoStub) GetCommentByID(context.Context, int64) (bool, *CampusForumComment, error) {
	return false, nil, nil
}
func (r *campusRepoStub) GetAnyCommentByID(context.Context, int64) (bool, *CampusForumComment, error) {
	return false, nil, nil
}
func (r *campusRepoStub) DeleteComment(context.Context, int64) error { return nil }
func (r *campusRepoStub) UpdateCommentStatus(context.Context, int64, int32, string) error {
	return nil
}
func (r *campusRepoStub) GetCommentLikeStatus(context.Context, string, []int64) (map[int64]bool, error) {
	return nil, nil
}
func (r *campusRepoStub) AddCommentLike(context.Context, int64, string, int64) error { return nil }
func (r *campusRepoStub) RemoveCommentLike(context.Context, string, int64) error     { return nil }
func (r *campusRepoStub) GetPostLikeStatus(context.Context, string, []int64) (map[int64]bool, error) {
	return nil, nil
}
func (r *campusRepoStub) AddPostLike(context.Context, int64, string, int64) error { return nil }
func (r *campusRepoStub) RemovePostLike(context.Context, string, int64) error     { return nil }
func (r *campusRepoStub) GetPostCollectionStatus(context.Context, string, []int64) (map[int64]bool, error) {
	return nil, nil
}
func (r *campusRepoStub) AddPostCollection(context.Context, int64, string, int64) error {
	return nil
}
func (r *campusRepoStub) RemovePostCollection(context.Context, string, int64) error {
	return nil
}
func (r *campusRepoStub) CreateReport(context.Context, *CampusForumReport) error { return nil }
func (r *campusRepoStub) ListReports(context.Context, int32, int, int) ([]*CampusForumReport, int64, error) {
	return nil, 0, nil
}
func (r *campusRepoStub) UpdateReportStatus(context.Context, int64, int32) error { return nil }
func (r *campusRepoStub) CreateFeedback(ctx context.Context, feedback *CampusFeedback) error {
	_ = ctx
	r.lastFeedback = feedback
	return nil
}
func (r *campusRepoStub) ListFeedback(context.Context, int32, int, int) ([]*CampusFeedback, int64, error) {
	return nil, 0, nil
}
func (r *campusRepoStub) UpdateFeedbackStatus(context.Context, int64, int32, string) error {
	return nil
}
func (r *campusRepoStub) IsIPBlocked(ctx context.Context, ip string) (bool, error) {
	_ = ctx
	return r.blockedIPs[ip], nil
}
func (r *campusRepoStub) AllowCampusRequest(context.Context, string, int64, time.Duration) (bool, error) {
	return true, nil
}
func (r *campusRepoStub) CreateAccessLog(context.Context, *CampusAccessLog) error { return nil }
func (r *campusRepoStub) CreateAccessLogs(context.Context, []*CampusAccessLog) error {
	return nil
}
func (r *campusRepoStub) GetSecurityOverview(context.Context) (*CampusSecurityOverview, error) {
	return &CampusSecurityOverview{}, nil
}
func (r *campusRepoStub) BlockIP(context.Context, *CampusIPBlock) error         { return nil }
func (r *campusRepoStub) UnblockIP(context.Context, string) error               { return nil }
func (r *campusRepoStub) CreateAuditLog(context.Context, *CampusAuditLog) error { return nil }
func (r *campusRepoStub) TrackEvent(context.Context, *TrackCampusEventInput) error {
	return nil
}
func (r *campusRepoStub) TrackEvents(context.Context, []*TrackCampusEventInput) error {
	return nil
}
func (r *campusRepoStub) GetAdminSummary(context.Context) (*CampusAdminSummary, error) {
	return nil, nil
}
func (r *campusRepoStub) ReconcileCampusStats(context.Context) (*CampusStatsReconcileResult, error) {
	return &CampusStatsReconcileResult{}, nil
}
func (r *campusRepoStub) ListCampusUsers(context.Context, string, int, int) ([]*CampusAdminUser, int64, error) {
	return nil, 0, nil
}
func (r *campusRepoStub) UpsertCampusOperator(context.Context, string, string) error { return nil }
func (r *campusRepoStub) RemoveCampusOperator(context.Context, string) error         { return nil }

type campusCoreStub struct {
	users map[string]*UserBaseInfo
}

func (r *campusCoreStub) CreateUser(context.Context, string, string, string) (string, error) {
	return "", nil
}
func (r *campusCoreStub) GetUserBaseInfo(context.Context, string, string) (*UserBaseInfo, error) {
	return nil, nil
}
func (r *campusCoreStub) BatchGetUserBaseInfo(ctx context.Context, userIDs []string) ([]*UserBaseInfo, error) {
	users := make([]*UserBaseInfo, 0, len(userIDs))
	for _, id := range userIDs {
		if user := r.users[id]; user != nil {
			users = append(users, user)
		}
	}
	return users, nil
}
func (r *campusCoreStub) GetUserInfoByIdList(context.Context, []string) ([]*UserInfo, error) {
	return nil, nil
}
func (r *campusCoreStub) UpdateUserInfo(context.Context, string, string, string, string, string, string, int32) error {
	return nil
}
func (r *campusCoreStub) SearchUsers(context.Context, string, int32, int32) (int64, []*UserBaseInfo, error) {
	return 0, nil, nil
}
func (r *campusCoreStub) SaveVideoInfo(context.Context, string, string, string, string, string) (string, error) {
	return "", nil
}
func (r *campusCoreStub) GetVideoById(context.Context, string, string) (*Video, error) {
	return nil, nil
}
func (r *campusCoreStub) GetVideoByIdList(context.Context, []string) ([]*Video, error) {
	return nil, nil
}
func (r *campusCoreStub) ListPublishedVideo(context.Context, string, *PageStats) (int64, []*Video, error) {
	return 0, nil, nil
}
func (r *campusCoreStub) IsUserFavoriteVideo(context.Context, string, []string) (map[string]bool, error) {
	return nil, nil
}
func (r *campusCoreStub) IsFollowing(context.Context, string, []string) (map[string]bool, error) {
	return nil, nil
}
func (r *campusCoreStub) IsCollected(context.Context, string, []string) (map[string]bool, error) {
	return nil, nil
}
func (r *campusCoreStub) CountComments4Video(context.Context, []string) (map[string]int64, error) {
	return nil, nil
}
func (r *campusCoreStub) CountFavorite4Video(context.Context, []string) (map[string]FavoriteCount, error) {
	return nil, nil
}
func (r *campusCoreStub) CountFavorite4Comment(context.Context, []string) (map[string]FavoriteCount, error) {
	return nil, nil
}
func (r *campusCoreStub) CountCollected4Video(context.Context, []string) (map[string]int64, error) {
	return nil, nil
}
func (r *campusCoreStub) GetFeed(context.Context, string, int64, int32, int32) ([]*FeedItem, int64, error) {
	return nil, 0, nil
}
func (r *campusCoreStub) CreateComment(context.Context, string, string, string, string, string) (*Comment, error) {
	return nil, nil
}
func (r *campusCoreStub) GetCommentById(context.Context, string) (*Comment, error) { return nil, nil }
func (r *campusCoreStub) RemoveComment(context.Context, string, string) error      { return nil }
func (r *campusCoreStub) ListChildComment(context.Context, string, *PageStats) (int64, []*Comment, error) {
	return 0, nil, nil
}
func (r *campusCoreStub) ListComment4Video(context.Context, string, *PageStats) (int64, []*Comment, error) {
	return 0, nil, nil
}
func (r *campusCoreStub) AddFavorite(context.Context, string, string, *FavoriteTarget, *FavoriteType) error {
	return nil
}
func (r *campusCoreStub) RemoveFavorite(context.Context, string, string, *FavoriteTarget, *FavoriteType) error {
	return nil
}
func (r *campusCoreStub) ListUserFavoriteVideo(context.Context, string, *PageStats) (int64, []string, error) {
	return 0, nil, nil
}
func (r *campusCoreStub) GetFavoriteStats(context.Context, string, *FavoriteTarget) (*FavoriteStats, error) {
	return nil, nil
}
func (r *campusCoreStub) CountBeFavoriteNumber4User(context.Context, string) (int64, error) {
	return 0, nil
}
func (r *campusCoreStub) BatchIsFavorite(context.Context, string, []string, FavoriteTarget) (*core.BatchIsFavoriteResp, error) {
	return nil, nil
}
func (r *campusCoreStub) IsFavorite(context.Context, string, string, *FavoriteTarget) (*IsFavoriteResult, error) {
	return nil, nil
}
func (r *campusCoreStub) AddFollow(context.Context, string, string) error    { return nil }
func (r *campusCoreStub) RemoveFollow(context.Context, string, string) error { return nil }
func (r *campusCoreStub) ListFollow(context.Context, string, *FollowType, *PageStats) (int64, []string, error) {
	return 0, nil, nil
}
func (r *campusCoreStub) GetCollectionById(context.Context, string) (*Collection, error) {
	return nil, nil
}
func (r *campusCoreStub) AddVideo2Collection(context.Context, string, string, string) error {
	return nil
}
func (r *campusCoreStub) AddCollection(context.Context, *Collection) error { return nil }
func (r *campusCoreStub) ListCollection(context.Context, string, *PageStats) (int64, []*Collection, error) {
	return 0, nil, nil
}
func (r *campusCoreStub) ListVideo4Collection(context.Context, string, *PageStats) (int64, []string, error) {
	return 0, nil, nil
}
func (r *campusCoreStub) RemoveCollection(context.Context, string, string) error { return nil }
func (r *campusCoreStub) RemoveVideo4Collection(context.Context, string, string, string) error {
	return nil
}
func (r *campusCoreStub) UpdateCollection(context.Context, *Collection) error { return nil }
func (r *campusCoreStub) CountFollow4User(context.Context, string) ([]int64, error) {
	return nil, nil
}
