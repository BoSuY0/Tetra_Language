package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tetra_language/tools/internal/ramvalidate"
	"tetra_language/tools/validators/memoryprod"
)

func main() {
	reportDir := flag.String("report-dir", "", "Memory100 aggregate report directory")
	currentGitHead := flag.String("current-git-head", "", "optional current git HEAD to require")
	flag.Parse()
	if strings.TrimSpace(*reportDir) == "" {
		fmt.Fprintln(os.Stderr, "error: --report-dir is required")
		os.Exit(2)
	}
	if err := validateMemory100ReportDir(*reportDir, *currentGitHead); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateMemory100ReportDir(reportDir string, currentGitHead string) error {
	reportDir = filepath.Clean(reportDir)
	info, err := os.Lstat(reportDir)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("Memory100 report dir must not be a symlink: %s", reportDir)
	}
	if !info.IsDir() {
		return fmt.Errorf("Memory100 report dir is not a directory: %s", reportDir)
	}

	manifestPath := filepath.Join(reportDir, "memory-100-prod-stable-manifest.json")
	var manifest memory100Manifest
	if err := readMemory100StrictJSON(manifestPath, &manifest); err != nil {
		return fmt.Errorf("Memory100 manifest: %w", err)
	}

	var issues []string
	issues = append(issues, validateMemory100ManifestEnvelope(manifest, currentGitHead)...)
	issues = append(issues, validateMemory100GeneratedAtFreshness(reportDir, manifest.GeneratedAt)...)
	issues = append(issues, validateMemory100Commands(manifest.Commands, reportDir, manifest.GitHead)...)
	issues = append(issues, validateMemory100Claims("claims", manifest.Claims, false)...)
	issues = append(issues, validateMemory100Claims("non_claims", manifest.NonClaims, true)...)
	issues = append(issues, validateMemory100Artifacts(reportDir, manifest.Artifacts, manifest.GitHead)...)
	issues = append(issues, validateMemory100RuntimeMemoryContractTargetMatrix(filepath.Join(reportDir, "runtime-memory", "runtime-memory-contract.json"), manifest.GitHead, manifest.TargetMatrix)...)
	issues = append(issues, validateMemory100RAMContractBundle(reportDir, manifest.GitHead)...)
	issues = append(issues, validateMemory100AllocationLoweringRAMConsistency(reportDir)...)
	issues = append(issues, validateMemory100HashManifest(filepath.Join(reportDir, manifest.HashManifest), reportDir)...)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateMemory100ManifestEnvelope(manifest memory100Manifest, currentGitHead string) []string {
	var issues []string
	if manifest.Schema != memory100ManifestSchema {
		issues = append(issues, fmt.Sprintf("Memory100 manifest schema is %q, want %s", manifest.Schema, memory100ManifestSchema))
	}
	if manifest.Status != "pass" {
		issues = append(issues, fmt.Sprintf("Memory100 manifest status is %q, want pass", manifest.Status))
	}
	if strings.TrimSpace(manifest.Verdict) == "" {
		issues = append(issues, "Memory100 manifest verdict is required")
	}
	if !isMemory100GitHead(manifest.GitHead) {
		issues = append(issues, "Memory100 manifest git_head must be a 40-character lowercase hex commit")
	}
	currentGitHead = strings.TrimSpace(currentGitHead)
	if currentGitHead != "" && manifest.GitHead != currentGitHead {
		issues = append(issues, fmt.Sprintf("Memory100 manifest git_head %s does not match current git head %s", manifest.GitHead, currentGitHead))
	}
	if _, err := time.Parse(time.RFC3339, manifest.GeneratedAt); err != nil {
		issues = append(issues, fmt.Sprintf("Memory100 manifest generated_at must be RFC3339: %v", err))
	}
	if manifest.GitDirty == nil {
		issues = append(issues, "Memory100 manifest git_dirty is required")
	} else {
		statusLines := nonEmptyMemory100Strings(manifest.GitStatus)
		if len(statusLines) == 0 {
			issues = append(issues, "Memory100 manifest git_status_short_branch must not be empty")
		} else {
			statusDirty := memory100GitStatusSnapshotDirty(statusLines)
			if *manifest.GitDirty != statusDirty {
				issues = append(issues, fmt.Sprintf("Memory100 manifest git_dirty is %v but git_status_short_branch dirty state is %v", *manifest.GitDirty, statusDirty))
			}
			if *manifest.GitDirty && memory100VerdictClaimsClean(manifest.Verdict) {
				issues = append(issues, fmt.Sprintf("Memory100 manifest verdict %q claims clean/release-candidate status on a dirty checkout", manifest.Verdict))
			}
			issues = append(issues, validateMemory100VerdictDirtyTier(manifest.Verdict, *manifest.GitDirty)...)
		}
	}
	if len(manifest.TargetMatrix) == 0 {
		issues = append(issues, "Memory100 manifest target_matrix must not be empty")
	}
	if manifest.HashManifest != "artifact-hashes.json" {
		issues = append(issues, fmt.Sprintf("Memory100 manifest hash_manifest is %q, want artifact-hashes.json", manifest.HashManifest))
	}
	if len(manifest.Claims) == 0 {
		issues = append(issues, "Memory100 manifest claims must not be empty")
	}
	if len(manifest.NonClaims) == 0 {
		issues = append(issues, "Memory100 manifest non_claims must not be empty")
	}
	return issues
}

func validateMemory100GeneratedAtFreshness(reportDir string, aggregateGeneratedAt string) []string {
	return validateMemory100GeneratedAtFreshnessWithin(reportDir, "memory-100-prod-stable-manifest.json", aggregateGeneratedAt, "Memory100 manifest")
}

func validateMemory100GeneratedAtFreshnessWithin(rootDir string, parentRel string, parentGeneratedAt string, label string) []string {
	parentAt, err := time.Parse(time.RFC3339, parentGeneratedAt)
	if err != nil {
		return nil
	}
	parentRel = filepath.ToSlash(filepath.Clean(filepath.FromSlash(parentRel)))
	var issues []string
	err = filepath.WalkDir(rootDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			issues = append(issues, fmt.Sprintf("%s generated_at freshness walk %s: %v", label, filepath.ToSlash(path), walkErr))
			return nil
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			issues = append(issues, fmt.Sprintf("%s generated_at freshness read %s: %v", label, filepath.ToSlash(path), err))
			return nil
		}
		var obj map[string]any
		if err := json.Unmarshal(raw, &obj); err != nil {
			return nil
		}
		rel, err := filepath.Rel(rootDir, path)
		if err != nil {
			rel = path
		}
		rel = filepath.ToSlash(filepath.Clean(rel))
		if rel == parentRel {
			return nil
		}
		for _, key := range []string{"generated_at", "generated_at_utc"} {
			value, ok := obj[key].(string)
			if !ok || strings.TrimSpace(value) == "" {
				continue
			}
			childAt, err := time.Parse(time.RFC3339, value)
			if err != nil {
				issues = append(issues, fmt.Sprintf("%s child evidence %s %s must be RFC3339: %v", label, filepath.ToSlash(rel), key, err))
				continue
			}
			if childAt.After(parentAt) {
				issues = append(issues, fmt.Sprintf("%s generated_at %s is older than child evidence %s %s %s", label, parentGeneratedAt, filepath.ToSlash(rel), key, value))
			}
		}
		return nil
	})
	if err != nil {
		issues = append(issues, fmt.Sprintf("%s generated_at freshness walk failed: %v", label, err))
	}
	return issues
}

