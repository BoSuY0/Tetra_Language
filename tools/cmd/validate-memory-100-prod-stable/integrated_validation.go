package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tetra_language/tools/validators/islandproof"
	"tetra_language/tools/validators/surface"
)

func validateMemory100IntegratedSurfaceEvidence(integratedDir string, gitHead string) []string {
	surfaceDir := filepath.Join(integratedDir, "surface-release-v1")
	var issues []string
	issues = append(issues, validateMemory100IntegratedSurfaceReleaseSummary(filepath.Join(surfaceDir, "surface-release-summary.json"), gitHead)...)
	issues = append(issues, validateMemory100NestedHashManifest(filepath.Join(surfaceDir, "artifact-hashes.json"), surfaceDir, "integrated surface release artifact-hashes.json")...)
	issues = append(issues, validateMemory100IntegratedSurfaceExperimentalRegression(filepath.Join(integratedDir, "surface-experimental-regression"))...)
	issues = append(issues, validateMemory100IntegratedSafeViewLifetimeSummary(filepath.Join(integratedDir, "safe-view-lifetime", "safe-view-lifetime-summary.json"))...)
	issues = append(issues, validateMemory100IntegratedSurfaceAPIStabilitySummary(filepath.Join(integratedDir, "surface-api-stability-v1", "surface-api-stability-summary.json"))...)
	return issues
}

func validateMemory100IntegratedSurfaceExperimentalRegression(dir string) []string {
	issues := validateMemory100NestedHashManifest(filepath.Join(dir, "artifact-hashes.json"), dir, "integrated surface experimental artifact-hashes.json")
	paths, err := listMemory100ArtifactPaths(dir)
	if err != nil {
		return append(issues, fmt.Sprintf("integrated surface experimental artifacts: %v", err))
	}
	for _, rel := range paths {
		if rel == "artifact-hashes.json" || !strings.HasSuffix(rel, ".json") {
			continue
		}
		path := filepath.Join(dir, filepath.FromSlash(rel))
		var envelope memory100SchemaEnvelope
		if err := readMemory100JSON(path, &envelope); err != nil {
			issues = append(issues, fmt.Sprintf("integrated surface experimental JSON %s invalid: %v", rel, err))
			continue
		}
		if memory100SchemaOf(envelope) != surface.SchemaV1 {
			continue
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			issues = append(issues, fmt.Sprintf("integrated surface experimental runtime report %s unreadable: %v", rel, err))
			continue
		}
		if err := surface.ValidateReport(raw); err != nil {
			issues = append(issues, fmt.Sprintf("integrated surface experimental runtime report %s invalid: %v", rel, err))
		}
	}
	return issues
}

func validateMemory100IntegratedSurfaceAPIStabilitySummary(path string) []string {
	var summary struct {
		Schema                string `json:"schema"`
		Status                string `json:"status"`
		ReleaseScope          string `json:"release_scope"`
		DocsManifestValidated bool   `json:"docs_manifest_validated"`
	}
	if err := readMemory100JSON(path, &summary); err != nil {
		return []string{fmt.Sprintf("integrated surface API stability summary missing or invalid: %v", err)}
	}
	var issues []string
	if summary.Schema != "tetra.surface.api-stability.v1" {
		issues = append(issues, fmt.Sprintf("integrated surface API stability summary schema is %q, want tetra.surface.api-stability.v1", summary.Schema))
	}
	if summary.Status != "pass" {
		issues = append(issues, fmt.Sprintf("integrated surface API stability summary status is %q, want pass", summary.Status))
	}
	if summary.ReleaseScope != "surface-v1-linux-web" {
		issues = append(issues, fmt.Sprintf("integrated surface API stability summary release_scope is %q, want surface-v1-linux-web", summary.ReleaseScope))
	}
	if !summary.DocsManifestValidated {
		issues = append(issues, "integrated surface API stability summary docs_manifest_validated must be true")
	}
	return issues
}

