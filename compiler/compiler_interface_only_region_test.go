package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildInterfaceOnlyModeDoesNotRequireMain(t *testing.T) {
	tmp := t.TempDir()
	writeTestFiles(t, tmp, map[string]string{
		"math/core.t4": "module math.core\npub func add(a: Int, b: Int) -> Int:\n    return a + b\n",
	})

	outPath := filepath.Join(tmp, "out", "app")
	stats, err := BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("math/core.t4")),
		outPath,
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err != nil {
		t.Fatalf("BuildFileWithStatsOpt interface-only no main: %v", err)
	}
	if _, err := os.Stat(outPath); !os.IsNotExist(err) {
		t.Fatalf("interface-only build should not emit %s, stat err=%v", outPath, err)
	}
	if len(stats.InterfaceModules) != 0 {
		t.Fatalf("InterfaceModules = %#v, want none for source-only graph", stats.InterfaceModules)
	}
}

func TestBuildInterfaceOnlyModeAcceptsGeneratedT4IWithImportedSignatureType(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module math.core

import math.types as mt

pub func norm(v: mt.Vec) -> Int:
    return v.x
`), "math/core.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4":   "module app.main\nimport math.core as math\nfunc main() -> Int:\n    return 0\n",
		"math/core.t4i": string(iface),
		"math/types.t4": "module math.types\npub struct Vec:\n    x: Int\n",
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err != nil {
		t.Fatalf("BuildFileWithStatsOpt interface-only imported signature type: %v", err)
	}
}

func TestBuildInterfaceOnlyModeAcceptsGeneratedT4IWithStructReturnStub(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module math.core

pub struct Point:
    x: Int

pub func origin() -> Point:
    return Point(x: 0)
`), "math/core.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4":   "module app.main\nimport math.core as math\nfunc main() -> Int:\n    math.origin()\n    return 0\n",
		"math/core.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err != nil {
		t.Fatalf("BuildFileWithStatsOpt interface-only struct return stub: %v", err)
	}
}

func TestBuildInterfaceOnlyModeRejectsAggregateRegionReturnEscape(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.buffers

pub struct PairBuf:
    left: []u8
    right: []u8

pub func make_pair(a: island, b: island) -> PairBuf
uses alloc, islands, mem:
    return PairBuf(left: core.island_make_u8(a, 1), right: core.island_make_u8(b, 1))
`), "lib/buffers.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.buffers as buffers

func main() -> Int
uses alloc, islands, mem:
    var pair: buffers.PairBuf = buffers.PairBuf(left: make_u8(1), right: make_u8(1))
    island(64) as a:
        island(64) as b:
            pair = buffers.make_pair(a, b)
    return pair.left[0]
`,
		"lib/buffers.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only aggregate region return escape diagnostic")
	}
	if !strings.Contains(err.Error(), "slice from scoped island cannot escape") {
		t.Fatalf("error = %v, want scoped island escape diagnostic\ninterface:\n%s", err, iface)
	}
}

func TestBuildInterfaceOnlyModeRejectsOptionalAggregateRegionReturnEscape(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.buffers

pub struct PairBuf:
    left: []u8
    right: []u8

pub func maybe_pair(a: island, b: island) -> PairBuf?
uses alloc, islands, mem:
    var out: PairBuf? = none
    out = PairBuf(left: core.island_make_u8(a, 1), right: core.island_make_u8(b, 1))
    return out
`), "lib/buffers.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.buffers as buffers

func main() -> Int
uses alloc, islands, mem:
    var maybe: buffers.PairBuf? = none
    island(64) as a:
        island(64) as b:
            maybe = buffers.maybe_pair(a, b)
    match maybe:
    case some(pair):
        return pair.left[0]
    case none:
        return 0
`,
		"lib/buffers.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only optional aggregate region return escape diagnostic\ninterface:\n%s", iface)
	}
	if !strings.Contains(err.Error(), "slice from scoped island cannot escape") {
		t.Fatalf("error = %v, want scoped island escape diagnostic\ninterface:\n%s", err, iface)
	}
}

