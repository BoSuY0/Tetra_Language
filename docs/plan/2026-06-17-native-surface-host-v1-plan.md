# Native Surface Host v1 Implementation Plan

**Status:** planning document, not completion evidence. **Date:** 2026-06-17. **Owner:** Tetra
Surface native runtime track.

## Goal

Make native Tetra Surface programs work through a direct, honest runtime path:

```text
Tetra source
-> tetra build/run
-> linux-x64 Tetra binary
-> Tetra Surface Host v1
-> real Linux/Wayland window
-> real native events
-> Tetra app loop
-> app-produced RGBA frames
-> live presentation
```

The implementation must not satisfy this goal with screenshots, SVG, PNG, HTML, browser canvas
captures, ImageMagick, pre-rendered frame files, probe viewers, or similar-looking demos.

## Current Facts

- `lib/core/surface/surface.tetra` already exposes `Surface`, `Frame`, `Event`, `open`,
  `poll_event`, `begin_frame`, `present`, `close`, `now_ms`, and `request_redraw`.
- `compiler/internal/runtimeabi/runtimeabi.go` already lists the Surface runtime symbols under
  `RequiredSurfaceSymbols`.
- `compiler/internal/actorsrt/actorsrt_core.go` currently implements Linux-x64 Surface as a starter
  ABI: `surface_open` creates a `memfd`, `surface_present_rgba` writes RGBA bytes to that fd, and
  event polling returns synthetic events.
- `tools/cmd/surface-runtime-smoke/wayland_linux.go` contains a direct Wayland SHM presenter, but it
  is currently probe infrastructure, not the runtime path of a compiled Tetra app.
- `cli/cmd/tetra/tetra_core.go` implements `tetra run` as build plus execute. It does not yet launch
  or supervise a native Surface host.
- `lib/core/surface/draw.tetra` currently renders text through pseudo-glyphs derived from glyph
  index, not actual character glyphs.
- Existing Surface validators already reject some screenshot-only and docs-only evidence, but they
  still allow real-window probe evidence as part of current reports. The new native-app gate must be
  stricter.

## Directness Rules For This Track

- A native Surface app is not a viewer displaying a pre-rendered file.
- A native Surface app is not a browser/HTML/SVG/canvas preview.
- A native Surface app is not a smoke report that only proves a frame can be copied into a separate
  window.
- The app process must be a compiled Tetra linux-x64 binary built from the reported source.
- Frames must originate from `surface.present` calls made by that running app.
- Events must enter the app through `surface.poll_event` from the live native host event queue.
- Any diagnostic helper or compatibility probe must be labelled as diagnostic evidence and must not
  pass the final native Surface app gate.

## Architecture Decision

For v1, implement an official **Tetra Surface Host process** launched by
`tetra run --surface-host wayland`.

This host process is allowed because it is the native runtime host, not a viewer:

- it owns the Wayland connection and real window;
- it exposes a local IPC endpoint to the compiled Tetra binary;
- it receives live `present_rgba` frames from app memory;
- it queues real Wayland input/close/resize events for the app;
- it never accepts PNG/SVG/HTML/screenshots/pre-rendered RGBA files as app UI;
- it exits when the app exits or when the window is closed.

The later v1.1 target can move more host logic into a single standalone binary, but v1 must first
make the direct app-host runtime path real and observable.

## Surface Host IPC v1

Add a documented host-client protocol named `tetra.surface.host-ipc.v1`.

Environment used by `tetra run --surface-host wayland`:

```text
TETRA_SURFACE_HOST=wayland
TETRA_SURFACE_HOST_SOCKET=<absolute unix socket path>
TETRA_SURFACE_HOST_REQUIRED=1
TETRA_SURFACE_HOST_PROTOCOL=tetra.surface.host-ipc.v1
```

Common request header, little-endian:

```text
u32 magic        # 0x31534854, ASCII "TSH1"
u32 op
u32 request_id
u32 handle
i32 width
i32 height
i32 stride
u32 payload_len
```

Common response header, little-endian:

```text
u32 magic        # 0x31534854
u32 op
u32 request_id
i32 status       # 0 means ok; nonzero is a host/runtime error
i32 value0       # handle, copied count, or event kind
i32 value1
i32 value2
i32 value3
u32 payload_len
```

Operations:

| Op   | Name                       | Request payload                           | Response                                                  |
| ---- | -------------------------- | ----------------------------------------- | --------------------------------------------------------- |
| `1`  | `open`                     | UTF-8 title bytes, width/height in header | `value0 = positive handle`                                |
| `2`  | `close`                    | none                                      | `status = 0`                                              |
| `3`  | `begin_frame`              | none                                      | `status = 0`                                              |
| `4`  | `present_rgba`             | raw RGBA bytes from app memory            | `status = 0`, host records checksum/provenance            |
| `5`  | `poll_event_into`          | none                                      | payload is 9 `i32` event slots, or event kind `0`         |
| `6`  | `poll_event_text_into`     | none                                      | payload is UTF-8 text bytes copied from queued text event |
| `7`  | `clipboard_write_text`     | UTF-8 bytes                               | `value0 = bytes accepted`                                 |
| `8`  | `clipboard_read_text_into` | none                                      | payload is UTF-8 clipboard bytes                          |
| `9`  | `poll_composition_into`    | none                                      | payload is 4 `i32` composition slots                      |
| `10` | `now_ms`                   | none                                      | `value0 = monotonic host timestamp ms`                    |
| `11` | `request_redraw`           | none                                      | host queues an `event_frame` when appropriate             |

In `TETRA_SURFACE_HOST_REQUIRED=1` mode, runtime symbols must not silently fall back to
memfd/synthetic events. Host connection failure is a runtime failure for native Surface run mode.

## Task 0 - Freeze Current Truth And Add RED Guards

**Goal:** Prevent another false native completion claim before implementation.

**Files:**

- Modify `tools/validators/surface/surface_runtime_validation.go`.
- Modify `tools/validators/surface/surface_suite_test.go`.
- Modify `tools/cmd/validate-surface-runtime/main.go` only if a new release envelope flag is needed.
- Inspect `tools/cmd/surface-runtime-smoke/surface_smoke_core.go` and
  `tools/cmd/surface-runtime-smoke/surface_smoke_render.go`.

**Approach:**

- Add a validator distinction between current `linux-x64-real-window` evidence and the new native
  app evidence level: `linux-x64-native-surface-host-v1`.
- Add negative fixtures that must fail when native app evidence contains: `--real-window-probe`,
  `--probe-frame`, `guest_viewer`, `.png`, `.svg`, `.html`, `browser-canvas`, `ImageMagick`,
  `pre-rendered`, or a frame path used as the UI source.
- Require new native app reports to include: compiled app process, host process, IPC protocol, app
  pid, host pid, live event counts, app-produced frame count, frame provenance, and no pre-rendered
  file source.

**Verification:**

```sh
GOCACHE="$PWD/.cache/go-build-surface-native-host-plan" \
GOTMPDIR="$PWD/.cache/go-tmp-surface-native-host-plan" \
go test -buildvcs=false ./tools/validators/surface ./tools/cmd/validate-surface-runtime \
  -run 'NativeSurfaceHost|SurfaceRuntime|RealWindow|Screenshot|PreRendered|Probe' \
  -count=1
```

**Done when:**

- New negative tests fail before implementation for the right reason.
- Existing release validators still describe the old real-window probe as probe/evidence only, not
  as final native app proof.

## Task 1 - Specify Native Host Report Evidence

**Goal:** Create the machine-readable proof boundary for the direct path.

**Files:**

- Add `docs/spec/surface/native_surface_host_v1.md`.
- Add or extend report structs in `tools/validators/surface/surface_core.go`.
- Extend validation in `tools/validators/surface/surface_runtime_validation.go`.

**Approach:**

Define a `native_surface_host` section for `tetra.surface.runtime.v1` reports:

```json
{
  "schema": "tetra.surface.native-host.v1",
  "host": "wayland",
  "protocol": "tetra.surface.host-ipc.v1",
  "app_process_kind": "compiled-tetra-linux-x64",
  "host_process_kind": "tetra-surface-host-wayland",
  "surface_open_from_app": true,
  "present_from_app_memory": true,
  "pre_rendered_frame_source": false,
  "real_window": true,
  "real_close_event": true,
  "real_pointer_event_count": 1,
  "real_key_event_count": 1,
  "frame_count": 2,
  "app_loop_observed": true
}
```

Frame reports promoted to this level must include:

- `producer = compiled-tetra-app`;
- `evidence_role = native-surface-live-frame`;
- `source = <reported .tetra source>`;
- checksum of bytes received through `present_rgba`;
- no `probe-frame` or screenshot path as frame origin.

**Verification:**

```sh
GOCACHE="$PWD/.cache/go-build-surface-native-host-report" \
GOTMPDIR="$PWD/.cache/go-tmp-surface-native-host-report" \
go test -buildvcs=false ./tools/validators/surface ./tools/cmd/validate-surface-runtime \
  -run 'NativeSurfaceHost|ValidateReport' \
  -count=1
```

**Done when:**

The validator accepts only reports that prove the direct runtime path and rejects all substitute
delivery paths.

## Task 2 - Build The Wayland Surface Host Process

**Goal:** Add a first-class native runtime host that owns the real Wayland window and speaks
`tetra.surface.host-ipc.v1`.

**Files:**

- Add `tools/cmd/tetra-surface-host/main.go`.
- Add `tools/internal/surfacehost/wayland_linux.go`.
- Add `tools/internal/surfacehost/protocol.go`.
- Add `tools/internal/surfacehost/report.go`.
- Reuse protocol knowledge from `tools/cmd/surface-runtime-smoke/wayland_linux.go` by moving shared
  Wayland helpers into `tools/internal/surfacehost`.

**Approach:**

- `tetra-surface-host --backend wayland --socket <path> --report <path>` starts a Unix socket server
  and connects to the Wayland compositor.
- On `open`, create one `xdg_toplevel` window with the provided title and size.
- On `present_rgba`, copy live frame bytes into Wayland SHM and commit them.
- On Wayland close, queue `surface.event_close()`.
- On pointer events, queue mouse move/down/up events with compositor-provided coordinates.
- On keyboard events, queue key down/up events and text input bytes when the compositor provides
  text.
- On resize/configure, queue `surface.event_resize()` with the configured size.
- Maintain an in-memory event queue per surface handle.
- Write a host-side report with app pid, host pid, event counts, frame counts, frame checksums, and
  source of every frame as `ipc-present-rgba`.
- Remove any runtime path that accepts `--probe-frame` as app UI for this host.

**Verification:**

```sh
GOCACHE="$PWD/.cache/go-build-tetra-surface-host" \
GOTMPDIR="$PWD/.cache/go-tmp-tetra-surface-host" \
go test -buildvcs=false ./tools/internal/surfacehost ./tools/cmd/tetra-surface-host -count=1
```

Manual/local compositor smoke, only when `WAYLAND_DISPLAY` and `XDG_RUNTIME_DIR` are set:

```sh
GOCACHE="$PWD/.cache/go-build-tetra-surface-host-manual" \
GOTMPDIR="$PWD/.cache/go-tmp-tetra-surface-host-manual" \
go run -buildvcs=false ./tools/cmd/tetra-surface-host \
  --backend wayland \
  --socket "$XDG_RUNTIME_DIR/tetra-surface-host-manual.sock" \
  --report reports/surface-native/manual-host.json
```

**Done when:**

The host can create a real Wayland window, present bytes received over IPC, and record real
close/input events without reading a pre-rendered UI file.

## Task 3 - Replace Linux-x64 Surface Runtime Fallback In Host-Required Mode

**Goal:** Make compiled Tetra binaries talk to the official host when run with
`TETRA_SURFACE_HOST_REQUIRED=1`.

**Files:**

