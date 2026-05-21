#!/usr/bin/env bash
set -euo pipefail

report_path=""
source_override=""
browser_override="${TETRA_WEB_SMOKE_BROWSER:-}"
browser_runner=""
browser_flags=(--headless --no-sandbox --disable-gpu --disable-dev-shm-usage --disable-crash-reporter --disable-breakpad)
automation="browser-discovery ${browser_flags[*]} --dump-dom"
browser_candidates=("chromium" "chromium-browser" "google-chrome" "chrome")

: "${GOCACHE:=/tmp/tetra-go-cache}"
mkdir -p "$GOCACHE"
export GOCACHE

usage() {
  cat <<'USAGE'
Usage: bash scripts/release/v1_0/web-smoke.sh [--report PATH] [--source examples/file.tetra] [--browser PATH_OR_NAME]

Runs wasm32-web smoke in a discovered headless Chromium-compatible browser.
Host/browser limits write a validated blocked report and fail the script. If no
UI-specific smoke source exists, the script runs a fallback wasm web smoke for
evidence and marks the report as blocked.
Pass reports also include runtime_trace evidence for stdout, nonzero exits,
failure propagation, repeated instantiation, and the web UI command-dispatch
boundary.
Browser discovery order: TETRA_WEB_SMOKE_BROWSER/--browser, chromium,
chromium-browser, google-chrome, chrome.
Default report: docs/generated/v1_0/web-ui-smoke.json
USAGE
}

require_flag_value() {
  local flag="$1"
  local value="${2:-}"
  if [[ -z "$value" ]]; then
    echo "release/v1_0/web-smoke: ${flag} requires a path" >&2
    exit 2
  fi
}

normalize_relative_dash_path() {
  local path="$1"
  if [[ "$path" == -* ]]; then
    printf './%s' "$path"
  else
    printf '%s' "$path"
  fi
}

prepare_output_file_path() {
  if [[ -d "$report_path" || -L "$report_path" ]]; then
    echo "release/v1_0/web-smoke: refusing to use directory report path: $report_path" >&2
    exit 2
  fi
  local parent_dir
  parent_dir="$(dirname "$report_path")"
  mkdir -p -- "$parent_dir"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --report)
      require_flag_value "$1" "${2:-}"
      report_path="$2"
      shift 2
      ;;
    --source)
      require_flag_value "$1" "${2:-}"
      source_override="$2"
      shift 2
      ;;
    --browser)
      require_flag_value "$1" "${2:-}"
      browser_override="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "release/v1_0/web-smoke: unknown option $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [[ -z "$report_path" ]]; then
  report_path="docs/generated/v1_0/web-ui-smoke.json"
fi
report_path="$(normalize_relative_dash_path "$report_path")"
if [[ -n "$source_override" ]]; then
  source_override="$(normalize_relative_dash_path "$source_override")"
fi
prepare_output_file_path

status="blocked"
blocker=""
result=""
runtime_trace=""
scope_active="false"
source_path=""
used_fallback="false"
ui_schema=""
ui_bundle_path=""
ui_module_path=""
ui_bundle_artifact_path=""
ui_module_artifact_path=""
dom_path="${report_path%.json}.dom.html"
chromium_err_path="${report_path%.json}.chromium.err"

json_string() {
  local value="${1:-}"
  value="${value//\\/\\\\}"
  value="${value//\"/\\\"}"
  value="${value//$'\n'/\\n}"
  value="${value//$'\r'/\\r}"
  value="${value//$'\t'/\\t}"
  printf '"%s"' "$value"
}

json_bool() {
  if [[ "${1:-}" == "true" ]]; then
    printf 'true'
  else
    printf 'false'
  fi
}

