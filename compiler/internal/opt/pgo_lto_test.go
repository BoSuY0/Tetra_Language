package opt

import (
	"strings"
	"testing"
)

func TestPGOLTOTargetCPUCoverageAuditsP17PlanList(t *testing.T) {
	report, err := PGOLTOTargetCPUCoverage()
	if err != nil {
		t.Fatalf("PGOLTOTargetCPUCoverage: %v", err)
	}
	if report.SchemaVersion != "tetra.optimizer.pgo_lto_target_cpu.v1" {
		t.Fatalf("schema = %q", report.SchemaVersion)
	}
	if !containsString(report.NonClaims, "no PGO, LTO, target-cpu, or profile flag changes safe-program semantics") {
		t.Fatalf("non-claims = %#v, want explicit safe-semantics non-claim", report.NonClaims)
	}

	want := []PGOLTOTargetCPUID{
		PGOLTOTargetCPUProfileCollectionFormat,
		PGOLTOTargetCPUPGOOptimizerInput,
		PGOLTOTargetCPUTargetCPUFeatureDetection,
		PGOLTOTargetCPULTOIncrementalModuleSummary,
		PGOLTOTargetCPUSafeSemanticsFlags,
	}
	if len(report.Rows) != len(want) {
		t.Fatalf("coverage rows = %d, want %d: %#v", len(report.Rows), len(want), report.Rows)
	}
	byID := map[PGOLTOTargetCPUID]PGOLTOTargetCPUCoverageRow{}
	for _, row := range report.Rows {
		byID[row.ID] = row
		if row.Name == "" || row.Status == "" || row.Reason == "" || row.Evidence == "" || row.Boundary == "" {
			t.Fatalf("row missing required P17.4 evidence: %#v", row)
		}
		if row.ChangesSafeSemantics {
			t.Fatalf("P17.4 row changes safe semantics: %#v", row)
		}
	}
	for _, id := range want {
		if _, ok := byID[id]; !ok {
			t.Fatalf("missing P17.4 row %s", id)
		}
	}

	profile := byID[PGOLTOTargetCPUProfileCollectionFormat]
	if profile.Status != PGOLTOTargetCPUImplementedNarrow {
		t.Fatalf("profile format row = %#v, want implemented_narrow", profile)
	}
	if profile.OptimizerInput {
		t.Fatalf("profile format row must be inert evidence, not optimizer input: %#v", profile)
	}
	for _, want := range []string{"tetra.optimizer.profile.v1", "canonical JSON", "duplicate", "negative counter", "inert"} {
		if !strings.Contains(profile.Reason+" "+profile.Evidence+" "+profile.Boundary, want) {
			t.Fatalf("profile format row missing %q: %#v", want, profile)
		}
	}
	for _, want := range []string{"schema_validation", "canonical_json", "duplicate_rejection", "negative_counter_rejection"} {
		if !containsString(profile.RequiredFacts, want) {
			t.Fatalf("profile format row missing required fact %q: %#v", want, profile)
		}
	}

	pgo := byID[PGOLTOTargetCPUPGOOptimizerInput]
	if pgo.Status != PGOLTOTargetCPUImplementedNarrow {
		t.Fatalf("pgo_optimizer_input row = %#v, want implemented_narrow", pgo)
	}
	if !pgo.OptimizerInput {
		t.Fatalf("pgo_optimizer_input row should record optimizer input evidence: %#v", pgo)
	}
	if len(pgo.MissingFacts) != 0 {
		t.Fatalf("pgo_optimizer_input row has missing facts after foundation evidence: %#v", pgo)
	}
	for _, want := range []string{"Options.ProfileInput", "profile_input_policy", "validation metadata", "translation validation", "profile-guided rewrite policy rejected", "no profile-guided rewrite"} {
		if !strings.Contains(pgo.Reason+" "+pgo.Evidence+" "+pgo.Boundary, want) {
			t.Fatalf("pgo_optimizer_input row missing %q: %#v", want, pgo)
		}
	}
	for _, want := range []string{"optimizer_profile_input_api", "pass_contract_profile_metadata", "translation_validation_for_profile_guided_decisions", "negative_safe_semantics_tests"} {
		if !containsString(pgo.RequiredFacts, want) {
			t.Fatalf("pgo_optimizer_input row missing required fact %q: %#v", want, pgo)
		}
	}

	targetCPU := byID[PGOLTOTargetCPUTargetCPUFeatureDetection]
	if targetCPU.Status != PGOLTOTargetCPUImplementedNarrow {
		t.Fatalf("target_cpu_feature_detection row = %#v, want implemented_narrow", targetCPU)
	}
	if targetCPU.OptimizerInput || targetCPU.ChangesSafeSemantics {
		t.Fatalf("target_cpu_feature_detection must not enable optimizer input or semantic change: %#v", targetCPU)
	}
	if len(targetCPU.MissingFacts) != 0 {
		t.Fatalf("target_cpu_feature_detection row has missing facts after foundation evidence: %#v", targetCPU)
	}
	for _, want := range []string{"target feature model", "portable baseline fallback", "guarded codegen contract", "negative safe-semantics", "no target-specific rewrite"} {
		if !strings.Contains(targetCPU.Reason+" "+targetCPU.Evidence+" "+targetCPU.Boundary, want) {
			t.Fatalf("target_cpu_feature_detection row missing %q: %#v", want, targetCPU)
		}
	}
	for _, want := range []string{"target_feature_model", "portable_baseline_fallback", "guarded_codegen_contract", "negative_safe_semantics_tests"} {
		if !containsString(targetCPU.RequiredFacts, want) {
			t.Fatalf("target_cpu_feature_detection row missing required fact %q: %#v", want, targetCPU)
		}
	}

	lto := byID[PGOLTOTargetCPULTOIncrementalModuleSummary]
	if lto.Status != PGOLTOTargetCPUImplementedNarrow {
		t.Fatalf("lto_incremental_module_summary row = %#v, want implemented_narrow", lto)
	}
	if lto.OptimizerInput || lto.ChangesSafeSemantics {
		t.Fatalf("lto_incremental_module_summary must not enable optimizer input or semantic change: %#v", lto)
	}
	if len(lto.MissingFacts) != 0 {
		t.Fatalf("lto_incremental_module_summary row has missing facts after foundation evidence: %#v", lto)
	}
	for _, want := range []string{"tetra.incremental.module_summary.v1", "dependency hash contract", "cross-module validation", "non-consumer boundary", "no LTO optimizer"} {
		if !strings.Contains(lto.Reason+" "+lto.Evidence+" "+lto.Boundary, want) {
			t.Fatalf("lto_incremental_module_summary row missing %q: %#v", want, lto)
		}
	}
	for _, want := range []string{"module_summary_schema", "dependency_hash_contract", "cross_module_validation_row", "incremental_cache_negative_tests", "non_consumer_boundary"} {
		if !containsString(lto.RequiredFacts, want) {
			t.Fatalf("lto_incremental_module_summary row missing required fact %q: %#v", want, lto)
		}
	}

	safe := byID[PGOLTOTargetCPUSafeSemanticsFlags]
	if safe.Status != PGOLTOTargetCPUImplementedNarrow {
		t.Fatalf("safe semantics row = %#v, want implemented_narrow guard", safe)
	}
	if safe.OptimizerInput || safe.ChangesSafeSemantics {
		t.Fatalf("safe semantics guard must not enable optimizer input or semantic change: %#v", safe)
	}
	if !containsString(safe.RequiredFacts, "validators_reject_fake_claims") {
		t.Fatalf("safe semantics row missing validators_reject_fake_claims fact: %#v", safe)
	}
	for _, want := range []string{"no public BuildOptions flag", "profile parsing is evidence-only", "no optimizer pass consumes profile", "safe-program semantics unchanged"} {
		if !strings.Contains(safe.Reason+" "+safe.Evidence+" "+safe.Boundary, want) {
			t.Fatalf("safe semantics row missing %q: %#v", want, safe)
		}
	}
}

