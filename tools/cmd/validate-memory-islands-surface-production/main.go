package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const integratedManifestSchema = "tetra.memory-islands-surface.production-gate.v1"
const integratedHashSchema = "tetra.release-artifact-hashes.v1alpha1"

type integratedManifest struct {
	Schema       string                  `json:"schema"`
	Status       string                  `json:"status"`
	GitHead      string                  `json:"git_head"`
	GeneratedAt  string                  `json:"generated_at"`
	ReportDir    string                  `json:"report_dir"`
	HashManifest string                  `json:"hash_manifest"`
	Commands     []integratedCommand     `json:"commands"`
	Artifacts    []integratedArtifactRef `json:"artifacts"`
}

type integratedCommand struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

type integratedArtifactRef struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	Schema string `json:"schema,omitempty"`
}

type requiredIntegratedArtifact struct {
	Path   string
	Kind   string
	Schema string
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
	Schema string `json:"schema,omitempty"`
}

type schemaEnvelope struct {
	Schema        string `json:"schema"`
	SchemaVersion string `json:"schema_version"`
}

var requiredManifestArtifacts = []requiredIntegratedArtifact{
	{Path: "memory/memory-production-linux-x64.json", Kind: "memory_production_report", Schema: "tetra.memory.production.v1"},
	{Path: "memory/memory-release-manifest.json", Kind: "memory_release_manifest", Schema: "tetra.memory.release-manifest.v1"},
	{Path: "memory/island-proof-verifier.json", Kind: "island_proof_verifier_report", Schema: "tetra.island.proof.v1"},
	{Path: "memory/island-proof-memory-report.json", Kind: "island_proof_memory_report", Schema: "tetra.memory-report.v1"},
	{Path: "memory/memory-fuzz-tier1/island-proof-fuzz-summary.json", Kind: "island_proof_fuzz_summary", Schema: "tetra.island-proof-fuzz-summary.v1"},
	{Path: "islands-debug-smoke.json", Kind: "islands_debug_smoke_report", Schema: "tetra.release.v0_2_0.smoke-report.v1"},
	{Path: "surface-release-v1/surface-release-summary.json", Kind: "surface_release_summary", Schema: "tetra.surface.release.v1"},
	{Path: "surface-experimental-regression/artifact-hashes.json", Kind: "surface_experimental_hash_manifest", Schema: "tetra.release-artifact-hashes.v1alpha1"},
	{Path: "safe-view-lifetime/safe-view-lifetime-summary.json", Kind: "safe_view_lifetime_summary", Schema: "tetra.safe-view-lifetime.gate.v1"},
	{Path: "surface-api-stability-v1/surface-api-stability-summary.json", Kind: "surface_api_stability_summary", Schema: "tetra.surface.api-stability.v1"},
	{Path: "artifact-hashes.json", Kind: "integrated_hash_manifest", Schema: "tetra.release-artifact-hashes.v1alpha1"},
}

var requiredHashArtifacts = []string{
	"memory-islands-surface-production-manifest.json",
	"memory/memory-production-linux-x64.json",
	"memory/memory-release-manifest.json",
	"memory/artifact-hashes.json",
	"memory/island-proof-verifier.json",
	"memory/island-proof-memory-report.json",
	"memory/memory-fuzz-tier1/island-proof-fuzz-summary.json",
	"islands-debug-smoke.json",
	"surface-release-v1/surface-release-summary.json",
	"surface-release-v1/artifact-hashes.json",
	"surface-experimental-regression/artifact-hashes.json",
	"safe-view-lifetime/safe-view-lifetime-summary.json",
	"surface-api-stability-v1/surface-api-stability-summary.json",
}

var requiredCommands = map[string]string{
	"memory-production-gate":               "memory-production-linux-x64-smoke.sh",
	"islands-debug-smoke":                  "go run ./cli/cmd/tetra smoke --target linux-x64 --run=true --islands-debug",
	"validate-islands-debug-smoke":         "go run ./tools/cmd/smoke-report-to-checklist --validate-only",
	"surface-release-gate":                 "scripts/release/surface/release-gate.sh",
	"surface-experimental-regression-gate": "scripts/release/surface/gate.sh",
	"safe-view-lifetime-gate":              "scripts/release/safe-view-lifetime/gate.sh",
	"surface-api-stability-gate":           "scripts/release/surface/api-stability-gate.sh",
	"validate-manifest":                    "go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json",
	"verify-docs":                          "go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json",
	"artifact-hashes-write":                "go run ./tools/cmd/validate-artifact-hashes --write",
	"artifact-hashes-validate":             "go run ./tools/cmd/validate-artifact-hashes --manifest",
	"integrated-release-validator":         "go run ./tools/cmd/validate-memory-islands-surface-production --report-dir",
}

