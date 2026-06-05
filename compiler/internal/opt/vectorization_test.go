package opt

import (
	"strings"
	"testing"
)

func TestVectorizationCoverageAuditsP17PlanList(t *testing.T) {
	report, err := VectorizationCoverage()
	if err != nil {
		t.Fatalf("VectorizationCoverage: %v", err)
	}
	if report.SchemaVersion != "tetra.optimizer.vectorization.v1" {
		t.Fatalf("schema = %q", report.SchemaVersion)
	}
	if !containsString(report.NonClaims, "no broad SIMD or auto-vectorization claim") {
		t.Fatalf("non-claims = %#v, want explicit broad-SIMD non-claim", report.NonClaims)
	}

	want := []VectorizationID{
		VectorizationSumI32,
		VectorizationCopyU8,
		VectorizationMemsetMemcpy,
		VectorizationMapI32,
	}
	if len(report.Rows) != len(want) {
		t.Fatalf("coverage rows = %d, want %d: %#v", len(report.Rows), len(want), report.Rows)
	}
	byID := map[VectorizationID]VectorizationCoverageRow{}
	for _, row := range report.Rows {
		byID[row.ID] = row
		if row.Name == "" || row.Status == "" || row.Decision == "" || row.Reason == "" || row.Evidence == "" || row.Boundary == "" {
			t.Fatalf("row missing required vectorization evidence: %#v", row)
		}
	}
	for _, id := range want {
		if _, ok := byID[id]; !ok {
			t.Fatalf("missing vectorization row %s", id)
		}
	}

	sum := byID[VectorizationSumI32]
	if sum.Status != VectorizationImplementedNarrow || sum.Decision != VectorizationVectorized {
		t.Fatalf("sum_i32 row = %#v, want implemented_narrow vectorized candidate", sum)
	}
	if !sum.Candidate || !sum.RangeProof || sum.ProofID == "" {
		t.Fatalf("sum_i32 row missing candidate/range-proof evidence: %#v", sum)
	}
	for _, want := range []string{"proof-tagged", "noalias not required", "safe unaligned", "vector backend lowering", "scalar tail", "scalar-i32-slice-sum", "vector-i32x4-slice-sum-plan", "native SIMD", "linux-x64", "translation/differential"} {
		if !strings.Contains(sum.Reason+" "+sum.Evidence+" "+sum.Boundary, want) {
			t.Fatalf("sum_i32 row missing %q: %#v", want, sum)
		}
	}
	if len(sum.MissingFacts) != 0 {
		t.Fatalf("sum_i32 row reports missing facts after native SIMD validation: %#v", sum)
	}
	for _, want := range []string{"native_simd_codegen", "translation_differential_validation"} {
		if !containsString(sum.RequiredFacts, want) {
			t.Fatalf("sum_i32 row missing required fact %q: %#v", want, sum)
		}
	}

	copyU8 := byID[VectorizationCopyU8]
	if copyU8.Status != VectorizationImplementedNarrow || copyU8.Decision != VectorizationVectorized {
		t.Fatalf("copy_u8 row = %#v, want implemented_narrow vectorized candidate", copyU8)
	}
	if !copyU8.Candidate || !copyU8.RangeProof || copyU8.ProofID == "" {
		t.Fatalf("copy_u8 row missing candidate/range-proof evidence: %#v", copyU8)
	}
	for _, want := range []string{"copy-loop", "noalias required", "source/dest disjoint", "safe unaligned", "vector backend lowering", "native SIMD", "linux-x64", "translation/differential", "scalar tail", "scalar-u8-copy", "vector-u8x16-copy-plan"} {
		if !strings.Contains(copyU8.Reason+" "+copyU8.Evidence+" "+copyU8.Boundary, want) {
			t.Fatalf("copy_u8 row missing %q: %#v", want, copyU8)
		}
	}
	if len(copyU8.MissingFacts) != 0 {
		t.Fatalf("copy_u8 row reports missing facts after native SIMD validation: %#v", copyU8)
	}
	for _, want := range []string{"native_simd_codegen", "translation_differential_validation"} {
		if !containsString(copyU8.RequiredFacts, want) {
			t.Fatalf("copy_u8 row missing required fact %q: %#v", want, copyU8)
		}
	}

	mapI32 := byID[VectorizationMapI32]
	if mapI32.Status != VectorizationImplementedNarrow || mapI32.Decision != VectorizationVectorized {
		t.Fatalf("map_i32 row = %#v, want implemented_narrow vectorized candidate", mapI32)
	}
	if !mapI32.Candidate || !mapI32.RangeProof || mapI32.ProofID == "" {
		t.Fatalf("map_i32 row missing candidate/range-proof evidence: %#v", mapI32)
	}
	for _, want := range []string{"map-loop", "single mutable slice in-place", "safe unaligned", "vector backend lowering", "native SIMD", "linux-x64", "translation/differential", "scalar tail", "scalar-i32-map", "vector-i32x4-map-add-const-plan"} {
		if !strings.Contains(mapI32.Reason+" "+mapI32.Evidence+" "+mapI32.Boundary, want) {
			t.Fatalf("map_i32 row missing %q: %#v", want, mapI32)
		}
	}
	if len(mapI32.MissingFacts) != 0 {
		t.Fatalf("map_i32 row reports missing facts after native SIMD validation: %#v", mapI32)
	}
	for _, want := range []string{"native_simd_codegen", "translation_differential_validation"} {
		if !containsString(mapI32.RequiredFacts, want) {
			t.Fatalf("map_i32 row missing required fact %q: %#v", want, mapI32)
		}
	}

	memsetMemcpy := byID[VectorizationMemsetMemcpy]
	if memsetMemcpy.Status != VectorizationImplementedNarrow || memsetMemcpy.Decision != VectorizationVectorized {
		t.Fatalf("memset_memcpy row = %#v, want implemented_narrow vectorized candidate", memsetMemcpy)
	}
	if !memsetMemcpy.Candidate || !memsetMemcpy.RangeProof || memsetMemcpy.ProofID == "" {
		t.Fatalf("memset_memcpy row missing candidate/range-proof evidence: %#v", memsetMemcpy)
	}
	for _, want := range []string{"memset-loop", "memcpy helper via copy []u8", "zero-fill helper", "single mutable slice zero-fill", "safe unaligned", "vector backend lowering", "native SIMD", "linux-x64", "translation/differential", "scalar tail", "scalar-u8-memset-zero", "vector-u8x16-memset-zero-plan"} {
		if !strings.Contains(memsetMemcpy.Reason+" "+memsetMemcpy.Evidence+" "+memsetMemcpy.Boundary, want) {
			t.Fatalf("memset_memcpy row missing %q: %#v", want, memsetMemcpy)
		}
	}
	if len(memsetMemcpy.MissingFacts) != 0 {
		t.Fatalf("memset_memcpy row reports missing facts after native SIMD validation: %#v", memsetMemcpy)
	}
	for _, want := range []string{"native_simd_codegen", "translation_differential_validation"} {
		if !containsString(memsetMemcpy.RequiredFacts, want) {
			t.Fatalf("memset_memcpy row missing required fact %q: %#v", want, memsetMemcpy)
		}
	}
}
