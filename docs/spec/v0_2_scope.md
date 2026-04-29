# Tetra v0.2.0 Scope Contract

Status: current minor scope. This document defines what must stay true for the
repository to claim the `v0.2.0` profile.

Current public release truth is `v0.2.0` only when:

1. version metadata is intentionally promoted;
2. `scripts/release_v0_2_0_gate.sh` is run with fresh evidence;
3. `docs/checklists/v0_2_0_release_gate.md` is fully satisfied.

## Scope Goals

- Keep the current `v0.2.0` profile stable while improving verification depth.
- Strengthen diagnostics, parser/formatter coverage, and validator quality.
- Improve release process clarity for the next minor line.
- Keep unsupported/post-v1 features explicitly deferred and documented.

## Mandatory v0.2.0 Areas

1. Release identity and evidence:
`v0.2.0` scope/checklist/gate/release-notes/cut-guide/handoff docs exist and
use concrete commands and artifact paths.
2. Frontend and formatting hardening:
parser/lexer/formatter diagnostics and fixtures are tightened with focused tests.
3. Tooling and validator hardening:
machine-readable reports, schema validators, and release scripts are stricter.
4. Docs and user guidance:
version truth, supported surface, and limitations are consistently documented.
5. IR and lowering verification:
the public `compiler.Lower*` APIs and internal lowering package must run the
target-neutral IR verifier before codegen, covering main metadata, function slot
metadata, branch labels, stack height, local slot bounds, representative
optional/error, loop/control, actor/task, unsafe/memory, and UI metadata paths.
Unsupported lowering paths must return named diagnostics before backend codegen.

## Non-Goals

- Declaring `v1.0.0` readiness.
- Promoting distributed EcoNet/TetraHub production publishing.
- Claiming full UI runtime event dispatch or full native widget rendering.
- Claiming distributed actors or structured concurrency guarantees.

## Required Verification Envelope

- `go test ./compiler/... ./cli/... ./tools/... -count=1`
- `bash scripts/test_all.sh --quick`
- `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`
- `git diff --check`
- Epic 08 focused gate:
  `go test ./compiler/internal/lower ./compiler -run "Lower|IR|Verify|Unsupported|Loop|Task|Actor|UI|Unsafe" -count=1`
- final gate:
  `TETRA_SECURITY_REVIEW_SIGNOFF=<path> bash scripts/release_v0_2_0_gate.sh --report-dir <dir>`

## Final Closure Matrix

The final `v0.2.0` closure matrix is
`docs/checklists/v0_2_0_release_gate.md#final-verification-matrix`. It covers
workspace tests, quick/full wrappers, formatter and Flow scans, CLI JSON
reports, smoke reports, docs/API validation, release-state and hash validation,
security/reproducible-build/performance evidence, and the final release gate.

Every row must point at current report-directory evidence. A row with missing
logs, copied historical summaries, a stale version, or an unreviewed blocker is
not complete.

## Closure Rule

A `v0.2.0` task is complete only when implementation, tests/docs, and evidence
all exist in the same branch state. Checklist-only closure is invalid.