func validateMemory100Commands(commands []memory100Command, reportDir string, gitHead string) []string {
	seen := map[string]string{}
	var issues []string
	for _, command := range commands {
		name := strings.TrimSpace(command.Name)
		text := strings.TrimSpace(command.Command)
		if name == "" {
			issues = append(issues, "Memory100 command name is required")
			continue
		}
		if _, ok := seen[name]; ok {
			issues = append(issues, fmt.Sprintf("duplicate Memory100 command %s", name))
		}
		seen[name] = text
		if text == "" {
			issues = append(issues, fmt.Sprintf("Memory100 command %s command is required", name))
			continue
		}
		if strings.Contains(text, "|| true") || strings.Contains(text, "continue-on-error") || strings.Contains(text, "set +e") {
			issues = append(issues, fmt.Sprintf("Memory100 command %s contains bypass marker", name))
		}
	}
	for name, fragment := range requiredMemory100Commands {
		text, ok := seen[name]
		if !ok {
			issues = append(issues, fmt.Sprintf("missing Memory100 command %s containing %q", name, fragment))
			continue
		}
		if !strings.Contains(text, fragment) {
			issues = append(issues, fmt.Sprintf("Memory100 command %s must contain %q", name, fragment))
		}
	}
	issues = append(issues, validateMemory100CommandProvenance(seen, reportDir, gitHead)...)
	return issues
}

