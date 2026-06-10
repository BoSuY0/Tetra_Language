# Tetra Surface Renderer Backend Decision

Status: experimental GPU/compositor decision gate.

This document defines `tetra.surface.renderer-backend.v1`, the Surface
renderer/backend report used by `SURFACE-PROD-P07`.

The current decision is:

`software-only-prod-go-gpu-experimental`

That decision means the scoped `PROD_STABLE_SCOPED_LINUX_WEB_APP_UI` path may
continue on the software RGBA renderer as the production baseline. GPU and
compositor backends are optional future acceleration paths, not prerequisites
for the first scoped linux/web production claim.

## Current Production Baseline

The production rendering baseline is software RGBA:

- backend: `software-rgba`;
- evidence schema: `tetra.surface.software-renderer.v1`;
- gate: `scripts/release/surface/release-gate.sh`;
- required evidence: deterministic raster output, source-over alpha blending,
  scissor clipping, frame/repeat/golden checksums, resize/scale/DPI behavior,
  use-after-present rejection, and frame-alias rejection.

The renderer backend report must point to a same-commit software renderer report
or release-gate output. This keeps the production claim grounded in rendered
pixels instead of metadata.

## GPU / Compositor Status

`ui.surface-gpu` is experimental/nonclaim.

GPU renderer production is forbidden until a later gate provides target-host
backend evidence. A valid experimental report records:

- `gpu_compositor.status = experimental`;
- `gpu_compositor.production_claim = false`;
- `gpu_compositor.fallback = software-rgba`;
- `gpu_compositor.target_host_backend_reports = []`;
- `gpu_compositor.same_scene_equivalence = false`.

The required GPU/compositor capabilities are named now so later work cannot
declare production with a vague backend:

- `layer_compositing`;
- `transforms`;
- `clipping`;
- `texture_atlas`;
- `vsync_frame_timing`.

## Promotion Requirements

GPU production can be considered only when all claimed targets have real
target-host backend reports:

- linux target-host GPU smoke;
- web compositor/canvas evidence;
- Windows/macOS target-host GPU evidence if either target is claimed;
- fallback behavior to `software-rgba`;
- same-scene equivalence against the software renderer.

Until those reports exist and pass the validator, docs, manifests, release
reports, and public positioning must keep GPU as experimental/nonclaim.

## Validator

The validator command is:

```sh
go run -buildvcs=false ./tools/cmd/validate-surface-renderer-report \
  --report <report>
```

The test command for this gate is:

```sh
go test -buildvcs=false ./tools/cmd/validate-surface-renderer-report -count=1
```

The validator rejects:

- GPU production claims without target-host backend reports;
- missing named GPU/compositor capabilities;
- docs or reports that present GPU renderer production while backend evidence is
  absent;
- missing software renderer production baseline evidence.

## Report Shape

```json
{
  "schema": "tetra.surface.renderer-backend.v1",
  "status": "pass",
  "decision": "software-only-prod-go-gpu-experimental",
  "scope": "surface-prod-scoped-linux-web",
  "producer": "tools/cmd/validate-surface-renderer-report",
  "software_baseline": {
    "backend": "software-rgba",
    "production_path": true,
    "evidence_schema": "tetra.surface.software-renderer.v1"
  },
  "gpu_compositor": {
    "status": "experimental",
    "production_claim": false,
    "required_capabilities": [
      "layer_compositing",
      "transforms",
      "clipping",
      "texture_atlas",
      "vsync_frame_timing"
    ],
    "target_host_backend_reports": [],
    "fallback": "software-rgba",
    "same_scene_equivalence": false
  }
}
```

This is intentionally a decision gate, not a GPU implementation plan. The
software renderer remains the production path until performance evidence proves
otherwise.
