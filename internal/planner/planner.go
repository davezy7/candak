package planner

import (
	"fmt"

	"candak/internal/models"
)

// ChunkPlanner defines the strategy for calculating chunk boundaries across a file.
type ChunkPlanner interface {
	// Plan creates the slice of Chunks based on size, resumability, and connection constraints.
	Plan(downloadID string, totalSize int64, resumable bool, connections int) []*models.Chunk
}

// DefaultChunkPlanner partitions bytes evenly across the requested connection count.
type DefaultChunkPlanner struct {
	MinChunkSize int64 // Minimum chunk size in bytes to justify splitting (e.g. 512KB)
}

// NewDefaultChunkPlanner creates a DefaultChunkPlanner with standard defaults.
func NewDefaultChunkPlanner() *DefaultChunkPlanner {
	return &DefaultChunkPlanner{
		MinChunkSize: 512 * 1024, // 512 KB
	}
}

// Plan partitions totalSize into balanced, non-overlapping contiguous chunks.
func (p *DefaultChunkPlanner) Plan(downloadID string, totalSize int64, resumable bool, connections int) []*models.Chunk {
	if connections <= 0 {
		connections = 4
	}

	// Single stream if not resumable or size is unknown
	if !resumable || totalSize <= 0 {
		end := totalSize - 1
		if end < 0 {
			end = 0
		}
		return []*models.Chunk{
			{
				ID:            fmt.Sprintf("%s_c0", downloadID),
				DownloadID:    downloadID,
				ChunkIndex:    0,
				StartOffset:   0,
				EndOffset:     end,
				CurrentOffset: 0,
				Status:        models.ChunkStatusPending,
			},
		}
	}

	// Adjust connections if totalSize is too small to benefit from multi-part downloading
	if p.MinChunkSize > 0 && totalSize < p.MinChunkSize*int64(connections) {
		feasible := int(totalSize / p.MinChunkSize)
		if feasible < 1 {
			feasible = 1
		}
		if feasible < connections {
			connections = feasible
		}
	}

	if connections <= 1 {
		return []*models.Chunk{
			{
				ID:            fmt.Sprintf("%s_c0", downloadID),
				DownloadID:    downloadID,
				ChunkIndex:    0,
				StartOffset:   0,
				EndOffset:     totalSize - 1,
				CurrentOffset: 0,
				Status:        models.ChunkStatusPending,
			},
		}
	}

	chunks := make([]*models.Chunk, connections)
	chunkSize := totalSize / int64(connections)

	for i := 0; i < connections; i++ {
		start := int64(i) * chunkSize
		end := start + chunkSize - 1
		if i == connections-1 {
			end = totalSize - 1 // Last chunk absorbs any remainder
		}

		chunks[i] = &models.Chunk{
			ID:            fmt.Sprintf("%s_c%d", downloadID, i),
			DownloadID:    downloadID,
			ChunkIndex:    i,
			StartOffset:   start,
			EndOffset:     end,
			CurrentOffset: start,
			Status:        models.ChunkStatusPending,
		}
	}

	return chunks
}
