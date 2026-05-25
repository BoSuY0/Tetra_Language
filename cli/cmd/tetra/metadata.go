package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"runtime"
	"strings"

	"tetra_language/compiler"
	ctarget "tetra_language/compiler/target"
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
	UIRuntimeStatus         string `json:"ui_runtime_status"`
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

type formatsReport struct {
	Formats []compiler.FormatInfo `json:"formats"`
}

type featuresReport struct {
	Schema   string                 `json:"schema"`
	Version  string                 `json:"version"`
	Features []compiler.FeatureInfo `json:"features"`
}

func runTargets(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("targets", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "text", "output format: text or json")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "targets does not accept positional arguments")
		return 2
	}
	report := targetsReport{
		Supported: ctarget.SupportedTriples(),
		BuildOnly: ctarget.BuildOnlyTriples(),
		Planned:   ctarget.PlannedTriples(),
		Targets:   buildTargetReportEntries(),
	}
	switch *format {
	case "text", "":
		fmt.Fprintln(stdout, "Supported targets:")
		for _, triple := range report.Supported {
			fmt.Fprintf(stdout, "  %s\n", describeTargetForText(triple))
		}
		fmt.Fprintln(stdout, "Build-only targets:")
		for _, triple := range report.BuildOnly {
			fmt.Fprintf(stdout, "  %s\n", describeTargetForText(triple))
		}
		fmt.Fprintln(stdout, "Planned targets:")
		for _, triple := range report.Planned {
			fmt.Fprintf(stdout, "  %s\n", triple)
		}
		return 0
	case "json":
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	default:
		fmt.Fprintln(stderr, "unsupported --format")
		return 2
	}
}

func runFeatures(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("features", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "text", "output format: text or json")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "features does not accept positional arguments")
		return 2
	}
	report := featuresReport{
		Schema:   "tetra.features.v1",
		Version:  compiler.Version(),
		Features: compiler.FeatureRegistry(),
	}
	switch *format {
	case "text", "":
		fmt.Fprintf(stdout, "Tetra features (%s):\n", report.Version)
		for _, feature := range report.Features {
			fmt.Fprintf(stdout, "  %s [%s] - %s\n", feature.ID, feature.Status, feature.Name)
		}
		return 0
	case "json":
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	default:
		fmt.Fprintln(stderr, "unsupported --format")
		return 2
	}
}

func runFormats(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("formats", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "text", "output format: text or json")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "formats does not accept positional arguments")
		return 2
	}
	report := formatsReport{Formats: compiler.T4Formats()}
	switch *format {
	case "text", "":
		fmt.Fprintln(stdout, "T4 formats:")
		for _, item := range report.Formats {
			suffix := item.Extension
			if suffix == "" {
				suffix = item.FileName
			}
			markers := []string{item.Role}
			if item.Primary {
				markers = append(markers, "primary")
			}
			if item.Legacy {
				markers = append(markers, "legacy")
			}
			fmt.Fprintf(stdout, "  %s - %s (%s)\n", suffix, item.Name, strings.Join(markers, ", "))
		}
		return 0
	case "json":
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	default:
		fmt.Fprintln(stderr, "unsupported --format")
		return 2
	}
}

