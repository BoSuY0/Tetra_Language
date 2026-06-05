package compiler

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/opt"
)

const (
	protocolTraitObjectDecisionSchemaV1       = "tetra.language.protocol_trait_object_decision.v1"
	protocolTraitObjectDecisionScopeP222      = "p22.2_protocol_trait_object_decision"
	protocolTraitObjectDecisionKeepStaticOnly = "keep_static_conformance_only"

	protocolTraitStaticConformanceWitnessID    = "static_conformance_direct_call"
	protocolTraitProtocolBoundGenericWitnessID = "protocol_bound_generic_monomorphized_call"
	protocolTraitRuntimeBoundaryWitnessID      = "runtime_protocol_value_and_requirement_call_rejections"
	protocolTraitSpecializationWitnessID       = "p17_p21_static_specialization_boundaries"
)

type ProtocolTraitObjectDecisionID string

const (
	ProtocolTraitStaticConformanceFastPath       ProtocolTraitObjectDecisionID = "static_conformance_fast_path"
	ProtocolTraitStaticProtocolBoundGenerics     ProtocolTraitObjectDecisionID = "static_protocol_bound_generics"
	ProtocolTraitRuntimeExistentialDecision      ProtocolTraitObjectDecisionID = "runtime_existential_decision"
	ProtocolTraitExplicitDynamicDispatchGate     ProtocolTraitObjectDecisionID = "explicit_dynamic_dispatch_gate"
	ProtocolTraitSpecializationStaticAbstraction ProtocolTraitObjectDecisionID = "specialization_static_abstraction"
	ProtocolTraitWitnessTableBoundary            ProtocolTraitObjectDecisionID = "witness_table_boundary"
	ProtocolTraitTraitObjectBoundary             ProtocolTraitObjectDecisionID = "trait_object_boundary"
	ProtocolTraitRegistryDocsAlignment           ProtocolTraitObjectDecisionID = "registry_docs_alignment"
)

type ProtocolTraitObjectDecisionReport struct {
	SchemaVersion                  string                           `json:"schema_version"`
	Scope                          string                           `json:"scope"`
	Decision                       string                           `json:"decision"`
	Rows                           []ProtocolTraitObjectDecisionRow `json:"rows"`
	Witnesses                      []ProtocolTraitObjectWitness     `json:"witnesses"`
	NonClaims                      []string                         `json:"non_claims"`
	RuntimeExistentialsPromoted    bool                             `json:"runtime_existentials_promoted"`
	TraitObjectsPromoted           bool                             `json:"trait_objects_promoted"`
	WitnessTablesPromoted          bool                             `json:"witness_tables_promoted"`
	DynamicDispatchPromoted        bool                             `json:"dynamic_dispatch_promoted"`
	ConformanceTableLookupPromoted bool                             `json:"conformance_table_lookup_promoted"`
	RuntimeProtocolValuesPromoted  bool                             `json:"runtime_protocol_values_promoted"`
	BroadSpecializationClaimed     bool                             `json:"broad_specialization_claimed"`
	PerformanceClaimed             bool                             `json:"performance_claimed"`
	RuntimeBehaviorChanged         bool                             `json:"runtime_behavior_changed"`
	SafeSemanticsChanged           bool                             `json:"safe_semantics_changed"`
}

type ProtocolTraitObjectDecisionRow struct {
	ID         ProtocolTraitObjectDecisionID `json:"id"`
	Name       string                        `json:"name"`
	Status     string                        `json:"status"`
	Decision   string                        `json:"decision"`
	Evidence   []string                      `json:"evidence"`
	Tests      []string                      `json:"tests"`
	Boundaries []string                      `json:"boundaries"`
	WitnessIDs []string                      `json:"witness_ids"`
}

type ProtocolTraitObjectWitness struct {
	ID                               string `json:"id"`
	Kind                             string `json:"kind"`
	ProtocolCount                    int    `json:"protocol_count"`
	ImplCount                        int    `json:"impl_count"`
	HasStaticMethodSig               bool   `json:"has_static_method_sig"`
	DirectCallTarget                 string `json:"direct_call_target"`
	MonomorphizedSig                 string `json:"monomorphized_sig"`
	MonomorphizedSigConcrete         bool   `json:"monomorphized_sig_concrete"`
	LoweredDirectCall                bool   `json:"lowered_direct_call"`
	RuntimeProtocolValueDiagnostic   string `json:"runtime_protocol_value_diagnostic"`
	GenericRequirementCallDiagnostic string `json:"generic_requirement_call_diagnostic"`
	InliningSchema                   string `json:"inlining_schema"`
	MachineSchema                    string `json:"machine_schema"`
	KnownDirectSymbolEvidence        bool   `json:"known_direct_symbol_evidence"`
	SpecializationNoDynamicDispatch  bool   `json:"specialization_no_dynamic_dispatch"`
	MachineNoOpCall                  bool   `json:"machine_no_op_call"`
}

