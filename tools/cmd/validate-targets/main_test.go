package main

import (
	"strings"
	"testing"
)

func TestValidateTargetsReportAcceptsExpectedShape(t *testing.T) {
	raw := []byte(`{
  "supported":["linux-x64","windows-x64","macos-x64","wasm32-wasi","wasm32-web"],
  "build_only":[],
  "planned":[],
  "targets":[
    {"triple":"linux-x64","status":"supported","os":"linux","arch":"x64","abi":"sysv","format":"elf","exe_ext":"","build_only":false,"run_mode":"host_native","run_supported":true,"supports_debug_info":true,"supports_release_optimize":true},
    {"triple":"windows-x64","status":"supported","os":"windows","arch":"x64","abi":"win64","format":"pe","exe_ext":".exe","build_only":false,"run_mode":"host_native","run_supported":false,"run_unsupported_reason":"windows-x64 cannot run on host linux/amd64","supports_debug_info":true,"supports_release_optimize":true},
    {"triple":"macos-x64","status":"supported","os":"macos","arch":"x64","abi":"sysv","format":"macho","exe_ext":"","build_only":false,"run_mode":"host_native","run_supported":false,"run_unsupported_reason":"macos-x64 cannot run on host linux/amd64","supports_debug_info":true,"supports_release_optimize":true},
    {"triple":"wasm32-wasi","status":"supported","os":"wasi","arch":"wasm32","abi":"wasi","format":"wasm","exe_ext":".wasm","build_only":false,"run_mode":"wasi_runner","run_runner":"wasmtime","run_supported":true,"supports_debug_info":false,"supports_release_optimize":true},
    {"triple":"wasm32-web","status":"supported","os":"web","arch":"wasm32","abi":"web","format":"wasm","exe_ext":".wasm","build_only":false,"run_mode":"web_runner","run_supported":false,"run_unsupported_reason":"web runner unavailable: chromium-compatible executable not found","supports_debug_info":false,"supports_release_optimize":true}
  ]
}`)
	if err := validateTargetsReport(raw); err != nil {
		t.Fatalf("validate targets: %v", err)
	}
}

func TestValidateTargetsReportAcceptsMissingWASIRunner(t *testing.T) {
	raw := []byte(`{
  "supported":["linux-x64","windows-x64","macos-x64","wasm32-wasi","wasm32-web"],
  "build_only":[],
  "planned":[],
  "targets":[
    {"triple":"linux-x64","status":"supported","os":"linux","arch":"x64","abi":"sysv","format":"elf","exe_ext":"","build_only":false,"run_mode":"host_native","run_supported":true,"supports_debug_info":true,"supports_release_optimize":true},
    {"triple":"windows-x64","status":"supported","os":"windows","arch":"x64","abi":"win64","format":"pe","exe_ext":".exe","build_only":false,"run_mode":"host_native","run_supported":false,"run_unsupported_reason":"windows-x64 cannot run on host linux/amd64","supports_debug_info":true,"supports_release_optimize":true},
    {"triple":"macos-x64","status":"supported","os":"macos","arch":"x64","abi":"sysv","format":"macho","exe_ext":"","build_only":false,"run_mode":"host_native","run_supported":false,"run_unsupported_reason":"macos-x64 cannot run on host linux/amd64","supports_debug_info":true,"supports_release_optimize":true},
    {"triple":"wasm32-wasi","status":"supported","os":"wasi","arch":"wasm32","abi":"wasi","format":"wasm","exe_ext":".wasm","build_only":false,"run_mode":"wasi_runner","run_supported":false,"run_unsupported_reason":"cannot run target wasm32-wasi: missing WASI runner: need wasmtime or node","supports_debug_info":false,"supports_release_optimize":true},
    {"triple":"wasm32-web","status":"supported","os":"web","arch":"wasm32","abi":"web","format":"wasm","exe_ext":".wasm","build_only":false,"run_mode":"web_runner","run_supported":false,"run_unsupported_reason":"web runner unavailable: chromium-compatible executable not found","supports_debug_info":false,"supports_release_optimize":true}
  ]
}`)
	if err := validateTargetsReport(raw); err != nil {
		t.Fatalf("validate targets without WASI runner: %v", err)
	}
}

