package target

import (
	"fmt"
	"runtime"
	"strings"
)

type UnsupportedTargetError struct {
	Triple  string
	Planned bool
}

func (e UnsupportedTargetError) Error() string {
	if e.Planned {
		return fmt.Sprintf("planned target not implemented: %s", e.Triple)
	}
	return fmt.Sprintf("unsupported target: %s", e.Triple)
}

type OS int

const (
	OSUnknown OS = iota
	OSLinux
	OSWindows
	OSMacOS
	OSWASI
	OSWeb
)

func (o OS) String() string {
	switch o {
	case OSLinux:
		return "linux"
	case OSWindows:
		return "windows"
	case OSMacOS:
		return "macos"
	case OSWASI:
		return "wasi"
	case OSWeb:
		return "web"
	default:
		return "unknown"
	}
}

type Arch int

const (
	ArchUnknown Arch = iota
	ArchX86
	ArchX64
	ArchWASM32
)

func (a Arch) String() string {
	switch a {
	case ArchX86:
		return "x86"
	case ArchX64:
		return "x64"
	case ArchWASM32:
		return "wasm32"
	default:
		return "unknown"
	}
}

type ABI int

const (
	ABIUnknown ABI = iota
	ABI386SysV
	ABISysV
	ABIX32SysV
	ABIWin64
	ABIWASI
	ABIWeb
)

func (a ABI) String() string {
	switch a {
	case ABI386SysV:
		return "i386-sysv"
	case ABISysV:
		return "sysv"
	case ABIX32SysV:
		return "x32-sysv"
	case ABIWin64:
		return "win64"
	case ABIWASI:
		return "wasi"
	case ABIWeb:
		return "web"
	default:
		return "unknown"
	}
}

type Endian int

const (
	EndianUnknown Endian = iota
	EndianLittle
	EndianBig
)

func (e Endian) String() string {
	switch e {
	case EndianLittle:
		return "little"
	case EndianBig:
		return "big"
	default:
		return "unknown"
	}
}

type DataModel int

const (
	DataModelUnknown DataModel = iota
	DataModelILP32
	DataModelLP64
	DataModelLLP64
	DataModelX32
)

func (m DataModel) String() string {
	switch m {
	case DataModelILP32:
		return "ilp32"
	case DataModelLP64:
		return "lp64"
	case DataModelLLP64:
		return "llp64"
	case DataModelX32:
		return "x32"
	default:
		return "unknown"
	}
}

type Format int

const (
	FormatUnknown Format = iota
	FormatELF
	FormatPE
	FormatMachO
	FormatWASM
)

func (f Format) String() string {
	switch f {
	case FormatELF:
		return "elf"
	case FormatPE:
		return "pe"
	case FormatMachO:
		return "macho"
	case FormatWASM:
		return "wasm"
	default:
		return "unknown"
	}
}

type Status int

const (
	StatusUnknown Status = iota
	StatusSupported
	StatusBuildOnly
	StatusPlanned
)

func (s Status) String() string {
	switch s {
	case StatusSupported:
		return "supported"
	case StatusBuildOnly:
		return "build_only"
	case StatusPlanned:
		return "planned"
	default:
		return "unknown"
	}
}

type RunMode int

const (
	RunModeUnknown RunMode = iota
	RunModeHostNative
	RunModeHostProbed
	RunModeWASIRunner
	RunModeWebRunner
	RunModeUnsupported
)

func (m RunMode) String() string {
	switch m {
	case RunModeHostNative:
		return "host_native"
	case RunModeHostProbed:
		return "host_probed"
	case RunModeWASIRunner:
		return "wasi_runner"
	case RunModeWebRunner:
		return "web_runner"
	case RunModeUnsupported:
		return "unsupported"
	default:
		return "unknown"
	}
}

type ScalarLayout struct {
	Name       string
	SizeBytes  int
	AlignBytes int
	ABIBytes   int
}

type LayoutField struct {
	Name   string
	Type   string
	Packed bool
	Fields []LayoutField
}

