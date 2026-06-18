package stdlibrt

import (
	"fmt"
	"strings"
)

type RegionAwareStdlibEvidenceID string

const (
	RegionStdlibStringBuilder        RegionAwareStdlibEvidenceID = "string_builder"
	RegionStdlibVecArray             RegionAwareStdlibEvidenceID = "vec_array"
	RegionStdlibHashMap              RegionAwareStdlibEvidenceID = "hash_map"
	RegionStdlibJSONParserBuilder    RegionAwareStdlibEvidenceID = "json_parser_builder"
	RegionStdlibHTTPParserBuilder    RegionAwareStdlibEvidenceID = "http_parser_builder"
	RegionStdlibPostgreSQLProtocol   RegionAwareStdlibEvidenceID = "postgresql_protocol_helpers"
	RegionStdlibBuffers              RegionAwareStdlibEvidenceID = "buffers"
	RegionStdlibRingBuffers          RegionAwareStdlibEvidenceID = "ring_buffers"
	RegionStdlibBorrowedViews        RegionAwareStdlibEvidenceID = "borrowed_views"
	RegionStdlibCopyReports          RegionAwareStdlibEvidenceID = "copy_only_when_needed_reports"
	RegionStdlibNoHiddenHeapReports  RegionAwareStdlibEvidenceID = "no_hidden_heap_reports"
	RegionStdlibProductionBoundaries RegionAwareStdlibEvidenceID = "production_boundaries"
)

type RegionAwareStdlibEvidenceStatus string

const (
	RegionAwareStdlibImplementedNarrow  RegionAwareStdlibEvidenceStatus = "implemented_narrow"
	RegionAwareStdlibEvidenceOnly       RegionAwareStdlibEvidenceStatus = "evidence_only"
	RegionAwareStdlibBoundaryDocumented RegionAwareStdlibEvidenceStatus = "boundary_documented"
)

type RegionAwareStdlibCoverageReport struct {
	SchemaVersion                    string                         `json:"schema_version"`
	Rows                             []RegionAwareStdlibEvidenceRow `json:"rows"`
	NonClaims                        []string                       `json:"non_claims"`
	FullProductionWebStackClaimed    bool                           `json:"full_production_web_stack_claimed"`
	OfficialTechEmpowerResultClaimed bool                           `json:"official_techempower_result_claimed"`
	ProductionPostgreSQLStackClaimed bool                           `json:"production_postgresql_stack_claimed"`
	GenericCollectionAPIClaimed      bool                           `json:"generic_collection_api_claimed"`
	RuntimeBehaviorChanged           bool                           `json:"runtime_behavior_changed"`
}

type RegionAwareStdlibEvidenceRow struct {
	ID                  RegionAwareStdlibEvidenceID     `json:"id"`
	Name                string                          `json:"name"`
	Status              RegionAwareStdlibEvidenceStatus `json:"status"`
	RequiredFacts       []string                        `json:"required_facts,omitempty"`
	MissingFacts        []string                        `json:"missing_facts,omitempty"`
	Evidence            string                          `json:"evidence"`
	Boundary            string                          `json:"boundary"`
	RegionFirst         bool                            `json:"region_first"`
	BorrowedViews       bool                            `json:"borrowed_views"`
	CopyOnlyWhenNeeded  bool                            `json:"copy_only_when_needed"`
	HiddenHeapInHotPath bool                            `json:"hidden_heap_in_hot_path"`
}

func RegionAwareStdlibCoverage() (RegionAwareStdlibCoverageReport, error) {
	return RegionAwareStdlibCoverageReport{
		SchemaVersion: "tetra.stdlib.region_aware.v1",
		Rows: []RegionAwareStdlibEvidenceRow{
			regionStdlibStringBuilderRow(),
			regionStdlibVecArrayRow(),
			regionStdlibHashMapRow(),
			regionStdlibJSONParserBuilderRow(),
			regionStdlibHTTPParserBuilderRow(),
			regionStdlibPostgreSQLProtocolRow(),
			regionStdlibBuffersRow(),
			regionStdlibRingBuffersRow(),
			regionStdlibBorrowedViewsRow(),
			regionStdlibCopyReportsRow(),
			regionStdlibNoHiddenHeapReportsRow(),
			regionStdlibProductionBoundariesRow(),
		},
		NonClaims: []string{
			"full production web stack is not claimed",
			"official TechEmpower result is not claimed",
			"production PostgreSQL stack is not claimed",
			"generic collection API is not claimed",
			"runtime behavior is unchanged by this P19.0 evidence layer",
		},
		FullProductionWebStackClaimed:    false,
		OfficialTechEmpowerResultClaimed: false,
		ProductionPostgreSQLStackClaimed: false,
		GenericCollectionAPIClaimed:      false,
		RuntimeBehaviorChanged:           false,
	}, nil
}

