package target

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	for _, triple := range SupportedTriples() {
		tgt, err := Parse(triple)
		if err != nil {
			t.Fatalf("Parse(%q): %v", triple, err)
		}
		if tgt.Triple != triple {
			t.Fatalf("triple mismatch: got=%q want=%q", tgt.Triple, triple)
		}
		if triple == "windows-x64" && tgt.ExeExt != ".exe" {
			t.Fatalf("windows exe ext mismatch: %q", tgt.ExeExt)
		}
		if triple == "wasm32-wasi" || triple == "wasm32-web" {
			if tgt.ExeExt != ".wasm" {
				t.Fatalf("wasm exe ext mismatch: %q", tgt.ExeExt)
			}
			continue
		}
		if triple != "windows-x64" && tgt.ExeExt != "" {
			t.Fatalf("native non-windows exe ext mismatch: %q", tgt.ExeExt)
		}
	}
}

func TestTargetListsAreStable(t *testing.T) {
	if got := SupportedTriples(); len(got) != 5 || got[0] != "linux-x64" || got[1] != "windows-x64" || got[2] != "macos-x64" || got[3] != "wasm32-wasi" || got[4] != "wasm32-web" {
		t.Fatalf("supported triples = %#v", got)
	}
	if got := BuildOnlyTriples(); len(got) != 2 || got[0] != "linux-x86" || got[1] != "linux-x32" {
		t.Fatalf("build-only triples = %#v", got)
	}
	if got := PlannedTriples(); len(got) != 0 {
		t.Fatalf("planned triples = %#v", got)
	}
	if got := ActorRuntimeTriples(); len(got) != 3 || got[0] != "linux-x64" || got[1] != "macos-x64" || got[2] != "windows-x64" {
		t.Fatalf("actor runtime triples = %#v", got)
	}
}

func TestUIRuntimeMetadataIsTruthfulPerTarget(t *testing.T) {
	cases := []struct {
		triple   string
		status   string
		contract string
	}{
		{"linux-x64", "production", "tetra.ui.platform.v1"},
		{"windows-x64", "requires_target_host_evidence", "tetra.ui.platform.v1"},
		{"macos-x64", "requires_target_host_evidence", "tetra.ui.platform.v1"},
		{"wasm32-web", "production", "tetra.ui.platform.v1"},
		{"wasm32-wasi", "unsupported", ""},
		{"linux-x86", "unsupported", ""},
		{"linux-x32", "unsupported", ""},
	}
	for _, tc := range cases {
		if got := UIRuntimeStatus(tc.triple); got != tc.status {
			t.Fatalf("UIRuntimeStatus(%q) = %q, want %q", tc.triple, got, tc.status)
		}
		if got := UIRuntimeContract(tc.triple); got != tc.contract {
			t.Fatalf("UIRuntimeContract(%q) = %q, want %q", tc.triple, got, tc.contract)
		}
		if got := UIRuntimeEvidence(tc.triple); strings.TrimSpace(got) == "" {
			t.Fatalf("UIRuntimeEvidence(%q) is empty", tc.triple)
		}
	}
}

func TestTargetStatusValues(t *testing.T) {
	cases := []struct {
		triple string
		status Status
	}{
		{"linux-x64", StatusSupported},
		{"windows-x64", StatusSupported},
		{"macos-x64", StatusSupported},
		{"wasm32-wasi", StatusSupported},
		{"wasm32-web", StatusSupported},
	}
	for _, tc := range cases {
		tgt, err := Parse(tc.triple)
		if err != nil {
			t.Fatalf("Parse(%q): %v", tc.triple, err)
		}
		if tgt.Status != tc.status {
			t.Fatalf("Parse(%q).Status = %q, want %q", tc.triple, tgt.Status, tc.status)
		}
	}
	if StatusSupported.String() != "supported" || StatusBuildOnly.String() != "build_only" || StatusPlanned.String() != "planned" {
		t.Fatalf("unexpected status strings: %q %q %q", StatusSupported, StatusBuildOnly, StatusPlanned)
	}
	if RunModeHostNative.String() != "host_native" || RunModeHostProbed.String() != "host_probed" || RunModeWASIRunner.String() != "wasi_runner" || RunModeWebRunner.String() != "web_runner" {
		t.Fatalf("unexpected run mode strings: %q %q %q %q", RunModeHostNative, RunModeHostProbed, RunModeWASIRunner, RunModeWebRunner)
	}
}

