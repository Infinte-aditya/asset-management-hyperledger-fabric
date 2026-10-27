<script>
  import { api } from "../../api.js";

  let { container, open, onDone, onToast } = $props();

  let action = $state("GateIn");
  let location = $state("");
  let by = $state("");
  let notes = $state("");
  let error = $state("");
  let dialogEl;

  $effect(() => {
    if (open) {
      dialogEl?.showModal();
      action = "GateIn"; location = ""; by = ""; notes = ""; error = "";
      if (container) location = container.Location || "";
    } else {
      dialogEl?.close();
    }
  });

  async function handleSubmit(e) {
    e.preventDefault();
    error = "";
    if (!action) { error = "Action is required."; return; }
    if (!by.trim()) { error = "Reported by is required."; return; }
    try {
      const res = await api(`/api/containers/${encodeURIComponent(container.ID)}/events`, {
        method: "POST",
        body: JSON.stringify({ action, location: location.trim(), by: by.trim(), notes: notes.trim() }),
      });
      open = false;
      onToast?.(res.message, "success");
      onDone?.();
    } catch (err) {
      error = err.message;
    }
  }
</script>

<dialog class="dialog" bind:this={dialogEl} onclose={() => (open = false)}>
  <form onsubmit={handleSubmit} novalidate>
    <div class="dialog-head">
      <h2>Report Event</h2>
      <button type="button" class="icon-btn" onclick={() => (open = false)}>&times;</button>
    </div>
    <div class="dialog-body">
      {#if container}
        <p class="transfer-context">
          Container <span class="mono">{container.ID}</span> &middot; Current status: <strong>{container.Status}</strong>
        </p>
      {/if}
      <div class="field">
        <label for="fAction">Action <span class="req">*</span></label>
        <select id="fAction" class="select" bind:value={action} style="width:100%">
          <option value="GateIn">Gate In</option>
          <option value="GateOut">Gate Out</option>
          <option value="Loaded">Loaded (Vessel)</option>
          <option value="Discharged">Discharged</option>
          <option value="InTransit">In Transit</option>
          <option value="CustomsHold">Customs Hold</option>
          <option value="DamageReported">Damage Reported</option>
          <option value="Repaired">Repaired</option>
        </select>
      </div>
      <div class="field">
        <label for="fLoc">Location</label>
        <input id="fLoc" type="text" placeholder="e.g. CNSHA, USLAX, OnBoard:VesselName" bind:value={location}>
      </div>
      <div class="field-row">
        <div class="field">
          <label for="fBy">Reported By <span class="req">*</span></label>
          <input id="fBy" type="text" placeholder="e.g. Terminal-Ops" bind:value={by} required>
        </div>
      </div>
      <div class="field">
        <label for="fNotes">Notes</label>
        <input id="fNotes" type="text" placeholder="Additional details…" bind:value={notes}>
      </div>
      {#if error}
        <p class="form-error">{error}</p>
      {/if}
    </div>
    <div class="dialog-foot">
      <button type="button" class="btn btn-ghost" onclick={() => (open = false)}>Cancel</button>
      <button type="submit" class="btn btn-primary">Report Event</button>
    </div>
  </form>
</dialog>