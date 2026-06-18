# RAM Contract Compiler

Status: current scoped report/gate contract.

The RAM Contract Compiler is the compiler-owned reporting layer that projects existing
MemoryFactGraph and AllocPlan evidence into a RAM-focused artifact set. It does not reconstruct
truth from JSON reports. The compiler emits `tetra.ram-contract-report.v1`,
`tetra.memory-grade-report.v1`, `tetra.proof-store-summary.v1`, and
`tetra.validation-pipeline-coverage.v1` from compiler-owned facts, then writes `heap-blockers.json`
and `copy-blockers.json` as explicit blocker indexes.

Release bundles add `ram-contract-release-manifest.json`,
`artifact-hashes.json`, and `fuzz/ram-contract-fuzz-oracle.json`. The release
validator checks required files, artifact hashes, manifest entries, git-head
consistency, proof references, fuzz mutation exit evidence, validation
pipeline entrypoint coverage, and cross-file heap/copy/grade consistency.

## Data Flow

- MemoryFactGraph records provenance, unsafe classification, storage class, and proof relationship
  facts.
- AllocPlan records heap, stack, region, static, copy, and boundedness decisions after lowering.
- ProofStore records stable proof references; stale hashes and unsafe_unknown promotion are
  rejected.
- The RAM contract report combines those sources into rows with bytes, grade, boundedness, proof
  references, and blocker explanations.
- `TETRA4100` is the diagnostic code for RAM contract enforcement failures.

## Artifacts

- `ram-contract-report.json`: `tetra.ram-contract-report.v1`.
- `memory-grade-report.json`: `tetra.memory-grade-report.v1`.
- `proof-store-summary.json`: `tetra.proof-store-summary.v1`.
- `validation-pipeline-coverage.json`: `tetra.validation-pipeline-coverage.v1`.
- `heap-blockers.json`: heap blocker rows.
- `copy-blockers.json`: copy blocker rows.
- `fuzz/ram-contract-fuzz-oracle.json`: deterministic fake-evidence mutation
  evidence with validator commands, non-zero exits, excerpts, and mutated file
  paths.
- `artifact-hashes.json`: hash manifest covering the release bundle.
- `ram-contract-release-manifest.json`: release command and artifact index.

## Enforcement

The compiler accepts `--emit-ram-contract-report`, `--fail-if-heap`,
`--fail-if-copy`, `--fail-if-unbounded`, `--memory-budget`, and
`--ram-contract`. The release gate is
`scripts/release/post_v0_4/ram-contract-linux-x64-smoke.sh`; it validates every
artifact before upload and rejects stale report directories.

## Nonclaims

- no zero heap for all programs claim
- no zero-copy for all programs claim
- no full formal proof claim
- no all-target RAM parity claim
- no production object memory claim
- no production persistent memory claim
- no performance claim
