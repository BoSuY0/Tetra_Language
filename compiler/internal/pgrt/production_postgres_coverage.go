package pgrt

import (
	"fmt"
	"strings"
)

type ProductionPostgresEvidenceID string

const (
	ProductionPostgresStartupSCRAM            ProductionPostgresEvidenceID = "startup_scram"
	ProductionPostgresPreparedStatements      ProductionPostgresEvidenceID = "prepared_statements"
	ProductionPostgresBinaryProtocol          ProductionPostgresEvidenceID = "binary_protocol"
	ProductionPostgresPoolingBackpressure     ProductionPostgresEvidenceID = "pooling_backpressure"
	ProductionPostgresBorrowedRegionRowDecode ProductionPostgresEvidenceID = "borrowed_region_row_decode"
	ProductionPostgresEndpointWorkloads       ProductionPostgresEvidenceID = "db_queries_updates_fortunes"
	ProductionPostgresSourceFirstBenchmark    ProductionPostgresEvidenceID = "source_first_db_benchmark_gate"
	ProductionPostgresLiveBenchmarkHonesty    ProductionPostgresEvidenceID = "live_measured_db_benchmark_honesty"
)

type ProductionPostgresEvidenceStatus string

const (
	ProductionPostgresImplementedNarrow  ProductionPostgresEvidenceStatus = "implemented_narrow"
	ProductionPostgresEvidenceOnly       ProductionPostgresEvidenceStatus = "evidence_only"
	ProductionPostgresBoundaryDocumented ProductionPostgresEvidenceStatus = "boundary_documented"
)

type ProductionPostgresCoverageReport struct {
	SchemaVersion                      string                          `json:"schema_version"`
	Rows                               []ProductionPostgresEvidenceRow `json:"rows"`
	NonClaims                          []string                        `json:"non_claims"`
	OfficialTechEmpowerResultClaimed   bool                            `json:"official_techempower_result_claimed"`
	ProductionDatabaseBenchmarkClaimed bool                            `json:"production_database_benchmark_claimed"`
	P20PerformanceMatrixClaimed        bool                            `json:"p20_performance_matrix_claimed"`
	CPlusPlusRustParityClaimed         bool                            `json:"c_plus_plus_rust_parity_claimed"`
	ExternalProductionDatabaseClaimed  bool                            `json:"external_production_database_claimed"`
	FullSourceLevelDriverClaimed       bool                            `json:"full_source_level_driver_claimed"`
	RuntimeBehaviorChanged             bool                            `json:"runtime_behavior_changed"`
}

type ProductionPostgresEvidenceRow struct {
	ID                                ProductionPostgresEvidenceID     `json:"id"`
	Name                              string                           `json:"name"`
	Status                            ProductionPostgresEvidenceStatus `json:"status"`
	RequiredFacts                     []string                         `json:"required_facts,omitempty"`
	MissingFacts                      []string                         `json:"missing_facts,omitempty"`
	Evidence                          string                           `json:"evidence"`
	Boundary                          string                           `json:"boundary"`
	SourceFirst                       bool                             `json:"source_first"`
	BorrowedViews                     bool                             `json:"borrowed_views,omitempty"`
	Backpressure                      bool                             `json:"backpressure,omitempty"`
	ClaimsOfficialTechEmpowerResult   bool                             `json:"claims_official_techempower_result,omitempty"`
	ClaimsProductionDatabaseBenchmark bool                             `json:"claims_production_database_benchmark,omitempty"`
	ClaimsP20PerformanceMatrix        bool                             `json:"claims_p20_performance_matrix,omitempty"`
	ClaimsCPlusPlusRustParity         bool                             `json:"claims_c_plus_plus_rust_parity,omitempty"`
	ClaimsExternalProductionDatabase  bool                             `json:"claims_external_production_database,omitempty"`
	ClaimsFullSourceLevelDriver       bool                             `json:"claims_full_source_level_driver,omitempty"`
	ClaimsRuntimeBehaviorChange       bool                             `json:"claims_runtime_behavior_change,omitempty"`
}

