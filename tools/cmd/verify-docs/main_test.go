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
