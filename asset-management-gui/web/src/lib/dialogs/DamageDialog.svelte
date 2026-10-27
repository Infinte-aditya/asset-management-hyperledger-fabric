<script>
  import { api } from "../../api.js";

  let { container, open, onDone, onToast } = $props();

  let damageType = $state("Dent");
  let description = $state("");
  let reportedBy = $state("");
  let error = $state("");
  let dialogEl;

  $effect(() => {
    if (open) {
      dialogEl?.showModal();
      damageType = "Dent"; description = ""; reportedBy = ""; error = "";
    } else {
      dialogEl?.close();
    }
  });

  async function handleSubmit(e) {
    e.preventDefault();
    error = "";
    if (!damageType) { error = "Damage type is required."; return; }
    if (!reportedBy.trim()) { error = "Reported by is required."; return; }
    try {
      const res = await api(`/api/containers/${encodeURIComponent(container.ID)}/damage`, {
        method: "POST",
        body: JSON.stringify({ damageType, description: description.trim(), reportedBy: reportedBy.trim() }),
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
      <h2 class="danger-title">Report Damage</h2>
      <button type="button" class="icon-btn" onclick={() => (open = false)}>&times;</button>
    </div>
    <div class="dialog-body">
      {#if container}
        <p class="transfer-context">
          Container <span class="mono">{container.ID}</span> &middot; Current location: <strong>{container.Location}</strong>
        </p>
      {/if}
      <div class="field">
        <label for="fDamageType">Damage Type <span class="req">*</span></label>
        <select id="fDamageType" class="select" bind:value={damageType} style="width:100%">
          <option value="Dent">Dent</option>
          <option value="Crack">Crack</option>
          <option value="Rust">Corrosion / Rust</option>
          <option value="Hole">Hole / Puncture</option>
          <option value="Structural">Structural Damage</option>
          <option value="Door">Door Damage</option>
          <option value="Floor">Floor Damage</option>
          <option value="Reefer">Reefer Malfunction</option>
          <option value="Other">Other</option>
        </select>
      </div>
      <div class="field">
        <label for="fDesc">Description</label>
        <input id="fDesc" type="text" placeholder="e.g. 30cm dent on right wall panel" bind:value={description}>
      </div>
      <div class="field">
        <label for="fRepBy">Reported By <span class="req">*</span></label>
        <input id="fRepBy" type="text" placeholder="e.g. Surveyor, Terminal-Ops" bind:value={reportedBy} required>
      </div>
      {#if error}
        <p class="form-error">{error}</p>
      {/if}
    </div>
    <div class="dialog-foot">
      <button type="button" class="btn btn-ghost" onclick={() => (open = false)}>Cancel</button>
      <button type="submit" class="btn btn-danger">Report Damage</button>
    </div>
  </form>
</dialog>