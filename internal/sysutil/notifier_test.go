package sysutil

import "testing"

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "Unknown size"},
		{-5, "Unknown size"},
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1048576, "1.0 MB"},
		{1572864, "1.5 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tc := range tests {
		if got := FormatBytes(tc.bytes); got != tc.expected {
			t.Errorf("FormatBytes(%d) = %q; want %q", tc.bytes, got, tc.expected)
		}
	}
}

func TestEncodeUTF16LEBase64(t *testing.T) {
	input := "Write-Host 'hello'"
	encoded := encodeUTF16LEBase64(input)
	if encoded == "" {
		t.Fatalf("expected non-empty base64 string")
	}
}
