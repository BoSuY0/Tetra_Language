# P24.0 Capability Surface Map

Status: current-branch P24.0 audit artifact for schema `tetra.security.review_gate.v1` and scope
`p24.0_security_review_gate`.

Primary sources:

- `docs/spec/runtime/capabilities.md`
- `docs/spec/runtime/effects_capabilities_privacy_v1.md`
- `docs/spec/policy/eco_publishing_v1.md`

## Capability Types

Capability records:

- Capability: `cap.mem`
  Grants: permission to enter raw memory operations requiring memory capability.
  Does not grant: pointer validity, provenance, allocation lifetime, bounds,
  alias exclusivity, or actor sendability.
  Acquisition: `core.cap_mem()` inside `unsafe`.
- Capability: `cap.io`
  Grants: permission to enter MMIO-style operations requiring IO capability.
  Does not grant: general host IO authority, network access, or runtime
  scheduling authority.
  Acquisition: `core.cap_io()` inside `unsafe`.
- Capability: `consent.token`
  Grants: static privacy/consent call-shape authorization for the v1 privacy
  surface.
  Does not grant: cryptographic secrecy, distributed consent enforcement, or
  durable secret storage isolation.
  Acquisition: privacy surface via `core.consent_token()`.
- Capability: `capsule.mem`
  Grants: attenuation permission key for memory-sensitive capability groups.
  Does not grant: alias for `mem` effect or automatic `cap.mem` token.
  Acquisition: capsule/effect metadata policy.
- Capability: `capsule.io`
  Grants: attenuation permission key for IO-sensitive capability groups.
  Does not grant: alias for `io` effect or automatic `cap.io` token.
  Acquisition: capsule/effect metadata policy.

## Effect And Permission Rules

Rule records:

- Rule: canonical `uses` names include `actors`, `alloc`, `budget`,
  `capability`, `control`, `io`, `islands`, `link`, `mem`, `mmio`, `privacy`,
  `runtime`, `capsule.io`, and `capsule.mem`.
  Evidence: `docs/spec/runtime/effects_capabilities_privacy_v1.md`.
- Rule: `cap.io` aliases `io` and `cap.mem` aliases `mem` only as accepted
  `uses` spelling.
  Evidence: `docs/spec/runtime/effects_capabilities_privacy_v1.md`.
- Rule: `capsule.io` and `capsule.mem` are permission keys, not effect aliases.
  Evidence: `docs/spec/runtime/effects_capabilities_privacy_v1.md`.
- Rule: declaring `uses mem` or `uses io` does not create `cap.mem` or
  `cap.io`.
  Evidence: `docs/spec/runtime/capabilities.md`.
- Rule: capability attenuation checks apply when a function declares
  attenuation groups such as `effects.cap.mem`, `effects.cap.io`, or
  `effects.all`.
  Evidence: `docs/spec/runtime/effects_capabilities_privacy_v1.md`.
- Rule: stable `lib/core` modules carry `// Effects:` metadata, and docs
  verification fails if public `uses` declarations drift.
  Evidence: `docs/spec/runtime/effects_capabilities_privacy_v1.md`.

## Eco Permission Surface

Surface records:

- Surface: capsule manifest.
  Evidence: `tetra.capsule.v1` fields in
  `docs/spec/policy/eco_publishing_v1.md`.
  Boundary: local manifest parsing and compatibility; not remote registry
  identity.
- Surface: permissions model.
  Evidence: `tetra.eco.permissions.v1`.
  Boundary: dependency checks prevent permission escalation by requiring
  dependers to include dependency permissions.
- Surface: lock graph.
  Evidence: `Tetra.lock` policy/artifact hash fields.
  Boundary: lock hash includes policy keys, dependency edges, targets,
  modules, public API hashes, and artifact SHA-256 values.
- Surface: package publish.
  Evidence: `tetra.eco.publish.v1`; `validate-eco-publish`.
  Boundary: stable metadata validates schema/channel, paths, package bytes,
  and optional trust snapshot hash.
- Surface: vault/trust.
  Evidence: `tetra.eco.trust-snapshot.v1`; `validate-eco-vault`.
  Boundary: local object store hashes and trust metadata; not a global trust
  federation.
- Surface: mirror/fetch.
  Evidence: `tetra.eco.mirror.v1`; `validate-eco-mirror`.
  Boundary: transport result is validated by package, metadata, and trust
  hashes before local store writes.

## Abuse Cases

Abuse case records:

- Abuse case: safe source declares `uses mem` and calls `core.load_i32`
  without `unsafe` or `cap.mem`.
  Expected outcome: checker rejects the operation.
- Abuse case: source obtains `cap.mem` and passes a stale or out-of-bounds
  pointer.
  Expected outcome: capability policy does not prove validity; runtime
  raw-pointer metadata handles only supported verified roots.
- Abuse case: a capsule dependency requires permissions absent from the
  depender.
  Expected outcome: Eco permission validation rejects escalation.
- Abuse case: publish metadata contains `../` or absolute paths.
  Expected outcome: Eco publish/download/mirror validators reject unsafe paths.
- Abuse case: trust snapshot hash does not match bytes in the local store or
  fetched package.
  Expected outcome: Eco validators reject the mismatch before local trust is
  recorded.

## Focused Verification

```sh
GOCACHE=$(pwd)/.cache/go-build-ideal-plan \
  go test ./compiler/... \
  -run 'Capability|Effect|Uses|Privacy|Consent|Budget|Capsule' \
  -count=1

GOCACHE=$(pwd)/.cache/go-build-ideal-plan \
  go test ./cli/... ./tools/... \
  -run 'Eco|Permission|Capsule|Trust' \
  -count=1

GOCACHE=$(pwd)/.cache/go-build-ideal-plan \
  go run ./tools/cmd/verify-docs \
  --manifest docs/generated/manifest.json
```

## Non-Claims

- Capability review does not claim pointer provenance or full memory safety.
- Privacy consent is static-policy and lowering-shape evidence, not cryptographic secret storage.
- Eco trust is local metadata validation, not a global trust network.
- Security certification, external penetration testing, CVE-free status, and release security
  signoff are not claimed.
