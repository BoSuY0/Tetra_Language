package abisuite

import (
	"fmt"

	ctarget "tetra_language/compiler/target"
)

func CheckX86TargetModel(tgt ctarget.Target) error {
	if tgt.Triple != "linux-x86" || tgt.OS != ctarget.OSLinux || tgt.Arch != ctarget.ArchX86 || tgt.ABI != ctarget.ABI386SysV {
		return fmt.Errorf("x86 identity = triple=%s os=%s arch=%s abi=%s, want linux-x86/linux/x86/i386-sysv", tgt.Triple, tgt.OS, tgt.Arch, tgt.ABI)
	}
	if tgt.DataModel != ctarget.DataModelILP32 || tgt.Format != ctarget.FormatELF || tgt.Endian != ctarget.EndianLittle {
		return fmt.Errorf("x86 platform = model=%s format=%s endian=%s, want ilp32/elf/little", tgt.DataModel, tgt.Format, tgt.Endian)
	}
	if tgt.PointerWidthBits != 32 || tgt.NativeIntWidthBits != 32 || tgt.RegisterWidthBits != 32 || tgt.StackAlignmentBytes != 16 || tgt.MaxAtomicWidthBits != 32 {
		return fmt.Errorf("x86 widths = ptr=%d native=%d reg=%d stack=%d atomic=%d, want 32/32/32/16/32", tgt.PointerWidthBits, tgt.NativeIntWidthBits, tgt.RegisterWidthBits, tgt.StackAlignmentBytes, tgt.MaxAtomicWidthBits)
	}
	for _, scalar := range []struct {
		name  string
		size  int
		align int
	}{
		{name: "ptr", size: 4, align: 4},
		{name: "usize", size: 4, align: 4},
		{name: "c_long", size: 4, align: 4},
		{name: "i64", size: 8, align: 4},
	} {
		if err := expectTargetScalarLayout(tgt, scalar.name, scalar.size, scalar.align); err != nil {
			return err
		}
	}
	if _, err := tgt.AtomicLayout(64); err == nil {
		return fmt.Errorf("x86 accepted 64-bit lock-free atomic without a CPU feature model")
	}
	return nil
}

func CheckX64TargetModel(tgt ctarget.Target) error {
	if tgt.Arch != ctarget.ArchX64 || tgt.PointerWidthBits != 64 || tgt.NativeIntWidthBits != 64 || tgt.RegisterWidthBits != 64 || tgt.StackAlignmentBytes != 16 || tgt.MaxAtomicWidthBits != 64 {
		return fmt.Errorf("x64 widths = arch=%s ptr=%d native=%d reg=%d stack=%d atomic=%d, want x64/64/64/64/16/64", tgt.Arch, tgt.PointerWidthBits, tgt.NativeIntWidthBits, tgt.RegisterWidthBits, tgt.StackAlignmentBytes, tgt.MaxAtomicWidthBits)
	}
	if tgt.Endian != ctarget.EndianLittle {
		return fmt.Errorf("x64 endian = %s, want little", tgt.Endian)
	}
	if err := expectTargetScalarLayout(tgt, "ptr", 8, 8); err != nil {
		return err
	}
	if err := expectTargetScalarLayout(tgt, "usize", 8, 8); err != nil {
		return err
	}
	switch tgt.ABI {
	case ctarget.ABISysV:
		if tgt.DataModel != ctarget.DataModelLP64 || tgt.Format != ctarget.FormatELF && tgt.Format != ctarget.FormatMachO {
			return fmt.Errorf("x64 SysV platform = model=%s format=%s, want lp64/elf-or-macho", tgt.DataModel, tgt.Format)
		}
		if err := expectTargetScalarLayout(tgt, "c_long", 8, 8); err != nil {
			return err
		}
	case ctarget.ABIWin64:
		if tgt.DataModel != ctarget.DataModelLLP64 || tgt.Format != ctarget.FormatPE {
			return fmt.Errorf("x64 Win64 platform = model=%s format=%s, want llp64/pe", tgt.DataModel, tgt.Format)
		}
		if err := expectTargetScalarLayout(tgt, "c_long", 4, 4); err != nil {
			return err
		}
	default:
		return fmt.Errorf("x64 unsupported ABI %s", tgt.ABI)
	}
	return nil
}

