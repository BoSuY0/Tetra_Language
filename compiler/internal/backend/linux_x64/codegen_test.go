package linux_x64

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/format/elf"
	"tetra_language/compiler/internal/format/tobj"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/linker"
)

func TestCodegenObjectLinuxX64SetsTargetAndUsesSysVRelocs(t *testing.T) {
	obj, err := CodegenObjectLinuxX64([]ir.IRFunc{writeHelloMainFunc()})
	if err != nil {
		t.Fatalf("CodegenObjectLinuxX64: %v", err)
	}
	if obj.Target != "linux-x64" {
		t.Fatalf("target = %q, want linux-x64", obj.Target)
	}
	if len(obj.Code) == 0 {
		t.Fatalf("expected object code")
	}
	if !bytes.Contains(obj.Data, []byte("hello")) {
		t.Fatalf("data = %q, want hello literal", string(obj.Data))
	}
	if hasRelocKind(obj.Relocs, tobj.RelocIATDisp32) {
		t.Fatalf("linux object unexpectedly collected Windows IAT relocs: %#v", obj.Relocs)
	}
	if !hasSymbol(obj.Symbols, "main", 0, 1) {
		t.Fatalf("missing main symbol with expected ABI: %#v", obj.Symbols)
	}
}

func TestCodegenObjectLinuxX64EmptyStringLiteralHasNoDataReloc(t *testing.T) {
	obj, err := CodegenObjectLinuxX64([]ir.IRFunc{{
		Name:        "main",
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRStrLit, Str: nil},
			{Kind: ir.IRWrite},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
		},
	}})
	if err != nil {
		t.Fatalf("CodegenObjectLinuxX64 empty string: %v", err)
	}
	if len(obj.Data) != 0 {
		t.Fatalf("data = %q, want empty data section", string(obj.Data))
	}
	if hasRelocKind(obj.Relocs, tobj.RelocDataDisp32) {
		t.Fatalf("empty string emitted data relocation into empty data section: %#v", obj.Relocs)
	}
	if err := tobj.WriteObject(filepath.Join(t.TempDir(), "empty-string.tobj"), obj); err != nil {
		t.Fatalf("WriteObject empty-string object: %v", err)
	}
}

func TestCodegenObjectLinuxX64UsesScalarRegisterPathForSimpleAdd(t *testing.T) {
	obj, err := CodegenObjectLinuxX64([]ir.IRFunc{{
		Name:        "add",
		ParamSlots:  2,
		LocalSlots:  2,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRReturn},
		},
	}})
	if err != nil {
		t.Fatalf("CodegenObjectLinuxX64: %v", err)
	}
	for _, forbidden := range [][]byte{{0x50}, {0x58}, {0x59}} {
		if bytes.Contains(obj.Code, forbidden) {
			t.Fatalf(
				"scalar register path emitted stack-machine push/pop byte % x in code: % x",
				forbidden,
				obj.Code,
			)
		}
	}
	if !bytes.Contains(obj.Code, []byte{0x01, 0xC8}) {
		t.Fatalf("scalar register path missing add eax,ecx sequence: % x", obj.Code)
	}
}

func TestCodegenObjectLinuxX64UsesPostgreSQLFrameTypeAtRegisterPath(t *testing.T) {
	obj, err := CodegenObjectLinuxX64([]ir.IRFunc{postgresqlFrameTypeAtIRFunc()})
	if err != nil {
		t.Fatalf("CodegenObjectLinuxX64: %v", err)
	}
	wantLoad := []byte{0x0F, 0xB6, 0x04, 0x0E}
	if !bytes.Contains(obj.Code, wantLoad) {
		t.Fatalf("frame_type_at register path missing movzx [rsi+rcx]: % x", obj.Code)
	}
	for _, forbidden := range [][]byte{{0x50}, {0x58}, {0x59}} {
		if bytes.Contains(obj.Code, forbidden) {
			t.Fatalf(
				"frame_type_at register path emitted stack-machine push/pop byte % x in code: % x",
				forbidden,
				obj.Code,
			)
		}
	}
}

func TestPostgreSQLFrameTypeAtLinuxX64NativeHelperMatchesStackFallback(t *testing.T) {
	funcs := []ir.IRFunc{
		postgresqlFrameTypeAtIRFunc(),
		mainCallsPostgreSQLFrameTypeAtIRFunc(),
	}
	fast := buildLinkRunLinuxX64(
		t,
		funcs,
		x64.CodegenOptions{},
		"postgresql-frame-type-at-fast",
	)
	stack := buildLinkRunLinuxX64(
		t,
		funcs,
		x64.CodegenOptions{DisableMachinePaths: true},
		"postgresql-frame-type-at-stack",
	)
	if fast != 42 || stack != 42 {
		t.Fatalf("frame_type_at fast exit = %d, stack fallback exit = %d, want both 42", fast, stack)
	}
}

func TestCodegenObjectLinuxX64UsesRegisterLoopPathForSumN(t *testing.T) {
	obj, err := CodegenObjectLinuxX64([]ir.IRFunc{sumNIRFunc()})
	if err != nil {
		t.Fatalf("CodegenObjectLinuxX64: %v", err)
	}
	for _, forbidden := range [][]byte{{0x50}, {0x58}, {0x59}} {
		if bytes.Contains(obj.Code, forbidden) {
			t.Fatalf(
				"loop register path emitted stack-machine push/pop byte % x in code: % x",
				forbidden,
				obj.Code,
			)
		}
	}
	for _, want := range [][]byte{
		{0x39, 0xCA},       // cmp edx,ecx
		{0x01, 0xC8},       // add eax,ecx
		{0x83, 0xC1, 0x01}, // add ecx,1
	} {
		if !bytes.Contains(obj.Code, want) {
			t.Fatalf("loop register path missing % x sequence: % x", want, obj.Code)
		}
	}
}

func TestCodegenObjectLinuxX64UsesRegisterLoopPathForConstantStride(t *testing.T) {
	obj, err := CodegenObjectLinuxX64([]ir.IRFunc{sumStrideIRFunc()})
	if err != nil {
		t.Fatalf("CodegenObjectLinuxX64: %v", err)
	}
	for _, want := range [][]byte{
		{0x39, 0xCA},       // cmp edx,ecx
		{0x01, 0xC8},       // add eax,ecx
		{0x83, 0xC1, 0x02}, // add ecx,2
	} {
		if !bytes.Contains(obj.Code, want) {
			t.Fatalf("constant-stride register loop missing % x sequence: % x", want, obj.Code)
		}
	}
}

func TestCodegenObjectLinuxX64UsesRegisterConstModuloLoopPathForIntegerLoops(t *testing.T) {
	obj, err := CodegenObjectLinuxX64([]ir.IRFunc{integerLoopsBenchmarkMainIRFunc()})
	if err != nil {
		t.Fatalf("CodegenObjectLinuxX64: %v", err)
	}
	for _, forbidden := range [][]byte{{0x50}, {0x58}, {0x59}} {
		if bytes.Contains(obj.Code, forbidden) {
			t.Fatalf(
				"const-modulo loop register path emitted stack-machine push/pop byte % x in code: % x",
				forbidden,
				obj.Code,
			)
		}
	}
	for _, want := range [][]byte{
		{0x48, 0x81, 0xF9, 0x40, 0x0D, 0x03, 0x00}, // cmp rcx,200000
		{0x41, 0xF7, 0xF8},                         // idiv r8d
		{0x41, 0x01, 0xD2},                         // add r10d,edx
		{0x83, 0xC1, 0x01},                         // add ecx,1
	} {
		if !bytes.Contains(obj.Code, want) {
			t.Fatalf("const-modulo loop register path missing % x sequence: % x", want, obj.Code)
		}
	}
}

func TestCodegenObjectLinuxX64RegisterLoopMatchesStackFallback(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}
	funcs := []ir.IRFunc{sumNIRFunc(), mainCallsSumNIRFunc()}
	fast := buildLinkRunLinuxX64(t, funcs, x64.CodegenOptions{}, "fast-loop")
	stack := buildLinkRunLinuxX64(
		t,
		funcs,
		x64.CodegenOptions{DisableMachinePaths: true},
		"stack-fallback",
	)
	if fast != stack {
		t.Fatalf("fast loop exit = %d, stack fallback exit = %d", fast, stack)
	}
	if fast != 45 {
		t.Fatalf("sum_n(10) exit = %d, want 45", fast)
	}
}

