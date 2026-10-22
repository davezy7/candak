package models

type DownloadStatus string

const (
	StatusQueued      DownloadStatus = "queued"
	StatusDownloading DownloadStatus = "downloading"
	StatusPaused      DownloadStatus = "paused"
	StatusCompleted   DownloadStatus = "completed"
	StatusError       DownloadStatus = "error"
	StatusCancelled   DownloadStatus = "cancelled"
)

type ChunkStatus string

const (
	ChunkStatusPending     ChunkStatus = "pending"
	ChunkStatusDownloading ChunkStatus = "downloading"
	ChunkStatusCompleted   ChunkStatus = "completed"
	ChunkStatusError       ChunkStatus = "error"
)

type Chunk struct {
	ID            string      `json:"id"`
	DownloadID    string      `json:"downloadId"`
	ChunkIndex    int         `json:"chunkIndex"`
	StartOffset   int64       `json:"startOffset"`
	EndOffset     int64       `json:"endOffset"`
	CurrentOffset int64       `json:"currentOffset"`
	Status        ChunkStatus `json:"status"`
}

type Download struct {
	ID             string         `json:"id"`
	URL            string         `json:"url"`
	Filename       string         `json:"filename"`
	SavePath       string         `json:"savePath"`
	TotalSize      int64          `json:"totalSize"`
	DownloadedSize int64          `json:"downloadedSize"`
	Status         DownloadStatus `json:"status"`
	Resumable      bool           `json:"resumable"`
	TotalChunks    int            `json:"totalChunks"`
	Speed          int64          `json:"speed"` // bytes per second
	ETA            int64          `json:"eta"`   // estimated seconds remaining
	ErrorMessage   string         `json:"errorMessage"`
	CreatedAt      string         `json:"createdAt"`
	UpdatedAt      string         `json:"updatedAt"`
	Chunks         []*Chunk       `json:"chunks,omitempty"`
}

type URLHistoryItem struct {
	ID            int64  `json:"id"`
	URL           string `json:"url"`
	Filename      string `json:"filename"`
	LastUsedAt    string `json:"lastUsedAt"`
	DownloadCount int    `json:"downloadCount"`
}

type ProbeResult struct {
	URL         string `json:"url"`
	Filename    string `json:"filename"`
	TotalSize   int64  `json:"totalSize"`
	Resumable   bool   `json:"resumable"`
	ContentType string `json:"contentType"`
}

type AddDownloadRequest struct {
	URL         string `json:"url"`
	Filename    string `json:"filename"`
	SaveDir     string `json:"saveDir"`
	Connections int    `json:"connections"`
}

type ProgressEvent struct {
	DownloadID     string         `json:"downloadId"`
	DownloadedSize int64          `json:"downloadedSize"`
	TotalSize      int64          `json:"totalSize"`
	Progress       float64        `json:"progress"` // 0.0 - 100.0
	Speed          int64          `json:"speed"`
	ETA            int64          `json:"eta"`
	Status         DownloadStatus `json:"status"`
	Chunks         []*Chunk       `json:"chunks,omitempty"`
}
