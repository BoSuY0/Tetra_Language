package main

import (
	"errors"
	"strings"
	"testing"
)

func TestReadTargetsReportDefaultsToTargetsCommand(t *testing.T) {
	old := runTargetsCommand
	runTargetsCommand = func() ([]byte, error) {
		return []byte(`{
  "supported":["linux-x64","windows-x64","macos-x64","wasm32-wasi","wasm32-web"],
  "build_only":["linux-x86","linux-x32"],
  "planned":[],
  "targets":[
    {"triple":"linux-x64","status":"supported","os":"linux","arch":"x64","abi":"sysv","format":"elf","exe_ext":"","build_only":false,"run_mode":"host_native","run_supported":true,"supports_debug_info":true,"supports_release_optimize":true},
    {"triple":"windows-x64","status":"supported","os":"windows","arch":"x64","abi":"win64","format":"pe","exe_ext":".exe","build_only":false,"run_mode":"host_native","run_supported":false,"run_unsupported_reason":"windows-x64 cannot run on host linux/amd64","supports_debug_info":true,"supports_release_optimize":true},
    {"triple":"macos-x64","status":"supported","os":"macos","arch":"x64","abi":"sysv","format":"macho","exe_ext":"","build_only":false,"run_mode":"host_native","run_supported":false,"run_unsupported_reason":"macos-x64 cannot run on host linux/amd64","supports_debug_info":true,"supports_release_optimize":true},
    {"triple":"wasm32-wasi","status":"supported","os":"wasi","arch":"wasm32","abi":"wasi","format":"wasm","exe_ext":".wasm","build_only":false,"run_mode":"wasi_runner","run_runner":"wasmtime","run_supported":true,"supports_debug_info":false,"supports_release_optimize":true},
    {"triple":"wasm32-web","status":"supported","os":"web","arch":"wasm32","abi":"web","format":"wasm","exe_ext":".wasm","build_only":false,"run_mode":"web_runner","run_supported":false,"run_unsupported_reason":"web runner unavailable: chromium-compatible executable not found","supports_debug_info":false,"supports_release_optimize":true},
    {"triple":"linux-x86","status":"build_only","os":"linux","arch":"x86","abi":"i386-sysv","format":"elf","exe_ext":"","build_only":true,"run_mode":"host_probed","run_supported":false,"run_unsupported_reason":"host does not support Linux i386 execution; no host fallback is allowed","supports_debug_info":false,"supports_release_optimize":false},
    {"triple":"linux-x32","status":"build_only","os":"linux","arch":"x64","abi":"x32-sysv","format":"elf","exe_ext":"","build_only":true,"run_mode":"host_probed","run_supported":false,"run_unsupported_reason":"host does not support Linux x32 ABI execution; no host fallback is allowed","supports_debug_info":false,"supports_release_optimize":false}
  ]
}`), nil
	}
	defer func() { runTargetsCommand = old }()

	raw, err := readTargetsReport("")
	if err != nil {
		t.Fatalf("read default targets report: %v", err)
	}
	if err := validateTargetsReport(raw); err != nil {
		t.Fatalf("validate default targets report: %v", err)
	}
}

func TestReadTargetsReportReportsTargetsCommandFailure(t *testing.T) {
	old := runTargetsCommand
	runTargetsCommand = func() ([]byte, error) {
		return []byte("runner failed"), errors.New("exit 127")
	}
	defer func() { runTargetsCommand = old }()

	_, err := readTargetsReport("")
	if err == nil || !strings.Contains(err.Error(), "runner failed") || !strings.Contains(err.Error(), "exit 127") {
		t.Fatalf("unexpected default targets command error: %v", err)
	}
}

