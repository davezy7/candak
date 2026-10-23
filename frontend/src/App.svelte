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
    Search, 
    Folder, 
    Settings, 
    ArrowDown,
    X
  } from 'lucide-svelte';

  let downloads = [];
  let filter = 'all'; // 'all' | 'downloading' | 'completed' | 'paused'
  let searchQuery = '';
  let showSearchBar = false;
  let isAddModalOpen = false;
  let isHistoryModalOpen = false;
  let initialUrlForModal = '';
  let activeTab = 'downloads'; // 'downloads' | 'files' | 'settings'
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

  // Reactive Counts
  $: totalCount = downloads.length;
  $: downloadingCount = downloads.filter(d => d.status === 'downloading').length;
  $: completedCount = downloads.filter(d => d.status === 'completed').length;
  $: pausedCount = downloads.filter(d => d.status === 'paused').length;

  // Filtered List
  $: filteredDownloads = downloads.filter(d => {
    if (filter === 'downloading' && d.status !== 'downloading') return false;
    if (filter === 'completed' && d.status !== 'completed') return false;
    if (filter === 'paused' && d.status !== 'paused') return false;

    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase();
      return d.filename.toLowerCase().includes(q) || d.url.toLowerCase().includes(q);
    }
    return true;
  });
</script>

