package main

import (
	"fmt"
	"strings"
)

type ownershipAuditOptions struct {
	ExpectedStatus string
}

func validateOwnershipAudit(raw []byte, options ownershipAuditOptions) error {
	text := string(raw)
	lower := strings.ToLower(text)
	for _, phrase := range forbiddenOwnershipAuditPhrases {
		if strings.Contains(lower, phrase) {
			return fmt.Errorf("audit contains forbidden claim phrase %q", phrase)
		}
	}

	status, err := parseOwnershipAuditStatus(text)
	if err != nil {
		return err
	}
	expectedStatus := normalizeOwnershipAuditStatus(options.ExpectedStatus)
	if expectedStatus == "" {
		expectedStatus = "not-achieved"
	}
	if status != expectedStatus {
		return fmt.Errorf("audit status = %s, want %s", status, expectedStatus)
	}

	rows, err := parseOwnershipAuditRows(text)
	if err != nil {
		return err
	}
	evidenceDetails := parseOwnershipAuditEvidenceDetails(text)
	for i := range rows {
		if detail := evidenceDetails[rows[i].Requirement]; detail != "" {
			rows[i].Evidence = rows[i].Evidence + "\n" + detail
		}
	}
	if err := validateOwnershipAuditRows(rows, expectedStatus); err != nil {
		return err
	}
	if expectedStatus == "not-achieved" {
		summary, err := ownershipAuditSection(text, "Missing Work Summary")
		if err != nil {
			return fmt.Errorf("missing work summary is required for not-achieved audit")
		}
		for _, phrase := range []string{"interprocedural lifetime", "alias/provenance", "heap/global/thread"} {
			if !strings.Contains(strings.ToLower(summary), phrase) {
				return fmt.Errorf("missing work summary does not mention %q", phrase)
			}
		}
	}
	return nil
}

func validateOwnershipAuditRows(rows []ownershipAuditRow, expectedStatus string) error {
	seen := map[string]bool{}
	nonPassingRows := 0
	for _, row := range rows {
		if row.Requirement == "" {
			return fmt.Errorf("checklist row has empty requirement")
		}
		if row.Artifact == "" {
			return fmt.Errorf("checklist row %q has empty required artifact or command", row.Requirement)
		}
		if row.Evidence == "" {
			return fmt.Errorf("checklist row %q has empty current evidence", row.Requirement)
		}
		classification, err := classifyOwnershipAuditResult(row.Result)
		if err != nil {
			return fmt.Errorf("checklist row %q: %w", row.Requirement, err)
		}
		if classification != "pass" {
			nonPassingRows++
			if expectedStatus == "achieved" {
				return fmt.Errorf("achieved audit has non-passing checklist row %q with result %q", row.Requirement, row.Result)
			}
		}
		for _, phrase := range requiredOwnershipAuditEvidencePhrases[row.Requirement] {
			if !containsOwnershipAuditPhrase(row.Artifact+" "+row.Evidence, phrase) {
				return fmt.Errorf("checklist row %q must mention %q", row.Requirement, phrase)
			}
		}
		seen[row.Requirement] = true
	}

	var missing []string
	for _, requirement := range requiredOwnershipAuditRequirements {
		if !seen[requirement] {
			missing = append(missing, requirement)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required checklist requirement: %s", strings.Join(missing, "; "))
	}
	if expectedStatus == "not-achieved" && nonPassingRows == 0 {
		return fmt.Errorf("not-achieved audit must include at least one non-passing checklist row")
	}
	return nil
}

func classifyOwnershipAuditResult(result string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(result))
	switch {
	case strings.HasPrefix(normalized, "pass"):
		return "pass", nil
	case strings.HasPrefix(normalized, "partial"):
		return "partial", nil
	case strings.HasPrefix(normalized, "fail"):
		return "fail", nil
	case strings.HasPrefix(normalized, "weak"):
		return "weak", nil
	default:
		return "", fmt.Errorf("unsupported result %q", result)
	}
}

func normalizeOwnershipAuditStatus(status string) string {
	status = strings.ToLower(strings.TrimSpace(status))
	status = strings.ReplaceAll(status, "_", "-")
	status = strings.ReplaceAll(status, " ", "-")
	return status
}

func containsOwnershipAuditPhrase(text, phrase string) bool {
	return strings.Contains(normalizeOwnershipAuditPhraseText(text), normalizeOwnershipAuditPhraseText(phrase))
}

func normalizeOwnershipAuditPhraseText(text string) string {
	return strings.ToLower(strings.Join(strings.Fields(text), " "))
}
