package main

import (
	"path/filepath"
	"testing"
)

func TestCheckCommandJSONDiagnosticsForRepresentationMetadataAssignmentCode(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
	writeCLIProjectFile(t, dir, "src/lib/model.t4", `module lib.model

pub struct BufferBox:
    bytes: []u8
`)
	srcPath := filepath.Join(dir, "src", "app", "main.t4")
	writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.model as model

func main() -> Int
uses alloc, mem:
    var box: model.BufferBox = model.BufferBox(bytes: make_u8(2))
    box.bytes.len = 9
    return 0
`)

	assertCLIJSONSemanticDiagnostic(t, srcPath, srcPath, "representation metadata field 'len' is not user-assignable in safe code")
}
