const app = document.getElementById("app");
const defaultScreens = [
  "Dashboard",
  "Profiles",
  "Fans/Backends",
  "Diagnostics",
  "Logs",
  "Settings",
];
const defaultProfiles = ["quiet", "balanced", "performance", "custom"];
const requestedScreen = new URLSearchParams(window.location.search).get("screen");

const state = {
  activeScreen: "Dashboard",
  selectedProfile: "balanced",
  dryRun: true,
  snapshot: null,
  tetraBundle: null,
  loading: true,
  error: "",
  lastProfileResult: null,
};

function text(value, fallback = "unavailable") {
  if (value === null || value === undefined || value === "") return fallback;
  return String(value);
}

function parseTetraInit(value) {
  const raw = String(value || "");
  if (raw.length >= 2 && raw.startsWith('"') && raw.endsWith('"')) return raw.slice(1, -1);
  return raw;
}

function tetraFields() {
  return state.tetraBundle?.states?.[0]?.fields || [];
}

function tetraContract() {
  const fields = tetraFields();
  const screens = fields
    .filter((field) => field.name.startsWith("screen"))
    .map((field) => parseTetraInit(field.init))
    .filter(Boolean);
  const profiles = fields
    .filter((field) => field.name.startsWith("profile"))
    .map((field) => parseTetraInit(field.init).toLowerCase())
    .filter(Boolean);
  return {
    screens: screens.length ? screens : defaultScreens,
    profiles: profiles.length ? profiles : defaultProfiles,
  };
}

function percent(value) {
  const n = Number(value);
  if (!Number.isFinite(n)) return 0;
  return Math.max(0, Math.min(100, n));
}

function statusClass(status) {
  const normalized = String(status || "unknown").toLowerCase();
  if (normalized === "supported" || normalized === "ok" || normalized === "applied") return "good";
  if (normalized === "dry-run" || normalized === "partial") return "warn";
  return "bad";
}

function el(tag, attrs = {}, children = []) {
  const node = document.createElement(tag);
  for (const [key, value] of Object.entries(attrs)) {
    if (key === "class") node.className = value;
    else if (key === "text") node.textContent = value;
    else if (key.startsWith("on") && typeof value === "function")
      node.addEventListener(key.slice(2), value);
    else if (value !== false && value !== null && value !== undefined)
      node.setAttribute(key, String(value));
  }
  for (const child of Array.isArray(children) ? children : [children]) {
    if (child === null || child === undefined) continue;
    node.appendChild(typeof child === "string" ? document.createTextNode(child) : child);
  }
  return node;
}

function pill(label, status) {
  return el("span", { class: `pill ${statusClass(status)}`, text: label });
}

function metric(title, value, detail = "", level = null) {
  const children = [
    el("div", { class: "metric-title", text: title }),
    el("div", { class: "metric-value", text: text(value) }),
  ];
  if (detail) children.push(el("div", { class: "metric-detail", text: detail }));
  if (level !== null) {
    const bar = el("div", { class: "meter" }, el("span", { style: `width:${percent(level)}%` }));
    children.push(bar);
  }
  return el("article", { class: "metric" }, children);
}

function table(rows) {
  const body = el(
    "tbody",
    {},
    rows.map(([key, value]) =>
      el("tr", {}, [el("th", { text: key }), el("td", { text: text(value) })]),
    ),
  );
  return el("table", { class: "kv" }, body);
}

async function api(path, options = {}) {
  const response = await fetch(path, {
    headers: { "Content-Type": "application/json" },
    ...options,
  });
  const payload = await response.json();
  if (!response.ok) {
    throw new Error(payload.reason || payload.status || `HTTP ${response.status}`);
  }
  return payload;
}

async function refreshSnapshot() {
  state.error = "";
  try {
    state.snapshot = await api("/api/snapshot");
  } catch (err) {
    state.error = String(err.message || err);
  }
  render();
}

async function loadTetraBundle() {
  try {
    const response = await fetch("/build/tetra_control_center.ui.json", { cache: "no-store" });
    if (!response.ok) throw new Error(`HTTP ${response.status}`);
    state.tetraBundle = await response.json();
    if (tetraContract().screens.includes(requestedScreen)) {
      state.activeScreen = requestedScreen;
    }
  } catch (err) {
    state.tetraBundle = { schema: "missing", error: String(err.message || err) };
  }
}

async function applyProfile(profile) {
  state.selectedProfile = profile;
  try {
    state.lastProfileResult = await api("/api/profile", {
      method: "POST",
      body: JSON.stringify({ profile, dry_run: state.dryRun }),
    });
    await refreshSnapshot();
  } catch (err) {
    state.lastProfileResult = { status: "rejected", reason: String(err.message || err) };
    render();
  }
}

