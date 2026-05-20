# v1.0 Performance Thresholds

Benchmarks are release-candidate evidence, not absolute promises. A candidate
must include the command, machine notes, `benchstat` comparison when available,
and a reviewer decision for any threshold breach.

The JSON evidence is deterministic by policy:
- no wall-clock `generated_at` fields;
- stable metric ordering by metric name;
- a summary hash (`summary.metrics_sha256`) computed from the metric rows.

## Command

```sh
go test ./compiler/... -bench='Benchmark(CompileRepresentativeExamples|FormatRepresentativeSources|GenerateAPIDocsDogfoodProjects|BinarySizeBaselines)' -run '^$' -count=5
bash scripts/release/v1_0/binary-size.sh --report docs/generated/v1_0/binary-size-thresholds.json
```

## Thresholds

- Representative compile benchmarks: no more than 15 percent slower than the
  accepted RC baseline without release-owner approval.
- Formatter benchmarks: no more than 10 percent slower than the accepted RC
  baseline.
- API docs generation benchmark: no more than 15 percent slower than the
  accepted RC baseline.
- Native binary size: no more than 10 percent larger than the accepted RC
  baseline for the same source and target.
- WASM binary size: no more than 10 percent larger than the accepted RC baseline
  for the same source and target.
- Release gate hard size caps for `examples/flow_struct_smoke.tetra`: native targets
  (`linux-x64`, `macos-x64`, `windows-x64`) must stay at or below 4 MiB, and
  WASM targets (`wasm32-wasi`, `wasm32-web`) must stay at or below 1 MiB.
- Release gate soft size caps for `examples/flow_struct_smoke.tetra`: native targets
  warn above 2 MiB, and WASM targets warn above 512 KiB.

## Evidence Format

```text
date:
git_head:
host:
go_version:
command:
baseline_artifact:
benchstat_summary:
threshold_decision:
summary.metric_count:
summary.total_iterations:
summary.max_ns_per_op:
summary.metrics_sha256:
```

The current release artifact is
`docs/generated/v1_0/performance-regression.json`, with a readable summary in
`docs/generated/v1_0/performance-regression.md`. Validate the JSON shape with:

```sh
go run ./tools/cmd/validate-performance-report --report docs/generated/v1_0/performance-regression.json
```

## Latest Local Measurement Snapshot

Captured for plan250 QA evidence on 2026-04-29 with:

```sh
go test ./compiler/... -bench='Benchmark(CompileRepresentativeExamples|FormatRepresentativeSources|GenerateAPIDocsDogfoodProjects|BinarySizeBaselines)' -run '^$' -count=1
```

Host evidence from the benchmark output: `linux/amd64`, CPU
`Intel(R) Core(TM) i9-14900HX`.

| metric | sample result |
| --- | --- |
| `BenchmarkCompileRepresentativeExamples/flow_hello-32` | `212936 ns/op` |
| `BenchmarkCompileRepresentativeExamples/core_math-32` | `375410 ns/op` |
| `BenchmarkCompileRepresentativeExamples/dogfood_cli-32` | `597881 ns/op` |
| `BenchmarkFormatRepresentativeSources/flow_hello-32` | `12716 ns/op` |
| `BenchmarkFormatRepresentativeSources/web_ui-32` | `27295 ns/op` |
| `BenchmarkGenerateAPIDocsDogfoodProjects-32` | `81425 ns/op` |
| `BenchmarkBinarySizeBaselines/linux_flow_hello-32` | `207359 ns/op`, `4113 artifact_bytes` |
| `BenchmarkBinarySizeBaselines/macos_flow_hello-32` | `230899 ns/op`, `12288 artifact_bytes` |
| `BenchmarkBinarySizeBaselines/windows_flow_hello-32` | `228565 ns/op`, `3072 artifact_bytes` |
| `BenchmarkBinarySizeBaselines/wasi_dogfood-32` | `136403 ns/op`, `231 artifact_bytes` |
| `BenchmarkBinarySizeBaselines/web_dogfood-32` | `266975 ns/op`, `131 artifact_bytes` |

This snapshot is not a multi-run release baseline. A release candidate still
needs the `-count=5` command and validated JSON report before threshold
decisions can be closed.
