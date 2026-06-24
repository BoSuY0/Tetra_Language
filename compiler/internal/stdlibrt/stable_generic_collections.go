package stdlibrt

import (
	"fmt"
	"strings"
)

type StableGenericCollectionsEvidenceID string

const (
	StableGenericCollectionsTetraSourceAPI          StableGenericCollectionsEvidenceID = "tetra_source_api"
	StableGenericCollectionsValueRepresentation     StableGenericCollectionsEvidenceID = "generic_value_representation"
	StableGenericCollectionsMonomorphizedOperations StableGenericCollectionsEvidenceID = "monomorphized_operations"
	StableGenericCollectionsCommonSpecializations   StableGenericCollectionsEvidenceID = "common_specializations"
	StableGenericCollectionsAllocationReports       StableGenericCollectionsEvidenceID = "allocation_reports"
	StableGenericCollectionsBenchmarkGate           StableGenericCollectionsEvidenceID = "benchmark_gate"
)

type StableGenericCollectionsEvidenceStatus string

const (
	StableGenericCollectionsImplementedNarrow StableGenericCollectionsEvidenceStatus = "implemented_narrow"
	StableGenericCollectionsEvidenceOnly      StableGenericCollectionsEvidenceStatus = "evidence_only"
)

type StableGenericCollectionsCoverageReport struct {
	SchemaVersion                 string                                `json:"schema_version"`
	Rows                          []StableGenericCollectionsEvidenceRow `json:"rows"`
	NonClaims                     []string                              `json:"non_claims"`
	CPlusPlusRustParityClaimed    bool                                  `json:"c_plus_plus_rust_parity_claimed"`
	BroadProductionStdlibClaimed  bool                                  `json:"broad_production_stdlib_claimed"`
	HiddenRuntimeAllocatorClaimed bool                                  `json:"hidden_runtime_allocator_claimed"`
}

type StableGenericCollectionsEvidenceRow struct {
	ID            StableGenericCollectionsEvidenceID     `json:"id"`
	Name          string                                 `json:"name"`
	Status        StableGenericCollectionsEvidenceStatus `json:"status"`
	RequiredFacts []string                               `json:"required_facts,omitempty"`
	MissingFacts  []string                               `json:"missing_facts,omitempty"`
	Evidence      string                                 `json:"evidence"`
	Boundary      string                                 `json:"boundary"`
}

func StableGenericCollectionsCoverage() (StableGenericCollectionsCoverageReport, error) {
	return StableGenericCollectionsCoverageReport{
		SchemaVersion: "tetra.stdlib.generic_collections.v1",
		Rows: []StableGenericCollectionsEvidenceRow{
			stableGenericCollectionsTetraSourceAPIRow(),
			stableGenericCollectionsValueRepresentationRow(),
			stableGenericCollectionsMonomorphizedOperationsRow(),
			stableGenericCollectionsCommonSpecializationsRow(),
			stableGenericCollectionsAllocationReportsRow(),
			stableGenericCollectionsBenchmarkGateRow(),
		},
		NonClaims: []string{
			"C++/Rust parity is not claimed",
			"broad production stdlib is not claimed",
			"collection storage allocation remains caller-owned",
			"P19.0 byte-oriented runtime helpers remain evidence helpers",
		},
		CPlusPlusRustParityClaimed:    false,
		BroadProductionStdlibClaimed:  false,
		HiddenRuntimeAllocatorClaimed: false,
	}, nil
}

