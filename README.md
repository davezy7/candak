# Candak Download Manager

**Version: 0.2.0**

A fast, lightweight, and modern desktop download manager built with Go, Wails v2, Svelte 5, and SQLite. Candak features high-throughput multi-threaded chunking, crash-resilient pause and resume support, a compact mini prompt dialog, and companion browser integration.

---

## Key Features

- **Multi-Threaded Turbo Engine**: Dynamic byte-range chunking (1–16 threads) writes directly to disk offsets, eliminating slow post-download file reassembly.
- **Crash-Resilient Pause & Resume**: Byte-accurate offset persistence in SQLite WAL. Automatically recovers interrupted downloads after app exits or system crashes with zero data loss.
- **In-Page Video Grabber & yt-dlp**: Companion extension detects web media on page. Integrated `yt-dlp` modal resolves formats up to 4K with bitrate-based sizes and DRM stream protection.
- **Batch Download & Multi-Select**: Paste multi-line links with concurrent 4-worker link probing. Select multiple items (`Shift+Click`, `Cmd/Ctrl+A`) for bulk pause, resume, or deletion.
- **High-Density Desktop Table**: Dense data table with search, sort, shift-range multi-select, contextual batch action bar, right-click context menu, and aggregate bandwidth gauge.
- **Browser Integration & Clipboard Watcher**: Captures downloads seamlessly from Chrome, Brave, Edge, and Firefox via local loopback bridge, with automatic clipboard link detection.

---

## Tech Stack

- **Backend**: Go 1.22+, [Wails v2](https://wails.io) (Native Webview IPC), Pure-Go SQLite (`modernc.org/sqlite`, CGO-free storage with WAL mode).
- **Frontend**: Svelte 5, Tailwind CSS, Lucide Icons, Vite.
- **Browser Extension**: WebExtensions API / Manifest V3.

---

## Quick Start

### Prerequisites
- **Go 1.20+**
- **Node.js 18+** & **npm**
- **Wails v2 CLI**:
  ```bash
  go install github.com/wailsapp/wails/v2/cmd/wails@latest
  ```

### Development Mode (Hot-Reload)
Run live-reloading for both Go backend and Svelte frontend:
```bash
wails dev
```

### Production Build
Build the optimized standalone executable / application bundle:
```bash
wails build
```
Outputs:
- **macOS**: `build/bin/candak.app`
- **Windows**: `build/bin/candak.exe`
- **Linux**: `build/bin/candak`

### Run Tests
```bash
# Run unit and storage reconciliation tests
go test -v ./...
```

---

## Browser Extension Setup

1. Launch Candak and open **Settings** (gear icon in the bottom navigation).
2. Click **Open Extension Folder** to reveal the extracted extension directory in Finder or File Explorer.
3. In your browser:
   - Navigate to `chrome://extensions` (Chromium) or `edge://extensions` (Edge).
   - Enable **Developer mode** (top-right toggle).
   - Click **Load unpacked** and select the opened `browser-extension` folder.
4. Downloads initiated in your browser will automatically route to Candak's mini prompt.

---

## Documentation

- [User Guide & IPC Reference](docs/USAGE.md)
- [Architecture & Design Decisions](docs/ARCHITECTURE.md)

---

## License

Distributed under the GNU General Public License v3.0. See [LICENSE](LICENSE) for details.
