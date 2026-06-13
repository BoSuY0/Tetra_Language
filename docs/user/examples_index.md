# Examples Index

Status: release-covered examples index. The current support boundary is
`docs/spec/current_supported_surface.md`. Validate with:

```sh
./tetra smoke --list --format=json > reports/smoke-list-linux-x64.json
go run ./tools/cmd/validate-example-index --smoke-list reports/smoke-list-linux-x64.json --index docs/user/examples_index.md
```

## Generated Docs Naming Policy

Generated docs may show examples with two spellings. If an example source file
declares `module ...`, generated docs render its dotted module path, such as
`examples.core_math_smoke`. If an example source file has no module declaration,
generated docs render its portable file path, such as `examples/flow_hello.tetra`.

This index always lists repository file paths under `examples/` so smoke-list
validation and release evidence stay portable. When comparing this index with
generated docs, map dotted `examples.*` module names back to their source files
before treating the rendering difference as drift.

## Surface Claim Tier Notes

Surface examples use the claim tiers from
`docs/spec/current_supported_surface.md`: `PROD_STABLE_SCOPED`,
`BETA_TARGET_HOST`, `EXPERIMENTAL`, `UNSUPPORTED`, and `NONCLAIM`. In this
index, release-supported examples are current only inside the bounded
`surface-v1-linux-web` evidence scope, experimental Block/Morph entries remain
evidence tracks, and unsupported targets/features must stay explicit
nonclaims. The product gate command is
`bash scripts/release/surface/product-gate.sh --report-dir reports/surface-product-v1`,
but that gate is not the P29 final `PROD_STABLE_SCOPED` verdict and does not
create broad Electron, React, CSS, DOM, Windows, macOS, GPU, rich-text, bidi,
or full screen-reader claims.

## Release-Supported Surface Examples

- `examples/surface_release_counter.tetra`: release counter/input evidence for
  linux-x64 real-window and wasm32-web browser-canvas host validation.
- `examples/surface_release_form.tetra`: scoped `production-widgets-v1` release
  subset form evidence for the bounded linux-x64/web Surface release scope; it
  is not a platform-native widget or future core primitive claim.
- `examples/surface_release_text_input.tetra`: text/input, invalid UTF-8,
  multiline storage, selection copy/paste, clipboard, and IME/composition
  baseline evidence.
- `examples/surface_release_accessibility.tetra`: accessibility metadata plus
  platform bridge evidence for supported Linux and web targets.
- `examples/surface_linux_app_shell_notes.tetra`: scoped Linux app-shell notes
  reference for lifecycle, multi-window, resize/DPI/cursor, clipboard, IME,
  accessibility, and blocked-pass shell-feature evidence without native widgets.

## Experimental Legacy Surface Evidence

- `examples/surface_counter.tetra`
- `examples/surface_toolkit_form.tetra`
- `examples/surface_toolkit_settings.tetra`
- `examples/surface_accessibility_settings.tetra`

These examples remain useful regression evidence, but the public release docs
point new work to `ui.surface-toolkit-v1`, `ui.surface-text-input-v1`, and
`ui.surface-accessibility-v1`.

## Experimental Block-First Beauty Examples

These examples are P15-P18 evidence for polished UI built from `Block`
configuration only. The complete Block-system gate is
`scripts/release/surface/block-system-gate.sh --report-dir reports/surface-block/p18-budget`;
it writes same-commit headless,
linux-x64 real-window, and wasm32-web browser-canvas Block reports plus
artifact hashes and `block_system.memory_budget` evidence. The headless smoke
also writes `surface-block-examples.json`. These examples do not introduce core
Button/Card/TextField/Sidebar/Modal abstractions and do not promote Block to
production support.

- `examples/surface_block_command_palette.tetra`: command palette overlay,
  editable query field, and command rows using Block layout, layered paint,
  text, assets, accessibility, state selectors, motion, and scene checksum
  evidence.
- `examples/surface_block_project_dashboard.tetra`: sidebar-like shell, metric
  panels, activity card, and action affordance built as Block configurations.
- `examples/surface_block_settings.tetra`: settings form with labels, editable
  fields, save/reset actions, focus order, and label relationships through
  Block metadata.
- `examples/surface_block_editor_shell.tetra`: editor shell with rail, tabs,
  scrollable code panel, selected line styling, and deterministic state/motion
  evidence.
- `examples/surface_block_glass_panel.tetra`: glass overlay/control panel with
  image/icon assets, overlay paint, rounded capsules, focus order, and
  motion-backed interaction evidence.

## Experimental Morph Capsule Examples

The `ui.surface-morph-capsule` P08 evidence set imports `lib.core.morph`, uses
capsule recipes, records recipe expansions, and then validates the resulting
`BlockTree`:

- `examples/surface_morph_command_palette.tetra`
- `examples/surface_morph_project_dashboard.tetra`
- `examples/surface_morph_settings.tetra`
- `examples/surface_morph_editor_shell.tetra`
- `examples/surface_morph_glass_panel.tetra`

The gate is
`scripts/release/surface/morph-gate.sh --report-dir reports/surface-morph/gate`.
It writes `tetra.surface.morph.v1` headless evidence plus a
`tetra.surface.morph.gate.v1` summary. These examples are not Surface v1
production support and do not introduce core Button/Card/TextField/Sidebar
or Modal primitives.

## Surface Project Templates

`tetra new surface-app --template <kind>` generates onboarding projects for:

- `command-palette`
- `settings`
- `dashboard`
- `editor-shell`
- `multi-window-notes`
- `web-canvas`

The generated sources use `lib.core.surface`, `lib.core.block`, and
`lib.core.morph`; the notes template also uses the scoped
`lib.core.surface_app_shell` helpers. The template smoke gate is
`scripts/release/surface/surface-template-smoke.sh --report-dir reports/surface-templates/gate`.
It writes `tetra.surface.template-smoke.v1` /
`surface-template-smoke-v1` evidence for generation, check, build, run,
inspection, visual diff, and packaging.

