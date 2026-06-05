package opt

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/ir"
)

func TestManagerVerifiesBeforeAndAfterPass(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			LocalSlots:  0,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
			},
		}},
	}
	manager := NewManager()
	report, err := manager.Run(prog, p17ContractTestPass("noop"))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(report.Passes) != 1 || report.Passes[0].Name != "noop" || !report.Passes[0].VerifiedInput || !report.Passes[0].VerifiedOutput || !report.Passes[0].VerifiedProofs {
		t.Fatalf("report = %#v", report)
	}
	row := report.Passes[0]
	if row.InputKind != IRKindStack || row.OutputKind != IRKindStack || row.ValidationStrategy != ValidationTranslation || row.ReportOutput != "noop.opt.json" {
		t.Fatalf("metadata row = %#v", row)
	}
	for _, want := range []string{"func main", "const_i32 7", "return"} {
		if !strings.Contains(row.BeforeDump, want) || !strings.Contains(row.AfterDump, want) {
			t.Fatalf("before/after dumps missing %q:\nbefore:\n%s\nafter:\n%s", want, row.BeforeDump, row.AfterDump)
		}
	}
}

func TestManagerRejectsPassThatProducesInvalidIR(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			LocalSlots:  0,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRReturn},
			},
		}},
	}
	manager := NewManager()
	pass := p17ContractTestPass("break-return")
	pass.Run = func(p *ir.IRProgram) error {
		p.Funcs[0].Instrs = p.Funcs[0].Instrs[:1]
		return nil
	}
	_, err := manager.Run(prog, pass)
	if err == nil || !strings.Contains(err.Error(), "break-return output verification failed") {
		t.Fatalf("Run error = %v", err)
	}
}

func TestManagerCanRunOnePassByNameForTests(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 1},
				{Kind: ir.IRReturn},
			},
		}},
	}
	manager := NewManager()
	ran := []string{}
	first := p17ContractTestPass("first")
	first.Run = func(p *ir.IRProgram) error {
		ran = append(ran, "first")
		return nil
	}
	second := p17ContractTestPass("second")
	second.Run = func(p *ir.IRProgram) error {
		ran = append(ran, "second")
		return nil
	}
	report, err := manager.RunWithOptions(prog, Options{OnlyPass: "second"}, first, second)
	if err != nil {
		t.Fatalf("RunWithOptions: %v", err)
	}
	if strings.Join(ran, ",") != "second" {
		t.Fatalf("ran passes = %v, want only second", ran)
	}
	if len(report.Passes) != 1 || report.Passes[0].Name != "second" {
		t.Fatalf("report passes = %#v, want only second", report.Passes)
	}
}

func TestManagerRejectsMissingPassMetadata(t *testing.T) {
	prog := validTinyProgram()
	manager := NewManager()
	_, err := manager.Run(prog, Pass{
		Name: "nameless-metadata",
		Run:  func(p *ir.IRProgram) error { return nil },
	})
	if err == nil || !strings.Contains(err.Error(), "missing input IR kind") {
		t.Fatalf("Run error = %v, want metadata rejection", err)
	}
}

func TestManagerRunsTranslationValidationStrategy(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{
			{
				Name:        "main",
				ReturnSlots: 1,
				Instrs: []ir.IRInstr{
					{Kind: ir.IRConstI32, Imm: 1},
					{Kind: ir.IRReturn},
				},
			},
			{
				Name:        "helper",
				ReturnSlots: 1,
				Instrs: []ir.IRInstr{
					{Kind: ir.IRConstI32, Imm: 2},
					{Kind: ir.IRReturn},
				},
			},
		},
	}
	manager := NewManager()
	pass := p17ContractTestPass("bad-delete-helper")
	pass.Run = func(p *ir.IRProgram) error {
		p.Funcs = p.Funcs[:1]
		return nil
	}
	_, err := manager.Run(prog, pass)
	if err == nil || !strings.Contains(err.Error(), "translation validation failed") {
		t.Fatalf("Run error = %v, want translation validation failure", err)
	}
}

