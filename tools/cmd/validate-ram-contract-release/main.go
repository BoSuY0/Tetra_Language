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
	"strings"

	"tetra_language/tools/internal/ramvalidate"
)

const maxReleaseJSONSchemaSniffBytes = 64 * 1024

var requiredReleaseBundleFiles = []string{
	"ram-contract-release-manifest.json",
	"ram-contract-report.json",
	"memory-grade-report.json",
	"proof-store-summary.json",
	"validation-pipeline-coverage.json",
	"heap-blockers.json",
	"copy-blockers.json",
	"fuzz/ram-contract-fuzz-oracle.json",
	"artifact-hashes.json",
}

var requiredReleaseHashEntries = []string{
	"ram-contract-release-manifest.json",
	"ram-contract-report.json",
	"memory-grade-report.json",
	"proof-store-summary.json",
	"validation-pipeline-coverage.json",
	"heap-blockers.json",
	"copy-blockers.json",
	"fuzz/ram-contract-fuzz-oracle.json",
}

var requiredReleaseManifestArtifacts = []string{
	"ram-contract-report.json",
	"memory-grade-report.json",
	"proof-store-summary.json",
	"validation-pipeline-coverage.json",
	"heap-blockers.json",
	"copy-blockers.json",
	"fuzz/ram-contract-fuzz-oracle.json",
	"artifact-hashes.json",
}

var expectedReleaseManifestArtifactSchemas = map[string]string{
	"ram-contract-report.json":           ramvalidate.ReportSchemaV1,
	"memory-grade-report.json":           ramvalidate.GradeReportSchemaV1,
	"proof-store-summary.json":           ramvalidate.ProofStoreSummarySchemaV1,
	"validation-pipeline-coverage.json":  ramvalidate.PipelineCoverageSchemaV1,
	"heap-blockers.json":                 ramvalidate.BlockerReportSchemaV1,
	"copy-blockers.json":                 ramvalidate.BlockerReportSchemaV1,
	"fuzz/ram-contract-fuzz-oracle.json": "tetra.ram-contract-fuzz-oracle.v1",
	"artifact-hashes.json":               "tetra.release-artifact-hashes.v1alpha1",
}

