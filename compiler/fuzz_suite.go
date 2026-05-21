package compiler

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	ctarget "tetra_language/compiler/target"
)

type FuzzCheck struct {
	Name  string
	Error string
}

func RunTargetFuzzChecks(targetName string) ([]FuzzCheck, error) {
	tgt, err := ctarget.Parse(targetName)
	if err != nil {
		return nil, err
	}
	if tgt.Arch != ctarget.ArchX86 && tgt.Arch != ctarget.ArchX64 {
		return nil, fmt.Errorf("fuzz/property suite for target %s requires an x86/x64 native target model", tgt.Triple)
	}
	prefix := targetFuzzPrefix(tgt)
	return runFuzzChecks([]struct {
		name string
		run  func() error
	}{
		{name: prefix + " layout fuzz", run: func() error { return checkTargetLayoutFuzz(tgt) }},
		{name: prefix + " object signature fuzz", run: func() error { return checkTargetObjectSignatureFuzz(tgt) }},
		{name: prefix + " target alias fuzz", run: checkTargetAliasFuzz},
	}), nil
}

func runFuzzChecks(cases []struct {
	name string
	run  func() error
}) []FuzzCheck {
	out := make([]FuzzCheck, 0, len(cases))
	for _, tc := range cases {
		check := FuzzCheck{Name: tc.name}
		if err := tc.run(); err != nil {
			check.Error = err.Error()
		}
		out = append(out, check)
	}
	return out
}

func checkTargetLayoutFuzz(tgt ctarget.Target) error {
	x86, err := ctarget.Parse("x86")
	if err != nil {
		return err
	}
	x32, err := ctarget.Parse("x32")
	if err != nil {
		return err
	}
	x64, err := ctarget.Parse("x64")
	if err != nil {
		return err
	}
	seed := int64(0x6432)
	if tgt.Arch == ctarget.ArchX86 {
		seed = 0x8632
	}
	if tgt.ABI == ctarget.ABIX32SysV {
		seed = 0x3232
	}
	rng := rand.New(rand.NewSource(seed))
	fieldTypes := []string{"u8", "u16", "i32", "ptr", "usize", "isize", "size_t", "c_long", "i64"}
	for i := 0; i < 200; i++ {
		fields := make([]ctarget.LayoutField, 0, 1+rng.Intn(8))
		for j, count := 0, 1+rng.Intn(8); j < count; j++ {
			fields = append(fields, ctarget.LayoutField{
				Name: fmt.Sprintf("f_%d_%d", i, j),
				Type: fieldTypes[rng.Intn(len(fieldTypes))],
			})
		}
		got, err := tgt.StructLayout(fields)
		if err != nil {
			return fmt.Errorf("layout fuzz case %d: %w", i, err)
		}
		want, err := referenceFuzzStructLayout(tgt, fields, false)
		if err != nil {
			return fmt.Errorf("reference layout fuzz case %d: %w", i, err)
		}
		if err := compareFuzzAggregateLayout(fmt.Sprintf("%s struct fuzz case %d", targetFuzzPrefix(tgt), i), got, want); err != nil {
			return err
		}
		packedGot, err := tgt.PackedStructLayout(fields)
		if err != nil {
			return fmt.Errorf("packed layout fuzz case %d: %w", i, err)
		}
		packedWant, err := referenceFuzzStructLayout(tgt, fields, true)
		if err != nil {
			return fmt.Errorf("reference packed layout fuzz case %d: %w", i, err)
		}
		if err := compareFuzzAggregateLayout(fmt.Sprintf("%s packed struct fuzz case %d", targetFuzzPrefix(tgt), i), packedGot, packedWant); err != nil {
			return err
		}
		arrayType := fieldTypes[rng.Intn(len(fieldTypes))]
		count := rng.Intn(32)
		arr, err := tgt.ArrayLayout(arrayType, count)
		if err != nil {
			return fmt.Errorf("array layout fuzz case %d: %w", i, err)
		}
		scalar, ok := tgt.ScalarLayout(arrayType)
		if !ok {
			return fmt.Errorf("array layout fuzz case %d missing scalar %s", i, arrayType)
		}
		wantArraySize := fuzzAlignUp(scalar.SizeBytes, scalar.AlignBytes) * count
		if arr.SizeBytes != wantArraySize || arr.AlignBytes != scalar.AlignBytes {
			return fmt.Errorf("%s array fuzz case %d = %#v, want size=%d align=%d", targetFuzzPrefix(tgt), i, arr, wantArraySize, scalar.AlignBytes)
		}
	}
	pointerFields := []ctarget.LayoutField{{Name: "p", Type: "ptr"}, {Name: "n", Type: "usize"}}
	x86Struct, err := x86.StructLayout(pointerFields)
	if err != nil {
		return err
	}
	x32Struct, err := x32.StructLayout(pointerFields)
	if err != nil {
		return err
	}
	x64Struct, err := x64.StructLayout(pointerFields)
	if err != nil {
		return err
	}
	if x86Struct.SizeBytes != 8 || x86Struct.AlignBytes != 4 || x32Struct.SizeBytes != 8 || x32Struct.AlignBytes != 4 || x64Struct.SizeBytes != 16 || x64Struct.AlignBytes != 8 {
		return fmt.Errorf("x86/x32/x64 pointer-sensitive struct layouts collapsed: x86=%#v x32=%#v x64=%#v", x86Struct, x32Struct, x64Struct)
	}
	if x86.RegisterWidthBits == x32.RegisterWidthBits || x32.RegisterWidthBits != 64 {
		return fmt.Errorf("x86/x32 register models collapsed: x86=%d x32=%d", x86.RegisterWidthBits, x32.RegisterWidthBits)
	}
	if err := checkTargetArrayBoundaryFuzz(x86, x32, x64); err != nil {
		return err
	}
	return nil
}

