# Tetra v0.4.0 Scope Decisions

Status: Linux-x64 production scope selected.

Decision basis: the requested objective was narrowed to Linux x64 first, and
EcoNet was explicitly excluded. Therefore `v0.4.0` is a scoped production
release: selected Linux-x64 language/runtime behavior must be real and
release-evidenced. The canonical gate must also produce Memory Production Core,
Parallelism Production Core, and Compiler Production Core artifacts. EcoNet,
non-Linux targets, WASM target runtimes, and future v1 guarantees are outside
this production claim.

| Kind | ID | Current status | Decision | Notes |
| --- | --- | --- | --- | --- |
| `feature` | `language.callable-level1` | `current` | `implement` | production non-capturing callable Level 1 slice |
| `feature` | `language.callable-level2` | `current` | `implement` | production captured-closure `fnptr` Level 2 slice with stable diagnostics for excluded escapes |
| `feature` | `language.lifetime-ssa` | `current` | `implement` | production local/control-flow lifetime join solver for selected ownership/resource flows |
| `production-core` | `memory.production-core` | `gate-required` | `implement-production-evidence` | Linux-x64 memory ownership/allocation/finalization evidence via `tetra.memory.production.v1` |
| `production-core` | `parallel.production-core` | `gate-required` | `implement-production-evidence` | Linux-x64 tasks/scheduling/cancellation evidence via `tetra.parallel.production.v1` |
| `production-core` | `compiler.production-core` | `gate-required` | `implement-production-evidence` | Linux-x64 compile/run/object/interface/WASM emission evidence via `tetra.compiler.production.v1` |
| `feature` | `stdlib.experimental-mirrors` | `current` | `implement` | compatibility mirrors that forward to `lib.core.*`; stable callers should import `lib.core.*` directly |
| `feature` | `ui.metadata-v1` | `current` | `implement` | production UI metadata contract for the selected Linux-x64 UI/native shell surface |
| `feature` | `wasm.runtime-execution` | `current` | `exclude-from-v0.4.0-prod` | outside the Linux-x64-only production claim |
| `feature` | `language.full-v1-guarantees` | `planned` | `exclude-from-v0.4.0-prod` | future v1.0 release-contract label, not a v0.4.0 requirement |
| `feature` | `language.full-first-class-callables` | `current` | `implement` | selected safe first-class callable model is production for v0.4.0 |
| `feature` | `eco.distributed-network` | `post-v1` | `exclude-from-v0.4.0-prod` | explicitly excluded from the initial Linux-x64 production scope |
| `feature` | `actors.distributed-runtime` | `current` | `implement` | Linux-x64 distributed actor runtime path with executable smoke evidence |
| `feature` | `ui.native-runtime` | `current` | `implement` | Linux-x64 native UI runtime path with executable smoke evidence |
| `target-runtime` | `linux-x64` | `supported` | `implement-production-runtime` | sole `v0.4.0` production runtime target |
| `target-runtime` | `windows-x64` | `supported` | `exclude-from-v0.4.0-prod` | outside the Linux-x64-only production claim |
| `target-runtime` | `macos-x64` | `supported` | `exclude-from-v0.4.0-prod` | outside the Linux-x64-only production claim |
| `target-runtime` | `wasm32-wasi` | `supported` | `exclude-from-v0.4.0-prod` | outside the Linux-x64-only production claim |
| `target-runtime` | `wasm32-web` | `supported` | `exclude-from-v0.4.0-prod` | outside the Linux-x64-only production claim |

Completion rule: every implemented Linux-x64 `v0.4.0` production-scope decision
must have implementation, tests, docs, and release-gate evidence, and the
production-core artifacts above appear in the final gate hash manifest.
Excluded entries must not be described as part of the `v0.4.0` production
claim.
