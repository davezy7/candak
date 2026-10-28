// Candak Download Manager - Video Grabber Overlay (content script)
// One isolated Shadow DOM singleton per frame, bound on hover/playback to the
// hovered or playing <video>, docking to its exterior top-right (or bottom-right
// when clipped) edge and handing streams to Candak via the background worker.
(() => {
  if (window.__candakOverlayLoaded) return;
  window.__candakOverlayLoaded = true;

  if (!/^https?:$/.test(location.protocol)) return;

  const MIN_VIDEO_W = 200;
  const MIN_VIDEO_H = 150;
  const GAP = 8;
  const DEBOUNCE_MS = 200;
  const EDGE_PAD = 4;

  // --- Settings ------------------------------------------------------------
  let settings = { overlayEnabled: true, excludedSites: [] };
  function enabledForSite() {
    return settings.overlayEnabled && !settings.excludedSites.includes(location.hostname);
  }
  chrome.storage.local.get({ overlayEnabled: true, excludedSites: [] }, (res) => {
    settings = res;
    if (!enabledForSite()) detach();
  });
  chrome.storage.onChanged.addListener((changes) => {
    if (changes.overlayEnabled) settings.overlayEnabled = changes.overlayEnabled.newValue;
    if (changes.excludedSites) settings.excludedSites = changes.excludedSites.newValue;
    if (!enabledForSite()) {
      detach();
    } else {
      scheduleSync();
    }
  });

  // --- YouTube page-world bridge ------------------------------------------------
  // Isolated content-script worlds cannot see page globals (ytInitialPlayerResponse,
  // movie_player methods), and strict page CSP/Trusted Types block inline-script
  // injection. The bridge therefore ships as page-bridge.js, a world:"MAIN"
  // content script declared in the manifest (Chromium 111+, Firefox 128+).
  // Formats flow back and quality-switch commands flow forward via window.postMessage.
  let ytFormats = null; // Slim format list last relayed by the bridge

  window.addEventListener("message", (e) => {
    if (e.source !== window || !e.data || e.data.source !== "candak-page") return;
    ytFormats = e.data.formats || null;
    if (targetVideo && pill && pill.classList.contains("on")) loadVariants();
  });

  // --- Overlay state ---------------------------------------------------------
  let host = null;
  let pill = null;
  let menu = null;
  let grabLabelEl = null;
  let targetVideo = null;
  let dockedTop = true;
  let manualOffset = null; // Viewport-locked drag offset for this session
  let curX = 0;
  let curY = 0;
  let visible = true;
  let rafId = 0;
  let intersectionObserver = null;
  let dismissed = new WeakSet(); // Videos the user closed the overlay for
  let variants = []; // Offered streams: {label, url, ext, itag?, height?}
  let unbuffered = []; // YouTube metadata qualities not yet intercepted
  let pendingGrab = null; // {itag, label} waiting for interception after quality switch
  let menuOpen = false;

  // --- UI construction -------------------------------------------------------
  const HOST_ID = "candak-grabber-overlay-host";

  function ensureUI() {
    if (host && host.isConnected) return;
    host = document.createElement("div");
    host.id = HOST_ID;
    const root = host.attachShadow({ mode: "open" });
    root.innerHTML = `
      <style>
        :host { all: initial; }
        .pill, .menu {
          position: fixed;
          left: 0;
          top: 0;
          will-change: transform;
          font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
          font-size: 12px;
          color: #f1f5f9;
          background: #0f172a;
          border: 1px solid #334155;
          border-radius: 8px;
          box-shadow: 0 4px 16px rgba(0, 0, 0, 0.45);
          z-index: 2147483647;
          user-select: none;
          -webkit-user-select: none;
        }
        .pill {
          display: none;
          align-items: center;
          gap: 2px;
          padding: 4px 6px;
        }
        .pill.on { display: inline-flex; }
        .pill button {
          all: unset;
          cursor: pointer;
          color: #f1f5f9;
          padding: 3px 7px;
          border-radius: 5px;
          line-height: 1.2;
          white-space: nowrap;
        }
        .pill button:hover { background: #1e293b; color: #38bdf8; }
        .grip { cursor: grab !important; color: #94a3b8 !important; letter-spacing: -1px; }
        .close { color: #94a3b8 !important; font-size: 11px; }
        .menu {
          display: none;
          list-style: none;
          margin: 0;
          padding: 4px;
          max-height: 240px;
          overflow-y: auto;
          min-width: 150px;
        }
        .menu.on { display: block; }
        .menu li {
          padding: 6px 10px;
          border-radius: 5px;
          cursor: pointer;
          white-space: nowrap;
        }
        .menu li:hover { background: #1e293b; color: #38bdf8; }
        .menu li .tracks { color: #64748b; font-size: 10px; margin-left: 6px; }
        .menu li.pending .tracks { color: #f59e0b; }
      </style>
      <div class="pill">
        <button class="grip" title="Drag to move (double-click resets)">⠿</button>
        <button class="grab">⬇ <span class="grab-label">Download video</span></button>
        <button class="caret" title="Resolutions">▾</button>
        <button class="close" title="Dismiss overlay">✕</button>
      </div>
      <ul class="menu"></ul>
    `;
    (document.body || document.documentElement).appendChild(host);
    pill = root.querySelector(".pill");
    menu = root.querySelector(".menu");
    grabLabelEl = root.querySelector(".grab-label");

    root.querySelector(".grab").addEventListener("click", () => {
      if (variants.length > 0) grabVariant(variants[0]);
    });
    root.querySelector(".caret").addEventListener("click", (e) => {
      e.stopPropagation();
      menuOpen ? closeMenu() : openMenu();
    });
    root.querySelector(".close").addEventListener("click", () => {
      if (targetVideo) dismissed.add(targetVideo);
      detach();
    });

    const grip = root.querySelector(".grip");
    let drag = null;
    grip.addEventListener("pointerdown", (e) => {
      if (e.button !== 0) return;
      drag = { startX: e.clientX, startY: e.clientY, baseX: curX, baseY: curY, moved: false };
      grip.setPointerCapture(e.pointerId);
    });
    grip.addEventListener("pointermove", (e) => {
      if (!drag) return;
      const dx = e.clientX - drag.startX;
      const dy = e.clientY - drag.startY;
      if (!drag.moved && Math.abs(dx) + Math.abs(dy) < 3) return;
      drag.moved = true;
      manualOffset = { x: drag.baseX + dx, y: drag.baseY + dy };
      scheduleSync();
    });
    grip.addEventListener("pointerup", () => { drag = null; });
    grip.addEventListener("dblclick", () => {
      manualOffset = null;
      scheduleSync();
    });

    // Close the menu on clicks outside the overlay
    document.addEventListener("click", (e) => {
      if (menuOpen && !e.composedPath().includes(host)) closeMenu();
    }, true);
  }

  // --- Video discovery ---------------------------------------------------------
  function debounce(fn, ms) {
    let t;
    return () => {
      clearTimeout(t);
      t = setTimeout(fn, ms);
    };
  }

  const mo = new MutationObserver(debounce(() => {
    if (!enabledForSite()) return;
    scheduleSync(); // Layout shifts (videos added/removed) move the target rect
  }, DEBOUNCE_MS));
  mo.observe(document.documentElement, { childList: true, subtree: true });

  function isCandidate(v) {
    if (v.videoWidth > 0 && (v.videoWidth < MIN_VIDEO_W || v.videoHeight < MIN_VIDEO_H)) return false;
    return true;
  }

  // --- Activation (hover or playback) -----------------------------------------
  document.addEventListener("mouseover", (e) => {
    const v = e.composedPath().find(n => n instanceof HTMLVideoElement);
    if (v) activate(v);
  }, true);
  document.addEventListener("play", (e) => {
    if (e.target instanceof HTMLVideoElement) activate(e.target);
  }, true);

  function activate(v) {
    if (!enabledForSite() || dismissed.has(v) || !isCandidate(v)) return;
    if (targetVideo === v) {
      scheduleSync();
      return;
    }
    targetVideo = v;
    manualOffset = null;
    pendingGrab = null;
    variants = [];
    unbuffered = [];
    ensureUI();
    pill.classList.add("on");
    pill.style.visibility = "visible";
    // Nudge the page-world bridge in case formats arrived after injection
    window.postMessage({ source: "candak-content", cmd: "collect" }, location.origin);
    loadVariants();
    startTracking();
    scheduleSync();
  }

  function detach() {
    stopTracking();
    targetVideo = null;
    pendingGrab = null;
    if (pill) {
      pill.classList.remove("on");
      closeMenu();
    }
  }

  // --- Stream variants ----------------------------------------------------------
  async function queryStreams() {
    try {
      const res = await chrome.runtime.sendMessage({ type: "candak:getStreams" });
      return (res && res.streams) || [];
    } catch (err) {
      return [];
    }
  }

  function extFromMime(mime) {
    if (!mime) return "mp4";
    if (/mpegurl/i.test(mime)) return "m3u8";
    if (/webm/i.test(mime)) return "webm";
    if (/dash\+xml/i.test(mime)) return "mpd";
    return "mp4";
  }

  function extFromUrl(url) {
    const m = /\.(mp4|m3u8|webm|mpd)(?:$|\?)/i.exec(url);
    return m ? m[1].toLowerCase() : "mp4";
  }

  function formatSize(bytes) {
    const n = Number(bytes);
    if (!n || n <= 0) return "";
    const units = ["KB", "MB", "GB"];
    let v = n / 1024;
    let i = 0;
    while (v >= 1024 && i < units.length - 1) {
      v /= 1024;
      i++;
    }
    return `${v >= 100 ? v.toFixed(0) : v.toFixed(1)} ${units[i]}`;
  }

  // YouTube player metadata relayed by the page-world bridge, all qualities keyed
  // by itag. Only pre-signed `url` fields are used; signatureCipher is never
  // unpacked (invalid without deciphering), so intercepted URLs win whenever the
  // sniffer has seen them.
  function collectYouTubeMeta() {
    if (!ytFormats || ytFormats.length === 0) return null;
    const meta = new Map();
    for (const f of ytFormats) {
      const mime = (f.mimeType || "").split(";")[0].trim();
      const isAudio = mime.startsWith("audio/");
      meta.set(String(f.itag), {
        itag: String(f.itag),
        label: isAudio ? "Audio" : (f.qualityLabel || (f.height ? `${f.height}p` : "Video")),
        height: f.height || 0,
        ext: extFromMime(mime),
        isAudio,
        hasAudio: isAudio || !!f.hasAudio,
        sizeBytes: Number(f.contentLength) || 0,
        url: f.url || null
      });
    }
    return meta;
  }

  function qualityName(height) {
    if (height >= 2160) return "hd2160";
    if (height >= 1440) return "hd1440";
    if (height >= 1080) return "hd1080";
    if (height >= 720) return "hd720";
    if (height >= 480) return "large";
    if (height >= 360) return "medium";
    if (height >= 240) return "small";
    return "tiny";
  }

  async function loadVariants() {
    const v = targetVideo;
    if (!v) return;
    const streams = await queryStreams();
    if (targetVideo !== v) return; // Target changed while querying

    const ytMeta = collectYouTubeMeta();
    variants = [];
    unbuffered = [];

    if (ytMeta) {
      const byItag = new Map();
      for (const s of streams) {
        if (s.itag) byItag.set(String(s.itag), s);
      }
      // One entry per resolution: the player serves whichever codec it prefers
      // (AVC/VP9/AV1 share "1080p"), so codec duplicates are meaningless.
      // Buffered URLs win over quality-switch entries so already-sniffed
      // streams grab instantly instead of forcing a rebuffer.
      const byQuality = new Map();
      for (const m of ytMeta.values()) {
        const key = m.isAudio ? "audio" : `v${m.height || m.label}`;
        const s = byItag.get(m.itag);
        const url = s ? s.url : m.url; // Intercepted (signed) beats pre-signed
        const prev = byQuality.get(key);
        if (!prev || (!prev.url && url)) byQuality.set(key, { m, url });
      }
      for (const { m, url } of byQuality.values()) {
        if (!url) {
          unbuffered.push(m); // Offered via quality-switch interception
          continue;
        }
        const suffix = !m.isAudio && !m.hasAudio ? " (video-only)" : "";
        variants.push({ label: `${m.label}${suffix}`, size: formatSize(m.sizeBytes), url, ext: m.ext, itag: m.itag, height: m.height });
      }
      variants.sort((a, b) => (b.height || 0) - (a.height || 0));
      unbuffered.sort((a, b) => (b.height || 0) - (a.height || 0));
    } else {
      for (const s of streams) {
        if (variants.some(x => x.url === s.url)) continue;
        if (/mpegurl/i.test(s.mimeType) || /\.m3u8$/i.test(s.url)) {
          // Lazy-parse HLS masters only when the overlay is actually used
          try {
            const res = await chrome.runtime.sendMessage({ type: "candak:parseHLS", url: s.url });
            if (targetVideo !== v) return;
            if (res && res.variants) {
              for (const hv of res.variants) {
                variants.push({
                  label: hv.label || (hv.bandwidth ? `${(hv.bandwidth / 1e6).toFixed(1)} Mbps` : "Original"),
                  url: hv.url,
                  ext: "m3u8",
                  tracks: hv.tracks
                });
              }
            }
          } catch (err) {
            // Manifest fetch failed; skip this stream
          }
        } else {
          // Plain file: resolution is only knowable for the source actually
          // feeding the target video; everything else keeps the size alone.
          const src = v.currentSrc || v.src || "";
          const h = src && s.url === src && v.videoHeight > 0 ? `${v.videoHeight}p` : "";
          variants.push({
            label: h || "Original",
            size: formatSize(s.sizeBytes),
            url: s.url,
            ext: extFromMime(s.mimeType) || extFromUrl(s.url)
          });
        }
      }
    }

    if (targetVideo === v) renderMenu();
  }

  function renderMenu() {
    if (!menu) return;
    menu.innerHTML = "";
    for (const v of variants) {
      const li = document.createElement("li");
      const size = v.size ? ` <span class="tracks">${v.size}</span>` : "";
      const tracks = v.tracks ? ` <span class="tracks">${v.tracks}</span>` : "";
      li.innerHTML = `${v.label}${size}${tracks}`;
      li.addEventListener("click", (e) => {
        e.stopPropagation();
        closeMenu();
        grabVariant(v);
      });
      menu.appendChild(li);
    }
    for (const m of unbuffered) {
      const li = document.createElement("li");
      li.className = "pending";
      const suffix = !m.isAudio && !m.hasAudio ? " (video-only)" : "";
      li.innerHTML = `${m.label}${suffix} <span class="tracks">click to buffer</span>`;
      li.addEventListener("click", (e) => {
        e.stopPropagation();
        closeMenu();
        requestQuality(m);
      });
      menu.appendChild(li);
    }
    if (variants.length + unbuffered.length === 0) {
      const li = document.createElement("li");
      li.textContent = "No stream detected yet";
      li.style.cursor = "default";
      li.style.color = "#64748b";
      menu.appendChild(li);
    }
    if (grabLabelEl) {
      grabLabelEl.textContent = variants.length > 1 ? "Download video ▾" : "Download video";
    }
  }

  function openMenu() {
    if (!menu) return;
    menuOpen = true;
    // Ask the page-world bridge for fresh formats (SPA navigations go stale)
    window.postMessage({ source: "candak-content", cmd: "collect" }, location.origin);
    renderMenu();
    menu.classList.add("on");
    scheduleSync();
  }

  function closeMenu() {
    menuOpen = false;
    if (menu) menu.classList.remove("on");
  }

  function requestQuality(meta) {
    // The player API lives in the page world; the bridge relays the switch
    window.postMessage({ source: "candak-content", cmd: "quality", quality: qualityName(meta.height) }, location.origin);
    const suffix = !meta.isAudio && !meta.hasAudio ? " (video-only)" : "";
    // The player may answer the quality request with a different codec itag than
    // the metadata entry clicked (e.g. VP9 instead of AVC), so the pending grab
    // is matched by resolution, with the itag kept as a metadata-free fallback.
    pendingGrab = { itag: meta.itag, height: meta.height, label: `${meta.label}${suffix}` };
    // The target quality may already sit in the sniffer registry (buffered
    // earlier in the session) — grab immediately instead of waiting for a refetch.
    onStreamsUpdated();
  }

  function sanitizeTitle(title) {
    return (title || "video").replace(/[\\/:*?"<>|]+/g, "").trim().slice(0, 120) || "video";
  }

  async function grabVariant(variant) {
    if (!variant) return;
    const filename = `${sanitizeTitle(document.title)} (${variant.label.replace(" (video-only)", "")}).${variant.ext}`;
    try {
      await chrome.runtime.sendMessage({
        type: "candak:grab",
        payload: {
          url: variant.url,
          filename,
          headers: { Referer: location.href }
        }
      });
    } catch (err) {
      console.warn("[Candak] Grab handoff failed:", err);
    }
  }

  // Background nudges us whenever a new media response is sniffed
  chrome.runtime.onMessage.addListener((msg) => {
    if (msg.type === "candak:streamsUpdated") onStreamsUpdated();
    return false;
  });

  async function onStreamsUpdated() {
    if (!targetVideo || !pill || !pill.classList.contains("on")) return;
    if (pendingGrab) {
      const streams = await queryStreams();
      // Match the newly sniffed stream by resolution via metadata (codec-agnostic);
      // fall back to the exact itag when metadata is unavailable.
      const ytMeta = collectYouTubeMeta();
      const s = streams.find((x) => {
        if (!x.itag) return false;
        const m = ytMeta && ytMeta.get(String(x.itag));
        if (m) return !m.isAudio && pendingGrab.height > 0 && m.height === pendingGrab.height;
        return String(x.itag) === pendingGrab.itag;
      });
      if (s) {
        const p = pendingGrab;
        pendingGrab = null;
        grabVariant({ label: p.label, url: s.url, ext: extFromMime(s.mimeType) });
      }
    } else {
      loadVariants();
    }
  }

  // --- Docking, flipping and GPU-composited tracking ----------------------------
  function startTracking() {
    stopTracking();
    if (intersectionObserver) intersectionObserver.disconnect();
    intersectionObserver = new IntersectionObserver((entries) => {
      visible = entries.length > 0 && entries[0].isIntersecting;
      if (visible) {
        scheduleSync();
      } else if (pill) {
        pill.style.visibility = "hidden";
        closeMenu();
      }
    }, { threshold: 0 });
    if (targetVideo) intersectionObserver.observe(targetVideo);
    addEventListener("scroll", scheduleSync, { passive: true, capture: true });
    addEventListener("resize", scheduleSync, { passive: true });
    rafId = 0;
  }

  function stopTracking() {
    if (intersectionObserver) {
      intersectionObserver.disconnect();
      intersectionObserver = null;
    }
    removeEventListener("scroll", scheduleSync, { capture: true });
    removeEventListener("resize", scheduleSync);
    if (rafId) {
      cancelAnimationFrame(rafId);
      rafId = 0;
    }
  }

  function scheduleSync() {
    if (rafId) return;
    rafId = requestAnimationFrame(() => {
      rafId = 0;
      sync();
    });
  }

  function clampViewport(x, y, w, h) {
    return {
      x: Math.max(EDGE_PAD, Math.min(x, innerWidth - w - EDGE_PAD)),
      y: Math.max(EDGE_PAD, Math.min(y, innerHeight - h - EDGE_PAD))
    };
  }

  function sync() {
    if (!targetVideo || !pill || !pill.classList.contains("on")) return;
    if (!visible) return;
    const rect = targetVideo.getBoundingClientRect();
    if (rect.width === 0 && rect.height === 0) return;
    const pw = pill.offsetWidth;
    const ph = pill.offsetHeight;

    dockedTop = rect.top - ph - GAP >= 0;
    let x = rect.right - pw;
    let y = dockedTop ? rect.top - ph - GAP : rect.bottom + GAP;
    if (manualOffset) {
      x = manualOffset.x;
      y = manualOffset.y;
    }
    const pos = clampViewport(x, y, pw, ph);
    curX = pos.x;
    curY = pos.y;
    pill.style.transform = `translate3d(${curX}px, ${curY}px, 0)`;
    positionMenu();
  }

  function positionMenu() {
    if (!menu || !menuOpen) return;
    const mw = menu.offsetWidth;
    const mh = menu.offsetHeight;
    let x = curX;
    let y = dockedTop ? curY + pill.offsetHeight + GAP : curY - mh - GAP;
    const pos = clampViewport(x, y, mw, mh);
    menu.style.transform = `translate3d(${pos.x}px, ${pos.y}px, 0)`;
  }

  // --- Fullscreen suppression -----------------------------------------------------
  function onFullscreenChange() {
    if (!pill) return;
    if (document.fullscreenElement) {
      pill.style.visibility = "hidden";
      closeMenu();
    } else if (targetVideo) {
      pill.style.visibility = "visible";
      scheduleSync();
    }
  }
  document.addEventListener("fullscreenchange", onFullscreenChange);
})();
