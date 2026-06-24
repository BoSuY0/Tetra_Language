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
- `actors_capacity_test.go`: actor mailbox capacity and scheduling edge cases
  exercise package-private actor runtime helpers split out of the root actor
  suite.
- `actors_declaration_state_test.go`: actor declaration and state checks still
  depend on package-private compiler helpers.
- `actors_runtime_symbols_test.go`: actor runtime symbol coverage validates
  package-private required-symbol collection.
- `actors_scheduler_targets_test.go`: actor scheduler target matrix coverage
  uses package-private target support helpers.
- `actors_test.go`: actor runtime selection and required runtime symbol helpers
  are package-private.
- `actors_typed_messages_test.go`: typed actor message diagnostics use
  package-private compiler entry points from the root actor suite.
- `atomic_suite_test.go`: target atomic stress suite aggregation covers
  package-private stress iteration and check helpers.
- `atomic_target_diagnostics_test.go`: target-specific atomic lowering
  diagnostics exercise package-private IR target classification and build
  helpers.
- `compatibility_stability_v1_test.go`: P24.2 compatibility/stability report
  validation checks package-private compatibility rows and stability witnesses
  before a public report command exists.
- `compiler_build_helpers_test.go`: shared build helper coverage remains with
  package-private root compiler integration helpers.
- `compiler_core_runtime_test.go`: core runtime build behavior is split from
  the root compiler suite but still exercises package-private helpers.
- `compiler_examples_microservices_test.go`: example and microservice compiler
  coverage uses root package helpers that are not public APIs.
- `compiler_interface_only_callable_test.go`: interface-only callable build
  coverage depends on package-private interface-only helpers.
- `compiler_interface_only_region_test.go`: interface-only region/resource
  validation remains package-private.
- `compiler_interface_only_resources_test.go`: interface-only resource checks
  use package-private build helpers.
- `compiler_language_world_test.go`: language-world compiler integration
  coverage uses package-private world construction helpers.
- `compiler_pipeline_stage_test.go`: native build planning and cache pipeline
  stages are package-private.
- `compiler_targets_cache_test.go`: target/cache integration coverage remains
  tied to package-private compiler build helpers.
- `compiler_test.go`: package-private integration harness and legacy helper
  definitions that still cover compiler package internals.
- `compiler_external_test.go`: merged external package tests that exercise the
  public compiler facade from outside `package compiler`.
- `compiler_suite_test.go`: merged root package suite for package-private
  compiler facade, report, gate, runtime, and target integration coverage.
- `compiler_t13_optimizer_test.go`: Memory Core v2 optimizer production wiring
  checks package-private compiler release-optimization state transitions before
  that evidence has a public report command.
- `distributed_actor_runtime_test.go`: distributed actor runtime usage,
  required-symbol, target-support, and builtin-runtime build checks still
  exercise package-private compiler collection and runtime validation helpers.
- `explain_reports_alloc_test.go`: allocation explain report coverage is split
  from the root explain suite and still checks package-private report wiring.
- `explain_reports_bounds_test.go`: bounds explain report coverage checks
  package-private compiler report wiring.
- `explain_reports_memory_test.go`: memory explain report coverage checks
  package-private compiler report wiring.
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
- `net_runtime_http_test.go`: HTTP runtime coverage is split from the root
  networking suite and still uses package-private runtime helpers.
- `net_runtime_linux_x64_epoll_test.go`: Linux x64 epoll runtime coverage uses
  package-private runtime object helpers.
- `net_runtime_linux_x64_socket_test.go`: Linux x64 socket runtime coverage
  uses package-private runtime object helpers.
- `net_runtime_target_helpers_test.go`: target helper coverage for networking
  remains package-private.
- `net_runtime_target_smoke_test.go`: networking target smoke tests use
  package-private compiler runtime validation helpers.
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
- `runtime_heap_telemetry_test.go`: runtime heap telemetry smoke tests compile
  and run linux-x64 binaries through package-private compiler build options
  before the telemetry sidecar contract has a black-box command wrapper.
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
- `task_runtime_cancellation_actor_test.go`: task cancellation and actor
  interactions are split from the root task runtime suite but use
  package-private runtime helpers.
- `task_runtime_deadlines_test.go`: task deadline coverage uses package-private
  task runtime helpers.
- `task_runtime_group_cancel_test.go`: task group cancellation coverage uses
  package-private runtime helpers.
- `task_runtime_helpers_test.go`: shared task runtime helper coverage remains
  package-private.
- `task_runtime_lowering_test.go`: task lowering coverage uses package-private
  compiler and runtime helpers.
- `task_runtime_targets_test.go`: task target matrix coverage uses
  package-private runtime target helpers.
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
