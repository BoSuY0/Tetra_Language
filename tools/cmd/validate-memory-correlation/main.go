package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
)

var requiredCorrelationFields = []string{
	"requirement_id",
	"claim",
	"source_fact_id",
	"validator",
	"report_row",
	"negative_test",
	"target_level",
	"status",
}

var requiredV0RequirementIDs = map[string]bool{
	"MEM-REP-001":    true,
	"MEM-BORROW-001": true,
	"MEM-ALIAS-001":  true,
}

var requiredV1RequirementIDs = map[string]bool{
	"MEM-BORROW-002": true,
	"MEM-BORROW-003": true,
}

var requiredV2RequirementIDs = map[string]bool{
	"MEM-BORROW-004": true,
	"MEM-BORROW-005": true,
	"MEM-ALIAS-002":  true,
}

var requiredV3RequirementIDs = map[string]bool{
	"MEM-BORROW-006": true,
	"MEM-BORROW-007": true,
	"MEM-ALIAS-003":  true,
}

var requiredV4RequirementIDs = map[string]bool{
	"MEM-BORROW-008": true,
	"MEM-BORROW-009": true,
	"MEM-BORROW-010": true,
	"MEM-ALIAS-004":  true,
}

var requiredV5RequirementIDs = map[string]bool{
	"MEM-UNSAFE-001": true,
	"MEM-UNSAFE-002": true,
	"MEM-UNSAFE-003": true,
	"MEM-UNSAFE-004": true,
}

var requiredV6RequirementIDs = map[string]bool{
	"MEM-BOUNDS-001": true,
	"MEM-BOUNDS-002": true,
	"MEM-BOUNDS-003": true,
	"MEM-BOUNDS-004": true,
}

var requiredV7RequirementIDs = map[string]bool{
	"MEM-FFI-001": true,
	"MEM-FFI-002": true,
	"MEM-FFI-003": true,
	"MEM-FFI-004": true,
}

var requiredV8RequirementIDs = map[string]bool{
	"MEM-REPORT-001": true,
	"MEM-REPORT-002": true,
	"MEM-REPORT-003": true,
	"MEM-REPORT-004": true,
	"MEM-REPORT-005": true,
}

var requiredV9RequirementIDs = map[string]bool{
	"MEM-STORAGE-001": true,
	"MEM-STORAGE-002": true,
	"MEM-STORAGE-003": true,
	"MEM-STORAGE-004": true,
}

var expectedV8Statuses = map[string]string{
	"MEM-REPORT-001": "validated_narrow",
	"MEM-REPORT-002": "validated_narrow",
	"MEM-REPORT-003": "validated_narrow",
	"MEM-REPORT-004": "validated_narrow",
	"MEM-REPORT-005": "rejected",
}

var expectedV9Statuses = map[string]string{
	"MEM-STORAGE-001": "rejected",
	"MEM-STORAGE-002": "validated_narrow",
	"MEM-STORAGE-003": "validated_narrow",
	"MEM-STORAGE-004": "conservative",
}

var allowedStatuses = map[string]bool{
	"validated":         true,
	"validated_narrow":  true,
	"conservative":      true,
	"rejected":          true,
	"future":            true,
	"explicit_non_goal": true,
}

func main() {
	path := flag.String("file", "", "path to Memory Ideal Vertical Slice v0 correlation Markdown")
	flag.Parse()
	if strings.TrimSpace(*path) == "" {
		fmt.Fprintln(os.Stderr, "error: --file is required")
		os.Exit(2)
	}
	if err := validateCorrelationFile(*path); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateCorrelationFile(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	rows, err := parseCorrelationRows(string(raw))
	if err != nil {
		return err
	}
	return validateCorrelationRows(rows)
}

func parseCorrelationRows(raw string) ([]map[string]string, error) {
	var headers []string
	var rows []map[string]string
	tableStarted := false
	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "|") {
			if tableStarted && len(rows) > 0 {
				break
			}
			continue
		}
		cells := splitMarkdownRow(trimmed)
		if len(cells) == 0 {
			continue
		}
		if !tableStarted {
			normalized := normalizeCells(cells)
			if !sameStringSet(normalized, requiredCorrelationFields) {
				continue
			}
			headers = normalized
			tableStarted = true
			continue
		}
		if isMarkdownSeparatorRow(cells) {
			continue
		}
		if len(cells) != len(headers) {
			return nil, fmt.Errorf("correlation row has %d fields, want %d", len(cells), len(headers))
		}
		row := map[string]string{}
		for i, header := range headers {
			row[header] = strings.TrimSpace(cells[i])
		}
		rows = append(rows, row)
	}
	if !tableStarted {
		return nil, errors.New("correlation table with required fields was not found")
	}
	return rows, nil
}