func validateMemory100IntegratedSafeViewLifetimeSummary(path string) []string {
	var summary struct {
		Schema          string `json:"schema"`
		Status          string `json:"status"`
		Bounded         bool   `json:"bounded"`
		ReleaseBlocking bool   `json:"release_blocking"`
	}
	if err := readMemory100JSON(path, &summary); err != nil {
		return []string{fmt.Sprintf("integrated safe-view lifetime summary missing or invalid: %v", err)}
	}
	var issues []string
	if summary.Schema != "tetra.safe-view-lifetime.gate.v1" {
		issues = append(issues, fmt.Sprintf("integrated safe-view lifetime summary schema is %q, want tetra.safe-view-lifetime.gate.v1", summary.Schema))
	}
	if summary.Status != "pass" {
		issues = append(issues, fmt.Sprintf("integrated safe-view lifetime summary status is %q, want pass", summary.Status))
	}
	if !summary.Bounded {
		issues = append(issues, "integrated safe-view lifetime summary bounded must be true")
	}
	if !summary.ReleaseBlocking {
		issues = append(issues, "integrated safe-view lifetime summary release_blocking must be true")
	}
	return issues
}

func validateMemory100IntegratedSurfaceReleaseSummary(path string, gitHead string) []string {
	var summary struct {
		Schema       string `json:"schema"`
		Status       string `json:"status"`
		GitHead      string `json:"git_head"`
		ReleaseScope string `json:"release_scope"`
	}
	if err := readMemory100JSON(path, &summary); err != nil {
		return []string{fmt.Sprintf("integrated surface release summary missing or invalid: %v", err)}
	}
	var issues []string
	if summary.Schema != "tetra.surface.release.v1" {
		issues = append(issues, fmt.Sprintf("integrated surface release summary schema is %q, want tetra.surface.release.v1", summary.Schema))
	}
	if summary.Status != "current" {
		issues = append(issues, fmt.Sprintf("integrated surface release summary status is %q, want current", summary.Status))
	}
	if gitHead != "" && summary.GitHead != gitHead {
		issues = append(issues, fmt.Sprintf("integrated surface release summary git_head %s does not match Memory100 git_head %s", summary.GitHead, gitHead))
	}
	if summary.ReleaseScope != "surface-v1-linux-web" {
		issues = append(issues, fmt.Sprintf("integrated surface release summary release_scope is %q, want surface-v1-linux-web", summary.ReleaseScope))
	}
	return issues
}

func validateMemory100IntegratedNestedMemory(integratedDir string, gitHead string) []string {
	memoryDir := filepath.Join(integratedDir, "memory")
	productionPath := filepath.Join(memoryDir, "memory-production-linux-x64.json")
	var issues []string
	for _, issue := range validateMemory100MemoryProductionReport(productionPath) {
		issues = append(issues, "integrated "+issue)
	}
	var envelope memory100SchemaEnvelope
	if err := readMemory100JSON(productionPath, &envelope); err != nil {
		issues = append(issues, fmt.Sprintf("integrated memory production report envelope: %v", err))
	} else {
		if memory100SchemaOf(envelope) != "tetra.memory.production.v1" {
			issues = append(issues, fmt.Sprintf("integrated memory production report schema is %q, want tetra.memory.production.v1", memory100SchemaOf(envelope)))
		}
		if gitHead != "" && envelope.GitHead != gitHead {
			issues = append(issues, fmt.Sprintf("integrated memory production report git_head %s does not match Memory100 git_head %s", envelope.GitHead, gitHead))
		}
	}
	for _, issue := range validateMemory100MemoryReleaseManifest(filepath.Join(memoryDir, "memory-release-manifest.json"), gitHead) {
		issues = append(issues, "integrated "+issue)
	}
	issues = append(issues, validateMemory100IntegratedIslandProofEvidence(memoryDir, gitHead)...)
	for _, issue := range validateMemory100MemoryFuzzBundle(filepath.Join(memoryDir, "memory-fuzz-tier1"), gitHead) {
		issues = append(issues, "integrated "+issue)
	}
	issues = append(issues, validateMemory100NestedHashManifestWithRequired(filepath.Join(memoryDir, "artifact-hashes.json"), memoryDir, "integrated memory artifact-hashes.json", memory100MemoryReleaseRequiredHashPaths())...)
	return issues
}

