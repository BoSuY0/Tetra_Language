package memory100validate

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

func memory100StringSliceHas(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func validateMemory100MemoryReleaseCommands(
	commands []memory100Command,
	memoryDir string,
	gitHead string,
) []string {
	seen := map[string]string{}
	var issues []string
	for _, command := range commands {
		name := strings.TrimSpace(command.Name)
		text := strings.TrimSpace(command.Command)
		if name == "" {
			issues = append(issues, "memory release manifest command name is required")
			continue
		}
		if _, ok := seen[name]; ok {
			issues = append(
				issues,
				fmt.Sprintf("duplicate memory release manifest command %s", name),
			)
		}
		seen[name] = text
		if text == "" {
			issues = append(
				issues,
				fmt.Sprintf("memory release manifest command %s command is required", name),
			)
		}
		if strings.Contains(text, "|| true") || strings.Contains(text, "continue-on-error") ||
			strings.Contains(text, "set +e") {
			issues = append(
				issues,
				fmt.Sprintf("memory release manifest command %s contains bypass marker", name),
			)
		}
	}
	for name, fragment := range requiredMemory100MemoryReleaseCommands {
		text, ok := seen[name]
		if !ok {
			issues = append(
				issues,
				fmt.Sprintf(
					"missing memory release manifest command %s containing %q",
					name,
					fragment,
				),
			)
			continue
		}
		if !strings.Contains(text, fragment) {
			issues = append(
				issues,
				fmt.Sprintf("memory release manifest command %s must contain %q", name, fragment),
			)
		}
	}
	issues = append(
		issues,
		validateMemory100MemoryReleaseCommandProvenance(seen, memoryDir, gitHead)...)
	return issues
}

func validateMemory100MemoryReleaseCommandProvenance(
	commands map[string]string,
	memoryDir string,
	gitHead string,
) []string {
	type pathRequirement struct {
		name string
		flag string
		rel  string
	}
	pathRequirements := []pathRequirement{
		{
			name: "memory-production-smoke",
			flag: "--report",
			rel:  "memory-production-linux-x64.json",
		},
		{name: "target-report", flag: ">", rel: "targets.json"},
		{name: "validate-targets", flag: "--report", rel: "targets.json"},
		{name: "memory-fuzz-short", flag: "--report-dir", rel: "memory-fuzz-tier1"},
		{
			name: "validate-memory-fuzz-oracle",
			flag: "--report",
			rel:  "memory-fuzz-tier1/memory-fuzz-oracle.json",
		},
		{name: "validate-memory-fuzz-oracle", flag: "--artifact-dir", rel: "memory-fuzz-tier1"},
		{name: "ram-contract-gate", flag: "--report-dir", rel: "ram-contract"},
		{name: "island-proof-verifier", flag: "--proof", rel: "island-proof-verifier.json"},
		{
			name: "island-proof-verifier",
			flag: "--memory-report",
			rel:  "island-proof-memory-report.json",
		},
		{name: "artifact-hashes-write", flag: "--root", rel: ""},
		{name: "artifact-hashes-write", flag: "--out", rel: "artifact-hashes.json"},
		{name: "artifact-hashes-validate", flag: "--manifest", rel: "artifact-hashes.json"},
	}
	var issues []string
	for _, requirement := range pathRequirements {
		text := strings.TrimSpace(commands[requirement.name])
		if text == "" {
			continue
		}
		wantPath := memoryDir
		if requirement.rel != "" {
			wantPath = filepath.Join(memoryDir, filepath.FromSlash(requirement.rel))
		}
		if !memory100CommandContainsAnyPath(
			text,
			requirement.flag,
			memory100EquivalentPathForms(wantPath),
		) {
			issues = append(
				issues,
				fmt.Sprintf(
					("memory release manifest command %s must use %s under the "+
						"current memory production report dir for %s"),
					requirement.name,
					requirement.flag,
					requirement.rel,
				),
			)
		}
	}
	for _, requirement := range []struct {
		name string
		flag string
	}{
		{name: "memory-production-smoke", flag: "--git-head"},
		{name: "memory-fuzz-short", flag: "--git-head"},
		{name: "validate-memory-fuzz-oracle", flag: "--current-git-head"},
		{name: "island-proof-verifier", flag: "--current-git-head"},
	} {
		text := strings.TrimSpace(commands[requirement.name])
		if text == "" || strings.TrimSpace(gitHead) == "" {
			continue
		}
		if !strings.Contains(text, requirement.flag+" "+gitHead) {
			issues = append(
				issues,
				fmt.Sprintf(
					"memory release manifest command %s must use %s %s",
					requirement.name,
					requirement.flag,
					gitHead,
				),
			)
		}
	}
	if text := strings.TrimSpace(commands["island-proof-verifier"]); text != "" &&
		!strings.Contains(text, "--require-same-commit") {
		issues = append(
			issues,
			"memory release manifest command island-proof-verifier must require same-commit validation",
		)
	}
	return issues
}

func validateMemory100MemoryReleaseArtifactRefs(
	artifacts []memory100MemoryReleaseArtifactRef,
	memoryDir string,
	gitHead string,
) []string {
	byKind := map[string]memory100MemoryReleaseArtifactRef{}
	seenPath := map[string]bool{}
	requiredKinds := map[string]bool{}
	for _, required := range requiredMemory100MemoryReleaseArtifacts {
		requiredKinds[required.Kind] = true
	}
	var issues []string
	for _, artifact := range artifacts {
		if err := validateMemory100SafeRel(artifact.Path); err != nil {
			issues = append(
				issues,
				fmt.Sprintf(
					"memory release manifest artifact path %q is invalid: %v",
					artifact.Path,
					err,
				),
			)
			continue
		}
		if strings.TrimSpace(artifact.Kind) == "" {
			issues = append(
				issues,
				fmt.Sprintf("memory release manifest artifact %s kind is required", artifact.Path),
			)
			continue
		}
		if !requiredKinds[artifact.Kind] {
			issues = append(
				issues,
				fmt.Sprintf(
					"unexpected memory release manifest artifact kind %s at %s",
					artifact.Kind,
					artifact.Path,
				),
			)
		}
		if _, ok := byKind[artifact.Kind]; ok {
			issues = append(
				issues,
				fmt.Sprintf("duplicate memory release manifest artifact kind %s", artifact.Kind),
			)
		}
		byKind[artifact.Kind] = artifact
		if seenPath[artifact.Path] {
			issues = append(
				issues,
				fmt.Sprintf("duplicate memory release manifest artifact path %s", artifact.Path),
			)
		}
		seenPath[artifact.Path] = true
		if artifact.Target != "linux-x64" {
			issues = append(
				issues,
				fmt.Sprintf(
					"memory release manifest artifact %s target is %q, want linux-x64",
					artifact.Kind,
					artifact.Target,
				),
			)
		}
		if strings.TrimSpace(artifact.Command) == "" {
			issues = append(
				issues,
				fmt.Sprintf(
					"memory release manifest artifact %s command is required",
					artifact.Kind,
				),
			)
		}
		issues = append(
			issues,
			validateMemory100MemoryReleaseArtifactCommand(artifact, memoryDir, gitHead)...)
	}
	for _, required := range requiredMemory100MemoryReleaseArtifacts {
		artifact, ok := byKind[required.Kind]
		if !ok {
			issues = append(
				issues,
				fmt.Sprintf("missing memory release manifest artifact %s", required.Kind),
			)
			continue
		}
		if artifact.Path != required.Path {
			issues = append(
				issues,
				fmt.Sprintf(
					"memory release manifest artifact %s path is %q, want %s",
					required.Kind,
					artifact.Path,
					required.Path,
				),
			)
		}
		if required.Schema != "" && artifact.Schema != required.Schema {
			issues = append(
				issues,
				fmt.Sprintf(
					"memory release manifest artifact %s schema is %q, want %s",
					required.Kind,
					artifact.Schema,
					required.Schema,
				),
			)
		}
		if required.CommandFragment != "" &&
			!strings.Contains(artifact.Command, required.CommandFragment) {
			issues = append(
				issues,
				fmt.Sprintf(
					"memory release manifest artifact %s command must contain %q",
					required.Kind,
					required.CommandFragment,
				),
			)
		}
	}
	return issues
}

func validateMemory100MemoryReleaseArtifactCommand(
	artifact memory100MemoryReleaseArtifactRef,
	memoryDir string,
	gitHead string,
) []string {
	type pathRequirement struct {
		flag string
		rel  string
	}
	requirementsByKind := map[string][]pathRequirement{
		"memory_production_report": {
			{flag: "--report", rel: "memory-production-linux-x64.json"},
		},
		"target_report":                    {{flag: ">", rel: "targets.json"}},
		"memory_fuzz_oracle_report":        {{flag: "--report-dir", rel: "memory-fuzz-tier1"}},
		"memory_fuzz_summary":              {{flag: "--report-dir", rel: "memory-fuzz-tier1"}},
		"memory_fuzz_island_proof_summary": {{flag: "--report-dir", rel: "memory-fuzz-tier1"}},
		"ram_contract_release_manifest":    {{flag: "--report-dir", rel: "ram-contract"}},
		"ram_contract_report":              {{flag: "--report-dir", rel: "ram-contract"}},
		"ram_memory_grade_report":          {{flag: "--report-dir", rel: "ram-contract"}},
		"ram_proof_store_summary":          {{flag: "--report-dir", rel: "ram-contract"}},
		"ram_validation_pipeline_coverage": {{flag: "--report-dir", rel: "ram-contract"}},
		"ram_heap_blockers":                {{flag: "--report-dir", rel: "ram-contract"}},
		"ram_copy_blockers":                {{flag: "--report-dir", rel: "ram-contract"}},
		"ram_contract_fuzz_oracle":         {{flag: "--report-dir", rel: "ram-contract"}},
		"ram_contract_hash_manifest":       {{flag: "--report-dir", rel: "ram-contract"}},
		"island_proof_verifier_report": {
			{flag: "--proof", rel: "island-proof-verifier.json"},
			{flag: "--memory-report", rel: "island-proof-memory-report.json"},
		},
		"island_proof_memory_report": {
			{flag: "--proof", rel: "island-proof-verifier.json"},
			{flag: "--memory-report", rel: "island-proof-memory-report.json"},
		},
		"artifact_hash_manifest": {
			{flag: "--root", rel: ""},
			{flag: "--out", rel: "artifact-hashes.json"},
		},
	}
	var issues []string
	requirements := requirementsByKind[artifact.Kind]
	for _, requirement := range requirements {
		wantPath := memoryDir
		if requirement.rel != "" {
			wantPath = filepath.Join(memoryDir, filepath.FromSlash(requirement.rel))
		}
		if !memory100CommandContainsAnyPath(
			artifact.Command,
			requirement.flag,
			memory100EquivalentPathForms(wantPath),
		) {
			issues = append(
				issues,
				fmt.Sprintf(
					("memory release manifest artifact %s command must use %s under "+
						"the current memory production report dir for %s"),
					artifact.Kind,
					requirement.flag,
					requirement.rel,
				),
			)
		}
	}
	if strings.TrimSpace(gitHead) != "" {
		switch artifact.Kind {
		case "memory_production_report",
			"memory_fuzz_oracle_report",
			"memory_fuzz_summary",
			"memory_fuzz_island_proof_summary",
			"island_proof_verifier_report",
			"island_proof_memory_report":
			if !strings.Contains(artifact.Command, gitHead) {
				issues = append(
					issues,
					fmt.Sprintf(
						"memory release manifest artifact %s command must include git head %s",
						artifact.Kind,
						gitHead,
					),
				)
			}
		}
	}
	if (artifact.Kind == "island_proof_verifier_report" || artifact.Kind == "island_proof_memory_report") &&
		!strings.Contains(artifact.Command, "--require-same-commit") {
		issues = append(
			issues,
			fmt.Sprintf(
				"memory release manifest artifact %s command must require same-commit validation",
				artifact.Kind,
			),
		)
	}
	return issues
}

func validateMemory100RAMContractReleaseManifest(path string, gitHead string) []string {
	var manifest memory100RAMContractReleaseManifest
	if err := readMemory100StrictJSON(path, &manifest); err != nil {
		return []string{fmt.Sprintf("RAM contract release manifest invalid: %v", err)}
	}
	var issues []string
	if manifest.Schema != "tetra.ram-contract.release-manifest.v1" {
		issues = append(
			issues,
			fmt.Sprintf(
				"RAM contract release manifest schema is %q, want tetra.ram-contract.release-manifest.v1",
				manifest.Schema,
			),
		)
	}
	if manifest.Status != "pass" {
		issues = append(
			issues,
			fmt.Sprintf("RAM contract release manifest status is %q, want pass", manifest.Status),
		)
	}
	if manifest.Target != "linux-x64" {
		issues = append(
			issues,
			fmt.Sprintf(
				"RAM contract release manifest target is %q, want linux-x64",
				manifest.Target,
			),
		)
	}
	if gitHead != "" && manifest.GitHead != gitHead {
		issues = append(
			issues,
			fmt.Sprintf(
				"RAM contract release manifest git_head %s does not match Memory100 git_head %s",
				manifest.GitHead,
				gitHead,
			),
		)
	}
	if _, err := time.Parse(time.RFC3339, manifest.GeneratedAt); err != nil {
		issues = append(
			issues,
			fmt.Sprintf("RAM contract release manifest generated_at must be RFC3339: %v", err),
		)
	}
	if manifest.ReportDir != "." {
		issues = append(
			issues,
			fmt.Sprintf(
				"RAM contract release manifest report_dir is %q, want .",
				manifest.ReportDir,
			),
		)
	}
	if manifest.HashManifest != "artifact-hashes.json" {
		issues = append(
			issues,
			fmt.Sprintf(
				"RAM contract release manifest hash_manifest is %q, want artifact-hashes.json",
				manifest.HashManifest,
			),
		)
	}
	if len(nonEmptyMemory100Strings(manifest.NonClaims)) == 0 {
		issues = append(issues, "RAM contract release manifest non_claims must not be empty")
	}
	issues = append(
		issues,
		validateMemory100Claims(
			"RAM contract release manifest non_claims",
			manifest.NonClaims,
			true,
		)...)
	issues = append(
		issues,
		validateMemory100RAMContractReleaseCommands(
			manifest.Commands,
			filepath.Dir(path),
			gitHead,
		)...)
	issues = append(issues, validateMemory100RAMContractReleaseArtifactRefs(manifest.Artifacts)...)
	return issues
}

func validateMemory100RAMContractReleaseCommands(
	commands []memory100Command,
	ramDir string,
	gitHead string,
) []string {
	seen := map[string]string{}
	var issues []string
	for _, command := range commands {
		name := strings.TrimSpace(command.Name)
		text := strings.TrimSpace(command.Command)
		if name == "" {
			issues = append(issues, "RAM contract release manifest command name is required")
			continue
		}
		if _, ok := seen[name]; ok {
			issues = append(
				issues,
				fmt.Sprintf("duplicate RAM contract release manifest command %s", name),
			)
		}
		seen[name] = text
		if text == "" {
			issues = append(
				issues,
				fmt.Sprintf("RAM contract release manifest command %s command is required", name),
			)
			continue
		}
		if strings.Contains(text, "|| true") || strings.Contains(text, "continue-on-error") ||
			strings.Contains(text, "set +e") {
			issues = append(
				issues,
				fmt.Sprintf(
					"RAM contract release manifest command %s contains bypass marker",
					name,
				),
			)
		}
	}
	for name, fragment := range requiredMemory100RAMContractReleaseCommands {
		text, ok := seen[name]
		if !ok {
			issues = append(
				issues,
				fmt.Sprintf(
					"missing RAM contract release manifest command %s containing %q",
					name,
					fragment,
				),
			)
			continue
		}
		if !strings.Contains(text, fragment) {
			issues = append(
				issues,
				fmt.Sprintf(
					"RAM contract release manifest command %s must contain %q",
					name,
					fragment,
				),
			)
		}
	}
	type pathRequirement struct {
		name string
		flag string
		rel  string
	}
	for _, requirement := range []pathRequirement{
		{name: "validate-ram-contract-report", flag: "--report", rel: "ram-contract-report.json"},
		{name: "validate-memory-grade-report", flag: "--report", rel: "memory-grade-report.json"},
		{name: "validate-proof-store-summary", flag: "--report", rel: "proof-store-summary.json"},
		{
			name: "validate-validation-pipeline-coverage",
			flag: "--report",
			rel:  "validation-pipeline-coverage.json",
		},
		{name: "validate-heap-blockers", flag: "--report", rel: "heap-blockers.json"},
		{name: "validate-copy-blockers", flag: "--report", rel: "copy-blockers.json"},
		{name: "ram-contract-fuzz-short", flag: "--report-dir", rel: "fuzz"},
		{
			name: "validate-ram-contract-fuzz-oracle",
			flag: "--report",
			rel:  "fuzz/ram-contract-fuzz-oracle.json",
		},
		{name: "artifact-hashes-write", flag: "--root", rel: ""},
		{name: "artifact-hashes-write", flag: "--out", rel: "artifact-hashes.json"},
		{name: "artifact-hashes-validate", flag: "--manifest", rel: "artifact-hashes.json"},
		{name: "ram-contract-release-validator", flag: "--report-dir", rel: ""},
	} {
		text := strings.TrimSpace(seen[requirement.name])
		if text == "" {
			continue
		}
		wantPath := ramDir
		if requirement.rel != "" {
			wantPath = filepath.Join(ramDir, filepath.FromSlash(requirement.rel))
		}
		if !memory100CommandContainsAnyPath(
			text,
			requirement.flag,
			memory100EquivalentPathForms(wantPath),
		) {
			issues = append(
				issues,
				fmt.Sprintf(
					("RAM contract release manifest command %s must use %s under the "+
						"current RAM contract report dir for %s"),
					requirement.name,
					requirement.flag,
					requirement.rel,
				),
			)
		}
	}
	for _, requirement := range []struct {
		name string
		flag string
	}{
		{name: "ram-contract-fuzz-short", flag: "--git-head"},
		{name: "validate-ram-contract-fuzz-oracle", flag: "--current-git-head"},
		{name: "ram-contract-release-validator", flag: "--current-git-head"},
	} {
		text := strings.TrimSpace(seen[requirement.name])
		if text == "" || strings.TrimSpace(gitHead) == "" {
			continue
		}
		if !strings.Contains(text, requirement.flag+" "+gitHead) {
			issues = append(
				issues,
				fmt.Sprintf(
					"RAM contract release manifest command %s must use %s %s",
					requirement.name,
					requirement.flag,
					gitHead,
				),
			)
		}
	}
	return issues
}

func validateMemory100RAMContractReleaseArtifactRefs(
	artifacts []memory100RAMContractReleaseArtifactRef,
) []string {
	byKind := map[string]memory100RAMContractReleaseArtifactRef{}
	seenPath := map[string]bool{}
	requiredKinds := map[string]bool{}
	for _, required := range requiredMemory100RAMContractReleaseArtifacts {
		requiredKinds[required.Kind] = true
	}
	var issues []string
	for _, artifact := range artifacts {
		if err := validateMemory100SafeRel(artifact.Path); err != nil {
			issues = append(
				issues,
				fmt.Sprintf(
					"RAM contract release manifest artifact path %q is invalid: %v",
					artifact.Path,
					err,
				),
			)
			continue
		}
		if strings.TrimSpace(artifact.Kind) == "" {
			issues = append(
				issues,
				fmt.Sprintf(
					"RAM contract release manifest artifact %s kind is required",
					artifact.Path,
				),
			)
			continue
		}
		if !requiredKinds[artifact.Kind] {
			issues = append(
				issues,
				fmt.Sprintf(
					"unexpected RAM contract release manifest artifact kind %s at %s",
					artifact.Kind,
					artifact.Path,
				),
			)
		}
		if _, ok := byKind[artifact.Kind]; ok {
			issues = append(
				issues,
				fmt.Sprintf(
					"duplicate RAM contract release manifest artifact kind %s",
					artifact.Kind,
				),
			)
		}
		byKind[artifact.Kind] = artifact
		if seenPath[artifact.Path] {
			issues = append(
				issues,
				fmt.Sprintf(
					"duplicate RAM contract release manifest artifact path %s",
					artifact.Path,
				),
			)
		}
		seenPath[artifact.Path] = true
	}
	for _, required := range requiredMemory100RAMContractReleaseArtifacts {
		artifact, ok := byKind[required.Kind]
		if !ok {
			issues = append(
				issues,
				fmt.Sprintf("missing RAM contract release manifest artifact %s", required.Kind),
			)
			continue
		}
		if artifact.Path != required.Path {
			issues = append(
				issues,
				fmt.Sprintf(
					"RAM contract release manifest artifact %s path is %q, want %s",
					required.Kind,
					artifact.Path,
					required.Path,
				),
			)
		}
		if artifact.Schema != required.Schema {
			issues = append(
				issues,
				fmt.Sprintf(
					"RAM contract release manifest artifact %s schema is %q, want %s",
					required.Kind,
					artifact.Schema,
					required.Schema,
				),
			)
		}
	}
	return issues
}
