package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateAPIDocs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "api.tetra")
	src := `struct Vec2:
    x: Int
    y: Int

enum Color:
    case red

protocol Renderable:
    func draw(self: Vec2) -> Int

impl Vec2: Renderable

const answer: Int = 40 + 2

func add(v: borrow Vec2) -> Int
uses mem, io:
    return v.x + v.y

test "math":
    expect 40 + 2 == 42
`
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	docs, err := GenerateAPIDocs([]string{path})
	if err != nil {
		t.Fatalf("GenerateAPIDocs: %v", err)
	}
	out := string(docs)
	for _, want := range []string{
		"# Tetra API Docs",
		"`Vec2`",
		"`Color`",
		"`const answer: Int`",
		"`protocol Renderable`",
		"`impl Vec2: Renderable`",
		"`func add(v: borrow Vec2) -> Int uses io, mem`",
		"`math`",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("docs missing %q:\n%s", want, out)
		}
	}
}