## Surface Reference App Suite

The `ui.surface-reference-app-suite-v1` gate proves practical product shapes
with Block/Morph authoring over Block. Run it with:

```sh
bash scripts/release/surface/surface-reference-apps-smoke.sh \
  --report-dir reports/surface-reference-apps/gate
```

It writes `tetra.surface.reference-app-suite.v1` /
`surface-reference-app-suite-v1` plus `tetra.surface.visual-regression.v1`
evidence for headless, linux-x64 real-window, and wasm32-web browser-canvas
targets. Every app is checked, built, run, visually diffed, and recorded with
token/theme, layout, interaction, accessibility, performance, and artifact-hash
evidence. `lib.core.widgets` is allowed only in the migration compatibility
example.

- `examples/surface_reference_command_palette.tetra`
- `examples/surface_reference_settings.tetra`
- `examples/surface_reference_dashboard.tetra`
- `examples/surface_reference_editor_shell.tetra`
- `examples/surface_reference_file_manager.tetra`
- `examples/surface_reference_dialog_notification.tetra`
- `examples/surface_reference_localized_form.tetra`
- `examples/surface_reference_accessibility_form.tetra`
- `examples/surface_reference_multi_window_notes.tetra`
- `examples/surface_reference_migration.tetra`

## Legacy Metadata UI Examples

Legacy `tetra.ui.v1` metadata examples are compatibility evidence for
`ui.metadata-v1`, not Surface v1 release evidence.

