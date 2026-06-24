# optimizer-proof-review

reviewed_commit: 8f7529505a13b5da72fbc0c34c5bb110541c020f

reviewer_agent: SUBAGENT-C / gpt-5.5-xhigh worker

reviewed_paths:
- `compiler/compiler_facade.go`
- `compiler/compiler_t13_optimizer_test.go`
- `compiler/internal/opt/opt_core.go`
- `compiler/internal/opt/opt_t13_memory_test.go`
- `compiler/internal/opt/opt_suite_test.go`
- `compiler/internal/memoryfacts/snapshot.go`
- `compiler/internal/memoryfacts/graph.go`
- `compiler/internal/memoryfacts/fromoptimizer/from_optimizer.go`
- `compiler/internal/memoryfacts/fromoptimizer/from_optimizer_t13_test.go`
- `compiler/internal/validation/validation.go`
- `compiler/internal/validation/validation_translation.go`
- `compiler/internal/islandkernel/kernel.go`
- `compiler/internal/machine`
- `compiler/internal/allocplan`
- `compiler/internal/lower`
- `docs/spec/memory/memory_core_v2.md`
- `tools/validators/memorycorev2/report.go`
- `tools/cmd/validate-memory-core-v2`

commands_executed:
- `git rev-parse HEAD` -> `8f7529505a13b5da72fbc0c34c5bb110541c020f`.
- `git status --porcelain=v1 --untracked-files=all` -> clean before review file creation.
- `find .. -name AGENTS.md -print` -> found this worktree's `AGENTS.md`; one unrelated sibling worktree path returned permission denied.
- `test -f graphify-out/GRAPH_REPORT.md && sed -n '1,220p' graphify-out/GRAPH_REPORT.md || true` -> no graphify report in this worktree.
- `rg -n "Proof|proof|Memory|memory|bounds|noalias|scalar|sinking|rewrite|determin" compiler/internal/opt compiler/internal/memoryfacts docs/spec/memory tools` -> located optimizer, memoryfacts, spec, and validator surfaces.
- `rg --files compiler/internal/opt compiler/internal/memoryfacts docs/spec/memory tools` -> enumerated review-scope files.
- Targeted `rg`/`nl -ba` inspection over `compiler/internal/opt/opt_core.go`, `compiler/internal/opt/opt_t13_memory_test.go`, `compiler/internal/memoryfacts/snapshot.go`, `compiler/internal/memoryfacts/graph.go`, `compiler/internal/memoryfacts/fromoptimizer`, `compiler/internal/validation`, `compiler/compiler_facade.go`, `compiler/compiler_t13_optimizer_test.go`, `compiler/internal/islandkernel/kernel.go`, `docs/spec/memory/memory_core_v2.md`, and `tools/validators/memorycorev2/report.go`.
- `GOCACHE=/home/tetra/.codex/worktrees/Tetra_Language-stabilize-memory-core-v2/.cache/go-build-review-c GOTMPDIR=/home/tetra/.codex/worktrees/Tetra_Language-stabilize-memory-core-v2/.cache/go-tmp-review-c go test ./compiler/internal/opt ./compiler/internal/memoryfacts/... ./compiler/internal/validation ./tools/validators/memorycorev2 ./tools/cmd/validate-memory-core-v2` -> pass.
- `GOCACHE=/home/tetra/.codex/worktrees/Tetra_Language-stabilize-memory-core-v2/.cache/go-build-review-c GOTMPDIR=/home/tetra/.codex/worktrees/Tetra_Language-stabilize-memory-core-v2/.cache/go-tmp-review-c go test ./compiler -run 'TestT13(ReleaseOptimizeAdvancesCanonicalMemoryStateThroughOptimizer|NonReleaseAdvancesWithoutFabricatedOptimizerFacts)$'` -> pass.
- `GOCACHE=/home/tetra/.codex/worktrees/Tetra_Language-stabilize-memory-core-v2/.cache/go-build-review-c GOTMPDIR=/home/tetra/.codex/worktrees/Tetra_Language-stabilize-memory-core-v2/.cache/go-tmp-review-c go test ./compiler/internal/islandkernel ./compiler/internal/machine ./compiler/internal/allocplan ./compiler/internal/lower -run 'Test(.*NoAlias.*|VectorU8x16CopyLoopFromStackIRRequiresRangeNoAliasSafeUnalignedTailAndFallback|VectorI32x4SliceSumLoopFromStackIRUsesSafeUnalignedTailAndScalarFallback|VectorI32x4MapAddConstFromStackIRRequiresRangeSafeUnalignedTailAndFallback|VectorU8x16MemsetZeroHelperFromStackIRRequiresRangeSafeUnalignedTailAndFallback|PlannerStackLowersNonEscapingCopyOfFixedLocalView|PlannerEliminatesScalarReplacedTinyConstantIndexSlice|LowerScalarReplacementEliminatesTinyConstantIndexSlice|LowerCopyScalarReplacementRequiresDirectConstantUses|LowerUnusedCopyEliminatesFreshAllocation|LowerStackAllocationForFixedNoEscapeSlice|LowerNonEscapingCopyOfStackViewUsesStackStorage)$'` -> pass; `islandkernel` regex matched no top-level tests.
- `GOCACHE=/home/tetra/.codex/worktrees/Tetra_Language-stabilize-memory-core-v2/.cache/go-build-review-c GOTMPDIR=/home/tetra/.codex/worktrees/Tetra_Language-stabilize-memory-core-v2/.cache/go-tmp-review-c go test ./compiler/internal/islandkernel` -> pass.
- `GOCACHE=/home/tetra/.codex/worktrees/Tetra_Language-stabilize-memory-core-v2/.cache/go-build-review-c go clean -cache` -> ran after Go evidence commands.

