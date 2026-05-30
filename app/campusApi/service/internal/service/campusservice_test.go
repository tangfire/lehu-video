package service

import (
	"encoding/json"
	"net/http"
	"testing"

	"lehu-video/app/campusApi/service/internal/biz"
)

func TestClientIPDoesNotTrustForwardedHeaderFromPublicRemote(t *testing.T) {
	t.Setenv("LEHU_TRUSTED_PROXY_CIDRS", "10.0.0.0/8")
	req := &http.Request{
		RemoteAddr: "203.0.113.10:4567",
		Header: http.Header{
			"X-Forwarded-For": []string{"198.51.100.8"},
		},
	}

	if got := clientIP(req); got != "203.0.113.10" {
		t.Fatalf("clientIP() = %q, want public remote address", got)
	}
}

func TestClientIPTrustsForwardedHeaderFromTrustedProxy(t *testing.T) {
	t.Setenv("LEHU_TRUSTED_PROXY_CIDRS", "172.16.0.0/12")
	req := &http.Request{
		RemoteAddr: "172.18.0.5:4567",
		Header: http.Header{
			"X-Forwarded-For": []string{"198.51.100.8, 172.18.0.1"},
		},
	}

	if got := clientIP(req); got != "198.51.100.8" {
		t.Fatalf("clientIP() = %q, want forwarded client IP", got)
	}
}

func TestClientIPTrustsRealIPFromTrustedProxy(t *testing.T) {
	t.Setenv("LEHU_TRUSTED_PROXY_CIDRS", "127.0.0.0/8")
	req := &http.Request{
		RemoteAddr: "127.0.0.1:4567",
		Header: http.Header{
			"X-Real-Ip": []string{"198.51.100.9"},
		},
	}

	if got := clientIP(req); got != "198.51.100.9" {
		t.Fatalf("clientIP() = %q, want real client IP", got)
	}
}

func TestClientIPHandlesIPv6RemoteAddr(t *testing.T) {
	req := &http.Request{
		RemoteAddr: "[2001:db8::1]:4567",
		Header: http.Header{
			"X-Forwarded-For": []string{"198.51.100.8"},
		},
	}

	if got := clientIP(req); got != "2001:db8::1" {
		t.Fatalf("clientIP() = %q, want IPv6 remote address", got)
	}
}

func TestParseRawInt64ListKeepsSnowflakePrecision(t *testing.T) {
	values := []json.RawMessage{
		json.RawMessage(`"2060350884549840896"`),
		json.RawMessage(`2060350884549840896`),
		json.RawMessage(`"2060350884549840896"`),
	}

	got, err := parseRawInt64List(values)
	if err != nil {
		t.Fatalf("parseRawInt64List() error = %v", err)
	}
	if len(got) != 1 || got[0] != 2060350884549840896 {
		t.Fatalf("parseRawInt64List() = %#v, want deduped exact snowflake id", got)
	}
}

func TestParseRawInt64ListRejectsUnsafeNumberForms(t *testing.T) {
	values := []json.RawMessage{json.RawMessage(`2.0603508845498409e18`)}

	if _, err := parseRawInt64List(values); err == nil {
		t.Fatalf("parseRawInt64List() error = nil, want error for exponent notation")
	}
}

func TestPostToMapAddsClientFriendlyPublishState(t *testing.T) {
	cases := []struct {
		name       string
		status     int32
		state      string
		label      string
		publicShow bool
	}{
		{name: "visible", status: biz.CampusAuditStatusVisible, state: "visible", label: "已发布", publicShow: true},
		{name: "pending", status: biz.CampusAuditStatusPending, state: "syncing", label: "同步中", publicShow: false},
		{name: "rejected", status: biz.CampusAuditStatusRejected, state: "needs_attention", label: "需修改", publicShow: false},
		{name: "hidden", status: biz.CampusAuditStatusDeleted, state: "hidden", label: "已隐藏", publicShow: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := postToMap(&biz.CampusForumPost{ID: 1, Status: tc.status})
			if got["publish_state"] != tc.state {
				t.Fatalf("publish_state = %v, want %s", got["publish_state"], tc.state)
			}
			if got["client_status_label"] != tc.label {
				t.Fatalf("client_status_label = %v, want %s", got["client_status_label"], tc.label)
			}
			if got["public_visible"] != tc.publicShow {
				t.Fatalf("public_visible = %v, want %v", got["public_visible"], tc.publicShow)
			}
		})
	}
}