func main() {
	reportDir := flag.String("report-dir", "", "integrated Memory/Islands/Surface production report directory")
	currentGitHead := flag.String("current-git-head", "", "optional current git HEAD to require in same-commit artifacts")
	flag.Parse()
	if *reportDir == "" {
		fmt.Fprintln(os.Stderr, "error: --report-dir is required")
		os.Exit(2)
	}
	if err := validateIntegratedReportDir(*reportDir, *currentGitHead); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateIntegratedReportDir(reportDir string, currentGitHead string) error {
	reportDir = filepath.Clean(reportDir)
	info, err := os.Lstat(reportDir)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("integrated report dir must not be a symlink: %s", reportDir)
	}
	if !info.IsDir() {
		return fmt.Errorf("integrated report dir is not a directory: %s", reportDir)
	}

	manifestPath := filepath.Join(reportDir, "memory-islands-surface-production-manifest.json")
	var manifest integratedManifest
	if err := readStrictJSON(manifestPath, &manifest); err != nil {
		return fmt.Errorf("integrated manifest: %w", err)
	}

	var issues []string
	issues = append(issues, validateManifestEnvelope(manifest, currentGitHead)...)
	issues = append(issues, validateManifestCommands(manifest.Commands)...)
	issues = append(issues, validateManifestArtifactRefs(manifest.Artifacts)...)
	issues = append(issues, validateRequiredEvidence(reportDir, manifest.GitHead)...)
	issues = append(issues, validateIntegratedHashManifest(filepath.Join(reportDir, manifest.HashManifest), reportDir)...)
	if len(issues) > 0 {
		return fmt.Errorf(strings.Join(issues, "; "))
	}
	return nil
}

func validateManifestEnvelope(manifest integratedManifest, currentGitHead string) []string {
	var issues []string
	if manifest.Schema != integratedManifestSchema {
		issues = append(issues, fmt.Sprintf("integrated manifest schema is %q, want %q", manifest.Schema, integratedManifestSchema))
	}
	if manifest.Status != "pass" {
		issues = append(issues, fmt.Sprintf("integrated manifest status is %q, want pass", manifest.Status))
	}
	if !isFullGitHead(manifest.GitHead) {
		issues = append(issues, "integrated git_head must be a 40-character lowercase hex commit")
	}
	if strings.TrimSpace(currentGitHead) != "" && manifest.GitHead != strings.TrimSpace(currentGitHead) {
		issues = append(issues, fmt.Sprintf("integrated git_head %s does not match current git head %s", manifest.GitHead, strings.TrimSpace(currentGitHead)))
	}
	if _, err := time.Parse(time.RFC3339, manifest.GeneratedAt); err != nil {
		issues = append(issues, fmt.Sprintf("integrated manifest generated_at must be RFC3339: %v", err))
	}
	if manifest.ReportDir != "." {
		issues = append(issues, fmt.Sprintf("integrated manifest report_dir is %q, want .", manifest.ReportDir))
	}
	if manifest.HashManifest != "artifact-hashes.json" {
		issues = append(issues, fmt.Sprintf("integrated manifest hash_manifest is %q, want artifact-hashes.json", manifest.HashManifest))
	}
	return issues
}