type FieldLayout struct {
	Name        string
	Type        string
	OffsetBytes int
	SizeBytes   int
	AlignBytes  int
	ABIBytes    int
	Fields      []FieldLayout
}

type AggregateLayout struct {
	SizeBytes          int
	AlignBytes         int
	Fields             []FieldLayout
	PayloadOffsetBytes int
	PayloadSizeBytes   int
}

type TypeLayout struct {
	Name       string
	SizeBytes  int
	AlignBytes int
	ABIBytes   int
	ElemType   string
	Len        int
	Fields     []FieldLayout
}

type EnumCaseLayout struct {
	Name    string
	Payload []LayoutField
}

type MemoryOrder int

const (
	MemoryOrderUnknown MemoryOrder = iota
	MemoryOrderRelaxed
	MemoryOrderAcquire
	MemoryOrderRelease
	MemoryOrderAcqRel
	MemoryOrderSeqCst
)

func (o MemoryOrder) String() string {
	switch o {
	case MemoryOrderRelaxed:
		return "relaxed"
	case MemoryOrderAcquire:
		return "acquire"
	case MemoryOrderRelease:
		return "release"
	case MemoryOrderAcqRel:
		return "acq_rel"
	case MemoryOrderSeqCst:
		return "seq_cst"
	default:
		return "unknown"
	}
}

type AtomicOp int

const (
	AtomicOpUnknown AtomicOp = iota
	AtomicLoad
	AtomicStore
	AtomicExchange
	AtomicCompareExchange
	AtomicCompareExchangeWeak
	AtomicFetchAdd
	AtomicFetchSub
	AtomicFetchAnd
	AtomicFetchOr
	AtomicFetchXor
	AtomicFence
)

func (op AtomicOp) String() string {
	switch op {
	case AtomicLoad:
		return "load"
	case AtomicStore:
		return "store"
	case AtomicExchange:
		return "exchange"
	case AtomicCompareExchange:
		return "compare_exchange"
	case AtomicCompareExchangeWeak:
		return "compare_exchange_weak"
	case AtomicFetchAdd:
		return "fetch_add"
	case AtomicFetchSub:
		return "fetch_sub"
	case AtomicFetchAnd:
		return "fetch_and"
	case AtomicFetchOr:
		return "fetch_or"
	case AtomicFetchXor:
		return "fetch_xor"
	case AtomicFence:
		return "fence"
	default:
		return "unknown"
	}
}

type AtomicLayout struct {
	WidthBits         int
	SizeBytes         int
	AlignBytes        int
	RegisterWidthBits int
	LockFree          bool
	PointerSized      bool
}

type Target struct {
	Triple                  string
	Status                  Status
	OS                      OS
	Arch                    Arch
	ABI                     ABI
	DataModel               DataModel
	Format                  Format
	ExeExt                  string
	CollectImports          bool
	RunMode                 RunMode
	RunRunner               string
	PointerWidthBits        int
	RegisterWidthBits       int
	NativeIntWidthBits      int
	Endian                  Endian
	StackAlignmentBytes     int
	MaxAtomicWidthBits      int
	UnsupportedReason       string
	SupportsDebugInfo       bool
	SupportsReleaseOptimize bool
}

func All() []Target {
	out := make([]Target, 0, len(SupportedTriples()))
	for _, triple := range SupportedTriples() {
		t, err := Parse(triple)
		if err == nil {
			out = append(out, t)
		}
	}
	return out
}

func AllBuildable() []Target {
	triples := append([]string{}, SupportedTriples()...)
	triples = append(triples, BuildOnlyTriples()...)
	out := make([]Target, 0, len(triples))
	for _, triple := range triples {
		t, err := Parse(triple)
		if err == nil {
			out = append(out, t)
		}
	}
	return out
}

func SupportedTriples() []string {
	return []string{"linux-x64", "windows-x64", "macos-x64", "wasm32-wasi", "wasm32-web"}
}

func BuildOnlyTriples() []string {
	return []string{"linux-x86", "linux-x32"}
}

