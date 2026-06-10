# Memory Production Core v1 Artifact Map

Status: validated after the final command set was rerun and recorded in
`GOAL.md`.

This map links final audit claims to concrete source, docs, reports, and command
evidence. It does not convert artifacts into truth: reports are projections of
compiler-owned facts.

## Core Documents

| Artifact | Purpose |
| --- | --- |
| `docs/audits/memory-production-core-v1-final.md` | Final MPC-0 through MPC-16 classification rows. |
| `docs/audits/memory-production-core-v1-artifact-map.md` | This evidence map. |
| `docs/audits/memory-production-core-v1-nonclaims.md` | Explicit nonclaims and overclaim boundaries. |
| `docs/design/memory_production_core_v1.md` | Design law: facts are compiler-owned, reports are projections. |
| `docs/spec/memory_report_schema_v1.md` | Schema and validation rules for memory report projections. |
| `docs/design/memory_cost_model.md` | MPC-14 cost classes and dynamic-check boundaries. |
| `docs/audits/memory-fuzz-oracle-v1.md` | MPC-15 oracle categories, tiers, generator tiers, and invariants. |
| `docs/audits/memory-target-capability-matrix.md` | Target-tier evidence and no cross-target promotion rule. |

## Compiler and Validator Sources

| Artifact | Purpose |
| --- | --- |
| `compiler/internal/memoryfacts` | `MemoryFactGraph`, fact validation, report projection, unsafe/cost/storage guards. |
| `tools/cmd/validate-memory-report` | CLI validation for `tetra.memory-report.v1`. |
| `compiler/internal/plir` | PLIR raw/memory/storage evidence vocabulary. |
| `compiler/internal/allocplan` | Planned storage and allocation intent evidence. |
| `compiler/internal/lower` | Actual lowering storage evidence. |
| `compiler/internal/validation` | IR and explicit-island/function-temp validation. |
| `compiler/internal/runtimeabi` | Runtime ABI memory bounds and allocation contracts. |
| `compiler/target` and `tools/cmd/validate-targets` | Target memory capability matrix and claim inflation guards. |
| `compiler/memory_fuzz_oracle_v1.go` | `tetra.memory-fuzz.oracle.v1` compiler-owned oracle report builder/validator. |
| `tools/cmd/memory-fuzz-short` | Tier 1 deterministic memory fuzz oracle smoke. |
| `tools/cmd/validate-memory-fuzz-oracle` | Standalone oracle report validator. |

## Report Artifacts

| Artifact | Purpose |
| --- | --- |
| `reports/memory-production-core-v1/mpc8/memory-production-linux-x64.json` | Raw pointer verified-root bounds linux-x64 evidence. |
| `reports/memory-production-core-v1/mpc8/artifact-hashes.json` | Hashes for MPC-8 report artifacts. |
| `reports/memory-production-core-v1/mpc9/memory-production-linux-x64.json` | Raw slice gateway linux-x64 runtime/report evidence. |
| `reports/memory-production-core-v1/mpc9/artifact-hashes.json` | Hashes for MPC-9 report artifacts. |
| `reports/memory-fuzz-short/mpc15/memory-fuzz-oracle.json` | Tier 1 oracle report with explicit categories and invariants. |
| `reports/memory-fuzz-short/mpc15/summary.md` | Human-readable Tier 1 memory fuzz smoke summary. |

## MPC-16 Release Gate Artifacts

Required quick gate command:

```bash
bash scripts/ci/test-all.sh --quick --keep-going --report-dir reports/memory-production-core-v1/test-all-quick
```

Expected quick output directory:

- `reports/memory-production-core-v1/test-all-quick/summary.json`
- `reports/memory-production-core-v1/test-all-quick/summary.md`
- `reports/memory-production-core-v1/test-all-quick/logs/`

Observed quick output:

- `reports/memory-production-core-v1/test-all-quick/summary.json`
- `reports/memory-production-core-v1/test-all-quick/summary.md`
- `reports/memory-production-core-v1/test-all-quick/logs/`
- validated by `go run ./tools/cmd/validate-test-all-summary --summary reports/memory-production-core-v1/test-all-quick/summary.json --report-dir reports/memory-production-core-v1/test-all-quick`

Quick mode is evidence for the required MPC-16 command. It is not the full
`full` or `stabilization` mode. The final audit records no official benchmark
result and no target parity evidence from quick output.

The script rejects unsafe report directories: `reports/memory-production-core-v1/test-all-quick`
must not be a symlink, must stay under the workspace, and must be empty or absent
before the command is rerun.

## Final Command Matrix

| Command | Evidence classification |
| --- | --- |
| `go test ./compiler/internal/memoryfacts -count=1` | Memory fact graph and report validator unit evidence. |
| `go test ./compiler/internal/plir ./compiler/internal/validation ./compiler/internal/allocplan ./compiler/internal/lower -count=1` | IR/planner/lowering validator evidence. |
| `go test ./compiler -run 'Memory|Borrow|Lifetime|Alias|Unsafe|Bounds|Alloc|Region|Island|Report' -count=1` | Compiler surface evidence for memory-related reports and checks. |
| `go test ./compiler/... ./cli/... ./tools/... -count=1` | Broad workspace Go evidence. |
| `bash scripts/ci/test.sh` | CI release-suite evidence. |
| `bash scripts/ci/test-all.sh --quick --keep-going --report-dir reports/memory-production-core-v1/test-all-quick` | Quick all-suite report evidence. |
| `go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json` | Manifest feature/docs gate. |
| `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json` | Docs gate including final audit docs. |
| `git diff --check` | Whitespace/diff hygiene. |
| `graphify update .` | Code graph refresh after code changes. |
