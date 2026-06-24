reviewed_commit: 8f7529505a13b5da72fbc0c34c5bb110541c020f

reviewer_agent: SUBAGENT-A / gpt-5.5 xhigh worker

reviewed_paths:
- docs/spec/memory/memory_core_v2.md
- docs/spec/memory/memory_report_schema_v1.md
- compiler/compiler_facade.go
- compiler/compiler_reports.go
- compiler/internal/memorypipeline/state.go
- compiler/internal/memorypipeline/lowering.go
- compiler/internal/memorypipeline/digest.go
- compiler/internal/memoryfacts/facts.go
- compiler/internal/memoryfacts/graph.go
- compiler/internal/memoryfacts/snapshot.go
- compiler/internal/memoryfacts/report.go
- compiler/internal/memoryfacts/fromplir/from_plir.go
- compiler/internal/memoryfacts/fromallocplan/from_allocplan.go
- compiler/internal/memoryfacts/fromlowering/from_lowering.go
- compiler/internal/allocplan/build.go
- compiler/internal/allocplan/plan.go
- compiler/internal/allocplan/verify.go
- compiler/internal/lower/lower_core.go
- compiler/internal/lower/lower_planned.go
- compiler/internal/lower/lower_expressions.go
- compiler/internal/loweringevidence/evidence.go
- compiler/internal/opt/opt_core.go
- compiler/cmd/validate-memory-report/main.go
- tools/validators/memorycorev2/report.go
- tools/cmd/validate-memory-core-v2/main.go
- scripts/release/memory/memory-core-v2-gate.sh

commands_executed:
- `pwd && git rev-parse HEAD && git status --porcelain=v1 --untracked-files=all && find .. -name AGENTS.md -print` -> confirmed worktree path and reviewed commit `8f7529505a13b5da72fbc0c34c5bb110541c020f`; `git status` output was empty. The trailing `find ..` hit a permission-denied path outside the worktree and was not used as evidence.
- `sed -n '1,240p' AGENTS.md` -> read local instructions, including review/reporting and Go cache constraints.
- `git branch --show-current && git rev-parse --show-toplevel && git log --oneline -1 --decorate` -> confirmed branch `stabilize/memory-core-v2`, expected worktree root, and HEAD at the reviewed commit.
- `git status --porcelain=v1 --untracked-files=all` -> clean before writing this review file.
- `test -f graphify-out/GRAPH_REPORT.md && sed -n '1,220p' graphify-out/GRAPH_REPORT.md || true` -> no graphify report artifact was present in this worktree.
- `rg -n "MemoryFactGraph|MemoryFact|allocation plan|lowered_artifact_id|proof_id|report" compiler tools docs/spec/memory scripts/release/memory` -> located Memory Core v2 compiler, report, validator, and release-gate surfaces.
- `rg --files compiler tools docs/spec/memory scripts/release/memory` -> enumerated candidate review files.
- `rg --files compiler/internal compiler/cmd tools/cmd tools/validators docs/spec/memory scripts/release/memory | rg '(memoryfacts|allocationplan|memorypipeline|validate-memory|memorycorev2|memory_core|memory-report|lower|buildapi|cmd/tetra|release/memory|spec/memory)'` -> narrowed reviewed paths to canonical memory facts, allocation plan, lowering, validators, reports, and specs.
- `rg -n "func LowerModules|LowerModules\\(" compiler/compiler_facade.go compiler/internal/lower/lower_core.go compiler/tests/runtime/linker_test.go` -> found the public and internal multi-module lowering path that bypasses the canonical memory pipeline.
- `nl -ba compiler/compiler_facade.go | sed -n '111,162p'` -> confirmed `Lower` and `LowerModule` use `memorypipeline.Build`/`LowerPlannedProgram`/`ApplyLowering`, while `LowerModules` delegates directly to `lower.LowerModules`.
- `nl -ba compiler/internal/lower/lower_core.go | sed -n '3528,3582p'` -> confirmed internal `lower.LowerModules` calls `lowerCheckedFuncWithOptions(..., Options{}, nil, ...)` without a memory fact graph, allocation plan, or lowering evidence.
- `nl -ba compiler/internal/memorypipeline/state.go | sed -n '1,140p'` -> verified semantic facts flow through PLIR, `fromplir.Build`, canonical graph snapshot, and `allocplan.Build`.
- `nl -ba compiler/internal/memorypipeline/lowering.go | sed -n '1,230p'` -> verified lowering applies explicit evidence rows to graph facts, rejects missing evidence rows, verifies lowered plan, and blocks invalid validated storage.
- `nl -ba compiler/internal/allocplan/build.go | sed -n '1,240p'` and `nl -ba compiler/internal/allocplan/build.go | sed -n '400,520p'` -> verified allocation planning consumes the graph snapshot, requires allocation-site facts, sorts source facts, and falls back conservatively when proof is missing or unsafe.
- `nl -ba compiler/internal/lower/lower_planned.go | sed -n '1,260p'` -> verified planned lowering requires `VerifyPlanned`, consumes the explicit plan, and emits per-allocation lowering evidence.
- `nl -ba compiler/internal/memoryfacts/report.go | sed -n '1,280p'` -> verified report projection and validation are derived from graph facts and enforce deterministic row ordering and no overclaims.
- `nl -ba compiler/compiler_reports.go | sed -n '230,330p'` -> verified explain/report emission projects from the same memory state and validates report projection.
- `nl -ba compiler/internal/opt/opt_core.go | sed -n '1,240p'` and `rg -n "requireMemoryProofs|MemoryFacts|proof" compiler/internal/opt/opt_core.go` -> verified optimizer memory decisions require current valid proofs and reject missing/stale/unsafe proof states.
- `nl -ba compiler/cmd/validate-memory-report/main.go | sed -n '1,260p'` -> verified the standalone report validator rejects schema/order/storage/proof mismatches and can cross-check allocation reports.
- `nl -ba tools/validators/memorycorev2/report.go | sed -n '1,260p'` -> verified release evidence validator checks decision parity, report flags, digests, route counts, nonclaims, and `release_security_review_status=pending_final_rc`.
- `nl -ba scripts/release/memory/memory-core-v2-gate.sh | sed -n '1,260p'` -> verified the release gate uses repo-local Go caches and runs focused compiler/runtime/validator checks.
- `mkdir -p .cache/go-build-review-a .cache/go-tmp-review-a && GOTELEMETRY=off GOCACHE="$(pwd)/.cache/go-build-review-a" GOTMPDIR="$(pwd)/.cache/go-tmp-review-a" go test ./compiler/internal/memoryfacts ./compiler/internal/memorypipeline ./compiler/internal/allocplan ./compiler/internal/lower ./compiler/cmd/validate-memory-report ./tools/validators/memorycorev2 ./tools/cmd/validate-memory-core-v2 -count=1` -> passed: all listed packages `ok`; `tools/cmd/validate-memory-core-v2` had no test files.
- `GOTELEMETRY=off GOCACHE="$(pwd)/.cache/go-build-review-a" GOTMPDIR="$(pwd)/.cache/go-tmp-review-a" go test ./compiler -run 'TestReportFlagsDoNotChangeBorrowedReturnFailure|TestBuildCommandEmitMemoryReportWritesSchemaV1|TestValidateMemoryReportForEmissionRejectsAlteredProjection|TestValidateMemoryReportForEmissionRejectsDroppedProjectedFact|TestValidateAllocationPlanReportRejectsMismatch|TestT13ReleaseOptimizeAdvancesCanonicalMemoryStateThroughOptimizer|TestT13NonReleaseAdvancesWithoutFabricatedOptimizerFacts|TestCompilerReportsDeterministicAcrossJobs' -count=1` -> passed: `ok tetra_language/compiler 0.051s`.

