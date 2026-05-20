const TETRA_UI_URL = new URL("ui_web_smoke.ui.json", import.meta.url);

function addLine(root, text) {
  const line = document.createElement("div");
  line.textContent = text;
  root.appendChild(line);
}

export async function mountTetraUI(root = document.body, uiURL = TETRA_UI_URL) {
  const response = await fetch(uiURL);
  if (!response.ok) {
    throw new Error("tetra_ui: failed to fetch metadata: " + response.status);
  }
  const bundle = await response.json();
  if (!bundle || bundle.schema !== "tetra.ui.v1") {
    throw new Error("tetra_ui: unsupported schema: " + String(bundle && bundle.schema));
  }
  const host = document.createElement("section");
  host.setAttribute("data-tetra-ui", "v1");
  addLine(host, "Tetra UI Shell");
  addLine(host, "runtime: metadata-only preview (no event dispatch)");
  for (const view of (bundle.views || [])) {
    addLine(host, "view " + view.name + " (state: " + view.state_type + ")");
    for (const binding of (view.bindings || [])) {
      addLine(host, "  bind " + binding.name + ": " + binding.type + " <- " + binding.source);
    }
    for (const event of (view.events || [])) {
      addLine(host, "  event " + event.name + " -> " + event.command);
    }
  }
  root.appendChild(host);
  return bundle;
}
