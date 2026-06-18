package x86abi

import (
	"fmt"
	"strings"

	ctarget "tetra_language/compiler/target"
)

type ABIClass string

const (
	ABIClassUnknown ABIClass = "unknown"
	ABIClassInteger ABIClass = "integer"
	ABIClassX87     ABIClass = "x87"
	ABIClassMemory  ABIClass = "memory"
)

type ABIExtension string

const (
	ABIExtendNone ABIExtension = "none"
	ABIExtendZero ABIExtension = "zero_extend"
	ABIExtendSign ABIExtension = "sign_extend"
)

type StackCleanup string

const (
	StackCleanupCaller StackCleanup = "caller"
	StackCleanupCallee StackCleanup = "callee"
)

type ABIParam struct {
	Name   string
	Type   string
	Fields []ctarget.LayoutField
	Packed bool
}

type ABISignature struct {
	Params          []ABIParam
	Return          *ABIParam
	Variadic        bool
	FixedParamCount int
}

type ABILocation struct {
	Name              string
	Type              string
	Class             ABIClass
	Register          string
	Registers         []string
	StackOffsetBytes  int
	StackSlotBytes    int
	SizeBytes         int
	AlignBytes        int
	ABIBytes          int
	RegisterWidthBits int
	Extension         ABIExtension
	Indirect          bool
}

type ABIPlan struct {
	Name                    string
	PointerWidthBits        int
	RegisterWidthBits       int
	StackCleanup            StackCleanup
	Params                  []ABILocation
	Return                  ABILocation
	Variadic                bool
	FixedParamCount         int
	VarargStartIndex        int
	RegisterVarargs         bool
	VarargRegisterSaveBytes int
}

type Classifier struct {
	target    ctarget.Target
	name      string
	stackSlot int
}

func NewClassifier(tgt ctarget.Target) (Classifier, error) {
	if tgt.Arch != ctarget.ArchX86 || tgt.ABI != ctarget.ABI386SysV {
		return Classifier{}, fmt.Errorf(
			"x86abi classifier requires x86 i386-sysv, got arch=%s abi=%s for %s",
			tgt.Arch,
			tgt.ABI,
			tgt.Triple,
		)
	}
	return Classifier{
		target:    tgt,
		name:      tgt.ABI.String(),
		stackSlot: 4,
	}, nil
}

func (c Classifier) Name() string {
	return c.name
}

func (c Classifier) StackCleanup() StackCleanup {
	return StackCleanupCaller
}

func (c Classifier) ClassifySignature(sig ABISignature) (ABIPlan, error) {
	if err := validateVariadicSignature(
		sig.Variadic,
		sig.FixedParamCount,
		len(sig.Params),
	); err != nil {
		return ABIPlan{}, err
	}
	plan := ABIPlan{
		Name:              c.name,
		PointerWidthBits:  c.target.PointerWidthBits,
		RegisterWidthBits: c.target.RegisterWidthBits,
		StackCleanup:      c.StackCleanup(),
		Params:            make([]ABILocation, 0, len(sig.Params)),
	}
	if sig.Variadic {
		plan.Variadic = true
		plan.FixedParamCount = sig.FixedParamCount
		plan.VarargStartIndex = sig.FixedParamCount
	}
	stackOffset := 0
	if sig.Return != nil {
		ret, err := c.classifyValue(*sig.Return)
		if err != nil {
			return ABIPlan{}, err
		}
		c.assignReturn(&ret)
		plan.Return = ret
		if ret.Indirect {
			stackOffset += c.stackSlot
		}
	}
	for _, param := range sig.Params {
		loc, err := c.classifyValue(param)
		if err != nil {
			return ABIPlan{}, err
		}
		loc.StackOffsetBytes = stackOffset
		loc.StackSlotBytes = alignStackSlot(loc.SizeBytes, c.stackSlot)
		stackOffset += loc.StackSlotBytes
		plan.Params = append(plan.Params, loc)
	}
	return plan, nil
}

func (c Classifier) classifyValue(param ABIParam) (ABILocation, error) {
	if len(param.Fields) > 0 {
		return c.classifyAggregate(param)
	}
	layout, ok := c.target.ScalarLayout(param.Type)
	if !ok {
		return ABILocation{}, fmt.Errorf(
			"%s cannot classify ABI type %q",
			c.target.Triple,
			param.Type,
		)
	}
	class := ABIClassInteger
	registerWidth := c.target.RegisterWidthBits
	extension := abiExtensionFor(param.Type, layout.SizeBytes*8, registerWidth)
	if isFloatABIType(param.Type) {
		class = ABIClassX87
		registerWidth = 80
		extension = ABIExtendNone
	}
	return ABILocation{
		Name:              param.Name,
		Type:              param.Type,
		Class:             class,
		SizeBytes:         layout.SizeBytes,
		AlignBytes:        layout.AlignBytes,
		ABIBytes:          layout.ABIBytes,
		RegisterWidthBits: registerWidth,
		Extension:         extension,
	}, nil
}

func (c Classifier) classifyAggregate(param ABIParam) (ABILocation, error) {
	var layout ctarget.AggregateLayout
	var err error
	if param.Packed {
		layout, err = c.target.PackedStructLayout(param.Fields)
	} else {
		layout, err = c.target.StructLayout(param.Fields)
	}
	if err != nil {
		return ABILocation{}, fmt.Errorf(
			"%s cannot classify ABI aggregate %q: %w",
			c.target.Triple,
			param.Type,
			err,
		)
	}
	return ABILocation{
		Name:              param.Name,
		Type:              param.Type,
		Class:             ABIClassMemory,
		SizeBytes:         layout.SizeBytes,
		AlignBytes:        layout.AlignBytes,
		ABIBytes:          layout.SizeBytes,
		RegisterWidthBits: c.target.RegisterWidthBits,
		Extension:         ABIExtendNone,
	}, nil
}

func (c Classifier) assignReturn(loc *ABILocation) {
	switch loc.Class {
	case ABIClassMemory:
		loc.Indirect = true
		loc.Register = "sret@stack+0"
		loc.Registers = []string{loc.Register}
		loc.StackOffsetBytes = 0
		loc.StackSlotBytes = c.stackSlot
	case ABIClassX87:
		loc.Register = "st0"
		loc.Registers = []string{loc.Register}
	default:
		if loc.SizeBytes > 4 {
			loc.Register = "edx:eax"
			loc.Registers = []string{"eax", "edx"}
		} else {
			loc.Register = "eax"
			loc.Registers = []string{"eax"}
		}
	}
}

func isFloatABIType(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "f32", "f64":
		return true
	default:
		return false
	}
}

func abiExtensionFor(name string, widthBits int, registerWidthBits int) ABIExtension {
	if widthBits >= registerWidthBits {
		return ABIExtendNone
	}
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "i8", "i16", "i32", "isize", "c_long", "ssize_t", "native_int":
		return ABIExtendSign
	default:
		return ABIExtendZero
	}
}

func alignStackSlot(size int, slot int) int {
	if size <= slot {
		return slot
	}
	remainder := size % slot
	if remainder == 0 {
		return size
	}
	return size + slot - remainder
}

func validateVariadicSignature(variadic bool, fixedParamCount int, paramCount int) error {
	if !variadic {
		return nil
	}
	if fixedParamCount < 0 || fixedParamCount > paramCount {
		return fmt.Errorf(
			"invalid variadic fixed parameter count %d for %d parameters",
			fixedParamCount,
			paramCount,
		)
	}
	return nil
}