func ProductionPostgresCoverage() (ProductionPostgresCoverageReport, error) {
	return ProductionPostgresCoverageReport{
		SchemaVersion: "tetra.stdlib.postgresql.production_driver.v1",
		Rows: []ProductionPostgresEvidenceRow{
			productionPostgresStartupSCRAMRow(),
			productionPostgresPreparedStatementsRow(),
			productionPostgresBinaryProtocolRow(),
			productionPostgresPoolingBackpressureRow(),
			productionPostgresBorrowedRegionRowDecodeRow(),
			productionPostgresEndpointWorkloadsRow(),
			productionPostgresSourceFirstBenchmarkRow(),
			productionPostgresLiveBenchmarkHonestyRow(),
		},
		NonClaims: []string{
			"official TechEmpower result is not claimed",
			"production database benchmark is not claimed",
			"P20 performance matrix is not claimed",
			"C++/Rust parity is not claimed",
			"external production database deployment is not claimed",
			"full source-level PostgreSQL driver API is not claimed",
			"runtime behavior is unchanged",
		},
		OfficialTechEmpowerResultClaimed:   false,
		ProductionDatabaseBenchmarkClaimed: false,
		P20PerformanceMatrixClaimed:        false,
		CPlusPlusRustParityClaimed:         false,
		ExternalProductionDatabaseClaimed:  false,
		FullSourceLevelDriverClaimed:       false,
		RuntimeBehaviorChanged:             false,
	}, nil
}

