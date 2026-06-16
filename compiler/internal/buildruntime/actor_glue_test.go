package buildruntime

import (
	"testing"

	"tetra_language/compiler/internal/format/tobj"
	"tetra_language/compiler/internal/ir"
)

func TestBuildActorGlueObjectGeneratesMissingGlue(t *testing.T) {
	rt := &tobj.Object{}
	var gotFuncs []ir.IRFunc
	glue, built, err := BuildActorGlueObject(
		rt,
		"linux-x64",
		[]string{"main"},
		nil,
		func(funcs []ir.IRFunc, dataPrefix [][]byte) (*tobj.Object, error) {
			if len(dataPrefix) != 0 {
				t.Fatalf("actor glue codegen data prefix len = %d, want 0", len(dataPrefix))
			}
			gotFuncs = append(gotFuncs, funcs...)
			return &tobj.Object{Code: []byte{0x01}}, nil
		},
	)
	if err != nil {
		t.Fatalf("BuildActorGlueObject error = %v", err)
	}
	if !built {
		t.Fatalf("BuildActorGlueObject built = false, want true")
	}
	if glue == nil {
		t.Fatalf("BuildActorGlueObject returned nil glue")
	}
	if glue.Target != "linux-x64" || glue.Module != "__actorsglue" {
		t.Fatalf("glue identity = (%q, %q), want linux-x64/__actorsglue", glue.Target, glue.Module)
	}
	if len(gotFuncs) != 2 {
		t.Fatalf("actor glue funcs len = %d, want 2", len(gotFuncs))
	}
	if gotFuncs[0].Name != "__tetra_actor_dispatch" || gotFuncs[1].Name != "__tetra_actor_main_entry_id" {
		t.Fatalf("actor glue funcs = %q, %q", gotFuncs[0].Name, gotFuncs[1].Name)
	}
}

func TestBuildActorGlueObjectSkipsExistingGlue(t *testing.T) {
	rt := &tobj.Object{Symbols: []tobj.Symbol{
		{Name: "__tetra_actor_dispatch"},
		{Name: "__tetra_actor_main_entry_id"},
	}}
	called := false
	glue, built, err := BuildActorGlueObject(
		rt,
		"linux-x64",
		[]string{"main"},
		nil,
		func(funcs []ir.IRFunc, dataPrefix [][]byte) (*tobj.Object, error) {
			called = true
			return &tobj.Object{}, nil
		},
	)
	if err != nil {
		t.Fatalf("BuildActorGlueObject error = %v", err)
	}
	if built || glue != nil {
		t.Fatalf("BuildActorGlueObject built=%v glue=%v, want no glue", built, glue)
	}
	if called {
		t.Fatalf("BuildActorGlueObject called codegen despite existing glue symbols")
	}
}