func TestValidateTargetsReportRejectsLinuxHostNativeMarkedUnsupported(t *testing.T) {
	raw := []byte(`{
  "supported":["linux-x64","windows-x64","macos-x64","wasm32-wasi","wasm32-web"],
  "build_only":[],
  "planned":[],
  "targets":[
    {"triple":"linux-x64","status":"supported","os":"linux","arch":"x64","abi":"sysv","format":"elf","exe_ext":"","build_only":false,"run_mode":"host_native","run_supported":false,"run_unsupported_reason":"linux-x64 cannot run on host linux/amd64","supports_debug_info":true,"supports_release_optimize":true},
    {"triple":"windows-x64","status":"supported","os":"windows","arch":"x64","abi":"win64","format":"pe","exe_ext":".exe","build_only":false,"run_mode":"host_native","run_supported":false,"run_unsupported_reason":"windows-x64 cannot run on host linux/amd64","supports_debug_info":true,"supports_release_optimize":true},
    {"triple":"macos-x64","status":"supported","os":"macos","arch":"x64","abi":"sysv","format":"macho","exe_ext":"","build_only":false,"run_mode":"host_native","run_supported":false,"run_unsupported_reason":"macos-x64 cannot run on host linux/amd64","supports_debug_info":true,"supports_release_optimize":true},
    {"triple":"wasm32-wasi","status":"supported","os":"wasi","arch":"wasm32","abi":"wasi","format":"wasm","exe_ext":".wasm","build_only":false,"run_mode":"wasi_runner","run_runner":"wasmtime","run_supported":true,"supports_debug_info":false,"supports_release_optimize":true},
    {"triple":"wasm32-web","status":"supported","os":"web","arch":"wasm32","abi":"web","format":"wasm","exe_ext":".wasm","build_only":false,"run_mode":"web_runner","run_supported":false,"run_unsupported_reason":"web runner unavailable: chromium-compatible executable not found","supports_debug_info":false,"supports_release_optimize":true}
  ]
}`)
	err := validateTargetsReport(raw)
	if err == nil {
		t.Fatalf("expected linux host-native run_supported=false failure")
	}
	if !strings.Contains(err.Error(), "linux-x64") || !strings.Contains(err.Error(), "run_supported") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateTargetsReportRejectsWrongOrder(t *testing.T) {
	raw := []byte(`{"supported":["windows-x64","linux-x64","macos-x64","wasm32-wasi","wasm32-web"],"build_only":[],"planned":[],"targets":[]}`)
	if err := validateTargetsReport(raw); err == nil {
		t.Fatalf("expected wrong-order failure")
	}
}

func TestValidateTargetsReportRejectsUnknownFields(t *testing.T) {
	raw := []byte(`{"supported":["linux-x64","windows-x64","macos-x64","wasm32-wasi","wasm32-web"],"build_only":[],"planned":[],"targets":[],"extra":true}`)
	if err := validateTargetsReport(raw); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown top-level field failure, got %v", err)
	}
	raw = []byte(`{
  "supported":["linux-x64","windows-x64","macos-x64","wasm32-wasi","wasm32-web"],
  "build_only":[],
  "planned":[],
  "targets":[
    {"triple":"linux-x64","status":"supported","os":"linux","arch":"x64","abi":"sysv","format":"elf","exe_ext":"","build_only":false,"run_supported":true,"supports_debug_info":true,"supports_release_optimize":true,"extra":true}
  ]
}`)
	if err := validateTargetsReport(raw); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown nested field failure, got %v", err)
	}
}

func TestValidateTargetsReportRejectsDuplicate(t *testing.T) {
	if err := validateTargetList("supported", []string{"linux-x64", "linux-x64"}, []string{"linux-x64", "linux-x64"}); err == nil {
		t.Fatalf("expected duplicate failure")
	}
}

