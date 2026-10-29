package models

// DownloadStatus represents the lifecycle state of a download.
type DownloadStatus string

const (
	// StatusQueued indicates the download is registered and awaiting execution.
	StatusQueued DownloadStatus = "queued"
	// StatusDownloading indicates the download is actively receiving data chunks.
	StatusDownloading DownloadStatus = "downloading"
	// StatusPaused indicates the download was suspended and can be resumed.
	StatusPaused DownloadStatus = "paused"
	// StatusCompleted indicates all chunks have been received and verified.
	StatusCompleted DownloadStatus = "completed"
	// StatusError indicates a non-recoverable error halted the download.
	StatusError DownloadStatus = "error"
	// StatusCancelled indicates the user cancelled and aborted the download.
	StatusCancelled DownloadStatus = "cancelled"
)

// ChunkStatus represents the state of an individual download chunk partition.
type ChunkStatus string

const (
	// ChunkStatusPending indicates the chunk has not started downloading.
	ChunkStatusPending ChunkStatus = "pending"
	// ChunkStatusDownloading indicates the chunk is currently streaming bytes.
	ChunkStatusDownloading ChunkStatus = "downloading"
	// ChunkStatusCompleted indicates the chunk reached its end offset.
	ChunkStatusCompleted ChunkStatus = "completed"
	// ChunkStatusError indicates the chunk worker encountered an error.
	ChunkStatusError ChunkStatus = "error"
)

// Chunk represents a contiguous byte range of a download task.
type Chunk struct {
	ID            string      `json:"id"`
	DownloadID    string      `json:"downloadId"`
	ChunkIndex    int         `json:"chunkIndex"`
	StartOffset   int64       `json:"startOffset"`
	EndOffset     int64       `json:"endOffset"`
	CurrentOffset int64       `json:"currentOffset"`
	Status        ChunkStatus `json:"status"`
}

// Download represents a top-level file download entity.
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
	Speed          int64             `json:"speed"` // Bytes per second
	ETA            int64             `json:"eta"`   // Estimated remaining seconds
	ErrorMessage   string            `json:"errorMessage"`
	Headers        map[string]string `json:"headers,omitempty"`
	CreatedAt      string            `json:"createdAt"`
	UpdatedAt      string            `json:"updatedAt"`
	Chunks         []*Chunk          `json:"chunks,omitempty"`
}

// URLHistoryItem represents a recorded URL entry in the SQLite history log.
type URLHistoryItem struct {
	ID            int64  `json:"id"`
	URL           string `json:"url"`
	Filename      string `json:"filename"`
	LastUsedAt    string `json:"lastUsedAt"`
	DownloadCount int    `json:"downloadCount"`
}

// ProbeResult contains metadata obtained from an HTTP inspection request.
type ProbeResult struct {
	URL         string            `json:"url"`
	Filename    string            `json:"filename"`
	TotalSize   int64             `json:"totalSize"`
	Resumable   bool              `json:"resumable"`
	ContentType string            `json:"contentType"`
	Headers     map[string]string `json:"headers,omitempty"`
	Error       string            `json:"error,omitempty"`
}

// AddDownloadRequest holds the parameters for initializing a new download task.
type AddDownloadRequest struct {
	URL         string            `json:"url"`
	Filename    string            `json:"filename"`
	SaveDir     string            `json:"saveDir"`
	Connections int               `json:"connections"`
	Headers     map[string]string `json:"headers,omitempty"`
	TotalSize   int64             `json:"totalSize,omitempty"`
	Resumable   bool              `json:"resumable,omitempty"`
}

// DownloadRequest is an alias for AddDownloadRequest.
type DownloadRequest = AddDownloadRequest

// GrabRequest represents a download request forwarded from a browser extension or external caller.
type GrabRequest struct {
	URL      string            `json:"url"`
	Filename string            `json:"filename,omitempty"`
	Headers  map[string]string `json:"headers,omitempty"`
}

// ProgressEvent is emitted via the event bus to report real-time progress.
type ProgressEvent struct {
	DownloadID     string         `json:"downloadId"`
	DownloadedSize int64          `json:"downloadedSize"`
	TotalSize      int64          `json:"totalSize"`
	Progress       float64        `json:"progress"` // 0.0 - 100.0%
	Speed          int64          `json:"speed"`    // Bytes/sec
	ETA            int64          `json:"eta"`      // Seconds remaining
	Status         DownloadStatus `json:"status"`
	Chunks         []*Chunk       `json:"chunks,omitempty"`
}

// StartupConfig represents the autostart and launch minimized preferences.
type StartupConfig struct {
	Enabled   bool `json:"enabled"`
	Minimized bool `json:"minimized"`
}
