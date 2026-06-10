package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBuildReportRecordsP8MatrixMetadataAndTetraArtifacts(t *testing.T) {
	dir := t.TempDir()
	manifest := completeP8Manifest(t, dir)
	report, err := buildReport(manifest, false, time.Second)
	if err != nil {
		t.Fatalf("buildReport: %v", err)
	}
	wantRows := len(RequiredBenchmarkCategories()) * len(RequiredBenchmarkLanguages())
	if report.Schema != schemaV1 || len(report.Benchmarks) != wantRows {
		t.Fatalf("report schema/rows = %s/%d", report.Schema, len(report.Benchmarks))
	}
	tetra := findBenchmarkRow(t, report, "integer_loop_tetra")
	if tetra.Ran {
		t.Fatalf("dry-run report should not mark benchmark as ran")
	}
	if tetra.CompilerVersion == "" || tetra.TargetCPU == "" {
		t.Fatalf("tetra metadata missing compiler/CPU: %+v", tetra)
	}
	if tetra.BinarySizeBytes <= 0 {
		t.Fatalf("tetra binary size = %d, want positive recorded size", tetra.BinarySizeBytes)
	}
	if len(tetra.TetraProofReports) != 1 || !tetra.TetraProofReports[0].Exists {
		t.Fatalf("tetra proof artifacts = %+v, want existing report path", tetra.TetraProofReports)
	}
	if len(tetra.TetraAllocationReports) != 1 || !tetra.TetraAllocationReports[0].Exists {
		t.Fatalf("tetra allocation artifacts = %+v, want existing report path", tetra.TetraAllocationReports)
	}
	if len(tetra.TetraBoundsReports) != 1 || !tetra.TetraBoundsReports[0].Exists {
		t.Fatalf("tetra bounds artifacts = %+v, want existing report path", tetra.TetraBoundsReports)
	}
	if report.Host.GOOS == "" || report.Host.CPUs == 0 {
		t.Fatalf("host info = %+v", report.Host)
	}
	if len(report.Claims) == 0 {
		t.Fatalf("report should include claim policy note")
	}
	if err := validateReport(report); err != nil {
		t.Fatalf("validateReport: %v", err)
	}
}

func TestValidateManifestRequiresP8MatrixCommandsAndTetraReports(t *testing.T) {
	dir := t.TempDir()
	manifest := completeP8Manifest(t, dir)
	manifest.Benchmarks = manifest.Benchmarks[:len(manifest.Benchmarks)-1]
	err := validateManifest(manifest)
	if err == nil || !strings.Contains(err.Error(), "missing benchmark matrix row") {
		t.Fatalf("validateManifest without full P8 matrix = %v, want missing row", err)
	}

	manifest = completeP8Manifest(t, dir)
	for i := range manifest.Benchmarks {
		if manifest.Benchmarks[i].Language == "c" {
			manifest.Benchmarks[i].BuildCommand = []string{"cc", "-O2", "app.c", "-o", "app"}
			break
		}
	}
	err = validateManifest(manifest)
	if err == nil || !strings.Contains(err.Error(), "clang -O3") {
		t.Fatalf("validateManifest accepted non-P8 C command: %v", err)
	}

	err = validateManifest(Manifest{Benchmarks: []BenchmarkSpec{{
		Name: "bad_tetra", Category: "integer loop", Language: "tetra", CompilerVersion: "tetra dev",
		BuildCommand: []string{"tetra", "build", "app.tetra", "--explain"},
		RunCommand:   []string{"./app"},
	}}})
	if err == nil {
		t.Fatalf("validateManifest without Tetra reports succeeded")
	}
	err = validateManifest(Manifest{Benchmarks: []BenchmarkSpec{{
		Name: "bad_lang", Category: "integer loop", Language: "zig",
		BuildCommand: []string{"zig", "build-exe", "app.zig"},
		RunCommand:   []string{"./app"},
	}}})
	if err == nil {
		t.Fatalf("validateManifest with unsupported language succeeded")
	}
}

func TestValidateReportRejectsBroadClaims(t *testing.T) {
	dir := t.TempDir()
	report, err := buildReport(completeP8Manifest(t, dir), false, time.Second)
	if err != nil {
		t.Fatalf("buildReport: %v", err)
	}
	report.Claims = []string{"Tetra is the fastest language."}
	err = validateReport(report)
	if err == nil || !strings.Contains(err.Error(), "fastest language") {
		t.Fatalf("validateReport accepted broad fastest-language claim: %v", err)
	}
	report.Claims = []string{"On benchmark integer_loop_tetra, report tetra.truth.benchmark.v1, target linux-x64, Tetra ran 10 ms vs Rust 12 ms."}
	if err := validateReport(report); err != nil {
		t.Fatalf("validateReport rejected benchmark-specific claim: %v", err)
	}
}

func TestP19GenericCollectionsScopeRequiresTetraCppRustHashTableEquivalents(t *testing.T) {
	dir := t.TempDir()
	manifest := p19GenericCollectionsManifest(t, dir)
	report, err := buildReport(manifest, false, time.Second)
	if err != nil {
		t.Fatalf("buildReport P19.1 collections: %v", err)
	}
	if report.Scope != "p19.1_generic_collections" {
		t.Fatalf("report scope = %q", report.Scope)
	}
	if len(report.Benchmarks) != 3 {
		t.Fatalf("P19.1 collections rows = %d, want tetra/cpp/rust only", len(report.Benchmarks))
	}
	for _, want := range []string{"generic_collections_hash_table_tetra", "generic_collections_hash_table_cpp", "generic_collections_hash_table_rust"} {
		if row := findBenchmarkRow(t, report, want); row.Category != "hash table" || row.AlgorithmID != p19GenericCollectionsAlgoID || row.InputDescription == "" {
			t.Fatalf("row %s missing equivalent hash-table metadata: %+v", want, row)
		}
	}
	tetra := findBenchmarkRow(t, report, "generic_collections_hash_table_tetra")
	if len(tetra.TetraProofReports) != 1 || !tetra.TetraProofReports[0].Exists {
		t.Fatalf("tetra proof reports = %+v", tetra.TetraProofReports)
	}
	if len(tetra.TetraAllocationReports) != 1 || !tetra.TetraAllocationReports[0].Exists {
		t.Fatalf("tetra allocation reports = %+v", tetra.TetraAllocationReports)
	}
	if len(tetra.TetraBoundsReports) != 1 || !tetra.TetraBoundsReports[0].Exists {
		t.Fatalf("tetra bounds reports = %+v", tetra.TetraBoundsReports)
	}
	if len(tetra.TetraReports) != 1 || !tetra.TetraReports[0].Exists {
		t.Fatalf("tetra performance reports = %+v", tetra.TetraReports)
	}
}

