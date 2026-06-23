reviewed_commit: 8f7529505a13b5da72fbc0c34c5bb110541c020f

reviewer_agent: SUBAGENT-B / gpt-5.5 xhigh / worker

reviewed_paths:
- AGENTS.md
- docs/spec/memory/memory_core_v2.md
- docs/spec/memory/memory_domains_vnext.md
- docs/spec/memory/memory_backend_vnext.md
- docs/spec/memory/islands.md
- docs/release/memory/memory-core-v2-release-boundary.md
- scripts/release/memory/memory-core-v2-gate.sh
- compiler/internal/runtimeabi/memory_domain.go
- compiler/internal/runtimeabi/memory_domain_ledger.go
- compiler/internal/runtimeabi/memory_domain_ledger_test.go
- compiler/internal/runtimeabi/memory_domain_ledger_t12_test.go
- compiler/internal/runtimeabi/memory_backend.go
- compiler/internal/runtimeabi/memory_backend_test.go
- compiler/internal/runtimeabi/runtimeabi_test/memory_backend_test.go
- compiler/internal/runtimeabi/region_allocator.go
- compiler/internal/runtimeabi/runtimeabi_test/region_allocator_test.go
- compiler/internal/islandkernel/kernel.go
- compiler/internal/islandkernel/kernel_test.go
- compiler/internal/islandkernel/coverage.go
- compiler/internal/backend/linux_x64/codegen.go
- compiler/internal/backend/linux_x64/codegen_test.go
- compiler/internal/backend/x64abi/sysv_unix.go
- compiler/internal/backend/x64abi/abi_test.go
- compiler/internal/backend/wasm32_wasi/codegen_helpers.go
- tools/validators/memorycorev2/report.go
- tools/validators/memorycorev2/report_test.go
- tools/validators/memorycorev2/testdata/positive.json

commands_executed:
- `git rev-parse HEAD`
  - result: `8f7529505a13b5da72fbc0c34c5bb110541c020f`; matched required reviewed base commit.
- `git branch --show-current`
  - result: `stabilize/memory-core-v2`.
- `git status --porcelain=v1 --untracked-files=all`
  - result: clean before review work.
- `ls -la graphify-out`
  - result: failed with `No such file or directory`; no Graphify artifacts available in this worktree.
- `rg -n "Domain|domain|island|epoch|stale|release|requested|reserved|committed|current|peak|budget|ownership|linux_x64|unsupported" compiler docs/spec/memory scripts/release/memory tools`
  - result: broad surface discovery found runtime ABI, islandkernel, linux_x64 backend, memory specs, release gate, and validators.
- `rg --files compiler docs/spec/memory scripts/release/memory tools`
  - result: broad file inventory for review scope.
- `find .. -name AGENTS.md -print`
  - result: found this worktree's `AGENTS.md`; command also hit a permission-denied path in a sibling worktree, not relevant to this review.
- `sed -n '1,220p' AGENTS.md`
  - result: confirmed local repo instructions.
- `rg -n "type .*Domain|Domain|domain|epoch|stale|release|requested|reserved|committed|current|peak|budget|ownership|Transfer|Close|Allocate|Backend|linux_x64|unsupported" compiler/internal/runtimeabi compiler/internal/islandkernel compiler/internal/backend/linux_x64 docs/spec/memory scripts/release/memory`
  - result: focused runtime/domain/island/backend hit list.
- `nl -ba` inspections of the reviewed paths listed above
  - result: inspected implementation, tests, specs, release gate, and validator line-level evidence.
- `mkdir -p .cache/go-build-review-b .cache/go-tmp-review-b`
  - result: created repo-local Go cache/temp dirs for evidence runs.
- `env GOCACHE="$PWD/.cache/go-build-review-b" GOTMPDIR="$PWD/.cache/go-tmp-review-b" GOTELEMETRY=off go test -buildvcs=false ./compiler/internal/runtimeabi/... ./compiler/internal/islandkernel ./compiler/internal/backend/linux_x64 -run 'Memory|Backend|Domain|Ledger|Island|Route' -count=1`
  - result: PASS for `compiler/internal/runtimeabi`, `runtimeabi/runtimeabi_test`, `islandkernel`, and `backend/linux_x64`; `runtimeabi/smallheap` had no test files.
