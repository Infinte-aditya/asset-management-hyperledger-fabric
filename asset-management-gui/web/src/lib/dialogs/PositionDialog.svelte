<script>
  import { api } from "../../api.js";

  let { container, open, onDone, onToast } = $props();

  let mode = $state("single"); // "single" | "bulk"
  let lat = $state("");
  let lon = $state("");
  let timestamp = $state("");
  let speed = $state("");
  let heading = $state("");
  let temperature = $state("");
  let source = $state("GPS:ui");
  let bulk = $state("");
  let keepOpen = $state(true);
  let error = $state("");
  let saving = $state(false);
  let dialogEl;

  $effect(() => {
    if (open) {
      dialogEl?.showModal();
      mode = "single";
      error = "";
      saving = false;
      keepOpen = true;
      bulk = "";
      timestamp = "";
      speed = "";
      heading = "";
      source = "GPS:ui";
      // Seed from the container's last known fix for convenience.
      const p = container?.Position;
      lat = p?.Lat ?? "";
      lon = p?.Lon ?? "";
      temperature = p?.Temperature ?? "";
    } else {
      dialogEl?.close();
    }
  });

  function validCoord(a, min, max) {
    const n = Number(a);
    return Number.isFinite(n) && n >= min && n <= max;
  }

  function nowRfc3339(offsetSeconds = 0) {
    return new Date(Date.now() + offsetSeconds * 1000).toISOString();
  }

  function buildBody(l, o, ts, extra = {}) {
    return {
      lat: String(l),
      lon: String(o),
      timestamp: ts || "",
      speed: extra.speed ?? speed,
      heading: extra.heading ?? heading,
      temperature: extra.temperature ?? temperature,
      source: source || "GPS:ui",
    };
  }

  async function postOne(id, body) {
    return api(`/api/containers/${encodeURIComponent(id)}/position`, {
      method: "POST",
      body: JSON.stringify(body),
    });
  }

  // Parse "lat, lon[, temp]" lines (comma or whitespace separated).
  function parseBulk(text) {
    const rows = [];
    for (const raw of text.split(/\r?\n/)) {
      const line = raw.trim();
      if (!line || line.startsWith("#")) continue;
      const parts = line.split(/[\s,;]+/).filter(Boolean);
      if (parts.length < 2) continue;
      rows.push({ lat: parts[0], lon: parts[1], temp: parts[2] ?? "" });
    }
    return rows;
  }

  async function addSingle() {
    error = "";
    if (!validCoord(lat, -90, 90)) { error = "Latitude must be a number between -90 and 90."; return; }
    if (!validCoord(lon, -180, 180)) { error = "Longitude must be a number between -180 and 180."; return; }
    saving = true;
    try {
      const rec = await postOne(container.ID, buildBody(lat, lon, timestamp));
      onToast?.(rec.message || "GPS stop added", "success");
      onDone?.();
      if (keepOpen) {
        // Clear coords so the next stop can be entered quickly.
        lat = ""; lon = ""; speed = ""; heading = "";
      } else {
        open = false;
      }
    } catch (err) {
      error = err.message;
    } finally {
      saving = false;
    }
  }

  async function addBulk() {
    error = "";
    const rows = parseBulk(bulk);
    if (rows.length === 0) { error = "Paste at least one line: lat, lon[, temp]"; return; }
    for (const r of rows) {
      if (!validCoord(r.lat, -90, 90) || !validCoord(r.lon, -180, 180)) {
        error = `Invalid coordinates: "${r.lat}, ${r.lon}". Lat must be ±90, lon ±180.`;
        return;
      }
    }
    saving = true;
    let ok = 0, fail = 0, lastErr = "";
    const base = Date.now();
    for (let i = 0; i < rows.length; i++) {
      const r = rows[i];
      // Space the fixes 60s apart so the track orders correctly.
      const ts = nowRfc3339(Math.round((base + i * 60000 - Date.now()) / 1000));
      try {
        await postOne(container.ID, buildBody(r.lat, r.lon, ts, { temperature: r.temp }));
        ok++;
      } catch (err) {
        fail++;
        lastErr = err.message;
      }
    }
    saving = false;
    onDone?.();
    if (fail === 0) {
      onToast?.(`Added ${ok} GPS stop${ok === 1 ? "" : "s"}`, "success");
      bulk = "";
      open = false;
    } else {
      error = `Added ${ok}, failed ${fail}. ${lastErr}`;
    }
  }

  function submit(e) {
    e.preventDefault();
    if (mode === "bulk") addBulk();
    else addSingle();
  }
</script>

