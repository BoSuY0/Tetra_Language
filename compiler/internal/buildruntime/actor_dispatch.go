package buildruntime

import (
	"fmt"
	"hash/fnv"
	"sort"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/semantics"
)

func fnv1a32(s string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return h.Sum32()
}

func typedTaskRuntimeWrapperName(target, errorType string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(target))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(errorType))
	return fmt.Sprintf("__tetra_task_typed_%08x", h.Sum32())
}

func BuildActorDispatchFunc(entries []string, checked *semantics.CheckedProgram) (ir.IRFunc, error) {
	if len(entries) == 0 {
		return ir.IRFunc{}, fmt.Errorf("missing actor entries")
	}
	seen := make(map[uint32]string, len(entries))
	for _, name := range entries {
		id := fnv1a32(name)
		if other, exists := seen[id]; exists && other != name {
			return ir.IRFunc{}, fmt.Errorf("actor entry ID collision: %q and %q both hash to %d", other, name, id)
		}
		seen[id] = name
	}

	initByEntry := map[string][]semantics.ActorStateField{}
	if checked != nil {
		for _, fn := range checked.Funcs {
			if len(fn.ActorState) == 0 {
				continue
			}
			fields := make([]semantics.ActorStateField, 0, len(fn.ActorState))
			for _, field := range fn.ActorState {
				fields = append(fields, field)
			}
			sort.Slice(fields, func(i, j int) bool {
				return fields[i].Slot < fields[j].Slot
			})
			initByEntry[fn.Name] = fields
		}
	}

	var instrs []ir.IRInstr
	localSlots := 1
	if len(initByEntry) > 0 {
		localSlots = 2
	}
	nextLabel := 1
	for _, name := range entries {
		id := int32(fnv1a32(name))
		skipLabel := nextLabel
		nextLabel++

		instrs = append(instrs,
			ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
			ir.IRInstr{Kind: ir.IRConstI32, Imm: id},
			ir.IRInstr{Kind: ir.IRCmpEqI32},
			ir.IRInstr{Kind: ir.IRJmpIfZero, Label: skipLabel},
		)
		if fields, ok := initByEntry[name]; ok {
			for _, field := range fields {
				instrs = append(instrs,
					ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(field.Slot)},
					ir.IRInstr{Kind: ir.IRConstI32, Imm: field.Init},
					ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_state_store", ArgSlots: 2, RetSlots: 1},
					ir.IRInstr{Kind: ir.IRStoreLocal, Local: 1},
				)
			}
		}
		instrs = append(instrs,
			ir.IRInstr{Kind: ir.IRCall, Name: name, ArgSlots: 0, RetSlots: 1},
			ir.IRInstr{Kind: ir.IRReturn},
			ir.IRInstr{Kind: ir.IRLabel, Label: skipLabel},
		)
	}

	instrs = append(instrs,
		ir.IRInstr{Kind: ir.IRConstI32, Imm: 1},
		ir.IRInstr{Kind: ir.IRReturn},
	)

	return ir.IRFunc{
		Name:        "__tetra_actor_dispatch",
		ParamSlots:  1,
		LocalSlots:  localSlots,
		ReturnSlots: 1,
		Instrs:      instrs,
	}, nil
}

func BuildActorMainEntryIDFunc(mainName string) (ir.IRFunc, error) {
	if mainName == "" {
		return ir.IRFunc{}, fmt.Errorf("missing main name")
	}
	id := int32(fnv1a32(mainName))
	return ir.IRFunc{
		Name:        "__tetra_actor_main_entry_id",
		ParamSlots:  0,
		LocalSlots:  0,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: id},
			{Kind: ir.IRReturn},
		},
	}, nil
}
