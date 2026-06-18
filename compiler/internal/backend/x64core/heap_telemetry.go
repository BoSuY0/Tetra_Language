package x64core

import (
	"bytes"
	"fmt"
	"path"
	"strconv"
	"strings"

	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/backend/x64abi"
	"tetra_language/compiler/internal/backend/x64obj"
)

const (
	runtimeHeapTelemetrySchema = "tetra.runtime.heap_telemetry.v1"
	runtimeHeapTelemetryTarget = "linux-x64"
	runtimeHeapTelemetryMethod = "tetra_linux_x64_heap_telemetry_v1"

	runtimeHeapTelemetryNumberWidth = 20
	runtimeHeapTelemetryATFDCWD     = 0xffffff9c
	runtimeHeapTelemetryOpenAt      = 257
	runtimeHeapTelemetryClose       = 3
	runtimeHeapTelemetryOpenFlags   = 0x241
	runtimeHeapTelemetryOpenMode    = 0o644
)

const (
	runtimeHeapTelemetryCurrentOffset int32 = iota * 8
	runtimeHeapTelemetryPeakOffset
	runtimeHeapTelemetryTotalOffset
	runtimeHeapTelemetryCountOffset
	runtimeHeapTelemetryRequestedOffset
	runtimeHeapTelemetryReservedOffset
	runtimeHeapTelemetrySmallPathCountOffset
)

type runtimeHeapTelemetryState struct {
	dataIndex int
	layout    runtimeHeapTelemetryLayout
}

type runtimeHeapTelemetryLayout struct {
	pathOffset     int32
	zeroJSONOffset int32
	zeroJSONLength uint32
	pathJSONOffset int32
	pathJSONLength uint32
	zeroNumbers    runtimeHeapTelemetryNumberOffsets
	pathNumbers    runtimeHeapTelemetryNumberOffsets
}

type runtimeHeapTelemetryNumberOffsets struct {
	current         int32
	peak            int32
	total           int32
	count           int32
	requested       int32
	reserved        int32
	domainRequested int32
	domainReserved  int32
	domainCurrent   int32
	domainPeak      int32
	smallPath       int32
}

type runtimeHeapTelemetryNumberField struct {
	counterOffset int32
	fieldOffset   int32
}

type runtimeHeapTelemetryFlushFunc func() error

func (f runtimeHeapTelemetryFlushFunc) emit() error {
	if f == nil {
		return nil
	}
	return f()
}

func buildRuntimeHeapTelemetryBlob(opt x64.CodegenOptions) ([]byte, runtimeHeapTelemetryLayout, error) {
	program := strings.TrimSpace(opt.RuntimeHeapTelemetryProgram)
	if program == "" {
		return nil, runtimeHeapTelemetryLayout{}, fmt.Errorf("runtime heap telemetry program is required")
	}
	dir := strings.TrimSpace(opt.RuntimeHeapTelemetryDir)
	if dir == "" {
		return nil, runtimeHeapTelemetryLayout{}, fmt.Errorf("runtime heap telemetry dir is required")
	}
	sidecarPath := path.Join(strings.ReplaceAll(dir, "\\", "/"), program+".heap.json")

	zeroJSON, zeroNumbers := runtimeHeapTelemetryJSON(program, false)
	pathJSON, pathNumbers := runtimeHeapTelemetryJSON(program, true)

	blob := make([]byte, 0, 64+len(sidecarPath)+1+len(zeroJSON)+len(pathJSON))
	blob = append(blob, make([]byte, int(runtimeHeapTelemetrySmallPathCountOffset)+8)...)
	layout := runtimeHeapTelemetryLayout{
		pathOffset: int32(len(blob)),
	}
	blob = append(blob, []byte(sidecarPath)...)
	blob = append(blob, 0)

	layout.zeroJSONOffset = int32(len(blob))
	layout.zeroJSONLength = uint32(len(zeroJSON))
	layout.zeroNumbers = zeroNumbers
	blob = append(blob, zeroJSON...)

	layout.pathJSONOffset = int32(len(blob))
	layout.pathJSONLength = uint32(len(pathJSON))
	layout.pathNumbers = pathNumbers
	blob = append(blob, pathJSON...)

	return blob, layout, nil
}

