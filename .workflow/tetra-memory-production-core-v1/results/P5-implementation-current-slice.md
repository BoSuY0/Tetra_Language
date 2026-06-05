# P5 Implementation Current Slice Result

Status: integrated.

Implemented:

- `compiler/internal/memoryfacts` graph, fact enums, validation, report
  projection, and PLIR/allocation-plan adapter.
- `tools/cmd/validate-memory-report` schema-v1 validator.
- `BuildOptions.EmitMemoryReport` plus `tetra build --emit-memory-report`.
- `.memory.json` emission through `emitExplainReports`.
- Memory Production Core v1 docs, schema spec, manifest linkage, and unsafe
  boundary note.

TDD evidence:

- RED:
  `GOCACHE=$(pwd)/.cache/go-build-memoryfacts-red go test ./compiler/internal/memoryfacts -count=1`
  failed on missing graph/report types before implementation.
- RED:
  `GOCACHE=$(pwd)/.cache/go-build-memory-report-red go test ./tools/cmd/validate-memory-report -count=1`
  failed on missing validator before implementation.
- RED:
  `GOCACHE=$(pwd)/.cache/go-build-memory-integration-red go test ./compiler -run TestBuildCommandEmitMemoryReportWritesSchemaV1 -count=1`
  failed on missing `BuildOptions.EmitMemoryReport` before integration.
- RED:
  `GOCACHE=$(pwd)/.cache/go-build-memoryfacts-red-conservative go test ./compiler/internal/memoryfacts -run TestMemoryFactsKeepsRawSliceUnknownConservative -count=1`
  failed because unknown raw slices projected as `evidence_only`.

Green evidence:

- `GOCACHE=$(pwd)/.cache/go-build-memoryfacts-final-focus go test ./compiler/internal/memoryfacts -count=1` passed.
- `GOCACHE=$(pwd)/.cache/go-build-memory-report-final-focus go test ./tools/cmd/validate-memory-report -count=1` passed.
- `GOCACHE=$(pwd)/.cache/go-build-memory-compiler-focus go test ./compiler -run 'Memory|Raw|Unsafe|Bounds|Report' -count=1` passed.
- `GOCACHE=$(pwd)/.cache/go-build-memory-cli-focus go test ./cli/cmd/tetra -run 'Build|Report|Help' -count=1` passed.
- `GOCACHE=$(pwd)/.cache/go-build-memory-internal-focus go test ./compiler/internal/plir ./compiler/internal/allocplan ./compiler/internal/validation -count=1` passed.

## MPC-9 Addendum: Raw Slice Gateway Hardening

Status: integrated and verified.

Implemented:

- `core.raw_slice_*_from_parts` remains unsafe-only with `mem` effect and explicit
  `cap.mem` checks in semantics.
- Runtime ABI distinguishes `verified_allocation_root`, `external_unknown`,
  `rejected_negative_length`, and `rejected_length_overflow`.
- PLIR/memoryfacts project verified raw-slice evidence as `unsafe_checked`
  evidence-only rows and keep unknown raw slices conservative.
- linux-x64 codegen traps before view construction for negative raw-slice length
  and i32 target byte-overflow; other targets have scoped build/codegen evidence.
- Memory production smoke and validators require raw-slice negative/overflow cases
  and raw-slice memory report evidence.

Review integration:

- Code review blockers fixed: stricter x64 signed length guard, verifier contract
  for `IRRawSliceFromParts.Imm`, raw-slice rejected report rows, and production
  report fixture updates.
- Spec review blockers fixed: non-x64 scoped backend tests for linux-x86,
  linux-x32, wasm32_wasi, and wasm32_web; `docs/spec/runtime_abi.md` now lists
  raw-slice builtins and documents linux-x64-only runtime trap scope.

Green evidence:

- `GOCACHE=$(pwd)/.cache/go-build-mpc9-final3-runtimeabi go test ./compiler/internal/runtimeabi -run 'RawSlice|RawPointer|Bounds|Overflow' -count=1` passed.
- `GOCACHE=$(pwd)/.cache/go-build-mpc9-final3-plir go test ./compiler/internal/plir -run 'RawSlice|Unsafe|Bounds|AllocBytes|Pointer' -count=1` passed.
- `GOCACHE=$(pwd)/.cache/go-build-mpc9-final3-memoryfacts go test ./compiler/internal/memoryfacts -run 'RawSlice|Unsafe|Bounds|Report|NoAlias|Index' -count=1` passed.
- `GOCACHE=$(pwd)/.cache/go-build-mpc9-final3-validate-memory-report go test ./tools/cmd/validate-memory-report -run 'RawSlice|Unsafe|Bounds|Report' -count=1` passed.
- `GOCACHE=$(pwd)/.cache/go-build-mpc9-final3-safety go test ./compiler/tests/safety -run 'RawSlice|Unsafe|Capability|Mem' -count=1` passed.
- `GOCACHE=$(pwd)/.cache/go-build-mpc9-final3-semantics go test ./compiler/tests/semantics -run 'RawSlice|Unsafe|Capability|Mem' -count=1` passed.
- `GOCACHE=$(pwd)/.cache/go-build-mpc9-final3-compiler go test ./compiler -run 'Memory|Raw|Unsafe|Bounds|Report' -count=1` passed.
- `GOCACHE=$(pwd)/.cache/go-build-mpc9-final3-memory-smoke-unit go test ./tools/cmd/memory-production-smoke -run 'Raw|Bounds|Memory' -count=1` passed.
- `GOCACHE=$(pwd)/.cache/go-build-mpc9-final3-memory-smoke bash scripts/release/post_v0_4/memory-production-linux-x64-smoke.sh --report-dir reports/memory-production-core-v1/mpc9` passed and wrote `reports/memory-production-core-v1/mpc9/memory-production-linux-x64.json` plus `artifact-hashes.json`.
- `GOCACHE=$(pwd)/.cache/go-build-mpc9-final3-broad go test ./compiler/... ./cli/... ./tools/... -count=1` passed.
- `GOCACHE=$(pwd)/.cache/go-build-mpc9-final3-ci bash scripts/ci/test.sh` passed with `OK` and `Artifact: tetra.release.v0_4_0.go-test-suite.v1`.
- Docs/manifest gates and pre/post-graph `git diff --check` passed.
- `graphify update .` rebuilt `graphify-out` with 21000 nodes and 65848 edges.
