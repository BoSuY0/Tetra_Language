package verifydocs

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type memoryProductionContractDocPaths struct {
	RuntimeABI             string
	Ownership              string
	Unsafe                 string
	Capabilities           string
	Stdlib                 string
	StdlibGuide            string
	CoreMemory             string
	TargetCapabilityMatrix string
	MemoryCostModel        string
	MemoryFuzzOracle       string
	MemoryProductionFinal  string
	MemoryProductionMap    string
	MemoryProductionClaims string
}

type memoryProductionContractRequirement struct {
	Name     string
	Path     string
	Required []string
}

func defaultMemoryProductionContractDocPaths() memoryProductionContractDocPaths {
	return memoryProductionContractDocPaths{
		RuntimeABI:   filepath.FromSlash("docs/spec/runtime/runtime_abi.md"),
		Ownership:    filepath.FromSlash("docs/spec/runtime/ownership_v1.md"),
		Unsafe:       filepath.FromSlash("docs/spec/runtime/unsafe.md"),
		Capabilities: filepath.FromSlash("docs/spec/runtime/capabilities.md"),
		Stdlib:       filepath.FromSlash("docs/spec/standard_library/stdlib.md"),
		StdlibGuide:  filepath.FromSlash("docs/user/platform/standard_library_guide.md"),
		CoreMemory:   filepath.FromSlash("lib/core/memory/memory.tetra"),
		TargetCapabilityMatrix: filepath.FromSlash(
			"docs/audits/memory/islands/memory-target-capability-matrix.md",
		),
		MemoryCostModel: filepath.FromSlash("docs/design/memory/memory_cost_model.md"),
		MemoryFuzzOracle: filepath.FromSlash(
			"docs/audits/memory/islands/memory-fuzz-oracle-v1.md",
		),
		MemoryProductionFinal: filepath.FromSlash(
			"docs/audits/memory/production/memory-production-core-v1-final.md",
		),
		MemoryProductionMap: filepath.FromSlash(
			"docs/audits/memory/production/memory-production-core-v1-artifact-map.md",
		),
		MemoryProductionClaims: filepath.FromSlash(
			"docs/audits/memory/production/memory-production-core-v1-nonclaims.md",
		),
	}
}

