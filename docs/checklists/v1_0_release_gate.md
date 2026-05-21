# v1.0 Release Gate Checklist

Status: active release-gate scaffold for the future `v1.0.0` release line.
This checklist is not release proof until every unchecked row cites fresh
evidence from the exact release branch state. The current public release line
remains `v0.4.0`; `scripts/release/v1_0/gate.sh` must block at version
preflight until freshly bootstrapped `./tetra version` reports `v1.0.0`.

Scope contract: `docs/spec/v1_scope.md`.
Current supported surface: `docs/spec/current_supported_surface.md`.
Artifact policy: `docs/release/artifact_policy.md`.
Cut guide: `docs/release/v1_0_release_cut_guide.md`.
Final handoff template: `docs/release/v1_0_final_handoff.md`.

## Hard Blockers

- [ ] Version preflight: `bash scripts/dev/bootstrap.sh`, `./tetra version`, and
      `./t version` prove the exact intended `v1.0.0` candidate. Any `v0.x`
      output blocks before mandatory checks run.
- [ ] Final v1 gate: `bash scripts/release/v1_0/gate.sh --report-dir <report-dir>`
      completes with `status: pass`, `failed_count: 0`, and artifact id
      `tetra.release.v1_0.gate-report.v1`.
- [ ] Scope closure: every mandatory row in `docs/spec/v1_scope.md` has same
      branch evidence for implementation, tests, docs, and artifacts.
- [ ] Safety evidence closure: ownership, lifetime, resource, actor/task,
      unsafe, capability, effect, privacy, consent, budget, MMIO, and memory
      checks cite the aggregate compiler command, docs verifier, and concrete
      `<report-dir>/logs/*safety*` and `<report-dir>/logs/*docs*` paths from
      this exact branch state.
- [ ] No post-v1 leakage: features listed under `Explicitly Post-v1 Unless
      Promoted By Review` remain blocked, deferred, or have approved promotion
      evidence in `docs/release/post_v1_promotion_checklist.md`.
- [ ] Security signoff: reviewer identity, reviewed commit, report directory,
      evidence commands, artifact hashes, release decision, and residual risks
      are complete for the exact candidate.
- [ ] Release notes and handoff: `docs/release-notes/v1_0.md` and
      `docs/release/v1_0_final_handoff.md` cite the final report directory and
      do not reuse stale evidence from another commit, branch, or version.
- [ ] Evidence freshness: every checked row cites command output from the same
      commit, branch, version, and report directory as the final gate.
- [ ] Handoff/signoff lint: `docs/release/v1_0_final_handoff.md` and the final
      security signoff have no unresolved `TODO`, `TBD`, or template placeholder
      text.

## Required Commands

Each command must be run against the final `v1.0.0` candidate unless the final
v1 gate summary cites the same command as a passing gate step.

