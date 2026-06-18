package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"tetra_language/tools/internal/artifacts"
	"tetra_language/tools/internal/gatecontract"
	"tetra_language/tools/internal/reportdir"
)

const (
	runGateContractExecEnv      = "TETRA_RUN_GATE_CONTRACT_EXEC"
	runGateContractIDEnv        = "TETRA_RUN_GATE_CONTRACT_ID"
	runGateContractReportDirEnv = "TETRA_RUN_GATE_REPORT_DIR"
)

type dryRunPlan struct {
	Mode                    string                           `json:"mode"`
	Contract                dryRunContractIdentity           `json:"contract"`
	ReportDir               string                           `json:"report_dir"`
	ResolvedReportDir       string                           `json:"resolved_report_dir"`
	Steps                   []gatecontract.Step              `json:"steps"`
	RequiredReports         []gatecontract.RequiredReport    `json:"required_reports"`
	Validators              []gatecontract.Validator         `json:"validators"`
	ArtifactHashes          *gatecontract.ArtifactHashPolicy `json:"artifact_hashes"`
	ArtifactHashCommandPlan dryRunHashCommandPlan            `json:"artifact_hash_command_plan"`
	Claims                  []gatecontract.Claim             `json:"claims"`
	Nonclaims               []gatecontract.Nonclaim          `json:"nonclaims"`
	CIArtifacts             []gatecontract.CIArtifact        `json:"ci_artifacts"`
}

type dryRunContractIdentity struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Scope      string `json:"scope"`
	Producer   string `json:"producer"`
	Entrypoint string `json:"entrypoint"`
}

type dryRunHashCommandPlan struct {
	ManifestPath string            `json:"manifest_path"`
	Write        dryRunCommandPlan `json:"write"`
	Validate     dryRunCommandPlan `json:"validate"`
}

