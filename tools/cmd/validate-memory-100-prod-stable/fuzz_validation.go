package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func validateMemory100MemoryFuzzBundle(dir string, gitHead string) []string {
	var issues []string
	for _, rel := range []string{"memory-fuzz-oracle.json", "summary.md", "summary.json", "island-proof-fuzz-summary.json", "artifact-hashes.json"} {
		issues = append(issues, requireMemory100MemoryFuzzFile(dir, rel)...)
	}
	for _, rel := range []string{"reproducers/compiler-crash", "reproducers/miscompile", "reducers/miscompile"} {
		issues = append(issues, requireMemory100MemoryFuzzDir(dir, rel)...)
	}
	if len(issues) > 0 {
		return issues
	}

	summaryMD, err := os.ReadFile(filepath.Join(dir, "summary.md"))
	if err != nil {
		issues = append(issues, fmt.Sprintf("memory fuzz summary.md missing or unreadable: %v", err))
	} else {
		text := string(summaryMD)
		for _, want := range []string{"Memory Fuzz Short Summary", "Tier 1", "memory-fuzz-oracle.json"} {
			if !strings.Contains(text, want) {
				issues = append(issues, fmt.Sprintf("memory fuzz summary.md missing %q", want))
			}
		}
	}

	var summary memory100MemoryFuzzSummary
	if err := readMemory100StrictJSON(filepath.Join(dir, "summary.json"), &summary); err != nil {
		issues = append(issues, fmt.Sprintf("memory fuzz summary.json invalid: %v", err))
	} else {
		issues = append(issues, validateMemory100MemoryFuzzSummary(summary, gitHead, dir)...)
	}

	var proofSummary memory100IslandProofFuzzSummary
	if err := readMemory100StrictJSON(filepath.Join(dir, "island-proof-fuzz-summary.json"), &proofSummary); err != nil {
		issues = append(issues, fmt.Sprintf("memory fuzz island-proof-fuzz-summary.json invalid: %v", err))
	} else {
		issues = append(issues, validateMemory100IslandProofFuzzSummary(proofSummary)...)
	}

	issues = append(issues, validateMemory100MemoryFuzzHashManifest(filepath.Join(dir, "artifact-hashes.json"), dir)...)
	return issues
}

func requireMemory100MemoryFuzzFile(dir string, rel string) []string {
	path := filepath.Join(dir, filepath.FromSlash(rel))
	info, err := os.Lstat(path)
	if err != nil {
		return []string{fmt.Sprintf("memory fuzz artifact %s is missing: %v", rel, err)}
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return []string{fmt.Sprintf("memory fuzz artifact %s must not be a symlink", rel)}
	}
	if !info.Mode().IsRegular() {
		return []string{fmt.Sprintf("memory fuzz artifact %s is not a regular file", rel)}
	}
	if info.Size() == 0 {
		return []string{fmt.Sprintf("memory fuzz artifact %s is empty", rel)}
	}
	return nil
}

func requireMemory100MemoryFuzzDir(dir string, rel string) []string {
	if err := validateMemory100SafeRel(rel); err != nil {
		return []string{fmt.Sprintf("memory fuzz artifact dir %s path invalid: %v", rel, err)}
	}
	path := filepath.Join(dir, filepath.FromSlash(rel))
	info, err := os.Lstat(path)
	if err != nil {
		return []string{fmt.Sprintf("memory fuzz artifact dir %s is missing: %v", rel, err)}
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return []string{fmt.Sprintf("memory fuzz artifact dir %s must not be a symlink", rel)}
	}
	if !info.IsDir() {
		return []string{fmt.Sprintf("memory fuzz artifact dir %s is not a directory", rel)}
	}
	return nil
}

