<script>
  import { containers } from "../stores.js";
  import { fmtMoney, fmtWeight, fmtDate } from "../utils.js";

  let stats = $derived.by(() => {
    const all = $containers;
    const total = all.length;
    const inTransit = all.filter((c) => c.Status === "InTransit").length;
    const underCustoms = all.filter((c) => c.CustomsStatus === "Hold" || c.Status === "UnderCustoms").length;
    const damaged = all.filter((c) => c.Status === "Damaged" || c.Status === "Repairing").length;
    const reefers = all.filter((c) => c.Type === "40RH").length;
    return { total, inTransit, underCustoms, damaged, reefers };
  });
</script>

<section class="stats">
  <div class="stat-card">
    <div class="stat-icon stat-icon-indigo">
      <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="4" width="18" height="16" rx="1"/><line x1="3" y1="10" x2="21" y2="10"/><line x1="12" y1="4" x2="12" y2="20"/></svg>
    </div>
    <div><div class="stat-value">{stats.total}</div><div class="stat-label">Total Containers</div></div>
  </div>
  <div class="stat-card">
    <div class="stat-icon stat-icon-indigo" style="background:rgba(99,102,241,0.14);color:#8183f7;">
      <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 15h18M3 15l3-9h12l3 9M3 15v4h18v-4"/><path d="M8 15v-2a4 4 0 0 1 8 0v2"/></svg>
    </div>
    <div><div class="stat-value">{stats.inTransit}</div><div class="stat-label">In Transit</div></div>
  </div>
  <div class="stat-card">
    <div class="stat-icon stat-icon-amber">
      <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
    </div>
    <div><div class="stat-value">{stats.underCustoms}</div><div class="stat-label">Under Customs</div></div>
  </div>
  <div class="stat-card">
    <div class="stat-icon" style="background:rgba(239,68,68,0.12);color:#f87171;">
      <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 9v4"/><circle cx="12" cy="20" r="1"/><path d="M12 3v1"/></svg>
    </div>
    <div><div class="stat-value">{stats.damaged}</div><div class="stat-label">Damaged / Repairing</div></div>
  </div>
  <div class="stat-card">
    <div class="stat-icon" style="background:rgba(6,182,212,0.12);color:#22d3ee;">
      <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="4" width="20" height="16" rx="2"/><line x1="6" y1="10" x2="6" y2="14"/><circle cx="9" cy="12" r="1" fill="currentColor"/><circle cx="12" cy="12" r="1" fill="currentColor"/><circle cx="15" cy="12" r="1" fill="currentColor"/></svg>
    </div>
    <div><div class="stat-value">{stats.reefers}</div><div class="stat-label">Active Reefers</div></div>
  </div>
</section>