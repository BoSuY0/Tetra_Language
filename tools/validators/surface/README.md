# Surface Validator

`tools/validators/surface` validates `tetra.surface.runtime.v1` evidence.

The validator is intentionally strict while Tetra Surface is still planned:
reports must include process evidence, pure-Tetra component abilities including
static hierarchy dispatch, component layout bounds, root-to-child dispatch
paths, host event-buffer dispatch, text-input dispatch with host payload bytes
copied into caller-owned buffers, focus dispatch, and accessibility metadata,
event dispatch, state transitions, presented frames, and deterministic frame
checksums. Process evidence must include a build command for the reported
source and an executable Surface component app process with the expected app
exit. Artifact evidence must include a `component-app` SHA-256 hash and size
linked to that process path; the `validate-surface-runtime` CLI also
recomputes local artifact file size and SHA-256 before accepting a report.
Headless reports must include a hashed `runner-trace` artifact for the
compiler-owned deterministic frame/event trace and a positive `headless actual
runner trace` case; the CLI validates that the trace schema is headless and
that its `source` matches the reported source and every trace frame matches
reported Surface frame evidence.
Linux-x64 reports must include both the small app-presented RGBA readback probe
and a counter component app-presented 320x200 frame read back from the Surface
host memfd. wasm32-web reports must include an actual presented-frame trace
from the compiler-owned Node Surface runner plus a hashed `runner-trace`
artifact; the CLI validates the web runner-trace schema and maps runner frame
orders back to the reported Surface frames only when the trace `wasm_path`
matches the reported `.wasm` component artifact.
Pointer events must hit the reported target component bounds, and component
type evidence must match the reported Tetra source module path. Reports must
include an `artifact_scan` with positive checked-file count, no forbidden
paths, and `pass=true`; the checked-file count must cover at least every
reported artifact, and every reported artifact must live under that scanned
root. It requires a wasm32-web `compiler-owned-loader`
`.mjs` artifact and rejects HTML, legacy `.ui.*`, and non-loader JavaScript
artifact paths. It
rejects legacy metadata-only, sidecar-only, web-only, docs-only, fake, mock,
placeholder, and build-only evidence markers.
