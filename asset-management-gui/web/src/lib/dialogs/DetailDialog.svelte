<script>
  import { fmtMoney, fmtWeight, fmtDate, fmtDateTime, typeLabel, initials, ownerHue, containerTypeIcon } from "../../utils.js";
  import JourneyTimeline from "../JourneyTimeline.svelte";

  let { container, open, onTransfer } = $props();

  let dialogEl;

  $effect(() => {
    if (open) dialogEl?.showModal();
    else dialogEl?.close();
  });

  function val(v, fallback = "—") {
    return v ?? fallback;
  }
</script>

<dialog class="dialog dialog-wide" bind:this={dialogEl} onclose={() => (open = false)}>
  <div class="dialog-head">
    <h2>Container Details</h2>
    <button type="button" class="icon-btn" onclick={() => (open = false)}>&times;</button>
  </div>
  <div class="dialog-body">
    {#if container}
      <div class="detail-grid">
        <div class="detail-item full">
          <div style="display:flex;align-items:center;gap:10px">
            <span class="type-badge type-{container.Type}">{@html containerTypeIcon(container.Type)} {typeLabel(container.Type)}</span>
            <span class="mono" style="font-size:18px;font-weight:700">{container.ID}</span>
          </div>
        </div>
        <div class="detail-item">
          <div class="detail-label">Status</div>
          <div class="detail-value"><span class="status-badge {(()=>{const m={'Empty':'gray','Loaded':'blue','InTransit':'indigo','AtPort':'teal','UnderCustoms':'amber','Damaged':'red','Repairing':'orange'};return m[container.Status]||'gray'})()}"><span class="status-dot-sm"></span>{container.Status}</span></div>
        </div>
        <div class="detail-item">
          <div class="detail-label">Location</div>
          <div class="detail-value mono" style="font-size:14px">{val(container.Location)}</div>
        </div>
        <div class="detail-item">
          <div class="detail-label">Owner</div>
          <div class="detail-value">
            <span class="owner-cell">
              <span class="owner-avatar" style="background:hsl({ownerHue(container.Owner)} 55% 42%)">{initials(container.Owner)}</span>
              {container.Owner}
            </span>
          </div>
        </div>
        <div class="detail-item">
          <div class="detail-label">Destination</div>
          <div class="detail-value mono">{val(container.Destination)}</div>
        </div>
        <div class="detail-item">
          <div class="detail-label">Gross / Net Weight</div>
          <div class="detail-value mono">{fmtWeight(container.GrossWeight)} / {fmtWeight(container.NetWeight)}</div>
        </div>
        <div class="detail-item">
          <div class="detail-label">Tare Weight</div>
          <div class="detail-value mono">{fmtWeight(container.TareWeight)}</div>
        </div>
        <div class="detail-item">
          <div class="detail-label">Seal Number</div>
          <div class="detail-value mono">{val(container.SealNumber)}</div>
        </div>
        <div class="detail-item">
          <div class="detail-label">Cargo Description</div>
          <div class="detail-value">{val(container.CargoDesc)}</div>
        </div>
        <div class="detail-item">
          <div class="detail-label">Bill of Lading</div>
          <div class="detail-value mono">{val(container.BillOfLading)}</div>
        </div>
        <div class="detail-item">
          <div class="detail-label">Last Inspection</div>
          <div class="detail-value">{fmtDate(container.LastInspection)}</div>
        </div>
        <div class="detail-item">
          <div class="detail-label">Customs Status</div>
          <div class="detail-value">
            {#if container.CustomsStatus}
              <span class="customs-badge {container.CustomsStatus === 'Hold' ? 'red' : container.CustomsStatus === 'ExamRequired' ? 'amber' : 'green'}">{container.CustomsStatus}</span>
            {:else}—{/if}
          </div>
        </div>
        {#if container.Type === "40RH"}
          <div class="detail-item">
            <div class="detail-label">Temperature Setpoint</div>
            <div class="detail-value">{container.TempSetpoint ?? "—"}°C</div>
          </div>
        {/if}
      </div>

      <div style="margin-top: 20px">
        <h3 style="font-size:14px;font-weight:600;color:var(--text-dim);margin:0 0 8px;text-transform:uppercase;letter-spacing:0.06em">Journey Timeline</h3>
        <JourneyTimeline events={container.Journey || []} />
      </div>
    {/if}
  </div>
  <div class="dialog-foot">
    <button type="button" class="btn btn-ghost" onclick={() => (open = false)}>Close</button>
    {#if container}
      <button type="button" class="btn btn-primary" onclick={() => onTransfer?.(container)}>Transfer…</button>
    {/if}
  </div>
</dialog>

<style>
  .dialog-wide { width: min(640px, calc(100vw - 32px)); }
</style>