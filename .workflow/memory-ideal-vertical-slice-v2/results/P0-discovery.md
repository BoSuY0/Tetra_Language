# P0 Discovery Result

Status: accepted

## Evidence

- Graphify MCP:
  - `query_graph("function typed value callback borrow escape inout alias memoryfacts MiniMemoryModel validate-memory-report correlation")`
  - `get_neighbors("function_types.go")`
  - `shortest_path("function_types.go", "from_plir.go")`
- Local reads:
  - `compiler/internal/memoryfacts/from_plir.go`
  - `compiler/internal/memoryfacts/from_plir_test.go`
  - `compiler/internal/memoryfacts/report.go`
  - `compiler/internal/memoryfacts/validate.go`
  - `compiler/internal/memorymodel/mini.go`
  - `compiler/internal/memorymodel/mini_test.go`
  - `compiler/internal/semantics/exprs.go`
  - `compiler/internal/semantics/function_types.go`
  - `compiler/tests/semantics/borrow_copy_test.go`
  - `compiler/tests/semantics/closures_semantic_clauses_test.go`
  - `tools/cmd/validate-memory-correlation/main.go`
  - `tools/cmd/validate-memory-report/main.go`
  - `docs/spec/memory_report_schema_v1.md`
  - `docs/generated/manifest.json`

## Findings

- v1 derived borrow projections are implemented in
  `compiler/internal/memoryfacts/from_plir.go` by deriving claim rows from a
  safe borrowed parent PLIR fact plus direct wrapper context in value/op/reason
  text. The smallest v2 path is to extend that narrow projection mechanism to
  function-value and callback facts.
- Report validation has duplicated claim-parent and cost-class allowlists in
  `compiler/internal/memoryfacts/validate.go` and
  `tools/cmd/validate-memory-report/main.go`; v2 claims must be added to both.
- Correlation validation selects exact row sets by requirement IDs. v2 requires
  a third exact row set for `MEM-BORROW-004`, `MEM-BORROW-005`, and
  `MEM-ALIAS-002`.
- `compiler/internal/semantics/exprs.go` already rejects borrowed values passed
  to non-borrow function-typed local/field/callback parameters, borrowed values
  consumed by function-typed calls, borrowed values passed as `inout`, and
  `inout` aliasing in function-typed calls.
- `compiler/internal/semantics/function_types.go` already tracks
  function-typed locals, struct fields, enum payloads, callback arguments,
  known direct callback targets, unknown callback targets under strict semantic
  clauses, and captured callback metadata. Existing tests in
  `compiler/tests/semantics/closures_semantic_clauses_test.go` already cover
  many callable escape diagnostics.
- `compiler/tests/semantics/borrow_copy_test.go` is the right focused test file
  for the new `MemoryIdealV2` borrow/callback slice because it already contains
  v0/v1 memory ideal tests and function-typed borrowed return contracts.
- `docs/generated/manifest.json`, `tools/cmd/validate-manifest/main.go`, and
  `tools/cmd/validate-manifest/main_test.go` mirror the v1 audit doc list under
  `safety.production-core`; v2 docs must be synchronized there.

## RED Test Targets

- `compiler/internal/memorymodel/mini_test.go`:
  add v2 cases for function-value local use, callback escape rejection, copied
  callback escape, callback `inout` alias conservatism/rejection, and unknown
  callback target conservatism.
- `compiler/internal/memoryfacts/from_plir_test.go`:
  add v2 projection tests for `function_value_contains_borrow`,
  `callback_arg_contains_borrow`, and `callback_inout_conservative`, including
  unknown callback target conservative/non-trusted behavior.
- `compiler/internal/memoryfacts/report_test.go` and
  `tools/cmd/validate-memory-report/main_test.go`:
  add parent-fact tests for the three v2 derived claims and keep broad noalias
  rejected.
- `tools/cmd/validate-memory-correlation/main_test.go`:
  add v2 accept/missing/mixed-row tests.
- `compiler/tests/semantics/borrow_copy_test.go`:
  add `MemoryIdealV2` positive/negative tests using current function-typed call
  diagnostics; prefer conservative tests for unknown/capturing callback cases.

## Decisions

- Do not add callable ABI behavior.
- Do not broaden captured or escaping closures.
- Treat existing semantic rejections as acceptable conservative evidence for
  required negative cases.
- Keep v2 facts PLIR/report-level and narrow; `MemoryFactGraph` remains the
  source of truth.