func ValidateProductionPostgresCoverage(report ProductionPostgresCoverageReport) error {
	if report.SchemaVersion != "tetra.stdlib.postgresql.production_driver.v1" {
		return fmt.Errorf("production PostgreSQL coverage: schema = %q", report.SchemaVersion)
	}
	if report.OfficialTechEmpowerResultClaimed {
		return fmt.Errorf("production PostgreSQL coverage: official TechEmpower claim is forbidden for P19.3")
	}
	if report.ProductionDatabaseBenchmarkClaimed {
		return fmt.Errorf("production PostgreSQL coverage: production database benchmark claim is forbidden for P19.3")
	}
	if report.P20PerformanceMatrixClaimed {
		return fmt.Errorf("production PostgreSQL coverage: P20 performance matrix claim is forbidden for P19.3")
	}
	if report.CPlusPlusRustParityClaimed {
		return fmt.Errorf("production PostgreSQL coverage: C++/Rust parity claim is forbidden for P19.3")
	}
	if report.ExternalProductionDatabaseClaimed {
		return fmt.Errorf("production PostgreSQL coverage: external production database claim is forbidden for P19.3")
	}
	if report.FullSourceLevelDriverClaimed {
		return fmt.Errorf("production PostgreSQL coverage: full source-level driver claim is forbidden for P19.3")
	}
	if report.RuntimeBehaviorChanged {
		return fmt.Errorf("production PostgreSQL coverage: runtime behavior change claim is forbidden for report-only P19.3 coverage")
	}
	for _, want := range []string{
		"official TechEmpower result is not claimed",
		"production database benchmark is not claimed",
		"P20 performance matrix is not claimed",
		"C++/Rust parity is not claimed",
		"external production database deployment is not claimed",
		"full source-level PostgreSQL driver API is not claimed",
		"runtime behavior is unchanged",
	} {
		if !containsProductionPostgresText(report.NonClaims, want) {
			return fmt.Errorf("production PostgreSQL coverage: missing non-claim %q", want)
		}
	}

	expectedStatus := map[ProductionPostgresEvidenceID]ProductionPostgresEvidenceStatus{
		ProductionPostgresStartupSCRAM:            ProductionPostgresImplementedNarrow,
		ProductionPostgresPreparedStatements:      ProductionPostgresImplementedNarrow,
		ProductionPostgresBinaryProtocol:          ProductionPostgresImplementedNarrow,
		ProductionPostgresPoolingBackpressure:     ProductionPostgresImplementedNarrow,
		ProductionPostgresBorrowedRegionRowDecode: ProductionPostgresImplementedNarrow,
		ProductionPostgresEndpointWorkloads:       ProductionPostgresImplementedNarrow,
		ProductionPostgresSourceFirstBenchmark:    ProductionPostgresEvidenceOnly,
		ProductionPostgresLiveBenchmarkHonesty:    ProductionPostgresEvidenceOnly,
	}
	if len(report.Rows) != len(expectedStatus) {
		return fmt.Errorf("production PostgreSQL coverage: row count = %d, want %d", len(report.Rows), len(expectedStatus))
	}
	rows := map[ProductionPostgresEvidenceID]ProductionPostgresEvidenceRow{}
	for _, row := range report.Rows {
		if row.ID == "" {
			return fmt.Errorf("production PostgreSQL coverage: row missing id")
		}
		wantStatus, ok := expectedStatus[row.ID]
		if !ok {
			return fmt.Errorf("production PostgreSQL coverage: unexpected row %q", row.ID)
		}
		if _, exists := rows[row.ID]; exists {
			return fmt.Errorf("production PostgreSQL coverage: duplicate row %q", row.ID)
		}
		rows[row.ID] = row
		if row.Status != wantStatus {
			return fmt.Errorf("production PostgreSQL coverage: row %q status = %q, want %q", row.ID, row.Status, wantStatus)
		}
		if strings.TrimSpace(row.Name) == "" || strings.TrimSpace(row.Evidence) == "" || strings.TrimSpace(row.Boundary) == "" {
			return fmt.Errorf("production PostgreSQL coverage: row %q missing evidence or boundary", row.ID)
		}
		if len(row.RequiredFacts) == 0 {
			return fmt.Errorf("production PostgreSQL coverage: row %q missing required facts", row.ID)
		}
		if row.ClaimsOfficialTechEmpowerResult {
			return fmt.Errorf("production PostgreSQL coverage: row %q claims official TechEmpower result", row.ID)
		}
		if row.ClaimsProductionDatabaseBenchmark {
			return fmt.Errorf("production PostgreSQL coverage: row %q claims production database benchmark", row.ID)
		}
		if row.ClaimsP20PerformanceMatrix {
			return fmt.Errorf("production PostgreSQL coverage: row %q claims P20 performance matrix", row.ID)
		}
		if row.ClaimsCPlusPlusRustParity {
			return fmt.Errorf("production PostgreSQL coverage: row %q claims C++/Rust parity", row.ID)
		}
		if row.ClaimsExternalProductionDatabase {
			return fmt.Errorf("production PostgreSQL coverage: row %q claims external production database", row.ID)
		}
		if row.ClaimsFullSourceLevelDriver {
			return fmt.Errorf("production PostgreSQL coverage: row %q claims full source-level driver", row.ID)
		}
		if row.ClaimsRuntimeBehaviorChange {
			return fmt.Errorf("production PostgreSQL coverage: row %q claims runtime behavior change", row.ID)
		}
	}
	for id := range expectedStatus {
		if _, ok := rows[id]; !ok {
			return fmt.Errorf("production PostgreSQL coverage: missing row %q", id)
		}
	}

	checks := []struct {
		id    ProductionPostgresEvidenceID
		wants []string
	}{
		{ProductionPostgresStartupSCRAM, []string{"readStartupReady", "AuthSASL", "TestClientCompletesSCRAMSHA256Authentication"}},
		{ProductionPostgresPreparedStatements, []string{"PreparedQueryFormat", "AppendParse", "TestClientPreparedQueryUsesExtendedProtocol"}},
		{ProductionPostgresBinaryProtocol, []string{"AppendBindFormat", "BinaryFormat", "TestAppendBindBinaryFormatsAndDecodeInt4"}},
		{ProductionPostgresPoolingBackpressure, []string{"NewPool", "ErrPoolExhausted", "TestPoolReusesReleasedConnectionAndCapsOpenConnections"}},
		{ProductionPostgresBorrowedRegionRowDecode, []string{"DecodeDataRowBorrowed", "RowStorageBorrowed", "TestDecodeDataRowBorrowedDoesNotCopyCells"}},
		{ProductionPostgresEndpointWorkloads, []string{"/db", "/queries", "/updates", "/fortunes", "TestServerFortunesEndpointFetchesSortsAndEscapesHTML"}},
		{ProductionPostgresSourceFirstBenchmark, []string{"p19.3_postgres_source_first", "DB single query", "DB fortunes"}},
		{ProductionPostgresLiveBenchmarkHonesty, []string{"validate-techempower-report", "techempower_scram_single_query_local_report.json", "techempower_scram_single_query_matrix_local_report.json", "techempower_scram_endpoint_matrix_local_report.json", "SCRAM-SHA-256", "/db", "/queries", "/updates", "/fortunes", "p19.3_postgres_source_first"}},
	}
	for _, check := range checks {
		row := rows[check.id]
		for _, want := range check.wants {
			if !containsProductionPostgresRowText(row, want) {
				return fmt.Errorf("production PostgreSQL coverage: row %q missing fact %q", row.ID, want)
			}
		}
	}
	return nil
}

