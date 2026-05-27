package biz

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
)

type feedTestVideoRepo struct {
	latest []*Video
	hot    []*Video
}

func (r *feedTestVideoRepo) PublishVideo(context.Context, *Video) (int64, error) {
	return 0, nil
}
func (r *feedTestVideoRepo) GetVideoById(context.Context, int64) (bool, *Video, error) {
	return false, nil, nil
}
func (r *feedTestVideoRepo) GetVideoListByUid(context.Context, int64, time.Time, PageStats) (int64, []*Video, error) {
	return 0, nil, nil
}
func (r *feedTestVideoRepo) GetVideoByIdList(context.Context, []int64) ([]*Video, error) {
	return nil, nil
}
func (r *feedTestVideoRepo) GetFeedVideos(context.Context, time.Time, PageStats) ([]*Video, error) {
	return nil, nil
}
func (r *feedTestVideoRepo) GetHotVideos(context.Context, int) ([]*Video, error) {
	return r.hot, nil
}
func (r *feedTestVideoRepo) GetVideosByAuthors(context.Context, []string, int64, int) ([]*Video, error) {
	return nil, nil
}
func (r *feedTestVideoRepo) GetVideosByAuthorsExclude(context.Context, []string, int64, int, []string) ([]*Video, error) {
	return nil, nil
}
func (r *feedTestVideoRepo) GetAuthorInfo(context.Context, string) (*Author, error) {
	return nil, nil
}
func (r *feedTestVideoRepo) GetVideoListByTime(_ context.Context, latestTime time.Time, limit int) ([]*Video, error) {
	out := make([]*Video, 0, len(r.latest))
	for _, video := range r.latest {
		if video.UploadTime.Before(latestTime) {
			out = append(out, video)
		}
		if len(out) == limit {
			break
		}
	}
	return out, nil
}
func (r *feedTestVideoRepo) GetVideoStats(context.Context, string) (*VideoStats, error) {
	return nil, nil
}
func (r *feedTestVideoRepo) GetAllVideoIDs(context.Context, int64, int) ([]string, error) {
	return nil, nil
}
func (r *feedTestVideoRepo) BatchGetVideoAuthors(context.Context, []int64) (map[int64]int64, error) {
	return nil, nil
}

type feedTestFollowRepo struct{}

func (r *feedTestFollowRepo) CreateFollow(context.Context, int64, int64) error { return nil }
func (r *feedTestFollowRepo) GetFollow(context.Context, int64, int64) (bool, int64, bool, error) {
	return false, 0, false, nil
}
func (r *feedTestFollowRepo) UpdateFollowStatus(context.Context, int64, bool) error { return nil }
func (r *feedTestFollowRepo) GetFollowsByCondition(context.Context, map[string]interface{}) ([]FollowData, error) {
	return nil, nil
}
func (r *feedTestFollowRepo) CountFollowsByCondition(context.Context, map[string]interface{}) (int64, error) {
	return 0, nil
}
func (r *feedTestFollowRepo) CountFollowing(context.Context, int64) (int64, error) { return 0, nil }
func (r *feedTestFollowRepo) CountFollower(context.Context, int64) (int64, error)  { return 0, nil }
func (r *feedTestFollowRepo) BatchGetFollowing(context.Context, int64, []int64) ([]FollowData, error) {
	return nil, nil
}
func (r *feedTestFollowRepo) ListRelations(context.Context, FollowListQuery) ([]FollowData, int64, error) {
	return nil, 0, nil
}
func (r *feedTestFollowRepo) ListFollowing(context.Context, string, int32, *PageStats) ([]string, error) {
	return nil, nil
}
func (r *feedTestFollowRepo) GetFollowers(context.Context, string) ([]string, error) {
	return nil, nil
}
func (r *feedTestFollowRepo) GetFollowersPaginated(context.Context, string, int, int) ([]string, int64, error) {
	return nil, 0, nil
}
func (r *feedTestFollowRepo) CountFollowers(context.Context, string) (int64, error) { return 0, nil }

type feedTestProducer struct{}

func (p feedTestProducer) SendMessage(string, []byte, []byte) error { return nil }

func newFeedTestUsecase(t *testing.T, repo *feedTestVideoRepo) (*FeedUsecase, *redis.Client) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	uc := NewFeedUsecase(repo, &feedTestFollowRepo{}, client, feedTestProducer{}, NewRecentViewedManager(client), log.DefaultLogger)
	t.Cleanup(func() {
		uc.Close()
		_ = client.Close()
	})
	return uc, client
}

