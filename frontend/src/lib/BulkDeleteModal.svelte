<script>
  import ConfirmModal from './ConfirmModal.svelte';

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
  title="Delete Selected Downloads"
  message="Delete {count} selected {count === 1 ? 'download' : 'downloads'}?"
  confirmText="Delete {count} {count === 1 ? 'Item' : 'Items'}"
  cancelText="Cancel"
  danger={true}
  showCheckbox={true}
  checkboxLabel="Also delete downloaded files from disk"
  bind:checkboxChecked={deleteFiles}
  onConfirm={handleConfirm}
  {onCancel}
/>