func TestParseNormalizesX86X64X32Aliases(t *testing.T) {
	cases := []struct {
		raw       string
		canonical string
		arch      Arch
		abi       ABI
		dataModel DataModel
		ptrBits   int
		regBits   int
		nativeInt int
		status    Status
	}{
		{"x86", "linux-x86", ArchX86, ABI386SysV, DataModelILP32, 32, 32, 32, StatusBuildOnly},
		{"i386", "linux-x86", ArchX86, ABI386SysV, DataModelILP32, 32, 32, 32, StatusBuildOnly},
		{"i686", "linux-x86", ArchX86, ABI386SysV, DataModelILP32, 32, 32, 32, StatusBuildOnly},
		{"linux-i386", "linux-x86", ArchX86, ABI386SysV, DataModelILP32, 32, 32, 32, StatusBuildOnly},
		{"linux-i686", "linux-x86", ArchX86, ABI386SysV, DataModelILP32, 32, 32, 32, StatusBuildOnly},
		{"i386-linux-gnu", "linux-x86", ArchX86, ABI386SysV, DataModelILP32, 32, 32, 32, StatusBuildOnly},
		{"i686-linux-gnu", "linux-x86", ArchX86, ABI386SysV, DataModelILP32, 32, 32, 32, StatusBuildOnly},
		{"i686-unknown-linux-gnu", "linux-x86", ArchX86, ABI386SysV, DataModelILP32, 32, 32, 32, StatusBuildOnly},
		{"i686-pc-linux-gnu", "linux-x86", ArchX86, ABI386SysV, DataModelILP32, 32, 32, 32, StatusBuildOnly},
		{"x64", "linux-x64", ArchX64, ABISysV, DataModelLP64, 64, 64, 64, StatusSupported},
		{"amd64", "linux-x64", ArchX64, ABISysV, DataModelLP64, 64, 64, 64, StatusSupported},
		{"x86_64", "linux-x64", ArchX64, ABISysV, DataModelLP64, 64, 64, 64, StatusSupported},
		{"linux-amd64", "linux-x64", ArchX64, ABISysV, DataModelLP64, 64, 64, 64, StatusSupported},
		{"linux-x86_64", "linux-x64", ArchX64, ABISysV, DataModelLP64, 64, 64, 64, StatusSupported},
		{"x86_64-linux-gnu", "linux-x64", ArchX64, ABISysV, DataModelLP64, 64, 64, 64, StatusSupported},
		{"x86_64-unknown-linux-gnu", "linux-x64", ArchX64, ABISysV, DataModelLP64, 64, 64, 64, StatusSupported},
		{"x86_64-pc-linux-gnu", "linux-x64", ArchX64, ABISysV, DataModelLP64, 64, 64, 64, StatusSupported},
		{"amd64-linux-gnu", "linux-x64", ArchX64, ABISysV, DataModelLP64, 64, 64, 64, StatusSupported},
		{"win-x64", "windows-x64", ArchX64, ABIWin64, DataModelLLP64, 64, 64, 64, StatusSupported},
		{"windows-amd64", "windows-x64", ArchX64, ABIWin64, DataModelLLP64, 64, 64, 64, StatusSupported},
		{"windows-x86_64", "windows-x64", ArchX64, ABIWin64, DataModelLLP64, 64, 64, 64, StatusSupported},
		{"x86_64-pc-windows-msvc", "windows-x64", ArchX64, ABIWin64, DataModelLLP64, 64, 64, 64, StatusSupported},
		{"x86_64-pc-windows-gnu", "windows-x64", ArchX64, ABIWin64, DataModelLLP64, 64, 64, 64, StatusSupported},
		{"amd64-windows-msvc", "windows-x64", ArchX64, ABIWin64, DataModelLLP64, 64, 64, 64, StatusSupported},
		{"darwin-x64", "macos-x64", ArchX64, ABISysV, DataModelLP64, 64, 64, 64, StatusSupported},
		{"macos-amd64", "macos-x64", ArchX64, ABISysV, DataModelLP64, 64, 64, 64, StatusSupported},
		{"macos-x86_64", "macos-x64", ArchX64, ABISysV, DataModelLP64, 64, 64, 64, StatusSupported},
		{"x86_64-apple-darwin", "macos-x64", ArchX64, ABISysV, DataModelLP64, 64, 64, 64, StatusSupported},
		{"amd64-apple-darwin", "macos-x64", ArchX64, ABISysV, DataModelLP64, 64, 64, 64, StatusSupported},
		{"x32", "linux-x32", ArchX64, ABIX32SysV, DataModelX32, 32, 64, 32, StatusBuildOnly},
		{"x86_64-x32", "linux-x32", ArchX64, ABIX32SysV, DataModelX32, 32, 64, 32, StatusBuildOnly},
		{"linux-x32", "linux-x32", ArchX64, ABIX32SysV, DataModelX32, 32, 64, 32, StatusBuildOnly},
		{"linux-x86_64-x32", "linux-x32", ArchX64, ABIX32SysV, DataModelX32, 32, 64, 32, StatusBuildOnly},
		{"x86_64-linux-gnux32", "linux-x32", ArchX64, ABIX32SysV, DataModelX32, 32, 64, 32, StatusBuildOnly},
		{"x86_64-unknown-linux-gnux32", "linux-x32", ArchX64, ABIX32SysV, DataModelX32, 32, 64, 32, StatusBuildOnly},
		{"x86_64-pc-linux-gnux32", "linux-x32", ArchX64, ABIX32SysV, DataModelX32, 32, 64, 32, StatusBuildOnly},
		{"linux-x86_64-gnux32", "linux-x32", ArchX64, ABIX32SysV, DataModelX32, 32, 64, 32, StatusBuildOnly},
	}

	for _, tc := range cases {
		tgt, err := Parse(tc.raw)
		if err != nil {
			t.Fatalf("Parse(%q): %v", tc.raw, err)
		}
		if tgt.Triple != tc.canonical || tgt.Arch != tc.arch || tgt.ABI != tc.abi || tgt.DataModel != tc.dataModel || tgt.Status != tc.status {
			t.Fatalf("Parse(%q) = %#v", tc.raw, tgt)
		}
		if tgt.PointerWidthBits != tc.ptrBits || tgt.RegisterWidthBits != tc.regBits || tgt.NativeIntWidthBits != tc.nativeInt {
			t.Fatalf("Parse(%q) widths ptr=%d reg=%d native=%d, want ptr=%d reg=%d native=%d",
				tc.raw, tgt.PointerWidthBits, tgt.RegisterWidthBits, tgt.NativeIntWidthBits, tc.ptrBits, tc.regBits, tc.nativeInt)
		}
		if tgt.Endian != EndianLittle {
			t.Fatalf("Parse(%q).Endian = %s, want little", tc.raw, tgt.Endian)
		}
	}
}