<div class="flex flex-col h-screen bg-[#F5FAFA] text-[#1F2937] select-none font-sans overflow-hidden">
  <!-- Top Navigation & Header -->
  <header class="px-8 pt-7 pb-4 shrink-0" style="--wails-draggable:drag">
    <div class="flex items-center justify-between">
      <!-- Title -->
      <h1 class="text-[26px] font-bold tracking-tight text-[#1F2937]">Downloads</h1>

      <!-- Header Action Buttons -->
      <div class="flex items-center gap-2.5" style="--wails-draggable:no-drag">
        <!-- Search Toggle -->
        <button 
          on:click={() => showSearchBar = !showSearchBar}
          class="p-2.5 rounded-full text-[#1F2937] hover:text-[#0D9488] hover:bg-white border border-transparent hover:border-[#E2E8F0] shadow-2xs transition"
          title="Search downloads"
          aria-label="Search downloads"
        >
          <Search size={20} strokeWidth={2.2} />
        </button>

        <!-- New Download Button -->
        <button 
          on:click={() => { initialUrlForModal = ''; isAddModalOpen = true; }}
          class="flex items-center gap-1.5 px-4 py-2 bg-[#0D9488] hover:bg-[#0F766E] text-white text-xs font-bold rounded-full shadow-sm hover:shadow transition active:scale-95"
          title="Add New Download"
        >
          <Plus size={16} strokeWidth={3} />
          <span>Add</span>
        </button>
      </div>
    </div>

    <!-- Optional Search Input Field -->
    {#if showSearchBar}
      <div class="mt-3 relative" style="--wails-draggable:no-drag">
        <input 
          type="text" 
          bind:value={searchQuery}
          placeholder="Filter by filename or URL..."
          class="w-full bg-white border border-[#E2E8F0] rounded-2xl pl-10 pr-9 py-2 text-sm text-[#1F2937] placeholder-[#64748B] focus:outline-none focus:border-[#0D9488] focus:ring-2 focus:ring-[#0D9488]/20 transition shadow-xs"
        />
        <Search size={16} class="absolute left-3.5 top-1/2 -translate-y-1/2 text-[#64748B]" />
        {#if searchQuery}
          <button 
            on:click={() => searchQuery = ''}
            class="absolute right-3 top-1/2 -translate-y-1/2 text-[#64748B] hover:text-[#1F2937]"
          >
            <X size={15} />
          </button>
        {/if}
      </div>
    {/if}

    <!-- Filter Pills matching reference screenshot -->
    <div class="flex items-center gap-2 mt-4 overflow-x-auto no-scrollbar" style="--wails-draggable:no-drag">
      <button 
        on:click={() => filter = 'all'}
        class="px-4 py-1.5 rounded-full text-[13px] font-semibold transition-all {
          filter === 'all' 
            ? 'bg-[#CCFBF1] text-[#0D9488]' 
            : 'text-[#64748B] hover:text-[#1F2937] hover:bg-white/60'
        }"
      >
        All files {totalCount}
      </button>

      <button 
        on:click={() => filter = 'downloading'}
        class="px-4 py-1.5 rounded-full text-[13px] font-semibold transition-all {
          filter === 'downloading' 
            ? 'bg-[#CCFBF1] text-[#0D9488]' 
            : 'text-[#64748B] hover:text-[#1F2937] hover:bg-white/60'
        }"
      >
        Downloading {downloadingCount}
      </button>

      <button 
        on:click={() => filter = 'completed'}
        class="px-4 py-1.5 rounded-full text-[13px] font-semibold transition-all {
          filter === 'completed' 
            ? 'bg-[#CCFBF1] text-[#0D9488]' 
            : 'text-[#64748B] hover:text-[#1F2937] hover:bg-white/60'
        }"
      >
        Completed {completedCount}
      </button>

      {#if pausedCount > 0}
        <button 
          on:click={() => filter = 'paused'}
          class="px-4 py-1.5 rounded-full text-[13px] font-semibold transition-all {
            filter === 'paused' 
              ? 'bg-[#CCFBF1] text-[#0D9488]' 
              : 'text-[#64748B] hover:text-[#1F2937] hover:bg-white/60'
          }"
        >
          Paused {pausedCount}
        </button>
      {/if}
    </div>
  </header>

  <!-- Downloads Cards List -->
  <main class="flex-1 overflow-y-auto px-8 pb-24 space-y-3.5">
    {#if filteredDownloads.length === 0}
      <div class="h-full flex flex-col items-center justify-center py-20 text-center">
        <div class="w-16 h-16 rounded-3xl bg-white border border-[#E2E8F0] shadow-xs flex items-center justify-center text-[#64748B] mb-3">
          <Download size={26} strokeWidth={2} />
        </div>
        <h3 class="text-base font-bold text-[#1F2937]">
          {searchQuery ? "No matching downloads" : "No downloads yet"}
        </h3>
        <p class="text-xs text-[#64748B] max-w-xs mt-1">
          {searchQuery ? "Try refining your search query." : "Click '+ Add' to start your first file download."}
        </p>
        {#if !searchQuery}
          <button 
            on:click={() => isAddModalOpen = true}
            class="mt-4 px-5 py-2.5 text-xs font-bold text-white bg-[#0D9488] hover:bg-[#0F766E] rounded-full shadow-sm hover:shadow transition"
          >
            Start a Download
          </button>
        {/if}
      </div>
    {:else}
      {#each filteredDownloads as download (download.id)}
        <DownloadItem {download} onAction={fetchDownloads} />
      {/each}
    {/if}
  </main>

  <!-- Bottom Navigation Bar matching reference screenshot -->
  <footer class="fixed bottom-4 left-0 right-0 px-8 flex justify-center pointer-events-none">
    <nav class="bg-white border border-[#E2E8F0] rounded-3xl shadow-lg shadow-slate-200/50 py-2.5 px-10 flex items-center gap-12 pointer-events-auto">
      <!-- Downloads Tab -->
      <button 
        on:click={() => activeTab = 'downloads'}
        class="flex flex-col items-center gap-1 transition-colors {
          activeTab === 'downloads' ? 'text-[#0D9488]' : 'text-[#64748B] hover:text-[#1F2937]'
        }"
      >
        <ArrowDown size={19} strokeWidth={activeTab === 'downloads' ? 2.6 : 2} />
        <span class="text-[11px] font-semibold">Downloads</span>
      </button>

      <!-- Files / History Tab -->
      <button 
        on:click={() => { isHistoryModalOpen = true; }}
        class="flex flex-col items-center gap-1 transition-colors {
          activeTab === 'files' ? 'text-[#0D9488]' : 'text-[#64748B] hover:text-[#1F2937]'
        }"
      >
        <Folder size={19} strokeWidth={activeTab === 'files' ? 2.6 : 2} />
        <span class="text-[11px] font-semibold">Files</span>
      </button>

      <!-- Settings / Info Tab -->
      <button 
        on:click={() => { isHistoryModalOpen = true; }}
        class="flex flex-col items-center gap-1 transition-colors {
          activeTab === 'settings' ? 'text-[#0D9488]' : 'text-[#64748B] hover:text-[#1F2937]'
        }"
      >
        <Settings size={19} strokeWidth={activeTab === 'settings' ? 2.6 : 2} />
        <span class="text-[11px] font-semibold">Settings</span>
      </button>
    </nav>
  </footer>

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