func PlannedTriples() []string {
	return []string{}
}

func WASMTriples() []string {
	return []string{"wasm32-wasi", "wasm32-web"}
}

func ActorRuntimeTriples() []string {
	return []string{"linux-x64", "macos-x64", "windows-x64"}
}

func UIRuntimeContract(triple string) string {
	switch normalizeAlias(triple) {
	case "linux-x64", "windows-x64", "macos-x64", "wasm32-web":
		return "tetra.ui.platform.v1"
	default:
		return ""
	}
}

func UIRuntimeStatus(triple string) string {
	switch normalizeAlias(triple) {
	case "linux-x64", "wasm32-web":
		return "production"
	case "windows-x64", "macos-x64":
		return "requires_target_host_evidence"
	default:
		return "unsupported"
	}
}

func UIRuntimeEvidence(triple string) string {
	switch normalizeAlias(triple) {
	case "linux-x64":
		return "scripts/release/post_v0_4/ui-production-runtime-linux-x64-smoke.sh"
	case "windows-x64":
		return "scripts/release/full_platform/windows-ui-runtime-smoke.sh --evidence <windows-target-host-report>"
	case "macos-x64":
		return "scripts/release/full_platform/macos-ui-runtime-smoke.sh --evidence <macos-target-host-report>"
	case "wasm32-web":
		return "scripts/release/v1_0/web-smoke.sh"
	case "wasm32-wasi":
		return "wasm32-wasi does not provide UI event dispatch runtime"
	default:
		return "target does not provide production UI runtime"
	}
}

