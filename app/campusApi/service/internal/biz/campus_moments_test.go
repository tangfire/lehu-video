package biz

import (
	"archive/zip"
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseMomentsDateUsesShanghaiDayRange(t *testing.T) {
	day, start, end, err := parseMomentsDate("2026-05-29")
	if err != nil {
		t.Fatalf("parseMomentsDate() error = %v", err)
	}
	if day != "2026-05-29" {
		t.Fatalf("day = %q, want 2026-05-29", day)
	}
	if start.Format("2006-01-02 15:04:05") != "2026-05-29 00:00:00" {
		t.Fatalf("start = %s", start)
	}
	if end.Sub(start) != 24*time.Hour {
		t.Fatalf("range = %s, want 24h", end.Sub(start))
	}
}

func TestValidateMomentsImageURLRequiresAllowlistedHost(t *testing.T) {
	t.Setenv("LEHU_STORAGE_PROVIDER", "cos")
	t.Setenv("COS_PUBLIC_CDN_BASE_URL", "https://cdn.example.com")
	t.Setenv("LEHU_ADMIN_MOMENTS_IMAGE_HOST_ALLOWLIST", "")

	if err := validateMomentsImageURL("https://cdn.example.com/campus/post.jpg"); err != nil {
		t.Fatalf("validate allowlisted url error = %v", err)
	}
	if err := validateMomentsImageURL("https://evil.example.com/post.jpg"); err == nil {
		t.Fatalf("validate non-allowlisted url should fail")
	}
	if err := validateMomentsImageURL("http://169.254.169.254/latest/meta-data"); err == nil {
		t.Fatalf("validate private metadata url should fail")
	}
}

func TestRewriteMomentsImageURLForContainerNetwork(t *testing.T) {
	t.Setenv("MINIO_PUBLIC_HOST_REWRITE", "localhost:19000=minio:9000")
	got := rewriteMomentsImageURL("http://localhost:19000/campus/public/1.jpg")
	want := "http://minio:9000/campus/public/1.jpg"
	if got != want {
		t.Fatalf("rewriteMomentsImageURL() = %q, want %q", got, want)
	}
}

func TestEncodeMockMomentsQRCodeReturnsPNG(t *testing.T) {
	data, err := encodeMockMomentsQRCode("id=123")
	if err != nil {
		t.Fatalf("encodeMockMomentsQRCode() error = %v", err)
	}
	img, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode mock qr error = %v", err)
	}
	if format != "png" || img.Width != 430 || img.Height != 430 {
		t.Fatalf("mock qr = %s %dx%d, want png 430x430", format, img.Width, img.Height)
	}
}

func TestWriteMomentsZipIncludesImagesAndCaption(t *testing.T) {
	dir := t.TempDir()
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	fillRect(img, img.Bounds(), color.White)
	imagePath := filepath.Join(dir, "ezai-moments-20260529-01.png")
	file, err := os.Create(imagePath)
	if err != nil {
		t.Fatalf("create image: %v", err)
	}
	if err := png.Encode(file, img); err != nil {
		t.Fatalf("encode image: %v", err)
	}
	_ = file.Close()

	out := &CampusMomentsPackageOutput{
		PackageID: "123",
		Date:      "2026-05-29",
		Caption:   "今日热帖",
		Posts: []*CampusMomentsPackagePost{
			{Slot: 1, PostID: "42"},
		},
	}
	if err := writeMomentsZip(dir, out); err != nil {
		t.Fatalf("writeMomentsZip() error = %v", err)
	}
	reader, err := zip.OpenReader(filepath.Join(dir, "ezai-moments-20260529-123.zip"))
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	defer reader.Close()
	seen := map[string]bool{}
	for _, file := range reader.File {
		seen[file.Name] = true
	}
	for _, name := range []string{"朋友圈文案.txt", "ezai-moments-20260529-01.png"} {
		if !seen[name] {
			t.Fatalf("zip missing %s", name)
		}
	}
}