| Example | Purpose | Target group | Expected behavior |
| --- | --- | --- | --- |
| `examples/hello.tetra` | Minimal legacy hello-world program. | wasm | build-only exits 0 contract (excluded from native smoke profile) |
| `examples/islands_hello.tetra` | Minimal island program. | native | exits 0 |
| `examples/islands_i32.tetra` | Island integer access. | native | exits 55 |
| `examples/islands_overflow.tetra` | Island bounds diagnostic smoke. | native | exits 1 |
| `examples/islands_double_free.tetra` | Island debug double-free diagnostic smoke. | native debug-only | exits 2 with `--islands-debug`; excluded from normal run smoke |
| `examples/mmio_smoke.tetra` | MMIO builtin smoke. | native | exits 123 |
| `examples/cap_mem_smoke.tetra` | Memory capability smoke. | native | exits 77 |
| `examples/cap_mem_ptr_smoke.tetra` | Pointer load/store through `cap.mem`. | native | exits 77 |
| `examples/memset_smoke.tetra` | Memory set helper smoke. | native | exits 88 |
| `examples/actors_pingpong.tetra` | Actor ping-pong runtime smoke. | native | exits 0 |
| `examples/actor_sleep_pingpong.tetra` | Actor timer wake smoke. | native | exits 0 |
| `examples/actors_decl_spawn.tetra` | Actor declaration spawn target smoke. | native | exits 0 |
| `examples/actors_tagged_stress.tetra` | Tagged actor message stress smoke. | native | exits 0 |
| `examples/flow_hello.tetra` | Minimal canonical Flow program. | native | exits 0 |
| `examples/flow_struct_smoke.tetra` | Flow struct syntax and field access. | native | exits 42 |
| `examples/flow_islands_smoke.tetra` | Flow syntax with islands. | native | exits 0 |
| `examples/flow_unsafe_cap_mem_smoke.tetra` | Flow unsafe capability memory path. | native | exits 42 |
| `examples/flow_grammar_surface_smoke.tetra` | Broad Flow grammar surface and test-block smoke. | native | exits 128 in linux/amd64 compiler evidence; `tetra test` block passes |
| `examples/ui_native_shell_smoke.tetra` | UI metadata native shell smoke. | native | exits 0 |
| `examples/bool_smoke.tetra` | Boolean branch smoke. | native | exits 42 |
| `examples/for_range_smoke.tetra` | Range loop smoke. | native | exits 55 |
| `examples/for_collection_smoke.tetra` | Collection loop smoke. | native | exits 42 |
| `examples/for_collection_u8_smoke.tetra` | Byte collection loop smoke. | native | exits 42 |
| `examples/loop_control_smoke.tetra` | Break and continue control flow. | native | exits 42 |
| `examples/complex_control_flow_smoke.tetra` | Nested control flow coverage. | native | exits 42 |
| `examples/unary_not_smoke.tetra` | Unary boolean negation. | native | exits 42 |
| `examples/const_smoke.tetra` | Global const expression smoke. | native | exits 42 |
| `examples/const_bool_smoke.tetra` | Boolean constant smoke. | native | exits 42 |
| `examples/local_const_smoke.tetra` | Local const binding smoke. | native | exits 42 |
| `examples/globals_smoke.tetra` | Top-level `var`/`val` global storage smoke. | native | exits 49 |
| `examples/compound_assignment_smoke.tetra` | Compound assignment smoke. | native | exits 42 |
| `examples/else_if_smoke.tetra` | Else-if lowering smoke. | native | exits 42 |
| `examples/enum_match_smoke.tetra` | Enum match smoke. | native | exits 42 |
| `examples/enum_exhaustive_match_smoke.tetra` | Exhaustive enum match smoke. | native | exits 42 |
| `examples/enum_payload_smoke.tetra` | Enum payload constructor and match smoke. | native | exits 42 |
| `examples/effects_io_smoke.tetra` | IO effect declaration smoke. | native wasm | exits 0 |
| `examples/effects_mem_smoke.tetra` | Memory effect declaration smoke. | native | exits 17 |
| `examples/effects_actors_smoke.tetra` | Actor effect declaration smoke. | native | exits 0 |
| `examples/optional_smoke.tetra` | Optional value smoke. | native | exits 42 |
| `examples/optional_match_smoke.tetra` | Optional none match smoke. | native | exits 42 |
| `examples/optional_match_some_smoke.tetra` | Optional some match smoke. | native | exits 42 |
| `examples/ownership_smoke.tetra` | Ownership transfer and optional borrow smoke. | native | exits 42 |
| `examples/safe_view_borrow_return.tetra` | Safe View Lifetime Contracts v1 borrowed slice return smoke. | native | exits 0 after using a borrowed `[]u8` view locally without allocation |
| `examples/safe_view_string_borrow_return.tetra` | Safe View Lifetime Contracts v1 borrowed String return smoke. | native | exits 0 after checking the borrowed byte-window contents |
| `examples/safe_view_copy_escape.tetra` | Safe View Lifetime Contracts v1 copy escape smoke. | native | exits 0 after copying a borrowed returned view into owned storage |
| `examples/safe_view_actor_copy_boundary.tetra` | Safe View Lifetime Contracts v1 actor boundary copy smoke. | native | exits 0 after sending a copied byte view across the typed actor boundary |
| `examples/safe_view_task_copy_boundary.tetra` | Safe View Lifetime Contracts v1 task boundary smoke for the current typed-task transfer surface. | native | exits 0; direct task payload transfer is not exposed yet, so the borrowed view is copied before the typed task boundary path |
| `examples/safe_view_aggregate_copy_escape.tetra` | Safe View Lifetime Contracts v1 aggregate copy escape smoke. | native | exits 0 after returning a struct that contains an owned copy rather than a borrowed view |
| `examples/typed_errors_smoke.tetra` | Typed error syntax smoke. | native | exits 42 |
| `examples/async_smoke.tetra` | Async and await smoke. | native | exits 42 |
| `examples/task_smoke.tetra` | Task runtime handle smoke. | native | exits 42 |
| `examples/time_sleep_smoke.tetra` | Logical runtime sleep/deadline smoke. | native | exits 0 |
| `examples/task_sleep_deadline_smoke.tetra` | Task sleep deadline ordering smoke. | native | exits 0 |
| `examples/task_join_wait_smoke.tetra` | Task join waiter wake smoke. | native | exits 5 |
| `examples/task_group_cancel_smoke.tetra` | Task group cancellation wakes a sleeping child before its timer and returns cancellation error. | native | exits 1 |
| `examples/task_group_lifecycle_smoke.tetra` | Task group open, spawn/join, close, status, and canceled-state lifecycle smoke. | native | exits 42 |
| `examples/task_bounded_stress.tetra` | Bounded cooperative task spawn/join stress smoke. | native | exits 42 |
| `examples/deadline_aware_waits_smoke.tetra` | Deadline-aware sleep, task join, and actor receive smoke. | native | exits 0 |
| `examples/wait_composition_smoke.tetra` | Poll, yield, timer-ready, tagged receive deadline, and task/timer select smoke. | native | exits 0 |
| `examples/ctx_switch_sysv_smoke.tetra` | `core.ctx_switch` SysV x64 stack-switch smoke. | native linux-x64 macos-x64 | exits 66 |
| `examples/ctx_switch_win64_smoke.tetra` | `core.ctx_switch` Win64 stack-switch smoke. | native windows-x64 | exits 66; excluded from linux-x64 smoke profile by target |
| `examples/core_async_smoke.tetra` | Current core async helper smoke for `select_or`, with `pair_sum` probe coverage kept compile-visible. | native | exits 42 through the deterministic `select_or` path; does not claim broader async runtime coverage |
| `examples/core_accessibility_smoke.tetra` | Experimental Tetra Surface accessibility metadata smoke for role, action, value, and validation helper counts through `lib.core.accessibility`. | native | exits 42 through pure metadata helper calls; does not claim production accessibility tree runtime support |
| `examples/core_capability_smoke.tetra` | Current core capability token acquisition smoke for `cap.mem` and `cap.io`. | native | exits 42 using only caller-owned heap memory and local MMIO storage; does not imply host permission grant |
| `examples/core_block_smoke.tetra` | Current Block primitive data model smoke for fixed layout, paint, text, input, event, state, motion, accessibility, and asset metadata through `lib.core.block`. | native | exits 0 |
| `examples/core_surface_app_smoke.tetra` | Surface app command/reducer helper smoke for caller-owned state, event bindings, navigation, focus scope, async, and undo/redo through `lib.core.surface_app`. | native | exits 42 |
| `examples/core_surface_app_shell_smoke.tetra` | Scoped Linux Surface app-shell helper smoke for explicit window lifecycle, resize/DPI/cursor state, `electron-feature-ledger-v1`, `surface-security-permission-v1` default-deny policy evidence, `surface-performance-budget-v1` local budget evidence, honest blocked-pass features, target-evidenced rows, and scoped adapter feature checks through `lib.core.surface_app_shell`. | native | exits 42 through pure helper calls; does not claim GTK/Qt/native widget UI, Electron/React runtime, DOM UI, user JavaScript app logic, unrestricted filesystem/network access, tray/notification/file-picker support, external benchmark results, unsupported Electron speed comparisons, or all-platform app-shell support |
| `examples/core_morph_smoke.tetra` | Experimental Morph Capsule helper smoke for `lib.core.morph` recipes and Block expansion checks. | native | exits 42 through pure Morph self-checks; does not claim Surface v1 production support |
| `examples/surface_app_model.tetra` | Headless Surface app-model reference exercising explicit command dispatch, navigation underflow rejection, focus modal trap, async completion/cancel boundary, and undo/redo without React hooks or DOM event state. | native headless | exits 1 under `surface-runtime-smoke --mode headless-app-model`; release evidence is currently scoped to the P11 headless app-model report |
| `examples/surface_linux_app_shell_notes.tetra` | Linux app-shell notes reference exercising window lifecycle, two-window notes state, resize/DPI/cursor changes, clipboard/IME/accessibility adapter evidence, `electron-feature-ledger-v1`, `surface-security-permission-v1` default-deny permission evidence, `surface-performance-budget-v1` startup/frame/memory/cache/framebuffer/binary-size/CPU-proxy evidence, scoped crash/error reporting adapters, and blocked-pass dialog/file picker/notification/tray nonclaims through `lib.core.surface_app_shell`. | native linux-x64 target-host | exits 1 under `surface-runtime-smoke --mode linux-x64-release-app-shell`; release evidence is scoped to `tetra.surface.linux-app-shell.v1` and rejects GTK/Qt/native widget UI substitutes, unrestricted filesystem/network access, remote asset fetching, external benchmark results, and unsupported Electron speed comparisons |
| `examples/surface_morph_command_palette.tetra` | Experimental Morph Capsule command palette over `lib.core.morph`, expanding capsule recipes into Block graph evidence. | native headless | exits 1 under the existing Morph runtime-smoke fixture; release evidence comes from `surface-headless-morph-smoke.sh` validating recipe expansions, BlockTree evidence, accessibility projection, and memory-budget checks |
| `examples/surface_morph_project_dashboard.tetra` | Experimental Morph Capsule dashboard reference app using region, metric, list row, and toast recipes over Block. | native | exits 0 through recipe expansion and BlockTree validation |
| `examples/surface_morph_settings.tetra` | Experimental Morph Capsule settings reference app using form field, field text, tab, and action recipes over Block. | native | exits 0 through recipe expansion and BlockTree validation |
| `examples/surface_morph_editor_shell.tetra` | Experimental Morph Capsule editor shell reference app using navigation, tab, command item, and region recipes over Block. | native | exits 0 through recipe expansion and BlockTree validation |
| `examples/surface_morph_glass_panel.tetra` | Experimental Morph Capsule dialog/glass panel reference app using dialog, toast, action, and region recipes over Block. | native | exits 0 through recipe expansion and BlockTree validation |
| `examples/core_collections_smoke.tetra` | Current core collections smoke for stable generic `Vec<T>`/`HashMap<K,V>` source views plus legacy `[]i32` length, contains, count, and first-or helpers. | native | exits 42 |
| `examples/core_component_smoke.tetra` | Planned Tetra Surface static component helper smoke for `clamp_size`, `inset_rect`, and `center_rect`. | native | exits 42 through pure measurement/layout helpers; does not claim dynamic component lists or runtime widget dispatch |
| `examples/core_widgets_smoke.tetra` | Experimental Tetra Surface minimal widget helper smoke for `Panel` initialization and content bounds through `lib.core.widgets`. | native | exits 42 through pure widget helper calls; does not claim production widget toolkit support |
| `examples/core_crypto_smoke.tetra` | Current core crypto placeholder smoke for checksum, seed mixing, and equality branches. | native | exits 42; placeholder helpers are not cryptographic primitives |
| `examples/core_draw_smoke.tetra` | Planned Tetra Surface software draw helper smoke for RGBA clear, rectangles, outlines, and text markers. | native linux-x64 | exits 42 through the starter scalar Surface host ABI; full `tetra.surface.runtime.v1` frame/event/checksum validation remains future |
| `examples/core_style_smoke.tetra` | Stable Surface v1 widget style and theme helper smoke for default themes and focused state colors through `lib.core.style`. | native | exits 42 through pure style helper calls; does not claim production widget toolkit support |
| `examples/core_filesystem_smoke.tetra` | Current core filesystem placeholder smoke for path-string helper behavior. | native | exits 42; does not perform host filesystem access |
| `examples/core_http_smoke.tetra` | Current v0.4.0 core HTTP/1.1 String and byte-buffer request-line routing, request-head framing, and response byte-buffer helper smoke for TechEmpower paths. | native | exits 42 using caller-owned heap memory; does not open sockets, parse full request bodies, or talk to PostgreSQL |
| `examples/core_io_smoke.tetra` | Current core IO capability/MMIO helper smoke. | native | exits 42 using caller-owned local MMIO storage; does not imply host IO permission grant |
| `examples/core_json_smoke.tetra` | Current v0.4.0 core JSON byte-buffer helper smoke for compact response object writing and escaping. | native | exits 42 using caller-owned heap memory; does not perform HTTP or network IO |
| `examples/core_math_smoke.tetra` | Current core math module smoke for `add_i32`, `min_i32`, `max_i32`, and `clamp_i32`. | native | exits 42 |
| `examples/core_memory_smoke.tetra` | Current core memory module smoke for capability-bound `memset_u8` and `memcpy_u8`. | native | exits 42 |
| `examples/core_memory_negative_length_smoke.tetra` | Current core memory negative-length diagnostic smoke for capability-bound `memset_u8` and `memcpy_u8`. | native | exits 2 when both helpers reject negative lengths |
| `examples/core_net_smoke.tetra` | Current v0.4.0 core networking runtime smoke for real linux-x64 TCP socket open, nonblocking mode, `SO_REUSEPORT`, `TCP_NODELAY`, loopback bind/listen, epoll create/add-read/add-read-write/mod-read-write/mod-read/delete/wait-zero/wait-one-into-zero, fd/flag extraction, event predicates, and close helpers; compiler integration separately covers loopback connect plus read/recv/write/send payload exchange. | native linux-x64 | exits 42; does not accept clients, read/write payloads, run a full event-loop abstraction, or talk to PostgreSQL |
| `examples/core_networking_smoke.tetra` | Current core networking placeholder smoke for port and retry-backoff helpers. | native | exits 42; does not perform network IO |
| `examples/core_postgres_smoke.tetra` | Current v0.4.0 core PostgreSQL wire-frame byte-buffer helper smoke for startup, Simple Query, Terminate, and big-endian length fields. | native | exits 42 using caller-owned heap memory; does not open sockets, authenticate, parse server frames, or pool connections |
| `examples/core_postgres_prepared_smoke.tetra` | Current v0.4.0 core PostgreSQL prepared-statement wire-frame smoke for Parse, Bind, Describe, Execute, Sync, one- and two-parameter text binds, and i16/i32 length fields. | native | exits 42 using caller-owned heap memory; does not open sockets, authenticate, parse server frames, manage prepared statement state, or pool connections |
| `examples/core_postgres_result_smoke.tetra` | Current v0.4.0 core PostgreSQL result-frame smoke for typed frame headers, RowDescription type OIDs, DataRow value offsets/lengths, ASCII integer values, CommandComplete affected rows, and ReadyForQuery status bytes. | native | exits 42 using caller-owned heap memory; does not open sockets, authenticate, own connection state, manage prepared statements, or pool connections |
| `examples/core_serialization_smoke.tetra` | Current core serialization helper smoke for byte-pair packing and checksum behavior. | native | exits 42 |
| `examples/core_slices_smoke.tetra` | Current core slices helper smoke for `sum_i32`, `weighted_sum_i32`, and `sum_u8`. | native | exits 42 |
| `examples/core_strings_smoke.tetra` | Current core strings helper smoke for `ascii_len`, `ascii_sum`, and `is_empty`. | native | exits 42 |
| `examples/core_text_smoke.tetra` | Surface v1 core text buffer helper smoke for caller-owned UTF-8 byte storage, caret movement, selection copy, paste, invalid UTF-8 rejection, multiline byte storage, and ABI-friendly composition lifecycle helpers. | native | exits 42 using caller-owned heap memory; does not claim full platform IME, rich text, bidi shaping, grapheme-cluster caret movement, or platform text-input host evidence |
| `examples/core_i18n_smoke.tetra` | Surface v1 bounded i18n helper smoke for locale selection, string-table fallback, missing-key diagnostics, deterministic formatting hooks, and RTL placeholder nonclaims through `lib.core.i18n`. | native | exits 42; does not claim full ICU, full bidi shaping, RTL production text layout, or platform locale dependency |
| `examples/core_surface_smoke.tetra` | Planned Tetra Surface core type-contract smoke for `Size`, `Rect`, and host/frame/event wrapper visibility. | native linux-x64 | exits 42 through the starter scalar Surface host ABI; full `tetra.surface.runtime.v1` frame/event/checksum validation remains future |
| `examples/core_sync_smoke.tetra` | Current core sync helper smoke for status merge, countdown, barrier target, and readiness behavior. | native | exits 42 |
| `examples/core_testing_smoke.tetra` | Current core testing helper smoke for assertion status composition. | native | exits 42 |
| `examples/core_time_smoke.tetra` | Current core time helper smoke for deterministic duration arithmetic. | native | exits 42; does not claim wall-clock runtime behavior |
| `examples/experimental_math_smoke.tetra` | Experimental stdlib math mirror smoke; evidence only, not a stable support claim. | native | experimental evidence only; Excluded from linux-x64 smoke profile; exits 42 in linux/amd64 compiler test evidence |
| `examples/experimental_memcpy_smoke.tetra` | Experimental stdlib memory mirror memcpy/memset smoke; evidence only, not a stable support claim. | native | experimental evidence only; Excluded from linux-x64 smoke profile; exits 93 in linux/amd64 compiler test evidence |
| `examples/extension_smoke.tetra` | Extension method smoke. | native | exits 42 |
| `examples/generic_smoke.tetra` | Generic function smoke. | native | exits 42 |
| `examples/generic_struct_smoke.tetra` | Experimental generic struct smoke; evidence only, not a stable support claim. | native | exits 42 when experimental path is covered |
| `examples/struct_ctor_smoke.tetra` | Call-style struct constructor smoke. | native | exits 94 |
| `examples/protocol_impl_smoke.tetra` | Protocol implementation smoke. | native | exits 42 |
| `examples/surface_counter.tetra` | Experimental pure-Tetra Surface counter component smoke using ordinary structs, static `measure`/`layout`/`draw`/`event`/`focus`/`text`/`accessibility` abilities, parent/child hierarchy, component layout bounds, draw helpers, event-buffer records, host text payload copy, and event handling. | native linux-x64 headless wasm32-web | exits 1 through the starter Surface host ABI after consuming a host-provided event buffer dispatched through the `CounterApp` to `CounterButton` `dispatch_path`, copying host text bytes into a Tetra `[]u8`, handling `event_text_input`, and presenting distinct pre/post state frames; `scripts/release/surface/surface-headless-smoke.sh` emits `tetra.surface.runtime.v1` headless frame/event/checksum plus static component ability, bounds-checked hierarchy-dispatch, event-buffer, text-dispatch, host-text-payload, focus-dispatch, accessibility-metadata, host-event, pre/post frame, and no-legacy-sidecar evidence; `scripts/release/surface/surface-linux-x64-smoke.sh` adds kernel-backed Host ABI probe evidence, no-legacy-sidecar scanning, and app-presented RGBA checksum evidence; `scripts/release/surface/surface-wasm32-web-smoke.sh` builds/runs wasm32-web through compiler-owned `tetra_surface_host_v1` imports, strict import validation, compiler-owned loader evidence, and no legacy UI sidecars |
| `examples/surface_browser_counter.tetra` | Pure-Tetra Surface browser-canvas counter/input app using ordinary structs, draw helpers, browser input records, resize layout, text payload handling, and RGBA presentation. | wasm32-web browser-canvas | exits 1 after opening a browser Surface, presenting a 320x200 frame, consuming pointer/key/resize/text events through `tetra_surface_host_v1`, updating Tetra-owned count/key/layout/text state, presenting a 400x240 frame, and closing; `scripts/release/surface/surface-wasm32-web-browser-canvas-smoke.sh` emits `tetra.surface.runtime.v1` evidence with `host_evidence.level = wasm32-web-browser-canvas-input`, real Chromium-compatible canvas open/readback, matching source/canvas RGBA checksums, browser-native pointer/key/resize/beforeinput events, no legacy UI sidecars, and validator rejection of Node-only/DOM-only/user-JS/metadata/fake/stale evidence for browser canvas promotion |
| `examples/surface_window_counter.tetra` | Pure-Tetra Surface real-window counter app using ordinary structs, draw helpers, event-buffer records, key handling, resize layout, small text payload handling, and close handling. | native linux-x64-real-window | exits 1 after opening a Surface, drawing a counter/button frame, consuming click/key/resize/text/close events, presenting an updated frame, and closing cleanly; `scripts/release/surface/surface-linux-x64-real-window-smoke.sh` emits `tetra.surface.runtime.v1` evidence with `host_evidence.level = linux-x64-real-window`, a Wayland shm RGBA real-window probe, frame order 5 at 400x240, no legacy UI sidecars, and validator rejection of headless/memfd/docs/build/metadata/fake/legacy DOM evidence for real-window promotion |
| `examples/surface_component_counter.tetra` | Static Tetra Surface component-model fixture with nested ordinary structs. | native surface-component | semantic `CheckWorld` fixture for `measure`, `layout`, `draw`, `event`, `focus`, `text_input`, and `accessibility_role` extension methods using `lib.core.component`; not Linux-x64 real-window or wasm promotion evidence |
| `examples/surface_text_input.tetra` | Pure-Tetra Surface TextBox component fixture with caller-owned host text bytes copied into component-owned `[]u8` storage. | native surface-component | exits 42 after the Linux-x64 Surface Host ABI reports deterministic `OK` text bytes, the ordinary `TextBox` struct stores and validates them without built-in widget magic, and the app presents a Surface frame; this is below the stricter `production-text-input-v1` gate and does not claim full String/IME editing |
| `examples/surface_textbox_app.tetra` | Editable pure-Tetra Surface TextBox app with `FocusManager`, focused keyboard routing, caret movement, backspace/delete, resize preservation, and RGBA redraw evidence. | native linux-x64 headless linux-x64-real-window wasm32-web browser-canvas | exits 1 after click focus, text insertion, caret movement, backspace/delete, Tab focus transfer to `SubmitButton`, focused Space activation, resize, and final redraw; `surface-headless-text-focus-input-smoke.sh`, `surface-linux-x64-real-window-text-focus-input-smoke.sh`, and `surface-wasm32-web-browser-canvas-text-focus-input-smoke.sh` emit strict `tetra.surface.runtime.v1` evidence for the TextBox milestone; scoped clipboard/IME evidence belongs to `examples/surface_release_text_input.tetra`, while rich text, bidi shaping, platform accessibility tree, user JS, DOM UI, and legacy sidecars remain nonclaims |
| `examples/surface_tree_app.tetra` | Experimental pure-Tetra Surface component-tree app using the hardened `lib.core.component` `ComponentTree`/`TreeNode` helper API, Column/Row layout, TextBox, and Submit/Reset Buttons. | native linux-x64 headless linux-x64-real-window wasm32-web browser-canvas | exits 1 after helper-routed tree hit testing/focus/text/button/resize handling; `surface-headless-component-tree-smoke.sh`, `surface-linux-x64-real-window-component-tree-smoke.sh`, `surface-wasm32-web-browser-canvas-component-tree-smoke.sh`, `surface-headless-component-tree-api-smoke.sh`, `surface-linux-x64-real-window-component-tree-api-smoke.sh`, and `surface-wasm32-web-browser-canvas-component-tree-api-smoke.sh` emit strict `tetra.surface.runtime.v1`, `tetra.surface.component-tree.v1`, and `tetra.surface.component-tree-api.v1` evidence for stable node IDs, parent/child links, layout bounds, draw order, focus order, root-to-leaf dispatch paths, `tree_add_root`/`tree_add_child` builder calls, `tree_validate` invariants, Column/Row layout helpers, helper hit tests, focus wrap, `manual_bookkeeping:false`, focused TextBox text routing, Button action routing, resize relayout, changed frame checksums, and rejection of fake/DOM/user-JS/Node-only/legacy sidecar evidence |
| `examples/surface_toolkit_form.tetra` | Experimental pure-Tetra Surface minimal toolkit form using reusable `lib.core.widgets` Panel/Column/Text/TextBox/Row/Button helpers over the hardened ComponentTree API. | native linux-x64 headless linux-x64-real-window wasm32-web browser-canvas | exits 1 after helper-routed TextBox focus/editing, caret/backspace/delete, Tab cycling through Submit/Reset, keyboard-routed Submit and Reset, StatusText updates, resize relayout, and redraw; `surface-headless-minimal-toolkit-smoke.sh`, `surface-linux-x64-real-window-minimal-toolkit-smoke.sh`, and `surface-wasm32-web-browser-canvas-minimal-toolkit-smoke.sh` emit strict `tetra.surface.runtime.v1`, `tetra.surface.component-tree.v1`, `tetra.surface.component-tree-api.v1`, and `tetra.surface.toolkit.v1` evidence with `toolkit_level = minimal-widgets-v1`, `module = lib.core.widgets`, `experimental:true`, `production_claim:false`, `uses_component_tree_api:true`, `manual_bookkeeping:false`, no DOM UI, no user JS, no platform widgets, no magic widgets, changed frame checksums, and no legacy sidecars |
| `examples/surface_toolkit_settings.tetra` | Experimental pure-Tetra Surface toolkit reuse settings form using the same reusable `lib.core.widgets` Panel/Column/Text/TextBox/Row/Button helpers across a second app shape. | native linux-x64 headless linux-x64-real-window wasm32-web browser-canvas | exits 1 after helper-routed NameTextBox click focus, independent NameTextBox/EmailTextBox text input, Tab traversal to SaveButton and ResetButton, keyboard-routed Save/Reset actions, StatusText updates, both TextBoxes clearing on Reset, resize relayout to 480x320, and redraw; `surface-headless-toolkit-reuse-smoke.sh`, `surface-linux-x64-real-window-toolkit-reuse-smoke.sh`, and `surface-wasm32-web-browser-canvas-toolkit-reuse-smoke.sh` emit strict `tetra.surface.runtime.v1`, `tetra.surface.component-tree.v1`, `tetra.surface.component-tree-api.v1`, and `tetra.surface.toolkit.v1` evidence with `toolkit_level = toolkit-reuse-v1`, `reuse_level = multi-form-widget-reuse-v1`, both toolkit example sources, `module = lib.core.widgets`, `experimental:true`, `production_claim:false`, `manual_bookkeeping:false`, `demo_specific_widget_structs:false`, two TextBoxes, two Buttons, focused-only mutation evidence, changed frame checksums, artifact scans, no DOM UI, no user JS, no platform widgets, no magic widgets, no Node-only browser promotion, and no legacy sidecars |
| `examples/surface_accessibility_settings.tetra` | Experimental pure-Tetra Surface accessibility metadata tree over reusable `lib.core.widgets` settings form helpers and `lib.core.accessibility` metadata helpers. | native linux-x64 headless linux-x64-real-window wasm32-web browser-canvas | exits 1 after helper-routed NameTextBox/EmailTextBox focus and text input, Save/Reset actions, StatusText updates, reset clearing both TextBoxes, resize relayout to 480x320, and redraw; `surface-headless-accessibility-metadata-smoke.sh`, `surface-linux-x64-real-window-accessibility-metadata-smoke.sh`, and `surface-wasm32-web-browser-canvas-accessibility-metadata-smoke.sh` emit strict `tetra.surface.runtime.v1`, `tetra.surface.component-tree.v1`, `tetra.surface.component-tree-api.v1`, `tetra.surface.toolkit.v1`, and `tetra.surface.accessibility-tree.v1` evidence for roles, labels, label relationships, values, states, bounds, focus order, reading order, edit/press/save/reset actions, snapshots, metadata and bounds checksum changes, artifact scans, no DOM/ARIA evidence, no user JS, no platform accessibility host, no screen-reader claim, no production accessibility claim, and no legacy sidecars |
| `examples/surface_migration_ui_web_smoke.tetra` | Pure-Tetra Surface migration of `examples/ui_web_smoke.tetra`. | native surface-migration | exits 2 through the Linux-x64 Surface Host ABI using ordinary struct state, draw/event methods, local Surface events, and no metadata sidecars; not Linux-x64 real-window or wasm promotion evidence |
| `examples/surface_migration_ui_native_shell_smoke.tetra` | Pure-Tetra Surface migration of `examples/ui_native_shell_smoke.tetra`. | native surface-migration | exits 11 through the Linux-x64 Surface Host ABI with native-shell commands expressed as Tetra event handling and no metadata sidecars; not Linux-x64 real-window or wasm promotion evidence |
| `examples/surface_migration_dogfood_web_ui.tetra` | Pure-Tetra Surface migration of the dogfood web UI project. | native surface-migration | exits 3 through the Linux-x64 Surface Host ABI for dogfood counter/select state without metadata sidecars; not Linux-x64 real-window or wasm promotion evidence |
| `examples/surface_migration_tetra_control_center.tetra` | Pure-Tetra Surface migration of the Tetra Control Center metadata UI. | native surface-migration | exits 5 through the Linux-x64 Surface Host ABI for dashboard/profile/dry-run/refresh events without platform widgets or metadata sidecars; not Linux-x64 real-window or wasm promotion evidence |
| `examples/tooling_tests.tetra` | Minimal `tetra test` block smoke. | native test-only | `tetra test` passes; no `main`, so excluded from run smoke |
| `examples/projects/hello_t4/src/main.t4` | Minimal project-first `.t4` app with `Capsule.t4`. | native | exits 0 |
| `examples/projects/hello_t4/Capsule.t4` | Project-first capsule manifest for the hello `.t4` app. | native project metadata | declares `src/main.t4`, `src`, `tests`, `linux-x64`, and `io`; not a runnable entry itself |
| `examples/projects/hello_t4/tests/main_test.t4` | Project-first `.t4` test block for `hello_t4`. | native test-only | `tetra test .` passes; no `main`, so excluded from run smoke |
| `examples/projects/dogfood_cli/src/main.tetra` | Dogfood CLI project build smoke. | native | exits 0 |
| `examples/projects/dogfood_actor_task/src/main.tetra` | Dogfood actor/task project smoke. | native | exits 0 |
| `examples/projects/eco_dogfood/src/main.tetra` | Eco dogfood project baseline smoke. | native | exits 0 (excluded from linux-x64 smoke profile) |
| `examples/ui_web_smoke.tetra` | UI metadata web smoke. | wasm | artifact/import preflight by default; runtime exit 0 only with explicit browser gate evidence |
| `examples/projects/dogfood_wasi/src/main.tetra` | Dogfood WASI project smoke. | wasm | artifact/import preflight by default; runtime exit 0 only with explicit runner evidence |
| `examples/projects/dogfood_web_ui/src/main.tetra` | Dogfood web UI project smoke. | wasm | artifact/import preflight by default; runtime exit 0 only with explicit browser gate evidence |

