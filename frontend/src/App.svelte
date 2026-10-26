<script>
  import { onMount, onDestroy } from 'svelte';
  import { 
    GetDownloads, 
    SetClipboardWatching, 
    SetWindowVisible,
    BatchPauseDownloads,
    BatchResumeDownloads,
    BatchDeleteDownloads
  } from '@wails/go/main/App.js';
  import { EventsOn } from '@wails/runtime/runtime.js';
  import { formatBytes, formatSpeed } from './lib/formatters.js';
  import DownloadItem from './lib/DownloadItem.svelte';
  import AddDownloadModal from './lib/AddDownloadModal.svelte';
  import HistoryModal from './lib/HistoryModal.svelte';
  import SettingsModal from './lib/SettingsModal.svelte';
  import MultiSelectToolbar from './lib/MultiSelectToolbar.svelte';
  import BulkDeleteModal from './lib/BulkDeleteModal.svelte';
  import { 
    Download, 
    Plus, 
    Settings, 
    ArrowDown,
    History
  } from 'lucide-svelte';

  let downloads = [];
  let filter = 'all'; // 'all' | 'downloading' | 'completed' | 'paused'
  let isAddModalOpen = false;
  let isHistoryModalOpen = false;
  let isSettingsModalOpen = false;
  let isBulkDeleteModalOpen = false;
  let initialUrlForModal = '';
  let grabbedData = null;
  let isStandalonePrompt = false;
  let selectedIds = new Set();
  let lastSelectedId = null;
  let isModifierKeyPressed = false;
  let unlistenProgress = null;
  let unlistenStatus = null;
  let unlistenDeleted = null;
  let unlistenGrabbed = null;
  let unlistenPromptMode = null;

  onMount(async () => {
    // Sync clipboard watcher state from saved preference
    const savedClipboard = localStorage.getItem('candak_clipboard_watching') === 'true';
    if (savedClipboard) {
      try {
        await SetClipboardWatching(true);
      } catch (err) {
        console.error('Failed to initialize clipboard watcher:', err);
      }
    }

    await fetchDownloads();

    // Listen to real-time progress events from Go backend
    unlistenProgress = EventsOn('download:progress', (evt) => {
      const idx = downloads.findIndex(d => d.id === evt.downloadId);
      if (idx !== -1) {
        downloads[idx] = {
          ...downloads[idx],
          downloadedSize: evt.downloadedSize,
          speed: evt.speed,
          eta: evt.eta,
          status: evt.status,
          chunks: evt.chunks || downloads[idx].chunks
        };
      }
    });

    unlistenStatus = EventsOn('download:status_change', (dl) => {
      const idx = downloads.findIndex(d => d.id === dl.id);
      if (idx !== -1) {
        downloads[idx] = { ...downloads[idx], ...dl };
      } else {
        downloads = [dl, ...downloads];
      }
      downloads = [...downloads];
    });

    unlistenDeleted = EventsOn('download:deleted', (id) => {
      downloads = downloads.filter(d => d.id !== id);
      if (selectedIds.has(id)) {
        const next = new Set(selectedIds);
        next.delete(id);
        selectedIds = next;
      }
    });

    // Listen for downloads grabbed from browser extension or clipboard
    unlistenGrabbed = EventsOn('download:grabbed', (data) => {
      isStandalonePrompt = true;
      grabbedData = data;
      initialUrlForModal = data.url || '';
      isAddModalOpen = true;
    });

    // Listen for standalone prompt mode
    unlistenPromptMode = EventsOn('download:prompt_mode', (evt) => {
      if (evt && evt.standalone) {
        isStandalonePrompt = true;
        if (evt.data) {
          grabbedData = evt.data;
          initialUrlForModal = evt.data.url || '';
        }
        isAddModalOpen = true;
      } else {
        isStandalonePrompt = false;
        isAddModalOpen = false;
        grabbedData = null;
      }
    });

    document.addEventListener('visibilitychange', handleVisibilityChange);
    window.addEventListener('focus', handleWindowFocus);
  });

  onDestroy(() => {
    if (unlistenProgress) unlistenProgress();
    if (unlistenStatus) unlistenStatus();
    if (unlistenDeleted) unlistenDeleted();
    if (unlistenGrabbed) unlistenGrabbed();
    if (unlistenPromptMode) unlistenPromptMode();
    document.removeEventListener('visibilitychange', handleVisibilityChange);
    window.removeEventListener('focus', handleWindowFocus);
  });

  const handleVisibilityChange = async () => {
    const isVisible = document.visibilityState === 'visible';
    try {
      await SetWindowVisible(isVisible);
    } catch (err) {
      console.error('Failed to update window visibility:', err);
    }
    if (isVisible) {
      await fetchDownloads();
    }
  };

  const handleWindowFocus = async () => {
    try {
      await SetWindowVisible(true);
    } catch (err) {
      console.error('Failed to update window visibility on focus:', err);
    }
    await fetchDownloads();
  };

  async function fetchDownloads() {
    try {
      const list = await GetDownloads();
      downloads = list || [];
    } catch (err) {
      console.error('Failed to load downloads:', err);
    }
  }

  function handleSelectHistoryUrl(url) {
    initialUrlForModal = url;
    isAddModalOpen = true;
  }

  // Reactive Counts
  $: totalCount = downloads.length;
  $: downloadingCount = downloads.filter(d => d.status === 'downloading').length;
  $: completedCount = downloads.filter(d => d.status === 'completed').length;
  $: pausedCount = downloads.filter(d => d.status === 'paused').length;

  // Filtered List based on pill selection
  $: filteredDownloads = downloads.filter(d => {
    if (filter === 'downloading' && d.status !== 'downloading') return false;
    if (filter === 'completed' && d.status !== 'completed') return false;
    if (filter === 'paused' && d.status !== 'paused') return false;
    return true;
  });

  $: isMultiSelectMode = selectedIds.size > 0;

  function clearSelection() {
    selectedIds = new Set();
    lastSelectedId = null;
  }

  function setFilter(newFilter) {
    if (filter !== newFilter) {
      filter = newFilter;
      clearSelection();
    }
  }

  function handleNavDownloads() {
    if (filter !== 'all') {
      setFilter('all');
    } else {
      clearSelection();
    }
  }

  function handleToggleSelect(downloadId, event) {
    const next = new Set(selectedIds);
    if (event?.shiftKey && lastSelectedId && filteredDownloads.length > 0) {
      const idx1 = filteredDownloads.findIndex(d => d.id === lastSelectedId);
      const idx2 = filteredDownloads.findIndex(d => d.id === downloadId);
      if (idx1 !== -1 && idx2 !== -1) {
        const start = Math.min(idx1, idx2);
        const end = Math.max(idx1, idx2);
        for (let i = start; i <= end; i++) {
          next.add(filteredDownloads[i].id);
        }
        selectedIds = next;
        lastSelectedId = downloadId;
        return;
      }
    }

    if (next.has(downloadId)) {
      next.delete(downloadId);
      if (next.size === 0) {
        lastSelectedId = null;
      }
    } else {
      next.add(downloadId);
      lastSelectedId = downloadId;
    }
    selectedIds = next;
  }

  function handleToggleSelectAll() {
    if (selectedIds.size >= filteredDownloads.length && filteredDownloads.length > 0) {
      selectedIds = new Set();
      lastSelectedId = null;
    } else {
      selectedIds = new Set(filteredDownloads.map(d => d.id));
      if (filteredDownloads.length > 0) {
        lastSelectedId = filteredDownloads[filteredDownloads.length - 1].id;
      }
    }
  }

  async function handleBatchPause() {
    const ids = Array.from(selectedIds);
    if (ids.length === 0) return;
    try {
      await BatchPauseDownloads(ids);
    } catch (err) {
      console.error('Failed to batch pause downloads:', err);
    } finally {
      await fetchDownloads();
    }
  }

  async function handleBatchResume() {
    const ids = Array.from(selectedIds);
    if (ids.length === 0) return;
    try {
      await BatchResumeDownloads(ids);
    } catch (err) {
      console.error('Failed to batch resume downloads:', err);
    } finally {
      await fetchDownloads();
    }
  }

  async function handleConfirmBatchDelete(deleteFiles) {
    const ids = Array.from(selectedIds);
    isBulkDeleteModalOpen = false;
    if (ids.length === 0) return;
    try {
      await BatchDeleteDownloads(ids, deleteFiles);
      clearSelection();
    } catch (err) {
      console.error('Failed to batch delete downloads:', err);
    } finally {
      await fetchDownloads();
    }
  }

  function handleKeyDown(e) {
    if (e.key === 'Shift' || e.key === 'Meta' || e.key === 'Control') {
      isModifierKeyPressed = true;
    }

    if (e.key === 'Escape') {
      if (isBulkDeleteModalOpen) {
        isBulkDeleteModalOpen = false;
        return;
      }
      if (selectedIds.size > 0) {
        selectedIds = new Set();
        lastSelectedId = null;
      }
    }

    if ((e.metaKey || e.ctrlKey) && (e.key === 'a' || e.key === 'A')) {
      const activeEl = document.activeElement;
      const isInput = activeEl && (activeEl.tagName === 'INPUT' || activeEl.tagName === 'TEXTAREA' || activeEl.isContentEditable);
      if (!isInput && !isAddModalOpen && !isHistoryModalOpen && !isSettingsModalOpen && !isBulkDeleteModalOpen) {
        e.preventDefault();
        if (filteredDownloads.length > 0) {
          selectedIds = new Set(filteredDownloads.map(d => d.id));
          lastSelectedId = filteredDownloads[filteredDownloads.length - 1].id;
        }
      }
    }
  }

  function handleKeyUp(e) {
    if (!e.shiftKey && !e.metaKey && !e.ctrlKey) {
      isModifierKeyPressed = false;
    }
  }