func validateMemory100MemoryFuzzSummary(summary memory100MemoryFuzzSummary, gitHead string, artifactDir string) []string {
	var issues []string
	artifactDirs := memory100EquivalentPathForms(artifactDir)
	artifactDirLabel := artifactDirs[0]
	var oraclePaths []string
	for _, dir := range artifactDirs {
		oraclePaths = append(oraclePaths, dir+"/memory-fuzz-oracle.json")
	}
	if summary.SchemaVersion != "tetra.memory-fuzz-short.summary.v1" {
		issues = append(issues, fmt.Sprintf("memory fuzz summary.json schema_version is %q, want tetra.memory-fuzz-short.summary.v1", summary.SchemaVersion))
	}
	if summary.Kind != "tier1_short_ci_smoke" || summary.Tier != "tier1_short_ci_smoke" || summary.Status != "pass" {
		issues = append(issues, fmt.Sprintf("memory fuzz summary.json identity/status must record passing Tier 1 short CI smoke, got kind=%q tier=%q status=%q", summary.Kind, summary.Tier, summary.Status))
	}
	issues = append(issues, validateMemory100MemoryFuzzFailureClassification(summary)...)
	issues = append(issues, validateMemory100MemoryFuzzReproducibilitySeeds(summary.ReproducibilitySeeds)...)
	for key, want := range map[string]string{
		"artifact_hashes":           "artifact-hashes.json",
		"island_proof_fuzz_summary": "island-proof-fuzz-summary.json",
		"oracle_report":             "memory-fuzz-oracle.json",
		"summary_md":                "summary.md",
		"summary_json":              "summary.json",
	} {
		got := summary.Artifacts[key]
		if got != want {
			issues = append(issues, fmt.Sprintf("memory fuzz summary.json artifact %s is %q, want %q", key, got, want))
		}
		if strings.TrimSpace(got) != "" {
			if err := validateMemory100SafeRel(got); err != nil {
				issues = append(issues, fmt.Sprintf("memory fuzz summary.json artifact %s path invalid: %v", key, err))
			}
		}
	}
	var sawRunner, sawValidator bool
	for _, command := range summary.Commands {
		if command.Status != "pass" {
			issues = append(issues, fmt.Sprintf("memory fuzz summary.json command %s status is %q, want pass", command.Name, command.Status))
		}
		switch command.Name {
		case "memory-fuzz-short":
			if strings.Contains(command.Command, "go run ./tools/cmd/memory-fuzz-short") && memory100CommandContainsAnyPath(command.Command, "--report-dir", artifactDirs) && strings.Contains(command.Command, "--git-head "+gitHead) {
				sawRunner = true
			}
		case "validate-memory-fuzz-oracle":
			if strings.Contains(command.Command, "go run ./tools/cmd/validate-memory-fuzz-oracle") && memory100CommandContainsAnyPath(command.Command, "--report", oraclePaths) && memory100CommandContainsAnyPath(command.Command, "--artifact-dir", artifactDirs) && strings.Contains(command.Command, "--current-git-head "+gitHead) {
				sawValidator = true
			}
		}
	}
	if !sawRunner {
		issues = append(issues, fmt.Sprintf("memory fuzz summary.json missing memory-fuzz-short same-commit command provenance for current artifact dir %s", artifactDirLabel))
	}
	if !sawValidator {
		issues = append(issues, fmt.Sprintf("memory fuzz summary.json missing validate-memory-fuzz-oracle same-commit command provenance for current artifact dir %s", artifactDirLabel))
	}
	return issues
}

func validateMemory100MemoryFuzzReproducibilitySeeds(seeds []string) []string {
	if len(seeds) == 0 {
		return []string{"memory fuzz summary.json reproducibility_seeds are required"}
	}
	var issues []string
	if len(seeds) < 12 {
		issues = append(issues, fmt.Sprintf("memory fuzz summary.json reproducibility_seeds has %d entries, want at least 12 for v0-v11", len(seeds)))
	}
	seen := map[string]bool{}
	for _, seed := range seeds {
		text := strings.TrimSpace(seed)
		if text == "" {
			issues = append(issues, "memory fuzz summary.json reproducibility_seeds contains empty seed")
			continue
		}
		lower := strings.ToLower(text)
		for _, forbidden := range []string{"todo", "placeholder", "fake", "mock"} {
			if strings.Contains(lower, forbidden) {
				issues = append(issues, fmt.Sprintf("memory fuzz summary.json reproducibility_seeds contains forbidden marker %q", forbidden))
			}
		}
		if seen[text] {
			issues = append(issues, fmt.Sprintf("memory fuzz summary.json reproducibility_seeds duplicate seed %q", text))
		}
		seen[text] = true
	}
	joined := "\n" + strings.Join(seeds, "\n") + "\n"
	for i := 0; i < 12; i++ {
		if !strings.Contains(joined, fmt.Sprintf(":v%d:", i)) {
			issues = append(issues, fmt.Sprintf("memory fuzz summary.json reproducibility_seeds missing v%d seed", i))
		}
	}
	return issues
}

