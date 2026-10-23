# Candak - Download Manager Desktop App

A multi-threaded desktop download manager engineered in **Go** and **Wails v2**, featuring an instant **Svelte 5 + Tailwind CSS v4** interface and pure-Go **SQLite** persistence (`modernc.org/sqlite`).

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

For full documentation and API reference, see [USAGE.md](file:///Users/diksha/Documents/candak/docs/USAGE.md) and [ARCHITECTURE.md](file:///Users/diksha/Documents/candak/docs/ARCHITECTURE.md).

---

## License
See [LICENSE](file:///Users/diksha/Documents/candak/LICENSE) file.