func checkTargetArrayBoundaryFuzz(x86 ctarget.Target, x32 ctarget.Target, x64 ctarget.Target) error {
	near32BitByteLimit := 1<<30 - 1
	over32BitByteLimit := 1 << 30
	for _, tgt := range []ctarget.Target{x86, x32} {
		near, err := tgt.ArrayLayout("ptr", near32BitByteLimit)
		if err != nil {
			return fmt.Errorf("%s near-limit pointer array rejected: %w", tgt.Triple, err)
		}
		if got, want := uint64(near.SizeBytes), (uint64(1)<<32)-4; got != want {
			return fmt.Errorf("%s near-limit pointer array size = %d, want %d", tgt.Triple, got, want)
		}
		err = expectArrayLayoutError(tgt, "ptr", over32BitByteLimit)
		if err != nil {
			return err
		}
	}
	large, err := x64.ArrayLayout("ptr", over32BitByteLimit)
	if err != nil {
		return fmt.Errorf("x64 large pointer array rejected at x32 boundary: %w", err)
	}
	if got, want := uint64(large.SizeBytes), uint64(1)<<33; got != want {
		return fmt.Errorf("x64 large pointer array size = %d, want %d", got, want)
	}
	return nil
}

func expectArrayLayoutError(tgt ctarget.Target, elemType string, count int) error {
	if _, err := tgt.ArrayLayout(elemType, count); err == nil {
		return fmt.Errorf("%s [%d]%s layout accepted target-native overflow", tgt.Triple, count, elemType)
	} else if !strings.Contains(err.Error(), "exceeds 32-bit native size limit") {
		return fmt.Errorf("%s [%d]%s overflow error = %q, want native-size diagnostic", tgt.Triple, count, elemType, err)
	}
	return nil
}

func referenceFuzzStructLayout(tgt ctarget.Target, fields []ctarget.LayoutField, packed bool) (ctarget.AggregateLayout, error) {
	out := ctarget.AggregateLayout{AlignBytes: 1, Fields: make([]ctarget.FieldLayout, 0, len(fields))}
	offset := 0
	for _, field := range fields {
		if len(field.Fields) > 0 {
			return ctarget.AggregateLayout{}, fmt.Errorf("nested reference fields are not part of this fuzz oracle")
		}
		scalar, ok := tgt.ScalarLayout(field.Type)
		if !ok {
			return ctarget.AggregateLayout{}, fmt.Errorf("unknown reference layout type %q", field.Type)
		}
		align := scalar.AlignBytes
		if packed || field.Packed {
			align = 1
		}
		offset = fuzzAlignUp(offset, align)
		out.Fields = append(out.Fields, ctarget.FieldLayout{
			Name:        field.Name,
			Type:        scalar.Name,
			OffsetBytes: offset,
			SizeBytes:   scalar.SizeBytes,
			AlignBytes:  align,
			ABIBytes:    scalar.ABIBytes,
		})
		offset += scalar.SizeBytes
		if align > out.AlignBytes {
			out.AlignBytes = align
		}
	}
	out.SizeBytes = fuzzAlignUp(offset, out.AlignBytes)
	return out, nil
}

