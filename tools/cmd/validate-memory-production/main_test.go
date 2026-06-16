package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestValidateMemoryProductionReportAcceptsValidReport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.json")
	if err := os.WriteFile(path, []byte(validMemoryProductionReport()), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := validateMemoryProductionReport(path); err != nil {
		t.Fatalf("validateMemoryProductionReport failed: %v", err)
	}
}

func TestValidateMemoryProductionReportRejectsInvalidReport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.json")
	raw := strings.Replace(validMemoryProductionReport(), `"schema": "tetra.memory.production.v1"`, `"schema": "tetra.memory.fake.v1"`, 1)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemoryProductionReport(path)
	if err == nil {
		t.Fatalf("expected invalid memory production report to fail")
	}
	if !strings.Contains(err.Error(), "tetra.memory.production.v1") {
		t.Fatalf("error = %v, want schema rejection", err)
	}
}

func TestValidateMemoryProductionReportRejectsMissingRealMemoryExamplesCase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.json")
	raw := strings.Replace(validMemoryProductionReport(), `    {"name":"real memory examples","kind":"positive","ran":true,"pass":true},
`, "", 1)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemoryProductionReport(path)
	if err == nil {
		t.Fatalf("expected missing real memory examples case to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "real memory examples") {
		t.Fatalf("error = %v, want real memory examples rejection", err)
	}
}

func TestValidateMemoryProductionReportRejectsMissingRealMemoryExamplesAudit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.json")
	raw := strings.Replace(validMemoryProductionReport(), `    {"requirement":"real memory examples","artifact":"examples/core_memory_smoke.tetra; examples/ownership_smoke.tetra; examples/flow_unsafe_cap_mem_smoke.tetra","evidence":"checked-in memory, ownership, and unsafe cap.mem examples build and run under the memory production release gate","result":"pass"},
`, "", 1)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemoryProductionReport(path)
	if err == nil {
		t.Fatalf("expected missing real memory examples audit to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "real memory examples") {
		t.Fatalf("error = %v, want real memory examples rejection", err)
	}
}

func TestValidateMemoryProductionReportRejectsMissingHeapClosureHandleCoverageCase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.json")
	raw := strings.Replace(validMemoryProductionReport(), `    {"name":"heap closure handle coverage","kind":"positive","ran":true,"pass":true},
`, "", 1)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemoryProductionReport(path)
	if err == nil {
		t.Fatalf("expected missing heap closure handle coverage case to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "heap closure handle coverage") {
		t.Fatalf("error = %v, want heap closure handle coverage rejection", err)
	}
}

func TestValidateMemoryProductionReportRejectsMissingSliceStructBorrowEscapeCoverageCase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.json")
	raw := strings.Replace(validMemoryProductionReport(), `    {"name":"slice struct borrow escape coverage","kind":"negative","ran":true,"pass":true,"expected_error":"borrow escape"},
`, "", 1)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemoryProductionReport(path)
	if err == nil {
		t.Fatalf("expected missing slice struct borrow escape coverage case to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "slice struct borrow escape coverage") {
		t.Fatalf("error = %v, want slice struct borrow escape coverage rejection", err)
	}
}

func TestValidateMemoryProductionReportRejectsMissingCapMemUnsafeBoundaryCase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.json")
	raw := strings.Replace(validMemoryProductionReport(), `    {"name":"cap.mem unsafe boundary","kind":"negative","ran":true,"pass":true,"expected_error":"only allowed in unsafe blocks"},
`, "", 1)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemoryProductionReport(path)
	if err == nil {
		t.Fatalf("expected missing cap.mem unsafe boundary case to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "cap.mem unsafe boundary") {
		t.Fatalf("error = %v, want cap.mem unsafe boundary rejection", err)
	}
}

func TestValidateMemoryProductionReportRejectsMissingCallableMutableCaptureHeapEscapeCase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.json")
	raw := strings.Replace(validMemoryProductionReport(), `    {"name":"callable mutable capture heap escape","kind":"negative","ran":true,"pass":true,"expected_error":"heap-escaped function value captures mutable local"},
`, "", 1)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemoryProductionReport(path)
	if err == nil {
		t.Fatalf("expected missing callable mutable capture heap escape case to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "callable mutable capture heap escape") {
		t.Fatalf("error = %v, want callable mutable capture heap escape rejection", err)
	}
}

func TestValidateMemoryProductionReportRejectsMissingFunctionTypedSliceAggregateBorrowEscapeCoverageCase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.json")
	raw := strings.Replace(validMemoryProductionReport(), `    {"name":"function-typed slice aggregate borrow escape coverage","kind":"negative","ran":true,"pass":true,"expected_error":"borrow escape"},
`, "", 1)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemoryProductionReport(path)
	if err == nil {
		t.Fatalf("expected missing function-typed slice aggregate borrow escape coverage case to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "function-typed slice aggregate borrow escape coverage") {
		t.Fatalf("error = %v, want function-typed slice aggregate borrow escape coverage rejection", err)
	}
}

func TestValidateMemoryProductionReportRejectsMissingLeakResourceEvidence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.json")
	raw := validMemoryProductionReport()
	for _, row := range []string{
		`    {"name":"host resource leak and finalization checks","status":"pass","evidence":"actornet TestBrokerCloseWithoutCancelStopsServeWatcher plus compiler resource_finalization_test.go selectors prove close-without-cancel goroutine watcher cleanup and resource finalization diagnostics"},
`,
		`    {"name":"actornet broker close-without-cancel leak smoke","kind":"stress","ran":true,"pass":true},
`,
		`    {"name":"compiler resource finalization diagnostics","kind":"negative","ran":true,"pass":true,"expected_error":"resource finalization"},
`,
		`    {"requirement":"leak/resource finalization evidence","artifact":"cli/internal/actornet/broker_test.go; compiler/tests/runtime/resource_finalization_test.go; tools/cmd/memory-production-smoke","evidence":"release smoke runs actornet close-without-cancel watcher leak coverage and compiler TaskHandle/TaskGroup/Island resource finalization diagnostics for optional, enum, function-typed, branch, loop, match, join, close, and free paths","result":"pass"},
`,
	} {
		raw = strings.Replace(raw, row, "", 1)
	}
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemoryProductionReport(path)
	if err == nil {
		t.Fatalf("expected missing leak/resource evidence to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "leak") {
		t.Fatalf("error = %v, want leak/resource rejection", err)
	}
}

func TestValidateMemoryProductionReleaseManifestAcceptsFreshProvenance(t *testing.T) {
	reportDir, reportPath, manifestPath := writeMemoryProductionReleaseFixture(t, nil)
	if err := validateMemoryProductionReleaseManifest(reportPath, manifestPath, reportDir, "0123456789abcdef0123456789abcdef01234567"); err != nil {
		t.Fatalf("validateMemoryProductionReleaseManifest failed: %v", err)
	}
}

