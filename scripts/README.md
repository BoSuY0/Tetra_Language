# scripts

Script entrypoints are grouped by purpose:

- `scripts/ci`: CI-facing test and verification wrappers
- `scripts/dev`: local developer workflow entrypoints
- `scripts/release`: versioned release gate, security, and smoke wrappers
- `scripts/tools`: implementation helpers used by scripts

New or migrated workflows must live in the domain directory directly. Do not add
root-level compatibility wrappers; migrate callers to the canonical path before
removing an old root-level entrypoint.