func TestP19GenericCollectionsScopeRejectsMissingEquivalenceAndParityClaim(t *testing.T) {
	dir := t.TempDir()
	manifest := p19GenericCollectionsManifest(t, dir)
	manifest.Benchmarks = manifest.Benchmarks[:2]
	err := validateManifest(manifest)
	if err == nil || !strings.Contains(err.Error(), "missing benchmark matrix row") || !strings.Contains(err.Error(), "rust") {
		t.Fatalf("validateManifest accepted missing Rust row: %v", err)
	}

	manifest = p19GenericCollectionsManifest(t, dir)
	manifest.Benchmarks[1].AlgorithmID = "different_algorithm"
	err = validateManifest(manifest)
	if err == nil || !strings.Contains(err.Error(), "algorithm_id") {
		t.Fatalf("validateManifest accepted mismatched algorithm id: %v", err)
	}

	report, err := buildReport(p19GenericCollectionsManifest(t, dir), false, time.Second)
	if err != nil {
		t.Fatalf("buildReport P19.1 collections: %v", err)
	}
	report.Claims = []string{"This proves C++/Rust parity for stable generic collections."}
	err = validateReport(report)
	if err == nil || !strings.Contains(err.Error(), "C++/Rust parity") {
		t.Fatalf("validateReport accepted fake parity claim: %v", err)
	}
	report.Claims = []string{"This is an official benchmark result for stable generic collections."}
	err = validateReport(report)
	if err == nil || !strings.Contains(err.Error(), "official benchmark") {
		t.Fatalf("validateReport accepted fake official benchmark claim: %v", err)
	}
}

func TestP19HTTPJSONSourceFirstScopeRequiresTetraOnlyHTTPAndJSONRows(t *testing.T) {
	dir := t.TempDir()
	manifest := p19HTTPJSONSourceFirstManifest(t, dir)
	report, err := buildReport(manifest, false, time.Second)
	if err != nil {
		t.Fatalf("buildReport P19.2 HTTP/JSON source-first: %v", err)
	}
	if report.Scope != scopeP19HTTPJSONSourceFirst {
		t.Fatalf("report scope = %q", report.Scope)
	}
	if len(report.Benchmarks) != 2 {
		t.Fatalf("P19.2 HTTP/JSON rows = %d, want plaintext/json Tetra rows", len(report.Benchmarks))
	}
	for _, want := range []struct {
		name     string
		category string
	}{
		{name: "http_plaintext_tetra_source", category: "HTTP plaintext"},
		{name: "http_json_tetra_source", category: "HTTP JSON"},
	} {
		row := findBenchmarkRow(t, report, want.name)
		if row.Category != want.category || row.Language != "tetra" || row.AlgorithmID == "" || !strings.Contains(row.InputDescription, "lib.core") {
			t.Fatalf("row %s missing source-first metadata: %+v", want.name, row)
		}
		if len(row.TetraProofReports) != 1 || !row.TetraProofReports[0].Exists {
			t.Fatalf("row %s proof reports = %+v", want.name, row.TetraProofReports)
		}
		if len(row.TetraAllocationReports) != 1 || !row.TetraAllocationReports[0].Exists {
			t.Fatalf("row %s allocation reports = %+v", want.name, row.TetraAllocationReports)
		}
		if len(row.TetraBoundsReports) != 1 || !row.TetraBoundsReports[0].Exists {
			t.Fatalf("row %s bounds reports = %+v", want.name, row.TetraBoundsReports)
		}
		if len(row.TetraReports) != 1 || !row.TetraReports[0].Exists {
			t.Fatalf("row %s P19.2 reports = %+v", want.name, row.TetraReports)
		}
	}
	if err := validateReport(report); err != nil {
		t.Fatalf("validateReport P19.2 HTTP/JSON source-first: %v", err)
	}
}

func TestP19HTTPJSONSourceFirstScopeRejectsRuntimeOnlyAndOfficialClaims(t *testing.T) {
	dir := t.TempDir()
	manifest := p19HTTPJSONSourceFirstManifest(t, dir)
	manifest.Benchmarks = manifest.Benchmarks[:1]
	err := validateManifest(manifest)
	if err == nil || !strings.Contains(err.Error(), "missing benchmark matrix row") || !strings.Contains(err.Error(), "HTTP JSON") {
		t.Fatalf("validateManifest accepted missing HTTP JSON row: %v", err)
	}

	manifest = p19HTTPJSONSourceFirstManifest(t, dir)
	manifest.Benchmarks[0].BuildCommand = []string{"go", "test", "./compiler/internal/webrt"}
	err = validateManifest(manifest)
	if err == nil || !strings.Contains(err.Error(), "Tetra build command") {
		t.Fatalf("validateManifest accepted runtime-only Go command: %v", err)
	}

	report, err := buildReport(p19HTTPJSONSourceFirstManifest(t, dir), false, time.Second)
	if err != nil {
		t.Fatalf("buildReport P19.2 HTTP/JSON source-first: %v", err)
	}
	report.Claims = []string{"This is an official TechEmpower result for P19.2."}
	err = validateReport(report)
	if err == nil || !strings.Contains(err.Error(), "official TechEmpower") {
		t.Fatalf("validateReport accepted fake official TechEmpower claim: %v", err)
	}
	report.Claims = []string{"This proves C++/Rust parity for the P19.2 HTTP/JSON stack."}
	err = validateReport(report)
	if err == nil || !strings.Contains(err.Error(), "C++/Rust parity") {
		t.Fatalf("validateReport accepted fake C++/Rust parity claim: %v", err)
	}
}

