# Local Benchmark Tier 1 V1 Audit

Status: P25.0 local benchmark evidence artifact.

This audit records a local-only execution of the P20 matrix.

It does not claim:

- Tetra is the fastest language;
- an official benchmark result;
- cross-machine reproduction;
- TechEmpower publication;
- production readiness.

Primary artifact:
- Root: `reports/benchmark-vnext-memory-baseline`
- Run: `tier1-native-memory-final`
- File: `report.json`

Summary artifact:
- Root: `reports/benchmark-vnext-memory-baseline`
- Run: `tier1-native-memory-final`
- File: `summary.md`

## Classifications

### integer loops

- Classification: `comparable`.
- Reason: Tetra median 0.826 ms is within 20% of the fastest local competitor median 1.011 ms.

### slice sum

- Classification: `faster than C/C++/Rust locally`.
- Reason: Tetra median 0.511 ms is more than 20% below the fastest local competitor median 0.774 ms.

### bounds-check loops

- Classification: `comparable`.
- Reason: Tetra median 0.658 ms is within 20% of the fastest local competitor median 0.809 ms.

### function calls

- Classification: `faster than C/C++/Rust locally`.
- Reason: Tetra median 1.397 ms is more than 20% below the fastest local competitor median 2.365 ms.

### recursion

- Classification: `faster than C/C++/Rust locally`.
- Reason: Tetra median 0.358 ms is more than 20% below the fastest local competitor median 0.728 ms.

### matrix multiply

- Classification: `faster than C/C++/Rust locally`.
- Reason: Tetra median 0.391 ms is more than 20% below the fastest local competitor median 0.683 ms.

### hash table

- Classification: `faster than C/C++/Rust locally`.
- Reason: Tetra median 0.272 ms is more than 20% below the fastest local competitor median 0.825 ms.

### allocation

- Classification: `faster than C/C++/Rust locally`.
- Reason: Tetra median 0.212 ms is more than 20% below the fastest local competitor median 0.775 ms.

### region/island allocation

- Classification: `faster than C/C++/Rust locally`.
- Reason: Tetra median 0.343 ms is more than 20% below the fastest local competitor median 0.553 ms.

### JSON parse/stringify

- Classification: `invalid/inconclusive`.
- Reason: This Tier 1 run measures deterministic local helper kernels, not a full local
  service/database benchmark for this category.

### HTTP plaintext/json

- Classification: `invalid/inconclusive`.
- Reason: This Tier 1 run measures deterministic local helper kernels, not a full local
  service/database benchmark for this category.

### PostgreSQL single/multiple/update

- Classification: `invalid/inconclusive`.
- Reason: This Tier 1 run measures deterministic local helper kernels, not a full local
  service/database benchmark for this category.

### actor ping-pong

- Classification: `blocked by actor/runtime limitation`.
- Reason: Current local actor/task runtime evidence is bounded and not a production parallel
  benchmark claim. Perf blockers: actor_copy.borrowed_data_boundary.

### parallel map/reduce

- Classification: `blocked by actor/runtime limitation`.
- Reason: Current local actor/task runtime evidence is bounded and not a production parallel
  benchmark claim. Perf blockers: actor_copy.borrowed_data_boundary,
  register_spill.live_range_pressure. Actor-domain memory evidence is missing or unsupported.

### startup time

- Classification: `faster than C/C++/Rust locally`.
- Reason: Tetra median 0.204 ms is more than 20% below the fastest local competitor median 0.538 ms.

### binary size

- Classification: `comparable`.
- Reason: binary_size_bytes local evidence: Tetra=13238, C=15832, C++=15840, Rust=445728; no
  binary-size superiority or production-size claim is promoted.

### compile time

- Classification: `faster than C/C++/Rust locally`.
- Reason: Tetra compile_time_ms 9.038 is more than 20% below the fastest local competitor
  compile_time_ms 40.355.


## Required Verification

```bash
report_root="reports/benchmark-vnext-memory-baseline"
report_dir="$report_root/tier1-native-memory-final"
go run ./tools/cmd/validate-local-benchmark-tier1 \
  --report "$report_dir/report.json"
go run ./tools/cmd/verify-docs \
  --manifest docs/generated/manifest.json
go run ./tools/cmd/validate-manifest \
  --manifest docs/generated/manifest.json
git diff --check
graphify update .
go test ./compiler/... ./cli/... ./tools/... -count=1
```
