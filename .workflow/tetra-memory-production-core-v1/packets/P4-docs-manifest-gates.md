# Packet P4: Docs, Manifest, And Gates

## Objective

Read-only audit of documentation/manifest/release-gate conventions for adding Memory Production Core v1 artifacts.

## Context

Current slice must add docs under `docs/audits/`, `docs/spec/`, and likely `docs/design/`, and keep `docs/generated/manifest.json` compatible with `verify-docs` and `validate-manifest`.

## Files / Sources

Start with:

- `docs/generated/manifest.json`
- `tools/cmd/verify-docs`
- `tools/cmd/validate-manifest`
- `docs/audits/`
- `docs/spec/`
- `docs/design/`
- `scripts/ci/test.sh`

## Ownership

Read-only. Do not edit files.

## Do

- Identify how new docs should be registered.
- Identify validator expectations for docs/spec/audit artifacts.
- Identify existing memory/raw/provenance report validators that should be extended or left alone.
- Cite files/lines and commands.

## Do Not

- Do not update manifest or docs.
- Do not run broad CI.

## Expected Output

Markdown report with doc registration rules, required manifest updates, risk areas, files inspected, commands run, uncertainty.

## Verification

Read-only `rg`, `go test -list`, and targeted inspection only.