func validateCorrelationRows(rows []map[string]string) error {
	var issues []string
	requiredRequirementIDs := requiredRequirementIDsForRows(rows)
	if len(rows) != len(requiredRequirementIDs) {
		issues = append(issues, fmt.Sprintf("correlation table has %d rows, want %d", len(rows), len(requiredRequirementIDs)))
	}
	seen := map[string]bool{}
	for index, row := range rows {
		id := strings.TrimSpace(row["requirement_id"])
		prefix := fmt.Sprintf("row %d", index)
		if id != "" {
			prefix = fmt.Sprintf("%s (%s)", prefix, id)
		}
		if id == "" {
			issues = append(issues, prefix+": requirement_id is required")
		} else if !requiredRequirementIDs[id] {
			issues = append(issues, prefix+": unexpected requirement_id")
		}
		if seen[id] {
			issues = append(issues, prefix+": duplicate requirement_id")
		}
		seen[id] = true
		for _, field := range requiredCorrelationFields {
			if strings.TrimSpace(row[field]) == "" {
				issues = append(issues, fmt.Sprintf("%s: %s is required", prefix, field))
			}
		}
		if !hasNegativeTest(row["negative_test"]) {
			issues = append(issues, prefix+": negative_test must name at least one negative test")
		}
		status := strings.TrimSpace(row["status"])
		if status != "" && !allowedStatuses[status] {
			issues = append(issues, fmt.Sprintf("%s: unknown status %q", prefix, status))
		}
		if want, ok := expectedV8Statuses[id]; ok && status != "" && status != want {
			issues = append(issues, fmt.Sprintf("%s: widened v8 status %q, want %q", prefix, status, want))
		}
		if want, ok := expectedV9Statuses[id]; ok && status != "" && status != want {
			issues = append(issues, fmt.Sprintf("%s: widened v9 status %q, want %q", prefix, status, want))
		}
		if issue := validateMemoryClaimDrift(row); issue != "" {
			issues = append(issues, prefix+": "+issue)
		}
	}
	for id := range requiredRequirementIDs {
		if !seen[id] {
			issues = append(issues, "missing requirement_id "+id)
		}
	}
	if len(issues) > 0 {
		sort.Strings(issues)
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func requiredRequirementIDsForRows(rows []map[string]string) map[string]bool {
	for _, row := range rows {
		id := strings.TrimSpace(row["requirement_id"])
		if requiredV9RequirementIDs[id] {
			return requiredV9RequirementIDs
		}
		if requiredV8RequirementIDs[id] {
			return requiredV8RequirementIDs
		}
		if requiredV7RequirementIDs[id] {
			return requiredV7RequirementIDs
		}
		if requiredV6RequirementIDs[id] {
			return requiredV6RequirementIDs
		}
		if requiredV5RequirementIDs[id] {
			return requiredV5RequirementIDs
		}
		if requiredV4RequirementIDs[id] {
			return requiredV4RequirementIDs
		}
		if requiredV3RequirementIDs[id] {
			return requiredV3RequirementIDs
		}
		if requiredV2RequirementIDs[id] {
			return requiredV2RequirementIDs
		}
		if requiredV1RequirementIDs[id] {
			return requiredV1RequirementIDs
		}
	}
	return requiredV0RequirementIDs
}

func validateMemoryClaimDrift(row map[string]string) string {
	claim := strings.ToLower(strings.TrimSpace(row["claim"]))
	if claim == "" || !containsBroadMemoryClaim(claim) {
		return ""
	}
	if isMemoryClaimProhibition(claim) {
		return ""
	}
	if strings.Contains(claim, "memory 100%") ||
		strings.Contains(claim, "memory 100 percent") ||
		strings.Contains(claim, "broad safety") ||
		strings.Contains(claim, "arbitrary external pointer safety") ||
		strings.Contains(claim, "ffi lifetime safety") {
		return "memory claim drift: broad safety claim is outside narrow Memory Ideal evidence"
	}
	return ""
}

func containsBroadMemoryClaim(claim string) bool {
	return strings.Contains(claim, "memory 100%") ||
		strings.Contains(claim, "memory 100 percent") ||
		strings.Contains(claim, "broad safety") ||
		strings.Contains(claim, "broad memory safety") ||
		strings.Contains(claim, "arbitrary external pointer safety") ||
		strings.Contains(claim, "ffi lifetime safety")
}

func isMemoryClaimProhibition(claim string) bool {
	return strings.Contains(claim, "cannot claim") ||
		strings.Contains(claim, "must not claim") ||
		strings.Contains(claim, "do not claim") ||
		strings.Contains(claim, "does not claim") ||
		strings.Contains(claim, "no memory 100%") ||
		strings.Contains(claim, "not memory 100%") ||
		strings.Contains(claim, "nonclaim")
}

func hasNegativeTest(value string) bool {
	for _, part := range strings.Split(value, ",") {
		if strings.TrimSpace(part) != "" {
			return true
		}
	}
	return false
}

func splitMarkdownRow(line string) []string {
	trimmed := strings.TrimSpace(line)
	trimmed = strings.TrimPrefix(trimmed, "|")
	trimmed = strings.TrimSuffix(trimmed, "|")
	parts := strings.Split(trimmed, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func normalizeCells(cells []string) []string {
	normalized := make([]string, len(cells))
	for i, cell := range cells {
		normalized[i] = strings.ToLower(strings.TrimSpace(cell))
	}
	return normalized
}

func isMarkdownSeparatorRow(cells []string) bool {
	if len(cells) == 0 {
		return false
	}
	for _, cell := range cells {
		trimmed := strings.TrimSpace(cell)
		if trimmed == "" {
			return false
		}
		for _, r := range trimmed {
			if r != '-' && r != ':' {
				return false
			}
		}
	}
	return true
}

func sameStringSet(got []string, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	gotSet := map[string]bool{}
	for _, value := range got {
		gotSet[value] = true
	}
	for _, value := range want {
		if !gotSet[value] {
			return false
		}
	}
	return true
}