func validateMemory100CommandProvenance(commands map[string]string, reportDir string, gitHead string) []string {
	type pathRequirement struct {
		name string
		flag string
		rel  string
	}
	pathRequirements := []pathRequirement{
		{name: "memory-production-gate", flag: "--report-dir", rel: "memory-production"},
		{name: "ram-contract-gate", flag: "--report-dir", rel: "ram-contract"},
		{name: "integrated-gate", flag: "--report-dir", rel: "integrated"},
		{name: "memory-fuzz-short", flag: "--report-dir", rel: "memory-fuzz"},
		{name: "memory-fuzz-validator", flag: "--report", rel: "memory-fuzz/memory-fuzz-oracle.json"},
		{name: "memory-fuzz-validator", flag: "--artifact-dir", rel: "memory-fuzz"},
		{name: "artifact-hashes-write", flag: "--root", rel: ""},
		{name: "artifact-hashes-write", flag: "--out", rel: "artifact-hashes.json"},
		{name: "memory-100-validator", flag: "--report-dir", rel: ""},
	}
	var issues []string
	for _, requirement := range pathRequirements {
		text := strings.TrimSpace(commands[requirement.name])
		if text == "" {
			continue
		}
		wantPath := reportDir
		if requirement.rel != "" {
			wantPath = filepath.Join(reportDir, filepath.FromSlash(requirement.rel))
		}
		if !memory100CommandContainsAnyPath(text, requirement.flag, memory100EquivalentPathForms(wantPath)) {
			issues = append(issues, fmt.Sprintf("Memory100 command %s must use %s under the current report dir for %s", requirement.name, requirement.flag, requirement.rel))
		}
	}
	for _, requirement := range []struct {
		name string
		flag string
	}{
		{name: "memory-fuzz-short", flag: "--git-head"},
		{name: "memory-fuzz-validator", flag: "--current-git-head"},
		{name: "memory-100-validator", flag: "--current-git-head"},
	} {
		text := strings.TrimSpace(commands[requirement.name])
		if text == "" || strings.TrimSpace(gitHead) == "" {
			continue
		}
		if !strings.Contains(text, requirement.flag+" "+gitHead) {
			issues = append(issues, fmt.Sprintf("Memory100 command %s must use %s %s", requirement.name, requirement.flag, gitHead))
		}
	}
	return issues
}

func validateMemory100Artifacts(reportDir string, artifacts []memory100Artifact, gitHead string) []string {
	byKind := map[string]memory100Artifact{}
	seenPath := map[string]bool{}
	requiredKinds := map[string]bool{}
	for _, required := range requiredMemory100Artifacts {
		requiredKinds[required.Kind] = true
	}
	var issues []string
	for _, artifact := range artifacts {
		if err := validateMemory100SafeRel(artifact.Path); err != nil {
			issues = append(issues, fmt.Sprintf("Memory100 artifact path %q is invalid: %v", artifact.Path, err))
			continue
		}
		if strings.TrimSpace(artifact.Kind) == "" {
			issues = append(issues, fmt.Sprintf("Memory100 artifact %s kind is required", artifact.Path))
			continue
		}
		if !requiredKinds[artifact.Kind] {
			issues = append(issues, fmt.Sprintf("unexpected Memory100 artifact kind %s at %s", artifact.Kind, artifact.Path))
		}
		if _, ok := byKind[artifact.Kind]; ok {
			issues = append(issues, fmt.Sprintf("duplicate Memory100 artifact kind %s", artifact.Kind))
		}
		byKind[artifact.Kind] = artifact
		if seenPath[artifact.Path] {
			issues = append(issues, fmt.Sprintf("duplicate Memory100 artifact path %s", artifact.Path))
		}
		seenPath[artifact.Path] = true
	}
	for _, required := range requiredMemory100Artifacts {
		artifact, ok := byKind[required.Kind]
		if !ok {
			issues = append(issues, fmt.Sprintf("missing Memory100 artifact %s", required.Kind))
			continue
		}
		if artifact.Path != required.Path {
			issues = append(issues, fmt.Sprintf("Memory100 artifact %s path is %q, want %s", required.Kind, artifact.Path, required.Path))
		}
		if artifact.Schema != required.Schema {
			issues = append(issues, fmt.Sprintf("Memory100 artifact %s schema is %q, want %s", required.Kind, artifact.Schema, required.Schema))
		}
		issues = append(issues, validateMemory100ArtifactFile(reportDir, required, gitHead)...)
	}
	return issues
}

