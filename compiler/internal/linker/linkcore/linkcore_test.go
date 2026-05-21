package linkcore

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"testing"

	"tetra_language/compiler/internal/format/tobj"
)

func readI32LE(b []byte) int32 {
	return int32(binary.LittleEndian.Uint32(b))
}

func TestLinkX64ObjectsPatchesEntryAndCalls(t *testing.T) {
	stub := []byte{0xE8, 0, 0, 0, 0, 0xC3}
	stubCallAt := 1

	objMain := &tobj.Object{
		Target:  "linux-x64",
		Module:  "a",
		Code:    []byte{0xC3},
		Symbols: []tobj.Symbol{{Name: "main", Offset: 0}},
	}
	objCaller := &tobj.Object{
		Target:  "linux-x64",
		Module:  "b",
		Code:    []byte{0xE8, 0, 0, 0, 0, 0xC3},
		Symbols: []tobj.Symbol{{Name: "caller", Offset: 0}},
		Relocs:  []tobj.Reloc{{Kind: tobj.RelocCallRel32, At: 1, Name: "main"}},
	}

	res, err := LinkX64Objects([]*tobj.Object{objCaller, objMain}, "main", stub, stubCallAt, 0)
	if err != nil {
		t.Fatalf("LinkX64Objects: %v", err)
	}

	mainOff, ok := res.Symbols["main"]
	if !ok {
		t.Fatalf("missing main symbol")
	}
	if mainOff != len(stub) {
		t.Fatalf("main offset mismatch: got=%d want=%d", mainOff, len(stub))
	}

	stubDisp := readI32LE(res.Text[stubCallAt : stubCallAt+4])
	wantStubDisp := int32(mainOff - (stubCallAt + 4))
	if stubDisp != wantStubDisp {
		t.Fatalf("stub rel32 mismatch: got=%d want=%d", stubDisp, wantStubDisp)
	}

	callerTextBase := len(stub) + len(objMain.Code)
	callAt := callerTextBase + 1
	callDisp := readI32LE(res.Text[callAt : callAt+4])
	wantCallDisp := int32(mainOff - (callAt + 4))
	if callDisp != wantCallDisp {
		t.Fatalf("call rel32 mismatch: got=%d want=%d", callDisp, wantCallDisp)
	}
}

func TestLinkX64ObjectsPatchesFunctionAddressRelocs(t *testing.T) {
	stub := []byte{0xE8, 0, 0, 0, 0, 0xC3}
	stubCallAt := 1

	objMain := &tobj.Object{
		Target:  "linux-x64",
		Module:  "a",
		Code:    []byte{0xC3},
		Symbols: []tobj.Symbol{{Name: "main", Offset: 0}},
	}
	objAddr := &tobj.Object{
		Target:  "linux-x64",
		Module:  "b",
		Code:    []byte{0x48, 0x8D, 0x05, 0, 0, 0, 0, 0xC3}, // lea rax, [rip+disp32]; ret
		Symbols: []tobj.Symbol{{Name: "addr_user", Offset: 0}},
		Relocs:  []tobj.Reloc{{Kind: tobj.RelocFuncAddrDisp32, At: 3, Name: "main"}},
	}

	res, err := LinkX64Objects([]*tobj.Object{objAddr, objMain}, "main", stub, stubCallAt, 0)
	if err != nil {
		t.Fatalf("LinkX64Objects: %v", err)
	}

	mainOff := res.Symbols["main"]
	funcAddrAt := len(stub) + len(objMain.Code) + 3
	disp := readI32LE(res.Text[funcAddrAt : funcAddrAt+4])
	want := int32(mainOff - (funcAddrAt + 4))
	if disp != want {
		t.Fatalf("function address rel32 mismatch: got=%d want=%d", disp, want)
	}
}

