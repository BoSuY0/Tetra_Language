# Notes: Memory Ideal Vertical Slice v6 Bounds

## Chronological Notes

- 2026-06-05: Audit seed says current honest claim is v0-v5
  validated/rejected/conservative for bounded surfaces, not "Memory 100%".
- 2026-06-05: Highest-risk next gap is optimization trust, especially removed
  bounds checks without Memory Ideal proof-row coverage.
- 2026-06-05: Existing live checkout HEAD matches dump metadata
  `5129f2623d9639990076a7d422e56f02b0ed3254`, but the worktree is heavily
  dirty. Preserve unrelated changes.
- 2026-06-05: Existing proof-id substrate includes
  `validation.CheckBoundsProofsWithPLIR`, `plir.VerifyProgram`, and
  `compiler/internal/lower/proof_bce_test.go`; v6 should connect these to
  MemoryFactGraph/report/correlation evidence.

## Nonclaims To Repeat

- No "Memory 100% complete".
- No broad optimizer correctness.
- No target parity.
- No performance claim.
- No arbitrary unsafe pointer arithmetic proof.
- No arbitrary external pointer safety.
- No FFI lifetime model.
- No full theorem prover.