func ValidateStableGenericCollectionsCoverage(report StableGenericCollectionsCoverageReport) error {
	if report.SchemaVersion != "tetra.stdlib.generic_collections.v1" {
		return fmt.Errorf("stable generic collections coverage: schema = %q", report.SchemaVersion)
	}
	if report.CPlusPlusRustParityClaimed {
		return fmt.Errorf(
			"stable generic collections coverage: C++/Rust parity claim is forbidden for P19.1",
		)
	}
	if report.BroadProductionStdlibClaimed {
		return fmt.Errorf(
			"stable generic collections coverage: broad production stdlib claim is forbidden for P19.1",
		)
	}
	if report.HiddenRuntimeAllocatorClaimed {
		return fmt.Errorf(
			"stable generic collections coverage: hidden runtime allocator claim is forbidden for P19.1",
		)
	}
	for _, want := range []string{
		"C++/Rust parity is not claimed",
		"broad production stdlib is not claimed",
		"collection storage allocation remains caller-owned",
		"P19.0 byte-oriented runtime helpers remain evidence helpers",
	} {
		if !containsStableGenericCollectionsText(report.NonClaims, want) {
			return fmt.Errorf("stable generic collections coverage: missing non-claim %q", want)
		}
	}

	expectedStatus := map[StableGenericCollectionsEvidenceID]StableGenericCollectionsEvidenceStatus{
		StableGenericCollectionsTetraSourceAPI:          StableGenericCollectionsImplementedNarrow,
		StableGenericCollectionsValueRepresentation:     StableGenericCollectionsImplementedNarrow,
		StableGenericCollectionsMonomorphizedOperations: StableGenericCollectionsImplementedNarrow,
		StableGenericCollectionsCommonSpecializations:   StableGenericCollectionsImplementedNarrow,
		StableGenericCollectionsAllocationReports:       StableGenericCollectionsEvidenceOnly,
		StableGenericCollectionsBenchmarkGate:           StableGenericCollectionsEvidenceOnly,
	}
	if len(report.Rows) != len(expectedStatus) {
		return fmt.Errorf(
			"stable generic collections coverage: row count = %d, want %d",
			len(report.Rows),
			len(expectedStatus),
		)
	}
	rows := map[StableGenericCollectionsEvidenceID]StableGenericCollectionsEvidenceRow{}
	for _, row := range report.Rows {
		wantStatus, ok := expectedStatus[row.ID]
		if !ok {
			return fmt.Errorf("stable generic collections coverage: unexpected row %q", row.ID)
		}
		if _, exists := rows[row.ID]; exists {
			return fmt.Errorf("stable generic collections coverage: duplicate row %q", row.ID)
		}
		if strings.Contains(string(row.Status), "parity") {
			return fmt.Errorf(
				"stable generic collections coverage: benchmark parity status is forbidden for row %q",
				row.ID,
			)
		}
		if row.Status != wantStatus {
			return fmt.Errorf(
				"stable generic collections coverage: row %q status = %q, want %q",
				row.ID,
				row.Status,
				wantStatus,
			)
		}
		if strings.TrimSpace(row.Name) == "" || strings.TrimSpace(row.Evidence) == "" ||
			strings.TrimSpace(row.Boundary) == "" {
			return fmt.Errorf(
				"stable generic collections coverage: row %q missing evidence or boundary",
				row.ID,
			)
		}
		if len(row.RequiredFacts) == 0 {
			return fmt.Errorf(
				"stable generic collections coverage: row %q missing required facts",
				row.ID,
			)
		}
		rows[row.ID] = row
	}
	for id := range expectedStatus {
		if _, ok := rows[id]; !ok {
			return fmt.Errorf("stable generic collections coverage: missing row %q", id)
		}
	}

	checks := []struct {
		id    StableGenericCollectionsEvidenceID
		wants []string
	}{
		{
			StableGenericCollectionsTetraSourceAPI,
			[]string{"lib.core.collections.Vec<T>", "HashMap<K,V>", "caller-owned slices"},
		},
		{
			StableGenericCollectionsValueRepresentation,
			[]string{"genericTypeName", "mangleGenericName", "[]T"},
		},
		{
			StableGenericCollectionsMonomorphizedOperations,
			[]string{"vec_from_slice<T>", "hash_map_from_slices<K,V>", "concrete before lowering"},
		},
		{
			StableGenericCollectionsCommonSpecializations,
			[]string{"hash_map_get_i32_i32_or", "hash_map_get_u8_i32_or"},
		},
		{
			StableGenericCollectionsAllocationReports,
			[]string{"core.make_*", "allocation-plan reports", "no internal allocation"},
		},
		{
			StableGenericCollectionsBenchmarkGate,
			[]string{
				"truth-bench-harness",
				"p19.1_generic_collections",
				"reports/stable-generic-collections-v1/benchmarks/generic-collections-hash-table-manifest.json",
				"reports/stable-generic-collections-v1/benchmarks/generic-collections-hash-table-report.json",
				"Tetra/C++/Rust equivalents",
				"tetra",
				"cpp",
				"rust",
				"allocation/proof/bounds",
				"no parity claim",
			},
		},
	}
	for _, check := range checks {
		row := rows[check.id]
		for _, want := range check.wants {
			if !containsStableGenericCollectionsText(row.RequiredFacts, want) {
				return fmt.Errorf(
					"stable generic collections coverage: row %q missing fact %q",
					row.ID,
					want,
				)
			}
		}
	}
	return nil
}

