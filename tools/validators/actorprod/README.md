# tools/validators/actorprod

Validator package for scoped actor runtime production foundation evidence.

This boundary owns the `tetra.actor.production_foundation.v1` report contract
used by the actor runtime foundation gate. It composes Linux-x64 distributed
actor runtime evidence, Linux-x64 parallel production evidence, command logs,
artifact hashes, and scoped nonclaims. It must reject docs-only/build-only
evidence, stale git heads, missing subreports, missing hash manifests, and
cross-target distributed actor runtime claims without matching target-host smoke
evidence.
