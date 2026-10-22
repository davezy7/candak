<script>
  import { onMount, onDestroy } from 'svelte';
  import { GetDownloads } from '@wails/go/main/App.js';
  import { EventsOn } from '@wails/runtime/runtime.js';
  import { formatBytes, formatSpeed } from './lib/formatters.js';
  import DownloadItem from './lib/DownloadItem.svelte';
  import AddDownloadModal from './lib/AddDownloadModal.svelte';
  import HistoryModal from './lib/HistoryModal.svelte';
  import { 
    Download, 
    Plus, 
    History, 
    Search, 
    Zap, 
    CheckCircle2, 
    PauseCircle, 
    ArrowDownCircle, 
    HardDrive,
    Layers,
    Database
  } from 'lucide-svelte';

  let downloads = [];
  let filter = 'all'; // 'all' | 'downloading' | 'paused' | 'completed'
  let searchQuery = '';
  let isAddModalOpen = false;
  let isHistoryModalOpen = false;
  let initialUrlForModal = '';
  let unlistenProgress = null;
  let unlistenStatus = null;
  let unlistenDeleted = null;

  onMount(async () => {
    await fetchDownloads();

    // Listen to real-time progress events from Go backend
    unlistenProgress = EventsOn('download:progress', (evt) => {
      const idx = downloads.findIndex(d => d.id === evt.downloadId);
      if (idx !== -1) {
        downloads[idx].downloadedSize = evt.downloadedSize;
        downloads[idx].speed = evt.speed;
        downloads[idx].eta = evt.eta;
        downloads[idx].status = evt.status;
        if (evt.chunks) {
          downloads[idx].chunks = evt.chunks;
        }
        downloads = [...downloads];
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
    });
  });

  onDestroy(() => {
    if (unlistenProgress) unlistenProgress();
    if (unlistenStatus) unlistenStatus();
    if (unlistenDeleted) unlistenDeleted();
  });

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

  // Reactive Stats
  $: activeCount = downloads.filter(d => d.status === 'downloading').length;
  $: pausedCount = downloads.filter(d => d.status === 'paused').length;
  $: completedCount = downloads.filter(d => d.status === 'completed').length;
  $: totalSpeed = downloads.reduce((acc, d) => d.status === 'downloading' ? acc + (d.speed || 0) : acc, 0);

  // Filtered list
  $: filteredDownloads = downloads.filter(d => {
    if (filter === 'downloading' && d.status !== 'downloading') return false;
    if (filter === 'paused' && d.status !== 'paused') return false;
    if (filter === 'completed' && d.status !== 'completed') return false;

    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase();
      return d.filename.toLowerCase().includes(q) || d.url.toLowerCase().includes(q);
    }
    return true;
  });
</script>

