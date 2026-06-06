# MEM-RELEASE-013 Memory Evidence Freeze Goal

<goal>
Implement **MEM-RELEASE-013: Memory Evidence Freeze and Dirty Worktree
Triage**.

Completion means v12 / `MEM-FUZZ-012` remains accepted as slice-level
`validated_narrow`, and the v0-v12 memory evidence chain is frozen into a
release-grade evidence packet with every `git status --short` entry classified.

This is a release/evidence hygiene slice, not a new memory semantics slice. It
must not claim clean release unless the worktree is clean or every dirty entry
has explicit triage evidence, and it must not claim "Memory 100%".
</goal>

<context>
Read first on every continuation:

- `AGENTS.md`
- `GOAL.md`
- `PLAN.md`
- `ATTEMPTS.md`
- `NOTES.md`
- `CONTROL.md`
- Graphify MCP context for memory evidence, release gates, dirty worktree
  policy, `MemoryFactGraph`, memory report projection, memory fuzz oracle, and
  correlation validators
- `graphify-out/GRAPH_REPORT.md`
- `.workflow/memory-ideal-vertical-slice-v12-fuzz-oracle/final-report.md`
- `reports/memory-fuzz-short/v12/memory-fuzz-oracle.json`
- `reports/memory-fuzz-short/v12/summary.md`
- `docs/audits/memory-fuzz-oracle-v1.md`
- `docs/testing/fuzz_property_stress.md`
- `docs/design/memory_production_core_v1.md`
- `docs/spec/memory_report_schema_v1.md`
- `docs/generated/manifest.json`
- `tools/cmd/memory-fuzz-short`
- `tools/cmd/validate-memory-fuzz-oracle`
- `tools/cmd/validate-memory-correlation`
- `tools/cmd/validate-manifest`
- `tools/cmd/verify-docs`

Current baseline:

- v12 / `MEM-FUZZ-012` is accepted as `accepted` and slice-level
  `validated_narrow`.
- v12 adds deterministic Tier 1 memory fuzz oracle release evidence for v0-v11
  cross-slice coverage.
- v12 does not prove exhaustive fuzz safety, arbitrary unsafe safety, full
  runtime/ABI/target parity, performance, clean release, replacement for
  `MemoryFactGraph`, or "Memory 100%".
- Current `git status --short` is non-empty. This is the repeated release
  blocker that v13 must classify without destructive cleanup.
</context>

<constraints>
- Always communicate with the user in Ukrainian.
- Keep scope to `MEM-RELEASE-001` through `MEM-RELEASE-005`.
- In scope:
  - collect and classify `git status --short`;
  - separate memory-slice-owned, release-owned, unrelated, and blocker entries;
  - produce a v0-v12 memory evidence packet;
  - regenerate and validate v12-style Tier 1 fuzz artifacts under v13;
  - run v0-v12 validators from the frozen evidence state;
  - add or update release summary lint evidence that rejects broad claims.
- Non-scope:
  - new memory semantics;
  - runtime/ABI proof;
  - target parity;
  - performance claim;
  - destructive cleanup;
  - automatic revert/delete of unrelated dirty files;
  - "Memory 100%".
- `MemoryFactGraph` remains truth. Reports and generated oracle artifacts are
  evidence/projections, not replacements for graph/report/compiler validators.
- Do not delete, revert, move, or archive unrelated dirty/untracked files
  automatically.
- If a status entry is unrelated or ambiguous, record it as a blocker or human
  decision item instead of cleaning it.
- Clean-release claim requires either clean `git status --short` or explicit
  triage for every dirty entry.
- Preserve the existing v12 decision: `accepted`, `validated_narrow`, and
  `proceed_with_blockers` while dirty state remains.
- Use persistent Go caches under `.cache/` or `$HOME/.cache`; never set
  `GOCACHE` to `/tmp`.
- If code files change, run `graphify update .` before completion.
</constraints>

<scorecard>
Primary metric: all five `MEM-RELEASE-*` requirements have evidence artifacts,
validator/check evidence, final report rows, and explicit nonclaims.

Passing threshold:

- `MEM-RELEASE-001`: v0-v12 evidence packet lists every memory artifact and
  validator gate.
- `MEM-RELEASE-002`: every `git status --short` entry is classified as
  `memory_owned`, `release_owned`, `unrelated`, or `blocker`.
- `MEM-RELEASE-003`: clean-release claim is rejected unless status is clean or
  every dirty entry has explicit triage.
