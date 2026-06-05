package opt

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/lower"
	"tetra_language/compiler/internal/validation"
)

type IRKind string

const (
	IRKindStack     IRKind = "stack_ir"
	IRKindOptimized IRKind = "optimized_ir"
	IRKindMachine   IRKind = "machine_ir"
)

type Fact string

const (
	FactIRVerified   Fact = "ir_verified"
	FactBoundsProofs Fact = "bounds_proofs"
	FactLiveness     Fact = "liveness"
)

type ValidationStrategy string

const (
	ValidationTranslation ValidationStrategy = "translation_validation"
)

type ProfileInputPolicy string

const (
	ProfileInputUnused        ProfileInputPolicy = "unused"
	ProfileInputGuidedRewrite ProfileInputPolicy = "guided_rewrite"
)

const (
	VerifierLowerVerifyProgram         = "lower.VerifyProgram"
	VerifierMachineVerifyFunction      = "machine.VerifyFunction"
	TranslationHookValidateTranslation = "validation.ValidateTranslation"
	NegativeTestPassContractV1         = "compiler/internal/opt/manager_test.go::TestManagerRejectsIncompletePassContractEvidence"
)

type ProofRule string

const (
	ProofRulePreserveBoundsProofs              ProofRule = "preserve_bounds_proofs"
	ProofRulePreserveBoundsInvalidateLiveness  ProofRule = "preserve_bounds_proofs_invalidate_liveness"
	ProofRuleInvalidateBoundsProofs            ProofRule = "invalidate_bounds_proofs"
	ProofRuleInvalidateBoundsProofsAndLiveness ProofRule = "invalidate_bounds_proofs_and_liveness"
)

type Pass struct {
	Name                      string
	InputKind                 IRKind
	OutputKind                IRKind
	InputVerifier             string
	OutputVerifier            string
	RequiredFacts             []Fact
	PreservedFacts            []Fact
	InvalidatedFacts          []Fact
	ProofRule                 ProofRule
	ValidationStrategy        ValidationStrategy
	TranslationValidationHook string
	ReportOutput              string
	ReportRows                []string
	NegativeTestMarker        string
	ProfileInputPolicy        ProfileInputPolicy
	Run                       func(*ir.IRProgram) error
	Decisions                 func() []PassDecision
}

type Manager struct{}

type Options struct {
	OnlyPass     string
	ProfileInput *ProfileCollection
}

type Report struct {
	Passes []PassReport `json:"passes"`
}

type PassReport struct {
	Name                      string                                     `json:"name"`
	InputKind                 IRKind                                     `json:"input_ir_kind"`
	OutputKind                IRKind                                     `json:"output_ir_kind"`
	InputVerifier             string                                     `json:"input_verifier"`
	OutputVerifier            string                                     `json:"output_verifier"`
	RequiredFacts             []Fact                                     `json:"required_facts,omitempty"`
	PreservedFacts            []Fact                                     `json:"preserved_facts,omitempty"`
	InvalidatedFacts          []Fact                                     `json:"invalidated_facts,omitempty"`
	ProofRule                 ProofRule                                  `json:"proof_rule"`
	ValidationStrategy        ValidationStrategy                         `json:"validation_strategy"`
	TranslationValidationHook string                                     `json:"translation_validation_hook"`
	ReportOutput              string                                     `json:"report_output"`
	ReportRows                []string                                   `json:"report_rows"`
	NegativeTestMarker        string                                     `json:"negative_test_marker"`
	ProfileInputPolicy        ProfileInputPolicy                         `json:"profile_input_policy"`
	ProfileInput              *OptimizerProfileInputEvidence             `json:"profile_input,omitempty"`
	BeforeDump                string                                     `json:"before_dump"`
	AfterDump                 string                                     `json:"after_dump"`
	VerifiedInput             bool                                       `json:"verified_input"`
	VerifiedOutput            bool                                       `json:"verified_output"`
	VerifiedProofs            bool                                       `json:"verified_proofs"`
	TranslationValidated      bool                                       `json:"translation_validated,omitempty"`
	TranslationReport         *validation.TranslationReport              `json:"translation_report,omitempty"`
	ValidationMetadata        *validation.OptimizationValidationMetadata `json:"validation_metadata,omitempty"`
	Decisions                 []PassDecision                             `json:"decisions,omitempty"`
}

