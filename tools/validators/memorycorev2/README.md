# tools/validators/memorycorev2

Validator package for Memory Core v2 release evidence.

This boundary owns `tetra.memory-core-v2.evidence.v1` validation and the
bounded Memory Core v2 claim scanner. It verifies canonical pipeline evidence,
direct island route coverage, backend support honesty, optimizer proof
metadata, shadow-model removal, and final signoff guards. The CLI wrapper lives
in `tools/cmd/validate-memory-core-v2`.
