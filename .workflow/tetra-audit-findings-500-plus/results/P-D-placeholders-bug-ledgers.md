# P-D Placeholders / Bug Ledgers

Status: completed read-only sub-agent audit.

Scope:

- 90 placeholder / unfinished / fake marker findings.
- 195 documented bug / regression-risk ledger findings.

Live checkout state:

- 81 of 90 placeholder findings still have a marker at the stated line.
- 9 of 90 are line-drift/no-exact-token cases with nearby evidence.
- 195 of 195 documented bug ledger entries are live in `Tetra_BUGS.md` and `TetraProjects_Bugs.md`.

Placeholder grouping:

- `placeholder-policy`: 14
- `TODO/TBD`: 31
- `fake`: 45
- Highest-risk code groups include `compiler/features.go` and selected `tools/*` fake markers.
- Many docs/generated/plans markers are policy or roadmap text and should be classified, not blindly deleted.

Bug ledger grouping:

- `Tetra_BUGS.md`: 120 entries.
- `TetraProjects_Bugs.md`: 75 entries.

Recommended integration:

- Build a triage ledger first.
- Treat bug ledgers as live open work unless tests prove closure.
- Normalize markers into accepted labels such as roadmap/non-claim/owner-target-version, or fix/remove truly fake code/test behavior.
