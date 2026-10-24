<script>
  import { onMount } from 'svelte';
  import { theme } from './theme.js';
  import { 
    GetDefaultDownloadDir, 
    SelectDirectory, 
    OpenFolder,
    GetLoopbackStatus,
    GetExtensionDir,
    SetClipboardWatching,
    IsClipboardWatching
  } from '@wails/go/main/App.js';
  import { 
    X, 
    Settings, 
    Sun, 
    Moon, 
    Monitor, 
    Folder, 
    Check, 
    Zap,
    Plus,
    Edit2,
    RotateCcw,
    FileCheck,
    Globe,
    Clipboard,
    ExternalLink,
    ShieldCheck
  } from 'lucide-svelte';
  import { DEFAULT_ACCEPTED_EXTENSIONS } from './formatters.js';

  export let isOpen = false;
  export let onClose = () => {};

  let currentTheme = 'system';
  let downloadDir = '';
  let defaultConnections = 4;
  let statusMessage = '';

  // Accepted Extensions State
  let acceptedExtensions = [...DEFAULT_ACCEPTED_EXTENSIONS];
  let enforceExtensions = false;
  let newExtInput = '';
  let editingIndex = null;
  let editingValue = '';
  let extError = '';

  // Browser & Clipboard Integration State
  let loopbackStatus = { running: false, address: '', port: 41920 };
  let extensionDir = '';
  let clipboardWatching = false;

  theme.subscribe((val) => {
    currentTheme = val;
  });

  onMount(async () => {
    try {
      const savedDir = localStorage.getItem('candak_custom_download_dir');
      if (savedDir) {
        downloadDir = savedDir;
      } else {
        downloadDir = await GetDefaultDownloadDir();
      }
      const savedConn = localStorage.getItem('candak_default_connections');
      if (savedConn) defaultConnections = parseInt(savedConn, 10);

      const savedExts = localStorage.getItem('candak_accepted_extensions');
      if (savedExts) {
        try {
          const parsed = JSON.parse(savedExts);
          if (Array.isArray(parsed) && parsed.length > 0) {
            acceptedExtensions = parsed;
          }
        } catch (_) {}
      }

      enforceExtensions = localStorage.getItem('candak_enforce_extensions') === 'true';

      // Load loopback and clipboard status
      try {
        loopbackStatus = await GetLoopbackStatus();
        extensionDir = await GetExtensionDir();
        clipboardWatching = await IsClipboardWatching();
      } catch (e) {
        console.error('Failed to load integration status:', e);
      }
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
    if (e && e.target) {
      defaultConnections = parseInt(e.target.value, 10);
    }
    localStorage.setItem('candak_default_connections', String(defaultConnections));
  }

  function setConnections(val) {
    defaultConnections = val;
    localStorage.setItem('candak_default_connections', String(defaultConnections));
  }

  function saveExtensions() {
    localStorage.setItem('candak_accepted_extensions', JSON.stringify(acceptedExtensions));
    localStorage.setItem('candak_enforce_extensions', String(enforceExtensions));
  }

  function handleToggleEnforce() {
    enforceExtensions = !enforceExtensions;
    saveExtensions();
    flashMessage(enforceExtensions ? 'Extension filtering enabled' : 'Extension filtering disabled');
  }

  function handleAddExtension() {
    extError = '';
    const clean = newExtInput.trim().replace(/^\.+/, '').toLowerCase();
    if (!clean) return;
    if (!/^[a-z0-9_-]+(\.[a-z0-9_-]+)*$/.test(clean)) {
      extError = 'Extension must contain alphanumeric characters (e.g. zip, tar.gz)';
      return;
    }
    if (acceptedExtensions.includes(clean)) {
      extError = `.${clean} is already in the list`;
      return;
    }
    acceptedExtensions = [...acceptedExtensions, clean];
    newExtInput = '';
    saveExtensions();
    flashMessage(`Added .${clean}`);
  }

  function handleStartEdit(idx) {
    editingIndex = idx;
    editingValue = acceptedExtensions[idx];
    extError = '';
  }

  function handleSaveEdit(idx) {
    extError = '';
    const clean = editingValue.trim().replace(/^\.+/, '').toLowerCase();
    if (!clean) return;
    if (!/^[a-z0-9_-]+(\.[a-z0-9_-]+)*$/.test(clean)) {
      extError = 'Extension must contain alphanumeric characters (e.g. zip, tar.gz)';
      return;
    }
    if (acceptedExtensions.some((ext, i) => i !== idx && ext === clean)) {
      extError = `.${clean} already exists in the list`;
      return;
    }
    acceptedExtensions[idx] = clean;
    acceptedExtensions = [...acceptedExtensions];
    editingIndex = null;
    saveExtensions();
    flashMessage(`Updated to .${clean}`);
  }

  function handleCancelEdit() {
    editingIndex = null;
    editingValue = '';
    extError = '';
  }

  function handleRemoveExtension(idx) {
    const removed = acceptedExtensions[idx];
    acceptedExtensions = acceptedExtensions.filter((_, i) => i !== idx);
    if (editingIndex === idx) editingIndex = null;
    saveExtensions();
    flashMessage(`Removed .${removed}`);
  }

  function handleResetExtensions() {
    acceptedExtensions = [...DEFAULT_ACCEPTED_EXTENSIONS];
    saveExtensions();
    flashMessage('Reset extensions to standard defaults');
  }

  function flashMessage(msg) {
    statusMessage = msg;
    setTimeout(() => {
      statusMessage = '';
    }, 2500);
  }

  async function handleToggleClipboardWatcher() {
    try {
      const nextState = !clipboardWatching;
      const res = await SetClipboardWatching(nextState);
      clipboardWatching = res;
      localStorage.setItem('candak_clipboard_watching', String(res));
      flashMessage(res ? 'Clipboard URL grabber enabled' : 'Clipboard URL grabber disabled');
    } catch (err) {
      console.error(err);
    }
  }

  async function handleOpenExtensionFolder() {
    try {
      if (!extensionDir) {
        extensionDir = await GetExtensionDir();
      }
      await OpenFolder(extensionDir);
      flashMessage('Opened browser extension folder');
    } catch (err) {
      console.error(err);
    }
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
                bind:value={defaultConnections} 
                on:input={handleConnectionsChange}
                class="w-full accent-[var(--color-accent)] bg-[var(--color-border)] h-2 rounded-lg cursor-pointer mt-1"
              />
              <div class="relative text-[10px] text-[var(--color-secondary)] mt-1.5 h-3.5 select-none">
                <button type="button" on:click={() => setConnections(1)} class="absolute left-0 hover:text-[var(--color-text)] transition cursor-pointer">1 (Single)</button>
                <button type="button" on:click={() => setConnections(4)} class="absolute left-[calc(8px+0.20*(100%-16px))] -translate-x-1/2 hover:text-[var(--color-text)] transition cursor-pointer">4 (Standard)</button>
                <button type="button" on:click={() => setConnections(8)} class="absolute left-[calc(8px+0.467*(100%-16px))] -translate-x-1/2 hover:text-[var(--color-text)] transition cursor-pointer">8 (Fast)</button>
                <button type="button" on:click={() => setConnections(16)} class="absolute right-0 hover:text-[var(--color-text)] transition cursor-pointer">16 (Max)</button>
              </div>
            </div>
          </div>
        </div>

        <!-- Section: Accepted File Extensions -->
        <div>
          <div class="flex items-center justify-between mb-2.5">
            <h4 class="text-xs font-bold text-[var(--color-secondary)] uppercase tracking-wider">
              Accepted Extensions
            </h4>
            <!-- Toggle Enforce -->
            <button 
              type="button"
              on:click={handleToggleEnforce}
              class="flex items-center gap-2 text-xs font-semibold px-2.5 py-1 rounded-full border transition {
                enforceExtensions 
                  ? 'bg-[var(--color-accent-subtle)] text-[var(--color-accent)] border-[var(--color-accent)]/30' 
                  : 'bg-[var(--color-bg)] text-[var(--color-secondary)] border-[var(--color-border)]'
              }"
              title="Toggle whether only accepted file types can be downloaded"
            >
              <span class="w-2 h-2 rounded-full {enforceExtensions ? 'bg-[var(--color-accent)]' : 'bg-gray-400'}"></span>
              <span>{enforceExtensions ? 'Filter Active' : 'Allow All'}</span>
            </button>
          </div>

          <div class="bg-[var(--color-bg)] border border-[var(--color-border)] rounded-2xl p-4 space-y-3.5">
            <!-- Add Input Row -->
            <form on:submit|preventDefault={handleAddExtension} class="flex items-center gap-2">
              <div class="relative flex-1">
                <input 
                  type="text" 
                  bind:value={newExtInput}
                  placeholder="Add extension (e.g. epub, iso, dmg)..."
                  class="w-full bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl px-3.5 py-2 text-xs text-[var(--color-text)] placeholder-[var(--color-secondary)]/60 focus:outline-none focus:border-[var(--color-accent)] focus:ring-2 focus:ring-[var(--color-accent)]/20 font-mono transition"
                />
              </div>
              <button 
                type="submit" 
                disabled={!newExtInput.trim()}
                class="px-4 py-2 bg-[var(--color-accent)] hover:bg-[var(--color-accent-hover)] text-white text-xs font-bold rounded-xl transition flex items-center gap-1.5 shadow-2xs disabled:opacity-40 shrink-0"
              >
                <Plus size={14} strokeWidth={2.5} />
                <span>Add</span>
              </button>
            </form>

            {#if extError}
              <div class="text-[11px] text-rose-500 font-medium px-1">
                {extError}
              </div>
            {/if}

            <!-- Tags Container -->
            <div class="flex flex-wrap gap-1.5 max-h-44 overflow-y-auto pr-1">
              {#each acceptedExtensions as ext, i}
                {#if editingIndex === i}
                  <!-- Inline Edit Tag -->
                  <div class="flex items-center gap-1 bg-[var(--color-surface)] border-2 border-[var(--color-accent)] rounded-lg px-2 py-1 shadow-xs">
                    <span class="text-[11px] text-[var(--color-secondary)] font-mono">.</span>
                    <input 
                      type="text" 
                      bind:value={editingValue}
                      on:keydown={(e) => { if (e.key === 'Enter') handleSaveEdit(i); if (e.key === 'Escape') handleCancelEdit(); }}
                      class="w-16 bg-transparent text-xs font-mono font-bold text-[var(--color-text)] focus:outline-none"
                    />
                    <button 
                      type="button" 
                      on:click={() => handleSaveEdit(i)} 
                      class="text-[var(--color-accent)] hover:opacity-80 p-0.5"
                      title="Save"
                    >
                      <Check size={13} strokeWidth={3} />
                    </button>
                    <button 
                      type="button" 
                      on:click={handleCancelEdit} 
                      class="text-[var(--color-secondary)] hover:text-rose-500 p-0.5"
                      title="Cancel"
                    >
                      <X size={13} strokeWidth={2.5} />
                    </button>
                  </div>
                {:else}
                  <!-- Display Tag -->
                  <div class="group flex items-center gap-1 bg-[var(--color-surface)] border border-[var(--color-border)] rounded-lg pl-2.5 pr-1.5 py-1 text-xs font-mono transition-colors hover:border-[var(--color-accent)]/50">
                    <button 
                      type="button"
                      on:click={() => handleStartEdit(i)}
                      class="text-[var(--color-text)] hover:text-[var(--color-accent)] font-semibold flex items-center gap-0.5"
                      title="Click to edit extension"
                    >
                      <span>.{ext}</span>
                      <Edit2 size={10} class="opacity-0 group-hover:opacity-60 text-[var(--color-secondary)] transition-opacity ml-1" />
                    </button>

                    <button 
                      type="button" 
                      on:click={() => handleRemoveExtension(i)}
                      class="text-[var(--color-secondary)] hover:text-rose-500 p-0.5 rounded transition"
                      title="Remove extension"
                    >
                      <X size={12} strokeWidth={2.2} />
                    </button>
                  </div>
                {/if}
              {/each}
            </div>

            <!-- Footer Stats & Reset -->
            <div class="pt-2 border-t border-[var(--color-border)] flex items-center justify-between text-[11px] text-[var(--color-secondary)]">
              <span>{acceptedExtensions.length} accepted file types</span>
              <button 
                type="button"
                on:click={handleResetExtensions}
                class="hover:text-[var(--color-accent)] flex items-center gap-1 transition"
                title="Reset to default common file extensions"
              >
                <RotateCcw size={12} />
                <span>Reset Defaults</span>
              </button>
            </div>
          </div>
        </div>

        <!-- Section: Browser & Clipboard Integration -->
        <div>
          <h4 class="text-xs font-bold text-[var(--color-secondary)] uppercase tracking-wider mb-2.5">
            Browser & URL Grabber Integration
          </h4>
          <div class="bg-[var(--color-bg)] border border-[var(--color-border)] rounded-2xl p-4 space-y-4">
            <!-- Loopback Server Status -->
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2.5">
                <div class="w-8 h-8 rounded-xl bg-emerald-500/10 text-emerald-500 flex items-center justify-center">
                  <Globe size={18} />
                </div>
                <div>
                  <span class="text-xs font-semibold text-[var(--color-text)] block">Loopback Interception Server</span>
                  <span class="text-[11px] text-[var(--color-secondary)]">Listening on 127.0.0.1:{loopbackStatus.port || 41920}</span>
                </div>
              </div>
              <span class="px-2.5 py-1 rounded-full text-[11px] font-bold bg-emerald-500/15 text-emerald-500 flex items-center gap-1.5">
                <span class="w-2 h-2 rounded-full bg-emerald-500 animate-pulse"></span>
                Active
              </span>
            </div>

            <!-- Clipboard Watcher Toggle -->
            <div class="pt-3 border-t border-[var(--color-border)] flex items-center justify-between">
              <div class="flex items-center gap-2.5">
                <div class="w-8 h-8 rounded-xl bg-[var(--color-accent-subtle)] text-[var(--color-accent)] flex items-center justify-center">
                  <Clipboard size={17} />
                </div>
                <div>
                  <span class="text-xs font-semibold text-[var(--color-text)] block">Clipboard URL Grabber</span>
                  <span class="text-[11px] text-[var(--color-secondary)]">Auto-prompt download when a downloadable link is copied</span>
                </div>
              </div>
              <button 
                type="button"
                on:click={handleToggleClipboardWatcher}
                class="flex items-center gap-2 text-xs font-semibold px-2.5 py-1 rounded-full border transition cursor-pointer {
                  clipboardWatching 
                    ? 'bg-[var(--color-accent-subtle)] text-[var(--color-accent)] border-[var(--color-accent)]/30' 
                    : 'bg-[var(--color-surface)] text-[var(--color-secondary)] border-[var(--color-border)]'
                }"
              >
                <span class="w-2 h-2 rounded-full {clipboardWatching ? 'bg-[var(--color-accent)]' : 'bg-gray-400'}"></span>
                <span>{clipboardWatching ? 'Enabled' : 'Disabled'}</span>
              </button>
            </div>

            <!-- Companion Browser Extension Setup -->
            <div class="pt-3 border-t border-[var(--color-border)] space-y-2.5">
              <div class="flex items-center justify-between">
                <div>
                  <span class="text-xs font-semibold text-[var(--color-text)] block">Companion Browser Extension</span>
                  <span class="text-[11px] text-[var(--color-secondary)]">Directly intercepts browser download clicks</span>
                </div>
                <button 
                  type="button"
                  on:click={handleOpenExtensionFolder}
                  class="px-3 py-1.5 text-xs font-semibold bg-[var(--color-surface)] hover:bg-[var(--color-bg)] text-[var(--color-accent)] border border-[var(--color-border)] rounded-xl shadow-2xs transition flex items-center gap-1.5 cursor-pointer"
                >
                  <Folder size={14} />
                  <span>Open Extension Folder</span>
                </button>
              </div>

              <!-- Quick Setup Steps -->
              <div class="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl p-3 text-[11px] space-y-1.5 text-[var(--color-secondary)]">
                <div class="font-bold text-[var(--color-text)] flex items-center gap-1.5">
                  <ShieldCheck size={14} class="text-[var(--color-accent)]" />
                  Quick Installation (Chrome, Edge, Brave, Opera):
                </div>
                <ol class="list-decimal list-inside space-y-1 pl-1">
                  <li>Open <code class="px-1 py-0.5 rounded bg-[var(--color-bg)] font-mono text-[10px]">chrome://extensions</code> in your browser.</li>
                  <li>Enable <strong class="text-[var(--color-text)]">Developer mode</strong> (top-right toggle).</li>
                  <li>Click <strong class="text-[var(--color-text)]">Load unpacked</strong> and select the Candak extension folder.</li>
                </ol>
              </div>
            </div>
          </div>
        </div>

        <!-- Section: About -->
        <div class="pt-1 text-center text-xs text-[var(--color-secondary)]">
          <p class="font-bold text-[var(--color-text)]">Candak Download Manager v0.1.1</p>
          <p class="mt-0.5 text-[11px]">Fast, multi-threaded desktop download manager</p>
        </div>
      </div>
    </div>
  </div>
{/if}