func TestValidateTargetsReportAcceptsExpectedShape(t *testing.T) {
	raw := []byte(`{
  "supported":["linux-x64","windows-x64","macos-x64","wasm32-wasi","wasm32-web"],
  "build_only":["linux-x86","linux-x32"],
  "planned":[],
  "targets":[
    {"triple":"linux-x64","status":"supported","os":"linux","arch":"x64","abi":"sysv","format":"elf","exe_ext":"","build_only":false,"run_mode":"host_native","run_supported":true,"supports_debug_info":true,"supports_release_optimize":true},
    {"triple":"windows-x64","status":"supported","os":"windows","arch":"x64","abi":"win64","format":"pe","exe_ext":".exe","build_only":false,"run_mode":"host_native","run_supported":false,"run_unsupported_reason":"windows-x64 cannot run on host linux/amd64","supports_debug_info":true,"supports_release_optimize":true},
    {"triple":"macos-x64","status":"supported","os":"macos","arch":"x64","abi":"sysv","format":"macho","exe_ext":"","build_only":false,"run_mode":"host_native","run_supported":false,"run_unsupported_reason":"macos-x64 cannot run on host linux/amd64","supports_debug_info":true,"supports_release_optimize":true},
    {"triple":"wasm32-wasi","status":"supported","os":"wasi","arch":"wasm32","abi":"wasi","format":"wasm","exe_ext":".wasm","build_only":false,"run_mode":"wasi_runner","run_runner":"wasmtime","run_supported":true,"supports_debug_info":false,"supports_release_optimize":true},
    {"triple":"wasm32-web","status":"supported","os":"web","arch":"wasm32","abi":"web","format":"wasm","exe_ext":".wasm","build_only":false,"run_mode":"web_runner","run_supported":false,"run_unsupported_reason":"web runner unavailable: chromium-compatible executable not found","supports_debug_info":false,"supports_release_optimize":true},
    {"triple":"linux-x86","status":"build_only","os":"linux","arch":"x86","abi":"i386-sysv","format":"elf","exe_ext":"","build_only":true,"run_mode":"host_probed","run_supported":false,"run_unsupported_reason":"host does not support Linux i386 execution; no host fallback is allowed","supports_debug_info":false,"supports_release_optimize":false},
    {"triple":"linux-x32","status":"build_only","os":"linux","arch":"x64","abi":"x32-sysv","format":"elf","exe_ext":"","build_only":true,"run_mode":"host_probed","run_supported":false,"run_unsupported_reason":"host does not support Linux x32 ABI execution; no host fallback is allowed","supports_debug_info":false,"supports_release_optimize":false}
  ]
}`)
	if err := validateTargetsReport(raw); err != nil {
		t.Fatalf("validate targets: %v", err)
	}
}

func TestValidateTargetsReportAcceptsMissingWASIRunner(t *testing.T) {
	raw := []byte(`{
  "supported":["linux-x64","windows-x64","macos-x64","wasm32-wasi","wasm32-web"],
  "build_only":["linux-x86","linux-x32"],
  "planned":[],
  "targets":[
    {"triple":"linux-x64","status":"supported","os":"linux","arch":"x64","abi":"sysv","format":"elf","exe_ext":"","build_only":false,"run_mode":"host_native","run_supported":true,"supports_debug_info":true,"supports_release_optimize":true},
    {"triple":"windows-x64","status":"supported","os":"windows","arch":"x64","abi":"win64","format":"pe","exe_ext":".exe","build_only":false,"run_mode":"host_native","run_supported":false,"run_unsupported_reason":"windows-x64 cannot run on host linux/amd64","supports_debug_info":true,"supports_release_optimize":true},
    {"triple":"macos-x64","status":"supported","os":"macos","arch":"x64","abi":"sysv","format":"macho","exe_ext":"","build_only":false,"run_mode":"host_native","run_supported":false,"run_unsupported_reason":"macos-x64 cannot run on host linux/amd64","supports_debug_info":true,"supports_release_optimize":true},
    {"triple":"wasm32-wasi","status":"supported","os":"wasi","arch":"wasm32","abi":"wasi","format":"wasm","exe_ext":".wasm","build_only":false,"run_mode":"wasi_runner","run_supported":false,"run_unsupported_reason":"cannot run target wasm32-wasi: missing WASI runner: need wasmtime or node","supports_debug_info":false,"supports_release_optimize":true},
    {"triple":"wasm32-web","status":"supported","os":"web","arch":"wasm32","abi":"web","format":"wasm","exe_ext":".wasm","build_only":false,"run_mode":"web_runner","run_supported":false,"run_unsupported_reason":"web runner unavailable: chromium-compatible executable not found","supports_debug_info":false,"supports_release_optimize":true},
    {"triple":"linux-x86","status":"build_only","os":"linux","arch":"x86","abi":"i386-sysv","format":"elf","exe_ext":"","build_only":true,"run_mode":"host_probed","run_supported":false,"run_unsupported_reason":"host does not support Linux i386 execution; no host fallback is allowed","supports_debug_info":false,"supports_release_optimize":false},
    {"triple":"linux-x32","status":"build_only","os":"linux","arch":"x64","abi":"x32-sysv","format":"elf","exe_ext":"","build_only":true,"run_mode":"host_probed","run_supported":false,"run_unsupported_reason":"host does not support Linux x32 ABI execution; no host fallback is allowed","supports_debug_info":false,"supports_release_optimize":false}
  ]
}`)
	if err := validateTargetsReport(raw); err != nil {
		t.Fatalf("validate targets without WASI runner: %v", err)
	}
}

