# Final Report: Safe View Lifetime Contracts v1

## Goal Name

Safe View Lifetime Contracts v1

## Summary

Implemented the supported v1 lifetime contract layer for borrowed slice and
String byte-view surfaces. The implementation adds explicit borrowed return
syntax, semantic propagation through function signatures and function-typed
values, aggregate hidden-borrow escape checks, actor/task boundary rejection,
and stricter PLIR/proof/allocation evidence validation.

## Files Changed For This Goal

- `compiler/internal/frontend/ast.go`
- `compiler/internal/frontend/parser.go`
- `compiler/internal/frontend/parser_test.go`
- `compiler/internal/semantics/types.go`
- `compiler/internal/semantics/builtins.go`
- `compiler/internal/semantics/exprs.go`
- `compiler/internal/semantics/checker.go`
- `compiler/internal/semantics/function_types.go`
- `compiler/interface.go`
- `compiler/lsp.go`
- `compiler/internal/plir/plir.go`
- `compiler/internal/plir/verify.go`
- `compiler/internal/plir/plir_test.go`
- `compiler/explain_reports_test.go`
- `compiler/tests/semantics/borrow_copy_test.go`
- `compiler/tests/semantics/interface_test.go`
- `compiler/tests/ownership/ownership_test.go`
- `compiler/tests/safety/safety_diagnostics_test.go`
- `cli/cmd/tetra/check_diagnostics_lifetime_borrow_test.go`
- `cli/cmd/tetra/check_diagnostics_lifetime_global_assignment_test.go`
- `cli/cmd/tetra/test_structure_test.go`
- `compiler/features.go`
- `docs/generated/manifest.json`
- `docs/design/provenance_lifetime_ir.md`
- `docs/design/truthful_intent_architecture.md`
- `docs/design/truthful_safe_values.md`
- `docs/spec/current_supported_surface.md`
- `docs/user/examples_index.md`
- `examples/safe_view_actor_copy_boundary.tetra`
- `examples/safe_view_aggregate_copy_escape.tetra`
- `examples/safe_view_borrow_return.tetra`
- `examples/safe_view_copy_escape.tetra`
- `examples/safe_view_string_borrow_return.tetra`
- `examples/safe_view_task_copy_boundary.tetra`
- `scripts/release/safe-view-lifetime/gate.sh`
- `scripts/release/safe-view-lifetime/README.md`
- `graphify-out/GRAPH_REPORT.md`
- `graphify-out/graph.json`
- `graphify-out/manifest.json`
- `.workflow/safe-view-lifetime-contracts-v1/final-report.md`

## Supported Syntax And Examples

- `func view(xs: borrow []u8) -> borrow []u8:` for borrowed slice returns.
- `func view(s: borrow String) -> borrow String:` for borrowed String returns.
- Function-typed parameters, locals, fields, enum payloads, callbacks, and
  interface signatures now preserve borrowed return ownership metadata.
- Borrowed view returns can be used locally, passed to borrow-only parameters,
  forwarded through borrowed-return functions, or converted to owned storage
  with `.copy()`.

Examples added:

- `examples/safe_view_borrow_return.tetra`
- `examples/safe_view_string_borrow_return.tetra`
- `examples/safe_view_copy_escape.tetra`
- `examples/safe_view_actor_copy_boundary.tetra`
- `examples/safe_view_task_copy_boundary.tetra`
- `examples/safe_view_aggregate_copy_escape.tetra`

## Unsupported And Non-Goals

- No named lifetimes such as `'a` or generic lifetime parameters.
- No claim of full Rust-like borrow checker parity.
- No full mutable alias model or production FFI lifetime system.
- No generalized lifetime inference for arbitrary generic containers.
- No Unicode/grapheme-aware String lifetime or editing model.

## Diagnostics Added Or Tightened

- Reject borrowed slice/String returns from owned return types unless the
  function declares `-> borrow ...` or the expression is explicitly copied.
- Reject borrowed returns derived from local owned storage that dies in the
  callee.
- Reject inconsistent borrowed return owners across branches.
- Reject function-type assignment/callback/interface mismatches where borrowed
  return ownership differs.
- Reject hidden borrowed slices inside structs, enums, and optionals when they
  escape through owned returns, globals, actor sends, task boundaries, consumes,
  or inout assignments.
- Preserve CLI JSON diagnostic coverage for lifetime, actor boundary, task
  boundary, and aggregate hidden-borrow cases.

