package wasm32_web

import (
	"bytes"
	"testing"

	"tetra_language/compiler/internal/ir"
)

func TestLinkObjectWritesWASMHeaderAndTetraMainExport(t *testing.T) {
	obj, err := CodegenObject([]ir.IRFunc{
		{
			Name:        "main",
			ParamSlots:  0,
			LocalSlots:  0,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRReturn},
			},
		},
	}, "main")
	if err != nil {
		t.Fatalf("CodegenObject: %v", err)
	}
	mod, err := LinkObject(obj)
	if err != nil {
		t.Fatalf("LinkObject: %v", err)
	}
	if len(mod) < 8 {
		t.Fatalf("module too short: %d", len(mod))
	}
	if !bytes.Equal(mod[:4], []byte{0x00, 0x61, 0x73, 0x6d}) {
		t.Fatalf("missing wasm magic: % x", mod[:4])
	}
	if !bytes.Equal(mod[4:8], []byte{0x01, 0x00, 0x00, 0x00}) {
		t.Fatalf("unexpected version header: % x", mod[4:8])
	}
	if !bytes.Contains(mod, []byte("tetra_main")) {
		t.Fatalf("missing tetra_main export")
	}
}

func TestLinkObjectRejectsUnsupportedInstruction(t *testing.T) {
	obj, err := CodegenObject([]ir.IRFunc{
		{
			Name:        "main",
			ParamSlots:  0,
			LocalSlots:  0,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRAllocBytes},
				{Kind: ir.IRReturn},
			},
		},
	}, "main")
	if err != nil {
		t.Fatalf("CodegenObject: %v", err)
	}
	_, err = LinkObject(obj)
	if err == nil {
		t.Fatalf("expected unsupported instruction error")
	}
	if got := err.Error(); !bytes.Contains([]byte(got), []byte("unsupported IR instruction")) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoaderModuleIncludesRuntimeNamespaceAndEntry(t *testing.T) {
	loader := string(LoaderModule("app.wasm"))
	if !bytes.Contains([]byte(loader), []byte("tetra_web_v1")) {
		t.Fatalf("loader missing runtime namespace:\n%s", loader)
	}
	if !bytes.Contains([]byte(loader), []byte("tetra_main")) {
		t.Fatalf("loader missing tetra_main call:\n%s", loader)
	}
}