func TestValidateTargetsReportRejectsLinuxHostNativeMarkedUnsupported(t *testing.T) {
	raw := []byte(`{
  "supported":["linux-x64","windows-x64","macos-x64","wasm32-wasi","wasm32-web"],
  "build_only":["linux-x86","linux-x32"],
  "planned":[],
  "targets":[
    {"triple":"linux-x64","status":"supported","os":"linux","arch":"x64","abi":"sysv","format":"elf","exe_ext":"","build_only":false,"run_mode":"host_native","run_supported":false,"run_unsupported_reason":"linux-x64 cannot run on host linux/amd64","supports_debug_info":true,"supports_release_optimize":true},
    {"triple":"windows-x64","status":"supported","os":"windows","arch":"x64","abi":"win64","format":"pe","exe_ext":".exe","build_only":false,"run_mode":"host_native","run_supported":false,"run_unsupported_reason":"windows-x64 cannot run on host linux/amd64","supports_debug_info":true,"supports_release_optimize":true},
    {"triple":"macos-x64","status":"supported","os":"macos","arch":"x64","abi":"sysv","format":"macho","exe_ext":"","build_only":false,"run_mode":"host_native","run_supported":false,"run_unsupported_reason":"macos-x64 cannot run on host linux/amd64","supports_debug_info":true,"supports_release_optimize":true},
    {"triple":"wasm32-wasi","status":"supported","os":"wasi","arch":"wasm32","abi":"wasi","format":"wasm","exe_ext":".wasm","build_only":false,"run_mode":"wasi_runner","run_runner":"wasmtime","run_supported":true,"supports_debug_info":false,"supports_release_optimize":true},
    {"triple":"wasm32-web","status":"supported","os":"web","arch":"wasm32","abi":"web","format":"wasm","exe_ext":".wasm","build_only":false,"run_mode":"web_runner","run_supported":false,"run_unsupported_reason":"web runner unavailable: chromium-compatible executable not found","supports_debug_info":false,"supports_release_optimize":true},
    {"triple":"linux-x86","status":"build_only","os":"linux","arch":"x86","abi":"i386-sysv","format":"elf","exe_ext":"","build_only":true,"run_mode":"host_probed","run_supported":false,"run_unsupported_reason":"host does not support Linux i386 execution; no host fallback is allowed","supports_debug_info":false,"supports_release_optimize":false},
    {"triple":"linux-x32","status":"build_only","os":"linux","arch":"x64","abi":"x32-sysv","format":"elf","exe_ext":"","build_only":true,"run_mode":"host_probed","run_supported":false,"run_unsupported_reason":"host does not support Linux x32 ABI execution; no host fallback is allowed","supports_debug_info":false,"supports_release_optimize":false}
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

func TestValidateRunContractAcceptsHostProbedBuildOnly(t *testing.T) {
	entry := targetReportEntry{
		Triple:               "linux-x32",
		Status:               "build_only",
		BuildOnly:            true,
		RunMode:              "host_probed",
		RunSupported:         false,
		RunUnsupportedReason: "host does not support Linux x32 ABI execution; no host fallback is allowed",
	}
	if err := validateRunContract(entry); err != nil {
		t.Fatalf("validate host-probed run contract: %v", err)
	}
	entry.RunSupported = true
	entry.RunUnsupportedReason = ""
	if err := validateRunContract(entry); err != nil {
		t.Fatalf("validate supported host-probed run contract: %v", err)
	}
	entry.BuildOnly = false
	if err := validateRunContract(entry); err == nil || !strings.Contains(err.Error(), "build-only") {
		t.Fatalf("expected non-build-only host-probed failure, got %v", err)
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
