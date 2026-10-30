<script>
  import { 
    formatBytes, 
    formatSpeed, 
    formatETA, 
    formatDate,
    getExtensionBadge
  } from './formatters.js';
  import {
    sortDownloads,
    filterDownloads,
    updateSelection
  } from './tableUtils.js';
  import { 
    Play, 
    Pause, 
    Folder, 
    Trash2, 
    Plus, 
    Search, 
    ArrowUpDown, 
    Filter, 
    Settings, 
    History, 
    UploadCloud, 
    Gauge, 
    ChevronDown,
    Check, 
    CheckSquare, 
    Square, 
    Copy, 
    FileText,
    AlertCircle,
    Clock,
    XCircle,
    X 
  } from 'lucide-svelte';

  export let downloads = [];
  export let filter = 'all'; // 'all' | 'downloading' | 'completed' | 'paused'
  export let onFilterChange = (f) => {};
  export let onOpenAdd = () => {};
  export let onOpenHistory = () => {};
  export let onOpenSettings = () => {};
  export let onPause = (id) => {};
  export let onResume = (id) => {};
  export let onDelete = (id) => {};
  export let onOpenFile = (id) => {};
  export let onShowInFolder = (id) => {};
  export let selectedIds = new Set();
  export let onSelectionChange = (newSelectedIds, lastId) => {};
  export let onBatchPause = () => {};
  export let onBatchResume = () => {};
  export let onBatchDelete = () => {};
  export let modalOpen = false;

  let searchQuery = '';
  let sortBy = 'newest'; // 'newest' | 'oldest' | 'name' | 'size'
  let isTurtleMode = false;
  let isSortOpen = false;
  let isFilterOpen = false;
  let lastSelectedId = null;

  $: if (selectedIds.size === 0) lastSelectedId = null;

  let contextMenu = {
    isOpen: false,
    x: 0,
    y: 0,
    download: null,
    isBatch: false
  };

  // Filter & search (Active table view slice)
  $: filteredList = sortDownloads(filterDownloads(downloads, filter, searchQuery), sortBy);

  // Aggregate metrics
  $: totalDownloadSpeed = downloads.reduce((acc, d) => acc + (d.status === 'downloading' ? (d.speed || 0) : 0), 0);
  $: activeCount = downloads.filter(d => d.status === 'downloading').length;
  $: completedCount = downloads.filter(d => d.status === 'completed').length;
  $: pausedCount = downloads.filter(d => d.status === 'paused').length;
  $: latestActive = downloads.find(d => d.status === 'downloading') || downloads[0];

  function handleItemSelect(id, e, isCheckbox = false) {
    const res = updateSelection({
      selectedIds,
      lastSelectedId,
      targetId: id,
      shiftKey: !!e?.shiftKey,
      modifierKey: !!(e?.metaKey || e?.ctrlKey),
      isCheckbox,
      visibleList: filteredList
    });
    lastSelectedId = res.nextLastId;
    onSelectionChange(res.nextSelectedIds, res.nextLastId);
  }

  function handleSelectAllToggle() {
    if (selectedIds.size >= filteredList.length && filteredList.length > 0) {
      onSelectionChange(new Set(), null);
    } else {
      const next = new Set(filteredList.map(d => d.id));
      onSelectionChange(next, filteredList.length > 0 ? filteredList[filteredList.length - 1].id : null);
    }
  }

  function handleClearSelection() {
    onSelectionChange(new Set(), null);
  }

  function handleRowClick(dl, e) {
    closeContextMenu();
    handleItemSelect(dl.id, e, false);
  }

  function handleRowDoubleClick(dl) {
    closeContextMenu();
    if (dl.status === 'completed') {
      onOpenFile(dl.id);
    } else if (dl.status === 'downloading') {
      onPause(dl.id);
    } else if (dl.status === 'paused') {
      onResume(dl.id);
    }
  }

  function handleContextMenu(dl, e) {
    e.preventDefault();
    e.stopPropagation();

    const isAlreadySelected = selectedIds.has(dl.id);
    if (!isAlreadySelected && selectedIds.size <= 1) {
      handleItemSelect(dl.id, e, false);
    }

    const isBatch = selectedIds.size > 1 && (isAlreadySelected || selectedIds.has(dl.id));
    const x = Math.min(e.clientX, window.innerWidth - 240);
    const y = Math.min(e.clientY, window.innerHeight - 280);

    contextMenu = {
      isOpen: true,
      x: Math.max(10, x),
      y: Math.max(10, y),
      download: dl,
      isBatch
    };
  }

  function closeContextMenu() {
    contextMenu = {
      isOpen: false,
      x: 0,
      y: 0,
      download: null,
      isBatch: false
    };
  }

  async function copyToClipboard(text) {
    if (!text) return;
    try {
      await navigator.clipboard.writeText(text);
    } catch (err) {
      console.error('Failed to copy text:', err);
    }
  }

  async function copySelectedUrls() {
    const urls = downloads
      .filter(d => selectedIds.has(d.id) && d.url)
      .map(d => d.url)
      .join('\n');
    await copyToClipboard(urls);
  }

  function getProgress(d) {
    if (!d.totalSize || d.totalSize <= 0) return 0;
    return Math.min(100, Math.max(0, (d.downloadedSize / d.totalSize) * 100));
  }

  function handleWindowKeyDown(e) {
    if (modalOpen) return;

    if (e.key === 'Escape') {
      if (contextMenu.isOpen) {
        closeContextMenu();
        return;
      }
      if (isSortOpen || isFilterOpen) {
        isSortOpen = false;
        isFilterOpen = false;
        return;
      }
      if (selectedIds.size > 0) {
        handleClearSelection();
      }
      return;
    }

    if ((e.metaKey || e.ctrlKey) && (e.key === 'a' || e.key === 'A')) {
      const activeEl = document.activeElement;
      const isInput = activeEl && (activeEl.tagName === 'INPUT' || activeEl.tagName === 'TEXTAREA' || activeEl.isContentEditable);
      if (!isInput && filteredList.length > 0) {
        e.preventDefault();
        const next = new Set(filteredList.map(d => d.id));
        const lastId = filteredList[filteredList.length - 1].id;
        onSelectionChange(next, lastId);
      }
    }
  }
