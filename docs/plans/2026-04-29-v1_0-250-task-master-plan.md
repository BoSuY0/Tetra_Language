# Tetra v1.0 Master Execution Plan (250 Tasks)

Status: active execution plan for parallel implementation.

Rules:
- Закриття задачі тільки з evidence: зміни у файлах + команди перевірки + результат.
- Якщо задача виявляється `post-v1`, фіксувати explicit decision у `docs/spec/v1_scope.md`/checklist.
- Після code-змін: focused tests, потім broad verification.
- Після docs/generated змін: `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`.
- Перед фінальним заявленням success: актуальний gate/report у тій самій гілці стану.

## Consolidated Evidence Index

Integrity rule: for any checked task without inline `Evidence:`, the evidence is
the matching epic-range row in this consolidated index plus the corresponding
wave report named in that row. Inline `Evidence:` remains task-specific proof
when present; missing or superseded inline artifact paths are not reused unless
the artifact exists in the current worktree.

| Task range | Epic | Consolidated evidence artifacts | Evidence commands / checks |
| --- | --- | --- | --- |
| T250-001..T250-025 | Scope, Contracts, Governance | `reports/plan250/agent-docs-evidence.md`; `reports/plan250/generated-manifest.json`; `reports/plan250/tetra-docs.md`; `reports/plan250/smoke-list-linux-x64.json`; release docs/spec/checklist files named inline in this range. | `./tetra version`; `./tetra features --format=json`; `go test ./compiler -run FeatureRegistry -count=1`; `go test ./tools/scriptstest -run 'ReleaseV10Gate|CurrentSupportedSurface' -count=1`; `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`; `go run ./tools/cmd/validate-manifest --manifest reports/plan250/generated-manifest.json`; `git diff --check`. |
| T250-026..T250-050 | Frontend (Lexer, Parser, Formatter, Flow) | `reports/plan250/wave2-implC-frontend-evidence.md`; `reports/plan250/frontend-summary.md`; parser/lexer/formatter fixtures named in the wave report. | `go test ./compiler/internal/frontend/... -count=1`; `./tetra fmt --check examples lib __rt compiler/selfhostrt`; `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`; `go run ./tools/cmd/validate-flow-only examples lib __rt compiler/selfhostrt`; targeted `tools/scriptstest` flow/formatter checks. |
| T250-051..T250-075 | Semantics (Types, Generics, Protocols, Enums, Modules) | `reports/plan250/wave2-implC-semantics-evidence.md`; `reports/plan250/semantics-summary.md`; `reports/plan250/tetra-docs.md`. | `go test ./compiler/... -run 'Type|Inference|Enum|Optional|Protocol|Extension|Module|Generic|Conformance' -count=1`; `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`; `go run ./tools/cmd/validate-api-docs --docs reports/plan250/tetra-docs.md`; `go test ./compiler -run FeatureRegistry -count=1`. |
| T250-076..T250-100 | Safety (Ownership, Lifetimes, Effects, Privacy, Budgets) | `reports/plan250/agent-safety-runtime-evidence.md`; `reports/plan250/safety-summary.md`; docs/spec and docs/user files named in the wave report. | `go test ./compiler -run 'Plan250Safety|Plan250Runtime|Plan250Link' -count=1`; `go test ./compiler/... -run 'Ownership|Borrow|Consume|Inout|Lifetime|Resource|Island|Actor|Task|Unsafe|Capability|Effect|Privacy|Consent|Budget|MMIO|Mem' -count=1`; `go test ./tools/cmd/validate-diagnostic/... -count=1`; `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`; `git diff --check`. |
| T250-101..T250-125 | Runtime (Async, Task, Actors, ABI) | `reports/plan250/agent-safety-runtime-evidence.md`; `reports/plan250/runtime-summary.md`; `reports/plan250/runtime-linux-x64-smoke.json`. | `go test ./compiler/... -run 'Async|Await|Task|Runtime|Stress|Actor|Actors|Ownership|ABI|Object|Link|SelfHost' -count=1`; targeted actor/task smoke Go tests; `go test ./compiler/internal/lower ./compiler/internal/actorsrt ./compiler/internal/backend/x64core -run 'Typed|Task|Actor|Ctx|Runtime|Lower|Verify' -count=1`; `./tetra smoke --target linux-x64 --run=true --report reports/plan250/runtime-linux-x64-smoke.json`; `git diff --check`. |
| T250-126..T250-150 | Backends (x64, WASI/Web, UI Metadata) | `reports/plan250/agent-backend-cli-release-evidence.md`; `reports/plan250/wave2-implA-evidence.md`; `reports/plan250/waveA-impl3-backend-summary.md`; `reports/plan250/backend/wasm32-wasi-smoke.json`; `reports/plan250/backend/wasm32-web-smoke.json`; `reports/plan250/backend/wasi-smoke.json`; `reports/plan250/backend/web-ui-smoke.json`. | `go test ./compiler/... -run 'WASM|Web|Wasi|UI|NativeShell|Lower|IR|Backend' -count=1`; `./tetra smoke --target wasm32-wasi --run=false --report reports/plan250/backend/wasm32-wasi-smoke.json`; `./tetra smoke --target wasm32-web --run=false --report reports/plan250/backend/wasm32-web-smoke.json`; `bash scripts/release/v1_0/wasi-smoke.sh --report reports/plan250/backend/wasi-smoke.json`; `bash scripts/release/v1_0/web-smoke.sh --report reports/plan250/backend/web-ui-smoke.json`; `go run ./tools/cmd/validate-web-ui-smoke --report reports/plan250/backend/web-ui-smoke.json`. |
| T250-151..T250-175 | CLI, Eco, Workspace, LSP, Tooling | `reports/plan250/agent-backend-cli-release-evidence.md`; `reports/plan250/waveA-impl3-cli-tools-summary.md`; `reports/plan250/cli-tools/tetra-test-report.json`; `reports/plan250/cli-tools/smoke-list.json`; `reports/plan250/cli-tools/targets.json`. | `go test ./cli/... -count=1`; `go test ./tools/... -count=1`; `go test ./tools/cmd/validate-lsp-stdio/... ./tools/cmd/validate-lsp-smoke/... -count=1`; `go test ./tools/cmd/validate-diagnostic/... -count=1`; `go run ./tools/cmd/validate-test-report --report reports/plan250/cli-tools/tetra-test-report.json`; `go run ./tools/cmd/validate-smoke-list --report reports/plan250/cli-tools/smoke-list.json`; `go run ./tools/cmd/validate-targets --report reports/plan250/cli-tools/targets.json`. |
| T250-176..T250-200 | Stdlib, Examples, Docs Quality | `reports/plan250/agent-docs-evidence.md`; `reports/plan250/generated-manifest.json`; `reports/plan250/tetra-docs.md`; `reports/plan250/smoke-list-linux-x64.json`; `reports/plan250/waveA-impl1-evidence.md`; docs/user and docs/release files named inline in this range. | `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`; `go test ./tools/cmd/verify-docs ./tools/cmd/validate-example-index ./tools/cmd/validate-manifest ./tools/cmd/validate-api-docs -count=1`; `go run ./tools/cmd/gen-manifest -o reports/plan250/generated-manifest.json`; `go run ./tools/cmd/validate-manifest --manifest reports/plan250/generated-manifest.json`; `./tetra doc examples > reports/plan250/tetra-docs.md`; `go run ./tools/cmd/validate-api-docs --docs reports/plan250/tetra-docs.md`; `go run ./tools/cmd/validate-example-index --docs docs/user/examples_index.md`; `git diff --check`. |
| T250-201..T250-225 | QA, Fuzz, Perf, Security | `reports/plan250/agent-backend-cli-release-evidence.md`; `reports/plan250/waveA-impl3-qa-security-summary.md`; `reports/plan250/waveA-impl4-evidence.md`; `reports/plan250/waveC-impl3-evidence.md`; `reports/plan250/waveC-impl4-evidence.md`; `reports/plan250/qa-index.md`; `reports/plan250/waveF-security/security-review-named.md` (primary security signoff); `reports/plan250/waveC-security/security-review-dry-run.md` (superseded historical dry-run note). | `bash scripts/dev/fuzz-nightly.sh --short --out-dir <path>`; `go test ./tools/scriptstest -count=1`; `go test ./tools/cmd/validate-performance-report/... -count=1`; `go test ./tools/cmd/validate-web-ui-smoke/... -count=1`; `go test ./tools/cmd/validate-artifact-hashes/... -count=1`; `go run ./tools/cmd/validate-release-state --expected-version v0.3.0 --format=text --report-dir <path>`; `bash scripts/release/v1_0/security-review.sh --signoff reports/plan250/waveF-security/security-review-named.md`; `git diff --check`. |
| T250-226..T250-250 | Release Engineering, Final Gate, Handoff | `reports/plan250/final-candidate/test-all-quick/summary.md`; `reports/plan250/waveB-full-rerun/summary.md`; `reports/plan250/waveB-stabilization/summary.md`; `reports/plan250/waveC-v0_3_gate-rerun/summary.md`; `reports/plan250/waveD-full/summary.md`; `reports/plan250/waveD-stabilization/summary.md`; `reports/plan250/waveD-impl1-plan-reconcile.md`; `reports/plan250/waveD-impl2-evidence.md`; `reports/plan250/waveD-impl3-evidence.md`; `reports/plan250/waveD-impl4-evidence.md`; `reports/plan250/final-verification-matrix.md`; `reports/plan250/final-summary.md`; `reports/plan250/final-risk-register.md`; `reports/plan250/final-change-inventory.md`; `reports/plan250/qa-index.md`. | `bash scripts/dev/bootstrap.sh`; `./tetra version`; `./t version`; `go test ./compiler/... ./cli/... ./tools/... -count=1`; `bash scripts/ci/test-all.sh --quick --keep-going --report-dir <path>`; `bash scripts/ci/test-all.sh --full --keep-going --report-dir <path>`; `bash scripts/ci/test-all.sh --stabilization --keep-going --report-dir <path>`; `bash scripts/release/v0_3_0/gate.sh --report-dir <path>`; `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`; `go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`; `git diff --check`. |

