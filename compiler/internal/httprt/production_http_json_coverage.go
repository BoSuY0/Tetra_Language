package httprt

import (
	"fmt"
	"strings"
)

type ProductionHTTPJSONEvidenceID string

const (
	ProductionHTTPJSONRequestHeadParser      ProductionHTTPJSONEvidenceID = "http1_request_head_parser"
	ProductionHTTPJSONPipelinedRequestHeads  ProductionHTTPJSONEvidenceID = "pipelined_request_heads"
	ProductionHTTPJSONHeadersBodyKeepAlive   ProductionHTTPJSONEvidenceID = "headers_body_keepalive"
	ProductionHTTPJSONNoHeapRequestView      ProductionHTTPJSONEvidenceID = "no_heap_request_view"
	ProductionHTTPJSONJSONParseStringify     ProductionHTTPJSONEvidenceID = "json_parse_stringify"
	ProductionHTTPJSONResponseBuilder        ProductionHTTPJSONEvidenceID = "response_builder"
	ProductionHTTPJSONDateCacheBoundary      ProductionHTTPJSONEvidenceID = "date_cache_boundary"
	ProductionHTTPJSONWritevSendfileBoundary ProductionHTTPJSONEvidenceID = "writev_sendfile_boundary"
	ProductionHTTPJSONSourceFirstBenchmark   ProductionHTTPJSONEvidenceID = "source_first_benchmark_gate"
)

type ProductionHTTPJSONEvidenceStatus string

const (
	ProductionHTTPJSONImplementedNarrow  ProductionHTTPJSONEvidenceStatus = "implemented_narrow"
	ProductionHTTPJSONEvidenceOnly       ProductionHTTPJSONEvidenceStatus = "evidence_only"
	ProductionHTTPJSONBoundaryDocumented ProductionHTTPJSONEvidenceStatus = "boundary_documented"
)

type ProductionHTTPJSONCoverageReport struct {
	SchemaVersion                    string                          `json:"schema_version"`
	Rows                             []ProductionHTTPJSONEvidenceRow `json:"rows"`
	NonClaims                        []string                        `json:"non_claims"`
	FullProductionWebStackClaimed    bool                            `json:"full_production_web_stack_claimed"`
	OfficialTechEmpowerResultClaimed bool                            `json:"official_techempower_result_claimed"`
	ProductionPostgreSQLStackClaimed bool                            `json:"production_postgresql_stack_claimed"`
	P20PerformanceMatrixClaimed      bool                            `json:"p20_performance_matrix_claimed"`
	CPlusPlusRustParityClaimed       bool                            `json:"c_plus_plus_rust_parity_claimed"`
	RuntimeBehaviorChanged           bool                            `json:"runtime_behavior_changed"`
}

type ProductionHTTPJSONEvidenceRow struct {
	ID                              ProductionHTTPJSONEvidenceID     `json:"id"`
	Name                            string                           `json:"name"`
	Status                          ProductionHTTPJSONEvidenceStatus `json:"status"`
	RequiredFacts                   []string                         `json:"required_facts,omitempty"`
	MissingFacts                    []string                         `json:"missing_facts,omitempty"`
	Evidence                        string                           `json:"evidence"`
	Boundary                        string                           `json:"boundary"`
	SourceFirst                     bool                             `json:"source_first"`
	BorrowedViews                   bool                             `json:"borrowed_views,omitempty"`
	NoHeapHotPath                   bool                             `json:"no_heap_hot_path,omitempty"`
	ClaimsFullProductionWebStack    bool                             `json:"claims_full_production_web_stack,omitempty"`
	ClaimsOfficialTechEmpowerResult bool                             `json:"claims_official_techempower_result,omitempty"`
	ClaimsProductionPostgreSQLStack bool                             `json:"claims_production_postgresql_stack,omitempty"`
	ClaimsP20PerformanceMatrix      bool                             `json:"claims_p20_performance_matrix,omitempty"`
	ClaimsCPlusPlusRustParity       bool                             `json:"claims_c_plus_plus_rust_parity,omitempty"`
	ClaimsRuntimeBehaviorChange     bool                             `json:"claims_runtime_behavior_change,omitempty"`
}