func Parse(triple string) (Target, error) {
	rawTriple := strings.TrimSpace(triple)
	canonical := normalizeAlias(rawTriple)
	switch canonical {
	case "linux-x86":
		return Target{
			Triple:                  "linux-x86",
			Status:                  StatusBuildOnly,
			OS:                      OSLinux,
			Arch:                    ArchX86,
			ABI:                     ABI386SysV,
			DataModel:               DataModelILP32,
			Format:                  FormatELF,
			ExeExt:                  "",
			CollectImports:          false,
			RunMode:                 RunModeHostProbed,
			PointerWidthBits:        32,
			RegisterWidthBits:       32,
			NativeIntWidthBits:      32,
			Endian:                  EndianLittle,
			StackAlignmentBytes:     16,
			MaxAtomicWidthBits:      32,
			UnsupportedReason:       "full linux-x86 runtime/stdlib/FFI support is not implemented yet; no-runtime executable build/link, i386-compatible Linux run/test execution, stdout write/string literal data, stack-argument, scalar global, symbol-backed callback, heap-backed slice allocation/indexing, raw ptr_add/load/store, MMIO read/write, scoped island bump allocation/free plus debug double-free guard/page-protect object codegen, ELF/linker primitives, i386 SysV ABI classifier, explicit filesystem/networking stdlib plus time/task/actors target-runtime boundary diagnostics, x86 pointer/native-libc/function-pointer @export diagnostics, source native scalar diagnostics, pointer-only atomic ABI-width object check, source-level atomic diagnostics, and full 8/16/32-bit atomic/fuzz object checks are available",
			SupportsDebugInfo:       false,
			SupportsReleaseOptimize: false,
		}, nil
	case "linux-x64":
		return Target{
			Triple:                  "linux-x64",
			Status:                  StatusSupported,
			OS:                      OSLinux,
			Arch:                    ArchX64,
			ABI:                     ABISysV,
			DataModel:               DataModelLP64,
			Format:                  FormatELF,
			ExeExt:                  "",
			CollectImports:          false,
			RunMode:                 RunModeHostNative,
			PointerWidthBits:        64,
			RegisterWidthBits:       64,
			NativeIntWidthBits:      64,
			Endian:                  EndianLittle,
			StackAlignmentBytes:     16,
			MaxAtomicWidthBits:      64,
			SupportsDebugInfo:       true,
			SupportsReleaseOptimize: true,
		}, nil
	case "linux-x32":
		return Target{
			Triple:                  "linux-x32",
			Status:                  StatusBuildOnly,
			OS:                      OSLinux,
			Arch:                    ArchX64,
			ABI:                     ABIX32SysV,
			DataModel:               DataModelX32,
			Format:                  FormatELF,
			ExeExt:                  "",
			CollectImports:          false,
			RunMode:                 RunModeHostProbed,
			PointerWidthBits:        32,
			RegisterWidthBits:       64,
			NativeIntWidthBits:      32,
			Endian:                  EndianLittle,
			StackAlignmentBytes:     16,
			MaxAtomicWidthBits:      64,
			UnsupportedReason:       "full linux-x32 runtime/stdlib/FFI support is not implemented yet; executable build/link, object codegen, ELF/linker primitives, no-runtime programs, raw ptr_add/load/store, pointer load/store, MMIO read/write, scoped island bump allocation/free, self-host runtime builds, compiler-owned target suites, x32 SysV ABI classifier, explicit filesystem/networking stdlib plus x32 multi-spawn actors/task, task-group, and typed-task runtime boundary diagnostics, scalar i32 @export object smoke, x32 pointer/native-libc/function-pointer @export diagnostics, source native scalar diagnostics, x32 syscall numbers, pointer-only atomic ABI-width object check, dword pointer atomics, and host-probed source run/test execution are available when the Linux kernel supports the x32 ABI",
			SupportsDebugInfo:       false,
			SupportsReleaseOptimize: false,
		}, nil
	case "windows-x64":
		return Target{
			Triple:                  "windows-x64",
			Status:                  StatusSupported,
			OS:                      OSWindows,
			Arch:                    ArchX64,
			ABI:                     ABIWin64,
			DataModel:               DataModelLLP64,
			Format:                  FormatPE,
			ExeExt:                  ".exe",
			CollectImports:          true,
			RunMode:                 RunModeHostNative,
			PointerWidthBits:        64,
			RegisterWidthBits:       64,
			NativeIntWidthBits:      64,
			Endian:                  EndianLittle,
			StackAlignmentBytes:     16,
			MaxAtomicWidthBits:      64,
			SupportsDebugInfo:       true,
			SupportsReleaseOptimize: true,
		}, nil
	case "macos-x64":
		return Target{
			Triple:                  "macos-x64",
			Status:                  StatusSupported,
			OS:                      OSMacOS,
			Arch:                    ArchX64,
			ABI:                     ABISysV,
			DataModel:               DataModelLP64,
			Format:                  FormatMachO,
			ExeExt:                  "",
			CollectImports:          false,
			RunMode:                 RunModeHostNative,
			PointerWidthBits:        64,
			RegisterWidthBits:       64,
			NativeIntWidthBits:      64,
			Endian:                  EndianLittle,
			StackAlignmentBytes:     16,
			MaxAtomicWidthBits:      64,
			SupportsDebugInfo:       true,
			SupportsReleaseOptimize: true,
		}, nil
	case "wasm32-wasi":
		return Target{
			Triple:                  "wasm32-wasi",
			Status:                  StatusSupported,
			OS:                      OSWASI,
			Arch:                    ArchWASM32,
			ABI:                     ABIWASI,
			DataModel:               DataModelILP32,
			Format:                  FormatWASM,
			ExeExt:                  ".wasm",
			CollectImports:          false,
			RunMode:                 RunModeWASIRunner,
			RunRunner:               "wasmtime",
			PointerWidthBits:        32,
			RegisterWidthBits:       32,
			NativeIntWidthBits:      32,
			Endian:                  EndianLittle,
			StackAlignmentBytes:     16,
			MaxAtomicWidthBits:      64,
			SupportsDebugInfo:       false,
			SupportsReleaseOptimize: true,
		}, nil
	case "wasm32-web":
		return Target{
			Triple:                  "wasm32-web",
			Status:                  StatusSupported,
			OS:                      OSWeb,
			Arch:                    ArchWASM32,
			ABI:                     ABIWeb,
			DataModel:               DataModelILP32,
			Format:                  FormatWASM,
			ExeExt:                  ".wasm",
			CollectImports:          false,
			RunMode:                 RunModeWebRunner,
			PointerWidthBits:        32,
			RegisterWidthBits:       32,
			NativeIntWidthBits:      32,
			Endian:                  EndianLittle,
			StackAlignmentBytes:     16,
			MaxAtomicWidthBits:      64,
			SupportsDebugInfo:       false,
			SupportsReleaseOptimize: true,
		}, nil
	default:
		return Target{}, UnsupportedTargetError{Triple: rawTriple}
	}
}

