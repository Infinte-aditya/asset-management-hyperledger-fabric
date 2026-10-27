<script>
  import { api } from "../../api.js";

  let { container, open, onDone, onToast } = $props();

  let customsStatus = $state("Hold");
  let by = $state("");
  let notes = $state("");
  let error = $state("");
  let dialogEl;

  $effect(() => {
    if (open) {
      dialogEl?.showModal();
      customsStatus = "Hold"; by = ""; notes = ""; error = "";
    } else {
      dialogEl?.close();
    }
  });

  async function handleSubmit(e) {
    e.preventDefault();
    error = "";
    if (!customsStatus) { error = "Customs status is required."; return; }
    if (!by.trim()) { error = "Officer name is required."; return; }
    try {
      const res = await api(`/api/containers/${encodeURIComponent(container.ID)}/customs`, {
        method: "POST",
        body: JSON.stringify({ customsStatus, by: by.trim(), notes: notes.trim() }),
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
      <h2>Customs Action</h2>
      <button type="button" class="icon-btn" onclick={() => (open = false)}>&times;</button>
    </div>
    <div class="dialog-body">
      {#if container}
        <p class="transfer-context">
          Container <span class="mono">{container.ID}</span> &middot; Current: <strong>{container.CustomsStatus}</strong>
        </p>
      {/if}
      <div class="field">
        <label for="fCustoms">Action <span class="req">*</span></label>
        <select id="fCustoms" class="select" bind:value={customsStatus} style="width:100%">
          <option value="Hold">Hold for Inspection</option>
          <option value="Clear">Clear / Release</option>
          <option value="ExamRequired">Exam Required</option>
        </select>
      </div>
      <div class="field">
        <label for="fCustomsBy">Officer <span class="req">*</span></label>
        <input id="fCustomsBy" type="text" placeholder="e.g. CBP-Officer" bind:value={by} required>
      </div>
      <div class="field">
        <label for="fCustomsNotes">Notes</label>
        <input id="fCustomsNotes" type="text" placeholder="Reason for action…" bind:value={notes}>
      </div>
      {#if error}
        <p class="form-error">{error}</p>
      {/if}
    </div>
    <div class="dialog-foot">
      <button type="button" class="btn btn-ghost" onclick={() => (open = false)}>Cancel</button>
      <button type="submit" class="btn btn-primary">Submit</button>
    </div>
  </form>
</dialog>