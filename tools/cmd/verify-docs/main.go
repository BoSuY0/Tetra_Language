package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"tetra_language/compiler"
	ctarget "tetra_language/compiler/target"
)

type manifest struct {
	Targets []struct {
		Triple string `json:"triple"`
	} `json:"targets"`
	Builtins []struct {
		Name         string `json:"name"`
		UnsafePolicy string `json:"unsafe_policy"`
	} `json:"builtins"`
	RuntimeABI struct {
		ActorsSupportedTargets   []string `json:"actors_supported_targets"`
		ActorsRequiredSymbols    []string `json:"actors_required_symbols"`
		TimeRequiredSymbols      []string `json:"time_required_symbols"`
		ActorsProgramGlueSymbols []string `json:"actors_program_glue_symbols"`
	} `json:"runtime_abi"`
	Features []featureManifest `json:"features"`
}

type featureManifest struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Status    string   `json:"status"`
	Since     string   `json:"since,omitempty"`
	Scope     string   `json:"scope"`
	Stability string   `json:"stability"`
	Docs      []string `json:"docs"`
}

func main() {
	manifestPath := flag.String("manifest", "docs/generated/manifest.json", "path to generated manifest json")
	flag.Parse()

	data, err := os.ReadFile(*manifestPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	var m manifest
	if err := json.Unmarshal(data, &m); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	var errs []string
	checkContains := func(path string, required []string) {
		b, err := os.ReadFile(path)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", path, err))
			return
		}
		s := string(b)
		for _, r := range required {
			if !strings.Contains(s, r) {
				errs = append(errs, fmt.Sprintf("%s: missing %q", path, r))
			}
		}
	}

	// Specs must mention supported targets and runtime ABI surface.
	checkContains("docs/spec/actors.md", m.RuntimeABI.ActorsSupportedTargets)
	runtimeSymbols := append([]string(nil), m.RuntimeABI.ActorsRequiredSymbols...)
	runtimeSymbols = append(runtimeSymbols, m.RuntimeABI.TimeRequiredSymbols...)
	runtimeSymbols = append(runtimeSymbols, m.RuntimeABI.ActorsProgramGlueSymbols...)
	checkContains("docs/spec/runtime_abi.md", runtimeSymbols)

	// Unsafe spec must list all builtins that require unsafe (always/conditional).
	var unsafeBuiltins []string
	for _, b := range m.Builtins {
		if b.UnsafePolicy == "always" || b.UnsafePolicy == "conditional" {
			unsafeBuiltins = append(unsafeBuiltins, b.Name)
		}
	}
	checkContains("docs/spec/unsafe.md", unsafeBuiltins)

	// CLI should advertise the same target triples (minimum parity).
	var triples []string
	for _, t := range m.Targets {
		if t.Triple != "" {
			triples = append(triples, t.Triple)
		}
	}
	checkContains(filepath.FromSlash("cli/cmd/tetra/main.go"), triples)

	stableModulePaths := currentStableModulePaths()
	experimentalModulePaths := currentExperimentalModulePaths()
	if err := verifyStdlibModulePaths(stableModulePaths, experimentalModulePaths); err != nil {
		errs = append(errs, err.Error())
	}
	if err := verifyStableModuleDocs(stableModulePaths); err != nil {
		errs = append(errs, err.Error())
	}
	if err := verifyDoctestBlocks([]string{"README.md", "docs/spec/flow_syntax_mvp.md", "docs/spec/ui_v1.md"}); err != nil {
		errs = append(errs, err.Error())
	}
	if err := verifySpecCodeBlocks(currentSpecMarkdownPaths()); err != nil {
		errs = append(errs, err.Error())
	}
	if err := verifyRequiredDoctestBlocks(stableModulePaths); err != nil {
		errs = append(errs, err.Error())
	}
	if err := verifyStableModuleDoctestCoverage(stableModulePaths); err != nil {
		errs = append(errs, err.Error())
	}
	if err := verifyStableModuleExamples(stableModulePaths); err != nil {
		errs = append(errs, err.Error())
	}
	if err := verifyStableExamplesDoNotImportExperimental(stableModuleExamplePaths(stableModulePaths)); err != nil {
		errs = append(errs, err.Error())
	}
	if err := verifyStableModuleEffectsMetadata(stableModulePaths); err != nil {
		errs = append(errs, err.Error())
	}
	if err := verifyExperimentalModuleMirrors(experimentalModulePaths); err != nil {
		errs = append(errs, err.Error())
	}
	if err := verifyStdlibGuide(filepath.FromSlash("docs/user/standard_library_guide.md"), stableModulePaths, experimentalModulePaths); err != nil {
		errs = append(errs, err.Error())
	}
	if err := verifyEpic14ExampleIndex(filepath.FromSlash("docs/user/examples_index.md")); err != nil {
		errs = append(errs, err.Error())
	}
	if err := verifyReleaseTruthDocs(currentReleaseTruthDocPaths()); err != nil {
		errs = append(errs, err.Error())
	}
	if err := verifyFeatureRegistry(m.Features); err != nil {
		errs = append(errs, err.Error())
	}
	if err := verifyWASMBackendPlan("docs/backend/wasm_backend_plan.md", ctarget.WASMTriples()); err != nil {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintln(os.Stderr, "verify-docs:", e)
		}
		os.Exit(1)
	}
}