type PassDecision struct {
	Action string `json:"action"`
	Caller string `json:"caller,omitempty"`
	Callee string `json:"callee,omitempty"`
	Site   int    `json:"site"`
	Reason string `json:"reason"`
}

func NewManager() Manager {
	return Manager{}
}

func RegisteredPasses() []Pass {
	return []Pass{
		BasicScalarPass(),
		SCCPPass(),
		Mem2RegPass(),
		InlineSmallPurePass(),
		LoopCanonicalizationPass(),
		LICMPureInvariantPass(),
	}
}

func (m Manager) Run(prog *ir.IRProgram, passes ...Pass) (Report, error) {
	return m.RunWithOptions(prog, Options{}, passes...)
}

func (m Manager) RunWithOptions(prog *ir.IRProgram, opt Options, passes ...Pass) (Report, error) {
	selected, err := selectPassesForRun(opt, passes)
	if err != nil {
		return Report{}, err
	}
	var profileEvidence *OptimizerProfileInputEvidence
	if opt.ProfileInput != nil {
		evidence, err := BuildOptimizerProfileInputEvidence(*opt.ProfileInput)
		if err != nil {
			return Report{}, fmt.Errorf("optimizer profile input: %w", err)
		}
		profileEvidence = &evidence
	}
	return m.runSelected(prog, profileEvidence, selected...)
}

func (m Manager) runSelected(prog *ir.IRProgram, profileEvidence *OptimizerProfileInputEvidence, passes ...Pass) (Report, error) {
	report := Report{Passes: make([]PassReport, 0, len(passes))}
	for _, pass := range passes {
		if err := validatePassMetadata(pass); err != nil {
			return report, err
		}
		row := newPassReport(pass, profileEvidence)
		if err := lower.VerifyProgram(prog); err != nil {
			return report, fmt.Errorf("%s input verification failed: %w", pass.Name, err)
		}
		row.VerifiedInput = true
		row.BeforeDump = FormatProgram(prog)
		before := cloneProgram(prog)
		if err := pass.Run(prog); err != nil {
			report.Passes = append(report.Passes, row)
			return report, fmt.Errorf("%s failed: %w", pass.Name, err)
		}
		if pass.Decisions != nil {
			row.Decisions = append([]PassDecision(nil), pass.Decisions()...)
		}
		row.AfterDump = FormatProgram(prog)
		if err := lower.VerifyProgram(prog); err != nil {
			report.Passes = append(report.Passes, row)
			return report, fmt.Errorf("%s output verification failed: %w", pass.Name, err)
		}
		row.VerifiedOutput = true
		if pass.ValidationStrategy == ValidationTranslation {
			translationReport, err := validation.ValidateTranslation(before, prog)
			if err != nil {
				report.Passes = append(report.Passes, row)
				return report, fmt.Errorf("%s translation validation failed: %w", pass.Name, err)
			}
			metadata, err := validation.BuildOptimizationValidationMetadata(before, prog, validation.OptimizationMetadataOptions{
				PassName:                  pass.Name,
				InputKind:                 string(pass.InputKind),
				OutputKind:                string(pass.OutputKind),
				InputVerifier:             pass.InputVerifier,
				OutputVerifier:            pass.OutputVerifier,
				ValidationStrategy:        string(pass.ValidationStrategy),
				RequiredFacts:             factStrings(pass.RequiredFacts),
				PreservedFacts:            factStrings(pass.PreservedFacts),
				InvalidatedFacts:          factStrings(pass.InvalidatedFacts),
				ProofRule:                 string(pass.ProofRule),
				TranslationValidationHook: pass.TranslationValidationHook,
				ReportRows:                append([]string(nil), pass.ReportRows...),
				NegativeTestMarker:        pass.NegativeTestMarker,
				ProfileInputPolicy:        string(pass.ProfileInputPolicy),
			})
			if err != nil {
				report.Passes = append(report.Passes, row)
				return report, fmt.Errorf("%s validation metadata failed: %w", pass.Name, err)
			}
			if profileEvidence != nil {
				metadata.ProfileInputDigest = profileEvidence.Digest
				metadata.ProfileInputSchemaVersion = profileEvidence.SchemaVersion
				if err := validation.ValidateOptimizationValidationMetadata(metadata); err != nil {
					report.Passes = append(report.Passes, row)
					return report, fmt.Errorf("%s validation metadata failed: %w", pass.Name, err)
				}
			}
			row.TranslationValidated = true
			row.TranslationReport = &translationReport
			row.ValidationMetadata = &metadata
		}
		if _, err := validation.CheckBoundsProofs(prog); err != nil {
			report.Passes = append(report.Passes, row)
			return report, fmt.Errorf("%s proof verification failed: %w", pass.Name, err)
		}
		row.VerifiedProofs = true
		report.Passes = append(report.Passes, row)
	}
	return report, nil
}

