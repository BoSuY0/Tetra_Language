# Memory Production Core v1 Artifact Map

Status: validated after the final command set was rerun and recorded in `GOAL.md`.

This map links final audit claims to concrete source, docs, reports, and command evidence. It does
not convert artifacts into truth: reports are projections of compiler-owned facts.

## Core Documents

- Artifact: `docs/audits/memory/production/memory-production-core-v1-final.md`
  Purpose: Final MPC-0 through MPC-16 classification rows.
- Artifact: `docs/audits/memory/production/memory-production-core-v1-artifact-map.md`
  Purpose: This evidence map.
- Artifact: `docs/audits/memory/production/memory-production-core-v1-nonclaims.md`
  Purpose: Explicit nonclaims and overclaim boundaries.
- Artifact: `docs/design/memory/memory_production_core_v1.md`
  Purpose: Design law: facts are compiler-owned, reports are projections.
- Artifact: `docs/spec/memory/memory_report_schema_v1.md`
  Purpose: Schema and validation rules for memory report projections.
- Artifact: `docs/design/memory/memory_cost_model.md`
  Purpose: MPC-14 cost classes and dynamic-check boundaries.
- Artifact: `docs/audits/memory/islands/memory-fuzz-oracle-v1.md`
  Purpose: MPC-15 oracle categories, tiers, generator tiers, and invariants.
- Artifact: `docs/audits/memory/islands/memory-target-capability-matrix.md`
  Purpose: Target-tier evidence and no cross-target promotion rule.

## Compiler and Validator Sources

- Artifact: `compiler/internal/memoryfacts`
  Purpose: `MemoryFactGraph`, fact validation, report projection, and guards.
- Artifact: `tools/cmd/validate-memory-report`
  Purpose: CLI validation for `tetra.memory-report.v1`.
- Artifact: `compiler/internal/plir`
  Purpose: PLIR raw/memory/storage evidence vocabulary.
- Artifact: `compiler/internal/allocplan`
  Purpose: Planned storage and allocation intent evidence.
- Artifact: `compiler/internal/lower`
  Purpose: Actual lowering storage evidence.
- Artifact: `compiler/internal/validation`
  Purpose: IR and explicit-island/function-temp validation.
- Artifact: `compiler/internal/runtimeabi`
  Purpose: Runtime ABI memory bounds and allocation contracts.
- Artifact: `compiler/target` and `tools/cmd/validate-targets`
  Purpose: Target memory capability matrix and claim inflation guards.
- Artifact: `compiler/compiler_evidence_gates.go`
  Purpose: Compiler-owned oracle report builder and validator.
- Artifact: `tools/cmd/memory-fuzz-short`
  Purpose: Tier 1 deterministic memory fuzz oracle smoke.
- Artifact: `tools/cmd/validate-memory-fuzz-oracle`
  Purpose: Standalone oracle report validator.

## Report Artifacts

- Artifact: `reports/memory-production-core-v1/mpc8/memory-production-linux-x64.json`
  Purpose: Raw pointer verified-root bounds linux-x64 evidence.
- Artifact: `reports/memory-production-core-v1/mpc8/artifact-hashes.json`
  Purpose: Hashes for MPC-8 report artifacts.
- Artifact: `reports/memory-production-core-v1/mpc9/memory-production-linux-x64.json`
  Purpose: Raw slice gateway linux-x64 runtime/report evidence.
- Artifact: `reports/memory-production-core-v1/mpc9/artifact-hashes.json`
  Purpose: Hashes for MPC-9 report artifacts.
- Artifact: `reports/memory-fuzz-short/mpc15/memory-fuzz-oracle.json`
  Purpose: Tier 1 oracle report with explicit categories and invariants.
- Artifact: `reports/memory-fuzz-short/mpc15/summary.md`
  Purpose: Human-readable Tier 1 memory fuzz smoke summary.

## MPC-16 Release Gate Artifacts

Required quick gate command:

`scripts/ci/test-all.sh --quick --keep-going`

```bash
bash scripts/ci/test-all.sh \
  --quick \
  --keep-going \
  --report-dir reports/memory-production-core-v1/test-all-quick
```

Expected quick output directory:

- `reports/memory-production-core-v1/test-all-quick/summary.json`
- `reports/memory-production-core-v1/test-all-quick/summary.md`
- `reports/memory-production-core-v1/test-all-quick/logs/`

Observed quick output:

- `reports/memory-production-core-v1/test-all-quick/summary.json`
- `reports/memory-production-core-v1/test-all-quick/summary.md`
- `reports/memory-production-core-v1/test-all-quick/logs/`
- validated by:

```bash
go run ./tools/cmd/validate-test-all-summary \
  --summary reports/memory-production-core-v1/test-all-quick/summary.json \
  --report-dir reports/memory-production-core-v1/test-all-quick
```

Quick mode is evidence for the required MPC-16 command. It is not the full `full` or `stabilization`
mode. The final audit records no official benchmark result and no target parity evidence from quick
output.

The script rejects unsafe report directories: `reports/memory-production-core-v1/test-all-quick`
must not be a symlink, must stay under the workspace, and must be empty or absent before the command
is rerun.

## Final Command Matrix

- Command:

```bash
go test ./compiler/internal/memoryfacts -count=1
```

  Evidence classification: Memory fact graph and report validator unit evidence.

- Command:

```bash
go test ./compiler/internal/plir ./compiler/internal/validation \
  ./compiler/internal/allocplan ./compiler/internal/lower \
  -count=1
```

  Evidence classification: IR/planner/lowering validator evidence.

- Command:

```bash
go test ./compiler \
  -run 'Memory|Borrow|Lifetime|Alias|Unsafe|Bounds|Alloc|Region|Island|Report' \
  -count=1
```

  Evidence classification: Compiler surface evidence for memory reports and checks.

- Command:

```bash
go test ./compiler/... ./cli/... ./tools/... -count=1
```

  Evidence classification: Broad workspace Go evidence.

- Command:

```bash
bash scripts/ci/test.sh
```

  Evidence classification: CI release-suite evidence.

- Command:

```bash
bash scripts/ci/test-all.sh \
  --quick \
  --keep-going \
  --report-dir reports/memory-production-core-v1/test-all-quick
```

  Evidence classification: Quick all-suite report evidence.

- Command:

```bash
go run ./tools/cmd/validate-manifest \
  --manifest docs/generated/manifest.json
```

  Evidence classification: Manifest feature/docs gate.

- Command:

```bash
go run ./tools/cmd/verify-docs \
  --manifest docs/generated/manifest.json
```

  Evidence classification: Docs gate including final audit docs.

- Command:

```bash
git diff --check
```

  Evidence classification: Whitespace/diff hygiene.

- Command:

```bash
graphify update .
```

  Evidence classification: Code graph refresh after code changes.