func compareFuzzAggregateLayout(name string, got ctarget.AggregateLayout, want ctarget.AggregateLayout) error {
	if got.SizeBytes != want.SizeBytes || got.AlignBytes != want.AlignBytes || len(got.Fields) != len(want.Fields) {
		return fmt.Errorf("%s layout = %#v, want %#v", name, got, want)
	}
	for i := range got.Fields {
		gf := got.Fields[i]
		wf := want.Fields[i]
		if gf.Name != wf.Name || gf.Type != wf.Type || gf.OffsetBytes != wf.OffsetBytes || gf.SizeBytes != wf.SizeBytes || gf.AlignBytes != wf.AlignBytes || gf.ABIBytes != wf.ABIBytes {
			return fmt.Errorf("%s field %d = %#v, want %#v", name, i, gf, wf)
		}
	}
	return nil
}

func fuzzAlignUp(value int, align int) int {
	if align <= 1 {
		return value
	}
	remainder := value % align
	if remainder == 0 {
		return value
	}
	return value + align - remainder
}

func checkTargetObjectSignatureFuzz(tgt ctarget.Target) error {
	tmpDir, err := os.MkdirTemp("", "tetra-fuzz-suite-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	prefix := targetFuzzPrefix(tgt)
	srcPath := filepath.Join(tmpDir, prefix+"_signature_fuzz.tetra")
	outPath := filepath.Join(tmpDir, prefix+"_signature_fuzz.tobj")
	rng := rand.New(rand.NewSource(0x32F00D))
	if tgt.Arch == ctarget.ArchX86 {
		rng = rand.New(rand.NewSource(0x86F00D))
	} else if tgt.ABI != ctarget.ABIX32SysV {
		rng = rand.New(rand.NewSource(0x64F00D))
	}
	var src strings.Builder
	var symbols []string
	for i := 0; i < 48; i++ {
		switch rng.Intn(4) {
		case 0:
			name := fmt.Sprintf("fuzz_i32_%02d", i)
			fmt.Fprintf(&src, "fun %s(a: i32, b: i32): i32 { return a + b }\n", name)
			symbols = append(symbols, name)
		case 1:
			name := fmt.Sprintf("fuzz_i64_%02d", i)
			fmt.Fprintf(&src, "fun %s(a: i64): i64 { return a }\n", name)
			symbols = append(symbols, name)
		case 2:
			name := fmt.Sprintf("fuzz_ptr_%02d", i)
			fmt.Fprintf(&src, "fun %s(p: ptr): ptr { return p }\n", name)
			symbols = append(symbols, name)
		default:
			name := fmt.Sprintf("fuzz_mixed_%02d", i)
			fmt.Fprintf(&src, "fun %s(a: i32, p: ptr): i32 { return a }\n", name)
			symbols = append(symbols, name)
		}
	}
	if err := os.WriteFile(srcPath, []byte(src.String()), 0o644); err != nil {
		return err
	}
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, tgt.Triple, BuildOptions{Emit: EmitLibrary}); err != nil {
		return err
	}
	obj, err := ReadObject(outPath)
	if err != nil {
		return err
	}
	if obj.Target != tgt.Triple {
		return fmt.Errorf("target mismatch: got %q want %s", obj.Target, tgt.Triple)
	}
	for _, symbol := range symbols {
		if !objectHasSymbol(obj, symbol) {
			return fmt.Errorf("object signature fuzz missing symbol %s", symbol)
		}
	}
	if tgt.OS != ctarget.OSWindows {
		for _, reloc := range obj.Relocs {
			if reloc.Kind == RelocIATDisp32 {
				return fmt.Errorf("%s signature fuzz unexpectedly has Windows IAT reloc: %#v", tgt.Triple, obj.Relocs)
			}
		}
	}
	return nil
}