func TestP19PostgresSourceFirstScopeRequiresTetraOnlyDBEndpointRows(t *testing.T) {
	dir := t.TempDir()
	manifest := p19PostgresSourceFirstManifest(t, dir)
	report, err := buildReport(manifest, false, time.Second)
	if err != nil {
		t.Fatalf("buildReport P19.3 PostgreSQL source-first: %v", err)
	}
	if report.Scope != scopeP19PostgresSourceFirst {
		t.Fatalf("report scope = %q", report.Scope)
	}
	if len(report.Benchmarks) != 4 {
		t.Fatalf("P19.3 PostgreSQL rows = %d, want four Tetra DB endpoint rows", len(report.Benchmarks))
	}
	for _, want := range []struct {
		name     string
		category string
	}{
		{name: "postgres_db_single_query_tetra_source", category: "DB single query"},
		{name: "postgres_db_multiple_queries_tetra_source", category: "DB multiple queries"},
		{name: "postgres_db_updates_tetra_source", category: "DB updates"},
		{name: "postgres_db_fortunes_tetra_source", category: "DB fortunes"},
	} {
		row := findBenchmarkRow(t, report, want.name)
		if row.Category != want.category || row.Language != "tetra" || row.AlgorithmID == "" || !strings.Contains(row.InputDescription, "lib.core.postgres") {
			t.Fatalf("row %s missing source-first PostgreSQL metadata: %+v", want.name, row)
		}
		if len(row.TetraProofReports) != 1 || !row.TetraProofReports[0].Exists {
			t.Fatalf("row %s proof reports = %+v", want.name, row.TetraProofReports)
		}
		if len(row.TetraAllocationReports) != 1 || !row.TetraAllocationReports[0].Exists {
			t.Fatalf("row %s allocation reports = %+v", want.name, row.TetraAllocationReports)
		}
		if len(row.TetraBoundsReports) != 1 || !row.TetraBoundsReports[0].Exists {
			t.Fatalf("row %s bounds reports = %+v", want.name, row.TetraBoundsReports)
		}
		if len(row.TetraReports) != 1 || !row.TetraReports[0].Exists {
			t.Fatalf("row %s P19.3 reports = %+v", want.name, row.TetraReports)
		}
	}
	if err := validateReport(report); err != nil {
		t.Fatalf("validateReport P19.3 PostgreSQL source-first: %v", err)
	}
}

func TestP19PostgresSourceFirstScopeRejectsRuntimeOnlyAndFakeDBClaims(t *testing.T) {
	dir := t.TempDir()
	manifest := p19PostgresSourceFirstManifest(t, dir)
	manifest.Benchmarks = manifest.Benchmarks[:3]
	err := validateManifest(manifest)
	if err == nil || !strings.Contains(err.Error(), "missing benchmark matrix row") || !strings.Contains(err.Error(), "DB fortunes") {
		t.Fatalf("validateManifest accepted missing DB fortunes row: %v", err)
	}

	manifest = p19PostgresSourceFirstManifest(t, dir)
	manifest.Benchmarks[0].BuildCommand = []string{"go", "test", "./compiler/internal/pgrt"}
	err = validateManifest(manifest)
	if err == nil || !strings.Contains(err.Error(), "Tetra build command") {
		t.Fatalf("validateManifest accepted runtime-only Go command: %v", err)
	}

	report, err := buildReport(p19PostgresSourceFirstManifest(t, dir), false, time.Second)
	if err != nil {
		t.Fatalf("buildReport P19.3 PostgreSQL source-first: %v", err)
	}
	report.Claims = []string{"This is an official TechEmpower result for P19.3 PostgreSQL."}
	err = validateReport(report)
	if err == nil || !strings.Contains(err.Error(), "official TechEmpower") {
		t.Fatalf("validateReport accepted fake official TechEmpower claim: %v", err)
	}
	report.Claims = []string{"This proves C++/Rust parity for the P19.3 PostgreSQL driver."}
	err = validateReport(report)
	if err == nil || !strings.Contains(err.Error(), "C++/Rust parity") {
		t.Fatalf("validateReport accepted fake C++/Rust parity claim: %v", err)
	}
}

func TestP20BenchmarkMatrixScopeRequiresMasterPlanRowsRawOutputsAndTetraReports(t *testing.T) {
	dir := t.TempDir()
	manifest := p20BenchmarkMatrixManifest(t, dir)
	report, err := buildReport(manifest, false, time.Second)
	if err != nil {
		t.Fatalf("buildReport P20.0 benchmark matrix: %v", err)
	}
	if report.Scope != scopeP20BenchmarkMatrix {
		t.Fatalf("report scope = %q", report.Scope)
	}
	wantRows := len(P20BenchmarkCategories()) * len(RequiredBenchmarkLanguages())
	if len(report.Benchmarks) != wantRows {
		t.Fatalf("P20.0 benchmark rows = %d, want %d", len(report.Benchmarks), wantRows)
	}
	for _, category := range []string{"function calls", "recursion", "matrix multiply", "JSON parse/stringify", "HTTP plaintext/json", "PostgreSQL single/multiple/update", "startup time", "binary size", "compile time"} {
		for _, language := range RequiredBenchmarkLanguages() {
			row := findBenchmarkRow(t, report, slugCategory(category)+"_"+language)
			if row.Category != category || row.Language != language {
				t.Fatalf("row identity for %s/%s = %+v", category, language, row)
			}
			if row.AlgorithmID == "" || row.InputDescription == "" {
				t.Fatalf("row %s missing equivalence metadata: %+v", row.Name, row)
			}
			if len(row.RawOutputArtifacts) != 1 || !row.RawOutputArtifacts[0].Exists {
				t.Fatalf("row %s raw output artifacts = %+v", row.Name, row.RawOutputArtifacts)
			}
			if row.TargetCPU != report.Host.TargetCPU {
				t.Fatalf("row %s target_cpu = %q, want host %q", row.Name, row.TargetCPU, report.Host.TargetCPU)
			}
		}
	}
	tetra := findBenchmarkRow(t, report, "compile_time_tetra")
	if len(tetra.TetraProofReports) != 1 || !tetra.TetraProofReports[0].Exists {
		t.Fatalf("tetra proof reports = %+v", tetra.TetraProofReports)
	}
	if len(tetra.TetraAllocationReports) != 1 || !tetra.TetraAllocationReports[0].Exists {
		t.Fatalf("tetra allocation reports = %+v", tetra.TetraAllocationReports)
	}
	if len(tetra.TetraBoundsReports) != 1 || !tetra.TetraBoundsReports[0].Exists {
		t.Fatalf("tetra bounds reports = %+v", tetra.TetraBoundsReports)
	}
	if len(tetra.TetraReports) != 1 || !tetra.TetraReports[0].Exists {
		t.Fatalf("tetra performance reports = %+v", tetra.TetraReports)
	}
	if err := validateReport(report); err != nil {
		t.Fatalf("validateReport P20.0 benchmark matrix: %v", err)
	}
}