- `MEM-RELEASE-004`: v13 Tier 1 fuzz artifacts are regenerated and validated
  from the frozen evidence state.
- `MEM-RELEASE-005`: release summary lint rejects broad "Memory 100%", target
  parity, performance, arbitrary unsafe, and clean-release-over-dirty claims.

Scoring command or inspection path:

- inspect `reports/memory-release-v13/git-status-short.txt`;
- inspect `reports/memory-release-v13/triage.md`;
- inspect `reports/memory-release-v13/evidence-packet.md`;
- inspect `reports/memory-release-v13/release-summary-lint.md` or the validator
  artifact that replaces it;
- inspect `reports/memory-fuzz-short/v13/memory-fuzz-oracle.json`;
- inspect `.workflow/memory-release-v13/final-report.md`;
- run the acceptance gates in `<verification_loop>`.

Regression checks:

- v0-v12 memory correlation regression;
- v13 fuzz oracle generation and validation;
- manifest/docs gates;
- broad `go test`;
- canonical `scripts/ci/test.sh`;
- `git diff --check`;
- `graphify update .`.

Stop condition:

- Stop and record a blocker if v13 requires broad fuzz runtime, target parity,
  arbitrary unsafe proof, long nightly run as mandatory Tier 1, destructive
  cleanup, or automatic resolution of unrelated dirty worktree entries.
- Stop after the same focused or full gate fails twice for the same reason
  without new evidence.
</scorecard>

<done_when>
The goal is complete only when all are true:

- `reports/memory-release-v13/git-status-short.txt` exists and records the
  exact `git status --short` output used for the freeze.
- `reports/memory-release-v13/triage.md` classifies every status entry as
  `memory_owned`, `release_owned`, `unrelated`, or `blocker`, with a proposed
  human decision for every unrelated or ambiguous entry.
- `reports/memory-release-v13/evidence-packet.md` lists v0-v12 memory artifacts,
  validator gates, generated reports, and final audit documents.
- `.workflow/memory-release-v13/final-report.md` records accepted/rejected or
  blocker classification for `MEM-RELEASE-001` through `MEM-RELEASE-005`.
- v13 Tier 1 artifact generation passes:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v13-release-fuzz go run ./tools/cmd/memory-fuzz-short --tier=1 --report-dir reports/memory-fuzz-short/v13`.
- v13 Tier 1 artifact validation passes:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v13-release-fuzz-validate go run ./tools/cmd/validate-memory-fuzz-oracle --report reports/memory-fuzz-short/v13/memory-fuzz-oracle.json`.
- v0-v12 correlation regression passes:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v13-release-regression bash -lc 'for f in docs/audits/memory-ideal-vslice-v*-correlation.md; do go run ./tools/cmd/validate-memory-correlation --file "$f"; done'`.
- Docs/manifest gates pass:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v13-release-manifest go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`;
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v13-release-docs go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`.
- Broad gates pass:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v13-release-broad go test ./compiler/... ./cli/... ./tools/... -count=1`;
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v13-release-ci bash scripts/ci/test.sh`.
- Hygiene gates are recorded:
  `git diff --check`;
  `git status --short`;
  `graphify update .`.
- Final report repeats nonclaims: no new memory semantics, no runtime/ABI
  proof, no target parity, no performance claim, no destructive cleanup, no
  arbitrary unsafe proof, no clean-release claim over dirty state, and no
  "Memory 100%".
</done_when>

<feedback_loop>
Fast iterative check:

```bash
git status --short > reports/memory-release-v13/git-status-short.txt
```

Expected runtime: immediate.
Cadence: run at the start of every v13 iteration and before final report.
Proxy validity: the primary blocker is dirty worktree state; status capture is
the fastest representative check for the freeze/triage slice.

Focused validation after evidence artifacts change:

```bash
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v13-release-fuzz go run ./tools/cmd/memory-fuzz-short --tier=1 --report-dir reports/memory-fuzz-short/v13
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v13-release-fuzz-validate go run ./tools/cmd/validate-memory-fuzz-oracle --report reports/memory-fuzz-short/v13/memory-fuzz-oracle.json
```

Slower escalation checks: correlation regression, docs/manifest, broad
`go test`, CI, `git diff --check`, and Graphify update.
</feedback_loop>

<workflow>
1. Re-read `GOAL.md`, `CONTROL.md`, and `git status --short`.
2. Use Graphify MCP first for release/memory evidence relationships, then
   verify concrete files with `rg`, `sed`, and commands.
3. Capture `git status --short` into
   `reports/memory-release-v13/git-status-short.txt`.
4. Classify every dirty/untracked entry in
   `reports/memory-release-v13/triage.md`.
5. Build `reports/memory-release-v13/evidence-packet.md` for v0-v12 artifacts,
   validators, reports, final audits, and nonclaims.
6. Regenerate and validate v13 Tier 1 fuzz artifacts.
7. Add release-summary lint evidence or a validator-backed equivalent for
   broad-claim rejection.
8. Run focused gates, then broad gates, hygiene checks, dirty-worktree record,
   and Graphify update.
9. Write `.workflow/memory-release-v13/final-report.md`.
10. Complete only if every `done_when` item has evidence. If unrelated or
    ambiguous dirty entries remain unresolved, keep the release/worktree
    decision as `proceed_with_blockers`.
</workflow>

<working_memory>
Maintain:

- `PLAN.md`: current v13 strategy, phase checklist, open decisions, and current
  iteration.
- `ATTEMPTS.md`: every meaningful evidence capture, validation attempt, result,
  and next adjustment.
- `NOTES.md`: durable discoveries, baseline caveats, blocker rationale, and
  nonclaims.
- `CONTROL.md`: compact human control surface for v13.
- `.workflow/memory-release-v13/`: workflow-local plan, attempts, notes,
  control, and final report.

Update `GOAL.md ## Progress` before each meaningful action batch with concise
state transitions and evidence references.
</working_memory>

