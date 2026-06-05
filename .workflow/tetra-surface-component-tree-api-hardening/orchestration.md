# Orchestration: Tetra Surface Component Tree API Hardening

## Execution Rules

- Keep the original objective intact.
- Ask for approval before risky, expensive, external, or destructive actions.
- Keep immediate blocking work local.
- Delegate only bounded, disjoint, materially useful packets.
- Integrate packet results before final verification.

## Branching Rules

## Packet Prompts

## Completion Audit
# Orchestration

1. Read plan, Graphify context, and current files.
2. Add RED tests for API evidence, source anti-regression, smoke modes, and
   release scripts.
3. Implement the smallest helper API that satisfies the plan and current Tetra
   language/runtime constraints.
4. Rewrite the example app to call helpers for construction, layout, hit-test,
   focus, and dispatch-path evidence.
5. Extend smoke reports and validator rules, including negative fixtures.
6. Add API smoke scripts and aggregate gate wiring.
7. Update docs, feature registry, manifest, workflow final report, and Graphify.
8. Run targeted checks, then the full Definition of Done verification suite.

Branching rules:
- If Tetra cannot support `[]TreeNode` directly, use an equivalent helper-owned
  fixed-node API and document the equivalence.
- If an unrelated dirty change causes failure, isolate and report it instead of
  masking it with this milestone.
- If a smoke target fails because its external host environment is unavailable,
  keep the generated report/log evidence and state the exact blocker.

Packets:
- `api-core`: `lib/core/component.tetra`, example compile behavior.
- `example-source`: `examples/surface_tree_app.tetra`, source scan tests.
- `validator-evidence`: `tools/validators/surface`, `tools/cmd/surface-runtime-smoke`.
- `release-docs`: `scripts/release/surface`, docs, generated manifest.
