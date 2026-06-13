# GPT-5.5 Pro Prompt: Tetra Surface Electron-Competitor Final Plan

Use this prompt with GPT-5.5 Pro as a senior analyzer/architect. The analyzer's
job is to create a brutal, implementation-ready Markdown plan that can bring
Tetra Surface from its current bounded Surface/Morph state to a full, stable,
production UI platform that can compete with Electron while preserving Tetra's
own architecture.

Recommended settings:

- `reasoning.effort`: `xhigh`
- `text.verbosity`: `high`
- Output language: Ukrainian
- Output artifact: one Markdown plan file

---

## Prompt

You are GPT-5.5 Pro acting as a senior analyzer, UI runtime architect, compiler
architect, release engineer, and adversarial product reviewer for the Tetra
Language repository.

Your task is not to implement code. Your task is to inspect the repository and
write a final, decisive, evidence-driven plan in a Markdown file that would let
Codex implement Tetra Surface until it can honestly claim:

```text
Tetra Surface is a full stable production UI platform that can replace
Electron/React/CSS for production app UI in the supported scope.
```

This must be a real engineering plan, not a motivational essay. It must be
strong enough that another coding agent can execute it packet by packet, run
gates, and know exactly when the claim is true and when it is still false.

### Required Output File

Create a new Markdown file in the repository:

```text
docs/plans/YYYY-MM-DD-surface-electron-competitor-production-plan.md
```

Use the current date in `YYYY-MM-DD`. Do not overwrite existing plan files.

### Current User Goal

The user wants the final level:

- Surface should not merely be "beautiful".
- Surface should not merely have experimental Morph evidence.
- Surface should become a stable production UI platform competitive with
  Electron.
- The final goal is to replace Electron/React/CSS for production app UI in the
  supported scope.

You must transform that goal into a concrete plan with hard blockers,
implementation packets, acceptance criteria, validators, release gates, and a
final readiness definition.

### Non-Negotiable Truth Rule

Do not claim the current repository already has this. The plan must distinguish:

- what is proven now;
- what is experimental now;
- what is missing;
- what is required to make the final claim true;
- what must remain a nonclaim until evidence exists.

If the repository contradicts any assumption below, trust the repository and
record the contradiction in the plan.

### Repository Context To Inspect

Inspect real files before making repo-specific claims. Start with:

```text
AGENTS.md
GOAL.md
PLAN.md
graphify-out/GRAPH_REPORT.md
compiler/features.go
docs/generated/manifest.json
docs/spec/current_supported_surface.md
docs/spec/surface_morph.md
docs/user/surface_guide.md
docs/user/examples_index.md
docs/release/surface_v1_release_contract.md
docs/release/surface_v1_release_audit.md
docs/plans/2026-06-10-gpt-55-pro-tetra-surface-beauty-analysis-prompt.md
docs/plans/2026-05-22-full-platform-ui-runtime.md
lib/core/surface.tetra
lib/core/block.tetra
lib/core/morph.tetra
lib/core/widgets.tetra
tools/cmd/validate-surface-runtime
tools/cmd/validate-surface-morph-report
tools/cmd/validate-surface-release-state
tools/validators/surface
scripts/release/surface
examples/surface_*.tetra
examples/core_*_smoke.tetra
```

If Graphify artifacts are stale, note that they are navigation aids only and
verify concrete files directly.

### Current State To Treat As Working Hypothesis

The repository has a bounded Surface v1 and an experimental Block/Morph path:

- Surface v1 currently has scoped release evidence for headless, Linux x64
  real-window, and wasm32-web browser-canvas paths.
- Surface v1 has a compatibility widget subset.
- The Block System is the Block-first primitive model.
- Morph Capsule is the experimental visual grammar over Block.
- Morph currently provides evidence that beautiful UI can be expressed as
  Block-first composition.
- This is not yet an Electron replacement, React replacement, CSS replacement,
  full desktop app shell, full cross-platform UI platform, full accessibility
  platform, full rich-text system, full renderer stack, or full production
  ecosystem claim.

Verify these facts against the repo. If any are wrong, correct them in the plan.

### The Final Target Claim To Plan Toward

The final claim is intentionally ambitious. Define it precisely.

The plan must specify the exact supported scope required for a truthful
Electron-competitor claim. At minimum, analyze and disposition these dimensions:

- desktop app shell;
- renderer/compositor;
- layout engine;
- style system replacing CSS runtime;
- authoring model replacing React-style component sprawl;
- state/event/application model;
- text shaping, editing, selections, IME, clipboard;
- accessibility across supported platforms;
- assets, fonts, icons, image decoding, caching;
- animation, transitions, reduced motion, frame timing;
- input: keyboard, pointer, wheel, touch where supported, drag/drop;
- menus, dialogs, notifications, tray, window management, DPI, cursors;
- IPC/process model where Electron normally provides main/renderer separation;
- filesystem and OS integration;
- dev workflow: hot reload, inspector, diagnostics, profiling, screenshots;
- packaging, signing/notarization where applicable, auto-update strategy;
- web target story and whether it is a first-class target or separate output;
- security sandbox and permission model;
- performance, memory, startup time, binary size, latency, power usage;
- crash recovery and error reporting;
- internationalization/localization;
- testing and visual regression infrastructure;
- CI/release gates;
- migration path from existing Surface v1 widgets and Morph examples.

### Important Product Constraint

Do not make "replace Electron/React/CSS" mean copying Electron/React/CSS
internals.

Prefer this direction unless repo evidence proves another path:

```text
Block remains the primitive.
Morph or a successor becomes the stable authoring/style/recipe system.
Platform adapters provide native window/input/accessibility/app-shell services.
The renderer is Tetra-owned and evidence-backed.
React/CSS/Electron become migration targets or compatibility imports, not
runtime dependencies.
```

### Hard Constraints

The final plan must not depend on:

- Electron;
- Chromium as the desktop runtime shell;
- React runtime;
- browser DOM as the production desktop UI tree;
- CSS cascade as the runtime style engine;
- user JavaScript app logic as the default UI model;
- Qt, GTK, Cocoa, WinUI, or platform-native widgets as the user-facing widget
  layer.

You may propose platform APIs for host integration, windows, accessibility,
IME, clipboard, menus, dialogs, notifications, GPU surfaces, and packaging.
Those are adapters, not the UI framework.

### Required Honesty

The final plan may recommend phased production scopes. If full
Windows/macOS/Linux/Web parity is too large for one release, the plan must
state which claims become:

- `PROD_STABLE_SCOPED`;
- `BETA_TARGET_HOST`;
- `EXPERIMENTAL`;
- `UNSUPPORTED`;
- `NONCLAIM`.

However, the plan must still describe the complete route to the user's desired
end state: a Surface platform competitive with Electron.

### What The Plan Must Contain

Write a Markdown plan with these sections exactly:

1. `Executive Verdict`
   - State the recommended route to an Electron-competitive Surface.
   - Say plainly whether the repo is currently ready for the final claim.

2. `Current Truth From The Repository`
   - Observed files, gates, docs, examples, and validators.
   - What is proven now.
   - What is experimental.
   - What is missing.
   - What facts are assumptions because evidence was unavailable.

3. `Final Claim Definition`
   - Define what "replaces Electron/React/CSS" means in this repo.
   - Define the minimum supported target matrix.
   - Define final readiness names and prohibited claims.

4. `Electron/React/CSS Parity Matrix`
   - A table comparing Electron/React/CSS capability areas against current
     Surface and required Surface.
   - Include app shell, renderer, layout, style, state, devtools, packaging,
     accessibility, performance, security, and ecosystem.

5. `Recommended Architecture`
   - Describe the final Surface stack.
   - Include layer diagram:
     raw Block, stable Morph/style graph, renderer/compositor, platform host
     ABI, app shell, devtools, validators, release gates.
   - Mark stable vs experimental vs internal layers.

6. `Architectural Alternatives Considered`
   - At least 4 alternatives.
   - Include "copy Electron", "copy React/CSS", "native widget wrappers",
     "Block/Morph-first Tetra-owned stack".
   - Explain why the recommended direction wins.

7. `Implementation Packets`
   - Create a numbered packet list, e.g. `SURFACE-PROD-P00` through as many
     packets as needed.
   - Every packet must include:
     - goal;
     - files/directories likely affected;
     - implementation notes;
     - tests;
     - validators/gates;
     - acceptance criteria;
     - fake-claim rejection cases;
     - dependencies;
     - risk.

8. `Renderer And Compositor Plan`
   - Software renderer baseline.
   - GPU/compositor path if needed.
   - Text/glyph pipeline.
   - Paint model.
   - Image/vector/icon pipeline.
   - Frame scheduling.
   - Visual regression strategy.
   - Performance budgets.

9. `Layout And Style Plan`
   - Block layout model.
   - Stable Morph/style/token graph.
   - CSS replacement boundaries.
   - Conflict/override rules.
   - Theming, variants, responsive constraints, density, DPI.
   - How to prevent CSS-like global chaos.

10. `Application Model Plan`
    - State, events, commands, navigation, focus, async work, undo/redo.
    - Component/recipe authoring model without React runtime.
    - App shell integration.
    - IPC/process model if needed.

