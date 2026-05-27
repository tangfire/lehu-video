package data

import (
	"strings"
	"testing"

	"lehu-video/app/videoApi/service/internal/biz"
)

func TestCampusPostOrderRecommendUsesDecayAndPinnedFirst(t *testing.T) {
	order := campusPostOrder(biz.CampusPostSortRecommend, false)
	for _, want := range []string{
		"campus_forum_post.is_pinned DESC",
		"campus_forum_post.sort_weight * 10",
		"IF(campus_forum_post.is_featured, 80, 0)",
		"IF(campus_forum_post.is_official, 30, 0)",
		"TIMESTAMPDIFF(HOUR, campus_forum_post.created_at, NOW())",
		"POW(",
		"campus_forum_post.like_count * 2",
		"campus_forum_post.comment_count * 4",
		"campus_forum_post.collected_count * 5",
	} {
		if !strings.Contains(order, want) {
			t.Fatalf("recommend order missing %q: %s", want, order)
		}
	}
}

func TestCampusPostOrderHotUsesDecayedInteraction(t *testing.T) {
	order := campusPostOrder(biz.CampusPostSortHot, false)
	for _, want := range []string{
		"campus_forum_post.is_pinned DESC",
		"campus_forum_post.is_featured DESC",
		"campus_forum_post.sort_weight DESC",
		"TIMESTAMPDIFF(HOUR, campus_forum_post.created_at, NOW())",
		"POW(",
	} {
		if !strings.Contains(order, want) {
			t.Fatalf("hot order missing %q: %s", want, order)
		}
	}
}

func TestCampusPostOrderCollectionsKeepCollectionTime(t *testing.T) {
	order := campusPostOrder(biz.CampusPostSortRecommend, true)
	if order != "c.updated_at DESC, c.id DESC" {
		t.Fatalf("collection order = %q", order)
	}
}

func TestCampusPostOrderNewIsPureLatest(t *testing.T) {
	order := campusPostOrder(biz.CampusPostSortNew, false)
	if order != "campus_forum_post.created_at DESC, campus_forum_post.id DESC" {
		t.Fatalf("new order = %q", order)
	}
}
