package compiler

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/semantics"
)

const (
	firstClassCallableCoverageSchemaV1  = "tetra.language.first_class_callables.v1"
	firstClassCallableCoverageScopeP221 = "p22.1_first_class_callables_v1"

	firstClassCallableFnPtrWitnessID     = "bounded_one_capture_fnptr"
	firstClassCallableHandleWitnessID    = "nine_capture_handle"
	firstClassCallableInterfaceWitnessID = "cross_module_returned_handle_metadata"
)

type FirstClassCallableCoverageID string

const (
	FirstClassCallableFnPtrFastPath             FirstClassCallableCoverageID = "fnptr_fast_path"
	FirstClassCallableFatHandle                 FirstClassCallableCoverageID = "fat_callable_handle"
	FirstClassCallableCaptureSafetyClassifier   FirstClassCallableCoverageID = "capture_safety_classifier"
	FirstClassCallableMutableCaptureDiagnostics FirstClassCallableCoverageID = "mutable_capture_escape_diagnostics"
	FirstClassCallableResourceThreadDiagnostics FirstClassCallableCoverageID = "resource_thread_escape_diagnostics"
	FirstClassCallableFixedABIWidth             FirstClassCallableCoverageID = "fixed_abi_width"
	FirstClassCallableInterfaceMetadata         FirstClassCallableCoverageID = "cross_module_interface_metadata"
	FirstClassCallableStorageCallbackPaths      FirstClassCallableCoverageID = "storage_and_callback_paths"
)

type FirstClassCallableCoverageReport struct {
	SchemaVersion                             string                          `json:"schema_version"`
	Scope                                     string                          `json:"scope"`
	Rows                                      []FirstClassCallableCoverageRow `json:"rows"`
	Witnesses                                 []FirstClassCallableABIWitness  `json:"witnesses"`
	NonClaims                                 []string                        `json:"non_claims"`
	VariableABIWidthClaimed                   bool                            `json:"variable_abi_width_claimed"`
	ExplodingReturnSlotsClaimed               bool                            `json:"exploding_return_slots_claimed"`
	MutableByRefCaptureClaimed                bool                            `json:"mutable_by_ref_capture_claimed"`
	PointerResourceCaptureClaimed             bool                            `json:"pointer_resource_capture_claimed"`
	ThreadBoundaryCallableTransferClaimed     bool                            `json:"thread_boundary_callable_transfer_claimed"`
	RuntimeGenericCallablePolymorphismClaimed bool                            `json:"runtime_generic_callable_polymorphism_claimed"`
	DynamicCallableDispatchClaimed            bool                            `json:"dynamic_callable_dispatch_claimed"`
	UnsafeLifetimeRelaxationClaimed           bool                            `json:"unsafe_lifetime_relaxation_claimed"`
	PerformanceClaimed                        bool                            `json:"performance_claimed"`
	RuntimeBehaviorChanged                    bool                            `json:"runtime_behavior_changed"`
	SafeSemanticsChanged                      bool                            `json:"safe_semantics_changed"`
}

type FirstClassCallableCoverageRow struct {
	ID         FirstClassCallableCoverageID `json:"id"`
	Name       string                       `json:"name"`
	Status     string                       `json:"status"`
	Evidence   []string                     `json:"evidence"`
	Tests      []string                     `json:"tests"`
	Boundaries []string                     `json:"boundaries"`
	WitnessIDs []string                     `json:"witness_ids"`
}

type FirstClassCallableABIWitness struct {
	ID                         string `json:"id"`
	Kind                       string `json:"kind"`
	CaptureCount               int    `json:"capture_count"`
	FnPtrSlotCount             int    `json:"fnptr_slot_count"`
	CallableHandleSlotCount    int    `json:"callable_handle_slot_count"`
	LocalSlotCount             int    `json:"local_slot_count"`
	UsesHandle                 bool   `json:"uses_handle"`
	AllocBytesCount            int    `json:"alloc_bytes_count"`
	EnvWriteCount              int    `json:"env_write_count"`
	EnvReadCount               int    `json:"env_read_count"`
	CallArgSlots               int    `json:"call_arg_slots"`
	CallRetSlots               int    `json:"call_ret_slots"`
	ReturnSlots                int    `json:"return_slots"`
	FunctionEscapeKind         string `json:"function_escape_kind"`
	InterfaceMetadataPreserved bool   `json:"interface_metadata_preserved"`
}