func ValidateRegionAwareStdlibCoverage(report RegionAwareStdlibCoverageReport) error {
	if report.SchemaVersion != "tetra.stdlib.region_aware.v1" {
		return fmt.Errorf("region-aware stdlib coverage: schema = %q", report.SchemaVersion)
	}
	if report.FullProductionWebStackClaimed {
		return fmt.Errorf(
			"region-aware stdlib coverage: full production web stack claim is forbidden for P19.0",
		)
	}
	if report.OfficialTechEmpowerResultClaimed {
		return fmt.Errorf(
			"region-aware stdlib coverage: official TechEmpower result claim is forbidden for P19.0",
		)
	}
	if report.ProductionPostgreSQLStackClaimed {
		return fmt.Errorf(
			"region-aware stdlib coverage: production PostgreSQL stack claim is forbidden for P19.0",
		)
	}
	if report.GenericCollectionAPIClaimed {
		return fmt.Errorf(
			"region-aware stdlib coverage: generic collection API claim is forbidden for P19.0",
		)
	}
	if report.RuntimeBehaviorChanged {
		return fmt.Errorf(
			"region-aware stdlib coverage: runtime behavior change claim is forbidden for P19.0",
		)
	}
	for _, want := range []string{
		"full production web stack is not claimed",
		"official TechEmpower result is not claimed",
		"production PostgreSQL stack is not claimed",
		"generic collection API is not claimed",
		"runtime behavior is unchanged",
	} {
		if !containsRegionStdlibText(report.NonClaims, want) {
			return fmt.Errorf("region-aware stdlib coverage: missing non-claim %q", want)
		}
	}

	expectedStatus := map[RegionAwareStdlibEvidenceID]RegionAwareStdlibEvidenceStatus{
		RegionStdlibStringBuilder:        RegionAwareStdlibImplementedNarrow,
		RegionStdlibVecArray:             RegionAwareStdlibImplementedNarrow,
		RegionStdlibHashMap:              RegionAwareStdlibImplementedNarrow,
		RegionStdlibJSONParserBuilder:    RegionAwareStdlibImplementedNarrow,
		RegionStdlibHTTPParserBuilder:    RegionAwareStdlibImplementedNarrow,
		RegionStdlibPostgreSQLProtocol:   RegionAwareStdlibImplementedNarrow,
		RegionStdlibBuffers:              RegionAwareStdlibImplementedNarrow,
		RegionStdlibRingBuffers:          RegionAwareStdlibImplementedNarrow,
		RegionStdlibBorrowedViews:        RegionAwareStdlibEvidenceOnly,
		RegionStdlibCopyReports:          RegionAwareStdlibEvidenceOnly,
		RegionStdlibNoHiddenHeapReports:  RegionAwareStdlibEvidenceOnly,
		RegionStdlibProductionBoundaries: RegionAwareStdlibBoundaryDocumented,
	}
	if len(report.Rows) != len(expectedStatus) {
		return fmt.Errorf(
			"region-aware stdlib coverage: row count = %d, want %d",
			len(report.Rows),
			len(expectedStatus),
		)
	}
	rows := map[RegionAwareStdlibEvidenceID]RegionAwareStdlibEvidenceRow{}
	for _, row := range report.Rows {
		wantStatus, ok := expectedStatus[row.ID]
		if !ok {
			return fmt.Errorf("region-aware stdlib coverage: unexpected row %q", row.ID)
		}
		if _, exists := rows[row.ID]; exists {
			return fmt.Errorf("region-aware stdlib coverage: duplicate row %q", row.ID)
		}
		rows[row.ID] = row
		if row.Status != wantStatus {
			return fmt.Errorf(
				"region-aware stdlib coverage: row %q status = %q, want %q",
				row.ID,
				row.Status,
				wantStatus,
			)
		}
		if strings.TrimSpace(row.Name) == "" || strings.TrimSpace(row.Evidence) == "" ||
			strings.TrimSpace(row.Boundary) == "" {
			return fmt.Errorf(
				"region-aware stdlib coverage: row %q missing evidence or boundary",
				row.ID,
			)
		}
		if len(row.RequiredFacts) == 0 {
			return fmt.Errorf("region-aware stdlib coverage: row %q missing required facts", row.ID)
		}
		if row.HiddenHeapInHotPath {
			return fmt.Errorf(
				"region-aware stdlib coverage: row %q claims hidden heap in hot path",
				row.ID,
			)
		}
	}
	for id := range expectedStatus {
		if _, ok := rows[id]; !ok {
			return fmt.Errorf("region-aware stdlib coverage: missing row %q", id)
		}
	}

	checks := []struct {
		id    RegionAwareStdlibEvidenceID
		wants []string
	}{
		{
			RegionStdlibStringBuilder,
			[]string{"NewStringBuilder", "StorageRegion", "borrowed BytesView"},
		},
		{RegionStdlibVecArray, []string{"NewVecBytes", "Vec/Array equivalent", "CopyOperations"}},
		{RegionStdlibHashMap, []string{"NewHashMapBytes", "open addressing", "BytesCopied"}},
		{
			RegionStdlibJSONParserBuilder,
			[]string{"ParseValueView", "AppendValue", "BorrowedStrings", "CopiedStrings"},
		},
		{
			RegionStdlibHTTPParserBuilder,
			[]string{"ParseRequestViewInRegion", "AppendResponseWithReport", "HeaderViewsBorrowed"},
		},
		{
			RegionStdlibPostgreSQLProtocol,
			[]string{"AppendBindFormat", "DecodeDataRowBorrowed", "RowDecodeReport"},
		},
		{RegionStdlibBuffers, []string{"NewByteBuffer", "StorageReport"}},
		{RegionStdlibRingBuffers, []string{"NewRingBuffer", "wrapped readable window", "Copied"}},
		{RegionStdlibBorrowedViews, []string{"BytesView", "StorageBorrowed", "StorageRegion"}},
		{
			RegionStdlibCopyReports,
			[]string{"copy only when needed", "CopyOperations", "BytesCopied"},
		},
		{RegionStdlibNoHiddenHeapReports, []string{"HiddenHeap=false", "HeapAllocations=0"}},
		{
			RegionStdlibProductionBoundaries,
			[]string{
				"no full production web stack",
				"no official TechEmpower",
				"no production PostgreSQL",
			},
		},
	}
	for _, check := range checks {
		row := rows[check.id]
		for _, want := range check.wants {
			if !containsRegionStdlibText(row.RequiredFacts, want) {
				return fmt.Errorf(
					"region-aware stdlib coverage: row %q missing fact %q",
					row.ID,
					want,
				)
			}
		}
	}
	return nil
}

