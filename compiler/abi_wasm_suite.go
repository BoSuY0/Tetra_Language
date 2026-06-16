package compiler

import (
	"tetra_language/compiler/internal/abisuite"
	ctarget "tetra_language/compiler/target"
)

func runWASMABIChecks(tgt ctarget.Target) []ABICheck {
	prefix := tgt.Triple
	return runABIChecks([]struct {
		name string
		run  func() error
	}{
		{name: prefix + " target model", run: func() error { return checkWASMTargetModel(tgt) }},
		{name: prefix + " slot ABI metadata", run: func() error { return checkWASMSlotABIMetadata(tgt) }},
		{name: prefix + " struct/enum/slice/String return layout", run: func() error { return checkWASMAggregateReturnLayouts(tgt) }},
		{name: prefix + " call boundary validation", run: func() error { return checkWASMCallBoundaryValidation(tgt) }},
		{name: prefix + " FFI repr(C) boundary policy", run: func() error { return checkWASMFFIReprCBoundaryPolicy(tgt) }},
	})
}

func checkWASMTargetModel(tgt ctarget.Target) error {
	return abisuite.CheckWASMTargetModel(tgt)
}

func checkWASMSlotABIMetadata(tgt ctarget.Target) error {
	return abisuite.CheckWASMSlotABIMetadata(tgt)
}

func checkWASMAggregateReturnLayouts(tgt ctarget.Target) error {
	return abisuite.CheckWASMAggregateReturnLayouts(tgt)
}

func checkWASMCallBoundaryValidation(tgt ctarget.Target) error {
	return abisuite.CheckWASMCallBoundaryValidation(tgt)
}

func checkWASMFFIReprCBoundaryPolicy(tgt ctarget.Target) error {
	return abisuite.CheckWASMFFIReprCBoundaryPolicy(tgt)
}
