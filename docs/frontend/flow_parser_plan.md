# Flow Parser Migration Plan

Status: decided (2026-04-26)

This document resolves:

- TODO 237: decide whether `normalizeFlowSyntax` remains compatibility tooling or is replaced by a Flow parser.
- Open investigation 659: decide whether Flow gets a native parser or keeps normalization during migration.

## Decision

Tetra will ship a native Flow parser as the canonical frontend path for v1.0.
`normalizeFlowSyntax` remains only as temporary migration compatibility tooling
while this cutover is in progress.

This means:

- Release-profile compile/check/fmt paths must run through the native Flow parser.
- `normalizeFlowSyntax` is allowed only for migration workflows, not as the
  canonical v1.0 parser path.
- Before v1.0 release, legacy brace syntax and normalization-first parsing must
  be removed from the canonical compiler path.

## Migration Phases

### Phase 0: Decision Lock And Docs Alignment

Goal: make the posture explicit in frontend docs and specs.

Done when:

- This document exists and is referenced by the Flow MVP spec.
- The Flow parser posture is explicit: native parser is canonical; normalization
  is migration-only.

Verification:

```sh
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
```

### Phase 1: Dual-Path Migration (Native Primary, Normalization Compatibility)

Goal: make native Flow parsing the primary implementation path while keeping
normalization available only for migration inputs.

Done when:

- Parser and formatter tests validate native Flow parsing for supported syntax.
- Migration compatibility behavior is covered without redefining canonical parse
  behavior.
- Flow-only source scanning remains green for release-profile source trees.
- Frontend regression coverage includes `test`/`expect` tokenization, test
  declaration diagnostics, CRLF/tab/Unicode span checks, and migration
  normalization boundary diagnostics.

Verification:

```sh
go test ./compiler/internal/frontend ./compiler/... -run 'Flow|Parser|Lexer|Format'
go run ./tools/cmd/validate-flow-only examples lib __rt compiler/selfhostrt
```

### Phase 2: Release-Profile Cutover

Goal: remove normalization-first behavior from canonical release paths.

Done when:

- Canonical release checks no longer depend on `normalizeFlowSyntax`.
- Legacy brace syntax is removed from the canonical compiler path.
- Release-profile docs, examples, formatter output, and smoke coverage are
  Flow-only.

Verification:

```sh
go run ./tools/cmd/validate-flow-only examples lib __rt compiler/selfhostrt
bash scripts/release_v1_0_gate.sh
```

### Phase 3: Compatibility Tooling Sunset

Goal: either retire `normalizeFlowSyntax` or keep it as explicitly
non-canonical migration tooling with clear deprecation rules.

Done when:

- Migration tooling status is documented (retained short-term or removed).
- No canonical compile/check/fmt code path routes through normalization.

Verification:

```sh
go test ./compiler/internal/frontend ./compiler/... -run 'Flow|Parser|Lexer|Format'
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
```

## Gate Commands Summary

Use these commands as the decision gates during migration and release
readiness:

```sh
go test ./compiler/internal/frontend ./compiler/... -run 'Flow|Parser|Lexer|Format'
go run ./tools/cmd/validate-flow-only examples lib __rt compiler/selfhostrt
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
bash scripts/release_v1_0_gate.sh
```
