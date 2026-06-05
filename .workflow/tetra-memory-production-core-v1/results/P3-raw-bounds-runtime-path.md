# P3 Raw Bounds Runtime Path Result

Status: integrated.

Read-only audit summary from Huygens:

- `compiler/internal/runtimeabi/raw_pointer_bounds.go` already models verified
  `core.alloc_bytes` roots, derived offsets, checked external unknown pointers,
  rejected bounds, and raw-slice unknown provenance.
- Existing compiler/runtime tests already cover negative offset, upper-bound,
  access-width, and external unknown cases.
- The missing slice was report projection and schema validation, not a runtime
  rewrite.

Integrated artifacts:

- `compiler/internal/memoryfacts/from_plir.go`
- `compiler/internal/memoryfacts/report.go`
- `compiler/explain_reports_test.go`

Integration decision:

- Preserve existing raw-bounds behavior and project bounded facts into
  `tetra.memory-report.v1`.
