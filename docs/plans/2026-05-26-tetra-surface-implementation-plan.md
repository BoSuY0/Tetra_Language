# Tetra Surface Implementation Plan

**Goal:** Move the UI direction toward a pure Tetra Surface Object System while preserving the existing metadata UI path as legacy compatibility until Surface has real evidence.
**Context:** The current repository has `ui.metadata-v1`, web preview sidecars, native-shell sidecars, and Linux-x64 native UI smoke evidence. `/home/tetra/Downloads/Tetra_Surface_Implementation_Plan.md` defines the new direction: user code stays pure Tetra, widgets are ordinary structs implementing Surface abilities, and platforms expose only a tiny Surface Host ABI.
**Execution:** Work milestone by milestone with TDD. Do not claim Linux/Web/macOS/Windows Surface support until the matching host evidence and strict validators exist.

## Tasks

1. **Scope lock and release truth**
   - **Goal:** Declare Surface as the new direction without deleting the legacy metadata UI.
   - **Files:** add `docs/spec/surface_v1.md`, `docs/user/surface_guide.md`; modify `docs/spec/ui_v1.md`, `docs/spec/current_supported_surface.md`, `compiler/features.go`, `compiler/tests/semantics/features_test.go`, and regenerated `docs/generated/manifest.json`.
   - **Approach:** add planned feature IDs for `ui.surface-core`, `ui.surface-headless`, `ui.surface-linux-x64`, `ui.surface-web-wasm`, and `ui.surface-component-model`. Mark `ui.metadata-v1` as legacy metadata compatibility and explicitly state that old `.ui.web.mjs`/`.ui.html` artifacts are not the Surface path.
   - **Verification:** `go test ./compiler/tests/semantics -run FeatureRegistry -count=1`; `go run ./tools/cmd/gen-manifest`; `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`.
   - **Done when:** feature registry, generated manifest, and docs agree that Surface is planned while metadata UI remains legacy compatibility.

2. **Surface core library contract**
   - **Goal:** Add the first pure-Tetra API surface for host handles, frames, events, and drawing.
   - **Files:** add `lib/core/surface.tetra`, `lib/core/draw.tetra`, and `examples/surface_counter.tetra`; inspect existing stable stdlib verification before deciding whether these modules enter the stable module doc checks immediately.
   - **Approach:** start with data types and wrappers matching the planned Surface Host ABI. Keep API minimal: open/close/poll_event/begin_frame/present/now_ms/request_redraw plus RGBA software drawing helpers.
   - **Verification:** parse/check tests for the new example and any stdlib module checks that apply.
   - **Done when:** `examples/surface_counter.tetra` is a pure Tetra component example and does not use legacy `state/view` metadata.

3. **Surface effect and lifetime safety**
   - **Goal:** Make `uses surface` a real effect boundary and prevent unsafe escape of Surface resources.
   - **Files:** inspect and modify `compiler/internal/semantics/builtins.go`, `compiler/internal/semantics/effects.go`, relevant ownership/lifetime checker files, and add tests under `compiler/tests/semantics` or `compiler/tests/ownership`.
   - **Approach:** add failing tests first for missing `uses surface`, `Frame` use-after-present, `DrawContext` global/storage escape, `Frame.pixels` escape, double-close, and task/actor transfer rejection.
   - **Verification:** targeted semantics/ownership tests, then broader compiler tests.
   - **Done when:** Surface builtins are gated by `uses surface` and unsafe escapes get stable diagnostics.

4. **Headless deterministic runtime and validators**
   - **Goal:** Prove Surface behavior without a window through scripted events, framebuffer output, and checksums.
   - **Files:** add `compiler/internal/surfacert` or equivalent runtime host package, `tools/validators/surface`, `tools/cmd/validate-surface-runtime`, `tools/cmd/surface-runtime-smoke`, and `scripts/release/surface/surface-headless-smoke.sh`.
   - **Approach:** define `tetra.surface.runtime.v1` report schema with frames, events, checksums, state transitions, executable process evidence, and explicit rejection of docs-only, metadata-only, web-only, fake, mock, and placeholder reports.
   - **Verification:** validator RED/GREEN tests, smoke command test, and the headless release script.
   - **Done when:** `examples/surface_counter.tetra` produces deterministic headless frame/event/checksum evidence accepted by the strict validator.

5. **Linux-x64 Surface host**
   - **Goal:** Add a Linux-x64 host behind the same Surface Host ABI without exposing GTK/Qt/OS widgets to Tetra code.
   - **Files:** add or extend Linux runtime/backend integration after the headless ABI is stable; add `scripts/release/surface/surface-linux-x64-smoke.sh`.
   - **Approach:** start from the smallest executable host proof that opens or presents a Surface, processes scripted click input, and records at least two frames through the same report schema.
   - **Verification:** Linux-x64 smoke script plus `validate-surface-runtime` on the Linux report.
   - **Done when:** Linux-x64 evidence proves a real Surface host path, not metadata sidecar playback.

6. **WASM Surface boundary**
   - **Goal:** Add `wasm32-web-surface` as Tetra Surface in wasm with no user JS and no DOM UI.
   - **Files:** inspect `compiler/internal/backend/wasm32_web`, add Surface-specific backend tests and validators before changing output naming.
   - **Approach:** keep legacy `.ui.web.mjs`/`.ui.html` only for `ui.metadata-v1`. If a browser bootloader is unavoidable, make it compiler-owned, tiny, and non-UI.
   - **Verification:** backend tests prove no legacy UI sidecars for Surface apps; web Surface validator rejects DOM UI/user-JS evidence.
   - **Done when:** wasm Surface reports truthfully distinguish compiler boot from user JS and avoid DOM UI claims.

7. **Migration examples and release gate**
   - **Goal:** Move user-facing examples and release docs toward Surface after evidence exists.
   - **Files:** add Surface migrations for `examples/ui_web_smoke.tetra`, `examples/ui_native_shell_smoke.tetra`, `examples/projects/dogfood_web_ui`, and `examples/projects/tetra_control_center`; update release docs and manifests.
   - **Approach:** keep old examples for compatibility, but make new examples prefer Surface.
   - **Verification:** Surface gate script, docs verification, smoke list validation, and graph update.
   - **Done when:** release claims prefer Surface only for targets with passing Surface reports.
