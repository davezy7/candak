// Candak Download Manager - YouTube page-world bridge (content script, world: MAIN).
// The isolated content-script world cannot see page globals (ytInitialPlayerResponse,
// movie_player), and YouTube's Trusted Types policy blocks inline-script injection,
// so this file runs directly in the page world. It relays slim format lists to the
// overlay via window.postMessage and forwards quality-switch commands back.
(() => {
  if (window.__candakPageBridge) return;
  window.__candakPageBridge = true;

  const slim = (pr) => {
    if (!pr || !pr.streamingData) return null;
    const all = [].concat(pr.streamingData.formats || [], pr.streamingData.adaptiveFormats || []);
    if (!all.length) return null;
    return all.map((f) => ({
      itag: f.itag,
      mimeType: f.mimeType || "",
      qualityLabel: f.qualityLabel || "",
      height: f.height || 0,
      hasAudio: !!(f.audioQuality || f.audioTrack),
      contentLength: f.contentLength || 0,
      url: f.url || null
    }));
  };

  const currentFormats = () => {
    let pr = window.ytInitialPlayerResponse;
    try {
      const p = document.getElementById("movie_player");
      if (p && typeof p.getResponse === "function") {
        const r = p.getResponse(); // Fresh across SPA navigations
        if (r && r.streamingData) pr = r;
      }
    } catch (err) { /* Player API unavailable */ }
    return slim(pr);
  };

  const post = () => {
    const formats = currentFormats();
    if (formats) window.postMessage({ source: "candak-page", formats }, window.location.origin);
  };

  let prevQuality = null; // Quality the player was on before a fetch-and-download switch

  function triggerQualityMenuSwitch(targetText) {
    if (!targetText) return;
    const gear = document.querySelector(".ytp-settings-button");
    if (!gear) return;

    // Check if settings menu is already open
    const isMenuOpen = () => {
      const p = document.querySelector(".ytp-popup.ytp-settings-menu");
      return p && p.style.display !== "none" && p.offsetWidth > 0;
    };
    if (!isMenuOpen()) gear.click();

    let tries = 0;
    const step1 = setInterval(() => {
      tries++;
      const items = Array.from(document.querySelectorAll(".ytp-popup.ytp-settings-menu .ytp-menuitem"));
      const qualityItem = items.find(it => /quality|kualitas|qualit|calidad|qualität|画質|화질|качество/i.test(it.textContent));
      if (qualityItem) {
        clearInterval(step1);
        qualityItem.click();

        let subTries = 0;
        const cleanTarget = targetText.toLowerCase().replace(/[^0-9p]/g, "");
        const heightOnly = cleanTarget.split("p")[0] + "p"; // e.g. "2160p", "1080p"

        const step2 = setInterval(() => {
          subTries++;
          const opts = Array.from(document.querySelectorAll(".ytp-popup.ytp-settings-menu .ytp-panel-menu .ytp-menuitem"));
          const target = opts.find(o => {
            const t = o.textContent.toLowerCase();
            return t.includes(cleanTarget) || t.includes(heightOnly);
          });
          if (target) {
            clearInterval(step2);
            target.click();
          } else if (subTries > 12) {
            clearInterval(step2);
            if (isMenuOpen()) gear.click();
          }
        }, 80);
      } else if (tries > 12) {
        clearInterval(step1);
        if (isMenuOpen()) gear.click();
      }
    }, 80);
  }

  window.addEventListener("yt-navigate-finish", post);
  window.addEventListener("message", (e) => {
    if (e.source !== window || !e.data || e.data.source !== "candak-content") return;
    if (e.data.cmd === "collect") post();
    if (e.data.cmd === "quality" && e.data.quality) {
      try {
        const p = document.getElementById("movie_player");
        if (p) {
          if (!prevQuality && typeof p.getPlaybackQuality === "function") {
            const q = p.getPlaybackQuality();
            if (q && q !== "unknown") prevQuality = q;
          }
          if (typeof p.setPlaybackQualityRange === "function") {
            p.setPlaybackQualityRange(e.data.quality, e.data.quality);
          }
          if (typeof p.setPlaybackQuality === "function") {
            p.setPlaybackQuality(e.data.quality);
          }
          // A full buffer can defer the switch indefinitely; a same-position
          // seek flushes it and forces an immediate fetch of the new quality
          // so the sniffer can intercept it.
          if (typeof p.seekTo === "function" && typeof p.getCurrentTime === "function") {
            p.seekTo(p.getCurrentTime(), true);
          }
          if (typeof p.playVideo === "function" && typeof p.getPlayerState === "function" && p.getPlayerState() === 2) {
            p.playVideo();
          }
        }
      } catch (err) { /* Player refused the switch */ }

      // UI automation fallback: modern YouTube often ignores setPlaybackQuality
      // unless chosen through the player's quality menu
      try {
        triggerQualityMenuSwitch(e.data.label || e.data.quality);
      } catch (_) {}
    }
    if (e.data.cmd === "restore" && prevQuality) {
      try {
        const p = document.getElementById("movie_player");
        if (p) {
          if (typeof p.setPlaybackQualityRange === "function") {
            p.setPlaybackQualityRange(prevQuality, prevQuality);
          }
          if (typeof p.setPlaybackQuality === "function") {
            p.setPlaybackQuality(prevQuality);
          }
        }
      } catch (err) { /* Player refused the restore */ }
      prevQuality = null;
    }
  });
  post();
})();
