package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"tetra_language/compiler"
)

func verifyReleaseTruthDocs(paths []string) error {
	type confusingPattern struct {
		label string
		re    *regexp.Regexp
	}
	patterns := []confusingPattern{
		{label: "current.*v0.3", re: regexp.MustCompile(`(?is)\bcurrent\b.{0,120}\bv0\.3\.0\b`)},
		{label: "current.*v0.6", re: regexp.MustCompile(`(?is)\bcurrent\b.{0,120}\bv0\.6\b`)},
		{label: "v0.1.2", re: regexp.MustCompile(`\bv0\.1\.2\b`)},
		{label: "ready for v1.0", re: regexp.MustCompile(`(?is)\bready\s+for\s+` + "`?" + `v1\.0`)},
	}

	var errs []string
	for _, path := range paths {
		if releaseTruthDocExcluded(path) {
			continue
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", path, err))
			continue
		}
		text := string(raw)
		for _, pattern := range patterns {
			if pattern.re.MatchString(text) {
				errs = append(errs, fmt.Sprintf("%s: misleading release language matched %q", path, pattern.label))
			}
		}
		for _, claim := range forbiddenPublicPerformanceClaims(text) {
			errs = append(errs, fmt.Sprintf("%s: forbidden %s claim in release truth docs", path, claim))
		}
		for _, claim := range forbiddenPersistentObjectMemoryClaims(text) {
			errs = append(errs, fmt.Sprintf("%s: forbidden %s claim in release truth docs", path, claim))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

func releaseTruthDocExcluded(path string) bool {
	clean := filepath.ToSlash(path)
	base := strings.ToLower(filepath.Base(clean))
	if strings.Contains(base, "todo") {
		return true
	}
	return strings.HasPrefix(clean, "docs/plans/")
}

func verifyDoctestBlocks(paths []string) error {
	return verifyDoctestBlocksWithPolicy(paths, false)
}

func verifyRequiredDoctestBlocks(paths []string) error {
	return verifyDoctestBlocksWithPolicy(paths, true)
}

func verifyStableModuleDoctestCoverage(paths []string) error {
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("%s: %v", path, err)
		}
		blocks, err := extractTetraDoctests(string(raw))
		if err != nil {
			return fmt.Errorf("%s: %v", path, err)
		}
		name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		moduleRef := "lib.core." + name + "."
		covered := false
		for _, block := range blocks {
			if strings.Contains(block, moduleRef) {
				covered = true
				break
			}
		}
		if !covered {
			return fmt.Errorf("%s: doctest does not reference lib.core.%s", path, name)
		}
	}
	return nil
}

func verifyDoctestBlocksWithPolicy(paths []string, requireAtLeastOne bool) error {
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("%s: %v", path, err)
		}
		blocks, err := extractTetraDoctests(string(raw))
		if err != nil {
			return fmt.Errorf("%s: %v", path, err)
		}
		if requireAtLeastOne && len(blocks) == 0 {
			return fmt.Errorf("%s: missing tetra doctest block", path)
		}
		for i, block := range blocks {
			if _, err := compiler.ParseFile([]byte(block), fmt.Sprintf("%s#doctest%d", path, i+1)); err != nil {
				return fmt.Errorf("%s doctest %d: %v", path, i+1, err)
			}
		}
	}
	return nil
}

type specCodeBlock struct {
	lang      string
	info      string
	body      string
	startLine int
	check     bool
	skip      bool
}

func currentSpecMarkdownPaths() []string {
	paths, err := filepath.Glob(filepath.FromSlash("docs/spec/*.md"))
	if err != nil {
		return nil
	}
	sort.Strings(paths)
	return paths
}

func currentReleaseTruthDocPaths() []string {
	return []string{
		"README.md",
		filepath.FromSlash("docs/spec/current_supported_surface.md"),
		filepath.FromSlash("docs/spec/islands.md"),
		filepath.FromSlash("docs/spec/surface_v1.md"),
		filepath.FromSlash("docs/spec/v0_2_scope.md"),
		filepath.FromSlash("docs/spec/v1_feature_status.md"),
		filepath.FromSlash("docs/spec/v1_scope.md"),
		filepath.FromSlash("docs/user/async_actors_guide.md"),
		filepath.FromSlash("docs/user/eco_package_guide.md"),
		filepath.FromSlash("docs/user/examples_index.md"),
		filepath.FromSlash("docs/user/getting_started.md"),
		filepath.FromSlash("docs/user/language_tour.md"),
		filepath.FromSlash("docs/user/ownership_effects_guide.md"),
		filepath.FromSlash("docs/user/standard_library_guide.md"),
		filepath.FromSlash("docs/user/surface_guide.md"),
		filepath.FromSlash("docs/user/troubleshooting.md"),
		filepath.FromSlash("docs/user/wasm_ui_guide.md"),
		filepath.FromSlash("docs/release/memory_islands_surface_scope.md"),
	}
}

