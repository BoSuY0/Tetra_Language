package compiler_test

import (
	"path/filepath"
	"strings"
	"testing"

	compiler "tetra_language/compiler"
)

func TestFunctionTypedLocalDeclaredEffectsPropagate(t *testing.T) {
	requireCheckErrorContains(t, `
func main() -> Int:
  let f: fn(Int) -> Int uses io = fn(x: Int) -> Int
  uses io:
    print("call\n")
    return x
  return f(41)
`, "uses effect 'io'")

	requireCheckOK(t, `
func main() -> Int
uses io:
  let f: fn(Int) -> Int uses io = fn(x: Int) -> Int
  uses io:
    print("call\n")
    return x
  return f(41)
`)
}

func TestFunctionTypeDeclaredEffectsEnforcedForCallbackBody(t *testing.T) {
	requireCheckErrorContains(t, `
func apply(x: Int, cb: fn(Int) -> Int uses io) -> Int:
  return cb(x)

func main() -> Int:
  return 0
`, "function 'apply' uses effect 'io'")

	requireCheckOK(t, `
func say(x: Int) -> Int
uses io:
  print("call\n")
  return x

func apply(x: Int, cb: fn(Int) -> Int uses io) -> Int
uses io:
  return cb(x)

func main() -> Int
uses io:
  return apply(41, say)
`)
}

func TestCallbackWrapperDeclaredEffectsCannotBypassSemanticClauses(t *testing.T) {
	requireCheckErrorContains(t, `
func allocer(x: Int) -> Int
uses alloc, mem:
  unsafe:
    let _: ptr = core.alloc_bytes(4)
  return x

func apply(x: Int, cb: fn(Int) -> Int uses alloc, mem) -> Int
uses alloc, mem:
  return cb(x)

func main() -> Int
noalloc:
  return apply(41, allocer)
`, "semantic clause 'noalloc' forbids call")

	requireCheckErrorContains(t, `
func sleeper(x: Int) -> Int
uses runtime:
  let _: Int = core.sleep_ms(1)
  return x

func apply(x: Int, cb: fn(Int) -> Int uses runtime) -> Int
uses runtime:
  return cb(x)

func main() -> Int
noblock:
  return apply(41, sleeper)
`, "semantic clause 'noblock' forbids call")

	requireCheckErrorContains(t, `
func sleeper(x: Int) -> Int
uses runtime:
  let _: Int = core.sleep_ms(1)
  return x

func apply(x: Int, cb: fn(Int) -> Int uses runtime) -> Int
uses runtime:
  return cb(x)

func main() -> Int
realtime
noalloc
noblock:
  return apply(41, sleeper)
`, "semantic clause 'realtime' forbids call")
}

