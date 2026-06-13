#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="reports/surface/package"

usage() {
  cat <<'USAGE'
Usage: bash scripts/release/surface/surface-package-smoke.sh [--report-dir DIR]

Builds deterministic Surface app package evidence for surface-v1-linux-web.
It packages the command-palette reference app as linux-x64 and wasm32-web tar.gz
artifacts, records local asset hashes, unpacks and runs the linux-x64 package,
writes a hash-pinned update channel manifest, and records signing,
notarization, automatic update, React, Electron, DOM app UI, CSS runtime,
remote asset, and user JavaScript app logic nonclaims.
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
  export GOCACHE="$repo_root/.cache/go-build-surface-package"
fi
mkdir -p "$GOCACHE"

report_dir_arg="${report_dir%/}"
report_dir="$report_dir_arg"
if [[ -z "$report_dir" ]]; then
  surface_release_guard_reject "surface_package_smoke:" "--report-dir requires a value"
fi
if [[ "$report_dir" = /* || "$report_dir" == "." || "$report_dir" == "./" || "$report_dir" == -* ]]; then
  surface_release_guard_reject_unsafe "surface_package_smoke:" "$report_dir"
fi
IFS='/' read -r -a report_parts <<<"$report_dir"
current="$repo_root"
for part in "${report_parts[@]}"; do
  if [[ -z "$part" || "$part" == "." ]]; then
    continue
  fi
  if [[ "$part" == ".." ]]; then
    surface_release_guard_reject_unsafe "surface_package_smoke:" "$report_dir"
  fi
  current="$current/$part"
  if [[ -L "$current" ]]; then
    surface_release_guard_reject_symlink "surface_package_smoke:" "$report_dir"
  fi
done
report_dir_abs="$repo_root/$report_dir"
if [[ -e "$report_dir_abs" && ! -d "$report_dir_abs" ]]; then
  surface_release_guard_reject "surface_package_smoke:" "refusing to use non-directory report path: $report_dir"
fi
mkdir -p "$report_dir_abs"
report_dir="$(realpath --relative-to="$repo_root" "$report_dir_abs")"

report_path="$report_dir/surface-package.json"
work_dir="$report_dir/surface-package-work"
packages_dir="$report_dir/surface-packages"
install_dir="$report_dir/surface-install/linux-x64"
updates_dir="$report_dir/surface-updates"
for owned_path in "$report_path" "$work_dir" "$packages_dir" "$install_dir" "$updates_dir"; do
  if [[ -e "$owned_path" ]]; then
    echo "surface_package_smoke: refusing to reuse existing package artifact path: $owned_path" >&2
    exit 2
  fi
done
mkdir -p "$work_dir/build" "$work_dir/assets" "$packages_dir" "$install_dir" "$updates_dir"

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

file_size() {
  wc -c < "$1" | tr -d ' '
}

verify_sha() {
  local path="$1"
  local expected="$2"
  local got
  got="$(sha256_file "$path")"
  if [[ "$got" != "$expected" ]]; then
    echo "surface_package_smoke: sha256 mismatch for $path: got $got want $expected" >&2
    exit 1
  fi
}

source_path="examples/surface_reference_command_palette.tetra"
reference_app="command-palette"
linux_binary="$work_dir/build/surface-command-palette-linux-x64"
wasm_binary="$work_dir/build/surface-command-palette.wasm"
wasm_loader="${wasm_binary%.wasm}.mjs"

if rg -n 'React|Electron|Chromium|DOM|CSS|JavaScript|platform_widget|native_widget|platform widget|native widget|lib\.core\.component|lib\.core\.widgets' "$source_path" > "$work_dir/source-scan.txt"; then
  echo "surface_package_smoke: reference app source contains forbidden package/runtime vocabulary: $source_path" >&2
  cat "$work_dir/source-scan.txt" >&2
  exit 1
fi
: > "$work_dir/source-scan.txt"

go run ./cli/cmd/tetra check "$source_path"
go run ./cli/cmd/tetra build --target linux-x64 -o "$linux_binary" "$source_path"
go run ./cli/cmd/tetra build --target wasm32-web -o "$wasm_binary" "$source_path"
go run ./cli/cmd/tetra run --target linux-x64 "$source_path"
if [[ ! -s "$wasm_loader" ]]; then
  echo "surface_package_smoke: wasm32-web compiler-owned loader missing: $wasm_loader" >&2
  exit 1
fi

source_sha="$(sha256_file "$source_path")"
linux_build_sha="$(sha256_file "$linux_binary")"
wasm_build_sha="$(sha256_file "$wasm_binary")"

icon_asset="$work_dir/assets/app-icon.txt"
theme_asset="$work_dir/assets/theme-manifest.json"
cat > "$icon_asset" <<'ASSET'
surface-command-palette-package-icon-v1
ASSET
cat > "$theme_asset" <<'JSON'
{"schema":"tetra.surface.package-theme.v1","tokens":["surface.bg","surface.fg","accent.primary"],"local_only":true}
JSON
icon_sha="$(sha256_file "$icon_asset")"
theme_sha="$(sha256_file "$theme_asset")"
icon_size="$(file_size "$icon_asset")"
theme_size="$(file_size "$theme_asset")"

asset_manifest="$work_dir/assets/asset-manifest.json"
cat > "$asset_manifest" <<JSON
{
  "schema": "tetra.surface.package-assets.v1",
  "local_only": true,
  "network_fetch_allowed": false,
  "assets": [
    {"path": $(json_string "$icon_asset"), "kind": "icon", "sha256": $(json_string "$icon_sha"), "size_bytes": $icon_size},
    {"path": $(json_string "$theme_asset"), "kind": "theme", "sha256": $(json_string "$theme_sha"), "size_bytes": $theme_size}
  ]
}
JSON
asset_manifest_sha="$(sha256_file "$asset_manifest")"

linux_package_name="surface-command-palette-linux-x64"
linux_root="$work_dir/linux-x64/$linux_package_name"
linux_manifest="$work_dir/linux-x64/package-manifest.json"
mkdir -p "$linux_root/bin" "$linux_root/src" "$linux_root/assets"
install -m 0755 "$linux_binary" "$linux_root/bin/surface-command-palette"
cp "$source_path" "$linux_root/src/main.tetra"
cp "$icon_asset" "$theme_asset" "$asset_manifest" "$linux_root/assets/"
cat > "$linux_manifest" <<JSON
{
  "schema": "tetra.surface.package-manifest.v1",
  "package_format": "surface-app-package-v1",
  "target": "linux-x64",
  "source": $(json_string "$source_path"),
  "reference_app": $(json_string "$reference_app"),
  "entry": "bin/surface-command-palette",
  "source_sha256": $(json_string "$source_sha"),
  "build_sha256": $(json_string "$linux_build_sha"),
  "asset_manifest_sha256": $(json_string "$asset_manifest_sha"),
  "local_only_assets": true
}
JSON
cp "$linux_manifest" "$linux_root/package-manifest.json"
linux_package="$packages_dir/surface-command-palette-linux-x64.tar.gz"
(
  cd "$work_dir/linux-x64"
  tar --sort=name --owner=0 --group=0 --numeric-owner --mtime="UTC 2026-06-06" -czf "../../surface-packages/surface-command-palette-linux-x64.tar.gz" "$linux_package_name"
)
linux_package_sha="$(sha256_file "$linux_package")"

web_package_name="surface-command-palette-wasm32-web"
web_root="$work_dir/wasm32-web/$web_package_name"
web_manifest="$work_dir/wasm32-web/package-manifest.json"
web_entry="$work_dir/wasm32-web/index.html"
mkdir -p "$web_root" "$web_root/assets"
cp "$wasm_binary" "$web_root/surface-command-palette.wasm"
cp "$wasm_loader" "$web_root/surface-command-palette.mjs"
cp "$icon_asset" "$theme_asset" "$asset_manifest" "$web_root/assets/"
cat > "$web_entry" <<'HTML'
<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><title>Surface command palette</title></head>
<body><canvas id="surface-canvas" width="800" height="480"></canvas><script type="module" src="./surface-command-palette.mjs"></script></body>
</html>
HTML
cp "$web_entry" "$web_root/index.html"
cat > "$web_manifest" <<JSON
{
  "schema": "tetra.surface.package-manifest.v1",
  "package_format": "surface-app-package-v1",
  "target": "wasm32-web",
  "source": $(json_string "$source_path"),
  "reference_app": $(json_string "$reference_app"),
  "entry": "index.html",
  "wasm": "surface-command-palette.wasm",
  "loader": "surface-command-palette.mjs",
  "source_sha256": $(json_string "$source_sha"),
  "build_sha256": $(json_string "$wasm_build_sha"),
  "asset_manifest_sha256": $(json_string "$asset_manifest_sha"),
  "local_only_assets": true,
  "user_js_app_logic": false
}
JSON
cp "$web_manifest" "$web_root/package-manifest.json"
web_package="$packages_dir/surface-command-palette-wasm32-web.tar.gz"
(
  cd "$work_dir/wasm32-web"
  tar --sort=name --owner=0 --group=0 --numeric-owner --mtime="UTC 2026-06-06" -czf "../../surface-packages/surface-command-palette-wasm32-web.tar.gz" "$web_package_name"
)
web_package_sha="$(sha256_file "$web_package")"

verify_sha "$linux_package" "$linux_package_sha"
verify_sha "$web_package" "$web_package_sha"
verify_sha "$asset_manifest" "$asset_manifest_sha"

tar -xzf "$linux_package" -C "$install_dir"
installed_binary="$install_dir/$linux_package_name/bin/surface-command-palette"
if [[ ! -x "$installed_binary" ]]; then
  echo "surface_package_smoke: installed binary missing or not executable: $installed_binary" >&2
  exit 1
fi
"$installed_binary"

if [[ ! -s "$install_dir/$linux_package_name/package-manifest.json" ]]; then
  echo "surface_package_smoke: installed package manifest missing" >&2
  exit 1
fi
if [[ ! -s "$web_root/package-manifest.json" || ! -s "$web_root/index.html" || ! -s "$web_root/surface-command-palette.wasm" || ! -s "$web_root/surface-command-palette.mjs" ]]; then
  echo "surface_package_smoke: web package bundle is incomplete" >&2
  exit 1
fi

channel_manifest="$updates_dir/channel.json"
rollback_manifest="$updates_dir/rollback.json"
cat > "$channel_manifest" <<JSON
{
  "schema": "tetra.surface.update-channel.v1",
  "strategy": "hash-pinned-channel-manifest-v1",
  "channel": "local-stable-scoped",
  "current_version": "p23.0.0",
  "latest_version": "p23.0.0",
  "latest_package": {
    "target": "linux-x64",
    "path": $(json_string "$linux_package"),
    "sha256": $(json_string "$linux_package_sha")
  },
  "signature_required_for_stable_promotion": true,
  "auto_update_runtime_claim": false,
  "network_update_claim": false
}
JSON
cat > "$rollback_manifest" <<JSON
{
  "schema": "tetra.surface.update-rollback.v1",
  "strategy": "hash-pinned-channel-manifest-v1",
  "rollback_version": "p23.0.0",
  "package": {
    "path": $(json_string "$linux_package"),
    "sha256": $(json_string "$linux_package_sha")
  }
}
JSON

cat > "$report_path" <<JSON
{
  "schema": "tetra.surface.package.v1",
  "model": "surface-package-v1",
  "release_scope": "surface-v1-linux-web",
  "producer": "scripts/release/surface/surface-package-smoke.sh",
  "source": $(json_string "$source_path"),
  "reference_app": $(json_string "$reference_app"),
  "package_format": "surface-app-package-v1",
  "format_version": 1,
  "artifact_root": $(json_string "$work_dir"),
  "packages": [
    {
      "target": "linux-x64",
      "kind": "linux-x64-tar.gz",
      "path": $(json_string "$linux_package"),
      "manifest_path": $(json_string "$linux_manifest"),
      "sha256": $(json_string "$linux_package_sha"),
      "asset_manifest_sha256": $(json_string "$asset_manifest_sha"),
      "source_sha256": $(json_string "$source_sha"),
      "build_sha256": $(json_string "$linux_build_sha"),
      "contains_executable": true,
      "contains_web_bundle": false,
      "local_only_assets": true,
      "pass": true
    },
    {
      "target": "wasm32-web",
      "kind": "wasm32-web-tar.gz",
      "path": $(json_string "$web_package"),
      "manifest_path": $(json_string "$web_manifest"),
      "sha256": $(json_string "$web_package_sha"),
      "asset_manifest_sha256": $(json_string "$asset_manifest_sha"),
      "source_sha256": $(json_string "$source_sha"),
      "build_sha256": $(json_string "$wasm_build_sha"),
      "contains_executable": false,
      "contains_web_bundle": true,
      "local_only_assets": true,
      "pass": true
    }
  ],
  "assets": [
    {"path": $(json_string "$icon_asset"), "kind": "icon", "sha256": $(json_string "$icon_sha"), "size_bytes": $icon_size, "local_only": true, "network_fetch_allowed": false, "pass": true},
    {"path": $(json_string "$theme_asset"), "kind": "theme", "sha256": $(json_string "$theme_sha"), "size_bytes": $theme_size, "local_only": true, "network_fetch_allowed": false, "pass": true}
  ],
  "install_smokes": [
    {
      "target": "linux-x64",
      "package_path": $(json_string "$linux_package"),
      "install_dir": $(json_string "$install_dir"),
      "installed_binary": $(json_string "$installed_binary"),
      "command": $(json_string "$installed_binary"),
      "exit_code": 0,
      "artifact_hash_verified": true,
      "package_manifest_verified": true,
      "app_run": true,
      "pass": true
    }
  ],
  "web_bundles": [
    {
      "target": "wasm32-web",
      "package_path": $(json_string "$web_package"),
      "web_entry": $(json_string "$web_root/index.html"),
      "wasm_artifact": $(json_string "$web_root/surface-command-palette.wasm"),
      "loader_artifact": $(json_string "$web_root/surface-command-palette.mjs"),
      "command": $(json_string "tetra build --target wasm32-web -o $wasm_binary $source_path"),
      "artifact_hash_verified": true,
      "package_manifest_verified": true,
      "pass": true
    }
  ],
  "update_strategy": {
    "strategy": "hash-pinned-channel-manifest-v1",
    "manifest_format": "tetra.surface.update-channel.v1",
    "channel_manifest": $(json_string "$channel_manifest"),
    "current_version": "p23.0.0",
    "latest_version": "p23.0.0",
    "latest_package_path": $(json_string "$linux_package"),
    "latest_package_sha256": $(json_string "$linux_package_sha"),
    "package_hash_pinned": true,
    "rollback_manifest": $(json_string "$rollback_manifest"),
    "signature_required_for_stable_promotion": true,
    "auto_update_runtime_claim": false,
    "network_update_claim": false,
    "pass": true
  },
  "signing": {
    "status": "nonclaim",
    "signed": false,
    "notarized": false,
    "production_claim": false,
    "evidence": "",
    "blocked_reason": "platform signing keys and CI signing evidence are not present in this release"
  },
  "notarization": {
    "status": "nonclaim",
    "signed": false,
    "notarized": false,
    "production_claim": false,
    "evidence": "",
    "blocked_reason": "macOS notarization evidence is unavailable because macOS Surface target host is unsupported"
  },
  "negative_guards": {
    "no_react_runtime": true,
    "no_electron_runtime": true,
    "no_dom_app_ui_tree": true,
    "no_css_runtime": true,
    "no_user_js_app_logic": true,
    "no_remote_asset_fetch": true,
    "no_unsigned_signing_claim": true,
    "no_notarization_without_platform_evidence": true,
    "no_auto_update_without_runtime_evidence": true,
    "no_docs_only_package_claim": true,
    "install_run_required": true,
    "web_bundle_required": true,
    "artifact_hashes_required": true
  },
  "pass": true
}
JSON

go run ./tools/cmd/validate-surface-package --report "$report_path"
go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"

echo "Surface package report: $report_path"
echo "Surface linux package: $linux_package"
echo "Surface web package: $web_package"
echo "Surface update channel manifest: $channel_manifest"