func verifyWASMBackendPlan(path string, plannedTargets []string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("%s: %v", path, err)
	}
	text := string(raw)
	required := []string{
		"Status: planned",
		"Phase 0: Target contract",
		"Phase 1: WASM IR emitter",
		"Phase 2: WASI runner",
		"Phase 3: Web runtime",
		"Phase 4: v1.0 release gate",
		"go run ./tools/cmd/validate-targets",
		"bash scripts/release_v1_0_gate.sh",
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

func verifyFeatureRegistry(features []featureManifest) error {
	if len(features) == 0 {
		return fmt.Errorf("feature registry is required in generated manifest")
	}
	allowedStatus := map[string]bool{
		"current":      true,
		"experimental": true,
		"planned":      true,
		"post-v1":      true,
	}
	requiredStatus := map[string]bool{
		"current":      false,
		"experimental": false,
		"planned":      false,
		"post-v1":      false,
	}
	requiredIDs := map[string]string{
		"cli.core":                            "current",
		"language.flow":                       "current",
		"language.callable-mvp":               "current",
		"language.callable-level1":            "experimental",
		"targets.wasm-build-only":             "current",
		"stdlib.experimental-mirrors":         "experimental",
		"language.enum-payload-match":         "experimental",
		"language.callable-level2":            "planned",
		"wasm.runtime-execution":              "planned",
		"language.full-v1-guarantees":         "planned",
		"eco.distributed-network":             "post-v1",
		"language.full-first-class-callables": "post-v1",
	}
	seen := map[string]string{}
	var currentCount int
	for _, feature := range features {
		if feature.ID == "" {
			return fmt.Errorf("feature registry entry missing id")
		}
		if feature.Name == "" || feature.Scope == "" || feature.Stability == "" {
			return fmt.Errorf("feature %s missing name, scope, or stability", feature.ID)
		}
		if !allowedStatus[feature.Status] {
			return fmt.Errorf("feature %s has invalid status %q", feature.ID, feature.Status)
		}
		if seenStatus, ok := seen[feature.ID]; ok {
			return fmt.Errorf("feature %s is duplicated with statuses %s and %s", feature.ID, seenStatus, feature.Status)
		}
		seen[feature.ID] = feature.Status
		requiredStatus[feature.Status] = true
		if feature.Status == "current" {
			currentCount++
			if feature.Since == "" {
				return fmt.Errorf("current feature %s missing since", feature.ID)
			}
		}
		if len(feature.Docs) == 0 {
			return fmt.Errorf("feature %s must cite docs", feature.ID)
		}
		for _, doc := range feature.Docs {
			if doc == "" {
				return fmt.Errorf("feature %s has empty doc reference", feature.ID)
			}
			docPath := filepath.ToSlash(doc)
			if filepath.IsAbs(doc) || strings.Contains(docPath, "..") {
				return fmt.Errorf("feature %s has unsafe doc reference %q", feature.ID, doc)
			}
			if !strings.HasPrefix(docPath, "docs/") || !strings.HasSuffix(docPath, ".md") {
				return fmt.Errorf("feature %s doc reference %q must point at docs/*.md", feature.ID, doc)
			}
			if _, err := statFromRepoRoot(docPath); err != nil {
				return fmt.Errorf("feature %s doc reference %q is not readable: %v", feature.ID, doc, err)
			}
		}
	}
	if currentCount == 0 {
		return fmt.Errorf("feature registry must include current features")
	}
	for status, present := range requiredStatus {
		if !present {
			return fmt.Errorf("feature registry missing %s feature", status)
		}
	}
	for id, wantStatus := range requiredIDs {
		if gotStatus, ok := seen[id]; !ok {
			return fmt.Errorf("feature registry missing %s", id)
		} else if gotStatus != wantStatus {
			return fmt.Errorf("feature registry %s status = %s, want %s", id, gotStatus, wantStatus)
		}
	}
	return nil
}

func statFromRepoRoot(path string) (os.FileInfo, error) {
	if info, err := os.Stat(filepath.FromSlash(path)); err == nil {
		return info, nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	for dir := wd; ; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, filepath.FromSlash(path))
		if info, err := os.Stat(candidate); err == nil {
			return info, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return nil, os.ErrNotExist
}

func verifyReleaseTruthDocs(paths []string) error {
	type confusingPattern struct {
		label string
		re    *regexp.Regexp
	}
	patterns := []confusingPattern{
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
		filepath.FromSlash("docs/user/troubleshooting.md"),
		filepath.FromSlash("docs/user/wasm_ui_guide.md"),
	}
}

func verifySpecCodeBlocks(paths []string) error {
	var errs []string
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", path, err))
			continue
		}
		blocks, err := extractSpecCodeBlocks(string(raw))
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", path, err))
			continue
		}
		for i, block := range blocks {
			if block.skip {
				continue
			}
			filename := fmt.Sprintf("%s#spec%d", path, i+1)
			if _, err := compiler.ParseFile([]byte(block.body), filename); err != nil {
				errs = append(errs, fmt.Sprintf("%s spec block %d parse: %v", path, i+1, err))
				continue
			}
			if !block.check {
				continue
			}
			prog, err := compiler.Parse([]byte(block.body))
			if err != nil {
				errs = append(errs, fmt.Sprintf("%s spec block %d check setup: %v", path, i+1, err))
				continue
			}
			if _, err := compiler.Check(prog); err != nil {
				errs = append(errs, fmt.Sprintf("%s spec block %d check: %v", path, i+1, err))
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

func extractSpecCodeBlocks(doc string) ([]specCodeBlock, error) {
	var blocks []specCodeBlock
	lines := strings.Split(doc, "\n")
	inBlock := false
	var current []string
	var block specCodeBlock
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !inBlock {
			lang, info, ok := specCodeFenceInfo(trimmed)
			if !ok {
				continue
			}
			inBlock = true
			current = nil
			block = specCodeBlock{
				lang:      lang,
				info:      info,
				startLine: i + 1,
				check:     specCodeBlockHasTag(info, "check"),
				skip:      specCodeBlockSkipped(info),
			}
			continue
		}
		if trimmed == "```" {
			block.body = strings.Join(current, "\n") + "\n"
			blocks = append(blocks, block)
			inBlock = false
			current = nil
			block = specCodeBlock{}
			continue
		}
		current = append(current, line)
	}
	if inBlock {
		return nil, fmt.Errorf("unterminated %s spec block starting at line %d", block.lang, block.startLine)
	}
	return blocks, nil
}

func specCodeFenceInfo(trimmed string) (lang string, info string, ok bool) {
	if !strings.HasPrefix(trimmed, "```") {
		return "", "", false
	}
	info = strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
	if info == "" {
		return "", "", false
	}
	fields := strings.Fields(info)
	if len(fields) == 0 {
		return "", "", false
	}
	lang = strings.ToLower(fields[0])
	if lang != "tetra" && lang != "t4" {
		return "", "", false
	}
	return lang, strings.ToLower(info), true
}

func specCodeBlockSkipped(info string) bool {
	for _, tag := range []string{"pseudocode", "negative", "unsupported", "skip", "noverify", "no-verify"} {
		if specCodeBlockHasTag(info, tag) {
			return true
		}
	}
	return false
}

func specCodeBlockHasTag(info string, tag string) bool {
	for _, field := range strings.Fields(strings.ToLower(info)) {
		if field == tag {
			return true
		}
	}
	return false
}

func extractTetraDoctests(doc string) ([]string, error) {
	var blocks []string
	lines := strings.Split(doc, "\n")
	inBlock := false
	commentBlock := false
	var current []string
	startLine := 0
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		commentLine, hasCommentPrefix := stripLineCommentPrefix(line)
		commentTrimmed := strings.TrimSpace(commentLine)
		if !inBlock {
			switch {
			case trimmed == "```tetra doctest":
				inBlock = true
				commentBlock = false
				current = nil
				startLine = i + 1
			case hasCommentPrefix && commentTrimmed == "```tetra doctest":
				inBlock = true
				commentBlock = true
				current = nil
				startLine = i + 1
			}
			continue
		}
		if (!commentBlock && trimmed == "```") || (commentBlock && hasCommentPrefix && commentTrimmed == "```") {
			blocks = append(blocks, strings.Join(current, "\n")+"\n")
			inBlock = false
			commentBlock = false
			current = nil
			startLine = 0
			continue
		}
		if commentBlock {
			if !hasCommentPrefix {
				return nil, fmt.Errorf("non-comment line in tetra doctest block starting at line %d", startLine)
			}
			current = append(current, commentLine)
			continue
		}
		current = append(current, line)
	}
	if inBlock {
		return nil, fmt.Errorf("unterminated tetra doctest block starting at line %d", startLine)
	}
	return blocks, nil
}

func stripLineCommentPrefix(line string) (string, bool) {
	trimmedLeft := strings.TrimLeft(line, " \t")
	if !strings.HasPrefix(trimmedLeft, "//") {
		return "", false
	}
	afterPrefix := strings.TrimPrefix(trimmedLeft, "//")
	if strings.HasPrefix(afterPrefix, " ") {
		afterPrefix = strings.TrimPrefix(afterPrefix, " ")
	}
	return afterPrefix, true
}

func verifyStableModuleDocs(paths []string) error {
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("%s: %v", path, err)
		}
		lines := strings.Split(string(raw), "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue
			}
			if !strings.HasPrefix(trimmed, "//") {
				return fmt.Errorf("%s: missing stable module docs comment", path)
			}
			break
		}
	}
	return nil
}

