package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
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
	Format                  string `json:"format"`
	ExeExt                  string `json:"exe_ext"`
	BuildOnly               bool   `json:"build_only"`
	RunSupported            bool   `json:"run_supported"`
	RunUnsupportedReason    string `json:"run_unsupported_reason,omitempty"`
	SupportsDebugInfo       bool   `json:"supports_debug_info"`
	SupportsReleaseOptimize bool   `json:"supports_release_optimize"`
}

func main() {
	var path string
	flag.StringVar(&path, "report", "", "path to tetra targets --format=json output")
	flag.Parse()
	if path == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
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
	if err := validateTargetList("supported", report.Supported, []string{"linux-x64", "windows-x64", "macos-x64"}); err != nil {
		return err
	}
	if err := validateTargetList("build_only", report.BuildOnly, []string{"wasm32-wasi", "wasm32-web"}); err != nil {
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
		{Triple: "linux-x64", Status: "supported", OS: "linux", Arch: "x64", ABI: "sysv", Format: "elf", ExeExt: "", BuildOnly: false, SupportsDebugInfo: true, SupportsReleaseOptimize: true},
		{Triple: "windows-x64", Status: "supported", OS: "windows", Arch: "x64", ABI: "win64", Format: "pe", ExeExt: ".exe", BuildOnly: false, SupportsDebugInfo: true, SupportsReleaseOptimize: true},
		{Triple: "macos-x64", Status: "supported", OS: "macos", Arch: "x64", ABI: "sysv", Format: "macho", ExeExt: "", BuildOnly: false, SupportsDebugInfo: true, SupportsReleaseOptimize: true},
		{Triple: "wasm32-wasi", Status: "build_only", OS: "wasi", Arch: "wasm32", ABI: "wasi", Format: "wasm", ExeExt: ".wasm", BuildOnly: true, SupportsDebugInfo: false, SupportsReleaseOptimize: true},
		{Triple: "wasm32-web", Status: "build_only", OS: "web", Arch: "wasm32", ABI: "web", Format: "wasm", ExeExt: ".wasm", BuildOnly: true, SupportsDebugInfo: false, SupportsReleaseOptimize: true},
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
		if got[i].BuildOnly && got[i].RunSupported {
			return fmt.Errorf("target metadata[%s].run_supported must be false for build-only targets", got[i].Triple)
		}
		if got[i].BuildOnly {
			if !strings.Contains(got[i].RunUnsupportedReason, "build-only") || !strings.Contains(got[i].RunUnsupportedReason, "does not provide a production runtime runner") {
				return fmt.Errorf("target metadata[%s].run_unsupported_reason must explain build-only artifact-only runtime status", got[i].Triple)
			}
		} else if !got[i].RunSupported && got[i].RunUnsupportedReason == "" {
			return fmt.Errorf("target metadata[%s].run_unsupported_reason is required when run_supported is false", got[i].Triple)
		}
		if got[i].SupportsDebugInfo != want[i].SupportsDebugInfo {
			return fmt.Errorf("target metadata[%s].supports_debug_info = %v, want %v", got[i].Triple, got[i].SupportsDebugInfo, want[i].SupportsDebugInfo)
		}
		if got[i].SupportsReleaseOptimize != want[i].SupportsReleaseOptimize {
			return fmt.Errorf("target metadata[%s].supports_release_optimize = %v, want %v", got[i].Triple, got[i].SupportsReleaseOptimize, want[i].SupportsReleaseOptimize)
		}
	}
	return nil
}