## Epic 01: Scope, Contracts, Governance (T250-001..T250-025)

- [x] T250-001 Узгодити `docs/spec/current_supported_surface.md` з фактичним `v0.3.0` surface без суперечностей. Evidence: `reports/plan250/wave2-implD-evidence.md`.
- [x] T250-002 Узгодити `docs/spec/v1_scope.md` з `docs/checklists/v1_0_release_gate.md` по mandatory rows. Evidence: `reports/plan250/wave2-implD-evidence.md`.
- [x] T250-003 Додати явний mapping `feature -> evidence command -> artifact path` для frontend scope. Evidence: `docs/spec/v1_scope.md`, `reports/plan250/wave2-implD-evidence.md`.
- [x] T250-004 Додати явний mapping `feature -> evidence command -> artifact path` для semantics scope. Evidence: `docs/spec/v1_scope.md`, `reports/plan250/wave2-implD-evidence.md`.
- [x] T250-005 Додати явний mapping `feature -> evidence command -> artifact path` для safety scope. Evidence: `docs/spec/v1_scope.md`, `reports/plan250/wave2-implD-evidence.md`.
- [x] T250-006 Додати явний mapping `feature -> evidence command -> artifact path` для runtime scope. Evidence: `docs/spec/v1_scope.md`, `reports/plan250/wave2-implD-evidence.md`.
- [x] T250-007 Додати явний mapping `feature -> evidence command -> artifact path` для backend scope. Evidence: `docs/spec/v1_scope.md`, `reports/plan250/wave2-implD-evidence.md`.
- [x] T250-008 Додати явний mapping `feature -> evidence command -> artifact path` для CLI/tools scope. Evidence: `docs/spec/v1_scope.md`, `reports/plan250/wave2-implD-evidence.md`.
- [x] T250-009 Додати явний mapping `feature -> evidence command -> artifact path` для docs/LSP/Eco scope. Evidence: `docs/spec/v1_scope.md`, `reports/plan250/wave2-implD-evidence.md`.
- [x] T250-010 Нормалізувати статуси у `compiler/features.go` (`current/experimental/planned/post-v1`) проти docs truth. Evidence: `./tetra features --format=json`, `go test ./compiler -run 'TestFeatureRegistry' -count=1`.
- [x] T250-011 Перевірити, що `./tetra features --format=json` відображає ті самі статуси, що й docs. Evidence: `reports/plan250/wave2-implD-evidence.md`.
- [x] T250-012 Додати regression test на consistency feature registry vs docs manifest anchors. Evidence: `go test ./tools/cmd/verify-docs ...`, `go test ./compiler -run 'TestFeatureRegistry' -count=1`.
- [x] T250-013 Оновити `docs/user/status.md` під актуальний стан v0.3.0 без застарілих маркерів. Evidence: `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`.
- [x] T250-014 Оновити `docs/user/v0_3_preview.md` щоб не промотував `planned` як stable. Evidence: `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`.
- [x] T250-015 Звірити `README.md` посилання на активні gate/checklist/handoff документи. Evidence: `go test ./tools/scriptstest -run 'TestReleaseV10Gate|TestCurrentSupportedSurfaceDocumentIsReleaseAligned|TestReleaseV030ChecklistIsNonClaimingAndVersionScoped' -count=1`.
- [x] T250-016 Додати розділ "Scope Drift Policy" у release docs з чітким протоколом зміни scope. Evidence: `docs/release/rc_process.md`.
- [x] T250-017 Додати policy для маркування experimental slices у user docs. Evidence: `docs/release/rc_process.md`.
- [x] T250-018 Додати policy для "no stale evidence reuse" з прикладами анти-патернів. Evidence: `docs/release/rc_process.md`, `docs/release/artifact_policy.md`.
- [x] T250-019 Додати validation крок для перевірки, що `v1` файли не трактуються як proof при `v0.3.0`. Evidence: `docs/checklists/v1_0_release_gate.md`, `scripts/release/v1_0/gate.sh` tests.
- [x] T250-020 Додати test/script assert для version-preflight блокування `scripts/release/v1_0/gate.sh` при `!= v1.0.0`. Evidence: `go test ./tools/scriptstest -run 'TestReleaseV10Gate|TestCurrentSupportedSurfaceDocumentIsReleaseAligned|TestReleaseV030ChecklistIsNonClaimingAndVersionScoped' -count=1`.
- [x] T250-021 Синхронізувати тексти "current public release line" у v1 checklist/release notes template. Evidence: `docs/release-notes/v1_0.md`, `reports/plan250/wave2-implD-evidence.md`.
- [x] T250-022 Оновити `docs/release/v1_0_final_handoff.md` з non-placeholder evidence fields. Evidence: `reports/plan250/wave2-implD-evidence.md`.
- [x] T250-023 Додати lint-check на TODO/TBD/placeholders у v1 handoff/signoff. Evidence: checklist lint row plus `rg` handoff audit in `reports/plan250/wave2-implD-evidence.md`.
- [x] T250-024 Додати перевірку коректності cross-reference між `v1_scope`, checklist, handoff. Evidence: cross-reference audit in `reports/plan250/wave2-implD-evidence.md`.
- [x] T250-025 Згенерувати audit note `reports/plan250/scope-governance-summary.md` з результатами епіку. Evidence: `reports/plan250/scope-governance-summary.md`.

