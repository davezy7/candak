<script>
  import { 
    formatBytes, 
    formatSpeed, 
    formatETA, 
    formatDate,
    getExtensionBadge,
    getFileCategory
  } from './formatters.js';
  import {
    sortDownloads,
    filterDownloads,
    updateSelection
  } from './tableUtils.js';
  import {
    DOWNLOAD_STATUS,
    FILTER_MODE,
    SORT_MODE
  } from './constants.js';
  import { t, locale } from './i18n/index.js';
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
    ChevronDown,
    Check, 
    CheckSquare, 
    Square, 
    Copy, 
    FileText,
    File,
    FileCode,
    Video,
    Music,
    Image,
    Archive,
    Cpu,
    AlertCircle,
    Clock,
    XCircle,
    X 
  } from 'lucide-svelte';

  export let downloads = [];
  export let filter = FILTER_MODE.ALL;
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
  let sortBy = SORT_MODE.NEWEST;
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
  $: totalDownloadSpeed = downloads.reduce((acc, d) => acc + (d.status === DOWNLOAD_STATUS.DOWNLOADING ? (d.speed || 0) : 0), 0);
  $: activeCount = downloads.filter(d => d.status === DOWNLOAD_STATUS.DOWNLOADING).length;
  $: queuedCount = downloads.filter(d => d.status === DOWNLOAD_STATUS.QUEUED).length;
  $: inProgressCount = activeCount + queuedCount;
  $: completedCount = downloads.filter(d => d.status === DOWNLOAD_STATUS.COMPLETED).length;
  $: pausedCount = downloads.filter(d => d.status === DOWNLOAD_STATUS.PAUSED).length;
  $: latestActive = downloads.find(d => d.status === DOWNLOAD_STATUS.DOWNLOADING) || downloads[0];

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
    if (dl.status === DOWNLOAD_STATUS.COMPLETED) {
      onOpenFile(dl.id);
    } else if (dl.status === DOWNLOAD_STATUS.DOWNLOADING || dl.status === DOWNLOAD_STATUS.QUEUED) {
      onPause(dl.id);
    } else if (dl.status === DOWNLOAD_STATUS.PAUSED) {
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
            {$t('table.selection.selected_count', { count: selectedIds.size })}
          </span>
          <span class="text-zinc-400 dark:text-zinc-600">·</span>
          <button
            type="button"
            on:click={handleClearSelection}
            class="text-[11px] text-[var(--color-secondary)] hover:text-[var(--color-text)] underline cursor-pointer"
            title="{$t('table.selection.deselect_esc')}"
          >
            {$t('table.selection.deselect_esc')}
          </button>
        </div>
      </div>

      <!-- Right: Contextual Action Pills -->
      <div class="flex items-center gap-2">
        <button
          type="button"
          on:click={onBatchPause}
          class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] hover:bg-[var(--color-surface-hover)] text-xs font-semibold text-[var(--color-text)] transition cursor-pointer"
          title="{$t('table.actions.pause_batch', { count: selectedIds.size })}"
        >
          <Pause size={13} class="text-[var(--color-status-paused)]" />
          <span>{$t('table.actions.pause_batch', { count: selectedIds.size })}</span>
        </button>

        <button
          type="button"
          on:click={onBatchResume}
          class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] hover:bg-[var(--color-surface-hover)] text-xs font-semibold text-[var(--color-text)] transition cursor-pointer"
          title="{$t('table.actions.resume_batch', { count: selectedIds.size })}"
        >
          <Play size={13} class="text-[var(--color-accent)]" />
          <span>{$t('table.actions.resume_batch', { count: selectedIds.size })}</span>
        </button>

        <button
          type="button"
          on:click={copySelectedUrls}
          class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] hover:bg-[var(--color-surface-hover)] text-xs font-semibold text-[var(--color-secondary)] hover:text-[var(--color-text)] transition cursor-pointer"
          title="{$t('table.actions.copy_urls')}"
        >
          <Copy size={13} />
          <span>{$t('table.actions.copy_urls')}</span>
        </button>

        <button
          type="button"
          on:click={onBatchDelete}
          class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-rose-500/10 hover:bg-rose-500/20 border border-rose-500/30 text-xs font-semibold text-[var(--color-status-error)] transition cursor-pointer"
          title="{$t('table.actions.delete_batch', { count: selectedIds.size })}"
        >
          <Trash2 size={13} />
          <span>{$t('table.actions.delete_batch', { count: selectedIds.size })}</span>
        </button>

        <div class="h-4 w-px bg-[var(--color-border)] mx-1"></div>

        <button
          type="button"
          on:click={handleClearSelection}
          class="p-1.5 rounded-lg text-[var(--color-secondary)] hover:text-[var(--color-text)] hover:bg-[var(--color-surface-hover)] transition cursor-pointer"
          title="{$t('table.selection.exit_selection')}"
          aria-label="{$t('table.selection.exit_selection')}"
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
          title="{$t('table.selection.select_all')}"
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
            <span class="capitalize">{$t('table.sort.' + sortBy)}</span>
            <ChevronDown size={12} class="text-[var(--color-secondary)] opacity-70" />
          </button>

          {#if isSortOpen}
            <div class="absolute left-0 mt-1 w-32 py-1 bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl shadow-lg z-30 text-xs">
              {#each [SORT_MODE.NEWEST, SORT_MODE.OLDEST, SORT_MODE.NAME, SORT_MODE.SIZE] as opt}
                <button
                  type="button"
                  on:click={() => { sortBy = opt; isSortOpen = false; }}
                  class="w-full text-left px-3 py-1.5 hover:bg-[var(--color-surface-hover)] capitalize flex items-center justify-between {sortBy === opt ? 'font-bold text-[var(--color-accent)]' : ''}"
                >
                  {$t('table.sort.' + opt)}
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
            <span class="capitalize">{filter === FILTER_MODE.ALL ? $t('table.navigation.all_files_count', { count: downloads.length }) : $t('status.' + filter)}</span>
            <ChevronDown size={12} class="text-[var(--color-secondary)] opacity-70" />
          </button>

          {#if isFilterOpen}
            <div class="absolute left-0 mt-1 w-44 py-1 bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl shadow-lg z-30 text-xs">
              {#each [
                { id: FILTER_MODE.ALL, label: $t('table.navigation.all_files_count', { count: downloads.length }) },
                { id: FILTER_MODE.DOWNLOADING, label: $t('table.navigation.downloading_count', { count: inProgressCount }) },
                { id: FILTER_MODE.COMPLETED, label: $t('table.navigation.completed_count', { count: completedCount }) },
                { id: FILTER_MODE.PAUSED, label: $t('table.navigation.paused_count', { count: pausedCount }) }
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

      <!-- Center/Right controls: Search, Modals -->
      <div class="flex items-center gap-2.5 flex-1 max-w-xl justify-end">
        <!-- Search Bar -->
        <div class="relative w-56 sm:w-64">
          <Search size={13} class="absolute left-2.5 top-1/2 -translate-y-1/2 text-[var(--color-secondary)]" />
          <input
            type="text"
            bind:value={searchQuery}
            placeholder="{$t('table.navigation.search_placeholder')}"
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
          title="{$t('table.navigation.history_title')}"
        >
          <History size={15} />
        </button>

        <!-- Settings Modal button -->
        <button
          type="button"
          on:click={onOpenSettings}
          class="p-1.5 rounded-lg border border-[var(--color-border)] bg-[var(--color-bg)] hover:bg-[var(--color-surface-hover)] text-[var(--color-secondary)] hover:text-[var(--color-text)] transition cursor-pointer"
          title="{$t('table.navigation.settings_title')}"
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
      <div class="grid grid-cols-[28px_32px_minmax(180px,1.8fr)_80px_160px_90px_80px_90px_76px] gap-2 px-3 py-2 text-[11px] font-semibold text-[var(--color-secondary)] border-b border-[var(--color-border)] tracking-wider">
        <div></div>
        <div></div>
        <div>{$t('table.columns.name')}</div>
        <div>{$t('table.columns.size')}</div>
        <div>{$t('table.columns.status')}</div>
        <div>{$t('table.columns.speed')}</div>
        <div>{$t('table.columns.eta')}</div>
        <div>{$t('table.columns.added')}</div>
        <div class="text-right">{$t('table.columns.actions')}</div>
      </div>

      <!-- Table Body Rows -->
      {#if filteredList.length === 0}
        <div class="py-16 text-center text-xs text-[var(--color-secondary)]">
          <p class="font-medium text-sm text-[var(--color-text)]">{$t('table.empty.title')}</p>
          <p class="mt-1">{$t('table.empty.subtitle')}</p>
        </div>
      {:else}
        <div class="divide-y divide-[var(--color-border)]/60">
          {#each filteredList as dl, index (dl.id)}
            {@const isSelected = selectedIds.has(dl.id)}
            {@const pct = getProgress(dl)}
            {@const cat = getFileCategory(dl.filename, dl.savePath)}

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
              <div class="grid grid-cols-[28px_32px_minmax(180px,1.8fr)_80px_160px_90px_80px_90px_76px] gap-2 px-3 py-2.5 items-center text-xs cursor-pointer">
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
                    title={isSelected ? $t('table.row_actions.deselect') : $t('table.row_actions.select')}
                  >
                    {#if isSelected}
                      <Check size={11} strokeWidth={3} />
                    {/if}
                  </button>
                </div>

                <!-- File Type Icon -->
                <div class="flex items-center justify-center text-[var(--color-secondary)]" title={cat}>
                  {#if cat === 'video'}
                    <Video size={15} class="text-rose-500 dark:text-rose-400" />
                  {:else if cat === 'audio'}
                    <Music size={15} class="text-amber-500 dark:text-amber-400" />
                  {:else if cat === 'image'}
                    <Image size={15} class="text-emerald-500 dark:text-emerald-400" />
                  {:else if cat === 'document'}
                    <FileText size={15} class="text-sky-500 dark:text-sky-400" />
                  {:else if cat === 'archive'}
                    <Archive size={15} class="text-purple-500 dark:text-purple-400" />
                  {:else if cat === 'code'}
                    <FileCode size={15} class="text-teal-500 dark:text-teal-400" />
                  {:else if cat === 'executable'}
                    <Cpu size={15} class="text-orange-500 dark:text-orange-400" />
                  {:else}
                    <File size={15} />
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
                  {formatBytes(dl.totalSize || dl.downloadedSize, 1, $locale)}
                </div>

                <!-- Status column with progress bar -->
                <div class="flex items-center gap-2">
                  {#if dl.status === DOWNLOAD_STATUS.DOWNLOADING}
                    {#if dl.speed === 0 && (!dl.downloadedSize || dl.downloadedSize === 0)}
                      <span class="text-[var(--color-accent)] font-semibold text-[11px] animate-pulse">
                        {$t('status.connecting')}
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
                  {:else if dl.status === DOWNLOAD_STATUS.COMPLETED}
                    <span class="text-[var(--color-status-completed)] font-semibold text-[11px] flex items-center gap-1">
                      <Check size={12} strokeWidth={2.5} /> {$t('status.completed')}
                    </span>
                  {:else if dl.status === DOWNLOAD_STATUS.PAUSED}
                    <span class="text-[var(--color-status-paused)] font-medium text-[11px]">
                      {$t('status.paused')} ({pct.toFixed(0)}%)
                    </span>
                  {:else if dl.status === DOWNLOAD_STATUS.QUEUED}
                    <span class="text-[var(--color-status-queued)] font-medium text-[11px] flex items-center gap-1">
                      <Clock size={12} strokeWidth={2.2} /> {$t('status.queued')}
                    </span>
                  {:else if dl.status === DOWNLOAD_STATUS.ERROR}
                    <span class="text-[var(--color-status-error)] font-medium text-[11px] truncate max-w-[150px]" title={dl.errorMessage || dl.error || $t('status.error')}>
                      {dl.errorMessage || dl.error || $t('status.error')}
                    </span>
                  {:else if dl.status === DOWNLOAD_STATUS.CANCELLED}
                    <span class="text-[var(--color-status-cancelled)] font-medium text-[11px]">
                      {$t('status.cancelled')}
                    </span>
                  {:else}
                    <span class="text-[var(--color-secondary)] text-[11px] capitalize">{dl.status}</span>
                  {/if}
                </div>

                <!-- Download Speed -->
                <div class="font-mono text-[11px] text-[var(--color-text)] truncate">
                  {#if dl.status === DOWNLOAD_STATUS.DOWNLOADING}
                    {#if dl.speed > 0}
                      {formatSpeed(dl.speed, $locale)}
                    {:else}
                      <span class="text-[var(--color-secondary)]">0 B/s</span>
                    {/if}
                  {:else}
                    <span class="text-[var(--color-secondary)]">0 B/s</span>
                  {/if}
                </div>

                <!-- ETA -->
                <div class="font-mono text-[11px] text-[var(--color-secondary)] truncate">
                  {#if dl.status === DOWNLOAD_STATUS.DOWNLOADING && dl.eta > 0}
                    {formatETA(dl.eta).replace(' remaining', '')}
                  {:else if dl.status === DOWNLOAD_STATUS.QUEUED}
                    {$t('table.queued_waiting')}
                  {:else}
                    —
                  {/if}
                </div>

                <!-- Added date -->
                <div class="text-[11px] text-[var(--color-secondary)] truncate">
                  {formatDate(dl.createdAt || dl.addedAt, $locale) || $t('common.today')}
                </div>

                <!-- Actions Menu -->
                <div class="flex items-center justify-end gap-1">
                  {#if dl.status === DOWNLOAD_STATUS.DOWNLOADING || dl.status === DOWNLOAD_STATUS.QUEUED}
                    <button
                      type="button"
                      on:click|stopPropagation={() => onPause(dl.id)}
                      class="p-1 rounded text-[var(--color-status-paused)] hover:opacity-80 transition cursor-pointer"
                      title="{$t('table.row_actions.pause')}"
                    >
                      <Pause size={13} class="fill-current" />
                    </button>
                  {:else if dl.status === DOWNLOAD_STATUS.PAUSED}
                    <button
                      type="button"
                      on:click|stopPropagation={() => onResume(dl.id)}
                      class="p-1 rounded text-[var(--color-accent)] hover:opacity-80 transition cursor-pointer"
                      title="{$t('table.row_actions.resume')}"
                    >
                      <Play size={13} class="fill-current" />
                    </button>
                  {:else if dl.status === DOWNLOAD_STATUS.COMPLETED}
                    <button
                      type="button"
                      on:click|stopPropagation={() => onOpenFile(dl.id)}
                      class="p-1 rounded text-[var(--color-secondary)] hover:text-[var(--color-accent)] transition cursor-pointer"
                      title="{$t('table.row_actions.open_file')}"
                    >
                      <Folder size={13} class="text-[var(--color-accent)]" />
                    </button>
                  {/if}
                  <button
                    type="button"
                    on:click|stopPropagation={() => onShowInFolder(dl.id)}
                    class="p-1 rounded text-[var(--color-secondary)] hover:text-[var(--color-accent)] hover:bg-[var(--color-bg)] transition cursor-pointer"
                    title="{$t('table.row_actions.show_folder')}"
                  >
                    <Folder size={13} />
                  </button>
                  <button
                    type="button"
                    on:click|stopPropagation={() => onDelete(dl.id)}
                    class="p-1 rounded text-[var(--color-secondary)] hover:text-[var(--color-status-error)] hover:bg-rose-500/10 transition cursor-pointer"
                    title="{$t('table.row_actions.delete')}"
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
      <span class="font-medium">{$t('table.drop_zone.title')}</span>
    </button>
  </main>

  <!-- Aggregate Status Bar (Bottom Desktop Bar) -->
  <footer class="px-5 py-2 border-t border-[var(--color-border)] bg-[var(--color-surface)] shrink-0 flex items-center justify-between text-xs font-mono">
    <!-- Left: Global speed & active tasks -->
    <div class="flex items-center gap-3">
      <!-- Global speed -->
      <div class="flex items-center gap-1.5 text-[var(--color-text)] font-semibold">
        <span class="text-[var(--color-accent)] font-bold">↓</span>
        <span>{formatSpeed(totalDownloadSpeed, $locale)}</span>
      </div>

      <!-- Active tasks -->
      <div class="text-[var(--color-secondary)] text-[11px]">
        {$t('table.footer.active_done_summary', { active: activeCount, done: completedCount })}
      </div>
    </div>

    <!-- Right: Current active file summary -->
    <div class="text-[11px] text-[var(--color-secondary)] truncate max-w-sm">
      {#if latestActive && latestActive.status === DOWNLOAD_STATUS.DOWNLOADING}
        <span class="text-[var(--color-text)] font-sans truncate">{latestActive.filename}</span>
      {:else}
        <span>{$t('table.footer.idle_message')}</span>
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
          {$t('context_menu.selected_count', { count: selectedIds.size })}
        </div>

        <button 
          type="button"
          on:click={() => { onBatchPause(); closeContextMenu(); }}
          class="w-full px-3 py-1.5 text-left flex items-center gap-2 hover:bg-[var(--color-surface-hover)] text-[var(--color-text)] transition cursor-pointer"
        >
          <Pause size={13} class="text-[var(--color-status-paused)]" />
          <span>{$t('context_menu.pause_selected', { count: selectedIds.size })}</span>
        </button>

        <button 
          type="button"
          on:click={() => { onBatchResume(); closeContextMenu(); }}
          class="w-full px-3 py-1.5 text-left flex items-center gap-2 hover:bg-[var(--color-surface-hover)] text-[var(--color-text)] transition cursor-pointer"
        >
          <Play size={13} class="text-[var(--color-accent)]" />
          <span>{$t('context_menu.resume_selected', { count: selectedIds.size })}</span>
        </button>

        <button 
          type="button"
          on:click={() => { copySelectedUrls(); closeContextMenu(); }}
          class="w-full px-3 py-1.5 text-left flex items-center gap-2 hover:bg-[var(--color-surface-hover)] text-[var(--color-secondary)] hover:text-[var(--color-text)] transition cursor-pointer"
        >
          <Copy size={13} />
          <span>{$t('context_menu.copy_selected_urls')}</span>
        </button>

        <div class="h-px bg-[var(--color-border)]/60 my-1"></div>

        <button 
          type="button"
          on:click={() => { onBatchDelete(); closeContextMenu(); }}
          class="w-full px-3 py-1.5 text-left flex items-center gap-2 hover:bg-rose-500/10 text-[var(--color-status-error)] font-medium transition cursor-pointer"
        >
          <Trash2 size={13} />
          <span>{$t('context_menu.delete_selected', { count: selectedIds.size })}</span>
        </button>
      {:else if contextMenu.download}
        <!-- Single Item Actions -->
        {#if contextMenu.download.status === DOWNLOAD_STATUS.COMPLETED}
          <button 
            type="button"
            on:click={() => { onOpenFile(contextMenu.download.id); closeContextMenu(); }}
            class="w-full px-3 py-1.5 text-left flex items-center gap-2 hover:bg-[var(--color-surface-hover)] text-[var(--color-text)] font-semibold transition cursor-pointer"
          >
            <Folder size={13} class="text-[var(--color-accent)]" />
            <span>{$t('context_menu.open_file')}</span>
          </button>

          <button 
            type="button"
            on:click={() => { onShowInFolder(contextMenu.download.id); closeContextMenu(); }}
            class="w-full px-3 py-1.5 text-left flex items-center gap-2 hover:bg-[var(--color-surface-hover)] text-[var(--color-text)] transition cursor-pointer"
          >
            <Folder size={13} class="text-[var(--color-secondary)]" />
            <span>{$t('context_menu.reveal_folder')}</span>
          </button>
        {:else if contextMenu.download.status === DOWNLOAD_STATUS.DOWNLOADING || contextMenu.download.status === DOWNLOAD_STATUS.QUEUED}
          <button 
            type="button"
            on:click={() => { onPause(contextMenu.download.id); closeContextMenu(); }}
            class="w-full px-3 py-1.5 text-left flex items-center gap-2 hover:bg-[var(--color-surface-hover)] text-[var(--color-text)] font-semibold transition cursor-pointer"
          >
            <Pause size={13} class="text-[var(--color-status-paused)]" />
            <span>{$t('context_menu.pause_download')}</span>
          </button>

          <button 
            type="button"
            on:click={() => { onShowInFolder(contextMenu.download.id); closeContextMenu(); }}
            class="w-full px-3 py-1.5 text-left flex items-center gap-2 hover:bg-[var(--color-surface-hover)] text-[var(--color-text)] transition cursor-pointer"
          >
            <Folder size={13} class="text-[var(--color-secondary)]" />
            <span>{$t('context_menu.show_in_folder')}</span>
          </button>
        {:else if contextMenu.download.status === DOWNLOAD_STATUS.PAUSED}
          <button 
            type="button"
            on:click={() => { onResume(contextMenu.download.id); closeContextMenu(); }}
            class="w-full px-3 py-1.5 text-left flex items-center gap-2 hover:bg-[var(--color-surface-hover)] text-[var(--color-text)] font-semibold transition cursor-pointer"
          >
            <Play size={13} class="text-[var(--color-accent)]" />
            <span>{$t('context_menu.resume_download')}</span>
          </button>

          <button 
            type="button"
            on:click={() => { onShowInFolder(contextMenu.download.id); closeContextMenu(); }}
            class="w-full px-3 py-1.5 text-left flex items-center gap-2 hover:bg-[var(--color-surface-hover)] text-[var(--color-text)] transition cursor-pointer"
          >
            <Folder size={13} class="text-[var(--color-secondary)]" />
            <span>{$t('context_menu.show_in_folder')}</span>
          </button>
        {/if}

        <div class="h-px bg-[var(--color-border)]/60 my-1"></div>

        <button 
          type="button"
          on:click={() => { copyToClipboard(contextMenu.download.url); closeContextMenu(); }}
          class="w-full px-3 py-1.5 text-left flex items-center gap-2 hover:bg-[var(--color-surface-hover)] text-[var(--color-secondary)] hover:text-[var(--color-text)] transition cursor-pointer"
        >
          <Copy size={13} />
          <span>{$t('context_menu.copy_download_url')}</span>
        </button>

        <button 
          type="button"
          on:click={() => { copyToClipboard(contextMenu.download.savePath || contextMenu.download.filename); closeContextMenu(); }}
          class="w-full px-3 py-1.5 text-left flex items-center gap-2 hover:bg-[var(--color-surface-hover)] text-[var(--color-secondary)] hover:text-[var(--color-text)] transition cursor-pointer"
        >
          <FileText size={13} />
          <span>{$t('context_menu.copy_file_path')}</span>
        </button>

        <div class="h-px bg-[var(--color-border)]/60 my-1"></div>

        <button 
          type="button"
          on:click={() => { onDelete(contextMenu.download.id); closeContextMenu(); }}
          class="w-full px-3 py-1.5 text-left flex items-center gap-2 hover:bg-rose-500/10 text-[var(--color-status-error)] font-medium transition cursor-pointer"
        >
          <Trash2 size={13} />
          <span>{$t('context_menu.delete_download')}</span>
        </button>
      {/if}
    </div>
  {/if}
</div>
