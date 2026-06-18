# Actor Runtime Post-Scope Production Blockers

Status: post-scope blocker ledger for Actor Runtime Final Production.

This file records what remains outside the current actor runtime production
foundation. It is not an implementation plan for this session and does not
promote the current foundation to a full production actor runtime.

## Current Foundation Boundary

Current evidence is scoped to the actor runtime production foundation:

- Linux-x64 local actor/task runtime foundation evidence.
- Linux-x64 distributed loopback runtime smoke through `actornet`.
- Actor/island ownership and typed mailbox transfer guards.
- Parallel production smoke rows and scheduler prototype evidence.
- Actor foundation gate, validator, docs, and artifact hash checks.

The current foundation still does not claim full Erlang/OTP parity, a full
production actor runtime, cluster membership, reconnect/retry production,
non-Linux distributed actor runtime support, distributed zero-copy pointer or
region transfer, official actor benchmarks, or a formal race/liveness proof.

## Post-Scope Blockers

| Blocker | Current status | Required promotion evidence |
| --- | --- | --- |
| Production multi-threaded actor scheduler | Blocked. The current actor runtime is single-thread cooperative; per-core scheduler work is prototype/model evidence only. | Integrated runtime scheduler implementation, actor/task compatibility tests, stress tests, race-enabled evidence, and validator rejection of prototype-only scheduler promotion. |
| Supervision and restart tree | Blocked. Done-actor sends return checked failure, but there is no supervision, linking, restart policy, mailbox drain lifecycle, or crash tree. | Source API/spec, runtime state machine, failure propagation, restart semantics, lifecycle tests, docs, and release-gated evidence. |
| Cluster membership | Blocked. Distributed actors are bounded to Linux-x64 loopback broker evidence and explicitly do not provide cluster membership. | Membership protocol, discovery or static membership model, node join/leave/failure semantics, partition behavior, executable multi-node evidence, and validator non-fake checks. |
| Reconnect/retry/TLS/auth | Blocked. Current distributed evidence does not claim reconnect/retry production, TLS, public internet routing, or authentication. | Reconnect and retry state machine, authenticated transport, TLS policy, credential handling, negative security tests, broker/node compatibility evidence, and threat-model updates. |
| Non-Linux distributed runtime gates | Blocked. Non-Linux distributed actor runtime support is not claimed. | Target-specific runtime implementation and host execution evidence for macOS/Windows or other promoted targets, plus same-head distributed reports and artifact hashes. |
| Distributed serialization for owned regions | Blocked. `zero_copy_move` is local typed mailbox evidence only; cross-node pointer or region zero-copy is not claimed. | Serialization contract for owned regions, receiver ownership reconstruction, lifetime/provenance checks, negative unsafe payload cases, and distributed smoke evidence. |
| Full structured concurrency | Blocked. Current task group cancellation is cooperative and scoped; it is not a full actor supervision or structured-concurrency model for every blocking API. | Unified task/actor cancellation model, parent/child lifetime rules, deadline propagation, resource cleanup semantics, and cross-API conformance tests. |
| Stronger liveness/race proof | Blocked. Current race-enabled slices and leak cleanup are bounded evidence, not formal or exhaustive liveness/race proof. | Expanded race/stress matrix, liveness properties, starvation/fairness checks, scheduler/transport adversarial tests, and formal or machine-checkable proof artifacts where claimed. |
| Production broker deployment evidence | Blocked. `actornet` loopback broker evidence is executable, but there is no production deployment, operations, upgrade, availability, or hardening proof. | Deployment profile, operational limits, observability, graceful upgrade/shutdown behavior, external network hardening, security review, and release-gated deployment smoke. |

## Promotion Rule

Any future plan may promote one blocker only when it adds code/test/script or
validator evidence for that blocker and updates the public docs without
weakening the nonclaims above. A docs-only change cannot upgrade the actor
runtime foundation to a full production actor runtime.

## Source Anchors

- `docs/spec/actors.md`
- `docs/user/async_actors_guide.md`
- `docs/design/actor_region_transfer.md`
- `docs/audits/actor-runtime-production-boundary-v1.md`
- `docs/audits/actor-runtime-production-foundation-final.md`
