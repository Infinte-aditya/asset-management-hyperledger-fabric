<script>
  import { fmtMoney, fmtWeight, fmtDate, fmtDateTime, fmtLatLon, fmtTemperature, fmtHeading, typeLabel, initials, ownerHue, containerTypeIcon } from "../../utils.js";
  import { api } from "../../api.js";
  import JourneyTimeline from "../JourneyTimeline.svelte";
  import L from "leaflet";
  import "leaflet/dist/leaflet.css";

  let { container, open, onTransfer } = $props();

  let dialogEl;
  let mapEl = $state(null);

  let positions = $state([]);
  let posLoading = $state(false);
  let posError = $state(null);
  let loadedFor = $state(null);

  $effect(() => {
    if (!open) {
      positions = [];
      posLoading = false;
      posError = null;
      loadedFor = null;
      return;
    }
    const id = container?.ID;
    if (!id || loadedFor === id) return;
    loadedFor = id;
    positions = [];
    posError = null;
    posLoading = true;
    api(`/api/containers/${encodeURIComponent(id)}/positions`)
      .then((data) => {
        if (loadedFor !== id) return;
        positions = Array.isArray(data) ? data : [];
      })
      .catch((err) => {
        if (loadedFor !== id) return;
        posError = err?.message || "Could not load position history.";
      })
      .finally(() => {
        if (loadedFor === id) posLoading = false;
      });
  });

  $effect(() => {
    if (open) dialogEl?.showModal();
    else dialogEl?.close();
  });

  // Unwrap longitudes so the Pacific leg (e.g. 160 -> -175 -> -160) is drawn
  // eastward across the antimeridian instead of wrapping the wrong way round.
  function unwrapTrack(list) {
    const pts = list
      .map((p) => ({ lat: Number(p.Lat), lng: Number(p.Lon), raw: p }))
      .filter((p) => Number.isFinite(p.lat) && Number.isFinite(p.lng));
    let offset = 0;
    let prev = null;
    return pts.map((p) => {
      if (prev != null) {
        const delta = p.lng + offset - prev;
        if (delta > 180) offset -= 360;
        else if (delta < -180) offset += 360;
      }
      const lng = p.lng + offset;
      prev = lng;
      return { ...p, lng };
    });
  }

  $effect(() => {
    const isOpen = open;
    const fixes = positions;
    const el = mapEl;
    if (!isOpen || !el || !fixes || fixes.length === 0) return;

    const track = unwrapTrack(fixes);
    if (track.length === 0) return;

    const instance = L.map(el, {
      worldCopyJump: true,
      zoomControl: true,
      attributionControl: true,
    });

    L.tileLayer("https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", {
      maxZoom: 19,
      attribution:
        '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors',
    }).addTo(instance);

    const latlngs = track.map((p) => [p.lat, p.lng]);
    const line = L.polyline(latlngs, {
      color: "#6366f1",
      weight: 3,
      opacity: 0.9,
      lineJoin: "round",
    }).addTo(instance);

    track.forEach((p) => {
      L.circleMarker([p.lat, p.lng], {
        radius: 4,
        color: "#a5a7f8",
        weight: 2,
        fillColor: "#6366f1",
        fillOpacity: 1,
      })
        .addTo(instance)
        .bindTooltip(
          `${p.raw.Timestamp ? fmtDateTime(p.raw.Timestamp) : "Fix"} · ${fmtTemperature(p.raw.Temperature)}`,
          { direction: "top", className: "track-tip" }
        );
    });

    const bounds = line.getBounds();

    function resize() {
      instance.invalidateSize();
      instance.fitBounds(bounds, { padding: [28, 28], maxZoom: 6 });
    }

    const ro = new ResizeObserver(resize);
    ro.observe(el);
    // Wait for the dialog layout to settle before framing the track.
    const raf = requestAnimationFrame(resize);

    return () => {
      cancelAnimationFrame(raf);
      ro.disconnect();
      instance.remove();
    };
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
        {#if container.Position}
          <div class="detail-item full">
            <div class="detail-label">Live Position</div>
            <div class="pos-live">
              <span class="pos-coord mono" title={container.Position.Timestamp ? fmtDateTime(container.Position.Timestamp) : "Fix time unknown"}>
                {fmtLatLon(container.Position)}
              </span>
              <div class="pos-meta">
                <span>Speed <b class="mono">{val(container.Position.Speed)}</b></span>
                <span>Heading <b class="mono">{fmtHeading(container.Position.Heading)}</b></span>
                <span>Temp <b class="mono">{fmtTemperature(container.Position.Temperature)}</b></span>
                <span>Source <b class="mono">{val(container.Position.Source)}</b></span>
                <span class="pos-time">{container.Position.Timestamp ? fmtDateTime(container.Position.Timestamp) : "—"}</span>
              </div>
            </div>
          </div>
        {/if}
      </div>

      <div style="margin-top: 20px">
        <h3 style="font-size:14px;font-weight:600;color:var(--text-dim);margin:0 0 8px;text-transform:uppercase;letter-spacing:0.06em">Journey Timeline</h3>
        <JourneyTimeline events={container.Journey || []} />
      </div>

      <div style="margin-top: 20px">
        <h3 style="font-size:14px;font-weight:600;color:var(--text-dim);margin:0 0 8px;text-transform:uppercase;letter-spacing:0.06em">Position History</h3>
        {#if posLoading}
          <div class="pos-state">Loading position history…</div>
        {:else if posError}
          <div class="pos-state pos-state-error">{posError}</div>
        {:else if positions.length === 0}
          <div class="pos-state">No position history recorded yet.</div>
        {:else}
          <div class="track-map" bind:this={mapEl} role="region" aria-label="Route map of recorded positions"></div>
          <ul class="pos-list">
            {#each [...positions].reverse() as p, i (i)}
              <li class="pos-row">
                <span class="pos-row-time mono">{p.Timestamp ? fmtDateTime(p.Timestamp) : "—"}</span>
                <span class="pos-row-coord mono">{fmtLatLon(p)}</span>
                <span class="pos-row-meta">
                  {p.Speed != null && p.Speed !== "" ? `Speed ${p.Speed}` : ""}
                  {p.Heading != null && p.Heading !== "" ? `${p.Speed != null && p.Speed !== "" ? " · " : ""}Heading ${fmtHeading(p.Heading)}` : ""}
                  <span class="pos-row-temp"> · Temp {fmtTemperature(p.Temperature)}</span>
                </span>
              </li>
            {/each}
          </ul>
        {/if}
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

  .pos-live { display: flex; flex-direction: column; gap: 8px; }
  .pos-coord { font-size: 16px; font-weight: 700; color: #a5a7f8; letter-spacing: 0.01em; }
  .pos-meta {
    display: flex; flex-wrap: wrap; align-items: center; gap: 8px 16px;
    font-size: 12.5px; color: var(--text-dim);
  }
  .pos-meta b { color: var(--text); font-weight: 600; }
  .pos-time { margin-left: auto; color: var(--text-faint); font-family: var(--mono); font-size: 12px; }

  .pos-state {
    padding: 14px 16px;
    border: 1px dashed var(--border);
    border-radius: 10px;
    background: var(--bg-soft);
    color: var(--text-dim);
    font-size: 13px;
    text-align: center;
  }
  .pos-state-error { color: #fca5a5; border-color: rgba(239, 68, 68, 0.35); background: rgba(239, 68, 68, 0.08); }

  .track-map {
    height: 240px;
    margin-bottom: 10px;
    border: 1px solid var(--border);
    border-radius: 10px;
    overflow: hidden;
    background: #0f1420;
    z-index: 0;
  }
  :global(.leaflet-container) {
    background: #0f1420;
    font-family: var(--sans, inherit);
  }
  :global(.leaflet-control-attribution) {
    background: rgba(15, 20, 32, 0.82);
    color: var(--text-faint);
    font-size: 10px;
  }
  :global(.leaflet-control-attribution a) { color: var(--text-dim); }
  :global(.track-tip) {
    background: #0f1420;
    color: #e6e9f2;
    border: 1px solid #2a3450;
    border-radius: 6px;
    box-shadow: 0 6px 16px rgba(0, 0, 0, 0.45);
    font-family: var(--mono);
    font-size: 11px;
  }
  :global(.track-tip::before) { border-top-color: #2a3450; }

  .pos-list {
    list-style: none;
    margin: 0;
    padding: 0;
    border: 1px solid var(--border-soft);
    border-radius: 10px;
    overflow: hidden;
    max-height: 220px;
    overflow-y: auto;
  }
  .pos-row {
    display: flex;
    align-items: baseline;
    gap: 12px;
    padding: 9px 14px;
    border-bottom: 1px solid var(--border-soft);
    font-size: 13px;
  }
  .pos-row:last-child { border-bottom: none; }
  .pos-row:nth-child(odd) { background: var(--bg-soft); }
  .pos-row-time { color: var(--text-faint); font-size: 12px; white-space: nowrap; }
  .pos-row-coord { color: #a5a7f8; font-weight: 600; white-space: nowrap; }
  .pos-row-meta { margin-left: auto; color: var(--text-dim); font-size: 12px; text-align: right; }
  .pos-row-temp { color: #9aa3b8; }
</style>