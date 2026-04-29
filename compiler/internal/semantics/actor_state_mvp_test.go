package semantics

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/frontend"
)

func TestCheckActorStateBuildsSlotMapping(t *testing.T) {
	src := []byte(`
actor Worker:
    var count: Int = 0
    val step: Int = 2
    const enabled: Bool = true
    func run() -> Int:
        if enabled:
            count = count + step
        return count

func main() -> Int
uses actors:
    let _peer: actor = core.spawn("Worker.run")
    return 0
`)
	prog, err := frontend.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}

	var run CheckedFunc
	found := false
	for _, fn := range checked.Funcs {
		if fn.Name == "Worker.run" {
			run = fn
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("missing checked function Worker.run")
	}
	if len(run.ActorState) != 3 {
		t.Fatalf("actor state count = %d, want 3", len(run.ActorState))
	}

	count := run.ActorState["count"]
	if count.Slot != 0 || !count.Mutable || count.Const || count.TypeName != "i32" || count.Init != 0 {
		t.Fatalf("count field = %#v", count)
	}
	step := run.ActorState["step"]
	if step.Slot != 1 || step.Mutable || step.Const || step.TypeName != "i32" || step.Init != 2 {
		t.Fatalf("step field = %#v", step)
	}
	enabled := run.ActorState["enabled"]
	if enabled.Slot != 2 || enabled.Mutable || !enabled.Const || enabled.TypeName != "bool" || enabled.Init != 1 {
		t.Fatalf("enabled field = %#v", enabled)
	}
}

func TestCheckActorStateRejectsUnsupportedType(t *testing.T) {
	src := []byte(`
actor Worker:
    val title: String = "worker"
    func run() -> Int:
        return 0

func main() -> Int
uses actors:
    let _peer: actor = core.spawn("Worker.run")
    return 0
`)
	prog, err := frontend.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected actor state type diagnostic")
	}
	if !strings.Contains(err.Error(), "type 'str' is not supported in this MVP") {
		t.Fatalf("error = %v", err)
	}
}

func TestCheckActorStateSupportsExtendedScalarTypes(t *testing.T) {
	src := []byte(`
actor Worker:
    var err: task.error = 0
    val byteStep: UInt8 = 7
    const wide: UInt16 = 9
    func run() -> Int:
        err = err + 1
        return err + byteStep + wide

func main() -> Int
uses actors:
    let _peer: actor = core.spawn("Worker.run")
    return 0
`)
	prog, err := frontend.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}

	var run CheckedFunc
	found := false
	for _, fn := range checked.Funcs {
		if fn.Name == "Worker.run" {
			run = fn
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("missing checked function Worker.run")
	}
	if len(run.ActorState) != 3 {
		t.Fatalf("actor state count = %d, want 3", len(run.ActorState))
	}

	errField := run.ActorState["err"]
	if errField.Slot != 0 || !errField.Mutable || errField.Const || errField.TypeName != "task.error" || errField.Init != 0 {
		t.Fatalf("err field = %#v", errField)
	}
	byteStep := run.ActorState["byteStep"]
	if byteStep.Slot != 1 || byteStep.Mutable || byteStep.Const || byteStep.TypeName != "u8" || byteStep.Init != 7 {
		t.Fatalf("byteStep field = %#v", byteStep)
	}
	wide := run.ActorState["wide"]
	if wide.Slot != 2 || wide.Mutable || !wide.Const || wide.TypeName != "u16" || wide.Init != 9 {
		t.Fatalf("wide field = %#v", wide)
	}
}

func TestCheckActorStateRejectsPtrType(t *testing.T) {
	src := []byte(`
actor Worker:
    val raw: ptr = 0
    func run() -> Int:
        return 0

func main() -> Int
uses actors:
    let _peer: actor = core.spawn("Worker.run")
    return 0
`)
	prog, err := frontend.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected actor state type diagnostic")
	}
	if !strings.Contains(err.Error(), "type 'ptr' is not supported in this MVP") {
		t.Fatalf("error = %v", err)
	}
}
