# tools/validators/memoryprod

Validator package for executable Memory Production Core evidence.

This boundary owns the `tetra.memory.production.v1` report contract. A passing
report must show real Linux-x64 memory runtime execution, ownership and
borrow/consume cases, unsafe `cap.mem`/raw memory rules, bounds diagnostics,
stress/fuzz-style evidence, checked-in examples, classified allocator evidence,
and completion-audit rows.

The required benchmark row is `small heap allocation syscall reduction`. The
smoke command builds a generated Linux-x64 allocation benchmark with
`--emit-alloc-report`, reads the schema-v2 allocation summary, and records an
`allocation_report_estimate` using the `allocation_report_summary` method. This
compares the estimated old mmap-per-allocation baseline against the 64 KiB
chunk-refill path; it is not a runtime RSS, pprof, MemStats, `time -v`, or
`strace` measurement. Release bundles also carry `ram-measurement.json` as a
separate `tetra.memory.ram-measurement.v1` capture artifact. That artifact is
validated as parseable MemStats evidence, or as an explicit `blocked` result
when a measurement tool is unavailable. Passing artifacts now include
`summary` and `metric_samples` rows for `heap_alloc_bytes`,
`bytes_requested`, `bytes_reserved`, `bytes_copied`, `rss_current`,
`rss_peak`, and `per_actor_domain_bytes`. MemStats may provide runtime-measured
heap bytes, but it is rejected as runtime-measured RSS evidence; RSS must use a
real RSS-capable method or remain `unsupported`/`blocked`.