func ProductionHTTPJSONCoverage() (ProductionHTTPJSONCoverageReport, error) {
	return ProductionHTTPJSONCoverageReport{
		SchemaVersion: "tetra.stdlib.http_json.production_stack.v1",
		Rows: []ProductionHTTPJSONEvidenceRow{
			productionHTTPJSONRequestHeadParserRow(),
			productionHTTPJSONPipelinedRequestHeadsRow(),
			productionHTTPJSONHeadersBodyKeepAliveRow(),
			productionHTTPJSONNoHeapRequestViewRow(),
			productionHTTPJSONJSONParseStringifyRow(),
			productionHTTPJSONResponseBuilderRow(),
			productionHTTPJSONDateCacheBoundaryRow(),
			productionHTTPJSONWritevSendfileBoundaryRow(),
			productionHTTPJSONSourceFirstBenchmarkRow(),
		},
		NonClaims: []string{
			"full production web stack is not claimed",
			"official TechEmpower result is not claimed",
			"production PostgreSQL stack is not claimed",
			"P20 performance matrix is not claimed",
			"C++/Rust parity is not claimed",
			"externally visible runtime behavior is unchanged by the per-second Date cache helper",
		},
		FullProductionWebStackClaimed:    false,
		OfficialTechEmpowerResultClaimed: false,
		ProductionPostgreSQLStackClaimed: false,
		P20PerformanceMatrixClaimed:      false,
		CPlusPlusRustParityClaimed:       false,
		RuntimeBehaviorChanged:           false,
	}, nil
}