</script>

<svelte:window on:keydown={handleWindowKeyDown} />

<div class="flex flex-col h-full bg-[var(--color-bg)] text-[var(--color-text)] font-sans select-none overflow-hidden">
  
  <!-- Top Command Bar -->
  {#if selectedIds.size > 0}
    <!-- Contextual Selection Header (Activated when items are selected) -->
    <header class="px-5 py-2.5 border-b border-[var(--color-accent)]/30 bg-[var(--color-accent-subtle)]/30 shrink-0 flex items-center justify-between gap-3 shadow-xs transition-colors">
      <!-- Left: Checkbox + Count + Deselect button -->
      <div class="flex items-center gap-3">
        <button
          type="button"
          on:click={handleSelectAllToggle}
          class="w-7 h-7 rounded-md border border-[var(--color-accent)] bg-[var(--color-surface)] flex items-center justify-center text-[var(--color-accent)] transition cursor-pointer"
          title={selectedIds.size >= filteredList.length ? "Deselect all visible" : "Select all visible"}
        >
          {#if selectedIds.size >= filteredList.length && filteredList.length > 0}
            <CheckSquare size={15} class="text-[var(--color-accent)]" />
          {:else}
            <span class="w-3 h-0.5 bg-[var(--color-accent)] rounded-full"></span>
          {/if}
        </button>

        <div class="flex items-center gap-2">
          <span class="font-bold text-xs text-[var(--color-text)]">
            {selectedIds.size} {selectedIds.size === 1 ? 'download' : 'downloads'} selected
          </span>
          <span class="text-zinc-400 dark:text-zinc-600">·</span>
          <button
            type="button"
            on:click={handleClearSelection}
            class="text-[11px] text-[var(--color-secondary)] hover:text-[var(--color-text)] underline cursor-pointer"
            title="Deselect all (Esc)"
          >
            Deselect (Esc)
          </button>
        </div>
      </div>

      <!-- Right: Contextual Action Pills -->
      <div class="flex items-center gap-2">
        <button
          type="button"
          on:click={onBatchPause}
          class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] hover:bg-[var(--color-surface-hover)] text-xs font-semibold text-[var(--color-text)] transition cursor-pointer"
          title="Pause selected downloads"
        >
          <Pause size={13} class="text-amber-500" />
          <span>Pause ({selectedIds.size})</span>
        </button>

        <button
          type="button"
          on:click={onBatchResume}
          class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] hover:bg-[var(--color-surface-hover)] text-xs font-semibold text-[var(--color-text)] transition cursor-pointer"
          title="Resume selected downloads"
        >
          <Play size={13} class="text-[var(--color-accent)]" />
          <span>Resume ({selectedIds.size})</span>
        </button>

        <button
          type="button"
          on:click={copySelectedUrls}
          class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] hover:bg-[var(--color-surface-hover)] text-xs font-semibold text-[var(--color-secondary)] hover:text-[var(--color-text)] transition cursor-pointer"
          title="Copy URLs of selected downloads"
        >
          <Copy size={13} />
          <span>Copy URLs</span>
        </button>

        <button
          type="button"
          on:click={onBatchDelete}
          class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-rose-500/10 hover:bg-rose-500/20 border border-rose-500/30 text-xs font-semibold text-rose-500 transition cursor-pointer"
          title="Delete selected downloads"
        >
          <Trash2 size={13} />
          <span>Delete ({selectedIds.size})...</span>
        </button>

        <div class="h-4 w-px bg-[var(--color-border)] mx-1"></div>

        <button
          type="button"
          on:click={handleClearSelection}
          class="p-1.5 rounded-lg text-[var(--color-secondary)] hover:text-[var(--color-text)] hover:bg-[var(--color-surface-hover)] transition cursor-pointer"
          title="Exit multi-select mode (Esc)"
          aria-label="Exit selection"
        >
          <X size={15} />
        </button>
      </div>
    </header>
  {:else}
    <!-- Standard Navigation Header -->
    <header class="px-5 py-3 border-b border-[var(--color-border)] bg-[var(--color-surface)] shrink-0 flex items-center justify-between gap-3 shadow-xs">
      <!-- Left actions: Checkbox, Sort, Filter -->
      <div class="flex items-center gap-2">
        <!-- Select all toggle -->
        <button
          type="button"
          on:click={handleSelectAllToggle}
          class="w-7 h-7 rounded-md border border-[var(--color-border)] hover:border-[var(--color-accent)] bg-[var(--color-bg)] flex items-center justify-center text-[var(--color-secondary)] hover:text-[var(--color-text)] transition cursor-pointer"
          title="Select all"
        >
          {#if selectedIds.size > 0 && selectedIds.size === filteredList.length}
            <CheckSquare size={14} class="text-[var(--color-accent)]" />
          {:else if selectedIds.size > 0}
            <span class="w-2.5 h-0.5 bg-[var(--color-accent)] rounded-full"></span>
          {:else}
            <Square size={14} />
          {/if}
        </button>

        <!-- Sort dropdown button -->
        <div class="relative">
          <button
            type="button"
            on:click={() => { isSortOpen = !isSortOpen; isFilterOpen = false; }}
            class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-[var(--color-border)] bg-[var(--color-bg)] hover:bg-[var(--color-surface-hover)] text-xs font-semibold transition cursor-pointer"
          >
            <ArrowUpDown size={13} class="text-[var(--color-secondary)]" />
            <span class="capitalize">{sortBy}</span>
            <ChevronDown size={12} class="text-[var(--color-secondary)] opacity-70" />
          </button>

          {#if isSortOpen}
            <div class="absolute left-0 mt-1 w-32 py-1 bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl shadow-lg z-30 text-xs">
              {#each ['newest', 'oldest', 'name', 'size'] as opt}
                <button
                  type="button"
                  on:click={() => { sortBy = opt; isSortOpen = false; }}
                  class="w-full text-left px-3 py-1.5 hover:bg-[var(--color-surface-hover)] capitalize flex items-center justify-between {sortBy === opt ? 'font-bold text-[var(--color-accent)]' : ''}"
                >
                  {opt}
                  {#if sortBy === opt}<Check size={12} />{/if}
                </button>
              {/each}
            </div>
          {/if}
        </div>

        <!-- Filter dropdown button -->
        <div class="relative">
          <button
            type="button"
            on:click={() => { isFilterOpen = !isFilterOpen; isSortOpen = false; }}
            class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-[var(--color-border)] bg-[var(--color-bg)] hover:bg-[var(--color-surface-hover)] text-xs font-semibold transition cursor-pointer"
          >
            <Filter size={13} class="text-[var(--color-secondary)]" />
            <span class="capitalize">{filter === 'all' ? 'All files' : filter}</span>
            <ChevronDown size={12} class="text-[var(--color-secondary)] opacity-70" />
          </button>

          {#if isFilterOpen}
            <div class="absolute left-0 mt-1 w-36 py-1 bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl shadow-lg z-30 text-xs">
              {#each [
                { id: 'all', label: `All files (${downloads.length})` },
                { id: 'downloading', label: `Downloading (${activeCount})` },
                { id: 'completed', label: `Completed (${completedCount})` },
                { id: 'paused', label: `Paused (${pausedCount})` }
              ] as f}
                <button
                  type="button"
                  on:click={() => { onFilterChange(f.id); isFilterOpen = false; }}
                  class="w-full text-left px-3 py-1.5 hover:bg-[var(--color-surface-hover)] flex items-center justify-between {filter === f.id ? 'font-bold text-[var(--color-accent)]' : ''}"
                >
                  {f.label}
                  {#if filter === f.id}<Check size={12} />{/if}
                </button>
              {/each}
            </div>
          {/if}
        </div>
      </div>

      <!-- Center/Right controls: + Add Download, Search, Modals -->
      <div class="flex items-center gap-2.5 flex-1 max-w-xl justify-end">
        <!-- Search Bar -->
        <div class="relative w-56 sm:w-64">
          <Search size={13} class="absolute left-2.5 top-1/2 -translate-y-1/2 text-[var(--color-secondary)]" />
          <input
            type="text"
            bind:value={searchQuery}
            placeholder="Search in downloads..."
            class="w-full pl-8 pr-3 py-1.5 bg-[var(--color-bg)] border border-[var(--color-border)] rounded-lg text-xs placeholder:text-[var(--color-secondary)] focus:outline-none focus:border-[var(--color-accent)] transition"
          />
          {#if searchQuery}
            <button
              type="button"
              on:click={() => searchQuery = ''}
              class="absolute right-2 top-1/2 -translate-y-1/2 text-[10px] text-[var(--color-secondary)] hover:text-[var(--color-text)]"
            >
              ✕
            </button>
          {/if}
        </div>

        <!-- History Modal button -->
        <button
          type="button"
          on:click={onOpenHistory}
          class="p-1.5 rounded-lg border border-[var(--color-border)] bg-[var(--color-bg)] hover:bg-[var(--color-surface-hover)] text-[var(--color-secondary)] hover:text-[var(--color-text)] transition cursor-pointer"
          title="Download History"
        >
          <History size={15} />
        </button>

        <!-- Settings Modal button -->
        <button
          type="button"
          on:click={onOpenSettings}
          class="p-1.5 rounded-lg border border-[var(--color-border)] bg-[var(--color-bg)] hover:bg-[var(--color-surface-hover)] text-[var(--color-secondary)] hover:text-[var(--color-text)] transition cursor-pointer"
          title="Settings"
        >
          <Settings size={15} />
        </button>
      </div>
    </header>
  {/if}

  <!-- Table Content Area -->
  <main class="flex-1 overflow-y-auto px-5 py-2 flex flex-col justify-between">
    <div class="w-full">
      <!-- Table Header Row -->
      <div class="grid grid-cols-[28px_32px_minmax(180px,1.8fr)_80px_160px_90px_80px_90px_60px] gap-2 px-3 py-2 text-[11px] font-semibold text-[var(--color-secondary)] border-b border-[var(--color-border)] tracking-wider">
        <div></div>
        <div>#</div>
        <div>Name</div>
        <div>Size</div>
        <div>Status</div>
        <div>Download</div>
        <div>ETA</div>
        <div>Added</div>
        <div class="text-right">Actions</div>
      </div>

      <!-- Table Body Rows -->
      {#if filteredList.length === 0}
        <div class="py-16 text-center text-xs text-[var(--color-secondary)]">
          <p class="font-medium text-sm text-[var(--color-text)]">No downloads found</p>
          <p class="mt-1">Add a URL or drag and drop a file into the zone below.</p>
        </div>
      {:else}
        <div class="divide-y divide-[var(--color-border)]/60">
          {#each filteredList as dl, index (dl.id)}
            {@const isSelected = selectedIds.has(dl.id)}
            {@const pct = getProgress(dl)}

            <div 
              class="group transition-colors {
                isSelected 
                  ? 'bg-[var(--color-accent-subtle)]/30' 
                  : 'hover:bg-[var(--color-surface-hover)]/70'
              }"
              role="button"
              tabindex="0"
              on:click={(e) => handleRowClick(dl, e)}
              on:keydown={(e) => { if (e.key === 'Enter') handleRowDoubleClick(dl); }}
              on:dblclick={() => handleRowDoubleClick(dl)}
              on:contextmenu={(e) => handleContextMenu(dl, e)}
            >
              <!-- Main Row Grid -->
              <div class="grid grid-cols-[28px_32px_minmax(180px,1.8fr)_80px_160px_90px_80px_90px_60px] gap-2 px-3 py-2.5 items-center text-xs cursor-pointer">
                <!-- Checkbox -->
                <div>
                  <button
                    type="button"
                    on:click|stopPropagation={(e) => handleItemSelect(dl.id, e, true)}
                    class="w-4 h-4 rounded border flex items-center justify-center transition cursor-pointer {
                      isSelected 
                        ? 'bg-[var(--color-accent)] border-[var(--color-accent)] text-white' 
                        : 'border-[var(--color-border)] hover:border-[var(--color-accent)] bg-[var(--color-bg)]'
                    }"
                    title={isSelected ? "Deselect" : "Select"}
                  >
                    {#if isSelected}
                      <Check size={11} strokeWidth={3} />
                    {/if}
                  </button>
                </div>

                <!-- Status / Play / Pause inline icon -->
                <div class="flex items-center justify-center">
                  {#if dl.status === 'downloading'}
                    {#if dl.speed === 0 && (!dl.downloadedSize || dl.downloadedSize === 0)}
                      <span class="w-2.5 h-2.5 rounded-full bg-[var(--color-accent)] animate-ping" title="Connecting..."></span>
                    {:else}
                      <button
                        type="button"
                        on:click|stopPropagation={() => onPause(dl.id)}
                        class="text-[var(--color-accent)] hover:opacity-80 transition cursor-pointer"
                        title="Pause"
                      >
                        <Pause size={14} class="fill-current" />
                      </button>
                    {/if}
                  {:else if dl.status === 'paused'}
                    <button
                      type="button"
                      on:click|stopPropagation={() => onResume(dl.id)}
                      class="text-amber-500 hover:opacity-80 transition cursor-pointer"
                      title="Resume"
                    >
                      <Play size={14} class="fill-current" />
                    </button>
                  {:else if dl.status === 'completed'}
                    <button
                      type="button"
                      on:click|stopPropagation={() => onOpenFile(dl.id)}
                      class="text-[var(--color-secondary)] hover:text-[var(--color-accent)] transition cursor-pointer"
                      title="Open file"
                    >
                      <Folder size={14} class="text-[var(--color-accent)]/80 fill-[var(--color-accent)]/20" />
                    </button>
                  {:else if dl.status === 'queued'}
                    <span title="Queued">
                      <Clock size={13} class="text-sky-400" />
                    </span>
                  {:else if dl.status === 'error'}
                    <span title={dl.errorMessage || dl.error || 'Download failed'}>
                      <AlertCircle size={13} class="text-rose-500" />
                    </span>
                  {:else if dl.status === 'cancelled'}
                    <span title="Download cancelled">
                      <XCircle size={13} class="text-zinc-500" />
                    </span>
                  {:else}
                    <span class="w-1.5 h-1.5 rounded-full bg-zinc-400"></span>
                  {/if}
                </div>

                <!-- File Name & Extension Badge -->
                <div class="flex items-center gap-2 min-w-0 pr-2">
                  <span class="truncate font-medium text-[var(--color-text)]" title={dl.filename}>
                    {dl.filename}
                  </span>
                  <span class="px-1.5 py-0.2 text-[9px] font-mono font-bold rounded bg-zinc-800/10 dark:bg-zinc-700/30 text-zinc-600 dark:text-zinc-300 shrink-0">
                    {getExtensionBadge(dl.filename, dl.savePath)}
                  </span>
                </div>

                <!-- Size -->
                <div class="text-[var(--color-secondary)] font-mono text-[11px] truncate">
                  {formatBytes(dl.totalSize || dl.downloadedSize)}
                </div>

                <!-- Status column with progress bar -->
                <div class="flex items-center gap-2">
                  {#if dl.status === 'downloading'}
                    {#if dl.speed === 0 && (!dl.downloadedSize || dl.downloadedSize === 0)}
                      <span class="text-[var(--color-accent)] font-semibold text-[11px] animate-pulse">
                        Connecting...
                      </span>
                    {:else}
                      <div class="flex-1 h-2 rounded-full bg-zinc-200 dark:bg-zinc-800 overflow-hidden">
                        <div 
                          class="h-full bg-[var(--color-accent)] rounded-full transition-all duration-300"
                          style="width: {pct}%"
                        ></div>
                      </div>
                      <span class="font-mono text-[11px] font-bold text-[var(--color-accent)] w-9 text-right">
                        {pct.toFixed(0)}%
                      </span>
                    {/if}
                  {:else if dl.status === 'completed'}
                    <span class="text-emerald-500 dark:text-emerald-400 font-semibold text-[11px] flex items-center gap-1">
                      <Check size={12} strokeWidth={2.5} /> Completed
                    </span>
                  {:else if dl.status === 'paused'}
                    <span class="text-amber-500 font-medium text-[11px]">
                      Paused ({pct.toFixed(0)}%)
                    </span>
                  {:else if dl.status === 'queued'}
                    <span class="text-sky-500 dark:text-sky-400 font-medium text-[11px]">
                      Queued
                    </span>
                  {:else if dl.status === 'error'}
                    <span class="text-rose-500 font-medium text-[11px] truncate max-w-[150px]" title={dl.errorMessage || dl.error || 'Error'}>
                      {dl.errorMessage || dl.error || 'Error'}
                    </span>
                  {:else if dl.status === 'cancelled'}
                    <span class="text-zinc-500 font-medium text-[11px]">
                      Cancelled
                    </span>
                  {:else}
                    <span class="text-[var(--color-secondary)] text-[11px] capitalize">{dl.status}</span>
                  {/if}
                </div>

                <!-- Download Speed -->
                <div class="font-mono text-[11px] text-[var(--color-text)] truncate">
                  {#if dl.status === 'downloading'}
                    {#if dl.speed > 0}
                      {formatSpeed(dl.speed)}
                    {:else}
                      <span class="text-[var(--color-secondary)]">0 B/s</span>
                    {/if}
                  {:else}
                    <span class="text-[var(--color-secondary)]">0 B/s</span>
                  {/if}
                </div>

                <!-- ETA -->
                <div class="font-mono text-[11px] text-[var(--color-secondary)] truncate">
                  {dl.status === 'downloading' && dl.eta > 0 ? formatETA(dl.eta).replace(' remaining', '') : '—'}
                </div>

                <!-- Added date -->
                <div class="text-[11px] text-[var(--color-secondary)] truncate">
                  {formatDate(dl.createdAt || dl.addedAt) || 'Today'}
                </div>

                <!-- Actions Menu -->
                <div class="flex items-center justify-end gap-1">
                  <button
                    type="button"
                    on:click|stopPropagation={() => onShowInFolder(dl.id)}
                    class="p-1 rounded text-[var(--color-secondary)] hover:text-[var(--color-accent)] hover:bg-[var(--color-bg)] transition cursor-pointer"
                    title="Show in folder"
                  >
                    <Folder size={13} />
                  </button>
                  <button
                    type="button"
                    on:click|stopPropagation={() => onDelete(dl.id)}
                    class="p-1 rounded text-[var(--color-secondary)] hover:text-rose-500 hover:bg-rose-500/10 transition cursor-pointer"
                    title="Delete"
                  >
                    <Trash2 size={13} />
                  </button>
                </div>
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </div>

    <!-- Drop Target Zone (anchored at list bottom) -->
    <button 
      type="button"
      on:click={onOpenAdd}
      class="mt-6 mb-2 mx-1 p-5 rounded-xl border-2 border-dashed border-[var(--color-border)] hover:border-[var(--color-accent)]/60 bg-[var(--color-surface)]/50 hover:bg-[var(--color-surface)] flex flex-col items-center justify-center gap-1.5 text-xs text-[var(--color-secondary)] hover:text-[var(--color-text)] transition cursor-pointer group w-[calc(100%-8px)]"
    >
      <UploadCloud size={20} class="text-[var(--color-secondary)] group-hover:text-[var(--color-accent)] group-hover:scale-110 transition" />
      <span class="font-medium">Drag-and-drop URL or file to download</span>
    </button>
  </main>

  <!-- Aggregate Status Bar (Bottom Desktop Bar) -->
  <footer class="px-5 py-2 border-t border-[var(--color-border)] bg-[var(--color-surface)] shrink-0 flex items-center justify-between text-xs font-mono">
    <!-- Left: Turtle / Speed mode & total download rate -->
    <div class="flex items-center gap-3">
      <!-- Turtle mode toggle -->
      <button
        type="button"
        on:click={() => isTurtleMode = !isTurtleMode}
        class="flex items-center gap-1 px-2 py-0.5 rounded text-[10px] font-bold border transition cursor-pointer {
          isTurtleMode 
            ? 'bg-amber-500/20 text-amber-400 border-amber-500/40' 
            : 'bg-[var(--color-bg)] text-[var(--color-secondary)] border-[var(--color-border)] hover:text-[var(--color-text)]'
        }"
        title="Toggle speed limit (Turtle mode)"
      >
        <Gauge size={11} />
        <span>{isTurtleMode ? 'LIMIT ON' : 'LIMIT OFF'}</span>
      </button>

      <!-- Global speed -->
      <div class="flex items-center gap-1.5 text-[var(--color-text)] font-semibold">
        <span class="text-[var(--color-accent)] font-bold">↓</span>
        <span>{formatSpeed(totalDownloadSpeed)}</span>
      </div>

      <!-- Active tasks -->
      <div class="text-[var(--color-secondary)] text-[11px]">
        {activeCount} active · {completedCount} done
      </div>

      <!-- Priority badge -->
      <span class="px-1.5 py-0.2 rounded text-[9px] font-sans font-bold bg-emerald-500/15 text-emerald-500 dark:text-emerald-400">
        HIGH
      </span>
    </div>

    <!-- Right: Current active file summary -->
    <div class="text-[11px] text-[var(--color-secondary)] truncate max-w-sm">
      {#if latestActive && latestActive.status === 'downloading'}
        <span class="text-[var(--color-text)] font-sans truncate">{latestActive.filename}</span>
      {:else}
        <span>All background streams idle</span>
      {/if}
    </div>
  </footer>

  <!-- Desktop Context Menu (Right Click) -->
  {#if contextMenu.isOpen}
    <!-- Click outside overlay -->
    <div 
      class="fixed inset-0 z-50 bg-transparent"
      role="presentation"
      on:click={closeContextMenu}
      on:contextmenu|preventDefault={closeContextMenu}
    ></div>

    <!-- Floating Context Card -->
    <div 
      class="fixed z-50 w-56 py-1.5 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)]/95 shadow-xl backdrop-blur-md text-xs select-none ring-1 ring-black/5"
      style="left: {contextMenu.x}px; top: {contextMenu.y}px;"
      role="menu"
      tabindex="-1"
      on:click|stopPropagation
      on:keydown={(e) => { if (e.key === 'Escape') closeContextMenu(); }}
    >
      {#if contextMenu.isBatch}
        <!-- Multi-select Batch Actions -->
        <div class="px-3 py-1 text-[11px] font-semibold text-[var(--color-secondary)] border-b border-[var(--color-border)]/50 mb-1">
          {selectedIds.size} downloads selected
        </div>

        <button 
          type="button"
          on:click={() => { onBatchPause(); closeContextMenu(); }}
          class="w-full px-3 py-1.5 text-left flex items-center gap-2 hover:bg-[var(--color-surface-hover)] text-[var(--color-text)] transition cursor-pointer"
        >
          <Pause size={13} class="text-amber-500" />
          <span>Pause Selected ({selectedIds.size})</span>
        </button>

        <button 
          type="button"
          on:click={() => { onBatchResume(); closeContextMenu(); }}
          class="w-full px-3 py-1.5 text-left flex items-center gap-2 hover:bg-[var(--color-surface-hover)] text-[var(--color-text)] transition cursor-pointer"
        >
          <Play size={13} class="text-[var(--color-accent)]" />
          <span>Resume Selected ({selectedIds.size})</span>
        </button>

        <button 
          type="button"
          on:click={() => { copySelectedUrls(); closeContextMenu(); }}
          class="w-full px-3 py-1.5 text-left flex items-center gap-2 hover:bg-[var(--color-surface-hover)] text-[var(--color-secondary)] hover:text-[var(--color-text)] transition cursor-pointer"
        >
          <Copy size={13} />
          <span>Copy Selected URLs</span>
        </button>

        <div class="h-px bg-[var(--color-border)]/60 my-1"></div>

        <button 
          type="button"
          on:click={() => { onBatchDelete(); closeContextMenu(); }}
          class="w-full px-3 py-1.5 text-left flex items-center gap-2 hover:bg-rose-500/10 text-rose-500 font-medium transition cursor-pointer"
        >
          <Trash2 size={13} />
          <span>Delete Selected ({selectedIds.size})...</span>
        </button>
      {:else if contextMenu.download}
        <!-- Single Item Actions -->
        {#if contextMenu.download.status === 'completed'}
          <button 
            type="button"
            on:click={() => { onOpenFile(contextMenu.download.id); closeContextMenu(); }}
            class="w-full px-3 py-1.5 text-left flex items-center gap-2 hover:bg-[var(--color-surface-hover)] text-[var(--color-text)] font-semibold transition cursor-pointer"
          >
            <Folder size={13} class="text-[var(--color-accent)]" />
            <span>Open File</span>
          </button>

          <button 
            type="button"
            on:click={() => { onShowInFolder(contextMenu.download.id); closeContextMenu(); }}
            class="w-full px-3 py-1.5 text-left flex items-center gap-2 hover:bg-[var(--color-surface-hover)] text-[var(--color-text)] transition cursor-pointer"
          >
            <Folder size={13} class="text-[var(--color-secondary)]" />
            <span>Reveal in Folder</span>
          </button>
        {:else if contextMenu.download.status === 'downloading'}
          <button 
            type="button"
            on:click={() => { onPause(contextMenu.download.id); closeContextMenu(); }}
            class="w-full px-3 py-1.5 text-left flex items-center gap-2 hover:bg-[var(--color-surface-hover)] text-[var(--color-text)] font-semibold transition cursor-pointer"
          >
            <Pause size={13} class="text-amber-500" />
            <span>Pause Download</span>
          </button>

          <button 
            type="button"
            on:click={() => { onShowInFolder(contextMenu.download.id); closeContextMenu(); }}
            class="w-full px-3 py-1.5 text-left flex items-center gap-2 hover:bg-[var(--color-surface-hover)] text-[var(--color-text)] transition cursor-pointer"
          >
            <Folder size={13} class="text-[var(--color-secondary)]" />
            <span>Show in Folder</span>
          </button>
        {:else if contextMenu.download.status === 'paused'}
          <button 
            type="button"
            on:click={() => { onResume(contextMenu.download.id); closeContextMenu(); }}
            class="w-full px-3 py-1.5 text-left flex items-center gap-2 hover:bg-[var(--color-surface-hover)] text-[var(--color-text)] font-semibold transition cursor-pointer"
          >
            <Play size={13} class="text-[var(--color-accent)]" />
            <span>Resume Download</span>
          </button>

          <button 
            type="button"
            on:click={() => { onShowInFolder(contextMenu.download.id); closeContextMenu(); }}
            class="w-full px-3 py-1.5 text-left flex items-center gap-2 hover:bg-[var(--color-surface-hover)] text-[var(--color-text)] transition cursor-pointer"
          >
            <Folder size={13} class="text-[var(--color-secondary)]" />
            <span>Show in Folder</span>
          </button>
        {/if}

        <div class="h-px bg-[var(--color-border)]/60 my-1"></div>

        <button 
          type="button"
          on:click={() => { copyToClipboard(contextMenu.download.url); closeContextMenu(); }}
          class="w-full px-3 py-1.5 text-left flex items-center gap-2 hover:bg-[var(--color-surface-hover)] text-[var(--color-secondary)] hover:text-[var(--color-text)] transition cursor-pointer"
        >
          <Copy size={13} />
          <span>Copy Download URL</span>
        </button>

        <button 
          type="button"
          on:click={() => { copyToClipboard(contextMenu.download.savePath || contextMenu.download.filename); closeContextMenu(); }}
          class="w-full px-3 py-1.5 text-left flex items-center gap-2 hover:bg-[var(--color-surface-hover)] text-[var(--color-secondary)] hover:text-[var(--color-text)] transition cursor-pointer"
        >
          <FileText size={13} />
          <span>Copy File Path</span>
        </button>

        <div class="h-px bg-[var(--color-border)]/60 my-1"></div>

        <button 
          type="button"
          on:click={() => { onDelete(contextMenu.download.id); closeContextMenu(); }}
          class="w-full px-3 py-1.5 text-left flex items-center gap-2 hover:bg-rose-500/10 text-rose-500 font-medium transition cursor-pointer"
        >
          <Trash2 size={13} />
          <span>Delete Download</span>
        </button>
      {/if}
    </div>
  {/if}
</div>
