package machine

import (
	"fmt"
	"sort"
	"strings"
)

type Program struct {
	Functions []Function `json:"functions,omitempty"`
}

type Function struct {
	Name   string  `json:"name"`
	Target string  `json:"target,omitempty"`
	Params []VReg  `json:"params,omitempty"`
	Blocks []Block `json:"blocks"`
}

type Block struct {
	Name       string   `json:"name"`
	Instrs     []Instr  `json:"instrs,omitempty"`
	Successors []string `json:"successors,omitempty"`
}

type Instr struct {
	Op       Opcode    `json:"op"`
	Defs     []VReg    `json:"defs,omitempty"`
	Uses     []VReg    `json:"uses,omitempty"`
	Imm      int64     `json:"imm,omitempty"`
	Target   string    `json:"target,omitempty"`
	Call     string    `json:"call,omitempty"`
	ABI      string    `json:"abi,omitempty"`
	Clobbers []PhysReg `json:"clobbers,omitempty"`
	Note     string    `json:"note,omitempty"`
}

type Opcode string

const (
	OpMov        Opcode = "mov"
	OpLoad       Opcode = "load"
	OpStore      Opcode = "store"
	OpAdd        Opcode = "add"
	OpSub        Opcode = "sub"
	OpMul        Opcode = "mul"
	OpDiv        Opcode = "div"
	OpMod        Opcode = "mod"
	OpCmp        Opcode = "cmp"
	OpInc        Opcode = "inc"
	OpBranch     Opcode = "branch"
	OpBranchIf   Opcode = "branch_if"
	OpCall       Opcode = "call"
	OpReturn     Opcode = "return"
	OpSpill      Opcode = "spill"
	OpReload     Opcode = "reload"
	OpPush       Opcode = "push"
	OpPop        Opcode = "pop"
	OpIndexLoad  Opcode = "index_load"
	OpIndexStore Opcode = "index_store"

	OpVectorZeroI32x4           Opcode = "vector_zero_i32x4"
	OpVectorCanLoadI32x4        Opcode = "vector_can_load_i32x4"
	OpVectorLoadI32x4Unaligned  Opcode = "vector_load_i32x4_unaligned"
	OpVectorAddI32x4            Opcode = "vector_add_i32x4"
	OpVectorHorizontalAddI32x4  Opcode = "vector_horizontal_add_i32x4"
	OpTailScalarI32Sum          Opcode = "tail_scalar_i32_sum"
	OpVectorCanCopyU8x16        Opcode = "vector_can_copy_u8x16"
	OpVectorLoadU8x16Unaligned  Opcode = "vector_load_u8x16_unaligned"
	OpVectorStoreU8x16Unaligned Opcode = "vector_store_u8x16_unaligned"
	OpTailScalarU8Copy          Opcode = "tail_scalar_u8_copy"
	OpVectorZeroU8x16           Opcode = "vector_zero_u8x16"
	OpVectorCanMemsetU8x16      Opcode = "vector_can_memset_u8x16"
	OpTailScalarU8Memset        Opcode = "tail_scalar_u8_memset"
	OpVectorSplatI32x4          Opcode = "vector_splat_i32x4"
	OpVectorCanMapI32x4         Opcode = "vector_can_map_i32x4"
	OpVectorStoreI32x4Unaligned Opcode = "vector_store_i32x4_unaligned"
	OpTailScalarI32Map          Opcode = "tail_scalar_i32_map"
)

type VReg string

type PhysReg string

type Liveness struct {
	Blocks map[string]BlockLiveness `json:"blocks"`
}

type BlockLiveness struct {
	Use     []VReg `json:"use,omitempty"`
	Def     []VReg `json:"def,omitempty"`
	LiveIn  []VReg `json:"live_in,omitempty"`
	LiveOut []VReg `json:"live_out,omitempty"`
}

type Interval struct {
	Reg   VReg `json:"reg"`
	Start int  `json:"start"`
	End   int  `json:"end"`
}

type Allocation struct {
	Assignments map[VReg]PhysReg `json:"assignments,omitempty"`
	Spills      map[VReg]int     `json:"spills,omitempty"`
}

