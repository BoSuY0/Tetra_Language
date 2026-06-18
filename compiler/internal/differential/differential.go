package differential

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"tetra_language/compiler/internal/backend/linux_x64"
	"tetra_language/compiler/internal/format/elf"
	"tetra_language/compiler/internal/format/tobj"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/linker"
	"tetra_language/compiler/internal/machine"
	"tetra_language/compiler/internal/opt"
	"tetra_language/compiler/internal/ssair"
)

type Lane string

const (
	LaneSourceInterpreter    Lane = "source_interpreter"
	LaneStackBackend         Lane = "stack_backend"
	LaneRegisterBackend      Lane = "register_backend"
	LaneOptimizedBackend     Lane = "optimized_backend"
	LaneStackIRInterpreter   Lane = "stack_ir_interpreter"
	LaneOptimizedStackIR     Lane = "optimized_stack_ir"
	LaneSSAInterpreter       Lane = "ssa_interpreter"
	LaneMachineIRInterpreter Lane = "machine_ir_interpreter"
	LaneNativeExecution      Lane = "native_execution"
)

type ScalarI32Case struct {
	Name          string
	Function      ir.IRFunc
	Samples       [][]int32
	Source        func(args []int32) (int32, bool)
	Optimizations []opt.Pass
}

type BackendMatrixCase struct {
	Name              string
	Functions         []ir.IRFunc
	Entry             string
	Samples           []MatrixSample
	Source            func(MatrixSample) (int32, bool)
	Optimizations     []opt.Pass
	Native            func(BackendMatrixCase, MatrixSample) (int32, error)
	RandomSeed        int64
	RandomSampleCount int
}

type MatrixSample struct {
	Name      string            `json:"name,omitempty"`
	Args      []int32           `json:"args"`
	I32Slices map[int32][]int32 `json:"i32_slices,omitempty"`
}

type Report struct {
	SchemaVersion string         `json:"schema_version"`
	Case          string         `json:"case"`
	StableSubset  string         `json:"stable_subset"`
	Lanes         []Lane         `json:"lanes"`
	Samples       []SampleReport `json:"samples"`
}

type MatrixReport struct {
	SchemaVersion string               `json:"schema_version"`
	Case          string               `json:"case"`
	StableSubset  string               `json:"stable_subset"`
	Matrix        MatrixSummary        `json:"matrix"`
	Lanes         []Lane               `json:"lanes"`
	Randomized    RandomizedSummary    `json:"randomized,omitempty"`
	Samples       []MatrixSampleReport `json:"samples"`
	Mismatch      *MismatchReport      `json:"mismatch,omitempty"`
	Unsupported   []UnsupportedRow     `json:"unsupported,omitempty"`
}

type MatrixSummary struct {
	Lanes map[Lane]string `json:"lanes"`
	Cases map[string]int  `json:"cases"`
}

type RandomizedSummary struct {
	Seed      int64  `json:"seed,omitempty"`
	Generated int    `json:"generated,omitempty"`
	Bounds    string `json:"bounds,omitempty"`
}

type MatrixSampleReport struct {
	Name    string       `json:"name,omitempty"`
	Args    []int32      `json:"args"`
	Results []LaneResult `json:"results"`
}

type MismatchReport struct {
	Case          string  `json:"case"`
	SampleName    string  `json:"sample_name,omitempty"`
	Args          []int32 `json:"args"`
	Lane          Lane    `json:"lane"`
	ExpectedLane  Lane    `json:"expected_lane"`
	Expected      int32   `json:"expected"`
	Got           int32   `json:"got"`
	ReducerStatus string  `json:"reducer_status"`
	Reproducer    string  `json:"reproducer"`
}

type UnsupportedRow struct {
	Function string `json:"function"`
	Reason   string `json:"reason"`
}

type SampleReport struct {
	Args    []int32      `json:"args"`
	Results []LaneResult `json:"results"`
}

type LaneResult struct {
	Lane  Lane  `json:"lane"`
	Value int32 `json:"value"`
}

func (r Report) HasLane(want Lane) bool {
	for _, lane := range r.Lanes {
		if lane == want {
			return true
		}
	}
	return false
}

func (r MatrixReport) HasLane(want Lane) bool {
	for _, lane := range r.Lanes {
		if lane == want {
			return true
		}
	}
	return false
}

func (s SampleReport) ResultMap() map[Lane]int32 {
	out := make(map[Lane]int32, len(s.Results))
	for _, result := range s.Results {
		out[result.Lane] = result.Value
	}
	return out
}

func (s MatrixSampleReport) ResultMap() map[Lane]int32 {
	out := make(map[Lane]int32, len(s.Results))
	for _, result := range s.Results {
		out[result.Lane] = result.Value
	}
	return out
}

