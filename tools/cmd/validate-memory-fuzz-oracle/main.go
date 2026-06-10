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

	"tetra_language/compiler"
)

type memoryFuzzShortArtifactSummary struct {
	SchemaVersion string                           `json:"schema_version"`
	Kind          string                           `json:"kind"`
	Tier          string                           `json:"tier"`
	Status        string                           `json:"status"`
	Artifacts     map[string]string                `json:"artifacts"`
	Commands      []memoryFuzzShortArtifactCommand `json:"commands"`
	Policies      []string                         `json:"policies,omitempty"`
	NonClaims     []string                         `json:"non_claims,omitempty"`
}

type memoryFuzzShortArtifactCommand struct {
	Name    string `json:"name"`
	Command string `json:"command"`
	Status  string `json:"status"`
}

type memoryFuzzArtifactHashManifest struct {
	Schema    string                     `json:"schema"`
	Root      string                     `json:"root"`
	Artifacts []memoryFuzzHashedArtifact `json:"artifacts"`
}

type memoryFuzzHashedArtifact struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
	Schema string `json:"schema,omitempty"`
}

type islandProofFuzzArtifactSummary struct {
	SchemaVersion string `json:"schema_version"`
	Status        string `json:"status"`
	Corpus        string `json:"corpus,omitempty"`
	Total         int    `json:"total"`
	Rejected      int    `json:"rejected"`
	Accepted      int    `json:"accepted"`
	Cases         []struct {
		Name              string `json:"name"`
		Status            string `json:"status"`
		Mutation          string `json:"mutation,omitempty"`
		Error             string `json:"error,omitempty"`
		ExpectedRejection string `json:"expected_rejection,omitempty"`
	} `json:"cases"`
	NonClaims []string `json:"non_claims,omitempty"`
}

