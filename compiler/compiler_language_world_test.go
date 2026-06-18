package compiler

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

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

func TestBuildImportedGlobalOnlyModuleSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/constants.t4": `module lib.constants

pub val answer: Int = 42
`,
		"app/main.t4": `module app.main
import lib.constants as constants

func main() -> Int:
    return 42
`,
	}
	stdout, exitCode := buildAndRunFiles(t, files, "app/main.t4")
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
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
		"app/game.tetra":      "module app.game\nimport engine.render as render\nfun main(): i32 {\n  val v: i32 = render.add_one(41)\n  if (v == 42) { return 1 }\n  return 0\n}\n",
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

func TestLoadWorldRejectsDuplicateImportPath(t *testing.T) {
	tmp := t.TempDir()
	files := map[string]string{
		"engine/render.tetra": "module engine.render\nfun add_one(x: i32): i32 {\n  return x + 1\n}\n",
		"app/game.tetra":      "module app.game\nimport engine.render as render\nimport engine.render as r\nfun main(): i32 {\n  return render.add_one(41)\n}\n",
	}
	writeTestFiles(t, tmp, files)

	_, err := LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/game.tetra")))
	if err == nil {
		t.Fatalf("expected duplicate import path error")
	}
	if !strings.Contains(err.Error(), "duplicate import 'engine.render'") {
		t.Fatalf("error = %v, want duplicate import path diagnostic", err)
	}
}

func TestLoadWorldReportsMissingImportPath(t *testing.T) {
	tmp := t.TempDir()
	files := map[string]string{
		"app/game.tetra": "module app.game\nimport engine.missing as missing\nfun main(): i32 {\n  return missing.add_one(41)\n}\n",
	}
	writeTestFiles(t, tmp, files)

	_, err := LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/game.tetra")))
	if err == nil {
		t.Fatalf("expected missing import error")
	}
	if !strings.Contains(err.Error(), "load module 'engine.missing'") || !strings.Contains(err.Error(), "read source") {
		t.Fatalf("error = %v, want missing import path diagnostic", err)
	}
}

func TestLoadWorldReportsImportCycle(t *testing.T) {
	tmp := t.TempDir()
	files := map[string]string{
		"app/game.tetra": "module app.game\nimport mod.a as a\nfun main(): i32 {\n  return a.ping()\n}\n",
		"mod/a.tetra":    "module mod.a\nimport mod.b as b\nfun ping(): i32 {\n  return b.pong()\n}\n",
		"mod/b.tetra":    "module mod.b\nimport mod.a as a\nfun pong(): i32 {\n  return 1\n}\n",
	}
	writeTestFiles(t, tmp, files)

	_, err := LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/game.tetra")))
	if err == nil {
		t.Fatalf("expected import cycle error")
	}
	if !strings.Contains(err.Error(), "import cycle detected at 'mod.a'") {
		t.Fatalf("error = %v, want import cycle diagnostic", err)
	}
}

func TestLoadWorldReportsDuplicateModuleDeclaration(t *testing.T) {
	tmp := t.TempDir()
	files := map[string]string{
		"engine/render.tetra":       "module engine.render\nfun add_one(x: i32): i32 {\n  return x + 1\n}\n",
		"engine/render_alias.tetra": "module engine.render\nfun add_two(x: i32): i32 {\n  return x + 2\n}\n",
		"app/game.tetra":            "module app.game\nimport engine.render as r\nimport engine.render_alias as r2\nfun main(): i32 {\n  return r.add_one(1) + r2.add_two(1)\n}\n",
	}
	writeTestFiles(t, tmp, files)

	_, err := LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/game.tetra")))
	if err == nil {
		t.Fatalf("expected duplicate module error")
	}
	if !strings.Contains(err.Error(), "duplicate module 'engine.render'") {
		t.Fatalf("error = %v, want duplicate module diagnostic", err)
	}
}

func TestCheckWorldRejectsImportAliasShadowingTopLevelName(t *testing.T) {
	tmp := t.TempDir()
	files := map[string]string{
		"engine/math.tetra": "module engine.math\nfun inc(x: i32): i32 {\n  return x + 1\n}\n",
		"app/game.tetra":    "module app.game\nimport engine.math as math\nfun math(): i32 {\n  return 1\n}\nfun main(): i32 {\n  return math()\n}\n",
	}
	writeTestFiles(t, tmp, files)

	world, err := LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/game.tetra")))
	if err != nil {
		t.Fatalf("load world: %v", err)
	}
	_, err = CheckWorld(world)
	if err == nil {
		t.Fatalf("expected alias shadowing error")
	}
	if !strings.Contains(err.Error(), "import alias 'math' conflicts with declaration 'math'") {
		t.Fatalf("error = %v, want alias shadowing diagnostic", err)
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
