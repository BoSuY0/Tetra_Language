package differential

import (
	"fmt"
	"runtime"
	"strings"
	"testing"

	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/opt"
)

func TestCheckScalarI32ComparesSourceStackRegisterAndOptimizedResults(t *testing.T) {
	fn := ir.IRFunc{
		Name:        "add_zero",
		ParamSlots:  1,
		LocalSlots:  1,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRReturn},
		},
	}

	report, err := CheckScalarI32(ScalarI32Case{
		Name:     "add-zero",
		Function: fn,
		Samples:  [][]int32{{-2}, {0}, {7}},
		Source: func(args []int32) (int32, bool) {
			return args[0], true
		},
		Optimizations: []opt.Pass{opt.BasicScalarPass()},
	})
	if err != nil {
		t.Fatalf("CheckScalarI32: %v", err)
	}
	if report.SchemaVersion != "tetra.differential.scalar_i32.v1" {
		t.Fatalf("schema = %q", report.SchemaVersion)
	}
	for _, want := range []Lane{LaneSourceInterpreter, LaneStackBackend, LaneRegisterBackend, LaneOptimizedBackend} {
		if !report.HasLane(want) {
			t.Fatalf("report lanes = %v, missing %s", report.Lanes, want)
		}
	}
	if len(report.Samples) != 3 {
		t.Fatalf("samples = %d, want 3", len(report.Samples))
	}
	for _, sample := range report.Samples {
		values := sample.ResultMap()
		for _, lane := range report.Lanes[1:] {
			if values[lane] != values[LaneSourceInterpreter] {
				t.Fatalf("sample %+v mismatch: %v", sample.Args, values)
			}
		}
	}
}

func TestCheckScalarI32RejectsLaneMismatch(t *testing.T) {
	fn := ir.IRFunc{
		Name:        "add",
		ParamSlots:  2,
		LocalSlots:  2,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRReturn},
		},
	}

	_, err := CheckScalarI32(ScalarI32Case{
		Name:     "bad-source-oracle",
		Function: fn,
		Samples:  [][]int32{{4, 3}},
		Source: func(args []int32) (int32, bool) {
			return args[0] - args[1], true
		},
	})
	if err == nil || !strings.Contains(err.Error(), "differential mismatch") {
		t.Fatalf("CheckScalarI32 error = %v, want differential mismatch", err)
	}
}

func TestCheckScalarI32SupportsCanonicalLoopSubset(t *testing.T) {
	report, err := CheckScalarI32(ScalarI32Case{
		Name:     "sum-n",
		Function: sumNStackIRFunc(),
		Samples:  [][]int32{{0}, {1}, {5}},
		Source: func(args []int32) (int32, bool) {
			n := args[0]
			var total int32
			for i := int32(0); i < n; i++ {
				total += i
			}
			return total, true
		},
	})
	if err != nil {
		t.Fatalf("CheckScalarI32 loop: %v", err)
	}
	if len(report.Samples) != 3 {
		t.Fatalf("samples = %d, want 3", len(report.Samples))
	}
}

