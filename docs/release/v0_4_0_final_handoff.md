# Tetra v0.4.0 Final Handoff

Status: final Linux-x64 production handoff for the scoped `v0.4.0` release.

This handoff records the scoped `v0.4.0` Linux-x64/no-EcoNet release candidate.
The canonical gate requires memory, parallelism, and compiler production-core
artifacts as part of the final release contract.

## Release Truth

- Current local version metadata: `v0.4.0`
- Requested target profile: Linux-x64 production only
- Explicitly excluded: EcoNet, non-Linux target runtimes, WASM target runtimes,
  and full v1.0 language guarantees
- Release status: final clean candidate validated by the canonical
  `--require-clean` gate
- Final candidate branch: `codex/v0.4.0-final-production`
- Current supported surface: `docs/spec/current_supported_surface.md`
- Selected `v0.4.0` scope: `docs/spec/v0_4_scope.md`
- Completion audit: `docs/release/v0_4_0_completion_audit.md`
- Gap matrix: `docs/release/v0_4_0_prod_gap_matrix.json`
- Scope decisions: `docs/release/v0_4_0_scope_decisions.json`
- Gate checklist: `docs/checklists/v0_4_0_release_gate.md`
- Release notes: `docs/release-notes/v0_4_0.md`

## Current Evidence

The scoped readiness preflight passes with the repository evidence:

```sh
go run ./tools/cmd/validate-v0-4-readiness \
  --expected-version v0.4.0 \
  --features reports/v0.4.0/features.json \
  --targets reports/v0.4.0/targets.json \
  --manifest docs/generated/manifest.json \
  --scope-decisions docs/release/v0_4_0_scope_decisions.json
```

Linux-x64 smoke evidence:

- `reports/v0.4.0/linux-host-smoke.json`
- `reports/v0.4.0/distributed-actors-linux-x64.json`
- `reports/v0.4.0/native-ui-linux-x64.json`

Historical clean scoped gate evidence from the older gate:

- `reports/v0.4.0/release-gate-clean/summary.json`
- `reports/v0.4.0/release-gate-clean/artifact-hashes.json`
- `reports/v0.4.0/release-gate-clean/artifacts/release-state.json`

That gate report validates as `tetra.release.v0_4_0.gate-report.v1`, has
`status: pass`, `failed_count: 0`, and a release-state artifact with
`git.clean: true`; it remains historical evidence for the earlier gate shape.

## Final Evidence

The clean final candidate is proven by this canonical gate command:

```sh
./tetra version
./t version
./tetra features --format=json
./tetra targets --format=json
go test ./compiler/... ./cli/... ./tools/... -count=1
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
go run ./cli/cmd/tetra smoke --target linux-x64 --run=true --report reports/v0.4.0/linux-host-smoke.json
bash scripts/release/post_v0_4/memory-production-linux-x64-smoke.sh --report-dir reports/v0.4.0
go run ./tools/cmd/validate-memory-production --report reports/v0.4.0/memory-production-linux-x64.json
bash scripts/release/post_v0_4/parallel-production-linux-x64-smoke.sh --report-dir reports/v0.4.0
go run ./tools/cmd/validate-parallel-production --report reports/v0.4.0/parallel-production-linux-x64.json
bash scripts/release/post_v0_4/compiler-production-linux-x64-smoke.sh --report-dir reports/v0.4.0
go run ./tools/cmd/validate-compiler-production --report reports/v0.4.0/compiler-production-linux-x64.json
bash scripts/release/v0_4_0/distributed-actors-linux-x64-smoke.sh --report-dir reports/v0.4.0
bash scripts/release/v0_4_0/native-ui-linux-x64-smoke.sh --report-dir reports/v0.4.0
go run ./tools/cmd/validate-v0-4-readiness --features reports/v0.4.0/features.json --targets reports/v0.4.0/targets.json --manifest docs/generated/manifest.json --scope-decisions docs/release/v0_4_0_scope_decisions.json
bash scripts/release/v0_4_0/security-review.sh --signoff <security-review.md>
git diff --check
git status --porcelain --untracked-files=all
bash scripts/release/v0_4_0/gate.sh --report-dir /tmp/tetra-v0.4.0-final-production-gate --require-clean
go run ./tools/cmd/validate-artifact-hashes --manifest /tmp/tetra-v0.4.0-final-production-gate/artifact-hashes.json
```

## Required Implementation Closures

The following scoped closures are selected for `v0.4.0` production:

- Callable Level 1, Callable Level 2, and selected first-class callable model.
- Local/control-flow lifetime SSA ownership/resource behavior.
- Linux-x64 Memory Production Core.
- Linux-x64 Parallelism Production Core.
- Linux-x64 Compiler Production Core.
- Standard-library compatibility mirror policy.
- UI metadata v1 for the selected Linux-x64 UI/native shell surface.
- Linux-x64 distributed actor runtime behavior.
- Linux-x64 native UI runtime behavior.
- Linux-x64 host-native runtime smoke behavior.

The following are not required for this scoped `v0.4.0` release:

- EcoNet / distributed Eco / hosted production TetraHub networking.
- Full v1.0 language guarantees.
- WASI/Web production runtime behavior.
- Windows or macOS production runtime behavior.

## Tag-Ready Rule

The scoped release candidate is tag-ready when the clean candidate branch runs
`scripts/release/v0_4_0/gate.sh --require-clean` and the resulting report
contains memory, parallelism, and compiler production artifacts plus a passing
release-state artifact.