func regionStdlibStringBuilderRow() RegionAwareStdlibEvidenceRow {
	return RegionAwareStdlibEvidenceRow{
		ID:     RegionStdlibStringBuilder,
		Name:   "Region StringBuilder",
		Status: RegionAwareStdlibImplementedNarrow,
		RequiredFacts: []string{
			"NewStringBuilder allocates StringBuilder storage from StorageRegion when a Region is provided",
			"StringBuilder.View returns a borrowed BytesView over region storage",
			"StorageReport records HiddenHeap=false for region-backed StringBuilder hot paths",
		},
		Evidence: ("compiler/internal/stdlibrt/collections.go::NewStringBuilder; " +
			"compiler/internal/stdlibrt/collections_" +
			"test.go::TestRegionAwareStdlibHelpersUseRegionReportsAndBorrowedViews"),
		Boundary: ("byte-oriented StringBuilder helper only; this is not a public " +
			"generic string API or Unicode rope implementation"),
		RegionFirst:         true,
		BorrowedViews:       true,
		CopyOnlyWhenNeeded:  true,
		HiddenHeapInHotPath: false,
	}
}

func regionStdlibVecArrayRow() RegionAwareStdlibEvidenceRow {
	return RegionAwareStdlibEvidenceRow{
		ID:     RegionStdlibVecArray,
		Name:   "Region Vec/Array equivalent",
		Status: RegionAwareStdlibImplementedNarrow,
		RequiredFacts: []string{
			"NewVecBytes provides a narrow Vec/Array equivalent for []u8 storage",
			"AppendBorrowed reports CopyOperations only when borrowed bytes must be retained",
			"VecBytes.View returns a borrowed BytesView over region storage",
		},
		Evidence: ("compiler/internal/stdlibrt/collections.go::NewVecBytes; " +
			"compiler/internal/stdlibrt/collections_" +
			"test.go::TestRegionAwareStdlibHelpersUseRegionReportsAndBorrowedViews"),
		Boundary: ("VecBytes is a narrow runtime helper for bytes; broad generic " +
			"Vec<T> syntax/API remains outside P19.0"),
		RegionFirst:         true,
		BorrowedViews:       true,
		CopyOnlyWhenNeeded:  true,
		HiddenHeapInHotPath: false,
	}
}

