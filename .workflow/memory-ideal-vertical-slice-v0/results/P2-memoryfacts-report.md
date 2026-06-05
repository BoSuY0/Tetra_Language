# P2-memoryfacts-report Result

Status: completed_read_only

Sub-agent: Curie (`019e91ed-3238-7262-8b54-d9b1340a7b67`)

## Accepted Leads

- `compiler/internal/memoryfacts/report.go`, `facts.go`, and `validate.go` are
  the key MemoryFactGraph/report projection and invariant files for this slice.
- `tools/cmd/validate-memory-report/main.go` is the existing CLI report
  validator hook.
- `compiler/internal/memoryfacts/from_plir.go` already contains borrow/copy and
  inout/noalias projection hooks such as borrow owner/source facts, copy
  metadata facts, and exclusive inout noalias facts.
- Existing tests to extend or preserve include
  `compiler/internal/memoryfacts/report_test.go`,
  `compiler/internal/memoryfacts/from_plir_test.go`,
  `compiler/internal/memoryfacts/graph_test.go`, and
  `tools/cmd/validate-memory-report/main_test.go`.

## Risks

- The current report validator is stronger for generic report invariants than
  for the exact `MEM-BORROW-001` ownership chain, so the v0 correlation layer
  still needs explicit rows and a focused validator.
- `MEM-ALIAS-001` evidence is split across PLIR verification, memoryfacts
  projection, and report validation; end-to-end focused tests are required.
- The sub-agent did not run validators; this result is static discovery only.
