<script>
  import { 
    Pause, 
    Play, 
    Trash2, 
    CheckSquare, 
    Square, 
    X 
  } from 'lucide-svelte';

  export let selectedCount = 0;
  export let totalVisibleCount = 0;
  export let onToggleSelectAll = () => {};
  export let onPauseSelected = () => {};
  export let onResumeSelected = () => {};
  export let onDeleteSelected = () => {};
  export let onClear = () => {};

  $: isAllSelected = totalVisibleCount > 0 && selectedCount >= totalVisibleCount;
</script>

<div class="fixed bottom-20 left-1/2 -translate-x-1/2 z-40 animate-in fade-in slide-in-from-bottom-3 duration-200 pointer-events-auto">
  <div class="flex items-center gap-2 bg-[var(--color-surface)]/95 backdrop-blur-md border border-[var(--color-border)] shadow-xl rounded-full py-1.5 px-3 text-xs text-[var(--color-text)]">
    <!-- Selection Counter -->
    <div class="flex items-center gap-2 pl-2 pr-3 py-1 font-semibold border-r border-[var(--color-border)]">
      <span class="w-2 h-2 rounded-full bg-[var(--color-accent)] animate-pulse"></span>
      <span>{selectedCount} {selectedCount === 1 ? 'item' : 'items'} selected</span>
    </div>

    <!-- Select All / Deselect All Button -->
    <button
      type="button"
      on:click={onToggleSelectAll}
      class="flex items-center gap-1.5 px-2.5 py-1.5 rounded-full hover:bg-[var(--color-bg)] text-[var(--color-secondary)] hover:text-[var(--color-text)] transition font-medium"
      title={isAllSelected ? "Deselect All" : "Select All"}
    >
      {#if isAllSelected}
        <CheckSquare size={14} class="text-[var(--color-accent)]" />
        <span>Deselect All</span>
      {:else}
        <Square size={14} />
        <span>Select All</span>
      {/if}
    </button>

    <!-- Pause Selected -->
    <button
      type="button"
      on:click={onPauseSelected}
      class="flex items-center gap-1.5 px-2.5 py-1.5 rounded-full hover:bg-[var(--color-bg)] text-[var(--color-secondary)] hover:text-[var(--color-text)] transition font-medium"
      title="Pause Selected Downloads"
    >
      <Pause size={14} />
      <span>Pause</span>
    </button>

    <!-- Resume Selected -->
    <button
      type="button"
      on:click={onResumeSelected}
      class="flex items-center gap-1.5 px-2.5 py-1.5 rounded-full hover:bg-[var(--color-bg)] text-[var(--color-secondary)] hover:text-[var(--color-text)] transition font-medium"
      title="Resume Selected Downloads"
    >
      <Play size={14} />
      <span>Resume</span>
    </button>

    <!-- Delete Selected -->
    <button
      type="button"
      on:click={onDeleteSelected}
      class="flex items-center gap-1.5 px-2.5 py-1.5 rounded-full hover:bg-rose-500/10 text-rose-500 hover:text-rose-600 transition font-medium"
      title="Delete Selected Downloads"
    >
      <Trash2 size={14} />
      <span>Delete</span>
    </button>

    <!-- Clear / Dismiss -->
    <button
      type="button"
      on:click={onClear}
      class="p-1.5 rounded-full hover:bg-[var(--color-bg)] text-[var(--color-secondary)] hover:text-[var(--color-text)] transition ml-1"
      title="Clear selection (Esc)"
      aria-label="Clear selection"
    >
      <X size={14} />
    </button>
  </div>
</div>
