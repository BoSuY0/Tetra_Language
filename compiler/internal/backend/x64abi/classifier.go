package x64abi

import (
	"fmt"
	"strings"

	ctarget "tetra_language/compiler/target"
)

type ABIClass string

const (
	ABIClassUnknown ABIClass = "unknown"
	ABIClassInteger ABIClass = "integer"
	ABIClassSSE     ABIClass = "sse"
	ABIClassMemory  ABIClass = "memory"
)

type ABIExtension string

const (
	ABIExtendNone ABIExtension = "none"
	ABIExtendZero ABIExtension = "zero_extend"
	ABIExtendSign ABIExtension = "sign_extend"
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
	Classes           []ABIClass
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
	Params                  []ABILocation
	Return                  ABILocation
	Variadic                bool
	FixedParamCount         int
	VarargStartIndex        int
	SysVRequiresAL          bool
	SysV_ALSSERegisterCount int
	Win64ShadowSpaceBytes   int
	Win64VarargFloatMirrors []VarargFloatMirror
	VarargRegisterSaveBytes int
	RegisterVarargs         bool
}

type VarargFloatMirror struct {
	ParamIndex  int
	XMMRegister string
	GPRegister  string
}

type Classifier struct {
	target       ctarget.Target
	name         string
	intArgRegs   []string
	floatArgRegs []string
	intRetReg    string
	floatRetReg  string
	stackSlot    int
}

func NewClassifier(tgt ctarget.Target) (Classifier, error) {
	if tgt.Arch != ctarget.ArchX64 {
		return Classifier{}, fmt.Errorf(
			"x64abi classifier requires x64 ISA, got %s for %s",
			tgt.Arch,
			tgt.Triple,
		)
	}
	switch tgt.ABI {
	case ctarget.ABISysV, ctarget.ABIX32SysV:
		return Classifier{
			target:       tgt,
			name:         tgt.ABI.String(),
			intArgRegs:   []string{"rdi", "rsi", "rdx", "rcx", "r8", "r9"},
			floatArgRegs: []string{"xmm0", "xmm1", "xmm2", "xmm3", "xmm4", "xmm5", "xmm6", "xmm7"},
			intRetReg:    "rax",
			floatRetReg:  "xmm0",
			stackSlot:    8,
		}, nil
	case ctarget.ABIWin64:
		return Classifier{
			target:       tgt,
			name:         tgt.ABI.String(),
			intArgRegs:   []string{"rcx", "rdx", "r8", "r9"},
			floatArgRegs: []string{"xmm0", "xmm1", "xmm2", "xmm3"},
			intRetReg:    "rax",
			floatRetReg:  "xmm0",
			stackSlot:    8,
		}, nil
	default:
		return Classifier{}, fmt.Errorf(
			"x64abi classifier does not support ABI %s for %s",
			tgt.ABI,
			tgt.Triple,
		)
	}
}

func (c Classifier) Name() string {
	return c.name
}

