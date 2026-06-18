package compiler_test

import (
	"testing"

	"tetra_language/compiler/internal/testkit"
)

func TestIslandFinalizationRejectsDoubleFree(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        free(isl)
        free(isl)
    }
    return 0
`, "cannot use freed resource 'isl'")
}

func TestIslandFinalizationRejectsIslandMakeAfterFree(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        free(isl)
        let xs: []u8 = core.island_make_u8(isl, 1)
    }
    return 0
`, "cannot use freed resource 'isl'")
}

func TestIslandResetConsumesSourceToken(t *testing.T) {
	testkit.RequireFileSemanticCheckErrorContains(t, `
func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let next: island = core.island_reset(isl)
        free(isl)
        free(next)
    }
    return 0
`, "cannot use consumed value 'isl'")
}

func TestIslandResetRejectsAliasUseAfterSourceReset(t *testing.T) {
	testkit.RequireFileSemanticCheckErrorContains(t, `
func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let alias: island = isl
        let next: island = core.island_reset(isl)
        let xs: []u8 = core.island_make_u8(alias, 1)
        free(next)
        return xs[0]
    }
    return 0
`, "cannot use consumed value 'alias'")
}

func TestIslandResetInvalidatesPreviousSliceBorrow(t *testing.T) {
	testkit.RequireFileSemanticCheckErrorContains(t, `
func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let old: []u8 = core.island_make_u8(isl, 1)
        let next: island = core.island_reset(isl)
        free(next)
        return old[0]
    }
    return 0
`, "cannot reset island 'isl' while borrowed slice 'old' is alive")
}

func TestIslandResetRejectsWhilePreviousSliceBorrowAlive(t *testing.T) {
	testkit.RequireFileSemanticCheckErrorContains(t, `
func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let old: []u8 = core.island_make_u8(isl, 1)
        let next: island = core.island_reset(isl)
        free(next)
    }
    return 0
`, "cannot reset island 'isl' while borrowed slice 'old' is alive")
}

func TestIslandResetAllowsAfterPreviousSliceOwnerCleared(t *testing.T) {
	testkit.RequireFileSemanticCheckOK(t, `
func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        var old: []u8 = core.island_make_u8(isl, 1)
        old = make_u8(1)
        let next: island = core.island_reset(isl)
        let fresh: []u8 = core.island_make_u8(next, 1)
        let value: Int = old[0] + fresh[0]
        free(next)
        return value
    }
    return 0
`)
}

func TestIslandResetReturnedTokenCanAllocateAndFree(t *testing.T) {
	testkit.RequireFileSemanticCheckOK(t, `
func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let next: island = core.island_reset(isl)
        let fresh: []u8 = core.island_make_u8(next, 1)
        let value: Int = fresh[0]
        free(next)
        return value
    }
    return 0
`)
}

func TestIslandFinalizationReportsMaybeFreedAfterMerge(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        if 1:
            free(isl)
        free(isl)
    }
    return 0
`, "may have been freed after control-flow merge")
}

func TestIslandFinalizationReportsMaybeFreedAfterLoop(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        var i: Int = 0
        while i < 1:
            free(isl)
            i = i + 1
        free(isl)
    }
    return 0
`, "may have been freed after control-flow merge")
}

func TestIslandFinalizationReportsMaybeFreedAfterMatch(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
enum Choice:
    case freeit
    case keep

func choose() -> Choice:
    return Choice.freeit

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let choice: Choice = choose()
        match choice:
        case Choice.freeit:
            free(isl)
        case Choice.keep:
            let kept: Int = 0
        free(isl)
    }
    return 0
`, "may have been freed after control-flow merge")
}

func TestIslandFinalizationRejectsAliasDoubleFree(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let alias: island = isl
        free(isl)
        free(alias)
    }
    return 0
`, "cannot use freed resource 'alias'")
}

func TestIslandFinalizationRejectsStructFieldAliasDoubleFree(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
struct IslandBox:
    handle: island

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let box: IslandBox = IslandBox(handle: core.island_new(16))
        let alias: IslandBox = box
        free(box.handle)
        free(alias.handle)
    }
    return 0
