# P4 Verification Packet

## Scope

Run final focused and full verification, refresh Graphify, and close the v3
workflow report.

## Required Gates

- focused memoryfacts, memorymodel, semantics, and tool gates from `GOAL.md`;
- v3 correlation validation on the real markdown file;
- broad `go test ./compiler/... ./cli/... ./tools/... -count=1`;
- `bash scripts/ci/test.sh`;
- `validate-manifest`;
- `verify-docs`;
- `git diff --check`;
- `graphify update .`.

## Acceptance

Accepted only after all gates pass or unrelated failures are classified with
evidence, and `.workflow/memory-ideal-vertical-slice-v3/final-report.md`
contains accepted/rejected/conflict summary plus verification evidence.
