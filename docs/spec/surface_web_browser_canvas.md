# Surface wasm32-web Browser Canvas Target

Status: current for the scoped Surface v1 browser-canvas release target.

`tetra.surface.browser-canvas-target.v1` is the first-class wasm32-web
browser-canvas target evidence object. It is required for
`wasm32-web-browser-canvas-release-v1` production reports and keeps the web
target distinct from DOM, React, legacy metadata UI, and user-authored
JavaScript application logic.

## Required Evidence

A valid browser-canvas target report must include:

- `level:"wasm32-web-first-class-browser-canvas-target-v1"`;
- `target:"wasm32-web"` and `runtime:"surface-wasm32-web"`;
- `host_abi:"tetra.surface.host-abi.v1"`;
- `backend:"browser-canvas-rgba-accessible"`;
- `trace_schema:"tetra.surface.browser-canvas-trace.v1"`;
- compiler-owned boot and compiler-owned loader evidence;
- browser canvas, browser input, clipboard, composition, accessibility
  snapshot, and accessibility mirror evidence;
- at least two presented frame checksums;
- pointer, keyboard, resize, and text-input event evidence;
- component app, compiler-owned loader, and runner-trace artifact evidence.

The JavaScript loader is compiler-owned boot glue. It is not user application
logic and must not be presented as a DOM renderer.

## Rejection Rules

The validator rejects:

- Node runtime substitution as browser-canvas production evidence;
- DOM snapshot output counted as the Surface renderer;
- user script command dispatch counted as Tetra app logic;
- metadata-only and legacy `.ui.*` sidecars;
- React runtime evidence.

This target can be a production web Surface target only inside the bounded
`surface-v1-linux-web` claim. It is not DOM/React compatibility and does not
allow user-authored JavaScript UI logic.