func TestCodegenObjectLinuxX64RegisterStrideLoopMatchesStackFallback(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}
	funcs := []ir.IRFunc{sumStrideIRFunc(), mainCallsSumStrideIRFunc()}
	fast := buildLinkRunLinuxX64(t, funcs, x64.CodegenOptions{}, "fast-stride-loop")
	stack := buildLinkRunLinuxX64(
		t,
		funcs,
		x64.CodegenOptions{DisableMachinePaths: true},
		"stack-stride-loop",
	)
	if fast != stack {
		t.Fatalf("fast stride loop exit = %d, stack fallback exit = %d", fast, stack)
	}
	if fast != 20 {
		t.Fatalf("sum_stride(10) exit = %d, want 20", fast)
	}
}

func TestCodegenObjectLinuxX64RegisterConstModuloLoopMatchesStackFallback(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}
	funcs := []ir.IRFunc{integerLoopsBenchmarkMainIRFunc()}
	fast := buildLinkRunLinuxX64(t, funcs, x64.CodegenOptions{}, "fast-const-modulo-loop")
	stack := buildLinkRunLinuxX64(
		t,
		funcs,
		x64.CodegenOptions{DisableMachinePaths: true},
		"stack-const-modulo-loop",
	)
	if fast != stack {
		t.Fatalf("fast const-modulo loop exit = %d, stack fallback exit = %d", fast, stack)
	}
	if fast != 0 {
		t.Fatalf("integer_loops benchmark exit = %d, want 0", fast)
	}
}

func TestCodegenObjectLinuxX64AllocationLoopNativePathMatchesStackFallback(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}
	funcs := []ir.IRFunc{allocationLoopIRFunc()}
	fastObj, err := CodegenObjectLinuxX64(funcs)
	if err != nil {
		t.Fatalf("CodegenObjectLinuxX64: %v", err)
	}
	for _, forbidden := range [][]byte{{0x50}, {0x58}, {0x59}} {
		if bytes.Contains(fastObj.Code, forbidden) {
			t.Fatalf(
				"allocation-loop native path emitted stack-machine push/pop byte % x in code: % x",
				forbidden,
				fastObj.Code,
			)
		}
	}
	for _, want := range [][]byte{
		{0x48, 0x81, 0xF9, 0x00, 0x04, 0x00, 0x00}, // cmp rcx,1024
		{0x0F, 0x83}, // checked bounds failure branch
		{0x89, 0x08}, // mov [rax],ecx for xs[0] = r
		{0x8B, 0x00}, // mov eax,[rax] for xs[0]
	} {
		if !bytes.Contains(fastObj.Code, want) {
			t.Fatalf("allocation-loop native path missing % x sequence: % x", want, fastObj.Code)
		}
	}

	fast := buildLinkRunLinuxX64(t, funcs, x64.CodegenOptions{}, "fast-allocation-loop")
	stack := buildLinkRunLinuxX64(
		t,
		funcs,
		x64.CodegenOptions{DisableMachinePaths: true},
		"stack-allocation-loop",
	)
	if fast != stack {
		t.Fatalf("fast allocation loop exit = %d, stack fallback exit = %d", fast, stack)
	}
	if fast != 0 {
		t.Fatalf("allocation benchmark exit = %d, want 0", fast)
	}
}

func TestCodegenObjectLinuxX64RegionIslandAllocationMainNativePathMatchesStackFallback(
	t *testing.T,
) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}
	funcs := []ir.IRFunc{regionIslandAllocationMainIRFunc()}
	fastObj, err := CodegenObjectLinuxX64(funcs)
	if err != nil {
		t.Fatalf("CodegenObjectLinuxX64: %v", err)
	}
	for _, forbidden := range [][]byte{{0x50}, {0x58}, {0x59}} {
		if bytes.Contains(fastObj.Code, forbidden) {
			t.Fatalf(
				"region_island_allocation native path emitted stack-machine push/pop byte % x in code: % x",
				forbidden,
				fastObj.Code,
			)
		}
	}
	loopBound := &x64.Emitter{}
	loopBound.CmpRcxImm32(256)
	for _, want := range [][]byte{
		loopBound.Buf,
		{0x89, 0x08},
		{0x8B, 0x00},
	} {
		if !bytes.Contains(fastObj.Code, want) {
			t.Fatalf("region_island_allocation native path missing % x sequence: % x", want, fastObj.Code)
		}
	}

	fast := buildLinkRunLinuxX64(
		t,
		funcs,
		x64.CodegenOptions{},
		"fast-region-island-allocation-main",
	)
	stack := buildLinkRunLinuxX64(
		t,
		funcs,
		x64.CodegenOptions{DisableMachinePaths: true},
		"stack-region-island-allocation-main",
	)
	if fast != stack {
		t.Fatalf("fast region_island_allocation exit = %d, stack fallback exit = %d", fast, stack)
	}
	if fast != 0 {
		t.Fatalf("region_island_allocation benchmark exit = %d, want 0", fast)
	}
}

func TestCodegenObjectLinuxX64BoundsCheckLoopsUsesNativePath(t *testing.T) {
	obj, err := CodegenObjectLinuxX64([]ir.IRFunc{boundsCheckLoopsIRFunc()})
	if err != nil {
		t.Fatalf("CodegenObjectLinuxX64: %v", err)
	}
	for _, forbidden := range [][]byte{{0x50}, {0x58}, {0x59}} {
		if bytes.Contains(obj.Code, forbidden) {
			t.Fatalf(
				"bounds-check-loops native path emitted stack-machine push/pop byte % x in code: % x",
				forbidden,
				obj.Code,
			)
		}
	}
	for _, want := range [][]byte{
		{0x48, 0x81, 0xF9, 0x00, 0x10, 0x00, 0x00}, // cmp rcx,4096
		{0x48, 0x81, 0xF9, 0x40, 0x0D, 0x03, 0x00}, // cmp rcx,200000
		{0x41, 0xF7, 0xF8},                         // idiv r8d
		{0x44, 0x89, 0x00},                         // mov [rax],r8d
		{0x8B, 0x00},                               // mov eax,[rax]
	} {
		if !bytes.Contains(obj.Code, want) {
			t.Fatalf("bounds-check-loops native path missing % x sequence: % x", want, obj.Code)
		}
	}
}

func TestCodegenObjectLinuxX64BoundsCheckLoopsNativePathMatchesStackFallback(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}
	funcs := []ir.IRFunc{boundsCheckLoopsIRFunc()}
	fast := buildLinkRunLinuxX64(t, funcs, x64.CodegenOptions{}, "fast-bounds-check-loops")
	stack := buildLinkRunLinuxX64(
		t,
		funcs,
		x64.CodegenOptions{DisableMachinePaths: true},
		"stack-bounds-check-loops",
	)
	if fast != stack {
		t.Fatalf("fast bounds_check_loops exit = %d, stack fallback exit = %d", fast, stack)
	}
	if fast != 0 {
		t.Fatalf("bounds_check_loops benchmark exit = %d, want 0", fast)
	}
}

func TestCodegenObjectLinuxX64MatrixMultiplyMainNativePathMatchesStackFallback(t *testing.T) {
	funcs := []ir.IRFunc{matrixMultiplyMainIRFunc()}
	fastObj, err := CodegenObjectLinuxX64(funcs)
	if err != nil {
		t.Fatalf("CodegenObjectLinuxX64: %v", err)
	}
	for _, forbidden := range [][]byte{{0x50}, {0x58}, {0x59}} {
		if bytes.Contains(fastObj.Code, forbidden) {
			t.Fatalf(
				"matrix_multiply native path emitted stack-machine push/pop byte % x in code: % x",
				forbidden,
				fastObj.Code,
			)
		}
	}
	repeat := &x64.Emitter{}
	repeat.CmpRcxImm32(2000)
	dimension := &x64.Emitter{}
	dimension.CmpEdxImm32(3)
	for _, want := range [][]byte{
		repeat.Buf,
		dimension.Buf,
	} {
		if !bytes.Contains(fastObj.Code, want) {
			t.Fatalf("matrix_multiply native path missing % x sequence: % x", want, fastObj.Code)
		}
	}

	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}
	fast := buildLinkRunLinuxX64(t, funcs, x64.CodegenOptions{}, "fast-matrix-multiply-main")
	stack := buildLinkRunLinuxX64(
		t,
		funcs,
		x64.CodegenOptions{DisableMachinePaths: true},
		"stack-matrix-multiply-main",
	)
	if fast != stack {
		t.Fatalf("fast matrix_multiply exit = %d, stack fallback exit = %d", fast, stack)
	}
	if fast != 0 {
		t.Fatalf("matrix_multiply benchmark exit = %d, want 0", fast)
	}
}

