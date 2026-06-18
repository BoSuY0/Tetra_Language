# Surface App Model

Status: experimental production-candidate evidence for the Block System track.
It is not a broad Electron, React, DOM, or CSS compatibility claim.

`tetra.surface.app-model.v1` records the state, event, command, async, focus,
shortcut, error, and redraw evidence needed for Surface apps to run without a
React runtime. The required quality level is `production-app-model-v1`.

## Contract

The app model is valid only when a Surface runtime report also contains Block
system evidence, Block graph evidence, event traces, state transitions, frame
checksums, and target-host/runtime evidence for the same target.

Required policies:

- `owned-state-store-v1`: stores are owned by the app surface and are not React
  state.
- `typed-command-dispatch-v1`: commands are typed, target existing Blocks, and
  mutate state or request redraw.
- `block-event-trace-v1`: every command is tied to an ordered Block event trace.
- `actor-task-safe-boundary-v1`: async commands cross the actor/task runtime
  only through the safe app-model boundary.
- `navigation-focus-scopes-v1`: navigation uses graph-derived focus scopes and
  focus-trap evidence.
- `scoped-shortcuts-v1`: shortcuts are scoped to global, command palette, and
  editor-shell contexts.
- `command-error-propagation-v1`: command errors are propagated and handled.
- `explicit-redraw-invalidation-v1`: redraws record invalidation, frame order,
  and checksum changes.
- `safe-app-model-boundary-v1`: app state/events do not escape as unsafe
  actor/task payloads.

Required app surfaces are `command_palette`, `dashboard`, `settings`, and
`editor_shell`. The validator rejects app-model evidence that lacks any of
these surfaces.

## Fake-Claim Rejection

The app-model validator rejects:

- missing event traces;
- disabled controls that still dispatch;
- text input delivered to an unfocused Block;
- async commands without the safe actor/task boundary;
- React runtime or React hooks inside the app model;
- DOM event or user-authored script logic as the app event layer.

`ReactRuntimeAbsent`, `ReactHooksAbsent`, `DOMEventsAbsent`, and `UserJSAbsent`
are required evidence flags. These flags mean Surface owns the app state/event
flow for the scoped target; they do not imply drop-in compatibility with React
applications.

## Tetra API

`lib.core.block` exposes compact app-model helpers:

- `AppStateStore`, `app_state_store`, and `app_state_store_valid`;
- `AppCommand`, `app_command`, `app_command_safe`, and
  `app_command_dispatch_status`;
- `AppEventTrace`, `app_event_trace`, and `app_event_trace_valid`;
- `AppModelPolicy`, `app_model_policy_production`, and
  `app_model_policy_valid`;
- `app_async_boundary_safe`, `AppNavigationStep`, `app_navigation_step`,
  `app_navigation_valid`, `app_shortcut_scope_allows`,
  `app_error_propagated_handled`, `AppRedrawRequest`, `app_redraw_request`, and
  `app_redraw_valid`.

These helpers are intentionally smaller than a full framework. They expose the
state/event/command boundary that Surface reports must prove while preserving
Block and Morph as the primary UI architecture.
