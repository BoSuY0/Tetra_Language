# Orchestration: Tetra Memory Production Core v1

## Execution Rules

- Keep the active objective intact: implement the external Memory Production Core v1 plan, using sub-agents.
- Execute the plan as verified vertical slices; current slice is MPC-16 Production gate and final audit.
- Keep immediate blocking design and implementation local to the orchestrator.
- Delegate sidecar read-only audits first because the worktree is dirty and broad.
- Do not let agents edit overlapping files unless a later packet has an explicit disjoint write set.
- Require files inspected, commands run, observed evidence, and uncertainty in every sub-agent result.
- Run TDD: add red tests before production code where the slice changes behavior.

## Branching Rules

- If a required existing feature already exists, preserve it and add graph/report validation around it.
- If full actor/task/request runtime semantics are too large, use MPC-12's stop condition: keep compiler-owned conservative checks and document runtime production work as post-MPC.
- If generic/lifetime/actor/runtime scope expands beyond the current boundary rules, document as later MPC work rather than broadening this patch.
- If a target has build/lower evidence but no runtime evidence, keep its claim level build/lower scoped until a target-specific smoke proves runtime behavior.
- If a cost row needs a dynamic check, keep that check in the normal build or classify the row as conservative; do not promote it to zero-cost.
- If `unsafe_unknown` is involved, keep the optimization claim conservative unless compiler-owned proof proves a safer class.
- If a generated program lacks an oracle outcome, treat it as generator coverage only, not a passing fuzz proof.
- If fuzz output depends on unsupported unsafe or target behavior, classify it as conservative/rejected rather than upgrading safety claims.
- If a final audit row lacks concrete artifact or command evidence, classify it as `future`, `explicit_non_goal`, or a blocker rather than upgrading it to implemented/validated.
- If a required MPC-16 nonclaim is missing from the final audit docs, treat docs/manifest validation as incomplete even if Go tests pass.
- If tests fail outside the current slice, classify as unrelated unless evidence shows this work caused them.
- If the same fix fails twice, record a blocker in `GOAL.md` instead of trying a third blind variant.

## Packet Prompts

Packet files under `packets/` are authoritative. Each packet is self-contained and must not ask the sub-agent to reread the whole plan.

## Completion Audit

- Workflow artifact completeness must pass `verify_workflow.py`.
- Packet results must be integrated with `collect_results.py`.
- `GOAL.md` progress must record evidence anchors before any completion claim.
- `update_goal complete` is allowed only after every current-slice `done_when` item has fresh evidence.