func main() {
	var reportPath string
	var artifactDir string
	flag.StringVar(&reportPath, "report", "", "path to tetra.memory-fuzz.oracle.v1 report")
	flag.StringVar(&artifactDir, "artifact-dir", "", "optional Tier 1 artifact directory to validate alongside the oracle report")
	flag.Parse()
	if reportPath == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	if err := validateMemoryFuzzOracleReportFile(reportPath, artifactDir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateMemoryFuzzOracleReportFile(path string, artifactDirs ...string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var report compiler.MemoryFuzzOracleReport
	if err := decodeStrictMemoryFuzzJSON(raw, &report, "memory fuzz oracle report"); err != nil {
		return err
	}
	if err := compiler.ValidateMemoryFuzzOracleReport(report); err != nil {
		return err
	}
	if len(artifactDirs) == 0 || strings.TrimSpace(artifactDirs[0]) == "" {
		return nil
	}
	return validateMemoryFuzzOracleArtifactDir(path, artifactDirs[0])
}

func validateMemoryFuzzOracleArtifactDir(reportPath string, artifactDir string) error {
	info, err := os.Lstat(artifactDir)
	if err != nil {
		return fmt.Errorf("memory fuzz artifact dir: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("memory fuzz artifact dir %s must not be a symlink", artifactDir)
	}
	if !info.IsDir() {
		return fmt.Errorf("memory fuzz artifact dir %s is not a directory", artifactDir)
	}

	expectedReport := filepath.Join(artifactDir, "memory-fuzz-oracle.json")
	if same, err := sameCleanPath(reportPath, expectedReport); err != nil {
		return err
	} else if !same {
		return fmt.Errorf("--report must point at %s when --artifact-dir is used, got %s", expectedReport, reportPath)
	}
	for _, rel := range []string{"memory-fuzz-oracle.json", "summary.md", "summary.json", "island-proof-fuzz-summary.json", "artifact-hashes.json"} {
		if err := requireMemoryFuzzArtifactFile(artifactDir, rel); err != nil {
			return err
		}
	}
	summaryMD, err := os.ReadFile(filepath.Join(artifactDir, "summary.md"))
	if err != nil {
		return err
	}
	summaryText := string(summaryMD)
	for _, want := range []string{"Memory Fuzz Short Summary", "Tier 1", "memory-fuzz-oracle.json"} {
		if !strings.Contains(summaryText, want) {
			return fmt.Errorf("summary.md missing %q", want)
		}
	}
	raw, err := os.ReadFile(filepath.Join(artifactDir, "summary.json"))
	if err != nil {
		return err
	}
	var summary memoryFuzzShortArtifactSummary
	if err := decodeStrictMemoryFuzzJSON(raw, &summary, "memory fuzz summary.json"); err != nil {
		return err
	}
	if summary.SchemaVersion != "tetra.memory-fuzz-short.summary.v1" {
		return fmt.Errorf("summary.json schema_version = %q, want tetra.memory-fuzz-short.summary.v1", summary.SchemaVersion)
	}
	if summary.Kind != "tier1_short_ci_smoke" || summary.Tier != string(compiler.MemoryFuzzTier1ShortCI) || summary.Status != "pass" {
		return fmt.Errorf("summary.json identity/status must record passing Tier 1 short CI smoke, got kind=%q tier=%q status=%q", summary.Kind, summary.Tier, summary.Status)
	}
	for key, want := range map[string]string{
		"artifact_hashes":           "artifact-hashes.json",
		"oracle_report":             "memory-fuzz-oracle.json",
		"summary_md":                "summary.md",
		"summary_json":              "summary.json",
		"island_proof_fuzz_summary": "island-proof-fuzz-summary.json",
	} {
		got := summary.Artifacts[key]
		if got != want {
			return fmt.Errorf("summary.json artifact %s = %q, want %q", key, got, want)
		}
		if err := requireMemoryFuzzRelativeArtifactPath(got); err != nil {
			return fmt.Errorf("summary.json artifact %s: %w", key, err)
		}
	}
	var sawRunner, sawValidator bool
	for _, command := range summary.Commands {
		if command.Status != "pass" {
			return fmt.Errorf("summary.json command %s status = %q, want pass", command.Name, command.Status)
		}
		switch command.Name {
		case "memory-fuzz-short":
			if strings.Contains(command.Command, "go run ./tools/cmd/memory-fuzz-short") && strings.Contains(command.Command, "--report-dir") {
				sawRunner = true
			}
		case "validate-memory-fuzz-oracle":
			if strings.Contains(command.Command, "go run ./tools/cmd/validate-memory-fuzz-oracle") && strings.Contains(command.Command, "--report") && strings.Contains(command.Command, "--artifact-dir") {
				sawValidator = true
			}
		}
	}
	if !sawRunner {
		return fmt.Errorf("summary.json missing memory-fuzz-short command provenance")
	}
	if !sawValidator {
		return fmt.Errorf("summary.json missing validate-memory-fuzz-oracle command provenance")
	}
	if err := validateIslandProofFuzzArtifactSummary(filepath.Join(artifactDir, "island-proof-fuzz-summary.json")); err != nil {
		return err
	}
	if err := validateMemoryFuzzArtifactHashes(filepath.Join(artifactDir, "artifact-hashes.json")); err != nil {
		return err
	}
	return nil
}

func validateIslandProofFuzzArtifactSummary(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var summary islandProofFuzzArtifactSummary
	if err := decodeStrictMemoryFuzzJSON(raw, &summary, "island-proof-fuzz-summary.json"); err != nil {
		return err
	}
	if summary.SchemaVersion != "tetra.island-proof-fuzz-summary.v1" {
		return fmt.Errorf("island-proof-fuzz-summary.json schema_version = %q, want tetra.island-proof-fuzz-summary.v1", summary.SchemaVersion)
	}
	if summary.Status != "pass" {
		return fmt.Errorf("island-proof-fuzz-summary.json status = %q, want pass", summary.Status)
	}
	if summary.Total < 10 {
		return fmt.Errorf("island-proof-fuzz-summary.json total = %d, want at least 10", summary.Total)
	}
	if summary.Accepted != 0 || summary.Rejected != summary.Total {
		return fmt.Errorf("island-proof-fuzz-summary.json counts total=%d rejected=%d accepted=%d, want all rejected", summary.Total, summary.Rejected, summary.Accepted)
	}
	seen := map[string]bool{}
	for _, c := range summary.Cases {
		if c.Status != "rejected" {
			return fmt.Errorf("island proof fuzz case %s status = %q, want rejected", c.Name, c.Status)
		}
		seen[c.Name] = true
	}
	for _, name := range []string{
		"malformed_proof_json",
		"stale_epoch",
		"mismatched_island_id",
		"wrong_base_allocation",
		"broken_dominance",
		"missing_proof_id",
		"wrong_operation",
		"unsafe_unknown_promotion",
		"noalias_broad_proof",
		"storage_heap_fallback",
		"transform_lost_metadata",
	} {
		if !seen[name] {
			return fmt.Errorf("island-proof-fuzz-summary.json missing mutation case %s", name)
		}
	}
	return nil
}

func validateMemoryFuzzArtifactHashes(manifestPath string) error {
	manifestPath = filepath.Clean(manifestPath)
	if err := rejectMemoryFuzzSymlinkPath(manifestPath, "artifact-hashes.json"); err != nil {
		return err
	}
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	var manifest memoryFuzzArtifactHashManifest
	if err := decodeStrictMemoryFuzzJSON(raw, &manifest, "artifact-hashes.json"); err != nil {
		return err
	}
	if manifest.Schema != "tetra.release-artifact-hashes.v1alpha1" {
		return fmt.Errorf("artifact-hashes.json schema = %q, want tetra.release-artifact-hashes.v1alpha1", manifest.Schema)
	}
	if manifest.Root != "." {
		return fmt.Errorf("artifact-hashes.json root = %q, want .", manifest.Root)
	}
	if len(manifest.Artifacts) == 0 {
		return fmt.Errorf("artifact-hashes.json artifacts must not be empty")
	}
	root := filepath.Dir(manifestPath)
	seen := map[string]bool{}
	lastPath := ""
	for _, expected := range manifest.Artifacts {
		if err := validateMemoryFuzzHashPath(expected.Path); err != nil {
			return err
		}
		if expected.Path == "artifact-hashes.json" {
			return fmt.Errorf("artifact-hashes.json must not list itself")
		}
		if lastPath != "" && expected.Path < lastPath {
			return fmt.Errorf("artifact-hashes.json artifacts must be sorted by path: %s appears before %s", expected.Path, lastPath)
		}
		lastPath = expected.Path
		if seen[expected.Path] {
			return fmt.Errorf("artifact-hashes.json duplicate artifact %s", expected.Path)
		}
		seen[expected.Path] = true
		if expected.Size < 0 {
			return fmt.Errorf("artifact-hashes.json artifact %s has negative size", expected.Path)
		}
		if err := validateMemoryFuzzSHA256(expected.SHA256, expected.Path); err != nil {
			return err
		}
		actual, err := hashMemoryFuzzArtifact(root, expected.Path)
		if err != nil {
			return err
		}
		if actual.Size != expected.Size {
			return fmt.Errorf("artifact-hashes.json size mismatch for %s: got %d want %d", expected.Path, actual.Size, expected.Size)
		}
		if actual.SHA256 != expected.SHA256 {
			return fmt.Errorf("artifact-hashes.json sha256 mismatch for %s: got %s want %s", expected.Path, actual.SHA256, expected.SHA256)
		}
		if actual.Schema != expected.Schema {
			return fmt.Errorf("artifact-hashes.json schema mismatch for %s: got %q want %q", expected.Path, actual.Schema, expected.Schema)
		}
	}
	actualPaths, err := listMemoryFuzzArtifactPaths(root, "artifact-hashes.json")
	if err != nil {
		return err
	}
	for _, path := range actualPaths {
		if !seen[path] {
			return fmt.Errorf("artifact-hashes.json missing listed artifact %s", path)
		}
	}
	return nil
}

func decodeStrictMemoryFuzzJSON(raw []byte, out any, label string) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return fmt.Errorf("%s is malformed: %w", label, err)
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("%s must contain a single JSON document", label)
	}
	return nil
}

func sameCleanPath(a string, b string) (bool, error) {
	absA, err := filepath.Abs(a)
	if err != nil {
		return false, err
	}
	absB, err := filepath.Abs(b)
	if err != nil {
		return false, err
	}
	return filepath.Clean(absA) == filepath.Clean(absB), nil
}

func requireMemoryFuzzArtifactFile(dir string, rel string) error {
	if err := requireMemoryFuzzRelativeArtifactPath(rel); err != nil {
		return err
	}
	path := filepath.Join(dir, rel)
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("missing required memory fuzz artifact %s", rel)
		}
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("required memory fuzz artifact %s is a directory", rel)
	}
	if info.Size() == 0 {
		return fmt.Errorf("required memory fuzz artifact %s is empty", rel)
	}
	return nil
}