func TestX32IsNeitherX86NorX64Layout(t *testing.T) {
	x86 := mustParse(t, "x86")
	x64 := mustParse(t, "x64")
	x32 := mustParse(t, "x32")

	if x32.Arch != ArchX64 || x32.RegisterWidthBits != 64 {
		t.Fatalf("x32 ISA/register model = arch %s reg %d, want x64 ISA with 64-bit registers", x32.Arch, x32.RegisterWidthBits)
	}
	if x32.PointerWidthBits != 32 || x32.NativeIntWidthBits != 32 {
		t.Fatalf("x32 pointer/native int widths = %d/%d, want 32/32", x32.PointerWidthBits, x32.NativeIntWidthBits)
	}
	if x32.ABI == x86.ABI || x32.ABI == x64.ABI {
		t.Fatalf("x32 ABI must be distinct from x86 and x64: x86=%s x64=%s x32=%s", x86.ABI, x64.ABI, x32.ABI)
	}
	if x32.PointerWidthBits != x86.PointerWidthBits {
		t.Fatalf("x32 and x86 pointer width should both be 32 bits")
	}
	if x32.RegisterWidthBits == x86.RegisterWidthBits {
		t.Fatalf("x32 register width must not collapse to x86 register width")
	}
	if x32.PointerWidthBits == x64.PointerWidthBits {
		t.Fatalf("x32 pointer width must not collapse to x64 pointer width")
	}
}