func TestFeedRecommendFallsBackToLatestAndCalculatesNextTime(t *testing.T) {
	now := time.Now()
	repo := &feedTestVideoRepo{latest: []*Video{
		{Id: 101, Author: &Author{Id: 1}, UploadTime: now.Add(-1 * time.Hour)},
		{Id: 102, Author: &Author{Id: 2}, UploadTime: now.Add(-2 * time.Hour)},
	}}
	uc, _ := newFeedTestUsecase(t, repo)

	result, err := uc.GetFeed(context.Background(), &FeedQuery{
		UserID:     "0",
		LatestTime: now.Unix(),
		PageSize:   10,
		FeedType:   1,
	})
	if err != nil {
		t.Fatalf("GetFeed() error = %v", err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result.Items))
	}
	if result.Items[0].VideoID != "101" || result.Items[1].VideoID != "102" {
		t.Fatalf("unexpected item order: %#v", result.Items)
	}
	wantNext := now.Add(-2*time.Hour).Unix() - 1
	if result.NextTime != wantNext {
		t.Fatalf("NextTime = %d, want %d", result.NextTime, wantNext)
	}
}

func TestFeedFiltersRecentViewedForLoggedInUser(t *testing.T) {
	now := time.Now()
	repo := &feedTestVideoRepo{latest: []*Video{
		{Id: 201, Author: &Author{Id: 1}, UploadTime: now.Add(-1 * time.Hour)},
		{Id: 202, Author: &Author{Id: 1}, UploadTime: now.Add(-2 * time.Hour)},
	}}
	uc, client := newFeedTestUsecase(t, repo)
	if err := NewRecentViewedManager(client).Add(context.Background(), "7", "201"); err != nil {
		t.Fatalf("mark recent viewed: %v", err)
	}

	result, err := uc.GetFeed(context.Background(), &FeedQuery{
		UserID:     "7",
		LatestTime: now.Unix(),
		PageSize:   10,
		FeedType:   1,
	})
	if err != nil {
		t.Fatalf("GetFeed() error = %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].VideoID != "202" {
		t.Fatalf("expected only video 202, got %#v", result.Items)
	}
}

func TestFollowingFeedForGuestFallsBackToRecommend(t *testing.T) {
	now := time.Now()
	repo := &feedTestVideoRepo{latest: []*Video{
		{Id: 401, Author: &Author{Id: 1}, UploadTime: now.Add(-1 * time.Hour)},
	}}
	uc, _ := newFeedTestUsecase(t, repo)

	result, err := uc.GetFeed(context.Background(), &FeedQuery{
		UserID:     "0",
		LatestTime: now.Unix(),
		PageSize:   10,
		FeedType:   0,
	})
	if err != nil {
		t.Fatalf("GetFeed() error = %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].VideoID != "401" {
		t.Fatalf("expected guest following feed to fallback to latest recommend, got %#v", result.Items)
	}
}

func TestMergeAndDeduplicateKeepsFirstSliceOrder(t *testing.T) {
	uc := &FeedUsecase{}
	items := uc.mergeAndDeduplicate(
		[]*FeedItem{{VideoID: "1"}, nil, {VideoID: "2"}},
		[]*FeedItem{{VideoID: "2"}, {VideoID: ""}, {VideoID: "3"}},
	)
	got := make([]string, 0, len(items))
	for _, item := range items {
		got = append(got, item.VideoID)
	}
	want := []string{"1", "2", "3"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestCalculateNextTimeSkipsNilItems(t *testing.T) {
	uc := &FeedUsecase{}
	got := uc.calculateNextTime([]*FeedItem{
		nil,
		{VideoID: "1", Timestamp: 100},
		{VideoID: "2", Timestamp: 80},
	})
	if got != 79 {
		t.Fatalf("NextTime = %d, want 79", got)
	}
}

func TestHotPoolReturnsStoredOrder(t *testing.T) {
	now := time.Now().Unix()
	repo := &feedTestVideoRepo{}
	uc, client := newFeedTestUsecase(t, repo)
	for i, score := range []float64{10, 30, 20} {
		videoID := strconv.Itoa(300 + i)
		uc.hotPool.AddVideo(context.Background(), videoID, "1", now+int64(i))
		if err := client.ZAdd(context.Background(), "feed:hot:pool", redis.Z{
			Score:  score,
			Member: videoID + ":1:" + strconv.FormatInt(now+int64(i), 10),
		}).Err(); err != nil {
			t.Fatalf("seed hot pool: %v", err)
		}
	}

	result, err := uc.GetFeed(context.Background(), &FeedQuery{
		UserID:     "0",
		LatestTime: now,
		PageSize:   2,
		FeedType:   2,
	})
	if err != nil {
		t.Fatalf("GetFeed() error = %v", err)
	}
	if len(result.Items) != 2 || result.Items[0].VideoID != "301" || result.Items[1].VideoID != "302" {
		t.Fatalf("unexpected hot order: %#v", result.Items)
	}
}