func verifyStableModuleExamples(paths []string) error {
	for _, path := range paths {
		name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		example := stableModuleExamplePath(name)
		if example == "" {
			return fmt.Errorf("%s: no stable module example mapping for %q", path, name)
		}
		raw, err := os.ReadFile(example)
		if err != nil {
			return fmt.Errorf("%s: missing stable module example %q: %v", path, example, err)
		}
		moduleRef := "lib.core." + name
		if !strings.Contains(string(raw), moduleRef) {
			return fmt.Errorf("%s: stable module example %q does not reference %q", path, example, moduleRef)
		}
	}
	return nil
}

func stableModuleExamplePath(moduleName string) string {
	switch moduleName {
	case "capability":
		return filepath.FromSlash("examples/core_memory_smoke.tetra")
	default:
		return filepath.FromSlash(fmt.Sprintf("examples/core_%s_smoke.tetra", moduleName))
	}
}

func stableModuleExamplePaths(modulePaths []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, path := range modulePaths {
		name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		example := stableModuleExamplePath(name)
		if example == "" {
			continue
		}
		if _, ok := seen[example]; ok {
			continue
		}
		seen[example] = struct{}{}
		out = append(out, example)
	}
	sort.Strings(out)
	return out
}

func verifyStdlibModulePaths(stablePaths []string, experimentalPaths []string) error {
	for _, path := range stablePaths {
		if err := verifyStdlibModulePath(path, "core", "lib.core.", true); err != nil {
			return err
		}
	}
	for _, path := range experimentalPaths {
		if err := verifyStdlibModulePath(path, "experimental", "lib.experimental.", false); err != nil {
			return err
		}
	}
	return nil
}

