# P24.0 Threat Model

Status: current-branch P24.0 audit artifact for schema `tetra.security.review_gate.v1` and scope
`p24.0_security_review_gate`.

Review date: 2026-06-03.

## Assumptions

- The attacker can provide Tetra source, capsule manifests, Eco package metadata, local package
  archives, PostgreSQL wire-protocol bytes from a test or local server, and network/runtime event
  timing.
- The attacker cannot modify the compiler binary, Go toolchain, repository checkout, or trusted
  local filesystem outside normal user write permissions.
- This review is source-and-validator evidence for the current branch. It is not an external
  penetration test, cryptographic audit, or release signoff.

## Assets

- Asset: Safe Tetra program semantics
  - Security property: Safe code must not gain unsafe powers through missing syntax, missing
    effect checks, or forged capability tokens.
- Asset: Raw memory operations
  - Security property: Unsafe raw memory entry points must remain explicit, auditable, and
    bounded by current runtime metadata where implemented.
- Asset: Runtime allocators
  - Security property: Allocation contracts must reject invalid sizes before allocator or
    metadata access and expose report hooks for review.
- Asset: Network runtime
  - Security property: Event readiness, backpressure, cancellation, and platform boundaries
    must be recorded without overstating production coverage.
- Asset: Actor runtime
  - Security property: Actor capacity, mailbox, scheduler, and distributed-runtime limits must
    remain explicit.
- Asset: PostgreSQL protocol path
  - Security property: Frame lengths, SCRAM state, authentication errors, borrowed rows, and
    pool limits must fail safely under malformed or hostile input.
- Asset: Eco package metadata
  - Security property: Lock hashes, package hashes, trust snapshots, mirrors, vault objects, and
    paths must validate before local trust or writes.
- Asset: Release review artifacts
  - Security property: Release security signoff must require current commit, reviewer, evidence
    commands, artifact hashes, and residual risks.

## Trust Boundaries

- Boundary: Safe source to unsafe operation
  - Inputs crossing boundary: `unsafe { ... }`, unsafe builtins, `uses` clauses.
  - Existing control: Checker policies in `docs/spec/runtime/unsafe.md`; focused
    `Unsafe Capability Effect MMIO Mem` tests.
- Boundary: Capability token acquisition
  - Inputs crossing boundary: `core.cap_mem`, `core.cap_io`, wrapper APIs.
  - Existing control: Tokens are obtained only in unsafe blocks; `uses` does not manufacture
    tokens.
- Boundary: Raw pointer to allocator metadata
  - Inputs crossing boundary: `core.alloc_bytes`, `core.ptr_add`, raw loads/stores, raw slices.
  - Existing control: `RuntimeAllocationContracts` and `RuntimeRawPointerBoundsABI`.
- Boundary: Network event source to runtime
  - Inputs crossing boundary: epoll readiness, accept/read/write, timers, cancellation.
  - Existing control: `netrt.IOReactorCoverage` and focused netrt tests.
- Boundary: Actor/task messages to runtime state
  - Inputs crossing boundary: actor spawn/send/receive, scheduler prototype metadata.
  - Existing control: `ActorRuntimeProductionBoundaryAudit`; capacity limits and blockers.
- Boundary: PostgreSQL socket to runtime DB path
  - Inputs crossing boundary: startup/auth frames, query result frames, row data.
  - Existing control: `ReadFrame` size limits, malformed-frame errors, SCRAM validation, pool
    backpressure.
- Boundary: Package archive/metadata to local store
  - Inputs crossing boundary: `.todex`, `metadata.json`, `trust.snapshot.json`, vault objects.
  - Existing control: Eco validators enforce normalized relative paths and sha256 matches.
- Boundary: Release artifact directory to signoff
  - Inputs crossing boundary: reviewer signoff Markdown and artifact hashes.
  - Existing control: `scripts/release/v1_0/security-review.sh` and
    `tools/scriptstest/security_review_test.go`.

## Abuse Paths And Mitigations

- Abuse path: Source tries to call unsafe builtins from safe code.
  - Mitigation evidence: `docs/spec/runtime/unsafe.md`; compiler safety test slice.
  - Residual risk: Review depends on manifest/checker coverage staying aligned.
- Abuse path: Source declares `uses mem` and treats it as `cap.mem`.
  - Mitigation evidence: `docs/spec/runtime/capabilities.md`;
    `docs/spec/runtime/effects_capabilities_privacy_v1.md`.
  - Residual risk: Wrappers taking `cap.mem` still need documented caller pointer obligations.
- Abuse path: Raw pointer offset escapes allocation bounds.
  - Mitigation evidence: `runtimeabi.RuntimeRawPointerBoundsABI`; rejected negative and
    upper-bound offsets.
  - Residual risk: External or unknown pointers remain unknown, not proven safe.
- Abuse path: Network code treats Linux epoll evidence as portable runtime evidence.
  - Mitigation evidence: `netrt.IOReactorCoverage` rejects cross-platform parity and io_uring
    claims.
  - Residual risk: Non-Linux event adapters remain future work.
- Abuse path: Actor runtime evidence is promoted to production scheduler evidence.
  - Mitigation evidence: `ActorRuntimeProductionBoundaryAudit` rejects full production actor
    runtime claims.
  - Residual risk: Message-pool recovery and full race-safety evidence remain incomplete.
- Abuse path: PostgreSQL server sends malformed frames or oversized payloads.
  - Mitigation evidence: `ReadFrame` returns `ErrMalformedFrame` or `ErrFrameTooLarge`;
    production coverage validates protocol rows.
  - Residual risk: TLS/channel binding and external production deployment remain outside
    current evidence.
- Abuse path: Package metadata uses path traversal or mismatched hashes.
  - Mitigation evidence: Eco publish/download/mirror/vault/unpack validators reject unsafe
    paths and hash mismatches.
  - Residual risk: Remote identity and federation trust are not established.
- Abuse path: Release signoff uses stale commit or template text.
  - Mitigation evidence: `security-review.sh` validates current commit, reviewer fields,
    decision, evidence commands, artifact hashes, residual risks, and template-text rejection.
  - Residual risk: Human reviewer judgement remains required for final release decisions.

## Open Questions For Future Evidence

- Which release candidate will first require a named reviewer security signoff derived from these
  P24.0 artifacts?
- Which non-Linux network runtime adapters are planned for executable security-focused smokes?
- Which external database deployment profile, TLS policy, and channel binding expectations are in
  scope for the first production DB claim?
- Which package registry identity model will replace the current local metadata-only trust boundary?

## Non-Claims

- Security certification is not claimed.
- External penetration test is not claimed.
- CVE-free status is not claimed.
- Release security signoff is not claimed.
- Runtime behavior does not change.
- Safe-program semantics do not change.
- No performance claim is made.