func memory100MemoryReleaseRequiredHashPaths() []string {
	var paths []string
	for _, required := range requiredMemory100MemoryReleaseArtifacts {
		if required.Path == "artifact-hashes.json" {
			continue
		}
		paths = append(paths, required.Path)
	}
	return paths
}

func validateMemory100IntegratedIslandProofEvidence(memoryDir string, gitHead string) []string {
	return validateMemory100IslandProofEvidence(memoryDir, gitHead, "integrated island proof")
}

func validateMemory100IslandProofEvidence(memoryDir string, gitHead string, label string) []string {
	proofPath := filepath.Join(memoryDir, "island-proof-verifier.json")
	memoryPath := filepath.Join(memoryDir, "island-proof-memory-report.json")
	manifestPath := filepath.Join(memoryDir, "memory-release-manifest.json")
	proofRaw, err := os.ReadFile(proofPath)
	if err != nil {
		return []string{fmt.Sprintf("%s verifier unreadable: %v", label, err)}
	}
	memoryRaw, err := os.ReadFile(memoryPath)
	if err != nil {
		return []string{fmt.Sprintf("%s memory report unreadable: %v", label, err)}
	}
	manifestRaw, err := os.ReadFile(manifestPath)
	if err != nil {
		return []string{fmt.Sprintf("%s release manifest unreadable: %v", label, err)}
	}
	if err := islandproof.Validate(proofRaw, islandproof.Options{
		MemoryReport:      memoryRaw,
		Manifest:          manifestRaw,
		CurrentGitHead:    gitHead,
		RequireSameCommit: gitHead != "",
	}); err != nil {
		return []string{fmt.Sprintf("%s verifier: %v", label, err)}
	}
	return nil
}

func validateMemory100IntegratedIslandsDebugSmoke(path string, gitHead string) []string {
	var report struct {
		Target  string `json:"target"`
		GitHead string `json:"git_head"`
		Islands bool   `json:"islands_debug"`
		Total   *int   `json:"total"`
		Passed  *int   `json:"passed"`
		Failed  *int   `json:"failed"`
		Cases   []struct {
			Name         string `json:"name"`
			SrcPath      string `json:"src_path"`
			ExpectedExit int    `json:"expected_exit"`
			ActualExit   *int   `json:"actual_exit"`
			Ran          bool   `json:"ran"`
			Pass         bool   `json:"pass"`
		} `json:"cases"`
	}
	if err := readMemory100JSON(path, &report); err != nil {
		return []string{fmt.Sprintf("integrated islands debug smoke missing or invalid: %v", err)}
	}
	var issues []string
	if report.Target != "linux-x64" {
		issues = append(issues, fmt.Sprintf("integrated islands debug smoke target is %q, want linux-x64", report.Target))
	}
	if report.GitHead != "" && gitHead != "" && !memory100SameGitHead(report.GitHead, gitHead) {
		issues = append(issues, fmt.Sprintf("integrated islands debug smoke git_head %s does not match Memory100 git_head %s", report.GitHead, gitHead))
	}
	if !report.Islands {
		issues = append(issues, "integrated islands debug smoke islands_debug must be true")
	}
	if report.Total == nil || report.Passed == nil || report.Failed == nil {
		issues = append(issues, "integrated islands debug smoke counts are required")
	} else {
		passed := 0
		for _, c := range report.Cases {
			if c.Pass {
				passed++
			}
		}
		failed := len(report.Cases) - passed
		if *report.Total != len(report.Cases) || *report.Passed != passed || *report.Failed != failed {
			issues = append(issues, fmt.Sprintf("integrated islands debug smoke counts mismatch: got total=%d passed=%d failed=%d computed total=%d passed=%d failed=%d", *report.Total, *report.Passed, *report.Failed, len(report.Cases), passed, failed))
		}
	}
	foundTrap := false
	for _, c := range report.Cases {
		if c.Name != "islands_overflow" {
			continue
		}
		foundTrap = true
		if c.SrcPath != "examples/memory/islands/islands_overflow.tetra" {
			issues = append(issues, fmt.Sprintf("integrated islands debug smoke trap src_path is %q, want examples/memory/islands/islands_overflow.tetra", c.SrcPath))
		}
		if c.ExpectedExit == 0 {
			issues = append(issues, "integrated islands debug smoke trap expected_exit must be non-zero")
		}
		if c.ActualExit == nil {
			issues = append(issues, "integrated islands debug smoke trap missing actual_exit")
		} else if *c.ActualExit != c.ExpectedExit {
			issues = append(issues, fmt.Sprintf("integrated islands debug smoke trap actual_exit=%d, want %d", *c.ActualExit, c.ExpectedExit))
		}
		if !c.Ran || !c.Pass {
			issues = append(issues, "integrated islands debug smoke trap must run and pass")
		}
	}
	if !foundTrap {
		issues = append(issues, "integrated islands debug smoke missing islands_overflow trap")
	}
	return issues
}

