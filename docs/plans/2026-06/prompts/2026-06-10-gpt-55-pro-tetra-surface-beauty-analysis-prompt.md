# GPT-5.5 Pro Prompt: Tetra Surface Beauty Layer Analysis

Use this prompt with GPT-5.5 Pro as a senior analyzer/architect. It follows
the local GPT-5.5 prompting guide:
`/home/tetra/Downloads/gpt-5-5-prompting-guide-en.md`.

Recommended settings:

- `reasoning.effort`: `high` or `xhigh`
- `text.verbosity`: `medium`
- Output language: Ukrainian, unless the caller asks for English.

---

## Prompt

You are a senior product + UI systems architect analyzing Tetra Surface.

Your job is not to implement code. Your job is to design the missing
developer-facing beauty layer that lets Tetra Surface reach Electron/React-class
visual richness without Electron, Chromium, React, DOM UI, CSS runtime, or user
JavaScript app logic.

### Goal

Create a concrete architecture proposal for a Tetra Surface Beauty Layer built
on the existing Block-first Surface System.

The key challenge:

Tetra already has the engine, primitives, reports, gates, examples, validators,
and scoped evidence. What is missing is the elegant authoring model that makes
apps feel as polished as Electron/React apps while keeping the Tetra idea pure:

```text
No core Button.
No core Label.
No core Card.
No core Modal.
No React-style component zoo.

Block + properties + behavior + states + animation + accessibility metadata.
```

The answer must invent something stronger than "add components" or "copy CSS".
It should feel mathematically elegant: not plain `2 + 2`, but more like
`sqrt(64) * 2 - 3^2 - 2 * 2.5`: internally layered and expressive, yet the
visible result is simple enough that developers can hold it in their head.

Do not make the system so clever that developers will hate Tetra Surface.
Sophistication must live inside the model, not leak into every app file.

### Current State To Treat As Facts

Tetra Surface has two relevant layers:

1. `Tetra Surface v1`
   - Current for bounded `surface-v1-linux-web` release scope.
   - Supports pure-Tetra UI over the tiny Surface Host ABI.
   - Has headless evidence, linux-x64 real-window evidence, and wasm32-web
     browser-canvas evidence.
   - Has a production widget subset today: `Text`, `Label`, `StatusText`,
     `Button`, `TextBox`, `Checkbox`, `Row`, `Column`, `Panel`, `Stack`,
     `Scroll`, and `Spacer`.
   - Has text/input, clipboard, IME/composition baseline, accessibility
     metadata plus platform bridge evidence for supported targets.

2. `Tetra Surface Block System`
   - Experimental.
   - Block-first architecture where `Block` is the core Surface primitive for
     layout, paint, text, assets, input/events, states, motion, and
     accessibility metadata.
   - Existing button-like/card-like/input-like/sidebar-like/modal-like shapes
     must become Block recipes or compatibility layers, not core primitives.
   - Has evidence through `tetra.surface.block-system.gate.v1`.
   - Has same-commit headless, linux-x64 real-window, and wasm32-web
     browser-canvas Block reports.
   - Has `block_system.memory_budget` evidence.
   - Has validators rejecting fake core primitive claims and unsupported target
     claims.
   - Has polished Block-only examples:
     - `examples/surface_block_command_palette.tetra`
     - `examples/surface_block_project_dashboard.tetra`
     - `examples/surface_block_settings.tetra`
     - `examples/surface_block_editor_shell.tetra`
     - `examples/surface_block_glass_panel.tetra`

Final audit status:

- P00-P20 are completed.
- The final audit verdict is `NEAR_READY_WITH_BLOCKERS`, not
  `PROD_READY_SCOPED`, because the checkout was dirty.
- The technical gates passed, but final production signoff requires a clean
  committed checkout rerun.

Important source files and artifacts to inspect if available:

- `GOAL.md`
- `docs/spec/surface_v1.md`
- `docs/spec/current_supported_surface.md`
- `docs/user/surface_guide.md`
- `docs/user/examples_index.md`
- `docs/release/surface_v1_release_contract.md`
- `docs/release/surface_v1_release_audit.md`
- `reports/surface-block/final/final-readiness-audit.md`
- `reports/surface-block/final/block-system/surface-block-system-gate-summary.json`
- `reports/surface-block/final/surface-release-v1/surface-release-summary.json`
- `compiler/features.go`
- `docs/generated/manifest.json`
- `lib/core/block.tetra`
- `lib/core/widgets.tetra`
- `examples/surface_block_*.tetra`

### What You Must Design

Design the missing Beauty Layer. It should answer:

- How does a developer create Electron/React-level visual polish using Blocks?
- What general abstractions should exist above raw Block properties?
- How do themes, tokens, style state, layout, typography, icons/assets, motion,
  shadows, focus rings, and accessibility fit together?
- What is the smallest authoring API that feels powerful rather than tiny?
- Which concepts are primitives, which are recipes, and which are only
  compatibility adapters?
- How do we avoid a React component zoo while still making app authoring fast?
- How do we avoid CSS-like chaos, global style conflicts, duplicated recipes,
  naming collisions, and hidden magic?
- How do we keep performance/RAM bounded and validator-friendly?
- What should be validated before any beauty claim is allowed?

### Required Design Philosophy

The answer must preserve the Tetra Surface idea:

```text
Block is the primitive.
Everything beautiful is composition.
Everything reusable is a recipe, theme, token, or behavior bundle.
Nothing becomes a magical core widget unless there is an overwhelming reason.
```

Think in terms of layered expressive systems:

- `Block` as physical material.
- Tokens as the shared visual physics.
- Recipes as reusable construction patterns.
- State selectors as behavior grammar.
- Motion presets as time grammar.
- Typography as rhythm.
- Assets/icons as semantic material.
- Accessibility metadata as a first-class output of the same graph.
- Validators as truth boundaries.

But keep the developer experience simple. A developer should be able to write a
beautiful command palette or dashboard without needing to manually specify 80
fields per Block.

### Constraints

- Do not propose Electron, Chromium, React, DOM UI, CSS runtime, user
  JavaScript app logic, Qt, GTK, Cocoa, WinUI, or platform-native widgets as
  the user-facing UI dependency.
- Do not make `Button`, `Card`, `TextField`, `Sidebar`, or `Modal` core
  primitives.
- Do not claim GPU rendering, macOS Surface production, Windows Surface
  production, wasm32-wasi Surface UI production, full screen-reader production,
  or full rich-text editor support.
- Do not erase the current release widget subset; treat it as compatibility and
  migration surface.
- Do not invent facts about the repo. If you cannot verify a fact, label it as
  an assumption.
- Do not output a vague essay. Produce a decision-grade proposal.
- Do not design a system that requires every app to create custom one-off
  styles from scratch.
- Do not design a system with duplicated sources of truth for color, spacing,
  typography, state, or motion.
- Do not design a system that forces developers to understand renderer internals
  for normal app authoring.

### Success Criteria

Your response succeeds only if it includes:

- A clear recommendation for the Beauty Layer architecture.
- A concise explanation of why this is better than copying React/CSS/Electron.
- A layered model showing what belongs in:
  - raw Block primitive;
  - token/theme system;
  - recipe system;
  - state/motion system;
  - asset/icon system;
  - accessibility output;
  - validators and reports.
- At least 3 alternative designs considered, with tradeoffs.
- A recommended hybrid or final direction.
- A small authoring API sketch in Tetra-like pseudocode.
- Examples for at least:
  - command palette;
  - dashboard shell;
  - settings form;
  - editor shell;
  - glass/control panel.
- Anti-complexity rules that prevent duplication, conflicts, style drift,
  framework bloat, and "developers will hate this" failure modes.
- A migration path from the existing widget subset to Block recipes.
- Validator and evidence requirements before any "beautiful UI" claim is
  accepted.
- A phased implementation roadmap with small packets and verification commands.
- A risk register.
- A final "what to do next" section.

### Desired Conceptual Bar

Do not stop at obvious abstractions like:

```text
theme + button recipe + card recipe
```