func TestBuildInterfaceOnlyModeRejectsEnumPayloadRegionReturnEscape(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.buffers

pub enum BufMsg:
    case both([]u8, []u8)
    case empty

pub func make_msg(a: island, b: island) -> BufMsg
uses alloc, islands, mem:
    return BufMsg.both(core.island_make_u8(a, 1), core.island_make_u8(b, 1))
`), "lib/buffers.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.buffers as buffers

func main() -> Int
uses alloc, islands, mem:
    var msg: buffers.BufMsg = buffers.BufMsg.empty
    island(64) as a:
        island(64) as b:
            msg = buffers.make_msg(a, b)
    match msg:
    case buffers.BufMsg.both(left, right):
        return left[0]
    case buffers.BufMsg.empty:
        return 0
`,
		"lib/buffers.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only enum payload region return escape diagnostic\ninterface:\n%s", iface)
	}
	if !strings.Contains(err.Error(), "slice from scoped island cannot escape") {
		t.Fatalf("error = %v, want scoped island escape diagnostic\ninterface:\n%s", err, iface)
	}
}

func TestBuildInterfaceOnlyModeRejectsOptionalEnumPayloadRegionReturnEscape(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.buffers

pub enum BufMsg:
    case both([]u8, []u8)
    case empty

pub func maybe_msg(a: island, b: island) -> BufMsg?
uses alloc, islands, mem:
    var out: BufMsg? = none
    out = BufMsg.both(core.island_make_u8(a, 1), core.island_make_u8(b, 1))
    return out
`), "lib/buffers.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.buffers as buffers

func main() -> Int
uses alloc, islands, mem:
    var maybe: buffers.BufMsg? = none
    island(64) as a:
        island(64) as b:
            maybe = buffers.maybe_msg(a, b)
    match maybe:
    case some(msg):
        match msg:
        case buffers.BufMsg.both(left, right):
            return left[0]
        case buffers.BufMsg.empty:
            return 0
    case none:
        return 0
`,
		"lib/buffers.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only optional enum payload region return escape diagnostic\ninterface:\n%s", iface)
	}
	if !strings.Contains(err.Error(), "slice from scoped island cannot escape") {
		t.Fatalf("error = %v, want scoped island escape diagnostic\ninterface:\n%s", err, iface)
	}
}

func TestBuildInterfaceOnlyModeRejectsBranchAggregateRegionReturnEscape(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.buffers

pub struct PairBuf:
    left: []u8
    right: []u8

pub func choose_pair(flag: Bool, a: island, b: island) -> PairBuf
uses alloc, islands, mem:
    if flag:
        return PairBuf(left: core.island_make_u8(a, 1), right: core.island_make_u8(b, 1))
    else:
        return PairBuf(left: core.island_make_u8(a, 1), right: core.island_make_u8(b, 1))
`), "lib/buffers.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.buffers as buffers

func main() -> Int
uses alloc, islands, mem:
    var pair: buffers.PairBuf = buffers.PairBuf(left: make_u8(1), right: make_u8(1))
    island(64) as a:
        island(64) as b:
            pair = buffers.choose_pair(true, a, b)
    return pair.left[0]
`,
		"lib/buffers.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only branch aggregate region return escape diagnostic\ninterface:\n%s", iface)
	}
	if !strings.Contains(err.Error(), "slice from scoped island cannot escape") {
		t.Fatalf("error = %v, want scoped island escape diagnostic\ninterface:\n%s", err, iface)
	}
}

func TestBuildInterfaceOnlyModeRejectsBranchOptionalAggregateRegionReturnEscape(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.buffers

pub struct PairBuf:
    left: []u8
    right: []u8

pub func choose_pair(flag: Bool, a: island, b: island) -> PairBuf?
uses alloc, islands, mem:
    var out: PairBuf? = none
    if flag:
        out = PairBuf(left: core.island_make_u8(a, 1), right: core.island_make_u8(b, 1))
    else:
        out = PairBuf(left: core.island_make_u8(a, 1), right: core.island_make_u8(b, 1))
    return out
`), "lib/buffers.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.buffers as buffers

func main() -> Int
uses alloc, islands, mem:
    var maybe: buffers.PairBuf? = none
    island(64) as a:
        island(64) as b:
            maybe = buffers.choose_pair(true, a, b)
    match maybe:
    case some(pair):
        return pair.left[0]
    case none:
        return 0
`,
		"lib/buffers.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only branch optional aggregate region return escape diagnostic\ninterface:\n%s", iface)
	}
	if !strings.Contains(err.Error(), "slice from scoped island cannot escape") {
		t.Fatalf("error = %v, want scoped island escape diagnostic\ninterface:\n%s", err, iface)
	}
}