- Modify `compiler/internal/actorsrt/actorsrt_core.go`.
- Modify `compiler/internal/actorsrt/actorsrt_suite_test.go`.
- Modify `compiler/internal/runtimeabi/runtimeabi.go` only if a new symbol is needed; prefer keeping
  the existing Surface ABI.

**Approach:**

- Keep existing Surface symbol names so `lib.core.surface` does not need a breaking API change.
- Add Linux-x64 runtime client code for: `socket`, `connect`, `write`, `read`, and `close` against
  `TETRA_SURFACE_HOST_SOCKET`.
- Cache the host connection per app process after first `surface_open`.
- Implement `__tetra_surface_open` as:
  - host-required mode: connect to host, send `open`, return host handle;
  - legacy mode: preserve current memfd starter behavior for old non-promoted smokes until those are
    migrated.
- Implement `present_rgba` by sending the live app memory bytes to the host.
- Implement event/text/clipboard/composition calls by reading host responses.
- Implement `now_ms` using host response in host mode and current runtime path otherwise.
- In host-required mode, any host connection/protocol failure must fail loudly: return nonzero error
  to the app or terminate with a stable runtime diagnostic. It must not fall back to memfd or
  synthetic event records.

**Verification:**

```sh
GOCACHE="$PWD/.cache/go-build-surface-runtime-client" \
GOTMPDIR="$PWD/.cache/go-tmp-surface-runtime-client" \
go test -buildvcs=false ./compiler/internal/actorsrt ./compiler/internal/runtimeabi \
  -run 'Surface|RuntimeExports|HostRequired|Protocol' \
  -count=1
```

**Done when:**

A linux-x64 binary built from a Surface source can send `open`, `present_rgba`, and
`poll_event_into` requests to a test host over Unix socket, and tests prove host-required mode
cannot silently use memfd/synthetic events.

## Task 4 - Add CLI Native Surface Entrypoint

**Goal:** Make the user-facing command launch the real host and the compiled app as one native
Surface run.

**Files:**

- Modify `cli/cmd/tetra/tetra_core.go`.
- Modify `cli/cmd/tetra/tetra_suite_test.go`.
- Add CLI helper code in `cli/cmd/tetra/` if needed.

**Approach:**

- Extend `tetra run` with:

```sh
./tetra run --target linux-x64 --surface-host wayland <source.tetra>
```

- For `--surface-host wayland`:
  - require target `linux-x64`;
  - build the app as today;
  - start `tetra-surface-host --backend wayland` with a private Unix socket;
  - set `TETRA_SURFACE_HOST_*` env vars for the app process;
  - run the compiled Tetra binary;
  - forward interrupt/termination to both app and host;
  - wait for app and host shutdown;
  - surface host errors as CLI errors, not as successful app runs.
- Add `tetra surface run <source>` as a later alias only after the main flag is stable. The first
  implementation should avoid two command surfaces.
- Do not add browser, screenshot, SVG, or image fallback.

**Verification:**

```sh
GOCACHE="$PWD/.cache/go-build-tetra-cli-surface-host" \
GOTMPDIR="$PWD/.cache/go-tmp-tetra-cli-surface-host" \
go test -buildvcs=false ./cli/cmd/tetra \
  -run 'SurfaceHost|RunSurface|RunCommand' \
  -count=1
```

**Done when:**

`tetra run --target linux-x64 --surface-host wayland ...` starts exactly one compiled app process
and one official Wayland Surface host process, with no pre-rendered UI input.

## Task 5 - Convert `surface_window_counter.tetra` To A Live App Loop

**Goal:** Make the first native Surface example live until the user closes the window.

**Files:**

- Modify `examples/surface/runtime/surface_window_counter.tetra`.
- If existing scripted smoke behavior is still needed, add
  `examples/surface/probes/surface_window_counter_scripted_probe.tetra`.
- Update `tools/cmd/surface-runtime-smoke/surface_smoke_core.go` only to point old synthetic checks
  at the scripted probe, not the live app.

**Approach:**

Use this canonical loop shape:

```text
open window
layout app
while !closed:
    drain available events with surface.poll_event
    update app state
    begin_frame
    draw full frame
    present frame
    request_redraw
    core.sleep_ms(16)
close window
```

Event handling rules:

- `event_close` sets `closed = true`.
- `event_resize` updates app width/height and relayouts.
- pointer up inside the button increments the counter.
- key down increments key count or handles reset shortcuts.
- text input stores the visible text length for evidence.
- `event_frame` triggers redraw but does not mutate state.

**Verification:**

```sh
./tetra check examples/surface/runtime/surface_window_counter.tetra

GOCACHE="$PWD/.cache/go-build-surface-window-counter" \
GOTMPDIR="$PWD/.cache/go-tmp-surface-window-counter" \
go test -buildvcs=false ./tools/cmd/surface-runtime-smoke \
  -run 'SurfaceWindowCounter|NativeSurfaceHost|ScriptedProbe' \
  -count=1
```

Manual/local native run:

```sh
./tetra run --target linux-x64 --surface-host wayland \
  examples/surface/runtime/surface_window_counter.tetra
```

**Done when:**

The example remains alive until close, reacts to real pointer/key events, and exits because the
native close event reached Tetra code.

## Task 6 - Implement Real Text Rendering v1

**Goal:** Make `draw.text(ctx, "Count", ...)` render actual character glyphs.

**Files:**

- Modify `lib/core/surface/draw.tetra`.
- Inspect and possibly extend `lib/core/base/strings.tetra`.
- Add focused tests in the existing Surface smoke/render test package that checks frame pixels for
  known words.

**Approach:**

- Replace `glyph_mask_5x7(ctx, x, y, glyph_index, color)` with a character-aware path:
  `glyph_mask_5x7_char(ctx, x, y, ascii_code, color)`.
- Read bytes from the `String` in draw order. If direct `text[i]` access is not accepted by the
  compiler in this module, add a small helper to `lib.core.strings` for ASCII byte access and test
  it first.
- Implement a deterministic 5x7 bitmap fallback font for printable ASCII used by Surface examples:
  letters, digits, space, punctuation required by labels, and replacement box for unsupported UTF-8
  bytes.
- Keep full shaping, bidi, rich text, font fallback, and antialiasing as explicit nonclaims for v1.
- Update text render reports so they distinguish real bitmap glyph rendering from pseudo-glyph
  placeholder rendering.

**Verification:**

```sh
./tetra check examples/surface/runtime/surface_window_counter.tetra

GOCACHE="$PWD/.cache/go-build-surface-text-rendering" \
GOTMPDIR="$PWD/.cache/go-tmp-surface-text-rendering" \
go test -buildvcs=false ./tools/cmd/surface-runtime-smoke ./tools/validators/surface \
  -run 'Text|Glyph|SurfaceWindowCounter|NativeSurfaceHost' \
  -count=1
```

**Done when:**

Pixel tests prove that changing the string from `"Count"` to another word changes glyph shapes
according to character bytes, not string length or glyph index.

## Task 7 - Produce Direct Native Evidence Gate

**Goal:** Add the release-quality gate that proves the direct native path and rejects substitutes.

**Files:**

- Add `scripts/release/surface/surface-linux-x64-native-host-smoke.sh`.
- Modify `tools/cmd/surface-runtime-smoke/surface_smoke_core.go`.
- Modify `tools/cmd/surface-runtime-smoke/surface_smoke_render.go`.
- Modify `tools/cmd/surface-runtime-smoke/surface_smoke_suite_test.go`.
- Modify `tools/cmd/validate-surface-runtime/main.go`.
- Modify `tools/validators/surface/surface_runtime_validation.go`.

**Approach:**

The gate must:

1. Build `examples/surface/runtime/surface_window_counter.tetra` to linux-x64.
2. Launch the official Wayland host.
3. Launch the compiled Tetra binary with host-required env vars.
4. Capture at least two live `present_rgba` frames from app memory.
5. Capture one real close event.
6. Capture at least one real pointer or key event.
7. Write `tetra.surface.runtime.v1` with `native_surface_host`.
8. Validate the report and artifact hashes.