func BuildP22ProtocolTraitObjectDecision() (ProtocolTraitObjectDecisionReport, error) {
	staticWitness, err := buildP22ProtocolStaticConformanceWitness()
	if err != nil {
		return ProtocolTraitObjectDecisionReport{}, err
	}
	genericWitness, err := buildP22ProtocolBoundGenericWitness()
	if err != nil {
		return ProtocolTraitObjectDecisionReport{}, err
	}
	runtimeBoundaryWitness, err := buildP22ProtocolRuntimeBoundaryWitness()
	if err != nil {
		return ProtocolTraitObjectDecisionReport{}, err
	}
	specializationWitness, err := buildP22ProtocolSpecializationWitness()
	if err != nil {
		return ProtocolTraitObjectDecisionReport{}, err
	}

	report := ProtocolTraitObjectDecisionReport{
		SchemaVersion: protocolTraitObjectDecisionSchemaV1,
		Scope:         protocolTraitObjectDecisionScopeP222,
		Decision:      protocolTraitObjectDecisionKeepStaticOnly,
		Witnesses: []ProtocolTraitObjectWitness{
			staticWitness,
			genericWitness,
			runtimeBoundaryWitness,
			specializationWitness,
		},
		Rows: []ProtocolTraitObjectDecisionRow{
			p22ProtocolTraitRow(ProtocolTraitStaticConformanceFastPath, "Static conformance fast path", "current_static_only", protocolTraitObjectDecisionKeepStaticOnly,
				[]string{
					"compiler/internal/semantics/checker.go stores protocols separately from value types and validates impl Type: Protocol clauses through compareProtocolRequirement.",
					"Static witness static_conformance_direct_call records one protocol, one impl, a Vec2.draw FuncSig, and a known direct IRCall to Vec2.draw after Parse/Check/Lower.",
					"compiler/tests/semantics/protocol_conformance_test.go covers extension/static method conformance, throws, ownership, effects, generic requirement shape, and imported extension clauses.",
				},
				[]string{
					"go test ./compiler -run 'P22ProtocolTrait|ValidateP22ProtocolTrait'",
					"go test ./compiler/tests/semantics -run 'ProtocolConformance'",
				},
				[]string{
					"static conformance remains the fast path",
					"known direct IRCall evidence is required for static dispatch claims",
					"no runtime protocol values, trait objects, witness tables, or dynamic dispatch are promoted",
				},
				[]string{protocolTraitStaticConformanceWitnessID}),
			p22ProtocolTraitRow(ProtocolTraitStaticProtocolBoundGenerics, "Static protocol-bound generics", "current_static_only", protocolTraitObjectDecisionKeepStaticOnly,
				[]string{
					"compiler/internal/semantics/checker.go validateGenericFuncDecl validates protocol bounds during monomorphization and rejects non-protocol, unknown, or private bounds.",
					"Static generic witness protocol_bound_generic_monomorphized_call records concrete id__T_Vec2 monomorphization and a direct call to id__T_Vec2 after Parse/Check/Lower.",
					"compiler/tests/semantics/generics_test.go covers same-module and cross-module protocol-bound conformance and stable rejection diagnostics.",
				},
				[]string{
					"go test ./compiler/tests/semantics -run 'GenericFunctionProtocolBound'",
					"go test ./compiler -run 'P22ProtocolTrait|ValidateP22ProtocolTrait'",
				},
				[]string{
					"protocol-bound generics are validated statically during monomorphization",
					"no runtime generic values are introduced",
					"generic-bound requirement calls remain unsupported until a report-visible dispatch model exists",
				},
				[]string{protocolTraitProtocolBoundGenericWitnessID, protocolTraitRuntimeBoundaryWitnessID}),
			p22ProtocolTraitRow(ProtocolTraitRuntimeExistentialDecision, "Runtime existential decision", "not_promoted", protocolTraitObjectDecisionKeepStaticOnly,
				[]string{
					"P22.2 decision is keep_static_conformance_only: runtime existential ABI is not designed in this slice.",
					"Runtime boundary witness records protocol runtime value rejection with unknown type 'Drawable' because protocols are not value types in the current checker.",
					"docs/spec/current_supported_surface.md keeps runtime protocol values outside the current v0.4.0 support claim.",
				},
				[]string{
					"go test ./compiler/tests/semantics -run 'Plan250ProtocolConformanceAndDynamicDispatchBoundaries'",
					"go test ./compiler -run 'P22ProtocolTrait|ValidateP22ProtocolTrait'",
				},
				[]string{
					"runtime protocol values remain unsupported",
					"runtime existential promotion requires future ABI, lifetime, ownership, diagnostics, docs, and report evidence",
					"not promoted in P22.2",
				},
				[]string{protocolTraitRuntimeBoundaryWitnessID}),
			p22ProtocolTraitRow(ProtocolTraitExplicitDynamicDispatchGate, "Explicit dynamic dispatch gate", "not_promoted", protocolTraitObjectDecisionKeepStaticOnly,
				[]string{
					"Dynamic dispatch must be explicit and report-visible before promotion.",
					"Runtime boundary witness records generic-bound requirement call rejection rather than lowering through witness-table dispatch.",
					"FeatureRegistry language.protocol-bound-generics-static says calling protocol requirements through generic bounds, witness tables, trait objects, runtime protocol values, and dynamic dispatch remain unsupported.",
				},
				[]string{
					"go test ./compiler/tests/semantics -run 'GenericFunctionProtocolBoundRequirementCallUnsupported'",
					"go test ./compiler -run 'P22ProtocolTrait|ValidateP22ProtocolTrait'",
				},
				[]string{
					"dynamic dispatch must be explicit and report-visible",
					"dynamic dispatch is not promoted",
					"generic-bound requirement calls remain diagnostics",
				},
				[]string{protocolTraitRuntimeBoundaryWitnessID, protocolTraitSpecializationWitnessID}),
			p22ProtocolTraitRow(ProtocolTraitSpecializationStaticAbstraction, "Specialization removes static abstraction", "bounded_existing_evidence", protocolTraitObjectDecisionKeepStaticOnly,
				[]string{
					"P17.2 InliningSpecializationCoverage records static protocol/conformance calls only after lowering to a known direct Stack IR function symbol.",
					"P21.2 SpecializationMachineCodeCoverage records protocol/static conformance rows and Machine IR contains no OpCall for the bounded known-direct witness.",
					"Specialization witness p17_p21_static_specialization_boundaries records P17.2 and P21.2 schemas, known direct symbol evidence, no dynamic dispatch claim, and Machine IR contains no OpCall.",
				},
				[]string{
					"go test ./compiler/internal/opt -run 'InliningSpecialization|SpecializationMachineCode'",
					"go test ./compiler/tests/semantics -run 'InliningSpecialization|ProtocolConformance|GenericFunctionProtocolBound'",
				},
				[]string{
					"static abstraction removal is limited to known direct Stack IR function symbols",
					"Machine IR contains no OpCall only for the bounded direct-call witness",
					"no broad protocol specialization, witness-table removal, dynamic dispatch removal, or performance claim is made",
				},
				[]string{protocolTraitSpecializationWitnessID}),
			p22ProtocolTraitRow(ProtocolTraitWitnessTableBoundary, "Witness-table boundary", "not_promoted", protocolTraitObjectDecisionKeepStaticOnly,
				[]string{
					"No current lowering path emits witness tables for protocol values.",
					"Current specialization rows mention witness tables only as non-claims and future boundaries.",
					"Future witness tables require future ABI evidence, lifetime/ownership rules, generated metadata, diagnostics, docs, and report-visible dynamic dispatch rows.",
				},
				[]string{
					"go test ./compiler/internal/opt -run 'InliningSpecialization|SpecializationMachineCode'",
					"go test ./compiler -run 'P22ProtocolTrait|ValidateP22ProtocolTrait'",
				},
				[]string{
					"witness tables are not emitted",
					"witness-table promotion is forbidden without future ABI evidence",
					"conformance-table lookup is not promoted",
				},
				[]string{protocolTraitSpecializationWitnessID}),
			p22ProtocolTraitRow(ProtocolTraitTraitObjectBoundary, "Trait-object boundary", "not_promoted", protocolTraitObjectDecisionKeepStaticOnly,
				[]string{
					"Trait objects are not promoted by P22.2.",
					"Runtime existential values are not value types in the current checker and remain a future design question.",
					"P22.0 feature surface audit routes protocol/trait-object runtime values to P22.2 and P22.2 keeps them out of the current branch without same-branch ABI/lifetime evidence.",
				},
				[]string{
					"go test ./compiler/tests/semantics -run 'Plan250ProtocolConformanceAndDynamicDispatchBoundaries'",
					"go test ./compiler -run 'P22ProtocolTrait|ValidateP22ProtocolTrait'",
				},
				[]string{
					"trait objects are not promoted",
					"runtime existential ABI is not designed in this slice",
					"trait-object promotion requires future ABI, lifetime, ownership, and report evidence",
				},
				[]string{protocolTraitRuntimeBoundaryWitnessID}),
			p22ProtocolTraitRow(ProtocolTraitRegistryDocsAlignment, "Registry and docs alignment", "current_static_only", protocolTraitObjectDecisionKeepStaticOnly,
				[]string{
					"FeatureRegistry records language.protocol-conformance-mvp as static conformance with no witness tables, trait objects, or dynamic dispatch model.",
					"FeatureRegistry records language.protocol-bound-generics-static as static monomorphization-time validation with runtime protocol values and dynamic dispatch unsupported.",
					"docs/spec/current_supported_surface.md, docs/design/explainable_one_build.md, and docs/design/truthful_intent_architecture.md preserve the same static-only decision boundary.",
				},
				[]string{
					"go test ./compiler/tests/semantics -run 'FeatureRegistry'",
					"go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json",
					"go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json",
				},
				[]string{
					"FeatureRegistry and docs must agree on the static-only decision",
					"language.protocol-conformance-mvp remains current static conformance",
					"language.protocol-bound-generics-static remains current static protocol-bound generics",
				},
				[]string{protocolTraitStaticConformanceWitnessID, protocolTraitProtocolBoundGenericWitnessID, protocolTraitRuntimeBoundaryWitnessID, protocolTraitSpecializationWitnessID}),
		},
		NonClaims: p22ProtocolTraitNonClaims(),
	}
	return report, nil
}

