package compiler

import (
	"tetra_language/compiler/internal/backend/linux_x64"
	"tetra_language/compiler/internal/backend/linux_x86"
	"tetra_language/compiler/internal/backend/macos_x64"
	"tetra_language/compiler/internal/backend/windows_x64"
	"tetra_language/compiler/internal/format/elf"
	"tetra_language/compiler/internal/format/macho"
	"tetra_language/compiler/internal/format/pe"
	"tetra_language/compiler/internal/format/tobj"
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/linker"
	"tetra_language/compiler/internal/lower"
	"tetra_language/compiler/internal/module"
	"tetra_language/compiler/internal/semantics"
)

type Program = frontend.Program
type FileAST = frontend.FileAST
type CheckedProgram = semantics.CheckedProgram
type IRProgram = ir.IRProgram
type IRFunc = ir.IRFunc
type UILoweredBundle = lower.UILoweredBundle

type Object = tobj.Object
type Symbol = tobj.Symbol
type Reloc = tobj.Reloc
type RelocKind = tobj.RelocKind

type World = module.World
type WorldOptions = module.LoadOptions
type ModuleRoot = module.ModuleRoot
type CheckOptions = semantics.CheckOptions

const (
	RelocCallRel32      = tobj.RelocCallRel32
	RelocIATDisp32      = tobj.RelocIATDisp32
	RelocDataDisp32     = tobj.RelocDataDisp32
	RelocFuncAddrDisp32 = tobj.RelocFuncAddrDisp32
	RelocDataAbs32      = tobj.RelocDataAbs32
	RelocFuncAddrAbs32  = tobj.RelocFuncAddrAbs32
)

func Parse(src []byte) (*Program, error) {
	return frontend.Parse(src)
}

func ParseFile(src []byte, filename string) (*FileAST, error) {
	return frontend.ParseFile(src, filename)
}

func NormalizeFlowForMigration(src []byte, filename string) ([]byte, error) {
	return frontend.NormalizeFlowForMigration(src, filename)
}

func LoadWorld(entryPath string) (*World, error) {
	return module.LoadWorld(entryPath)
}

func LoadWorldOpt(entryPath string, opt WorldOptions) (*World, error) {
	return module.LoadWorldOpt(entryPath, opt)
}

// Check validates a single already-parsed source program.
// Boundary: this API does not resolve filesystem modules/import graphs; use
// LoadWorld + CheckWorld for cross-module checking.
func Check(prog *Program) (*CheckedProgram, error) {
	return semantics.Check(prog)
}

// CheckWorld validates a module graph loaded from filesystem sources.
func CheckWorld(world *World) (*CheckedProgram, error) {
	return semantics.CheckWorld(world)
}

func CheckWorldOpt(world *World, opt CheckOptions) (*CheckedProgram, error) {
	return semantics.CheckWorldOpt(world, opt)
}

func Lower(checked *CheckedProgram) (*IRProgram, error) {
	return lower.Lower(checked)
}

func LowerModule(checked *CheckedProgram, module string) ([]IRFunc, error) {
	return lower.LowerModule(checked, module)
}

func LowerModules(checked *CheckedProgram) (map[string][]IRFunc, error) {
	return lower.LowerModules(checked)
}

func LowerUI(checked *CheckedProgram) (*UILoweredBundle, error) {
	return lower.LowerUI(checked)
}

func VerifyIRProgram(prog *IRProgram) error {
	return lower.VerifyProgram(prog)
}

func VerifyIRFunc(fn IRFunc) error {
	return lower.VerifyFunc(fn)
}

func CodegenObjectLinuxX64(funcs []IRFunc) (*Object, error) {
	if err := verifyIRFuncs(funcs); err != nil {
		return nil, err
	}
	return linux_x64.CodegenObjectLinuxX64(funcs)
}

func CodegenObjectLinuxX86(funcs []IRFunc) (*Object, error) {
	if err := verifyIRFuncs(funcs); err != nil {
		return nil, err
	}
	return linux_x86.CodegenObjectLinuxX86(funcs)
}

func CodegenObjectWindowsX64(funcs []IRFunc) (*Object, error) {
	if err := verifyIRFuncs(funcs); err != nil {
		return nil, err
	}
	return windows_x64.CodegenObjectWindowsX64(funcs)
}

func CodegenObjectMacOSX64(funcs []IRFunc) (*Object, error) {
	if err := verifyIRFuncs(funcs); err != nil {
		return nil, err
	}
	return macos_x64.CodegenObjectMacOSX64(funcs)
}

func verifyIRFuncs(funcs []IRFunc) error {
	for _, fn := range funcs {
		if err := lower.VerifyFunc(fn); err != nil {
			return err
		}
	}
	return nil
}

func LinkLinuxX64(objects []*Object, mainName string) (*elf.Image, error) {
	return linker.LinkLinuxX64(objects, mainName)
}

func LinkLinuxX32(objects []*Object, mainName string) (*elf.Image, error) {
	return linker.LinkLinuxX32(objects, mainName)
}

func LinkLinuxX86(objects []*Object, mainName string) (*elf.Image, error) {
	return linker.LinkLinuxX86(objects, mainName)
}

func LinkWindowsX64(objects []*Object, mainName string) (*pe.PEImage, error) {
	return linker.LinkWindowsX64(objects, mainName)
}

func LinkMacOSX64(objects []*Object, mainName string) (*macho.MachOImage, error) {
	return linker.LinkMacOSX64(objects, mainName)
}

func WriteELF64LinuxX64(path string, img *elf.Image) error {
	return elf.WriteELF64LinuxX64(path, img)
}

func WriteELF32LinuxX32(path string, img *elf.Image) error {
	return elf.WriteELF32LinuxX32(path, img)
}

func WriteELF32LinuxX86(path string, img *elf.Image) error {
	return elf.WriteELF32LinuxX86(path, img)
}

func WritePE64WindowsX64(path string, img *pe.PEImage) error {
	return pe.WritePE64WindowsX64(path, img)
}

func WriteMachO64MacOSX64(path string, img *macho.MachOImage) error {
	return macho.WriteMachO64MacOSX64(path, img)
}

func ReadObject(path string) (*Object, error) {
	return tobj.ReadObject(path)
}

func WriteObject(path string, obj *Object) error {
	return tobj.WriteObject(path, obj)
}
