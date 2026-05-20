package lower

import (
	"fmt"
	"hash/fnv"
	"sort"
	"strings"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/semantics"
)

type runtimePolicy struct {
	hasBudget    bool
	budget       int32
	consentParam string
}

const consentTokenRuntimeSentinel int32 = -0x43544f4b

func runtimePolicyFromClauses(clauses []frontend.SemanticClause) runtimePolicy {
	policy := runtimePolicy{}
	for _, clause := range clauses {
		switch clause.Name {
		case "budget":
			if v, ok := clauseConstI32(clause.Value); ok {
				policy.hasBudget = true
				policy.budget = v
			}
		case "consent":
			if ident, ok := clause.Value.(*frontend.IdentExpr); ok {
				policy.consentParam = ident.Name
			}
		}
	}
	return policy
}

func clauseConstI32(expr frontend.Expr) (int32, bool) {
	switch e := expr.(type) {
	case *frontend.NumberExpr:
		return e.Value, true
	case *frontend.UnaryExpr:
		if e.Op != frontend.TokenMinus {
			return 0, false
		}
		v, ok := e.X.(*frontend.NumberExpr)
		if !ok {
			return 0, false
		}
		return -v.Value, true
	default:
		return 0, false
	}
}

type budgetCharge struct {
	kind ir.IRInstrKind
	cost int32
}

const (
	policyFailureDefaultSlot int32 = 0
	policyFailureStatusTrap  int32 = 1
)

var budgetChargeTable = []budgetCharge{
	{kind: ir.IRWrite, cost: 1},
	{kind: ir.IRCall, cost: 1},
	{kind: ir.IRAllocBytes, cost: 1},
	{kind: ir.IRMakeSliceU8, cost: 1},
	{kind: ir.IRMakeSliceU16, cost: 1},
	{kind: ir.IRMakeSliceI32, cost: 1},
	{kind: ir.IRIndexLoadI32, cost: 1},
	{kind: ir.IRIndexStoreI32, cost: 1},
	{kind: ir.IRIndexLoadU8, cost: 1},
	{kind: ir.IRIndexStoreU8, cost: 1},
	{kind: ir.IRIndexLoadU16, cost: 1},
	{kind: ir.IRIndexStoreU16, cost: 1},
	{kind: ir.IRIslandNew, cost: 1},
	{kind: ir.IRIslandMakeSliceU8, cost: 1},
	{kind: ir.IRIslandMakeSliceU16, cost: 1},
	{kind: ir.IRIslandMakeSliceI32, cost: 1},
	{kind: ir.IRIslandFree, cost: 1},
	{kind: ir.IRCapIO, cost: 1},
	{kind: ir.IRCapMem, cost: 1},
	{kind: ir.IRMemReadI32, cost: 1},
	{kind: ir.IRMemWriteI32, cost: 1},
	{kind: ir.IRMemReadU8, cost: 1},
	{kind: ir.IRMemWriteU8, cost: 1},
	{kind: ir.IRMemReadPtr, cost: 1},
	{kind: ir.IRMemWritePtr, cost: 1},
	{kind: ir.IRMemReadI32Offset, cost: 1},
	{kind: ir.IRMemWriteI32Offset, cost: 1},
	{kind: ir.IRMemReadU8Offset, cost: 1},
	{kind: ir.IRMemWriteU8Offset, cost: 1},
	{kind: ir.IRMemReadPtrOffset, cost: 1},
	{kind: ir.IRMemWritePtrOffset, cost: 1},
	{kind: ir.IRPtrAdd, cost: 1},
	{kind: ir.IRMmioReadI32, cost: 1},
	{kind: ir.IRMmioWriteI32, cost: 1},
	{kind: ir.IRSymAddr, cost: 1},
	{kind: ir.IRCtxSwitch, cost: 1},
}

func budgetChargeForInstr(kind ir.IRInstrKind) (int32, bool) {
	for _, charge := range budgetChargeTable {
		if charge.kind == kind {
			return charge.cost, true
		}
	}
	return 0, false
}

func budgetChargedInstr(kind ir.IRInstrKind) bool {
	_, ok := budgetChargeForInstr(kind)
	return ok
}

func Lower(checked *semantics.CheckedProgram) (*ir.IRProgram, error) {
	if checked == nil {
		return nil, fmt.Errorf("missing checked program")
	}
	if len(checked.Funcs) == 0 {
		return nil, fmt.Errorf("expected at least one function")
	}

	prog := ir.IRProgram{MainIndex: checked.MainIndex, MainName: checked.MainName}
	wrappers := collectTypedTaskWrappers(checked, "")
	stagedTargets := collectStagedTypedTaskTargets(wrappers)
	callableTargets := collectFunctionTypedParamTargets(checked, "")
	for _, fn := range checked.Funcs {
		irFunc, err := lowerCheckedFunc(fn, checked.Types, checked.FuncSigs, checked.GlobalsByModule[fn.Module], stagedTargets[fn.Name], callableTargets[fn.Name])
		if err != nil {
			return nil, err
		}
		if err := VerifyFunc(irFunc); err != nil {
			return nil, err
		}
		prog.Funcs = append(prog.Funcs, irFunc)
	}
	for _, wrapper := range wrappers {
		irFunc, err := lowerTypedTaskWrapper(wrapper)
		if err != nil {
			return nil, err
		}
		if err := VerifyFunc(irFunc); err != nil {
			return nil, err
		}
		prog.Funcs = append(prog.Funcs, irFunc)
	}
	if err := VerifyProgram(&prog); err != nil {
		return nil, err
	}
	return &prog, nil
}

func LowerModule(checked *semantics.CheckedProgram, module string) ([]ir.IRFunc, error) {
	if checked == nil {
		return nil, fmt.Errorf("missing checked program")
	}
	var out []ir.IRFunc
	wrappers := collectTypedTaskWrappers(checked, module)
	stagedTargets := collectStagedTypedTaskTargets(wrappers)
	callableTargets := collectFunctionTypedParamTargets(checked, "")
	for _, fn := range checked.Funcs {
		if fn.Module != module {
			continue
		}
		irFunc, err := lowerCheckedFunc(fn, checked.Types, checked.FuncSigs, checked.GlobalsByModule[fn.Module], stagedTargets[fn.Name], callableTargets[fn.Name])
		if err != nil {
			return nil, err
		}
		if err := VerifyFunc(irFunc); err != nil {
			return nil, err
		}
		out = append(out, irFunc)
	}
	for _, wrapper := range wrappers {
		irFunc, err := lowerTypedTaskWrapper(wrapper)
		if err != nil {
			return nil, err
		}
		if err := VerifyFunc(irFunc); err != nil {
			return nil, err
		}
		out = append(out, irFunc)
	}
	return out, nil
}

func LowerModules(checked *semantics.CheckedProgram) (map[string][]ir.IRFunc, error) {
	if checked == nil {
		return nil, fmt.Errorf("missing checked program")
	}
	modules := make(map[string][]ir.IRFunc)
	wrappers := collectTypedTaskWrappers(checked, "")
	stagedTargets := collectStagedTypedTaskTargets(wrappers)
	callableTargets := collectFunctionTypedParamTargets(checked, "")
	for _, fn := range checked.Funcs {
		irFunc, err := lowerCheckedFunc(fn, checked.Types, checked.FuncSigs, checked.GlobalsByModule[fn.Module], stagedTargets[fn.Name], callableTargets[fn.Name])
		if err != nil {
			return nil, err
		}
		if err := VerifyFunc(irFunc); err != nil {
			return nil, err
		}
		modules[fn.Module] = append(modules[fn.Module], irFunc)
	}
	for _, wrapper := range wrappers {
		irFunc, err := lowerTypedTaskWrapper(wrapper)
		if err != nil {
			return nil, err
		}
		if err := VerifyFunc(irFunc); err != nil {
			return nil, err
		}
		modules[wrapper.Module] = append(modules[wrapper.Module], irFunc)
	}
	return modules, nil
}

func lowerCheckedFunc(fn semantics.CheckedFunc, types map[string]*semantics.TypeInfo, funcs map[string]semantics.FuncSig, globals map[string]semantics.GlobalInfo, stagedTarget typedTaskStagedTarget, callableParamTargets map[string][]string) (ir.IRFunc, error) {
	throwSuccessSlots := 0
	throwErrorSlots := 0
	throwCompact := false
	throwScratchBase := 0
	if fn.ThrowsType != "" {
		var err error
		throwSuccessSlots, throwErrorSlots, throwCompact, err = throwingLayout(fn.ReturnType, fn.ThrowsType, types)
		if err != nil {
			return ir.IRFunc{}, err
		}
		throwScratchBase = fn.LocalSlots - throwErrorSlots
		if throwScratchBase < 0 {
			return ir.IRFunc{}, fmt.Errorf("internal error: invalid throwing scratch layout for '%s'", fn.Name)
		}
	}
	policy := runtimePolicyFromClauses(fn.Decl.SemanticClauses)
	localSlots := fn.LocalSlots
	budgetLocal := -1
	if policy.hasBudget {
		budgetLocal = localSlots
		localSlots++
	}
	effectiveReturnSlots := fn.ReturnSlots
	if stagedTarget.SlotCount > 4 {
		effectiveReturnSlots = 1
	}
	l := &lowerer{
		locals:               fn.Locals,
		actorState:           fn.ActorState,
		globals:              globals,
		types:                types,
		funcs:                funcs,
		imports:              fn.Imports,
		module:               fn.Module,
		localSlots:           localSlots,
		returnType:           fn.ReturnType,
		throwsType:           fn.ThrowsType,
		returnSlots:          effectiveReturnSlots,
		throwSuccessSlots:    throwSuccessSlots,
		throwErrorSlots:      throwErrorSlots,
		throwCompact:         throwCompact,
		throwScratchBase:     throwScratchBase,
		policyFailLabel:      -1,
		budgetEnabled:        policy.hasBudget,
		budgetLocal:          budgetLocal,
		discardLocal:         -1,
		budgetScratchBase:    -1,
		stagedTaskTarget:     stagedTarget,
		callableParamTargets: callableParamTargets,
		rawPtrOffsetLocals:   map[int]rawPtrOffsetLocal{},
	}
	if policy.hasBudget || policy.consentParam != "" {
		l.policyFailLabel = l.newLabel()
	}
	if policy.hasBudget {
		l.emitRaw(ir.IRInstr{Kind: ir.IRConstI32, Imm: policy.budget, Pos: fn.Decl.Pos})
		l.emitRaw(ir.IRInstr{Kind: ir.IRStoreLocal, Local: budgetLocal, Pos: fn.Decl.Pos})
	}
	if policy.consentParam != "" {
		info, ok := l.locals[policy.consentParam]
		if !ok {
			return ir.IRFunc{}, fmt.Errorf("%s: semantic clause 'consent' references unknown local '%s' during lowering", frontend.FormatPos(fn.Decl.Pos), policy.consentParam)
		}
		if info.SlotCount != 1 {
			return ir.IRFunc{}, fmt.Errorf("%s: semantic clause 'consent' expects 1-slot token parameter '%s'", frontend.FormatPos(fn.Decl.Pos), policy.consentParam)
		}
		l.emitRaw(ir.IRInstr{Kind: ir.IRLoadLocal, Local: info.Base, Pos: fn.Decl.Pos})
		l.emitRaw(ir.IRInstr{Kind: ir.IRConstI32, Imm: consentTokenRuntimeSentinel, Pos: fn.Decl.Pos})
		l.emitRaw(ir.IRInstr{Kind: ir.IRCmpEqI32, Pos: fn.Decl.Pos})
		l.emitRaw(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: l.policyFailLabel, Pos: fn.Decl.Pos})
	}
	if err := l.lowerBlock(fn.Decl.Body, fn.Decl.Pos); err != nil {
		return ir.IRFunc{}, err
	}
	if l.policyFailLabel >= 0 {
		l.emitPolicyFailureHandler(fn.Decl.Pos)
	}
	irPolicy := ir.IRPolicy{
		HasBudget:    policy.hasBudget,
		Budget:       policy.budget,
		BudgetLocal:  budgetLocal,
		HasConsent:   policy.consentParam != "",
		ConsentLocal: -1,
		FailLabel:    l.policyFailLabel,
	}
	if policy.consentParam != "" {
		irPolicy.ConsentLocal = l.locals[policy.consentParam].Base
	}
	return ir.IRFunc{
		Name:        fn.Name,
		ExportName:  fn.Decl.ExportName,
		ParamSlots:  fn.ParamSlots,
		LocalSlots:  l.localSlots,
		ReturnSlots: l.returnSlots,
		Policy:      irPolicy,
		Instrs:      l.instrs,
	}, nil
}

// lowerer owns function-local IR emission state. stackHeight is emission-side
// bookkeeping used to preserve local invariants while lowering; VerifyFunc is
// still the final target-neutral check before codegen sees the function.
type lowerer struct {
	instrs               []ir.IRInstr
	locals               map[string]semantics.LocalInfo
	actorState           map[string]semantics.ActorStateField
	globals              map[string]semantics.GlobalInfo
	types                map[string]*semantics.TypeInfo
	funcs                map[string]semantics.FuncSig
	imports              map[string]string
	module               string
	localSlots           int
	returnType           string
	throwsType           string
	returnSlots          int
	throwSuccessSlots    int
	throwErrorSlots      int
	throwCompact         bool
	throwScratchBase     int
	policyFailLabel      int
	budgetEnabled        bool
	budgetLocal          int
	discardLocal         int
	budgetScratchBase    int
	budgetScratchSlots   int
	stagedTaskTarget     typedTaskStagedTarget
	callableParamTargets map[string][]string
	rawPtrOffsetLocals   map[int]rawPtrOffsetLocal
	preparedStringFields map[string]bool
	stackHeight          int
	nextLabel            int
	cleanupIslands       []int
	deferFrames          []deferFrame
	loopStack            []loopLabels
}

type rawPtrOffsetLocal struct {
	BaseLocal    int
	OffsetLocal  int
	OffsetImm    int32
	HasOffsetImm bool
}

type typedTaskWrapper struct {
	Name              string
	Target            string
	Module            string
	ErrorType         string
	TargetThrowsType  string
	SlotCount         int
	StatusSlot        int
	TargetReturnSlots int
}

type typedTaskStagedTarget struct {
	SlotCount int
	ErrorType string
}

func typedTaskWrapperName(target, errorType string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(target))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(errorType))
	return fmt.Sprintf("__tetra_task_typed_%08x", h.Sum32())
}

func collectTypedTaskWrappers(checked *semantics.CheckedProgram, module string) []typedTaskWrapper {
	if checked == nil {
		return nil
	}
	targetModules := make(map[string]string, len(checked.Funcs))
	targetReturnSlots := make(map[string]int, len(checked.FuncSigs))
	targetThrowsTypes := make(map[string]string, len(checked.FuncSigs))
	for _, fn := range checked.Funcs {
		targetModules[fn.Name] = fn.Module
	}
	for name, sig := range checked.FuncSigs {
		targetReturnSlots[name] = sig.ReturnSlots
		targetThrowsTypes[name] = sig.ThrowsType
	}
	seen := make(map[string]typedTaskWrapper)

	var walkExpr func(frontend.Expr)
	var walkStmt func(frontend.Stmt)
	addCall := func(call *frontend.CallExpr, workerArg int) {
		if len(call.TypeArgs) != 1 || call.TypeArgs[0].Name == "" || len(call.Args) <= workerArg {
			return
		}
		lit, ok := call.Args[workerArg].(*frontend.StringLitExpr)
		if !ok || string(lit.Value) == "" {
			return
		}
		target := string(lit.Value)
		targetModule, targetOK := targetModules[target]
		if !targetOK || (module != "" && targetModule != module) {
			return
		}
		_, handleInfo, err := semantics.EnsureTypedTaskHandleType(call.TypeArgs[0].Name, checked.Types)
		if err != nil {
			return
		}
		name := typedTaskWrapperName(target, call.TypeArgs[0].Name)
		targetSlots := targetReturnSlots[target]
		if handleInfo.SlotCount > 4 {
			targetSlots = 1
		}
		seen[name] = typedTaskWrapper{
			Name:              name,
			Target:            target,
			Module:            targetModule,
			ErrorType:         call.TypeArgs[0].Name,
			TargetThrowsType:  targetThrowsTypes[target],
			SlotCount:         handleInfo.SlotCount,
			StatusSlot:        handleInfo.SlotCount - 1,
			TargetReturnSlots: targetSlots,
		}
	}

	walkExpr = func(expr frontend.Expr) {
		switch e := expr.(type) {
		case *frontend.CallExpr:
			name := e.Name
			if builtin, ok := semantics.ResolveBuiltinAlias(name); ok {
				name = builtin
			}
			switch name {
			case "core.task_spawn_i32_typed":
				addCall(e, 0)
			case "core.task_spawn_group_i32_typed":
				addCall(e, 1)
			}
			for _, arg := range e.Args {
				walkExpr(arg)
			}
		case *frontend.StructLitExpr:
			for _, field := range e.Fields {
				walkExpr(field.Value)
			}
		case *frontend.FieldAccessExpr:
			walkExpr(e.Base)
		case *frontend.IndexExpr:
			walkExpr(e.Base)
			walkExpr(e.Index)
		case *frontend.BinaryExpr:
			walkExpr(e.Left)
			walkExpr(e.Right)
		case *frontend.UnaryExpr:
			walkExpr(e.X)
		case *frontend.TryExpr:
			walkExpr(e.X)
		case *frontend.MatchExpr:
			walkExpr(e.Value)
			for _, c := range e.Cases {
				if c.Pattern != nil {
					walkExpr(c.Pattern)
				}
				if c.Guard != nil {
					walkExpr(c.Guard)
				}
				walkExpr(c.Value)
			}
		case *frontend.CatchExpr:
			walkExpr(e.Call)
			for _, c := range e.Cases {
				if c.Pattern != nil {
					walkExpr(c.Pattern)
				}
				if c.Guard != nil {
					walkExpr(c.Guard)
				}
				walkExpr(c.Value)
			}
		}
	}

	walkStmt = func(stmt frontend.Stmt) {
		switch s := stmt.(type) {
		case *frontend.PrintStmt:
			walkExpr(s.Value)
		case *frontend.ReturnStmt:
			walkExpr(s.Value)
		case *frontend.ThrowStmt:
			walkExpr(s.Value)
		case *frontend.DeferStmt:
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		case *frontend.LetStmt:
			walkExpr(s.Value)
		case *frontend.AssignStmt:
			walkExpr(s.Target)
			walkExpr(s.Value)
		case *frontend.IfStmt:
			walkExpr(s.Cond)
			for _, inner := range s.Then {
				walkStmt(inner)
			}
			for _, inner := range s.Else {
				walkStmt(inner)
			}
		case *frontend.WhileStmt:
			walkExpr(s.Cond)
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		case *frontend.ForRangeStmt:
			if s.Iterable != nil {
				walkExpr(s.Iterable)
			} else {
				walkExpr(s.Start)
				walkExpr(s.End)
			}
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		case *frontend.MatchStmt:
			walkExpr(s.Value)
			for _, c := range s.Cases {
				if !c.Default {
					walkExpr(c.Pattern)
				}
				for _, inner := range c.Body {
					walkStmt(inner)
				}
			}
		case *frontend.FreeStmt:
			walkExpr(s.Value)
		case *frontend.UnsafeStmt:
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		case *frontend.IslandStmt:
			walkExpr(s.Size)
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		}
	}

	for _, fn := range checked.Funcs {
		if fn.Decl == nil {
			continue
		}
		for _, stmt := range fn.Decl.Body {
			walkStmt(stmt)
		}
	}

	out := make([]typedTaskWrapper, 0, len(seen))
	for _, wrapper := range seen {
		out = append(out, wrapper)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func collectStagedTypedTaskTargets(wrappers []typedTaskWrapper) map[string]typedTaskStagedTarget {
	if len(wrappers) == 0 {
		return nil
	}
	out := map[string]typedTaskStagedTarget{}
	for _, wrapper := range wrappers {
		if wrapper.SlotCount <= 4 {
			continue
		}
		if wrapper.ErrorType == "" {
			continue
		}
		if wrapper.TargetThrowsType != wrapper.ErrorType {
			continue
		}
		out[wrapper.Target] = typedTaskStagedTarget{SlotCount: wrapper.SlotCount, ErrorType: wrapper.ErrorType}
	}
	return out
}

func lowerTypedTaskWrapper(wrapper typedTaskWrapper) (ir.IRFunc, error) {
	if wrapper.SlotCount < 2 || wrapper.SlotCount > 8 {
		return ir.IRFunc{}, lowerUnsupportedError(frontend.Position{}, "typed task wrapper %s has unsupported slot count %d", wrapper.Name, wrapper.SlotCount)
	}
	discard := wrapper.SlotCount
	var instrs []ir.IRInstr
	if wrapper.SlotCount > 4 {
		if wrapper.TargetReturnSlots != 1 {
			return ir.IRFunc{}, lowerUnsupportedError(frontend.Position{}, "typed task wrapper %s staged mode requires a 1-slot target return, got %d", wrapper.Name, wrapper.TargetReturnSlots)
		}
		if wrapper.ErrorType != "" && wrapper.TargetThrowsType == wrapper.ErrorType {
			instrs = append(instrs, ir.IRInstr{Kind: ir.IRCall, Name: wrapper.Target, ArgSlots: 0, RetSlots: 1})
			instrs = append(instrs, ir.IRInstr{Kind: ir.IRReturn})
			return ir.IRFunc{
				Name:        wrapper.Name,
				ParamSlots:  0,
				LocalSlots:  0,
				ReturnSlots: 1,
				Instrs:      instrs,
			}, nil
		}
		instrs = append(instrs, ir.IRInstr{Kind: ir.IRCall, Name: wrapper.Target, ArgSlots: 0, RetSlots: 1})
		instrs = append(instrs, ir.IRInstr{Kind: ir.IRStoreLocal, Local: 0})
		instrs = append(instrs,
			ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(wrapper.SlotCount)},
			ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_result_begin", ArgSlots: 1, RetSlots: 1},
			ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard},
			ir.IRInstr{Kind: ir.IRConstI32, Imm: 0},
			ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
			ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_result_slot", ArgSlots: 2, RetSlots: 1},
			ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard},
		)
		for slot := 1; slot < wrapper.SlotCount-1; slot++ {
			instrs = append(instrs,
				ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(slot)},
				ir.IRInstr{Kind: ir.IRConstI32, Imm: 0},
				ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_result_slot", ArgSlots: 2, RetSlots: 1},
				ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard},
			)
		}
		instrs = append(instrs,
			ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(wrapper.StatusSlot)},
			ir.IRInstr{Kind: ir.IRConstI32, Imm: 0},
			ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_result_slot", ArgSlots: 2, RetSlots: 1},
			ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard},
			ir.IRInstr{Kind: ir.IRConstI32, Imm: 0},
			ir.IRInstr{Kind: ir.IRReturn},
		)
		return ir.IRFunc{
			Name:        wrapper.Name,
			ParamSlots:  0,
			LocalSlots:  wrapper.SlotCount + 1,
			ReturnSlots: 1,
			Instrs:      instrs,
		}, nil
	}
	instrs = append(instrs, ir.IRInstr{Kind: ir.IRCall, Name: wrapper.Target, ArgSlots: 0, RetSlots: wrapper.SlotCount})
	for slot := wrapper.SlotCount - 1; slot >= 0; slot-- {
		instrs = append(instrs, ir.IRInstr{Kind: ir.IRStoreLocal, Local: slot})
	}
	instrs = append(instrs,
		ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(wrapper.SlotCount)},
		ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_result_begin", ArgSlots: 1, RetSlots: 1},
		ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard},
	)
	for slot := 0; slot < wrapper.SlotCount; slot++ {
		instrs = append(instrs,
			ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(slot)},
			ir.IRInstr{Kind: ir.IRLoadLocal, Local: slot},
			ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_result_slot", ArgSlots: 2, RetSlots: 1},
			ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard},
		)
	}
	instrs = append(instrs,
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: wrapper.StatusSlot},
		ir.IRInstr{Kind: ir.IRReturn},
	)
	return ir.IRFunc{
		Name:        wrapper.Name,
		ParamSlots:  0,
		LocalSlots:  wrapper.SlotCount + 1,
		ReturnSlots: 1,
		Instrs:      instrs,
	}, nil
}