## Epic 02: Frontend (Lexer, Parser, Formatter, Flow) (T250-026..T250-050)

- [x] T250-026 Провести full audit parser coverage проти `docs/spec/flow_syntax_v1.md`.
- [x] T250-027 Додати missing positive parser fixtures для release-covered declaration forms.
- [x] T250-028 Додати missing negative parser fixtures для invalid declaration forms.
- [x] T250-029 Додати parser tests для edge cases indentation recovery.
- [x] T250-030 Додати parser tests для `else if` chain malformed branch diagnostics.
- [x] T250-031 Додати parser tests для function-type syntax boundaries (`fn(...) -> ...`).
- [x] T250-032 Додати parser tests для callable-unsupported forms з стабільними diagnostics.
- [x] T250-033 Додати parser tests для capsule metadata duplicate/shape errors.
- [x] T250-034 Додати lexer tests для mixed LF/CRLF/EOF edge cases.
- [x] T250-035 Додати lexer tests для invalid UTF-8 і deterministic diagnostic positions.
- [x] T250-036 Додати lexer tests для string escape corner cases.
- [x] T250-037 Додати lexer tests для doc-comment tokenization rules.
- [x] T250-038 Додати formatter idempotence tests на representative corpus з `examples/lib/__rt`.
- [x] T250-039 Додати formatter regression tests для comment-preservation у складних блоках.
- [x] T250-040 Додати formatter regression tests для blank-line normalization rules.
- [x] T250-041 Додати formatter diagnostics tests на malformed syntax input.
- [x] T250-042 Підсилити `validate-flow-only` покриття на нові release-covered каталоги.
- [x] T250-043 Додати CI-check для formatter+flow-only pair як обов'язковий fast gate.
- [x] T250-044 Додати docs section "Flow source-of-truth examples" з компілябельними snippets.
- [x] T250-045 Підчистити parser error text для стабільної machine validation.
- [x] T250-046 Забезпечити, що planned features повертають `planned feature ... not implemented` стабільно.
- [x] T250-047 Додати focused frontend test target у scripts (`frontend-focused`). Evidence: `reports/plan250/waveB-impl2-evidence.md`, `reports/plan250/waveC-impl2-plan-reconcile.md`.
- [x] T250-048 Прогнати `go test ./compiler/internal/frontend/... -count=1` і зафіксувати evidence.
- [x] T250-049 Прогнати `./tetra fmt --check examples lib __rt compiler/selfhostrt` і зафіксувати evidence.
- [x] T250-050 Згенерувати audit note `reports/plan250/frontend-summary.md`.