11. `Platform Host Plan`
    - Linux, Windows, macOS, Web.
    - Window management, menus, dialogs, tray, notifications, cursors, drag/drop.
    - Clipboard/IME/accessibility per platform.
    - Target-host evidence requirements.
    - Unsupported target diagnostics.

12. `Developer Experience Plan`
    - CLI commands.
    - Hot reload or fast rebuild loop.
    - UI inspector.
    - Screenshot/golden tests.
    - Accessibility inspector.
    - Performance profiler.
    - Error diagnostics and source maps/source locations.
    - Project templates.

13. `Security And Sandbox Plan`
    - Permissions.
    - Filesystem/network boundaries.
    - IPC hardening.
    - Untrusted assets/fonts/images.
    - Supply chain/capsules.
    - Electron-class attack surfaces that Surface must avoid.

14. `Performance And Memory Plan`
    - Startup budgets.
    - Frame budgets.
    - Memory budgets.
    - Binary size budgets.
    - Power/CPU budgets.
    - Benchmark methodology against Electron without overclaiming.

15. `Accessibility And Internationalization Plan`
    - Roles, states, names, relationships.
    - Screen-reader evidence.
    - Keyboard navigation.
    - IME/composition.
    - Text shaping and bidi.
    - Localization hooks.

16. `Testing And Release Gates`
    - Unit, semantics, renderer, runtime, visual, accessibility, target-host,
      packaging, fuzz/property, stress/soak, performance.
    - New validators to add.
    - New release scripts to add.
    - Required broad commands.
    - Artifact hash and same-commit evidence.

17. `Documentation And Claim Governance`
    - Docs to update.
    - Manifest feature statuses.
    - Claim-tier language.
    - Overclaim scanners.
    - Nonclaims that must remain explicit until evidence changes.

18. `Migration Plan`
    - Existing `lib.core.widgets`.
    - Existing Block examples.
    - Existing Morph examples.
    - How apps migrate without breakage.
    - Compatibility/deprecation policy.

19. `Risk Register`
    - At least 20 risks.
    - Each risk must have severity, detection, mitigation, and blocking gate.

20. `Final Definition Of Done`
    - Exact final command list.
    - Required reports.
    - Required clean git state.
    - Required target-host evidence.
    - Exact conditions for `PROD_STABLE_SCOPED`.
    - Exact conditions for broader "Electron replacement" claim.

21. `Recommended First 10 Commits`
    - A practical sequence of the first ten implementation commits.
    - Each commit should be independently reviewable and testable.

22. `Prompt For Codex Goal`
    - Write a concise `/goal`-ready objective that can launch the plan.
    - Include the plan file path and done_when criteria.

### Required Packet Themes

Your packet list must cover, at minimum:

- baseline truth audit;
- claim taxonomy and overclaim validator;
- stable Morph/style graph promotion;
- Block ABI and renderer contract freeze;
- layout engine hardening;
- renderer/compositor production path;
- text shaping/editing/IME production path;
- accessibility production path per target;
- platform host adapters for Linux, Windows, macOS, Web;
- app shell features comparable to Electron;
- development inspector and hot reload;
- visual regression infrastructure;
- packaging/signing/update story;
- security/sandbox model;
- performance and memory gates;
- migration from widgets to recipes;
- docs/manifest/release governance;
- CI and release gate integration;
- final same-commit clean audit.

### Validation Requirements For Your Plan

Before writing the final plan file, check your own plan against these questions:

- Does it accidentally claim current Surface is already an Electron
  replacement?
- Does it rely on Electron/React/CSS/DOM as runtime dependencies?
- Does it hide platform-specific work behind vague words like "support
  Windows/macOS" without target-host evidence?
- Does it include real fake-claim validators?
- Does it include visual regression and accessibility evidence?
- Does it cover app-shell features, not only drawing pixels?
- Does it define performance and memory budgets?
- Does it give Codex enough file-level and command-level detail to execute?
- Does it preserve Tetra's Block/Morph-first model?
- Does it provide a clean final Definition Of Done?

If any answer is bad, revise the plan before saving it.

### Tone

Be direct, demanding, and practical. The user wants a "final and finishing"
plan, so the plan should be ambitious and slightly ruthless. But every claim
must be evidence-backed, scoped, and testable.

The right vibe:

```text
No more vibes.
No more "experimental beauty layer".
No fake Electron replacement claim.
Here is the exact mountain, the route, the gates, and the summit flag.
```

Finish by saving the Markdown plan file and reporting the path plus a very short
summary of the highest-risk blockers.