func stableGenericCollectionsTetraSourceAPIRow() StableGenericCollectionsEvidenceRow {
	return StableGenericCollectionsEvidenceRow{
		ID:     StableGenericCollectionsTetraSourceAPI,
		Name:   "Stable Tetra-source generic collection API",
		Status: StableGenericCollectionsImplementedNarrow,
		RequiredFacts: []string{
			"lib.core.collections.Vec<T> stores caller-owned slices as items: []T plus logical_len",
			"lib.core.collections.HashMap<K,V> stores caller-owned slices as keys: []K and values: []V",
			"caller-owned slices prevent hidden allocator semantics in the v1 source API",
		},
		Evidence: ("lib/core/data/collections.tetra::Vec<T>; lib/core/data/" +
			"collections.tetra::HashMap<K,V>; compiler/tests/semantics/semantics_" +
			"types_protocols_" +
			"test.go::TestStableGenericCollectionSourceAPIMonomorphizesVecAndHashMap"),
		Boundary: ("source-level view API only; not a production allocator-backed " +
			"Vec<T>/HashMap<K,V> runtime"),
	}
}

func stableGenericCollectionsValueRepresentationRow() StableGenericCollectionsEvidenceRow {
	return StableGenericCollectionsEvidenceRow{
		ID:     StableGenericCollectionsValueRepresentation,
		Name:   "Generic value representation",
		Status: StableGenericCollectionsImplementedNarrow,
		RequiredFacts: []string{
			"genericTypeName records instantiated []T, []K, and []V field types",
			"mangleGenericName records deterministic concrete type arguments",
			"generic struct instantiation substitutes []T fields before lowering",
		},
		Evidence: ("compiler/internal/semantics/semantics_" +
			"expressions.go::genericTypeName; compiler/internal/semantics/semantics_" +
			"expressions.go::mangleGenericName; compiler/tests/semantics/semantics_" +
			"types_protocols_" +
			"test.go::TestGenericFunctionInfersThroughGenericStructParameter"),
		Boundary: ("static monomorphized representation only; no runtime generic " +
			"value descriptor or dynamic dispatch table"),
	}
}

func stableGenericCollectionsMonomorphizedOperationsRow() StableGenericCollectionsEvidenceRow {
	return StableGenericCollectionsEvidenceRow{
		ID:     StableGenericCollectionsMonomorphizedOperations,
		Name:   "Monomorphized collection operations",
		Status: StableGenericCollectionsImplementedNarrow,
		RequiredFacts: []string{
			"vec_from_slice<T> monomorphizes to concrete slice and Vec<T> instantiations",
			("hash_map_from_slices<K,V> monomorphizes to concrete key/value " +
				"slice and HashMap<K,V> instantiations"),
			"collection operations are concrete before lowering",
		},
		Evidence: ("compiler/internal/semantics/semantics_" +
			"expressions.go::bindGenericNamedTypeArgs; compiler/tests/semantics/" +
			"semantics_types_protocols_" +
			"test.go::TestStableGenericCollectionSourceAPIMonomorphizesVecAndHashMap"),
		Boundary: ("only operations inferable from arguments are promoted; return-" +
			"only inference and nested generic struct fields remain unsupported"),
	}
}

