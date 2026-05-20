package main

import (
	"fmt"
	"regexp"
	"strings"
)

type ownershipAuditRow struct {
	Requirement string
	Artifact    string
	Evidence    string
	Result      string
}

func parseOwnershipAuditStatus(text string) (string, error) {
	matches := regexp.MustCompile(`(?m)^Status:\s*(.+?)\.\s*$`).FindStringSubmatch(text)
	if len(matches) != 2 {
		return "", fmt.Errorf("missing status line")
	}
	status := normalizeOwnershipAuditStatus(matches[1])
	switch status {
	case "not-achieved", "achieved":
		return status, nil
	default:
		return "", fmt.Errorf("unsupported audit status %q", matches[1])
	}
}

func parseOwnershipAuditRows(text string) ([]ownershipAuditRow, error) {
	section, err := ownershipAuditSection(text, "Prompt-To-Artifact Checklist")
	if err != nil {
		return nil, err
	}
	var rows []ownershipAuditRow
	for _, line := range strings.Split(section, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") {
			continue
		}
		cells := splitOwnershipAuditTableRow(line)
		if len(cells) != 4 {
			return nil, fmt.Errorf("checklist table row has %d cells, want 4: %s", len(cells), line)
		}
		if strings.EqualFold(cells[0], "Requirement") || isOwnershipAuditSeparatorRow(cells) {
			continue
		}
		rows = append(rows, ownershipAuditRow{
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

func parseOwnershipAuditEvidenceDetails(text string) map[string]string {
	section, err := ownershipAuditSection(text, "Evidence Details")
	if err != nil {
		return nil
	}
	details := map[string]string{}
	var current string
	var body []string
	flush := func() {
		if current == "" {
			return
		}
		details[current] = strings.TrimSpace(strings.Join(body, "\n"))
	}
	for _, line := range strings.Split(section, "\n") {
		if strings.HasPrefix(line, "### ") {
			flush()
			current = strings.TrimSpace(strings.TrimPrefix(line, "### "))
			body = nil
			continue
		}
		if current != "" {
			body = append(body, line)
		}
	}
	flush()
	return details
}

func ownershipAuditSection(text, heading string) (string, error) {
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

func splitOwnershipAuditTableRow(line string) []string {
	trimmed := strings.Trim(line, "|")
	parts := strings.Split(trimmed, "|")
	cells := make([]string, 0, len(parts))
	for _, part := range parts {
		cells = append(cells, strings.TrimSpace(part))
	}
	return cells
}

func isOwnershipAuditSeparatorRow(cells []string) bool {
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