func BuildP22FirstClassCallableCoverage() (FirstClassCallableCoverageReport, error) {
	fnptr, err := buildP22FnPtrCallableWitness()
	if err != nil {
		return FirstClassCallableCoverageReport{}, err
	}
	handle, err := buildP22HandleCallableWitness()
	if err != nil {
		return FirstClassCallableCoverageReport{}, err
	}
	iface, err := buildP22InterfaceCallableWitness()
	if err != nil {
		return FirstClassCallableCoverageReport{}, err
	}

	report := FirstClassCallableCoverageReport{
		SchemaVersion: firstClassCallableCoverageSchemaV1,
		Scope:         firstClassCallableCoverageScopeP221,
		Witnesses:     []FirstClassCallableABIWitness{fnptr, handle, iface},
		Rows: []FirstClassCallableCoverageRow{
			p22FirstClassCallableRow(FirstClassCallableFnPtrFastPath, "Bounded fnptr fast path", "current_evidence",
				[]string{
					"compiler/internal/semantics/types.go fixes FnPtrEnvSlotCount = 8 and FnPtrSlotCount = 9 for the compact fnptr representation.",
					"Parse/Check/Lower witness bounded_one_capture_fnptr records a one-capture fnptr value with 9-slot local metadata and no heap environment allocation.",
					"compiler/internal/lower/callables.go emits IRSymAddr plus padded environment slots for fnptr values without IRAllocBytes in the bounded witness.",
				},
				[]string{
					"go test ./compiler -run 'P22FirstClassCallable|ValidateP22FirstClassCallable'",
					"go test ./compiler/internal/lower -run 'Callable'",
				},
				[]string{
					"fnptr fast path is bounded to safe captures whose environment fits within FnPtrEnvSlotCount = 8",
					"no variable-width fnptr ABI is claimed",
					"no heap environment is allocated for the bounded fnptr witness",
				},
				[]string{firstClassCallableFnPtrWitnessID}),
			p22FirstClassCallableRow(FirstClassCallableFatHandle, "Fat callable handle for larger captures", "current_evidence",
				[]string{
					"compiler/internal/semantics/types.go fixes CallableHandleSlotCount = 4 for callable handles.",
					"Parse/Check/Lower witness nine_capture_handle records a nine-capture callable with a 4-slot handle local and heap escape metadata.",
					"compiler/internal/lower/callables.go emits IRAllocBytes, IRMemWritePtrOffset, and IRMemReadPtrOffset for the handle witness while calling the closure with explicit argument plus 9 env slots.",
				},
				[]string{
					"go test ./compiler -run 'P22FirstClassCallable|ValidateP22FirstClassCallable'",
					"go test ./compiler/internal/lower -run 'Callable'",
				},
				[]string{
					"larger safe immutable captures use the fixed handle path",
					"callable returns and locals do not explode beyond the 4-slot handle",
					"handle lowering evidence is IR-only; no performance claim is made",
				},
				[]string{firstClassCallableHandleWitnessID}),
			p22FirstClassCallableRow(FirstClassCallableCaptureSafetyClassifier, "Capture safety classifier", "current_evidence",
				[]string{
					"compiler/internal/semantics/callable_escape.go classifies local, return, global, callback, and thread callable escape boundaries.",
					"compiler/internal/semantics/closure_captures.go restricts escaping callable captures to safe immutable by-value Int/Bool/String/simple aggregate payloads.",
					"docs/spec/current_supported_surface.md records the safe immutable by-value callable capture boundary.",
				},
				[]string{
					"go test ./compiler/internal/semantics -run 'Callable|Closure|FunctionType'",
					"go test ./compiler/tests/semantics -run 'Callable|Closure|FunctionType|Interface'",
				},
				[]string{
					"generic closure captures remain outside this report",
					"surface ephemeral values cannot escape through callable capture",
					"safe immutable by-value captures are the promoted subset",
				},
				[]string{firstClassCallableFnPtrWitnessID, firstClassCallableHandleWitnessID}),
			p22FirstClassCallableRow(FirstClassCallableMutableCaptureDiagnostics, "Mutable capture escape diagnostics", "current_evidence",
				[]string{
					"compiler/internal/semantics/callable_escape.go rejects mutable by-reference capture when a callable crosses heap-escape or global-escape boundaries.",
					"compiler/internal/semantics/callable_escape_test.go covers mutable global-escape and thread-boundary rejection cases.",
					"docs/spec/current_supported_surface.md documents mutable by-reference capture as a diagnostic, not a supported escape model.",
				},
				[]string{
					"go test ./compiler/internal/semantics -run 'Callable|Closure|FunctionType'",
					"go test ./compiler/tests/semantics -run 'Callable|Closure|FunctionType'",
				},
				[]string{
					"mutable by-reference capture support is not claimed",
					"global-escape mutable capture remains diagnostic",
					"heap-escape mutable capture remains diagnostic",
				},
				[]string{firstClassCallableHandleWitnessID}),
			p22FirstClassCallableRow(FirstClassCallableResourceThreadDiagnostics, "Resource and thread escape diagnostics", "current_evidence",
				[]string{
					"compiler/internal/semantics/callable_escape.go rejects pointer/resource capture escape and classifies thread-boundary callable escape separately.",
					"compiler/internal/semantics/callable_escape_test.go covers pointer/resource capture and thread-boundary callable escape rejection.",
					"docs/spec/current_supported_surface.md keeps pointer/resource capture and thread-boundary callable escape outside the supported callable model.",
				},
				[]string{
					"go test ./compiler/internal/semantics -run 'Callable|Closure|FunctionType'",
					"go test ./compiler/tests/semantics -run 'Callable|Closure|FunctionType'",
				},
				[]string{
					"pointer/resource capture support is not claimed",
					"thread-boundary callable escape is rejected without sync/ownership transfer evidence",
					"no unsafe lifetime relaxation is claimed",
				},
				[]string{firstClassCallableHandleWitnessID}),
			p22FirstClassCallableRow(FirstClassCallableFixedABIWidth, "Fixed callable ABI width", "current_evidence",
				[]string{
					"compiler/internal/semantics/types.go declares FnPtrEnvSlotCount = 8, FnPtrSlotCount = 9, and CallableHandleSlotCount = 4.",
					"The bounded fnptr witness records FnPtrSlotCount = 9 and CallableHandleSlotCount = 4 without heap allocation.",
					"The nine-capture handle witness records CallableHandleSlotCount = 4, ReturnSlots = 4, and call ArgSlots = 10 RetSlots = 1 for the closure dispatch.",
				},
				[]string{
					"go test ./compiler -run 'P22FirstClassCallable|ValidateP22FirstClassCallable'",
					"go test ./compiler/internal/lower -run 'Callable'",
				},
				[]string{
					"fixed ABI width is evidence, not a new runtime mode",
					"no variable-width callable ABI is claimed",
					"no exploding callable return slots are claimed",
				},
				[]string{firstClassCallableFnPtrWitnessID, firstClassCallableHandleWitnessID, firstClassCallableInterfaceWitnessID}),
			p22FirstClassCallableRow(FirstClassCallableInterfaceMetadata, "Cross-module interface metadata", "current_evidence",
				[]string{
					"compiler/interface.go preserves returned function handle metadata in generated .t4i stubs.",
					"compiler/tests/semantics/interface_test.go verifies ReturnFunctionHandleValue, heap escape metadata, and ReturnSlots = 4 for returned nine-capture callables.",
					"ParseFile/CheckWorld witness cross_module_returned_handle_metadata records generated .t4i metadata preserved for a returned nine-capture handle.",
				},
				[]string{
					"go test ./compiler/tests/semantics -run 'Interface'",
					"go test ./compiler -run 'P22FirstClassCallable|ValidateP22FirstClassCallable'",
				},
				[]string{
					".t4i metadata is checked evidence for returned callable handles",
					"cross-module metadata preservation does not add dynamic callable dispatch",
					"ReturnFunctionHandleValue and ReturnSlots = 4 are required for handle returns",
				},
				[]string{firstClassCallableInterfaceWitnessID}),
			p22FirstClassCallableRow(FirstClassCallableStorageCallbackPaths, "Storage, callback, and return paths", "current_evidence",
				[]string{
					"docs/spec/current_supported_surface.md records aliases, struct fields, enum payloads, callback arguments, returns, and same-module global snapshots for safe callable values.",
					"compiler/tests/semantics/closures_semantic_clauses_test.go covers returned, struct-field, enum-payload, and callback handle movement with nine captured values.",
					"compiler/internal/lower/callables.go routes stable callable targets through direct dispatch while handle values carry fixed-width environment metadata.",
				},
				[]string{
					"go test ./compiler/tests/semantics -run 'Callable|Closure|FunctionType|Interface'",
					"go test ./compiler/internal/lower -run 'Callable|FunctionType'",
				},
				[]string{
					"aliases, struct fields, enum payloads, callback arguments, and returns are covered only for safe by-value callable captures",
					"runtime generic callable polymorphism is not claimed",
					"dynamic callable dispatch is not claimed",
				},
				[]string{firstClassCallableFnPtrWitnessID, firstClassCallableHandleWitnessID, firstClassCallableInterfaceWitnessID}),
		},
		NonClaims: p22FirstClassCallableNonClaims(),
	}
	return report, nil
}

