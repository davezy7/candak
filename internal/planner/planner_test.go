package planner

import (
	"testing"
)

func TestDefaultChunkPlanner(t *testing.T) {
	p := NewDefaultChunkPlanner()

	// 1. Resumable large file split across 4 connections
	totalSize := int64(100 * 1024 * 1024) // 100MB
	chunks := p.Plan("dl1", totalSize, true, 4)

	if len(chunks) != 4 {
		t.Fatalf("Expected 4 chunks, got %d", len(chunks))
	}

	// Verify exact byte coverage without gaps or overlaps
	var lastOffset int64 = 0
	for i, c := range chunks {
		if c.StartOffset != lastOffset {
			t.Errorf("Chunk %d gap detected: start %d != last %d", i, c.StartOffset, lastOffset)
		}
		if c.EndOffset < c.StartOffset {
			t.Errorf("Chunk %d inverted offsets: start %d, end %d", i, c.StartOffset, c.EndOffset)
		}
		lastOffset = c.EndOffset + 1
	}
	if lastOffset != totalSize {
		t.Errorf("Total bytes covered %d != totalSize %d", lastOffset, totalSize)
	}

	// 2. Non-resumable file
	chunksNonRes := p.Plan("dl2", totalSize, false, 8)
	if len(chunksNonRes) != 1 {
		t.Errorf("Expected 1 chunk for non-resumable download, got %d", len(chunksNonRes))
	}
}
