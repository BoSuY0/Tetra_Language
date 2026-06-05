# Final Report: Memory Ideal Vertical Slice v2

Status: complete

## Outcome

Memory Ideal Vertical Slice v2 is accepted for exactly three rows:
`MEM-BORROW-004`, `MEM-BORROW-005`, and `MEM-ALIAS-002`.

The slice extends the v0/v1 MemoryFactGraph correlation pattern to
function-typed values and callback boundaries without claiming full callable
ABI, captured/escaping closures, async/task/actor boundaries, raw pointer
expansion, target parity, broad noalias, or performance.

## Accepted Results

- P0 discovery: accepted. v1 patterns and v2 callback/function-typed semantic
  integration points were grounded in Graphify MCP output and local file reads.
- P1 memoryfacts/model: accepted. MemoryFactGraph projections, report
  validator allowlists, exact-row correlation validation, and MiniMemoryModel
  v2 cases cover `function_value_contains_borrow`,
  `callback_arg_contains_borrow`, and `callback_inout_conservative`.
- P2 semantics: accepted. `MemoryIdealV2` tests cover local known callback use,
  function-typed field local use, `.copy()` before callback escape, non-borrow
  callback parameters, owned return/global escape, consume/inout rejection,
  callback inout alias rejection, unknown target conservatism, and captured
  callback conservative rejection.
- P3 docs/audit/manifest: accepted. v2 correlation/final audit docs, schema,
  production-core boundary docs, generated manifest, and manifest validator
  fixtures are synchronized.
- P4 verification: accepted. Focused gates, broad gates, CI script,
  docs/manifest gates, diff check, and Graphify update passed on the current
  worktree.

## Rejected Results

- Broad noalias and universal callable-memory claims.
- Trusting unknown callback targets as lifetime-safe borrow facts.
- Captured or escaping callback support beyond existing conservative rejection.
- Interface/protocol callable values, async/task/actor boundaries, raw pointer
  expansion, target parity, and performance claims.
- Treating reports as truth instead of projections from `MemoryFactGraph`.

## Conflicts Resolved

- No unresolved packet conflicts remain.
- The v2 callback/reentrant `inout` row is intentionally `conservative`, not a
  validated noalias row.
- Existing checker rejections for unknown/capturing callback cases were kept as
  conservative evidence rather than widening callable support.

## Verification Evidence

- Passed: `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v2-memoryfacts go test ./compiler/internal/memoryfacts -count=1`.
- Passed: `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v2-mini go test ./compiler/internal/memorymodel -count=1`.
- Passed: `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v2-semantics go test ./compiler/tests/semantics -run 'MemoryIdealV2|Callback|FunctionTyped|Borrow|Inout|Alias' -count=1`.
- Passed: `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v2-tools go test ./tools/cmd/validate-memory-report ./tools/cmd/validate-memory-correlation -count=1`.
- Passed: `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v2-correlation go run ./tools/cmd/validate-memory-correlation --file docs/audits/memory-ideal-vslice-v2-correlation.md`.
- Passed: `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v2-broad go test ./compiler/... ./cli/... ./tools/... -count=1`.
- Passed: `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v2-ci bash scripts/ci/test.sh`; final output included `OK` and `Artifact: tetra.release.v0_4_0.go-test-suite.v1`.
- Passed: `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v2-manifest go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`.
- Passed: `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v2-docs go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`.
- Passed: `git diff --check`.
- Passed: `graphify update .`; rebuilt 21239 nodes, 66396 edges, and 1173
  communities.

## Remaining Risks

- The repository remains heavily dirty with substantial unrelated tracked and
  untracked state. This workflow preserved that state and did not classify
  unrelated files as v2 work.
- v2 intentionally remains narrow. Future callable slices must add their own
  facts, validators, tests, and audits instead of broadening these rows.

## Reusable Follow-up

- Future memory carrier slices can reuse the v2 recipe: add a narrow wrapper or
  boundary kind, derive only parent-linked MemoryFactGraph facts, project report
  rows with row-specific validators, add exact-row correlation validation, and
  close with focused semantics/model/report tests plus a final audit naming
  nonclaims.