`, "cannot use freed resource 'alias.handle'")
}

func TestIslandFinalizationRejectsGenericStructFieldAliasDoubleFree(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
struct Box<T>:
    value: T

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let box: Box<island> = Box<island>{value: core.island_new(16)}
        let alias: Box<island> = box
        free(box.value)
        free(alias.value)
    }
    return 0
`, "cannot use freed resource 'alias.value'")
}

func TestIslandFinalizationRejectsCrossModuleGenericStructFieldAliasDoubleFree(t *testing.T) {
	files := map[string]string{
		"lib/resources.t4": `module lib.resources

pub struct Box<T>:
    value: T
`,
		"app/main.t4": `module app.main
import lib.resources as resources

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let box: resources.Box<island> = resources.Box<island>{value: core.island_new(16)}
        let alias: resources.Box<island> = box
        free(box.value)
        free(alias.value)
    }
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"cannot use freed resource 'alias.value'",
	)
}

func TestIslandFinalizationRejectsStructFieldFreeThenOriginalFree(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
struct IslandBox:
    handle: island

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let box: IslandBox = IslandBox(handle: core.island_new(16))
        let alias: island = box.handle
        free(box.handle)
        free(alias)
    }
    return 0
`, "cannot use freed resource 'alias'")
}

func TestIslandTransferRejectsAggregateAliasFieldReuse(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
struct IslandBox:
    handle: island

struct OuterBox:
    inner: IslandBox

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let box: IslandBox = IslandBox(handle: core.island_new(16))
        let alias: IslandBox = box
        let _: OuterBox = OuterBox(inner: box)
        free(alias.handle)
    }
    return 0
`, "cannot use consumed value 'alias.handle'")
}

func TestIslandTransferRejectsFieldAccessAggregateAliasFieldReuse(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
struct IslandBox:
    handle: island

struct HolderBox:
    inner: IslandBox

struct OuterBox:
    inner: IslandBox

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let box: IslandBox = IslandBox(handle: core.island_new(16))
        let holder: HolderBox = HolderBox(inner: box)
        let alias: IslandBox = holder.inner
        let _: OuterBox = OuterBox(inner: holder.inner)
        free(alias.handle)
    }
    return 0
`, "cannot use consumed value 'alias.handle'")
}

func TestIslandFinalizationAllowsSingleStructFieldFree(t *testing.T) {
	testkit.RequireFileSemanticCheckOK(t, `
struct IslandBox:
    handle: island

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let box: IslandBox = IslandBox(handle: core.island_new(16))
        free(box.handle)
    }
    return 0
`)
}

func TestIslandFinalizationRejectsStructFieldMergeAmbiguity(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
struct IslandBox:
    handle: island

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let left: island = core.island_new(16)
        let right: island = core.island_new(16)
        var box: IslandBox = IslandBox(handle: left)
        if 1:
            box = IslandBox(handle: right)
        free(box.handle)
    }
    return 0
`, "ambiguous resource provenance for 'box.handle'")
}

func TestIslandFinalizationRejectsInterproceduralStructFieldAliasDoubleFree(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
struct IslandBox:
    handle: island

func unwrap(box: IslandBox) -> island:
    return box.handle

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let box: IslandBox = IslandBox(handle: core.island_new(16))
        let other: island = unwrap(box)
        free(box.handle)
        free(other)
    }
    return 0
`, "cannot use freed resource 'other'")
}

func TestIslandFinalizationRejectsInterproceduralStructFieldReturnAmbiguity(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
struct Pair:
    left: island
    right: island

func pick(pair: Pair, flag: Int) -> island:
    if flag:
        return pair.left
    return pair.right

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let pair: Pair = Pair(left: core.island_new(16), right: core.island_new(32))
        let picked: island = pick(pair, 1)
        free(picked)
    }
    return 0
`, "return mixes resource provenance")
}

func TestIslandFinalizationRejectsInterproceduralAliasDoubleFree(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func alias(isl: island) -> island:
    return isl

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let other: island = alias(isl)
        free(isl)
        free(other)
    }
    return 0
`, "cannot use freed resource 'other'")
}

func TestIslandFinalizationRejectsTransitiveInterproceduralAliasDoubleFree(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func alias_one(isl: island) -> island:
    return isl

func alias_two(isl: island) -> island:
    return alias_one(isl)

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let other: island = alias_two(isl)
        free(isl)
        free(other)
    }
    return 0