func validateMemory100MemoryFuzzFailureClassification(summary memory100MemoryFuzzSummary) []string {
	var issues []string
	counts := []struct {
		name  string
		value *int
	}{
		{name: "observed_failures", value: summary.ObservedFailures},
		{name: "classified_failures", value: summary.ClassifiedFailures},
		{name: "unclassified_failures", value: summary.UnclassifiedFailures},
		{name: "release_blocking_failures", value: summary.ReleaseBlockingFailures},
	}
	values := map[string]int{}
	for _, count := range counts {
		if count.value == nil {
			issues = append(issues, fmt.Sprintf("memory fuzz summary.json %s is required", count.name))
			continue
		}
		if *count.value < 0 {
			issues = append(issues, fmt.Sprintf("memory fuzz summary.json %s is %d, want non-negative", count.name, *count.value))
		}
		values[count.name] = *count.value
	}
	if len(issues) > 0 {
		return issues
	}
	if values["classified_failures"]+values["unclassified_failures"] != values["observed_failures"] {
		issues = append(issues, "memory fuzz summary.json classified_failures + unclassified_failures must equal observed_failures")
	}
	if values["release_blocking_failures"] > values["observed_failures"] {
		issues = append(issues, fmt.Sprintf("memory fuzz summary.json release_blocking_failures is %d, exceeds observed_failures %d", values["release_blocking_failures"], values["observed_failures"]))
	}
	if values["unclassified_failures"] != 0 {
		issues = append(issues, fmt.Sprintf("memory fuzz summary.json unclassified_failures is %d, want 0", values["unclassified_failures"]))
	}
	if summary.Status == "pass" && (values["observed_failures"] != 0 || values["classified_failures"] != 0 || values["release_blocking_failures"] != 0) {
		issues = append(issues, fmt.Sprintf("memory fuzz summary.json passing Tier 1 summary must record zero observed/classified/release_blocking failures, got observed=%d classified=%d release_blocking=%d", values["observed_failures"], values["classified_failures"], values["release_blocking_failures"]))
	}
	return issues
}

func memory100EquivalentPathForms(path string) []string {
	seen := map[string]bool{}
	add := func(value string, out *[]string) {
		value = filepath.ToSlash(filepath.Clean(value))
		if value != "" && !seen[value] {
			seen[value] = true
			*out = append(*out, value)
		}
	}
	var out []string
	add(path, &out)
	if abs, err := filepath.Abs(path); err == nil {
		add(abs, &out)
		if cwd, err := os.Getwd(); err == nil {
			if rel, err := filepath.Rel(cwd, abs); err == nil && rel != "." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".." {
				add(rel, &out)
			}
		}
	}
	return out
}

func memory100CommandContainsAnyPath(command string, flag string, paths []string) bool {
	for _, path := range paths {
		if strings.Contains(command, flag+" "+path) {
			return true
		}
	}
	return false
}