func normalizeAlias(triple string) string {
	switch strings.ToLower(strings.TrimSpace(triple)) {
	case "x86", "i386", "i686", "linux-x86", "linux-i386", "linux-i686",
		"i386-linux-gnu", "i686-linux-gnu", "i686-unknown-linux-gnu", "i686-pc-linux-gnu":
		return "linux-x86"
	case "x64", "amd64", "x86_64", "linux-x64", "linux-amd64", "linux-x86_64",
		"x86_64-linux-gnu", "x86_64-unknown-linux-gnu", "x86_64-pc-linux-gnu", "amd64-linux-gnu":
		return "linux-x64"
	case "win-x64", "windows-amd64", "windows-x86_64",
		"x86_64-pc-windows-msvc", "x86_64-pc-windows-gnu", "amd64-windows-msvc":
		return "windows-x64"
	case "darwin-x64", "macos-amd64", "macos-x86_64",
		"x86_64-apple-darwin", "amd64-apple-darwin":
		return "macos-x64"
	case "x32", "x86_64-x32", "linux-x32", "linux-x86_64-x32",
		"x86_64-linux-gnux32", "x86_64-unknown-linux-gnux32", "x86_64-pc-linux-gnux32", "linux-x86_64-gnux32":
		return "linux-x32"
	default:
		return strings.ToLower(strings.TrimSpace(triple))
	}
}

func (t Target) ScalarLayout(name string) (ScalarLayout, bool) {
	scalar := strings.ToLower(strings.TrimSpace(name))
	switch scalar {
	case "bool", "i8", "u8", "byte":
		return scalarLayout(scalar, 1, 1), true
	case "i16", "u16":
		return scalarLayout(scalar, 2, 2), true
	case "i32", "u32", "int", "uint", "c_int", "c_uint", "f32":
		return scalarLayout(scalar, 4, 4), true
	case "i64", "u64", "f64":
		return scalarLayout(scalar, 8, t.align64()), true
	case "ptr", "ref", "fnptr", "nullable_ptr", "rawptr":
		return scalarLayout(scalar, t.pointerBytes(), t.pointerAlignBytes()), true
	case "usize", "isize", "size_t", "ssize_t", "native_int", "native_uint":
		return scalarLayout(scalar, t.nativeIntBytes(), t.nativeIntAlignBytes()), true
	case "c_long", "c_ulong":
		return scalarLayout(scalar, t.cLongBytes(), t.cLongAlignBytes()), true
	default:
		return ScalarLayout{}, false
	}
}

func (t Target) StructLayout(fields []LayoutField) (AggregateLayout, error) {
	return t.structLayout(fields, false)
}

func (t Target) PackedStructLayout(fields []LayoutField) (AggregateLayout, error) {
	return t.structLayout(fields, true)
}