func TestManagerRejectsSemanticChangingTranslationPass(t *testing.T) {
	prog := &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 1},
				{Kind: ir.IRReturn},
			},
		}},
	}
	pass := p17ContractTestPass("bad-constant-fold")
	pass.Run = func(p *ir.IRProgram) error {
		p.Funcs[0].Instrs[0].Imm = 2
		return nil
	}
	_, err := NewManager().Run(prog, pass)
	if err == nil || !strings.Contains(err.Error(), "semantic local equivalence") {
		t.Fatalf("Run error = %v, want semantic translation validation failure", err)
	}
}

func TestManagerIncludesTranslationReportEvidence(t *testing.T) {
	prog := validTinyProgram()
	report, err := NewManager().Run(prog, p17ContractTestPass("noop-translation"))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	row := report.Passes[0]
	if !row.TranslationValidated || row.TranslationReport == nil {
		t.Fatalf("translation report evidence missing: %#v", row)
	}
	if row.TranslationReport.FunctionsCompared != 1 || row.TranslationReport.SemanticLocalChecks != 1 {
		t.Fatalf("translation report = %+v, want function and semantic evidence", row.TranslationReport)
	}
	if row.ValidationMetadata == nil {
		t.Fatalf("validation metadata evidence missing: %#v", row)
	}
	if row.ValidationMetadata.SchemaVersion != "tetra.translation.validation.metadata.v1" || row.ValidationMetadata.BeforeHash == "" || row.ValidationMetadata.AfterHash == "" {
		t.Fatalf("validation metadata = %+v", row.ValidationMetadata)
	}
}

func TestManagerAcceptsProfileInputAsValidatedMetadataWithoutChangingIR(t *testing.T) {
	prog := validTinyProgram()
	before := FormatProgram(prog)
	profile := ProfileCollection{
		SchemaVersion: ProfileCollectionSchemaVersion,
		ProgramHash:   "sha256:managerprofile",
		TargetTriple:  "linux-x64",
		Functions: []ProfileFunction{{
			ID:         "fn:main",
			Name:       "main",
			EntryCount: 99,
			Counters: []ProfileCounter{
				{Kind: "edge", Name: "return", Count: 99},
			},
		}},
	}

	report, err := NewManager().RunWithOptions(prog, Options{ProfileInput: &profile}, p17ContractTestPass("noop-profile-input"))
	if err != nil {
		t.Fatalf("RunWithOptions: %v", err)
	}
	after := FormatProgram(prog)
	if before != after {
		t.Fatalf("profile input changed IR:\nbefore:\n%s\nafter:\n%s", before, after)
	}
	row := report.Passes[0]
	if row.ProfileInputPolicy != ProfileInputUnused {
		t.Fatalf("profile policy = %q, want %q", row.ProfileInputPolicy, ProfileInputUnused)
	}
	if row.ProfileInput == nil {
		t.Fatalf("profile input evidence missing: %#v", row)
	}
	if row.ProfileInput.SchemaVersion != ProfileCollectionSchemaVersion || row.ProfileInput.ProgramHash != "sha256:managerprofile" || row.ProfileInput.Functions != 1 || row.ProfileInput.TotalEntryCount != 99 {
		t.Fatalf("profile input evidence = %+v", row.ProfileInput)
	}
	if !strings.HasPrefix(row.ProfileInput.Digest, "sha256:") {
		t.Fatalf("profile digest = %q, want sha256", row.ProfileInput.Digest)
	}
	if !containsString(row.ProfileInput.CounterKinds, "edge") {
		t.Fatalf("profile counter kinds = %#v, want edge", row.ProfileInput.CounterKinds)
	}
	if row.ValidationMetadata == nil {
		t.Fatalf("validation metadata missing: %#v", row)
	}
	if row.ValidationMetadata.ProfileInputPolicy != string(ProfileInputUnused) || row.ValidationMetadata.ProfileInputDigest != row.ProfileInput.Digest {
		t.Fatalf("validation profile metadata mismatch: row=%#v metadata=%+v", row.ProfileInput, row.ValidationMetadata)
	}
}

