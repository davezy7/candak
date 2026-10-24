# Feature Specification & Plan: Candak v0.2.0

**Target Version:** v0.2.0  
**Primary Focus:** Download List & Item Card Overhaul (Density, Layout Differentiation, and Navigation Consolidation)  
**Status:** Approved via Grill-Me Session  

---

## 1. Executive Summary

Candak v0.2.0 delivers a high-density, differentiated list and card interface:
1. **Consolidated Navigation**: Eliminates the redundant "Files" tab and promotes "History" into the bottom dock alongside "Downloads" and "Settings".
2. **Differentiated Item States**:
   - **Completed Downloads**: Ultra-compact single-row (~46px) layout displaying file type badge, file size, target directory, and one-click open/folder actions. Chunk visualizers are completely omitted.
   - **Active & Paused Downloads**: Dense 2-tier row (~68px) layout featuring an inline progress track, real-time speed, ETA, and collapsible chunk drawer.
3. **Ultra-Compact Dimensions**: Decreases card padding from `p-5` (20px) to `p-2.5` (10px) and icon squircle from `48px` to `32px`, increasing on-screen visible items from ~3–4 to 10–12 items on a 720p window.

---

## 2. Architecture & UI Specifications

### 2.1 Navigation & Filtering (`frontend/src/App.svelte`)

- **Remove**:
  - `activeTab === 'files'` branch.
  - Redundant secondary header title logic (`{activeTab === 'files' ? 'Downloaded Files' : 'Downloads'}`).
- **Retain & Enhance**:
  - Primary filter pills: `All files ({count})`, `Downloading ({count})`, `Completed ({count})`, and `Paused ({count})`.
- **Bottom Dock**:
  - Button 1: **Downloads** (activates main dashboard).
  - Button 2: **History** (triggers `HistoryModal`).
  - Button 3: **Settings** (triggers `SettingsModal`).

### 2.2 Completed Item Component (`frontend/src/lib/DownloadItem.svelte`)

- **Height**: ~44–48px.
- **Left Side**:
  - `32x32px` rounded squircle icon container with category-based icons (`size={16}`).
  - Filename in bold text (`14px`), clickable to trigger `OpenFile(download.savePath)`.
  - Uppercase extension badge: e.g. `<span class="px-1.5 py-0.5 text-[10px] font-bold bg-sky-500/10 text-sky-600 rounded">ZIP</span>`.
- **Center / Subtitle**:
  - Formatted file size (e.g. `1.2 GB`).
  - Separator dot `·`.
  - Truncated save directory path with tooltip.
- **Right Side**:
  - Persistent icon buttons:
    - **Open Folder** (`<Folder size={15} />`) -> calls `ShowInFolder(download.savePath)`.
    - **Delete** (`<Trash2 size={15} />`) -> opens `ConfirmModal`.

### 2.3 Active & Paused Item Component (`frontend/src/lib/DownloadItem.svelte`)

- **Height**: ~66–70px (collapsed).
- **Tier 1 (Header)**:
  - `32x32px` squircle icon (`size={16}`).
  - Filename + extension badge.
  - Pause / Resume circular action button (`w-7 h-7`).
- **Tier 2 (Progress & Stats)**:
  - 4px sleek progress track (`h-1 rounded-full`).
  - Speed indicator (e.g. `8.4 MB/s`) and ETA (e.g. `2m 14s`).
  - Progress percentage and downloaded / total size.
  - Collapsible chunk toggle (`Hide/View Chunks`) rendering `ChunkVisualizer` only when expanded.

---

## 3. Implementation Steps

1. **Update `App.svelte`**:
   - Remove `activeTab` state and "Files" view logic.
   - Wire the bottom dock's center button to open `HistoryModal`.
   - Update header counter indicators.
2. **Update `DownloadItem.svelte`**:
   - Extract file extension helper for uppercase extension badges.
   - Implement conditional template for `status === 'completed'` vs active/paused states.
   - Apply `p-2.5` container padding and `32px` icon sizing.
   - Make file title and icon clickable to launch the file.
3. **Verify and Test**:
   - Verify layout on active downloads, paused downloads, and completed downloads.
   - Verify keyboard accessibility and click actions.
   - Run unit tests and frontend build (`npm run build`).

---

## 4. Acceptance Criteria

- [ ] "Files" tab is cleanly removed from top navigation and bottom dock.
- [ ] Bottom dock has 3 balanced buttons: Downloads, History, and Settings.
- [ ] Completed downloads do not display any chunk visualizer tracks or chunk toggle buttons.
- [ ] Completed downloads show uppercase extension badges, file size, and target folder path.
- [ ] Completed cards are under 50px in height.
- [ ] Active and paused downloads retain speed, ETA, progress bar, and chunk visualizer drawer.
- [ ] All downloads list smoothly in ultra-compact layout fitting 10+ items on standard 720p window.