// throwingLayout computes the slot layout for typed-error returns. The compact
// path is only valid when both success and error payloads fit in one slot.
func throwingLayout(returnType, throwsType string, types map[string]*semantics.TypeInfo) (int, int, bool, error) {
	if throwsType == "" {
		return 0, 0, false, nil
	}
	retInfo, ok := types[returnType]
	if !ok {
		return 0, 0, false, fmt.Errorf("unknown type '%s'", returnType)
	}
	throwInfo, ok := types[throwsType]
	if !ok {
		return 0, 0, false, fmt.Errorf("unknown type '%s'", throwsType)
	}
	compact := retInfo.SlotCount == 1 && throwInfo.SlotCount == 1
	return retInfo.SlotCount, throwInfo.SlotCount, compact, nil
}

func throwingReturnSlotCount(successSlots, errorSlots int) int {
	if successSlots == 1 && errorSlots == 1 {
		return 2
	}
	return successSlots + errorSlots + 1
}

type loopLabels struct {
	continueLabel int
	breakLabel    int
	cleanupDepth  int
	deferDepth    int
}

type deferFrame struct {
	bodies [][]frontend.Stmt
}

func (l *lowerer) newLabel() int {
	id := l.nextLabel
	l.nextLabel++
	return id
}

func (l *lowerer) emit(instr ir.IRInstr) {
	if cost, ok := budgetChargeForInstr(instr.Kind); l.budgetEnabled && l.policyFailLabel >= 0 && ok {
		l.emitBudgetGuardPreservingStack(instr.Pos, cost)
	}
	l.emitRaw(instr)
}

func (l *lowerer) emitRaw(instr ir.IRInstr) {
	l.instrs = append(l.instrs, instr)
	pop, push, _ := stackEffect(instr)
	if l.stackHeight < pop {
		l.stackHeight = 0
	} else {
		l.stackHeight = l.stackHeight - pop + push
	}
	if instr.Kind == ir.IRReturn {
		l.stackHeight = 0
	}
}

func (l *lowerer) emitBudgetGuardPreservingStack(pos frontend.Position, cost int32) {
	depth := l.stackHeight
	if depth == 0 {
		l.emitBudgetGuard(pos, cost)
		return
	}
	base := l.ensureBudgetScratchSlots(depth)
	for slot := depth - 1; slot >= 0; slot-- {
		l.emitRaw(ir.IRInstr{Kind: ir.IRStoreLocal, Local: base + slot, Pos: pos})
	}
	l.emitBudgetGuard(pos, cost)
	for slot := 0; slot < depth; slot++ {
		l.emitRaw(ir.IRInstr{Kind: ir.IRLoadLocal, Local: base + slot, Pos: pos})
	}
}

func (l *lowerer) ensureBudgetScratchSlots(slots int) int {
	if l.budgetScratchBase >= 0 && l.budgetScratchSlots >= slots {
		return l.budgetScratchBase
	}
	if l.budgetScratchBase >= 0 {
		l.localSlots += slots - l.budgetScratchSlots
		l.budgetScratchSlots = slots
		return l.budgetScratchBase
	}
	l.budgetScratchBase = l.localSlots
	l.budgetScratchSlots = slots
	l.localSlots += slots
	return l.budgetScratchBase
}

func (l *lowerer) emitBudgetGuard(pos frontend.Position, cost int32) {
	if l.budgetLocal < 0 {
		return
	}
	l.emitRaw(ir.IRInstr{Kind: ir.IRLoadLocal, Local: l.budgetLocal, Pos: pos})
	l.emitRaw(ir.IRInstr{Kind: ir.IRConstI32, Imm: cost, Pos: pos})
	l.emitRaw(ir.IRInstr{Kind: ir.IRSubI32, Pos: pos})
	l.emitRaw(ir.IRInstr{Kind: ir.IRStoreLocal, Local: l.budgetLocal, Pos: pos})
	l.emitRaw(ir.IRInstr{Kind: ir.IRLoadLocal, Local: l.budgetLocal, Pos: pos})
	l.emitRaw(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: pos})
	l.emitRaw(ir.IRInstr{Kind: ir.IRCmpGeI32, Pos: pos})
	l.emitRaw(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: l.policyFailLabel, Pos: pos})
}

func (l *lowerer) emitCleanup(pos frontend.Position) {
	l.emitCleanupSince(0, pos)
}

func (l *lowerer) emitCleanupSince(start int, pos frontend.Position) {
	for i := len(l.cleanupIslands) - 1; i >= 0; i-- {
		if i < start {
			break
		}
		base := l.cleanupIslands[i]
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: base, Pos: pos})
		l.emit(ir.IRInstr{Kind: ir.IRIslandFree, Pos: pos})
	}
}

func (l *lowerer) emitCleanupRaw(pos frontend.Position) {
	l.emitCleanupRawSince(0, pos)
}

func (l *lowerer) emitCleanupRawSince(start int, pos frontend.Position) {
	for i := len(l.cleanupIslands) - 1; i >= 0; i-- {
		if i < start {
			break
		}
		base := l.cleanupIslands[i]
		l.emitRaw(ir.IRInstr{Kind: ir.IRLoadLocal, Local: base, Pos: pos})
		l.emitRaw(ir.IRInstr{Kind: ir.IRIslandFree, Pos: pos})
	}
}

func (l *lowerer) emitZeroSlots(count int, pos frontend.Position) {
	for i := 0; i < count; i++ {
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: pos})
	}
}

func (l *lowerer) emitZeroSlotsRaw(count int, pos frontend.Position) {
	for i := 0; i < count; i++ {
		l.emitRaw(ir.IRInstr{Kind: ir.IRConstI32, Imm: policyFailureDefaultSlot, Pos: pos})
	}
}

// emitPolicyFailureHandler is the public lowering ABI for budget exhaustion and
// the current local policy-failure path. Non-throwing functions return their
// normal result shape filled with zero/default slots. Throwing functions return
// the normal throwing result shape with a zero/default error payload and status
// 1, so catch/try observe a deterministic trap-shaped error result.
func (l *lowerer) emitPolicyFailureHandler(pos frontend.Position) {
	l.emitRaw(ir.IRInstr{Kind: ir.IRLabel, Label: l.policyFailLabel, Pos: pos})
	if l.stagedTaskTarget.SlotCount > 4 {
		if err := l.emitStageTypedTaskStatus(policyFailureDefaultSlot, policyFailureStatusTrap, l.stagedTaskTarget.SlotCount, pos); err == nil {
			l.emitCleanupRaw(pos)
			l.emitRaw(ir.IRInstr{Kind: ir.IRConstI32, Imm: policyFailureStatusTrap, Pos: pos})
			l.emitRaw(ir.IRInstr{Kind: ir.IRReturn, Pos: pos})
			return
		}
	}
	if l.throwsType != "" {
		if l.throwCompact {
			l.emitRaw(ir.IRInstr{Kind: ir.IRConstI32, Imm: policyFailureDefaultSlot, Pos: pos})
		} else {
			l.emitZeroSlotsRaw(l.throwSuccessSlots, pos)
			l.emitZeroSlotsRaw(l.throwErrorSlots, pos)
		}
		l.emitRaw(ir.IRInstr{Kind: ir.IRConstI32, Imm: policyFailureStatusTrap, Pos: pos})
	} else {
		l.emitZeroSlotsRaw(l.returnSlots, pos)
	}
	l.emitCleanupRaw(pos)
	l.emitRaw(ir.IRInstr{Kind: ir.IRReturn, Pos: pos})
}

func (l *lowerer) emitConvertedThrowFromScratch(srcType, dstType string, pos frontend.Position) (int, error) {
	return l.emitConvertedValueFromScratch(srcType, dstType, l.throwScratchBase, pos)
}

func (l *lowerer) lowerTypedTaskJoin(call *frontend.CallExpr, pos frontend.Position) (int, error) {
	if l.throwsType == "" {
		return 0, fmt.Errorf("%s: try is only allowed in throwing functions", frontend.FormatPos(pos))
	}
	if len(call.TypeArgs) != 1 {
		return 0, fmt.Errorf("%s: task_join_i32_typed expects one explicit error type argument", frontend.FormatPos(call.At))
	}
	errorType := call.TypeArgs[0].Name
	if errorType == "" {
		return 0, fmt.Errorf("%s: task_join_i32_typed missing resolved error type", frontend.FormatPos(call.At))
	}
	if errorType != l.throwsType {
		return 0, fmt.Errorf("%s: thrown error type mismatch: expected '%s', got '%s'", frontend.FormatPos(call.At), l.throwsType, errorType)
	}
	errorInfo, ok := l.types[errorType]
	if !ok || errorInfo.Kind != semantics.TypeEnum {
		return 0, fmt.Errorf("%s: typed task error argument must be an enum", frontend.FormatPos(call.TypeArgs[0].At))
	}
	handleType, handleInfo, err := semantics.EnsureTypedTaskHandleType(errorType, l.types)
	if err != nil {
		return 0, fmt.Errorf("%s: %v", frontend.FormatPos(call.TypeArgs[0].At), err)
	}
	if len(call.Args) != 1 {
		return 0, fmt.Errorf("%s: task_join_i32_typed expects 1 argument", frontend.FormatPos(call.At))
	}
	argType, err := l.inferExprType(call.Args[0])
	if err != nil {
		return 0, err
	}
	if argType != handleType {
		return 0, fmt.Errorf("%s: task_join_i32_typed expects a %s handle", frontend.FormatPos(call.Args[0].Pos()), handleType)
	}
	slots, err := l.lowerExpr(call.Args[0])
	if err != nil {
		return 0, err
	}
	if slots != handleInfo.SlotCount {
		return 0, fmt.Errorf("%s: task_join_i32_typed handle slot mismatch", frontend.FormatPos(call.Args[0].Pos()))
	}
	if handleInfo.SlotCount > 4 {
		statusLocal := l.allocScratchSlots(1)
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: typedTaskJoinRuntimeSymbol(handleInfo.SlotCount), ArgSlots: handleInfo.SlotCount, RetSlots: 1, Pos: pos})
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: statusLocal, Pos: pos})
		if err := l.emitLoadTypedTaskResultSlots(handleInfo.SlotCount-1, pos); err != nil {
			return 0, err
		}
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: statusLocal, Pos: pos})

		okLabel := l.newLabel()
		l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: okLabel, Pos: pos})

		if errorInfo.SlotCount == 1 && l.throwCompact {
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 1, Pos: pos})
		} else {
			for slot := errorInfo.SlotCount - 1; slot >= 0; slot-- {
				l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: l.throwScratchBase + slot, Pos: pos})
			}
			if errorInfo.SlotCount > 1 {
				discard := l.ensureDiscardLocal()
				l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard, Pos: pos})
			}
			l.emitZeroSlots(l.throwSuccessSlots, pos)
			propagated, err := l.emitConvertedThrowFromScratch(errorType, l.throwsType, pos)
			if err != nil {
				return 0, err
			}
			if propagated != l.throwErrorSlots {
				return 0, fmt.Errorf("%s: task_join_i32_typed error slot mismatch", frontend.FormatPos(pos))
			}
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 1, Pos: pos})
		}
		l.emitCleanup(pos)
		l.emit(ir.IRInstr{Kind: ir.IRReturn, Pos: pos})
		l.emitZeroSlots(handleInfo.SlotCount-1, pos)
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: okLabel, Pos: pos})
		if errorInfo.SlotCount > 1 {
			discard := l.ensureDiscardLocal()
			for slot := 0; slot < errorInfo.SlotCount; slot++ {
				l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard, Pos: pos})
			}
		}
		return 1, nil
	}
	l.emit(ir.IRInstr{Kind: ir.IRCall, Name: typedTaskJoinRuntimeSymbol(handleInfo.SlotCount), ArgSlots: handleInfo.SlotCount, RetSlots: handleInfo.SlotCount, Pos: pos})

	okLabel := l.newLabel()
	l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: okLabel, Pos: pos})

	if errorInfo.SlotCount == 1 && l.throwCompact {
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 1, Pos: pos})
	} else {
		for slot := errorInfo.SlotCount - 1; slot >= 0; slot-- {
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: l.throwScratchBase + slot, Pos: pos})
		}
		if errorInfo.SlotCount > 1 {
			discard := l.ensureDiscardLocal()
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard, Pos: pos})
		}
		l.emitZeroSlots(l.throwSuccessSlots, pos)
		propagated, err := l.emitConvertedThrowFromScratch(errorType, l.throwsType, pos)
		if err != nil {
			return 0, err
		}
		if propagated != l.throwErrorSlots {
			return 0, fmt.Errorf("%s: task_join_i32_typed error slot mismatch", frontend.FormatPos(pos))
		}
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 1, Pos: pos})
	}
	l.emitCleanup(pos)
	l.emit(ir.IRInstr{Kind: ir.IRReturn, Pos: pos})
	l.emitZeroSlots(handleInfo.SlotCount-1, pos)
	l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: okLabel, Pos: pos})
	if errorInfo.SlotCount > 1 {
		discard := l.ensureDiscardLocal()
		for slot := 0; slot < errorInfo.SlotCount; slot++ {
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard, Pos: pos})
		}
	}
	return 1, nil
}

func (l *lowerer) lowerTypedTaskJoinForCatch(call *frontend.CallExpr, pos frontend.Position) (int, error) {
	if len(call.TypeArgs) != 1 {
		return 0, fmt.Errorf("%s: task_join_i32_typed expects one explicit error type argument", frontend.FormatPos(call.At))
	}
	errorType := call.TypeArgs[0].Name
	if errorType == "" {
		return 0, fmt.Errorf("%s: task_join_i32_typed missing resolved error type", frontend.FormatPos(call.At))
	}
	if info, ok := l.types[errorType]; !ok || info.Kind != semantics.TypeEnum {
		return 0, fmt.Errorf("%s: typed task error argument must be an enum", frontend.FormatPos(call.TypeArgs[0].At))
	}
	handleType, handleInfo, err := semantics.EnsureTypedTaskHandleType(errorType, l.types)
	if err != nil {
		return 0, fmt.Errorf("%s: %v", frontend.FormatPos(call.TypeArgs[0].At), err)
	}
	if len(call.Args) != 1 {
		return 0, fmt.Errorf("%s: task_join_i32_typed expects 1 argument", frontend.FormatPos(call.At))
	}
	argType, err := l.inferExprType(call.Args[0])
	if err != nil {
		return 0, err
	}
	if argType != handleType {
		return 0, fmt.Errorf("%s: task_join_i32_typed expects a %s handle", frontend.FormatPos(call.Args[0].Pos()), handleType)
	}
	slots, err := l.lowerExpr(call.Args[0])
	if err != nil {
		return 0, err
	}
	if slots != handleInfo.SlotCount {
		return 0, fmt.Errorf("%s: task_join_i32_typed handle slot mismatch", frontend.FormatPos(call.Args[0].Pos()))
	}
	if handleInfo.SlotCount > 4 {
		statusLocal := l.allocScratchSlots(1)
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: typedTaskJoinRuntimeSymbol(handleInfo.SlotCount), ArgSlots: handleInfo.SlotCount, RetSlots: 1, Pos: pos})
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: statusLocal, Pos: pos})
		if err := l.emitLoadTypedTaskResultSlots(handleInfo.SlotCount-1, pos); err != nil {
			return 0, err
		}
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: statusLocal, Pos: pos})
		return handleInfo.SlotCount, nil
	}
	l.emit(ir.IRInstr{Kind: ir.IRCall, Name: typedTaskJoinRuntimeSymbol(handleInfo.SlotCount), ArgSlots: handleInfo.SlotCount, RetSlots: handleInfo.SlotCount, Pos: pos})
	return handleInfo.SlotCount, nil
}

func isTypedTaskJoinCall(name string) bool {
	return name == "core.task_join_i32_typed" || name == "core.task_join_group_i32_typed"
}

func typedTaskJoinRuntimeSymbol(slotCount int) string {
	return fmt.Sprintf("__tetra_task_join_typed_%d", slotCount)
}

func (l *lowerer) emitLoadTypedTaskResultSlots(count int, pos frontend.Position) error {
	if count < 0 || count > 8 {
		return fmt.Errorf("%s: staged typed task slot count %d is out of range", frontend.FormatPos(pos), count)
	}
	for slot := 0; slot < count; slot++ {
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(slot), Pos: pos})
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_result_get", ArgSlots: 1, RetSlots: 1, Pos: pos})
	}
	return nil
}

func (l *lowerer) emitStageTypedTaskStatus(value int32, status int32, slots int, pos frontend.Position) error {
	if slots < 5 || slots > 8 {
		return fmt.Errorf("%s: staged typed task slots out of range: %d", frontend.FormatPos(pos), slots)
	}
	discard := l.ensureDiscardLocal()
	l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(slots), Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_result_begin", ArgSlots: 1, RetSlots: 1, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard, Pos: pos})

	l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: value, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_result_slot", ArgSlots: 2, RetSlots: 1, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard, Pos: pos})
	for slot := 1; slot < slots-1; slot++ {
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(slot), Pos: pos})
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: pos})
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_result_slot", ArgSlots: 2, RetSlots: 1, Pos: pos})
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard, Pos: pos})
	}
	l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(slots - 1), Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: status, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_result_slot", ArgSlots: 2, RetSlots: 1, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard, Pos: pos})
	return nil
}

func (l *lowerer) emitStageTypedTaskFromLocals(valueLocal int, errBase int, slots int, status int32, pos frontend.Position) error {
	if slots < 5 || slots > 8 {
		return fmt.Errorf("%s: staged typed task slots out of range: %d", frontend.FormatPos(pos), slots)
	}
	discard := l.ensureDiscardLocal()
	l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(slots), Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_result_begin", ArgSlots: 1, RetSlots: 1, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard, Pos: pos})

	l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: pos})
	if valueLocal >= 0 {
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: valueLocal, Pos: pos})
	} else {
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: pos})
	}
	l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_result_slot", ArgSlots: 2, RetSlots: 1, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard, Pos: pos})

	errorSlots := slots - 2
	for slot := 0; slot < errorSlots; slot++ {
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(slot + 1), Pos: pos})
		if errBase >= 0 {
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: errBase + slot, Pos: pos})
		} else {
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: pos})
		}
		l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_result_slot", ArgSlots: 2, RetSlots: 1, Pos: pos})
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard, Pos: pos})
	}
	l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(slots - 1), Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: status, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_result_slot", ArgSlots: 2, RetSlots: 1, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard, Pos: pos})
	return nil
}

func (l *lowerer) emitConvertedValueFromScratch(srcType, dstType string, base int, pos frontend.Position) (int, error) {
	srcInfo, ok := l.types[srcType]
	if !ok {
		return 0, fmt.Errorf("%s: unknown type '%s'", frontend.FormatPos(pos), srcType)
	}
	if srcType == dstType || (isThrowIntLike(srcType) && isThrowIntLike(dstType)) {
		for slot := 0; slot < srcInfo.SlotCount; slot++ {
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: base + slot, Pos: pos})
		}
		return srcInfo.SlotCount, nil
	}
	dstInfo, ok := l.types[dstType]
	if ok && dstInfo.Kind == semantics.TypeOptional {
		slots, err := l.emitConvertedValueFromScratch(srcType, dstInfo.ElemType, base, pos)
		if err == nil {
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 1, Pos: pos})
			return slots + 1, nil
		}
	}
	return 0, fmt.Errorf("%s: try error slot mismatch", frontend.FormatPos(pos))
}

func isThrowIntLike(typeName string) bool {
	switch typeName {
	case "i32", "u8", "task.error":
		return true
	default:
		return false
	}
}

func (l *lowerer) pushLoop(continueLabel, breakLabel int) {
	l.loopStack = append(l.loopStack, loopLabels{
		continueLabel: continueLabel,
		breakLabel:    breakLabel,
		cleanupDepth:  len(l.cleanupIslands),
		deferDepth:    len(l.deferFrames),
	})
}

func (l *lowerer) popLoop() {
	l.loopStack = l.loopStack[:len(l.loopStack)-1]
}

func (l *lowerer) currentLoop() (loopLabels, bool) {
	if len(l.loopStack) == 0 {
		return loopLabels{}, false
	}
	return l.loopStack[len(l.loopStack)-1], true
}

func (l *lowerer) lowerBlock(stmts []frontend.Stmt, pos frontend.Position) error {
	frameIndex := len(l.deferFrames)
	l.deferFrames = append(l.deferFrames, deferFrame{})
	for _, stmt := range stmts {
		if err := l.lowerStmt(stmt); err != nil {
			l.deferFrames = l.deferFrames[:frameIndex]
			return err
		}
	}
	if err := l.emitDeferredFrame(frameIndex, pos); err != nil {
		l.deferFrames = l.deferFrames[:frameIndex]
		return err
	}
	l.deferFrames = l.deferFrames[:frameIndex]
	return nil
}

func (l *lowerer) emitDeferredFrame(frameIndex int, pos frontend.Position) error {
	if frameIndex < 0 || frameIndex >= len(l.deferFrames) {
		return nil
	}
	bodies := l.deferFrames[frameIndex].bodies
	for i := len(bodies) - 1; i >= 0; i-- {
		if err := l.lowerBlock(bodies[i], pos); err != nil {
			return err
		}
	}
	return nil
}