func TestCodegenObjectLinuxX64RegisterDivModMatchesStackFallback(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}
	funcs := []ir.IRFunc{divModIRFunc(), mainCallsDivModIRFunc()}
	fast := buildLinkRunLinuxX64(t, funcs, x64.CodegenOptions{}, "fast-div-mod")
	stack := buildLinkRunLinuxX64(
		t,
		funcs,
		x64.CodegenOptions{DisableMachinePaths: true},
		"stack-div-mod",
	)
	if fast != stack {
		t.Fatalf("fast div/mod exit = %d, stack fallback exit = %d", fast, stack)
	}
	if fast != 13 {
		t.Fatalf("div_mod exit = %d, want 13", fast)
	}
}

func TestCodegenObjectLinuxX64UsesVectorSliceSumPathForProofLoop(t *testing.T) {
	obj, err := CodegenObjectLinuxX64([]ir.IRFunc{sumSliceIRFunc(true)})
	if err != nil {
		t.Fatalf("CodegenObjectLinuxX64: %v", err)
	}
	for _, forbidden := range [][]byte{{0x50}, {0x58}, {0x59}} {
		if bytes.Contains(obj.Code, forbidden) {
			t.Fatalf(
				"slice-sum vector path emitted stack-machine push/pop byte % x in code: % x",
				forbidden,
				obj.Code,
			)
		}
	}
	for _, want := range [][]byte{
		{0x66, 0x0F, 0xEF, 0xC9},             // pxor xmm1,xmm1
		{0xF3, 0x41, 0x0F, 0x6F, 0x04, 0x89}, // movdqu xmm0,[r9+rcx*4]
		{0x66, 0x0F, 0xFE, 0xC8},             // paddd xmm1,xmm0
		{0x66, 0x0F, 0x70, 0xC1, 0x4E},       // pshufd xmm0,xmm1,0x4e
		{0x66, 0x0F, 0x7E, 0xC8},             // movd eax,xmm1
	} {
		if !bytes.Contains(obj.Code, want) {
			t.Fatalf("slice-sum vector path missing % x sequence: % x", want, obj.Code)
		}
	}
}

func TestCodegenObjectLinuxX64VectorSliceSumMatchesStackFallbackWithTail(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}
	funcs := []ir.IRFunc{sumSliceIRFunc(true), mainCallsSumSliceTailIRFunc()}
	fast := buildLinkRunLinuxX64(t, funcs, x64.CodegenOptions{}, "fast-vector-slice-sum-tail")
	stack := buildLinkRunLinuxX64(
		t,
		funcs,
		x64.CodegenOptions{DisableMachinePaths: true},
		"stack-vector-slice-sum-tail",
	)
	if fast != stack {
		t.Fatalf("fast vector slice sum exit = %d, stack fallback exit = %d", fast, stack)
	}
	if fast != 28 {
		t.Fatalf("sum([1..7]) exit = %d, want 28", fast)
	}
}

func TestCodegenObjectLinuxX64UsesVectorCopyU8PathForProofLoop(t *testing.T) {
	obj, err := CodegenObjectLinuxX64([]ir.IRFunc{copyU8IRFunc(true)})
	if err != nil {
		t.Fatalf("CodegenObjectLinuxX64: %v", err)
	}
	for _, forbidden := range [][]byte{{0x50}, {0x58}, {0x59}} {
		if bytes.Contains(obj.Code, forbidden) {
			t.Fatalf(
				"copy-u8 vector path emitted stack-machine push/pop byte % x in code: % x",
				forbidden,
				obj.Code,
			)
		}
	}
	for _, want := range [][]byte{
		{0xF3, 0x41, 0x0F, 0x6F, 0x04, 0x09}, // movdqu xmm0,[r9+rcx]
		{0xF3, 0x0F, 0x7F, 0x04, 0x0F},       // movdqu [rdi+rcx],xmm0
		{0x83, 0xC1, 0x10},                   // add ecx,16
		{0x0F, 0xB6, 0x04, 0x0E},             // movzx eax,byte ptr [rsi+rcx]
		{0x88, 0x04, 0x0F},                   // mov [rdi+rcx],al
	} {
		if !bytes.Contains(obj.Code, want) {
			t.Fatalf("copy-u8 vector path missing % x sequence: % x", want, obj.Code)
		}
	}
}

func TestCodegenObjectLinuxX64VectorCopyU8MatchesStackFallbackWithTail(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}
	funcs := []ir.IRFunc{copyU8IRFunc(true), mainCallsCopyU8TailIRFunc()}
	fast := buildLinkRunLinuxX64(t, funcs, x64.CodegenOptions{}, "fast-vector-copy-u8-tail")
	stack := buildLinkRunLinuxX64(
		t,
		funcs,
		x64.CodegenOptions{DisableMachinePaths: true},
		"stack-vector-copy-u8-tail",
	)
	if fast != stack {
		t.Fatalf("fast vector copy_u8 exit = %d, stack fallback exit = %d", fast, stack)
	}
	if fast != 53 {
		t.Fatalf("copy_u8 selected dst bytes exit = %d, want 53", fast)
	}
}

func TestCodegenObjectLinuxX64UsesVectorMapI32AddConstPathForProofLoop(t *testing.T) {
	obj, err := CodegenObjectLinuxX64([]ir.IRFunc{mapAddI32IRFunc(true)})
	if err != nil {
		t.Fatalf("CodegenObjectLinuxX64: %v", err)
	}
	for _, forbidden := range [][]byte{{0x50}, {0x58}, {0x59}} {
		if bytes.Contains(obj.Code, forbidden) {
			t.Fatalf(
				"map-i32 vector path emitted stack-machine push/pop byte % x in code: % x",
				forbidden,
				obj.Code,
			)
		}
	}
	for _, want := range [][]byte{
		{0x66, 0x0F, 0x6E, 0xC8},             // movd xmm1,eax
		{0x66, 0x0F, 0x70, 0xC9, 0x00},       // pshufd xmm1,xmm1,0
		{0xF3, 0x41, 0x0F, 0x6F, 0x04, 0x89}, // movdqu xmm0,[r9+rcx*4]
		{0x66, 0x0F, 0xFE, 0xC1},             // paddd xmm0,xmm1
		{0xF3, 0x41, 0x0F, 0x7F, 0x04, 0x89}, // movdqu [r9+rcx*4],xmm0
		{0x83, 0xC1, 0x04},                   // add ecx,4
	} {
		if !bytes.Contains(obj.Code, want) {
			t.Fatalf("map-i32 vector path missing % x sequence: % x", want, obj.Code)
		}
	}
}

func TestCodegenObjectLinuxX64VectorMapI32AddConstMatchesStackFallbackWithTail(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}
	funcs := []ir.IRFunc{mapAddI32IRFunc(true), mainCallsMapAddI32TailIRFunc()}
	fast := buildLinkRunLinuxX64(t, funcs, x64.CodegenOptions{}, "fast-vector-map-i32-tail")
	stack := buildLinkRunLinuxX64(
		t,
		funcs,
		x64.CodegenOptions{DisableMachinePaths: true},
		"stack-vector-map-i32-tail",
	)
	if fast != stack {
		t.Fatalf("fast vector map_i32 exit = %d, stack fallback exit = %d", fast, stack)
	}
	if fast != 21 {
		t.Fatalf("map_i32 selected updated values exit = %d, want 21", fast)
	}
}

func TestCodegenObjectLinuxX64UsesVectorMemsetZeroU8PathForProofHelper(t *testing.T) {
	obj, err := CodegenObjectLinuxX64([]ir.IRFunc{memsetZeroU8IRFunc(true)})
	if err != nil {
		t.Fatalf("CodegenObjectLinuxX64: %v", err)
	}
	for _, forbidden := range [][]byte{{0x50}, {0x58}, {0x59}} {
		if bytes.Contains(obj.Code, forbidden) {
			t.Fatalf(
				"memset-zero-u8 vector path emitted stack-machine push/pop byte % x in code: % x",
				forbidden,
				obj.Code,
			)
		}
	}
	for _, want := range [][]byte{
		{0x66, 0x0F, 0xEF, 0xC0},       // pxor xmm0,xmm0
		{0xF3, 0x0F, 0x7F, 0x04, 0x0F}, // movdqu [rdi+rcx],xmm0
		{0x83, 0xC1, 0x10},             // add ecx,16
		{0x88, 0x04, 0x0F},             // mov [rdi+rcx],al
	} {
		if !bytes.Contains(obj.Code, want) {
			t.Fatalf("memset-zero-u8 vector path missing % x sequence: % x", want, obj.Code)
		}
	}
}