`, "cannot use freed resource 'other'")
}

func TestIslandFinalizationRejectsCrossModuleTransitiveInterproceduralAliasDoubleFree(
	t *testing.T,
) {
	files := map[string]string{
		"lib/resources.t4": `module lib.resources

pub func alias_one(isl: island) -> island:
    return isl

pub func alias_two(isl: island) -> island:
    return alias_one(isl)
`,
		"app/main.t4": `module app.main
import lib.resources as resources

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let other: island = resources.alias_two(isl)
        free(isl)
        free(other)
    }
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"cannot use freed resource 'other'",
	)
}

func TestIslandFinalizationRejectsBranchReturnedAliasDoubleFree(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func branch_alias(isl: island, flag: Int) -> island:
    if flag:
        return isl
    return isl

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let other: island = branch_alias(isl, 1)
        free(isl)
        free(other)
    }
    return 0
`, "cannot use freed resource 'other'")
}

func TestIslandFinalizationRejectsAmbiguousResourceReturn(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func choose_island(left: island, right: island, flag: Int) -> island:
    if flag:
        return left
    return right

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let left: island = core.island_new(16)
        let right: island = core.island_new(16)
        let picked: island = choose_island(left, right, 1)
        free(picked)
    }
    return 0
`, "return mixes resource provenance")
}

func TestIslandFinalizationRejectsMergedLocalAmbiguousResourceReturn(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func choose_island(left: island, right: island, flag: Int) -> island:
    var picked: island = left
    if flag:
        picked = right
    return picked

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let left: island = core.island_new(16)
        let right: island = core.island_new(16)
        let picked: island = choose_island(left, right, 1)
        free(picked)
    }
    return 0
`, "ambiguous resource provenance for 'picked'")
}

func TestActorConsumeRejectsMergedLocalAmbiguousResourceReturn(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func worker() -> Int:
    return 0

func choose_actor(left: actor, right: actor, flag: Int) -> actor:
    var picked: actor = left
    if flag:
        picked = right
    return picked

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let left: actor = core.spawn("worker")
    let right: actor = core.spawn("worker")
    let picked: actor = choose_actor(left, right, 1)
    let _: Int = take_actor(left)
    return core.send(picked, 1)
`, "ambiguous resource provenance for 'picked'")
}

func TestIslandFinalizationRejectsUninferredRecursiveResourceReturn(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func recursive_alias(isl: island) -> island:
    let other: island = recursive_alias(isl)
    return other

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let other: island = recursive_alias(isl)
        free(isl)
        free(other)
    }
    return 0
`, "ambiguous resource provenance for 'other'")
}

func TestIslandFinalizationRejectsEnumPayloadAliasDoubleFree(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
enum MoveMsg:
    case take(island)

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let msg: MoveMsg = MoveMsg.take(core.island_new(16))
        match msg:
        case MoveMsg.take(other):
            let alias: island = other
            free(other)
            free(alias)
    }
    return 0
`, "cannot use freed resource 'alias'")
}

func TestIslandFinalizationRejectsInterproceduralEnumPayloadAliasDoubleFree(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
enum MoveMsg:
    case take(island)

func unwrap(msg: MoveMsg) -> island:
    match msg:
    case MoveMsg.take(handle):
        return handle

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let msg: MoveMsg = MoveMsg.take(core.island_new(16))
        let other: island = unwrap(msg)
        match msg:
        case MoveMsg.take(handle):
            free(handle)
            free(other)
    }
    return 0
`, "cannot use freed resource 'other'")
}

func TestIslandFinalizationRejectsCrossModuleEnumPayloadAliasDoubleFree(t *testing.T) {
	files := map[string]string{
		"lib/resources.t4": `module lib.resources

enum MoveMsg:
    case take(island)

func unwrap(msg: MoveMsg) -> island:
    match msg:
    case MoveMsg.take(handle):
        return handle
`,
		"app/main.t4": `module app.main
import lib.resources as resources

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let msg: resources.MoveMsg = resources.MoveMsg.take(core.island_new(16))
        let other: island = resources.unwrap(msg)
        match msg:
        case resources.MoveMsg.take(handle):
            free(handle)
            free(other)
    }
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(
		t,
		files,
		"app/main.t4",
		"cannot use freed resource 'other'",
	)
}
