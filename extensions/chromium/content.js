// Candak Download Manager - Video Grabber Overlay (content script)
// One isolated Shadow DOM singleton per frame, bound on hover/playback to the
// hovered or playing <video>, docking to its exterior top-right (or bottom-right
// when clipped) edge and handing streams to Candak via the background worker.
(() => {
  if (window.__candakOverlayLoaded) return;
  window.__candakOverlayLoaded = true;

  if (!/^https?:$/.test(location.protocol)) return;

  const IS_YOUTUBE = /(^|\.)youtube(-nocookie)?\.com$/.test(location.hostname);
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

  // --- Overlay state ---------------------------------------------------------
  let host = null;
  let pill = null;
  let menu = null;
  let grabLabelEl = null;
  let caretEl = null;
  let targetVideo = null;
  let dockedTop = true;
  let manualOffset = null; // Viewport-locked drag offset for this session
  let curX = 0;
  let curY = 0;
  let visible = true;
  let rafId = 0;
  let intersectionObserver = null;
  let dismissed = new WeakSet(); // Videos the user closed the overlay for
  let variants = []; // Offered streams: {label, url, ext, size?, tracks?}
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
    caretEl = root.querySelector(".caret");

    if (IS_YOUTUBE) {
      caretEl.style.display = "none";
    }

    root.querySelector(".grab").addEventListener("click", () => {
      if (isDrm) return;
      if (IS_YOUTUBE) {
        grabYouTubeVideo();
        return;
      }
      if (variants.length > 0) grabVariant(variants[0]);
    });

    caretEl.addEventListener("click", (e) => {
      if (isDrm) return;
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
    scheduleSync();
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
    variants = [];
    ensureUI();
    pill.classList.add("on");
    pill.style.visibility = "visible";
    loadVariants();
    startTracking();
    scheduleSync();
  }

  function detach() {
    stopTracking();
    targetVideo = null;
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

  let isDrm = false;

  function hasDrmProtection(v) {
    if (!v) return false;
    if (v.mediaKeys !== null && v.mediaKeys !== undefined) return true;
    if (v.getAttribute && (v.getAttribute("data-key-system") || v.getAttribute("keysystem"))) return true;
    return false;
  }

  async function loadVariants() {
    const v = targetVideo;
    if (!v) return;

    if (IS_YOUTUBE) {
      // YouTube is handled by Candak's dedicated YouTube downloader
      if (grabLabelEl) grabLabelEl.textContent = "Download video";
      scheduleSync();
      return;
    }

    let hasDrm = hasDrmProtection(v);
    const streams = await queryStreams();
    if (targetVideo !== v) return;

    variants = [];
    for (const s of streams) {
      if (variants.some(x => x.url === s.url)) continue;
      if (/mpegurl/i.test(s.mimeType) || /\.m3u8$/i.test(s.url)) {
        try {
          const res = await chrome.runtime.sendMessage({ type: "candak:parseHLS", url: s.url });
          if (targetVideo !== v) return;
          if (res && res.isDrm) {
            hasDrm = true;
          }
          if (res && res.variants) {
            for (const hv of res.variants) {
              const hLabel = hv.label || (v.videoHeight > 0 ? `${v.videoHeight}p` : "");
              variants.push({
                label: hLabel || (hv.bandwidth ? `${(hv.bandwidth / 1e6).toFixed(1)} Mbps` : "Auto Quality"),
                url: hv.url,
                ext: "mp4",
                tracks: hv.tracks
              });
            }
          }
        } catch (_) {}
      } else {
        let label = "";
        if (v.videoHeight > 0) {
          label = `${v.videoHeight}p`;
        }
        if (!label) {
          const ext = extFromMime(s.mimeType) || extFromUrl(s.url);
          label = ext ? `${ext.toUpperCase()} Video` : "Video Stream";
        }
        variants.push({
          label,
          size: formatSize(s.sizeBytes),
          url: s.url,
          ext: extFromMime(s.mimeType) || extFromUrl(s.url)
        });
      }
    }

    isDrm = hasDrm;
    const btn = root && root.querySelector(".grab");
    if (isDrm) {
      variants = [];
      if (btn) {
        btn.disabled = true;
        btn.style.opacity = "0.55";
        btn.style.cursor = "not-allowed";
        btn.title = "This stream is DRM-protected and cannot be downloaded.";
      }
      if (grabLabelEl) {
        grabLabelEl.textContent = "🔒 DRM Protected";
      }
      if (caretEl) caretEl.style.display = "none";
      closeMenu();
      scheduleSync();
      return;
    }

    if (btn) {
      btn.disabled = false;
      btn.style.opacity = "1";
      btn.style.cursor = "pointer";
      btn.title = "";
    }
    if (caretEl && !IS_YOUTUBE) {
      caretEl.style.display = variants.length > 1 ? "" : "none";
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
    if (variants.length === 0) {
      const li = document.createElement("li");
      li.textContent = "No stream detected yet";
      li.style.cursor = "default";
      li.style.color = "#64748b";
      menu.appendChild(li);
    }
    if (grabLabelEl) {
      const first = variants[0];
      const arrow = variants.length > 1 ? " ▾" : "";
      if (first && first.label) {
        grabLabelEl.textContent = `Download ${first.label}${arrow}`;
      } else {
        grabLabelEl.textContent = `Download video${arrow}`;
      }
    }
    if (caretEl) {
      caretEl.style.display = variants.length > 1 ? "" : "none";
    }
    scheduleSync();
  }

  function openMenu() {
    if (!menu || IS_YOUTUBE) return;
    menuOpen = true;
    renderMenu();
    menu.classList.add("on");
    scheduleSync();
  }

  function closeMenu() {
    menuOpen = false;
    if (menu) menu.classList.remove("on");
  }

  function sanitizeTitle(title) {
    return (title || "video").replace(/[\\/:*?"<>|]+/g, "").trim().slice(0, 120) || "video";
  }

  async function grabVariant(variant) {
    if (!variant) return;
    const filename = `${sanitizeTitle(document.title)} (${variant.label}).${variant.ext}`;
    try {
      await chrome.runtime.sendMessage({
        type: "candak:grab",
        payload: {
          url: variant.url,
          filename,
          headers: {
            "User-Agent": navigator.userAgent,
            Referer: location.href
          }
        }
      });
    } catch (err) {
      console.warn("[Candak] Grab handoff failed:", err);
    }
  }

  async function grabYouTubeVideo() {
    const filename = `${sanitizeTitle(document.title)}.mp4`;
    try {
      await chrome.runtime.sendMessage({
        type: "candak:grab",
        payload: {
          url: location.href,
          filename,
          headers: {
            "User-Agent": navigator.userAgent,
            Referer: location.href
          }
        }
      });
    } catch (err) {
      console.warn("[Candak] YouTube grab handoff failed:", err);
    }
  }

  // Background nudges us whenever a new media response is sniffed
  chrome.runtime.onMessage.addListener((msg) => {
    if (msg.type === "candak:streamsUpdated") {
      if (targetVideo && pill && pill.classList.contains("on")) {
        loadVariants();
      }
    }
    return false;
  });

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
