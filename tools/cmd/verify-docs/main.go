package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
		SurfaceRequiredSymbols   []string `json:"surface_required_symbols"`
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
	runtimeSymbols = append(runtimeSymbols, m.RuntimeABI.SurfaceRequiredSymbols...)
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
	if err := verifySurfaceReleaseDocs(surfaceReleaseDocPaths()); err != nil {
		errs = append(errs, err.Error())
	}
	if err := verifyMemoryIslandsSurfaceReleaseDocs(memoryIslandsSurfaceReleaseDocPaths()); err != nil {
		errs = append(errs, err.Error())
	}
	if err := verifyMemoryIslandsFinalProductionReadinessAudit(memoryIslandsFinalProductionReadinessAuditDocPaths()); err != nil {
		errs = append(errs, err.Error())
	}
	if err := verifyMemoryIslandsFinalActorBenchmarkHandoff(memoryIslandsFinalActorBenchmarkHandoffDocPaths()); err != nil {
		errs = append(errs, err.Error())
	}
	if err := verifyActorRuntimeFoundationDocs(defaultActorRuntimeFoundationDocPaths(), m.Features); err != nil {
		errs = append(errs, err.Error())
	}
	if err := verifyFinalMemoryIslandsSurfaceProductionAudit(finalMemoryIslandsSurfaceProductionAuditDocPaths()); err != nil {
		errs = append(errs, err.Error())
	}
	if err := verifyFeatureRegistry(m.Features); err != nil {
		errs = append(errs, err.Error())
	}
	if err := verifyWASMBackendPlan("docs/backend/wasm_backend_plan.md", ctarget.WASMTriples()); err != nil {
		errs = append(errs, err.Error())
	}
	if err := verifyNetworkingRuntimeBoundaryDocs(defaultNetworkingRuntimeBoundaryDocPaths()); err != nil {
		errs = append(errs, err.Error())
	}
	if err := verifyMemoryProductionContractDocs(defaultMemoryProductionContractDocPaths()); err != nil {
		errs = append(errs, err.Error())
	}
	if err := verifyRAMContractCompilerDocs(defaultRAMContractCompilerDocPaths(), m.Features); err != nil {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintln(os.Stderr, "verify-docs:", e)
		}
		os.Exit(1)
	}
}
