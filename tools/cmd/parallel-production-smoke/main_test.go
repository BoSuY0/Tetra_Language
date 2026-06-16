package main

import (
	"encoding/json"
	"strings"
	"testing"

	"tetra_language/tools/validators/parallelprod"
)

func TestBuildReportProducesValidParallelProductionEvidence(t *testing.T) {
	report := buildReport("tools/cmd/parallel-production-smoke", []parallelprod.ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "/tmp/tetra", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "parallel smoke app", Kind: "app", Path: "/tmp/parallel-smoke", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "parallel stress", Kind: "stress", Path: "/tmp/parallel-stress", Ran: true, Pass: true, ExitCode: intPtr(0)},
	}, requiredPassingCases())
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := parallelprod.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}

func TestActorBenchmarkPrepRowsAreTierZeroWithRawArtifactsAndNonClaims(t *testing.T) {
	report := buildReport("tools/cmd/parallel-production-smoke", []parallelprod.ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "/tmp/tetra", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "parallel smoke app", Kind: "app", Path: "/tmp/parallel-smoke", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "parallel stress", Kind: "stress", Path: "/tmp/parallel-stress", Ran: true, Pass: true, ExitCode: intPtr(0)},
	}, requiredPassingCases())
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Benchmarks []benchmarkPrepRow `json:"benchmarks"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"actor ping-pong benchmark prep",
		"actor fanout/fanin benchmark prep",
		"actor mailbox throughput benchmark prep",
		"actor backpressure latency benchmark prep",
		"zero_copy_move local typed mailbox benchmark prep",
	} {
		row, ok := benchmarkByName(decoded.Benchmarks, want)
		if !ok {
			t.Fatalf("benchmark prep rows missing %q: %s", want, raw)
		}
		if row.ClaimTier != "tier0_local_smoke_only" {
			t.Fatalf("%s claim_tier = %q, want tier0_local_smoke_only", want, row.ClaimTier)
		}
		if row.Ran {
			t.Fatalf("%s ran=true, want dry-run Tier 0 prep", want)
		}
		if row.ImprovementRatio != 0 {
			t.Fatalf("%s improvement_ratio = %.3f, want 0 for Tier 0 prep", want, row.ImprovementRatio)
		}
		if len(row.RawOutputArtifacts) == 0 {
			t.Fatalf("%s missing raw_output_artifacts", want)
		}
		lowerClaim := strings.ToLower(row.Claim)
		for _, forbidden := range []string{"fastest", "faster than", "superiority", "official benchmark", "c++/rust parity", "production throughput guarantee"} {
			if strings.Contains(lowerClaim, forbidden) {
				t.Fatalf("%s claim contains forbidden wording %q: %q", want, forbidden, row.Claim)
			}
		}
		if strings.Contains(want, "zero_copy_move") && strings.Contains(lowerClaim, "production runtime") {
			t.Fatalf("%s claim promotes zero_copy_move to production runtime: %q", want, row.Claim)
		}
	}
}

func TestBuildReportCarriesActorMemoryDomainEvidence(t *testing.T) {
	report := buildReport("tools/cmd/parallel-production-smoke", []parallelprod.ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "/tmp/tetra", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "parallel smoke app", Kind: "app", Path: "/tmp/parallel-smoke", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "parallel stress", Kind: "stress", Path: "/tmp/parallel-stress", Ran: true, Pass: true, ExitCode: intPtr(0)},
	}, requiredPassingCases())
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := parallelprod.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	if len(report.ActorMemoryDomains) != 2 {
		t.Fatalf("actor memory domains = %#v, want copy and owned-region domains", report.ActorMemoryDomains)
	}
	foundOwnedRegion := false
	foundByteBackpressure := false
	for _, domain := range report.ActorMemoryDomains {
		if domain.RuntimeMeasured || domain.RuntimeBlockedReason == "" {
			t.Fatalf("actor domain runtime evidence = measured:%v reason:%q, want explicit local-model block", domain.RuntimeMeasured, domain.RuntimeBlockedReason)
		}
		if len(domain.OwnedRegions) > 0 {
			foundOwnedRegion = true
		}
		if domain.Backpressure.Status == "byte_limit_reached" {
			foundByteBackpressure = true
		}
	}
	if !foundOwnedRegion || !foundByteBackpressure {
		t.Fatalf("actor memory domains missing owned-region:%v byte-backpressure:%v", foundOwnedRegion, foundByteBackpressure)
	}
}

func TestParseParallelSchedulerEvidenceRequiresActorMemoryDomainsForObjectEvidence(t *testing.T) {
	benchmarks := parallelSchedulerBenchmarks()
	domains := parallelSchedulerActorMemoryDomains()
	raw, err := json.Marshal(struct {
		Schema             string                                 `json:"schema"`
		Benchmarks         []parallelprod.BenchmarkReport         `json:"benchmarks"`
		ActorMemoryDomains []parallelprod.ActorMemoryDomainReport `json:"actor_memory_domains"`
	}{
		Schema:             "tetra.parallelrt.prototype-evidence.v1",
		Benchmarks:         benchmarks,
		ActorMemoryDomains: domains,
	})
	if err != nil {
		t.Fatal(err)
	}
	gotBenchmarks, gotDomains, err := parseParallelSchedulerEvidence(string(raw))
	if err != nil {
		t.Fatalf("parseParallelSchedulerEvidence failed: %v", err)
	}
	if len(gotBenchmarks) != len(benchmarks) || len(gotDomains) != len(domains) {
		t.Fatalf("parsed benchmarks/domains = %d/%d, want %d/%d", len(gotBenchmarks), len(gotDomains), len(benchmarks), len(domains))
	}

	raw, err = json.Marshal(struct {
		Schema     string                         `json:"schema"`
		Benchmarks []parallelprod.BenchmarkReport `json:"benchmarks"`
	}{
		Schema:     "tetra.parallelrt.prototype-evidence.v1",
		Benchmarks: benchmarks,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := parseParallelSchedulerEvidence(string(raw)); err == nil || !strings.Contains(err.Error(), "actor_memory_domains") {
		t.Fatalf("parseParallelSchedulerEvidence without actor domains error = %v, want actor_memory_domains rejection", err)
	}
}

func TestActorBenchmarkClaimGuardsRejectMissingRawArtifactsAndOverclaims(t *testing.T) {
	report := buildReport("tools/cmd/parallel-production-smoke", []parallelprod.ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "/tmp/tetra", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "parallel smoke app", Kind: "app", Path: "/tmp/parallel-smoke", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "parallel stress", Kind: "stress", Path: "/tmp/parallel-stress", Ran: true, Pass: true, ExitCode: intPtr(0)},
	}, requiredPassingCases())
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	benchmarks := decoded["benchmarks"].([]any)
	first := benchmarks[0].(map[string]any)
	first["claim"] = "Actor benchmark report proves Tetra actors are faster than Rust actors."
	raw, err = json.Marshal(decoded)
	if err != nil {
		t.Fatal(err)
	}
	err = parallelprod.ValidateReport(raw)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "actor benchmark") {
		t.Fatalf("ValidateReport accepted actor benchmark superiority claim: %v", err)
	}

	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	benchmarks = decoded["benchmarks"].([]any)
	first = benchmarks[0].(map[string]any)
	first["claim"] = "Actor benchmark prep only; no measured speed is claimed."
	first["raw_output_artifacts"] = []any{}
	raw, err = json.Marshal(decoded)
	if err != nil {
		t.Fatal(err)
	}
	err = parallelprod.ValidateReport(raw)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "raw output") {
		t.Fatalf("ValidateReport accepted missing actor benchmark raw artifacts: %v", err)
	}

	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	benchmarks = decoded["benchmarks"].([]any)
	last := benchmarks[len(benchmarks)-1].(map[string]any)
	last["raw_output_artifacts"] = []any{"reports/actor-runtime-foundation/P15/parallelrt-evidence.raw.json"}
	last["claim"] = "zero_copy_move prototype benchmark proves production runtime zero-copy."
	raw, err = json.Marshal(decoded)
	if err != nil {
		t.Fatal(err)
	}
	err = parallelprod.ValidateReport(raw)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "zero_copy_move") {
		t.Fatalf("ValidateReport accepted zero_copy_move production runtime claim: %v", err)
	}
}

func TestRequiredPassingCasesIncludeParallelEdgeCases(t *testing.T) {
	cases := requiredPassingCases()
	for _, want := range []string{
		"task group cancel wakes deadline join",
		"nested cancellation propagation",
		"task actor mailbox handoff",
		"resource double join diagnostic",
		"task group use-after-close diagnostic",
	} {
		if !hasCase(cases, want) {
			t.Fatalf("requiredPassingCases missing %q", want)
		}
	}
}

func TestRequiredPassingCasesRecordBoundedStressMetadata(t *testing.T) {
	cases := requiredPassingCases()
	want := map[string]struct {
		iterations int
		seed       string
	}{
		"actor fanout mailbox drain soak": {iterations: 512, seed: "actor-fanout-mailbox-drain-v1"},
		"many tasks stress":               {iterations: 64, seed: "task-bounded-stress-seed-17"},
		"many actor messages stress":      {iterations: 256, seed: "actors-tagged-stress-v1"},
		"cancellation storm":              {iterations: 16, seed: "parallel-cancellation-storm-v1"},
		"timeouts stress":                 {iterations: 1, seed: "deadline-aware-waits-v1"},
	}
	for name, expected := range want {
		c, ok := caseByName(cases, name)
		if !ok {
			t.Fatalf("requiredPassingCases missing stress case %q", name)
		}
		if c.Kind != "stress" {
			t.Fatalf("%s kind = %q, want stress", name, c.Kind)
		}
		if c.Iterations != expected.iterations {
			t.Fatalf("%s iterations = %d, want %d", name, c.Iterations, expected.iterations)
		}
		if c.DeterministicSeed != expected.seed {
			t.Fatalf("%s deterministic_seed = %q, want %q", name, c.DeterministicSeed, expected.seed)
		}
		if c.MaxDurationMS <= 0 {
			t.Fatalf("%s max_duration_ms = %d, want positive bound", name, c.MaxDurationMS)
		}
	}
}

func hasCase(cases []parallelprod.CaseReport, name string) bool {
	for _, c := range cases {
		if c.Name == name {
			return true
		}
	}
	return false
}

func caseByName(cases []parallelprod.CaseReport, name string) (parallelprod.CaseReport, bool) {
	for _, c := range cases {
		if c.Name == name {
			return c, true
		}
	}
	return parallelprod.CaseReport{}, false
}

type benchmarkPrepRow struct {
	Name               string   `json:"name"`
	ClaimTier          string   `json:"claim_tier"`
	Claim              string   `json:"claim"`
	RawOutputArtifacts []string `json:"raw_output_artifacts"`
	Ran                bool     `json:"ran"`
	ImprovementRatio   float64  `json:"improvement_ratio"`
}

func benchmarkByName(rows []benchmarkPrepRow, name string) (benchmarkPrepRow, bool) {
	for _, row := range rows {
		if row.Name == name {
			return row, true
		}
	}
	return benchmarkPrepRow{}, false
}