func memoryProductionContractRequirements(
	paths memoryProductionContractDocPaths,
) []memoryProductionContractRequirement {
	return []memoryProductionContractRequirement{
		{
			Name: "runtime ABI",
			Path: paths.RuntimeABI,
			Required: []string{
				"Linux-x64 Memory Production ABI",
				"`core.alloc_bytes(size: i32) -> ptr`",
				"`core.cap_mem() -> cap.mem`",
				"`core.ptr_add(ptr, offset: i32, mem: cap.mem) -> ptr`",
				"`core.load_u8(ptr, mem: cap.mem) -> u8`",
				"`core.store_u8(ptr, value: u8, mem: cap.mem) -> u8`",
				"invalid allocation sizes",
				"allocator failure semantics",
				"runtime bounds diagnostics",
				"negative `core.ptr_add` offsets",
				"allocation-base `core.ptr_add` upper bounds",
				"allocation-base `core.store_i32` width bounds",
				"allocation-base `core.store_ptr` width bounds",
				"negative `memcpy_u8` and `memset_u8` lengths",
				"no cross-target memory production claim",
			},
		},
		{
			Name: "ownership",
			Path: paths.Ownership,
			Required: []string{
				"Memory Production Extension",
				"heap, slices, structs, closures",
				"borrow escape",
				"actor/task transfer",
				"conservative rejection",
				"`TETRA2101`",
				"`TETRA2102`",
			},
		},
		{
			Name: "unsafe",
			Path: paths.Unsafe,
			Required: []string{
				"Memory Production Contract Boundary",
				"`cap.mem` authorizes the raw operation",
				"does not prove pointer validity or bounds",
				"runtime bounds diagnostics",
				"negative `core.ptr_add` offsets",
				"allocation-base `core.ptr_add` upper bounds",
				"allocation-base `core.store_i32` width bounds",
				"allocation-base `core.store_ptr` width bounds",
				"`memcpy_u8`",
				"`memset_u8`",
				"negative `memcpy_u8` and `memset_u8` lengths",
				"invalid allocation sizes",
			},
		},
		{
			Name: "capabilities",
			Path: paths.Capabilities,
			Required: []string{
				"Memory Production Boundary",
				"`cap.mem` is permission, not provenance",
				"raw memory access",
				"runtime bounds diagnostics",
				"pointer validity",
			},
		},
		{
			Name: "stdlib",
			Path: paths.Stdlib,
			Required: []string{
				"`lib.core.memory` Production Boundary",
				"`memcpy_u8`",
				"`memset_u8`",
				"does not allocate",
				"does not perform bounds checks",
				"Memory Production Core",
			},
		},
		{
			Name: "stdlib guide",
			Path: paths.StdlibGuide,
			Required: []string{
				"Writing Raw Memory Safely",
				"`cap.mem` is not ownership",
				"check sizes before calling",
				"Memory Production Core",
				"runtime bounds diagnostics",
				"negative `core.ptr_add` offsets",
				"allocation-base `core.ptr_add` upper bounds",
				"allocation-base `core.store_i32` width bounds",
				"allocation-base `core.store_ptr` width bounds",
				"negative `memcpy_u8` and `memset_u8` lengths",
			},
		},
		{
			Name: "core memory module",
			Path: paths.CoreMemory,
			Required: []string{
				"Memory Production Core boundary",
				"`cap.mem` authorizes raw byte access",
				"caller owns pointer validity and bounds",
				"func memset_u8",
				"func memcpy_u8",
			},
		},
		{
			Name: "target capability matrix",
			Path: paths.TargetCapabilityMatrix,
			Required: []string{
				("Target | Build | Lower | Run | Raw diagnostics | Region " +
					"lowering | Alignment semantics | Claim level"),
				"| linux-x64 | yes | yes | yes | yes | yes/partial | yes | production/host_runtime |",
				("| linux-x86 | yes | yes | no/host-dependent | partial | partial " +
					"| partial | build_lower_only |"),
				("| linux-x32 | yes | yes | no/host-dependent | partial | partial " +
					"| special | build_lower_only |"),
				("| macos-x64 | yes | yes | host-required | host-required | host-" +
					"required | host-required | build_lower_only unless run |"),
				("| windows-x64 | yes | yes | host-required | host-required | " +
					"host-required | host-required | build_lower_only unless run |"),
				("| wasm32-wasi | yes | yes | runner-smoke if available | safe-" +
					"only | limited | wasm rules | artifact/runtime tiered |"),
				("| wasm32-web | yes | yes | browser-smoke if available | safe-" +
					"only | limited | wasm rules | artifact/runtime tiered |"),
				"no cross-target memory production claim without target evidence",
			},
		},
		{
			Name: "memory cost model",
			Path: paths.MemoryCostModel,
			Required: []string{
				"Memory Cost Model",
				"zero_cost_proven",
				"dynamic_check_required",
				"instrumentation_only",
				"unsupported_rejected",
				"conservative_fallback",
				"normal build does not run heavy validators at runtime",
				"report generation is optional and artifact-only",
				"unsafe_unknown may be checked, trapped, or conservative, but never optimized as trusted",
				"`cost_class`",
				"`normal_build_check`",
			},
		},
		{
			Name: "memory fuzz oracle",
			Path: paths.MemoryFuzzOracle,
			Required: []string{
				"Memory Fuzz Oracle v1",
				"tetra.memory-fuzz.oracle.v1",
				"checker reject expected",
				"runtime trap expected",
				"compiled output equals interpreter/reference expected",
				"compiler crash is bug",
				"miscompile is bug",
				"unsafe_unknown optimized as safe is bug",
				"report validation failure is bug",
				"Tier 1 short CI smoke",
				"Tier 2 nightly fuzz",
				"Tier 3 release-blocking focused memory fuzz",
				"no safe metadata mutation",
				"no borrowed escape",
				"no unsafe_unknown -> safe_known",
				"no removed bounds check without proof id",
				"no stack/region storage if escape exists",
				"reports validate against MemoryFactGraph",
				"reports/memory-fuzz-short",
			},
		},
		{
			Name: "memory production final audit",
			Path: paths.MemoryProductionFinal,
			Required: []string{
				"Memory Production Core v1 Final Audit",
				"MPC-0",
				"MPC-16",
				"implemented",
				"implemented_narrow",
				"validated",
				"conservative",
				"rejected",
				"future",
				"explicit_non_goal",
				"MemoryFactGraph",
				"reports are projections",
				"docs/audits/memory/production/memory-production-core-v1-artifact-map.md",
				"docs/audits/memory/production/memory-production-core-v1-nonclaims.md",
			},
		},
		{
			Name: "memory production artifact map",
			Path: paths.MemoryProductionMap,
			Required: []string{
				"Memory Production Core v1 Artifact Map",
				"reports/memory-production-core-v1/test-all-quick",
				"summary.json",
				"summary.md",
				"scripts/ci/test-all.sh --quick --keep-going",
				"reports/memory-fuzz-short/mpc15/memory-fuzz-oracle.json",
				"reports/memory-production-core-v1/mpc8/memory-production-linux-x64.json",
				"reports/memory-production-core-v1/mpc9/memory-production-linux-x64.json",
			},
		},
		{
			Name: "memory production nonclaims",
			Path: paths.MemoryProductionClaims,
			Required: []string{
				"Memory Production Core v1 Nonclaims",
				"perfect memory in all possible programs",
				"full Rust-like borrow checker parity",
				"full FFI lifetime system",
				"safety for arbitrary unsafe external pointers",
				"full derived-pointer provenance for every raw address",
				"full production actor runtime",
				"full target runtime parity",
				"production object memory",
				"production persistent memory",
				"fastest language",
				"official benchmark result",
			},
		},
	}
}