func TestLinkX64ObjectsCollectsAbsoluteFunctionAddressRelocs(t *testing.T) {
	stub := []byte{0xE8, 0, 0, 0, 0, 0xC3}
	stubCallAt := 1

	objMain := &tobj.Object{
		Target:  "linux-x86",
		Module:  "a",
		Code:    []byte{0xC3},
		Symbols: []tobj.Symbol{{Name: "main", Offset: 0}},
	}
	objAddr := &tobj.Object{
		Target:  "linux-x86",
		Module:  "b",
		Code:    []byte{0xB8, 0, 0, 0, 0, 0xC3}, // mov eax, imm32; ret
		Symbols: []tobj.Symbol{{Name: "addr_user", Offset: 0}},
		Relocs:  []tobj.Reloc{{Kind: tobj.RelocFuncAddrAbs32, At: 1, Name: "main"}},
	}

	res, err := LinkX64Objects([]*tobj.Object{objAddr, objMain}, "main", stub, stubCallAt, 0)
	if err != nil {
		t.Fatalf("LinkX64Objects: %v", err)
	}
	if len(res.FuncAbsRelocs) != 1 {
		t.Fatalf("expected 1 absolute function reloc, got %d", len(res.FuncAbsRelocs))
	}
	if res.FuncAbsRelocs[0].At != len(stub)+len(objMain.Code)+1 {
		t.Fatalf("absolute function reloc at mismatch: got=%d want=%d", res.FuncAbsRelocs[0].At, len(stub)+len(objMain.Code)+1)
	}
	if res.FuncAbsRelocs[0].TargetOff != len(stub) {
		t.Fatalf("absolute function reloc target mismatch: got=%d want=%d", res.FuncAbsRelocs[0].TargetOff, len(stub))
	}
}

func TestLinkX64ObjectsCollectsDataRelocs(t *testing.T) {
	stub := []byte{0xE8, 0, 0, 0, 0, 0xC3}
	stubCallAt := 1

	obj := &tobj.Object{
		Target:  "linux-x64",
		Module:  "m",
		Code:    []byte{0x48, 0x8D, 0x05, 0, 0, 0, 0, 0xC3}, // lea rax, [rip+disp32]; ret
		Data:    []byte("ABCD"),
		Symbols: []tobj.Symbol{{Name: "main", Offset: 0}},
		Relocs:  []tobj.Reloc{{Kind: tobj.RelocDataDisp32, At: 3, Addend: 1}},
	}

	res, err := LinkX64Objects([]*tobj.Object{obj}, "main", stub, stubCallAt, 0)
	if err != nil {
		t.Fatalf("LinkX64Objects: %v", err)
	}
	if string(res.Data) != "ABCD" {
		t.Fatalf("data mismatch: %q", string(res.Data))
	}
	if len(res.DataRelocs) != 1 {
		t.Fatalf("expected 1 data reloc, got %d", len(res.DataRelocs))
	}
	if res.DataRelocs[0].At != len(stub)+3 {
		t.Fatalf("data reloc at mismatch: got=%d want=%d", res.DataRelocs[0].At, len(stub)+3)
	}
	if res.DataRelocs[0].TargetOff != 1 {
		t.Fatalf("data reloc target mismatch: got=%d want=%d", res.DataRelocs[0].TargetOff, 1)
	}
}

func TestLinkX64ObjectsCollectsAbsoluteDataRelocs(t *testing.T) {
	stub := []byte{0xE8, 0, 0, 0, 0, 0xC3}
	stubCallAt := 1

	obj := &tobj.Object{
		Target:  "linux-x86",
		Module:  "m",
		Code:    []byte{0xA1, 0, 0, 0, 0, 0xC3}, // mov eax, moffs32; ret
		Data:    []byte("ABCDEFGH"),
		Symbols: []tobj.Symbol{{Name: "main", Offset: 0}},
		Relocs:  []tobj.Reloc{{Kind: tobj.RelocDataAbs32, At: 1, Addend: 4}},
	}

	res, err := LinkX64Objects([]*tobj.Object{obj}, "main", stub, stubCallAt, 0)
	if err != nil {
		t.Fatalf("LinkX64Objects: %v", err)
	}
	if len(res.DataRelocs) != 0 {
		t.Fatalf("pc-relative data relocs = %#v, want none", res.DataRelocs)
	}
	if len(res.DataAbsRelocs) != 1 {
		t.Fatalf("expected 1 absolute data reloc, got %d", len(res.DataAbsRelocs))
	}
	if res.DataAbsRelocs[0].At != len(stub)+1 {
		t.Fatalf("absolute data reloc at mismatch: got=%d want=%d", res.DataAbsRelocs[0].At, len(stub)+1)
	}
	if res.DataAbsRelocs[0].TargetOff != 4 {
		t.Fatalf("absolute data reloc target mismatch: got=%d want=%d", res.DataAbsRelocs[0].TargetOff, 4)
	}
}