func ValidateP22FirstClassCallableCoverage(report FirstClassCallableCoverageReport) error {
	if report.SchemaVersion != firstClassCallableCoverageSchemaV1 {
		return fmt.Errorf("first-class callable coverage schema = %q, want %q", report.SchemaVersion, firstClassCallableCoverageSchemaV1)
	}
	if report.Scope != firstClassCallableCoverageScopeP221 {
		return fmt.Errorf("first-class callable coverage scope = %q, want %q", report.Scope, firstClassCallableCoverageScopeP221)
	}
	if report.VariableABIWidthClaimed {
		return fmt.Errorf("first-class callable coverage: variable-width ABI claim is forbidden")
	}
	if report.ExplodingReturnSlotsClaimed {
		return fmt.Errorf("first-class callable coverage: exploding return slots claim is forbidden")
	}
	if report.MutableByRefCaptureClaimed {
		return fmt.Errorf("first-class callable coverage: mutable by-reference capture claim is forbidden")
	}
	if report.PointerResourceCaptureClaimed {
		return fmt.Errorf("first-class callable coverage: pointer/resource capture claim is forbidden")
	}
	if report.ThreadBoundaryCallableTransferClaimed {
		return fmt.Errorf("first-class callable coverage: thread-boundary callable transfer claim is forbidden")
	}
	if report.RuntimeGenericCallablePolymorphismClaimed {
		return fmt.Errorf("first-class callable coverage: runtime generic callable polymorphism claim is forbidden")
	}
	if report.DynamicCallableDispatchClaimed {
		return fmt.Errorf("first-class callable coverage: dynamic callable dispatch claim is forbidden")
	}
	if report.UnsafeLifetimeRelaxationClaimed {
		return fmt.Errorf("first-class callable coverage: unsafe lifetime relaxation claim is forbidden")
	}
	if report.PerformanceClaimed {
		return fmt.Errorf("first-class callable coverage: performance claim is forbidden")
	}
	if report.RuntimeBehaviorChanged {
		return fmt.Errorf("first-class callable coverage: runtime behavior change claim is forbidden")
	}
	if report.SafeSemanticsChanged {
		return fmt.Errorf("first-class callable coverage: safe-program semantics change is forbidden")
	}
	for _, nonClaim := range p22FirstClassCallableNonClaims() {
		if !p22FirstClassCallableReportHasString(report.NonClaims, nonClaim) {
			return fmt.Errorf("first-class callable coverage: missing non-claim %q", nonClaim)
		}
	}
	if err := validateP22FirstClassCallableStrings("non-claim", report.NonClaims); err != nil {
		return err
	}

	witnesses := map[string]FirstClassCallableABIWitness{}
	for _, witness := range report.Witnesses {
		if strings.TrimSpace(witness.ID) == "" || strings.TrimSpace(witness.Kind) == "" {
			return fmt.Errorf("first-class callable coverage: witness missing required metadata: %#v", witness)
		}
		if _, ok := witnesses[witness.ID]; ok {
			return fmt.Errorf("first-class callable coverage: duplicate witness %s", witness.ID)
		}
		witnesses[witness.ID] = witness
	}
	for _, id := range []string{firstClassCallableFnPtrWitnessID, firstClassCallableHandleWitnessID, firstClassCallableInterfaceWitnessID} {
		if _, ok := witnesses[id]; !ok {
			return fmt.Errorf("first-class callable coverage: missing witness %s", id)
		}
	}
	if err := validateP22FirstClassCallableFnPtrWitness(witnesses[firstClassCallableFnPtrWitnessID]); err != nil {
		return err
	}
	if err := validateP22FirstClassCallableHandleWitness(witnesses[firstClassCallableHandleWitnessID]); err != nil {
		return err
	}
	if err := validateP22FirstClassCallableInterfaceWitness(witnesses[firstClassCallableInterfaceWitnessID]); err != nil {
		return err
	}

	expected := map[FirstClassCallableCoverageID]bool{}
	for _, id := range p22FirstClassCallableCoverageIDs() {
		expected[id] = true
	}
	seen := map[FirstClassCallableCoverageID]bool{}
	for _, row := range report.Rows {
		if row.ID == "" || strings.TrimSpace(row.Name) == "" || strings.TrimSpace(row.Status) == "" {
			return fmt.Errorf("first-class callable coverage: row missing required metadata: %#v", row)
		}
		if !expected[row.ID] {
			return fmt.Errorf("first-class callable coverage: unexpected row %s", row.ID)
		}
		if seen[row.ID] {
			return fmt.Errorf("first-class callable coverage: duplicate row %s", row.ID)
		}
		seen[row.ID] = true
		if err := validateP22FirstClassCallableStrings("row "+string(row.ID)+" evidence", row.Evidence); err != nil {
			return err
		}
		if err := validateP22FirstClassCallableStrings("row "+string(row.ID)+" tests", row.Tests); err != nil {
			return err
		}
		if err := validateP22FirstClassCallableStrings("row "+string(row.ID)+" boundaries", row.Boundaries); err != nil {
			return err
		}
		if len(row.WitnessIDs) == 0 {
			return fmt.Errorf("first-class callable coverage: row %s missing witness reference", row.ID)
		}
		for _, id := range row.WitnessIDs {
			if _, ok := witnesses[id]; !ok {
				return fmt.Errorf("first-class callable coverage: row %s references missing witness %s", row.ID, id)
			}
		}
	}
	for _, id := range p22FirstClassCallableCoverageIDs() {
		if !seen[id] {
			return fmt.Errorf("first-class callable coverage: missing row %s", id)
		}
	}
	return nil
}