func (t Target) ArrayLayout(elemType string, count int) (TypeLayout, error) {
	if count < 0 {
		return TypeLayout{}, fmt.Errorf("%s array %s has negative length %d", t.Triple, elemType, count)
	}
	elem, ok := t.namedTypeLayout(elemType)
	if !ok {
		return TypeLayout{}, fmt.Errorf("unknown array element layout type %q for target %s", elemType, t.Triple)
	}
	stride := alignUp(elem.SizeBytes, elem.AlignBytes)
	size64, err := checkedLayoutProduct(stride, count)
	if err != nil {
		return TypeLayout{}, fmt.Errorf("%s array [%d]%s: %w", t.Triple, count, elemType, err)
	}
	if limit, ok := t.maxNativeSizeBytes(); ok && size64 > limit {
		return TypeLayout{}, fmt.Errorf("%s array [%d]%s size %d exceeds %d-bit native size limit %d", t.Triple, count, elemType, size64, t.NativeIntWidthBits, limit)
	}
	if size64 > maxHostInt() {
		return TypeLayout{}, fmt.Errorf("%s array [%d]%s size %d exceeds host layout size limit %d", t.Triple, count, elemType, size64, maxHostInt())
	}
	size := int(size64)
	return TypeLayout{
		Name:       fmt.Sprintf("[%d]%s", count, elemType),
		SizeBytes:  size,
		AlignBytes: elem.AlignBytes,
		ABIBytes:   size,
		ElemType:   elemType,
		Len:        count,
	}, nil
}

func (t Target) SliceLayout(elemType string) (AggregateLayout, error) {
	if _, ok := t.namedTypeLayout(elemType); !ok {
		return AggregateLayout{}, fmt.Errorf("unknown slice element layout type %q for target %s", elemType, t.Triple)
	}
	return t.StructLayout([]LayoutField{
		{Name: "ptr", Type: "ptr"},
		{Name: "len", Type: "i32"},
	})
}

func (t Target) StringLayout() (AggregateLayout, error) {
	return t.SliceLayout("u8")
}

func (t Target) EnumLayout(cases []EnumCaseLayout) (AggregateLayout, error) {
	if len(cases) == 0 {
		return AggregateLayout{}, fmt.Errorf("%s enum layout requires at least one case", t.Triple)
	}
	maxPayloadSize := 0
	maxPayloadAlign := 1
	for _, enumCase := range cases {
		if len(enumCase.Payload) == 0 {
			continue
		}
		payload, err := t.StructLayout(enumCase.Payload)
		if err != nil {
			return AggregateLayout{}, fmt.Errorf("%s enum case %s payload: %w", t.Triple, enumCase.Name, err)
		}
		if payload.SizeBytes > maxPayloadSize {
			maxPayloadSize = payload.SizeBytes
		}
		if payload.AlignBytes > maxPayloadAlign {
			maxPayloadAlign = payload.AlignBytes
		}
	}
	align := maxInt(4, maxPayloadAlign)
	payloadOffset := alignUp(4, maxPayloadAlign)
	size := alignUp(payloadOffset+maxPayloadSize, align)
	return AggregateLayout{
		SizeBytes:          size,
		AlignBytes:         align,
		PayloadOffsetBytes: payloadOffset,
		PayloadSizeBytes:   maxPayloadSize,
		Fields: []FieldLayout{
			{Name: "tag", Type: "i32", OffsetBytes: 0, SizeBytes: 4, AlignBytes: 4, ABIBytes: 4},
			{Name: "payload", Type: "union", OffsetBytes: payloadOffset, SizeBytes: maxPayloadSize, AlignBytes: maxPayloadAlign, ABIBytes: maxPayloadSize},
		},
	}, nil
}

func (t Target) structLayout(fields []LayoutField, packed bool) (AggregateLayout, error) {
	out := AggregateLayout{AlignBytes: 1, Fields: make([]FieldLayout, 0, len(fields))}
	offset := 0
	for _, field := range fields {
		layout, err := t.fieldTypeLayout(field, packed)
		if err != nil {
			return AggregateLayout{}, err
		}
		align := layout.AlignBytes
		if packed || field.Packed {
			align = 1
		}
		offset = alignUp(offset, align)
		out.Fields = append(out.Fields, FieldLayout{
			Name:        field.Name,
			Type:        layout.Name,
			OffsetBytes: offset,
			SizeBytes:   layout.SizeBytes,
			AlignBytes:  align,
			ABIBytes:    layout.ABIBytes,
			Fields:      layout.Fields,
		})
		offset += layout.SizeBytes
		if align > out.AlignBytes {
			out.AlignBytes = align
		}
	}
	out.SizeBytes = alignUp(offset, out.AlignBytes)
	return out, nil
}