<human_control_surface>
Maintain `CONTROL.md` as the operator panel for this goal.

Before each phase change, expensive step, strategic pivot, or status-entry
decision, reread `CONTROL.md`. If it changed, summarize the relevant change in
`PLAN.md` and adapt before proceeding.

`CONTROL.md` may narrow scope, pause work, or require approval for specific
dirty entries, but it cannot silently weaken `done_when`, nonclaims, or the
clean-release rule.
</human_control_surface>

<verification_loop>
Run these acceptance gates in order unless an earlier gate fails:

```bash
git status --short > reports/memory-release-v13/git-status-short.txt
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v13-release-regression bash -lc 'for f in docs/audits/memory-ideal-vslice-v*-correlation.md; do go run ./tools/cmd/validate-memory-correlation --file "$f"; done'
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v13-release-fuzz go run ./tools/cmd/memory-fuzz-short --tier=1 --report-dir reports/memory-fuzz-short/v13
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v13-release-fuzz-validate go run ./tools/cmd/validate-memory-fuzz-oracle --report reports/memory-fuzz-short/v13/memory-fuzz-oracle.json
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v13-release-manifest go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v13-release-docs go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v13-release-broad go test ./compiler/... ./cli/... ./tools/... -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v13-release-ci bash scripts/ci/test.sh
git diff --check
graphify update .
```

If a check cannot run, record the exact reason and strongest substitute
evidence in `ATTEMPTS.md` and the final report.
</verification_loop>

<execution_rules>
- Check git status before edits.
- Preserve unrelated user changes.
- Prefer `rg` over `grep`.
- Use `apply_patch` for manual file edits.
- Read context files before implementation.
- Batch independent file reads in parallel when possible.
- Keep the scorecard current.
- Use the fastest representative feedback check while iterating; reserve
  slower checks for escalation points and final verification.
- Update `PLAN.md`, `ATTEMPTS.md`, `NOTES.md`, and `GOAL.md ## Progress` with
  concise evidence as work proceeds.
- Run focused checks before broad checks.
- Do not paper over failures.
- Do not widen scope.
- Never set `GOCACHE` to `/tmp`; use persistent `.cache/` or `$HOME/.cache`
  caches.
- Run `graphify update .` after code changes and before completion when the
  acceptance gate requires it.
- Keep the final answer concise and Ukrainian.
</execution_rules>

<output_contract>
Final artifacts:

- `reports/memory-release-v13/git-status-short.txt`;
- `reports/memory-release-v13/triage.md`;
- `reports/memory-release-v13/evidence-packet.md`;
- `reports/memory-release-v13/release-summary-lint.md` or validator-backed
  equivalent evidence;
- `reports/memory-fuzz-short/v13/memory-fuzz-oracle.json`;
- `reports/memory-fuzz-short/v13/summary.md`;
- `.workflow/memory-release-v13/final-report.md`;
- command evidence in `ATTEMPTS.md` and workflow attempts;
- dirty-worktree caveat in final response.

