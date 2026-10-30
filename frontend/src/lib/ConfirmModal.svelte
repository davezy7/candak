<script>
  import { X, AlertTriangle } from 'lucide-svelte';
  import { t } from './i18n/index.js';

  export let isOpen = false;
  export let title = '';
  export let message = '';
  export let submessage = '';
  export let confirmText = '';
  export let cancelText = '';
  export let danger = true;
  export let showCheckbox = false;
  export let checkboxLabel = '';
  export let checkboxChecked = false;
  export let onConfirm = () => {};
  export let onCancel = () => {};

  $: resolvedTitle = title || $t('modals.confirm.title');
  $: resolvedMessage = message || $t('modals.confirm.message');
  $: resolvedConfirmText = confirmText || (danger ? $t('common.delete') : $t('modals.confirm.confirm'));
  $: resolvedCancelText = cancelText || $t('common.cancel');
  $: resolvedCheckboxLabel = checkboxLabel || $t('modals.bulk_delete.delete_files_checkbox');

  function handleKeydown(e) {
    if (e.key === 'Escape' && isOpen) {
      onCancel();
    }
  }
</script>

<svelte:window on:keydown={handleKeydown} />

{#if isOpen}
  <div
    class="fixed inset-0 bg-black/50 backdrop-blur-xs z-50 flex items-center justify-center p-4 animate-in fade-in duration-100"
    role="presentation"
    on:click|self={onCancel}
  >
    <div
      class="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-3xl w-full max-w-md shadow-2xl overflow-hidden animate-in fade-in zoom-in-95 duration-150 p-6 flex flex-col gap-5"
      role="dialog"
      aria-modal="true"
    >
      <div class="flex items-start gap-3.5">
        <div class="w-10 h-10 rounded-2xl {danger ? 'bg-rose-500/10 text-rose-500' : 'bg-[var(--color-accent-subtle)] text-[var(--color-accent)]'} flex items-center justify-center shrink-0">
          <AlertTriangle size={20} strokeWidth={2.2} />
        </div>
        <div class="flex-1 min-w-0">
          <h3 class="font-bold text-[var(--color-text)] text-base">{resolvedTitle}</h3>
          <p class="text-xs text-[var(--color-secondary)] mt-1 break-words">{resolvedMessage}</p>
          {#if submessage}
            <div class="text-[11px] font-mono text-[var(--color-secondary)]/90 mt-2 truncate bg-[var(--color-bg)] px-2.5 py-1.5 rounded-xl border border-[var(--color-border)]">
              {submessage}
            </div>
          {/if}
        </div>
        <button
          on:click={onCancel}
          class="text-[var(--color-secondary)] hover:text-[var(--color-text)] p-1 rounded-lg hover:bg-[var(--color-border)]/50 transition shrink-0"
          aria-label={$t('common.close')}
        >
          <X size={16} />
        </button>
      </div>

      {#if showCheckbox}
        <label class="flex items-center gap-2.5 px-3 py-2.5 bg-[var(--color-bg)] rounded-xl border border-[var(--color-border)] text-xs text-[var(--color-text)] cursor-pointer select-none hover:border-[var(--color-border)] transition">
          <input
            type="checkbox"
            bind:checked={checkboxChecked}
            class="rounded border-[var(--color-border)] text-rose-500 focus:ring-rose-500 w-4 h-4 accent-rose-500"
          />
          <span class="font-medium">{resolvedCheckboxLabel}</span>
        </label>
      {/if}

      <div class="flex items-center justify-end gap-2.5 pt-2 border-t border-[var(--color-border)]">
        <button
          type="button"
          on:click={onCancel}
          class="px-4 py-2 text-xs font-semibold rounded-xl border border-[var(--color-border)] hover:bg-[var(--color-bg)] text-[var(--color-secondary)] hover:text-[var(--color-text)] transition"
        >
          {resolvedCancelText}
        </button>
        <button
          type="button"
          on:click={() => onConfirm({ checked: checkboxChecked })}
          class="px-4 py-2 text-xs font-semibold rounded-xl transition shadow-xs flex items-center gap-1.5 {danger ? 'bg-rose-500 hover:bg-rose-600 text-white' : 'bg-[var(--color-accent)] hover:opacity-90 text-white'}"
        >
          {resolvedConfirmText}
        </button>
      </div>
    </div>
  </div>
{/if}