func p22FirstClassCallableCoverageIDs() []FirstClassCallableCoverageID {
	return []FirstClassCallableCoverageID{
		FirstClassCallableFnPtrFastPath,
		FirstClassCallableFatHandle,
		FirstClassCallableCaptureSafetyClassifier,
		FirstClassCallableMutableCaptureDiagnostics,
		FirstClassCallableResourceThreadDiagnostics,
		FirstClassCallableFixedABIWidth,
		FirstClassCallableInterfaceMetadata,
		FirstClassCallableStorageCallbackPaths,
	}
}

func p22FirstClassCallableNonClaims() []string {
	return []string{
		"no variable-width callable ABI is claimed",
		"no exploding callable return slots are claimed",
		"no mutable by-reference capture support is claimed",
		"no pointer/resource capture support is claimed",
		"no thread-boundary callable transfer is claimed",
		"no runtime generic callable polymorphism is claimed",
		"no dynamic callable dispatch is claimed",
		"no unsafe lifetime relaxation is claimed",
		"no performance claim is made",
		"no runtime behavior change beyond the existing callable ABI is claimed",
		"safe-program semantics do not change",
	}
}

func p22FirstClassCallableRow(id FirstClassCallableCoverageID, name, status string, evidence, tests, boundaries, witnessIDs []string) FirstClassCallableCoverageRow {
	return FirstClassCallableCoverageRow{
		ID:         id,
		Name:       name,
		Status:     status,
		Evidence:   append([]string{}, evidence...),
		Tests:      append([]string{}, tests...),
		Boundaries: append([]string{}, boundaries...),
		WitnessIDs: append([]string{}, witnessIDs...),
	}
}