func TestValidateMemoryProductionReleaseManifestRejectsMissingRAMContractArtifacts(t *testing.T) {
	reportDir, reportPath, manifestPath := writeMemoryProductionReleaseFixture(t, func(manifest *memoryReleaseTestManifest) {
		manifest.Commands = removeMemoryReleaseTestCommand(manifest.Commands, "ram-contract-gate")
		for _, kind := range []string{
			"ram_contract_release_manifest",
			"ram_contract_report",
			"ram_memory_grade_report",
			"ram_proof_store_summary",
			"ram_validation_pipeline_coverage",
			"ram_heap_blockers",
			"ram_copy_blockers",
			"ram_contract_fuzz_oracle",
			"ram_contract_hash_manifest",
		} {
			manifest.Artifacts = removeMemoryReleaseTestArtifact(manifest.Artifacts, kind)
		}
	})
	err := validateMemoryProductionReleaseManifest(reportPath, manifestPath, reportDir, "")
	if err == nil {
		t.Fatalf("expected missing RAM contract release evidence to fail")
	}
	got := err.Error()
	if !strings.Contains(got, "ram-contract-gate") && !strings.Contains(got, "ram_contract_report") {
		t.Fatalf("error = %v, want RAM contract release evidence rejection", err)
	}
}

func TestValidateMemoryProductionReleaseManifestRejectsMismatchedCurrentGitHead(t *testing.T) {
	reportDir, reportPath, manifestPath := writeMemoryProductionReleaseFixture(t, nil)
	currentHead := "abcdefabcdefabcdefabcdefabcdefabcdefabcd"
	err := validateMemoryProductionReleaseManifest(reportPath, manifestPath, reportDir, currentHead)
	if err == nil {
		t.Fatalf("expected release manifest from stale git head to fail")
	}
	got := strings.ToLower(err.Error())
	if !strings.Contains(got, "current git head") || !strings.Contains(got, currentHead) {
		t.Fatalf("error = %v, want current git head mismatch", err)
	}
}