func TestP20BenchmarkMatrixScopeRejectsWeakEvidenceAndFakeClaims(t *testing.T) {
	dir := t.TempDir()
	manifest := p20BenchmarkMatrixManifest(t, dir)
	manifest.Benchmarks = manifest.Benchmarks[:len(manifest.Benchmarks)-1]
	err := validateManifest(manifest)
	if err == nil || !strings.Contains(err.Error(), "missing benchmark matrix row") || !strings.Contains(err.Error(), "compile time") {
		t.Fatalf("validateManifest accepted missing P20.0 row: %v", err)
	}

	manifest = p20BenchmarkMatrixManifest(t, dir)
	manifest.Benchmarks[1].AlgorithmID = "different_algorithm"
	err = validateManifest(manifest)
	if err == nil || !strings.Contains(err.Error(), "algorithm_id") {
		t.Fatalf("validateManifest accepted mismatched P20.0 algorithm id: %v", err)
	}

	manifest = p20BenchmarkMatrixManifest(t, dir)
	manifest.Benchmarks[0].RawOutputArtifacts = nil
	err = validateManifest(manifest)
	if err == nil || !strings.Contains(err.Error(), "raw output") {
		t.Fatalf("validateManifest accepted missing raw output artifact: %v", err)
	}

	report, err := buildReport(p20BenchmarkMatrixManifest(t, dir), false, time.Second)
	if err != nil {
		t.Fatalf("buildReport P20.0 benchmark matrix: %v", err)
	}
	report.Benchmarks[0].TargetCPU = "different target CPU"
	err = validateReport(report)
	if err == nil || !strings.Contains(err.Error(), "target CPU") {
		t.Fatalf("validateReport accepted target CPU mismatch: %v", err)
	}

	report, err = buildReport(p20BenchmarkMatrixManifest(t, dir), false, time.Second)
	if err != nil {
		t.Fatalf("buildReport P20.0 benchmark matrix: %v", err)
	}
	report.Claims = []string{"This proves C++/Rust parity for the P20 benchmark matrix."}
	err = validateReport(report)
	if err == nil || !strings.Contains(err.Error(), "C++/Rust parity") {
		t.Fatalf("validateReport accepted fake P20 parity claim: %v", err)
	}
}

func TestP15ActorBenchmarkPrepScopeRequiresRowsRawArtifactsAndNoClaims(t *testing.T) {
	dir := t.TempDir()
	manifest := p15ActorBenchmarkPrepManifest(t, dir)
	report, err := buildReport(manifest, false, time.Second)
	if err != nil {
		t.Fatalf("buildReport P15 actor benchmark prep: %v", err)
	}
	if report.Scope != "p15_actor_benchmark_prep" {
		t.Fatalf("report scope = %q", report.Scope)
	}
	if len(report.Benchmarks) != 5 {
		t.Fatalf("P15 actor benchmark rows = %d, want 5", len(report.Benchmarks))
	}
	for _, want := range []string{
		"actor_ping_pong_tetra",
		"actor_fanout_fanin_tetra",
		"actor_mailbox_throughput_tetra",
		"actor_backpressure_latency_tetra",
		"zero_copy_move_local_typed_mailbox_tetra",
	} {
		row := findBenchmarkRow(t, report, want)
		if row.Language != "tetra" || row.AlgorithmID == "" || row.InputDescription == "" {
			t.Fatalf("row %s missing actor benchmark identity metadata: %+v", want, row)
		}
		if row.Ran {
			t.Fatalf("row %s should remain dry-run Tier 0 prep until explicitly run", want)
		}
		if len(row.RawOutputArtifacts) != 1 || !row.RawOutputArtifacts[0].Exists {
			t.Fatalf("row %s raw output artifacts = %+v, want existing raw artifact", want, row.RawOutputArtifacts)
		}
	}
	if err := validateReport(report); err != nil {
		t.Fatalf("validateReport P15 actor benchmark prep: %v", err)
	}
}

func TestP15ActorBenchmarkPrepRejectsMissingRawArtifactsAndOverclaims(t *testing.T) {
	dir := t.TempDir()
	manifest := p15ActorBenchmarkPrepManifest(t, dir)
	manifest.Benchmarks[0].RawOutputArtifacts = nil
	err := validateManifest(manifest)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "raw output") {
		t.Fatalf("validateManifest accepted missing P15 actor raw output artifact: %v", err)
	}

	report, err := buildReport(p15ActorBenchmarkPrepManifest(t, dir), false, time.Second)
	if err != nil {
		t.Fatalf("buildReport P15 actor benchmark prep: %v", err)
	}
	report.Claims = []string{"Actor benchmark report proves Tetra actors are faster than Rust actors."}
	err = validateReport(report)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "actor benchmark") {
		t.Fatalf("validateReport accepted actor benchmark superiority claim: %v", err)
	}

	report, err = buildReport(p15ActorBenchmarkPrepManifest(t, dir), false, time.Second)
	if err != nil {
		t.Fatalf("buildReport P15 actor benchmark prep: %v", err)
	}
	report.Claims = []string{"The zero_copy_move prototype benchmark proves production runtime zero-copy for actors."}
	err = validateReport(report)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "zero_copy_move") {
		t.Fatalf("validateReport accepted zero_copy_move production runtime claim: %v", err)
	}
}