## Epic 03: Semantics (Types, Generics, Protocols, Enums, Modules) (T250-051..T250-075)

- [x] T250-051 Перевірити canonical type display policy (`i32/u8/bool/str/ptr/...`) проти docs/api-docs.
- [x] T250-052 Додати regression tests на type alias normalization у diagnostics.
- [x] T250-053 Додати regression tests на struct field resolution diagnostics.
- [x] T250-054 Додати regression tests на module boundary visibility diagnostics.
- [x] T250-055 Додати regression tests на generic specialization naming determinism.
- [x] T250-056 Додати regression tests на cross-module generic monomorphization.
- [x] T250-057 Додати regression tests на generic inference ambiguous-path diagnostics.
- [x] T250-058 Додати regression tests на protocol requirement signature-shape enforcement.
- [x] T250-059 Додати regression tests на impl conformance mismatch diagnostics.
- [x] T250-060 Додати regression tests на protocol-bound generic static validation paths.
- [x] T250-061 Додати regression tests на explicit non-support for dynamic dispatch through protocols.
- [x] T250-062 Додати regression tests на enum payload arity/type diagnostics.
- [x] T250-063 Додати regression tests на exhaustive unguarded enum match/catch checks.
- [x] T250-064 Додати regression tests на default-order/duplicate-case enum diagnostics.
- [x] T250-065 Додати regression tests на optionals+typed-errors interaction у supported boundary.
- [x] T250-066 Додати regression tests на extension method resolution order.
- [x] T250-067 Додати regression tests на function-type local-to-local binding constraints.
- [x] T250-068 Додати regression tests на callback parameter callable subset boundary.
- [x] T250-069 Уточнити docs для unsupported advanced ADT/pattern features (no accidental promotion).
- [x] T250-070 Додати semantic validation on capsule metadata no-runtime-coupling guarantee.
- [x] T250-071 Прогнати `go test ./compiler/... -run 'Type|Inference|Enum|Optional|Protocol|Extension|Module|Generic|Conformance' -count=1`.
- [x] T250-072 Прогнати `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json` після docs sync.
- [x] T250-073 Прогнати `go run ./tools/cmd/validate-api-docs --docs <generated-docs>` з новим evidence.
- [x] T250-074 Оновити feature descriptions у `compiler/features.go` тільки якщо реально змінився behavior.
- [x] T250-075 Згенерувати audit note `reports/plan250/semantics-summary.md`.

## Epic 04: Safety (Ownership, Lifetimes, Effects, Privacy, Budgets) (T250-076..T250-100)

- [x] T250-076 Додати regression tests на borrow/inout/consume alias rejection edge cases.
- [x] T250-077 Додати regression tests на borrow escape diagnostics across branch merges.
- [x] T250-078 Додати regression tests на use-after-consume across loops/conditionals.
- [x] T250-079 Додати regression tests на resource lifetime for task/group/island handles.
- [x] T250-080 Додати regression tests на region-backed slices lifetime ambiguity diagnostics.
- [x] T250-081 Додати regression tests на struct-contained resource lifetime behavior.
- [x] T250-082 Додати regression tests actor/task transfer checks for unsupported result shapes.
- [x] T250-083 Додати regression tests for use-after-transfer diagnostics stability.
- [x] T250-084 Додати regression tests for sendability checks across module boundaries.
- [x] T250-085 Додати regression tests for `uses effect` propagation through call graphs.
- [x] T250-086 Додати regression tests for `noalloc/noblock/realtime` checker paths.
- [x] T250-087 Додати regression tests for `realtime requires noalloc+noblock` diagnostics.
- [x] T250-088 Додати regression tests for privacy clause static checks.
- [x] T250-089 Додати regression tests for consent-token signature requirements.
- [x] T250-090 Додати regression tests for deterministic local budget guards lowering.
- [x] T250-091 Додати regression tests for unsafe/capability boundary misuse diagnostics.
- [x] T250-092 Синхронізувати `docs/spec/ownership_v1.md` із conservative MVP boundaries.
- [x] T250-093 Синхронізувати `docs/spec/effects_capabilities_privacy_v1.md` з реальною перевіркою.
- [x] T250-094 Переконатися, що lifetime SSA позначено planned і не промотується випадково.
- [x] T250-095 Прогнати `go test ./compiler/... -run 'Ownership|Borrow|Consume|Inout|Lifetime|Resource|Island|Actor|Task|Unsafe|Capability|Effect|Privacy|Consent|Budget|MMIO|Mem' -count=1`.
- [x] T250-096 Прогнати `go test ./tools/cmd/validate-diagnostic/... -count=1`.
- [x] T250-097 Додати/оновити docs приклади для safety diagnostics у user guide.
- [x] T250-098 Додати safety-focused section у `scripts/ci/test-all.sh --stabilization` evidence summary.
- [x] T250-099 Перевірити `git diff --check` після safety wave.
- [x] T250-100 Згенерувати audit note `reports/plan250/safety-summary.md`.

