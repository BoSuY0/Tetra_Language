# P1 MemoryFacts And MiniMemoryModel

Objective: add narrow v2 source facts, report projections, validators, and
MiniMemoryModel cases.

Ownership:
- `compiler/internal/memoryfacts`
- `compiler/internal/memorymodel`
- `tools/cmd/validate-memory-report`

Do:
- Write RED tests first where practical.
- Add facts/projections:
  `function_value_contains_borrow`, `callback_arg_contains_borrow`,
  `callback_inout_conservative`.
- Add validator names:
  `function_value_borrow_escape_validator`,
  `callback_borrow_escape_validator`,
  `callback_alias_conservative_validator`.
- Keep unknown/unsafe sources conservative.

Do not:
- Implement full callable ABI.
- Emit trusted facts for unknown callback targets.
- Add broad noalias.

Expected output: `.workflow/memory-ideal-vertical-slice-v2/results/P1-memoryfacts-model.md`.

Verification:
- `go test ./compiler/internal/memoryfacts -count=1`
- `go test ./compiler/internal/memorymodel -count=1`
- `go test ./tools/cmd/validate-memory-report -count=1`