func ValidateP22ProtocolTraitObjectDecision(report ProtocolTraitObjectDecisionReport) error {
	if report.SchemaVersion != protocolTraitObjectDecisionSchemaV1 {
		return fmt.Errorf("protocol trait-object decision: schema = %q, want %q", report.SchemaVersion, protocolTraitObjectDecisionSchemaV1)
	}
	if report.Scope != protocolTraitObjectDecisionScopeP222 {
		return fmt.Errorf("protocol trait-object decision: scope = %q, want %q", report.Scope, protocolTraitObjectDecisionScopeP222)
	}
	if report.Decision != protocolTraitObjectDecisionKeepStaticOnly {
		return fmt.Errorf("protocol trait-object decision: decision = %q, want %q", report.Decision, protocolTraitObjectDecisionKeepStaticOnly)
	}
	if report.RuntimeExistentialsPromoted {
		return fmt.Errorf("protocol trait-object decision: runtime existential promotion is forbidden")
	}
	if report.TraitObjectsPromoted {
		return fmt.Errorf("protocol trait-object decision: trait object promotion is forbidden")
	}
	if report.WitnessTablesPromoted {
		return fmt.Errorf("protocol trait-object decision: witness table promotion is forbidden")
	}
	if report.DynamicDispatchPromoted {
		return fmt.Errorf("protocol trait-object decision: dynamic dispatch promotion is forbidden")
	}
	if report.ConformanceTableLookupPromoted {
		return fmt.Errorf("protocol trait-object decision: conformance-table lookup promotion is forbidden")
	}
	if report.RuntimeProtocolValuesPromoted {
		return fmt.Errorf("protocol trait-object decision: runtime protocol value promotion is forbidden")
	}
	if report.BroadSpecializationClaimed {
		return fmt.Errorf("protocol trait-object decision: broad specialization claim is forbidden")
	}
	if report.PerformanceClaimed {
		return fmt.Errorf("protocol trait-object decision: performance claim is forbidden")
	}
	if report.RuntimeBehaviorChanged {
		return fmt.Errorf("protocol trait-object decision: runtime behavior change claim is forbidden")
	}
	if report.SafeSemanticsChanged {
		return fmt.Errorf("protocol trait-object decision: safe-program semantics change is forbidden")
	}
	for _, nonClaim := range p22ProtocolTraitNonClaims() {
		if !p22ProtocolTraitReportHasString(report.NonClaims, nonClaim) {
			return fmt.Errorf("protocol trait-object decision: missing non-claim %q", nonClaim)
		}
	}
	if err := validateP22ProtocolTraitStrings("non-claim", report.NonClaims); err != nil {
		return err
	}

	witnesses := map[string]ProtocolTraitObjectWitness{}
	for _, witness := range report.Witnesses {
		if strings.TrimSpace(witness.ID) == "" || strings.TrimSpace(witness.Kind) == "" {
			return fmt.Errorf("protocol trait-object decision: witness missing required metadata: %#v", witness)
		}
		if _, ok := witnesses[witness.ID]; ok {
			return fmt.Errorf("protocol trait-object decision: duplicate witness %s", witness.ID)
		}
		witnesses[witness.ID] = witness
	}
	for _, id := range []string{
		protocolTraitStaticConformanceWitnessID,
		protocolTraitProtocolBoundGenericWitnessID,
		protocolTraitRuntimeBoundaryWitnessID,
		protocolTraitSpecializationWitnessID,
	} {
		if _, ok := witnesses[id]; !ok {
			return fmt.Errorf("protocol trait-object decision: missing witness %s", id)
		}
	}
	if err := validateP22ProtocolStaticWitness(witnesses[protocolTraitStaticConformanceWitnessID]); err != nil {
		return err
	}
	if err := validateP22ProtocolGenericWitness(witnesses[protocolTraitProtocolBoundGenericWitnessID]); err != nil {
		return err
	}
	if err := validateP22ProtocolRuntimeBoundaryWitness(witnesses[protocolTraitRuntimeBoundaryWitnessID]); err != nil {
		return err
	}
	if err := validateP22ProtocolSpecializationWitness(witnesses[protocolTraitSpecializationWitnessID]); err != nil {
		return err
	}

	expected := map[ProtocolTraitObjectDecisionID]bool{}
	for _, id := range p22ProtocolTraitObjectDecisionIDs() {
		expected[id] = true
	}
	seen := map[ProtocolTraitObjectDecisionID]bool{}
	for _, row := range report.Rows {
		if row.ID == "" || strings.TrimSpace(row.Name) == "" || strings.TrimSpace(row.Status) == "" || strings.TrimSpace(row.Decision) == "" {
			return fmt.Errorf("protocol trait-object decision: row missing required metadata: %#v", row)
		}
		if !expected[row.ID] {
			return fmt.Errorf("protocol trait-object decision: unexpected row %s", row.ID)
		}
		if seen[row.ID] {
			return fmt.Errorf("protocol trait-object decision: duplicate row %s", row.ID)
		}
		seen[row.ID] = true
		if row.Decision != protocolTraitObjectDecisionKeepStaticOnly {
			return fmt.Errorf("protocol trait-object decision: row %s decision = %q, want %q", row.ID, row.Decision, protocolTraitObjectDecisionKeepStaticOnly)
		}
		if err := validateP22ProtocolTraitStrings("row "+string(row.ID)+" evidence", row.Evidence); err != nil {
			return err
		}
		if err := validateP22ProtocolTraitStrings("row "+string(row.ID)+" tests", row.Tests); err != nil {
			return err
		}
		if err := validateP22ProtocolTraitStrings("row "+string(row.ID)+" boundaries", row.Boundaries); err != nil {
			return err
		}
		if len(row.WitnessIDs) == 0 {
			return fmt.Errorf("protocol trait-object decision: row %s missing witness reference", row.ID)
		}
		for _, id := range row.WitnessIDs {
			if _, ok := witnesses[id]; !ok {
				return fmt.Errorf("protocol trait-object decision: row %s references missing witness %s", row.ID, id)
			}
		}
	}
	for _, id := range p22ProtocolTraitObjectDecisionIDs() {
		if !seen[id] {
			return fmt.Errorf("protocol trait-object decision: missing row %s", id)
		}
	}
	return nil
}