findings:
- id: A-001
  severity: medium
  title: Public multi-module lowering bypasses the canonical Memory Core v2 pipeline
  evidence: `compiler/compiler_facade.go` exposes `LowerModules(checked []*CheckedProgram)` as a direct call to `lower.LowerModules(checked)`, while neighboring `Lower` and `LowerModule` build a `memorypipeline.State`, lower via `LowerPlannedProgram`, and apply lowering evidence. `compiler/internal/lower/lower_core.go` implements `LowerModules` by calling `lowerCheckedFuncWithOptions(..., Options{}, nil, ...)`, so that route has no canonical `memoryfacts.Graph`, no `allocplan.Plan`, no per-allocation lowering evidence, and no validator handoff. Memory-sensitive lowering helpers in `compiler/internal/lower/lower_expressions.go` depend on `l.allocationPlan`; the nil-plan route is therefore a separate conservative/direct lowering path rather than the canonical evidence-backed route.
  impact: The reviewed normal compiler build/report path appears to use the canonical state and validators, so this does not prove a release-build decision divergence. It does leave an exported compiler API and internal helper capable of producing lowered modules outside the reviewed Memory Core v2 decision chain, which weakens the "absence of parallel/shadow memory decisions" claim for the full compiler surface.
  reproduction: Run `rg -n "func LowerModules|LowerModules\\(" compiler/compiler_facade.go compiler/internal/lower/lower_core.go compiler/tests/runtime/linker_test.go`, then inspect `compiler/compiler_facade.go:143` through `compiler/compiler_facade.go:161` and `compiler/internal/lower/lower_core.go:3532` through `compiler/internal/lower/lower_core.go:3579`.
  required_fix: Make public `compiler.LowerModules` mirror `Lower`/`LowerModule`: build one canonical `memorypipeline.State` for the checked programs, lower through `lower.LowerPlannedProgram` with `state.Plan`, apply lowering with `state.ApplyLowering`, run the same allocation/bounds validation used by the compiler build path where applicable, and return the grouped modules from that result. Alternatively, remove or deprecate the public unplanned path and restrict the internal helper to explicit test-only use. Add a regression test proving `LowerModules` cannot lower memory-sensitive programs without the canonical allocation plan/evidence path.

severity: medium

reproduction:
- See finding A-001 reproduction commands for the only issue found.
- Focused validation can be reproduced with the two `go test` commands listed under `commands_executed`, using repo-local `GOCACHE` and `GOTMPDIR`.

required_fix:
- Required for A-001: route `compiler.LowerModules` through the canonical Memory Core v2 pipeline or remove/deprecate the unplanned public lowering surface; add regression coverage for that route.

unresolved_risks:
- The full release gate script was inspected but not executed end-to-end in this review; focused compiler, memory pipeline, lowering, report, and validator tests were executed instead.
- Existing repo-local `.cache` directories from other work were present and ignored by git; this review did not modify or clean unrelated cache directories.
- Human security review remains out of scope and should remain `release_security_review_status=pending_final_rc`.
- Until A-001 is fixed or explicitly scoped out, the full exported compiler surface still has a non-canonical multi-module lowering route.

verdict: PASS_WITH_NONBLOCKING_FINDINGS
