<script>
  import { 
    PauseDownload, 
    ResumeDownload, 
    CancelDownload, 
    OpenFile, 
    ShowInFolder 
  } from '@wails/go/main/App.js';
  import { formatBytes, formatSpeed, formatETA, getFileCategory } from './formatters.js';
  import ChunkVisualizer from './ChunkVisualizer.svelte';
  import ConfirmModal from './ConfirmModal.svelte';
  import { 
    Play, 
    Pause, 
    Trash2, 
    Folder, 
    FileText, 
    FileArchive, 
    FileCode, 
    FileVideo, 
    FileAudio, 
    FileImage, 
    ChevronDown, 
    ChevronUp, 
    ExternalLink, 
    Check
  } from 'lucide-svelte';

  export let download;
  export let onAction = () => {};
  export let isSelected = false;
  export let isMultiSelectMode = false;
  export let isModifierKeyPressed = false;
  export let onToggleSelect = () => {};

  let showChunks = false;
  let isActionLoading = false;
  let isDeleteModalOpen = false;
  let deleteFromDisk = false;

  $: category = getFileCategory(download.filename);
  $: progress = download.totalSize > 0 
    ? Math.min(100, Math.max(0, (download.downloadedSize / download.totalSize) * 100))
    : 0;

  function handleCardClick(e) {
    if (isMultiSelectMode) {
      onToggleSelect(e);
    }
  }

  function handleIconClick(e) {
    if (isMultiSelectMode) {
      onToggleSelect(e);
      return;
    }
    if (download.status === 'completed') {
      handleOpenFile();
    }
  }

  function handleNameClick(e) {
    if (isMultiSelectMode) {
      onToggleSelect(e);
      return;
    }
    if (download.status === 'completed') {
      handleOpenFile();
    }
  }

  async function handlePause() {
    isActionLoading = true;
    try {
      await PauseDownload(download.id);
      onAction();
    } catch (err) {
      console.error(err);
    } finally {
      isActionLoading = false;
    }
  }

  async function handleResume() {
    isActionLoading = true;
    try {
      await ResumeDownload(download.id);
      onAction();
    } catch (err) {
      console.error(err);
    } finally {
      isActionLoading = false;
    }
  }

  function handleDelete() {
    deleteFromDisk = false;
    isDeleteModalOpen = true;
  }

  async function handleConfirmDelete(detail) {
    isDeleteModalOpen = false;
    try {
      await CancelDownload(download.id, detail.checked);
      onAction();
    } catch (err) {
      console.error(err);
    }
  }

  async function handleOpenFile() {
    try {
      await OpenFile(download.savePath);
    } catch (err) {
      console.error(err);
    }
  }

  async function handleShowFolder() {
    try {
      await ShowInFolder(download.savePath);
    } catch (err) {
      console.error(err);
    }
  }
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<!-- svelte-ignore a11y_click_events_have_key_events -->
<div 
  on:click={handleCardClick}
  class="bg-[var(--color-surface)] rounded-2xl p-5 border shadow-xs hover:shadow-md transition-all duration-200 group relative {
    isSelected 
      ? 'border-[var(--color-accent)] ring-1 ring-[var(--color-accent)]/30' 
      : 'border-[var(--color-border)]'
  } {isMultiSelectMode ? 'cursor-pointer select-none' : ''}"