func TestCodegenObjectLinuxX64VectorMemsetZeroU8MatchesStackFallbackWithTail(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}
	funcs := []ir.IRFunc{memsetZeroU8IRFunc(true), mainCallsMemsetZeroU8TailIRFunc()}
	fast := buildLinkRunLinuxX64(t, funcs, x64.CodegenOptions{}, "fast-vector-memset-zero-u8-tail")
	stack := buildLinkRunLinuxX64(
		t,
		funcs,
		x64.CodegenOptions{DisableMachinePaths: true},
		"stack-vector-memset-zero-u8-tail",
	)
	if fast != stack {
		t.Fatalf("fast vector memset_zero_u8 exit = %d, stack fallback exit = %d", fast, stack)
	}
	if fast != 0 {
		t.Fatalf("memset_zero_u8 selected zeroed bytes exit = %d, want 0", fast)
	}
}

func TestCodegenObjectLinuxX64UsesRegisterSliceSumPathForProofConstantStrideLoop(t *testing.T) {
	obj, err := CodegenObjectLinuxX64([]ir.IRFunc{sumSliceStrideIRFunc(true, 2)})
	if err != nil {
		t.Fatalf("CodegenObjectLinuxX64: %v", err)
	}
	for _, want := range [][]byte{
		{0x39, 0xCA},             // cmp edx,ecx
		{0x45, 0x8B, 0x04, 0x89}, // mov r8d,[r9+rcx*4]
		{0x45, 0x01, 0xC2},       // add r10d,r8d
		{0x83, 0xC1, 0x02},       // add ecx,2
	} {
		if !bytes.Contains(obj.Code, want) {
			t.Fatalf("slice-stride register path missing % x sequence: % x", want, obj.Code)
		}
	}
}

func TestCodegenObjectLinuxX64SliceSumWithoutProofUsesCheckedStackFallback(t *testing.T) {
	obj, err := CodegenObjectLinuxX64([]ir.IRFunc{sumSliceIRFunc(false)})
	if err != nil {
		t.Fatalf("CodegenObjectLinuxX64: %v", err)
	}
	for _, want := range [][]byte{
		{0x50},
		{0x58},
		{0x59},
		{0x0F, 0x83}, // checked bounds failure branch
	} {
		if !bytes.Contains(obj.Code, want) {
			t.Fatalf("checked slice-sum fallback missing % x sequence: % x", want, obj.Code)
		}
	}
}

func TestCodegenObjectLinuxX64UsesRegisterCallPathForNestedCalls(t *testing.T) {
	obj, err := CodegenObjectLinuxX64([]ir.IRFunc{incIRFunc(), nestedCallMainIRFunc()})
	if err != nil {
		t.Fatalf("CodegenObjectLinuxX64: %v", err)
	}
	for _, forbidden := range [][]byte{{0x50}, {0x58}, {0x59}} {
		if bytes.Contains(obj.Code, forbidden) {
			t.Fatalf(
				"register call path emitted stack-machine push/pop byte % x in code: % x",
				forbidden,
				obj.Code,
			)
		}
	}
	if !bytes.Contains(obj.Code, []byte{0xE8}) {
		t.Fatalf("register call path missing direct call instruction: % x", obj.Code)
	}
	if len(obj.Relocs) != 0 {
		t.Fatalf("local scalar calls should be patched in-object, relocs=%#v", obj.Relocs)
	}
}

func TestCodegenObjectLinuxX64RegisterCallsMatchStackFallback(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}
	funcs := []ir.IRFunc{incIRFunc(), nestedCallMainIRFunc()}
	fast := buildLinkRunLinuxX64(t, funcs, x64.CodegenOptions{}, "fast-call")
	stack := buildLinkRunLinuxX64(
		t,
		funcs,
		x64.CodegenOptions{DisableMachinePaths: true},
		"stack-call",
	)
	if fast != stack {
		t.Fatalf("fast call exit = %d, stack fallback exit = %d", fast, stack)
	}
	if fast != 42 {
		t.Fatalf("nested inc exit = %d, want 42", fast)
	}
}

func TestCodegenObjectLinuxX64RegisterCallKeepsScratchValuesAcrossCall(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}
	funcs := []ir.IRFunc{incIRFunc(), callWithLiveScratchMainIRFunc()}
	fast := buildLinkRunLinuxX64(t, funcs, x64.CodegenOptions{}, "fast-call-spill")
	stack := buildLinkRunLinuxX64(
		t,
		funcs,
		x64.CodegenOptions{DisableMachinePaths: true},
		"stack-call-spill",
	)
	if fast != stack {
		t.Fatalf("fast call-spill exit = %d, stack fallback exit = %d", fast, stack)
	}
	if fast != 43 {
		t.Fatalf("call with live scratch exit = %d, want 43", fast)
	}
}

func TestCodegenObjectLinuxX64RegisterCallLoopMatchesStackFallback(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}
	funcs := []ir.IRFunc{incIRFunc(), sumCallLoopIRFunc(), mainCallsSumCallIRFunc()}
	fastObj, err := CodegenObjectLinuxX64(funcs)
	if err != nil {
		t.Fatalf("CodegenObjectLinuxX64: %v", err)
	}
	for _, forbidden := range [][]byte{{0x50}, {0x58}, {0x59}} {
		if bytes.Contains(fastObj.Code, forbidden) {
			t.Fatalf(
				"register call-loop path emitted stack-machine push/pop byte % x in code: % x",
				forbidden,
				fastObj.Code,
			)
		}
	}
	if !bytes.Contains(fastObj.Code, []byte{0xE8}) {
		t.Fatalf("register call-loop path missing direct call instruction: % x", fastObj.Code)
	}

	fast := buildLinkRunLinuxX64(t, funcs, x64.CodegenOptions{}, "fast-call-loop")
	stack := buildLinkRunLinuxX64(
		t,
		funcs,
		x64.CodegenOptions{DisableMachinePaths: true},
		"stack-call-loop",
	)
	if fast != stack {
		t.Fatalf("fast call-loop exit = %d, stack fallback exit = %d", fast, stack)
	}
	if fast != 55 {
		t.Fatalf("sum_call(10) exit = %d, want 55", fast)
	}
}

func TestCodegenObjectLinuxX64RegisterTwoArgConstBoundCallLoopMatchesStackFallback(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}
	funcs := []ir.IRFunc{mixTwoArgIRFunc(), functionCallsBenchmarkMainIRFunc()}
	fastObj, err := CodegenObjectLinuxX64(funcs)
	if err != nil {
		t.Fatalf("CodegenObjectLinuxX64: %v", err)
	}
	for _, forbidden := range [][]byte{{0x50}, {0x58}, {0x59}} {
		if bytes.Contains(fastObj.Code, forbidden) {
			t.Fatalf(
				"register two-arg call-loop path emitted stack-machine push/pop byte % x in code: % x",
				forbidden,
				fastObj.Code,
			)
		}
	}
	if !bytes.Contains(fastObj.Code, []byte{0xE8}) {
		t.Fatalf(
			"register two-arg call-loop path missing direct call instruction: % x",
			fastObj.Code,
		)
	}

	fast := buildLinkRunLinuxX64(t, funcs, x64.CodegenOptions{}, "fast-two-arg-call-loop")
	stack := buildLinkRunLinuxX64(
		t,
		funcs,
		x64.CodegenOptions{DisableMachinePaths: true},
		"stack-two-arg-call-loop",
	)
	if fast != stack {
		t.Fatalf("fast two-arg call-loop exit = %d, stack fallback exit = %d", fast, stack)
	}
	if fast != 0 {
		t.Fatalf("function_calls benchmark main exit = %d, want 0", fast)
	}
}

func TestCodegenObjectLinuxX64RegisterCompileTimeCallLoopMatchesStackFallback(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}
	funcs := []ir.IRFunc{
		compileTimeF0IRFunc(),
		compileTimeF1IRFunc(),
		compileTimeF2IRFunc(),
		compileTimeBenchmarkMainIRFunc(),
	}
	fastObj, err := CodegenObjectLinuxX64(funcs)
	if err != nil {
		t.Fatalf("CodegenObjectLinuxX64: %v", err)
	}
	for _, forbidden := range [][]byte{{0x50}, {0x58}, {0x59}} {
		if bytes.Contains(fastObj.Code, forbidden) {
			t.Fatalf(
				"register compile_time call-loop path emitted stack-machine push/pop byte % x in code: % x",
				forbidden,
				fastObj.Code,
			)
		}
	}
	for _, want := range [][]byte{
		{0xE8},                         // direct call rel32 patches for f1/f2/main calls.
		{0x3D, 0x00, 0x00, 0x00, 0x00}, // cmp eax,0 for total == 0 final guard.
	} {
		if !bytes.Contains(fastObj.Code, want) {
			t.Fatalf("compile_time native path missing % x sequence: % x", want, fastObj.Code)
		}
	}

	fast := buildLinkRunLinuxX64(t, funcs, x64.CodegenOptions{}, "fast-compile-time-call-loop")
	stack := buildLinkRunLinuxX64(
		t,
		funcs,
		x64.CodegenOptions{DisableMachinePaths: true},
		"stack-compile-time-call-loop",
	)
	if fast != stack {
		t.Fatalf("fast compile_time exit = %d, stack fallback exit = %d", fast, stack)
	}
	if fast != 0 {
		t.Fatalf("compile_time benchmark main exit = %d, want 0", fast)
	}
}

