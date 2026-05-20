package compiler_test

import (
	"testing"

	compiler "tetra_language/compiler"
	"tetra_language/compiler/internal/ir"
)

func findIRFunc(t *testing.T, funcs []compiler.IRFunc, name string) compiler.IRFunc {
	t.Helper()
	for _, fn := range funcs {
		if fn.Name == name {
			return fn
		}
	}
	t.Fatalf("missing IR function %q", name)
	return compiler.IRFunc{}
}

func hasIRCall(fn compiler.IRFunc, name string) bool {
	for _, instr := range fn.Instrs {
		if instr.Kind == ir.IRCall && instr.Name == name {
			return true
		}
	}
	return false
}
