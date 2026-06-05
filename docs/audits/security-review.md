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

| Surface | Primary evidence | Current boundary |
| --- | --- | --- |
| Unsafe APIs | `docs/spec/unsafe.md`; `examples/flow_unsafe_cap_mem_smoke.tetra`; `lib/core/capability.tetra` | Inventory and policy review; not a proof that every unsafe caller is memory safe. |
| Capabilities | `docs/spec/capabilities.md`; `docs/spec/effects_capabilities_privacy_v1.md` | `uses` declarations audit effects and do not grant `cap.mem` or `cap.io`. |
| Memory allocator | `compiler/internal/runtimeabi/allocation_contract.go`; `compiler/internal/runtimeabi/raw_pointer_bounds.go` | Runtime ABI evidence with raw-pointer bounds metadata; not a formal memory-safety proof. |
| Network runtime | `compiler/internal/netrt/io_reactor_coverage.go`; `compiler/internal/netrt/netrt_linux.go` | Linux epoll v1 and focused reactor evidence; no full production web-stack or cross-platform parity claim. |
| Actor runtime | `compiler/internal/actorsrt/production_boundary.go`; `docs/spec/actors.md` | Current actor limits and scheduler prototype evidence; not a production multi-threaded actor runtime. |
| DB protocol | `compiler/internal/pgrt/production_postgres_coverage.go`; `compiler/internal/pgrt/wire.go`; `compiler/internal/pgrt/scram.go`; `compiler/internal/pgrt/pool.go` | Local PostgreSQL wire-protocol evidence; no TLS, channel binding, or external production database deployment claim. |
| Package/Eco system | `docs/spec/eco_publishing_v1.md`; `cli/cmd/tetra/eco_publish.go`; `tools/cmd/validate-eco-*` | Local lock/hash/trust metadata validation; no global package trust network. |
| Build scripts | `scripts/release/v1_0/security-review.sh`; `tools/scriptstest/security_review_test.go` | Release signoff validator exists separately from this P24.0 audit artifact. |
| Supply chain | `go.sum`; Eco lock/publish/vault/mirror/unpack validators | Local checksum and metadata validation; no SLSA, CVE-free, or external registry trust claim. |

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
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler -run 'P24SecurityReviewGate' -count=1
```

Related runtime/package evidence:

```sh
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/runtimeabi ./compiler/internal/netrt ./compiler/internal/actorsrt ./compiler/internal/parallelrt ./compiler/internal/pgrt -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./tools/scriptstest -run 'SecurityReview' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./tools/cmd/validate-eco-lock ./tools/cmd/validate-eco-publish ./tools/cmd/validate-eco-vault ./tools/cmd/validate-eco-mirror ./tools/cmd/validate-eco-unpack -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/tests/semantics -run 'FeatureRegistry' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
```

## Residual Risks

| Surface | Residual risk | Current mitigation |
| --- | --- | --- |
| Unsafe APIs | Capability tokens authorize entry to raw operations but do not prove pointer validity, lifetime, aliasing, or actor sendability. | Unsafe syntax, effects, capability arguments, raw-pointer bounds metadata, and focused safety tests. |
| Network runtime | Linux epoll coverage does not imply kqueue, IOCP, WASI/web parity, or production network hardening. | `netrt.IOReactorCoverage` records explicit platform non-claims. |
| Actor runtime | Message-pool exhaustion/reclamation, full cancellation, race-safety proof, and production scheduler integration remain incomplete. | `actorsrt.ActorRuntimeProductionBoundaryAudit` records blockers. |
| DB protocol | Local SCRAM and frame parsing evidence does not include TLS, channel binding, or external production deployment review. | `pgrt.ProductionPostgresCoverage` records protocol and benchmark honesty boundaries. |
| Eco supply chain | Local hashes and trust snapshots do not establish a global trust federation. | Eco validators reject unsafe paths, unknown fields, and hash mismatches before local writes. |

## Non-Claims

- Security certification is not claimed.
- External penetration test is not claimed.
- CVE-free status is not claimed.
- Release security signoff is not claimed.
- Runtime behavior does not change.
- Safe-program semantics do not change.
- No performance claim is made.
