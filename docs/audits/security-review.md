# P24.0 Security Review

Status: current-branch P24.0 audit artifact for schema
`tetra.security.review_gate.v1` and scope `p24.0_security_review_gate`.

Review date: 2026-06-03.

This file is an input to future release review. It is not the named reviewer
release signoff consumed by `scripts/release/v1_0/security-review.sh`.

## Scope

P24.0 reviews the surfaces that can bypass normal safety, touch host or runtime
capabilities, parse external protocol bytes, or move package trust across local
boundaries:

- unsafe APIs
- capabilities
- memory allocator
- network runtime
- actor runtime
- DB protocol
- package/Eco system
- build scripts
- supply chain

## Evidence Summary

### Unsafe APIs

- Primary evidence: `docs/spec/unsafe.md`.
- Primary evidence: `examples/flow_unsafe_cap_mem_smoke.tetra`.
- Primary evidence: `lib/core/capability.tetra`.
- Current boundary: inventory and policy review; not proof that every unsafe
  caller is memory safe.

### Capabilities

- Primary evidence: `docs/spec/capabilities.md`.
- Primary evidence: `docs/spec/effects_capabilities_privacy_v1.md`.
- Current boundary: `uses` declarations audit effects and do not grant
  `cap.mem` or `cap.io`.

### Memory Allocator

- Primary evidence: `compiler/internal/runtimeabi/allocation_contract.go`.
- Primary evidence: `compiler/internal/runtimeabi/raw_pointer_bounds.go`.
- Current boundary: runtime ABI evidence with raw-pointer bounds metadata; not
  a formal memory-safety proof.

### Network Runtime

- Primary evidence: `compiler/internal/netrt/io_reactor_coverage.go`.
- Primary evidence: `compiler/internal/netrt/netrt_linux.go`.
- Current boundary: Linux epoll v1 and focused reactor evidence; no full
  production web-stack or cross-platform parity claim.

### Actor Runtime

- Primary evidence: `compiler/internal/actorsrt/production_boundary.go`.
- Primary evidence: `docs/spec/actors.md`.
- Current boundary: current actor limits and scheduler prototype evidence; not
  a production multi-threaded actor runtime.

### DB Protocol

- Primary evidence: `compiler/internal/pgrt/production_postgres_coverage.go`.
- Primary evidence: `compiler/internal/pgrt/wire.go`.
- Primary evidence: `compiler/internal/pgrt/scram.go`.
- Primary evidence: `compiler/internal/pgrt/pool.go`.
- Current boundary: local PostgreSQL wire-protocol evidence; no TLS, channel
  binding, or external production database deployment claim.

### Package/Eco System

- Primary evidence: `docs/spec/eco_publishing_v1.md`.
- Primary evidence: `cli/cmd/tetra/eco_publish.go`.
- Primary evidence: `tools/cmd/validate-eco-*`.
- Current boundary: local lock/hash/trust metadata validation; no global
  package trust network.

### Build Scripts

- Primary evidence: `scripts/release/v1_0/security-review.sh`.
- Primary evidence: `tools/scriptstest/security_review_test.go`.
- Current boundary: release signoff validator exists separately from this
  P24.0 audit artifact.

### Supply Chain

- Primary evidence: `go.sum`.
- Primary evidence: Eco lock/publish/vault/mirror/unpack validators.
- Current boundary: local checksum and metadata validation; no SLSA, CVE-free,
  or external registry trust claim.

## Required Artifacts

The security review gate requires these same-branch artifacts:

- `docs/audits/security-review.md`
- `docs/audits/threat-model.md`
- `docs/audits/unsafe-surface-map.md`
- `docs/audits/capability-surface-map.md`

`BuildP24SecurityReviewGateV1Report` checks that all four files exist and
`ValidateP24SecurityReviewGateV1Report` rejects missing artifacts.

## Evidence Commands

Focused gate:

```sh
CACHE="$PWD/.cache/go-build-ideal-plan"
GOCACHE="$CACHE" \
  go test ./compiler \
    -run 'P24SecurityReviewGate' \
    -count=1
```

Related runtime/package evidence:

```sh
CACHE="$PWD/.cache/go-build-ideal-plan"
GOCACHE="$CACHE" \
  go test \
    ./compiler/internal/runtimeabi \
    ./compiler/internal/netrt \
    ./compiler/internal/actorsrt \
    ./compiler/internal/parallelrt \
    ./compiler/internal/pgrt \
    -count=1
GOCACHE="$CACHE" \
  go test ./tools/scriptstest \
    -run 'SecurityReview' \
    -count=1
GOCACHE="$CACHE" \
  go test \
    ./tools/cmd/validate-eco-lock \
    ./tools/cmd/validate-eco-publish \
    ./tools/cmd/validate-eco-vault \
    ./tools/cmd/validate-eco-mirror \
    ./tools/cmd/validate-eco-unpack \
    -count=1
GOCACHE="$CACHE" \
  go test ./compiler/tests/semantics \
    -run 'FeatureRegistry' \
    -count=1
GOCACHE="$CACHE" \
  go run ./tools/cmd/verify-docs \
    --manifest docs/generated/manifest.json
GOCACHE="$CACHE" \
  go run ./tools/cmd/validate-manifest \
    --manifest docs/generated/manifest.json
```

## Residual Risks

### Unsafe APIs

- Residual risk: capability tokens authorize entry to raw operations but do not
  prove pointer validity, lifetime, aliasing, or actor sendability.
- Current mitigation: unsafe syntax, effects, capability arguments,
  raw-pointer bounds metadata, and focused safety tests.

### Network Runtime

- Residual risk: Linux epoll coverage does not imply kqueue, IOCP, WASI/web
  parity, or production network hardening.
- Current mitigation: `netrt.IOReactorCoverage` records explicit platform
  non-claims.

### Actor Runtime

- Residual risk: message-pool exhaustion/reclamation, full cancellation,
  race-safety proof, and production scheduler integration remain incomplete.
- Current mitigation: `actorsrt.ActorRuntimeProductionBoundaryAudit` records
  blockers.

### DB Protocol

- Residual risk: local SCRAM and frame parsing evidence does not include TLS,
  channel binding, or external production deployment review.
- Current mitigation: `pgrt.ProductionPostgresCoverage` records protocol and
  benchmark honesty boundaries.

### Eco Supply Chain

- Residual risk: local hashes and trust snapshots do not establish a global
  trust federation.
- Current mitigation: Eco validators reject unsafe paths, unknown fields, and
  hash mismatches before local writes.

## Non-Claims

- Security certification is not claimed.
- External penetration test is not claimed.
- CVE-free status is not claimed.
- Release security signoff is not claimed.
- Runtime behavior does not change.
- Safe-program semantics do not change.
- No performance claim is made.
