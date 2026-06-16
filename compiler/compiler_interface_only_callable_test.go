package compiler

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildInterfaceOnlyModeFunctionTypedParameterReturnGlobalEscapeDiagnostic(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.identity

pub func identity(f: fn(Int) -> Int) -> fn(Int) -> Int:
    return f
`), "lib/identity.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.identity as id

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func store(f: fn(Int) -> Int) -> Int:
    cb = id.identity(f)
    return 0

func main() -> Int:
    let base: Int = 1
    return store(fn(x: Int) -> Int:
        return x + base
    )
`,
		"lib/identity.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only function-typed parameter-return global escape diagnostic")
	}
	want := "function-typed parameter 'f' cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestBuildInterfaceOnlyModeFunctionTypedParameterLocalAliasReturnGlobalEscapeDiagnostic(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.identity

pub func identity(f: fn(Int) -> Int) -> fn(Int) -> Int:
    let alias: fn(Int) -> Int = f
    return alias
`), "lib/identity.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.identity as id

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func store(f: fn(Int) -> Int) -> Int:
    cb = id.identity(f)
    return 0

func main() -> Int:
    let base: Int = 1
    return store(fn(x: Int) -> Int:
        return x + base
    )
`,
		"lib/identity.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only function-typed parameter local-alias return global escape diagnostic")
	}
	want := "function-typed parameter 'f' cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestBuildInterfaceOnlyModeFunctionTypedStructFieldReturnGlobalEscapeDiagnostic(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub struct Holder:
    cb: fn(Int) -> Int

pub func pick(holder: Holder) -> fn(Int) -> Int:
    return holder.cb
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	libIface, err := ParseFile(iface, "lib/callbacks.t4i")
	if err != nil {
		t.Fatalf("ParseFile interface: %v\ninterface:\n%s", err, iface)
	}
	checkedIface, err := CheckWorldOpt(&World{
		EntryModule:      "lib.callbacks",
		Files:            []*FileAST{libIface},
		InterfaceModules: map[string]bool{"lib.callbacks": true},
		ByModule: map[string]*FileAST{
			"lib.callbacks": libIface,
		},
	}, CheckOptions{RequireMain: false})
	if err != nil {
		t.Fatalf("CheckWorld interface: %v\ninterface:\n%s", err, iface)
	}
	pickSig := checkedIface.FuncSigs["lib.callbacks.pick"]
	if got := pickSig.ReturnFunctionParamName; got != "holder.cb" {
		t.Fatalf("pick ReturnFunctionParamName = %q, want holder.cb; interface:\n%s", got, iface)
	}
	if len(pickSig.ParamTypes) != 1 || pickSig.ParamTypes[0] != "lib.callbacks.Holder" {
		t.Fatalf("pick ParamTypes = %#v, want lib.callbacks.Holder; interface:\n%s", pickSig.ParamTypes, iface)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func store(f: fn(Int) -> Int) -> Int:
    let holder: callbacks.Holder = callbacks.Holder(cb: f)
    cb = callbacks.pick(holder)
    return 0

func main() -> Int:
    let base: Int = 1
    return store(fn(x: Int) -> Int:
        return x + base
    )
`,
		"lib/callbacks.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only function-typed struct-field-return global escape diagnostic")
	}
	want := "function-typed parameter 'f' cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestBuildInterfaceOnlyModeFunctionTypedNestedStructFieldReturnGlobalEscapeDiagnostic(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub struct Holder:
    cb: fn(Int) -> Int

pub struct Box:
    holder: Holder

pub func pick(box: Box) -> fn(Int) -> Int:
    return box.holder.cb
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func store(f: fn(Int) -> Int) -> Int:
    let box: callbacks.Box = callbacks.Box(holder: callbacks.Holder(cb: f))
    cb = callbacks.pick(box)
    return 0

func main() -> Int:
    let base: Int = 1
    return store(fn(x: Int) -> Int:
        return x + base
    )
`,
		"lib/callbacks.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only function-typed nested-struct-field-return global escape diagnostic")
	}
	want := "function-typed parameter 'f' cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestBuildInterfaceOnlyModeFunctionTypedStructParameterWholeReturnGlobalEscapeDiagnostic(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub struct Holder:
    cb: fn(Int) -> Int