func (t Target) fieldTypeLayout(field LayoutField, packed bool) (TypeLayout, error) {
	if len(field.Fields) > 0 {
		layout, err := t.structLayout(field.Fields, packed || field.Packed)
		if err != nil {
			return TypeLayout{}, err
		}
		return TypeLayout{
			Name:       "struct",
			SizeBytes:  layout.SizeBytes,
			AlignBytes: layout.AlignBytes,
			ABIBytes:   layout.SizeBytes,
			Fields:     layout.Fields,
		}, nil
	}
	layout, ok := t.namedTypeLayout(field.Type)
	if !ok {
		return TypeLayout{}, fmt.Errorf("unknown layout type %q for target %s", field.Type, t.Triple)
	}
	return layout, nil
}

func (t Target) namedTypeLayout(name string) (TypeLayout, bool) {
	scalar, ok := t.ScalarLayout(name)
	if ok {
		return TypeLayout{
			Name:       scalar.Name,
			SizeBytes:  scalar.SizeBytes,
			AlignBytes: scalar.AlignBytes,
			ABIBytes:   scalar.ABIBytes,
		}, true
	}
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "str", "string":
		layout, err := t.StringLayout()
		if err != nil {
			return TypeLayout{}, false
		}
		return TypeLayout{
			Name:       "str",
			SizeBytes:  layout.SizeBytes,
			AlignBytes: layout.AlignBytes,
			ABIBytes:   layout.SizeBytes,
			ElemType:   "u8",
			Fields:     layout.Fields,
		}, true
	default:
		return TypeLayout{}, false
	}
}

func (t Target) AtomicWidthBits() []int {
	out := []int{8, 16, 32}
	if t.MaxAtomicWidthBits >= 64 {
		out = append(out, 64)
	}
	return out
}

func (t Target) AtomicLayout(widthBits int) (AtomicLayout, error) {
	if widthBits != 8 && widthBits != 16 && widthBits != 32 && widthBits != 64 {
		return AtomicLayout{}, fmt.Errorf("%s unsupported atomic width %d bits", t.Triple, widthBits)
	}
	if widthBits > t.MaxAtomicWidthBits {
		return AtomicLayout{}, fmt.Errorf("%s unsupported atomic width %d bits (max=%d)", t.Triple, widthBits, t.MaxAtomicWidthBits)
	}
	size := widthBits / 8
	return AtomicLayout{
		WidthBits:         widthBits,
		SizeBytes:         size,
		AlignBytes:        size,
		RegisterWidthBits: t.RegisterWidthBits,
		LockFree:          true,
		PointerSized:      widthBits == t.PointerWidthBits,
	}, nil
}

func (t Target) AtomicPointerLayout() (AtomicLayout, error) {
	if t.PointerWidthBits <= 0 {
		return AtomicLayout{}, fmt.Errorf("%s does not define pointer-sized atomics", t.Triple)
	}
	layout, err := t.AtomicLayout(t.PointerWidthBits)
	if err != nil {
		return AtomicLayout{}, err
	}
	layout.PointerSized = true
	return layout, nil
}

func (t Target) ValidateAtomic(op AtomicOp, widthBits int, alignmentBytes int, order MemoryOrder) error {
	if op == AtomicFence {
		if !validMemoryOrder(order) {
			return fmt.Errorf("%s atomic fence has unsupported memory order %s", t.Triple, order)
		}
		return nil
	}
	layout, err := t.AtomicLayout(widthBits)
	if err != nil {
		return err
	}
	if alignmentBytes <= 0 || alignmentBytes < layout.AlignBytes || alignmentBytes%layout.AlignBytes != 0 {
		return fmt.Errorf("%s misaligned %d-bit atomic: alignment=%d required=%d", t.Triple, widthBits, alignmentBytes, layout.AlignBytes)
	}
	if !atomicOrderAllowed(op, order) {
		return fmt.Errorf("%s atomic %s does not support memory order %s", t.Triple, op, order)
	}
	return nil
}

