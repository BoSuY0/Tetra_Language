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

type releaseEvidenceRow struct {
	Requirement string
	Files       string
	Tests       string
	Docs        string
	Evidence    string
	Status      string
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
	auditPath := flag.String(
		"audit",
		"docs/release/v0_4/v0_4_0_completion_audit.md",
		"v0.4.0 completion audit Markdown",
	)
	expectedStatus := flag.String(
		"expected-status",
		"not-achieved",
		"expected audit status: not-achieved or achieved",
	)
	flag.Parse()

	raw, err := os.ReadFile(*auditPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "validate-v0-4-completion-audit: read audit: %v\n", err)
		os.Exit(2)
	}
	if err := validateCompletionAudit(
		raw,
		completionAuditOptions{ExpectedStatus: *expectedStatus},
	); err != nil {
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
	releaseRows, err := parseReleaseEvidenceRows(text)
	if err != nil {
		return err
	}
	if err := validateReleaseEvidenceRows(releaseRows, expectedStatus); err != nil {
		return err
	}
	if expectedStatus == "not-achieved" &&
		!hasCompletionAuditSection(text, "Missing Work Summary") {
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
	tableRows, err := logicalCompletionAuditTableRows(section, 4, "checklist")
	if err != nil {
		return nil, err
	}
	var rows []completionAuditRow
	for _, cells := range tableRows {
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

func parseReleaseEvidenceRows(text string) ([]releaseEvidenceRow, error) {
	section, err := completionAuditSection(text, "Release Evidence Matrix")
	if err != nil {
		return nil, err
	}
	tableRows, err := logicalCompletionAuditTableRows(section, 6, "release evidence matrix")
	if err != nil {
		return nil, err
	}
	var rows []releaseEvidenceRow
	for _, cells := range tableRows {
		rows = append(rows, releaseEvidenceRow{
			Requirement: cells[0],
			Files:       cells[1],
			Tests:       cells[2],
			Docs:        cells[3],
			Evidence:    cells[4],
			Status:      cells[5],
		})
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("release evidence matrix has no rows")
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
			return fmt.Errorf(
				"checklist row %q has empty required artifact or command",
				row.Requirement,
			)
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
				return fmt.Errorf(
					"achieved audit has non-passing checklist row %q with result %q",
					row.Requirement,
					row.Result,
				)
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

func validateReleaseEvidenceRows(rows []releaseEvidenceRow, expectedStatus string) error {
	for _, row := range rows {
		if row.Requirement == "" {
			return fmt.Errorf("release evidence matrix row has empty requirement")
		}
		if row.Files == "" || row.Tests == "" || row.Docs == "" || row.Evidence == "" {
			return fmt.Errorf("release evidence row %q has an empty evidence cell", row.Requirement)
		}
		if !containsEvidenceKey(row.Files, "implementation") {
			return fmt.Errorf(
				"release evidence row %q files must include implementation:",
				row.Requirement,
			)
		}
		if !containsEvidenceKey(row.Tests, "positive") {
			return fmt.Errorf(
				"release evidence row %q tests must include positive:",
				row.Requirement,
			)
		}
		if !containsEvidenceKey(row.Tests, "negative") {
			return fmt.Errorf(
				"release evidence row %q tests must include negative:",
				row.Requirement,
			)
		}
		if !containsEvidenceKey(row.Docs, "docs") {
			return fmt.Errorf("release evidence row %q docs must include docs:", row.Requirement)
		}
		if !containsEvidenceKey(row.Docs, "manifest") {
			return fmt.Errorf(
				"release evidence row %q docs must include manifest:",
				row.Requirement,
			)
		}
		if !containsEvidenceKey(row.Evidence, "report") {
			return fmt.Errorf(
				"release evidence row %q evidence must include report:",
				row.Requirement,
			)
		}
		if !containsEvidenceKey(row.Evidence, "graphify") {
			return fmt.Errorf(
				"release evidence row %q evidence must include graphify:",
				row.Requirement,
			)
		}
		if !containsEvidenceKey(row.Evidence, "ci") {
			return fmt.Errorf("release evidence row %q evidence must include ci:", row.Requirement)
		}
		classification, err := classifyCompletionAuditResult(row.Status)
		if err != nil {
			return fmt.Errorf("release evidence row %q: %w", row.Requirement, err)
		}
		if classification == "pass" && releaseEvidenceContainsBlocker(row.Evidence) {
			return fmt.Errorf(
				"release evidence row %q pass status contains blocker evidence",
				row.Requirement,
			)
		}
		if expectedStatus == "achieved" && classification != "pass" {
			return fmt.Errorf(
				"achieved audit has non-passing release evidence row %q with status %q",
				row.Requirement,
				row.Status,
			)
		}
	}
	return nil
}

func containsEvidenceKey(cell string, key string) bool {
	normalized := strings.ToLower(cell)
	return strings.Contains(normalized, strings.ToLower(key)+":")
}

func releaseEvidenceContainsBlocker(cell string) bool {
	normalized := strings.ToLower(cell)
	for _, token := range []string{
		"blocked",
		"dirty worktree",
		"pending",
		"missing",
		"failing",
		"failed",
		"stale",
	} {
		if strings.Contains(normalized, token) {
			return true
		}
	}
	return false
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

func logicalCompletionAuditTableRows(
	section string,
	wantCells int,
	tableName string,
) ([][]string, error) {
	var rows [][]string
	for _, line := range strings.Split(section, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") {
			continue
		}
		cells := splitCompletionAuditTableRow(line)
		if len(cells) != wantCells {
			return nil, fmt.Errorf(
				"%s table row has %d cells, want %d: %s",
				tableName,
				len(cells),
				wantCells,
				line,
			)
		}
		if strings.EqualFold(cells[0], "Requirement") {
			continue
		}
		if isCompletionAuditSeparatorRow(cells) {
			continue
		}
		if cells[0] == "" {
			if len(rows) == 0 {
				return nil, fmt.Errorf("%s table starts with continuation row", tableName)
			}
			appendCompletionAuditCells(rows[len(rows)-1], cells)
			continue
		}
		rows = append(rows, cells)
	}
	return rows, nil
}

func appendCompletionAuditCells(dst []string, extra []string) {
	for i := range dst {
		extraCell := strings.TrimSpace(extra[i])
		if extraCell == "" {
			continue
		}
		if dst[i] == "" {
			dst[i] = extraCell
			continue
		}
		dst[i] = dst[i] + " " + extraCell
	}
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
