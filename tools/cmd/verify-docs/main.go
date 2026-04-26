package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
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

	if err := verifyDoctestBlocks([]string{"README.md", "docs/spec/flow_syntax_mvp.md"}); err != nil {
		errs = append(errs, err.Error())
	}
	if err := verifyRequiredDoctestBlocks(currentStableModulePaths()); err != nil {
		errs = append(errs, err.Error())
	}
	if err := verifyWASMBackendPlan("docs/backend/wasm_backend_plan.md", ctarget.PlannedTriples()); err != nil {
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

func currentStableModulePaths() []string {
	return []string{
		filepath.FromSlash("lib/core/capability.tetra"),
		filepath.FromSlash("lib/core/math.tetra"),
		filepath.FromSlash("lib/core/memory.tetra"),
	}
}
