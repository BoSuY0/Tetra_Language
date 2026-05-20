# tools/validators/parallelprod

Validator package for executable Parallelism Production Core evidence.

This boundary owns the `tetra.parallel.production.v1` report contract. A
passing report must show real Linux-x64 task scheduler and actor runtime
evidence, join/cancel/deadline/select/group lifecycle cases, mailbox
backpressure and failure handling, transfer/race-safety diagnostics, stress
evidence, and safe/unsafe/forbidden boundary coverage.
