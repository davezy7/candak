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
    File, 
    ChevronDown, 
    ChevronUp, 
    ExternalLink, 
    Check,
    CheckCircle2,
    AlertCircle
  } from 'lucide-svelte';

  export let download;
  export let onAction = () => {};

  let showChunks = false;
  let isActionLoading = false;

  $: category = getFileCategory(download.filename);
  $: progress = download.totalSize > 0 
    ? Math.min(100, Math.max(0, (download.downloadedSize / download.totalSize) * 100))
    : 0;

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

  async function handleDelete() {
    const deleteFromDisk = confirm(`Delete "${download.filename}"?\n\nClick OK to also remove the file from your disk.`);
    try {
      await CancelDownload(download.id, deleteFromDisk);
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

<div class="bg-white rounded-2xl p-5 border border-[#E2E8F0] shadow-sm hover:shadow-md transition-all duration-200 group">
  <!-- Top Row -->
  <div class="flex items-center justify-between gap-4">
    <!-- Icon & Titles -->
    <div class="flex items-center gap-3.5 min-w-0 flex-1">
      <!-- Squircle File Icon -->
      <div class="w-12 h-12 rounded-2xl flex items-center justify-center shrink-0 transition-colors {
        download.status === 'completed' 
          ? 'bg-[#F1F5F9] text-[#64748B]' 
          : 'bg-[#CCFBF1]/60 text-[#0D9488]'
      }">
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
      </div>

      <!-- File Name & Subtitle -->
      <div class="min-w-0 flex-1">
        <h4 class="font-bold text-[#1F2937] text-[15px] truncate leading-tight" title={download.filename}>
          {download.filename}
        </h4>
        
        <p class="text-xs text-[#64748B] mt-1 font-medium flex items-center gap-1.5 truncate">
          {#if download.status === 'downloading'}
            <span>{formatBytes(download.totalSize)}</span>
            <span>·</span>
            <span>{progress.toFixed(0)}% downloaded</span>
          {:else if download.status === 'completed'}
            <span>{formatBytes(download.totalSize)}</span>
            <span>·</span>
            <span class="text-[#0D9488] font-semibold">Completed</span>
          {:else if download.status === 'paused'}
            <span>{formatBytes(download.downloadedSize)} / {formatBytes(download.totalSize)}</span>
            <span>·</span>
            <span class="text-[#64748B] font-medium">Paused</span>
          {:else if download.status === 'error'}
            <span class="text-rose-600 font-medium">Failed</span>
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
        {#if download.status === 'completed'}
          <button 
            on:click={handleOpenFile}
            class="p-2 text-[#64748B] hover:text-[#0D9488] hover:bg-[#F1F5F9] rounded-xl transition"
            title="Open File"
            aria-label="Open File"
          >
            <ExternalLink size={16} />
          </button>
        {/if}

        <button 
          on:click={handleShowFolder}
          class="p-2 text-[#64748B] hover:text-[#0D9488] hover:bg-[#F1F5F9] rounded-xl transition"
          title="Show in Folder"
          aria-label="Show in Folder"
        >
          <Folder size={16} />
        </button>

        <button 
          on:click={handleDelete}
          class="p-2 text-[#64748B] hover:text-rose-600 hover:bg-rose-50 rounded-xl transition"
          title="Delete Download"
          aria-label="Delete Download"
        >
          <Trash2 size={16} />
        </button>
      </div>

      <!-- Primary Circular Status Action Button (matching reference image) -->
      {#if download.status === 'downloading'}
        <button 
          on:click={handlePause}
          disabled={isActionLoading}
          class="w-8 h-8 rounded-full border-2 border-[#0D9488] text-[#0D9488] flex items-center justify-center hover:bg-[#CCFBF1]/30 transition active:scale-95"
          title="Pause Download"
          aria-label="Pause Download"
        >
          <Pause size={14} strokeWidth={2.5} />
        </button>
      {:else if download.status === 'paused' || download.status === 'error'}
        <button 
          on:click={handleResume}
          disabled={isActionLoading}
          class="w-8 h-8 rounded-full border-2 border-[#0D9488] text-[#0D9488] flex items-center justify-center hover:bg-[#CCFBF1]/30 transition active:scale-95"
          title="Resume Download"
          aria-label="Resume Download"
        >
          <Play size={14} strokeWidth={2.5} class="ml-0.5" />
        </button>
      {:else if download.status === 'completed'}
        <div 
          class="w-8 h-8 rounded-full border-2 border-[#0D9488] text-[#0D9488] flex items-center justify-center bg-[#CCFBF1]/20"
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
      <div class="h-1.5 w-full bg-[#E2E8F0] rounded-full overflow-hidden">
        <div 
          class="h-full bg-[#0D9488] rounded-full transition-all duration-300"
          style="width: {progress}%;"
        ></div>
      </div>

      <!-- Stats row matching screenshot -->
      <div class="flex items-center justify-between text-xs font-medium">
        <span class="text-[#64748B]">
          {#if download.status === 'downloading'}
            {formatSpeed(download.speed)}
          {:else}
            Paused
          {/if}
        </span>

        {#if download.status === 'downloading' && download.eta > 0}
          <span class="text-[#0D9488] font-semibold">
            {formatETA(download.eta)}
          </span>
        {/if}
      </div>
    </div>
  {/if}

  <!-- Chunks Drawer Button (Compact) -->
  {#if download.chunks && download.chunks.length > 0}
    <div class="mt-2 pt-2 flex items-center justify-between text-[11px] text-[#64748B]">
      <button 
        on:click={() => showChunks = !showChunks}
        class="hover:text-[#0D9488] flex items-center gap-1 font-medium transition"
      >
        {#if showChunks}
          <ChevronUp size={12} />
          Hide {download.chunks.length} Chunks
        {:else}
          <ChevronDown size={12} />
          View {download.chunks.length} Chunks
        {/if}
      </button>

      <span class="text-[10px] text-[#64748B]/70 truncate max-w-[200px]" title={download.savePath}>
        {download.savePath}
      </span>
    </div>

    {#if showChunks}
      <ChunkVisualizer chunks={download.chunks} totalSize={download.totalSize} />
    {/if}
  {/if}
</div>
