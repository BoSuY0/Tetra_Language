# tools/validators/memoryprod

Validator package for executable Memory Production Core evidence.

This boundary owns the `tetra.memory.production.v1` report contract. A passing
report must show real Linux-x64 memory runtime execution, ownership and
borrow/consume cases, unsafe `cap.mem`/raw memory rules, bounds diagnostics,
stress/fuzz-style evidence, checked-in examples, measured benchmark evidence,
and completion-audit rows.

The required benchmark row is `small heap allocation syscall reduction`. The
smoke command builds a generated Linux-x64 allocation benchmark with
`--emit-alloc-report`, reads the schema-v2 allocation summary, counts
`per_core_small_heap` rows with
`same_core_same_size_class_free_list` reuse policy, and compares the estimated
old mmap-per-allocation baseline against the 64 KiB chunk-refill path.
