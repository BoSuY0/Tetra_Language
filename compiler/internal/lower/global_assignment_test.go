package lower

import (
	"testing"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/module"
	"tetra_language/compiler/internal/semantics"
)

func TestLowerGlobalStructFieldAssignmentStoresGlobalSlot(t *testing.T) {
	checked, mainFn := lowerGlobalAssignmentProgram(t, `
struct Box:
    first: Int
    value: Int

var box: Box

func main() -> Int:
    var first: Int = 11
    var second: Int = 22
    box.value = 42
    return first + second + box.value
`)

	box := checked.GlobalsByModule[""]["box"]
	valueSlot := box.DataIndex + 1
	if !hasConstStore(mainFn.Instrs, ir.IRStoreGlobal, valueSlot, 42) {
		t.Fatalf("global field assignment did not store 42 into global slot %d: %#v", valueSlot, mainFn.Instrs)
	}
	if hasConstStore(mainFn.Instrs, ir.IRStoreLocal, 1, 42) {
		t.Fatalf("global field assignment still stores 42 into local slot 1: %#v", mainFn.Instrs)
	}
}

func TestLowerGlobalStructFieldAssignmentWithoutLocalsVerifies(t *testing.T) {
	_, _ = lowerGlobalAssignmentProgram(t, `
struct Box:
    value: Int

var box: Box

func main() -> Int:
    box.value = 42
    return box.value
`)
}

func lowerGlobalAssignmentProgram(t *testing.T, src string) (*semantics.CheckedProgram, ir.IRFunc) {
	t.Helper()
	file, err := frontend.ParseFile([]byte(src), "global_assignment.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &module.World{
		EntryModule: "",
		Files:       []*frontend.FileAST{file},
		ByModule:    map[string]*frontend.FileAST{"": file},
	}
	checked, err := semantics.CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	irProg, err := Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	for _, fn := range irProg.Funcs {
		if fn.Name == "main" {
			return checked, fn
		}
	}
	t.Fatalf("main function not found")
	return nil, ir.IRFunc{}
}

func hasConstStore(instrs []ir.IRInstr, kind ir.IRInstrKind, slot int, value int32) bool {
	for i := 0; i+1 < len(instrs); i++ {
		if instrs[i].Kind == ir.IRConstI32 && instrs[i].Imm == value &&
			instrs[i+1].Kind == kind && instrs[i+1].Local == slot {
			return true
		}
	}
	return false
}
