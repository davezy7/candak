// Tests for the pure helpers inside background.js (HLS parsing, media detection).
// Run: node --test browser-extension/
const { test } = require("node:test");
const assert = require("node:assert");
const fs = require("node:fs");
const path = require("node:path");
const vm = require("node:vm");

function loadBackground(manifestText) {
  const code = fs.readFileSync(path.join(__dirname, "background.js"), "utf8");
  const noopListener = { addListener() {} };
  const ctx = {
    chrome: {
      downloads: { onDeterminingFilename: noopListener },
      cookies: { getAll: async () => [] },
      storage: { local: { get: (defaults, cb) => cb(defaults) } },
      webRequest: { onHeadersReceived: noopListener },
      tabs: {
        onRemoved: noopListener,
        onUpdated: noopListener,
        sendMessage: async () => {}
      },
      runtime: { onMessage: noopListener }
    },
    fetch: async () => ({ ok: true, status: 200, text: async () => manifestText }),
    console,
    URL,
    Number,
    String,
    parseInt,
    isNaN,
    Map,
    Error,
    AbortController,
    setTimeout,
    clearTimeout
  };
  vm.createContext(ctx);
  vm.runInContext(code, ctx);
  return ctx;
}

test("parseHlsMasterPlaylist extracts labeled variants with track annotations", async () => {
  const manifest = [
    "#EXTM3U",
    "#EXT-X-STREAM-INF:BANDWIDTH=5000000,RESOLUTION=1920x1080,CODECS=\"avc1.640028,mp4a.40.2\"",
    "video-1080.m3u8",
    "#EXT-X-STREAM-INF:BANDWIDTH=2500000,RESOLUTION=1280x720,CODECS=\"avc1.64001f\",AUDIO=\"eng\"",
    "video-720.m3u8",
    "#EXT-X-STREAM-INF:BANDWIDTH=100000,CODECS=\"mp4a.40.2\"",
    "audio.m3u8"
  ].join("\n");
  const ctx = loadBackground(manifest);
  const variants = await ctx.parseHlsMasterPlaylist("https://cdn.example.com/master.m3u8");

  assert.equal(variants.length, 3);
  assert.equal(variants[0].label, "1080p");
  assert.equal(variants[0].tracks, "(Video + Audio)");
  assert.equal(variants[0].url, "https://cdn.example.com/video-1080.m3u8");
  assert.equal(variants[1].label, "720p");
  assert.equal(variants[1].tracks, "(Video only)");
  assert.equal(variants[2].label, null);
  assert.equal(variants[2].tracks, "(Audio only)");
});

test("parseHlsMasterPlaylist treats a media playlist as a single variant", async () => {
  const ctx = loadBackground("#EXTM3U\n#EXTINF:10,\nseg0.ts\n");
  const variants = await ctx.parseHlsMasterPlaylist("https://cdn.example.com/media.m3u8");
  assert.equal(variants.length, 1);
  assert.equal(variants[0].label, null);
  assert.equal(variants[0].url, "https://cdn.example.com/media.m3u8");
});

test("media detection helpers classify MIME, extensions, itag and sizes", () => {
  const ctx = loadBackground("");
  assert.equal(ctx.isMediaMime("video/mp4"), true);
  assert.equal(ctx.isMediaMime("audio/webm"), true);
  assert.equal(ctx.isMediaMime("application/vnd.apple.mpegurl"), true);
  assert.equal(ctx.isMediaMime("application/dash+xml"), true);
  assert.equal(ctx.isMediaMime("text/html"), false);
  assert.equal(ctx.isMediaUrl("https://x.com/a/b.MP4?token=1"), true);
  assert.equal(ctx.isMediaUrl("https://x.com/a/b.m3u8"), true);
  assert.equal(ctx.isMediaUrl("https://r4---sn.googlevideo.com/videoplayback?itag=137&id=o-A"), true);
  assert.equal(ctx.isMediaUrl("https://x.com/a/b.html"), false);
  assert.equal(ctx.extractItag("https://r4---sn.googlevideo.com/videoplayback?itag=137&id=o-A"), "137");
  assert.equal(ctx.extractItag("https://x.com/video.mp4"), null);
  assert.equal(
    ctx.extractSizeBytes([{ name: "Content-Range", value: "bytes 0-999/104857600" }]),
    104857600
  );
  assert.equal(
    ctx.extractSizeBytes([{ name: "Content-Length", value: "512" }]),
    512
  );
});

test("grab relay forwards the sniffed URL untouched and sets a full-range header", async () => {
  const ctx = loadBackground("");
  const chunkUrl = "https://rr3.googlevideo.com/videoplayback?itag=137&range=0-1048575&expire=999&sig=abc";
  ctx.registerStream(1, { url: chunkUrl, mimeType: "video/mp4", sizeBytes: 104857600, itag: "137" });
  let captured = null;
  ctx.fetch = async (url, opts) => {
    captured = { url, body: JSON.parse(opts.body) };
    return { ok: true };
  };
  await ctx.handleGrabRelay({ url: chunkUrl, filename: "v.mp4", headers: {} });
  assert.equal(captured.url, "http://127.0.0.1:41920/api/grab");
  assert.equal(captured.body.url, chunkUrl, "URL must be forwarded byte-for-byte (tampering risks 403)");
  assert.equal(captured.body.headers["Range"], "bytes=0-104857600");
});