## Epic 05: Runtime (Async, Task, Actors, ABI) (T250-101..T250-125)

- [x] T250-101 Додати regression tests async parse/check/lower path for supported `try await`.
- [x] T250-102 Додати regression tests rejecting unsupported `await try` forms з stable diagnostics.
- [x] T250-103 Додати regression tests task spawn/join/group typing boundaries.
- [x] T250-104 Додати regression tests typed task handle slots 2..8 support boundaries.
- [x] T250-105 Додати regression tests rejecting unsupported task slot counts >8.
- [x] T250-106 Додати bounded stress regression for task runtime determinism.
- [x] T250-107 Додати actor runtime regression for tagged messages happy path.
- [x] T250-108 Додати actor runtime regression for unsupported actor member diagnostics.
- [x] T250-109 Додати actor ownership transfer regression coverage.
- [x] T250-110 Додати selfhost/builtin runtime parity regression tests where applicable.
- [x] T250-111 Додати runtime override regression tests (`--runtime` options and diagnostics).
- [x] T250-112 Додати runtime ABI regression tests for reserved `__tetra_*` symbols.
- [x] T250-113 Додати linker regression tests for repeated link objects and mismatch diagnostics.
- [x] T250-114 Додати TOBJ metadata regression tests for target mismatch handling.
- [x] T250-115 Додати ctx-switch ABI regression tests for sysv/win64 coverage.
- [x] T250-116 Додати runtime diagnostic quality checks for panic/exit boundaries.
- [x] T250-117 Прогнати `go test ./compiler/... -run 'Async|Await|Task|Runtime|Stress|Actor|Actors|Ownership|ABI|Object|Link|SelfHost' -count=1`.
- [x] T250-118 Прогнати actor/task smoke examples under native host.
- [x] T250-119 Синхронізувати runtime ABI docs (`docs/spec/runtime_abi.md`) з реалізацією.
- [x] T250-120 Оновити `docs/user/async_actors_guide.md` по фактичному MVP boundary.
- [x] T250-121 Перевірити, що distributed actors лишаються post-v1 і не промотуються docs.
- [x] T250-122 Додати runtime-focused summary step у release evidence.
- [x] T250-123 Прогнати `./tetra smoke --target linux-x64 --run=true --report <path>`.
- [x] T250-124 Перевірити `git diff --check` після runtime wave.
- [x] T250-125 Згенерувати audit note `reports/plan250/runtime-summary.md`.

## Epic 06: Backends (x64, WASI/Web, UI Metadata) (T250-126..T250-150)

- [x] T250-126 Додати regression tests backend IR instruction coverage for x64 path.
- [x] T250-127 Додати regression tests for unsupported IR diagnostics consistency in backends.
- [x] T250-128 Додати regression tests for wasm32-wasi symbol import validation.
- [x] T250-129 Додати regression tests for wasm32-web symbol import validation.
- [x] T250-130 Додати regression tests for wasm non-zero stack verifier error paths.
- [x] T250-131 Додати regression tests for wasm entry function return-slot constraints.
- [x] T250-132 Додати regression tests for wasm data/global layout deterministic behavior.
- [x] T250-133 Додати regression tests for native shell UI sidecar schema handling.
- [x] T250-134 Додати regression tests for web UI bundle schema validation.
- [x] T250-135 Додати regression tests for UI lowering deterministic JSON output.
- [x] T250-136 Додати regression tests for UI accessibility metadata allowed types.
- [x] T250-137 Додати regression tests for UI style metadata allowed types.
- [x] T250-138 Додати `wasm32-wasi` smoke report schema validation in full/stabilization runs. Evidence: `reports/plan250/waveA-impl2-evidence.md`.
- [x] T250-139 Додати `wasm32-web` smoke report schema validation in full/stabilization runs. Evidence: `reports/plan250/waveA-impl2-evidence.md`.
- [x] T250-140 Прогнати `./tetra smoke --target wasm32-wasi --run=false --report <path>`.
- [x] T250-141 Прогнати `./tetra smoke --target wasm32-web --run=false --report <path>`.
- [x] T250-142 Прогнати `bash scripts/release/v1_0/wasi-smoke.sh --report <path>`.
- [x] T250-143 Прогнати `bash scripts/release/v1_0/web-smoke.sh --report <path>`.
- [x] T250-144 Прогнати `go test ./compiler/... -run 'WASM|Web|Wasi|UI|NativeShell|Lower|IR|Backend' -count=1`.
- [x] T250-145 Оновити `docs/spec/ui_v1.md` тільки за фактом реалізованого metadata surface. Evidence: `reports/plan250/waveA-impl3-evidence.md`, `reports/plan250/waveA-impl3-backend-summary.md`.
- [x] T250-146 Оновити `docs/user/wasm_ui_guide.md` по реальному web/wasi smoke behavior. Evidence: `reports/plan250/waveA-impl3-evidence.md`, `reports/plan250/waveA-impl3-backend-summary.md`.
- [x] T250-147 Переконатися, що native UI runtime widgets лишаються post-v1 у docs. Evidence: `reports/plan250/waveA-impl3-evidence.md`, `reports/plan250/waveA-impl3-backend-summary.md`.
- [x] T250-148 Додати backend-focused summary step до release handoff template. Evidence: `reports/plan250/waveA-impl3-evidence.md`, `reports/plan250/waveA-impl3-backend-summary.md`.
- [x] T250-149 Перевірити `git diff --check` після backend wave. Evidence: `reports/plan250/waveA-impl3-evidence.md`.
- [x] T250-150 Згенерувати audit note `reports/plan250/backend-summary.md`. Evidence: `reports/plan250/waveA-impl3-backend-summary.md`.