func validateManifestCommands(commands []integratedCommand) []string {
	var issues []string
	seen := map[string]bool{}
	for _, command := range commands {
		name := strings.TrimSpace(command.Name)
		if name == "" {
			issues = append(issues, "integrated manifest command name is required")
			continue
		}
		if seen[name] {
			issues = append(issues, fmt.Sprintf("duplicate integrated manifest command %s", name))
		}
		seen[name] = true
		if strings.TrimSpace(command.Command) == "" {
			issues = append(issues, fmt.Sprintf("integrated manifest command %s command is required", name))
		}
		if strings.Contains(command.Command, "|| true") || strings.Contains(command.Command, "continue-on-error") {
			issues = append(issues, fmt.Sprintf("integrated manifest command %s contains bypass marker", name))
		}
	}
	for name, fragment := range requiredCommands {
		found := false
		for _, command := range commands {
			if command.Name == name && strings.Contains(command.Command, fragment) {
				found = true
				break
			}
		}
		if !found {
			issues = append(issues, fmt.Sprintf("missing integrated manifest command %s containing %q", name, fragment))
		}
	}
	return issues
}

func validateManifestArtifactRefs(artifacts []integratedArtifactRef) []string {
	var issues []string
	byKind := map[string]integratedArtifactRef{}
	byPath := map[string]bool{}
	for _, artifact := range artifacts {
		if err := validateSafeRel(artifact.Path); err != nil {
			issues = append(issues, fmt.Sprintf("integrated manifest artifact path %q is invalid: %v", artifact.Path, err))
			continue
		}
		if strings.TrimSpace(artifact.Kind) == "" {
			issues = append(issues, fmt.Sprintf("integrated manifest artifact %s kind is required", artifact.Path))
			continue
		}
		if _, ok := byKind[artifact.Kind]; ok {
			issues = append(issues, fmt.Sprintf("duplicate integrated manifest artifact kind %s", artifact.Kind))
		}
		byKind[artifact.Kind] = artifact
		if byPath[artifact.Path] {
			issues = append(issues, fmt.Sprintf("duplicate integrated manifest artifact path %s", artifact.Path))
		}
		byPath[artifact.Path] = true
	}
	for _, required := range requiredManifestArtifacts {
		artifact, ok := byKind[required.Kind]
		if !ok {
			issues = append(issues, fmt.Sprintf("missing integrated manifest artifact %s", required.Kind))
			continue
		}
		if artifact.Path != required.Path {
			issues = append(issues, fmt.Sprintf("integrated manifest artifact %s path is %q, want %q", required.Kind, artifact.Path, required.Path))
		}
		if required.Schema != "" && artifact.Schema != required.Schema {
			issues = append(issues, fmt.Sprintf("integrated manifest artifact %s schema is %q, want %q", required.Kind, artifact.Schema, required.Schema))
		}
	}
	return issues
}