type dryRunCommandPlan struct {
	Name string   `json:"name"`
	Args []string `json:"args"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	var contractPath string
	var reportDir string
	var dryRun bool
	var jsonOutput bool

	fs := flag.NewFlagSet("run-gate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&contractPath, "contract", "", "path to tetra gate contract JSON")
	fs.StringVar(
		&reportDir,
		"report-dir",
		"",
		"fresh report directory, relative to repository root",
	)
	fs.BoolVar(&dryRun, "dry-run", false, "print the deterministic gate plan without executing it")
	fs.BoolVar(&jsonOutput, "json", false, "write dry-run plan as JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if contractPath == "" {
		fmt.Fprintln(stderr, "error: --contract is required")
		return 2
	}
	if reportDir == "" {
		fmt.Fprintln(stderr, "error: --report-dir is required")
		return 2
	}
	if !dryRun {
		return runContractEntrypoint(contractPath, reportDir, stdout, stderr)
	}

	plan, err := buildDryRunPlan(contractPath, reportDir)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if jsonOutput {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(plan); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	}

	fmt.Fprintf(stdout, "dry-run gate plan: %s (%s)\n", plan.Contract.ID, plan.Contract.Scope)
	fmt.Fprintf(stdout, "entrypoint: %s\n", plan.Contract.Entrypoint)
	fmt.Fprintf(stdout, "report_dir: %s\n", plan.ReportDir)
	fmt.Fprintf(stdout, "resolved_report_dir: %s\n", plan.ResolvedReportDir)
	fmt.Fprintf(stdout, "steps: %d\n", len(plan.Steps))
	fmt.Fprintf(stdout, "required_reports: %d\n", len(plan.RequiredReports))
	fmt.Fprintf(stdout, "validators: %d\n", len(plan.Validators))
	return 0
}

func buildDryRunPlan(contractPath, reportDir string) (dryRunPlan, error) {
	contract, _, resolvedReportDir, err := loadContractRunInputs(contractPath, reportDir)
	if err != nil {
		return dryRunPlan{}, err
	}

	hashPlan, err := artifacts.NewHashCommandPlan(resolvedReportDir)
	if err != nil {
		return dryRunPlan{}, err
	}

	return dryRunPlan{
		Mode: "dry-run",
		Contract: dryRunContractIdentity{
			ID:         contract.ID,
			Title:      contract.Title,
			Scope:      contract.Scope,
			Producer:   contract.Producer,
			Entrypoint: contract.Entrypoint,
		},
		ReportDir:               reportDir,
		ResolvedReportDir:       resolvedReportDir,
		Steps:                   contract.Steps,
		RequiredReports:         contract.RequiredReports,
		Validators:              contract.Validators,
		ArtifactHashes:          contract.ArtifactHashes,
		ArtifactHashCommandPlan: newDryRunHashCommandPlan(hashPlan),
		Claims:                  contract.Claims,
		Nonclaims:               contract.Nonclaims,
		CIArtifacts:             contract.CIArtifacts,
	}, nil
}

func runContractEntrypoint(contractPath, reportDir string, stdout, stderr io.Writer) int {
	contract, repoRoot, _, err := loadContractRunInputs(contractPath, reportDir)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if os.Getenv(runGateContractExecEnv) == "1" && os.Getenv(runGateContractIDEnv) == contract.ID {
		fmt.Fprintf(
			stderr,
			"error: refusing recursive run-gate execution for contract %q\n",
			contract.ID,
		)
		return 2
	}
	entrypoint, err := cleanContractEntrypoint(contract.Entrypoint)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	cmd := exec.Command("bash", entrypoint, "--report-dir", reportDir)
	cmd.Dir = repoRoot
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = runGateEntrypointEnv(os.Environ(), contract.ID, reportDir)
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		fmt.Fprintf(stderr, "execute gate entrypoint %q: %v\n", contract.Entrypoint, err)
		return 1
	}
	return 0
}

func loadContractRunInputs(
	contractPath, reportDir string,
) (gatecontract.Contract, string, string, error) {
	contract, err := gatecontract.Load(contractPath)
	if err != nil {
		return gatecontract.Contract{}, "", "", err
	}

	repoRoot, err := os.Getwd()
	if err != nil {
		return gatecontract.Contract{}, "", "", fmt.Errorf(
			"resolve current working directory: %w",
			err,
		)
	}
	resolvedReportDir, err := reportdir.ValidateFresh(repoRoot, reportDir)
	if err != nil {
		return gatecontract.Contract{}, "", "", err
	}
	return contract, repoRoot, filepath.Clean(resolvedReportDir), nil
}

func cleanContractEntrypoint(entrypoint string) (string, error) {
	if strings.TrimSpace(entrypoint) == "" {
		return "", fmt.Errorf("empty contract entrypoint")
	}
	if filepath.IsAbs(entrypoint) {
		return "", fmt.Errorf("absolute contract entrypoint is not allowed: %s", entrypoint)
	}
	cleaned := filepath.Clean(filepath.FromSlash(entrypoint))
	if cleaned == "." || cleaned == ".." ||
		strings.HasPrefix(cleaned, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf(
			"parent traversal is not allowed in contract entrypoint: %s",
			entrypoint,
		)
	}
	if strings.HasPrefix(cleaned, "-") {
		return "", fmt.Errorf("dash-prefixed contract entrypoint is not allowed: %s", entrypoint)
	}
	return cleaned, nil
}

func runGateEntrypointEnv(base []string, contractID, reportDir string) []string {
	env := make([]string, 0, len(base)+3)
	for _, value := range base {
		if envNameIs(value, runGateContractExecEnv) || envNameIs(value, runGateContractIDEnv) ||
			envNameIs(value, runGateContractReportDirEnv) {
			continue
		}
		env = append(env, value)
	}
	env = append(env,
		runGateContractExecEnv+"=1",
		runGateContractIDEnv+"="+contractID,
		runGateContractReportDirEnv+"="+reportDir,
	)
	return env
}

func envNameIs(value, name string) bool {
	return strings.HasPrefix(value, name+"=")
}

func newDryRunHashCommandPlan(plan artifacts.HashCommandPlan) dryRunHashCommandPlan {
	return dryRunHashCommandPlan{
		ManifestPath: plan.ManifestPath,
		Write:        newDryRunCommandPlan(plan.Write),
		Validate:     newDryRunCommandPlan(plan.Validate),
	}
}

func newDryRunCommandPlan(plan artifacts.CommandPlan) dryRunCommandPlan {
	return dryRunCommandPlan{
		Name: plan.Name,
		Args: plan.Args,
	}
}