func verifyStdlibModulePath(path string, wantDir string, wantPrefix string, stable bool) error {
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if !validStdlibLeafName(name) {
		return fmt.Errorf("%s: invalid stdlib module leaf name %q", path, name)
	}
	if stable && hasStableVersionSuffix(name) {
		return fmt.Errorf("%s: stable module name must not contain version suffix: %q", path, name)
	}
	if filepath.Base(filepath.Dir(path)) != wantDir {
		return fmt.Errorf("%s: expected path under lib/%s", path, wantDir)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("%s: %v", path, err)
	}
	file, err := compiler.ParseFile(raw, path)
	if err != nil {
		return fmt.Errorf("%s: %v", path, err)
	}
	want := wantPrefix + name
	if file.Module != want {
		return fmt.Errorf("%s: expected module %s, got %s", path, want, file.Module)
	}
	return nil
}

func validStdlibLeafName(name string) bool {
	if name == "" || name[0] < 'a' || name[0] > 'z' {
		return false
	}
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			continue
		}
		return false
	}
	return true
}

func hasStableVersionSuffix(name string) bool {
	for _, part := range strings.Split(name, "_") {
		if len(part) < 2 || part[0] != 'v' {
			continue
		}
		allDigits := true
		for _, r := range part[1:] {
			if r < '0' || r > '9' {
				allDigits = false
				break
			}
		}
		if allDigits {
			return true
		}
	}
	return false
}

