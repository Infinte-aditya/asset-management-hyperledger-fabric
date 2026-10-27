<script>
  import { containers, query, statusFilter, typeFilter, sort } from "../stores.js";
  import { initials, ownerHue, typeLabel, statusColor, customsBadge, fmtWeight, containerTypeIcon } from "../utils.js";

  let { loading, onDetail, onTransfer, onEvent, onDamage, onCustoms, onDelete, onSeed } = $props();

  let filtered = $derived.by(() => {
    const all = $containers;
    const q = $query.toLowerCase();
    const sf = $statusFilter;
    const tf = $typeFilter;
    const sv = $sort;

    let list = all.filter((c) => {
      if (sf && c.Status !== sf) return false;
      if (tf && c.Type !== tf) return false;
      if (q && !c.ID.toLowerCase().includes(q) && !c.Owner.toLowerCase().includes(q) && !(c.Location || "").toLowerCase().includes(q))
        return false;
      return true;
    });

    const cmp = {
      id: (a, b) => a.ID.localeCompare(b.ID, undefined, { numeric: true }),
      type: (a, b) => a.Type.localeCompare(b.Type),
      status: (a, b) => a.Status.localeCompare(b.Status),
      owner: (a, b) => a.Owner.localeCompare(b.Owner) || a.ID.localeCompare(b.ID),
      weight: (a, b) => (b.GrossWeight || 0) - (a.GrossWeight || 0),
    };
    return [...list].sort(cmp[sv] || cmp.id);
  });

  function dotCls(status) {
    return `status-dot-sm ${statusColor(status)}`;
  }

  function customsCls(s) {
    return `customs-badge ${customsBadge(s)}`;
  }
</script>

<section class="table-card">
  <table class="asset-table">
    <thead>
      <tr>
        <th>ID</th>
        <th>Type</th>
        <th>Status</th>
        <th>Location</th>
        <th>Owner</th>
        <th>Cargo</th>
        <th class="num">Gross Wt</th>
        <th>Destination</th>
        <th>Customs</th>
        <th class="actions-col">Actions</th>
      </tr>
    </thead>
    <tbody id="assetTableBody">
      {#each filtered as c (c.ID)}
        <tr>
          <td>
            <button class="asset-id" onclick={() => onDetail(c)}>{c.ID}</button>
          </td>
          <td>
            <span class="type-badge type-{c.Type}">
              {@html containerTypeIcon(c.Type)}
              <span>{typeLabel(c.Type)}</span>
            </span>
          </td>
          <td>
            <span class="status-badge {statusColor(c.Status)}">
              <span class={dotCls(c.Status)}></span>
              {c.Status}
            </span>
          </td>
          <td><span class="mono" style="font-size:13px">{c.Location || "—"}</span></td>
          <td>
            <span class="owner-cell">
              <span class="owner-avatar" style="background:hsl({ownerHue(c.Owner)} 55% 42%)">{initials(c.Owner)}</span>
              <span class="owner-name">{c.Owner}</span>
            </span>
          </td>
          <td><span style="color:var(--text-dim);font-size:13px">{(c.CargoDesc || "—").slice(0, 24)}{(c.CargoDesc || "").length > 24 ? "…" : ""}</span></td>
          <td class="num"><span class="mono">{fmtWeight(c.GrossWeight)}</span></td>
          <td><span class="mono" style="font-size:13px">{c.Destination || "—"}</span></td>
          <td>
            {#if c.CustomsStatus}
              <span class={customsCls(c.CustomsStatus)}>{c.CustomsStatus}</span>
            {:else}
              <span style="color:var(--text-faint)">—</span>
            {/if}
          </td>
          <td class="actions-col">
            <div class="row-actions">
              <button class="icon-btn accent" title="Details" onclick={() => onDetail(c)}>
                <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>
              </button>
              <button class="icon-btn accent" title="Report event" onclick={() => onEvent(c)}>
                <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
              </button>
              <button class="icon-btn accent" title="Transfer" onclick={() => onTransfer(c)}>
                <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2"><polyline points="17 1 21 5 17 9"/><path d="M3 11V9a4 4 0 0 1 4-4h14"/><polyline points="7 23 3 19 7 15"/><path d="M21 13v2a4 4 0 0 1-4 4H3"/></svg>
              </button>
              <button class="icon-btn danger" title="Delete" onclick={() => onDelete(c)}>
                <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
              </button>
            </div>
          </td>
        </tr>
      {:else}
        <tr>
          <td colspan="10">
            <div class="empty-state">
              <svg viewBox="0 0 24 24" width="56" height="56" fill="none" stroke="currentColor" stroke-width="1.2"><rect x="3" y="4" width="18" height="16" rx="1"/><line x1="3" y1="10" x2="21" y2="10"/><line x1="12" y1="4" x2="12" y2="20"/></svg>
              <h3>No containers on the ledger yet</h3>
              <p>Register your first container, or load the built-in sample data set.</p>
              <div class="empty-actions">
                <button class="btn btn-primary" onclick={() => onDetail(null)}>+ Register Container</button>
                <button class="btn btn-ghost" onclick={onSeed}>Load sample data</button>
              </div>
            </div>
          </td>
        </tr>
      {/each}
    </tbody>
  </table>
  <div class="table-footer">
    <span class="toolbar-count">{filtered.length} of {$containers.length} containers</span>
  </div>
</section>