func regionStdlibHashMapRow() RegionAwareStdlibEvidenceRow {
	return RegionAwareStdlibEvidenceRow{
		ID:     RegionStdlibHashMap,
		Name:   "Region HashMap",
		Status: RegionAwareStdlibImplementedNarrow,
		RequiredFacts: []string{
			"NewHashMapBytes stores fixed-capacity byte keys and values with open addressing",
			("HashMapBytes.Put records BytesCopied and CopyOperations because " +
				"retained key/value bytes must be copied"),
			"HashMapBytes.Get returns a borrowed BytesView over retained region storage",
		},
		Evidence: ("compiler/internal/stdlibrt/collections.go::NewHashMapBytes; " +
			"compiler/internal/stdlibrt/collections_" +
			"test.go::TestRegionAwareStdlibHelpersUseRegionReportsAndBorrowedViews"),
		Boundary: ("HashMapBytes is a fixed-capacity byte-key helper; generic " +
			"HashMap<K,V>, deletion, resizing, and hashing policy tuning remain " +
			"future work"),
		RegionFirst:         true,
		BorrowedViews:       true,
		CopyOnlyWhenNeeded:  true,
		HiddenHeapInHotPath: false,
	}
}

func regionStdlibJSONParserBuilderRow() RegionAwareStdlibEvidenceRow {
	return RegionAwareStdlibEvidenceRow{
		ID:     RegionStdlibJSONParserBuilder,
		Name:   "Region-aware JSON parser/builder",
		Status: RegionAwareStdlibImplementedNarrow,
		RequiredFacts: []string{
			"ParseValueView borrows unescaped JSON strings and reports BorrowedStrings",
			("escaped JSON strings copy into the request Region and report " +
				"CopiedStrings without HeapAllocations when a Region is supplied"),
			("AppendValue and AppendString provide deterministic JSON builder " +
				"evidence; generic ParseValue remains heap-backed and is not the hot " +
				"borrowed-view path"),
		},
		Evidence: ("compiler/internal/jsonrt/view.go::ParseValueView; compiler/" +
			"internal/jsonrt/json.go::AppendValue; compiler/internal/jsonrt/" +
			"json.go::AppendString; compiler/internal/jsonrt/view_test.go"),
		Boundary: ("JSON evidence is a narrow parser/builder slice; it is not a " +
			"complete DOM-free streaming JSON runtime"),
		RegionFirst:         true,
		BorrowedViews:       true,
		CopyOnlyWhenNeeded:  true,
		HiddenHeapInHotPath: false,
	}
}

func regionStdlibHTTPParserBuilderRow() RegionAwareStdlibEvidenceRow {
	return RegionAwareStdlibEvidenceRow{
		ID:     RegionStdlibHTTPParserBuilder,
		Name:   "Region-aware HTTP parser/builder",
		Status: RegionAwareStdlibImplementedNarrow,
		RequiredFacts: []string{
			("ParseRequestViewInRegion reports request-region storage while " +
				"request/header slices remain borrowed"),
			"RequestParseReport records HeaderViewsBorrowed and HeaderValuesCopied",
			("AppendResponseWithReport records region response-buffer storage " +
				"and HeapAllocations=0 for caller-provided region buffers"),
		},
		Evidence: ("compiler/internal/httprt/request_" +
			"view.go::ParseRequestViewInRegion; compiler/internal/httprt/request_" +
			"view.go::AppendResponseWithReport; compiler/internal/httprt/request_" +
			"region.go::RequestRegionScope"),
		Boundary: ("HTTP evidence covers request-head/body views and response " +
			"building; it is not a full production web stack or router claim"),
		RegionFirst:         true,
		BorrowedViews:       true,
		CopyOnlyWhenNeeded:  true,
		HiddenHeapInHotPath: false,
	}
}