func validateMemory100ArtifactFile(reportDir string, required memory100RequiredArtifact, gitHead string) []string {
	path := filepath.Join(reportDir, filepath.FromSlash(required.Path))
	info, err := os.Lstat(path)
	if err != nil {
		return []string{fmt.Sprintf("%s artifact %s is missing: %v", required.Kind, required.Path, err)}
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return []string{fmt.Sprintf("%s artifact %s must not be a symlink", required.Kind, required.Path)}
	}
	if !info.Mode().IsRegular() {
		return []string{fmt.Sprintf("%s artifact %s is not a regular file", required.Kind, required.Path)}
	}
	if info.Size() == 0 {
		return []string{fmt.Sprintf("%s artifact %s is empty", required.Kind, required.Path)}
	}
	var envelope memory100SchemaEnvelope
	if err := readMemory100JSON(path, &envelope); err != nil {
		return []string{fmt.Sprintf("%s artifact %s is invalid JSON: %v", required.Kind, required.Path, err)}
	}
	var issues []string
	if memory100SchemaOf(envelope) != required.Schema {
		issues = append(issues, fmt.Sprintf("%s artifact schema is %q, want %s", required.Kind, memory100SchemaOf(envelope), required.Schema))
	}
	if required.Schema != memory100HashSchema {
		if !isMemory100GitHead(envelope.GitHead) {
			issues = append(issues, fmt.Sprintf("%s artifact git_head must be a 40-character lowercase hex commit", required.Kind))
		} else if gitHead != "" && envelope.GitHead != gitHead {
			issues = append(issues, fmt.Sprintf("%s artifact git_head %s does not match Memory100 git_head %s", required.Kind, envelope.GitHead, gitHead))
		}
	}
	issues = append(issues, validateMemory100ArtifactContent(path, required.Kind, gitHead)...)
	return issues
}

func validateMemory100ArtifactContent(path string, kind string, gitHead string) []string {
	switch kind {
	case "memory_production_report":
		return validateMemory100MemoryProductionReport(path)
	case "memory_release_manifest":
		return validateMemory100MemoryReleaseManifest(path, gitHead)
	case "ram_contract_release_manifest":
		return validateMemory100RAMContractReleaseManifest(path, gitHead)
	case "memory_production_hash_manifest":
		return validateMemory100NestedHashManifestWithRequired(path, filepath.Dir(path), "memory production artifact-hashes.json", memory100MemoryReleaseRequiredHashPaths())
	case "ram_contract_hash_manifest":
		return validateMemory100NestedHashManifest(path, filepath.Dir(path), "ram contract artifact-hashes.json")
	case "ram_contract_fuzz_oracle":
		return validateMemory100RAMContractFuzzOracle(path, gitHead)
	case "raw_memory_contract_report":
		return validateMemory100RawMemoryContract(path, gitHead)
	case "allocation_lowering_report":
		return validateMemory100AllocationLowering(path, gitHead)
	case "proof_store_summary":
		return validateMemory100ProofStoreSummary(path)
	case "memory_fuzz_oracle_report":
		return validateMemory100MemoryFuzzBundle(filepath.Dir(path), gitHead)
	case "proof_transition_report":
		return validateMemory100ProofTransitionReport(path, gitHead)
	case "runtime_memory_contract":
		return validateMemory100RuntimeMemoryContract(path, gitHead, nil)
	case "memory_semantic_safety_matrix":
		return validateMemory100SemanticSafetyMatrix(path, gitHead)
	case "leak_resource_report":
		return validateMemory100LeakResource(path, gitHead)
	case "integrated_memory_islands_surface_manifest":
		return validateMemory100IntegratedManifest(path, gitHead)
	case "integrated_hash_manifest":
		issues := validateMemory100NestedHashManifestWithRequired(path, filepath.Dir(path), "integrated artifact-hashes.json", requiredMemory100IntegratedHashPaths)
		issues = append(issues, validateMemory100IntegratedNestedMemory(filepath.Dir(path), gitHead)...)
		issues = append(issues, validateMemory100IntegratedNestedRAMContract(filepath.Dir(path), gitHead)...)
		issues = append(issues, validateMemory100IntegratedIslandsDebugSmoke(filepath.Join(filepath.Dir(path), "islands-debug-smoke.json"), gitHead)...)
		issues = append(issues, validateMemory100IntegratedSurfaceEvidence(filepath.Dir(path), gitHead)...)
		return issues
	case "docs_claim_policy":
		return validateMemory100ClaimPolicyArtifact(path, gitHead)
	default:
		return nil
	}
}

