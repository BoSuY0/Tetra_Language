package compiler

import (
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

func TestEffectsRequireActorsUse(t *testing.T) {
	requireCheckErrorContains(t, `
func main() -> Int:
  let a: actor = core.spawn("main")
  return 0
`, "uses effect 'actors'")
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
uses budget, privacy:
  return 1

func main() -> Int:
  return audit()
`, "uses effect 'budget'")

	requireCheckOK(t, `
func audit() -> Int
uses budget, privacy:
  return 1

func main() -> Int
uses effects.policy:
  return audit()
`)
}