func TestCheckBackendMatrixCoversCallLoopSliceAndNativeLanes(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 native execution lane only")
	}

	callReport, err := CheckBackendMatrix(BackendMatrixCase{
		Name:      "sum-call-loop",
		Functions: []ir.IRFunc{incStackIRFunc(), sumCallStackIRFunc()},
		Entry:     "sum_call",
		Samples: []MatrixSample{
			{Name: "zero", Args: []int32{0}},
			{Name: "five", Args: []int32{5}},
		},
		Source: func(sample MatrixSample) (int32, bool) {
			n := sample.Args[0]
			var total int32
			for i := int32(0); i < n; i++ {
				total += i + 1
			}
			return total, true
		},
		Optimizations: []opt.Pass{opt.BasicScalarPass()},
		Native: func(tc BackendMatrixCase, sample MatrixSample) (int32, error) {
			funcs := append([]ir.IRFunc{}, tc.Functions...)
			funcs = append(funcs, mainCallingFunction("main", tc.Entry, sample.Args))
			return EvalNativeLinuxX64Exit(funcs, "main", t.TempDir(), tc.Name+"-"+sample.Name)
		},
	})
	if err != nil {
		t.Fatalf("CheckBackendMatrix call-loop: %v", err)
	}
	for _, want := range []Lane{
		LaneSourceInterpreter,
		LaneStackIRInterpreter,
		LaneOptimizedStackIR,
		LaneSSAInterpreter,
		LaneMachineIRInterpreter,
		LaneNativeExecution,
	} {
		if !callReport.HasLane(want) {
			t.Fatalf("call-loop lanes = %+v, missing %s", callReport.Lanes, want)
		}
	}
	requireMatrixSamplesAgree(t, callReport)

	sliceReport, err := CheckBackendMatrix(BackendMatrixCase{
		Name:      "slice-sum",
		Functions: []ir.IRFunc{sumSliceStackIRFunc(true)},
		Entry:     "sum",
		Samples: []MatrixSample{{
			Name:      "four-elements",
			Args:      []int32{1, 4},
			I32Slices: map[int32][]int32{1: {1, 2, 3, 4}},
		}},
		Source: func(sample MatrixSample) (int32, bool) {
			xs := sample.I32Slices[sample.Args[0]]
			var total int32
			for i := int32(0); i < sample.Args[1]; i++ {
				total += xs[i]
			}
			return total, true
		},
		Native: func(tc BackendMatrixCase, sample MatrixSample) (int32, error) {
			funcs := append([]ir.IRFunc{}, tc.Functions...)
			funcs = append(funcs, mainCallingSliceSum("main", tc.Entry, sample.I32Slices[sample.Args[0]]))
			return EvalNativeLinuxX64Exit(funcs, "main", t.TempDir(), tc.Name+"-"+sample.Name)
		},
	})
	if err != nil {
		t.Fatalf("CheckBackendMatrix slice-sum: %v", err)
	}
	if sliceReport.StableSubset != "backend_differential_matrix_v1" {
		t.Fatalf("slice stable subset = %q", sliceReport.StableSubset)
	}
	requireMatrixSamplesAgree(t, sliceReport)
	if sliceReport.Matrix.Cases["slice_sum_i32"] != 1 || sliceReport.Matrix.Cases["call_loop_i32"] != 0 {
		t.Fatalf("slice matrix cases = %+v, want slice_sum_i32=1 and call_loop_i32=0", sliceReport.Matrix.Cases)
	}
}

func TestCheckBackendMatrixRecordsRandomizedSamplesAndMismatchReducer(t *testing.T) {
	fn := ir.IRFunc{
		Name:        "add",
		ParamSlots:  2,
		LocalSlots:  2,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRReturn},
		},
	}

	report, err := CheckBackendMatrix(BackendMatrixCase{
		Name:              "bad-add-oracle",
		Functions:         []ir.IRFunc{fn},
		Entry:             "add",
		Samples:           []MatrixSample{{Name: "fixed", Args: []int32{4, 3}}},
		RandomSeed:        16,
		RandomSampleCount: 3,
		Source: func(sample MatrixSample) (int32, bool) {
			return sample.Args[0] - sample.Args[1], true
		},
	})
	if err == nil || !strings.Contains(err.Error(), "differential mismatch") {
		t.Fatalf("CheckBackendMatrix error = %v, want differential mismatch", err)
	}
	if report.Randomized.Seed != 16 || report.Randomized.Generated != 3 {
		t.Fatalf("randomized metadata = %+v, want seed 16 and 3 generated samples", report.Randomized)
	}
	if report.Mismatch == nil || report.Mismatch.ReducerStatus != "reduced_to_single_sample" || report.Mismatch.SampleName != "fixed" {
		t.Fatalf("mismatch reducer = %+v, want reduced fixed sample", report.Mismatch)
	}
	if !strings.Contains(report.Mismatch.Reproducer, "bad-add-oracle") || !strings.Contains(report.Mismatch.Reproducer, "args=[4 3]") {
		t.Fatalf("mismatch reproducer = %q, want case and args", report.Mismatch.Reproducer)
	}
}

func sumNStackIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "sum_n",
		ParamSlots:  1,
		LocalSlots:  3,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRReturn},
		},
	}
}

func incStackIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "inc",
		ParamSlots:  1,
		LocalSlots:  1,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRReturn},
		},
	}
}

func sumCallStackIRFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "sum_call",
		ParamSlots:  1,
		LocalSlots:  3,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRCall, Name: "inc", ArgSlots: 1, RetSlots: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRReturn},
		},
	}
}

func sumSliceStackIRFunc(proof bool) ir.IRFunc {
	loadKind := ir.IRIndexLoadI32
	proofID := ""
	if proof {
		loadKind = ir.IRIndexLoadI32Unchecked
		proofID = "proof:while:i:xs:1:1"
	}
	return ir.IRFunc{
		Name:        "sum",
		ParamSlots:  2,
		LocalSlots:  4,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: loadKind, ProofID: proofID},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRReturn},
		},
	}
}

func mainCallingFunction(name string, callee string, args []int32) ir.IRFunc {
	instrs := make([]ir.IRInstr, 0, len(args)+2)
	for _, arg := range args {
		instrs = append(instrs, ir.IRInstr{Kind: ir.IRConstI32, Imm: arg})
	}
	instrs = append(instrs,
		ir.IRInstr{Kind: ir.IRCall, Name: callee, ArgSlots: len(args), RetSlots: 1},
		ir.IRInstr{Kind: ir.IRReturn},
	)
	return ir.IRFunc{Name: name, ReturnSlots: 1, Instrs: instrs}
}

func mainCallingSliceSum(name string, callee string, values []int32) ir.IRFunc {
	backingBase := 2
	instrs := []ir.IRInstr{
		{Kind: ir.IRConstI32, Imm: int32(len(values))},
		{Kind: ir.IRStackSliceI32, Local: backingBase, ArgSlots: len(values), Imm: int32(len(values)), Name: "matrix.xs"},
		{Kind: ir.IRStoreLocal, Local: 1},
		{Kind: ir.IRStoreLocal, Local: 0},
	}
	for i, value := range values {
		instrs = append(instrs,
			ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
			ir.IRInstr{Kind: ir.IRLoadLocal, Local: 1},
			ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(i)},
			ir.IRInstr{Kind: ir.IRConstI32, Imm: value},
			ir.IRInstr{Kind: ir.IRIndexStoreI32},
		)
	}
	instrs = append(instrs,
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 1},
		ir.IRInstr{Kind: ir.IRCall, Name: callee, ArgSlots: 2, RetSlots: 1},
		ir.IRInstr{Kind: ir.IRReturn},
	)
	return ir.IRFunc{Name: name, LocalSlots: backingBase + len(values), ReturnSlots: 1, Instrs: instrs}
}

func requireMatrixSamplesAgree(t *testing.T, report MatrixReport) {
	t.Helper()
	for _, sample := range report.Samples {
		values := sample.ResultMap()
		source := values[LaneSourceInterpreter]
		for _, lane := range report.Lanes {
			if values[lane] != source {
				t.Fatalf("%s sample %s lane %s = %d, source = %d; results=%+v", report.Case, sample.Name, lane, values[lane], source, sample.Results)
			}
		}
	}
}

func TestMainCallingSliceSumWrapperHasEnoughLocals(t *testing.T) {
	fn := mainCallingSliceSum("main", "sum", []int32{1, 2, 3})
	if fn.LocalSlots < 5 {
		t.Fatalf("wrapper locals = %d, want backing locals plus ptr/len", fn.LocalSlots)
	}
	if !strings.Contains(fmt.Sprint(fn.Instrs), "matrix.xs") {
		t.Fatalf("wrapper instructions = %+v, want named stack slice", fn.Instrs)
	}
}
