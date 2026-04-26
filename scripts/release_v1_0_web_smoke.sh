#!/usr/bin/env bash
set -euo pipefail

report_path=""
source_override=""

usage() {
  cat <<'USAGE'
Usage: bash scripts/release_v1_0_web_smoke.sh --report PATH [--source examples/file.tetra]

Runs wasm32-web smoke in headless Chromium.
If no UI-specific smoke source exists, the script runs a fallback wasm web smoke
for evidence and marks the report as blocked.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --report)
      report_path="$2"
      shift 2
      ;;
    --source)
      source_override="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "release_v1_0_web_smoke: unknown option $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [[ -z "$report_path" ]]; then
  echo "release_v1_0_web_smoke: --report is required" >&2
  exit 2
fi

mkdir -p "$(dirname "$report_path")"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

status="blocked"
blocker=""
result=""
scope_active="false"
source_path=""
used_fallback="false"

if [[ -n "$source_override" ]]; then
  source_path="$source_override"
  scope_active="true"
else
  ui_candidate="$(
    {
      git ls-files 'examples/*.tetra'
      find examples -maxdepth 1 -type f -name '*.tetra' 2>/dev/null
    } | sort -u | grep -E '/[^/]*(ui.*smoke|view.*smoke|state.*smoke)[^/]*\.tetra$' | head -n 1 || true
  )"
  if [[ -n "$ui_candidate" ]]; then
    source_path="$ui_candidate"
    scope_active="true"
  else
    source_path="examples/flow_hello.tetra"
    scope_active="false"
    used_fallback="true"
  fi
fi

if ! command -v chromium >/dev/null 2>&1; then
  blocker="missing chromium in PATH"
  status="blocked"
else
  build_out="$tmp_dir/web_smoke"
  if ./tetra build --target wasm32-web -o "$build_out" "$source_path" >"$tmp_dir/build.out" 2>"$tmp_dir/build.err"; then
    cat >"$tmp_dir/index.html" <<'HTML'
<!doctype html>
<html>
  <body>
    <pre id="result">pending</pre>
    <script type="module">
      import { runTetra } from './web_smoke.mjs';
      const el = document.getElementById('result');
      try {
        const code = await runTetra();
        el.textContent = `ok:${code}`;
      } catch (err) {
        el.textContent = `error:${String(err)}`;
      }
    </script>
  </body>
</html>
HTML

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
    else
      python3 -m http.server "$port" --directory "$tmp_dir" >"$tmp_dir/server.log" 2>&1 &
      server_pid=$!
      sleep 1
      dom_out="$tmp_dir/dom.html"
      chromium_err="$tmp_dir/chromium.err"
      if chromium --headless --disable-gpu --virtual-time-budget=5000 --dump-dom "http://127.0.0.1:${port}/index.html" >"$dom_out" 2>"$chromium_err"; then
        result="$(sed -n 's/.*id="result">\([^<]*\)<.*/\1/p' "$dom_out" | head -n 1)"
        if [[ "$result" == ok:* ]]; then
          if [[ "$scope_active" == "true" ]]; then
            status="pass"
          else
            status="blocked"
            blocker="no UI-specific smoke source found in examples/; fallback wasm web smoke ran successfully"
          fi
        else
          status="fail"
          blocker="browser automation did not produce ok:* result"
        fi
      else
        status="fail"
        blocker="headless chromium command failed"
      fi
      kill "$server_pid" >/dev/null 2>&1 || true
    fi
  else
    status="fail"
    blocker="wasm32-web build failed for ${source_path}"
  fi
fi

dom_path="${report_path%.json}.dom.html"
chromium_err_path="${report_path%.json}.chromium.err"
if [[ -f "$tmp_dir/dom.html" ]]; then
  cp "$tmp_dir/dom.html" "$dom_path"
fi
if [[ -f "$tmp_dir/chromium.err" ]]; then
  cp "$tmp_dir/chromium.err" "$chromium_err_path"
fi

REPORT_PATH="$report_path" \
STATUS="$status" \
BLOCKER="$blocker" \
RESULT_TEXT="$result" \
SCOPE_ACTIVE="$scope_active" \
SOURCE_PATH="$source_path" \
USED_FALLBACK="$used_fallback" \
DOM_PATH="$dom_path" \
CHROMIUM_ERR_PATH="$chromium_err_path" \
node <<'JS'
const fs = require('fs');
const reportPath = process.env.REPORT_PATH;
const report = {
  schema: 'tetra.web-ui-smoke.v1alpha1',
  generated_at: new Date().toISOString(),
  target: 'wasm32-web',
  ui_scope_active: process.env.SCOPE_ACTIVE === 'true',
  source: process.env.SOURCE_PATH,
  used_fallback_source: process.env.USED_FALLBACK === 'true',
  automation: 'chromium --headless --dump-dom',
  status: process.env.STATUS,
  result: process.env.RESULT_TEXT || '',
  blocker: process.env.BLOCKER || '',
  dom_snapshot: process.env.DOM_PATH || '',
  chromium_stderr: process.env.CHROMIUM_ERR_PATH || '',
};
fs.writeFileSync(reportPath, JSON.stringify(report, null, 2) + '\n');
JS

if [[ "$status" != "pass" ]]; then
  echo "release_v1_0_web_smoke: $blocker" >&2
  exit 1
fi

echo "web ui smoke report: $report_path"