func verifyStableExamplesDoNotImportExperimental(paths []string) error {
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("%s: %v", path, err)
		}
		file, err := compiler.ParseFile(raw, path)
		if err != nil {
			return fmt.Errorf("%s: %v", path, err)
		}
		for _, imp := range file.Imports {
			if imp.Path == "lib.experimental" || strings.HasPrefix(imp.Path, "lib.experimental.") {
				return fmt.Errorf("%s: stable example imports experimental module %q", path, imp.Path)
			}
		}
	}
	return nil
}

func verifyStableModuleEffectsMetadata(paths []string) error {
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("%s: %v", path, err)
		}
		lines := strings.Split(string(raw), "\n")
		var metadata string
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "module ") {
				break
			}
			if strings.HasPrefix(trimmed, "// Effects:") {
				metadata = strings.TrimSpace(strings.TrimPrefix(trimmed, "// Effects:"))
			}
		}
		if metadata == "" {
			return fmt.Errorf("%s: missing effects metadata", path)
		}
		metadataEffects, err := parseStableEffectsMetadata(path, metadata)
		if err != nil {
			return err
		}
		declaredEffects, err := stableModuleDeclaredEffects(path, raw)
		if err != nil {
			return err
		}
		if !sameEffectSet(metadataEffects, declaredEffects) {
			return fmt.Errorf("%s: effects metadata mismatch: got %s want %s", path, formatEffectSet(metadataEffects), formatEffectSet(declaredEffects))
		}
	}
	return nil
}

func verifyExperimentalModuleMirrors(paths []string) error {
	for _, path := range paths {
		name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		raw, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("%s: %v", path, err)
		}
		text := string(raw)
		if !strings.Contains(text, "Experimental") || !strings.Contains(text, "no stability guarantees") {
			return fmt.Errorf("%s: missing experimental stability disclaimer", path)
		}
		stableModule := "lib.core." + name
		if !strings.Contains(text, "Promotion note:") || !strings.Contains(text, stableModule) {
			return fmt.Errorf("%s: missing promotion note for %s", path, stableModule)
		}
		file, err := compiler.ParseFile(raw, path)
		if err != nil {
			return fmt.Errorf("%s: %v", path, err)
		}
		foundStableImport := false
		for _, imp := range file.Imports {
			if imp.Path == stableModule {
				foundStableImport = true
				break
			}
		}
		if !foundStableImport {
			return fmt.Errorf("%s: experimental mirror must import %s", path, stableModule)
		}
	}
	return nil
}

