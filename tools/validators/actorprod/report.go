package actorprod

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"tetra_language/tools/validators/actordist"
	"tetra_language/tools/validators/parallelprod"
)

const SchemaV1 = "tetra.actor.production_foundation.v1"
const ArtifactHashSchema = "tetra.release-artifact-hashes.v1alpha1"

type Options struct {
	CurrentGitHead string
}

type Report struct {
	Schema         string           `json:"schema"`
	Status         string           `json:"status"`
	Target         string           `json:"target"`
	Host           string           `json:"host"`
	GitHead        string           `json:"git_head"`
	ReportDir      string           `json:"report_dir"`
	ArtifactHashes string           `json:"artifact_hashes"`
	Claims         []string         `json:"claims,omitempty"`
	NonClaims      []string         `json:"nonclaims"`
	Commands       []CommandReport  `json:"commands"`
	Artifacts      []ArtifactReport `json:"artifacts"`
}

type CommandReport struct {
	Name    string `json:"name"`
	Command string `json:"command"`
	Status  string `json:"status"`
	Log     string `json:"log"`
}

type ArtifactReport struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	Schema string `json:"schema"`
}

type hashManifest struct {
	Schema    string         `json:"schema"`
	Root      string         `json:"root"`
	Artifacts []hashArtifact `json:"artifacts"`
}

type hashArtifact struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
	Schema string `json:"schema"`
}

func ValidateReport(raw []byte) error {
	return ValidateReportWithOptions(raw, Options{})
}

func ValidateReportWithOptions(raw []byte, opts Options) error {
	var report Report
	if err := decodeStrict(raw, &report); err != nil {
		return err
	}
	var issues []string
	issues = append(issues, rejectPaperEvidence(raw)...)
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
	if !isHexGitHead(report.GitHead) {
		issues = append(issues, fmt.Sprintf("git_head is %q, want 40 lowercase hex characters", report.GitHead))
	}
	if opts.CurrentGitHead != "" && report.GitHead != opts.CurrentGitHead {
		issues = append(issues, fmt.Sprintf("git_head %q does not match current git head %q", report.GitHead, opts.CurrentGitHead))
	}
	if report.ReportDir != "." {
		issues = append(issues, fmt.Sprintf("report_dir is %q, want .", report.ReportDir))
	}
	if report.ArtifactHashes != "artifact-hashes.json" {
		issues = append(issues, fmt.Sprintf("artifact_hashes is %q, want artifact-hashes.json", report.ArtifactHashes))
	}
	issues = append(issues, validateClaims(report.Claims)...)
	issues = append(issues, validateNonClaims(report.NonClaims)...)
	issues = append(issues, validateCommands(report.Commands)...)
	issues = append(issues, validateArtifacts(report.Artifacts)...)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func ValidateReportDir(dir string, opts Options) error {
	manifestPath := filepath.Join(dir, "actor-runtime-foundation-manifest.json")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	if err := ValidateReportWithOptions(raw, opts); err != nil {
		return err
	}
	var report Report
	if err := decodeStrict(raw, &report); err != nil {
		return err
	}
	if err := validateSubreport(filepath.Join(dir, "parallel-production-linux-x64", "parallel-production-linux-x64.json"), parallelprod.ValidateReport); err != nil {
		return err
	}
	distributedPath := filepath.Join(dir, "distributed-actors-linux-x64", "distributed-actors-linux-x64.json")
	if err := validateSubreport(distributedPath, actordist.ValidateReport); err != nil {
		return err
	}
	if err := validateDistributedGitHead(distributedPath, report.GitHead); err != nil {
		return err
	}
	expected := expectedHashSchemas(report.Artifacts)
	delete(expected, "artifact-hashes.json")
	if err := validateArtifactHashManifest(dir, "artifact-hashes.json", expected); err != nil {
		return err
	}
	if err := validateArtifactHashManifest(filepath.Join(dir, "parallel-production-linux-x64"), "artifact-hashes.json", map[string]string{
		"parallel-production-linux-x64.json": "tetra.parallel.production.v1",
	}); err != nil {
		return err
	}
	if err := validateArtifactHashManifest(filepath.Join(dir, "distributed-actors-linux-x64"), "artifact-hashes.json", map[string]string{
		"distributed-actors-linux-x64.json": "tetra.actors.distributed-runtime.v1",
	}); err != nil {
		return err
	}
	return nil
}

func validateSubreport(path string, validate func([]byte) error) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("%s: %w", filepath.ToSlash(path), err)
	}
	if err := validate(raw); err != nil {
		return fmt.Errorf("%s: %w", filepath.ToSlash(path), err)
	}
	return nil
}

