<script>
  import { onMount } from 'svelte';
  import { 
    ProbeURL, 
    ProbeURLWithHeaders, 
    AddDownload, 
    AddBatchDownloads, 
    BatchProbeURLs, 
    SelectDirectory, 
    GetDefaultDownloadDir, 
    SuppressClipboardURL, 
    ClosePromptWindow,
    IsYouTubeURL,
    GetYouTubeMetadata,
    DownloadYouTubeVideo
  } from '@wails/go/main/App.js';
  import { WindowSetSize } from '@wails/runtime/runtime.js';
  import { formatBytes, getFileExtension, DEFAULT_ACCEPTED_EXTENSIONS } from './formatters.js';
  import { STORAGE_KEYS } from './constants.js';
  import { t } from './i18n/index.js';
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
    ChevronDown,
    Video,
    Music,
    Film
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

  // YouTube Dedicated Mode State
  const YT_REGEX = /^(https?:\/\/)?(www\.|m\.)?(youtube\.com\/(watch\?.*v=|shorts\/|live\/)|youtu\.be\/)[a-zA-Z0-9_-]+/i;
  const HLS_REGEX = /\.m3u8($|\?)/i;
  let isYouTube = false;
  let isAnalyzingYT = false;
  let ytMeta = null;
  let ytError = '';
  let selectedYTFormat = null;

  function isYouTubeLink(raw) {
    return raw ? YT_REGEX.test(raw.trim()) : false;
  }

  function isHlsLink(raw) {
    return raw ? HLS_REGEX.test(raw.trim()) : false;
  }

  async function fetchYouTubeDetails(targetUrl) {
    const trimmed = (targetUrl || '').trim();
    if (!trimmed || !isYouTubeLink(trimmed)) return;
    isYouTube = true;
    isAnalyzingYT = true;
    ytError = '';
    ytMeta = null;
    selectedYTFormat = null;
    if (isStandalone) {
      WindowSetSize(560, 520);
    }
    try {
      const meta = await GetYouTubeMetadata(trimmed);
      ytMeta = meta;
      if (meta && meta.title) {
        filename = meta.title.replace(/[\\/:*?"<>|]/g, '').trim() + '.mp4';
      }
      if (meta && meta.options && meta.options.length > 0) {
        selectedYTFormat = meta.options[0];
      }
    } catch (err) {
      ytError = String(err);
    } finally {
      isAnalyzingYT = false;
    }
  }

  function selectYTFormat(opt) {
    selectedYTFormat = opt;
    if (ytMeta && ytMeta.title) {
      const base = ytMeta.title.replace(/[\\/:*?"<>|]/g, '').trim();
      filename = `${base}.${opt.ext || 'mp4'}`;
    }
  }

  async function handleYouTubeSubmit() {
    if (!selectedYTFormat || isSubmitting) return;
    isSubmitting = true;
    try {
      const finalName = filename || (ytMeta ? ytMeta.title + '.mp4' : 'youtube_download.mp4');
      await DownloadYouTubeVideo(url.trim(), selectedYTFormat.format_id, saveDir, finalName, selectedYTFormat.filesize || 0);
      if (isStandalone) {
        await ClosePromptWindow(true, finalName, selectedYTFormat.filesize || 0);
      }
      onAdded();
      handleClose();
    } catch (err) {
      ytError = String(err);
    } finally {
      isSubmitting = false;
    }
  }

  // Mode Selection: 'single' | 'batch'
  let activeMode = 'single';
  $: isBatchMode = activeMode === 'batch';

  // Batch Mode State
  let batchText = '';
  let batchItems = [];
  let isBatchProbing = false;
  let batchProbeTimer = null;

  $: hasErrors = batchItems.some(it => it.status === 'error');
  $: selectedCount = batchItems.filter(it => it.selected).length;

  function setMode(mode) {
    activeMode = mode;
    if (mode === 'batch') {
      if (!batchText.trim() && url.trim()) {
        batchText = url.trim();
        syncBatchFromUrls(extractDistinctUrls(batchText));
      }
    } else if (mode === 'single') {
      if (!url.trim() && batchItems.length > 0) {
        url = batchItems[0].url;
        filename = batchItems[0].filename;
        if (batchItems[0].status === 'valid') {
          probeResult = {
            filename: batchItems[0].filename,
            totalSize: batchItems[0].totalSize,
            resumable: batchItems[0].resumable
          };
        }
      }
    }
    if (isStandalone) {
      if (mode === 'batch') {
        WindowSetSize(620, 540);
      } else {
        WindowSetSize(500, isExpanded ? 390 : 220);
      }
    }
  }

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
    if (grabbedData.url && grabbedData.url !== url && grabbedData.url !== batchText) {
      if (!checkAndTransitionBatch(grabbedData.url)) {
        url = grabbedData.url;
        filename = grabbedData.filename || '';
        headers = grabbedData.headers || {};
        if (isYouTubeLink(url)) {
          fetchYouTubeDetails(url);
        } else {
          isYouTube = false;
          handleProbe();
        }
      }
    }
  } else if (isOpen && initialUrl && initialUrl !== url && initialUrl !== batchText) {
    if (!checkAndTransitionBatch(initialUrl)) {
      url = initialUrl;
      headers = {};
      if (isYouTubeLink(url)) {
        fetchYouTubeDetails(url);
      } else {
        isYouTube = false;
        handleProbe();
      }
    }
  }

  $: if (isOpen) {
    const savedConn = localStorage.getItem(STORAGE_KEYS.DEFAULT_CONNECTIONS);
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

    const enforce = localStorage.getItem(STORAGE_KEYS.ENFORCE_EXTENSIONS) === 'true';
    if (!enforce) return true;

    const raw = localStorage.getItem(STORAGE_KEYS.ACCEPTED_EXTENSIONS);
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
    const raw = localStorage.getItem(STORAGE_KEYS.ACCEPTED_EXTENSIONS);
    let accepted = [...DEFAULT_ACCEPTED_EXTENSIONS];
    if (raw) {
      try { 
        const parsed = JSON.parse(raw);
        if (Array.isArray(parsed) && parsed.length > 0) accepted = parsed;
      } catch (_) {}
    }
    if (!accepted.includes(rejectedExtension)) {
      accepted.push(rejectedExtension);
      localStorage.setItem(STORAGE_KEYS.ACCEPTED_EXTENSIONS, JSON.stringify(accepted));
    }
    extensionNotice = '';
    rejectedExtension = '';
  }

  onMount(async () => {
    try {
      const savedDir = localStorage.getItem(STORAGE_KEYS.CUSTOM_DOWNLOAD_DIR);
      if (savedDir) {
        saveDir = savedDir;
      } else {
        saveDir = await GetDefaultDownloadDir();
      }

      const savedConn = localStorage.getItem(STORAGE_KEYS.DEFAULT_CONNECTIONS);
      if (savedConn) {
        connections = parseInt(savedConn, 10) || 4;
      }
    } catch (e) {
      console.error(e);
    }
  });

  function extractDistinctUrls(text) {
    if (!text) return [];
    // Extract any http/https URLs from free text, ignoring brackets, quotes, and whitespace
    const matches = text.match(/https?:\/\/[^\s"'<>\)\]\}]+/gi) || [];
    const cleaned = matches.map(u => {
      let trimmed = u.trim();
      // Remove trailing punctuation common in free-form text or sentences
      trimmed = trimmed.replace(/[.,;:]+$/, '');
      return trimmed;
    }).filter(u => u.startsWith('http://') || u.startsWith('https://'));
    return Array.from(new Set(cleaned));
  }

  function getDomain(rawUrl) {
    try {
      const u = new URL(rawUrl);
      return u.hostname;
    } catch (_) {
      return 'link';
    }
  }

  function getFallbackFilename(rawUrl) {
    try {
      const u = new URL(rawUrl);
      const parts = u.pathname.split('/').filter(Boolean);
      if (parts.length > 0) {
        return decodeURIComponent(parts[parts.length - 1]);
      }
    } catch (_) {}
    return 'download.bin';
  }

  function checkAndTransitionBatch(text) {
    if (!text) return false;
    const urls = extractDistinctUrls(text);
    const hasNewline = text.includes('\n');
    if (urls.length > 1 || (hasNewline && urls.length >= 1 && text.trim().split('\n').filter(l => l.trim()).length > 1)) {
      setMode('batch');
      batchText = text;
      syncBatchFromUrls(urls);
      return true;
    }
    return false;
  }

  function isExtensionAccepted(targetName) {
    const enforce = localStorage.getItem(STORAGE_KEYS.ENFORCE_EXTENSIONS) === 'true';
    if (!enforce) return true;
    const raw = localStorage.getItem(STORAGE_KEYS.ACCEPTED_EXTENSIONS);
    let accepted = DEFAULT_ACCEPTED_EXTENSIONS;
    if (raw) {
      try {
        const parsed = JSON.parse(raw);
        if (Array.isArray(parsed) && parsed.length > 0) accepted = parsed;
      } catch (_) {}
    }
    const ext = getFileExtension(targetName);
    return !ext || accepted.includes(ext.toLowerCase());
  }

  function syncBatchFromUrls(urls) {
    const existingMap = new Map();
    for (const item of batchItems) {
      existingMap.set(item.url, item);
    }

    const unprobed = [];
    const newItems = urls.map((u, idx) => {
      if (existingMap.has(u)) {
        return existingMap.get(u);
      }
      unprobed.push(u);
      const fallback = getFallbackFilename(u);
      const extAllowed = isExtensionAccepted(fallback);
      return {
        id: `batch_${idx}_${Date.now()}`,
        url: u,
        domain: getDomain(u),
        selected: extAllowed,
        blockedExt: !extAllowed,
        status: 'probing',
        filename: fallback,
        totalSize: 0,
        resumable: false,
        error: ''
      };
    });

    batchItems = newItems;
    if (unprobed.length > 0) {
      runBatchProbe(unprobed);
    }
  }

  function handleBatchInput() {
    clearTimeout(batchProbeTimer);
    batchProbeTimer = setTimeout(() => {
      const urls = extractDistinctUrls(batchText);
      syncBatchFromUrls(urls);
    }, 400);
  }

  async function runBatchProbe(urls) {
    if (!urls || urls.length === 0) return;
    isBatchProbing = true;
    try {
      const results = await BatchProbeURLs(urls);
      results.forEach((res, i) => {
        const targetUrl = urls[i];
        const item = batchItems.find(it => it.url === targetUrl);
        if (item) {
          if (res.error) {
            item.status = 'error';
            item.error = res.error;
            item.filename = res.filename || item.filename;
          } else {
            item.status = 'valid';
            item.error = '';
            item.filename = res.filename || item.filename;
            item.totalSize = res.totalSize || 0;
            item.resumable = res.resumable || false;
            const extAllowed = isExtensionAccepted(item.filename);
            item.blockedExt = !extAllowed;
            if (!extAllowed) item.selected = false;
          }
        }
      });
      batchItems = [...batchItems];
    } catch (err) {
      console.error('Batch probe failed:', err);
      batchItems.forEach(it => {
        if (urls.includes(it.url) && it.status === 'probing') {
          it.status = 'error';
          it.error = String(err);
        }
      });
      batchItems = [...batchItems];
    } finally {
      isBatchProbing = false;
    }
  }

  function handleSelectAll() {
    batchItems = batchItems.map(it => ({ ...it, selected: true }));
  }

  function handleDeselectBroken() {
    batchItems = batchItems.map(it => {
      if (it.status === 'error') {
        return { ...it, selected: false };
      }
      return it;
    });
  }

  function switchToSingle() {
    setMode('single');
  }

  function handleUrlInput() {
    clearTimeout(probeTimer);
    if (checkAndTransitionBatch(url)) return;
    const trimmed = url.trim();
    if (!trimmed || !trimmed.startsWith('http')) return;
    if (trimmed === lastProbedUrl) return;

    if (isYouTubeLink(trimmed)) {
      lastProbedUrl = trimmed;
      fetchYouTubeDetails(trimmed);
      return;
    }

    isYouTube = false;
    // Automatic probe debounced after user stops typing
    probeTimer = setTimeout(() => {
      handleProbe();
    }, 400);
  }

  function handleUrlPaste(event) {
    clearTimeout(probeTimer);
    const clipboardText = event.clipboardData ? event.clipboardData.getData('text') : '';
    if (clipboardText && checkAndTransitionBatch(clipboardText)) {
      event.preventDefault();
      return;
    }
    setTimeout(() => {
      if (checkAndTransitionBatch(url)) return;
      if (isYouTubeLink(url)) {
        lastProbedUrl = url.trim();
        fetchYouTubeDetails(url);
      } else {
        isYouTube = false;
        handleProbe();
      }
    }, 50);
  }

  async function handleProbe() {
    const target = url.trim();
    if (!target || !target.startsWith('http')) return;

    lastProbedUrl = target;
    isProbing = true;
    probeError = '';
    probeResult = null;

    if (isHlsLink(target)) {
      isProbing = false;
      if (!filename || filename.endsWith('.m3u8')) {
        try {
          const parsed = new URL(target);
          const base = parsed.pathname.split('/').filter(Boolean).pop() || 'video';
          filename = base.replace(/\.m3u8$/i, '') + '.mp4';
        } catch (_) {
          filename = 'video.mp4';
        }
      }
      probeResult = {
        filename: filename,
        totalSize: 0,
        resumable: true
      };
      return;
    }

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
    if (isProbing) return; // Wait for size calculation to finish first

    if (isHlsLink(url)) {
      isSubmitting = true;
      try {
        let finalName = filename.trim();
        if (!finalName) {
          finalName = 'video.mp4';
        } else if (finalName.endsWith('.m3u8')) {
          finalName = finalName.replace(/\.m3u8$/i, '.mp4');
        } else if (!finalName.endsWith('.mp4')) {
          finalName = finalName + '.mp4';
        }
        await DownloadYouTubeVideo(url.trim(), 'best', saveDir, finalName, 0);
        if (isStandalone) {
          await ClosePromptWindow(true, finalName, 0);
        }
        onAdded();
        handleClose();
      } catch (err) {
        probeError = String(err);
      } finally {
        isSubmitting = false;
      }
      return;
    }

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

  async function handleBatchSubmit() {
    const selected = batchItems.filter(it => it.selected);
    if (selected.length === 0) return;
    if (isBatchProbing) return; // Wait for size calculation to finish first

    const enforce = localStorage.getItem(STORAGE_KEYS.ENFORCE_EXTENSIONS) === 'true';
    if (enforce) {
      const blocked = selected.filter(it => !isExtensionAccepted(it.filename));
      if (blocked.length > 0) {
        probeError = `${blocked.length} selected item(s) have unapproved extensions. Uncheck them or allow their extension.`;
        return;
      }
    }

    isSubmitting = true;

    try {
      const requests = selected.map(it => ({
        url: it.url,
        filename: it.filename || '',
        saveDir: saveDir,
        connections: parseInt(connections, 10) || 4,
        headers: {},
        totalSize: it.totalSize || 0,
        resumable: !!it.resumable
      }));

      await AddBatchDownloads(requests);
      if (isStandalone) {
        await ClosePromptWindow(true, `${selected.length} downloads`, 0);
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
    activeMode = 'single';
    batchText = '';
    batchItems = [];
    isBatchProbing = false;
    isYouTube = false;
    isAnalyzingYT = false;
    ytMeta = null;
    ytError = '';
    selectedYTFormat = null;
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
    } else if (event.key === 'Enter' && !isSubmitting) {
      if (event.target && event.target.tagName === 'TEXTAREA') {
        return;
      }
      if (isYouTube) {
        if (selectedYTFormat) handleYouTubeSubmit();
      } else if (isBatchMode) {
        if (selectedCount > 0) handleBatchSubmit();
      } else if (url.trim()) {
        handleSubmit();
      }
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
        : `bg-[var(--color-surface)] border border-[var(--color-border)] rounded-3xl w-full ${isBatchMode ? 'max-w-2xl' : 'max-w-lg'} shadow-2xl overflow-hidden animate-in zoom-in-95 duration-150`}
      style={isStandalone ? "--wails-draggable: drag;" : ""}
      role="dialog"
      aria-modal="true"
      aria-labelledby="modal-title"
    >
      <!-- Header: Only rendered for in-app modal (header entirely removed in popup window) -->
      {#if !isStandalone}
        <div class="px-6 py-5 border-b border-[var(--color-border)] flex items-center justify-between bg-[var(--color-bg)]/60">
          <div class="flex items-center gap-3">
            <div class="w-10 h-10 rounded-2xl {isYouTube ? 'bg-red-500/10 text-red-500' : 'bg-[var(--color-accent-subtle)] text-[var(--color-accent)]'} flex items-center justify-center">
              {#if isYouTube}
                <Film size={20} strokeWidth={2.5} />
              {:else if isBatchMode}
                <Link size={20} strokeWidth={2.5} />
              {:else}
                <Download size={20} strokeWidth={2.5} />
              {/if}
            </div>
            <div>
              <h3 id="modal-title" class="font-bold text-[var(--color-text)] text-base">
                {isYouTube ? 'YouTube Video Downloader' : (isBatchMode ? 'Batch Multi-Download' : (grabbedData ? 'Download Prompt' : 'Add New Download'))}
              </h3>
              <p class="text-xs text-[var(--color-secondary)]">
                {isYouTube ? (ytMeta?.title || 'Select resolution and download YouTube video') : (isBatchMode ? `${batchItems.length} link${batchItems.length === 1 ? '' : 's'} detected • Pre-flight review & batch enqueue` : (grabbedData ? 'URL captured • Review details and start download' : 'Multi-thread concurrent chunking with resume support'))}
              </p>
            </div>
          </div>
          <button 
            on:click={handleClose}
            class="text-[var(--color-secondary)] hover:text-[var(--color-text)] p-1.5 rounded-xl hover:bg-[var(--color-border)]/50 transition cursor-pointer"
            aria-label={$t('common.close')}
          >
            <X size={18} />
          </button>
        </div>
      {/if}

      <!-- Form Body -->
      <form on:submit|preventDefault={isYouTube ? handleYouTubeSubmit : (isBatchMode ? handleBatchSubmit : handleSubmit)} class="{isStandalone ? 'p-4 space-y-3' : 'p-6 space-y-4'} flex-1 overflow-y-auto">
        <!-- Explicit Mode Tabs: Single File vs Batch URLs -->
        {#if !isStandalone && !isYouTube}
          <div class="flex items-center p-1 bg-[var(--color-input)] rounded-2xl border border-[var(--color-border)]">
            <button
              type="button"
              on:click={() => setMode('single')}
              class="flex-1 py-1.5 text-xs font-semibold rounded-xl transition flex items-center justify-center gap-1.5 cursor-pointer {activeMode === 'single' ? 'bg-[var(--color-surface)] text-[var(--color-accent)] shadow-xs' : 'text-[var(--color-secondary)] hover:text-[var(--color-text)]'}"
            >
              <Download size={13} strokeWidth={2.2} />
              {$t('modals.add.tab_direct')}
            </button>
            <button
              type="button"
              on:click={() => setMode('batch')}
              class="flex-1 py-1.5 text-xs font-semibold rounded-xl transition flex items-center justify-center gap-1.5 cursor-pointer {activeMode === 'batch' ? 'bg-[var(--color-surface)] text-[var(--color-accent)] shadow-xs' : 'text-[var(--color-secondary)] hover:text-[var(--color-text)]'}"
            >
              <Link size={13} strokeWidth={2.2} />
              {$t('modals.add.tab_batch')}
              {#if batchItems.length > 0}
                <span class="px-1.5 py-0.2 text-[10px] rounded-full bg-[var(--color-accent)]/15 text-[var(--color-accent)] font-mono">
                  {batchItems.length}
                </span>
              {/if}
            </button>
          </div>
        {/if}

        {#if isYouTube}
          <div class="space-y-3.5">
            <!-- YouTube URL Banner -->
            <div class="flex items-center justify-between p-2 rounded-xl bg-red-500/10 border border-red-500/20 text-xs">
              <div class="flex items-center gap-2 min-w-0">
                <Film size={15} class="text-red-500 shrink-0" />
                <span class="font-bold text-red-500 text-[11px] uppercase tracking-wide">YouTube</span>
                <span class="text-[11px] text-[var(--color-secondary)] font-mono truncate">{url}</span>
              </div>
              <button
                type="button"
                on:click={() => { isYouTube = false; handleProbe(); }}
                class="text-[11px] text-[var(--color-secondary)] hover:text-[var(--color-text)] underline shrink-0 cursor-pointer ml-2"
              >
                Standard mode
              </button>
            </div>

            {#if isAnalyzingYT}
              <div class="py-12 flex flex-col items-center justify-center gap-3">
                <Loader2 size={30} class="animate-spin text-[var(--color-accent)]" />
                <p class="text-xs font-semibold text-[var(--color-text)]">Analyzing YouTube stream...</p>
                <p class="text-[11px] text-[var(--color-secondary)]">Extracting video formats and audio options</p>
              </div>
            {:else if ytError}
              <div class="p-3 bg-rose-500/10 border border-rose-500/20 rounded-xl space-y-2">
                <div class="flex items-start gap-2 text-rose-500 text-xs">
                  <AlertCircle size={15} class="shrink-0 mt-0.5" />
                  <span class="break-all">{ytError}</span>
                </div>
                <button
                  type="button"
                  on:click={() => fetchYouTubeDetails(url)}
                  class="text-xs px-3 py-1 bg-rose-500/20 text-rose-600 dark:text-rose-400 rounded-lg hover:bg-rose-500/30 transition cursor-pointer font-medium"
                >
                  Retry
                </button>
              </div>
            {:else if ytMeta}
              <!-- Video Metadata Card -->
              <div class="flex gap-3 p-2.5 rounded-2xl bg-[var(--color-input)]/50 border border-[var(--color-border)]">
                {#if ytMeta.thumbnail}
                  <div class="relative w-32 aspect-video rounded-xl overflow-hidden bg-black/40 shrink-0 border border-[var(--color-border)]">
                    <img src={ytMeta.thumbnail} alt={ytMeta.title} class="w-full h-full object-cover" />
                    {#if ytMeta.duration_str}
                      <span class="absolute bottom-1 right-1 px-1.5 py-0.2 rounded text-[10px] font-mono font-bold bg-black/80 text-white">
                        {ytMeta.duration_str}
                      </span>
                    {/if}
                  </div>
                {/if}
                <div class="flex-1 min-w-0 flex flex-col justify-between py-0.5">
                  <div>
                    <h4 class="text-xs font-bold text-[var(--color-text)] line-clamp-2 leading-tight" title={ytMeta.title}>
                      {ytMeta.title}
                    </h4>
                    {#if ytMeta.uploader}
                      <p class="text-[11px] text-[var(--color-secondary)] mt-1 truncate">
                        {ytMeta.uploader}
                      </p>
                    {/if}
                  </div>
                  <p class="text-[10px] text-[var(--color-secondary)]/80 font-mono mt-1 truncate">
                    {filename}
                  </p>
                </div>
              </div>

              <!-- Format / Resolution Selector -->
              <div>
                <span class="block text-xs font-semibold text-[var(--color-text)] mb-1.5">
                  Select Resolution / Format
                </span>
                <div class="grid grid-cols-2 gap-2 max-h-44 overflow-y-auto pr-1">
                  {#each ytMeta.options || [] as opt}
                    <button
                      type="button"
                      on:click={() => selectYTFormat(opt)}
                      class="p-2 rounded-xl border text-left transition flex items-center justify-between cursor-pointer {selectedYTFormat === opt ? 'border-[var(--color-accent)] bg-[var(--color-accent-subtle)] text-[var(--color-accent)] ring-1 ring-[var(--color-accent)]' : 'border-[var(--color-border)] bg-[var(--color-input)]/40 hover:bg-[var(--color-input)] text-[var(--color-text)]'}"
                    >
                      <div class="flex items-center gap-2 min-w-0">
                        {#if opt.is_audio}
                          <Music size={14} class="shrink-0 {selectedYTFormat === opt ? 'text-[var(--color-accent)]' : 'text-[var(--color-secondary)]'}" />
                        {:else}
                          <Video size={14} class="shrink-0 {selectedYTFormat === opt ? 'text-[var(--color-accent)]' : 'text-[var(--color-secondary)]'}" />
                        {/if}
                        <div class="truncate">
                          <div class="text-xs font-bold truncate leading-tight">{opt.resolution}</div>
                          <div class="text-[10px] text-[var(--color-secondary)] uppercase">{opt.ext}</div>
                        </div>
                      </div>
                      {#if opt.filesize > 0}
                        <span class="text-[11px] font-mono font-medium shrink-0 ml-1.5 opacity-90">
                          {formatBytes(opt.filesize)}
                        </span>
                      {/if}
                    </button>
                  {/each}
                </div>
              </div>

              <!-- Save Directory -->
              <div>
                <label for="yt-save-dir" class="block text-xs font-semibold text-[var(--color-text)] mb-1">Save Path</label>
                <div class="flex gap-2">
                  <input 
                    id="yt-save-dir"
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
            {/if}
          </div>
        {:else if isBatchMode}
          <!-- Batch URL Importer (Multi-line field / Free text) -->
          <div>
            <div class="flex items-center justify-between mb-1.5">
              <label for="batch-urls" class="text-xs font-semibold text-[var(--color-text)] flex items-center gap-1.5">
                <Link size={13} class="text-[var(--color-accent)]" />
                Batch Links / Free Text
              </label>
              <span class="text-[11px] text-[var(--color-secondary)] font-medium">
                {batchItems.length} link{batchItems.length === 1 ? '' : 's'} detected
              </span>
            </div>
            <textarea
              id="batch-urls"
              bind:value={batchText}
              on:input={handleBatchInput}
              placeholder="Paste links (one per line, comma-separated) or free text containing URLs...&#10;https://example.com/file1.zip&#10;https://example.com/file2.zip"
              rows="4"
              class="w-full bg-[var(--color-input)] border border-[var(--color-border)] rounded-xl px-3.5 py-2.5 text-xs font-mono text-[var(--color-text)] placeholder-[var(--color-secondary)]/60 focus:outline-none focus:border-[var(--color-accent)] focus:ring-2 focus:ring-[var(--color-accent)]/20 resize-y transition"
            ></textarea>
            <p class="text-[11px] text-[var(--color-secondary)]/75 mt-1">
              Candak automatically detects and extracts distinct HTTP/HTTPS links from multi-line lists or unstructured text.
            </p>
          </div>

          <!-- Pre-Flight Batch Review List & Controls -->
          <div>
            <div class="flex items-center justify-between mb-2">
              <span class="text-xs font-semibold text-[var(--color-text)]">
                Pre-Flight Review ({selectedCount}/{batchItems.length} selected)
              </span>
              <div class="flex items-center gap-1.5">
                <button
                  type="button"
                  on:click={handleSelectAll}
                  class="text-[11px] font-medium text-[var(--color-secondary)] hover:text-[var(--color-text)] px-2 py-0.5 rounded-lg border border-[var(--color-border)]/60 bg-[var(--color-bg)] transition cursor-pointer"
                >
                  Select All
                </button>
                {#if hasErrors}
                  <button
                    type="button"
                    on:click={handleDeselectBroken}
                    class="text-[11px] font-medium text-rose-500 hover:text-rose-600 px-2 py-0.5 rounded-lg border border-rose-500/30 bg-rose-500/10 transition cursor-pointer"
                  >
                    Deselect Broken
                  </button>
                {/if}
              </div>
            </div>

            <!-- Scrollable List of Detected URLs -->
            <div class="space-y-1.5 max-h-52 overflow-y-auto pr-1">
              {#each batchItems as item, idx (item.url + '_' + idx)}
                <div class="flex items-center gap-2.5 p-2 rounded-xl border border-[var(--color-border)]/60 bg-[var(--color-input)]/40 hover:bg-[var(--color-input)] transition">
                  <input
                    type="checkbox"
                    bind:checked={item.selected}
                    class="accent-[var(--color-accent)] w-4 h-4 rounded cursor-pointer shrink-0"
                  />
                  <div class="flex-1 min-w-0">
                    <div class="flex items-center gap-1.5 mb-0.5">
                      <span class="px-1.5 py-0.2 rounded text-[10px] font-mono font-semibold bg-[var(--color-surface)] border border-[var(--color-border)] text-[var(--color-secondary)] shrink-0">
                        {item.domain}
                      </span>
                      <span class="text-xs font-semibold text-[var(--color-text)] truncate" title={item.filename}>
                        {item.filename}
                      </span>
                      {#if item.blockedExt}
                        <span class="px-1.5 py-0.2 rounded text-[10px] font-semibold bg-amber-500/10 text-amber-500 border border-amber-500/20 shrink-0">
                          Blocked ext
                        </span>
                      {/if}
                    </div>
                    <div class="text-[11px] text-[var(--color-secondary)]/80 font-mono truncate" title={item.url}>
                      {item.url}
                    </div>
                  </div>
                  <div class="shrink-0 text-right">
                    {#if item.status === 'probing'}
                      <span class="flex items-center gap-1 text-[11px] text-[var(--color-accent)] font-medium">
                        <Loader2 size={12} class="animate-spin" />
                        Probing...
                      </span>
                    {:else if item.status === 'valid'}
                      <div class="flex items-center gap-1.5">
                        <span class="text-[11px] font-bold text-[var(--color-accent)] bg-[var(--color-accent-subtle)] px-2 py-0.5 rounded-md">
                          {item.totalSize > 0 ? formatBytes(item.totalSize) : 'Unknown size'}
                        </span>
                        <CheckCircle size={14} class="text-emerald-500" />
                      </div>
                    {:else if item.status === 'error'}
                      <div class="flex items-center gap-1 text-[11px] text-rose-500 bg-rose-500/10 border border-rose-500/20 px-2 py-0.5 rounded-md max-w-[130px] truncate" title={item.error}>
                        <AlertCircle size={12} class="shrink-0" />
                        <span class="truncate">{item.error || 'Failed'}</span>
                      </div>
                    {/if}
                  </div>
                </div>
              {/each}
            </div>
          </div>

          <!-- Unified Save Directory & Connections -->
          <div class="space-y-3 pt-2 border-t border-[var(--color-border)]/50">
            <div>
              <label for="batch-save-dir" class="block text-xs font-semibold text-[var(--color-text)] mb-1">Destination Directory</label>
              <div class="flex gap-2">
                <input 
                  id="batch-save-dir"
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

            <div>
              <div class="flex items-center justify-between mb-1">
                <label for="batch-connections" class="text-xs font-semibold text-[var(--color-text)] flex items-center gap-1">
                  <Zap size={13} class="text-[var(--color-accent)]" />
                  Parallel Connections (per download)
                </label>
                <span class="text-xs font-bold text-[var(--color-accent)]">{connections} threads</span>
              </div>
              <input 
                id="batch-connections"
                type="range" 
                min="1" 
                max="16" 
                step="1" 
                bind:value={connections}
                class="w-full accent-[var(--color-accent)] bg-[var(--color-border)] h-1.5 rounded-lg cursor-pointer"
              />
            </div>
          </div>
        {:else}
          <!-- Single URL Input with Simplified Inline File Size & Copy Button -->
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
                  <label for="download-dir" class="block text-xs font-semibold text-[var(--color-text)] mb-1">{$t('modals.add.save_path_label')}</label>
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
                      {$t('modals.add.browse')}
                    </button>
                  </div>
                </div>

                <!-- Concurrent Chunks / Connections -->
                <div>
                  <div class="flex items-center justify-between mb-1">
                    <label for="download-connections" class="text-xs font-semibold text-[var(--color-text)] flex items-center gap-1">
                      <Zap size={13} class="text-[var(--color-accent)]" />
                      {$t('modals.add.threads_label')}
                    </label>
                    <span class="text-xs font-bold text-[var(--color-accent)]">{$t('modals.settings.chunks_count', { count: connections })}</span>
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
        {/if}

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
              {$t('common.cancel')}
            </button>
            <button 
              type={isYouTube ? "button" : "submit"} 
              on:click={isYouTube ? handleYouTubeSubmit : undefined}
              disabled={isSubmitting || (isYouTube ? (!selectedYTFormat || isAnalyzingYT) : (isBatchMode ? (isBatchProbing || selectedCount === 0) : (isProbing || !url.trim())))}
              class="px-5 py-2 text-xs font-bold text-white bg-[var(--color-accent)] hover:bg-[var(--color-accent-hover)] rounded-xl shadow-sm hover:shadow transition flex items-center gap-2 disabled:opacity-50 cursor-pointer"
            >
              {#if isSubmitting}
                <Loader2 size={14} class="animate-spin" />
                {$t('modals.update.starting')}
              {:else if isYouTube}
                <Download size={14} strokeWidth={2.5} />
                {$t('modals.history.download')} {selectedYTFormat?.resolution || 'Video'}
              {:else if (isBatchMode ? isBatchProbing : isProbing)}
                <Loader2 size={14} class="animate-spin" />
                Calculating...
              {:else}
                <Download size={14} strokeWidth={2.5} />
                {isBatchMode ? `${$t('modals.history.download')} (${selectedCount})` : $t('modals.history.download')}
              {/if}
            </button>
          </div>
        </div>
      </form>
    </div>
  </div>
{/if}
