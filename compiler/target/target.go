package target

import (
	"fmt"
	"runtime"
)

type OS int

const (
	OSUnknown OS = iota
	OSLinux
	OSWindows
	OSMacOS
)

func (o OS) String() string {
	switch o {
	case OSLinux:
		return "linux"
	case OSWindows:
		return "windows"
	case OSMacOS:
		return "macos"
	default:
		return "unknown"
	}
}

type Arch int

const (
	ArchUnknown Arch = iota
	ArchX64
)

func (a Arch) String() string {
	switch a {
	case ArchX64:
		return "x64"
	default:
		return "unknown"
	}
}

type ABI int

const (
	ABIUnknown ABI = iota
	ABISysV
	ABIWin64
)

func (a ABI) String() string {
	switch a {
	case ABISysV:
		return "sysv"
	case ABIWin64:
		return "win64"
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
)

func (f Format) String() string {
	switch f {
	case FormatELF:
		return "elf"
	case FormatPE:
		return "pe"
	case FormatMachO:
		return "macho"
	default:
		return "unknown"
	}
}

type Target struct {
	Triple         string
	OS             OS
	Arch           Arch
	ABI            ABI
	Format         Format
	ExeExt         string
	CollectImports bool
}

func All() []Target {
	out := make([]Target, 0, 3)
	for _, triple := range []string{"linux-x64", "windows-x64", "macos-x64"} {
		t, err := Parse(triple)
		if err == nil {
			out = append(out, t)
		}
	}
	return out
}

func Parse(triple string) (Target, error) {
	switch triple {
	case "linux-x64":
		return Target{
			Triple:         "linux-x64",
			OS:             OSLinux,
			Arch:           ArchX64,
			ABI:            ABISysV,
			Format:         FormatELF,
			ExeExt:         "",
			CollectImports: false,
		}, nil
	case "windows-x64":
		return Target{
			Triple:         "windows-x64",
			OS:             OSWindows,
			Arch:           ArchX64,
			ABI:            ABIWin64,
			Format:         FormatPE,
			ExeExt:         ".exe",
			CollectImports: true,
		}, nil
	case "macos-x64":
		return Target{
			Triple:         "macos-x64",
			OS:             OSMacOS,
			Arch:           ArchX64,
			ABI:            ABISysV,
			Format:         FormatMachO,
			ExeExt:         "",
			CollectImports: false,
		}, nil
	default:
		return Target{}, fmt.Errorf("unsupported target: %s", triple)
	}
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