func validateDistributedGitHead(path string, gitHead string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var report actordist.Report
	if err := decodeStrict(raw, &report); err != nil {
		return err
	}
	if report.GitHead != gitHead {
		return fmt.Errorf("distributed actor git_head %q does not match foundation git_head %q", report.GitHead, gitHead)
	}
	return nil
}

func validateCommands(commands []CommandReport) []string {
	required := map[string]bool{
		"distributed-actors-smoke":   false,
		"parallel-production-smoke":  false,
		"focused-actor-tests":        false,
		"race-actor-slice":           false,
		"validate-manifest":          false,
		"verify-docs":                false,
		"artifact-hashes-write":      false,
		"artifact-hashes-validate":   false,
		"actor-foundation-validator": false,
	}
	var issues []string
	seen := map[string]bool{}
	for _, c := range commands {
		name := strings.TrimSpace(c.Name)
		if name == "" {
			issues = append(issues, "command name is required")
			continue
		}
		if seen[name] {
			issues = append(issues, fmt.Sprintf("duplicate command %s", name))
		}
		seen[name] = true
		if _, ok := required[name]; ok {
			required[name] = true
		}
		if c.Status != "pass" {
			issues = append(issues, fmt.Sprintf("command %s status is %q, want pass", name, c.Status))
		}
		command := strings.TrimSpace(c.Command)
		if command == "" {
			issues = append(issues, fmt.Sprintf("command %s command text is required", name))
		}
		if strings.TrimSpace(c.Log) == "" {
			issues = append(issues, fmt.Sprintf("command %s log is required", name))
		}
		if name == "race-actor-slice" && !strings.Contains(command, "-race") {
			issues = append(issues, "race-actor-slice command must include -race")
		}
		for _, forbidden := range []string{"|| true", "continue-on-error", "set +e"} {
			if strings.Contains(command, forbidden) {
				issues = append(issues, fmt.Sprintf("command %s contains bypass marker %q", name, forbidden))
			}
		}
	}
	for name, ok := range required {
		if !ok {
			issues = append(issues, fmt.Sprintf("missing required command %s", name))
		}
	}
	return issues
}

func validateArtifacts(artifacts []ArtifactReport) []string {
	required := map[string]string{
		"actor-runtime-foundation-manifest.json":                           SchemaV1,
		"parallel-production-linux-x64/parallel-production-linux-x64.json": "tetra.parallel.production.v1",
		"parallel-production-linux-x64/artifact-hashes.json":               ArtifactHashSchema,
		"distributed-actors-linux-x64/distributed-actors-linux-x64.json":   "tetra.actors.distributed-runtime.v1",
		"distributed-actors-linux-x64/artifact-hashes.json":                ArtifactHashSchema,
		"artifact-hashes.json":                                             ArtifactHashSchema,
	}
	var issues []string
	seen := map[string]bool{}
	for _, a := range artifacts {
		path := strings.TrimSpace(a.Path)
		if path == "" {
			issues = append(issues, "artifact path is required")
			continue
		}
		if seen[path] {
			issues = append(issues, fmt.Sprintf("duplicate artifact %s", path))
		}
		seen[path] = true
		if filepath.IsAbs(path) || strings.Contains(path, "..") || strings.Contains(path, "\\") {
			issues = append(issues, fmt.Sprintf("artifact path %q must be relative slash path", path))
		}
		if strings.TrimSpace(a.Kind) == "" {
			issues = append(issues, fmt.Sprintf("artifact %s kind is required", path))
		}
		if strings.TrimSpace(a.Schema) == "" {
			issues = append(issues, fmt.Sprintf("artifact %s schema is required", path))
		}
		if want, ok := required[path]; ok {
			if a.Schema != want {
				issues = append(issues, fmt.Sprintf("artifact %s schema is %q, want %q", path, a.Schema, want))
			}
			delete(required, path)
		}
	}
	for path := range required {
		issues = append(issues, fmt.Sprintf("missing required artifact %s", path))
	}
	return issues
}

