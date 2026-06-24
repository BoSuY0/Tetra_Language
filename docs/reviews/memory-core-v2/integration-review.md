# Memory Core v2 Integration Review

reviewed_commit: 8f7529505a13b5da72fbc0c34c5bb110541c020f

reviewer_agent: SUBAGENT-D / gpt-5.5 xhigh / worker

reviewed_paths:
- scripts/release/memory/memory-core-v2-gate.sh
- scripts/release/memory/README.md
- tools/cmd/validate-memory-core-v2/main.go
- tools/validators/memorycorev2/report.go
- tools/validators/memorycorev2/report_test.go
- tools/validators/memorycorev2/testdata/*
- docs/spec/memory/memory_core_v2.md
- docs/release/memory/memory-core-v2-release-boundary.md
- docs/audits/memory/README.md
- docs/audits/memory/production/memory-production-core-v1-supported-surface.md
- compiler/internal/memoryfacts
- compiler/internal/memorypipeline
- compiler/internal/runtimeabi
- compiler/internal/islandkernel
- compiler/internal/opt
- compiler/compiler_suite_test.go
- compiler/compiler_external_test.go
- .workflow/memory-core-v2-stabilization/orchestration.md

commands_executed:
- `git rev-parse HEAD` in assigned worktree: `8f7529505a13b5da72fbc0c34c5bb110541c020f`.
- `git branch --show-current`: `stabilize/memory-core-v2`.
- `git status --porcelain=v1 --untracked-files=all`: initially clean; later showed untracked peer review files `compiler-soundness-review.md`, `optimizer-proof-review.md`, and `runtime-domain-review.md` before this file was written.
- `rg -n "negative|malformed|mutation|dirty_worktree|memory-core-v2-gate|claim|backward|determin|report-only|buildvcs" scripts/release/memory tools/validators/memorycorev2 docs/spec/memory docs/release/memory compiler`: located the gate, validator guards, docs claims, dirty-worktree field, report-only guard, deterministic tests, and compatibility tests.
- `go test -buildvcs=false ./compiler/... ./cli/... ./tools/... ./... -count=1`: first combined run failed in `tools/scriptstest/workspace` with transient `$WORK/.../_pkg_.a` import archive errors; isolated rerun of `tools/scriptstest/workspace` passed.
- `go test -buildvcs=false ./compiler/... -count=1`: pass in assigned worktree.
- `go test -buildvcs=false ./cli/... ./tools/... -count=1`: first rerun failed once in `tools/scriptstest/test_all`; isolated package and final full rerun passed.
- `go test -buildvcs=false ./... -count=1`: pass in assigned worktree.
- `go test -race -buildvcs=false ./compiler/internal/memoryfacts/... -run 'Snapshot|Delta|Proof|Digest' -count=1`: pass.
- `go test -race -buildvcs=false ./compiler/internal/runtimeabi/... -run 'MemoryDomain|Ledger' -count=1`: pass.
- `go run ./tools/cmd/validate-memory-core-v2 --report tools/validators/memorycorev2/testdata/positive.json --current-git-head 0123456789abcdef0123456789abcdef01234567`: pass.
- `go run ./tools/cmd/validate-memory-core-v2` negative assertions via process substitution: `negative_broad_claim`, `claims_negative`, `malformed_trailing`, `dirty_worktree_true`, `report_only_state`, and `proof_mutation` all failed as expected with exit code 1 and targeted validator messages.
- `go test -buildvcs=false ./compiler ./compiler/internal/memorypipeline ./compiler/internal/memoryfacts ./tools/validators/memorycorev2 -run 'ReportFlagsDoNotChangeBorrowedReturnFailure|BuildReportsMemoryFactIDsAndSiteIDsStableAcrossRuns|P7JSONReportHashesStableAcrossWorkerCounts|BuildCreatesPlannedStateWithoutReportOptions|BuildReportFromGraphDeterministicProjection|MemoryReportJSONMutationDoesNotMutateGraphOrSnapshot|LegacyFinalSignoffAlias' -count=1`: pass.
- `go run ./tools/cmd/validate-memory-core-v2 --claim-path docs/spec/memory/memory_core_v2.md --claim-path docs/audits/memory/README.md --claim-path docs/audits/memory/production/memory-production-core-v1-supported-surface.md --claim-path docs/release/memory/memory-core-v2-release-boundary.md`: pass.
- Clean clone: `git clone --no-local /home/tetra/Desktop/Projects/Tetra_Language /home/tetra/.codex/review-clones/mcv2-review-d`; `git checkout 8f7529505a13b5da72fbc0c34c5bb110541c020f`; `git status --porcelain=v1 --untracked-files=all`: clean.
- Clean clone: `go test -buildvcs=false ./compiler/... -count=1`: pass.
- Clean clone: `go test -buildvcs=false ./cli/... ./tools/... -count=1`: failed once with local-origin repo inference and once with transient `tools/scriptstest/test_all`; after setting the disposable clone origin to `https://github.com/BoSuY0/Tetra_Language.git`, focused failing tests passed and final full rerun passed.
- Clean clone: `go test -buildvcs=false ./... -count=1`: pass.
- Clean clone gate run A: `bash scripts/release/memory/memory-core-v2-gate.sh --report-dir .cache/review-d-gate-a`: pass.
- Clean clone gate run B: `bash scripts/release/memory/memory-core-v2-gate.sh --report-dir .cache/review-d-gate-b`: pass.
- Clean clone deterministic comparison: `cmp -s .cache/review-d-gate-a/memory-core-v2-evidence.json .cache/review-d-gate-b/memory-core-v2-evidence.json`: pass.
- Clean clone evidence digest: both gate reports had SHA-256 `4560842acc54c931cf9cc0046085a072fe1fca33f50ea3d89959806fc5744905`.
- Clean clone evidence fields: `git_head=8f7529505a13b5da72fbc0c34c5bb110541c020f`, `dirty_worktree=false`, `implementation_complete=true`, `memory_core_gate=pass`, `release_security_review_status=pending_final_rc`.
- Gate non-empty report-dir check: rerunning the gate against `.cache/review-d-gate-a` exited 2 with `refusing to reuse non-empty report directory`.

findings:
- id: D-001
  severity: low
  summary: Broad workspace/scriptstest runs showed non-deterministic environment coupling outside the Memory Core v2 gate path.
  reproduction: The initial combined `go test -buildvcs=false ./compiler/... ./cli/... ./tools/... ./... -count=1` failed once in `tools/scriptstest/workspace` with missing `$WORK/.../_pkg_.a` archives. A later broad `go test ./cli/... ./tools/...` failed once in `tools/scriptstest/test_all` on the unsafe-promotion quick gate. In the clean clone, local-origin `origin` caused `tools/scriptstest/workflows/TestReleaseFullPlatformTargetHostEvidenceRequestBundle` to fail until the disposable clone remote was changed to the GitHub URL, and one broad run failed in `tools/scriptstest/test_all` on the memory-fuzz-oracle fake gate. Focused reruns of the failing packages/tests passed, and final broad reruns passed.
  required_fix: Nonblocking for Memory Core v2 stabilization. Follow up by hardening `tools/scriptstest` fake-repo and nested-Go-test isolation: avoid dependence on GitHub-shaped `origin` for local clean clones or document it, and make fake `test-all` gates independent of broad package execution order and shared temporary/cache state.

severity:
- D-001: low

reproduction:
- D-001: See the failed broad commands and passing focused/final reruns listed under `commands_executed`.

required_fix:
- D-001: Track as test-infrastructure hardening; no Memory Core v2 code, gate, schema, or documentation change is required by this review.

unresolved_risks:
- The assigned worktree was no longer clean by the time gate validation started because peer review files were present as untracked docs. I did not modify those files. Canonical clean-gate and deterministic-build evidence therefore came from a disposable clean clone at the reviewed commit.
- Human security review remains out of scope and correctly stays `release_security_review_status=pending_final_rc`; this review does not approve a release, a `v0.5.0` tag, or any final security signoff.
- Clean clone reproducibility assumes a GitHub-shaped remote URL for workflow script tests that infer `BoSuY0/Tetra_Language`; a purely local-origin clone exposed this assumption.

verdict: PASS_WITH_NONBLOCKING_FINDINGS
