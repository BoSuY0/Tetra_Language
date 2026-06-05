# Memory Production Core v1 Nonclaims

Status: MPC-16 explicit nonclaims.

Memory Production Core v1 intentionally does not claim:

- perfect memory in all possible programs
- full Rust-like borrow checker parity
- full FFI lifetime system
- safety for arbitrary unsafe external pointers
- full derived-pointer provenance for every raw address
- full production actor runtime
- full target runtime parity
- fastest language
- official benchmark result

## Boundaries

- Safe supported surface claims are limited to the documented current compiler
  surface and the evidence listed in
  `docs/audits/memory-production-core-v1-final.md`.
- Unknown unsafe memory remains conservative. `unsafe_unknown` may be checked,
  trapped, or rejected, but it is never trusted as safe provenance, noalias,
  bounds-check elimination, or trusted storage.
- Target claims are tiered by `docs/audits/memory-target-capability-matrix.md`.
  Linux-x64 runtime evidence does not imply full target runtime parity.
- Actor/task/request memory claims are conservative unless a row explicitly
  names validated runtime evidence. Evidence-only zero-copy rows are not full
  production actor runtime proof.
- Fuzz/property/stress output is oracle-backed evidence, not exhaustive proof.
- Performance reports and quick CI artifacts are evidence, not an official
  benchmark result and not a fastest language claim.
