# ABI Verification v1 Audit

Status: P21.1 evidence audit for the Ideal Master Plan goal loop.

## Scope

P21.1 adds an evidence contract for ABI verification across these targets:

- `linux-x64` SysV
- `linux-x86` i386 SysV
- `linux-x32` x32 SysV
- `macos-x64` SysV
- `windows-x64` Win64
- `wasm32-wasi`
- `wasm32-web`

The machine-readable report contract is:

- schema: `tetra.abi.verification.v1`
- scope: `p21.1_abi_verification`

`BuildP21ABIVerificationReport` returns one target row for every required target and task rows for:

- `abi_test_corpus`
- `struct_enum_slice_string_return_validation`
- `call_boundary_validation`
- `ffi_repr_c_tests`

`ValidateP21ABIVerificationReport` rejects missing targets, missing task rows, placeholder evidence,
fake runtime-execution claims, fake default-struct C ABI claims, and fake wasm native C aggregate
ABI claims.

## Evidence

- Native targets reuse `RunTargetABIChecks` corpus rows in `compiler/compiler_evidence_gates.go`:
  target models, x86 i386 SysV classifier checks, x64 SysV/Win64/x32 classifier checks, varargs and
  aggregate checks, call-boundary metadata, object ABI smokes, and target-specific FFI object
  smokes.
- Native exported aggregate FFI boundaries remain guarded by `compiler/compiler_build_runtime.go`
  and diagnostics in `compiler/compiler_suite_test.go`; public aggregate FFI requires explicit
  `repr(C)`.
- `wasm32-wasi` and `wasm32-web` now participate in `RunTargetABIChecks`. Their ABI checks verify
  ILP32 target metadata, i32 slot ABI metadata, struct/slice/String/enum return layout sizes and
  alignments, and backend `IRCall` arg/return slot matching.
- WASM call boundary validation uses existing backend metadata validation in
  `compiler/internal/backend/wasm32_wasi/codegen.go` and
  `compiler/internal/backend/wasm32_web/codegen.go`.

## Non-Claims

- No runtime execution claim for build-only native targets or wasm targets.
- No C ABI claim for default structs.
- No native C aggregate ABI claim for wasm targets.
- No performance claim.
- No safe-program semantics change.
