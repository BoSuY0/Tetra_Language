# Surface Layout Engine Evidence

`tetra.surface.layout-engine.v1` is the Block System evidence contract for
production-grade responsive layout behavior without a CSS runtime. It is emitted
inside `tetra.surface.runtime.v1` reports when Block layout evidence is present.

The evidence covers:

- layout modes: row, column, stack, grid, dock, absolute, overlay, and scroll;
- constraints: min, max, fit, fill, fixed, density, overflow, and clip;
- density: target-independent DPI/device-pixel-ratio facts and pixel-grid
  snapping;
- overflow: explicit visible/clip/scroll rules, scroll bounds checks, and
  rejected accidental hidden overflow;
- invalidation: resize and scroll dirty-root traces with before/after checksums
  and cache reuse/invalidation counts;
- cache budget: bounded LRU layout cache evidence with entry and byte limits;
- responsive profiles: app shell, settings forms, dashboards, and editor shells.

The validator rejects:

- CSS flexbox/grid parity claims;
- platform widget layout claims;
- accidental overflow-hidden behavior;
- unbounded layout cache evidence;
- missing DPI/density evidence;
- invalid resize/scroll invalidation traces.

This contract is part of the experimental `ui.surface-block-system` track. It
does not promote the Block System to current production Surface support by
itself, and it does not claim Electron, React, CSS, DOM, Chromium, GPU renderer,
or cross-platform desktop replacement parity.
