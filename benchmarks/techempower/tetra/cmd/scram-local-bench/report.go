package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func newMatrixReport(opt options, levels []benchLevel, endpointNames []string, workerLevels []int, appBin string, benchBin string, plan buildPlan, pg postgresEvidence, baseURL string, probe []semanticCheck) matrixReport {
	now := time.Now()
	return matrixReport{
		Schema:           matrixSchema,
		Status:           "pass",
		GeneratedAt:      now.UTC().Format(time.RFC3339),
		GeneratedLocalAt: now.Local().Format(time.RFC3339),
		Command:          commandLine(),
		Environment:      detectEnvironment(),
		Git:              detectGitState(),
		Build: buildEvidence{
			AppBinary:       appBin,
			BenchBinary:     benchBin,
			Mode:            plan.Mode,
			BuildCommand:    plan.BuildCommand,
			GoBuildTrimpath: plan.GoBuildTrimpath,
			Stripped:        plan.Stripped,
		},
		Postgres: pg,
		Server: serverEvidence{
			BaseURL:      baseURL,
			Workers:      opt.Workers,
			WorkerLevels: append([]int(nil), workerLevels...),
			PoolSize:     opt.PoolSize,
		},
		SemanticProbe: probe,
		Summary:       matrixSummary{Decision: "pass"},
		Artifacts: map[string]string{
			"semantic_report": opt.SemanticReportPath,
			"matrix_report":   opt.MatrixReportPath,
			"endpoints":       strings.Join(endpointNames, ","),
			"levels":          formatLevels(levels),
			"worker_levels":   formatInts(workerLevels),
		},
		Limitations: []string{
			"local embedded PostgreSQL harness; official TechEmpower publication is not implied",
			"SCRAM-SHA-256-PLUS channel binding and SASLprep are not implemented in the Tetra PostgreSQL runtime",
			"matrix duration and concurrency are caller controlled; use --duration 30s or --duration 60s for release gates",
		},
	}
}

func commandLine() string {
	if override := strings.TrimSpace(os.Getenv("TETRA_TE_RUNNER_COMMAND")); override != "" {
		return override
	}
	args := []string{"GOWORK=off", "go", "run", "./benchmarks/techempower/tetra/cmd/scram-local-bench"}
	args = append(args, os.Args[1:]...)
	return strings.Join(args, " ")
}

func formatLevels(levels []benchLevel) string {
	parts := make([]string, 0, len(levels))
	for _, level := range levels {
		parts = append(parts, fmt.Sprintf("%d:%d", level.Concurrency, level.Connections))
	}
	return strings.Join(parts, ",")
}

func formatInts(values []int) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, strconv.Itoa(value))
	}
	return strings.Join(parts, ",")
}

func summarizeMatrix(runs []dbRunReport) matrixSummary {
	summary := matrixSummary{RunCount: len(runs), Decision: "pass"}
	for _, run := range runs {
		summary.TotalRequests += run.Requests
		summary.TotalFailures += run.Failures
		if run.RPS > summary.BestRPS {
			summary.BestRPS = run.RPS
		}
		if run.P99LatencyMS > summary.WorstP99MS {
			summary.WorstP99MS = run.P99LatencyMS
		}
		if run.P999LatencyMS > summary.WorstP999MS {
			summary.WorstP999MS = run.P999LatencyMS
		}
		if run.Failures != 0 || run.Successes == 0 || strings.TrimSpace(run.Error) != "" {
			summary.Decision = "fail"
		}
	}
	if len(runs) == 0 {
		summary.Decision = "fail"
	}
	return summary
}

