# Surface Performance And Memory Budget Design

Status: approved by the active Surface Electron/React Beauty goal and the
external P18 plan. This is the repo-local design gate for
`SURFACE-BEAUTY-P18`.

## Goal

Add report-backed Surface performance and memory budgets without making an
unsupported speed comparison. P18 should make the current Linux/web Surface
release gate fail when runtime reports omit startup/frame/memory/binary/cache/
CPU-power proxy evidence or when docs/reports claim "faster than Electron"
without a fair, explicitly non-official methodology.

## Observed Repo Shape

- `tools/validators/surface/report.go` already validates generic
  `tetra.surface.runtime.v1` reports and final `tetra.surface.release.v1`
  summaries.
- Block System reports already carry `memory_budget`; Morph reports already
  carry a Morph memory budget. Those are useful lower-layer evidence, but they
  do not cover startup, binary size, or CPU/power proxy for release runtime
  reports.
- `tools/cmd/surface-runtime-smoke` is the common report producer for
  headless, linux-x64, wasm32-web, app-model, app-shell, toolkit, accessibility,
  Block, and Morph slices.
- `scripts/release/surface/release-gate.sh` writes the final release summary
  and runs release-state, artifact-hash, and claim validators.

## Design

Add `surface_performance_budget` to `tetra.surface.runtime.v1` reports:

- `schema:"tetra.surface.performance-budget.v1"`
- `model:"surface-performance-budget-v1"`
- `release_scope:"surface-v1-linux-web"`
- `source`, `target`, and `runtime` copied from the runtime report
- startup rows: launch-to-first-frame milliseconds and budget
- frame rows: frame count, p50/p95 build/present milliseconds, budget, idle/work
  loop counters
- scene rows: block count, recipe expansion count, paint commands, layout
  passes, text runs
- cache/memory rows: glyph, asset, layout, paint cache bytes; framebuffer peak
  and total bytes; allocation count/bytes; RSS measured flag plus
  `peak_rss_bytes`
- binary rows: target artifact path, size, and budget
- CPU/power proxy rows: idle/work frame-loop counters and no-real-power claim
- methodology rows: local deterministic report methodology, no official
  benchmark result, no Electron speed superiority claim
- negative guards: bounded caches, unbounded cache rejection, stale report
  rejection, no faster-than-Electron claim, no benchmark parity claim, no
  missing peak-memory field

The final release summary gains:

```json
"performance_budget": "surface-performance-budget-v1"
```

The release gate validates the app-shell release report through
`validate-surface-performance-budget`, and release-state validation requires
`surface_performance_budget` in the app-shell report. Broader Surface v1
runtime release validation also requires the field for final runtime reports.

## Nonclaims

P18 does not claim Tetra is faster than Electron, lower-power than Electron,
memory-superior to Electron, an official benchmark result, or cross-machine
benchmark evidence. Fair Electron comparison remains allowed only as future
local non-official methodology evidence.

## Test Strategy

- RED tests:
  - runtime report missing `surface_performance_budget` fails P18.
  - `validate-surface-performance-budget` accepts a valid runtime report.
  - fake "faster than Electron" claim fails.
  - missing peak memory field fails.
  - release summary missing `performance_budget` fails.
  - release-state accepts no app-shell report without P18 evidence.
- GREEN implementation:
  - add schema structs and validation helpers to `tools/validators/surface`.
  - add CLI `tools/cmd/validate-surface-performance-budget`.
  - emit P18 budget evidence from `surface-runtime-smoke`.
  - wire standalone app-shell smoke and final release gate.
  - update docs, feature registry, generated manifest, and workflow evidence.

## Rollout

Keep P18 scoped to local deterministic Surface release reports. P22/P23 can add
larger reference-app and packaging performance matrices later; P18 is the
mandatory budget floor that prevents unbounded caches and unsupported
performance marketing claims.
