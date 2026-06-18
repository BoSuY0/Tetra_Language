package runtimeabi_test

import (
	"strings"
	"testing"

	. "tetra_language/compiler/internal/runtimeabi"
)

func TestRuntimeAllocationContractsCoverP5Entrypoints(t *testing.T) {
	contracts := RuntimeAllocationContracts()
	seen := map[string]bool{}
	for _, contract := range contracts {
		seen[contract.API] = true
		if err := ValidateRuntimeAllocationContract(contract); err != nil {
			t.Fatalf("%s contract invalid: %v", contract.API, err)
		}
	}

	for _, api := range []string{
		"core.alloc_bytes",
		"make_u8",
		"make_u16",
		"make_i32",
		"make_bool",
		"core.island_new",
		"core.island_make_u8",
		"core.island_make_u16",
		"core.island_make_i32",
		"core.island_make_bool",
		"region.temp",
	} {
		if !seen[api] {
			t.Fatalf("RuntimeAllocationContracts missing %s", api)
		}
	}
}

func TestRuntimeAllocationContractForMakeSlice(t *testing.T) {
	contract, ok := RuntimeAllocationContractForAPI("make_u8")
	if !ok {
		t.Fatal("missing make_u8 contract")
	}
	if contract.ZeroSizeBehavior != AllocationZeroCanonicalEmpty {
		t.Fatalf(
			"zero-size behavior = %q, want %q",
			contract.ZeroSizeBehavior,
			AllocationZeroCanonicalEmpty,
		)
	}
	if contract.NegativeBehavior != AllocationRejectBeforeAllocator ||
		contract.OverflowBehavior != AllocationRejectBeforeAllocator {
		t.Fatalf(
			"make_u8 guards = negative %q overflow %q, want reject before allocator",
			contract.NegativeBehavior,
			contract.OverflowBehavior,
		)
	}
	if contract.AlignmentBytes != 16 {
		t.Fatalf("alignment = %d, want 16", contract.AlignmentBytes)
	}
	for _, hook := range []AllocationReportHook{
		AllocationReportStorageClass,
		AllocationReportRuntimePath,
		AllocationReportBytesRequested,
		AllocationReportBytesReserved,
		AllocationReportLifetime,
		AllocationReportDomainID,
		AllocationReportDomainKind,
		AllocationReportDomainOwner,
		AllocationReportDomainLifetime,
	} {
		if !contract.HasReportHook(hook) {
			t.Fatalf("make_u8 contract missing report hook %s", hook)
		}
	}
}

func TestRuntimeAllocationContractDistinguishesAllocBytesAndIsland(t *testing.T) {
	raw, ok := RuntimeAllocationContractForAPI("core.alloc_bytes")
	if !ok {
		t.Fatal("missing core.alloc_bytes contract")
	}
	if raw.ZeroSizeBehavior != AllocationZeroInvalidPrecondition {
		t.Fatalf(
			"alloc_bytes zero-size behavior = %q, want invalid precondition",
			raw.ZeroSizeBehavior,
		)
	}
	if raw.RuntimePath != AllocationPathHeap {
		t.Fatalf("alloc_bytes runtime path = %q, want heap", raw.RuntimePath)
	}
	for _, hook := range []AllocationReportHook{
		AllocationReportRawPointerBounds,
		AllocationReportRawPointerBase,
		AllocationReportRawPointerSlicePolicy,
	} {
		if !raw.HasReportHook(hook) {
			t.Fatalf("alloc_bytes contract missing raw pointer report hook %s", hook)
		}
	}
	abi := RuntimeRawPointerBoundsABI()
	if abi.AllocationBaseStatus != RawPointerBoundsAllocationBase ||
		abi.UnknownPointerStatus != RawPointerBoundsCheckedExternalUnknown {
		t.Fatalf(
			"raw pointer bounds ABI = %+v, want allocation-base and checked-unknown statuses",
			abi,
		)
	}

	island, ok := RuntimeAllocationContractForAPI("core.island_make_i32")
	if !ok {
		t.Fatal("missing core.island_make_i32 contract")
	}
	if island.RuntimePath != AllocationPathExplicitIsland {
		t.Fatalf("island_make_i32 runtime path = %q, want explicit island", island.RuntimePath)
	}
	if island.ZeroSizeBehavior != AllocationZeroCanonicalEmptyNoMetadata {
		t.Fatalf(
			"island_make_i32 zero-size behavior = %q, want no-metadata empty",
			island.ZeroSizeBehavior,
		)
	}
	if !island.HasDebugInstrumentation(AllocationDebugDoubleFree) ||
		!island.HasDebugInstrumentation(AllocationDebugUseAfterFree) {
		t.Fatalf(
			"island_make_i32 debug instrumentation = %v, want double-free and use-after-free hooks",
			island.DebugInstrumentation,
		)
	}
}

