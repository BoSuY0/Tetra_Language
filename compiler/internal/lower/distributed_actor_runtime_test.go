package lower

import (
	"testing"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/semantics"
)

func TestLowerDistributedActorRuntimeBuiltins(t *testing.T) {
	src := []byte(`
func worker() -> Int:
    return 0

func main() -> Int
uses actors, runtime:
    let _connected: Int = core.actor_node_connect(2, 5010)
    let peer: actor = core.spawn_remote(2, "worker")
    let _status: Int = core.actor_node_status(2)
    return core.send(peer, 7)
`)
	prog, err := frontend.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := semantics.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	irProg, err := Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}

	mainFn := findIRFuncByName(t, irProg.Funcs, "main")
	for _, name := range []string{
		"__tetra_actor_node_connect",
		"__tetra_actor_spawn_remote",
		"__tetra_actor_node_status",
		"__tetra_actor_send",
	} {
		if !hasIRCallName(mainFn, name) {
			t.Fatalf("main is missing %s call: %#v", name, mainFn.Instrs)
		}
	}
}
