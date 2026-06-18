# Surface Dev Loop

Status: experimental Block-system developer evidence for
`ui.surface-block-system`.

`tetra.surface.dev-loop.v1` is the P24 fast development loop contract for
Surface templates and hot-reload evidence. It is intentionally JSON-first and
deterministic: the supported production-evidence path is `tetra surface dev
--once`, which records a baseline source hash, then validates a later source
hash delta as reload evidence.

## CLI Contract

```sh
tetra new surface-app [--template surface-dashboard] <dir>
tetra surface dev --project <dir> --once \
  --state <dir>/.tetra/surface-dev-state.json \
  --report <dir>/.tetra/surface-dev-report.json
go run -buildvcs=false ./tools/cmd/validate-surface-dev-report \
  --report <dir>/.tetra/surface-dev-report.json
tetra surface package <dir> -o <dir>/dist/surface-app.tdx
```

The reusable gate is:

```sh
bash scripts/release/surface/dev-loop-gate.sh \
  --report-dir reports/surface-prod/P24-dev-loop-gate
```

The deterministic reload smoke is two-phase:

1. Run `tetra surface dev --once` to record `tetra.surface.dev-state.v1`.
2. Edit the source file.
3. Run `tetra surface dev --once` again to write a valid
   `tetra.surface.dev-loop.v1` report.

`--require-change` rejects reports that do not contain a real source hash
delta. This prevents hot-reload claims from being backed only by a docs-only
template or an unchanged file scan.

## Template Set

The required P24 template names are:

- `surface-minimal`
- `surface-dashboard`
- `surface-form`
- `surface-editor-shell`
- `surface-tray-app`
- `surface-web-canvas`

`tetra new surface-app` defaults to `surface-minimal`. Each scaffold writes
`surface.template.json` with schema `tetra.surface.template.v1`, the template
name, the full required template set, and the expected developer commands.

## Report Requirements

A valid `surface-fast-dev-loop-v1` report includes:

- `reloads[]` with `source-change-reload`, previous/current SHA-256, mtimes,
  `change_detected`, `rebuild_triggered`, `reload_applied`, and
  `inspector_updated`.
- Operation rows for check, run, inspect, and package. The check row is backed
  by the compiler semantic pass; run/inspect/package rows are scoped developer
  evidence and do not claim target-host production runtime.
- Template smoke coverage for all six required template names.
- State preservation policy `schema-compatible-owned-state-only`, preserving
  owned app state only when the source state schema remains compatible.
- Negative guards for missing source change traces, Electron dev-server
  substitution, React Fast Refresh, CSS runtime injection, and DOM hot reload.

## Nonclaims

This slice is not an Electron dev server, not React Fast Refresh, not CSS HMR,
not DOM hot reload, not browser devtools parity, and not state preservation
across incompatible schemas. It also does not promote Block to production
support or replace P26 packaging/signing requirements.
