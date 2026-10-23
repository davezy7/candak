package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureExtensionFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "candak-ext-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	app := NewApp()
	targetDir := filepath.Join(tempDir, "browser-extension")
	if err := app.ensureExtensionFiles(targetDir); err != nil {
		t.Fatalf("ensureExtensionFiles failed: %v", err)
	}

	manifestPath := filepath.Join(targetDir, "manifest.json")
	if _, err := os.Stat(manifestPath); err != nil {
		t.Errorf("manifest.json not extracted: %v", err)
	}

	dir := app.GetExtensionDir()
	if _, err := os.Stat(filepath.Join(dir, "manifest.json")); err != nil {
		t.Errorf("GetExtensionDir does not contain manifest.json: %v", err)
	}
}
