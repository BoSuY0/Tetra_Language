package target

import (
	"fmt"
	"runtime"
)

type UnsupportedTargetError struct {
	Triple  string
	Planned bool
}

func (e UnsupportedTargetError) Error() string {
	if e.Planned {
		return fmt.Sprintf("planned target not implemented: %s", e.Triple)
	}
	return fmt.Sprintf("unsupported target: %s", e.Triple)
}

type OS int

const (
	OSUnknown OS = iota
	OSLinux
	OSWindows
	OSMacOS
	OSWASI
	OSWeb
)

func (o OS) String() string {
	switch o {
	case OSLinux:
		return "linux"
	case OSWindows:
		return "windows"
	case OSMacOS:
		return "macos"
	case OSWASI:
		return "wasi"
	case OSWeb:
		return "web"
	default:
		return "unknown"
	}
}

type Arch int

const (
	ArchUnknown Arch = iota
	ArchX64
	ArchWASM32
)

func (a Arch) String() string {
	switch a {
	case ArchX64:
		return "x64"
	case ArchWASM32:
		return "wasm32"
	default:
		return "unknown"
	}
}

type ABI int

const (
	ABIUnknown ABI = iota
	ABISysV
	ABIWin64
	ABIWASI
	ABIWeb
)

func (a ABI) String() string {
	switch a {
	case ABISysV:
		return "sysv"
	case ABIWin64:
		return "win64"
	case ABIWASI:
		return "wasi"
	case ABIWeb:
		return "web"
	default:
		return "unknown"
	}
}

type Format int

const (
	FormatUnknown Format = iota
	FormatELF
	FormatPE
	FormatMachO
	FormatWASM
)

func (f Format) String() string {
	switch f {
	case FormatELF:
		return "elf"
	case FormatPE:
		return "pe"
	case FormatMachO:
		return "macho"
	case FormatWASM:
		return "wasm"
	default:
		return "unknown"
	}
}

type Status int

const (
	StatusUnknown Status = iota
	StatusSupported
	StatusBuildOnly
	StatusPlanned
)

func (s Status) String() string {
	switch s {
	case StatusSupported:
		return "supported"
	case StatusBuildOnly:
		return "build_only"
	case StatusPlanned:
		return "planned"
	default:
		return "unknown"
	}
}

type RunMode int

const (
	RunModeUnknown RunMode = iota
	RunModeHostNative
	RunModeWASIRunner
	RunModeWebRunner
)

func (m RunMode) String() string {
	switch m {
	case RunModeHostNative:
		return "host_native"
	case RunModeWASIRunner:
		return "wasi_runner"
	case RunModeWebRunner:
		return "web_runner"
	default:
		return "unknown"
	}
}

type Target struct {
	Triple                  string
	Status                  Status
	OS                      OS
	Arch                    Arch
	ABI                     ABI
	Format                  Format
	ExeExt                  string
	CollectImports          bool
	RunMode                 RunMode
	RunRunner               string
	SupportsDebugInfo       bool
	SupportsReleaseOptimize bool
}

func All() []Target {
	out := make([]Target, 0, len(SupportedTriples()))
	for _, triple := range SupportedTriples() {
		t, err := Parse(triple)
		if err == nil {
			out = append(out, t)
		}
	}
	return out
}

func AllBuildable() []Target {
	triples := append([]string{}, SupportedTriples()...)
	triples = append(triples, BuildOnlyTriples()...)
	out := make([]Target, 0, len(triples))
	for _, triple := range triples {
		t, err := Parse(triple)
		if err == nil {
			out = append(out, t)
		}
	}
	return out
}

func SupportedTriples() []string {
	return []string{"linux-x64", "windows-x64", "macos-x64", "wasm32-wasi", "wasm32-web"}
}

func BuildOnlyTriples() []string {
	return []string{}
}

func PlannedTriples() []string {
	return []string{}
}

func WASMTriples() []string {
	return []string{"wasm32-wasi", "wasm32-web"}
}

func ActorRuntimeTriples() []string {
	return []string{"linux-x64", "macos-x64", "windows-x64"}
}

func Parse(triple string) (Target, error) {
	switch triple {
	case "linux-x64":
		return Target{
			Triple:                  "linux-x64",
			Status:                  StatusSupported,
			OS:                      OSLinux,
			Arch:                    ArchX64,
			ABI:                     ABISysV,
			Format:                  FormatELF,
			ExeExt:                  "",
			CollectImports:          false,
			RunMode:                 RunModeHostNative,
			SupportsDebugInfo:       true,
			SupportsReleaseOptimize: true,
		}, nil
	case "windows-x64":
		return Target{
			Triple:                  "windows-x64",
			Status:                  StatusSupported,
			OS:                      OSWindows,
			Arch:                    ArchX64,
			ABI:                     ABIWin64,
			Format:                  FormatPE,
			ExeExt:                  ".exe",
			CollectImports:          true,
			RunMode:                 RunModeHostNative,
			SupportsDebugInfo:       true,
			SupportsReleaseOptimize: true,
		}, nil
	case "macos-x64":
		return Target{
			Triple:                  "macos-x64",
			Status:                  StatusSupported,
			OS:                      OSMacOS,
			Arch:                    ArchX64,
			ABI:                     ABISysV,
			Format:                  FormatMachO,
			ExeExt:                  "",
			CollectImports:          false,
			RunMode:                 RunModeHostNative,
			SupportsDebugInfo:       true,
			SupportsReleaseOptimize: true,
		}, nil
	case "wasm32-wasi":
		return Target{
			Triple:                  "wasm32-wasi",
			Status:                  StatusSupported,
			OS:                      OSWASI,
			Arch:                    ArchWASM32,
			ABI:                     ABIWASI,
			Format:                  FormatWASM,
			ExeExt:                  ".wasm",
			CollectImports:          false,
			RunMode:                 RunModeWASIRunner,
			RunRunner:               "wasmtime",
			SupportsDebugInfo:       false,
			SupportsReleaseOptimize: true,
		}, nil
	case "wasm32-web":
		return Target{
			Triple:                  "wasm32-web",
			Status:                  StatusSupported,
			OS:                      OSWeb,
			Arch:                    ArchWASM32,
			ABI:                     ABIWeb,
			Format:                  FormatWASM,
			ExeExt:                  ".wasm",
			CollectImports:          false,
			RunMode:                 RunModeWebRunner,
			SupportsDebugInfo:       false,
			SupportsReleaseOptimize: true,
		}, nil
	default:
		return Target{}, UnsupportedTargetError{Triple: triple}
	}
}

func IsBuildOnlyTarget(triple string) bool {
	for _, buildOnly := range BuildOnlyTriples() {
		if triple == buildOnly {
			return true
		}
	}
	return false
}

func IsPlannedTarget(triple string) bool {
	for _, planned := range PlannedTriples() {
		if triple == planned {
			return true
		}
	}
	return false
}

func Host() (Target, bool) {
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "linux/amd64":
		t, _ := Parse("linux-x64")
		return t, true
	case "windows/amd64":
		t, _ := Parse("windows-x64")
		return t, true
	case "darwin/amd64":
		t, _ := Parse("macos-x64")
		return t, true
	default:
		return Target{}, false
	}
}