func (l *lowerer) emitDeferredFramesSince(start int, pos frontend.Position) error {
	end := len(l.deferFrames) - 1
	for i := end; i >= start; i-- {
		if err := l.emitDeferredFrame(i, pos); err != nil {
			return err
		}
	}
	return nil
}

func (l *lowerer) prepareGlobalStringFieldAccessesForStmt(stmt frontend.Stmt) map[string]frontend.Position {
	prepared := map[string]frontend.Position{}
	var collectExpr func(frontend.Expr)
	collectExpr = func(expr frontend.Expr) {
		switch e := expr.(type) {
		case *frontend.FieldAccessExpr:
			baseName, fields, _, ok := splitFieldPathLower(e)
			if ok && len(fields) > 0 {
				if g, exists := l.globals[baseName]; exists && g.TypeName == "str" && g.HasStringLiteralInit {
					prepared[baseName] = e.At
				}
			}
			collectExpr(e.Base)
		case *frontend.IndexExpr:
			collectExpr(e.Base)
			collectExpr(e.Index)
		case *frontend.BinaryExpr:
			collectExpr(e.Left)
			collectExpr(e.Right)
		case *frontend.UnaryExpr:
			collectExpr(e.X)
		case *frontend.CallExpr:
			for _, arg := range e.Args {
				collectExpr(arg)
			}
		case *frontend.StructLitExpr:
			for _, field := range e.Fields {
				collectExpr(field.Value)
			}
		case *frontend.MatchExpr:
			collectExpr(e.Value)
			for _, c := range e.Cases {
				if c.Pattern != nil {
					collectExpr(c.Pattern)
				}
				if c.Guard != nil {
					collectExpr(c.Guard)
				}
				collectExpr(c.Value)
			}
		case *frontend.CatchExpr:
			collectExpr(e.Call)
			for _, c := range e.Cases {
				if c.Pattern != nil {
					collectExpr(c.Pattern)
				}
				if c.Guard != nil {
					collectExpr(c.Guard)
				}
				collectExpr(c.Value)
			}
		case *frontend.TryExpr:
			collectExpr(e.X)
		case *frontend.AwaitExpr:
			collectExpr(e.X)
		}
	}

	switch s := stmt.(type) {
	case *frontend.PrintStmt:
		collectExpr(s.Value)
	case *frontend.FreeStmt:
		collectExpr(s.Value)
	case *frontend.ReturnStmt:
		collectExpr(s.Value)
	case *frontend.ThrowStmt:
		collectExpr(s.Value)
	case *frontend.IslandStmt:
		collectExpr(s.Size)
	case *frontend.LetStmt:
		collectExpr(s.Value)
	case *frontend.AssignStmt:
		collectExpr(s.Target)
		collectExpr(s.Value)
	case *frontend.IfStmt:
		collectExpr(s.Cond)
	case *frontend.IfLetStmt:
		collectExpr(s.Value)
	case *frontend.WhileStmt:
		collectExpr(s.Cond)
	case *frontend.ForRangeStmt:
		if s.Iterable != nil {
			collectExpr(s.Iterable)
		} else {
			collectExpr(s.Start)
			collectExpr(s.End)
		}
	case *frontend.MatchStmt:
		collectExpr(s.Value)
		for _, c := range s.Cases {
			if c.Pattern != nil {
				collectExpr(c.Pattern)
			}
			if c.Guard != nil {
				collectExpr(c.Guard)
			}
		}
	case *frontend.ExprStmt:
		collectExpr(s.Expr)
	}

	if len(prepared) == 0 {
		return nil
	}
	names := make([]string, 0, len(prepared))
	for name := range prepared {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		l.emitGlobalStringLiteralInitIfNeeded(l.globals[name], prepared[name])
	}
	return prepared
}

func (l *lowerer) lowerStmt(stmt frontend.Stmt) error {
	prepared := l.prepareGlobalStringFieldAccessesForStmt(stmt)
	if len(prepared) == 0 {
		return l.lowerStmtPrepared(stmt)
	}
	old := l.preparedStringFields
	merged := make(map[string]bool, len(old)+len(prepared))
	for name := range old {
		merged[name] = true
	}
	for name := range prepared {
		merged[name] = true
	}
	l.preparedStringFields = merged
	err := l.lowerStmtPrepared(stmt)
	l.preparedStringFields = old
	return err
}

func (l *lowerer) lowerStmtPrepared(stmt frontend.Stmt) error {
	switch s := stmt.(type) {
	case *frontend.PrintStmt:
		slots, err := l.lowerExpr(s.Value)
		if err != nil {
			return err
		}
		if slots != 2 {
			return fmt.Errorf("%s: print expects str or []u8", frontend.FormatPos(s.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRWrite, Pos: s.At})
	case *frontend.FreeStmt:
		slots, err := l.lowerExpr(s.Value)
		if err != nil {
			return err
		}
		if slots != 1 {
			return fmt.Errorf("%s: free expects island (1 slot)", frontend.FormatPos(s.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRIslandFree, Pos: s.At})
	case *frontend.ReturnStmt:
		if l.stagedTaskTarget.SlotCount > 4 {
			valueSlots, err := l.lowerExprAs(s.Value, l.returnType)
			if err != nil {
				return err
			}
			if valueSlots != 1 {
				return fmt.Errorf("%s: staged typed task return expects 1-slot value", frontend.FormatPos(s.At))
			}
			valueLocal := l.allocScratchSlots(1)
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: valueLocal, Pos: s.At})
			if err := l.emitStageTypedTaskFromLocals(valueLocal, -1, l.stagedTaskTarget.SlotCount, 0, s.At); err != nil {
				return err
			}
			if err := l.emitDeferredFramesSince(0, s.At); err != nil {
				return err
			}
			l.emitCleanup(s.At)
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRReturn, Pos: s.At})
			return nil
		}
		slots := 0
		if closure, ok := s.Value.(*frontend.ClosureExpr); ok && l.returnType == "fnptr" {
			if l.returnSlots == semantics.CallableHandleSlotCount {
				slots = l.emitCallableHandleValue(l.closureSymbolName(closure), closure.Captures, closure.At)
			} else {
				slots = l.emitFunctionSymbolValue(l.closureSymbolName(closure), l.closureEnvLocals(closure.Captures), closure.At)
			}
		} else if id, ok := s.Value.(*frontend.IdentExpr); ok && l.returnType == "fnptr" {
			if info, exists := l.locals[id.Name]; exists && info.FunctionValue != "" && len(info.FunctionCaptures) > 0 {
				if l.returnSlots == semantics.CallableHandleSlotCount || info.FunctionHandleValue || len(l.closureEnvLocalsUnbounded(info.FunctionCaptures)) > semantics.FnPtrEnvSlotCount {
					slots = l.emitCallableHandleValue(info.FunctionValue, info.FunctionCaptures, s.At)
				} else {
					slots = l.emitFunctionSymbolValue(info.FunctionValue, l.capturedClosureEnvLocals(info), s.At)
				}
			}
		} else if target, ok := importedFunctionTargetFromExpr(s.Value, l.imports, l.funcs); ok {
			slots = l.emitFunctionSymbolValue(target, nil, s.At)
		}
		if slots == 0 {
			var err error
			slots, err = l.lowerExprAs(s.Value, l.returnType)
			if err != nil {
				return err
			}
		}
		expectedSlots := l.returnSlots
		if l.throwsType != "" {
			expectedSlots = l.throwSuccessSlots
		}
		if slots != expectedSlots {
			return fmt.Errorf("%s: return slot mismatch", frontend.FormatPos(s.At))
		}
		if l.throwsType != "" {
			if !l.throwCompact {
				l.emitZeroSlots(l.throwErrorSlots, s.At)
			}
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: s.At})
		}
		if err := l.emitDeferredFramesSince(0, s.At); err != nil {
			return err
		}
		l.emitCleanup(s.At)
		l.emit(ir.IRInstr{Kind: ir.IRReturn, Pos: s.At})
	case *frontend.ThrowStmt:
		if l.stagedTaskTarget.SlotCount > 4 {
			if l.throwsType == "" {
				return fmt.Errorf("%s: throw is only allowed in throwing functions", frontend.FormatPos(s.At))
			}
			slots, err := l.lowerExprAs(s.Value, l.throwsType)
			if err != nil {
				return err
			}
			if slots != l.throwErrorSlots {
				return fmt.Errorf("%s: throw slot mismatch", frontend.FormatPos(s.At))
			}
			errBase := l.allocScratchSlots(l.throwErrorSlots)
			for slot := l.throwErrorSlots - 1; slot >= 0; slot-- {
				l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: errBase + slot, Pos: s.At})
			}
			if err := l.emitStageTypedTaskFromLocals(-1, errBase, l.stagedTaskTarget.SlotCount, 1, s.At); err != nil {
				return err
			}
			if err := l.emitDeferredFramesSince(0, s.At); err != nil {
				return err
			}
			l.emitCleanup(s.At)
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 1, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRReturn, Pos: s.At})
			return nil
		}
		if l.throwsType == "" {
			return fmt.Errorf("%s: throw is only allowed in throwing functions", frontend.FormatPos(s.At))
		}
		if !l.throwCompact {
			l.emitZeroSlots(l.throwSuccessSlots, s.At)
		}
		slots, err := l.lowerExprAs(s.Value, l.throwsType)
		if err != nil {
			return err
		}
		if slots != l.throwErrorSlots {
			return fmt.Errorf("%s: throw slot mismatch", frontend.FormatPos(s.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 1, Pos: s.At})
		if err := l.emitDeferredFramesSince(0, s.At); err != nil {
			return err
		}
		l.emitCleanup(s.At)
		l.emit(ir.IRInstr{Kind: ir.IRReturn, Pos: s.At})
	case *frontend.DeferStmt:
		if len(l.deferFrames) == 0 {
			return fmt.Errorf("%s: defer outside block", frontend.FormatPos(s.At))
		}
		frameIndex := len(l.deferFrames) - 1
		l.deferFrames[frameIndex].bodies = append(l.deferFrames[frameIndex].bodies, s.Body)
	case *frontend.BreakStmt:
		loop, ok := l.currentLoop()
		if !ok {
			return fmt.Errorf("%s: break outside loop", frontend.FormatPos(s.At))
		}
		if err := l.emitDeferredFramesSince(loop.deferDepth, s.At); err != nil {
			return err
		}
		l.emitCleanupSince(loop.cleanupDepth, s.At)
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: loop.breakLabel, Pos: s.At})
	case *frontend.ContinueStmt:
		loop, ok := l.currentLoop()
		if !ok {
			return fmt.Errorf("%s: continue outside loop", frontend.FormatPos(s.At))
		}
		if err := l.emitDeferredFramesSince(loop.deferDepth, s.At); err != nil {
			return err
		}
		l.emitCleanupSince(loop.cleanupDepth, s.At)
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: loop.continueLabel, Pos: s.At})
	case *frontend.IslandStmt:
		slots, err := l.lowerExpr(s.Size)
		if err != nil {
			return err
		}
		if slots != 1 {
			return fmt.Errorf("%s: island size must be i32", frontend.FormatPos(s.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRIslandNew, Pos: s.At})
		info, ok := l.locals[s.Name]
		if !ok {
			return fmt.Errorf("unknown local '%s'", s.Name)
		}
		if info.SlotCount != 1 {
			return fmt.Errorf("%s: island slot mismatch", frontend.FormatPos(s.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: info.Base, Pos: s.At})
		l.cleanupIslands = append(l.cleanupIslands, info.Base)
		if err := l.lowerBlock(s.Body, s.At); err != nil {
			return err
		}
		l.cleanupIslands = l.cleanupIslands[:len(l.cleanupIslands)-1]
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: info.Base, Pos: s.At})
		l.emit(ir.IRInstr{Kind: ir.IRIslandFree, Pos: s.At})
	case *frontend.LetStmt:
		info, ok := l.locals[s.Name]
		if !ok {
			return fmt.Errorf("unknown local '%s'", s.Name)
		}
		slots := 0
		if info.FunctionTypeValue {
			if _, ok := s.Value.(*frontend.ClosureExpr); ok && info.FunctionValue != "" {
				if info.FunctionHandleValue {
					closure := s.Value.(*frontend.ClosureExpr)
					slots = l.emitCallableHandleValue(info.FunctionValue, closure.Captures, s.At)
				} else {
					slots = l.emitFunctionSymbolValue(info.FunctionValue, l.capturedClosureEnvLocals(info), s.At)
				}
			} else if id, ok := s.Value.(*frontend.IdentExpr); ok && info.FunctionValue != "" {
				if source, ok := l.locals[id.Name]; ok && source.FunctionTypeValue {
					for slot := 0; slot < source.SlotCount; slot++ {
						l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: source.Base + slot, Pos: s.At})
					}
					slots = source.SlotCount
				} else if source, ok := l.locals[id.Name]; ok && !source.FunctionTypeValue && source.FunctionValue != "" && (source.FunctionHandleValue || len(l.closureEnvLocalsUnbounded(source.FunctionCaptures)) > semantics.FnPtrEnvSlotCount) {
					slots = l.emitCallableHandleValue(source.FunctionValue, source.FunctionCaptures, s.At)
				} else if len(info.FunctionCaptures) > 0 {
					slots = l.emitFunctionSymbolValue(info.FunctionValue, l.capturedClosureEnvLocals(info), s.At)
				} else {
					slots = l.emitFunctionSymbolValue(info.FunctionValue, nil, s.At)
				}
			} else if _, ok := functionTypedGlobalFieldTargetFromExpr(s.Value, l.globals); ok && info.FunctionValue != "" {
				slots = l.emitFunctionSymbolValue(info.FunctionValue, nil, s.At)
			}
		} else if len(info.FunctionFields) > 0 {
			if call, ok := s.Value.(*frontend.CallExpr); ok {
				var handled bool
				var err error
				slots, handled, err = l.lowerStructConstructorCall(call, info.FunctionFields)
				if err != nil {
					return err
				}
				if !handled {
					slots = 0
				}
			} else if lit, ok := s.Value.(*frontend.StructLitExpr); ok {
				var err error
				slots, err = l.lowerStructLiteralExpr(lit, info.FunctionFields)
				if err != nil {
					return err
				}
			}
		} else if len(info.EnumPayloadFunctions) > 0 {
			if call, ok := s.Value.(*frontend.CallExpr); ok {
				var handled bool
				var err error
				slots, handled, err = l.lowerEnumCaseConstructorCall(call, info.EnumPayloadFunctions)
				if err != nil {
					return err
				}
				if !handled {
					slots = 0
				}
			}
		}
		if slots == 0 {
			var err error
			slots, err = l.lowerExprAs(s.Value, info.TypeName)
			if err != nil {
				return err
			}
		}
		if slots != info.SlotCount {
			return fmt.Errorf("%s: slot mismatch for '%s'", frontend.FormatPos(s.At), s.Name)
		}
		for i := info.SlotCount - 1; i >= 0; i-- {
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: info.Base + i, Pos: s.At})
		}
		if info.SlotCount == 1 {
			l.rememberRawPtrOffsetAlias(info.Base, s.Value)
		}
	case *frontend.AssignStmt:
		if id, ok := s.Target.(*frontend.IdentExpr); ok {
			if info, ok := l.locals[id.Name]; ok && info.ActorField {
				l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(info.ActorFieldSlot), Pos: s.At})
				slots, err := l.lowerExprAs(s.Value, info.TypeName)
				if err != nil {
					return err
				}
				if slots != 1 {
					return fmt.Errorf("%s: actor state assignment expects single-slot value", frontend.FormatPos(s.At))
				}
				l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_state_store", ArgSlots: 2, RetSlots: 1, Pos: s.At})
				l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: l.ensureDiscardLocal(), Pos: s.At})
				return nil
			}
		}
		if idx, ok := s.Target.(*frontend.IndexExpr); ok {
			elemType, err := l.indexElemType(idx.Base)
			if err != nil {
				return err
			}
			baseSlots, err := l.lowerExpr(idx.Base)
			if err != nil {
				return err
			}
			if baseSlots != 2 {
				return fmt.Errorf("%s: index base slot mismatch", frontend.FormatPos(idx.At))
			}
			idxSlots, err := l.lowerExpr(idx.Index)
			if err != nil {
				return err
			}
			if idxSlots != 1 {
				return fmt.Errorf("%s: index must be i32", frontend.FormatPos(idx.At))
			}
			valSlots, err := l.lowerExpr(s.Value)
			if err != nil {
				return err
			}
			if valSlots != 1 {
				return fmt.Errorf("%s: index assignment expects single-slot value", frontend.FormatPos(s.At))
			}
			targetKind, ok := lowerIndexStoreKind(elemType, l.types)
			if !ok {
				return lowerUnsupportedError(s.At, "unsupported index element type '%s'", elemType)
			}
			l.emit(ir.IRInstr{Kind: targetKind, Pos: s.At})
			return nil
		}
		if id, ok := s.Target.(*frontend.IdentExpr); ok {
			if g, ok := l.globals[id.Name]; ok {
				var slots int
				var err error
				if g.FunctionTypeValue {
					slots, err = l.lowerFunctionTypedLocalAssignmentValue(s.Value, semantics.LocalInfo{
						SlotCount:         gSlotCount(g.TypeName, l.types),
						TypeName:          g.TypeName,
						FunctionTypeValue: true,
					}, s.At)
				} else {
					slots, err = l.lowerExprAs(s.Value, g.TypeName)
				}
				if err != nil {
					return err
				}
				slotCount := gSlotCount(g.TypeName, l.types)
				if slots != slotCount {
					return fmt.Errorf("%s: slot mismatch for assignment", frontend.FormatPos(s.At))
				}
				for i := slotCount - 1; i >= 0; i-- {
					l.emit(ir.IRInstr{Kind: ir.IRStoreGlobal, Local: g.DataIndex + i, Pos: s.At})
				}
				return nil
			}
			if info, ok := l.locals[id.Name]; ok && info.FunctionTypeValue {
				slots, err := l.lowerFunctionTypedLocalAssignmentValue(s.Value, info, s.At)
				if err != nil {
					return err
				}
				if slots != info.SlotCount {
					return fmt.Errorf("%s: slot mismatch for assignment", frontend.FormatPos(s.At))
				}
				for i := info.SlotCount - 1; i >= 0; i-- {
					l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: info.Base + i, Pos: s.At})
				}
				return nil
			}
		} else if targetName := functionTypedFieldNameFromExpr(s.Target); targetName != "" {
			if _, ok, _ := resolveFunctionFieldName(targetName, l.locals); ok {
				target, err := l.resolveLValue(s.Target)
				if err != nil {
					return err
				}
				slots, err := l.lowerFunctionTypedLocalAssignmentValue(s.Value, semantics.LocalInfo{SlotCount: target.SlotCount, TypeName: target.TypeName, FunctionTypeValue: true}, s.At)
				if err != nil {
					return err
				}
				if slots != target.SlotCount {
					return fmt.Errorf("%s: slot mismatch for assignment", frontend.FormatPos(s.At))
				}
				storeKind := ir.IRStoreLocal
				if target.Global {
					storeKind = ir.IRStoreGlobal
				}
				for i := target.SlotCount - 1; i >= 0; i-- {
					l.emit(ir.IRInstr{Kind: storeKind, Local: target.Base + i, Pos: s.At})
				}
				return nil
			}
		}
		target, err := l.resolveLValue(s.Target)
		if err != nil {
			return err
		}
		slots, err := l.lowerExprAs(s.Value, target.TypeName)
		if err != nil {
			return err
		}
		if slots != target.SlotCount {
			return fmt.Errorf("%s: slot mismatch for assignment", frontend.FormatPos(s.At))
		}
		storeKind := ir.IRStoreLocal
		if target.Global {
			storeKind = ir.IRStoreGlobal
		}
		for i := target.SlotCount - 1; i >= 0; i-- {
			l.emit(ir.IRInstr{Kind: storeKind, Local: target.Base + i, Pos: s.At})
		}
		if !target.Global && target.SlotCount == 1 {
			if id, ok := s.Target.(*frontend.IdentExpr); ok {
				if info, ok := l.locals[id.Name]; ok && info.Base == target.Base {
					l.rememberRawPtrOffsetAlias(target.Base, s.Value)
				}
			}
		}
	case *frontend.IfStmt:
		elseLabel := l.newLabel()
		endLabel := -1
		if len(s.Else) > 0 {
			endLabel = l.newLabel()
		}
		slots, err := l.lowerExpr(s.Cond)
		if err != nil {
			return err
		}
		if slots != 1 {
			return fmt.Errorf("%s: condition must be i32", frontend.FormatPos(s.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: elseLabel, Pos: s.At})
		if err := l.lowerBlock(s.Then, s.At); err != nil {
			return err
		}
		if len(s.Else) > 0 {
			l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: endLabel, Pos: s.At})
		}
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: elseLabel, Pos: s.At})
		if len(s.Else) > 0 {
			if err := l.lowerBlock(s.Else, s.At); err != nil {
				return err
			}
			l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel, Pos: s.At})
		}
	case *frontend.IfLetStmt:
		valueInfo, ok := l.locals[s.ValueLocal]
		if !ok {
			return fmt.Errorf("%s: unknown if-let value local", frontend.FormatPos(s.At))
		}
		slots, err := l.lowerExpr(s.Value)
		if err != nil {
			return err
		}
		if slots != valueInfo.SlotCount {
			return fmt.Errorf("%s: if-let value slot mismatch", frontend.FormatPos(s.At))
		}
		for i := valueInfo.SlotCount - 1; i >= 0; i-- {
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: valueInfo.Base + i, Pos: s.At})
		}
		elseLabel := l.newLabel()
		endLabel := -1
		if len(s.Else) > 0 {
			endLabel = l.newLabel()
		}
		if s.Pattern == nil {
			bindInfo, ok := l.locals[s.Name]
			if !ok {
				return fmt.Errorf("%s: unknown if-let local '%s'", frontend.FormatPos(s.At), s.Name)
			}
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: valueInfo.Base + bindInfo.SlotCount, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: elseLabel, Pos: s.At})
			for i := 0; i < bindInfo.SlotCount; i++ {
				l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: valueInfo.Base + i, Pos: s.At})
			}
			for i := bindInfo.SlotCount - 1; i >= 0; i-- {
				l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: bindInfo.Base + i, Pos: s.At})
			}
		} else {
			if err := l.emitIfLetPatternCheck(s.Pattern, valueInfo, elseLabel, s.At); err != nil {
				return err
			}
			if err := l.emitIfLetPatternBindings(s.Pattern, valueInfo); err != nil {
				return err
			}
		}
		if err := l.lowerBlock(s.Then, s.At); err != nil {
			return err
		}
		if len(s.Else) > 0 {
			l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: endLabel, Pos: s.At})
		}
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: elseLabel, Pos: s.At})
		if len(s.Else) > 0 {
			if err := l.lowerBlock(s.Else, s.At); err != nil {
				return err
			}
			l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel, Pos: s.At})
		}
	case *frontend.WhileStmt:
		startLabel := l.newLabel()
		endLabel := l.newLabel()
		l.pushLoop(startLabel, endLabel)
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: startLabel, Pos: s.At})
		slots, err := l.lowerExpr(s.Cond)
		if err != nil {
			l.popLoop()
			return err
		}
		if slots != 1 {
			l.popLoop()
			return fmt.Errorf("%s: condition must be i32", frontend.FormatPos(s.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: endLabel, Pos: s.At})
		if err := l.lowerBlock(s.Body, s.At); err != nil {
			l.popLoop()
			return err
		}
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: startLabel, Pos: s.At})
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel, Pos: s.At})
		l.popLoop()
	case *frontend.ForRangeStmt:
		loopInfo, ok := l.locals[s.Name]
		if !ok {
			return fmt.Errorf("%s: unknown for local '%s'", frontend.FormatPos(s.At), s.Name)
		}
		endInfo, ok := l.locals[s.EndLocal]
		if !ok {
			return fmt.Errorf("%s: unknown for end local", frontend.FormatPos(s.At))
		}
		if s.Iterable != nil {
			iterInfo, ok := l.locals[s.IterableLocal]
			if !ok {
				return fmt.Errorf("%s: unknown for iterable local", frontend.FormatPos(s.At))
			}
			indexInfo, ok := l.locals[s.IndexLocal]
			if !ok {
				return fmt.Errorf("%s: unknown for index local", frontend.FormatPos(s.At))
			}
			iterSlots, err := l.lowerExpr(s.Iterable)
			if err != nil {
				return err
			}
			if iterSlots != iterInfo.SlotCount || iterInfo.SlotCount != 2 {
				return fmt.Errorf("%s: for collection iterable slot mismatch", frontend.FormatPos(s.At))
			}
			for i := iterInfo.SlotCount - 1; i >= 0; i-- {
				l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: iterInfo.Base + i, Pos: s.At})
			}
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: indexInfo.Base, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: iterInfo.Base + 1, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: endInfo.Base, Pos: s.At})
			startLabel := l.newLabel()
			continueLabel := l.newLabel()
			endLabel := l.newLabel()
			l.pushLoop(continueLabel, endLabel)
			l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: startLabel, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: indexInfo.Base, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: endInfo.Base, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRCmpLtI32, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: endLabel, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: iterInfo.Base, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: iterInfo.Base + 1, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: indexInfo.Base, Pos: s.At})
			loadKind, ok := lowerIndexLoadKind(loopInfo.TypeName, l.types)
			if !ok {
				return lowerUnsupportedError(s.At, "unsupported for collection element type '%s'", loopInfo.TypeName)
			}
			l.emit(ir.IRInstr{Kind: loadKind, Pos: s.At})
			if loopInfo.SlotCount != 1 {
				return fmt.Errorf("%s: for collection element slot mismatch", frontend.FormatPos(s.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: loopInfo.Base, Pos: s.At})
			if err := l.lowerBlock(s.Body, s.At); err != nil {
				l.popLoop()
				return err
			}
			l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: continueLabel, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: indexInfo.Base, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 1, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRAddI32, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: indexInfo.Base, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: startLabel, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel, Pos: s.At})
			l.popLoop()
			return nil
		}
		startSlots, err := l.lowerExpr(s.Start)
		if err != nil {
			return err
		}
		if startSlots != 1 || loopInfo.SlotCount != 1 {
			return fmt.Errorf("%s: for range start slot mismatch", frontend.FormatPos(s.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: loopInfo.Base, Pos: s.At})
		endSlots, err := l.lowerExpr(s.End)
		if err != nil {
			return err
		}
		if endSlots != 1 || endInfo.SlotCount != 1 {
			return fmt.Errorf("%s: for range end slot mismatch", frontend.FormatPos(s.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: endInfo.Base, Pos: s.At})
		startLabel := l.newLabel()
		continueLabel := l.newLabel()
		endLabel := l.newLabel()
		l.pushLoop(continueLabel, endLabel)
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: startLabel, Pos: s.At})
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: loopInfo.Base, Pos: s.At})
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: endInfo.Base, Pos: s.At})
		l.emit(ir.IRInstr{Kind: ir.IRCmpLtI32, Pos: s.At})
		l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: endLabel, Pos: s.At})
		if err := l.lowerBlock(s.Body, s.At); err != nil {
			l.popLoop()
			return err
		}
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: continueLabel, Pos: s.At})
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: loopInfo.Base, Pos: s.At})
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 1, Pos: s.At})
		l.emit(ir.IRInstr{Kind: ir.IRAddI32, Pos: s.At})
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: loopInfo.Base, Pos: s.At})
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: startLabel, Pos: s.At})
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel, Pos: s.At})
		l.popLoop()
	case *frontend.MatchStmt:
		info, ok := l.locals[s.ScrutineeLocal]
		if !ok {
			return fmt.Errorf("%s: unknown match scrutinee local", frontend.FormatPos(s.At))
		}
		valueSlots, err := l.lowerExpr(s.Value)
		if err != nil {
			return err
		}
		if valueSlots != info.SlotCount {
			return fmt.Errorf("%s: match value slot mismatch", frontend.FormatPos(s.At))
		}
		for i := info.SlotCount - 1; i >= 0; i-- {
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: info.Base + i, Pos: s.At})
		}
		endLabel := l.newLabel()
		defaultLabel := -1
		caseLabels := make([]int, len(s.Cases))
		guardFailLabels := make([]int, len(s.Cases))
		scrutTypeInfo, scrutTypeOK := l.types[info.TypeName]
		for i, c := range s.Cases {
			guardFailLabels[i] = endLabel
			caseLabels[i] = l.newLabel()
			if c.Default {
				defaultLabel = caseLabels[i]
				continue
			}
			nextLabel := l.newLabel()
			guardFailLabels[i] = nextLabel
			if scrutTypeOK && scrutTypeInfo.Kind == semantics.TypeOptional {
				if _, ok := c.Pattern.(*frontend.SomePatternExpr); ok {
					l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: info.Base + info.SlotCount - 1, Pos: c.At})
					l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: nextLabel, Pos: c.At})
					l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: caseLabels[i], Pos: c.At})
					l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: nextLabel, Pos: c.At})
					continue
				}
				if !isNoneExpr(c.Pattern) {
					return fmt.Errorf("%s: optional match supports only 'none', 'some(name)', and '_' patterns", frontend.FormatPos(c.At))
				}
				l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: info.Base + info.SlotCount - 1, Pos: c.At})
				l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: c.At})
			} else if scrutTypeOK && scrutTypeInfo.Kind == semantics.TypeEnum {
				l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: info.Base, Pos: c.At})
				switch pat := c.Pattern.(type) {
				case *frontend.FieldAccessExpr:
					if pat.EnumType == "" {
						return fmt.Errorf("%s: enum match pattern was not resolved", frontend.FormatPos(c.At))
					}
					l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: pat.EnumOrdinal, Pos: c.At})
				case *frontend.EnumCasePatternExpr:
					if pat.EnumType == "" {
						return fmt.Errorf("%s: enum match pattern was not resolved", frontend.FormatPos(c.At))
					}
					if err := l.validateEnumPatternLayout(pat, info); err != nil {
						return err
					}
					l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: pat.EnumOrdinal, Pos: c.At})
				default:
					return fmt.Errorf("%s: enum match supports enum case patterns and '_'", frontend.FormatPos(c.At))
				}
			} else {
				if info.SlotCount != 1 {
					return fmt.Errorf("%s: match value slot mismatch", frontend.FormatPos(s.At))
				}
				l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: info.Base, Pos: c.At})
				patSlots, err := l.lowerExpr(c.Pattern)
				if err != nil {
					return err
				}
				if patSlots != 1 {
					return fmt.Errorf("%s: match pattern slot mismatch", frontend.FormatPos(c.At))
				}
			}
			l.emit(ir.IRInstr{Kind: ir.IRCmpEqI32, Pos: c.At})
			l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: nextLabel, Pos: c.At})
			l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: caseLabels[i], Pos: c.At})
			l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: nextLabel, Pos: c.At})
		}
		if defaultLabel >= 0 {
			l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: defaultLabel, Pos: s.At})
		} else {
			l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: endLabel, Pos: s.At})
		}
		for i, c := range s.Cases {
			l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: caseLabels[i], Pos: c.At})
			if some, ok := c.Pattern.(*frontend.SomePatternExpr); ok {
				bindInfo, ok := l.locals[some.Name]
				if !ok {
					return fmt.Errorf("%s: unknown some binding '%s'", frontend.FormatPos(some.At), some.Name)
				}
				if bindInfo.SlotCount != info.SlotCount-1 {
					return fmt.Errorf("%s: optional some binding slot mismatch", frontend.FormatPos(some.At))
				}
				for slot := 0; slot < bindInfo.SlotCount; slot++ {
					l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: info.Base + slot, Pos: some.At})
				}
				for slot := bindInfo.SlotCount - 1; slot >= 0; slot-- {
					l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: bindInfo.Base + slot, Pos: some.At})
				}
			}
			if enumPat, ok := c.Pattern.(*frontend.EnumCasePatternExpr); ok {
				if err := l.emitIfLetPatternBindings(enumPat, info); err != nil {
					return err
				}
			}
			if c.Guard != nil {
				slots, err := l.lowerExpr(c.Guard)
				if err != nil {
					return err
				}
				if slots != 1 {
					return fmt.Errorf("%s: match guard must be single-slot", frontend.FormatPos(c.Guard.Pos()))
				}
				l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: guardFailLabels[i], Pos: c.Guard.Pos()})
			}
			if err := l.lowerBlock(c.Body, c.At); err != nil {
				return err
			}
			l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: endLabel, Pos: c.At})
		}
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel, Pos: s.At})
	case *frontend.ExprStmt:
		slots, err := l.lowerExpr(s.Expr)
		if err != nil {
			return err
		}
		discardLocal := l.ensureDiscardLocal()
		for i := 0; i < slots; i++ {
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: discardLocal, Pos: s.At})
		}
	case *frontend.UnsafeStmt:
		return l.lowerBlock(s.Body, s.At)
	default:
		return lowerUnsupportedError(s.Pos(), "unsupported statement kind %T", s)
	}
	return nil
}

