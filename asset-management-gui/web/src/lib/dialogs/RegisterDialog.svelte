<script>
  import { api } from "../../api.js";

  let { open, onDone, onToast } = $props();

  let id = $state("");
  let type = $state("20GP");
  let owner = $state("");
  let tareWeight = $state("");
  let destination = $state("");
  let error = $state("");

  let dialogEl;

  $effect(() => {
    if (open) {
      dialogEl?.showModal();
      id = ""; type = "20GP"; owner = ""; tareWeight = ""; destination = ""; error = "";
    } else {
      dialogEl?.close();
    }
  });

  async function handleSubmit(e) {
    e.preventDefault();
    error = "";
    if (!id.trim()) { error = "Container ID is required."; return; }
    if (!owner.trim()) { error = "Owner is required."; return; }
    if (!tareWeight || isNaN(tareWeight) || Number(tareWeight) < 0) { error = "Tare weight must be a non-negative number."; return; }
    if (!destination.trim()) { error = "Destination is required."; return; }
    try {
      const res = await api("/api/containers", {
        method: "POST",
        body: JSON.stringify({ id: id.trim(), type, owner: owner.trim(), tareWeight: Number(tareWeight), destination: destination.trim() }),
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
      <h2>Register Container</h2>
      <button type="button" class="icon-btn" onclick={() => (open = false)}>&times;</button>
    </div>
    <div class="dialog-body">
      <div class="field">
        <label for="fId">Container ID <span class="req">*</span></label>
        <input id="fId" type="text" placeholder="e.g. MSCU1234567" bind:value={id} autocomplete="off" required>
      </div>
      <div class="field-row">
        <div class="field">
          <label for="fType">Type <span class="req">*</span></label>
          <select id="fType" class="select" bind:value={type} style="width:100%">
            <option value="20GP">20' Dry (20GP)</option>
            <option value="40GP">40' Dry (40GP)</option>
            <option value="40RH">40' Reefer (40RH)</option>
            <option value="20OT">20' Open Top (20OT)</option>
            <option value="40FR">40' Flat Rack (40FR)</option>
          </select>
        </div>
        <div class="field">
          <label for="fTare">Tare Weight (kg) <span class="req">*</span></label>
          <input id="fTare" type="number" min="0" placeholder="e.g. 3700" bind:value={tareWeight} required>
        </div>
      </div>
      <div class="field">
        <label for="fOwner">Owner <span class="req">*</span></label>
        <input id="fOwner" type="text" placeholder="e.g. MSC, Maersk, COSCO" bind:value={owner} required>
      </div>
      <div class="field">
        <label for="fDest">Destination <span class="req">*</span></label>
        <input id="fDest" type="text" placeholder="e.g. USNYC, NLRTM, DEHAM" bind:value={destination} required>
      </div>
      {#if error}
        <p class="form-error">{error}</p>
      {/if}
    </div>
    <div class="dialog-foot">
      <button type="button" class="btn btn-ghost" onclick={() => (open = false)}>Cancel</button>
      <button type="submit" class="btn btn-primary">Register Container</button>
    </div>
  </form>
</dialog>