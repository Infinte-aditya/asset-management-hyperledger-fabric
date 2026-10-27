const money = new Intl.NumberFormat("en-US", {
  style: "currency",
  currency: "USD",
  maximumFractionDigits: 0,
});

const weight = (kg) => `${kg.toLocaleString()} kg`;

export function fmtMoney(v) {
  return money.format(v || 0);
}

export function fmtWeight(kg) {
  return kg ? weight(kg) : "—";
}

export function fmtDate(iso) {
  if (!iso) return "—";
  const d = new Date(iso);
  return d.toLocaleDateString("en-US", {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
}

export function fmtDateTime(iso) {
  if (!iso) return "—";
  const d = new Date(iso);
  return d.toLocaleString("en-US", {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export function typeLabel(t) {
  const map = {
    "20GP": "20' Dry",
    "40GP": "40' Dry",
    "40RH": "40' Reefer",
    "20OT": "20' Open Top",
    "40FR": "40' Flat Rack",
  };
  return map[t] || t;
}

export function statusColor(status) {
  const map = {
    Empty: "gray",
    Loaded: "blue",
    InTransit: "indigo",
    AtPort: "teal",
    UnderCustoms: "amber",
    Damaged: "red",
    Repairing: "orange",
  };
  return map[status] || "gray";
}

export function customsBadge(s) {
  const map = {
    Clear: "green",
    Hold: "red",
    ExamRequired: "amber",
    Release: "green",
  };
  return map[s] || "gray";
}

export function initials(name) {
  const parts = String(name).trim().split(/\s+/).slice(0, 2);
  return parts.map((p) => p[0].toUpperCase()).join("") || "?";
}

export function ownerHue(name) {
  let hash = 0;
  for (const ch of String(name)) hash = (hash * 31 + ch.charCodeAt(0)) >>> 0;
  return hash % 360;
}

export function containerTypeIcon(type) {
  switch (type) {
    case "40RH":
      return `<svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="4" width="20" height="16" rx="2"/><line x1="6" y1="10" x2="6" y2="14"/><circle cx="9" cy="12" r="1" fill="#60a5fa"/><circle cx="12" cy="12" r="1" fill="#60a5fa"/><circle cx="15" cy="12" r="1" fill="#60a5fa"/></svg>`;
    case "20OT":
      return `<svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18v12H3z"/><line x1="3" y1="6" x2="3" y2="18"/><line x1="21" y1="6" x2="21" y2="18"/><line x1="9" y1="6" x2="9" y2="18"/><line x1="15" y1="6" x2="15" y2="18"/></svg>`;
    case "40FR":
      return `<svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="3" width="20" height="18" rx="2"/><line x1="6" y1="7" x2="6" y2="17"/><line x1="10" y1="7" x2="10" y2="17"/><line x1="14" y1="7" x2="14" y2="17"/><line x1="18" y1="7" x2="18" y2="17"/></svg>`;
    default:
      return `<svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="4" width="18" height="16" rx="1"/><line x1="3" y1="10" x2="21" y2="10"/><line x1="12" y1="4" x2="12" y2="20"/></svg>`;
  }
}

export function eventIcon(action) {
  const map = {
    GateIn: `<circle cx="12" cy="12" r="4"/><path d="M12 8v8"/><path d="M8 12h8"/>`,
    GateOut: `<circle cx="12" cy="12" r="4"/><path d="M12 8v8"/><path d="M8 12h8"/>`,
    Loaded: `<path d="M3 15h18M3 15l3-9h12l3 9M3 15v4h18v-4"/><path d="M8 15v-2a4 4 0 0 1 8 0v2"/>`,
    Discharged: `<path d="M3 15h18M3 15l3-9h12l3 9M3 15v4h18v-4"/><path d="M8 15v2a4 4 0 0 0 8 0v-2"/>`,
    DamageReported: `<path d="M12 9v4"/><circle cx="12" cy="20" r="1"/><path d="M12 3v1"/><path d="M18.36 5.64l-.7.7"/><path d="M5.64 5.64l.7.7"/><path d="M4 12h1"/><path d="M19 12h1"/>`,
    Repaired: `<path d="M22 12h-4l-3 9L9 3l-3 9H2"/>`,
    CustomsHold: `<rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/>`,
    "Customs:Clear": `<path d="M9 12l2 2 4-4"/><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/>`,
    Transferred: `<polyline points="17 1 21 5 17 9"/><path d="M3 11V9a4 4 0 0 1 4-4h14"/><polyline points="7 23 3 19 7 15"/><path d="M21 13v2a4 4 0 0 1-4 4H3"/>`,
    Registered: `<circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="16"/><line x1="8" y1="12" x2="16" y2="12"/>`,
  };
  return map[action] || `<circle cx="12" cy="12" r="4"/>`;
}