func regionStdlibPostgreSQLProtocolRow() RegionAwareStdlibEvidenceRow {
	return RegionAwareStdlibEvidenceRow{
		ID:     RegionStdlibPostgreSQLProtocol,
		Name:   "PostgreSQL protocol helpers",
		Status: RegionAwareStdlibImplementedNarrow,
		RequiredFacts: []string{
			"AppendBindFormat builds PostgreSQL binary/text bind frames into caller-owned buffers",
			"DecodeDataRowBorrowed returns borrowed cell slices and a RowDecodeReport",
			"RowDecodeReport records borrowed cells without promoting a production PostgreSQL stack",
		},
		Evidence: ("compiler/internal/pgrt/wire.go::AppendBindFormat; compiler/" +
			"internal/pgrt/wire.go::DecodeDataRowBorrowed; compiler/internal/pgrt/" +
			"row_decode_test.go::TestDecodeDataRowBorrowedDoesNotCopyCells"),
		Boundary: ("PostgreSQL evidence covers protocol helpers and borrowed row " +
			"decoding; production pooling/driver readiness is not claimed"),
		RegionFirst:         true,
		BorrowedViews:       true,
		CopyOnlyWhenNeeded:  true,
		HiddenHeapInHotPath: false,
	}
}

func regionStdlibBuffersRow() RegionAwareStdlibEvidenceRow {
	return RegionAwareStdlibEvidenceRow{
		ID:     RegionStdlibBuffers,
		Name:   "Region buffers",
		Status: RegionAwareStdlibImplementedNarrow,
		RequiredFacts: []string{
			"NewByteBuffer allocates buffer storage from Region when provided",
			"ByteBuffer.View returns borrowed BytesView slices",
			"StorageReport records StorageRegion, RegionID, HiddenHeap=false, BytesReserved, and BytesUsed",
		},
		Evidence: ("compiler/internal/stdlibrt/collections.go::NewByteBuffer; " +
			"compiler/internal/stdlibrt/collections_" +
			"test.go::TestByteBufferViewsPreserveRegionProvenance"),
		Boundary: ("ByteBuffer is a bounded region helper; dynamic resizing and " +
			"generic buffer abstractions remain future work"),
		RegionFirst:         true,
		BorrowedViews:       true,
		CopyOnlyWhenNeeded:  true,
		HiddenHeapInHotPath: false,
	}
}

func regionStdlibRingBuffersRow() RegionAwareStdlibEvidenceRow {
	return RegionAwareStdlibEvidenceRow{
		ID:     RegionStdlibRingBuffers,
		Name:   "Region ring buffers",
		Status: RegionAwareStdlibImplementedNarrow,
		RequiredFacts: []string{
			"NewRingBuffer allocates ring and snapshot storage from Region",
			"PeekView borrows contiguous readable windows and copies only a wrapped readable window",
			"wrapped views set BytesView.Copied and update CopyOperations/BytesCopied",
		},
		Evidence: ("compiler/internal/stdlibrt/collections.go::NewRingBuffer; " +
			"compiler/internal/stdlibrt/collections_" +
			"test.go::TestRegionAwareStdlibHelpersUseRegionReportsAndBorrowedViews"),
		Boundary: ("RingBuffer is bounded FIFO byte storage; multi-producer " +
			"concurrency and network-reactor integration remain outside P19.0"),
		RegionFirst:         true,
		BorrowedViews:       true,
		CopyOnlyWhenNeeded:  true,
		HiddenHeapInHotPath: false,
	}
}

