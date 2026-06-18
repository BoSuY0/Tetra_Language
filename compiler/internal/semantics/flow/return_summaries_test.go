package flow

import (
	"testing"

	"tetra_language/compiler/internal/semantics/model"
)

func TestCloneReturnRegionSummaryCopiesMap(t *testing.T) {
	in := model.ReturnRegionSummary{"": 0, "field": 1}

	got := CloneReturnRegionSummary(in)
	if !ReturnRegionSummariesEqual(got, in) {
		t.Fatalf("CloneReturnRegionSummary() = %#v, want %#v", got, in)
	}
	got["field"] = 9
	if in["field"] != 1 {
		t.Fatalf("clone mutation changed original: %#v", in)
	}
}

func TestCloneReturnResourceSummaryCopiesProvenanceSlices(t *testing.T) {
	in := model.ReturnResourceSummary{
		"": {
			{ParamIndex: 0, ParamPath: ""},
			{ParamIndex: 1, ParamPath: "field"},
		},
	}

	got := CloneReturnResourceSummary(in)
	if !ReturnResourceSummariesEqual(got, in) {
		t.Fatalf("CloneReturnResourceSummary() = %#v, want %#v", got, in)
	}
	got[""][0].ParamIndex = 7
	if in[""][0].ParamIndex != 0 {
		t.Fatalf("clone mutation changed original: %#v", in)
	}
}

func TestReturnSummaryEqualityChecksKeysAndValues(t *testing.T) {
	if ReturnRegionSummariesEqual(
		model.ReturnRegionSummary{"x": 1},
		model.ReturnRegionSummary{"x": 2},
	) {
		t.Fatalf("ReturnRegionSummariesEqual returned true for different values")
	}
	if ReturnResourceSummariesEqual(
		model.ReturnResourceSummary{"x": {{ParamIndex: 1, ParamPath: "a"}}},
		model.ReturnResourceSummary{"x": {{ParamIndex: 1, ParamPath: "b"}}},
	) {
		t.Fatalf("ReturnResourceSummariesEqual returned true for different provenances")
	}
}
