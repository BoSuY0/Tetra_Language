# Truth Benchmark Harness

Status: P8 evidence harness with bounded P19/P20 master-plan scopes and P20.2
claim-tier validation.

`tools/cmd/truth-bench-harness` records fair benchmark rows for Tetra, C,
C++, and Rust without making broad performance claims. The default P8 contract
requires the full benchmark matrix before a report validates. A manifest may
also name a narrower checked scope when a later master-plan slice needs a
bounded equivalent artifact before the full P20 matrix.

The manifest names each benchmark, category, language, compiler version, build
command, run command, binary path, optional `algorithm_id` and
`input_description`, optional raw output artifacts, and for Tetra rows the
proof/allocation/bounds reports produced by the default build, normally via
`tetra build app.tetra --explain`. Scopes that require Tetra performance
reports validate `tetra_reports` too.

Supported manifest scopes:

- default / `p8_full`: full P8 matrix across all required categories and Tetra,
  C, C++, and Rust.
- `p19.1_generic_collections`: checked dry-run P19.1 generic-collection
  equivalent artifact for `hash table` with Tetra, C++, and Rust rows. This
  scope requires matching `algorithm_id` and `input_description` across all
  rows and a Tetra performance report artifact.
- `p19.2_http_json_source_first`: checked dry-run P19.2 HTTP/JSON source gate
  for Tetra-only `HTTP plaintext` and `HTTP JSON` rows with proof/allocation/
  bounds and P19.2 coverage artifacts.
- `p19.3_postgres_source_first`: checked dry-run P19.3 PostgreSQL source gate
  for Tetra-only `DB single query`, `DB multiple queries`, `DB updates`, and
  `DB fortunes` rows with proof/allocation/bounds and P19.3 coverage
  artifacts.
- `p15_actor_benchmark_prep`: checked dry-run ACTOR-P15 actor benchmark prep
  gate for Tetra-only actor ping-pong, fanout/fanin, mailbox throughput,
  backpressure latency, and `zero_copy_move` local typed mailbox rows. This
  scope requires raw output artifact references, proof/allocation/bounds
  reports, and Tetra report artifacts. Rows remain `ran=false` unless a
  separate Tier 1 local run exists.
- `p20.0_benchmark_matrix`: checked dry-run P20.0 master-plan matrix gate with
  Tetra, C, C++, and Rust rows for every P20 category. This scope requires
  matching `algorithm_id` and `input_description` for equivalent rows, raw
  output artifact paths on every row, Tetra proof/allocation/bounds/performance
  artifacts for every Tetra row, and row target CPUs matching the report host
  target CPU.

The required language baselines are:

- Tetra through `tetra build ... --explain`
- C through `clang -O3`
- C++ through `clang++ -O3`
- Rust through `rustc -C opt-level=3`

The P8 required categories are:

- integer loop
- slice sum
- bounds-check loop
- allocation microbench
- stack allocation
- region/island allocation
- copy/copy_into
- hash table
- JSON parse
- HTTP plaintext
- DB single query
- actor ping-pong
- actor zero-copy transfer
- parallel map/reduce

The P20.0 benchmark-matrix categories are:

- integer loops
- slice sum
- bounds-check loops
- function calls
- recursion
- matrix multiply
- hash table
- allocation
- region/island allocation
- JSON parse/stringify
- HTTP plaintext/json
- PostgreSQL single/multiple/update
- actor ping-pong
- parallel map/reduce
- startup time
- binary size
- compile time

The P15 actor benchmark prep categories are:

- actor ping-pong
- actor fanout/fanin
- actor mailbox throughput
- actor backpressure latency
- `zero_copy_move` local typed mailbox

The generated `tetra.truth.benchmark.v1` report records:

- scope, when the manifest requested a named bounded scope
- host OS, architecture, CPU count, and target CPU
- compiler version for each row
- optional algorithm and input equivalence metadata
- exact build and run commands
- whether commands were actually run
- runtime duration
- binary size
- raw output artifact paths and whether they existed, for scopes that require
  raw output evidence
- Tetra proof report artifact paths and whether they existed
- Tetra allocation report artifact paths and whether they existed
- Tetra bounds report artifact paths and whether they existed
- Tetra performance report artifact paths and whether they existed, for scopes
  that require them
- a claim-policy note forbidding global "fastest language" conclusions

The P20.2 claim-tier report records:

- schema `tetra.performance.claim_tiers.v1`
- scope `p20.2_claim_tiers`
- the exact Tier 0 through Tier 4 policy rows from the master plan
- current P20.0/P20.1 wording as Tier 0 local smoke only
- explicit non-claims for measured speed, C++/Rust parity, official benchmark,
  official TechEmpower, cross-machine reproduction, independent reproduction,
  throughput advantage, and latency advantage