findings:

C-001: Manager proof-decision validation does not itself prove that nonempty proof IDs are canonical.

severity: medium

reproduction:
- Code inspection shows `validateMemoryDecisionEvidence` rejects `rewrite_applied` memory decisions only when `ProofIDs` is empty (`compiler/internal/opt/opt_core.go:2883`). It does not resolve those IDs through `memoryfacts.Snapshot`.
- Canonical resolution exists in `PassContext.requireMemoryProofs` (`compiler/internal/opt/opt_core.go:2371`) and is used by current `loop-canonicalization` and `licm-pure-invariant` pass bodies, but it is not enforced by the manager for every memory rewrite decision.
- Existing negative coverage rejects the missing-proof-ID case (`compiler/internal/opt/opt_t13_memory_test.go:143`) and invalidated/unsafe facts for the current loop pass, but I did not find a negative test for a custom pass that emits `DecisionCodeRewriteApplied` with `ProofIDs: []string{"proof:bogus"}` under `RunWithOptions(... MemoryFacts: snapshot ...)`.

required_fix:
- Add manager-level validation for memory rewrite decisions when `MemoryContext.Enabled`: every proof ID should resolve through the current `memoryfacts.Snapshot`, be validated, be non-unsafe, match the expected proof kind/category, and populate `ProofFactIDs`; otherwise fail or downgrade to a proof rejection decision.
- Add a regression test using a contract test pass with a bogus nonempty proof ID to prove the manager cannot be bypassed by a pass that forgets to call `requireMemoryProofs`.

C-002: Standalone `opt.Manager.Run` disables canonical memory proof resolution for proof-sensitive passes.

severity: low

reproduction:
- `Manager.Run` delegates to `RunWithOptions` with empty options (`compiler/internal/opt/opt_core.go:2550`), which creates a disabled `MemoryContext` for an empty snapshot (`compiler/internal/opt/opt_core.go:2366`).
- `requireMemoryProofs` returns success when memory context is absent or disabled (`compiler/internal/opt/opt_core.go:2378`), so standalone tests/callers can run `LoopCanonicalizationPass` or `LICMPureInvariantPass` using only IR proof IDs plus `CheckBoundsProofs`, not canonical snapshot resolution.
- Production build wiring does use the canonical path: `compiler/compiler_facade.go:888` obtains `state.Snapshot()`, runs `optimizer.NewManager().RunWithOptions(... Options{MemoryFacts: snapshot} ...)`, converts `fromoptimizer.Delta`, and applies it back to `memorypipeline.State`. `compiler/compiler_t13_optimizer_test.go:10` covers release optimize advancing canonical state through optimizer.

required_fix:
- Either make `Run` reject passes with `RequiredProofKinds` / memory rewrite categories unless `Options.MemoryFacts` is supplied, or clearly mark `Run` as noncanonical/test-only and add a negative test that proof-sensitive passes skip or fail when canonical memory facts are absent.

