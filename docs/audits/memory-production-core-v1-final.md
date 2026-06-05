# Memory Production Core v1 Final Audit

Status: validated after the MPC-16 command set passed on 2026-06-04.

Memory Production Core v1 is scoped to supported safe-memory behavior,
conservative unsafe boundaries, compiler-owned facts, schema-versioned report
projections, target-tiered claims, cost-model classification, and oracle-backed
memory fuzz smoke evidence.

Allowed row statuses:

- `implemented`
- `implemented_narrow`
- `validated`
- `conservative`
- `rejected`
- `future`
- `explicit_non_goal`

reports are projections of `MemoryFactGraph` and adjacent compiler-owned IR,
allocation-plan, lowering, runtime-ABI, target, and validator facts. Reports are
not sources of truth.

Linked docs:

- `docs/audits/memory-production-core-v1-artifact-map.md`
- `docs/audits/memory-production-core-v1-nonclaims.md`

## Final Classification

| Row | Area | Final status | Evidence refs | Scope notes | Overclaim risk |
| --- | --- | --- | --- | --- | --- |
| MPC-0 | Baseline and gap map | `implemented` | `docs/audits/memory-production-core-v1-baseline.md`, `docs/audits/memory-production-core-v1-gap-map.md` | Establishes current surface and gaps; not a proof of future rows. | low |
| MPC-1 | Memory Fact Graph v0 | `validated` | `compiler/internal/memoryfacts`, `docs/spec/memory_report_schema_v1.md`, final command `go test ./compiler/internal/memoryfacts -count=1` | Compiler-owned fact graph is the truth source for report projection. | medium |
| MPC-2 | Memory Report Schema v1 and validator | `validated` | `tools/cmd/validate-memory-report`, `compiler/internal/memoryfacts/report.go`, `docs/spec/memory_report_schema_v1.md` | Schema validation rejects invalid projections; it does not infer facts the compiler did not own. | medium |
| Raw-bounds closure | Verified `core.alloc_bytes` roots and conservative unknown raw pointers | `validated` | `compiler/internal/runtimeabi/raw_pointer_bounds.go`, `compiler/internal/plir`, `reports/memory-production-core-v1/mpc8/memory-production-linux-x64.json` | Runtime evidence is linux-x64 scoped unless a target-specific row says otherwise. | high |
| MPC-3 | Safe representation invariant hardening | `validated` | `compiler/internal/semantics`, `compiler/tests/semantics`, `docs/audits/memory-production-core-v1-supported-surface.md` | Safe metadata assignment is rejected before lowering. | medium |
| MPC-4 | Borrow/lifetime supported surface hardening | `implemented_narrow` | `compiler/tests/safety`, `compiler/tests/semantics`, `compiler/internal/memoryfacts` | Covers documented slice/String and local supported cases, not full Rust-like lifetime parity. | high |
| MPC-5 | Mutable alias / inout conservative subset | `conservative` | `compiler/internal/memoryfacts`, `tools/cmd/validate-memory-report`, `docs/audits/memory-production-core-v1-supported-surface.md` | Unknown/maybe/call-invalidated alias state blocks noalias promotion. | high |
| MPC-6 | Provenance/resource summaries v2 | `implemented_narrow` | `compiler/internal/memoryfacts/from_plir.go`, `tools/cmd/validate-memory-report`, summary tests | Summary vocabulary covers PLIR-visible supported cases; unknown external/resource returns remain conservative. | medium |
| MPC-7 | Unsafe fact classes | `validated` | `compiler/internal/memoryfacts`, `compiler/internal/plir`, `docs/spec/unsafe.md` | `unsafe_unknown` cannot become safe provenance, noalias, trusted storage, or removed-check evidence. | high |
| MPC-8 | Raw pointer verified-root bounds | `validated` | `compiler/internal/runtimeabi`, `compiler/internal/plir`, `reports/memory-production-core-v1/mpc8/memory-production-linux-x64.json` | Verified roots are allocation-base scoped; arbitrary external raw pointers remain conservative. | high |
| MPC-9 | Raw slice gateway hardening | `validated` | `compiler/internal/runtimeabi`, `compiler/internal/lower`, `reports/memory-production-core-v1/mpc9/memory-production-linux-x64.json` | Raw slice runtime trap evidence is linux-x64 scoped; non-x64 rows remain target-tiered. | high |
| MPC-10 | Storage truth: stack/heap/explicit island | `validated` | `compiler/internal/allocplan`, `compiler/internal/lower`, `compiler/internal/validation`, `compiler/internal/memoryfacts` | Planned storage and actual lowering storage remain separate; heap fallback is not a validated stack/region claim. | high |
| MPC-11 | Function-temp implicit region narrow slice | `implemented_narrow` | `compiler/internal/allocplan`, `compiler/internal/lower`, `compiler/internal/validation`, `compiler/internal/backend/x64core` | One narrow linux-x64 function-temp region path; broad region reuse/control-flow cleanup remains future. | medium |
| MPC-12 | Actor/task/request conservative memory rules | `conservative` | `compiler/tests/safety`, `compiler/tests/semantics`, `compiler/internal/actorsafety`, `compiler/internal/parallelrt`, `compiler/internal/httprt` | Actor zero-copy move rows are evidence-only; full production actor runtime is not claimed. | high |
| MPC-13 | Target capability matrix | `validated` | `docs/audits/memory-target-capability-matrix.md`, `compiler/target`, `tools/cmd/validate-targets` | No cross-target memory production claim without target evidence. | high |
| MPC-14 | Memory cost model | `validated` | `docs/design/memory_cost_model.md`, `compiler/internal/memoryfacts`, `compiler/reports.go`, `tools/cmd/validate-memory-report` | Cost classes classify evidence; fake zero-cost or trusted unsafe optimization wording is rejected. | medium |
| MPC-15 | Memory fuzz/property/stress with oracle | `validated` | `docs/audits/memory-fuzz-oracle-v1.md`, `compiler/memory_fuzz_oracle_v1.go`, `tools/cmd/memory-fuzz-short`, `reports/memory-fuzz-short/mpc15/memory-fuzz-oracle.json` | Tier 1 is deterministic oracle smoke; random generation is not proof by itself. | medium |
| MPC-16 | Production gate and final audit | `validated` | This doc, `docs/audits/memory-production-core-v1-artifact-map.md`, `docs/audits/memory-production-core-v1-nonclaims.md`, `reports/memory-production-core-v1/test-all-quick/summary.json`, `reports/memory-production-core-v1/test-all-quick/summary.md` | Full MPC-16 command set passed; quick output remains quick evidence, not a full/stabilization or benchmark claim. | high |