func p22ProtocolTraitObjectDecisionIDs() []ProtocolTraitObjectDecisionID {
	return []ProtocolTraitObjectDecisionID{
		ProtocolTraitStaticConformanceFastPath,
		ProtocolTraitStaticProtocolBoundGenerics,
		ProtocolTraitRuntimeExistentialDecision,
		ProtocolTraitExplicitDynamicDispatchGate,
		ProtocolTraitSpecializationStaticAbstraction,
		ProtocolTraitWitnessTableBoundary,
		ProtocolTraitTraitObjectBoundary,
		ProtocolTraitRegistryDocsAlignment,
	}
}

func p22ProtocolTraitNonClaims() []string {
	return []string{
		"runtime protocol values are not promoted",
		"trait objects are not promoted",
		"witness tables are not promoted",
		"dynamic dispatch is not promoted",
		"conformance-table lookup is not promoted",
		"runtime existential ABI is not designed in this slice",
		"broad protocol specialization is not claimed",
		"performance is not claimed",
		"runtime behavior does not change",
		"safe-program semantics do not change",
	}
}

func p22ProtocolTraitRow(id ProtocolTraitObjectDecisionID, name, status, decision string, evidence, tests, boundaries, witnessIDs []string) ProtocolTraitObjectDecisionRow {
	return ProtocolTraitObjectDecisionRow{
		ID:         id,
		Name:       name,
		Status:     status,
		Decision:   decision,
		Evidence:   append([]string{}, evidence...),
		Tests:      append([]string{}, tests...),
		Boundaries: append([]string{}, boundaries...),
		WitnessIDs: append([]string{}, witnessIDs...),
	}
}