func main() {
	reportDir := flag.String("report-dir", "", "RAM contract release report directory")
	currentGitHead := flag.String("current-git-head", "", "optional current git HEAD to require")
	flag.Parse()
	if *reportDir == "" {
		fmt.Fprintln(os.Stderr, "error: --report-dir is required")
		os.Exit(2)
	}
	if err := validateRAMContractRelease(*reportDir, *currentGitHead); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateRAMContractRelease(reportDir string, currentGitHead string) error {
	var issues []string
	for _, rel := range requiredReleaseBundleFiles {
		if err := requireReleaseFile(reportDir, rel); err != nil {
			issues = append(issues, err.Error())
		}
	}
	ramPath := filepath.Join(reportDir, "ram-contract-report.json")
	if err := ramvalidate.ValidateReportFile(ramPath); err != nil {
		issues = append(issues, "ram-contract-report.json: "+err.Error())
	}
	if err := ramvalidate.ValidateGradeReportFile(filepath.Join(reportDir, "memory-grade-report.json")); err != nil {
		issues = append(issues, "memory-grade-report.json: "+err.Error())
	}
	if err := ramvalidate.ValidateProofStoreSummaryFile(filepath.Join(reportDir, "proof-store-summary.json")); err != nil {
		issues = append(issues, "proof-store-summary.json: "+err.Error())
	}
	if err := ramvalidate.ValidatePipelineCoverageFile(filepath.Join(reportDir, "validation-pipeline-coverage.json")); err != nil {
		issues = append(issues, "validation-pipeline-coverage.json: "+err.Error())
	}
	if err := ramvalidate.ValidateBlockerReportFile(filepath.Join(reportDir, "heap-blockers.json"), "heap"); err != nil {
		issues = append(issues, "heap-blockers.json: "+err.Error())
	}
	if err := ramvalidate.ValidateBlockerReportFile(filepath.Join(reportDir, "copy-blockers.json"), "copy"); err != nil {
		issues = append(issues, "copy-blockers.json: "+err.Error())
	}
	var report ramvalidate.Report
	if err := ramvalidate.ReadStrictJSONFile(ramPath, &report); err == nil && strings.TrimSpace(currentGitHead) != "" && report.GitHead != strings.TrimSpace(currentGitHead) {
		issues = append(issues, fmt.Sprintf("ram-contract-report git_head %s does not match current git head %s", report.GitHead, strings.TrimSpace(currentGitHead)))
	}
	if err := validateReleaseHashManifest(filepath.Join(reportDir, "artifact-hashes.json")); err != nil {
		issues = append(issues, "artifact-hashes.json: "+err.Error())
	}
	if err := validateReleaseManifest(filepath.Join(reportDir, "ram-contract-release-manifest.json"), currentGitHead); err != nil {
		issues = append(issues, "ram-contract-release-manifest.json: "+err.Error())
	}
	if err := validateReleaseFuzzOracle(filepath.Join(reportDir, "fuzz", "ram-contract-fuzz-oracle.json")); err != nil {
		issues = append(issues, "fuzz/ram-contract-fuzz-oracle.json: "+err.Error())
	}
	if err := validateJSONArtifactGitHeads(reportDir, currentGitHead); err != nil {
		issues = append(issues, err.Error())
	}
	if err := validateReleaseProofStoreCoversRAMReport(reportDir); err != nil {
		issues = append(issues, err.Error())
	}
	if err := validateReleaseCrossFileConsistency(reportDir); err != nil {
		issues = append(issues, err.Error())
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func requireReleaseFile(reportDir string, rel string) error {
	path := filepath.Join(reportDir, filepath.FromSlash(rel))
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("missing required release artifact %s", rel)
		}
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("required release artifact %s is a directory", rel)
	}
	if info.Size() == 0 {
		return fmt.Errorf("required release artifact %s is empty", rel)
	}
	return nil
}

func validateReleaseManifest(path string, currentGitHead string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var manifest struct {
		Schema       string `json:"schema"`
		Status       string `json:"status"`
		Target       string `json:"target"`
		GitHead      string `json:"git_head"`
		GeneratedAt  string `json:"generated_at,omitempty"`
		ReportDir    string `json:"report_dir,omitempty"`
		HashManifest string `json:"hash_manifest"`
		Commands     []struct {
			Name    string `json:"name"`
			Command string `json:"command"`
		} `json:"commands"`
		Artifacts []struct {
			Path   string `json:"path"`
			Kind   string `json:"kind"`
			Schema string `json:"schema"`
		} `json:"artifacts"`
		NonClaims []string `json:"non_claims"`
	}
	if err := decodeReleaseStrictJSON(raw, &manifest); err != nil {
		return err
	}
	var issues []string
	if manifest.Schema != "tetra.ram-contract.release-manifest.v1" {
		issues = append(issues, fmt.Sprintf("schema is %q, want tetra.ram-contract.release-manifest.v1", manifest.Schema))
	}
	if manifest.Status != "pass" {
		issues = append(issues, fmt.Sprintf("status is %q, want pass", manifest.Status))
	}
	if strings.TrimSpace(manifest.Target) == "" {
		issues = append(issues, "target is required")
	}
	if strings.TrimSpace(currentGitHead) != "" && manifest.GitHead != strings.TrimSpace(currentGitHead) {
		issues = append(issues, fmt.Sprintf("git_head %s does not match current git head %s", manifest.GitHead, strings.TrimSpace(currentGitHead)))
	}
	if manifest.HashManifest != "" && manifest.HashManifest != "artifact-hashes.json" {
		issues = append(issues, fmt.Sprintf("hash_manifest is %q, want artifact-hashes.json", manifest.HashManifest))
	}
	if manifest.ReportDir != "" && manifest.ReportDir != "." {
		issues = append(issues, fmt.Sprintf("report_dir is %q, want .", manifest.ReportDir))
	}
	if len(manifest.Commands) == 0 {
		issues = append(issues, "commands are required")
	}
	for i, command := range manifest.Commands {
		if strings.TrimSpace(command.Name) == "" || strings.TrimSpace(command.Command) == "" {
			issues = append(issues, fmt.Sprintf("command %d requires name and command", i))
			continue
		}
		if !releaseCommandHasMachineCheckablePath(command.Command) {
			issues = append(issues, fmt.Sprintf("command %d %q must include a machine-checkable producer or validator command path", i, command.Name))
		}
	}
	seenArtifacts := map[string]bool{}
	for i, artifact := range manifest.Artifacts {
		if strings.TrimSpace(artifact.Path) == "" {
			issues = append(issues, fmt.Sprintf("artifact %d path is required", i))
			continue
		}
		if filepath.IsAbs(artifact.Path) || strings.Contains(artifact.Path, "..") || strings.Contains(artifact.Path, "\\") {
			issues = append(issues, fmt.Sprintf("unsafe artifact path %s", artifact.Path))
			continue
		}
		if seenArtifacts[artifact.Path] {
			issues = append(issues, fmt.Sprintf("duplicate artifact %s", artifact.Path))
			continue
		}
		seenArtifacts[artifact.Path] = true
		if strings.TrimSpace(artifact.Kind) == "" {
			issues = append(issues, fmt.Sprintf("artifact %s kind is required", artifact.Path))
		}
		expectedSchema, ok := expectedReleaseManifestArtifactSchemas[artifact.Path]
		if !ok {
			issues = append(issues, fmt.Sprintf("unexpected artifact %s", artifact.Path))
			continue
		}
		if artifact.Schema != expectedSchema {
			issues = append(issues, fmt.Sprintf("artifact %s schema is %q, want %q", artifact.Path, artifact.Schema, expectedSchema))
		}
	}
	for _, required := range requiredReleaseManifestArtifacts {
		if !seenArtifacts[required] {
			issues = append(issues, fmt.Sprintf("manifest missing artifact %s", required))
		}
	}
	issues = append(issues, ramvalidate.ValidateNonClaims(manifest.NonClaims)...)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func releaseCommandHasMachineCheckablePath(command string) bool {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return false
	}
	if fields[0] != "go" || len(fields) < 3 || fields[1] != "run" {
		return len(fields) >= 2 && fields[0] == "bash" && strings.HasPrefix(fields[1], "scripts/")
	}
	for _, next := range fields[2:] {
		if strings.HasPrefix(next, "-") {
			continue
		}
		return strings.HasPrefix(next, "./tools/cmd/") || next == "./cli/cmd/tetra"
	}
	return false
}

func validateReleaseFuzzOracle(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var oracle struct {
		SchemaVersion string `json:"schema_version"`
		GitHead       string `json:"git_head,omitempty"`
		GeneratedAt   string `json:"generated_at,omitempty"`
		Observations  []struct {
			Mutation         string `json:"mutation"`
			Rejected         bool   `json:"rejected"`
			Validator        string `json:"validator"`
			ValidatorCommand string `json:"validator_command"`
			ExitCode         *int   `json:"exit_code"`
			OutputExcerpt    string `json:"output_excerpt"`
			MutatedFile      string `json:"mutated_file"`
			Reason           string `json:"reason,omitempty"`
		} `json:"observations"`
		Summary struct {
			Mutations int `json:"mutations"`
			Rejected  int `json:"rejected"`
		} `json:"summary"`
		NonClaims []string `json:"non_claims"`
	}
	if err := decodeReleaseStrictJSON(raw, &oracle); err != nil {
		return err
	}
	if oracle.SchemaVersion != "tetra.ram-contract-fuzz-oracle.v1" {
		return fmt.Errorf("schema_version is %q, want tetra.ram-contract-fuzz-oracle.v1", oracle.SchemaVersion)
	}
	rejected := 0
	for _, obs := range oracle.Observations {
		if !obs.Rejected || strings.TrimSpace(obs.Validator) == "" || strings.TrimSpace(obs.ValidatorCommand) == "" || obs.ExitCode == nil || *obs.ExitCode == 0 || strings.TrimSpace(obs.OutputExcerpt) == "" || strings.TrimSpace(obs.MutatedFile) == "" {
			return fmt.Errorf("mutation %s missing validator exit evidence", obs.Mutation)
		}
		rejected++
	}
	if oracle.Summary.Mutations != len(oracle.Observations) || oracle.Summary.Rejected != rejected {
		return fmt.Errorf("summary mismatch")
	}
	if issues := ramvalidate.ValidateNonClaims(oracle.NonClaims); len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func decodeReleaseStrictJSON(raw []byte, out any) error {
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

func validateReleaseHashManifest(path string) error {
	var manifest releaseHashManifest
	if err := ramvalidate.ReadStrictJSONFile(path, &manifest); err != nil {
		return err
	}
	if manifest.Schema != "tetra.release-artifact-hashes.v1alpha1" {
		return fmt.Errorf("schema is %q, want tetra.release-artifact-hashes.v1alpha1", manifest.Schema)
	}
	if manifest.Root == "" || filepath.IsAbs(manifest.Root) || strings.Contains(manifest.Root, "..") {
		return fmt.Errorf("unsafe root %q", manifest.Root)
	}
	root := filepath.Join(filepath.Dir(path), filepath.FromSlash(manifest.Root))
	seen := map[string]bool{}
	lastPath := ""
	for _, artifact := range manifest.Artifacts {
		if artifact.Path == "" {
			return fmt.Errorf("artifact missing path")
		}
		if filepath.IsAbs(artifact.Path) || strings.Contains(artifact.Path, "..") {
			return fmt.Errorf("unsafe artifact path %s", artifact.Path)
		}
		if lastPath != "" && artifact.Path < lastPath {
			return fmt.Errorf("artifacts must be sorted by path: %s appears before %s", artifact.Path, lastPath)
		}
		lastPath = artifact.Path
		if seen[artifact.Path] {
			return fmt.Errorf("duplicate artifact path %s", artifact.Path)
		}
		seen[artifact.Path] = true
		actual, err := hashReleaseArtifact(root, artifact.Path)
		if err != nil {
			return err
		}
		if actual.Size != artifact.Size {
			return fmt.Errorf("size mismatch for %s: got %d want %d", artifact.Path, actual.Size, artifact.Size)
		}
		if actual.SHA256 != artifact.SHA256 {
			return fmt.Errorf("sha256 mismatch for %s: got %s want %s", artifact.Path, actual.SHA256, artifact.SHA256)
		}
		if artifact.Schema != "" && actual.Schema != artifact.Schema {
			return fmt.Errorf("schema mismatch for %s: got %q want %q", artifact.Path, actual.Schema, artifact.Schema)
		}
	}
	for _, required := range requiredReleaseHashEntries {
		if !seen[required] {
			return fmt.Errorf("missing hash entry for %s", required)
		}
	}
	actualPaths, err := listReleaseArtifacts(root, filepath.Base(path))
	if err != nil {
		return err
	}
	for _, actual := range actualPaths {
		if !seen[actual] {
			return fmt.Errorf("unlisted artifact %s", actual)
		}
	}
	return nil
}

type releaseHashManifest struct {
	Schema    string                  `json:"schema"`
	Root      string                  `json:"root"`
	Artifacts []releaseHashedArtifact `json:"artifacts"`
}

type releaseHashedArtifact struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
	Schema string `json:"schema,omitempty"`
}

func hashReleaseArtifact(root string, rel string) (releaseHashedArtifact, error) {
	path := filepath.Join(root, filepath.FromSlash(rel))
	info, err := os.Lstat(path)
	if err != nil {
		return releaseHashedArtifact{}, err
	}
	if !info.Mode().IsRegular() {
		return releaseHashedArtifact{}, fmt.Errorf("artifact %s is not a regular file", rel)
	}
	file, err := os.Open(path)
	if err != nil {
		return releaseHashedArtifact{}, err
	}
	defer file.Close()
	h := sha256.New()
	prefix := newReleaseSchemaSniffPrefix(maxReleaseJSONSchemaSniffBytes)
	size, err := io.Copy(io.MultiWriter(h, prefix), file)
	if err != nil {
		return releaseHashedArtifact{}, err
	}
	return releaseHashedArtifact{
		Path:   filepath.ToSlash(rel),
		SHA256: "sha256:" + hex.EncodeToString(h.Sum(nil)),
		Size:   size,
		Schema: detectReleaseJSONSchemaFromPrefix(path, prefix.Bytes(), size > int64(prefix.Len())),
	}, nil
}

func listReleaseArtifacts(root string, hashManifestName string) ([]string, error) {
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
		if rel == hashManifestName {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink artifact %s is not allowed", rel)
		}
		paths = append(paths, rel)
		return nil
	})
	return paths, err
}

