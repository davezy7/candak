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

func TestHTTPProberHeadFallback(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			// Simulate server rejecting HEAD with 405 Method Not Allowed
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.Header.Get("Range") == "bytes=0-0" {
			w.Header().Set("Content-Range", "bytes 0-0/10485760")
			w.Header().Set("Content-Length", "1")
			w.WriteHeader(http.StatusPartialContent)
			_, _ = w.Write([]byte("X"))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	p := NewHTTPProber(WithHTTPClient(ts.Client()))
	res, err := p.Probe(context.Background(), ts.URL+"/largefile.iso")
	if err != nil {
		t.Fatalf("Probe error: %v", err)
	}

	if res.TotalSize != 10485760 {
		t.Errorf("Expected size from Content-Range 10485760, got %d", res.TotalSize)
	}
	if !res.Resumable {
		t.Errorf("Expected resumable=true from Content-Range, got false")
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
		{"..", "download.bin"},
		{".", "download.bin"},
		{"../../../etc/passwd", "download.bin"},
		{"..secret", "download.bin"},
	}

	for _, tt := range tests {
		got := SanitizeFilename(tt.input)
		if got != tt.expected {
			t.Errorf("SanitizeFilename(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}

func TestBatchProbe(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/file1.zip":
			w.Header().Set("Content-Length", "1000")
			w.WriteHeader(http.StatusOK)
		case "/file2.tar":
			w.Header().Set("Content-Length", "2000")
			w.WriteHeader(http.StatusOK)
		case "/error404":
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	p := NewHTTPProber(WithHTTPClient(server.Client()))
	urls := []string{
		server.URL + "/file1.zip",
		server.URL + "/error404",
		server.URL + "/file2.tar",
		"http://invalid.local.nonexistent.domain/file.bin",
	}

	results, err := BatchProbe(context.Background(), p, urls, 4)
	if err != nil {
		t.Fatalf("BatchProbe failed: %v", err)
	}

	if len(results) != 4 {
		t.Fatalf("Expected 4 results, got %d", len(results))
	}

	if results[0].Error != "" || results[0].Filename != "file1.zip" || results[0].TotalSize != 1000 {
		t.Errorf("Unexpected result[0]: %+v", results[0])
	}

	if results[1].Error == "" {
		t.Errorf("Expected result[1] to have error for 404, got nil error: %+v", results[1])
	}

	if results[2].Error != "" || results[2].Filename != "file2.tar" || results[2].TotalSize != 2000 {
		t.Errorf("Unexpected result[2]: %+v", results[2])
	}

	if results[3].Error == "" {
		t.Errorf("Expected result[3] to have error for nonexistent domain, got nil: %+v", results[3])
	}
}