- `env GOCACHE="$PWD/.cache/go-build-review-b" GOTMPDIR="$PWD/.cache/go-tmp-review-b" GOTELEMETRY=off go test -buildvcs=false ./tools/validators/memorycorev2 ./tools/cmd/validate-memory-core-v2 -run 'Memory|Claim|Validate' -count=1`
  - result: PASS for `tools/validators/memorycorev2`; command package had no test files.
- Initial parallel exact `-v` batch for backend/runtimeabi/islandkernel/validator
  - result: failed before tests with `go: creating work dir: stat .../.cache/go-tmp-review-b: no such file or directory`; recreated GOTMPDIR and reran exact slices sequentially.
- `env GOCACHE="$PWD/.cache/go-build-review-b" GOTMPDIR="$PWD/.cache/go-tmp-review-b" GOTELEMETRY=off go test -buildvcs=false -v ./compiler/internal/backend/linux_x64 -run TestCodegenObjectLinuxX64MemoryBackendRuntimeSmoke -count=1`
  - result: PASS; `TestCodegenObjectLinuxX64MemoryBackendRuntimeSmoke` ran, not skipped.
- `env GOCACHE="$PWD/.cache/go-build-review-b" GOTMPDIR="$PWD/.cache/go-tmp-review-b" GOTELEMETRY=off go test -buildvcs=false -v ./compiler/internal/runtimeabi -run 'TestMemoryDomainLedger|TestT12|TestRuntimeMemoryBackend|TestMemoryBackendRuntime|TestMeasureMemoryFootprint' -count=1`
  - result: PASS; covered ledger accounting, lifecycle move/copy/close, invalid events, concurrent snapshot, task close cleanup, actor copy accounting, backend support rows, unsupported WASM release, and footprint evidence classes.
- `env GOCACHE="$PWD/.cache/go-build-review-b" GOTMPDIR="$PWD/.cache/go-tmp-review-b" GOTELEMETRY=off go test -buildvcs=false -v ./compiler/internal/islandkernel -run 'TestIslandKernelRequiredDecisionQuestions|TestIslandKernelDecisionMetadata|TestCanPlanExplicitIslandRejectsMissingIdentityOwnerAndUnsafe|TestIslandKernelRejectsExternalUnsafeTrustedStoragePromotion|TestIslandKernelDangerousDecisionRouteCoverage' -count=1`
  - result: PASS; covered stale epoch rejection, live-borrow boundary rejection, free/reset live-borrow rejection, owner/epoch/proof checks, unsafe/external storage rejection, and 16 direct dangerous-decision routes.
- `env GOCACHE="$PWD/.cache/go-build-review-b" GOTMPDIR="$PWD/.cache/go-tmp-review-b" GOTELEMETRY=off go test -buildvcs=false -v ./tools/validators/memorycorev2 -run TestMemoryCoreV2ValidateReportRejectsRequiredGuards -count=1`
  - result: PASS; covered missing digest, report-only state, route-count mismatch, proofless optimizer rewrite, unsupported backend marked supported, memorymodel parity incomplete, and failed requirement with implementation complete.
- `env GOCACHE="$PWD/.cache/go-build-review-b" GOTMPDIR="$PWD/.cache/go-tmp-review-b" GOTELEMETRY=off go test -buildvcs=false -v ./compiler/internal/backend/x64abi -run 'TestSysVReleaseAllocationMunmapsAllocBytesHeaderMapping|TestEmitIslandFreeDebugEmitsDoubleFreeGuard|TestEmitIslandResetDebugChecksFreedMarkerAndReturnsHandle' -count=1`
  - result: PASS; covered linux SysV release `munmap`, island debug double-free guard, protected/decommit free path, and reset freed-marker guard.