func validateMemory100IslandProofFuzzSummary(summary memory100IslandProofFuzzSummary) []string {
	var issues []string
	if summary.SchemaVersion != "tetra.island-proof-fuzz-summary.v1" {
		issues = append(issues, fmt.Sprintf("memory fuzz island-proof-fuzz-summary.json schema_version is %q, want tetra.island-proof-fuzz-summary.v1", summary.SchemaVersion))
	}
	if summary.Status != "pass" {
		issues = append(issues, fmt.Sprintf("memory fuzz island-proof-fuzz-summary.json status is %q, want pass", summary.Status))
	}
	if summary.Total < 10 {
		issues = append(issues, fmt.Sprintf("memory fuzz island-proof-fuzz-summary.json total is %d, want at least 10", summary.Total))
	}
	if summary.Accepted != 0 || summary.Rejected != summary.Total {
		issues = append(issues, fmt.Sprintf("memory fuzz island-proof-fuzz-summary.json counts total=%d rejected=%d accepted=%d, want all rejected", summary.Total, summary.Rejected, summary.Accepted))
	}
	seen := map[string]bool{}
	for _, c := range summary.Cases {
		if c.Status != "rejected" {
			issues = append(issues, fmt.Sprintf("memory fuzz island proof fuzz case %s status is %q, want rejected", c.Name, c.Status))
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
			issues = append(issues, fmt.Sprintf("memory fuzz island-proof-fuzz-summary.json missing mutation case %s", name))
		}
	}
	return issues
}

func validateMemory100MemoryFuzzHashManifest(hashPath string, dir string) []string {
	var manifest memory100HashManifest
	if err := readMemory100StrictJSON(hashPath, &manifest); err != nil {
		return []string{fmt.Sprintf("memory fuzz artifact-hashes.json missing or invalid: %v", err)}
	}
	var issues []string
	if manifest.Schema != memory100HashSchema {
		issues = append(issues, fmt.Sprintf("memory fuzz artifact-hashes.json schema is %q, want %s", manifest.Schema, memory100HashSchema))
	}
	if manifest.Root != "." {
		issues = append(issues, fmt.Sprintf("memory fuzz artifact-hashes.json root is %q, want .", manifest.Root))
	}
	if len(manifest.Artifacts) == 0 {
		issues = append(issues, "memory fuzz artifact-hashes.json artifacts must not be empty")
	}
	seen := map[string]memory100HashArtifact{}
	lastPath := ""
	for _, artifact := range manifest.Artifacts {
		if err := validateMemory100SafeRel(artifact.Path); err != nil {
			issues = append(issues, fmt.Sprintf("memory fuzz artifact-hashes.json path %q is invalid: %v", artifact.Path, err))
			continue
		}
		if artifact.Path == "artifact-hashes.json" {
			issues = append(issues, "memory fuzz artifact-hashes.json must not list itself")
		}
		if lastPath != "" && artifact.Path < lastPath {
			issues = append(issues, "memory fuzz artifact-hashes.json artifacts must be sorted by path")
		}
		lastPath = artifact.Path
		if _, ok := seen[artifact.Path]; ok {
			issues = append(issues, fmt.Sprintf("duplicate memory fuzz hash entry for %s", artifact.Path))
		}
		seen[artifact.Path] = artifact
		if err := validateMemory100SHA256(artifact.SHA256, artifact.Path); err != nil {
			issues = append(issues, err.Error())
		}
		actual, err := hashMemory100File(dir, artifact.Path)
		if err != nil {
			issues = append(issues, fmt.Sprintf("hash memory fuzz artifact %s: %v", artifact.Path, err))
			continue
		}
		if actual.Size != artifact.Size {
			issues = append(issues, fmt.Sprintf("memory fuzz size mismatch for %s: got %d want %d", artifact.Path, actual.Size, artifact.Size))
		}
		if actual.SHA256 != artifact.SHA256 {
			issues = append(issues, fmt.Sprintf("memory fuzz sha256 mismatch for %s: got %s want %s", artifact.Path, actual.SHA256, artifact.SHA256))
		}
		if actual.Schema != artifact.Schema {
			issues = append(issues, fmt.Sprintf("memory fuzz schema mismatch for %s: got %q want %q", artifact.Path, actual.Schema, artifact.Schema))
		}
	}
	for _, rel := range []string{"memory-fuzz-oracle.json", "summary.md", "summary.json", "island-proof-fuzz-summary.json"} {
		if _, ok := seen[rel]; !ok {
			issues = append(issues, fmt.Sprintf("missing memory fuzz hash manifest entry for %s", rel))
		}
	}
	actualPaths, err := listMemory100ArtifactPaths(dir)
	if err != nil {
		issues = append(issues, fmt.Sprintf("list memory fuzz artifacts: %v", err))
	} else {
		for _, rel := range actualPaths {
			if _, ok := seen[rel]; !ok {
				issues = append(issues, fmt.Sprintf("unlisted memory fuzz artifact %s", rel))
			}
		}
	}
	return issues
}