func buildP22FnPtrCallableWitness() (FirstClassCallableABIWitness, error) {
	checked, prog, err := p22ParseCheckLowerCallable(`
func main() -> Int:
    let base: Int = 1
    let cb: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + base
    return cb(41)
`)
	if err != nil {
		return FirstClassCallableABIWitness{}, err
	}
	main, ok := p22FindCheckedFunc(checked, "main")
	if !ok {
		return FirstClassCallableABIWitness{}, fmt.Errorf("first-class callable coverage: fnptr witness missing checked main")
	}
	cb := main.Locals["cb"]
	fn, ok := p22FindIRFunc(prog, "main")
	if !ok {
		return FirstClassCallableABIWitness{}, fmt.Errorf("first-class callable coverage: fnptr witness missing lowered main")
	}
	argSlots, retSlots := p22FirstMatchingCallSlots(fn, 2, 1)
	return FirstClassCallableABIWitness{
		ID:                      firstClassCallableFnPtrWitnessID,
		Kind:                    "fnptr_fast_path",
		CaptureCount:            p22CallableCaptureCount(cb),
		FnPtrSlotCount:          semantics.FnPtrSlotCount,
		CallableHandleSlotCount: semantics.CallableHandleSlotCount,
		LocalSlotCount:          cb.SlotCount,
		UsesHandle:              cb.FunctionHandleValue,
		AllocBytesCount:         p22CountIRKind(fn, ir.IRAllocBytes),
		EnvWriteCount:           p22CountIRKind(fn, ir.IRMemWritePtrOffset),
		EnvReadCount:            p22CountIRKind(fn, ir.IRMemReadPtrOffset),
		CallArgSlots:            argSlots,
		CallRetSlots:            retSlots,
		ReturnSlots:             semantics.FnPtrSlotCount,
		FunctionEscapeKind:      string(cb.FunctionEscapeKind),
	}, nil
}

