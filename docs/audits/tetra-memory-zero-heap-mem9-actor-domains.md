# Tetra Memory Zero-Heap MEM-9 Actor Domains

Date: 2026-06-16.
Status: complete for MEM-9 local actor-domain evidence and explicit production
runtime block.

## Scope

MEM-9 closes the actor memory-domain slice by adding validated local
`parallelrt` evidence for:

- mailbox byte capacity and queued bytes;
- message slab/live/reclaimed bytes;
- owned-region bytes charged to the receiver actor domain;
- copied bytes and copy count;
- byte-based mailbox backpressure;
- local owned-region `zero_copy_move` as domain owner movement.

This is not a production actor-runtime memory claim. Production per-actor byte
sampling remains explicitly blocked until the production runtime exposes that
sampler.

## Implemented Evidence

- `compiler/internal/parallelrt/scheduler_model.go`
  - `TypedMailbox` tracks queued, peak, reclaimed, requested, copied, and moved
    bytes.
  - `ActorMemoryDomainReport` includes evidence class/method, mailbox,
    message-pool, owned-region, backpressure, and nonclaim fields.
  - `ValidateActorMemoryDomainReport` validates schema, byte consistency,
    nonclaims, local-model blocked runtime reason, and production/distributed
    overclaim guards.
  - `CollectPrototypeEvidence` emits `benchmarks` plus
    `actor_memory_domains`.
- `compiler/cmd/parallelrt-evidence/main.go`
  - emits `tetra.parallelrt.prototype-evidence.v1` with benchmark prep rows and
    actor memory-domain reports.
- `tools/cmd/parallel-production-smoke/main.go`
  - parses the new evidence object, writes the raw artifact, and carries
    `actor_memory_domains` into the validated parallel production report.
- `tools/validators/parallelprod/report.go`
  - requires `actor_memory_domains`, rejects allocation-report-only evidence,
    validates byte consistency, requires byte-limit backpressure evidence,
    requires owned-region byte evidence, and rejects production/distributed
    actor claims.

## Current Raw Evidence Shape

```text
schema=tetra.parallelrt.prototype-evidence.v1
benchmarks=5
actor_memory_domains=2
actor-mailbox-copy:
  evidence_class=local_parallelrt_model
  runtime_measured=false
  current_bytes=48
  bytes_copied=32
  backpressure=available
actor-frame:
  evidence_class=local_parallelrt_model
  runtime_measured=false
  current_bytes=272
  owned_regions=1
  backpressure=byte_limit_reached
```

## Verification

```sh
GOCACHE=$(pwd)/.cache/go-build-actor-domain go test ./compiler/internal/parallelrt/... ./compiler/internal/actorsrt/... ./compiler/internal/actorsafety/... ./tools/validators/parallelprod/... ./tools/cmd/parallel-production-smoke -run 'Actor|MemoryDomain|Mailbox|Budget|Bytes|Backpressure|ZeroCopy|Claim' -count=1
```

Result: passed.

Additional checks:

```sh
GOCACHE=$(pwd)/.cache/go-build-actor-domain-tests go test ./compiler/internal/parallelrt/... ./tools/validators/parallelprod/... ./tools/cmd/parallel-production-smoke ./tools/cmd/validate-parallel-production -run 'Actor|MemoryDomain|Mailbox|Budget|Bytes|Backpressure|ZeroCopy|Claim|ValidateParallelProduction|BuildReport|ParseParallelSchedulerEvidence' -count=1
GOCACHE=$(pwd)/.cache/go-build-actor-domain-tests go run ./compiler/cmd/parallelrt-evidence
```

Result: passed.

## Nonclaims

- No production actor-runtime memory sampler is claimed.
- No distributed actor zero-copy is claimed.
- No cross-runtime or network zero-copy is claimed.
- No actor benchmark superiority, C++/Rust parity, or official benchmark claim
  is introduced.
- Local `parallelrt` actor-domain bytes are model/report evidence unless paired
  with future production runtime measurement.