func buildP22ProtocolStaticConformanceWitness() (ProtocolTraitObjectWitness, error) {
	src := []byte(`
struct Vec2:
    x: Int

protocol Renderable:
    func draw(self: Vec2) -> Int

extension Vec2:
    func draw(self: Vec2) -> Int:
        return self.x

impl Vec2: Renderable

func main() -> Int:
    return Vec2.draw(Vec2(x: 42))
`)
	prog, err := Parse(src)
	if err != nil {
		return ProtocolTraitObjectWitness{}, fmt.Errorf("protocol trait-object decision: parse static witness: %w", err)
	}
	checked, err := Check(prog)
	if err != nil {
		return ProtocolTraitObjectWitness{}, fmt.Errorf("protocol trait-object decision: check static witness: %w", err)
	}
	lowered, err := Lower(checked)
	if err != nil {
		return ProtocolTraitObjectWitness{}, fmt.Errorf("protocol trait-object decision: lower static witness: %w", err)
	}
	main, ok := p222FindIRFunc(lowered, "main")
	if !ok {
		return ProtocolTraitObjectWitness{}, fmt.Errorf("protocol trait-object decision: static witness missing lowered main")
	}
	directCallTarget := ""
	if p222HasIRCall(main, "Vec2.draw") {
		directCallTarget = "Vec2.draw"
	}
	_, hasSig := checked.FuncSigs["Vec2.draw"]
	return ProtocolTraitObjectWitness{
		ID:                 protocolTraitStaticConformanceWitnessID,
		Kind:               "static_conformance_fast_path",
		ProtocolCount:      len(prog.Protocols),
		ImplCount:          len(prog.Impls),
		HasStaticMethodSig: hasSig,
		DirectCallTarget:   directCallTarget,
		LoweredDirectCall:  directCallTarget != "",
	}, nil
}

