# Surface Production Example Suite

`tetra.surface.example-suite-report.v1` records
`surface-production-example-suite-v1` evidence for realistic Surface app
shapes. The release gate is
`scripts/release/surface/example-suite-gate.sh`; the validator is
`validate-surface-example-suite`.

The required app shapes are:

- command palette app;
- settings app;
- project dashboard;
- editor shell;
- file manager shell;
- multi-window notes app;
- system tray/status app;
- notification/dialog demo;
- localized form;
- accessibility-heavy form.

Each example must be an executable `examples/surface_prod_*.tetra` program that
uses Block/Morph, includes event/state/accessibility/performance-budget
evidence, and avoids React, Electron, DOM runtime UI, external CSS, platform
widgets, and widget-backed production authorship where Block/Morph is required.

The scoped target coverage rows are `headless`, `linux-x64`, and `wasm32-web`.
They are evidence for `PROD_STABLE_SCOPED_LINUX_WEB_APP_UI`; they are not broad
cross-platform desktop parity, GPU parity, native platform-widget UI, or
screenshot-only production proof.

The ecosystem seed section must also prove the template catalog, scaffold smoke,
package smoke, examples index update, Surface guide update, and package report
artifacts. Screenshot-only demos, examples requiring React/Electron/DOM runtime,
missing app shapes, missing scoped target coverage, and toy visual-only examples
are rejected.