func runtimeHeapTelemetryJSON(program string, includePaths bool) ([]byte, runtimeHeapTelemetryNumberOffsets) {
	var b bytes.Buffer
	numbers := runtimeHeapTelemetryNumberOffsets{smallPath: -1}
	placeholder := strings.Repeat(" ", runtimeHeapTelemetryNumberWidth)

	writeString := func(name string, value string, comma bool) {
		fmt.Fprintf(&b, "  %s: %s", strconv.Quote(name), strconv.Quote(value))
		if comma {
			b.WriteByte(',')
		}
		b.WriteByte('\n')
	}
	writeNumber := func(name string, comma bool) int32 {
		fmt.Fprintf(&b, "  %s: ", strconv.Quote(name))
		off := int32(b.Len())
		b.WriteString(placeholder)
		if comma {
			b.WriteByte(',')
		}
		b.WriteByte('\n')
		return off
	}
	writeIndentedNumber := func(indent string, name string, comma bool) int32 {
		fmt.Fprintf(&b, "%s%s: ", indent, strconv.Quote(name))
		off := int32(b.Len())
		b.WriteString(placeholder)
		if comma {
			b.WriteByte(',')
		}
		b.WriteByte('\n')
		return off
	}

	b.WriteString("{\n")
	writeString("schema", runtimeHeapTelemetrySchema, true)
	writeString("target", runtimeHeapTelemetryTarget, true)
	writeString("method", runtimeHeapTelemetryMethod, true)
	writeString("program", program, true)
	numbers.current = writeNumber("heap_current_bytes", true)
	numbers.peak = writeNumber("heap_peak_bytes", true)
	numbers.total = writeNumber("heap_total_alloc_bytes", true)
	numbers.count = writeNumber("heap_allocation_count", true)
	numbers.requested = writeNumber("bytes_requested", true)
	numbers.reserved = writeNumber("bytes_reserved", true)
	b.WriteString("  \"exit_status\": 0,\n")
	if includePaths {
		b.WriteString("  \"allocation_paths\": {\n")
		numbers.smallPath = writeIndentedNumber("    ", "small_heap_make_slice", false)
		b.WriteString("  },\n")
	}
	b.WriteString("  \"domain_bytes\": [\n")
	b.WriteString("    {\n")
	b.WriteString("      \"domain_id\": \"process\",\n")
	b.WriteString("      \"kind\": \"process\",\n")
	numbers.domainRequested = writeIndentedNumber("      ", "requested_bytes", true)
	numbers.domainReserved = writeIndentedNumber("      ", "reserved_bytes", true)
	numbers.domainCurrent = writeIndentedNumber("      ", "current_bytes", true)
	numbers.domainPeak = writeIndentedNumber("      ", "peak_bytes", false)
	b.WriteString("    }\n")
	b.WriteString("  ],\n")
	b.WriteString("  \"notes\": [\"bytes_reserved is 0 because this sidecar counts Tetra heap allocation requests, not OS mmap reservations\"]\n")
	b.WriteString("}\n")
	return b.Bytes(), numbers
}

func (o runtimeHeapTelemetryNumberOffsets) fields() []runtimeHeapTelemetryNumberField {
	fields := []runtimeHeapTelemetryNumberField{
		{counterOffset: runtimeHeapTelemetryCurrentOffset, fieldOffset: o.current},
		{counterOffset: runtimeHeapTelemetryPeakOffset, fieldOffset: o.peak},
		{counterOffset: runtimeHeapTelemetryTotalOffset, fieldOffset: o.total},
		{counterOffset: runtimeHeapTelemetryCountOffset, fieldOffset: o.count},
		{counterOffset: runtimeHeapTelemetryRequestedOffset, fieldOffset: o.requested},
		{counterOffset: runtimeHeapTelemetryReservedOffset, fieldOffset: o.reserved},
		{counterOffset: runtimeHeapTelemetryRequestedOffset, fieldOffset: o.domainRequested},
		{counterOffset: runtimeHeapTelemetryReservedOffset, fieldOffset: o.domainReserved},
		{counterOffset: runtimeHeapTelemetryCurrentOffset, fieldOffset: o.domainCurrent},
		{counterOffset: runtimeHeapTelemetryPeakOffset, fieldOffset: o.domainPeak},
	}
	if o.smallPath >= 0 {
		fields = append(fields, runtimeHeapTelemetryNumberField{
			counterOffset: runtimeHeapTelemetrySmallPathCountOffset,
			fieldOffset:   o.smallPath,
		})
	}
	return fields
}

