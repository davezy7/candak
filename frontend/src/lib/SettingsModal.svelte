<script>
  import { onMount } from 'svelte';
  import { theme } from './theme.js';
  import { 
    GetDefaultDownloadDir, 
    SelectDirectory, 
    OpenFolder, 
    ClearURLHistory 
  } from '@wails/go/main/App.js';
  import { 
    X, 
    Settings, 
    Sun, 
    Moon, 
    Monitor, 
    Folder, 
    HardDrive, 
    ExternalLink, 
    Trash2, 
    History, 
    Check, 
    Zap 
  } from 'lucide-svelte';

  export let isOpen = false;
  export let onClose = () => {};
  export let onOpenHistory = () => {};

  let currentTheme = 'system';
  let downloadDir = '';
  let defaultConnections = 4;
  let statusMessage = '';

  theme.subscribe((val) => {
    currentTheme = val;
  });

  onMount(async () => {
    try {
      downloadDir = await GetDefaultDownloadDir();
      const savedConn = localStorage.getItem('candak_default_connections');
      if (savedConn) defaultConnections = parseInt(savedConn, 10);
    } catch (e) {
      console.error(e);
    }
  });

  function setTheme(t) {
    theme.set(t);
  }

  async function handleBrowseDir() {
    try {
      const selected = await SelectDirectory();
      if (selected) {
        downloadDir = selected;
        localStorage.setItem('candak_custom_download_dir', selected);
        flashMessage('Default download directory updated');
      }
    } catch (err) {
      console.error(err);
    }
  }

  async function handleOpenDownloads() {
    try {
      await OpenFolder(downloadDir);
    } catch (err) {
      console.error(err);
    }
  }

  function handleConnectionsChange(e) {
    defaultConnections = parseInt(e.target.value, 10);
    localStorage.setItem('candak_default_connections', String(defaultConnections));
  }

  async function handleClearHistory() {
    if (confirm('Are you sure you want to clear all URL history from SQLite?')) {
      try {
        await ClearURLHistory();
        flashMessage('URL history cleared successfully');
      } catch (err) {
        console.error(err);
      }
    }
  }

  function flashMessage(msg) {
    statusMessage = msg;
    setTimeout(() => {
      statusMessage = '';
    }, 2500);
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
      class="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-3xl w-full max-w-xl shadow-2xl overflow-hidden animate-in fade-in zoom-in-95 duration-150 flex flex-col max-h-[85vh]"
      role="dialog"
      aria-modal="true"
      aria-labelledby="settings-title"
    >
      <!-- Header -->
      <div class="px-6 py-5 border-b border-[var(--color-border)] flex items-center justify-between bg-[var(--color-bg)]/50">
        <div class="flex items-center gap-3">
          <div class="w-10 h-10 rounded-2xl bg-[var(--color-accent-subtle)] flex items-center justify-center text-[var(--color-accent)]">
            <Settings size={20} strokeWidth={2.5} />
          </div>
          <div>
            <h3 id="settings-title" class="font-bold text-[var(--color-text)] text-base">Settings</h3>
            <p class="text-xs text-[var(--color-secondary)]">Manage preferences, appearance, and storage</p>
          </div>
        </div>
        <button 
          on:click={onClose}
          class="text-[var(--color-secondary)] hover:text-[var(--color-text)] p-1.5 rounded-xl hover:bg-[var(--color-border)]/50 transition"
          aria-label="Close settings"
        >
          <X size={18} />
        </button>
      </div>

      <!-- Feedback Banner -->
      {#if statusMessage}
        <div class="bg-[var(--color-accent-subtle)] px-6 py-2.5 text-xs text-[var(--color-accent)] font-semibold flex items-center gap-2 border-b border-[var(--color-border)] animate-in fade-in">
          <Check size={14} strokeWidth={3} />
          <span>{statusMessage}</span>
        </div>
      {/if}

      <!-- Body Content -->
      <div class="p-6 overflow-y-auto space-y-6 flex-1 text-sm">
        <!-- Section: Appearance -->
        <div>
          <h4 class="text-xs font-bold text-[var(--color-secondary)] uppercase tracking-wider mb-2.5">
            Appearance
          </h4>
          <div class="grid grid-cols-3 gap-2.5">
            <button 
              on:click={() => setTheme('light')}
              class="flex flex-col items-center gap-2 p-3.5 rounded-2xl border transition-all text-xs font-semibold {
                currentTheme === 'light' 
                  ? 'border-[var(--color-accent)] bg-[var(--color-accent-subtle)] text-[var(--color-accent)] shadow-2xs' 
                  : 'border-[var(--color-border)] bg-[var(--color-bg)] text-[var(--color-secondary)] hover:text-[var(--color-text)]'
              }"
            >
              <Sun size={20} strokeWidth={2.2} />
              <span>Light Mode</span>
            </button>

            <button 
              on:click={() => setTheme('dark')}
              class="flex flex-col items-center gap-2 p-3.5 rounded-2xl border transition-all text-xs font-semibold {
                currentTheme === 'dark' 
                  ? 'border-[var(--color-accent)] bg-[var(--color-accent-subtle)] text-[var(--color-accent)] shadow-2xs' 
                  : 'border-[var(--color-border)] bg-[var(--color-bg)] text-[var(--color-secondary)] hover:text-[var(--color-text)]'
              }"
            >
              <Moon size={20} strokeWidth={2.2} />
              <span>Dark Mode</span>
            </button>

            <button 
              on:click={() => setTheme('system')}
              class="flex flex-col items-center gap-2 p-3.5 rounded-2xl border transition-all text-xs font-semibold {
                currentTheme === 'system' 
                  ? 'border-[var(--color-accent)] bg-[var(--color-accent-subtle)] text-[var(--color-accent)] shadow-2xs' 
                  : 'border-[var(--color-border)] bg-[var(--color-bg)] text-[var(--color-secondary)] hover:text-[var(--color-text)]'
              }"
            >
              <Monitor size={20} strokeWidth={2.2} />
              <span>Auto (System)</span>
            </button>
          </div>
        </div>

        <!-- Section: Downloads & Directory -->
        <div>
          <h4 class="text-xs font-bold text-[var(--color-secondary)] uppercase tracking-wider mb-2.5">
            Downloads Folder
          </h4>
          <div class="bg-[var(--color-bg)] border border-[var(--color-border)] rounded-2xl p-4 space-y-3">
            <div class="flex items-center justify-between gap-3">
              <div class="min-w-0 flex-1">
                <span class="text-xs font-semibold text-[var(--color-text)] block">Default Save Path</span>
                <span class="text-xs font-mono text-[var(--color-secondary)] truncate block mt-0.5" title={downloadDir}>
                  {downloadDir || "Loading..."}
                </span>
              </div>
              <div class="flex items-center gap-2 shrink-0">
                <button 
                  on:click={handleBrowseDir}
                  class="px-3 py-1.5 text-xs font-semibold bg-[var(--color-surface)] hover:bg-[var(--color-bg)] text-[var(--color-text)] border border-[var(--color-border)] rounded-xl shadow-2xs transition"
                >
                  Change
                </button>
                <button 
                  on:click={handleOpenDownloads}
                  class="p-2 text-[var(--color-secondary)] hover:text-[var(--color-accent)] hover:bg-[var(--color-surface)] border border-transparent hover:border-[var(--color-border)] rounded-xl transition"
                  title="Open folder in Finder / Explorer"
                  aria-label="Open folder"
                >
                  <Folder size={16} />
                </button>
              </div>
            </div>

            <!-- Default Thread / Connection Slider -->
            <div class="pt-3 border-t border-[var(--color-border)]">
              <div class="flex items-center justify-between text-xs font-semibold text-[var(--color-text)] mb-1">
                <span class="flex items-center gap-1.5">
                  <Zap size={14} class="text-[var(--color-accent)]" />
                  Default Download Threads
                </span>
                <span class="text-[var(--color-accent)] font-bold">{defaultConnections} chunks</span>
              </div>
              <input 
                type="range" 
                min="1" 
                max="16" 
                step="1" 
                value={defaultConnections} 
                on:input={handleConnectionsChange}
                class="w-full accent-[var(--color-accent)] bg-[var(--color-border)] h-2 rounded-lg cursor-pointer mt-1"
              />
              <div class="flex justify-between text-[10px] text-[var(--color-secondary)] mt-1">
                <span>1 (Single)</span>
                <span>4 (Standard)</span>
                <span>8 (Fast)</span>
                <span>16 (Max)</span>
              </div>
            </div>
          </div>
        </div>

        <!-- Section: Storage & Database -->
        <div>
          <h4 class="text-xs font-bold text-[var(--color-secondary)] uppercase tracking-wider mb-2.5">
            Database & History
          </h4>
          <div class="bg-[var(--color-bg)] border border-[var(--color-border)] rounded-2xl p-4 space-y-3">
            <div class="flex items-center justify-between gap-3">
              <div>
                <span class="text-xs font-semibold text-[var(--color-text)] block">SQLite Database</span>
                <span class="text-[11px] text-[var(--color-secondary)] block font-mono mt-0.5">~/.candak/candak.db</span>
              </div>
              <span class="text-[10px] font-bold px-2 py-0.5 rounded-full bg-[var(--color-accent-subtle)] text-[var(--color-accent)] border border-[var(--color-accent)]/30">
                WAL Mode
              </span>
            </div>

            <div class="pt-3 border-t border-[var(--color-border)] flex items-center justify-between gap-3">
              <div>
                <span class="text-xs font-semibold text-[var(--color-text)] block">URL History Log</span>
                <span class="text-[11px] text-[var(--color-secondary)] block">Tracks previous URLs and download frequencies</span>
              </div>
              <div class="flex items-center gap-2 shrink-0">
                <button 
                  on:click={() => { onClose(); onOpenHistory(); }}
                  class="px-3 py-1.5 text-xs font-semibold bg-[var(--color-surface)] hover:bg-[var(--color-bg)] text-[var(--color-text)] border border-[var(--color-border)] rounded-xl transition flex items-center gap-1.5 shadow-2xs"
                >
                  <History size={13} class="text-[var(--color-accent)]" />
                  View Log
                </button>
                <button 
                  on:click={handleClearHistory}
                  class="p-2 text-rose-500 hover:text-rose-600 hover:bg-rose-500/10 border border-transparent hover:border-rose-500/30 rounded-xl transition"
                  title="Clear URL History"
                  aria-label="Clear URL History"
                >
                  <Trash2 size={15} />
                </button>
              </div>
            </div>
          </div>
        </div>

        <!-- Section: About -->
        <div class="pt-1 text-center text-xs text-[var(--color-secondary)]">
          <p class="font-bold text-[var(--color-text)]">Candak Download Manager v1.0.0</p>
          <p class="mt-0.5 text-[11px]">Go · Wails v2 · Svelte 5 · Tailwind CSS v4 · modernc.org/sqlite</p>
        </div>
      </div>
    </div>
  </div>
{/if}