function header(snapshot) {
  const current = snapshot?.power?.profile?.current || "unavailable";
  const mode = snapshot?.settings?.allow_writes ? "allow-writes" : "read-only";
  return el("header", { class: "topbar" }, [
    el("div", {}, [
      el("h1", { text: "Tetra Control Center" }),
      el("p", { text: "DREAM MACHINES V3xxSNP_SNN_SNM" }),
    ]),
    el("div", { class: "top-actions" }, [
      pill(`profile ${current}`, snapshot?.power?.profile?.status),
      pill(mode, mode === "read-only" ? "dry-run" : "supported"),
      el(
        "button",
        {
          class: "icon-button",
          title: "Refresh",
          "aria-label": "Refresh",
          onclick: refreshSnapshot,
        },
        [el("span", { text: "R" })],
      ),
    ]),
  ]);
}

function nav() {
  const { screens } = tetraContract();
  return el(
    "nav",
    { class: "side-nav" },
    screens.map((screen) =>
      el("button", {
        class: state.activeScreen === screen ? "active" : "",
        text: screen,
        onclick: () => {
          state.activeScreen = screen;
          render();
        },
      }),
    ),
  );
}

function dashboard(snapshot) {
  const cpu = snapshot?.cpu || {};
  const memory = snapshot?.memory || {};
  const battery = snapshot?.battery?.[0] || {};
  const nvidia = snapshot?.gpu?.nvidia || {};
  const gpu = nvidia.gpus?.[0] || {};
  const sensors = snapshot?.sensors?.hwmon || [];
  const fanCount = sensors.flatMap((item) => item.fans || []).length;
  const tempCount = sensors.flatMap((item) => item.temperatures || []).length;
  const support = snapshot?.dashboard?.driver_support || [];
  return section("Dashboard", [
    el("div", { class: "metrics-grid" }, [
      metric(
        "CPU",
        cpu.governors?.join(", ") || cpu.model,
        `EPP: ${(cpu.epp || []).join(", ") || "unavailable"}`,
      ),
      metric("GPU", gpu.name || nvidia.status, `driver ${text(gpu.driver_version)}`),
      metric(
        "RAM",
        memory.used_percent !== null ? `${memory.used_percent}% used` : "unavailable",
        `${text(memory.available_kb)} kB available`,
        memory.used_percent,
      ),
      metric(
        "Battery",
        battery.capacity ? `${battery.capacity}%` : "unavailable",
        text(battery.status),
        battery.capacity,
      ),
      metric("Power", snapshot?.power?.profile?.current, snapshot?.power?.profile?.reason),
      metric("Sensors", `${tempCount} temp / ${fanCount} fan`, "read-only hwmon discovery"),
    ]),
    el(
      "div",
      { class: "support-strip" },
      support.map((item) =>
        el("div", { class: "support-item" }, [
          pill(item.name, item.status),
          el("span", { text: item.reason }),
        ]),
      ),
    ),
  ]);
}

function profilesScreen(snapshot) {
  const current = snapshot?.profiles?.current || "unavailable";
  const { profiles } = tetraContract();
  return section("Profiles", [
    el(
      "div",
      { class: "profile-grid" },
      profiles.map((profile) =>
        el(
          "button",
          {
            class: `profile-tile ${state.selectedProfile === profile ? "selected" : ""}`,
            onclick: () => applyProfile(profile),
          },
          [
            el("strong", { text: profile[0].toUpperCase() + profile.slice(1) }),
            el("span", { text: profileDescription(profile) }),
          ],
        ),
      ),
    ),
    el("label", { class: "toggle" }, [
      el("input", {
        type: "checkbox",
        checked: state.dryRun ? "checked" : null,
        onchange: (event) => {
          state.dryRun = event.target.checked;
          render();
        },
      }),
      el("span", { text: "Dry-run" }),
    ]),
    el("div", { class: "result-line" }, [
      pill(`current ${current}`, snapshot?.power?.profile?.status),
      state.lastProfileResult
        ? pill(state.lastProfileResult.status, state.lastProfileResult.status)
        : null,
      state.lastProfileResult ? el("span", { text: state.lastProfileResult.reason }) : null,
    ]),
  ]);
}

function profileDescription(profile) {
  return {
    quiet: "power-saver, powersave governor, power EPP",
    balanced: "balanced profile, powersave governor, balance EPP",
    performance: "performance profile and performance EPP",
    custom: "no write operation; reserved for manual policy",
  }[profile];
}

