# P18.1 Typed Actor Ownership Transfer v1 Design

## Scope

P18.1 closes the typed actor ownership-transfer slice as bounded compiler
evidence. It does not introduce a new actor scheduler, transport, mailbox
runtime, distributed zero-copy path, or public unsafe-send surface.

The slice records and validates these existing and new anchors:

- semantics `core.send_typed` transfer checks reject borrowed views unless the
  expression explicitly copies, consume island/owned-region payloads, and keep
  sender use-after-move diagnostics stable;
- PLIR records `moved` facts for typed actor sends that move island handles and
  local region-backed slice payloads;
- typed mailbox model/report evidence stays local and bounded through
  `compiler/internal/parallelrt` and `<output>.actor-transfer.json`;
- a new `tetra.actors.ownership_transfer.v1` coverage report rejects fake
  distributed zero-copy, fake runtime-behavior changes, missing PLIR moved
  facts, missing stress diagnostics, and docs-only ownership claims.

## Implementation Plan

1. Add RED tests in `compiler/internal/actorsafety` for the P18.1 coverage
   report and fake-claim validator.
2. Add RED tests in `compiler/internal/plir` proving `core.send_typed` emits an
   `actor_send` operation plus `FactMoved` rows for the moved island and the
   associated region-backed slice.
3. Wire PLIR actor-send recording narrowly from checked AST/type metadata. The
   builder may trust the semantic checker for legality, but the PLIR verifier
   must reject fake moved facts on borrowed values.
4. Add the actorsafety coverage report and validator with row-level required
   facts for every P18.1 rule/task.
5. Update audit/report docs, feature manifest text, sidecars, and run focused
   plus broad verification with the persistent Go cache.

## Non-Claims

- No distributed pointer or region zero-copy is claimed.
- No production actor-runtime promotion is claimed.
- No safe typed actor raw pointer payload is enabled.
- The unsafe pointer row is limited to the internal checker model and existing
  rejection of safe typed actor pointer payloads until a separate audited
  unsafe-send surface exists.
