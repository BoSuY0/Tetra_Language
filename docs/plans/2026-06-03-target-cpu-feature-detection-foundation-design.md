# P17.4 Target CPU Feature Detection Foundation Design

Status: Approved by active `GOAL.md` P17.4 Bridge. This document narrows the
target-cpu batch to evidence and contracts only.

## Goal

Promote the P17.4 `target_cpu_feature_detection` row from `not_yet_covered` to
`implemented_narrow` by adding a deterministic internal target-feature model,
portable baseline fallback, guarded codegen contract, and negative
safe-semantics tests.

## Observed Facts

- `compiler/internal/opt/pgo_lto.go` owns the
  `tetra.optimizer.pgo_lto_target_cpu.v1` coverage rows.
- `compiler/internal/backend/x64.CodegenOptions` is the internal native x64
  backend option carrier.
- Public `compiler.BuildOptions` has no PGO/profile/LTO/target-cpu semantic
  tuning fields and is guarded by `TestBuildOptionsExposeNoBackendSemanticMode`.
- Existing vectorized linux-x64 paths already use baseline SSE2-style opcodes,
  but P17.4 must not claim performance or target-specific tuning from that.

## Design

Use an explicit internal target-feature input model instead of host CPU probing.
The zero value resolves to a portable baseline. For x64/x32 targets this
baseline records `sse2`; for non-x64 register-width targets it records no SIMD
features and keeps machine-specific feature use disabled.

The feature model is reportable evidence only:

- it validates and canonicalizes target feature names;
- it rejects unknown features and features below the portable baseline;
- it records source as `portable_baseline` or `explicit`;
- it exposes whether a backend may use a named feature;
- it does not flow from public `BuildOptions`;
- it does not change source, IR, validation, or safe-program semantics.

## Implementation Plan

1. Add RED tests in `compiler/internal/backend/x64` for target-feature
   resolution, portable fallback, explicit feature validation, and baseline
   rejection.
2. Add RED tests in `compiler/internal/opt` that the P17.4 target-cpu row is
   `implemented_narrow` only after target-feature model evidence, portable
   baseline fallback, guarded codegen contract, and negative safe-semantics
   evidence exist.
3. Add RED tests in `compiler` that public `BuildOptions` still has no
   target-cpu/profile/LTO semantic field and that native codegen options resolve
   only to the portable baseline by default.
4. Implement the smallest internal x64 target-feature model needed by the
   tests. Keep existing codegen behavior unchanged.
5. Update the P17.4 coverage row, feature registry text, audit/progress docs,
   generated manifest, and goal sidecars with bounded non-claims.

## Verification

Focused RED/GREEN:

```bash
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/backend/x64 ./compiler/internal/opt ./compiler ./compiler/tests/semantics -run 'TestTargetFeatureModelUsesPortableBaselineAndRejectsUnsafeDrift|TestCodegenOptionsTargetFeatureGuardIsEvidenceOnly|TestPGOLTOTargetCPUCoverageAuditsP17PlanList|TestNativeCodegenOptionsUsePortableTargetFeatureBaseline|TestBuildOptionsExposeNoBackendSemanticMode|TestFeatureRegistryCoversReleaseStatusesAndKeyBoundaries' -count=1
```

Broad relevant gate:

```bash
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/backend/x64 ./compiler/internal/backend/x64core ./compiler/internal/backend/linux_x64 ./compiler/internal/opt ./compiler/internal/validation ./compiler ./compiler/tests/semantics ./tools/cmd/validate-manifest ./tools/cmd/verify-docs
```

After code changes, run `graphify update .`, `git diff --check`, stale/overclaim
scans, scratch scan, and `GOCACHE=$(pwd)/.cache/go-build-ideal-plan go clean
-cache`.

## Non-Claims

- No host CPU detector is implemented in this batch.
- No target-specific rewrite, AVX/AVX2 path, LTO behavior, PGO behavior,
  performance claim, or C/Rust parity claim is made.
- No public target-cpu/profile/LTO flag changes safe-program semantics.
