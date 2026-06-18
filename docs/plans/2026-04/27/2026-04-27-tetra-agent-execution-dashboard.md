# Tetra Agent Execution Dashboard

Status: active coordination file for the 2026-04-27 stabilization backlog.

This file tracks agent assignment, changed files, verification evidence, and
blockers so parallel work does not depend on chat history alone.

| Wave | Agent | REAL IDs | Scope | Changed files | Verification | Blocker status |
| --- | --- | --- | --- | --- | --- | --- |
| F | Wave F implementation agent | REAL-0501..REAL-0510 | Eco, security, docs, user adoption | `cli/cmd/tetra/eco.go`; `cli/cmd/tetra/eco_wave10_test.go`; `cli/cmd/tetra/testdata/eco_capsules/matrix/**`; `scripts/release/v1_0/security-review.sh`; `scripts/dev/fuzz-nightly.sh`; `tools/cmd/validate-example-index/**`; `tools/cmd/validate-performance-report/**`; `tools/scriptstest/security_review_test.go`; `tools/scriptstest/fuzz_nightly_test.go`; `docs/user/**`; `docs/release/post_v1_promotion_checklist.md`; `docs/performance/v1_0_thresholds.md`; `docs/generated/v1_0/performance-regression.*`; `docs/spec/v1_scope.md`; `README.md` | Focused Wave F commands recorded in final report | No Wave F blocker; broad `./tools/...` currently has unrelated build failures in pre-existing tool tests |

## Table Rules

- `REAL IDs` must list exact backlog IDs.
- `Changed files` should name files or narrow globs owned by the wave.
- `Verification` must include exact commands after they run.
- `Blocker status` must distinguish Wave-owned blockers from unrelated dirty
  worktree or other-agent failures.

## Handoff Checklist

- [ ] Agent has not edited `docs/plans/2026-04-27-tetra-real-stabilization-agent-backlog.md`.
- [ ] Agent lists every Wave-owned file changed.
- [ ] Agent reports exact verification commands and outcomes.
- [ ] Agent names blockers and residual risk without hiding unrelated failures.