func TestP20ClaimTierReportDefinesFiveTiersAndCurrentTierZeroClaims(t *testing.T) {
	report := BuildP20ClaimTierReport()
	if report.Schema != claimTierSchemaV1 || report.Scope != scopeP20ClaimTiers {
		t.Fatalf("claim-tier report schema/scope = %q/%q", report.Schema, report.Scope)
	}
	wantPolicies := []struct {
		id       string
		label    string
		evidence string
	}{
		{id: "tier0_local_smoke_only", label: "Tier 0: local smoke only", evidence: "local_smoke"},
		{id: "tier1_local_benchmark_evidence", label: "Tier 1: local benchmark evidence", evidence: "local_benchmark"},
		{id: "tier2_reproducible_cross_machine_benchmark", label: "Tier 2: reproducible cross-machine benchmark", evidence: "cross_machine_reproduction"},
		{id: "tier3_independent_reproduced_benchmark", label: "Tier 3: independent reproduced benchmark", evidence: "independent_reproduction"},
		{id: "tier4_official_upstream_benchmark_submission", label: "Tier 4: official upstream benchmark submission", evidence: "official_upstream_submission"},
	}
	policies := map[string]ClaimTierPolicy{}
	for _, policy := range report.Policies {
		policies[policy.ID] = policy
	}
	if len(policies) != len(wantPolicies) {
		t.Fatalf("claim-tier policies = %d, want %d: %+v", len(policies), len(wantPolicies), report.Policies)
	}
	for rank, want := range wantPolicies {
		policy, ok := policies[want.id]
		if !ok {
			t.Fatalf("missing policy %q in %+v", want.id, report.Policies)
		}
		if policy.Rank != rank || policy.Label != want.label {
			t.Fatalf("policy %s rank/label = %d/%q, want %d/%q", want.id, policy.Rank, policy.Label, rank, want.label)
		}
		if !containsString(policy.RequiredEvidenceClasses, want.evidence) {
			t.Fatalf("policy %s evidence = %+v, want %q", want.id, policy.RequiredEvidenceClasses, want.evidence)
		}
	}
	if len(report.Claims) == 0 {
		t.Fatalf("claim-tier report has no public claims")
	}
	current := report.Claims[0]
	if current.ID != "p20_current_local_smoke_only" || current.Tier != "tier0_local_smoke_only" {
		t.Fatalf("current claim id/tier = %q/%q", current.ID, current.Tier)
	}
	lower := strings.ToLower(current.Text)
	for _, want := range []string{"local smoke", "dry-run benchmark matrix", "performance-blocker explanation", "no measured speed", "no c++/rust parity", "no official benchmark"} {
		if !strings.Contains(lower, want) {
			t.Fatalf("current Tier 0 claim missing %q: %q", want, current.Text)
		}
	}
	for _, want := range []string{"local_smoke", "dry_run_matrix", "performance_blocker_report"} {
		if !claimHasEvidenceClass(current, want) {
			t.Fatalf("current Tier 0 claim evidence = %+v, want class %q", current.Evidence, want)
		}
	}
	for _, want := range []string{"measured speed", "C++/Rust parity", "official benchmark", "official TechEmpower", "cross-machine", "independent reproduced"} {
		if !containsString(report.NonClaims, want) {
			t.Fatalf("claim-tier non-claims = %+v, want %q", report.NonClaims, want)
		}
	}
	if err := ValidateClaimTierReport(report); err != nil {
		t.Fatalf("ValidateClaimTierReport: %v", err)
	}
}

func TestP20ClaimTierReportIncludesActorBenchmarkPrepNonClaims(t *testing.T) {
	report := BuildP20ClaimTierReport()
	var actorClaim PublicPerformanceClaim
	for _, claim := range report.Claims {
		if claim.ID == "p15_actor_benchmark_prep_tier0" {
			actorClaim = claim
			break
		}
	}
	if actorClaim.ID == "" {
		t.Fatalf("claim-tier report missing P15 actor benchmark prep claim: %+v", report.Claims)
	}
	if actorClaim.Tier != "tier0_local_smoke_only" {
		t.Fatalf("actor benchmark claim tier = %q, want tier0_local_smoke_only", actorClaim.Tier)
	}
	lower := strings.ToLower(actorClaim.Text)
	for _, want := range []string{"actor ping-pong", "fanout/fanin", "mailbox throughput", "backpressure latency", "zero_copy_move", "no measured speed", "no official benchmark", "no distributed zero-copy"} {
		if !strings.Contains(lower, want) {
			t.Fatalf("actor benchmark Tier 0 claim missing %q: %q", want, actorClaim.Text)
		}
	}
	for _, want := range []string{"production throughput guarantee", "distributed zero-copy", "actor benchmark superiority"} {
		if !containsString(report.NonClaims, want) {
			t.Fatalf("claim-tier non-claims = %+v, want %q", report.NonClaims, want)
		}
	}
	if err := ValidateClaimTierReport(report); err != nil {
		t.Fatalf("ValidateClaimTierReport with actor benchmark prep: %v", err)
	}
}

func TestValidateP20ClaimTierReportRejectsOverstatedWording(t *testing.T) {
	for _, tc := range []struct {
		name    string
		tier    string
		text    string
		wantErr string
	}{
		{
			name:    "local benchmark wording needs Tier 1 evidence",
			tier:    "tier0_local_smoke_only",
			text:    "Tetra has local benchmark evidence and a measured local benchmark result.",
			wantErr: "tier 1",
		},
		{
			name:    "cross-machine wording needs Tier 2 evidence",
			tier:    "tier1_local_benchmark_evidence",
			text:    "Tetra has a reproducible cross-machine benchmark result.",
			wantErr: "tier 2",
		},
		{
			name:    "independent wording needs Tier 3 evidence",
			tier:    "tier1_local_benchmark_evidence",
			text:    "Tetra performance was independently reproduced by a third party.",
			wantErr: "tier 3",
		},
		{
			name:    "official wording needs Tier 4 evidence",
			tier:    "tier0_local_smoke_only",
			text:    "Tetra has an official upstream benchmark submission and official TechEmpower result.",
			wantErr: "tier 4",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			report := BuildP20ClaimTierReport()
			report.Claims[0].Tier = tc.tier
			report.Claims[0].Text = tc.text
			err := ValidateClaimTierReport(report)
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), tc.wantErr) {
				t.Fatalf("ValidateClaimTierReport error = %v, want %q", err, tc.wantErr)
			}
		})
	}
	report := BuildP20ClaimTierReport()
	report.Claims[0].Tier = "tier1_local_benchmark_evidence"
	err := ValidateClaimTierReport(report)
	if err == nil || !strings.Contains(err.Error(), "local_benchmark") {
		t.Fatalf("ValidateClaimTierReport accepted Tier 1 without local benchmark evidence: %v", err)
	}

	report = BuildP20ClaimTierReport()
	report.Claims[0].Evidence[0].Artifact = "TODO"
	err = ValidateClaimTierReport(report)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "placeholder") {
		t.Fatalf("ValidateClaimTierReport accepted placeholder artifact: %v", err)
	}
}

