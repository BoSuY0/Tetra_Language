package macos_x64

import (
	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/backend/x64abi"
	"tetra_language/compiler/internal/backend/x64core"
	"tetra_language/compiler/internal/backend/x64obj"
	"tetra_language/compiler/internal/format/tobj"
	"tetra_language/compiler/internal/ir"
)

func CodegenObjectMacOSX64(funcs []ir.IRFunc) (*tobj.Object, error) {
	return CodegenObjectMacOSX64WithOptions(funcs, x64.CodegenOptions{})
}

func CodegenObjectMacOSX64WithOptions(funcs []ir.IRFunc, opt x64.CodegenOptions) (*tobj.Object, error) {
	return CodegenObjectMacOSX64WithOptionsAndDataPrefix(funcs, nil, opt)
}

func CodegenObjectMacOSX64WithOptionsAndDataPrefix(funcs []ir.IRFunc, dataPrefix [][]byte, opt x64.CodegenOptions) (*tobj.Object, error) {
	obj, err := x64obj.BuildObjectWithDataPrefix(funcs, dataPrefix, x64core.NewEmitFunc(x64abi.MacSysV()), opt, x64obj.Options{
		CollectImports: false,
	})
	if err != nil {
		return nil, err
	}
	obj.Target = "macos-x64"
	return obj, nil
}
