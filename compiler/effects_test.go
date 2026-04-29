package compiler

import (
	"path/filepath"
	"strings"
	"testing"
)

func requireCheckErrorContains(t *testing.T, src string, want string) {
	t.Helper()
	err := checkProgram(src)
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected error containing %q, got: %v", want, err)
	}
}

func requireCheckOK(t *testing.T, src string) {
	t.Helper()
	if err := checkProgram(src); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}

func TestEffectsRequireUsesIOForPrint(t *testing.T) {
	requireCheckErrorContains(t, `
func main() -> Int:
  print("hi\n")
  return 0
`, "uses effect 'io'")
}

func TestEffectsAllowUsesIOForPrint(t *testing.T) {
	requireCheckOK(t, `
func main() -> Int
uses io:
  print("hi\n")
  return 0
`)
}

func TestEffectsAliasesAndUnsafeRemainSeparate(t *testing.T) {
	requireCheckOK(t, `
func main() -> Int
uses cap.mem, alloc, capability:
  unsafe:
    let mem: cap.mem = core.cap_mem()
    let p: ptr = core.alloc_bytes(4)
    let _: Int = core.store_i32(p, 7, mem)
    return core.load_i32(p, mem)
  return 0
`)

	requireCheckErrorContains(t, `
func main() -> Int
uses cap.mem, alloc, capability:
  let mem: cap.mem = core.cap_mem()
  return 0
`, "only allowed in unsafe blocks")
}

func TestEffectsRejectUnknownUse(t *testing.T) {
	requireCheckErrorContains(t, `
func main() -> Int
uses sparkle:
  return 0
`, "unknown effect 'sparkle'")
}

func TestEffectsPropagateFunctionCalls(t *testing.T) {
	requireCheckErrorContains(t, `
func say() -> Int
uses io:
  print("hi\n")
  return 0

func main() -> Int:
  return say()
`, "uses effect 'io'")

	requireCheckOK(t, `
func say() -> Int
uses io:
  print("hi\n")
  return 0

func main() -> Int
uses io:
  return say()
`)
}

func TestEffectsPropagateAcrossImportedWrapper(t *testing.T) {
	files := map[string]string{
		"lib/logger.tetra": `module lib.logger

func write() -> Int
uses io:
  print("wrapped\n")
  return 1
`,
		"app/main.tetra": `module app.main
import lib.logger as logger

func call_logger() -> Int:
  return logger.write()

func main() -> Int:
  return call_logger()
`,
	}
	tmp := t.TempDir()
	writeTestFiles(t, tmp, files)
	world, err := LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/main.tetra")))
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	_, err = CheckWorld(world)
	if err == nil {
		t.Fatalf("expected imported effect propagation error")
	}
	for _, want := range []string{"function 'app.main.call_logger'", "uses effect 'io'"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want substring %q", err, want)
		}
	}
}

func TestEffectsRequireActorsUse(t *testing.T) {
	requireCheckErrorContains(t, `
func main() -> Int:
  let a: actor = core.spawn("main")
  return 0
`, "uses effect 'actors'")
}

func TestEffectsRejectUICommandMissingUses(t *testing.T) {
	requireCheckErrorContains(t, `
state ConsoleState:
  var count: Int = 0

view ConsoleView(state: ConsoleState):
  command log:
    print("event\n")

func main() -> Int:
  return 0
`, "command 'log'")
	requireCheckErrorContains(t, `
state ConsoleState:
  var count: Int = 0

view ConsoleView(state: ConsoleState):
  command log:
    print("event\n")

func main() -> Int:
  return 0
`, "uses effect 'io'")
}

func TestEffectGroupsExpandUsesForMemory(t *testing.T) {
	requireCheckOK(t, `
func main() -> Int
uses effects.memory:
  var xs: []Int = make_i32(2)
  xs[0] = 1
  return xs[0]
`)
}

func TestEffectsPropagateThroughGenericsWithGroups(t *testing.T) {
	requireCheckErrorContains(t, `
func first<T>(x: T) -> Int
uses effects.memory:
  var xs: []Int = make_i32(1)
  return xs[0]

func main() -> Int:
  return first(7)
`, "uses effect 'alloc'")

	requireCheckOK(t, `
func first<T>(x: T) -> Int
uses effects.memory:
  var xs: []Int = make_i32(1)
  return xs[0]

func main() -> Int
uses effects.memory:
  return first(7)
`)
}

