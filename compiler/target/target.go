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

type Target struct {
	Triple                  string
	OS                      OS
	Arch                    Arch
	ABI                     ABI
	Format                  Format
	ExeExt                  string
	CollectImports          bool
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

func SupportedTriples() []string {
	return []string{"linux-x64", "windows-x64", "macos-x64"}
}

func BuildOnlyTriples() []string {
	return []string{"wasm32-wasi", "wasm32-web"}
}

func PlannedTriples() []string {
	return []string{}
}

func WASMTriples() []string {
	return []string{"wasm32-wasi", "wasm32-web"}
}

func Parse(triple string) (Target, error) {
	switch triple {
	case "linux-x64":
		return Target{
			Triple:                  "linux-x64",
			OS:                      OSLinux,
			Arch:                    ArchX64,
			ABI:                     ABISysV,
			Format:                  FormatELF,
			ExeExt:                  "",
			CollectImports:          false,
			SupportsDebugInfo:       true,
			SupportsReleaseOptimize: true,
		}, nil
	case "windows-x64":
		return Target{
			Triple:                  "windows-x64",
			OS:                      OSWindows,
			Arch:                    ArchX64,
			ABI:                     ABIWin64,
			Format:                  FormatPE,
			ExeExt:                  ".exe",
			CollectImports:          true,
			SupportsDebugInfo:       true,
			SupportsReleaseOptimize: true,
		}, nil
	case "macos-x64":
		return Target{
			Triple:                  "macos-x64",
			OS:                      OSMacOS,
			Arch:                    ArchX64,
			ABI:                     ABISysV,
			Format:                  FormatMachO,
			ExeExt:                  "",
			CollectImports:          false,
			SupportsDebugInfo:       true,
			SupportsReleaseOptimize: true,
		}, nil
	case "wasm32-wasi":
		return Target{
			Triple:                  "wasm32-wasi",
			OS:                      OSWASI,
			Arch:                    ArchWASM32,
			ABI:                     ABIWASI,
			Format:                  FormatWASM,
			ExeExt:                  ".wasm",
			CollectImports:          false,
			SupportsDebugInfo:       false,
			SupportsReleaseOptimize: true,
		}, nil
	case "wasm32-web":
		return Target{
			Triple:                  "wasm32-web",
			OS:                      OSWeb,
			Arch:                    ArchWASM32,
			ABI:                     ABIWeb,
			Format:                  FormatWASM,
			ExeExt:                  ".wasm",
			CollectImports:          false,
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
