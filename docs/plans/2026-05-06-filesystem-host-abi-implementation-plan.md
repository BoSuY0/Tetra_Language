# Filesystem Host ABI Implementation Plan

**Goal:** promote the first `lib.core.filesystem` host-backed API slice from
placeholder path helpers to verified runtime behavior without broad production
claims.

**Context:** `validate-tooling-stdlib-readiness` still blocks the active
tooling+stdlib production goal on `stdlib.core-current` and
`docs/spec/stdlib.md` placeholder phrases. Observed repo pattern for
runtime-backed APIs is `core.*` builtin signature/effects, lowering to a
`__tetra_*` runtime symbol, required-symbol validation, manifest exposure,
runtime ABI docs, native runtime exports, examples, and release-gate evidence.

**Scope decision pending:** this plan assumes the first production slice may be
`linux-x64` host execution with explicit unsupported diagnostics for other
targets until their host filesystem runtime is implemented. If the release
claim must cover every native target immediately, split this plan by platform
before implementation.

## Task 1: Freeze The Filesystem Host Contract

- **Goal:** define the smallest real filesystem API that can be implemented and
  tested end to end.
- **Files:** inspect `lib/core/filesystem.tetra`, `docs/spec/stdlib.md`,
  `docs/user/standard_library_guide.md`, `compiler/internal/semantics/builtins.go`,
  `compiler/internal/lower/lower.go`, and `docs/spec/runtime_abi.md`.
- **Approach:** add a capability-gated API shape:
  `exists(path: String, io_cap: cap.io) -> Bool uses io`, backed by
  `core.fs_exists(path, io_cap) -> bool uses io`. Keep existing pure path
  helpers as pure utilities.
- **Verification:** docs review plus `go test ./tools/cmd/verify-docs -count=1`
  after docs are updated.
- **Done when:** the docs describe exact arguments, effects, return semantics,
  unsupported target behavior, and failure mode without using placeholder
  language for the implemented slice.
- **Notes:** `String`/`str` is passed as `ptr,len`; the runtime symbol must not
  assume NUL-terminated paths.

## Task 2: Add Semantic Builtin Evidence

- **Goal:** make `core.fs_exists` a typed, effect-checked builtin.
- **Files:** modify `compiler/internal/semantics/builtins.go`; add or extend
  tests in `compiler/internal/semantics/manifest_test.go`,
  `compiler/effects_test.go`, and nearby builtin manifest tests.
- **Approach:** add a builtin signature accepting `str` plus `cap.io` and
  returning `bool`, mark effects as `io`, and keep the `cap.io` token as the
  capability gate even for direct `core.fs_exists` calls.
- **Verification:** `go test ./compiler/internal/semantics ./compiler -run 'Builtin|Manifest|Effect|filesystem|Filesystem' -count=1`.
- **Done when:** the builtin appears in the manifest with stable effects and
  semantic checking rejects missing `uses io`.
- **Notes:** avoid granting host access by import alone; require an explicit
  `cap.io` token at the public stdlib layer.

## Task 3: Lower To A Runtime Symbol

- **Goal:** route `core.fs_exists(path)` to a runtime symbol with a stable ABI.
- **Files:** modify `compiler/internal/lower/lower.go`; add tests near existing
  lowering tests, likely `compiler/internal/lower/verify_test.go` or a focused
  lowering test that already inspects `IRCall`.
- **Approach:** lower `core.fs_exists` to
  `IRCall{Name: "__tetra_fs_exists", ArgSlots: 3, RetSlots: 1}`.
- **Verification:** `go test ./compiler/internal/lower -run 'Filesystem|FS|Builtin|IRCall' -count=1`.
- **Done when:** a source call to the builtin emits the exact runtime symbol and
  slot counts.
- **Notes:** the first two argument slots are the `str` pointer and length; the
  third is the `cap.io` token.

## Task 4: Add Runtime ABI Validation

- **Goal:** require filesystem runtime symbols when filesystem host builtins are
  used.
- **Files:** modify `compiler/compiler.go`, `compiler/manifest.go`,
  `compiler/manifest_test.go`, and `docs/spec/runtime_abi.md`.
- **Approach:** add `requiredFilesystemRuntimeSymbols()` with
  `__tetra_fs_exists`, add `runtimeObjectSignature("__tetra_fs_exists") =
  paramSlots: 3, returnSlots: 1`, expose it in the manifest, detect source use
  of `core.fs_exists`, and validate runtime objects before linking.
