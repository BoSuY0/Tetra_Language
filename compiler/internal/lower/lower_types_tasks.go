package lower

import (
	"fmt"
	"tetra_language/compiler/internal/allocplan"
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	lowertasks "tetra_language/compiler/internal/lower/tasks"
	"tetra_language/compiler/internal/semantics"
)

type lowerer struct {
	instrs                     []ir.IRInstr
	locals                     map[string]semantics.LocalInfo
	actorState                 map[string]semantics.ActorStateField
	globals                    map[string]semantics.GlobalInfo
	types                      map[string]*semantics.TypeInfo
	funcs                      map[string]semantics.FuncSig
	imports                    map[string]string
	module                     string
	localSlots                 int
	returnType                 string
	throwsType                 string
	returnSlots                int
	abiReturnSlots             int
	inoutReturnLocals          []inoutReturnLocal
	throwSuccessSlots          int
	throwErrorSlots            int
	throwCompact               bool
	throwScratchBase           int
	policyFailLabel            int
	budgetEnabled              bool
	budgetLocal                int
	discardLocal               int
	budgetScratchBase          int
	budgetScratchSlots         int
	stagedTaskTarget           typedTaskStagedTarget
	callableParamTargets       map[string][]string
	allocationPlan             map[string]allocplan.Allocation
	stackAllocationLowering    bool
	functionTempRegionLowering bool
	functionTempRegionEntered  bool
	scalarSlices               map[string]scalarSliceLocal
	rawPtrOffsetLocals         map[int]rawPtrOffsetLocal
	preparedStringFields       map[string]bool
	zeroLocals                 map[string]bool
	constIntLocals             map[string]int64
	lenBoundLocals             map[string]string
	externalSliceLocals        map[string]bool
	invalidSliceLocals         map[string]bool
	whileRangeProofs           []whileRangeProof
	stackHeight                int
	nextLabel                  int
	cleanupIslands             []int
	deferFrames                []deferFrame
	loopStack                  []loopLabels
}

type scalarSliceLocal struct {
	elemType    string
	length      int64
	elementBase int
}

type whileRangeProof struct {
	indexName string
	baseName  string
	proofID   string
	active    bool
}

type rawPtrOffsetLocal struct {
	BaseLocal    int
	OffsetLocal  int
	OffsetImm    int32
	HasOffsetImm bool
}

type typedTaskWrapper = lowertasks.Wrapper

type typedTaskStagedTarget = lowertasks.StagedTarget

type inoutReturnLocal struct {
	Base      int
	SlotCount int
}

type inoutWriteback struct {
	Base      int
	SlotCount int
	Global    bool
}

func collectInoutReturnLocals(fn semantics.CheckedFunc) ([]inoutReturnLocal, error) {
	locals := make([]inoutReturnLocal, 0)
	for _, param := range fn.Decl.Params {
		if param.Ownership != "inout" {
			continue
		}
		info, ok := fn.Locals[param.Name]
		if !ok {
			return nil, fmt.Errorf("%s: inout parameter '%s' is missing lowering local metadata", frontend.FormatPos(param.At), param.Name)
		}
		locals = append(locals, inoutReturnLocal{Base: info.Base, SlotCount: info.SlotCount})
	}
	return locals, nil
}

func inoutReturnSlotCount(locals []inoutReturnLocal) int {
	total := 0
	for _, local := range locals {
		total += local.SlotCount
	}
	return total
}

func inoutWritebackSlotCount(writebacks []inoutWriteback) int {
	total := 0
	for _, writeback := range writebacks {
		total += writeback.SlotCount
	}
	return total
}

func typedTaskWrapperName(target, errorType string) string {
	return lowertasks.WrapperName(target, errorType)
}

func typedActorMessageTagBase(typeName string) int32 {
	return lowertasks.ActorMessageTagBase(typeName)
}

func collectTypedTaskWrappers(checked *semantics.CheckedProgram, module string) []typedTaskWrapper {
	return lowertasks.CollectWrappers(checked, module)
}

func collectStagedTypedTaskTargets(wrappers []typedTaskWrapper) map[string]typedTaskStagedTarget {
	return lowertasks.CollectStagedTargets(wrappers)
}

func lowerTypedTaskWrapper(wrapper typedTaskWrapper) (ir.IRFunc, error) {
	return lowertasks.LowerWrapper(wrapper, lowerUnsupportedError)
}

func lowerCallExprWithName(call *frontend.CallExpr, name string) *frontend.CallExpr {
	return lowertasks.CallExprWithName(call, name)
}

