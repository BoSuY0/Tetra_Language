# P-B Generated Release / Dump / Binary Evidence

Status: completed read-only sub-agent audit.

Covered F-IDs:

- `F-0006..F-0008`
- `F-0009..F-0023`
- `F-0024..F-0028`

Accepted classifications:

- `F-0006..F-0008`: dump-only limitations; cannot be fixed from live checkout without the original dump generation source or a new atomic dump.
- `F-0009..F-0023`: live binary artifacts exist under `docs/generated/v1_0/...` and are non-empty; the finding is about text-dump omission/unverifiability, not absence in this checkout.
- `F-0024..F-0028`: live issue or deliberate historical snapshot issue in `docs/generated/v1_0/release-state.*` and `release_gate_summary.json`; `docs/generated/v1_0/README.md` says this directory is a mixed compatibility workspace and not a final `v1.0.0` release archive.

Risks:

- Regenerating `docs/generated/v1_0` may overwrite intentionally historical compatibility evidence.
- `release_gate_summary.json` contains old version data and `/tmp/...` report paths, so it is not reproducible current proof.

Recommended integration:

- Do not rewrite release-state artifacts without an explicit release-lane decision.
- Add/classify historical workspace policy in the triage ledger.
- Verify binary existence/hash locally for the 15 binary findings.
