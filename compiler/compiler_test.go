package compiler

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestBuildHello(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 {\n  print(\"Hello from Tetra!\\n\");\n  return 0;\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "Hello from Tetra!\n" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildTwoPrints(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 {\n  print(\"A\");\n  print(\"B\\n\");\n  return 0;\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "AB\n" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildStrLiteralValue(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fun main(): i32 {\n  val s: str = \"A\\n\"\n  print(s)\n  return 0\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "A\n" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildStrParam(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fun echo(x: str): i32 {\n  print(x)\n  return 0\n}\nfun main(): i32 {\n  return echo(\"Hi\\n\")\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "Hi\n" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildStrReturn(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fun greet(): str {\n  return \"Hey\\n\"\n}\nfun main(): i32 {\n  print(greet())\n  return 0\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "Hey\n" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildMakeI32Slice(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fun main(): i32 {\n  var xs: []i32 = make_i32(3)\n  xs[0] = 10\n  xs[1] = 20\n  xs[2] = xs[0] + xs[1]\n  return xs[2]\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 30 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildMakeU8Print(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fun main(): i32 {\n  var xs: []u8 = make_u8(2)\n  xs[0] = 65\n  xs[1] = 66\n  print(xs)\n  return 0\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "AB" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildMmioSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fun main(): i32 {\n  var out: i32 = 0\n  unsafe {\n    let io: cap.io = core.cap_io()\n    let p: ptr = core.alloc_bytes(4)\n    let _w: i32 = core.mmio_write_i32(p, 123, io)\n    out = core.mmio_read_i32(p, io)\n  }\n  return out\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 123 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildIslandsDebugDoubleFree(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fun main(): i32 {\n  unsafe {\n    let isl: island = core.island_new(64)\n    free(isl)\n    free(isl)\n  }\n  return 0\n}\n"
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{Jobs: 1, IslandsDebug: true})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 2 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildSliceBoundsCheck(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fun main(): i32 {\n  var xs: []i32 = make_i32(2)\n  xs[2] = 1\n  return 0\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 1 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildNonZeroReturn(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 {\n  print(\"Done\\n\");\n  return 7;\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "Done\n" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 7 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildLetExpr(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 {\n  let x: i32 = 2 + 3;\n  return x;\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 5 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildIfElseReturn(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 {\n  if (0) { return 1; } else { return 2; }\n  return 3;\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 2 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildWhileCounter(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 {\n  let n: i32 = 3;\n  let acc: i32 = 0;\n  while (n) {\n    acc = acc + 1;\n    n = n - 1;\n  }\n  return acc;\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 3 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildLessThan(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 {\n  return 2 < 3;\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 1 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildEqEqFalse(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 {\n  return 2 == 3;\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildWhileLess(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 {\n  let i: i32 = 0;\n  while (i < 3) {\n    i = i + 1;\n  }\n  return i;\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 3 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildNewStyleNoSemicolons(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fun main(): i32 {\n  var x: i32 = 2 + 3\n  return x\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 5 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildValAssignmentFails(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fun main(): i32 {\n  val x: i32 = 1\n  x = 2\n  return x\n}\n"
	if err := buildOnly(t, src); err == nil {
		t.Fatalf("expected compilation error")
	}
}

func TestBuildStructLiteralAndFieldAssign(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "struct Vec2 { x: i32, y: i32 }\nfun main(): i32 {\n  var v: Vec2 = Vec2{ x: 1, y: 2 }\n  v.x = 10\n  return v.x + v.y\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 12 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildStructParam(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "struct Vec2 { x: i32, y: i32 }\nfun sum(v: Vec2): i32 {\n  return v.x + v.y\n}\nfun main(): i32 {\n  return sum(Vec2{ x: 5, y: 7 })\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 12 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildStructCrossModule(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"engine/math.tetra": "module engine.math\nstruct Vec2 { x: i32, y: i32 }\nfun sum(v: Vec2): i32 {\n  return v.x + v.y\n}\n",
		"app/game.tetra":    "module app.game\nimport engine.math as m\nfun main(): i32 {\n  var v: m.Vec2 = m.Vec2{ x: 2, y: 3 }\n  return m.sum(v)\n}\n",
	}
	stdout, exitCode := buildAndRunFiles(t, files, "app/game.tetra")
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 5 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildStructValFieldAssignFails(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "struct Vec2 { x: i32, y: i32 }\nfun main(): i32 {\n  val v: Vec2 = Vec2{ x: 1, y: 2 }\n  v.x = 3\n  return v.x\n}\n"
	if err := buildOnly(t, src); err == nil {
		t.Fatalf("expected compilation error")
	}
}

func TestBuildFunctionCall(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fun add(a: i32, b: i32): i32 {\n  return a + b\n}\nfun main(): i32 {\n  return add(2, 3)\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 5 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildFunctionCallSevenArgs(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fun sum7(a: i32, b: i32, c: i32, d: i32, e: i32, f: i32, g: i32): i32 {\n  return a + b + c + d + e + f + g\n}\nfun main(): i32 {\n  return sum7(1, 2, 3, 4, 5, 6, 7)\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 28 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildFunctionCallEightArgs(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fun sum8(a: i32, b: i32, c: i32, d: i32, e: i32, f: i32, g: i32, h: i32): i32 {\n  return a + b + c + d + e + f + g + h\n}\nfun main(): i32 {\n  return sum8(1, 2, 3, 4, 5, 6, 7, 8)\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 36 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildFunctionCallNineArgs(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fun sum9(a: i32, b: i32, c: i32, d: i32, e: i32, f: i32, g: i32, h: i32, i: i32): i32 {\n  return a + b + c + d + e + f + g + h + i\n}\nfun main(): i32 {\n  return sum9(1, 2, 3, 4, 5, 6, 7, 8, 9)\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 45 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildFunctionCallPackNineArgs(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fun pack9(a: i32, b: i32, c: i32, d: i32, e: i32, f: i32, g: i32, h: i32, i: i32): i32 {\n  var ok: i32 = 1\n  if (a == 1) { ; } else { ok = 0 }\n  if (b == 2) { ; } else { ok = 0 }\n  if (c == 3) { ; } else { ok = 0 }\n  if (d == 4) { ; } else { ok = 0 }\n  if (e == 5) { ; } else { ok = 0 }\n  if (f == 6) { ; } else { ok = 0 }\n  if (g == 7) { ; } else { ok = 0 }\n  if (h == 8) { ; } else { ok = 0 }\n  if (i == 9) { ; } else { ok = 0 }\n  return ok\n}\nfun main(): i32 {\n  return pack9(1, 2, 3, 4, 5, 6, 7, 8, 9)\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 1 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildFunctionCallEightArgsNonEmptyStack(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fun sum8(a: i32, b: i32, c: i32, d: i32, e: i32, f: i32, g: i32, h: i32): i32 {\n  return a + b + c + d + e + f + g + h\n}\nfun main(): i32 {\n  return 1 + sum8(1, 2, 3, 4, 5, 6, 7, 8)\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 37 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildFunctionCallNineArgsNonEmptyStack(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fun sum9(a: i32, b: i32, c: i32, d: i32, e: i32, f: i32, g: i32, h: i32, i: i32): i32 {\n  return a + b + c + d + e + f + g + h + i\n}\nfun main(): i32 {\n  return 1 + sum9(1, 2, 3, 4, 5, 6, 7, 8, 9)\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 46 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildFunctionCallSevenArgsNonEmptyStack(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fun sum7(a: i32, b: i32, c: i32, d: i32, e: i32, f: i32, g: i32): i32 {\n  return a + b + c + d + e + f + g\n}\nfun main(): i32 {\n  return 1 + sum7(1, 2, 3, 4, 5, 6, 7)\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 29 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildFunctionCallNestedArgs(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fun sum2(a: i32, b: i32): i32 {\n  return a + b\n}\nfun pack9(a: i32, b: i32, c: i32, d: i32, e: i32, f: i32, g: i32, h: i32, i: i32): i32 {\n  var ok: i32 = 1\n  if (a == 21) { ; } else { ok = 0 }\n  if (b == 2) { ; } else { ok = 0 }\n  if (c == 3) { ; } else { ok = 0 }\n  if (d == 4) { ; } else { ok = 0 }\n  if (e == 5) { ; } else { ok = 0 }\n  if (f == 6) { ; } else { ok = 0 }\n  if (g == 7) { ; } else { ok = 0 }\n  if (h == 8) { ; } else { ok = 0 }\n  if (i == 9) { ; } else { ok = 0 }\n  return ok\n}\nfun main(): i32 {\n  return pack9(sum2(10, 11), 2, 3, 4, 5, 6, 7, 8, 9)\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 1 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildMultiFileCrossModuleCall(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"engine/render.tetra": "module engine.render\nfun add_one(x: i32): i32 {\n  return x + 1\n}\n",
		"app/game.tetra":      "module app.game\nimport engine.render as render\nfun main(): i32 {\n  val v: i32 = render.add_one(41)\n  return v == 42\n}\n",
	}
	stdout, exitCode := buildAndRunFiles(t, files, "app/game.tetra")
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 1 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildMultiFileAliasCall(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"engine/render.tetra": "module engine.render\nfun add_one(x: i32): i32 {\n  return x + 1\n}\n",
		"app/game.tetra":      "module app.game\nimport engine.render as r\nfun main(): i32 {\n  return r.add_one(41)\n}\n",
	}
	stdout, exitCode := buildAndRunFiles(t, files, "app/game.tetra")
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildMultiFileMissingModule(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"app/game.tetra": "module app.game\nimport engine.render as r\nfun main(): i32 {\n  return r.add_one(1)\n}\n",
	}
	if err := buildOnlyFiles(t, files, "app/game.tetra"); err == nil {
		t.Fatalf("expected compilation error")
	}
}

func TestBuildMultiFileImportCycle(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"app/game.tetra": "module app.game\nimport mod.a as a\nfun main(): i32 {\n  return a.ping()\n}\n",
		"mod/a.tetra":    "module mod.a\nimport mod.b as b\nfun ping(): i32 {\n  return b.pong()\n}\n",
		"mod/b.tetra":    "module mod.b\nimport mod.a as a\nfun pong(): i32 {\n  return 1\n}\n",
	}
	if err := buildOnlyFiles(t, files, "app/game.tetra"); err == nil {
		t.Fatalf("expected compilation error")
	}
}

func TestBuildMultiFileDuplicateModule(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"engine/render.tetra":       "module engine.render\nfun add_one(x: i32): i32 {\n  return x + 1\n}\n",
		"engine/render_alias.tetra": "module engine.render\nfun add_two(x: i32): i32 {\n  return x + 2\n}\n",
		"app/game.tetra":            "module app.game\nimport engine.render as r\nimport engine.render_alias as r2\nfun main(): i32 {\n  return r.add_one(1) + r2.add_two(1)\n}\n",
	}
	if err := buildOnlyFiles(t, files, "app/game.tetra"); err == nil {
		t.Fatalf("expected compilation error")
	}
}

func TestBuildValArgument(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fun add(a: i32, b: i32): i32 {\n  return a + b\n}\nfun main(): i32 {\n  val x: i32 = 4\n  return add(x, 1)\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 5 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildEmptyStatements(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 {\n  ;;;\n  let x: i32 = 2 + 3;\n  ;;;\n  return x;\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 5 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func buildAndRun(t *testing.T, src string) (string, int) {
	t.Helper()

	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "main.tetra")
	outPath := filepath.Join(tmp, "app")

	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := BuildFile(srcPath, outPath, "linux-x64"); err != nil {
		t.Fatalf("build: %v", err)
	}
	if err := verifyELF(outPath); err != nil {
		t.Fatalf("verify ELF: %v", err)
	}
	return runBinary(t, outPath)
}

func buildAndRunWithOptions(t *testing.T, src string, opt BuildOptions) (string, int) {
	t.Helper()

	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "main.tetra")
	outPath := filepath.Join(tmp, "app")

	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", opt); err != nil {
		t.Fatalf("build: %v", err)
	}
	if err := verifyELF(outPath); err != nil {
		t.Fatalf("verify ELF: %v", err)
	}
	return runBinary(t, outPath)
}

func buildOnly(t *testing.T, src string) error {
	t.Helper()

	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "main.tetra")
	outPath := filepath.Join(tmp, "app")

	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	return BuildFile(srcPath, outPath, "linux-x64")
}

func buildAndRunFiles(t *testing.T, files map[string]string, entry string) (string, int) {
	t.Helper()

	tmp := t.TempDir()
	writeTestFiles(t, tmp, files)

	entryPath := filepath.Join(tmp, filepath.FromSlash(entry))
	outPath := filepath.Join(tmp, "out", "app")
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if err := BuildFile(entryPath, outPath, "linux-x64"); err != nil {
		t.Fatalf("build: %v", err)
	}
	if err := verifyELF(outPath); err != nil {
		t.Fatalf("verify ELF: %v", err)
	}
	return runBinary(t, outPath)
}

func buildOnlyFiles(t *testing.T, files map[string]string, entry string) error {
	t.Helper()

	tmp := t.TempDir()
	writeTestFiles(t, tmp, files)

	entryPath := filepath.Join(tmp, filepath.FromSlash(entry))
	outPath := filepath.Join(tmp, "out", "app")
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	return BuildFile(entryPath, outPath, "linux-x64")
}

func writeTestFiles(t *testing.T, base string, files map[string]string) {
	t.Helper()

	for path, src := range files {
		full := filepath.Join(base, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(src), 0o644); err != nil {
			t.Fatalf("write source: %v", err)
		}
	}
}

func runBinary(t *testing.T, path string) (string, int) {
	t.Helper()

	cmd := exec.Command(path)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return out.String(), exitErr.ProcessState.ExitCode()
		}
		t.Fatalf("run binary: %v", err)
	}
	return out.String(), cmd.ProcessState.ExitCode()
}

func verifyELF(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	hdr := make([]byte, 64)
	if _, err := io.ReadFull(f, hdr); err != nil {
		return err
	}
	if !bytes.Equal(hdr[:4], []byte{0x7f, 'E', 'L', 'F'}) {
		return fmt.Errorf("missing ELF magic")
	}
	if hdr[4] != 2 {
		return fmt.Errorf("expected ELF64")
	}
	if hdr[5] != 1 {
		return fmt.Errorf("expected little-endian")
	}
	eType := binary.LittleEndian.Uint16(hdr[16:18])
	eMachine := binary.LittleEndian.Uint16(hdr[18:20])
	entry := binary.LittleEndian.Uint64(hdr[24:32])
	if eType != 2 {
		return fmt.Errorf("expected ET_EXEC")
	}
	if eMachine != 0x3e {
		return fmt.Errorf("expected x86_64 machine")
	}
	if entry == 0 {
		return fmt.Errorf("entrypoint is zero")
	}
	return nil
}