func productionPostgresStartupSCRAMRow() ProductionPostgresEvidenceRow {
	return ProductionPostgresEvidenceRow{
		ID:     ProductionPostgresStartupSCRAM,
		Name:   "Startup and SCRAM-SHA-256 authentication",
		Status: ProductionPostgresImplementedNarrow,
		RequiredFacts: []string{
			"compiler/internal/pgrt/wire.go::Connect writes AppendStartupMessage before reading startup auth frames",
			"compiler/internal/pgrt/wire.go::readStartupReady handles AuthSASL, AuthSASLContinue, and AuthSASLFinal for SCRAM-SHA-256",
			"compiler/internal/pgrt/wire_test.go::TestClientCompletesSCRAMSHA256Authentication proves the local startup/SCRAM exchange",
		},
		Evidence:    "compiler/internal/pgrt/wire.go::readStartupReady; compiler/internal/pgrt/scram.go; compiler/internal/pgrt/wire_test.go::TestClientCompletesSCRAMSHA256Authentication; compiler/internal/pgrt/scram_test.go",
		Boundary:    "startup/SCRAM evidence is local PostgreSQL wire-protocol compatibility; it is not TLS, channel binding, or external production database deployment evidence",
		SourceFirst: false,
	}
}

func productionPostgresPreparedStatementsRow() ProductionPostgresEvidenceRow {
	return ProductionPostgresEvidenceRow{
		ID:     ProductionPostgresPreparedStatements,
		Name:   "Prepared statements and extended query protocol",
		Status: ProductionPostgresImplementedNarrow,
		RequiredFacts: []string{
			"compiler/internal/pgrt/wire.go::PreparedQueryFormat prepares statements before Bind/Describe/Execute/Sync",
			"compiler/internal/pgrt/wire.go::AppendParse and AppendBindFormat build extended-protocol frames",
			"compiler/internal/pgrt/wire_test.go::TestClientPreparedQueryUsesExtendedProtocol proves Parse then Bind/Describe/Execute/Sync sequencing",
		},
		Evidence:    "compiler/internal/pgrt/wire.go::Prepare; compiler/internal/pgrt/wire.go::PreparedQueryFormat; compiler/internal/pgrt/wire_test.go::TestClientPreparedQueryUsesExtendedProtocol; compiler/internal/webrt/db_test.go::TestServerDBEndpointUsesPoolAndSerializesWorld",
		Boundary:    "prepared-statement evidence covers named statement reuse in the local runtime client; it does not claim a public source-level driver API or query planner behavior",
		SourceFirst: false,
	}
}

func productionPostgresBinaryProtocolRow() ProductionPostgresEvidenceRow {
	return ProductionPostgresEvidenceRow{
		ID:     ProductionPostgresBinaryProtocol,
		Name:   "Binary protocol helpers",
		Status: ProductionPostgresImplementedNarrow,
		RequiredFacts: []string{
			"compiler/internal/pgrt/wire.go::AppendBindFormat writes parameter and result format code lists",
			"compiler/internal/pgrt/wire.go::BinaryFormat and AppendInt4Binary encode TechEmpower int4 parameters",
			"compiler/internal/pgrt/row_decode_test.go::TestAppendBindBinaryFormatsAndDecodeInt4 proves binary int4 bind/decode evidence",
		},
		Evidence:    "compiler/internal/pgrt/wire.go::AppendBindFormat; compiler/internal/pgrt/wire.go::AppendInt4Binary; compiler/internal/pgrt/wire.go::DecodeInt4; compiler/internal/pgrt/row_decode_test.go::TestAppendBindBinaryFormatsAndDecodeInt4",
		Boundary:    "binary protocol support is bounded to int4 parameters/results used by the DB endpoint path; no complete PostgreSQL binary type matrix is claimed",
		SourceFirst: false,
	}
}

