# Formal Core Semantics

Status: P23.2 internal verified-track evidence.

The formal core is deliberately small. It is not a full formalization of Tetra.
It names the semantic facts that optimization, allocation, and backend evidence
must preserve before wider verification work or self-hosting claims can build on
them.

## Concepts

The current `compiler/internal/formalcore` minimum spec covers:

- `values`: stable observable scalar results for the current differential subset;
- `provenance`: whether a memory value is owned, borrowed, derived, island-backed,
  external, or unknown;
- `regions`: explicit region identities for region-backed memory values, views,
  and borrows;
- `borrow_copy`: borrow preserves a source relation, while copy creates owned
  provenance;
- `bounds_proof`: removed checks require proof ids and live dominance evidence;
- `allocation_length_contract`: allocation lengths are classified as valid
  empty, normal, rejected negative, rejected overflow, or invalid before storage
  claims are trusted;
- `allocation_intent`: storage plans must match allocation intent and lowered IR;
- `raw_pointer_bounds_metadata`: raw pointer metadata remains allocation-base,
  derived-offset, rejected, or checked external/unknown without forging
  provenance;
- `check_elimination_validity`: unchecked lowered operations are valid only when
  the proof that justifies them is preserved.

## Machine Checks

Each rule must name a machine-checkable path. The current minimum spec maps the
core rules to existing or new internal evidence:

- `compiler/internal/differential.CheckScalarI32` for stable scalar value
  equivalence across source interpreter, stack backend, register backend, and
  optimized backend lanes;
- `compiler/internal/validation.CheckBoundsProofsWithPLIR` for proof-before-check
  elimination;
- `compiler/internal/validation.ValidateAllocationLowering` for allocation
  intent and lowered-storage consistency;
- `compiler/internal/allocplan.VerifyPlan` for allocation length contracts;
- `compiler/internal/runtimeabi.RawPointerBoundsMetadata` for raw pointer bounds
  metadata;
- `compiler/internal/plir.VerifyProgram` for provenance and borrow/copy fact
  consistency.

## P23.2 Report

`BuildP23FormalCoreV1Report` emits `tetra.formal_core.v1` /
`p23.2_formal_core_v1` rows for values, borrows and owned/copy, provenance and
regions, bounds proof id semantics, allocation length contracts, allocation
intent lowering, raw pointer bounds metadata, and check-elimination validity.
Its validator rejects fake full-formal-proof, broad-language-proof,
unsafe-policy-change, runtime-behavior-change, safe-semantics-change, and
performance claims.

## Translation Metadata

`compiler/internal/validation.BuildOptimizationValidationMetadata` produces
`tetra.translation.validation.metadata.v1` records. A record includes the pass
name, input/output IR kinds, declared fact contract, sha256 hashes of the
before/after IR, the compared function set, and translation-validation counters.
The opt manager stores that metadata for translation-validation passes.

## Self-hosting Gate

`compiler/internal/selfhostgate.Evaluate` keeps the self-hosting path gated. A
self-hosting claim is blocked until the compiler subset, register backend,
optimizer, allocator/runtime, stdlib, small compiler component compile,
Go-vs-Tetra output comparison, deterministic bootstrap chain, and
cross-platform bootstrap evidence are present.

P23.3 records this as `tetra.self_hosting.gate.v1` /
`p23.3_self_hosting_gate`. The current report requires
`SelfHostingClaimed=false` and `GateDecision.Allowed=false`; it records current
backend/optimizer/allocator/runtime/stdlib evidence while keeping the small
compiler component, output comparison, deterministic bootstrap, and
cross-platform bootstrap rows blocked.

## Non-goals

- No public source interpreter mode.
- No public backend selector.
- No fastest-language or official benchmark claim.
- No full formal proof of Tetra.
- No broad language theorem prover.
- No self-hosting claim without the explicit gate evidence above.
