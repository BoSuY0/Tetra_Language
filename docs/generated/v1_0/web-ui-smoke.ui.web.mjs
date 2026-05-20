const TETRA_UI_URL = new URL("web_smoke.ui.json", import.meta.url);

function addLine(root, text) {
  const line = document.createElement("div");
  line.textContent = text;
  root.appendChild(line);
}

function parseInit(field) {
  if (field.type === "i32" || field.type === "u8" || field.type === "u16") {
    return Number.parseInt(field.init, 10) || 0;
  }
  if (field.type === "bool") {
    return field.init === "true";
  }
  if (field.type === "str" && field.init.length >= 2) {
    return field.init.slice(1, -1);
  }
  return field.init;
}

function initialState(bundle) {
  const state = {};
  for (const group of (bundle.states || [])) {
    const fields = {};
    for (const field of (group.fields || [])) {
      fields[field.name] = parseInit(field);
    }
    state[group.name] = fields;
  }
  return state;
}

function statePath(path) {
  const parts = String(path || "").split(".");
  if (parts.length !== 2 || parts[0] !== "state") {
    throw new Error("tetra_ui: unsupported state path: " + path);
  }
  return parts[1];
}

function stateForView(state, view) {
  return state[view.state_type] || {};
}

function applyTetraCommand(state, view, command) {
  const viewState = stateForView(state, view);
  for (const op of (command.operations || [])) {
    const field = statePath(op.target);
    switch (op.kind) {
    case "state_add":
      viewState[field] = (Number(viewState[field]) || 0) + (Number(op.value) || 0);
      break;
    case "state_set":
      viewState[field] = op.value;
      break;
    default:
      throw new Error("tetra_ui: unsupported command operation: " + op.kind);
    }
  }
}

function bindingValue(state, view, binding) {
  const source = String(binding.source || "");
  if (source.startsWith("state.")) {
    return String(stateForView(state, view)[statePath(source)]);
  }
  return source;
}

function renderBindings(host, state, view) {
  for (const node of host.querySelectorAll("[data-tetra-binding]")) {
    const binding = (view.bindings || []).find((item) => item.name === node.getAttribute("data-tetra-binding"));
    if (binding) {
      node.textContent = "  bind " + binding.name + ": " + binding.type + " = " + bindingValue(state, view, binding);
    }
  }
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
  const state = initialState(bundle);
  addLine(host, "Tetra UI Shell");
  addLine(host, "runtime: web command dispatch");
  for (const view of (bundle.views || [])) {
    addLine(host, "view " + view.name + " (state: " + view.state_type + ")");
    for (const binding of (view.bindings || [])) {
      const line = document.createElement("div");
      line.setAttribute("data-tetra-binding", binding.name);
      line.textContent = "  bind " + binding.name + ": " + binding.type + " = " + bindingValue(state, view, binding);
      host.appendChild(line);
    }
    for (const event of (view.events || [])) {
      const button = document.createElement("button");
      button.type = "button";
      button.textContent = "event " + event.name + " -> " + event.command;
      button.addEventListener("click", () => {
        const command = (view.commands || []).find((item) => item.name === event.command);
        if (!command) {
          throw new Error("tetra_ui: unknown command: " + event.command);
        }
        applyTetraCommand(state, view, command);
        renderBindings(host, state, view);
      });
      host.appendChild(button);
    }
  }
  root.appendChild(host);
  return bundle;
}
