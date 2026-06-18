# Native Surface Host v1

Status: implementation contract for the Linux-x64 Wayland native Surface track. This document is not
completion evidence by itself.

## Purpose

`tetra.surface.native-host.v1` proves the direct native Surface runtime path:

```text
Tetra source -> compiled linux-x64 app -> Tetra Surface Host -> Wayland window
-> native events -> Tetra app loop -> app-produced RGBA frames
```

It is stricter than the older `linux-x64-real-window` probe evidence. A probe, viewer, screenshot,
PNG, SVG, HTML page, browser canvas capture, ImageMagick window, or pre-rendered frame file must not
satisfy this contract.

## Launch Contract

The canonical launch command is:

```sh
tetra run --target linux-x64 --surface-host wayland <source.tetra>
```

The stabilized shortcut is the same launch path, not a viewer or screenshot runner:

```sh
tetra surface run <source.tetra>
```

`tetra surface run` must expand to the canonical native-host-required `tetra run` flow with the
same report semantics.

`tetra run` is responsible for:

- building the reported Tetra source as a linux-x64 executable;
- starting `tetra-surface-host-wayland`;
- creating an absolute Unix socket path for host IPC;
- launching the compiled app with host-required environment;
- shutting the host down when the app exits or the window closes;
- reporting host or app failures as failures, not successful previews.

Host-required environment:

```text
TETRA_SURFACE_HOST=wayland
TETRA_SURFACE_HOST_SOCKET=<absolute unix socket path>
TETRA_SURFACE_HOST_REQUIRED=1
TETRA_SURFACE_HOST_PROTOCOL=tetra.surface.host-ipc.v1
```

When `TETRA_SURFACE_HOST_REQUIRED=1` is active, runtime symbols must not fall back to memfd,
synthetic events, browser canvas, or pre-rendered frames.

## IPC Protocol

Protocol name: `tetra.surface.host-ipc.v1`.

All integers are little-endian. The request header is:

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

The response header is:

```text
u32 magic
u32 op
u32 request_id
i32 status       # 0 ok, nonzero runtime/host error
i32 value0
i32 value1
i32 value2
i32 value3
u32 payload_len
```

Operations:

| Op   | Name                       | Payload                    | Response                              |
| ---- | -------------------------- | -------------------------- | ------------------------------------- |
| `1`  | `open`                     | UTF-8 title bytes          | `value0` is a positive surface handle |
| `2`  | `close`                    | none                       | status                                |
| `3`  | `begin_frame`              | none                       | status                                |
| `4`  | `present_rgba`             | RGBA bytes from app memory | status; host records checksum         |
| `5`  | `poll_event_into`          | none                       | payload is 9 `i32` event slots        |
| `6`  | `poll_event_text_into`     | none                       | UTF-8 text payload                    |
| `7`  | `clipboard_write_text`     | UTF-8 bytes                | `value0` is accepted byte count       |
| `8`  | `clipboard_read_text_into` | none                       | UTF-8 clipboard bytes                 |
| `9`  | `poll_composition_into`    | none                       | payload is 4 `i32` slots              |
| `10` | `now_ms`                   | none                       | `value0` is host monotonic ms         |
| `11` | `request_redraw`           | none                       | status; may queue `event_frame`       |

## Runtime Report Section

`tetra.surface.runtime.v1` reports use `host_evidence.level`: `linux-x64-native-surface-host-v1`,
with backend `wayland-surface-host-v1`.

The report must include:

```json
{
  "native_surface_host": {
    "schema": "tetra.surface.native-host.v1",
    "host": "wayland",
    "protocol": "tetra.surface.host-ipc.v1",
    "app_process_kind": "compiled-linux-x64-tetra-app",
    "host_process_kind": "tetra-surface-host-wayland",
    "app_pid": 4242,
    "host_pid": 4243,
    "surface_open_from_app": true,
    "poll_event_from_host": true,
    "present_from_app_rgba": true,
    "app_loop_observed": true,
    "real_window": true,
    "real_close_event": true,
    "real_pointer_event_count": 1,
    "real_key_event_count": 1,
    "presented_frame_count": 2,
    "pre_rendered_frame_source": false,
    "delivery_path": "compiled-tetra-app-to-wayland-surface"
  }
}
```

Frame reports promoted to this level must be presented, non-precomputed, produced by the running
app, and source-linked to the reported `.tetra` source. `artifact_path` must not be a PNG, SVG,
HTML, browser canvas capture, or other pre-rendered UI source.

## Negative Evidence

The native-host validator must reject:

- `--probe-frame` and `real-window-probe`;
- `guest_viewer`, ImageMagick, `display -title`, and viewer commands;
- `.png`, `.svg`, `.html`, `.mjs`, and `.js` frame delivery paths;
- `browser-canvas` and `wasm32-web` delivery paths;
- `pre_rendered_frame_source=true`;
- frames whose producer is not the running app.

## Current Boundary

The older `linux-x64-real-window` level remains valid as probe evidence only. It must not be used to
claim a live native Tetra Surface app.