func TestScalarLayoutsForNativeIntegerPointerAndCABIWidths(t *testing.T) {
	cases := []struct {
		target  string
		scalar  string
		size    int
		align   int
		abiSize int
	}{
		{"x86", "ptr", 4, 4, 4},
		{"x86", "usize", 4, 4, 4},
		{"x86", "isize", 4, 4, 4},
		{"x86", "c_long", 4, 4, 4},
		{"x64", "ptr", 8, 8, 8},
		{"x64", "usize", 8, 8, 8},
		{"x64", "isize", 8, 8, 8},
		{"x64", "c_long", 8, 8, 8},
		{"windows-x64", "c_long", 4, 4, 4},
		{"x32", "ptr", 4, 4, 4},
		{"x32", "usize", 4, 4, 4},
		{"x32", "isize", 4, 4, 4},
		{"x32", "c_long", 4, 4, 4},
		{"x32", "u64", 8, 8, 8},
		{"x32", "fnptr", 4, 4, 4},
	}

	for _, tc := range cases {
		tgt := mustParse(t, tc.target)
		got, ok := tgt.ScalarLayout(tc.scalar)
		if !ok {
			t.Fatalf("%s scalar %s not found", tc.target, tc.scalar)
		}
		if got.SizeBytes != tc.size || got.AlignBytes != tc.align || got.ABIBytes != tc.abiSize {
			t.Fatalf("%s scalar %s layout = size %d align %d abi %d, want size %d align %d abi %d",
				tc.target, tc.scalar, got.SizeBytes, got.AlignBytes, got.ABIBytes, tc.size, tc.align, tc.abiSize)
		}
	}
}

func TestStructLayoutSeparatesPointerWidthFromRegisterWidth(t *testing.T) {
	fields := []LayoutField{
		{Name: "tag", Type: "u8"},
		{Name: "raw", Type: "ptr"},
		{Name: "count", Type: "u16"},
	}

	x64Layout, err := mustParse(t, "x64").StructLayout(fields)
	if err != nil {
		t.Fatalf("x64 StructLayout: %v", err)
	}
	assertStructLayout(t, "x64", x64Layout, 24, 8, []int{0, 8, 16})

	x32Layout, err := mustParse(t, "x32").StructLayout(fields)
	if err != nil {
		t.Fatalf("x32 StructLayout: %v", err)
	}
	assertStructLayout(t, "x32", x32Layout, 12, 4, []int{0, 4, 8})

	x86Layout, err := mustParse(t, "x86").StructLayout(fields)
	if err != nil {
		t.Fatalf("x86 StructLayout: %v", err)
	}
	assertStructLayout(t, "x86", x86Layout, 12, 4, []int{0, 4, 8})
}