<div class="flex flex-col h-screen bg-[#0b0f17] text-slate-100 select-none">
  <!-- Top App Navigation Bar -->
  <header class="h-16 border-b border-slate-800/80 bg-slate-950/70 backdrop-blur px-6 flex items-center justify-between shrink-0" style="--wails-draggable:drag">
    <!-- Brand / Logo -->
    <div class="flex items-center gap-3">
      <div class="w-9 h-9 rounded-xl bg-gradient-to-tr from-cyan-600 to-blue-500 flex items-center justify-center text-white shadow-lg shadow-cyan-500/20">
        <ArrowDownCircle size={22} />
      </div>
      <div>
        <div class="flex items-center gap-2">
          <h1 class="text-base font-bold tracking-tight text-white">Candak</h1>
          <span class="px-2 py-0.5 text-[10px] font-semibold bg-cyan-950 text-cyan-400 border border-cyan-800/60 rounded-full">
            Wails + Svelte
          </span>
          <span class="px-2 py-0.5 text-[10px] font-medium bg-slate-800 text-slate-400 border border-slate-700/60 rounded-full flex items-center gap-1">
            <Database size={10} /> SQLite
          </span>
        </div>
        <p class="text-[11px] text-slate-400">High-performance chunked download manager</p>
      </div>
    </div>

    <!-- Live Stats Pill -->
    <div class="hidden md:flex items-center gap-6 px-4 py-1.5 bg-slate-900/80 border border-slate-800 rounded-lg text-xs font-mono">
      <div class="flex items-center gap-2">
        <span class="w-2 h-2 rounded-full {activeCount > 0 ? 'bg-emerald-400 animate-pulse' : 'bg-slate-600'}"></span>
        <span class="text-slate-400">Active:</span>
        <span class="text-slate-200 font-semibold">{activeCount}</span>
      </div>

      <div class="w-px h-3 bg-slate-800"></div>

      <div class="flex items-center gap-1.5 text-cyan-400">
        <Zap size={14} />
        <span class="font-semibold">{formatSpeed(totalSpeed)}</span>
      </div>

      <div class="w-px h-3 bg-slate-800"></div>

      <div class="flex items-center gap-1.5 text-slate-400">
        <CheckCircle2 size={14} class="text-emerald-400" />
        <span>Done:</span>
        <span class="text-slate-200 font-semibold">{completedCount}</span>
      </div>
    </div>

    <!-- Actions -->
    <div class="flex items-center gap-2.5" style="--wails-draggable:no-drag">
      <button 
        on:click={() => isHistoryModalOpen = true}
        class="px-3 py-2 text-xs font-medium text-slate-300 hover:text-white bg-slate-900 hover:bg-slate-800 border border-slate-800 rounded-lg transition flex items-center gap-1.5 shadow-sm"
        title="URL History (SQLite)"
      >
        <History size={15} class="text-amber-400" />
        <span class="hidden sm:inline">URL History</span>
      </button>

      <button 
        on:click={() => { initialUrlForModal = ''; isAddModalOpen = true; }}
        class="px-3.5 py-2 text-xs font-semibold text-slate-950 bg-cyan-400 hover:bg-cyan-300 rounded-lg transition flex items-center gap-1.5 shadow-lg shadow-cyan-500/20 active:scale-95"
      >
        <Plus size={16} />
        <span>New Download</span>
      </button>
    </div>
  </header>

  <!-- Controls Bar: Filter Tabs & Search -->
  <div class="px-6 py-3 border-b border-slate-800/80 bg-slate-950/40 flex flex-wrap items-center justify-between gap-3 shrink-0">
    <!-- Filter Tabs -->
    <div class="flex items-center gap-1 bg-slate-900/90 p-1 rounded-lg border border-slate-800 text-xs">
      <button 
        on:click={() => filter = 'all'}
        class="px-3 py-1 rounded-md transition font-medium {filter === 'all' ? 'bg-cyan-500/10 text-cyan-400 border border-cyan-500/30' : 'text-slate-400 hover:text-slate-200'}"
      >
        All ({downloads.length})
      </button>
      <button 
        on:click={() => filter = 'downloading'}
        class="px-3 py-1 rounded-md transition font-medium {filter === 'downloading' ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/30' : 'text-slate-400 hover:text-slate-200'}"
      >
        Downloading ({activeCount})
      </button>
      <button 
        on:click={() => filter = 'paused'}
        class="px-3 py-1 rounded-md transition font-medium {filter === 'paused' ? 'bg-amber-500/10 text-amber-400 border border-amber-500/30' : 'text-slate-400 hover:text-slate-200'}"
      >
        Paused ({pausedCount})
      </button>
      <button 
        on:click={() => filter = 'completed'}
        class="px-3 py-1 rounded-md transition font-medium {filter === 'completed' ? 'bg-blue-500/10 text-blue-400 border border-blue-500/30' : 'text-slate-400 hover:text-slate-200'}"
      >
        Completed ({completedCount})
      </button>
    </div>

    <!-- Search Input -->
    <div class="relative w-64">
      <Search size={14} class="absolute left-3 top-1/2 -translate-y-1/2 text-slate-500" />
      <input 
        type="text" 
        bind:value={searchQuery}
        placeholder="Search downloads..."
        class="w-full bg-slate-900 border border-slate-800 rounded-lg pl-8 pr-3 py-1.5 text-xs text-slate-200 placeholder-slate-500 focus:outline-none focus:border-cyan-500 transition"
      />
    </div>
  </div>

  <!-- Downloads Content Area -->
  <main class="flex-1 overflow-y-auto p-6 space-y-3.5">
    {#if filteredDownloads.length === 0}
      <div class="h-full flex flex-col items-center justify-center py-20 text-center">
        <div class="w-16 h-16 rounded-2xl bg-slate-900 border border-slate-800 flex items-center justify-center text-slate-600 mb-4">
          <Download size={28} />
        </div>
        <h3 class="text-sm font-semibold text-slate-300">
          {searchQuery ? "No downloads matched your search" : "No downloads in this view"}
        </h3>
        <p class="text-xs text-slate-500 max-w-sm mt-1">
          {searchQuery ? "Try checking for typos or clear your search query." : "Click '+ New Download' to start a multi-part file download."}
        </p>
        {#if !searchQuery}
          <button 
            on:click={() => isAddModalOpen = true}
            class="mt-4 px-4 py-2 text-xs font-semibold text-slate-900 bg-cyan-400 hover:bg-cyan-300 rounded-lg transition shadow-md shadow-cyan-500/20"
          >
            Start Your First Download
          </button>
        {/if}
      </div>
    {:else}
      {#each filteredDownloads as download (download.id)}
        <DownloadItem {download} onAction={fetchDownloads} />
      {/each}
    {/if}
  </main>

  <!-- Modals -->
  <AddDownloadModal 
    isOpen={isAddModalOpen} 
    initialUrl={initialUrlForModal}
    onClose={() => isAddModalOpen = false} 
    onAdded={fetchDownloads} 
  />

  <HistoryModal 
    isOpen={isHistoryModalOpen} 
    onClose={() => isHistoryModalOpen = false} 
    onSelectUrl={handleSelectHistoryUrl} 
  />
</div>