func TestLinkX64ObjectsCollectsIATRelocs(t *testing.T) {
	stub := []byte{0xE8, 0, 0, 0, 0, 0xC3}
	stubCallAt := 1

	obj := &tobj.Object{
		Target:  "windows-x64",
		Module:  "m",
		Code:    []byte{0xFF, 0x15, 0, 0, 0, 0, 0xC3}, // call qword ptr [rip+disp32]; ret
		Symbols: []tobj.Symbol{{Name: "main", Offset: 0}},
		Relocs:  []tobj.Reloc{{Kind: tobj.RelocIATDisp32, At: 2, Name: "kernel32.VirtualAlloc"}},
	}

	res, err := LinkX64Objects([]*tobj.Object{obj}, "main", stub, stubCallAt, 0)
	if err != nil {
		t.Fatalf("LinkX64Objects: %v", err)
	}
	if len(res.IATRelocs) != 1 {
		t.Fatalf("expected 1 IAT reloc, got %d", len(res.IATRelocs))
	}
	if res.IATRelocs[0].At != len(stub)+2 {
		t.Fatalf("IAT reloc at mismatch: got=%d want=%d", res.IATRelocs[0].At, len(stub)+2)
	}
	if res.IATRelocs[0].Name != "kernel32.VirtualAlloc" {
		t.Fatalf("IAT reloc name mismatch: %q", res.IATRelocs[0].Name)
	}
}

func TestCollectImportsSortsAndDedupes(t *testing.T) {
	imports := CollectImports(
		[]IATDisp32Reloc{{Name: "b"}, {Name: "a"}, {Name: "a"}},
		[]string{"c", "b"},
	)
	if len(imports) != 3 {
		t.Fatalf("expected 3 imports, got %d", len(imports))
	}
	if imports[0] != "a" || imports[1] != "b" || imports[2] != "c" {
		t.Fatalf("unexpected imports: %#v", imports)
	}
}

