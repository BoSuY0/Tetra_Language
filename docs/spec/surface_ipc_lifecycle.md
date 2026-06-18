# Surface IPC And App Lifecycle

Status: experimental production-candidate evidence for the Block System track.
It is not an Electron main/renderer parity claim and does not claim a general
desktop process sandbox.

`tetra.surface.ipc-lifecycle-report.v1` records the app-main, UI isolate,
background service, owned message passing, UI dispatcher, and crash-isolation
evidence needed for Surface apps to cover the app lifecycle responsibilities
that Electron apps often put behind a main/renderer split, without depending on
Electron, React, DOM UI, user JavaScript app logic, or a CSS runtime.

The required quality level is `surface-ipc-lifecycle-v1`.

## Contract

The report is valid only inside the scoped
`PROD_STABLE_SCOPED_LINUX_WEB_APP_UI` release boundary. It must be paired with
the app-model, app-shell, security, package, and target-host reports for the
same commit before any production platform claim can use it.

Required policies:

- `single-owner-ui-dispatcher-v1`: the UI isolate owns Surface handles, frames,
  events, and UI state mutation.
- `owned-message-passing-v1`: actor/task messages carry typed owned data only.
- `surface-boundary-rejection-v1`: Surface handles, frames, and events cannot
  cross actor/task boundaries as message payloads.
- `dispatcher-ui-update-v1`: background tasks and services can update UI only
  by dispatching owned data back to the UI isolate.
- `supervised-background-services-v1`: background service failure records a
  crash report and restart plan without transferring Surface handles.

Required app lifecycle evidence includes launch/start, suspend, and shutdown
steps. Required message evidence includes a positive owned background-to-UI
message and negative cases for Surface handle, Surface frame, Surface event,
borrowed payload, and untyped channel rejection.

## Fake-Claim Rejection

The IPC/lifecycle validator rejects:

- accepted actor/task messages that carry a Surface handle;
- accepted actor/task messages that carry a Surface frame;
- accepted actor/task messages that carry a Surface event;
- accepted borrowed payloads;
- accepted untyped IPC channels;
- background UI mutation without dispatcher routing;
- lifecycle reports without crash-isolation evidence.

These checks are intentionally narrower than a full multiprocess desktop shell.
P27 owns the security sandbox and permission model. P29 owns deeper crash
recovery and diagnostics. P28 only proves the typed IPC and app lifecycle
boundary that keeps Surface UI ownership inside the UI isolate.

## Tetra API

`lib.core.surface` exposes compact IPC/lifecycle helpers:

- `SurfaceLifecyclePolicy`, `lifecycle_policy`, and
  `lifecycle_policy_valid`;
- `SurfaceIPCMessagePolicy`, `ipc_message_policy`, and
  `ipc_message_allowed`;
- `SurfaceUIUpdatePolicy`, `ui_update_policy`, and `ui_update_allowed`;
- `crash_isolation_valid`.

These helpers do not create a framework runtime. They provide report-aligned
policy values that keep app main, UI isolate, background tasks, owned messages,
and dispatcher-routed UI updates explicit in Surface code.
