# P7 Transient IR Release Plan

**Goal:** Make compiler phase profiles distinguish transient PLIR/lowering/allocation-plan retention
from the live checked program, and release transient references before the `module_codegen` phase
snapshot.

**Context:** P7.2 requires shortening heavyweight compiler object lifetimes. Current phase profiles
already prove native object references are dropped before report generation, but the
`compileNativeModulePlan` profile does not expose whether transient PLIR/allocation summary state
is still retained after module codegen completes.

## Task 1 - Add RED Profile Coverage

- **Goal:** Add a regression proving transient compiler IR/allocation-plan counts become zero by
  the `module_codegen` phase.
- **Files:** `compiler/compiler_external_test.go`.
- **Approach:** Build a small native executable with compiler phase profiling enabled.
  Assert `plir_construction`, `allocation_planning`, or `ir_lowering` records nonzero transient
  counts, and `module_codegen`, `object_retention_link`, `report_generation`, and `final_cleanup`
  record zero transient counts.
- **Verification:** `GOTELEMETRY=off GOCACHE="$(pwd)/.cache/go-build-ram-p7-transient-ir-red" go test -count=1 ./compiler -run 'TestP7CompilerPhaseProfileReleasesTransientIRBeforeModuleCodegen' -v`.
- **Done when:** The test fails first because the transient profile fields are missing or retained.

## Task 2 - Release And Profile Transient State

- **Goal:** Clear transient allocation-plan/IR summary references once module workers finish and
  record that boundary in the phase profile.
- **Files:** `compiler/compiler_facade.go`, `compiler/compiler_phase_profile.go`.
- **Approach:** Add explicit transient count fields to the phase profile. Populate them during
  PLIR/allocation/lowering phases; after workers finish and validation no longer needs the
  allocation plan or summary program, nil those references before the `module_codegen` capture.
- **Verification:** Targeted GREEN test plus focused P7 profile/hash tests.
- **Done when:** The profile shows nonzero transient counts during construction/lowering and zero
  transient counts from `module_codegen` onward.

## Task 3 - Evidence And Kernel Update

- **Goal:** Keep durable goal evidence current without claiming final P7 completion.
- **Files:** `.workflow/tetra-ram-optimization-master-plan/**`,
  `docs/spec/telemetry/process_rss_telemetry.md`.
- **Approach:** Document the new profile fields and record RED/GREEN/focused/full checks.
- **Verification:** `./compiler/...`, docs verifier, `git diff --check`, `graphify update .`,
  and persistent Go cache cleanup.
- **Done when:** Kernel state points to this slice as verified and final P7 remains open for
  baseline-vs-candidate RSS gates.
