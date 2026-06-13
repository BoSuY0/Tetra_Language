package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"tetra_language/tools/validators/islandproof"
	"tetra_language/tools/validators/memoryprod"
)

const (
	memoryReleaseManifestSchema = "tetra.memory.release-manifest.v1"
	memoryReleaseHashSchema     = "tetra.release-artifact-hashes.v1alpha1"
)

type memoryReleaseManifest struct {
	Schema       string                  `json:"schema"`
	Target       string                  `json:"target"`
	GitHead      string                  `json:"git_head"`
	GeneratedAt  string                  `json:"generated_at"`
	ReportDir    string                  `json:"report_dir"`
	HashManifest string                  `json:"hash_manifest"`
	Commands     []memoryReleaseCommand  `json:"commands"`
	Artifacts    []memoryReleaseArtifact `json:"artifacts"`
}

type memoryReleaseCommand struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

type memoryReleaseArtifact struct {
	Path    string `json:"path"`
	Kind    string `json:"kind"`
	Schema  string `json:"schema,omitempty"`
	Target  string `json:"target"`
	Command string `json:"command"`
}

type memoryReleaseHashManifest struct {
	Schema    string                      `json:"schema"`
	Root      string                      `json:"root"`
	Artifacts []memoryReleaseHashArtifact `json:"artifacts"`
}

type memoryReleaseHashArtifact struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
	Schema string `json:"schema,omitempty"`
}

type requiredMemoryReleaseArtifact struct {
	Kind            string
	Path            string
	Schema          string
	CommandFragment string
	RequireHash     bool
}

