<script>
  import { onMount } from 'svelte';
  import { GetURLHistory, ClearURLHistory } from '@wails/go/main/App.js';
  import { formatDate } from './formatters.js';
  import { X, History, Trash2, DownloadCloud, Copy, Check } from 'lucide-svelte';

  export let isOpen = false;
  export let onClose = () => {};
  export let onSelectUrl = (url) => {};

  let historyItems = [];
  let isLoading = false;
  let copiedId = null;

  $: if (isOpen) {
    loadHistory();
  }

  async function loadHistory() {
    isLoading = true;
    try {
      historyItems = await GetURLHistory() || [];
    } catch (err) {
      console.error(err);
    } finally {
      isLoading = false;
    }
  }

  async function handleClear() {
    if (confirm('Clear all URL history stored in SQLite?')) {
      try {
        await ClearURLHistory();
        historyItems = [];
      } catch (err) {
        console.error(err);
      }
    }
  }

  function copyToClipboard(item) {
    navigator.clipboard.writeText(item.url);
    copiedId = item.id;
    setTimeout(() => {
      copiedId = null;
    }, 2000);
  }

  function handleReDownload(url) {
    onSelectUrl(url);
    onClose();
  }

  function handleKeydown(event) {
    if (event.key === 'Escape') {
      onClose();
    }
  }
</script>

{#if isOpen}
  <div 
    class="fixed inset-0 bg-black/70 backdrop-blur-sm z-50 flex items-center justify-center p-4"
    role="presentation"
    on:click|self={onClose}
    on:keydown={handleKeydown}
  >
    <div 
      class="bg-slate-900 border border-slate-700/80 rounded-xl w-full max-w-2xl shadow-2xl overflow-hidden animate-in fade-in zoom-in-95 duration-150 flex flex-col max-h-[80vh]"
      role="dialog"
      aria-modal="true"
      aria-labelledby="history-modal-title"
    >
      <!-- Header -->
      <div class="px-5 py-4 border-b border-slate-800 flex items-center justify-between bg-slate-950/40">
        <div class="flex items-center gap-2.5">
          <div class="w-8 h-8 rounded-lg bg-amber-500/10 border border-amber-500/20 flex items-center justify-center text-amber-400">
            <History size={18} />
          </div>
          <div>
            <h3 id="history-modal-title" class="font-semibold text-slate-100 text-sm">URL History</h3>
            <p class="text-xs text-slate-400">Past download targets recorded in SQLite database</p>
          </div>
        </div>
        <div class="flex items-center gap-2">
          {#if historyItems.length > 0}
            <button 
              on:click={handleClear}
              class="px-2.5 py-1 text-xs text-rose-400 hover:text-rose-300 hover:bg-rose-950/40 border border-rose-900/40 rounded-md transition flex items-center gap-1"
            >
              <Trash2 size={12} />
              Clear All
            </button>
          {/if}
          <button 
            on:click={onClose}
            class="text-slate-400 hover:text-slate-200 p-1 rounded-md hover:bg-slate-800 transition"
            aria-label="Close dialog"
          >
            <X size={18} />
          </button>
        </div>
      </div>

      <!-- Content -->
      <div class="p-5 overflow-y-auto flex-1 space-y-2">
        {#if isLoading}
          <div class="py-12 text-center text-slate-400 text-xs">
            <span class="inline-block w-4 h-4 border-2 border-cyan-400 border-t-transparent rounded-full animate-spin mb-2"></span>
            <div>Loading history...</div>
          </div>
        {:else if historyItems.length === 0}
          <div class="py-16 text-center text-slate-500 text-xs">
            <History size={32} class="mx-auto mb-2 opacity-40 text-slate-400" />
            <div>No previous downloads recorded in URL history yet.</div>
          </div>
        {:else}
          {#each historyItems as item}
            <div class="p-3 bg-slate-950/50 hover:bg-slate-950/90 border border-slate-800/80 rounded-lg flex items-center justify-between gap-4 transition group">
              <div class="min-w-0 flex-1">
                <div class="font-medium text-xs text-slate-200 truncate">
                  {item.filename || 'Unknown filename'}
                </div>
                <div class="text-[11px] font-mono text-cyan-400/80 truncate mt-0.5">
                  {item.url}
                </div>
                <div class="text-[10px] text-slate-500 mt-1 flex items-center gap-3">
                  <span>Last downloaded: {formatDate(item.lastUsedAt)}</span>
                  <span>•</span>
                  <span>Times: {item.downloadCount}</span>
                </div>
              </div>

              <div class="flex items-center gap-1.5 shrink-0">
                <button 
                  on:click={() => copyToClipboard(item)}
                  class="p-1.5 text-slate-400 hover:text-slate-200 hover:bg-slate-800 rounded-md border border-transparent hover:border-slate-700 transition"
                  title="Copy URL"
                  aria-label="Copy URL"
                >
                  {#if copiedId === item.id}
                    <Check size={14} class="text-emerald-400" />
                  {:else}
                    <Copy size={14} />
                  {/if}
                </button>
                <button 
                  on:click={() => handleReDownload(item.url)}
                  class="px-2.5 py-1 text-xs bg-cyan-950/60 hover:bg-cyan-900/80 text-cyan-300 border border-cyan-800/60 rounded-md transition flex items-center gap-1"
                >
                  <DownloadCloud size={12} />
                  Download
                </button>
              </div>
            </div>
          {/each}
        {/if}
      </div>
    </div>
  </div>
{/if}
