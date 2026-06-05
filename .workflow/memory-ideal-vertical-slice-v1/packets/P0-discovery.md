# P0-discovery

Packet ID: P0-discovery

Objective: Discover v0 patterns and v1 integration points before edits.

Context: v1 extends v0 from struct/optional borrow carriers to enum payload and
monomorphized generic wrapper carriers only.

Files / sources:

- `GOAL.md`
- `AGENTS.md`
- `graphify-out/GRAPH_REPORT.md`
- `docs/audits/memory-ideal-vslice-v0-correlation.md`
- `docs/audits/memory-ideal-vslice-v0-final.md`
- `.workflow/memory-ideal-vertical-slice-v0/final-report.md`
- `compiler/internal/memoryfacts`
- `compiler/internal/memorymodel`
- `compiler/internal/semantics`
- `compiler/tests/semantics/borrow_copy_test.go`
- `tools/cmd/validate-memory-report`
- `tools/cmd/validate-memory-correlation`

Ownership: read-only discovery.

Do: identify exact symbols, file paths, and focused commands for v1.

Do not: edit files or broaden scope.

Expected output: `.workflow/memory-ideal-vertical-slice-v1/results/P0-discovery.md`.

Verification: direct source citations and command names from inspected files.
