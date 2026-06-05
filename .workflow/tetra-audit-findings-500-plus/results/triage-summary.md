# Tetra Audit Findings 500+ Triage Summary

Source: `/home/tetra/Downloads/tetra_audit_findings_500plus.md`
Findings: `748`

## Resolution Status Counts

| Status | Count |
| --- | ---: |
| `referenced_artifact_live_or_ignored_policy_classified` | 347 |
| `documented_bug_fixed_ledger` | 195 |
| `negative_guard_or_policy_marker_verified` | 82 |
| `ignored_report_artifact_policy_classified` | 34 |
| `documentation_reference_fixed_or_stale_audit_verified` | 31 |
| `live_artifact_exists_dump_unverifiable` | 15 |
| `release_report_artifact_external_blocker` | 15 |
| `documented_scope_boundary_verified` | 8 |
| `fixed_verified` | 6 |
| `historical_release_evidence_blocked` | 5 |
| `fixed_by_compatibility_verified` | 4 |
| `dump_only_blocked` | 3 |
| `relative_documentation_reference_verified` | 2 |
| `historical_generated_snapshot_blocked` | 1 |

## Category / Status Matrix

| Category | Status | Count |
| --- | --- | ---: |
| Build/test failure | `fixed_verified` | 1 |
| Documented bug / regression-risk ledger | `documented_bug_fixed_ledger` | 195 |
| Dump integrity | `dump_only_blocked` | 3 |
| Missing or unverifiable referenced artifact/path | `documentation_reference_fixed_or_stale_audit_verified` | 31 |
| Missing or unverifiable referenced artifact/path | `historical_generated_snapshot_blocked` | 1 |
| Missing or unverifiable referenced artifact/path | `ignored_report_artifact_policy_classified` | 34 |
| Missing or unverifiable referenced artifact/path | `referenced_artifact_live_or_ignored_policy_classified` | 347 |
| Missing or unverifiable referenced artifact/path | `relative_documentation_reference_verified` | 2 |
| Missing or unverifiable referenced artifact/path | `release_report_artifact_external_blocker` | 15 |
| Placeholder / unfinished / fake marker | `documented_scope_boundary_verified` | 8 |
| Placeholder / unfinished / fake marker | `negative_guard_or_policy_marker_verified` | 82 |
| Release evidence contradiction | `historical_release_evidence_blocked` | 2 |
| Shell/release-script robustness | `fixed_verified` | 5 |
| Stale/generated evidence | `historical_release_evidence_blocked` | 3 |
| Toolchain/version mismatch | `fixed_by_compatibility_verified` | 4 |
| Unverifiable binary artifact | `live_artifact_exists_dump_unverifiable` | 15 |

## Severity / Status Matrix

| Severity | Status | Count |
| --- | --- | ---: |
| critical | `fixed_verified` | 1 |
| critical | `historical_release_evidence_blocked` | 2 |
| high | `dump_only_blocked` | 1 |
| high | `fixed_by_compatibility_verified` | 4 |
| high | `historical_release_evidence_blocked` | 3 |
| high | `ignored_report_artifact_policy_classified` | 25 |
| high | `referenced_artifact_live_or_ignored_policy_classified` | 104 |
| high | `release_report_artifact_external_blocker` | 9 |
| medium | `documentation_reference_fixed_or_stale_audit_verified` | 13 |
| medium | `documented_bug_fixed_ledger` | 195 |
| medium | `documented_scope_boundary_verified` | 8 |
| medium | `dump_only_blocked` | 2 |
| medium | `fixed_verified` | 5 |
| medium | `historical_generated_snapshot_blocked` | 1 |
| medium | `live_artifact_exists_dump_unverifiable` | 15 |
| medium | `negative_guard_or_policy_marker_verified` | 82 |
| medium | `referenced_artifact_live_or_ignored_policy_classified` | 200 |
| medium | `release_report_artifact_external_blocker` | 6 |
| low | `documentation_reference_fixed_or_stale_audit_verified` | 18 |
| low | `ignored_report_artifact_policy_classified` | 9 |
| low | `referenced_artifact_live_or_ignored_policy_classified` | 43 |
| low | `relative_documentation_reference_verified` | 2 |

## Closed Live Code / Script Findings

- `F-0001`: `tools/cmd/validate-v0-4-readiness` test package does not compile -> `fixed_verified`
- `F-0002`: `go.mod` declares `go 1.20` while the code uses newer test APIs -> `fixed_by_compatibility_verified`
- `F-0003`: `compiler/go.mod` declares `go 1.20` while the code uses newer test APIs -> `fixed_by_compatibility_verified`
- `F-0004`: `cli/go.mod` declares `go 1.20` while the code uses newer test APIs -> `fixed_by_compatibility_verified`
- `F-0005`: `tools/go.mod` declares `go 1.20` while the code uses newer test APIs -> `fixed_by_compatibility_verified`
- `F-0744`: uses `node` inside `smoke_source_for_case` but has no prerequisite check before invoking it -> `fixed_verified`
- `F-0745`: starts `python3 -m http.server` but only removes the temp directory in the EXIT trap -> `fixed_verified`
- `F-0746`: uses a fixed small port range 8711-8715 with a time-of-check/time-of-use race -> `fixed_verified`
- `F-0747`: requires evidence strings with literal `<path>` in command names -> `fixed_verified`
- `F-0748`: stores only the linux-x64 artifact as legacy `native` evidence -> `fixed_verified`

## Classified Blocker / Policy Groups

- `release_report_artifact_external_blocker`: 15 findings (F-0147, F-0148, F-0149, F-0150, F-0158, F-0159, F-0160, F-0163, F-0164, F-0173...)
- `historical_release_evidence_blocked`: 5 findings (F-0024, F-0025, F-0026, F-0027, F-0028)
- `historical_generated_snapshot_blocked`: 1 findings (F-0174)
- `dump_only_blocked`: 3 findings (F-0006, F-0007, F-0008)

## Integration Notes

- Placeholder/fake marker findings are classified as negative guards or documented support boundaries when the live line is a validator/test/policy rejection of fake evidence rather than a claim placeholder.
- `reports/` and release `artifacts/` findings are classified through `docs/release/artifact_policy.md`; regenerating historical release archives requires an explicit release-lane decision.
- Doc-path findings for moved tests and path-like prose were fixed where the replacement was unambiguous.
- Example README links that are valid relative to their own directory are classified as source-relative documentation references.
- Documented bug ledger findings remain historical regression evidence because the source ledgers state fixed/closed status.
