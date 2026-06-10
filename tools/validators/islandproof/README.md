# tools/validators/islandproof

Validator package for island proof evidence.

This boundary owns the `tetra.island.proof.v1` report contract used by the
`validate-island-proof` CLI and release gates. It must reject malformed,
stale, producer-only, same-commit mismatched, unsafe-unknown, noalias, storage,
and bounds proof claims unless the proof rows are tied to matching memory fact
report rows and independent verifier checks.