func lowerCallExprWithBuiltinAlias(call *frontend.CallExpr) *frontend.CallExpr {
	return lowertasks.CallExprWithBuiltinAlias(call)
}

// throwingLayout computes the slot layout for typed-error returns. The compact
// path is only valid when both success and error payloads fit in one slot.
func throwingLayout(returnType, throwsType string, types map[string]*semantics.TypeInfo) (int, int, bool, error) {
	return lowertasks.ThrowingLayout(returnType, throwsType, types)
}

func throwingReturnSlotCount(successSlots, errorSlots int) int {
	return lowertasks.ThrowingReturnSlotCount(successSlots, errorSlots)
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

func (l *lowerer) emitInoutReturnSlots(pos frontend.Position) {
	for _, local := range l.inoutReturnLocals {
		for slot := 0; slot < local.SlotCount; slot++ {
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: local.Base + slot, Pos: pos})
		}
	}
}

func (l *lowerer) collectInoutWritebacks(args []frontend.Expr, ownership []string) ([]inoutWriteback, error) {
	if len(ownership) == 0 {
		return nil, nil
	}
	writebacks := make([]inoutWriteback, 0)
	for i, owner := range ownership {
		if owner != "inout" {
			continue
		}
		if i >= len(args) {
			break
		}
		target, err := l.resolveLValue(args[i])
		if err != nil {
			return nil, err
		}
		if !target.Global && target.Base < 0 {
			return nil, fmt.Errorf("%s: inout writeback target cannot be lowered", frontend.FormatPos(args[i].Pos()))
		}
		writebacks = append(writebacks, inoutWriteback{
			Base:      target.Base,
			SlotCount: target.SlotCount,
			Global:    target.Global,
		})
	}
	return writebacks, nil
}

func (l *lowerer) emitInoutWritebacks(writebacks []inoutWriteback, pos frontend.Position) {
	for i := len(writebacks) - 1; i >= 0; i-- {
		writeback := writebacks[i]
		storeKind := ir.IRStoreLocal
		if writeback.Global {
			storeKind = ir.IRStoreGlobal
		}
		for slot := writeback.SlotCount - 1; slot >= 0; slot-- {
			l.emit(ir.IRInstr{Kind: storeKind, Local: writeback.Base + slot, Pos: pos})
		}
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
		l.emitZeroSlotsRaw(l.abiReturnSlots, pos)
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
	slots, err := l.lowerTypedTaskJoinHandleArg(call.Args[0], handleType, handleInfo)
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
		l.emitFunctionTempRegionReset(pos)
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
	l.emitFunctionTempRegionReset(pos)
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
	slots, err := l.lowerTypedTaskJoinHandleArg(call.Args[0], handleType, handleInfo)
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
	return lowertasks.IsTypedTaskJoinCall(name)
}

func typedTaskJoinRuntimeSymbol(slotCount int) string {
	return lowertasks.TypedTaskJoinRuntimeSymbol(slotCount)
}

func (l *lowerer) lowerTypedTaskJoinHandleArg(expr frontend.Expr, handleType string, handleInfo *semantics.TypeInfo) (int, error) {
	argType, err := l.inferExprType(expr)
	if err != nil {
		return 0, err
	}
	if argType != handleType && !semantics.TypedTaskHandleTypesCompatible(handleType, argType) {
		return 0, fmt.Errorf("%s: task_join_i32_typed expects a %s handle", frontend.FormatPos(expr.Pos()), handleType)
	}
	slots, err := l.lowerExpr(expr)
	if err != nil {
		return 0, err
	}
	if slots == handleInfo.SlotCount {
		return slots, nil
	}
	if argType == "task.i32" && semantics.IsTypedTaskHandleTypeName(handleType) && slots == 2 {
		return l.expandPublicTypedTaskHandleSlots(expr.Pos(), handleInfo.SlotCount), nil
	}
	return slots, nil
}

func (l *lowerer) expandPublicTypedTaskHandleSlots(pos frontend.Position, targetSlots int) int {
	if targetSlots <= 2 {
		return 2
	}
	statusLocal := l.allocScratchSlots(1)
	handleLocal := l.allocScratchSlots(1)
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: statusLocal, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: handleLocal, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: handleLocal, Pos: pos})
	l.emitZeroSlots(targetSlots-2, pos)
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: statusLocal, Pos: pos})
	return targetSlots
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
	return lowertasks.IsThrowIntLike(typeName)
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