func emitRuntimeHeapTelemetryRecordAllocation(e *x64.Emitter, leaPatches *[]x64obj.LeaPatch, state *runtimeHeapTelemetryState) error {
	if state == nil {
		return nil
	}
	if leaPatches == nil {
		return fmt.Errorf("runtime heap telemetry: missing data patches")
	}
	e.PushRax()
	e.PushRdi()
	e.PushRdx()
	e.PushR8()
	e.PushRsi()

	emitRuntimeHeapTelemetryLoadBase(e, leaPatches, state)
	e.MovRdiRdx()

	e.MovRaxFromRdiDisp(runtimeHeapTelemetryCurrentOffset)
	e.AddRaxRsi()
	e.MovMem64RdiDispRax(runtimeHeapTelemetryCurrentOffset)
	e.MovRdxFromRdiDisp(runtimeHeapTelemetryPeakOffset)
	e.CmpRdxRax()
	keepPeakAt := e.JaeRel32()
	e.MovMem64RdiDispRax(runtimeHeapTelemetryPeakOffset)
	keepPeakOff := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, keepPeakAt, keepPeakOff); err != nil {
		return err
	}

	e.MovRaxFromRdiDisp(runtimeHeapTelemetryTotalOffset)
	e.AddRaxRsi()
	e.MovMem64RdiDispRax(runtimeHeapTelemetryTotalOffset)
	e.MovRaxFromRdiDisp(runtimeHeapTelemetryRequestedOffset)
	e.AddRaxRsi()
	e.MovMem64RdiDispRax(runtimeHeapTelemetryRequestedOffset)
	e.MovRaxFromRdiDisp(runtimeHeapTelemetryCountOffset)
	e.AddRaxImm32(1)
	e.MovMem64RdiDispRax(runtimeHeapTelemetryCountOffset)
	e.MovRaxFromRdiDisp(runtimeHeapTelemetrySmallPathCountOffset)
	e.AddRaxImm32(1)
	e.MovMem64RdiDispRax(runtimeHeapTelemetrySmallPathCountOffset)

	e.PopRsi()
	e.PopR8()
	e.PopRdx()
	e.PopRdi()
	e.PopRax()
	return nil
}

