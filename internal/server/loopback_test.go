package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"candak/internal/models"
)

func TestLoopbackServer_Origins(t *testing.T) {
	tests := []struct {
		origin  string
		allowed bool
	}{
		{"", true},
		{"chrome-extension://abcdefghijklmnop", true},
		{"moz-extension://12345678-1234-1234-1234-123456789abc", true},
		{"http://127.0.0.1:3000", true},
		{"http://localhost:5173", true},
		{"https://malicious-website.com", false},
		{"http://attacker.org:8080", false},
	}

	for _, tc := range tests {
		if got := isAllowedOrigin(tc.origin); got != tc.allowed {
			t.Errorf("isAllowedOrigin(%q) = %v; want %v", tc.origin, got, tc.allowed)
		}
	}
}

func TestLoopbackServer_Health(t *testing.T) {
	srv := NewLoopbackServer(Config{Port: 41920})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Origin", "chrome-extension://test")
	w := httptest.NewRecorder()

	srv.handleHealth(w, req)
	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var data map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Fatalf("failed to decode json: %v", err)
	}
	if data["status"] != "ok" || data["app"] != "candak" {
		t.Fatalf("unexpected health payload: %+v", data)
	}
}

func TestLoopbackServer_GrabUnauthorizedOrigin(t *testing.T) {
	srv := NewLoopbackServer(Config{Port: 41920})
	req := httptest.NewRequest(http.MethodPost, "/api/grab", bytes.NewBufferString(`{"url":"https://example.com/file.zip"}`))
	req.Header.Set("Origin", "https://evil.com")
	w := httptest.NewRecorder()

	srv.handleGrab(w, req)
	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for evil.com origin, got %d", resp.StatusCode)
	}
}

func TestLoopbackServer_GrabSuccess(t *testing.T) {
	var captured models.GrabRequest
	srv := NewLoopbackServer(Config{
		Port: 41920,
		OnGrab: func(req models.GrabRequest) error {
			captured = req
			return nil
		},
	})

	body := `{"url":"https://example.com/archive.tar.gz","filename":"archive.tar.gz","headers":{"Cookie":"session=abc"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/grab", bytes.NewBufferString(body))
	req.Header.Set("Origin", "chrome-extension://candak-extension")
	w := httptest.NewRecorder()

	srv.handleGrab(w, req)
	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	if captured.URL != "https://example.com/archive.tar.gz" {
		t.Fatalf("expected url, got %q", captured.URL)
	}
	if captured.Headers["Cookie"] != "session=abc" {
		t.Fatalf("expected cookie header, got %+v", captured.Headers)
	}
}
