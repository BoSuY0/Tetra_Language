# PGO Optimizer Input Foundation Design

Status: approved by the active Ideal Master Plan `GOAL.md` P17.4 Bridge.

## Scope

This batch implements only the `pgo_optimizer_input` foundation row:

- internal optimizer-manager profile input API;
- pass-contract profile metadata;
- report and validation metadata evidence;
- negative safe-semantics tests for unsupported profile-guided rewrites.

It does not implement a profile-guided rewrite, target-cpu detection,
LTO/incremental summaries, performance claims, or public `BuildOptions` flags.

## Design

`compiler/internal/opt.Options` gains an internal `ProfileInput` field carrying
the existing canonical `tetra.optimizer.profile.v1` `ProfileCollection`.
`Manager.RunWithOptions` validates and summarizes that profile before any pass
runs. The summary is report evidence: schema version, program hash, target
triple, function count, total entry count, counter kinds, and a stable digest.

Every optimizer `Pass` declares a `profile_input_policy`. The only supported
policy in this foundation batch is `unused`; registered passes must report
that policy and preserve existing IR behavior. A pass that requests a
profile-guided rewrite policy is rejected until a separate profile-guided
translation-validation hook exists.

`validation.OptimizationValidationMetadata` records the profile input policy and
optional profile digest, so a pass report can prove whether profile data was
available and whether the pass consumed it.

## Verification

- RED/GREEN manager tests prove validated profile input is reported without IR
  changes.
- Contract tests prove every registered pass records `profile_input_policy`.
- Negative tests reject profile-guided rewrite policy without a validation hook.
- P17.4 coverage tests promote only `pgo_optimizer_input` to
  `implemented_narrow` and keep all non-claims explicit.