func VerifyFunction(fn Function) error {
	if fn.Name == "" {
		return fmt.Errorf("machine verifier: function with empty name")
	}
	if len(fn.Blocks) == 0 {
		return fmt.Errorf("machine verifier: %s has no blocks", fn.Name)
	}
	blocks := map[string]bool{}
	defined := map[VReg]bool{}
	for _, param := range fn.Params {
		if param == "" {
			return fmt.Errorf("machine verifier: %s has empty parameter vreg", fn.Name)
		}
		defined[param] = true
	}
	for _, block := range fn.Blocks {
		if block.Name == "" {
			return fmt.Errorf("machine verifier: %s has block with empty name", fn.Name)
		}
		if blocks[block.Name] {
			return fmt.Errorf("machine verifier: %s duplicate block %q", fn.Name, block.Name)
		}
		blocks[block.Name] = true
		for _, instr := range block.Instrs {
			for _, def := range instr.Defs {
				if def == "" {
					return fmt.Errorf("machine verifier: %s.%s has empty def vreg", fn.Name, block.Name)
				}
				defined[def] = true
			}
		}
	}
	for _, block := range fn.Blocks {
		if len(block.Instrs) == 0 {
			return fmt.Errorf("machine verifier: %s.%s missing terminator", fn.Name, block.Name)
		}
		last := block.Instrs[len(block.Instrs)-1]
		if !isMachineTerminator(last.Op) {
			return fmt.Errorf("machine verifier: %s.%s missing terminator", fn.Name, block.Name)
		}
		branchTargets := map[string]bool{}
		successors := map[string]bool{}
		for _, succ := range block.Successors {
			successors[succ] = true
		}
		for _, instr := range block.Instrs {
			if instr.Op == "" {
				return fmt.Errorf("machine verifier: %s.%s has instruction with empty opcode", fn.Name, block.Name)
			}
			if err := verifyInstrShape(fn.Name, block.Name, instr); err != nil {
				return err
			}
			for _, use := range instr.Uses {
				if use == "" {
					return fmt.Errorf("machine verifier: %s.%s has empty use vreg", fn.Name, block.Name)
				}
				if !defined[use] {
					return fmt.Errorf("machine verifier: %s.%s uses undefined vreg %q", fn.Name, block.Name, use)
				}
			}
			if instr.Op == OpBranch || instr.Op == OpBranchIf {
				if !blocks[instr.Target] {
					return fmt.Errorf("machine verifier: %s.%s unknown branch target %q", fn.Name, block.Name, instr.Target)
				}
				branchTargets[instr.Target] = true
				if len(block.Successors) > 0 && !successors[instr.Target] {
					return fmt.Errorf("machine verifier: %s.%s branch target %q missing from successors", fn.Name, block.Name, instr.Target)
				}
			}
		}
		for i, instr := range block.Instrs[:len(block.Instrs)-1] {
			if instr.Op == OpBranch || instr.Op == OpReturn {
				return fmt.Errorf("machine verifier: %s.%s terminator at instruction %d is not last", fn.Name, block.Name, i)
			}
		}
		for _, succ := range block.Successors {
			if !branchTargets[succ] {
				return fmt.Errorf("machine verifier: %s.%s successor %q has no branch instruction", fn.Name, block.Name, succ)
			}
		}
	}
	for _, block := range fn.Blocks {
		for _, succ := range block.Successors {
			if !blocks[succ] {
				return fmt.Errorf("machine verifier: %s.%s references unknown successor %q", fn.Name, block.Name, succ)
			}
		}
	}
	return nil
}