func main() {
	reportPath := flag.String("report", "", "path to tetra.memory.production.v1 JSON report")
	manifestPath := flag.String("manifest", "", "path to tetra.memory.release-manifest.v1 JSON manifest")
	reportDir := flag.String("report-dir", "", "memory release report directory")
	currentGitHead := flag.String("current-git-head", "", "optional current git HEAD to require in release manifest provenance")
	flag.Parse()
	if *reportPath == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	if *manifestPath == "" && *reportDir != "" {
		fmt.Fprintln(os.Stderr, "error: --report-dir requires --manifest")
		os.Exit(2)
	}
	var err error
	if *manifestPath != "" {
		root := *reportDir
		if root == "" {
			root = filepath.Dir(*manifestPath)
		}
		err = validateMemoryProductionReleaseManifest(*reportPath, *manifestPath, root, *currentGitHead)
	} else {
		err = validateMemoryProductionReport(*reportPath)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateMemoryProductionReport(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return memoryprod.ValidateReport(raw)
}

func validateMemoryProductionReleaseManifest(reportPath string, manifestPath string, reportDir string, currentGitHead string) error {
	if strings.TrimSpace(manifestPath) == "" {
		return fmt.Errorf("--manifest is required for release provenance validation")
	}
	if strings.TrimSpace(reportDir) == "" {
		return fmt.Errorf("--report-dir is required for release provenance validation")
	}
	if err := validateMemoryProductionReport(reportPath); err != nil {
		return fmt.Errorf("memory production report: %w", err)
	}

	reportDirAbs, err := filepath.Abs(reportDir)
	if err != nil {
		return err
	}
	reportPathAbs, err := filepath.Abs(reportPath)
	if err != nil {
		return err
	}
	manifestPathAbs, err := filepath.Abs(manifestPath)
	if err != nil {
		return err
	}

	var issues []string
	if cleanAbs(reportPathAbs) != cleanAbs(filepath.Join(reportDirAbs, "memory-production-linux-x64.json")) {
		issues = append(issues, fmt.Sprintf("--report must be %s", filepath.Join(reportDir, "memory-production-linux-x64.json")))
	}
	if cleanAbs(manifestPathAbs) != cleanAbs(filepath.Join(reportDirAbs, "memory-release-manifest.json")) {
		issues = append(issues, fmt.Sprintf("--manifest must be %s", filepath.Join(reportDir, "memory-release-manifest.json")))
	}

	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	var manifest memoryReleaseManifest
	if err := decodeMemoryReleaseStrictJSON(raw, &manifest); err != nil {
		return err
	}
	issues = append(issues, validateMemoryReleaseManifestEnvelope(manifest, currentGitHead)...)
	issues = append(issues, validateMemoryReleaseCommands(manifest.Commands)...)
	required := requiredMemoryReleaseArtifacts()
	issues = append(issues, validateMemoryReleaseArtifacts(reportDirAbs, manifest, required)...)
	issues = append(issues, validateMemoryReleaseHashManifest(reportDirAbs, manifest.HashManifest, required)...)
	issues = append(issues, validateMemoryReleaseIslandProofVerifier(reportDirAbs, manifest)...)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateMemoryReleaseManifestEnvelope(manifest memoryReleaseManifest, currentGitHead string) []string {
	var issues []string
	if manifest.Schema != memoryReleaseManifestSchema {
		issues = append(issues, fmt.Sprintf("release manifest schema is %q, want %q", manifest.Schema, memoryReleaseManifestSchema))
	}
	if manifest.Target != "linux-x64" {
		issues = append(issues, fmt.Sprintf("release manifest target is %q, want linux-x64", manifest.Target))
	}
	if !isMemoryReleaseGitHead(manifest.GitHead) {
		issues = append(issues, "release manifest git_head must be a 40-character lowercase hex commit")
	}
	currentGitHead = strings.TrimSpace(currentGitHead)
	if currentGitHead != "" {
		if !isMemoryReleaseGitHead(currentGitHead) {
			issues = append(issues, "current git head must be a 40-character lowercase hex commit")
		} else if manifest.GitHead != currentGitHead {
			issues = append(issues, fmt.Sprintf("release manifest git_head %s does not match current git head %s", manifest.GitHead, currentGitHead))
		}
	}
	if _, err := time.Parse(time.RFC3339, manifest.GeneratedAt); err != nil {
		issues = append(issues, fmt.Sprintf("release manifest generated_at must be RFC3339: %v", err))
	}
	if manifest.ReportDir != "." {
		issues = append(issues, fmt.Sprintf("release manifest report_dir is %q, want .", manifest.ReportDir))
	}
	if manifest.HashManifest != "artifact-hashes.json" {
		issues = append(issues, fmt.Sprintf("release manifest hash_manifest is %q, want artifact-hashes.json", manifest.HashManifest))
	}
	return issues
}

func validateMemoryReleaseCommands(commands []memoryReleaseCommand) []string {
	required := map[string]string{
		"memory-production-smoke":     "go run ./tools/cmd/memory-production-smoke",
		"target-report":               "go run ./cli/cmd/tetra targets",
		"validate-targets":            "go run ./tools/cmd/validate-targets",
		"memory-fuzz-short":           "go run ./tools/cmd/memory-fuzz-short",
		"validate-memory-fuzz-oracle": "go run ./tools/cmd/validate-memory-fuzz-oracle",
		"ram-contract-gate":           "ram-contract-linux-x64-smoke.sh",
		"island-proof-verifier":       "go run ./tools/cmd/validate-island-proof",
		"artifact-hashes-write":       "go run ./tools/cmd/validate-artifact-hashes --write",
		"artifact-hashes-validate":    "go run ./tools/cmd/validate-artifact-hashes --manifest",
	}
	seen := map[string]string{}
	var issues []string
	for _, command := range commands {
		name := strings.TrimSpace(command.Name)
		text := strings.TrimSpace(command.Command)
		if name == "" {
			issues = append(issues, "release manifest command name is required")
			continue
		}
		if _, ok := seen[name]; ok {
			issues = append(issues, fmt.Sprintf("duplicate release manifest command %s", name))
		}
		seen[name] = text
		if text == "" {
			issues = append(issues, fmt.Sprintf("release manifest command %s command is required", name))
		}
		if fragment, ok := required[name]; ok && !strings.Contains(text, fragment) {
			issues = append(issues, fmt.Sprintf("release manifest command %s must contain %q", name, fragment))
		}
	}
	for name, fragment := range required {
		text, ok := seen[name]
		if !ok {
			issues = append(issues, fmt.Sprintf("missing release manifest command %s", name))
			continue
		}
		if strings.TrimSpace(text) == "" {
			issues = append(issues, fmt.Sprintf("release manifest command %s command is required", name))
			continue
		}
		if !strings.Contains(text, fragment) {
			issues = append(issues, fmt.Sprintf("release manifest command %s must contain %q", name, fragment))
		}
	}
	return issues
}

func validateMemoryReleaseArtifacts(reportDir string, manifest memoryReleaseManifest, required []requiredMemoryReleaseArtifact) []string {
	var issues []string
	byKind := map[string]memoryReleaseArtifact{}
	seenPath := map[string]bool{}
	for _, artifact := range manifest.Artifacts {
		if artifact.Kind == "" {
			issues = append(issues, "release manifest artifact kind is required")
			continue
		}
		if _, ok := byKind[artifact.Kind]; ok {
			issues = append(issues, fmt.Sprintf("duplicate release manifest artifact kind %s", artifact.Kind))
		}
		byKind[artifact.Kind] = artifact
		if err := validateMemoryReleaseSafeRel(artifact.Path); err != nil {
			issues = append(issues, fmt.Sprintf("artifact %s path is invalid: %v", artifact.Kind, err))
			continue
		}
		if seenPath[artifact.Path] {
			issues = append(issues, fmt.Sprintf("duplicate release manifest artifact path %s", artifact.Path))
		}
		seenPath[artifact.Path] = true
		if artifact.Target != manifest.Target {
			issues = append(issues, fmt.Sprintf("%s target is %q, want %q", artifact.Kind, artifact.Target, manifest.Target))
		}
		if strings.TrimSpace(artifact.Command) == "" {
			issues = append(issues, fmt.Sprintf("%s command is required", artifact.Kind))
		}
	}
	for _, req := range required {
		artifact, ok := byKind[req.Kind]
		if !ok {
			issues = append(issues, fmt.Sprintf("missing release manifest artifact %s", req.Kind))
			continue
		}
		if artifact.Path != req.Path {
			issues = append(issues, fmt.Sprintf("%s path is %q, want %s", req.Kind, artifact.Path, req.Path))
		}
		if req.Schema != "" && artifact.Schema != req.Schema {
			issues = append(issues, fmt.Sprintf("%s schema is %q, want %s", req.Kind, artifact.Schema, req.Schema))
		}
		if req.CommandFragment != "" && strings.TrimSpace(artifact.Command) != "" && !strings.Contains(artifact.Command, req.CommandFragment) {
			issues = append(issues, fmt.Sprintf("%s command must contain %q", req.Kind, req.CommandFragment))
		}
		path := filepath.Join(reportDir, filepath.FromSlash(req.Path))
		if _, err := os.Stat(path); err != nil {
			issues = append(issues, fmt.Sprintf("%s artifact %s is missing: %v", req.Kind, req.Path, err))
			continue
		}
		if req.Schema != "" {
			actualSchema := detectMemoryReleaseJSONSchema(path)
			if actualSchema != req.Schema {
				issues = append(issues, fmt.Sprintf("%s artifact schema is %q, want %s", req.Kind, actualSchema, req.Schema))
			}
		}
	}
	return issues
}

func validateMemoryReleaseHashManifest(reportDir string, manifestRel string, required []requiredMemoryReleaseArtifact) []string {
	if manifestRel == "" {
		return []string{"release manifest hash_manifest is required"}
	}
	if err := validateMemoryReleaseSafeRel(manifestRel); err != nil {
		return []string{fmt.Sprintf("release manifest hash_manifest path is invalid: %v", err)}
	}
	hashPath := filepath.Join(reportDir, filepath.FromSlash(manifestRel))
	raw, err := os.ReadFile(hashPath)
	if err != nil {
		return []string{fmt.Sprintf("read hash manifest: %v", err)}
	}
	var manifest memoryReleaseHashManifest
	if err := decodeMemoryReleaseStrictJSON(raw, &manifest); err != nil {
		return []string{fmt.Sprintf("decode hash manifest: %v", err)}
	}

	var issues []string
	if manifest.Schema != memoryReleaseHashSchema {
		issues = append(issues, fmt.Sprintf("artifact-hashes schema is %q, want %s", manifest.Schema, memoryReleaseHashSchema))
	}
	if manifest.Root != "." {
		issues = append(issues, fmt.Sprintf("artifact-hashes root is %q, want .", manifest.Root))
	}
	if len(manifest.Artifacts) == 0 {
		issues = append(issues, "artifact-hashes artifacts must not be empty")
	}

	byPath := map[string]memoryReleaseHashArtifact{}
	lastPath := ""
	for _, artifact := range manifest.Artifacts {
		if err := validateMemoryReleaseSafeRel(artifact.Path); err != nil {
			issues = append(issues, fmt.Sprintf("artifact-hashes path %q is invalid: %v", artifact.Path, err))
			continue
		}
		if lastPath != "" && artifact.Path < lastPath {
			issues = append(issues, "artifact-hashes artifacts must be sorted by path")
		}
		lastPath = artifact.Path
		if _, ok := byPath[artifact.Path]; ok {
			issues = append(issues, fmt.Sprintf("duplicate hash manifest entry for %s", artifact.Path))
		}
		if err := validateMemoryReleaseSHA256(artifact.SHA256, artifact.Path); err != nil {
			issues = append(issues, err.Error())
		}
		byPath[artifact.Path] = artifact
	}

	requiredHashPaths := map[string]bool{"memory-release-manifest.json": true}
	for _, req := range required {
		if req.RequireHash {
			requiredHashPaths[req.Path] = true
		}
	}
	var sortedRequired []string
	for path := range requiredHashPaths {
		sortedRequired = append(sortedRequired, path)
	}
	sort.Strings(sortedRequired)
	for _, rel := range sortedRequired {
		expected, ok := byPath[rel]
		if !ok {
			issues = append(issues, fmt.Sprintf("missing hash manifest entry for %s", rel))
			continue
		}
		actual, err := hashMemoryReleaseFile(reportDir, rel)
		if err != nil {
			issues = append(issues, fmt.Sprintf("hash %s: %v", rel, err))
			continue
		}
		if actual.Size != expected.Size {
			issues = append(issues, fmt.Sprintf("size mismatch for %s: got %d want %d", rel, actual.Size, expected.Size))
		}
		if actual.SHA256 != expected.SHA256 {
			issues = append(issues, fmt.Sprintf("sha256 mismatch for %s: got %s want %s", rel, actual.SHA256, expected.SHA256))
		}
		if actual.Schema != expected.Schema {
			issues = append(issues, fmt.Sprintf("schema mismatch for %s: got %q want %q", rel, actual.Schema, expected.Schema))
		}
	}
	return issues
}

func requiredMemoryReleaseArtifacts() []requiredMemoryReleaseArtifact {
	return []requiredMemoryReleaseArtifact{
		{
			Kind:            "memory_production_report",
			Path:            "memory-production-linux-x64.json",
			Schema:          memoryprod.SchemaV1,
			CommandFragment: "go run ./tools/cmd/memory-production-smoke",
			RequireHash:     true,
		},
		{
			Kind:            "target_report",
			Path:            "targets.json",
			CommandFragment: "go run ./cli/cmd/tetra targets",
			RequireHash:     true,
		},
		{
			Kind:            "memory_fuzz_oracle_report",
			Path:            "memory-fuzz-tier1/memory-fuzz-oracle.json",
			Schema:          "tetra.memory-fuzz.oracle.v1",
			CommandFragment: "go run ./tools/cmd/memory-fuzz-short",
			RequireHash:     true,
		},
		{
			Kind:            "memory_fuzz_summary",
			Path:            "memory-fuzz-tier1/summary.json",
			Schema:          "tetra.memory-fuzz-short.summary.v1",
			CommandFragment: "go run ./tools/cmd/memory-fuzz-short",
			RequireHash:     true,
		},
		{
			Kind:            "memory_fuzz_island_proof_summary",
			Path:            "memory-fuzz-tier1/island-proof-fuzz-summary.json",
			Schema:          "tetra.island-proof-fuzz-summary.v1",
			CommandFragment: "go run ./tools/cmd/memory-fuzz-short",
			RequireHash:     true,
		},
		{
			Kind:            "ram_contract_release_manifest",
			Path:            "ram-contract/ram-contract-release-manifest.json",
			Schema:          "tetra.ram-contract.release-manifest.v1",
			CommandFragment: "ram-contract-linux-x64-smoke.sh",
			RequireHash:     true,
		},
		{
			Kind:            "ram_contract_report",
			Path:            "ram-contract/ram-contract-report.json",
			Schema:          "tetra.ram-contract-report.v1",
			CommandFragment: "ram-contract-linux-x64-smoke.sh",
			RequireHash:     true,
		},
		{
			Kind:            "ram_memory_grade_report",
			Path:            "ram-contract/memory-grade-report.json",
			Schema:          "tetra.memory-grade-report.v1",
			CommandFragment: "ram-contract-linux-x64-smoke.sh",
			RequireHash:     true,
		},
		{
			Kind:            "ram_proof_store_summary",
			Path:            "ram-contract/proof-store-summary.json",
			Schema:          "tetra.proof-store-summary.v1",
			CommandFragment: "ram-contract-linux-x64-smoke.sh",
			RequireHash:     true,
		},
		{
			Kind:            "ram_validation_pipeline_coverage",
			Path:            "ram-contract/validation-pipeline-coverage.json",
			Schema:          "tetra.validation-pipeline-coverage.v1",
			CommandFragment: "ram-contract-linux-x64-smoke.sh",
			RequireHash:     true,
		},
		{
			Kind:            "ram_heap_blockers",
			Path:            "ram-contract/heap-blockers.json",
			Schema:          "tetra.ram-blockers.v1",
			CommandFragment: "ram-contract-linux-x64-smoke.sh",
			RequireHash:     true,
		},
		{
			Kind:            "ram_copy_blockers",
			Path:            "ram-contract/copy-blockers.json",
			Schema:          "tetra.ram-blockers.v1",
			CommandFragment: "ram-contract-linux-x64-smoke.sh",
			RequireHash:     true,
		},
		{
			Kind:            "ram_contract_fuzz_oracle",
			Path:            "ram-contract/fuzz/ram-contract-fuzz-oracle.json",
			Schema:          "tetra.ram-contract-fuzz-oracle.v1",
			CommandFragment: "ram-contract-linux-x64-smoke.sh",
			RequireHash:     true,
		},
		{
			Kind:            "ram_contract_hash_manifest",
			Path:            "ram-contract/artifact-hashes.json",
			Schema:          memoryReleaseHashSchema,
			CommandFragment: "ram-contract-linux-x64-smoke.sh",
			RequireHash:     true,
		},
		{
			Kind:            "island_proof_verifier_report",
			Path:            "island-proof-verifier.json",
			Schema:          islandproof.SchemaV1,
			CommandFragment: "go run ./tools/cmd/validate-island-proof",
			RequireHash:     true,
		},
		{
			Kind:            "island_proof_memory_report",
			Path:            "island-proof-memory-report.json",
			Schema:          "tetra.memory-report.v1",
			CommandFragment: "go run ./tools/cmd/validate-island-proof",
			RequireHash:     true,
		},
		{
			Kind:            "artifact_hash_manifest",
			Path:            "artifact-hashes.json",
			Schema:          memoryReleaseHashSchema,
			CommandFragment: "go run ./tools/cmd/validate-artifact-hashes --write",
			RequireHash:     false,
		},
	}
}

func validateMemoryReleaseIslandProofVerifier(reportDir string, manifest memoryReleaseManifest) []string {
	proofPath := filepath.Join(reportDir, "island-proof-verifier.json")
	memoryPath := filepath.Join(reportDir, "island-proof-memory-report.json")
	proofRaw, err := os.ReadFile(proofPath)
	if err != nil {
		return []string{fmt.Sprintf("island proof verifier artifact island-proof-verifier.json is missing: %v", err)}
	}
	memoryRaw, err := os.ReadFile(memoryPath)
	if err != nil {
		return []string{fmt.Sprintf("island proof verifier memory report island-proof-memory-report.json is missing: %v", err)}
	}
	manifestRaw, err := json.Marshal(manifest)
	if err != nil {
		return []string{fmt.Sprintf("island proof verifier manifest metadata cannot be encoded: %v", err)}
	}
	if err := islandproof.Validate(proofRaw, islandproof.Options{
		MemoryReport:      memoryRaw,
		Manifest:          manifestRaw,
		CurrentGitHead:    manifest.GitHead,
		RequireSameCommit: true,
	}); err != nil {
		return []string{fmt.Sprintf("island proof verifier validation failed: %v", err)}
	}
	return nil
}

func decodeMemoryReleaseStrictJSON(raw []byte, out any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("JSON must contain a single document")
	}
	return nil
}

func validateMemoryReleaseSafeRel(rel string) error {
	if strings.TrimSpace(rel) == "" {
		return fmt.Errorf("path is required")
	}
	if filepath.IsAbs(rel) {
		return fmt.Errorf("absolute paths are not allowed")
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(rel)))
	if clean == "." || clean != rel {
		return fmt.Errorf("path must be clean relative path")
	}
	for _, part := range strings.Split(clean, "/") {
		if part == ".." {
			return fmt.Errorf("parent traversal is not allowed")
		}
	}
	return nil
}

func validateMemoryReleaseSHA256(value string, path string) error {
	if !strings.HasPrefix(value, "sha256:") {
		return fmt.Errorf("artifact %s has invalid sha256 format %q", path, value)
	}
	hexPart := strings.TrimPrefix(value, "sha256:")
	if len(hexPart) != 64 {
		return fmt.Errorf("artifact %s sha256 must contain 64 hex chars", path)
	}
	for _, ch := range hexPart {
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') {
			return fmt.Errorf("artifact %s sha256 has non-hex characters", path)
		}
	}
	return nil
}

func hashMemoryReleaseFile(root string, rel string) (memoryReleaseHashArtifact, error) {
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return memoryReleaseHashArtifact{}, err
	}
	sum := sha256.Sum256(raw)
	return memoryReleaseHashArtifact{
		Path:   rel,
		SHA256: "sha256:" + hex.EncodeToString(sum[:]),
		Size:   int64(len(raw)),
		Schema: detectMemoryReleaseJSONSchema(filepath.Join(root, filepath.FromSlash(rel))),
	}, nil
}

func detectMemoryReleaseJSONSchema(path string) string {
	if filepath.Ext(path) != ".json" {
		return ""
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var envelope struct {
		Schema        string `json:"schema"`
		SchemaVersion string `json:"schema_version"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return ""
	}
	if envelope.Schema != "" {
		return envelope.Schema
	}
	return envelope.SchemaVersion
}

func isMemoryReleaseGitHead(value string) bool {
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

func cleanAbs(path string) string {
	return filepath.Clean(path)
}
