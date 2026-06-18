package httprt

import (
	"strings"
	"testing"
)

func TestProductionHTTPJSONCoverageDefinesP19SourceFirstSlice(t *testing.T) {
	report, err := ProductionHTTPJSONCoverage()
	if err != nil {
		t.Fatalf("ProductionHTTPJSONCoverage: %v", err)
	}
	if err := ValidateProductionHTTPJSONCoverage(report); err != nil {
		t.Fatalf("ValidateProductionHTTPJSONCoverage: %v", err)
	}
	if report.SchemaVersion != "tetra.stdlib.http_json.production_stack.v1" {
		t.Fatalf("schema = %q", report.SchemaVersion)
	}
	if report.FullProductionWebStackClaimed || report.OfficialTechEmpowerResultClaimed ||
		report.ProductionPostgreSQLStackClaimed ||
		report.P20PerformanceMatrixClaimed ||
		report.CPlusPlusRustParityClaimed ||
		report.RuntimeBehaviorChanged {
		t.Fatalf("coverage report contains forbidden claim flags: %#v", report)
	}

	rows := productionHTTPJSONRows(t, report.Rows)
	expected := map[ProductionHTTPJSONEvidenceID]ProductionHTTPJSONEvidenceStatus{
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
	if len(rows) != len(expected) {
		t.Fatalf("row count = %d, want %d", len(rows), len(expected))
	}
	for id, status := range expected {
		row, ok := rows[id]
		if !ok {
			t.Fatalf("missing row %q", id)
		}
		if row.Status != status {
			t.Fatalf("row %q status = %q, want %q", id, row.Status, status)
		}
	}

	requireProductionHTTPJSONFacts(
		t,
		rows[ProductionHTTPJSONRequestHeadParser],
		"lib/core/io/http.tetra",
		"request_head_len_bytes_at",
		"ParseRequestView",
	)
	requireProductionHTTPJSONFacts(
		t,
		rows[ProductionHTTPJSONPipelinedRequestHeads],
		"pipelining",
		"consumed",
		"examples/core/platform/core_http_smoke.tetra",
	)
	requireProductionHTTPJSONFacts(
		t,
		rows[ProductionHTTPJSONHeadersBodyKeepAlive],
		"Content-Length",
		"Body",
		"KeepAlive",
	)
	requireProductionHTTPJSONFacts(
		t,
		rows[ProductionHTTPJSONNoHeapRequestView],
		"HeapAllocations=0",
		"HeaderViewsBorrowed",
		"TestParseRequestViewBorrowsHeadersWithoutAllocating",
	)
	requireProductionHTTPJSONFacts(
		t,
		rows[ProductionHTTPJSONJSONParseStringify],
		"lib/core/data/json.tetra",
		"ParseValueView",
		"AppendMessageObject",
	)
	requireProductionHTTPJSONFacts(
		t,
		rows[ProductionHTTPJSONResponseBuilder],
		"write_plaintext_response",
		"write_json_message_response",
		"AppendResponseWithReport",
	)
	requireProductionHTTPJSONFacts(
		t,
		rows[ProductionHTTPJSONDateCacheBoundary],
		"HTTPDateCache",
		"FormatWithReport",
		"TestHTTPDateCacheRefreshesOncePerSecond",
		"source-level lib.core.http cached-date helper remains out of scope",
	)
	requireProductionHTTPJSONFacts(
		t,
		rows[ProductionHTTPJSONWritevSendfileBoundary],
		"netrt.Writev",
		"netrt.Sendfile",
		"TestWritevWritesMultipleBuffersOnConnectedTCP",
		"TestSendfileCopiesFileBytesToConnectedTCPAndAdvancesOffset",
		"webrt.flush remains single-buffer",
	)
	requireProductionHTTPJSONFacts(
		t,
		rows[ProductionHTTPJSONSourceFirstBenchmark],
		"p19.2_http_json_source_first",
		"HTTP plaintext",
		"HTTP JSON",
	)
}

func TestProductionHTTPJSONCoverageRejectsFakeClaims(t *testing.T) {
	report, err := ProductionHTTPJSONCoverage()
	if err != nil {
		t.Fatalf("ProductionHTTPJSONCoverage: %v", err)
	}
	report.FullProductionWebStackClaimed = true
	err = ValidateProductionHTTPJSONCoverage(report)
	if err == nil || !strings.Contains(err.Error(), "full production web stack") {
		t.Fatalf("ValidateProductionHTTPJSONCoverage accepted full-production claim: %v", err)
	}

	report, _ = ProductionHTTPJSONCoverage()
	report.OfficialTechEmpowerResultClaimed = true
	err = ValidateProductionHTTPJSONCoverage(report)
	if err == nil || !strings.Contains(err.Error(), "official TechEmpower") {
		t.Fatalf("ValidateProductionHTTPJSONCoverage accepted official TechEmpower claim: %v", err)
	}

	report, _ = ProductionHTTPJSONCoverage()
	report.P20PerformanceMatrixClaimed = true
	err = ValidateProductionHTTPJSONCoverage(report)
	if err == nil || !strings.Contains(err.Error(), "P20 performance") {
		t.Fatalf("ValidateProductionHTTPJSONCoverage accepted P20 performance claim: %v", err)
	}

	report, _ = ProductionHTTPJSONCoverage()
	report.Rows[0].ClaimsCPlusPlusRustParity = true
	err = ValidateProductionHTTPJSONCoverage(report)
	if err == nil || !strings.Contains(err.Error(), "C++/Rust parity") {
		t.Fatalf("ValidateProductionHTTPJSONCoverage accepted row parity claim: %v", err)
	}
}

func productionHTTPJSONRows(
	t *testing.T,
	rows []ProductionHTTPJSONEvidenceRow,
) map[ProductionHTTPJSONEvidenceID]ProductionHTTPJSONEvidenceRow {
	t.Helper()
	out := map[ProductionHTTPJSONEvidenceID]ProductionHTTPJSONEvidenceRow{}
	for _, row := range rows {
		if row.ID == "" {
			t.Fatalf("row missing id: %#v", row)
		}
		if _, exists := out[row.ID]; exists {
			t.Fatalf("duplicate row %q", row.ID)
		}
		out[row.ID] = row
	}
	return out
}

func requireProductionHTTPJSONFacts(
	t *testing.T,
	row ProductionHTTPJSONEvidenceRow,
	wants ...string,
) {
	t.Helper()
	text := strings.Join(
		row.RequiredFacts,
		"\n",
	) + "\n" + row.Evidence + "\n" + row.Boundary + "\n" + strings.Join(
		row.MissingFacts,
		"\n",
	)
	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Fatalf("row %q missing fact %q:\n%s", row.ID, want, text)
		}
	}
}
