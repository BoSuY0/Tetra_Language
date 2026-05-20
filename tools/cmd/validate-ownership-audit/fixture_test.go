package main

import (
	"strings"
	"testing"
)

func validBlockedOwnershipAudit() string {
	rows := []ownershipAuditRow{
		{
			Requirement: "Borrow/consume/inout model",
			Artifact:    "`compiler/tests/ownership/ownership_test.go`",
			Evidence:    "Marker parser/checker tests and ownership smoke cover local calls.",
			Result:      "pass",
		},
		{
			Requirement: "SSA local lifetime analysis",
			Artifact:    "`language.lifetime-ssa`",
			Evidence: auditParagraph(
				"Local control-flow solver, branch/match/loop task-handle maybe-joined, task-group maybe-closed, island maybe-freed diagnostics,",
				"and branch/match/loop resource finalization merge diagnostics with stable TETRA2101 JSON evidence.",
			),
			Result: "pass",
		},
		{
			Requirement: "Interprocedural lifetime analysis",
			Artifact:    "ownership audit and tests",
			Evidence: auditParagraph(
				"Local return-resource summaries, typed-error throw-resource summaries including rethrow-through-`try` are covered, while richer interprocedural lifetime proofs remain open;",
				"same-module and interface-only cross-module per-field interprocedural region summaries for aggregate returns from multiple island parameters, including optional aggregate wrappers, enum payload wrappers, branch aggregate wrappers, match aggregate wrappers, if-let aggregate wrappers, mixed safe/provenance aggregate branch and match returns, and optional mixed safe/provenance aggregate branch merges are covered in the bounded interface-only slice,",
				"alongside generated `.t4i` direct/local/aggregate-local-alias/aggregate-field-access/aggregate-field-local-alias resource return, assignment/let/direct-if-let/direct-match/field-local/if-let/match optional and nested/field-local nested optional resource return, typed-error direct/field-local-alias throw, and rethrow-through-`try` direct/field-local-alias provenance stubs,",
				"plus selected same-module/cross-module transitive interprocedural resource cases, including task-handle, task-group, island, struct-field, enum-payload, enum-constructor return, same-module throw/catch enum-payload, if-let/match optional-payload, and nested struct/enum optional-payload return resource aliases.",
			),
			Result: "partial",
		},
		{
			Requirement: "Alias/provenance tracking",
			Artifact:    "ownership/resource tests",
			Evidence: auditParagraph(
				"Ownership paths, enum payload aliases, borrowed ptr-leaf aliases for ptr-containing aggregate parameters, borrowed scalar `ptr` assignment into optional `ptr?` payloads, borrowed region-bearing slice assignment into optional `[]u8?` payloads, and pattern-bound enum/optional payloads,",
				"optional payload consume aliases, if-let/match optional resource aliases, resource provenance, typed-error throw/catch and rethrow-through-try enum-payload resource aliases with stable TETRA2101 JSON diagnostic evidence,",
				"generated `.t4i` direct/local/aggregate-local-alias/aggregate-field-access/aggregate-field-local-alias resource return, assignment/let/direct-if-let/direct-match/field-local/if-let/match optional and nested/field-local nested optional resource return, typed-error direct/field-local-alias throw, and rethrow-through-`try` direct/field-local-alias provenance stubs,",
				"optional resource wrapper aliases including nested struct/enum wrappers,",
				"same-module/cross-module actor if-let/match optional-payload, struct-field, enum-payload, and transitive interprocedural consume aliases including same-module/cross-module actor struct-field/enum-payload alias transfer diagnostics with stable TETRA2101 JSON diagnostic evidence,",
				"same-module/cross-module task-handle/task-group struct-field/enum-payload transfer/join/close aliases including same-module/cross-module task-handle struct-field/enum-payload alias transfer diagnostics with stable TETRA2101 JSON diagnostic evidence, same-module/cross-module task-handle struct-field/enum-payload alias join diagnostics with stable TETRA2101 JSON diagnostic evidence and same-module/cross-module task-group struct-field/enum-payload alias close diagnostics with stable TETRA2101 JSON diagnostic evidence,",
				"same-module/cross-module monomorphized generic struct actor consume aliases, same-module/cross-module monomorphized generic struct task-handle/task-group/island resource aliases with stable TETRA2101 CLI JSON evidence,",
				"same-module/cross-module enum-constructor return resource aliases with stable TETRA2101 CLI JSON evidence,",
				"same-module/cross-module transitive interprocedural task-handle/task-group/island resource aliases with stable TETRA2101 CLI JSON evidence, and ambiguous resource provenance exist; broad alias/provenance remains open.",
			),
			Result: "partial",
		},
		{
			Requirement: "Move/copy/drop/finalization semantics",
			Artifact:    "ownership/resource finalization tests",
			Evidence: auditParagraph(
				"Local moves, mutable reinitialization, task/island/task-group finalization including stable `TETRA2101` task-group use-after-close, resource finalization,",
				"same-module/cross-module struct/enum whole-value call/let/return rejection after partial consume,",
				"same-module/cross-module enum wrapper-constructor rejection after partial field/payload consume,",
				"same-module/cross-module optional-payload whole-value rejection after payload consume/free with stable TETRA2101 JSON diagnostic evidence,",
				"same-module/cross-module task-handle struct-field/enum-payload alias join diagnostics with stable TETRA2101 JSON diagnostic evidence,",
				"same-module/cross-module task-group struct-field/enum-payload alias close diagnostics with stable TETRA2101 JSON diagnostic evidence,",
				"same-module/cross-module task-handle/task-group if-let/match optional-payload join/close aliases with stable TETRA2101 CLI JSON evidence,",
				"same-module/cross-module nested optional resource wrapper alias use-after-free CLI JSON diagnostics,",
				"same-module/cross-module struct-field/enum-payload alias use-after-free CLI JSON diagnostics, same-module/cross-module task_group_cancel return provenance diagnostics with stable TETRA2101 CLI JSON evidence, and same-module/cross-module struct-field and enum-payload alias use-after-free with stable TETRA2101 JSON diagnostic evidence are covered.",
			),
			Result: "partial",
		},
		{
			Requirement: "Partial moves for struct/enum fields",
			Artifact:    "ownership tests and example",
			Evidence:    "Struct fields and enum payload partial consume are covered.",
			Result:      "pass",
		},
		{
			Requirement: "Ownership-aware generics/interfaces/callables",
			Artifact:    "ownership/generic/callable tests",
			Evidence: auditParagraph(
				"Generic fnptr and protocol marker checks plus generic function-typed global consume-marker preservation and ownership mismatch diagnostics,",
				"same-module/cross-module generic aggregate and optional-ptr owned/consume/inout instantiations including slice-containing struct/enum aggregate instantiations with stable TETRA2101 CLI JSON evidence,",
				"same-module/cross-module generic borrow-aggregate/optional-ptr return diagnostics with stable TETRA2102 CLI JSON evidence,",
				"same-module/cross-module monomorphized generic struct actor consume alias diagnostics plus same-module/cross-module monomorphized generic struct task-handle/task-group/island resource aliases with stable TETRA2101 CLI JSON evidence,",
				"same-module/cross-module function-typed value/struct-field/enum-payload optional-ptr owned/consume/inout callback diagnostics with stable TETRA2101 CLI JSON evidence,",
				"function-typed value/struct-field/enum-payload callback slice-containing struct/enum owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence,",
				"same-module/cross-module protocol parameter ownership matching plus same-module/cross-module protocol impl parameter ownership mismatch diagnostics with stable TETRA2001 CLI JSON evidence,",
				"same-module/cross-module generic protocol requirement parameter ownership mismatch diagnostics with stable TETRA2001 JSON diagnostic evidence, generated `.t4i` function-typed parameter local-alias return metadata for interface-only global-storage diagnostics, and function type ownership markers parse/format plus function-typed callable ownership-marker diagnostics are covered.",
			),
			Result: "partial",
		},
		{
			Requirement: "Heap/global/thread/callback escape analysis",
			Artifact:    "callable escape tests",
			Evidence: auditParagraph(
				"Callable escape, same-module/cross-module fixed-array alias return, same-module/cross-module direct fixed-array global assignment, same-module/cross-module optional fixed-array global assignment, same-module/cross-module fixed-array inout assignment, fixed-array escapes including inout assignment with stable TETRA2102 diagnostic evidence, borrowed string alias return/global assignment,",
				"same-module/cross-module direct slice global assignment with stable TETRA2102 JSON diagnostic evidence, same-module/cross-module optional ptr global assignment with stable TETRA2102 JSON diagnostic evidence, same-module/cross-module optional aggregate global assignment with stable TETRA2102 JSON diagnostic evidence,",
				"same-module/cross-module ptr optional assignment if-let/match global escape with stable TETRA2102 JSON diagnostic evidence, same-module/cross-module ptr enum alias return escape with stable TETRA2102 JSON diagnostic evidence, same-module/cross-module ptr-containing aggregate whole/field/alias/nested-field return escapes with stable TETRA2102 JSON diagnostic evidence,",
				"same-module/cross-module ptr enum-payload return/global/inout assignment escapes with stable TETRA2102 JSON diagnostic evidence, same-module/cross-module ptr optional-payload return/global/inout assignment escapes with stable TETRA2102 JSON diagnostic evidence, same-module/cross-module slice optional-payload inout/global assignment escapes with stable TETRA2102 JSON diagnostic evidence,",
				"same-module/cross-module nested slice enum-payload return/inout/global assignment escapes with stable TETRA2102 JSON diagnostic evidence, same-module/cross-module nested slice struct return/inout/global assignment escapes with stable TETRA2102 JSON diagnostic evidence, same-module/cross-module whole-aggregate global assignment with stable TETRA2102 JSON diagnostic evidence,",
				"same-module/cross-module ptr-containing enum whole-value global assignment with stable TETRA2102 JSON diagnostic evidence, same-module/cross-module global field target assignment with stable TETRA2102 JSON diagnostic evidence, same-module/cross-module aggregate and nested-aggregate global field escapes with stable TETRA2102 JSON diagnostic evidence,",
				"same-module/cross-module ptr-containing/nested aggregate owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence, same-module/cross-module ptr enum-payload owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence, same-module/cross-module ptr optional-payload owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence,",
				"same-module/cross-module slice optional-payload owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence, and imported direct ptr-containing/nested aggregate owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence are covered.",
			),
			Result: "partial",
		},
		{
			Requirement: "Actor/task/island/resource transfer rules",
			Artifact:    "actor/task/resource tests",
			Evidence: auditParagraph(
				"Local actor/task/island/resource transfer diagnostics, branch/match/loop actor consume reuse diagnostics with stable TETRA2101 CLI JSON evidence, actor/task use-after-transfer diagnostics with stable TETRA2101 CLI JSON evidence, island transfer non-local-payload rejection with stable TETRA2101 CLI JSON evidence,",
				"same-module/cross-module transitive actor consume alias diagnostics with stable TETRA2101 CLI JSON evidence, same-module/cross-module monomorphized generic struct actor consume alias diagnostics with stable TETRA2101 CLI JSON evidence, same-module/cross-module task_group_cancel return provenance diagnostics with stable TETRA2101 CLI JSON evidence,",
				"same-module/cross-module actor struct-field/enum-payload alias transfer diagnostics with stable TETRA2101 JSON diagnostic evidence, same-module/cross-module actor/task if-let/match optional-payload alias transfer diagnostics with stable TETRA2101 JSON diagnostic evidence, and same-module/cross-module task-handle struct-field/enum-payload alias transfer diagnostics with stable TETRA2101 JSON diagnostic evidence are covered.",
			),
			Result: "partial",
		},
		{
			Requirement: "Stable forbidden-case diagnostics",
			Artifact:    "negative tests and diagnostic shape gate",
			Evidence:    stableForbiddenCaseDiagnosticsFixtureEvidence(),
			Result:      "partial",
		},
		{
			Requirement: "Spec/docs/examples/tests evidence",
			Artifact:    "ownership spec, supported surface, `examples/ownership_smoke.tetra`",
			Evidence:    "Specs and runnable example are present.",
			Result:      "pass",
		},
		{
			Requirement: "Dedicated ownership validator evidence",
			Artifact:    "`tools/cmd/validate-ownership-audit`",
			Evidence:    "This validator checks audit integrity.",
			Result:      "pass as blocker",
		},
		{
			Requirement: "Feature registry evidence",
			Artifact:    "`./tetra features --format=json`",
			Evidence:    "Registry has current bounded ownership/lifetime slices.",
			Result:      "partial",
		},
		{
			Requirement: "Release-gate evidence",
			Artifact:    "`bash scripts/ci/test-all.sh`",
			Evidence:    "Structured test-all report `docs/generated/v1_0/test-all/summary.json` records `status: pass`, `failed_count: 0`, per-step `exit_code: 0`, and is checked by `validate-test-all-summary`; this prevents accepting the release-gate command name as proxy evidence.",
			Result:      "partial",
		},
	}
	return renderBlockedOwnershipAudit(rows)
}

