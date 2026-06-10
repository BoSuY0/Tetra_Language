# Actor Runtime Production Foundation Goal

<goal>
Implement the full plan in
`/home/tetra/Downloads/actor-runtime-production-foundation-codex-plan.md`.

Mission: bring Actor Runtime Production Foundation v1 to
`PROD_STABLE_SCOPED` for an honestly bounded Linux-x64 actor/task runtime
foundation, without broad Tetra production, non-Linux distributed actor,
distributed zero-copy, official benchmark, or full Erlang/OTP-style claims.
</goal>

<context>
Active `/goal` objective:

`$goal-forge $goal-loop $define-goal Реалізуй повністю весь цей план - /home/tetra/Downloads/actor-runtime-production-foundation-codex-plan.md`

Primary source of scope:

- `/home/tetra/Downloads/actor-runtime-production-foundation-codex-plan.md`

Working evidence root:

- `reports/actor-runtime-foundation/`

Tracker note: previous Surface Block tracking in `GOAL.md`, `PLAN.md`,
`NOTES.md`, or `CONTROL.md` is stale for this active thread goal. Keep these
files aligned to the actor-runtime objective unless the user explicitly changes
the active objective.
</context>

<constraints>
- Always communicate with the user in Ukrainian.
- Preserve unrelated dirty worktree changes.
- Use persistent Go caches under `.cache/`, `.gocache`, or
  `${XDG_CACHE_HOME:-$HOME/.cache}/tetra-language/...`; never set `GOCACHE` to
  `/tmp`.
- Use `GOTELEMETRY=off` for Go evidence commands.
- Evidence for this goal lives under `reports/actor-runtime-foundation/`.
- Do not delete RED tests to obtain green status.
- Do not use docs as truth without code/test/script/validator evidence.
- Do not make an actor runtime production claim unless release gates and
  validators force the evidence.
- Do not claim cross-target distributed runtime parity, distributed pointer or
  region zero-copy, official benchmark/speed superiority, full Erlang/OTP
  supervision, cluster membership, formal race-safety proof, or full Tetra
  production readiness.
- Do not touch Memory/Islands/Surface unless an actor packet explicitly bridges
  there and its regressions run.
- After modifying code, run `graphify update .` before closing the packet.
</constraints>

<scorecard>
Primary checklist: every packet `ACTOR-P00` through `ACTOR-P17` is implemented
or explicitly dispositioned with same-commit evidence under
`reports/actor-runtime-foundation/`.

Passing threshold: all Final Definition Of Done bullets in the external plan
are proven by current repo files, command logs, gate artifacts, and final audit.

Regression checks: actor foundation gate, broad compiler/cli/tools tests, race
actor slice, docs/manifest verification, artifact hash validation, fake-claim
validator tests, `git diff --check`, `git status --short`, and Graphify update
after code changes.

Stop condition: if the same gate/test failure repeats twice without new
evidence, record the blocker in `GOAL.md` and stop changing implementation
until new evidence or user direction exists.
</scorecard>

<done_when>
The goal is complete only when all are true:

- `ACTOR-P00` baseline discovery/truth map exists with command status evidence.
- `ACTOR-P01` scheduler boundary is either `RUNTIME_PROVEN_SCOPED` or explicitly
  `PROTOTYPE_ONLY_NON_GOAL` with validator-enforced nonclaim evidence.
- `ACTOR-P02` message pool exhaustion/reclamation behavior is checked,
  recoverable, or release-blocking without silent overflow.
- `ACTOR-P03` bounded mailbox backpressure and send/recv policy are executable
  and validator-enforced.
- `ACTOR-P04` typed mailbox ownership and actor/island transfer proof rejects
  borrowed, unsafe, stale, and distributed zero-copy overclaims.
- `ACTOR-P05` actor failure, shutdown, done actor, and invalid handle semantics
  are deterministic.
- `ACTOR-P06` actor/task cancellation and structured concurrency slice passes.
- `ACTOR-P07` race-safety conservative rejection matrix passes.
- `ACTOR-P08` actor/island boundary integration passes without violating
  IslandID/Epoch/token/provenance rules.
- `ACTOR-P09` Linux-x64 distributed loopback actor runtime smoke is hardened,
  same-commit, and validated.
- `ACTOR-P10` leak, race, and soak evidence exists for actor/broker/stress
  scope.
- `ACTOR-P11` stable diagnostics and JSON evidence exist for negative actor/task
  cases.
- `ACTOR-P12` dedicated actor foundation validator and release gate exist and
  pass.
- `ACTOR-P13` CI and package release workflows include the actor gate without
  hidden `continue-on-error` bypass for actor foundation claims.
- `ACTOR-P14` docs/spec/user guides are corrected and docs validators reject
  actor overclaims.
- `ACTOR-P15` benchmark work is Tier 0/Tier 1 preparation only, with no public
  speed or official benchmark claim.
- `ACTOR-P16` runtime ABI/selfhostrt parity and unsupported target diagnostics
  are checked.
- `ACTOR-P17` final same-commit evidence bundle and final audit exist with
  exact commands, artifacts, hashes, git head, dirty/clean state, verdict, and
  nonclaims.