func validateRequiredEvidence(reportDir string, gitHead string) []string {
	var issues []string

	var memoryReport struct {
		Schema string `json:"schema"`
		Status string `json:"status"`
		Target string `json:"target"`
	}
	if err := readJSON(filepath.Join(reportDir, "memory", "memory-production-linux-x64.json"), &memoryReport); err != nil {
		issues = append(issues, fmt.Sprintf("memory production report missing or invalid: %v", err))
	} else {
		if memoryReport.Schema != "tetra.memory.production.v1" {
			issues = append(issues, fmt.Sprintf("memory production report schema is %q, want tetra.memory.production.v1", memoryReport.Schema))
		}
		if memoryReport.Status != "pass" {
			issues = append(issues, fmt.Sprintf("memory production report status is %q, want pass", memoryReport.Status))
		}
		if memoryReport.Target != "linux-x64" {
			issues = append(issues, fmt.Sprintf("memory production report target is %q, want linux-x64", memoryReport.Target))
		}
	}

	var memoryManifest struct {
		Schema  string `json:"schema"`
		GitHead string `json:"git_head"`
	}
	if err := readJSON(filepath.Join(reportDir, "memory", "memory-release-manifest.json"), &memoryManifest); err != nil {
		issues = append(issues, fmt.Sprintf("memory release manifest missing or invalid: %v", err))
	} else {
		if memoryManifest.Schema != "tetra.memory.release-manifest.v1" {
			issues = append(issues, fmt.Sprintf("memory release manifest schema is %q, want tetra.memory.release-manifest.v1", memoryManifest.Schema))
		}
		if memoryManifest.GitHead != gitHead {
			issues = append(issues, fmt.Sprintf("memory release git_head %s does not match integrated git_head %s", memoryManifest.GitHead, gitHead))
		}
	}

	var islandProof struct {
		Schema  string `json:"schema"`
		GitHead string `json:"git_head"`
	}
	if err := readJSON(filepath.Join(reportDir, "memory", "island-proof-verifier.json"), &islandProof); err != nil {
		issues = append(issues, fmt.Sprintf("island proof verifier report missing or invalid: %v", err))
	} else {
		if islandProof.Schema != "tetra.island.proof.v1" {
			issues = append(issues, fmt.Sprintf("island proof verifier schema is %q, want tetra.island.proof.v1", islandProof.Schema))
		}
		if islandProof.GitHead != gitHead {
			issues = append(issues, fmt.Sprintf("island proof git_head %s does not match integrated git_head %s", islandProof.GitHead, gitHead))
		}
	}

	var islandMemory schemaEnvelope
	if err := readJSON(filepath.Join(reportDir, "memory", "island-proof-memory-report.json"), &islandMemory); err != nil {
		issues = append(issues, fmt.Sprintf("island proof memory report missing or invalid: %v", err))
	} else if schemaOf(islandMemory) != "tetra.memory-report.v1" {
		issues = append(issues, fmt.Sprintf("island proof memory report schema is %q, want tetra.memory-report.v1", schemaOf(islandMemory)))
	}

	var proofFuzz struct {
		Schema        string `json:"schema"`
		SchemaVersion string `json:"schema_version"`
		Status        string `json:"status"`
		Total         int    `json:"total"`
		Rejected      int    `json:"rejected"`
		Accepted      int    `json:"accepted"`
	}
	if err := readJSON(filepath.Join(reportDir, "memory", "memory-fuzz-tier1", "island-proof-fuzz-summary.json"), &proofFuzz); err != nil {
		issues = append(issues, fmt.Sprintf("island proof fuzz summary missing or invalid: %v", err))
	} else {
		proofFuzzSchema := proofFuzz.Schema
		if proofFuzzSchema == "" {
			proofFuzzSchema = proofFuzz.SchemaVersion
		}
		if proofFuzzSchema != "tetra.island-proof-fuzz-summary.v1" {
			issues = append(issues, fmt.Sprintf("island proof fuzz summary schema is %q, want tetra.island-proof-fuzz-summary.v1", proofFuzzSchema))
		}
		if proofFuzz.Status != "pass" {
			issues = append(issues, fmt.Sprintf("island proof fuzz summary status is %q, want pass", proofFuzz.Status))
		}
		if proofFuzz.Total <= 0 || proofFuzz.Rejected != proofFuzz.Total || proofFuzz.Accepted != 0 {
			issues = append(issues, fmt.Sprintf("island proof fuzz summary counts total=%d rejected=%d accepted=%d, want all rejected and zero accepted", proofFuzz.Total, proofFuzz.Rejected, proofFuzz.Accepted))
		}
	}

	issues = append(issues, validateIslandsDebugSmoke(filepath.Join(reportDir, "islands-debug-smoke.json"), gitHead)...)
	issues = append(issues, validateSurfaceReleaseSummary(filepath.Join(reportDir, "surface-release-v1", "surface-release-summary.json"), gitHead)...)
	issues = append(issues, validateHashManifestEnvelope(filepath.Join(reportDir, "memory", "artifact-hashes.json"), "memory hash manifest")...)
	issues = append(issues, validateHashManifestEnvelope(filepath.Join(reportDir, "surface-release-v1", "artifact-hashes.json"), "surface release hash manifest")...)
	issues = append(issues, validateHashManifestEnvelope(filepath.Join(reportDir, "surface-experimental-regression", "artifact-hashes.json"), "surface experimental hash manifest")...)
	issues = append(issues, validateSafeViewSummary(filepath.Join(reportDir, "safe-view-lifetime", "safe-view-lifetime-summary.json"))...)
	issues = append(issues, validateSurfaceAPISummary(filepath.Join(reportDir, "surface-api-stability-v1", "surface-api-stability-summary.json"))...)
	return issues
}