func stableForbiddenCaseDiagnosticsFixtureEvidence() string {
	return auditParagraph(
		"use-after-move/use-after-consume, partial struct/enum consume whole-value rejection, partial struct/enum whole-copy rejection, partial struct/enum enum-constructor rejection, optional payload consume/free whole-value rejection, actor/task use-after-transfer,",
		"same-module/cross-module actor struct-field/enum-payload alias transfer diagnostics with stable TETRA2101 JSON diagnostic evidence, same-module/cross-module task-handle struct-field/enum-payload alias transfer diagnostics with stable TETRA2101 JSON diagnostic evidence,",
		"task-handle struct-field/enum-payload alias use-after-transfer/join, task-group use-after-close, branch/match/loop actor consume reuse with stable branch actor CLI JSON, maybe-consumed joins,",
		"branch/match/loop task-handle maybe-joined, task-group maybe-closed, and island maybe-freed merge diagnostics; branch/match/loop resource finalization merge diagnostics with stable TETRA2101 JSON evidence,",
		"borrow escape, alias conflicts, use-after-free/join/close, resource use-after-free/double-join/ambiguous-provenance, island transfer non-local-payload, callable mutable-capture global/heap-escape, callable pointer/resource capture escape,",
		"function-typed storage/return unsupported capture rejection, callable global-storage escape, unsupported function-value escape, unsupported function-value call, capturing closure raw-ptr escape, captured closure explicit type-arg rejection, function-typed explicit type-arg rejection,",
		"generic closure/generic callback-closure capture, generic closure pointer/direct-call, and imported mutable function-typed global boundary JSON diagnostics, double-drop/double-finalization, callable escape diagnostics, and CLI JSON ownership/lifetime safety codes for borrow-escape including fixed-array alias return/global assignment/optional global assignment/inout assignment and borrowed string alias return/global assignment,",
		"slice-containing struct literal/alias/nested struct/enum-payload return and inout assignment escapes plus slice-containing enum direct/alias return escape CLI JSON evidence,",
		"slice-containing struct/enum owned/consume/inout call escape CLI JSON evidence including imported direct cases, same-module/cross-module generic borrow-aggregate/optional-ptr return diagnostics with stable TETRA2102 CLI JSON evidence,",
		"same-module/cross-module function-typed value/struct-field/enum-payload optional-ptr owned/consume/inout callback diagnostics with stable TETRA2101 CLI JSON evidence,",
		"function-typed value/struct-field/enum-payload callback slice-containing struct/enum owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence,",
		"ptr/slice optional assignment return/owned/consume/inout escape with stable same-module/cross-module slice optional assignment return/owned/consume/inout CLI JSON evidence exist.",
		"same-module/cross-module slice optional payload binding owned/consume/inout call, `inout` assignment, and global assignment CLI JSON evidence exists.",
		"same-module/cross-module direct slice global assignment with stable TETRA2102 JSON diagnostic evidence exists.",
		"same-module/cross-module optional ptr global assignment with stable TETRA2102 JSON diagnostic evidence exists.",
		"same-module/cross-module optional aggregate global assignment with stable TETRA2102 JSON diagnostic evidence exists.",
		"same-module/cross-module ptr optional assignment if-let/match global escape with stable TETRA2102 JSON diagnostic evidence exists.",
		"same-module/cross-module ptr enum alias return escape with stable TETRA2102 JSON diagnostic evidence exists.",
		"same-module/cross-module ptr-containing aggregate whole/field/alias/nested-field return escapes with stable TETRA2102 JSON diagnostic evidence exists.",
		"same-module/cross-module whole-aggregate global assignment with stable TETRA2102 JSON diagnostic evidence exists.",
		"same-module/cross-module ptr-containing enum whole-value global assignment with stable TETRA2102 JSON diagnostic evidence exists.",
		"same-module/cross-module global field target assignment with stable TETRA2102 JSON diagnostic evidence exists.",
		"same-module/cross-module aggregate and nested-aggregate global field escapes with stable TETRA2102 JSON diagnostic evidence exists.",
		"same-module/cross-module ptr enum-payload return/global/inout assignment escapes with stable TETRA2102 JSON diagnostic evidence exists.",
		"same-module/cross-module ptr optional-payload return/global/inout assignment escapes with stable TETRA2102 JSON diagnostic evidence exists.",
		"same-module/cross-module slice optional-payload inout/global assignment escapes with stable TETRA2102 JSON diagnostic evidence exists.",
		"same-module/cross-module nested slice enum-payload return/inout/global assignment escapes with stable TETRA2102 JSON diagnostic evidence exists.",
		"same-module/cross-module nested slice struct return/inout/global assignment escapes with stable TETRA2102 JSON diagnostic evidence exists.",
		"same-module/cross-module ptr-containing/nested aggregate owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence exists.",
		"same-module/cross-module ptr enum-payload owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence exists.",
	)
}