func TestLinkX64ObjectsRejectsMissingTarget(t *testing.T) {
	stub := []byte{0xE8, 0, 0, 0, 0, 0xC3}
	stubCallAt := 1
	_, err := LinkX64Objects([]*tobj.Object{
		{Module: "m", Code: []byte{0xC3}, Symbols: []tobj.Symbol{{Name: "main", Offset: 0}}},
	}, "main", stub, stubCallAt, 0)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestLinkX64ObjectsRejectsIncompatibleEntrySignature(t *testing.T) {
	stub := []byte{0xE8, 0, 0, 0, 0, 0xC3}
	stubCallAt := 1
	tests := []struct {
		name string
		sym  tobj.Symbol
	}{
		{
			name: "entry_has_params",
			sym:  tobj.Symbol{Name: "main", Offset: 0, HasSignature: true, ParamSlots: 1, ReturnSlots: 1},
		},
		{
			name: "entry_has_two_returns",
			sym:  tobj.Symbol{Name: "main", Offset: 0, HasSignature: true, ParamSlots: 0, ReturnSlots: 2},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LinkX64Objects([]*tobj.Object{
				{
					Target:  "linux-x64",
					Module:  "m",
					Code:    []byte{0xC3},
					Symbols: []tobj.Symbol{tt.sym},
				},
			}, "main", stub, stubCallAt, 0)
			if err == nil {
				t.Fatalf("expected incompatible entry signature error")
			}
			if !strings.Contains(err.Error(), "entry symbol 'main' has incompatible signature") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestLinkX64ObjectsRejectsMixedTargets(t *testing.T) {
	stub := []byte{0xE8, 0, 0, 0, 0, 0xC3}
	stubCallAt := 1
	_, err := LinkX64Objects([]*tobj.Object{
		{Target: "linux-x64", Module: "a", Code: []byte{0xC3}, Symbols: []tobj.Symbol{{Name: "main", Offset: 0}}},
		{Target: "windows-x64", Module: "b", Code: []byte{0xC3}},
	}, "main", stub, stubCallAt, 0)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestLinkX64ObjectsRejectsSymbolOffsetOutOfRange(t *testing.T) {
	stub := []byte{0xE8, 0, 0, 0, 0, 0xC3}
	stubCallAt := 1
	for _, tc := range []struct {
		name   string
		offset uint32
	}{
		{name: "one_past_end", offset: 1},
		{name: "past_end", offset: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LinkX64Objects([]*tobj.Object{
				{
					Target:  "linux-x64",
					Module:  "m",
					Code:    []byte{0xC3},
					Symbols: []tobj.Symbol{{Name: "main", Offset: tc.offset}},
				},
			}, "main", stub, stubCallAt, 0)
			if err == nil {
				t.Fatalf("expected symbol offset error, got nil")
			}
			if !strings.Contains(err.Error(), "symbol offset out of range") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestLinkX64ObjectsRejectsEmptySymbolName(t *testing.T) {
	stub := []byte{0xE8, 0, 0, 0, 0, 0xC3}
	stubCallAt := 1
	_, err := LinkX64Objects([]*tobj.Object{
		{
			Target:  "linux-x64",
			Module:  "m",
			Code:    []byte{0xC3},
			Symbols: []tobj.Symbol{{Name: "", Offset: 0}, {Name: "main", Offset: 0}},
		},
	}, "main", stub, stubCallAt, 0)
	if err == nil {
		t.Fatalf("expected empty symbol name error, got nil")
	}
	if !strings.Contains(err.Error(), "empty symbol name") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLinkX64ObjectsRejectsEmptyCallRelocName(t *testing.T) {
	stub := []byte{0xE8, 0, 0, 0, 0, 0xC3}
	stubCallAt := 1
	_, err := LinkX64Objects([]*tobj.Object{
		{
			Target:  "linux-x64",
			Module:  "m",
			Code:    []byte{0xE8, 0, 0, 0, 0, 0xC3},
			Symbols: []tobj.Symbol{{Name: "main", Offset: 0}},
			Relocs:  []tobj.Reloc{{Kind: tobj.RelocCallRel32, At: 1, Name: ""}},
		},
	}, "main", stub, stubCallAt, 0)
	if err == nil {
		t.Fatalf("expected empty call relocation symbol name error, got nil")
	}
	if !strings.Contains(err.Error(), "call relocation with empty symbol name") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLinkX64ObjectsRejectsEmptyIATRelocName(t *testing.T) {
	stub := []byte{0xE8, 0, 0, 0, 0, 0xC3}
	stubCallAt := 1
	_, err := LinkX64Objects([]*tobj.Object{
		{
			Target:  "windows-x64",
			Module:  "m",
			Code:    []byte{0xFF, 0x15, 0, 0, 0, 0, 0xC3},
			Symbols: []tobj.Symbol{{Name: "main", Offset: 0}},
			Relocs:  []tobj.Reloc{{Kind: tobj.RelocIATDisp32, At: 2, Name: ""}},
		},
	}, "main", stub, stubCallAt, 0)
	if err == nil {
		t.Fatalf("expected empty IAT relocation symbol name error, got nil")
	}
	if !strings.Contains(err.Error(), "IAT relocation with empty symbol name") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLinkX64ObjectsRejectsEmptyFunctionAddressRelocName(t *testing.T) {
	stub := []byte{0xE8, 0, 0, 0, 0, 0xC3}
	stubCallAt := 1
	tests := []struct {
		name  string
		code  []byte
		reloc tobj.Reloc
	}{
		{
			name:  "pc_relative",
			code:  []byte{0x48, 0x8D, 0x05, 0, 0, 0, 0, 0xC3},
			reloc: tobj.Reloc{Kind: tobj.RelocFuncAddrDisp32, At: 3, Name: ""},
		},
		{
			name:  "absolute",
			code:  []byte{0xB8, 0, 0, 0, 0, 0xC3},
			reloc: tobj.Reloc{Kind: tobj.RelocFuncAddrAbs32, At: 1, Name: ""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LinkX64Objects([]*tobj.Object{
				{
					Target:  "linux-x64",
					Module:  "m",
					Code:    tt.code,
					Symbols: []tobj.Symbol{{Name: "main", Offset: 0}},
					Relocs:  []tobj.Reloc{tt.reloc},
				},
			}, "main", stub, stubCallAt, 0)
			if err == nil {
				t.Fatalf("expected empty function address relocation symbol name error, got nil")
			}
			if !strings.Contains(err.Error(), "function address relocation with empty symbol name") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestLinkX64ObjectsRejectsNonDataRelocationAddends(t *testing.T) {
	stub := []byte{0xE8, 0, 0, 0, 0, 0xC3}
	stubCallAt := 1
	tests := []struct {
		name  string
		reloc tobj.Reloc
		want  string
	}{
		{
			name:  "call",
			reloc: tobj.Reloc{Kind: tobj.RelocCallRel32, At: 1, Name: "main", Addend: 7},
			want:  "call relocation addend must be zero",
		},
		{
			name:  "iat",
			reloc: tobj.Reloc{Kind: tobj.RelocIATDisp32, At: 2, Name: "kernel32.VirtualAlloc", Addend: 7},
			want:  "IAT relocation addend must be zero",
		},
		{
			name:  "function_address",
			reloc: tobj.Reloc{Kind: tobj.RelocFuncAddrDisp32, At: 1, Name: "main", Addend: 7},
			want:  "function address relocation addend must be zero",
		},
		{
			name:  "absolute_function_address",
			reloc: tobj.Reloc{Kind: tobj.RelocFuncAddrAbs32, At: 1, Name: "main", Addend: 7},
			want:  "function address relocation addend must be zero",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LinkX64Objects([]*tobj.Object{
				{
					Target:  "linux-x64",
					Module:  "m",
					Code:    []byte{0xE8, 0, 0, 0, 0, 0xC3},
					Symbols: []tobj.Symbol{{Name: "main", Offset: 0}},
					Relocs:  []tobj.Reloc{tt.reloc},
				},
			}, "main", stub, stubCallAt, 0)
			if err == nil {
				t.Fatalf("expected non-data relocation addend error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestLinkX64ObjectsRejectsNamedDataRelocation(t *testing.T) {
	stub := []byte{0xE8, 0, 0, 0, 0, 0xC3}
	stubCallAt := 1
	_, err := LinkX64Objects([]*tobj.Object{
		{
			Target:  "linux-x64",
			Module:  "m",
			Code:    []byte{0, 0, 0, 0, 0xC3},
			Data:    []byte("A"),
			Symbols: []tobj.Symbol{{Name: "main", Offset: 0}},
			Relocs:  []tobj.Reloc{{Kind: tobj.RelocDataDisp32, At: 0, Name: "data.symbol", Addend: 0}},
		},
	}, "main", stub, stubCallAt, 0)
	if err == nil {
		t.Fatalf("expected named data relocation error")
	}
	if !strings.Contains(err.Error(), "data relocation symbol name must be empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLinkX64ObjectsRandomizedCallRel32Patches(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewSource(1))
	stub := []byte{0xE8, 0, 0, 0, 0, 0xC3}
	stubCallAt := 1

	for iter := 0; iter < 50; iter++ {
		objCount := rng.Intn(6) + 2 // at least 2 objects
		mainIdx := rng.Intn(objCount)

		objects := make([]*tobj.Object, 0, objCount)
		modules := make([]string, 0, objCount)

		for i := 0; i < objCount; i++ {
			mod := fmt.Sprintf("m%02d_%08x", i, rng.Uint32())
			modules = append(modules, mod)
			if i == mainIdx {
				objects = append(objects, &tobj.Object{
					Target:  "linux-x64",
					Module:  mod,
					Code:    []byte{0xC3}, // ret
					Symbols: []tobj.Symbol{{Name: "main", Offset: 0}},
				})
				continue
			}
			objects = append(objects, &tobj.Object{
				Target:  "linux-x64",
				Module:  mod,
				Code:    []byte{0xE8, 0, 0, 0, 0, 0xC3}, // call rel32; ret
				Symbols: []tobj.Symbol{{Name: fmt.Sprintf("caller_%s", mod), Offset: 0}},
				Relocs:  []tobj.Reloc{{Kind: tobj.RelocCallRel32, At: 1, Name: "main"}},
			})
		}

		rng.Shuffle(len(objects), func(i, j int) { objects[i], objects[j] = objects[j], objects[i] })

		res, err := LinkX64Objects(objects, "main", stub, stubCallAt, 0)
		if err != nil {
			t.Fatalf("iter %d: LinkX64Objects: %v", iter, err)
		}

		mainOff, ok := res.Symbols["main"]
		if !ok {
			t.Fatalf("iter %d: missing main symbol", iter)
		}

		// Validate that every caller object has its call rel32 patched to main.
		objsSorted := append([]*tobj.Object(nil), objects...)
		sort.Slice(objsSorted, func(i, j int) bool { return objsSorted[i].Module < objsSorted[j].Module })

		textBase := len(stub)
		for _, obj := range objsSorted {
			if len(obj.Relocs) == 0 {
				textBase += len(obj.Code)
				continue
			}
			callAt := textBase + 1
			disp := readI32LE(res.Text[callAt : callAt+4])
			want := int32(mainOff - (callAt + 4))
			if disp != want {
				t.Fatalf("iter %d: module %q call rel32 mismatch: got=%d want=%d", iter, obj.Module, disp, want)
			}
			textBase += len(obj.Code)
		}
	}
}