func TestValidateClaimsRejectsFakeHigherTierWording(t *testing.T) {
	for _, tc := range []struct {
		claim   string
		wantErr string
	}{
		{claim: "This is local benchmark evidence for a measured Tetra result.", wantErr: "tier 1"},
		{claim: "This is a reproducible cross-machine benchmark result for Tetra.", wantErr: "tier 2"},
		{claim: "Tetra performance was independently reproduced by a third party.", wantErr: "tier 3"},
		{claim: "This is an official upstream benchmark submission for Tetra.", wantErr: "tier 4"},
	} {
		err := validateClaims([]string{tc.claim})
		if err == nil || !strings.Contains(strings.ToLower(err.Error()), tc.wantErr) {
			t.Fatalf("validateClaims(%q) = %v, want %q", tc.claim, err, tc.wantErr)
		}
	}
	safe := "No official upstream benchmark submission, independent reproduced benchmark, cross-machine benchmark, measured speed, or C++/Rust parity is claimed."
	if err := validateClaims([]string{safe}); err != nil {
		t.Fatalf("validateClaims rejected explicit non-claim: %v", err)
	}
}

func completeP8Manifest(t *testing.T, dir string) Manifest {
	t.Helper()
	proofPath := writeFixture(t, dir, "app.proof.json", `{"kind":"proof"}`)
	allocPath := writeFixture(t, dir, "app.alloc.json", `{"kind":"alloc"}`)
	boundsPath := writeFixture(t, dir, "app.bounds.json", `{"kind":"bounds"}`)
	out := Manifest{}
	for _, category := range RequiredBenchmarkCategories() {
		slug := slugCategory(category)
		for _, language := range RequiredBenchmarkLanguages() {
			name := slug + "_" + language
			binary := writeFixture(t, dir, name, "binary")
			bench := BenchmarkSpec{
				Name:            name,
				Category:        category,
				Language:        language,
				CompilerVersion: language + " compiler version",
				BuildCommand:    p8BuildCommand(language, slug),
				RunCommand:      []string{filepath.Join(dir, name)},
				Binary:          binary,
			}
			if language == "tetra" {
				bench.TetraProofReports = []string{proofPath}
				bench.TetraAllocationReports = []string{allocPath}
				bench.TetraBoundsReports = []string{boundsPath}
			}
			out.Benchmarks = append(out.Benchmarks, bench)
		}
	}
	return out
}

func p19GenericCollectionsManifest(t *testing.T, dir string) Manifest {
	t.Helper()
	proofPath := writeFixture(t, dir, "generic_collections_hash_table.proof.json", `{"kind":"proof","benchmark":"generic_collections_hash_table"}`)
	allocPath := writeFixture(t, dir, "generic_collections_hash_table.allocation.json", `{"kind":"allocation","benchmark":"generic_collections_hash_table"}`)
	boundsPath := writeFixture(t, dir, "generic_collections_hash_table.bounds.json", `{"kind":"bounds","benchmark":"generic_collections_hash_table"}`)
	perfPath := writeFixture(t, dir, "generic_collections_hash_table.perf.json", `{"kind":"performance","benchmark":"generic_collections_hash_table","claim":"no parity claim"}`)
	algorithmID := p19GenericCollectionsAlgoID
	input := "deterministic 1024-key i32 lookup workload with identical keys, queries, and fallback value"
	rows := []BenchmarkSpec{
		{
			Name:                   "generic_collections_hash_table_tetra",
			Category:               "hash table",
			Language:               "tetra",
			CompilerVersion:        "tetra dev",
			AlgorithmID:            algorithmID,
			InputDescription:       input,
			BuildCommand:           []string{"tetra", "build", "benchmarks/generic_collections/hash_table.tetra", "--explain"},
			RunCommand:             []string{filepath.Join(dir, "generic_collections_hash_table_tetra")},
			Binary:                 writeFixture(t, dir, "generic_collections_hash_table_tetra", "binary"),
			TetraProofReports:      []string{proofPath},
			TetraAllocationReports: []string{allocPath},
			TetraBoundsReports:     []string{boundsPath},
			TetraReports:           []string{perfPath},
		},
		{
			Name:             "generic_collections_hash_table_cpp",
			Category:         "hash table",
			Language:         "cpp",
			CompilerVersion:  "clang++ test",
			AlgorithmID:      algorithmID,
			InputDescription: input,
			BuildCommand:     []string{"clang++", "-O3", "benchmarks/generic_collections/hash_table.cpp", "-o", "generic_collections_hash_table_cpp"},
			RunCommand:       []string{filepath.Join(dir, "generic_collections_hash_table_cpp")},
			Binary:           writeFixture(t, dir, "generic_collections_hash_table_cpp", "binary"),
		},
		{
			Name:             "generic_collections_hash_table_rust",
			Category:         "hash table",
			Language:         "rust",
			CompilerVersion:  "rustc test",
			AlgorithmID:      algorithmID,
			InputDescription: input,
			BuildCommand:     []string{"rustc", "-C", "opt-level=3", "benchmarks/generic_collections/hash_table.rs", "-o", "generic_collections_hash_table_rust"},
			RunCommand:       []string{filepath.Join(dir, "generic_collections_hash_table_rust")},
			Binary:           writeFixture(t, dir, "generic_collections_hash_table_rust", "binary"),
		},
	}
	return Manifest{Scope: "p19.1_generic_collections", Benchmarks: rows}
}