func validateMemory100ProofTransitionReport(path string, gitHead string) []string {
	var report memory100ProofTransitionReport
	if err := readMemory100JSON(path, &report); err != nil {
		return []string{fmt.Sprintf("proof transition report invalid: %v", err)}
	}
	var issues []string
	if report.Status != "pass" {
		issues = append(issues, fmt.Sprintf("proof transition report status is %q, want pass", report.Status))
	}
	if gitHead != "" && report.GitHead != gitHead {
		issues = append(issues, fmt.Sprintf("proof transition report git_head %s does not match Memory100 git_head %s", report.GitHead, gitHead))
	}
	if len(nonEmptyMemory100Strings(report.NonClaims)) == 0 {
		issues = append(issues, "proof transition report non_claims must not be empty")
	}

	required := map[string]string{
		"stable_hash_semantic_fields":                "invalidated",
		"bounds_proof_preserved_through_translation": "preserved",
		"translation_missing_proof_requires_recheck": "requires_recheck",
		"optimization_invalidates_bounds_proofs":     "invalidated",
		"lowering_refines_bounds_proof_use":          "refined",
		"new_proof_requires_store_reference":         "new",
	}
	seenTransitions := map[string]bool{}
	seenRows := map[string]memory100ProofTransitionRow{}
	for i, row := range report.Rows {
		name := strings.TrimSpace(row.Name)
		if name == "" {
			issues = append(issues, fmt.Sprintf("proof transition row %d missing name", i))
			continue
		}
		if _, ok := seenRows[name]; ok {
			issues = append(issues, fmt.Sprintf("duplicate proof transition row %s", name))
		}
		seenRows[name] = row
		transition := strings.TrimSpace(row.Transition)
		if !memory100KnownProofTransition(transition) {
			issues = append(issues, fmt.Sprintf("proof transition row %s has unknown transition %q", name, row.Transition))
		} else {
			seenTransitions[transition] = true
		}
		if strings.TrimSpace(row.Evidence) == "" {
			issues = append(issues, fmt.Sprintf("proof transition row %s missing evidence", name))
		}
		if len(nonEmptyMemory100Strings(row.SourceArtifacts)) == 0 {
			issues = append(issues, fmt.Sprintf("proof transition row %s missing source_artifacts", name))
		}
		if len(nonEmptyMemory100Strings(row.Tests)) == 0 {
			issues = append(issues, fmt.Sprintf("proof transition row %s missing tests", name))
		}
		switch transition {
		case "preserved", "refined":
			if strings.TrimSpace(row.BeforeArtifact) == "" || strings.TrimSpace(row.AfterArtifact) == "" {
				issues = append(issues, fmt.Sprintf("proof transition row %s transition %s requires before_artifact and after_artifact", name, transition))
			}
		case "invalidated", "requires_recheck":
			action := strings.ToLower(strings.TrimSpace(row.ConsumerAction))
			if !strings.Contains(action, "recheck") && !strings.Contains(action, "block") {
				issues = append(issues, fmt.Sprintf("proof transition row %s transition %s requires consumer_action with recheck or block", name, transition))
			}
		case "new":
			if strings.TrimSpace(row.AfterArtifact) == "" {
				issues = append(issues, fmt.Sprintf("proof transition row %s transition new requires after_artifact", name))
			}
		}
	}
	for _, transition := range []string{"preserved", "refined", "invalidated", "new", "requires_recheck"} {
		if !seenTransitions[transition] {
			issues = append(issues, fmt.Sprintf("proof transition report missing transition %s", transition))
		}
	}
	for name, wantTransition := range required {
		row, ok := seenRows[name]
		if !ok {
			issues = append(issues, fmt.Sprintf("proof transition report missing row %s", name))
			continue
		}
		if row.Transition != wantTransition {
			issues = append(issues, fmt.Sprintf("proof transition row %s transition is %q, want %q", name, row.Transition, wantTransition))
		}
		issues = append(issues, validateMemory100ProofTransitionEvidence(row)...)
	}
	return issues
}