- `env GOCACHE="$PWD/.cache/go-build-review-b" GOTMPDIR="$PWD/.cache/go-tmp-review-b" GOTELEMETRY=off go clean -cache`
  - result: PASS; cleaned repo-local Go cache.
- `rm -rf .cache/go-build-review-b .cache/go-tmp-review-b`
  - result: removed review scratch dirs.
- `rg -n "^(reviewed_commit|reviewer_agent|reviewed_paths|commands_executed|findings|severity|reproduction|required_fix|unresolved_risks|verdict):" docs/reviews/memory-core-v2/runtime-domain-review.md`
  - result: PASS; all required grep fields/sections present.
- `git status --porcelain=v1 --untracked-files=all`
  - result: showed `?? docs/reviews/memory-core-v2/runtime-domain-review.md` plus unrelated `?? docs/reviews/memory-core-v2/compiler-soundness-review.md` and `?? docs/reviews/memory-core-v2/optimizer-proof-review.md`; only `runtime-domain-review.md` was created by this review.
- `test ! -e /home/tetra/Desktop/Projects/Tetra_Language/docs/reviews/memory-core-v2/runtime-domain-review.md`
  - result: PASS; corrected an initial local patch placement mistake and confirmed no stray copy remains in the original cwd.

findings:

B-001: Release evidence treats `wasm32-wasi reserve` as unsupported while runtime ABI treats WASM reserve/commit as supported.

severity: medium

reproduction:
- Inspect `compiler/internal/runtimeabi/memory_backend.go:508-517`: `MemoryBackendSupportMatrix("wasm32-wasi")` marks `reserve` and `commit` supported via `wasm_memory_grow_combined_reserve_commit`, while `decommit`, `release`, `trim`, and `footprint` are unsupported.
- Inspect `compiler/internal/runtimeabi/memory_backend_test.go:49-55`: tests assert WASM `reserve` and `commit` support rows are supported and mention `memory_grow`.
- Inspect `scripts/release/memory/memory-core-v2-gate.sh:176-180` and `tools/validators/memorycorev2/testdata/positive.json:49-52`: Memory Core v2 release evidence marks `wasm32-wasi` `reserve` as `supported: false` with `runtime memory backend operation is not implemented for wasm32-wasi`.
- Inspect `tools/validators/memorycorev2/report.go:397-406` and `tools/validators/memorycorev2/report_test.go:65-73`: the release validator accepts supported backend operations only for `linux-x64`, and the negative test flips the `wasm32-wasi reserve` row to `supported: true` expecting rejection.
- Verification evidence: the runtime ABI exact slice passed `TestRuntimeMemoryBackendContractUsesOperationSupportRows`; the validator exact slice passed `TestMemoryCoreV2ValidateReportRejectsRequiredGuards`.

required_fix:
- Align release evidence and validator policy with the runtime ABI contract.
- If Memory Core v2 release evidence intentionally wants all WASM runtime backend operations unsupported, change `MemoryBackendSupportMatrix` and its tests so `wasm32-wasi`/`wasm32-web` `reserve` and `commit` are unsupported too.
- If the runtime ABI contract is correct, update `scripts/release/memory/memory-core-v2-gate.sh`, `tools/validators/memorycorev2`, and fixtures so WASM `reserve`/`commit` can be represented as supported with evidence, and use actually unsupported WASM operations such as `decommit`, `release`, `trim`, or `footprint` for the unsupported-target row.
- In either path, update the Memory Core v2 backend matrix wording so it distinguishes unsupported footprint/release behavior from partial operation support.

unresolved_risks:
- Human security review remains out of scope and pending for the final v0.5.0 RC as `release_security_review_status=pending_final_rc`.
- This review did not run the full `scripts/release/memory/memory-core-v2-gate.sh`; it ran focused runtime/domain/island/backend/validator slices.
- The B-001 mismatch is nonblocking for linux-x64 runtime/domain lifecycle but should be fixed before relying on Memory Core v2 evidence for precise unsupported-target semantics.

verdict: PASS_WITH_NONBLOCKING_FINDINGS
