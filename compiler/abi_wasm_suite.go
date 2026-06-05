package compiler

import (
	"fmt"
	"strings"

	wasm32wasi "tetra_language/compiler/internal/backend/wasm32_wasi"
	wasm32web "tetra_language/compiler/internal/backend/wasm32_web"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/semantics"
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
	if tgt.Arch != ctarget.ArchWASM32 || tgt.Format != ctarget.FormatWASM || tgt.DataModel != ctarget.DataModelILP32 || tgt.Endian != ctarget.EndianLittle {
		return fmt.Errorf("%s target model = arch=%s format=%s model=%s endian=%s, want wasm32/wasm/ilp32/little", tgt.Triple, tgt.Arch, tgt.Format, tgt.DataModel, tgt.Endian)
	}
	if tgt.PointerWidthBits != 32 || tgt.NativeIntWidthBits != 32 || tgt.RegisterWidthBits != 32 || tgt.StackAlignmentBytes != 16 || tgt.MaxAtomicWidthBits != 64 {
		return fmt.Errorf("%s widths = ptr=%d native=%d reg=%d stack=%d atomic=%d, want 32/32/32/16/64", tgt.Triple, tgt.PointerWidthBits, tgt.NativeIntWidthBits, tgt.RegisterWidthBits, tgt.StackAlignmentBytes, tgt.MaxAtomicWidthBits)
	}
	switch tgt.Triple {
	case "wasm32-wasi":
		if tgt.OS != ctarget.OSWASI || tgt.ABI != ctarget.ABIWASI || tgt.RunMode != ctarget.RunModeWASIRunner || tgt.ExeExt != ".wasm" {
			return fmt.Errorf("%s identity = os=%s abi=%s run=%s ext=%q, want wasi/wasi/wasi_runner/.wasm", tgt.Triple, tgt.OS, tgt.ABI, tgt.RunMode, tgt.ExeExt)
		}
	case "wasm32-web":
		if tgt.OS != ctarget.OSWeb || tgt.ABI != ctarget.ABIWeb || tgt.RunMode != ctarget.RunModeWebRunner || tgt.ExeExt != ".wasm" {
			return fmt.Errorf("%s identity = os=%s abi=%s run=%s ext=%q, want web/web/web_runner/.wasm", tgt.Triple, tgt.OS, tgt.ABI, tgt.RunMode, tgt.ExeExt)
		}
	default:
		return fmt.Errorf("unsupported wasm ABI target %s", tgt.Triple)
	}
	for _, scalar := range []struct {
		name  string
		size  int
		align int
	}{
		{name: "ptr", size: 4, align: 4},
		{name: "fnptr", size: 4, align: 4},
		{name: "usize", size: 4, align: 4},
		{name: "isize", size: 4, align: 4},
		{name: "c_long", size: 4, align: 4},
	} {
		if err := expectTargetScalarLayout(tgt, scalar.name, scalar.size, scalar.align); err != nil {
			return err
		}
	}
	return nil
}

func checkWASMSlotABIMetadata(tgt ctarget.Target) error {
	for _, layoutCase := range []struct {
		name      string
		size      int
		alignment int
	}{
		{name: "ptr", size: 4, alignment: 4},
		{name: "usize", size: 4, alignment: 4},
		{name: "fnptr", size: 4, alignment: 4},
		{name: "string", size: 8, alignment: 4},
	} {
		layout, ok := tgt.ScalarLayout(layoutCase.name)
		if layoutCase.name == "string" {
			str, err := tgt.StringLayout()
			if err != nil {
				return err
			}
			if str.SizeBytes != layoutCase.size || str.AlignBytes != layoutCase.alignment {
				return fmt.Errorf("%s string slot layout = size=%d align=%d, want %d/%d", tgt.Triple, str.SizeBytes, str.AlignBytes, layoutCase.size, layoutCase.alignment)
			}
			continue
		}
		if !ok {
			return fmt.Errorf("%s missing scalar slot layout %s", tgt.Triple, layoutCase.name)
		}
		if layout.SizeBytes != layoutCase.size || layout.AlignBytes != layoutCase.alignment || layout.ABIBytes != layoutCase.size {
			return fmt.Errorf("%s %s slot layout = %#v, want size/align/abi %d/%d/%d", tgt.Triple, layoutCase.name, layout, layoutCase.size, layoutCase.alignment, layoutCase.size)
		}
	}

	atomic, err := tgt.AtomicPointerLayout()
	if err != nil {
		return err
	}
	if atomic.WidthBits != 32 || atomic.RegisterWidthBits != 32 || !atomic.PointerSized {
		return fmt.Errorf("%s pointer atomic slot layout = %#v, want 32-bit pointer-sized wasm slot", tgt.Triple, atomic)
	}
	return nil
}

