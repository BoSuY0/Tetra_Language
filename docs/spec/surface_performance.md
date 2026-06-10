# Surface Performance And Memory Evidence

Surface performance evidence is experimental under `ui.surface-block-system`.
The P31 contract uses schema `tetra.surface.perf-report.v1` and level
`surface-performance-memory-v1`. The supported deterministic path is
`scripts/release/surface/perf-gate.sh`, `surface-perf-smoke`, and
`validate-surface-perf-report`.

The report is scoped to `surface-v1-scoped-linux-web-performance-memory` and
the release scope `PROD_STABLE_SCOPED_LINUX_WEB_APP_UI`. It covers linux-x64
real-window and wasm32-web browser-canvas evidence only. macOS, Windows,
wasm32-wasi, GPU compositor performance, platform-native widget performance,
and broad Electron/Chromium parity remain nonclaims.

Required budget rows:

- `startup_time` in milliseconds;
- `first_frame_time` in milliseconds;
- `steady_frame_time_p95` in milliseconds;
- `peak_rss` in megabytes;
- `frame_allocations` per frame;
- `layout_cache_bytes`;
- `glyph_cache_bytes`;
- `asset_cache_bytes`;
- `binary_size` in bytes;
- `cpu_idle_power_proxy` as a minimum percentage;
- `input_latency_p95` in milliseconds;
- `animation_frame_jitter_p95` in milliseconds.

Each row must include a positive budget, a positive observed value, a pass
decision, and a comparator. Cache rows must also be backed by bounded cache
evidence for layout, glyph, and asset caches with an eviction policy such as
`bounded-lru`.

Baseline evidence is required. A valid baseline records the same app shape, the
same OS/target, the same cold/warm state, a captured hardware/environment
record, a 40-hex commit, and a relative artifact path. The gate writes baseline
artifacts under `baselines/` and validates them with the main report through
artifact hashes.

Electron comparison evidence is a fairness record, not a promotional claim. The
report may only claim faster-than-Electron performance when the same app shape,
same OS/target, same cold/warm state, same hardware/environment, and at least
five statistically supported samples are present. The default P31 report makes
no faster-than-Electron claim.

P36 adds method-first Surface-vs-Electron comparison evidence with schema
`tetra.surface.electron-comparison-report.v1` and level
`surface-electron-comparison-method-v1`. The supported gate is
`scripts/release/surface/electron-comparison-gate.sh`, the report generator is
`surface-electron-comparison`, and the validator is
`validate-surface-electron-comparison-report`. The only allowed public
positioning is competitive with Electron in the supported scope. Public
benchmark-superiority claim rejection, cherry-picked hardware rejection,
missing variance rejection, unfair app shape rejection, missing environment
rejection, and single-smoke faster-than-Electron claim rejection are required.

Fake-claim rejection is part of the contract. The validator rejects:

- missing baseline environment evidence;
- impossible or non-positive performance numbers;
- unbounded layout/glyph/asset caches;
- unsupported faster-than-Electron claims;
- fastest UI framework claims;
- zero memory overhead claims.

This evidence is not a guarantee that Surface is the fastest UI framework. It
does not claim zero memory overhead, broad Electron replacement performance,
cross-platform desktop performance parity, GPU compositor timing parity, CSS
browser rendering parity, or Chromium benchmark parity.
