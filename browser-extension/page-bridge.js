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

  window.addEventListener("yt-navigate-finish", post);
  window.addEventListener("message", (e) => {
    if (e.source !== window || !e.data || e.data.source !== "candak-content") return;
    if (e.data.cmd === "collect") post();
    if (e.data.cmd === "quality" && e.data.quality) {
      try {
        const p = document.getElementById("movie_player");
        if (p && typeof p.setPlaybackQualityRange === "function") {
          if (!prevQuality && typeof p.getPlaybackQuality === "function") {
            const q = p.getPlaybackQuality();
            if (q && q !== "unknown") prevQuality = q;
          }
          p.setPlaybackQualityRange(e.data.quality, e.data.quality);
          // A full buffer can defer the switch indefinitely; a same-position
          // seek flushes it and forces an immediate fetch of the new quality
          // so the sniffer can intercept it.
          if (typeof p.seekTo === "function" && typeof p.getCurrentTime === "function") {
            p.seekTo(p.getCurrentTime(), true);
          }
        }
      } catch (err) { /* Player refused the switch */ }
    }
    if (e.data.cmd === "restore" && prevQuality) {
      try {
        const p = document.getElementById("movie_player");
        if (p && typeof p.setPlaybackQualityRange === "function") {
          p.setPlaybackQualityRange(prevQuality, prevQuality);
        }
      } catch (err) { /* Player refused the restore */ }
      prevQuality = null;
    }
  });
  post();
})();