func detectReleaseJSONSchema(path string) string {
	if filepath.Ext(path) != ".json" {
		return ""
	}
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()
	prefix, truncated, err := readReleaseSchemaSniffPrefix(file, maxReleaseJSONSchemaSniffBytes)
	if err != nil {
		return ""
	}
	return detectReleaseJSONSchemaFromPrefix(path, prefix, truncated)
}

func readReleaseSchemaSniffPrefix(r io.Reader, maxBytes int64) ([]byte, bool, error) {
	if maxBytes <= 0 {
		return nil, false, nil
	}
	raw, err := io.ReadAll(io.LimitReader(r, maxBytes+1))
	if err != nil {
		return nil, false, err
	}
	if int64(len(raw)) > maxBytes {
		return raw[:maxBytes], true, nil
	}
	return raw, false, nil
}

func detectReleaseJSONSchemaFromPrefix(path string, prefix []byte, truncated bool) string {
	if filepath.Ext(path) != ".json" {
		return ""
	}
	return detectReleaseJSONSchemaPrefix(prefix, truncated)
}

func detectReleaseJSONSchemaPrefix(prefix []byte, truncated bool) string {
	dec := json.NewDecoder(bytes.NewReader(prefix))
	token, err := dec.Token()
	if err != nil {
		return releaseSchemaSniffAfterError("", truncated)
	}
	delim, ok := token.(json.Delim)
	if !ok || delim != '{' {
		return ""
	}
	var schema string
	var schemaVersion string
	for dec.More() {
		token, err := dec.Token()
		if err != nil {
			return releaseSchemaSniffAfterError(schema, truncated)
		}
		key, ok := token.(string)
		if !ok {
			return ""
		}
		if key == "schema" || key == "schema_version" {
			var raw json.RawMessage
			if err := dec.Decode(&raw); err != nil {
				return releaseSchemaSniffAfterError(schema, truncated)
			}
			value, ok, invalid := releaseSchemaSniffStringValue(raw)
			if invalid {
				return ""
			}
			if key == "schema" {
				if ok {
					schema = value
				}
			} else {
				if ok {
					schemaVersion = value
				}
			}
			continue
		}
		var discard json.RawMessage
		if err := dec.Decode(&discard); err != nil {
			return releaseSchemaSniffAfterError(schema, truncated)
		}
	}
	if _, err := dec.Token(); err != nil {
		return releaseSchemaSniffAfterError(schema, truncated)
	}
	if truncated {
		return ""
	}
	if err := releaseSchemaSniffRequireEOF(dec); err != nil {
		return ""
	}
	if schema != "" {
		return schema
	}
	return schemaVersion
}

