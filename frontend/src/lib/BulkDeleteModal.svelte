<script>
  import ConfirmModal from './ConfirmModal.svelte';
  import { t } from './i18n/index.js';

  export let isOpen = false;
  export let count = 0;
  export let onConfirm = () => {};
  export let onCancel = () => {};

  let deleteFiles = false;

  $: if (!isOpen) {
    deleteFiles = false;
  }

  function handleConfirm(detail) {
    onConfirm(detail ? detail.checked : deleteFiles);
  }
</script>

<ConfirmModal
  {isOpen}
  title={$t('modals.bulk_delete.title')}
  message={$t('modals.bulk_delete.message', { count })}
  confirmText={$t('modals.bulk_delete.confirm', { count })}
  cancelText={$t('common.cancel')}
  danger={true}
  showCheckbox={true}
  checkboxLabel={$t('modals.bulk_delete.delete_files_checkbox')}
  bind:checkboxChecked={deleteFiles}
  onConfirm={handleConfirm}
  {onCancel}
/>
