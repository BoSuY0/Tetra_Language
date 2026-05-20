package x64obj

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/format/tobj"
	"tetra_language/compiler/internal/ir"
)

func TestObjectBuildCollectsRelocsAndSymbols(t *testing.T) {
	emit := func(
		e *x64.Emitter,
		fn ir.IRFunc,
		dataBlobs *[][]byte,
		leaPatches *[]LeaPatch,
		callPatches *[]CallPatch,
		importPatches *[]ImportPatch,
		opt x64.CodegenOptions,
	) error {
		e.Emit(0x90, 0x90, 0x90, 0x90, 0xC3)
		*dataBlobs = append(*dataBlobs, []byte("blob"))
		*leaPatches = append(*leaPatches, LeaPatch{At: 0, DataIndex: 0})
		*callPatches = append(*callPatches, CallPatch{At: 0, Name: "ext.call"})
		if importPatches != nil {
			*importPatches = append(*importPatches, ImportPatch{At: 0, Name: "kernel32.ExitProcess"})
		}
		return nil
	}

	obj, err := BuildObject([]ir.IRFunc{{
		Name:        "main",
		ExportName:  "entry",
		ParamSlots:  0,
		LocalSlots:  0,
		ReturnSlots: 1,
		Instrs:      []ir.IRInstr{{Kind: ir.IRConstI32, Imm: 0}, {Kind: ir.IRReturn}},
	}}, emit, x64.CodegenOptions{}, Options{CollectImports: true})
	if err != nil {
		t.Fatalf("BuildObject: %v", err)
	}
	if len(obj.Code) != 5 {
		t.Fatalf("code len = %d, want 5", len(obj.Code))
	}
	if string(obj.Data) != "blob" {
		t.Fatalf("data = %q, want blob", string(obj.Data))
	}
	hasKind := func(kind tobj.RelocKind) bool {
		for _, r := range obj.Relocs {
			if r.Kind == kind {
				return true
			}
		}
		return false
	}
	if !hasKind(tobj.RelocCallRel32) || !hasKind(tobj.RelocIATDisp32) || !hasKind(tobj.RelocDataDisp32) {
		t.Fatalf("missing expected reloc kinds: %#v", obj.Relocs)
	}
	if len(obj.Symbols) != 2 || obj.Symbols[0].Name != "entry" || obj.Symbols[1].Name != "main" {
		t.Fatalf("unexpected symbols ordering/content: %#v", obj.Symbols)
	}
}

