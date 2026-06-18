# Master Plan Final Artifact Map

Status: dump-visible artifact map for
`reports/master-plan-final-20260602/closure.md`.

This map lists the evidence artifacts exported by P12.0. Logs under `reports/`
may be absent from dumps; this document keeps their path, command, status, and
claim boundary visible.

## Broad Gate Logs

### Compiler, CLI, and Tools

- Artifact:
  `reports/master-plan-final-20260602/go-test-compiler-cli-tools-final.log`
- Command or role: `go test ./compiler/... ./cli/... ./tools/... -count=1`
- Status: pass

### CI Test

- Artifact: `reports/master-plan-final-20260602/ci-test-final.log`
- Command or role: `bash scripts/ci/test.sh`
- Status: pass

### Quick Test-All Log

- Artifact: `reports/master-plan-final-20260602/test-all-quick.log`
- Command:

```bash
bash scripts/ci/test-all.sh \
  --quick \
  --keep-going \
  --report-dir reports/master-plan-final-20260602/test-all-quick
```

- Status: pass

### Quick Test-All Machine Summary

- Artifact: `reports/master-plan-final-20260602/test-all-quick/summary.json`
- Command or role: quick machine summary, `status=pass`, `failed_count=0`
- Status: pass

### Quick Test-All Human Summary

- Artifact: `reports/master-plan-final-20260602/test-all-quick/summary.md`
- Command or role: quick human summary
- Status: pass

### Verify Docs

- Artifact: `reports/master-plan-final-20260602/verify-docs.log`
- Command:

```bash
go run ./tools/cmd/verify-docs \
  --manifest docs/generated/manifest.json
```

- Status: pass

### Validate Manifest

- Artifact: `reports/master-plan-final-20260602/validate-manifest.log`
- Command:

```bash
go run ./tools/cmd/validate-manifest \
  --manifest docs/generated/manifest.json
```

- Status: pass

### Git Diff Check

- Artifact: `reports/master-plan-final-20260602/git-diff-check.log`
- Command or role: `git diff --check`
- Status: pass

### Graphify Update

- Artifact: `reports/master-plan-final-20260602/graphify-update.log`
- Command or role: `graphify update .`
- Status: pass

## Preserved Failure Evidence

### Quick Gate Before Formatter Fix

- Artifact:
  `reports/master-plan-final-20260602/test-all-quick-before-format-fix.log`
- Command or role: first quick gate before formatter repair
- Status: fail, preserved

### Pre-Fix Quick Gate Summary

- Artifact:
  `reports/master-plan-final-20260602/test-all-quick-before-format-fix/summary.json`
- Command or role: machine summary for the failed pre-fix quick gate
- Status: fail, preserved

### Formatter Check After Fix

- Artifact: `reports/master-plan-final-20260602/formatter-check-after-fix.log`
- Command or role: focused formatter rerun after repairing reported examples
- Status: pass

## Focused Gate Logs

### P2 Allocation Lowering

- Artifact:
  `reports/master-plan-final-20260602/focused-p2-allocation-lowering.log`
- Status: pass

### P3 Register Backend

- Artifact: `reports/master-plan-final-20260602/focused-p3-register-backend.log`
- Status: pass

### P4 Optimizer / Translation Validation

- Artifact:
  `reports/master-plan-final-20260602/focused-p4-optimizer-validation.log`
- Status: pass

### P5 Allocator / Runtime

- Artifact: `reports/master-plan-final-20260602/focused-p5-allocator-runtime.log`
- Status: pass

### P6 Actor Transfer

- Artifact: `reports/master-plan-final-20260602/focused-p6-actor-transfer.log`
- Status: pass

### P8 Truth-Bench Dry Run

- Artifact:
  `reports/master-plan-final-20260602/focused-p8-truth-bench-dry-run.log`
- Status: pass

### P11 Differential / Formalcore / Selfhostgate

- Artifact:
  `reports/master-plan-final-20260602/focused-p11-differential-formal-selfhost.log`
- Status: pass

## Benchmark Dry-Run Artifacts

### Dry-Run Manifest

- Artifact: `reports/master-plan-final-20260602/p8-dry-run-manifest.json`
- Role: truth-benchmark manifest for the dry-run matrix
- Status: pass

### Dry-Run Report

- Artifact: `reports/master-plan-final-20260602/p8-dry-run-report.json`
- Role: `tetra.truth.benchmark.v1` dry-run report
- Status: pass

### Dry-Run Summary

- Artifact: `reports/master-plan-final-20260602/p8-dry-run-summary.txt`
- Role: human summary for the dry-run report
- Status: pass

### Dry-Run Fixtures

- Artifact: `reports/master-plan-final-20260602/p8-dry-run-fixtures/`
- Role: 14 benchmark categories x 4 languages plus Tetra report fixtures
- Status: pass

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

### Final Audit

- Artifact: `docs/audits/master-plan-final-20260602.md`
- Role: dump-visible closure summary and classification table.

### Artifact Map

- Artifact: `docs/audits/master-plan-final-20260602-artifact-map.md`
- Role: dump-visible command, log, artifact, pass/fail, and non-claim map.
