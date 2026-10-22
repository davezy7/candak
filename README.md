# Candak - Download Manager Desktop App

A high-performance, multi-threaded desktop download manager built with:
- **Go 1.26+**
- **Wails v2** (native OS desktop wrapper with zero Chromium bloat)
- **Svelte 5 + Tailwind CSS v4** (instant reactive UI with virtually no boilerplate)
- **modernc.org/sqlite** (Pure Go SQLite driver without CGO, tracking chunk offsets, download resumes, and URL history)

---

## Architecture Overview

```
candak/
├── app.go                      # Wails Go application layer (exposed methods to Svelte)
├── main.go                     # Entrypoint & desktop window configuration
├── wails.json                  # Wails build & dev configuration
├── go.mod / go.sum             # Go module dependencies
├── internal/
│   ├── models/models.go        # Data structures (Download, Chunk, ProgressEvent, etc.)
│   ├── db/
│   │   ├── db.go               # SQLite storage using modernc.org/sqlite
│   │   └── db_test.go          # SQLite unit tests
│   └── downloader/
│       ├── manager.go          # Multi-chunk concurrent downloader with resume support
│       └── manager_test.go     # End-to-end range download test
└── frontend/                   # Svelte + Tailwind CSS UI
    ├── package.json
    ├── vite.config.js          # Vite 8 with Tailwind CSS plugin & @wails alias
    ├── src/
    │   ├── App.svelte          # Main dashboard & live stats bar
    │   ├── main.js             # Svelte client entrypoint
    │   ├── style.css           # Tailwind v4 theme & custom scrollbar
    │   └── lib/
    │       ├── AddDownloadModal.svelte  # New download dialog with URL probing
    │       ├── ChunkVisualizer.svelte   # Visual chunk offsets progress bar
    │       ├── DownloadItem.svelte      # Download card with controls
    │       ├── HistoryModal.svelte      # SQLite URL history viewer
    │       └── formatters.js            # Size, speed, ETA, and date helpers
    └── wailsjs/                # Generated Go-to-JS bindings & TypeScript models
```

---

## Features

1. **Multi-Part / Concurrent Chunking**:
   - Analyzes target server `Accept-Ranges` and `Content-Length`.
   - Divides files into configurable concurrent chunks (1 to 16 threads).
   - Writes directly to disk using random-access file writing (`io.WriterAt`), avoiding temporary file concatenation.

2. **Resume & Chunk Offset Tracking (modernc.org/sqlite)**:
   - Stores byte-level chunk offsets (`start_offset`, `end_offset`, `current_offset`) in SQLite.
   - Saves progress regularly and on pause/shutdown.
   - When resumed, each worker picks up from its exact `current_offset`.

3. **URL History Tracking**:
   - Automatically records downloaded URLs, filenames, and download frequency in SQLite.
   - Access previous URLs from the "URL History" drawer to re-download or copy with one click.

4. **Real-time Event Streaming**:
   - Sends real-time progress events (`download:progress`, `download:status_change`) over the Wails event bus.
   - Live speed calculation (rolling EMA) and ETA estimation.

5. **Modern Desktop UI**:
   - Dark theme styling with Tailwind CSS.
   - Native OS directory picker dialog (`SelectDirectory`).
   - Native open file and "Show in Folder" (Finder / Explorer) integration.
   - Interactive chunk visualizer showing each chunk's progress bar.

---

## Development & Running

### Prerequisites
- **Go 1.20+**
- **Node.js 18+** & **npm**
- **Wails CLI** (installed at `~/go/bin/wails`):
  ```bash
  go install github.com/wailsapp/wails/v2/cmd/wails@latest
  ```

### Run in Development Mode
Starts live-reloading for both Go backend and Svelte frontend:
```bash
wails dev
```

### Build Production Desktop App
Builds an optimized, native desktop application (.app on macOS, .exe on Windows, ELF binary on Linux):
```bash
wails build
```

Or build a standalone Go binary directly (assets are embedded via `//go:embed`):
```bash
cd frontend && npm run build && cd ..
go build -o candak
```

### Run Tests
```bash
go test -v ./internal/db
go test -v ./internal/downloader
```

---

## SQLite Database Location
The application creates and maintains its SQLite database at:
- **macOS / Linux**: `~/.candak/candak.db`
- **Windows**: `%USERPROFILE%\.candak\candak.db`