func (c Classifier) UsesX64Registers() bool {
	return c.target.RegisterWidthBits == 64
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
		Params:            make([]ABILocation, 0, len(sig.Params)),
	}
	if sig.Variadic {
		plan.Variadic = true
		plan.FixedParamCount = sig.FixedParamCount
		plan.VarargStartIndex = sig.FixedParamCount
		plan.RegisterVarargs = true
		if c.target.ABI == ctarget.ABIWin64 {
			plan.Win64ShadowSpaceBytes = 32
		}
	}
	intRegs := 0
	floatRegs := 0
	stackOffset := 0
	for paramIndex, param := range sig.Params {
		loc, err := c.classifyValue(param)
		if err != nil {
			return ABIPlan{}, err
		}
		if c.target.ABI == ctarget.ABIWin64 && len(loc.Classes) == 0 {
			if paramIndex < len(c.intArgRegs) {
				if loc.Class == ABIClassSSE {
					loc.Register = c.floatArgRegs[paramIndex]
				} else {
					loc.Register = c.intArgRegs[paramIndex]
				}
				loc.Registers = []string{loc.Register}
				c.recordVarargFloatMetadata(&plan, sig, paramIndex, loc)
			} else {
				loc.StackOffsetBytes = stackOffset
				loc.StackSlotBytes = c.stackSlot
				stackOffset += c.stackSlot
			}
			plan.Params = append(plan.Params, loc)
			continue
		}
		if len(loc.Classes) > 0 && loc.Class != ABIClassMemory {
			if !c.assignAggregateArgRegisters(&loc, &intRegs, &floatRegs) {
				loc.StackOffsetBytes = stackOffset
				loc.StackSlotBytes = alignStackSlot(loc.SizeBytes, c.stackSlot)
				stackOffset += loc.StackSlotBytes
			}
			plan.Params = append(plan.Params, loc)
			continue
		}
		switch loc.Class {
		case ABIClassInteger:
			if intRegs < len(c.intArgRegs) {
				loc.Register = c.intArgRegs[intRegs]
				loc.Registers = []string{loc.Register}
				intRegs++
			} else {
				loc.StackOffsetBytes = stackOffset
				loc.StackSlotBytes = c.stackSlot
				stackOffset += c.stackSlot
			}
		case ABIClassSSE:
			if floatRegs < len(c.floatArgRegs) {
				loc.Register = c.floatArgRegs[floatRegs]
				loc.Registers = []string{loc.Register}
				floatRegs++
			} else {
				loc.StackOffsetBytes = stackOffset
				loc.StackSlotBytes = c.stackSlot
				stackOffset += c.stackSlot
			}
		default:
			loc.StackOffsetBytes = stackOffset
			loc.StackSlotBytes = alignStackSlot(loc.SizeBytes, c.stackSlot)
			stackOffset += loc.StackSlotBytes
		}
		if loc.Class == ABIClassSSE && loc.Register != "" {
			c.recordVarargFloatMetadata(&plan, sig, paramIndex, loc)
		}
		plan.Params = append(plan.Params, loc)
	}
	if sig.Return != nil {
		ret, err := c.classifyValue(*sig.Return)
		if err != nil {
			return ABIPlan{}, err
		}
		if len(ret.Classes) > 0 && ret.Class != ABIClassMemory {
			c.assignAggregateReturnRegisters(&ret)
			plan.Return = ret
			return plan, nil
		}
		switch ret.Class {
		case ABIClassInteger:
			ret.Register = c.intRetReg
			ret.Registers = []string{ret.Register}
		case ABIClassSSE:
			ret.Register = c.floatRetReg
			ret.Registers = []string{ret.Register}
		case ABIClassMemory:
			ret.Indirect = true
			ret.Register = c.intArgRegs[0]
			ret.Registers = []string{ret.Register}
		default:
			c.assignAggregateReturnRegisters(&ret)
		}
		plan.Return = ret
	}
	return plan, nil
}

func (c Classifier) recordVarargFloatMetadata(
	plan *ABIPlan,
	sig ABISignature,
	paramIndex int,
	loc ABILocation,
) {
	if !sig.Variadic {
		return
	}
	switch c.target.ABI {
	case ctarget.ABISysV, ctarget.ABIX32SysV:
		if loc.Register != "" && strings.HasPrefix(loc.Register, "xmm") {
			plan.SysVRequiresAL = true
			plan.SysV_ALSSERegisterCount++
			plan.VarargRegisterSaveBytes = 176
		}
	case ctarget.ABIWin64:
		if paramIndex < sig.FixedParamCount || loc.Register == "" ||
			!strings.HasPrefix(loc.Register, "xmm") {
			return
		}
		if gp, ok := c.win64MirrorRegisterForParam(paramIndex); ok {
			plan.Win64VarargFloatMirrors = append(plan.Win64VarargFloatMirrors, VarargFloatMirror{
				ParamIndex:  paramIndex,
				XMMRegister: loc.Register,
				GPRegister:  gp,
			})
		}
	}
}

func (c Classifier) win64MirrorRegisterForParam(paramIndex int) (string, bool) {
	if paramIndex < 0 || paramIndex >= len(c.intArgRegs) {
		return "", false
	}
	return c.intArgRegs[paramIndex], true
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
		class = ABIClassSSE
		registerWidth = 128
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
	loc := ABILocation{
		Name:              param.Name,
		Type:              param.Type,
		SizeBytes:         layout.SizeBytes,
		AlignBytes:        layout.AlignBytes,
		ABIBytes:          layout.SizeBytes,
		RegisterWidthBits: c.target.RegisterWidthBits,
		Extension:         ABIExtendNone,
	}
	if c.usesWin64AggregateRules() {
		if isWin64IntegerAggregateSize(layout.SizeBytes) {
			loc.Class = ABIClassInteger
			loc.ABIBytes = layout.SizeBytes
			return loc, nil
		}
		loc.Class = ABIClassMemory
		loc.Indirect = true
		loc.ABIBytes = c.target.PointerWidthBits / 8
		return loc, nil
	}
	if c.usesSysVAggregateRules() && aggregateHasUnalignedLeaf(c.target, layout.Fields, 0) {
		loc.Class = ABIClassMemory
		return loc, nil
	}
	if layout.SizeBytes > 16 {
		loc.Class = ABIClassMemory
		return loc, nil
	}
	loc.Classes = aggregateEightbyteClasses(layout)
	if len(loc.Classes) == 0 {
		loc.Classes = []ABIClass{ABIClassInteger}
	}
	loc.Class = loc.Classes[0]
	return loc, nil
}