func surfaceReleaseDocPaths() []string {
	return []string{
		filepath.FromSlash("docs/spec/current_supported_surface.md"),
		filepath.FromSlash("docs/spec/surface_v1.md"),
		filepath.FromSlash("docs/release/surface_v1_release_contract.md"),
		filepath.FromSlash("docs/release/surface_v1_release_notes.md"),
		filepath.FromSlash("docs/release/surface_v1_release_audit.md"),
		filepath.FromSlash("docs/user/examples_index.md"),
		filepath.FromSlash("docs/user/surface_guide.md"),
		filepath.FromSlash("docs/user/surface_cookbook.md"),
		filepath.FromSlash("docs/user/surface_morph_recipe_cookbook.md"),
	}
}

func memoryIslandsSurfaceReleaseDocPaths() []string {
	return []string{
		"README.md",
		filepath.FromSlash("docs/spec/current_supported_surface.md"),
		filepath.FromSlash("docs/spec/islands.md"),
		filepath.FromSlash("docs/release/memory_islands_surface_scope.md"),
	}
}

func finalMemoryIslandsSurfaceProductionAuditDocPaths() []string {
	return []string{
		filepath.FromSlash("docs/audits/memory-islands-surface-final-production-readiness.md"),
	}
}

func memoryIslandsFinalProductionReadinessAuditDocPaths() []string {
	return []string{
		filepath.FromSlash("docs/audits/memory-islands-final-production-readiness.md"),
	}
}

func memoryIslandsFinalActorBenchmarkHandoffDocPaths() []string {
	return []string{
		filepath.FromSlash("docs/audits/memory-islands-final-production-handoff.md"),
	}
}

func defaultActorRuntimeFoundationDocPaths() []string {
	return []string{
		filepath.FromSlash("docs/spec/actors.md"),
		filepath.FromSlash("docs/user/async_actors_guide.md"),
		filepath.FromSlash("docs/design/actor_region_transfer.md"),
		filepath.FromSlash("docs/audits/actor-runtime-production-boundary-v1.md"),
		filepath.FromSlash("docs/audits/actor-runtime-production-foundation-final.md"),
		filepath.FromSlash("docs/checklists/actors_linux_smoke.md"),
		filepath.FromSlash("docs/checklists/actors_platform_smoke.md"),
	}
}

