# Memory Fuzz Oracle v1

Schema: `tetra.memory-fuzz.oracle.v1`

MPC-15 adds memory fuzzing with an oracle, not random-noise evidence. The
oracle is compiler-owned and artifact-only: it classifies generated memory
program outcomes, then validates the report before a short fuzz artifact can be
used as release evidence.

## Oracle Categories

- checker reject expected: a generated unsupported or unsafe program must be
  rejected by the checker with the expected diagnostic.
- runtime trap expected: a generated normal-build program must keep the
  required runtime check and trap with the expected bounds or memory diagnostic.
- compiled output equals interpreter/reference expected: a generated supported
  program must match the interpreter/reference lane for the bounded sample.
- compiler crash is bug: a parser, checker, lowerer, backend, or tool crash is
  a bug and never a passing fuzz result.
- miscompile is bug: compiled output that differs from the reference is a bug
  and must produce a reducer/reproducer artifact.
- unsafe_unknown optimized as safe is bug: `unsafe_unknown` may remain checked,
  trapped, or conservative, but it must never become `safe_known`.
- report validation failure is bug: memory fuzz reports must validate against
  `MemoryFactGraph` and the memory cost model before they are promoted.

## Fuzz Tiers

- Tier 1 short CI smoke: deterministic bounded memory oracle cases that can run
  in normal CI and write `reports/memory-fuzz-short/...` artifacts.
- Tier 2 nightly fuzz: longer seeded fuzz/property/stress runs that reuse the
  existing nightly fuzz report conventions and preserve unstable-seed triage.
- Tier 3 release-blocking focused memory fuzz: release-focused memory fuzz and
  stress gates whose failures block promotion until reduced or classified.

## MEM-FUZZ-012 Release Evidence Rows

`MEM-FUZZ-012` narrows the oracle into an auditable release-evidence layer for
the v0-v11 Memory Ideal chain:

- `MEM-FUZZ-001`: Tier 1 short CI smoke covers deterministic v0-v11 memory
  oracle cases.
- `MEM-FUZZ-002`: compiler crash and miscompile outcomes require
  reducer/reproducer artifact slots.
- `MEM-FUZZ-003`: `unsafe_unknown` optimized as safe, bounds check elimination
  without proof id, trusted storage under escape, or report validation failure
  blocks release promotion.
- `MEM-FUZZ-004`: Tier 2 nightly fuzz records seed preservation, unstable
  triage, and minimized repro expectations.
- `MEM-FUZZ-005`: Tier 3 release-blocking focused memory fuzz must pass or
  classify every failure before release promotion.

The Tier 1 v12 command writes:

```sh
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v12-fuzz-oracle go run ./tools/cmd/memory-fuzz-short --tier=1 --report-dir reports/memory-fuzz-short/v12
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v12-fuzz-validate go run ./tools/cmd/validate-memory-fuzz-oracle --report reports/memory-fuzz-short/v12/memory-fuzz-oracle.json
```

The generated report includes five `MEM-FUZZ-*` requirement rows, twelve
deterministic slice coverage rows (`v0` through `v11`), blocking-case rows, Tier
policy rows, and required artifact kinds for compiler-crash reproducers,
miscompile reproducers, and miscompile reducers.

## Generator Surface Tiers

- Tier 1 supported now: slices, Strings, borrow/copy, simple
  structs/enums/optionals, safe views, `make_*`, and explicit islands.
- Tier 2 supported narrow: generics, function-typed borrowed returns,
  async/task boundary smoke, and raw verified roots.
- Tier 3 conservative/rejected: arbitrary unsafe pointers, unknown external
  calls, and unsupported target behavior.
- Tier 4 future: full FFI lifetime, full actor zero-copy runtime, and generic
  lifetimes.

## Required Invariants

- no safe metadata mutation
- no borrowed escape
- no unsafe_unknown -> safe_known
- no removed bounds check without proof id
- no stack/region storage if escape exists
- reports validate against MemoryFactGraph
- memory report rows keep `cost_class` and `normal_build_check` rules from the
  memory cost model.

## Release-Blocking Cases

The oracle treats these as blocking classifications before promotion:

- `unsafe_unknown_optimized_as_safe`
- `bounds_check_eliminated_without_proof_id`
- `trusted_storage_under_escape`
- `report_validation_failure`

These are not runtime proofs. They are report/oracle classifications that keep
release evidence conservative until the underlying MemoryFactGraph validators
and focused repro artifacts support the claim.

## Non-Claims

This artifact is not exhaustive fuzzing, a full program-correctness proof, a
full unsafe pointer safety claim, a performance claim, a runtime behavior
change, or a safe-program semantics change. `MEM-FUZZ-012` also makes no
arbitrary unsafe safety claim, no full runtime/ABI/target parity proof, no
clean-release claim under a dirty worktree, no replacement for MemoryFactGraph
validators, and no "Memory 100%" claim.
