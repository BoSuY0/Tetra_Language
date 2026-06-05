package opt

import "testing"

func TestCoreHotLoopShapeEvidenceReportsRegisterRows(t *testing.T) {
	report, err := CoreHotLoopShapeEvidence()
	if err != nil {
		t.Fatalf("CoreHotLoopShapeEvidence: %v", err)
	}
	if report.SchemaVersion != "tetra.optimizer.hot_loop_shape.v1" {
		t.Fatalf("schema = %q", report.SchemaVersion)
	}
	if !containsString(report.NonClaims, "no C/Rust -O1/-O2 performance parity claim") {
		t.Fatalf("non-claims = %#v, want explicit no performance parity claim", report.NonClaims)
	}

	rows := hotLoopRowsByID(report.Rows)
	for _, id := range []string{"scalar-sum-loop", "scalar-stride-sum-loop", "scalar-sum-squares-loop", "scalar-product-loop", "scalar-max-loop", "scalar-affine-sum-loop", "scalar-countdown-loop", "proof-slice-sum-loop", "proof-slice-stride-sum-loop", "call-sum-loop", "checked-slice-sum-fallback"} {
		if _, ok := rows[id]; !ok {
			t.Fatalf("missing hot-loop row %q in %#v", id, report.Rows)
		}
	}

	scalar := rows["scalar-sum-loop"]
	assertHotLoopRegisterRow(t, scalar, "machine-ir-loop", "scalar-int-loop", []string{"cmp", "branch_if", "add", "inc"})

	stride := rows["scalar-stride-sum-loop"]
	assertHotLoopRegisterRow(t, stride, "machine-ir-stride-loop", "scalar-int-loop", []string{"mov", "cmp", "branch_if", "add"})
	if containsString(stride.RequiredOps, "inc") {
		t.Fatalf("constant-stride row should use explicit stride add, not inc: %#v", stride)
	}

	squares := rows["scalar-sum-squares-loop"]
	assertHotLoopRegisterRow(t, squares, "machine-ir-sum-squares-loop", "scalar-int-sum-squares-loop", []string{"cmp", "branch_if", "mul", "add", "inc"})

	product := rows["scalar-product-loop"]
	assertHotLoopRegisterRow(t, product, "machine-ir-product-loop", "scalar-int-product-loop", []string{"cmp", "branch_if", "add", "mul", "inc"})

	max := rows["scalar-max-loop"]
	assertHotLoopRegisterRow(t, max, "machine-ir-max-loop", "scalar-int-max-loop", []string{"cmp", "branch_if", "mov", "inc"})

	affine := rows["scalar-affine-sum-loop"]
	assertHotLoopRegisterRow(t, affine, "machine-ir-affine-loop", "scalar-int-affine-loop", []string{"cmp", "branch_if", "mul", "add", "inc"})

	countdown := rows["scalar-countdown-loop"]
	assertHotLoopRegisterRow(t, countdown, "machine-ir-countdown-loop", "scalar-int-countdown-loop", []string{"cmp", "branch_if", "add", "sub"})

	slice := rows["proof-slice-sum-loop"]
	assertHotLoopRegisterRow(t, slice, "machine-ir-slice-sum", "scalar-i32-slice-sum", []string{"cmp", "branch_if", "index_load", "add", "inc"})
	if slice.ProofID == "" {
		t.Fatalf("slice row missing proof id: %#v", slice)
	}

	sliceStride := rows["proof-slice-stride-sum-loop"]
	assertHotLoopRegisterRow(t, sliceStride, "machine-ir-slice-stride-sum", "scalar-i32-slice-sum", []string{"mov", "cmp", "branch_if", "index_load", "add"})
	if containsString(sliceStride.RequiredOps, "inc") {
		t.Fatalf("slice constant-stride row should use explicit stride add, not inc: %#v", sliceStride)
	}
	if sliceStride.ProofID == "" {
		t.Fatalf("slice stride row missing proof id: %#v", sliceStride)
	}

	call := rows["call-sum-loop"]
	assertHotLoopRegisterRow(t, call, "machine-ir-call-loop", "scalar-int-call-loop", []string{"cmp", "branch_if", "call", "add", "inc"})
	if call.CallABI != "sysv" {
		t.Fatalf("call ABI = %q, want sysv in row %#v", call.CallABI, call)
	}
}

func TestCoreHotLoopShapeEvidenceReportsCheckedSliceFallback(t *testing.T) {
	report, err := CoreHotLoopShapeEvidence()
	if err != nil {
		t.Fatalf("CoreHotLoopShapeEvidence: %v", err)
	}
	row := hotLoopRowsByID(report.Rows)["checked-slice-sum-fallback"]
	if row.RegisterPath || row.SSAVerified || row.MachinePath != "stack-fallback" {
		t.Fatalf("checked slice fallback row = %#v, want explicit non-register fallback", row)
	}
	if row.Reason != "proof_tag_required_for_slice_sum_register_shape" {
		t.Fatalf("fallback reason = %q, want proof_tag_required_for_slice_sum_register_shape", row.Reason)
	}
	if row.Boundary == "" {
		t.Fatalf("fallback boundary missing: %#v", row)
	}
}

func assertHotLoopRegisterRow(t *testing.T, row HotLoopShapeRow, path string, target string, ops []string) {
	t.Helper()
	if !row.RegisterPath || !row.SSAVerified || row.MachinePath != path || row.MachineTarget != target {
		t.Fatalf("row = %#v, want register path %s target %s with SSA verified", row, path, target)
	}
	if !row.SpillFree || row.StackChurnOps != 0 {
		t.Fatalf("row = %#v, want spill-free and no stack churn", row)
	}
	for _, op := range ops {
		if !containsString(row.RequiredOps, op) {
			t.Fatalf("row ops = %#v, want %q in row %#v", row.RequiredOps, op, row)
		}
	}
	if row.Boundary == "" || row.Evidence == "" {
		t.Fatalf("row missing evidence/boundary: %#v", row)
	}
}

func hotLoopRowsByID(rows []HotLoopShapeRow) map[string]HotLoopShapeRow {
	out := map[string]HotLoopShapeRow{}
	for _, row := range rows {
		out[row.ID] = row
	}
	return out
}
