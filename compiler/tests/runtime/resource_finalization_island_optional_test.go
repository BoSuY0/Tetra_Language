package compiler_test

import (
	"testing"

	"tetra_language/compiler/internal/testkit"
)

func TestIslandFinalizationRejectsWholeOptionalUseAfterPayloadFree(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func use(value: island?) -> Int:
    return 0

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let maybe: island? = core.island_new(16)
        if let handle = maybe:
            free(handle)
        return use(maybe)
    }
`, "ambiguous resource provenance for 'maybe.$elem' after control-flow merge")
}

func TestIslandFinalizationRejectsCrossModuleWholeOptionalUseAfterPayloadFree(t *testing.T) {
	files := map[string]string{
		"lib/resources.t4": `module lib.resources

pub func pass(maybe: island?) -> island?:
    return maybe
`,
		"app/main.t4": `module app.main
import lib.resources as resources

func use(value: island?) -> Int:
    return 0

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let maybe: island? = isl
        let returned: island? = resources.pass(maybe)
        match returned:
        case some(other):
            free(other)
            return use(returned)
        case none:
            return 0
    }
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use freed resource 'returned.$elem'")
}

func TestIslandFinalizationRejectsCrossModuleWholeOptionalIfLetUseAfterPayloadFree(t *testing.T) {
	files := map[string]string{
		"lib/resources.t4": `module lib.resources

pub func pass(maybe: island?) -> island?:
    return maybe
`,
		"app/main.t4": `module app.main
import lib.resources as resources

func use(value: island?) -> Int:
    return 0

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let maybe: island? = isl
        let returned: island? = resources.pass(maybe)
        if let other = returned:
            free(other)
        return use(returned)
    }
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use freed resource 'returned.$elem'")
}

func TestIslandFinalizationRejectsClosureCaptureBeforeFree(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let cb: fn(Int) -> Int = fn(x: Int) -> Int:
            let alias: island = isl
            return x
        free(isl)
    }
    return 0
`, "function-typed storage 'cb' captures unsupported local 'isl' of type 'island'")
}

func TestIslandFinalizationRejectsInterproceduralOptionalPayloadAliasDoubleFree(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func pass(maybe: island?) -> island?:
    return maybe

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let maybe: island? = isl
        let returned: island? = pass(maybe)
        if let other = returned:
            free(isl)
            free(other)
    }
    return 0
`, "cannot use freed resource 'other'")
}

func TestIslandFinalizationRejectsInterproceduralOptionalMatchPayloadAliasDoubleFree(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func pass(maybe: island?) -> island?:
    return maybe

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let maybe: island? = isl
        let returned: island? = pass(maybe)
        match returned:
        case some(other):
            free(isl)
            free(other)
        case none:
            return 0
    }
    return 0
`, "cannot use freed resource 'other'")
}

func TestIslandFinalizationRejectsCrossModuleOptionalPayloadAliasDoubleFree(t *testing.T) {
	files := map[string]string{
		"lib/resources.t4": `module lib.resources

func pass(maybe: island?) -> island?:
    return maybe
`,
		"app/main.t4": `module app.main
import lib.resources as resources

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let maybe: island? = isl
        let returned: island? = resources.pass(maybe)
        if let other = returned:
            free(isl)
            free(other)
    }
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use freed resource 'other'")
}

func TestIslandFinalizationRejectsCrossModuleOptionalMatchPayloadAliasDoubleFree(t *testing.T) {
	files := map[string]string{
		"lib/resources.t4": `module lib.resources

func pass(maybe: island?) -> island?:
    return maybe
`,
		"app/main.t4": `module app.main
import lib.resources as resources

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let maybe: island? = isl
        let returned: island? = resources.pass(maybe)
        match returned:
        case some(other):
            free(isl)
            free(other)
        case none:
            return 0
    }
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use freed resource 'other'")
}

func TestIslandFinalizationRejectsInterproceduralOptionalWrappedReturnAliasDoubleFree(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func wrap(isl: island) -> island?:
    return isl

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let returned: island? = wrap(isl)
        if let other = returned:
            free(isl)
            free(other)
    }
    return 0
`, "cannot use freed resource 'other'")
}

func TestIslandFinalizationRejectsCrossModuleOptionalWrappedReturnAliasDoubleFree(t *testing.T) {
	files := map[string]string{
		"lib/resources.t4": `module lib.resources

func wrap(isl: island) -> island?:
    return isl
`,
		"app/main.t4": `module app.main
import lib.resources as resources

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let returned: island? = resources.wrap(isl)
        if let other = returned:
            free(isl)
            free(other)
    }
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use freed resource 'other'")
}

func TestIslandFinalizationRejectsInterproceduralStructOptionalPayloadAliasDoubleFree(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
struct MaybeBox:
    maybe: island?

func pass(box: MaybeBox) -> MaybeBox:
    return box

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let box: MaybeBox = MaybeBox(maybe: isl)
        let returned: MaybeBox = pass(box)
        if let other = returned.maybe:
            free(isl)
            free(other)
    }
    return 0
`, "cannot use freed resource 'other'")
}

func TestIslandFinalizationRejectsCrossModuleStructOptionalPayloadAliasDoubleFree(t *testing.T) {
	files := map[string]string{
		"lib/resources.t4": `module lib.resources

pub struct MaybeBox:
    maybe: island?

pub func pass(box: MaybeBox) -> MaybeBox:
    return box
`,
		"app/main.t4": `module app.main
import lib.resources as resources

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let box: resources.MaybeBox = resources.MaybeBox(maybe: isl)
        let returned: resources.MaybeBox = resources.pass(box)
        if let other = returned.maybe:
            free(isl)
            free(other)
    }
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use freed resource 'other'")
}

func TestIslandFinalizationRejectsInterproceduralEnumOptionalPayloadAliasDoubleFree(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
enum MaybeEnvelope:
    case wrap(island?)
    case empty

func pass(msg: MaybeEnvelope) -> MaybeEnvelope:
    return msg

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let msg: MaybeEnvelope = MaybeEnvelope.wrap(isl)
        let returned: MaybeEnvelope = pass(msg)
        match returned:
        case MaybeEnvelope.wrap(maybe):
            if let other = maybe:
                free(isl)
                free(other)
        case MaybeEnvelope.empty:
            return 0
    }
    return 0
`, "cannot use freed resource 'other'")
}

func TestIslandFinalizationRejectsCrossModuleEnumOptionalPayloadAliasDoubleFree(t *testing.T) {
	files := map[string]string{
		"lib/resources.t4": `module lib.resources

pub enum MaybeEnvelope:
    case wrap(island?)
    case empty

pub func pass(msg: MaybeEnvelope) -> MaybeEnvelope:
    return msg
`,
		"app/main.t4": `module app.main
import lib.resources as resources

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let msg: resources.MaybeEnvelope = resources.MaybeEnvelope.wrap(isl)
        let returned: resources.MaybeEnvelope = resources.pass(msg)
        match returned:
        case resources.MaybeEnvelope.wrap(maybe):
            if let other = maybe:
                free(isl)
                free(other)
        case resources.MaybeEnvelope.empty:
            return 0
    }
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "cannot use freed resource 'other'")
}