func regionStdlibBorrowedViewsRow() RegionAwareStdlibEvidenceRow {
	return RegionAwareStdlibEvidenceRow{
		ID:     RegionStdlibBorrowedViews,
		Name:   "Borrowed views where possible",
		Status: RegionAwareStdlibEvidenceOnly,
		RequiredFacts: []string{
			"BytesView carries StorageBorrowed for input-owned JSON/HTTP views",
			"BytesView carries StorageRegion and RegionID for region-owned collection views",
			"borrowed views preserve provenance and do not imply escaped ownership",
		},
		Evidence: ("compiler/internal/stdlibrt/collections.go::BytesView; compiler/" +
			"internal/jsonrt/view.go::parseString; compiler/internal/httprt/request_" +
			"view.go::parseHeaderLineView"),
		Boundary: ("borrowed-view evidence is runtime helper metadata, not a new " +
			"lifetime type-system feature"),
		RegionFirst:         true,
		BorrowedViews:       true,
		CopyOnlyWhenNeeded:  true,
		HiddenHeapInHotPath: false,
	}
}

func regionStdlibCopyReportsRow() RegionAwareStdlibEvidenceRow {
	return RegionAwareStdlibEvidenceRow{
		ID:     RegionStdlibCopyReports,
		Name:   "Copy-only-when-needed reports",
		Status: RegionAwareStdlibEvidenceOnly,
		RequiredFacts: []string{
			"copy only when needed is reported by CopyOperations and BytesCopied in StorageReport",
			"JSON view parsing reports CopiedStrings only for escaped strings that must be decoded",
			"RingBuffer wrapped views report Copied because a contiguous borrowed view is impossible",
		},
		Evidence: ("compiler/internal/stdlibrt/collections.go::StorageReport; " +
			"compiler/internal/jsonrt/view.go::ParseViewReport; compiler/internal/" +
			"stdlibrt/collections.go::RingBuffer.PeekView"),
		Boundary: ("copy reports are evidence rows; they do not promise global zero-" +
			"copy behavior"),
		RegionFirst:         true,
		BorrowedViews:       true,
		CopyOnlyWhenNeeded:  true,
		HiddenHeapInHotPath: false,
	}
}

func regionStdlibNoHiddenHeapReportsRow() RegionAwareStdlibEvidenceRow {
	return RegionAwareStdlibEvidenceRow{
		ID:     RegionStdlibNoHiddenHeapReports,
		Name:   "No hidden heap in hot paths without report",
		Status: RegionAwareStdlibEvidenceOnly,
		RequiredFacts: []string{
			"region-backed StorageReport rows use HiddenHeap=false",
			"HTTP/JSON hot-path reports require HeapAllocations=0 when region buffers are supplied",
			"heap fallback remains report-visible through HiddenHeap=true or HeapAllocations>0",
		},
		Evidence: ("compiler/internal/stdlibrt/collections.go::StorageReport; " +
			"compiler/internal/jsonrt/view.go::ParseViewReport; compiler/internal/" +
			"httprt/request_view.go::ResponseBufferReport"),
		Boundary: ("this is report-visible helper evidence, not a proof that all Go " +
			"tests or all future stdlib code allocate zero heap"),
		RegionFirst:         true,
		BorrowedViews:       true,
		CopyOnlyWhenNeeded:  true,
		HiddenHeapInHotPath: false,
	}
}

func regionStdlibProductionBoundariesRow() RegionAwareStdlibEvidenceRow {
	return RegionAwareStdlibEvidenceRow{
		ID:     RegionStdlibProductionBoundaries,
		Name:   "Production boundary non-claims",
		Status: RegionAwareStdlibBoundaryDocumented,
		RequiredFacts: []string{
			"no full production web stack is claimed by P19.0",
			"no official TechEmpower result is claimed by P19.0",
			"no production PostgreSQL stack is claimed by P19.0",
		},
		Evidence: ("compiler/internal/stdlibrt/region_aware_" +
			"coverage.go::ValidateRegionAwareStdlibCoverage; docs/plans/2026-06-03/" +
			"backend-stdlib/2026-06-03-region-aware-stdlib-v1-design.md"),
		Boundary: ("P19.0 is runtime-helper evidence only; P19.2/P19.3 and later " +
			"gates must prove any production web/database promotion separately"),
		RegionFirst:         true,
		BorrowedViews:       true,
		CopyOnlyWhenNeeded:  true,
		HiddenHeapInHotPath: false,
	}
}

func containsRegionStdlibText(items []string, want string) bool {
	for _, item := range items {
		if strings.Contains(item, want) {
			return true
		}
	}
	return false
}