func TestEffectsPropagateThroughProtocolsInitialSubset(t *testing.T) {
	requireCheckErrorContains(t, `
struct Device:
  id: Int

protocol Reader:
  func read(self: Device) -> Int uses io

extension Device:
  func read(self: Device) -> Int:
    return self.id

impl Device: Reader

func main() -> Int:
  return 0
`, "missing required effects")

	requireCheckOK(t, `
struct Device:
  id: Int

protocol Reader:
  func read(self: Device) -> Int uses io

extension Device:
  func read(self: Device) -> Int
  uses effects.cap.io:
    return self.id

impl Device: Reader

func main() -> Int:
  return 0
`)
}

func TestCapabilityAttenuationRequiresCapsulePermission(t *testing.T) {
	requireCheckErrorContains(t, `
func main() -> Int
uses effects.cap.mem, effects.memory:
  unsafe:
    let mem: cap.mem = core.cap_mem()
    let p: ptr = core.alloc_bytes(4)
    let _: Int = core.store_i32(p, 7, mem)
    return core.load_i32(p, mem)
  return 0
`, "capsule permission 'capsule.mem'")

	requireCheckOK(t, `
func main() -> Int
uses capsule.mem, effects.cap.mem, effects.memory:
  unsafe:
    let mem: cap.mem = core.cap_mem()
    let p: ptr = core.alloc_bytes(4)
    let _: Int = core.store_i32(p, 7, mem)
    return core.load_i32(p, mem)
  return 0
`)
}

func TestUnsafeStillRequiredWithEffectGroups(t *testing.T) {
	requireCheckErrorContains(t, `
func main() -> Int
uses effects.memory:
  let p: ptr = core.alloc_bytes(4)
  return 0
`, "only allowed in unsafe blocks")
}

func TestBudgetPrivacyEffectsAndPolicyGroup(t *testing.T) {
	requireCheckErrorContains(t, `
func audit() -> Int
uses budget, privacy
privacy:
  return 1

func main() -> Int:
  return audit()
`, "uses effect 'budget'")

	requireCheckOK(t, `
func audit() -> Int
uses budget, privacy
budget(1)
privacy:
  return 1

func main() -> Int
uses effects.policy
budget(1)
privacy:
  return audit()
`)
}

func TestSemanticClauseBudgetRequiresBudgetEffect(t *testing.T) {
	requireCheckErrorContains(t, `
func main() -> Int
budget(1):
  return 0
`, "requires function 'main' to declare uses effect 'budget'")
}

func TestSemanticClauseNoallocNoblockRealtimeChecks(t *testing.T) {
	requireCheckErrorContains(t, `
func main() -> Int
uses alloc
noalloc:
  return 0
`, "noalloc")

	requireCheckErrorContains(t, `
func main() -> Int
uses io
noblock:
  return 0
`, "noblock")

	requireCheckErrorContains(t, `
func main() -> Int
uses runtime
realtime:
  return 0
`, "realtime")
}

func TestSemanticClauseRealtimeRequiresNoallocAndNoblock(t *testing.T) {
	requireCheckErrorContains(t, `
func main() -> Int
realtime
noblock:
  return 0
`, "requires semantic clause 'noalloc'")

	requireCheckErrorContains(t, `
func main() -> Int
realtime
noalloc:
  return 0
`, "requires semantic clause 'noblock'")
}

func TestSemanticClauseNoallocDirectClosureAndCallGraphChecks(t *testing.T) {
	requireCheckOK(t, `
func inc(x: Int) -> Int:
  return x + 1

func main() -> Int
noalloc:
  let f: fn(Int) -> Int = fn(x: Int) -> Int:
    return x + 1
  return f(inc(40))
`)

	requireCheckErrorContains(t, `
func allocer(x: Int) -> Int
uses alloc, mem:
  unsafe:
    let _: ptr = core.alloc_bytes(4)
  return x

func main() -> Int
noalloc:
  return allocer(1)
`, "semantic clause 'noalloc' forbids call")
}

func TestSemanticClauseNoblockDirectCallGraphChecks(t *testing.T) {
	requireCheckOK(t, `
func inc(x: Int) -> Int:
  return x + 1

func main() -> Int
noblock:
  return inc(41)
`)

	requireCheckErrorContains(t, `
func sleeper(x: Int) -> Int
uses runtime:
  let _: Int = core.sleep_ms(1)
  return x

func main() -> Int
noblock:
  return sleeper(1)
`, "semantic clause 'noblock' forbids call")
}