## Excluded from linux-x64 smoke profile

The `./tetra smoke --list --format=json` report also emits `excluded_examples`.
These examples are intentionally outside the default linux-x64 smoke profile, but
remain visible here with the exact exclusion reason reported by the smoke list.

| Example | Reason |
| --- | --- |
| `examples/actors_decl_spawn.tetra` | not part of linux-x64 smoke profile |
| `examples/actors_tagged_stress.tetra` | not part of linux-x64 smoke profile |
| `examples/cap_mem_ptr_smoke.tetra` | not part of linux-x64 smoke profile |
| `examples/ctx_switch_sysv_smoke.tetra` | not part of linux-x64 smoke profile |
| `examples/ctx_switch_win64_smoke.tetra` | not part of linux-x64 smoke profile |
| `examples/enum_payload_smoke.tetra` | not part of linux-x64 smoke profile |
| `examples/experimental_math_smoke.tetra` | not part of linux-x64 smoke profile |
| `examples/experimental_memcpy_smoke.tetra` | not part of linux-x64 smoke profile |
| `examples/flow_grammar_surface_smoke.tetra` | not part of linux-x64 smoke profile |
| `examples/generic_struct_smoke.tetra` | not part of linux-x64 smoke profile |
| `examples/globals_smoke.tetra` | not part of linux-x64 smoke profile |
| `examples/hello.tetra` | not part of linux-x64 smoke profile |
| `examples/islands_double_free.tetra` | not part of linux-x64 smoke profile |
| `examples/projects/dogfood_wasi/src/main.tetra` | not part of linux-x64 smoke profile |
| `examples/projects/dogfood_web_ui/src/main.tetra` | not part of linux-x64 smoke profile |
| `examples/projects/eco_dogfood/src/main.tetra` | not part of linux-x64 smoke profile |
| `examples/projects/hello_t4/Capsule.t4` | not part of linux-x64 smoke profile |
| `examples/projects/hello_t4/src/main.t4` | not part of linux-x64 smoke profile |
| `examples/projects/hello_t4/tests/main_test.t4` | not part of linux-x64 smoke profile |
| `examples/struct_ctor_smoke.tetra` | not part of linux-x64 smoke profile |
| `examples/task_bounded_stress.tetra` | not part of linux-x64 smoke profile |
| `examples/tooling_tests.tetra` | not part of linux-x64 smoke profile |
| `examples/ui_web_smoke.tetra` | not part of linux-x64 smoke profile |

