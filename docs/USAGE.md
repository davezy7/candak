# Candak Usage & Developer Guide

This guide provides instructions for building, running, extending, and interacting with **Candak**.

---

## 1. Quick Start

### Prerequisites
- **Go 1.20+** (tested with Go 1.26+)
- **Node.js 18+** & **npm**
- **Wails v2 CLI**:
  ```bash
  go install github.com/wailsapp/wails/v2/cmd/wails@latest
  ```

### Development Mode (Hot-Reload)
Run live-reloading for both Go code and Svelte UI:
```bash
wails dev
```
- Edits in `frontend/src/*` trigger instant Vite HMR.
- Edits in `*.go` automatically recompile and restart the desktop window.

### Production Build
Create an optimized native binary:
```bash
wails build
```
Outputs:
- **macOS**: `build/bin/candak.app`
- **Windows**: `build/bin/candak.exe`
- **Linux**: `build/bin/candak`

---

## 2. Testing

Run all unit and integration tests:

```bash
# Test all packages
go test -v ./...

# Test specific domain modules
go test -v ./internal/storage/sqlite
go test -v ./internal/planner
go test -v ./internal/prober
go test -v ./internal/downloader
```

---

## 3. IPC API Reference (Go to Svelte)

All public methods on [`App`](file:///Users/diksha/Documents/candak/app.go) are automatically bound into JavaScript and available via `@wails/go/main/App`:

### `ProbeURL(url string) (*models.ProbeResult, error)`
Inspects a remote target URL and extracts headers without downloading the payload.
```javascript
import { ProbeURL } from '@wails/go/main/App.js';

const result = await ProbeURL("https://example.com/bigfile.zip");
console.log(result.filename, result.totalSize, result.resumable);
```

### `AddDownload(req models.AddDownloadRequest) (*models.Download, error)`
Registers and starts a multi-threaded download.
```javascript
import { AddDownload } from '@wails/go/main/App.js';

const dl = await AddDownload({
  url: "https://example.com/bigfile.zip",
  filename: "custom_name.zip", // Optional (auto-detected if omitted)
  saveDir: "/Users/username/Downloads",
  connections: 8 // Number of concurrent chunks (1-16)
});
```

### `PauseDownload(id string) error`
Pauses streaming chunks and flushes offsets to SQLite.
```javascript
import { PauseDownload } from '@wails/go/main/App.js';

await PauseDownload("dl_123");
```

### `ResumeDownload(id string) error`
Restarts an interrupted or paused download from the exact SQLite byte offsets.
```javascript
import { ResumeDownload } from '@wails/go/main/App.js';

await ResumeDownload("dl_123");
```

### `CancelDownload(id string, deleteFile bool) error`
Aborts download, removes DB records, and optionally deletes incomplete file on disk.
```javascript
import { CancelDownload } from '@wails/go/main/App.js';

await CancelDownload("dl_123", true); // true = delete file from disk
```

### `GetDownloads() ([]*models.Download, error)`
Returns all downloads currently recorded in SQLite.
```javascript
import { GetDownloads } from '@wails/go/main/App.js';

const all = await GetDownloads();
```

### `GetURLHistory() ([]*models.URLHistoryItem, error)`
Returns the most recently downloaded URLs and frequency counts.
```javascript
import { GetURLHistory } from '@wails/go/main/App.js';

const history = await GetURLHistory();
```

### `OpenFile(filePath string) error`
Opens the downloaded file using the default OS viewer (Preview, VLC, Acrobat, etc.).
```javascript
import { OpenFile } from '@wails/go/main/App.js';

await OpenFile("/path/to/download.pdf");
```

### `ShowInFolder(filePath string) error`
Reveals and highlights the file inside macOS Finder, Windows Explorer, or Linux File Manager.
```javascript
import { ShowInFolder } from '@wails/go/main/App.js';

await ShowInFolder("/path/to/download.pdf");
```

---

## 4. Real-Time Events (Backend to Frontend)

Subscribe to real-time events using the Wails runtime:

```javascript
import { EventsOn } from '@wails/runtime/runtime.js';

// High-frequency progress updates (300ms intervals)
const unlistenProgress = EventsOn('download:progress', (event) => {
  console.log(`Download: ${event.downloadId}, Progress: ${event.progress}%, Speed: ${event.speed} B/s`);
});

// Lifecycle status transitions (queued, downloading, paused, completed, error)
const unlistenStatus = EventsOn('download:status_change', (download) => {
  console.log(`Status changed for ${download.id} to ${download.status}`);
});

// Download deleted/cancelled
const unlistenDeleted = EventsOn('download:deleted', (downloadId) => {
  console.log(`Download ${downloadId} deleted`);
});
```

---

## 5. Extending the Application (SOLID Examples)

### Implementing a Custom Storage Engine
To use another database (e.g. BadgerDB or PostgreSQL), create a struct satisfying [`storage.Repository`](file:///Users/diksha/Documents/candak/internal/storage/repository.go):

```go
type CustomRepository struct { /* ... */ }

func (c *CustomRepository) InsertDownload(ctx context.Context, dl *models.Download) error { ... }
func (c *CustomRepository) SaveChunks(ctx context.Context, chunks []*models.Chunk) error { ... }
// ... implement remaining methods ...
```

Then in `app.go`:
```go
repo := custom.NewRepository(...)
a.manager = downloader.NewManager(downloader.Config{
    Repository: repo,
    // ...
})
```

No changes in `downloader.Manager` or the UI are required (Open/Closed Principle).

### Implementing an Alternative Chunk Planner
If you want dynamic or adaptive chunk partitioning based on bandwidth:
```go
type AdaptivePlanner struct{}

func (a *AdaptivePlanner) Plan(downloadID string, totalSize int64, resumable bool, connections int) []*models.Chunk {
    // Custom adaptive partition logic
}
```
Inject it into `downloader.NewManager(downloader.Config{ ChunkPlanner: &AdaptivePlanner{} })`.