func verifyInstrShape(fnName string, blockName string, instr Instr) error {
	exact := func(kind string, got int, want int) error {
		if got != want {
			return fmt.Errorf("machine verifier: %s.%s %s has %d slots, want %d", fnName, blockName, kind, got, want)
		}
		return nil
	}
	atLeast := func(kind string, got int, want int) error {
		if got < want {
			return fmt.Errorf("machine verifier: %s.%s %s has %d slots, want at least %d", fnName, blockName, kind, got, want)
		}
		return nil
	}
	switch instr.Op {
	case OpMov:
		if err := exact("mov defs", len(instr.Defs), 1); err != nil {
			return err
		}
		if len(instr.Uses) > 1 {
			return fmt.Errorf("machine verifier: %s.%s mov has %d uses, want 0 or 1", fnName, blockName, len(instr.Uses))
		}
	case OpLoad:
		if err := exact("load defs", len(instr.Defs), 1); err != nil {
			return err
		}
		return atLeast("load uses", len(instr.Uses), 1)
	case OpStore:
		if err := exact("store defs", len(instr.Defs), 0); err != nil {
			return err
		}
		return atLeast("store uses", len(instr.Uses), 2)
	case OpAdd, OpSub, OpMul, OpDiv, OpMod, OpCmp:
		if err := exact(string(instr.Op)+" defs", len(instr.Defs), 1); err != nil {
			return err
		}
		return exact(string(instr.Op)+" uses", len(instr.Uses), 2)
	case OpInc:
		if err := exact("inc defs", len(instr.Defs), 1); err != nil {
			return err
		}
		return exact("inc uses", len(instr.Uses), 1)
	case OpBranch:
		if instr.Target == "" {
			return fmt.Errorf("machine verifier: %s.%s branch missing target", fnName, blockName)
		}
		if len(instr.Defs) != 0 || len(instr.Uses) != 0 {
			return fmt.Errorf("machine verifier: %s.%s branch must not define or use vregs", fnName, blockName)
		}
	case OpBranchIf:
		if instr.Target == "" {
			return fmt.Errorf("machine verifier: %s.%s branch missing target", fnName, blockName)
		}
		if err := exact("branch_if defs", len(instr.Defs), 0); err != nil {
			return err
		}
		return atLeast("branch_if uses", len(instr.Uses), 1)
	case OpCall:
		if instr.Call == "" {
			return fmt.Errorf("machine verifier: %s.%s call missing callee", fnName, blockName)
		}
		if instr.ABI == "" {
			return fmt.Errorf("machine verifier: %s.%s call %q missing ABI", fnName, blockName, instr.Call)
		}
		if len(instr.Clobbers) == 0 {
			return fmt.Errorf("machine verifier: %s.%s call %q missing clobber metadata", fnName, blockName, instr.Call)
		}
	case OpReturn:
		if len(instr.Defs) != 0 {
			return fmt.Errorf("machine verifier: %s.%s return must not define vregs", fnName, blockName)
		}
	case OpSpill:
		if err := exact("spill defs", len(instr.Defs), 0); err != nil {
			return err
		}
		if err := exact("spill uses", len(instr.Uses), 1); err != nil {
			return err
		}
		if instr.Imm < 0 {
			return fmt.Errorf("machine verifier: %s.%s spill has negative slot %d", fnName, blockName, instr.Imm)
		}
	case OpReload:
		if err := exact("reload defs", len(instr.Defs), 1); err != nil {
			return err
		}
		if err := exact("reload uses", len(instr.Uses), 0); err != nil {
			return err
		}
		if instr.Imm < 0 {
			return fmt.Errorf("machine verifier: %s.%s reload has negative slot %d", fnName, blockName, instr.Imm)
		}
	case OpPush:
		if err := exact("push defs", len(instr.Defs), 0); err != nil {
			return err
		}
		return exact("push uses", len(instr.Uses), 1)
	case OpPop:
		if err := exact("pop defs", len(instr.Defs), 1); err != nil {
			return err
		}
		return exact("pop uses", len(instr.Uses), 0)
	case OpIndexLoad:
		if err := exact("index_load defs", len(instr.Defs), 1); err != nil {
			return err
		}
		return exact("index_load uses", len(instr.Uses), 3)
	case OpIndexStore:
		if err := exact("index_store defs", len(instr.Defs), 0); err != nil {
			return err
		}
		return exact("index_store uses", len(instr.Uses), 4)
	case OpVectorZeroI32x4:
		if err := exact("vector_zero_i32x4 defs", len(instr.Defs), 1); err != nil {
			return err
		}
		return exact("vector_zero_i32x4 uses", len(instr.Uses), 0)
	case OpVectorCanLoadI32x4:
		if err := exact("vector_can_load_i32x4 defs", len(instr.Defs), 1); err != nil {
			return err
		}
		return exact("vector_can_load_i32x4 uses", len(instr.Uses), 2)
	case OpVectorLoadI32x4Unaligned:
		if err := exact("vector_load_i32x4_unaligned defs", len(instr.Defs), 1); err != nil {
			return err
		}
		return exact("vector_load_i32x4_unaligned uses", len(instr.Uses), 3)
	case OpVectorAddI32x4:
		if err := exact("vector_add_i32x4 defs", len(instr.Defs), 1); err != nil {
			return err
		}
		return exact("vector_add_i32x4 uses", len(instr.Uses), 2)
	case OpVectorHorizontalAddI32x4:
		if err := exact("vector_horizontal_add_i32x4 defs", len(instr.Defs), 1); err != nil {
			return err
		}
		return exact("vector_horizontal_add_i32x4 uses", len(instr.Uses), 1)
	case OpTailScalarI32Sum:
		if err := exact("tail_scalar_i32_sum defs", len(instr.Defs), 1); err != nil {
			return err
		}
		return exact("tail_scalar_i32_sum uses", len(instr.Uses), 4)
	case OpVectorCanCopyU8x16:
		if err := exact("vector_can_copy_u8x16 defs", len(instr.Defs), 1); err != nil {
			return err
		}
		return exact("vector_can_copy_u8x16 uses", len(instr.Uses), 2)
	case OpVectorLoadU8x16Unaligned:
		if err := exact("vector_load_u8x16_unaligned defs", len(instr.Defs), 1); err != nil {
			return err
		}
		return exact("vector_load_u8x16_unaligned uses", len(instr.Uses), 3)
	case OpVectorStoreU8x16Unaligned:
		if err := exact("vector_store_u8x16_unaligned defs", len(instr.Defs), 0); err != nil {
			return err
		}
		return exact("vector_store_u8x16_unaligned uses", len(instr.Uses), 4)
	case OpTailScalarU8Copy:
		if err := exact("tail_scalar_u8_copy defs", len(instr.Defs), 0); err != nil {
			return err
		}
		return exact("tail_scalar_u8_copy uses", len(instr.Uses), 4)
	case OpVectorZeroU8x16:
		if err := exact("vector_zero_u8x16 defs", len(instr.Defs), 1); err != nil {
			return err
		}
		return exact("vector_zero_u8x16 uses", len(instr.Uses), 0)
	case OpVectorCanMemsetU8x16:
		if err := exact("vector_can_memset_u8x16 defs", len(instr.Defs), 1); err != nil {
			return err
		}
		return exact("vector_can_memset_u8x16 uses", len(instr.Uses), 2)
	case OpTailScalarU8Memset:
		if err := exact("tail_scalar_u8_memset defs", len(instr.Defs), 0); err != nil {
			return err
		}
		return exact("tail_scalar_u8_memset uses", len(instr.Uses), 4)
	case OpVectorSplatI32x4:
		if err := exact("vector_splat_i32x4 defs", len(instr.Defs), 1); err != nil {
			return err
		}
		return exact("vector_splat_i32x4 uses", len(instr.Uses), 0)
	case OpVectorCanMapI32x4:
		if err := exact("vector_can_map_i32x4 defs", len(instr.Defs), 1); err != nil {
			return err
		}
		return exact("vector_can_map_i32x4 uses", len(instr.Uses), 2)
	case OpVectorStoreI32x4Unaligned:
		if err := exact("vector_store_i32x4_unaligned defs", len(instr.Defs), 0); err != nil {
			return err
		}
		return exact("vector_store_i32x4_unaligned uses", len(instr.Uses), 4)
	case OpTailScalarI32Map:
		if err := exact("tail_scalar_i32_map defs", len(instr.Defs), 0); err != nil {
			return err
		}
		return exact("tail_scalar_i32_map uses", len(instr.Uses), 4)
	default:
		return fmt.Errorf("machine verifier: %s.%s unknown opcode %q", fnName, blockName, instr.Op)
	}
	return nil
}

