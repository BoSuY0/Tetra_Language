package actordist

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
)

const SchemaV1 = "tetra.actors.distributed-runtime.v1"

type Report struct {
	Schema         string          `json:"schema"`
	Status         string          `json:"status"`
	Target         string          `json:"target"`
	Host           string          `json:"host"`
	Runtime        string          `json:"runtime"`
	Transport      string          `json:"transport"`
	GitHead        string          `json:"git_head"`
	ArtifactHashes string          `json:"artifact_hashes"`
	Claims         []string        `json:"claims,omitempty"`
	NonClaims      []string        `json:"nonclaims"`
	Broker         BrokerReport    `json:"broker"`
	Processes      []ProcessReport `json:"processes"`
	FrameCounts    FrameCounts     `json:"frame_counts"`
	FrameOrder     []string        `json:"frame_order"`
	Cases          []CaseReport    `json:"cases"`
}

type BrokerReport struct {
	Runtime             string `json:"runtime"`
	Transport           string `json:"transport"`
	ListenAddr          string `json:"listen_addr"`
	AcceptedConnections int64  `json:"accepted_connections"`
	RoutedFrames        int64  `json:"routed_frames"`
	DroppedFrames       int64  `json:"dropped_frames"`
	DecodeErrors        int64  `json:"decode_errors,omitempty"`
	LastError           string `json:"last_error,omitempty"`
}

type ProcessReport struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Path     string `json:"path"`
	Ran      bool   `json:"ran"`
	Pass     bool   `json:"pass"`
	ExitCode *int   `json:"exit_code,omitempty"`
}

type FrameCounts struct {
	Hello     int64 `json:"hello"`
	HelloAck  int64 `json:"hello_ack"`
	SpawnReq  int64 `json:"spawn_req"`
	SpawnAck  int64 `json:"spawn_ack"`
	SendI32   int64 `json:"send_i32"`
	SendMsg   int64 `json:"send_msg"`
	SendTyped int64 `json:"send_typed"`
	NodeDown  int64 `json:"node_down"`
}

type CaseReport struct {
	Name          string `json:"name"`
	Ran           bool   `json:"ran"`
	Pass          bool   `json:"pass"`
	ExpectedExit  int    `json:"expected_exit"`
	ActualExit    *int   `json:"actual_exit,omitempty"`
	NodeProcesses int    `json:"node_processes"`
	Error         string `json:"error,omitempty"`
}