## Epic 14 Verification Commands

```sh
./tetra fmt --check examples
./tetra smoke --list --format=json > reports/smoke-list-linux-x64.json
go run ./tools/cmd/validate-smoke-list --report reports/smoke-list-linux-x64.json --examples-root examples
./tetra test --report=json examples > reports/examples-test-report.json
go run ./tools/cmd/validate-test-report --report reports/examples-test-report.json
go run ./tools/cmd/validate-example-index --smoke-list reports/smoke-list-linux-x64.json --index docs/user/examples_index.md
./tetra run --target linux-x64 examples/projects/dogfood_cli/src/main.tetra
./tetra run --target linux-x64 examples/projects/dogfood_actor_task/src/main.tetra
./tetra run --target linux-x64 examples/projects/eco_dogfood/src/main.tetra
```

## Validator Notes

- Validator schema IDs may retain historical artifact names even when the current
  release profile advances. `validate-example-index`, `validate-smoke-list`, and
  `validate-test-report` enforce strict JSON shape, deterministic smoke profiles,
  and failure evidence shape for the current branch state.

## Troubleshooting Notes (Epic 14)

Use these notes to separate unsupported profile boundaries from real regressions.

### Basic language examples (`V020-0701..0705`)

- `examples/hello.tetra` is intentionally excluded from linux-x64 smoke matrix; this is unsupported profile scope, not a compiler/runtime break.
- If `examples/flow_hello.tetra` or `examples/bool_smoke.tetra` stop compiling/running on native, treat as a regression and rerun `./tetra smoke --list --format=json`.

