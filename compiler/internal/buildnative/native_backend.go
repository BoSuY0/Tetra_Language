package buildnative

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/actorsrt"
	"tetra_language/compiler/internal/backend/linux_x32"
	"tetra_language/compiler/internal/backend/linux_x64"
	"tetra_language/compiler/internal/backend/linux_x86"
	"tetra_language/compiler/internal/backend/macos_x64"
	"tetra_language/compiler/internal/backend/windows_x64"
	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/buildapi"
	"tetra_language/compiler/internal/format/elf"
	"tetra_language/compiler/internal/format/macho"
	"tetra_language/compiler/internal/format/pe"
	"tetra_language/compiler/internal/format/tobj"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/linker"
	ctarget "tetra_language/compiler/target"
)

type CodegenFunc func([]ir.IRFunc, [][]byte) (*tobj.Object, error)

type ExecutableBackend struct {
	Name         string
	OS           ctarget.OS
	Format       ctarget.Format
	Codegen      func(x64.CodegenOptions) CodegenFunc
	Link         func(outputPath string, objects []*tobj.Object, mainName string) error
	ActorRuntime func(actorEntries []string) (*tobj.Object, error)
}

func CodegenForTarget(tgt ctarget.Target, opt buildapi.BuildOptions) (CodegenFunc, error) {
	backend, ok := ExecutableBackendForTarget(tgt)
	if !ok {
		return nil, fmt.Errorf("unsupported target: %s", tgt.Triple)
	}
	return backend.Codegen(CodegenOptionsForTarget(tgt, opt)), nil
}

func ExecutableBackendForTarget(tgt ctarget.Target) (ExecutableBackend, bool) {
	if tgt.Triple == "linux-x86" {
		return LinuxX86ExecutableBackend(), true
	}
	if tgt.Arch != ctarget.ArchX64 {
		return ExecutableBackend{}, false
	}
	if tgt.Triple == "linux-x32" {
		return LinuxX32ExecutableBackend(), true
	}
	backend, ok := ExecutableBackends()[tgt.OS]
	if !ok || backend.Format != tgt.Format {
		return ExecutableBackend{}, false
	}
	return backend, true
}

func CodegenOptions(opt buildapi.BuildOptions) x64.CodegenOptions {
	return CodegenOptionsForTarget(ctarget.Target{}, opt)
}

func CodegenOptionsForTarget(tgt ctarget.Target, opt buildapi.BuildOptions) x64.CodegenOptions {
	return x64.CodegenOptions{
		IslandsDebug:                     opt.IslandsDebug,
		DebugInfo:                        opt.DebugInfo,
		ReleaseOptimize:                  opt.ReleaseOptimize,
		PointerWidthBits:                 tgt.PointerWidthBits,
		NativeIntWidthBits:               tgt.NativeIntWidthBits,
		RegisterWidthBits:                tgt.RegisterWidthBits,
		EmitRuntimeHeapTelemetry:         opt.EmitRuntimeHeapTelemetry,
		RuntimeHeapTelemetryActorDomains: opt.RuntimeHeapTelemetryActorDomains,
		RuntimeHeapTelemetryDir:          opt.RuntimeHeapTelemetryDir,
		RuntimeHeapTelemetryProgram:      opt.RuntimeHeapTelemetryProgram,
		RuntimeHeapTelemetryMain:         opt.RuntimeHeapTelemetryMain,
	}
}

func ValidateRuntimeHeapTelemetryBuildOptions(tgt ctarget.Target, opt buildapi.BuildOptions) error {
	if strings.TrimSpace(opt.RuntimeHeapTelemetryDir) != "" && !opt.EmitRuntimeHeapTelemetry {
		return fmt.Errorf("runtime heap telemetry dir requires EmitRuntimeHeapTelemetry")
	}
	if !opt.EmitRuntimeHeapTelemetry {
		return nil
	}
	if tgt.Triple != "linux-x64" {
		return fmt.Errorf(
			"runtime heap telemetry is only supported for linux-x64, got %s",
			tgt.Triple,
		)
	}
	if strings.TrimSpace(opt.RuntimeHeapTelemetryDir) == "" {
		return fmt.Errorf("runtime heap telemetry requires RuntimeHeapTelemetryDir")
	}
	return nil
}

