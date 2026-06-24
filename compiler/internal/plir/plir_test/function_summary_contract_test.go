package plir_test

import (
	"testing"

	"tetra_language/compiler/internal/memoryfacts"
	"tetra_language/compiler/internal/memoryfacts/fromplir"
	. "tetra_language/compiler/internal/plir"
	"tetra_language/compiler/internal/semantics"
)

func TestFunctionSummaryDigestMatchesSemanticContract(t *testing.T) {
	checked := checkedProgram(t, `
func borrow_bytes(xs: borrow []u8) -> borrow []u8:
    return xs.borrow()

func main() -> Int:
    return 0
`)
	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	fn := findFunction(t, prog, "borrow_bytes")
	if fn.Summary == nil {
		t.Fatalf("borrow_bytes missing FunctionSummary")
	}
	sig := checked.FuncSigs["borrow_bytes"]
	wantDigest, err := semantics.FunctionContractDigest("borrow_bytes", sig)
	if err != nil {
		t.Fatalf("FunctionContractDigest: %v", err)
	}
	if got := fn.Summary.ContractSchema; got != semantics.FunctionContractSchemaV1 {
		t.Fatalf("FunctionSummary.ContractSchema = %q, want %q", got, semantics.FunctionContractSchemaV1)
	}
	if got := fn.Summary.ContractDigest; got != wantDigest {
		t.Fatalf("FunctionSummary.ContractDigest = %q, want %q", got, wantDigest)
	}
}

