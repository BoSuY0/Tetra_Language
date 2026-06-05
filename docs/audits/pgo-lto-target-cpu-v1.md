# PGO / LTO / Target CPU Evidence v1

Status: P17.4 progress audit for the Ideal Master Plan.

## Summary

`compiler/internal/opt.PGOLTOTargetCPUCoverage()` emits schema
`tetra.optimizer.pgo_lto_target_cpu.v1` for the P17.4 target list. The current
bounded slice adds `tetra.optimizer.profile.v1`, a canonical JSON profile
collection format with schema validation, deterministic function/counter
ordering, duplicate identity rejection, and negative counter rejection.

The format is inert evidence. No optimizer pass consumes profile data in this
slice. The optimizer manager now accepts that validated profile as internal
`Options.ProfileInput`, records `profile_input_policy` in pass reports and
translation-validation metadata, and rejects profile-guided rewrite policy until
a dedicated validation hook exists. Target-cpu feature detection now has an
internal evidence-only target-feature model with portable baseline fallback and
a guarded codegen contract. No host CPU detector, target-specific rewrite, or
target-specific codegen is implemented. The LTO/incremental row now has an
internal `tetra.incremental.module_summary.v1` summary schema with dependency
hash contract evidence and an explicit non-consumer boundary.
`PGOLTOTargetCPUSafeSemanticsClosure()` now validates the final P17.4
safe-semantics row and records negative fake-claim guards. `BuildOptions` still
exposes no public PGO/profile/LTO/target-cpu semantic tuning flag, so
safe-program semantics remain unchanged.

## Coverage

| Target | Status | Evidence | Boundary |
| --- | --- | --- | --- |
| profile collection format | `implemented_narrow` | `ProfileCollection`, `MarshalProfileCollection`, `ParseProfileCollection`, `TestProfileCollectionFormatV1RoundTripsAndRejectsUnsafeDrift` | `tetra.optimizer.profile.v1` records canonical JSON function entry counts and named counters, rejects duplicate function/counter identity and negative counter JSON, and remains inert evidence only. |
| PGO input to optimizer | `implemented_narrow` | `Options.ProfileInput`, `TestManagerAcceptsProfileInputAsValidatedMetadataWithoutChangingIR`, `TestManagerRejectsProfileGuidedRewritePolicyUntilValidationExists`, `TestBuildOptimizationValidationMetadataRecordsMachineCheckableEvidence` | Internal optimizer input is limited to validating a canonical profile, recording profile digest and `profile_input_policy=unused` in pass reports and validation metadata, running normal translation validation for the pass, and rejecting profile-guided rewrite policy. No profile-guided rewrite is implemented. |
| target-cpu feature detection | `implemented_narrow` | `CodegenOptions.TargetFeatureEvidence`, `TestTargetFeatureModelUsesPortableBaselineAndRejectsUnsafeDrift`, `TestCodegenOptionsTargetFeatureGuardIsEvidenceOnly`, `TestNativeCodegenOptionsUsePortableTargetFeatureBaseline` | Internal target-feature evidence resolves a portable baseline fallback, records `sse2` for 64-bit x64/x32 register targets, rejects unknown features and explicit feature sets below baseline, and exposes guarded feature queries without enabling target-specific rewrites. No host CPU detector or public target-cpu flag is implemented. |
| LTO/incremental module summary | `implemented_narrow` | `IncrementalModuleSummary`, `TestIncrementalModuleSummaryV1RecordsDependencyHashContractAndRejectsConsumers`, `TestPGOLTOTargetCPUCoverageAuditsP17PlanList` | `tetra.incremental.module_summary.v1` records module source hash, dependency hash, public API hash, external callee/type dependency inputs, cross-module validation rows, and explicit `codegen_consumer=false` plus `linker_consumer=false`. No LTO optimizer, cross-module inlining, linker consumer, codegen consumer, cache mode, or incremental speedup claim is implemented. |
| safe semantics for flags | `implemented_narrow` | `compiler/compiler.go::BuildOptions`, `TestBuildOptionsExposeNoBackendSemanticMode`, `PGOLTOTargetCPUCoverage`, `ValidatePGOLTOTargetCPUSafeSemanticsClosure`, `TestPGOLTOTargetCPUSafeSemanticsClosureRejectsFakeClaims` | No public `BuildOptions` field enables PGO, profile, LTO, or target-cpu behavior; profile parsing is evidence-only, all registered optimizer passes declare `profile_input_policy=unused`, no optimizer pass consumes profile counts, target-feature data is internal evidence only, LTO/incremental summaries are non-consumer evidence only, and the closure validator rejects fake semantic-changing coverage rows, incomplete rows, fake profile-format optimizer input, fake target-cpu/LTO optimizer input, and missing safe-program truth facts. |

