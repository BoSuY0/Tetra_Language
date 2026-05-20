# tools/validators/memoryprod

Validator package for executable Memory Production Core evidence.

This boundary owns the `tetra.memory.production.v1` report contract. A passing
report must show real Linux-x64 memory runtime execution, ownership and
borrow/consume cases, unsafe `cap.mem`/raw memory rules, bounds diagnostics,
stress/fuzz-style evidence, checked-in examples, and completion-audit rows.