func isMachineTerminator(op Opcode) bool {
	return op == OpBranch || op == OpReturn
}

func VerifyAllocation(fn Function, alloc Allocation, regs []PhysReg, spillSlots int) error {
	if err := VerifyFunction(fn); err != nil {
		return err
	}
	if spillSlots < 0 {
		return fmt.Errorf("machine allocation verifier: negative spill slot count %d", spillSlots)
	}
	allowedRegs := map[PhysReg]bool{}
	for _, reg := range regs {
		if reg == "" {
			return fmt.Errorf("machine allocation verifier: empty physreg")
		}
		allowedRegs[reg] = true
	}
	vregs := functionVRegs(fn)
	for reg, phys := range alloc.Assignments {
		if !vregs[reg] {
			return fmt.Errorf("machine allocation verifier: assignment for unknown vreg %q", reg)
		}
		if !allowedRegs[phys] {
			return fmt.Errorf("machine allocation verifier: invalid physreg %q for %s", phys, reg)
		}
		if _, spilled := alloc.Spills[reg]; spilled {
			return fmt.Errorf("machine allocation verifier: %s cannot be assigned and spilled", reg)
		}
	}
	for reg, slot := range alloc.Spills {
		if !vregs[reg] {
			return fmt.Errorf("machine allocation verifier: spill for unknown vreg %q", reg)
		}
		if slot < 0 || slot >= spillSlots {
			return fmt.Errorf("machine allocation verifier: spill slot %d for %s out of bounds (slots=%d)", slot, reg, spillSlots)
		}
	}
	intervals, err := BuildIntervals(fn)
	if err != nil {
		return err
	}
	byReg := map[VReg]Interval{}
	for _, interval := range intervals {
		byReg[interval.Reg] = interval
	}
	assigned := make([]VReg, 0, len(alloc.Assignments))
	for reg := range alloc.Assignments {
		assigned = append(assigned, reg)
	}
	sort.Slice(assigned, func(i, j int) bool { return assigned[i] < assigned[j] })
	for i, left := range assigned {
		for _, right := range assigned[i+1:] {
			if alloc.Assignments[left] != alloc.Assignments[right] {
				continue
			}
			if intervalsOverlap(byReg[left], byReg[right]) {
				return fmt.Errorf("machine allocation verifier: overlapping vregs %s and %s share physreg %s", left, right, alloc.Assignments[left])
			}
		}
	}
	return nil
}