func CheckScalarI32(tc ScalarI32Case) (Report, error) {
	if tc.Name == "" {
		return Report{}, fmt.Errorf("differential scalar-i32: missing case name")
	}
	if tc.Source == nil {
		return Report{}, fmt.Errorf("differential scalar-i32: missing source interpreter")
	}
	samples := tc.Samples
	if len(samples) == 0 {
		samples = defaultSamples(tc.Function.ParamSlots)
	}
	mfn, ok, err := machineScalarFunction(tc.Function)
	if err != nil {
		return Report{}, err
	}
	if !ok {
		return Report{}, fmt.Errorf(
			"differential scalar-i32: %s is outside register backend stable subset",
			tc.Function.Name,
		)
	}
	optFn := tc.Function
	if len(tc.Optimizations) > 0 {
		prog := &ir.IRProgram{
			MainIndex: 0,
			MainName:  tc.Function.Name,
			Funcs:     []ir.IRFunc{cloneIRFunc(tc.Function)},
		}
		if _, err := opt.NewManager().Run(prog, tc.Optimizations...); err != nil {
			return Report{}, fmt.Errorf("differential scalar-i32: optimize %s: %w", tc.Name, err)
		}
		optFn = prog.Funcs[0]
	}
	report := Report{
		SchemaVersion: "tetra.differential.scalar_i32.v1",
		Case:          tc.Name,
		StableSubset:  "scalar_i32_stack_ir_and_machine_ir_v1",
		Lanes: []Lane{
			LaneSourceInterpreter,
			LaneStackBackend,
			LaneRegisterBackend,
			LaneOptimizedBackend,
		},
		Samples: make([]SampleReport, 0, len(samples)),
	}
	for _, args := range samples {
		source, ok := tc.Source(append([]int32(nil), args...))
		if !ok {
			return report, fmt.Errorf(
				"differential scalar-i32: source interpreter rejected args=%v",
				args,
			)
		}
		stackValue, err := EvalStackI32(tc.Function, args)
		if err != nil {
			return report, fmt.Errorf(
				"differential scalar-i32: stack backend %s args=%v: %w",
				tc.Function.Name,
				args,
				err,
			)
		}
		registerValue, err := EvalMachineI32(mfn, args)
		if err != nil {
			return report, fmt.Errorf(
				"differential scalar-i32: register backend %s args=%v: %w",
				tc.Function.Name,
				args,
				err,
			)
		}
		optimizedValue, err := EvalStackI32(optFn, args)
		if err != nil {
			return report, fmt.Errorf(
				"differential scalar-i32: optimized backend %s args=%v: %w",
				optFn.Name,
				args,
				err,
			)
		}
		results := []LaneResult{
			{Lane: LaneSourceInterpreter, Value: source},
			{Lane: LaneStackBackend, Value: stackValue},
			{Lane: LaneRegisterBackend, Value: registerValue},
			{Lane: LaneOptimizedBackend, Value: optimizedValue},
		}
		for _, result := range results[1:] {
			if result.Value != source {
				return report, fmt.Errorf(
					"differential mismatch for %s args=%v lane=%s source=%d got=%d",
					tc.Name,
					args,
					result.Lane,
					source,
					result.Value,
				)
			}
		}
		report.Samples = append(
			report.Samples,
			SampleReport{Args: append([]int32(nil), args...), Results: results},
		)
	}
	return report, nil
}

func CheckBackendMatrix(tc BackendMatrixCase) (MatrixReport, error) {
	if tc.Name == "" {
		return MatrixReport{}, fmt.Errorf("backend differential matrix: missing case name")
	}
	if tc.Entry == "" {
		return MatrixReport{}, fmt.Errorf("backend differential matrix: missing entry function")
	}
	if tc.Source == nil {
		return MatrixReport{}, fmt.Errorf("backend differential matrix: missing source interpreter")
	}
	if len(tc.Functions) == 0 {
		return MatrixReport{}, fmt.Errorf("backend differential matrix: missing functions")
	}
	funcs := cloneIRFuncs(tc.Functions)
	if _, ok := irFuncByName(funcs, tc.Entry); !ok {
		return MatrixReport{}, fmt.Errorf(
			"backend differential matrix: entry %q not found",
			tc.Entry,
		)
	}
	samples := normalizeMatrixSamples(tc)
	report := MatrixReport{
		SchemaVersion: "tetra.differential.backend_matrix.v1",
		Case:          tc.Name,
		StableSubset:  "backend_differential_matrix_v1",
		Matrix: MatrixSummary{
			Lanes: map[Lane]string{
				LaneSourceInterpreter:    "required",
				LaneStackIRInterpreter:   "required",
				LaneOptimizedStackIR:     "required",
				LaneSSAInterpreter:       "required",
				LaneMachineIRInterpreter: "required",
				LaneNativeExecution:      "not_configured",
			},
			Cases: classifyMatrixCases(funcs),
		},
		Lanes: []Lane{
			LaneSourceInterpreter,
			LaneStackIRInterpreter,
			LaneOptimizedStackIR,
			LaneSSAInterpreter,
			LaneMachineIRInterpreter,
		},
		Samples: make([]MatrixSampleReport, 0, len(samples)),
	}
	if tc.Native != nil {
		report.Matrix.Lanes[LaneNativeExecution] = "required"
		report.Lanes = append(report.Lanes, LaneNativeExecution)
	}
	if tc.RandomSampleCount > 0 {
		report.Randomized = RandomizedSummary{
			Seed:      tc.RandomSeed,
			Generated: tc.RandomSampleCount,
			Bounds:    "i32 args in [-8,8], deterministic seed",
		}
	}
	optFuncs, err := optimizedMatrixFunctions(funcs, tc.Optimizations)
	if err != nil {
		return report, err
	}
	ssaFuncs, unsupported, err := ssaMatrixFunctions(funcs)
	if err != nil {
		return report, err
	}
	report.Unsupported = append(report.Unsupported, unsupported...)
	machineFuncs, unsupported, err := machineMatrixFunctions(funcs)
	if err != nil {
		return report, err
	}
	report.Unsupported = append(report.Unsupported, unsupported...)
	if len(report.Unsupported) > 0 {
		return report, fmt.Errorf(
			"backend differential matrix: unsupported functions: %s",
			formatUnsupportedRows(report.Unsupported),
		)
	}
	for _, sample := range samples {
		source, ok := tc.Source(cloneMatrixSample(sample))
		if !ok {
			return report, fmt.Errorf(
				"backend differential matrix: source interpreter rejected sample %s args=%v",
				sampleName(sample),
				sample.Args,
			)
		}
		results := []LaneResult{{Lane: LaneSourceInterpreter, Value: source}}
		laneValues := []struct {
			lane Lane
			run  func() (int32, error)
		}{
			{
				lane: LaneStackIRInterpreter,
				run: func() (int32, error) {
					return EvalStackProgramI32(funcs, tc.Entry, sample.Args, sample.I32Slices)
				},
			},
			{
				lane: LaneOptimizedStackIR,
				run: func() (int32, error) {
					return EvalStackProgramI32(optFuncs, tc.Entry, sample.Args, sample.I32Slices)
				},
			},
			{
				lane: LaneSSAInterpreter,
				run: func() (int32, error) {
					return EvalSSAProgramI32(ssaFuncs, tc.Entry, sample.Args, sample.I32Slices)
				},
			},
			{
				lane: LaneMachineIRInterpreter,
				run: func() (int32, error) {
					return EvalMachineProgramI32(
						machineFuncs,
						tc.Entry,
						sample.Args,
						sample.I32Slices,
					)
				},
			},
		}
		if tc.Native != nil {
			laneValues = append(laneValues, struct {
				lane Lane
				run  func() (int32, error)
			}{
				lane: LaneNativeExecution,
				run: func() (int32, error) {
					return tc.Native(tc, sample)
				},
			})
		}
		for _, lane := range laneValues {
			value, err := lane.run()
			if err != nil {
				return report, fmt.Errorf(
					"backend differential matrix: %s sample %s args=%v lane=%s: %w",
					tc.Name,
					sampleName(sample),
					sample.Args,
					lane.lane,
					err,
				)
			}
			results = append(results, LaneResult{Lane: lane.lane, Value: value})
			if value != source {
				report.Mismatch = matrixMismatch(
					tc.Name,
					sample,
					lane.lane,
					source,
					value,
					len(samples),
				)
				report.Samples = append(
					report.Samples,
					MatrixSampleReport{
						Name:    sample.Name,
						Args:    append([]int32(nil), sample.Args...),
						Results: results,
					},
				)
				return report, fmt.Errorf(
					"differential mismatch for %s sample=%s args=%v lane=%s source=%d got=%d",
					tc.Name,
					sampleName(sample),
					sample.Args,
					lane.lane,
					source,
					value,
				)
			}
		}
		report.Samples = append(
			report.Samples,
			MatrixSampleReport{
				Name:    sample.Name,
				Args:    append([]int32(nil), sample.Args...),
				Results: results,
			},
		)
	}
	return report, nil
}