type releaseSchemaSniffPrefix struct {
	buf       bytes.Buffer
	remaining int64
}

func newReleaseSchemaSniffPrefix(maxBytes int64) *releaseSchemaSniffPrefix {
	return &releaseSchemaSniffPrefix{remaining: maxBytes}
}

func (w *releaseSchemaSniffPrefix) Write(p []byte) (int, error) {
	if w.remaining > 0 {
		n := len(p)
		if int64(n) > w.remaining {
			n = int(w.remaining)
		}
		_, _ = w.buf.Write(p[:n])
		w.remaining -= int64(n)
	}
	return len(p), nil
}

func (w *releaseSchemaSniffPrefix) Bytes() []byte {
	return w.buf.Bytes()
}

func (w *releaseSchemaSniffPrefix) Len() int {
	return w.buf.Len()
}

func releaseSchemaSniffStringValue(raw json.RawMessage) (string, bool, bool) {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return "", false, false
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", false, true
	}
	return value, true, false
}

func releaseSchemaSniffRequireEOF(dec *json.Decoder) error {
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("release JSON schema sniff must contain a single JSON document")
	}
	return nil
}

func releaseSchemaSniffAfterError(schema string, truncated bool) string {
	if !truncated {
		return ""
	}
	if schema != "" {
		return schema
	}
	return ""
}

