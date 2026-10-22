<script>
  import { formatBytes } from './formatters.js';

  export let chunks = [];
  export let totalSize = 0;
</script>

<div class="mt-2.5 pt-2.5 border-t border-slate-800/80">
  <div class="flex items-center justify-between text-xs text-slate-400 mb-1.5 font-mono">
    <span class="flex items-center gap-1.5">
      <span class="w-1.5 h-1.5 rounded-full bg-cyan-400"></span>
      Chunk Offsets ({chunks.length} {chunks.length === 1 ? 'part' : 'concurrent chunks'})
    </span>
    <span class="text-slate-500">Tracked in SQLite</span>
  </div>

  <div class="flex gap-1 h-3 rounded bg-slate-900/90 p-0.5 border border-slate-800 overflow-hidden">
    {#each chunks as chunk, i}
      {@const chunkTotal = Math.max(1, chunk.endOffset - chunk.startOffset + 1)}
      {@const chunkProgress = Math.min(100, Math.max(0, ((chunk.currentOffset - chunk.startOffset) / chunkTotal) * 100))}
      {@const chunkWidth = totalSize > 0 ? (chunkTotal / totalSize) * 100 : 100 / (chunks.length || 1)}
      
      <div 
        class="relative h-full bg-slate-800/80 rounded-sm overflow-hidden group transition-all"
        style="width: {chunkWidth}%;"
        title="Part {i + 1}: {chunkProgress.toFixed(1)}% ({formatBytes(chunk.currentOffset - chunk.startOffset)} / {formatBytes(chunkTotal)}) | Offset: {chunk.currentOffset}"
      >
        <div 
          class="h-full transition-all duration-300 {chunk.status === 'completed' ? 'bg-emerald-500' : 'bg-gradient-to-r from-cyan-500 to-blue-500'}"
          style="width: {chunkProgress}%;"
        ></div>
      </div>
    {/each}
  </div>

  <div class="flex flex-wrap gap-2 mt-2 text-[11px] font-mono text-slate-400">
    {#each chunks as chunk, i}
      {@const chunkTotal = Math.max(1, chunk.endOffset - chunk.startOffset + 1)}
      {@const chunkProgress = Math.min(100, Math.max(0, ((chunk.currentOffset - chunk.startOffset) / chunkTotal) * 100))}
      <div class="bg-slate-900/60 border border-slate-800 px-1.5 py-0.5 rounded text-[10px] flex items-center gap-1">
        <span class="text-slate-500">#{i + 1}</span>
        <span class="text-slate-300 font-semibold">{chunkProgress.toFixed(0)}%</span>
        <span class="text-slate-500">({formatBytes(chunk.currentOffset)})</span>
      </div>
    {/each}
  </div>
</div>
