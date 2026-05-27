package biz

import "testing"

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