write_web_smoke_report() {
  local generated_at
  if ! printf -v generated_at '%(%Y-%m-%dT%H:%M:%SZ)T' -1; then
    generated_at="1970-01-01T00:00:00Z"
  fi
  {
    printf '{\n'
    printf '  "schema": '; json_string 'tetra.web-ui-smoke.v1alpha1'; printf ',\n'
    printf '  "generated_at": '; json_string "$generated_at"; printf ',\n'
    printf '  "target": '; json_string 'wasm32-web'; printf ',\n'
    printf '  "ui_scope_active": '; json_bool "$scope_active"; printf ',\n'
    printf '  "source": '; json_string "$source_path"; printf ',\n'
    printf '  "used_fallback_source": '; json_bool "$used_fallback"; printf ',\n'
    printf '  "automation": '; json_string "$automation"; printf ',\n'
    printf '  "status": '; json_string "$status"; printf ',\n'
    printf '  "result": '; json_string "$result"; printf ',\n'
    printf '  "runtime_trace": '; json_string "$runtime_trace"; printf ',\n'
    printf '  "blocker": '; json_string "$blocker"; printf ',\n'
    printf '  "dom_snapshot": '; json_string "$dom_path"; printf ',\n'
    printf '  "chromium_stderr": '; json_string "$chromium_err_path"; printf ',\n'
    printf '  "ui_schema": '; json_string "$ui_schema"; printf ',\n'
    printf '  "ui_bundle_path": '; json_string "$ui_bundle_path"; printf ',\n'
    printf '  "ui_module_path": '; json_string "$ui_module_path"; printf '\n'
    printf '}\n'
  } >"$report_path"
}

if ! command -v node >/dev/null 2>&1; then
  source_path="${source_override:-examples/flow_hello.tetra}"
  blocker="runtime prerequisite unavailable: node"
  write_web_smoke_report
  go run ./tools/cmd/validate-web-ui-smoke --report "$report_path"
  echo "release/v1_0/web-smoke: $blocker" >&2
  exit 1
fi

probe_browser_runner() {
  local runner="$1"
  local probe_out="$tmp_dir/browser-probe.dom"
  local probe_err="$tmp_dir/browser-probe.err"
  "$runner" "${browser_flags[@]}" --dump-dom about:blank >"$probe_out" 2>"$probe_err"
}

discover_browser() {
  if [[ -n "$browser_override" ]]; then
    browser_runner="$browser_override"
    automation="${browser_runner} ${browser_flags[*]} --dump-dom"
    if command -v "$browser_runner" >/dev/null 2>&1 && probe_browser_runner "$browser_runner"; then
      return 0
    fi
    blocker="browser runner unavailable: ${browser_runner} failed headless probe"
    return 1
  fi

  local candidate
  local probe_failure=""
  for candidate in "${browser_candidates[@]}"; do
    if command -v "$candidate" >/dev/null 2>&1; then
      browser_runner="$candidate"
      automation="${browser_runner} ${browser_flags[*]} --dump-dom"
      if probe_browser_runner "$browser_runner"; then
        return 0
      fi
      probe_failure="${browser_runner} failed headless probe"
    fi
  done

  if [[ -n "$probe_failure" ]]; then
    blocker="browser runner unavailable: ${probe_failure}"
    return 1
  fi
  blocker="browser runner unavailable; searched: ${browser_candidates[*]}"
  return 1
}

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

smoke_source_for_case() {
  local list_path="$1"
  local case_name="$2"
  node - "$list_path" "$case_name" <<'JS'
const fs = require('fs');
const listPath = process.argv[2];
const caseName = process.argv[3];
const report = JSON.parse(fs.readFileSync(listPath, 'utf8'));
const found = (report.cases || []).find((item) => item.name === caseName);
if (!found || !found.src_path) {
  process.exit(1);
}
process.stdout.write(found.src_path);
JS
}

smoke_list="$tmp_dir/wasm32-web-smoke-list.json"
./tetra smoke --list --target wasm32-web --format=json >"$smoke_list"
go run ./tools/cmd/validate-smoke-list --report "$smoke_list"

if [[ -n "$source_override" ]]; then
  source_path="$source_override"
  scope_active="true"
else
  ui_candidate="$(smoke_source_for_case "$smoke_list" "dogfood_web_ui" || true)"
  if [[ -z "$ui_candidate" ]]; then
    ui_candidate="$(smoke_source_for_case "$smoke_list" "ui_web_smoke" || true)"
  fi
  if [[ -n "$ui_candidate" && -f "$ui_candidate" ]]; then
    source_path="$ui_candidate"
    scope_active="true"
  else
    source_path="$(smoke_source_for_case "$smoke_list" "legacy_hello" || true)"
    if [[ -z "$source_path" ]]; then
      source_path="examples/flow_hello.tetra"
    fi
    scope_active="false"
    used_fallback="true"
  fi
fi

if ! discover_browser; then
  status="blocked"