func ValidateProductionHTTPJSONCoverage(report ProductionHTTPJSONCoverageReport) error {
	if report.SchemaVersion != "tetra.stdlib.http_json.production_stack.v1" {
		return fmt.Errorf("production HTTP/JSON coverage: schema = %q", report.SchemaVersion)
	}
	if report.FullProductionWebStackClaimed {
		return fmt.Errorf("production HTTP/JSON coverage: full production web stack claim is forbidden for P19.2")
	}
	if report.OfficialTechEmpowerResultClaimed {
		return fmt.Errorf("production HTTP/JSON coverage: official TechEmpower claim is forbidden for P19.2")
	}
	if report.ProductionPostgreSQLStackClaimed {
		return fmt.Errorf("production HTTP/JSON coverage: production PostgreSQL stack claim is forbidden for P19.2")
	}
	if report.P20PerformanceMatrixClaimed {
		return fmt.Errorf("production HTTP/JSON coverage: P20 performance matrix claim is forbidden for P19.2")
	}
	if report.CPlusPlusRustParityClaimed {
		return fmt.Errorf("production HTTP/JSON coverage: C++/Rust parity claim is forbidden for P19.2")
	}
	if report.RuntimeBehaviorChanged {
		return fmt.Errorf("production HTTP/JSON coverage: runtime behavior change claim is forbidden for report-only P19.2 coverage")
	}
	for _, want := range []string{
		"full production web stack is not claimed",
		"official TechEmpower result is not claimed",
		"production PostgreSQL stack is not claimed",
		"P20 performance matrix is not claimed",
		"C++/Rust parity is not claimed",
		"runtime behavior is unchanged",
	} {
		if !containsProductionHTTPJSONText(report.NonClaims, want) {
			return fmt.Errorf("production HTTP/JSON coverage: missing non-claim %q", want)
		}
	}

	expectedStatus := map[ProductionHTTPJSONEvidenceID]ProductionHTTPJSONEvidenceStatus{
		ProductionHTTPJSONRequestHeadParser:      ProductionHTTPJSONImplementedNarrow,
		ProductionHTTPJSONPipelinedRequestHeads:  ProductionHTTPJSONImplementedNarrow,
		ProductionHTTPJSONHeadersBodyKeepAlive:   ProductionHTTPJSONImplementedNarrow,
		ProductionHTTPJSONNoHeapRequestView:      ProductionHTTPJSONEvidenceOnly,
		ProductionHTTPJSONJSONParseStringify:     ProductionHTTPJSONImplementedNarrow,
		ProductionHTTPJSONResponseBuilder:        ProductionHTTPJSONImplementedNarrow,
		ProductionHTTPJSONDateCacheBoundary:      ProductionHTTPJSONImplementedNarrow,
		ProductionHTTPJSONWritevSendfileBoundary: ProductionHTTPJSONImplementedNarrow,
		ProductionHTTPJSONSourceFirstBenchmark:   ProductionHTTPJSONEvidenceOnly,
	}
	if len(report.Rows) != len(expectedStatus) {
		return fmt.Errorf("production HTTP/JSON coverage: row count = %d, want %d", len(report.Rows), len(expectedStatus))
	}
	rows := map[ProductionHTTPJSONEvidenceID]ProductionHTTPJSONEvidenceRow{}
	for _, row := range report.Rows {
		if row.ID == "" {
			return fmt.Errorf("production HTTP/JSON coverage: row missing id")
		}
		wantStatus, ok := expectedStatus[row.ID]
		if !ok {
			return fmt.Errorf("production HTTP/JSON coverage: unexpected row %q", row.ID)
		}
		if _, exists := rows[row.ID]; exists {
			return fmt.Errorf("production HTTP/JSON coverage: duplicate row %q", row.ID)
		}
		rows[row.ID] = row
		if row.Status != wantStatus {
			return fmt.Errorf("production HTTP/JSON coverage: row %q status = %q, want %q", row.ID, row.Status, wantStatus)
		}
		if strings.TrimSpace(row.Name) == "" || strings.TrimSpace(row.Evidence) == "" || strings.TrimSpace(row.Boundary) == "" {
			return fmt.Errorf("production HTTP/JSON coverage: row %q missing evidence or boundary", row.ID)
		}
		if len(row.RequiredFacts) == 0 {
			return fmt.Errorf("production HTTP/JSON coverage: row %q missing required facts", row.ID)
		}
		if row.ClaimsFullProductionWebStack {
			return fmt.Errorf("production HTTP/JSON coverage: row %q claims full production web stack", row.ID)
		}
		if row.ClaimsOfficialTechEmpowerResult {
			return fmt.Errorf("production HTTP/JSON coverage: row %q claims official TechEmpower result", row.ID)
		}
		if row.ClaimsProductionPostgreSQLStack {
			return fmt.Errorf("production HTTP/JSON coverage: row %q claims production PostgreSQL stack", row.ID)
		}
		if row.ClaimsP20PerformanceMatrix {
			return fmt.Errorf("production HTTP/JSON coverage: row %q claims P20 performance matrix", row.ID)
		}
		if row.ClaimsCPlusPlusRustParity {
			return fmt.Errorf("production HTTP/JSON coverage: row %q claims C++/Rust parity", row.ID)
		}
		if row.ClaimsRuntimeBehaviorChange {
			return fmt.Errorf("production HTTP/JSON coverage: row %q claims runtime behavior change", row.ID)
		}
	}
	for id := range expectedStatus {
		if _, ok := rows[id]; !ok {
			return fmt.Errorf("production HTTP/JSON coverage: missing row %q", id)
		}
	}

	checks := []struct {
		id    ProductionHTTPJSONEvidenceID
		wants []string
	}{
		{ProductionHTTPJSONRequestHeadParser, []string{"lib/core/http.tetra", "request_head_len_bytes_at", "ParseRequestView"}},
		{ProductionHTTPJSONPipelinedRequestHeads, []string{"pipelining", "consumed", "examples/core_http_smoke.tetra"}},
		{ProductionHTTPJSONHeadersBodyKeepAlive, []string{"Content-Length", "Body", "KeepAlive"}},
		{ProductionHTTPJSONNoHeapRequestView, []string{"HeapAllocations=0", "HeaderViewsBorrowed", "TestParseRequestViewBorrowsHeadersWithoutAllocating"}},
		{ProductionHTTPJSONJSONParseStringify, []string{"lib/core/json.tetra", "ParseValueView", "AppendMessageObject"}},
		{ProductionHTTPJSONResponseBuilder, []string{"write_plaintext_response", "write_json_message_response", "AppendResponseWithReport"}},
		{ProductionHTTPJSONDateCacheBoundary, []string{"HTTPDateCache", "FormatWithReport", "TestHTTPDateCacheRefreshesOncePerSecond", "source-level lib.core.http cached-date helper remains out of scope"}},
		{ProductionHTTPJSONWritevSendfileBoundary, []string{"netrt.Writev", "netrt.Sendfile", "TestWritevWritesMultipleBuffersOnConnectedTCP", "TestSendfileCopiesFileBytesToConnectedTCPAndAdvancesOffset", "webrt.flush remains single-buffer"}},
		{ProductionHTTPJSONSourceFirstBenchmark, []string{"p19.2_http_json_source_first", "HTTP plaintext", "HTTP JSON"}},
	}
	for _, check := range checks {
		row := rows[check.id]
		for _, want := range check.wants {
			if !containsProductionHTTPJSONRowText(row, want) {
				return fmt.Errorf("production HTTP/JSON coverage: row %q missing fact %q", row.ID, want)
			}
		}
	}
	return nil
}