- [ ] `bash scripts/release/v1_0/gate.sh --report-dir <report-dir>`
- [ ] `bash scripts/dev/bootstrap.sh`
- [ ] `./tetra version`
- [ ] `./t version`
- [ ] `go test ./compiler/internal/frontend/... -count=1`
- [ ] `go test ./compiler/... -run 'Closure|FunctionType|Callable|Capsule|Property' -count=1`
- [ ] `go test ./compiler/... -run 'Type|Inference|Enum|Optional|Protocol|Extension|Module' -count=1`
- [ ] `go test ./compiler/... -run 'Ownership|Borrow|Consume|Inout|Lifetime|Resource|Island|Actor|Task|Unsafe|Capability|Effect|Privacy|Consent|Budget|MMIO|Mem' -count=1`
- [ ] `go test ./compiler/... -run 'Privacy|Consent|Budget|Effect' -count=1`
- [ ] `go test ./compiler/... -run 'Async|Await|Task|TypedError' -count=1`
- [ ] `go test ./compiler/... -run 'Task|Runtime|Async|Stress' -count=1`
- [ ] `go test ./compiler/... -run 'Actor|Actors|Runtime|Ownership' -count=1`
- [ ] `go test ./compiler/... -run 'Runtime|ABI|Object|Link' -count=1`
- [ ] `go test ./compiler/... -run 'UI|View|State|Style|Accessibility|NativeShell' -count=1`
- [ ] `go test ./cli/... -count=1`
- [ ] `go test ./tools/... -count=1`
- [ ] `bash scripts/ci/test-all.sh --full --keep-going --report-dir <report-dir>/artifacts/test-all`
- [ ] `go run ./tools/cmd/validate-flow-only examples lib __rt compiler/selfhostrt`
- [ ] `./tetra fmt --check examples lib __rt compiler/selfhostrt`
- [ ] `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`
- [ ] `go run ./tools/cmd/validate-api-docs --docs <generated-docs>`
- [ ] `go test ./tools/cmd/validate-diagnostic/... -count=1`
- [ ] `go test ./tools/cmd/validate-lsp-stdio/... ./tools/cmd/validate-lsp-smoke/... -count=1`
- [ ] `./tetra smoke --target linux-x64 --run=true --report <report-dir>/artifacts/host-smoke.json`
- [ ] `./tetra smoke --target macos-x64 --run=false --report <report-dir>/artifacts/macos-smoke.json`
- [ ] `./tetra smoke --target windows-x64 --run=false --report <report-dir>/artifacts/windows-smoke.json`
- [ ] `./tetra smoke --target wasm32-wasi --run=false --report <report-dir>/artifacts/wasm32-wasi-smoke.json`
- [ ] `./tetra smoke --target wasm32-web --run=false --report <report-dir>/artifacts/wasm32-web-smoke.json`
- [ ] `bash scripts/release/v1_0/wasi-smoke.sh --report <report-dir>/artifacts/wasi-smoke.json`
- [ ] `bash scripts/release/v1_0/web-smoke.sh --report <report-dir>/artifacts/web-ui-smoke.json`
- [ ] `bash scripts/release/v1_0/security-review.sh --signoff <report-dir>/artifacts/security-review.md`
- [ ] `bash scripts/release/v1_0/binary-size.sh --report <report-dir>/artifacts/binary-size-thresholds.json`
- [ ] `bash scripts/release/v1_0/reproducible-build.sh --report <report-dir>/artifacts/reproducible-build.json`
- [ ] `go run ./tools/cmd/validate-release-state --expected-version v1.0.0 --format=text --report-dir <report-dir>`
- [ ] `go run ./tools/cmd/validate-artifact-hashes --manifest <report-dir>/artifacts/artifact-hashes.json`
- [ ] `! rg -n 'TODO|TBD|<[A-Za-z0-9_ ./:-]+>' docs/release/v1_0_final_handoff.md <report-dir>/artifacts/security-review.md`
- [ ] Cross-reference audit: `docs/spec/v1_scope.md`, `docs/checklists/v1_0_release_gate.md`, and `docs/release/v1_0_final_handoff.md` all cite each other and the same final `<report-dir>`.
- [ ] `git diff --check`

## Required Artifacts

The final handoff must cite concrete paths under one fresh `<report-dir>`.

- [ ] `<report-dir>/summary.json`
- [ ] `<report-dir>/summary.md`
- [ ] `<report-dir>/logs/*.log`
- [ ] `<report-dir>/artifacts/release-state.json`
- [ ] `<report-dir>/artifacts/release-state.txt`
- [ ] `<report-dir>/artifacts/artifact-hashes.json`
- [ ] `<report-dir>/artifacts/known_issues.md`
- [ ] `<report-dir>/artifacts/security-review.md`
- [ ] `<report-dir>/artifacts/security-review.md.sha256`
- [ ] `<report-dir>/artifacts/reproducible-build.json`
- [ ] `<report-dir>/artifacts/binary-size-thresholds.json`
- [ ] `<report-dir>/artifacts/performance-regression.json`
- [ ] `<report-dir>/artifacts/targets.json`
- [ ] `<report-dir>/artifacts/doctor.json`
- [ ] `<report-dir>/artifacts/tetra-test-report.json`
- [ ] `<report-dir>/artifacts/smoke-list.json`
- [ ] `<report-dir>/artifacts/host-smoke.json`
- [ ] `<report-dir>/artifacts/linux-smoke.json`
- [ ] `<report-dir>/artifacts/macos-smoke.json`
- [ ] `<report-dir>/artifacts/windows-smoke.json`
- [ ] `<report-dir>/artifacts/wasm32-wasi-smoke.json`
- [ ] `<report-dir>/artifacts/wasm32-web-smoke.json`
- [ ] `<report-dir>/artifacts/wasi-smoke.json`
- [ ] `<report-dir>/artifacts/web-ui-smoke.json`
- [ ] `<report-dir>/artifacts/api-diff/api-docs.md`
- [ ] `<report-dir>/artifacts/api-diff/api-diff.json`
- [ ] `<report-dir>/artifacts/tetra-docs.md`
- [ ] `<report-dir>/artifacts/test-all/summary.json`
- [ ] `<report-dir>/artifacts/test-all/summary.md`

