# compiler/tests

Black-box and domain-oriented compiler tests belong under this tree.

These tests should import `compiler/internal/testkit` instead of reaching into
large package-local test files. New directories should be named by compiler
domain, not by implementation accident.

## Root Package Exceptions

The canonical no-wrapper structure keeps black-box tests under `compiler/tests`.
Root-level `compiler/*_test.go` files are allowed only when they verify
package-private compiler contracts that do not yet have a production API.

Owner: compiler maintainers.

Verification: `cd compiler && go test ./... -count=1`.

- `abi_suite_test.go`: target ABI suite aggregation relies on package-private
  target check helpers while the CLI exposes only the runner/report surface.
- `actors_test.go`: actor runtime selection and required runtime symbol helpers
  are package-private.
- `atomic_suite_test.go`: target atomic stress suite aggregation covers
  package-private stress iteration and check helpers.
- `atomic_target_diagnostics_test.go`: target-specific atomic lowering
  diagnostics exercise package-private IR target classification and build
  helpers.
- `compiler_pipeline_stage_test.go`: native build planning and cache pipeline
  stages are package-private.
- `compiler_test.go`: package-private integration harness and legacy helper
  definitions that still cover compiler package internals.
- `distributed_actor_runtime_test.go`: distributed actor runtime usage,
  required-symbol, target-support, and builtin-runtime build checks still
  exercise package-private compiler collection and runtime validation helpers.
- `ffi_target_diagnostics_test.go`: native FFI boundary diagnostics exercise
  package-private target ABI gate helpers and compiler build validation.
- `filesystem_runtime_test.go`: filesystem runtime symbol validation helpers are
  package-private.
- `fuzz_suite_test.go`: target fuzz suite aggregation relies on package-private
  target fuzz check helpers while CLI report coverage remains external.
- `link_object_contract_test.go`: `readLinkObjects` duplicate and metadata
  validation is package-private.
- `manifest_test.go`: manifest assertions compare package-private required
  runtime symbol lists.
- `net_runtime_test.go`: networking runtime usage collection, required-symbol,
  target-support, and runtime object validation helpers are package-private.
- `runtime_override_test.go`: runtime override validation uses package-private
  actor/runtime annotation helpers.
- `task_runtime_test.go`: task runtime mode selection, ABI symbol lists, and
  runtime object validators are package-private.
- `tetra_bug_regression_test.go`: regression cases for recently fixed compiler
  bugs still use package-private build-and-run helpers.
- `wasm_policy_test.go`: WASM IR policy validation is package-private.
- `wasm_runtime_diagnostics_test.go`: WASM runtime diagnostic guard is
  package-private.