func ValidateReport(raw []byte) error {
	var report Report
	if err := decodeStrict(raw, &report); err != nil {
		return err
	}
	var issues []string
	if report.Schema != SchemaV1 {
		issues = append(issues, fmt.Sprintf("schema is %q, want %q", report.Schema, SchemaV1))
	}
	if report.Status != "pass" {
		issues = append(issues, fmt.Sprintf("status is %q, want pass", report.Status))
	}
	if report.Target != "linux-x64" {
		issues = append(issues, fmt.Sprintf("target is %q, want linux-x64", report.Target))
	}
	if report.Host != "linux-x64" {
		issues = append(issues, fmt.Sprintf("host is %q, want linux-x64", report.Host))
	}
	if report.Runtime != "actornet" {
		issues = append(issues, fmt.Sprintf("runtime is %q, want actornet", report.Runtime))
	}
	if report.Transport != "loopback-tcp" {
		issues = append(issues, fmt.Sprintf("transport is %q, want loopback-tcp", report.Transport))
	}
	if !isHexGitHead(report.GitHead) {
		issues = append(issues, fmt.Sprintf("git_head is %q, want 40 hex characters", report.GitHead))
	}
	if report.ArtifactHashes != "artifact-hashes.json" {
		issues = append(issues, fmt.Sprintf("artifact_hashes is %q, want artifact-hashes.json", report.ArtifactHashes))
	}
	issues = append(issues, validateClaims(report.Claims)...)
	issues = append(issues, validateNonClaims(report.NonClaims)...)
	issues = append(issues, validateBroker(report.Broker)...)
	issues = append(issues, validateProcesses(report.Processes)...)
	issues = append(issues, validateFrameCounts(report.FrameCounts)...)
	issues = append(issues, validateFrameOrder(report.FrameOrder)...)
	issues = append(issues, validateCases(report.Cases)...)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func ValidateReportForCurrentHead(raw []byte, currentGitHead string) error {
	if err := ValidateReport(raw); err != nil {
		return err
	}
	currentGitHead = strings.ToLower(strings.TrimSpace(currentGitHead))
	if !isHexGitHead(currentGitHead) {
		return fmt.Errorf("current git head is %q, want 40 hex characters", currentGitHead)
	}
	var report Report
	if err := decodeStrict(raw, &report); err != nil {
		return err
	}
	if report.GitHead != currentGitHead {
		return fmt.Errorf("git_head %q does not match current git head %q", report.GitHead, currentGitHead)
	}
	return nil
}

func validateBroker(b BrokerReport) []string {
	var issues []string
	if b.Runtime != "actornet" {
		issues = append(issues, fmt.Sprintf("broker runtime is %q, want actornet", b.Runtime))
	}
	if b.Transport != "loopback-tcp" {
		issues = append(issues, fmt.Sprintf("broker transport is %q, want loopback-tcp", b.Transport))
	}
	host, _, err := net.SplitHostPort(b.ListenAddr)
	if err != nil {
		issues = append(issues, fmt.Sprintf("broker listen_addr must be loopback host:port: %v", err))
	} else if host != "127.0.0.1" && host != "localhost" && host != "::1" {
		issues = append(issues, fmt.Sprintf("broker listen_addr host is %q, want loopback", host))
	}
	if b.AcceptedConnections < 2 {
		issues = append(issues, fmt.Sprintf("broker accepted_connections = %d, want at least 2", b.AcceptedConnections))
	}
	if b.RoutedFrames < 3 {
		issues = append(issues, fmt.Sprintf("broker routed_frames = %d, want real cross-node frame routing", b.RoutedFrames))
	}
	if b.DroppedFrames < 1 {
		issues = append(issues, fmt.Sprintf("broker dropped_frames = %d, want missing-node negative evidence", b.DroppedFrames))
	}
	if b.DecodeErrors != 0 {
		issues = append(issues, fmt.Sprintf("broker decode_errors = %d, want 0", b.DecodeErrors))
	}
	if strings.TrimSpace(b.LastError) != "" {
		issues = append(issues, "broker last_error must be empty for passing evidence")
	}
	return issues
}

func isHexGitHead(value string) bool {
	if len(value) != 40 {
		return false
	}
	for _, ch := range value {
		if (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') {
			continue
		}
		return false
	}
	return true
}

func validateNonClaims(nonclaims []string) []string {
	if len(nonclaims) == 0 {
		return []string{"nonclaims must include cluster, reconnect/retry, and non-linux scope boundaries"}
	}
	joined := strings.ToLower(strings.Join(nonclaims, "\n"))
	var issues []string
	for _, required := range []string{"cluster", "reconnect", "retry", "non-linux"} {
		if !strings.Contains(joined, required) {
			issues = append(issues, fmt.Sprintf("nonclaims missing %q boundary", required))
		}
	}
	return issues
}

func validateClaims(claims []string) []string {
	var issues []string
	for _, claim := range claims {
		lower := strings.ToLower(claim)
		for _, forbidden := range []string{"cluster", "reconnect", "retry", "non-linux"} {
			if strings.Contains(lower, forbidden) {
				issues = append(issues, fmt.Sprintf("forbidden distributed actor claim %q mentions %q", claim, forbidden))
			}
		}
	}
	return issues
}

func validateProcesses(processes []ProcessReport) []string {
	var issues []string
	if len(processes) < 3 {
		issues = append(issues, fmt.Sprintf("process evidence has %d entries, want broker plus at least two node processes", len(processes)))
	}
	seenBroker := false
	nodeCount := 0
	names := map[string]bool{}
	for _, p := range processes {
		if strings.TrimSpace(p.Name) == "" {
			issues = append(issues, "process name is required")
		} else if names[p.Name] {
			issues = append(issues, fmt.Sprintf("duplicate process %s", p.Name))
		}
		names[p.Name] = true
		switch p.Kind {
		case "broker":
			seenBroker = true
		case "node":
			nodeCount++
		default:
			issues = append(issues, fmt.Sprintf("process %s kind is %q, want broker or node", p.Name, p.Kind))
		}
		if strings.TrimSpace(p.Path) == "" {
			issues = append(issues, fmt.Sprintf("process %s path is required", p.Name))
		}
		if !p.Ran {
			issues = append(issues, fmt.Sprintf("process %s did not run", p.Name))
		}
		if !p.Pass {
			issues = append(issues, fmt.Sprintf("process %s did not pass", p.Name))
		}
		if p.ExitCode == nil {
			issues = append(issues, fmt.Sprintf("process %s missing exit_code", p.Name))
		} else if *p.ExitCode != 0 {
			issues = append(issues, fmt.Sprintf("process %s exit_code = %d, want 0", p.Name, *p.ExitCode))
		}
	}
	if !seenBroker {
		issues = append(issues, "process evidence missing broker process")
	}
	if nodeCount < 2 {
		issues = append(issues, fmt.Sprintf("process evidence has %d node processes, want at least 2", nodeCount))
	}
	return issues
}

func validateFrameCounts(counts FrameCounts) []string {
	var issues []string
	required := []struct {
		name string
		got  int64
	}{
		{name: "hello frame count", got: counts.Hello},
		{name: "hello_ack frame count", got: counts.HelloAck},
		{name: "spawn_req frame count", got: counts.SpawnReq},
		{name: "spawn_ack frame count", got: counts.SpawnAck},
		{name: "send_i32 frame count", got: counts.SendI32},
		{name: "send_msg frame count", got: counts.SendMsg},
		{name: "send_typed frame count", got: counts.SendTyped},
		{name: "node_down frame count", got: counts.NodeDown},
	}
	for _, item := range required {
		if item.got < 1 {
			issues = append(issues, fmt.Sprintf("%s = %d, want at least 1", item.name, item.got))
		}
	}
	return issues
}

func validateFrameOrder(order []string) []string {
	if len(order) == 0 {
		return []string{"frame_order is required"}
	}
	var issues []string
	allowed := map[string]bool{
		"hello":      true,
		"hello_ack":  true,
		"spawn_req":  true,
		"spawn_ack":  true,
		"send_i32":   true,
		"send_msg":   true,
		"send_typed": true,
		"node_down":  true,
	}
	for i, name := range order {
		name = strings.TrimSpace(name)
		if name == "" {
			issues = append(issues, fmt.Sprintf("frame_order[%d] is empty", i))
		} else if !allowed[name] {
			issues = append(issues, fmt.Sprintf("frame_order[%d] = %q is not a known actorwire frame", i, name))
		}
	}
	required := []string{"hello", "hello_ack", "spawn_req", "spawn_ack", "send_i32", "send_msg", "send_typed", "node_down"}
	next := 0
	for _, name := range order {
		if next < len(required) && strings.TrimSpace(name) == required[next] {
			next++
		}
	}
	if next < len(required) {
		issues = append(issues, fmt.Sprintf("frame_order missing ordered loopback sequence at %q", required[next]))
	}
	return issues
}

func validateCases(cases []CaseReport) []string {
	var issues []string
	required := map[string]bool{
		"cross-node i32 send/receive":    false,
		"cross-node tagged send/receive": false,
		"cross-node typed send/receive":  false,
		"missing-node failure/status":    false,
		"task cancel/join compatibility": false,
	}
	for _, c := range cases {
		if strings.TrimSpace(c.Name) == "" {
			issues = append(issues, "case name is required")
			continue
		}
		if _, ok := required[c.Name]; ok {
			required[c.Name] = true
		}
		if !c.Ran {
			issues = append(issues, fmt.Sprintf("case %s did not run", c.Name))
		}
		if !c.Pass {
			issues = append(issues, fmt.Sprintf("case %s did not pass", c.Name))
		}
		if c.ActualExit == nil {
			issues = append(issues, fmt.Sprintf("case %s missing actual_exit", c.Name))
		} else if *c.ActualExit != c.ExpectedExit {
			issues = append(issues, fmt.Sprintf("case %s actual_exit = %d, want %d", c.Name, *c.ActualExit, c.ExpectedExit))
		}
		if c.NodeProcesses < 1 {
			issues = append(issues, fmt.Sprintf("case %s node_processes = %d, want executable node process evidence", c.Name, c.NodeProcesses))
		}
		if strings.TrimSpace(c.Error) != "" {
			issues = append(issues, fmt.Sprintf("case %s has error text", c.Name))
		}
	}
	for name, seen := range required {
		if !seen {
			issues = append(issues, fmt.Sprintf("missing required case %s", name))
		}
	}
	return issues
}

func decodeStrict(raw []byte, out any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err != nil {
			return err
		}
		return fmt.Errorf("multiple JSON values")
	}
	return nil
}