func TestImportedCallbackTargetDeclaredEffectsPropagate(t *testing.T) {
	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

func allocer(x: Int) -> Int
uses alloc, mem:
  unsafe:
    let _: ptr = core.alloc_bytes(4)
  return x

func apply(x: Int, cb: fn(Int) -> Int uses alloc, mem) -> Int
uses alloc, mem:
  return cb(x)
`,
		"app/main.t4": `module app.main
import lib.callbacks.{apply, allocer}

func main() -> Int:
  return apply(41, allocer)
`,
	}
	tmp := t.TempDir()
	writeEffectsTestFiles(t, tmp, files)
	world, err := compiler.LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/main.t4")))
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected imported callback target declared effect propagation error")
	}
	for _, want := range []string{"function 'app.main.main'", "uses effect 'alloc'"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want substring %q", err, want)
		}
	}
}

func TestReturnedFunctionTypedValuesPropagateEffects(t *testing.T) {
	requireCheckErrorContains(t, `
func allocer(x: Int) -> Int
uses alloc, mem:
  unsafe:
    let _: ptr = core.alloc_bytes(4)
  return x

func pick() -> fn(Int) -> Int uses alloc, mem:
  let f: fn(Int) -> Int uses alloc, mem = allocer
  return f

func main() -> Int:
  let f: fn(Int) -> Int uses alloc, mem = pick()
  return f(41)
`, "uses effect 'alloc'")

	requireCheckErrorContains(t, `
func allocer(x: Int) -> Int
uses alloc, mem:
  unsafe:
    let _: ptr = core.alloc_bytes(4)
  return x

func pick() -> fn(Int) -> Int uses alloc, mem:
  let f: fn(Int) -> Int uses alloc, mem = allocer
  return f

func main() -> Int
noalloc:
  let f: fn(Int) -> Int uses alloc, mem = pick()
  return f(41)
`, "semantic clause 'noalloc' forbids call")
}

func TestTargetSetCallbackAllowedUnderSemanticClauseWhenDeclaredEffectsAreSafe(t *testing.T) {
	requireCheckOK(t, `
func add1(x: Int) -> Int:
  return x + 1

func add2(x: Int) -> Int:
  return x + 2

func pick(use_second: Int) -> fn(Int) -> Int:
  if use_second:
    return add2
  else:
    return add1

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
  return cb(x)

func main() -> Int
noalloc:
  let cb: fn(Int) -> Int = pick(0)
  return apply(cb, 41)
`)
}

func TestSemanticClauseAllowsDeclaredSafeFunctionTypeCallbackBody(t *testing.T) {
	requireCheckOK(t, `
func add1(x: Int) -> Int:
  return x + 1

func apply(cb: fn(Int) -> Int, x: Int) -> Int
noalloc:
  return cb(x)

func main() -> Int:
  return apply(add1, 41)
`)

	requireCheckErrorContains(t, `
func apply(x: Int, cb: fn(Int) -> Int uses alloc, mem) -> Int
noalloc:
  return cb(x)

func main() -> Int:
  return 0
`, "semantic clause 'noalloc' forbids call to 'cb' because it may allocate")
}

func TestSemanticClauseFunctionTypedGlobalDirectCallDiagnosticUsesGlobalName(t *testing.T) {
	requireFileCheckErrorContains(t, `
val cb: fn(Int) -> Int uses alloc, mem = allocer

func allocer(x: Int) -> Int
uses alloc, mem:
  unsafe:
    let _: ptr = core.alloc_bytes(4)
  return x

func main() -> Int
noalloc:
  return cb(1)
`, "semantic clause 'noalloc' forbids function-typed global call 'cb' because it may allocate")
}

func TestSemanticClauseFunctionTypedStructFieldDirectCallDiagnosticUsesFieldName(t *testing.T) {
	requireCheckErrorContains(t, `
struct Holder:
  cb: fn(Int) -> Int uses alloc, mem

func allocer(x: Int) -> Int
uses alloc, mem:
  unsafe:
    let _: ptr = core.alloc_bytes(4)
  return x

func main() -> Int
noalloc:
  let holder: Holder = Holder(cb: allocer)
  return holder.cb(1)
`, ("semantic clause 'noalloc' forbids function-typed struct field " +
		"call 'holder.cb' because it may allocate"))
}

func TestSemanticClauseFunctionTypedLocalDirectCallDiagnosticUsesLocalName(t *testing.T) {
	requireCheckErrorContains(t, `
func allocer(x: Int) -> Int
uses alloc, mem:
  unsafe:
    let _: ptr = core.alloc_bytes(4)
  return x

func main() -> Int
noalloc:
  let f: fn(Int) -> Int uses alloc, mem = allocer
  return f(1)
`, "semantic clause 'noalloc' forbids call to callback 'f' because it may allocate")
}

func TestSemanticClauseCapturedFunctionTypedLocalDirectCallDiagnosticUsesLocalName(t *testing.T) {
	requireCheckErrorContains(t, `
func main() -> Int
noalloc:
  let base: Int = 1
  let f: fn(Int) -> Int uses alloc, mem = fn(x: Int) -> Int
  uses alloc, mem:
    unsafe:
      let _: ptr = core.alloc_bytes(4)
    return x + base
  return f(41)
`, "semantic clause 'noalloc' forbids function-typed callback 'f' because it may allocate")
}

func TestSemanticClauseFunctionTypedEnumPayloadDirectCallDiagnosticUsesBindingName(t *testing.T) {
	requireCheckErrorContains(t, `
enum MaybeCallback:
  case some(fn(Int) -> Int uses alloc, mem)
  case empty

func allocer(x: Int) -> Int
uses alloc, mem:
  unsafe:
    let _: ptr = core.alloc_bytes(4)
  return x

func main() -> Int
noalloc:
  let choice: MaybeCallback = MaybeCallback.some(allocer)
  match choice:
  case MaybeCallback.some(cb):
    return cb(1)
  case MaybeCallback.empty:
    return 0
`, ("semantic clause 'noalloc' forbids function-typed enum payload " +
		"call 'cb' because it may allocate"))
}

func TestSemanticClauseFunctionTypedLocalCallbackArgumentDiagnosticUsesArgumentName(t *testing.T) {
	requireCheckErrorContains(t, `
func allocer(x: Int) -> Int
uses alloc, mem:
  unsafe:
    let _: ptr = core.alloc_bytes(4)
  return x

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
  return cb(x)

func main() -> Int
noalloc:
  let f: fn(Int) -> Int uses alloc, mem = allocer
  return apply(f, 41)
`, "semantic clause 'noalloc' forbids callback argument 'f' because it may allocate")
}

func TestSemanticClauseFunctionTypedGlobalCallbackArgumentDiagnosticUsesArgumentName(t *testing.T) {
	requireFileCheckErrorContains(t, `
val cb: fn(Int) -> Int uses alloc, mem = allocer

func allocer(x: Int) -> Int
uses alloc, mem:
  unsafe:
    let _: ptr = core.alloc_bytes(4)
  return x

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
  return cb(x)

func main() -> Int
noalloc:
  return apply(cb, 41)
`, "semantic clause 'noalloc' forbids callback argument 'cb' because it may allocate")
}

func TestSemanticClauseFunctionTypedStructFieldCallbackArgumentDiagnosticUsesFieldName(
	t *testing.T,
) {
	requireCheckErrorContains(t, `
struct Holder:
  cb: fn(Int) -> Int uses alloc, mem

func allocer(x: Int) -> Int
uses alloc, mem:
  unsafe:
    let _: ptr = core.alloc_bytes(4)
  return x

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
  return cb(x)

func main() -> Int
noalloc:
  let holder: Holder = Holder(cb: allocer)
  return apply(holder.cb, 41)
`, "semantic clause 'noalloc' forbids callback argument 'holder.cb' because it may allocate")
}

func TestSemanticClauseFunctionTypedReturnCallCallbackArgumentDiagnosticUsesCallName(t *testing.T) {
	requireCheckErrorContains(t, `
func allocer(x: Int) -> Int
uses alloc, mem:
  unsafe:
    let _: ptr = core.alloc_bytes(4)
  return x

func pick() -> fn(Int) -> Int uses alloc, mem:
  let f: fn(Int) -> Int uses alloc, mem = allocer
  return f

func apply(x: Int, cb: fn(Int) -> Int) -> Int:
  return cb(x)

func main() -> Int:
  return apply(41, pick())
`, ("callback function symbol 'pick()' requires effects alloc, mem " +
		"but function type does not declare them"))
}

func TestDirectClosureLiteralCallbackArgumentDiagnosticUsesClosureLiteralName(t *testing.T) {
	requireCheckErrorContains(t, `
func apply(x: Int, cb: fn(Int) -> Int) -> Int:
  return cb(x)

func main() -> Int:
  return apply(41, fn(x: Int) -> Int
  uses alloc, mem:
    unsafe:
      let _: ptr = core.alloc_bytes(4)
    return x
  )
`, ("callback argument 'closure literal' requires effects alloc, mem " +
		"but function type does not declare them"))
}

func TestSemanticClauseAllowsDeclaredSafeFunctionTypeCallbackBodyForNoblockRealtime(t *testing.T) {
	requireCheckOK(t, `
func add1(x: Int) -> Int:
  return x + 1

func apply(cb: fn(Int) -> Int, x: Int) -> Int
noblock:
  return cb(x)

func main() -> Int:
  return apply(add1, 41)
`)

	requireCheckOK(t, `
func add1(x: Int) -> Int:
  return x + 1

func apply(cb: fn(Int) -> Int, x: Int) -> Int
realtime
noalloc
noblock:
  return cb(x)

func main() -> Int:
  return apply(add1, 41)
`)

	requireCheckErrorContains(t, `
func apply(x: Int, cb: fn(Int) -> Int uses runtime) -> Int
noblock:
  return cb(x)

func main() -> Int:
  return 0
`, "semantic clause 'noblock' forbids call to 'cb' because it may block")

	requireCheckErrorContains(t, `
func apply(x: Int, cb: fn(Int) -> Int uses runtime) -> Int
realtime
noalloc
noblock:
  return cb(x)

func main() -> Int:
  return 0
`, "semantic clause 'realtime' forbids call to 'cb' because it is not realtime-safe")
}

func TestSemanticClauseAllowsTargetSetStructAndEnumCallbackArgumentsWhenDeclaredSafe(t *testing.T) {
	requireCheckOK(t, `
struct Holder:
  cb: fn(Int) -> Int

func add1(x: Int) -> Int:
  return x + 1

func add2(x: Int) -> Int:
  return x + 2

func pick(use_second: Int) -> fn(Int) -> Int:
  if use_second:
    return add2
  else:
    return add1

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
  return cb(x)

func main() -> Int
noalloc:
  let holder: Holder = Holder(cb: pick(0))
  return apply(holder.cb, 41)
`)

	requireCheckOK(t, `
enum MaybeCallback:
  case some(fn(Int) -> Int)
  case empty

func add1(x: Int) -> Int:
  return x + 1

func add2(x: Int) -> Int:
  return x + 2

func pick(use_second: Int) -> fn(Int) -> Int:
  if use_second:
    return add2
  else:
    return add1

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
  return cb(x)

func main() -> Int
noalloc:
  let choice: MaybeCallback = MaybeCallback.some(pick(0))
  match choice:
  case MaybeCallback.some(cb):
    return apply(cb, 41)
  case MaybeCallback.empty:
    return 0
`)
}

func TestSemanticClauseRejectsDeclaredEffectStructCallbackArgument(t *testing.T) {
	requireCheckErrorContains(t, `
struct Holder:
  cb: fn(Int) -> Int uses alloc, mem

func allocer(x: Int) -> Int
uses alloc, mem:
  unsafe:
    let _: ptr = core.alloc_bytes(4)
  return x

func apply(x: Int, cb: fn(Int) -> Int uses alloc, mem) -> Int
uses alloc, mem:
  return cb(x)

func main() -> Int
noalloc:
  let holder: Holder = Holder(cb: allocer)
  return apply(41, holder.cb)
`, "semantic clause 'noalloc' forbids call")
}

func TestFunctionTypeDeclaredEffectsRejectUndeclaredTargets(t *testing.T) {
	requireCheckErrorContains(t, `
func allocer(x: Int) -> Int
uses alloc, mem:
  unsafe:
    let _: ptr = core.alloc_bytes(4)
  return x

func main() -> Int:
  let f: fn(Int) -> Int uses alloc = allocer
  return f(41)
`, "requires effects mem but function type does not declare them")
}

func TestPrivacyEffectRequiresPrivacyClause(t *testing.T) {
	requireCheckErrorContains(t, `
func main() -> Int
uses privacy:
  return 0
`, "requires semantic clause 'privacy'")
}

func TestPrivacyConsentSecretSignatureChecks(t *testing.T) {
	requireCheckErrorContains(t, `
func seal(token: consent.token) -> secret.i32
uses privacy
privacy:
  return core.secret_seal_i32(1, token)
`, "require semantic clause consent(<token>)")

	requireCheckErrorContains(t, `
func seal(token: Int) -> secret.i32
uses privacy
privacy
consent(token):
  return 0
`, "must have type consent.token")

	requireCheckOK(t, `
func seal(token: consent.token) -> secret.i32
uses privacy
privacy
consent(token):
  return core.secret_seal_i32(1, token)

func reveal(token: consent.token, value: secret.i32) -> Int
uses privacy
privacy
consent(token):
  return core.secret_unseal_i32(value, token)

func main() -> Int:
  return 0
`)
}

func TestPrivacyConsentRecursiveSecretSignatureChecks(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		wantErr string
	}{
		{
			name: "struct containing secret requires consent",
			src: `
struct SecretBox:
  value: secret.i32

func inspect(token: consent.token, box: SecretBox) -> Int
uses privacy
privacy:
  return 0

func main() -> Int:
  return 0
`,
			wantErr: "require semantic clause consent(<token>)",
		},
		{
			name: "enum payload containing secret requires consent",
			src: `
enum SecretResult:
  case sealed(secret.i32)
  case empty

func inspect(token: consent.token, value: SecretResult) -> Int
uses privacy
privacy:
  return 0

func main() -> Int:
  return 0
`,
			wantErr: "require semantic clause consent(<token>)",
		},
		{
			name: "optional secret container requires consent",
			src: `
func inspect(
  token: consent.token,
  maybeSecret: secret.i32?
) -> Int
uses privacy
privacy:
  return 0

func main() -> Int:
  return 0
`,
			wantErr: "require semantic clause consent(<token>)",
		},
		{
			name: "array secret container is currently unsupported",
			src: `
func inspect(
  token: consent.token,
  fixedSecrets: [2]secret.i32
) -> Int
uses privacy
privacy:
  return 0
`,
			wantErr: "array element type 'secret.i32' is not supported",
		},
		{
			name: "slice secret container is currently unsupported",
			src: `
func inspect(
  token: consent.token,
  manySecrets: []secret.i32
) -> Int
uses privacy
privacy:
  return 0
`,
			wantErr: "slice element type 'secret.i32' is not supported",
		},
		{
			name: "function-typed parameter with secret parameter requires consent",
			src: `
func inspect(
  token: consent.token,
  cb: fn(secret.i32) -> Int
) -> Int
uses privacy
privacy:
  return 0

func main() -> Int:
  return 0
`,
			wantErr: "require semantic clause consent(<token>)",
		},
		{
			name: "function-typed parameter with secret return requires consent",
			src: `
func inspect(
  token: consent.token,
  cb: fn() -> secret.i32
) -> Int
uses privacy
privacy:
  return 0

func main() -> Int:
  return 0
`,
			wantErr: "require semantic clause consent(<token>)",
		},
		{
			name: "function returning secret-bearing callable requires consent",
			src: `
func produce(token: consent.token) -> secret.i32
uses privacy
privacy
consent(token):
  return core.secret_seal_i32(42, token)

func make(token: consent.token) -> fn(consent.token) -> secret.i32 uses privacy:
  return produce

func main() -> Int:
  return 0
`,
			wantErr: "require semantic clause 'privacy'",
		},
		{
			name: "struct function-typed field with secret parameter requires consent",
			src: `
struct HandlerBox:
  cb: fn(secret.i32) -> Int

func inspect(token: consent.token, box: HandlerBox) -> Int
uses privacy
privacy:
  return 0

func main() -> Int:
  return 0
`,
			wantErr: "require semantic clause consent(<token>)",
		},
		{
			name: "enum function-typed payload with secret return requires consent",
			src: `
enum HandlerChoice:
  case some(fn() -> secret.i32)
  case empty

func inspect(token: consent.token, choice: HandlerChoice) -> Int
uses privacy
privacy:
  return 0

func main() -> Int:
  return 0
`,
			wantErr: "require semantic clause consent(<token>)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requireCheckErrorContains(t, tt.src, tt.wantErr)
		})
	}

	requireCheckOK(t, `
struct PlainBox:
  value: Int

enum PlainResult:
  case ok(Int)
  case empty

func inspect(
  box: PlainBox,
  value: PlainResult,
  maybeInt: Int?
) -> Int:
  return 0

func main() -> Int:
  let box: PlainBox = PlainBox(value: 1)
  let value: PlainResult = PlainResult.ok(2)
  let maybeInt: Int? = none
  return inspect(box, value, maybeInt)
`)
}

func TestPrivacySecretTaintBeyondFunctionSignatures(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		wantErr string
	}{
		{
			name: "exported function cannot return unsealed secret-tainted local",
			src: `
@export("leak_plain")
func leak(seed: Int) -> Int
uses privacy
privacy:
  let token: consent.token = core.consent_token()
  let value: secret.i32 = core.secret_seal_i32(seed, token)
  let raw: Int = core.secret_unseal_i32(value, token)
  return raw
`,
			wantErr: "secret-tainted value cannot be returned from @export function 'leak'",
		},
		{
			name: "secret-tainted field taints plain struct container",
			src: `
repr(C) struct PlainBox:
  value: Int

@export("leak_box")
func leak_box(seed: Int) -> PlainBox
uses privacy
privacy:
  let token: consent.token = core.consent_token()
  let value: secret.i32 = core.secret_seal_i32(seed, token)
  let raw: Int = core.secret_unseal_i32(value, token)
  let box: PlainBox = PlainBox(value: raw)
  return box
`,
			wantErr: "secret-tainted value cannot be returned from @export function 'leak_box'",
		},
		{
			name: "secret-tainted value cannot be stored in global",
			src: `
var leaked: Int = 0

func store(token: consent.token, value: secret.i32) -> Int
uses privacy
privacy
consent(token):
  let raw: Int = core.secret_unseal_i32(value, token)
  leaked = raw
  return 0
`,
			wantErr: "secret-tainted value cannot be stored in global 'leaked'",
		},
		{
			name: "secret-tainted helper return remains tainted at caller",
			src: `
func reveal(token: consent.token, value: secret.i32) -> Int
uses privacy
privacy
consent(token):
  return core.secret_unseal_i32(value, token)

@export("leak_via_helper")
func leak(seed: Int) -> Int
uses privacy
privacy:
  let token: consent.token = core.consent_token()
  let value: secret.i32 = core.secret_seal_i32(seed, token)
  return reveal(token, value)
`,
			wantErr: "secret-tainted value cannot be returned from @export function 'leak'",
		},
		{
			name: "secret-tainted value cannot be laundered via plain identity helper",
			src: `
func id(x: Int) -> Int:
  return x

@export("leak_via_id")
func leak(seed: Int) -> Int
uses privacy
privacy:
  let token: consent.token = core.consent_token()
  let value: secret.i32 = core.secret_seal_i32(seed, token)
  let raw: Int = core.secret_unseal_i32(value, token)
  return id(raw)
`,
			wantErr: "secret-tainted value cannot be returned from @export function 'leak'",
		},
		{
			name: "secret-tainted helper chain remains tainted",
			src: `
func id1(x: Int) -> Int:
  return x

func id2(x: Int) -> Int:
  return id1(x)

@export("leak_via_chain")
func leak(seed: Int) -> Int
uses privacy
privacy:
  let token: consent.token = core.consent_token()
  let value: secret.i32 = core.secret_seal_i32(seed, token)
  let raw: Int = core.secret_unseal_i32(value, token)
  return id2(raw)
`,
			wantErr: "secret-tainted value cannot be returned from @export function 'leak'",
		},
		{
			name: "exported function cannot throw unsealed secret-tainted enum payload",
			src: `
enum LeakErr:
  case raw(Int)

@export("leak_throw")
func leak(seed: Int) -> Int throws LeakErr
uses privacy
privacy:
  let token: consent.token = core.consent_token()
  let value: secret.i32 = core.secret_seal_i32(seed, token)
  let raw: Int = core.secret_unseal_i32(value, token)
  throw LeakErr.raw(raw)
`,
			wantErr: "secret-tainted value cannot be thrown from @export function 'leak'",
		},
		{
			name: "secret-tainted byte buffer cannot be printed",
			src: `
func leak(token: consent.token, value: secret.i32) -> Int
uses alloc, io, mem, privacy
privacy
consent(token):
  var bytes: []UInt8 = core.make_u8(2)
  let raw: Int = core.secret_unseal_i32(value, token)
  bytes[0] = raw
  bytes[1] = 10
  print(bytes)
  return 0
`,
			wantErr: "secret-tainted value cannot be printed",
		},
		{
			name: "secret-tainted if condition cannot select exported return",
			src: `
@export("leak_branch")
func leak(seed: Int) -> Int
uses privacy
privacy:
  let token: consent.token = core.consent_token()
  let value: secret.i32 = core.secret_seal_i32(seed, token)
  let raw: Int = core.secret_unseal_i32(value, token)
  if raw == 1:
    return 42
  else:
    return 7
`,
			wantErr: "secret-tainted value cannot be returned from @export function 'leak'",
		},
		{
			name: "secret-tainted if condition cannot select global assignment",
			src: `
var leaked: Int = 0

func store(token: consent.token, value: secret.i32) -> Int
uses privacy
privacy
consent(token):
  let raw: Int = core.secret_unseal_i32(value, token)
  if raw == 1:
    leaked = 42
  else:
    leaked = 7
  return 0
`,
			wantErr: "secret-tainted value cannot be stored in global 'leaked'",
		},
		{
			name: "secret-tainted match expression cannot select exported return",
			src: `
@export("leak_match_expr")
func leak(seed: Int) -> Int
uses privacy
privacy:
  let token: consent.token = core.consent_token()
  let value: secret.i32 = core.secret_seal_i32(seed, token)
  let raw: Int = core.secret_unseal_i32(value, token)
  let out: Int = match raw:
  case 1:
    42
  case _:
    7
  return out
`,
			wantErr: "secret-tainted value cannot be returned from @export function 'leak'",
		},
		{
			name: "secret-tainted while condition cannot select exported return",
			src: `
@export("leak_while")
func leak(seed: Int) -> Int
uses privacy
privacy:
  let token: consent.token = core.consent_token()
  let value: secret.i32 = core.secret_seal_i32(seed, token)
  let raw: Int = core.secret_unseal_i32(value, token)
  var count: Int = raw
  var out: Int = 7
  while count > 0:
    out = 42
    count = 0
  return out
`,
			wantErr: "secret-tainted value cannot be returned from @export function 'leak'",
		},
		{
			name: "secret-tainted value cannot be sent through actor mailbox",
			src: `
@export("leak_actor_mailbox")
func leak(seed: Int) -> Int
uses actors, privacy
privacy:
  let token: consent.token = core.consent_token()
  let value: secret.i32 = core.secret_seal_i32(seed, token)
  let raw: Int = core.secret_unseal_i32(value, token)
  let self_actor: actor = core.self()
  let _sent: Int = core.send(self_actor, raw)
  return core.recv()
`,
			wantErr: "secret-tainted value cannot be sent through actor mailbox",
		},
		{
			name: "secret-tainted enum payload cannot be sent through typed actor mailbox",
			src: `
enum LeakMsg:
  case raw(Int)
  case empty

@export("leak_typed_actor_mailbox")
func leak(seed: Int) -> Int
uses actors, privacy
privacy:
  let token: consent.token = core.consent_token()
  let value: secret.i32 = core.secret_seal_i32(seed, token)
  let raw: Int = core.secret_unseal_i32(value, token)
  let self_actor: actor = core.self()
  let _sent: Int = core.send_typed(self_actor, LeakMsg.raw(raw))
  let msg: LeakMsg = core.recv_typed<LeakMsg>()
  match msg:
  case LeakMsg.raw(v):
    return v
  case LeakMsg.empty:
    return 0
`,
			wantErr: "secret-tainted value cannot be sent through actor mailbox",
		},
		{
			name: "secret-tainted tagged payload cannot be sent through actor mailbox",
			src: `
@export("leak_tagged_actor_mailbox")
func leak(seed: Int) -> Int
uses actors, privacy
privacy:
  let token: consent.token = core.consent_token()
  let value: secret.i32 = core.secret_seal_i32(seed, token)
  let raw: Int = core.secret_unseal_i32(value, token)
  let self_actor: actor = core.self()
  let _sent: Int = core.send_msg(self_actor, raw, 99)
  let msg: actor.msg = core.recv_msg()
  return msg.value
`,
			wantErr: "secret-tainted value cannot be sent through actor mailbox",
		},
		{
			name: "secret-tainted value cannot be stored through raw memory",
			src: `
@export("leak_raw_memory")
func leak(seed: Int) -> Int
uses alloc, capability, mem, privacy
privacy:
  let token: consent.token = core.consent_token()
  let value: secret.i32 = core.secret_seal_i32(seed, token)
  let raw: Int = core.secret_unseal_i32(value, token)
  unsafe:
    let mem: cap.mem = core.cap_mem()
    let p: ptr = core.alloc_bytes(4)
    let _stored: Int = core.store_i32(p, raw, mem)
    return core.load_i32(p, mem)
`,
			wantErr: "secret-tainted value cannot be stored through raw memory",
		},
		{
			name: "secret-tainted sleep duration cannot affect runtime time",
			src: `
@export("leak_sleep_time")
func leak(seed: Int) -> Int
uses privacy, runtime
privacy:
  let token: consent.token = core.consent_token()
  let value: secret.i32 = core.secret_seal_i32(seed, token)
  let raw: Int = core.secret_unseal_i32(value, token)
  let _slept: Int = core.sleep_ms(raw)
  return core.time_now_ms()
`,
			wantErr: "secret-tainted value cannot control runtime time",
		},
		{
			name: "secret-tainted value cannot be written through mmio",
			src: `
@export("leak_mmio")
func leak(seed: Int) -> Int
uses alloc, capability, io, mem, mmio, privacy
privacy:
  let token: consent.token = core.consent_token()
  let value: secret.i32 = core.secret_seal_i32(seed, token)
  let raw: Int = core.secret_unseal_i32(value, token)
  unsafe:
    let io_cap: cap.io = core.cap_io()
    let p: ptr = core.alloc_bytes(4)
    let _stored: Int = core.mmio_write_i32(p, raw, io_cap)
    return core.mmio_read_i32(p, io_cap)
`,
			wantErr: "secret-tainted value cannot be written through MMIO",
		},
		{
			name: "secret-tainted closure capture cannot be returned from export",
			src: `
@export("leak_closure_capture")
func leak(seed: Int) -> Int
uses privacy
privacy:
  let token: consent.token = core.consent_token()
  let value: secret.i32 = core.secret_seal_i32(seed, token)
  let raw: Int = core.secret_unseal_i32(value, token)
  let f: fn() -> Int = fn() -> Int:
    return raw
  return f()
`,
			wantErr: "secret-tainted value cannot be returned from @export function 'leak'",
		},
		{
			name: "secret-tainted value cannot be laundered into struct via helper",
			src: `
repr(C) struct PlainBox:
  value: Int

func id(x: Int) -> Int:
  return x

@export("leak_box_via_id")
func leak(seed: Int) -> PlainBox
uses privacy
privacy:
  let token: consent.token = core.consent_token()
  let value: secret.i32 = core.secret_seal_i32(seed, token)
  let raw: Int = core.secret_unseal_i32(value, token)
  let box: PlainBox = PlainBox(value: id(raw))
  return box
`,
			wantErr: "secret-tainted value cannot be returned from @export function 'leak'",
		},
		{
			name: "secret-tainted argument cannot be laundered through helper side effects",
			src: `
var leaked: Int = 0

func stash(x: Int) -> Int:
  leaked = x
  return 0

@export("leak_via_side_effect")
func leak(seed: Int) -> Int
uses privacy
privacy:
  let token: consent.token = core.consent_token()
  let value: secret.i32 = core.secret_seal_i32(seed, token)
  let raw: Int = core.secret_unseal_i32(value, token)
  let dropped: Int = stash(raw)
  return 0
`,
			wantErr: "secret-tainted value cannot be stored in global 'leaked'",
		},
		{
			name: "secret-tainted standalone helper call cannot launder through side effects",
			src: `
var leaked: Int = 0

func stash(x: Int) -> Int:
  leaked = x
  return 0

@export("leak_via_side_effect_stmt")
func leak(seed: Int) -> Int
uses privacy
privacy:
  let token: consent.token = core.consent_token()
  let value: secret.i32 = core.secret_seal_i32(seed, token)
  let raw: Int = core.secret_unseal_i32(value, token)
  stash(raw)
  return 0
`,
			wantErr: "secret-tainted value cannot be stored in global 'leaked'",
		},
		{
			name: "secret-tainted callback flow through unknown target is rejected",
			src: `
var leaked: Int = 0

func stash(x: Int) -> Int:
  leaked = x
  return 0

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
  cb(x)
  return 0

@export("leak_via_callback")
func leak(seed: Int) -> Int
uses privacy
privacy:
  let token: consent.token = core.consent_token()
  let value: secret.i32 = core.secret_seal_i32(seed, token)
  let raw: Int = core.secret_unseal_i32(value, token)
  apply(stash, raw)
  return 0
`,
			wantErr: "secret-tainted value cannot be passed through unknown callback target 'cb'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requireFileSemanticErrorContains(t, tt.src, tt.wantErr)
		})
	}

	requireCheckOK(t, `
func reveal(token: consent.token, value: secret.i32) -> Int
uses privacy
privacy
consent(token):
  let raw: Int = core.secret_unseal_i32(value, token)
  return raw

func main() -> Int
uses privacy
privacy:
  let token: consent.token = core.consent_token()
  let secret: secret.i32 = core.secret_seal_i32(7, token)
  return reveal(token, secret)
`)
}
