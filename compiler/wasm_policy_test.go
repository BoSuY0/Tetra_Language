package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/compiler/internal/ir"
)

func TestWASMPolicyRejectsUnsafeCapabilityMemoryMMIOAndCtxSwitchIR(t *testing.T) {
	cases := []struct {
		name    string
		instr   ir.IRInstr
		builtin string
	}{
		{name: "alloc_bytes", instr: ir.IRInstr{Kind: ir.IRAllocBytes}, builtin: "core.alloc_bytes"},
		{name: "cap_io", instr: ir.IRInstr{Kind: ir.IRCapIO}, builtin: "core.cap_io"},
		{name: "cap_mem", instr: ir.IRInstr{Kind: ir.IRCapMem}, builtin: "core.cap_mem"},
		{name: "load_i32", instr: ir.IRInstr{Kind: ir.IRMemReadI32}, builtin: "core.load_i32"},
		{name: "store_i32", instr: ir.IRInstr{Kind: ir.IRMemWriteI32}, builtin: "core.store_i32"},
		{name: "load_u8", instr: ir.IRInstr{Kind: ir.IRMemReadU8}, builtin: "core.load_u8"},
		{name: "store_u8", instr: ir.IRInstr{Kind: ir.IRMemWriteU8}, builtin: "core.store_u8"},
		{name: "load_ptr", instr: ir.IRInstr{Kind: ir.IRMemReadPtr}, builtin: "core.load_ptr"},
		{name: "store_ptr", instr: ir.IRInstr{Kind: ir.IRMemWritePtr}, builtin: "core.store_ptr"},
		{name: "load_i32_offset", instr: ir.IRInstr{Kind: ir.IRMemReadI32Offset}, builtin: "core.load_i32"},
		{name: "store_i32_offset", instr: ir.IRInstr{Kind: ir.IRMemWriteI32Offset}, builtin: "core.store_i32"},
		{name: "load_u8_offset", instr: ir.IRInstr{Kind: ir.IRMemReadU8Offset}, builtin: "core.load_u8"},
		{name: "store_u8_offset", instr: ir.IRInstr{Kind: ir.IRMemWriteU8Offset}, builtin: "core.store_u8"},
		{name: "load_ptr_offset", instr: ir.IRInstr{Kind: ir.IRMemReadPtrOffset}, builtin: "core.load_ptr"},
		{name: "store_ptr_offset", instr: ir.IRInstr{Kind: ir.IRMemWritePtrOffset}, builtin: "core.store_ptr"},
		{name: "ptr_add", instr: ir.IRInstr{Kind: ir.IRPtrAdd}, builtin: "core.ptr_add"},
		{name: "mmio_read_i32", instr: ir.IRInstr{Kind: ir.IRMmioReadI32}, builtin: "core.mmio_read_i32"},
		{name: "mmio_write_i32", instr: ir.IRInstr{Kind: ir.IRMmioWriteI32}, builtin: "core.mmio_write_i32"},
		{name: "ctx_switch", instr: ir.IRInstr{Kind: ir.IRCtxSwitch}, builtin: "core.ctx_switch"},
	}

	for _, target := range []string{"wasm32-wasi", "wasm32-web"} {
		for _, tc := range cases {
			t.Run(target+"/"+tc.name, func(t *testing.T) {
				err := validateWASMIRPolicy(target, []IRFunc{{
					Name:        "main",
					ReturnSlots: 1,
					Instrs: []ir.IRInstr{
						tc.instr,
						{Kind: ir.IRConstI32, Imm: 0},
						{Kind: ir.IRReturn},
					},
				}})
				if err == nil {
					t.Fatalf("expected WASM policy rejection for %s", tc.builtin)
				}
				for _, want := range []string{target, tc.builtin, "unsupported on WASM targets by policy"} {
					if !strings.Contains(err.Error(), want) {
						t.Fatalf("error = %v, want substring %q", err, want)
					}
				}
			})
		}
	}
}

func TestWASMBuildRejectsCapabilityBuiltinBeforeBackendEmission(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, filepath.FromSlash("app/main.tetra"))
	src := `module app.main
func main() -> Int
uses capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        return 0
`
	if err := os.MkdirAll(filepath.Dir(srcPath), 0o755); err != nil {
		t.Fatalf("mkdir source dir: %v", err)
	}
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	for _, target := range []string{"wasm32-wasi", "wasm32-web"} {
		t.Run(target, func(t *testing.T) {
			outPath := filepath.Join(tmp, target+".wasm")
			_, err := BuildFileWithStatsOpt(srcPath, outPath, target, BuildOptions{Jobs: 1})
			if err == nil {
				t.Fatalf("expected %s capability policy rejection", target)
			}
			for _, want := range []string{target, "core.cap_mem", "unsupported on WASM targets by policy"} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("error = %v, want substring %q", err, want)
				}
			}
		})
	}
}