func scalarLayout(name string, size int, align int) ScalarLayout {
	return ScalarLayout{Name: name, SizeBytes: size, AlignBytes: align, ABIBytes: size}
}

func validMemoryOrder(order MemoryOrder) bool {
	switch order {
	case MemoryOrderRelaxed, MemoryOrderAcquire, MemoryOrderRelease, MemoryOrderAcqRel, MemoryOrderSeqCst:
		return true
	default:
		return false
	}
}

func atomicOrderAllowed(op AtomicOp, order MemoryOrder) bool {
	if !validMemoryOrder(order) {
		return false
	}
	switch op {
	case AtomicLoad:
		return order == MemoryOrderRelaxed || order == MemoryOrderAcquire || order == MemoryOrderSeqCst
	case AtomicStore:
		return order == MemoryOrderRelaxed || order == MemoryOrderRelease || order == MemoryOrderSeqCst
	case AtomicExchange, AtomicCompareExchange, AtomicCompareExchangeWeak,
		AtomicFetchAdd, AtomicFetchSub, AtomicFetchAnd, AtomicFetchOr, AtomicFetchXor:
		return true
	default:
		return false
	}
}

func (t Target) pointerBytes() int {
	if t.PointerWidthBits <= 0 {
		return 0
	}
	return t.PointerWidthBits / 8
}

func (t Target) pointerAlignBytes() int {
	return t.pointerBytes()
}

func (t Target) nativeIntBytes() int {
	if t.NativeIntWidthBits <= 0 {
		return t.pointerBytes()
	}
	return t.NativeIntWidthBits / 8
}

func (t Target) nativeIntAlignBytes() int {
	return t.nativeIntBytes()
}

func (t Target) cLongBytes() int {
	switch t.DataModel {
	case DataModelLP64:
		return 8
	case DataModelILP32, DataModelLLP64, DataModelX32:
		return 4
	default:
		if t.PointerWidthBits == 64 && t.OS != OSWindows {
			return 8
		}
		return 4
	}
}

func (t Target) cLongAlignBytes() int {
	return t.cLongBytes()
}

func (t Target) align64() int {
	if t.Arch == ArchX86 {
		return 4
	}
	return 8
}

func alignUp(value int, align int) int {
	if align <= 1 {
		return value
	}
	remainder := value % align
	if remainder == 0 {
		return value
	}
	return value + align - remainder
}

func checkedLayoutProduct(stride int, count int) (uint64, error) {
	if stride < 0 || count < 0 {
		return 0, fmt.Errorf("negative layout product stride=%d count=%d", stride, count)
	}
	size := uint64(stride) * uint64(count)
	if count != 0 && size/uint64(count) != uint64(stride) {
		return 0, fmt.Errorf("layout size overflows uint64 stride=%d count=%d", stride, count)
	}
	return size, nil
}

func (t Target) maxNativeSizeBytes() (uint64, bool) {
	if t.NativeIntWidthBits <= 0 || t.NativeIntWidthBits >= 64 {
		return 0, false
	}
	return (uint64(1) << uint(t.NativeIntWidthBits)) - 1, true
}

func maxHostInt() uint64 {
	return uint64(^uint(0) >> 1)
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func IsBuildOnlyTarget(triple string) bool {
	for _, buildOnly := range BuildOnlyTriples() {
		if triple == buildOnly {
			return true
		}
	}
	return false
}

func IsPlannedTarget(triple string) bool {
	for _, planned := range PlannedTriples() {
		if triple == planned {
			return true
		}
	}
	return false
}

func Host() (Target, bool) {
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "linux/amd64":
		t, _ := Parse("linux-x64")
		return t, true
	case "windows/amd64":
		t, _ := Parse("windows-x64")
		return t, true
	case "darwin/amd64":
		t, _ := Parse("macos-x64")
		return t, true
	default:
		return Target{}, false
	}
}