func TestManagerRejectsProfileGuidedRewritePolicyUntilValidationExists(t *testing.T) {
	pass := p17ContractTestPass("bad-profile-guided-rewrite")
	pass.ProfileInputPolicy = ProfileInputGuidedRewrite
	_, err := NewManager().Run(validTinyProgram(), pass)
	if err == nil || !strings.Contains(err.Error(), "profile-guided optimizer decisions require dedicated validation") {
		t.Fatalf("Run error = %v, want profile-guided validation rejection", err)
	}
}

func TestRegisteredOptimizerPassesExposeP17ContractEvidence(t *testing.T) {
	passes := RegisteredPasses()
	wantNames := map[string]bool{
		"basic-scalar":              false,
		"inline-small-pure":         false,
		"licm-pure-invariant":       false,
		"loop-canonicalization":     false,
		"mem2reg-single-assignment": false,
		"sccp-constant-branch":      false,
	}
	if len(passes) != len(wantNames) {
		t.Fatalf("registered passes = %d, want %d: %#v", len(passes), len(wantNames), passes)
	}
	for _, pass := range passes {
		if _, ok := wantNames[pass.Name]; !ok {
			t.Fatalf("unexpected registered pass %q", pass.Name)
		}
		wantNames[pass.Name] = true
		if err := ValidatePassContract(pass); err != nil {
			t.Fatalf("registered pass %q contract invalid: %v", pass.Name, err)
		}
	}
	for name, seen := range wantNames {
		if !seen {
			t.Fatalf("registered pass %q missing", name)
		}
	}

	report, err := NewManager().Run(validTinyProgram(), passes...)
	if err != nil {
		t.Fatalf("Run registered passes: %v", err)
	}
	if len(report.Passes) != len(passes) {
		t.Fatalf("report passes = %d, want %d", len(report.Passes), len(passes))
	}
	for _, row := range report.Passes {
		if row.InputVerifier != VerifierLowerVerifyProgram || row.OutputVerifier != VerifierLowerVerifyProgram {
			t.Fatalf("%s verifier evidence missing: %#v", row.Name, row)
		}
		if row.ProofRule != ProofRulePreserveBoundsInvalidateLiveness {
			t.Fatalf("%s proof rule = %q, want %q", row.Name, row.ProofRule, ProofRulePreserveBoundsInvalidateLiveness)
		}
		if row.TranslationValidationHook != TranslationHookValidateTranslation || !row.TranslationValidated {
			t.Fatalf("%s translation hook evidence missing: %#v", row.Name, row)
		}
		if row.ProfileInputPolicy != ProfileInputUnused {
			t.Fatalf("%s profile input policy = %q, want %q", row.Name, row.ProfileInputPolicy, ProfileInputUnused)
		}
		for _, want := range RequiredP17ReportRows() {
			if !containsString(row.ReportRows, want) {
				t.Fatalf("%s report rows missing %q: %#v", row.Name, want, row.ReportRows)
			}
		}
		if row.NegativeTestMarker != NegativeTestPassContractV1 {
			t.Fatalf("%s negative-test marker = %q, want %q", row.Name, row.NegativeTestMarker, NegativeTestPassContractV1)
		}
		if row.ValidationMetadata == nil {
			t.Fatalf("%s missing validation metadata", row.Name)
		}
		if row.ValidationMetadata.InputVerifier != row.InputVerifier || row.ValidationMetadata.OutputVerifier != row.OutputVerifier {
			t.Fatalf("%s validation metadata verifier mismatch: row=%#v metadata=%+v", row.Name, row, row.ValidationMetadata)
		}
		if row.ValidationMetadata.ProofRule != string(row.ProofRule) || row.ValidationMetadata.TranslationValidationHook != row.TranslationValidationHook {
			t.Fatalf("%s validation metadata contract mismatch: row=%#v metadata=%+v", row.Name, row, row.ValidationMetadata)
		}
		if row.ValidationMetadata.ProfileInputPolicy != string(ProfileInputUnused) {
			t.Fatalf("%s validation profile metadata = %+v", row.Name, row.ValidationMetadata)
		}
	}
}

