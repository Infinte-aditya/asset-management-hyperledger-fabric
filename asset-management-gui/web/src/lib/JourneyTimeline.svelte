<script>
  import { fmtDateTime, statusColor, eventIcon } from "../utils.js";

  let { events = [] } = $props();
</script>

<div class="timeline">
  {#each [...events].reverse() as ev, i (i)}
    <div class="timeline-item">
      <div class="timeline-dot {statusColor(ev.Action === 'DamageReported' ? 'Damaged' : '')}"></div>
      {#if i < events.length - 1}<div class="timeline-line"></div>{/if}
      <div class="timeline-content">
        <div class="timeline-head">
          <span class="timeline-action">
            <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" style="vertical-align:middle">{@html eventIcon(ev.Action)}</svg>
            {ev.Action}
          </span>
          <span class="timeline-time">{fmtDateTime(ev.Timestamp)}</span>
        </div>
        <div class="timeline-meta">
          {#if ev.Location}<span class="timeline-loc">{ev.Location}</span>{/if}
          {#if ev.By}<span class="timeline-by">by {ev.By}</span>{/if}
        </div>
        {#if ev.Notes}
          <div class="timeline-notes">{ev.Notes}</div>
        {/if}
      </div>
    </div>
  {:else}
    <div style="color:var(--text-faint);text-align:center;padding:20px">No journey events recorded yet.</div>
  {/each}
</div>

<style>
  .timeline { position: relative; padding: 8px 0; }
  .timeline-item { display: flex; align-items: flex-start; gap: 12px; position: relative; padding: 6px 0; }
  .timeline-dot {
    width: 12px; height: 12px; border-radius: 50%; flex-shrink: 0;
    margin-top: 4px; z-index: 1;
    background: var(--accent);
  }
  .timeline-dot.Damaged { background: var(--red); }
  .timeline-dot.UnderCustoms { background: var(--amber); }
  .timeline-dot.InTransit { background: var(--accent); }
  .timeline-line {
    position: absolute; left: 5px; top: 22px; bottom: -6px;
    width: 2px; background: var(--border); z-index: 0;
  }
  .timeline-content { flex: 1; min-width: 0; }
  .timeline-head { display: flex; justify-content: space-between; align-items: center; gap: 8px; }
  .timeline-action { font-weight: 600; font-size: 13.5px; display: inline-flex; align-items: center; gap: 4px; }
  .timeline-time { font-size: 12px; color: var(--text-faint); font-family: var(--mono); white-space: nowrap; }
  .timeline-meta { display: flex; gap: 10px; margin-top: 2px; font-size: 12.5px; color: var(--text-dim); }
  .timeline-loc { font-family: var(--mono); }
  .timeline-by { color: var(--text-faint); }
  .timeline-notes { margin-top: 3px; font-size: 13px; color: var(--text-dim); background: var(--bg-soft); padding: 6px 10px; border-radius: 6px; }
</style>