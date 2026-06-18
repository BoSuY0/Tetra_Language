package windows_x64

import (
	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/backend/x64abi"
	"tetra_language/compiler/internal/backend/x64core"
	"tetra_language/compiler/internal/backend/x64obj"
	"tetra_language/compiler/internal/format/tobj"
	"tetra_language/compiler/internal/ir"
)

func CodegenObjectWindowsX64(funcs []ir.IRFunc) (*tobj.Object, error) {
	return CodegenObjectWindowsX64WithOptions(funcs, x64.CodegenOptions{})
}

func CodegenObjectWindowsX64WithOptions(
	funcs []ir.IRFunc,
	opt x64.CodegenOptions,
) (*tobj.Object, error) {
	return CodegenObjectWindowsX64WithOptionsAndDataPrefix(funcs, nil, opt)
}

func CodegenObjectWindowsX64WithOptionsAndDataPrefix(
	funcs []ir.IRFunc,
	dataPrefix [][]byte,
	opt x64.CodegenOptions,
) (*tobj.Object, error) {
	obj, err := x64obj.BuildObjectWithDataPrefix(
		funcs,
		dataPrefix,
		x64core.NewEmitFunc(x64abi.NewWin64()),
		opt,
		x64obj.Options{
			CollectImports: true,
		},
	)
	if err != nil {
		return nil, err
	}
	obj.Target = "windows-x64"
	return obj, nil
}
