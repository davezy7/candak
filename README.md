# Candak - Download Manager Desktop App

A multi-threaded desktop download manager engineered in **Go** and **Wails v2**, featuring an instant **Svelte 5 + Tailwind CSS v4** interface and pure-Go **SQLite** persistence (`modernc.org/sqlite`).

Designed strictly following **SOLID Principles**, **Clean Architecture**, and **Go Best Practices**.

---

## Architecture & SOLID Highlights

- **Single Responsibility Principle (SRP)**:
  - `storage/sqlite`: Handles pure database operations with WAL mode, foreign keys, and indexed queries.
  - `prober`: URL metadata inspection and filename extraction.
  - `planner`: Boundary division and chunk partition mathematics.
  - `downloader/worker`: Concurrent HTTP streaming and random-access file writing (`io.WriterAt`).
  - `downloader/metrics`: Transfer rate smoothing via exponential moving average (EMA) and ETA.
  - `events`: Decoupled system event broadcast sink.
  - `sysutil`: Native operating system integration (Finder/Explorer/open).
- **Open/Closed Principle (OCP)**: All core components interact through interfaces (`storage.Repository`, `prober.URLProber`, `planner.ChunkPlanner`, `events.Broadcaster`), permitting drop-in extensions without modifying existing orchestration code.
- **Liskov Substitution Principle (LSP)**: Mock and alternative implementations (e.g., `events.NullBroadcaster`, in-memory repositories) safely substitute concrete implementations without breaking contracts.
- **Interface Segregation Principle (ISP)**: Granular interfaces (`DownloadRepository`, `ChunkRepository`, `HistoryRepository`) ensure callers depend strictly on the methods they require.
- **Dependency Inversion Principle (DIP)**: `downloader.Manager` depends exclusively on abstract contracts, with concrete implementations injected at application startup.

For complete architectural breakdowns and diagrams, see [ARCHITECTURE.md](file:///Users/diksha/Documents/candak/docs/ARCHITECTURE.md).

---

## Features

1. **Multi-Threaded Concurrent Chunking**:
   - Queries server `Accept-Ranges` and divides files into balanced byte segments (1 to 16 parallel connections).
   - Writes directly to target files via `io.WriterAt`, avoiding post-download file concatenation.

2. **Resume & Chunk Offset Tracking (`modernc.org/sqlite`)**:
   - Stores exact byte-level chunk offsets (`current_offset`) in SQLite.
   - Saves progress periodically and immediately upon pause/shutdown.
   - When resumed, each connection picks up from its exact saved byte offset.

3. **URL History Management**:
   - Tracks previously downloaded URLs, target filenames, and download frequencies in SQLite.
   - Quick one-click re-downloading and clipboard copying from the URL history drawer.

4. **Real-time Event Streaming**:
   - Streams `download:progress` and `download:status_change` events over the Wails event bus without UI polling.
   - Responsive speed calculation and ETA estimation.

5. **Modern Desktop Interface**:
   - Sleek dark theme with Tailwind CSS v4 and Lucide icons.
   - Native OS directory picker dialog (`SelectDirectory`).
   - Native "Open File" and "Show in Folder" (macOS Finder / Windows Explorer) integration.
   - Interactive chunk visualizer showing each chunk's progress bar and byte offsets.

---

## Quick Start

### Prerequisites
- **Go 1.20+** (tested on Go 1.26+)
- **Node.js 18+** & **npm**
- **Wails CLI**:
  ```bash
  go install github.com/wailsapp/wails/v2/cmd/wails@latest
  ```

### Development Mode (with Live Reload)
```bash
wails dev
```

### Production Build
```bash
wails build
```

### Run Tests
```bash
go test -v ./...
```

For full documentation and API reference, see [USAGE.md](file:///Users/diksha/Documents/candak/docs/USAGE.md).

---

## License
See [LICENSE](file:///Users/diksha/Documents/candak/LICENSE) file.