pub struct Box:
    holder: Holder

pub func echo(box: Box) -> Box:
    return box
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func store(f: fn(Int) -> Int) -> Int:
    let box: callbacks.Box = callbacks.Box(holder: callbacks.Holder(cb: f))
    let returned: callbacks.Box = callbacks.echo(box)
    cb = returned.holder.cb
    return 0

func main() -> Int:
    let base: Int = 1
    return store(fn(x: Int) -> Int:
        return x + base
    )
`,
		"lib/callbacks.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only function-typed struct-parameter whole-return global escape diagnostic")
	}
	want := "function-typed parameter 'f' cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestBuildInterfaceOnlyModeFunctionTypedEnumParameterWholeReturnGlobalEscapeDiagnostic(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

pub func echo(choice: MaybeCallback) -> MaybeCallback:
    return choice
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func store(f: fn(Int) -> Int) -> Int:
    let choice: callbacks.MaybeCallback = callbacks.echo(callbacks.MaybeCallback.some(f))
    match choice:
    case callbacks.MaybeCallback.some(local):
        cb = local
        return 0
    case callbacks.MaybeCallback.empty:
        return 0

func main() -> Int:
    let base: Int = 1
    return store(fn(x: Int) -> Int:
        return x + base
    )
`,
		"lib/callbacks.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only function-typed enum-parameter whole-return global escape diagnostic")
	}
	want := "function-typed parameter 'f' cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestBuildInterfaceOnlyModeFunctionTypedEnumPayloadMatchReturnGlobalEscapeDiagnostic(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

pub func fallback(x: Int) -> Int:
    return x

pub func pick(choice: MaybeCallback) -> fn(Int) -> Int:
    match choice:
    case some(local):
        return local
    case empty:
        return fallback
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func store(f: fn(Int) -> Int) -> Int:
    cb = callbacks.pick(callbacks.MaybeCallback.some(f))
    return 0

func main() -> Int:
    let base: Int = 1
    return store(fn(x: Int) -> Int:
        return x + base
    )
`,
		"lib/callbacks.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only function-typed enum-payload match return global escape diagnostic\ninterface:\n%s", iface)
	}
	want := "function-typed parameter 'f' cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestBuildInterfaceOnlyModeReturnedAggregateClosurePayloadStub(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

pub struct Box:
    choice: MaybeCallback

pub func makeBox() -> Box:
    let base: Int = 1
    return Box(choice: MaybeCallback.some(fn(x: Int) -> Int:
        return x + base
    ))
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let box: callbacks.Box = callbacks.makeBox()
    match box.choice:
    case callbacks.MaybeCallback.some(local):
        cb = local
        return 0
    case callbacks.MaybeCallback.empty:
        return 0
`,
		"lib/callbacks.t4i": string(iface),
	})

	if _, err := BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	); err != nil {
		t.Fatalf("BuildFileWithStatsOpt interface-only returned aggregate closure stub: %v", err)
	}
}