func EvalStackProgramI32(
	funcs []ir.IRFunc,
	entry string,
	args []int32,
	memory map[int32][]int32,
) (int32, error) {
	funcsByName := map[string]ir.IRFunc{}
	for _, fn := range funcs {
		funcsByName[fn.Name] = fn
	}
	if _, ok := funcsByName[entry]; !ok {
		return 0, fmt.Errorf("stack ir interpreter: entry %q not found", entry)
	}
	return evalStackFunctionI32(funcsByName, entry, args, cloneI32Slices(memory), 0)
}

func EvalStackI32(fn ir.IRFunc, args []int32) (int32, error) {
	return EvalStackProgramI32([]ir.IRFunc{fn}, fn.Name, args, nil)
}

func evalStackFunctionI32(
	funcs map[string]ir.IRFunc,
	entry string,
	args []int32,
	memory map[int32][]int32,
	depth int,
) (int32, error) {
	if depth > 64 {
		return 0, fmt.Errorf("%s exceeded call depth limit", entry)
	}
	fn, ok := funcs[entry]
	if !ok {
		return 0, fmt.Errorf("callee %q not found", entry)
	}
	if fn.ParamSlots != len(args) {
		return 0, fmt.Errorf("%s param count %d, want %d", fn.Name, len(args), fn.ParamSlots)
	}
	if fn.ReturnSlots != 1 {
		return 0, fmt.Errorf("%s return slots %d, want 1", fn.Name, fn.ReturnSlots)
	}
	if fn.LocalSlots < fn.ParamSlots {
		return 0, fmt.Errorf(
			"%s local slots %d smaller than params %d",
			fn.Name,
			fn.LocalSlots,
			fn.ParamSlots,
		)
	}
	labels := map[int]int{}
	for i, instr := range fn.Instrs {
		if instr.Kind == ir.IRLabel {
			labels[instr.Label] = i
		}
	}
	locals := make([]int32, fn.LocalSlots)
	copy(locals, args)
	stack := []int32{}
	pop := func(kind ir.IRInstrKind) (int32, error) {
		if len(stack) == 0 {
			return 0, fmt.Errorf("%s stack underflow at %d", fn.Name, kind)
		}
		value := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		return value, nil
	}
	pop2 := func(kind ir.IRInstrKind) (int32, int32, error) {
		right, err := pop(kind)
		if err != nil {
			return 0, 0, err
		}
		left, err := pop(kind)
		if err != nil {
			return 0, 0, err
		}
		return left, right, nil
	}
	for pc, steps := 0, 0; pc < len(fn.Instrs); steps++ {
		if steps > 100000 {
			return 0, fmt.Errorf("%s exceeded interpreter step limit", fn.Name)
		}
		instr := fn.Instrs[pc]
		pc++
		switch instr.Kind {
		case ir.IRConstI32:
			stack = append(stack, instr.Imm)
		case ir.IRLoadLocal:
			if instr.Local < 0 || instr.Local >= len(locals) {
				return 0, fmt.Errorf("%s load local %d out of bounds", fn.Name, instr.Local)
			}
			stack = append(stack, locals[instr.Local])
		case ir.IRStoreLocal:
			if instr.Local < 0 || instr.Local >= len(locals) {
				return 0, fmt.Errorf("%s store local %d out of bounds", fn.Name, instr.Local)
			}
			value, err := pop(instr.Kind)
			if err != nil {
				return 0, err
			}
			locals[instr.Local] = value
		case ir.IRAddI32, ir.IRSubI32, ir.IRMulI32, ir.IRDivI32, ir.IRModI32,
			ir.IRCmpEqI32, ir.IRCmpLtI32, ir.IRCmpGtI32, ir.IRCmpGeI32,
			ir.IRCmpLeI32, ir.IRCmpNeI32:
			left, right, err := pop2(instr.Kind)
			if err != nil {
				return 0, err
			}
			value, err := evalStackBinary(instr.Kind, left, right)
			if err != nil {
				return 0, err
			}
			stack = append(stack, value)
		case ir.IRNegI32:
			value, err := pop(instr.Kind)
			if err != nil {
				return 0, err
			}
			stack = append(stack, -value)
		case ir.IRCall:
			if instr.Name == "" || instr.ArgSlots < 0 || instr.RetSlots != 1 {
				return 0, fmt.Errorf(
					"%s call %q has unsupported ABI args=%d rets=%d",
					fn.Name,
					instr.Name,
					instr.ArgSlots,
					instr.RetSlots,
				)
			}
			callArgs := make([]int32, instr.ArgSlots)
			for i := instr.ArgSlots - 1; i >= 0; i-- {
				arg, err := pop(instr.Kind)
				if err != nil {
					return 0, err
				}
				callArgs[i] = arg
			}
			value, err := evalStackFunctionI32(funcs, instr.Name, callArgs, memory, depth+1)
			if err != nil {
				return 0, err
			}
			stack = append(stack, value)
		case ir.IRIndexLoadI32, ir.IRIndexLoadI32Unchecked:
			index, length, base, err := popSliceAccess(fn.Name, instr.Kind, pop)
			if err != nil {
				return 0, err
			}
			value, err := loadI32Slice(memory, base, length, index, instr.Kind == ir.IRIndexLoadI32)
			if err != nil {
				return 0, fmt.Errorf("%s index load: %w", fn.Name, err)
			}
			stack = append(stack, value)
		case ir.IRIndexStoreI32:
			value, err := pop(instr.Kind)
			if err != nil {
				return 0, err
			}
			index, length, base, err := popSliceAccess(fn.Name, instr.Kind, pop)
			if err != nil {
				return 0, err
			}
			if err := storeI32Slice(memory, base, length, index, value); err != nil {
				return 0, fmt.Errorf("%s index store: %w", fn.Name, err)
			}
		case ir.IRLabel:
		case ir.IRJmp:
			next, ok := labels[instr.Label]
			if !ok {
				return 0, fmt.Errorf("%s unknown label %d", fn.Name, instr.Label)
			}
			pc = next
		case ir.IRJmpIfZero:
			value, err := pop(instr.Kind)
			if err != nil {
				return 0, err
			}
			if value == 0 {
				next, ok := labels[instr.Label]
				if !ok {
					return 0, fmt.Errorf("%s unknown label %d", fn.Name, instr.Label)
				}
				pc = next
			}
		case ir.IRReturn:
			value, err := pop(instr.Kind)
			if err != nil {
				return 0, err
			}
			if len(stack) != 0 {
				return 0, fmt.Errorf("%s return leaves %d stack values", fn.Name, len(stack))
			}
			return value, nil
		default:
			return 0, fmt.Errorf(
				"%s instruction %d is outside stable scalar-i32 subset",
				fn.Name,
				instr.Kind,
			)
		}
	}
	return 0, fmt.Errorf("%s fell off end without return", fn.Name)
}

