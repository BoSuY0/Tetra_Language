# API Diff Policy

Status: Accepted on 2026-04-26 (TODO 662).

Policy decision: release gating uses a committed JSON baseline artifact and a machine-readable diff report derived from generated API docs metadata plus canonical API symbol records.

## Baseline Artifact

Canonical baseline path (committed): `docs/baselines/api-diff-baseline.v1alpha1.json`.

Baseline JSON schema name: `tetra.api.diff-baseline.v1alpha1`.

Required baseline fields:

- `schema`: must be `tetra.api.diff-baseline.v1alpha1`.
- `created_at`: RFC 3339 UTC timestamp.
- `source_docs`: path used to generate the baseline.
- `source_docs_sha256`: SHA-256 of full markdown bytes, prefixed with `sha256:`.
- `api_metadata`:
  - `schema`: must be `tetra.api.v1alpha1`.
  - `api_hash`: must match docs metadata hash.
  - `module_count`: must match docs metadata module count.
  - `entry_count`: must match docs metadata entry count.
- `symbols`: sorted array of canonical symbol records.

Canonical symbol record:

- `id`: `<module>::<section>::<entry>`.
- `module`: text from `## <module>`.
- `section`: text from `### <section>`.
- `entry`: text from API bullet without markdown markers/backticks.
- `symbol_hash`: `sha256:` + SHA-256 of `<module>\n<section>\n<entry>`.

## Diff Schema

Diff JSON schema name: `tetra.api.diff.v1alpha1`.

Required diff fields:

- `schema`: must be `tetra.api.diff.v1alpha1`.
- `baseline_path`: baseline file path used for comparison.
- `candidate_path`: candidate docs path used for comparison.
- `baseline`: object with `api_hash`, `module_count`, `entry_count`.
- `candidate`: object with `api_hash`, `module_count`, `entry_count`.
- `summary`: object with integer `added`, `removed`, `changed`.
- `changes`: sorted array by `id`, then `kind`.

Change object fields:

- `kind`: one of `added`, `removed`, `changed`.
- `id`: canonical symbol id.
- `module`: module name.
- `section`: section name.
- `before_entry`: previous entry text or empty for `added`.
- `after_entry`: new entry text or empty for `removed`.
- `before_hash`: previous `symbol_hash` or empty for `added`.
- `after_hash`: new `symbol_hash` or empty for `removed`.
- `severity`: `minor` for `added`; `major` for `removed` and `changed`.

## Gate Command Contract

Command remains `go run ./tools/cmd/validate-api-docs`.

Contract:

- `--docs <path>`: required; validates docs shape/metadata/hash as today.
- `--write-baseline <path>`: optional; writes baseline JSON after successful docs validation.
- `--baseline <path>`: optional; compares candidate docs against baseline artifact.
- `--diff-out <path>`: optional; writes diff JSON, requires `--baseline`.
- `--enforce <mode>`: optional; one of:
  - `none` (default): validation and diff generation only.
  - `no-breaking`: fail when any `major` change exists.
  - `no-change`: fail when any change exists.

Error contract:

- Exit `0`: requested validation/diff operation succeeded and enforcement passed.
- Exit `1`: docs invalid, baseline/diff invalid, or enforcement failed.
- Exit `2`: flag usage error.

## Release Gate Decision

Release gate mode is `--enforce no-change` until versioned API compatibility rules are implemented.

Initial release-gate call shape:

```sh
go run ./tools/cmd/validate-api-docs \
  --docs reports/api-docs.md \
  --baseline docs/baselines/api-diff-baseline.v1alpha1.json \
  --diff-out reports/api-diff.json \
  --enforce no-change
```
