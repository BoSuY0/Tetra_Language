package testkit

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/lower"
	"tetra_language/compiler/internal/module"
	"tetra_language/compiler/internal/semantics"
)

func CheckProgram(src string) error {
	prog, err := frontend.Parse([]byte(src))
	if err != nil {
		return err
	}
	_, err = semantics.Check(prog)
	return err
}

func CheckFileProgram(src string) error {
	file, err := frontend.ParseFile([]byte(src), "test.tetra")
	if err != nil {
		return err
	}
	world := &module.World{
		EntryModule: "",
		Files:       []*frontend.FileAST{file},
		ByModule:    map[string]*frontend.FileAST{"": file},
	}
	checked, err := semantics.CheckWorld(world)
	if err != nil {
		return err
	}
	_, err = lower.Lower(checked)
	return err
}

func CheckFileSemanticProgram(src string) error {
	file, err := frontend.ParseFile([]byte(src), "test.tetra")
	if err != nil {
		return err
	}
	world := &module.World{
		EntryModule: file.Module,
		Files:       []*frontend.FileAST{file},
		ByModule:    map[string]*frontend.FileAST{file.Module: file},
	}
	_, err = semantics.CheckWorld(world)
	return err
}

func RequireCheckErrorContains(t testing.TB, src string, want string) {
	t.Helper()
	err := CheckProgram(src)
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected error containing %q, got: %v", want, err)
	}
}

func RequireCheckOK(t testing.TB, src string) {
	t.Helper()
	if err := CheckProgram(src); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}

func RequireFileCheckErrorContains(t testing.TB, src string, want string) {
	t.Helper()
	err := CheckFileProgram(src)
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected error containing %q, got: %v", want, err)
	}
}

func RequireFileCheckOK(t testing.TB, src string) {
	t.Helper()
	if err := CheckFileProgram(src); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}

func RequireFileSemanticCheckErrorContains(t testing.TB, src string, want string) {
	t.Helper()
	err := CheckFileSemanticProgram(src)
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected error containing %q, got: %v", want, err)
	}
}

func RequireFileSemanticCheckOK(t testing.TB, src string) {
	t.Helper()
	if err := CheckFileSemanticProgram(src); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}
