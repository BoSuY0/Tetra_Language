package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyDoctestBlocks(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "doc.md")
	if err := os.WriteFile(doc, []byte("```tetra doctest\nfunc main() -> Int:\n    return 0\n```\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := verifyDoctestBlocks([]string{doc}); err != nil {
		t.Fatalf("verifyDoctestBlocks: %v", err)
	}
}

func TestVerifyDoctestBlocksRejectsUnterminatedBlock(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "doc.md")
	if err := os.WriteFile(doc, []byte("text\n```tetra doctest\nfunc main() -> Int:\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := verifyDoctestBlocks([]string{doc})
	if err == nil {
		t.Fatalf("expected unterminated doctest failure")
	}
	if !strings.Contains(err.Error(), "unterminated tetra doctest block starting at line 2") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifySpecCodeBlocksChecksTetraAndT4Blocks(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "spec.md")
	body := strings.Join([]string{
		"# Spec",
		"",
		"```tetra check",
		"func main() -> Int:",
		"    return 0",
		"```",
		"",
		"```t4",
		"func helper() -> Int:",
		"    return 1",
		"```",
	}, "\n")
	if err := os.WriteFile(doc, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := verifySpecCodeBlocks([]string{doc}); err != nil {
		t.Fatalf("verifySpecCodeBlocks: %v", err)
	}
}

func TestVerifySpecCodeBlocksSkipsExplicitNonExecutableExamples(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "spec.md")
	body := strings.Join([]string{
		"# Spec",
		"",
		"```tetra pseudocode",
		"func broken(",
		"```",
		"",
		"```tetra negative",
		"func broken(",
		"```",
		"",
		"```t4 unsupported",
		"func broken(",
		"```",
	}, "\n")
	if err := os.WriteFile(doc, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := verifySpecCodeBlocks([]string{doc}); err != nil {
		t.Fatalf("verifySpecCodeBlocks: %v", err)
	}
}

func TestVerifySpecCodeBlocksRejectsParseDrift(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "spec.md")
	body := strings.Join([]string{
		"# Spec",
		"",
		"```tetra",
		"func broken(",
		"```",
	}, "\n")
	if err := os.WriteFile(doc, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	err := verifySpecCodeBlocks([]string{doc})
	if err == nil {
		t.Fatalf("expected parse drift failure")
	}
	if !strings.Contains(err.Error(), "spec block 1 parse") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifySpecCodeBlocksRejectsCheckDrift(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "spec.md")
	body := strings.Join([]string{
		"# Spec",
		"",
		"```tetra check",
		"func main() -> Int:",
		"    return missing_symbol",
		"```",
	}, "\n")
	if err := os.WriteFile(doc, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	err := verifySpecCodeBlocks([]string{doc})
	if err == nil {
		t.Fatalf("expected check drift failure")
	}
	if !strings.Contains(err.Error(), "spec block 1 check") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifySpecCodeBlocksRejectsUnterminatedBlock(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "spec.md")
	if err := os.WriteFile(doc, []byte("```tetra\nfunc main() -> Int:\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := verifySpecCodeBlocks([]string{doc})
	if err == nil {
		t.Fatalf("expected unterminated spec block failure")
	}
	if !strings.Contains(err.Error(), "unterminated tetra spec block starting at line 1") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifyWASMBackendPlanRequiresConcreteGateCommands(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "wasm_backend_plan.md")
	body := strings.Join([]string{
		"# WASM Backend Plan",
		"",
		"Status: current",
		"",
		"## Targets",
		"",
		"- `wasm32-wasi`",
		"- `wasm32-web`",
		"",
		"## Phases",
		"",
		"### Phase 0: Target contract",
		"### Phase 1: WASM IR emitter",
		"### Phase 2: WASI runner",
		"### Phase 3: Web runtime",
		"### Phase 4: v1.0 release gate",
		"",
		"## Gate Commands",
		"",
		"- `go run ./tools/cmd/validate-targets`",
		"- `./tetra smoke --target wasm32-wasi --run=false`",
		"- `bash scripts/release/v1_0/gate.sh`",
		"- wasmtime",
		"- browser automation",
	}, "\n")
	if err := os.WriteFile(doc, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	err := verifyWASMBackendPlan(doc, []string{"wasm32-wasi", "wasm32-web"})
	if err == nil {
		t.Fatalf("expected missing wasm32-web gate command failure")
	}
	if !strings.Contains(err.Error(), "./tetra smoke --target wasm32-web --run=false") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifyMemoryProductionContractDocsRejectsIncompleteContract(t *testing.T) {
	dir := t.TempDir()
	paths := memoryProductionContractDocPaths{
		RuntimeABI:             filepath.Join(dir, "runtime_abi.md"),
		Ownership:              filepath.Join(dir, "ownership_v1.md"),
		Unsafe:                 filepath.Join(dir, "unsafe.md"),
		Capabilities:           filepath.Join(dir, "capabilities.md"),
		Stdlib:                 filepath.Join(dir, "stdlib.md"),
		StdlibGuide:            filepath.Join(dir, "standard_library_guide.md"),
		CoreMemory:             filepath.Join(dir, "memory.tetra"),
		TargetCapabilityMatrix: filepath.Join(dir, "memory-target-capability-matrix.md"),
		MemoryCostModel:        filepath.Join(dir, "memory_cost_model.md"),
		MemoryFuzzOracle:       filepath.Join(dir, "memory-fuzz-oracle-v1.md"),
		MemoryProductionFinal:  filepath.Join(dir, "memory-production-core-v1-final.md"),
		MemoryProductionMap:    filepath.Join(dir, "memory-production-core-v1-artifact-map.md"),
		MemoryProductionClaims: filepath.Join(dir, "memory-production-core-v1-nonclaims.md"),
	}
	for _, path := range []string{paths.RuntimeABI, paths.Ownership, paths.Unsafe, paths.Capabilities, paths.Stdlib, paths.StdlibGuide, paths.CoreMemory, paths.TargetCapabilityMatrix, paths.MemoryCostModel, paths.MemoryFuzzOracle, paths.MemoryProductionFinal, paths.MemoryProductionMap, paths.MemoryProductionClaims} {
		if err := os.WriteFile(path, []byte("memory docs\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	err := verifyMemoryProductionContractDocs(paths)
	if err == nil {
		t.Fatalf("expected incomplete memory production contract failure")
	}
	for _, want := range []string{"runtime_abi.md", "Linux-x64 Memory Production ABI", "runtime bounds diagnostics"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in error, got %v", want, err)
		}
	}
}

func TestVerifyMemoryProductionContractDocsAcceptsRequiredContract(t *testing.T) {
	dir := t.TempDir()
	paths := memoryProductionContractDocPaths{
		RuntimeABI:             filepath.Join(dir, "runtime_abi.md"),
		Ownership:              filepath.Join(dir, "ownership_v1.md"),
		Unsafe:                 filepath.Join(dir, "unsafe.md"),
		Capabilities:           filepath.Join(dir, "capabilities.md"),
		Stdlib:                 filepath.Join(dir, "stdlib.md"),
		StdlibGuide:            filepath.Join(dir, "standard_library_guide.md"),
		CoreMemory:             filepath.Join(dir, "memory.tetra"),
		TargetCapabilityMatrix: filepath.Join(dir, "memory-target-capability-matrix.md"),
		MemoryCostModel:        filepath.Join(dir, "memory_cost_model.md"),
		MemoryFuzzOracle:       filepath.Join(dir, "memory-fuzz-oracle-v1.md"),
		MemoryProductionFinal:  filepath.Join(dir, "memory-production-core-v1-final.md"),
		MemoryProductionMap:    filepath.Join(dir, "memory-production-core-v1-artifact-map.md"),
		MemoryProductionClaims: filepath.Join(dir, "memory-production-core-v1-nonclaims.md"),
	}
	for _, requirement := range memoryProductionContractRequirements(paths) {
		body := strings.Join(requirement.Required, "\n")
		if err := os.WriteFile(requirement.Path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if err := verifyMemoryProductionContractDocs(paths); err != nil {
		t.Fatalf("verifyMemoryProductionContractDocs: %v", err)
	}
}

func TestVerifyMemoryProductionContractDocsRequiresTargetCapabilityMatrix(t *testing.T) {
	dir := t.TempDir()
	paths := memoryProductionContractDocPaths{
		RuntimeABI:             filepath.Join(dir, "runtime_abi.md"),
		Ownership:              filepath.Join(dir, "ownership_v1.md"),
		Unsafe:                 filepath.Join(dir, "unsafe.md"),
		Capabilities:           filepath.Join(dir, "capabilities.md"),
		Stdlib:                 filepath.Join(dir, "stdlib.md"),
		StdlibGuide:            filepath.Join(dir, "standard_library_guide.md"),
		CoreMemory:             filepath.Join(dir, "memory.tetra"),
		TargetCapabilityMatrix: filepath.Join(dir, "memory-target-capability-matrix.md"),
		MemoryCostModel:        filepath.Join(dir, "memory_cost_model.md"),
		MemoryFuzzOracle:       filepath.Join(dir, "memory-fuzz-oracle-v1.md"),
		MemoryProductionFinal:  filepath.Join(dir, "memory-production-core-v1-final.md"),
		MemoryProductionMap:    filepath.Join(dir, "memory-production-core-v1-artifact-map.md"),
		MemoryProductionClaims: filepath.Join(dir, "memory-production-core-v1-nonclaims.md"),
	}
	for _, requirement := range memoryProductionContractRequirements(paths) {
		if requirement.Path == paths.TargetCapabilityMatrix {
			continue
		}
		body := strings.Join(requirement.Required, "\n")
		if err := os.WriteFile(requirement.Path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	err := verifyMemoryProductionContractDocs(paths)
	if err == nil || !strings.Contains(err.Error(), "memory-target-capability-matrix.md") {
		t.Fatalf("expected missing target capability matrix failure, got %v", err)
	}
}

func TestVerifyMemoryProductionContractDocsRequiresMemoryCostModel(t *testing.T) {
	dir := t.TempDir()
	paths := memoryProductionContractDocPaths{
		RuntimeABI:             filepath.Join(dir, "runtime_abi.md"),
		Ownership:              filepath.Join(dir, "ownership_v1.md"),
		Unsafe:                 filepath.Join(dir, "unsafe.md"),
		Capabilities:           filepath.Join(dir, "capabilities.md"),
		Stdlib:                 filepath.Join(dir, "stdlib.md"),
		StdlibGuide:            filepath.Join(dir, "standard_library_guide.md"),
		CoreMemory:             filepath.Join(dir, "memory.tetra"),
		TargetCapabilityMatrix: filepath.Join(dir, "memory-target-capability-matrix.md"),
		MemoryCostModel:        filepath.Join(dir, "memory_cost_model.md"),
		MemoryFuzzOracle:       filepath.Join(dir, "memory-fuzz-oracle-v1.md"),
		MemoryProductionFinal:  filepath.Join(dir, "memory-production-core-v1-final.md"),
		MemoryProductionMap:    filepath.Join(dir, "memory-production-core-v1-artifact-map.md"),
		MemoryProductionClaims: filepath.Join(dir, "memory-production-core-v1-nonclaims.md"),
	}
	for _, requirement := range memoryProductionContractRequirements(paths) {
		if requirement.Path == paths.MemoryCostModel {
			continue
		}
		body := strings.Join(requirement.Required, "\n")
		if err := os.WriteFile(requirement.Path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	err := verifyMemoryProductionContractDocs(paths)
	if err == nil || !strings.Contains(err.Error(), "memory_cost_model.md") {
		t.Fatalf("expected missing memory cost model failure, got %v", err)
	}
}

func TestVerifyMemoryProductionContractDocsRejectsFastestBenchmarkClaim(t *testing.T) {
	dir := t.TempDir()
	paths := memoryProductionContractDocPaths{
		RuntimeABI:             filepath.Join(dir, "runtime_abi.md"),
		Ownership:              filepath.Join(dir, "ownership_v1.md"),
		Unsafe:                 filepath.Join(dir, "unsafe.md"),
		Capabilities:           filepath.Join(dir, "capabilities.md"),
		Stdlib:                 filepath.Join(dir, "stdlib.md"),
		StdlibGuide:            filepath.Join(dir, "standard_library_guide.md"),
		CoreMemory:             filepath.Join(dir, "memory.tetra"),
		TargetCapabilityMatrix: filepath.Join(dir, "memory-target-capability-matrix.md"),
		MemoryCostModel:        filepath.Join(dir, "memory_cost_model.md"),
		MemoryFuzzOracle:       filepath.Join(dir, "memory-fuzz-oracle-v1.md"),
		MemoryProductionFinal:  filepath.Join(dir, "memory-production-core-v1-final.md"),
		MemoryProductionMap:    filepath.Join(dir, "memory-production-core-v1-artifact-map.md"),
		MemoryProductionClaims: filepath.Join(dir, "memory-production-core-v1-nonclaims.md"),
	}
	for _, requirement := range memoryProductionContractRequirements(paths) {
		body := strings.Join(requirement.Required, "\n")
		if requirement.Path == paths.MemoryCostModel {
			body += "\nTetra is the fastest language and this is an official benchmark result.\n"
		}
		if err := os.WriteFile(requirement.Path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	err := verifyMemoryProductionContractDocs(paths)
	if err == nil || !strings.Contains(err.Error(), "fastest language") || !strings.Contains(err.Error(), "official benchmark") {
		t.Fatalf("expected fastest/official benchmark docs rejection, got %v", err)
	}
}

func TestForbiddenPublicPerformanceClaimsAllowsWrappedNonClaimSentence(t *testing.T) {
	cases := []string{
		strings.Join([]string{
			"The v1 generic collection surface does not claim a production",
			"allocator-backed vector/map runtime, generic hashing/equality",
			"protocols, resizing, collision handling, C++/Rust performance",
			"parity, and makes no official benchmark result claim.",
		}, "\n"),
		"This does not promote a full source-level PostgreSQL driver API or measured speed comparison. It makes no official result claim for TechEmpower and no P20 performance matrix claim.",
		"It does not promote a production HTTP server or source-level cached-date API. It makes no official result claim for TechEmpower and no P20 performance matrix claim.",
		"The final audit records no official benchmark result and no target parity evidence from quick output.",
		"It is not a runtime measurement, C++/Rust parity claim, and makes no official benchmark result claim.",
		"It is not a full source-level PostgreSQL driver API or external production database deployment. It makes no official result claim for TechEmpower, no production database benchmark claim, and no measured speed comparison claim.",
		"`MEM-FUZZ-012` makes no arbitrary unsafe safety claim, no full runtime/ABI/target parity proof, and no Memory 100% claim.",
		`Memory evidence includes MemoryFactGraph evidence; no broad "Memory 100%" claim.`,
		`<li>no broad memory-safety or <strong>"Memory 100%"</strong> claim;</li>`,
	}

	for _, text := range cases {
		if claims := forbiddenPublicPerformanceClaims(text); len(claims) != 0 {
			t.Fatalf("forbiddenPublicPerformanceClaims(%q) = %#v, want no claims", text, claims)
		}
	}
}

func TestForbiddenPublicPerformanceClaimsRejectsClaimAfterUnrelatedDoesNot(t *testing.T) {
	text := strings.Join([]string{
		"normal build does not run heavy validators at runtime",
		"Tetra is the fastest language and this is an official benchmark result.",
	}, "\n")

	claims := forbiddenPublicPerformanceClaims(text)
	if len(claims) == 0 || !strings.Contains(strings.Join(claims, ","), "fastest language") || !strings.Contains(strings.Join(claims, ","), "official benchmark") {
		t.Fatalf("forbiddenPublicPerformanceClaims() = %#v, want fastest/official rejection", claims)
	}
}

func TestForbiddenPublicPerformanceClaimsRejectsIslandKernelAndMemoryOverclaims(t *testing.T) {
	text := strings.Join([]string{
		"IslandKernel complete for production memory.",
		"Tetra Memory 100% is now guaranteed.",
		"The language is leak-free for all host tooling.",
	}, "\n")

	claims := strings.Join(forbiddenPublicPerformanceClaims(text), ",")
	for _, want := range []string{"islandkernel complete", "memory 100%", "leak-free"} {
		if !strings.Contains(claims, want) {
			t.Fatalf("forbiddenPublicPerformanceClaims() = %q, missing %q", claims, want)
		}
	}
}

func TestForbiddenPublicPerformanceClaimsAllowsIslandKernelNonClaims(t *testing.T) {
	text := strings.Join([]string{
		"IslandKernel is not complete and remains model-only until validate-island-proof evidence exists.",
		"This does not claim Memory 100%, leak-free host tooling, or arbitrary unsafe pointer safety.",
	}, "\n")

	if claims := forbiddenPublicPerformanceClaims(text); len(claims) != 0 {
		t.Fatalf("forbiddenPublicPerformanceClaims() = %#v, want no claims", claims)
	}
}

func TestVerifyMemoryProductionContractDocsRequiresMemoryFuzzOracle(t *testing.T) {
	dir := t.TempDir()
	paths := memoryProductionContractDocPaths{
		RuntimeABI:             filepath.Join(dir, "runtime_abi.md"),
		Ownership:              filepath.Join(dir, "ownership_v1.md"),
		Unsafe:                 filepath.Join(dir, "unsafe.md"),
		Capabilities:           filepath.Join(dir, "capabilities.md"),
		Stdlib:                 filepath.Join(dir, "stdlib.md"),
		StdlibGuide:            filepath.Join(dir, "standard_library_guide.md"),
		CoreMemory:             filepath.Join(dir, "memory.tetra"),
		TargetCapabilityMatrix: filepath.Join(dir, "memory-target-capability-matrix.md"),
		MemoryCostModel:        filepath.Join(dir, "memory_cost_model.md"),
		MemoryFuzzOracle:       filepath.Join(dir, "memory-fuzz-oracle-v1.md"),
		MemoryProductionFinal:  filepath.Join(dir, "memory-production-core-v1-final.md"),
		MemoryProductionMap:    filepath.Join(dir, "memory-production-core-v1-artifact-map.md"),
		MemoryProductionClaims: filepath.Join(dir, "memory-production-core-v1-nonclaims.md"),
	}
	for _, requirement := range memoryProductionContractRequirements(paths) {
		if requirement.Path == paths.MemoryFuzzOracle {
			continue
		}
		body := strings.Join(requirement.Required, "\n")
		if err := os.WriteFile(requirement.Path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	err := verifyMemoryProductionContractDocs(paths)
	if err == nil || !strings.Contains(err.Error(), "memory-fuzz-oracle-v1.md") {
		t.Fatalf("expected missing memory fuzz oracle failure, got %v", err)
	}
}

func TestVerifyMemoryProductionContractDocsRequiresFinalAuditDocs(t *testing.T) {
	dir := t.TempDir()
	paths := memoryProductionContractDocPaths{
		RuntimeABI:             filepath.Join(dir, "runtime_abi.md"),
		Ownership:              filepath.Join(dir, "ownership_v1.md"),
		Unsafe:                 filepath.Join(dir, "unsafe.md"),
		Capabilities:           filepath.Join(dir, "capabilities.md"),
		Stdlib:                 filepath.Join(dir, "stdlib.md"),
		StdlibGuide:            filepath.Join(dir, "standard_library_guide.md"),
		CoreMemory:             filepath.Join(dir, "memory.tetra"),
		TargetCapabilityMatrix: filepath.Join(dir, "memory-target-capability-matrix.md"),
		MemoryCostModel:        filepath.Join(dir, "memory_cost_model.md"),
		MemoryFuzzOracle:       filepath.Join(dir, "memory-fuzz-oracle-v1.md"),
		MemoryProductionFinal:  filepath.Join(dir, "memory-production-core-v1-final.md"),
		MemoryProductionMap:    filepath.Join(dir, "memory-production-core-v1-artifact-map.md"),
		MemoryProductionClaims: filepath.Join(dir, "memory-production-core-v1-nonclaims.md"),
	}
	for _, requirement := range memoryProductionContractRequirements(paths) {
		if requirement.Path == paths.MemoryProductionFinal {
			continue
		}
		body := strings.Join(requirement.Required, "\n")
		if err := os.WriteFile(requirement.Path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	err := verifyMemoryProductionContractDocs(paths)
	if err == nil || !strings.Contains(err.Error(), "memory-production-core-v1-final.md") {
		t.Fatalf("expected missing final audit failure, got %v", err)
	}
}

func TestVerifyMemoryProductionContractDocsRequiresArtifactMap(t *testing.T) {
	dir := t.TempDir()
	paths := memoryProductionContractDocPaths{
		RuntimeABI:             filepath.Join(dir, "runtime_abi.md"),
		Ownership:              filepath.Join(dir, "ownership_v1.md"),
		Unsafe:                 filepath.Join(dir, "unsafe.md"),
		Capabilities:           filepath.Join(dir, "capabilities.md"),
		Stdlib:                 filepath.Join(dir, "stdlib.md"),
		StdlibGuide:            filepath.Join(dir, "standard_library_guide.md"),
		CoreMemory:             filepath.Join(dir, "memory.tetra"),
		TargetCapabilityMatrix: filepath.Join(dir, "memory-target-capability-matrix.md"),
		MemoryCostModel:        filepath.Join(dir, "memory_cost_model.md"),
		MemoryFuzzOracle:       filepath.Join(dir, "memory-fuzz-oracle-v1.md"),
		MemoryProductionFinal:  filepath.Join(dir, "memory-production-core-v1-final.md"),
		MemoryProductionMap:    filepath.Join(dir, "memory-production-core-v1-artifact-map.md"),
		MemoryProductionClaims: filepath.Join(dir, "memory-production-core-v1-nonclaims.md"),
	}
	for _, requirement := range memoryProductionContractRequirements(paths) {
		if requirement.Path == paths.MemoryProductionMap {
			continue
		}
		body := strings.Join(requirement.Required, "\n")
		if err := os.WriteFile(requirement.Path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	err := verifyMemoryProductionContractDocs(paths)
	if err == nil || !strings.Contains(err.Error(), "memory-production-core-v1-artifact-map.md") {
		t.Fatalf("expected missing artifact map failure, got %v", err)
	}
}

func TestVerifyMemoryProductionContractDocsRequiresNonclaims(t *testing.T) {
	dir := t.TempDir()
	paths := memoryProductionContractDocPaths{
		RuntimeABI:             filepath.Join(dir, "runtime_abi.md"),
		Ownership:              filepath.Join(dir, "ownership_v1.md"),
		Unsafe:                 filepath.Join(dir, "unsafe.md"),
		Capabilities:           filepath.Join(dir, "capabilities.md"),
		Stdlib:                 filepath.Join(dir, "stdlib.md"),
		StdlibGuide:            filepath.Join(dir, "standard_library_guide.md"),
		CoreMemory:             filepath.Join(dir, "memory.tetra"),
		TargetCapabilityMatrix: filepath.Join(dir, "memory-target-capability-matrix.md"),
		MemoryCostModel:        filepath.Join(dir, "memory_cost_model.md"),
		MemoryFuzzOracle:       filepath.Join(dir, "memory-fuzz-oracle-v1.md"),
		MemoryProductionFinal:  filepath.Join(dir, "memory-production-core-v1-final.md"),
		MemoryProductionMap:    filepath.Join(dir, "memory-production-core-v1-artifact-map.md"),
		MemoryProductionClaims: filepath.Join(dir, "memory-production-core-v1-nonclaims.md"),
	}
	for _, requirement := range memoryProductionContractRequirements(paths) {
		if requirement.Path == paths.MemoryProductionClaims {
			continue
		}
		body := strings.Join(requirement.Required, "\n")
		if err := os.WriteFile(requirement.Path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	err := verifyMemoryProductionContractDocs(paths)
	if err == nil || !strings.Contains(err.Error(), "memory-production-core-v1-nonclaims.md") {
		t.Fatalf("expected missing nonclaims failure, got %v", err)
	}
}

func TestVerifyNetworkingRuntimeBoundaryDocsRejectsIncompleteBoundary(t *testing.T) {
	dir := t.TempDir()
	paths := networkingRuntimeBoundaryDocPaths{
		CurrentSurface: filepath.Join(dir, "current_supported_surface.md"),
		Stdlib:         filepath.Join(dir, "stdlib.md"),
		StdlibGuide:    filepath.Join(dir, "standard_library_guide.md"),
		CoreNet:        filepath.Join(dir, "net.tetra"),
		CoreNetworking: filepath.Join(dir, "networking.tetra"),
	}
	for _, path := range []string{paths.CurrentSurface, paths.Stdlib, paths.StdlibGuide, paths.CoreNet, paths.CoreNetworking} {
		if err := os.WriteFile(path, []byte("networking docs\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	err := verifyNetworkingRuntimeBoundaryDocs(paths)
	if err == nil {
		t.Fatalf("expected incomplete networking runtime boundary failure")
	}
	for _, want := range []string{"current_supported_surface.md", "TechEmpower-compatible web stack", "`lib.core.net`"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in error, got %v", want, err)
		}
	}
}

func TestVerifyNetworkingRuntimeBoundaryDocsAcceptsRequiredBoundary(t *testing.T) {
	dir := t.TempDir()
	paths := networkingRuntimeBoundaryDocPaths{
		CurrentSurface: filepath.Join(dir, "current_supported_surface.md"),
		Stdlib:         filepath.Join(dir, "stdlib.md"),
		StdlibGuide:    filepath.Join(dir, "standard_library_guide.md"),
		CoreNet:        filepath.Join(dir, "net.tetra"),
		CoreNetworking: filepath.Join(dir, "networking.tetra"),
	}
	for _, requirement := range networkingRuntimeBoundaryRequirements(paths) {
		body := strings.Join(requirement.Required, "\n")
		if err := os.WriteFile(requirement.Path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if err := verifyNetworkingRuntimeBoundaryDocs(paths); err != nil {
		t.Fatalf("verifyNetworkingRuntimeBoundaryDocs: %v", err)
	}
}

func TestVerifyFeatureRegistryAcceptsRequiredStatuses(t *testing.T) {
	features := []featureManifest{
		{ID: "cli.core", Name: "CLI", Status: "current", Since: "v0.2.0", Scope: "core CLI", Stability: "supported", Docs: []string{"docs/spec/current_supported_surface.md"}},
		{ID: "language.flow", Name: "Flow", Status: "current", Since: "v0.2.0", Scope: "flow syntax", Stability: "supported", Docs: []string{"docs/spec/flow_syntax_v1.md"}},
		{ID: "language.generics-mvp", Name: "Generics MVP", Status: "current", Since: "v0.2.0", Scope: "statically monomorphized generic functions with no runtime generic values or dynamic dispatch", Stability: "supported static MVP; generic structs remain future/post-v1", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/flow_syntax_v1.md", "docs/spec/v1_scope.md"}},
		{ID: "language.protocol-conformance-mvp", Name: "Protocol conformance MVP", Status: "current", Since: "v0.2.0", Scope: "checked statically with generic requirement signature shape and no witness tables", Stability: "dynamic dispatch remain post-v1", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/flow_syntax_v1.md", "docs/spec/v1_scope.md"}},
		{ID: "language.callable-mvp", Name: "Callable MVP", Status: "current", Since: "v0.2.0", Scope: "Level 0 callable surface", Stability: "current constrained MVP", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/flow_syntax_v1.md"}},
		{ID: "targets.wasm-artifact-preflight", Name: "WASM artifact/import preflight", Status: "current", Since: "v0.2.0", Scope: "artifact/import smoke", Stability: "supported", Docs: []string{"docs/backend/wasm_backend_plan.md"}},
		{ID: "stdlib.experimental-mirrors", Name: "Standard-library compatibility mirrors", Status: "current", Since: "v0.4.0", Scope: "production compatibility mirrors forward to lib.core modules", Stability: "stable callers should import lib.core directly", Docs: []string{"docs/spec/stdlib.md", "docs/spec/stdlib_naming_versioning.md", "docs/user/standard_library_guide.md"}},
		{ID: "language.callable-level1", Name: "Callable Level 1", Status: "current", Since: "v0.4.0", Scope: "production non-capturing symbol-backed callable Level 1 with function-typed locals, aliases, callbacks, and symbol-backed returns", Stability: "captured closure escape and full first-class function values remain out of scope", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/flow_syntax_v1.md", "docs/spec/v1_feature_status.md"}},
		{ID: "language.enum-payload-match", Name: "Enum payload", Status: "current", Since: "v0.3.0", Scope: "positional enum payload constructors and payload bindings for match/catch/if-let, with exhaustive unguarded enum match/catch", Stability: "nested destructuring patterns and guard expansion remain future/post-v1", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/flow_syntax_v1.md", "docs/spec/v0_3_scope.md"}},
		{ID: "language.protocol-bound-generics-static", Name: "Static protocol-bound generics", Status: "current", Since: "v0.3.0", Scope: "validated statically during monomorphization with same-module and cross-module impl conformance plus visibility diagnostics", Stability: "calling protocol requirements through generic bounds and dynamic dispatch remain unsupported", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/v0_3_scope.md", "docs/spec/flow_syntax_v1.md"}},
		{ID: "language.ownership-markers-mvp", Name: "Ownership markers MVP", Status: "current", Since: "v0.2.0", Scope: "conservative borrow/inout/consume marker checks with use-after-consume and borrow escape diagnostics", Stability: "supported conservative MVP; not a full SSA lifetime solver", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/ownership_v1.md", "docs/spec/v1_scope.md"}},
		{ID: "language.resource-lifetime-mvp", Name: "Resource lifetime MVP", Status: "current", Since: "v0.2.0", Scope: "conservative resource finalization checks for task handles, task groups, island handles, region-backed slices, and structs containing them, including double-use and ambiguous provenance diagnostics", Stability: "supported conservative MVP; tracks common local scope and control-flow merge cases, but is not a full SSA lifetime solver", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/ownership_v1.md", "docs/spec/v1_scope.md"}},
		{ID: "actors.task-transfer-safety", Name: "Actor/task transfer safety MVP", Status: "current", Since: "v0.2.0", Scope: "conservative actor/task ownership transfer checks for worker entrypoints and use-after-transfer diagnostics", Stability: "supported conservative local MVP; distributed actors remain outside current support", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/ownership_v1.md", "docs/spec/v1_scope.md"}},
		{ID: "language.lifetime-ssa", Name: "Lifetime SSA local join solver", Status: "current", Since: "v0.4.0", Scope: "production SSA-like local lifetime join analysis for ownership consume state, resource finalization state, branch/match/loop flow snapshots, and maybe-consumed diagnostics", Stability: "current local/control-flow solver; richer interprocedural lifetime proofs, broad alias modeling, race proofs, and full formal lifetime guarantees remain under full-v1 scope", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/ownership_v1.md", "docs/spec/v1_scope.md"}},
		{ID: "language.callable-level2", Name: "Callable Level 2", Status: "current", Since: "v0.4.0", Scope: "production captured closure Level 2 slice with function-typed locals called directly", Stability: "captured callback passing and full first-class callable semantics remain out of scope", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/flow_syntax_v1.md", "docs/spec/v1_feature_status.md"}},
		{ID: "ui.metadata-v1", Name: "UI metadata v1", Status: "current", Since: "v0.4.0", Scope: "production UI metadata contract with deterministic tetra.ui.v1 JSON", Stability: "web command dispatch; native widgets remain post-v1", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/ui_v1.md", "docs/user/wasm_ui_guide.md"}},
		{ID: "ui.native-runtime", Name: "Linux-x64 native UI runtime", Status: "current", Since: "v0.4.0", Scope: "production Linux-x64 native UI runtime path with native runtime widget instances, click/activate events, and state and widget updates", Stability: "tetra.ui.native-runtime.v1 smoke evidence rejects metadata-only, web-only, native-shell sidecar-only evidence; macOS/Windows remain outside this claim", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/ui_v1.md", "docs/user/wasm_ui_guide.md"}},
		{ID: "ui.platform-runtime", Name: "Cross-platform UI runtime gate", Status: "experimental", Since: "v0.4.0", Scope: "tetra.ui.platform-runtime.v1 full-platform UI runtime promotion gate requiring real Windows/macOS target-host reports", Stability: "not production until the gate rejects metadata-only, runtime-less, startup_failure evidence and accepts target-host reports", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/ui_v1.md", "docs/user/wasm_ui_guide.md"}},
		{ID: "wasm.runtime-execution", Name: "WASM runtime execution", Status: "current", Since: "v0.4.0", Scope: "production WASI runner and browser-backed wasm32-web execution", Stability: "supported with runner/browser availability diagnostics", Docs: []string{"docs/spec/current_supported_surface.md", "docs/backend/wasm_backend_plan.md", "docs/user/wasm_ui_guide.md"}},
		{ID: "actors.distributed-runtime", Name: "Distributed actor runtime for Linux x64", Status: "current", Since: "v0.4.0", Scope: "production Linux-x64 distributed actor runtime path with actornet loopback TCP broker, distributed node identity, remote actor handles, network mailbox send/receive, and scoped actor runtime foundation gate evidence through tetra.actor.production_foundation.v1", Stability: "current Linux-x64 runtime/lowering slice with executable tetra.actors.distributed-runtime.v1 smoke evidence, tetra.actor.production_foundation.v1 gate evidence from actor-runtime-foundation-linux-x64-gate.sh, and strict nonclaims for non-Linux distributed runtime, distributed zero-copy, cluster membership, reconnect/retry production, and formal race proof", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/actors.md", "docs/user/async_actors_guide.md", "docs/design/actor_region_transfer.md", "docs/audits/actor-runtime-production-boundary-v1.md", "docs/checklists/actors_linux_smoke.md", "docs/checklists/actors_platform_smoke.md"}},
		validVerifyDocsSafetyProductionCoreFeature(),
		validVerifyDocsRAMContractFeature(),
		{ID: "language.full-v1-guarantees", Name: "v1", Status: "planned", Scope: "v1", Stability: "planned", Docs: []string{"docs/spec/v1_scope.md"}},
		{ID: "eco.distributed-network", Name: "EcoNet", Status: "post-v1", Scope: "network", Stability: "deferred", Docs: []string{"docs/release/post_v1_promotion_checklist.md"}},
		{ID: "language.full-first-class-callables", Name: "Callables", Status: "current", Since: "v0.4.0", Scope: "safe by-value first-class callable semantics", Stability: "current safe-capture model", Docs: []string{"docs/spec/v1_feature_status.md"}},
		{ID: "ui.surface-core", Name: "Tetra Surface core", Status: "release_candidate", Scope: "surface-v1-linux-web", Stability: "release gate candidate", Docs: []string{"docs/spec/surface_v1.md"}},
		{
			ID:        "ui.surface-block-system",
			Name:      "Tetra Surface Block System",
			Status:    "experimental",
			Scope:     "Block-first Surface architecture with Block as the core Surface primitive, widgets as recipes/compatibility, `tetra.surface.block-system.gate.v1` gate reports, and `block_system.memory_budget` evidence under reports/surface-block/p18-budget",
			Stability: "experimental implementation track with same-commit target evidence for headless, linux-x64 real-window, and wasm32-web browser-canvas; not production support and no production Block claim",
			Docs: []string{
				"docs/spec/current_supported_surface.md",
				"docs/spec/surface_v1.md",
				"docs/user/surface_guide.md",
				"docs/user/examples_index.md",
				"docs/release/surface_v1_release_contract.md",
				"docs/release/surface_v1_release_notes.md",
				"docs/release/surface_v1_release_audit.md",
			},
		},
		{ID: "ui.surface-macos-x64", Name: "macOS Surface host", Status: "unsupported", Scope: "not in Surface v1", Stability: "no release evidence", Docs: []string{"docs/spec/surface_v1.md"}},
		{ID: "ui.metadata-legacy", Name: "UI metadata legacy compatibility", Status: "legacy_compatibility", Scope: "legacy metadata compatibility", Stability: "compatibility bridge", Docs: []string{"docs/spec/ui_v1.md"}},
	}
	if err := verifyFeatureRegistry(features); err != nil {
		t.Fatalf("verifyFeatureRegistry: %v", err)
	}
}

func TestVerifySurfaceBlockSystemFeatureBoundaryRequiresP18Evidence(t *testing.T) {
	features := map[string]featureManifest{
		"ui.surface-core": {
			ID:        "ui.surface-core",
			Name:      "Tetra Surface core",
			Status:    "current",
			Scope:     "surface-v1-linux-web",
			Stability: "current bounded release scope",
			Docs:      []string{"docs/spec/surface_v1.md"},
		},
		"ui.surface-block-system": {
			ID:        "ui.surface-block-system",
			Name:      "Tetra Surface Block System",
			Status:    "experimental",
			Scope:     "Block-first Surface architecture with Block as the core Surface primitive and widgets as recipes/compatibility",
			Stability: "implementation track; not current; no production Block claim",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/spec/surface_v1.md", "docs/user/surface_guide.md", "docs/release/surface_v1_release_contract.md"},
		},
	}

	err := verifySurfaceBlockSystemFeatureBoundary(features)
	if err == nil {
		t.Fatalf("expected missing P18 Block evidence boundary failure")
	}
	for _, want := range []string{"tetra.surface.block-system.gate.v1", "block_system.memory_budget", "reports/surface-block/p18-budget"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in error, got %v", want, err)
		}
	}
}

func TestVerifyFeatureRegistryRejectsFutureMismatch(t *testing.T) {
	features := []featureManifest{
		{ID: "cli.core", Name: "CLI", Status: "current", Since: "v0.2.0", Scope: "core CLI", Stability: "supported", Docs: []string{"docs/spec/current_supported_surface.md"}},
		{ID: "language.flow", Name: "Flow", Status: "current", Since: "v0.2.0", Scope: "flow syntax", Stability: "supported", Docs: []string{"docs/spec/flow_syntax_v1.md"}},
		{ID: "language.generics-mvp", Name: "Generics MVP", Status: "current", Since: "v0.2.0", Scope: "statically monomorphized generic functions with no runtime generic values or dynamic dispatch", Stability: "supported static MVP; generic structs remain future/post-v1", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/flow_syntax_v1.md", "docs/spec/v1_scope.md"}},
		{ID: "language.protocol-conformance-mvp", Name: "Protocol conformance MVP", Status: "current", Since: "v0.2.0", Scope: "checked statically with generic requirement signature shape and no witness tables", Stability: "dynamic dispatch remain post-v1", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/flow_syntax_v1.md", "docs/spec/v1_scope.md"}},
		{ID: "language.callable-mvp", Name: "Callable MVP", Status: "current", Since: "v0.2.0", Scope: "Level 0 callable surface", Stability: "current constrained MVP", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/flow_syntax_v1.md"}},
		{ID: "targets.wasm-artifact-preflight", Name: "WASM artifact/import preflight", Status: "current", Since: "v0.2.0", Scope: "artifact/import smoke", Stability: "supported", Docs: []string{"docs/backend/wasm_backend_plan.md"}},
		{ID: "stdlib.experimental-mirrors", Name: "Standard-library compatibility mirrors", Status: "current", Since: "v0.4.0", Scope: "production compatibility mirrors forward to lib.core modules", Stability: "stable callers should import lib.core directly", Docs: []string{"docs/spec/stdlib.md", "docs/spec/stdlib_naming_versioning.md", "docs/user/standard_library_guide.md"}},
		{ID: "language.callable-level1", Name: "Callable Level 1", Status: "current", Since: "v0.4.0", Scope: "production non-capturing symbol-backed callable Level 1 with function-typed locals, aliases, callbacks, and symbol-backed returns", Stability: "captured closure escape and full first-class function values remain out of scope", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/flow_syntax_v1.md", "docs/spec/v1_feature_status.md"}},
		{ID: "language.enum-payload-match", Name: "Enum payload", Status: "current", Since: "v0.3.0", Scope: "positional enum payload constructors and payload bindings for match/catch/if-let, with exhaustive unguarded enum match/catch", Stability: "nested destructuring patterns and guard expansion remain future/post-v1", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/flow_syntax_v1.md", "docs/spec/v0_3_scope.md"}},
		{ID: "language.protocol-bound-generics-static", Name: "Static protocol-bound generics", Status: "current", Since: "v0.3.0", Scope: "validated statically during monomorphization with same-module and cross-module impl conformance plus visibility diagnostics", Stability: "calling protocol requirements through generic bounds and dynamic dispatch remain unsupported", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/v0_3_scope.md", "docs/spec/flow_syntax_v1.md"}},
		{ID: "language.ownership-markers-mvp", Name: "Ownership markers MVP", Status: "current", Since: "v0.2.0", Scope: "conservative borrow/inout/consume marker checks with use-after-consume and borrow escape diagnostics", Stability: "supported conservative MVP; not a full SSA lifetime solver", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/ownership_v1.md", "docs/spec/v1_scope.md"}},
		{ID: "language.resource-lifetime-mvp", Name: "Resource lifetime MVP", Status: "current", Since: "v0.2.0", Scope: "conservative resource finalization checks for task handles, task groups, island handles, region-backed slices, and structs containing them, including double-use and ambiguous provenance diagnostics", Stability: "supported conservative MVP; tracks common local scope and control-flow merge cases, but is not a full SSA lifetime solver", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/ownership_v1.md", "docs/spec/v1_scope.md"}},
		{ID: "actors.task-transfer-safety", Name: "Actor/task transfer safety MVP", Status: "current", Since: "v0.2.0", Scope: "conservative actor/task ownership transfer checks for worker entrypoints and use-after-transfer diagnostics", Stability: "supported conservative local MVP; distributed actors remain outside current support", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/ownership_v1.md", "docs/spec/v1_scope.md"}},
		{ID: "language.lifetime-ssa", Name: "Lifetime SSA solver", Status: "planned", Scope: "stale planned lifetime solver fixture", Stability: "unsupported stale fixture", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/ownership_v1.md", "docs/spec/v1_scope.md"}},
		{ID: "language.callable-level2", Name: "Callable Level 2", Status: "current", Since: "v0.4.0", Scope: "production captured closure Level 2 slice with function-typed locals called directly", Stability: "captured callback passing and full first-class callable semantics remain out of scope", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/flow_syntax_v1.md", "docs/spec/v1_feature_status.md"}},
		{ID: "ui.metadata-v1", Name: "UI metadata v1", Status: "current", Since: "v0.4.0", Scope: "production UI metadata contract with deterministic tetra.ui.v1 JSON", Stability: "web command dispatch; native widgets remain post-v1", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/ui_v1.md", "docs/user/wasm_ui_guide.md"}},
		{ID: "ui.native-runtime", Name: "Linux-x64 native UI runtime", Status: "current", Since: "v0.4.0", Scope: "production Linux-x64 native UI runtime path with native runtime widget instances, click/activate events, and state and widget updates", Stability: "tetra.ui.native-runtime.v1 smoke evidence rejects metadata-only, web-only, native-shell sidecar-only evidence; macOS/Windows remain outside this claim", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/ui_v1.md", "docs/user/wasm_ui_guide.md"}},
		{ID: "ui.platform-runtime", Name: "Cross-platform UI runtime gate", Status: "experimental", Since: "v0.4.0", Scope: "tetra.ui.platform-runtime.v1 full-platform UI runtime promotion gate requiring real Windows/macOS target-host reports", Stability: "not production until the gate rejects metadata-only, runtime-less, startup_failure evidence and accepts target-host reports", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/ui_v1.md", "docs/user/wasm_ui_guide.md"}},
		{ID: "wasm.runtime-execution", Name: "WASM runtime execution", Status: "current", Since: "v0.4.0", Scope: "production WASI runner and browser-backed wasm32-web execution", Stability: "supported with runner/browser availability diagnostics", Docs: []string{"docs/spec/current_supported_surface.md", "docs/backend/wasm_backend_plan.md", "docs/user/wasm_ui_guide.md"}},
		{ID: "actors.distributed-runtime", Name: "Distributed actor runtime for Linux x64", Status: "current", Since: "v0.4.0", Scope: "production Linux-x64 distributed actor runtime path with actornet loopback TCP broker, distributed node identity, remote actor handles, network mailbox send/receive, and scoped actor runtime foundation gate evidence through tetra.actor.production_foundation.v1", Stability: "current Linux-x64 runtime/lowering slice with executable tetra.actors.distributed-runtime.v1 smoke evidence, tetra.actor.production_foundation.v1 gate evidence from actor-runtime-foundation-linux-x64-gate.sh, and strict nonclaims for non-Linux distributed runtime, distributed zero-copy, cluster membership, reconnect/retry production, and formal race proof", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/actors.md", "docs/user/async_actors_guide.md", "docs/design/actor_region_transfer.md", "docs/audits/actor-runtime-production-boundary-v1.md", "docs/checklists/actors_linux_smoke.md", "docs/checklists/actors_platform_smoke.md"}},
		validVerifyDocsSafetyProductionCoreFeature(),
		validVerifyDocsRAMContractFeature(),
		{ID: "language.full-v1-guarantees", Name: "v1", Status: "planned", Scope: "v1", Stability: "planned", Docs: []string{"docs/spec/v1_scope.md"}},
		{ID: "eco.distributed-network", Name: "EcoNet", Status: "post-v1", Scope: "network", Stability: "deferred", Docs: []string{"docs/release/post_v1_promotion_checklist.md"}},
		{ID: "language.full-first-class-callables", Name: "Callables", Status: "current", Since: "v0.4.0", Scope: "safe by-value first-class callable semantics", Stability: "current safe-capture model", Docs: []string{"docs/spec/v1_feature_status.md"}},
	}
	err := verifyFeatureRegistry(features)
	if err == nil {
		t.Fatalf("expected future mismatch failure")
	}
	if !strings.Contains(err.Error(), "language.lifetime-ssa") || !strings.Contains(err.Error(), "want current") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func validVerifyDocsSafetyProductionCoreFeature() featureManifest {
	return featureManifest{
		ID:        "safety.production-core",
		Name:      "Production safety core",
		Status:    "current",
		Since:     "v0.4.0",
		Scope:     "production local safety model for ownership/lifetime/borrow/consume/inout checks, Memory Production Core v1 report evidence, memory production final audit with artifact map and explicit nonclaims, validate-island-proof independent-ish verifier evidence, --islands-debug sanitizer smoke, island-proof-fuzz-summary deterministic mutation evidence, leak/resource finalization evidence, integrated Memory/Islands/Surface release gate with memory-islands-surface-production-manifest.json and artifact-hashes.json, and no Memory 100% claim",
		Stability: "release-gated current profile with explicit diagnostics for unsupported distributed, cryptographic, formal-proof, runtime-wide guarantees, arbitrary unsafe external pointer safety, full target parity, all-target Surface support, clean release-candidate checkout claims, and no production object memory or production persistent memory claim",
		Docs: []string{
			"docs/spec/current_supported_surface.md",
			"docs/spec/ownership_v1.md",
			"docs/spec/effects_capabilities_privacy_v1.md",
			"docs/spec/unsafe.md",
			"docs/spec/memory_report_schema_v1.md",
			"docs/spec/islands.md",
			"docs/design/memory_production_core_v1.md",
			"docs/design/memory_cost_model.md",
			"docs/audits/memory-fuzz-oracle-v1.md",
			"docs/audits/memory-production-core-v1-final.md",
			"docs/audits/memory-production-core-v1-artifact-map.md",
			"docs/audits/memory-production-core-v1-nonclaims.md",
			"docs/release/memory_islands_surface_scope.md",
		},
	}
}

func validVerifyDocsRAMContractFeature() featureManifest {
	return featureManifest{
		ID:        "compiler.ram-contracts",
		Name:      "RAM Contract Compiler reports",
		Status:    "current",
		Since:     "v0.4.0",
		Scope:     "RAM Contract Compiler report evidence with tetra.ram-contract-report.v1, tetra.memory-grade-report.v1, tetra.proof-store-summary.v1, tetra.validation-pipeline-coverage.v1, heap-blockers.json, copy-blockers.json, ram-contract-fuzz-oracle.json, --emit-ram-contract-report, --fail-if-heap, --fail-if-copy, --fail-if-unbounded, --memory-budget, --ram-contract, TETRA4100, validate-ram-contract-release, and ram-contract-linux-x64-smoke.sh",
		Stability: "current report/gate contract only; no zero heap for all programs claim, no zero-copy for all programs claim, no full formal proof claim, no all-target RAM parity claim, no production object memory claim, no production persistent memory claim, and no performance claim",
		Docs: []string{
			"docs/design/ram_contract_compiler.md",
			"docs/spec/ram_contract_report_schema.md",
			"docs/user/ram_contracts.md",
			"docs/audits/ram-contract-compiler-readiness.md",
			"docs/audits/ram-contract-compiler-handoff.md",
		},
	}
}

func TestVerifyReleaseTruthDocsRejectsMisleadingCurrentReleaseLanguage(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "current_supported_surface.md")
	body := strings.Join([]string{
		"# Current Surface",
		"",
		"The current public profile is v0.3.0.",
		"The current public baseline is v0.1.2.",
		"The current release is v0.6.",
		"Tetra is ready for v1.0.",
	}, "\n")
	if err := os.WriteFile(doc, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	err := verifyReleaseTruthDocs([]string{doc})
	if err == nil {
		t.Fatalf("expected misleading release language failure")
	}
	for _, want := range []string{"current.*v0.3", "v0.1.2", "current.*v0.6", "ready for v1.0"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in error, got %v", want, err)
		}
	}
}

func TestVerifyReleaseTruthDocsRejectsPerformanceAndTargetParityClaims(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "release_notes.md")
	body := strings.Join([]string{
		"# Release Notes",
		"",
		"Tetra is the fastest language in the official benchmark result.",
		"The package also proves target parity for memory production.",
		"The allocator has broad zero-cost performance across targets.",
	}, "\n")
	if err := os.WriteFile(doc, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	err := verifyReleaseTruthDocs([]string{doc})
	if err == nil {
		t.Fatalf("expected performance/target parity claim failure")
	}
	for _, want := range []string{"fastest language", "official benchmark", "target parity", "zero-cost performance"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in error, got %v", want, err)
		}
	}
}

func TestVerifyReleaseTruthDocsRejectsMemory100FormalProofAndLeakClaims(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "release_notes.md")
	body := strings.Join([]string{
		"# Release Notes",
		"",
		"The release proves full formal proof of memory safety.",
		"Memory production now has all-target memory parity.",
		"The memory model has no leaks.",
		"Memory 100% is guaranteed for users.",
	}, "\n")
	if err := os.WriteFile(doc, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	err := verifyReleaseTruthDocs([]string{doc})
	if err == nil {
		t.Fatalf("expected Memory100/formal/leak claim failure")
	}
	for _, want := range []string{"full formal proof", "all-target memory parity", "no leaks", "memory 100%"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in error, got %v", want, err)
		}
	}
}

func TestVerifyReleaseTruthDocsRejectsProductionPersistentObjectMemoryClaim(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "release_notes.md")
	body := strings.Join([]string{
		"# Release Notes",
		"",
		"Tetra now ships production object memory backed by persistent memory, Todium, memoryfield, WAL, FTS, vacuum, retention, stale memory, and false memory gates.",
	}, "\n")
	if err := os.WriteFile(doc, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	err := verifyReleaseTruthDocs([]string{doc})
	if err == nil {
		t.Fatalf("expected production persistent/object memory claim failure")
	}
	for _, want := range []string{"production object memory", "persistent memory", "todium", "memoryfield"} {
		if !strings.Contains(strings.ToLower(err.Error()), want) {
			t.Fatalf("expected %q in error, got %v", want, err)
		}
	}
}

func TestVerifyReleaseTruthDocsAllowsPersistentObjectMemoryNonGoal(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "release_notes.md")
	body := strings.Join([]string{
		"# Release Notes",
		"",
		"Persistent/object memory is an explicit non-goal for this release: no production object memory, no production persistent memory, and no Todium or memoryfield production claim exists until retention/WAL/FTS/vacuum/stale/false-memory gates exist.",
	}, "\n")
	if err := os.WriteFile(doc, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := verifyReleaseTruthDocs([]string{doc}); err != nil {
		t.Fatalf("verifyReleaseTruthDocs: %v", err)
	}
}

func TestVerifyReleaseTruthDocsAllowsHistoricalTodoExclusion(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "2026-04-27-tetra-stabilization-5000-todo.md")
	body := "Historical TODO mentions current v0.6 and v0.1.2 for audit context.\n"
	if err := os.WriteFile(doc, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := verifyReleaseTruthDocs([]string{doc}); err != nil {
		t.Fatalf("verifyReleaseTruthDocs: %v", err)
	}
}

func TestCurrentReleaseTruthDocPathsCoverCurrentUserAndSpecDocs(t *testing.T) {
	paths := currentReleaseTruthDocPaths()
	text := strings.Join(paths, "\n")
	for _, want := range []string{
		"README.md",
		"docs/spec/current_supported_surface.md",
		"docs/spec/surface_v1.md",
		"docs/spec/v0_2_scope.md",
		"docs/user/examples_index.md",
		"docs/user/getting_started.md",
		"docs/user/language_tour.md",
		"docs/user/surface_guide.md",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("currentReleaseTruthDocPaths missing %s in %v", want, paths)
		}
	}
	for _, forbidden := range []string{"docs/plans/", "docs/release-notes/"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("currentReleaseTruthDocPaths should not include historical %s paths: %v", forbidden, paths)
		}
	}
}

func TestVerifySurfaceReleaseDocsRejectsFakePromotionClaims(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want string
	}{
		{
			name: "macos-current",
			body: "macOS Surface is current for Surface v1.\nUnsupported targets: wasm32-wasi.\nbash scripts/release/surface/release-gate.sh\n",
			want: "macOS Surface",
		},
		{
			name: "windows-current",
			body: "Windows Surface is release-ready for Surface v1.\nUnsupported targets: wasm32-wasi.\nbash scripts/release/surface/release-gate.sh\n",
			want: "Windows Surface",
		},
		{
			name: "metadata-only-production-accessibility",
			body: "metadata-only accessibility is production accessibility.\nUnsupported targets: macOS, Windows, wasm32-wasi.\nbash scripts/release/surface/release-gate.sh\n",
			want: "metadata-only",
		},
		{
			name: "dom-ui-model",
			body: "DOM UI is the Surface model.\nUnsupported targets: macOS, Windows, wasm32-wasi.\nbash scripts/release/surface/release-gate.sh\n",
			want: "DOM UI",
		},
		{
			name: "user-js-allowed",
			body: "user JS app logic is allowed in Surface apps.\nUnsupported targets: macOS, Windows, wasm32-wasi.\nbash scripts/release/surface/release-gate.sh\n",
			want: "user JS",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc := writeSurfaceReleaseDoc(t, tc.body)
			err := verifySurfaceReleaseDocs([]string{doc})
			if err == nil {
				t.Fatalf("expected Surface release docs fake-promotion failure")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestVerifySurfaceReleaseDocsRequireUnsupportedTargetsAndReleaseGate(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want string
	}{
		{
			name: "missing-unsupported-targets",
			body: "Surface v1 scope is linux-x64 real-window and wasm32-web browser-canvas.\nbash scripts/release/surface/release-gate.sh\n",
			want: "unsupported targets",
		},
		{
			name: "missing-release-gate-command",
			body: "Surface v1 scope is linux-x64 real-window and wasm32-web browser-canvas. Unsupported targets: macOS, Windows, wasm32-wasi.\n",
			want: "release-gate.sh",
		},
		{
			name: "missing-claim-tier-vocabulary",
			body: "Surface v1 scope is linux-x64 real-window and wasm32-web browser-canvas. macOS Surface, Windows Surface, and wasm32-wasi Surface UI are unsupported targets.\n\nbash scripts/release/surface/release-gate.sh\nbash scripts/release/surface/product-gate.sh\n",
			want: "PROD_STABLE_SCOPED",
		},
		{
			name: "missing-product-gate-command",
			body: "Surface v1 scope is linux-x64 real-window and wasm32-web browser-canvas. macOS Surface, Windows Surface, and wasm32-wasi Surface UI are unsupported targets. Claim tiers: PROD_STABLE_SCOPED, BETA_TARGET_HOST, EXPERIMENTAL, UNSUPPORTED, NONCLAIM.\n\nbash scripts/release/surface/release-gate.sh\n",
			want: "product-gate.sh",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc := writeSurfaceReleaseDoc(t, tc.body)
			err := verifySurfaceReleaseDocs([]string{doc})
			if err == nil {
				t.Fatalf("expected Surface release docs requirement failure")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}

	okDoc := writeSurfaceReleaseDoc(t, "Surface v1 scope is linux-x64 real-window and wasm32-web browser-canvas. macOS Surface, Windows Surface, and wasm32-wasi Surface UI are unsupported targets. Metadata-only accessibility is not production accessibility. DOM UI and user JavaScript app logic are outside the Surface model. Claim tiers: PROD_STABLE_SCOPED, BETA_TARGET_HOST, EXPERIMENTAL, UNSUPPORTED, NONCLAIM.\n\nbash scripts/release/surface/release-gate.sh\nbash scripts/release/surface/product-gate.sh\n")
	if err := verifySurfaceReleaseDocs([]string{okDoc}); err != nil {
		t.Fatalf("verifySurfaceReleaseDocs accepted doc: %v", err)
	}
}

func TestVerifySurfaceReleaseDocsRequireP28GovernancePerDocument(t *testing.T) {
	fullDoc := writeSurfaceReleaseDoc(t, "Surface v1 scope is linux-x64 real-window and wasm32-web browser-canvas. macOS Surface, Windows Surface, and wasm32-wasi Surface UI are unsupported targets. Claim tiers: PROD_STABLE_SCOPED, BETA_TARGET_HOST, EXPERIMENTAL, UNSUPPORTED, NONCLAIM.\n\nbash scripts/release/surface/release-gate.sh\nbash scripts/release/surface/product-gate.sh\n")
	missingTierDoc := writeSurfaceReleaseDoc(t, "Surface v1 scope is linux-x64 real-window and wasm32-web browser-canvas. macOS Surface, Windows Surface, and wasm32-wasi Surface UI are unsupported targets.\n\nbash scripts/release/surface/release-gate.sh\nbash scripts/release/surface/product-gate.sh\n")
	err := verifySurfaceReleaseDocs([]string{fullDoc, missingTierDoc})
	if err == nil {
		t.Fatalf("expected per-document claim-tier requirement failure")
	}
	if !strings.Contains(err.Error(), "PROD_STABLE_SCOPED") {
		t.Fatalf("error = %v, want PROD_STABLE_SCOPED diagnostic", err)
	}
}

func TestVerifySurfaceReleaseDocsRejectsMixedGPUProductionWithoutEvidenceClause(t *testing.T) {
	doc := writeSurfaceReleaseDoc(t, "Surface v1 scope is linux-x64 real-window and wasm32-web browser-canvas. macOS Surface, Windows Surface, and wasm32-wasi Surface UI are unsupported targets. Claim tiers: PROD_STABLE_SCOPED, BETA_TARGET_HOST, EXPERIMENTAL, UNSUPPORTED, NONCLAIM.\n\nSurface GPU rendering is production supported without additional evidence.\n\nbash scripts/release/surface/release-gate.sh\nbash scripts/release/surface/product-gate.sh\n")
	err := verifySurfaceReleaseDocs([]string{doc})
	if err == nil {
		t.Fatalf("expected mixed GPU production claim failure")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "gpu") {
		t.Fatalf("error = %v, want GPU diagnostic", err)
	}
}

func TestVerifySurfaceReleaseDocsRejectsFinalCurrentClaimOwnership(t *testing.T) {
	doc := writeSurfaceReleaseDoc(t, "Surface v1 scope is linux-x64 real-window and wasm32-web browser-canvas. macOS Surface, Windows Surface, and wasm32-wasi Surface UI are unsupported targets. Claim tiers: PROD_STABLE_SCOPED, BETA_TARGET_HOST, EXPERIMENTAL, UNSUPPORTED, NONCLAIM.\n\nThe release gate is the source of truth for the final current claim.\n\nbash scripts/release/surface/release-gate.sh\nbash scripts/release/surface/product-gate.sh\n")
	err := verifySurfaceReleaseDocs([]string{doc})
	if err == nil {
		t.Fatalf("expected final current claim ownership failure")
	}
	if !strings.Contains(err.Error(), "final current claim") {
		t.Fatalf("error = %v, want final current claim diagnostic", err)
	}
}

func TestSurfaceDocsOverclaimRejectsTmpEvidenceAsCurrentProof(t *testing.T) {
	doc := writeSurfaceReleaseDoc(t, "Surface v1 scope is linux-x64 real-window and wasm32-web browser-canvas. macOS Surface, Windows Surface, and wasm32-wasi Surface UI are unsupported targets. Metadata-only accessibility is not production accessibility. DOM UI and user JavaScript app logic are outside the Surface model.\n\nbash scripts/release/surface/release-gate.sh --report-dir /tmp/tetra-surface-release-v1-current\n")
	err := verifySurfaceReleaseDocs([]string{doc})
	if err == nil {
		t.Fatalf("expected Surface release docs to reject /tmp current evidence")
	}
	if !strings.Contains(err.Error(), "/tmp") {
		t.Fatalf("error = %v, want /tmp rejection", err)
	}
}

func TestSurfaceOverclaimRejectsGPUAndNativeWidgets(t *testing.T) {
	doc := writeSurfaceReleaseDoc(t, "Surface v1 scope is linux-x64 real-window and wasm32-web browser-canvas. macOS Surface, Windows Surface, and wasm32-wasi Surface UI are unsupported targets. Metadata-only accessibility is not production accessibility. DOM UI and user JavaScript app logic are outside the Surface model.\n\nGPU rendering is production-supported for Surface v1. Platform-native widgets are release-ready.\n\nbash scripts/release/surface/release-gate.sh\n")
	err := verifySurfaceReleaseDocs([]string{doc})
	if err == nil {
		t.Fatalf("expected Surface docs to reject GPU/native-widget overclaims")
	}
	for _, want := range []string{"GPU", "native widget"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in error, got %v", want, err)
		}
	}
}

func TestUnsupportedSurfaceTargetsRejectsCrossPlatformProductionClaim(t *testing.T) {
	doc := writeSurfaceReleaseDoc(t, "Surface v1 scope is linux-x64 real-window and wasm32-web browser-canvas. macOS Surface, Windows Surface, and wasm32-wasi Surface UI are unsupported targets.\n\nSurface is a production cross-platform UI runtime across macOS, Windows, linux, and wasm32-wasi.\n\nbash scripts/release/surface/release-gate.sh\n")
	err := verifySurfaceReleaseDocs([]string{doc})
	if err == nil {
		t.Fatalf("expected Surface docs to reject cross-platform production overclaim")
	}
	if !strings.Contains(err.Error(), "cross-platform") {
		t.Fatalf("expected cross-platform in error, got %v", err)
	}
}

func TestSurfaceOverclaimRejectsRichTextScreenReaderDOMReactUserJS(t *testing.T) {
	doc := writeSurfaceReleaseDoc(t, "Surface v1 scope is linux-x64 real-window and wasm32-web browser-canvas. macOS Surface, Windows Surface, and wasm32-wasi Surface UI are unsupported targets.\n\nRich text editing is production-supported. Full screen-reader support is release-ready. DOM UI is production-supported. React apps are current Surface apps. User JS app logic is allowed in Surface apps.\n\nbash scripts/release/surface/release-gate.sh\n")
	err := verifySurfaceReleaseDocs([]string{doc})
	if err == nil {
		t.Fatalf("expected Surface docs to reject rich-text/screen-reader/DOM/React/user-JS overclaims")
	}
	for _, want := range []string{"rich text", "screen-reader", "DOM UI", "React", "user JS"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in error, got %v", want, err)
		}
	}
}

func TestSurfaceBlockSystemRejectsCoreWidgetPrimitiveClaims(t *testing.T) {
	doc := writeSurfaceReleaseDoc(t, "Surface v1 scope is linux-x64 real-window and wasm32-web browser-canvas. macOS Surface, Windows Surface, and wasm32-wasi Surface UI are unsupported targets.\n\nButton is a core Surface primitive. TextField is a core Surface primitive. Card is a core Surface primitive.\n\nbash scripts/release/surface/release-gate.sh\n")
	err := verifySurfaceReleaseDocs([]string{doc})
	if err == nil {
		t.Fatalf("expected Surface docs to reject core widget primitive claims")
	}
	for _, want := range []string{"Button", "TextField", "Card"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in error, got %v", want, err)
		}
	}
}

func TestVerifyMemoryIslandsSurfaceReleaseDocsRejectsIncompleteScope(t *testing.T) {
	doc := writeMemoryIslandsSurfaceReleaseDoc(t, "Memory/Islands/Surface scoped release truth.\n")
	err := verifyMemoryIslandsSurfaceReleaseDocs([]string{doc})
	if err == nil {
		t.Fatalf("expected incomplete Memory/Islands/Surface release docs failure")
	}
	for _, want := range []string{"validate-island-proof", "memory-islands-surface-production-gate.sh"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in error, got %v", want, err)
		}
	}
}

func TestVerifyMemoryIslandsSurfaceReleaseDocsRejectsBroadOverclaim(t *testing.T) {
	doc := writeMemoryIslandsSurfaceReleaseDoc(t, strings.Join([]string{
		validMemoryIslandsSurfaceReleaseDocBody(),
		"Memory/Islands/Surface is fully production-ready across all targets.",
	}, "\n"))
	err := verifyMemoryIslandsSurfaceReleaseDocs([]string{doc})
	if err == nil {
		t.Fatalf("expected Memory/Islands/Surface broad overclaim failure")
	}
	if !strings.Contains(err.Error(), "fully production-ready") {
		t.Fatalf("expected fully production-ready in error, got %v", err)
	}
}

func TestVerifyMemoryIslandsSurfaceReleaseDocsAcceptsScopedEvidence(t *testing.T) {
	doc := writeMemoryIslandsSurfaceReleaseDoc(t, validMemoryIslandsSurfaceReleaseDocBody())
	if err := verifyMemoryIslandsSurfaceReleaseDocs([]string{doc}); err != nil {
		t.Fatalf("verifyMemoryIslandsSurfaceReleaseDocs: %v", err)
	}
}

func TestVerifyFinalMemoryIslandsSurfaceProductionAuditRejectsMissingCommands(t *testing.T) {
	body := strings.ReplaceAll(validFinalMemoryIslandsSurfaceProductionAuditBody(), "git status --short", "")
	doc := writeFinalMemoryIslandsSurfaceProductionAudit(t, body)
	err := verifyFinalMemoryIslandsSurfaceProductionAudit([]string{doc})
	if err == nil {
		t.Fatalf("expected final production audit to reject missing command evidence")
	}
	if !strings.Contains(err.Error(), "git status --short") {
		t.Fatalf("expected git status command in error, got %v", err)
	}
}

func TestVerifyFinalMemoryIslandsSurfaceProductionAuditRejectsBroadReadyClaim(t *testing.T) {
	doc := writeFinalMemoryIslandsSurfaceProductionAudit(t, validFinalMemoryIslandsSurfaceProductionAuditBody()+"\nIntegrated: PROD_READY_PROVEN across all targets.\n")
	err := verifyFinalMemoryIslandsSurfaceProductionAudit([]string{doc})
	if err == nil {
		t.Fatalf("expected final production audit to reject broad PROD_READY_PROVEN claim")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "prod_ready_proven") {
		t.Fatalf("expected PROD_READY_PROVEN in error, got %v", err)
	}
}

func TestVerifyFinalMemoryIslandsSurfaceProductionAuditAcceptsScopedEvidence(t *testing.T) {
	doc := writeFinalMemoryIslandsSurfaceProductionAudit(t, validFinalMemoryIslandsSurfaceProductionAuditBody())
	if err := verifyFinalMemoryIslandsSurfaceProductionAudit([]string{doc}); err != nil {
		t.Fatalf("verifyFinalMemoryIslandsSurfaceProductionAudit: %v", err)
	}
}

func TestVerifyMemoryIslandsFinalProductionReadinessAuditRejectsMissingCommandLogArtifactHashesAndRisks(t *testing.T) {
	body := strings.ReplaceAll(validMemoryIslandsFinalProductionReadinessAuditBody(), "## Command Log", "## Commands")
	body = strings.ReplaceAll(body, "## Artifact Hashes", "## Hashes")
	body = strings.ReplaceAll(body, "## Residual Risks", "## Risks")
	doc := writeMemoryIslandsFinalProductionReadinessAudit(t, body)
	err := verifyMemoryIslandsFinalProductionReadinessAudit([]string{doc})
	if err == nil {
		t.Fatalf("expected final Memory/Islands audit to reject missing command log/artifact hash/residual risk sections")
	}
	for _, want := range []string{"command log", "artifact hashes", "residual risks"} {
		if !strings.Contains(strings.ToLower(err.Error()), want) {
			t.Fatalf("expected %q in error, got %v", want, err)
		}
	}
}

func TestVerifyMemoryIslandsFinalProductionReadinessAuditRejectsBroadReadyClaim(t *testing.T) {
	doc := writeMemoryIslandsFinalProductionReadinessAudit(t, validMemoryIslandsFinalProductionReadinessAuditBody()+"\nMemory verdict: `PROD_READY_PROVEN`\n")
	err := verifyMemoryIslandsFinalProductionReadinessAudit([]string{doc})
	if err == nil {
		t.Fatalf("expected final Memory/Islands audit to reject broad ready claim")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "prod_ready_proven") {
		t.Fatalf("expected PROD_READY_PROVEN in error, got %v", err)
	}
}

func TestVerifyMemoryIslandsFinalProductionReadinessAuditAcceptsHonestScopedEvidence(t *testing.T) {
	doc := writeMemoryIslandsFinalProductionReadinessAudit(t, validMemoryIslandsFinalProductionReadinessAuditBody())
	if err := verifyMemoryIslandsFinalProductionReadinessAudit([]string{doc}); err != nil {
		t.Fatalf("verifyMemoryIslandsFinalProductionReadinessAudit: %v", err)
	}
}

func TestVerifyMemoryIslandsFinalActorBenchmarkHandoffRejectsActorProductionClaimWithoutGate(t *testing.T) {
	doc := writeMemoryIslandsFinalActorBenchmarkHandoff(t, validMemoryIslandsFinalActorBenchmarkHandoffBody()+"\nThe production actor runtime is ready now and the actor production gate passed.\n")
	err := verifyMemoryIslandsFinalActorBenchmarkHandoff([]string{doc})
	if err == nil {
		t.Fatalf("expected final actor handoff to reject production actor claim without gate evidence")
	}
	for _, want := range []string{"production actor runtime", "actor production gate passed"} {
		if !strings.Contains(strings.ToLower(err.Error()), want) {
			t.Fatalf("expected %q in error, got %v", want, err)
		}
	}
}

func TestVerifyMemoryIslandsFinalActorBenchmarkHandoffRejectsBenchmarkOverclaim(t *testing.T) {
	doc := writeMemoryIslandsFinalActorBenchmarkHandoff(t, validMemoryIslandsFinalActorBenchmarkHandoffBody()+"\nBenchmark phase may claim an official benchmark result and C++/Rust parity.\n")
	err := verifyMemoryIslandsFinalActorBenchmarkHandoff([]string{doc})
	if err == nil {
		t.Fatalf("expected final actor handoff to reject benchmark overclaim")
	}
	for _, want := range []string{"official benchmark", "c++/rust parity"} {
		if !strings.Contains(strings.ToLower(err.Error()), want) {
			t.Fatalf("expected %q in error, got %v", want, err)
		}
	}
}

func TestVerifyMemoryIslandsFinalActorBenchmarkHandoffAcceptsScopedPreconditions(t *testing.T) {
	doc := writeMemoryIslandsFinalActorBenchmarkHandoff(t, validMemoryIslandsFinalActorBenchmarkHandoffBody())
	if err := verifyMemoryIslandsFinalActorBenchmarkHandoff([]string{doc}); err != nil {
		t.Fatalf("verifyMemoryIslandsFinalActorBenchmarkHandoff: %v", err)
	}
}

func TestVerifyActorRuntimeFoundationDocsRejectsBroadActorProductionClaim(t *testing.T) {
	doc := writeActorRuntimeFoundationDoc(t, validActorRuntimeFoundationDocBody()+".\nThe full production actor runtime is ready now.\n")
	err := verifyActorRuntimeFoundationDocs([]string{doc}, validActorRuntimeFoundationFeatures())
	if err == nil {
		t.Fatalf("expected actor foundation docs to reject broad production actor claim")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "full production actor runtime") {
		t.Fatalf("expected full production actor runtime in error, got %v", err)
	}
}

func TestVerifyActorRuntimeFoundationDocsRejectsProdReadyProvenClaim(t *testing.T) {
	doc := writeActorRuntimeFoundationDoc(t, validActorRuntimeFoundationDocBody()+".\nActor foundation verdict: PROD_READY_PROVEN.\n")
	err := verifyActorRuntimeFoundationDocs([]string{doc}, validActorRuntimeFoundationFeatures())
	if err == nil {
		t.Fatalf("expected actor foundation docs to reject PROD_READY_PROVEN claim")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "prod_ready_proven") {
		t.Fatalf("expected PROD_READY_PROVEN in error, got %v", err)
	}
}

func TestVerifyActorRuntimeFoundationDocsAllowsProdReadyProvenNonClaim(t *testing.T) {
	doc := writeActorRuntimeFoundationDoc(t, validActorRuntimeFoundationDocBody()+"\n`PROD_READY_PROVEN`: `NOT_CLAIMED`.\n")
	if err := verifyActorRuntimeFoundationDocs([]string{doc}, validActorRuntimeFoundationFeatures()); err != nil {
		t.Fatalf("verifyActorRuntimeFoundationDocs: %v", err)
	}
}

func TestVerifyActorRuntimeFoundationDocsRejectsNonLinuxDistributedClaim(t *testing.T) {
	doc := writeActorRuntimeFoundationDoc(t, validActorRuntimeFoundationDocBody()+".\nWindows distributed actor runtime support is production-ready.\n")
	err := verifyActorRuntimeFoundationDocs([]string{doc}, validActorRuntimeFoundationFeatures())
	if err == nil {
		t.Fatalf("expected actor foundation docs to reject non-Linux distributed actor claim")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "windows distributed actor runtime") {
		t.Fatalf("expected Windows distributed actor runtime in error, got %v", err)
	}
}

func TestVerifyActorRuntimeFoundationDocsRejectsDistributedZeroCopyClaim(t *testing.T) {
	doc := writeActorRuntimeFoundationDoc(t, validActorRuntimeFoundationDocBody()+".\nDistributed zero-copy pointer transfer is supported.\n")
	err := verifyActorRuntimeFoundationDocs([]string{doc}, validActorRuntimeFoundationFeatures())
	if err == nil {
		t.Fatalf("expected actor foundation docs to reject distributed zero-copy claim")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "distributed zero-copy") {
		t.Fatalf("expected distributed zero-copy in error, got %v", err)
	}
}

func TestVerifyActorRuntimeFoundationDocsRejectsMissingDistributedTargetMatrix(t *testing.T) {
	doc := writeActorRuntimeFoundationDoc(t, strings.Replace(validActorRuntimeFoundationDocBody(), distributedRuntimeTargetMatrixDocBody(), "", 1))
	err := verifyActorRuntimeFoundationDocs([]string{doc}, validActorRuntimeFoundationFeatures())
	if err == nil {
		t.Fatalf("expected actor foundation docs to require distributed target matrix")
	}
	if !strings.Contains(err.Error(), "Distributed Runtime Target Matrix") {
		t.Fatalf("expected Distributed Runtime Target Matrix in error, got %v", err)
	}
}

func TestVerifyActorRuntimeFoundationDocsRejectsMissingBenchmarkNonClaim(t *testing.T) {
	doc := writeActorRuntimeFoundationDoc(t, strings.Replace(validActorRuntimeFoundationDocBody(), "no benchmark superiority, no C++/Rust parity, and no official benchmark claim", "", 1))
	err := verifyActorRuntimeFoundationDocs([]string{doc}, validActorRuntimeFoundationFeatures())
	if err == nil {
		t.Fatalf("expected actor foundation docs to require actor benchmark nonclaim")
	}
	if !strings.Contains(err.Error(), "no benchmark superiority") {
		t.Fatalf("expected actor benchmark nonclaim in error, got %v", err)
	}
}

func TestVerifyActorRuntimeFoundationDocsRejectsStaleManifestFeature(t *testing.T) {
	doc := writeActorRuntimeFoundationDoc(t, validActorRuntimeFoundationDocBody())
	features := validActorRuntimeFoundationFeatures()
	features[0].Stability = "current Linux-x64 runtime evidence without the final actor foundation gate"
	err := verifyActorRuntimeFoundationDocs([]string{doc}, features)
	if err == nil {
		t.Fatalf("expected actor foundation docs to reject stale manifest feature")
	}
	if !strings.Contains(err.Error(), "tetra.actor.production_foundation.v1") {
		t.Fatalf("expected production foundation schema in error, got %v", err)
	}
}

func TestVerifyActorRuntimeFoundationDocsAcceptsScopedGateEvidence(t *testing.T) {
	doc := writeActorRuntimeFoundationDoc(t, validActorRuntimeFoundationDocBody())
	if err := verifyActorRuntimeFoundationDocs([]string{doc}, validActorRuntimeFoundationFeatures()); err != nil {
		t.Fatalf("verifyActorRuntimeFoundationDocs: %v", err)
	}
}

func TestDefaultActorRuntimeFoundationDocPathsIncludeHistoricalFinalAudit(t *testing.T) {
	paths := defaultActorRuntimeFoundationDocPaths()
	want := filepath.FromSlash("docs/audits/actor-runtime-production-foundation-final.md")
	for _, path := range paths {
		if path == want {
			return
		}
	}
	t.Fatalf("defaultActorRuntimeFoundationDocPaths() missing %q: %#v", want, paths)
}

func TestVerifyRAMContractCompilerDocsRejectsIncompleteDocs(t *testing.T) {
	paths := writeRAMContractDocsSet(t, "RAM Contract Compiler\n")
	err := verifyRAMContractCompilerDocs(paths, []featureManifest{validVerifyDocsRAMContractFeature()})
	if err == nil {
		t.Fatalf("expected incomplete RAM contract docs failure")
	}
	for _, want := range []string{"tetra.ram-contract-report.v1", "ram-contract-linux-x64-smoke.sh"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in error, got %v", want, err)
		}
	}
}

func TestVerifyRAMContractCompilerDocsRejectsForbiddenClaim(t *testing.T) {
	paths := writeRAMContractDocsSet(t, validRAMContractDocsBody()+"\nRAM Contract Compiler proves zero heap for all programs.\n")
	err := verifyRAMContractCompilerDocs(paths, []featureManifest{validVerifyDocsRAMContractFeature()})
	if err == nil {
		t.Fatalf("expected RAM contract docs forbidden claim failure")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "zero heap for all programs") {
		t.Fatalf("expected zero heap claim in error, got %v", err)
	}
}

func TestVerifyRAMContractCompilerDocsRejectsUnsupportedValidatorFlag(t *testing.T) {
	paths := writeRAMContractDocsSet(t, validRAMContractDocsBody()+"\ngo run ./tools/cmd/validate-ram-contract-release --report reports/ram-contract-release\n")
	err := verifyRAMContractCompilerDocs(paths, []featureManifest{validVerifyDocsRAMContractFeature()})
	if err == nil {
		t.Fatalf("expected unsupported RAM contract validator flag failure")
	}
	if !strings.Contains(err.Error(), "validate-ram-contract-release --report") {
		t.Fatalf("expected unsupported flag in error, got %v", err)
	}
}

func TestVerifyRAMContractCompilerDocsRejectsStaleReadinessHead(t *testing.T) {
	if _, ok := currentGitHeadForDocs(); !ok {
		t.Skip("git head unavailable")
	}
	paths := writeRAMContractDocsSet(t, validRAMContractDocsBody())
	stale := "0000000000000000000000000000000000000000"
	body := validRAMContractDocsBody() + "\nGit head: " + stale + "\n"
	if err := os.WriteFile(paths.Readiness, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	err := verifyRAMContractCompilerDocs(paths, []featureManifest{validVerifyDocsRAMContractFeature()})
	if err == nil {
		t.Fatalf("expected stale readiness git head failure")
	}
	if !strings.Contains(err.Error(), "stale readiness git head "+stale) {
		t.Fatalf("expected stale head in error, got %v", err)
	}
}

func TestVerifyRAMContractCompilerDocsAcceptsDirectParentReadinessHead(t *testing.T) {
	parent, ok := currentGitParentForDocs()
	if !ok {
		t.Skip("git parent unavailable")
	}
	paths := writeRAMContractDocsSet(t, validRAMContractDocsBody())
	body := validRAMContractDocsBody() + "\nGit head: " + parent + "\n"
	if err := os.WriteFile(paths.Readiness, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := verifyRAMContractCompilerDocs(paths, []featureManifest{validVerifyDocsRAMContractFeature()}); err != nil {
		t.Fatalf("verifyRAMContractCompilerDocs accepted direct parent evidence head: %v", err)
	}
}

func TestVerifyRAMContractCompilerDocsAcceptsScopedEvidence(t *testing.T) {
	paths := writeRAMContractDocsSet(t, validRAMContractDocsBody())
	if err := verifyRAMContractCompilerDocs(paths, []featureManifest{validVerifyDocsRAMContractFeature()}); err != nil {
		t.Fatalf("verifyRAMContractCompilerDocs: %v", err)
	}
}

func writeMemoryIslandsSurfaceReleaseDoc(t *testing.T, body string) string {
	t.Helper()
	doc := filepath.Join(t.TempDir(), "memory_islands_surface_scope.md")
	if err := os.WriteFile(doc, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return doc
}

func writeRAMContractDocsSet(t *testing.T, body string) ramContractCompilerDocPaths {
	t.Helper()
	dir := t.TempDir()
	paths := ramContractCompilerDocPaths{
		Design:    filepath.Join(dir, "ram_contract_compiler.md"),
		Spec:      filepath.Join(dir, "ram_contract_report_schema.md"),
		User:      filepath.Join(dir, "ram_contracts.md"),
		Readiness: filepath.Join(dir, "ram-contract-compiler-readiness.md"),
		Handoff:   filepath.Join(dir, "ram-contract-compiler-handoff.md"),
	}
	for _, path := range []string{paths.Design, paths.Spec, paths.User, paths.Readiness, paths.Handoff} {
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return paths
}

func validRAMContractDocsBody() string {
	return strings.Join([]string{
		"RAM Contract Compiler",
		"RAM Contract Report Schema",
		"Using RAM Contracts",
		"RAM Contract Compiler Readiness Audit",
		"RAM Contract Compiler Handoff",
		"tetra.ram-contract-report.v1",
		"tetra.memory-grade-report.v1",
		"tetra.proof-store-summary.v1",
		"tetra.validation-pipeline-coverage.v1",
		"tetra.ram-blockers.v1",
		"compiler-owned facts",
		"MemoryFactGraph",
		"AllocPlan",
		"ProofStore",
		"heap-blockers.json",
		"copy-blockers.json",
		"ram-contract-fuzz-oracle.json",
		"ram-contract-report.json",
		"memory-grade-report.json",
		"proof-store-summary.json",
		"validation-pipeline-coverage.json",
		"TETRA4100",
		"--emit-ram-contract-report",
		"--fail-if-heap",
		"--fail-if-copy",
		"--fail-if-unbounded",
		"--memory-budget",
		"--ram-contract",
		"validate-ram-contract-report",
		"validate-memory-grade-report",
		"validate-proof-store-summary",
		"validate-validation-pipeline-coverage",
		"validate-heap-blockers",
		"validate-copy-blockers",
		"validate-ram-contract-fuzz-oracle",
		"validate-ram-contract-release",
		"scripts/release/post_v0_4/ram-contract-linux-x64-smoke.sh",
		".github/workflows/ci.yml",
		".github/workflows/release-packages.yml",
		"go test -buildvcs=false",
		"go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json",
		"git diff --check",
		"reports/ram-contract-release",
		"Git head:",
		"Working tree:",
		"dirty working tree",
		"Verdict: `SCOPED_READY`",
		"Release gate:",
		"CI job:",
		"ram-contract-release-readiness-linux",
		"Package workflow:",
		"ram-contract-linux-x64",
		"Required artifacts:",
		"no zero heap for all programs claim",
		"no zero-copy for all programs claim",
		"no full formal proof claim",
		"no all-target RAM parity claim",
	}, "\n")
}

func validMemoryIslandsSurfaceReleaseDocBody() string {
	return strings.Join([]string{
		"Memory/Islands/Surface scoped release truth",
		"scripts/release/post_v0_4/memory-islands-surface-production-gate.sh",
		"tools/cmd/validate-memory-islands-surface-production",
		"tools/cmd/validate-island-proof",
		"--islands-debug",
		"islands-debug-smoke.json",
		"island-proof-verifier.json",
		"island-proof-fuzz-summary.json",
		"memory-islands-surface-production-manifest.json",
		"artifact-hashes.json",
		"leak/resource finalization evidence",
		"surface-v1-linux-web",
		"no Memory 100% claim",
		"no arbitrary unsafe external pointer safety",
		"no full formal proof",
		"no full target parity",
		"no all-target Surface claim",
		"no production object memory claim",
		"no production persistent memory claim",
		"not a clean release-candidate checkout claim",
	}, "\n")
}

func writeFinalMemoryIslandsSurfaceProductionAudit(t *testing.T, body string) string {
	t.Helper()
	doc := filepath.Join(t.TempDir(), "memory-islands-surface-final-production-readiness.md")
	if err := os.WriteFile(doc, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return doc
}

func writeMemoryIslandsFinalProductionReadinessAudit(t *testing.T, body string) string {
	t.Helper()
	doc := filepath.Join(t.TempDir(), "memory-islands-final-production-readiness.md")
	if err := os.WriteFile(doc, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return doc
}

func writeMemoryIslandsFinalActorBenchmarkHandoff(t *testing.T, body string) string {
	t.Helper()
	doc := filepath.Join(t.TempDir(), "memory-islands-final-production-handoff.md")
	if err := os.WriteFile(doc, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return doc
}

func writeActorRuntimeFoundationDoc(t *testing.T, body string) string {
	t.Helper()
	doc := filepath.Join(t.TempDir(), "actor-runtime-foundation.md")
	if err := os.WriteFile(doc, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return doc
}

func validActorRuntimeFoundationDocBody() string {
	return strings.Join([]string{
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
		distributedRuntimeTargetMatrixDocBody(),
	}, "\n")
}

func distributedRuntimeTargetMatrixDocBody() string {
	return strings.Join([]string{
		"Distributed Runtime Target Matrix",
		"| Target | Distributed actor runtime status | Current evidence | Promotion requirement |",
		"|---|---|---|---|",
		"`linux-x64` | current scoped | executable `tetra.actors.distributed-runtime.v1` smoke plus actor foundation gate | keep same-commit distributed smoke, artifact hashes, and foundation validator green |",
		"`macos-x64` | unsupported / nonclaim | no distributed actor symbols; actor net pump is no-op | add target runtime, smoke, validator, docs, and package gate before any support claim |",
		"`windows-x64` | unsupported / nonclaim | no distributed actor symbols; actor net pump is no-op | add target runtime, smoke, validator, docs, and package gate before any support claim |",
		"`wasm32-wasi` | unsupported / nonclaim | no distributed actor runtime gate | add target runtime, smoke, validator, docs, and package gate before any support claim |",
		"`wasm32-web` | unsupported / nonclaim | no distributed actor runtime gate | add target runtime, smoke, validator, docs, and package gate before any support claim |",
	}, "\n")
}

func validActorRuntimeFoundationFeatures() []featureManifest {
	return []featureManifest{
		{
			ID:        "actors.distributed-runtime",
			Name:      "Distributed actor runtime for Linux x64",
			Status:    "current",
			Since:     "v0.4.0",
			Scope:     "production Linux-x64 distributed actor runtime path with the scoped actor runtime foundation gate",
			Stability: "current Linux-x64 runtime/lowering slice with executable tetra.actors.distributed-runtime.v1 smoke evidence, tetra.actor.production_foundation.v1 gate evidence, actor-runtime-foundation-linux-x64-gate.sh, and strict nonclaims for non-Linux distributed runtime, distributed zero-copy, cluster membership, reconnect/retry production, and formal race proof",
			Docs: []string{
				"docs/spec/current_supported_surface.md",
				"docs/spec/actors.md",
				"docs/user/async_actors_guide.md",
				"docs/design/actor_region_transfer.md",
				"docs/audits/actor-runtime-production-boundary-v1.md",
				"docs/checklists/actors_linux_smoke.md",
				"docs/checklists/actors_platform_smoke.md",
			},
		},
	}
}

func validFinalMemoryIslandsSurfaceProductionAuditBody() string {
	return strings.Join([]string{
		"# Memory/Islands/Surface Final Production Readiness Audit",
		"Git head: e2c19b8ee276158f8eb2c54cf61e11bd84952893",
		"Working tree: dirty working tree evidence, not a clean release-candidate checkout claim",
		"Memory verdict: `PROD_STABLE_SCOPED`",
		"Islands verdict: `PROD_STABLE_SCOPED`",
		"Surface verdict: `PROD_STABLE_SCOPED` for `surface-v1-linux-web`",
		"Integrated verdict: `PROD_STABLE_SCOPED`",
		"Commands: go test -buildvcs=false ./tools/cmd/verify-docs -run 'Final|Production|Audit|Overclaim' -count=1",
		"Commands: git diff --check",
		"Commands: git status --short",
		"Commands: go run -buildvcs=false ./tools/cmd/verify-docs --manifest docs/generated/manifest.json",
		"Artifacts: reports/mis-ideal/P13/integrated",
		"Artifacts: reports/mis-ideal/P15/docs-manifest-overclaim.md",
		"Hashes: sha256 a7f3da4cab2494dda804bd3d4e5d00d7ccc403255b01eb07461b0bf126151953",
		"Changed Files: .github/workflows/ci.yml",
		"Changed Files: docs/generated/manifest.json",
		"Changed Files: docs/release/memory_islands_surface_scope.md",
		"Changed Files: scripts/release/post_v0_4/memory-islands-surface-production-gate.sh",
		"Changed Files: tools/cmd/validate-island-proof",
		"Changed Files: tools/cmd/validate-memory-islands-surface-production",
		"Residual Risks: remote GitHub Actions run was not executed",
		"Residual Risks: tools/cmd/dump-project optional broad scriptstest fixture still fails outside P14",
		"Residual Risks: tools/validators/postv04prod optional broad scriptstest fixture still fails outside P14",
		"Nonclaims: no Memory 100% claim",
		"Nonclaims: no arbitrary unsafe external pointer safety",
		"Nonclaims: no full formal proof",
		"Nonclaims: no full target parity",
		"Nonclaims: no all-target Surface claim",
	}, "\n")
}

func validMemoryIslandsFinalProductionReadinessAuditBody() string {
	return strings.Join([]string{
		"# Memory/Islands Final Production Readiness Audit",
		"Git head: e2c19b8ee276158f8eb2c54cf61e11bd84952893",
		"Working tree: dirty working tree evidence, not a clean release-candidate checkout claim.",
		"Memory verdict: `PROD_STABLE_SCOPED`",
		"Islands verdict: `PROD_STABLE_SCOPED`",
		"Integrated gate verdict: `PROD_STABLE_SCOPED`",
		"## Scope",
		"Memory/Islands scope: linux-x64 MemoryFactGraph-backed reports and island proof validation.",
		"Integrated gate scope: Memory/Islands plus existing scoped Surface dependency evidence.",
		"## Command Log",
		"`git status --short`",
		"`git rev-parse HEAD`",
		"`go test -buildvcs=false ./compiler/... ./cli/... ./tools/... -count=1`",
		"`go test -race -buildvcs=false ./compiler/internal/islandkernel ./compiler/internal/memoryfacts ./compiler/internal/memorymodel ./compiler/internal/semantics ./compiler/internal/plir ./compiler/internal/validation ./cli/internal/actornet -count=1`",
		"`bash scripts/release/post_v0_4/memory-production-linux-x64-smoke.sh --report-dir reports/memory-islands-ideal/final/memory-production`",
		"`bash scripts/release/post_v0_4/memory-islands-surface-production-gate.sh --report-dir reports/memory-islands-ideal/final/integrated`",
		"`go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`",
		"`go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`",
		"`git diff --check`",
		"## Artifact Log",
		"`reports/memory-islands-ideal/final/memory-production`",
		"`reports/memory-islands-ideal/final/integrated`",
		"`reports/memory-islands-ideal/final/artifact-sha256.txt`",
		"## Artifact Hashes",
		"`0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef`",
		"## Residual Risks",
		"- dirty working tree blocks clean release-candidate proof until committed or otherwise resolved.",
		"- remote GitHub Actions evidence is not present.",
		"## Nonclaims",
		"- no Memory 100% claim",
		"- no arbitrary unsafe external pointer safety",
		"- no full formal proof",
		"- no full target parity",
		"- no production actor runtime",
		"- no official benchmark result",
		"- not a clean release-candidate checkout claim",
	}, "\n")
}

func validMemoryIslandsFinalActorBenchmarkHandoffBody() string {
	return strings.Join([]string{
		"# Memory/Islands Final Production Audit and Actor Handoff",
		"Final verdict: `PROD_STABLE_SCOPED`",
		"Memory/Islands baseline: `docs/audits/memory-islands-final-production-readiness.md` and `reports/memory-islands-ideal/final/artifact-sha256.txt`.",
		"Actor handoff readiness: actor phase may start as a separate Actor Runtime Production Foundation plan.",
		"Actor runtime production status: not started in this plan.",
		"Actor phase preconditions:",
		"- production actor gate must prove scheduler, mailbox backpressure, message exhaustion/reclamation, race-safety, cross-target distributed runtime gates, structured concurrency, and fake-evidence rejection.",
		"- `docs/audits/actor-runtime-production-boundary-v1.md` remains the actor production boundary.",
		"- `MEMISL-P10` memory-boundary handoff evidence is an input, not actor runtime completion.",
		"Benchmark preconditions:",
		"- benchmark phase may start only as Tier 0/Tier 1 preparation until measured evidence exists.",
		"- no official benchmark result",
		"- no performance superiority",
		"- no C++/Rust parity",
		"- no measured speed comparison",
		"Nonclaims:",
		"- no production actor runtime",
		"- no actor production gate passed",
		"- no official benchmark result",
		"- no performance superiority",
		"- no `PROD_READY_PROVEN` claim",
	}, "\n")
}

func writeSurfaceReleaseDoc(t *testing.T, body string) string {
	t.Helper()
	doc := filepath.Join(t.TempDir(), "surface_v1.md")
	if err := os.WriteFile(doc, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return doc
}

func TestExtractTetraDoctestsParsesCommentFence(t *testing.T) {
	doc := strings.Join([]string{
		"// Stable module docs.",
		"// ```tetra doctest",
		"// func demo() -> Int:",
		"//     return 42",
		"// ```",
	}, "\n")
	blocks, err := extractTetraDoctests(doc)
	if err != nil {
		t.Fatalf("extractTetraDoctests: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 doctest block, got %d", len(blocks))
	}
	if !strings.Contains(blocks[0], "func demo() -> Int:") {
		t.Fatalf("unexpected doctest block: %q", blocks[0])
	}
}

func TestVerifyRequiredDoctestBlocksRejectsMissingDoctest(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "module.tetra")
	if err := os.WriteFile(doc, []byte(strings.Join([]string{
		"// Stable v0.5 module docs.",
		"module lib.core.sample",
		"",
		"func add(a: Int, b: Int) -> Int:",
		"    return a + b",
	}, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	err := verifyRequiredDoctestBlocks([]string{doc})
	if err == nil {
		t.Fatalf("expected missing doctest failure")
	}
	if !strings.Contains(err.Error(), "missing tetra doctest block") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifyRequiredDoctestBlocksAcceptsCommentFenceDoctest(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "module.tetra")
	if err := os.WriteFile(doc, []byte(strings.Join([]string{
		"// Stable v0.5 module docs.",
		"// ```tetra doctest",
		"// func demo() -> Int:",
		"//     return 0",
		"// ```",
		"module lib.core.sample",
		"",
		"func add(a: Int, b: Int) -> Int:",
		"    return a + b",
	}, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := verifyRequiredDoctestBlocks([]string{doc}); err != nil {
		t.Fatalf("verifyRequiredDoctestBlocks: %v", err)
	}
}

func TestVerifyStableModuleDoctestCoverageRejectsPlaceholder(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "memory.tetra")
	if err := os.WriteFile(doc, []byte(strings.Join([]string{
		"// Stable docs.",
		"// Effects: mem",
		"// ```tetra doctest",
		"// func memory_doctest() -> Int:",
		"//     return 0",
		"// ```",
		"module lib.core.memory",
		"",
		"func memset_u8(dst: ptr, v: UInt8, n: Int, mem: cap.mem) -> Int",
		"uses mem:",
		"    return 0",
	}, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}

	err := verifyStableModuleDoctestCoverage([]string{doc})
	if err == nil {
		t.Fatalf("expected placeholder doctest failure")
	}
	if !strings.Contains(err.Error(), "doctest does not reference lib.core.memory") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifyStableModuleDoctestCoverageAcceptsModuleReference(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "memory.tetra")
	if err := os.WriteFile(doc, []byte(strings.Join([]string{
		"// Stable docs.",
		"// Effects: mem",
		"// ```tetra doctest",
		"// func memory_doctest() -> Int:",
		"//     return lib.core.memory.memcpy_status()",
		"// ```",
		"module lib.core.memory",
		"",
		"func memcpy_status() -> Int:",
		"    return 0",
	}, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := verifyStableModuleDoctestCoverage([]string{doc}); err != nil {
		t.Fatalf("verifyStableModuleDoctestCoverage: %v", err)
	}
}

func TestVerifyStableModuleEffectsMetadataRejectsMissingMetadata(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "module.tetra")
	if err := os.WriteFile(doc, []byte(strings.Join([]string{
		"// Stable docs.",
		"module lib.core.sample",
		"",
		"func id(x: Int) -> Int:",
		"    return x",
	}, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	err := verifyStableModuleEffectsMetadata([]string{doc})
	if err == nil {
		t.Fatalf("expected missing effects metadata failure")
	}
	if !strings.Contains(err.Error(), "missing effects metadata") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifyStableModuleEffectsMetadataAcceptsDeclaredEffects(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "module.tetra")
	if err := os.WriteFile(doc, []byte(strings.Join([]string{
		"// Stable docs.",
		"// Effects: none",
		"module lib.core.sample",
		"",
		"func id(x: Int) -> Int:",
		"    return x",
	}, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := verifyStableModuleEffectsMetadata([]string{doc}); err != nil {
		t.Fatalf("verifyStableModuleEffectsMetadata: %v", err)
	}
}

func TestVerifyStableModuleEffectsMetadataRejectsMismatchedMetadata(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "module.tetra")
	if err := os.WriteFile(doc, []byte(strings.Join([]string{
		"// Stable docs.",
		"// Effects: none",
		"module lib.core.sample",
		"",
		"func len_i32(values: []i32) -> Int",
		"uses mem:",
		"    var count: Int = 0",
		"    for value in values:",
		"        count = count + 1",
		"    return count",
	}, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	err := verifyStableModuleEffectsMetadata([]string{doc})
	if err == nil {
		t.Fatalf("expected mismatched effects metadata failure")
	}
	if !strings.Contains(err.Error(), "effects metadata mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifyStdlibGuideRejectsMismatchedStableEffects(t *testing.T) {
	dir := t.TempDir()
	coreDir := filepath.Join(dir, "lib", "core")
	if err := os.MkdirAll(coreDir, 0o755); err != nil {
		t.Fatal(err)
	}
	modulePath := filepath.Join(coreDir, "strings.tetra")
	if err := os.WriteFile(modulePath, []byte(strings.Join([]string{
		"// Stable docs.",
		"// Effects: none",
		"// ```tetra doctest",
		"// func strings_doctest() -> Int:",
		"//     return lib.core.strings.ascii_len(\"x\")",
		"// ```",
		"module lib.core.strings",
		"",
		"func ascii_len(text: String) -> Int:",
		"    return 0",
	}, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	guidePath := filepath.Join(dir, "standard_library_guide.md")
	if err := os.WriteFile(guidePath, []byte(strings.Join([]string{
		"# Standard Library Guide",
		"",
		"| Need | Import | Example | Effects |",
		"| --- | --- | --- | --- |",
		"| String helpers | `import lib.core.strings as strings` | `examples/core_strings_smoke.tetra` | `mem` |",
	}, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}

	err := verifyStdlibGuide(guidePath, []string{modulePath}, nil)
	if err == nil {
		t.Fatalf("expected guide effects mismatch")
	}
	if !strings.Contains(err.Error(), "lib.core.strings effects mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifyStdlibGuideAcceptsStableAndExperimentalMirrors(t *testing.T) {
	dir := t.TempDir()
	coreDir := filepath.Join(dir, "lib", "core")
	experimentalDir := filepath.Join(dir, "lib", "experimental")
	if err := os.MkdirAll(coreDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(experimentalDir, 0o755); err != nil {
		t.Fatal(err)
	}
	corePath := filepath.Join(coreDir, "strings.tetra")
	if err := os.WriteFile(corePath, []byte(strings.Join([]string{
		"// Stable docs.",
		"// Effects: none",
		"// ```tetra doctest",
		"// func strings_doctest() -> Int:",
		"//     return lib.core.strings.ascii_len(\"x\")",
		"// ```",
		"module lib.core.strings",
		"",
		"func ascii_len(text: String) -> Int:",
		"    return 0",
	}, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	experimentalPath := filepath.Join(experimentalDir, "strings.tetra")
	if err := os.WriteFile(experimentalPath, []byte(strings.Join([]string{
		"// Experimental strings helpers (no stability guarantees).",
		"//",
		"// Promotion note: v1 stable callers should use lib.core.strings directly.",
		"module lib.experimental.strings",
		"",
		"import lib.core.strings as stable_strings",
		"",
		"func ascii_len(text: String) -> Int:",
		"    return stable_strings.ascii_len(text)",
	}, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	guidePath := filepath.Join(dir, "standard_library_guide.md")
	if err := os.WriteFile(guidePath, []byte(strings.Join([]string{
		"# Standard Library Guide",
		"",
		"| Need | Import | Example | Effects |",
		"| --- | --- | --- | --- |",
		"| String helpers | `import lib.core.strings as strings` | `examples/core_strings_smoke.tetra` | none |",
		"",
		"## Experimental Mirrors",
		"",
		"| Experimental import | Stable replacement | Status |",
		"| --- | --- | --- |",
		"| `import lib.experimental.strings as strings` | `import lib.core.strings as strings` | Experimental mirror; no stability guarantees. |",
	}, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := verifyStdlibGuide(guidePath, []string{corePath}, []string{experimentalPath}); err != nil {
		t.Fatalf("verifyStdlibGuide: %v", err)
	}
}

func TestVerifyExperimentalModuleMirrorsRejectsMissingPromotionNote(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lib", "experimental", "math.tetra")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.Join([]string{
		"// Experimental math helpers (no stability guarantees).",
		"module lib.experimental.math",
		"",
		"import lib.core.math as stable_math",
		"",
		"func add_i32(a: Int, b: Int) -> Int:",
		"    return stable_math.add_i32(a, b)",
	}, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}

	err := verifyExperimentalModuleMirrors([]string{path})
	if err == nil {
		t.Fatalf("expected missing promotion note failure")
	}
	if !strings.Contains(err.Error(), "missing promotion note") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifyStableModuleExamplesRejectsMissingExampleFile(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "sample.tetra")
	if err := os.WriteFile(doc, []byte(strings.Join([]string{
		"// Stable docs.",
		"// Effects: none",
		"module lib.core.sample",
		"",
		"func id(x: Int) -> Int:",
		"    return x",
	}, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	err := verifyStableModuleExamples([]string{doc})
	if err == nil {
		t.Fatalf("expected missing stable module example failure")
	}
	if !strings.Contains(err.Error(), "missing stable module example") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifyStdlibModulePathsRejectsMismatchedCoreModule(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "core", "math.tetra")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.Join([]string{
		"// Stable docs.",
		"// Effects: none",
		"module lib.experimental.math",
		"",
		"func add(a: Int, b: Int) -> Int:",
		"    return a + b",
	}, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}

	err := verifyStdlibModulePaths([]string{path}, nil)
	if err == nil {
		t.Fatalf("expected mismatched core module failure")
	}
	if !strings.Contains(err.Error(), "expected module lib.core.math") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifyStdlibModulePathsRejectsStableVersionSuffix(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "core", "math_v2.tetra")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.Join([]string{
		"// Stable docs.",
		"// Effects: none",
		"module lib.core.math_v2",
		"",
		"func add(a: Int, b: Int) -> Int:",
		"    return a + b",
	}, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}

	err := verifyStdlibModulePaths([]string{path}, nil)
	if err == nil {
		t.Fatalf("expected stable version suffix failure")
	}
	if !strings.Contains(err.Error(), "stable module name must not contain version suffix") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifyStableExamplesRejectExperimentalImports(t *testing.T) {
	dir := t.TempDir()
	example := filepath.Join(dir, "examples", "core_math_smoke.tetra")
	if err := os.MkdirAll(filepath.Dir(example), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(example, []byte(strings.Join([]string{
		"import lib.experimental.math as math",
		"",
		"func main() -> Int:",
		"    return math.add(40, 2)",
	}, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}

	err := verifyStableExamplesDoNotImportExperimental([]string{example})
	if err == nil {
		t.Fatalf("expected experimental import failure")
	}
	if !strings.Contains(err.Error(), "stable example imports experimental module") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifyEpic14ExampleIndexAcceptsRequiredCoverage(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "examples_index.md")
	examples := []string{
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
	headings := []string{
		"## Epic 14 Verification Commands",
		"## Troubleshooting Notes (Epic 14)",
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

	lines := []string{
		"# Examples Index",
		"",
		"| Example | Purpose | Target group | Expected behavior |",
		"| --- | --- | --- | --- |",
	}
	for _, example := range examples {
		lines = append(lines, "| `"+example+"` | test entry | native | exits 0 |")
	}
	for _, heading := range headings {
		lines = append(lines, "", heading, "", "unsupported profile note", "regression note")
	}

	if err := os.WriteFile(indexPath, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := verifyEpic14ExampleIndex(indexPath); err != nil {
		t.Fatalf("verifyEpic14ExampleIndex: %v", err)
	}
}

func TestVerifyEpic14ExampleIndexRejectsMissingGenericStructEntry(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "examples_index.md")
	examples := []string{
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
	headings := []string{
		"## Epic 14 Verification Commands",
		"## Troubleshooting Notes (Epic 14)",
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

	lines := []string{
		"# Examples Index",
		"",
		"| Example | Purpose | Target group | Expected behavior |",
		"| --- | --- | --- | --- |",
	}
	for _, example := range examples {
		lines = append(lines, "| `"+example+"` | test entry | native | exits 0 |")
	}
	for _, heading := range headings {
		lines = append(lines, "", heading, "", "unsupported profile note", "regression note")
	}

	if err := os.WriteFile(indexPath, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	err := verifyEpic14ExampleIndex(indexPath)
	if err == nil {
		t.Fatalf("expected missing generic struct coverage failure")
	}
	if !strings.Contains(err.Error(), "examples/generic_struct_smoke.tetra") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifyEpic14ExampleIndexRejectsMissingPrimaryT4ProjectEntry(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "examples_index.md")
	examples := []string{
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
		"examples/projects/dogfood_wasi/src/main.tetra",
		"examples/projects/dogfood_web_ui/src/main.tetra",
		"examples/projects/dogfood_cli/src/main.tetra",
		"examples/projects/dogfood_actor_task/src/main.tetra",
		"examples/projects/eco_dogfood/src/main.tetra",
	}
	headings := []string{
		"## Epic 14 Verification Commands",
		"## Troubleshooting Notes (Epic 14)",
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
	lines := []string{
		"# Examples Index",
		"",
		"| Example | Purpose | Target group | Expected behavior |",
		"| --- | --- | --- | --- |",
	}
	for _, example := range examples {
		lines = append(lines, "| `"+example+"` | test entry | native | exits 0 |")
	}
	for _, heading := range headings {
		lines = append(lines, "", heading, "", "unsupported profile note", "regression note")
	}
	if err := os.WriteFile(indexPath, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	err := verifyEpic14ExampleIndex(indexPath)
	if err == nil {
		t.Fatalf("expected missing primary .t4 project coverage failure")
	}
	if !strings.Contains(err.Error(), "examples/projects/hello_t4/src/main.t4") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifyEpic14ExampleIndexRejectsMissingEntry(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "examples_index.md")
	body := strings.Join([]string{
		"# Examples Index",
		"",
		"| Example | Purpose | Target group | Expected behavior |",
		"| --- | --- | --- | --- |",
		"| `examples/flow_hello.tetra` | test entry | native | exits 0 |",
		"## Epic 14 Verification Commands",
		"## Troubleshooting Notes (Epic 14)",
		"### Basic language examples (`V020-0701..0705`)",
		"unsupported regression note",
		"### Control-flow examples (`V020-0706..0710`)",
		"unsupported regression note",
		"### Const and assignment examples (`V020-0711..0715`)",
		"unsupported regression note",
		"### Enum/match examples (`V020-0716..0720`)",
		"unsupported regression note",
		"### Optional/error examples (`V020-0721..0725`)",
		"unsupported regression note",
		"### Generic/protocol/extension examples (`V020-0726..0730`)",
		"unsupported regression note",
		"### Safety/runtime examples (`V020-0731..0735`)",
		"unsupported regression note",
		"### Memory/capability examples (`V020-0736..0740`)",
		"unsupported regression note",
		"### UI/WASM examples (`V020-0741..0745`)",
		"unsupported regression note",
		"### Project dogfood examples (`V020-0746..0750`)",
		"unsupported regression note",
	}, "\n")
	if err := os.WriteFile(indexPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	err := verifyEpic14ExampleIndex(indexPath)
	if err == nil {
		t.Fatalf("expected Epic 14 missing coverage failure")
	}
	if !strings.Contains(err.Error(), "examples/hello.tetra") {
		t.Fatalf("unexpected error: %v", err)
	}
}
