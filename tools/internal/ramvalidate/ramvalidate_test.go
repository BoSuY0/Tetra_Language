package ramvalidate

import (
	"strings"
	"testing"
)

func TestValidateReportAllowsEliminatedCopyWithoutTrustedProof(t *testing.T) {
	rows := []Row{{
		SiteID:           "site:main:copy0",
		ValueID:          "copy0",
		Function:         "main",
		Intent:           "copy_eliminated",
		RequestedBytes:   0,
		Bounded:          true,
		Owner:            "function:main",
		Lifetime:         "function:main",
		EscapeStatus:     "no_escape",
		Placement:        "eliminated",
		ProofIDs:         nil,
		CopyReason:       "copy elided by lowering",
		ContractGrade:    "M0",
		ValidationStatus: "conservative",
	}}
	report := Report{
		SchemaVersion: ReportSchemaV1,
		Target:        "linux-x64",
		GeneratedBy:   "test",
		Rows:          rows,
		Summary:       SummarizeRows(rows),
		NonClaims: []string{
			"no Memory 100% claim",
			"not a full formal proof",
		},
	}
	if err := ValidateReport(report); err != nil {
		t.Fatalf("ValidateReport rejected eliminated copy without trusted proof: %v", err)
	}
}

func TestValidateReportAcceptsMemoryDomainMetadata(t *testing.T) {
	rows := []Row{{
		SiteID:           "site:main:heap",
		ValueID:          "heap",
		Function:         "main",
		Intent:           "heap_fallback",
		RequestedBytes:   8192,
		Bounded:          true,
		Owner:            "function:main",
		Lifetime:         "function:main",
		EscapeStatus:     "no_escape",
		Placement:        "heap_bounded",
		Blockers:         []string{"backend_conservative_heap_fallback"},
		ContractGrade:    "M4",
		ValidationStatus: "validated",
		Domain: &MemoryDomain{
			DomainID:       "domain:process",
			Kind:           "process",
			OwnerKind:      "process",
			OwnerID:        "current",
			Lifetime:       "process",
			BudgetBytes:    8192,
			RequestedBytes: 8192,
			ReservedBytes:  8192,
		},
	}}
	report := Report{
		SchemaVersion: ReportSchemaV1,
		Target:        "linux-x64",
		GeneratedBy:   "test",
		Rows:          rows,
		Summary:       SummarizeRows(rows),
		NonClaims: []string{
			"no Memory 100% claim",
			"not a full formal proof",
		},
	}
	if len(report.Summary.Domains) != 1 || report.Summary.Domains[0].DomainID != "domain:process" {
		t.Fatalf("summary domains = %+v, want process domain", report.Summary.Domains)
	}
	if err := ValidateReport(report); err != nil {
		t.Fatalf("ValidateReport rejected domain metadata: %v", err)
	}
}

func TestValidateBlockerReportRejectsHeapRowMissingActionableMetadata(t *testing.T) {
	report := BlockerReport{
		SchemaVersion: BlockerReportSchemaV1,
		Kind:          "heap",
		Target:        "linux-x64",
		GeneratedBy:   "test",
		Rows: []BlockerRow{{
			SiteID:        "site:main:heap",
			Function:      "main",
			Intent:        "heap_fallback",
			Placement:     "heap_unbounded",
			Blockers:      []string{"unknown_size"},
			ContractGrade: "M5",
		}},
		NonClaims: []string{"no Memory 100% claim"},
	}
	err := ValidateBlockerReport(report, "heap")
	if err == nil {
		t.Fatalf("expected weak heap blocker row rejection")
	}
	for _, want := range []string{"source_location_status", "severity", "suggested_fix", "safe_to_optimize"} {
		if !contains(err.Error(), want) {
			t.Fatalf("ValidateBlockerReport error = %v, want %q", err, want)
		}
	}
}

func TestValidateBlockerReportAcceptsUnavailableSourceLocationWithEvidence(t *testing.T) {
	safe := false
	report := BlockerReport{
		SchemaVersion: BlockerReportSchemaV1,
		Kind:          "heap",
		Target:        "linux-x64",
		GeneratedBy:   "test",
		Rows: []BlockerRow{{
			SiteID:               "site:generated:heap",
			Function:             "generated",
			Intent:               "heap_fallback",
			Placement:            "heap_unbounded",
			Blockers:             []string{"unknown_size"},
			ContractGrade:        "M5",
			SourceLocationStatus: "unavailable",
			Symbol:               "generated",
			Severity:             "P1",
			Reason:               "generated allocation site does not carry a user source span",
			SuggestedFix:         "add source span propagation before optimizing this allocation",
			EvidenceID:           "fact:ram:generated:heap",
			SafeToOptimize:       &safe,
		}},
		NonClaims: []string{"no Memory 100% claim"},
	}
	if err := ValidateBlockerReport(report, "heap"); err != nil {
		t.Fatalf("ValidateBlockerReport rejected unavailable source location with evidence: %v", err)
	}
}

func TestValidateBlockerReportRejectsCopyRowMissingSafetyMetadata(t *testing.T) {
	safe := false
	report := BlockerReport{
		SchemaVersion: BlockerReportSchemaV1,
		Kind:          "copy",
		Target:        "linux-x64",
		GeneratedBy:   "test",
		Rows: []BlockerRow{{
			SiteID:               "site:main:copy",
			Function:             "main",
			Intent:               "copy_heap_bounded",
			Placement:            "heap_bounded",
			CopyReason:           "copy_requires_bounded_heap_fallback",
			ContractGrade:        "M4",
			SourceLocationStatus: "available",
			File:                 "fixtures/copy.tetra",
			Line:                 9,
			Symbol:               "main",
			Severity:             "P2",
			Reason:               "copy requires bounded heap fallback",
			SuggestedFix:         "prove ownership/lifetime before replacing the copy",
			EvidenceID:           "fact:ram:main:copy",
			SafeToOptimize:       &safe,
		}},
		NonClaims: []string{"no Memory 100% claim"},
	}
	err := ValidateBlockerReport(report, "copy")
	if err == nil {
		t.Fatalf("expected copy safety metadata rejection")
	}
	for _, want := range []string{"copy_kind", "source_value", "destination_value", "safety_reason"} {
		if !contains(err.Error(), want) {
			t.Fatalf("ValidateBlockerReport error = %v, want %q", err, want)
		}
	}
}

func contains(haystack string, needle string) bool {
	return strings.Contains(haystack, needle)
}