func memory100KnownProofTransition(transition string) bool {
	switch transition {
	case "preserved", "refined", "invalidated", "new", "requires_recheck":
		return true
	default:
		return false
	}
}

func validateMemory100ProofTransitionEvidence(row memory100ProofTransitionRow) []string {
	text := strings.ToLower(strings.Join(append(append([]string{row.Evidence, row.BeforeArtifact, row.AfterArtifact, row.ConsumerAction}, row.SourceArtifacts...), row.Tests...), "\n"))
	wants := map[string][]string{
		"stable_hash_semantic_fields":                {"stablehash", "semantic"},
		"bounds_proof_preserved_through_translation": {"bounds", "proof", "translation"},
		"translation_missing_proof_requires_recheck": {"missing proof", "recheck"},
		"optimization_invalidates_bounds_proofs":     {"invalidat", "bounds"},
		"lowering_refines_bounds_proof_use":          {"lower", "proof"},
		"new_proof_requires_store_reference":         {"proof", "store"},
	}
	var issues []string
	for _, want := range wants[row.Name] {
		if !strings.Contains(text, want) {
			issues = append(issues, fmt.Sprintf("proof transition row %s evidence missing %q", row.Name, want))
		}
	}
	return issues
}

func validateMemory100RuntimeMemoryContractTargetMatrix(path string, gitHead string, targetMatrix []string) []string {
	return validateMemory100RuntimeMemoryContract(path, gitHead, nonEmptyMemory100Strings(targetMatrix))
}

