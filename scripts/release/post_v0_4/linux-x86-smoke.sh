#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="$repo_root/reports/post-v0.4/linux-x86"

usage() {
  cat <<'USAGE'
Usage: bash scripts/release/post_v0_4/linux-x86-smoke.sh [--report-dir DIR]

Runs linux-x86 ABI, atomic, fuzz, and target metadata evidence. This preserves
build-only status until full i386 runtime/stdlib/FFI promotion evidence exists.
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
: "${GOCACHE:=$repo_root/.cache/go-build-linux-x86-smoke}"
export GOCACHE
mkdir -p "$report_dir"

if [[ -n "${TETRA_CMD:-}" ]]; then
  read -r -a tetra_cmd <<<"$TETRA_CMD"
else
  tetra_cmd=(go run ./cli/cmd/tetra)
fi

targets_json="$report_dir/targets.json"
"${tetra_cmd[@]}" targets --format=json >"$targets_json"
go run ./tools/cmd/validate-targets --report "$targets_json"

"${tetra_cmd[@]}" test --target x86 --abi --report=json >"$report_dir/linux-x86-abi.json"
"${tetra_cmd[@]}" test --target x86 --atomic-stress --report=json >"$report_dir/linux-x86-atomic-stress.json"
"${tetra_cmd[@]}" test --target x86 --fuzz --report=json >"$report_dir/linux-x86-fuzz.json"

runner_src="$report_dir/linux-x86-runner-smoke.tetra"
cat >"$runner_src" <<'TETRA'
func runner_worker() -> Int:
    return 42

test "runner arithmetic":
    expect 40 + 2 == 42

test "runner alloc memory":
    var got: Int = 0
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(4)
        let _: Int = core.store_i32(p, 42, mem)
        got = core.load_i32(p, mem)
    expect got == 42

test "runner filesystem":
    var exists: Bool = false
    unsafe:
        let cap: cap.io = core.cap_io()
        exists = core.fs_exists("README.md", cap)
    expect exists

test "runner stderr fd":
    var written: Int = 0
    unsafe:
        let cap: cap.io = core.cap_io()
        var buf: []u8 = core.make_u8(1)
        buf[0] = 69
        written = core.net_write(2, buf, 0, 1, cap)
    expect written == 1

test "runner time":
    let now: Int = core.time_now_ms()
    expect now >= 0

test "runner network socket":
    var ok: Bool = false
    unsafe:
        let cap: cap.io = core.cap_io()
        let fd: Int = core.net_socket_tcp4(cap)
        let closed: Int = core.net_close(fd, cap)
        ok = fd >= 0 && closed == 0
    expect ok

test "runner network options":
    var ok: Bool = false
    unsafe:
        let cap: cap.io = core.cap_io()
        let fd: Int = core.net_socket_tcp4(cap)
        let nonblock: Int = core.net_set_nonblocking(fd, cap)
        let reuse: Int = core.net_set_reuseport(fd, cap)
        let closed: Int = core.net_close(fd, cap)
        ok = fd >= 0 && nonblock == 0 && reuse == 0 && closed == 0
    expect ok

test "runner task join":
    let task: task.i32 = core.task_spawn_i32("runner_worker")
    let value: Int = core.task_join_i32(task)
    expect value == 42
TETRA

if "${tetra_cmd[@]}" test --diagnostics=json --target x86 --format=json "$runner_src" >"$report_dir/linux-x86-runner.json" 2>"$report_dir/linux-x86-runner.err.json"; then
  rm -f "$report_dir/linux-x86-runner.err.json"
else
  code=$?
  if [[ "$code" -eq 2 ]] && grep -q 'no host fallback' "$report_dir/linux-x86-runner.err.json"; then
    mv "$report_dir/linux-x86-runner.err.json" "$report_dir/linux-x86-runner.json"
  else
    cat "$report_dir/linux-x86-runner.err.json" >&2
    exit "$code"
  fi
fi

go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"

go run ./tools/cmd/validate-linux-native-targets \
  --targets "$targets_json" \
  --artifact-hashes "$report_dir/artifact-hashes.json" \
  --target "linux-x86:$report_dir/linux-x86-abi.json:$report_dir/linux-x86-atomic-stress.json:$report_dir/linux-x86-fuzz.json" \
  --runner "linux-x86:$report_dir/linux-x86-runner.json"

echo "linux-x86 smoke reports: $report_dir"
echo "linux-x86 artifact hashes: $report_dir/artifact-hashes.json"
