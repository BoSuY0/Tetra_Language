package actorsrt

import (
	"testing"

	"tetra_language/compiler/internal/format/tobj"
)

func TestBuiltinRuntimeExportsActorStateSymbols(t *testing.T) {
	entries := []string{"main"}
	builders := []struct {
		name  string
		build func([]string) (*tobj.Object, error)
	}{
		{name: "linux-x64", build: BuildLinuxX64},
		{name: "macos-x64", build: BuildMacOSX64},
		{name: "windows-x64", build: BuildWindowsX64},
	}

	for _, tt := range builders {
		t.Run(tt.name, func(t *testing.T) {
			obj, err := tt.build(entries)
			if err != nil {
				t.Fatalf("build runtime: %v", err)
			}
			if !hasSymbol(obj.Symbols, "__tetra_actor_state_load") {
				t.Fatalf("runtime missing __tetra_actor_state_load")
			}
			if !hasSymbol(obj.Symbols, "__tetra_actor_state_store") {
				t.Fatalf("runtime missing __tetra_actor_state_store")
			}
		})
	}
}

func hasSymbol(symbols []tobj.Symbol, want string) bool {
	for _, sym := range symbols {
		if sym.Name == want {
			return true
		}
	}
	return false
}
