package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type completionAuditOptions struct {
	ExpectedStatus string
}

type completionAuditRow struct {
	Requirement string
	Artifact    string
	Evidence    string
	Result      string
}

var requiredCompletionAuditRequirements = []string{
	"Version is marked `v0.4.0`",
	"Manifest is marked `v0.4.0`",
	"Linux-x64 production scope is selected",
	"Feature registry has no required non-production gap",
	"Callable model is production",
	"Lifetime SSA is production for the selected surface",
	"Memory production core is production",
	"Parallel production core is production",
	"Compiler production core is production",
	"Standard library mirror policy is production",
	"UI metadata/runtime/native behavior is production",
	"Distributed actors are production",
	"Linux runtime is production",
	"WASM runtime execution is production",
	"Distributed EcoNet is production",
	"Windows runtime is production",
	"macOS runtime is production",
	"`v0.4.0` readiness preflight passes",
	"`v0.4.0` release gate exists",
	"`v0.4.0` security review exists",
	"Generated docs verification covers the objective",
	"Baseline tests pass",
	"Worktree is clean for release",
}

func main() {
	auditPath := flag.String("audit", "docs/release/v0_4_0_completion_audit.md", "v0.4.0 completion audit Markdown")
	expectedStatus := flag.String("expected-status", "not-achieved", "expected audit status: not-achieved or achieved")
	flag.Parse()

	raw, err := os.ReadFile(*auditPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "validate-v0-4-completion-audit: read audit: %v\n", err)
		os.Exit(2)
	}
	if err := validateCompletionAudit(raw, completionAuditOptions{ExpectedStatus: *expectedStatus}); err != nil {
		fmt.Fprintf(os.Stderr, "validate-v0-4-completion-audit: %v\n", err)
		os.Exit(1)
	}
}

func validateCompletionAudit(raw []byte, options completionAuditOptions) error {
	text := string(raw)
	status, err := parseCompletionAuditStatus(text)
	if err != nil {
		return err
	}
	expectedStatus := normalizeCompletionAuditStatus(options.ExpectedStatus)
	if expectedStatus == "" {
		expectedStatus = "not-achieved"
	}
	if status != expectedStatus {
		return fmt.Errorf("audit status = %s, want %s", status, expectedStatus)
	}

	rows, err := parseCompletionAuditRows(text)
	if err != nil {
		return err
	}
	if err := validateCompletionAuditRows(rows, expectedStatus); err != nil {
		return err
	}
	if expectedStatus == "not-achieved" && !hasCompletionAuditSection(text, "Missing Work Summary") {
		return fmt.Errorf("missing work summary is required for not-achieved audit")
	}
	return nil
}

func parseCompletionAuditStatus(text string) (string, error) {
	matches := regexp.MustCompile(`(?m)^Status:\s*(.+?)\.\s*$`).FindStringSubmatch(text)
	if len(matches) != 2 {
		return "", fmt.Errorf("missing status line")
	}
	status := normalizeCompletionAuditStatus(matches[1])
	switch status {
	case "not-achieved", "achieved":
		return status, nil
	default:
		return "", fmt.Errorf("unsupported audit status %q", matches[1])
	}
}

func parseCompletionAuditRows(text string) ([]completionAuditRow, error) {
	section, err := completionAuditSection(text, "Prompt-To-Artifact Checklist")
	if err != nil {
		return nil, err
	}
	var rows []completionAuditRow
	for _, line := range strings.Split(section, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") {
			continue
		}
		cells := splitCompletionAuditTableRow(line)
		if len(cells) != 4 {
			return nil, fmt.Errorf("checklist table row has %d cells, want 4: %s", len(cells), line)
		}
		if strings.EqualFold(cells[0], "Requirement") {
			continue
		}
		if isCompletionAuditSeparatorRow(cells) {
			continue
		}
		rows = append(rows, completionAuditRow{
			Requirement: cells[0],
			Artifact:    cells[1],
			Evidence:    cells[2],
			Result:      cells[3],
		})
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("prompt-to-artifact checklist has no rows")
	}
	return rows, nil
}

func validateCompletionAuditRows(rows []completionAuditRow, expectedStatus string) error {
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
		classification, err := classifyCompletionAuditResult(row.Result)
		if err != nil {
			return fmt.Errorf("checklist row %q: %w", row.Requirement, err)
		}
		if classification != "pass" {
			nonPassingRows++
			if expectedStatus == "achieved" {
				return fmt.Errorf("achieved audit has non-passing checklist row %q with result %q", row.Requirement, row.Result)
			}
		}
		seen[row.Requirement] = true
	}

	var missing []string
	for _, requirement := range requiredCompletionAuditRequirements {
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

func classifyCompletionAuditResult(result string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(result))
	switch {
	case strings.HasPrefix(normalized, "pass"):
		return "pass", nil
	case strings.HasPrefix(normalized, "not required"):
		return "pass", nil
	case strings.HasPrefix(normalized, "partial"):
		return "partial", nil
	case strings.HasPrefix(normalized, "pending"):
		return "partial", nil
	case strings.HasPrefix(normalized, "fail"):
		return "fail", nil
	case strings.HasPrefix(normalized, "blocked"):
		return "fail", nil
	case strings.HasPrefix(normalized, "weak"):
		return "weak", nil
	default:
		return "", fmt.Errorf("unsupported result %q", result)
	}
}

func normalizeCompletionAuditStatus(status string) string {
	status = strings.ToLower(strings.TrimSpace(status))
	status = strings.ReplaceAll(status, "_", "-")
	status = strings.ReplaceAll(status, " ", "-")
	return status
}

func hasCompletionAuditSection(text, heading string) bool {
	_, err := completionAuditSection(text, heading)
	return err == nil
}

func completionAuditSection(text, heading string) (string, error) {
	prefix := "## " + heading
	start := strings.Index(text, prefix)
	if start < 0 {
		return "", fmt.Errorf("missing %q section", heading)
	}
	bodyStart := start + len(prefix)
	next := strings.Index(text[bodyStart:], "\n## ")
	if next < 0 {
		return text[bodyStart:], nil
	}
	return text[bodyStart : bodyStart+next], nil
}

func splitCompletionAuditTableRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	parts := strings.Split(line, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func isCompletionAuditSeparatorRow(cells []string) bool {
	for _, cell := range cells {
		cell = strings.TrimSpace(cell)
		cell = strings.Trim(cell, "-: ")
		if cell != "" {
			return false
		}
	}
	return true
}