func AnalyzeLiveness(fn Function) (Liveness, error) {
	if err := VerifyFunction(fn); err != nil {
		return Liveness{}, err
	}
	out := Liveness{Blocks: map[string]BlockLiveness{}}
	for _, block := range fn.Blocks {
		useSet := map[VReg]bool{}
		defSet := map[VReg]bool{}
		for _, instr := range block.Instrs {
			for _, use := range instr.Uses {
				if !defSet[use] {
					useSet[use] = true
				}
			}
			for _, def := range instr.Defs {
				defSet[def] = true
			}
		}
		out.Blocks[block.Name] = BlockLiveness{
			Use: setToSortedRegs(useSet),
			Def: setToSortedRegs(defSet),
		}
	}
	changed := true
	for changed {
		changed = false
		for i := len(fn.Blocks) - 1; i >= 0; i-- {
			block := fn.Blocks[i]
			info := out.Blocks[block.Name]
			liveOut := map[VReg]bool{}
			for _, succ := range block.Successors {
				for _, reg := range out.Blocks[succ].LiveIn {
					liveOut[reg] = true
				}
			}
			liveIn := regsToSet(info.Use)
			for reg := range liveOut {
				if !containsReg(info.Def, reg) {
					liveIn[reg] = true
				}
			}
			next := BlockLiveness{
				Use:     info.Use,
				Def:     info.Def,
				LiveIn:  setToSortedRegs(liveIn),
				LiveOut: setToSortedRegs(liveOut),
			}
			if !sameRegs(info.LiveIn, next.LiveIn) || !sameRegs(info.LiveOut, next.LiveOut) {
				out.Blocks[block.Name] = next
				changed = true
			}
		}
	}
	return out, nil
}

func BuildIntervals(fn Function) ([]Interval, error) {
	if err := VerifyFunction(fn); err != nil {
		return nil, err
	}
	positions := map[VReg]Interval{}
	pos := 0
	touch := func(reg VReg) {
		if reg == "" {
			return
		}
		interval, ok := positions[reg]
		if !ok {
			positions[reg] = Interval{Reg: reg, Start: pos, End: pos}
			return
		}
		if pos < interval.Start {
			interval.Start = pos
		}
		if pos > interval.End {
			interval.End = pos
		}
		positions[reg] = interval
	}
	for _, param := range fn.Params {
		touch(param)
	}
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			for _, use := range instr.Uses {
				touch(use)
			}
			for _, def := range instr.Defs {
				touch(def)
			}
			pos++
		}
	}
	intervals := make([]Interval, 0, len(positions))
	for _, interval := range positions {
		intervals = append(intervals, interval)
	}
	sort.Slice(intervals, func(i, j int) bool {
		if intervals[i].Start == intervals[j].Start {
			return intervals[i].Reg < intervals[j].Reg
		}
		return intervals[i].Start < intervals[j].Start
	})
	return intervals, nil
}

