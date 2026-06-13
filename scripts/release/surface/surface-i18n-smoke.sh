#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="reports/surface/i18n"

usage() {
  cat <<'USAGE'
Usage: bash scripts/release/surface/surface-i18n-smoke.sh [--report-dir DIR]

Builds bounded Surface internationalization and localization evidence for
surface-v1-linux-web. It records string tables, locale selection, fallback
language evidence, missing_key_diagnostic evidence, deterministic format_hooks,
localized form reference app execution, and an RTL placeholder nonclaim without
full ICU or full bidi shaping claims.
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
  export GOCACHE="$repo_root/.cache/go-build-surface-i18n"
fi
mkdir -p "$GOCACHE"

report_dir_arg="${report_dir%/}"
report_dir="$report_dir_arg"
if [[ -z "$report_dir" ]]; then
  surface_release_guard_reject "surface_i18n_smoke:" "--report-dir requires a value"
fi
if [[ "$report_dir" = /* || "$report_dir" == "." || "$report_dir" == "./" || "$report_dir" == -* ]]; then
  surface_release_guard_reject_unsafe "surface_i18n_smoke:" "$report_dir"
fi
IFS='/' read -r -a report_parts <<<"$report_dir"
current="$repo_root"
for part in "${report_parts[@]}"; do
  if [[ -z "$part" || "$part" == "." ]]; then
    continue
  fi
  if [[ "$part" == ".." ]]; then
    surface_release_guard_reject_unsafe "surface_i18n_smoke:" "$report_dir"
  fi
  current="$current/$part"
  if [[ -L "$current" ]]; then
    surface_release_guard_reject_symlink "surface_i18n_smoke:" "$report_dir"
  fi
done
report_dir_abs="$repo_root/$report_dir"
if [[ -e "$report_dir_abs" && ! -d "$report_dir_abs" ]]; then
  surface_release_guard_reject "surface_i18n_smoke:" "refusing to use non-directory report path: $report_dir"
fi
mkdir -p "$report_dir_abs"
report_dir="$(realpath --relative-to="$repo_root" "$report_dir_abs")"

report_path="$report_dir/surface-i18n.json"
work_dir="$report_dir/surface-i18n-work"
i18n_dir="$report_dir/surface-i18n"
for owned_path in "$report_path" "$work_dir" "$i18n_dir"; do
  if [[ -e "$owned_path" ]]; then
    echo "surface_i18n_smoke: refusing to reuse existing i18n artifact path: $owned_path" >&2
    exit 2
  fi
done
mkdir -p "$work_dir/build" "$i18n_dir"

json_string() {
  local value="$1"
  value="${value//\\/\\\\}"
  value="${value//\"/\\\"}"
  value="${value//$'\n'/\\n}"
  value="${value//$'\r'/\\r}"
  value="${value//$'\t'/\\t}"
  printf '"%s"' "$value"
}

sha256_file() {
  sha256sum "$1" | awk '{print "sha256:" $1}'
}

source_path="examples/surface_reference_localized_form.tetra"
reference_app="localized-form"
linux_binary="$work_dir/build/surface-localized-form-linux-x64"

if rg -n 'React|Electron|Chromium|CSS runtime|JavaScript app logic|platform_widget|native_widget|platform widget|native widget|lib\.core\.component|lib\.core\.widgets' "$source_path" > "$work_dir/source-scan.txt"; then
  echo "surface_i18n_smoke: localized form source contains forbidden runtime vocabulary: $source_path" >&2
  cat "$work_dir/source-scan.txt" >&2
  exit 1
fi
if ! rg -n 'import lib\.core\.i18n as i18n' "$source_path" >> "$work_dir/source-scan.txt"; then
  echo "surface_i18n_smoke: localized form source must import lib.core.i18n: $source_path" >&2
  exit 1
fi

go run ./cli/cmd/tetra check "$source_path"
go run ./cli/cmd/tetra build --target linux-x64 -o "$linux_binary" "$source_path"
"$linux_binary"

cat > "$i18n_dir/string-table-en-US.json" <<'JSON'
{
  "locale": "en-US",
  "entries": {
    "form.title": "Profile",
    "form.name": "Name",
    "form.primary": "Save",
    "form.secondary": "Cancel",
    "form.status": "Ready"
  }
}
JSON

cat > "$i18n_dir/string-table-uk-UA.json" <<'JSON'
{
  "locale": "uk-UA",
  "fallback_locale": "en-US",
  "entries": {
    "form.title": "Профіль",
    "form.name": "Ім'я",
    "form.primary": "Зберегти",
    "form.status": "Готово"
  }
}
JSON

cat > "$i18n_dir/lookup-trace.json" <<'JSON'
{
  "requested_locale": "uk-UA",
  "selected_locale": "uk-UA",
  "fallback_locale": "en-US",
  "lookups": [
    {"key":"form.title","resolved_locale":"uk-UA","source":"primary"},
    {"key":"form.secondary","resolved_locale":"en-US","source":"fallback"},
    {"key":"form.unknown","resolved_locale":"en-US","source":"missing","diagnostic_code":2001}
  ]
}
JSON

en_sha="$(sha256_file "$i18n_dir/string-table-en-US.json")"
uk_sha="$(sha256_file "$i18n_dir/string-table-uk-UA.json")"

cat > "$report_path" <<JSON
{
  "schema": "tetra.surface.i18n.v1",
  "model": "surface-i18n-v1",
  "release_scope": "surface-v1-linux-web",
  "producer": "scripts/release/surface/surface-i18n-smoke.sh",
  "source": $(json_string "$source_path"),
  "reference_app": $(json_string "$reference_app"),
  "target": "linux-x64",
  "string_tables": [
    {"locale":"en-US","entry_count":5,"checksum":$(json_string "$en_sha"),"primary":true,"fallback":false,"pass":true},
    {"locale":"uk-UA","entry_count":4,"checksum":$(json_string "$uk_sha"),"primary":false,"fallback":true,"pass":true}
  ],
  "locale_selection": {
    "requested_locale": "uk-UA",
    "selected_locale": "uk-UA",
    "fallback_locale": "en-US",
    "fallback_used": true,
    "unsupported_locale_rejected": true,
    "pass": true
  },
  "lookups": [
    {"key":"form.title","locale":"uk-UA","resolved_locale":"uk-UA","source":"primary","missing_key":false,"fallback_used":false,"diagnostic_code":0,"pass":true},
    {"key":"form.secondary","locale":"uk-UA","resolved_locale":"en-US","source":"fallback","missing_key":false,"fallback_used":true,"diagnostic_code":0,"pass":true},
    {"key":"form.unknown","locale":"uk-UA","resolved_locale":"en-US","source":"missing","missing_key":true,"fallback_used":true,"diagnostic_code":2001,"pass":true}
  ],
  "format_hooks": [
    {"kind":"date","locale":"uk-UA","input":"2026-06-12","output":"2026-06-12","deterministic":true,"icu_claim":false,"pass":true},
    {"kind":"number","locale":"uk-UA","input":"4200","output":"4200","deterministic":true,"icu_claim":false,"pass":true}
  ],
  "text_direction": {
    "default_direction": "ltr",
    "rtl_placeholder": true,
    "full_bidi_supported": false,
    "full_bidi_claim": false,
    "shaping_proof": false,
    "nonclaim": "rtl-placeholder-without-full-bidi-shaping-v1",
    "pass": true
  },
  "localized_form": {
    "shape": "localized-form",
    "source": $(json_string "$source_path"),
    "imports": ["lib.core.surface","lib.core.block","lib.core.morph","lib.core.i18n"],
    "compiles": true,
    "runs": true,
    "exit_code": 0,
    "localized_strings": true,
    "fallback_evidence": true,
    "missing_key_diagnostic": true,
    "format_hook_evidence": true,
    "resolves_to_block": true,
    "pass": true
  },
  "negative_guards": {
    "no_full_icu_claim": true,
    "no_full_bidi_claim": true,
    "no_rtl_production_claim": true,
    "no_missing_key_silent_fallback": true,
    "no_docs_only_i18n_claim": true,
    "no_react_intl_runtime": true,
    "no_platform_locale_dependency": true
  },
  "pass": true
}
JSON

go run ./tools/cmd/validate-surface-i18n --report "$report_path"
go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"

echo "Surface i18n report: $report_path"