func p19HTTPJSONSourceFirstManifest(t *testing.T, dir string) Manifest {
	t.Helper()
	proofPath := writeFixture(t, dir, "http-json-source-first.proof.json", `{"kind":"proof","slice":"p19.2_http_json_source_first"}`)
	allocPath := writeFixture(t, dir, "http-json-source-first.allocation.json", `{"kind":"allocation","slice":"p19.2_http_json_source_first"}`)
	boundsPath := writeFixture(t, dir, "http-json-source-first.bounds.json", `{"kind":"bounds","slice":"p19.2_http_json_source_first"}`)
	coveragePath := writeFixture(t, dir, "http-json-source-first.coverage.json", `{"schema_version":"tetra.stdlib.http_json.production_stack.v1","claim":"not official and no C++/Rust parity claim"}`)
	httpBin := writeFixture(t, dir, "http_plaintext_tetra_source", "binary")
	jsonBin := writeFixture(t, dir, "http_json_tetra_source", "binary")
	return Manifest{
		Scope: scopeP19HTTPJSONSourceFirst,
		Benchmarks: []BenchmarkSpec{
			{
				Name:                   "http_plaintext_tetra_source",
				Category:               "HTTP plaintext",
				Language:               "tetra",
				CompilerVersion:        "tetra dev",
				AlgorithmID:            "p19.2.http_json.http_plaintext.request_head_response_tetra_source",
				InputDescription:       "deterministic lib.core.http request-head, pipelining, route, and plaintext response smoke",
				BuildCommand:           []string{"tetra", "build", "examples/core_http_smoke.tetra", "--explain", "--out", httpBin},
				RunCommand:             []string{httpBin},
				Binary:                 httpBin,
				TetraProofReports:      []string{proofPath},
				TetraAllocationReports: []string{allocPath},
				TetraBoundsReports:     []string{boundsPath},
				TetraReports:           []string{coveragePath},
			},
			{
				Name:                   "http_json_tetra_source",
				Category:               "HTTP JSON",
				Language:               "tetra",
				CompilerVersion:        "tetra dev",
				AlgorithmID:            "p19.2.http_json.json_message_response_tetra_source",
				InputDescription:       "deterministic lib.core.json message-object writer and lib.core.http JSON response smoke",
				BuildCommand:           []string{"tetra", "build", "examples/core_json_smoke.tetra", "--explain", "--out", jsonBin},
				RunCommand:             []string{jsonBin},
				Binary:                 jsonBin,
				TetraProofReports:      []string{proofPath},
				TetraAllocationReports: []string{allocPath},
				TetraBoundsReports:     []string{boundsPath},
				TetraReports:           []string{coveragePath},
			},
		},
	}
}

func p19PostgresSourceFirstManifest(t *testing.T, dir string) Manifest {
	t.Helper()
	proofPath := writeFixture(t, dir, "postgres-source-first.proof.json", `{"kind":"proof","slice":"p19.3_postgres_source_first"}`)
	allocPath := writeFixture(t, dir, "postgres-source-first.allocation.json", `{"kind":"allocation","slice":"p19.3_postgres_source_first"}`)
	boundsPath := writeFixture(t, dir, "postgres-source-first.bounds.json", `{"kind":"bounds","slice":"p19.3_postgres_source_first"}`)
	coveragePath := writeFixture(t, dir, "postgres-source-first.coverage.json", `{"schema_version":"tetra.stdlib.postgresql.production_driver.v1","claim":"not official and no C++/Rust parity claim"}`)
	input := "deterministic TechEmpower hello_world PostgreSQL workload through lib.core.postgres source helpers and local runtime endpoint coverage"
	rows := []BenchmarkSpec{
		{
			Name:                   "postgres_db_single_query_tetra_source",
			Category:               "DB single query",
			Language:               "tetra",
			CompilerVersion:        "tetra dev",
			AlgorithmID:            "p19.3.postgres.single_query.world_by_id_tetra_source",
			InputDescription:       input,
			BuildCommand:           []string{"tetra", "build", "examples/core_postgres_result_smoke.tetra", "--explain", "--out", filepath.Join(dir, "postgres_db_single_query_tetra_source")},
			RunCommand:             []string{filepath.Join(dir, "postgres_db_single_query_tetra_source")},
			Binary:                 writeFixture(t, dir, "postgres_db_single_query_tetra_source", "binary"),
			TetraProofReports:      []string{proofPath},
			TetraAllocationReports: []string{allocPath},
			TetraBoundsReports:     []string{boundsPath},
			TetraReports:           []string{coveragePath},
		},
		{
			Name:                   "postgres_db_multiple_queries_tetra_source",
			Category:               "DB multiple queries",
			Language:               "tetra",
			CompilerVersion:        "tetra dev",
			AlgorithmID:            "p19.3.postgres.multiple_queries.world_by_id_tetra_source",
			InputDescription:       input,
			BuildCommand:           []string{"tetra", "build", "examples/core_postgres_prepared_smoke.tetra", "--explain", "--out", filepath.Join(dir, "postgres_db_multiple_queries_tetra_source")},
			RunCommand:             []string{filepath.Join(dir, "postgres_db_multiple_queries_tetra_source")},
			Binary:                 writeFixture(t, dir, "postgres_db_multiple_queries_tetra_source", "binary"),
			TetraProofReports:      []string{proofPath},
			TetraAllocationReports: []string{allocPath},
			TetraBoundsReports:     []string{boundsPath},
			TetraReports:           []string{coveragePath},
		},
		{
			Name:                   "postgres_db_updates_tetra_source",
			Category:               "DB updates",
			Language:               "tetra",
			CompilerVersion:        "tetra dev",
			AlgorithmID:            "p19.3.postgres.updates.read_then_write_world_tetra_source",
			InputDescription:       input,
			BuildCommand:           []string{"tetra", "build", "examples/core_postgres_prepared_smoke.tetra", "--explain", "--out", filepath.Join(dir, "postgres_db_updates_tetra_source")},
			RunCommand:             []string{filepath.Join(dir, "postgres_db_updates_tetra_source")},
			Binary:                 writeFixture(t, dir, "postgres_db_updates_tetra_source", "binary"),
			TetraProofReports:      []string{proofPath},
			TetraAllocationReports: []string{allocPath},
			TetraBoundsReports:     []string{boundsPath},
			TetraReports:           []string{coveragePath},
		},
		{
			Name:                   "postgres_db_fortunes_tetra_source",
			Category:               "DB fortunes",
			Language:               "tetra",
			CompilerVersion:        "tetra dev",
			AlgorithmID:            "p19.3.postgres.fortunes.select_sort_escape_tetra_source",
			InputDescription:       input,
			BuildCommand:           []string{"tetra", "build", "examples/core_postgres_result_smoke.tetra", "--explain", "--out", filepath.Join(dir, "postgres_db_fortunes_tetra_source")},
			RunCommand:             []string{filepath.Join(dir, "postgres_db_fortunes_tetra_source")},
			Binary:                 writeFixture(t, dir, "postgres_db_fortunes_tetra_source", "binary"),
			TetraProofReports:      []string{proofPath},
			TetraAllocationReports: []string{allocPath},
			TetraBoundsReports:     []string{boundsPath},
			TetraReports:           []string{coveragePath},
		},
	}
	return Manifest{Scope: scopeP19PostgresSourceFirst, Benchmarks: rows}
}