func requireMemoryFuzzRelativeArtifactPath(rel string) error {
	if strings.TrimSpace(rel) == "" {
		return fmt.Errorf("path is required")
	}
	if filepath.IsAbs(rel) {
		return fmt.Errorf("path %q must be relative", rel)
	}
	clean := filepath.Clean(rel)
	if clean == "." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
		return fmt.Errorf("path %q must stay inside artifact dir", rel)
	}
	return nil
}

func validateMemoryFuzzHashPath(rel string) error {
	if err := requireMemoryFuzzRelativeArtifactPath(rel); err != nil {
		return err
	}
	if filepath.ToSlash(rel) != rel {
		return fmt.Errorf("artifact-hashes.json path %q must use slash separators", rel)
	}
	return nil
}

func validateMemoryFuzzSHA256(value string, path string) error {
	if !strings.HasPrefix(value, "sha256:") {
		return fmt.Errorf("artifact-hashes.json artifact %s has invalid sha256 format %q", path, value)
	}
	hexPart := strings.TrimPrefix(value, "sha256:")
	if len(hexPart) != 64 {
		return fmt.Errorf("artifact-hashes.json artifact %s sha256 must contain 64 hex chars", path)
	}
	if _, err := hex.DecodeString(hexPart); err != nil {
		return fmt.Errorf("artifact-hashes.json artifact %s sha256 has non-hex characters", path)
	}
	return nil
}