func TestCodegenObjectLinuxX64RecursionFibNativeMatchesStackFallback(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}
	funcs := []ir.IRFunc{recursionFibIRFunc(), mainCallsRecursionFibIRFunc(10)}
	fast := buildLinkRunLinuxX64(t, funcs, x64.CodegenOptions{}, "fast-recursion-fib")
	stack := buildLinkRunLinuxX64(
		t,
		funcs,
		x64.CodegenOptions{DisableMachinePaths: true},
		"stack-recursion-fib",
	)
	if fast != stack {
		t.Fatalf("fast fib exit = %d, stack fallback exit = %d", fast, stack)
	}
	if fast != 55 {
		t.Fatalf("fib(10) exit = %d, want 55", fast)
	}
}

func TestCodegenObjectLinuxX64RecursionBenchmarkNativeMatchesStackFallback(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}
	funcs := []ir.IRFunc{recursionFibIRFunc(), recursionBenchmarkMainIRFunc()}
	fastObj, err := CodegenObjectLinuxX64(funcs)
	if err != nil {
		t.Fatalf("CodegenObjectLinuxX64: %v", err)
	}
	for _, want := range [][]byte{
		{0xE8},                         // real call rel32 patches for fib recursion and main -> fib.
		{0x3D, 0x02, 0x00, 0x00, 0x00}, // cmp eax,2 in fib base guard.
		{0x0F, 0x8C},                   // jl rel32 for fib base return.
		{0x48, 0x81, 0xF9, 0x28, 0x00, 0x00, 0x00}, // cmp rcx,40 in main loop.
		{0x3D, 0x98, 0x08, 0x00, 0x00},             // cmp eax,2200 in main final guard.
	} {
		if !bytes.Contains(fastObj.Code, want) {
			t.Fatalf("recursion native path missing % x sequence: % x", want, fastObj.Code)
		}
	}
	if got := countBytes(fastObj.Code, []byte{0xE8}); got < 3 {
		t.Fatalf(
			"recursion native path call rel32 count = %d, want at least fib self-calls plus main call: % x",
			got,
			fastObj.Code,
		)
	}
	if len(fastObj.Relocs) != 0 {
		t.Fatalf("local recursion calls should be patched in-object, relocs=%#v", fastObj.Relocs)
	}

	fast := buildLinkRunLinuxX64(t, funcs, x64.CodegenOptions{}, "fast-recursion-benchmark")
	stack := buildLinkRunLinuxX64(
		t,
		funcs,
		x64.CodegenOptions{DisableMachinePaths: true},
		"stack-recursion-benchmark",
	)
	if fast != stack {
		t.Fatalf("fast recursion exit = %d, stack fallback exit = %d", fast, stack)
	}
	if fast != 0 {
		t.Fatalf("recursion benchmark exit = %d, want 0", fast)
	}
}

func TestCodegenObjectLinuxX64UsesSharedSmallHeapAllocatorForMakeSlices(t *testing.T) {
	obj, err := CodegenObjectLinuxX64([]ir.IRFunc{twoSmallMakeSlicesMainIRFunc()})
	if err != nil {
		t.Fatalf("CodegenObjectLinuxX64: %v", err)
	}
	if got := countBytes(obj.Code, []byte{0xB8, 0x09, 0x00, 0x00, 0x00, 0x0F, 0x05}); got != 2 {
		t.Fatalf(
			"small heap helper should contain chunk refill plus large-fallback mmap sites, got %d\ncode=% x",
			got,
			obj.Code,
		)
	}
	if got := countBytes(obj.Code, []byte{0xB8, 0x0A, 0x00, 0x00, 0x00, 0x0F, 0x05}); got != 2 {
		t.Fatalf(
			"small heap helper should commit chunk refill plus large fallback through mprotect, got %d\ncode=% x",
			got,
			obj.Code,
		)
	}
	if got := countBytes(obj.Code, []byte{0xE8}); got < 2 {
		t.Fatalf(
			"small make-slice sites should call the shared helper, helper calls=%d\ncode=% x",
			got,
			obj.Code,
		)
	}
	if !hasRelocKind(obj.Relocs, tobj.RelocDataDisp32) {
		t.Fatalf(
			"small heap helper should reference writable allocator state data, relocs=%#v",
			obj.Relocs,
		)
	}
	if len(obj.Data) < 16 {
		t.Fatalf("small heap helper data size = %d, want at least bump/end state", len(obj.Data))
	}
}

func TestCodegenObjectLinuxX64MemoryBackendRuntimeSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}
	if got := buildLinkRunLinuxX64(
		t,
		[]ir.IRFunc{memoryBackendSmokeMainIRFunc()},
		x64.CodegenOptions{},
		"memory-backend-smoke",
	); got != 0 {
		t.Fatalf("memory backend smoke exit = %d, want 0", got)
	}
}

func TestCodegenObjectLinuxX64SmallHeapMakeSlicesRunAndDoNotOverlap(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}
	if got := buildLinkRunLinuxX64(
		t,
		[]ir.IRFunc{twoSmallMakeSlicesMainIRFunc()},
		x64.CodegenOptions{},
		"small-heap-two-slices",
	); got != 30 {
		t.Fatalf("small heap two-slice exit = %d, want 30", got)
	}
	if got := buildLinkRunLinuxX64(
		t,
		[]ir.IRFunc{manySmallMakeSlicesMainIRFunc()},
		x64.CodegenOptions{},
		"small-heap-many-slices",
	); got != 200 {
		t.Fatalf("small heap stress exit = %d, want 200", got)
	}
	if got := buildLinkRunLinuxX64(
		t,
		[]ir.IRFunc{refillSmallHeapMakeSlicesMainIRFunc()},
		x64.CodegenOptions{},
		"small-heap-refill-slices",
	); got != 42 {
		t.Fatalf("small heap refill stress exit = %d, want 42", got)
	}
}

func TestCodegenObjectLinuxX64SmallHeapLargeMakeSliceFallbackRuns(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}
	if got := buildLinkRunLinuxX64(
		t,
		[]ir.IRFunc{largeMakeSliceMainIRFunc()},
		x64.CodegenOptions{},
		"small-heap-large-fallback",
	); got != 42 {
		t.Fatalf("large make-slice fallback exit = %d, want 42", got)
	}
}

func buildLinkRunLinuxX64(
	t *testing.T,
	funcs []ir.IRFunc,
	opt x64.CodegenOptions,
	name string,
) int {
	t.Helper()
	obj, err := CodegenObjectLinuxX64WithOptions(funcs, opt)
	if err != nil {
		t.Fatalf("%s CodegenObjectLinuxX64WithOptions: %v", name, err)
	}
	img, err := linker.LinkLinuxX64([]*tobj.Object{obj}, "main")
	if err != nil {
		t.Fatalf("%s LinkLinuxX64: %v", name, err)
	}
	path := filepath.Join(t.TempDir(), name)
	if err := elf.WriteELF64LinuxX64(path, img); err != nil {
		t.Fatalf("%s WriteELF64LinuxX64: %v", name, err)
	}
	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatalf("%s chmod: %v", name, err)
	}
	out, err := exec.Command(path).CombinedOutput()
	if len(out) != 0 {
		t.Fatalf("%s stdout/stderr = %q, want empty", name, out)
	}
	if exit, ok := err.(*exec.ExitError); ok {
		return exit.ExitCode()
	}
	if err != nil {
		t.Fatalf("%s run: %v", name, err)
	}
	return 0
}

func incIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "inc",
		ParamSlots:  1,
		LocalSlots:  1,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRReturn},
		},
	}
}

func nestedCallMainIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "main",
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 40},
			{Kind: ir.IRCall, Name: "inc", ArgSlots: 1, RetSlots: 1},
			{Kind: ir.IRCall, Name: "inc", ArgSlots: 1, RetSlots: 1},
			{Kind: ir.IRReturn},
		},
	}
}

func callWithLiveScratchMainIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "main",
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 7},
			{Kind: ir.IRConstI32, Imm: 35},
			{Kind: ir.IRCall, Name: "inc", ArgSlots: 1, RetSlots: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRReturn},
		},
	}
}

func sumNIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "sum_n",
		ParamSlots:  1,
		LocalSlots:  3,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRReturn},
		},
	}
}

