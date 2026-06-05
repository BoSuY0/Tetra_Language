# P0 Discovery

Objective: map existing v0/v1 patterns and the function/callback semantic
surface before code edits.

Files / sources:
- `compiler/internal/memoryfacts`
- `compiler/internal/memorymodel`
- `compiler/internal/semantics`
- `compiler/tests/semantics/borrow_copy_test.go`
- `tools/cmd/validate-memory-report`
- `tools/cmd/validate-memory-correlation`
- `docs/audits`
- `docs/spec/memory_report_schema_v1.md`
- `docs/generated/manifest.json`

Do:
- Identify exact symbols and tests to extend.
- Record RED test targets for P1/P2/P3.
- Note any cases already rejected conservatively by existing semantics.

Do not:
- Edit code.
- Broaden beyond v2 scope.

Expected output: `.workflow/memory-ideal-vertical-slice-v2/results/P0-discovery.md`.

Verification: evidence from Graphify MCP and local file inspection.