func TestBuildInterfaceOnlyModeRejectsBranchOptionalMixedAggregateRegionReturnEscape(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.buffers

pub struct PairBuf:
    left: []u8
    right: []u8

pub func choose_pair(flag: Bool, a: island, b: island) -> PairBuf?
uses alloc, islands, mem:
    var out: PairBuf? = none
    if flag:
        out = PairBuf(left: make_u8(1), right: make_u8(1))
    else:
        out = PairBuf(left: core.island_make_u8(a, 1), right: core.island_make_u8(b, 1))
    return out
`), "lib/buffers.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.buffers as buffers

func main() -> Int
uses alloc, islands, mem:
    var maybe: buffers.PairBuf? = none
    island(64) as a:
        island(64) as b:
            maybe = buffers.choose_pair(false, a, b)
    match maybe:
    case some(pair):
        return pair.left[0]
    case none:
        return 0
`,
		"lib/buffers.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only branch optional mixed aggregate region return escape diagnostic\ninterface:\n%s", iface)
	}
	if !strings.Contains(err.Error(), "slice from scoped island cannot escape") {
		t.Fatalf("error = %v, want scoped island escape diagnostic\ninterface:\n%s", err, iface)
	}
}

func TestBuildInterfaceOnlyModeRejectsMatchAggregateRegionReturnEscape(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.buffers

pub enum Mode:
    case fast
    case slow

pub struct PairBuf:
    left: []u8
    right: []u8

pub func choose_pair(mode: Mode, a: island, b: island) -> PairBuf
uses alloc, islands, mem:
    match mode:
    case Mode.fast:
        return PairBuf(left: core.island_make_u8(a, 1), right: core.island_make_u8(b, 1))
    case Mode.slow:
        return PairBuf(left: core.island_make_u8(a, 1), right: core.island_make_u8(b, 1))
`), "lib/buffers.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.buffers as buffers

func main() -> Int
uses alloc, islands, mem:
    var pair: buffers.PairBuf = buffers.PairBuf(left: make_u8(1), right: make_u8(1))
    island(64) as a:
        island(64) as b:
            pair = buffers.choose_pair(buffers.Mode.fast, a, b)
    return pair.left[0]
`,
		"lib/buffers.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only match aggregate region return escape diagnostic\ninterface:\n%s", iface)
	}
	if !strings.Contains(err.Error(), "slice from scoped island cannot escape") {
		t.Fatalf("error = %v, want scoped island escape diagnostic\ninterface:\n%s", err, iface)
	}
}

func TestBuildInterfaceOnlyModeRejectsMatchOptionalAggregateRegionReturnEscape(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.buffers

pub enum Mode:
    case fast
    case slow

pub struct PairBuf:
    left: []u8
    right: []u8

pub func choose_pair(mode: Mode, a: island, b: island) -> PairBuf?
uses alloc, islands, mem:
    var out: PairBuf? = none
    match mode:
    case Mode.fast:
        out = PairBuf(left: core.island_make_u8(a, 1), right: core.island_make_u8(b, 1))
    case Mode.slow:
        out = PairBuf(left: core.island_make_u8(a, 1), right: core.island_make_u8(b, 1))
    return out
`), "lib/buffers.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.buffers as buffers

func main() -> Int
uses alloc, islands, mem:
    var maybe: buffers.PairBuf? = none
    island(64) as a:
        island(64) as b:
            maybe = buffers.choose_pair(buffers.Mode.fast, a, b)
    match maybe:
    case some(pair):
        return pair.left[0]
    case none:
        return 0
`,
		"lib/buffers.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only match optional aggregate region return escape diagnostic\ninterface:\n%s", iface)
	}
	if !strings.Contains(err.Error(), "slice from scoped island cannot escape") {
		t.Fatalf("error = %v, want scoped island escape diagnostic\ninterface:\n%s", err, iface)
	}
}