func LinearScan(intervals []Interval, regs []PhysReg) (Allocation, error) {
	if len(regs) == 0 {
		return Allocation{}, fmt.Errorf("machine linear scan: no physical registers")
	}
	sortedIntervals := append([]Interval(nil), intervals...)
	sort.Slice(sortedIntervals, func(i, j int) bool {
		if sortedIntervals[i].Start == sortedIntervals[j].Start {
			return sortedIntervals[i].End < sortedIntervals[j].End
		}
		return sortedIntervals[i].Start < sortedIntervals[j].Start
	})
	alloc := Allocation{Assignments: map[VReg]PhysReg{}, Spills: map[VReg]int{}}
	active := []Interval{}
	free := append([]PhysReg(nil), regs...)
	nextSpill := 0
	expireOld := func(start int) {
		kept := active[:0]
		for _, interval := range active {
			if interval.End >= start {
				kept = append(kept, interval)
				continue
			}
			if reg, ok := alloc.Assignments[interval.Reg]; ok {
				free = append(free, reg)
			}
		}
		active = kept
		sort.Slice(free, func(i, j int) bool { return free[i] < free[j] })
	}
	for _, interval := range sortedIntervals {
		expireOld(interval.Start)
		if len(free) > 0 {
			reg := free[0]
			free = free[1:]
			alloc.Assignments[interval.Reg] = reg
			active = appendActive(active, interval)
			continue
		}
		spillAt := farthestEnding(active)
		if spillAt >= 0 && active[spillAt].End > interval.End {
			spilled := active[spillAt]
			reg := alloc.Assignments[spilled.Reg]
			delete(alloc.Assignments, spilled.Reg)
			alloc.Spills[spilled.Reg] = nextSpill
			nextSpill++
			alloc.Assignments[interval.Reg] = reg
			active[spillAt] = interval
			active = appendActive(nil, active...)
		} else {
			alloc.Spills[interval.Reg] = nextSpill
			nextSpill++
		}
	}
	return alloc, nil
}

func LinuxX64CallerSaved() []PhysReg {
	return []PhysReg{"rax", "rcx", "rdx", "rsi", "rdi", "r8", "r9", "r10", "r11"}
}

func Win64CallerSaved() []PhysReg {
	return []PhysReg{"rax", "rcx", "rdx", "r8", "r9", "r10", "r11"}
}

type CallABIInfo struct {
	Name        string
	Clobbers    []PhysReg
	MaxArgSlots int
	MaxRetSlots int
}

func SysVCallABIInfo() CallABIInfo {
	return CallABIInfo{
		Name:        "sysv",
		Clobbers:    LinuxX64CallerSaved(),
		MaxArgSlots: 6,
		MaxRetSlots: 1,
	}
}

func Win64CallABIInfo() CallABIInfo {
	return CallABIInfo{
		Name:        "win64",
		Clobbers:    Win64CallerSaved(),
		MaxArgSlots: 4,
		MaxRetSlots: 1,
	}
}

func SumToLoopFunction() Function {
	n := VReg("n")
	i := VReg("i")
	total := VReg("total")
	cmp := VReg("cmp")
	return Function{
		Name:   "sum_to",
		Target: "linux-x64",
		Params: []VReg{n},
		Blocks: []Block{
			{
				Name: "entry",
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{i}, Imm: 0, Note: "i = 0"},
					{Op: OpMov, Defs: []VReg{total}, Imm: 0, Note: "total = 0"},
					{Op: OpBranch, Target: "loop"},
				},
				Successors: []string{"loop"},
			},
			{
				Name: "loop",
				Instrs: []Instr{
					{Op: OpCmp, Defs: []VReg{cmp}, Uses: []VReg{i, n}, Note: "i < n"},
					{Op: OpBranchIf, Uses: []VReg{cmp}, Target: "exit"},
					{Op: OpAdd, Defs: []VReg{total}, Uses: []VReg{total, i}, Note: "total += i"},
					{Op: OpInc, Defs: []VReg{i}, Uses: []VReg{i}, Note: "i++"},
					{Op: OpBranch, Target: "loop"},
				},
				Successors: []string{"loop", "exit"},
			},
			{
				Name: "exit",
				Instrs: []Instr{
					{Op: OpReturn, Uses: []VReg{total}},
				},
			},
		},
	}
}

