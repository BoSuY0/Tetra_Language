package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tetra_language/compiler"
)

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
		"surface":     "surface",
	}
	if name, ok := canonical[effect]; ok {
		return []string{name}, nil
	}
	groups := map[string][]string{
		"effects.all":     {"actors", "alloc", "budget", "capability", "control", "io", "islands", "link", "mem", "mmio", "privacy", "runtime", "surface"},
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
