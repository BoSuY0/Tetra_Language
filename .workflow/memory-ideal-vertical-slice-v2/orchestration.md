# Orchestration: Memory Ideal Vertical Slice v2

## Execution Rules

- Keep the original objective intact.
- Ask for approval before risky, expensive, external, or destructive actions.
- Keep immediate blocking work local.
- Delegate only bounded, disjoint, materially useful packets.
- Integrate packet results before final verification.

## Branching Rules
- If a RED test fails for the expected missing capability, implement the
  narrowest behavior that makes it GREEN.
- If a required case is already rejected by existing semantics, classify it as
  conservative and add a regression test rather than widening callables.
- If unknown/capturing callback support would require excluded callable
  semantics, keep the row conservative and document the nonclaim.
- If the same fix fails twice, stop and record a blocker in `GOAL.md`.

## Packet Prompts
- P0 discovery:
  Inspect v0/v1 memoryfacts, memorymodel, correlation validator, report
  validator, docs, and semantics function-type tests. Output exact files,
  symbols, and recommended RED tests.
- P1 memoryfacts/model:
  Add tests and implementation for `function_value_contains_borrow`,
  `callback_arg_contains_borrow`, `callback_inout_conservative`, validator
  names, report projections, and MiniMemoryModel v2 cases.
- P2 semantics:
  Add focused semantics tests for known local callback, function-typed field,
  `.copy()` escape, non-borrow callback parameter, callback return/global
  escape, consumed callback argument, `inout`, alias/noalias conservatism,
  unknown target, capturing callback, and broad noalias rejection.
- P3 docs/audit/manifest:
  Add v2 correlation and final audit docs; update report schema, production
  core design only if needed, generated manifest, and manifest validator
  fixtures.
- P4 verification:
  Run all focused and full gates from `GOAL.md`, collect command evidence,
  run workflow helper scripts, refresh Graphify, and prepare final report.

## Completion Audit
Complete only when all v2 rows are classified as `validated_narrow` or
`conservative`, all required gates pass, and `GOAL.md` progress cites the
evidence paths and commands.
