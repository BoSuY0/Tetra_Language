package linux_x64

import (
	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/backend/x64abi"
	"tetra_language/compiler/internal/backend/x64core"
	"tetra_language/compiler/internal/backend/x64obj"
	"tetra_language/compiler/internal/format/tobj"
	"tetra_language/compiler/internal/ir"
)

func CodegenObjectLinuxX64(funcs []ir.IRFunc) (*tobj.Object, error) {
	return CodegenObjectLinuxX64WithOptions(funcs, x64.CodegenOptions{})
}

func CodegenObjectLinuxX64WithOptions(funcs []ir.IRFunc, opt x64.CodegenOptions) (*tobj.Object, error) {
	return CodegenObjectLinuxX64WithOptionsAndDataPrefix(funcs, nil, opt)
}

func CodegenObjectLinuxX64WithOptionsAndDataPrefix(funcs []ir.IRFunc, dataPrefix [][]byte, opt x64.CodegenOptions) (*tobj.Object, error) {
	return x64obj.BuildObjectWithDataPrefix(funcs, dataPrefix, x64core.NewEmitFunc(x64abi.LinuxSysV()), opt, x64obj.Options{
		CollectImports: false,
	})
}