### Control-flow examples (`V020-0706..0710`)

- Loop/control examples should keep deterministic exits (`42` or `55`) in native smoke; any parser or lowering failure is a regression.
- If only `examples/for_collection_u8_smoke.tetra` fails while others pass, suspect byte-collection semantics rather than global smoke config.

### Const and assignment examples (`V020-0711..0715`)

- `const` and compound assignment failures are regressions when they fail formatting, parsing, or expected exit checks in smoke/test.
- Unsupported behavior should be documented explicitly; silent drift in exit codes is treated as broken behavior.

### Enum/match examples (`V020-0716..0720`)

- `examples/enum_match_smoke.tetra` and `examples/enum_exhaustive_match_smoke.tetra` are required native smoke coverage.
- Missing exhaustiveness diagnostics or changed exit contracts indicate a regression, not an unsupported target limitation.

### Optional/error examples (`V020-0721..0725`)

- `optional` and `typed error` smoke examples are release-covered on native and must keep stable expected exits.
- If only one optional variant fails, verify matcher semantics before changing smoke profiles.

### Generic/protocol/extension examples (`V020-0726..0730`)

- `generic`, `protocol`, and `extension` MVP examples are required in native smoke and should fail loudly on semantic regressions.
- `generic_struct` coverage is experimental evidence only unless the feature registry promotes generic structs to `current`.
- Enum payload constructor/match examples are current only for the narrow
  positional match/catch/if-let slice recorded in
  `docs/spec/current_supported_surface.md`.