func TestCompoundLayoutsUseTargetPointerWidth(t *testing.T) {
	cases := []struct {
		target          string
		arraySize       int
		arrayAlign      int
		sliceSize       int
		sliceAlign      int
		sliceOffsets    []int
		enumSize        int
		enumAlign       int
		enumPayloadOff  int
		enumPayloadSize int
	}{
		{"x86", 12, 4, 8, 4, []int{0, 4}, 8, 4, 4, 4},
		{"x64", 24, 8, 16, 8, []int{0, 8}, 16, 8, 8, 8},
		{"x32", 12, 4, 8, 4, []int{0, 4}, 8, 4, 4, 4},
	}
	for _, tc := range cases {
		tgt := mustParse(t, tc.target)

		array, err := tgt.ArrayLayout("ptr", 3)
		if err != nil {
			t.Fatalf("%s ArrayLayout: %v", tc.target, err)
		}
		if array.SizeBytes != tc.arraySize || array.AlignBytes != tc.arrayAlign || array.ElemType != "ptr" || array.Len != 3 {
			t.Fatalf("%s ptr[3] layout = %#v, want size=%d align=%d elem=ptr len=3", tc.target, array, tc.arraySize, tc.arrayAlign)
		}

		slice, err := tgt.SliceLayout("u8")
		if err != nil {
			t.Fatalf("%s SliceLayout: %v", tc.target, err)
		}
		assertStructLayout(t, tc.target+" []u8", slice, tc.sliceSize, tc.sliceAlign, tc.sliceOffsets)
		if slice.Fields[0].Type != "ptr" || slice.Fields[1].Type != "i32" {
			t.Fatalf("%s slice fields = %#v, want ptr+i32", tc.target, slice.Fields)
		}

		str, err := tgt.StringLayout()
		if err != nil {
			t.Fatalf("%s StringLayout: %v", tc.target, err)
		}
		assertStructLayout(t, tc.target+" str", str, tc.sliceSize, tc.sliceAlign, tc.sliceOffsets)

		enumLayout, err := tgt.EnumLayout([]EnumCaseLayout{
			{Name: "none"},
			{Name: "some", Payload: []LayoutField{{Name: "value", Type: "ptr"}}},
		})
		if err != nil {
			t.Fatalf("%s EnumLayout: %v", tc.target, err)
		}
		if enumLayout.SizeBytes != tc.enumSize || enumLayout.AlignBytes != tc.enumAlign || enumLayout.PayloadOffsetBytes != tc.enumPayloadOff || enumLayout.PayloadSizeBytes != tc.enumPayloadSize {
			t.Fatalf("%s enum layout = %#v, want size=%d align=%d payloadOff=%d payloadSize=%d",
				tc.target, enumLayout, tc.enumSize, tc.enumAlign, tc.enumPayloadOff, tc.enumPayloadSize)
		}
	}
}

func TestNestedAndPackedStructLayouts(t *testing.T) {
	fields := []LayoutField{
		{Name: "tag", Type: "u8"},
		{Name: "inner", Fields: []LayoutField{
			{Name: "raw", Type: "ptr"},
			{Name: "count", Type: "u16"},
		}},
		{Name: "tail", Type: "u8"},
	}

	x64Layout, err := mustParse(t, "x64").StructLayout(fields)
	if err != nil {
		t.Fatalf("x64 nested StructLayout: %v", err)
	}
	assertStructLayout(t, "x64 nested", x64Layout, 32, 8, []int{0, 8, 24})
	if len(x64Layout.Fields[1].Fields) != 2 || x64Layout.Fields[1].Fields[1].OffsetBytes != 8 {
		t.Fatalf("x64 nested field layout = %#v", x64Layout.Fields[1])
	}

	x32Layout, err := mustParse(t, "x32").StructLayout(fields)
	if err != nil {
		t.Fatalf("x32 nested StructLayout: %v", err)
	}
	assertStructLayout(t, "x32 nested", x32Layout, 16, 4, []int{0, 4, 12})
	if len(x32Layout.Fields[1].Fields) != 2 || x32Layout.Fields[1].Fields[1].OffsetBytes != 4 {
		t.Fatalf("x32 nested field layout = %#v", x32Layout.Fields[1])
	}

	packed, err := mustParse(t, "x64").PackedStructLayout([]LayoutField{
		{Name: "tag", Type: "u8"},
		{Name: "raw", Type: "ptr"},
		{Name: "count", Type: "u16"},
	})
	if err != nil {
		t.Fatalf("PackedStructLayout: %v", err)
	}
	assertStructLayout(t, "x64 packed", packed, 11, 1, []int{0, 1, 9})
}

func TestLayoutRejectsInvalidCompoundInputs(t *testing.T) {
	tgt := mustParse(t, "x32")
	if _, err := tgt.ArrayLayout("ptr", -1); err == nil {
		t.Fatalf("negative array length accepted")
	}
	if _, err := tgt.ArrayLayout("missing", 2); err == nil {
		t.Fatalf("unknown array element layout accepted")
	}
	if _, err := tgt.StructLayout([]LayoutField{{Name: "bad", Type: "missing"}}); err == nil {
		t.Fatalf("unknown struct field layout accepted")
	}
	if _, err := tgt.EnumLayout(nil); err == nil {
		t.Fatalf("empty enum layout accepted")
	}
}

