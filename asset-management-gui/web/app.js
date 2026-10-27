/* Fabric Asset Manager — frontend application */
"use strict";

(() => {
  // ---------- State ----------
  const state = {
    assets: [],
    query: "",
    sort: "id",
    online: null,
    transferContext: null,
  };

  // ---------- Helpers ----------
  const $ = (sel) => document.querySelector(sel);
  const money = new Intl.NumberFormat("en-US", { style: "currency", currency: "USD", maximumFractionDigits: 0 });

  function el(tag, attrs = {}, ...children) {
    const node = document.createElement(tag);
    for (const [key, value] of Object.entries(attrs)) {
      if (key === "class") node.className = attrs.class;
      else if (key === "text") node.textContent = value;
      else if (key.startsWith("on") && typeof value === "function") node.addEventListener(key.slice(2), value);
      else if (value !== null && value !== undefined) node.setAttribute(key, String(value));
    }
    for (const child of children) {
      if (child === null || child === undefined) continue;
      node.append(child);
    }
    return node;
  }

  // ---------- API ----------
  async function api(path, options = {}) {
    const res = await fetch(path, {
      headers: { "Content-Type": "application/json" },
      ...options,
    });
    let data = {};
    try { data = await res.json(); } catch { /* empty body */ }
    if (!res.ok) {
      const err = new Error(data.error || `Request failed (${res.status})`);
      err.status = res.status;
      throw err;
    }
    return data;
  }

  // ---------- Toasts ----------
  function toast(message, kind = "info", ms = 4200) {
    const icons = { success: "✓", error: "✕", info: "ℹ" };
    const t = el("div", { class: `toast ${kind}` },
      el("span", { class: "toast-icon", text: icons[kind] || icons.info }),
      el("span", { text: message })
    );
    $("#toasts").append(t);
    setTimeout(() => {
      t.classList.add("leaving");
      setTimeout(() => t.remove(), 220);
    }, ms);
  }

  // ---------- Formatting ----------
  function colorHex(name) {
    const map = {
      red: "#ef4444", blue: "#3b82f6", green: "#22c55e", orange: "#f97316",
      yellow: "#eab308", purple: "#a855f7", pink: "#ec4899", black: "#475569",
      white: "#e2e8f0", maroon: "#9f1239", brown: "#92400e", cyan: "#06b6d4",
      magenta: "#d946ef", teal: "#14b8a6", lime: "#84cc16", navy: "#1e3a8a",
      violet: "#8b5cf6", indigo: "#6366f1", crimson: "#dc2626", gold: "#d4a017",
    };
    if (map[name.toLowerCase()]) return map[name.toLowerCase()];
    let hash = 0;
    for (const ch of String(name)) hash = (hash * 31 + ch.charCodeAt(0)) >>> 0;
    return `hsl(${hash % 360} 60% 45%)`;
  }

  function ownerHue(name) {
    let hash = 0;
    for (const ch of String(name)) hash = (hash * 31 + ch.charCodeAt(0)) >>> 0;
    return hash % 360;
  }

  function initials(name) {
    const parts = String(name).trim().split(/\s+/).slice(0, 2);
    return parts.map((p) => p[0].toUpperCase()).join("") || "?";
  }

  // ---------- Health ----------
  async function checkHealth() {
    const pill = $("#statusPill");
    try {
      const data = await api("/api/health");
      if (!data.ok) throw new Error(data.error);
      state.online = true;
      pill.className = "status-pill online";
      $("#statusText").textContent = "Connected";
      $("#chipChannel").textContent = data.channel;
      $("#chipCC").textContent = data.chaincode;
      $("#offlineBanner").classList.add("hidden");
      return true;
    } catch (err) {
      state.online = false;
      pill.className = "status-pill offline";
      $("#statusText").textContent = "Offline";
      $("#offlineBanner").classList.remove("hidden");
      $("#offlineMsg").textContent =
        "Cannot reach the Fabric network. Start it with: cd fabric-samples/test-network && ./network.sh up createChannel";
      return false;
    }
  }

  // ---------- Data ----------
  async function loadAssets({ silent = false } = {}) {
    try {
      state.assets = await api("/api/assets");
      if (Array.isArray(state.assets)) state.assets = state.assets.map(normalizeAsset);
      $("#offlineBanner").classList.add("hidden");
      render();
    } catch (err) {
      if (!silent) toast(err.message, "error");
      if (err.status === 503 || err.status === undefined) await checkHealth();
    }
  }

  function normalizeAsset(a) {
    return {
      ID: String(a.ID ?? a.id ?? ""),
      Color: String(a.Color ?? a.color ?? ""),
      Size: Number(a.Size ?? a.size ?? 0),
      Owner: String(a.Owner ?? a.owner ?? ""),
      AppraisedValue: Number(a.AppraisedValue ?? a.appraisedValue ?? 0),
    };
  }

  // ---------- Render ----------
  function visibleAssets() {
    const q = state.query.toLowerCase();
    let list = state.assets.filter((a) =>
      !q || a.ID.toLowerCase().includes(q) || a.Owner.toLowerCase().includes(q) || a.Color.toLowerCase().includes(q)
    );
    const bySort = {
      "id": (x, y) => x.ID.localeCompare(y.ID, undefined, { numeric: true }),
      "owner": (x, y) => x.Owner.localeCompare(y.Owner) || x.ID.localeCompare(y.ID),
      "value-asc": (x, y) => x.AppraisedValue - y.AppraisedValue,
      "value-desc": (x, y) => y.AppraisedValue - x.AppraisedValue,
    };
    return [...list].sort(bySort[state.sort] || bySort.id);
  }

  function renderStats() {
    const assets = state.assets;
    const total = assets.length;
    const value = assets.reduce((sum, a) => sum + (a.AppraisedValue || 0), 0);
    const owners = new Set(assets.map((a) => a.Owner)).size;
    $("#statTotal").textContent = total;
    $("#statValue").textContent = money.format(value);
    $("#statOwners").textContent = owners;
    $("#statAvg").textContent = total ? money.format(Math.round(value / total)) : money.format(0);
  }

  function renderTable() {
    const body = $("#assetTableBody");
    body.textContent = "";
    const list = visibleAssets();
    $("#assetCount").textContent = state.assets.length
      ? `${list.length} of ${state.assets.length} assets`
      : "";
    $("#emptyState").classList.toggle("hidden", state.assets.length > 0);
    $("#assetTableBody").closest(".asset-table").classList.toggle("hidden", state.assets.length === 0);

    for (const asset of list) {
      const actions = el("div", { class: "row-actions" },
        el("button", {
          class: "icon-btn accent", title: "View details",
          onclick: () => openDetails(asset),
        }, iconEye()),
        el("button", {
          class: "icon-btn accent", title: "Transfer asset",
          onclick: () => openTransfer(asset),
        }, iconTransfer()),
        el("button", {
          class: "icon-btn danger", title: "Delete asset",
          onclick: () => openDelete(asset),
        }, iconTrash()),
      );

      const row = el("tr", {},
        el("td", {},
          el("button", { class: "asset-id", title: "View details", onclick: () => openDetails(asset), text: asset.ID })),
        el("td", {}, el("span", { class: "color-cell" },
          el("span", { class: "color-swatch", style: `background:${colorHex(asset.Color)}` }),
          el("span", { class: "color-name", text: asset.Color }))),
        el("td", {}, el("span", { class: "mono", text: String(asset.Size) })),
        el("td", {}, el("span", { class: "owner-cell" },
          el("span", { class: "owner-avatar", style: `background:hsl(${ownerHue(asset.Owner)} 55% 42%)`, text: initials(asset.Owner) }),
          el("span", { class: "owner-name", text: asset.Owner }))),
        el("td", { class: "num" }, el("span", { class: "value-cell", text: money.format(asset.AppraisedValue) })),
        el("td", { class: "actions-col" }, actions),
      );
      body.append(row);
    }
  }

  function render() { renderStats(); renderTable(); }

  // Icons
  function iconEye() {
    return svg(`<path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/>`);
  }
  function iconTransfer() {
    return svg(`<polyline points="17 1 21 5 17 9"/><path d="M3 11V9a4 4 0 0 1 4-4h14"/><polyline points="7 23 3 19 7 15"/><path d="M21 13v2a4 4 0 0 1-4 4H3"/>`);
  }
  function iconTrash() {
    return svg(`<polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/><line x1="10" y1="11" x2="10" y2="17"/><line x1="14" y1="11" x2="14" y2="17"/>`);
  }
  function svg(inner, size = 16) {
    const wrap = document.createElement("span");
    wrap.innerHTML = `<svg viewBox="0 0 24 24" width="${size}" height="${size}" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">${inner}</svg>`;
    return wrap;
  }

  // ---------- Dialogs ----------
  function openDialog(id) { $(id).showModal(); }
  function closeDialog(id) { $(id).close(); }
  document.querySelectorAll("dialog [data-close]").forEach((btn) =>
    btn.addEventListener("click", () => btn.closest("dialog").close()));

  function formError(id, message) {
    const box = $(id);
    box.textContent = message;
    box.classList.toggle("hidden", !message);
  }

  // ----- Create -----
  function openCreate() {
    $("#createForm").reset();
    formError("#createError", "");
    openDialog("#createDialog");
    $("#fId").focus();
  }

  $("#createForm").addEventListener("submit", async (e) => {
    e.preventDefault();
    formError("#createError", "");
    const f = e.target.elements;
    const payload = {
      id: f.id.value.trim(),
      color: f.color.value.trim(),
      size: f.size.value === "" ? NaN : Number(f.size.value),
      owner: f.owner.value.trim(),
      appraisedValue: f.appraisedValue.value === "" ? NaN : Number(f.appraisedValue.value),
    };

    if (!payload.id) return formError("#createError", "Asset ID is required.");
    if (!payload.color) return formError("#createError", "Color is required.");
    if (!Number.isInteger(payload.size) || payload.size < 0) return formError("#createError", "Size must be a non-negative whole number.");
    if (!payload.owner) return formError("#createError", "Owner is required.");
    if (!Number.isInteger(payload.appraisedValue) || payload.appraisedValue < 0) return formError("#createError", "Appraised value must be a non-negative whole number.");

    const btn = $("#createSubmitBtn");
    btn.disabled = true;
    try {
      const res = await api("/api/assets", { method: "POST", body: JSON.stringify(payload) });
      closeDialog("#createDialog");
      toast(res.message, "success");
      await loadAssets({ silent: true });
    } catch (err) {
      formError("#createError", friendly(err));
    } finally {
      btn.disabled = false;
    }
  });

  // ----- Details -----
  function openDetails(asset) {
    const grid = $("#detailsGrid");
    grid.textContent = "";
    const item = (label, value, cls = "") =>
      el("div", { class: `detail-item ${cls}` },
        el("div", { class: "detail-label", text: label }),
        el("div", { class: "detail-value", text: value }));

    grid.append(
      item("Asset ID", asset.ID, "full"),
      el("div", { class: "detail-item" },
        el("div", { class: "detail-label", text: "Color" }),
        el("div", { class: "detail-value" },
          el("span", { class: "color-cell" },
            el("span", { class: "color-swatch", style: `background:${colorHex(asset.Color)}` }),
            el("span", { text: asset.Color })))),
      item("Size", String(asset.Size)),
      item("Owner", asset.Owner),
      item("Appraised Value", money.format(asset.AppraisedValue)),
    );
    $("#detailsTransferBtn").onclick = () => {
      closeDialog("#detailsDialog");
      openTransfer(asset);
    };
    openDialog("#detailsDialog");
  }

  // ----- Transfer -----
  function openTransfer(asset) {
    state.transferContext = asset;
    $("#transferAssetId").textContent = asset.ID;
    $("#transferCurrentOwner").textContent = asset.Owner;
    $("#transferOwnerChip").textContent = "";
    $("#transferOwnerChip").append(
      el("span", { class: "owner-avatar", style: `background:hsl(${ownerHue(asset.Owner)} 55% 42%)`, text: initials(asset.Owner) }),
      el("span", { text: asset.Owner }));
    $("#fNewOwner").value = "";
    formError("#transferError", "");
    openDialog("#transferDialog");
    $("#fNewOwner").focus();
  }

  $("#transferForm").addEventListener("submit", async (e) => {
    e.preventDefault();
    formError("#transferError", "");
    const newOwner = $("#fNewOwner").value.trim();
    if (!newOwner) return formError("#transferError", "New owner is required.");
    if (state.transferContext && newOwner === state.transferContext.Owner)
      return formError("#transferError", "The new owner must be different from the current owner.");

    const btn = $("#transferSubmitBtn");
    btn.disabled = true;
    try {
      const res = await api(`/api/assets/${encodeURIComponent(state.transferContext.ID)}/transfer`, {
        method: "POST",
        body: JSON.stringify({ newOwner }),
      });
      closeDialog("#transferDialog");
      const prev = res.previousOwner ? ` (was ${res.previousOwner})` : "";
      toast(`Asset "${state.transferContext.ID}" transferred to ${newOwner}${prev}`, "success");
      await loadAssets({ silent: true });
    } catch (err) {
      formError("#transferError", friendly(err));
    } finally {
      btn.disabled = false;
    }
  });

  // ----- Delete -----
  function openDelete(asset) {
    $("#deleteAssetId").textContent = asset.ID;
    $("#deleteOwner").textContent = asset.Owner;
    openDialog("#deleteDialog");
  }

  $("#deleteConfirmBtn").addEventListener("click", async () => {
    const id = $("#deleteAssetId").textContent;
    const btn = $("#deleteConfirmBtn");
    btn.disabled = true;
    try {
      const res = await api(`/api/assets/${encodeURIComponent(id)}`, { method: "DELETE" });
      closeDialog("#deleteDialog");
      toast(res.message, "success");
      await loadAssets({ silent: true });
    } catch (err) {
      closeDialog("#deleteDialog");
      toast(friendly(err), "error");
    } finally {
      btn.disabled = false;
    }
  });

  // ----- Seed -----
  $("#seedBtn").addEventListener("click", async () => {
    const btn = $("#seedBtn");
    btn.disabled = true;
    try {
      const res = await api("/api/seed", { method: "POST" });
      toast(res.message, "success");
      await loadAssets({ silent: true });
    } catch (err) {
      toast(friendly(err), "error");
    } finally {
      btn.disabled = false;
    }
  });

  // ---------- Misc UI events ----------
  $("#newAssetBtn").addEventListener("click", openCreate);
  $("#emptyNewBtn").addEventListener("click", openCreate);
  $("#refreshBtn").addEventListener("click", () => loadAssets());
  $("#retryBtn").addEventListener("click", async () => {
    if (await checkHealth()) {
      toast("Reconnected to the Fabric network", "success");
      await loadAssets();
    } else {
      toast("Still offline — is the test network running?", "error");
    }
  });
  $("#searchInput").addEventListener("input", (e) => {
    state.query = e.target.value.trim();
    renderTable();
  });
  $("#sortSelect").addEventListener("change", (e) => {
    state.sort = e.target.value;
    renderTable();
  });

  // Auto-refresh every 20s while the tab is visible and no dialog is open
  setInterval(() => {
    if (document.visibilityState !== "visible") return;
    if (document.querySelector("dialog[open]")) return;
    loadAssets({ silent: true });
  }, 20000);

  function friendly(err) {
    if (err.status === 409) return err.message;
    return err.message;
  }

  // ---------- Init ----------
  render();
  (async () => {
    await checkHealth();
    await loadAssets();
  })();
})();