## Epic 07: CLI, Eco, Workspace, LSP, Tooling (T250-151..T250-175)

- [x] T250-151 Додати regression tests на CLI `--format` error diagnostics consistency.
- [x] T250-152 Додати regression tests на unsupported target diagnostics consistency.
- [x] T250-153 Додати regression tests на build-only run/test unsupported diagnostics.
- [x] T250-154 Додати regression tests на `features/targets/doctor` JSON schema stability.
- [x] T250-155 Додати regression tests на `check/build/run/test/doc` exit code contracts.
- [x] T250-156 Додати regression tests на `project/workspace` graph/blocked-dependency reporting.
- [x] T250-157 Додати regression tests на workspace schema validation diagnostics.
- [x] T250-158 Додати regression tests на Eco lock schema/manifest/permissions validation.
- [x] T250-159 Додати regression tests на Eco unsupported target diagnostics.
- [x] T250-160 Додати regression tests на Eco publish metadata compatibility checks.
- [x] T250-161 Додати regression tests на Eco vault record hash validation.
- [x] T250-162 Додати regression tests на LSP stdio handshake/report validators.
- [x] T250-163 Додати regression tests на LSP smoke transcript validation.
- [x] T250-164 Прогнати `go test ./cli/... -count=1`.
- [x] T250-165 Прогнати `go test ./tools/... -count=1`.
- [x] T250-166 Прогнати `go test ./tools/cmd/validate-lsp-stdio/... ./tools/cmd/validate-lsp-smoke/... -count=1`.
- [x] T250-167 Прогнати `go test ./tools/cmd/validate-diagnostic/... -count=1`.
- [x] T250-168 Прогнати `go run ./tools/cmd/validate-test-report --report <path>`.
- [x] T250-169 Прогнати `go run ./tools/cmd/validate-smoke-list --report <path>`.
- [x] T250-170 Прогнати `go run ./tools/cmd/validate-targets --report <path>`.
- [x] T250-171 Оновити `docs/spec/cli_contracts.md` по фактичних contract guarantees. Evidence: `reports/plan250/waveA-impl3-evidence.md`, `reports/plan250/waveA-impl3-cli-tools-summary.md`.
- [x] T250-172 Оновити `docs/user/cli_cheatsheet.md` під актуальні команди/формати. Evidence: `reports/plan250/waveA-impl3-evidence.md`, `reports/plan250/waveA-impl3-cli-tools-summary.md`.
- [x] T250-173 Додати tooling summary report aggregation step у `scripts/ci/test-all.sh`. Evidence: `reports/plan250/waveA-impl2-evidence.md`.
- [x] T250-174 Перевірити `git diff --check` після CLI/tools wave. Evidence: `reports/plan250/waveA-impl3-evidence.md`.
- [x] T250-175 Згенерувати audit note `reports/plan250/cli-tools-summary.md`. Evidence: `reports/plan250/waveA-impl3-cli-tools-summary.md`.

## Epic 08: Stdlib, Examples, Docs Quality (T250-176..T250-200)

- [x] T250-176 Аудит `lib/core/*` проти `docs/spec/stdlib.md` та user guide. Evidence: `reports/plan250/docs-stdlib-summary.md`.
- [x] T250-177 Аудит experimental mirrors `lib/experimental/*` і коректність їх labeling. Evidence: `reports/plan250/docs-stdlib-summary.md`.
- [x] T250-178 Додати regression tests на stdlib docs effect metadata requirements. Evidence: `go test ./tools/cmd/verify-docs ...`.
- [x] T250-179 Додати regression tests на stdlib doctest presence for stable modules. Evidence: `go test ./tools/cmd/verify-docs ...`.
- [x] T250-180 Додати regression tests на examples index completeness. Evidence: `go test ./tools/cmd/validate-example-index -count=1`; tool contract blocker recorded separately in T250-192.
- [x] T250-181 Оновити `docs/user/examples_index.md` з поточними smoke/practical examples. Evidence: `reports/plan250/docs-stdlib-summary.md`.
- [x] T250-182 Оновити `docs/user/getting_started.md` на реальний v0.3.0 flow. Evidence: `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`.
- [x] T250-183 Оновити `docs/user/tutorial_path.md` з коротким path `check/build/run/test/doc`. Evidence: `docs/user/tutorial_path.md`.
- [x] T250-184 Оновити `docs/user/language_tour.md` з чіткими boundary notes для planned features. Evidence: `docs/user/language_tour.md`.
- [x] T250-185 Оновити `docs/user/ownership_effects_guide.md` під conservative MVP semantics. Evidence: `docs/user/ownership_effects_guide.md`.
- [x] T250-186 Оновити `docs/user/standard_library_guide.md` для placeholder-vs-production boundaries. Evidence: `docs/user/standard_library_guide.md`.
- [x] T250-187 Оновити `docs/user/troubleshooting.md` з діагностичними прикладами з validator tools. Evidence: `docs/user/troubleshooting.md`.
- [x] T250-188 Оновити `docs/user/eco_package_guide.md` з local-only lifecycle clarifications. Evidence: `docs/user/eco_package_guide.md`.
- [x] T250-189 Перевірити consistency `docs/spec/stdlib_naming_versioning.md` проти реального output docs. Evidence: `reports/plan250/wave2-implD-api-docs.md`, `validate-api-docs`.
- [x] T250-190 Прогнати `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`. Evidence: `reports/plan250/wave2-implD-evidence.md`.
- [x] T250-191 Прогнати `./tetra doc examples > <path>` та `validate-api-docs`. Evidence: `reports/plan250/wave2-implD-api-docs.md`.
- [x] T250-192 Прогнати `go run ./tools/cmd/validate-example-index --docs docs/user/examples_index.md`. Evidence: `reports/plan250/waveA-impl1-evidence.md`.
- [x] T250-193 Прогнати `go run ./tools/cmd/gen-manifest -o <path>` + `validate-manifest`. Evidence: `reports/plan250/wave2-implD-generated-manifest.json`.
- [x] T250-194 Додати docs QA note щодо заборони неактуальних release claims. Evidence: `docs/release/artifact_policy.md`.
- [x] T250-195 Додати docs QA note щодо expected evidence date/version binding. Evidence: `docs/release/artifact_policy.md`.
- [x] T250-196 Перевірити, що docs не містять суперечливих "current release line" тверджень. Evidence: `reports/plan250/wave2-implD-evidence.md`.
- [x] T250-197 Перевірити, що всі user guides посилаються на `current_supported_surface`. Evidence: `rg -n "current_supported_surface" docs/user`.
- [x] T250-198 Перевірити `git diff --check` після docs/stdlib wave. Evidence: `reports/plan250/wave2-implD-evidence.md`.
- [x] T250-199 Оновити `docs/generated/manifest.json` тільки за потреби і з валідацією. Evidence: generated report manifest matched tracked manifest; no update needed.
- [x] T250-200 Згенерувати audit note `reports/plan250/docs-stdlib-summary.md`. Evidence: `reports/plan250/docs-stdlib-summary.md`.