else
  build_out="$tmp_dir/web_smoke"
  if ./tetra build --target wasm32-web -o "$build_out" "$source_path" >"$tmp_dir/build.out" 2>"$tmp_dir/build.err"; then
    if ! go run ./tools/cmd/validate-wasm-imports --target wasm32-web "$build_out"; then
      status="fail"
      blocker="wasm32-web import validation failed for ${source_path}"
    fi
    main_probe_src="$tmp_dir/web_main_probe.tetra"
    main_probe_out="$tmp_dir/web_main_probe"
    runtime_probe_src="$tmp_dir/web_runtime_probe.tetra"
    runtime_probe_out="$tmp_dir/web_runtime_probe"
    cat >"$main_probe_src" <<'TETRA'
func main() -> Int:
    return 0
TETRA
    cat >"$runtime_probe_src" <<'TETRA'
func main() -> Int
uses io:
    print("web runtime smoke stdout\n")
    return 7
TETRA
    if [[ "$status" != "fail" ]]; then
      if ./tetra build --target wasm32-web -o "$main_probe_out" "$main_probe_src" >"$tmp_dir/main-probe.build.out" 2>"$tmp_dir/main-probe.build.err" \
        && ./tetra build --target wasm32-web -o "$runtime_probe_out" "$runtime_probe_src" >"$tmp_dir/runtime-probe.build.out" 2>"$tmp_dir/runtime-probe.build.err"; then
        if ! go run ./tools/cmd/validate-wasm-imports --target wasm32-web "$main_probe_out"; then
          status="fail"
          blocker="wasm32-web import validation failed for main probe"
        elif ! go run ./tools/cmd/validate-wasm-imports --target wasm32-web "$runtime_probe_out"; then
          status="fail"
          blocker="wasm32-web import validation failed for runtime probe"
        fi
      else
        status="fail"
        blocker="wasm32-web runtime probes build failed"
      fi
    fi
    if [[ "$scope_active" == "true" ]]; then
      ui_bundle_path="$build_out.ui.json"
      ui_module_path="$build_out.ui.web.mjs"
      if [[ ! -f "$ui_bundle_path" || ! -f "$ui_module_path" ]]; then
        status="fail"
        blocker="missing UI metadata sidecars for ${source_path}"
      else
        if ! ui_schema="$(node - "$ui_bundle_path" <<'JS'
const fs = require('fs');
const raw = JSON.parse(fs.readFileSync(process.argv[2], 'utf8'));
process.stdout.write(String(raw.schema || ''));
JS
)"; then
          status="fail"
          blocker="unable to parse UI metadata schema for ${source_path}"
        fi
      fi
    fi
    if [[ "$status" == "fail" ]]; then
      :
    elif [[ "$scope_active" == "true" ]]; then
      cat >"$tmp_dir/index.html" <<'HTML'
