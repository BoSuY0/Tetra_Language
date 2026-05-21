package linux_x32

import (
	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/backend/x64abi"
	"tetra_language/compiler/internal/backend/x64core"
	"tetra_language/compiler/internal/backend/x64obj"
	"tetra_language/compiler/internal/format/tobj"
	"tetra_language/compiler/internal/ir"
)

func CodegenObjectLinuxX32(funcs []ir.IRFunc) (*tobj.Object, error) {
	return CodegenObjectLinuxX32WithOptions(funcs, x64.CodegenOptions{})
}

func CodegenObjectLinuxX32WithOptions(funcs []ir.IRFunc, opt x64.CodegenOptions) (*tobj.Object, error) {
	return CodegenObjectLinuxX32WithOptionsAndDataPrefix(funcs, nil, opt)
}

func CodegenObjectLinuxX32WithOptionsAndDataPrefix(funcs []ir.IRFunc, dataPrefix [][]byte, opt x64.CodegenOptions) (*tobj.Object, error) {
	opt = linuxX32CodegenOptions(opt)
	obj, err := x64obj.BuildObjectWithDataPrefix(funcs, dataPrefix, x64core.NewEmitFunc(x64abi.LinuxX32SysV()), opt, x64obj.Options{
		CollectImports: false,
	})
	if err != nil {
		return nil, err
	}
	obj.Target = "linux-x32"
	return obj, nil
}

func linuxX32CodegenOptions(opt x64.CodegenOptions) x64.CodegenOptions {
	if opt.PointerWidthBits == 0 {
		opt.PointerWidthBits = 32
	}
	if opt.NativeIntWidthBits == 0 {
		opt.NativeIntWidthBits = 32
	}
	if opt.RegisterWidthBits == 0 {
		opt.RegisterWidthBits = 64
	}
	return opt
}
