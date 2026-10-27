<script>
  import { api } from "../../api.js";

  let { container, open, onDone, onToast } = $props();

  let dialogEl;

  $effect(() => {
    if (open) dialogEl?.showModal();
    else dialogEl?.close();
  });

  async function handleDelete() {
    try {
      const res = await api(`/api/containers/${encodeURIComponent(container.ID)}`, { method: "DELETE" });
      open = false;
      onToast?.(res.message, "success");
      onDone?.();
    } catch (err) {
      open = false;
      onToast?.(err.message, "error");
    }
  }
</script>

<dialog class="dialog" bind:this={dialogEl} onclose={() => (open = false)}>
  <div class="dialog-head">
    <h2 class="danger-title">De-Register Container</h2>
    <button type="button" class="icon-btn" onclick={() => (open = false)}>&times;</button>
  </div>
  <div class="dialog-body">
    {#if container}
      <p>Are you sure you want to de-register container <span class="mono">{container.ID}</span> owned by <strong>{container.Owner}</strong>?</p>
      <p class="warning-note">This submits a blockchain transaction that permanently removes the container from the ledger.</p>
    {/if}
  </div>
  <div class="dialog-foot">
    <button type="button" class="btn btn-ghost" onclick={() => (open = false)}>Cancel</button>
    <button type="button" class="btn btn-danger" onclick={handleDelete}>De-Register Container</button>
  </div>
</dialog>