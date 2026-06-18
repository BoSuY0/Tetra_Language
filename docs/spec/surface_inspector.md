# Surface Developer Inspector

Status: experimental Block-system evidence for `ui.surface-block-system`.

`tetra.surface.inspector-snapshot.v1` is the JSON-first developer inspector
contract for diagnosing why a Surface view looks or behaves wrong. It is the
P23 developer-workflow slice: small, stable, validator-backed JSON before any
interactive inspector UI.

## Snapshot Contract

A valid snapshot uses level `surface-inspector-json-mvp-v1` and records:

- Block tree nodes with parent path, bounds, layout box IDs, and source
  locations.
- Morph style resolution status, including the resolved capsule when Morph
  evidence is present or a Block-only diagnostic when it is absent.
- Layout boxes, paint layers, events, focus order, accessibility nodes, and
  performance counters.
- Source locations for every inspectable view item.
- Negative guards for docs-only trees, missing source locations, missing
  layout boxes, missing accessibility views, and missing performance counters.

The validator is `tools/cmd/validate-surface-inspector-snapshot` and the
generator is `tools/cmd/surface-inspect`.

```sh
go run -buildvcs=false ./tools/cmd/surface-inspect \
  --report reports/surface-prod/P23-inspector/headless/surface-headless-block-system.json \
  --out reports/surface-prod/P23-inspector/headless/surface-inspector-snapshot.json

go run -buildvcs=false ./tools/cmd/validate-surface-inspector-snapshot \
  --snapshot reports/surface-prod/P23-inspector/headless/surface-inspector-snapshot.json
```

The user-facing CLI wrapper is:

```sh
tetra surface inspect --report <surface-runtime-report.json> --out <snapshot.json>
```

## Nonclaims

This slice is not an interactive devtools UI, not perfect source maps, not a
production profiler, and not browser devtools parity. It also does not promote
Block to production support. It gives developers a stable JSON snapshot that
can be diffed, validated, and attached to release evidence.
