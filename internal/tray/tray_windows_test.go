//go:build windows

package tray

import (
	"strings"
	"testing"
)

func TestStringToUTF16Fixed(t *testing.T) {
	longStr := strings.Repeat("a", 300)

	arr64 := stringToUTF16Fixed64(longStr)
	if arr64[63] != 0 {
		t.Fatalf("expected arr64[63] to be null terminator, got %d", arr64[63])
	}
	if arr64[0] != 'a' || arr64[62] != 'a' {
		t.Fatalf("expected arr64[0..62] to be 'a'")
	}

	arr256 := stringToUTF16Fixed256(longStr)
	if arr256[255] != 0 {
		t.Fatalf("expected arr256[255] to be null terminator, got %d", arr256[255])
	}
	if arr256[0] != 'a' || arr256[254] != 'a' {
		t.Fatalf("expected arr256[0..254] to be 'a'")
	}

	shortStr := "hello"
	shortArr64 := stringToUTF16Fixed64(shortStr)
	if shortArr64[5] != 0 {
		t.Fatalf("expected null terminator at index 5")
	}
}