func TestBuildInterfaceOnlyModeReturnedEnumClosurePayloadStub(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

pub func makeChoice() -> MaybeCallback:
    let base: Int = 1
    return MaybeCallback.some(fn(x: Int) -> Int:
        return x + base
    )
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let choice: callbacks.MaybeCallback = callbacks.makeChoice()
    match choice:
    case callbacks.MaybeCallback.some(local):
        cb = local
        return 0
    case callbacks.MaybeCallback.empty:
        return 0
`,
		"lib/callbacks.t4i": string(iface),
	})

	if _, err := BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	); err != nil {
		t.Fatalf("BuildFileWithStatsOpt interface-only returned enum closure stub: %v", err)
	}
}

func TestBuildInterfaceOnlyModeReturnedThrowingAggregateClosurePayloadStub(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub enum Boom:
    case bad

pub enum MaybeCallback:
    case some(fn(Int) -> Int throws Boom)
    case empty

pub struct Box:
    choice: MaybeCallback

pub func makeBox() -> Box:
    let base: Int = 1
    return Box(choice: MaybeCallback.some(fn(x: Int) -> Int throws Boom:
        return x + base
    ))
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func caller() -> Int throws callbacks.Boom:
    let box: callbacks.Box = callbacks.makeBox()
    match box.choice:
    case callbacks.MaybeCallback.some(local):
        return try local(41)
    case callbacks.MaybeCallback.empty:
        return 0

func main() -> Int:
    return catch caller():
    case callbacks.Boom.bad:
        0
`,
		"lib/callbacks.t4i": string(iface),
	})

	if _, err := BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	); err != nil {
		t.Fatalf("BuildFileWithStatsOpt interface-only returned throwing aggregate closure stub: %v", err)
	}
}

func TestBuildInterfaceOnlyModeReturnedThrowingEnumClosurePayloadStub(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub enum Boom:
    case bad

pub enum MaybeCallback:
    case some(fn(Int) -> Int throws Boom)
    case empty

pub func makeChoice() -> MaybeCallback:
    let base: Int = 1
    return MaybeCallback.some(fn(x: Int) -> Int throws Boom:
        return x + base
    )
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func caller() -> Int throws callbacks.Boom:
    let choice: callbacks.MaybeCallback = callbacks.makeChoice()
    match choice:
    case callbacks.MaybeCallback.some(local):
        return try local(41)
    case callbacks.MaybeCallback.empty:
        return 0

func main() -> Int:
    return catch caller():
    case callbacks.Boom.bad:
        0
`,
		"lib/callbacks.t4i": string(iface),
	})

	if _, err := BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	); err != nil {
		t.Fatalf("BuildFileWithStatsOpt interface-only returned throwing enum closure stub: %v", err)
	}
}

func TestBuildInterfaceOnlyModeReturnedThrowingAggregateClosurePayloadRequiresTryDiagnostic(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub enum Boom:
    case bad

pub enum MaybeCallback:
    case some(fn(Int) -> Int throws Boom)
    case empty

pub struct Box:
    choice: MaybeCallback

pub func makeBox() -> Box:
    let base: Int = 1
    return Box(choice: MaybeCallback.some(fn(x: Int) -> Int throws Boom:
        return x + base
    ))
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func main() -> Int:
    let box: callbacks.Box = callbacks.makeBox()
    match box.choice:
    case callbacks.MaybeCallback.some(local):
        return local(41)
    case callbacks.MaybeCallback.empty:
        return 0
`,
		"lib/callbacks.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only returned throwing aggregate closure payload requires-try diagnostic")
	}
	want := "call to throwing function 'local' requires try"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestBuildInterfaceOnlyModeReturnedThrowingEnumClosurePayloadRequiresTryDiagnostic(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub enum Boom:
    case bad

pub enum MaybeCallback:
    case some(fn(Int) -> Int throws Boom)
    case empty

pub func makeChoice() -> MaybeCallback:
    let base: Int = 1
    return MaybeCallback.some(fn(x: Int) -> Int throws Boom:
        return x + base
    )
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func main() -> Int:
    let choice: callbacks.MaybeCallback = callbacks.makeChoice()
    match choice:
    case callbacks.MaybeCallback.some(local):
        return local(41)
    case callbacks.MaybeCallback.empty:
        return 0
`,
		"lib/callbacks.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only returned throwing enum closure payload requires-try diagnostic")
	}
	want := "call to throwing function 'local' requires try"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestBuildInterfaceOnlyModeReturnedThrowingStructFieldClosureStub(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub enum Boom:
    case bad

pub struct Holder:
    cb: fn(Int) -> Int throws Boom

pub func makeHolder() -> Holder:
    let base: Int = 1
    return Holder(cb: fn(x: Int) -> Int throws Boom:
        return x + base
    )
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func caller() -> Int throws callbacks.Boom:
    let holder: callbacks.Holder = callbacks.makeHolder()
    return try holder.cb(41)

func main() -> Int:
    return catch caller():
    case callbacks.Boom.bad:
        0
`,
		"lib/callbacks.t4i": string(iface),
	})

	if _, err := BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	); err != nil {
		t.Fatalf("BuildFileWithStatsOpt interface-only returned throwing struct-field closure stub: %v", err)
	}
}

