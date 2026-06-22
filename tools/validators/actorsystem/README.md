# tools/validators/actorsystem

Validator package for the scoped Linux-x64 actor system-message lane evidence.

This boundary owns the `tetra.actor.system_messages.v1` report contract used by
the V1-P01 system-message smoke gate. It accepts P01 fixture/test-hook evidence
for the separate source-level system receive API and isolated runtime-owned
system queue, while rejecting production lifecycle, supervision, cluster, or
authenticated node-down claims that belong to later packets.