func checkWASMAggregateReturnLayouts(tgt ctarget.Target) error {
	pair, err := tgt.StructLayout([]ctarget.LayoutField{
		{Name: "raw", Type: "ptr"},
		{Name: "count", Type: "usize"},
	})
	if err != nil {
		return err
	}
	if pair.SizeBytes != 8 || pair.AlignBytes != 4 || len(pair.Fields) != 2 || pair.Fields[1].OffsetBytes != 4 {
		return fmt.Errorf("%s struct return layout = %#v, want 8-byte two-slot pair", tgt.Triple, pair)
	}
	slice, err := tgt.SliceLayout("u8")
	if err != nil {
		return err
	}
	if slice.SizeBytes != 8 || slice.AlignBytes != 4 || len(slice.Fields) != 2 || slice.Fields[0].Type != "ptr" || slice.Fields[1].OffsetBytes != 4 {
		return fmt.Errorf("%s slice return layout = %#v, want ptr/i32 two-slot view", tgt.Triple, slice)
	}
	str, err := tgt.StringLayout()
	if err != nil {
		return err
	}
	if str.SizeBytes != slice.SizeBytes || str.AlignBytes != slice.AlignBytes {
		return fmt.Errorf("%s String return layout = %#v, want same layout as []u8 %#v", tgt.Triple, str, slice)
	}
	enum, err := tgt.EnumLayout([]ctarget.EnumCaseLayout{
		{Name: "Empty"},
		{Name: "Text", Payload: []ctarget.LayoutField{{Name: "value", Type: "string"}}},
	})
	if err != nil {
		return err
	}
	if enum.SizeBytes != 12 || enum.AlignBytes != 4 || enum.PayloadOffsetBytes != 4 || enum.PayloadSizeBytes != 8 {
		return fmt.Errorf("%s enum return layout = %#v, want tag plus 8-byte String payload", tgt.Triple, enum)
	}
	return nil
}

func checkWASMCallBoundaryValidation(tgt ctarget.Target) error {
	switch tgt.Triple {
	case "wasm32-wasi":
		obj, err := wasm32wasi.CodegenObject(wasmABIValidCallFuncs(), "main")
		if err != nil {
			return err
		}
		if _, err := wasm32wasi.LinkObject(obj); err != nil {
			return err
		}
		_, err = wasm32wasi.CodegenObject(wasmABIMismatchedCallFuncs(), "main")
		if err == nil || !strings.Contains(err.Error(), `call "helper" ABI mismatch`) {
			return fmt.Errorf("%s mismatched call metadata diagnostic = %v, want ABI mismatch", tgt.Triple, err)
		}
	case "wasm32-web":
		obj, err := wasm32web.CodegenObject(wasmABIValidCallFuncs(), "main")
		if err != nil {
			return err
		}
		if _, err := wasm32web.LinkObject(obj); err != nil {
			return err
		}
		_, err = wasm32web.CodegenObject(wasmABIMismatchedCallFuncs(), "main")
		if err == nil || !strings.Contains(err.Error(), `call "helper" ABI mismatch`) {
			return fmt.Errorf("%s mismatched call metadata diagnostic = %v, want ABI mismatch", tgt.Triple, err)
		}
	default:
		return fmt.Errorf("unsupported wasm call-boundary target %s", tgt.Triple)
	}
	return nil
}

func wasmABIValidCallFuncs() []ir.IRFunc {
	return []ir.IRFunc{
		{
			Name:        "helper",
			ParamSlots:  2,
			LocalSlots:  2,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRAddI32},
				{Kind: ir.IRReturn},
			},
		},
		{
			Name:        "main",
			ParamSlots:  0,
			LocalSlots:  0,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 1},
				{Kind: ir.IRConstI32, Imm: 2},
				{Kind: ir.IRCall, Name: "helper", ArgSlots: 2, RetSlots: 1},
				{Kind: ir.IRReturn},
			},
		},
	}
}

func wasmABIMismatchedCallFuncs() []ir.IRFunc {
	funcs := wasmABIValidCallFuncs()
	funcs[1].Instrs[2].ArgSlots = 1
	return funcs
}

func checkWASMFFIReprCBoundaryPolicy(tgt ctarget.Target) error {
	if targetRequiresExplicitAggregateExportGate(tgt.Triple) {
		return fmt.Errorf("%s unexpectedly requires the native aggregate C ABI export gate", tgt.Triple)
	}
	if targetRequiresExplicitPointerExportGate(tgt.Triple) {
		return fmt.Errorf("%s unexpectedly requires the native pointer C ABI export gate", tgt.Triple)
	}
	for _, native := range []string{"linux-x86", "linux-x64", "linux-x32", "macos-x64", "windows-x64"} {
		if !targetRequiresExplicitAggregateExportGate(native) {
			return fmt.Errorf("native target %s lost explicit repr(C) aggregate export gate", native)
		}
	}
	types := map[string]*semantics.TypeInfo{
		"Pair":   {Name: "Pair", Kind: semantics.TypeStruct},
		"Bytes":  {Name: "Bytes", Kind: semantics.TypeSlice},
		"String": {Name: "String", Kind: semantics.TypeStr},
		"Choice": {Name: "Choice", Kind: semantics.TypeEnum},
	}
	for _, typeName := range []string{"Pair", "Bytes", "String", "Choice"} {
		if !targetExportedFFIRequiresAggregateABI(typeName, types) {
			return fmt.Errorf("aggregate FFI detector did not recognize %s", typeName)
		}
	}
	return nil
}
