# Tetra v1.0 Canonical Scope

Status: pre-release scope contract. This document defines what must be true
before a build can be labeled `v1.0.0`; it is not a claim that the current
`v0.1.3` baseline already satisfies the scope.

The current public release gate is `scripts/release_v0_1_3_gate.sh`. A true
`v1.0.0` gate must be reintroduced from this contract when the version is
promoted to `v1.0.x` and every mandatory artifact below has fresh evidence.

## Mandatory Language Scope

| Feature | v1.0 decision | Required evidence | Blocking gate | Owner / agent slot |
| --- | --- | --- | --- | --- |
| Flow syntax as canonical source syntax | Required | Flow-only scan and formatter check over `examples`, `lib`, `__rt`, and `compiler/selfhostrt` | `go run ./tools/cmd/validate-flow-only examples lib __rt compiler/selfhostrt`; `./tetra fmt --check examples lib __rt compiler/selfhostrt` | frontend agent |
| Parser and diagnostics for supported Flow forms | Required | Frontend parser/diagnostic tests and docs verification | `go test ./compiler/internal/frontend/... -count=1` | frontend agent |
| Stable primitive, structural, optional, typed-error, no-payload enum, generic, protocol, extension, and module contracts | Required | Compiler tests plus spec alignment | `go test ./compiler/... -run 'Type|Inference|Enum|Optional|Protocol|Extension|Module' -count=1` | semantics agent |
| Ownership, lifetime, island, actor/task transfer, and race-safety checks | Required before release label | Negative tests for use-after-move, escaping borrows, aliasing, invalid transfers, and actor/task races | `go test ./compiler/... -run 'Ownership|Borrow|Lifetime|Island|Actor|Task' -count=1` | safety agent |
| Effects, capabilities, unsafe boundaries, and public diagnostics | Required | Spec/docs validation, stable module effect metadata audit, and diagnostics shape tests | `go test ./compiler/... -run 'Unsafe|Capability|Effect|Privacy|Consent|Budget|MMIO|Mem' -count=1`; `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`; `go test ./tools/cmd/validate-diagnostic/... -count=1` | safety/tooling agent |
| Privacy, consent, and budget contract | Required as static v1 MVP | Privacy clauses, consent-token signatures, and deterministic budget guards are checked and lowered; distributed/runtime-wide accounting is post-v1 | `go test ./compiler/... -run 'Privacy|Consent|Budget|Effect' -count=1` | safety/tooling agent |
| Async function MVP | Required as checked synchronous lowering | `async func`/`await` parse, check, lower, and reject unsupported async typed-error propagation with a stable diagnostic | `go test ./compiler/... -run 'Async|Await|Task|TypedError' -count=1` | runtime agent |
| Task runtime MVP | Required for local typed task handles | Spawn/join/group builtins are typed, `uses runtime` gated, documented, and covered by bounded stress | `go test ./compiler/... -run 'Task|Runtime|Async|Stress' -count=1` | runtime agent |
| Actors runtime MVP | Required for local actor runtime on supported native targets | Tagged messages, runtime selection, self-host/builtin parity, ownership checks, and target build matrix are tested; distributed actors are post-v1 | `go test ./compiler/... -run 'Actor|Actors|Runtime|Ownership' -count=1` | runtime agent |
| Runtime ABI and TOBJ linking | Required | Reserved `__tetra_*` symbols, TOBJ target metadata, runtime override, repeated link objects, and mismatch diagnostics are tested | `go test ./compiler/... -run 'Runtime|ABI|Object|Link' -count=1` | runtime agent |
| UI syntax and accessibility metadata | Required as metadata UI surface | `docs/spec/ui_v1.md`, UI parser/semantic/lowering tests, native shell sidecar smoke, and web browser smoke evidence | `go test ./compiler/... -run 'UI|View|State|Style|Accessibility|NativeShell' -count=1`; `bash scripts/release_v1_0_web_smoke.sh`; `./tetra smoke --target linux-x64 --run=false` | UI agent |

