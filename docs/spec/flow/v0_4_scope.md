# Tetra v0.4.0 Linux-x64 Production Scope

Status: selected scoped production contract, not a cross-platform or v1.0 claim.

The current release truth remains `docs/spec/core/current_supported_surface.md` until the `v0.4.0`
implementation and release gate are complete. This document records the production scope selected
for the requested `v0.4.0` line after the release objective was narrowed to Linux x64 first, with
EcoNet explicitly excluded.

## Scope Rule

`v0.4.0` is a Linux-x64 production release. It can be promoted only when every selected Linux-x64
feature and runtime decision has implementation, tests, documentation, release-gate evidence, and
security review where applicable.

The machine-readable decision source is `docs/release/v0_4/data/v0_4_0_scope_decisions.json`.

## Required Feature Promotions

| Feature ID                            | Required `v0.4.0` outcome                                                                                                |
| ------------------------------------- | ------------------------------------------------------------------------------------------------------------------------ |
| `language.callable-level1`            | Current production support for the selected non-capturing callable surface.                                              |
| `language.callable-level2`            | Current production support for the selected captured-closure `fnptr` slice with stable diagnostics for excluded escapes. |
| `language.full-first-class-callables` | Current production support for the selected safe first-class callable model.                                             |
| `language.lifetime-ssa`               | Current production local/control-flow lifetime analysis matching the selected ownership/resource claims.                 |
| `stdlib.experimental-mirrors`         | Current compatibility mirror policy with stable callers directed to `lib.core.*`.                                        |
| `ui.metadata-v1`                      | Current UI metadata contract for the selected Linux-x64 UI/native shell surface.                                         |
| `actors.distributed-runtime`          | Current Linux-x64 distributed actor runtime evidence.                                                                    |
| `ui.native-runtime`                   | Current Linux-x64 native UI runtime evidence.                                                                            |

## Required Target Runtime Promotions

| Target      | Required `v0.4.0` outcome                                          |
| ----------- | ------------------------------------------------------------------ |
| `linux-x64` | Host-native build and run evidence passes under the `v0.4.0` gate. |

## Explicit Non-Goals

These entries are not production requirements for the scoped `v0.4.0` release:

- `eco.distributed-network` / EcoNet / hosted production TetraHub networking.
- `language.full-v1-guarantees`.
- `wasm.runtime-execution`, `wasm32-wasi`, and `wasm32-web` production runtime claims.
- `windows-x64` and `macos-x64` production runtime evidence.
- GTK/Qt/OS toolkit native UI backends, broad platform accessibility integration, and non-Linux
  native UI runtime claims.

Existing implementation or experimental support outside this list may remain in the repository, but
it must not be used as a `v0.4.0` production claim unless a future scope decision explicitly
promotes it.

## Required Release Artifacts

Before `v0.4.0` can be marked current, these artifacts must exist and point at the same intended
release commit:

- `docs/checklists/v0_4_0_release_gate.md`
- `docs/release-notes/v0_4_0.md`
- `docs/release/v0_4/v0_4_0_final_handoff.md`
- `scripts/release/v0_4_0/gate.sh`
- `scripts/release/v0_4_0/security-review.sh`
- `reports/v0.4.0/features.json`
- `reports/v0.4.0/targets.json`
- `reports/v0.4.0/linux-host-smoke.json`
- `reports/v0.4.0/distributed-actors-linux-x64.json`
- `reports/v0.4.0/native-ui-linux-x64.json`
- a clean-worktree release status when tagging with `--require-clean`

## Verification Envelope

The final Linux-x64 gate must include at least:

```sh
./tetra version
./t version
./tetra features --format=json
./tetra targets --format=json
go test ./compiler/... ./cli/... ./tools/... -count=1
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
go run ./cli/cmd/tetra smoke --target linux-x64 --run=true --report reports/v0.4.0/linux-host-smoke.json
bash scripts/release/v0_4_0/distributed-actors-linux-x64-smoke.sh --report-dir reports/v0.4.0
bash scripts/release/v0_4_0/native-ui-linux-x64-smoke.sh --report-dir reports/v0.4.0
go run ./tools/cmd/validate-v0-4-readiness --features reports/v0.4.0/features.json --targets reports/v0.4.0/targets.json --manifest docs/generated/manifest.json --scope-decisions docs/release/v0_4/data/v0_4_0_scope_decisions.json
git diff --check
git status --porcelain --untracked-files=all
bash scripts/release/v0_4_0/gate.sh --report-dir <report-dir> --require-clean
```