## Scope Evidence Matrix

| Scope area | Required evidence | Artifact or log to cite | Status |
| --- | --- | --- | --- |
| Flow syntax and formatter | Flow-only scan plus formatter check over `examples`, `lib`, `__rt`, and `compiler/selfhostrt` | `<report-dir>/logs/*flow-only*`; `<report-dir>/logs/*formatter*` | blocked until fresh v1 evidence |
| Frontend parser and diagnostics | Frontend package tests and diagnostic validators | `<report-dir>/logs/*frontend*`; diagnostic validator logs | blocked until fresh v1 evidence |
| Function-type/callable MVP boundaries | Compiler tests prove supported direct-local callable subset and stable diagnostics for unsupported callable forms | compiler callable/closure test log | blocked until fresh v1 evidence |
| Capsule metadata declaration MVP | Frontend + compiler tests prove metadata-only capsule acceptance and validation without runtime coupling | frontend/compiler capsule test logs | blocked until fresh v1 evidence |
| Stable type and module contracts | Compiler tests for type, inference, enum, optional, protocol, extension, and module behavior | compiler test log | blocked until fresh v1 evidence |
| Safety closure: ownership, lifetimes, resources, islands, actors/tasks, unsafe, capabilities, effects, privacy, consent, budgets, MMIO, and memory | Aggregate compiler safety command plus docs verification and diagnostic shape tests from the same branch state | `<report-dir>/logs/*safety*`; `<report-dir>/logs/*docs*`; diagnostic log | blocked until fresh v1 evidence |
| Async, task runtime, and actor runtime MVP | Async/task/actor/runtime tests plus target smoke evidence | runtime test logs; smoke artifacts | blocked until fresh v1 evidence |
| Runtime ABI and TOBJ linking | Runtime ABI, object, link, override, and mismatch tests | runtime ABI test log | blocked until fresh v1 evidence |
| UI metadata surface | UI compiler tests, `docs/spec/ui_v1.md`, native shell smoke, and web smoke | UI test log; web smoke artifact; target smoke artifact | blocked until fresh v1 evidence |
| CLI and tooling | CLI package tests, tools package tests, JSON validators, release-state audit | CLI/tools logs; release-state artifacts | blocked until fresh v1 evidence |
| Docs and API docs | Manifest validation, docs verification, generated API docs validation | docs logs; `api-docs.md`; API diff artifacts | blocked until fresh v1 evidence |
| LSP baseline | LSP stdio and smoke validators | LSP test or transcript logs | blocked until fresh v1 evidence |
| Local Eco lifecycle | Eco verify/pack/unpack/lock/vault/publish metadata fixtures | Eco validator logs and artifacts | blocked until fresh v1 evidence |
| Target matrix | Linux host run, macOS/Windows build-only, WASI runner, web browser smoke | target smoke JSON and smoke script artifacts | blocked until fresh v1 evidence |

## Source Of Truth Guardrails

- [ ] No checkbox is marked complete without a command result and artifact path.
- [ ] No generated artifact, smoke report, signoff, or summary is reused from a
      different commit, branch, version, or report directory.
- [ ] Version metadata is not promoted from `v0.4.0` until the mandatory v1
      scope evidence is ready.
- [ ] The final handoff records command exit codes, changed release files,
      known issues, residual risks, and release decision.