func TestPGOLTOTargetCPUSafeSemanticsClosureProvesFinalP17Row(t *testing.T) {
	closure, err := PGOLTOTargetCPUSafeSemanticsClosure()
	if err != nil {
		t.Fatalf("PGOLTOTargetCPUSafeSemanticsClosure: %v", err)
	}
	if closure.SchemaVersion != "tetra.optimizer.pgo_lto_target_cpu.safe_semantics_closure.v1" {
		t.Fatalf("closure schema = %q", closure.SchemaVersion)
	}
	if closure.Status != PGOLTOTargetCPUImplementedNarrow {
		t.Fatalf("closure status = %q, want %q", closure.Status, PGOLTOTargetCPUImplementedNarrow)
	}
	if closure.ChangesSafeSemantics {
		t.Fatalf("closure must not change safe semantics: %#v", closure)
	}
	if closure.PublicSemanticFlagCount != 0 {
		t.Fatalf("public semantic flag count = %d, want 0", closure.PublicSemanticFlagCount)
	}
	for _, want := range []PGOLTOTargetCPUID{
		PGOLTOTargetCPUProfileCollectionFormat,
		PGOLTOTargetCPUPGOOptimizerInput,
		PGOLTOTargetCPUTargetCPUFeatureDetection,
		PGOLTOTargetCPULTOIncrementalModuleSummary,
		PGOLTOTargetCPUSafeSemanticsFlags,
	} {
		if !containsP17RowID(closure.CompletedRows, want) {
			t.Fatalf("closure missing completed row %s: %#v", want, closure.CompletedRows)
		}
	}
	for _, want := range []string{
		"public_build_options_semantic_flag_rejected",
		"profile_guided_rewrite_policy_rejected",
		"target_specific_optimization_evidence_rejected",
		"lto_codegen_consumer_rejected",
		"lto_linker_consumer_rejected",
		"coverage_validator_rejects_fake_claims",
	} {
		if !containsString(closure.RejectedUnsafeClaims, want) {
			t.Fatalf("closure missing rejected unsafe claim %q: %#v", want, closure.RejectedUnsafeClaims)
		}
	}
	for _, want := range []string{
		"compiler/internal/opt/pgo_lto.go::ValidatePGOLTOTargetCPUSafeSemanticsClosure",
		"compiler/reports_internal_test.go::TestBuildOptionsExposeNoBackendSemanticMode",
		"compiler/internal/opt/manager_test.go::TestManagerRejectsProfileGuidedRewritePolicyUntilValidationExists",
		"compiler/internal/cache/lto_summary_test.go::TestIncrementalModuleSummaryV1RecordsDependencyHashContractAndRejectsConsumers",
	} {
		if !containsString(closure.Evidence, want) {
			t.Fatalf("closure missing evidence %q: %#v", want, closure.Evidence)
		}
	}
	if !strings.Contains(closure.Boundary, "no PGO/profile/LTO/target-cpu public flag changes safe-program semantics") {
		t.Fatalf("closure boundary missing safe-semantics non-claim: %q", closure.Boundary)
	}
}

