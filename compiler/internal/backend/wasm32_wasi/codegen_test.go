package wasm32_wasi

import (
	"bytes"
	"testing"

	"tetra_language/compiler/internal/ir"
)

func TestLinkObjectWritesWASMHeaderAndStartExport(t *testing.T) {
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
	if !bytes.Contains(mod, []byte("_start")) {
		t.Fatalf("missing _start export")
	}
}

func TestLinkObjectWASIImportExportShape(t *testing.T) {
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

	imports := wasmImports(t, mod)
	if got := imports["wasi_snapshot_preview1"]; !stringSetHas(got, "fd_write") || !stringSetHas(got, "proc_exit") {
		t.Fatalf("WASI imports = %#v", imports)
	}
	if _, ok := imports["tetra_web_v1"]; ok {
		t.Fatalf("WASI module imported web host namespace: %#v", imports)
	}

	exports := wasmExports(t, mod)
	if exports["memory"] != 0x02 {
		t.Fatalf("memory export kind = 0x%x, want memory", exports["memory"])
	}
	if exports["_start"] != 0x00 {
		t.Fatalf("_start export kind = 0x%x, want func", exports["_start"])
	}
	if _, ok := exports["tetra_main"]; ok {
		t.Fatalf("WASI module exported tetra_main: %#v", exports)
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

func wasmImports(t *testing.T, mod []byte) map[string][]string {
	t.Helper()
	payload := wasmSection(t, mod, 2)
	if payload == nil {
		t.Fatalf("missing import section")
	}
	pos := 0
	count := int(readULEBForTest(t, payload, &pos))
	out := map[string][]string{}
	for i := 0; i < count; i++ {
		module := readWASMNameForTest(t, payload, &pos)
		name := readWASMNameForTest(t, payload, &pos)
		if pos >= len(payload) {
			t.Fatalf("truncated import kind")
		}
		kind := payload[pos]
		pos++
		if kind != 0x00 {
			t.Fatalf("import %s.%s kind = 0x%x, want func", module, name, kind)
		}
		_ = readULEBForTest(t, payload, &pos)
		out[module] = append(out[module], name)
	}
	return out
}

func wasmExports(t *testing.T, mod []byte) map[string]byte {
	t.Helper()
	payload := wasmSection(t, mod, 7)
	if payload == nil {
		t.Fatalf("missing export section")
	}
	pos := 0
	count := int(readULEBForTest(t, payload, &pos))
	out := map[string]byte{}
	for i := 0; i < count; i++ {
		name := readWASMNameForTest(t, payload, &pos)
		if pos >= len(payload) {
			t.Fatalf("truncated export kind")
		}
		kind := payload[pos]
		pos++
		_ = readULEBForTest(t, payload, &pos)
		out[name] = kind
	}
	return out
}

func wasmSection(t *testing.T, mod []byte, wantID byte) []byte {
	t.Helper()
	if len(mod) < 8 || !bytes.Equal(mod[:4], []byte{0x00, 0x61, 0x73, 0x6d}) {
		t.Fatalf("invalid wasm module")
	}
	pos := 8
	for pos < len(mod) {
		id := mod[pos]
		pos++
		size := int(readULEBForTest(t, mod, &pos))
		if pos+size > len(mod) {
			t.Fatalf("truncated section %d", id)
		}
		payload := mod[pos : pos+size]
		if id == wantID {
			return payload
		}
		pos += size
	}
	return nil
}

func readWASMNameForTest(t *testing.T, b []byte, pos *int) string {
	t.Helper()
	n := int(readULEBForTest(t, b, pos))
	if *pos+n > len(b) {
		t.Fatalf("truncated wasm name")
	}
	name := string(b[*pos : *pos+n])
	*pos += n
	return name
}

func readULEBForTest(t *testing.T, b []byte, pos *int) uint32 {
	t.Helper()
	var out uint32
	var shift uint
	for {
		if *pos >= len(b) {
			t.Fatalf("truncated uleb")
		}
		ch := b[*pos]
		*pos++
		out |= uint32(ch&0x7f) << shift
		if ch&0x80 == 0 {
			return out
		}
		shift += 7
		if shift > 28 {
			t.Fatalf("uleb too large")
		}
	}
}

func stringSetHas(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