The gate must fail if:

- frame origin is a path to `.png`, `.svg`, `.html`, `.rgba`, or screenshot;
- process path includes `--probe-frame`;
- evidence includes `guest_viewer` or ImageMagick;
- only headless, memfd, docs, browser canvas, or screenshot evidence is present;
- no live host process is recorded;
- no compiled Tetra app process is recorded;
- no real close event reaches the app.

**Verification:**

```sh
GOCACHE="$PWD/.cache/go-build-surface-native-host-gate" \
GOTMPDIR="$PWD/.cache/go-tmp-surface-native-host-gate" \
go test -buildvcs=false ./tools/cmd/surface-runtime-smoke ./tools/cmd/validate-surface-runtime ./tools/validators/surface \
  -run 'NativeSurfaceHost|Reject|PreRendered|Probe|RealClose|RealInput' \
  -count=1
```

Local compositor evidence:

```sh
bash scripts/release/surface/surface-linux-x64-native-host-smoke.sh \
  --report-dir reports/surface-native/linux-x64-native-host-v1

go run ./tools/cmd/validate-surface-runtime \
  --release linux-x64-native-host \
  --report reports/surface-native/linux-x64-native-host-v1/surface-linux-x64-native-host.json
```

**Done when:**

The new gate passes only for a compiled Tetra app talking to the official live Wayland host and
fails for all known substitute paths.

## Task 8 - Update Docs And Nonclaims

**Goal:** Make docs describe the new native path without overclaiming.

**Files:**

- Modify `docs/spec/surface/surface_v1.md`.
- Modify `docs/user/surface/surface_guide.md`.
- Modify `docs/release/surface/surface_v1_release_contract.md`.
- Modify `scripts/release/surface/README.md`.
- Modify claim validators if they contain old wording that treats probe evidence as product-native
  app evidence.

**Approach:**

- Add `linux-x64-native-surface-host-v1` as the stronger native app proof level.
- Keep old `linux-x64-real-window` probe wording as lower-level evidence.
- Document the canonical command:

```sh
./tetra run --target linux-x64 --surface-host wayland \
  examples/surface/runtime/surface_window_counter.tetra
```

- Explicitly state nonclaims: no macOS/Windows Surface production host, no GPU renderer, no native
  widget parity, no Electron API compatibility, no browser DOM app UI, no full rich text, no bidi
  shaping.

**Verification:**

```sh
GOCACHE="$PWD/.cache/go-build-surface-native-docs" \
GOTMPDIR="$PWD/.cache/go-tmp-surface-native-docs" \
go test -buildvcs=false ./tools/cmd/validate-surface-claims ./tools/cmd/verify-docs -count=1
```

If manifest generation is required by the touched docs:

```sh
go run ./tools/cmd/gen-manifest -o docs/generated/manifest.json
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
```

**Done when:**

Docs distinguish direct native app support from old probe evidence and do not claim unsupported
platform/runtime parity.

## Task 9 - Integrated End-To-End Validation

**Goal:** Prove the complete user flow from source to live native app evidence.

**Prerequisites:**

- Linux host with `WAYLAND_DISPLAY` and `XDG_RUNTIME_DIR`.
- No expectation that CI without a compositor can pass the native-host release gate. CI can run
  unit/protocol tests and report native compositor gate as skipped or blocked with an explicit
  reason.

**Commands:**

