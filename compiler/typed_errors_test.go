package compiler

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestTypedErrorsParseCheckAndLower(t *testing.T) {
	src := []byte(`
enum ReadError:
    case eof

func read(flag: Bool) -> Int throws ReadError:
    if flag:
        return 42
    else:
        throw ReadError.eof

func caller() -> Int throws ReadError:
    let value: Int = try read(true)
    return value

func main() -> Int:
    return 0
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !prog.Funcs[0].HasThrows || prog.Funcs[0].Throws.Name != "ReadError" {
		t.Fatalf("throws = %#v", prog.Funcs[0].Throws)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if got := checked.FuncSigs["read"].ThrowsType; got != "ReadError" {
		t.Fatalf("read throws = %q", got)
	}
	if _, err := Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestTypedErrorsRejectBareThrowingCall(t *testing.T) {
	src := []byte(`
enum E:
    case bad

func f() -> Int throws E:
    throw E.bad

func main() -> Int:
    return f()
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected bare throwing call error")
	}
	if !strings.Contains(err.Error(), "requires try") {
		t.Fatalf("error = %v", err)
	}
}

func TestTypedErrorsRejectTryOutsideThrowingFunction(t *testing.T) {
	src := []byte(`
enum E:
    case bad

func f() -> Int throws E:
    throw E.bad

func main() -> Int:
    return try f()
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected try context error")
	}
	if !strings.Contains(err.Error(), "try is only allowed in throwing functions") {
		t.Fatalf("error = %v", err)
	}
}

func TestTypedErrorsAllowMultiSlotErrorPayload(t *testing.T) {
	src := []byte(`
func fail(flag: Bool) -> Int throws String:
    if flag:
        return 7
    else:
        throw "bad"

func caller(flag: Bool) -> Int throws String:
    return try fail(flag)

func main() -> Int:
    return 0
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if got := checked.FuncSigs["fail"].ReturnSlots; got != 4 {
		t.Fatalf("fail return slots = %d, want 4", got)
	}
	if got := checked.FuncSigs["fail"].ThrowsType; got != "str" {
		t.Fatalf("fail throws type = %q, want str", got)
	}
	if _, err := Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestTypedErrorsAllowEnumPayloadError(t *testing.T) {
	src := []byte(`
enum ParseError:
    case unexpected(Int)
    case eof

func fail(flag: Bool) -> Int throws ParseError:
    if flag:
        return 7
    else:
        throw ParseError.unexpected(9)

func caller(flag: Bool) -> Int throws ParseError:
    return try fail(flag)

func main() -> Int:
    return 0
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if got := checked.FuncSigs["fail"].ThrowsType; got != "ParseError" {
		t.Fatalf("fail throws = %q, want ParseError", got)
	}
	if got := checked.FuncSigs["fail"].ReturnSlots; got != 4 {
		t.Fatalf("fail return slots = %d, want 4", got)
	}
	if _, err := Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestTypedErrorsTryPropagatesIntoOptionalThrows(t *testing.T) {
	src := []byte(`
func fail(flag: Bool) -> Int throws Int:
    if flag:
        return 7
    else:
        throw 11

func caller(flag: Bool) -> Int throws Int?:
    return try fail(flag)

func main() -> Int:
    return 0
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if got := checked.FuncSigs["caller"].ThrowsType; got != "i32?" {
		t.Fatalf("caller throws type = %q, want i32?", got)
	}
	if got := checked.FuncSigs["caller"].ReturnSlots; got != 4 {
		t.Fatalf("caller return slots = %d, want 4", got)
	}
	if _, err := Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestTypedErrorsTryPropagatesMultiSlotIntoOptionalThrows(t *testing.T) {
	src := []byte(`
func fail(flag: Bool) -> Int throws String:
    if flag:
        return 7
    else:
        throw "bad"

func caller(flag: Bool) -> Int throws String?:
    return try fail(flag)

func main() -> Int:
    return 0
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if got := checked.FuncSigs["caller"].ThrowsType; got != "str?" {
		t.Fatalf("caller throws type = %q, want str?", got)
	}
	if got := checked.FuncSigs["caller"].ReturnSlots; got != 5 {
		t.Fatalf("caller return slots = %d, want 5", got)
	}
	if _, err := Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestTypedErrorsGenericEnumThrowMonomorphizes(t *testing.T) {
	src := []byte(`
enum ReadError:
    case eof

func fail<T>(err: T) -> Int throws T:
    throw err

func caller() -> Int throws ReadError:
    let err: ReadError = ReadError.eof
    return try fail(err)

func main() -> Int:
    return 0
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if got := checked.FuncSigs["fail__T_ReadError"].ThrowsType; got != "ReadError" {
		t.Fatalf("monomorphized fail throws = %q, want ReadError", got)
	}
	if _, err := Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestTypedErrorsRejectWrongThrowType(t *testing.T) {
	src := []byte(`
enum ReadError:
    case eof

func read(flag: Bool) -> Int throws ReadError:
    if flag:
        return 1
    throw 7

func main() -> Int:
    return 0
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected throw type mismatch")
	}
	if !strings.Contains(err.Error(), "throw type mismatch: expected 'ReadError', got 'i32'") {
		t.Fatalf("error = %v", err)
	}
}

func TestTypedErrorsImportedThrowingFunctionCheckAndLower(t *testing.T) {
	files := map[string]string{
		"engine/errors.tetra": `module engine.errors
enum ReadError:
    case eof

func read(flag: Bool) -> Int throws ReadError:
    if flag:
        return 42
    throw ReadError.eof
`,
		"app/main.tetra": `module app.main
import engine.errors as errors

func caller(flag: Bool) -> Int throws errors.ReadError:
    return try errors.read(flag)

func main() -> Int:
    return 0
`,
	}
	tmp := t.TempDir()
	writeTestFiles(t, tmp, files)
	entry := filepath.Join(tmp, filepath.FromSlash("app/main.tetra"))

	world, err := LoadWorld(entry)
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	checked, err := CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	if got := checked.FuncSigs["engine.errors.read"].ThrowsType; got != "engine.errors.ReadError" {
		t.Fatalf("imported read throws type = %q, want engine.errors.ReadError", got)
	}
	if got := checked.FuncSigs["app.main.caller"].ThrowsType; got != "engine.errors.ReadError" {
		t.Fatalf("caller throws type = %q, want engine.errors.ReadError", got)
	}
	if _, err := LowerModules(checked); err != nil {
		t.Fatalf("LowerModules: %v", err)
	}
}

func TestTypedErrorsCatchExpressionEnumPayloadSmoke(t *testing.T) {
	src := []byte(`
enum ReadError:
    case eof
    case denied(Int)

func read(flag: Bool) -> Int throws ReadError:
    if flag:
        return 42
    throw ReadError.denied(7)

func main() -> Int:
    let value: Int = catch read(false):
    case ReadError.eof:
        0
    case ReadError.denied(code):
        code
    return value
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if _, err := Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestTypedErrorsCatchPayloadCaseRequiresDestructuringDiagnostic(t *testing.T) {
	src := []byte(`
enum ReadError:
    case eof
    case denied(Int)

func read() -> Int throws ReadError:
    throw ReadError.denied(7)

func main() -> Int:
    return catch read():
    case ReadError.eof:
        0
    case ReadError.denied:
        1
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected catch payload destructuring diagnostic")
	}
	if !strings.Contains(err.Error(), "carries 1 payload value(s); use 'ReadError.denied(value1)'") {
		t.Fatalf("error = %v", err)
	}
}

func TestTypedErrorsCatchNoPayloadCaseRejectsPayloadSyntaxDiagnostic(t *testing.T) {
	src := []byte(`
enum ReadError:
    case eof
    case denied(Int)

func read() -> Int throws ReadError:
    throw ReadError.eof

func main() -> Int:
    return catch read():
    case ReadError.eof(code):
        code
    case ReadError.denied(code):
        code
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected catch no-payload pattern diagnostic")
	}
	if !strings.Contains(err.Error(), "has no payload; use 'ReadError.eof'") {
		t.Fatalf("error = %v", err)
	}
}

func TestTypedErrorsCatchRejectsNonThrowingCall(t *testing.T) {
	src := []byte(`
func read() -> Int:
    return 42

func main() -> Int:
    return catch read():
    case _:
        0
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected catch non-throwing call error")
	}
	if !strings.Contains(err.Error(), "catch expects a throwing function call") {
		t.Fatalf("error = %v", err)
	}
}

func TestTypedErrorsCatchBindingScopeDiagnostic(t *testing.T) {
	src := []byte(`
enum ReadError:
    case denied(Int)

func read() -> Int throws ReadError:
    throw ReadError.denied(7)

func main() -> Int:
    let value: Int = catch read():
    case ReadError.denied(code):
        code
    return code
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected catch binding scope error")
	}
	if !strings.Contains(err.Error(), "identifier 'code' is out of scope") {
		t.Fatalf("error = %v", err)
	}
}

func TestTypedErrorsCatchRequiresExhaustiveCases(t *testing.T) {
	src := []byte(`
enum ReadError:
    case eof
    case denied(Int)

func read() -> Int throws ReadError:
    throw ReadError.eof

func main() -> Int:
    return catch read():
    case ReadError.eof:
        0
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected catch exhaustiveness error")
	}
	if !strings.Contains(err.Error(), "catch expression must be exhaustive") {
		t.Fatalf("error = %v", err)
	}
}

func TestTypedErrorsCatchRejectsHandlerTypeMismatch(t *testing.T) {
	src := []byte(`
enum ReadError:
    case eof

func read() -> Int throws ReadError:
    throw ReadError.eof

func main() -> Int:
    return catch read():
    case ReadError.eof:
        "bad"
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected catch handler type mismatch")
	}
	if !strings.Contains(err.Error(), "catch expression case type mismatch") {
		t.Fatalf("error = %v", err)
	}
}

func TestTypedErrorsCatchGuardEnumPayloadSmoke(t *testing.T) {
	src := []byte(`
enum ReadError:
    case denied(Int)

func read() -> Int throws ReadError:
    throw ReadError.denied(7)

func main() -> Int:
    return catch read():
    case ReadError.denied(code) if code > 0:
        code
    case ReadError.denied(other):
        other
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if _, err := Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestTypedErrorsCatchGuardedEnumPayloadCaseIsNotExhaustive(t *testing.T) {
	src := []byte(`
enum ReadError:
    case denied(Int)
    case eof

func read() -> Int throws ReadError:
    throw ReadError.denied(7)

func main() -> Int:
    return catch read():
    case ReadError.denied(code) if code > 0:
        code
    case ReadError.eof:
        0
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected guarded catch exhaustiveness error")
	}
	if !strings.Contains(err.Error(), "catch expression must be exhaustive") {
		t.Fatalf("error = %v", err)
	}
}

func TestTypedErrorsCatchDuplicateUnguardedEnumPayloadCaseDiagnostic(t *testing.T) {
	src := []byte(`
enum ReadError:
    case denied(Int)

func read() -> Int throws ReadError:
    throw ReadError.denied(7)

func main() -> Int:
    return catch read():
    case ReadError.denied(code):
        code
    case ReadError.denied(other):
        other
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected duplicate catch enum payload case diagnostic")
	}
	if !strings.Contains(err.Error(), "duplicate catch pattern") {
		t.Fatalf("error = %v", err)
	}
}

func TestTypedErrorsCatchDefaultMustBeLastDiagnostic(t *testing.T) {
	src := []byte(`
enum ReadError:
    case eof
    case denied(Int)

func read() -> Int throws ReadError:
    throw ReadError.eof

func main() -> Int:
    return catch read():
    case _:
        0
    case ReadError.eof:
        1
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected catch default ordering diagnostic")
	}
	if !strings.Contains(err.Error(), "catch default must be last") {
		t.Fatalf("error = %v", err)
	}
}

func TestTypedErrorsCatchRejectsWrongEnumCaseDiagnostic(t *testing.T) {
	src := []byte(`
enum ReadError:
    case denied(Int)
enum WriteError:
    case denied(Int)

func read() -> Int throws ReadError:
    throw ReadError.denied(7)

func main() -> Int:
    return catch read():
    case WriteError.denied(code):
        code
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected wrong catch enum case diagnostic")
	}
	if !strings.Contains(err.Error(), "enum pattern type mismatch") && !strings.Contains(err.Error(), "catch pattern type mismatch") {
		t.Fatalf("error = %v", err)
	}
}

func TestTypedErrorsCatchRuntimeErrorAndSuccessPaths(t *testing.T) {
	src := `
enum ReadError:
    case eof

func read(flag: Bool) -> Int throws ReadError:
    if flag:
        return 35
    throw ReadError.eof

func recover(flag: Bool) -> Int:
    return catch read(flag):
    case ReadError.eof:
        7

func main() -> Int:
    return recover(false) + recover(true)
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code = %d, want 42", exitCode)
	}
}