## Epic 09: QA, Fuzz, Perf, Security (T250-201..T250-225)

- [x] T250-201 Додати fuzz regression triage protocol у `docs/testing/fuzz_property_stress.md`.
- [x] T250-202 Розширити fuzz-short сценарій з фіксацією unstable seeds.
- [x] T250-203 Додати regression tests для validator tools на malformed JSON/report inputs.
- [x] T250-204 Додати regression tests для release-state validator on dirty worktree variants. Evidence: `reports/plan250/waveA-impl4-evidence.md`.
- [x] T250-205 Додати regression tests для artifact-hash validator mismatch scenarios. Evidence: `reports/plan250/waveA-impl4-evidence.md`.
- [x] T250-206 Додати regression tests для performance report schema validator.
- [x] T250-207 Додати regression tests для web-ui-smoke validator mismatch fields.
- [x] T250-208 Додати regression tests для wasi-smoke runner/report mismatch fields.
- [x] T250-209 Прогнати `bash scripts/dev/fuzz-nightly.sh --short --out-dir <path>`. Evidence: `reports/plan250/waveA-impl4-evidence.md`, `reports/plan250/qa-index.md`.
- [x] T250-210 Прогнати `go test ./tools/scriptstest -count=1`. Evidence: `reports/plan250/waveC-impl3-evidence.md`.
- [x] T250-211 Прогнати `go test ./tools/cmd/validate-performance-report/... -count=1`.
- [x] T250-212 Прогнати `go test ./tools/cmd/validate-web-ui-smoke/... -count=1`.
- [x] T250-213 Прогнати `go test ./tools/cmd/validate-artifact-hashes/... -count=1`.
- [x] T250-214 Прогнати `go run ./tools/cmd/validate-release-state --expected-version v0.3.0 --format=text --report-dir <path>`. Evidence: `reports/plan250/waveC-v0_3_gate-rerun/artifacts/release-state.txt`, `reports/plan250/waveC-impl4-evidence.md`.
- [x] T250-215 Оновити `docs/performance/v1_0_thresholds.md` за фактичними measurement даними.
- [x] T250-216 Оновити `docs/checklists/security_review_gate.md` з однозначними evidence полями. Evidence: `reports/plan250/waveA-impl3-evidence.md`, `reports/plan250/waveA-impl3-qa-security-summary.md`.
- [x] T250-217 Прогнати `bash scripts/release/v1_0/security-review.sh --signoff <path>` як dry-run validation. Evidence: `reports/plan250/waveF-security/security-review-named.md` (primary security signoff), `reports/plan250/waveC-impl4-evidence.md`, `reports/plan250/waveC-security/security-review-dry-run.md` (superseded historical dry-run note).
- [x] T250-218 Додати summary таблицю "known residual risks" для security/perf/fuzz. Evidence: `reports/plan250/waveA-impl3-evidence.md`, `reports/plan250/waveA-impl3-qa-security-summary.md`.
- [x] T250-219 Додати policy для triage flaky tests and deterministic rerun. Evidence: `reports/plan250/waveA-impl3-evidence.md`, `reports/plan250/waveA-impl3-qa-security-summary.md`.
- [x] T250-220 Додати policy для minimum reproducibility checks до релізного кандидата. Evidence: `reports/plan250/waveA-impl3-evidence.md`, `reports/plan250/waveA-impl3-qa-security-summary.md`.
- [x] T250-221 Перевірити узгодженість timestamps/version у generated reports. Evidence: checked in `reports/plan250/waveA-impl4-evidence.md` and `reports/plan250/qa-index.md`; stale `docs/generated/v1_0` evidence recorded and not reused.
- [x] T250-222 Перевірити `git diff --check` після QA/perf/security wave. Evidence: `reports/plan250/waveA-impl4-evidence.md`, `reports/plan250/qa-index.md`.
- [x] T250-223 Згенерувати consolidated QA index `reports/plan250/qa-index.md`. Evidence: `reports/plan250/qa-index.md`.
- [x] T250-224 Додати автоматичну перевірку на stale report reuse в `scripts/ci/test-all.sh`. Evidence: `reports/plan250/waveA-impl2-evidence.md`.
- [x] T250-225 Згенерувати audit note `reports/plan250/qa-security-summary.md`. Evidence: `reports/plan250/waveA-impl3-qa-security-summary.md`.