func validateMemory100RAMContractFuzzOracle(path string, gitHead string) []string {
	var report struct {
		SchemaVersion string `json:"schema_version"`
		GitHead       string `json:"git_head"`
		GeneratedAt   string `json:"generated_at"`
		Observations  []struct {
			Mutation         string `json:"mutation"`
			Rejected         bool   `json:"rejected"`
			Validator        string `json:"validator"`
			ValidatorCommand string `json:"validator_command"`
			ExitCode         *int   `json:"exit_code"`
			OutputExcerpt    string `json:"output_excerpt"`
			MutatedFile      string `json:"mutated_file"`
			Reason           string `json:"reason"`
		} `json:"observations"`
		Summary struct {
			Mutations int `json:"mutations"`
			Rejected  int `json:"rejected"`
		} `json:"summary"`
		NonClaims []string `json:"non_claims"`
	}
	if err := readMemory100StrictJSON(path, &report); err != nil {
		return []string{fmt.Sprintf("RAM contract fuzz oracle invalid: %v", err)}
	}
	var issues []string
	if report.SchemaVersion != "tetra.ram-contract-fuzz-oracle.v1" {
		issues = append(issues, fmt.Sprintf("RAM contract fuzz oracle schema_version is %q, want tetra.ram-contract-fuzz-oracle.v1", report.SchemaVersion))
	}
	if gitHead != "" && report.GitHead != gitHead {
		issues = append(issues, fmt.Sprintf("RAM contract fuzz oracle git_head %s does not match Memory100 git_head %s", report.GitHead, gitHead))
	}
	if _, err := time.Parse(time.RFC3339, report.GeneratedAt); err != nil {
		issues = append(issues, fmt.Sprintf("RAM contract fuzz oracle generated_at must be RFC3339: %v", err))
	}
	if len(nonEmptyMemory100Strings(report.NonClaims)) == 0 {
		issues = append(issues, "RAM contract fuzz oracle non_claims must not be empty")
	}
	issues = append(issues, validateMemory100Claims("RAM contract fuzz oracle non_claims", report.NonClaims, true)...)

	required := map[string]bool{
		"mutated_proof_id":        false,
		"widened_grade":           false,
		"missing_blocker":         false,
		"budget_drift":            false,
		"artifact_hash_drift":     false,
		"forbidden_nonclaim_text": false,
	}
	rejected := 0
	for i, obs := range report.Observations {
		mutation := strings.TrimSpace(obs.Mutation)
		if mutation == "" {
			issues = append(issues, fmt.Sprintf("RAM contract fuzz observation %d missing mutation", i))
			continue
		}
		if _, ok := required[mutation]; ok {
			required[mutation] = true
		}
		if !obs.Rejected {
			issues = append(issues, fmt.Sprintf("RAM contract fuzz mutation %s was not rejected", mutation))
		} else {
			rejected++
		}
		if strings.TrimSpace(obs.Validator) == "" || strings.TrimSpace(obs.ValidatorCommand) == "" || strings.TrimSpace(obs.Reason) == "" {
			issues = append(issues, fmt.Sprintf("RAM contract fuzz mutation %s missing validator command/reason evidence", mutation))
		}
		if obs.ExitCode == nil || *obs.ExitCode == 0 {
			issues = append(issues, fmt.Sprintf("RAM contract fuzz mutation %s missing nonzero exit evidence", mutation))
		}
		if strings.TrimSpace(obs.OutputExcerpt) == "" || strings.TrimSpace(obs.MutatedFile) == "" {
			issues = append(issues, fmt.Sprintf("RAM contract fuzz mutation %s missing output or mutated_file evidence", mutation))
		}
	}
	for mutation, seen := range required {
		if !seen {
			issues = append(issues, fmt.Sprintf("RAM contract fuzz oracle missing mutation class %s", mutation))
		}
	}
	if report.Summary.Mutations != len(report.Observations) || report.Summary.Rejected != rejected {
		issues = append(issues, fmt.Sprintf("RAM contract fuzz oracle summary mismatch: mutations=%d rejected=%d observations=%d counted_rejected=%d", report.Summary.Mutations, report.Summary.Rejected, len(report.Observations), rejected))
	}
	return issues
}

func validateMemory100ProofStoreSummary(path string) []string {
	if err := ramvalidate.ValidateProofStoreSummaryFile(path); err != nil {
		return []string{fmt.Sprintf("proof store summary: %v", err)}
	}
	return nil
}

func validateMemory100MemoryProductionReport(path string) []string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return []string{fmt.Sprintf("memory production report unreadable: %v", err)}
	}
	if err := memoryprod.ValidateReport(raw); err != nil {
		return []string{fmt.Sprintf("memory production report: %v", err)}
	}
	return nil
}

