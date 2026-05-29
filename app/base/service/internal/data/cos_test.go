package data

import (
	"context"
	"io"
	"net/url"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
)

func TestJoinPublicURL(t *testing.T) {
	got := joinPublicURL("https://cdn.example.com/media/", "/public/a b.jpg")
	want := "https://cdn.example.com/media/public/a%20b.jpg"
	if got != want {
		t.Fatalf("joinPublicURL() = %q, want %q", got, want)
	}
}

func TestIsPublicMediaObject(t *testing.T) {
	if !isPublicMediaObject("public/123.webp") {
		t.Fatal("expected webp to be public media")
	}
	if isPublicMediaObject("public/123.pdf") {
		t.Fatal("expected pdf to stay on fallback storage")
	}
}

func TestNewCOSConfigFromEnvRequiresAllValues(t *testing.T) {
	t.Setenv("COS_SECRET_ID", "")
	t.Setenv("COS_SECRET_KEY", "")
	t.Setenv("COS_REGION", "")
	t.Setenv("COS_BUCKET", "")
	t.Setenv("COS_PUBLIC_CDN_BASE_URL", "")

	if _, err := newCOSConfigFromEnv(); err == nil {
		t.Fatal("expected missing COS config error")
	}
}

func TestNewCOSConfigFromEnvRequiresHTTPSCDN(t *testing.T) {
	t.Setenv("COS_SECRET_ID", "sid")
	t.Setenv("COS_SECRET_KEY", "skey")
	t.Setenv("COS_REGION", "ap-guangzhou")
	t.Setenv("COS_BUCKET", "campus-1234567890")
	t.Setenv("COS_PUBLIC_CDN_BASE_URL", "http://cdn.example.com")

	if _, err := newCOSConfigFromEnv(); err == nil {
		t.Fatal("expected http CDN URL to be rejected")
	}
}

func TestNewCOSConfigFromEnv(t *testing.T) {
	t.Setenv("COS_SECRET_ID", "sid")
	t.Setenv("COS_SECRET_KEY", "skey")
	t.Setenv("COS_REGION", "ap-guangzhou")
	t.Setenv("COS_BUCKET", "campus-1234567890")
	t.Setenv("COS_PUBLIC_CDN_BASE_URL", "https://cdn.example.com/")

	cfg, err := newCOSConfigFromEnv()
	if err != nil {
		t.Fatalf("newCOSConfigFromEnv() error = %v", err)
	}
	if cfg.PublicBaseURL != "https://cdn.example.com" {
		t.Fatalf("PublicBaseURL = %q", cfg.PublicBaseURL)
	}
}

func TestCOSRepoPreSignPutURL(t *testing.T) {
	t.Setenv("COS_SECRET_ID", "sid")
	t.Setenv("COS_SECRET_KEY", "skey")
	t.Setenv("COS_REGION", "ap-guangzhou")
	t.Setenv("COS_BUCKET", "campus-1234567890")
	t.Setenv("COS_PUBLIC_CDN_BASE_URL", "https://cdn.example.com")

	repo, err := NewCOSRepoFromEnv(log.NewStdLogger(io.Discard))
	if err != nil {
		t.Fatalf("NewCOSRepoFromEnv() error = %v", err)
	}
	got, err := repo.PreSignPutUrl(context.Background(), "campus", "public/123.jpg", 3600)
	if err != nil {
		t.Fatalf("PreSignPutUrl() error = %v", err)
	}
	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse presigned URL: %v", err)
	}
	if parsed.Scheme != "https" || parsed.Host != "campus-1234567890.cos.ap-guangzhou.myqcloud.com" {
		t.Fatalf("unexpected COS upload host: %s://%s", parsed.Scheme, parsed.Host)
	}
	if parsed.Path != "/public/123.jpg" {
		t.Fatalf("unexpected object path: %q", parsed.Path)
	}
	if parsed.Query().Get("q-signature") == "" {
		t.Fatal("expected COS signature in query")
	}
}