func hashMemoryFuzzArtifact(root string, rel string) (memoryFuzzHashedArtifact, error) {
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := rejectMemoryFuzzSymlinkPath(path, "memory fuzz artifact "+filepath.ToSlash(rel)); err != nil {
		return memoryFuzzHashedArtifact{}, err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return memoryFuzzHashedArtifact{}, err
	}
	if !info.Mode().IsRegular() {
		return memoryFuzzHashedArtifact{}, fmt.Errorf("memory fuzz artifact %s is not a regular file", filepath.ToSlash(rel))
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return memoryFuzzHashedArtifact{}, err
	}
	sum := sha256.Sum256(raw)
	return memoryFuzzHashedArtifact{
		Path:   filepath.ToSlash(rel),
		SHA256: "sha256:" + hex.EncodeToString(sum[:]),
		Size:   int64(len(raw)),
		Schema: detectMemoryFuzzJSONSchema(raw),
	}, nil
}

func listMemoryFuzzArtifactPaths(root string, manifestName string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == manifestName {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("memory fuzz artifact %s must not be a symlink", rel)
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

func detectMemoryFuzzJSONSchema(raw []byte) string {
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

func rejectMemoryFuzzSymlinkPath(path string, label string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s must not be a symlink", label)
	}
	return nil
}