func productionHTTPJSONRequestHeadParserRow() ProductionHTTPJSONEvidenceRow {
	return ProductionHTTPJSONEvidenceRow{
		ID:     ProductionHTTPJSONRequestHeadParser,
		Name:   "HTTP/1.1 request-head parser",
		Status: ProductionHTTPJSONImplementedNarrow,
		RequiredFacts: []string{
			"lib/core/http.tetra::request_head_len_bytes_at locates CRLFCRLF request head boundaries from Tetra source",
			"compiler/internal/httprt/request_view.go::ParseRequestView parses method, request target, version, and headers without string conversion",
			"compiler/internal/httprt/http1.go::ParseRequest remains the current string-copying server parser and is not the zero-heap claim path",
		},
		Evidence:    "lib/core/http.tetra::request_head_len_bytes_at; compiler/internal/httprt/request_view.go::ParseRequestView; compiler/internal/httprt/http1_test.go::TestParseRequestHandlesPartialAndPipelinedRequests",
		Boundary:    "narrow HTTP/1.1 request-head parsing only; no HTTP/2, TLS, chunked transfer decoding, or complete production proxy behavior is claimed",
		SourceFirst: true,
	}
}

func productionHTTPJSONPipelinedRequestHeadsRow() ProductionHTTPJSONEvidenceRow {
	return ProductionHTTPJSONEvidenceRow{
		ID:     ProductionHTTPJSONPipelinedRequestHeads,
		Name:   "Pipelined request-head slicing",
		Status: ProductionHTTPJSONImplementedNarrow,
		RequiredFacts: []string{
			"examples/core_http_smoke.tetra proves source-level pipelining by using request_head_len_bytes_at to start the second request",
			"ParseRequestView and ParseRequest report consumed bytes so callers can continue parsing a pipelined buffer",
			"compiler/internal/webrt/server_test.go covers plaintext/json pipelining through the current local server path",
		},
		Evidence:    "examples/core_http_smoke.tetra; compiler/internal/httprt/http1_test.go::TestParseRequestHandlesPartialAndPipelinedRequests; compiler/internal/webrt/server_test.go::TestServerPlaintextKeepAliveAndPipelining; compiler/internal/webrt/server_test.go::TestServerJSONEndpointKeepAliveAndPipelining",
		Boundary:    "pipelining evidence is local HTTP/1.1 request sequencing; it is not an external load-test throughput or P20 performance claim",
		SourceFirst: true,
	}
}

func productionHTTPJSONHeadersBodyKeepAliveRow() ProductionHTTPJSONEvidenceRow {
	return ProductionHTTPJSONEvidenceRow{
		ID:     ProductionHTTPJSONHeadersBodyKeepAlive,
		Name:   "Headers, body, and keep-alive metadata",
		Status: ProductionHTTPJSONImplementedNarrow,
		RequiredFacts: []string{
			"ParseRequestView records Content-Length and exposes Body as a borrowed request slice",
			"ParseRequestView rejects unsupported Transfer-Encoding values and oversized Body inputs",
			"KeepAlive follows HTTP/1.1 close and HTTP/1.0 keep-alive Connection header rules",
		},
		Evidence:    "compiler/internal/httprt/request_view.go::applyHeaderMetadataView; compiler/internal/httprt/request_view_test.go::TestRequestRegionScopeInjectsRegionForHTTPJSONAndResetsAfterWrite; compiler/internal/httprt/http1_test.go::TestParseRequestHandlesBodyMetadataAndBoundaries",
		Boundary:    "header/body coverage is for non-chunked HTTP/1.1 request bodies under configured limits",
		SourceFirst: false,
	}
}

