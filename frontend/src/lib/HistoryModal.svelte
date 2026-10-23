<script>
  import { onMount } from 'svelte';
  import { GetURLHistory, ClearURLHistory, DeleteURLHistoryItem } from '@wails/go/main/App.js';
  import { formatDate } from './formatters.js';
  import ConfirmModal from './ConfirmModal.svelte';
  import { X, History, Trash2, DownloadCloud, Copy, Check } from 'lucide-svelte';

  export let isOpen = false;
  export let onClose = () => {};
  export let onSelectUrl = (url) => {};

  let historyItems = [];
  let isLoading = false;
  let copiedId = null;
  let itemToDelete = null;
  let isClearAllModalOpen = false;

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

  function handleClear() {
    isClearAllModalOpen = true;
  }

  async function confirmClearAll() {
    isClearAllModalOpen = false;
    try {
      await ClearURLHistory();
      historyItems = [];
    } catch (err) {
      console.error(err);
    }
  }

  function handleDeleteItem(item) {
    itemToDelete = item;
  }

  async function confirmDeleteItem() {
    if (!itemToDelete) return;
    const target = itemToDelete;
    itemToDelete = null;
    try {
      await DeleteURLHistoryItem(target.id);
      historyItems = historyItems.filter(h => h.id !== target.id);
    } catch (err) {
      console.error(err);
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
    class="fixed inset-0 bg-black/50 backdrop-blur-xs z-50 flex items-center justify-center p-4"
    role="presentation"
    on:click|self={onClose}
    on:keydown={handleKeydown}
  >
    <div 
      class="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-3xl w-full max-w-2xl shadow-2xl overflow-hidden animate-in fade-in zoom-in-95 duration-150 flex flex-col max-h-[80vh]"
      role="dialog"
      aria-modal="true"
      aria-labelledby="history-modal-title"
    >
      <!-- Header -->
      <div class="px-6 py-5 border-b border-[var(--color-border)] flex items-center justify-between bg-[var(--color-bg)]/50">
        <div class="flex items-center gap-3">
          <div class="w-10 h-10 rounded-2xl bg-[var(--color-accent-subtle)] flex items-center justify-center text-[var(--color-accent)]">
            <History size={20} strokeWidth={2.5} />
          </div>
          <div>
            <h3 id="history-modal-title" class="font-bold text-[var(--color-text)] text-base">URL History</h3>
            <p class="text-xs text-[var(--color-secondary)]">Past download targets</p>
          </div>
        </div>
        <div class="flex items-center gap-2">
          {#if historyItems.length > 0}
            <button 
              on:click={handleClear}
              class="px-3 py-1.5 text-xs text-rose-500 hover:text-rose-600 hover:bg-rose-500/10 border border-rose-500/30 rounded-xl transition flex items-center gap-1.5 font-medium"
            >
              <Trash2 size={13} />
              Clear All
            </button>
          {/if}
          <button 
            on:click={onClose}
            class="text-[var(--color-secondary)] hover:text-[var(--color-text)] p-1.5 rounded-xl hover:bg-[var(--color-border)]/50 transition"
            aria-label="Close dialog"
          >
            <X size={18} />
          </button>
        </div>
      </div>

      <!-- Content -->
      <div class="p-6 overflow-y-auto flex-1 space-y-2.5">
        {#if isLoading}
          <div class="py-12 text-center text-[var(--color-secondary)] text-xs">
            <span class="inline-block w-5 h-5 border-2 border-[var(--color-accent)] border-t-transparent rounded-full animate-spin mb-2"></span>
            <div>Loading history...</div>
          </div>
        {:else if historyItems.length === 0}
          <div class="py-16 text-center text-[var(--color-secondary)] text-xs">
            <History size={36} class="mx-auto mb-2 opacity-40 text-[var(--color-secondary)]" />
            <div class="font-medium">No previous downloads in history yet.</div>
          </div>
        {:else}
          {#each historyItems as item}
            <div class="p-4 bg-[var(--color-bg)] hover:opacity-90 border border-[var(--color-border)] rounded-2xl flex items-center justify-between gap-4 transition group">
              <div class="min-w-0 flex-1">
                <div class="font-bold text-sm text-[var(--color-text)] truncate">
                  {item.filename || 'Unknown filename'}
                </div>
                <div class="text-xs font-mono text-[var(--color-accent)] truncate mt-0.5">
                  {item.url}
                </div>
                <div class="text-[11px] text-[var(--color-secondary)] mt-1.5 flex items-center gap-3">
                  <span>Last downloaded: {formatDate(item.lastUsedAt)}</span>
                  <span>•</span>
                  <span>Downloaded {item.downloadCount} {item.downloadCount === 1 ? 'time' : 'times'}</span>
                </div>
              </div>

              <div class="flex items-center gap-2 shrink-0">
                <button 
                  on:click={() => copyToClipboard(item)}
                  class="p-2 text-[var(--color-secondary)] hover:text-[var(--color-text)] hover:bg-[var(--color-surface)] rounded-xl border border-transparent hover:border-[var(--color-border)] shadow-2xs transition"
                  title="Copy URL"
                  aria-label="Copy URL"
                >
                  {#if copiedId === item.id}
                    <Check size={16} class="text-[var(--color-accent)]" strokeWidth={2.5} />
                  {:else}
                    <Copy size={16} />
                  {/if}
                </button>
                <button 
                  on:click={() => handleReDownload(item.url)}
                  class="px-3 py-1.5 text-xs font-semibold bg-[var(--color-accent-subtle)] hover:bg-[var(--color-accent)] text-[var(--color-accent)] hover:text-white rounded-xl transition flex items-center gap-1.5 shadow-2xs"
                >
                  <DownloadCloud size={14} />
                  Download
                </button>
                <button 
                  on:click={() => handleDeleteItem(item)}
                  class="p-2 text-[var(--color-secondary)] hover:text-rose-500 hover:bg-rose-500/10 rounded-xl transition"
                  title="Delete from history"
                  aria-label="Delete from history"
                >
                  <Trash2 size={16} />
                </button>
              </div>
            </div>
          {/each}
        {/if}
      </div>
    </div>
  </div>

  <ConfirmModal
    isOpen={!!itemToDelete}
    title="Delete History Item"
    message="Are you sure you want to remove this URL from your history?"
    submessage={itemToDelete?.url}
    confirmText="Delete"
    onConfirm={confirmDeleteItem}
    onCancel={() => (itemToDelete = null)}
  />

  <ConfirmModal
    isOpen={isClearAllModalOpen}
    title="Clear All History"
    message="Are you sure you want to clear all recorded URL download history? This cannot be undone."
    confirmText="Clear All"
    onConfirm={confirmClearAll}
    onCancel={() => (isClearAllModalOpen = false)}
  />
{/if}