func TestArrayLayoutRejectsTargetNativeSizeOverflow(t *testing.T) {
	near32BitByteLimit := 1<<30 - 1
	over32BitByteLimit := 1 << 30

	for _, raw := range []string{"x86", "x32"} {
		tgt := mustParse(t, raw)
		near, err := tgt.ArrayLayout("ptr", near32BitByteLimit)
		if err != nil {
			t.Fatalf("%s ArrayLayout near 32-bit byte limit rejected: %v", raw, err)
		}
		if got, want := uint64(near.SizeBytes), (uint64(1)<<32)-4; got != want {
			t.Fatalf("%s near-limit ptr array size = %d, want %d", raw, got, want)
		}

		_, err = tgt.ArrayLayout("ptr", over32BitByteLimit)
		if err == nil {
			t.Fatalf("%s ArrayLayout accepted 4GiB pointer array, want target usize overflow diagnostic", raw)
		}
		if !strings.Contains(err.Error(), "exceeds 32-bit native size limit") {
			t.Fatalf("%s overflow error = %q, want explicit 32-bit native size diagnostic", raw, err)
		}
	}

	x64 := mustParse(t, "x64")
	large, err := x64.ArrayLayout("ptr", over32BitByteLimit)
	if err != nil {
		t.Fatalf("x64 ArrayLayout at 4GiB pointer-count boundary rejected: %v", err)
	}
	if got, want := uint64(large.SizeBytes), uint64(1)<<33; got != want {
		t.Fatalf("x64 large ptr array size = %d, want %d", got, want)
	}
}

func TestAtomicPolicyCoversPointerSizedAndFixedWidths(t *testing.T) {
	cases := []struct {
		target         string
		pointerWidth   int
		pointerAlign   int
		registerWidth  int
		supportedWidth []int
	}{
		{"x86", 32, 4, 32, []int{8, 16, 32}},
		{"x64", 64, 8, 64, []int{8, 16, 32, 64}},
		{"x32", 32, 4, 64, []int{8, 16, 32, 64}},
	}
	for _, tc := range cases {
		tgt := mustParse(t, tc.target)
		ptr, err := tgt.AtomicPointerLayout()
		if err != nil {
			t.Fatalf("%s AtomicPointerLayout: %v", tc.target, err)
		}
		if ptr.WidthBits != tc.pointerWidth || ptr.AlignBytes != tc.pointerAlign || ptr.RegisterWidthBits != tc.registerWidth || !ptr.PointerSized {
			t.Fatalf("%s pointer atomic = %#v, want width=%d align=%d reg=%d pointer-sized", tc.target, ptr, tc.pointerWidth, tc.pointerAlign, tc.registerWidth)
		}
		for _, width := range tc.supportedWidth {
			got, err := tgt.AtomicLayout(width)
			if err != nil {
				t.Fatalf("%s AtomicLayout(%d): %v", tc.target, width, err)
			}
			if got.WidthBits != width || got.SizeBytes != width/8 || got.AlignBytes != width/8 {
				t.Fatalf("%s AtomicLayout(%d) = %#v", tc.target, width, got)
			}
			if !got.LockFree {
				t.Fatalf("%s AtomicLayout(%d) LockFree = false", tc.target, width)
			}
		}
	}
	x86 := mustParse(t, "x86")
	if _, err := x86.AtomicLayout(64); err == nil {
		t.Fatalf("x86 AtomicLayout(64) succeeded, want explicit unsupported-width error without CPU-feature model")
	}
	if err := x86.ValidateAtomic(AtomicCompareExchange, 64, 8, MemoryOrderSeqCst); err == nil {
		t.Fatalf("x86 64-bit atomic compare_exchange succeeded, want explicit unsupported-width error")
	}
}

func TestAtomicPolicyRejectsUnsupportedWidthAlignmentAndOrder(t *testing.T) {
	tgt := mustParse(t, "x32")

	for _, width := range []int{0, 24, 128} {
		if _, err := tgt.AtomicLayout(width); err == nil {
			t.Fatalf("AtomicLayout(%d) succeeded, want explicit unsupported-width error", width)
		}
	}
	if err := tgt.ValidateAtomic(AtomicLoad, 32, 2, MemoryOrderAcquire); err == nil {
		t.Fatalf("misaligned 32-bit atomic load succeeded")
	}
	if err := tgt.ValidateAtomic(AtomicLoad, 32, 4, MemoryOrderRelease); err == nil {
		t.Fatalf("atomic load with release order succeeded")
	}
	if err := tgt.ValidateAtomic(AtomicStore, 32, 4, MemoryOrderAcquire); err == nil {
		t.Fatalf("atomic store with acquire order succeeded")
	}
	if err := tgt.ValidateAtomic(AtomicFetchAdd, 64, 8, MemoryOrderAcqRel); err != nil {
		t.Fatalf("atomic fetch-add acq_rel rejected: %v", err)
	}
	if err := tgt.ValidateAtomic(AtomicFence, 0, 0, MemoryOrderSeqCst); err != nil {
		t.Fatalf("seq_cst fence rejected: %v", err)
	}
}

