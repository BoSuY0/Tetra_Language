# P-C Missing / Unverifiable Referenced Paths

Status: completed read-only sub-agent audit.

Scope:

- 430 findings
- 224 unique referenced paths
- 47 source documents

Live checkout classification:

- 381 references point to `untracked_ignored` paths, mostly under ignored `reports/`.
- 48 references are currently `missing`.
- 1 reference is `tracked_exists` (`graphify-out/graph.json`), likely caused by dump snapshot mismatch.

Top groups:

- `reports/plan250/wave2-implD-evidence.md`: 17 findings, ignored.
- `reports/plan250/waveB-impl4-evidence.md`: 13 findings, ignored.
- `reports/plan250/waveA-impl3-evidence.md`: 12 findings, ignored.
- `docs/plans/2026-04-29-v1_0-250-task-master-plan.md`: 180 source references.
- `docs/release/v0_2_0_final_handoff.md`: 48 source references.

Recommended integration:

- Create a tracked triage/evidence policy rather than generating hundreds of ignored report files.
- For `missing` paths, either fix references, add tracked stable evidence, or mark as external with hash/URL metadata.
- Add a validator gate later: referenced evidence must exist, be ignored-with-manifest, or be external/archived with hash metadata.

Uncertainties:

- Some paths may be templates or directories, not concrete files.
- Ignored-path existence is not enough proof without hash/producer metadata.