func EvalMachineI32(fn machine.Function, args []int32) (int32, error) {
	return EvalMachineProgramI32(map[string]machine.Function{fn.Name: fn}, fn.Name, args, nil)
}

func EvalMachineProgramI32(
	funcs map[string]machine.Function,
	entry string,
	args []int32,
	memory map[int32][]int32,
) (int32, error) {
	if _, ok := funcs[entry]; !ok {
		return 0, fmt.Errorf("machine ir interpreter: entry %q not found", entry)
	}
	return evalMachineFunctionI32(funcs, entry, args, cloneI32Slices(memory), 0)
}

func evalMachineFunctionI32(
	funcs map[string]machine.Function,
	entry string,
	args []int32,
	memory map[int32][]int32,
	depth int,
) (int32, error) {
	if depth > 64 {
		return 0, fmt.Errorf("%s exceeded machine call depth limit", entry)
	}
	fn := funcs[entry]
	if err := machine.VerifyFunction(fn); err != nil {
		return 0, err
	}
	if len(args) != len(fn.Params) {
		return 0, fmt.Errorf("%s param count %d, want %d", fn.Name, len(args), len(fn.Params))
	}
	blocks := map[string]machine.Block{}
	for _, block := range fn.Blocks {
		blocks[block.Name] = block
	}
	current := fn.Blocks[0].Name
	regs := map[machine.VReg]int32{}
	for i, param := range fn.Params {
		regs[param] = args[i]
	}
	read := func(reg machine.VReg) (int32, error) {
		value, ok := regs[reg]
		if !ok {
			return 0, fmt.Errorf("%s reads undefined vreg %s", fn.Name, reg)
		}
		return value, nil
	}
	write := func(reg machine.VReg, value int32) {
		regs[reg] = value
	}
	for steps := 0; steps <= 100000; steps++ {
		block, ok := blocks[current]
		if !ok {
			return 0, fmt.Errorf("%s unknown block %s", fn.Name, current)
		}
		nextBlock := ""
	instrLoop:
		for _, instr := range block.Instrs {
			switch instr.Op {
			case machine.OpMov:
				value := int32(instr.Imm)
				if len(instr.Uses) == 1 {
					var err error
					value, err = read(instr.Uses[0])
					if err != nil {
						return 0, err
					}
				}
				write(instr.Defs[0], value)
			case machine.OpAdd, machine.OpSub, machine.OpMul, machine.OpDiv, machine.OpMod, machine.OpCmp:
				left, err := read(instr.Uses[0])
				if err != nil {
					return 0, err
				}
				right, err := read(instr.Uses[1])
				if err != nil {
					return 0, err
				}
				value, err := evalMachineBinary(instr, left, right)
				if err != nil {
					return 0, err
				}
				write(instr.Defs[0], value)
			case machine.OpIndexLoad:
				base, err := read(instr.Uses[0])
				if err != nil {
					return 0, err
				}
				length, err := read(instr.Uses[1])
				if err != nil {
					return 0, err
				}
				index, err := read(instr.Uses[2])
				if err != nil {
					return 0, err
				}
				value, err := loadI32Slice(memory, base, length, index, false)
				if err != nil {
					return 0, fmt.Errorf("%s machine index load: %w", fn.Name, err)
				}
				write(instr.Defs[0], value)
			case machine.OpCall:
				if instr.Call == "" || len(instr.Defs) > 1 {
					return 0, fmt.Errorf("%s unsupported machine call shape", fn.Name)
				}
				callArgs := make([]int32, len(instr.Uses))
				for i, arg := range instr.Uses {
					value, err := read(arg)
					if err != nil {
						return 0, err
					}
					callArgs[i] = value
				}
				value, err := evalMachineFunctionI32(funcs, instr.Call, callArgs, memory, depth+1)
				if err != nil {
					return 0, err
				}
				if len(instr.Defs) == 1 {
					write(instr.Defs[0], value)
				}
			case machine.OpInc:
				value, err := read(instr.Uses[0])
				if err != nil {
					return 0, err
				}
				write(instr.Defs[0], value+1)
			case machine.OpBranchIf:
				value, err := read(instr.Uses[0])
				if err != nil {
					return 0, err
				}
				if strings.Contains(instr.Note, "if_zero") {
					if value == 0 {
						nextBlock = instr.Target
						break instrLoop
					}
					continue
				}
				if value != 0 {
					nextBlock = instr.Target
					break instrLoop
				}
			case machine.OpBranch:
				nextBlock = instr.Target
				break instrLoop
			case machine.OpReturn:
				if len(instr.Uses) == 0 {
					return 0, nil
				}
				return read(instr.Uses[0])
			default:
				return 0, fmt.Errorf(
					"%s opcode %s is outside stable scalar-i32 machine subset",
					fn.Name,
					instr.Op,
				)
			}
		}
		if nextBlock == "" {
			return 0, fmt.Errorf("%s block %s did not transfer control", fn.Name, current)
		}
		current = nextBlock
	}
	return 0, fmt.Errorf("%s exceeded machine interpreter step limit", fn.Name)
}