func productionHTTPJSONNoHeapRequestViewRow() ProductionHTTPJSONEvidenceRow {
	return ProductionHTTPJSONEvidenceRow{
		ID:     ProductionHTTPJSONNoHeapRequestView,
		Name:   "No-heap request-view evidence",
		Status: ProductionHTTPJSONEvidenceOnly,
		RequiredFacts: []string{
			"RequestParseReport records HeapAllocations=0 for the borrowed request-head path",
			"RequestParseReport records HeaderViewsBorrowed and HeaderValuesCopied so copy behavior is auditable",
			"TestParseRequestViewBorrowsHeadersWithoutAllocating uses testing.AllocsPerRun to prove the hot request-head parse path",
		},
		Evidence:      "compiler/internal/httprt/request_view.go::RequestParseReport; compiler/internal/httprt/request_view_test.go::TestParseRequestViewBorrowsHeadersWithoutAllocating",
		Boundary:      "zero-heap evidence is limited to request-view/request-region parsing; the current server connection buffers and string parser are outside this claim",
		SourceFirst:   false,
		BorrowedViews: true,
		NoHeapHotPath: true,
	}
}

func productionHTTPJSONJSONParseStringifyRow() ProductionHTTPJSONEvidenceRow {
	return ProductionHTTPJSONEvidenceRow{
		ID:     ProductionHTTPJSONJSONParseStringify,
		Name:   "JSON parse/stringify",
		Status: ProductionHTTPJSONImplementedNarrow,
		RequiredFacts: []string{
			"lib/core/json.tetra writes deterministic JSON message objects and escaped strings from Tetra source",
			"compiler/internal/jsonrt/view.go::ParseValueView parses borrowed/region JSON values and reports copies only when escaped strings require decoding",
			"compiler/internal/jsonrt/json.go::AppendMessageObject serializes the TechEmpower JSON endpoint payload",
		},
		Evidence:    "lib/core/json.tetra; examples/core_json_smoke.tetra; compiler/internal/jsonrt/view_test.go::TestParseValueViewBorrowsUnescapedStringsWithoutHeap; compiler/internal/jsonrt/json_test.go::TestAppendMessageObjectWritesTechEmpowerPayload",
		Boundary:    "JSON support is byte-oriented and ASCII escape focused; this is not a full public JSON DOM/API stability claim",
		SourceFirst: true,
	}
}

func productionHTTPJSONResponseBuilderRow() ProductionHTTPJSONEvidenceRow {
	return ProductionHTTPJSONEvidenceRow{
		ID:     ProductionHTTPJSONResponseBuilder,
		Name:   "HTTP response builder",
		Status: ProductionHTTPJSONImplementedNarrow,
		RequiredFacts: []string{
			"lib/core/http.tetra::write_plaintext_response writes the source-level plaintext response",
			"lib/core/http.tetra::write_json_message_response writes the source-level JSON response using lib.core.json",
			"compiler/internal/httprt/request_view.go::AppendResponseWithReport records response buffer storage and heap allocation facts",
		},
		Evidence:    "lib/core/http.tetra::write_plaintext_response; lib/core/http.tetra::write_json_message_response; compiler/internal/httprt/request_view_test.go::TestAppendResponseWithReportRecordsBufferStorage; examples/core_http_smoke.tetra",
		Boundary:    "response building covers deterministic headers/body serialization; socket write scheduling and throughput claims remain separate",
		SourceFirst: true,
	}
}

func productionHTTPJSONDateCacheBoundaryRow() ProductionHTTPJSONEvidenceRow {
	return ProductionHTTPJSONEvidenceRow{
		ID:     ProductionHTTPJSONDateCacheBoundary,
		Name:   "Per-second Date/cache helper",
		Status: ProductionHTTPJSONImplementedNarrow,
		RequiredFacts: []string{
			"compiler/internal/webrt/date_cache.go::HTTPDateCache caches formatted IMF-fixdate values by UTC Unix second",
			"compiler/internal/webrt/date_cache.go::FormatWithReport reports refresh/reuse facts for P19.2 evidence",
			"compiler/internal/webrt/server.go::Server.date uses HTTPDateCache when Config.DateFunc is nil",
			"compiler/internal/webrt/server.go::Config.NowFunc gives deterministic server-date cache tests without overriding Date",
			"compiler/internal/webrt/date_cache_test.go::TestHTTPDateCacheRefreshesOncePerSecond proves same-second reuse and boundary refresh",
			"compiler/internal/webrt/date_cache_test.go::TestServerDateFuncOverrideBypassesCache proves DateFunc override priority",
		},
		MissingFacts: []string{
			"source-level lib.core.http cached-date helper remains out of scope",
			"cross-worker/global Date cache is not implemented",
		},
		Evidence:    "compiler/internal/webrt/date_cache.go; compiler/internal/webrt/date_cache_test.go::TestHTTPDateCacheRefreshesOncePerSecond; compiler/internal/webrt/date_cache_test.go::TestServerDateUsesPerSecondCacheWhenDateFuncAbsent; compiler/internal/webrt/server_test.go::TestServerPlaintextKeepAliveAndPipelining",
		Boundary:    "Date caching is implemented only as an internal per-server UTC-second cache; no source-level cached-date API, cross-worker cache, performance claim, or full production web-stack promotion is made",
		SourceFirst: false,
	}
}