func TestRuntimeAllocationMemoryDomainHelpers(t *testing.T) {
	process := DefaultProcessMemoryDomain(17, 32)
	if process.DomainID != "domain:process" || process.Kind != DomainProcess ||
		process.RequestedBytes != 17 ||
		process.ReservedBytes != 32 {
		t.Fatalf("process domain = %+v, want process domain with requested/reserved bytes", process)
	}
	if err := ValidateMemoryDomain(process); err != nil {
		t.Fatalf("ValidateMemoryDomain(process): %v", err)
	}

	island := IslandMemoryDomain("island:isl", "island:isl:scope", 17, 32)
	if island.DomainID != "domain:island:isl" || island.Kind != DomainIsland ||
		island.OwnerID != "isl" ||
		island.Lifetime != "island:isl:scope" {
		t.Fatalf("island domain = %+v, want island domain bound to region/lifetime", island)
	}
	if err := ValidateMemoryDomain(island); err != nil {
		t.Fatalf("ValidateMemoryDomain(island): %v", err)
	}
}

func TestRuntimeAllocationMemoryDomainHelpers(t *testing.T) {
	process := DefaultProcessMemoryDomain(17, 32)
	if process.DomainID != "domain:process" || process.Kind != DomainProcess || process.RequestedBytes != 17 || process.ReservedBytes != 32 {
		t.Fatalf("process domain = %+v, want process domain with requested/reserved bytes", process)
	}
	if err := ValidateMemoryDomain(process); err != nil {
		t.Fatalf("ValidateMemoryDomain(process): %v", err)
	}

	island := IslandMemoryDomain("island:isl", "island:isl:scope", 17, 32)
	if island.DomainID != "domain:island:isl" || island.Kind != DomainIsland || island.OwnerID != "isl" || island.Lifetime != "island:isl:scope" {
		t.Fatalf("island domain = %+v, want island domain bound to region/lifetime", island)
	}
	if err := ValidateMemoryDomain(island); err != nil {
		t.Fatalf("ValidateMemoryDomain(island): %v", err)
	}
}

func TestRuntimeAllocationContractValidationRejectsWeakSpecs(t *testing.T) {
	contract := RuntimeAllocationContract{
		API:              "bad",
		RuntimePath:      AllocationPathHeap,
		AlignmentBytes:   24,
		ZeroSizeBehavior: AllocationZeroCanonicalEmpty,
		NegativeBehavior: AllocationRejectBeforeAllocator,
		OverflowBehavior: AllocationRejectBeforeAllocator,
		FailureBehavior:  AllocationFailureTrapOrStatus,
		ReportHooks:      RequiredAllocationReportHooks(),
	}
	err := ValidateRuntimeAllocationContract(contract)
	if err == nil || !strings.Contains(err.Error(), "power-of-two") {
		t.Fatalf(
			"ValidateRuntimeAllocationContract error = %v, want power-of-two alignment diagnostic",
			err,
		)
	}

	contract.AlignmentBytes = 16
	contract.OverflowBehavior = ""
	err = ValidateRuntimeAllocationContract(contract)
	if err == nil || !strings.Contains(err.Error(), "overflow") {
		t.Fatalf("ValidateRuntimeAllocationContract error = %v, want overflow diagnostic", err)
	}
}
