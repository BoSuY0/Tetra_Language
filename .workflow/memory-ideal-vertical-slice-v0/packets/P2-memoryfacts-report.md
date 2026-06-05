# P2-memoryfacts-report

Packet ID: P2-memoryfacts-report

Objective: Identify minimal MemoryFactGraph/report/validator changes for
`MEM-REP-001`, `MEM-BORROW-001`, and `MEM-ALIAS-001`.

Context: Reports are projections. The v0 slice must add only minimal report
rows and validators needed by the three correlation rows.

Files / sources:

- `compiler/internal/memoryfacts`
- `compiler/internal/plir`
- `compiler/internal/validation`
- `tools/cmd/validate-memory-report`
- prospective `tools/cmd/validate-memory-correlation`
- `docs/spec/memory_report_schema_v1.md`

Ownership: read-only.

Do: Locate fact kinds, report row fields, validators, and tests related to
borrow/copy/inout/unsafe/noalias. Recommend minimal implementation and tests.

Do not: Edit files, design a universal correlation engine, or migrate the full
report schema.

Expected output: `.workflow/memory-ideal-vertical-slice-v0/results/P2-memoryfacts-report.md`
summary with evidence.

Verification: cite commands and file paths inspected.