func buildP22HandleCallableWitness() (FirstClassCallableABIWitness, error) {
	checked, prog, err := p22ParseCheckLowerCallable(`
func main() -> Int:
    let one: Int = 1
    let two: Int = 2
    let three: Int = 3
    let four: Int = 4
    let five: Int = 5
    let six: Int = 6
    let seven: Int = 7
    let eight: Int = 8
    let nine: Int = 9
    let cb: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + one + two + three + four + five + six + seven + eight + nine
    return cb(-3)
`)
	if err != nil {
		return FirstClassCallableABIWitness{}, err
	}
	main, ok := p22FindCheckedFunc(checked, "main")
	if !ok {
		return FirstClassCallableABIWitness{}, fmt.Errorf("first-class callable coverage: handle witness missing checked main")
	}
	cb := main.Locals["cb"]
	fn, ok := p22FindIRFunc(prog, "main")
	if !ok {
		return FirstClassCallableABIWitness{}, fmt.Errorf("first-class callable coverage: handle witness missing lowered main")
	}
	argSlots, retSlots := p22FirstMatchingCallSlots(fn, 10, 1)
	return FirstClassCallableABIWitness{
		ID:                      firstClassCallableHandleWitnessID,
		Kind:                    "fat_callable_handle",
		CaptureCount:            p22CallableCaptureCount(cb),
		FnPtrSlotCount:          semantics.FnPtrSlotCount,
		CallableHandleSlotCount: semantics.CallableHandleSlotCount,
		LocalSlotCount:          cb.SlotCount,
		UsesHandle:              cb.FunctionHandleValue,
		AllocBytesCount:         p22CountIRKind(fn, ir.IRAllocBytes),
		EnvWriteCount:           p22CountIRKind(fn, ir.IRMemWritePtrOffset),
		EnvReadCount:            p22CountIRKind(fn, ir.IRMemReadPtrOffset),
		CallArgSlots:            argSlots,
		CallRetSlots:            retSlots,
		ReturnSlots:             semantics.CallableHandleSlotCount,
		FunctionEscapeKind:      string(cb.FunctionEscapeKind),
	}, nil
}