func (c Classifier) usesWin64AggregateRules() bool {
	return c.target.ABI == ctarget.ABIWin64
}

func (c Classifier) usesSysVAggregateRules() bool {
	return c.target.ABI == ctarget.ABISysV || c.target.ABI == ctarget.ABIX32SysV
}

func isWin64IntegerAggregateSize(size int) bool {
	switch size {
	case 1, 2, 4, 8:
		return true
	default:
		return false
	}
}

func aggregateHasUnalignedLeaf(
	tgt ctarget.Target,
	fields []ctarget.FieldLayout,
	baseOffset int,
) bool {
	for _, field := range fields {
		fieldOffset := baseOffset + field.OffsetBytes
		if len(field.Fields) > 0 {
			if aggregateHasUnalignedLeaf(tgt, field.Fields, fieldOffset) {
				return true
			}
			continue
		}
		align := naturalFieldAlign(tgt, field.Type)
		if align > 1 && fieldOffset%align != 0 {
			return true
		}
	}
	return false
}

func naturalFieldAlign(tgt ctarget.Target, typ string) int {
	layout, ok := tgt.ScalarLayout(typ)
	if !ok {
		return 1
	}
	if layout.AlignBytes <= 0 {
		return 1
	}
	return layout.AlignBytes
}

func (c Classifier) assignAggregateArgRegisters(
	loc *ABILocation,
	intRegs *int,
	floatRegs *int,
) bool {
	needInt, needFloat := aggregateRegisterNeeds(loc.Classes)
	if *intRegs+needInt > len(c.intArgRegs) || *floatRegs+needFloat > len(c.floatArgRegs) {
		return false
	}
	for _, class := range loc.Classes {
		switch class {
		case ABIClassSSE:
			loc.Registers = append(loc.Registers, c.floatArgRegs[*floatRegs])
			*floatRegs++
		default:
			loc.Registers = append(loc.Registers, c.intArgRegs[*intRegs])
			*intRegs++
		}
	}
	if len(loc.Registers) > 0 {
		loc.Register = loc.Registers[0]
	}
	return true
}

func (c Classifier) assignAggregateReturnRegisters(loc *ABILocation) {
	intRetRegs := []string{"rax", "rdx"}
	floatRetRegs := []string{"xmm0", "xmm1"}
	intRegs := 0
	floatRegs := 0
	for _, class := range loc.Classes {
		switch class {
		case ABIClassSSE:
			if floatRegs < len(floatRetRegs) {
				loc.Registers = append(loc.Registers, floatRetRegs[floatRegs])
				floatRegs++
			}
		default:
			if intRegs < len(intRetRegs) {
				loc.Registers = append(loc.Registers, intRetRegs[intRegs])
				intRegs++
			}
		}
	}
	if len(loc.Registers) > 0 {
		loc.Register = loc.Registers[0]
	}
}

func aggregateRegisterNeeds(classes []ABIClass) (int, int) {
	intRegs := 0
	floatRegs := 0
	for _, class := range classes {
		if class == ABIClassSSE {
			floatRegs++
		} else {
			intRegs++
		}
	}
	return intRegs, floatRegs
}

func aggregateEightbyteClasses(layout ctarget.AggregateLayout) []ABIClass {
	if layout.SizeBytes <= 0 {
		return nil
	}
	classes := make([]ABIClass, (layout.SizeBytes+7)/8)
	var mark func([]ctarget.FieldLayout, int)
	mark = func(fields []ctarget.FieldLayout, base int) {
		for _, field := range fields {
			fieldBase := base + field.OffsetBytes
			if len(field.Fields) > 0 {
				mark(field.Fields, fieldBase)
				continue
			}
			class := ABIClassInteger
			if isFloatABIType(field.Type) {
				class = ABIClassSSE
			}
			start := fieldBase / 8
			end := (fieldBase + maxInt(field.SizeBytes, 1) - 1) / 8
			for i := start; i <= end && i < len(classes); i++ {
				classes[i] = mergeABIClass(classes[i], class)
			}
		}
	}
	mark(layout.Fields, 0)
	for i, class := range classes {
		if class == ABIClassUnknown {
			classes[i] = ABIClassInteger
		}
	}
	return classes
}

func mergeABIClass(a ABIClass, b ABIClass) ABIClass {
	if a == "" || a == ABIClassUnknown {
		return b
	}
	if a == b {
		return a
	}
	return ABIClassInteger
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

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
