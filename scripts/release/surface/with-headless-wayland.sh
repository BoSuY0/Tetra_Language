#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: scripts/release/surface/with-headless-wayland.sh COMMAND [ARG...]

Runs COMMAND with a headless Weston Wayland compositor when the current
environment does not already expose a usable Wayland socket.
USAGE
}

if [[ $# -eq 0 ]]; then
  usage >&2
  exit 2
fi

surface_wayland_socket_path() {
  local display="$1"
  if [[ "$display" = /* ]]; then
    printf "%s" "$display"
    return
  fi
  printf "%s/%s" "${XDG_RUNTIME_DIR:-}" "$display"
}

if [[ -n "${WAYLAND_DISPLAY:-}" && -n "${XDG_RUNTIME_DIR:-}" ]]; then
  existing_socket="$(surface_wayland_socket_path "$WAYLAND_DISPLAY")"
  if test -S "$existing_socket"; then
    exec "$@"
  fi
fi

if ! command -v weston >/dev/null 2>&1; then
  echo "weston is required for headless Wayland Surface gates" >&2
  exit 1
fi

runtime_base="${RUNNER_TEMP:-${TMPDIR:-/tmp}}"
mkdir -p "$runtime_base"
runtime_dir="$(mktemp -d "${runtime_base%/}/tetra-wayland.XXXXXX")"
chmod 700 "$runtime_dir"
weston_pid=""

cleanup_headless_wayland() {
  local code=$?
  if [[ -n "${weston_pid:-}" ]]; then
    kill "$weston_pid" 2>/dev/null || :
    wait "$weston_pid" 2>/dev/null || :
  fi
  rm -rf "$runtime_dir"
  exit "$code"
}
trap cleanup_headless_wayland EXIT INT TERM

export XDG_RUNTIME_DIR="$runtime_dir"
export WAYLAND_DISPLAY="tetra-wayland"
weston_log="$runtime_dir/weston.log"

weston --backend=headless-backend.so --socket="$WAYLAND_DISPLAY" --idle-time=0 --log="$weston_log" &
weston_pid=$!

for _ in $(seq 1 100); do
  if test -S "$XDG_RUNTIME_DIR/$WAYLAND_DISPLAY"; then
    "$@"
    exit $?
  fi
  if ! kill -0 "$weston_pid" 2>/dev/null; then
    echo "headless Weston exited before creating $XDG_RUNTIME_DIR/$WAYLAND_DISPLAY" >&2
    sed -n '1,160p' "$weston_log" >&2 || :
    exit 1
  fi
  sleep 0.1
done

echo "timed out waiting for headless Weston socket $XDG_RUNTIME_DIR/$WAYLAND_DISPLAY" >&2
sed -n '1,160p' "$weston_log" >&2 || :
exit 1
