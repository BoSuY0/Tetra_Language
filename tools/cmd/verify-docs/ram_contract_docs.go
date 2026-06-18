package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type ramContractCompilerDocPaths struct {
	Design    string
	Spec      string
	User      string
	Readiness string
	Handoff   string
}

type ramContractCompilerRequirement struct {
	Name     string
	Path     string
	Required []string
}

func defaultRAMContractCompilerDocPaths() ramContractCompilerDocPaths {
	return ramContractCompilerDocPaths{
		Design:    filepath.FromSlash("docs/design/ram_contract_compiler.md"),
		Spec:      filepath.FromSlash("docs/spec/ram_contract_report_schema.md"),
		User:      filepath.FromSlash("docs/user/ram_contracts.md"),
		Readiness: filepath.FromSlash("docs/audits/ram-contract-compiler-readiness.md"),
		Handoff:   filepath.FromSlash("docs/audits/ram-contract-compiler-handoff.md"),
	}
}

func ramContractCompilerRequirements(paths ramContractCompilerDocPaths) []ramContractCompilerRequirement {
	return []ramContractCompilerRequirement{
		{
			Name: "design",
			Path: paths.Design,
			Required: []string{
				"RAM Contract Compiler",
				"tetra.ram-contract-report.v1",
				"tetra.memory-grade-report.v1",
				"tetra.proof-store-summary.v1",
				"tetra.validation-pipeline-coverage.v1",
				"compiler-owned facts",
				"MemoryFactGraph",
				"AllocPlan",
				"ProofStore",
				"heap-blockers.json",
				"copy-blockers.json",
				"TETRA4100",
				"no zero heap for all programs claim",
			},
		},
		{
			Name: "schema",
			Path: paths.Spec,
			Required: []string{
				"RAM Contract Report Schema",
				"tetra.ram-contract-report.v1",
				"tetra.memory-grade-report.v1",
				"tetra.proof-store-summary.v1",
				"tetra.validation-pipeline-coverage.v1",
				"tetra.ram-blockers.v1",
				"ram-contract-fuzz-oracle.json",
				"validate-ram-contract-report",
				"validate-memory-grade-report",
				"validate-proof-store-summary",
				"validate-validation-pipeline-coverage",
				"validate-heap-blockers",
				"validate-copy-blockers",
				"validate-ram-contract-fuzz-oracle",
			},
		},
		{
			Name: "user guide",
			Path: paths.User,
			Required: []string{
				"Using RAM Contracts",
				"--emit-ram-contract-report",
				"--fail-if-heap",
				"--fail-if-copy",
				"--fail-if-unbounded",
				"--memory-budget",
				"--ram-contract",
				"TETRA4100",
				"validate-ram-contract-release",
				"no zero-copy for all programs claim",
			},
		},
		{
			Name: "readiness audit",
			Path: paths.Readiness,
			Required: []string{
				"RAM Contract Compiler Readiness Audit",
				"Git head:",
				"Working tree:",
				"dirty working tree",
				"Verdict: `SCOPED_READY`",
				"scripts/release/post_v0_4/ram-contract-linux-x64-smoke.sh",
				".github/workflows/ci.yml",
				".github/workflows/release-packages.yml",
				"go test -buildvcs=false",
				"go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json",
				"git diff --check",
				"reports/ram-contract-release",
				"no full formal proof claim",
			},
		},
		{
			Name: "handoff",
			Path: paths.Handoff,
			Required: []string{
				"RAM Contract Compiler Handoff",
				"Release gate:",
				"scripts/release/post_v0_4/ram-contract-linux-x64-smoke.sh",
				"CI job:",
				"ram-contract-release-readiness-linux",
				"Package workflow:",
				"ram-contract-linux-x64",
				"Required artifacts:",
				"ram-contract-report.json",
				"memory-grade-report.json",
				"proof-store-summary.json",
				"validation-pipeline-coverage.json",
				"heap-blockers.json",
				"copy-blockers.json",
				"ram-contract-fuzz-oracle.json",
				"no all-target RAM parity claim",
			},
		},
	}
}

