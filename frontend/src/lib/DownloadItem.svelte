<script>
  import { 
    PauseDownload, 
    ResumeDownload, 
    CancelDownload, 
    OpenFile, 
    ShowInFolder 
  } from '@wails/go/main/App.js';
  import { 
    formatBytes, 
    formatSpeed, 
    formatETA, 
    getFileCategory,
    getExtensionBadge,
    getFolderPath
  } from './formatters.js';
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
    Check
  } from 'lucide-svelte';

  export let download;
  export let onAction = () => {};
  export let isSelected = false;
  export let isMultiSelectMode = false;
  export let isModifierKeyPressed = false;
  export let onToggleSelect = () => {};

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

{#snippet fileIcon()}
  {#if category === 'archive'}
    <FileArchive size={16} strokeWidth={2} />
  {:else if category === 'video'}
    <FileVideo size={16} strokeWidth={2} />
  {:else if category === 'audio'}
    <FileAudio size={16} strokeWidth={2} />
  {:else if category === 'image'}
    <FileImage size={16} strokeWidth={2} />
  {:else if category === 'code'}
    <FileCode size={16} strokeWidth={2} />
  {:else}
    <FileText size={16} strokeWidth={2} />
  {/if}
{/snippet}

<!-- svelte-ignore a11y_no_static_element_interactions -->
<!-- svelte-ignore a11y_click_events_have_key_events -->
<div 
  on:click={handleCardClick}
  class="bg-[var(--color-surface)] rounded-xl px-3 py-2 border shadow-xs hover:shadow-md transition-all duration-200 group relative overflow-hidden {
    isSelected 
      ? 'border-[var(--color-accent)] ring-1 ring-[var(--color-accent)]/30' 
      : 'border-[var(--color-border)]'
  } {isMultiSelectMode ? 'cursor-pointer select-none' : ''}"
>
{#if download.status === 'completed'}
  <!-- Completed State Layout -->
  <div class="flex items-center justify-between gap-4">
    <!-- Icon & Titles -->
    <div class="flex items-center gap-3 min-w-0 flex-1">
      <!-- Selection Checkbox -->
      <button 
        type="button"
        on:click|stopPropagation={(e) => onToggleSelect(e)}
        class="w-[18px] h-[18px] rounded-md border flex items-center justify-center shrink-0 transition-all cursor-pointer {
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
        <Check size={11} strokeWidth={3} class={isSelected ? 'block' : 'opacity-0'} />
      </button>

      <!-- Squircle File Icon -->
      <button 
        type="button"
        on:click|stopPropagation={handleIconClick}
        class="w-8 h-8 rounded-xl flex items-center justify-center shrink-0 transition-colors bg-[var(--color-card-icon-muted)] text-[var(--color-secondary)] {
          !isMultiSelectMode ? 'cursor-pointer hover:opacity-80' : 'cursor-default'
        }"
        title={!isMultiSelectMode ? "Open File" : undefined}
      >
        {@render fileIcon()}
      </button>

      <!-- File Name & Metadata -->
      <div class="min-w-0 flex-1">
        <div class="flex items-center gap-2 max-w-full">
          <h4 class="font-semibold text-[var(--color-text)] text-sm truncate leading-tight">
            <button 
              type="button"
              on:click|stopPropagation={handleNameClick}
              class="text-left font-semibold text-[var(--color-text)] truncate max-w-full {
                !isMultiSelectMode ? 'cursor-pointer hover:underline' : 'cursor-default'
              }" 
              title={download.filename}
            >
              {download.filename}
            </button>
          </h4>
          <span class="px-1.5 py-0.5 text-[10px] font-bold tracking-wider rounded bg-sky-500/10 text-sky-500 dark:bg-sky-400/10 dark:text-sky-400 shrink-0">
            {getExtensionBadge(download.filename, download.savePath)}
          </span>
        </div>
        
        <p class="text-xs text-[var(--color-secondary)] mt-0.5 font-medium flex items-center gap-1.5 truncate">
          <span>{formatBytes(download.totalSize || download.downloadedSize)}</span>
          {#if getFolderPath(download.savePath)}
            <span>·</span>
            <span class="truncate" title={download.savePath}>
              {getFolderPath(download.savePath)}
            </span>
          {/if}
        </p>
      </div>
    </div>

    <!-- Actions -->
    <div class="flex items-center gap-1 shrink-0">
      {#if !isMultiSelectMode}
        <button 
          type="button"
          on:click|stopPropagation={handleShowFolder}
          class="p-1.5 text-[var(--color-secondary)] hover:text-[var(--color-accent)] hover:bg-[var(--color-bg)] rounded-lg transition cursor-pointer"
          title="Show in Folder"
          aria-label="Show in Folder"
        >
          <Folder size={15} />
        </button>
      {/if}

      <button 
        type="button"
        on:click|stopPropagation={handleDelete}
        class="p-1.5 text-[var(--color-secondary)] hover:text-rose-500 hover:bg-rose-500/10 rounded-lg transition cursor-pointer"
        title="Delete Download"
        aria-label="Delete Download"
      >
        <Trash2 size={15} />
      </button>
    </div>
  </div>
{:else}
  <!-- Active / Paused / Error State Layout -->
  <div class="flex items-center justify-between gap-4">
    <!-- Icon & Titles -->
    <div class="flex items-center gap-3 min-w-0 flex-1">
      <!-- Selection Checkbox -->
      <button 
        type="button"
        on:click|stopPropagation={(e) => onToggleSelect(e)}
        class="w-[18px] h-[18px] rounded-md border flex items-center justify-center shrink-0 transition-all cursor-pointer {
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
        <Check size={11} strokeWidth={3} class={isSelected ? 'block' : 'opacity-0'} />
      </button>

      <!-- Squircle File Icon -->
      <div 
        class="w-8 h-8 rounded-xl flex items-center justify-center shrink-0 transition-colors bg-[var(--color-accent-subtle)] text-[var(--color-accent)]"
      >
        {@render fileIcon()}
      </div>

      <!-- File Name & Subtitle -->
      <div class="min-w-0 flex-1">
        <h4 class="font-semibold text-[var(--color-text)] text-sm truncate leading-tight" title={download.filename}>
          {download.filename}
        </h4>
        
        <p class="text-xs text-[var(--color-secondary)] mt-0.5 font-medium flex items-center gap-1.5 truncate">
          {#if download.status === 'downloading'}
            <span>{formatBytes(download.downloadedSize)} / {formatBytes(download.totalSize)}</span>
            <span>·</span>
            <span>{progress.toFixed(0)}%</span>
            {#if download.speed > 0}
              <span>·</span>
              <span class="text-[var(--color-text)] font-semibold">{formatSpeed(download.speed)}</span>
            {:else}
              <span>·</span>
              <span class="text-[var(--color-accent)] animate-pulse font-medium">Connecting...</span>
            {/if}
            {#if download.eta > 0}
              <span>·</span>
              <span class="text-[var(--color-accent)] font-semibold">{formatETA(download.eta)}</span>
            {/if}
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

    <!-- Active Controls Slot -->
    <div class="flex items-center gap-1.5 shrink-0">
      <div class="opacity-0 group-hover:opacity-100 transition-opacity flex items-center gap-1">
        {#if !isMultiSelectMode}
          <button 
            type="button"
            on:click|stopPropagation={handleShowFolder}
            class="p-1.5 text-[var(--color-secondary)] hover:text-[var(--color-accent)] hover:bg-[var(--color-bg)] rounded-lg transition cursor-pointer"
            title="Show in Folder"
            aria-label="Show in Folder"
          >
            <Folder size={15} />
          </button>
        {/if}

        <button 
          type="button"
          on:click|stopPropagation={handleDelete}
          class="p-1.5 text-[var(--color-secondary)] hover:text-rose-500 hover:bg-rose-500/10 rounded-lg transition cursor-pointer"
          title="Cancel Download"
          aria-label="Cancel Download"
        >
          <Trash2 size={15} />
        </button>
      </div>

      <!-- Circular Pause / Resume Button -->
      {#if download.status === 'downloading'}
        <button 
          type="button"
          on:click|stopPropagation={handlePause}
          disabled={isActionLoading}
          class="w-7 h-7 rounded-full border-2 border-[var(--color-accent)] text-[var(--color-accent)] flex items-center justify-center hover:bg-[var(--color-accent-subtle)] transition active:scale-95 cursor-pointer disabled:opacity-50"
          title="Pause Download"
          aria-label="Pause Download"
        >
          <Pause size={13} strokeWidth={2.5} />
        </button>
      {:else if download.status === 'paused' || download.status === 'error'}
        <button 
          type="button"
          on:click|stopPropagation={handleResume}
          disabled={isActionLoading}
          class="w-7 h-7 rounded-full border-2 border-[var(--color-accent)] text-[var(--color-accent)] flex items-center justify-center hover:bg-[var(--color-accent-subtle)] transition active:scale-95 cursor-pointer disabled:opacity-50"
          title="Resume Download"
          aria-label="Resume Download"
        >
          <Play size={13} strokeWidth={2.5} class="ml-0.5" />
        </button>
      {/if}
    </div>
  </div>

  <!-- Bottom Accent Progress Bar -->
  {#if download.status === 'downloading' || download.status === 'paused' || download.status === 'error'}
    <div class="absolute bottom-0 left-0 right-0 h-1 bg-[var(--color-border)]/40 overflow-hidden">
      <div 
        class="h-full {download.status === 'error' ? 'bg-rose-500' : 'bg-[var(--color-accent)]'} transition-all duration-300"
        style="width: {progress}%;"
      ></div>
    </div>
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