func productionPostgresPoolingBackpressureRow() ProductionPostgresEvidenceRow {
	return ProductionPostgresEvidenceRow{
		ID:     ProductionPostgresPoolingBackpressure,
		Name:   "Connection pooling and backpressure",
		Status: ProductionPostgresImplementedNarrow,
		RequiredFacts: []string{
			"compiler/internal/pgrt/pool.go::NewPool creates a capped local connection pool",
			"compiler/internal/pgrt/pool.go::Checkout returns ErrPoolExhausted instead of over-opening when maxOpen is reached",
			"compiler/internal/pgrt/pool_test.go::TestPoolReusesReleasedConnectionAndCapsOpenConnections proves reuse and backpressure",
		},
		Evidence:     "compiler/internal/pgrt/pool.go::NewPool; compiler/internal/pgrt/pool.go::Checkout; compiler/internal/pgrt/pool_test.go::TestPoolReusesReleasedConnectionAndCapsOpenConnections; compiler/internal/pgrt/pool_test.go::TestPoolStatsTrackOpenIdleInUseAndClosedState",
		Boundary:     "pooling evidence is a bounded local cap/reuse model; no adaptive production pool, queue wait policy, or throughput claim is made",
		SourceFirst:  false,
		Backpressure: true,
	}
}

func productionPostgresBorrowedRegionRowDecodeRow() ProductionPostgresEvidenceRow {
	return ProductionPostgresEvidenceRow{
		ID:     ProductionPostgresBorrowedRegionRowDecode,
		Name:   "Borrowed DataRow decode evidence",
		Status: ProductionPostgresImplementedNarrow,
		RequiredFacts: []string{
			"compiler/internal/pgrt/wire.go::DecodeDataRowBorrowed returns RowStorageBorrowed with borrowed cell slices",
			"compiler/internal/pgrt/wire.go::RowDecodeReport records BorrowedCells and CopiedCells",
			"compiler/internal/pgrt/row_decode_test.go::TestDecodeDataRowBorrowedDoesNotCopyCells mutates the backing payload to prove borrowing",
		},
		Evidence:      "compiler/internal/pgrt/wire.go::DecodeDataRowBorrowed; compiler/internal/pgrt/wire.go::RowDecodeReport; compiler/internal/pgrt/row_decode_test.go::TestDecodeDataRowBorrowedDoesNotCopyCells",
		Boundary:      "borrowed row decode evidence is tied to caller-owned frame payload lifetime; no external region allocator integration or all-row zero-copy guarantee is claimed",
		SourceFirst:   false,
		BorrowedViews: true,
	}
}

func productionPostgresEndpointWorkloadsRow() ProductionPostgresEvidenceRow {
	return ProductionPostgresEvidenceRow{
		ID:     ProductionPostgresEndpointWorkloads,
		Name:   "TechEmpower DB endpoint workloads",
		Status: ProductionPostgresImplementedNarrow,
		RequiredFacts: []string{
			"compiler/internal/webrt/techempower.go exposes /db single query through DBHandler and fetchWorld",
			"compiler/internal/webrt/techempower.go exposes /queries multiple queries and /updates read-then-update paths",
			"compiler/internal/webrt/techempower.go exposes /fortunes through FortunesHandler and fetchFortunes",
			"compiler/internal/webrt/db_test.go covers /db, /queries, and /updates local fake-PostgreSQL wire paths",
			"compiler/internal/webrt/fortunes_test.go::TestServerFortunesEndpointFetchesSortsAndEscapesHTML covers /fortunes sorting and escaping",
		},
		Evidence:    "compiler/internal/webrt/techempower.go::DBHandler; compiler/internal/webrt/techempower.go::QueriesHandler; compiler/internal/webrt/techempower.go::UpdatesHandler; compiler/internal/webrt/techempower.go::FortunesHandler; compiler/internal/webrt/db_test.go; compiler/internal/webrt/fortunes_test.go",
		Boundary:    "endpoint evidence covers local runtime correctness with fake wire servers and existing local SCRAM harness artifacts; it is not an official benchmark or measured production DB deployment claim",
		SourceFirst: false,
	}
}

