# Master Plan Final Artifact Map

Status: dump-visible artifact map for
`reports/master-plan-final-20260602/closure.md`.

This map lists the evidence artifacts exported by P12.0. Logs under `reports/`
may be absent from dumps; this document keeps their path, command, status, and
claim boundary visible.

## Broad Gate Logs

### Compiler, CLI, and tools test log

- Artifact: `reports/master-plan-final-20260602/go-test-compiler-cli-tools-final.log`
- Command:

```sh
go test ./compiler/... ./cli/... ./tools/... -count=1
```

- Status: pass.

### CI test log

- Artifact: `reports/master-plan-final-20260602/ci-test-final.log`
- Command:

```sh
bash scripts/ci/test.sh
```

- Status: pass.

### Quick test-all log

- Artifact: `reports/master-plan-final-20260602/test-all-quick.log`
- Command:

```sh
bash scripts/ci/test-all.sh \
  --quick \
  --keep-going \
  --report-dir reports/master-plan-final-20260602/test-all-quick
```

- Status: pass.

### Quick test-all machine summary

- Artifact: `reports/master-plan-final-20260602/test-all-quick/summary.json`
- Role:
  quick test-all machine summary, `status=pass`, `failed_count=0`.
- Status: pass.

### Quick test-all human summary

- Artifact: `reports/master-plan-final-20260602/test-all-quick/summary.md`
- Role: quick test-all human summary.
- Status: pass.

### Docs verification log

- Artifact: `reports/master-plan-final-20260602/verify-docs.log`
- Command:

```sh
go run ./tools/cmd/verify-docs \
  --manifest docs/generated/manifest.json
```

- Status: pass.

### Manifest validation log

- Artifact: `reports/master-plan-final-20260602/validate-manifest.log`
- Command:

```sh
go run ./tools/cmd/validate-manifest \
  --manifest docs/generated/manifest.json
```

- Status: pass.

### Git diff check log

- Artifact: `reports/master-plan-final-20260602/git-diff-check.log`
- Command: `git diff --check`.
- Status: pass.

### Graphify update log

- Artifact: `reports/master-plan-final-20260602/graphify-update.log`
- Command: `graphify update .`.
- Status: pass.

## Preserved Failure Evidence

### Quick gate before formatter repair

- Artifact:
  `reports/master-plan-final-20260602/test-all-quick-before-format-fix.log`
- Role: first quick gate before formatter repair.
- Status: fail, preserved.

### Failed quick gate machine summary

- Artifact:
  `reports/master-plan-final-20260602/test-all-quick-before-format-fix/summary.json`
- Role: machine summary for the failed pre-fix quick gate.
- Status: fail, preserved.

### Formatter check after fix

- Artifact: `reports/master-plan-final-20260602/formatter-check-after-fix.log`
- Role: focused formatter rerun after repairing the reported examples.
- Status: pass.

## Focused Gate Logs

### P2 allocation lowering

- Artifact:
  `reports/master-plan-final-20260602/focused-p2-allocation-lowering.log`
- Status: pass.

### P3 register backend

- Artifact: `reports/master-plan-final-20260602/focused-p3-register-backend.log`
- Status: pass.

### P4 optimizer / translation validation

- Artifact:
  `reports/master-plan-final-20260602/focused-p4-optimizer-validation.log`
- Status: pass.

### P5 allocator / runtime

- Artifact:
  `reports/master-plan-final-20260602/focused-p5-allocator-runtime.log`
- Status: pass.

### P6 actor transfer

- Artifact: `reports/master-plan-final-20260602/focused-p6-actor-transfer.log`
- Status: pass.

### P8 truth-bench dry run

- Artifact:
  `reports/master-plan-final-20260602/focused-p8-truth-bench-dry-run.log`
- Status: pass.

### P11 differential / formalcore / selfhostgate

- Artifact:
  `reports/master-plan-final-20260602/focused-p11-differential-formal-selfhost.log`
- Status: pass.

## Benchmark Dry-Run Artifacts

### Dry-run manifest

- Artifact: `reports/master-plan-final-20260602/p8-dry-run-manifest.json`
- Role: truth-benchmark manifest for the dry-run matrix.
- Status: pass.

### Dry-run report

- Artifact: `reports/master-plan-final-20260602/p8-dry-run-report.json`
- Role: `tetra.truth.benchmark.v1` dry-run report.
- Status: pass.

### Dry-run summary

- Artifact: `reports/master-plan-final-20260602/p8-dry-run-summary.txt`
- Role: human summary for the dry-run report.
- Status: pass.

### Dry-run fixtures

- Artifact: `reports/master-plan-final-20260602/p8-dry-run-fixtures/`
- Role:
  14 benchmark categories x 4 languages plus Tetra proof, bounds, and
  allocation report fixtures.
- Status: pass.

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

### Closure summary audit

- Artifact: `docs/audits/master-plan/master-plan-final-20260602.md`
- Role: dump-visible closure summary and classification table.

### Artifact map audit

- Artifact:
  `docs/audits/master-plan/master-plan-final-20260602-artifact-map.md`
- Role: dump-visible command, log, artifact, pass/fail, and non-claim map.
