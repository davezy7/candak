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
