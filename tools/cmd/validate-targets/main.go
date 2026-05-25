package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

type targetsReport struct {
	Supported []string            `json:"supported"`
	BuildOnly []string            `json:"build_only"`
	Planned   []string            `json:"planned"`
	Targets   []targetReportEntry `json:"targets"`
}

type targetReportEntry struct {
	Triple                  string `json:"triple"`
	Status                  string `json:"status"`
	OS                      string `json:"os"`
	Arch                    string `json:"arch"`
	ABI                     string `json:"abi"`
	DataModel               string `json:"data_model"`
	Format                  string `json:"format"`
	ExeExt                  string `json:"exe_ext"`
	BuildOnly               bool   `json:"build_only"`
	RunMode                 string `json:"run_mode"`
	RunRunner               string `json:"run_runner,omitempty"`
	RunSupported            bool   `json:"run_supported"`
	RunUnsupportedReason    string `json:"run_unsupported_reason,omitempty"`
	UIRuntimeContract       string `json:"ui_runtime_contract,omitempty"`
	UIRuntimeStatus         string `json:"ui_runtime_status,omitempty"`
	UIRuntimeEvidence       string `json:"ui_runtime_evidence,omitempty"`
	PointerWidthBits        int    `json:"pointer_width_bits"`
	RegisterWidthBits       int    `json:"register_width_bits"`
	NativeIntWidthBits      int    `json:"native_int_width_bits"`
	Endian                  string `json:"endian"`
	StackAlignmentBytes     int    `json:"stack_alignment_bytes"`
	MaxAtomicWidthBits      int    `json:"max_atomic_width_bits"`
	AtomicWidthBits         []int  `json:"atomic_width_bits"`
	AtomicPointerWidthBits  int    `json:"atomic_pointer_width_bits"`
	UnsupportedReason       string `json:"unsupported_reason,omitempty"`
	SupportsDebugInfo       bool   `json:"supports_debug_info"`
	SupportsReleaseOptimize bool   `json:"supports_release_optimize"`
}

