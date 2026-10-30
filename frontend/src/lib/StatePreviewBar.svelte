<script>
  import { DOWNLOAD_STATUS } from './constants.js';
  import { 
    Activity, 
    Play, 
    Pause, 
    Check, 
    AlertCircle, 
    Clock, 
    XCircle, 
    RotateCcw, 
    Sliders, 
    ChevronUp, 
    ChevronDown, 
    X,
    Eye
  } from 'lucide-svelte';

  export let downloads = [];
  export let selectedIds = new Set();
  export let onApplyFixture = (fixtureList) => {};
  export let onRestoreReal = () => {};
  export let onMutateSelected = (status, extraProps) => {};

  let isExpanded = false;
  let isSimulating = false;
  let simTimer = null;

  // Comprehensive test fixture covering all possible download states
  const ALL_STATES_FIXTURE = [
    {
      id: 'fixture-queued',
      filename: 'Fedora-Workstation-40-x86_64.iso',
      url: 'https://download.fedoraproject.org/pub/fedora/linux/releases/40/Workstation.iso',
      savePath: '/Users/diksha/Downloads/Fedora-Workstation-40.iso',
      status: DOWNLOAD_STATUS.QUEUED,
      totalSize: 2200000000,
      downloadedSize: 0,
      speed: 0,
      eta: 0,
      createdAt: new Date().toISOString()
    },
    {
      id: 'fixture-connecting',
      filename: 'Debian-12.5.0-netinst.iso',
      url: 'https://cdimage.debian.org/debian-cd/current/amd64/iso-cd/debian-12.5.0.iso',
      savePath: '/Users/diksha/Downloads/debian-12.5.0.iso',
      status: DOWNLOAD_STATUS.DOWNLOADING,
      totalSize: 650000000,
      downloadedSize: 0,
      speed: 0,
      eta: 0,
      createdAt: new Date().toISOString()
    },
    {
      id: 'fixture-active-slow',
      filename: 'ArchLinux-2024.10.01-x86_64.iso',
      url: 'https://geo.mirror.pkgbuild.com/iso/latest/archlinux-x86_64.iso',
      savePath: '/Users/diksha/Downloads/archlinux-x86_64.iso',
      status: DOWNLOAD_STATUS.DOWNLOADING,
      totalSize: 1100000000,
      downloadedSize: 154000000,
      speed: 840000,
      eta: 1126,
      createdAt: new Date(Date.now() - 600000).toISOString()
    },
    {
      id: 'fixture-active-fast',
      filename: 'Ubuntu-24.04-desktop-amd64.iso',
      url: 'https://releases.ubuntu.com/24.04/ubuntu-24.04-desktop-amd64.iso',
      savePath: '/Users/diksha/Downloads/ubuntu-24.04-desktop-amd64.iso',
      status: DOWNLOAD_STATUS.DOWNLOADING,
      totalSize: 5800000000,
      downloadedSize: 4930000000,
      speed: 24500000,
      eta: 35,
      createdAt: new Date(Date.now() - 1800000).toISOString()
    },
    {
      id: 'fixture-paused-early',
      filename: 'FreeCAD-0.21.2-macOS-arm64.dmg',
      url: 'https://github.com/FreeCAD/FreeCAD/releases/download/0.21.2/FreeCAD.dmg',
      savePath: '/Users/diksha/Downloads/FreeCAD-0.21.2.dmg',
      status: DOWNLOAD_STATUS.PAUSED,
      totalSize: 320000000,
      downloadedSize: 64000000,
      speed: 0,
      eta: 0,
      createdAt: new Date(Date.now() - 3600000).toISOString()
    },
    {
      id: 'fixture-paused-mid',
      filename: 'VSCode-darwin-universal.zip',
      url: 'https://update.code.visualstudio.com/latest/darwin-universal/stable',
      savePath: '/Users/diksha/Downloads/VSCode-darwin-universal.zip',
      status: DOWNLOAD_STATUS.PAUSED,
      totalSize: 180000000,
      downloadedSize: 117000000,
      speed: 0,
      eta: 0,
      createdAt: new Date(Date.now() - 7200000).toISOString()
    },
    {
      id: 'fixture-completed-video',
      filename: 'Cosmos Laundromat – First Cycle (2015, 4K).mp4',
      url: 'https://download.blender.org/demo/movies/CosmosLaundromat.mp4',
      savePath: '/Users/diksha/Downloads/Cosmos Laundromat.mp4',
      status: DOWNLOAD_STATUS.COMPLETED,
      totalSize: 1350000000,
      downloadedSize: 1350000000,
      speed: 0,
      eta: 0,
      createdAt: new Date(Date.now() - 86400000).toISOString()
    },
    {
      id: 'fixture-completed-archive',
      filename: 'PostgreSQL-16.2-Source.tar.gz',
      url: 'https://ftp.postgresql.org/pub/source/v16.2/postgresql-16.2.tar.gz',
      savePath: '/Users/diksha/Downloads/postgresql-16.2.tar.gz',
      status: DOWNLOAD_STATUS.COMPLETED,
      totalSize: 34500000,
      downloadedSize: 34500000,
      speed: 0,
      eta: 0,
      createdAt: new Date(Date.now() - 86400000 * 3).toISOString()
    },
    {
      id: 'fixture-error-403',
      filename: 'NVIDIA-Driver-550.54.14-x86_64.run',
      url: 'https://us.download.nvidia.com/XFree86/Linux-x86_64/550.54.14/NVIDIA.run',
      savePath: '/Users/diksha/Downloads/NVIDIA-Driver.run',
      status: DOWNLOAD_STATUS.ERROR,
      errorMessage: 'HTTP 403 Forbidden: CDN token expired',
      totalSize: 420000000,
      downloadedSize: 12000000,
      speed: 0,
      eta: 0,
      createdAt: new Date(Date.now() - 86400000 * 5).toISOString()
    },
    {
      id: 'fixture-error-timeout',
      filename: 'Linux-Kernel-6.8.0.tar.xz',
      url: 'https://cdn.kernel.org/pub/linux/kernel/v6.x/linux-6.8.tar.xz',
      savePath: '/Users/diksha/Downloads/linux-6.8.tar.xz',
      status: DOWNLOAD_STATUS.ERROR,
      errorMessage: 'Connection timed out after 30s',
      totalSize: 145000000,
      downloadedSize: 78000000,
      speed: 0,
      eta: 0,
      createdAt: new Date(Date.now() - 86400000 * 8).toISOString()
    },
    {
      id: 'fixture-cancelled',
      filename: 'HugeDataDump_backup_2026.sql.gz',
      url: 'https://backup.corp.internal/dumps/2026.sql.gz',
      savePath: '/Users/diksha/Downloads/HugeDataDump.sql.gz',
      status: DOWNLOAD_STATUS.CANCELLED,
      totalSize: 15400000000,
      downloadedSize: 320000000,
      speed: 0,
      eta: 0,
      createdAt: new Date(Date.now() - 86400000 * 12).toISOString()
    }
  ];

  function loadAllStates() {
    onApplyFixture([...ALL_STATES_FIXTURE]);
  }

  function toggleSimulation() {
    isSimulating = !isSimulating;
    if (isSimulating) {
      simTimer = setInterval(() => {
        downloads = downloads.map(d => {
          if (d.status === DOWNLOAD_STATUS.DOWNLOADING && d.totalSize > 0) {
            const increment = (d.speed > 0 ? d.speed : 5000000) * 0.4;
            const newDownloaded = Math.min(d.totalSize, d.downloadedSize + increment);
            const isFinished = newDownloaded >= d.totalSize;
            return {
              ...d,
              downloadedSize: newDownloaded,
              status: isFinished ? DOWNLOAD_STATUS.COMPLETED : DOWNLOAD_STATUS.DOWNLOADING,
              speed: isFinished ? 0 : (d.speed || 4500000),
              eta: isFinished ? 0 : Math.max(1, Math.round((d.totalSize - newDownloaded) / (d.speed || 4500000)))
            };
          }
          return d;
        });
      }, 400);
    } else {
      if (simTimer) clearInterval(simTimer);
    }
  }

  function setStatus(status, extra = {}) {
    onMutateSelected(status, extra);
  }