func TestObjectBuildRejectsInvalidDataPatchIndex(t *testing.T) {
	emit := func(
		e *x64.Emitter,
		fn ir.IRFunc,
		dataBlobs *[][]byte,
		leaPatches *[]LeaPatch,
		callPatches *[]CallPatch,
		importPatches *[]ImportPatch,
		opt x64.CodegenOptions,
	) error {
		e.Emit(0x90, 0x90, 0x90, 0x90)
		*leaPatches = append(*leaPatches, LeaPatch{At: 0, DataIndex: 9})
		return nil
	}

	_, err := BuildObject([]ir.IRFunc{{
		Name:        "main",
		ParamSlots:  0,
		LocalSlots:  0,
		ReturnSlots: 1,
		Instrs:      []ir.IRInstr{{Kind: ir.IRConstI32, Imm: 0}, {Kind: ir.IRReturn}},
	}}, emit, x64.CodegenOptions{}, Options{})
	if err == nil {
		t.Fatalf("expected invalid data patch error")
	}
	if !strings.Contains(err.Error(), "invalid data patch index") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestObjectBuildRejectsEmptyFunctionName(t *testing.T) {
	emit := func(
		e *x64.Emitter,
		fn ir.IRFunc,
		dataBlobs *[][]byte,
		leaPatches *[]LeaPatch,
		callPatches *[]CallPatch,
		importPatches *[]ImportPatch,
		opt x64.CodegenOptions,
	) error {
		e.Emit(0xC3)
		return nil
	}

	_, err := BuildObject([]ir.IRFunc{{Name: ""}}, emit, x64.CodegenOptions{}, Options{})
	if err == nil {
		t.Fatalf("expected empty function name error")
	}
	if !strings.Contains(err.Error(), "function name is empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestObjectBuildRejectsInvalidFunctionSlots(t *testing.T) {
	emit := func(
		e *x64.Emitter,
		fn ir.IRFunc,
		dataBlobs *[][]byte,
		leaPatches *[]LeaPatch,
		callPatches *[]CallPatch,
		importPatches *[]ImportPatch,
		opt x64.CodegenOptions,
	) error {
		e.Emit(0xC3)
		return nil
	}

	cases := []struct {
		name string
		fn   ir.IRFunc
	}{
		{
			name: "negative_param_slots",
			fn:   ir.IRFunc{Name: "main", ParamSlots: -1, LocalSlots: 0},
		},
		{
			name: "negative_local_slots",
			fn:   ir.IRFunc{Name: "main", ParamSlots: 0, LocalSlots: -1},
		},
		{
			name: "params_exceed_locals",
			fn:   ir.IRFunc{Name: "main", ParamSlots: 2, LocalSlots: 1},
		},
		{
			name: "negative_return_slots",
			fn:   ir.IRFunc{Name: "main", ReturnSlots: -1},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := BuildObject([]ir.IRFunc{tc.fn}, emit, x64.CodegenOptions{}, Options{})
			if err == nil {
				t.Fatalf("expected invalid function slot error")
			}
			if !strings.Contains(err.Error(), "function 'main' has invalid slots") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestObjectBuildRejectsObjectLocalCallSignatureMismatch(t *testing.T) {
	emit := func(
		e *x64.Emitter,
		fn ir.IRFunc,
		dataBlobs *[][]byte,
		leaPatches *[]LeaPatch,
		callPatches *[]CallPatch,
		importPatches *[]ImportPatch,
		opt x64.CodegenOptions,
	) error {
		at := len(e.Buf)
		e.Emit(0x90, 0x90, 0x90, 0x90)
		if fn.Name == "main" {
			*callPatches = append(*callPatches, CallPatch{At: at, Name: "helper"})
		}
		return nil
	}

	_, err := BuildObject([]ir.IRFunc{
		{
			Name:        "helper",
			ParamSlots:  1,
			LocalSlots:  1,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRReturn},
			},
		},
		{
			Name:        "main",
			ParamSlots:  0,
			LocalSlots:  0,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRCall, Name: "helper", ArgSlots: 0, RetSlots: 1},
				{Kind: ir.IRReturn},
			},
		},
	}, emit, x64.CodegenOptions{}, Options{})
	if err == nil {
		t.Fatalf("expected call signature mismatch error")
	}
	if !strings.Contains(err.Error(), `call "helper" ABI mismatch`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestObjectBuildRejectsEmptyCallPatchName(t *testing.T) {
	emit := func(
		e *x64.Emitter,
		fn ir.IRFunc,
		dataBlobs *[][]byte,
		leaPatches *[]LeaPatch,
		callPatches *[]CallPatch,
		importPatches *[]ImportPatch,
		opt x64.CodegenOptions,
	) error {
		e.Emit(0x90, 0x90, 0x90, 0x90)
		*callPatches = append(*callPatches, CallPatch{At: 0, Name: ""})
		return nil
	}

	_, err := BuildObject([]ir.IRFunc{{Name: "main"}}, emit, x64.CodegenOptions{}, Options{})
	if err == nil {
		t.Fatalf("expected empty call patch name error")
	}
	if !strings.Contains(err.Error(), "call patch name is empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestObjectBuildRejectsEmptyImportPatchName(t *testing.T) {
	emit := func(
		e *x64.Emitter,
		fn ir.IRFunc,
		dataBlobs *[][]byte,
		leaPatches *[]LeaPatch,
		callPatches *[]CallPatch,
		importPatches *[]ImportPatch,
		opt x64.CodegenOptions,
	) error {
		e.Emit(0x90, 0x90, 0x90, 0x90)
		*importPatches = append(*importPatches, ImportPatch{At: 0, Name: ""})
		return nil
	}

	_, err := BuildObject([]ir.IRFunc{{Name: "main"}}, emit, x64.CodegenOptions{}, Options{CollectImports: true})
	if err == nil {
		t.Fatalf("expected empty import patch name error")
	}
	if !strings.Contains(err.Error(), "import patch name is empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestObjectBuildRejectsInvalidPatchOffsets(t *testing.T) {
	cases := []struct {
		name    string
		options Options
		emit    func(*x64.Emitter, *[][]byte, *[]LeaPatch, *[]CallPatch, *[]ImportPatch)
	}{
		{
			name: "external_call_negative",
			emit: func(e *x64.Emitter, dataBlobs *[][]byte, leaPatches *[]LeaPatch, callPatches *[]CallPatch, importPatches *[]ImportPatch) {
				e.Emit(0x90, 0x90, 0x90, 0x90)
				*callPatches = append(*callPatches, CallPatch{At: -1, Name: "external"})
			},
		},
		{
			name: "external_call_past_end",
			emit: func(e *x64.Emitter, dataBlobs *[][]byte, leaPatches *[]LeaPatch, callPatches *[]CallPatch, importPatches *[]ImportPatch) {
				e.Emit(0x90, 0x90, 0x90, 0x90)
				*callPatches = append(*callPatches, CallPatch{At: 1, Name: "external"})
			},
		},
		{
			name:    "import_negative",
			options: Options{CollectImports: true},
			emit: func(e *x64.Emitter, dataBlobs *[][]byte, leaPatches *[]LeaPatch, callPatches *[]CallPatch, importPatches *[]ImportPatch) {
				e.Emit(0x90, 0x90, 0x90, 0x90)
				*importPatches = append(*importPatches, ImportPatch{At: -1, Name: "kernel32.ExitProcess"})
			},
		},
		{
			name: "data_negative",
			emit: func(e *x64.Emitter, dataBlobs *[][]byte, leaPatches *[]LeaPatch, callPatches *[]CallPatch, importPatches *[]ImportPatch) {
				e.Emit(0x90, 0x90, 0x90, 0x90)
				*dataBlobs = append(*dataBlobs, []byte("blob"))
				*leaPatches = append(*leaPatches, LeaPatch{At: -1, DataIndex: 0})
			},
		},
		{
			name: "data_past_end",
			emit: func(e *x64.Emitter, dataBlobs *[][]byte, leaPatches *[]LeaPatch, callPatches *[]CallPatch, importPatches *[]ImportPatch) {
				e.Emit(0x90, 0x90, 0x90, 0x90)
				*dataBlobs = append(*dataBlobs, []byte("blob"))
				*leaPatches = append(*leaPatches, LeaPatch{At: 1, DataIndex: 0})
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			emit := func(
				e *x64.Emitter,
				fn ir.IRFunc,
				dataBlobs *[][]byte,
				leaPatches *[]LeaPatch,
				callPatches *[]CallPatch,
				importPatches *[]ImportPatch,
				opt x64.CodegenOptions,
			) error {
				tc.emit(e, dataBlobs, leaPatches, callPatches, importPatches)
				return nil
			}

			_, err := BuildObject([]ir.IRFunc{{Name: "main"}}, emit, x64.CodegenOptions{}, tc.options)
			if err == nil {
				t.Fatalf("expected invalid patch offset error")
			}
			if !strings.Contains(err.Error(), "invalid patch offset") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestObjectBuildRejectsDuplicateSymbols(t *testing.T) {
	emit := func(
		e *x64.Emitter,
		fn ir.IRFunc,
		dataBlobs *[][]byte,
		leaPatches *[]LeaPatch,
		callPatches *[]CallPatch,
		importPatches *[]ImportPatch,
		opt x64.CodegenOptions,
	) error {
		e.Emit(0xC3)
		return nil
	}

	_, err := BuildObject([]ir.IRFunc{
		{Name: "main", ExportName: "dup"},
		{Name: "second", ExportName: "dup"},
	}, emit, x64.CodegenOptions{}, Options{})
	if err == nil {
		t.Fatalf("expected duplicate symbol error")
	}
	if !strings.Contains(err.Error(), "duplicate exported symbol") {
		t.Fatalf("unexpected error: %v", err)
	}
}
