# Surface Host ABI and App Shell Evidence

This document records the P15 app-shell evidence boundary for the experimental
`ui.surface-block-system` track. It does not promote broad desktop production
support by itself; target-host production promotion remains gated by later
linux-x64, Windows, macOS, web, accessibility, packaging, security, IPC, crash,
performance, CI, audit, and comparison packets.

## Report Schema

Block System runtime reports that claim the app shell layer must include:

- `app_shell.schema = tetra.surface.app-shell.v1`
- `app_shell.level = production-app-shell-host-abi-v1`
- `app_shell.host_abi = tetra.surface.host-abi.v1`
- `app_shell.shell_policy = block-app-shell-host-abi-v1`

The same report must still include `block_system`, `app_model`, and
`keyboard_ux` evidence. App shell evidence without Block System, app model, and
keyboard UX evidence is incomplete.

## Required Capability Matrix

The capability matrix covers:

- windows and lifecycle
- menus and context menus
- dialogs and file pickers
- tray/status items
- notifications
- cursors
- drag/drop
- permissions
- clipboard and IME
- DPI/scale
- open URL and open file requests

Each supported capability requires a target-host action trace. Unsupported
capabilities require rejected diagnostics with `silent_noop:false`.

## Fake Claim Rejections

The validator rejects:

- menu support claimed without target-host action traces
- notification support claimed without delivered host reports
- unsupported host features that silently no-op
- window evidence without lifecycle evidence
- permission denial without a diagnostic
- platform-native widget shell delegation

These guards keep the P15 schema honest: the app shell may define a production
ABI evidence level, but it cannot silently substitute Electron, React, platform
widgets, or unreported host behavior.

## Standard Library Surface

`lib.core.surface` exposes compact value types for app shell ABI authoring and
validation:

- `ShellWindowSpec`
- `ShellCapability`
- `ShellActionTrace`
- `ShellPermission`
- `ShellDiagnostic`
- `ShellDPI`

The helpers are pure data helpers. Real target-host execution remains evidence
from runtime reports and target-specific adapters.
