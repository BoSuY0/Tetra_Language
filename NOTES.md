# Tetra Memory + IslandKernel Production Notes

- The external plan was written from a dump that predated the completed
  memory-production-ready goal. Live evidence from
  `.workflow/memory-production-ready-v1/final-report.md` should be used to mark
  old `MEM-D04`, `MEM-E02`, `MEM-E05`, `MEM-F02`, `MEM-F04`, and `MEM-G04`
  blockers as baseline complete unless current inspection contradicts it.
- IslandKernel production is broader than the previous memory surface goal. It
  requires first-class IslandKernel decisions, `IslandID/Epoch`, linear island
  token/free/reset semantics, independent proof validation, proof fuzzing,
  sanitizer/leak evidence, release attestation, and docs nonclaims.
- Public wording must remain conservative: no `Memory 100%`, perfect memory,
  Rust-like parity, arbitrary unsafe pointer safety, full actor scheduler proof,
  cross-target runtime parity, official benchmark superiority, or leak-free host
  tooling.
- The plan explicitly allows persistent object memory to be non-goal if the
  `internal/todium`/agent packages are absent. That status must be validated by
  a machine-checkable gate or final audit row.
- First recommended packet from the plan is `MEM-ISLAND-P13` when shipping
  reliability is urgent because it names a concrete `actornet.Broker` lifecycle
  risk. Otherwise `MEM-ISLAND-P02` is the architectural starting point.
- Live P0 audit confirmed the dump-era P13 broker risk: `Serve` spawned a
  context watcher that could only exit via `ctx.Done()`, while existing test
  helpers cancelled context before `Close()`. The new regression uses goroutine
  stack inspection through `runtime/pprof` rather than adding `goleak` to
  `go.mod`; this avoids a dependency change while proving the concrete leak.
- `MEM-ISLAND-P13` is intentionally `done_narrow`: close-without-cancel and
  quick gate evidence are covered, but full leak/soak/pprof release attestation
  remains in `MEM-ISLAND-P15` and `MEM-ISLAND-P16`.
- `MEM-ISLAND-P02` intentionally keeps IslandKernel pure and isolated. The new
  package gives later packets a small policy surface, but it does not yet make
  compiler facts, PLIR, lowering, reports, or release artifacts IslandKernel-
  verified. That integration belongs to P03-P11/P16.
- `MEM-ISLAND-P01` is a claim-contract guard, not a production proof. It adds
  shared vocabulary for island evidence tiers and blocks docs/report wording
  that would imply `Memory 100%`, completed IslandKernel, leak-free host
  tooling, or arbitrary unsafe pointer safety before P03-P16 evidence exists.
  Validated `island_proof_verified` report rows now require
  `validator_name: validate-island-proof`, but the verifier itself remains a
  P11 deliverable.
- `MEM-ISLAND-P03` gives facts/reports a machine-checkable island memory-ref
  identity: `island_id`, positive `epoch`, and `base_id`. PLIR currently
  defaults `ProvenanceIsland` facts to epoch `1`; this is intentionally only
  the identity/projection layer. P04/P10 must still introduce linear token
  invalidation and sanitizer/runtime traps for reset/free/stale access.
- `MEM-ISLAND-P04` adds the first concrete linear reset surface: `island_reset`
  consumes the source island token, returns a fresh token for the same
  `IslandID` with advanced epoch, rejects stale token/slice use in semantic
  checks, and carries `island_epoch_advanced` through PLIR and memoryfacts
  report vocabulary. This is not a sanitizer/runtime trap claim and not an
  independent IslandKernel proof; P10/P11/P16 remain responsible for those
  gates.
- `MEM-ISLAND-P05` is currently a BCE typed-proof slice, not full proof
  integration. `ProofID` remains the stable identifier, but PLIR now emits and
  verifies `ProofTerm` subject fields for bounds-check removal, validation
  copies the term into `ProofReport`, and memoryfacts/report validators require
  structured typed proof fields for validated bounds-proof rows. Noalias,
  storage, and island-move proof terms plus callback/epoch invalidation remain
  follow-up scope; do not use this slice to claim full proof-carrying IR.
