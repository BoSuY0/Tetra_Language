package semantics

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/frontend"
)

func TestCheckPreservesStructRepresentationMetadata(t *testing.T) {
	prog, err := frontend.Parse([]byte(`
repr(C) struct Header:
    tag: c_int
    ptr: ptr

struct Packet:
    bytes: []u8

func main() -> Int:
    return 0
`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if got := checked.Types["Header"].Repr; got != frontend.StructReprC {
		t.Fatalf("Header repr = %q, want %q", got, frontend.StructReprC)
	}
	if got := checked.Types["Packet"].Repr; got != frontend.StructReprDefault {
		t.Fatalf("Packet repr = %q, want %q", got, frontend.StructReprDefault)
	}
}

func TestExportedDefaultStructRequiresExplicitRepr(t *testing.T) {
	prog, err := frontend.Parse([]byte(`
struct Pair:
    lo: c_int
    hi: c_int

@export("ffi_pair_c")
func ffi_pair(pair: Pair) -> c_int:
    return pair.lo

func main() -> Int:
    return 0
`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("Check accepted exported default-layout struct")
	}
	for _, want := range []string{"exported function 'ffi_pair'", "parameter 'pair'", "type 'Pair'", "requires explicit repr(C)", "default Tetra layout is compiler-owned"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("diagnostic = %v, want substring %q", err, want)
		}
	}
}

func TestExportedReprCStructPassesExplicitReprGate(t *testing.T) {
	prog, err := frontend.Parse([]byte(`
repr(C) struct Pair:
    lo: c_int
    hi: c_int

@export("ffi_pair_c")
func ffi_pair(pair: Pair) -> c_int:
    return pair.lo

func main() -> Int:
    return 0
`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if _, err := Check(prog); err != nil {
		t.Fatalf("Check rejected repr(C) exported struct at explicit repr gate: %v", err)
	}
}

func TestBuiltinRuntimeABIAggregatesUseReprC(t *testing.T) {
	types := baseTypes()
	for _, name := range []string{
		"task.i32",
		"task.result_i32",
		"actor.msg",
		"actor.recv_result_i32",
		"actor.recv_msg_result",
	} {
		info := types[name]
		if info == nil {
			t.Fatalf("missing builtin type %s", name)
		}
		if got := info.Repr; got != frontend.StructReprC {
			t.Fatalf("%s repr = %q, want %q", name, got, frontend.StructReprC)
		}
	}
}