func buildTargetReportEntries() []targetReportEntry {
	host, hostOK := hostTarget()
	triples := append([]string{}, ctarget.SupportedTriples()...)
	triples = append(triples, ctarget.BuildOnlyTriples()...)
	triples = append(triples, ctarget.PlannedTriples()...)
	out := make([]targetReportEntry, 0, len(triples))
	for _, triple := range triples {
		tgt, err := ctarget.Parse(triple)
		if err != nil {
			continue
		}
		buildOnly := ctarget.IsBuildOnlyTarget(tgt.Triple)
		runSupported, runRunner, runUnsupportedReason := targetRunSupport(tgt, host, hostOK)
		out = append(out, targetReportEntry{
			Triple:                  tgt.Triple,
			Status:                  tgt.Status.String(),
			OS:                      tgt.OS.String(),
			Arch:                    tgt.Arch.String(),
			ABI:                     tgt.ABI.String(),
			DataModel:               tgt.DataModel.String(),
			Format:                  tgt.Format.String(),
			ExeExt:                  tgt.ExeExt,
			BuildOnly:               buildOnly,
			RunMode:                 tgt.RunMode.String(),
			RunRunner:               runRunner,
			RunSupported:            runSupported,
			RunUnsupportedReason:    runUnsupportedReason,
			UIRuntimeContract:       ctarget.UIRuntimeContract(tgt.Triple),
			UIRuntimeStatus:         ctarget.UIRuntimeStatus(tgt.Triple),
			UIRuntimeEvidence:       ctarget.UIRuntimeEvidence(tgt.Triple),
			PointerWidthBits:        tgt.PointerWidthBits,
			RegisterWidthBits:       tgt.RegisterWidthBits,
			NativeIntWidthBits:      tgt.NativeIntWidthBits,
			Endian:                  tgt.Endian.String(),
			StackAlignmentBytes:     tgt.StackAlignmentBytes,
			MaxAtomicWidthBits:      tgt.MaxAtomicWidthBits,
			AtomicWidthBits:         tgt.AtomicWidthBits(),
			AtomicPointerWidthBits:  atomicPointerWidthBits(tgt),
			UnsupportedReason:       tgt.UnsupportedReason,
			SupportsDebugInfo:       tgt.SupportsDebugInfo,
			SupportsReleaseOptimize: tgt.SupportsReleaseOptimize,
		})
	}
	return out
}

func atomicPointerWidthBits(tgt ctarget.Target) int {
	layout, err := tgt.AtomicPointerLayout()
	if err != nil {
		return 0
	}
	return layout.WidthBits
}

func targetRunSupport(tgt ctarget.Target, host string, hostOK bool) (bool, string, string) {
	switch tgt.RunMode {
	case ctarget.RunModeHostNative:
		if hostOK && host == tgt.Triple {
			return true, "", ""
		}
		return false, "", fmt.Sprintf("%s cannot run on host %s/%s", tgt.Triple, runtime.GOOS, runtime.GOARCH)
	case ctarget.RunModeHostProbed:
		if !ctarget.IsBuildOnlyTarget(tgt.Triple) {
			return false, "", fmt.Sprintf("%s host_probed run mode is only valid for build-only native targets", tgt.Triple)
		}
		if canRunBuildOnlyNativeTargetOnHost(tgt) {
			return true, "", ""
		}
		return false, "", buildOnlyNativeRunUnsupportedReason(tgt)
	case ctarget.RunModeWASIRunner:
		runner, err := discoverWASIRunner("")
		if err != nil {
			return false, "", err.Error()
		}
		return true, runner.Name, ""
	case ctarget.RunModeWebRunner:
		runner, err := discoverWebRuntimeRunner("")
		if err != nil {
			return false, "", err.Error()
		}
		return true, runner.Name, ""
	case ctarget.RunModeUnsupported:
		if tgt.UnsupportedReason != "" {
			return false, "", tgt.UnsupportedReason
		}
		return false, "", fmt.Sprintf("%s has unsupported runtime mode", tgt.Triple)
	default:
		return false, "", fmt.Sprintf("%s has unknown runtime mode %s", tgt.Triple, tgt.RunMode.String())
	}
}

func describeTargetForText(triple string) string {
	tgt, err := ctarget.Parse(triple)
	if err != nil {
		return triple
	}
	parts := []string{
		triple,
		"os=" + tgt.OS.String(),
		"arch=" + tgt.Arch.String(),
		"abi=" + tgt.ABI.String(),
		"data_model=" + tgt.DataModel.String(),
		"format=" + tgt.Format.String(),
		fmt.Sprintf("ptr=%d", tgt.PointerWidthBits),
		fmt.Sprintf("reg=%d", tgt.RegisterWidthBits),
		fmt.Sprintf("native_int=%d", tgt.NativeIntWidthBits),
		"endian=" + tgt.Endian.String(),
	}
	if tgt.ExeExt != "" {
		parts = append(parts, "exe_ext="+tgt.ExeExt)
	}
	if ctarget.IsBuildOnlyTarget(triple) {
		parts = append(parts, "build-only")
	}
	if tgt.RunMode != ctarget.RunModeUnknown {
		parts = append(parts, "run_mode="+tgt.RunMode.String())
	}
	if tgt.UnsupportedReason != "" {
		parts = append(parts, "unsupported_reason="+tgt.UnsupportedReason)
	}
	return strings.Join(parts, " ")
}