</script>

<svelte:window on:keydown={handleKeyDown} on:keyup={handleKeyUp} />

{#if isStandalonePrompt}
  <!-- Standalone Mini Prompt Window (Decoupled from Dashboard) -->
  <div class="w-screen h-screen bg-[var(--color-surface)] text-[var(--color-text)] flex flex-col overflow-hidden select-none font-sans">
    <AddDownloadModal 
      isOpen={true} 
      isStandalone={true}
      grabbedData={grabbedData}
      onClose={() => {
        isStandalonePrompt = false;
        isAddModalOpen = false;
        grabbedData = null;
        initialUrlForModal = '';
      }} 
      onAdded={fetchDownloads} 
    />
  </div>
{:else}
<div class="flex flex-col h-screen bg-[var(--color-bg)] text-[var(--color-text)] select-none font-sans overflow-hidden transition-colors duration-200">
  <!-- Top Navigation & Header -->
  <header class="px-8 pt-7 pb-4 shrink-0" style="--wails-draggable:drag">
    <div class="flex items-center justify-between">
      <!-- Title -->
      <div>
        <h1 class="text-[26px] font-bold tracking-tight text-[var(--color-text)]">
          Downloads
        </h1>
      </div>

      <!-- Header Action Buttons -->
      <div class="flex items-center gap-2" style="--wails-draggable:no-drag">
        <!-- New Download Button -->
        <button 
          on:click={() => { initialUrlForModal = ''; isAddModalOpen = true; }}
          class="flex items-center gap-1.5 px-4 py-2 bg-[var(--color-accent)] hover:bg-[var(--color-accent-hover)] text-white text-xs font-bold rounded-full shadow-sm hover:shadow transition active:scale-95"
          title="Add New Download"
        >
          <Plus size={16} strokeWidth={3} />
          <span>Add</span>
        </button>
      </div>
    </div>

    <!-- Filter Pills matching reference screenshot -->
    <div class="flex items-center gap-2 mt-4 overflow-x-auto no-scrollbar" style="--wails-draggable:no-drag">
      <button 
        on:click={() => setFilter('all')}
        class="px-4 py-1.5 rounded-full text-[13px] font-semibold transition-all {
          filter === 'all' 
            ? 'bg-[var(--color-accent-subtle)] text-[var(--color-accent-subtle-text)]' 
            : 'text-[var(--color-secondary)] hover:text-[var(--color-text)] hover:bg-[var(--color-surface)]/60'
        }"
      >
        All files {totalCount}
      </button>

      <button 
        on:click={() => setFilter('downloading')}
        class="px-4 py-1.5 rounded-full text-[13px] font-semibold transition-all {
          filter === 'downloading' 
            ? 'bg-[var(--color-accent-subtle)] text-[var(--color-accent-subtle-text)]' 
            : 'text-[var(--color-secondary)] hover:text-[var(--color-text)] hover:bg-[var(--color-surface)]/60'
        }"
      >
        Downloading {downloadingCount}
      </button>

      <button 
        on:click={() => setFilter('completed')}
        class="px-4 py-1.5 rounded-full text-[13px] font-semibold transition-all {
          filter === 'completed' 
            ? 'bg-[var(--color-accent-subtle)] text-[var(--color-accent-subtle-text)]' 
            : 'text-[var(--color-secondary)] hover:text-[var(--color-text)] hover:bg-[var(--color-surface)]/60'
        }"
      >
        Completed {completedCount}
      </button>

      {#if pausedCount > 0}
        <button 
          on:click={() => setFilter('paused')}
          class="px-4 py-1.5 rounded-full text-[13px] font-semibold transition-all {
            filter === 'paused' 
              ? 'bg-[var(--color-accent-subtle)] text-[var(--color-accent-subtle-text)]' 
              : 'text-[var(--color-secondary)] hover:text-[var(--color-text)] hover:bg-[var(--color-surface)]/60'
          }"
        >
          Paused {pausedCount}
        </button>
      {/if}
    </div>
  </header>

  <!-- Content List Area -->
  <main class="flex-1 overflow-y-auto px-8 pb-24 space-y-3.5">
    {#if filteredDownloads.length === 0}
      <div class="h-full flex flex-col items-center justify-center py-20 text-center">
        <div class="w-16 h-16 rounded-3xl bg-[var(--color-surface)] border border-[var(--color-border)] shadow-xs flex items-center justify-center text-[var(--color-secondary)] mb-3">
          <Download size={26} strokeWidth={2} />
        </div>
        <h3 class="text-base font-bold text-[var(--color-text)]">
          No downloads in this view
        </h3>
        <p class="text-xs text-[var(--color-secondary)] max-w-xs mt-1">
          Click '+ Add' to start your first file download.
        </p>
        <button 
          on:click={() => isAddModalOpen = true}
          class="mt-4 px-5 py-2.5 text-xs font-bold text-white bg-[var(--color-accent)] hover:bg-[var(--color-accent-hover)] rounded-full shadow-sm hover:shadow transition"
        >
          Start a Download
        </button>
      </div>
    {:else}
      {#each filteredDownloads as download (download.id)}
        <DownloadItem 
          {download} 
          isSelected={selectedIds.has(download.id)}
          {isMultiSelectMode}
          {isModifierKeyPressed}
          onToggleSelect={(e) => handleToggleSelect(download.id, e)}
          onAction={fetchDownloads} 
        />
      {/each}
    {/if}
  </main>

  <!-- Floating Multi-Select Toolbar -->
  {#if isMultiSelectMode}
    <MultiSelectToolbar 
      selectedCount={selectedIds.size}
      totalVisibleCount={filteredDownloads.length}
      onToggleSelectAll={handleToggleSelectAll}
      onPauseSelected={handleBatchPause}
      onResumeSelected={handleBatchResume}
      onDeleteSelected={() => isBulkDeleteModalOpen = true}
      onClear={() => { selectedIds = new Set(); lastSelectedId = null; }}
    />
  {/if}

  <!-- Bottom Navigation Bar matching reference screenshot -->
  <footer class="fixed bottom-4 left-0 right-0 px-8 flex justify-center pointer-events-none">
    <nav class="bg-[var(--color-nav-bg)] border border-[var(--color-border)] rounded-3xl shadow-lg shadow-black/5 py-2.5 px-10 flex items-center gap-12 pointer-events-auto">
      <!-- 1. Downloads Button -->
      <button 
        on:click={handleNavDownloads}
        class="flex flex-col items-center gap-1 transition-colors text-[var(--color-accent)]"
        title="Downloads"
        aria-label="Downloads"
      >
        <ArrowDown size={19} strokeWidth={2.6} />
        <span class="text-[11px] font-semibold">Downloads</span>
      </button>

      <!-- 2. History Button -->
      <button 
        on:click={() => isHistoryModalOpen = true}
        class="flex flex-col items-center gap-1 transition-colors text-[var(--color-secondary)] hover:text-[var(--color-text)]"
        title="Open History"
        aria-label="Open History"
      >
        <History size={19} strokeWidth={2} />
        <span class="text-[11px] font-semibold">History</span>
      </button>

      <!-- 3. Settings Button -->
      <button 
        on:click={() => isSettingsModalOpen = true}
        class="flex flex-col items-center gap-1 transition-colors text-[var(--color-secondary)] hover:text-[var(--color-text)]"
        title="Open Settings"
        aria-label="Open Settings"
      >
        <Settings size={19} strokeWidth={2} />
        <span class="text-[11px] font-semibold">Settings</span>
      </button>
    </nav>
  </footer>

  <!-- Modals -->
  <AddDownloadModal 
    isOpen={isAddModalOpen} 
    initialUrl={initialUrlForModal}
    grabbedData={grabbedData}
    onClose={() => {
      isAddModalOpen = false;
      grabbedData = null;
      initialUrlForModal = '';
    }} 
    onAdded={fetchDownloads} 
  />

  <HistoryModal 
    isOpen={isHistoryModalOpen} 
    onClose={() => isHistoryModalOpen = false} 
    onSelectUrl={handleSelectHistoryUrl} 
  />

  <SettingsModal 
    isOpen={isSettingsModalOpen} 
    onClose={() => isSettingsModalOpen = false} 
  />

  <BulkDeleteModal
    isOpen={isBulkDeleteModalOpen}
    count={selectedIds.size}
    onConfirm={handleConfirmBatchDelete}
    onCancel={() => isBulkDeleteModalOpen = false}
  />
</div>
{/if}
