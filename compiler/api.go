package compiler

import (
	"tetra_language/compiler/internal/backend/linux_x64"
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

const (
	RelocCallRel32  = tobj.RelocCallRel32
	RelocIATDisp32  = tobj.RelocIATDisp32
	RelocDataDisp32 = tobj.RelocDataDisp32
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

func Check(prog *Program) (*CheckedProgram, error) {
	return semantics.Check(prog)
}

func CheckWorld(world *World) (*CheckedProgram, error) {
	return semantics.CheckWorld(world)
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

func CodegenObjectLinuxX64(funcs []IRFunc) (*Object, error) {
	return linux_x64.CodegenObjectLinuxX64(funcs)
}

func CodegenObjectWindowsX64(funcs []IRFunc) (*Object, error) {
	return windows_x64.CodegenObjectWindowsX64(funcs)
}

func CodegenObjectMacOSX64(funcs []IRFunc) (*Object, error) {
	return macos_x64.CodegenObjectMacOSX64(funcs)
}

func LinkLinuxX64(objects []*Object, mainName string) (*elf.Image, error) {
	return linker.LinkLinuxX64(objects, mainName)
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
