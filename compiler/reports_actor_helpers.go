package compiler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"tetra_language/compiler/internal/allocplan"
	"tetra_language/compiler/internal/buildreports"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/plir"
	"tetra_language/compiler/internal/semantics"
)

func buildAllocReport(prog *ir.IRProgram, target string) allocReport {
	return buildreports.BuildAllocReport(prog, target)
}

func buildActorTransferReport(checked *semantics.CheckedProgram, target string) actorTransferReport {
	return buildreports.BuildActorTransferReport(checked, target)
}

func writeReport(path string, data any) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

func formatExplainText(target string, bounds boundsReport, alloc *allocplan.Plan, plirProg *plir.Program) string {
	var b strings.Builder
	fmt.Fprintf(&b, "target: %s\n", target)
	fmt.Fprintf(&b, "bounds checks removed: %d\n", bounds.Totals.Removed)
	fmt.Fprintf(&b, "bounds checks left: %d\n", bounds.Totals.Left)
	if alloc != nil {
		fmt.Fprintf(&b, "planned heap allocations: %d\n", alloc.Totals.Heap)
		fmt.Fprintf(&b, "planned stack allocations: %d\n", alloc.Totals.Stack)
		fmt.Fprintf(&b, "explicit island allocations: %d\n\n", alloc.Totals.ExplicitIsland)
		b.WriteString(allocplan.FormatText(alloc))
		b.WriteString("\n")
	}
	b.WriteString(plir.FormatText(plirProg))
	return b.String()
}