func validateIslandsDebugSmoke(path string, gitHead string) []string {
	var report struct {
		Target       string `json:"target"`
		GitHead      string `json:"git_head"`
		IslandsDebug bool   `json:"islands_debug"`
		Total        *int   `json:"total"`
		Passed       *int   `json:"passed"`
		Failed       *int   `json:"failed"`
		Cases        []struct {
			Name         string `json:"name"`
			SrcPath      string `json:"src_path"`
			ExpectedExit int    `json:"expected_exit"`
			ActualExit   *int   `json:"actual_exit"`
			Ran          bool   `json:"ran"`
			Pass         bool   `json:"pass"`
		} `json:"cases"`
	}
	if err := readJSON(path, &report); err != nil {
		return []string{fmt.Sprintf("islands debug smoke missing or invalid: %v", err)}
	}
	var issues []string
	if report.Target != "linux-x64" {
		issues = append(issues, fmt.Sprintf("islands debug smoke target is %q, want linux-x64", report.Target))
	}
	if report.GitHead != "" && !sameGitHead(report.GitHead, gitHead) {
		issues = append(issues, fmt.Sprintf("islands debug smoke git_head %s does not match integrated git_head %s", report.GitHead, gitHead))
	}
	if !report.IslandsDebug {
		issues = append(issues, "islands debug smoke islands_debug must be true")
	}
	if report.Total == nil || report.Passed == nil || report.Failed == nil {
		issues = append(issues, "islands debug smoke counts are required")
	}
	foundTrap := false
	for _, c := range report.Cases {
		if c.Name != "islands_overflow" {
			continue
		}
		foundTrap = true
		if c.SrcPath != "examples/islands_overflow.tetra" {
			issues = append(issues, fmt.Sprintf("islands debug smoke trap src_path is %q, want examples/islands_overflow.tetra", c.SrcPath))
		}
		if c.ExpectedExit == 0 {
			issues = append(issues, "islands debug smoke trap expected_exit must be non-zero")
		}
		if c.ActualExit == nil {
			issues = append(issues, "islands debug smoke trap missing actual_exit")
		} else if *c.ActualExit != c.ExpectedExit {
			issues = append(issues, fmt.Sprintf("islands debug smoke trap actual_exit=%d, want %d", *c.ActualExit, c.ExpectedExit))
		}
		if !c.Ran || !c.Pass {
			issues = append(issues, "islands debug smoke trap must run and pass")
		}
	}
	if !foundTrap {
		issues = append(issues, "islands debug smoke missing islands_overflow trap")
	}
	return issues
}

func validateSurfaceReleaseSummary(path string, gitHead string) []string {
	var summary struct {
		Schema       string `json:"schema"`
		Status       string `json:"status"`
		GitHead      string `json:"git_head"`
		ReleaseScope string `json:"release_scope"`
	}
	if err := readJSON(path, &summary); err != nil {
		return []string{fmt.Sprintf("surface release summary missing or invalid: %v", err)}
	}
	var issues []string
	if summary.Schema != "tetra.surface.release.v1" {
		issues = append(issues, fmt.Sprintf("surface release summary schema is %q, want tetra.surface.release.v1", summary.Schema))
	}
	if summary.Status != "current" {
		issues = append(issues, fmt.Sprintf("surface release summary status is %q, want current", summary.Status))
	}
	if summary.ReleaseScope != "surface-v1-linux-web" {
		issues = append(issues, fmt.Sprintf("surface release scope is %q, want surface-v1-linux-web", summary.ReleaseScope))
	}
	if summary.GitHead != gitHead {
		issues = append(issues, fmt.Sprintf("surface release git_head %s does not match integrated git_head %s", summary.GitHead, gitHead))
	}
	return issues
}

func validateHashManifestEnvelope(path string, label string) []string {
	var manifest struct {
		Schema    string `json:"schema"`
		Root      string `json:"root"`
		Artifacts []any  `json:"artifacts"`
	}
	if err := readJSON(path, &manifest); err != nil {
		return []string{fmt.Sprintf("%s missing or invalid: %v", label, err)}
	}
	var issues []string
	if manifest.Schema != integratedHashSchema {
		issues = append(issues, fmt.Sprintf("%s schema is %q, want %s", label, manifest.Schema, integratedHashSchema))
	}
	if manifest.Root != "." {
		issues = append(issues, fmt.Sprintf("%s root is %q, want .", label, manifest.Root))
	}
	if len(manifest.Artifacts) == 0 {
		issues = append(issues, fmt.Sprintf("%s artifacts must not be empty", label))
	}
	return issues
}

