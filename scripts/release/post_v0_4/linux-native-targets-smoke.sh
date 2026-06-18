#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="$repo_root/reports/post-v0.4/linux-native-targets"

usage() {
  cat << 'USAGE'
Usage: bash scripts/release/post_v0_4/linux-native-targets-smoke.sh [--report-dir DIR]

Runs Linux native target metadata, ABI, atomic, fuzz, and brutal evidence gates
for linux-x64, linux-x86, and linux-x32 without promoting build-only targets.
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
    -h | --help)
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
: "${GOCACHE:=$repo_root/.cache/go-build-linux-native-targets}"
export GOCACHE
mkdir -p "$report_dir"

if [[ -n "${TETRA_CMD:-}" ]]; then
  read -r -a tetra_cmd <<< "$TETRA_CMD"
else
  tetra_cmd=(go run ./cli/cmd/tetra)
fi

targets_json="$report_dir/targets.json"
"${tetra_cmd[@]}" targets --format=json > "$targets_json"
go run ./tools/cmd/validate-targets --report "$targets_json"

run_target_suites() {
  local raw_target="$1"
  local stem="$2"

  "${tetra_cmd[@]}" test --target "$raw_target" --abi --report=json > "$report_dir/$stem-abi.json"
  "${tetra_cmd[@]}" test --target "$raw_target" --atomic-stress --report=json > "$report_dir/$stem-atomic-stress.json"
  "${tetra_cmd[@]}" test --target "$raw_target" --fuzz --report=json > "$report_dir/$stem-fuzz.json"
}

runner_src="$report_dir/linux-native-runner-smoke.tetra"
cat > "$runner_src" << 'TETRA'
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

run_runner_smoke() {
  local raw_target="$1"
  local stem="$2"
  local report="$report_dir/$stem-runner.json"
  local err_report="$report_dir/$stem-runner.err.json"

  if "${tetra_cmd[@]}" test --diagnostics=json --target "$raw_target" --format=json "$runner_src" > "$report" 2> "$err_report"; then
    rm -f "$err_report"
    return 0
  fi
  local code=$?
  if [[ "$code" -eq 2 ]] && grep -q 'no host fallback' "$err_report"; then
    mv "$err_report" "$report"
    return 0
  fi
  cat "$err_report" >&2
  return "$code"
}

run_target_suites x64 linux-x64
run_target_suites x86 linux-x86
run_target_suites x32 linux-x32
run_runner_smoke x64 linux-x64
run_runner_smoke x86 linux-x86
run_runner_smoke x32 linux-x32

brutal_json="$report_dir/linux-native-targets-brutal.json"
"${tetra_cmd[@]}" test --all-targets --brutal --format=json > "$brutal_json"

go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"

go run ./tools/cmd/validate-linux-native-targets \
  --targets "$targets_json" \
  --artifact-hashes "$report_dir/artifact-hashes.json" \
  --target "linux-x64:$report_dir/linux-x64-abi.json:$report_dir/linux-x64-atomic-stress.json:$report_dir/linux-x64-fuzz.json" \
  --target "linux-x86:$report_dir/linux-x86-abi.json:$report_dir/linux-x86-atomic-stress.json:$report_dir/linux-x86-fuzz.json" \
  --target "linux-x32:$report_dir/linux-x32-abi.json:$report_dir/linux-x32-atomic-stress.json:$report_dir/linux-x32-fuzz.json" \
  --runner "linux-x64:$report_dir/linux-x64-runner.json" \
  --runner "linux-x86:$report_dir/linux-x86-runner.json" \
  --runner "linux-x32:$report_dir/linux-x32-runner.json" \
  --brutal "$brutal_json"

echo "linux native target smoke reports: $report_dir"
echo "linux native target artifact hashes: $report_dir/artifact-hashes.json"
