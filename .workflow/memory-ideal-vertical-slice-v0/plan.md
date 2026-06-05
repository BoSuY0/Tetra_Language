# Memory Ideal Vertical Slice v0

## Goal

Implement the immediate v0 slice from
`/home/tetra/Downloads/tetra_memory_ideal_vertical_slice_plan_20260604.md`.
The goal is not "Memory 100%"; it is a small, correlated, evidence-driven
vertical slice for:

- A0-lite baseline verification
- A1-lite correlation matrix with `MEM-REP-001`, `MEM-BORROW-001`,
  `MEM-ALIAS-001`
- B1-min representation metadata registry
- MiniMemoryModel v0
- B2a borrow through struct and optional only
- B3a minimal inout exclusivity only
- minimal report projection
- final audit docs and gates

## Success Criteria

- Baseline audit exists and is not `blocked`.
- Correlation matrix has exactly the three required rows and validates.
- Representation metadata assignment rejection is centralized through a
  registry before lowering.
- MiniMemoryModel v0 covers the required eight outcomes.
- Borrow/inout behavior is implemented only for the narrow v0 surface and
  remains conservative outside it.
- MemoryFactGraph/report rows for all three claims validate.
- Final audit and manifest updates exist.
- Focused and full verification gates pass or unrelated failures are clearly
  classified.

## Current Context

- `GOAL.md` is the canonical loop state.
- The worktree is heavily dirty with many prior compiler/docs/tooling changes.
- Graphify context has been queried for memoryfacts/report nodes:
  `BuildReportFromGraph()`, `ValidateReport()`, `FactID`, and semantics tests.
- `graphify-out/wiki/index.md` is absent; use `GRAPH_REPORT.md`, Graphify MCP,
  and concrete file reads.

## Constraints

- Communicate in Ukrainian.
- Preserve unrelated changes; never revert user/agent work.
- Use persistent Go caches under `.cache/` or `$HOME/.cache`, never `/tmp`.
- `MemoryFactGraph` is truth; reports are projections only.
- `unsafe_unknown` cannot become safe facts, noalias, bounds-check elimination,
  trusted storage, or lifetime-safe evidence.
- B1-min must precede B2/B3.
- No enum/generic/function/interface/async/actor/raw-pointer expansion in v0.
- No broad noalias claim; only narrow unique-local/sequential-inout wording.

## Risks

- The existing dirty tree may already include partial memory production work;
  integration must inspect rather than assume.
- Existing tests may cover broader surfaces than v0; avoid accidental scope
  expansion.
- Full gates may fail from unrelated dirty worktree state; classify with
  evidence rather than hiding failures.
- Report schema work must remain minimal and must not become a full migration.

## Approval Required

No approval is required for local non-destructive edits and tests requested by
the user. Ask before deleting, overwriting unrelated artifacts, force-pushing,
deploying, touching credentials, or broad destructive refactors.

## Work Packets

- `P0-baseline-docs`: read-only baseline and docs/manifest discovery.
- `P1-semantics-registry`: read-only semantics and representation metadata
  discovery.
- `P2-memoryfacts-report`: read-only MemoryFactGraph, report projection, and
  validator discovery.
- `P3-borrow-inout-surface`: read-only borrow/inout syntax, tests, and
  unsupported-surface discovery.
- `P4-final-review`: read-only spec and code review after implementation.

## Integration Policy

- Treat sub-agent outputs as leads, not truth.
- Resolve conflicts with direct repo inspection.
- Keep implementation local unless a write-enabled packet has disjoint
  ownership.
- Record accepted/rejected/conflict decisions in `final-report.md`.

## Verification

Focused:

```bash
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-memoryfacts go test ./compiler/internal/memoryfacts -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-semantics go test ./compiler/internal/semantics -run 'Representation|Borrow|Lifetime|Inout|Alias|MemoryIdeal' -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-plir-validation go test ./compiler/internal/plir ./compiler/internal/validation -run 'Borrow|Alias|MemoryIdeal|Report' -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-compiler go test ./compiler -run 'Memory|Borrow|Lifetime|Alias|Unsafe|Report' -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-tools go test ./tools/cmd/validate-memory-report ./tools/cmd/validate-memory-correlation -count=1
```

Full:

```bash
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-broad go test ./compiler/... ./cli/... ./tools/... -count=1
bash scripts/ci/test.sh
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-manifest go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-docs go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
git diff --check
graphify update .
```

## Reusable Artifacts

- Workflow packet prompts and final report under
  `.workflow/memory-ideal-vertical-slice-v0/`.
- Audit docs under `docs/audits/`.
