package semantics

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/frontend"
)

func TestCheckDistributedActorRuntimeBuiltins(t *testing.T) {
	src := []byte(`
func worker() -> Int:
    return 0

func main() -> Int
uses actors, runtime:
    let connected: Int = core.actor_node_connect(2, 5010)
    let peer: actor = core.spawn_remote(2, "worker")
    let sent: Int = core.send(peer, connected)
    return core.actor_node_status(2) + sent
`)
	prog, err := frontend.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if _, err := Check(prog); err != nil {
		t.Fatalf("Check: %v", err)
	}
}

func TestCheckDistributedActorRuntimeBuiltinsRequireRuntimeEffect(t *testing.T) {
	src := []byte(`
func worker() -> Int:
    return 0

func main() -> Int
uses actors:
    let _peer: actor = core.spawn_remote(2, "worker")
    return 0
`)
	prog, err := frontend.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected runtime effect diagnostic")
	}
	if !strings.Contains(err.Error(), "uses effect 'runtime'") {
		t.Fatalf("error = %v", err)
	}
}

func TestCheckSpawnRemoteRejectsNonLiteralTarget(t *testing.T) {
	src := []byte(`
func worker() -> Int:
    return 0

func main() -> Int
uses actors, runtime:
    let name: str = "worker"
    let _peer: actor = core.spawn_remote(2, name)
    return 0
`)
	prog, err := frontend.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected spawn_remote target diagnostic")
	}
	if !strings.Contains(err.Error(), "spawn_remote expects a string literal") {
		t.Fatalf("error = %v", err)
	}
}
