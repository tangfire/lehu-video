package biz

import (
	"context"
	"sort"
	"testing"
	"time"

	core "lehu-video/api/videoCore/service/v1"
	"lehu-video/pkg/apperror"

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

func TestGetPublicCampusUserProfileHidesSensitiveProfileFields(t *testing.T) {
	repo := &campusRepoStub{
		roles: map[string]string{"10": "operator"},
		profiles: map[string]*CampusProfile{
			"10": {
				UserID:       "10",
				SchoolName:   "深圳职业技术大学",
				StudentNo:    "20260001",
				RealName:     "真实姓名",
				ClassName:    "深汕一班",
				DormBuilding: "A栋",
				RoomNo:       "101",
				Mobile:       "13800000000",
				AuthStatus:   CampusAuthStatusVerified,
			},
		},
		publicStats: &CampusPublicUserStats{
			PostCount:      3,
			LikeCount:      12,
			CollectedCount: 5,
		},
	}
	core := &campusCoreStub{users: map[string]*UserBaseInfo{
		"10": {ID: "10", Name: "真实姓名", Nickname: "深汕e仔", Avatar: "https://example.com/avatar.png", Mobile: "13800000000"},
	}}
	uc := NewCampusUsecase(repo, nil, core, nil, fixedCampusIDGenerator(1001), "secret", log.NewStdLogger(ioDiscard{}))

	profile, err := uc.GetPublicCampusUserProfile(context.Background(), "10")
	if err != nil {
		t.Fatalf("GetPublicCampusUserProfile() error = %v", err)
	}
	if profile.UserID != "10" || profile.Name != "深汕e仔" || profile.Nickname != "深汕e仔" {
		t.Fatalf("public profile basic fields mismatch: %+v", profile)
	}
	if profile.SchoolName != "深圳职业技术大学" || profile.AuthStatus != CampusAuthStatusVerified {
		t.Fatalf("campus public fields mismatch: %+v", profile)
	}
	if !profile.IsOfficial || profile.Bio == "" {
		t.Fatalf("official profile not marked: %+v", profile)
	}
	if profile.Stats == nil || profile.Stats.PostCount != 3 || profile.Stats.LikeCount != 12 || profile.Stats.CollectedCount != 5 {
		t.Fatalf("stats mismatch: %+v", profile.Stats)
	}
}

func TestGetPublicCampusUserProfileReturnsNotFound(t *testing.T) {
	repo := &campusRepoStub{roles: map[string]string{}}
	uc := NewCampusUsecase(repo, nil, &campusCoreStub{users: map[string]*UserBaseInfo{}}, nil, fixedCampusIDGenerator(1001), "secret", log.NewStdLogger(ioDiscard{}))

	_, err := uc.GetPublicCampusUserProfile(context.Background(), "404")
	if err == nil {
		t.Fatalf("GetPublicCampusUserProfile() expected error")
	}
	appErr := apperror.From(err)
	if appErr.Code != apperror.CodeNotFound {
		t.Fatalf("error code = %d, want not found", appErr.Code)
	}
}

func TestListPublicUserPostsOnlyVisibleAuthorPosts(t *testing.T) {
	repo := &campusRepoStub{
		roles: map[string]string{},
		posts: map[int64]*CampusForumPost{
			1: {ID: 1, AuthorID: "10", Title: "可见帖子", Status: CampusAuditStatusVisible, CreatedAt: time.Now()},
			2: {ID: 2, AuthorID: "10", Title: "待审核帖子", Status: CampusAuditStatusPending, CreatedAt: time.Now().Add(time.Second)},
			3: {ID: 3, AuthorID: "11", Title: "别人帖子", Status: CampusAuditStatusVisible, CreatedAt: time.Now().Add(2 * time.Second)},
		},
	}
	core := &campusCoreStub{users: map[string]*UserBaseInfo{"10": {ID: "10", Nickname: "同学"}}}
	uc := NewCampusUsecase(repo, nil, core, nil, fixedCampusIDGenerator(1001), "secret", log.NewStdLogger(ioDiscard{}))

	out, err := uc.ListPublicUserPosts(context.Background(), &ListCampusPostsInput{
		CurrentUserID: "12",
		AuthorID:      "10",
		Sort:          CampusPostSortRecommend,
		Page:          1,
		Size:          20,
	})
	if err != nil {
		t.Fatalf("ListPublicUserPosts() error = %v", err)
	}
	if repo.lastListQuery.AuthorID != "10" || repo.lastListQuery.IncludeDeleted {
		t.Fatalf("query privacy filters mismatch: %+v", repo.lastListQuery)
	}
	if len(repo.lastListQuery.Statuses) != 1 || repo.lastListQuery.Statuses[0] != CampusAuditStatusVisible {
		t.Fatalf("statuses = %+v, want visible only", repo.lastListQuery.Statuses)
	}
	if repo.lastListQuery.Sort != CampusPostSortNew {
		t.Fatalf("sort = %q, want fallback new", repo.lastListQuery.Sort)
	}
	if out.Total != 1 || len(out.Posts) != 1 || out.Posts[0].ID != 1 {
		t.Fatalf("posts = total %d %+v, want only visible author post", out.Total, out.Posts)
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

func TestAdminCreateSystemNotificationQueuesOutbox(t *testing.T) {
	repo := &campusRepoStub{roles: map[string]string{"10": "operator"}}
	uc := NewCampusUsecase(repo, nil, &campusCoreStub{}, nil, fixedCampusIDGenerator(1001), "secret", log.NewStdLogger(ioDiscard{}))

	taskID, err := uc.AdminCreateSystemNotification(context.Background(), &CreateCampusAdminNotificationInput{
		UserID:   "10",
		Title:    "内测提醒",
		Content:  "欢迎体验深汕校园e站",
		LinkPage: "community",
		Audience: "all_users",
	})
	if err != nil {
		t.Fatalf("AdminCreateSystemNotification() error = %v", err)
	}
	if taskID != 1001 {
		t.Fatalf("taskID = %d, want 1001", taskID)
	}
	if len(repo.notificationOutboxes) != 1 {
		t.Fatalf("outbox count = %d, want 1", len(repo.notificationOutboxes))
	}
	outbox := repo.notificationOutboxes[0]
	if outbox.EventType != CampusNotificationTypeSystem || outbox.Audience != "all_users" {
		t.Fatalf("outbox event/audience = %s/%s", outbox.EventType, outbox.Audience)
	}
	if len(repo.notifications) != 0 {
		t.Fatalf("notifications count = %d, want 0 before worker", len(repo.notifications))
	}
}

func TestProcessNotificationOutboxDeliversSystemNotification(t *testing.T) {
	repo := &campusRepoStub{
		recipients: []string{"1", "2"},
		notificationOutboxes: []*CampusNotificationOutbox{{
			ID:        1001,
			ActorID:   "10",
			EventType: CampusNotificationTypeSystem,
			Title:     "内测提醒",
			Content:   "欢迎体验深汕校园e站",
			LinkPage:  "community",
			Audience:  "all_users",
			Status:    CampusNotificationOutboxStatusPending,
		}},
	}
	uc := NewCampusUsecase(repo, nil, &campusCoreStub{}, nil, fixedCampusIDGenerator(2001), "secret", log.NewStdLogger(ioDiscard{}))

	if err := uc.ProcessPendingNotificationOutbox(context.Background(), 100); err != nil {
		t.Fatalf("ProcessPendingNotificationOutbox() error = %v", err)
	}
	if len(repo.notifications) != 2 {
		t.Fatalf("notifications count = %d, want 2", len(repo.notifications))
	}
	if repo.notifications[0].DedupeKey != "campus:system:1001:1" || repo.notifications[1].DedupeKey != "campus:system:1001:2" {
		t.Fatalf("unexpected dedupe keys: %#v %#v", repo.notifications[0].DedupeKey, repo.notifications[1].DedupeKey)
	}
	if len(repo.doneOutboxIDs) != 1 || repo.doneOutboxIDs[0] != 1001 {
		t.Fatalf("done outbox ids = %#v, want [1001]", repo.doneOutboxIDs)
	}
}

type fixedCampusIDGenerator int64

func (g fixedCampusIDGenerator) NextID() int64 { return int64(g) }

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }

type campusRepoStub struct {
	category             *CampusForumCategory
	roles                map[string]string
	posts                map[int64]*CampusForumPost
	profiles             map[string]*CampusProfile
	publicStats          *CampusPublicUserStats
	blockedIPs           map[string]bool
	lastPost             *CampusForumPost
	lastFeedback         *CampusFeedback
	lastListQuery        ListCampusPostQuery
	recipients           []string
	notifications        []*CampusNotification
	notificationOutboxes []*CampusNotificationOutbox
	doneOutboxIDs        []int64
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
	if len(r.posts) == 0 {
		return []*CampusForumPost{}, 0, nil
	}
	statusSet := map[int32]bool{}
	for _, status := range query.Statuses {
		statusSet[status] = true
	}
	posts := make([]*CampusForumPost, 0, len(r.posts))
	for _, post := range r.posts {
		if post == nil {
			continue
		}
		if query.AuthorID != "" && post.AuthorID != query.AuthorID {
			continue
		}
		if query.PostType != "" && post.PostType != query.PostType {
			continue
		}
		if query.CategoryCode != "" && post.CategoryCode != query.CategoryCode {
			continue
		}
		if len(statusSet) > 0 && !statusSet[post.Status] {
			continue
		}
		copyPost := *post
		posts = append(posts, &copyPost)
	}
	sort.SliceStable(posts, func(i, j int) bool {
		return posts[i].CreatedAt.After(posts[j].CreatedAt)
	})
	total := int64(len(posts))
	if query.Offset >= len(posts) {
		return []*CampusForumPost{}, total, nil
	}
	end := query.Offset + query.Limit
	if query.Limit <= 0 || end > len(posts) {
		end = len(posts)
	}
	return posts[query.Offset:end], total, nil
}
func (r *campusRepoStub) GetPublicUserPostStats(context.Context, string) (*CampusPublicUserStats, error) {
	if r.publicStats != nil {
		return r.publicStats, nil
	}
	return &CampusPublicUserStats{}, nil
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
func (r *campusRepoStub) GetProfileByUserID(_ context.Context, userID string) (bool, *CampusProfile, error) {
	if r.profiles == nil {
		return false, nil, nil
	}
	profile := r.profiles[userID]
	if profile == nil {
		return false, nil, nil
	}
	copyProfile := *profile
	return true, &copyProfile, nil
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
func (r *campusRepoStub) CreateCommentWithOutbox(context.Context, *CampusForumComment, *CampusNotificationOutbox) error {
	return nil
}
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
func (r *campusRepoStub) AddCommentLikeWithOutbox(context.Context, int64, string, int64, *CampusNotificationOutbox) error {
	return nil
}
func (r *campusRepoStub) RemoveCommentLike(context.Context, string, int64) error { return nil }
func (r *campusRepoStub) GetPostLikeStatus(context.Context, string, []int64) (map[int64]bool, error) {
	return nil, nil
}
func (r *campusRepoStub) AddPostLike(context.Context, int64, string, int64) error { return nil }
func (r *campusRepoStub) AddPostLikeWithOutbox(context.Context, int64, string, int64, *CampusNotificationOutbox) error {
	return nil
}
func (r *campusRepoStub) RemovePostLike(context.Context, string, int64) error { return nil }
func (r *campusRepoStub) GetPostCollectionStatus(context.Context, string, []int64) (map[int64]bool, error) {
	return nil, nil
}
func (r *campusRepoStub) AddPostCollection(context.Context, int64, string, int64) error {
	return nil
}
func (r *campusRepoStub) AddPostCollectionWithOutbox(context.Context, int64, string, int64, *CampusNotificationOutbox) error {
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
func (r *campusRepoStub) CreateNotification(_ context.Context, notification *CampusNotification, _ bool) error {
	if notification != nil {
		copyNotification := *notification
		r.notifications = append(r.notifications, &copyNotification)
	}
	return nil
}
func (r *campusRepoStub) BulkCreateNotifications(context.Context, []*CampusNotification) error {
	return nil
}
func (r *campusRepoStub) CreateNotificationOutbox(_ context.Context, outbox *CampusNotificationOutbox) error {
	if outbox != nil {
		copyOutbox := *outbox
		r.notificationOutboxes = append(r.notificationOutboxes, &copyOutbox)
	}
	return nil
}
func (r *campusRepoStub) ClaimNotificationOutbox(context.Context, int, time.Duration) ([]*CampusNotificationOutbox, error) {
	out := r.notificationOutboxes
	r.notificationOutboxes = nil
	return out, nil
}
func (r *campusRepoStub) MarkNotificationOutboxDone(_ context.Context, id int64) error {
	r.doneOutboxIDs = append(r.doneOutboxIDs, id)
	return nil
}
func (r *campusRepoStub) MarkNotificationOutboxRetry(context.Context, int64, int32, *time.Time, string, bool) error {
	return nil
}
func (r *campusRepoStub) ListNotifications(context.Context, string, string, int, int) ([]*CampusNotification, int64, error) {
	return nil, 0, nil
}
func (r *campusRepoStub) CountUnreadNotifications(context.Context, string) (*CampusUnreadNotificationCount, error) {
	return &CampusUnreadNotificationCount{}, nil
}
func (r *campusRepoStub) MarkNotificationRead(context.Context, string, int64) error {
	return nil
}
func (r *campusRepoStub) MarkAllNotificationsRead(context.Context, string) error {
	return nil
}
func (r *campusRepoStub) ListNotificationRecipients(context.Context) ([]string, error) {
	return r.recipients, nil
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
func (r *campusRepoStub) ListCampusUsers(context.Context, string, string, int32, int, int) ([]*CampusAdminUser, int64, error) {
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
func (r *campusCoreStub) GetUserBaseInfo(ctx context.Context, userID, accountID string) (*UserBaseInfo, error) {
	_ = ctx
	_ = accountID
	if r.users != nil {
		return r.users[userID], nil
	}
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