func validateMemory100MemoryReleaseManifest(path string, gitHead string) []string {
	var manifest memory100MemoryReleaseManifest
	if err := readMemory100StrictJSON(path, &manifest); err != nil {
		return []string{fmt.Sprintf("memory release manifest invalid: %v", err)}
	}
	var issues []string
	if manifest.Schema != "tetra.memory.release-manifest.v1" {
		issues = append(issues, fmt.Sprintf("memory release manifest schema is %q, want tetra.memory.release-manifest.v1", manifest.Schema))
	}
	if manifest.Target != "linux-x64" {
		issues = append(issues, fmt.Sprintf("memory release manifest target is %q, want linux-x64", manifest.Target))
	}
	if gitHead != "" && manifest.GitHead != gitHead {
		issues = append(issues, fmt.Sprintf("memory release manifest git_head %s does not match Memory100 git_head %s", manifest.GitHead, gitHead))
	}
	if _, err := time.Parse(time.RFC3339, manifest.GeneratedAt); err != nil {
		issues = append(issues, fmt.Sprintf("memory release manifest generated_at must be RFC3339: %v", err))
	}
	if manifest.ReportDir != "." {
		issues = append(issues, fmt.Sprintf("memory release manifest report_dir is %q, want .", manifest.ReportDir))
	}
	if manifest.HashManifest != "artifact-hashes.json" {
		issues = append(issues, fmt.Sprintf("memory release manifest hash_manifest is %q, want artifact-hashes.json", manifest.HashManifest))
	}
	memoryDir := filepath.Dir(path)
	issues = append(issues, validateMemory100GeneratedAtFreshnessWithin(memoryDir, "memory-release-manifest.json", manifest.GeneratedAt, "memory release manifest")...)
	issues = append(issues, validateMemory100MemoryReleaseCommands(manifest.Commands, memoryDir, gitHead)...)
	issues = append(issues, validateMemory100MemoryReleaseArtifactRefs(manifest.Artifacts, memoryDir, gitHead)...)
	issues = append(issues, validateMemory100MemoryReleaseNestedEvidence(memoryDir, gitHead)...)
	return issues
}

func validateMemory100MemoryReleaseNestedEvidence(memoryDir string, gitHead string) []string {
	var issues []string
	issues = append(issues, validateMemory100TargetsReport(filepath.Join(memoryDir, "targets.json"), "memory release targets.json")...)
	for _, issue := range validateMemory100MemoryFuzzBundle(filepath.Join(memoryDir, "memory-fuzz-tier1"), gitHead) {
		issues = append(issues, "memory release "+issue)
	}
	issues = append(issues, validateMemory100IslandProofEvidence(memoryDir, gitHead, "memory release island proof")...)
	issues = append(issues, validateMemory100NestedRAMContract(filepath.Join(memoryDir, "ram-contract"), gitHead, "memory release")...)
	return issues
}

