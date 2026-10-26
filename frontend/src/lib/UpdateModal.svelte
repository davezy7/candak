<script>
  import { onMount, onDestroy } from 'svelte';
  import { EventsOn } from '@wails/runtime/runtime.js';
  import { DownloadUpdate, OpenFile, ShowInFolder } from '@wails/go/main/App.js';
  import { formatBytes, formatDate } from './formatters.js';
  import {
    X,
    Download,
    Rocket,
    ExternalLink,
    FolderOpen,
    CheckCircle2
  } from 'lucide-svelte';

  export let isOpen = false;
  export let check = null;
  export let onClose = () => {};

  let download = null;
  let downloadError = '';
  let startingDownload = false;
  let unlistenProgress = null;
  let unlistenStatus = null;

  onMount(() => {
    unlistenProgress = EventsOn('download:progress', (evt) => {
      if (download && evt.downloadId === download.id) {
        download = {
          ...download,
          downloadedSize: evt.downloadedSize,
          speed: evt.speed,
          eta: evt.eta,
          status: evt.status
        };
      }
    });

    unlistenStatus = EventsOn('download:status_change', (dl) => {
      if (download && dl.id === download.id) {
        download = { ...download, ...dl };
      }
    });
  });

  onDestroy(() => {
    if (unlistenProgress) unlistenProgress();
    if (unlistenStatus) unlistenStatus();
  });

  async function handleDownload() {
    if (!check?.assetUrl || startingDownload) return;
    startingDownload = true;
    downloadError = '';
    try {
      download = await DownloadUpdate(check.assetUrl, check.assetName);
    } catch (err) {
      console.error('Failed to enqueue update download:', err);
      downloadError = 'Could not start the download. Use the GitHub Releases link instead.';
    } finally {
      startingDownload = false;
    }
  }

  $: progress = download && download.totalSize > 0
    ? Math.min(100, (download.downloadedSize / download.totalSize) * 100)
    : 0;
  $: isCompleted = download?.status === 'completed';
  $: isFailed = download?.status === 'error' || download?.status === 'cancelled';

  // Minimal markdown rendering: escape first, then headings, bullets, bold and inline code.
  function renderMarkdown(md) {
    if (!md) return '';
    const escape = (s) => s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
    const inline = (s) => escape(s)
      .replace(/`([^`]+)`/g, '<code class="px-1 py-0.5 rounded bg-[var(--color-bg)] font-mono text-[10px]">$1</code>')
      .replace(/\*\*([^*]+)\*\*/g, '<strong class="text-[var(--color-text)]">$1</strong>');
    return md.split(/\r?\n/).map((line) => {
      if (/^#{1,6}\s/.test(line)) {
        return `<div class="font-bold text-[var(--color-text)] mt-2">${inline(line.replace(/^#{1,6}\s/, ''))}</div>`;
      }
      if (/^\s*[-*]\s/.test(line)) {
        return `<div class="flex gap-2"><span class="text-[var(--color-accent)]">•</span><span>${inline(line.replace(/^\s*[-*]\s/, ''))}</span></div>`;
      }
      if (!line.trim()) return '<div class="h-1.5"></div>';
      return `<div>${inline(line)}</div>`;
    }).join('');
  }

  function handleKeydown(event) {
    if (event.key === 'Escape') {
      onClose();
    }
  }
</script>