func TestBuildInterfaceOnlyModeRejectsIfLetOptionalAggregateRegionReturnEscape(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.buffers

pub struct PairBuf:
    left: []u8
    right: []u8

pub func choose_pair(flag: Bool?, a: island, b: island) -> PairBuf?
uses alloc, islands, mem:
    var out: PairBuf? = none
    if let enabled = flag:
        out = PairBuf(left: core.island_make_u8(a, 1), right: core.island_make_u8(b, 1))
    else:
        out = PairBuf(left: core.island_make_u8(a, 1), right: core.island_make_u8(b, 1))
    return out
`), "lib/buffers.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.buffers as buffers

func main() -> Int
uses alloc, islands, mem:
    var maybe: buffers.PairBuf? = none
    island(64) as a:
        island(64) as b:
            maybe = buffers.choose_pair(true, a, b)
    match maybe:
    case some(pair):
        return pair.left[0]
    case none:
        return 0
`,
		"lib/buffers.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only if-let optional aggregate region return escape diagnostic\ninterface:\n%s", iface)
	}
	if !strings.Contains(err.Error(), "slice from scoped island cannot escape") {
		t.Fatalf("error = %v, want scoped island escape diagnostic\ninterface:\n%s", err, iface)
	}
}

func TestBuildInterfaceOnlyModeRejectsIfLetMixedAggregateRegionReturnEscape(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.buffers

pub struct PairBuf:
    left: []u8
    right: []u8

pub func choose_pair(flag: Bool?, a: island, b: island) -> PairBuf
uses alloc, islands, mem:
    if let enabled = flag:
        return PairBuf(left: make_u8(1), right: make_u8(1))
    else:
        return PairBuf(left: core.island_make_u8(a, 1), right: core.island_make_u8(b, 1))
`), "lib/buffers.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.buffers as buffers

func main() -> Int
uses alloc, islands, mem:
    var pair: buffers.PairBuf = buffers.PairBuf(left: make_u8(1), right: make_u8(1))
    island(64) as a:
        island(64) as b:
            pair = buffers.choose_pair(none, a, b)
    return pair.left[0]
`,
		"lib/buffers.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only if-let mixed aggregate region return escape diagnostic\ninterface:\n%s", iface)
	}
	if !strings.Contains(err.Error(), "slice from scoped island cannot escape") {
		t.Fatalf("error = %v, want scoped island escape diagnostic\ninterface:\n%s", err, iface)
	}
}

func TestBuildInterfaceOnlyModeRejectsMatchMixedAggregateRegionReturnEscape(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.buffers

pub enum Mode:
    case fast
    case slow

pub struct PairBuf:
    left: []u8
    right: []u8

pub func choose_pair(mode: Mode, a: island, b: island) -> PairBuf
uses alloc, islands, mem:
    match mode:
    case Mode.fast:
        return PairBuf(left: make_u8(1), right: make_u8(1))
    case Mode.slow:
        return PairBuf(left: core.island_make_u8(a, 1), right: core.island_make_u8(b, 1))
`), "lib/buffers.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.buffers as buffers

func main() -> Int
uses alloc, islands, mem:
    var pair: buffers.PairBuf = buffers.PairBuf(left: make_u8(1), right: make_u8(1))
    island(64) as a:
        island(64) as b:
            pair = buffers.choose_pair(buffers.Mode.slow, a, b)
    return pair.left[0]
`,
		"lib/buffers.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only match mixed aggregate region return escape diagnostic\ninterface:\n%s", iface)
	}
	if !strings.Contains(err.Error(), "slice from scoped island cannot escape") {
		t.Fatalf("error = %v, want scoped island escape diagnostic\ninterface:\n%s", err, iface)
	}
}