func EvalSSAProgramI32(
	funcs map[string]ssair.Function,
	entry string,
	args []int32,
	memory map[int32][]int32,
) (int32, error) {
	if _, ok := funcs[entry]; !ok {
		return 0, fmt.Errorf("ssa interpreter: entry %q not found", entry)
	}
	return evalSSAFunctionI32(funcs, entry, args, cloneI32Slices(memory), 0)
}

func evalSSAFunctionI32(
	funcs map[string]ssair.Function,
	entry string,
	args []int32,
	memory map[int32][]int32,
	depth int,
) (int32, error) {
	if depth > 64 {
		return 0, fmt.Errorf("%s exceeded ssa call depth limit", entry)
	}
	fn := funcs[entry]
	if err := ssair.VerifyFunction(fn); err != nil {
		return 0, err
	}
	blocks := map[string]ssair.Block{}
	entryBlock := ""
	for _, block := range fn.Blocks {
		blocks[block.ID] = block
		if block.Entry {
			entryBlock = block.ID
		}
	}
	values := map[ssair.ValueID]int32{}
	paramIndex := 0
	for _, value := range fn.Values {
		switch {
		case value.Type == ssair.TypeEffect:
			values[value.ID] = 0
		case value.Origin == "param":
			if paramIndex >= len(args) {
				return 0, fmt.Errorf(
					"%s param count %d, want at least %d",
					fn.Name,
					len(args),
					paramIndex+1,
				)
			}
			values[value.ID] = args[paramIndex]
			paramIndex++
		case value.ID == "zero":
			values[value.ID] = 0
		case value.ID == "one":
			values[value.ID] = 1
		}
	}
	if paramIndex != len(args) {
		return 0, fmt.Errorf("%s param count %d, want %d", fn.Name, len(args), paramIndex)
	}
	current := entryBlock
	branchArgs := []int32{}
	for steps := 0; steps <= 100000; steps++ {
		block, ok := blocks[current]
		if !ok {
			return 0, fmt.Errorf("%s unknown ssa block %s", fn.Name, current)
		}
		if len(branchArgs) != len(block.Params) {
			return 0, fmt.Errorf(
				"%s block %s arg count %d, want %d",
				fn.Name,
				block.ID,
				len(branchArgs),
				len(block.Params),
			)
		}
		for i, param := range block.Params {
			values[param] = branchArgs[i]
		}
		for _, instr := range block.Instrs {
			value, hasValue, err := evalSSAInstr(funcs, instr, values, memory, depth)
			if err != nil {
				return 0, err
			}
			if hasValue && instr.Result != "" {
				values[instr.Result] = value
			}
		}
		switch block.Term.Kind {
		case ssair.TermReturn:
			if block.Term.Value == "" {
				return 0, nil
			}
			return readSSAValue(fn.Name, values, block.Term.Value)
		case ssair.TermBranch:
			args, err := readSSAArgs(fn.Name, values, block.Term.Args)
			if err != nil {
				return 0, err
			}
			current = block.Term.Target
			branchArgs = args
		case ssair.TermCondBr:
			cond, err := readSSAValue(fn.Name, values, block.Term.Cond)
			if err != nil {
				return 0, err
			}
			if cond != 0 {
				args, err := readSSAArgs(fn.Name, values, block.Term.IfTrueArgs)
				if err != nil {
					return 0, err
				}
				current = block.Term.IfTrue
				branchArgs = args
			} else {
				args, err := readSSAArgs(fn.Name, values, block.Term.IfFalseArgs)
				if err != nil {
					return 0, err
				}
				current = block.Term.IfFalse
				branchArgs = args
			}
		default:
			return 0, fmt.Errorf(
				"%s block %s has unsupported ssa terminator %s",
				fn.Name,
				block.ID,
				block.Term.Kind,
			)
		}
	}
	return 0, fmt.Errorf("%s exceeded ssa interpreter step limit", fn.Name)
}

