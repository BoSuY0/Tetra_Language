package buildreports

import (
	"fmt"

	"tetra_language/compiler/internal/allocplan"
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
)

func BuildAllocReport(prog *ir.IRProgram, target string) AllocReport {
	report := AllocReport{
		ReportEnvelope: ReportEnvelope{SchemaVersion: 1, Kind: "allocation", Target: target},
	}
	if prog == nil {
		return report
	}
	for _, fn := range prog.Funcs {
		row := AllocFunctionRow{Function: fn.Name}
		for _, instr := range fn.Instrs {
			switch instr.Kind {
			case ir.IRMakeSliceU8, ir.IRMakeSliceU16, ir.IRMakeSliceI32, ir.IRAllocBytes:
				report.Totals.Heap++
				row.Allocations = append(row.Allocations, AllocationDecision{
					Site:            reportPos(instr.Pos),
					Kind:            irAllocKind(instr.Kind),
					Storage:         "Heap",
					Reason:          "allocation planner v0 keeps conservative heap storage until escape facts select a narrower class",
					ReasonCodes:     []string{allocplan.HeapReasonDynamicLifetime},
					HeapReasonCodes: []string{allocplan.HeapReasonDynamicLifetime},
				})
			case ir.IRStackSliceU8, ir.IRStackSliceU16, ir.IRStackSliceI32:
				report.Totals.Stack++
				row.Allocations = append(row.Allocations, AllocationDecision{
					Site:    reportPos(instr.Pos),
					Kind:    irAllocKind(instr.Kind),
					Storage: "Stack",
					Reason:  "fixed small no-escape allocation lowers to stack frame storage",
				})
			case ir.IRIslandMakeSliceU8, ir.IRIslandMakeSliceU16, ir.IRIslandMakeSliceI32, ir.IRIslandNew:
				report.Totals.ExplicitIsland++
				row.Allocations = append(row.Allocations, AllocationDecision{
					Site:    reportPos(instr.Pos),
					Kind:    irAllocKind(instr.Kind),
					Storage: "ExplicitIsland",
					Reason:  "user-written island scope selects explicit region storage",
				})
			}
		}
		if len(row.Allocations) > 0 {
			report.Functions = append(report.Functions, row)
		}
	}
	return report
}

func irAllocKind(kind ir.IRInstrKind) string {
	switch kind {
	case ir.IRMakeSliceU8:
		return "make_u8"
	case ir.IRMakeSliceU16:
		return "make_u16"
	case ir.IRMakeSliceI32:
		return "make_i32_or_bool"
	case ir.IRStackSliceU8:
		return "stack_make_u8"
	case ir.IRStackSliceU16:
		return "stack_make_u16"
	case ir.IRStackSliceI32:
		return "stack_make_i32_or_bool"
	case ir.IRAllocBytes:
		return "alloc_bytes"
	case ir.IRIslandNew:
		return "island_new"
	case ir.IRIslandMakeSliceU8:
		return "island_make_u8"
	case ir.IRIslandMakeSliceU16:
		return "island_make_u16"
	case ir.IRIslandMakeSliceI32:
		return "island_make_i32_or_bool"
	default:
		return fmt.Sprintf("ir.%d", kind)
	}
}

func reportPos(pos frontend.Position) string {
	if pos.Line == 0 && pos.Col == 0 && pos.File == "" {
		return ""
	}
	return frontend.FormatPos(pos)
}
