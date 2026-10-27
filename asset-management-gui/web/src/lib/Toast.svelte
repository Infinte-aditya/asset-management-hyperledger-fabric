<script>
  let toasts = $state([]);
  let idCounter = 0;

  function add(m, k, ms) {
    const id = ++idCounter;
    toasts = [...toasts, { id, message: m, kind: k || "info" }];
    setTimeout(() => {
      toasts = toasts.filter((t) => t.id !== id);
    }, ms || 4200);
  }

  window.__toast = add;
</script>

<div id="toastContainer" class="toasts">
  {#each toasts as t (t.id)}
    <div class="toast {t.kind}">
      <span class="toast-icon">{t.kind === "success" ? "✓" : t.kind === "error" ? "✕" : "ℹ"}</span>
      <span>{t.message}</span>
    </div>
  {/each}
</div>