func validateSafeViewSummary(path string) []string {
	var summary struct {
		Schema          string `json:"schema"`
		Status          string `json:"status"`
		Bounded         bool   `json:"bounded"`
		ReleaseBlocking bool   `json:"release_blocking"`
	}
	if err := readJSON(path, &summary); err != nil {
		return []string{fmt.Sprintf("safe-view lifetime summary missing or invalid: %v", err)}
	}
	var issues []string
	if summary.Schema != "tetra.safe-view-lifetime.gate.v1" {
		issues = append(issues, fmt.Sprintf("safe-view lifetime summary schema is %q, want tetra.safe-view-lifetime.gate.v1", summary.Schema))
	}
	if summary.Status != "pass" {
		issues = append(issues, fmt.Sprintf("safe-view lifetime summary status is %q, want pass", summary.Status))
	}
	if !summary.Bounded {
		issues = append(issues, "safe-view lifetime summary bounded must be true")
	}
	if !summary.ReleaseBlocking {
		issues = append(issues, "safe-view lifetime summary release_blocking must be true")
	}
	return issues
}

func validateSurfaceAPISummary(path string) []string {
	var summary struct {
		Schema                string `json:"schema"`
		Status                string `json:"status"`
		ReleaseScope          string `json:"release_scope"`
		DocsManifestValidated bool   `json:"docs_manifest_validated"`
	}
	if err := readJSON(path, &summary); err != nil {
		return []string{fmt.Sprintf("surface API stability summary missing or invalid: %v", err)}
	}
	var issues []string
	if summary.Schema != "tetra.surface.api-stability.v1" {
		issues = append(issues, fmt.Sprintf("surface API stability summary schema is %q, want tetra.surface.api-stability.v1", summary.Schema))
	}
	if summary.Status != "pass" {
		issues = append(issues, fmt.Sprintf("surface API stability summary status is %q, want pass", summary.Status))
	}
	if summary.ReleaseScope != "surface-v1-linux-web" {
		issues = append(issues, fmt.Sprintf("surface API stability scope is %q, want surface-v1-linux-web", summary.ReleaseScope))
	}
	if !summary.DocsManifestValidated {
		issues = append(issues, "surface API stability summary docs_manifest_validated must be true")
	}
	return issues
}

func validateIntegratedHashManifest(path string, reportDir string) []string {
	var manifest hashManifest
	if err := readStrictJSON(path, &manifest); err != nil {
		return []string{fmt.Sprintf("integrated hash manifest missing or invalid: %v", err)}
	}
	var issues []string
	if manifest.Schema != integratedHashSchema {
		issues = append(issues, fmt.Sprintf("integrated hash manifest schema is %q, want %s", manifest.Schema, integratedHashSchema))
	}
	if manifest.Root != "." {
		issues = append(issues, fmt.Sprintf("integrated hash manifest root is %q, want .", manifest.Root))
	}
	if len(manifest.Artifacts) == 0 {
		issues = append(issues, "integrated hash manifest artifacts must not be empty")
	}
	seen := map[string]hashArtifact{}
	lastPath := ""
	for _, artifact := range manifest.Artifacts {
		if err := validateSafeRel(artifact.Path); err != nil {
			issues = append(issues, fmt.Sprintf("integrated hash manifest path %q is invalid: %v", artifact.Path, err))
			continue
		}
		if lastPath != "" && artifact.Path < lastPath {
			issues = append(issues, fmt.Sprintf("integrated hash manifest artifacts must be sorted: %s appears before %s", artifact.Path, lastPath))
		}
		lastPath = artifact.Path
		if _, ok := seen[artifact.Path]; ok {
			issues = append(issues, fmt.Sprintf("duplicate integrated hash manifest entry for %s", artifact.Path))
		}
		seen[artifact.Path] = artifact
		if err := validateSHA256(artifact.SHA256); err != nil {
			issues = append(issues, fmt.Sprintf("integrated hash manifest %s: %v", artifact.Path, err))
		}
	}
	for _, required := range requiredHashArtifacts {
		if _, ok := seen[required]; !ok {
			issues = append(issues, fmt.Sprintf("missing integrated hash manifest entry for %s", required))
		}
	}
	for rel, artifact := range seen {
		actual, err := hashFile(reportDir, rel)
		if err != nil {
			issues = append(issues, fmt.Sprintf("hash integrated artifact %s: %v", rel, err))
			continue
		}
		if actual.Size != artifact.Size {
			issues = append(issues, fmt.Sprintf("size mismatch for %s: got %d want %d", rel, actual.Size, artifact.Size))
		}
		if actual.SHA256 != artifact.SHA256 {
			issues = append(issues, fmt.Sprintf("sha256 mismatch for %s: got %s want %s", rel, actual.SHA256, artifact.SHA256))
		}
		if actual.Schema != artifact.Schema {
			issues = append(issues, fmt.Sprintf("schema mismatch for %s: got %q want %q", rel, actual.Schema, artifact.Schema))
		}
	}
	actualPaths, err := listActualArtifactPaths(reportDir)
	if err != nil {
		issues = append(issues, fmt.Sprintf("list integrated artifacts: %v", err))
	} else {
		for _, rel := range actualPaths {
			if _, ok := seen[rel]; !ok {
				issues = append(issues, fmt.Sprintf("unlisted integrated artifact %s", rel))
			}
		}
	}
	return issues
}

