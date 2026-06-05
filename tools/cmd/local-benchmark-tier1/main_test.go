package main

import "testing"

func TestBuildSpecsCoversP20MatrixAndRequiredCompilers(t *testing.T) {
	specs := buildBenchmarkSpecs("reports/local-benchmark-tier1-v1")
	wantRows := len(requiredP20Categories) * len(requiredLanguages)
	if len(specs) != wantRows {
		t.Fatalf("specs = %d, want %d", len(specs), wantRows)
	}
	seen := map[string]bool{}
	for _, spec := range specs {
		key := spec.Category + "\x00" + spec.Language
		if seen[key] {
			t.Fatalf("duplicate spec for %s/%s", spec.Category, spec.Language)
		}
		seen[key] = true
		if spec.AlgorithmID == "" || spec.InputDescription == "" || spec.Source == "" {
			t.Fatalf("spec %s missing equivalence/source metadata: %#v", spec.Name, spec)
		}
		switch spec.Language {
		case "tetra":
			if spec.BuildCommandKind != "tetra" {
				t.Fatalf("tetra spec %s build kind = %q", spec.Name, spec.BuildCommandKind)
			}
		case "c":
			if !containsSequence(spec.BuildArgs, "clang", "-O3") {
				t.Fatalf("c spec %s build args = %#v, want clang -O3", spec.Name, spec.BuildArgs)
			}
		case "cpp":
			if !containsSequence(spec.BuildArgs, "clang++", "-O3") {
				t.Fatalf("cpp spec %s build args = %#v, want clang++ -O3", spec.Name, spec.BuildArgs)
			}
		case "rust":
			if !containsSequence(spec.BuildArgs, "rustc", "-C", "opt-level=3") {
				t.Fatalf("rust spec %s build args = %#v, want rustc -C opt-level=3", spec.Name, spec.BuildArgs)
			}
		default:
			t.Fatalf("unexpected language %q", spec.Language)
		}
	}
	for _, category := range requiredP20Categories {
		for _, language := range requiredLanguages {
			if !seen[category+"\x00"+language] {
				t.Fatalf("missing spec for %s/%s", category, language)
			}
		}
	}
}

func TestClassifyCategoryPrefersEvidenceBlockers(t *testing.T) {
	tetra := benchmarkRow{Language: "tetra", Status: "measured", MedianRuntimeMS: 1, TetraMetadata: &tetraMetadata{BackendPath: "fallback"}}
	rows := []benchmarkRow{
		tetra,
		{Language: "c", Status: "measured", MedianRuntimeMS: 10},
		{Language: "cpp", Status: "measured", MedianRuntimeMS: 10},
		{Language: "rust", Status: "measured", MedianRuntimeMS: 10},
	}
	classification, _ := classifyCategory("integer loops", rows, 0.20)
	if classification != "blocked by fallback backend" {
		t.Fatalf("fallback classification = %q", classification)
	}

	rows[0].TetraMetadata = &tetraMetadata{BackendPath: "register", HeapAllocations: 1}
	classification, _ = classifyCategory("allocation", rows, 0.20)
	if classification != "blocked by heap allocation" {
		t.Fatalf("heap classification = %q", classification)
	}

	rows[0].TetraMetadata = &tetraMetadata{BackendPath: "register", BoundsLeft: 1}
	classification, _ = classifyCategory("slice sum", rows, 0.20)
	if classification != "blocked by bounds check" {
		t.Fatalf("bounds classification = %q", classification)
	}

	rows[0].TetraMetadata = &tetraMetadata{BackendPath: "register"}
	classification, _ = classifyCategory("actor ping-pong", rows, 0.20)
	if classification != "blocked by actor/runtime limitation" {
		t.Fatalf("actor classification = %q", classification)
	}
}

func TestClassifySpecialMetricCategoriesUseMetricSpecificEvidence(t *testing.T) {
	rows := []benchmarkRow{
		{Language: "tetra", Status: "measured", MedianRuntimeMS: 100, CompileTimeMS: 10, BinarySizeBytes: 10, TetraMetadata: &tetraMetadata{BackendPath: "register"}},
		{Language: "c", Status: "measured", MedianRuntimeMS: 1, CompileTimeMS: 100, BinarySizeBytes: 100},
		{Language: "cpp", Status: "measured", MedianRuntimeMS: 1, CompileTimeMS: 100, BinarySizeBytes: 100},
		{Language: "rust", Status: "measured", MedianRuntimeMS: 1, CompileTimeMS: 100, BinarySizeBytes: 100},
	}
	classification, reason := classifyCategory("binary size", rows, 0.20)
	if classification != "comparable" || !containsAll(reason, "binary_size_bytes", "10") {
		t.Fatalf("binary size classification/reason = %q/%q", classification, reason)
	}
	classification, reason = classifyCategory("compile time", rows, 0.20)
	if classification != "faster than C/C++/Rust locally" || !containsAll(reason, "compile_time_ms", "10") {
		t.Fatalf("compile time classification/reason = %q/%q", classification, reason)
	}
}

func containsSequence(items []string, want ...string) bool {
	if len(want) == 0 || len(want) > len(items) {
		return false
	}
	for i := 0; i <= len(items)-len(want); i++ {
		ok := true
		for j := range want {
			if items[i+j] != want[j] {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}

func containsAll(text string, wants ...string) bool {
	for _, want := range wants {
		if !containsSubstring(text, want) {
			return false
		}
	}
	return true
}

func containsSubstring(text string, want string) bool {
	for i := 0; i+len(want) <= len(text); i++ {
		if text[i:i+len(want)] == want {
			return true
		}
	}
	return false
}