- **Verification:** `go test ./compiler -run 'Runtime|Manifest|Filesystem|Override|Missing' -count=1`.
- **Done when:** missing or wrong-signature runtime objects fail before platform
  linking, and the manifest publishes the required filesystem symbol set.
- **Notes:** mirror the existing time runtime pattern rather than creating a
  parallel validation path.

## Task 5: Implement Linux Runtime Behavior

- **Goal:** make `__tetra_fs_exists(ptr,len,cap)` return `1` for an existing
  path and `0` for missing or invalid paths on `linux-x64`.
- **Files:** inspect and modify `compiler/internal/actorsrt/linux_x64.go` and
  related emitter helpers in `compiler/internal/actorsrt/linux_x64_emit.go`;
  inspect `compiler/internal/backend/x64` emitter support before choosing the
  instruction sequence.
- **Approach:** add a runtime function that copies the `ptr,len` path into a
  NUL-terminated temporary buffer, performs a Linux host filesystem existence
  check, cleans up temporary memory, and returns a boolean `i32`.
- **Verification:** focused actor runtime object tests plus a build-and-run
  smoke on `linux-x64`.
- **Done when:** generated runtime object exports `__tetra_fs_exists` with the
  expected signature and the program smoke observes both existing and missing
  paths.
- **Notes:** do not pass `ptr,len` directly to a syscall requiring C strings.
  Bound maximum path length or return `0` on allocation/copy failure; document
  the behavior.

## Task 6: Add Explicit Unsupported Diagnostics

- **Goal:** prevent silent false production claims on targets without host
  filesystem runtime behavior.
- **Files:** modify the target-runtime diagnostic paths in `compiler/compiler.go`
  and any WASM-specific runtime rejection tests.
- **Approach:** classify `__tetra_fs_*` calls as filesystem runtime use. For
  unsupported targets, emit a stable diagnostic saying filesystem runtime is
  unsupported on that target.
- **Verification:** `go test ./compiler -run 'WASM|Runtime|Filesystem|Target' -count=1`.
- **Done when:** WASM and unsupported native target attempts fail with stable
  diagnostics before producing a misleading artifact.
- **Notes:** if macOS/windows runtime support is required in the first slice,
  replace this task with platform implementations and tests.

## Task 7: Promote Stdlib Wrapper And Evidence

- **Goal:** expose the host-backed API through `lib.core.filesystem` and update
  release evidence.
- **Files:** modify `lib/core/filesystem.tetra`,
  `examples/core_filesystem_smoke.tetra`, `docs/spec/stdlib.md`,
  `docs/user/standard_library_guide.md`, `compiler/features.go`, and release
  audit docs.
- **Approach:** add the public wrapper requiring `cap.io`, keep pure helpers,
  add a smoke that checks an existing project file and a missing path, and
  remove placeholder language only for the implemented host-backed slice.
- **Verification:** `./tetra smoke --list --format=json`, `go run ./tools/cmd/validate-smoke-list --report <report> --examples-root examples`,
  `./tetra doc lib/core lib/experimental examples/core_*_smoke.tetra`,
  `go run ./tools/cmd/validate-api-docs --docs <docs>`, and
  `go run ./tools/cmd/validate-tooling-stdlib-readiness --features <features>`.
- **Done when:** filesystem no longer trips the readiness gate and the docs
  explain exactly which filesystem behavior is production-backed.
- **Notes:** networking and crypto will remain blockers until their own
  production slices exist.

## Task 8: Release-Gate Integration

- **Goal:** ensure the new filesystem evidence is part of repeatable release
  validation.
- **Files:** inspect `scripts/release/v0_4_0/gate.sh`,
  `scripts/ci/test-all.sh`, and related `tools/scriptstest` coverage before
  editing.
- **Approach:** add the filesystem smoke/doc/readiness commands to the relevant
  release gate only after the focused implementation tests pass.
- **Verification:** focused `tools/scriptstest` run for the edited gate plus
  the dedicated readiness command.
- **Done when:** a clean release-gate path fails without filesystem evidence and
  passes when the implemented artifacts are present.
- **Notes:** do not make the gate depend on host paths that vary across
  machines; use repository files and temp missing paths.