```sh
./tetra check examples/surface/runtime/surface_window_counter.tetra

GOCACHE="$PWD/.cache/go-build-surface-native-e2e" \
GOTMPDIR="$PWD/.cache/go-tmp-surface-native-e2e" \
go test -buildvcs=false \
  ./compiler/internal/actorsrt \
  ./compiler/internal/runtimeabi \
  ./cli/cmd/tetra \
  ./tools/internal/surfacehost \
  ./tools/cmd/tetra-surface-host \
  ./tools/cmd/surface-runtime-smoke \
  ./tools/cmd/validate-surface-runtime \
  ./tools/validators/surface \
  -run 'Surface|NativeSurfaceHost|Wayland|Glyph|Text|RunCommand' \
  -count=1

bash scripts/release/surface/surface-linux-x64-native-host-smoke.sh \
  --report-dir reports/surface-native/linux-x64-native-host-v1

go run ./tools/cmd/validate-surface-runtime \
  --release linux-x64-native-host \
  --report reports/surface-native/linux-x64-native-host-v1/surface-linux-x64-native-host.json

go run ./tools/cmd/validate-artifact-hashes \
  --write \
  --root reports/surface-native/linux-x64-native-host-v1 \
  --out reports/surface-native/linux-x64-native-host-v1/artifact-hashes.json

go run ./tools/cmd/validate-artifact-hashes \
  --manifest reports/surface-native/linux-x64-native-host-v1/artifact-hashes.json

git diff --check -- \
  compiler/internal/actorsrt \
  compiler/internal/runtimeabi \
  cli/cmd/tetra \
  lib/core/surface \
  lib/core/base/strings.tetra \
  examples/surface/runtime/surface_window_counter.tetra \
  tools/internal/surfacehost \
  tools/cmd/tetra-surface-host \
  tools/cmd/surface-runtime-smoke \
  tools/cmd/validate-surface-runtime \
  tools/validators/surface \
  scripts/release/surface \
  docs/spec/surface \
  docs/user/surface \
  docs/release/surface

graphify update .
```

After Go evidence runs, clean the repo-local Go cache used for this track:

```sh
GOCACHE="$PWD/.cache/go-build-surface-native-e2e" go clean -cache
```

**Done when:**

- The local command opens a live Wayland window from a compiled Tetra source.
- The app stays alive until close.
- Real pointer/key/close events reach Tetra code.
- Frames shown in the window are produced by `surface.present`.
- Text labels are real bitmap glyphs, not placeholder bars or index patterns.
- The validator rejects substitute evidence paths.
- Final status can be `DONE` only if the above end-to-end evidence is present.

## Execution Order

1. Task 0: negative guards.
2. Task 1: report/spec boundary.
3. Task 2: host process.
4. Task 3: linux-x64 runtime client.
5. Task 4: CLI launch path.
6. Task 5: live example loop.
7. Task 6: real text rendering.
8. Task 7: native evidence gate.
9. Task 8: docs/nonclaims.
10. Task 9: end-to-end validation.

Do not start Task 5 as a visual demo before Tasks 2-4 can run a real host path. Do not mark the
track complete from unit tests alone. Completion requires the native run command and the strict
evidence gate.

## Risks And Blockers

- Wayland availability is host-dependent. If no compositor is available, native host end-to-end
  evidence is `BLOCKED`, not replaced with screenshots or a browser preview.
- Emitting Unix socket client logic in the linux-x64 runtime object is lower level than the current
  memfd starter path. Keep protocol tests small and TDD every runtime symbol.
- Current repo worktree is dirty. Implementation must not revert unrelated changes. Start every
  coding batch with `git status --short` and inspect touched files before editing.
- Current real-window smoke reports use probe evidence. The new gate must not delete useful
  historical probes, but it must prevent them from satisfying the stronger native app claim.
- Full text shaping is out of scope for v1. The required v1 outcome is real deterministic bitmap
  glyph rendering for visible ASCII labels plus explicit fallback for unsupported bytes.

## Final Acceptance Criteria

The track is complete only when this exact flow works:

```sh
./tetra run --target linux-x64 --surface-host wayland \
  examples/surface/runtime/surface_window_counter.tetra
```

Acceptance:

- a real Wayland window opens;
- the window is driven by the compiled Tetra app, not by a viewer;
- the app remains alive until the window is closed;
- clicking or pressing a key updates app state through `surface.poll_event`;
- text labels render as actual glyphs;
- `surface.present` frames are visible in the native window;
- a strict report validates compiled source, live host, real close event, real pointer/key event,
  app-produced frames, and no pre-rendered image path;
- final report status is `DONE` only after end-to-end evidence passes.