Push one level deeper. Find the abstraction that makes many beautiful things
fall out naturally.

For example, consider whether Tetra needs concepts like:

- visual material;
- surface grammar;
- interaction grammar;
- semantic affordance;
- recipe algebra;
- token resolution graph;
- stateful style lens;
- bounded style capsule;
- motion contract;
- accessibility projection;
- evidence-backed visual contract.

You do not have to use these names. Invent better names if needed. The point is
to discover the right abstraction layer, not to decorate ordinary components
with fancy words.

### Output Format

Return a Markdown document with these sections:

1. `Executive Verdict`
   - One paragraph.
   - State the recommended Beauty Layer in plain language.

2. `Current Truth`
   - What exists today.
   - What is proven.
   - What is still experimental.
   - What must not be claimed.

3. `Design Thesis`
   - The deep idea.
   - Explain the "sqrt(64) * 2 - 3^2 - 2 * 2.5" metaphor as:
     internally composed sophistication, externally simple authoring.

4. `Three Candidate Architectures`
   - Candidate A.
   - Candidate B.
   - Candidate C.
   - Tradeoffs.
   - Why the recommended one wins.

5. `Recommended Architecture`
   - Layer diagram in text.
   - Responsibilities of each layer.
   - What is stable, experimental, internal, and recipe-only.

6. `Developer Authoring Model`
   - Tetra-like pseudocode.
   - Show how a developer creates polished UI without core Button/Card/etc.
   - Show override rules.
   - Show how duplication is avoided.

7. `Beauty Primitives`
   - Tokens/theme.
   - Paint/material.
   - Layout.
   - Typography.
   - Assets/icons.
   - State selectors.
   - Motion.
   - Accessibility.
   - Validation/report evidence.

8. `Recipes Without Component Bloat`
   - Define recipe rules.
   - Explain how recipes remain Block configurations.
   - Explain naming, composition, override, and versioning.

9. `Example App Blueprints`
   - Command palette.
   - Dashboard shell.
   - Settings form.
   - Editor shell.
   - Glass/control panel.

10. `Migration Path`
    - How current `lib.core.widgets` maps to Block recipes.
    - Compatibility strategy.
    - Deprecation/non-deprecation recommendation.

11. `Validator And Evidence Plan`
    - What reports must prove.
    - What fake claims must be rejected.
    - What memory/performance budget facts must be recorded.

12. `Anti-Hate Rules`
    - Specific rules to stop conflicts, duplication, style drift, excessive
      abstraction, and hidden magic.

13. `Implementation Roadmap`
    - Small packets.
    - Each packet should have acceptance criteria and suggested verification.

14. `Risks And Failure Modes`
    - Include mitigation for each.

15. `Final Recommendation`
    - Clear next 3 actions.

### Validation Before Final Answer

Before finalizing:

- Check whether the proposal accidentally introduces core `Button`, `Card`,
  `TextField`, `Sidebar`, or `Modal`.
- Check whether the proposal depends on Electron, React, DOM, CSS runtime,
  Chromium, user JavaScript, or platform-native widgets.
- Check whether the proposed API has duplicated sources of truth.
- Check whether a developer can build a polished app without writing excessive
  boilerplate.
- Check whether a validator could prove or reject the claims.
- Check whether every production-like claim is scoped and evidence-backed.
- If any check fails, revise before answering.

### Stop Rules

- If source files are available, inspect them before making repo-specific
  claims.
- If source files are unavailable, state which facts are assumed and proceed
  with an architecture proposal.
- Ask at most one narrow clarifying question only if missing information would
  materially change the architecture.
- Do not implement code.
- Do not produce a marketing page.
- Finish with a decision-grade proposal, not brainstorming fragments.

### Tone

Be sharp, imaginative, and practical.

The user wants something that feels powerful and slightly magical, but the
engineering result must be boring in the best way: predictable, composable,
testable, and hard to misuse.

The Beauty Layer should feel like:

```text
Electron-class visual richness
minus Electron-class memory appetite
minus React-style component sprawl
minus CSS global chaos
plus Tetra's Block-first truth model
```