func verifyActorRuntimeFoundationDocs(paths []string, features []featureManifest) error {
	var errs []string
	var combined strings.Builder
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", path, err))
			continue
		}
		text := string(raw)
		combined.WriteString(text)
		combined.WriteByte('\n')
		for _, claim := range forbiddenActorRuntimeFoundationClaims(text) {
			errs = append(errs, fmt.Sprintf("%s: forbidden actor runtime foundation claim %q", path, claim))
		}
	}
	combinedText := combined.String()
	for _, required := range []string{
		"Actor runtime foundation scoped release truth",
		"tetra.actor.production_foundation.v1",
		"scripts/release/post_v0_4/actor-runtime-foundation-linux-x64-gate.sh",
		".github/workflows/ci.yml",
		".github/workflows/release-packages.yml",
		"reports/actor-runtime-foundation/final/actor-runtime-foundation-manifest.json",
		"reports/actor-runtime-foundation/final/artifact-hashes.json",
		"distributed-actors-linux-x64/distributed-actors-linux-x64.json",
		"parallel-production-linux-x64/parallel-production-linux-x64.json",
		"subordinate to current same-commit actor foundation gates",
		"no full Erlang/OTP actor runtime claim",
		"no cluster membership or reconnect/retry production claim",
		"no non-Linux distributed actor runtime support claim",
		"no distributed zero-copy pointer or region transfer claim",
		"no formal race proof claim",
		"no benchmark superiority, no C++/Rust parity, and no official benchmark claim",
		"Distributed Runtime Target Matrix",
		"| Target | Distributed actor runtime status | Current evidence | Promotion requirement |",
		"`linux-x64` | current scoped",
		"`macos-x64` | unsupported / nonclaim",
		"`windows-x64` | unsupported / nonclaim",
		"`wasm32-wasi` | unsupported / nonclaim",
		"`wasm32-web` | unsupported / nonclaim",
	} {
		if !strings.Contains(combinedText, required) {
			errs = append(errs, fmt.Sprintf("actor runtime foundation docs missing %q", required))
		}
	}
	if err := verifyActorRuntimeFoundationManifestFeature(features); err != nil {
		errs = append(errs, err.Error())
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

func verifyActorRuntimeFoundationManifestFeature(features []featureManifest) error {
	var feature *featureManifest
	for i := range features {
		if features[i].ID == "actors.distributed-runtime" {
			feature = &features[i]
			break
		}
	}
	if feature == nil {
		return fmt.Errorf("feature registry missing actors.distributed-runtime")
	}
	haystack := feature.Scope + " " + feature.Stability + " " + strings.Join(feature.Docs, " ")
	for _, required := range []string{
		"tetra.actor.production_foundation.v1",
		"actor-runtime-foundation-linux-x64-gate.sh",
		"non-Linux distributed",
		"distributed zero-copy",
		"cluster membership",
		"reconnect/retry production",
		"formal race proof",
		"docs/design/actor_region_transfer.md",
		"docs/audits/actor-runtime-production-boundary-v1.md",
		"docs/checklists/actors_linux_smoke.md",
		"docs/checklists/actors_platform_smoke.md",
	} {
		if !strings.Contains(haystack, required) {
			return fmt.Errorf("feature registry actors.distributed-runtime missing actor foundation phrase %q", required)
		}
	}
	return nil
}

func forbiddenActorRuntimeFoundationClaims(text string) []string {
	lower := strings.ToLower(text)
	var claims []string
	for _, phrase := range []string{
		"full production actor runtime",
		"full production actor-runtime",
		"production actor runtime",
		"production actor-runtime",
		"actor runtime production ready",
		"actor runtime is production ready",
		"actor production gate passed",
		"actor production gate is passed",
		"production actor gate passed",
		"production actor gate is passed",
		"full erlang/otp actor runtime",
		"erlang/otp actor runtime",
		"windows distributed actor runtime",
		"macos distributed actor runtime",
		"non-linux distributed actor runtime support",
		"non-linux distributed actor runtime target",
		"non-linux distributed actor targets",
		"distributed zero-copy",
		"distributed zero copy",
		"cluster membership",
		"reconnect/retry production",
		"formal race proof",
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
			if !explicitActorFoundationNonClaimContext(clause) && !explicitActorFoundationNonClaimContext(sentence) {
				claims = append(claims, phrase)
			}
			searchFrom = absolute + len(phrase)
		}
	}
	sort.Strings(claims)
	return compactStrings(claims)
}

