package buildruntime

import (
	"os"
	"path/filepath"
	"testing"

	"tetra_language/compiler/internal/format/tobj"
	"tetra_language/compiler/internal/ir"
)

func TestBuildEmbeddedSelfHostRuntimeObject(t *testing.T) {
	src, err := os.ReadFile(filepath.FromSlash("../../selfhostrt/time_ilp32.tetra"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	var gotFuncs []ir.IRFunc
	obj, err := BuildEmbeddedSelfHostRuntimeObject(
		"linux-x86",
		src,
		"<test selfhost time_ilp32>",
		func(funcs []ir.IRFunc, dataPrefix [][]byte) (*tobj.Object, error) {
			gotFuncs = append(gotFuncs, funcs...)
			return &tobj.Object{Code: []byte{0x01}}, nil
		},
	)
	if err != nil {
		t.Fatalf("BuildEmbeddedSelfHostRuntimeObject error = %v", err)
	}
	if obj.Target != "linux-x86" || obj.Module != "__selfhostrt" {
		t.Fatalf("runtime object identity = (%q, %q), want linux-x86/__selfhostrt", obj.Target, obj.Module)
	}
	if len(gotFuncs) == 0 {
		t.Fatalf("BuildEmbeddedSelfHostRuntimeObject did not lower any funcs")
	}
}
