package flow

import "tetra_language/compiler/internal/semantics/model"

func CloneReturnRegionSummary(in model.ReturnRegionSummary) model.ReturnRegionSummary {
	if len(in) == 0 {
		return nil
	}
	out := make(model.ReturnRegionSummary, len(in))
	for path, paramIndex := range in {
		out[path] = paramIndex
	}
	return out
}

func ReturnRegionSummariesEqual(a, b model.ReturnRegionSummary) bool {
	if len(a) != len(b) {
		return false
	}
	for path, left := range a {
		right, ok := b[path]
		if !ok || left != right {
			return false
		}
	}
	return true
}

func CloneReturnResourceSummary(in model.ReturnResourceSummary) model.ReturnResourceSummary {
	if len(in) == 0 {
		return nil
	}
	out := make(model.ReturnResourceSummary, len(in))
	for path, provenances := range in {
		out[path] = append([]model.ResourceProvenance(nil), provenances...)
	}
	return out
}

func ReturnResourceSummariesEqual(a, b model.ReturnResourceSummary) bool {
	if len(a) != len(b) {
		return false
	}
	for path, left := range a {
		right, ok := b[path]
		if !ok || len(left) != len(right) {
			return false
		}
		for i := range left {
			if left[i] != right[i] {
				return false
			}
		}
	}
	return true
}
