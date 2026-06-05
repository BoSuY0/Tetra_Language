# Tetra Surface Dynamic Component Tree

## Goal
Implement the experimental Tetra Surface Dynamic Component Tree milestone from
`/home/tetra/Downloads/tetra_surface_dynamic_component_tree_plan.md`.

The milestone must prove a pure-Tetra semi-dynamic component tree where ordinary
Tetra structs are composed under `RootApp -> Column -> TextLabel/TextBox/Row ->
SubmitButton/ResetButton`, and layout, draw, hit testing, focus traversal, text
routing, button actions, resize relayout, and runtime evidence traverse/report
parent-child links instead of a metadata-only or host-widget shortcut.

## Success Criteria
- `examples/surface_tree_app.tetra` exists, builds, and exercises at least seven
  component nodes with stable ids, parent ids, child indexes, bounds, focus
  order, and tree-owned routing behavior.
- `tools/cmd/surface-runtime-smoke` supports and emits strict reports for
  `headless-component-tree`, `linux-x64-real-window-component-tree`, and
  `wasm32-web-browser-canvas-component-tree`.
- `tools/cmd/validate-surface-runtime` rejects missing, fake, metadata-only,
  hardcoded, bad-parent, bad-path, wrong-host, DOM/user-JS/legacy sidecar, and
  Node-only browser evidence.
- Release scripts and aggregate `scripts/release/surface/gate.sh` run all three
  component-tree target reports.
- Docs, feature registry/manifest outputs, and Graphify artifacts are updated.
- Final checks from Section 21 of the source plan pass or failures are reported
  with exact evidence.

## Current Context
- Graphify highlighted `tools/validators/surface/report_test.go`,
  `validHeadlessComponentTreeSurfaceReportJSON()`, and `ComponentTreeReport`
  as the main strict-evidence community.
- The working tree is already dirty with many pre-existing Surface/runtime,
  compiler, docs, generated, and Graphify changes. Do not revert unrelated
  changes; inspect before touching.
- Existing implementation artifacts already include `lib/core/component.tetra`,
  `examples/surface_tree_app.tetra`, component-tree smoke modes, validator
  structs/rules, scripts, docs, and generated/Graphify files. The critical path
  is to audit for plan gaps, add missing RED coverage, patch narrowly, then
  verify.

## Constraints
- Communicate in Ukrainian.
- Keep Surface pure-Tetra: no platform widget tree, DOM UI, user JavaScript app
  logic, React/HTML UI, legacy `.ui.*` sidecars, or production toolkit claim.
- Keep the dynamic level honest: `semi-dynamic-child-list`, not final
  trait-object/witness-table UI dispatch unless actually implemented.
- Do not set `GOCACHE` under `/tmp`; use a persistent cache path if needed.
- Use `apply_patch` for manual file edits.
- Run `graphify update .` after modifying code files.

## Risks
- Existing artifacts may be partially implemented but still accept hardcoded or
  metadata-only evidence.
- The full verification suite is large and may expose unrelated dirty-worktree
  failures; report exact failing commands rather than smoothing over them.
- Generated files must be regenerated with repo tools, not hand-edited.

## Approval Required
No destructive/external approvals are planned. Ask before any deletion,
mass-rename, force push, migration, deployment, or external write. Subagents are
not used unless the user explicitly authorizes delegation.

## Work Packets
1. Discovery packet: map the current component-tree implementation, validator,
   scripts, docs, feature registry, manifest, and tests against the plan.
2. Validator/TDD packet: identify missing negative rules, add RED tests, and
   implement strict checks.
3. Runtime/smoke packet: ensure all component-tree modes generate target-specific
   process, artifact, event, frame, and host evidence.
4. Docs/registry packet: ensure docs and feature registry describe the milestone
   honestly and generated manifest is refreshed.
5. Integration/verification packet: run targeted checks first, then Section 21
   checks as far as the environment permits, and update Graphify.

## Integration Policy
Accept only changes backed by inspected files and current command evidence.
If packet results conflict, inspect the source file or validator output and use
the stricter interpretation from the milestone plan. Leave unrelated dirty
worktree changes intact.

## Verification
- Targeted tests: `go test ./tools/cmd/surface-runtime-smoke
  ./tools/cmd/validate-surface-runtime ./tools/validators/surface -run
  'ComponentTree|Surface' -count=1`
- Script tests: `go test ./tools/scriptstest -run Surface -count=1`
- Smoke reports for all three component-tree modes under a persistent report
  directory outside `/tmp` if possible.
- Final plan checks: full Go test command, `verify-docs`, `gen-manifest`,
  `validate-manifest`, `git diff --check`, `graphify update .`, three
  component-tree smoke scripts, and aggregate gate.

## Reusable Artifacts
Keep this workflow directory as the run artifact. Do not save bulky logs,
secrets, or transient report JSON beyond normal repo outputs unless needed for
evidence.
