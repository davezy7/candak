package prober

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPProber(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Length", "2048")
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", `attachment; filename="archive.zip"`)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	p := NewHTTPProber(WithHTTPClient(ts.Client()))
	res, err := p.Probe(context.Background(), ts.URL+"/files/dummy")
	if err != nil {
		t.Fatalf("Probe error: %v", err)
	}

	if res.Filename != "archive.zip" {
		t.Errorf("Expected filename archive.zip, got %s", res.Filename)
	}
	if res.TotalSize != 2048 {
		t.Errorf("Expected size 2048, got %d", res.TotalSize)
	}
	if !res.Resumable {
		t.Errorf("Expected resumable=true, got false")
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"my/file:name*.zip", "my_file_name_.zip"},
		{"   ", "download.bin"},
		{"valid_name.tar.gz", "valid_name.tar.gz"},
	}

	for _, tt := range tests {
		got := SanitizeFilename(tt.input)
		if got != tt.expected {
			t.Errorf("SanitizeFilename(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}