func verifyStdlibGuide(path string, stablePaths []string, experimentalPaths []string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("%s: %v", path, err)
	}
	text := string(raw)
	stableRows := parseStdlibGuideStableRows(text)
	var errs []string
	for _, modulePath := range stablePaths {
		name := strings.TrimSuffix(filepath.Base(modulePath), filepath.Ext(modulePath))
		moduleImport := "import lib.core." + name + " as "
		row, ok := stableRows[moduleImport]
		if !ok {
			errs = append(errs, fmt.Sprintf("missing stable guide row for lib.core.%s", name))
			continue
		}
		expectedExample := stableModuleExamplePath(name)
		if !strings.Contains(row.example, expectedExample) {
			errs = append(errs, fmt.Sprintf("lib.core.%s example mismatch: got %q want %q", name, row.example, expectedExample))
		}
		moduleRaw, err := os.ReadFile(modulePath)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", modulePath, err))
			continue
		}
		declaredEffects, err := stableModuleDeclaredEffects(modulePath, moduleRaw)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		guideEffects, err := parseGuideEffectSet(path, row.effects)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		if !sameEffectSet(guideEffects, declaredEffects) {
			errs = append(errs, fmt.Sprintf("lib.core.%s effects mismatch: got %s want %s", name, formatEffectSet(guideEffects), formatEffectSet(declaredEffects)))
		}
	}
	if len(experimentalPaths) > 0 && !strings.Contains(text, "## Experimental Mirrors") {
		errs = append(errs, "missing experimental mirrors section")
	}
	for _, modulePath := range experimentalPaths {
		name := strings.TrimSuffix(filepath.Base(modulePath), filepath.Ext(modulePath))
		experimentalImport := "import lib.experimental." + name + " as "
		stableImport := "import lib.core." + name + " as "
		if !strings.Contains(text, experimentalImport) {
			errs = append(errs, fmt.Sprintf("missing experimental guide row for lib.experimental.%s", name))
		}
		if !strings.Contains(text, stableImport) {
			errs = append(errs, fmt.Sprintf("missing stable replacement for lib.experimental.%s", name))
		}
	}
	if len(experimentalPaths) > 0 && (!strings.Contains(text, "Experimental mirror") || !strings.Contains(text, "no stability guarantees")) {
		errs = append(errs, "experimental mirrors section must state Experimental mirror and no stability guarantees")
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s: %s", path, strings.Join(errs, "; "))
	}
	return nil
}

type stdlibGuideStableRow struct {
	example string
	effects string
}

func parseStdlibGuideStableRows(text string) map[string]stdlibGuideStableRow {
	rows := map[string]stdlibGuideStableRow{}
	inStableTable := false
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") && !strings.HasPrefix(trimmed, "## Stable Module Choices") {
			inStableTable = false
		}
		if trimmed == "| Need | Import | Example | Effects |" {
			inStableTable = true
			continue
		}
		if !inStableTable || !strings.HasPrefix(trimmed, "|") || strings.Contains(trimmed, "---") {
			continue
		}
		cells := splitMarkdownTableRow(trimmed)
		if len(cells) != 4 {
			continue
		}
		importCell := strings.ReplaceAll(cells[1], "`", "")
		importStart := strings.Index(importCell, "import lib.core.")
		if importStart == -1 {
			continue
		}
		importText := importCell[importStart:]
		asIndex := strings.Index(importText, " as ")
		if asIndex == -1 {
			continue
		}
		importKey := importText[:asIndex+4]
		rows[importKey] = stdlibGuideStableRow{
			example: cells[2],
			effects: cells[3],
		}
	}
	return rows
}

