# TOON Support Boundary

Status: current boundary for scoped TOON support work.

This document locks the format boundary for TOON as an opt-in structured
output/input format beside JSON. Public support is scoped to the surfaces below
that have matching code, tests, validators, scripts, and evidence.

## Spec Lock

Tetra targets the official TOON specification:

| Field | Value |
| --- | --- |
| Name | Token-Oriented Object Notation |
| Source | `https://github.com/toon-format/spec` |
| Specification file | `SPEC.md` |
| Version | `3.3` |
| Date | `2026-05-21` |
| Status | Working Draft |
| Media type | `text/toon` (provisional, UTF-8) |
| File extension | `.toon` |

Because the upstream document is a working draft, Tetra support is scoped to
the tested behavior in this repository. Future upstream TOON major-version
changes require compatibility review before Tetra may update this lock.

## Core Policy

JSON remains the default and canonical format for existing Tetra artifacts.
TOON is an opt-in adapter over the JSON data model. Existing JSON output shape,
schemas, diagnostics codes, report paths, release manifests, artifact hashes,
LSP JSON-RPC, HTTP `application/json`, and Eco lock/trust/package metadata must
not change as a side effect of TOON support.

TOON encoders and decoders must normalize through the same JSON-shaped data
model used by existing typed structs and validators. Validators that accept
TOON must decode TOON into the existing typed struct or canonical JSON value
and then reuse existing validation logic.

## Initial Supported Data Model

The initial Tetra TOON implementation may support only the tested JSON data
model subset:

- `null`, booleans, strings, and finite JSON numbers.
- Objects with string keys.
- Arrays, including empty arrays, mixed arrays, nested arrays, and nested
  objects.
- Uniform arrays of objects when the implementation can encode/decode them
  deterministically.
- Unicode strings and required string escaping.

The implementation must reject or explicitly defer unsupported host values,
non-finite numbers, malformed indentation, bad array lengths, row or column
count mismatches, duplicate sibling keys in strict mode, invalid string
escapes, invalid UTF-8, multiple unsupported top-level values, and configured
limit violations.

## Scoped Product Surfaces

Supported structured CLI/report/validator paths are JSON-compatible and
testable:

- `tetra targets --format=toon`
- `tetra features --format=toon`
- `tetra formats --format=toon`
- `tetra doctor --format=toon`
- `--diagnostics=toon` for commands that already emit structured JSON
  diagnostics
- `tetra test --report=toon` and `tetra test --format=toon`
- `tetra smoke --list --format=toon`
- `tetra smoke --report <path> --report-format=toon|both`
- `tetra lsp --stdio-smoke --format=toon`
- `tetra eco verify --lock-format=toon|both`
- `tetra eco seed export --format=toon|both`
- `tetra eco seed import --seed-format=toon|auto`
- `tetra eco needmap --format=toon|both`
- `tetra eco trust snapshot --format=toon|both`
- `tetra eco materialize --metadata-format toon|both`
- `tetra eco tetrahub mirror|fetch --format=toon|both`
- `tools/cmd/gen-manifest --format=toon|both`
- `scripts/ci/test-all.sh --report-format toon|both`
- selected validators that accept TOON by decoding into existing validation
  models

HTTP support is limited to Tetra-owned JSON-shaped runtime endpoints that have
an explicit `Accept: text/toon` branch and tests. Default HTTP responses remain
`application/json`; benchmark compatibility paths remain JSON-first.

## JSON-Canonical Boundaries

TOON does not replace:

- LSP JSON-RPC wire frames. `tetra lsp --stdio` remains Content-Length framed
  JSON-RPC.
- Default HTTP API `application/json` payloads.
- Canonical Eco package/store/index files such as `tetra.package.json`,
  registry/TetraHub `metadata.json`, vault `records.json`, and Todex archive
  compatibility metadata.
- Canonical release manifests such as `docs/generated/manifest.json` and
  `artifact-hashes.json`; TOON is a mirror/input format only where a validator
  or generator explicitly supports it.
- Third-party fixtures or `.ui.json` artifacts.
- Any secret/config migration path.

For path-based reports, `both` writes canonical JSON plus a `.toon` mirror.
Existing `.json` paths, release gates, and `--json-only` stdout contracts remain
JSON unless the command explicitly documents a TOON mirror.

## Claims Policy

This repository must not claim that TOON is faster, always smaller, cheaper, or
token-saving unless measured benchmark artifacts are added and validated in the
same change series. Byte-size examples are not token claims.

Allowed claim after evidence passes:

```text
TOON is an opt-in structured output/input format for the scoped Tetra surfaces
listed in this document. JSON remains default and canonical.
```

Forbidden without additional evidence:

```text
TOON replaces JSON.
TOON is supported everywhere.
TOON is faster.
TOON always saves tokens.
TOON is Tetra's internal protocol.
```

## Evidence Required Before Public Support

TOON support is public only when the implementation series provides:

- baseline JSON truth-map artifacts under `reports/toon/`;
- `internal/toon` encoder/decoder/conversion tests;
- deterministic encoder tests;
- strict decoder and negative/security tests;
- JSON/TOON roundtrip tests;
- selected CLI tests proving JSON behavior remains unchanged and TOON is opt-in;
- selected validator tests proving TOON input reuses existing validation;
- focused script/CI gate evidence;
- docs/generated manifest validation if generated docs change;
- final evidence under the active goal/report directory for the scoped surface.
