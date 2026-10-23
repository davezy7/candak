const CANDAK_LOOPBACK_URL = "http://127.0.0.1:41920";

const statusDot = document.getElementById("statusDot");
const statusText = document.getElementById("statusText");
const toggle = document.getElementById("interceptToggle");

// Load toggle setting
chrome.storage.local.get({ interceptEnabled: true }, (res) => {
  toggle.checked = !!res.interceptEnabled;
});

toggle.addEventListener("change", () => {
  chrome.storage.local.set({ interceptEnabled: toggle.checked });
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