## Tests Added Or Updated

- Parser coverage for `-> borrow` return syntax and invalid borrowed return
  type forms.
- Semantics coverage for borrowed slice/String return contracts, branch origin
  consistency, function-typed borrowed returns, forwarding requirements,
  aggregate hidden-borrow escapes, actor sends, task boundaries, and generated
  interfaces.
- PLIR verifier negative tests for borrowed facts without `no_escape`,
  `derived_window` facts without source, and copy allocation intents without
  owned facts.
- CLI JSON diagnostic tests for safe-view lifetime, actor/task boundary, and
  aggregate hidden-borrow diagnostic families.
- Ownership and safety diagnostic fixtures updated to the stabilized message
  order.

## Reports And Release Gate

- Added focused gate: `scripts/release/safe-view-lifetime/gate.sh`.
- Added gate README: `scripts/release/safe-view-lifetime/README.md`.
- Gate artifacts produced under `/tmp/tetra-safe-view-lifetime-gate-current`:
  - `safe-view-borrow-return.proof.json`
  - `safe-view-borrow-return.alloc.json`
  - `safe-view-copy-escape.proof.json`
  - `safe-view-copy-escape.alloc.json`
  - `safe-view-boundary-negative.txt`
  - `safe-view-lifetime-summary.json`
- Manifest regenerated and validated with
  `language.safe-view-lifetime-contracts-v1`.
- Graphify code graph updated after source changes.

## Verification Evidence

- `GOCACHE="$HOME/.cache/tetra-language/go-build-safe-view-final" go test ./compiler/tests/safety -run 'TestSafetyDiagnosticCodesForKeyFamilies' -count=1`: passed.
- `GOCACHE="$HOME/.cache/tetra-language/go-build-safe-view-final" go test ./tools/scriptstest -run 'TestNoWrapperReleaseDirectoriesHaveReadmes|TestShellScriptsDoNotDefaultTetraArtifactsToTmpfs' -count=1`: passed.
- `GOCACHE="$HOME/.cache/tetra-language/go-build-safe-view-final" go test ./tools/scriptstest -run 'TestWorkspaceModules' -count=1`: passed.
- `GOCACHE="$HOME/.cache/tetra-language/go-build-safe-view-final" go test ./compiler/... ./cli/... ./tools/... -count=1`: passed.
- `GOCACHE="$HOME/.cache/tetra-language/go-build-safe-view-final" go test ./... ./compiler/... ./cli/... ./tools/... -count=1`: passed.
- `GOCACHE="$HOME/.cache/tetra-language/go-build-safe-view-ci" bash scripts/ci/test.sh`: passed; output ended with `OK` and `Artifact: tetra.release.v0_4_0.go-test-suite.v1`.
- `GOCACHE="$HOME/.cache/tetra-language/go-build-safe-view-final" go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`: passed.
- `GOCACHE="$HOME/.cache/tetra-language/go-build-safe-view-final" go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`: passed.
- `GOCACHE="$HOME/.cache/tetra-language/go-build-safe-view-gate" bash scripts/release/safe-view-lifetime/gate.sh --report-dir /tmp/tetra-safe-view-lifetime-gate-current`: passed.
- `git diff --check`: passed.
- `graphify update .`: passed; rebuilt 17468 nodes, 53393 edges, 1077 communities.

All goal-specific Go build caches were cleaned with `go clean -cache` after the
evidence runs.

## Unrelated Dirty Files Encountered

The worktree was already heavily dirty before this goal work, including broad
modified and untracked files across `.github`, `AGENTS.md`, runtime/backends,
Surface examples, release scripts, tooling, documentation, and Graphify
artifacts. This workflow did not revert or clean unrelated changes. The exact
current inventory is available from `git status --short`.

## Deferred Work

No requirement from the Safe View Lifetime Contracts v1 plan is intentionally
deferred. The following remain explicit non-goals rather than incomplete v1
work: named lifetimes, a full Rust-like borrow checker, generic lifetime
parameters, and a production FFI lifetime system.

## Final Status

Safe View Lifetime Contracts v1 is complete for supported slice/String
byte-view surfaces. Borrowed returns, cross-module borrow signatures,
actor/task copy boundaries, aggregate hidden-borrow escape rejection, and
PLIR/proof/alloc evidence are implemented and tested. This is not a full
named-lifetime or Rust-like borrow checker.