func TestValidateMemoryProductionReleaseManifestRejectsHashMismatchedArtifact(t *testing.T) {
	reportDir, reportPath, manifestPath := writeMemoryProductionReleaseFixture(t, nil)
	summaryPath := filepath.Join(reportDir, "memory-fuzz-tier1", "summary.json")
	if err := os.WriteFile(summaryPath, []byte(`{"schema_version":"tetra.memory-fuzz-short.summary.v1","tier":1,"status":"tampered"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemoryProductionReleaseManifest(reportPath, manifestPath, reportDir, "")
	if err == nil {
		t.Fatalf("expected hash-mismatched release artifact to fail")
	}
	if !strings.Contains(err.Error(), "sha256 mismatch") || !strings.Contains(err.Error(), "memory-fuzz-tier1/summary.json") {
		t.Fatalf("error = %v, want summary hash mismatch", err)
	}
}

func TestMemoryReleaseHashingSchemaSniffIsBounded(t *testing.T) {
	root := t.TempDir()
	reportPath := filepath.Join(root, "large-report.json")
	largePrefix := strings.Repeat("x", maxMemoryReleaseJSONSchemaSniffBytes+1024)
	raw := `{"padding":"` + largePrefix + `","schema":"too-late"}`
	if err := os.WriteFile(reportPath, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	artifact, err := hashMemoryReleaseFile(root, "large-report.json")
	if err != nil {
		t.Fatal(err)
	}
	if artifact.Schema != "" {
		t.Fatalf("artifact schema = %q, want empty schema when field is beyond bounded sniff window", artifact.Schema)
	}
}

func TestMemoryReleaseHashingKeepsEarlySchemaForLargeJSON(t *testing.T) {
	root := t.TempDir()
	reportPath := filepath.Join(root, "schema-first-large.json")
	largePayload := strings.Repeat("x", maxMemoryReleaseJSONSchemaSniffBytes+1024)
	raw := `{"schema":"schema-first","payload":"` + largePayload + `"}`
	if err := os.WriteFile(reportPath, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	artifact, err := hashMemoryReleaseFile(root, "schema-first-large.json")
	if err != nil {
		t.Fatal(err)
	}
	if artifact.Schema != "schema-first" {
		t.Fatalf("artifact schema = %q, want early schema from bounded prefix", artifact.Schema)
	}
	if artifact.Size != int64(len(raw)) {
		t.Fatalf("artifact size = %d, want %d", artifact.Size, len(raw))
	}
	sum := sha256.Sum256([]byte(raw))
	if artifact.SHA256 != "sha256:"+hex.EncodeToString(sum[:]) {
		t.Fatalf("artifact sha256 = %q, want streaming hash of whole artifact", artifact.SHA256)
	}
}

func TestMemoryReleaseHashingDoesNotFallbackWhenSchemaMayBeLater(t *testing.T) {
	root := t.TempDir()
	reportPath := filepath.Join(root, "schema-version-first-large.json")
	largePayload := strings.Repeat("x", maxMemoryReleaseJSONSchemaSniffBytes+1024)
	raw := `{"schema_version":"version-first","payload":"` + largePayload + `","schema":"schema-too-late"}`
	if err := os.WriteFile(reportPath, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	artifact, err := hashMemoryReleaseFile(root, "schema-version-first-large.json")
	if err != nil {
		t.Fatal(err)
	}
	if artifact.Schema != "" {
		t.Fatalf("artifact schema = %q, want empty schema_version fallback when schema may be beyond bounded sniff window", artifact.Schema)
	}
}

func TestMemoryReleaseHashingPreservesSchemaPrecedence(t *testing.T) {
	root := t.TempDir()
	reportPath := filepath.Join(root, "dual-schema.json")
	raw := `{"schema_version":"version-first","schema":"schema-wins"}`
	if err := os.WriteFile(reportPath, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	artifact, err := hashMemoryReleaseFile(root, "dual-schema.json")
	if err != nil {
		t.Fatal(err)
	}
	if artifact.Schema != "schema-wins" {
		t.Fatalf("artifact schema = %q, want schema field to take precedence over schema_version", artifact.Schema)
	}
}

func TestMemoryReleaseHashingFallsBackFromNullSchema(t *testing.T) {
	root := t.TempDir()
	reportPath := filepath.Join(root, "null-schema.json")
	raw := `{"schema":null,"schema_version":"version-fallback"}`
	if err := os.WriteFile(reportPath, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	artifact, err := hashMemoryReleaseFile(root, "null-schema.json")
	if err != nil {
		t.Fatal(err)
	}
	if artifact.Schema != "version-fallback" {
		t.Fatalf("artifact schema = %q, want schema_version fallback when schema is null", artifact.Schema)
	}
}

func TestMemoryReleaseHashingRejectsNonStringSchemaFallback(t *testing.T) {
	root := t.TempDir()
	reportPath := filepath.Join(root, "object-schema.json")
	raw := `{"schema_version":"version-fallback","schema":{"bad":true}}`
	if err := os.WriteFile(reportPath, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	artifact, err := hashMemoryReleaseFile(root, "object-schema.json")
	if err != nil {
		t.Fatal(err)
	}
	if artifact.Schema != "" {
		t.Fatalf("artifact schema = %q, want empty schema for non-string schema field", artifact.Schema)
	}
}

func TestMemoryReleaseHashingRejectsNonStringSchemaVersion(t *testing.T) {
	root := t.TempDir()
	reportPath := filepath.Join(root, "object-schema-version.json")
	raw := `{"schema":"schema-first","schema_version":{"bad":true}}`
	if err := os.WriteFile(reportPath, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	artifact, err := hashMemoryReleaseFile(root, "object-schema-version.json")
	if err != nil {
		t.Fatal(err)
	}
	if artifact.Schema != "" {
		t.Fatalf("artifact schema = %q, want empty schema when schema_version has non-string type", artifact.Schema)
	}
}

func TestMemoryReleaseHashingRejectsMalformedJSONTail(t *testing.T) {
	root := t.TempDir()
	reportPath := filepath.Join(root, "malformed-tail.json")
	raw := `{"schema":"looks-valid","broken":`
	if err := os.WriteFile(reportPath, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	artifact, err := hashMemoryReleaseFile(root, "malformed-tail.json")
	if err != nil {
		t.Fatal(err)
	}
	if artifact.Schema != "" {
		t.Fatalf("artifact schema = %q, want empty schema for malformed JSON tail", artifact.Schema)
	}
}

func TestMemoryReleaseHashingRejectsTrailingJunkAfterJSON(t *testing.T) {
	root := t.TempDir()
	reportPath := filepath.Join(root, "trailing-junk.json")
	raw := `{"schema":"looks-valid"}junk`
	if err := os.WriteFile(reportPath, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	artifact, err := hashMemoryReleaseFile(root, "trailing-junk.json")
	if err != nil {
		t.Fatal(err)
	}
	if artifact.Schema != "" {
		t.Fatalf("artifact schema = %q, want empty schema for trailing junk after JSON object", artifact.Schema)
	}
}

func TestValidateMemoryProductionReleaseManifestRejectsMissingGeneratorCommand(t *testing.T) {
	reportDir, reportPath, manifestPath := writeMemoryProductionReleaseFixture(t, func(manifest *memoryReleaseTestManifest) {
		for i := range manifest.Artifacts {
			if manifest.Artifacts[i].Kind == "memory_production_report" {
				manifest.Artifacts[i].Command = ""
			}
		}
	})
	err := validateMemoryProductionReleaseManifest(reportPath, manifestPath, reportDir, "")
	if err == nil {
		t.Fatalf("expected missing generator command to fail")
	}
	if !strings.Contains(err.Error(), "memory_production_report command is required") {
		t.Fatalf("error = %v, want generator command rejection", err)
	}
}

func TestValidateMemoryProductionReleaseManifestRejectsMissingIslandProofVerifier(t *testing.T) {
	reportDir, reportPath, manifestPath := writeMemoryProductionReleaseFixture(t, func(manifest *memoryReleaseTestManifest) {
		manifest.Commands = removeMemoryReleaseTestCommand(manifest.Commands, "island-proof-verifier")
		manifest.Artifacts = removeMemoryReleaseTestArtifact(manifest.Artifacts, "island_proof_verifier_report")
	})
	err := validateMemoryProductionReleaseManifest(reportPath, manifestPath, reportDir, "")
	if err == nil {
		t.Fatalf("expected missing island proof verifier artifact to fail")
	}
	if got := err.Error(); !strings.Contains(got, "island-proof-verifier") && !strings.Contains(got, "island_proof_verifier_report") {
		t.Fatalf("expected island proof verifier error, got %v", err)
	}
}

func TestValidateMemoryProductionReleaseManifestRejectsMissingHashEntry(t *testing.T) {
	reportDir, reportPath, manifestPath := writeMemoryProductionReleaseFixture(t, nil)
	writeMemoryReleaseTestHashManifest(t, reportDir, []string{
		"memory-production-linux-x64.json",
		"memory-fuzz-tier1/memory-fuzz-oracle.json",
		"memory-fuzz-tier1/summary.json",
		"memory-release-manifest.json",
	})
	err := validateMemoryProductionReleaseManifest(reportPath, manifestPath, reportDir, "")
	if err == nil {
		t.Fatalf("expected missing hash manifest entry to fail")
	}
	if !strings.Contains(err.Error(), "missing hash manifest entry for targets.json") {
		t.Fatalf("error = %v, want missing targets hash entry", err)
	}
}

func TestValidateMemoryProductionReleaseManifestRejectsInvalidIslandProofVerifier(t *testing.T) {
	reportDir, reportPath, manifestPath := writeMemoryProductionReleaseFixture(t, nil)
	proofPath := filepath.Join(reportDir, "island-proof-verifier.json")
	proof := strings.Replace(validIslandProofVerifierReport(), `"operation": "island_borrow"`, `"operation": "island_reset"`, 1)
	if err := os.WriteFile(proofPath, []byte(proof), 0o644); err != nil {
		t.Fatal(err)
	}
	writeMemoryReleaseTestHashManifest(t, reportDir, memoryReleaseTestHashPaths())

	err := validateMemoryProductionReleaseManifest(reportPath, manifestPath, reportDir, "")
	if err == nil {
		t.Fatalf("expected invalid island proof verifier artifact to fail")
	}
	if got := err.Error(); !strings.Contains(got, "island proof verifier") || !strings.Contains(got, "operation mismatch") {
		t.Fatalf("expected island proof verifier mismatch, got %v", err)
	}
}

func TestValidateMemoryProductionReleaseManifestRejectsMissingRAMMeasurementArtifact(t *testing.T) {
	reportDir, reportPath, manifestPath := writeMemoryProductionReleaseFixture(t, func(manifest *memoryReleaseTestManifest) {
		manifest.Artifacts = removeMemoryReleaseTestArtifact(manifest.Artifacts, "ram_measurement_report")
	})

	err := validateMemoryProductionReleaseManifest(reportPath, manifestPath, reportDir, "")
	if err == nil {
		t.Fatalf("expected missing RAM measurement artifact to fail")
	}
	if !strings.Contains(err.Error(), "ram_measurement_report") {
		t.Fatalf("error = %v, want missing ram_measurement_report", err)
	}
}

func TestValidateMemoryProductionReleaseManifestRejectsMalformedRAMMeasurementArtifact(t *testing.T) {
	reportDir, reportPath, manifestPath := writeMemoryProductionReleaseFixture(t, nil)
	measurementPath := filepath.Join(reportDir, "ram-measurement.json")
	raw := `{"schema":"tetra.memory.ram-measurement.v1","status":"pass","target":"linux-x64","evidence_class":"runtime_measured","method":"MemStats","snapshots":[{"name":"start"}]}`
	if err := os.WriteFile(measurementPath, []byte(raw+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeMemoryReleaseTestHashManifest(t, reportDir, memoryReleaseTestHashPaths())

	err := validateMemoryProductionReleaseManifest(reportPath, manifestPath, reportDir, "")
	if err == nil {
		t.Fatalf("expected malformed RAM measurement artifact to fail")
	}
	if got := err.Error(); !strings.Contains(got, "RAM measurement") || !strings.Contains(got, "snapshot") {
		t.Fatalf("error = %v, want RAM measurement snapshot rejection", err)
	}
}

func TestValidateMemoryProductionReleaseManifestAcceptsBlockedRAMMeasurementArtifact(t *testing.T) {
	reportDir, reportPath, manifestPath := writeMemoryProductionReleaseFixture(t, nil)
	measurementPath := filepath.Join(reportDir, "ram-measurement.json")
	raw := `{"schema":"tetra.memory.ram-measurement.v1","status":"blocked","target":"linux-x64","evidence_class":"blocked","method":"time_v","blocked_reason":"/usr/bin/time unavailable"}`
	if err := os.WriteFile(measurementPath, []byte(raw+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeMemoryReleaseTestHashManifest(t, reportDir, memoryReleaseTestHashPaths())

	if err := validateMemoryProductionReleaseManifest(reportPath, manifestPath, reportDir, ""); err != nil {
		t.Fatalf("blocked RAM measurement artifact should classify as blocked, not fail: %v", err)
	}
}

func TestValidateMemoryProductionReleaseManifestRejectsBlockedRAMMeasurementAsPass(t *testing.T) {
	reportDir, reportPath, manifestPath := writeMemoryProductionReleaseFixture(t, nil)
	measurementPath := filepath.Join(reportDir, "ram-measurement.json")
	raw := `{"schema":"tetra.memory.ram-measurement.v1","status":"pass","target":"linux-x64","evidence_class":"blocked","method":"time_v","blocked_reason":"/usr/bin/time unavailable"}`
	if err := os.WriteFile(measurementPath, []byte(raw+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeMemoryReleaseTestHashManifest(t, reportDir, memoryReleaseTestHashPaths())

	err := validateMemoryProductionReleaseManifest(reportPath, manifestPath, reportDir, "")
	if err == nil {
		t.Fatalf("expected fake pass for blocked RAM measurement to fail")
	}
	if got := err.Error(); !strings.Contains(got, "RAM measurement") || !strings.Contains(got, "evidence_class") {
		t.Fatalf("error = %v, want RAM measurement evidence_class rejection", err)
	}
}

func TestValidateMemoryProductionReleaseManifestRejectsMissingRAMMetricSamples(t *testing.T) {
	reportDir, reportPath, manifestPath := writeMemoryProductionReleaseFixture(t, nil)
	measurementPath := filepath.Join(reportDir, "ram-measurement.json")
	start := strings.Index(validRAMMeasurementReport(), `  "metric_samples": [`)
	end := strings.Index(validRAMMeasurementReport(), `  "snapshots": [`)
	if start < 0 || end < 0 || end <= start {
		t.Fatalf("valid RAM measurement fixture missing metric_samples/snapshots anchors")
	}
	raw := validRAMMeasurementReport()[:start] + `  "metric_samples": [],
` + validRAMMeasurementReport()[end:]
	if err := os.WriteFile(measurementPath, []byte(raw+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeMemoryReleaseTestHashManifest(t, reportDir, memoryReleaseTestHashPaths())

	err := validateMemoryProductionReleaseManifest(reportPath, manifestPath, reportDir, "")
	if err == nil {
		t.Fatalf("expected missing RAM metric samples to fail")
	}
	if got := err.Error(); !strings.Contains(got, "RAM measurement") || !strings.Contains(got, "metric sample") {
		t.Fatalf("error = %v, want RAM measurement metric sample rejection", err)
	}
}

func TestValidateMemoryProductionReleaseManifestRejectsMemStatsRSSMeasuredClaim(t *testing.T) {
	reportDir, reportPath, manifestPath := writeMemoryProductionReleaseFixture(t, nil)
	measurementPath := filepath.Join(reportDir, "ram-measurement.json")
	raw := strings.Replace(validRAMMeasurementReport(), `"name":"rss_current","evidence_class":"unsupported","method":"MemStats","unsupported_reason":"MemStats does not expose process RSS"`, `"name":"rss_current","evidence_class":"runtime_measured","method":"MemStats","current_bytes":2048`, 1)
	if err := os.WriteFile(measurementPath, []byte(raw+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeMemoryReleaseTestHashManifest(t, reportDir, memoryReleaseTestHashPaths())

	err := validateMemoryProductionReleaseManifest(reportPath, manifestPath, reportDir, "")
	if err == nil {
		t.Fatalf("expected MemStats RSS measured claim to fail")
	}
	if got := err.Error(); !strings.Contains(got, "rss_current") || !strings.Contains(got, "MemStats") {
		t.Fatalf("error = %v, want MemStats RSS rejection", err)
	}
}

func validMemoryProductionReport() string {
	return `{
  "schema": "tetra.memory.production.v1",
  "status": "pass",
  "target": "linux-x64",
  "host": "linux-x64",
  "runtime": "memory-linux-x64",
  "source": "examples/core_memory_smoke.tetra",
  "processes": [
    {"name":"tetra build","kind":"build","path":"/tmp/tetra","ran":true,"pass":true,"exit_code":0},
    {"name":"memory smoke app","kind":"app","path":"/tmp/memory-smoke","ran":true,"pass":true,"exit_code":0},
    {"name":"memory stress","kind":"stress","path":"tools/cmd/memory-production-smoke","ran":true,"pass":true,"exit_code":0},
    {"name":"actornet close-without-cancel leak coverage","kind":"stress","path":"go test -buildvcs=false ./cli/internal/actornet -run TestBrokerCloseWithoutCancelStopsServeWatcher -count=1","ran":true,"pass":true,"exit_code":0},
    {"name":"compiler resource finalization diagnostics","kind":"stress","path":"go test -buildvcs=false ./compiler/tests/runtime -run ^(TestTaskHandleFinalization|TestTaskGroupFinalization|TestIslandFinalization) -count=1","ran":true,"pass":true,"exit_code":0}
  ],
  "benchmarks": [
    {"name":"small heap allocation syscall reduction","kind":"allocator","metric":"estimated_os_syscalls","unit":"syscalls","evidence_class":"allocation_report_estimate","method":"allocation_report_summary","baseline_value":64,"measured_value":1,"improvement_ratio":64.0,"evidence":"allocation report schema v2 estimates 64 per_core_small_heap allocation intents inside one 64KiB chunk refill; allocation_report_estimate only, not a runtime measurement","ran":true,"pass":true}
  ],
  "contracts": [
    {"name":"allocator runtime model","status":"pass","evidence":"allocator lifecycle returns deterministic handles and failure status"},
    {"name":"allocator failure semantics","status":"pass","evidence":"linux-x64 mmap failure exits deterministically before returning an invalid pointer"},
    {"name":"ownership escape model","status":"pass","evidence":"heap, slices, structs, and closures preserve borrow/consume diagnostics"},
    {"name":"unsafe cap.mem raw memory rules","status":"pass","evidence":"raw memory helpers require unsafe and explicit cap.mem"},
    {"name":"runtime bounds diagnostics","status":"pass","evidence":"out-of-bounds memory access reports deterministic runtime diagnostic"},
    {"name":"raw pointer bounds metadata","status":"pass","evidence":"allocation_base_metadata, derived_allocation_offset, checked_external_unknown, and external_unknown raw-slice policy"},
    {"name":"host resource leak and finalization checks","status":"pass","evidence":"actornet TestBrokerCloseWithoutCancelStopsServeWatcher plus compiler resource_finalization_test.go selectors prove close-without-cancel goroutine watcher cleanup and resource finalization diagnostics"},
    {"name":"actor task transfer rules","status":"pass","evidence":"memory-bearing values cannot cross actor/task boundaries without checked transfer"}
  ],
  "cases": [
    {"name":"allocator alloc/free lifecycle","kind":"positive","ran":true,"pass":true},
    {"name":"allocator failure semantics","kind":"negative","ran":true,"pass":true,"expected_error":"allocation failure"},
    {"name":"allocator invalid size precondition","kind":"negative","ran":true,"pass":true,"expected_error":"invalid allocation size"},
    {"name":"cap.mem unsafe boundary","kind":"negative","ran":true,"pass":true,"expected_error":"only allowed in unsafe blocks"},
    {"name":"memcpy/memset capability path","kind":"positive","ran":true,"pass":true},
    {"name":"runtime bounds check","kind":"negative","ran":true,"pass":true,"expected_error":"bounds"},
    {"name":"raw ptr_add negative offset bounds","kind":"negative","ran":true,"pass":true,"expected_error":"negative ptr_add offset"},
    {"name":"raw ptr_add allocation upper bound","kind":"negative","ran":true,"pass":true,"expected_error":"allocation upper bound"},
    {"name":"raw allocation-base i32 access width","kind":"negative","ran":true,"pass":true,"expected_error":"i32 access width exceeds allocation"},
    {"name":"raw allocation-base ptr access width","kind":"negative","ran":true,"pass":true,"expected_error":"ptr access width exceeds allocation"},
    {"name":"raw slice negative length","kind":"negative","ran":true,"pass":true,"expected_error":"negative raw slice length"},
    {"name":"raw slice i32 length byte overflow","kind":"negative","ran":true,"pass":true,"expected_error":"raw slice length byte overflow"},
    {"name":"raw pointer bounds metadata report","kind":"positive","ran":true,"pass":true},
    {"name":"memcpy/memset negative length","kind":"negative","ran":true,"pass":true,"expected_error":"negative helper length"},
    {"name":"reject use-after-free","kind":"negative","ran":true,"pass":true,"expected_error":"use-after-free"},
    {"name":"reject double-free","kind":"negative","ran":true,"pass":true,"expected_error":"double-free"},
    {"name":"reject borrow escape","kind":"negative","ran":true,"pass":true,"expected_error":"borrow escape"},
    {"name":"reject aliasing violation","kind":"negative","ran":true,"pass":true,"expected_error":"alias"},
    {"name":"callable mutable capture heap escape","kind":"negative","ran":true,"pass":true,"expected_error":"heap-escaped function value captures mutable local"},
    {"name":"reject actor task transfer violation","kind":"negative","ran":true,"pass":true,"expected_error":"transfer"},
    {"name":"heap closure handle coverage","kind":"positive","ran":true,"pass":true},
    {"name":"slice struct borrow escape coverage","kind":"negative","ran":true,"pass":true,"expected_error":"borrow escape"},
    {"name":"function-typed slice aggregate borrow escape coverage","kind":"negative","ran":true,"pass":true,"expected_error":"borrow escape"},
    {"name":"actornet broker close-without-cancel leak smoke","kind":"stress","ran":true,"pass":true},
    {"name":"compiler resource finalization diagnostics","kind":"negative","ran":true,"pass":true,"expected_error":"resource finalization"},
    {"name":"real memory examples","kind":"positive","ran":true,"pass":true},
    {"name":"stress allocator reuse","kind":"stress","ran":true,"pass":true},
    {"name":"deterministic memcpy/memset fuzz","kind":"stress","ran":true,"pass":true}
  ],
  "audit": [
    {"requirement":"stable allocator/runtime memory model","artifact":"lib/core/memory.tetra; compiler/internal/actorsrt/linux_x64_emit.go; tools/cmd/memory-production-smoke","evidence":"allocator alloc/free lifecycle, allocator invalid size precondition, allocator failure semantics, and stress allocator reuse cases ran on linux-x64","result":"pass"},
    {"requirement":"ownership/borrow/consume escape model","artifact":"compiler/tests/ownership; compiler/tests/safety","evidence":"borrow escape, use-after-free, double-free, aliasing, callable heap escape, and actor/task transfer diagnostics are required memory production cases","result":"pass"},
    {"requirement":"heap, slices, structs, and closures memory coverage","artifact":"docs/spec/ownership_v1.md; compiler/tests/ownership; compiler/tests/semantics/closures_semantic_clauses_test.go","evidence":"heap closure handle coverage, callable heap escape rejection, slice struct borrow escape coverage, and function-typed slice aggregate borrow escape coverage run compiler tests for closure heap handles, nested slice/struct escapes, and conservative rejection of unsafe escapes","result":"pass"},
    {"requirement":"unsafe/cap.mem/raw memory/memcpy/memset rules","artifact":"docs/spec/unsafe.md; docs/spec/capabilities.md; lib/core/memory.tetra","evidence":"cap.mem unsafe boundary plus memcpy/memset capability path and negative helper length cases require unsafe and explicit cap.mem","result":"pass"},
    {"requirement":"runtime bounds checks and diagnostics","artifact":"docs/spec/runtime_abi.md; compiler/compiler_test.go; tools/cmd/memory-production-smoke","evidence":"slice bounds, ptr_add negative offset, allocation upper bound, i32 width, ptr width, and negative helper length diagnostics are required cases","result":"pass"},
    {"requirement":"raw pointer bounds metadata","artifact":"compiler/internal/runtimeabi/raw_pointer_bounds.go; compiler/internal/plir/plir.go; compiler/internal/allocplan/plan.go; tools/cmd/memory-production-smoke","evidence":"core.alloc_bytes allocation reports include allocation_base_metadata and external_unknown raw-slice policy; PLIR records derived_allocation_offset and checked_external_unknown raw pointer paths","result":"pass"},
    {"requirement":"stress/fuzz evidence","artifact":"tools/cmd/memory-production-smoke","evidence":"stress allocator reuse and deterministic memcpy/memset fuzz cases ran through the release-gate entrypoint","result":"pass"},
    {"requirement":"allocator benchmark evidence classification","artifact":"tools/cmd/memory-production-smoke; compiler allocation report schema v2","evidence":"small heap allocation syscall reduction benchmark is classified as allocation_report_estimate from the emitted allocation report and does not claim runtime RSS, pprof, MemStats, time_v, or strace measurement","result":"pass"},
    {"requirement":"use-after-free, double-free, borrow escape, and aliasing safety","artifact":"compiler/tests/safety; compiler/tests/ownership; compiler","evidence":"required compiler safety cases reject use-after-free, double-free, borrow escape, and inout aliasing violations","result":"pass"},
    {"requirement":"actor/task transfer safety","artifact":"compiler/tests/ownership","evidence":"TestReleaseTraceabilityLifetimeAndRaceSafetyNegativeActorTaskOwnership rejects unsafe actor/task transfer boundaries","result":"pass"},
    {"requirement":"leak/resource finalization evidence","artifact":"cli/internal/actornet/broker_test.go; compiler/tests/runtime/resource_finalization_test.go; tools/cmd/memory-production-smoke","evidence":"release smoke runs actornet close-without-cancel watcher leak coverage and compiler TaskHandle/TaskGroup/Island resource finalization diagnostics for optional, enum, function-typed, branch, loop, match, join, close, and free paths","result":"pass"},
    {"requirement":"real memory examples","artifact":"examples/core_memory_smoke.tetra; examples/ownership_smoke.tetra; examples/flow_unsafe_cap_mem_smoke.tetra","evidence":"checked-in memory, ownership, and unsafe cap.mem examples build and run under the memory production release gate","result":"pass"},
    {"requirement":"safe memory documentation","artifact":"docs/spec/runtime_abi.md; docs/spec/ownership_v1.md; docs/spec/unsafe.md; docs/user/standard_library_guide.md","evidence":"verify-docs requires the Memory Production ABI, ownership extension, unsafe boundary, and writing raw memory safely guide sections","result":"pass"},
    {"requirement":"release-gate entrypoint","artifact":"scripts/release/post_v0_4/memory-production-linux-x64-smoke.sh","evidence":"entrypoint writes memory-production-linux-x64.json and runs memory-production-smoke plus validate-memory-production","result":"pass"}
  ]
	}`
}

func validIslandProofVerifierReport() string {
	return `{
  "schema": "tetra.island.proof.v1",
  "producer": "tools/validators/islandproof/release-fixture",
  "producer_command": "go run ./tools/cmd/validate-island-proof",
  "git_head": "0123456789abcdef0123456789abcdef01234567",
  "generated_at": "2026-06-07T20:15:00Z",
  "proofs": [
    {
      "proof_id": "proof:release:island:borrow:1",
      "operation": "island_borrow",
      "proof_kind": "island_epoch",
      "subject_base_id": "alloc:release:island:0",
      "island_id": "island:release:0",
      "epoch": 1,
      "source_fact_id": "fact:release:island-proof:1",
      "claim": "island_proof_verified",
      "provenance_class": "safe_known",
      "unsafe_class": "safe",
      "validator_name": "validate-island-proof",
      "validator_status": "pass",
      "planned_storage": "ExplicitIsland",
      "actual_lowering_storage": "ExplicitIsland",
      "dominance": "entry dominates release island borrow",
      "distinct_live_islands": ["island:release:0", "island:release:1"]
    }
  ]
}` + "\n"
}

func validIslandProofMemoryReport() string {
	return `{
  "schema_version": "tetra.memory-report.v1",
  "rows": [
    {
      "program_id": "release-memory-production",
      "function_id": "island-proof-verifier-fixture",
      "site_id": "island:release:borrow:1",
      "source_fact_id": "fact:release:island-proof:1",
      "source_stage": "validation",
      "claim": "island_proof_verified",
      "claim_level": "validated",
      "provenance_class": "safe_known",
      "unsafe_class": "safe",
      "alias_state": "unique",
      "island_id": "island:release:0",
      "epoch": 1,
      "base_id": "alloc:release:island:0",
      "proof_id": "proof:release:island:borrow:1",
      "proof_kind": "island_epoch",
      "proof_subject_base_id": "alloc:release:island:0",
      "proof_operation": "island_borrow",
      "planned_storage": "ExplicitIsland",
      "actual_lowering_storage": "ExplicitIsland",
      "validator_name": "validate-island-proof",
      "validator_status": "pass",
      "cost_class": "instrumentation_only",
      "reason": "release fixture proving independent island verifier gate"
    }
  ]
}` + "\n"
}

func validIslandProofFuzzSummary() string {
	return `{
  "schema_version": "tetra.island-proof-fuzz-summary.v1",
  "status": "pass",
  "corpus": "deterministic-short",
  "total": 11,
  "rejected": 11,
  "accepted": 0,
  "cases": [
    {"name": "malformed_proof_json", "status": "rejected"},
    {"name": "stale_epoch", "status": "rejected"},
    {"name": "mismatched_island_id", "status": "rejected"},
    {"name": "wrong_base_allocation", "status": "rejected"},
    {"name": "broken_dominance", "status": "rejected"},
    {"name": "missing_proof_id", "status": "rejected"},
    {"name": "wrong_operation", "status": "rejected"},
    {"name": "unsafe_unknown_promotion", "status": "rejected"},
    {"name": "noalias_broad_proof", "status": "rejected"},
    {"name": "storage_heap_fallback", "status": "rejected"},
    {"name": "transform_lost_metadata", "status": "rejected"}
  ]
}` + "\n"
}

type memoryReleaseTestManifest struct {
	Schema       string                      `json:"schema"`
	Target       string                      `json:"target"`
	GitHead      string                      `json:"git_head"`
	GeneratedAt  string                      `json:"generated_at"`
	ReportDir    string                      `json:"report_dir"`
	HashManifest string                      `json:"hash_manifest"`
	Commands     []memoryReleaseTestCommand  `json:"commands"`
	Artifacts    []memoryReleaseTestArtifact `json:"artifacts"`
}

type memoryReleaseTestCommand struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

type memoryReleaseTestArtifact struct {
	Path    string `json:"path"`
	Kind    string `json:"kind"`
	Schema  string `json:"schema,omitempty"`
	Target  string `json:"target"`
	Command string `json:"command"`
}

type memoryReleaseTestHashManifest struct {
	Schema    string                          `json:"schema"`
	Root      string                          `json:"root"`
	Artifacts []memoryReleaseTestHashArtifact `json:"artifacts"`
}

type memoryReleaseTestHashArtifact struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
	Schema string `json:"schema,omitempty"`
}

func writeMemoryProductionReleaseFixture(t *testing.T, mutate func(*memoryReleaseTestManifest)) (string, string, string) {
	t.Helper()
	reportDir := t.TempDir()
	fuzzDir := filepath.Join(reportDir, "memory-fuzz-tier1")
	if err := os.MkdirAll(fuzzDir, 0o755); err != nil {
		t.Fatal(err)
	}
	reportPath := filepath.Join(reportDir, "memory-production-linux-x64.json")
	if err := os.WriteFile(reportPath, []byte(validMemoryProductionReport()), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reportDir, "targets.json"), []byte(`[
  {"triple":"linux-x64","status":"supported","memory_claim_level":"production/host_runtime"}
]`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fuzzDir, "memory-fuzz-oracle.json"), []byte(`{"schema_version":"tetra.memory-fuzz.oracle.v1","tier":1,"target":"linux-x64"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fuzzDir, "summary.json"), []byte(`{"schema_version":"tetra.memory-fuzz-short.summary.v1","tier":1,"status":"pass"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fuzzDir, "island-proof-fuzz-summary.json"), []byte(validIslandProofFuzzSummary()), 0o644); err != nil {
		t.Fatal(err)
	}
	writeMemoryReleaseTestRAMBundle(t, reportDir)
	if err := os.WriteFile(filepath.Join(reportDir, "island-proof-verifier.json"), []byte(validIslandProofVerifierReport()), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reportDir, "island-proof-memory-report.json"), []byte(validIslandProofMemoryReport()), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reportDir, "ram-measurement.json"), []byte(validRAMMeasurementReport()), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := memoryReleaseTestManifest{
		Schema:       "tetra.memory.release-manifest.v1",
		Target:       "linux-x64",
		GitHead:      "0123456789abcdef0123456789abcdef01234567",
		GeneratedAt:  "2026-06-07T20:15:00Z",
		ReportDir:    ".",
		HashManifest: "artifact-hashes.json",
		Commands: []memoryReleaseTestCommand{
			{Name: "memory-production-smoke", Command: "go run ./tools/cmd/memory-production-smoke --report $report_path"},
			{Name: "target-report", Command: "go run ./cli/cmd/tetra targets --format=json > $targets_path"},
			{Name: "validate-targets", Command: "go run ./tools/cmd/validate-targets --report $targets_path"},
			{Name: "memory-fuzz-short", Command: "go run ./tools/cmd/memory-fuzz-short --tier 1 --report-dir $memory_fuzz_dir"},
			{Name: "validate-memory-fuzz-oracle", Command: "go run ./tools/cmd/validate-memory-fuzz-oracle --report $memory_fuzz_dir/memory-fuzz-oracle.json --artifact-dir $memory_fuzz_dir"},
			{Name: "ram-contract-gate", Command: "bash scripts/release/post_v0_4/ram-contract-linux-x64-smoke.sh --report-dir $report_dir/ram-contract"},
			{Name: "island-proof-verifier", Command: "go run ./tools/cmd/validate-island-proof --proof $report_dir/island-proof-verifier.json --memory-report $report_dir/island-proof-memory-report.json --current-git-head 0123456789abcdef0123456789abcdef01234567 --require-same-commit"},
			{Name: "artifact-hashes-write", Command: "go run ./tools/cmd/validate-artifact-hashes --write --root $report_dir --out $report_dir/artifact-hashes.json"},
			{Name: "artifact-hashes-validate", Command: "go run ./tools/cmd/validate-artifact-hashes --manifest $report_dir/artifact-hashes.json"},
		},
		Artifacts: []memoryReleaseTestArtifact{
			{Path: "memory-production-linux-x64.json", Kind: "memory_production_report", Schema: "tetra.memory.production.v1", Target: "linux-x64", Command: "go run ./tools/cmd/memory-production-smoke --report $report_path"},
			{Path: "ram-measurement.json", Kind: "ram_measurement_report", Schema: "tetra.memory.ram-measurement.v1", Target: "linux-x64", Command: "go run ./tools/cmd/memory-production-smoke --report $report_path --ram-measurement-report $report_dir/ram-measurement.json"},
			{Path: "targets.json", Kind: "target_report", Target: "linux-x64", Command: "go run ./cli/cmd/tetra targets --format=json > $targets_path"},
			{Path: "memory-fuzz-tier1/memory-fuzz-oracle.json", Kind: "memory_fuzz_oracle_report", Schema: "tetra.memory-fuzz.oracle.v1", Target: "linux-x64", Command: "go run ./tools/cmd/memory-fuzz-short --tier 1 --report-dir $memory_fuzz_dir"},
			{Path: "memory-fuzz-tier1/summary.json", Kind: "memory_fuzz_summary", Schema: "tetra.memory-fuzz-short.summary.v1", Target: "linux-x64", Command: "go run ./tools/cmd/memory-fuzz-short --tier 1 --report-dir $memory_fuzz_dir"},
			{Path: "memory-fuzz-tier1/island-proof-fuzz-summary.json", Kind: "memory_fuzz_island_proof_summary", Schema: "tetra.island-proof-fuzz-summary.v1", Target: "linux-x64", Command: "go run ./tools/cmd/memory-fuzz-short --tier 1 --report-dir $memory_fuzz_dir"},
			{Path: "ram-contract/ram-contract-release-manifest.json", Kind: "ram_contract_release_manifest", Schema: "tetra.ram-contract.release-manifest.v1", Target: "linux-x64", Command: "bash scripts/release/post_v0_4/ram-contract-linux-x64-smoke.sh --report-dir $report_dir/ram-contract"},
			{Path: "ram-contract/ram-contract-report.json", Kind: "ram_contract_report", Schema: "tetra.ram-contract-report.v1", Target: "linux-x64", Command: "bash scripts/release/post_v0_4/ram-contract-linux-x64-smoke.sh --report-dir $report_dir/ram-contract"},
			{Path: "ram-contract/memory-grade-report.json", Kind: "ram_memory_grade_report", Schema: "tetra.memory-grade-report.v1", Target: "linux-x64", Command: "bash scripts/release/post_v0_4/ram-contract-linux-x64-smoke.sh --report-dir $report_dir/ram-contract"},
			{Path: "ram-contract/proof-store-summary.json", Kind: "ram_proof_store_summary", Schema: "tetra.proof-store-summary.v1", Target: "linux-x64", Command: "bash scripts/release/post_v0_4/ram-contract-linux-x64-smoke.sh --report-dir $report_dir/ram-contract"},
			{Path: "ram-contract/validation-pipeline-coverage.json", Kind: "ram_validation_pipeline_coverage", Schema: "tetra.validation-pipeline-coverage.v1", Target: "linux-x64", Command: "bash scripts/release/post_v0_4/ram-contract-linux-x64-smoke.sh --report-dir $report_dir/ram-contract"},
			{Path: "ram-contract/heap-blockers.json", Kind: "ram_heap_blockers", Schema: "tetra.ram-blockers.v1", Target: "linux-x64", Command: "bash scripts/release/post_v0_4/ram-contract-linux-x64-smoke.sh --report-dir $report_dir/ram-contract"},
			{Path: "ram-contract/copy-blockers.json", Kind: "ram_copy_blockers", Schema: "tetra.ram-blockers.v1", Target: "linux-x64", Command: "bash scripts/release/post_v0_4/ram-contract-linux-x64-smoke.sh --report-dir $report_dir/ram-contract"},
			{Path: "ram-contract/fuzz/ram-contract-fuzz-oracle.json", Kind: "ram_contract_fuzz_oracle", Schema: "tetra.ram-contract-fuzz-oracle.v1", Target: "linux-x64", Command: "bash scripts/release/post_v0_4/ram-contract-linux-x64-smoke.sh --report-dir $report_dir/ram-contract"},
			{Path: "ram-contract/artifact-hashes.json", Kind: "ram_contract_hash_manifest", Schema: "tetra.release-artifact-hashes.v1alpha1", Target: "linux-x64", Command: "bash scripts/release/post_v0_4/ram-contract-linux-x64-smoke.sh --report-dir $report_dir/ram-contract"},
			{Path: "island-proof-verifier.json", Kind: "island_proof_verifier_report", Schema: "tetra.island.proof.v1", Target: "linux-x64", Command: "go run ./tools/cmd/validate-island-proof --proof $report_dir/island-proof-verifier.json --memory-report $report_dir/island-proof-memory-report.json --current-git-head 0123456789abcdef0123456789abcdef01234567 --require-same-commit"},
			{Path: "island-proof-memory-report.json", Kind: "island_proof_memory_report", Schema: "tetra.memory-report.v1", Target: "linux-x64", Command: "go run ./tools/cmd/validate-island-proof --proof $report_dir/island-proof-verifier.json --memory-report $report_dir/island-proof-memory-report.json --current-git-head 0123456789abcdef0123456789abcdef01234567 --require-same-commit"},
			{Path: "artifact-hashes.json", Kind: "artifact_hash_manifest", Schema: "tetra.release-artifact-hashes.v1alpha1", Target: "linux-x64", Command: "go run ./tools/cmd/validate-artifact-hashes --write --root $report_dir --out $report_dir/artifact-hashes.json"},
		},
	}
	if mutate != nil {
		mutate(&manifest)
	}
	manifestPath := filepath.Join(reportDir, "memory-release-manifest.json")
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(manifestPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	writeMemoryReleaseTestHashManifest(t, reportDir, memoryReleaseTestHashPaths())
	return reportDir, reportPath, manifestPath
}

func writeMemoryReleaseTestRAMBundle(t *testing.T, reportDir string) {
	t.Helper()
	ramDir := filepath.Join(reportDir, "ram-contract")
	files := map[string]string{
		"ram-contract-release-manifest.json":                   `{"schema":"tetra.ram-contract.release-manifest.v1","status":"pass","target":"linux-x64","git_head":"0123456789abcdef0123456789abcdef01234567","hash_manifest":"artifact-hashes.json"}` + "\n",
		"ram-contract-report.json":                             `{"schema_version":"tetra.ram-contract-report.v1","git_head":"0123456789abcdef0123456789abcdef01234567"}` + "\n",
		"memory-grade-report.json":                             `{"schema_version":"tetra.memory-grade-report.v1","git_head":"0123456789abcdef0123456789abcdef01234567"}` + "\n",
		"proof-store-summary.json":                             `{"schema_version":"tetra.proof-store-summary.v1","git_head":"0123456789abcdef0123456789abcdef01234567"}` + "\n",
		"validation-pipeline-coverage.json":                    `{"schema_version":"tetra.validation-pipeline-coverage.v1","git_head":"0123456789abcdef0123456789abcdef01234567"}` + "\n",
		"heap-blockers.json":                                   `{"schema_version":"tetra.ram-blockers.v1","kind":"heap","git_head":"0123456789abcdef0123456789abcdef01234567"}` + "\n",
		"copy-blockers.json":                                   `{"schema_version":"tetra.ram-blockers.v1","kind":"copy","git_head":"0123456789abcdef0123456789abcdef01234567"}` + "\n",
		filepath.Join("fuzz", "ram-contract-fuzz-oracle.json"): `{"schema_version":"tetra.ram-contract-fuzz-oracle.v1","git_head":"0123456789abcdef0123456789abcdef01234567"}` + "\n",
	}
	for rel, raw := range files {
		path := filepath.Join(ramDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeMemoryReleaseTestNestedRAMHashManifest(t, ramDir)
}

func writeMemoryReleaseTestNestedRAMHashManifest(t *testing.T, ramDir string) {
	t.Helper()
	paths := []string{
		"copy-blockers.json",
		"fuzz/ram-contract-fuzz-oracle.json",
		"heap-blockers.json",
		"memory-grade-report.json",
		"proof-store-summary.json",
		"ram-contract-release-manifest.json",
		"ram-contract-report.json",
		"validation-pipeline-coverage.json",
	}
	sort.Strings(paths)
	manifest := memoryReleaseTestHashManifest{
		Schema: "tetra.release-artifact-hashes.v1alpha1",
		Root:   ".",
	}
	for _, rel := range paths {
		raw, err := os.ReadFile(filepath.Join(ramDir, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(raw)
		manifest.Artifacts = append(manifest.Artifacts, memoryReleaseTestHashArtifact{
			Path:   rel,
			SHA256: "sha256:" + hex.EncodeToString(sum[:]),
			Size:   int64(len(raw)),
			Schema: memoryReleaseTestJSONSchema(raw),
		})
	}
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(filepath.Join(ramDir, "artifact-hashes.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

func removeMemoryReleaseTestCommand(commands []memoryReleaseTestCommand, name string) []memoryReleaseTestCommand {
	out := commands[:0]
	for _, command := range commands {
		if command.Name == name {
			continue
		}
		out = append(out, command)
	}
	return out
}

func removeMemoryReleaseTestArtifact(artifacts []memoryReleaseTestArtifact, kind string) []memoryReleaseTestArtifact {
	out := artifacts[:0]
	for _, artifact := range artifacts {
		if artifact.Kind == kind {
			continue
		}
		out = append(out, artifact)
	}
	return out
}

func memoryReleaseTestHashPaths() []string {
	return []string{
		"island-proof-memory-report.json",
		"island-proof-verifier.json",
		"memory-fuzz-tier1/island-proof-fuzz-summary.json",
		"memory-fuzz-tier1/memory-fuzz-oracle.json",
		"memory-fuzz-tier1/summary.json",
		"memory-production-linux-x64.json",
		"memory-release-manifest.json",
		"ram-measurement.json",
		"ram-contract/artifact-hashes.json",
		"ram-contract/copy-blockers.json",
		"ram-contract/fuzz/ram-contract-fuzz-oracle.json",
		"ram-contract/heap-blockers.json",
		"ram-contract/memory-grade-report.json",
		"ram-contract/proof-store-summary.json",
		"ram-contract/ram-contract-release-manifest.json",
		"ram-contract/ram-contract-report.json",
		"ram-contract/validation-pipeline-coverage.json",
		"targets.json",
	}
}

func validRAMMeasurementReport() string {
	return `{
  "schema": "tetra.memory.ram-measurement.v1",
  "status": "pass",
  "target": "linux-x64",
  "evidence_class": "runtime_measured",
  "method": "MemStats",
  "tool": "tools/cmd/memory-production-smoke",
  "git_head": "0123456789abcdef0123456789abcdef01234567",
  "generated_at": "2026-06-07T20:15:00Z",
  "summary": {"heap_alloc_bytes":1536,"bytes_requested":0,"bytes_reserved":0,"bytes_copied":0,"rss_current_bytes":0,"rss_peak_bytes":0,"per_actor_domain_bytes":[]},
  "metric_samples": [
    {"name":"heap_alloc_bytes","evidence_class":"runtime_measured","method":"MemStats","current_bytes":1536,"peak_bytes":1536},
    {"name":"bytes_requested","evidence_class":"unsupported","method":"not_collected","unsupported_reason":"allocation report summary is not attached to ram-measurement.json"},
    {"name":"bytes_reserved","evidence_class":"unsupported","method":"not_collected","unsupported_reason":"allocation report summary is not attached to ram-measurement.json"},
    {"name":"bytes_copied","evidence_class":"unsupported","method":"not_collected","unsupported_reason":"copy report summary is not attached to ram-measurement.json"},
    {"name":"rss_current","evidence_class":"unsupported","method":"MemStats","unsupported_reason":"MemStats does not expose process RSS"},
    {"name":"rss_peak","evidence_class":"unsupported","method":"MemStats","unsupported_reason":"MemStats does not expose process RSS"},
    {"name":"per_actor_domain_bytes","evidence_class":"unsupported","method":"not_collected","unsupported_reason":"actor memory domain report is not attached to ram-measurement.json"}
  ],
  "snapshots": [
    {"name":"start","timestamp":"2026-06-07T20:15:01Z","alloc_bytes":1024,"total_alloc_bytes":2048,"sys_bytes":8192,"heap_alloc_bytes":1024,"heap_sys_bytes":4096,"heap_idle_bytes":1024,"heap_released_bytes":512,"num_gc":0,"gc_cpu_fraction":0},
    {"name":"end","timestamp":"2026-06-07T20:15:02Z","alloc_bytes":1536,"total_alloc_bytes":4096,"sys_bytes":12288,"heap_alloc_bytes":1536,"heap_sys_bytes":8192,"heap_idle_bytes":2048,"heap_released_bytes":1024,"num_gc":1,"gc_cpu_fraction":0.001}
  ]
}` + "\n"
}

func writeMemoryReleaseTestHashManifest(t *testing.T, root string, paths []string) {
	t.Helper()
	sort.Strings(paths)
	manifest := memoryReleaseTestHashManifest{
		Schema: "tetra.release-artifact-hashes.v1alpha1",
		Root:   ".",
	}
	for _, rel := range paths {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(raw)
		manifest.Artifacts = append(manifest.Artifacts, memoryReleaseTestHashArtifact{
			Path:   rel,
			SHA256: "sha256:" + hex.EncodeToString(sum[:]),
			Size:   int64(len(raw)),
			Schema: memoryReleaseTestJSONSchema(raw),
		})
	}
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(filepath.Join(root, "artifact-hashes.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

func memoryReleaseTestJSONSchema(raw []byte) string {
	var envelope struct {
		Schema        string `json:"schema"`
		SchemaVersion string `json:"schema_version"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return ""
	}
	if envelope.Schema != "" {
		return envelope.Schema
	}
	return envelope.SchemaVersion
}
