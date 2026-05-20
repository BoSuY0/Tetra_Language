package compiler_test

import (
	"strings"
	"testing"

	compiler "tetra_language/compiler"
)

func TestExportAddsAliasSymbol(t *testing.T) {
	src := "@export(\"X\")\nfun foo(): i32 { return 0 }\nfun main(): i32 { return foo() }\n"
	prog, err := compiler.Parse([]byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	irProg, err := compiler.Lower(checked)
	if err != nil {
		t.Fatalf("lower: %v", err)
	}
	obj, err := compiler.CodegenObjectLinuxX64(irProg.Funcs)
	if err != nil {
		t.Fatalf("codegen object: %v", err)
	}

	symbols := make(map[string]uint32, len(obj.Symbols))
	for _, sym := range obj.Symbols {
		symbols[sym.Name] = sym.Offset
	}
	if _, ok := symbols["foo"]; !ok {
		t.Fatalf("missing foo symbol")
	}
	if _, ok := symbols["X"]; !ok {
		t.Fatalf("missing export alias symbol")
	}
	if symbols["X"] != symbols["foo"] {
		t.Fatalf("export alias offset mismatch: X=%d foo=%d", symbols["X"], symbols["foo"])
	}
}

func TestExportRejectsDuplicateNames(t *testing.T) {
	src := "@export(\"X\")\nfun a(): i32 { return 0 }\n@export(\"X\")\nfun b(): i32 { return 0 }\nfun main(): i32 { return a() + b() }\n"
	prog, err := compiler.Parse([]byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "duplicate @export name") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExportRejectsReservedPrefixOutsideInternalModule(t *testing.T) {
	src := "@export(\"__tetra_entry\")\nfun entry(): i32 { return 0 }\nfun main(): i32 { return entry() }\n"
	prog, err := compiler.Parse([]byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "reserved") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExportAllowsReservedPrefixInInternalModule(t *testing.T) {
	src := "module __rt\n@export(\"__tetra_entry\")\nfun entry(): i32 { return 0 }\nfun main(): i32 { return entry() }\n"
	file, err := compiler.ParseFile([]byte(src), "test.tetra")
	if err != nil {
		t.Fatalf("parse file: %v", err)
	}
	world := &compiler.World{
		EntryModule: file.Module,
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{file.Module: file},
	}
	checked, err := compiler.CheckWorld(world)
	if err != nil {
		t.Fatalf("check world: %v", err)
	}
	irProg, err := compiler.Lower(checked)
	if err != nil {
		t.Fatalf("lower: %v", err)
	}
	obj, err := compiler.CodegenObjectLinuxX64(irProg.Funcs)
	if err != nil {
		t.Fatalf("codegen object: %v", err)
	}
	found := false
	for _, sym := range obj.Symbols {
		if sym.Name == "__tetra_entry" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("missing __tetra_entry export symbol")
	}
}
