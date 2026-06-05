# P24.0 Threat Model

Status: current-branch P24.0 audit artifact for schema
`tetra.security.review_gate.v1` and scope `p24.0_security_review_gate`.

Review date: 2026-06-03.

## Assumptions

- The attacker can provide Tetra source, capsule manifests, Eco package
  metadata, local package archives, PostgreSQL wire-protocol bytes from a test
  or local server, and network/runtime event timing.
- The attacker cannot modify the compiler binary, Go toolchain, repository
  checkout, or trusted local filesystem outside normal user write permissions.
- This review is source-and-validator evidence for the current branch. It is not
  an external penetration test, cryptographic audit, or release signoff.

## Assets

| Asset | Security property |
| --- | --- |
| Safe Tetra program semantics | Safe code must not gain unsafe powers through missing syntax, missing effect checks, or forged capability tokens. |
| Raw memory operations | Unsafe raw memory entry points must remain explicit, auditable, and bounded by current runtime metadata where implemented. |
| Runtime allocators | Allocation contracts must reject invalid sizes before allocator or metadata access and expose report hooks for review. |
| Network runtime | Event readiness, backpressure, cancellation, and platform boundaries must be recorded without overstating production coverage. |
| Actor runtime | Actor capacity, mailbox, scheduler, and distributed-runtime limits must remain explicit. |
| PostgreSQL protocol path | Frame lengths, SCRAM state, authentication errors, borrowed rows, and pool limits must fail safely under malformed or hostile input. |
| Eco package metadata | Lock hashes, package hashes, trust snapshots, mirrors, vault objects, and paths must validate before local trust or writes. |
| Release review artifacts | Release security signoff must require current commit, reviewer, evidence commands, artifact hashes, and residual risks. |

## Trust Boundaries

| Boundary | Inputs crossing boundary | Existing control |
| --- | --- | --- |
| Safe source to unsafe operation | `unsafe { ... }`, unsafe builtins, `uses` clauses | Checker policies in `docs/spec/unsafe.md`; focused `Unsafe|Capability|Effect|MMIO|Mem` tests. |
| Capability token acquisition | `core.cap_mem`, `core.cap_io`, wrapper APIs | Tokens are obtained only in unsafe blocks; `uses` does not manufacture tokens. |
| Raw pointer to allocator metadata | `core.alloc_bytes`, `core.ptr_add`, raw loads/stores, raw slices | `RuntimeAllocationContracts` and `RuntimeRawPointerBoundsABI`. |
| Network event source to runtime | epoll readiness, accept/read/write, timers, cancellation | `netrt.IOReactorCoverage` and focused netrt tests. |
| Actor/task messages to runtime state | actor spawn/send/receive, scheduler prototype metadata | `ActorRuntimeProductionBoundaryAudit`; capacity limits and blockers. |
| PostgreSQL socket to runtime DB path | startup/auth frames, query result frames, row data | `ReadFrame` size limits, malformed-frame errors, SCRAM validation, pool backpressure. |
| Package archive/metadata to local store | `.todex`, `metadata.json`, `trust.snapshot.json`, vault objects | Eco validators enforce normalized relative paths and sha256 matches. |
| Release artifact directory to signoff | reviewer signoff Markdown and artifact hashes | `scripts/release/v1_0/security-review.sh` and `tools/scriptstest/security_review_test.go`. |

## Abuse Paths And Mitigations

| Abuse path | Mitigation evidence | Residual risk |
| --- | --- | --- |
| Source tries to call unsafe builtins from safe code. | `docs/spec/unsafe.md`; compiler safety test slice. | Review depends on manifest/checker coverage staying aligned. |
| Source declares `uses mem` and treats it as `cap.mem`. | `docs/spec/capabilities.md`; `docs/spec/effects_capabilities_privacy_v1.md`. | Wrappers taking `cap.mem` still need documented caller pointer obligations. |
| Raw pointer offset escapes allocation bounds. | `runtimeabi.RuntimeRawPointerBoundsABI`; rejected negative and upper-bound offsets. | External or unknown pointers remain unknown, not proven safe. |
| Network code treats Linux epoll evidence as portable runtime evidence. | `netrt.IOReactorCoverage` rejects cross-platform parity and io_uring claims. | Non-Linux event adapters remain future work. |
| Actor runtime evidence is promoted to production scheduler evidence. | `ActorRuntimeProductionBoundaryAudit` rejects full production actor runtime claims. | Message-pool recovery and full race-safety evidence remain incomplete. |
| PostgreSQL server sends malformed frames or oversized payloads. | `ReadFrame` returns `ErrMalformedFrame` or `ErrFrameTooLarge`; production coverage validates protocol rows. | TLS/channel binding and external production deployment remain outside current evidence. |
| Package metadata uses path traversal or mismatched hashes. | Eco publish/download/mirror/vault/unpack validators reject unsafe paths and hash mismatches. | Remote identity and federation trust are not established. |
| Release signoff uses stale commit or template text. | `security-review.sh` validates current commit, reviewer fields, decision, evidence commands, artifact hashes, residual risks, and template-text rejection. | Human reviewer judgement remains required for final release decisions. |

## Open Questions For Future Evidence

- Which release candidate will first require a named reviewer security signoff
  derived from these P24.0 artifacts?
- Which non-Linux network runtime adapters are planned for executable
  security-focused smokes?
- Which external database deployment profile, TLS policy, and channel binding
  expectations are in scope for the first production DB claim?
- Which package registry identity model will replace the current local
  metadata-only trust boundary?

## Non-Claims

- Security certification is not claimed.
- External penetration test is not claimed.
- CVE-free status is not claimed.
- Release security signoff is not claimed.
- Runtime behavior does not change.
- Safe-program semantics do not change.
- No performance claim is made.