func productionPostgresSourceFirstBenchmarkRow() ProductionPostgresEvidenceRow {
	return ProductionPostgresEvidenceRow{
		ID:     ProductionPostgresSourceFirstBenchmark,
		Name:   "Source-first PostgreSQL benchmark gate",
		Status: ProductionPostgresEvidenceOnly,
		RequiredFacts: []string{
			"tools/cmd/truth-bench-harness supports p19.3_postgres_source_first scope",
			"p19.3_postgres_source_first requires Tetra-source DB single query, DB multiple queries, DB updates, and DB fortunes rows",
			"DB single query, DB multiple queries, DB updates, and DB fortunes rows must list proof, allocation, bounds, and P19.3 coverage artifacts",
		},
		Evidence:    "tools/cmd/truth-bench-harness/main.go::policyForBenchmarkScope; tools/cmd/truth-bench-harness/main_test.go::TestP19PostgresSourceFirstScopeRequiresTetraOnlyDBEndpointRows; reports/production-postgres-v1/benchmarks/postgres-source-first-report.json",
		Boundary:    "source-first benchmark gate records local dry-run/source evidence only; it is not an official TechEmpower result, production database benchmark, P20 performance matrix, or C++/Rust parity claim",
		SourceFirst: true,
	}
}

func productionPostgresLiveBenchmarkHonestyRow() ProductionPostgresEvidenceRow {
	return ProductionPostgresEvidenceRow{
		ID:     ProductionPostgresLiveBenchmarkHonesty,
		Name:   "Live local PostgreSQL benchmark honesty gate",
		Status: ProductionPostgresEvidenceOnly,
		RequiredFacts: []string{
			"reports/production-postgres-v1/benchmarks/postgres-source-first-report.json records p19.3_postgres_source_first dry-run source coverage for DB single query, DB multiple queries, DB updates, and DB fortunes rows",
			"docs/benchmarks/techempower_scram_single_query_local_report.json validates with tools/cmd/validate-techempower-report and covers all six local endpoints including /db, /queries, /updates, and /fortunes",
			"docs/benchmarks/techempower_scram_single_query_matrix_local_report.json validates with tools/cmd/validate-techempower-report and records SCRAM-SHA-256 PostgreSQL 16.9.0 /db matrix evidence",
			"docs/benchmarks/techempower_scram_endpoint_matrix_local_report.json validates with tools/cmd/validate-techempower-report and records SCRAM-SHA-256 /queries, /updates, and /fortunes matrix evidence",
			"tools/validators/techempower/report_test.go rejects weak SCRAM metadata, spoofed command provenance, missing git head, grid mismatches, resource snapshot gaps, summary mismatches, and weak placeholder evidence",
		},
		Evidence:    "docs/benchmarks/techempower_scram_single_query_local_report.json; docs/benchmarks/techempower_scram_single_query_matrix_local_report.json; docs/benchmarks/techempower_scram_endpoint_matrix_local_report.json; tools/cmd/validate-techempower-report; tools/validators/techempower/report_test.go",
		Boundary:    "live local SCRAM reports measure the local Tetra runtime and PostgreSQL path honestly; they are not an official TechEmpower result, production database benchmark, P20 performance matrix, C++/Rust parity claim, external production database deployment, measured speed comparison, or runtime behavior change",
		SourceFirst: true,
	}
}

func containsProductionPostgresText(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}

func containsProductionPostgresRowText(row ProductionPostgresEvidenceRow, want string) bool {
	text := strings.Join(row.RequiredFacts, "\n") + "\n" + strings.Join(row.MissingFacts, "\n") + "\n" + row.Evidence + "\n" + row.Boundary
	return strings.Contains(text, want)
}