function fansScreen(snapshot) {
  const fans = snapshot?.fans?.control || {};
  const rpm = fans.rpm_sensors || [];
  const nbfc = snapshot?.diagnostics?.nbfc || {};
  const tuxedo = snapshot?.diagnostics?.tuxedo || {};
  return section("Fans/Backends", [
    el("div", { class: "result-line" }, [
      pill(fans.status || "unknown", fans.status),
      el("span", { text: fans.reason }),
    ]),
    el("div", { class: "metrics-grid compact" }, [
      metric("RPM sensors", rpm.length, "hwmon read-only"),
      metric("NBFC-Linux", nbfc.status, nbfc.reason),
      metric("TUXEDO/DKMS/TCC", tuxedo.status, tuxedo.reason),
    ]),
    table(
      rpm.length
        ? rpm.map((item) => [item.device, `${text(item.rpm)} RPM`])
        : [["fan control", "unsupported until a validated backend exists"]],
    ),
  ]);
}

function diagnosticsScreen(snapshot) {
  const diag = snapshot?.diagnostics || {};
  const dmi = diag.dmi || {};
  const modules = diag.kernel_modules || {};
  const sysfs = diag.sysfs_capabilities || {};
  return section("Diagnostics", [
    el("div", { class: "split" }, [
      el("div", {}, [el("h3", { text: "DMI" }), table(Object.entries(dmi))]),
      el("div", {}, [
        el("h3", { text: "Kernel modules" }),
        table(
          Object.entries(modules).map(([key, value]) => [key, value ? "loaded" : "not loaded"]),
        ),
      ]),
    ]),
    el("h3", { text: "Sysfs capabilities" }),
    table([
      ["cpufreq policies", (sysfs.cpufreq_policies || []).length],
      ["hwmon devices", (sysfs.hwmon_devices || []).length],
    ]),
    el("h3", { text: "Support matrix" }),
    el(
      "div",
      { class: "support-strip" },
      (diag.support || []).map((item) =>
        el("div", { class: "support-item" }, [
          pill(item.name, item.status),
          el("span", { text: item.reason }),
        ]),
      ),
    ),
  ]);
}

function logsScreen(snapshot) {
  const logs = snapshot?.logs || [];
  return section("Logs", [
    el("div", { class: "result-line" }, [
      el("button", { text: "Refresh logs", onclick: refreshSnapshot }),
      el("span", { text: `${logs.length} audit entries loaded` }),
    ]),
    el(
      "div",
      { class: "log-list" },
      logs.length
        ? logs
            .slice()
            .reverse()
            .map((entry) => el("pre", { text: JSON.stringify(entry, null, 2) }))
        : [el("p", { class: "muted", text: "No audit entries yet." })],
    ),
  ]);
}

function settingsScreen(snapshot) {
  const settings = snapshot?.settings || {};
  const bundle = state.tetraBundle || {};
  return section("Settings", [
    table([
      ["helper mode", settings.allow_writes ? "allow-writes" : "read-only"],
      ["dry-run default", text(settings.dry_run_default)],
      ["audit log", settings.audit_log],
      ["Tetra UI schema", bundle.schema || "missing"],
      [
        "Tetra views",
        bundle.views ? bundle.views.map((view) => view.name).join(", ") : text(bundle.error),
      ],
    ]),
  ]);
}

function section(title, children) {
  return el("section", { class: "screen" }, [
    el("div", { class: "screen-title" }, [
      el("h2", { text: title }),
      el("span", { text: state.snapshot?.app?.generated_at || "" }),
    ]),
    ...children,
  ]);
}

function mainContent(snapshot) {
  if (state.loading) return section("Loading", [el("p", { text: "Loading snapshot." })]);
  if (state.error)
    return section("Backend unavailable", [el("p", { class: "error", text: state.error })]);
  const renderers = {
    Dashboard: dashboard,
    Profiles: profilesScreen,
    "Fans/Backends": fansScreen,
    Diagnostics: diagnosticsScreen,
    Logs: logsScreen,
    Settings: settingsScreen,
  };
  return renderers[state.activeScreen](snapshot);
}

function render() {
  app.replaceChildren(
    header(state.snapshot),
    el("div", { class: "workspace" }, [nav(), mainContent(state.snapshot)]),
  );
}

async function start() {
  render();
  await Promise.all([loadTetraBundle(), refreshSnapshot()]);
  state.loading = false;
  render();
  window.setInterval(refreshSnapshot, 15000);
}

start();