func readStrictJSON(path string, out any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("%s must contain a single JSON document", path)
	}
	return nil
}

func readJSON(path string, out any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return err
	}
	return nil
}

func validateSafeRel(rel string) error {
	if strings.TrimSpace(rel) == "" {
		return fmt.Errorf("path is empty")
	}
	if filepath.IsAbs(rel) {
		return fmt.Errorf("absolute paths are not allowed")
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(rel)))
	if clean == "." || clean != rel || strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") {
		return fmt.Errorf("path must be clean and stay under report root")
	}
	return nil
}

func isFullGitHead(value string) bool {
	if len(value) != 40 {
		return false
	}
	for _, ch := range value {
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') {
			return false
		}
	}
	return true
}

func sameGitHead(got string, want string) bool {
	got = strings.TrimSpace(got)
	want = strings.TrimSpace(want)
	if got == want {
		return true
	}
	if len(got) >= 7 && len(want) == 40 && strings.HasPrefix(want, got) {
		return true
	}
	if len(want) >= 7 && len(got) == 40 && strings.HasPrefix(got, want) {
		return true
	}
	return false
}

func schemaOf(envelope schemaEnvelope) string {
	if envelope.Schema != "" {
		return envelope.Schema
	}
	return envelope.SchemaVersion
}

func validateSHA256(value string) error {
	if !strings.HasPrefix(value, "sha256:") {
		return fmt.Errorf("sha256 must start with sha256:")
	}
	hexPart := strings.TrimPrefix(value, "sha256:")
	if len(hexPart) != 64 {
		return fmt.Errorf("sha256 must contain 64 hex chars")
	}
	for _, ch := range hexPart {
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') {
			return fmt.Errorf("sha256 contains non-hex character %q", ch)
		}
	}
	return nil
}

func hashFile(root string, rel string) (hashArtifact, error) {
	if err := validateSafeRel(rel); err != nil {
		return hashArtifact{}, err
	}
	path := filepath.Join(root, filepath.FromSlash(rel))
	info, err := os.Lstat(path)
	if err != nil {
		return hashArtifact{}, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return hashArtifact{}, fmt.Errorf("symlink artifact is not allowed")
	}
	if !info.Mode().IsRegular() {
		return hashArtifact{}, fmt.Errorf("artifact is not a regular file")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return hashArtifact{}, err
	}
	sum := sha256.Sum256(raw)
	return hashArtifact{
		Path:   rel,
		SHA256: "sha256:" + hex.EncodeToString(sum[:]),
		Size:   int64(len(raw)),
		Schema: detectJSONSchema(raw),
	}, nil
}

func detectJSONSchema(raw []byte) string {
	var envelope schemaEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return ""
	}
	return schemaOf(envelope)
}

func listActualArtifactPaths(root string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink artifact %s is not allowed", path)
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "artifact-hashes.json" {
			return nil
		}
		paths = append(paths, rel)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	return paths, nil
}