func (l *lowerer) ensureDiscardLocal() int {
	if l.discardLocal >= 0 {
		return l.discardLocal
	}
	l.discardLocal = l.localSlots
	l.localSlots++
	return l.discardLocal
}

func (l *lowerer) allocScratchSlots(slots int) int {
	base := l.localSlots
	l.localSlots += slots
	return base
}

func (l *lowerer) lowerMatchExpr(e *frontend.MatchExpr) (int, error) {
	info, ok := l.locals[e.ScrutineeLocal]
	if !ok {
		return 0, fmt.Errorf("%s: unknown match expression scrutinee local", frontend.FormatPos(e.At))
	}
	resultInfo, ok := l.locals[e.ResultLocal]
	if !ok {
		return 0, fmt.Errorf("%s: unknown match expression result local", frontend.FormatPos(e.At))
	}
	valueSlots, err := l.lowerExpr(e.Value)
	if err != nil {
		return 0, err
	}
	if valueSlots != info.SlotCount {
		return 0, fmt.Errorf("%s: match value slot mismatch", frontend.FormatPos(e.At))
	}
	for i := info.SlotCount - 1; i >= 0; i-- {
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: info.Base + i, Pos: e.At})
	}
	endLabel := l.newLabel()
	defaultLabel := -1
	caseLabels := make([]int, len(e.Cases))
	guardFailLabels := make([]int, len(e.Cases))
	scrutTypeInfo, scrutTypeOK := l.types[info.TypeName]
	for i, c := range e.Cases {
		guardFailLabels[i] = endLabel
		caseLabels[i] = l.newLabel()
		if c.Default {
			defaultLabel = caseLabels[i]
			continue
		}
		nextLabel := l.newLabel()
		guardFailLabels[i] = nextLabel
		if scrutTypeOK && scrutTypeInfo.Kind == semantics.TypeOptional {
			if _, ok := c.Pattern.(*frontend.SomePatternExpr); ok {
				l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: info.Base + info.SlotCount - 1, Pos: c.At})
				l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: nextLabel, Pos: c.At})
				l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: caseLabels[i], Pos: c.At})
				l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: nextLabel, Pos: c.At})
				continue
			}
			if !isNoneExpr(c.Pattern) {
				return 0, fmt.Errorf("%s: optional match supports only 'none', 'some(name)', and '_' patterns", frontend.FormatPos(c.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: info.Base + info.SlotCount - 1, Pos: c.At})
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: c.At})
		} else if scrutTypeOK && scrutTypeInfo.Kind == semantics.TypeEnum {
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: info.Base, Pos: c.At})
			switch pat := c.Pattern.(type) {
			case *frontend.FieldAccessExpr:
				if pat.EnumType == "" {
					return 0, fmt.Errorf("%s: enum match pattern was not resolved", frontend.FormatPos(c.At))
				}
				l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: pat.EnumOrdinal, Pos: c.At})
			case *frontend.EnumCasePatternExpr:
				if pat.EnumType == "" {
					return 0, fmt.Errorf("%s: enum match pattern was not resolved", frontend.FormatPos(c.At))
				}
				if err := l.validateEnumPatternLayout(pat, info); err != nil {
					return 0, err
				}
				l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: pat.EnumOrdinal, Pos: c.At})
			default:
				return 0, fmt.Errorf("%s: enum match supports enum case patterns and '_'", frontend.FormatPos(c.At))
			}
		} else {
			if info.SlotCount != 1 {
				return 0, fmt.Errorf("%s: match value slot mismatch", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: info.Base, Pos: c.At})
			patSlots, err := l.lowerExpr(c.Pattern)
			if err != nil {
				return 0, err
			}
			if patSlots != 1 {
				return 0, fmt.Errorf("%s: match pattern slot mismatch", frontend.FormatPos(c.At))
			}
		}
		l.emit(ir.IRInstr{Kind: ir.IRCmpEqI32, Pos: c.At})
		l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: nextLabel, Pos: c.At})
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: caseLabels[i], Pos: c.At})
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: nextLabel, Pos: c.At})
	}
	if defaultLabel >= 0 {
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: defaultLabel, Pos: e.At})
	} else {
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: endLabel, Pos: e.At})
	}
	for i, c := range e.Cases {
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: caseLabels[i], Pos: c.At})
		if some, ok := c.Pattern.(*frontend.SomePatternExpr); ok {
			bindInfo, ok := l.locals[some.Name]
			if !ok {
				return 0, fmt.Errorf("%s: unknown some binding '%s'", frontend.FormatPos(some.At), some.Name)
			}
			if bindInfo.SlotCount != info.SlotCount-1 {
				return 0, fmt.Errorf("%s: optional some binding slot mismatch", frontend.FormatPos(some.At))
			}
			for slot := 0; slot < bindInfo.SlotCount; slot++ {
				l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: info.Base + slot, Pos: some.At})
			}
			for slot := bindInfo.SlotCount - 1; slot >= 0; slot-- {
				l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: bindInfo.Base + slot, Pos: some.At})
			}
		}
		if enumPat, ok := c.Pattern.(*frontend.EnumCasePatternExpr); ok {
			if err := l.emitIfLetPatternBindings(enumPat, info); err != nil {
				return 0, err
			}
		}
		if c.Guard != nil {
			slots, err := l.lowerExpr(c.Guard)
			if err != nil {
				return 0, err
			}
			if slots != 1 {
				return 0, fmt.Errorf("%s: match guard must be single-slot", frontend.FormatPos(c.Guard.Pos()))
			}
			l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: guardFailLabels[i], Pos: c.Guard.Pos()})
		}
		slots, err := l.lowerExprAs(c.Value, e.ResultType)
		if err != nil {
			return 0, err
		}
		if slots != resultInfo.SlotCount {
			return 0, fmt.Errorf("%s: match expression result slot mismatch", frontend.FormatPos(c.At))
		}
		for slot := resultInfo.SlotCount - 1; slot >= 0; slot-- {
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: resultInfo.Base + slot, Pos: c.At})
		}
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: endLabel, Pos: c.At})
	}
	l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel, Pos: e.At})
	for slot := 0; slot < resultInfo.SlotCount; slot++ {
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: resultInfo.Base + slot, Pos: e.At})
	}
	return resultInfo.SlotCount, nil
}

func (l *lowerer) lowerCatchExpr(e *frontend.CatchExpr) (int, error) {
	call, ok := e.Call.(*frontend.CallExpr)
	if !ok {
		return 0, fmt.Errorf("%s: catch expects a throwing function call", frontend.FormatPos(e.At))
	}
	errorInfo, ok := l.locals[e.ErrorLocal]
	if !ok {
		return 0, fmt.Errorf("%s: unknown catch error local", frontend.FormatPos(e.At))
	}
	resultInfo, ok := l.locals[e.ResultLocal]
	if !ok {
		return 0, fmt.Errorf("%s: unknown catch result local", frontend.FormatPos(e.At))
	}
	if builtin, ok := semantics.ResolveBuiltinAlias(call.Name); ok {
		call.Name = builtin
	}
	var callSuccessSlots int
	var callErrorSlots int
	var callCompact bool
	var expectedReturnSlots int
	if isTypedTaskJoinCall(call.Name) {
		if len(call.TypeArgs) != 1 || call.TypeArgs[0].Name == "" {
			return 0, fmt.Errorf("%s: task_join_i32_typed missing resolved error type", frontend.FormatPos(call.At))
		}
		errorInfo, ok := l.types[call.TypeArgs[0].Name]
		if !ok || errorInfo.Kind != semantics.TypeEnum {
			return 0, fmt.Errorf("%s: typed task error argument must be an enum", frontend.FormatPos(call.TypeArgs[0].At))
		}
		_, handleInfo, err := semantics.EnsureTypedTaskHandleType(call.TypeArgs[0].Name, l.types)
		if err != nil {
			return 0, fmt.Errorf("%s: %v", frontend.FormatPos(call.TypeArgs[0].At), err)
		}
		callSuccessSlots = 1
		callErrorSlots = errorInfo.SlotCount
		callCompact = errorInfo.SlotCount == 1
		expectedReturnSlots = handleInfo.SlotCount
	} else {
		sig, ok := l.funcs[call.Name]
		if !ok {
			return 0, fmt.Errorf("%s: unknown function '%s'", frontend.FormatPos(call.At), call.Name)
		}
		if sig.ThrowsType == "" {
			return 0, fmt.Errorf("%s: catch expects a throwing function call", frontend.FormatPos(e.At))
		}
		var err error
		callSuccessSlots, callErrorSlots, callCompact, err = throwingLayout(sig.ReturnType, sig.ThrowsType, l.types)
		if err != nil {
			return 0, err
		}
		expectedReturnSlots = sig.ReturnSlots
	}
	if callSuccessSlots != resultInfo.SlotCount || callErrorSlots != errorInfo.SlotCount {
		return 0, fmt.Errorf("%s: catch slot mismatch", frontend.FormatPos(e.At))
	}
	var slots int
	var err error
	if isTypedTaskJoinCall(call.Name) {
		slots, err = l.lowerTypedTaskJoinForCatch(call, e.At)
	} else {
		slots, err = l.lowerExpr(call)
	}
	if err != nil {
		return 0, err
	}
	if slots != expectedReturnSlots {
		return 0, fmt.Errorf("%s: catch call result slot mismatch", frontend.FormatPos(e.At))
	}

	successLabel := l.newLabel()
	endLabel := l.newLabel()
	l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: successLabel, Pos: e.At})

	if callCompact {
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: errorInfo.Base, Pos: e.At})
	} else {
		for slot := callErrorSlots - 1; slot >= 0; slot-- {
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: errorInfo.Base + slot, Pos: e.At})
		}
		discard := l.ensureDiscardLocal()
		for slot := 0; slot < callSuccessSlots; slot++ {
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard, Pos: e.At})
		}
	}

	defaultLabel := -1
	caseLabels := make([]int, len(e.Cases))
	guardFailLabels := make([]int, len(e.Cases))
	errorTypeInfo, errorTypeOK := l.types[errorInfo.TypeName]
	for i, c := range e.Cases {
		guardFailLabels[i] = endLabel
		caseLabels[i] = l.newLabel()
		if c.Default {
			defaultLabel = caseLabels[i]
			continue
		}
		nextLabel := l.newLabel()
		guardFailLabels[i] = nextLabel
		if errorTypeOK && errorTypeInfo.Kind == semantics.TypeOptional {
			if _, ok := c.Pattern.(*frontend.SomePatternExpr); ok {
				l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: errorInfo.Base + errorInfo.SlotCount - 1, Pos: c.At})
				l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: nextLabel, Pos: c.At})
				l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: caseLabels[i], Pos: c.At})
				l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: nextLabel, Pos: c.At})
				continue
			}
			if !isNoneExpr(c.Pattern) {
				return 0, fmt.Errorf("%s: optional catch supports only 'none', 'some(name)', and '_' patterns", frontend.FormatPos(c.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: errorInfo.Base + errorInfo.SlotCount - 1, Pos: c.At})
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: c.At})
		} else if errorTypeOK && errorTypeInfo.Kind == semantics.TypeEnum {
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: errorInfo.Base, Pos: c.At})
			switch pat := c.Pattern.(type) {
			case *frontend.FieldAccessExpr:
				if pat.EnumType == "" {
					return 0, fmt.Errorf("%s: enum catch pattern was not resolved", frontend.FormatPos(c.At))
				}
				l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: pat.EnumOrdinal, Pos: c.At})
			case *frontend.EnumCasePatternExpr:
				if pat.EnumType == "" {
					return 0, fmt.Errorf("%s: enum catch pattern was not resolved", frontend.FormatPos(c.At))
				}
				if err := l.validateEnumPatternLayout(pat, errorInfo); err != nil {
					return 0, err
				}
				l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: pat.EnumOrdinal, Pos: c.At})
			default:
				return 0, fmt.Errorf("%s: enum catch supports enum case patterns and '_'", frontend.FormatPos(c.At))
			}
		} else {
			if errorInfo.SlotCount != 1 {
				return 0, fmt.Errorf("%s: catch error slot mismatch", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: errorInfo.Base, Pos: c.At})
			patSlots, err := l.lowerExpr(c.Pattern)
			if err != nil {
				return 0, err
			}
			if patSlots != 1 {
				return 0, fmt.Errorf("%s: catch pattern slot mismatch", frontend.FormatPos(c.At))
			}
		}
		l.emit(ir.IRInstr{Kind: ir.IRCmpEqI32, Pos: c.At})
		l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: nextLabel, Pos: c.At})
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: caseLabels[i], Pos: c.At})
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: nextLabel, Pos: c.At})
	}
	if defaultLabel >= 0 {
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: defaultLabel, Pos: e.At})
	} else {
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: endLabel, Pos: e.At})
	}
	for i, c := range e.Cases {
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: caseLabels[i], Pos: c.At})
		if err := l.emitIfLetPatternBindings(c.Pattern, errorInfo); err != nil {
			return 0, err
		}
		if c.Guard != nil {
			slots, err := l.lowerExpr(c.Guard)
			if err != nil {
				return 0, err
			}
			if slots != 1 {
				return 0, fmt.Errorf("%s: catch guard must be single-slot", frontend.FormatPos(c.Guard.Pos()))
			}
			l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: guardFailLabels[i], Pos: c.Guard.Pos()})
		}
		slots, err := l.lowerExprAs(c.Value, e.ResultType)
		if err != nil {
			return 0, err
		}
		if slots != resultInfo.SlotCount {
			return 0, fmt.Errorf("%s: catch expression result slot mismatch", frontend.FormatPos(c.At))
		}
		for slot := resultInfo.SlotCount - 1; slot >= 0; slot-- {
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: resultInfo.Base + slot, Pos: c.At})
		}
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: endLabel, Pos: c.At})
	}

	successEntrySlots := callSuccessSlots
	if !callCompact {
		successEntrySlots += callErrorSlots
	}
	l.emitZeroSlots(successEntrySlots, e.At)
	l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: successLabel, Pos: e.At})
	if !callCompact {
		discard := l.ensureDiscardLocal()
		for slot := 0; slot < callErrorSlots; slot++ {
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard, Pos: e.At})
		}
	}
	for slot := resultInfo.SlotCount - 1; slot >= 0; slot-- {
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: resultInfo.Base + slot, Pos: e.At})
	}
	l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: endLabel, Pos: e.At})

	l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel, Pos: e.At})
	for slot := 0; slot < resultInfo.SlotCount; slot++ {
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: resultInfo.Base + slot, Pos: e.At})
	}
	return resultInfo.SlotCount, nil
}