### Safety/runtime examples (`V020-0731..0735`)

- `ownership`, `async`, `task`, `time_sleep`, `task_sleep_deadline`, `task_join_wait`, `task_group_cancel`, `deadline_aware_waits`, `actors_pingpong`, and `actor_sleep_pingpong` are native release-covered examples with deterministic exits.
- Scheduling-related nondeterminism is considered broken for these smokes; unsupported status must be documented as an exclusion.

### Memory/capability examples (`V020-0736..0740`)

- `islands_*`, `cap_mem`, `mmio`, and `memset` examples are split between required smoke cases and profile exclusions by design.
- If an excluded example appears as a failing smoke case unexpectedly, verify smoke-list config drift before changing code.

### UI/WASM examples (`V020-0741..0745`)

- `ui_web` and dogfood wasm/web examples are allowed as artifact/import preflight evidence on wasm targets.
- Native smoke exclusion for wasm-specific examples is expected; compile/link failures on wasm targets remain regressions.

### Project dogfood examples (`V020-0746..0750`)

- `dogfood_cli` and `dogfood_actor_task` are required native smoke entries with exit `0`; failures are regressions.
- `eco_dogfood` is intentionally excluded from linux-x64 smoke profile; local `./tetra run --target linux-x64 examples/projects/eco_dogfood/src/main.tetra` is the fallback check.