func factStrings(facts []Fact) []string {
	out := make([]string, len(facts))
	for i, fact := range facts {
		out[i] = string(fact)
	}
	return out
}

func selectPassesForRun(opt Options, passes []Pass) ([]Pass, error) {
	if opt.OnlyPass == "" {
		return passes, nil
	}
	var selected []Pass
	for _, pass := range passes {
		if pass.Name == opt.OnlyPass {
			selected = append(selected, pass)
		}
	}
	if len(selected) == 0 {
		return nil, fmt.Errorf("optimizer pass %q not found", opt.OnlyPass)
	}
	if len(selected) > 1 {
		return nil, fmt.Errorf("optimizer pass %q is ambiguous", opt.OnlyPass)
	}
	return selected, nil
}

func validatePassMetadata(pass Pass) error {
	return ValidatePassContract(pass)
}

func ValidatePassContract(pass Pass) error {
	if pass.Name == "" {
		return fmt.Errorf("optimizer pass is missing name")
	}
	if pass.InputKind == "" {
		return fmt.Errorf("optimizer pass %q missing input IR kind", pass.Name)
	}
	if pass.OutputKind == "" {
		return fmt.Errorf("optimizer pass %q missing output IR kind", pass.Name)
	}
	if pass.InputVerifier == "" {
		return fmt.Errorf("optimizer pass %q missing input verifier", pass.Name)
	}
	if !supportsVerifier(pass.InputKind, pass.InputVerifier) {
		return fmt.Errorf("optimizer pass %q unsupported input verifier %q for %s", pass.Name, pass.InputVerifier, pass.InputKind)
	}
	if pass.OutputVerifier == "" {
		return fmt.Errorf("optimizer pass %q missing output verifier", pass.Name)
	}
	if !supportsVerifier(pass.OutputKind, pass.OutputVerifier) {
		return fmt.Errorf("optimizer pass %q unsupported output verifier %q for %s", pass.Name, pass.OutputVerifier, pass.OutputKind)
	}
	if pass.ProofRule == "" {
		return fmt.Errorf("optimizer pass %q missing proof preservation or invalidation rule", pass.Name)
	}
	if err := validateProofRule(pass); err != nil {
		return err
	}
	if pass.ValidationStrategy == "" {
		return fmt.Errorf("optimizer pass %q missing validation strategy", pass.Name)
	}
	switch pass.ValidationStrategy {
	case ValidationTranslation:
	default:
		return fmt.Errorf("optimizer pass %q must use translation validation strategy, got %q", pass.Name, pass.ValidationStrategy)
	}
	if pass.TranslationValidationHook == "" {
		return fmt.Errorf("optimizer pass %q missing translation validation hook", pass.Name)
	}
	if pass.TranslationValidationHook != TranslationHookValidateTranslation {
		return fmt.Errorf("optimizer pass %q unsupported translation validation hook %q", pass.Name, pass.TranslationValidationHook)
	}
	if pass.ReportOutput == "" {
		return fmt.Errorf("optimizer pass %q missing report output", pass.Name)
	}
	if len(pass.ReportRows) == 0 {
		return fmt.Errorf("optimizer pass %q missing report rows", pass.Name)
	}
	for _, row := range RequiredP17ReportRows() {
		if !hasReportRow(pass.ReportRows, row) {
			return fmt.Errorf("optimizer pass %q missing required report row %q", pass.Name, row)
		}
	}
	if pass.NegativeTestMarker == "" {
		return fmt.Errorf("optimizer pass %q missing negative-test marker", pass.Name)
	}
	if pass.NegativeTestMarker != NegativeTestPassContractV1 {
		return fmt.Errorf("optimizer pass %q unknown negative-test marker %q", pass.Name, pass.NegativeTestMarker)
	}
	if pass.ProfileInputPolicy == "" {
		return fmt.Errorf("optimizer pass %q missing profile input policy", pass.Name)
	}
	switch pass.ProfileInputPolicy {
	case ProfileInputUnused:
	case ProfileInputGuidedRewrite:
		return fmt.Errorf("optimizer pass %q profile-guided optimizer decisions require dedicated validation", pass.Name)
	default:
		return fmt.Errorf("optimizer pass %q unsupported profile input policy %q", pass.Name, pass.ProfileInputPolicy)
	}
	if pass.Run == nil {
		return fmt.Errorf("optimizer pass %q is missing run function", pass.Name)
	}
	return nil
}