>
  <!-- Top Row -->
  <div class="flex items-center justify-between gap-4">
    <!-- Icon & Titles -->
    <div class="flex items-center gap-3.5 min-w-0 flex-1">
      <!-- Selection Checkbox -->
      <button 
        type="button"
        on:click|stopPropagation={(e) => onToggleSelect(e)}
        class="w-5 h-5 rounded-md border flex items-center justify-center shrink-0 transition-all cursor-pointer {
          isMultiSelectMode || isModifierKeyPressed || isSelected 
            ? 'opacity-100' 
            : 'opacity-0 group-hover:opacity-100'
        } {
          isSelected 
            ? 'bg-[var(--color-accent)] border-[var(--color-accent)] text-white shadow-xs' 
            : 'border-[var(--color-border)] hover:border-[var(--color-accent)] bg-[var(--color-bg)]/80 text-transparent'
        }"
        title={isSelected ? "Deselect" : "Select"}
        aria-label="Select download"
      >
        <Check size={12} strokeWidth={3} class={isSelected ? 'block' : 'opacity-0'} />
      </button>

      <!-- Squircle File Icon -->
      <button 
        type="button"
        on:click|stopPropagation={handleIconClick}
        class="w-12 h-12 rounded-2xl flex items-center justify-center shrink-0 transition-colors {
          download.status === 'completed' 
            ? 'bg-[var(--color-card-icon-muted)] text-[var(--color-secondary)]' 
            : 'bg-[var(--color-accent-subtle)] text-[var(--color-accent)]'
        } {download.status === 'completed' && !isMultiSelectMode ? 'cursor-pointer hover:opacity-80' : 'cursor-default'}"
        title={download.status === 'completed' && !isMultiSelectMode ? "Open File" : undefined}
      >
        {#if category === 'archive'}
          <FileArchive size={22} strokeWidth={2} />
        {:else if category === 'video'}
          <FileVideo size={22} strokeWidth={2} />
        {:else if category === 'audio'}
          <FileAudio size={22} strokeWidth={2} />
        {:else if category === 'image'}
          <FileImage size={22} strokeWidth={2} />
        {:else if category === 'code'}
          <FileCode size={22} strokeWidth={2} />
        {:else}
          <FileText size={22} strokeWidth={2} />
        {/if}
      </button>

      <!-- File Name & Subtitle -->
      <div class="min-w-0 flex-1">
        <h4 class="font-bold text-[var(--color-text)] text-[15px] truncate leading-tight">
          <button 
            type="button"
            on:click|stopPropagation={handleNameClick}
            class="text-left font-bold text-[var(--color-text)] truncate max-w-full {
              download.status === 'completed' && !isMultiSelectMode ? 'cursor-pointer hover:underline' : 'cursor-default'
            }" 
            title={download.filename}
          >
            {download.filename}
          </button>
        </h4>
        
        <p class="text-xs text-[var(--color-secondary)] mt-1 font-medium flex items-center gap-1.5 truncate">
          {#if download.status === 'downloading'}
            <span>{formatBytes(download.totalSize)}</span>
            <span>·</span>
            <span>{progress.toFixed(0)}% downloaded</span>
          {:else if download.status === 'completed'}
            <span>{formatBytes(download.totalSize)}</span>
            <span>·</span>
            <span class="text-[var(--color-accent)] font-semibold">Completed</span>
          {:else if download.status === 'paused'}
            <span>{formatBytes(download.downloadedSize)} / {formatBytes(download.totalSize)}</span>
            <span>·</span>
            <span class="text-[var(--color-secondary)] font-medium">Paused</span>
          {:else if download.status === 'error'}
            <span class="text-rose-500 font-medium">Failed</span>
            {#if download.errorMessage}
              <span class="truncate">· {download.errorMessage}</span>
            {/if}
          {:else}
            <span>{download.status}</span>
          {/if}
        </p>
      </div>
    </div>

    <!-- Actions -->
    <div class="flex items-center gap-2 shrink-0">
      <!-- Hover Actions (Open, Folder, Delete) -->
      <div class="opacity-0 group-hover:opacity-100 transition-opacity flex items-center gap-1">
        {#if !isMultiSelectMode}
          {#if download.status === 'completed'}
            <button 
              on:click|stopPropagation={handleOpenFile}
              class="p-2 text-[var(--color-secondary)] hover:text-[var(--color-accent)] hover:bg-[var(--color-bg)] rounded-xl transition"
              title="Open File"
              aria-label="Open File"
            >
              <ExternalLink size={16} />
            </button>
          {/if}

          <button 
            on:click|stopPropagation={handleShowFolder}
            class="p-2 text-[var(--color-secondary)] hover:text-[var(--color-accent)] hover:bg-[var(--color-bg)] rounded-xl transition"
            title="Show in Folder"
            aria-label="Show in Folder"
          >
            <Folder size={16} />
          </button>
        {/if}

        <button 
          on:click|stopPropagation={handleDelete}
          class="p-2 text-[var(--color-secondary)] hover:text-rose-500 hover:bg-rose-500/10 rounded-xl transition"
          title="Delete Download"
          aria-label="Delete Download"
        >
          <Trash2 size={16} />
        </button>
      </div>

      <!-- Primary Circular Status Action Button -->
      {#if download.status === 'downloading'}
        <button 
          on:click|stopPropagation={handlePause}
          disabled={isActionLoading}
          class="w-8 h-8 rounded-full border-2 border-[var(--color-accent)] text-[var(--color-accent)] flex items-center justify-center hover:bg-[var(--color-accent-subtle)] transition active:scale-95"
          title="Pause Download"
          aria-label="Pause Download"
        >
          <Pause size={14} strokeWidth={2.5} />
        </button>
      {:else if download.status === 'paused' || download.status === 'error'}
        <button 
          on:click|stopPropagation={handleResume}
          disabled={isActionLoading}
          class="w-8 h-8 rounded-full border-2 border-[var(--color-accent)] text-[var(--color-accent)] flex items-center justify-center hover:bg-[var(--color-accent-subtle)] transition active:scale-95"
          title="Resume Download"
          aria-label="Resume Download"
        >
          <Play size={14} strokeWidth={2.5} class="ml-0.5" />
        </button>
      {:else if download.status === 'completed'}
        <div 
          class="w-8 h-8 rounded-full border-2 border-[var(--color-accent)] text-[var(--color-accent)] flex items-center justify-center bg-[var(--color-accent-subtle)]"
          title="Download Completed"
        >
          <Check size={16} strokeWidth={2.5} />
        </div>
      {/if}
    </div>
  </div>

  <!-- Progress Bar (Only visible if downloading or paused) -->
  {#if download.status === 'downloading' || download.status === 'paused'}
    <div class="mt-4 space-y-2">
      <!-- Progress Bar Track -->
      <div class="h-1.5 w-full bg-[var(--color-border)] rounded-full overflow-hidden">
        <div 
          class="h-full bg-[var(--color-accent)] rounded-full transition-all duration-300"
          style="width: {progress}%;"
        ></div>
      </div>

      <!-- Stats row matching reference design -->
      <div class="flex items-center justify-between text-xs font-medium">
        <span class="text-[var(--color-secondary)]">
          {#if download.status === 'downloading'}
            {formatSpeed(download.speed)}
          {:else}
            Paused
          {/if}
        </span>

        {#if download.status === 'downloading' && download.eta > 0}
          <span class="text-[var(--color-accent)] font-semibold">
            {formatETA(download.eta)}
          </span>
        {/if}
      </div>
    </div>
  {/if}

  <!-- Chunks Drawer Button (Compact) -->
  {#if download.chunks && download.chunks.length > 0}
    <div class="mt-2 pt-2 flex items-center justify-between text-[11px] text-[var(--color-secondary)]">
      <button 
        on:click|stopPropagation={() => showChunks = !showChunks}
        class="hover:text-[var(--color-accent)] flex items-center gap-1 font-medium transition"
      >
        {#if showChunks}
          <ChevronUp size={12} />
          Hide {download.chunks.length} Chunks
        {:else}
          <ChevronDown size={12} />
          View {download.chunks.length} Chunks
        {/if}
      </button>

      <span class="text-[10px] text-[var(--color-secondary)]/70 truncate max-w-[200px]" title={download.savePath}>
        {download.savePath}
      </span>
    </div>

    {#if showChunks}
      <ChunkVisualizer chunks={download.chunks} totalSize={download.totalSize} />
    {/if}
  {/if}
</div>

<ConfirmModal
  isOpen={isDeleteModalOpen}
  title="Delete Download"
  message="Are you sure you want to remove this download from your list?"
  submessage={download.filename}
  confirmText="Delete"
  showCheckbox={true}
  checkboxLabel="Also delete downloaded file from disk"
  bind:checkboxChecked={deleteFromDisk}
  onConfirm={handleConfirmDelete}
  onCancel={() => (isDeleteModalOpen = false)}
/>