<dialog class="dialog" bind:this={dialogEl} onclose={() => (open = false)}>
  <form onsubmit={submit} novalidate>
    <div class="dialog-head">
      <h2>Add GPS Stop</h2>
      <button type="button" class="icon-btn" onclick={() => (open = false)}>&times;</button>
    </div>
    <div class="dialog-body">
      {#if container}
        <p class="transfer-context">
          Recording a position for <span class="mono">{container.ID}</span>
          {#if container.Position && container.Position.Timestamp}
            &middot; last fix {container.Position.Lat}, {container.Position.Lon}
          {/if}
        </p>
      {/if}

      <div class="mode-tabs">
        <button type="button" class="mode-tab" class:active={mode === "single"} onclick={() => (mode = "single")}>Single stop</button>
        <button type="button" class="mode-tab" class:active={mode === "bulk"} onclick={() => (mode = "bulk")}>Bulk paste</button>
      </div>

      {#if mode === "single"}
        <div class="field-row">
          <div class="field">
            <label for="pLat">Latitude <span class="req">*</span></label>
            <input id="pLat" type="text" inputmode="decimal" placeholder="e.g. 31.2304" bind:value={lat} required>
          </div>
          <div class="field">
            <label for="pLon">Longitude <span class="req">*</span></label>
            <input id="pLon" type="text" inputmode="decimal" placeholder="e.g. 121.4737" bind:value={lon} required>
          </div>
        </div>
        <p class="hint">Only lat/lon are required — the timestamp defaults to now and source to <span class="mono">GPS:ui</span>.</p>

        <details class="adv">
          <summary>Optional details (time, speed, heading, temp, source)</summary>
          <div class="adv-body">
            <div class="field">
              <label for="pTime">Timestamp (RFC3339 UTC)</label>
              <input id="pTime" type="text" placeholder="blank = now, e.g. 2026-11-19T09:00:00Z" bind:value={timestamp}>
            </div>
            <div class="field-row">
              <div class="field">
                <label for="pSpeed">Speed (knots)</label>
                <input id="pSpeed" type="text" inputmode="decimal" placeholder="e.g. 12.5" bind:value={speed}>
              </div>
              <div class="field">
                <label for="pHead">Heading (°)</label>
                <input id="pHead" type="text" inputmode="decimal" placeholder="0–360" bind:value={heading}>
              </div>
            </div>
            <div class="field-row">
              <div class="field">
                <label for="pTemp">Temperature (°C)</label>
                <input id="pTemp" type="text" inputmode="decimal" placeholder="e.g. 4.5" bind:value={temperature}>
              </div>
              <div class="field">
                <label for="pSrc">Source</label>
                <input id="pSrc" type="text" placeholder="GPS:ui" bind:value={source}>
              </div>
            </div>
          </div>
        </details>

        <label class="check">
          <input type="checkbox" bind:checked={keepOpen}>
          Keep open to add another stop
        </label>
      {:else}
        <div class="field">
          <label for="pBulk">Stops — one per line: <span class="mono">lat, lon[, temp]</span></label>
          <textarea id="pBulk" rows="7" spellcheck="false" placeholder={"31.2304, 121.4737, 4.5\n30.5, 124.0, 5.0\n32.0, 130.0"} bind:value={bulk}></textarea>
        </div>
        <p class="hint">Each line becomes a fix, spaced 60s apart so the track orders correctly. Lines starting with <span class="mono">#</span> are ignored.</p>
      {/if}

      {#if error}
        <p class="form-error">{error}</p>
      {/if}
    </div>
    <div class="dialog-foot">
      <button type="button" class="btn btn-ghost" onclick={() => (open = false)}>Cancel</button>
      <button type="submit" class="btn btn-primary" disabled={saving}>
        {saving ? "Saving…" : mode === "bulk" ? "Add stops" : "Add stop"}
      </button>
    </div>
  </form>
</dialog>

<style>
  .mode-tabs { display: flex; gap: 6px; margin-bottom: 16px; background: var(--bg-soft); padding: 4px; border-radius: 10px; border: 1px solid var(--border-soft); }
  .mode-tab {
    flex: 1; border: none; background: transparent; color: var(--text-dim);
    font-size: 13px; font-weight: 600; padding: 8px 10px; border-radius: 7px; cursor: pointer;
  }
  .mode-tab.active { background: var(--card, #1b2233); color: var(--text); box-shadow: 0 1px 3px rgba(0,0,0,0.3); }
  .hint { margin: 6px 0 14px; font-size: 12px; color: var(--text-faint, var(--text-dim)); line-height: 1.5; }
  .adv { margin: 4px 0 14px; }
  .adv summary { cursor: pointer; font-size: 12.5px; color: var(--text-dim); user-select: none; }
  .adv-body { margin-top: 12px; }
  .check { display: flex; align-items: center; gap: 8px; font-size: 13px; color: var(--text-dim); cursor: pointer; }
  .check input { width: 15px; height: 15px; }
  textarea {
    width: 100%; resize: vertical; padding: 10px 12px; border-radius: 8px;
    border: 1px solid var(--border); background: var(--bg-soft); color: var(--text);
    font-family: var(--mono, monospace); font-size: 13px; line-height: 1.5;
  }
  textarea:focus { outline: 2px solid var(--accent, #6366f1); outline-offset: 1px; }
  :global(.dialog .mono) { font-family: var(--mono, monospace); }
</style>
