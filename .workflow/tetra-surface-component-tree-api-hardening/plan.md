# Tetra Surface Component Tree API Hardening

## Goal
Implement the experimental pure-Tetra ComponentTree helper API described in
`/home/tetra/Downloads/tetra_surface_component_tree_api_hardening_plan.md`.
The milestone is complete only when the example app uses helpers instead of
manual structural node bookkeeping, all three API smoke targets produce strict
evidence, and the aggregate Surface gate validates the new reports.

## Success Criteria
- `lib/core/component.tetra` exposes helper API for init/reset, add root/child,
  bounds, validation, Column/Row layout, focus traversal, hit testing, dispatch
  paths, and draw order.
- `examples/surface_tree_app.tetra` uses the helpers and does not manually write
  `id`, `parent_id`, `child_index`, `first_child`, `child_count`, or final
  `tree.len = 7` setup.
- `component_tree_api` report evidence exists for headless, Linux real-window,
  and wasm32 browser canvas/input reports.
- `validate-surface-runtime` rejects missing/fake API evidence and old manual
  component-tree reports for the API milestone.
- API smoke scripts are integrated into `scripts/release/surface/gate.sh`.
- Docs, feature registry, manifest, and Graphify are updated.

## Current Context
- The prior Dynamic Component Tree milestone is present in the worktree.
- The repository has a dirty worktree with unrelated changes; do not revert or
  normalize files outside this milestone.
- Graphify artifacts exist under `graphify-out/`; use Graphify MCP first, then
  verify concrete files directly.

## Constraints
- Pure Tetra only for the component tree layer; no compiler magic, JS UI, DOM
  sidecars, trait-object ABI, witness-table dispatch, virtual DOM, or toolkit
  promotion.
- Preserve existing TextBox, SubmitButton, ResetButton, focus, text, resize,
  frame checksum, and close behavior.
- Do not broaden unrelated allowlists or hide failures.
- Do not set `GOCACHE` under `/tmp`.

## Risks
- API evidence could be faked while the example remains manually wired.
- Browser smoke may be fragile; reuse existing browser infrastructure.
- Dirty worktree may mix unrelated changes into review context.

## Approval Required
None for planned local code/test/doc edits. Ask before destructive Git actions,
external writes, force pushes, or mass rewrites outside the milestone.

## Work Packets
- API surface: inspect current Tetra limits, add focused tests, implement helper
  functions conservatively in `lib/core/component.tetra`.
- Example app: rewrite tree construction/event routing/layout calls through the
  helper API and add a source anti-regression test.
- Evidence and validator: add `component_tree_api` schema, positive/negative
  tests, API smoke modes, and strict host/source checks.
- Release scripts and docs: add three API scripts, gate integration, docs,
  feature registry, generated manifest, and Graphify update.
- Verification: run targeted tests first, then the full DoD command suite and
  preserve final reports in review directories.

## Integration Policy
Integrate changes only after local tests for each packet pass. If current repo
state differs from plan syntax, adapt to existing project patterns and document
the equivalence.

## Verification
- Targeted: component library/example compile checks, validator tests, smoke
  generator tests, script tests.
- Final: `go test ./... ./compiler/... ./cli/... ./tools/... -count=1`,
  docs/manifest checks, `git diff --check`, `graphify update .`, three API
  smoke scripts, aggregate gate, and individual report validation.

## Reusable Artifacts
Keep final report paths and changed-file summary in this workflow directory.
