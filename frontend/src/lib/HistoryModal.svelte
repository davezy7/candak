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
    class="fixed inset-0 bg-slate-900/40 backdrop-blur-xs z-50 flex items-center justify-center p-4"
    role="presentation"
    on:click|self={onClose}
    on:keydown={handleKeydown}
  >
    <div 
      class="bg-white border border-[#E2E8F0] rounded-3xl w-full max-w-2xl shadow-2xl overflow-hidden animate-in fade-in zoom-in-95 duration-150 flex flex-col max-h-[80vh]"
      role="dialog"
      aria-modal="true"
      aria-labelledby="history-modal-title"
    >
      <!-- Header -->
      <div class="px-6 py-5 border-b border-[#E2E8F0] flex items-center justify-between bg-[#F8FAFC]">
        <div class="flex items-center gap-3">
          <div class="w-10 h-10 rounded-2xl bg-[#CCFBF1] flex items-center justify-center text-[#0D9488]">
            <History size={20} strokeWidth={2.5} />
          </div>
          <div>
            <h3 id="history-modal-title" class="font-bold text-[#1F2937] text-base">URL History</h3>
            <p class="text-xs text-[#64748B]">Past download targets recorded in SQLite</p>
          </div>
        </div>
        <div class="flex items-center gap-2">
          {#if historyItems.length > 0}
            <button 
              on:click={handleClear}
              class="px-3 py-1.5 text-xs text-rose-600 hover:text-rose-700 hover:bg-rose-50 border border-rose-200 rounded-xl transition flex items-center gap-1.5 font-medium"
            >
              <Trash2 size={13} />
              Clear All
            </button>
          {/if}
          <button 
            on:click={onClose}
            class="text-[#64748B] hover:text-[#1F2937] p-1.5 rounded-xl hover:bg-[#E2E8F0]/60 transition"
            aria-label="Close dialog"
          >
            <X size={18} />
          </button>
        </div>
      </div>

      <!-- Content -->
      <div class="p-6 overflow-y-auto flex-1 space-y-2.5">
        {#if isLoading}
          <div class="py-12 text-center text-[#64748B] text-xs">
            <span class="inline-block w-5 h-5 border-2 border-[#0D9488] border-t-transparent rounded-full animate-spin mb-2"></span>
            <div>Loading history...</div>
          </div>
        {:else if historyItems.length === 0}
          <div class="py-16 text-center text-[#64748B] text-xs">
            <History size={36} class="mx-auto mb-2 text-[#CBD5E1]" />
            <div class="text-[#64748B] font-medium">No previous downloads in history yet.</div>
          </div>
        {:else}
          {#each historyItems as item}
            <div class="p-4 bg-[#F8FAFC] hover:bg-[#F1F5F9] border border-[#E2E8F0] rounded-2xl flex items-center justify-between gap-4 transition group">
              <div class="min-w-0 flex-1">
                <div class="font-bold text-sm text-[#1F2937] truncate">
                  {item.filename || 'Unknown filename'}
                </div>
                <div class="text-xs font-mono text-[#0D9488] truncate mt-0.5">
                  {item.url}
                </div>
                <div class="text-[11px] text-[#64748B] mt-1.5 flex items-center gap-3">
                  <span>Last downloaded: {formatDate(item.lastUsedAt)}</span>
                  <span>•</span>
                  <span>Downloaded {item.downloadCount} {item.downloadCount === 1 ? 'time' : 'times'}</span>
                </div>
              </div>

              <div class="flex items-center gap-2 shrink-0">
                <button 
                  on:click={() => copyToClipboard(item)}
                  class="p-2 text-[#64748B] hover:text-[#1F2937] hover:bg-white rounded-xl border border-transparent hover:border-[#E2E8F0] shadow-2xs transition"
                  title="Copy URL"
                  aria-label="Copy URL"
                >
                  {#if copiedId === item.id}
                    <Check size={16} class="text-[#0D9488]" strokeWidth={2.5} />
                  {:else}
                    <Copy size={16} />
                  {/if}
                </button>
                <button 
                  on:click={() => handleReDownload(item.url)}
                  class="px-3 py-1.5 text-xs font-semibold bg-[#CCFBF1] hover:bg-[#0D9488] text-[#0D9488] hover:text-white rounded-xl transition flex items-center gap-1.5 shadow-2xs"
                >
                  <DownloadCloud size={14} />
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
