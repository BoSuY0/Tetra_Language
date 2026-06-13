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
- `compatibility_stability_v1_test.go`: P24.2 compatibility/stability report
  validation checks package-private compatibility rows and stability witnesses
  before a public report command exists.
- `compiler_pipeline_stage_test.go`: native build planning and cache pipeline
  stages are package-private.
- `compiler_test.go`: package-private integration harness and legacy helper
  definitions that still cover compiler package internals.
- `distributed_actor_runtime_test.go`: distributed actor runtime usage,
  required-symbol, target-support, and builtin-runtime build checks still
  exercise package-private compiler collection and runtime validation helpers.
- `explain_reports_test.go`: explain/report emission keeps root-level
  coverage while the report-producing build options and artifact assertions are
  validated through the public compiler package boundary.
- `feature_surface_audit_test.go`: P22.0 feature-surface report validation
  checks package-private registry/report rows before the audit has a public
  command wrapper.
- `ffi_target_diagnostics_test.go`: native FFI boundary diagnostics exercise
  package-private target ABI gate helpers and compiler build validation.
- `filesystem_runtime_test.go`: filesystem runtime symbol validation helpers are
  package-private.
- `first_class_callables_coverage_test.go`: P22.1 first-class callables report
  validation uses package-private callable lowering witnesses and fake-claim
  rejection helpers.
- `formal_core_v1_test.go`: P23.2 formal-core report validation reuses
  package-private parse/check/lowering witnesses before a public report command
  exists.
- `fuzz_property_differential_v1_test.go`: P23.1 fuzz/property/differential
  report validation checks package-private compiler witnesses and reducer
  contracts.
- `fuzz_suite_test.go`: target fuzz suite aggregation relies on package-private
  target fuzz check helpers while CLI report coverage remains external.
- `link_object_contract_test.go`: `readLinkObjects` duplicate and metadata
  validation is package-private.
- `manifest_test.go`: manifest assertions compare package-private required
  runtime symbol lists.
- `memory_fuzz_oracle_v1_test.go`: MPC-15 memory fuzz oracle report validation
  checks package-private MemoryFactGraph witness construction before a public
  report command owns that compiler-side evidence.
- `net_runtime_test.go`: networking runtime usage collection, required-symbol,
  target-support, and runtime object validation helpers are package-private.
- `plir_api_test.go`: PLIR public API formatting coverage currently lives with
  the compiler root package tests until the PLIR-facing black-box suite is
  split under `compiler/tests`.
- `protocol_trait_object_decision_test.go`: P22.2 protocol/trait-object
  decision report validation uses package-private static conformance and
  lowering witnesses.
- `ram_contract_build_test.go`: RAM contract build-option tests verify
  package-private report emission, fail-if gates, and artifact wiring before a
  dedicated black-box command covers the compiler-side contract.
- `reports_internal_test.go`: allocation report-vs-plan validation is a
  package-private compiler report contract.
- `runtime_hardening_v1_test.go`: P24.1 runtime-hardening report validation
  checks package-private runtime hardening rows and witness aggregation before
  a public report command exists.
- `runtime_override_test.go`: runtime override validation uses package-private
  actor/runtime annotation helpers.
- `security_review_gate_v1_test.go`: P24.0 security review gate validation
  checks package-private report rows, live runtime witnesses, and artifact
  presence before a public report command exists.
- `self_hosting_gate_v1_test.go`: P23.3 self-hosting gate validation checks
  package-private self-hosting blocker rows and witness aggregation.
- `surface_runtime_test.go`: Surface runtime usage collection, required-symbol,
  target-support, and runtime object validation helpers are package-private.
- `task_runtime_test.go`: task runtime mode selection, ABI symbol lists, and
  runtime object validators are package-private.
- `tetra_bug_regression_test.go`: regression cases for recently fixed compiler
  bugs still use package-private build-and-run helpers.
- `translation_validation_v2_test.go`: P23.0 translation validation v2 report
  validation uses package-private optimizer, validator, and backend differential
  witnesses.
- `wasm_policy_test.go`: WASM IR policy validation is package-private.
- `wasm_runtime_diagnostics_test.go`: WASM runtime diagnostic guard is
  package-private.