- Final commands in the external plan pass or any non-passing item is recorded
  as a blocker/nonclaim that prevents completion.
</done_when>

<feedback_loop>
Fast loop while iterating: targeted `go test -buildvcs=false` package slices
named in the active packet, with `GOTELEMETRY=off`, repo-local `GOCACHE`, and
repo-local `GOTMPDIR`.

Packet loop: run packet-specific smoke/validator commands, `git diff --check`,
and update `reports/actor-runtime-foundation/PXX/` plus tracker files.

Final loop: run the actor foundation gate, broad compiler/cli/tools test slice,
race actor slice, docs/manifest validators, artifact hash validators, and
Graphify update if code changed.
</feedback_loop>

<workflow>
1. Re-read `GOAL.md`, `AGENTS.md`, and the relevant external plan section at
   each continuation.
2. Compare current evidence against the active packet and completion checklist.
3. Add RED/negative tests first when implementation behavior is missing.
4. Implement only the smallest scoped runtime/validator/script/doc change needed
   by the active packet.
5. Run targeted verification, inspect logs, and record evidence.
6. Update `PLAN.md`, `ATTEMPTS.md`, `NOTES.md`, `CONTROL.md`, and `GOAL.md`
   progress tersely.
7. Before completion, perform requirement-by-requirement audit against the
   external plan and current repo state.
</workflow>

<working_memory>
Maintain:

- `GOAL.md`: canonical objective, acceptance, and progress.
- `PLAN.md`: packet matrix and current strategy.
- `ATTEMPTS.md`: completed attempts and evidence links.
- `NOTES.md`: durable discoveries and blockers.
- `CONTROL.md`: active packet, next actions, stop conditions, cache rules.
</working_memory>

<human_control_surface>
The user may edit `CONTROL.md` to pause, reorder packets, narrow final gates, or
record a strategic pivot. Re-read it before changing packet, final verdict, or
release claim posture.
</human_control_surface>

<verification_loop>
Core final commands, adjusted only when repo reality requires:

```bash
bash scripts/release/post_v0_4/actor-runtime-foundation-linux-x64-gate.sh --report-dir reports/actor-runtime-foundation/final
go test -buildvcs=false ./compiler/... ./cli/... ./tools/... -count=1
go test -race -buildvcs=false ./cli/internal/actornet ./compiler/internal/actorsrt ./compiler/internal/parallelrt ./compiler/internal/actorsafety -count=1
go run -buildvcs=false ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
go run -buildvcs=false ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
go run -buildvcs=false ./tools/cmd/validate-actor-runtime-foundation --report-dir reports/actor-runtime-foundation/final --current-git-head $(git rev-parse --verify HEAD)
git diff --check
git status --short
```

If any command fails, classify it as `FAIL`, `BLOCKED`, or `MISSING`, fix if in
scope, or record an explicit blocker/nonclaim.
</verification_loop>

<execution_rules>
- Check git status before edits.
- Preserve unrelated user changes.
- Prefer `rg` over `grep`.
- Use `apply_patch` for manual file edits.
- Read context files before implementation.
- Batch independent reads in parallel when safe.
- Run focused tests before broad tests.
- Do not paper over failures.
- Do not widen scope.
- Keep final answers concise and evidence-backed.
</execution_rules>

<output_contract>
Final answer after completion must include created/updated files, scoped
verdicts, main evidence, commands run with PASS/FAIL/BLOCKED, blockers if any,
and residual scoped nonclaims. Call `update_goal` complete only after every
acceptance item is proven by current evidence.
</output_contract>

## Progress

- 2026-06-09: `ACTOR-P00` completed. Evidence:
  `reports/actor-runtime-foundation/P00/truth-summary.md`,
  `reports/actor-runtime-foundation/P00/truth-summary.json`, and
  `reports/actor-runtime-foundation/P00/command-status.tsv`.
- 2026-06-10: `ACTOR-P01` completed as `PROTOTYPE_ONLY_NON_GOAL` for the
  multi-threaded actor scheduler path. Evidence:
  `reports/actor-runtime-foundation/P01/summary.md`,
  `reports/actor-runtime-foundation/P01/summary.json`,
  `reports/actor-runtime-foundation/P01/command-status.tsv`,
  `reports/actor-runtime-foundation/P01/green-scheduler-boundary.log`, and
  `reports/actor-runtime-foundation/P01/green-scheduler-validator-smoke.log`.
- 2026-06-09: `ACTOR-P02` completed. Evidence:
  `reports/actor-runtime-foundation/P02/summary.md`,
  `reports/actor-runtime-foundation/P02/summary.json`, and
  `reports/actor-runtime-foundation/P02/command-status.tsv`.
- 2026-06-09: `ACTOR-P03` completed. Evidence:
  `reports/actor-runtime-foundation/P03/summary.md`,
  `reports/actor-runtime-foundation/P03/summary.json`, and
  `reports/actor-runtime-foundation/P03/command-status.tsv`.