func checkTargetAliasFuzz() error {
	x32Aliases := []string{
		"x32", "x86_64-x32", "linux-x32", "linux-x86_64-x32",
		"x86_64-linux-gnux32", "x86_64-unknown-linux-gnux32", "x86_64-pc-linux-gnux32", "linux-x86_64-gnux32",
	}
	for _, alias := range x32Aliases {
		tgt, err := ctarget.Parse(alias)
		if err != nil {
			return fmt.Errorf("x32 alias %q rejected: %w", alias, err)
		}
		if tgt.Triple != "linux-x32" || tgt.Arch != ctarget.ArchX64 || tgt.ABI != ctarget.ABIX32SysV || tgt.PointerWidthBits != 32 || tgt.RegisterWidthBits != 64 {
			return fmt.Errorf("x32 alias %q parsed as %#v", alias, tgt)
		}
	}
	x86Aliases := []string{
		"x86", "i386", "i686", "linux-i386", "linux-i686",
		"i386-linux-gnu", "i686-linux-gnu", "i686-unknown-linux-gnu", "i686-pc-linux-gnu",
	}
	for _, alias := range x86Aliases {
		tgt, err := ctarget.Parse(alias)
		if err != nil {
			return fmt.Errorf("x86 alias %q rejected: %w", alias, err)
		}
		if tgt.Triple != "linux-x86" || tgt.Arch != ctarget.ArchX86 || tgt.ABI != ctarget.ABI386SysV || tgt.PointerWidthBits != 32 || tgt.RegisterWidthBits != 32 {
			return fmt.Errorf("x86 alias %q parsed as %#v", alias, tgt)
		}
	}
	x64Aliases := []string{
		"x64", "amd64", "x86_64", "linux-amd64", "linux-x86_64",
		"x86_64-linux-gnu", "x86_64-unknown-linux-gnu", "x86_64-pc-linux-gnu", "amd64-linux-gnu",
	}
	for _, alias := range x64Aliases {
		tgt, err := ctarget.Parse(alias)
		if err != nil {
			return fmt.Errorf("x64 alias %q rejected: %w", alias, err)
		}
		if tgt.Triple != "linux-x64" || tgt.Arch != ctarget.ArchX64 || tgt.ABI != ctarget.ABISysV || tgt.PointerWidthBits != 64 || tgt.RegisterWidthBits != 64 {
			return fmt.Errorf("x64 alias %q parsed as %#v", alias, tgt)
		}
	}
	windowsX64Aliases := []string{
		"win-x64", "windows-amd64", "windows-x86_64",
		"x86_64-pc-windows-msvc", "x86_64-pc-windows-gnu", "amd64-windows-msvc",
	}
	for _, alias := range windowsX64Aliases {
		tgt, err := ctarget.Parse(alias)
		if err != nil {
			return fmt.Errorf("windows x64 alias %q rejected: %w", alias, err)
		}
		if tgt.Triple != "windows-x64" || tgt.Arch != ctarget.ArchX64 || tgt.ABI != ctarget.ABIWin64 || tgt.DataModel != ctarget.DataModelLLP64 || tgt.Format != ctarget.FormatPE {
			return fmt.Errorf("windows x64 alias %q parsed as %#v", alias, tgt)
		}
	}
	macosX64Aliases := []string{
		"darwin-x64", "macos-amd64", "macos-x86_64",
		"x86_64-apple-darwin", "amd64-apple-darwin",
	}
	for _, alias := range macosX64Aliases {
		tgt, err := ctarget.Parse(alias)
		if err != nil {
			return fmt.Errorf("macos x64 alias %q rejected: %w", alias, err)
		}
		if tgt.Triple != "macos-x64" || tgt.Arch != ctarget.ArchX64 || tgt.ABI != ctarget.ABISysV || tgt.DataModel != ctarget.DataModelLP64 || tgt.Format != ctarget.FormatMachO {
			return fmt.Errorf("macos x64 alias %q parsed as %#v", alias, tgt)
		}
	}
	for _, alias := range []string{"x86-x32", "linux-amd64-x32", "linux-x64-x32", "x32_64"} {
		if tgt, err := ctarget.Parse(alias); err == nil {
			return fmt.Errorf("invalid target alias %q parsed as %#v", alias, tgt)
		}
	}
	return nil
}

func targetFuzzPrefix(tgt ctarget.Target) string {
	if tgt.Arch == ctarget.ArchX86 {
		return "x86"
	}
	if tgt.ABI == ctarget.ABIX32SysV {
		return "x32"
	}
	if tgt.Triple == "windows-x64" || tgt.Triple == "macos-x64" {
		return tgt.Triple
	}
	return "x64"
}