## Epic 10: Release Engineering, Final Gate, Handoff (T250-226..T250-250)

- [x] T250-226 Зібрати clean release report dir для поточного branch-state (`reports/plan250/final-candidate`).
- [x] T250-227 Прогнати `bash scripts/dev/bootstrap.sh` і зафіксувати версійні evidence. Evidence: `reports/plan250/waveA-impl4-evidence.md`, `reports/plan250/qa-index.md`.
- [x] T250-228 Прогнати `./tetra version` і `./t version` та зафіксувати parity evidence.
- [x] T250-229 Прогнати `go test ./compiler/... ./cli/... ./tools/... -count=1`. Evidence: `reports/plan250/waveC-v0_3_gate-rerun/logs/03-go-test-packages.log`, `reports/plan250/waveC-impl4-evidence.md`.
- [x] T250-230 Прогнати `bash scripts/ci/test-all.sh --quick --keep-going --report-dir <path>`. Evidence: `reports/plan250/final-candidate/test-all-quick/summary.md`, `reports/plan250/waveC-impl4-evidence.md`.
- [x] T250-231 Прогнати `bash scripts/ci/test-all.sh --full --keep-going --report-dir <path>`. Evidence: `reports/plan250/waveB-impl4-evidence.md`, `reports/plan250/waveB-full-rerun/summary.md`, `reports/plan250/waveC-impl2-plan-reconcile.md`.
- [x] T250-232 Прогнати `bash scripts/ci/test-all.sh --stabilization --keep-going --report-dir <path>`. Evidence: `reports/plan250/waveB-impl4-evidence.md`, `reports/plan250/waveB-stabilization/summary.md`, `reports/plan250/waveC-impl2-plan-reconcile.md`.
- [x] T250-233 Прогнати `bash scripts/release/v0_3_0/gate.sh --report-dir <path>`. Evidence: `reports/plan250/waveC-v0_3_gate-rerun/summary.md`, `reports/plan250/waveC-impl4-evidence.md`.
- [x] T250-234 Зафіксувати і усунути blockers `validate-release-state` (dirty tracked/untracked/stale evidence). Evidence: `reports/plan250/waveC-v0_3_gate-rerun/artifacts/release-state.txt`, `reports/plan250/waveC-impl3-evidence.md`, `reports/plan250/waveC-impl4-evidence.md`.
- [x] T250-235 Перегенерувати artifact-hash manifest та прогнати його валідацію. Evidence: `reports/plan250/waveB-impl4-evidence.md`, `reports/plan250/waveB-v0_3_gate/artifacts/artifact-hashes.json`.
- [x] T250-236 Перевірити availability і валідність required artifact set для handoff. Evidence: `reports/plan250/waveB-impl4-evidence.md`, `reports/plan250/waveB-v0_3_gate/artifacts/release-state.txt`, `reports/plan250/waveC-impl2-plan-reconcile.md`.
- [x] T250-237 Оновити `docs/release/v0_3_0_final_handoff.md` з новим report-dir evidence. Evidence: `reports/plan250/waveB-impl4-evidence.md`.
- [x] T250-238 Оновити `docs/release-notes/v0_3_0.md` з фактичним переліком підтверджених slices. Evidence: `reports/plan250/waveB-impl4-evidence.md`.
- [x] T250-239 Підготувати `docs/release/known_issues_template.md` instance для поточного кандидата. Evidence: `reports/plan250/waveB-impl4-evidence.md`, `docs/release/known_issues_v0_3_0_waveB_candidate.md`.
- [x] T250-240 Звірити відповідність final handoff з `docs/checklists/v0_3_0_release_gate.md`. Evidence: `reports/plan250/waveB-impl4-evidence.md`.
- [x] T250-241 Звірити відповідність final handoff з `docs/spec/current_supported_surface.md`. Evidence: `reports/plan250/waveB-impl4-evidence.md`.
- [x] T250-242 Звірити відповідність final handoff з `compiler/features.go` runtime truth. Evidence: `reports/plan250/waveB-impl4-evidence.md`.
- [x] T250-243 Додати `reports/plan250/final-verification-matrix.md` (task -> evidence path). Evidence: `reports/plan250/final-verification-matrix.md`.
- [x] T250-244 Додати `reports/plan250/final-risk-register.md` (open risks + mitigation + owner). Evidence: `reports/plan250/final-risk-register.md`.
- [x] T250-245 Додати `reports/plan250/final-change-inventory.md` (files changed by epic). Evidence: `reports/plan250/final-change-inventory.md`.
- [x] T250-246 Прогнати `git diff --check` фінально. Evidence: `reports/plan250/waveB-impl4-evidence.md`, `reports/plan250/waveC-impl2-plan-reconcile.md`.
- [x] T250-247 Прогнати `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json` фінально. Evidence: `reports/plan250/waveB-impl4-evidence.md`, `reports/plan250/waveC-impl2-plan-reconcile.md`.
- [x] T250-248 Прогнати `go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json` фінально. Evidence: `reports/plan250/waveB-impl4-evidence.md`, `reports/plan250/waveC-impl2-plan-reconcile.md`.
- [x] T250-249 Підготувати фінальний execution summary `reports/plan250/final-summary.md`. Evidence: `reports/plan250/final-summary.md`.
- [x] T250-250 Закрити план: усі 250 задач мають evidence або documented/blocker status з власником і next step. Evidence: `reports/plan250/waveC-impl4-evidence.md`, `reports/plan250/waveD-impl1-plan-reconcile.md`.
