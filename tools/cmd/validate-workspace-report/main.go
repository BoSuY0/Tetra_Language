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

type workspaceListReport struct {
	Root          string                  `json:"root"`
	WorkspacePath string                  `json:"workspace_path"`
	Members       []workspaceMemberReport `json:"members"`
}

type workspaceCheckReport struct {
	Status        string                  `json:"status"`
	Root          string                  `json:"root"`
	WorkspacePath string                  `json:"workspace_path"`
	Members       []workspaceMemberReport `json:"members"`
}

type workspaceGraphReport struct {
	Status        string                  `json:"status"`
	Root          string                  `json:"root"`
	WorkspacePath string                  `json:"workspace_path"`
	Nodes         []workspaceMemberReport `json:"nodes"`
	Edges         []workspaceGraphEdge    `json:"edges"`
}

type workspaceMemberReport struct {
	Path         string `json:"path"`
	ResolvedPath string `json:"resolved_path,omitempty"`
	CapsulePath  string `json:"capsule_path,omitempty"`
	CapsuleID    string `json:"capsule_id,omitempty"`
	Version      string `json:"version,omitempty"`
	Status       string `json:"status"`
	Detail       string `json:"detail,omitempty"`
}

type workspaceGraphEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	ID   string `json:"id"`
}

func main() {
	var reportPath string
	var kind string
	flag.StringVar(
		&reportPath,
		"report",
		"",
		"path to tetra workspace list/check/graph --format=json output",
	)
	flag.StringVar(&kind, "kind", "", "workspace report kind: list, check, or graph")
	flag.Parse()
	if reportPath == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	if kind == "" {
		fmt.Fprintln(os.Stderr, "error: --kind is required")
		os.Exit(2)
	}
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := validateWorkspaceReport(raw, kind); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateWorkspaceReport(raw []byte, kind string) error {
	switch kind {
	case "list":
		var report workspaceListReport
		if err := decodeStrictJSON(raw, &report); err != nil {
			return fmt.Errorf("invalid workspace list JSON: %w", err)
		}
		if err := validateWorkspaceHeader(report.Root, report.WorkspacePath); err != nil {
			return err
		}
		if report.Members == nil {
			return fmt.Errorf("workspace members is required")
		}
		return validateWorkspaceMembers("member", report.Members)
	case "check":
		var report workspaceCheckReport
		if err := decodeStrictJSON(raw, &report); err != nil {
			return fmt.Errorf("invalid workspace check JSON: %w", err)
		}
		if err := validateWorkspaceHeader(report.Root, report.WorkspacePath); err != nil {
			return err
		}
		if report.Members == nil {
			return fmt.Errorf("workspace members is required")
		}
		if err := validateWorkspaceMembers("member", report.Members); err != nil {
			return err
		}
		return validateWorkspaceStatus(report.Status, report.Members)
	case "graph":
		var report workspaceGraphReport
		if err := decodeStrictJSON(raw, &report); err != nil {
			return fmt.Errorf("invalid workspace graph JSON: %w", err)
		}
		if err := validateWorkspaceHeader(report.Root, report.WorkspacePath); err != nil {
			return err
		}
		if report.Nodes == nil {
			return fmt.Errorf("workspace nodes is required")
		}
		if report.Edges == nil {
			return fmt.Errorf("workspace edges is required")
		}
		if err := validateWorkspaceMembers("node", report.Nodes); err != nil {
			return err
		}
		if err := validateWorkspaceGraphEdges(report.Nodes, report.Edges); err != nil {
			return err
		}
		return validateWorkspaceStatus(report.Status, report.Nodes)
	default:
		return fmt.Errorf("unsupported --kind %q, want list, check, or graph", kind)
	}
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

func validateWorkspaceHeader(root string, workspacePath string) error {
	if strings.TrimSpace(root) == "" {
		return fmt.Errorf("workspace root is required")
	}
	if strings.TrimSpace(workspacePath) == "" {
		return fmt.Errorf("workspace_path is required")
	}
	if filepath.Base(workspacePath) != "Tetra.workspace" {
		return fmt.Errorf("workspace_path must end with Tetra.workspace")
	}
	return nil
}

func validateWorkspaceMembers(label string, members []workspaceMemberReport) error {
	seenPath := map[string]bool{}
	seenResolvedPath := map[string]string{}
	seenCapsuleID := map[string]string{}
	for _, member := range members {
		if err := validateWorkspaceMember(label, member); err != nil {
			return err
		}
		if seenPath[member.Path] {
			return fmt.Errorf("duplicate workspace %s path %s", label, member.Path)
		}
		seenPath[member.Path] = true
		if member.ResolvedPath != "" {
			key := filepath.Clean(member.ResolvedPath)
			if prev := seenResolvedPath[key]; prev != "" {
				return fmt.Errorf(
					"duplicate workspace %s resolved_path %s (also %s)",
					label,
					member.ResolvedPath,
					prev,
				)
			}
			seenResolvedPath[key] = member.Path
		}
		if member.Status == "ok" {
			if prev := seenCapsuleID[member.CapsuleID]; prev != "" {
				return fmt.Errorf("duplicate ok capsule_id %s (also %s)", member.CapsuleID, prev)
			}
			seenCapsuleID[member.CapsuleID] = member.Path
		}
	}
	return nil
}

func validateWorkspaceMember(label string, member workspaceMemberReport) error {
	if strings.TrimSpace(member.Path) == "" {
		return fmt.Errorf("workspace %s missing path", label)
	}
	if filepath.IsAbs(member.Path) || filepath.Clean(member.Path) == "." ||
		strings.HasPrefix(filepath.ToSlash(filepath.Clean(member.Path)), "../") {
		return fmt.Errorf("workspace %s %s path must be workspace-relative", label, member.Path)
	}
	switch member.Status {
	case "ok":
		if member.ResolvedPath == "" {
			return fmt.Errorf("workspace %s %s missing resolved_path", label, member.Path)
		}
		if member.CapsulePath == "" {
			return fmt.Errorf("workspace %s %s missing capsule_path", label, member.Path)
		}
		if member.CapsuleID == "" {
			return fmt.Errorf("workspace %s %s missing capsule_id", label, member.Path)
		}
		if member.Version == "" {
			return fmt.Errorf("workspace %s %s missing version", label, member.Path)
		}
		if member.Detail != "" {
			return fmt.Errorf(
				"workspace %s %s ok status must not include detail",
				label,
				member.Path,
			)
		}
	case "missing", "invalid", "fail":
		if member.Detail == "" {
			return fmt.Errorf(
				"workspace %s %s status %s requires detail",
				label,
				member.Path,
				member.Status,
			)
		}
		if member.CapsuleID != "" || member.Version != "" {
			return fmt.Errorf(
				"workspace %s %s status %s must not include capsule_id or version",
				label,
				member.Path,
				member.Status,
			)
		}
	default:
		return fmt.Errorf(
			"workspace %s %s status = %q, want ok, missing, invalid, or fail",
			label,
			member.Path,
			member.Status,
		)
	}
	return nil
}

func validateWorkspaceStatus(status string, members []workspaceMemberReport) error {
	switch status {
	case "pass", "fail":
	default:
		return fmt.Errorf("workspace status = %q, want pass or fail", status)
	}
	hasIssue := false
	for _, member := range members {
		if member.Status != "ok" {
			hasIssue = true
			break
		}
	}
	if status == "pass" && hasIssue {
		return fmt.Errorf("workspace status = \"pass\", want fail when any member status is not ok")
	}
	if status == "fail" && !hasIssue {
		return fmt.Errorf("workspace status = \"fail\", want pass when all member statuses are ok")
	}
	return nil
}

func validateWorkspaceGraphEdges(nodes []workspaceMemberReport, edges []workspaceGraphEdge) error {
	byPath := map[string]workspaceMemberReport{}
	for _, node := range nodes {
		byPath[node.Path] = node
	}
	seen := map[string]bool{}
	for _, edge := range edges {
		if edge.From == "" || edge.To == "" || edge.ID == "" {
			return fmt.Errorf("workspace graph edge requires from, to, and id")
		}
		from, ok := byPath[edge.From]
		if !ok {
			return fmt.Errorf("edge %s -> %s references unknown from node", edge.From, edge.To)
		}
		to, ok := byPath[edge.To]
		if !ok {
			return fmt.Errorf("edge %s -> %s references unknown to node", edge.From, edge.To)
		}
		if from.Status != "ok" || to.Status != "ok" {
			return fmt.Errorf("edge %s -> %s must reference ok nodes", edge.From, edge.To)
		}
		if edge.ID != to.CapsuleID {
			return fmt.Errorf(
				"edge %s -> %s id = %q, want to node capsule_id %q",
				edge.From,
				edge.To,
				edge.ID,
				to.CapsuleID,
			)
		}
		key := edge.From + "\x00" + edge.To + "\x00" + edge.ID
		if seen[key] {
			return fmt.Errorf("duplicate edge %s -> %s (%s)", edge.From, edge.To, edge.ID)
		}
		seen[key] = true
	}
	return nil
}
