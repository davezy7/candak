<script>
  import { onMount } from 'svelte';
  import { ProbeURL, AddDownload, SelectDirectory, GetDefaultDownloadDir } from '@wails/go/main/App.js';
  import { formatBytes } from './formatters.js';
  import { X, Search, Folder, Zap, AlertCircle, CheckCircle, Link } from 'lucide-svelte';

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
        connections: parseInt(connections, 10) || 4
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

  function handleKeydown(event) {
    if (event.key === 'Escape') {
      handleClose();
    }
  }
</script>

{#if isOpen}
  <!-- Backdrop -->
  <div 
    class="fixed inset-0 bg-slate-900/40 backdrop-blur-xs z-50 flex items-center justify-center p-4"
    role="presentation"
    on:click|self={handleClose}
    on:keydown={handleKeydown}
  >
    <div 
      class="bg-white border border-[#E2E8F0] rounded-3xl w-full max-w-lg shadow-2xl overflow-hidden animate-in fade-in zoom-in-95 duration-150"
      role="dialog"
      aria-modal="true"
      aria-labelledby="modal-title"
    >
      <!-- Modal Header -->
      <div class="px-6 py-5 border-b border-[#E2E8F0] flex items-center justify-between bg-[#F8FAFC]">
        <div class="flex items-center gap-3">
          <div class="w-10 h-10 rounded-2xl bg-[#CCFBF1] flex items-center justify-center text-[#0D9488]">
            <Link size={20} strokeWidth={2.5} />
          </div>
          <div>
            <h3 id="modal-title" class="font-bold text-[#1F2937] text-base">Add New Download</h3>
            <p class="text-xs text-[#64748B]">Multi-thread chunked download with resume support</p>
          </div>
        </div>
        <button 
          on:click={handleClose}
          class="text-[#64748B] hover:text-[#1F2937] p-1.5 rounded-xl hover:bg-[#E2E8F0]/60 transition"
          aria-label="Close dialog"
        >
          <X size={18} />
        </button>
      </div>

      <!-- Modal Body -->
      <form on:submit|preventDefault={handleSubmit} class="p-6 space-y-4">
        <!-- URL Input -->
        <div>
          <label for="download-url" class="block text-xs font-semibold text-[#1F2937] mb-1.5">Download URL</label>
          <div class="relative flex items-center">
            <input 
              id="download-url"
              type="url" 
              bind:value={url}
              on:blur={handleProbe}
              placeholder="https://example.com/file.zip"
              required
              class="w-full bg-[#F5FAFA] border border-[#E2E8F0] rounded-xl pl-3.5 pr-20 py-2.5 text-sm text-[#1F2937] placeholder-[#64748B]/60 focus:outline-none focus:border-[#0D9488] focus:ring-2 focus:ring-[#0D9488]/20 font-mono transition"
            />
            <button 
              type="button"
              on:click={handleProbe}
              disabled={isProbing || !url}
              class="absolute right-1.5 px-3 py-1.5 text-xs bg-white hover:bg-[#F1F5F9] text-[#0D9488] font-medium rounded-lg border border-[#E2E8F0] shadow-2xs transition flex items-center gap-1 disabled:opacity-50"
            >
              {#if isProbing}
                <span class="w-3 h-3 border-2 border-[#0D9488] border-t-transparent rounded-full animate-spin"></span>
              {:else}
                <Search size={12} strokeWidth={2.5} />
              {/if}
              Probe
            </button>
          </div>
        </div>

        <!-- Probe Status Banner -->
        {#if probeResult}
          <div class="bg-[#CCFBF1]/30 border border-[#0D9488]/30 rounded-2xl p-3.5 text-xs space-y-1.5">
            <div class="flex items-center justify-between">
              <span class="text-[#0D9488] font-bold flex items-center gap-1.5">
                <CheckCircle size={15} strokeWidth={2.5} />
                Target Accessible
              </span>
              <span class="font-bold text-[#1F2937]">{formatBytes(probeResult.totalSize)}</span>
            </div>
            <div class="flex items-center justify-between text-[#64748B] text-[11px]">
              <span>Byte-Ranges / Resume:</span>
              <span class={probeResult.resumable ? "text-[#0D9488] font-bold" : "text-amber-600 font-medium"}>
                {probeResult.resumable ? "✓ Supported (Multi-chunk enabled)" : "⚠ Not Supported (Single thread)"}
              </span>
            </div>
          </div>
        {:else if probeError}
          <div class="bg-rose-50 border border-rose-200 rounded-2xl p-3 text-xs text-rose-700 flex items-start gap-2">
            <AlertCircle size={16} class="shrink-0 mt-0.5 text-rose-600" />
            <span class="break-all">{probeError}</span>
          </div>
        {/if}

        <!-- Custom Filename -->
        <div>
          <label for="download-filename" class="block text-xs font-semibold text-[#1F2937] mb-1.5">Save As Filename</label>
          <input 
            id="download-filename"
            type="text" 
            bind:value={filename}
            placeholder={probeResult?.filename || "auto-detected-filename.bin"}
            class="w-full bg-[#F5FAFA] border border-[#E2E8F0] rounded-xl px-3.5 py-2.5 text-sm text-[#1F2937] placeholder-[#64748B]/60 focus:outline-none focus:border-[#0D9488] focus:ring-2 focus:ring-[#0D9488]/20 transition"
          />
        </div>

        <!-- Save Directory -->
        <div>
          <label for="download-dir" class="block text-xs font-semibold text-[#1F2937] mb-1.5">Save Directory</label>
          <div class="flex gap-2">
            <input 
              id="download-dir"
              type="text" 
              bind:value={saveDir}
              placeholder="/Downloads"
              class="w-full bg-[#F5FAFA] border border-[#E2E8F0] rounded-xl px-3.5 py-2.5 text-xs text-[#1F2937] font-mono focus:outline-none focus:border-[#0D9488] focus:ring-2 focus:ring-[#0D9488]/20 transition"
            />
            <button 
              type="button"
              on:click={handleBrowse}
              class="px-3.5 py-2.5 bg-white hover:bg-[#F1F5F9] text-[#1F2937] text-xs font-medium rounded-xl border border-[#E2E8F0] shadow-2xs transition flex items-center gap-1.5 shrink-0"
            >
              <Folder size={15} class="text-[#64748B]" />
              Browse
            </button>
          </div>
        </div>

        <!-- Concurrent Chunks / Connections -->
        <div>
          <div class="flex items-center justify-between mb-1.5">
            <label for="download-connections" class="text-xs font-semibold text-[#1F2937] flex items-center gap-1">
              <Zap size={14} class="text-[#0D9488]" />
              Concurrent Chunk Connections
            </label>
            <span class="text-xs font-bold text-[#0D9488]">{connections} threads</span>
          </div>
          <input 
            id="download-connections"
            type="range" 
            min="1" 
            max="16" 
            step="1" 
            bind:value={connections}
            disabled={probeResult && !probeResult.resumable}
            class="w-full accent-[#0D9488] bg-[#E2E8F0] h-2 rounded-lg cursor-pointer disabled:opacity-40"
          />
          <div class="flex justify-between text-[11px] text-[#64748B] mt-1 font-medium">
            <span>1 (Single)</span>
            <span>4 (Standard)</span>
            <span>8 (Fast)</span>
            <span>16 (Max)</span>
          </div>
        </div>

        <!-- Submit Button -->
        <div class="pt-4 border-t border-[#E2E8F0] flex justify-end gap-2.5">
          <button 
            type="button" 
            on:click={handleClose}
            class="px-4 py-2.5 text-xs font-semibold text-[#64748B] hover:text-[#1F2937] hover:bg-[#F1F5F9] rounded-xl transition"
          >
            Cancel
          </button>
          <button 
            type="submit" 
            disabled={isSubmitting || !url}
            class="px-5 py-2.5 text-xs font-bold text-white bg-[#0D9488] hover:bg-[#0F766E] rounded-xl shadow-sm hover:shadow transition flex items-center gap-2 disabled:opacity-50"
          >
            {#if isSubmitting}
              <span class="w-3.5 h-3.5 border-2 border-white border-t-transparent rounded-full animate-spin"></span>
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