func TestBuildOnlyArchitecturesHaveHonestRuntimeMetadata(t *testing.T) {
	requiredEvidence := map[string][]string{
		"x86": {
			"i386 SysV ABI classifier",
			"explicit filesystem/networking stdlib plus time/task/actors target-runtime boundary diagnostics",
			"x86 pointer/native-libc/function-pointer @export diagnostics",
			"source native scalar diagnostics",
			"pointer-only atomic ABI-width object check",
			"source-level atomic diagnostics",
		},
		"x32": {
			"x32 SysV ABI classifier",
			"raw ptr_add/load/store",
			"pointer load/store",
			"MMIO read/write",
			"scoped island bump allocation/free",
			"explicit filesystem/networking stdlib plus x32 multi-spawn actors/task, task-group, and typed-task runtime boundary diagnostics",
			"x32 pointer/native-libc/function-pointer @export diagnostics",
			"source native scalar diagnostics",
			"pointer-only atomic ABI-width object check",
			"dword pointer atomics",
			"x32 syscall numbers",
		},
	}
	for _, raw := range []string{"x86", "x32"} {
		tgt := mustParse(t, raw)
		if tgt.Status != StatusBuildOnly || !IsBuildOnlyTarget(tgt.Triple) {
			t.Fatalf("%s status/build-only = %s/%v, want build_only/true", raw, tgt.Status, IsBuildOnlyTarget(tgt.Triple))
		}
		if tgt.RunMode != RunModeHostProbed {
			t.Fatalf("%s run mode = %s, want host-probed", raw, tgt.RunMode)
		}
		if tgt.UnsupportedReason == "" {
			t.Fatalf("%s missing unsupported reason", raw)
		}
		for _, want := range requiredEvidence[raw] {
			if !strings.Contains(tgt.UnsupportedReason, want) {
				t.Fatalf("%s unsupported reason missing %q: %q", raw, want, tgt.UnsupportedReason)
			}
		}
	}
}

func mustParse(t *testing.T, raw string) Target {
	t.Helper()
	tgt, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse(%q): %v", raw, err)
	}
	return tgt
}

func assertStructLayout(t *testing.T, name string, got AggregateLayout, size int, align int, offsets []int) {
	t.Helper()
	if got.SizeBytes != size || got.AlignBytes != align {
		t.Fatalf("%s struct layout = size %d align %d, want size %d align %d", name, got.SizeBytes, got.AlignBytes, size, align)
	}
	if len(got.Fields) != len(offsets) {
		t.Fatalf("%s field count = %d, want %d", name, len(got.Fields), len(offsets))
	}
	for i, want := range offsets {
		if got.Fields[i].OffsetBytes != want {
			t.Fatalf("%s field %s offset = %d, want %d", name, got.Fields[i].Name, got.Fields[i].OffsetBytes, want)
		}
	}
}