func renderBlockedOwnershipAudit(rows []ownershipAuditRow) string {
	var builder strings.Builder
	builder.WriteString(`# Tetra Ownership Production Audit

Status: not achieved.

## Prompt-To-Artifact Checklist

| Requirement | Required artifact or command | Current evidence | Result |
| --- | --- | --- | --- |
`)
	for _, row := range rows {
		builder.WriteString(markdownAuditRow(row))
	}
	builder.WriteString(`
## Missing Work Summary

The objective is not achieved. Remaining work includes interprocedural lifetime
analysis, broad alias/provenance tracking, and heap/global/thread escape
coverage for all ownership data.
`)
	return builder.String()
}

func markdownAuditRow(row ownershipAuditRow) string {
	return "| " + row.Requirement + " | " + row.Artifact + " | " + row.Evidence + " | " + row.Result + " |\n"
}

func auditParagraph(parts ...string) string {
	return strings.Join(parts, " ")
}

func replaceOwnershipAuditRowEvidence(t *testing.T, audit, requirement, evidence string) string {
	t.Helper()
	prefix := "| " + requirement + " |"
	lines := strings.Split(audit, "\n")
	for i, line := range lines {
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		cells := splitOwnershipAuditTableRow(line)
		if len(cells) != 4 {
			t.Fatalf("row %q has %d cells, want 4", requirement, len(cells))
		}
		cells[2] = evidence
		lines[i] = "| " + strings.Join(cells, " | ") + " |"
		return strings.Join(lines, "\n")
	}
	t.Fatalf("missing row %q", requirement)
	return ""
}
