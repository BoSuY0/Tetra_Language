# Surface Block Beauty Examples Implementation Plan

**Goal:** Add release-gated polished Surface Block examples that prove expressive UI can be built
from `Block` configuration and modifiers without introducing core widget abstractions.

**Context:** `SURFBLOCK-P15` requires five beautiful example scenes, Block-only validation,
accessibility/state/motion/asset evidence, docs coverage, and inclusion in the headless Block-system
release gate.

## Task 1: Add P15 Validator Contract

**Goal:** Make Block-only beauty evidence machine-checkable.

**Files:** `tools/cmd/validate-surface-block-examples/`,
`compiler/tests/semantics/surface_stdlib_test.go`,
`tools/scriptstest/release_surface_smoke_test.go`.

**Approach:** Add tests that require the five P15 example files, reject
`widgets.Button`/`widgets.TextBox`/`Button(` style core-widget usage, reject missing accessibility
roles, and reject missing hover/focus/pressed state evidence.

**Verification:** `go test -buildvcs=false ./compiler/tests/semantics -run 'SurfaceBlockExamples|Examples' -count=1` and targeted validator command tests.

**Done when:** The tests fail before implementation for the expected missing validator/examples
reason, then pass after implementation.

## Task 2: Create Five Block-Only Scenes

**Goal:** Add polished scenes for command palette, project dashboard, settings, editor shell, and
glass panel.

**Files:** `examples/surface_block_command_palette.tetra`,
`examples/surface_block_project_dashboard.tetra`, `examples/surface_block_settings.tetra`,
`examples/surface_block_editor_shell.tetra`, `examples/surface_block_glass_panel.tetra`.

**Approach:** Use `lib.core.block` paint/layout/text/input/event/state/motion/accessibility/asset
APIs directly. Demonstrate sidebar-like, card-like, input-like, command-item-like,
overlay/panel-like, and action-like shapes as configured Blocks, with dark/light tokens and
deterministic checksum-style evidence.

**Verification:** Build/run each example through the semantics test suite and validator command.

**Done when:** Each example compiles, exits with its expected success code, imports
`lib.core.block`, avoids `lib.core.widgets`, and contains accessibility/state/motion/asset evidence.

## Task 3: Wire Into Release Gate

**Goal:** Ensure the examples are release-gated, not docs-only.

**Files:** `scripts/release/surface/surface-headless-block-system-smoke.sh`,
`scripts/release/surface/README.md`.

**Approach:** Keep the existing headless Block-system runtime report and add a Block examples
validator report under the same report directory before artifact hash validation.

**Verification:** `bash scripts/release/surface/surface-headless-block-system-smoke.sh --report-dir reports/surface-block/examples-headless`.

**Done when:** The script emits both the existing runtime report and the P15 examples report, then
validates hashes.

## Task 4: Update User Docs

**Goal:** Explain the Block-first visual grammar without promoting unsupported Electron/DOM/platform
claims.

**Files:** `docs/user/surface_guide.md`, `docs/user/examples_index.md`.

**Approach:** Document the five P15 examples as experimental Block-first evidence and state that
button/card/input/modal-like shapes are ordinary Blocks, not core widgets.

**Verification:** Targeted searches for the example names and forbidden widget patterns.

**Done when:** Docs list all five examples and preserve the bounded Surface support claims.

## Task 5: Close P15 With Evidence

**Goal:** Record verified completion for `SURFBLOCK-P15`.

**Files:** `GOAL.md`, `reports/surface-block/examples-headless/`.

**Approach:** Run the required P15 commands, inspect generated reports, run `graphify update .`, and
update `GOAL.md` with concise evidence.

**Verification:** Required P15 commands pass in current worktree.

**Done when:** `GOAL.md` marks P15 complete with command/report evidence, and the overall `/goal`
remains active for P16-P20.