func p20BenchmarkMatrixManifest(t *testing.T, dir string) Manifest {
	t.Helper()
	proofPath := writeFixture(t, dir, "p20.proof.json", `{"kind":"proof","slice":"p20.0_benchmark_matrix"}`)
	allocPath := writeFixture(t, dir, "p20.allocation.json", `{"kind":"allocation","slice":"p20.0_benchmark_matrix"}`)
	boundsPath := writeFixture(t, dir, "p20.bounds.json", `{"kind":"bounds","slice":"p20.0_benchmark_matrix"}`)
	perfPath := writeFixture(t, dir, "p20.perf.json", `{"kind":"performance","slice":"p20.0_benchmark_matrix","claim":"no parity claim"}`)
	out := Manifest{Scope: scopeP20BenchmarkMatrix}
	for _, category := range P20BenchmarkCategories() {
		slug := slugCategory(category)
		algorithmID := "p20.0." + strings.ReplaceAll(slug, "_", ".")
		input := "deterministic P20.0 " + category + " workload with identical inputs across Tetra, C, C++, and Rust"
		for _, language := range RequiredBenchmarkLanguages() {
			name := slug + "_" + language
			binary := writeFixture(t, dir, name, "binary")
			rawOutput := writeFixture(t, dir, name+".raw.txt", "raw output for "+name+"\n")
			bench := BenchmarkSpec{
				Name:               name,
				Category:           category,
				Language:           language,
				CompilerVersion:    language + " compiler version",
				AlgorithmID:        algorithmID,
				InputDescription:   input,
				BuildCommand:       p8BuildCommand(language, slug),
				RunCommand:         []string{binary},
				Binary:             binary,
				RawOutputArtifacts: []string{rawOutput},
			}
			if language == "tetra" {
				bench.TetraProofReports = []string{proofPath}
				bench.TetraAllocationReports = []string{allocPath}
				bench.TetraBoundsReports = []string{boundsPath}
				bench.TetraReports = []string{perfPath}
			}
			out.Benchmarks = append(out.Benchmarks, bench)
		}
	}
	return out
}

func p15ActorBenchmarkPrepManifest(t *testing.T, dir string) Manifest {
	t.Helper()
	proofPath := writeFixture(t, dir, "p15-actor.proof.json", `{"kind":"proof","slice":"p15_actor_benchmark_prep"}`)
	allocPath := writeFixture(t, dir, "p15-actor.allocation.json", `{"kind":"allocation","slice":"p15_actor_benchmark_prep"}`)
	boundsPath := writeFixture(t, dir, "p15-actor.bounds.json", `{"kind":"bounds","slice":"p15_actor_benchmark_prep"}`)
	perfPath := writeFixture(t, dir, "p15-actor.perf.json", `{"kind":"performance","slice":"p15_actor_benchmark_prep","claim":"Tier 0 prep only; no measured speed"}`)
	categories := []struct {
		name      string
		category  string
		algorithm string
		input     string
	}{
		{
			name:      "actor_ping_pong_tetra",
			category:  "actor ping-pong",
			algorithm: "p15.actor.ping_pong.local_mailbox",
			input:     "dry-run actor ping-pong local Linux-x64 mailbox workload with raw artifact references only",
		},
		{
			name:      "actor_fanout_fanin_tetra",
			category:  "actor fanout/fanin",
			algorithm: "p15.actor.fanout_fanin.local_mailbox",
			input:     "dry-run actor fanout/fanin local Linux-x64 mailbox workload with raw artifact references only",
		},
		{
			name:      "actor_mailbox_throughput_tetra",
			category:  "actor mailbox throughput",
			algorithm: "p15.actor.mailbox_throughput.local_mailbox",
			input:     "dry-run actor mailbox throughput local Linux-x64 prep row without production throughput claim",
		},
		{
			name:      "actor_backpressure_latency_tetra",
			category:  "actor backpressure latency",
			algorithm: "p15.actor.backpressure_latency.local_mailbox",
			input:     "dry-run actor backpressure latency diagnostic prep row without real-world SLA claim",
		},
		{
			name:      "zero_copy_move_local_typed_mailbox_tetra",
			category:  "zero_copy_move local typed mailbox",
			algorithm: "p15.actor.zero_copy_move.local_owned_region",
			input:     "dry-run local owned-region typed mailbox transfer prep row without distributed zero-copy claim",
		},
	}
	out := Manifest{Scope: "p15_actor_benchmark_prep"}
	for _, category := range categories {
		binary := writeFixture(t, dir, category.name, "binary")
		rawOutput := writeFixture(t, dir, category.name+".raw.txt", "raw output for "+category.name+"\n")
		out.Benchmarks = append(out.Benchmarks, BenchmarkSpec{
			Name:                   category.name,
			Category:               category.category,
			Language:               "tetra",
			CompilerVersion:        "tetra dev",
			AlgorithmID:            category.algorithm,
			InputDescription:       category.input,
			BuildCommand:           []string{"tetra", "build", "examples/actors_pingpong.tetra", "--explain", "--out", binary},
			RunCommand:             []string{binary},
			Binary:                 binary,
			TetraProofReports:      []string{proofPath},
			TetraAllocationReports: []string{allocPath},
			TetraBoundsReports:     []string{boundsPath},
			TetraReports:           []string{perfPath},
			RawOutputArtifacts:     []string{rawOutput},
		})
	}
	return out
}

func p8BuildCommand(language string, slug string) []string {
	switch language {
	case "tetra":
		return []string{"tetra", "build", slug + ".tetra", "--explain"}
	case "c":
		return []string{"clang", "-O3", slug + ".c", "-o", slug + "_c"}
	case "cpp":
		return []string{"clang++", "-O3", slug + ".cpp", "-o", slug + "_cpp"}
	case "rust":
		return []string{"rustc", "-C", "opt-level=3", slug + ".rs", "-o", slug + "_rust"}
	default:
		return []string{language, slug}
	}
}

func writeFixture(t *testing.T, dir string, name string, body string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write fixture %s: %v", name, err)
	}
	return path
}

func findBenchmarkRow(t *testing.T, report Report, name string) BenchmarkResult {
	t.Helper()
	for _, row := range report.Benchmarks {
		if row.Name == name {
			return row
		}
	}
	t.Fatalf("benchmark row %q not found", name)
	return BenchmarkResult{}
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func claimHasEvidenceClass(claim PublicPerformanceClaim, want string) bool {
	for _, evidence := range claim.Evidence {
		if evidence.Class == want {
			return true
		}
	}
	return false
}