func TestSemanticClauseRealtimeDirectCallGraphChecks(t *testing.T) {
	requireCheckOK(t, `
func pure(x: Int) -> Int
noalloc
noblock:
  return x + 1

func main() -> Int
realtime
noalloc
noblock:
  return pure(41)
`)

	requireCheckErrorContains(t, `
func sleeper(x: Int) -> Int
uses runtime:
  let _: Int = core.sleep_ms(1)
  return x

func main() -> Int
realtime
noalloc
noblock:
  return sleeper(1)
`, "semantic clause 'realtime' forbids call")
}

func TestSemanticClauseCallbackChecksForNoallocNoblockRealtime(t *testing.T) {
	requireCheckErrorContains(t, `
func alloc_cb(x: Int) -> Int
uses alloc, mem:
  unsafe:
    let _: ptr = core.alloc_bytes(4)
  return x

func apply(cb: fn(Int) -> Int, x: Int) -> Int
noalloc:
  return cb(x)

func main() -> Int:
  return apply(alloc_cb, 41)
`, "function-typed callback 'cb' has unknown target and cannot be called under semantic clause 'noalloc'")

	requireCheckErrorContains(t, `
func sleep_cb(x: Int) -> Int
uses runtime:
  let _: Int = core.sleep_ms(1)
  return x

func apply(cb: fn(Int) -> Int, x: Int) -> Int
noblock:
  return cb(x)

func main() -> Int:
  return apply(sleep_cb, 41)
`, "function-typed callback 'cb' has unknown target and cannot be called under semantic clause 'noblock'")

	requireCheckErrorContains(t, `
func sleep_cb(x: Int) -> Int
uses runtime:
  let _: Int = core.sleep_ms(1)
  return x

func apply(cb: fn(Int) -> Int, x: Int) -> Int
realtime
noalloc
noblock:
  return cb(x)

func main() -> Int:
  return apply(sleep_cb, 41)
`, "function-typed callback 'cb' has unknown target and cannot be called under semantic clause 'realtime'")
}

func TestSemanticClauseCallbackWrapperBypassRegression(t *testing.T) {
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
  return apply(allocer, 41)
`, "semantic clause 'noalloc' forbids call")

	requireCheckErrorContains(t, `
func sleeper(x: Int) -> Int
uses runtime:
  let _: Int = core.sleep_ms(1)
  return x

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
  return cb(x)

func main() -> Int
noblock:
  return apply(sleeper, 41)
`, "semantic clause 'noblock' forbids call")

	requireCheckErrorContains(t, `
func sleeper(x: Int) -> Int
uses runtime:
  let _: Int = core.sleep_ms(1)
  return x

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
  return cb(x)

func main() -> Int
realtime
noalloc
noblock:
  return apply(sleeper, 41)
`, "semantic clause 'realtime' forbids call")
}

func TestCallbackWrapperRequiresTargetEffects(t *testing.T) {
	requireCheckErrorContains(t, `
func allocer(x: Int) -> Int
uses alloc, mem:
  unsafe:
    let _: ptr = core.alloc_bytes(4)
  return x

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
  return cb(x)

func main() -> Int:
  return apply(allocer, 41)
`, "callback function symbol 'allocer' requires effects alloc, mem but function type does not declare them")
}

func TestCallbackWrapperRequiresLocalSymbolBackedTargetEffects(t *testing.T) {
	requireCheckErrorContains(t, `
func allocer(x: Int) -> Int
uses alloc, mem:
  unsafe:
    let _: ptr = core.alloc_bytes(4)
  return x

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
  return cb(x)

func main() -> Int:
  let f: fn(Int) -> Int = allocer
  return apply(f, 41)
`, "function-typed local 'f' requires effects alloc, mem but function type does not declare them")
}

func TestCallbackWrapperRequiresImportedTargetEffects(t *testing.T) {
	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

func allocer(x: Int) -> Int
uses alloc, mem:
  unsafe:
    let _: ptr = core.alloc_bytes(4)
  return x

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
  return cb(x)
`,
		"app/main.t4": `module app.main
import lib.callbacks.{apply, allocer}

func main() -> Int:
  return apply(allocer, 41)
`,
	}
	tmp := t.TempDir()
	writeTestFiles(t, tmp, files)
	world, err := LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/main.t4")))
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	_, err = CheckWorld(world)
	if err == nil {
		t.Fatalf("expected imported callback target effect propagation error")
	}
	for _, want := range []string{"callback function symbol 'allocer' requires effects alloc, mem but function type does not declare them"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want substring %q", err, want)
		}
	}
}

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
	writeTestFiles(t, tmp, files)
	world, err := LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/main.t4")))
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	_, err = CheckWorld(world)
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
