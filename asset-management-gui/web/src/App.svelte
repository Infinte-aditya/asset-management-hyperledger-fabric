<script>
  import { containers, online, query, statusFilter, typeFilter, sort } from "./stores.js";
  import { api } from "./api.js";
  import StatsPanel from "./lib/StatsPanel.svelte";
  import ContainerFilters from "./lib/ContainerFilters.svelte";
  import ContainerTable from "./lib/ContainerTable.svelte";
  import RegisterDialog from "./lib/dialogs/RegisterDialog.svelte";
  import DetailDialog from "./lib/dialogs/DetailDialog.svelte";
  import TransferDialog from "./lib/dialogs/TransferDialog.svelte";
  import EventDialog from "./lib/dialogs/EventDialog.svelte";
  import DamageDialog from "./lib/dialogs/DamageDialog.svelte";
  import CustomsDialog from "./lib/dialogs/CustomsDialog.svelte";
  import DeleteDialog from "./lib/dialogs/DeleteDialog.svelte";
  import Toast from "./lib/Toast.svelte";

  let loading = $state(false);
  let showRegister = $state(false);
  let showDetail = $state(false);
  let showTransfer = $state(false);
  let showEvent = $state(false);
  let showDamage = $state(false);
  let showCustoms = $state(false);
  let showDelete = $state(false);
  let selectedContainer = $state(null);

  async function checkHealth() {
    const pill = document.getElementById("statusPill");
    try {
      const data = await api("/api/health");
      if (!data.ok) throw new Error(data.error);
      online.set(true);
      pill.className = "status-pill online";
      document.getElementById("statusText").textContent = "Connected";
      document.getElementById("chipChannel").textContent = data.channel;
      document.getElementById("chipCC").textContent = data.chaincode;
      document.getElementById("offlineBanner").classList.add("hidden");
      return true;
    } catch {
      online.set(false);
      pill.className = "status-pill offline";
      document.getElementById("statusText").textContent = "Offline";
      document.getElementById("offlineBanner").classList.remove("hidden");
      document.getElementById("offlineMsg").textContent =
        "Cannot reach the Fabric network. Start it with: cd test-network && ./network.sh up createChannel";
      return false;
    }
  }

  async function loadContainers(silent = false) {
    loading = true;
    try {
      const data = await api("/api/containers");
      containers.set(Array.isArray(data) ? data : []);
      document.getElementById("offlineBanner").classList.add("hidden");
    } catch (err) {
      if (!silent) addToast(err.message, "error");
      if (err.status === 503 || err.status === undefined) await checkHealth();
    } finally {
      loading = false;
    }
  }

  async function seed() {
    try {
      const res = await api("/api/seed", { method: "POST" });
      addToast(res.message, "success");
      await loadContainers(true);
    } catch (err) {
      addToast(err.message, "error");
    }
  }

  function addToast(message, kind = "info", ms = 4200) {
    const icons = { success: "✓", error: "✕", info: "ℹ" };
    const t = document.createElement("div");
    t.className = `toast ${kind}`;
    t.innerHTML = `<span class="toast-icon">${icons[kind] || icons.info}</span><span>${message}</span>`;
    document.getElementById("toasts").append(t);
    setTimeout(() => {
      t.classList.add("leaving");
      setTimeout(() => t.remove(), 220);
    }, ms);
  }

  function handleOpenDetail(c) {
    selectedContainer = c;
    showDetail = true;
  }

  function handleOpenTransfer(c) {
    selectedContainer = c;
    showTransfer = true;
  }

  function handleOpenEvent(c) {
    selectedContainer = c;
    showEvent = true;
  }

  function handleOpenDamage(c) {
    selectedContainer = c;
    showDamage = true;
  }

  function handleOpenCustoms(c) {
    selectedContainer = c;
    showCustoms = true;
  }

  function handleOpenDelete(c) {
    selectedContainer = c;
    showDelete = true;
  }

  function handleRefresh() {
    loadContainers(true);
  }

  function handleRetry() {
    checkHealth().then((ok) => {
      if (ok) {
        addToast("Reconnected to the Fabric network", "success");
        loadContainers();
      } else {
        addToast("Still offline — is the test network running?", "error");
      }
    });
  }

  $effect(() => {
    loadContainers(true);
    checkHealth();
    const interval = setInterval(() => {
      if (document.visibilityState !== "visible") return;
      if (document.querySelector("dialog[open]")) return;
      loadContainers(true);
    }, 15000);
    return () => clearInterval(interval);
  });
</script>

<div id="offlineBanner" class="offline-banner hidden">
  <span class="banner-icon">⚠</span>
  <span id="offlineMsg">Cannot reach the Fabric network. Make sure the test network is running.</span>
  <button class="btn btn-sm btn-ghost" onclick={handleRetry}>Retry</button>
</div>

<header class="header">
  <div class="header-inner">
    <div class="brand">
      <div class="brand-mark">
        <svg viewBox="0 0 24 24" width="22" height="22" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <rect x="3" y="4" width="18" height="16" rx="1"/>
          <line x1="3" y1="10" x2="21" y2="10"/>
          <line x1="12" y1="4" x2="12" y2="20"/>
        </svg>
      </div>
      <div class="brand-text">
        <h1>Container Ledger</h1>
        <p>Hyperledger Fabric &middot; Container Shipment Management</p>
      </div>
    </div>
    <div class="header-actions">
      <span class="chip" id="chipChannel">mychannel</span>
      <span class="chip" id="chipCC">container-cc</span>
      <span class="status-pill" id="statusPill">
        <span class="status-dot"></span>
        <span id="statusText">Checking&hellip;</span>
      </span>
      <button class="btn btn-ghost" onclick={handleRefresh} title="Refresh">
        <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/></svg>
        Refresh
      </button>
      <button class="btn btn-primary" onclick={() => (showRegister = true)}>
        <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
        Register Container
      </button>
    </div>
  </div>
</header>

<main class="main">
  <StatsPanel />
  <ContainerFilters />
  <ContainerTable
    {loading}
    onDetail={handleOpenDetail}
    onTransfer={handleOpenTransfer}
    onEvent={handleOpenEvent}
    onDamage={handleOpenDamage}
    onCustoms={handleOpenCustoms}
    onDelete={handleOpenDelete}
    onSeed={seed}
  />
  <footer class="footer">
    Containers are recorded on an immutable Hyperledger Fabric ledger — every action is a blockchain transaction.
  </footer>
</main>

<RegisterDialog bind:open={showRegister} onDone={() => loadContainers(true)} onToast={addToast} />
<DetailDialog container={selectedContainer} bind:open={showDetail} onTransfer={() => { showDetail = false; showTransfer = true; }} />
<TransferDialog container={selectedContainer} bind:open={showTransfer} onDone={() => loadContainers(true)} onToast={addToast} />
<EventDialog container={selectedContainer} bind:open={showEvent} onDone={() => loadContainers(true)} onToast={addToast} />
<DamageDialog container={selectedContainer} bind:open={showDamage} onDone={() => loadContainers(true)} onToast={addToast} />
<CustomsDialog container={selectedContainer} bind:open={showCustoms} onDone={() => loadContainers(true)} onToast={addToast} />
<DeleteDialog container={selectedContainer} bind:open={showDelete} onDone={() => loadContainers(true)} onToast={addToast} />
<Toast />