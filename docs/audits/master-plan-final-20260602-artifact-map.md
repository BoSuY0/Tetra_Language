# Master Plan Final Artifact Map

Status: dump-visible artifact map for
`reports/master-plan-final-20260602/closure.md`.

This map lists the evidence artifacts exported by P12.0. Logs under `reports/`
may be absent from dumps; this document keeps their path, command, status, and
claim boundary visible.

## Broad Gate Logs

| Artifact | Command or role | Status |
|---|---|---:|
| `reports/master-plan-final-20260602/go-test-compiler-cli-tools-final.log` | `go test ./compiler/... ./cli/... ./tools/... -count=1` | pass |
| `reports/master-plan-final-20260602/ci-test-final.log` | `bash scripts/ci/test.sh` | pass |
| `reports/master-plan-final-20260602/test-all-quick.log` | `bash scripts/ci/test-all.sh --quick --keep-going --report-dir reports/master-plan-final-20260602/test-all-quick` | pass |
| `reports/master-plan-final-20260602/test-all-quick/summary.json` | Quick test-all machine summary, `status=pass`, `failed_count=0` | pass |
| `reports/master-plan-final-20260602/test-all-quick/summary.md` | Quick test-all human summary | pass |
| `reports/master-plan-final-20260602/verify-docs.log` | `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json` | pass |
| `reports/master-plan-final-20260602/validate-manifest.log` | `go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json` | pass |
| `reports/master-plan-final-20260602/git-diff-check.log` | `git diff --check` | pass |
| `reports/master-plan-final-20260602/graphify-update.log` | `graphify update .` | pass |

## Preserved Failure Evidence

| Artifact | Command or role | Status |
|---|---|---:|
| `reports/master-plan-final-20260602/test-all-quick-before-format-fix.log` | First quick gate before formatter repair | fail, preserved |
| `reports/master-plan-final-20260602/test-all-quick-before-format-fix/summary.json` | Machine summary for the failed pre-fix quick gate | fail, preserved |
| `reports/master-plan-final-20260602/formatter-check-after-fix.log` | Focused formatter rerun after repairing the reported examples | pass |

## Focused Gate Logs

| Plan area | Artifact | Status |
|---|---|---:|
| P2 allocation lowering | `reports/master-plan-final-20260602/focused-p2-allocation-lowering.log` | pass |
| P3 register backend | `reports/master-plan-final-20260602/focused-p3-register-backend.log` | pass |
| P4 optimizer / translation validation | `reports/master-plan-final-20260602/focused-p4-optimizer-validation.log` | pass |
| P5 allocator / runtime | `reports/master-plan-final-20260602/focused-p5-allocator-runtime.log` | pass |
| P6 actor transfer | `reports/master-plan-final-20260602/focused-p6-actor-transfer.log` | pass |
| P8 truth-bench dry run | `reports/master-plan-final-20260602/focused-p8-truth-bench-dry-run.log` | pass |
| P11 differential / formalcore / selfhostgate | `reports/master-plan-final-20260602/focused-p11-differential-formal-selfhost.log` | pass |

## Benchmark Dry-Run Artifacts

| Artifact | Role | Status |
|---|---|---:|
| `reports/master-plan-final-20260602/p8-dry-run-manifest.json` | Truth-benchmark manifest for the dry-run matrix | pass |
| `reports/master-plan-final-20260602/p8-dry-run-report.json` | `tetra.truth.benchmark.v1` dry-run report | pass |
| `reports/master-plan-final-20260602/p8-dry-run-summary.txt` | Human summary for the dry-run report | pass |
| `reports/master-plan-final-20260602/p8-dry-run-fixtures/` | 14 benchmark categories x 4 languages plus Tetra proof/bounds/allocation report fixtures | pass |

The P8 dry run validates harness shape and claim discipline only. It does not
claim real benchmark speedups or an official TechEmpower result.

## Non-Claims Boundaries

The artifacts above do not claim:

- fastest language status;
- official TechEmpower publication;
- full formal proof;
- self-hosting;
- full production actor scheduler/runtime;
- public semantic backend selection;
- unsafe fast mode or disabled safe checks.

## Dump-Visible Audit Docs

| Artifact | Role |
|---|---|
| `docs/audits/master-plan-final-20260602.md` | Dump-visible closure summary and classification table. |
| `docs/audits/master-plan-final-20260602-artifact-map.md` | Dump-visible command, log, artifact, pass/fail, and non-claim map. |