func validateMemory100RuntimeMemoryContract(path string, gitHead string, targetMatrix []string) []string {
	var report memory100RuntimeMemoryContract
	if err := readMemory100JSON(path, &report); err != nil {
		return []string{fmt.Sprintf("runtime memory contract invalid: %v", err)}
	}
	var issues []string
	if report.Status != "pass" {
		issues = append(issues, fmt.Sprintf("runtime memory contract status is %q, want pass", report.Status))
	}
	if gitHead != "" && report.GitHead != gitHead {
		issues = append(issues, fmt.Sprintf("runtime memory contract git_head %s does not match Memory100 git_head %s", report.GitHead, gitHead))
	}
	for _, want := range []string{
		"no all-target memory parity claim",
		"OOM recovery guarantee is not claimed",
		"full stack-overflow protection is not claimed",
		"full allocator-corruption detection proof is not claimed",
		"production actor runtime is not claimed",
	} {
		if !memory100ContainsFold(report.NonClaims, want) {
			issues = append(issues, fmt.Sprintf("runtime memory contract missing non_claim %q", want))
		}
	}

	required := map[string]string{
		"linux-x64":   "production_host_runtime",
		"windows-x64": "host_required_nonclaim",
		"macos-x64":   "host_required_nonclaim",
		"wasm32-wasi": "artifact_runtime_tiered_nonclaim",
		"wasm32-web":  "artifact_runtime_tiered_nonclaim",
		"linux-x86":   "build_lower_only_nonclaim",
		"linux-x32":   "build_lower_only_nonclaim",
	}
	rowsByTarget := map[string]memory100RuntimeMemoryRow{}
	for i, row := range report.Rows {
		target := strings.TrimSpace(row.Target)
		if target == "" {
			issues = append(issues, fmt.Sprintf("runtime memory row %d missing target", i))
			continue
		}
		if _, ok := rowsByTarget[target]; ok {
			issues = append(issues, fmt.Sprintf("duplicate runtime memory target row %s", target))
		}
		rowsByTarget[target] = row
		if strings.TrimSpace(row.RuntimeStatus) == "" || strings.TrimSpace(row.MemoryRun) == "" || strings.TrimSpace(row.MemoryClaimLevel) == "" {
			issues = append(issues, fmt.Sprintf("runtime memory row %s missing runtime_status, memory_run, or memory_claim_level", target))
		}
		if strings.TrimSpace(row.Evidence) == "" {
			issues = append(issues, fmt.Sprintf("runtime memory row %s missing evidence", target))
		}
		if len(nonEmptyMemory100Strings(row.SourceArtifacts)) == 0 {
			issues = append(issues, fmt.Sprintf("runtime memory row %s missing source_artifacts", target))
		}
		if len(nonEmptyMemory100Strings(row.Tests)) == 0 {
			issues = append(issues, fmt.Sprintf("runtime memory row %s missing tests", target))
		}
		if len(nonEmptyMemory100Strings(row.NonClaims)) == 0 {
			issues = append(issues, fmt.Sprintf("runtime memory row %s missing non_claims", target))
		}
		if row.IncludedInMemory100TargetMatrix {
			if row.MemoryClaimLevel != "production_host_runtime" {
				issues = append(issues, fmt.Sprintf("runtime memory row %s is included but claim level is %q, want production_host_runtime", target, row.MemoryClaimLevel))
			}
			if row.RuntimeStatus != "production" || row.MemoryRun != "yes" {
				issues = append(issues, fmt.Sprintf("runtime memory row %s is included but runtime_status=%q memory_run=%q, want production/yes", target, row.RuntimeStatus, row.MemoryRun))
			}
			if !memory100ContainsFold(append(append([]string{row.Evidence}, row.SourceArtifacts...), row.Tests...), "runtime hardening") {
				issues = append(issues, fmt.Sprintf("runtime memory row %s included production evidence must mention runtime hardening", target))
			}
			if !memory100ContainsFold(append(append([]string{row.Evidence}, row.SourceArtifacts...), row.Tests...), "runtimeabi") {
				issues = append(issues, fmt.Sprintf("runtime memory row %s included production evidence must mention runtimeabi", target))
			}
		} else {
			if strings.TrimSpace(row.ExcludedReason) == "" {
				issues = append(issues, fmt.Sprintf("runtime memory row %s is excluded but missing excluded_reason", target))
			}
			if row.MemoryClaimLevel == "production_host_runtime" {
				issues = append(issues, fmt.Sprintf("runtime memory row %s is excluded but claims production_host_runtime", target))
			}
		}
	}
	for target, wantClaim := range required {
		row, ok := rowsByTarget[target]
		if !ok {
			issues = append(issues, fmt.Sprintf("runtime memory contract missing target row %s", target))
			continue
		}
		if row.MemoryClaimLevel != wantClaim {
			issues = append(issues, fmt.Sprintf("runtime memory row %s claim level is %q, want %q", target, row.MemoryClaimLevel, wantClaim))
		}
		if target == "linux-x64" && !row.IncludedInMemory100TargetMatrix {
			issues = append(issues, "runtime memory row linux-x64 must be included in Memory100 target matrix")
		}
		if target != "linux-x64" && row.IncludedInMemory100TargetMatrix {
			issues = append(issues, fmt.Sprintf("runtime memory row %s must not be included in Memory100 target matrix without target-host evidence", target))
		}
	}
	if len(targetMatrix) > 0 {
		included := map[string]struct{}{}
		for _, row := range rowsByTarget {
			if row.IncludedInMemory100TargetMatrix {
				included[row.Target] = struct{}{}
			}
		}
		matrixSet := memory100StringSet(targetMatrix)
		for _, missing := range memory100MissingSetKeys(matrixSet, included) {
			issues = append(issues, fmt.Sprintf("Memory100 target_matrix target %s missing included runtime memory row", missing))
		}
		for _, extra := range memory100MissingSetKeys(included, matrixSet) {
			issues = append(issues, fmt.Sprintf("runtime memory included target %s missing from Memory100 target_matrix", extra))
		}
	}
	return issues
}