func splitMarkdownTableRow(row string) []string {
	trimmed := strings.Trim(row, "|")
	parts := strings.Split(trimmed, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func parseGuideEffectSet(path string, raw string) ([]string, error) {
	normalized := strings.TrimSpace(strings.ReplaceAll(raw, "`", ""))
	if strings.EqualFold(normalized, "none") {
		return nil, nil
	}
	effects := map[string]struct{}{}
	for _, part := range strings.Split(normalized, ",") {
		effect := strings.TrimSpace(part)
		if effect == "" {
			return nil, fmt.Errorf("%s: invalid guide effects %q", path, raw)
		}
		expanded, err := expandStableEffect(effect)
		if err != nil {
			return nil, fmt.Errorf("%s: %v", path, err)
		}
		for _, name := range expanded {
			effects[name] = struct{}{}
		}
	}
	return sortedEffectNames(effects), nil
}

func parseStableEffectsMetadata(path string, metadata string) ([]string, error) {
	if strings.EqualFold(metadata, "none") {
		return nil, nil
	}
	effects := map[string]struct{}{}
	for _, rawEffect := range strings.Split(metadata, ",") {
		effect := strings.TrimSpace(rawEffect)
		if effect == "" {
			return nil, fmt.Errorf("%s: invalid effects metadata %q", path, metadata)
		}
		expanded, err := expandStableEffect(effect)
		if err != nil {
			return nil, fmt.Errorf("%s: %v", path, err)
		}
		for _, name := range expanded {
			effects[name] = struct{}{}
		}
	}
	return sortedEffectNames(effects), nil
}

func stableModuleDeclaredEffects(path string, raw []byte) ([]string, error) {
	file, err := compiler.ParseFile(raw, path)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", path, err)
	}
	effects := map[string]struct{}{}
	addUses := func(uses []string) error {
		for _, rawEffect := range uses {
			expanded, err := expandStableEffect(rawEffect)
			if err != nil {
				return err
			}
			for _, name := range expanded {
				effects[name] = struct{}{}
			}
		}
		return nil
	}
	for _, fn := range file.Funcs {
		if err := addUses(fn.Uses); err != nil {
			return nil, fmt.Errorf("%s: %v", path, err)
		}
	}
	for _, proto := range file.Protocols {
		for _, req := range proto.Requirements {
			if err := addUses(req.Uses); err != nil {
				return nil, fmt.Errorf("%s: %v", path, err)
			}
		}
	}
	for _, ext := range file.Extensions {
		for _, method := range ext.Methods {
			if err := addUses(method.Uses); err != nil {
				return nil, fmt.Errorf("%s: %v", path, err)
			}
		}
	}
	return sortedEffectNames(effects), nil
}

func expandStableEffect(effect string) ([]string, error) {
	canonical := map[string]string{
		"actors":      "actors",
		"alloc":       "alloc",
		"budget":      "budget",
		"cap.io":      "io",
		"cap.mem":     "mem",
		"capability":  "capability",
		"capsule.io":  "capsule.io",
		"capsule.mem": "capsule.mem",
		"control":     "control",
		"io":          "io",
		"islands":     "islands",
		"link":        "link",
		"mem":         "mem",
		"mmio":        "mmio",
		"privacy":     "privacy",
		"runtime":     "runtime",
	}
	if name, ok := canonical[effect]; ok {
		return []string{name}, nil
	}
	groups := map[string][]string{
		"effects.all":     {"actors", "alloc", "budget", "capability", "control", "io", "islands", "link", "mem", "mmio", "privacy", "runtime"},
		"effects.cap.io":  {"capability", "io", "mmio"},
		"effects.cap.mem": {"capability", "mem"},
		"effects.memory":  {"alloc", "islands", "mem"},
		"effects.policy":  {"budget", "privacy"},
		"effects.runtime": {"actors", "control", "link", "runtime"},
	}
	if members, ok := groups[effect]; ok {
		return members, nil
	}
	for _, r := range effect {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '_' && r != '.' {
			return nil, fmt.Errorf("invalid effect name %q in metadata", effect)
		}
	}
	return nil, fmt.Errorf("unknown stable effect %q in metadata", effect)
}

func sortedEffectNames(effects map[string]struct{}) []string {
	out := make([]string, 0, len(effects))
	for effect := range effects {
		out = append(out, effect)
	}
	sort.Strings(out)
	return out
}

func formatEffectSet(effects []string) string {
	if len(effects) == 0 {
		return "none"
	}
	return strings.Join(effects, ", ")
}

func sameEffectSet(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func currentStableModulePaths() []string {
	paths, err := filepath.Glob(filepath.FromSlash("lib/core/*.tetra"))
	if err != nil {
		return nil
	}
	sort.Strings(paths)
	return paths
}

func currentExperimentalModulePaths() []string {
	paths, err := filepath.Glob(filepath.FromSlash("lib/experimental/*.tetra"))
	if err != nil {
		return nil
	}
	sort.Strings(paths)
	return paths
}

func verifyEpic14ExampleIndex(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("%s: %v", path, err)
	}
	text := string(raw)

	requiredExamples := []string{
		"examples/hello.tetra",
		"examples/flow_hello.tetra",
		"examples/bool_smoke.tetra",
		"examples/for_range_smoke.tetra",
		"examples/for_collection_smoke.tetra",
		"examples/loop_control_smoke.tetra",
		"examples/const_smoke.tetra",
		"examples/const_bool_smoke.tetra",
		"examples/local_const_smoke.tetra",
		"examples/compound_assignment_smoke.tetra",
		"examples/enum_match_smoke.tetra",
		"examples/enum_exhaustive_match_smoke.tetra",
		"examples/optional_smoke.tetra",
		"examples/optional_match_smoke.tetra",
		"examples/typed_errors_smoke.tetra",
		"examples/generic_smoke.tetra",
		"examples/generic_struct_smoke.tetra",
		"examples/protocol_impl_smoke.tetra",
		"examples/extension_smoke.tetra",
		"examples/ownership_smoke.tetra",
		"examples/async_smoke.tetra",
		"examples/task_smoke.tetra",
		"examples/actors_pingpong.tetra",
		"examples/islands_hello.tetra",
		"examples/islands_i32.tetra",
		"examples/islands_overflow.tetra",
		"examples/cap_mem_smoke.tetra",
		"examples/mmio_smoke.tetra",
		"examples/memset_smoke.tetra",
		"examples/ui_web_smoke.tetra",
		"examples/ui_native_shell_smoke.tetra",
		"examples/projects/hello_t4/src/main.t4",
		"examples/projects/dogfood_wasi/src/main.tetra",
		"examples/projects/dogfood_web_ui/src/main.tetra",
		"examples/projects/dogfood_cli/src/main.tetra",
		"examples/projects/dogfood_actor_task/src/main.tetra",
		"examples/projects/eco_dogfood/src/main.tetra",
	}

	requiredHeadings := []string{
		"### Basic language examples (`V020-0701..0705`)",
		"### Control-flow examples (`V020-0706..0710`)",
		"### Const and assignment examples (`V020-0711..0715`)",
		"### Enum/match examples (`V020-0716..0720`)",
		"### Optional/error examples (`V020-0721..0725`)",
		"### Generic/protocol/extension examples (`V020-0726..0730`)",
		"### Safety/runtime examples (`V020-0731..0735`)",
		"### Memory/capability examples (`V020-0736..0740`)",
		"### UI/WASM examples (`V020-0741..0745`)",
		"### Project dogfood examples (`V020-0746..0750`)",
	}

	var missing []string
	for _, example := range requiredExamples {
		if !strings.Contains(text, "`"+example+"`") {
			missing = append(missing, "example entry "+example)
		}
	}
	for _, heading := range requiredHeadings {
		if !strings.Contains(text, heading) {
			missing = append(missing, "heading "+heading)
		}
	}
	if !strings.Contains(text, "## Epic 14 Verification Commands") && !strings.Contains(text, "## Epic 15 Verification Commands") {
		missing = append(missing, "heading ## Epic 14 Verification Commands or ## Epic 15 Verification Commands")
	}
	if !strings.Contains(text, "## Troubleshooting Notes (Epic 14)") && !strings.Contains(text, "## Troubleshooting Notes (Epic 15)") {
		missing = append(missing, "heading ## Troubleshooting Notes (Epic 14) or ## Troubleshooting Notes (Epic 15)")
	}
	if !strings.Contains(strings.ToLower(text), "unsupported") {
		missing = append(missing, "troubleshooting keyword unsupported")
	}
	if !strings.Contains(strings.ToLower(text), "regression") {
		missing = append(missing, "troubleshooting keyword regression")
	}

	if len(missing) > 0 {
		return fmt.Errorf("%s: missing Epic 14 index coverage: %s", path, strings.Join(missing, ", "))
	}
	return nil
}
