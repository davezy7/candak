const CANDAK_LOOPBACK_URL = "http://127.0.0.1:41920";

const statusDot = document.getElementById("statusDot");
const statusText = document.getElementById("statusText");
const toggle = document.getElementById("interceptToggle");
const overlayToggle = document.getElementById("overlayToggle");
const siteBtn = document.getElementById("siteExclusionBtn");

// Load toggle setting
chrome.storage.local.get({ interceptEnabled: true }, (res) => {
  toggle.checked = !!res.interceptEnabled;
});

toggle.addEventListener("change", () => {
  chrome.storage.local.set({ interceptEnabled: toggle.checked });
});

// Video Grabber Overlay settings
chrome.storage.local.get({ overlayEnabled: true, excludedSites: [] }, (res) => {
  overlayToggle.checked = !!res.overlayEnabled;

  // Per-site exclusion toggle for the active tab's hostname
  chrome.tabs.query({ active: true, currentWindow: true }, (tabs) => {
    let hostname = "";
    try {
      hostname = tabs[0] && tabs[0].url ? new URL(tabs[0].url).hostname : "";
    } catch (err) {
      hostname = "";
    }
    if (!hostname) {
      siteBtn.style.display = "none";
      return;
    }
    const updateBtn = (excluded) => {
      siteBtn.textContent = excluded
        ? `Enable on this site (${hostname})`
        : `Disable on this site (${hostname})`;
    };
    updateBtn(res.excludedSites.includes(hostname));
    siteBtn.addEventListener("click", () => {
      chrome.storage.local.get({ excludedSites: [] }, (r2) => {
        const excluded = r2.excludedSites.includes(hostname);
        const list = excluded
          ? r2.excludedSites.filter(h => h !== hostname)
          : [...r2.excludedSites, hostname];
        chrome.storage.local.set({ excludedSites: list }, () => updateBtn(!excluded));
      });
    });
  });
});

overlayToggle.addEventListener("change", () => {
  chrome.storage.local.set({ overlayEnabled: overlayToggle.checked });
});

// Check health
async function checkStatus() {
  try {
    const res = await fetch(`${CANDAK_LOOPBACK_URL}/health`);
    if (res.ok) {
      statusDot.classList.add("connected");
      statusText.textContent = "Connected";
      statusText.style.color = "#10b981";
      return;
    }
  } catch (_) {}
  statusDot.classList.remove("connected");
  statusText.textContent = "Offline";
  statusText.style.color = "#ef4444";
}

checkStatus();