func productionHTTPJSONWritevSendfileBoundaryRow() ProductionHTTPJSONEvidenceRow {
	return ProductionHTTPJSONEvidenceRow{
		ID:     ProductionHTTPJSONWritevSendfileBoundary,
		Name:   "Linux writev/sendfile helpers",
		Status: ProductionHTTPJSONImplementedNarrow,
		RequiredFacts: []string{
			"compiler/internal/netrt/netrt_linux.go::Writev calls Linux SYS_WRITEV over non-empty byte chunks",
			"compiler/internal/netrt/netrt_linux.go::Sendfile delegates to Linux syscall.Sendfile",
			"compiler/internal/netrt/netrt_unsupported.go::Writev returns ErrUnsupported on non-Linux targets",
			"compiler/internal/netrt/netrt_unsupported.go::Sendfile returns ErrUnsupported on non-Linux targets",
			"compiler/internal/netrt/netrt_linux_test.go::TestWritevWritesMultipleBuffersOnConnectedTCP proves netrt.Writev on a connected TCP fd",
			"compiler/internal/netrt/netrt_linux_test.go::TestSendfileCopiesFileBytesToConnectedTCPAndAdvancesOffset proves netrt.Sendfile on a file-to-TCP fd and offset advancement",
			"compiler/internal/webrt/server.go::flush / webrt.flush remains single-buffer netrt.Write in this helper slice",
		},
		MissingFacts: []string{
			"HTTP static-file response path that proves sendfile semantics",
			"webrt.flush scatter/gather integration",
			"non-Linux writev/sendfile parity",
		},
		Evidence:    "compiler/internal/netrt/netrt_linux.go::Writev; compiler/internal/netrt/netrt_linux.go::Sendfile; compiler/internal/netrt/netrt_linux_test.go::TestWritevWritesMultipleBuffersOnConnectedTCP; compiler/internal/netrt/netrt_linux_test.go::TestSendfileCopiesFileBytesToConnectedTCPAndAdvancesOffset; compiler/internal/webrt/server.go::flush",
		Boundary:    "writev/sendfile support is limited to Linux netrt helpers; no HTTP static-file response path, webrt scatter/gather integration, zero-copy production file-serving, performance claim, or cross-platform parity is made",
		SourceFirst: false,
	}
}

func productionHTTPJSONSourceFirstBenchmarkRow() ProductionHTTPJSONEvidenceRow {
	return ProductionHTTPJSONEvidenceRow{
		ID:     ProductionHTTPJSONSourceFirstBenchmark,
		Name:   "Source-first HTTP/JSON benchmark gate",
		Status: ProductionHTTPJSONEvidenceOnly,
		RequiredFacts: []string{
			"tools/cmd/truth-bench-harness supports p19.2_http_json_source_first scope",
			"p19.2_http_json_source_first requires Tetra-source HTTP plaintext and HTTP JSON rows",
			"HTTP plaintext and HTTP JSON benchmark rows must list proof, allocation, bounds, and P19.2 evidence artifacts",
		},
		Evidence:    "tools/cmd/truth-bench-harness/main.go::policyForBenchmarkScope; tools/cmd/truth-bench-harness/main_test.go::TestP19HTTPJSONSourceFirstScopeRequiresTetraOnlyHTTPAndJSONRows; reports/production-http-json-v1/benchmarks/http-json-source-first-report.json",
		Boundary:    "source-first benchmark gate records local dry-run/source evidence only; it is not an official TechEmpower result, P20 performance matrix, or C++/Rust parity claim",
		SourceFirst: true,
	}
}

func containsProductionHTTPJSONText(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}

func containsProductionHTTPJSONRowText(row ProductionHTTPJSONEvidenceRow, want string) bool {
	text := strings.Join(row.RequiredFacts, "\n") + "\n" + strings.Join(row.MissingFacts, "\n") + "\n" + row.Evidence + "\n" + row.Boundary
	return strings.Contains(text, want)
}