func validateJSONArtifactGitHeads(reportDir string, currentGitHead string) error {
	currentGitHead = strings.TrimSpace(currentGitHead)
	if currentGitHead == "" {
		return nil
	}
	var issues []string
	err := filepath.WalkDir(reportDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}
		gitHead, ok, err := readReleaseJSONGitHead(path)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		if gitHead != currentGitHead {
			rel, relErr := filepath.Rel(reportDir, path)
			if relErr != nil {
				rel = path
			}
			issues = append(issues, fmt.Sprintf("%s git_head %s does not match current git head %s", filepath.ToSlash(rel), gitHead, currentGitHead))
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func readReleaseJSONGitHead(path string) (string, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", false, err
	}
	defer file.Close()
	var envelope struct {
		GitHead string `json:"git_head"`
	}
	dec := json.NewDecoder(file)
	if err := dec.Decode(&envelope); err != nil {
		return "", false, nil
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return "", false, nil
	}
	if strings.TrimSpace(envelope.GitHead) == "" {
		return "", false, nil
	}
	return envelope.GitHead, true, nil
}

func validateReleaseProofStoreCoversRAMReport(reportDir string) error {
	var report ramvalidate.Report
	if err := ramvalidate.ReadStrictJSONFile(filepath.Join(reportDir, "ram-contract-report.json"), &report); err != nil {
		return err
	}
	var proofStore ramvalidate.ProofStoreSummary
	if err := ramvalidate.ReadStrictJSONFile(filepath.Join(reportDir, "proof-store-summary.json"), &proofStore); err != nil {
		return err
	}
	proofs := map[string]ramvalidate.ProofSummary{}
	for _, proof := range proofStore.Proofs {
		proofs[proof.ProofID] = proof
	}
	var issues []string
	for i, row := range report.Rows {
		for _, proofID := range row.ProofIDs {
			proof, ok := proofs[proofID]
			if !ok {
				issues = append(issues, fmt.Sprintf("proof-store-summary.json missing proof_id %q referenced by ram-contract-report row %d", proofID, i))
				continue
			}
			if proof.Status == "rejected" || proof.Status == "unknown" {
				issues = append(issues, fmt.Sprintf("proof-store-summary.json proof_id %q referenced by ram-contract-report row %d has unusable status %s", proofID, i, proof.Status))
			}
		}
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateReleaseCrossFileConsistency(reportDir string) error {
	var report ramvalidate.Report
	if err := ramvalidate.ReadStrictJSONFile(filepath.Join(reportDir, "ram-contract-report.json"), &report); err != nil {
		return fmt.Errorf("ram-contract-report.json: %w", err)
	}
	var gradeReport ramvalidate.GradeReport
	if err := ramvalidate.ReadStrictJSONFile(filepath.Join(reportDir, "memory-grade-report.json"), &gradeReport); err != nil {
		return fmt.Errorf("memory-grade-report.json: %w", err)
	}
	var heapBlockers ramvalidate.BlockerReport
	if err := ramvalidate.ReadStrictJSONFile(filepath.Join(reportDir, "heap-blockers.json"), &heapBlockers); err != nil {
		return fmt.Errorf("heap-blockers.json: %w", err)
	}
	var copyBlockers ramvalidate.BlockerReport
	if err := ramvalidate.ReadStrictJSONFile(filepath.Join(reportDir, "copy-blockers.json"), &copyBlockers); err != nil {
		return fmt.Errorf("copy-blockers.json: %w", err)
	}

	var issues []string
	if gradeReport.ArtifactGrade != report.Summary.ArtifactGrade {
		issues = append(issues, fmt.Sprintf("memory-grade-report.json artifact_grade %q does not match RAM report summary artifact_grade %q", gradeReport.ArtifactGrade, report.Summary.ArtifactGrade))
	}
	if !sameReleaseSummary(gradeReport.Summary, report.Summary) {
		issues = append(issues, fmt.Sprintf("memory-grade-report.json summary %+v does not match RAM report summary %+v", gradeReport.Summary, report.Summary))
	}

	rowsBySite := map[string]ramvalidate.Row{}
	ramHeapSites := map[string]ramvalidate.Row{}
	ramCopySites := map[string]ramvalidate.Row{}
	for i, row := range report.Rows {
		if strings.TrimSpace(row.SiteID) == "" {
			continue
		}
		if _, exists := rowsBySite[row.SiteID]; exists {
			issues = append(issues, fmt.Sprintf("ram-contract-report.json row %d duplicate site_id %q", i, row.SiteID))
			continue
		}
		rowsBySite[row.SiteID] = row
		if releaseRowIsHeap(row) {
			ramHeapSites[row.SiteID] = row
		}
		if releaseRowIsCopy(row) {
			ramCopySites[row.SiteID] = row
		}
	}

	heapBlockerSites := map[string]bool{}
	for i, row := range heapBlockers.Rows {
		if strings.TrimSpace(row.SiteID) == "" {
			continue
		}
		if heapBlockerSites[row.SiteID] {
			issues = append(issues, fmt.Sprintf("heap-blockers.json row %d duplicate site_id %q", i, row.SiteID))
			continue
		}
		heapBlockerSites[row.SiteID] = true
		ramRow, ok := rowsBySite[row.SiteID]
		if !ok {
			issues = append(issues, fmt.Sprintf("heap-blockers.json row %d site_id %q missing from ram-contract-report.json", i, row.SiteID))
			continue
		}
		if !releaseRowIsHeap(ramRow) {
			issues = append(issues, fmt.Sprintf("heap-blockers.json row %d site_id %q is not a heap RAM report row", i, row.SiteID))
		}
	}
	for siteID := range ramHeapSites {
		if !heapBlockerSites[siteID] {
			issues = append(issues, fmt.Sprintf("ram-contract-report.json heap row site_id %q missing from heap-blockers.json", siteID))
		}
	}

	copyBlockerSites := map[string]bool{}
	for i, row := range copyBlockers.Rows {
		if strings.TrimSpace(row.SiteID) == "" {
			continue
		}
		if copyBlockerSites[row.SiteID] {
			issues = append(issues, fmt.Sprintf("copy-blockers.json row %d duplicate site_id %q", i, row.SiteID))
			continue
		}
		copyBlockerSites[row.SiteID] = true
		ramRow, ok := rowsBySite[row.SiteID]
		if !ok {
			issues = append(issues, fmt.Sprintf("copy-blockers.json row %d site_id %q missing from ram-contract-report.json", i, row.SiteID))
			continue
		}
		if !releaseRowIsCopy(ramRow) {
			issues = append(issues, fmt.Sprintf("copy-blockers.json row %d site_id %q is not a copy RAM report row", i, row.SiteID))
		}
	}
	for siteID := range ramCopySites {
		if !copyBlockerSites[siteID] {
			issues = append(issues, fmt.Sprintf("ram-contract-report.json copy row site_id %q missing from copy-blockers.json", siteID))
		}
	}

	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func sameReleaseSummary(a ramvalidate.Summary, b ramvalidate.Summary) bool {
	return a.RowCount == b.RowCount &&
		a.ArtifactGrade == b.ArtifactGrade &&
		a.HeapRows == b.HeapRows &&
		a.CopyRows == b.CopyRows &&
		a.UnboundedRows == b.UnboundedRows &&
		a.BudgetBytes == b.BudgetBytes
}

func releaseRowIsHeap(row ramvalidate.Row) bool {
	return row.Placement == "heap_bounded" || row.Placement == "heap_unbounded"
}

func releaseRowIsCopy(row ramvalidate.Row) bool {
	return strings.HasPrefix(row.Intent, "copy")
}
