# P4 Verification

Objective: run all required focused and full gates, collect evidence, refresh
Graphify, and write final workflow report.

Ownership:
- `.workflow/memory-ideal-vertical-slice-v2/results/P4-verification.md`
- `.workflow/memory-ideal-vertical-slice-v2/final-report.md`
- `GOAL.md` progress section

Do:
- Run the exact verification commands in `GOAL.md` with persistent caches.
- Run `git diff --check`.
- Run `graphify update .` after code changes.
- Run workflow helper scripts if available.
- Record accepted/rejected/conflict decisions.

Do not:
- Mark complete with failing gates.
- Hide unrelated failures.

Expected output: final evidence and completion decision.