## Mandatory Tooling, CLI, LSP, Docs, And Eco Scope

| Feature | v1.0 decision | Required evidence | Blocking gate | Owner / agent slot |
| --- | --- | --- | --- | --- |
| CLI commands: `check`, `build`, `run`, `fmt`, `test`, `doc`, `lsp`, `eco`, `clean`, `version` | Required | CLI package tests and release gate command coverage | `go test ./cli/... -count=1`; current gate: `bash scripts/release_v0_1_3_gate.sh`; future v1 gate: blocked until promotion | CLI agent |
| Formatter contract | Required | Idempotence and comment-preservation coverage for release sources | `go test ./compiler/... -run 'Format|Formatter|Comment' -count=1`; `./tetra fmt --check examples lib __rt compiler/selfhostrt` | tooling agent |
| Docs manifest, doctests, and generated API docs | Required | Manifest validation, docs verification, API docs validation | `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`; `go run ./tools/cmd/validate-api-docs --docs <generated-docs>` | docs agent |
| JSON diagnostics, test reports, target reports, doctor reports, smoke reports | Required | Schema validator tests and release gate validator steps | `go test ./tools/... -count=1`; `bash scripts/test_all.sh --full --keep-going` | tools agent |
| LSP stdio baseline | Required | LSP validator and transcript coverage | `go test ./tools/cmd/validate-lsp-stdio/... ./tools/cmd/validate-lsp-smoke/... -count=1` | LSP agent |
| Local Eco package lifecycle | Required | Capsule verify/pack/unpack/lock/vault/publish metadata fixtures | relevant Eco validator tests; `bash scripts/test_all.sh --full --keep-going` | Eco agent |

## Target Matrix

| Target | v1.0 status required before release | Required evidence |
| --- | --- | --- |
| `linux-x64` | Native build and host smoke when running on Linux | `./tetra smoke --target linux-x64 --run=true --report <path>` |
| `macos-x64` | Build-only cross-target smoke | `./tetra smoke --target macos-x64 --run=false --report <path>` |
| `windows-x64` | Build-only cross-target smoke | `./tetra smoke --target windows-x64 --run=false --report <path>` |
| `wasm32-wasi` | Build-only smoke plus WASI runner smoke | `bash scripts/release_v1_0_wasi_smoke.sh --report <path>` |
| `wasm32-web` | Build-only smoke plus browser automation smoke | `bash scripts/release_v1_0_web_smoke.sh --report <path>` |

## Explicitly Post-v1 Unless Promoted By Review

Promotion requires `docs/release/post_v1_promotion_checklist.md` evidence in
the same branch state as the implementation, tests, docs, gates, compatibility
notes, and security review when applicable.

- Distributed EcoNet and TetraHub production publishing.
- Proof-carrying capsules and global trust scoring.
- EcoOracle, live evolution, time-travel execution, and multiverse optimizer
  features.
- Advanced AI/model types and model-runtime integration.
- Distributed actors beyond the release actor/task safety contract.
- Async typed-error propagation, cancellation, and structured concurrency beyond
  the checked synchronous async MVP.
- Enum payload cases, payload constructors, and payload destructuring patterns
  beyond the no-payload enum and exhaustive match MVP.
- Distributed privacy/consent enforcement and runtime-wide resource-budget
  accounting beyond deterministic local guard lowering.
- Real macOS/Windows host execution evidence for actor/runtime binaries when
  collecting it from non-matching Linux hosts.
- Full native widget rendering, platform accessibility integration, runtime UI
  event dispatch, and layout engines beyond the UI v1 metadata artifacts in
  `docs/spec/ui_v1.md`.
- Any feature still labeled `planned`, `beta`, `deferred-post-v1`, or
  `blocked-by-prerequisite` in the release checklist.

## Release Closure Rule

The release checklist, release notes, and artifact archive must cite this
document. A checkbox may be marked complete only when the implementation,
tests, documentation, and artifact evidence exist in the same branch state.