func sumStrideIRFunc() ir.IRFunc {
	fn := sumNIRFunc()
	fn.Name = "sum_stride"
	fn.Instrs[14].Imm = 2
	return fn
}

func sumCallLoopIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "sum_call",
		ParamSlots:  1,
		LocalSlots:  3,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRCall, Name: "inc", ArgSlots: 1, RetSlots: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRReturn},
		},
	}
}

func mainCallsSumCallIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "main",
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 10},
			{Kind: ir.IRCall, Name: "sum_call", ArgSlots: 1, RetSlots: 1},
			{Kind: ir.IRReturn},
		},
	}
}

func mixTwoArgIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "mix",
		ParamSlots:  2,
		LocalSlots:  2,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 3},
			{Kind: ir.IRMulI32},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRConstI32, Imm: 97},
			{Kind: ir.IRModI32},
			{Kind: ir.IRReturn},
		},
	}
}

func functionCallsBenchmarkMainIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "main",
		LocalSlots:  2,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRLabel, Label: 0},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 200000},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRCall, Name: "mix", ArgSlots: 2, RetSlots: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRJmp, Label: 0},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRCmpGeI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRReturn},
		},
	}
}

func compileTimeF0IRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "p25.compile_time.f0",
		ParamSlots:  1,
		LocalSlots:  1,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRReturn},
		},
	}
}

func compileTimeF1IRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "p25.compile_time.f1",
		ParamSlots:  1,
		LocalSlots:  1,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRCall, Name: "p25.compile_time.f0", ArgSlots: 1, RetSlots: 1},
			{Kind: ir.IRConstI32, Imm: 3},
			{Kind: ir.IRMulI32},
			{Kind: ir.IRReturn},
		},
	}
}

func compileTimeF2IRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "p25.compile_time.f2",
		ParamSlots:  1,
		LocalSlots:  1,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRCall, Name: "p25.compile_time.f1", ArgSlots: 1, RetSlots: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRCall, Name: "p25.compile_time.f0", ArgSlots: 1, RetSlots: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRReturn},
		},
	}
}

func compileTimeBenchmarkMainIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "p25.compile_time.main",
		ExportName:  "main",
		LocalSlots:  2,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRLabel, Label: 0},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 200000},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRCall, Name: "p25.compile_time.f2", ArgSlots: 1, RetSlots: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRJmp, Label: 0},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRCmpEqI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRReturn},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
		},
	}
}

func recursionFibIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "p25.recursion.fib",
		ParamSlots:  1,
		LocalSlots:  1,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 2},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 0},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRReturn},
			{Kind: ir.IRLabel, Label: 0},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRSubI32},
			{Kind: ir.IRCall, Name: "p25.recursion.fib", ArgSlots: 1, RetSlots: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 2},
			{Kind: ir.IRSubI32},
			{Kind: ir.IRCall, Name: "p25.recursion.fib", ArgSlots: 1, RetSlots: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRReturn},
		},
	}
}

func recursionBenchmarkMainIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "p25.recursion.main",
		ExportName:  "main",
		LocalSlots:  2,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRLabel, Label: 0},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 40},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 10},
			{Kind: ir.IRCall, Name: "p25.recursion.fib", ArgSlots: 1, RetSlots: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRJmp, Label: 0},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 2200},
			{Kind: ir.IRCmpEqI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRReturn},
		},
	}
}

func mainCallsRecursionFibIRFunc(n int32) ir.IRFunc {
	return ir.IRFunc{
		Name:        "main",
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: n},
			{Kind: ir.IRCall, Name: "p25.recursion.fib", ArgSlots: 1, RetSlots: 1},
			{Kind: ir.IRReturn},
		},
	}
}

func integerLoopsBenchmarkMainIRFunc() ir.IRFunc {
	fn := functionCallsBenchmarkMainIRFunc()
	fn.Name = "main"
	fn.Instrs = append([]ir.IRInstr(nil), fn.Instrs...)
	fn.Instrs[10] = ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0}
	fn.Instrs[11] = ir.IRInstr{Kind: ir.IRConstI32, Imm: 7}
	fn.Instrs[12] = ir.IRInstr{Kind: ir.IRModI32}
	return fn
}

func allocationLoopIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "p25.allocation.main",
		ExportName:  "main",
		LocalSlots:  20,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRLabel, Label: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 1024},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 1},
			{Kind: ir.IRConstI32, Imm: 32},
			{Kind: ir.IRStackSliceI32, Local: 4, ArgSlots: 16, Imm: 32, Name: "xs"},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRIndexStoreI32},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRIndexLoadI32},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRJmp, Label: 0},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRCmpGtI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRReturn},
		},
	}
}

func regionIslandAllocationMainIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "p25.region_island_allocation.main",
		ExportName:  "main",
		LocalSlots:  5,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRLabel, Label: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 256},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 1},
			{Kind: ir.IRConstI32, Imm: 256},
			{Kind: ir.IRIslandNew},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRConstI32, Imm: 16},
			{Kind: ir.IRIslandMakeSliceI32, Name: "xs"},
			{Kind: ir.IRStoreLocal, Local: 4},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 4},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRIndexStoreI32, ProofID: "proof:allocation-zero:literal0:xs:9:13"},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 4},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRIndexLoadI32Unchecked, ProofID: "proof:allocation-zero:literal0:xs:10:35"},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRIslandFree},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRJmp, Label: 0},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRCmpGtI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRReturn},
		},
	}
}

func boundsCheckLoopsIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "p25.bounds_check_loops.main",
		ExportName:  "main",
		LocalSlots:  2054,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 4096},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRStackSliceI32, Local: 6, ArgSlots: 2048, Imm: 4096, Name: "xs"},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRLabel, Label: 0},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRConstI32, Imm: 97},
			{Kind: ir.IRModI32},
			{Kind: ir.IRIndexStoreI32, ProofID: "proof:while:i:xs:8:5"},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRJmp, Label: 0},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 4},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRConstI32, Imm: 200000},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 3},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRConstI32, Imm: 17},
			{Kind: ir.IRMulI32},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRModI32},
			{Kind: ir.IRStoreLocal, Local: 5},
			{Kind: ir.IRLoadLocal, Local: 4},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 5},
			{Kind: ir.IRIndexLoadI32Unchecked, ProofID: "proof:modulo:idx:xs:14:33"},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 4},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRJmp, Label: 2},
			{Kind: ir.IRLabel, Label: 3},
			{Kind: ir.IRLoadLocal, Local: 4},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRCmpGeI32},
			{Kind: ir.IRJmpIfZero, Label: 4},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
			{Kind: ir.IRLabel, Label: 4},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRReturn},
		},
	}
}

func matrixMultiplyMainIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "p25.matrix_multiply.main",
		ExportName:  "main",
		LocalSlots:  28,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 9},
			{Kind: ir.IRStackSliceI32, Local: 13, ArgSlots: 5, Imm: 9, Name: "a"},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 9},
			{Kind: ir.IRStackSliceI32, Local: 18, ArgSlots: 5, Imm: 9, Name: "b"},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRConstI32, Imm: 9},
			{Kind: ir.IRStackSliceI32, Local: 23, ArgSlots: 5, Imm: 9, Name: "c"},
			{Kind: ir.IRStoreLocal, Local: 5},
			{Kind: ir.IRStoreLocal, Local: 4},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 6},
			{Kind: ir.IRLabel, Label: 0},
			{Kind: ir.IRLoadLocal, Local: 6},
			{Kind: ir.IRConstI32, Imm: 9},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 6},
			{Kind: ir.IRLoadLocal, Local: 6},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRIndexStoreI32, ProofID: "proof:while-const:i:a:10:9"},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 6},
			{Kind: ir.IRConstI32, Imm: 9},
			{Kind: ir.IRLoadLocal, Local: 6},
			{Kind: ir.IRSubI32},
			{Kind: ir.IRIndexStoreI32, ProofID: "proof:while-const:i:b:11:9"},
			{Kind: ir.IRLoadLocal, Local: 4},
			{Kind: ir.IRLoadLocal, Local: 5},
			{Kind: ir.IRLoadLocal, Local: 6},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRIndexStoreI32, ProofID: "proof:while-const:i:c:12:9"},
			{Kind: ir.IRLoadLocal, Local: 6},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 6},
			{Kind: ir.IRJmp, Label: 0},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 7},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 8},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 8},
			{Kind: ir.IRConstI32, Imm: 2000},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 3},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 9},
			{Kind: ir.IRLabel, Label: 4},
			{Kind: ir.IRLoadLocal, Local: 9},
			{Kind: ir.IRConstI32, Imm: 3},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 5},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 10},
			{Kind: ir.IRLabel, Label: 6},
			{Kind: ir.IRLoadLocal, Local: 10},
			{Kind: ir.IRConstI32, Imm: 3},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 7},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 11},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 12},
			{Kind: ir.IRLabel, Label: 8},
			{Kind: ir.IRLoadLocal, Local: 11},
			{Kind: ir.IRConstI32, Imm: 3},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 9},
			{Kind: ir.IRLoadLocal, Local: 12},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 9},
			{Kind: ir.IRConstI32, Imm: 3},
			{Kind: ir.IRMulI32},
			{Kind: ir.IRLoadLocal, Local: 11},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRIndexLoadI32Unchecked, ProofID: "proof:affine-const:row_k:a:24:38"},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 11},
			{Kind: ir.IRConstI32, Imm: 3},
			{Kind: ir.IRMulI32},
			{Kind: ir.IRLoadLocal, Local: 10},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRIndexLoadI32Unchecked, ProofID: "proof:affine-const:k_col:b:24:55"},
			{Kind: ir.IRMulI32},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 12},
			{Kind: ir.IRLoadLocal, Local: 11},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 11},
			{Kind: ir.IRJmp, Label: 8},
			{Kind: ir.IRLabel, Label: 9},
			{Kind: ir.IRLoadLocal, Local: 4},
			{Kind: ir.IRLoadLocal, Local: 5},
			{Kind: ir.IRLoadLocal, Local: 9},
			{Kind: ir.IRConstI32, Imm: 3},
			{Kind: ir.IRMulI32},
			{Kind: ir.IRLoadLocal, Local: 10},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRLoadLocal, Local: 12},
			{Kind: ir.IRIndexStoreI32, ProofID: "proof:affine-const:row_col:c:26:19"},
			{Kind: ir.IRLoadLocal, Local: 10},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 10},
			{Kind: ir.IRJmp, Label: 6},
			{Kind: ir.IRLabel, Label: 7},
			{Kind: ir.IRLoadLocal, Local: 9},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 9},
			{Kind: ir.IRJmp, Label: 4},
			{Kind: ir.IRLabel, Label: 5},
			{Kind: ir.IRLoadLocal, Local: 7},
			{Kind: ir.IRLoadLocal, Local: 4},
			{Kind: ir.IRLoadLocal, Local: 5},
			{Kind: ir.IRLoadLocal, Local: 8},
			{Kind: ir.IRConstI32, Imm: 9},
			{Kind: ir.IRModI32},
			{Kind: ir.IRIndexLoadI32Unchecked, ProofID: "proof:modulo:modulo_const:c:29:37"},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 7},
			{Kind: ir.IRLoadLocal, Local: 8},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 8},
			{Kind: ir.IRJmp, Label: 2},
			{Kind: ir.IRLabel, Label: 3},
			{Kind: ir.IRLoadLocal, Local: 7},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRCmpGtI32},
			{Kind: ir.IRJmpIfZero, Label: 10},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
			{Kind: ir.IRLabel, Label: 10},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRReturn},
		},
	}
}

func mainCallsSumNIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "main",
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 10},
			{Kind: ir.IRCall, Name: "sum_n", ArgSlots: 1, RetSlots: 1},
			{Kind: ir.IRReturn},
		},
	}
}

func mainCallsSumStrideIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "main",
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 10},
			{Kind: ir.IRCall, Name: "sum_stride", ArgSlots: 1, RetSlots: 1},
			{Kind: ir.IRReturn},
		},
	}
}

func divModIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "div_mod",
		ParamSlots:  2,
		LocalSlots:  2,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRDivI32},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRModI32},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRReturn},
		},
	}
}

func mainCallsDivModIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "main",
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 85},
			{Kind: ir.IRConstI32, Imm: 7},
			{Kind: ir.IRCall, Name: "div_mod", ArgSlots: 2, RetSlots: 1},
			{Kind: ir.IRReturn},
		},
	}
}

func postgresqlFrameTypeAtIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "p25.postgresql_single_multiple_update.frame_type_at",
		ParamSlots:  3,
		LocalSlots:  3,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRIndexLoadU8Unchecked, ProofID: "proof:helper-offset:offset:src:4:16"},
			{Kind: ir.IRReturn},
		},
	}
}

func mainCallsPostgreSQLFrameTypeAtIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "main",
		LocalSlots:  2,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 8},
			{Kind: ir.IRMakeSliceU8},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 3},
			{Kind: ir.IRConstI32, Imm: 42},
			{Kind: ir.IRIndexStoreU8},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 3},
			{
				Kind:     ir.IRCall,
				Name:     "p25.postgresql_single_multiple_update.frame_type_at",
				ArgSlots: 3,
				RetSlots: 1,
			},
			{Kind: ir.IRReturn},
		},
	}
}

func sumSliceIRFunc(proof bool) ir.IRFunc {
	loadKind := ir.IRIndexLoadI32
	proofID := ""
	if proof {
		loadKind = ir.IRIndexLoadI32Unchecked
		proofID = "proof:while:i:xs:1:1"
	}
	return ir.IRFunc{
		Name:        "sum",
		ParamSlots:  2,
		LocalSlots:  4,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: loadKind, ProofID: proofID},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRReturn},
		},
	}
}

func sumSliceStrideIRFunc(proof bool, step int32) ir.IRFunc {
	fn := sumSliceIRFunc(proof)
	fn.Name = "sum_stride"
	fn.Instrs[17].Imm = step
	return fn
}

func copyU8IRFunc(proof bool) ir.IRFunc {
	loadKind := ir.IRIndexLoadU8
	proofID := ""
	if proof {
		loadKind = ir.IRIndexLoadU8Unchecked
		proofID = "proof:copy-loop:u8:1:1"
	}
	return ir.IRFunc{
		Name:        "copy_u8",
		ParamSlots:  3,
		LocalSlots:  4,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: loadKind, ProofID: proofID},
			{Kind: ir.IRIndexStoreU8},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
		},
	}
}

func mainCallsSumSliceTailIRFunc() ir.IRFunc {
	instrs := []ir.IRInstr{
		{Kind: ir.IRConstI32, Imm: 7},
		{Kind: ir.IRMakeSliceI32},
		{Kind: ir.IRStoreLocal, Local: 1},
		{Kind: ir.IRStoreLocal, Local: 0},
	}
	for i := int32(0); i < 7; i++ {
		instrs = append(instrs,
			ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
			ir.IRInstr{Kind: ir.IRLoadLocal, Local: 1},
			ir.IRInstr{Kind: ir.IRConstI32, Imm: i},
			ir.IRInstr{Kind: ir.IRConstI32, Imm: i + 1},
			ir.IRInstr{Kind: ir.IRIndexStoreI32},
		)
	}
	instrs = append(instrs,
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 1},
		ir.IRInstr{Kind: ir.IRCall, Name: "sum", ArgSlots: 2, RetSlots: 1},
		ir.IRInstr{Kind: ir.IRReturn},
	)
	return ir.IRFunc{
		Name:        "main",
		LocalSlots:  2,
		ReturnSlots: 1,
		Instrs:      instrs,
	}
}

func mainCallsCopyU8TailIRFunc() ir.IRFunc {
	instrs := []ir.IRInstr{
		{Kind: ir.IRConstI32, Imm: 19},
		{Kind: ir.IRMakeSliceU8},
		{Kind: ir.IRStoreLocal, Local: 1},
		{Kind: ir.IRStoreLocal, Local: 0},
		{Kind: ir.IRConstI32, Imm: 19},
		{Kind: ir.IRMakeSliceU8},
		{Kind: ir.IRStoreLocal, Local: 3},
		{Kind: ir.IRStoreLocal, Local: 2},
	}
	for i := int32(0); i < 19; i++ {
		instrs = append(instrs,
			ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
			ir.IRInstr{Kind: ir.IRLoadLocal, Local: 1},
			ir.IRInstr{Kind: ir.IRConstI32, Imm: i},
			ir.IRInstr{Kind: ir.IRConstI32, Imm: i + 1},
			ir.IRInstr{Kind: ir.IRIndexStoreU8},
		)
	}
	instrs = append(instrs,
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 2},
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 1},
		ir.IRInstr{Kind: ir.IRCall, Name: "copy_u8", ArgSlots: 3, RetSlots: 1},
		ir.IRInstr{Kind: ir.IRStoreLocal, Local: 4},
	)
	for _, idx := range []int32{0, 15, 16, 18} {
		instrs = append(instrs,
			ir.IRInstr{Kind: ir.IRLoadLocal, Local: 2},
			ir.IRInstr{Kind: ir.IRLoadLocal, Local: 3},
			ir.IRInstr{Kind: ir.IRConstI32, Imm: idx},
			ir.IRInstr{Kind: ir.IRIndexLoadU8},
		)
		if idx != 0 {
			instrs = append(instrs, ir.IRInstr{Kind: ir.IRAddI32})
		}
	}
	instrs = append(instrs, ir.IRInstr{Kind: ir.IRReturn})
	return ir.IRFunc{
		Name:        "main",
		LocalSlots:  5,
		ReturnSlots: 1,
		Instrs:      instrs,
	}
}