func validateClaims(claims []string) []string {
	var issues []string
	for _, claim := range claims {
		lower := strings.ToLower(claim)
		for _, forbidden := range []string{"erlang/otp", "cluster", "reconnect", "retry", "non-linux", "formal race", "distributed zero-copy"} {
			if strings.Contains(lower, forbidden) {
				issues = append(issues, fmt.Sprintf("forbidden actor foundation claim %q mentions %q", claim, forbidden))
			}
		}
		if strings.Contains(lower, "distributed actor") {
			for _, unsupportedTarget := range []string{"windows", "win64", "macos", "darwin", "all-target", "all target", "cross-target", "cross target", "cross-platform", "cross platform"} {
				if strings.Contains(lower, unsupportedTarget) {
					issues = append(issues, fmt.Sprintf("cross-target distributed actor claim %q lacks target-host smoke evidence for %q", claim, unsupportedTarget))
				}
			}
		}
	}
	return issues
}

func validateNonClaims(nonclaims []string) []string {
	if len(nonclaims) == 0 {
		return []string{"nonclaims must include actor foundation scope boundaries"}
	}
	joined := strings.ToLower(strings.Join(nonclaims, "\n"))
	var issues []string
	for _, required := range []string{"erlang", "cluster", "reconnect", "retry", "non-linux", "zero-copy", "formal race"} {
		if !strings.Contains(joined, required) {
			issues = append(issues, fmt.Sprintf("nonclaims missing %q boundary", required))
		}
	}
	return issues
}

func expectedHashSchemas(artifacts []ArtifactReport) map[string]string {
	expected := map[string]string{}
	for _, artifact := range artifacts {
		expected[artifact.Path] = artifact.Schema
	}
	return expected
}

func validateArtifactHashManifest(root, rel string, expected map[string]string) error {
	manifestPath := filepath.Join(root, rel)
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("%s: %w", filepath.ToSlash(manifestPath), err)
	}
	var manifest hashManifest
	if err := decodeStrict(raw, &manifest); err != nil {
		return fmt.Errorf("%s: %w", filepath.ToSlash(manifestPath), err)
	}
	var issues []string
	if manifest.Schema != ArtifactHashSchema {
		issues = append(issues, fmt.Sprintf("%s schema is %q, want %s", rel, manifest.Schema, ArtifactHashSchema))
	}
	if manifest.Root != "." {
		issues = append(issues, fmt.Sprintf("%s root is %q, want .", rel, manifest.Root))
	}
	seen := map[string]bool{}
	for _, artifact := range manifest.Artifacts {
		if artifact.Path == rel {
			issues = append(issues, fmt.Sprintf("%s must not list itself", rel))
		}
		if filepath.IsAbs(artifact.Path) || strings.Contains(artifact.Path, "..") || strings.Contains(artifact.Path, "\\") {
			issues = append(issues, fmt.Sprintf("%s artifact path %q must be relative slash path", rel, artifact.Path))
			continue
		}
		if seen[artifact.Path] {
			issues = append(issues, fmt.Sprintf("%s duplicate artifact %s", rel, artifact.Path))
		}
		seen[artifact.Path] = true
		if want, ok := expected[artifact.Path]; ok {
			if artifact.Schema != want {
				issues = append(issues, fmt.Sprintf("%s artifact %s schema is %q, want %q", rel, artifact.Path, artifact.Schema, want))
			}
			delete(expected, artifact.Path)
		}
		if err := validateHashedArtifact(root, artifact); err != nil {
			issues = append(issues, err.Error())
		}
	}
	for path := range expected {
		issues = append(issues, fmt.Sprintf("%s missing artifact %s", rel, path))
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateHashedArtifact(root string, artifact hashArtifact) error {
	path := filepath.Join(root, filepath.FromSlash(artifact.Path))
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("artifact-hashes.json artifact %s: %w", artifact.Path, err)
	}
	sum := sha256.Sum256(raw)
	wantSHA := fmt.Sprintf("sha256:%x", sum)
	var issues []string
	if artifact.SHA256 != wantSHA {
		issues = append(issues, fmt.Sprintf("artifact-hashes.json sha256 mismatch for %s: got %s want %s", artifact.Path, artifact.SHA256, wantSHA))
	}
	if artifact.Size != int64(len(raw)) {
		issues = append(issues, fmt.Sprintf("artifact-hashes.json size mismatch for %s: got %d want %d", artifact.Path, artifact.Size, len(raw)))
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func rejectPaperEvidence(raw []byte) []string {
	lower := strings.ToLower(string(raw))
	var issues []string
	for _, marker := range []string{"metadata-only", "build-only", "docs-only", "sidecar-only", " fake", "\"fake\"", "mock", "placeholder"} {
		if strings.Contains(lower, marker) {
			issues = append(issues, fmt.Sprintf("report contains forbidden non-production evidence marker %q", strings.Trim(marker, " \"")))
		}
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

func decodeStrict(raw []byte, out any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("trailing JSON data")
	}
	return nil
}
