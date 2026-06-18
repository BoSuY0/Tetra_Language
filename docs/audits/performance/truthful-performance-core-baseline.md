# Truthful Performance Core Baseline

Baseline ID: `tetra.truthful-performance-core.baseline.20260602.v1`

Status: accepted evidence baseline for future Ideal Master Plan slices.

This baseline freezes the current accepted state after P12.0. It is an evidence anchor, not a
feature promotion and not a marketing claim. Future plan slices must cite this baseline ID when they
compare new proof, allocation, backend, optimizer, runtime, benchmark, release, or claim-boundary
evidence against the current state.

## Captured State

### Git HEAD

- Value: `5129f2623d9639990076a7d422e56f02b0ed3254`.

### Dump Timestamp

- Value: `20260602_173943Z`.

### Dump Artifacts

- Value: `dumps/tetra_language_dump_20260602_173943Z_part_001.md`.
- Value: `dumps/tetra_language_dump_20260602_173943Z_part_002.md`.

### Manifest

- Value: `docs/generated/manifest.json`.

### Manifest SHA-256

- Value:
  `0d8f358a019d6ac9eb98212c961303bfe68ed8b4619e75f3f654185be45d59d4`.

### Graphify Report

- Value: `graphify-out/GRAPH_REPORT.md`.

### Graphify Code Graph Counts

- Value: 18811 nodes, 60398 edges, 1090 communities.

### Source Closure

- Value: `reports/master-plan-final-20260602/closure.md`.

### Dump-Visible Audit

- Value: `docs/audits/master-plan/master-plan-final-20260602.md`.

### Dump-Visible Artifact Map

- Value: `docs/audits/master-plan/master-plan-final-20260602-artifact-map.md`.

### Active Ideal Master Plan

- Value: `/home/tetra/Downloads/tetra_ideal_master_plan_20260602.md`.

## Supported Claim Boundary

This baseline supports only the conservative state described by the P12.0 audit:

| Area                                    | Baseline status          |
| --------------------------------------- | ------------------------ |
| P0 truth foundation                     | implemented              |
| P1 proof/dominance/range foundation     | implemented narrow slice |
| P2 allocation planner lowering          | implemented narrow slice |
| P3 machine IR / register backend        | implemented narrow slice |
| P4 optimizer and translation validation | implemented narrow slice |
| P5 runtime allocator evidence           | partial                  |
| P6 actor transfer / scheduler prototype | partial                  |
| P7 local web/runtime evidence           | implemented narrow slice |
| P8 benchmark discipline                 | implemented narrow slice |
| P9 layout / ABI policy                  | implemented narrow slice |
| P10 release evidence discipline         | implemented              |
| P11 verified-track seed                 | implemented narrow slice |

The words `implemented narrow slice`, `partial`, and `evidence-only` are part of the claim boundary.
They must not be collapsed into broad production claims.

## Explicit Non-Claims

This baseline does not claim:

- fastest language status;
- official TechEmpower publication;
- full formal proof of Tetra;
- self-hosting;
- full production actor scheduler/runtime;
- public semantic backend selection;
- unsafe fast mode or disabled safe checks;
- broad measured cross-language benchmark wins;
- full implicit region lowering beyond modeled/evidence slices.

## Future Plan Reference Rule

Every future Ideal Master Plan slice should cite:

`tetra.truthful-performance-core.baseline.20260602.v1`

when it:

- promotes a plan item beyond this baseline;
- changes a supported claim boundary;
- adds new proof/allocation/backend/runtime evidence;
- compares performance against previous local evidence;
- creates a release or audit artifact that depends on P0-P11 state.

If a future slice cannot cite this baseline because the baseline is stale, it must first create a
successor baseline with a new ID, fresh command evidence, fresh manifest hash, fresh dump
visibility, and fresh Graphify counts.

## P12.1 Verification Contract

The P12.1 slice is complete only after these checks pass in the current worktree:

### Verified Track Citation Test

- Command:

```bash
GOCACHE=$(pwd)/.cache/go-build-ideal-plan \
  go test ./compiler/tests/semantics \
  -run TestVerifiedTrackCitesMasterPlanAuditDocs \
  -count=1
```

- Required result: pass.

### Verify Docs

- Command:

```bash
go run ./tools/cmd/verify-docs \
  --manifest docs/generated/manifest.json
```

- Required result: pass.

### Validate Manifest

- Command:

```bash
go run ./tools/cmd/validate-manifest \
  --manifest docs/generated/manifest.json
```

- Required result: pass.

### Diff Check

- Command: `git diff --check`.
- Required result: pass.

### Graphify Update

- Command: `graphify update .`.
- Required result: pass after registry/code edits.

### Dump Creation

- Command: `go run ./create_dumps.go`.
- Required result: creates current dump artifacts.

### Dump Visibility

- Command:

```bash
rg -n \
  "truthful-performance-core-baseline" \
  "tetra.truthful-performance-core.baseline.20260602.v1" \
  dumps
```

- Required result: proves dump visibility.
