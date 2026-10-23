<script>
  import { formatBytes } from './formatters.js';

  export let chunks = [];
  export let totalSize = 0;
</script>

<div class="mt-3 pt-3 border-t border-[#E2E8F0]">
  <div class="flex items-center justify-between text-xs text-[#64748B] mb-1.5 font-medium">
    <span class="flex items-center gap-1.5">
      <span class="w-1.5 h-1.5 rounded-full bg-[#0D9488]"></span>
      Chunk Offsets ({chunks.length} {chunks.length === 1 ? 'part' : 'threads'})
    </span>
    <span class="text-[11px] text-[#64748B]/80 font-mono">SQLite Resumable</span>
  </div>

  <div class="flex gap-1 h-2.5 rounded-full bg-[#F1F5F9] p-0.5 border border-[#E2E8F0] overflow-hidden">
    {#each chunks as chunk, i}
      {@const chunkTotal = Math.max(1, chunk.endOffset - chunk.startOffset + 1)}
      {@const chunkProgress = Math.min(100, Math.max(0, ((chunk.currentOffset - chunk.startOffset) / chunkTotal) * 100))}
      {@const chunkWidth = totalSize > 0 ? (chunkTotal / totalSize) * 100 : 100 / (chunks.length || 1)}
      
      <div 
        class="relative h-full bg-[#E2E8F0] rounded-full overflow-hidden group transition-all"
        style="width: {chunkWidth}%;"
        title="Part {i + 1}: {chunkProgress.toFixed(1)}% ({formatBytes(chunk.currentOffset - chunk.startOffset)} / {formatBytes(chunkTotal)}) | Offset: {chunk.currentOffset}"
      >
        <div 
          class="h-full transition-all duration-300 {chunk.status === 'completed' ? 'bg-[#0D9488]' : 'bg-[#0D9488]'}"
          style="width: {chunkProgress}%;"
        ></div>
      </div>
    {/each}
  </div>

  <div class="flex flex-wrap gap-1.5 mt-2 text-[11px] font-mono text-[#64748B]">
    {#each chunks as chunk, i}
      {@const chunkTotal = Math.max(1, chunk.endOffset - chunk.startOffset + 1)}
      {@const chunkProgress = Math.min(100, Math.max(0, ((chunk.currentOffset - chunk.startOffset) / chunkTotal) * 100))}
      <div class="bg-[#F8FAFC] border border-[#E2E8F0] px-2 py-0.5 rounded-md text-[10px] flex items-center gap-1.5">
        <span class="text-[#64748B]">#{i + 1}</span>
        <span class="text-[#1F2937] font-semibold">{chunkProgress.toFixed(0)}%</span>
        <span class="text-[#64748B]/80">({formatBytes(chunk.currentOffset)})</span>
      </div>
    {/each}
  </div>
</div>