func buildP22ProtocolBoundGenericWitness() (ProtocolTraitObjectWitness, error) {
	src := []byte(`
struct Vec2:
    x: Int

protocol Echoable:
    func echo(self: Vec2) -> Vec2

extension Vec2:
    func echo(self: Vec2) -> Vec2:
        return self

impl Vec2: Echoable

func id<T: Echoable>(x: T) -> T:
    return x

func main() -> Int:
    let v: Vec2 = Vec2(x: 42)
    let out: Vec2 = id(v)
    return out.x
`)
	prog, err := Parse(src)
	if err != nil {
		return ProtocolTraitObjectWitness{}, fmt.Errorf("protocol trait-object decision: parse generic witness: %w", err)
	}
	checked, err := Check(prog)
	if err != nil {
		return ProtocolTraitObjectWitness{}, fmt.Errorf("protocol trait-object decision: check generic witness: %w", err)
	}
	lowered, err := Lower(checked)
	if err != nil {
		return ProtocolTraitObjectWitness{}, fmt.Errorf("protocol trait-object decision: lower generic witness: %w", err)
	}
	sig, ok := checked.FuncSigs["id__T_Vec2"]
	main, hasMain := p222FindIRFunc(lowered, "main")
	loweredDirectCall := hasMain && p222HasIRCall(main, "id__T_Vec2")
	return ProtocolTraitObjectWitness{
		ID:                       protocolTraitProtocolBoundGenericWitnessID,
		Kind:                     "static_protocol_bound_generics",
		ProtocolCount:            len(prog.Protocols),
		ImplCount:                len(prog.Impls),
		MonomorphizedSig:         "id__T_Vec2",
		MonomorphizedSigConcrete: ok && !sig.Generic,
		LoweredDirectCall:        loweredDirectCall,
		DirectCallTarget:         "id__T_Vec2",
	}, nil
}