func buildP22InterfaceCallableWitness() (FirstClassCallableABIWitness, error) {
	src := []byte(`module lib.maker

pub func make() -> fn(Int) -> Int:
    let one: Int = 1
    let two: Int = 2
    let three: Int = 3
    let four: Int = 4
    let five: Int = 5
    let six: Int = 6
    let seven: Int = 7
    let eight: Int = 8
    let nine: Int = 9
    return fn(x: Int) -> Int:
        return x + one + two + three + four + five + six + seven + eight + nine
`)
	iface, err := GenerateInterfaceFromSource(src, "lib/maker.t4")
	if err != nil {
		return FirstClassCallableABIWitness{}, fmt.Errorf("first-class callable coverage: generate interface witness: %w", err)
	}
	maker, err := ParseFile(iface, "lib/maker.t4i")
	if err != nil {
		return FirstClassCallableABIWitness{}, fmt.Errorf("first-class callable coverage: parse generated interface witness: %w", err)
	}
	app, err := ParseFile([]byte(`module app.main
import lib.maker as maker

func main() -> Int:
    let cb: fn(Int) -> Int = maker.make()
    return cb(-3)
`), "app/main.t4")
	if err != nil {
		return FirstClassCallableABIWitness{}, fmt.Errorf("first-class callable coverage: parse app witness: %w", err)
	}
	checked, err := CheckWorld(&World{
		EntryModule:      "app.main",
		Files:            []*FileAST{maker, app},
		InterfaceModules: map[string]bool{"lib.maker": true},
		ByModule: map[string]*FileAST{
			"lib.maker": maker,
			"app.main":  app,
		},
	})
	if err != nil {
		return FirstClassCallableABIWitness{}, fmt.Errorf("first-class callable coverage: check interface witness: %w", err)
	}
	makeSig := checked.FuncSigs["lib.maker.make"]
	main, ok := p22FindCheckedFunc(checked, "app.main.main")
	if !ok {
		return FirstClassCallableABIWitness{}, fmt.Errorf("first-class callable coverage: interface witness missing app.main.main")
	}
	cb := main.Locals["cb"]
	return FirstClassCallableABIWitness{
		ID:                         firstClassCallableInterfaceWitnessID,
		Kind:                       "cross_module_interface_metadata",
		CaptureCount:               len(makeSig.ReturnFunctionCaptures),
		FnPtrSlotCount:             semantics.FnPtrSlotCount,
		CallableHandleSlotCount:    semantics.CallableHandleSlotCount,
		LocalSlotCount:             cb.SlotCount,
		UsesHandle:                 makeSig.ReturnFunctionHandleValue && cb.FunctionHandleValue,
		ReturnSlots:                makeSig.ReturnSlots,
		FunctionEscapeKind:         string(makeSig.ReturnFunctionEscapeKind),
		InterfaceMetadataPreserved: makeSig.ReturnFunctionSymbol != "" && len(makeSig.ReturnFunctionCaptures) == 9 && cb.FunctionHandleValue,
	}, nil
}

func p22ParseCheckLowerCallable(src string) (*CheckedProgram, *IRProgram, error) {
	prog, err := Parse([]byte(src))
	if err != nil {
		return nil, nil, fmt.Errorf("first-class callable coverage: parse witness: %w", err)
	}
	checked, err := Check(prog)
	if err != nil {
		return nil, nil, fmt.Errorf("first-class callable coverage: check witness: %w", err)
	}
	lowered, err := Lower(checked)
	if err != nil {
		return nil, nil, fmt.Errorf("first-class callable coverage: lower witness: %w", err)
	}
	return checked, lowered, nil
}

func p22FindCheckedFunc(checked *CheckedProgram, name string) (semantics.CheckedFunc, bool) {
	for _, fn := range checked.Funcs {
		if fn.Name == name {
			return fn, true
		}
	}
	return semantics.CheckedFunc{}, false
}

func p22FindIRFunc(prog *IRProgram, name string) (ir.IRFunc, bool) {
	for _, fn := range prog.Funcs {
		if fn.Name == name {
			return fn, true
		}
	}
	return ir.IRFunc{}, false
}

func p22CountIRKind(fn ir.IRFunc, kind ir.IRInstrKind) int {
	count := 0
	for _, instr := range fn.Instrs {
		if instr.Kind == kind {
			count++
		}
	}
	return count
}

