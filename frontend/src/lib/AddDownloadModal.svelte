<script>
  import { onMount } from 'svelte';
  import { ProbeURL, ProbeURLWithHeaders, AddDownload, SelectDirectory, GetDefaultDownloadDir, SuppressClipboardURL, ClosePromptWindow } from '@wails/go/main/App.js';
  import { WindowSetSize } from '@wails/runtime/runtime.js';
  import { formatBytes, getFileExtension, DEFAULT_ACCEPTED_EXTENSIONS } from './formatters.js';
  import { 
    X, 
    Folder, 
    Zap, 
    AlertCircle, 
    CheckCircle, 
    Link, 
    ShieldAlert, 
    Plus, 
    Copy, 
    Check, 
    Download, 
    Loader2,
    ChevronDown
  } from 'lucide-svelte';

  export let isOpen = false;
  export let onClose = () => {};
  export let onAdded = () => {};
  export let initialUrl = '';
  export let grabbedData = null;
  export let isStandalone = false;

  let url = '';
  let filename = '';
  let saveDir = '';
  let connections = 4;
  let headers = {};
  let isProbing = false;
  let probeResult = null;
  let probeError = '';
  let isSubmitting = false;
  let extensionNotice = '';
  let rejectedExtension = '';
  let copiedUrl = false;
  let probeTimer = null;
  let lastProbedUrl = '';
  let isExpanded = false;

  function toggleExpand() {
    isExpanded = !isExpanded;
    if (isStandalone) {
      if (isExpanded) {
        WindowSetSize(500, 390);
      } else {
        WindowSetSize(500, 220);
      }
    }
  }

  $: if (isOpen && grabbedData) {
    if (grabbedData.url && grabbedData.url !== url) {
      url = grabbedData.url;
      filename = grabbedData.filename || '';
      headers = grabbedData.headers || {};
      handleProbe();
    }
  } else if (isOpen && initialUrl && initialUrl !== url) {
    url = initialUrl;
    headers = {};
    handleProbe();
  }

  $: if (isOpen) {
    const savedConn = localStorage.getItem('candak_default_connections');
    if (savedConn && (!probeResult || probeResult.resumable)) {
      connections = parseInt(savedConn, 10) || 4;
    }
  }

  $: {
    const candidate = filename || probeResult?.filename;
    if (candidate) {
      checkAcceptedExtension(candidate);
    } else if (url && !isProbing) {
      const ext = getFileExtension(url);
      if (ext) checkAcceptedExtension(url);
      else { extensionNotice = ''; rejectedExtension = ''; }
    } else {
      extensionNotice = '';
      rejectedExtension = '';
    }
  }

  function checkAcceptedExtension(targetName) {
    extensionNotice = '';
    rejectedExtension = '';

    const enforce = localStorage.getItem('candak_enforce_extensions') === 'true';
    if (!enforce) return true;

    const raw = localStorage.getItem('candak_accepted_extensions');
    let accepted = DEFAULT_ACCEPTED_EXTENSIONS;
    if (raw) {
      try { 
        const parsed = JSON.parse(raw);
        if (Array.isArray(parsed) && parsed.length > 0) accepted = parsed;
      } catch (_) {}
    }

    const ext = getFileExtension(targetName);
    if (ext && !accepted.includes(ext.toLowerCase())) {
      rejectedExtension = ext.toLowerCase();
      extensionNotice = `.${rejectedExtension} is not in your accepted extensions list.`;
      return false;
    }
    return true;
  }

  function handleAllowExtension() {
    if (!rejectedExtension) return;
    const raw = localStorage.getItem('candak_accepted_extensions');
    let accepted = [...DEFAULT_ACCEPTED_EXTENSIONS];
    if (raw) {
      try { 
        const parsed = JSON.parse(raw);
        if (Array.isArray(parsed) && parsed.length > 0) accepted = parsed;
      } catch (_) {}
    }
    if (!accepted.includes(rejectedExtension)) {
      accepted.push(rejectedExtension);
      localStorage.setItem('candak_accepted_extensions', JSON.stringify(accepted));
    }
    extensionNotice = '';
    rejectedExtension = '';
  }

  onMount(async () => {
    try {
      const savedDir = localStorage.getItem('candak_custom_download_dir');
      if (savedDir) {
        saveDir = savedDir;
      } else {
        saveDir = await GetDefaultDownloadDir();
      }

      const savedConn = localStorage.getItem('candak_default_connections');
      if (savedConn) {
        connections = parseInt(savedConn, 10) || 4;
      }
    } catch (e) {
      console.error(e);
    }
  });

  function handleUrlInput() {
    clearTimeout(probeTimer);
    const trimmed = url.trim();
    if (!trimmed || !trimmed.startsWith('http')) return;
    if (trimmed === lastProbedUrl) return;

    // Automatic probe debounced after user stops typing
    probeTimer = setTimeout(() => {
      handleProbe();
    }, 400);
  }

  function handleUrlPaste() {
    clearTimeout(probeTimer);
    setTimeout(() => {
      handleProbe();
    }, 50);
  }

  async function handleProbe() {
    const target = url.trim();
    if (!target || !target.startsWith('http')) return;

    lastProbedUrl = target;
    isProbing = true;
    probeError = '';
    probeResult = null;

    try {
      let res;
      if (headers && Object.keys(headers).length > 0) {
        res = await ProbeURLWithHeaders(target, headers);
      } else {
        res = await ProbeURL(target);
      }
      probeResult = res;
      if (res.filename && !filename) {
        filename = res.filename;
      }
      if (!res.resumable) {
        connections = 1;
      }
    } catch (err) {
      probeError = String(err);
      // Fallback: don't strictly block download if probing fails
      if (!filename) {
        try {
          const parsed = new URL(target);
          const pathParts = parsed.pathname.split('/').filter(Boolean);
          if (pathParts.length > 0) {
            filename = decodeURIComponent(pathParts[pathParts.length - 1]);
          }
        } catch (_) {}
      }
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

  async function handleCopyUrl() {
    if (!url) return;
    try {
      await SuppressClipboardURL(url);
    } catch (_) {}
    try {
      await navigator.clipboard.writeText(url);
      copiedUrl = true;
      setTimeout(() => copiedUrl = false, 2000);
    } catch (err) {
      console.error(err);
    }
  }

  async function handleSubmit() {
    if (!url.trim()) return;

    if (!checkAcceptedExtension(filename || probeResult?.filename || url)) {
      return;
    }

    isSubmitting = true;

    try {
      await AddDownload({
        url: url.trim(),
        filename: filename.trim(),
        saveDir: saveDir,
        connections: parseInt(connections, 10) || 4,
        headers: headers || {}
      });
      if (isStandalone) {
        const finalName = filename.trim() || probeResult?.filename || 'download';
        await ClosePromptWindow(true, finalName, probeResult?.totalSize || 0);
      }
      onAdded();
      handleClose();
    } catch (err) {
      probeError = String(err);
    } finally {
      isSubmitting = false;
    }
  }

  function handleClose() {
    if (isStandalone) {
      ClosePromptWindow(false, '', 0);
    }
    url = '';
    filename = '';
    headers = {};
    probeResult = null;
    probeError = '';
    lastProbedUrl = '';
    isExpanded = false;
    isSubmitting = false;
    isProbing = false;
    extensionNotice = '';
    rejectedExtension = '';
    onClose();
  }

  function handleKeydown(event) {
    if (isStandalone) {
      // Hotkeys completely disabled in popup window
      if (event.key === 'Enter' || event.key === 'Escape') {
        event.preventDefault();
        event.stopPropagation();
      }
      return;
    }
    if (event.key === 'Escape') {
      handleClose();
    } else if (event.key === 'Enter' && !isSubmitting && url.trim()) {
      handleSubmit();
    }
  }
</script>

<svelte:window on:keydown|capture={handleKeydown} />

{#if isOpen}
  <!-- Outer Wrapper: Backdrop when in-app modal, or Full Window when standalone -->
  <div 
    class={isStandalone 
      ? "w-full h-full bg-[var(--color-surface)] flex flex-col justify-between overflow-hidden select-none" 
      : "fixed inset-0 bg-black/60 backdrop-blur-xs z-50 flex items-center justify-center p-4 animate-in fade-in duration-150"}
    role={isStandalone ? "dialog" : "presentation"}
    on:click|self={handleClose}
  >
    <div 
      class={isStandalone 
        ? "w-full h-full flex flex-col justify-between overflow-hidden" 
        : "bg-[var(--color-surface)] border border-[var(--color-border)] rounded-3xl w-full max-w-lg shadow-2xl overflow-hidden animate-in zoom-in-95 duration-150"}
      style={isStandalone ? "--wails-draggable: drag;" : ""}
      role="dialog"
      aria-modal="true"
      aria-labelledby="modal-title"
    >
      <!-- Header: Only rendered for in-app modal (header entirely removed in popup window) -->
      {#if !isStandalone}
        <div class="px-6 py-5 border-b border-[var(--color-border)] flex items-center justify-between bg-[var(--color-bg)]/60">
          <div class="flex items-center gap-3">
            <div class="w-10 h-10 rounded-2xl bg-[var(--color-accent-subtle)] flex items-center justify-center text-[var(--color-accent)]">
              <Download size={20} strokeWidth={2.5} />
            </div>
            <div>
              <h3 id="modal-title" class="font-bold text-[var(--color-text)] text-base">
                {grabbedData ? 'Download Prompt' : 'Add New Download'}
              </h3>
              <p class="text-xs text-[var(--color-secondary)]">
                {grabbedData ? 'URL captured • Review details and start download' : 'Multi-thread concurrent chunking with resume support'}
              </p>
            </div>
          </div>
          <button 
            on:click={handleClose}
            class="text-[var(--color-secondary)] hover:text-[var(--color-text)] p-1.5 rounded-xl hover:bg-[var(--color-border)]/50 transition cursor-pointer"
            aria-label="Close dialog"
          >
            <X size={18} />
          </button>
        </div>
      {/if}

      <!-- Form Body -->
      <form on:submit|preventDefault={handleSubmit} class="{isStandalone ? 'p-4 space-y-3' : 'p-6 space-y-4'} flex-1 overflow-y-auto">
        <!-- URL Input with Simplified Inline File Size & Copy Button -->
        <div>
          <div class="flex items-center justify-between mb-1.5">
            <label for="download-url" class="text-xs font-semibold text-[var(--color-text)]">Download URL</label>
            <div class="flex items-center gap-2.5">
              <!-- Simplified Inline File Size -->
              {#if isProbing}
                <span class="flex items-center gap-1 text-[11px] text-[var(--color-accent)] font-medium">
                  <Loader2 size={11} class="animate-spin" />
                  Calculating...
                </span>
              {:else if probeResult && probeResult.totalSize > 0}
                <span class="text-xs font-bold text-[var(--color-accent)] bg-[var(--color-accent-subtle)] px-2 py-0.5 rounded-md">
                  {formatBytes(probeResult.totalSize)}
                </span>
              {:else if probeResult || probeError}
                <span class="text-[11px] font-medium text-[var(--color-secondary)]">Size: Unknown</span>
              {/if}

              {#if url}
                <button
                  type="button"
                  on:click={handleCopyUrl}
                  class="text-[11px] font-medium text-[var(--color-secondary)] hover:text-[var(--color-accent)] flex items-center gap-1 transition cursor-pointer"
                  title="Copy URL"
                >
                  {#if copiedUrl}
                    <Check size={12} class="text-emerald-500" />
                    <span class="text-emerald-500">Copied</span>
                  {:else}
                    <Copy size={12} />
                    <span>Copy</span>
                  {/if}
                </button>
              {/if}
            </div>
          </div>

          <div class="relative flex items-center">
            <input 
              id="download-url"
              type="url" 
              bind:value={url}
              on:input={handleUrlInput}
              on:paste={handleUrlPaste}
              on:blur={handleProbe}
              placeholder="https://example.com/file.zip"
              required
              class="w-full bg-[var(--color-input)] border border-[var(--color-border)] rounded-xl px-3.5 py-2 text-sm text-[var(--color-text)] placeholder-[var(--color-secondary)]/60 focus:outline-none focus:border-[var(--color-accent)] focus:ring-2 focus:ring-[var(--color-accent)]/20 font-mono transition {isProbing ? 'pr-9' : ''}"
            />
            {#if isProbing}
              <div class="absolute right-3 pointer-events-none text-[var(--color-accent)]">
                <Loader2 size={15} class="animate-spin" />
              </div>
            {/if}
          </div>
        </div>

        <!-- Extension Warning Banner -->
        {#if extensionNotice}
          <div class="bg-amber-500/10 border border-amber-500/30 rounded-2xl p-2.5 text-xs text-amber-600 dark:text-amber-400 flex items-center justify-between gap-2 animate-in fade-in">
            <div class="flex items-center gap-2 min-w-0">
              <ShieldAlert size={15} class="shrink-0 text-amber-500" />
              <span class="truncate">{extensionNotice}</span>
            </div>
            <button 
              type="button"
              on:click={handleAllowExtension}
              class="px-2 py-0.5 bg-amber-500/20 hover:bg-amber-500/30 text-amber-700 dark:text-amber-300 font-bold rounded-lg transition text-[11px] shrink-0 flex items-center gap-1 cursor-pointer"
            >
              <Plus size={11} strokeWidth={3} />
              <span>Allow .{rejectedExtension}</span>
            </button>
          </div>
        {/if}

        <!-- Expandable Options (Filename, Save Path & Connections) -->
        <div>
          <button 
            type="button"
            on:click={toggleExpand}
            class="flex items-center gap-1 text-xs text-[var(--color-secondary)] hover:text-[var(--color-accent)] font-medium transition cursor-pointer select-none py-0.5"
          >
            <ChevronDown size={14} class="transition-transform duration-200 {isExpanded ? 'rotate-180' : ''}" />
            <span>{isExpanded ? 'Hide options' : 'More options (filename, save path)'}</span>
            {#if !isExpanded && (filename || probeResult?.filename)}
              <span class="text-[11px] text-[var(--color-secondary)]/70 font-mono truncate max-w-[200px] ml-1">
                ({filename || probeResult?.filename})
              </span>
            {/if}
          </button>

          {#if isExpanded}
            <div class="mt-2.5 space-y-3 pt-2.5 border-t border-[var(--color-border)]/50 animate-in fade-in duration-150">
              <!-- Target Filename -->
              <div>
                <label for="download-filename" class="block text-xs font-semibold text-[var(--color-text)] mb-1">Save As Filename</label>
                <input 
                  id="download-filename"
                  type="text" 
                  bind:value={filename}
                  placeholder={probeResult?.filename || "auto-detected-filename.bin"}
                  class="w-full bg-[var(--color-input)] border border-[var(--color-border)] rounded-xl px-3 py-2 text-xs text-[var(--color-text)] placeholder-[var(--color-secondary)]/60 focus:outline-none focus:border-[var(--color-accent)] focus:ring-2 focus:ring-[var(--color-accent)]/20 transition"
                />
              </div>

              <!-- Save Directory -->
              <div>
                <label for="download-dir" class="block text-xs font-semibold text-[var(--color-text)] mb-1">Save Path</label>
                <div class="flex gap-2">
                  <input 
                    id="download-dir"
                    type="text" 
                    bind:value={saveDir}
                    placeholder="/Downloads"
                    class="w-full bg-[var(--color-input)] border border-[var(--color-border)] rounded-xl px-3 py-2 text-xs text-[var(--color-text)] font-mono focus:outline-none focus:border-[var(--color-accent)] focus:ring-2 focus:ring-[var(--color-accent)]/20 transition"
                  />
                  <button 
                    type="button"
                    on:click={handleBrowse}
                    class="px-3 py-2 bg-[var(--color-surface)] hover:bg-[var(--color-bg)] text-[var(--color-text)] text-xs font-medium rounded-xl border border-[var(--color-border)] shadow-2xs transition flex items-center gap-1.5 shrink-0 cursor-pointer"
                  >
                    <Folder size={14} class="text-[var(--color-secondary)]" />
                    Browse
                  </button>
                </div>
              </div>

              <!-- Concurrent Chunks / Connections -->
              <div>
                <div class="flex items-center justify-between mb-1">
                  <label for="download-connections" class="text-xs font-semibold text-[var(--color-text)] flex items-center gap-1">
                    <Zap size={13} class="text-[var(--color-accent)]" />
                    Parallel Connections
                  </label>
                  <span class="text-xs font-bold text-[var(--color-accent)]">{connections} threads</span>
                </div>
                <input 
                  id="download-connections"
                  type="range" 
                  min="1" 
                  max="16" 
                  step="1" 
                  bind:value={connections}
                  disabled={probeResult && !probeResult.resumable}
                  class="w-full accent-[var(--color-accent)] bg-[var(--color-border)] h-1.5 rounded-lg cursor-pointer disabled:opacity-40"
                />
              </div>
            </div>
          {/if}
        </div>

        <!-- Error Banner (Error retention) -->
        {#if probeError}
          <div class="bg-rose-500/10 border border-rose-500/30 rounded-2xl p-3 text-xs text-rose-500 flex items-start gap-2 animate-in fade-in">
            <AlertCircle size={16} class="shrink-0 mt-0.5" />
            <span class="break-all">{probeError}</span>
          </div>
        {/if}

        <!-- Action Buttons: Cancel and Download -->
        <div class="pt-3 border-t border-[var(--color-border)] flex items-center justify-between" style="--wails-draggable: no-drag;">
          {#if !isStandalone}
            <span class="text-[11px] text-[var(--color-secondary)]">
              Press <kbd class="px-1.5 py-0.5 rounded bg-[var(--color-bg)] border border-[var(--color-border)] font-mono text-[10px]">Esc</kbd> to cancel, <kbd class="px-1.5 py-0.5 rounded bg-[var(--color-bg)] border border-[var(--color-border)] font-mono text-[10px]">Enter</kbd> to download
            </span>
          {:else}
            <div></div>
          {/if}
          <div class="flex items-center gap-2.5">
            <button 
              type="button" 
              on:click={handleClose}
              class="px-4 py-2 text-xs font-semibold text-[var(--color-secondary)] hover:text-[var(--color-text)] hover:bg-[var(--color-bg)] rounded-xl transition cursor-pointer"
            >
              Cancel
            </button>
            <button 
              type="submit" 
              disabled={isSubmitting || !url.trim()}
              class="px-5 py-2 text-xs font-bold text-white bg-[var(--color-accent)] hover:bg-[var(--color-accent-hover)] rounded-xl shadow-sm hover:shadow transition flex items-center gap-2 disabled:opacity-50 cursor-pointer"
            >
              {#if isSubmitting}
                <Loader2 size={14} class="animate-spin" />
                Starting...
              {:else}
                <Download size={14} strokeWidth={2.5} />
                Download
              {/if}
            </button>
          </div>
        </div>
      </form>
    </div>
  </div>
{/if}