## Conservative Boundaries

- Unknown unsafe memory is `conservative`, not safe.
- Unknown target runtime behavior is `conservative` or `future`, not target parity.
- Unsupported arbitrary external raw-pointer safety is `explicit_non_goal`.
- Full actor runtime production guarantees are `explicit_non_goal` outside the
  documented local/evidence-only slices.
- Full Rust-like borrow checker parity is `explicit_non_goal`.

## MPC-16 Command Evidence

The command evidence is recorded in `GOAL.md` progress and mirrored by
`docs/audits/memory-production-core-v1-artifact-map.md`. The required commands
passed:

- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-mpc16-core go test -p=1 ./compiler/internal/memoryfacts ./compiler/internal/plir ./compiler/internal/validation ./compiler/internal/allocplan ./compiler/internal/lower -count=1`
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-mpc16-compiler go test -p=1 ./compiler -run 'Memory|Borrow|Lifetime|Alias|Unsafe|Bounds|Alloc|Region|Island|Report' -count=1`
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-mpc16 go test -p=1 ./compiler/... ./cli/... ./tools/... -count=1`
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-mpc16 bash scripts/ci/test.sh`
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-mpc16-test-all bash scripts/ci/test-all.sh --quick --keep-going --report-dir reports/memory-production-core-v1/test-all-quick`
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-mpc16-test-all go run ./tools/cmd/validate-test-all-summary --summary reports/memory-production-core-v1/test-all-quick/summary.json --report-dir reports/memory-production-core-v1/test-all-quick`
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-mpc16-docs go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-mpc16-docs go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`
- `git diff --check`
- `graphify update .`