func evalSSAInstr(
	funcs map[string]ssair.Function,
	instr ssair.Instr,
	values map[ssair.ValueID]int32,
	memory map[int32][]int32,
	depth int,
) (int32, bool, error) {
	switch instr.Kind {
	case ssair.OpConstI32:
		return instr.Imm, true, nil
	case ssair.OpAddI32, ssair.OpSubI32, ssair.OpMulI32, ssair.OpDivI32, ssair.OpModI32,
		ssair.OpCmpEqI32, ssair.OpCmpLtI32, ssair.OpCmpGtI32, ssair.OpCmpGeI32,
		ssair.OpCmpLeI32, ssair.OpCmpNeI32:
		left, err := readSSAValue("ssa", values, instr.Args[0])
		if err != nil {
			return 0, false, err
		}
		right, err := readSSAValue("ssa", values, instr.Args[1])
		if err != nil {
			return 0, false, err
		}
		value, err := evalSSABinary(instr.Kind, left, right)
		return value, true, err
	case ssair.OpNegI32:
		value, err := readSSAValue("ssa", values, instr.Args[0])
		if err != nil {
			return 0, false, err
		}
		return -value, true, nil
	case ssair.OpCall:
		callArgs, err := readSSAArgs("ssa", values, instr.Args)
		if err != nil {
			return 0, false, err
		}
		value, err := evalSSAFunctionI32(funcs, instr.Call, callArgs, memory, depth+1)
		if instr.EffectOut != "" {
			values[instr.EffectOut] = 0
		}
		return value, instr.Result != "", err
	case ssair.OpIndexLoadI32:
		base, length, index, err := readSSASliceAccess(values, instr.Args)
		if err != nil {
			return 0, false, err
		}
		value, err := loadI32Slice(memory, base, length, index, false)
		if instr.EffectOut != "" {
			values[instr.EffectOut] = 0
		}
		return value, true, err
	default:
		return 0, false, fmt.Errorf(
			"ssa instruction %s is outside backend differential subset",
			instr.Kind,
		)
	}
}

func machineScalarFunction(fn ir.IRFunc) (machine.Function, bool, error) {
	if mfn, ok, err := machine.ScalarI32SliceSumLoopFunctionFromStackIR(fn); err != nil || ok {
		return mfn, ok, err
	}
	if mfn, ok, err := machine.ScalarIntCallLoopFunctionFromStackIR(fn); err != nil || ok {
		return mfn, ok, err
	}
	if mfn, ok, err := machine.ScalarIntFunctionFromStackIR(fn); err != nil || ok {
		return mfn, ok, err
	}
	if mfn, ok, err := machine.ScalarIntLoopFunctionFromStackIR(fn); err != nil || ok {
		return mfn, ok, err
	}
	return machine.Function{}, false, nil
}

func cloneIRFunc(fn ir.IRFunc) ir.IRFunc {
	out := fn
	out.Instrs = append([]ir.IRInstr(nil), fn.Instrs...)
	return out
}

func defaultSamples(params int) [][]int32 {
	switch params {
	case 0:
		return [][]int32{{}}
	case 1:
		return [][]int32{{-2}, {-1}, {0}, {1}, {2}, {7}}
	default:
		return [][]int32{{0, 0}, {1, 2}, {-2, 7}}
	}
}

func evalStackBinary(kind ir.IRInstrKind, left int32, right int32) (int32, error) {
	switch kind {
	case ir.IRAddI32:
		return left + right, nil
	case ir.IRSubI32:
		return left - right, nil
	case ir.IRMulI32:
		return left * right, nil
	case ir.IRDivI32:
		if right == 0 {
			return 0, fmt.Errorf("division by zero")
		}
		return left / right, nil
	case ir.IRModI32:
		if right == 0 {
			return 0, fmt.Errorf("modulo by zero")
		}
		return left % right, nil
	case ir.IRCmpEqI32:
		return boolI32(left == right), nil
	case ir.IRCmpLtI32:
		return boolI32(left < right), nil
	case ir.IRCmpGtI32:
		return boolI32(left > right), nil
	case ir.IRCmpGeI32:
		return boolI32(left >= right), nil
	case ir.IRCmpLeI32:
		return boolI32(left <= right), nil
	case ir.IRCmpNeI32:
		return boolI32(left != right), nil
	default:
		return 0, fmt.Errorf("unsupported binary kind %d", kind)
	}
}

func evalMachineBinary(instr machine.Instr, left int32, right int32) (int32, error) {
	switch instr.Op {
	case machine.OpAdd:
		return left + right, nil
	case machine.OpSub:
		return left - right, nil
	case machine.OpMul:
		return left * right, nil
	case machine.OpDiv:
		if right == 0 {
			return 0, fmt.Errorf("division by zero")
		}
		return left / right, nil
	case machine.OpMod:
		if right == 0 {
			return 0, fmt.Errorf("modulo by zero")
		}
		return left % right, nil
	case machine.OpCmp:
		return boolI32(left < right), nil
	default:
		return 0, fmt.Errorf("unsupported machine binary op %s", instr.Op)
	}
}