func TestParseRejectsUnknown(t *testing.T) {
	if _, err := Parse("plan9-x64"); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestParseAcceptsWASIRuntimeTarget(t *testing.T) {
	tgt, err := Parse("wasm32-wasi")
	if err != nil {
		t.Fatalf("Parse(wasm32-wasi): %v", err)
	}
	if tgt.Triple != "wasm32-wasi" || tgt.ExeExt != ".wasm" {
		t.Fatalf("wasm32-wasi target = %#v", tgt)
	}
	if IsBuildOnlyTarget("wasm32-wasi") {
		t.Fatalf("IsBuildOnlyTarget(wasm32-wasi) = true")
	}
	if IsPlannedTarget("wasm32-wasi") {
		t.Fatalf("IsPlannedTarget(wasm32-wasi) = true")
	}
}

func TestParseAcceptsWASMWebRuntimeTarget(t *testing.T) {
	tgt, err := Parse("wasm32-web")
	if err != nil {
		t.Fatalf("Parse(wasm32-web): %v", err)
	}
	if tgt.Triple != "wasm32-web" || tgt.ExeExt != ".wasm" {
		t.Fatalf("wasm32-web target = %#v", tgt)
	}
	if IsBuildOnlyTarget("wasm32-web") {
		t.Fatalf("IsBuildOnlyTarget(wasm32-web) = true")
	}
	if IsPlannedTarget("wasm32-web") {
		t.Fatalf("IsPlannedTarget(wasm32-web) = true")
	}
}

func TestParseRejectsUnknownAsUnplanned(t *testing.T) {
	_, err := Parse("plan9-x64")
	targetErr, ok := err.(UnsupportedTargetError)
	if !ok {
		t.Fatalf("error type = %T, want UnsupportedTargetError", err)
	}
	if targetErr.Planned {
		t.Fatalf("unknown target marked planned: %#v", targetErr)
	}
}

func TestTargetCapabilitiesForDebugInfoAndReleaseOptimize(t *testing.T) {
	native, err := Parse("linux-x64")
	if err != nil {
		t.Fatalf("Parse(linux-x64): %v", err)
	}
	if !native.SupportsDebugInfo {
		t.Fatalf("linux-x64 SupportsDebugInfo = false")
	}
	if !native.SupportsReleaseOptimize {
		t.Fatalf("linux-x64 SupportsReleaseOptimize = false")
	}

	wasmWASI, err := Parse("wasm32-wasi")
	if err != nil {
		t.Fatalf("Parse(wasm32-wasi): %v", err)
	}
	if wasmWASI.SupportsDebugInfo {
		t.Fatalf("wasm32-wasi SupportsDebugInfo = true")
	}
	if !wasmWASI.SupportsReleaseOptimize {
		t.Fatalf("wasm32-wasi SupportsReleaseOptimize = false")
	}

	wasmWeb, err := Parse("wasm32-web")
	if err != nil {
		t.Fatalf("Parse(wasm32-web): %v", err)
	}
	if wasmWeb.SupportsDebugInfo {
		t.Fatalf("wasm32-web SupportsDebugInfo = true")
	}
	if !wasmWeb.SupportsReleaseOptimize {
		t.Fatalf("wasm32-web SupportsReleaseOptimize = false")
	}
}

func TestCurrentTargetContractIncludesRunnableWASMTargets(t *testing.T) {
	all := All()
	if len(all) != len(SupportedTriples()) {
		t.Fatalf("All() = %#v, want supported triples %#v", all, SupportedTriples())
	}
	for _, tgt := range all {
		if tgt.Status != StatusSupported || IsBuildOnlyTarget(tgt.Triple) || IsPlannedTarget(tgt.Triple) {
			t.Fatalf("All() included non-supported target: %#v", tgt)
		}
	}
	for _, triple := range WASMTriples() {
		tgt, err := Parse(triple)
		if err != nil {
			t.Fatalf("Parse(%q): %v", triple, err)
		}
		if tgt.Status != StatusSupported || IsBuildOnlyTarget(triple) || IsPlannedTarget(triple) {
			t.Fatalf("WASM target %s contract drifted: %#v", triple, tgt)
		}
		if tgt.Arch != ArchWASM32 || tgt.Format != FormatWASM || tgt.ExeExt != ".wasm" || tgt.SupportsDebugInfo {
			t.Fatalf("WASM target %s metadata drifted: %#v", triple, tgt)
		}
		if triple == "wasm32-wasi" && (tgt.RunMode != RunModeWASIRunner || tgt.RunRunner != "wasmtime") {
			t.Fatalf("WASM target %s runner metadata drifted: %#v", triple, tgt)
		}
		if triple == "wasm32-web" && (tgt.RunMode != RunModeWebRunner || tgt.RunRunner != "") {
			t.Fatalf("WASM target %s runner metadata drifted: %#v", triple, tgt)
		}
	}
}
