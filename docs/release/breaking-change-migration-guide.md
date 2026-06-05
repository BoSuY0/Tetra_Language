# Breaking Change Migration Guide

Status: current policy for incompatible language, CLI, manifest, report schema,
stdlib, runtime ABI, and project-layout changes.

## Breaking Change Triage

Treat a change as breaking when it removes or changes documented public
behavior, stable diagnostic code identity, stable stdlib names, manifest shape,
report schema identity, runtime ABI behavior, or release-gated CLI behavior.

Before implementation, classify the change:

- `security_exception`: required to fix a security issue; needs compatibility
  impact, mitigation, focused tests, and release notes.
- `major_line`: belongs in a later major release line.
- `minor_line`: may be allowed for experimental or pre-stable surfaces when
  release notes and migration notes are present.
- `patch_line_blocked`: not allowed in patch releases unless it is a documented
  security exception.

## Migration Steps

Every accepted breaking change must include:

1. A release note naming the old behavior, new behavior, and affected users.
2. A migration path with code, command, manifest, or configuration examples.
3. A diagnostic, validator error, or docs pointer that identifies the old form.
4. A compatibility test or validator fixture for the old-form rejection.
5. A manifest or report-schema update when machine-readable artifacts change.

## Diagnostics

Stable diagnostic codes should remain stable when the semantic category remains
the same. Messages may improve, but code identity and JSON object shape must not
drift silently. New categories need a new code and a registry entry.

## Report Schema

Machine-readable report changes need an explicit versioned schema. A validator
must reject unsupported schemas instead of accepting unknown shape drift.

## Manifest

Generated manifest changes must be regenerated from the same branch state and
validated with `tools/cmd/validate-manifest`. Breaking manifest changes require
release notes and migration instructions for downstream tooling.

## API Diff

`docs/spec/api_diff_policy.md` remains the API review source of truth. Removed
or changed API entries are `breaking_requires_review`, and release gates stay in
`--enforce no-change` mode until versioned API compatibility rules exist.