func evalSSABinary(kind ssair.OpKind, left int32, right int32) (int32, error) {
	switch kind {
	case ssair.OpAddI32:
		return left + right, nil
	case ssair.OpSubI32:
		return left - right, nil
	case ssair.OpMulI32:
		return left * right, nil
	case ssair.OpDivI32:
		if right == 0 {
			return 0, fmt.Errorf("division by zero")
		}
		return left / right, nil
	case ssair.OpModI32:
		if right == 0 {
			return 0, fmt.Errorf("modulo by zero")
		}
		return left % right, nil
	case ssair.OpCmpEqI32:
		return boolI32(left == right), nil
	case ssair.OpCmpLtI32:
		return boolI32(left < right), nil
	case ssair.OpCmpGtI32:
		return boolI32(left > right), nil
	case ssair.OpCmpGeI32:
		return boolI32(left >= right), nil
	case ssair.OpCmpLeI32:
		return boolI32(left <= right), nil
	case ssair.OpCmpNeI32:
		return boolI32(left != right), nil
	default:
		return 0, fmt.Errorf("unsupported ssa binary op %s", kind)
	}
}

func EvalNativeLinuxX64Exit(
	funcs []ir.IRFunc,
	entry string,
	workDir string,
	name string,
) (int32, error) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return 0, fmt.Errorf(
			"native linux-x64 execution requires linux/amd64 host, got %s/%s",
			runtime.GOOS,
			runtime.GOARCH,
		)
	}
	if workDir == "" {
		return 0, fmt.Errorf("native linux-x64 execution requires workDir")
	}
	if name == "" {
		name = "differential-native"
	}
	obj, err := linux_x64.CodegenObjectLinuxX64(funcs)
	if err != nil {
		return 0, fmt.Errorf("CodegenObjectLinuxX64: %w", err)
	}
	img, err := linker.LinkLinuxX64([]*tobj.Object{obj}, entry)
	if err != nil {
		return 0, fmt.Errorf("LinkLinuxX64: %w", err)
	}
	path := filepath.Join(workDir, name)
	if err := elf.WriteELF64LinuxX64(path, img); err != nil {
		return 0, fmt.Errorf("WriteELF64LinuxX64: %w", err)
	}
	if err := os.Chmod(path, 0o755); err != nil {
		return 0, fmt.Errorf("chmod native executable: %w", err)
	}
	out, err := exec.Command(path).CombinedOutput()
	if len(out) != 0 {
		return 0, fmt.Errorf("native executable wrote output %q", out)
	}
	if exit, ok := err.(*exec.ExitError); ok {
		return int32(exit.ExitCode()), nil
	}
	if err != nil {
		return 0, fmt.Errorf("run native executable: %w", err)
	}
	return 0, nil
}

func normalizeMatrixSamples(tc BackendMatrixCase) []MatrixSample {
	samples := make([]MatrixSample, 0, len(tc.Samples)+tc.RandomSampleCount)
	for i, sample := range tc.Samples {
		if sample.Name == "" {
			sample.Name = fmt.Sprintf("sample-%d", i)
		}
		samples = append(samples, cloneMatrixSample(sample))
	}
	if len(samples) == 0 {
		if entry, ok := irFuncByName(tc.Functions, tc.Entry); ok {
			for i, args := range defaultSamples(entry.ParamSlots) {
				samples = append(
					samples,
					MatrixSample{Name: fmt.Sprintf("default-%d", i), Args: args},
				)
			}
		}
	}
	if tc.RandomSampleCount > 0 {
		seed := tc.RandomSeed
		if seed == 0 {
			seed = 1
		}
		entry, _ := irFuncByName(tc.Functions, tc.Entry)
		rng := rand.New(rand.NewSource(seed))
		for i := 0; i < tc.RandomSampleCount; i++ {
			args := make([]int32, entry.ParamSlots)
			for j := range args {
				args[j] = int32(rng.Intn(17) - 8)
			}
			samples = append(samples, MatrixSample{Name: fmt.Sprintf("random-%d", i), Args: args})
		}
	}
	return samples
}

func optimizedMatrixFunctions(funcs []ir.IRFunc, passes []opt.Pass) ([]ir.IRFunc, error) {
	out := cloneIRFuncs(funcs)
	if len(passes) == 0 {
		return out, nil
	}
	prog := &ir.IRProgram{Funcs: out}
	if len(out) > 0 {
		prog.MainName = out[0].Name
	}
	if _, err := opt.NewManager().Run(prog, passes...); err != nil {
		return nil, fmt.Errorf("backend differential matrix: optimize: %w", err)
	}
	return prog.Funcs, nil
}

func ssaMatrixFunctions(funcs []ir.IRFunc) (map[string]ssair.Function, []UnsupportedRow, error) {
	out := map[string]ssair.Function{}
	var unsupported []UnsupportedRow
	for _, fn := range funcs {
		ssaFn, ok, err := ssair.FromStackIRFunction(fn)
		if err != nil {
			return nil, nil, fmt.Errorf(
				"backend differential matrix: ssa lowering %s: %w",
				fn.Name,
				err,
			)
		}
		if !ok {
			unsupported = append(
				unsupported,
				UnsupportedRow{Function: fn.Name, Reason: "ssa_lowering_unsupported"},
			)
			continue
		}
		out[fn.Name] = ssaFn
	}
	return out, unsupported, nil
}

func machineMatrixFunctions(
	funcs []ir.IRFunc,
) (map[string]machine.Function, []UnsupportedRow, error) {
	out := map[string]machine.Function{}
	var unsupported []UnsupportedRow
	for _, fn := range funcs {
		mfn, ok, err := machineScalarFunction(fn)
		if err != nil {
			return nil, nil, fmt.Errorf(
				"backend differential matrix: machine lowering %s: %w",
				fn.Name,
				err,
			)
		}
		if !ok {
			unsupported = append(
				unsupported,
				UnsupportedRow{Function: fn.Name, Reason: "machine_lowering_unsupported"},
			)
			continue
		}
		out[fn.Name] = mfn
	}
	return out, unsupported, nil
}