func p22FirstMatchingCallSlots(fn ir.IRFunc, argSlots, retSlots int) (int, int) {
	for _, instr := range fn.Instrs {
		if instr.Kind == ir.IRCall && instr.ArgSlots == argSlots && instr.RetSlots == retSlots {
			return instr.ArgSlots, instr.RetSlots
		}
	}
	return 0, 0
}

func p22CallableCaptureCount(local semantics.LocalInfo) int {
	return len(local.FunctionCaptures) + len(local.FunctionEscapeCaptures)
}

func validateP22FirstClassCallableFnPtrWitness(witness FirstClassCallableABIWitness) error {
	if witness.CaptureCount != 1 || witness.UsesHandle || witness.FnPtrSlotCount != semantics.FnPtrSlotCount || witness.LocalSlotCount != semantics.FnPtrSlotCount {
		return fmt.Errorf("first-class callable coverage: fnptr witness drift: %#v", witness)
	}
	if witness.CallableHandleSlotCount != semantics.CallableHandleSlotCount || witness.AllocBytesCount != 0 || witness.EnvWriteCount != 0 || witness.EnvReadCount != 0 {
		return fmt.Errorf("first-class callable coverage: fnptr witness allocated heap env or lost fixed ABI: %#v", witness)
	}
	return nil
}

func validateP22FirstClassCallableHandleWitness(witness FirstClassCallableABIWitness) error {
	if witness.CaptureCount != 9 || !witness.UsesHandle || witness.LocalSlotCount != semantics.CallableHandleSlotCount {
		return fmt.Errorf("first-class callable coverage: handle witness drift: %#v", witness)
	}
	if witness.FnPtrSlotCount != semantics.FnPtrSlotCount || witness.CallableHandleSlotCount != semantics.CallableHandleSlotCount {
		return fmt.Errorf("first-class callable coverage: fixed ABI drift in handle witness: %#v", witness)
	}
	if witness.AllocBytesCount != 1 || witness.EnvWriteCount != 9 || witness.EnvReadCount != 9 || witness.CallArgSlots != 10 || witness.CallRetSlots != 1 {
		return fmt.Errorf("first-class callable coverage: handle witness IR drift: %#v", witness)
	}
	if witness.ReturnSlots != semantics.CallableHandleSlotCount || witness.FunctionEscapeKind != string(semantics.CallableEscapeHeap) {
		return fmt.Errorf("first-class callable coverage: handle witness escape/return metadata drift: %#v", witness)
	}
	return nil
}

func validateP22FirstClassCallableInterfaceWitness(witness FirstClassCallableABIWitness) error {
	if witness.CaptureCount != 9 || !witness.UsesHandle || !witness.InterfaceMetadataPreserved {
		return fmt.Errorf("first-class callable coverage: interface witness drift: %#v", witness)
	}
	if witness.CallableHandleSlotCount != semantics.CallableHandleSlotCount || witness.LocalSlotCount != semantics.CallableHandleSlotCount || witness.ReturnSlots != semantics.CallableHandleSlotCount {
		return fmt.Errorf("first-class callable coverage: interface fixed ABI drift: %#v", witness)
	}
	if witness.FunctionEscapeKind != string(semantics.CallableEscapeHeap) {
		return fmt.Errorf("first-class callable coverage: interface witness escape metadata drift: %#v", witness)
	}
	return nil
}

func validateP22FirstClassCallableStrings(label string, items []string) error {
	if len(items) == 0 {
		return fmt.Errorf("first-class callable coverage: %s missing", label)
	}
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			return fmt.Errorf("first-class callable coverage: %s contains empty item", label)
		}
		if p22FirstClassCallableContainsPlaceholder(trimmed) {
			return fmt.Errorf("first-class callable coverage: %s contains placeholder evidence: %q", label, item)
		}
	}
	return nil
}

func p22FirstClassCallableContainsPlaceholder(text string) bool {
	lower := strings.ToLower(text)
	for _, token := range []string{"todo", "tbd", "placeholder", "fixme", "???"} {
		if strings.Contains(lower, token) {
			return true
		}
	}
	return false
}

func p22FirstClassCallableReportHasString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
