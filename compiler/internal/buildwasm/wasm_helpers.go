package buildwasm

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"tetra_language/compiler/internal/backend/native_shell"
	"tetra_language/compiler/internal/backend/wasm32_web"
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/lower"
	"tetra_language/compiler/internal/semantics"
)

type IRPolicy struct {
	Builtin  string
	Category string
}

func WebLoaderPath(outputPath string) string {
	ext := filepath.Ext(outputPath)
	if strings.EqualFold(ext, ".wasm") {
		return strings.TrimSuffix(outputPath, ext) + ".mjs"
	}
	return outputPath + ".mjs"
}

func RelocateGlobalSlots(funcs []ir.IRFunc, offset int) []ir.IRFunc {
	if offset == 0 {
		return funcs
	}
	out := make([]ir.IRFunc, len(funcs))
	for i, fn := range funcs {
		out[i] = fn
		if len(fn.Instrs) == 0 {
			continue
		}
		out[i].Instrs = append([]ir.IRInstr(nil), fn.Instrs...)
		for j := range out[i].Instrs {
			switch out[i].Instrs[j].Kind {
			case ir.IRLoadGlobal, ir.IRStoreGlobal:
				out[i].Instrs[j].Local += offset
			}
		}
	}
	return out
}

func FirstUnsupportedRuntimeBuiltin(funcs []ir.IRFunc, target string) (frontend.Position, string, bool) {
	for _, fn := range funcs {
		for _, instr := range fn.Instrs {
			if instr.Kind != ir.IRCall {
				continue
			}
			runtimeName, ok := RuntimeNameForBuiltin(instr.Name, target)
			if !ok {
				continue
			}
			return instr.Pos, runtimeName, true
		}
	}
	return frontend.Position{}, "", false
}

func RuntimeNameForBuiltin(name string, target string) (string, bool) {
	switch {
	case name == "__tetra_actor_node_connect", name == "__tetra_actor_spawn_remote", name == "__tetra_actor_node_status":
		return "distributed actors", true
	case strings.HasPrefix(name, "__tetra_actor_"):
		return "actors", true
	case strings.HasPrefix(name, "__tetra_task_"):
		return "task", true
	case strings.HasPrefix(name, "__tetra_fs_"):
		return "filesystem", true
	case strings.HasPrefix(name, "__tetra_net_"):
		return "networking", true
	case strings.HasPrefix(name, "__tetra_surface_"):
		if target == "wasm32-web" {
			return "", false
		}
		return "surface", true
	case strings.HasPrefix(name, "__tetra_time_"), name == "__tetra_sleep_ms", name == "__tetra_sleep_until_ms", name == "__tetra_deadline_ms", name == "__tetra_timer_ready_ms":
		return "time", true
	default:
		return "", false
	}
}

func FirstBlockedIRPolicy(target string, funcs []ir.IRFunc) (frontend.Position, IRPolicy, bool) {
	if !strings.HasPrefix(target, "wasm32-") {
		return frontend.Position{}, IRPolicy{}, false
	}
	for _, fn := range funcs {
		for _, instr := range fn.Instrs {
			policy, blocked := BlockedIRPolicy(instr.Kind)
			if !blocked {
				continue
			}
			return instr.Pos, policy, true
		}
	}
	return frontend.Position{}, IRPolicy{}, false
}