func classifyMatrixCases(funcs []ir.IRFunc) map[string]int {
	out := map[string]int{
		"scalar_i32":      0,
		"scalar_loop_i32": 0,
		"call_loop_i32":   0,
		"slice_sum_i32":   0,
	}
	for _, fn := range funcs {
		if _, ok, _ := machine.ScalarI32SliceSumLoopFunctionFromStackIR(fn); ok {
			out["slice_sum_i32"]++
			continue
		}
		if _, ok, _ := machine.ScalarIntCallLoopFunctionFromStackIR(fn); ok {
			out["call_loop_i32"]++
			continue
		}
		if _, ok, _ := machine.ScalarIntLoopFunctionFromStackIR(fn); ok {
			out["scalar_loop_i32"]++
			continue
		}
		if _, ok, _ := machine.ScalarIntFunctionFromStackIR(fn); ok {
			out["scalar_i32"]++
		}
	}
	return out
}

func matrixMismatch(
	caseName string,
	sample MatrixSample,
	lane Lane,
	expected int32,
	got int32,
	sampleCount int,
) *MismatchReport {
	return &MismatchReport{
		Case:          caseName,
		SampleName:    sampleName(sample),
		Args:          append([]int32(nil), sample.Args...),
		Lane:          lane,
		ExpectedLane:  LaneSourceInterpreter,
		Expected:      expected,
		Got:           got,
		ReducerStatus: "reduced_to_single_sample",
		Reproducer: fmt.Sprintf(
			"case=%s sample=%s args=%v lane=%s original_samples=%d",
			caseName,
			sampleName(sample),
			sample.Args,
			lane,
			sampleCount,
		),
	}
}

func readSSAValue(fnName string, values map[ssair.ValueID]int32, id ssair.ValueID) (int32, error) {
	value, ok := values[id]
	if !ok {
		return 0, fmt.Errorf("%s reads undefined ssa value %s", fnName, id)
	}
	return value, nil
}

func readSSAArgs(
	fnName string,
	values map[ssair.ValueID]int32,
	args []ssair.ValueID,
) ([]int32, error) {
	out := make([]int32, len(args))
	for i, arg := range args {
		value, err := readSSAValue(fnName, values, arg)
		if err != nil {
			return nil, err
		}
		out[i] = value
	}
	return out, nil
}

func readSSASliceAccess(
	values map[ssair.ValueID]int32,
	args []ssair.ValueID,
) (int32, int32, int32, error) {
	if len(args) != 3 {
		return 0, 0, 0, fmt.Errorf("ssa index load arg count %d, want 3", len(args))
	}
	base, err := readSSAValue("ssa", values, args[0])
	if err != nil {
		return 0, 0, 0, err
	}
	length, err := readSSAValue("ssa", values, args[1])
	if err != nil {
		return 0, 0, 0, err
	}
	index, err := readSSAValue("ssa", values, args[2])
	if err != nil {
		return 0, 0, 0, err
	}
	return base, length, index, nil
}

func popSliceAccess(
	fnName string,
	kind ir.IRInstrKind,
	pop func(ir.IRInstrKind) (int32, error),
) (int32, int32, int32, error) {
	index, err := pop(kind)
	if err != nil {
		return 0, 0, 0, err
	}
	length, err := pop(kind)
	if err != nil {
		return 0, 0, 0, err
	}
	base, err := pop(kind)
	if err != nil {
		return 0, 0, 0, err
	}
	if length < 0 {
		return 0, 0, 0, fmt.Errorf("%s slice length %d is negative", fnName, length)
	}
	return index, length, base, nil
}

func loadI32Slice(
	memory map[int32][]int32,
	base int32,
	length int32,
	index int32,
	checked bool,
) (int32, error) {
	if checked && (index < 0 || index >= length) {
		return 0, fmt.Errorf("index %d out of bounds len %d", index, length)
	}
	xs, ok := memory[base]
	if !ok {
		return 0, fmt.Errorf("missing i32 slice base %d", base)
	}
	if index < 0 || index >= int32(len(xs)) || index >= length {
		return 0, fmt.Errorf(
			"index %d outside backing len %d logical len %d",
			index,
			len(xs),
			length,
		)
	}
	return xs[index], nil
}

func storeI32Slice(
	memory map[int32][]int32,
	base int32,
	length int32,
	index int32,
	value int32,
) error {
	if index < 0 || index >= length {
		return fmt.Errorf("index %d out of bounds len %d", index, length)
	}
	xs, ok := memory[base]
	if !ok {
		return fmt.Errorf("missing i32 slice base %d", base)
	}
	if index >= int32(len(xs)) {
		return fmt.Errorf("index %d outside backing len %d", index, len(xs))
	}
	xs[index] = value
	return nil
}

func cloneI32Slices(memory map[int32][]int32) map[int32][]int32 {
	if len(memory) == 0 {
		return map[int32][]int32{}
	}
	out := make(map[int32][]int32, len(memory))
	for base, values := range memory {
		out[base] = append([]int32(nil), values...)
	}
	return out
}

func cloneMatrixSample(sample MatrixSample) MatrixSample {
	return MatrixSample{
		Name:      sample.Name,
		Args:      append([]int32(nil), sample.Args...),
		I32Slices: cloneI32Slices(sample.I32Slices),
	}
}

func cloneIRFuncs(funcs []ir.IRFunc) []ir.IRFunc {
	out := make([]ir.IRFunc, len(funcs))
	for i, fn := range funcs {
		out[i] = cloneIRFunc(fn)
	}
	return out
}

func irFuncByName(funcs []ir.IRFunc, name string) (ir.IRFunc, bool) {
	for _, fn := range funcs {
		if fn.Name == name {
			return fn, true
		}
	}
	return ir.IRFunc{}, false
}

func sampleName(sample MatrixSample) string {
	if sample.Name != "" {
		return sample.Name
	}
	return "sample"
}

func formatUnsupportedRows(rows []UnsupportedRow) string {
	parts := make([]string, 0, len(rows))
	for _, row := range rows {
		parts = append(parts, row.Function+":"+row.Reason)
	}
	return strings.Join(parts, ",")
}

func boolI32(value bool) int32 {
	if value {
		return 1
	}
	return 0
}