func (l *lowerer) emitIfLetPatternCheck(pattern frontend.Expr, valueInfo semantics.LocalInfo, elseLabel int, pos frontend.Position) error {
	scrutTypeInfo, scrutTypeOK := l.types[valueInfo.TypeName]
	if scrutTypeOK && scrutTypeInfo.Kind == semantics.TypeOptional {
		if _, ok := pattern.(*frontend.SomePatternExpr); ok {
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: valueInfo.Base + valueInfo.SlotCount - 1, Pos: pos})
			l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: elseLabel, Pos: pos})
			return nil
		}
		if !isNoneExpr(pattern) {
			return fmt.Errorf("%s: optional if let supports only 'none' and 'some(name)' patterns", frontend.FormatPos(pos))
		}
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: valueInfo.Base + valueInfo.SlotCount - 1, Pos: pos})
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: pos})
		l.emit(ir.IRInstr{Kind: ir.IRCmpEqI32, Pos: pos})
		l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: elseLabel, Pos: pos})
		return nil
	}
	if scrutTypeOK && scrutTypeInfo.Kind == semantics.TypeEnum {
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: valueInfo.Base, Pos: pos})
		switch pat := pattern.(type) {
		case *frontend.FieldAccessExpr:
			if pat.EnumType == "" {
				return fmt.Errorf("%s: enum if-let pattern was not resolved", frontend.FormatPos(pos))
			}
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: pat.EnumOrdinal, Pos: pos})
		case *frontend.EnumCasePatternExpr:
			if pat.EnumType == "" {
				return fmt.Errorf("%s: enum if-let pattern was not resolved", frontend.FormatPos(pos))
			}
			if err := l.validateEnumPatternLayout(pat, valueInfo); err != nil {
				return err
			}
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: pat.EnumOrdinal, Pos: pos})
		default:
			return fmt.Errorf("%s: enum if let supports enum case patterns", frontend.FormatPos(pos))
		}
		l.emit(ir.IRInstr{Kind: ir.IRCmpEqI32, Pos: pos})
		l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: elseLabel, Pos: pos})
		return nil
	}
	return fmt.Errorf("%s: if let pattern requires optional or enum value", frontend.FormatPos(pos))
}

func enumPayloadSlotCount(pat *frontend.EnumCasePatternExpr, fallbackBindings map[string]semantics.LocalInfo) (int, error) {
	if pat == nil {
		return 0, nil
	}
	if len(pat.PayloadSlots) > 0 {
		if len(pat.PayloadSlots) != len(pat.Bindings) {
			return 0, fmt.Errorf("%s: enum payload pattern slot metadata mismatch", frontend.FormatPos(pat.At))
		}
		total := 0
		for _, slots := range pat.PayloadSlots {
			if slots <= 0 {
				return 0, fmt.Errorf("%s: enum payload pattern slot metadata mismatch", frontend.FormatPos(pat.At))
			}
			total += slots
		}
		return total, nil
	}
	total := 0
	for _, binding := range pat.Bindings {
		bindInfo, ok := fallbackBindings[binding]
		if !ok {
			return 0, fmt.Errorf("%s: unknown enum payload binding '%s'", frontend.FormatPos(pat.At), binding)
		}
		if bindInfo.SlotCount <= 0 {
			return 0, fmt.Errorf("%s: enum payload binding '%s' slot mismatch", frontend.FormatPos(pat.At), binding)
		}
		total += bindInfo.SlotCount
	}
	return total, nil
}

func (l *lowerer) validateEnumPatternLayout(pattern frontend.Expr, valueInfo semantics.LocalInfo) error {
	enumPat, ok := pattern.(*frontend.EnumCasePatternExpr)
	if !ok {
		return nil
	}
	payloadSlots, err := enumPayloadSlotCount(enumPat, l.locals)
	if err != nil {
		return err
	}
	if payloadSlots > valueInfo.SlotCount-1 {
		return fmt.Errorf("%s: enum payload pattern exceeds value layout", frontend.FormatPos(enumPat.At))
	}
	return nil
}

func (l *lowerer) emitIfLetPatternBindings(pattern frontend.Expr, valueInfo semantics.LocalInfo) error {
	if some, ok := pattern.(*frontend.SomePatternExpr); ok {
		bindInfo, ok := l.locals[some.Name]
		if !ok {
			return fmt.Errorf("%s: unknown some binding '%s'", frontend.FormatPos(some.At), some.Name)
		}
		for slot := 0; slot < bindInfo.SlotCount; slot++ {
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: valueInfo.Base + slot, Pos: some.At})
		}
		for slot := bindInfo.SlotCount - 1; slot >= 0; slot-- {
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: bindInfo.Base + slot, Pos: some.At})
		}
	}
	if enumPat, ok := pattern.(*frontend.EnumCasePatternExpr); ok {
		payloadOffset := 1
		for i, binding := range enumPat.Bindings {
			bindInfo, ok := l.locals[binding]
			if !ok {
				return fmt.Errorf("%s: unknown enum payload binding '%s'", frontend.FormatPos(enumPat.At), binding)
			}
			wantSlots := bindInfo.SlotCount
			if i < len(enumPat.PayloadSlots) {
				wantSlots = enumPat.PayloadSlots[i]
			}
			if bindInfo.SlotCount != wantSlots {
				return fmt.Errorf("%s: enum payload binding '%s' slot mismatch", frontend.FormatPos(enumPat.At), binding)
			}
			for slot := 0; slot < bindInfo.SlotCount; slot++ {
				l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: valueInfo.Base + payloadOffset + slot, Pos: enumPat.At})
			}
			for slot := bindInfo.SlotCount - 1; slot >= 0; slot-- {
				l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: bindInfo.Base + slot, Pos: enumPat.At})
			}
			payloadOffset += wantSlots
		}
	}
	return nil
}

func rawPtrAddCall(expr frontend.Expr) (*frontend.CallExpr, bool) {
	call, ok := expr.(*frontend.CallExpr)
	if !ok {
		return nil, false
	}
	name := call.Name
	if builtin, ok := semantics.ResolveBuiltinAlias(name); ok {
		name = builtin
	}
	if name != "core.ptr_add" {
		return nil, false
	}
	return call, true
}

func (l *lowerer) rawPtrOffsetAliasFromExpr(expr frontend.Expr) (rawPtrOffsetLocal, bool) {
	call, ok := rawPtrAddCall(expr)
	if !ok || len(call.Args) != 3 {
		return rawPtrOffsetLocal{}, false
	}
	base, ok := call.Args[0].(*frontend.IdentExpr)
	if !ok {
		return rawPtrOffsetLocal{}, false
	}
	baseInfo, ok := l.locals[base.Name]
	if !ok || baseInfo.SlotCount != 1 {
		return rawPtrOffsetLocal{}, false
	}
	alias := rawPtrOffsetLocal{BaseLocal: baseInfo.Base, OffsetLocal: -1}
	switch offset := call.Args[1].(type) {
	case *frontend.NumberExpr:
		alias.OffsetImm = offset.Value
		alias.HasOffsetImm = true
	case *frontend.IdentExpr:
		offsetInfo, ok := l.locals[offset.Name]
		if !ok || offsetInfo.SlotCount != 1 {
			return rawPtrOffsetLocal{}, false
		}
		alias.OffsetLocal = offsetInfo.Base
	default:
		return rawPtrOffsetLocal{}, false
	}
	return alias, true
}

func (l *lowerer) rememberRawPtrOffsetAlias(local int, expr frontend.Expr) {
	l.clearRawPtrOffsetAliasesForLocal(local)
	alias, ok := l.rawPtrOffsetAliasFromExpr(expr)
	if !ok {
		delete(l.rawPtrOffsetLocals, local)
		return
	}
	l.rawPtrOffsetLocals[local] = alias
}

func (l *lowerer) clearRawPtrOffsetAliasesForLocal(local int) {
	delete(l.rawPtrOffsetLocals, local)
	for aliasLocal, alias := range l.rawPtrOffsetLocals {
		if alias.BaseLocal == local || (!alias.HasOffsetImm && alias.OffsetLocal == local) {
			delete(l.rawPtrOffsetLocals, aliasLocal)
		}
	}
}

func (l *lowerer) lowerRawOffsetAlias(alias rawPtrOffsetLocal, pos frontend.Position) {
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: alias.BaseLocal, Pos: pos})
	if alias.HasOffsetImm {
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: alias.OffsetImm, Pos: pos})
		return
	}
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: alias.OffsetLocal, Pos: pos})
}

func (l *lowerer) lowerRawOffsetAddress(expr frontend.Expr, pos frontend.Position) (bool, error) {
	if id, ok := expr.(*frontend.IdentExpr); ok {
		if info, ok := l.locals[id.Name]; ok {
			if alias, ok := l.rawPtrOffsetLocals[info.Base]; ok {
				l.lowerRawOffsetAlias(alias, pos)
				return true, nil
			}
		}
	}
	call, ok := rawPtrAddCall(expr)
	if !ok {
		return false, nil
	}
	if len(call.Args) != 3 {
		return true, fmt.Errorf("%s: ptr_add expects 3 arguments", frontend.FormatPos(call.At))
	}
	baseSlots, err := l.lowerExpr(call.Args[0])
	if err != nil {
		return true, err
	}
	if baseSlots != 1 {
		return true, fmt.Errorf("%s: ptr_add expects a 1-slot base pointer", frontend.FormatPos(call.Args[0].Pos()))
	}
	offsetSlots, err := l.lowerExpr(call.Args[1])
	if err != nil {
		return true, err
	}
	if offsetSlots != 1 {
		return true, fmt.Errorf("%s: ptr_add expects a 1-slot offset", frontend.FormatPos(call.Args[1].Pos()))
	}
	memSlots, err := l.lowerExpr(call.Args[2])
	if err != nil {
		return true, err
	}
	if memSlots != 1 {
		return true, fmt.Errorf("%s: ptr_add expects a 1-slot memory capability", frontend.FormatPos(call.Args[2].Pos()))
	}
	discard := l.ensureDiscardLocal()
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard, Pos: pos})
	return true, nil
}

func (l *lowerer) lowerRawOffsetCall(e *frontend.CallExpr) (int, bool, error) {
	switch e.Name {
	case "core.load_i32", "core.load_u8", "core.load_ptr":
		if len(e.Args) != 2 {
			return 0, true, fmt.Errorf("%s: %s expects 2 arguments", frontend.FormatPos(e.At), strings.TrimPrefix(e.Name, "core."))
		}
		ok, err := l.lowerRawOffsetAddress(e.Args[0], e.At)
		if err != nil || !ok {
			return 0, ok, err
		}
		memSlots, err := l.lowerExpr(e.Args[1])
		if err != nil {
			return 0, true, err
		}
		if memSlots != 1 {
			return 0, true, fmt.Errorf("%s: %s expects a 1-slot memory capability", frontend.FormatPos(e.Args[1].Pos()), strings.TrimPrefix(e.Name, "core."))
		}
		switch e.Name {
		case "core.load_i32":
			l.emit(ir.IRInstr{Kind: ir.IRMemReadI32Offset, Pos: e.At})
		case "core.load_u8":
			l.emit(ir.IRInstr{Kind: ir.IRMemReadU8Offset, Pos: e.At})
		default:
			l.emit(ir.IRInstr{Kind: ir.IRMemReadPtrOffset, Pos: e.At})
		}
		return 1, true, nil
	case "core.store_i32", "core.store_u8", "core.store_ptr":
		if len(e.Args) != 3 {
			return 0, true, fmt.Errorf("%s: %s expects 3 arguments", frontend.FormatPos(e.At), strings.TrimPrefix(e.Name, "core."))
		}
		ok, err := l.lowerRawOffsetAddress(e.Args[0], e.At)
		if err != nil || !ok {
			return 0, ok, err
		}
		valueSlots, err := l.lowerExpr(e.Args[1])
		if err != nil {
			return 0, true, err
		}
		if valueSlots != 1 {
			return 0, true, fmt.Errorf("%s: %s expects a 1-slot value", frontend.FormatPos(e.Args[1].Pos()), strings.TrimPrefix(e.Name, "core."))
		}
		memSlots, err := l.lowerExpr(e.Args[2])
		if err != nil {
			return 0, true, err
		}
		if memSlots != 1 {
			return 0, true, fmt.Errorf("%s: %s expects a 1-slot memory capability", frontend.FormatPos(e.Args[2].Pos()), strings.TrimPrefix(e.Name, "core."))
		}
		switch e.Name {
		case "core.store_i32":
			l.emit(ir.IRInstr{Kind: ir.IRMemWriteI32Offset, Pos: e.At})
		case "core.store_u8":
			l.emit(ir.IRInstr{Kind: ir.IRMemWriteU8Offset, Pos: e.At})
		default:
			l.emit(ir.IRInstr{Kind: ir.IRMemWritePtrOffset, Pos: e.At})
		}
		return 1, true, nil
	default:
		return 0, false, nil
	}
}

