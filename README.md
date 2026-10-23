# Candak - Download Manager Desktop App

A fast, reliable desktop download manager featuring multi-threaded chunking, pause & resume support, and URL history tracking.

---

## Features

1. **Multi-Threaded Concurrent Chunking**:
   - Queries server range capabilities and divides files into balanced byte segments (1 to 16 parallel connections).
   - Writes directly to target files concurrently, eliminating slow post-download file merging.

2. **Resume & Chunk Offset Tracking**:
   - Stores exact byte-level chunk offsets locally.
   - Saves progress periodically and immediately upon pause or shutdown.
   - When resumed, each connection picks up from its exact saved byte offset.

3. **URL History Management**:
   - Tracks previously downloaded URLs, target filenames, and download frequencies.
   - Quick one-click re-downloading and clipboard copying from the history drawer.

4. **Real-time Event Streaming & Progress**:
   - Live visual chunk monitors and real-time progress updates without polling.
   - Responsive transfer speed calculation and estimated time remaining.

---

## Quick Start

### Build and Run
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
