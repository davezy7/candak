<script>
  import { onMount } from 'svelte';
  import { ProbeURL, AddDownload, SelectDirectory, GetDefaultDownloadDir } from '@wails/go/main/App.js';
  import { formatBytes } from './formatters.js';
  import { X, Search, Folder, Zap, AlertCircle, CheckCircle, Globe, Link } from 'lucide-svelte';

  export let isOpen = false;
  export let onClose = () => {};
  export let onAdded = () => {};
  export let initialUrl = '';

  let url = '';
  let filename = '';
  let saveDir = '';
  let connections = 4;
  let isProbing = false;
  let probeResult = null;
  let probeError = '';
  let isSubmitting = false;

  $: if (isOpen && initialUrl) {
    url = initialUrl;
    handleProbe();
  }

  onMount(async () => {
    try {
      saveDir = await GetDefaultDownloadDir();
    } catch (e) {
      console.error(e);
    }
  });

  async function handleProbe() {
    if (!url || !url.trim().startsWith('http')) return;
    isProbing = true;
    probeError = '';
    probeResult = null;

    try {
      const res = await ProbeURL(url.trim());
      probeResult = res;
      if (res.filename && !filename) {
        filename = res.filename;
      }
    } catch (err) {
      probeError = String(err);
    } finally {
      isProbing = false;
    }
  }

  async function handleBrowse() {
    try {
      const selected = await SelectDirectory();
      if (selected) {
        saveDir = selected;
      }
    } catch (err) {
      console.error(err);
    }
  }

  async function handleSubmit() {
    if (!url.trim()) return;
    isSubmitting = true;

    try {
      await AddDownload({
        url: url.trim(),
        filename: filename.trim(),
        saveDir: saveDir,
        connections: parseInt(connections) || 4
      });
      onAdded();
      handleClose();
    } catch (err) {
      probeError = String(err);
    } finally {
      isSubmitting = false;
    }
  }

  function handleClose() {
    url = '';
    filename = '';
    probeResult = null;
    probeError = '';
    isSubmitting = false;
    onClose();
  }
</script>

