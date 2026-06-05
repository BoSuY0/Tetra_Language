package pgrt

import (
	"strings"
	"testing"
)

func TestProductionPostgresCoverageDefinesP19DriverPoolSlice(t *testing.T) {
	report, err := ProductionPostgresCoverage()
	if err != nil {
		t.Fatalf("ProductionPostgresCoverage: %v", err)
	}
	if err := ValidateProductionPostgresCoverage(report); err != nil {
		t.Fatalf("ValidateProductionPostgresCoverage: %v", err)
	}
	if report.SchemaVersion != "tetra.stdlib.postgresql.production_driver.v1" {
		t.Fatalf("schema = %q", report.SchemaVersion)
	}
	if report.OfficialTechEmpowerResultClaimed || report.ProductionDatabaseBenchmarkClaimed || report.P20PerformanceMatrixClaimed || report.CPlusPlusRustParityClaimed || report.ExternalProductionDatabaseClaimed || report.FullSourceLevelDriverClaimed || report.RuntimeBehaviorChanged {
		t.Fatalf("coverage report contains forbidden claim flags: %#v", report)
	}

	rows := productionPostgresRows(t, report.Rows)
	expected := map[ProductionPostgresEvidenceID]ProductionPostgresEvidenceStatus{
		ProductionPostgresStartupSCRAM:            ProductionPostgresImplementedNarrow,
		ProductionPostgresPreparedStatements:      ProductionPostgresImplementedNarrow,
		ProductionPostgresBinaryProtocol:          ProductionPostgresImplementedNarrow,
		ProductionPostgresPoolingBackpressure:     ProductionPostgresImplementedNarrow,
		ProductionPostgresBorrowedRegionRowDecode: ProductionPostgresImplementedNarrow,
		ProductionPostgresEndpointWorkloads:       ProductionPostgresImplementedNarrow,
		ProductionPostgresSourceFirstBenchmark:    ProductionPostgresEvidenceOnly,
		ProductionPostgresLiveBenchmarkHonesty:    ProductionPostgresEvidenceOnly,
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

	requireProductionPostgresFacts(t, rows[ProductionPostgresStartupSCRAM], "readStartupReady", "AuthSASL", "TestClientCompletesSCRAMSHA256Authentication")
	requireProductionPostgresFacts(t, rows[ProductionPostgresPreparedStatements], "PreparedQueryFormat", "AppendParse", "TestClientPreparedQueryUsesExtendedProtocol")
	requireProductionPostgresFacts(t, rows[ProductionPostgresBinaryProtocol], "AppendBindFormat", "BinaryFormat", "TestAppendBindBinaryFormatsAndDecodeInt4")
	requireProductionPostgresFacts(t, rows[ProductionPostgresPoolingBackpressure], "NewPool", "ErrPoolExhausted", "TestPoolReusesReleasedConnectionAndCapsOpenConnections")
	requireProductionPostgresFacts(t, rows[ProductionPostgresBorrowedRegionRowDecode], "DecodeDataRowBorrowed", "RowStorageBorrowed", "TestDecodeDataRowBorrowedDoesNotCopyCells")
	requireProductionPostgresFacts(t, rows[ProductionPostgresEndpointWorkloads], "/db", "/queries", "/updates", "/fortunes", "TestServerFortunesEndpointFetchesSortsAndEscapesHTML")
	requireProductionPostgresFacts(t, rows[ProductionPostgresSourceFirstBenchmark], "p19.3_postgres_source_first", "DB single query", "DB fortunes")
	requireProductionPostgresFacts(t, rows[ProductionPostgresLiveBenchmarkHonesty], "validate-techempower-report", "techempower_scram_single_query_local_report.json", "techempower_scram_single_query_matrix_local_report.json", "techempower_scram_endpoint_matrix_local_report.json", "SCRAM-SHA-256", "/db", "/queries", "/updates", "/fortunes", "p19.3_postgres_source_first")
}

func TestProductionPostgresCoverageRejectsFakeClaims(t *testing.T) {
	report, err := ProductionPostgresCoverage()
	if err != nil {
		t.Fatalf("ProductionPostgresCoverage: %v", err)
	}
	report.OfficialTechEmpowerResultClaimed = true
	err = ValidateProductionPostgresCoverage(report)
	if err == nil || !strings.Contains(err.Error(), "official TechEmpower") {
		t.Fatalf("ValidateProductionPostgresCoverage accepted official TechEmpower claim: %v", err)
	}

	report, _ = ProductionPostgresCoverage()
	report.ProductionDatabaseBenchmarkClaimed = true
	err = ValidateProductionPostgresCoverage(report)
	if err == nil || !strings.Contains(err.Error(), "production database benchmark") {
		t.Fatalf("ValidateProductionPostgresCoverage accepted production database benchmark claim: %v", err)
	}

	report, _ = ProductionPostgresCoverage()
	report.P20PerformanceMatrixClaimed = true
	err = ValidateProductionPostgresCoverage(report)
	if err == nil || !strings.Contains(err.Error(), "P20 performance") {
		t.Fatalf("ValidateProductionPostgresCoverage accepted P20 performance claim: %v", err)
	}

	report, _ = ProductionPostgresCoverage()
	report.Rows[0].ClaimsRuntimeBehaviorChange = true
	err = ValidateProductionPostgresCoverage(report)
	if err == nil || !strings.Contains(err.Error(), "runtime behavior change") {
		t.Fatalf("ValidateProductionPostgresCoverage accepted row runtime change claim: %v", err)
	}
}

func productionPostgresRows(t *testing.T, rows []ProductionPostgresEvidenceRow) map[ProductionPostgresEvidenceID]ProductionPostgresEvidenceRow {
	t.Helper()
	out := map[ProductionPostgresEvidenceID]ProductionPostgresEvidenceRow{}
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

func requireProductionPostgresFacts(t *testing.T, row ProductionPostgresEvidenceRow, wants ...string) {
	t.Helper()
	text := strings.Join(row.RequiredFacts, "\n") + "\n" + strings.Join(row.MissingFacts, "\n") + "\n" + row.Evidence + "\n" + row.Boundary
	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Fatalf("row %q missing fact %q:\n%s", row.ID, want, text)
		}
	}
}