func validateMemory100TargetsReport(path string, label string) []string {
	var report memory100TargetsReport
	if err := readMemory100StrictJSON(path, &report); err != nil {
		return []string{fmt.Sprintf("%s missing or invalid: %v", label, err)}
	}
	var issues []string
	issues = append(issues, validateMemory100StringSequence(label+" supported", report.Supported, []string{"linux-x64", "windows-x64", "macos-x64", "wasm32-wasi", "wasm32-web"})...)
	issues = append(issues, validateMemory100StringSequence(label+" build_only", report.BuildOnly, []string{"linux-x86", "linux-x32"})...)
	issues = append(issues, validateMemory100StringSequence(label+" planned", report.Planned, nil)...)

	expected := []struct {
		triple                  string
		status                  string
		os                      string
		arch                    string
		abi                     string
		format                  string
		exeExt                  string
		buildOnly               bool
		runMode                 string
		supportsDebugInfo       bool
		supportsReleaseOptimize bool
	}{
		{triple: "linux-x64", status: "supported", os: "linux", arch: "x64", abi: "sysv", format: "elf", exeExt: "", buildOnly: false, runMode: "host_native", supportsDebugInfo: true, supportsReleaseOptimize: true},
		{triple: "windows-x64", status: "supported", os: "windows", arch: "x64", abi: "win64", format: "pe", exeExt: ".exe", buildOnly: false, runMode: "host_native", supportsDebugInfo: true, supportsReleaseOptimize: true},
		{triple: "macos-x64", status: "supported", os: "macos", arch: "x64", abi: "sysv", format: "macho", exeExt: "", buildOnly: false, runMode: "host_native", supportsDebugInfo: true, supportsReleaseOptimize: true},
		{triple: "wasm32-wasi", status: "supported", os: "wasi", arch: "wasm32", abi: "wasi", format: "wasm", exeExt: ".wasm", buildOnly: false, runMode: "wasi_runner", supportsDebugInfo: false, supportsReleaseOptimize: true},
		{triple: "wasm32-web", status: "supported", os: "web", arch: "wasm32", abi: "web", format: "wasm", exeExt: ".wasm", buildOnly: false, runMode: "web_runner", supportsDebugInfo: false, supportsReleaseOptimize: true},
		{triple: "linux-x86", status: "build_only", os: "linux", arch: "x86", abi: "i386-sysv", format: "elf", exeExt: "", buildOnly: true, runMode: "host_probed", supportsDebugInfo: false, supportsReleaseOptimize: false},
		{triple: "linux-x32", status: "build_only", os: "linux", arch: "x64", abi: "x32-sysv", format: "elf", exeExt: "", buildOnly: true, runMode: "host_probed", supportsDebugInfo: false, supportsReleaseOptimize: false},
	}
	if len(report.Targets) != len(expected) {
		issues = append(issues, fmt.Sprintf("%s target metadata count = %d, want %d", label, len(report.Targets), len(expected)))
	}
	seen := map[string]bool{}
	for i, want := range expected {
		if i >= len(report.Targets) {
			break
		}
		row := report.Targets[i]
		if seen[row.Triple] {
			issues = append(issues, fmt.Sprintf("%s target metadata %q is duplicated", label, row.Triple))
		}
		seen[row.Triple] = true
		if row.Triple != want.triple {
			issues = append(issues, fmt.Sprintf("%s target metadata[%d].triple = %q, want %q", label, i, row.Triple, want.triple))
			continue
		}
		if row.Status != want.status {
			issues = append(issues, fmt.Sprintf("%s target metadata[%s].status = %q, want %q", label, row.Triple, row.Status, want.status))
		}
		if row.OS != want.os || row.Arch != want.arch || row.ABI != want.abi || row.Format != want.format {
			issues = append(issues, fmt.Sprintf("%s target metadata[%s] platform = os:%s arch:%s abi:%s format:%s, want os:%s arch:%s abi:%s format:%s", label, row.Triple, row.OS, row.Arch, row.ABI, row.Format, want.os, want.arch, want.abi, want.format))
		}
		if row.ExeExt != want.exeExt {
			issues = append(issues, fmt.Sprintf("%s target metadata[%s].exe_ext = %q, want %q", label, row.Triple, row.ExeExt, want.exeExt))
		}
		if row.BuildOnly != want.buildOnly {
			issues = append(issues, fmt.Sprintf("%s target metadata[%s].build_only = %v, want %v", label, row.Triple, row.BuildOnly, want.buildOnly))
		}
		if row.RunMode != want.runMode {
			issues = append(issues, fmt.Sprintf("%s target metadata[%s].run_mode = %q, want %q", label, row.Triple, row.RunMode, want.runMode))
		}
		if row.SupportsDebugInfo != want.supportsDebugInfo {
			issues = append(issues, fmt.Sprintf("%s target metadata[%s].supports_debug_info = %v, want %v", label, row.Triple, row.SupportsDebugInfo, want.supportsDebugInfo))
		}
		if row.SupportsReleaseOptimize != want.supportsReleaseOptimize {
			issues = append(issues, fmt.Sprintf("%s target metadata[%s].supports_release_optimize = %v, want %v", label, row.Triple, row.SupportsReleaseOptimize, want.supportsReleaseOptimize))
		}
		issues = append(issues, validateMemory100TargetMemoryClaims(label, row)...)
	}
	return issues
}

func validateMemory100StringSequence(label string, got []string, want []string) []string {
	if len(got) != len(want) {
		return []string{fmt.Sprintf("%s count = %d, want %d", label, len(got), len(want))}
	}
	var issues []string
	seen := map[string]bool{}
	for i := range want {
		if got[i] != want[i] {
			issues = append(issues, fmt.Sprintf("%s[%d] = %q, want %q", label, i, got[i], want[i]))
		}
		if seen[got[i]] {
			issues = append(issues, fmt.Sprintf("%s %q is duplicated", label, got[i]))
		}
		seen[got[i]] = true
	}
	return issues
}