func buildP22ProtocolRuntimeBoundaryWitness() (ProtocolTraitObjectWitness, error) {
	runtimeValueErr := p222CheckDiagnostic(`
struct Vec2:
    x: Int

protocol Drawable:
    func draw(self: Vec2) -> Int

func main() -> Int:
    let value: Drawable = Vec2(x: 1)
    return 0
`)
	if runtimeValueErr == "" {
		return ProtocolTraitObjectWitness{}, fmt.Errorf("protocol trait-object decision: runtime protocol value witness unexpectedly checked")
	}
	requirementCallErr := p222CheckDiagnostic(`
struct Vec2:
    x: Int

protocol Echoable:
    func echo(self: Vec2) -> Vec2

extension Vec2:
    func echo(self: Vec2) -> Vec2:
        return self

impl Vec2: Echoable

func echoThroughBound<T: Echoable>(x: T) -> T:
    return T.echo(x)

func main() -> Int:
    let v: Vec2 = Vec2(x: 42)
    let out: Vec2 = echoThroughBound(v)
    return out.x
`)
	if requirementCallErr == "" {
		return ProtocolTraitObjectWitness{}, fmt.Errorf("protocol trait-object decision: generic requirement call witness unexpectedly checked")
	}
	return ProtocolTraitObjectWitness{
		ID:                               protocolTraitRuntimeBoundaryWitnessID,
		Kind:                             "runtime_existential_and_dynamic_dispatch_boundary",
		RuntimeProtocolValueDiagnostic:   runtimeValueErr,
		GenericRequirementCallDiagnostic: requirementCallErr,
	}, nil
}

func buildP22ProtocolSpecializationWitness() (ProtocolTraitObjectWitness, error) {
	inlining := opt.InliningSpecializationCoverage()
	staticInline, ok := p222FindInliningRow(inlining, opt.InliningSpecializationStaticProtocolConformanceCalls)
	if !ok {
		return ProtocolTraitObjectWitness{}, fmt.Errorf("protocol trait-object decision: missing P17.2 static protocol/conformance row")
	}
	machine, err := opt.SpecializationMachineCodeCoverage()
	if err != nil {
		return ProtocolTraitObjectWitness{}, fmt.Errorf("protocol trait-object decision: P21.2 specialization witness: %w", err)
	}
	if err := opt.ValidateSpecializationMachineCodeCoverage(machine); err != nil {
		return ProtocolTraitObjectWitness{}, fmt.Errorf("protocol trait-object decision: validate P21.2 specialization witness: %w", err)
	}
	staticMachine, ok := p222FindMachineRow(machine, opt.SpecializationMachineCodeProtocolStaticConformance)
	if !ok {
		return ProtocolTraitObjectWitness{}, fmt.Errorf("protocol trait-object decision: missing P21.2 protocol/static conformance row")
	}
	machineNoOpCall := false
	if len(machine.Witnesses) > 0 {
		machineNoOpCall = !machine.Witnesses[0].MachineIRHasCall && machine.Witnesses[0].MachineIRVerified
	}
	combined := staticInline.Boundary + " " + staticInline.Evidence + " " + staticMachine.SourceEvidence + " " + staticMachine.OptimizedIREvidence + " " + staticMachine.MachineCodeEvidence + " " + staticMachine.Boundary
	return ProtocolTraitObjectWitness{
		ID:                              protocolTraitSpecializationWitnessID,
		Kind:                            "specialization_static_abstraction",
		InliningSchema:                  inlining.SchemaVersion,
		MachineSchema:                   machine.SchemaVersion,
		KnownDirectSymbolEvidence:       strings.Contains(combined, "known direct Stack IR function symbol"),
		SpecializationNoDynamicDispatch: strings.Contains(combined, "no witness tables") && strings.Contains(combined, "dynamic dispatch"),
		MachineNoOpCall:                 machineNoOpCall && strings.Contains(combined, "Machine IR contains no OpCall"),
	}, nil
}