func BlockedIRPolicy(kind ir.IRInstrKind) (IRPolicy, bool) {
	switch kind {
	case ir.IRAllocBytes:
		return IRPolicy{Builtin: "core.alloc_bytes", Category: "raw memory allocation"}, true
	case ir.IRCapIO:
		return IRPolicy{Builtin: "core.cap_io", Category: "capability token construction"}, true
	case ir.IRCapMem:
		return IRPolicy{Builtin: "core.cap_mem", Category: "capability token construction"}, true
	case ir.IRMemReadI32:
		return IRPolicy{Builtin: "core.load_i32", Category: "raw memory access"}, true
	case ir.IRMemWriteI32:
		return IRPolicy{Builtin: "core.store_i32", Category: "raw memory access"}, true
	case ir.IRMemReadU8:
		return IRPolicy{Builtin: "core.load_u8", Category: "raw memory access"}, true
	case ir.IRMemWriteU8:
		return IRPolicy{Builtin: "core.store_u8", Category: "raw memory access"}, true
	case ir.IRMemReadPtr:
		return IRPolicy{Builtin: "core.load_ptr", Category: "raw pointer memory access"}, true
	case ir.IRMemWritePtr:
		return IRPolicy{Builtin: "core.store_ptr", Category: "raw pointer memory access"}, true
	case ir.IRMemWriteArchPtr:
		return IRPolicy{Builtin: "core.store_arch_ptr", Category: "raw architectural pointer memory access"}, true
	case ir.IRMemReadI32Offset:
		return IRPolicy{Builtin: "core.load_i32", Category: "raw memory access"}, true
	case ir.IRMemWriteI32Offset:
		return IRPolicy{Builtin: "core.store_i32", Category: "raw memory access"}, true
	case ir.IRMemReadU8Offset:
		return IRPolicy{Builtin: "core.load_u8", Category: "raw memory access"}, true
	case ir.IRMemWriteU8Offset:
		return IRPolicy{Builtin: "core.store_u8", Category: "raw memory access"}, true
	case ir.IRMemReadPtrOffset:
		return IRPolicy{Builtin: "core.load_ptr", Category: "raw pointer memory access"}, true
	case ir.IRMemWritePtrOffset:
		return IRPolicy{Builtin: "core.store_ptr", Category: "raw pointer memory access"}, true
	case ir.IRMemWriteArchPtrOffset:
		return IRPolicy{Builtin: "core.store_arch_ptr", Category: "raw architectural pointer memory access"}, true
	case ir.IRPtrAdd:
		return IRPolicy{Builtin: "core.ptr_add", Category: "raw pointer arithmetic"}, true
	case ir.IRMmioReadI32:
		return IRPolicy{Builtin: "core.mmio_read_i32", Category: "MMIO"}, true
	case ir.IRMmioWriteI32:
		return IRPolicy{Builtin: "core.mmio_write_i32", Category: "MMIO"}, true
	case ir.IRCtxSwitch:
		return IRPolicy{Builtin: "core.ctx_switch", Category: "context switching"}, true
	default:
		return IRPolicy{}, false
	}
}

func EmitUIArtifacts(outputPath string, target string, checked *semantics.CheckedProgram) error {
	bundle, err := lower.LowerUI(checked)
	if err != nil {
		return err
	}
	if bundle == nil || len(bundle.Views) == 0 {
		return nil
	}
	base := UIArtifactBasePath(outputPath)
	uiJSONPath := base + ".ui.json"
	raw, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(uiJSONPath, raw, 0o644); err != nil {
		return err
	}
	toolkitBundle, err := lower.LowerUIToolkit(bundle)
	if err != nil {
		return err
	}
	if toolkitBundle != nil {
		toolkitPath := base + ".ui.toolkit.json"
		toolkitRaw, err := json.MarshalIndent(toolkitBundle, "", "  ")
		if err != nil {
			return err
		}
		toolkitRaw = append(toolkitRaw, '\n')
		if err := os.WriteFile(toolkitPath, toolkitRaw, 0o644); err != nil {
			return err
		}
	}
	if target == "wasm32-web" {
		uiModulePath := base + ".ui.web.mjs"
		uiModule := wasm32_web.UIModule(filepath.Base(uiJSONPath))
		if err := os.WriteFile(uiModulePath, uiModule, 0o644); err != nil {
			return err
		}
		htmlPath := base + ".ui.html"
		html := wasm32_web.UIHTMLPage(filepath.Base(outputPath), filepath.Base(WebLoaderPath(outputPath)), filepath.Base(uiModulePath))
		if err := os.WriteFile(htmlPath, html, 0o644); err != nil {
			return err
		}
		return nil
	}
	if strings.HasPrefix(target, "wasm32-") {
		return nil
	}
	shellPath := base + ".ui.shell.txt"
	if err := os.WriteFile(shellPath, native_shell.Render(bundle), 0o644); err != nil {
		return err
	}
	shellJSONPath := base + ".ui.shell.json"
	if err := os.WriteFile(shellJSONPath, native_shell.RenderJSON(bundle), 0o644); err != nil {
		return err
	}
	return nil
}

func UIArtifactBasePath(outputPath string) string {
	ext := filepath.Ext(outputPath)
	switch {
	case strings.EqualFold(ext, ".wasm"):
		return strings.TrimSuffix(outputPath, ext)
	case strings.EqualFold(ext, ".exe"):
		return strings.TrimSuffix(outputPath, ext)
	default:
		return outputPath
	}
}