func stableGenericCollectionsCommonSpecializationsRow() StableGenericCollectionsEvidenceRow {
	return StableGenericCollectionsEvidenceRow{
		ID:     StableGenericCollectionsCommonSpecializations,
		Name:   "Common key/value specializations",
		Status: StableGenericCollectionsImplementedNarrow,
		RequiredFacts: []string{
			"hash_map_get_i32_i32_or provides equality lookup for HashMap<Int,Int>",
			"hash_map_get_u8_i32_or provides equality lookup for HashMap<UInt8,Int>",
		},
		Evidence: ("lib/core/data/collections.tetra::hash_map_get_i32_i32_or; lib/" +
			"core/data/collections.tetra::hash_map_get_u8_i32_or"),
		Boundary: ("no generic hashing/equality protocol, no collision strategy, " +
			"and no resizing policy is claimed"),
	}
}

func stableGenericCollectionsAllocationReportsRow() StableGenericCollectionsEvidenceRow {
	return StableGenericCollectionsEvidenceRow{
		ID:     StableGenericCollectionsAllocationReports,
		Name:   "Allocation report connection",
		Status: StableGenericCollectionsEvidenceOnly,
		RequiredFacts: []string{
			"collection storage comes from caller-side core.make_* or core.island_make_* slice constructors",
			"allocation-plan reports already cover those slice constructors and their storage classes",
			"stable generic collection helpers perform no internal allocation",
		},
		Evidence: ("compiler/internal/memorypipeline/state.go::Build; compiler/internal/allocplan/build.go::Build; " +
			"compiler/compiler_reports.go::allocationPlanReport; lib/core/data/" +
			"collections.tetra"),
		Boundary: ("report linkage only; this row does not add a new allocator, " +
			"runtime mode, or hidden heap path"),
	}
}

func stableGenericCollectionsBenchmarkGateRow() StableGenericCollectionsEvidenceRow {
	return StableGenericCollectionsEvidenceRow{
		ID:     StableGenericCollectionsBenchmarkGate,
		Name:   "Benchmark gate",
		Status: StableGenericCollectionsEvidenceOnly,
		RequiredFacts: []string{
			("truth-bench-harness checks the P19.1 collection benchmark " +
				"subset scope p19.1_generic_collections"),
			("reports/stable-generic-collections-v1/benchmarks/generic-" +
				"collections-hash-table-manifest.json records hash table Tetra/C++/Rust " +
				"equivalents"),
			("reports/stable-generic-collections-v1/benchmarks/generic-" +
				"collections-hash-table-report.json validates tetra, cpp, and rust rows " +
				"under schema tetra.truth.benchmark.v1"),
			("Tetra row includes allocation/proof/bounds report artifacts " +
				"plus a Tetra performance report artifact"),
			"no parity claim is made by this P19.1 benchmark row",
		},
		Evidence: ("tools/cmd/truth-bench-harness/" +
			"main.go::scopeP19GenericCollections; reports/stable-generic-collections-" +
			"v1/benchmarks/generic-collections-hash-table-report.json"),
		Boundary: ("checked dry-run equivalent artifact only; no C++/Rust parity, " +
			"speedup, measured runtime comparison, or official benchmark result is " +
			"claimed"),
	}
}

func containsStableGenericCollectionsText(items []string, want string) bool {
	for _, item := range items {
		if strings.Contains(item, want) {
			return true
		}
	}
	return false
}
