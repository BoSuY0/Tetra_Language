# Surface Block Fake-Proof Validators Implementation Plan

**Goal:** Make `tetra.surface.block-system.v1` readiness machine-checkable so fake Block production
reports fail and scoped reports pass only with concrete same-commit evidence.

**Context:** `SURFBLOCK-P16` follows the P12-P15 runtime/example evidence. Existing Surface
validators reject many fake Surface claims, but Block-system validation still needs stricter
requirements for text, state, motion, assets, report locality, same-commit evidence, and fake
primitive claims.

## Task 1: Add RED Coverage For P16 Weak Proofs

**Goal:** Capture the current gaps before implementation.

**Files:** `tools/validators/surface/report_test.go`,
`tools/cmd/validate-surface-block-report/main_test.go`.

**Approach:** Add negative tests for Block-system reports missing text measurement, state
selector/state transition, motion frames, asset cache/manifest, fake core primitives (`Button`,
`Card`, `TextField`), unknown fields, report path outside the report directory, symlinked report
directories, stale artifact hashes, and same-commit mismatch.

**Verification:** `go test -buildvcs=false ./tools/validators/surface -run 'Block|Negative|Fake|SameCommit|Artifact' -count=1` and `go test -buildvcs=false ./tools/cmd/validate-surface-block-report -count=1`.

**Done when:** Tests fail for the expected missing strictness.

## Task 2: Enforce Full Block Evidence In Runtime Reports

**Goal:** Ensure real Block-system runtime reports contain all concrete feature evidence P16
requires.

**Files:** `tools/cmd/surface-runtime-smoke/main.go`,
`tools/cmd/surface-runtime-smoke/main_test.go`.

**Approach:** Reuse existing Block text/state/motion/asset helper evidence in
`runBlockSystemScenario`, add the extra components/events/state transitions needed by the
already-existing validators, and keep `block_system.frames` aligned with runtime frames.

**Verification:** `go test -buildvcs=false ./tools/cmd/surface-runtime-smoke -run 'BlockSystem|Block' -count=1`.

**Done when:** Generated headless/linux/web Block-system reports pass the stricter validator.

## Task 3: Harden Block Validator Rules

**Goal:** Make fake Block readiness claims fail at the validator layer.

**Files:** `tools/validators/surface/report.go`, `tools/validators/surface/report_test.go`.

**Approach:** Extend `validateBlockSystemEvidence` to require text, state, motion, and asset
evidence, and reject `Button`/`Card`/`TextField` as core primitive claims in Block-system reports
while allowing scoped nonclaims and existing toolkit evidence outside Block-system reports.

**Verification:** P16 validator test command.

**Done when:** Valid Block-system fixtures pass only with full evidence; missing/fake evidence fails
with specific diagnostics.

## Task 4: Harden CLI Evidence Checks

**Goal:** Ensure report files cannot be validated from stale or unsafe artifact layouts.

**Files:** `tools/cmd/validate-surface-block-report/main.go`,
`tools/cmd/validate-surface-block-report/main_test.go`.

**Approach:** Add `--same-commit`, strict JSON decode, report directory symlink rejection, artifact
path locality checks, local SHA-256/size recomputation for report artifacts, and a help path that
documents the options.

**Verification:** `go test -buildvcs=false ./tools/cmd/validate-surface-block-report -count=1` and
`go run ./tools/cmd/validate-surface-block-report --help >/dev/null`.

**Done when:** CLI rejects stale artifact hashes, outside paths, symlink report dirs, and
same-commit mismatches.

## Task 5: Close P16 With Evidence

**Goal:** Record verified completion without claiming P17-P20.

**Files:** `GOAL.md`, `reports/surface-block/validators/`.

**Approach:** Run P16 commands, regenerate a strict Block report under
`reports/surface-block/validators/`, run `graphify update .`, and update `GOAL.md`.

**Verification:** Required P16 commands pass in the current worktree.

**Done when:** `GOAL.md` marks P16 complete with concrete report/test evidence and the active
`/goal` remains open for P17-P20.
