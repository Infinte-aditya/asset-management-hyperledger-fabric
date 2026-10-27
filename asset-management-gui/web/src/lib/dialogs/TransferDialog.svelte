<script>
  import { api } from "../../api.js";
  import { initials, ownerHue } from "../../utils.js";

  let { container, open, onDone, onToast } = $props();

  let newOwner = $state("");
  let error = $state("");
  let dialogEl;

  $effect(() => {
    if (open) {
      dialogEl?.showModal();
      newOwner = "";
      error = "";
    } else {
      dialogEl?.close();
    }
  });

  async function handleSubmit(e) {
    e.preventDefault();
    error = "";
    if (!newOwner.trim()) { error = "New owner is required."; return; }
    if (container && newOwner.trim() === container.Owner) { error = "The new owner must be different from the current owner."; return; }
    try {
      const res = await api(`/api/containers/${encodeURIComponent(container.ID)}/transfer`, {
        method: "POST",
        body: JSON.stringify({ newOwner: newOwner.trim() }),
      });
      open = false;
      const prev = res.previousOwner ? ` (was ${res.previousOwner})` : "";
      onToast?.(`Container "${container.ID}" transferred to ${newOwner.trim()}${prev}`, "success");
      onDone?.();
    } catch (err) {
      error = err.message;
    }
  }
</script>

<dialog class="dialog" bind:this={dialogEl} onclose={() => (open = false)}>
  <form onsubmit={handleSubmit} novalidate>
    <div class="dialog-head">
      <h2>Transfer Container</h2>
      <button type="button" class="icon-btn" onclick={() => (open = false)}>&times;</button>
    </div>
    <div class="dialog-body">
      {#if container}
        <p class="transfer-context">
          <span class="mono" id="transferAssetId">{container.ID}</span>
          currently owned by <strong>{container.Owner}</strong>
        </p>
        <div class="transfer-flow">
          <span class="owner-chip">
            <span class="owner-avatar" style="background:hsl({ownerHue(container.Owner)} 55% 42%)">{initials(container.Owner)}</span>
            <span>{container.Owner}</span>
          </span>
          <svg viewBox="0 0 24 24" width="22" height="22" fill="none" stroke="currentColor" stroke-width="2"><line x1="5" y1="12" x2="19" y2="12"/><polyline points="12 5 19 12 12 19"/></svg>
          <input type="text" class="owner-input" placeholder="New owner" bind:value={newOwner} required autocomplete="off">
        </div>
      {/if}
      {#if error}
        <p class="form-error">{error}</p>
      {/if}
    </div>
    <div class="dialog-foot">
      <button type="button" class="btn btn-ghost" onclick={() => (open = false)}>Cancel</button>
      <button type="submit" class="btn btn-primary">Transfer</button>
    </div>
  </form>
</dialog>