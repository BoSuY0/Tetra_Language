# tools/validators/postv04prod

Validator package for the combined post-v0.4 production completion audit.

This boundary owns the
`tetra.release.post_v0_4.memory_parallel_ui_completion_audit.v1` report
contract. It combines Memory, Parallelism, UI, and native UI runtime evidence
into one ordered checklist and rejects missing artifacts, unchecked audit rows,
or stale hash manifests.