func TestValidateTargetsReportRejectsMissingMetadata(t *testing.T) {
	raw := []byte(`{"supported":["linux-x64","windows-x64","macos-x64","wasm32-wasi","wasm32-web"],"build_only":[],"planned":[]}`)
	if err := validateTargetsReport(raw); err == nil {
		t.Fatalf("expected missing metadata failure")
	}
}

func TestValidateTargetsReportRejectsWrongABI(t *testing.T) {
	raw := []byte(`{
  "supported":["linux-x64","windows-x64","macos-x64","wasm32-wasi","wasm32-web"],
  "build_only":[],
  "planned":[],
  "targets":[
    {"triple":"linux-x64","status":"supported","os":"linux","arch":"x64","abi":"win64","format":"elf","exe_ext":"","build_only":false,"run_supported":true,"supports_debug_info":true,"supports_release_optimize":true},
    {"triple":"windows-x64","status":"supported","os":"windows","arch":"x64","abi":"win64","format":"pe","exe_ext":".exe","build_only":false,"run_supported":false,"supports_debug_info":true,"supports_release_optimize":true},
    {"triple":"macos-x64","status":"supported","os":"macos","arch":"x64","abi":"sysv","format":"macho","exe_ext":"","build_only":false,"run_supported":false,"supports_debug_info":true,"supports_release_optimize":true},
    {"triple":"wasm32-wasi","status":"supported","os":"wasi","arch":"wasm32","abi":"wasi","format":"wasm","exe_ext":".wasm","build_only":false,"run_supported":false,"run_unsupported_reason":"cannot run target wasm32-wasi: missing WASI runner: need wasmtime or node","supports_debug_info":false,"supports_release_optimize":true},
    {"triple":"wasm32-web","status":"supported","os":"web","arch":"wasm32","abi":"web","format":"wasm","exe_ext":".wasm","build_only":false,"run_supported":false,"run_unsupported_reason":"web runner unavailable: chromium-compatible executable not found","supports_debug_info":false,"supports_release_optimize":true}
  ]
}`)
	if err := validateTargetsReport(raw); err == nil {
		t.Fatalf("expected wrong ABI failure")
	}
}

func TestValidateTargetsReportRejectsMissingStatus(t *testing.T) {
	raw := []byte(`{
  "supported":["linux-x64","windows-x64","macos-x64","wasm32-wasi","wasm32-web"],
  "build_only":[],
  "planned":[],
  "targets":[
    {"triple":"linux-x64","os":"linux","arch":"x64","abi":"sysv","format":"elf","exe_ext":"","build_only":false,"run_supported":true,"supports_debug_info":true,"supports_release_optimize":true},
    {"triple":"windows-x64","status":"supported","os":"windows","arch":"x64","abi":"win64","format":"pe","exe_ext":".exe","build_only":false,"run_supported":false,"supports_debug_info":true,"supports_release_optimize":true},
    {"triple":"macos-x64","status":"supported","os":"macos","arch":"x64","abi":"sysv","format":"macho","exe_ext":"","build_only":false,"run_supported":false,"supports_debug_info":true,"supports_release_optimize":true},
    {"triple":"wasm32-wasi","status":"supported","os":"wasi","arch":"wasm32","abi":"wasi","format":"wasm","exe_ext":".wasm","build_only":false,"run_supported":false,"run_unsupported_reason":"cannot run target wasm32-wasi: missing WASI runner: need wasmtime or node","supports_debug_info":false,"supports_release_optimize":true},
    {"triple":"wasm32-web","status":"supported","os":"web","arch":"wasm32","abi":"web","format":"wasm","exe_ext":".wasm","build_only":false,"run_supported":false,"run_unsupported_reason":"web runner unavailable: chromium-compatible executable not found","supports_debug_info":false,"supports_release_optimize":true}
  ]
}`)
	if err := validateTargetsReport(raw); err == nil {
		t.Fatalf("expected missing status failure")
	}
}
