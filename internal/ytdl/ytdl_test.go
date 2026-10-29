package ytdl

import (
	"strings"
	"testing"
)

func TestIsYouTubeURL(t *testing.T) {
	valid := []string{
		"https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		"http://youtube.com/watch?v=dQw4w9WgXcQ&feature=shared",
		"https://youtu.be/dQw4w9WgXcQ",
		"https://m.youtube.com/watch?v=dQw4w9WgXcQ",
		"https://www.youtube.com/shorts/abc123xyz_-",
		"https://www.youtube.com/live/livestream123",
	}

	for _, u := range valid {
		if !IsYouTubeURL(u) {
			t.Errorf("expected IsYouTubeURL(%q) = true, got false", u)
		}
	}

	invalid := []string{
		"https://vimeo.com/123456",
		"https://example.com/video.mp4",
		"https://dailymotion.com/video/x7tgad0",
		"",
	}

	for _, u := range invalid {
		if IsYouTubeURL(u) {
			t.Errorf("expected IsYouTubeURL(%q) = false, got true", u)
		}
	}
}

func TestCleanVideoTitle(t *testing.T) {
	raw := "Rick Astley - Never Gonna Give You Up (Official Music Video) [4K] /:*?\"<>|"
	cleaned := CleanVideoTitle(raw)
	if strings.ContainsAny(cleaned, `\/:*?"<>|`) {
		t.Errorf("CleanVideoTitle contained invalid chars: %q", cleaned)
	}

	long := strings.Repeat("A", 200)
	if len(CleanVideoTitle(long)) > 120 {
		t.Errorf("CleanVideoTitle did not cap length at 120 chars")
	}

	if CleanVideoTitle("   ") != "youtube_video" {
		t.Errorf("expected fallback for empty title")
	}
}

func TestFetchMetadata(t *testing.T) {
	meta, err := FetchMetadata(t.Context(), "https://www.youtube.com/watch?v=dQw4w9WgXcQ")
	if err != nil {
		t.Fatalf("FetchMetadata failed: %v", err)
	}
	t.Logf("Video title: %s", meta.Title)
	for _, opt := range meta.Options {
		t.Logf("Option: %s, size: %d, id: %s", opt.Resolution, opt.Filesize, opt.FormatID)
	}
}
