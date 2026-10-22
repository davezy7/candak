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
    AlertCircle,
    CheckCircle2,
    RefreshCw
  } from 'lucide-svelte';

  export let download;
  export let onAction = () => {};

  let showChunks = true;
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
    const deleteFromDisk = confirm(`Delete "${download.filename}"?\n\nClick OK to also delete the file on disk, or Cancel to just remove it from the list.`);
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

<div class="bg-slate-900/80 hover:bg-slate-900 border border-slate-800 hover:border-slate-700/80 rounded-xl p-4 transition-all duration-200 shadow-sm hover:shadow-md">
  <div class="flex items-start justify-between gap-3">
    <!-- Icon & Titles -->
    <div class="flex items-start gap-3 min-w-0 flex-1">
      <div class="w-10 h-10 rounded-lg bg-slate-800/80 border border-slate-700/60 flex items-center justify-center text-slate-300 shrink-0 mt-0.5">
        {#if category === 'archive'}
          <FileArchive size={20} class="text-amber-400" />
        {:else if category === 'video'}
          <FileVideo size={20} class="text-rose-400" />
        {:else if category === 'audio'}
          <FileAudio size={20} class="text-purple-400" />
        {:else if category === 'image'}
          <FileImage size={20} class="text-emerald-400" />
        {:else if category === 'code'}
          <FileCode size={20} class="text-cyan-400" />
        {:else}
          <FileText size={20} class="text-blue-400" />
        {/if}
      </div>

      <div class="min-w-0 flex-1">
        <div class="flex items-center gap-2">
          <h4 class="font-semibold text-slate-100 text-sm truncate" title={download.filename}>
            {download.filename}
          </h4>
          
          <!-- Status Badge -->
          {#if download.status === 'downloading'}
            <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-medium bg-emerald-950/60 text-emerald-400 border border-emerald-800/60">
              <span class="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse"></span>
              Downloading
            </span>
          {:else if download.status === 'paused'}
            <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-medium bg-amber-950/60 text-amber-400 border border-amber-800/60">
              Paused
            </span>
          {:else if download.status === 'completed'}
            <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-medium bg-cyan-950/60 text-cyan-400 border border-cyan-800/60">
              <CheckCircle2 size={10} />
              Completed
            </span>
          {:else if download.status === 'error'}
            <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-medium bg-rose-950/60 text-rose-400 border border-rose-800/60">
              <AlertCircle size={10} />
              Error
            </span>
          {:else}
            <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-medium bg-slate-800 text-slate-400">
              {download.status}
            </span>
          {/if}

          {#if download.resumable}
            <span class="text-[10px] font-mono text-slate-400 bg-slate-800/60 px-1.5 py-0.5 rounded border border-slate-700/40">
              Resume OK
            </span>
          {/if}
        </div>

        <p class="text-xs text-slate-400 truncate mt-0.5 font-mono" title={download.url}>
          {download.url}
        </p>
      </div>
    </div>

    <!-- Actions -->
    <div class="flex items-center gap-1.5 shrink-0">
      {#if download.status === 'downloading'}
        <button 
          on:click={handlePause}
          disabled={isActionLoading}
          class="p-2 text-amber-400 hover:text-amber-300 hover:bg-slate-800 rounded-lg border border-slate-800 hover:border-slate-700 transition"
          title="Pause Download"
        >
          <Pause size={16} />
        </button>
      {:else if download.status === 'paused' || download.status === 'error'}
        <button 
          on:click={handleResume}
          disabled={isActionLoading}
          class="p-2 text-emerald-400 hover:text-emerald-300 hover:bg-slate-800 rounded-lg border border-slate-800 hover:border-slate-700 transition"
          title="Resume Download"
        >
          <Play size={16} />
        </button>
      {/if}

      {#if download.status === 'completed'}
        <button 
          on:click={handleOpenFile}
          class="p-2 text-cyan-400 hover:text-cyan-300 hover:bg-slate-800 rounded-lg border border-slate-800 hover:border-slate-700 transition"
          title="Open File"
        >
          <ExternalLink size={16} />
        </button>
      {/if}

      <button 
        on:click={handleShowFolder}
        class="p-2 text-slate-400 hover:text-slate-200 hover:bg-slate-800 rounded-lg border border-slate-800 hover:border-slate-700 transition"
        title="Show in Folder"
      >
        <Folder size={16} />
      </button>

      <button 
        on:click={handleDelete}
        class="p-2 text-rose-400/80 hover:text-rose-300 hover:bg-rose-950/30 rounded-lg border border-transparent hover:border-rose-900/40 transition"
        title="Delete Download"
      >
        <Trash2 size={16} />
      </button>
    </div>
  </div>

  <!-- Master Progress Bar -->
  <div class="mt-3.5 space-y-1.5">
    <div class="flex justify-between items-center text-xs font-mono">
      <div class="text-slate-300 flex items-center gap-2">
        <span>{formatBytes(download.downloadedSize)} / {formatBytes(download.totalSize)}</span>
        {#if download.totalSize > 0}
          <span class="text-slate-500">({progress.toFixed(1)}%)</span>
        {/if}
      </div>

      <div class="flex items-center gap-3 text-slate-400 text-[11px]">
        {#if download.status === 'downloading'}
          <span class="text-cyan-400 font-semibold">{formatSpeed(download.speed)}</span>
          <span>ETA: {formatETA(download.eta)}</span>
        {:else if download.status === 'error' && download.errorMessage}
          <span class="text-rose-400 truncate max-w-xs">{download.errorMessage}</span>
        {/if}
      </div>
    </div>

    <!-- Progress Track -->
    <div class="h-2 w-full bg-slate-950 rounded-full overflow-hidden border border-slate-800/80">
      <div 
        class="h-full transition-all duration-300 rounded-full {
          download.status === 'completed' 
            ? 'bg-cyan-400' 
            : download.status === 'error' 
              ? 'bg-rose-500' 
              : download.status === 'paused'
                ? 'bg-amber-400'
                : 'bg-gradient-to-r from-cyan-500 via-blue-500 to-indigo-500'
        }"
        style="width: {progress}%;"
      ></div>
    </div>
  </div>

  <!-- Chunk Offsets Drawer Toggle -->
  {#if download.chunks && download.chunks.length > 0}
    <div class="mt-2.5 flex items-center justify-between">
      <button 
        on:click={() => showChunks = !showChunks}
        class="text-[11px] text-slate-400 hover:text-cyan-400 flex items-center gap-1 transition"
      >
        {#if showChunks}
          <ChevronUp size={12} />
          Hide Chunk Offsets ({download.chunks.length} parts)
        {:else}
          <ChevronDown size={12} />
          Show Chunk Offsets ({download.chunks.length} parts)
        {/if}
      </button>

      <span class="text-[10px] text-slate-500 font-mono truncate max-w-xs" title={download.savePath}>
        {download.savePath}
      </span>
    </div>

    {#if showChunks}
      <ChunkVisualizer chunks={download.chunks} totalSize={download.totalSize} />
    {/if}
  {/if}
</div>