- 2026-06-10: `ACTOR-P04` completed. Evidence:
  `reports/actor-runtime-foundation/P04/summary.md`,
  `reports/actor-runtime-foundation/P04/summary.json`,
  `reports/actor-runtime-foundation/P04/command-status.tsv`,
  `reports/actor-runtime-foundation/P04/green-actor-island-transfer.log`, and
  `reports/actor-runtime-foundation/P04/green-zero-copy-transfer-guards.log`.
- 2026-06-09: `ACTOR-P05` completed. Evidence:
  `reports/actor-runtime-foundation/P05/summary.md`,
  `reports/actor-runtime-foundation/P05/summary.json`, and
  `reports/actor-runtime-foundation/P05/command-status.tsv`.
- 2026-06-09: `ACTOR-P06` completed. Evidence:
  `reports/actor-runtime-foundation/P06/summary.md`,
  `reports/actor-runtime-foundation/P06/summary.json`, and
  `reports/actor-runtime-foundation/P06/command-status.tsv`.
- 2026-06-09: `ACTOR-P07` completed. Evidence:
  `reports/actor-runtime-foundation/P07/summary.md`,
  `reports/actor-runtime-foundation/P07/summary.json`, and
  `reports/actor-runtime-foundation/P07/command-status.tsv`.
- 2026-06-09: `ACTOR-P08` completed. Evidence:
  `reports/actor-runtime-foundation/P08/summary.md`,
  `reports/actor-runtime-foundation/P08/summary.json`, and
  `reports/actor-runtime-foundation/P08/command-status.tsv`.
- 2026-06-09: `ACTOR-P09` completed. Evidence:
  `reports/actor-runtime-foundation/P09/summary.md`,
  `reports/actor-runtime-foundation/P09/summary.json`, and
  `reports/actor-runtime-foundation/P09/command-status.tsv`.
- 2026-06-09: `ACTOR-P10` completed. Evidence:
  `reports/actor-runtime-foundation/P10/summary.md`,
  `reports/actor-runtime-foundation/P10/summary.json`, and
  `reports/actor-runtime-foundation/P10/command-status.tsv`.
- 2026-06-09: `ACTOR-P11` completed. Evidence:
  `reports/actor-runtime-foundation/P11/summary.md`,
  `reports/actor-runtime-foundation/P11/summary.json`, and
  `reports/actor-runtime-foundation/P11/command-status.tsv`.
- 2026-06-09: `ACTOR-P12` completed. Evidence:
  `reports/actor-runtime-foundation/P12/summary.md`,
  `reports/actor-runtime-foundation/P12/summary.json`,
  `reports/actor-runtime-foundation/P12/command-status.tsv`, and
  `reports/actor-runtime-foundation/P12/actor-runtime-foundation-linux-x64-gate-final.log`.
- 2026-06-09: `ACTOR-P13` completed. Evidence:
  `reports/actor-runtime-foundation/P13/summary.md`,
  `reports/actor-runtime-foundation/P13/summary.json`, and
  `reports/actor-runtime-foundation/P13/command-status.tsv`.
- 2026-06-09: `ACTOR-P14` completed. Evidence:
  `reports/actor-runtime-foundation/P14/summary.md`,
  `reports/actor-runtime-foundation/P14/summary.json`, and
  `reports/actor-runtime-foundation/P14/command-status.tsv`.
- 2026-06-09: `ACTOR-P15` completed. Evidence:
  `reports/actor-runtime-foundation/P15/summary.md`,
  `reports/actor-runtime-foundation/P15/summary.json`,
  `reports/actor-runtime-foundation/P15/command-status.tsv`, and
  `reports/actor-runtime-foundation/P15/parallelrt-evidence.raw.json`.
- 2026-06-09: `ACTOR-P16` completed. Evidence:
  `reports/actor-runtime-foundation/P16/summary.md`,
  `reports/actor-runtime-foundation/P16/summary.json`,
  `reports/actor-runtime-foundation/P16/command-status.tsv`, and
  `reports/actor-runtime-foundation/P16/actor-runtime-source-sha256.txt`.
- 2026-06-10: `ACTOR-P17` completed. Evidence:
  `docs/audits/actor-runtime-production-foundation-final.md`,
  `reports/actor-runtime-foundation/P17/summary.md`,
  `reports/actor-runtime-foundation/P17/summary.json`,
  `reports/actor-runtime-foundation/P17/command-status.tsv`,
  `reports/actor-runtime-foundation/final/actor-runtime-foundation-manifest.json`,
  `reports/actor-runtime-foundation/final/artifact-hashes.json`,
  `reports/actor-runtime-foundation/P17/broad-compiler-cli-tools-refresh.log`,
  `reports/actor-runtime-foundation/P17/race-actor-slice-final.log`,
  `reports/actor-runtime-foundation/P17/git-diff-check-final.log`, and
  `reports/actor-runtime-foundation/P17/git-status-short-final.txt`.
  Verdict: `PROD_STABLE_SCOPED`; release-candidate and `PROD_READY_PROVEN` are
  not claimed because the worktree is dirty and remote CI/package publication
  were not run in this session.
