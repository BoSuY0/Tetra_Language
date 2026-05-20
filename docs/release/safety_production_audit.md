# Tetra Safety Production Audit

Status: achieved for the current `v0.4.0` local safety profile.

Audit date: 2026-05-06.

This audit maps the safety production goal to concrete evidence. It does not
claim mathematical proof of all possible memory, race, privacy, or distributed
runtime behavior. It claims the release-covered local safety model documented in
the current specs and enforced by compiler diagnostics, validators, and gates.

## Objective Restatement

Production safety for the current profile requires:

- ownership/lifetime/`borrow`/`consume`/`inout` checks
- resource finalization and local lifetime join analysis
- callable escape diagnostics for unsafe capture/escape cases
- effects, capabilities, privacy, consent, and budget policy checks
- explicit unsafe boundaries
- actor/task transfer safety
- pointer/MMIO/memory capability gates
- stable diagnostics for rejected safety cases
- complete docs, examples, tests, validators, feature registry evidence, and
  release-gate evidence
- no filler or demo-only safety claims

## Prompt-To-Artifact Checklist

| Requirement | Evidence | Result |
| --- | --- | --- |
| Aggregate production claim exists | `safety.production-core` in `./tetra features --format=json` and `docs/generated/manifest.json` | pass |
| Ownership markers are current | `language.ownership-markers-mvp`; `docs/spec/ownership_v1.md`; `compiler/tests/ownership/ownership_test.go` | pass |
| Lifetime join solver is current | `language.lifetime-ssa`; current-surface docs; focused safety tests | pass |
| Resource finalization is checked | `language.resource-lifetime-mvp`; `compiler/tests/runtime/resource_finalization_test.go` | pass |
| Callable escape is safety-gated | `language.full-first-class-callables`; callable escape diagnostics for mutable by-reference, pointer/resource, and thread-boundary escape | pass |
| Effects propagate and missing uses are diagnostics | `safety.effects-mvp`; `compiler/tests/safety/effects_test.go`; `docs/spec/effects_capabilities_privacy_v1.md` | pass |
| Capability and unsafe boundaries are explicit | `safety.capabilities-mvp`; `docs/spec/unsafe.md`; `docs/spec/capabilities.md`; manifest builtin `unsafe_policy` fields | pass |
| Privacy and consent are checked | `safety.privacy-consent-mvp`; privacy/consent tests and lowering tests | pass |
| Budget checks are deterministic | `safety.budget-mvp`; budget clause tests and lowering verifier coverage | pass |
| Actor/task transfer safety is checked | `actors.task-transfer-safety`; actor/task ownership and resource finalization tests | pass |
| Pointer/MMIO/memory gates are checked | `docs/spec/effects_capabilities_privacy_v1.md`; unsafe/capability/MMIO/mem tests | pass |
| Stable diagnostics are part of the evidence | `tools/cmd/validate-diagnostic`; JSON diagnostic shape gate; safety negative tests | pass |
| Dedicated safety validator exists | `tools/cmd/validate-safety-readiness` | pass |
| Release gate includes safety evidence | `scripts/ci/test-all.sh` step `safety readiness evidence` writes `safety-readiness.json` and runs the aggregate compiler safety command | pass |
| No stale safety production claim remains | `validate-safety-readiness` rejects forbidden filler wording, stale lifetime-planned wording, and stale full-SSA wording in safety docs | pass |

## Fresh Verification Commands

Focused validator and safety compiler gate:

```sh
./tetra features --format=json > /tmp/tetra-safety-features.json
go run ./tools/cmd/validate-safety-readiness \
  --features /tmp/tetra-safety-features.json \
  --current-surface docs/spec/current_supported_surface.md \
  --ownership-spec docs/spec/ownership_v1.md \
  --effects-spec docs/spec/effects_capabilities_privacy_v1.md \
  --out reports/safety-readiness.json
go test ./compiler/... -run 'Ownership|Borrow|Consume|Inout|Lifetime|Resource|Island|Actor|Task|Unsafe|Capability|Effect|Privacy|Consent|Budget|MMIO|Mem' -count=1
```

Docs and manifest gates:

```sh
go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
```

Release-style gate:

```sh
bash scripts/ci/test-all.sh
```

Latest fresh result: pass, 28/28 steps, with `safety readiness evidence` as
step 18 in `reports/test-all-20260506-194123/summary.md`.

## Scope Boundaries

The production safety core is local and release-covered. These remain outside
the current claim until separately designed, implemented, and gated:

- distributed actor/runtime safety
- cryptographic privacy isolation
- broad formal proof of all aliasing/lifetime cases
- synchronization-aware heap/global/thread escape acceptance
- aggregate runtime-wide or distributed budget accounting
- broad safe-code capability construction beyond the documented token gates