func validateMatrixReport(report matrixReport) error {
	var issues []string
	if report.Schema != matrixSchema {
		issues = append(issues, "invalid matrix schema")
	}
	if report.GeneratedAt == "" || report.GeneratedLocalAt == "" || report.Command == "" {
		issues = append(issues, "report integrity metadata is required")
	}
	if report.Postgres.AuthMethod != "scram-sha-256" || report.Postgres.PasswordEncryption != "scram-sha-256" || report.Postgres.VerifierPrefix != "SCRAM-SHA-256" {
		issues = append(issues, "PostgreSQL SCRAM evidence is incomplete")
	}
	if report.Resource.Start.RSSKB <= 0 || report.Resource.End.RSSKB <= 0 {
		issues = append(issues, "resource start/end RSS evidence is required")
	}
	if semanticFailed(report.SemanticProbe) {
		issues = append(issues, "semantic probe failed")
	}
	issues = append(issues, validateSemanticProbeCoverage(report.SemanticProbe)...)
	if report.Soak != nil {
		if report.Soak.Requests <= 0 || report.Soak.Successes <= 0 || report.Soak.Failures != 0 || !report.Soak.ShutdownClean {
			issues = append(issues, "soak evidence did not pass")
		}
		if report.Soak.ResourceStart.RSSKB <= 0 || report.Soak.ResourceEnd.RSSKB <= 0 {
			issues = append(issues, "soak resource evidence is incomplete")
		}
	}
	if report.Summary.Decision != "pass" {
		issues = append(issues, "matrix summary did not pass")
	}
	for _, run := range report.Runs {
		if strings.TrimSpace(run.Endpoint) == "" || strings.TrimSpace(run.Path) == "" || strings.TrimSpace(run.Kind) == "" || run.Workers <= 0 {
			issues = append(issues, "run endpoint/worker metadata is incomplete")
		}
		if run.Requests <= 0 || run.Successes <= 0 || run.Failures != 0 || run.RPS <= 0 {
			issues = append(issues, fmt.Sprintf("invalid %s run at workers=%d c%d/k%d repeat %d", run.Endpoint, run.Workers, run.Level.Concurrency, run.Level.Connections, run.Repeat))
		}
		if run.P99LatencyMS > run.MaxLatencyMS && run.MaxLatencyMS > 0 {
			issues = append(issues, fmt.Sprintf("p99 exceeds max at workers=%d c%d/k%d repeat %d", run.Workers, run.Level.Concurrency, run.Level.Connections, run.Repeat))
		}
		if run.P999LatencyMS > run.MaxLatencyMS && run.MaxLatencyMS > 0 {
			issues = append(issues, fmt.Sprintf("p999 exceeds max at workers=%d c%d/k%d repeat %d", run.Workers, run.Level.Concurrency, run.Level.Connections, run.Repeat))
		}
		if run.Resource.RSSKB <= 0 {
			issues = append(issues, "run resource RSS evidence is required")
		}
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateSemanticProbeCoverage(checks []semanticCheck) []string {
	seen := make(map[string]bool, len(checks))
	for _, check := range checks {
		seen[check.Name] = true
	}
	var issues []string
	for _, name := range requiredSemanticProbeNames() {
		if !seen[name] {
			issues = append(issues, "semantic probe missing "+name)
		}
	}
	return issues
}

func requiredSemanticProbeNames() []string {
	return []string{
		"plaintext headers/body",
		"json headers/body",
		"db real read",
		"query clamping",
		"updates persistence",
		"fortunes insertion escaping sorting",
	}
}

func writeJSON(root string, path string, value any) error {
	abs := absPath(root, path)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(abs, raw, 0o644)
}

func absPath(root string, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(root, path)
}

func detectEnvironment() benchmarkEnvironment {
	hostname, err := os.Hostname()
	if err != nil || strings.TrimSpace(hostname) == "" {
		hostname = "unknown"
	}
	return benchmarkEnvironment{
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		GoVersion: runtime.Version(),
		Hostname:  hostname,
	}
}

func detectGitState() gitState {
	head := strings.TrimSpace(runGit("rev-parse", "--short=12", "HEAD"))
	if head == "" {
		head = "unknown"
	}
	status := strings.TrimSpace(runGit("status", "--porcelain", "--untracked-files=all"))
	worktreeStatus := "clean"
	if status != "" {
		worktreeStatus = "dirty"
	}
	return gitState{Head: head, WorktreeStatus: worktreeStatus}
}

func runGit(args ...string) string {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(out)
}