func mapAddI32IRFunc(proof bool) ir.IRFunc {
	loadKind := ir.IRIndexLoadI32
	proofID := ""
	if proof {
		loadKind = ir.IRIndexLoadI32Unchecked
		proofID = "proof:map-loop:i32:1:1"
	}
	return ir.IRFunc{
		Name:        "map_i32_add1",
		ParamSlots:  2,
		LocalSlots:  4,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: loadKind, ProofID: proofID},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRIndexStoreI32},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
		},
	}
}

func memsetZeroU8IRFunc(proof bool) ir.IRFunc {
	proofID := ""
	if proof {
		proofID = "proof:memset-loop:u8:zero:1:1"
	}
	return ir.IRFunc{
		Name:        "memset_zero_u8",
		ParamSlots:  2,
		LocalSlots:  3,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRIndexStoreU8, ProofID: proofID},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
		},
	}
}

func mainCallsMapAddI32TailIRFunc() ir.IRFunc {
	instrs := []ir.IRInstr{
		{Kind: ir.IRConstI32, Imm: 7},
		{Kind: ir.IRMakeSliceI32},
		{Kind: ir.IRStoreLocal, Local: 1},
		{Kind: ir.IRStoreLocal, Local: 0},
	}
	for i := int32(0); i < 7; i++ {
		instrs = append(instrs,
			ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
			ir.IRInstr{Kind: ir.IRLoadLocal, Local: 1},
			ir.IRInstr{Kind: ir.IRConstI32, Imm: i},
			ir.IRInstr{Kind: ir.IRConstI32, Imm: i + 1},
			ir.IRInstr{Kind: ir.IRIndexStoreI32},
		)
	}
	instrs = append(instrs,
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 1},
		ir.IRInstr{Kind: ir.IRCall, Name: "map_i32_add1", ArgSlots: 2, RetSlots: 1},
		ir.IRInstr{Kind: ir.IRStoreLocal, Local: 2},
	)
	for _, idx := range []int32{0, 3, 4, 6} {
		instrs = append(instrs,
			ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
			ir.IRInstr{Kind: ir.IRLoadLocal, Local: 1},
			ir.IRInstr{Kind: ir.IRConstI32, Imm: idx},
			ir.IRInstr{Kind: ir.IRIndexLoadI32},
		)
		if idx != 0 {
			instrs = append(instrs, ir.IRInstr{Kind: ir.IRAddI32})
		}
	}
	instrs = append(instrs, ir.IRInstr{Kind: ir.IRReturn})
	return ir.IRFunc{
		Name:        "main",
		LocalSlots:  3,
		ReturnSlots: 1,
		Instrs:      instrs,
	}
}

func mainCallsMemsetZeroU8TailIRFunc() ir.IRFunc {
	instrs := []ir.IRInstr{
		{Kind: ir.IRConstI32, Imm: 19},
		{Kind: ir.IRMakeSliceU8},
		{Kind: ir.IRStoreLocal, Local: 1},
		{Kind: ir.IRStoreLocal, Local: 0},
	}
	for i := int32(0); i < 19; i++ {
		instrs = append(instrs,
			ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
			ir.IRInstr{Kind: ir.IRLoadLocal, Local: 1},
			ir.IRInstr{Kind: ir.IRConstI32, Imm: i},
			ir.IRInstr{Kind: ir.IRConstI32, Imm: i + 1},
			ir.IRInstr{Kind: ir.IRIndexStoreU8},
		)
	}
	instrs = append(instrs,
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 1},
		ir.IRInstr{Kind: ir.IRCall, Name: "memset_zero_u8", ArgSlots: 2, RetSlots: 1},
		ir.IRInstr{Kind: ir.IRStoreLocal, Local: 2},
	)
	for _, idx := range []int32{0, 15, 16, 18} {
		instrs = append(instrs,
			ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
			ir.IRInstr{Kind: ir.IRLoadLocal, Local: 1},
			ir.IRInstr{Kind: ir.IRConstI32, Imm: idx},
			ir.IRInstr{Kind: ir.IRIndexLoadU8},
		)
		if idx != 0 {
			instrs = append(instrs, ir.IRInstr{Kind: ir.IRAddI32})
		}
	}
	instrs = append(instrs, ir.IRInstr{Kind: ir.IRReturn})
	return ir.IRFunc{
		Name:        "main",
		LocalSlots:  3,
		ReturnSlots: 1,
		Instrs:      instrs,
	}
}

func writeHelloMainFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "main",
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRStrLit, Str: []byte("hello")},
			{Kind: ir.IRWrite},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
		},
	}
}

func twoSmallMakeSlicesMainIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "main",
		LocalSlots:  4,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 4},
			{Kind: ir.IRMakeSliceU8},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 4},
			{Kind: ir.IRMakeSliceU8},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRConstI32, Imm: 10},
			{Kind: ir.IRIndexStoreU8},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRConstI32, Imm: 20},
			{Kind: ir.IRIndexStoreU8},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRIndexLoadU8},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRIndexLoadU8},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRReturn},
		},
	}
}

func manySmallMakeSlicesMainIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "main",
		LocalSlots:  4,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 200},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRConstI32, Imm: 4},
			{Kind: ir.IRMakeSliceU8},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRIndexStoreU8},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRIndexLoadU8},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRReturn},
		},
	}
}

func refillSmallHeapMakeSlicesMainIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "main",
		LocalSlots:  3,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 5000},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRConstI32, Imm: 4},
			{Kind: ir.IRMakeSliceU8},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRIndexStoreU8},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRIndexLoadU8},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRCmpNeI32},
			{Kind: ir.IRJmpIfZero, Label: 3},
			{Kind: ir.IRConstI32, Imm: 7},
			{Kind: ir.IRReturn},
			{Kind: ir.IRLabel, Label: 3},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRConstI32, Imm: 42},
			{Kind: ir.IRReturn},
		},
	}
}

func largeMakeSliceMainIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "main",
		LocalSlots:  2,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 5000},
			{Kind: ir.IRMakeSliceU8},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 4999},
			{Kind: ir.IRConstI32, Imm: 42},
			{Kind: ir.IRIndexStoreU8},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 4999},
			{Kind: ir.IRIndexLoadU8},
			{Kind: ir.IRReturn},
		},
	}
}

func memoryBackendSmokeMainIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "main",
		LocalSlots:  5,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 32},
			{Kind: ir.IRAllocBytes, Name: "owned_heap"},
			{
				Kind:            ir.IRDropOwned,
				LayoutID:        "layout:owned_heap",
				OwnershipDomain: ir.IROwnershipDomainHeap,
				ReleaseKind:     ir.IRReleaseKindLinuxMmap,
			},
			{
				Kind:            ir.IRReleaseAllocation,
				LayoutID:        "layout:owned_heap",
				OwnershipDomain: ir.IROwnershipDomainHeap,
				ReleaseKind:     ir.IRReleaseKindLinuxMmap,
			},
			{Kind: ir.IRRegionEnter},
			{Kind: ir.IRConstI32, Imm: 8},
			{Kind: ir.IRRegionMakeSliceU8, Name: "region_bytes"},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRConstI32, Imm: 7},
			{Kind: ir.IRIndexStoreU8},
			{Kind: ir.IRRegionReset},
			{Kind: ir.IRConstI32, Imm: 64},
			{Kind: ir.IRIslandNew},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 4},
			{Kind: ir.IRIslandMakeSliceU8, Name: "island_bytes"},
			{Kind: ir.IRStoreLocal, Local: 4},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 4},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRConstI32, Imm: 35},
			{Kind: ir.IRIndexStoreU8},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRIslandReset},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRIslandFree},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
		},
	}
}

func countBytes(buf []byte, needle []byte) int {
	count := 0
	for {
		idx := bytes.Index(buf, needle)
		if idx < 0 {
			return count
		}
		count++
		buf = buf[idx+len(needle):]
	}
}

func hasRelocKind(relocs []tobj.Reloc, kind tobj.RelocKind) bool {
	for _, reloc := range relocs {
		if reloc.Kind == kind {
			return true
		}
	}
	return false
}

func hasSymbol(symbols []tobj.Symbol, name string, params, returns int) bool {
	for _, sym := range symbols {
		if sym.Name == name && sym.HasSignature && sym.ParamSlots == params &&
			sym.ReturnSlots == returns {
			return true
		}
	}
	return false
}