## Profile Format

The v1 profile collection is a JSON object with:

- `schema_version`: exactly `tetra.optimizer.profile.v1`.
- `program_hash`: required `sha256:` program identity.
- `target_triple`: required target triple string.
- `functions`: non-empty function rows with stable `id`, `name`,
  `entry_count`, and optional named counters.

`MarshalProfileCollection` validates before encoding and sorts functions by
`id` then `name`; counters sort by `kind` then `name`. `ParseProfileCollection`
uses strict JSON decoding, rejects unknown schema drift, rejects duplicate
function IDs/names and duplicate per-function counters, and lets JSON numeric
decoding reject negative unsigned counts.

## Optimizer Input

`Options.ProfileInput` is internal to `compiler/internal/opt`. `RunWithOptions`
validates the profile with the same v1 schema, records a stable profile digest,
and attaches the evidence to each pass report. Passes must declare
`profile_input_policy`; the registered optimizer passes currently use `unused`.
`validation.OptimizationValidationMetadata` records the policy and digest so the
report is machine-checkable.

This is an input/reporting foundation only. A pass that asks for
`guided_rewrite` is rejected because profile-guided decisions need a dedicated
validation hook and negative tests before promotion.

## Target Feature Model

`compiler/internal/backend/x64.CodegenOptions` now has an internal
`TargetFeatures` field and evidence helpers. The zero value resolves to a
portable baseline. For 64-bit register x64/x32 targets, that baseline includes
`sse2`; for 32-bit register targets it records no SIMD feature. Explicit
feature sets are canonicalized, reject unknown feature names, and must include
the portable baseline.

This is a guarded codegen contract only. It allows future codegen paths to ask
whether a feature is present, but this slice does not add host CPU probing,
public target-cpu flags, target-specific rewrites, AVX/AVX2 lowering,
throughput claims, or safe-semantics changes.

## Incremental Module Summary

`compiler/internal/cache.IncrementalModuleSummary` records
`tetra.incremental.module_summary.v1` JSON evidence for a module's source hash,
dependency hash, public API hash, external callee/type dependency names, and
validation rows. The dependency hash contract uses the same
`DepSigHashFromDepsWithInterfaceHashes` path that already protects object-cache
keys from cross-module signature and interface-hash drift.

This summary is not an optimizer input. Validation rejects missing hashes,
missing validation rows, and any `codegen_consumer` or `linker_consumer` value
set to `true`.

## Safe-Semantics Closure

`compiler/internal/opt.PGOLTOTargetCPUSafeSemanticsClosure()` records the final
P17.4 safe-semantics evidence. It accepts the P17.4 coverage matrix only when
all rows are `implemented_narrow`, all rows report
`changes_safe_semantics=false`, no row has missing facts, and each row preserves
its evidence-only or guarded boundary.

`ValidatePGOLTOTargetCPUSafeSemanticsClosure()` rejects fake completion and
fake semantic-changing claims. The negative guard covers semantic-changing
coverage rows, incomplete rows, missing required facts, fake profile-format
optimizer input, fake LTO optimizer input, missing safe-program truth facts,
profile-guided rewrite policy, target-specific optimization evidence, and
LTO codegen/linker consumers.

## Non-Claims

- No profile-guided optimizer rewrite is implemented.
- No host CPU detector or target-feature-specific codegen is implemented.
- No LTO optimizer, cross-module inlining, linker/codegen consumer, cache
  speedup, or incremental performance claim is made.
- No throughput, C/Rust parity, or performance claim is made.
- No PGO, LTO, target-cpu, or profile flag changes safe-program semantics.