func verifyMemoryProductionContractDocs(paths memoryProductionContractDocPaths) error {
	var errs []string
	for _, requirement := range memoryProductionContractRequirements(paths) {
		raw, err := os.ReadFile(requirement.Path)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", requirement.Path, err))
			continue
		}
		text := string(raw)
		for _, want := range requirement.Required {
			if !strings.Contains(text, want) {
				errs = append(
					errs,
					fmt.Sprintf(
						"%s: missing %q for %s memory production contract",
						requirement.Path,
						want,
						requirement.Name,
					),
				)
			}
		}
		if requirement.Path != paths.MemoryProductionClaims {
			for _, claim := range forbiddenPublicPerformanceClaims(text) {
				errs = append(
					errs,
					fmt.Sprintf(
						"%s: forbidden %s claim in %s memory production contract",
						requirement.Path,
						claim,
						requirement.Name,
					),
				)
			}
			for _, claim := range forbiddenPersistentObjectMemoryClaims(text) {
				errs = append(
					errs,
					fmt.Sprintf(
						"%s: forbidden %s claim in %s memory production contract",
						requirement.Path,
						claim,
						requirement.Name,
					),
				)
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

func forbiddenPublicPerformanceClaims(text string) []string {
	lower := strings.ToLower(text)
	var claims []string
	for _, phrase := range []string{
		"fastest language",
		"fastest-language",
		"official benchmark result",
		"official benchmark",
		"official techempower result",
		"official techempower",
		"target parity",
		"target-parity",
		"all-target memory parity",
		"all target memory parity",
		"zero-cost performance",
		"zero cost performance",
		"memory 100%",
		"memory 100 percent",
		"full formal proof",
		"formal proof of memory safety",
		"perfect memory",
		"leak-free",
		"leak free",
		"leak freedom",
		"no leaks",
		"no memory leaks",
		"islandkernel complete",
		"islandkernel is complete",
		"island kernel complete",
		"island kernel is complete",
		"full islandkernel",
		"full island kernel",
		"arbitrary unsafe pointer safety",
		"arbitrary external pointer safety",
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

func forbiddenPersistentObjectMemoryClaims(text string) []string {
	lower := strings.ToLower(text)
	var claims []string
	for _, phrase := range []string{
		"object memory",
		"persistent memory",
		"persistent/object memory",
		"object/persistent memory",
		"production object memory",
		"object memory production",
		"production persistent memory",
		"persistent memory production",
		"todium",
		"memoryfield",
		"memoryruntime",
		"memoryeval",
		"false memory",
		"stale memory",
		"wal-backed object memory",
		"wal backed object memory",
		"fts-backed object memory",
		"fts backed object memory",
		"vacuum-backed object memory",
		"retention-backed object memory",
	} {
		searchFrom := 0
		for {
			index := strings.Index(lower[searchFrom:], phrase)
			if index < 0 {
				break
			}
			absolute := searchFrom + index
			clause := clauseAround(lower, absolute, len(phrase), 260)
			if !explicitNonClaimContext(clause) &&
				persistentObjectMemoryClaimContext(phrase, clause) {
				claims = append(claims, phrase)
			}
			searchFrom = absolute + len(phrase)
		}
	}
	sort.Strings(claims)
	return compactStrings(claims)
}

func persistentObjectMemoryClaimContext(phrase string, clause string) bool {
	switch phrase {
	case "object memory",
		"persistent memory",
		"persistent/object memory",
		"object/persistent memory":
		for _, qualifier := range []string{
			"production",
			"prod_ready",
			"release-ready",
			"release ready",
			"supported",
			"current",
			"ships",
			"backed by",
		} {
			if strings.Contains(clause, qualifier) {
				return true
			}
		}
		return false
	default:
		return true
	}
}

func excerptAround(text string, index int, length int, radius int) string {
	start := index - radius
	if start < 0 {
		start = 0
	}
	end := index + length + radius
	if end > len(text) {
		end = len(text)
	}
	return text[start:end]
}

func sentenceAround(text string, index int, length int, maxSide int) string {
	start := index
	for start > 0 && !sentenceBoundary(text, start-1) {
		start--
		if index-start >= maxSide {
			break
		}
	}
	end := index + length
	for end < len(text) && !sentenceBoundary(text, end) {
		end++
		if end-(index+length) >= maxSide {
			break
		}
	}
	if end < len(text) && sentenceBoundary(text, end) {
		end++
	}
	return text[start:end]
}

func sentenceBoundary(text string, index int) bool {
	if index < 0 || index >= len(text) || !strings.ContainsRune(".!?", rune(text[index])) {
		return false
	}
	return strings.Count(text[:index], "`")%2 == 0
}

func explicitNonClaimContext(lower string) bool {
	normalized := strings.NewReplacer(`"`, "", "`", "", "'", "").Replace(lower)
	for _, marker := range []string{
		"does not claim",
		"do not claim",
		"does not prove",
		"do not prove",
		"does not promote",
		"do not promote",
		"must not use",
		"not an official",
		"not a fastest",
		"not fastest",
		"not target parity",
		"not a benchmark",
		"not a full",
		"not full",
		"not a runtime measurement",
		"not complete",
		"not leak-free",
		"not leak free",
		"not memory 100",
		"not a clean release-candidate",
		"not clean release-candidate",
		"no official",
		"no fastest",
		"no target parity",
		"no leak-free",
		"no leak free",
		"no memory 100",
		"no arbitrary unsafe",
		"no broad memory",
		"no full",
		"makes no",
		"model-only",
		"model only",
		"non-goal",
		"non goal",
		"out of scope",
		"not included",
		"does not include",
		"absent",
		"no production object memory",
		"no production persistent memory",
		"no production actor runtime",
		"no actor production gate passed",
		"no performance superiority",
		"no c++/rust parity",
		"no measured speed comparison",
		"no todium",
		"no memoryfield",
		"no prod_ready",
		"not prod_ready",
		"no prod_ready_proven",
		"not prod_ready_proven",
		"not_claimed",
		"without an official",
		"without official",
		"forbid",
		"forbidden",
		"non-claim",
		"nonclaim",
	} {
		if strings.Contains(lower, marker) || strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func compactStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := values[:0]
	var previous string
	for _, value := range values {
		if value == previous {
			continue
		}
		out = append(out, value)
		previous = value
	}
	return out
}
