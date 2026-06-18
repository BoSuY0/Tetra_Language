package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type workspaceExecReport struct {
	WorkspaceRoot string                `json:"workspace_root"`
	Command       string                `json:"command"`
	Target        string                `json:"target,omitempty"`
	Total         int                   `json:"total"`
	Passed        int                   `json:"passed"`
	Failed        int                   `json:"failed"`
	Skipped       int                   `json:"skipped"`
	Members       []workspaceExecMember `json:"members"`
}

type workspaceExecMember struct {
	Path      string `json:"path"`
	CapsuleID string `json:"capsule_id,omitempty"`
	Status    string `json:"status"`
	Detail    string `json:"detail,omitempty"`
	ExitCode  *int   `json:"exit_code,omitempty"`
}

func main() {
	var reportPath string
	flag.StringVar(
		&reportPath,
		"report",
		"",
		"path to tetra workspace build/test --format=json output",
	)
	flag.Parse()
	if reportPath == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := validateWorkspaceExecReport(raw); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateWorkspaceExecReport(raw []byte) error {
	if err := validateWorkspaceExecReportShape(raw); err != nil {
		return fmt.Errorf("invalid workspace execution JSON: %w", err)
	}
	var report workspaceExecReport
	if err := decodeStrictJSON(raw, &report); err != nil {
		return fmt.Errorf("invalid workspace execution JSON: %w", err)
	}
	if strings.TrimSpace(report.WorkspaceRoot) == "" {
		return fmt.Errorf("workspace_root is required")
	}
	switch report.Command {
	case "build", "test":
	default:
		return fmt.Errorf("command = %q, want build or test", report.Command)
	}
	if report.Total < 0 || report.Passed < 0 || report.Failed < 0 || report.Skipped < 0 {
		return fmt.Errorf("workspace execution counts must be non-negative")
	}

	passed, failed, skipped := 0, 0, 0
	seenPath := map[string]bool{}
	seenCapsuleID := map[string]string{}
	for _, member := range report.Members {
		if err := validateWorkspaceExecMember(member); err != nil {
			return err
		}
		if seenPath[member.Path] {
			return fmt.Errorf("duplicate member path %s", member.Path)
		}
		seenPath[member.Path] = true
		if prev := seenCapsuleID[member.CapsuleID]; prev != "" {
			return fmt.Errorf("duplicate member capsule_id %s (also %s)", member.CapsuleID, prev)
		}
		seenCapsuleID[member.CapsuleID] = member.Path
		switch member.Status {
		case "pass":
			passed++
		case "fail":
			failed++
		case "skipped":
			skipped++
		}
	}
	if report.Total != len(report.Members) || report.Passed != passed || report.Failed != failed ||
		report.Skipped != skipped {
		return fmt.Errorf(
			("count mismatch: got total=%d passed=%d failed=%d skipped=%d, " +
				"computed total=%d passed=%d failed=%d skipped=%d"),
			report.Total,
			report.Passed,
			report.Failed,
			report.Skipped,
			len(report.Members),
			passed,
			failed,
			skipped,
		)
	}
	return nil
}

func validateWorkspaceExecReportShape(raw []byte) error {
	var fields map[string]json.RawMessage
	if err := decodeStrictJSON(raw, &fields); err != nil {
		return err
	}
	for _, name := range []string{"total", "passed", "failed", "skipped", "members"} {
		value, ok := fields[name]
		if !ok {
			return fmt.Errorf("%s is required", name)
		}
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return fmt.Errorf("%s must not be null", name)
		}
	}
	return nil
}

func validateWorkspaceExecMember(member workspaceExecMember) error {
	if strings.TrimSpace(member.Path) == "" {
		return fmt.Errorf("member missing path")
	}
	if isWorkspaceExecWindowsAbsPath(member.Path) || filepath.IsAbs(member.Path) ||
		filepath.Clean(member.Path) == "." ||
		strings.HasPrefix(filepath.ToSlash(filepath.Clean(member.Path)), "../") {
		return fmt.Errorf("member %s path must be workspace-relative", member.Path)
	}
	if strings.Contains(member.Path, `\`) {
		return fmt.Errorf("member %s path must use portable slash delimiters", member.Path)
	}
	if strings.TrimSpace(member.CapsuleID) == "" {
		return fmt.Errorf("member %s missing capsule_id", member.Path)
	}
	switch member.Status {
	case "pass":
		if member.ExitCode == nil {
			return fmt.Errorf("pass member %s missing exit_code", member.Path)
		}
		if *member.ExitCode != 0 {
			return fmt.Errorf(
				"pass member %s has non-zero exit_code %d",
				member.Path,
				*member.ExitCode,
			)
		}
	case "fail":
		if member.ExitCode == nil {
			return fmt.Errorf("fail member %s missing exit_code", member.Path)
		}
		if *member.ExitCode == 0 {
			return fmt.Errorf("fail member %s has zero exit_code", member.Path)
		}
	case "skipped":
		if member.ExitCode != nil {
			return fmt.Errorf("skipped member %s must not include exit_code", member.Path)
		}
	default:
		return fmt.Errorf(
			"member %s status = %q, want pass, fail, or skipped",
			member.Path,
			member.Status,
		)
	}
	return nil
}

func isWorkspaceExecWindowsAbsPath(memberPath string) bool {
	if len(memberPath) >= 3 && isASCIIAlpha(memberPath[0]) && memberPath[1] == ':' &&
		(memberPath[2] == '\\' || memberPath[2] == '/') {
		return true
	}
	return strings.HasPrefix(memberPath, `\\`)
}

func isASCIIAlpha(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}

func decodeStrictJSON(raw []byte, out any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		if err != nil {
			return err
		}
		return fmt.Errorf("multiple JSON values")
	}
	return nil
}