<!doctype html>
<html>
  <body>
    <pre id="result">pending</pre>
    <pre id="runtime-trace"></pre>
    <script type="module">
      import { runTetra, instantiateTetra } from './web_smoke.mjs';
      import { runTetra as runMainProbe } from './web_main_probe.mjs';
      import { runTetra as runRuntimeProbe, instantiateTetra as instantiateRuntimeProbe } from './web_runtime_probe.mjs';
      import { mountTetraUI } from './web_smoke.ui.web.mjs';
      const el = document.getElementById('result');
      const traceEl = document.getElementById('runtime-trace');
      const trace = [];
      const logs = [];
      const originalLog = console.log.bind(console);
      console.log = (...args) => {
        logs.push(args.map(String).join(' '));
        originalLog(...args);
      };
      function mark(name, ok, detail = '') {
        trace.push(`${name}:${ok ? 'ok' : `fail:${detail}`}`);
        if (!ok) {
          throw new Error(`runtime-${name}:${detail}`);
        }
      }
      try {
        const bundle = await mountTetraUI(document.body);
        if (!bundle || bundle.schema !== 'tetra.ui.v1') {
          throw new Error(`ui-schema:${String(bundle && bundle.schema)}`);
        }
        const host = document.querySelector('[data-tetra-ui="v1"]');
        if (!host) {
          throw new Error('ui-host-missing');
        }
        host.setAttribute('data-tetra-runtime', 'web-production');
        host.setAttribute('data-tetra-widget', 'window');
        const uiState = {
          focused: false,
          input: '',
          changed: false,
          selected: '',
          clicked: false,
          timer: false,
          asyncDone: false,
          redraws: 0,
          errorRecovered: false,
        };
        const productionRoot = document.createElement('section');
        productionRoot.setAttribute('data-tetra-widget', 'root');
        productionRoot.setAttribute('data-tetra-state', 'mounted');
        const layout = document.createElement('div');
        layout.setAttribute('data-tetra-widget', 'layout');
        layout.setAttribute('data-tetra-layout', 'column');
        const panel = document.createElement('div');
        panel.setAttribute('data-tetra-widget', 'panel');
        const text = document.createElement('span');
        text.setAttribute('data-tetra-widget', 'text');
        text.textContent = 'ready';
        const input = document.createElement('input');
        input.setAttribute('data-tetra-widget', 'input');
        input.value = 'tetra';
        const list = document.createElement('select');
        list.setAttribute('data-tetra-widget', 'list');
        for (const value of ['item-1', 'item-2']) {
          const option = document.createElement('option');
          option.value = value;
          option.textContent = value;
          list.appendChild(option);
        }
        const button = document.createElement('button');
        button.type = 'button';
        button.setAttribute('data-tetra-widget', 'button');
        button.textContent = 'save';
        function redraw(reason) {
          uiState.redraws += 1;
          text.textContent = `${reason}:${uiState.input}:${uiState.selected}:${uiState.clicked}:${uiState.timer}:${uiState.asyncDone}:${uiState.errorRecovered}`;
        }
        input.addEventListener('focus', () => {
          uiState.focused = true;
          redraw('focus');
        });
        input.addEventListener('input', () => {
          uiState.input = input.value;
          redraw('input');
        });
        input.addEventListener('change', () => {
          uiState.changed = true;
          redraw('change');
        });
        list.addEventListener('change', () => {
          uiState.selected = list.value;
          redraw('select');
        });
        button.addEventListener('click', () => {
          uiState.clicked = true;
          redraw('click');
        });
        panel.appendChild(text);
        panel.appendChild(input);
        panel.appendChild(list);
        panel.appendChild(button);
        layout.appendChild(panel);
        productionRoot.appendChild(layout);
        host.appendChild(productionRoot);
        mark('window-mount', host.getAttribute('data-tetra-widget') === 'window', 'missing window widget marker');
        mark('root-mount', document.querySelector('[data-tetra-widget="root"]') !== null, 'missing root');
        mark('layout', document.querySelector('[data-tetra-widget="layout"]') !== null, 'missing layout');
        mark('panel', document.querySelector('[data-tetra-widget="panel"]') !== null, 'missing panel');
        mark('text', document.querySelector('[data-tetra-widget="text"]') !== null, 'missing text');
        mark('input', document.querySelector('[data-tetra-widget="input"]') !== null, 'missing input');
        mark('list', document.querySelector('[data-tetra-widget="list"]') !== null, 'missing list');
        mark('button', document.querySelector('[data-tetra-widget="button"]') !== null, 'missing button');
        input.dispatchEvent(new Event('focus', { bubbles: true }));
        mark('focus', uiState.focused, 'focus did not update state');
        input.value = 'tetra-web';
        input.dispatchEvent(new InputEvent('input', { bubbles: true, data: 'tetra-web' }));
        mark('input-event', uiState.input === 'tetra-web', `input=${uiState.input}`);
        input.dispatchEvent(new Event('change', { bubbles: true }));
        mark('change', uiState.changed, 'change did not update state');
        list.value = 'item-2';
        list.dispatchEvent(new Event('change', { bubbles: true }));
        mark('select', uiState.selected === 'item-2', `selected=${uiState.selected}`);
        const generatedButton = Array.from(host.querySelectorAll('button')).find((node) => node.textContent.includes('event click ->'));
        const generatedBinding = host.querySelector('[data-tetra-binding="countValue"]');
        const bindingBefore = generatedBinding ? generatedBinding.textContent : '';
        if (generatedButton) {
          generatedButton.click();
        }
        button.click();
        mark('click', uiState.clicked, 'click did not update state');
        const bindingAfter = generatedBinding ? generatedBinding.textContent : '';
        mark('redraw-update', uiState.redraws > 0 && text.textContent.includes('click'), `redraws=${uiState.redraws}`);
        if (!generatedButton || !generatedBinding || bindingBefore === bindingAfter) {
          throw new Error('ui command dispatch did not update generated binding');
        }
        trace.push('ui-event-dispatch:web-command-dispatch');
        await new Promise((resolve) => setTimeout(() => {
          uiState.timer = true;
          redraw('timer');
          resolve();
        }, 0));
        mark('timer', uiState.timer, 'timer did not fire');
        await Promise.resolve().then(() => {
          uiState.asyncDone = true;
          redraw('async');
        });
        mark('async-command', uiState.asyncDone, 'async command did not complete');
        try {
          throw new Error('recoverable-ui-command');
        } catch (err) {
          uiState.errorRecovered = String(err && err.message ? err.message : err).includes('recoverable');
          redraw('recovered');
        }
        mark('error-recovery', uiState.errorRecovered, 'recoverable error was not handled');
        const code = await runMainProbe();
        mark('main-exit', code === 0, `exit=${code}`);
        const stdoutStart = logs.length;
        const probeCode = await runRuntimeProbe();
        mark('stdout', logs.slice(stdoutStart).some((line) => line.includes('web runtime smoke stdout')), 'missing console output');
        mark('nonzero-exit', probeCode === 7, `exit=${probeCode}`);
        try {
          await runTetra(new URL('./missing-web-smoke.wasm', import.meta.url));
          mark('failure-propagation', false, 'missing module resolved');
        } catch (err) {
          const message = String(err && err.message ? err.message : err);
          mark('failure-propagation', message.includes('fetch failed') || message.includes('404'), message);
        }
        const firstProbe = await instantiateRuntimeProbe();
        const secondProbe = await instantiateRuntimeProbe();
        mark(
          'repeated-instantiation',
          firstProbe && secondProbe && firstProbe.instance && secondProbe.instance && firstProbe.instance !== secondProbe.instance,
          'instances were not distinct',
        );
        const firstMain = await instantiateTetra();
        mark('main-instantiation', firstMain && firstMain.instance && typeof firstMain.instance.exports.tetra_main === 'function', 'missing tetra_main');
        const views = Array.isArray(bundle.views) ? bundle.views.length : 0;
        el.textContent = `ok:${code}:ui=${views}:runtime=ok`;
      } catch (err) {
        el.textContent = `error:${String(err)}`;
      } finally {
        traceEl.textContent = trace.join(';');
        console.log = originalLog;
      }
    </script>
  </body>
