# TOON Expanded Structured Surfaces Plan

Date: 2026-06-13

## Goal

Extend the current scoped TOON support beyond the initial CLI/report subset to the structured
surfaces that are currently excluded:

- HTTP JSON payload surfaces;
- LSP-related JSON-RPC reporting surfaces;
- Eco metadata artifacts;
- release manifests;
- path-based release reports.

JSON must remain the default, canonical, and compatibility-preserving format unless a packet
explicitly proves an opt-in TOON mirror or alternate response. TOON must be an adapter over the same
typed JSON-shaped data models, not a parallel schema family.

## Current Baseline

Existing TOON support covers:

- `internal/toon`;
- `internal/outputformat`;
- selected CLI `--format=toon` outputs;
- `--diagnostics=toon`;
- `tetra test --report=toon` / `--format=toon`;
- selected validators through `tools/internal/reportdecode`;
- `scripts/ci/toon-format-check.sh`.

`docs/spec/standard_library/toon_support.md` currently lists the requested areas as explicit
non-support. This plan supersedes that boundary only after implementation, tests, docs, validators,
and evidence prove each new surface.

## Non-Negotiable Constraints

- JSON remains default and canonical for existing commands, wire protocols, release gates, and
  generated artifacts.
- Existing `.json` files, schemas, field names, diagnostics codes, path guards, report-dir freshness
  checks, and validators must not regress.
- TOON support must be opt-in through explicit flags, file extensions, content negotiation, or
  documented mirror generation.
- Do not silently reinterpret JSON-RPC as TOON. JSON-RPC framing remains JSON by protocol. Any TOON
  LSP support must be a report/transcript mirror or clearly separate non-JSON-RPC adapter.
- Do not claim TOON replaces JSON, is supported everywhere, is faster, or saves tokens unless
  measured evidence is added in this same work.
- Preserve unrelated dirty worktree changes.
- Use persistent Go caches under repo-local `.cache/` or
  `${XDG_CACHE_HOME:-$HOME/.cache}/tetra-language/...`; never use `/tmp` for `GOCACHE`.
- After code changes, run `graphify update .`.

## Implementation Packets

### TOON-EXP-P00 Baseline Inventory

Record the current same-commit truth map under `reports/toon-expanded-surfaces/P00/`.

Inventory:

- current git HEAD and dirty state;
- all JSON-only HTTP payload writers and validators;
- LSP `--stdio`, `--stdio-smoke`, transcript validators, and JSON-RPC tests;
- Eco JSON metadata producers and consumers;
- release manifest producers and validators;
- path-based report writers using `--report`, `--report-dir`, or hardcoded `*.json` names;
- docs that currently call these surfaces JSON-only.

Acceptance:

- inventory report lists concrete files and commands;
- each requested surface is classified as `json-only`, `candidate-toon`, `mirror-only`, or
  `blocked-by-protocol`;
- no implementation edits are made in this packet.

### TOON-EXP-P01 Format Policy And Shared Helpers

Create or extend shared helpers so new surfaces do not hand-roll TOON behavior.

Required design:

- one shared writer for `json`, `toon`, and optional `both`;
- one shared strict decoder for JSON/TOON reports that reuses typed models;
- deterministic file naming rules for path-based output:
  - explicit `--format=json|toon|both` or `--report-format=json|toon|both`;
  - extension inference only where it cannot conflict with existing flags;
  - `both` writes canonical `.json` plus `.toon` mirror;
- media type constants for `application/json` and `text/toon; charset=utf-8`;
- stable errors for unsupported format combinations.

Acceptance:

- unit tests prove JSON bytes remain unchanged where required;
- TOON output roundtrips through existing typed validators;
- unsupported formats fail with stable messages.

### TOON-EXP-P02 HTTP Payload Support

Add opt-in TOON support for Tetra-owned HTTP JSON payload surfaces without breaking JSON defaults.

Candidate scope:

- `compiler/internal/webrt` TechEmpower-style JSON endpoints such as `/json`, `/db`, `/queries`, and
  `/updates`;
- local HTTP validation/benchmark tools that inspect `Content-Type` and bodies;
- docs/audits that describe HTTP JSON boundaries.

Required behavior:

- default responses remain `application/json`;
- requests with `Accept: text/toon` may receive `text/toon; charset=utf-8` where the response body
  is JSON-shaped and covered by tests;
- unsupported or ambiguous Accept headers fall back to JSON unless the design explicitly requires
  `406`;
- validators check both JSON default and TOON opt-in responses;
- official TechEmpower/public benchmark claims remain JSON-only unless a separate benchmark policy
  says otherwise.

Acceptance:

- HTTP tests cover default JSON and opt-in TOON response bodies;
- TOON response decodes to the same typed payload as JSON;
- benchmark/report validators do not lose existing JSON requirements.

### TOON-EXP-P03 LSP Reporting And JSON-RPC Boundary

Add TOON where it is honest for LSP surfaces while preserving JSON-RPC.

Required behavior:

- `tetra lsp --stdio` remains framed JSON-RPC with JSON bodies;
- `tetra lsp --stdio-smoke` gains `--format=json|toon` or equivalent opt-in TOON report output;
- `tools/cmd/validate-lsp-smoke` accepts JSON and TOON by decoding into the same typed report model;
- `tools/cmd/validate-lsp-stdio` remains a JSON-RPC transcript validator, but a separate TOON
  transcript summary/mirror may be added if useful;
- docs must explicitly say TOON does not replace JSON-RPC wire frames.

Acceptance:

- existing framed JSON-RPC transcript tests still pass;
- new TOON smoke report tests pass;
- any TOON transcript mirror is clearly named as a report, not as JSON-RPC.

### TOON-EXP-P04 Eco Metadata Support

Add opt-in TOON for Eco metadata artifacts while keeping canonical JSON compatibility.

Candidate artifacts:

- lock/provenance metadata from `tetra eco verify --lock`;
- seed export/import metadata;
- needmap metadata;
- trust snapshot metadata;
- materialization metadata;
- publish/download/TetraHub package metadata where it is not embedded in a compatibility-critical
  archive contract;
- validators under `tools/cmd/validate-eco-*`.

Required behavior:

- existing `.json` outputs and inputs remain supported and default;
- TOON is opt-in through explicit format flags, output extension, or mirror generation;
- import/consumer paths that accept TOON must decode into the existing model and reuse existing
  validation;
- package/archive compatibility must not change unless the packet proves a versioned migration path.

Acceptance:

- Eco CLI tests cover JSON default and TOON opt-in for every selected artifact;
- Eco validators accept TOON fixtures for selected artifacts;
- archive/package compatibility tests prove legacy JSON metadata still works.

### TOON-EXP-P05 Release Manifests

Add validated TOON mirrors or opt-in output for release manifest surfaces.

Candidate artifacts:

- `docs/generated/manifest.json`;
- release gate manifests such as `*-manifest.json`;
- `artifact-hashes.json` manifests when safe;
- release readiness summary manifests consumed by validators.

Required behavior:

- JSON manifests remain canonical for release gates and docs unless the plan later proves a complete
  migration;
- TOON mirrors are generated only from the same in-memory model or canonical JSON source;
- validators reject stale, partial, or mismatched TOON mirrors;
- artifact hash coverage includes TOON mirrors when they are emitted.

Acceptance:

- manifest generator can produce JSON and TOON, or a separate mirror tool can convert canonical JSON
  to TOON deterministically;
- validators accept TOON input where documented;
- JSON manifest diff checks remain unchanged.

### TOON-EXP-P06 Path-Based Release Reports

Extend path-based report writers to support TOON or JSON+TOON mirrors.

Candidate paths:

- `scripts/ci/test-all.sh --report-dir`;
- `tetra smoke --report`;
- release smoke/gate scripts under `scripts/release/**`;
- benchmark reports such as `tetra-techempower-bench --report`;
- surface, memory, actor, RAM, full-platform, and v1.0 gate summaries that currently write hardcoded
  `.json` files.

Required behavior:

- existing report paths and JSON filenames remain default;
- new support is explicit through `--report-format=json|toon|both`, `--emit-toon-mirrors`, or
  documented extension inference;
- report-dir freshness and symlink guards remain intact;
- validators accept TOON reports only after strict typed decoding;
- stale evidence prevention remains at least as strong as before.

Acceptance:

- focused tests cover representative Go writers and shell-script report dirs;
- `both` mode, if implemented, writes JSON first and TOON mirror from the same model/source;
- validators reject malformed TOON and JSON/TOON mismatches.

### TOON-EXP-P07 Docs, Examples, And Contract Update

Update public docs after behavior is implemented.

Required updates:

- `docs/spec/standard_library/toon_support.md` moves requested surfaces from explicit non-support to
  scoped support or mirror-only support;
- `docs/user/reference/toon_format.md` shows examples for new flags and boundaries;
- `docs/spec/policy/cli_contracts.md` and release docs mention JSON default/canonical behavior;
- examples include at least one HTTP, LSP smoke, Eco metadata, release manifest, and path-based
  report TOON fixture.

Acceptance:

- docs do not overclaim;
- docs/generated manifest is regenerated and validated only if source docs require it.

### TOON-EXP-P08 CI And Evidence Gate

Create or extend a focused TOON expanded-surfaces gate.

Required checks:

- JSON default compatibility checks;
- TOON opt-in checks for each implemented surface;
- JSON/TOON semantic equivalence checks;
- validators for TOON inputs;
- script portability checks for any shell changes.

Acceptance:

- focused gate runs locally and is added to CI where appropriate;
- final evidence is written under `reports/toon-expanded-surfaces/`;
- final report records pass, partial, or blocked status with exact gaps.

### TOON-EXP-P09 Broad Validation And Final Verdict

Run final verification and choose the truthful status.

Minimum validation ladder:

- focused Go tests for `internal/toon`, `internal/outputformat`, CLI, LSP, Eco, HTTP/webrt, and
  selected validators;
- focused shell script tests for changed release/CI scripts;
- `bash -n` for changed shell scripts;
- docs manifest generation/validation if docs changed;
- `git diff --check`;
- `graphify update .`;
- broad Go package slice with repo-local `GOCACHE` and `GOTMPDIR`.

Final status:

- `DONE` only if every requested surface has implemented and verified TOON support or a clearly
  accepted protocol-scoped mirror;
- `PARTIAL` if any surface remains JSON-only, unimplemented, or unverified;
- `BLOCKED` only for a specific missing tool, permission, dependency, or protocol conflict that
  cannot be resolved inside the repo.

## Required Final Report

The final report must include:

- `Status`;
- `Completed`;
- `Scope covered`;
- `Validation`;
- `Evidence`;
- `Not verified / risks`;
- explicit list of surfaces still JSON-only, if any.

## Suggested Evidence Root

Use:

```text
reports/toon-expanded-surfaces/
```

Suggested workflow kernel:

```text
.workflow/toon-expanded-structured-surfaces/
```
