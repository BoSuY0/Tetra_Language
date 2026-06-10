package ramvalidate

import "testing"

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