Call `update_goal complete` only after every `done_when` item is proven with
evidence. Do not mark complete for partial progress or budget exhaustion.
</output_contract>

## Progress

- 2026-06-06T11:00:00Z: v12 baseline accepted as `accepted` /
  `validated_narrow`; release/worktree remains `proceed_with_blockers` because
  `git status --short` is non-empty.
- 2026-06-06T11:00:00Z: Bridge: preparing `MEM-RELEASE-013` freeze/triage
  setup; this feeds `MEM-RELEASE-001` through `MEM-RELEASE-005` and the initial
  status-capture gate.
- 2026-06-06T14:22:44Z: Active goal created for `MEM-RELEASE-013`. Bridge:
  next action is status capture into
  `reports/memory-release-v13/git-status-short.txt`, feeding
  `MEM-RELEASE-001` through `MEM-RELEASE-003`.
- 2026-06-06T14:24:35Z: Bridge: capturing current `git status --short` into
  `reports/memory-release-v13/git-status-short.txt`; output will feed
  `reports/memory-release-v13/triage.md` and the clean-release blocker
  decision.
- 2026-06-06T14:24:35Z: Status freeze captured and triaged:
  `reports/memory-release-v13/git-status-short.txt` has 36 entries, and
  `reports/memory-release-v13/triage.md` classifies 29 `memory_owned`, 6
  `release_owned`, and 1 `unrelated`. Bridge: build v0-v12 evidence packet and
  broad-claim lint artifact.
- 2026-06-06T14:24:35Z: Evidence packet and lint artifact written:
  `reports/memory-release-v13/evidence-packet.md` and
  `reports/memory-release-v13/release-summary-lint.md`. Bridge: regenerate and
  validate `reports/memory-fuzz-short/v13/` for `MEM-RELEASE-004`.
- 2026-06-06T14:28:00Z: v13 Tier 1 fuzz artifacts generated and validated:
  `reports/memory-fuzz-short/v13/memory-fuzz-oracle.json` and
  `reports/memory-fuzz-short/v13/summary.md`; summary records 5
  `MEM-FUZZ-*` rows, 12 v0-v11 coverage rows, and 12 Tier 1 cases. Bridge:
  run correlation and docs/manifest gates.
- 2026-06-06T14:28:00Z: Correlation regression plus manifest/docs gates exited
  0 using the v13 release cache paths. Bridge: run broad Go gate and canonical
  CI.
- 2026-06-06T14:31:00Z: Broad Go gate failed in `tools/scriptstest` because
  `README.md` lacked exact marker `Tetra Language (v0.4.0)`. Bridge: apply
  minimal README marker fix and rerun targeted test plus broad gate.
- 2026-06-06T14:31:00Z: Targeted
  `TestCurrentSupportedSurfaceDocumentIsReleaseAligned` exited 0 after README
  marker fix. Bridge: rerun broad Go gate.
- 2026-06-06T14:34:00Z: Broad Go gate exited 0 after README marker fix; final
  output included `ok tetra_language/tools/scriptstest 78.647s`. Bridge: run
  canonical `scripts/ci/test.sh`.
- 2026-06-06T14:37:00Z: Canonical CI exited 0, ended `OK`, and emitted artifact
  `tetra.release.v0_4_0.go-test-suite.v1`. Bridge: re-capture dirty status
  after README marker fix, then run hygiene and Graphify gates.
- 2026-06-06T14:37:00Z: Refreshed status freeze now has 37 entries and triage
  classifies 29 `memory_owned`, 7 `release_owned`, and 1 `unrelated`;
  clean-release remains blocked. Bridge: run `git diff --check` and
  `graphify update .`.
- 2026-06-06T14:40:00Z: Hygiene/Graphify gates passed: `git diff --check`
  exited 0; `graphify update .` rebuilt `21427 nodes`, `66887 edges`, `1185`
  communities. Final report written at
  `.workflow/memory-release-v13/final-report.md`. Bridge: run completion audit
  against all `done_when` items.
- 2026-06-06T14:40:00Z: Completion audit passed: required artifacts exist,
  triage covers all 37 status rows, current status matches
  `reports/memory-release-v13/git-status-short.txt`, final
  `MEM-RELEASE-*` rows are recorded, and fresh `git diff --check` exited 0.
  Ready to mark active goal complete.
