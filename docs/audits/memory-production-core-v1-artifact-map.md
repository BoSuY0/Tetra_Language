# Memory Production Core v1 Artifact Map

Status: validated after the final command set was rerun and recorded in
`GOAL.md`.

This map links final audit claims to concrete source, docs, reports, and command
evidence. It does not convert artifacts into truth: reports are projections of
compiler-owned facts.

## Core Documents

### Final Audit

- Artifact: `docs/audits/memory-production-core-v1-final.md`.
- Purpose: final MPC-0 through MPC-16 classification rows.

### Artifact Map

- Artifact: `docs/audits/memory-production-core-v1-artifact-map.md`.
- Purpose: this evidence map.

### Nonclaims

- Artifact: `docs/audits/memory-production-core-v1-nonclaims.md`.
- Purpose: explicit nonclaims and overclaim boundaries.

### Design Law

- Artifact: `docs/design/memory_production_core_v1.md`.
- Purpose: facts are compiler-owned, reports are projections.

### Report Schema

- Artifact: `docs/spec/memory_report_schema_v1.md`.
- Purpose: schema and validation rules for memory report projections.

### Cost Model

- Artifact: `docs/design/memory_cost_model.md`.
- Purpose: MPC-14 cost classes and dynamic-check boundaries.

### Fuzz Oracle

- Artifact: `docs/audits/memory-fuzz-oracle-v1.md`.
- Purpose: MPC-15 oracle categories, tiers, generator tiers, and invariants.

### Target Capability Matrix

- Artifact: `docs/audits/memory-target-capability-matrix.md`.
- Purpose: target-tier evidence and no cross-target promotion rule.

## Compiler and Validator Sources

### Memory Facts

- Artifact: `compiler/internal/memoryfacts`.
- Purpose: `MemoryFactGraph`, fact validation, report projection, and
  unsafe/cost/storage guards.

### Memory Report Validator

- Artifact: `tools/cmd/validate-memory-report`.
- Purpose: CLI validation for `tetra.memory-report.v1`.

### PLIR

- Artifact: `compiler/internal/plir`.
- Purpose: PLIR raw/memory/storage evidence vocabulary.

### Allocation Planner

- Artifact: `compiler/internal/allocplan`.
- Purpose: planned storage and allocation intent evidence.

### Lowering

- Artifact: `compiler/internal/lower`.
- Purpose: actual lowering storage evidence.

### Validation

- Artifact: `compiler/internal/validation`.
- Purpose: IR and explicit-island/function-temp validation.

### Runtime ABI

- Artifact: `compiler/internal/runtimeabi`.
- Purpose: runtime ABI memory bounds and allocation contracts.

### Target Capability

- Artifact: `compiler/target` and `tools/cmd/validate-targets`.
- Purpose: target memory capability matrix and claim inflation guards.

### Fuzz Oracle Builder

- Artifact: `compiler/memory_fuzz_oracle_v1.go`.
- Purpose: `tetra.memory-fuzz.oracle.v1` compiler-owned oracle report
  builder/validator.

### Short Fuzz Smoke

- Artifact: `tools/cmd/memory-fuzz-short`.
- Purpose: Tier 1 deterministic memory fuzz oracle smoke.

### Oracle Validator

- Artifact: `tools/cmd/validate-memory-fuzz-oracle`.
- Purpose: standalone oracle report validator.

## Report Artifacts

### MPC-8 Runtime Report

- Artifact: `reports/memory-production-core-v1/mpc8/memory-production-linux-x64.json`.
- Purpose: raw pointer verified-root bounds linux-x64 evidence.

### MPC-8 Artifact Hashes

- Artifact: `reports/memory-production-core-v1/mpc8/artifact-hashes.json`.
- Purpose: hashes for MPC-8 report artifacts.

### MPC-9 Runtime Report

- Artifact: `reports/memory-production-core-v1/mpc9/memory-production-linux-x64.json`.
- Purpose: raw slice gateway linux-x64 runtime/report evidence.

### MPC-9 Artifact Hashes

- Artifact: `reports/memory-production-core-v1/mpc9/artifact-hashes.json`.
- Purpose: hashes for MPC-9 report artifacts.

### MPC-15 Oracle Report

- Artifact: `reports/memory-fuzz-short/mpc15/memory-fuzz-oracle.json`.
- Purpose: Tier 1 oracle report with explicit categories and invariants.

### MPC-15 Summary

- Artifact: `reports/memory-fuzz-short/mpc15/summary.md`.
- Purpose: human-readable Tier 1 memory fuzz smoke summary.

## MPC-16 Release Gate Artifacts

Required quick gate command:

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

Quick mode is evidence for the required MPC-16 command. It is not the full
`full` or `stabilization` mode. The final audit records no official benchmark
result and no target parity evidence from quick output.

The script rejects unsafe report directories: `reports/memory-production-core-v1/test-all-quick`
must not be a symlink, must stay under the workspace, and must be empty or absent
before the command is rerun.

## Final Command Matrix

### Memory Facts Unit Check

- Command: `go test ./compiler/internal/memoryfacts -count=1`.
- Evidence classification: memory fact graph and report validator unit evidence.

### IR/Planner/Lowering Check

- Command:

```bash
go test ./compiler/internal/plir \
  ./compiler/internal/validation \
  ./compiler/internal/allocplan \
  ./compiler/internal/lower \
  -count=1
```

- Evidence classification: IR/planner/lowering validator evidence.

### Compiler Memory Surface Check

- Command:

```bash
go test ./compiler \
  -run 'Memory|Borrow|Lifetime|Alias|Unsafe|Bounds|Alloc|Region|Island|Report' \
  -count=1
```

- Evidence classification: compiler surface evidence for memory-related
  reports and checks.

### Broad Go Check

- Command: `go test ./compiler/... ./cli/... ./tools/... -count=1`.
- Evidence classification: broad workspace Go evidence.

### CI Release Suite

- Command: `bash scripts/ci/test.sh`.
- Evidence classification: CI release-suite evidence.

### Quick All-Suite Report

- Command:

```bash
bash scripts/ci/test-all.sh \
  --quick \
  --keep-going \
  --report-dir reports/memory-production-core-v1/test-all-quick
```

- Evidence classification: quick all-suite report evidence.

### Manifest Gate

- Command: `go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`.
- Evidence classification: manifest feature/docs gate.

### Docs Gate

- Command: `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`.
- Evidence classification: docs gate including final audit docs.

### Diff Hygiene

- Command: `git diff --check`.
- Evidence classification: whitespace/diff hygiene.

### Graph Refresh

- Command: `graphify update .`.
- Evidence classification: code graph refresh after code changes.
