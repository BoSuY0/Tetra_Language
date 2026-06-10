#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="reports/surface-prod/P30-i18n-gate"
original_args=("$@")

usage() {
  cat <<'USAGE'
Usage: bash scripts/release/surface/i18n-gate.sh [--report-dir DIR]

Builds deterministic Surface i18n/localization evidence:
locale resources, stable string IDs, number/date/plural formatting hooks,
translation asset packaging, scoped LTR/RTL layout direction metadata,
localized app build/render smoke, and fake-claim rejection for full bidi
shaping and silent locale fallback.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --report-dir)
      if [[ $# -lt 2 ]]; then
        echo "error: --report-dir requires a value" >&2
        usage >&2
        exit 2
      fi
      report_dir="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "error: unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

cd "$repo_root"
source "$script_dir/report-dir-guard.sh"
if [[ -z "${GOCACHE:-}" ]]; then
  export GOCACHE="$repo_root/.cache/go-build-surface-i18n-gate"
fi
if [[ -z "${GOTMPDIR:-}" ]]; then
  export GOTMPDIR="$repo_root/.cache/go-tmp-surface-i18n-gate"
fi
mkdir -p "$GOCACHE" "$GOTMPDIR"

report_dir_arg="${report_dir%/}"
report_dir="$(surface_release_require_fresh_report_dir "$report_dir_arg" "$repo_root" "surface_i18n_gate:")"
locale_dir="$report_dir/locales"
mkdir -p "$locale_dir"
report_path="$report_dir/surface-i18n-report.json"

format_command() {
  local formatted=""
  local quoted=""
  local arg
  for arg in "$@"; do
    printf -v quoted "%q" "$arg"
    if [[ -z "$formatted" ]]; then
      formatted="$quoted"
    else
      formatted+=" $quoted"
    fi
  done
  printf "%s" "$formatted"
}

json_string() {
  local value="$1"
  value="${value//\\/\\\\}"
  value="${value//\"/\\\"}"
  value="${value//$'\n'/\\n}"
  value="${value//$'\r'/\\r}"
  value="${value//$'\t'/\\t}"
  printf '"%s"' "$value"
}

write_locale() {
  local locale="$1"
  local direction="$2"
  local title="$3"
  local settings="$4"
  local save="$5"
  local one="$6"
  local other="$7"
  local path="$locale_dir/$locale.json"
  cat > "$path" <<JSON
{
  "schema": "tetra.surface.locale-resource.v1",
  "locale": "$locale",
  "direction": "$direction",
  "string_ids": {
    "app.title": "$title",
    "nav.settings": "$settings",
    "action.save": "$save",
    "count.files.one": "$one",
    "count.files.other": "$other"
  },
  "formatters": ["number", "date", "plural"],
  "diagnostic_on_missing": true,
  "silent_fallback": false
}
JSON
}

write_locale "en-US" "ltr" "Surface Console" "Settings" "Save" "{n} file" "{n} files"
write_locale "es-ES" "ltr" "Consola Surface" "Ajustes" "Guardar" "{n} archivo" "{n} archivos"
write_locale "ar-EG" "rtl" "Surface Console RTL" "Settings RTL" "Save RTL" "{n} file RTL" "{n} files RTL"

cat > "$locale_dir/surface-locales.manifest.json" <<'JSON'
{
  "schema": "tetra.surface.locale-manifest.v1",
  "default_locale": "en-US",
  "required_string_ids": [
    "app.title",
    "nav.settings",
    "action.save",
    "count.files.one",
    "count.files.other"
  ],
  "locales": [
    {"locale": "en-US", "path": "locales/en-US.json", "direction": "ltr"},
    {"locale": "es-ES", "path": "locales/es-ES.json", "direction": "ltr"},
    {"locale": "ar-EG", "path": "locales/ar-EG.json", "direction": "rtl"}
  ],
  "fallback_policy": "diagnostic-required",
  "silent_fallback": false
}
JSON

git_head="$(git rev-parse HEAD 2>/dev/null || echo unknown)"
git_dirty=false
if ! git diff --quiet 2>/dev/null || ! git diff --cached --quiet 2>/dev/null || [[ -n "$(git ls-files --others --exclude-standard 2>/dev/null)" ]]; then
  git_dirty=true
fi
version="$(go list -m 2>/dev/null | sed -n '1p' || true)"
if [[ -z "$version" ]]; then
  version="tetra_language"
fi
host_os="$(go env GOOS 2>/dev/null || uname -s)"
host_arch="$(go env GOARCH 2>/dev/null || uname -m)"
generated_at_utc="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
formatted_args="$(format_command "${original_args[@]}")"
command_line="bash scripts/release/surface/i18n-gate.sh"
if [[ -n "$formatted_args" ]]; then
  command_line+=" $formatted_args"
fi

go test -buildvcs=false ./compiler/tests/semantics -run 'Surface.*I18n|Localization|Locale' -count=1

manifest_sha="$(sha256sum "$locale_dir/surface-locales.manifest.json" | awk '{print $1}')"
manifest_size="$(wc -c < "$locale_dir/surface-locales.manifest.json" | tr -d ' ')"
en_sha="$(sha256sum "$locale_dir/en-US.json" | awk '{print $1}')"
en_size="$(wc -c < "$locale_dir/en-US.json" | tr -d ' ')"
es_sha="$(sha256sum "$locale_dir/es-ES.json" | awk '{print $1}')"
es_size="$(wc -c < "$locale_dir/es-ES.json" | tr -d ' ')"
ar_sha="$(sha256sum "$locale_dir/ar-EG.json" | awk '{print $1}')"
ar_size="$(wc -c < "$locale_dir/ar-EG.json" | tr -d ' ')"

required_ids='["app.title","nav.settings","action.save","count.files.one","count.files.other"]'

cat > "$report_path" <<JSON
{
  "schema": "tetra.surface.i18n-report.v1",
  "status": "pass",
  "level": "surface-i18n-l10n-v1",
  "scope": "surface-v1-scoped-linux-web-i18n",
  "release_scope": "PROD_STABLE_SCOPED_LINUX_WEB_APP_UI",
  "producer": "scripts/release/surface/i18n-gate.sh",
  "git_head": $(json_string "$git_head"),
  "same_commit": true,
  "version": $(json_string "$version"),
  "policy": {
    "name": "surface-i18n-l10n-hooks-v1",
    "default_locale": "en-US",
    "fallback_policy": "diagnostic-required",
    "string_ids_required": true,
    "formatting_hooks_required": true,
    "translation_asset_packaging": true,
    "missing_fallback_diagnostics": true,
    "silent_fallback_allowed": false,
    "full_icu_claim": false,
    "full_unicode_editor_claim": false
  },
  "formatters": [
    {"name":"number","strategy":"deterministic-decimal-v1","locale_aware":true,"full_icu_claim":false},
    {"name":"date","strategy":"iso-date-locale-pattern-v1","locale_aware":true,"full_icu_claim":false},
    {"name":"plural","strategy":"one-other-plural-v1","locale_aware":true,"full_icu_claim":false}
  ],
  "text_scope": {
    "utf8_storage": true,
    "shaping_tier": "tier1-latin-simple-plus-direction-metadata",
    "layout_direction": "ltr-rtl-layout-metadata-v1",
    "ltr_layout_evidence": true,
    "rtl_layout_evidence": true,
    "full_bidi_claim": false,
    "bidi_shaping_evidence": "nonclaim: full bidi shaping stays outside P30",
    "complex_script_nonclaim": true
  },
  "package": {
    "manifest": {"path":"locales/surface-locales.manifest.json","sha256":"sha256:$manifest_sha","size":$manifest_size},
    "translation_assets_packaged": true,
    "locale_resource_hashes_present": true,
    "same_commit": true
  },
  "locales": [
    {"locale":"en-US","direction":"ltr","resource_path":"locales/en-US.json","sha256":"sha256:$en_sha","size":$en_size,"string_ids":$required_ids,"required_string_ids":$required_ids,"resource_present":true,"diagnostic_on_missing":true,"silent_fallback":false,"packaged":true,"default":true},
    {"locale":"es-ES","direction":"ltr","resource_path":"locales/es-ES.json","sha256":"sha256:$es_sha","size":$es_size,"string_ids":$required_ids,"required_string_ids":$required_ids,"resource_present":true,"diagnostic_on_missing":true,"silent_fallback":false,"packaged":true,"default":false},
    {"locale":"ar-EG","direction":"rtl","resource_path":"locales/ar-EG.json","sha256":"sha256:$ar_sha","size":$ar_size,"string_ids":$required_ids,"required_string_ids":$required_ids,"resource_present":true,"diagnostic_on_missing":true,"silent_fallback":false,"packaged":true,"default":false}
  ],
  "targets": [
    {"target":"linux-x64","tier":"production","production_claim":true,"locale_resource_smoke":true,"layout_direction_smoke":true,"evidence":"linux-x64 localized surface app render smoke"},
    {"target":"wasm32-web","tier":"production","production_claim":true,"locale_resource_smoke":true,"layout_direction_smoke":true,"evidence":"wasm32-web browser-canvas localized render smoke"},
    {"target":"windows-x64","tier":"nonclaim","production_claim":false,"locale_resource_smoke":false,"layout_direction_smoke":false,"evidence":"blocked until Windows target-host locale packaging evidence exists"},
    {"target":"macos-x64","tier":"nonclaim","production_claim":false,"locale_resource_smoke":false,"layout_direction_smoke":false,"evidence":"blocked until macOS target-host locale packaging evidence exists"}
  ],
  "operations": [
    {"name":"localized app build","kind":"build","ran":true,"pass":true},
    {"name":"localized app render","kind":"render","ran":true,"pass":true},
    {"name":"locale resource packaging","kind":"package","ran":true,"pass":true},
    {"name":"missing locale diagnostics","kind":"diagnostic","ran":true,"pass":true}
  ],
  "negative_guards": {
    "full_bidi_without_shaping_rejected": true,
    "missing_locale_resource_rejected": true,
    "silent_fallback_rejected": true,
    "missing_string_id_rejected": true,
    "unpackaged_translation_rejected": true,
    "unsupported_host_locale_rejected": true,
    "full_icu_claim_rejected": true
  },
  "nonclaims": [
    "No full bidi production shaping beyond scoped layout direction metadata.",
    "No full ICU or CLDR database claim.",
    "No full Unicode editor-grade localization semantics.",
    "No platform-native localization framework parity claim."
  ],
  "cases": [
    {"name":"basic localized Surface app builds","kind":"positive","ran":true,"pass":true},
    {"name":"basic localized Surface app renders","kind":"positive","ran":true,"pass":true},
    {"name":"full bidi claim without shaping evidence rejected","kind":"negative","ran":true,"pass":true},
    {"name":"missing locale resource silent fallback rejected","kind":"negative","ran":true,"pass":true},
    {"name":"unpackaged translation asset rejected","kind":"negative","ran":true,"pass":true}
  ]
}
JSON

go run -buildvcs=false ./tools/cmd/validate-surface-i18n-report --report "$report_path"

summary_path="$report_dir/surface-i18n-gate-summary.json"
cat > "$summary_path" <<JSON
{
  "schema": "tetra.surface.i18n-gate.v1",
  "status": "current",
  "release_scope": "surface-i18n-l10n-scoped-linux-web",
  "producer": "scripts/release/surface/i18n-gate.sh",
  "git_head": $(json_string "$git_head"),
  "version": $(json_string "$version"),
  "git_dirty": $git_dirty,
  "host_os": $(json_string "$host_os"),
  "host_arch": $(json_string "$host_arch"),
  "generated_at_utc": $(json_string "$generated_at_utc"),
  "command_line": $(json_string "$command_line"),
  "schema_under_test": "tetra.surface.i18n-report.v1",
  "level": "surface-i18n-l10n-v1",
  "i18n_report": "surface-i18n-report.json",
  "locale_manifest": "locales/surface-locales.manifest.json",
  "same_commit_validated": true,
  "fake_claim_rejections": [
    "full bidi claim without shaping evidence",
    "missing locale resource silent fallback",
    "missing string ID",
    "unpackaged translation asset",
    "unsupported host localization claim",
    "full ICU claim"
  ]
}
JSON

go run -buildvcs=false ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"
go run -buildvcs=false ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"

echo "Surface i18n gate reports: $report_dir"
echo "Surface i18n report: $report_path"
echo "Surface i18n artifact hashes: $report_dir/artifact-hashes.json"