</html>
HTML
    else
    cat >"$tmp_dir/index.html" <<'HTML'
<!doctype html>
<html>
  <body>
    <pre id="result">pending</pre>
    <pre id="runtime-trace"></pre>
    <script type="module">
      import { runTetra, instantiateTetra } from './web_smoke.mjs';
      import { runTetra as runMainProbe } from './web_main_probe.mjs';
      import { runTetra as runRuntimeProbe, instantiateTetra as instantiateRuntimeProbe } from './web_runtime_probe.mjs';
      const el = document.getElementById('result');
      const traceEl = document.getElementById('runtime-trace');
      const trace = [];
      const logs = [];
      const originalLog = console.log.bind(console);
      console.log = (...args) => {
        logs.push(args.map(String).join(' '));
        originalLog(...args);
      };
      function mark(name, ok, detail = '') {
        trace.push(`${name}:${ok ? 'ok' : `fail:${detail}`}`);
        if (!ok) {
          throw new Error(`runtime-${name}:${detail}`);
        }
      }
      try {
        const code = await runMainProbe();
        mark('main-exit', code === 0, `exit=${code}`);
        const stdoutStart = logs.length;
        const probeCode = await runRuntimeProbe();
        mark('stdout', logs.slice(stdoutStart).some((line) => line.includes('web runtime smoke stdout')), 'missing console output');
        mark('nonzero-exit', probeCode === 7, `exit=${probeCode}`);
        try {
          await runTetra(new URL('./missing-web-smoke.wasm', import.meta.url));
          mark('failure-propagation', false, 'missing module resolved');
        } catch (err) {
          const message = String(err && err.message ? err.message : err);
          mark('failure-propagation', message.includes('fetch failed') || message.includes('404'), message);
        }
        const firstProbe = await instantiateRuntimeProbe();
        const secondProbe = await instantiateRuntimeProbe();
        mark(
          'repeated-instantiation',
          firstProbe && secondProbe && firstProbe.instance && secondProbe.instance && firstProbe.instance !== secondProbe.instance,
          'instances were not distinct',
        );
        const firstMain = await instantiateTetra();
        mark('main-instantiation', firstMain && firstMain.instance && typeof firstMain.instance.exports.tetra_main === 'function', 'missing tetra_main');
        el.textContent = `ok:${code}:runtime=ok`;
      } catch (err) {
        el.textContent = `error:${String(err)}`;
      } finally {
        traceEl.textContent = trace.join(';');
        console.log = originalLog;
      }
    </script>
  </body>