{#if isOpen}
  <!-- Backdrop -->
  <div 
    class="fixed inset-0 bg-black/70 backdrop-blur-sm z-50 flex items-center justify-center p-4"
    on:click|self={handleClose}
  >
    <div class="bg-slate-900 border border-slate-700/80 rounded-xl w-full max-w-lg shadow-2xl overflow-hidden animate-in fade-in zoom-in-95 duration-150">
      <!-- Modal Header -->
      <div class="px-5 py-4 border-b border-slate-800 flex items-center justify-between bg-slate-950/40">
        <div class="flex items-center gap-2.5">
          <div class="w-8 h-8 rounded-lg bg-cyan-500/10 border border-cyan-500/20 flex items-center justify-center text-cyan-400">
            <Link size={18} />
          </div>
          <div>
            <h3 class="font-semibold text-slate-100 text-sm">Add New Download</h3>
            <p class="text-xs text-slate-400">Multi-part chunked download with auto-resume</p>
          </div>
        </div>
        <button 
          on:click={handleClose}
          class="text-slate-400 hover:text-slate-200 p-1 rounded-md hover:bg-slate-800 transition"
        >
          <X size={18} />
        </button>
      </div>

      <!-- Modal Body -->
      <form on:submit|preventDefault={handleSubmit} class="p-5 space-y-4">
        <!-- URL Input -->
        <div>
          <label class="block text-xs font-medium text-slate-300 mb-1.5">Download URL</label>
          <div class="relative flex items-center">
            <input 
              type="url" 
              bind:value={url}
              on:blur={handleProbe}
              placeholder="https://example.com/file.zip"
              required
              class="w-full bg-slate-950/70 border border-slate-700/80 rounded-lg pl-3 pr-20 py-2 text-sm text-slate-100 placeholder-slate-500 focus:outline-none focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 font-mono transition"
            />
            <button 
              type="button"
              on:click={handleProbe}
              disabled={isProbing || !url}
              class="absolute right-1.5 px-2.5 py-1 text-xs bg-slate-800 hover:bg-slate-700 text-slate-300 rounded-md border border-slate-700 transition flex items-center gap-1 disabled:opacity-50"
            >
              {#if isProbing}
                <span class="w-3 h-3 border-2 border-cyan-400 border-t-transparent rounded-full animate-spin"></span>
              {:else}
                <Search size={12} />
              {/if}
              Probe
            </button>
          </div>
        </div>

        <!-- Probe Status Banner -->
        {#if probeResult}
          <div class="bg-cyan-950/30 border border-cyan-900/60 rounded-lg p-3 text-xs space-y-1.5">
            <div class="flex items-center justify-between">
              <span class="text-cyan-300 font-medium flex items-center gap-1">
                <CheckCircle size={14} class="text-emerald-400" />
                Target Accessible
              </span>
              <span class="font-mono text-slate-300 font-semibold">{formatBytes(probeResult.totalSize)}</span>
            </div>
            <div class="flex items-center justify-between text-slate-400 text-[11px]">
              <span>Resumable / Byte-Ranges:</span>
              <span class={probeResult.resumable ? "text-emerald-400 font-medium" : "text-amber-400 font-medium"}>
                {probeResult.resumable ? "✓ Supported (Multi-chunk enabled)" : "⚠ Not Supported (Single-thread)"}
              </span>
            </div>
          </div>
        {:else if probeError}
          <div class="bg-rose-950/30 border border-rose-900/60 rounded-lg p-2.5 text-xs text-rose-300 flex items-start gap-2">
            <AlertCircle size={15} class="shrink-0 mt-0.5" />
            <span class="break-all">{probeError}</span>
          </div>
        {/if}

        <!-- Custom Filename -->
        <div>
          <label class="block text-xs font-medium text-slate-300 mb-1.5">Save As Filename</label>
          <input 
            type="text" 
            bind:value={filename}
            placeholder={probeResult?.filename || "auto-detected-filename.bin"}
            class="w-full bg-slate-950/70 border border-slate-700/80 rounded-lg px-3 py-2 text-sm text-slate-100 placeholder-slate-500 focus:outline-none focus:border-cyan-500 transition"
          />
        </div>

        <!-- Save Directory -->
        <div>
          <label class="block text-xs font-medium text-slate-300 mb-1.5">Save Directory</label>
          <div class="flex gap-2">
            <input 
              type="text" 
              bind:value={saveDir}
              placeholder="/Downloads"
              class="w-full bg-slate-950/70 border border-slate-700/80 rounded-lg px-3 py-2 text-xs text-slate-300 font-mono focus:outline-none focus:border-cyan-500 transition"
            />
            <button 
              type="button"
              on:click={handleBrowse}
              class="px-3 py-2 bg-slate-800 hover:bg-slate-700 text-slate-300 text-xs rounded-lg border border-slate-700 transition flex items-center gap-1.5 shrink-0"
            >
              <Folder size={14} />
              Browse
            </button>
          </div>
        </div>

        <!-- Concurrent Chunks / Connections -->
        <div>
          <div class="flex items-center justify-between mb-1.5">
            <label class="text-xs font-medium text-slate-300 flex items-center gap-1">
              <Zap size={14} class="text-amber-400" />
              Concurrent Chunk Connections
            </label>
            <span class="text-xs font-mono text-cyan-400 font-semibold">{connections} threads</span>
          </div>
          <input 
            type="range" 
            min="1" 
            max="16" 
            step="1" 
            bind:value={connections}
            disabled={probeResult && !probeResult.resumable}
            class="w-full accent-cyan-500 bg-slate-800 h-1.5 rounded cursor-pointer disabled:opacity-40"
          />
          <div class="flex justify-between text-[10px] text-slate-500 mt-1 font-mono">
            <span>1 (Single)</span>
            <span>4 (Standard)</span>
            <span>8 (Fast)</span>
            <span>16 (Max)</span>
          </div>
        </div>

        <!-- Submit Button -->
        <div class="pt-3 border-t border-slate-800 flex justify-end gap-2.5">
          <button 
            type="button" 
            on:click={handleClose}
            class="px-4 py-2 text-xs font-medium text-slate-300 hover:text-white bg-slate-800 hover:bg-slate-700 rounded-lg transition"
          >
            Cancel
          </button>
          <button 
            type="submit" 
            disabled={isSubmitting || !url}
            class="px-5 py-2 text-xs font-semibold text-slate-900 bg-cyan-400 hover:bg-cyan-300 rounded-lg shadow-lg shadow-cyan-500/20 transition flex items-center gap-1.5 disabled:opacity-50"
          >
            {#if isSubmitting}
              <span class="w-3.5 h-3.5 border-2 border-slate-900 border-t-transparent rounded-full animate-spin"></span>
              Starting...
            {:else}
              Start Download
            {/if}
          </button>
        </div>
      </form>
    </div>
  </div>
{/if}