func CheckX32TargetModel(tgt ctarget.Target) error {
	if tgt.Triple != "linux-x32" || tgt.OS != ctarget.OSLinux || tgt.Arch != ctarget.ArchX64 || tgt.ABI != ctarget.ABIX32SysV {
		return fmt.Errorf("x32 identity = triple=%s os=%s arch=%s abi=%s, want linux-x32/linux/x64/x32-sysv", tgt.Triple, tgt.OS, tgt.Arch, tgt.ABI)
	}
	if tgt.DataModel != ctarget.DataModelX32 || tgt.Format != ctarget.FormatELF || tgt.Endian != ctarget.EndianLittle {
		return fmt.Errorf("x32 platform = model=%s format=%s endian=%s, want x32/elf/little", tgt.DataModel, tgt.Format, tgt.Endian)
	}
	if tgt.PointerWidthBits != 32 || tgt.NativeIntWidthBits != 32 || tgt.RegisterWidthBits != 64 || tgt.StackAlignmentBytes != 16 || tgt.MaxAtomicWidthBits != 64 {
		return fmt.Errorf("x32 widths = ptr=%d native=%d reg=%d stack=%d atomic=%d, want 32/32/64/16/64", tgt.PointerWidthBits, tgt.NativeIntWidthBits, tgt.RegisterWidthBits, tgt.StackAlignmentBytes, tgt.MaxAtomicWidthBits)
	}
	if err := expectTargetScalarLayout(tgt, "ptr", 4, 4); err != nil {
		return err
	}
	if err := expectTargetScalarLayout(tgt, "usize", 4, 4); err != nil {
		return err
	}
	if err := expectTargetScalarLayout(tgt, "isize", 4, 4); err != nil {
		return err
	}
	if err := expectTargetScalarLayout(tgt, "size_t", 4, 4); err != nil {
		return err
	}
	if err := expectTargetScalarLayout(tgt, "i64", 8, 8); err != nil {
		return err
	}
	x86, err := ctarget.Parse("x86")
	if err != nil {
		return err
	}
	if x86.Arch == tgt.Arch || x86.RegisterWidthBits == tgt.RegisterWidthBits || x86.MaxAtomicWidthBits == tgt.MaxAtomicWidthBits {
		return fmt.Errorf("x32 collapsed into x86: x86 arch=%s reg=%d atomic=%d, x32 arch=%s reg=%d atomic=%d", x86.Arch, x86.RegisterWidthBits, x86.MaxAtomicWidthBits, tgt.Arch, tgt.RegisterWidthBits, tgt.MaxAtomicWidthBits)
	}
	x64, err := ctarget.Parse("x64")
	if err != nil {
		return err
	}
	if x64.PointerWidthBits == tgt.PointerWidthBits || x64.NativeIntWidthBits == tgt.NativeIntWidthBits || x64.ABI == tgt.ABI {
		return fmt.Errorf("x32 collapsed into x64: x64 ptr=%d native=%d abi=%s, x32 ptr=%d native=%d abi=%s", x64.PointerWidthBits, x64.NativeIntWidthBits, x64.ABI, tgt.PointerWidthBits, tgt.NativeIntWidthBits, tgt.ABI)
	}
	return nil
}

func ExpectTargetScalarLayout(tgt ctarget.Target, name string, size int, align int) error {
	return expectTargetScalarLayout(tgt, name, size, align)
}

func expectTargetScalarLayout(tgt ctarget.Target, name string, size int, align int) error {
	layout, ok := tgt.ScalarLayout(name)
	if !ok {
		return fmt.Errorf("%s missing scalar layout %s", tgt.Triple, name)
	}
	if layout.SizeBytes != size || layout.AlignBytes != align || layout.ABIBytes != size {
		return fmt.Errorf("%s scalar %s layout = size=%d align=%d abi=%d, want %d/%d/%d", tgt.Triple, name, layout.SizeBytes, layout.AlignBytes, layout.ABIBytes, size, align, size)
	}
	return nil
}
