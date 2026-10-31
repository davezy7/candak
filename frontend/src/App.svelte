<script>
  import { onMount, onDestroy } from 'svelte';
  import {
    GetDownloads, 
    SetClipboardWatching, 
    SetWindowVisible,
    BatchPauseDownloads,
    BatchResumeDownloads,
    BatchDeleteDownloads,
    CheckForUpdates,
    PauseDownload,
    ResumeDownload,
    CancelDownload,
    OpenFile,
    ShowInFolder,
    SetMaxConcurrentDownloads
  } from '@wails/go/main/App.js';
  import { EventsOn } from '@wails/runtime/runtime.js';
  import { FILTER_MODE, IPC_EVENTS, STORAGE_KEYS } from './lib/constants.js';
  import TableView from './lib/TableView.svelte';
  import AddDownloadModal from './lib/AddDownloadModal.svelte';
  import HistoryModal from './lib/HistoryModal.svelte';
  import SettingsModal from './lib/SettingsModal.svelte';
  import UpdateModal from './lib/UpdateModal.svelte';
  import BulkDeleteModal from './lib/BulkDeleteModal.svelte';
  import StatePreviewBar from './lib/StatePreviewBar.svelte';

  let downloads = [];
  let filter = FILTER_MODE.ALL;
  let isAddModalOpen = false;
  let isHistoryModalOpen = false;
  let isSettingsModalOpen = false;
  let isBulkDeleteModalOpen = false;
  let isUpdateModalOpen = false;
  let updateCheck = null;
  let updateError = '';
  let initialUrlForModal = '';
  let grabbedData = null;
  let isStandalonePrompt = false;
  let selectedIds = new Set();
  let pendingDeleteIds = [];
  let unlistenProgress = null;
  let unlistenStatus = null;
  let unlistenDeleted = null;
  let unlistenGrabbed = null;
  let unlistenPromptMode = null;

  onMount(async () => {
    // Sync clipboard watcher state from saved preference
    const savedClipboard = localStorage.getItem(STORAGE_KEYS.CLIPBOARD_WATCHING) === 'true';
    if (savedClipboard) {
      try {
        await SetClipboardWatching(true);
      } catch (err) {
        console.error('Failed to initialize clipboard watcher:', err);
      }
    }

    // Background update check, throttled to once per 24 hours via localStorage
    checkForUpdates(false);

    // Initialize saved concurrency limit in engine before initial fetch
    const savedConcurrency = localStorage.getItem(STORAGE_KEYS.MAX_CONCURRENT_DOWNLOADS);
    if (savedConcurrency) {
      const limit = parseInt(savedConcurrency, 10);
      if (!isNaN(limit) && limit >= 1 && limit <= 10) {
        SetMaxConcurrentDownloads(limit);
      }
    }

    await fetchDownloads();

    // Listen to real-time progress events from Go backend
    unlistenProgress = EventsOn(IPC_EVENTS.PROGRESS, (evt) => {
      const idx = downloads.findIndex(d => d.id === evt.downloadId);
      if (idx !== -1) {
        downloads[idx] = {
          ...downloads[idx],
          totalSize: evt.totalSize > 0 ? evt.totalSize : downloads[idx].totalSize,
          downloadedSize: evt.downloadedSize,
          speed: evt.speed,
          eta: evt.eta,
          status: evt.status,
          chunks: evt.chunks || downloads[idx].chunks
        };
      }
    });

    unlistenStatus = EventsOn(IPC_EVENTS.STATUS_CHANGE, (dl) => {
      const idx = downloads.findIndex(d => d.id === dl.id);
      if (idx !== -1) {
        downloads[idx] = { ...downloads[idx], ...dl };
      } else {
        downloads = [dl, ...downloads];
      }
      downloads = [...downloads];
    });

    unlistenDeleted = EventsOn(IPC_EVENTS.DELETED, (id) => {
      downloads = downloads.filter(d => d.id !== id);
      if (selectedIds.has(id)) {
        const next = new Set(selectedIds);
        next.delete(id);
        selectedIds = next;
      }
    });

    // Listen for downloads grabbed from browser extension or clipboard
    unlistenGrabbed = EventsOn(IPC_EVENTS.GRABBED, (data) => {
      isStandalonePrompt = true;
      grabbedData = data;
      initialUrlForModal = data.url || '';
      isAddModalOpen = true;
    });

    // Listen for standalone prompt mode
    unlistenPromptMode = EventsOn(IPC_EVENTS.PROMPT_MODE, (evt) => {
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

  const UPDATE_CHECK_INTERVAL_MS = 24 * 60 * 60 * 1000;

  async function checkForUpdates(force) {
    const last = parseInt(localStorage.getItem(STORAGE_KEYS.LAST_UPDATE_CHECK) || '0', 10);
    if (!force && Date.now() - last < UPDATE_CHECK_INTERVAL_MS) return;
    try {
      const result = await CheckForUpdates();
      localStorage.setItem(STORAGE_KEYS.LAST_UPDATE_CHECK, String(Date.now()));
      updateCheck = result;
      updateError = '';
    } catch (err) {
      console.error('Update check failed:', err);
      updateError = 'Check failed (network error)';
    }
  }

  $: updateAvailable = updateCheck?.updateAvailable === true;
  $: isAnyModalOpen = isAddModalOpen || isHistoryModalOpen || isSettingsModalOpen || isBulkDeleteModalOpen || isUpdateModalOpen || isStandalonePrompt;

  function handleSelectHistoryUrl(url) {
    initialUrlForModal = url;
    isAddModalOpen = true;
  }

  async function handleSinglePause(id) {
    try {
      await PauseDownload(id);
      await fetchDownloads();
    } catch (err) {
      console.error('Failed to pause download:', err);
    }
  }

  async function handleSingleResume(id) {
    try {
      await ResumeDownload(id);
      await fetchDownloads();
    } catch (err) {
      console.error('Failed to resume download:', err);
    }
  }

  function handleSingleDelete(id) {
    pendingDeleteIds = [id];
    isBulkDeleteModalOpen = true;
  }

  async function handleSingleOpenFile(id) {
    const dl = downloads.find(d => d.id === id);
    if (!dl) return;
    try {
      await OpenFile(dl.savePath);
    } catch (err) {
      console.error('Failed to open file:', err);
    }
  }

  async function handleSingleShowFolder(id) {
    const dl = downloads.find(d => d.id === id);
    if (!dl) return;
    try {
      await ShowInFolder(dl.savePath);
    } catch (err) {
      console.error('Failed to show in folder:', err);
    }
  }

  function handleApplyFixture(fixtureList) {
    clearSelection();
    downloads = [...fixtureList];
  }

  function handleRestoreReal() {
    clearSelection();
    fetchDownloads();
  }

  function handleMutateSelected(status, extra = {}) {
    if (selectedIds.size === 0) return;
    downloads = downloads.map(d => {
      if (selectedIds.has(d.id)) {
        return {
          ...d,
          status,
          ...extra
        };
      }
      return d;
    });
  }

  function clearSelection() {
    selectedIds = new Set();
  }

  function setFilter(newFilter) {
    if (filter !== newFilter) {
      filter = newFilter;
      clearSelection();
    }
  }

  function handleSelectionChange(newSet) {
    selectedIds = newSet;
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

  function handleOpenBatchDelete() {
    pendingDeleteIds = Array.from(selectedIds);
    isBulkDeleteModalOpen = true;
  }

  async function handleConfirmDelete(deleteFiles) {
    const ids = [...pendingDeleteIds];
    isBulkDeleteModalOpen = false;
    if (ids.length === 0) return;
    try {
      if (ids.length === 1) {
        await CancelDownload(ids[0], deleteFiles);
      } else {
        await BatchDeleteDownloads(ids, deleteFiles);
      }
      pendingDeleteIds = [];
      const next = new Set(selectedIds);
      let changed = false;
      for (const id of ids) {
        if (next.delete(id)) changed = true;
      }
      if (changed) {
        selectedIds = next;
      }
    } catch (err) {
      console.error('Failed to delete download(s):', err);
    } finally {
      await fetchDownloads();
    }
  }
</script>

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
<div class="flex flex-col h-screen bg-[var(--color-bg)] text-[var(--color-text)] select-none font-sans overflow-hidden transition-colors duration-200 relative">
  <TableView 
    {downloads}
    {filter}
    onFilterChange={setFilter}
    onOpenAdd={() => { initialUrlForModal = ''; isAddModalOpen = true; }}
    onOpenHistory={() => isHistoryModalOpen = true}
    onOpenSettings={() => isSettingsModalOpen = true}
    onPause={handleSinglePause}
    onResume={handleSingleResume}
    onDelete={handleSingleDelete}
    onOpenFile={handleSingleOpenFile}
    onShowInFolder={handleSingleShowFolder}
    {selectedIds}
    onSelectionChange={handleSelectionChange}
    onBatchPause={handleBatchPause}
    onBatchResume={handleBatchResume}
    onBatchDelete={handleOpenBatchDelete}
    modalOpen={isAnyModalOpen}
  />

  <!-- Temporary State Previewer Component (Development Only) -->
  {#if import.meta.env.DEV}
    <StatePreviewBar 
      bind:downloads
      {selectedIds}
      onApplyFixture={handleApplyFixture}
      onRestoreReal={handleRestoreReal}
      onMutateSelected={handleMutateSelected}
    />
  {/if}

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
    {updateCheck}
    {updateError}
    onCheckForUpdates={() => checkForUpdates(true)}
    onViewUpdate={() => {
      isSettingsModalOpen = false;
      isUpdateModalOpen = true;
    }}
  />

  <UpdateModal
    isOpen={isUpdateModalOpen}
    check={updateCheck}
    onClose={() => isUpdateModalOpen = false}
  />

  <BulkDeleteModal
    isOpen={isBulkDeleteModalOpen}
    count={pendingDeleteIds.length}
    onConfirm={handleConfirmDelete}
    onCancel={() => { isBulkDeleteModalOpen = false; pendingDeleteIds = []; }}
  />
</div>
{/if}