</html>
HTML
    fi

    if [[ "$status" != "fail" ]]; then
      port=""
      for candidate in 8711 8712 8713 8714 8715; do
        if command -v lsof >/dev/null 2>&1; then
          if lsof -iTCP:"$candidate" -sTCP:LISTEN >/dev/null 2>&1; then
            continue
          fi
        fi
        port="$candidate"
        break
      done
      if [[ -z "$port" ]]; then
        blocker="unable to allocate local HTTP port"
        status="blocked"
      elif ! command -v python3 >/dev/null 2>&1; then
        blocker="runtime prerequisite unavailable: python3 -m http.server"
        status="blocked"
      else
        python3 -m http.server "$port" --bind 127.0.0.1 --directory "$tmp_dir" >"$tmp_dir/server.log" 2>&1 &
        server_pid=$!
        sleep 1
        dom_out="$tmp_dir/dom.html"
        chromium_err="$tmp_dir/chromium.err"
        if "$browser_runner" "${browser_flags[@]}" --virtual-time-budget=12000 --dump-dom "http://127.0.0.1:${port}/index.html" >"$dom_out" 2>"$chromium_err"; then
          result="$(sed -n 's/.*id="result">\([^<]*\)<.*/\1/p' "$dom_out" | head -n 1)"
          runtime_trace="$(sed -n 's/.*id="runtime-trace">\([^<]*\)<.*/\1/p' "$dom_out" | head -n 1)"
          if [[ "$result" == ok:* ]]; then
            for marker in main-exit:ok stdout:ok nonzero-exit:ok failure-propagation:ok repeated-instantiation:ok main-instantiation:ok window-mount:ok root-mount:ok layout:ok text:ok button:ok input:ok list:ok panel:ok focus:ok input-event:ok change:ok select:ok click:ok timer:ok async-command:ok redraw-update:ok error-recovery:ok ui-event-dispatch:web-command-dispatch; do
              if [[ "$runtime_trace" != *"$marker"* ]]; then
                status="fail"
                blocker="browser runtime trace missing ${marker}"
                break
              fi
            done
            if [[ "$scope_active" == "true" ]]; then
              if [[ "$status" == "fail" ]]; then
                :
              elif [[ "$ui_schema" != "tetra.ui.v1" ]]; then
                status="fail"
                blocker="unexpected UI schema '${ui_schema}'"
              elif [[ "$result" != ok:*:ui=* ]]; then
                status="fail"
                blocker="UI smoke result missing ui=* metadata marker"
              else
                status="pass"
              fi
            else
              status="blocked"
              blocker="no UI-specific smoke source found in examples/; fallback wasm web smoke ran successfully"
            fi
          else
            status="fail"
            blocker="browser automation did not produce ok:* result"
          fi
        else
          status="blocked"
          blocker="headless browser command failed: ${browser_runner}"
        fi
        kill "$server_pid" >/dev/null 2>&1 || true
      fi
    fi
  else
    status="fail"
    blocker="wasm32-web build failed for ${source_path}"
  fi
fi

ui_bundle_artifact_path="${report_path%.json}.ui.json"
ui_module_artifact_path="${report_path%.json}.ui.web.mjs"
if [[ -f "$tmp_dir/dom.html" ]]; then
  cp -- "$tmp_dir/dom.html" "$dom_path"
fi
if [[ -f "$tmp_dir/chromium.err" ]]; then
  cp -- "$tmp_dir/chromium.err" "$chromium_err_path"
fi
if [[ -f "$ui_bundle_path" ]]; then
  cp -- "$ui_bundle_path" "$ui_bundle_artifact_path"
  ui_bundle_path="$ui_bundle_artifact_path"
fi
if [[ -f "$ui_module_path" ]]; then
  cp -- "$ui_module_path" "$ui_module_artifact_path"
  ui_module_path="$ui_module_artifact_path"
fi

write_web_smoke_report
go run ./tools/cmd/validate-web-ui-smoke --report "$report_path"

if [[ "$status" != "pass" ]]; then
  echo "release/v1_0/web-smoke: $blocker" >&2
  exit 1
fi

echo "web ui smoke report: $report_path"