func validateMemory100TargetMemoryClaims(label string, row memory100TargetsReportRow) []string {
	var issues []string
	if row.MemoryBuild != "yes" {
		issues = append(issues, fmt.Sprintf("%s target metadata[%s].memory_build = %q, want yes", label, row.Triple, row.MemoryBuild))
	}
	if row.MemoryLower != "yes" {
		issues = append(issues, fmt.Sprintf("%s target metadata[%s].memory_lower = %q, want yes", label, row.Triple, row.MemoryLower))
	}
	switch row.Triple {
	case "linux-x64":
		if row.RuntimeStatus != "production" {
			issues = append(issues, fmt.Sprintf("%s target metadata[linux-x64].runtime_status = %q, want production", label, row.RuntimeStatus))
		}
		if row.StdlibStatus != "production" {
			issues = append(issues, fmt.Sprintf("%s target metadata[linux-x64].stdlib_status = %q, want production", label, row.StdlibStatus))
		}
		if row.MemoryRun != "yes" {
			issues = append(issues, fmt.Sprintf("%s target metadata[linux-x64].memory_run = %q, want yes", label, row.MemoryRun))
		}
		if row.MemoryRawDiagnostics != "yes" {
			issues = append(issues, fmt.Sprintf("%s target metadata[linux-x64].memory_raw_diagnostics = %q, want yes", label, row.MemoryRawDiagnostics))
		}
		if row.MemoryRegionLowering != "yes/partial" {
			issues = append(issues, fmt.Sprintf("%s target metadata[linux-x64].memory_region_lowering = %q, want yes/partial", label, row.MemoryRegionLowering))
		}
		if row.MemoryAlignmentSemantics != "yes" {
			issues = append(issues, fmt.Sprintf("%s target metadata[linux-x64].memory_alignment_semantics = %q, want yes", label, row.MemoryAlignmentSemantics))
		}
		if row.MemoryClaimLevel != "production/host_runtime" {
			issues = append(issues, fmt.Sprintf("%s target metadata[linux-x64].memory_claim_level = %q, want production/host_runtime", label, row.MemoryClaimLevel))
		}
		for _, required := range []string{"targets.json", "linux-x64-runner.json", "linux-x64-abi.json"} {
			if !memory100StringSliceHas(row.EvidenceArtifacts, required) {
				issues = append(issues, fmt.Sprintf("%s target metadata[linux-x64].evidence_artifacts missing %s", label, required))
			}
		}
	case "linux-x86", "linux-x32":
		if row.RuntimeStatus != "partial_build_only" {
			issues = append(issues, fmt.Sprintf("%s target metadata[%s].runtime_status = %q, want partial_build_only", label, row.Triple, row.RuntimeStatus))
		}
		if row.StdlibStatus != "partial_build_only" {
			issues = append(issues, fmt.Sprintf("%s target metadata[%s].stdlib_status = %q, want partial_build_only", label, row.Triple, row.StdlibStatus))
		}
		if row.MemoryRun != "no/host-dependent" {
			issues = append(issues, fmt.Sprintf("%s target metadata[%s].memory_run = %q, want no/host-dependent", label, row.Triple, row.MemoryRun))
		}
		if row.MemoryClaimLevel != "build_lower_only" {
			issues = append(issues, fmt.Sprintf("%s target metadata[%s].memory_claim_level = %q, want build_lower_only", label, row.Triple, row.MemoryClaimLevel))
		}
		if row.RunnerProbeCommand == "" {
			issues = append(issues, fmt.Sprintf("%s target metadata[%s].runner_probe_command is required", label, row.Triple))
		}
		if row.ReleaseGate == "" {
			issues = append(issues, fmt.Sprintf("%s target metadata[%s].release_gate is required", label, row.Triple))
		}
		for _, required := range []string{"targets.json", row.Triple + "-runner.json", row.Triple + "-abi.json"} {
			if !memory100StringSliceHas(row.EvidenceArtifacts, required) {
				issues = append(issues, fmt.Sprintf("%s target metadata[%s].evidence_artifacts missing %s", label, row.Triple, required))
			}
		}
	default:
		if row.MemoryRun == "yes" {
			issues = append(issues, fmt.Sprintf("%s target metadata[%s].memory_run must not claim yes without target-host Memory100 evidence", label, row.Triple))
		}
		if row.MemoryClaimLevel == "production/host_runtime" {
			issues = append(issues, fmt.Sprintf("%s target metadata[%s].memory_claim_level must not claim production/host_runtime without target-host Memory100 evidence", label, row.Triple))
		}
	}
	return issues
}