func main() {
	var path string
	flag.StringVar(&path, "report", "", "path to tetra targets --format=json output")
	flag.Parse()
	var raw []byte
	var err error
	if path == "" {
		raw, err = exec.Command("./tetra", "targets", "--format=json").Output()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to run ./tetra targets --format=json: %v\n", err)
			os.Exit(1)
		}
	} else {
		raw, err = os.ReadFile(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	if err := validateTargetsReport(raw); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateTargetsReport(raw []byte) error {
	var report targetsReport
	if err := decodeStrictJSON(raw, &report); err != nil {
		return fmt.Errorf("invalid targets JSON: %w", err)
	}
	if err := validateTargetList("supported", report.Supported, []string{"linux-x64", "windows-x64", "macos-x64", "wasm32-wasi", "wasm32-web"}); err != nil {
		return err
	}
	if err := validateTargetList("build_only", report.BuildOnly, []string{"linux-x86", "linux-x32"}); err != nil {
		return err
	}
	if err := validateTargetList("planned", report.Planned, []string{}); err != nil {
		return err
	}
	if err := validateTargetMetadata(report.Targets); err != nil {
		return err
	}
	return nil
}

func decodeStrictJSON(raw []byte, out any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	return dec.Decode(out)
}

func validateTargetList(name string, got []string, want []string) error {
	if len(got) != len(want) {
		return fmt.Errorf("%s target count = %d, want %d", name, len(got), len(want))
	}
	seen := map[string]bool{}
	for i, target := range got {
		if target != want[i] {
			return fmt.Errorf("%s target[%d] = %q, want %q", name, i, target, want[i])
		}
		if seen[target] {
			return fmt.Errorf("%s target %q is duplicated", name, target)
		}
		seen[target] = true
	}
	return nil
}

func validateTargetMetadata(got []targetReportEntry) error {
	want := []targetReportEntry{
		{Triple: "linux-x64", Status: "supported", OS: "linux", Arch: "x64", ABI: "sysv", Format: "elf", ExeExt: "", BuildOnly: false, RunMode: "host_native", SupportsDebugInfo: true, SupportsReleaseOptimize: true},
		{Triple: "windows-x64", Status: "supported", OS: "windows", Arch: "x64", ABI: "win64", Format: "pe", ExeExt: ".exe", BuildOnly: false, RunMode: "host_native", SupportsDebugInfo: true, SupportsReleaseOptimize: true},
		{Triple: "macos-x64", Status: "supported", OS: "macos", Arch: "x64", ABI: "sysv", Format: "macho", ExeExt: "", BuildOnly: false, RunMode: "host_native", SupportsDebugInfo: true, SupportsReleaseOptimize: true},
		{Triple: "wasm32-wasi", Status: "supported", OS: "wasi", Arch: "wasm32", ABI: "wasi", Format: "wasm", ExeExt: ".wasm", BuildOnly: false, RunMode: "wasi_runner", SupportsDebugInfo: false, SupportsReleaseOptimize: true},
		{Triple: "wasm32-web", Status: "supported", OS: "web", Arch: "wasm32", ABI: "web", Format: "wasm", ExeExt: ".wasm", BuildOnly: false, RunMode: "web_runner", SupportsDebugInfo: false, SupportsReleaseOptimize: true},
		{Triple: "linux-x86", Status: "build_only", OS: "linux", Arch: "x86", ABI: "i386-sysv", Format: "elf", ExeExt: "", BuildOnly: true, RunMode: "host_probed", SupportsDebugInfo: false, SupportsReleaseOptimize: false},
		{Triple: "linux-x32", Status: "build_only", OS: "linux", Arch: "x64", ABI: "x32-sysv", Format: "elf", ExeExt: "", BuildOnly: true, RunMode: "host_probed", SupportsDebugInfo: false, SupportsReleaseOptimize: false},
	}
	if len(got) != len(want) {
		return fmt.Errorf("target metadata count = %d, want %d", len(got), len(want))
	}
	seen := map[string]bool{}
	for i := range want {
		if seen[got[i].Triple] {
			return fmt.Errorf("target metadata %q is duplicated", got[i].Triple)
		}
		seen[got[i].Triple] = true
		if got[i].Triple != want[i].Triple {
			return fmt.Errorf("target metadata[%d].triple = %q, want %q", i, got[i].Triple, want[i].Triple)
		}
		if got[i].Status != want[i].Status {
			return fmt.Errorf("target metadata[%s].status = %q, want %q", got[i].Triple, got[i].Status, want[i].Status)
		}
		if got[i].OS != want[i].OS || got[i].Arch != want[i].Arch || got[i].ABI != want[i].ABI || got[i].Format != want[i].Format {
			return fmt.Errorf("target metadata[%s] platform = os:%s arch:%s abi:%s format:%s, want os:%s arch:%s abi:%s format:%s",
				got[i].Triple, got[i].OS, got[i].Arch, got[i].ABI, got[i].Format, want[i].OS, want[i].Arch, want[i].ABI, want[i].Format)
		}
		if got[i].ExeExt != want[i].ExeExt {
			return fmt.Errorf("target metadata[%s].exe_ext = %q, want %q", got[i].Triple, got[i].ExeExt, want[i].ExeExt)
		}
		if got[i].BuildOnly != want[i].BuildOnly {
			return fmt.Errorf("target metadata[%s].build_only = %v, want %v", got[i].Triple, got[i].BuildOnly, want[i].BuildOnly)
		}
		if got[i].RunMode != want[i].RunMode {
			return fmt.Errorf("target metadata[%s].run_mode = %q, want %q", got[i].Triple, got[i].RunMode, want[i].RunMode)
		}
		if err := validateRunContract(got[i]); err != nil {
			return err
		}
		if !got[i].RunSupported && got[i].RunUnsupportedReason == "" {
			return fmt.Errorf("target metadata[%s].run_unsupported_reason is required when run_supported is false", got[i].Triple)
		}
		if got[i].SupportsDebugInfo != want[i].SupportsDebugInfo {
			return fmt.Errorf("target metadata[%s].supports_debug_info = %v, want %v", got[i].Triple, got[i].SupportsDebugInfo, want[i].SupportsDebugInfo)
		}
		if got[i].SupportsReleaseOptimize != want[i].SupportsReleaseOptimize {
			return fmt.Errorf("target metadata[%s].supports_release_optimize = %v, want %v", got[i].Triple, got[i].SupportsReleaseOptimize, want[i].SupportsReleaseOptimize)
		}
		if err := validateUIRuntimeMetadata(got[i]); err != nil {
			return err
		}
	}
	return nil
}

func validateUIRuntimeMetadata(entry targetReportEntry) error {
	if entry.UIRuntimeStatus == "" {
		return nil
	}
	wantStatus := map[string]string{
		"linux-x64":   "production",
		"windows-x64": "requires_target_host_evidence",
		"macos-x64":   "requires_target_host_evidence",
		"wasm32-web":  "production",
		"wasm32-wasi": "unsupported",
		"linux-x86":   "unsupported",
		"linux-x32":   "unsupported",
	}[entry.Triple]
	if entry.UIRuntimeStatus != wantStatus {
		return fmt.Errorf("target metadata[%s].ui_runtime_status = %q, want %q", entry.Triple, entry.UIRuntimeStatus, wantStatus)
	}
	if entry.UIRuntimeStatus == "production" || entry.UIRuntimeStatus == "requires_target_host_evidence" {
		if entry.UIRuntimeContract != "tetra.ui.platform.v1" {
			return fmt.Errorf("target metadata[%s].ui_runtime_contract = %q, want tetra.ui.platform.v1", entry.Triple, entry.UIRuntimeContract)
		}
		if strings.TrimSpace(entry.UIRuntimeEvidence) == "" {
			return fmt.Errorf("target metadata[%s].ui_runtime_evidence is required", entry.Triple)
		}
	}
	if (entry.Triple == "windows-x64" || entry.Triple == "macos-x64") && strings.Contains(entry.UIRuntimeStatus, "production") {
		return fmt.Errorf("target metadata[%s] must not mark UI runtime production without target-host evidence", entry.Triple)
	}
	return nil
}

func validateRunContract(entry targetReportEntry) error {
	switch entry.RunMode {
	case "host_native":
		if entry.RunRunner != "" {
			return fmt.Errorf("target metadata[%s].run_runner = %q, want empty for host_native", entry.Triple, entry.RunRunner)
		}
		hostTriple, hostOK := validatorHostTriple()
		if hostOK && entry.Triple == hostTriple {
			if !entry.RunSupported {
				return fmt.Errorf("target metadata[%s].run_supported = false, want true on host %s/%s", entry.Triple, runtime.GOOS, runtime.GOARCH)
			}
			if entry.RunUnsupportedReason != "" {
				return fmt.Errorf("target metadata[%s].run_unsupported_reason must be empty on matching host", entry.Triple)
			}
		} else if entry.RunSupported {
			return fmt.Errorf("target metadata[%s].run_supported = true, want false on host %s/%s", entry.Triple, runtime.GOOS, runtime.GOARCH)
		}
	case "wasi_runner":
		if entry.BuildOnly || entry.Triple != "wasm32-wasi" {
			return fmt.Errorf("target metadata[%s].run_mode wasi_runner is only valid for supported wasm32-wasi target", entry.Triple)
		}
		if entry.RunSupported {
			if entry.RunRunner != "wasmtime" && entry.RunRunner != "node-wasi" {
				return fmt.Errorf("target metadata[%s].run_runner = %q, want wasmtime or node-wasi when run_supported is true", entry.Triple, entry.RunRunner)
			}
			if entry.RunUnsupportedReason != "" {
				return fmt.Errorf("target metadata[%s].run_unsupported_reason must be empty when WASI runner is available", entry.Triple)
			}
		} else {
			if entry.RunRunner != "" {
				return fmt.Errorf("target metadata[%s].run_runner = %q, want empty when WASI runner is unavailable", entry.Triple, entry.RunRunner)
			}
			if !strings.Contains(entry.RunUnsupportedReason, "missing WASI runner") {
				return fmt.Errorf("target metadata[%s].run_unsupported_reason must explain missing WASI runner", entry.Triple)
			}
		}
	case "host_probed":
		if !entry.BuildOnly {
			return fmt.Errorf("target metadata[%s].run_mode host_probed is only valid for build-only native targets", entry.Triple)
		}
		if entry.RunRunner != "" {
			return fmt.Errorf("target metadata[%s].run_runner = %q, want empty for host_probed", entry.Triple, entry.RunRunner)
		}
		if entry.RunSupported {
			if entry.RunUnsupportedReason != "" {
				return fmt.Errorf("target metadata[%s].run_unsupported_reason must be empty when host probe succeeds", entry.Triple)
			}
		} else if !strings.Contains(entry.RunUnsupportedReason, "no host fallback") {
			return fmt.Errorf("target metadata[%s].run_unsupported_reason must explain host probe failure and no host fallback", entry.Triple)
		}
	case "web_runner":
		if entry.Triple != "wasm32-web" || entry.BuildOnly {
			return fmt.Errorf("target metadata[%s].run_mode web_runner is only valid for supported wasm32-web target", entry.Triple)
		}
		if entry.RunSupported {
			if entry.RunRunner == "" {
				return fmt.Errorf("target metadata[%s].run_runner is required when web runner is available", entry.Triple)
			}
			if entry.RunUnsupportedReason != "" {
				return fmt.Errorf("target metadata[%s].run_unsupported_reason must be empty when web runner is available", entry.Triple)
			}
		} else {
			if entry.RunRunner != "" {
				return fmt.Errorf("target metadata[%s].run_runner = %q, want empty when web runner is unavailable", entry.Triple, entry.RunRunner)
			}
			if !strings.Contains(entry.RunUnsupportedReason, "web runner unavailable") && !strings.Contains(entry.RunUnsupportedReason, "browser runner unavailable") {
				return fmt.Errorf("target metadata[%s].run_unsupported_reason must explain missing web runner", entry.Triple)
			}
		}
	default:
		return fmt.Errorf("target metadata[%s].run_mode = %q, want host_native, host_probed, wasi_runner, or web_runner", entry.Triple, entry.RunMode)
	}
	return nil
}

func validatorHostTriple() (string, bool) {
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "linux/amd64":
		return "linux-x64", true
	case "windows/amd64":
		return "windows-x64", true
	case "darwin/amd64":
		return "macos-x64", true
	default:
		return "", false
	}
}
