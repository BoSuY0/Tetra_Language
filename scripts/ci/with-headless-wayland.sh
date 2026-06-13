#!/usr/bin/env bash
set -euo pipefail

socket_name="wayland-tetra-ci"

usage() {
  cat <<'USAGE'
Usage: bash scripts/ci/with-headless-wayland.sh [--socket NAME] -- COMMAND [ARG...]

Runs COMMAND with a headless Weston Wayland compositor and exports
WAYLAND_DISPLAY for Linux CI release gates that need a real UI runtime target.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --socket)
      if [[ $# -lt 2 ]]; then
        echo "with-headless-wayland: --socket requires a value" >&2
        usage >&2
        exit 2
      fi
      socket_name="$2"
      shift 2
      ;;
    --)
      shift
      break
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "with-headless-wayland: unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [[ $# -lt 1 ]]; then
  echo "with-headless-wayland: missing command" >&2
  usage >&2
  exit 2
fi

missing_packages=()
if ! command -v weston >/dev/null 2>&1; then
  missing_packages+=("weston")
fi
if ! command -v rg >/dev/null 2>&1; then
  missing_packages+=("ripgrep")
fi
if [[ "${#missing_packages[@]}" -gt 0 ]]; then
  if command -v sudo >/dev/null 2>&1 && command -v apt-get >/dev/null 2>&1; then
    sudo apt-get update
    sudo apt-get install -y "${missing_packages[@]}"
  else
    echo "with-headless-wayland: missing required tools: ${missing_packages[*]}" >&2
    exit 127
  fi
fi
if ! command -v weston >/dev/null 2>&1 || ! command -v rg >/dev/null 2>&1; then
  echo "with-headless-wayland: weston and rg are required" >&2
  exit 127
fi

runtime_parent="${RUNNER_TEMP:-${TMPDIR:-/tmp}}"
export XDG_RUNTIME_DIR="${XDG_RUNTIME_DIR:-$runtime_parent/tetra-wayland-runtime}"
mkdir -p "$XDG_RUNTIME_DIR"
chmod 700 "$XDG_RUNTIME_DIR"

weston_log="$runtime_parent/${socket_name}.weston.log"
weston --backend=headless-backend.so --socket="$socket_name" --idle-time=0 >"$weston_log" 2>&1 &
weston_pid=$!

cleanup_weston() {
  trap - EXIT
  if kill "$weston_pid" >/dev/null 2>&1; then
    if wait "$weston_pid" >/dev/null 2>&1; then
      :
    fi
  fi
}
trap cleanup_weston EXIT

waited=0
while [[ ! -S "$XDG_RUNTIME_DIR/$socket_name" && "$waited" -lt 100 ]]; do
  sleep 0.1
  waited=$((waited + 1))
done
if [[ ! -S "$XDG_RUNTIME_DIR/$socket_name" ]]; then
  if [[ -f "$weston_log" ]]; then
    cat "$weston_log" >&2
  fi
  exit 1
fi

export WAYLAND_DISPLAY="$socket_name"
"$@"