func RequiredP17ReportRows() []string {
	return []string{
		"input_verifier",
		"output_verifier",
		"proof_rule",
		"translation_validation_hook",
		"translation_report",
		"validation_metadata",
		"before_dump",
		"after_dump",
		"profile_input_policy",
	}
}

func supportsVerifier(kind IRKind, verifier string) bool {
	switch kind {
	case IRKindStack, IRKindOptimized:
		return verifier == VerifierLowerVerifyProgram
	case IRKindMachine:
		return verifier == VerifierMachineVerifyFunction
	default:
		return false
	}
}

func validateProofRule(pass Pass) error {
	switch pass.ProofRule {
	case ProofRulePreserveBoundsProofs:
		if !hasFact(pass.PreservedFacts, FactBoundsProofs) {
			return fmt.Errorf("optimizer pass %q proof rule %q requires preserved fact %q", pass.Name, pass.ProofRule, FactBoundsProofs)
		}
	case ProofRulePreserveBoundsInvalidateLiveness:
		if !hasFact(pass.PreservedFacts, FactBoundsProofs) {
			return fmt.Errorf("optimizer pass %q proof rule %q requires preserved fact %q", pass.Name, pass.ProofRule, FactBoundsProofs)
		}
		if !hasFact(pass.InvalidatedFacts, FactLiveness) {
			return fmt.Errorf("optimizer pass %q proof rule %q requires invalidated fact %q", pass.Name, pass.ProofRule, FactLiveness)
		}
	case ProofRuleInvalidateBoundsProofs:
		if !hasFact(pass.InvalidatedFacts, FactBoundsProofs) {
			return fmt.Errorf("optimizer pass %q proof rule %q requires invalidated fact %q", pass.Name, pass.ProofRule, FactBoundsProofs)
		}
	case ProofRuleInvalidateBoundsProofsAndLiveness:
		if !hasFact(pass.InvalidatedFacts, FactBoundsProofs) || !hasFact(pass.InvalidatedFacts, FactLiveness) {
			return fmt.Errorf("optimizer pass %q proof rule %q requires invalidated facts %q and %q", pass.Name, pass.ProofRule, FactBoundsProofs, FactLiveness)
		}
	default:
		return fmt.Errorf("optimizer pass %q unknown proof preservation or invalidation rule %q", pass.Name, pass.ProofRule)
	}
	return nil
}

func hasFact(facts []Fact, want Fact) bool {
	for _, fact := range facts {
		if fact == want {
			return true
		}
	}
	return false
}

func hasReportRow(rows []string, want string) bool {
	for _, row := range rows {
		if row == want {
			return true
		}
	}
	return false
}

func newPassReport(pass Pass, profileEvidence *OptimizerProfileInputEvidence) PassReport {
	var profileInput *OptimizerProfileInputEvidence
	if profileEvidence != nil {
		copyEvidence := *profileEvidence
		copyEvidence.CounterKinds = append([]string(nil), profileEvidence.CounterKinds...)
		profileInput = &copyEvidence
	}
	return PassReport{
		Name:                      pass.Name,
		InputKind:                 pass.InputKind,
		OutputKind:                pass.OutputKind,
		InputVerifier:             pass.InputVerifier,
		OutputVerifier:            pass.OutputVerifier,
		RequiredFacts:             append([]Fact(nil), pass.RequiredFacts...),
		PreservedFacts:            append([]Fact(nil), pass.PreservedFacts...),
		InvalidatedFacts:          append([]Fact(nil), pass.InvalidatedFacts...),
		ProofRule:                 pass.ProofRule,
		ValidationStrategy:        pass.ValidationStrategy,
		TranslationValidationHook: pass.TranslationValidationHook,
		ReportOutput:              pass.ReportOutput,
		ReportRows:                append([]string(nil), pass.ReportRows...),
		NegativeTestMarker:        pass.NegativeTestMarker,
		ProfileInputPolicy:        pass.ProfileInputPolicy,
		ProfileInput:              profileInput,
	}
}