func emitRuntimeHeapTelemetryFlush(e *x64.Emitter, abi x64abi.ABI, leaPatches *[]x64obj.LeaPatch, state *runtimeHeapTelemetryState) error {
	if state == nil {
		return nil
	}
	if leaPatches == nil {
		return fmt.Errorf("runtime heap telemetry: missing data patches")
	}
	sysv, ok := abi.(*x64abi.SysVUnix)
	if !ok || sysv.SysExit != 60 || sysv.SysWrite != 1 {
		return fmt.Errorf("runtime heap telemetry requires linux-x64 SysV ABI")
	}

	e.PushRax()
	if err := emitRuntimeHeapTelemetryFillTemplate(e, leaPatches, state, state.layout.zeroJSONOffset, state.layout.zeroNumbers); err != nil {
		return err
	}
	if err := emitRuntimeHeapTelemetryFillTemplate(e, leaPatches, state, state.layout.pathJSONOffset, state.layout.pathNumbers); err != nil {
		return err
	}

	emitRuntimeHeapTelemetryLoadBase(e, leaPatches, state)
	e.MovRdiRdx()
	e.MovRaxFromRdiDisp(runtimeHeapTelemetryCountOffset)
	e.TestRaxRax()
	useZeroAt := e.JzRel32()

	emitRuntimeHeapTelemetryLoadBase(e, leaPatches, state)
	e.AddRdxImm32(state.layout.pathJSONOffset)
	e.MovR8Rdx()
	e.MovR9dImm32(state.layout.pathJSONLength)
	selectedAt := e.JmpRel32()

	zeroOff := len(e.Buf)
	emitRuntimeHeapTelemetryLoadBase(e, leaPatches, state)
	e.AddRdxImm32(state.layout.zeroJSONOffset)
	e.MovR8Rdx()
	e.MovR9dImm32(state.layout.zeroJSONLength)

	selectedOff := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, useZeroAt, zeroOff); err != nil {
		return err
	}
	if err := x64.PatchRel32(e.Buf, selectedAt, selectedOff); err != nil {
		return err
	}

	emitRuntimeHeapTelemetryLoadBase(e, leaPatches, state)
	e.AddRdxImm32(state.layout.pathOffset)
	e.MovRsiRdx()
	e.MovEdiImm32(runtimeHeapTelemetryATFDCWD)
	e.MovEdxImm32(runtimeHeapTelemetryOpenFlags)
	e.MovR10dImm32(runtimeHeapTelemetryOpenMode)
	e.MovEaxImm32(runtimeHeapTelemetryOpenAt)
	e.Syscall()
	e.CmpRaxImm32(-4095)
	openFailedAt := e.JaeRel32()

	e.PushRax()
	e.MovRdiRax()
	e.MovRsiR8()
	e.MovRdxR9()
	e.MovEaxImm32(sysv.SysWrite)
	e.Syscall()
	e.PopRdi()
	e.MovEaxImm32(runtimeHeapTelemetryClose)
	e.Syscall()

	doneOff := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, openFailedAt, doneOff); err != nil {
		return err
	}
	e.PopRax()
	return nil
}

func emitRuntimeHeapTelemetryFillTemplate(e *x64.Emitter, leaPatches *[]x64obj.LeaPatch, state *runtimeHeapTelemetryState, jsonOffset int32, numbers runtimeHeapTelemetryNumberOffsets) error {
	for _, field := range numbers.fields() {
		emitRuntimeHeapTelemetryLoadBase(e, leaPatches, state)
		e.MovRdiRdx()
		e.MovRaxFromRdiDisp(field.counterOffset)
		e.AddRdiImm32(jsonOffset + field.fieldOffset)
		if err := emitRuntimeHeapTelemetryWriteDecimal(e); err != nil {
			return err
		}
	}
	return nil
}

func emitRuntimeHeapTelemetryLoadBase(e *x64.Emitter, leaPatches *[]x64obj.LeaPatch, state *runtimeHeapTelemetryState) {
	leaPos := e.LeaRdxRipDisp()
	*leaPatches = append(*leaPatches, x64obj.LeaPatch{At: leaPos, DataIndex: state.dataIndex})
}

func emitRuntimeHeapTelemetryWriteDecimal(e *x64.Emitter) error {
	e.MovEcxImm32(runtimeHeapTelemetryNumberWidth)
	fillOff := len(e.Buf)
	e.MovMem8RdiRcxMinus1Imm8(' ')
	e.DecEcx()
	e.TestEcxEcx()
	fillAgainAt := e.JnzRel32()
	if err := x64.PatchRel32(e.Buf, fillAgainAt, fillOff); err != nil {
		return err
	}

	e.MovEcxImm32(runtimeHeapTelemetryNumberWidth)
	e.MovR9dImm32(10)
	e.TestRaxRax()
	nonZeroAt := e.JnzRel32()
	e.MovEdxImm32('0')
	e.MovMem8RdiDispDl(runtimeHeapTelemetryNumberWidth - 1)
	doneAt := e.JmpRel32()

	loopOff := len(e.Buf)
	if err := x64.PatchRel32(e.Buf, nonZeroAt, loopOff); err != nil {
		return err
	}
	e.XorEdxEdx()
	e.DivR9()
	e.AddDlImm8('0')
	e.DecEcx()
	e.MovMem8RdiRcxDl()
	e.TestRaxRax()
	loopAgainAt := e.JnzRel32()
	if err := x64.PatchRel32(e.Buf, loopAgainAt, loopOff); err != nil {
		return err
	}

	doneOff := len(e.Buf)
	return x64.PatchRel32(e.Buf, doneAt, doneOff)
}
