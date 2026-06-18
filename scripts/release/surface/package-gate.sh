#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="reports/surface-prod/P26-package-gate"
original_args=("$@")

usage() {
  cat <<'USAGE'
Usage: bash scripts/release/surface/package-gate.sh [--report-dir DIR]

Builds deterministic Surface package distribution evidence:
Surface app scaffold, `.tdx` app package, scoped Linux tar package root,
install extraction smoke, launcher smoke, package report validation, and
artifact hashes. Windows/macOS/update remain explicit nonclaims until signing,
notarization, and signed-channel evidence exists.
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
  export GOCACHE="$repo_root/.cache/go-build-surface-package-gate"
fi
if [[ -z "${GOTMPDIR:-}" ]]; then
  export GOTMPDIR="$repo_root/.cache/go-tmp-surface-package-gate"
fi
mkdir -p "$GOCACHE" "$GOTMPDIR"

report_dir_arg="${report_dir%/}"
report_dir="$(surface_release_require_fresh_report_dir "$report_dir_arg" "$repo_root" "surface_package_gate:")"
scaffold_dir="$report_dir/scaffold/SurfaceDesk"
package_root="$report_dir/package-root"
archive_name="surface-desk-linux-x64.tar.gz"
report_path="$report_dir/surface-package-report.json"
install_dir="$report_dir/install-smoke"

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
command_line="bash scripts/release/surface/package-gate.sh"
if [[ -n "$formatted_args" ]]; then
  command_line+=" $formatted_args"
fi

go run -buildvcs=false ./cli/cmd/tetra new surface-app --template surface-dashboard "$scaffold_dir"
go run -buildvcs=false ./cli/cmd/tetra surface check "$scaffold_dir"

mkdir -p \
  "$package_root/app/SurfaceDesk" \
  "$package_root/package" \
  "$package_root/assets" \
  "$package_root/bin"
cp "$scaffold_dir/Capsule.t4" "$package_root/app/SurfaceDesk/Capsule.t4"
cp -R "$scaffold_dir/src" "$package_root/app/SurfaceDesk/src"
cp "$scaffold_dir/surface.template.json" "$package_root/app/SurfaceDesk/surface.template.json"
cp "$scaffold_dir/README.md" "$package_root/app/SurfaceDesk/README.md"

go run -buildvcs=false ./cli/cmd/tetra surface package "$scaffold_dir" -o "$package_root/package/surface-desk.tdx"

cat > "$package_root/assets/surface-assets.json" <<'JSON'
{
  "schema": "tetra.surface.assets.v1",
  "scope": "surface-v1-scoped-linux-web-package",
  "assets": []
}
JSON

cat > "$package_root/permissions.json" <<'JSON'
{
  "schema": "tetra.surface.permissions.v1",
  "scope": "surface-v1-scoped-linux-web-package",
  "filesystem": "app-bundle-readonly",
  "network": "not-requested",
  "clipboard": "host-gated"
}
JSON

cat > "$package_root/host-adapter.json" <<'JSON'
{
  "schema": "tetra.surface.host-adapter.v1",
  "target": "linux-x64",
  "adapter": "surface-linux-host-adapter-v1",
  "app_shell_abi": "tetra.surface.app-shell.v1",
  "nonclaims": [
    "no GTK or Qt widget dependency",
    "no macOS package production claim",
    "no Windows package production claim"
  ]
}
JSON

cat > "$package_root/surface-package.json" <<'JSON'
{
  "schema": "tetra.surface.package.manifest.v1",
  "target": "linux-x64",
  "format": "surface-linux-tar-v1",
  "entry": "app/SurfaceDesk/Capsule.t4",
  "surface_app_package": "package/surface-desk.tdx",
  "assets": "assets/surface-assets.json",
  "permissions": "permissions.json",
  "host_adapter": "host-adapter.json",
  "signature": "sha256-checksum-manifest"
}
JSON

cat > "$package_root/bin/surface-run.sh" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
app_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/../app/SurfaceDesk" && pwd)"
test -f "$app_dir/Capsule.t4"
test -f "$(dirname "${BASH_SOURCE[0]}")/../package/surface-desk.tdx"
echo "surface package launcher smoke: $app_dir"
SH
chmod 0755 "$package_root/bin/surface-run.sh"

(
  cd "$package_root"
  tar --sort=name --owner=0 --group=0 --numeric-owner --mtime="UTC 2026-06-10" \
    -czf "$archive_name" app package assets permissions.json host-adapter.json surface-package.json bin
)

mkdir -p "$install_dir"
tar -xzf "$package_root/$archive_name" -C "$install_dir"
go run -buildvcs=false ./cli/cmd/tetra surface check "$install_dir/app/SurfaceDesk"
bash "$install_dir/bin/surface-run.sh"

go run -buildvcs=false ./tools/cmd/surface-package-report \
  --root "$package_root" \
  --archive "$archive_name" \
  --out "$report_path" \
  --producer "scripts/release/surface/package-gate.sh" \
  --git-head "$git_head" \
  --version "$version"
go run -buildvcs=false ./tools/cmd/validate-surface-package-report --report "$report_path" --root "$package_root"

summary_path="$report_dir/surface-package-gate-summary.json"
cat > "$summary_path" <<JSON
{
  "schema": "tetra.surface.package-gate.v1",
  "status": "current",
  "release_scope": "surface-package-distribution-scoped-linux",
  "producer": "scripts/release/surface/package-gate.sh",
  "git_head": $(json_string "$git_head"),
  "version": $(json_string "$version"),
  "git_dirty": $git_dirty,
  "host_os": $(json_string "$host_os"),
  "host_arch": $(json_string "$host_arch"),
  "generated_at_utc": $(json_string "$generated_at_utc"),
  "command_line": $(json_string "$command_line"),
  "schema_under_test": "tetra.surface.package-report.v1",
  "level": "surface-package-distribution-v1",
  "package_report": "surface-package-report.json",
  "package_root": "package-root",
  "linux_archive": "package-root/$archive_name",
  "same_commit_validated": true,
  "install_smoke": true,
  "launcher_smoke": true,
  "fake_claim_rejections": [
    "unsigned macOS production package",
    "omitted package asset",
    "updater without channel signature"
  ],
  "nonclaims": [
    "Windows signed installer production",
    "macOS signed and notarized bundle production",
    "auto-update production channel",
    "multi-target desktop installer parity"
  ]
}
JSON

go run -buildvcs=false ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"
go run -buildvcs=false ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"

echo "Surface package gate reports: $report_dir"
echo "Surface package report: $report_path"
echo "Surface package artifact hashes: $report_dir/artifact-hashes.json"
