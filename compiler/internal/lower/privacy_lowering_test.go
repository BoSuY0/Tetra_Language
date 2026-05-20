package lower

import (
	"testing"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/semantics"
)

func TestLowerPrivacySealUnsealI32DeterministicShapeAndNoSideEffects(t *testing.T) {
	src := []byte(`
func seal(token: consent.token, value: Int) -> secret.i32
uses privacy
privacy
consent(token):
    return core.secret_seal_i32(value, token)

func unseal(token: consent.token, value: secret.i32) -> Int
uses privacy
privacy
consent(token):
    return core.secret_unseal_i32(value, token)

func main() -> Int:
    return 0
`)
	prog, err := frontend.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := semantics.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	irProg, err := Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}

	seal := findLoweredFuncByName(t, irProg, "seal")
	unseal := findLoweredFuncByName(t, irProg, "unseal")
	wantPattern := []ir.IRInstrKind{ir.IRConstI32, ir.IRMulI32, ir.IRAddI32}

	if got := countKindPattern(seal.Instrs, wantPattern); got != 1 {
		t.Fatalf("seal lowering pattern count = %d, want 1; instrs=%#v", got, seal.Instrs)
	}
	if got := countKindPattern(unseal.Instrs, wantPattern); got != 1 {
		t.Fatalf("unseal lowering pattern count = %d, want 1; instrs=%#v", got, unseal.Instrs)
	}

	for _, fn := range []ir.IRFunc{seal, unseal} {
		assertNoPrivacySideEffects(t, fn)
	}
}

func TestLowerConsentTokenUsesOpaqueRuntimeSentinel(t *testing.T) {
	src := []byte(`
func require_token(token: consent.token) -> Int
uses privacy
privacy
consent(token):
    return 7

func main() -> Int
uses privacy
privacy:
    let token: consent.token = core.consent_token()
    return require_token(token)
`)
	prog, err := frontend.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := semantics.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	irProg, err := Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}

	requireToken := findLoweredFuncByName(t, irProg, "require_token")
	tokenSentinel := assertExactConsentGuard(t, requireToken)
	if tokenSentinel == 0 || tokenSentinel == 1 {
		t.Fatalf("consent token sentinel = %d, want opaque non-zero/non-one value", tokenSentinel)
	}

	mainFn := findLoweredFuncByName(t, irProg, "main")
	if !containsConstI32(mainFn.Instrs, tokenSentinel) {
		t.Fatalf("main does not mint the consent sentinel %d; instrs=%#v", tokenSentinel, mainFn.Instrs)
	}
	if containsConstI32(mainFn.Instrs, 1) {
		t.Fatalf("main still appears to mint forgeable consent token constant 1; instrs=%#v", mainFn.Instrs)
	}
}

func findLoweredFuncByName(t *testing.T, prog *ir.IRProgram, name string) ir.IRFunc {
	t.Helper()
	for _, fn := range prog.Funcs {
		if fn.Name == name {
			return fn
		}
	}
	t.Fatalf("lowered function %q not found", name)
	return ir.IRFunc{}
}

func countKindPattern(instrs []ir.IRInstr, pattern []ir.IRInstrKind) int {
	if len(pattern) == 0 {
		return 0
	}
	count := 0
	for i := 0; i+len(pattern) <= len(instrs); i++ {
		ok := true
		for j := range pattern {
			if instrs[i+j].Kind != pattern[j] {
				ok = false
				break
			}
		}
		if ok {
			count++
		}
	}
	return count
}

func assertExactConsentGuard(t *testing.T, fn ir.IRFunc) int32 {
	t.Helper()
	for i := 0; i+3 < len(fn.Instrs); i++ {
		if fn.Instrs[i].Kind == ir.IRLoadLocal &&
			fn.Instrs[i+1].Kind == ir.IRConstI32 &&
			fn.Instrs[i+2].Kind == ir.IRCmpEqI32 &&
			fn.Instrs[i+3].Kind == ir.IRJmpIfZero {
			return fn.Instrs[i+1].Imm
		}
	}
	t.Fatalf("%s missing exact consent guard; instrs=%#v", fn.Name, fn.Instrs)
	return 0
}

func containsConstI32(instrs []ir.IRInstr, imm int32) bool {
	for _, instr := range instrs {
		if instr.Kind == ir.IRConstI32 && instr.Imm == imm {
			return true
		}
	}
	return false
}

func assertNoPrivacySideEffects(t *testing.T, fn ir.IRFunc) {
	t.Helper()
	disallowed := map[ir.IRInstrKind]string{
		ir.IRCall:              "runtime call",
		ir.IRWrite:             "stdout write",
		ir.IRStoreGlobal:       "global storage write",
		ir.IRMemWriteI32:       "memory write i32",
		ir.IRMemWriteU8:        "memory write u8",
		ir.IRMemWritePtr:       "memory write ptr",
		ir.IRMemWriteI32Offset: "memory write i32 offset",
		ir.IRMemWriteU8Offset:  "memory write u8 offset",
		ir.IRMemWritePtrOffset: "memory write ptr offset",
		ir.IRMmioWriteI32:      "mmio write",
		ir.IRCtxSwitch:         "context switch",
	}
	for _, instr := range fn.Instrs {
		if reason, bad := disallowed[instr.Kind]; bad {
			t.Fatalf("%s contains disallowed %s instruction: %#v", fn.Name, reason, instr)
		}
	}
}
