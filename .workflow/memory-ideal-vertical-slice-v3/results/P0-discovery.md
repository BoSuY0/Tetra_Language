# P0 Discovery Result

Status: accepted

## Evidence

- Graphify MCP:
  - `query_graph("Memory Ideal v3 interface protocol existential dynamic dispatch static conformance borrow noalias MemoryFactGraph semantics tests")`
  - `get_neighbors("TestPlan250ProtocolConformanceAndDynamicDispatchBoundaries()")`
- Local reads:
  - `AGENTS.md`
  - `GOAL.md`
  - `graphify-out/GRAPH_REPORT.md`
  - `docs/audits/memory-ideal-vslice-v2-correlation.md`
  - `docs/audits/memory-ideal-vslice-v2-final.md`
  - `.workflow/memory-ideal-vertical-slice-v3/plan.md`
  - `.workflow/memory-ideal-vertical-slice-v3/orchestration.md`
  - `compiler/tests/semantics/plan250_semantics_test.go`
  - `compiler/tests/semantics/borrow_copy_test.go`
  - `compiler/internal/memoryfacts/from_plir.go`
  - `compiler/internal/memoryfacts/from_plir_test.go`
  - `compiler/internal/memoryfacts/validate.go`
  - `compiler/internal/memorymodel/mini.go`
  - `compiler/internal/memorymodel/mini_test.go`
  - `tools/cmd/validate-memory-report/main.go`
  - `tools/cmd/validate-memory-correlation/main.go`

## Findings

- The v3 implementation should follow the v2 projection path in
  `compiler/internal/memoryfacts/from_plir.go`: derive parent-linked report
  rows from PLIR facts and context text, then validate rows through
  `compiler/internal/memoryfacts/validate.go` and
  `tools/cmd/validate-memory-report/main.go`.
- Current protocol support is static-conformance-only. Existing semantics tests
  in `compiler/tests/semantics/plan250_semantics_test.go` reject generic bound
  requirement calls through `Drawable.draw` and runtime protocol values with
  `unknown type 'Drawable'`.
- Positive semantics coverage must use already-supported static/direct protocol
  or concrete paths. Runtime protocol values, trait objects, witness tables,
  existential containers, and dynamic dispatch remain conservative/nonclaims.
- MiniMemoryModel v3 can extend the existing wrapper/inout vocabulary with
  interface/protocol wrappers, static target knownness, and conservative
  protocol dispatch/noalias outcomes.

## RED Test Targets

- `compiler/internal/memorymodel/mini_test.go`: add v3 cases for static
  protocol local borrowed use, interface/protocol escape rejection,
  unknown/dynamic dispatch conservatism, and protocol dispatch noalias
  conservatism/rejection.
- `compiler/internal/memoryfacts/from_plir_test.go`: add projection tests for
  `interface_value_contains_borrow`,
  `protocol_dispatch_borrow_conservative`, and
  `protocol_dispatch_noalias_conservative`, plus unknown dynamic dispatch not
  emitting trusted interface borrow facts.
- `compiler/internal/memoryfacts/report_test.go` and
  `tools/cmd/validate-memory-report/main_test.go`: require `parent_fact_id` for
  v3 derived rows and keep broad noalias rejected.
- `tools/cmd/validate-memory-correlation/main_test.go`: add v3 accept,
  missing-row, and mixed-row rejection tests.
- `compiler/tests/semantics/borrow_copy_test.go` or
  `compiler/tests/semantics/plan250_semantics_test.go`: add focused
  `MemoryIdealV3` coverage using current static/conservative protocol behavior.

## Decisions

- Do not implement trait objects, runtime protocol values, witness tables, or
  dynamic dispatch.
- Treat existing runtime protocol and dynamic dispatch rejections as
  conservative evidence.
- Keep v3 facts PLIR/report-level and narrow; `MemoryFactGraph` remains the
  source of truth.
