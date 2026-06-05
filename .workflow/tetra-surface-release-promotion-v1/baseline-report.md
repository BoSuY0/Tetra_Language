# Tetra Surface Release Promotion v1 Baseline Report

Date: 2026-06-02

## Goal

Capture the current pre-promotion baseline before changing Surface release
status, production claims, docs, or feature registry entries.

## Git State

- HEAD: `5129f2623d9639990076a7d422e56f02b0ed3254`
- Dirty worktree summary at capture time: 264 entries
  - 151 modified entries
  - 113 untracked entries
- The worktree was already very dirty before this Surface promotion goal. This
  goal must preserve unrelated work rather than reverting it.
- This baseline iteration intentionally touched:
  - `GOAL.md`, `PLAN.md`, `ATTEMPTS.md`, `NOTES.md`, `CONTROL.md`
  - `.workflow/tetra-surface-release-promotion-v1/`
  - `examples/safe_view_borrow_return.tetra`
  - `examples/safe_view_copy_escape.tetra`

## Gate Results

- Experimental Surface gate:
  `GOCACHE=$(pwd)/.cache/go-build-surface-release bash scripts/release/surface/gate.sh --report-dir .workflow/tetra-surface-release-promotion-v1/baseline-reports/surface-experimental`
  -> PASS.
- Safe View Lifetime gate first failed because two fixture examples returned
  borrowed views as owned `[]u8` after the current Safe View contract became
  stricter. Fixed the fixtures by keeping borrowed windows local and copying
  only in the copy-escape case.
- Safe View Lifetime gate after fixture repair:
  `GOCACHE=$(pwd)/.cache/go-build-surface-release bash scripts/release/safe-view-lifetime/gate.sh --report-dir .workflow/tetra-surface-release-promotion-v1/baseline-reports/safe-view-lifetime`
  -> PASS.
- Focused/broad Go gate:
  `GOCACHE=$(pwd)/.cache/go-build-surface-release go test ./compiler/... ./cli/... ./tools/... -count=1`
  -> PASS.
- Full workspace Go gate:
  `GOCACHE=$(pwd)/.cache/go-build-surface-release go test ./... ./compiler/... ./cli/... ./tools/... -count=1`
  -> PASS.
- CI script:
  `GOCACHE=$(pwd)/.cache/go-build-surface-release bash scripts/ci/test.sh`
  -> PASS with `OK` and `Artifact: tetra.release.v0_4_0.go-test-suite.v1`.
- Docs:
  `GOCACHE=$(pwd)/.cache/go-build-surface-release go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`
  -> PASS.
- Manifest:
  `GOCACHE=$(pwd)/.cache/go-build-surface-release go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`
  -> PASS.
- Whitespace:
  `git diff --check`
  -> PASS.

## Baseline Artifacts

- Surface experimental reports:
  `.workflow/tetra-surface-release-promotion-v1/baseline-reports/surface-experimental`
  - 24 top-level JSON reports, including `artifact-hashes.json`.
- Safe View Lifetime reports:
  `.workflow/tetra-surface-release-promotion-v1/baseline-reports/safe-view-lifetime`
  - 10 top-level files including proof, allocation, negative diagnostics, and
    `safe-view-lifetime-summary.json`.

## Release Blockers Before Promotion

The current Surface chain is a strong experimental evidence chain, but not a
release-ready/current claim. The external plan identifies these blockers:

- no release contract yet;
- no `tetra.surface.release.v1` summary schema yet;
- no production `lib.core.text` API yet;
- no production text-input/clipboard/IME release evidence yet;
- toolkit reports remain experimental/minimal/reuse evidence rather than
  `production-widgets-v1`;
- accessibility reports remain metadata-only and do not prove platform bridge
  evidence;
- browser release still needs real browser canvas/readback/input/clipboard/IME
  and accessibility snapshot/mirror evidence;
- Linux release still needs real-window text/clipboard/IME/accessibility bridge
  evidence;
- no strict release-state validator or release negative fixture matrix yet;
- no final Surface release gate yet;
- feature registry and docs must not be promoted before release evidence exists;
- final reports, artifact hashes, final dump files, and final release audit do
  not exist yet.

## Next Slice

Proceed to SURF-2/SURF-3 skeleton work: release contract doc, scoped target
matrix, status vocabulary, and initial release schema/validator RED tests.
