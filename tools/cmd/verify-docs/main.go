package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
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
		ActorsProgramGlueSymbols []string `json:"actors_program_glue_symbols"`
	} `json:"runtime_abi"`
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
	checkContains("docs/spec/runtime_abi.md", append(append([]string(nil), m.RuntimeABI.ActorsRequiredSymbols...), m.RuntimeABI.ActorsProgramGlueSymbols...))

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
	if err := verifyRequiredDoctestBlocks(stableModulePaths); err != nil {
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

func verifyDoctestBlocks(paths []string) error {
	return verifyDoctestBlocksWithPolicy(paths, false)
}

func verifyRequiredDoctestBlocks(paths []string) error {
	return verifyDoctestBlocksWithPolicy(paths, true)
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
