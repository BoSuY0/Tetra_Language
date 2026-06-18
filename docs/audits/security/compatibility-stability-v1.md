# Compatibility and Stability v1

P24.2 adds `tetra.compatibility.stability.v1` as a bounded current-state compatibility and stability
report for scope `p24.2_compatibility_stability`.

## Evidence Rows

| Row                               | Current evidence                                                                                                                                                                                                        | Boundary                                                                                 |
| --------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- |
| `stable_diagnostic_codes`         | `DiagnosticCodeRegistry()` records public codes; `tools/cmd/validate-diagnostic` validates `tetra.release.v0_2_0.diagnostic-json.v1`; release docs preserve `TETRA0001` and `TETRA2001` compatibility.                  | Diagnostic messages are not frozen.                                                      |
| `versioned_report_schemas`        | Current P21-P24 reports carry explicit schemas such as `tetra.translation.validation.v2`, `tetra.runtime.hardening.v1`, and `tetra.compatibility.stability.v1`.                                                         | Versioned schemas do not imply automatic migration for every historical report.          |
| `manifest_compatibility_checks`   | `tools/cmd/validate-manifest` validates `tetra.release.v0_4_0.manifest-json.v1`, target order/coverage, builtins, runtime ABI, and `FeatureRegistry()` data.                                                            | This is current manifest validator evidence, not a future runtime ABI stability promise. |
| `breaking_change_migration_guide` | `docs/release/policy/breaking-change-migration-guide.md` and `docs/spec/policy/api_diff_policy.md` require breaking-change review, migration notes, and `--enforce no-change` until versioned API compatibility exists. | No automatic source migration is claimed.                                                |
| `deprecation_policy`              | `docs/release/policy/deprecation_policy.md`, `docs/release/v1_0/v1_0_x_maintenance_policy.md`, and `docs/spec/standard_library/stdlib_naming_versioning.md` require replacement paths and delayed removals.             | Removals still require explicit release-line policy or security exception documentation. |

## Non-Claims

- Full backward compatibility for all future versions is not claimed.
- Diagnostic messages are not frozen.
- Automatic migration for every breaking change is not claimed.
- Manifest/runtime ABI stability beyond current validated evidence is not claimed.
- Runtime behavior does not change.
- Safe-program semantics do not change.
- No performance claim is made.

## Verification

Focused validator:

```bash
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler -run 'P24CompatibilityStability' -count=1
```

Relevant package gates:

```bash
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/tests/frontend -run 'DiagnosticCodeRegistry|Diagnostic' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./tools/cmd/validate-diagnostic ./tools/cmd/validate-manifest -count=1
```