Evidence by review topic:

- Passes that consume memory proofs: `loop-canonicalization` and `licm-pure-invariant` declare `RequiredProofKinds: []ProofBounds` and call `requireMemoryProofs` before rewriting (`compiler/internal/opt/opt_core.go:1594`, `compiler/internal/opt/opt_core.go:1924`, `compiler/internal/opt/opt_core.go:1664`, `compiler/internal/opt/opt_core.go:1984`).
- Passes that preserve proofs: current registered optimizer passes are `basic-scalar`, `sccp-constant-branch`, `mem2reg-single-assignment`, `inline-small-pure`, `loop-canonicalization`, and `licm-pure-invariant`; each declares `PreservedProofKinds: []ProofBounds` and `ProofRulePreserveBoundsInvalidateLiveness`.
- Passes that invalidate proofs: no active registered optimizer pass declares `InvalidatedProofKinds`. The framework has invalidation metadata, `PassContext.InvalidateProof`, and a manager guard for invalidating rewrites without deltas, but current registered passes invalidate liveness facts, not proof kinds.
- Absence of rewrite without canonical proof: current consuming pass bodies use `requireMemoryProofs`, which resolves through `Snapshot.ResolveProof` and records proof fact IDs. Tests cover missing canonical proof, valid canonical proof, invalidated proof, unsafe proof, missing proof ID in memory rewrite decisions, and invalidating rewrite without delta. Findings C-001 and C-002 are the remaining contract/API gaps.
- Bounds-check elimination: `validation.CheckBoundsProofs` rejects unchecked index loads without proof IDs, and `CheckBoundsProofsWithPLIR` verifies PLIR guards, dominated proof use, typed proof terms, operation match, and `islandkernel` acceptance before removed checks are accepted (`compiler/internal/validation/validation.go:143`, `compiler/internal/validation/validation.go:175`). Focused tests passed.
- noalias: no registered optimizer pass consumes `ProofNoAlias`. The vectorization coverage row for `copy_u8` asks `islandkernel.CanClaimNoAlias`; that accepts only distinct live islands with verified `ProofNoAlias` and rejects unsafe/external memory (`compiler/internal/opt/opt_core.go:8237`, `compiler/internal/islandkernel/kernel.go:239`, `compiler/internal/islandkernel/kernel.go:429`). Other vector rows state noalias is not required for read-only/single mutable slice cases.
- scalar replacement: current evidence is lowering/planner scope rather than a standalone optimizer pass. Coverage points at `lower.scalar-replacement`, and focused planner/lower tests passed for tiny constant-index slice scalar replacement and direct-use constraints.
- allocation sinking: current evidence is lowering/planner scope rather than a standalone optimizer pass. Coverage points at `lower.stack-allocation` / `allocplan`; focused planner/lower tests passed for stack-lowered no-escape copies and stack storage.
- optimized/unoptimized equivalence: every optimizer pass contract requires translation validation; `Manager.runSelected` runs `validation.ValidateTranslation`, builds validation metadata, and checks bounds proofs after each pass (`compiler/internal/opt/opt_core.go:2616`). `ValidateTranslation` compares proof fact multisets and semantic/differential samples (`compiler/internal/validation/validation_translation.go:14`, `compiler/internal/validation/validation_translation.go:279`).
- deterministic optimizer decisions: pass order is fixed by `RegisteredPasses`, proof IDs are cleaned/deduplicated/sorted before decisions and deltas, snapshots digest facts in deterministic order, and focused tests including deterministic memory report projection passed (`compiler/internal/opt/opt_core.go:2540`, `compiler/internal/opt/opt_core.go:2517`, `compiler/internal/memoryfacts/snapshot.go:318`).

unresolved_risks:
- C-001 and C-002 are nonblocking for the inspected production route because `compiler/compiler_facade.go` supplies canonical `MemoryFacts`, but they remain contract/API hardening risks.
- I did not perform human security review; this remains out of scope and `release_security_review_status` remains `pending_final_rc`.
- `.cache` was used as the repo-local Go cache/GOTMPDIR per instructions. I ran `go clean -cache`; at handoff, unrelated active `go test ./...` processes in the shared worktree were using ignored `.cache` paths, so I did not force-delete active temp directories owned by those processes.

verdict: PASS_WITH_NONBLOCKING_FINDINGS
