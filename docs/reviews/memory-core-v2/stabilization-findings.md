# Memory Core v2 Stabilization Findings

reviewed_commit: 8f7529505a13b5da72fbc0c34c5bb110541c020f

review_set:
- `docs/reviews/memory-core-v2/compiler-soundness-review.md`
- `docs/reviews/memory-core-v2/runtime-domain-review.md`
- `docs/reviews/memory-core-v2/optimizer-proof-review.md`
- `docs/reviews/memory-core-v2/integration-review.md`

summary:
- blocker: 0
- critical: 0
- high: 0
- medium: 3
- low: 2
- informational: 0

## blocker

None.

## critical

None.

## high

None.

## medium

### A-001: Public multi-module lowering bypasses the canonical Memory Core v2 pipeline

source_review: `docs/reviews/memory-core-v2/compiler-soundness-review.md`

finding:
`compiler/compiler_facade.go` exposes `LowerModules(checked []*CheckedProgram)`
as a direct call to `lower.LowerModules(checked)`, while neighboring `Lower`
and `LowerModule` build a `memorypipeline.State`, lower via
`LowerPlannedProgram`, and apply lowering evidence. The internal
`lower.LowerModules` route calls `lowerCheckedFuncWithOptions(..., Options{},
nil, ...)`, so it has no canonical `memoryfacts.Graph`, no `allocplan.Plan`,
no per-allocation lowering evidence, and no validator handoff.

reproduction:
- `rg -n "func LowerModules|LowerModules\\(" compiler/compiler_facade.go compiler/internal/lower/lower_core.go compiler/tests/runtime/linker_test.go`
- Inspect `compiler/compiler_facade.go` around the public `LowerModules` API
  and `compiler/internal/lower/lower_core.go` around the internal
  `LowerModules` helper.

required_fix:
Route public `compiler.LowerModules` through the canonical Memory Core v2
pipeline or remove/deprecate the unplanned public lowering surface. Add
regression coverage proving `LowerModules` cannot lower memory-sensitive
programs without the canonical allocation plan/evidence path.

### B-001: Release evidence marks `wasm32-wasi reserve` unsupported while runtime ABI supports WASM reserve/commit

source_review: `docs/reviews/memory-core-v2/runtime-domain-review.md`

finding:
`MemoryBackendSupportMatrix("wasm32-wasi")` marks `reserve` and `commit`
supported through `wasm_memory_grow_combined_reserve_commit`, but the Memory
Core v2 gate and positive fixture mark `wasm32-wasi reserve` unsupported. The
release validator also expects flipping that row to `supported: true` to fail,
which contradicts the runtime ABI contract.

reproduction:
- Inspect `compiler/internal/runtimeabi/memory_backend.go` for
  `MemoryBackendSupportMatrix("wasm32-wasi")`.
- Inspect `compiler/internal/runtimeabi/memory_backend_test.go` for the WASM
  reserve/commit support assertions.
- Inspect `scripts/release/memory/memory-core-v2-gate.sh`,
  `tools/validators/memorycorev2/testdata/positive.json`, and
  `tools/validators/memorycorev2/report.go` for the release evidence and
  validator policy.

required_fix:
Align release evidence and validator policy with the runtime ABI contract. If
runtime ABI is correct, represent WASM reserve/commit as supported where
included and use an actually unsupported WASM operation such as `release`,
`decommit`, `trim`, or `footprint` for unsupported-target evidence.

### C-001: Manager proof-decision validation does not itself prove that nonempty proof IDs are canonical

source_review: `docs/reviews/memory-core-v2/optimizer-proof-review.md`

finding:
`validateMemoryDecisionEvidence` rejects `rewrite_applied` memory decisions
when `ProofIDs` is empty, but it does not resolve nonempty proof IDs through the
current `memoryfacts.Snapshot`. Current proof-consuming pass bodies call
`PassContext.requireMemoryProofs`, but the optimizer manager does not enforce
canonical proof resolution for every memory rewrite decision.

reproduction:
- Inspect `compiler/internal/opt/opt_core.go` around
  `validateMemoryDecisionEvidence`.
- Inspect `PassContext.requireMemoryProofs` and the current
  `loop-canonicalization` / `licm-pure-invariant` pass bodies.
- Add or run a contract pass that emits `DecisionCodeRewriteApplied` with
  `ProofIDs: []string{"proof:bogus"}` under `RunWithOptions` with
  `MemoryFacts` enabled; manager-level validation should reject it.

required_fix:
Add manager-level validation for memory rewrite decisions when canonical memory
facts are enabled: every proof ID on a memory rewrite decision must resolve
through the current `memoryfacts.Snapshot`, must be validated, must be non-unsafe
for proof-gated rewrites, and should populate canonical proof fact IDs or fail.
Add a regression test for a pass that emits a bogus nonempty proof ID.

## low

### C-002: Standalone `opt.Manager.Run` disables canonical memory proof resolution for proof-sensitive passes

source_review: `docs/reviews/memory-core-v2/optimizer-proof-review.md`

finding:
`Manager.Run` delegates to `RunWithOptions` with empty options, creating a
disabled memory context. `requireMemoryProofs` currently returns success when
memory context is disabled, so standalone callers can run proof-sensitive
passes using only IR proof IDs plus translation validation instead of canonical
snapshot resolution. Production build wiring supplies canonical `MemoryFacts`,
so this is a contract/API hardening risk rather than a known production-route
failure.

reproduction:
- Inspect `compiler/internal/opt/opt_core.go` around `Manager.Run`,
  memory context construction, and `requireMemoryProofs`.
- Inspect `compiler/compiler_facade.go` release optimization wiring, which
  supplies `Options{MemoryFacts: snapshot}`.

required_fix:
Either make `Run` reject proof-sensitive passes unless `Options.MemoryFacts` is
supplied, or clearly mark `Run` as noncanonical/test-only and add a negative
test that proof-sensitive passes skip or fail when canonical memory facts are
absent.

### D-001: Broad workspace/scriptstest runs showed transient environment coupling outside the Memory Core v2 gate path

source_review: `docs/reviews/memory-core-v2/integration-review.md`

finding:
Broad `go test` runs intermittently failed in `tools/scriptstest` fake-repo or
workspace scenarios, including missing `$WORK/.../_pkg_.a` import archives and
tests that assumed a GitHub-shaped remote URL. Focused reruns of the failing
packages/tests passed, final broad reruns passed, and clean-clone Memory Core
v2 gates were deterministic.

reproduction:
- See the failed broad commands and passing focused/final reruns in
  `docs/reviews/memory-core-v2/integration-review.md`.

required_fix:
No Memory Core v2 code, gate, schema, or documentation change is required by
this review. Track as test-infrastructure hardening: make `tools/scriptstest`
fake repos and nested Go test isolation independent of broad package execution
order, shared temporary/cache state, and GitHub-shaped remotes.

## informational

None.