func TestBuildInterfaceOnlyModeReturnedThrowingStructFieldClosureRequiresTryDiagnostic(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub enum Boom:
    case bad

pub struct Holder:
    cb: fn(Int) -> Int throws Boom

pub func makeHolder() -> Holder:
    let base: Int = 1
    return Holder(cb: fn(x: Int) -> Int throws Boom:
        return x + base
    )
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func main() -> Int:
    let holder: callbacks.Holder = callbacks.makeHolder()
    return holder.cb(41)
`,
		"lib/callbacks.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only returned throwing struct-field closure requires-try diagnostic")
	}
	want := "call to throwing function 'holder.cb' requires try"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestBuildInterfaceOnlyModeReturnedThrowingStructFieldClosureCallbackStub(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub enum Boom:
    case bad

pub struct Holder:
    cb: fn(Int) -> Int throws Boom

pub func makeHolder() -> Holder:
    let base: Int = 1
    return Holder(cb: fn(x: Int) -> Int throws Boom:
        return x + base
    )
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func apply(f: fn(Int) -> Int throws callbacks.Boom, x: Int) -> Int throws callbacks.Boom:
    return try f(x)

func caller() -> Int throws callbacks.Boom:
    let holder: callbacks.Holder = callbacks.makeHolder()
    return try apply(holder.cb, 41)

func main() -> Int:
    return catch caller():
    case callbacks.Boom.bad:
        0
`,
		"lib/callbacks.t4i": string(iface),
	})

	if _, err := BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	); err != nil {
		t.Fatalf("BuildFileWithStatsOpt interface-only returned throwing struct-field closure callback stub: %v", err)
	}
}

func TestBuildInterfaceOnlyModeReturnedThrowingStructFieldClosureCallbackThrowsMismatchDiagnostic(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub enum Boom:
    case bad

pub struct Holder:
    cb: fn(Int) -> Int throws Boom

pub func makeHolder() -> Holder:
    let base: Int = 1
    return Holder(cb: fn(x: Int) -> Int throws Boom:
        return x + base
    )
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func apply(f: fn(Int) -> Int, x: Int) -> Int:
    return f(x)

func main() -> Int:
    let holder: callbacks.Holder = callbacks.makeHolder()
    return apply(holder.cb, 41)
`,
		"lib/callbacks.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only returned throwing struct-field closure callback throws mismatch diagnostic")
	}
	want := "callback function symbol 'holder.cb' throws type mismatch: expected '', got 'lib.callbacks.Boom'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestBuildInterfaceOnlyModeReturnedThrowingAggregateClosurePayloadCallbackStub(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub enum Boom:
    case bad

pub enum MaybeCallback:
    case some(fn(Int) -> Int throws Boom)
    case empty

pub struct Box:
    choice: MaybeCallback

pub func makeBox() -> Box:
    let base: Int = 1
    return Box(choice: MaybeCallback.some(fn(x: Int) -> Int throws Boom:
        return x + base
    ))
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func apply(f: fn(Int) -> Int throws callbacks.Boom, x: Int) -> Int throws callbacks.Boom:
    return try f(x)

func caller() -> Int throws callbacks.Boom:
    let box: callbacks.Box = callbacks.makeBox()
    match box.choice:
    case callbacks.MaybeCallback.some(local):
        return try apply(local, 41)
    case callbacks.MaybeCallback.empty:
        return 0

func main() -> Int:
    return catch caller():
    case callbacks.Boom.bad:
        0
`,
		"lib/callbacks.t4i": string(iface),
	})

	if _, err := BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	); err != nil {
		t.Fatalf("BuildFileWithStatsOpt interface-only returned throwing aggregate closure callback stub: %v", err)
	}
}

