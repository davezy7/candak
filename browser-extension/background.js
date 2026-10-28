// Candak Download Manager - Background Service Worker (Manifest V3)
const CANDAK_LOOPBACK_URL = "http://127.0.0.1:41920";

// Check whether Candak desktop application is active
async function isCandakRunning() {
  try {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), 600);
    const response = await fetch(`${CANDAK_LOOPBACK_URL}/health`, {
      method: "GET",
      signal: controller.signal
    });
    clearTimeout(timeoutId);
    if (!response.ok) return false;
    const data = await response.json();
    return data && data.status === "ok" && data.app === "candak";
  } catch (err) {
    return false;
  }
}

// Retrieve relevant cookies for target URL to preserve authenticated downloads
async function getCookiesForUrl(targetUrl) {
  try {
    const cookies = await chrome.cookies.getAll({ url: targetUrl });
    if (!cookies || cookies.length === 0) return "";
    return cookies.map(c => `${c.name}=${c.value}`).join("; ");
  } catch (err) {
    console.warn("[Candak] Cookie retrieval skipped:", err);
    return "";
  }
}

// Listen for browser download initiation
chrome.downloads.onDeterminingFilename.addListener((downloadItem, suggest) => {
  // Check user enabled state from extension storage
  chrome.storage.local.get({ interceptEnabled: true }, async (result) => {
    if (!result.interceptEnabled) {
      suggest(); // Proceed with standard browser download
      return;
    }

    // Ignore downloads originating directly from blob/data URLs
    if (downloadItem.url.startsWith("blob:") || downloadItem.url.startsWith("data:")) {
      suggest();
      return;
    }

    // Quick health probe to verify Candak desktop is running
    const alive = await isCandakRunning();
    if (!alive) {
      console.log("[Candak] App not reachable, falling back to browser download.");
      suggest(); // Gracefully let native browser downloader proceed
      return;
    }

    // Candak is alive: cancel native browser download and hand over to Candak
    try {
      await chrome.downloads.cancel(downloadItem.id);
      await chrome.downloads.erase({ id: downloadItem.id });
    } catch (cancelErr) {
      console.warn("[Candak] Could not cancel native download:", cancelErr);
    }

    // Extract cookies and referer headers
    const cookieHeader = await getCookiesForUrl(downloadItem.url);
    const headers = {};
    if (cookieHeader) {
      headers["Cookie"] = cookieHeader;
    }
    if (downloadItem.referrer) {
      headers["Referer"] = downloadItem.referrer;
    }

    const payload = {
      url: downloadItem.url,
      filename: downloadItem.filename || "",
      headers: headers
    };

    try {
      const grabResp = await fetch(`${CANDAK_LOOPBACK_URL}/api/grab`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-Candak-Source": "browser-extension"
        },
        body: JSON.stringify(payload)
      });
      if (grabResp.ok) {
        console.log("[Candak] Successfully transferred download:", downloadItem.url);
      } else {
        console.error("[Candak] Grab endpoint returned error:", grabResp.status);
      }
    } catch (grabErr) {
      console.error("[Candak] Failed to send grab request:", grabErr);
    }
  });

  return true; // Keep callback channel open for async determination
});

// ---------------------------------------------------------------------------
// Media Stream Sniffer: tab-scoped registry of media responses observed on
// the network, used by the in-page Video Grabber Overlay (content.js).
// ---------------------------------------------------------------------------

const streamRegistry = new Map(); // tabId -> Map(url -> stream entry)

function isMediaUrl(url) {
  try {
    const path = new URL(url).pathname.toLowerCase();
    return /\.(mp4|m3u8|webm)$/.test(path);
  } catch (err) {
    return false;
  }
}

function isMediaMime(mime) {
  return /^video\//i.test(mime) ||
    /^application\/vnd\.apple\.mpegurl$/i.test(mime) ||
    /^application\/dash\+xml$/i.test(mime);
}

function extractItag(url) {
  const m = /[?&]itag=(\d+)/.exec(url);
  return m ? m[1] : null;
}

// Prefer the total size from Content-Range (range responses report chunk
// length in Content-Length); fall back to Content-Length.
function extractSizeBytes(responseHeaders) {
  let total = null;
  for (const h of responseHeaders) {
    const name = h.name.toLowerCase();
    if (name === "content-range") {
      const slash = h.value.lastIndexOf("/");
      if (slash !== -1) {
        const parsed = parseInt(h.value.slice(slash + 1), 10);
        if (!Number.isNaN(parsed) && parsed > 0) total = parsed;
      }
    } else if (name === "content-length" && total === null) {
      const parsed = parseInt(h.value, 10);
      if (!Number.isNaN(parsed) && parsed > 0) total = parsed;
    }
  }
  return total;
}

function registerStream(tabId, entry) {
  let perTab = streamRegistry.get(tabId);
  if (!perTab) {
    perTab = new Map();
    streamRegistry.set(tabId, perTab);
  }
  perTab.set(entry.url, entry);
  // Fire-and-forget nudge so the overlay can refresh or finish a pending grab
  chrome.tabs.sendMessage(tabId, { type: "candak:streamsUpdated" }).catch(() => {});
}