{#if isOpen && check}
  <div
    class="fixed inset-0 bg-black/50 backdrop-blur-xs z-50 flex items-center justify-center p-4"
    role="presentation"
    on:click|self={onClose}
    on:keydown={handleKeydown}
  >
    <div
      class="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-3xl w-full max-w-lg shadow-2xl overflow-hidden animate-in fade-in zoom-in-95 duration-150 flex flex-col max-h-[85vh]"
      role="dialog"
      aria-modal="true"
      aria-labelledby="update-title"
    >
      <!-- Header -->
      <div class="px-6 py-5 border-b border-[var(--color-border)] flex items-center justify-between bg-[var(--color-bg)]/50">
        <div class="flex items-center gap-3">
          <div class="w-10 h-10 rounded-2xl bg-[var(--color-accent-subtle)] flex items-center justify-center text-[var(--color-accent)]">
            <Rocket size={20} strokeWidth={2.5} />
          </div>
          <div>
            <h3 id="update-title" class="font-bold text-[var(--color-text)] text-base">Update Available</h3>
            <p class="text-xs text-[var(--color-secondary)]">
              v{check.latestVersion}
              {#if check.publishedAt}
                • published {formatDate(check.publishedAt)}
              {/if}
            </p>
          </div>
        </div>
        <button
          on:click={onClose}
          class="text-[var(--color-secondary)] hover:text-[var(--color-text)] p-1.5 rounded-xl hover:bg-[var(--color-border)]/50 transition"
          aria-label="Close update dialog"
        >
          <X size={18} />
        </button>
      </div>

      <!-- Release Notes -->
      <div class="p-6 overflow-y-auto flex-1 text-xs text-[var(--color-secondary)] leading-relaxed">
        <div class="text-[11px] font-bold text-[var(--color-secondary)] uppercase tracking-wider mb-2">Release Notes</div>
        {#if check.releaseNotes}
          {@html renderMarkdown(check.releaseNotes)}
        {:else}
          <p>No release notes provided.</p>
        {/if}
      </div>

      <!-- Footer / Actions -->
      <div class="px-6 py-4 border-t border-[var(--color-border)] bg-[var(--color-bg)]/50 space-y-3">
        {#if downloadError}
          <div class="text-[11px] text-rose-500 font-medium">{downloadError}</div>
        {/if}

        {#if !download}
          {#if check.assetUrl}
            <button
              on:click={handleDownload}
              disabled={startingDownload}
              class="w-full flex items-center justify-center gap-2 px-4 py-2.5 bg-[var(--color-accent)] hover:bg-[var(--color-accent-hover)] text-white text-xs font-bold rounded-xl shadow-sm hover:shadow transition active:scale-[0.98] disabled:opacity-50"
            >
              <Download size={15} strokeWidth={2.5} />
              <span>{startingDownload ? 'Starting...' : 'Download Update via Candak'}</span>
            </button>
          {:else}
            <p class="text-[11px] text-[var(--color-secondary)] text-center">
              No prebuilt installer for this platform in the release.
            </p>
          {/if}
          {#if check.releaseUrl}
            <button
              on:click={() => OpenFile(check.releaseUrl)}
              class="w-full flex items-center justify-center gap-1.5 px-4 py-2 text-[11px] font-semibold text-[var(--color-secondary)] hover:text-[var(--color-accent)] transition"
            >
              <ExternalLink size={12} />
              <span>Open GitHub Releases</span>
            </button>
          {/if}
        {:else if isCompleted}
          <div class="flex items-center gap-2 text-[11px] font-semibold text-emerald-500">
            <CheckCircle2 size={14} />
            <span>Download Complete — Run Installer</span>
          </div>
          <div class="flex gap-2">
            <button
              on:click={() => OpenFile(download.savePath)}
              class="flex-1 flex items-center justify-center gap-2 px-4 py-2.5 bg-[var(--color-accent)] hover:bg-[var(--color-accent-hover)] text-white text-xs font-bold rounded-xl shadow-sm transition active:scale-[0.98]"
            >
              <Rocket size={15} strokeWidth={2.5} />
              <span>Run Installer</span>
            </button>
            <button
              on:click={() => ShowInFolder(download.savePath)}
              class="px-4 py-2.5 flex items-center justify-center gap-2 text-xs font-semibold bg-[var(--color-surface)] hover:bg-[var(--color-bg)] text-[var(--color-text)] border border-[var(--color-border)] rounded-xl transition"
              title="Reveal in Finder / Explorer"
            >
              <FolderOpen size={15} />
              <span>Show in Folder</span>
            </button>
          </div>
        {:else}
          <div class="space-y-2">
            <div class="flex items-center justify-between text-[11px] font-semibold">
              <span class={isFailed ? 'text-rose-500' : 'text-[var(--color-text)]'}>
                {isFailed ? (download.errorMessage || 'Download failed') : `Downloading ${check.assetName}...`}
              </span>
              <span class="text-[var(--color-secondary)]">
                {formatBytes(download.downloadedSize)} / {formatBytes(download.totalSize)}
              </span>
            </div>
            <div class="w-full h-2 rounded-full bg-[var(--color-border)] overflow-hidden">
              <div
                class="h-full rounded-full bg-[var(--color-accent)] transition-all duration-300 {isFailed ? 'bg-rose-500' : ''}"
                style="width: {isFailed ? 100 : progress}%"
              ></div>
            </div>
            <p class="text-[10px] text-[var(--color-secondary)]">
              The download continues in the manager — you can close this window.
            </p>
          </div>
        {/if}
      </div>
    </div>
  </div>
{/if}