Use dry-run mode first:

```sh
go run ./tools/cmd/truth-bench-harness \
  --manifest reports/benchmarks/p8-full-manifest.json \
  --out reports/benchmarks/p8-full-report.json
```

Use execution mode only when all compilers and inputs are pinned:

```sh
go run ./tools/cmd/truth-bench-harness \
  --manifest reports/benchmarks/p8-full-manifest.json \
  --out reports/benchmarks/p8-full-report.json \
  --run
```

Generate the P20.2 claim-tier policy artifact with:

```sh
go run ./tools/cmd/truth-bench-harness \
  --claim-tiers-out reports/claim-tiers-v1/claim-tier-report.json
```

Named bounded scopes can be used before the full P20 matrix is ready:

- `p19.1_generic_collections`: checked dry-run P19.1 generic-collection
  hash-table rows for Tetra/C++/Rust equivalents.
- `p19.2_http_json_source_first`: checked dry-run P19.2 HTTP/JSON rows for
  Tetra-only `HTTP plaintext` and `HTTP JSON` source builds. This scope requires
  Tetra proof/allocation/bounds and P19.2 coverage artifacts. It is source-first
  gate evidence only; it does not record throughput, C++/Rust parity, a full
  production web stack, or an official TechEmpower result.
- `p19.3_postgres_source_first`: checked dry-run P19.3 PostgreSQL rows for
  Tetra-only `DB single query`, `DB multiple queries`, `DB updates`, and
  `DB fortunes` source builds. This scope requires Tetra proof/allocation/
  bounds and `tetra.stdlib.postgresql.production_driver.v1` coverage
  artifacts. It is source-first gate evidence only; it does not record database
  throughput, C++/Rust parity, a production database benchmark, or an official
  TechEmpower result.
- `p15_actor_benchmark_prep`: checked dry-run ACTOR-P15 actor benchmark prep
  rows for local Linux-x64 actor/task workload candidates. It records raw
  artifact references and Tier 0 scope only. It does not record measured speed,
  actor benchmark superiority, official benchmark status, a production
  throughput guarantee, real-world SLA evidence, distributed actor runtime
  promotion, or distributed/network zero-copy.
- `p20.0_benchmark_matrix`: checked dry-run P20.0 benchmark matrix across 17
  master-plan categories and the four required languages. The checked artifact
  is
  `reports/benchmark-matrix-hardening-v1/benchmarks/p20-matrix-hardening-report.json`;
  it has 68 rows, raw output artifacts on every row, Tetra performance report
  artifacts on every Tetra row, and `ran=false`. It proves the matrix/evidence
  contract only; it does not record measured speed, C/C++/Rust parity,
  production database throughput, or an official benchmark result.

Allowed claims must cite one benchmark row, report artifact, target, runtime,
and comparison row. Broad claims about Tetra being the fastest language, beating
C/C++/Rust generally, or being an official TechEmpower result are not
evidence-backed by this harness. The validator also rejects fake C++/Rust parity
claims, fake official benchmark claims, missing raw output artifacts where
required, and P20 row target-CPU drift unless the text is explicitly a
non-claim.

P20.2 public wording must also match the claim tier:

- Tier 0: local smoke only.
- Tier 1: local benchmark evidence.
- Tier 2: reproducible cross-machine benchmark.
- Tier 3: independent reproduced benchmark.
- Tier 4: official upstream benchmark submission.

Current P20.0/P20.1 evidence validates only Tier 0 wording. The validator
rejects local-benchmark, cross-machine, independent-reproduction,
official/upstream/TechEmpower, measured-speed, throughput/latency advantage, and
C++/Rust parity wording unless the report carries the matching tier evidence or
the text is explicitly a non-claim.

ACTOR-P15 actor benchmark prep also validates only Tier 0 wording unless a
separate Tier 1 local benchmark report is produced. Its claim-tier nonclaims
include production throughput guarantees, distributed zero-copy, and actor
benchmark superiority. The `zero_copy_move` local typed mailbox row is local
owned-region metadata only; it is not distributed/network zero-copy and is not a
production runtime promotion.

Compiler `--explain` performance reports also include P20.1 blocker diagnostics
in `.perf.json` schema version 3: `left bounds check: missing dominance`,
`heap allocation: escapes through return`, `heap allocation: unknown call`,
`not vectorized: no noalias proof`, `not inlined: code-size budget`,
`register spill: live range pressure`, `stack fallback: unsupported aggregate
return`, and `actor copy: borrowed data crosses boundary`. The P20.1 report
also records one explanation row for every P20.0 Tetra benchmark and remains
evidence-only; it does not run benchmarks or make measured performance claims.
