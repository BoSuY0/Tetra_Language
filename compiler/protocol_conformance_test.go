package compiler

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestProtocolConformanceChecksExtensionMethod(t *testing.T) {
	src := []byte(`
struct Vec2:
    x: Int

protocol Renderable:
    func draw(self: Vec2) -> Int

extension Vec2:
    func draw(self: Vec2) -> Int:
        return self.x

impl Vec2: Renderable

func main() -> Int:
    return Vec2.draw(Vec2(x: 42))
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(prog.Impls) != 1 {
		t.Fatalf("impls = %d", len(prog.Impls))
	}
	if _, err := Check(prog); err != nil {
		t.Fatalf("Check: %v", err)
	}
}

func TestProtocolConformanceReportsMissingMethod(t *testing.T) {
	src := []byte(`
struct Vec2:
    x: Int

protocol Renderable:
    func draw(self: Vec2) -> Int

impl Vec2: Renderable

func main() -> Int:
    return 0
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected conformance error")
	}
	if !strings.Contains(err.Error(), "missing protocol requirement 'draw'") {
		t.Fatalf("error = %v", err)
	}
}

func TestProtocolConformanceViaImportedExtensionClause(t *testing.T) {
	files := map[string]string{
		"engine/core.tetra": `module engine.core
struct Vec2:
    x: Int

protocol Renderable:
    func draw(self: Vec2) -> Int
`,
		"app/ext.tetra": `module app.ext
import engine.core as core

extension core.Vec2:
    func draw(self: core.Vec2) -> Int:
        return self.x

impl core.Vec2: core.Renderable
`,
		"app/main.tetra": `module app.main
import app.ext as ext
import engine.core as core

func main() -> Int:
    let v: core.Vec2 = core.Vec2(x: 7)
    return core.Vec2.draw(v)
`,
	}
	tmp := t.TempDir()
	writeTestFiles(t, tmp, files)
	entry := filepath.Join(tmp, filepath.FromSlash("app/main.tetra"))

	world, err := LoadWorld(entry)
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	checked, err := CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	if _, ok := checked.FuncSigs["engine.core.Vec2.draw"]; !ok {
		t.Fatalf("missing imported extension method signature: %#v", checked.FuncSigs)
	}
	if _, err := LowerModules(checked); err != nil {
		t.Fatalf("LowerModules: %v", err)
	}
}

func TestProtocolConformanceRejectsDuplicateImplClause(t *testing.T) {
	src := []byte(`
struct Vec2:
    x: Int

protocol Renderable:
    func draw(self: Vec2) -> Int

extension Vec2:
    func draw(self: Vec2) -> Int:
        return self.x

impl Vec2: Renderable
impl Vec2: Renderable

func main() -> Int:
    return 0
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected duplicate impl clause error")
	}
	if !strings.Contains(err.Error(), "duplicate impl conformance") {
		t.Fatalf("error = %v", err)
	}
}

func TestProtocolConformanceReportsWrongSignature(t *testing.T) {
	src := []byte(`
struct Vec2:
    x: Int

protocol Renderable:
    func draw(self: Vec2) -> Int

extension Vec2:
    func draw(self: Vec2) -> Bool:
        return true

impl Vec2: Renderable

func main() -> Int:
    return 0
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected wrong signature conformance error")
	}
	if !strings.Contains(err.Error(), "return type differs") {
		t.Fatalf("error = %v", err)
	}
}