func explicitActorFoundationNonClaimContext(lower string) bool {
	if explicitNonClaimContext(lower) {
		return true
	}
	for _, marker := range []string{
		"no cluster membership",
		"no reconnect/retry",
		"no non-linux distributed",
		"no distributed zero-copy",
		"no distributed zero copy",
		"no formal race proof",
		"does not implement",
		"does not mark",
		"not claimed",
		"not made",
		"remain rejected",
		"remains rejected",
		"rejected by",
		"requires separate promotion evidence",
		"require separate promotion evidence",
		"outside this claim",
		"outside the current",
		"outside this transfer contract",
		"outside the current transfer contract",
		"block a full",
		"blockers",
		"blocked",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func verifyMemoryIslandsSurfaceReleaseDocs(paths []string) error {
	var errs []string
	var combined strings.Builder
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", path, err))
			continue
		}
		text := string(raw)
		combined.WriteString(text)
		combined.WriteByte('\n')
		for _, claim := range forbiddenMemoryIslandsSurfaceReleaseClaims(text) {
			errs = append(errs, fmt.Sprintf("%s: forbidden Memory/Islands/Surface release claim %q", path, claim))
		}
	}
	combinedLower := strings.ToLower(combined.String())
	for _, required := range []string{
		"memory/islands/surface scoped release truth",
		"memory-islands-surface-production-gate.sh",
		"validate-memory-islands-surface-production",
		"validate-island-proof",
		"--islands-debug",
		"islands-debug-smoke.json",
		"island-proof-verifier.json",
		"island-proof-fuzz-summary.json",
		"memory-islands-surface-production-manifest.json",
		"artifact-hashes.json",
		"leak/resource finalization evidence",
		"surface-v1-linux-web",
		"no memory 100% claim",
		"no arbitrary unsafe external pointer safety",
		"no full formal proof",
		"no full target parity",
		"no all-target surface claim",
		"no production object memory claim",
		"no production persistent memory claim",
		"not a clean release-candidate checkout claim",
	} {
		if !strings.Contains(combinedLower, required) {
			errs = append(errs, fmt.Sprintf("Memory/Islands/Surface release docs missing %q", required))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

func forbiddenMemoryIslandsSurfaceReleaseClaims(text string) []string {
	lower := strings.ToLower(text)
	claims := forbiddenPublicPerformanceClaims(text)
	for _, phrase := range []string{
		"fully production-ready",
		"100% production-ready",
		"production-ready across all targets",
		"arbitrary unsafe external pointer safety",
		"full formal proof",
		"full target parity",
		"all-platform surface",
		"clean release-candidate checkout",
	} {
		searchFrom := 0
		for {
			index := strings.Index(lower[searchFrom:], phrase)
			if index < 0 {
				break
			}
			absolute := searchFrom + index
			if !explicitNonClaimContext(clauseAround(lower, absolute, len(phrase), 240)) {
				claims = append(claims, phrase)
			}
			searchFrom = absolute + len(phrase)
		}
	}
	sort.Strings(claims)
	return compactStrings(claims)
}

var (
	finalAuditGitHeadPattern = regexp.MustCompile(`(?i)\b[0-9a-f]{40}\b`)
	finalAuditSHA256Pattern  = regexp.MustCompile(`(?i)\b[0-9a-f]{64}\b`)
)

func verifyFinalMemoryIslandsSurfaceProductionAudit(paths []string) error {
	var errs []string
	var combined strings.Builder
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", path, err))
			continue
		}
		text := string(raw)
		combined.WriteString(text)
		combined.WriteByte('\n')
		for _, claim := range forbiddenFinalMemoryIslandsSurfaceProductionAuditClaims(text) {
			errs = append(errs, fmt.Sprintf("%s: forbidden final Memory/Islands/Surface audit claim %q", path, claim))
		}
	}
	combinedText := combined.String()
	combinedLower := strings.ToLower(combinedText)
	for _, required := range []string{
		"memory/islands/surface final production readiness audit",
		"git head:",
		"dirty working tree",
		"memory verdict: `prod_stable_scoped`",
		"islands verdict: `prod_stable_scoped`",
		"surface verdict: `prod_stable_scoped`",
		"integrated verdict: `prod_stable_scoped`",
		"go test -buildvcs=false ./tools/cmd/verify-docs -run 'final|production|audit|overclaim' -count=1",
		"git diff --check",
		"git status --short",
		"go run -buildvcs=false ./tools/cmd/verify-docs --manifest docs/generated/manifest.json",
		"reports/mis-ideal/p13/integrated",
		"reports/mis-ideal/p15/docs-manifest-overclaim.md",
		"changed files",
		".github/workflows/ci.yml",
		"docs/generated/manifest.json",
		"docs/release/memory_islands_surface_scope.md",
		"scripts/release/post_v0_4/memory-islands-surface-production-gate.sh",
		"tools/cmd/validate-island-proof",
		"tools/cmd/validate-memory-islands-surface-production",
		"residual risks",
		"remote github actions",
		"tools/cmd/dump-project",
		"tools/validators/postv04prod",
		"no memory 100% claim",
		"no arbitrary unsafe external pointer safety",
		"no full formal proof",
		"no full target parity",
		"no all-target surface claim",
	} {
		if !strings.Contains(combinedLower, required) {
			errs = append(errs, fmt.Sprintf("final Memory/Islands/Surface audit missing %q", required))
		}
	}
	if !finalAuditGitHeadPattern.MatchString(combinedText) {
		errs = append(errs, "final Memory/Islands/Surface audit missing 40-character git head")
	}
	if !finalAuditSHA256Pattern.MatchString(combinedText) {
		errs = append(errs, "final Memory/Islands/Surface audit missing sha256 evidence")
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

func verifyMemoryIslandsFinalProductionReadinessAudit(paths []string) error {
	var errs []string
	var combined strings.Builder
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			errs = append(errs, fmt.Sprintf("%s: %v", path, err))
			continue
		}
		text := string(raw)
		combined.WriteString(text)
		combined.WriteByte('\n')
		for _, claim := range forbiddenFinalMemoryIslandsSurfaceProductionAuditClaims(text) {
			errs = append(errs, fmt.Sprintf("%s: forbidden final Memory/Islands audit claim %q", path, claim))
		}
	}
	combinedText := combined.String()
	if combinedText == "" {
		if len(errs) > 0 {
			return fmt.Errorf("%s", strings.Join(errs, "; "))
		}
		return nil
	}
	combinedLower := strings.ToLower(combinedText)
	for _, required := range []string{
		"memory/islands final production readiness audit",
		"git head:",
		"working tree:",
		"dirty working tree",
		"memory verdict: `prod_stable_scoped`",
		"islands verdict: `prod_stable_scoped`",
		"integrated gate verdict: `prod_stable_scoped`",
		"memory/islands scope:",
		"integrated gate scope:",
		"## command log",
		"git status --short",
		"git rev-parse head",
		"go test -buildvcs=false ./compiler/... ./cli/... ./tools/... -count=1",
		"go test -race -buildvcs=false ./compiler/internal/islandkernel ./compiler/internal/memoryfacts ./compiler/internal/memorymodel ./compiler/internal/semantics ./compiler/internal/plir ./compiler/internal/validation ./cli/internal/actornet -count=1",
		"memory-production-linux-x64-smoke.sh --report-dir reports/memory-islands-ideal/final/memory-production",
		"memory-islands-surface-production-gate.sh --report-dir reports/memory-islands-ideal/final/integrated",
		"go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json",
		"go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json",
		"git diff --check",
		"## artifact log",
		"reports/memory-islands-ideal/final/memory-production",
		"reports/memory-islands-ideal/final/integrated",
		"reports/memory-islands-ideal/final/artifact-sha256.txt",
		"## artifact hashes",
		"## residual risks",
		"remote github actions",
		"## nonclaims",
		"no memory 100% claim",
		"no arbitrary unsafe external pointer safety",
		"no full formal proof",
		"no full target parity",
		"no production actor runtime",
		"no official benchmark result",
		"not a clean release-candidate checkout claim",
	} {
		if !strings.Contains(combinedLower, required) {
			errs = append(errs, fmt.Sprintf("final Memory/Islands audit missing %q", required))
		}
	}
	if !finalAuditGitHeadPattern.MatchString(combinedText) {
		errs = append(errs, "final Memory/Islands audit missing 40-character git head")
	}
	if !finalAuditSHA256Pattern.MatchString(combinedText) {
		errs = append(errs, "final Memory/Islands audit missing sha256 evidence")
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

func verifyMemoryIslandsFinalActorBenchmarkHandoff(paths []string) error {
	var errs []string
	var combined strings.Builder
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", path, err))
			continue
		}
		text := string(raw)
		combined.WriteString(text)
		combined.WriteByte('\n')
		for _, claim := range forbiddenFinalActorBenchmarkHandoffClaims(text) {
			errs = append(errs, fmt.Sprintf("%s: forbidden final actor/benchmark handoff claim %q", path, claim))
		}
	}
	combinedText := combined.String()
	combinedLower := strings.ToLower(combinedText)
	for _, required := range []string{
		"memory/islands final production audit and actor handoff",
		"final verdict: `prod_stable_scoped`",
		"memory/islands baseline:",
		"docs/audits/memory-islands-final-production-readiness.md",
		"reports/memory-islands-ideal/final/artifact-sha256.txt",
		"actor handoff readiness:",
		"actor phase may start",
		"separate actor runtime production foundation plan",
		"actor runtime production status:",
		"not started in this plan",
		"actor phase preconditions:",
		"production actor gate must prove",
		"scheduler",
		"mailbox backpressure",
		"message exhaustion/reclamation",
		"race-safety",
		"cross-target distributed runtime gates",
		"structured concurrency",
		"fake-evidence rejection",
		"docs/audits/actor-runtime-production-boundary-v1.md",
		"memisl-p10",
		"memory-boundary handoff evidence",
		"benchmark preconditions:",
		"tier 0/tier 1",
		"measured evidence",
		"no official benchmark result",
		"no performance superiority",
		"no c++/rust parity",
		"no measured speed comparison",
		"nonclaims:",
		"no production actor runtime",
		"no actor production gate passed",
		"no `prod_ready_proven` claim",
	} {
		if !strings.Contains(combinedLower, required) {
			errs = append(errs, fmt.Sprintf("final actor/benchmark handoff missing %q", required))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

func forbiddenFinalMemoryIslandsSurfaceProductionAuditClaims(text string) []string {
	lower := strings.ToLower(text)
	claims := forbiddenMemoryIslandsSurfaceReleaseClaims(text)
	for _, phrase := range []string{
		"prod_ready_proven",
		"prod_ready",
		"all gates passed",
		"no residual risks",
		"no blockers",
	} {
		searchFrom := 0
		for {
			index := strings.Index(lower[searchFrom:], phrase)
			if index < 0 {
				break
			}
			absolute := searchFrom + index
			if !explicitNonClaimContext(clauseAround(lower, absolute, len(phrase), 240)) {
				claims = append(claims, phrase)
			}
			searchFrom = absolute + len(phrase)
		}
	}
	sort.Strings(claims)
	return compactStrings(claims)
}

func forbiddenFinalActorBenchmarkHandoffClaims(text string) []string {
	lower := strings.ToLower(text)
	claims := forbiddenPublicPerformanceClaims(text)
	for _, phrase := range []string{
		"production actor runtime",
		"full production actor runtime",
		"actor runtime production ready",
		"actor runtime is production ready",
		"actor production gate passed",
		"actor production gate is passed",
		"production actor gate passed",
		"production actor gate is passed",
		"benchmark phase may claim",
		"performance superiority",
		"measured speed comparison",
		"c++/rust parity",
		"c++ and rust parity",
		"rust/c++ parity",
		"prod_ready_proven",
		"prod_ready",
	} {
		searchFrom := 0
		for {
			index := strings.Index(lower[searchFrom:], phrase)
			if index < 0 {
				break
			}
			absolute := searchFrom + index
			if !explicitNonClaimContext(clauseAround(lower, absolute, len(phrase), 240)) {
				claims = append(claims, phrase)
			}
			searchFrom = absolute + len(phrase)
		}
	}
	sort.Strings(claims)
	return compactStrings(claims)
}

func clauseAround(text string, index int, length int, maxSide int) string {
	start := index
	for start > 0 && !claimClauseBoundary(text[start-1]) {
		start--
		if index-start >= maxSide {
			break
		}
	}
	end := index + length
	for end < len(text) && !claimClauseBoundary(text[end]) {
		end++
		if end-(index+length) >= maxSide {
			break
		}
	}
	return text[start:end]
}

func claimClauseBoundary(b byte) bool {
	return b == '\n' || b == '.' || b == '!' || b == '?' || b == ';'
}

func verifySurfaceReleaseDocs(paths []string) error {
	var errs []string
	var combined strings.Builder
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", path, err))
			continue
		}
		text := string(raw)
		combined.WriteString(text)
		combined.WriteByte('\n')
		if strings.Contains(text, "/tmp/") {
			errs = append(errs, fmt.Sprintf("%s: Surface release docs must not use /tmp paths as current release evidence", path))
		}
		if surfaceReleaseDocRequiresP28Governance(path) {
			errs = append(errs, verifySurfaceReleaseDocP28Governance(path, text)...)
		}
		for _, clause := range surfaceReleaseDocClauses(text) {
			lower := strings.ToLower(clause)
			if surfaceReleaseClauseBoundaryOnly(lower) {
				continue
			}
			if containsAnySubstring(lower, []string{"macos surface", "macos/windows surface"}) &&
				containsAnySubstring(lower, []string{"current", "release-ready", "production-supported", "production supported"}) &&
				!surfaceReleaseExplicitNonClaimContext(lower) {
				errs = append(errs, fmt.Sprintf("%s: macOS Surface fake current claim: %q", path, strings.TrimSpace(clause)))
			}
			if containsAnySubstring(lower, []string{"windows surface", "macos/windows surface"}) &&
				containsAnySubstring(lower, []string{"current", "release-ready", "production-supported", "production supported"}) &&
				!surfaceReleaseExplicitNonClaimContext(lower) {
				errs = append(errs, fmt.Sprintf("%s: Windows Surface fake current claim: %q", path, strings.TrimSpace(clause)))
			}
			if strings.Contains(lower, "wasm32-wasi") && strings.Contains(lower, "surface") && surfaceReleaseClaimPromotes(lower) && !surfaceReleaseExplicitNonClaimContext(lower) {
				errs = append(errs, fmt.Sprintf("%s: wasm32-wasi Surface fake current claim: %q", path, strings.TrimSpace(clause)))
			}
			if strings.Contains(lower, "cross-platform") && surfaceReleaseClaimPromotes(lower) && !surfaceReleaseExplicitNonClaimContext(lower) {
				errs = append(errs, fmt.Sprintf("%s: cross-platform Surface fake production claim: %q", path, strings.TrimSpace(clause)))
			}
			if strings.Contains(lower, "gpu") && surfaceReleaseClaimPromotes(lower) && !surfaceReleaseExplicitNonClaimContext(lower) {
				errs = append(errs, fmt.Sprintf("%s: GPU Surface fake production claim: %q", path, strings.TrimSpace(clause)))
			}
			if containsAnySubstring(lower, []string{"platform-native widget", "native widget", "platform widget"}) &&
				surfaceReleaseClaimPromotes(lower) && !surfaceReleaseExplicitNonClaimContext(lower) {
				errs = append(errs, fmt.Sprintf("%s: native widget Surface fake production claim: %q", path, strings.TrimSpace(clause)))
			}
			if strings.Contains(lower, "rich text") && surfaceReleaseClaimPromotes(lower) && !surfaceReleaseExplicitNonClaimContext(lower) {
				errs = append(errs, fmt.Sprintf("%s: rich text Surface fake production claim: %q", path, strings.TrimSpace(clause)))
			}
			if containsAnySubstring(lower, []string{"screen-reader", "screen reader", "at-spi"}) &&
				surfaceReleaseClaimPromotes(lower) && !surfaceReleaseExplicitNonClaimContext(lower) {
				errs = append(errs, fmt.Sprintf("%s: screen-reader Surface fake production claim: %q", path, strings.TrimSpace(clause)))
			}
			if strings.Contains(lower, "metadata-only") && strings.Contains(lower, "production accessibility") && !surfaceReleaseExplicitNonClaimContext(lower) {
				errs = append(errs, fmt.Sprintf("%s: metadata-only accessibility fake production claim: %q", path, strings.TrimSpace(clause)))
			}
			if containsAnySubstring(lower, []string{"dom ui", "html ui"}) && surfaceReleaseClaimPromotes(lower) && !surfaceReleaseExplicitNonClaimContext(lower) {
				errs = append(errs, fmt.Sprintf("%s: DOM UI fake production claim: %q", path, strings.TrimSpace(clause)))
			}
			if strings.Contains(lower, "dom ui") && strings.Contains(lower, "surface model") && !surfaceReleaseExplicitNonClaimContext(lower) {
				errs = append(errs, fmt.Sprintf("%s: DOM UI fake Surface model claim: %q", path, strings.TrimSpace(clause)))
			}
			if strings.Contains(lower, "react") && surfaceReleaseClaimPromotes(lower) && !surfaceReleaseExplicitNonClaimContext(lower) {
				errs = append(errs, fmt.Sprintf("%s: React Surface fake production claim: %q", path, strings.TrimSpace(clause)))
			}
			if containsAnySubstring(lower, []string{"core surface primitive", "surface core primitive", "core primitive"}) &&
				containsAnySubstring(lower, []string{"button", "textfield", "text field", "card", "sidebar", "modal"}) &&
				!surfaceReleaseExplicitNonClaimContext(lower) {
				errs = append(errs, fmt.Sprintf("%s: core widget primitive fake Surface claim: %q", path, strings.TrimSpace(clause)))
			}
			if containsAnySubstring(lower, []string{"user js", "user javascript"}) &&
				containsAnySubstring(lower, []string{"allowed", "may use", "can use"}) {
				errs = append(errs, fmt.Sprintf("%s: user JS fake allowance claim: %q", path, strings.TrimSpace(clause)))
			}
			if strings.Contains(lower, "final current claim") {
				errs = append(errs, fmt.Sprintf("%s: final current claim ownership must stay with P29: %q", path, strings.TrimSpace(clause)))
			}
		}
	}
	combinedLower := strings.ToLower(combined.String())
	for _, required := range []string{"unsupported", "macos", "windows", "wasm32-wasi"} {
		if !strings.Contains(combinedLower, required) {
			errs = append(errs, fmt.Sprintf("Surface release docs missing unsupported targets evidence: %s", required))
		}
	}
	if !strings.Contains(combined.String(), "bash scripts/release/surface/release-gate.sh") {
		errs = append(errs, "Surface release docs missing release-gate.sh command link")
	}
	if !strings.Contains(combined.String(), "bash scripts/release/surface/product-gate.sh") {
		errs = append(errs, "Surface release docs missing product-gate.sh command link")
	}
	for _, tier := range []string{"PROD_STABLE_SCOPED", "BETA_TARGET_HOST", "EXPERIMENTAL", "UNSUPPORTED", "NONCLAIM"} {
		if !strings.Contains(combined.String(), tier) {
			errs = append(errs, fmt.Sprintf("Surface release docs missing claim tier %s", tier))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

func surfaceReleaseDocRequiresP28Governance(path string) bool {
	base := filepath.ToSlash(path)
	for _, required := range []string{
		"docs/spec/current_supported_surface.md",
		"docs/spec/surface_v1.md",
		"docs/spec/surface_morph.md",
		"docs/user/surface_guide.md",
		"docs/user/examples_index.md",
		"docs/user/surface_cookbook.md",
		"docs/release/surface_v1_release_audit.md",
		"docs/release/surface_v1_release_contract.md",
		"docs/release/surface_v1_release_notes.md",
	} {
		if strings.HasSuffix(base, required) {
			return true
		}
	}
	switch strings.ToLower(filepath.Base(base)) {
	case "current_supported_surface.md",
		"surface_v1.md",
		"surface_morph.md",
		"surface_guide.md",
		"examples_index.md",
		"surface_cookbook.md",
		"surface_v1_release_audit.md",
		"surface_v1_release_contract.md",
		"surface_v1_release_notes.md":
		return true
	default:
		return false
	}
}

func verifySurfaceReleaseDocP28Governance(path string, text string) []string {
	var errs []string
	for _, tier := range []string{"PROD_STABLE_SCOPED", "BETA_TARGET_HOST", "EXPERIMENTAL", "UNSUPPORTED", "NONCLAIM"} {
		if !strings.Contains(text, tier) {
			errs = append(errs, fmt.Sprintf("%s: Surface release doc missing claim tier %s", path, tier))
		}
	}
	if !strings.Contains(text, "bash scripts/release/surface/product-gate.sh") {
		errs = append(errs, fmt.Sprintf("%s: Surface release doc missing product-gate.sh command link", path))
	}
	return errs
}

func surfaceReleaseDocClauses(text string) []string {
	return strings.FieldsFunc(text, func(r rune) bool {
		return r == '.' || r == '\n' || r == ';'
	})
}

func surfaceReleaseClauseBoundaryOnly(lower string) bool {
	return surfaceReleaseClauseSafe(lower) && !surfaceReleaseClaimPromotes(lower)
}

func surfaceReleaseClauseSafe(lower string) bool {
	return containsAnySubstring(lower, []string{
		"future work",
		"remain future",
		"without",
		"unsupported",
		"outside",
		"remain outside",
		"requires real",
		"require real",
		"no release evidence",
		"forbid",
		"forbids",
		"rejected",
		"invalid until",
	})
}

func surfaceReleaseExplicitNonClaimContext(lower string) bool {
	return containsAnySubstring(lower, []string{
		" not ",
		"not ",
		"no ",
		"non-goal",
		"nonclaim",
		"non-claim",
		"not claimed",
		"not claim",
		"must not",
		"cannot",
		"forbid",
		"forbids",
		"rejected",
		"reject",
		"outside",
		"remain outside",
	})
}

func surfaceReleaseClaimPromotes(lower string) bool {
	return containsAnySubstring(lower, []string{
		"release-ready",
		"release ready",
		"production-supported",
		"production supported",
		"production support",
	}) ||
		containsVerifyDocsClaimWord(lower, "current") ||
		containsVerifyDocsClaimWord(lower, "production") ||
		containsVerifyDocsClaimWord(lower, "supported")
}

func containsVerifyDocsClaimWord(lower string, word string) bool {
	for _, field := range strings.FieldsFunc(lower, func(r rune) bool {
		return r < 'a' || r > 'z'
	}) {
		if field == word {
			return true
		}
	}
	return false
}

func containsAnySubstring(text string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}