func TestManagerRejectsIncompletePassContractEvidence(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Pass)
		want   string
	}{
		{
			name:   "missing input verifier",
			mutate: func(pass *Pass) { pass.InputVerifier = "" },
			want:   "missing input verifier",
		},
		{
			name:   "fake input verifier",
			mutate: func(pass *Pass) { pass.InputVerifier = "paper.input.Verifier" },
			want:   "unsupported input verifier",
		},
		{
			name:   "missing output verifier",
			mutate: func(pass *Pass) { pass.OutputVerifier = "" },
			want:   "missing output verifier",
		},
		{
			name:   "fake output verifier",
			mutate: func(pass *Pass) { pass.OutputVerifier = "paper.output.Verifier" },
			want:   "unsupported output verifier",
		},
		{
			name:   "missing proof rule",
			mutate: func(pass *Pass) { pass.ProofRule = "" },
			want:   "missing proof preservation or invalidation rule",
		},
		{
			name:   "fake proof rule",
			mutate: func(pass *Pass) { pass.ProofRule = "trust_me_preserved" },
			want:   "unknown proof preservation or invalidation rule",
		},
		{
			name:   "missing translation hook",
			mutate: func(pass *Pass) { pass.TranslationValidationHook = "" },
			want:   "missing translation validation hook",
		},
		{
			name:   "fake translation hook",
			mutate: func(pass *Pass) { pass.TranslationValidationHook = "paper.translation.Hook" },
			want:   "unsupported translation validation hook",
		},
		{
			name:   "missing report rows",
			mutate: func(pass *Pass) { pass.ReportRows = nil },
			want:   "missing report rows",
		},
		{
			name:   "missing required report row",
			mutate: func(pass *Pass) { pass.ReportRows = []string{"before_dump", "after_dump"} },
			want:   "missing required report row",
		},
		{
			name:   "missing profile input policy",
			mutate: func(pass *Pass) { pass.ProfileInputPolicy = "" },
			want:   "missing profile input policy",
		},
		{
			name:   "unsupported profile guided policy",
			mutate: func(pass *Pass) { pass.ProfileInputPolicy = ProfileInputGuidedRewrite },
			want:   "profile-guided optimizer decisions require dedicated validation",
		},
		{
			name:   "missing negative-test marker",
			mutate: func(pass *Pass) { pass.NegativeTestMarker = "" },
			want:   "missing negative-test marker",
		},
		{
			name:   "fake negative-test marker",
			mutate: func(pass *Pass) { pass.NegativeTestMarker = "paper-negative-tests" },
			want:   "unknown negative-test marker",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pass := p17ContractTestPass("contracted-noop")
			tc.mutate(&pass)
			_, err := NewManager().Run(validTinyProgram(), pass)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Run error = %v, want %q", err, tc.want)
			}
		})
	}
}

func p17ContractTestPass(name string) Pass {
	return Pass{
		Name:                      name,
		InputKind:                 IRKindStack,
		OutputKind:                IRKindStack,
		InputVerifier:             VerifierLowerVerifyProgram,
		OutputVerifier:            VerifierLowerVerifyProgram,
		RequiredFacts:             []Fact{FactIRVerified},
		PreservedFacts:            []Fact{FactBoundsProofs},
		InvalidatedFacts:          []Fact{FactLiveness},
		ProofRule:                 ProofRulePreserveBoundsInvalidateLiveness,
		ValidationStrategy:        ValidationTranslation,
		TranslationValidationHook: TranslationHookValidateTranslation,
		ReportOutput:              name + ".opt.json",
		ReportRows:                RequiredP17ReportRows(),
		NegativeTestMarker:        NegativeTestPassContractV1,
		ProfileInputPolicy:        ProfileInputUnused,
		Run:                       func(p *ir.IRProgram) error { return nil },
	}
}

func validTinyProgram() *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []ir.IRFunc{{
			Name:        "main",
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 1},
				{Kind: ir.IRReturn},
			},
		}},
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
