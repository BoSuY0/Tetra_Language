package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckCommandJSONDiagnosticsForOwnershipUseAfterConsumeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_ownership.tetra")
	src := `func take(x: consume Int) -> Int:
    return x

func main() -> Int:
    let a: Int = 1
    let b: Int = take(a)
    return a + b
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use consumed value 'a'")
}

func TestCheckCommandJSONDiagnosticsForOwnershipPartialStructConsumeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_partial_struct_consume.tetra")
	src := `struct Pair:
    left: Int
    right: Int

func take(value: consume Int) -> Int:
    return value

func use(pair: Pair) -> Int:
    return pair.left + pair.right

func main() -> Int:
    let pair: Pair = Pair(left: 1, right: 2)
    let moved: Int = take(pair.left)
    return use(pair) + moved
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use consumed value 'pair.left'")
}

func TestCheckCommandJSONDiagnosticsForOwnershipPartialStructCopyAfterConsumeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_partial_struct_copy_after_consume.tetra")
	src := `struct Pair:
    left: Int
    right: Int

func take(value: consume Int) -> Int:
    return value

func main() -> Int:
    let pair: Pair = Pair(left: 1, right: 2)
    let moved: Int = take(pair.left)
    let copy: Pair = pair
    return moved + copy.right
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use consumed value 'pair.left'")
}

func TestCheckCommandJSONDiagnosticsForOwnershipPartialEnumConsumeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_partial_enum_consume.tetra")
	src := `enum PairMsg:
    case both(Int, Int)
    case empty

func take(value: consume Int) -> Int:
    return value

func use(msg: PairMsg) -> Int:
    match msg:
    case PairMsg.both(left, right):
        return left + right
    case PairMsg.empty:
        return 0

func main() -> Int:
    let msg: PairMsg = PairMsg.both(1, 2)
    match msg:
    case PairMsg.both(left, right):
        let moved: Int = take(left)
        return use(msg) + moved
    case PairMsg.empty:
        return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use consumed value 'msg.$case0.payload0'")
}

func TestCheckCommandJSONDiagnosticsForOwnershipPartialEnumCopyAfterConsumeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_partial_enum_copy_after_consume.tetra")
	src := `enum PairMsg:
    case both(Int, Int)
    case empty

func take(value: consume Int) -> Int:
    return value

func main() -> Int:
    let msg: PairMsg = PairMsg.both(1, 2)
    match msg:
    case PairMsg.both(left, right):
        let moved: Int = take(left)
        let copy: PairMsg = msg
        return moved + right
    case PairMsg.empty:
        return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use consumed value 'msg.$case0.payload0'")
}

func TestCheckCommandJSONDiagnosticsForCrossModulePartialCopyAfterConsumeCodes(t *testing.T) {
	tests := []struct {
		name     string
		modelSrc string
		mainSrc  string
		wantText string
	}{
		{
			name: "struct",
			modelSrc: `module lib.model

pub struct Pair:
    left: Int
    right: Int
`,
			mainSrc: `module app.main
import lib.model as model

func take(value: consume Int) -> Int:
    return value

func main() -> Int:
    let pair: model.Pair = model.Pair(left: 1, right: 2)
    let moved: Int = take(pair.left)
    let copy: model.Pair = pair
    return moved + copy.right
`,
			wantText: "cannot use consumed value 'pair.left'",
		},
		{
			name: "enum",
			modelSrc: `module lib.model

pub enum PairMsg:
    case both(Int, Int)
    case empty
`,
			mainSrc: `module app.main
import lib.model as model

func take(value: consume Int) -> Int:
    return value

func main() -> Int:
    let msg: model.PairMsg = model.PairMsg.both(1, 2)
    match msg:
    case model.PairMsg.both(left, right):
        let moved: Int = take(left)
        let copy: model.PairMsg = msg
        return moved + right
    case model.PairMsg.empty:
        return 0
`,
			wantText: "cannot use consumed value 'msg.$case0.payload0'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
			writeCLIProjectFile(t, dir, "src/lib/model.t4", tt.modelSrc)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", tt.mainSrc)
			assertCLIJSONOwnershipDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForOwnershipPartialEnumConstructorAfterConsumeCodes(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantText string
	}{
		{
			name: "struct field",
			src: `struct Pair:
    left: Int
    right: Int

enum Wrap:
    case one(Pair)
    case empty

func take(value: consume Int) -> Int:
    return value

func main() -> Int:
    let pair: Pair = Pair(left: 1, right: 2)
    let moved: Int = take(pair.left)
    let wrapped: Wrap = Wrap.one(pair)
    return moved
`,
			wantText: "cannot use consumed value 'pair.left'",
		},
		{
			name: "enum payload",
			src: `enum PairMsg:
    case both(Int, Int)
    case empty

enum Wrap:
    case one(PairMsg)
    case empty

func take(value: consume Int) -> Int:
    return value

func main() -> Int:
    let msg: PairMsg = PairMsg.both(1, 2)
    match msg:
    case PairMsg.both(left, right):
        let moved: Int = take(left)
        let wrapped: Wrap = Wrap.one(msg)
        return moved + right
    case PairMsg.empty:
        return 0
`,
			wantText: "cannot use consumed value 'msg.$case0.payload0'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(dir, "bad_partial_enum_constructor_after_consume.tetra")
			if err := os.WriteFile(srcPath, []byte(tt.src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONOwnershipDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForCrossModulePartialEnumConstructorAfterConsumeCodes(t *testing.T) {
	tests := []struct {
		name     string
		modelSrc string
		mainSrc  string
		wantText string
	}{
		{
			name: "struct field",
			modelSrc: `module lib.model

pub struct Pair:
    left: Int
    right: Int

pub enum Wrap:
    case one(Pair)
    case empty
`,
			mainSrc: `module app.main
import lib.model as model

func take(value: consume Int) -> Int:
    return value

func main() -> Int:
    let pair: model.Pair = model.Pair(left: 1, right: 2)
    let moved: Int = take(pair.left)
    let wrapped: model.Wrap = model.Wrap.one(pair)
    return moved
`,
			wantText: "cannot use consumed value 'pair.left'",
		},
		{
			name: "enum payload",
			modelSrc: `module lib.model

pub enum PairMsg:
    case both(Int, Int)
    case empty

pub enum Wrap:
    case one(PairMsg)
    case empty
`,
			mainSrc: `module app.main
import lib.model as model

func take(value: consume Int) -> Int:
    return value

func main() -> Int:
    let msg: model.PairMsg = model.PairMsg.both(1, 2)
    match msg:
    case model.PairMsg.both(left, right):
        let moved: Int = take(left)
        let wrapped: model.Wrap = model.Wrap.one(msg)
        return moved + right
    case model.PairMsg.empty:
        return 0
`,
			wantText: "cannot use consumed value 'msg.$case0.payload0'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
			writeCLIProjectFile(t, dir, "src/lib/model.t4", tt.modelSrc)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", tt.mainSrc)
			assertCLIJSONOwnershipDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForOwnershipOptionalPayloadConsumeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_optional_payload_consume.tetra")
	src := `func take(raw: consume ptr) -> ptr:
    return raw

func use(value: ptr?) -> Int:
    return 0

func leak(maybe: ptr?) -> Int:
    match maybe:
    case some(raw):
        let moved: ptr = take(raw)
    case none:
        let untouched: Int = 0
    return use(maybe)

func main() -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use consumed value 'maybe.$elem'")
}
