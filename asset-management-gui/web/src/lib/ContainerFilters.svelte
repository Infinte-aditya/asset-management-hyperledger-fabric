<script>
  import { query, statusFilter, typeFilter, sort } from "../stores.js";

  const statusOptions = ["", "Empty", "Loaded", "InTransit", "AtPort", "UnderCustoms", "Damaged", "Repairing"];
  const typeOptions = ["", "20GP", "40GP", "40RH", "20OT", "40FR"];
  const sortOptions = [
    { value: "id", label: "Sort: ID" },
    { value: "type", label: "Sort: Type" },
    { value: "status", label: "Sort: Status" },
    { value: "owner", label: "Sort: Owner" },
    { value: "weight", label: "Sort: Weight" },
  ];

  let qVal = $state("");
  let sVal = $state("");
  let tVal = $state("");
  let sortVal = $state("id");

  $effect(() => {
    query.set(qVal);
  });
  $effect(() => {
    statusFilter.set(sVal);
  });
  $effect(() => {
    typeFilter.set(tVal);
  });
  $effect(() => {
    sort.set(sortVal);
  });
</script>

<section class="toolbar">
  <div class="search-wrap">
    <svg class="search-icon" viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
    <input type="search" id="searchInput" placeholder="Search by ID, owner or location…" bind:value={qVal}>
  </div>
  <select class="select" bind:value={tVal}>
    <option value="">All Types</option>
    {#each typeOptions.slice(1) as t}
      <option value={t}>{t}</option>
    {/each}
  </select>
  <select class="select" bind:value={sVal}>
    <option value="">All Statuses</option>
    {#each statusOptions.slice(1) as st}
      <option value={st}>{st}</option>
    {/each}
  </select>
  <select class="select" bind:value={sortVal}>
    {#each sortOptions as opt}
      <option value={opt.value}>{opt.label}</option>
    {/each}
  </select>
</section>