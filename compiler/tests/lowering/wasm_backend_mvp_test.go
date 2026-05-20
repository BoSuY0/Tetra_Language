package compiler_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	compiler "tetra_language/compiler"
)

func TestWasmBuildOnlyTopLevelStringPropertySmoke(t *testing.T) {
	src := `val greeting: String = "hello wasm\n"

func main() -> Int
uses io:
    print(greeting)
    return 0
`
	buildWasmTargets(t, src, "top-level-string")
}

func TestWasmBuildOnlyTopLevelStringPropertyFieldSmoke(t *testing.T) {
	src := `val greeting: String = "hello wasm"
property title: String = "abc"

func main() -> Int:
    return greeting.len + title.len
`
	buildWasmTargets(t, src, "top-level-string-field")
}

func TestWasmBuildOnlyShortCircuitAndSmoke(t *testing.T) {
	src := `func main() -> Int:
    if true && !false:
        return 42
    return 0
`
	buildWasmTargets(t, src, "short-circuit-and")
}

func TestWasmBuildOnlyShortCircuitOrSmoke(t *testing.T) {
	src := `func main() -> Int:
    if false || true:
        return 42
    return 0
`
	buildWasmTargets(t, src, "short-circuit-or")
}

func TestWasmWebRunShortCircuitBoolSmoke(t *testing.T) {
	src := `func main() -> Int:
    if true && !false:
        return 42
    return 0
`
	runWasmWebMainWithNode(t, src, "short-circuit-bool", 42)
}

func TestWasmWebRunShortCircuitOrSmoke(t *testing.T) {
	src := `func main() -> Int:
    if false || true:
        return 7
    return 0
`
	runWasmWebMainWithNode(t, src, "short-circuit-or", 7)
}

func TestWasmWebRunShortCircuitSkipsRHS(t *testing.T) {
	src := `var ran: Int = 0

func rhs() -> Bool:
    ran = 1
    return true

func main() -> Int:
    if true || rhs():
        return 7 + ran
    return 0
`
	runWasmWebMainWithNode(t, src, "short-circuit-skips-rhs", 7)
}

func TestWasmWebRunGlobalStringInitialFieldAccessSmoke(t *testing.T) {
	src := `val greeting: String = "hello"

func main() -> Int:
    return greeting.len
`
	runWasmWebMainWithNode(t, src, "global-string-initial-field", 5)
}

func TestWasmWebRunGlobalStringFieldAccessAfterAssignmentSmoke(t *testing.T) {
	src := `var title: String = "hello"

func main() -> Int:
    title = "bye"
    return title.len
`
	runWasmWebMainWithNode(t, src, "global-string-field-after-assignment", 3)
}

func TestWasmWebRunGlobalStringsAndConstBranchSmoke(t *testing.T) {
	src := `val greeting: String = "hello"
var title: String = "hi"
const base: Int = 34

func main() -> Int:
    title = "world"
    if greeting.len + title.len + base + 3 == 47:
        return 0
    return 1
`
	runWasmWebMainWithNode(t, src, "global-strings-and-const-branch", 0)
}

func TestWasmBuildOnlyDirectNamedCallableParamSmoke(t *testing.T) {
	src := `func add1(x: Int) -> Int:
    return x + 1

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    return apply(add1, 41)
`
	buildWasmTargets(t, src, "direct-named-callable")
}

func TestWasmBuildOnlyCallableAliasSmoke(t *testing.T) {
	src := `func add1(x: Int) -> Int:
    return x + 1

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let f: fn(Int) -> Int = add1
    let g: fn(Int) -> Int = f
    return apply(g, 41)
`
	buildWasmTargets(t, src, "callable-alias")
}

func TestWasmBuildOnlyReturnedCallableValueSmoke(t *testing.T) {
	src := `func add1(x: Int) -> Int:
    return x + 1

func pick() -> fn(Int) -> Int:
    let f: fn(Int) -> Int = add1
    let g: fn(Int) -> Int = f
    return g

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let cb: fn(Int) -> Int = pick()
    return apply(cb, 41)
`
	buildWasmTargets(t, src, "returned-callable")
}

