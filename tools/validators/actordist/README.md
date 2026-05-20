# tools/validators/actordist

Validator package for executable distributed actor runtime evidence.

This boundary owns the `tetra.actors.distributed-runtime.v1` report contract used
by release and readiness gates. It must reject transport-only, fake,
incomplete, or docs-only distributed actor claims and only accept reports backed
by real Linux-x64 actor runtime processes, an `actornet` loopback TCP broker,
frame counts, process exits, and required cross-node/failure/cancel cases.