func FormatProgram(prog *ir.IRProgram) string {
	if prog == nil {
		return "program stack_ir <nil>\n"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "program stack_ir main:%s index:%d\n", prog.MainName, prog.MainIndex)
	for i, fn := range prog.Funcs {
		if i > 0 {
			fmt.Fprintln(&b)
		}
		fmt.Fprintf(&b, "func %s params:%d locals:%d returns:%d\n", fn.Name, fn.ParamSlots, fn.LocalSlots, fn.ReturnSlots)
		for _, instr := range fn.Instrs {
			fmt.Fprintf(&b, "  %s", instrName(instr.Kind))
			switch instr.Kind {
			case ir.IRConstI32:
				fmt.Fprintf(&b, " %d", instr.Imm)
			case ir.IRLoadLocal, ir.IRStoreLocal, ir.IRLoadGlobal, ir.IRStoreGlobal:
				fmt.Fprintf(&b, " local:%d", instr.Local)
			case ir.IRCall:
				fmt.Fprintf(&b, " %s args:%d rets:%d", instr.Name, instr.ArgSlots, instr.RetSlots)
			case ir.IRLabel, ir.IRJmp, ir.IRJmpIfZero:
				fmt.Fprintf(&b, " label:%d", instr.Label)
			case ir.IRStrLit:
				fmt.Fprintf(&b, " bytes:%d", len(instr.Str))
			}
			if instr.ProofID != "" {
				fmt.Fprintf(&b, " proof:%s", instr.ProofID)
			}
			fmt.Fprintln(&b)
		}
	}
	return b.String()
}

func cloneProgram(prog *ir.IRProgram) *ir.IRProgram {
	if prog == nil {
		return nil
	}
	out := *prog
	out.Funcs = make([]ir.IRFunc, len(prog.Funcs))
	for i, fn := range prog.Funcs {
		out.Funcs[i] = fn
		out.Funcs[i].Instrs = append([]ir.IRInstr(nil), fn.Instrs...)
	}
	return &out
}

func instrName(kind ir.IRInstrKind) string {
	switch kind {
	case ir.IRWrite:
		return "write"
	case ir.IRStrLit:
		return "str_lit"
	case ir.IRConstI32:
		return "const_i32"
	case ir.IRLoadLocal:
		return "load_local"
	case ir.IRStoreLocal:
		return "store_local"
	case ir.IRLoadGlobal:
		return "load_global"
	case ir.IRStoreGlobal:
		return "store_global"
	case ir.IRAddI32:
		return "add_i32"
	case ir.IRSubI32:
		return "sub_i32"
	case ir.IRNegI32:
		return "neg_i32"
	case ir.IRCmpEqI32:
		return "cmp_eq_i32"
	case ir.IRCmpLtI32:
		return "cmp_lt_i32"
	case ir.IRMulI32:
		return "mul_i32"
	case ir.IRDivI32:
		return "div_i32"
	case ir.IRModI32:
		return "mod_i32"
	case ir.IRCmpGtI32:
		return "cmp_gt_i32"
	case ir.IRCmpGeI32:
		return "cmp_ge_i32"
	case ir.IRCmpLeI32:
		return "cmp_le_i32"
	case ir.IRCmpNeI32:
		return "cmp_ne_i32"
	case ir.IRCall:
		return "call"
	case ir.IRLabel:
		return "label"
	case ir.IRJmp:
		return "jmp"
	case ir.IRJmpIfZero:
		return "jmp_if_zero"
	case ir.IRReturn:
		return "return"
	default:
		return fmt.Sprintf("ir_%d", kind)
	}
}