func verifyRAMContractCompilerDocs(paths ramContractCompilerDocPaths, features []featureManifest) error {
	var errs []string
	for _, requirement := range ramContractCompilerRequirements(paths) {
		raw, err := os.ReadFile(requirement.Path)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", requirement.Path, err))
			continue
		}
		text := string(raw)
		for _, want := range requirement.Required {
			if !strings.Contains(text, want) {
				errs = append(errs, fmt.Sprintf("%s: missing %q for %s RAM contract docs", requirement.Path, want, requirement.Name))
			}
		}
		for _, claim := range forbiddenRAMContractClaims(text) {
			errs = append(errs, fmt.Sprintf("%s: forbidden RAM contract claim %q", requirement.Path, claim))
		}
		for _, flag := range unsupportedRAMContractValidatorFlags(text) {
			errs = append(errs, fmt.Sprintf("%s: unsupported RAM contract validator flag %q", requirement.Path, flag))
		}
		if requirement.Name == "readiness audit" {
			if head, ok := staleRAMContractReadinessGitHead(text); ok {
				errs = append(errs, fmt.Sprintf("%s: stale readiness git head %s", requirement.Path, head))
			}
		}
	}
	if err := verifyRAMContractManifestFeature(features); err != nil {
		errs = append(errs, err.Error())
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

func unsupportedRAMContractValidatorFlags(text string) []string {
	var flags []string
	for _, needle := range []string{
		"validate-ram-contract-release --report ",
		"validate-ram-contract-release --report=",
	} {
		if strings.Contains(text, needle) {
			flags = append(flags, "validate-ram-contract-release --report")
			break
		}
	}
	return flags
}

var ramContractReadinessGitHeadPattern = regexp.MustCompile(`(?m)^Git head:\s*([0-9a-f]{40})\s*$`)

func staleRAMContractReadinessGitHead(text string) (string, bool) {
	match := ramContractReadinessGitHeadPattern.FindStringSubmatch(text)
	if len(match) != 2 {
		return "", false
	}
	current, ok := currentGitHeadForDocs()
	if !ok {
		return "", false
	}
	if match[1] != current {
		parent, ok := currentGitParentForDocs()
		if !ok || match[1] != parent {
			return match[1], true
		}
	}
	return "", false
}

func currentGitHeadForDocs() (string, bool) {
	out, err := exec.Command("git", "rev-parse", "--verify", "HEAD").Output()
	if err != nil {
		return "", false
	}
	head := strings.TrimSpace(string(out))
	if len(head) != 40 {
		return "", false
	}
	return head, true
}

func currentGitParentForDocs() (string, bool) {
	out, err := exec.Command("git", "rev-parse", "--verify", "HEAD^").Output()
	if err != nil {
		return "", false
	}
	head := strings.TrimSpace(string(out))
	if len(head) != 40 {
		return "", false
	}
	return head, true
}

func verifyRAMContractManifestFeature(features []featureManifest) error {
	var feature *featureManifest
	for i := range features {
		if features[i].ID == "compiler.ram-contracts" {
			feature = &features[i]
			break
		}
	}
	if feature == nil {
		return fmt.Errorf("feature registry missing compiler.ram-contracts")
	}
	if feature.Status != "current" {
		return fmt.Errorf("feature registry compiler.ram-contracts status = %s, want current", feature.Status)
	}
	haystack := feature.Scope + " " + feature.Stability + " " + strings.Join(feature.Docs, " ")
	for _, required := range []string{
		"RAM Contract Compiler report evidence",
		"tetra.ram-contract-report.v1",
		"tetra.memory-grade-report.v1",
		"tetra.proof-store-summary.v1",
		"tetra.validation-pipeline-coverage.v1",
		"heap-blockers.json",
		"copy-blockers.json",
		"ram-contract-fuzz-oracle.json",
		"--emit-ram-contract-report",
		"--fail-if-heap",
		"--fail-if-copy",
		"--fail-if-unbounded",
		"--memory-budget",
		"--ram-contract",
		"TETRA4100",
		"validate-ram-contract-release",
		"ram-contract-linux-x64-smoke.sh",
		"no zero heap for all programs claim",
		"no zero-copy for all programs claim",
		"no full formal proof claim",
		"no all-target RAM parity claim",
		"docs/design/ram_contract_compiler.md",
		"docs/spec/ram_contract_report_schema.md",
		"docs/user/ram_contracts.md",
		"docs/audits/ram-contract-compiler-readiness.md",
		"docs/audits/ram-contract-compiler-handoff.md",
	} {
		if !strings.Contains(haystack, required) {
			return fmt.Errorf("feature registry compiler.ram-contracts missing RAM contract phrase %q", required)
		}
	}
	for _, claim := range forbiddenRAMContractClaims(feature.Scope + " " + feature.Stability) {
		return fmt.Errorf("feature registry compiler.ram-contracts forbidden RAM contract claim %q", claim)
	}
	return nil
}

func forbiddenRAMContractClaims(text string) []string {
	lower := strings.ToLower(text)
	claims := forbiddenPublicPerformanceClaims(text)
	for _, phrase := range []string{
		"zero heap for all programs",
		"zero-heap for all programs",
		"zero copy for all programs",
		"zero-copy for all programs",
		"heap-free for all programs",
		"copy-free for all programs",
		"all-target ram parity",
		"all target ram parity",
		"ram parity across all targets",
		"full formal proof",
		"proof complete",
		"proof-complete",
		"prod_ready_proven",
	} {
		searchFrom := 0
		for {
			index := strings.Index(lower[searchFrom:], phrase)
			if index < 0 {
				break
			}
			absolute := searchFrom + index
			clause := clauseAround(lower, absolute, len(phrase), 260)
			sentence := sentenceAround(lower, absolute, len(phrase), 320)
			if ramContractPhraseAllowedAsExactNonClaim(phrase, clause) {
				searchFrom = absolute + len(phrase)
				continue
			}
			if explicitRAMContractPromotionContext(clause) && !explicitNonClaimContext(clause) {
				claims = append(claims, phrase)
				searchFrom = absolute + len(phrase)
				continue
			}
			if !explicitNonClaimContext(clause) && !explicitNonClaimContext(sentence) {
				claims = append(claims, phrase)
			}
			searchFrom = absolute + len(phrase)
		}
	}
	sort.Strings(claims)
	return compactStrings(claims)
}

func explicitRAMContractPromotionContext(lower string) bool {
	normalized := strings.NewReplacer(`"`, "", "`", "", "'", "").Replace(lower)
	for _, marker := range []string{
		"proves",
		"prove",
		"guarantees",
		"guarantee",
		"supports",
		"support",
		"delivers",
		"deliver",
	} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func ramContractPhraseAllowedAsExactNonClaim(phrase string, contexts ...string) bool {
	normalizedPhrase := strings.ReplaceAll(phrase, " ", "-")
	for _, context := range contexts {
		normalized := strings.NewReplacer(`"`, "", "`", "", "'", "").Replace(context)
		normalized = strings.ReplaceAll(normalized, " ", "-")
		if strings.Contains(normalized, "no-"+normalizedPhrase+"-claim") {
			return true
		}
	}
	return false
}

func verifyWASMBackendPlan(path string, plannedTargets []string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("%s: %v", path, err)
	}
	text := string(raw)
	required := []string{
		"Status: current",
		"Phase 0: Target contract",
		"Phase 1: WASM IR emitter",
		"Phase 2: WASI runner",
		"Phase 3: Web runtime",
		"Phase 4: v1.0 release gate",
		"go run ./tools/cmd/validate-targets",
		"bash scripts/release/v1_0/gate.sh",
		"wasmtime",
		"browser automation",
	}
	for _, target := range plannedTargets {
		required = append(required, "`"+target+"`")
		required = append(required, "./tetra smoke --target "+target+" --run=false")
	}
	for _, want := range required {
		if !strings.Contains(text, want) {
			return fmt.Errorf("%s: missing %q", path, want)
		}
	}
	return nil
}