func TestFunctionSummaryContractDigestParityForSafetyFixtures(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		function string
		file     bool
	}{
		{
			name:     "scalar pure",
			function: "scalar_pure",
			source: `
func scalar_pure(x: Int) -> Int:
    return x + 1

func main() -> Int:
    return scalar_pure(41)
`,
		},
		{
			name:     "borrow return",
			function: "borrow_bytes",
			source: `
func borrow_bytes(xs: borrow []u8) -> borrow []u8:
    return xs.borrow()

func main() -> Int:
    return 0
`,
		},
		{
			name:     "resource return",
			function: "resource_return",
			source: `
func worker() -> Int:
    return 7

func resource_return(task: task.i32) -> task.i32:
    return task

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let other: task.i32 = resource_return(task)
    return core.task_join_i32(other)
`,
		},
		{
			name:     "typed error resource throw",
			function: "throw_task",
			source: `
enum TaskErr:
    case wrap(task.i32)

func worker() -> Int:
    return 7

func throw_task(task: task.i32) -> Int throws TaskErr:
    throw TaskErr.wrap(task)

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return catch throw_task(task):
    case TaskErr.wrap(other):
        core.task_join_i32(other)
`,
		},
		{
			name:     "callable return",
			function: "make_callback",
			source: `
func make_callback() -> fn(Int) -> Int:
    return fn(x: Int) -> Int:
        return x + 1

func main() -> Int:
    let cb: fn(Int) -> Int = make_callback()
    return cb(41)
`,
		},
		{
			name:     "async function",
			function: "async_answer",
			source: `
async func async_answer() -> Int:
    return 42

async func async_caller() -> Int:
    let value: Int = await async_answer()
    return value

func main() -> Int:
    return 0
`,
		},
		{
			name:     "effectful function",
			function: "effectful_alloc",
			source: `
func effectful_alloc() -> []u8
uses alloc, mem:
    return make_u8(1)

func main() -> Int
uses alloc, mem:
    var xs: []u8 = effectful_alloc()
    return xs.len
`,
		},
		{
			name:     "mutable global touching worker",
			function: "mutable_worker",
			file:     true,
			source: `
var counter: Int = 0

func mutable_worker() -> Int
uses mem:
    counter = counter + 1
    return counter

func main() -> Int
uses mem:
    return mutable_worker()
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var checked *semantics.CheckedProgram
			if tt.file {
				checked = checkedFileProgram(t, tt.source)
			} else {
				checked = checkedProgram(t, tt.source)
			}
			assertFunctionContractDigestParity(t, checked, tt.function)
		})
	}
}

func TestFunctionSummaryUnknownInterfaceContractDigestParity(t *testing.T) {
	sig := semantics.FuncSig{
		Public:              true,
		ParamNames:          []string{"task"},
		ParamTypes:          []string{"task.i32"},
		ParamOwnership:      []string{""},
		ParamSlots:          2,
		ReturnType:          "task.i32",
		ReturnSlots:         2,
		ReturnResourceParam: semantics.SummaryParamUnknown,
	}
	summary, err := FunctionSummaryFromFuncSig("lib.api.unknown_task", sig)
	if err != nil {
		t.Fatalf("FunctionSummaryFromFuncSig: %v", err)
	}
	wantDigest, err := semantics.FunctionContractDigest("lib.api.unknown_task", sig)
	if err != nil {
		t.Fatalf("FunctionContractDigest: %v", err)
	}
	if summary.ContractDigest != wantDigest {
		t.Fatalf("FunctionSummary.ContractDigest = %q, want %q", summary.ContractDigest, wantDigest)
	}
	if !summary.ReturnResourceUnknown {
		t.Fatalf("FunctionSummary.ReturnResourceUnknown = false, want true")
	}

	prog := &Program{Funcs: []Function{{
		Name:    "lib.api.unknown_task",
		Module:  "lib.api",
		Summary: summary,
	}}}
	graph, err := fromplir.Build("unknown-interface-contract", prog)
	if err != nil {
		t.Fatalf("fromplir.Build: %v", err)
	}
	fact := findFunctionContractFact(t, graph.Facts(), "lib.api.unknown_task")
	if fact.ContractDigest != wantDigest {
		t.Fatalf("MemoryFacts ContractDigest = %q, want %q", fact.ContractDigest, wantDigest)
	}
	if fact.ContractSchema != semantics.FunctionContractSchemaV1 {
		t.Fatalf("MemoryFacts ContractSchema = %q, want %q", fact.ContractSchema, semantics.FunctionContractSchemaV1)
	}
	for _, fact := range graph.Facts() {
		if fact.FunctionID != "lib.api.unknown_task" {
			continue
		}
		if fact.Claim == "no_escape" || fact.Claim == "no_alias" {
			t.Fatalf("unknown interface contract produced unsafe precise fact: %#v", fact)
		}
	}
}

func assertFunctionContractDigestParity(
	t *testing.T,
	checked *semantics.CheckedProgram,
	functionName string,
) {
	t.Helper()
	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	fn := findFunction(t, prog, functionName)
	if fn.Summary == nil {
		t.Fatalf("%s missing FunctionSummary", functionName)
	}
	sig, ok := checked.FuncSigs[functionName]
	if !ok {
		t.Fatalf("missing source FuncSig for %s", functionName)
	}
	wantDigest, err := semantics.FunctionContractDigest(functionName, sig)
	if err != nil {
		t.Fatalf("FunctionContractDigest(%s): %v", functionName, err)
	}
	if got := fn.Summary.ContractSchema; got != semantics.FunctionContractSchemaV1 {
		t.Fatalf("%s FunctionSummary.ContractSchema = %q, want %q", functionName, got, semantics.FunctionContractSchemaV1)
	}
	if got := fn.Summary.ContractDigest; got != wantDigest {
		t.Fatalf("%s FunctionSummary.ContractDigest = %q, want %q", functionName, got, wantDigest)
	}
	graph, err := fromplir.Build("contract-parity", prog)
	if err != nil {
		t.Fatalf("fromplir.Build: %v", err)
	}
	fact := findFunctionContractFact(t, graph.Facts(), functionName)
	if got := fact.ContractSchema; got != semantics.FunctionContractSchemaV1 {
		t.Fatalf("%s MemoryFacts ContractSchema = %q, want %q", functionName, got, semantics.FunctionContractSchemaV1)
	}
	if got := fact.ContractDigest; got != wantDigest {
		t.Fatalf("%s MemoryFacts ContractDigest = %q, want %q", functionName, got, wantDigest)
	}
}

func findFunctionContractFact(
	t *testing.T,
	facts []memoryfacts.Fact,
	functionName string,
) memoryfacts.Fact {
	t.Helper()
	for _, fact := range facts {
		if fact.FunctionID == functionName && fact.Claim == memoryfacts.ClaimFunctionContract {
			return fact
		}
	}
	t.Fatalf("missing function_contract fact for %s: %#v", functionName, facts)
	return memoryfacts.Fact{}
}

func TestFunctionSummaryFromFuncSigRejectsInvalidContract(t *testing.T) {
	_, err := FunctionSummaryFromFuncSig("bad", semantics.FuncSig{
		ParamTypes:        []string{"i32"},
		ReturnType:        "i32",
		ReturnRegionParam: 2,
	})
	if err == nil {
		t.Fatalf("FunctionSummaryFromFuncSig accepted invalid summary param")
	}
}
