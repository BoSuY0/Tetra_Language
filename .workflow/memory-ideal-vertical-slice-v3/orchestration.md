# Orchestration: Memory Ideal Vertical Slice v3

## Execution Rules

- Keep the original objective intact.
- Ask for approval before risky, expensive, external, or destructive actions.
- Keep immediate blocking work local.
- Delegate only if the user explicitly authorizes delegation or parallel
  agents.
- Integrate packet results before final verification.

## Branching Rules

- If a RED test fails for the expected missing capability, implement the
  narrowest behavior that makes it GREEN.
- If a required case is already rejected by existing semantics, classify it as
  conservative and add a regression test rather than widening protocols.
- If unknown/dynamic dispatch, runtime protocol values, trait objects, witness
  tables, or existential ABI support would be required, keep the row
  conservative and document the nonclaim.
- If the same fix fails twice, stop and record a blocker in `GOAL.md`.

## Packet Prompts

- P0 discovery:
  Inspect v0/v1/v2 memoryfacts, memorymodel, correlation validator, report
  validator, docs, and protocol/static conformance tests. Output exact files,
  symbols, and recommended RED tests.
- P1 memoryfacts/model:
  Add tests and implementation for `interface_value_contains_borrow`,
  `protocol_dispatch_borrow_conservative`,
  `protocol_dispatch_noalias_conservative`, validator names, report
  projections, and MiniMemoryModel v3 cases.
- P2 semantics:
  Add focused semantics tests for known/static protocol target local borrowed
  use, interface/protocol owned-return rejection, global-storage rejection,
  unknown/dynamic dispatch conservative rejection/no trusted facts, and broad
  noalias rejection.
- P3 docs/audit/manifest:
  Add v3 correlation and final audit docs; update report schema, production
  core design only if needed, generated manifest, and manifest validator
  fixtures.
- P4 verification:
  Run all focused and full gates from `GOAL.md`, collect command evidence,
  refresh Graphify, and prepare final report.

## Completion Audit

Complete only when all v3 rows are classified as `validated_narrow` or
`conservative`, all required gates pass, and `GOAL.md` progress cites evidence
paths and commands.
