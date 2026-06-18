# Post-v0.4 Script Tests

This directory owns post-v0.4 release script tests moved out of the flat
`tools/scriptstest/` package.

Keep post-v0.4 release script coverage here so the top-level script test
directory can stay focused on shared helpers and package-level wiring.

Nested packages own larger post-v0.4 domains:

- `memory/`: memory, RAM contract, actor runtime foundation, and
  memory/islands/surface gates.
- `production/`: combined production and WASM/UI/GUI gates.