func memory100SameGitHead(got string, want string) bool {
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

func validateMemory100IntegratedNestedRAMContract(integratedDir string, gitHead string) []string {
	ramDir := filepath.Join(integratedDir, "memory", "ram-contract")
	return validateMemory100NestedRAMContract(ramDir, gitHead, "integrated")
}

func validateMemory100NestedRAMContract(ramDir string, gitHead string, label string) []string {
	var issues []string
	for _, issue := range validateMemory100RAMContractBundleAt(ramDir, gitHead) {
		issues = append(issues, label+" "+issue)
	}
	for _, issue := range validateMemory100RAMContractReleaseManifest(filepath.Join(ramDir, "ram-contract-release-manifest.json"), gitHead) {
		issues = append(issues, label+" "+issue)
	}
	issues = append(issues, validateMemory100NestedHashManifest(filepath.Join(ramDir, "artifact-hashes.json"), ramDir, label+" RAM contract artifact-hashes.json")...)
	for _, issue := range validateMemory100RAMContractFuzzOracle(filepath.Join(ramDir, "fuzz", "ram-contract-fuzz-oracle.json"), gitHead) {
		issues = append(issues, label+" "+issue)
	}
	return issues
}

func validateMemory100NestedHashManifest(hashPath string, dir string, label string) []string {
	return validateMemory100NestedHashManifestWithRequired(hashPath, dir, label, nil)
}

func validateMemory100NestedHashManifestWithRequired(hashPath string, dir string, label string, requiredPaths []string) []string {
	var manifest memory100HashManifest
	if err := readMemory100StrictJSON(hashPath, &manifest); err != nil {
		return []string{fmt.Sprintf("%s missing or invalid: %v", label, err)}
	}
	var issues []string
	if manifest.Schema != memory100HashSchema {
		issues = append(issues, fmt.Sprintf("%s schema is %q, want %s", label, manifest.Schema, memory100HashSchema))
	}
	if manifest.Root != "." {
		issues = append(issues, fmt.Sprintf("%s root is %q, want .", label, manifest.Root))
	}
	if len(manifest.Artifacts) == 0 {
		issues = append(issues, fmt.Sprintf("%s artifacts must not be empty", label))
	}
	seen := map[string]memory100HashArtifact{}
	lastPath := ""
	for _, artifact := range manifest.Artifacts {
		if err := validateMemory100SafeRel(artifact.Path); err != nil {
			issues = append(issues, fmt.Sprintf("%s path %q is invalid: %v", label, artifact.Path, err))
			continue
		}
		if artifact.Path == "artifact-hashes.json" {
			issues = append(issues, fmt.Sprintf("%s must not list itself", label))
		}
		if lastPath != "" && artifact.Path < lastPath {
			issues = append(issues, fmt.Sprintf("%s artifacts must be sorted by path", label))
		}
		lastPath = artifact.Path
		if _, ok := seen[artifact.Path]; ok {
			issues = append(issues, fmt.Sprintf("duplicate %s entry for %s", label, artifact.Path))
		}
		seen[artifact.Path] = artifact
		if err := validateMemory100SHA256(artifact.SHA256, artifact.Path); err != nil {
			issues = append(issues, err.Error())
		}
		actual, err := hashMemory100File(dir, artifact.Path)
		if err != nil {
			issues = append(issues, fmt.Sprintf("hash %s artifact %s: %v", label, artifact.Path, err))
			continue
		}
		if actual.Size != artifact.Size {
			issues = append(issues, fmt.Sprintf("%s size mismatch for %s: got %d want %d", label, artifact.Path, actual.Size, artifact.Size))
		}
		if actual.SHA256 != artifact.SHA256 {
			issues = append(issues, fmt.Sprintf("%s sha256 mismatch for %s: got %s want %s", label, artifact.Path, actual.SHA256, artifact.SHA256))
		}
		if actual.Schema != artifact.Schema {
			issues = append(issues, fmt.Sprintf("%s schema mismatch for %s: got %q want %q", label, artifact.Path, actual.Schema, artifact.Schema))
		}
	}
	for _, rel := range requiredPaths {
		if _, ok := seen[rel]; !ok {
			issues = append(issues, fmt.Sprintf("missing %s entry for %s", label, rel))
		}
	}
	actualPaths, err := listMemory100ArtifactPaths(dir)
	if err != nil {
		issues = append(issues, fmt.Sprintf("list %s artifacts: %v", label, err))
	} else {
		for _, rel := range actualPaths {
			if _, ok := seen[rel]; !ok {
				issues = append(issues, fmt.Sprintf("unlisted %s artifact %s", label, rel))
			}
		}
	}
	return issues
}

func validateMemory100IntegratedManifest(path string, gitHead string) []string {
	var manifest memory100IntegratedManifest
	if err := readMemory100StrictJSON(path, &manifest); err != nil {
		return []string{fmt.Sprintf("integrated manifest invalid: %v", err)}
	}
	var issues []string
	if manifest.Schema != "tetra.memory-islands-surface.production-gate.v1" {
		issues = append(issues, fmt.Sprintf("integrated manifest schema is %q, want tetra.memory-islands-surface.production-gate.v1", manifest.Schema))
	}
	if manifest.Status != "pass" {
		issues = append(issues, fmt.Sprintf("integrated manifest status is %q, want pass", manifest.Status))
	}
	if gitHead != "" && manifest.GitHead != gitHead {
		issues = append(issues, fmt.Sprintf("integrated manifest git_head %s does not match Memory100 git_head %s", manifest.GitHead, gitHead))
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
	integratedDir := filepath.Dir(path)
	issues = append(issues, validateMemory100GeneratedAtFreshnessWithin(integratedDir, "memory-islands-surface-production-manifest.json", manifest.GeneratedAt, "integrated manifest")...)
	issues = append(issues, validateMemory100IntegratedCommands(manifest.Commands, integratedDir, gitHead)...)
	issues = append(issues, validateMemory100IntegratedArtifactRefs(manifest.Artifacts)...)
	return issues
}

func validateMemory100IntegratedCommands(commands []memory100Command, integratedDir string, gitHead string) []string {
	seen := map[string]string{}
	var issues []string
	for _, command := range commands {
		name := strings.TrimSpace(command.Name)
		text := strings.TrimSpace(command.Command)
		if name == "" {
			issues = append(issues, "integrated manifest command name is required")
			continue
		}
		if _, ok := seen[name]; ok {
			issues = append(issues, fmt.Sprintf("duplicate integrated manifest command %s", name))
		}
		seen[name] = text
		if text == "" {
			issues = append(issues, fmt.Sprintf("integrated manifest command %s command is required", name))
		}
		if strings.Contains(text, "|| true") || strings.Contains(text, "continue-on-error") || strings.Contains(text, "set +e") {
			issues = append(issues, fmt.Sprintf("integrated manifest command %s contains bypass marker", name))
		}
	}
	for name, fragment := range requiredMemory100IntegratedCommands {
		text, ok := seen[name]
		if !ok {
			issues = append(issues, fmt.Sprintf("missing integrated manifest command %s containing %q", name, fragment))
			continue
		}
		if !strings.Contains(text, fragment) {
			issues = append(issues, fmt.Sprintf("integrated manifest command %s must contain %q", name, fragment))
		}
	}
	issues = append(issues, validateMemory100IntegratedCommandProvenance(seen, integratedDir, gitHead)...)
	return issues
}

func validateMemory100IntegratedCommandProvenance(commands map[string]string, integratedDir string, gitHead string) []string {
	type pathRequirement struct {
		name string
		flag string
		rel  string
	}
	pathRequirements := []pathRequirement{
		{name: "memory-production-gate", flag: "--report-dir", rel: "memory"},
		{name: "islands-debug-smoke", flag: "--report", rel: "islands-debug-smoke.json"},
		{name: "validate-islands-debug-smoke", flag: "--report", rel: "islands-debug-smoke.json"},
		{name: "surface-release-gate", flag: "--report-dir", rel: "surface-release-v1"},
		{name: "surface-experimental-regression-gate", flag: "--report-dir", rel: "surface-experimental-regression"},
		{name: "safe-view-lifetime-gate", flag: "--report-dir", rel: "safe-view-lifetime"},
		{name: "surface-api-stability-gate", flag: "--report-dir", rel: "surface-api-stability-v1"},
		{name: "artifact-hashes-write", flag: "--root", rel: ""},
		{name: "artifact-hashes-write", flag: "--out", rel: "artifact-hashes.json"},
		{name: "artifact-hashes-validate", flag: "--manifest", rel: "artifact-hashes.json"},
		{name: "integrated-release-validator", flag: "--report-dir", rel: ""},
	}
	var issues []string
	for _, requirement := range pathRequirements {
		text := strings.TrimSpace(commands[requirement.name])
		if text == "" {
			continue
		}
		wantPath := integratedDir
		if requirement.rel != "" {
			wantPath = filepath.Join(integratedDir, filepath.FromSlash(requirement.rel))
		}
		if !memory100CommandContainsAnyPath(text, requirement.flag, memory100EquivalentPathForms(wantPath)) {
			issues = append(issues, fmt.Sprintf("integrated manifest command %s must use %s under the current integrated report dir for %s", requirement.name, requirement.flag, requirement.rel))
		}
	}
	if text := strings.TrimSpace(commands["integrated-release-validator"]); text != "" && strings.TrimSpace(gitHead) != "" {
		if !strings.Contains(text, "--current-git-head "+gitHead) {
			issues = append(issues, fmt.Sprintf("integrated manifest command integrated-release-validator must use --current-git-head %s", gitHead))
		}
	}
	return issues
}

func validateMemory100IntegratedArtifactRefs(artifacts []memory100IntegratedArtifactRef) []string {
	byKind := map[string]memory100IntegratedArtifactRef{}
	seenPath := map[string]bool{}
	requiredKinds := map[string]bool{}
	for _, required := range requiredMemory100IntegratedArtifacts {
		requiredKinds[required.Kind] = true
	}
	var issues []string
	for _, artifact := range artifacts {
		if err := validateMemory100SafeRel(artifact.Path); err != nil {
			issues = append(issues, fmt.Sprintf("integrated manifest artifact path %q is invalid: %v", artifact.Path, err))
			continue
		}
		if strings.TrimSpace(artifact.Kind) == "" {
			issues = append(issues, fmt.Sprintf("integrated manifest artifact %s kind is required", artifact.Path))
			continue
		}
		if !requiredKinds[artifact.Kind] {
			issues = append(issues, fmt.Sprintf("unexpected integrated manifest artifact kind %s at %s", artifact.Kind, artifact.Path))
		}
		if _, ok := byKind[artifact.Kind]; ok {
			issues = append(issues, fmt.Sprintf("duplicate integrated manifest artifact kind %s", artifact.Kind))
		}
		byKind[artifact.Kind] = artifact
		if seenPath[artifact.Path] {
			issues = append(issues, fmt.Sprintf("duplicate integrated manifest artifact path %s", artifact.Path))
		}
		seenPath[artifact.Path] = true
	}
	for _, required := range requiredMemory100IntegratedArtifacts {
		artifact, ok := byKind[required.Kind]
		if !ok {
			issues = append(issues, fmt.Sprintf("missing integrated manifest artifact %s", required.Kind))
			continue
		}
		if artifact.Path != required.Path {
			issues = append(issues, fmt.Sprintf("integrated manifest artifact %s path is %q, want %s", required.Kind, artifact.Path, required.Path))
		}
		if required.Schema != "" && artifact.Schema != required.Schema {
			issues = append(issues, fmt.Sprintf("integrated manifest artifact %s schema is %q, want %s", required.Kind, artifact.Schema, required.Schema))
		}
	}
	return issues
}
