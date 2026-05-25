# UI Platform Runtime Validator

Boundary: validates target-host `tetra.ui.platform.v1` reports for the
post-v0.4 full-platform UI runtime promotion gate.

This validator accepts only fresh runtime-backed Windows/macOS UI evidence
collected on the matching target host. Reports must include an RFC3339
`generated_at` timestamp; CLI validators reject target-host evidence older than
the default freshness window. It rejects blocked, stale, build-only,
metadata-only, runtime-less, docs-only, sidecar-only, fake/mock/placeholder,
and `startup_failure` evidence, including fake/mock markers embedded as path
segments or executable names. It does not execute UI apps itself; smoke scripts
or CI runners must produce the report, and this package verifies the report
shape and required runtime evidence.