func TestPGOLTOTargetCPUSafeSemanticsClosureRejectsFakeClaims(t *testing.T) {
	report, err := PGOLTOTargetCPUCoverage()
	if err != nil {
		t.Fatalf("PGOLTOTargetCPUCoverage: %v", err)
	}
	if err := ValidatePGOLTOTargetCPUSafeSemanticsClosure(report); err != nil {
		t.Fatalf("valid P17.4 coverage rejected: %v", err)
	}

	for name, tc := range map[string]struct {
		mutate func(PGOLTOTargetCPUCoverageReport) PGOLTOTargetCPUCoverageReport
		want   string
	}{
		"semantic change": {
			mutate: func(r PGOLTOTargetCPUCoverageReport) PGOLTOTargetCPUCoverageReport {
				return mutateP17Row(r, PGOLTOTargetCPUPGOOptimizerInput, func(row *PGOLTOTargetCPUCoverageRow) {
					row.ChangesSafeSemantics = true
				})
			},
			want: "changes safe semantics",
		},
		"incomplete row": {
			mutate: func(r PGOLTOTargetCPUCoverageReport) PGOLTOTargetCPUCoverageReport {
				return mutateP17Row(r, PGOLTOTargetCPUTargetCPUFeatureDetection, func(row *PGOLTOTargetCPUCoverageRow) {
					row.Status = PGOLTOTargetCPUNotYetCovered
				})
			},
			want: "not complete",
		},
		"missing fact": {
			mutate: func(r PGOLTOTargetCPUCoverageReport) PGOLTOTargetCPUCoverageReport {
				return mutateP17Row(r, PGOLTOTargetCPULTOIncrementalModuleSummary, func(row *PGOLTOTargetCPUCoverageRow) {
					row.MissingFacts = []string{"non_consumer_boundary"}
				})
			},
			want: "missing facts",
		},
		"profile format optimizer input": {
			mutate: func(r PGOLTOTargetCPUCoverageReport) PGOLTOTargetCPUCoverageReport {
				return mutateP17Row(r, PGOLTOTargetCPUProfileCollectionFormat, func(row *PGOLTOTargetCPUCoverageRow) {
					row.OptimizerInput = true
				})
			},
			want: "profile collection format",
		},
		"lto optimizer input": {
			mutate: func(r PGOLTOTargetCPUCoverageReport) PGOLTOTargetCPUCoverageReport {
				return mutateP17Row(r, PGOLTOTargetCPULTOIncrementalModuleSummary, func(row *PGOLTOTargetCPUCoverageRow) {
					row.OptimizerInput = true
				})
			},
			want: "LTO/incremental module summary",
		},
		"safe truth fact missing": {
			mutate: func(r PGOLTOTargetCPUCoverageReport) PGOLTOTargetCPUCoverageReport {
				return mutateP17Row(r, PGOLTOTargetCPUSafeSemanticsFlags, func(row *PGOLTOTargetCPUCoverageRow) {
					row.RequiredFacts = removeString(row.RequiredFacts, "safe_program_truth_preserved")
				})
			},
			want: "safe_program_truth_preserved",
		},
	} {
		t.Run(name, func(t *testing.T) {
			err := ValidatePGOLTOTargetCPUSafeSemanticsClosure(tc.mutate(report))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ValidatePGOLTOTargetCPUSafeSemanticsClosure error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestProfileCollectionFormatV1RoundTripsAndRejectsUnsafeDrift(t *testing.T) {
	profile := ProfileCollection{
		SchemaVersion: ProfileCollectionSchemaVersion,
		ProgramHash:   "sha256:abc123",
		TargetTriple:  "linux-x64",
		Functions: []ProfileFunction{
			{
				ID:         "fn:z",
				Name:       "main",
				EntryCount: 8,
				Counters: []ProfileCounter{
					{Kind: "edge", Name: "return", Count: 1},
					{Kind: "edge", Name: "loop", Count: 5},
				},
			},
			{
				ID:         "fn:a",
				Name:       "helper",
				EntryCount: 3,
				Counters: []ProfileCounter{
					{Kind: "block", Name: "entry", Count: 3},
				},
			},
		},
	}

	encoded, err := MarshalProfileCollection(profile)
	if err != nil {
		t.Fatalf("MarshalProfileCollection: %v", err)
	}
	const wantJSON = `{"schema_version":"tetra.optimizer.profile.v1","program_hash":"sha256:abc123","target_triple":"linux-x64","functions":[{"id":"fn:a","name":"helper","entry_count":3,"counters":[{"kind":"block","name":"entry","count":3}]},{"id":"fn:z","name":"main","entry_count":8,"counters":[{"kind":"edge","name":"loop","count":5},{"kind":"edge","name":"return","count":1}]}]}`
	if string(encoded) != wantJSON {
		t.Fatalf("canonical profile JSON:\n got %s\nwant %s", string(encoded), wantJSON)
	}
	decoded, err := ParseProfileCollection(encoded)
	if err != nil {
		t.Fatalf("ParseProfileCollection: %v", err)
	}
	reencoded, err := MarshalProfileCollection(decoded)
	if err != nil {
		t.Fatalf("MarshalProfileCollection(decoded): %v", err)
	}
	if string(reencoded) != wantJSON {
		t.Fatalf("round-trip profile JSON:\n got %s\nwant %s", string(reencoded), wantJSON)
	}

	for name, raw := range map[string][]byte{
		"wrong schema":     []byte(`{"schema_version":"tetra.optimizer.profile.v2","program_hash":"sha256:abc123","target_triple":"linux-x64","functions":[{"id":"fn:a","name":"helper","entry_count":1}]}`),
		"duplicate id":     []byte(`{"schema_version":"tetra.optimizer.profile.v1","program_hash":"sha256:abc123","target_triple":"linux-x64","functions":[{"id":"fn:a","name":"helper","entry_count":1},{"id":"fn:a","name":"other","entry_count":1}]}`),
		"duplicate name":   []byte(`{"schema_version":"tetra.optimizer.profile.v1","program_hash":"sha256:abc123","target_triple":"linux-x64","functions":[{"id":"fn:a","name":"helper","entry_count":1},{"id":"fn:b","name":"helper","entry_count":1}]}`),
		"negative counter": []byte(`{"schema_version":"tetra.optimizer.profile.v1","program_hash":"sha256:abc123","target_triple":"linux-x64","functions":[{"id":"fn:a","name":"helper","entry_count":1,"counters":[{"kind":"edge","name":"loop","count":-1}]}]}`),
	} {
		if _, err := ParseProfileCollection(raw); err == nil {
			t.Fatalf("%s: ParseProfileCollection succeeded, want rejection", name)
		}
	}
}

func containsP17RowID(values []PGOLTOTargetCPUID, want PGOLTOTargetCPUID) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func mutateP17Row(report PGOLTOTargetCPUCoverageReport, id PGOLTOTargetCPUID, mutate func(*PGOLTOTargetCPUCoverageRow)) PGOLTOTargetCPUCoverageReport {
	out := PGOLTOTargetCPUCoverageReport{
		SchemaVersion: report.SchemaVersion,
		Rows:          append([]PGOLTOTargetCPUCoverageRow(nil), report.Rows...),
		NonClaims:     append([]string(nil), report.NonClaims...),
	}
	for i := range out.Rows {
		out.Rows[i].RequiredFacts = append([]string(nil), out.Rows[i].RequiredFacts...)
		out.Rows[i].MissingFacts = append([]string(nil), out.Rows[i].MissingFacts...)
		if out.Rows[i].ID == id {
			mutate(&out.Rows[i])
		}
	}
	return out
}

func removeString(values []string, remove string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value != remove {
			out = append(out, value)
		}
	}
	return out
}
