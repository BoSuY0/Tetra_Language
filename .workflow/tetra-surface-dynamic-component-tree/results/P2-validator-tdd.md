# P2 Validator TDD Result

Added RED tests for the discovered gaps in `tools/validators/surface/report_test.go`
and `tools/scriptstest/release_surface_smoke_test.go`. Implemented stricter
validator checks in `tools/validators/surface/report.go`, updated report
fixtures and `tools/cmd/surface-runtime-smoke/main.go`, and added
`component.tree_hit_test_static` with example use.

Evidence:
- `go test ./tools/validators/surface -run 'ComponentTree(FocusOrder|MissingFocusWrap|ButtonActionWithoutFocusedKeyRoute|RowChildrenOverlap|ColumnChildrenOutOfOrder|SurfaceRuntimeEvidence)' -count=1`
- `go test ./tools/cmd/surface-runtime-smoke -run 'ComponentTreeModesProduceTreeEvidence' -count=1`
- `go test ./tools/scriptstest -run 'SurfaceTreeAppRoutesPointerHitTestThroughComponentTreeHelper' -count=1`