func TestWasmBuildOnlyMultiTargetCallableParamSmoke(t *testing.T) {
	src := `func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let a: Int = apply(add1, 10)
    let b: Int = apply(add2, 20)
    return a + b
`
	buildWasmTargets(t, src, "multi-target-callable")
}

func TestWasmBuildOnlyMultiTargetStringReturnCallableParamSmoke(t *testing.T) {
	src := `func word1(x: Int) -> String:
    return "cat"

func word2(x: Int) -> String:
    return "zebra"

func apply(cb: fn(Int) -> String, x: Int) -> String:
    return cb(x)

func main() -> Int:
    let a: String = apply(word1, 0)
    let b: String = apply(word2, 0)
    return a.len + b.len
`
	buildWasmTargets(t, src, "multi-target-string-callable")
}

func TestWasmBuildOnlyMultiTargetStructReturnCallableParamSmoke(t *testing.T) {
	src := `struct Pair:
    x: Int
    y: Int

func pair1(x: Int) -> Pair:
    return Pair(x: x, y: 1)

func pair2(x: Int) -> Pair:
    return Pair(x: x, y: 2)

func apply(cb: fn(Int) -> Pair, x: Int) -> Pair:
    return cb(x)

func main() -> Int:
    let a: Pair = apply(pair1, 10)
    let b: Pair = apply(pair2, 20)
    return a.x + a.y + b.x + b.y
`
	buildWasmTargets(t, src, "multi-target-struct-callable")
}

func buildWasmTargets(t *testing.T, src string, stem string) {
	t.Helper()

	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "main.tetra")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	for _, target := range []string{"wasm32-wasi", "wasm32-web"} {
		t.Run(target, func(t *testing.T) {
			outPath := filepath.Join(tmp, stem+"-"+target+".wasm")
			if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, target, compiler.BuildOptions{Jobs: 1}); err != nil {
				t.Fatalf("build %s: %v", target, err)
			}
			raw, err := os.ReadFile(outPath)
			if err != nil {
				t.Fatalf("read %s output: %v", target, err)
			}
			if len(raw) < 8 || !bytes.Equal(raw[:4], []byte{0x00, 0x61, 0x73, 0x6d}) {
				t.Fatalf("%s output missing wasm header", target)
			}
			validateWasmWithNode(t, outPath)
		})
	}
}

func validateWasmWithNode(t *testing.T, wasmPath string) {
	t.Helper()
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not found; skipping WebAssembly.compile validation")
	}
	script := `const fs = require("fs");
WebAssembly.compile(fs.readFileSync(process.argv[1]))
  .then(() => {})
  .catch((err) => {
    console.error(err.message);
    process.exit(1);
  });`
	if out, err := exec.Command(node, "-e", script, wasmPath).CombinedOutput(); err != nil {
		t.Fatalf("WebAssembly.compile %s: %v\n%s", filepath.Base(wasmPath), err, out)
	}
}

func runWasmWebMainWithNode(t *testing.T, src string, stem string, want int) {
	t.Helper()
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not found; skipping wasm32-web execution smoke")
	}

	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "main.tetra")
	outPath := filepath.Join(tmp, stem+".wasm")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "wasm32-web", compiler.BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build wasm32-web: %v", err)
	}

	script := `const fs = require("fs");
(async () => {
  const bytes = fs.readFileSync(process.argv[1]);
  const imports = {
    tetra_web_v1: {
      console_log() {},
      panic(code) { throw new Error("tetra panic(" + (code | 0) + ")"); },
    },
  };
  const { instance } = await WebAssembly.instantiate(bytes, imports);
  const main = instance.exports.tetra_main;
  if (typeof main !== "function") {
    throw new Error("missing tetra_main export");
  }
  const got = main() | 0;
  const want = Number(process.argv[2]);
  if (got !== want) {
    throw new Error("tetra_main returned " + got + ", want " + want);
  }
})().catch((err) => {
  console.error(err.message);
  process.exit(1);
});`
	if out, err := exec.Command(node, "-e", script, outPath, fmt.Sprint(want)).CombinedOutput(); err != nil {
		t.Fatalf("wasm32-web tetra_main: %v\n%s", err, out)
	}
}