func p222CheckDiagnostic(src string) string {
	prog, err := Parse([]byte(src))
	if err != nil {
		return err.Error()
	}
	_, err = Check(prog)
	if err == nil {
		return ""
	}
	return err.Error()
}

func p222FindIRFunc(prog *IRProgram, name string) (ir.IRFunc, bool) {
	for _, fn := range prog.Funcs {
		if fn.Name == name {
			return fn, true
		}
	}
	return ir.IRFunc{}, false
}

func p222HasIRCall(fn ir.IRFunc, name string) bool {
	for _, instr := range fn.Instrs {
		if instr.Kind == ir.IRCall && instr.Name == name {
			return true
		}
	}
	return false
}

func p222FindInliningRow(report opt.InliningSpecializationCoverageReport, id opt.InliningSpecializationID) (opt.InliningSpecializationCoverageRow, bool) {
	for _, row := range report.Rows {
		if row.ID == id {
			return row, true
		}
	}
	return opt.InliningSpecializationCoverageRow{}, false
}

func p222FindMachineRow(report opt.SpecializationMachineCodeCoverageReport, id opt.SpecializationMachineCodeID) (opt.SpecializationMachineCodeRow, bool) {
	for _, row := range report.Rows {
		if row.ID == id {
			return row, true
		}
	}
	return opt.SpecializationMachineCodeRow{}, false
}

func validateP22ProtocolStaticWitness(witness ProtocolTraitObjectWitness) error {
	if witness.ProtocolCount != 1 || witness.ImplCount != 1 || !witness.HasStaticMethodSig || witness.DirectCallTarget != "Vec2.draw" || !witness.LoweredDirectCall {
		return fmt.Errorf("protocol trait-object decision: static conformance witness drift: %#v", witness)
	}
	return nil
}

func validateP22ProtocolGenericWitness(witness ProtocolTraitObjectWitness) error {
	if witness.MonomorphizedSig != "id__T_Vec2" || !witness.MonomorphizedSigConcrete || witness.DirectCallTarget != "id__T_Vec2" || !witness.LoweredDirectCall {
		return fmt.Errorf("protocol trait-object decision: protocol-bound generic witness drift: %#v", witness)
	}
	return nil
}

func validateP22ProtocolRuntimeBoundaryWitness(witness ProtocolTraitObjectWitness) error {
	if !strings.Contains(witness.RuntimeProtocolValueDiagnostic, "unknown type 'Drawable'") || !strings.Contains(witness.GenericRequirementCallDiagnostic, "not supported in this MVP") {
		return fmt.Errorf("protocol trait-object decision: runtime boundary witness drift: %#v", witness)
	}
	return nil
}

func validateP22ProtocolSpecializationWitness(witness ProtocolTraitObjectWitness) error {
	if witness.InliningSchema != "tetra.optimizer.inlining_specialization.v1" || witness.MachineSchema != "tetra.optimizer.specialization_machine_code.v1" || !witness.KnownDirectSymbolEvidence || !witness.SpecializationNoDynamicDispatch || !witness.MachineNoOpCall {
		return fmt.Errorf("protocol trait-object decision: specialization witness drift: %#v", witness)
	}
	return nil
}

func validateP22ProtocolTraitStrings(label string, items []string) error {
	if len(items) == 0 {
		return fmt.Errorf("protocol trait-object decision: %s missing", label)
	}
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			return fmt.Errorf("protocol trait-object decision: %s contains empty item", label)
		}
		if p22ProtocolTraitContainsPlaceholder(trimmed) {
			return fmt.Errorf("protocol trait-object decision: %s contains placeholder evidence: %q", label, item)
		}
	}
	return nil
}

func p22ProtocolTraitContainsPlaceholder(text string) bool {
	lower := strings.ToLower(text)
	for _, token := range []string{"todo", "tbd", "placeholder", "fixme", "???"} {
		if strings.Contains(lower, token) {
			return true
		}
	}
	return false
}

func p22ProtocolTraitReportHasString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