func ExecutableBackends() map[ctarget.OS]ExecutableBackend {
	return map[ctarget.OS]ExecutableBackend{
		ctarget.OSLinux: {
			Name:   "linux-x64",
			OS:     ctarget.OSLinux,
			Format: ctarget.FormatELF,
			Codegen: func(opt x64.CodegenOptions) CodegenFunc {
				return func(funcs []ir.IRFunc, dataPrefix [][]byte) (*tobj.Object, error) {
					return linux_x64.CodegenObjectLinuxX64WithOptionsAndDataPrefix(
						funcs,
						dataPrefix,
						opt,
					)
				}
			},
			Link: func(outputPath string, objects []*tobj.Object, mainName string) error {
				img, err := linker.LinkLinuxX64(objects, mainName)
				if err != nil {
					return err
				}
				return elf.WriteELF64LinuxX64(outputPath, img)
			},
			ActorRuntime: actorsrt.BuildLinuxX64,
		},
		ctarget.OSWindows: {
			Name:   "windows-x64",
			OS:     ctarget.OSWindows,
			Format: ctarget.FormatPE,
			Codegen: func(opt x64.CodegenOptions) CodegenFunc {
				return func(funcs []ir.IRFunc, dataPrefix [][]byte) (*tobj.Object, error) {
					return windows_x64.CodegenObjectWindowsX64WithOptionsAndDataPrefix(
						funcs,
						dataPrefix,
						opt,
					)
				}
			},
			Link: func(outputPath string, objects []*tobj.Object, mainName string) error {
				img, err := linker.LinkWindowsX64(objects, mainName)
				if err != nil {
					return err
				}
				return pe.WritePE64WindowsX64(outputPath, img)
			},
			ActorRuntime: actorsrt.BuildWindowsX64,
		},
		ctarget.OSMacOS: {
			Name:   "macos-x64",
			OS:     ctarget.OSMacOS,
			Format: ctarget.FormatMachO,
			Codegen: func(opt x64.CodegenOptions) CodegenFunc {
				return func(funcs []ir.IRFunc, dataPrefix [][]byte) (*tobj.Object, error) {
					return macos_x64.CodegenObjectMacOSX64WithOptionsAndDataPrefix(
						funcs,
						dataPrefix,
						opt,
					)
				}
			},
			Link: func(outputPath string, objects []*tobj.Object, mainName string) error {
				img, err := linker.LinkMacOSX64(objects, mainName)
				if err != nil {
					return err
				}
				return macho.WriteMachO64MacOSX64(outputPath, img)
			},
			ActorRuntime: actorsrt.BuildMacOSX64,
		},
	}
}

func LinuxX32ExecutableBackend() ExecutableBackend {
	return ExecutableBackend{
		Name:   "linux-x32",
		OS:     ctarget.OSLinux,
		Format: ctarget.FormatELF,
		Codegen: func(opt x64.CodegenOptions) CodegenFunc {
			return func(funcs []ir.IRFunc, dataPrefix [][]byte) (*tobj.Object, error) {
				return linux_x32.CodegenObjectLinuxX32WithOptionsAndDataPrefix(
					funcs,
					dataPrefix,
					opt,
				)
			}
		},
		Link: func(outputPath string, objects []*tobj.Object, mainName string) error {
			img, err := linker.LinkLinuxX32(objects, mainName)
			if err != nil {
				return err
			}
			return elf.WriteELF32LinuxX32(outputPath, img)
		},
	}
}

func LinuxX86ExecutableBackend() ExecutableBackend {
	return ExecutableBackend{
		Name:   "linux-x86",
		OS:     ctarget.OSLinux,
		Format: ctarget.FormatELF,
		Codegen: func(opt x64.CodegenOptions) CodegenFunc {
			return func(funcs []ir.IRFunc, dataPrefix [][]byte) (*tobj.Object, error) {
				return linux_x86.CodegenObjectLinuxX86WithOptionsAndDataPrefix(
					funcs,
					dataPrefix,
					opt,
				)
			}
		},
		Link: func(outputPath string, objects []*tobj.Object, mainName string) error {
			img, err := linker.LinkLinuxX86(objects, mainName)
			if err != nil {
				return err
			}
			return elf.WriteELF32LinuxX86(outputPath, img)
		},
	}
}