func TestBuildInterfaceOnlyModeReturnedThrowingEnumClosurePayloadCallbackStub(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub enum Boom:
    case bad

pub enum MaybeCallback:
    case some(fn(Int) -> Int throws Boom)
    case empty

pub func makeChoice() -> MaybeCallback:
    let base: Int = 1
    return MaybeCallback.some(fn(x: Int) -> Int throws Boom:
        return x + base
    )
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func apply(f: fn(Int) -> Int throws callbacks.Boom, x: Int) -> Int throws callbacks.Boom:
    return try f(x)

func caller() -> Int throws callbacks.Boom:
    let choice: callbacks.MaybeCallback = callbacks.makeChoice()
    match choice:
    case callbacks.MaybeCallback.some(local):
        return try apply(local, 41)
    case callbacks.MaybeCallback.empty:
        return 0

func main() -> Int:
    return catch caller():
    case callbacks.Boom.bad:
        0
`,
		"lib/callbacks.t4i": string(iface),
	})

	if _, err := BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	); err != nil {
		t.Fatalf("BuildFileWithStatsOpt interface-only returned throwing enum closure callback stub: %v", err)
	}
}

func TestBuildInterfaceOnlyModeReturnedThrowingEnumClosurePayloadCallbackThrowsMismatchDiagnostic(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub enum Boom:
    case bad

pub enum MaybeCallback:
    case some(fn(Int) -> Int throws Boom)
    case empty

pub func makeChoice() -> MaybeCallback:
    let base: Int = 1
    return MaybeCallback.some(fn(x: Int) -> Int throws Boom:
        return x + base
    )
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func apply(f: fn(Int) -> Int, x: Int) -> Int:
    return f(x)

func main() -> Int:
    let choice: callbacks.MaybeCallback = callbacks.makeChoice()
    match choice:
    case callbacks.MaybeCallback.some(local):
        return apply(local, 41)
    case callbacks.MaybeCallback.empty:
        return 0
`,
		"lib/callbacks.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only returned throwing enum closure callback throws mismatch diagnostic")
	}
	want := "callback function symbol 'local' throws type mismatch: expected '', got 'lib.callbacks.Boom'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestBuildInterfaceOnlyModeReturnedThrowingAggregateClosurePayloadCallbackThrowsMismatchDiagnostic(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub enum Boom:
    case bad

pub enum MaybeCallback:
    case some(fn(Int) -> Int throws Boom)
    case empty

pub struct Box:
    choice: MaybeCallback

pub func makeBox() -> Box:
    let base: Int = 1
    return Box(choice: MaybeCallback.some(fn(x: Int) -> Int throws Boom:
        return x + base
    ))
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func apply(f: fn(Int) -> Int, x: Int) -> Int:
    return f(x)

func main() -> Int:
    let box: callbacks.Box = callbacks.makeBox()
    match box.choice:
    case callbacks.MaybeCallback.some(local):
        return apply(local, 41)
    case callbacks.MaybeCallback.empty:
        return 0
`,
		"lib/callbacks.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only returned throwing aggregate closure callback throws mismatch diagnostic")
	}
	want := "callback function symbol 'local' throws type mismatch: expected '', got 'lib.callbacks.Boom'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}
