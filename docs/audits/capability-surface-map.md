# P24.0 Capability Surface Map

Status: current-branch P24.0 audit artifact for schema
`tetra.security.review_gate.v1` and scope `p24.0_security_review_gate`.

Primary sources:

- `docs/spec/capabilities.md`
- `docs/spec/effects_capabilities_privacy_v1.md`
- `docs/spec/eco_publishing_v1.md`

## Capability Types

| Capability | Grants | Does not grant | Acquisition |
| --- | --- | --- | --- |
| `cap.mem` | Permission to enter raw memory operations requiring memory capability. | Pointer validity, provenance, allocation lifetime, bounds, alias exclusivity, actor sendability. | `core.cap_mem()` inside `unsafe`. |
| `cap.io` | Permission to enter MMIO-style operations requiring IO capability. | General host IO authority, network access, runtime scheduling authority. | `core.cap_io()` inside `unsafe`. |
| `consent.token` | Static privacy/consent call-shape authorization for the v1 privacy surface. | Cryptographic secrecy, distributed consent enforcement, durable secret storage isolation. | Privacy surface via `core.consent_token()`. |
| `capsule.mem` | Attenuation permission key for memory-sensitive capability groups. | Alias for `mem` effect or automatic `cap.mem` token. | Capsule/effect metadata policy. |
| `capsule.io` | Attenuation permission key for IO-sensitive capability groups. | Alias for `io` effect or automatic `cap.io` token. | Capsule/effect metadata policy. |

## Effect And Permission Rules

| Rule | Evidence |
| --- | --- |
| Canonical `uses` names include `actors`, `alloc`, `budget`, `capability`, `control`, `io`, `islands`, `link`, `mem`, `mmio`, `privacy`, `runtime`, `capsule.io`, and `capsule.mem`. | `docs/spec/effects_capabilities_privacy_v1.md` |
| `cap.io` aliases `io` and `cap.mem` aliases `mem` only as accepted `uses` spelling. | `docs/spec/effects_capabilities_privacy_v1.md` |
| `capsule.io` and `capsule.mem` are permission keys, not effect aliases. | `docs/spec/effects_capabilities_privacy_v1.md` |
| Declaring `uses mem` or `uses io` does not create `cap.mem` or `cap.io`. | `docs/spec/capabilities.md` |
| Capability attenuation checks apply when a function declares attenuation groups such as `effects.cap.mem`, `effects.cap.io`, or `effects.all`. | `docs/spec/effects_capabilities_privacy_v1.md` |
| Stable `lib/core` modules carry `// Effects:` metadata and docs verification fails if public `uses` declarations drift. | `docs/spec/effects_capabilities_privacy_v1.md` |

## Eco Permission Surface

| Surface | Evidence | Boundary |
| --- | --- | --- |
| Capsule manifest | `tetra.capsule.v1` fields in `docs/spec/eco_publishing_v1.md` | Local manifest parsing and compatibility; not remote registry identity. |
| Permissions model | `tetra.eco.permissions.v1` | Dependency checks prevent permission escalation by requiring dependers to include dependency permissions. |
| Lock graph | `Tetra.lock` policy/artifact hash fields | Lock hash includes policy keys, dependency edges, targets, modules, public API hashes, and artifact SHA-256 values. |
| Package publish | `tetra.eco.publish.v1`; `validate-eco-publish` | Stable metadata validates schema/channel, paths, package bytes, and optional trust snapshot hash. |
| Vault/trust | `tetra.eco.trust-snapshot.v1`; `validate-eco-vault` | Local object store hashes and trust metadata; not a global trust federation. |
| Mirror/fetch | `tetra.eco.mirror.v1`; `validate-eco-mirror` | Transport result is validated by package/metadata/trust hashes before local store writes. |

## Abuse Cases

| Abuse case | Expected outcome |
| --- | --- |
| Safe source declares `uses mem` and calls `core.load_i32` without `unsafe` or `cap.mem`. | Checker rejects the operation. |
| Source obtains `cap.mem` and passes a stale or out-of-bounds pointer. | Capability policy does not prove validity; runtime raw-pointer metadata handles only supported verified roots. |
| A capsule dependency requires permissions absent from the depender. | Eco permission validation rejects escalation. |
| Publish metadata contains `../` or absolute paths. | Eco publish/download/mirror validators reject unsafe paths. |
| Trust snapshot hash does not match bytes in the local store or fetched package. | Eco validators reject the mismatch before local trust is recorded. |

## Focused Verification

```sh
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/... -run 'Capability|Effect|Uses|Privacy|Consent|Budget|Capsule' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./cli/... ./tools/... -run 'Eco|Permission|Capsule|Trust' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
```

## Non-Claims

- Capability review does not claim pointer provenance or full memory safety.
- Privacy consent is static-policy and lowering-shape evidence, not
  cryptographic secret storage.
- Eco trust is local metadata validation, not a global trust network.
- Security certification, external penetration testing, CVE-free status, and
  release security signoff are not claimed.
