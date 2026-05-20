package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckCommandJSONDiagnosticsForBorrowedPtrOptionalGlobalAssignmentCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_lifetime_ptr_optional_global.tetra")
	src := `var leaked: ptr? = none

func leak(x: borrow ptr) -> Int:
    leaked = x
    return 0

func main() -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(t, srcPath, "borrowed local 'x' cannot escape via global assignment to 'leaked'")
}

func TestCheckCommandJSONDiagnosticsForBorrowedStringGlobalAssignmentCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_lifetime_string_global.tetra")
	src := `var leaked: str = ""

func leak(x: borrow str) -> Int:
    leaked = x
    return 0

func main() -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(t, srcPath, "borrowed local 'x' cannot escape via global assignment to 'leaked'")
}

func TestCheckCommandJSONDiagnosticsForCrossModuleBorrowedPtrOptionalGlobalAssignmentCode(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
	libPath := filepath.Join(dir, "src", "lib", "leaks.t4")
	writeCLIProjectFile(t, dir, "src/lib/leaks.t4", `module lib.leaks

var leaked: ptr? = none

pub func leak(x: borrow ptr) -> Int:
    leaked = x
    return 0
`)
	srcPath := filepath.Join(dir, "src", "app", "main.t4")
	writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`)
	assertCLIJSONLifetimeDiagnosticForPath(t, srcPath, libPath, "borrowed local 'x' cannot escape via global assignment to 'leaked'")
}

func TestCheckCommandJSONDiagnosticsForBorrowedPtrAggregateOptionalGlobalAssignmentCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_lifetime_ptr_aggregate_optional_global.tetra")
	src := `struct PtrBox:
    raw: ptr

var leaked: PtrBox? = none

func leak(box: borrow PtrBox) -> Int:
    leaked = box
    return 0

func main() -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(t, srcPath, "borrowed local 'box' cannot escape via global assignment to 'leaked'")
}

func TestCheckCommandJSONDiagnosticsForCrossModuleBorrowedPtrAggregateOptionalGlobalAssignmentCode(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
	writeCLIProjectFile(t, dir, "src/lib/model.t4", `module lib.model

pub struct PtrBox:
    raw: ptr
`)
	libPath := filepath.Join(dir, "src", "lib", "leaks.t4")
	writeCLIProjectFile(t, dir, "src/lib/leaks.t4", `module lib.leaks
import lib.model as model

var leaked: model.PtrBox? = none

pub func leak(box: borrow model.PtrBox) -> Int:
    leaked = box
    return 0
`)
	srcPath := filepath.Join(dir, "src", "app", "main.t4")
	writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`)
	assertCLIJSONLifetimeDiagnosticForPath(t, srcPath, libPath, "borrowed local 'box' cannot escape via global assignment to 'leaked'")
}

func TestCheckCommandJSONDiagnosticsForBorrowedSliceOptionalPayloadGlobalAssignmentCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_lifetime_slice_optional_payload_global.tetra")
	src := `var leaked: []u8? = none

func leak(maybe: borrow []u8?) -> Int:
    match maybe:
    case some(raw):
        leaked = raw
        return 0
    case none:
        return 0

func main() -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(t, srcPath, "borrowed local 'maybe' cannot escape via global assignment to 'leaked'")
}

func TestCheckCommandJSONDiagnosticsForBorrowedSliceGlobalAssignmentCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_lifetime_slice_global.tetra")
	src := `var leaked: []u8

func leak(x: borrow []u8) -> Int:
    leaked = x
    return 0

func main() -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(t, srcPath, "borrowed local 'x' cannot escape via global assignment to 'leaked'")
}

func TestCheckCommandJSONDiagnosticsForCrossModuleBorrowedSliceOptionalPayloadGlobalAssignmentCode(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
	libPath := filepath.Join(dir, "src", "lib", "leaks.t4")
	writeCLIProjectFile(t, dir, "src/lib/leaks.t4", `module lib.leaks

var leaked: []u8? = none

pub func leak(maybe: borrow []u8?) -> Int:
    match maybe:
    case some(raw):
        leaked = raw
        return 0
    case none:
        return 0
`)
	srcPath := filepath.Join(dir, "src", "app", "main.t4")
	writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`)
	assertCLIJSONLifetimeDiagnosticForPath(t, srcPath, libPath, "borrowed local 'maybe' cannot escape via global assignment to 'leaked'")
}

func TestCheckCommandJSONDiagnosticsForCrossModuleBorrowedSliceGlobalAssignmentCode(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
	libPath := filepath.Join(dir, "src", "lib", "leaks.t4")
	writeCLIProjectFile(t, dir, "src/lib/leaks.t4", `module lib.leaks

var leaked: []u8

pub func leak(x: borrow []u8) -> Int:
    leaked = x
    return 0
`)
	srcPath := filepath.Join(dir, "src", "app", "main.t4")
	writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.leaks as leaks

func main() -> Int:
    return 0
`)
	assertCLIJSONLifetimeDiagnosticForPath(t, srcPath, libPath, "borrowed local 'x' cannot escape via global assignment to 'leaked'")
}
