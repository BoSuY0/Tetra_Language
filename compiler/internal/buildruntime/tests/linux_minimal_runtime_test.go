package buildruntime_test

import (
	"testing"

	"tetra_language/compiler/internal/buildruntime"
)

func hasRuntimeSymbol(objSymbols []string, name string) bool {
	for _, sym := range objSymbols {
		if sym == name {
			return true
		}
	}
	return false
}

func TestLinuxX86FilesystemRuntimeObject(t *testing.T) {
	obj := buildruntime.BuildLinuxX86FilesystemRuntimeObject()
	if obj.Target != "linux-x86" || obj.Module != "__linux_x86_fsrt" {
		t.Fatalf(
			"filesystem runtime identity = (%q, %q), want linux-x86/__linux_x86_fsrt",
			obj.Target,
			obj.Module,
		)
	}
	if len(obj.Code) == 0 || len(obj.Symbols) != 1 {
		t.Fatalf(
			"filesystem runtime code/symbols = (%d, %d), want non-empty/1",
			len(obj.Code),
			len(obj.Symbols),
		)
	}
	sym := obj.Symbols[0]
	if sym.Name != "__tetra_fs_exists" || !sym.HasSignature || sym.ParamSlots != 3 ||
		sym.ReturnSlots != 1 {
		t.Fatalf("filesystem runtime symbol = %#v", sym)
	}
}

func TestLinuxX86NetRuntimeObjectAndAppend(t *testing.T) {
	obj := buildruntime.BuildLinuxX86BasicNetRuntimeObject()
	if obj.Target != "linux-x86" || obj.Module != "__linux_x86_netrt" {
		t.Fatalf(
			"net runtime identity = (%q, %q), want linux-x86/__linux_x86_netrt",
			obj.Target,
			obj.Module,
		)
	}
	if len(obj.Code) == 0 || len(obj.Symbols) != len(buildruntime.RequiredNetRuntimeSymbols()) {
		t.Fatalf(
			"net runtime code/symbols = (%d, %d), want non-empty/%d",
			len(obj.Code),
			len(obj.Symbols),
			len(buildruntime.RequiredNetRuntimeSymbols()),
		)
	}

	fs := buildruntime.BuildLinuxX86FilesystemRuntimeObject()
	if err := buildruntime.AppendLinuxX86BasicNetRuntimeObject(fs); err != nil {
		t.Fatalf("AppendLinuxX86BasicNetRuntimeObject error = %v", err)
	}
	names := make([]string, 0, len(fs.Symbols))
	for _, sym := range fs.Symbols {
		names = append(names, sym.Name)
	}
	for _, name := range buildruntime.RequiredNetRuntimeSymbols() {
		if !hasRuntimeSymbol(names, name) {
			t.Fatalf("appended linux-x86 runtime missing symbol %s", name)
		}
	}
}

func TestLinuxX32FilesystemAndNetRuntimeObjects(t *testing.T) {
	fs := buildruntime.BuildLinuxX32FilesystemRuntimeObject()
	if fs.Target != "linux-x32" || fs.Module != "__linux_x32_fsrt" {
		t.Fatalf(
			"filesystem runtime identity = (%q, %q), want linux-x32/__linux_x32_fsrt",
			fs.Target,
			fs.Module,
		)
	}
	if err := buildruntime.AppendLinuxX32BasicNetRuntimeObject(fs); err != nil {
		t.Fatalf("AppendLinuxX32BasicNetRuntimeObject error = %v", err)
	}
	names := make([]string, 0, len(fs.Symbols))
	for _, sym := range fs.Symbols {
		names = append(names, sym.Name)
	}
	for _, name := range buildruntime.RequiredNetRuntimeSymbols() {
		if !hasRuntimeSymbol(names, name) {
			t.Fatalf("appended linux-x32 runtime missing symbol %s", name)
		}
	}

	net := buildruntime.BuildLinuxX32BasicNetRuntimeObject()
	if net.Target != "linux-x32" || net.Module != "__linux_x32_netrt" {
		t.Fatalf(
			"net runtime identity = (%q, %q), want linux-x32/__linux_x32_netrt",
			net.Target,
			net.Module,
		)
	}
	if len(net.Code) == 0 || len(net.Symbols) != len(buildruntime.RequiredNetRuntimeSymbols()) {
		t.Fatalf(
			"net runtime code/symbols = (%d, %d), want non-empty/%d",
			len(net.Code),
			len(net.Symbols),
			len(buildruntime.RequiredNetRuntimeSymbols()),
		)
	}
}
