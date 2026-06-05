# Tetra Memory Production Core v1

## Goal

Implement `/home/tetra/Downloads/tetra_memory_production_core_v1_agent_plan_20260603.md` as an auditable sequence of Memory Production Core v1 slices.

Current slice: MPC-16 Production gate and final audit.

## Success Criteria

- MPC-0 baseline, gap map, and supported-surface docs exist and make unsupported/non-goal claims explicit.
- `compiler/internal/memoryfacts` owns memory fact graph v0, stable enums, validation, report projection, and focused tests.
- `docs/spec/memory_report_schema_v1.md` and `tools/cmd/validate-memory-report` enforce schema-v1 report rules.
- Raw bounds closure distinguishes verified `core.alloc_bytes` roots from unknown external raw pointers and reports validated rejections/conservative unknowns.
- Required focused and broad verification commands are run or failures are classified with evidence.
- MPC-13 prevents cross-target memory claim inflation with a target capability matrix and validator guards.
- MPC-14 defines memory cost classes, projects `cost_class` into memory/performance reports, and rejects fake zero-cost or trusted unsafe optimization claims.
- MPC-15 adds oracle-backed memory fuzz/property/stress tiers without treating random generation as proof.
- MPC-16 closes Memory Production Core v1 with dump-visible final audit docs, artifact map, explicit nonclaims, and release-gate command evidence.

## Current Context

- Active `/goal` objective asks for the full external plan and sub-agents.
- Repo worktree is heavily dirty; many memory/proof/runtime files are already changed or untracked.
- Existing `GOAL.md` was for a completed proof/validation audit and has been replaced with this active Memory Production Core contract.
- Graphify report exists at `graphify-out/GRAPH_REPORT.md` and was built from commit `5129f2623d9639990076a7d422e56f02b0ed3254`.

## Constraints

- Ukrainian user communication.
- Preserve unrelated dirty work.
- Keep facts compiler-owned; reports only project existing graph facts.
- No report/debug flag may change safe-program semantics.
- Use persistent Go caches outside `/tmp`.
- Use TDD for implementation slices when practical.
- Sub-agent packets must be bounded, disjoint, and evidence-based.

## Risks

- Dirty tree may already contain partial implementations; integration must distinguish existing work from new work.
- Full MPC-0..MPC-16 is too large for one safe patch; current slice must finish before later slices.
- Broad test commands may expose unrelated failures from the dirty worktree.
- Report/validator plumbing can accidentally reconstruct truth outside the compiler-owned graph.

## Approval Required

No approval needed for local non-destructive edits, tests, docs, and workflow artifacts.

Approval would be required before destructive git operations, mass renames, deployment/publishing, external writes, or touching credentials.

## Work Packets

- P1 baseline-current-state: read-only audit of current memory/proof/report/bounds capabilities against MPC-0 rows.
- P2 memoryfacts-design-review: read-only inspection of existing PLIR/allocplan/validation/report paths for graph integration points.
- P3 raw-bounds-runtime-path: read-only inspection of `core.alloc_bytes`, `ptr_add`, load/store, raw-slice, and target behavior.
- P4 docs-manifest-gates: read-only inspection of docs manifest, validators, and release gate conventions for dump-visible artifacts.
- P5 implementation-current-slice: orchestrator-owned TDD implementation and integration.
- P6 final-review-verification: spec review, code review, workflow verification, and final gates.
- MPC12-S1 semantic-boundary-audit: read-only audit of actor/task/request semantic gates and RED-test locations.
- MPC12-S2 report-runtime-claims-audit: read-only audit of memoryfacts/report/docs/runtime claims for actor/task zero-copy and region transfer conservatism.
- MPC13-S1 target-evidence-audit: read-only audit of target/runtime evidence sources and existing validators.
- MPC13-S2 docs-manifest-audit: read-only audit of target capability docs/manifest ownership and claim wording.
- MPC14-S1 cost-report-validator-audit: read-only audit of memory report schema/projection and cost-class validation insertion points.
- MPC14-S2 perf-docs-claim-audit: read-only audit of performance blocker reports, docs/manifest gates, and optimization claim wording.
- MPC15-S1 fuzz-oracle-tooling-audit: read-only audit of existing fuzz/property/differential tooling and oracle API insertion points.
- MPC15-S2 memory-report-invariant-audit: read-only audit of memory production reports, invariant validation, and docs/manifest gates.
- MPC16-S1 final-audit-classification-audit: read-only audit of MPC-0..MPC-16 row classification coverage and final audit doc requirements.
- MPC16-S2 artifact-map-release-gate-audit: read-only audit of artifact map, nonclaims, docs/manifest gates, and required release command evidence.

## Integration Policy

- Treat sub-agent reports as evidence leads, not truth.
- Reconcile every accepted finding against local files and command output.
- If packets disagree, inspect authoritative source code/docs before deciding.
- Keep implementation local unless a later write-enabled packet has a disjoint file scope.

## Verification

Fast:

- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-mpc16-final go test -p=1 ./tools/cmd/verify-docs ./tools/cmd/validate-manifest -run 'MemoryProduction|Final|Artifact|Nonclaim|Docs|Manifest' -count=1`
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-mpc16-core go test -p=1 ./compiler/internal/memoryfacts ./compiler/internal/plir ./compiler/internal/validation ./compiler/internal/allocplan ./compiler/internal/lower -count=1`
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-mpc16-compiler go test -p=1 ./compiler -run 'Memory|Borrow|Lifetime|Alias|Unsafe|Bounds|Alloc|Region|Island|Report' -count=1`

Final:

- `go test ./compiler/internal/memoryfacts -count=1`
- `go test ./tools/cmd/validate-memory-report -count=1`
- `go test ./compiler -run 'Memory|Raw|Unsafe|Bounds|Report' -count=1`
- `go test ./compiler/... ./cli/... ./tools/... -count=1`
- `bash scripts/ci/test.sh`
- `bash scripts/ci/test-all.sh --quick --keep-going --report-dir reports/memory-production-core-v1/test-all-quick`
- `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`
- `go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`
- `git diff --check`
- `graphify update .`

## Reusable Artifacts

- `.workflow/tetra-memory-production-core-v1/packets/`
- `.workflow/tetra-memory-production-core-v1/results/`
- `.workflow/tetra-memory-production-core-v1/final-report.md`
