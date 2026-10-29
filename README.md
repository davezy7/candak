# Candak Download Manager

**Version: 0.2.0**

A fast, lightweight, and modern desktop download manager built with Go, Wails v2, Svelte 5, and SQLite. Candak features high-throughput multi-threaded chunking, crash-resilient pause and resume support, a compact mini prompt dialog, and companion browser integration.

---

## Key Features

### 1. Multi-Threaded Chunking Engine
- **Range-Capable Segmentation**: Probes servers for byte-range support and dynamically divides downloads into balanced concurrent chunk workers (1 to 16 parallel threads).
- **Direct Concurrent File Writes**: Each chunk worker writes directly into its assigned byte offset in the destination file, eliminating CPU- and disk-intensive post-download file reassembly.
- **Resilient Fallback**: Gracefully handles non-range servers by falling back to reliable single-stream downloads without corruption.

### 2. Fault-Tolerant Pause, Resume & Crash Recovery
- **Byte-Accurate Offset Tracking**: Persists progress offsets transactionally in SQLite with WAL mode.
- **Interrupted Download Reconciliation**: Automatically detects and heals downloads interrupted by application exits, force-quits, or system crashes on startup, cleanly transitioning them to paused state with zero data loss.
- **Instant Resume**: Resumes picking up byte streams from the exact verified offset saved for each connection chunk.

### 3. Compact Mini Prompt & Background Daemon Mode
- **System Tray & Dock Daemon**: Keeps the application running unobtrusively in the background (`HideWindowOnClose`).
- **Dynamic Mini Prompt**: When a download is captured from the browser or clipboard, Candak presents a compact 500x220 dialog displaying filename, file size, resumability, and quick download actions.
- **Expandable Tuning**: Allows expanding the mini prompt to adjust destination directories and parallel thread count (with accurate thumb-aligned slider indicators).
- **Native OS Notifications**: Dispatches system desktop notifications on download start and completion.
- **Seamless Window Restoration**: Restores smoothly between the background daemon state, mini prompt, and full dashboard upon clicking the macOS Dock or Windows taskbar icon.

### 4. Browser Integration & Clipboard Watcher
- **Companion Extension (Manifest V3)**: Captures downloads directly from Chromium browsers (Chrome, Brave, Edge) and Firefox.
- **Embedded Deployment**: The browser extension is compiled directly into the binary and automatically unpacked to `~/.candak/browser-extension` on first launch.
- **One-Click Extension Access**: Dedicated "Open Extension Folder" button in Settings to easily load the unpacked extension in developer mode.
- **Local Loopback Bridge**: Communicates via an origin-restricted local server on `127.0.0.1:41920` with graceful fallback to standard browser downloads if Candak is closed.
- **Clipboard Watcher**: Background watcher detects copied downloadable URLs and prompts to download automatically.

### 5. Extension Whitelist & File Filtering
- **Accepted File Extensions**: Built-in whitelist covering common archives, executables, media, and documents.
- **Configurable Enforcement**: Option in Settings to restrict automated downloads to permitted formats.
- **Inline Whitelist Prompting**: In-dialog warning and one-click whitelist addition when downloading unrecognized file extensions.

### 6. Clean Svelte 5 Dashboard
- **Downloads View**: Live filter pills (All, Downloading, Completed, Paused) with individual chunk visualizer monitors, transfer speed gauges, and estimated time remaining.
- **Dedicated Files View**: Quick-access listing of completed downloads with one-click folder reveal in Finder / Explorer.
- **Settings Modal**: Configure default download paths, default chunk count, extension filtering, clipboard monitoring, and theme preferences.
- **Theme Support**: System, Light, and Dark mode color schemes configured via Settings.
- **URL History Drawer**: Inspect past downloads, re-download previous URLs with one click, or delete individual history items.

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

Distributed under the MIT License. See [LICENSE](LICENSE) for details.