func (l *lowerer) lowerExpr(expr frontend.Expr) (int, error) {
	switch e := expr.(type) {
	case *frontend.MatchExpr:
		return l.lowerMatchExpr(e)
	case *frontend.CatchExpr:
		return l.lowerCatchExpr(e)
	case *frontend.NumberExpr:
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: e.Value, Pos: e.At})
		return 1, nil
	case *frontend.BoolLitExpr:
		if e.Value {
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 1, Pos: e.At})
		} else {
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: e.At})
		}
		return 1, nil
	case *frontend.NoneLitExpr:
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: e.At})
		return 2, nil
	case *frontend.StringLitExpr:
		l.emit(ir.IRInstr{Kind: ir.IRStrLit, Str: e.Value, Pos: e.At})
		return 2, nil
	case *frontend.IdentExpr:
		info, ok := l.locals[e.Name]
		if !ok {
			if g, ok := l.globals[e.Name]; ok {
				if g.FunctionTypeValue && g.FunctionValue != "" {
					if g.Mutable {
						l.emitGlobalFunctionValueInitIfNeeded(g, e.At)
						slotCount := gSlotCount(g.TypeName, l.types)
						for i := 0; i < slotCount; i++ {
							l.emit(ir.IRInstr{Kind: ir.IRLoadGlobal, Local: g.DataIndex + i, Pos: e.At})
						}
						return slotCount, nil
					}
					return l.emitFunctionSymbolValue(g.FunctionValue, nil, e.At), nil
				}
				if g.TypeName == "str" && g.HasStringLiteralInit {
					l.emitGlobalStringLiteralInitIfNeeded(g, e.At)
				}
				slotCount := gSlotCount(g.TypeName, l.types)
				for i := 0; i < slotCount; i++ {
					l.emit(ir.IRInstr{Kind: ir.IRLoadGlobal, Local: g.DataIndex + i, Pos: e.At})
				}
				return slotCount, nil
			}
			if sig, ok := l.funcs[e.Name]; ok {
				if sig.Generic {
					return 0, fmt.Errorf("%s: generic function symbol '%s' cannot be lowered as a callable value in this MVP", frontend.FormatPos(e.At), e.Name)
				}
				return l.emitFunctionSymbolValue(e.Name, nil, e.At), nil
			}
			if field, ok := l.actorState[e.Name]; ok {
				l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(field.Slot), Pos: e.At})
				l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_state_load", ArgSlots: 1, RetSlots: 1, Pos: e.At})
				return 1, nil
			}
			return 0, fmt.Errorf("%s: unknown local '%s'", frontend.FormatPos(e.At), e.Name)
		}
		if info.ActorField {
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(info.ActorFieldSlot), Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_state_load", ArgSlots: 1, RetSlots: 1, Pos: e.At})
			return 1, nil
		}
		for i := 0; i < info.SlotCount; i++ {
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: info.Base + i, Pos: e.At})
		}
		return info.SlotCount, nil
	case *frontend.FieldAccessExpr:
		if e.EnumType != "" {
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: e.EnumOrdinal, Pos: e.At})
			info, ok := l.types[e.EnumType]
			if !ok {
				return 0, fmt.Errorf("%s: unknown enum type '%s'", frontend.FormatPos(e.At), e.EnumType)
			}
			l.emitZeroSlots(info.SlotCount-1, e.At)
			return info.SlotCount, nil
		}
		target, err := l.resolveLValue(e)
		if err != nil {
			return 0, err
		}
		if target.Global {
			if g, ok := l.globals[target.Name]; ok && g.TypeName == "str" && g.HasStringLiteralInit {
				if !l.preparedStringFields[target.Name] {
					l.emitGlobalStringLiteralInitIfNeeded(g, e.At)
				}
			}
			for i := 0; i < target.SlotCount; i++ {
				l.emit(ir.IRInstr{Kind: ir.IRLoadGlobal, Local: target.Base + i, Pos: e.At})
			}
			return target.SlotCount, nil
		}
		for i := 0; i < target.SlotCount; i++ {
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: target.Base + i, Pos: e.At})
		}
		return target.SlotCount, nil
	case *frontend.IndexExpr:
		elemType, err := l.indexElemType(e.Base)
		if err != nil {
			return 0, err
		}
		baseSlots, err := l.lowerExpr(e.Base)
		if err != nil {
			return 0, err
		}
		if baseSlots != 2 {
			return 0, fmt.Errorf("%s: index base slot mismatch", frontend.FormatPos(e.At))
		}
		idxSlots, err := l.lowerExpr(e.Index)
		if err != nil {
			return 0, err
		}
		if idxSlots != 1 {
			return 0, fmt.Errorf("%s: index must be i32", frontend.FormatPos(e.At))
		}
		loadKind, ok := lowerIndexLoadKind(elemType, l.types)
		if !ok {
			return 0, lowerUnsupportedError(e.At, "unsupported index element type '%s'", elemType)
		}
		l.emit(ir.IRInstr{Kind: loadKind, Pos: e.At})
		return 1, nil
	case *frontend.StructLitExpr:
		return l.lowerStructLiteralExpr(e, nil)
	case *frontend.TryExpr:
		call, ok := e.X.(*frontend.CallExpr)
		if !ok {
			if await, awaitOK := e.X.(*frontend.AwaitExpr); awaitOK {
				call, ok = await.X.(*frontend.CallExpr)
			}
		}
		if !ok {
			return 0, fmt.Errorf("%s: try expects a throwing function call", frontend.FormatPos(e.At))
		}
		if builtin, ok := semantics.ResolveBuiltinAlias(call.Name); ok {
			call.Name = builtin
		}
		var dynamicFunctionValueSig *semantics.FuncSig
		if local, ok := l.locals[call.Name]; ok && local.FunctionTypeValue && local.FunctionThrowsType != "" {
			dynamicFunctionValueSig = &semantics.FuncSig{
				ReturnType: local.FunctionReturnType,
				ThrowsType: local.FunctionThrowsType,
			}
		} else if local, ok := l.locals[call.Name]; ok && local.FunctionTypeValue && local.FunctionValue != "" {
			call.Name = local.FunctionValue
		} else if fieldInfo, _, ok, err := l.functionFieldCallSource(call.Name, call.At); err != nil {
			return 0, err
		} else if ok && fieldInfo.FunctionThrowsType != "" {
			dynamicFunctionValueSig = &semantics.FuncSig{
				ReturnType: fieldInfo.FunctionReturnType,
				ThrowsType: fieldInfo.FunctionThrowsType,
			}
		} else if global, ok := l.globals[call.Name]; ok && global.FunctionTypeValue && global.FunctionThrowsType != "" {
			dynamicFunctionValueSig = &semantics.FuncSig{
				ReturnType: global.FunctionReturnType,
				ThrowsType: global.FunctionThrowsType,
			}
		}
		if isTypedTaskJoinCall(call.Name) {
			return l.lowerTypedTaskJoin(call, e.At)
		}
		var sig semantics.FuncSig
		if dynamicFunctionValueSig != nil {
			sig = *dynamicFunctionValueSig
		} else {
			var ok bool
			sig, ok = l.funcs[call.Name]
			if !ok {
				return 0, fmt.Errorf("%s: unknown function '%s'", frontend.FormatPos(call.At), call.Name)
			}
		}
		if sig.ThrowsType == "" {
			return 0, fmt.Errorf("%s: try expects a throwing function call", frontend.FormatPos(e.At))
		}
		callSuccessSlots, callErrorSlots, callCompact, err := throwingLayout(sig.ReturnType, sig.ThrowsType, l.types)
		if err != nil {
			return 0, err
		}
		expectedReturnSlots := sig.ReturnSlots
		if expectedReturnSlots == 0 {
			expectedReturnSlots = throwingReturnSlotCount(callSuccessSlots, callErrorSlots)
		}
		slots, err := l.lowerExpr(call)
		if err != nil {
			return 0, err
		}
		if slots != expectedReturnSlots {
			return 0, fmt.Errorf("%s: try result slot mismatch", frontend.FormatPos(e.At))
		}
		okLabel := l.newLabel()
		l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: okLabel, Pos: e.At})

		if callCompact {
			if l.throwErrorSlots < 1 {
				return 0, fmt.Errorf("%s: try error slot mismatch", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: l.throwScratchBase, Pos: e.At})
		} else {
			if callErrorSlots > l.throwErrorSlots {
				return 0, fmt.Errorf("%s: try error slot mismatch", frontend.FormatPos(e.At))
			}
			for slot := callErrorSlots - 1; slot >= 0; slot-- {
				l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: l.throwScratchBase + slot, Pos: e.At})
			}
			for slot := 0; slot < callSuccessSlots; slot++ {
				l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: l.throwScratchBase, Pos: e.At})
			}
		}

		propagatedErrorSlots := 0
		if l.throwCompact {
			var convErr error
			propagatedErrorSlots, convErr = l.emitConvertedThrowFromScratch(sig.ThrowsType, l.throwsType, e.At)
			if convErr != nil {
				return 0, convErr
			}
			if propagatedErrorSlots != 1 {
				return 0, fmt.Errorf("%s: try error slot mismatch", frontend.FormatPos(e.At))
			}
		} else {
			l.emitZeroSlots(l.throwSuccessSlots, e.At)
			var convErr error
			propagatedErrorSlots, convErr = l.emitConvertedThrowFromScratch(sig.ThrowsType, l.throwsType, e.At)
			if convErr != nil {
				return 0, convErr
			}
			if propagatedErrorSlots != l.throwErrorSlots {
				return 0, fmt.Errorf("%s: try error slot mismatch", frontend.FormatPos(e.At))
			}
		}
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 1, Pos: e.At})
		l.emitCleanup(e.At)
		l.emit(ir.IRInstr{Kind: ir.IRReturn, Pos: e.At})

		// The x64 emitter tracks stack depth linearly. This unreachable padding
		// mirrors the success-entry stack depth at okLabel.
		successEntrySlots := callSuccessSlots
		if !callCompact {
			successEntrySlots += callErrorSlots
		}
		l.emitZeroSlots(successEntrySlots, e.At)
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: okLabel, Pos: e.At})

		if !callCompact {
			for slot := 0; slot < callErrorSlots; slot++ {
				l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: l.throwScratchBase, Pos: e.At})
			}
		}
		return callSuccessSlots, nil
	case *frontend.AwaitExpr:
		call, ok := e.X.(*frontend.CallExpr)
		if !ok {
			return 0, fmt.Errorf("%s: await expects an async function call", frontend.FormatPos(e.At))
		}
		return l.lowerExpr(call)
	case *frontend.CallExpr:
		if slots, ok, err := l.lowerEnumCaseConstructorCall(e, nil); ok {
			return slots, err
		}
		if slots, ok, err := l.lowerStructConstructorCall(e, nil); ok {
			return slots, err
		}
		if fieldInfo, base, ok, err := l.functionFieldCallSource(e.Name, e.At); err != nil {
			return 0, err
		} else if ok {
			return l.lowerStoredFunctionCall(e, fieldInfo, base)
		}
		if local, ok := l.locals[e.Name]; ok && local.FunctionTypeValue {
			if local.FunctionHandleValue {
				return l.lowerFunctionTypedParamCall(e, local)
			}
			if local.FunctionValue != "" && !local.Mutable {
				return l.lowerStoredFunctionCall(e, semantics.FunctionFieldInfo{
					FunctionValue:          local.FunctionValue,
					FunctionParamTypes:     append([]string(nil), local.FunctionParamTypes...),
					FunctionParamOwnership: append([]string(nil), local.FunctionParamOwnership...),
					FunctionReturnType:     local.FunctionReturnType,
					FunctionThrowsType:     local.FunctionThrowsType,
				}, local.Base)
			}
			return l.lowerFunctionTypedParamCall(e, local)
		}
		if global, ok := l.globals[e.Name]; ok && global.FunctionTypeValue {
			l.emitGlobalFunctionValueInitIfNeeded(global, e.At)
			return l.lowerGlobalStoredFunctionCall(e, global)
		}
		if builtin, ok := semantics.ResolveBuiltinAlias(e.Name); ok {
			e.Name = builtin
		}
		if slots, ok, err := l.lowerRawOffsetCall(e); ok {
			return slots, err
		}
		switch e.Name {
		case "core.spawn":
			if len(e.Args) != 1 {
				return 0, fmt.Errorf("%s: spawn expects 1 argument", frontend.FormatPos(e.At))
			}
			lit, ok := e.Args[0].(*frontend.StringLitExpr)
			if !ok {
				return 0, fmt.Errorf("%s: spawn expects a string literal", frontend.FormatPos(e.At))
			}
			name := string(lit.Value)
			if name == "" {
				return 0, fmt.Errorf("%s: spawn expects a non-empty name", frontend.FormatPos(e.At))
			}
			h := fnv.New32a()
			_, _ = h.Write([]byte(name))
			id := int32(h.Sum32())
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: id, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_spawn", ArgSlots: 1, RetSlots: 1, Pos: e.At})
			return 1, nil
		case "core.spawn_remote":
			if len(e.Args) != 2 {
				return 0, fmt.Errorf("%s: spawn_remote expects 2 arguments", frontend.FormatPos(e.At))
			}
			nodeSlots, err := l.lowerExpr(e.Args[0])
			if err != nil {
				return 0, err
			}
			if nodeSlots != 1 {
				return 0, fmt.Errorf("%s: spawn_remote expects a 1-slot node id", frontend.FormatPos(e.Args[0].Pos()))
			}
			lit, ok := e.Args[1].(*frontend.StringLitExpr)
			if !ok {
				return 0, fmt.Errorf("%s: spawn_remote expects a string literal", frontend.FormatPos(e.At))
			}
			name := string(lit.Value)
			if name == "" {
				return 0, fmt.Errorf("%s: spawn_remote expects a non-empty name", frontend.FormatPos(e.At))
			}
			h := fnv.New32a()
			_, _ = h.Write([]byte(name))
			id := int32(h.Sum32())
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: id, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_spawn_remote", ArgSlots: 2, RetSlots: 1, Pos: e.At})
			return 1, nil
		case "core.task_spawn_i32":
			if len(e.Args) != 1 {
				return 0, fmt.Errorf("%s: task_spawn_i32 expects 1 argument", frontend.FormatPos(e.At))
			}
			lit, ok := e.Args[0].(*frontend.StringLitExpr)
			if !ok {
				return 0, fmt.Errorf("%s: task_spawn_i32 expects a string literal", frontend.FormatPos(e.At))
			}
			name := string(lit.Value)
			if name == "" {
				return 0, fmt.Errorf("%s: task_spawn_i32 expects a non-empty name", frontend.FormatPos(e.At))
			}
			sig, ok := l.funcs[name]
			if !ok {
				return 0, fmt.Errorf("%s: unknown task target '%s'", frontend.FormatPos(e.At), name)
			}
			if sig.ReturnSlots != 1 {
				return 0, fmt.Errorf("%s: task_spawn_i32 target must return 1 slot", frontend.FormatPos(e.At))
			}
			h := fnv.New32a()
			_, _ = h.Write([]byte(name))
			id := int32(h.Sum32())
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: id, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_spawn_i32", ArgSlots: 1, RetSlots: 2, Pos: e.At})
			return 2, nil
		case "core.task_spawn_i32_typed":
			if len(e.TypeArgs) != 1 {
				return 0, fmt.Errorf("%s: task_spawn_i32_typed expects one explicit error type argument", frontend.FormatPos(e.At))
			}
			errorType := e.TypeArgs[0].Name
			if errorType == "" {
				return 0, fmt.Errorf("%s: task_spawn_i32_typed missing resolved error type", frontend.FormatPos(e.At))
			}
			_, handleInfo, err := semantics.EnsureTypedTaskHandleType(errorType, l.types)
			if err != nil {
				return 0, fmt.Errorf("%s: %v", frontend.FormatPos(e.TypeArgs[0].At), err)
			}
			if len(e.Args) != 1 {
				return 0, fmt.Errorf("%s: task_spawn_i32_typed expects 1 argument", frontend.FormatPos(e.At))
			}
			lit, ok := e.Args[0].(*frontend.StringLitExpr)
			if !ok {
				return 0, fmt.Errorf("%s: task_spawn_i32_typed expects a string literal", frontend.FormatPos(e.At))
			}
			name := string(lit.Value)
			if name == "" {
				return 0, fmt.Errorf("%s: task_spawn_i32_typed expects a non-empty name", frontend.FormatPos(e.At))
			}
			sig, ok := l.funcs[name]
			if !ok {
				return 0, fmt.Errorf("%s: unknown task target '%s'", frontend.FormatPos(e.At), name)
			}
			if handleInfo.SlotCount <= 4 {
				if sig.ReturnSlots != handleInfo.SlotCount {
					return 0, fmt.Errorf("%s: task_spawn_i32_typed target return slot mismatch", frontend.FormatPos(e.At))
				}
			} else if sig.ReturnType != "i32" {
				return 0, fmt.Errorf("%s: task_spawn_i32_typed staged mode requires target return type i32", frontend.FormatPos(e.At))
			}
			wrapperName := typedTaskWrapperName(name, errorType)
			h := fnv.New32a()
			_, _ = h.Write([]byte(wrapperName))
			id := int32(h.Sum32())
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: id, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_spawn_i32", ArgSlots: 1, RetSlots: 2, Pos: e.At})
			if handleInfo.SlotCount > 2 {
				statusLocal := l.allocScratchSlots(1)
				handleLocal := l.allocScratchSlots(1)
				l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: statusLocal, Pos: e.At})
				l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: handleLocal, Pos: e.At})
				l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: handleLocal, Pos: e.At})
				l.emitZeroSlots(handleInfo.SlotCount-2, e.At)
				l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: statusLocal, Pos: e.At})
			}
			return handleInfo.SlotCount, nil
		case "core.task_spawn_group_i32_typed":
			if len(e.TypeArgs) != 1 {
				return 0, fmt.Errorf("%s: task_spawn_group_i32_typed expects one explicit error type argument", frontend.FormatPos(e.At))
			}
			errorType := e.TypeArgs[0].Name
			if errorType == "" {
				return 0, fmt.Errorf("%s: task_spawn_group_i32_typed missing resolved error type", frontend.FormatPos(e.At))
			}
			_, handleInfo, err := semantics.EnsureTypedTaskHandleType(errorType, l.types)
			if err != nil {
				return 0, fmt.Errorf("%s: %v", frontend.FormatPos(e.TypeArgs[0].At), err)
			}
			if len(e.Args) != 2 {
				return 0, fmt.Errorf("%s: task_spawn_group_i32_typed expects 2 arguments", frontend.FormatPos(e.At))
			}
			groupSlots, err := l.lowerExpr(e.Args[0])
			if err != nil {
				return 0, err
			}
			if groupSlots != 1 {
				return 0, fmt.Errorf("%s: task_spawn_group_i32_typed expects a 1-slot task.group handle", frontend.FormatPos(e.At))
			}
			groupLocal := l.allocScratchSlots(1)
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: groupLocal, Pos: e.At})
			lit, ok := e.Args[1].(*frontend.StringLitExpr)
			if !ok {
				return 0, fmt.Errorf("%s: task_spawn_group_i32_typed expects a string literal worker name", frontend.FormatPos(e.At))
			}
			name := string(lit.Value)
			if name == "" {
				return 0, fmt.Errorf("%s: task_spawn_group_i32_typed expects a non-empty name", frontend.FormatPos(e.At))
			}
			sig, ok := l.funcs[name]
			if !ok {
				return 0, fmt.Errorf("%s: unknown task target '%s'", frontend.FormatPos(e.At), name)
			}
			if handleInfo.SlotCount <= 4 {
				if sig.ReturnSlots != handleInfo.SlotCount {
					return 0, fmt.Errorf("%s: task_spawn_group_i32_typed target return slot mismatch", frontend.FormatPos(e.At))
				}
			} else if sig.ReturnType != "i32" {
				return 0, fmt.Errorf("%s: task_spawn_group_i32_typed staged mode requires target return type i32", frontend.FormatPos(e.At))
			}

			activeLabel := l.newLabel()
			endLabel := l.newLabel()
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: groupLocal, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRCmpEqI32, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: activeLabel, Pos: e.At})
			l.emitZeroSlots(handleInfo.SlotCount-1, e.At)
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 1, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: endLabel, Pos: e.At})

			l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: activeLabel, Pos: e.At})
			wrapperName := typedTaskWrapperName(name, errorType)
			h := fnv.New32a()
			_, _ = h.Write([]byte(wrapperName))
			id := int32(h.Sum32())
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: groupLocal, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: id, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_spawn_group_i32", ArgSlots: 2, RetSlots: 2, Pos: e.At})
			if handleInfo.SlotCount > 2 {
				statusLocal := l.allocScratchSlots(1)
				handleLocal := l.allocScratchSlots(1)
				l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: statusLocal, Pos: e.At})
				l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: handleLocal, Pos: e.At})
				l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: handleLocal, Pos: e.At})
				l.emitZeroSlots(handleInfo.SlotCount-2, e.At)
				l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: statusLocal, Pos: e.At})
			}
			l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel, Pos: e.At})
			return handleInfo.SlotCount, nil
		case "core.task_spawn_group_i32":
			if len(e.Args) != 2 {
				return 0, fmt.Errorf("%s: task_spawn_group_i32 expects 2 arguments", frontend.FormatPos(e.At))
			}
			groupSlots, err := l.lowerExpr(e.Args[0])
			if err != nil {
				return 0, err
			}
			if groupSlots != 1 {
				return 0, fmt.Errorf("%s: task_spawn_group_i32 expects a 1-slot task.group handle", frontend.FormatPos(e.At))
			}
			groupLocal := l.allocScratchSlots(1)
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: groupLocal, Pos: e.At})
			lit, ok := e.Args[1].(*frontend.StringLitExpr)
			if !ok {
				return 0, fmt.Errorf("%s: task_spawn_group_i32 expects a string literal worker name", frontend.FormatPos(e.At))
			}
			name := string(lit.Value)
			if name == "" {
				return 0, fmt.Errorf("%s: task_spawn_group_i32 expects a non-empty name", frontend.FormatPos(e.At))
			}
			sig, ok := l.funcs[name]
			if !ok {
				return 0, fmt.Errorf("%s: unknown task target '%s'", frontend.FormatPos(e.At), name)
			}
			if sig.ReturnSlots != 1 {
				return 0, fmt.Errorf("%s: task_spawn_group_i32 target must return 1 slot", frontend.FormatPos(e.At))
			}

			activeLabel := l.newLabel()
			endLabel := l.newLabel()
			// group == 0 => canceled handle
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: groupLocal, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRCmpEqI32, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: activeLabel, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 1, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: endLabel, Pos: e.At})

			l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: activeLabel, Pos: e.At})
			h := fnv.New32a()
			_, _ = h.Write([]byte(name))
			id := int32(h.Sum32())
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: groupLocal, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: id, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_spawn_group_i32", ArgSlots: 2, RetSlots: 2, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel, Pos: e.At})
			return 2, nil
		case "core.recv":
			if len(e.Args) != 0 {
				return 0, fmt.Errorf("%s: recv expects 0 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_recv", ArgSlots: 0, RetSlots: 1, Pos: e.At})
			return 1, nil
		case "core.recv_msg":
			if len(e.Args) != 0 {
				return 0, fmt.Errorf("%s: recv_msg expects 0 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_recv_msg", ArgSlots: 0, RetSlots: 2, Pos: e.At})
			return 2, nil
		case "core.recv_typed":
			if len(e.Args) != 0 {
				return 0, fmt.Errorf("%s: recv_typed expects 0 arguments", frontend.FormatPos(e.At))
			}
			if len(e.TypeArgs) != 1 {
				return 0, fmt.Errorf("%s: recv_typed expects one explicit type argument", frontend.FormatPos(e.At))
			}
			msgType := e.TypeArgs[0].Name
			info, ok := l.types[msgType]
			if !ok || info.Kind != semantics.TypeEnum {
				return 0, fmt.Errorf("%s: recv_typed expects an enum type argument", frontend.FormatPos(e.At))
			}
			base := l.allocScratchSlots(info.SlotCount)
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_recv_begin", ArgSlots: 0, RetSlots: 1, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: base, Pos: e.At})
			for slot := 0; slot < info.SlotCount-1; slot++ {
				l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(slot), Pos: e.At})
				l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_recv_slot", ArgSlots: 1, RetSlots: 1, Pos: e.At})
				l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: base + 1 + slot, Pos: e.At})
			}
			for slot := 0; slot < info.SlotCount; slot++ {
				l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: base + slot, Pos: e.At})
			}
			return info.SlotCount, nil
		case "core.send_typed":
			if len(e.Args) != 2 {
				return 0, fmt.Errorf("%s: send_typed expects 2 arguments", frontend.FormatPos(e.At))
			}
			targetSlots, err := l.lowerExpr(e.Args[0])
			if err != nil {
				return 0, err
			}
			if targetSlots != 1 {
				return 0, fmt.Errorf("%s: send_typed expects actor target", frontend.FormatPos(e.Args[0].Pos()))
			}
			targetLocal := l.allocScratchSlots(1)
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: targetLocal, Pos: e.At})
			msgType, err := l.inferExprType(e.Args[1])
			if err != nil {
				return 0, err
			}
			info, ok := l.types[msgType]
			if !ok || info.Kind != semantics.TypeEnum {
				return 0, fmt.Errorf("%s: send_typed expects an enum message", frontend.FormatPos(e.Args[1].Pos()))
			}
			msgBase := l.allocScratchSlots(info.SlotCount)
			msgSlots, err := l.lowerExpr(e.Args[1])
			if err != nil {
				return 0, err
			}
			if msgSlots != info.SlotCount {
				return 0, fmt.Errorf("%s: send_typed message slot mismatch", frontend.FormatPos(e.Args[1].Pos()))
			}
			for slot := info.SlotCount - 1; slot >= 0; slot-- {
				l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: msgBase + slot, Pos: e.At})
			}
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: targetLocal, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: msgBase, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(info.SlotCount - 1), Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_send_begin", ArgSlots: 3, RetSlots: 1, Pos: e.At})
			discard := l.ensureDiscardLocal()
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard, Pos: e.At})
			for slot := 0; slot < info.SlotCount-1; slot++ {
				l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(slot), Pos: e.At})
				l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: msgBase + 1 + slot, Pos: e.At})
				l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_send_slot", ArgSlots: 2, RetSlots: 1, Pos: e.At})
				l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard, Pos: e.At})
			}
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_send_commit", ArgSlots: 0, RetSlots: 1, Pos: e.At})
			return 1, nil
		case "core.self":
			if len(e.Args) != 0 {
				return 0, fmt.Errorf("%s: self expects 0 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_self", ArgSlots: 0, RetSlots: 1, Pos: e.At})
			return 1, nil
		case "core.sender":
			if len(e.Args) != 0 {
				return 0, fmt.Errorf("%s: sender expects 0 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_sender", ArgSlots: 0, RetSlots: 1, Pos: e.At})
			return 1, nil
		case "core.sym_addr":
			if len(e.Args) != 1 {
				return 0, fmt.Errorf("%s: sym_addr expects 1 argument", frontend.FormatPos(e.At))
			}
			lit, ok := e.Args[0].(*frontend.StringLitExpr)
			if !ok {
				return 0, fmt.Errorf("%s: sym_addr expects a string literal", frontend.FormatPos(e.At))
			}
			name := string(lit.Value)
			if name == "" {
				return 0, fmt.Errorf("%s: sym_addr expects a non-empty symbol name", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRSymAddr, Name: name, Pos: e.At})
			return 1, nil
		}
		total := 0
		callSig, hasCallSig := l.funcs[e.Name]
		for i, arg := range e.Args {
			var slots int
			var err error
			if hasCallSig && i < len(callSig.ParamFunctionTypes) && callSig.ParamFunctionTypes[i] {
				slots, err = l.lowerFunctionTypedArgument(arg)
			} else if hasCallSig && i < len(callSig.ParamTypes) {
				slots, err = l.lowerExprAs(arg, callSig.ParamTypes[i])
			} else {
				slots, err = l.lowerExpr(arg)
			}
			if err != nil {
				return 0, err
			}
			total += slots
		}
		switch e.Name {
		case "core.cap_io":
			if total != 0 {
				return 0, fmt.Errorf("%s: cap_io expects 0 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRCapIO, Pos: e.At})
			return 1, nil
		case "core.cap_mem":
			if total != 0 {
				return 0, fmt.Errorf("%s: cap_mem expects 0 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRCapMem, Pos: e.At})
			return 1, nil
		case "core.alloc_bytes":
			if total != 1 {
				return 0, fmt.Errorf("%s: alloc_bytes expects 1 argument", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRAllocBytes, Pos: e.At})
			return 1, nil
		case "core.make_u8":
			if total != 1 {
				return 0, fmt.Errorf("%s: make_u8 expects 1 argument", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRMakeSliceU8, Pos: e.At})
			return 2, nil
		case "core.make_u16":
			if total != 1 {
				return 0, fmt.Errorf("%s: make_u16 expects 1 argument", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRMakeSliceU16, Pos: e.At})
			return 2, nil
		case "core.make_i32":
			if total != 1 {
				return 0, fmt.Errorf("%s: make_i32 expects 1 argument", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRMakeSliceI32, Pos: e.At})
			return 2, nil
		case "core.make_bool":
			if total != 1 {
				return 0, fmt.Errorf("%s: make_bool expects 1 argument", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRMakeSliceI32, Pos: e.At})
			return 2, nil
		case "core.island_new":
			if total != 1 {
				return 0, fmt.Errorf("%s: island_new expects 1 argument", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRIslandNew, Pos: e.At})
			return 1, nil
		case "core.island_make_u8":
			if total != 2 {
				return 0, fmt.Errorf("%s: island_make_u8 expects 2 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRIslandMakeSliceU8, Pos: e.At})
			return 2, nil
		case "core.island_make_u16":
			if total != 2 {
				return 0, fmt.Errorf("%s: island_make_u16 expects 2 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRIslandMakeSliceU16, Pos: e.At})
			return 2, nil
		case "core.island_make_i32":
			if total != 2 {
				return 0, fmt.Errorf("%s: island_make_i32 expects 2 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRIslandMakeSliceI32, Pos: e.At})
			return 2, nil
		case "core.island_make_bool":
			if total != 2 {
				return 0, fmt.Errorf("%s: island_make_bool expects 2 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRIslandMakeSliceI32, Pos: e.At})
			return 2, nil
		case "core.mmio_read_i32":
			if total != 2 {
				return 0, fmt.Errorf("%s: mmio_read_i32 expects 2 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRMmioReadI32, Pos: e.At})
			return 1, nil
		case "core.mmio_write_i32":
			if total != 3 {
				return 0, fmt.Errorf("%s: mmio_write_i32 expects 3 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRMmioWriteI32, Pos: e.At})
			return 1, nil
		case "core.fs_exists":
			if total != 3 {
				return 0, fmt.Errorf("%s: fs_exists expects 3 argument slots", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_fs_exists", ArgSlots: 3, RetSlots: 1, Pos: e.At})
			return 1, nil
		case "core.load_i32":
			if total != 2 {
				return 0, fmt.Errorf("%s: load_i32 expects 2 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRMemReadI32, Pos: e.At})
			return 1, nil
		case "core.store_i32":
			if total != 3 {
				return 0, fmt.Errorf("%s: store_i32 expects 3 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRMemWriteI32, Pos: e.At})
			return 1, nil
		case "core.load_u8":
			if total != 2 {
				return 0, fmt.Errorf("%s: load_u8 expects 2 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRMemReadU8, Pos: e.At})
			return 1, nil
		case "core.store_u8":
			if total != 3 {
				return 0, fmt.Errorf("%s: store_u8 expects 3 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRMemWriteU8, Pos: e.At})
			return 1, nil
		case "core.load_ptr":
			if total != 2 {
				return 0, fmt.Errorf("%s: load_ptr expects 2 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRMemReadPtr, Pos: e.At})
			return 1, nil
		case "core.store_ptr":
			if total != 3 {
				return 0, fmt.Errorf("%s: store_ptr expects 3 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRMemWritePtr, Pos: e.At})
			return 1, nil
		case "core.ptr_add":
			if total != 3 {
				return 0, fmt.Errorf("%s: ptr_add expects 3 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRPtrAdd, Pos: e.At})
			return 1, nil
		case "core.ctx_switch":
			if total != 3 {
				return 0, fmt.Errorf("%s: ctx_switch expects 3 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRCtxSwitch, Pos: e.At})
			return 1, nil
		case "core.consent_token":
			if total != 0 {
				return 0, fmt.Errorf("%s: consent_token expects 0 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: consentTokenRuntimeSentinel, Pos: e.At})
			return 1, nil
		case "core.secret_seal_i32":
			if total != 2 {
				return 0, fmt.Errorf("%s: secret_seal_i32 expects 2 arguments", frontend.FormatPos(e.At))
			}
			// Keep the first argument (secret payload) and consume the token.
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRMulI32, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRAddI32, Pos: e.At})
			return 1, nil
		case "core.secret_unseal_i32":
			if total != 2 {
				return 0, fmt.Errorf("%s: secret_unseal_i32 expects 2 arguments", frontend.FormatPos(e.At))
			}
			// Keep the first argument (sealed payload) and consume the token.
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRMulI32, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRAddI32, Pos: e.At})
			return 1, nil
		case "core.task_group_open":
			if total != 0 {
				return 0, fmt.Errorf("%s: task_group_open expects 0 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_group_open", ArgSlots: 0, RetSlots: 1, Pos: e.At})
			return 1, nil
		case "core.time_now_ms":
			if total != 0 {
				return 0, fmt.Errorf("%s: time_now_ms expects 0 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_time_now_ms", ArgSlots: 0, RetSlots: 1, Pos: e.At})
			return 1, nil
		case "core.sleep_ms":
			if total != 1 {
				return 0, fmt.Errorf("%s: sleep_ms expects 1 argument", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_sleep_ms", ArgSlots: 1, RetSlots: 1, Pos: e.At})
			return 1, nil
		case "core.sleep_until":
			if total != 1 {
				return 0, fmt.Errorf("%s: sleep_until expects 1 argument", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_sleep_until_ms", ArgSlots: 1, RetSlots: 1, Pos: e.At})
			return 1, nil
		case "core.deadline_ms":
			if total != 1 {
				return 0, fmt.Errorf("%s: deadline_ms expects 1 argument", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_deadline_ms", ArgSlots: 1, RetSlots: 1, Pos: e.At})
			return 1, nil
		case "core.timer_ready":
			if total != 1 {
				return 0, fmt.Errorf("%s: timer_ready expects 1 argument", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_timer_ready_ms", ArgSlots: 1, RetSlots: 1, Pos: e.At})
			return 1, nil
		case "core.yield":
			if total != 0 {
				return 0, fmt.Errorf("%s: yield expects 0 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_yield_now", ArgSlots: 0, RetSlots: 1, Pos: e.At})
			return 1, nil
		case "core.task_group_close":
			if total != 1 {
				return 0, fmt.Errorf("%s: task_group_close expects 1 argument", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_group_close", ArgSlots: 1, RetSlots: 1, Pos: e.At})
			return 1, nil
		case "core.task_group_cancel":
			if total != 1 {
				return 0, fmt.Errorf("%s: task_group_cancel expects 1 argument", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_group_cancel", ArgSlots: 1, RetSlots: 1, Pos: e.At})
			return 1, nil
		case "core.task_group_current":
			if total != 0 {
				return 0, fmt.Errorf("%s: task_group_current expects 0 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_group_current", ArgSlots: 0, RetSlots: 1, Pos: e.At})
			return 1, nil
		case "core.task_group_status":
			if total != 1 {
				return 0, fmt.Errorf("%s: task_group_status expects 1 argument", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_group_status", ArgSlots: 1, RetSlots: 1, Pos: e.At})
			return 1, nil
		case "core.task_is_canceled":
			if total != 0 {
				return 0, fmt.Errorf("%s: task_is_canceled expects 0 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_is_canceled", ArgSlots: 0, RetSlots: 1, Pos: e.At})
			return 1, nil
		case "core.task_checkpoint":
			if total != 0 {
				return 0, fmt.Errorf("%s: task_checkpoint expects 0 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_checkpoint", ArgSlots: 0, RetSlots: 1, Pos: e.At})
			return 1, nil
		case "core.task_join_i32":
			if total != 2 {
				return 0, fmt.Errorf("%s: task_join_i32 expects 1 argument", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_join_i32", ArgSlots: 2, RetSlots: 1, Pos: e.At})
			return 1, nil
		case "core.task_join_i32_typed", "core.task_join_group_i32_typed":
			return 0, fmt.Errorf("%s: task_join_i32_typed requires try", frontend.FormatPos(e.At))
		case "core.task_join_result_i32":
			if total != 2 {
				return 0, fmt.Errorf("%s: task_join_result_i32 expects 1 argument", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_join_result_i32", ArgSlots: 2, RetSlots: 2, Pos: e.At})
			return 2, nil
		case "core.task_join_until_i32":
			if total != 3 {
				return 0, fmt.Errorf("%s: task_join_until_i32 expects 2 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_join_until_i32", ArgSlots: 3, RetSlots: 2, Pos: e.At})
			return 2, nil
		case "core.task_poll_i32":
			if total != 2 {
				return 0, fmt.Errorf("%s: task_poll_i32 expects 1 argument", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_poll_i32", ArgSlots: 2, RetSlots: 2, Pos: e.At})
			return 2, nil
		case "core.select2_i32":
			if total != 3 {
				return 0, fmt.Errorf("%s: select2_i32 expects 2 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_task_join_until_i32", ArgSlots: 3, RetSlots: 2, Pos: e.At})
			return 2, nil
		case "core.actor_dispatch":
			if total != 1 {
				return 0, fmt.Errorf("%s: actor_dispatch expects 1 argument", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_dispatch", ArgSlots: 1, RetSlots: 1, Pos: e.At})
			return 1, nil
		case "core.actor_main_entry_id":
			if total != 0 {
				return 0, fmt.Errorf("%s: actor_main_entry_id expects 0 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_main_entry_id", ArgSlots: 0, RetSlots: 1, Pos: e.At})
			return 1, nil
		case "core.actor_node_connect":
			if total != 2 {
				return 0, fmt.Errorf("%s: actor_node_connect expects 2 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_node_connect", ArgSlots: 2, RetSlots: 1, Pos: e.At})
			return 1, nil
		case "core.actor_node_status":
			if total != 1 {
				return 0, fmt.Errorf("%s: actor_node_status expects 1 argument", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_node_status", ArgSlots: 1, RetSlots: 1, Pos: e.At})
			return 1, nil
		case "core.send":
			if total != 2 {
				return 0, fmt.Errorf("%s: send expects 2 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_send", ArgSlots: 2, RetSlots: 1, Pos: e.At})
			return 1, nil
		case "core.send_msg":
			if total != 3 {
				return 0, fmt.Errorf("%s: send_msg expects 3 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_send_msg", ArgSlots: 3, RetSlots: 1, Pos: e.At})
			return 1, nil
		case "core.recv_poll":
			if total != 0 {
				return 0, fmt.Errorf("%s: recv_poll expects 0 arguments", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_recv_poll", ArgSlots: 0, RetSlots: 2, Pos: e.At})
			return 2, nil
		case "core.recv_until":
			if total != 1 {
				return 0, fmt.Errorf("%s: recv_until expects 1 argument", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_recv_until", ArgSlots: 1, RetSlots: 2, Pos: e.At})
			return 2, nil
		case "core.recv_msg_until":
			if total != 1 {
				return 0, fmt.Errorf("%s: recv_msg_until expects 1 argument", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_recv_msg_until", ArgSlots: 1, RetSlots: 3, Pos: e.At})
			return 3, nil
		default:
			sig, ok := l.funcs[e.Name]
			if !ok {
				return 0, fmt.Errorf("%s: unknown function '%s'", frontend.FormatPos(e.At), e.Name)
			}
			l.emit(ir.IRInstr{Kind: ir.IRCall, Name: e.Name, ArgSlots: total, RetSlots: sig.ReturnSlots, Pos: e.At})
			return sig.ReturnSlots, nil
		}
	case *frontend.ClosureExpr:
		l.emit(ir.IRInstr{Kind: ir.IRSymAddr, Name: l.closureSymbolName(e), Pos: e.At})
		return 1, nil
	case *frontend.UnaryExpr:
		slots, err := l.lowerExpr(e.X)
		if err != nil {
			return 0, err
		}
		if slots != 1 {
			return 0, fmt.Errorf("%s: unary operand must be i32", frontend.FormatPos(e.At))
		}
		switch e.Op {
		case frontend.TokenMinus:
			l.emit(ir.IRInstr{Kind: ir.IRNegI32, Pos: e.At})
			return 1, nil
		case frontend.TokenBang:
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRCmpEqI32, Pos: e.At})
			return 1, nil
		default:
			return 0, lowerUnsupportedError(e.At, "unsupported unary operator '%s'", frontend.TokenName(e.Op))
		}
	case *frontend.BinaryExpr:
		if (e.Op == frontend.TokenEqEq || e.Op == frontend.TokenBangEq) && (isNoneExpr(e.Left) || isNoneExpr(e.Right)) {
			var value frontend.Expr
			if isNoneExpr(e.Left) {
				value = e.Right
			} else {
				value = e.Left
			}
			if err := l.lowerOptionalTag(value); err != nil {
				return 0, err
			}
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: e.At})
			if e.Op == frontend.TokenEqEq {
				l.emit(ir.IRInstr{Kind: ir.IRCmpEqI32, Pos: e.At})
			} else {
				l.emit(ir.IRInstr{Kind: ir.IRCmpNeI32, Pos: e.At})
			}
			return 1, nil
		}
		// Short-circuit &&
		if e.Op == frontend.TokenAmpAmp {
			resultLocal := l.allocScratchSlots(1)
			leftSlots, err := l.lowerExpr(e.Left)
			if err != nil {
				return 0, err
			}
			if leftSlots != 1 {
				return 0, fmt.Errorf("%s: && operand must be i32", frontend.FormatPos(e.At))
			}
			falseLabel := l.newLabel()
			endLabel := l.newLabel()
			l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: falseLabel, Pos: e.At})
			rightSlots, err := l.lowerExpr(e.Right)
			if err != nil {
				return 0, err
			}
			if rightSlots != 1 {
				return 0, fmt.Errorf("%s: && operand must be i32", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: resultLocal, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: endLabel, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: falseLabel, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: resultLocal, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: resultLocal, Pos: e.At})
			return 1, nil
		}

		// Short-circuit ||
		if e.Op == frontend.TokenPipePipe {
			resultLocal := l.allocScratchSlots(1)
			leftSlots, err := l.lowerExpr(e.Left)
			if err != nil {
				return 0, err
			}
			if leftSlots != 1 {
				return 0, fmt.Errorf("%s: || operand must be i32", frontend.FormatPos(e.At))
			}
			tryRightLabel := l.newLabel()
			endLabel := l.newLabel()
			l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: tryRightLabel, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 1, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: resultLocal, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: endLabel, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: tryRightLabel, Pos: e.At})
			rightSlots, err := l.lowerExpr(e.Right)
			if err != nil {
				return 0, err
			}
			if rightSlots != 1 {
				return 0, fmt.Errorf("%s: || operand must be i32", frontend.FormatPos(e.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: resultLocal, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel, Pos: e.At})
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: resultLocal, Pos: e.At})
			return 1, nil
		}

		leftSlots, err := l.lowerExpr(e.Left)
		if err != nil {
			return 0, err
		}
		rightSlots, err := l.lowerExpr(e.Right)
		if err != nil {
			return 0, err
		}
		if leftSlots != 1 || rightSlots != 1 {
			return 0, fmt.Errorf("%s: binary operands must be i32", frontend.FormatPos(e.At))
		}
		switch e.Op {
		case frontend.TokenPlus:
			l.emit(ir.IRInstr{Kind: ir.IRAddI32, Pos: e.At})
		case frontend.TokenMinus:
			l.emit(ir.IRInstr{Kind: ir.IRSubI32, Pos: e.At})
		case frontend.TokenStar:
			l.emit(ir.IRInstr{Kind: ir.IRMulI32, Pos: e.At})
		case frontend.TokenSlash:
			l.emit(ir.IRInstr{Kind: ir.IRDivI32, Pos: e.At})
		case frontend.TokenPercent:
			l.emit(ir.IRInstr{Kind: ir.IRModI32, Pos: e.At})
		case frontend.TokenEqEq:
			l.emit(ir.IRInstr{Kind: ir.IRCmpEqI32, Pos: e.At})
		case frontend.TokenBangEq:
			l.emit(ir.IRInstr{Kind: ir.IRCmpNeI32, Pos: e.At})
		case frontend.TokenLess:
			l.emit(ir.IRInstr{Kind: ir.IRCmpLtI32, Pos: e.At})
		case frontend.TokenLessEq:
			l.emit(ir.IRInstr{Kind: ir.IRCmpLeI32, Pos: e.At})
		case frontend.TokenGreater:
			l.emit(ir.IRInstr{Kind: ir.IRCmpGtI32, Pos: e.At})
		case frontend.TokenGreaterEq:
			l.emit(ir.IRInstr{Kind: ir.IRCmpGeI32, Pos: e.At})
		default:
			return 0, lowerUnsupportedError(e.At, "unsupported binary operator '%s'", frontend.TokenName(e.Op))
		}
		return 1, nil
	default:
		return 0, lowerUnsupportedError(expr.Pos(), "unsupported expression kind %T", expr)
	}
}

func (l *lowerer) closureSymbolName(closure *frontend.ClosureExpr) string {
	if closure == nil || closure.Name == "" {
		return ""
	}
	if _, ok := l.funcs[closure.Name]; ok {
		return closure.Name
	}
	if l.module != "" {
		qualified := l.module + "." + closure.Name
		if _, ok := l.funcs[qualified]; ok {
			return qualified
		}
	}
	return closure.Name
}

func (l *lowerer) lowerExprAs(expr frontend.Expr, expectedType string) (int, error) {
	if expectedType == "ptr" {
		if closure, ok := expr.(*frontend.ClosureExpr); ok {
			l.emit(ir.IRInstr{Kind: ir.IRSymAddr, Name: l.closureSymbolName(closure), Pos: closure.At})
			return 1, nil
		}
	}
	expectedInfo, ok := l.types[expectedType]
	if !ok || expectedInfo.Kind != semantics.TypeOptional {
		return l.lowerExpr(expr)
	}
	actualType, err := l.inferExprType(expr)
	if err != nil {
		return 0, err
	}
	if actualType == expectedType {
		return l.lowerExpr(expr)
	}
	if actualType == "none" {
		l.emitZeroSlots(expectedInfo.SlotCount-1, expr.Pos())
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: expr.Pos()})
		return expectedInfo.SlotCount, nil
	}
	if !l.optionalPayloadSlotCompatible(expectedInfo.ElemType, actualType) {
		return l.lowerExpr(expr)
	}
	slots, err := l.lowerExprAs(expr, expectedInfo.ElemType)
	if err != nil {
		return 0, err
	}
	if slots != expectedInfo.SlotCount-1 {
		return 0, fmt.Errorf("%s: optional payload slot mismatch", frontend.FormatPos(expr.Pos()))
	}
	l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 1, Pos: expr.Pos()})
	return expectedInfo.SlotCount, nil
}

func (l *lowerer) optionalPayloadSlotCompatible(expected, actual string) bool {
	if expected == actual {
		return true
	}
	if lowerInt32LikeType(expected) && lowerInt32LikeType(actual) {
		return true
	}
	if expectedInfo, ok := l.types[expected]; ok && expectedInfo.Kind == semantics.TypeOptional {
		return l.optionalPayloadSlotCompatible(expectedInfo.ElemType, actual)
	}
	return false
}

func lowerInt32LikeType(typeName string) bool {
	switch typeName {
	case "i32", "u8", "u16", "task.error":
		return true
	default:
		return false
	}
}

func gSlotCount(typeName string, types map[string]*semantics.TypeInfo) int {
	if info, ok := types[typeName]; ok {
		return info.SlotCount
	}
	return 1
}

func (l *lowerer) emitGlobalStringLiteralInitIfNeeded(g semantics.GlobalInfo, pos frontend.Position) {
	if g.TypeName != "str" || !g.HasStringLiteralInit {
		return
	}
	readyLabel := l.newLabel()
	l.emit(ir.IRInstr{Kind: ir.IRLoadGlobal, Local: g.DataIndex, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRCmpEqI32, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: readyLabel, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRStrLit, Str: g.StringLiteralInit, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRStoreGlobal, Local: g.DataIndex + 1, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRStoreGlobal, Local: g.DataIndex, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: readyLabel, Pos: pos})
}

func (l *lowerer) emitGlobalFunctionValueInitIfNeeded(g semantics.GlobalInfo, pos frontend.Position) {
	if !g.FunctionTypeValue || !g.Mutable || g.FunctionValue == "" {
		return
	}
	readyLabel := l.newLabel()
	l.emit(ir.IRInstr{Kind: ir.IRLoadGlobal, Local: g.DataIndex, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRCmpEqI32, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: readyLabel, Pos: pos})
	slots := l.emitFunctionSymbolValue(g.FunctionValue, nil, pos)
	for i := slots - 1; i >= 0; i-- {
		l.emit(ir.IRInstr{Kind: ir.IRStoreGlobal, Local: g.DataIndex + i, Pos: pos})
	}
	l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: readyLabel, Pos: pos})
}

type lvalueInfo struct {
	Base      int
	SlotCount int
	TypeName  string
	Name      string
	Global    bool
}

func (l *lowerer) resolveLValue(expr frontend.Expr) (lvalueInfo, error) {
	baseName, fields, pos, ok := splitFieldPathLower(expr)
	if !ok {
		return lvalueInfo{}, fmt.Errorf("%s: invalid assignment target", frontend.FormatPos(pos))
	}
	info, ok := l.locals[baseName]
	if !ok {
		if g, ok := l.globals[baseName]; ok {
			targetType, slotCount, offset, err := resolveFieldChainLower(g.TypeName, g.DataIndex, fields, l.types, pos)
			if err != nil {
				return lvalueInfo{}, err
			}
			if _, ok := l.types[targetType]; !ok {
				return lvalueInfo{}, fmt.Errorf("%s: unknown type '%s'", frontend.FormatPos(pos), targetType)
			}
			return lvalueInfo{Base: offset, SlotCount: slotCount, TypeName: targetType, Name: baseName, Global: true}, nil
		}
		return lvalueInfo{}, fmt.Errorf("%s: unknown local '%s'", frontend.FormatPos(pos), baseName)
	}
	targetType, slotCount, offset, err := resolveFieldChainLower(info.TypeName, info.Base, fields, l.types, pos)
	if err != nil {
		return lvalueInfo{}, err
	}
	if _, ok := l.types[targetType]; !ok {
		return lvalueInfo{}, fmt.Errorf("%s: unknown type '%s'", frontend.FormatPos(pos), targetType)
	}
	return lvalueInfo{Base: offset, SlotCount: slotCount, TypeName: targetType, Name: baseName}, nil
}

func splitFieldPathLower(expr frontend.Expr) (string, []string, frontend.Position, bool) {
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		return e.Name, nil, e.At, true
	case *frontend.FieldAccessExpr:
		baseName, fields, pos, ok := splitFieldPathLower(e.Base)
		if !ok {
			return "", nil, pos, false
		}
		fields = append(fields, e.Field)
		return baseName, fields, e.At, true
	default:
		return "", nil, expr.Pos(), false
	}
}

func resolveFieldChainLower(typeName string, baseOffset int, fields []string, types map[string]*semantics.TypeInfo, pos frontend.Position) (string, int, int, error) {
	offset := baseOffset
	current := typeName
	for _, field := range fields {
		info, ok := types[current]
		if !ok {
			return "", 0, 0, fmt.Errorf("%s: unknown type '%s'", frontend.FormatPos(pos), current)
		}
		if info.Kind != semantics.TypeStruct && info.Kind != semantics.TypeSlice && info.Kind != semantics.TypeArray && info.Kind != semantics.TypeStr {
			return "", 0, 0, fmt.Errorf("%s: '%s' is not a struct", frontend.FormatPos(pos), current)
		}
		fieldInfo, ok := info.FieldMap[field]
		if !ok {
			return "", 0, 0, fmt.Errorf("%s: unknown field '%s'", frontend.FormatPos(pos), field)
		}
		offset += fieldInfo.Offset
		current = fieldInfo.TypeName
	}
	info, ok := types[current]
	if !ok {
		return "", 0, 0, fmt.Errorf("%s: unknown type '%s'", frontend.FormatPos(pos), current)
	}
	return current, info.SlotCount, offset, nil
}

func isNoneExpr(expr frontend.Expr) bool {
	_, ok := expr.(*frontend.NoneLitExpr)
	return ok
}

func (l *lowerer) lowerOptionalTag(expr frontend.Expr) error {
	if isNoneExpr(expr) {
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: expr.Pos()})
		return nil
	}
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		info, ok := l.locals[e.Name]
		if !ok {
			return fmt.Errorf("%s: optional comparison to none requires a stored optional value", frontend.FormatPos(e.At))
		}
		typeInfo, ok := l.types[info.TypeName]
		if !ok || typeInfo.Kind != semantics.TypeOptional {
			return fmt.Errorf("%s: optional comparison to none requires optional value", frontend.FormatPos(e.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: info.Base + typeInfo.SlotCount - 1, Pos: e.At})
		return nil
	case *frontend.FieldAccessExpr:
		target, err := l.resolveLValue(e)
		if err != nil {
			return err
		}
		tname, err := l.inferExprType(e)
		if err != nil {
			return err
		}
		typeInfo, ok := l.types[tname]
		if !ok || typeInfo.Kind != semantics.TypeOptional {
			return fmt.Errorf("%s: optional comparison to none requires optional value", frontend.FormatPos(e.At))
		}
		kind := ir.IRLoadLocal
		if target.Global {
			kind = ir.IRLoadGlobal
		}
		l.emit(ir.IRInstr{Kind: kind, Local: target.Base + typeInfo.SlotCount - 1, Pos: e.At})
		return nil
	default:
		return fmt.Errorf("%s: optional comparison to none requires a stored optional value", frontend.FormatPos(expr.Pos()))
	}
}

func (l *lowerer) indexElemType(base frontend.Expr) (string, error) {
	baseType, err := l.inferExprType(base)
	if err != nil {
		return "", err
	}
	info, ok := l.types[baseType]
	if !ok {
		return "", fmt.Errorf("unknown type '%s'", baseType)
	}
	switch info.Kind {
	case semantics.TypeStr:
		return "u8", nil
	case semantics.TypeSlice:
		return info.ElemType, nil
	case semantics.TypeArray:
		return info.ElemType, nil
	default:
		return "", fmt.Errorf("%s: cannot index '%s'", frontend.FormatPos(base.Pos()), baseType)
	}
}

func lowerIndexLoadKind(elemType string, types map[string]*semantics.TypeInfo) (ir.IRInstrKind, bool) {
	switch elemType {
	case "i32":
		return ir.IRIndexLoadI32, true
	case "bool":
		return ir.IRIndexLoadI32, true
	case "u8":
		return ir.IRIndexLoadU8, true
	case "u16":
		return ir.IRIndexLoadU16, true
	}
	info, ok := types[elemType]
	if !ok {
		return 0, false
	}
	if info.Kind == semantics.TypeStruct && info.SlotCount == 1 {
		return ir.IRIndexLoadI32, true
	}
	return 0, false
}

func lowerIndexStoreKind(elemType string, types map[string]*semantics.TypeInfo) (ir.IRInstrKind, bool) {
	switch elemType {
	case "i32":
		return ir.IRIndexStoreI32, true
	case "bool":
		return ir.IRIndexStoreI32, true
	case "u8":
		return ir.IRIndexStoreU8, true
	case "u16":
		return ir.IRIndexStoreU16, true
	}
	info, ok := types[elemType]
	if !ok {
		return 0, false
	}
	if info.Kind == semantics.TypeStruct && info.SlotCount == 1 {
		return ir.IRIndexStoreI32, true
	}
	return 0, false
}

func (l *lowerer) inferExprType(expr frontend.Expr) (string, error) {
	switch e := expr.(type) {
	case *frontend.NumberExpr:
		return "i32", nil
	case *frontend.BoolLitExpr:
		return "bool", nil
	case *frontend.NoneLitExpr:
		return "none", nil
	case *frontend.StringLitExpr:
		return "str", nil
	case *frontend.IdentExpr:
		info, ok := l.locals[e.Name]
		if !ok {
			if g, ok := l.globals[e.Name]; ok {
				return g.TypeName, nil
			}
			if field, ok := l.actorState[e.Name]; ok {
				return field.TypeName, nil
			}
			return "", fmt.Errorf("%s: unknown local '%s'", frontend.FormatPos(e.At), e.Name)
		}
		if info.ActorField {
			return info.TypeName, nil
		}
		return info.TypeName, nil
	case *frontend.FieldAccessExpr:
		if e.EnumType != "" {
			return e.EnumType, nil
		}
		_, targetType, err := semantics.ResolveFieldAccessType(e, l.locals, l.globals, l.types)
		if err != nil {
			return "", err
		}
		return targetType, nil
	case *frontend.IndexExpr:
		elem, err := l.indexElemType(e.Base)
		if err != nil {
			return "", err
		}
		return elem, nil
	case *frontend.StructLitExpr:
		return e.Type.Name, nil
	case *frontend.CallExpr:
		if typeName, _, ok := l.resolveEnumCaseConstructor(e); ok {
			return typeName, nil
		}
		if tname, ok, err := l.inferStructConstructorCallType(e); ok {
			return tname, err
		}
		if local, ok := l.locals[e.Name]; ok && local.FunctionTypeValue && local.FunctionValue == "" {
			return local.FunctionReturnType, nil
		}
		if builtin, ok := semantics.ResolveBuiltinAlias(e.Name); ok {
			e.Name = builtin
		}
		if e.Name == "core.recv_typed" {
			if len(e.TypeArgs) != 1 {
				return "", fmt.Errorf("%s: recv_typed expects one explicit type argument", frontend.FormatPos(e.At))
			}
			return e.TypeArgs[0].Name, nil
		}
		if e.Name == "core.send_typed" {
			return "i32", nil
		}
		if e.Name == "core.task_spawn_i32_typed" || e.Name == "core.task_spawn_group_i32_typed" {
			if len(e.TypeArgs) != 1 || e.TypeArgs[0].Name == "" {
				return "", fmt.Errorf("%s: task_spawn_i32_typed missing resolved error type", frontend.FormatPos(e.At))
			}
			return semantics.TypedTaskHandleTypeName(e.TypeArgs[0].Name, l.types), nil
		}
		if isTypedTaskJoinCall(e.Name) {
			return "i32", nil
		}
		sig, ok := l.funcs[e.Name]
		if !ok {
			return "", fmt.Errorf("%s: unknown function '%s'", frontend.FormatPos(e.At), e.Name)
		}
		return sig.ReturnType, nil
	case *frontend.ClosureExpr:
		return "ptr", nil
	case *frontend.TryExpr:
		call, ok := e.X.(*frontend.CallExpr)
		if !ok {
			if await, awaitOK := e.X.(*frontend.AwaitExpr); awaitOK {
				call, ok = await.X.(*frontend.CallExpr)
			}
		}
		if !ok {
			return "", fmt.Errorf("%s: try expects a throwing function call", frontend.FormatPos(e.At))
		}
		if builtin, ok := semantics.ResolveBuiltinAlias(call.Name); ok {
			call.Name = builtin
		}
		sig, ok := l.funcs[call.Name]
		if !ok {
			return "", fmt.Errorf("%s: unknown function '%s'", frontend.FormatPos(call.At), call.Name)
		}
		return sig.ReturnType, nil
	case *frontend.CatchExpr:
		return e.ResultType, nil
	case *frontend.AwaitExpr:
		call, ok := e.X.(*frontend.CallExpr)
		if !ok {
			return "", fmt.Errorf("%s: await expects an async function call", frontend.FormatPos(e.At))
		}
		if builtin, ok := semantics.ResolveBuiltinAlias(call.Name); ok {
			call.Name = builtin
		}
		sig, ok := l.funcs[call.Name]
		if !ok {
			return "", fmt.Errorf("%s: unknown function '%s'", frontend.FormatPos(call.At), call.Name)
		}
		return sig.ReturnType, nil
	case *frontend.UnaryExpr:
		if e.Op == frontend.TokenBang {
			return "bool", nil
		}
		return "i32", nil
	case *frontend.BinaryExpr:
		return "i32", nil
	default:
		return "", lowerUnsupportedError(expr.Pos(), "unsupported expression kind %T", expr)
	}
}

func (l *lowerer) lowerStructConstructorCall(e *frontend.CallExpr, functionFields map[string]semantics.FunctionFieldInfo) (int, bool, error) {
	if len(e.Args) == 0 || len(e.ArgLabels) != len(e.Args) {
		return 0, false, nil
	}
	for _, label := range e.ArgLabels {
		if label == "" {
			return 0, false, nil
		}
	}

	info, ok := l.types[e.Name]
	if !ok || info.Kind != semantics.TypeStruct {
		return 0, false, nil
	}
	if len(e.Args) != len(info.Fields) {
		return 0, true, fmt.Errorf("%s: wrong field count for '%s'", frontend.FormatPos(e.At), e.Name)
	}

	argByLabel := make(map[string]frontend.Expr, len(e.Args))
	for i, label := range e.ArgLabels {
		if _, exists := argByLabel[label]; exists {
			return 0, true, fmt.Errorf("%s: duplicate field '%s'", frontend.FormatPos(e.Args[i].Pos()), label)
		}
		argByLabel[label] = e.Args[i]
	}
	for label, expr := range argByLabel {
		if _, ok := info.FieldMap[label]; !ok {
			return 0, true, fmt.Errorf("%s: unknown field '%s'", frontend.FormatPos(expr.Pos()), label)
		}
	}

	total := 0
	for _, field := range info.Fields {
		expr, ok := argByLabel[field.Name]
		if !ok {
			return 0, true, fmt.Errorf("%s: missing field '%s'", frontend.FormatPos(e.At), field.Name)
		}
		slots := 0
		if field.FunctionTypeValue {
			if closure, ok := expr.(*frontend.ClosureExpr); ok {
				if fieldInfo, ok := functionFields[field.Name]; ok && fieldInfo.FunctionHandleValue {
					slots = l.emitCallableHandleValue(fieldInfo.FunctionValue, fieldInfo.FunctionCaptures, closure.At)
					l.emitZeroSlots(field.SlotCount-slots, closure.At)
					slots = field.SlotCount
				} else if envLocals := l.closureEnvLocalsUnbounded(closure.Captures); len(envLocals) > semantics.FnPtrEnvSlotCount {
					slots = l.emitCallableHandleValue(l.closureSymbolName(closure), closure.Captures, closure.At)
					l.emitZeroSlots(field.SlotCount-slots, closure.At)
					slots = field.SlotCount
				} else {
					slots = l.emitFunctionSymbolValue(l.closureSymbolName(closure), l.closureEnvLocals(closure.Captures), closure.At)
				}
			} else if id, ok := expr.(*frontend.IdentExpr); ok {
				if source, ok := l.locals[id.Name]; ok && !source.FunctionTypeValue && source.FunctionValue != "" {
					slots = l.emitFunctionSymbolValue(source.FunctionValue, l.capturedClosureEnvLocals(source), expr.Pos())
				}
			} else if call, ok := expr.(*frontend.CallExpr); ok {
				if fieldInfo, ok := functionFields[field.Name]; ok && fieldInfo.FunctionHandleValue {
					var err error
					slots, err = l.lowerExpr(call)
					if err != nil {
						return 0, true, err
					}
					if slots < field.SlotCount {
						l.emitZeroSlots(field.SlotCount-slots, call.Pos())
						slots = field.SlotCount
					}
				}
			} else if target, ok := functionFieldTargetFromExpr(expr, l.locals); ok {
				slots = l.emitFunctionSymbolValue(target, nil, expr.Pos())
			} else if target, ok := functionTypedGlobalFieldTargetFromExpr(expr, l.globals); ok {
				slots = l.emitFunctionSymbolValue(target, nil, expr.Pos())
			} else if target, ok := importedFunctionTargetFromExpr(expr, l.imports, l.funcs); ok {
				slots = l.emitFunctionSymbolValue(target, nil, expr.Pos())
			}
		}
		if slots == 0 {
			var err error
			if field.FunctionTypeValue {
				slots, err = l.lowerExprAs(expr, field.TypeName)
			} else {
				slots, err = l.lowerExpr(expr)
			}
			if err != nil {
				return 0, true, err
			}
		}
		if slots != field.SlotCount {
			return 0, true, fmt.Errorf("%s: slot mismatch for field '%s'", frontend.FormatPos(expr.Pos()), field.Name)
		}
		total += slots
	}
	return total, true, nil
}

func (l *lowerer) lowerStructLiteralExpr(e *frontend.StructLitExpr, functionFields map[string]semantics.FunctionFieldInfo) (int, error) {
	info, ok := l.types[e.Type.Name]
	if !ok {
		return 0, fmt.Errorf("%s: unknown type '%s'", frontend.FormatPos(e.At), e.Type.Name)
	}
	fieldMap := make(map[string]frontend.Expr, len(e.Fields))
	for _, field := range e.Fields {
		fieldMap[field.Name] = field.Value
	}
	total := 0
	for _, field := range info.Fields {
		expr, ok := fieldMap[field.Name]
		if !ok {
			return 0, fmt.Errorf("%s: missing field '%s'", frontend.FormatPos(e.At), field.Name)
		}
		slots := 0
		if field.FunctionTypeValue {
			if closure, ok := expr.(*frontend.ClosureExpr); ok {
				if fieldInfo, ok := functionFields[field.Name]; ok && fieldInfo.FunctionHandleValue {
					slots = l.emitCallableHandleValue(fieldInfo.FunctionValue, fieldInfo.FunctionCaptures, closure.At)
					l.emitZeroSlots(field.SlotCount-slots, closure.At)
					slots = field.SlotCount
				} else if envLocals := l.closureEnvLocalsUnbounded(closure.Captures); len(envLocals) > semantics.FnPtrEnvSlotCount {
					slots = l.emitCallableHandleValue(l.closureSymbolName(closure), closure.Captures, closure.At)
					l.emitZeroSlots(field.SlotCount-slots, closure.At)
					slots = field.SlotCount
				} else {
					slots = l.emitFunctionSymbolValue(l.closureSymbolName(closure), l.closureEnvLocals(closure.Captures), closure.At)
				}
			} else if id, ok := expr.(*frontend.IdentExpr); ok {
				if source, ok := l.locals[id.Name]; ok && source.FunctionTypeValue {
					for slot := 0; slot < source.SlotCount; slot++ {
						l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: source.Base + slot, Pos: expr.Pos()})
					}
					slots = source.SlotCount
				} else if source, ok := l.locals[id.Name]; ok && !source.FunctionTypeValue && source.FunctionValue != "" {
					slots = l.emitFunctionSymbolValue(source.FunctionValue, l.capturedClosureEnvLocals(source), expr.Pos())
				} else if _, ok := l.funcs[id.Name]; ok {
					slots = l.emitFunctionSymbolValue(id.Name, nil, expr.Pos())
				}
			} else if call, ok := expr.(*frontend.CallExpr); ok {
				if fieldInfo, ok := functionFields[field.Name]; ok && fieldInfo.FunctionHandleValue {
					var err error
					slots, err = l.lowerExpr(call)
					if err != nil {
						return 0, err
					}
					if slots < field.SlotCount {
						l.emitZeroSlots(field.SlotCount-slots, call.Pos())
						slots = field.SlotCount
					}
				}
			} else if target, ok := functionFieldTargetFromExpr(expr, l.locals); ok {
				slots = l.emitFunctionSymbolValue(target, nil, expr.Pos())
			} else if target, ok := functionTypedGlobalFieldTargetFromExpr(expr, l.globals); ok {
				slots = l.emitFunctionSymbolValue(target, nil, expr.Pos())
			} else if target, ok := importedFunctionTargetFromExpr(expr, l.imports, l.funcs); ok {
				slots = l.emitFunctionSymbolValue(target, nil, expr.Pos())
			}
		}
		if slots == 0 {
			var err error
			if field.FunctionTypeValue {
				slots, err = l.lowerExprAs(expr, field.TypeName)
			} else {
				slots, err = l.lowerExpr(expr)
			}
			if err != nil {
				return 0, err
			}
		}
		if slots != field.SlotCount {
			return 0, fmt.Errorf("%s: slot mismatch for field '%s'", frontend.FormatPos(e.At), field.Name)
		}
		total += slots
	}
	return total, nil
}

func (l *lowerer) lowerEnumCaseConstructorCall(e *frontend.CallExpr, enumPayloadFunctions map[string]semantics.FunctionFieldInfo) (int, bool, error) {
	typeName, caseInfo, ok := l.resolveEnumCaseConstructor(e)
	if !ok {
		return 0, false, nil
	}
	info, ok := l.types[typeName]
	if !ok || info.Kind != semantics.TypeEnum {
		return 0, true, fmt.Errorf("%s: unknown enum type '%s'", frontend.FormatPos(e.At), typeName)
	}
	if len(e.Args) != len(caseInfo.PayloadTypes) {
		return 0, true, fmt.Errorf("%s: enum case '%s.%s' expects %d payload argument(s), got %d", frontend.FormatPos(e.At), typeName, caseInfo.Name, len(caseInfo.PayloadTypes), len(e.Args))
	}
	l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: caseInfo.Ordinal, Pos: e.At})
	payloadSlots := 0
	for i, arg := range e.Args {
		slots := 0
		if i < len(caseInfo.PayloadFunctionTypes) && caseInfo.PayloadFunctionTypes[i] {
			if closure, ok := arg.(*frontend.ClosureExpr); ok {
				if payloadInfo, ok := enumPayloadFunctions[enumPayloadTargetKey(caseInfo.Ordinal, i)]; ok && payloadInfo.FunctionHandleValue {
					slots = l.emitCallableHandleValue(payloadInfo.FunctionValue, payloadInfo.FunctionCaptures, closure.At)
					l.emitZeroSlots(caseInfo.PayloadSlots[i]-slots, closure.At)
					slots = caseInfo.PayloadSlots[i]
				} else if envLocals := l.closureEnvLocalsUnbounded(closure.Captures); len(envLocals) > semantics.FnPtrEnvSlotCount {
					slots = l.emitCallableHandleValue(l.closureSymbolName(closure), closure.Captures, closure.At)
					l.emitZeroSlots(caseInfo.PayloadSlots[i]-slots, closure.At)
					slots = caseInfo.PayloadSlots[i]
				} else {
					slots = l.emitFunctionSymbolValue(l.closureSymbolName(closure), l.closureEnvLocals(closure.Captures), closure.At)
				}
			} else if id, ok := arg.(*frontend.IdentExpr); ok {
				if source, ok := l.locals[id.Name]; ok && !source.FunctionTypeValue && source.FunctionValue != "" {
					slots = l.emitFunctionSymbolValue(source.FunctionValue, l.capturedClosureEnvLocals(source), arg.Pos())
				}
			} else if call, ok := arg.(*frontend.CallExpr); ok {
				if payloadInfo, ok := enumPayloadFunctions[enumPayloadTargetKey(caseInfo.Ordinal, i)]; ok && payloadInfo.FunctionHandleValue {
					var err error
					slots, err = l.lowerExpr(call)
					if err != nil {
						return 0, true, err
					}
					if slots < caseInfo.PayloadSlots[i] {
						l.emitZeroSlots(caseInfo.PayloadSlots[i]-slots, call.Pos())
						slots = caseInfo.PayloadSlots[i]
					}
				}
			} else if target, ok := functionFieldTargetFromExpr(arg, l.locals); ok {
				slots = l.emitFunctionSymbolValue(target, nil, arg.Pos())
			} else if target, ok := functionTypedGlobalFieldTargetFromExpr(arg, l.globals); ok {
				slots = l.emitFunctionSymbolValue(target, nil, arg.Pos())
			} else if target, ok := importedFunctionTargetFromExpr(arg, l.imports, l.funcs); ok {
				slots = l.emitFunctionSymbolValue(target, nil, arg.Pos())
			}
		}
		if slots == 0 {
			var err error
			if i < len(caseInfo.PayloadFunctionTypes) && caseInfo.PayloadFunctionTypes[i] {
				slots, err = l.lowerExprAs(arg, caseInfo.PayloadTypes[i])
			} else {
				slots, err = l.lowerExpr(arg)
			}
			if err != nil {
				return 0, true, err
			}
		}
		want := caseInfo.PayloadSlots[i]
		if slots != want {
			return 0, true, fmt.Errorf("%s: enum case '%s.%s' payload %d slot mismatch", frontend.FormatPos(arg.Pos()), typeName, caseInfo.Name, i+1)
		}
		payloadSlots += slots
	}
	padding := info.SlotCount - 1 - payloadSlots
	if padding < 0 {
		return 0, true, fmt.Errorf("%s: enum case '%s.%s' payload layout exceeds enum layout", frontend.FormatPos(e.At), typeName, caseInfo.Name)
	}
	l.emitZeroSlots(padding, e.At)
	return info.SlotCount, true, nil
}

func (l *lowerer) resolveEnumCaseConstructor(e *frontend.CallExpr) (string, semantics.EnumCaseInfo, bool) {
	if e.ResolvedType != "" {
		parts := strings.Split(e.Name, ".")
		if len(parts) >= 2 {
			caseName := parts[len(parts)-1]
			if info, ok := l.types[e.ResolvedType]; ok && info.Kind == semantics.TypeEnum {
				if caseInfo, ok := info.CaseMap[caseName]; ok {
					return e.ResolvedType, caseInfo, true
				}
			}
		}
	}
	parts := strings.Split(e.Name, ".")
	if len(parts) < 2 {
		return "", semantics.EnumCaseInfo{}, false
	}
	typeName := strings.Join(parts[:len(parts)-1], ".")
	caseName := parts[len(parts)-1]
	info, ok := l.types[typeName]
	if !ok || info.Kind != semantics.TypeEnum {
		if altName, altInfo, found := findUniqueEnumByShortNameInLower(typeName, l.types); found {
			typeName = altName
			info = altInfo
		} else {
			return "", semantics.EnumCaseInfo{}, false
		}
	}
	caseInfo, ok := info.CaseMap[caseName]
	if !ok {
		return "", semantics.EnumCaseInfo{}, false
	}
	return typeName, caseInfo, true
}

func findUniqueEnumByShortNameInLower(shortName string, types map[string]*semantics.TypeInfo) (string, *semantics.TypeInfo, bool) {
	var foundName string
	var foundInfo *semantics.TypeInfo
	for name, info := range types {
		if info == nil || info.Kind != semantics.TypeEnum {
			continue
		}
		if name != shortName && !strings.HasSuffix(name, "."+shortName) {
			continue
		}
		if foundInfo != nil && foundName != name {
			return "", nil, false
		}
		foundName = name
		foundInfo = info
	}
	return foundName, foundInfo, foundInfo != nil
}

func (l *lowerer) inferStructConstructorCallType(e *frontend.CallExpr) (string, bool, error) {
	if len(e.Args) == 0 || len(e.ArgLabels) != len(e.Args) {
		return "", false, nil
	}
	for _, label := range e.ArgLabels {
		if label == "" {
			return "", false, nil
		}
	}
	info, ok := l.types[e.Name]
	if !ok || info.Kind != semantics.TypeStruct {
		return "", false, nil
	}
	return e.Name, true, nil
}