chrome.webRequest.onHeadersReceived.addListener(
  (details) => {
    if (details.tabId < 0) return; // No originating tab (service worker, prerender)
    const mimeHeader = (details.responseHeaders || []).find(
      h => h.name.toLowerCase() === "content-type"
    );
    const mimeType = mimeHeader ? mimeHeader.value.split(";")[0].trim() : "";
    if (!isMediaMime(mimeType) && !isMediaUrl(details.url)) return;

    registerStream(details.tabId, {
      url: details.url,
      mimeType: mimeType,
      sizeBytes: extractSizeBytes(details.responseHeaders || []),
      itag: extractItag(details.url),
      frameId: details.frameId,
      source: "network"
    });
  },
  { urls: ["<all_urls>"], types: ["media", "xmlhttprequest", "other"] },
  ["responseHeaders"]
);

// Prune tab-scoped registries when tabs close or navigate away
chrome.tabs.onRemoved.addListener((tabId) => {
  streamRegistry.delete(tabId);
});
chrome.tabs.onUpdated.addListener((tabId, info) => {
  if (info.url) streamRegistry.delete(tabId);
});

function findStreamEntry(url) {
  for (const perTab of streamRegistry.values()) {
    const entry = perTab.get(url);
    if (entry) return entry;
  }
  return null;
}

// ---------------------------------------------------------------------------
// Lazy HLS manifest variant parsing (only invoked on user hover/click)
// ---------------------------------------------------------------------------

function parseHlsAttributes(str) {
  const attrs = {};
  const push = (pair) => {
    const eq = pair.indexOf("=");
    if (eq === -1) return;
    attrs[pair.slice(0, eq).trim()] = pair.slice(eq + 1).trim().replace(/^"|"$/g, "");
  };
  let cur = "";
  let inQuotes = false;
  for (const ch of str) {
    if (ch === '"') inQuotes = !inQuotes;
    if (ch === "," && !inQuotes) {
      push(cur);
      cur = "";
      continue;
    }
    cur += ch;
  }
  push(cur);
  return attrs;
}

function classifyHlsTracks(codecs, audioGroup) {
  const c = (codecs || "").toLowerCase();
  const hasVideo = /avc1|hvc1|hevc|av01|vp8|vp9|vp09/.test(c);
  const hasAudio = /mp4a|opus|ac-3|ec-3|flac/.test(c);
  if (hasVideo && hasAudio) return "(Video + Audio)";
  if (hasVideo) return "(Video only)";
  if (hasAudio) return "(Audio only)";
  return "(Video + Audio)";
}

async function parseHlsMasterPlaylist(manifestUrl) {
  const res = await fetch(manifestUrl);
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  const text = await res.text();
  const lines = text.split("\n").map(l => l.trim());

  if (!lines.some(l => l.startsWith("#EXT-X-STREAM-INF"))) {
    // Media playlist (single rendition), not a master playlist
    return [{
      label: null,
      url: manifestUrl,
      bandwidth: null,
      codecs: "",
      tracks: "(Video + Audio)"
    }];
  }

  const variants = [];
  for (let i = 0; i < lines.length; i++) {
    if (!lines[i].startsWith("#EXT-X-STREAM-INF")) continue;
    const attrs = parseHlsAttributes(lines[i].slice(lines[i].indexOf(":") + 1));
    const next = lines[i + 1];
    if (!next || next.startsWith("#")) continue;
    const height = parseInt((attrs["RESOLUTION"] || "").split("x")[1], 10);
    variants.push({
      label: !Number.isNaN(height) && height > 0 ? `${height}p` : null,
      url: new URL(next, manifestUrl).href,
      bandwidth: parseInt(attrs["BANDWIDTH"] || "0", 10) || null,
      codecs: attrs["CODECS"] || "",
      tracks: classifyHlsTracks(attrs["CODECS"], attrs["AUDIO"])
    });
  }
  return variants;
}

// ---------------------------------------------------------------------------
// Grab relay: content scripts cannot POST to the loopback server from https
// pages (CORS), so the overlay hands payloads here for dispatch.
// ---------------------------------------------------------------------------

async function handleGrabRelay(payload) {
  const headers = { ...(payload.headers || {}) };
  const cookieHeader = await getCookiesForUrl(payload.url);
  if (cookieHeader) headers["Cookie"] = cookieHeader;

  // Preserve Google Video CDN token validity: the sniffed URL is forwarded
  // exactly as the player requested it (any query re-serialization — including
  // dropping the player's chunk `range` param — risks a 403), and an explicit
  // full-range header makes the server serve the whole file.
  if (/googlevideo\.com/.test(payload.url)) {
    const entry = findStreamEntry(payload.url);
    if (entry && entry.sizeBytes > 0) {
      headers["Range"] = `bytes=0-${entry.sizeBytes}`;
    }
  }

  const resp = await fetch(`${CANDAK_LOOPBACK_URL}/api/grab`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "X-Candak-Source": "browser-extension"
    },
    body: JSON.stringify({
      url: payload.url,
      filename: payload.filename || "",
      headers
    })
  });
  if (!resp.ok) {
    throw new Error(`Grab endpoint returned ${resp.status}`);
  }
}

chrome.runtime.onMessage.addListener((msg, sender, sendResponse) => {
  if (msg.type === "candak:getStreams") {
    const perTab = streamRegistry.get(sender.tab ? sender.tab.id : -1);
    sendResponse({ streams: perTab ? [...perTab.values()] : [] });
    return false;
  }
  if (msg.type === "candak:parseHLS") {
    parseHlsMasterPlaylist(msg.url)
      .then(variants => sendResponse({ variants }))
      .catch(err => sendResponse({ error: String(err) }));
    return true; // Async response
  }
  if (msg.type === "candak:grab") {
    handleGrabRelay(msg.payload)
      .then(() => sendResponse({ ok: true }))
      .catch(err => sendResponse({ ok: false, error: String(err) }));
    return true; // Async response
  }
  return false;
});