</script>

<!-- Floating State Preview Tool (collapsible dev utility) -->
<aside 
  aria-label="Temporary State Previewer"
  class="fixed bottom-12 right-6 z-50 flex flex-col items-end select-none font-sans"
>
  {#if isExpanded}
    <div class="mb-2 p-3.5 rounded-2xl bg-zinc-950/95 text-zinc-100 border border-zinc-800 shadow-2xl backdrop-blur-xl w-80 ring-1 ring-white/10 flex flex-col gap-3 text-xs">
      <!-- Title header -->
      <div class="flex items-center justify-between border-b border-zinc-800 pb-2">
        <div class="flex items-center gap-1.5 font-bold text-emerald-400">
          <Eye size={14} />
          <span>Item State Previewer</span>
        </div>
        <button 
          type="button" 
          on:click={() => isExpanded = false}
          class="text-zinc-400 hover:text-white p-0.5 rounded transition cursor-pointer"
          title="Minimize"
        >
          <X size={14} />
        </button>
      </div>

      <!-- Quick Fixture Buttons -->
      <div class="flex flex-col gap-1.5">
        <span class="text-[10px] font-mono uppercase tracking-wider text-zinc-400 font-bold">1. Quick State Fixtures</span>
        <div class="grid grid-cols-2 gap-1.5">
          <button
            type="button"
            on:click={loadAllStates}
            class="px-2.5 py-1.5 rounded-lg bg-emerald-500/20 hover:bg-emerald-500/30 text-emerald-300 border border-emerald-500/40 font-semibold transition cursor-pointer text-left flex items-center justify-between"
          >
            <span>All States (11 items)</span>
            <span class="text-[10px] font-mono opacity-80">Load</span>
          </button>

          <button
            type="button"
            on:click={onRestoreReal}
            class="px-2.5 py-1.5 rounded-lg bg-zinc-800/80 hover:bg-zinc-800 text-zinc-300 border border-zinc-700 font-semibold transition cursor-pointer text-left flex items-center justify-between"
          >
            <span>Reset to Live</span>
            <RotateCcw size={12} class="opacity-70" />
          </button>
        </div>
      </div>

      <!-- Live Simulation Toggle -->
      <div class="flex items-center justify-between bg-zinc-900/80 p-2 rounded-xl border border-zinc-800">
        <div class="flex items-center gap-1.5">
          <Activity size={13} class={isSimulating ? "text-emerald-400 animate-pulse" : "text-zinc-500"} />
          <span class="font-medium text-[11px]">Simulate Byte Progress</span>
        </div>
        <button
          type="button"
          on:click={toggleSimulation}
          class="px-2.5 py-1 rounded-md text-[10px] font-bold font-mono transition cursor-pointer {
            isSimulating
              ? 'bg-amber-500/20 text-amber-300 border border-amber-500/40'
              : 'bg-zinc-800 text-zinc-300 hover:bg-zinc-700'
          }"
        >
          {isSimulating ? 'PAUSE TICK' : 'START TICK'}
        </button>
      </div>

      <!-- Selected Item State Mutator -->
      <div class="flex flex-col gap-1.5 border-t border-zinc-800 pt-2">
        <div class="flex items-center justify-between">
          <span class="text-[10px] font-mono uppercase tracking-wider text-zinc-400 font-bold">2. Mutate Selected Item</span>
          <span class="text-[10px] text-zinc-500">
            {selectedIds.size === 1 ? '1 item selected' : selectedIds.size > 1 ? `${selectedIds.size} items` : 'Click an item first'}
          </span>
        </div>

        <div class="grid grid-cols-3 gap-1">
          <button
            type="button"
            disabled={selectedIds.size === 0}
            on:click={() => setStatus('downloading', { speed: 8500000, eta: 42 })}
            class="px-1.5 py-1 rounded bg-zinc-900 hover:bg-zinc-800 text-emerald-400 border border-zinc-800 font-medium text-[10px] disabled:opacity-40 transition cursor-pointer text-center"
          >
            Downloading
          </button>

          <button
            type="button"
            disabled={selectedIds.size === 0}
            on:click={() => setStatus('downloading', { speed: 0, eta: 0, downloadedSize: 0 })}
            class="px-1.5 py-1 rounded bg-zinc-900 hover:bg-zinc-800 text-teal-300 border border-zinc-800 font-medium text-[10px] disabled:opacity-40 transition cursor-pointer text-center"
          >
            Connecting
          </button>

          <button
            type="button"
            disabled={selectedIds.size === 0}
            on:click={() => setStatus('paused', { speed: 0, eta: 0 })}
            class="px-1.5 py-1 rounded bg-zinc-900 hover:bg-zinc-800 text-amber-400 border border-zinc-800 font-medium text-[10px] disabled:opacity-40 transition cursor-pointer text-center"
          >
            Paused
          </button>

          <button
            type="button"
            disabled={selectedIds.size === 0}
            on:click={() => setStatus('completed', { speed: 0, eta: 0 })}
            class="px-1.5 py-1 rounded bg-zinc-900 hover:bg-zinc-800 text-emerald-400 border border-zinc-800 font-medium text-[10px] disabled:opacity-40 transition cursor-pointer text-center"
          >
            Completed
          </button>

          <button
            type="button"
            disabled={selectedIds.size === 0}
            on:click={() => setStatus('queued', { speed: 0, eta: 0, downloadedSize: 0 })}
            class="px-1.5 py-1 rounded bg-zinc-900 hover:bg-zinc-800 text-sky-400 border border-zinc-800 font-medium text-[10px] disabled:opacity-40 transition cursor-pointer text-center"
          >
            Queued
          </button>

          <button
            type="button"
            disabled={selectedIds.size === 0}
            on:click={() => setStatus('error', { errorMessage: 'HTTP 403 Forbidden: Server rejected range request' })}
            class="px-1.5 py-1 rounded bg-zinc-900 hover:bg-zinc-800 text-rose-400 border border-zinc-800 font-medium text-[10px] disabled:opacity-40 transition cursor-pointer text-center"
          >
            Error
          </button>

          <button
            type="button"
            disabled={selectedIds.size === 0}
            on:click={() => setStatus('cancelled', { speed: 0, eta: 0 })}
            class="col-span-3 px-1.5 py-1 rounded bg-zinc-900 hover:bg-zinc-800 text-zinc-400 border border-zinc-800 font-medium text-[10px] disabled:opacity-40 transition cursor-pointer text-center"
          >
            Cancelled
          </button>
        </div>
      </div>
    </div>
  {/if}

  <!-- Minimized Trigger Pill -->
  <button
    type="button"
    on:click={() => isExpanded = !isExpanded}
    class="flex items-center gap-2 px-3 py-1.5 rounded-full bg-zinc-950/95 text-zinc-200 border border-zinc-800 shadow-2xl backdrop-blur-md ring-1 ring-white/10 hover:border-emerald-500/50 hover:text-white transition active:scale-95 cursor-pointer text-xs font-medium"
    title="Toggle Temporary State Previewer"
  >
    <Eye size={13} class="text-emerald-400" />
    <span>State Previewer</span>
    {#if isExpanded}
      <ChevronDown size={13} class="text-zinc-500" />
    {:else}
      <ChevronUp size={13} class="text-zinc-500" />
    {/if}
  </button>
</aside>