func FormatFunction(fn Function) string {
	var b strings.Builder
	fmt.Fprintf(&b, "func %s target:%s", fn.Name, fn.Target)
	if len(fn.Params) > 0 {
		fmt.Fprintf(&b, " params:%s", joinRegs(fn.Params))
	}
	fmt.Fprintln(&b)
	for _, block := range fn.Blocks {
		fmt.Fprintf(&b, "%s:\n", block.Name)
		for _, instr := range block.Instrs {
			fmt.Fprintf(&b, "  %s", instr.Op)
			if instr.Call != "" {
				fmt.Fprintf(&b, " %s", instr.Call)
			}
			if len(instr.Defs) > 0 {
				fmt.Fprintf(&b, " defs:%s", joinRegs(instr.Defs))
			}
			if len(instr.Uses) > 0 {
				fmt.Fprintf(&b, " uses:%s", joinRegs(instr.Uses))
			}
			if instr.Target != "" {
				fmt.Fprintf(&b, " -> %s", instr.Target)
			}
			if instr.ABI != "" {
				fmt.Fprintf(&b, " abi:%s", instr.ABI)
			}
			if len(instr.Clobbers) > 0 {
				fmt.Fprintf(&b, " clobbers:%s", joinPhysRegs(instr.Clobbers))
			}
			if instr.Note != "" {
				fmt.Fprintf(&b, " ; %s", instr.Note)
			}
			fmt.Fprintln(&b)
		}
	}
	return b.String()
}

func FormatProgram(prog Program) string {
	var b strings.Builder
	fmt.Fprintln(&b, "program machine_ir")
	for i, fn := range prog.Functions {
		if i > 0 {
			fmt.Fprintln(&b)
		}
		b.WriteString(FormatFunction(fn))
	}
	return b.String()
}

func appendActive(active []Interval, intervals ...Interval) []Interval {
	active = append(active, intervals...)
	sort.Slice(active, func(i, j int) bool {
		if active[i].End == active[j].End {
			return active[i].Reg < active[j].Reg
		}
		return active[i].End < active[j].End
	})
	return active
}

func farthestEnding(active []Interval) int {
	if len(active) == 0 {
		return -1
	}
	idx := 0
	for i := 1; i < len(active); i++ {
		if active[i].End > active[idx].End {
			idx = i
		}
	}
	return idx
}

func setToSortedRegs(set map[VReg]bool) []VReg {
	out := make([]VReg, 0, len(set))
	for reg := range set {
		out = append(out, reg)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func regsToSet(regs []VReg) map[VReg]bool {
	out := map[VReg]bool{}
	for _, reg := range regs {
		out[reg] = true
	}
	return out
}

func containsReg(regs []VReg, want VReg) bool {
	for _, reg := range regs {
		if reg == want {
			return true
		}
	}
	return false
}

func sameRegs(a, b []VReg) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func joinRegs(regs []VReg) string {
	parts := make([]string, len(regs))
	for i, reg := range regs {
		parts[i] = string(reg)
	}
	return strings.Join(parts, ",")
}

func joinPhysRegs(regs []PhysReg) string {
	parts := make([]string, len(regs))
	for i, reg := range regs {
		parts[i] = string(reg)
	}
	return strings.Join(parts, ",")
}

func functionVRegs(fn Function) map[VReg]bool {
	out := map[VReg]bool{}
	for _, param := range fn.Params {
		if param != "" {
			out[param] = true
		}
	}
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			for _, reg := range instr.Defs {
				if reg != "" {
					out[reg] = true
				}
			}
			for _, reg := range instr.Uses {
				if reg != "" {
					out[reg] = true
				}
			}
		}
	}
	return out
}

func intervalsOverlap(a Interval, b Interval) bool {
	return a.Start <= b.End && b.Start <= a.End
}
