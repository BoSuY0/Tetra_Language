#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
target=""
report_path=""
expected_version=""
expected_git_head=""

usage() {
  cat <<'USAGE'
Usage: bash scripts/release/full_platform/target-host-ui-runtime-smoke.sh [--target TARGET] [--report FILE] [--expected-version VERSION] [--expected-git-head SHA]

Runs the platform UI runtime smoke on a real target host and validates the
result. TARGET is windows-x64 or macos-x64. When TARGET is omitted, the script
detects Windows or macOS from the current host. Linux hosts are not valid
target-host evidence for Windows or macOS.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --target)
      if [[ $# -lt 2 ]]; then
        echo "error: --target requires a value" >&2
        usage >&2
        exit 2
      fi
      target="$2"
      shift 2
      ;;
    --report)
      if [[ $# -lt 2 ]]; then
        echo "error: --report requires a value" >&2
        usage >&2
        exit 2
      fi
      report_path="$2"
      shift 2
      ;;
    --expected-version)
      if [[ $# -lt 2 ]]; then
        echo "error: --expected-version requires a value" >&2
        usage >&2
        exit 2
      fi
      expected_version="$2"
      shift 2
      ;;
    --expected-git-head)
      if [[ $# -lt 2 ]]; then
        echo "error: --expected-git-head requires a value" >&2
        usage >&2
        exit 2
      fi
      expected_git_head="$2"
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

detect_target() {
  local os_name
  os_name="$(uname -s 2>/dev/null || printf unknown)"
  case "$os_name" in
    Darwin)
      printf 'macos-x64'
      ;;
    MINGW*|MSYS*|CYGWIN*)
      printf 'windows-x64'
      ;;
    *)
      case "${OS:-}" in
        Windows_NT)
          printf 'windows-x64'
          ;;
        *)
          return 1
          ;;
      esac
      ;;
  esac
}

if [[ -z "$target" ]]; then
  if ! target="$(detect_target)"; then
    echo "error: target-host UI runtime evidence requires a real Windows or macOS target host" >&2
    exit 1
  fi
fi

case "$target" in
  windows-x64|macos-x64)
    ;;
  *)
    echo "error: unsupported target-host UI runtime target: $target" >&2
    usage >&2
    exit 2
    ;;
esac

if [[ -z "$report_path" ]]; then
  case "$target" in
    windows-x64) report_path="$repo_root/reports/full-platform-ui-runtime/windows-ui-runtime.json" ;;
    macos-x64) report_path="$repo_root/reports/full-platform-ui-runtime/macos-ui-runtime.json" ;;
  esac
fi

cd "$repo_root"
mkdir -p "$(dirname "$report_path")"

if [[ -z "$expected_version" ]]; then
  expected_version="$("./tetra" version 2>/dev/null || go run ./cli/cmd/tetra version)"
fi
if [[ -z "$expected_git_head" ]]; then
  expected_git_head="$(git rev-parse HEAD)"
fi

go run ./tools/cmd/platform-ui-runtime-smoke \
  --target "$target" \
  --report "$report_path"

case "$target" in
  windows-x64)
    go run ./tools/cmd/validate-windows-ui-runtime --report "$report_path" --expected-version "$expected_version" --expected-git-head "$expected_git_head"
    ;;
  macos-x64)
    go run ./tools/cmd/validate-macos-ui-runtime --report "$report_path" --expected-version "$expected_version" --expected-git-head "$expected_git_head"
    ;;
esac

echo "target-host UI runtime report: $report_path"
