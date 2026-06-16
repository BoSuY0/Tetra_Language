package abisuite

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

type stdoutExecutableSmokeOptions struct {
	target      string
	stem        string
	label       string
	wantClass   byte
	wantMachine uint16
	wantLiteral string
	wantCode    [][]byte
	forbidCode  []byte
}

type stderrFDRuntimeSmokeOptions struct {
	target      string
	stem        string
	label       string
	wantClass   byte
	wantMachine uint16
	wantCode    [][]byte
	forbidCode  []byte
}

type allocatorExecutableSmokeOptions struct {
	target      string
	stem        string
	label       string
	wantClass   byte
	wantMachine uint16
	wantCode    [][]byte
	forbidCode  [][]byte
}

type allocatorFailureExecutableSmokeOptions struct {
	target      string
	stem        string
	label       string
	wantClass   byte
	wantMachine uint16
	wantCode    [][]byte
	forbidCode  [][]byte
}

type rawMemoryBoundsExecutableSmokeOptions struct {
	target      string
	stem        string
	label       string
	wantClass   byte
	wantMachine uint16
	wantCode    [][]byte
	forbidCode  [][]byte
}

type rawPointerSlotExecutableSmokeOptions struct {
	target      string
	stem        string
	label       string
	wantClass   byte
	wantMachine uint16
	wantCode    [][]byte
	forbidCode  [][]byte
}

type networkingLifecycleRuntimeSmokeOptions struct {
	target      string
	stem        string
	label       string
	wantClass   byte
	wantMachine uint16
	wantCode    [][]byte
	forbidCode  []byte
}

type islandFreeExecutableSmokeOptions struct {
	target          string
	stem            string
	label           string
	wantClass       byte
	wantMachine     uint16
	wantCode        [][]byte
	wantDebugCode   [][]byte
	forbidCode      [][]byte
	forbidDebugCode [][]byte
}

func CheckX86StdoutExecutableSmoke(deps RuntimeSmokeDeps) error {
	return checkStdoutExecutableSmoke(stdoutExecutableSmokeOptions{
		target:      "linux-x86",
		stem:        "x86_stdout",
		label:       "x86 stdout executable",
		wantClass:   1,
		wantMachine: 0x03,
		wantLiteral: "x86 stdout\n",
		wantCode: [][]byte{
			{0xB8, 0x04, 0x00, 0x00, 0x00},
			{0xCD, 0x80},
		},
		forbidCode: []byte{0x0F, 0x05},
	}, deps)
}

func CheckX32StdoutExecutableSmoke(deps RuntimeSmokeDeps) error {
	return checkStdoutExecutableSmoke(stdoutExecutableSmokeOptions{
		target:      "linux-x32",
		stem:        "x32_stdout",
		label:       "x32 stdout executable",
		wantClass:   1,
		wantMachine: 0x3e,
		wantLiteral: "x32 stdout\n",
		wantCode: [][]byte{
			{0xB8, 0x01, 0x00, 0x00, 0x40},
			{0x0F, 0x05},
		},
		forbidCode: []byte{0xCD, 0x80},
	}, deps)
}

func CheckX86StderrFDRuntimeSmoke(deps RuntimeSmokeDeps) error {
	return checkStderrFDRuntimeSmoke(stderrFDRuntimeSmokeOptions{
		target:      "linux-x86",
		stem:        "x86_stderr_fd",
		label:       "x86 stderr fd runtime",
		wantClass:   1,
		wantMachine: 0x03,
		wantCode: [][]byte{
			{0xB8, 0x02, 0x00, 0x00, 0x00, 0x50},
			{0x8B, 0x5D, 0x08, 0x8B, 0x4D, 0x0C, 0x03, 0x4D, 0x14},
			{0xB8, 0x04, 0x00, 0x00, 0x00, 0xCD, 0x80},
		},
		forbidCode: []byte{0x0F, 0x05},
	}, deps)
}

func CheckX32StderrFDRuntimeSmoke(deps RuntimeSmokeDeps) error {
	return checkStderrFDRuntimeSmoke(stderrFDRuntimeSmokeOptions{
		target:      "linux-x32",
		stem:        "x32_stderr_fd",
		label:       "x32 stderr fd runtime",
		wantClass:   1,
		wantMachine: 0x3e,
		wantCode: [][]byte{
			{0xB8, 0x02, 0x00, 0x00, 0x00, 0x50},
			{0x48, 0x63, 0xC9, 0x48, 0x01, 0xCE, 0x4C, 0x89, 0xC2},
			{0xB8, 0x01, 0x00, 0x00, 0x40, 0x0F, 0x05},
		},
		forbidCode: []byte{0xCD, 0x80},
	}, deps)
}

func CheckX86AllocatorExecutableSmoke(deps RuntimeSmokeDeps) error {
	return checkAllocatorExecutableSmoke(allocatorExecutableSmokeOptions{
		target:      "linux-x86",
		stem:        "x86_allocator",
		label:       "x86 allocator executable",
		wantClass:   1,
		wantMachine: 0x03,
		wantCode: [][]byte{
			{0xB8, 0xC0, 0x00, 0x00, 0x00, 0xCD, 0x80},
			{0x3D, 0x01, 0xF0, 0xFF, 0xFF, 0x0F, 0x83},
			{0xBB, 0x02, 0x00, 0x00, 0x00, 0xB8, 0x01, 0x00, 0x00, 0x00, 0xCD, 0x80},
			{0x89, 0x08, 0x83, 0xC0, 0x08},
		},
		forbidCode: [][]byte{{0x0F, 0x05}},
	}, deps)
}

func CheckX86AllocatorFailureExecutableSmoke(deps RuntimeSmokeDeps) error {
	return checkAllocatorFailureExecutableSmoke(allocatorFailureExecutableSmokeOptions{
		target:      "linux-x86",
		stem:        "x86_allocator_failure",
		label:       "x86 allocator failure executable",
		wantClass:   1,
		wantMachine: 0x03,
		wantCode: [][]byte{
			{0x83, 0xF9, 0x01, 0x0F, 0x8D},
			{0xBB, 0x02, 0x00, 0x00, 0x00, 0xB8, 0x01, 0x00, 0x00, 0x00, 0xCD, 0x80},
			{0xB8, 0xC0, 0x00, 0x00, 0x00, 0xCD, 0x80},
		},
		forbidCode: [][]byte{{0x0F, 0x05}},
	}, deps)
}

func CheckX32AllocatorExecutableSmoke(deps RuntimeSmokeDeps) error {
	return checkAllocatorExecutableSmoke(allocatorExecutableSmokeOptions{
		target:      "linux-x32",
		stem:        "x32_allocator",
		label:       "x32 allocator executable",
		wantClass:   1,
		wantMachine: 0x3e,
		wantCode: [][]byte{
			{0xB8, 0x09, 0x00, 0x00, 0x40, 0x0F, 0x05},
			{0x48, 0x3D, 0x01, 0xF0, 0xFF, 0xFF, 0x0F, 0x83},
			{0xBF, 0x02, 0x00, 0x00, 0x00, 0xB8, 0x3C, 0x00, 0x00, 0x40, 0x0F, 0x05},
			{0x89, 0x30, 0x48, 0x05, 0x08, 0x00, 0x00, 0x00},
		},
		forbidCode: [][]byte{
			{0xCD, 0x80},
			{0xB8, 0x09, 0x00, 0x00, 0x00, 0x0F, 0x05},
			{0xB8, 0x3C, 0x00, 0x00, 0x00, 0x0F, 0x05},
		},
	}, deps)
}

func CheckX32AllocatorFailureExecutableSmoke(deps RuntimeSmokeDeps) error {
	return checkAllocatorFailureExecutableSmoke(allocatorFailureExecutableSmokeOptions{
		target:      "linux-x32",
		stem:        "x32_allocator_failure",
		label:       "x32 allocator failure executable",
		wantClass:   1,
		wantMachine: 0x3e,
		wantCode: [][]byte{
			{0x89, 0xF0, 0x3D, 0x01, 0x00, 0x00, 0x00, 0x0F, 0x8D},
			{0xBF, 0x02, 0x00, 0x00, 0x00, 0xB8, 0x3C, 0x00, 0x00, 0x40, 0x0F, 0x05},
			{0xB8, 0x09, 0x00, 0x00, 0x40, 0x0F, 0x05},
		},
		forbidCode: [][]byte{
			{0xCD, 0x80},
			{0xB8, 0x3C, 0x00, 0x00, 0x00, 0x0F, 0x05},
			{0xB8, 0x09, 0x00, 0x00, 0x00, 0x0F, 0x05},
		},
	}, deps)
}

func CheckX86RawMemoryBoundsExecutableSmoke(deps RuntimeSmokeDeps) error {
	return checkRawMemoryBoundsExecutableSmoke(rawMemoryBoundsExecutableSmokeOptions{
		target:      "linux-x86",
		stem:        "x86_raw_memory_bounds",
		label:       "x86 raw memory bounds executable",
		wantClass:   1,
		wantMachine: 0x03,
		wantCode: [][]byte{
			{0xB8, 0xC0, 0x00, 0x00, 0x00, 0xCD, 0x80},
			{0x3D, 0x01, 0xF0, 0xFF, 0xFF, 0x0F, 0x83},
			{0xBB, 0x02, 0x00, 0x00, 0x00, 0xB8, 0x01, 0x00, 0x00, 0x00, 0xCD, 0x80},
			{0xBA, 0x00, 0x00, 0x00, 0x00, 0x83, 0xFA, 0x00, 0x0F, 0x8D},
			{0x81, 0xE7, 0x00, 0xF0, 0xFF, 0xFF},
			{0x8B, 0x0F, 0x83, 0xC7, 0x08},
			{0x01, 0xC2, 0x83, 0xC2, 0x04, 0x39, 0xCA},
			{0x01, 0xC2, 0x83, 0xC2, 0x01, 0x39, 0xCA},
			{0x88, 0x18, 0x53},
			{0x0F, 0xB6, 0x00, 0x50},
		},
		forbidCode: [][]byte{{0x0F, 0x05}},
	}, deps)
}

func CheckX32RawMemoryBoundsExecutableSmoke(deps RuntimeSmokeDeps) error {
	return checkRawMemoryBoundsExecutableSmoke(rawMemoryBoundsExecutableSmokeOptions{
		target:      "linux-x32",
		stem:        "x32_raw_memory_bounds",
		label:       "x32 raw memory bounds executable",
		wantClass:   1,
		wantMachine: 0x3e,
		wantCode: [][]byte{
			{0xB8, 0x09, 0x00, 0x00, 0x40, 0x0F, 0x05},
			{0x48, 0x3D, 0x01, 0xF0, 0xFF, 0xFF, 0x0F, 0x83},
			{0xBF, 0x02, 0x00, 0x00, 0x00, 0xB8, 0x3C, 0x00, 0x00, 0x40, 0x0F, 0x05},
			{0xBA, 0x00, 0x00, 0x00, 0x00, 0x81, 0xFA, 0x00, 0x00, 0x00, 0x00, 0x0F, 0x8D},
			{0x48, 0x89, 0xC7, 0x48, 0x81, 0xE7, 0x00, 0xF0, 0xFF, 0xFF},
			{0x8B, 0x8F, 0x00, 0x00, 0x00, 0x00, 0x48, 0x81, 0xC7, 0x08, 0x00, 0x00, 0x00},
			{0x48, 0x29, 0xF8, 0x48, 0x01, 0xC2, 0x81, 0xC2, 0x04, 0x00, 0x00, 0x00, 0x39, 0xCA},
			{0x48, 0x29, 0xF8, 0x48, 0x01, 0xC2, 0x81, 0xC2, 0x01, 0x00, 0x00, 0x00, 0x39, 0xCA},
			{0x48, 0x63, 0xD2, 0x48, 0x89, 0xF8, 0x48, 0x01, 0xD0},
			{0x44, 0x88, 0x00, 0x41, 0x50},
			{0x0F, 0xB6, 0x00, 0x50},
		},
		forbidCode: [][]byte{
			{0xCD, 0x80},
			{0xB8, 0x09, 0x00, 0x00, 0x00, 0x0F, 0x05},
			{0xB8, 0x3C, 0x00, 0x00, 0x00, 0x0F, 0x05},
		},
	}, deps)
}

func CheckX86RawPointerSlotExecutableSmoke(deps RuntimeSmokeDeps) error {
	return checkRawPointerSlotExecutableSmoke(rawPointerSlotExecutableSmokeOptions{
		target:      "linux-x86",
		stem:        "x86_raw_pointer_slot",
		label:       "x86 raw pointer slot executable",
		wantClass:   1,
		wantMachine: 0x03,
		wantCode: [][]byte{
			{0xB8, 0xC0, 0x00, 0x00, 0x00, 0xCD, 0x80},
			{0x3D, 0x01, 0xF0, 0xFF, 0xFF, 0x0F, 0x83},
			{0xBB, 0x02, 0x00, 0x00, 0x00, 0xB8, 0x01, 0x00, 0x00, 0x00, 0xCD, 0x80},
			{0xBA, 0x00, 0x00, 0x00, 0x00, 0x83, 0xFA, 0x00, 0x0F, 0x8D},
			{0x81, 0xE7, 0x00, 0xF0, 0xFF, 0xFF},
			{0x8B, 0x0F, 0x83, 0xC7, 0x08},
			{0x01, 0xC2, 0x83, 0xC2, 0x04, 0x39, 0xCA},
			{0x89, 0x18, 0x53},
			{0x8B, 0x00, 0x50},
		},
		forbidCode: [][]byte{{0x0F, 0x05}},
	}, deps)
}

func CheckX32RawPointerSlotExecutableSmoke(deps RuntimeSmokeDeps) error {
	return checkRawPointerSlotExecutableSmoke(rawPointerSlotExecutableSmokeOptions{
		target:      "linux-x32",
		stem:        "x32_raw_pointer_slot",
		label:       "x32 raw pointer slot executable",
		wantClass:   1,
		wantMachine: 0x3e,
		wantCode: [][]byte{
			{0xB8, 0x09, 0x00, 0x00, 0x40, 0x0F, 0x05},
			{0x48, 0x3D, 0x01, 0xF0, 0xFF, 0xFF, 0x0F, 0x83},
			{0xBF, 0x02, 0x00, 0x00, 0x00, 0xB8, 0x3C, 0x00, 0x00, 0x40, 0x0F, 0x05},
			{0xBA, 0x00, 0x00, 0x00, 0x00, 0x81, 0xFA, 0x00, 0x00, 0x00, 0x00, 0x0F, 0x8D},
			{0x48, 0x89, 0xC7, 0x48, 0x81, 0xE7, 0x00, 0xF0, 0xFF, 0xFF},
			{0x8B, 0x8F, 0x00, 0x00, 0x00, 0x00, 0x48, 0x81, 0xC7, 0x08, 0x00, 0x00, 0x00},
			{0x48, 0x29, 0xF8, 0x48, 0x01, 0xC2, 0x81, 0xC2, 0x04, 0x00, 0x00, 0x00, 0x39, 0xCA},
			{0x48, 0x63, 0xD2, 0x48, 0x89, 0xF8, 0x48, 0x01, 0xD0},
			{0x48, 0x89, 0xC7, 0x45, 0x89, 0xC0, 0x44, 0x89, 0x07, 0x41, 0x50},
			{0x8B, 0x00, 0x50},
		},
		forbidCode: [][]byte{
			{0xCD, 0x80},
			{0xB8, 0x09, 0x00, 0x00, 0x00, 0x0F, 0x05},
			{0xB8, 0x3C, 0x00, 0x00, 0x00, 0x0F, 0x05},
		},
	}, deps)
}

func CheckX86RawPointerOffsetSlotExecutableSmoke(deps RuntimeSmokeDeps) error {
	return checkRawPointerOffsetSlotExecutableSmoke(rawPointerSlotExecutableSmokeOptions{
		target:      "linux-x86",
		stem:        "x86_raw_pointer_offset_slot",
		label:       "x86 raw pointer offset slot executable",
		wantClass:   1,
		wantMachine: 0x03,
		wantCode: [][]byte{
			{0xB8, 0xC0, 0x00, 0x00, 0x00, 0xCD, 0x80},
			{0x3D, 0x01, 0xF0, 0xFF, 0xFF, 0x0F, 0x83},
			{0xBB, 0x02, 0x00, 0x00, 0x00, 0xB8, 0x01, 0x00, 0x00, 0x00, 0xCD, 0x80},
			{0x81, 0xE7, 0x00, 0xF0, 0xFF, 0xFF},
			{0x8B, 0x0F, 0x83, 0xC7, 0x08},
			{0x01, 0xC2, 0x83, 0xC2, 0x04, 0x39, 0xCA},
			{0x89, 0x18, 0x53},
			{0x8B, 0x00, 0x50},
		},
		forbidCode: [][]byte{
			{0x0F, 0x05},
			{0x01, 0xC2, 0x83, 0xC2, 0x01, 0x39, 0xCA},
		},
	}, deps)
}

func CheckX32RawPointerOffsetSlotExecutableSmoke(deps RuntimeSmokeDeps) error {
	return checkRawPointerOffsetSlotExecutableSmoke(rawPointerSlotExecutableSmokeOptions{
		target:      "linux-x32",
		stem:        "x32_raw_pointer_offset_slot",
		label:       "x32 raw pointer offset slot executable",
		wantClass:   1,
		wantMachine: 0x3e,
		wantCode: [][]byte{
			{0xB8, 0x09, 0x00, 0x00, 0x40, 0x0F, 0x05},
			{0x48, 0x3D, 0x01, 0xF0, 0xFF, 0xFF, 0x0F, 0x83},
			{0xBF, 0x02, 0x00, 0x00, 0x00, 0xB8, 0x3C, 0x00, 0x00, 0x40, 0x0F, 0x05},
			{0x48, 0x89, 0xC7, 0x48, 0x81, 0xE7, 0x00, 0xF0, 0xFF, 0xFF},
			{0x8B, 0x8F, 0x00, 0x00, 0x00, 0x00, 0x48, 0x81, 0xC7, 0x08, 0x00, 0x00, 0x00},
			{0x48, 0x29, 0xF8, 0x48, 0x01, 0xC2, 0x81, 0xC2, 0x04, 0x00, 0x00, 0x00, 0x39, 0xCA},
			{0x48, 0x63, 0xD2, 0x48, 0x89, 0xF8, 0x48, 0x01, 0xD0},
			{0x48, 0x89, 0xC7, 0x45, 0x89, 0xC0, 0x44, 0x89, 0x07, 0x41, 0x50},
			{0x8B, 0x00, 0x50},
		},
		forbidCode: [][]byte{
			{0xCD, 0x80},
			{0xB8, 0x09, 0x00, 0x00, 0x00, 0x0F, 0x05},
			{0xB8, 0x3C, 0x00, 0x00, 0x00, 0x0F, 0x05},
			{0x48, 0x29, 0xF8, 0x48, 0x01, 0xC2, 0x81, 0xC2, 0x01, 0x00, 0x00, 0x00, 0x39, 0xCA},
		},
	}, deps)
}

func CheckX86NetworkingLifecycleRuntimeSmoke(deps RuntimeSmokeDeps) error {
	return checkNetworkingLifecycleRuntimeSmoke(networkingLifecycleRuntimeSmokeOptions{
		target:      "linux-x86",
		stem:        "x86_networking_lifecycle",
		label:       "x86 networking lifecycle runtime",
		wantClass:   1,
		wantMachine: 0x03,
		wantCode: [][]byte{
			{0xB8, 0x66, 0x00, 0x00, 0x00},
			{0xBB, 0x01, 0x00, 0x00, 0x00},
			{0xBB, 0x02, 0x00, 0x00, 0x00},
			{0xBB, 0x03, 0x00, 0x00, 0x00},
			{0xBB, 0x04, 0x00, 0x00, 0x00},
			{0xBB, 0x09, 0x00, 0x00, 0x00},
			{0xBB, 0x0A, 0x00, 0x00, 0x00},
			{0xBB, 0x0E, 0x00, 0x00, 0x00},
			{0xBB, 0x12, 0x00, 0x00, 0x00},
			{0xB8, 0x03, 0x00, 0x00, 0x00},
			{0xB8, 0x04, 0x00, 0x00, 0x00},
			{0xB8, 0x49, 0x01, 0x00, 0x00},
			{0xB8, 0xFF, 0x00, 0x00, 0x00},
			{0xB8, 0x00, 0x01, 0x00, 0x00},
			{0xB8, 0x37, 0x00, 0x00, 0x00},
			{0x0D, 0x00, 0x08, 0x00, 0x00},
			{0xB8, 0x06, 0x00, 0x00, 0x00},
			{0xCD, 0x80},
		},
		forbidCode: []byte{0xB8, 0x03, 0x00, 0x00, 0x40},
	}, deps)
}

func CheckX32NetworkingLifecycleRuntimeSmoke(deps RuntimeSmokeDeps) error {
	return checkNetworkingLifecycleRuntimeSmoke(networkingLifecycleRuntimeSmokeOptions{
		target:      "linux-x32",
		stem:        "x32_networking_lifecycle",
		label:       "x32 networking lifecycle runtime",
		wantClass:   1,
		wantMachine: 0x3e,
		wantCode: [][]byte{
			{0xB8, 0x29, 0x00, 0x00, 0x40},
			{0xB8, 0x31, 0x00, 0x00, 0x40},
			{0xB8, 0x2A, 0x00, 0x00, 0x40},
			{0xB8, 0x32, 0x00, 0x00, 0x40},
			{0xB8, 0x20, 0x01, 0x00, 0x40},
			{0xB8, 0x00, 0x00, 0x00, 0x40},
			{0xB8, 0x01, 0x00, 0x00, 0x40},
			{0xB8, 0x2C, 0x00, 0x00, 0x40},
			{0xB8, 0x05, 0x02, 0x00, 0x40},
			{0xB8, 0x1D, 0x02, 0x00, 0x40},
			{0xB8, 0xE8, 0x00, 0x00, 0x40},
			{0xB8, 0xE9, 0x00, 0x00, 0x40},
			{0xB8, 0x23, 0x01, 0x00, 0x40},
			{0xB8, 0x48, 0x00, 0x00, 0x40},
			{0x0D, 0x00, 0x08, 0x00, 0x00},
			{0xB8, 0x03, 0x00, 0x00, 0x40},
			{0x0F, 0x05},
		},
		forbidCode: []byte{0xCD, 0x80},
	}, deps)
}

func CheckX86IslandFreeExecutableSmoke(deps RuntimeSmokeDeps) error {
	return checkIslandFreeExecutableSmoke(islandFreeExecutableSmokeOptions{
		target:      "linux-x86",
		stem:        "x86_island_free",
		label:       "x86 island free executable",
		wantClass:   1,
		wantMachine: 0x03,
		wantCode: [][]byte{
			{0xB8, 0xC0, 0x00, 0x00, 0x00, 0xCD, 0x80},
			{0xC7, 0x00, 0x10, 0x00, 0x00, 0x00},
			{0x8B, 0x4B, 0x08, 0xB8, 0x5B, 0x00, 0x00, 0x00, 0xCD, 0x80},
		},
		wantDebugCode: [][]byte{
			{0xC7, 0x00, 0x00, 0x10, 0x00, 0x00},
			{0x8B, 0x43, 0x0C, 0x85, 0xC0, 0x0F, 0x84},
			{0xBB, 0x02, 0x00, 0x00, 0x00, 0xB8, 0x01, 0x00, 0x00, 0x00, 0xCD, 0x80},
			{0xC7, 0x43, 0x0C, 0x01, 0x00, 0x00, 0x00},
			{0xB8, 0x7D, 0x00, 0x00, 0x00, 0xCD, 0x80},
		},
		forbidCode:      [][]byte{{0x0F, 0x05}},
		forbidDebugCode: [][]byte{{0x8B, 0x4B, 0x08, 0xB8, 0x5B, 0x00, 0x00, 0x00, 0xCD, 0x80}, {0x0F, 0x05}},
	}, deps)
}

func CheckX32IslandFreeExecutableSmoke(deps RuntimeSmokeDeps) error {
	return checkIslandFreeExecutableSmoke(islandFreeExecutableSmokeOptions{
		target:      "linux-x32",
		stem:        "x32_island_free",
		label:       "x32 island free executable",
		wantClass:   1,
		wantMachine: 0x3e,
		wantCode: [][]byte{
			{0xB8, 0x09, 0x00, 0x00, 0x40, 0x0F, 0x05},
			{0xC7, 0x00, 0x10, 0x00, 0x00, 0x00},
			{0x8B, 0x77, 0x08, 0xB8, 0x0B, 0x00, 0x00, 0x40, 0x0F, 0x05},
		},
		wantDebugCode: [][]byte{
			{0xC7, 0x00, 0x00, 0x10, 0x00, 0x00},
			{0x8B, 0x47, 0x0C, 0x85, 0xC0, 0x0F, 0x84},
			{0xBF, 0x02, 0x00, 0x00, 0x00, 0xB8, 0x3C, 0x00, 0x00, 0x40, 0x0F, 0x05},
			{0x48, 0x89, 0xF8, 0xC7, 0x40, 0x0C, 0x01, 0x00, 0x00, 0x00},
			{0x8B, 0x47, 0x08, 0x2D, 0x00, 0x10, 0x00, 0x00, 0x48, 0x89, 0xC6},
			{0xB8, 0x0A, 0x00, 0x00, 0x40, 0x0F, 0x05},
		},
		forbidCode: [][]byte{
			{0xCD, 0x80},
			{0xB8, 0x0B, 0x00, 0x00, 0x00, 0x0F, 0x05},
		},
		forbidDebugCode: [][]byte{
			{0xCD, 0x80},
			{0xB8, 0x0A, 0x00, 0x00, 0x00, 0x0F, 0x05},
			{0xB8, 0x0B, 0x00, 0x00, 0x40, 0x0F, 0x05},
		},
	}, deps)
}

func checkStderrFDRuntimeSmoke(opts stderrFDRuntimeSmokeOptions, deps RuntimeSmokeDeps) error {
	tmpDir, err := os.MkdirTemp("", "tetra-"+opts.stem+"-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	srcPath := filepath.Join(tmpDir, opts.stem+".tetra")
	outPath := filepath.Join(tmpDir, opts.stem)
	src := `
func main() -> Int
uses alloc, capability, io, mem:
    unsafe:
        let cap: cap.io = core.cap_io()
        var buf: []u8 = core.make_u8(1)
        buf[0] = 69
        let written: Int = core.net_write(2, buf, 0, 1, cap)
        if written == 999:
            return 7
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if err := buildExecutable(deps, srcPath, outPath, opts.target); err != nil {
		return err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return err
	}
	if err := checkRuntimeSmokeELF(data, opts.label, opts.wantClass, opts.wantMachine); err != nil {
		return err
	}
	for _, wantCode := range opts.wantCode {
		if !bytes.Contains(data, wantCode) {
			return fmt.Errorf("%s missing stderr fd/write sequence % x", opts.label, wantCode)
		}
	}
	if len(opts.forbidCode) > 0 && bytes.Contains(data, opts.forbidCode) {
		return fmt.Errorf("%s contains forbidden syscall sequence % x", opts.label, opts.forbidCode)
	}
	return nil
}

func checkStdoutExecutableSmoke(opts stdoutExecutableSmokeOptions, deps RuntimeSmokeDeps) error {
	tmpDir, err := os.MkdirTemp("", "tetra-"+opts.stem+"-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	srcPath := filepath.Join(tmpDir, opts.stem+".tetra")
	outPath := filepath.Join(tmpDir, opts.stem)
	src := fmt.Sprintf("func main() -> Int\nuses io:\n    print(%q)\n    return 0\n", opts.wantLiteral)
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if err := buildExecutable(deps, srcPath, outPath, opts.target); err != nil {
		return err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return err
	}
	if err := checkRuntimeSmokeELF(data, opts.label, opts.wantClass, opts.wantMachine); err != nil {
		return err
	}
	if !bytes.Contains(data, []byte(opts.wantLiteral)) {
		return fmt.Errorf("%s missing stdout string literal %q", opts.label, opts.wantLiteral)
	}
	for _, wantCode := range opts.wantCode {
		if !bytes.Contains(data, wantCode) {
			return fmt.Errorf("%s missing target write syscall sequence % x", opts.label, wantCode)
		}
	}
	if len(opts.forbidCode) > 0 && bytes.Contains(data, opts.forbidCode) {
		return fmt.Errorf("%s contains forbidden syscall sequence % x", opts.label, opts.forbidCode)
	}
	return nil
}

func checkAllocatorExecutableSmoke(opts allocatorExecutableSmokeOptions, deps RuntimeSmokeDeps) error {
	tmpDir, err := os.MkdirTemp("", "tetra-"+opts.stem+"-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	srcPath := filepath.Join(tmpDir, opts.stem+".tetra")
	outPath := filepath.Join(tmpDir, opts.stem)
	src := `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(4)
        let _: Int = core.store_i32(p, 42, mem)
        return core.load_i32(p, mem)
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if err := buildExecutable(deps, srcPath, outPath, opts.target); err != nil {
		return err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return err
	}
	if err := checkRuntimeSmokeELF(data, opts.label, opts.wantClass, opts.wantMachine); err != nil {
		return err
	}
	for _, wantCode := range opts.wantCode {
		if !bytes.Contains(data, wantCode) {
			return fmt.Errorf("%s missing target allocator sequence % x", opts.label, wantCode)
		}
	}
	for _, forbidCode := range opts.forbidCode {
		if len(forbidCode) > 0 && bytes.Contains(data, forbidCode) {
			return fmt.Errorf("%s contains forbidden allocator sequence % x", opts.label, forbidCode)
		}
	}
	return nil
}

func checkAllocatorFailureExecutableSmoke(opts allocatorFailureExecutableSmokeOptions, deps RuntimeSmokeDeps) error {
	tmpDir, err := os.MkdirTemp("", "tetra-"+opts.stem+"-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	srcPath := filepath.Join(tmpDir, opts.stem+".tetra")
	outPath := filepath.Join(tmpDir, opts.stem)
	src := `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let _: ptr = core.alloc_bytes(0)
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if err := buildExecutable(deps, srcPath, outPath, opts.target); err != nil {
		return err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return err
	}
	if err := checkRuntimeSmokeELF(data, opts.label, opts.wantClass, opts.wantMachine); err != nil {
		return err
	}
	for _, wantCode := range opts.wantCode {
		if !bytes.Contains(data, wantCode) {
			return fmt.Errorf("%s missing allocator failure sequence % x", opts.label, wantCode)
		}
	}
	for _, forbidCode := range opts.forbidCode {
		if len(forbidCode) > 0 && bytes.Contains(data, forbidCode) {
			return fmt.Errorf("%s contains forbidden allocator failure sequence % x", opts.label, forbidCode)
		}
	}
	return nil
}

func checkRawMemoryBoundsExecutableSmoke(opts rawMemoryBoundsExecutableSmokeOptions, deps RuntimeSmokeDeps) error {
	tmpDir, err := os.MkdirTemp("", "tetra-"+opts.stem+"-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	srcPath := filepath.Join(tmpDir, opts.stem+".tetra")
	outPath := filepath.Join(tmpDir, opts.stem)
	src := `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(8)
        let stored_i32: Int = core.store_i32(p, 42, mem)
        let q: ptr = core.ptr_add(p, 1, mem)
        let stored_u8: u8 = core.store_u8(q, 7, mem)
        let direct: Int = core.load_i32(p, mem)
        let loaded_u8: u8 = core.load_u8(q, mem)
        return direct
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if err := buildExecutable(deps, srcPath, outPath, opts.target); err != nil {
		return err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return err
	}
	if err := checkRuntimeSmokeELF(data, opts.label, opts.wantClass, opts.wantMachine); err != nil {
		return err
	}
	for _, wantCode := range opts.wantCode {
		if !bytes.Contains(data, wantCode) {
			return fmt.Errorf("%s missing raw memory bounds sequence % x", opts.label, wantCode)
		}
	}
	for _, forbidCode := range opts.forbidCode {
		if len(forbidCode) > 0 && bytes.Contains(data, forbidCode) {
			return fmt.Errorf("%s contains forbidden raw memory bounds sequence % x", opts.label, forbidCode)
		}
	}
	return nil
}

func checkRawPointerSlotExecutableSmoke(opts rawPointerSlotExecutableSmokeOptions, deps RuntimeSmokeDeps) error {
	tmpDir, err := os.MkdirTemp("", "tetra-"+opts.stem+"-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	srcPath := filepath.Join(tmpDir, opts.stem+".tetra")
	outPath := filepath.Join(tmpDir, opts.stem)
	src := `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let slot: ptr = core.alloc_bytes(4)
        let payload: ptr = core.alloc_bytes(4)
        let stored: ptr = core.store_ptr(slot, payload, mem)
        let loaded: ptr = core.load_ptr(slot, mem)
        return 0
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if err := buildExecutable(deps, srcPath, outPath, opts.target); err != nil {
		return err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return err
	}
	if err := checkRuntimeSmokeELF(data, opts.label, opts.wantClass, opts.wantMachine); err != nil {
		return err
	}
	for _, wantCode := range opts.wantCode {
		if !bytes.Contains(data, wantCode) {
			return fmt.Errorf("%s missing raw pointer slot sequence % x", opts.label, wantCode)
		}
	}
	for _, forbidCode := range opts.forbidCode {
		if len(forbidCode) > 0 && bytes.Contains(data, forbidCode) {
			return fmt.Errorf("%s contains forbidden raw pointer slot sequence % x", opts.label, forbidCode)
		}
	}
	return nil
}

func checkRawPointerOffsetSlotExecutableSmoke(opts rawPointerSlotExecutableSmokeOptions, deps RuntimeSmokeDeps) error {
	tmpDir, err := os.MkdirTemp("", "tetra-"+opts.stem+"-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	srcPath := filepath.Join(tmpDir, opts.stem+".tetra")
	outPath := filepath.Join(tmpDir, opts.stem)
	src := `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let slot: ptr = core.alloc_bytes(8)
        let payload: ptr = core.alloc_bytes(4)
        let stored: ptr = core.store_ptr(core.ptr_add(slot, 3, mem), payload, mem)
        let loaded: ptr = core.load_ptr(core.ptr_add(slot, 3, mem), mem)
        return 0
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if err := buildExecutable(deps, srcPath, outPath, opts.target); err != nil {
		return err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return err
	}
	if err := checkRuntimeSmokeELF(data, opts.label, opts.wantClass, opts.wantMachine); err != nil {
		return err
	}
	for _, wantCode := range opts.wantCode {
		if !bytes.Contains(data, wantCode) {
			return fmt.Errorf("%s missing raw pointer offset slot sequence % x", opts.label, wantCode)
		}
	}
	for _, forbidCode := range opts.forbidCode {
		if len(forbidCode) > 0 && bytes.Contains(data, forbidCode) {
			return fmt.Errorf("%s contains forbidden raw pointer offset slot sequence % x", opts.label, forbidCode)
		}
	}
	return nil
}

func checkNetworkingLifecycleRuntimeSmoke(opts networkingLifecycleRuntimeSmokeOptions, deps RuntimeSmokeDeps) error {
	tmpDir, err := os.MkdirTemp("", "tetra-"+opts.stem+"-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	srcPath := filepath.Join(tmpDir, opts.stem+".tetra")
	outPath := filepath.Join(tmpDir, opts.stem)
	src := `
func main() -> Int
uses alloc, capability, io, mem:
    unsafe:
        let cap: cap.io = core.cap_io()
        let server: Int = core.net_socket_tcp4(cap)
        let client: Int = core.net_socket_tcp4(cap)
        if server < 0 || client < 0:
            return 11
        var buf: []u8 = core.make_u8(4)
        buf[0] = 80
        buf[1] = 73
        buf[2] = 78
        buf[3] = 71
        let bind_status: Int = core.net_bind_tcp4_loopback(server, 0, cap)
        let listen_status: Int = core.net_listen(server, 8, cap)
        let connect_status: Int = core.net_connect_tcp4_loopback(client, 0, cap)
        let accepted: Int = core.net_accept4(server, 0, cap)
        let written: Int = core.net_write(client, buf, 0, 1, cap)
        let read_status: Int = core.net_read(client, buf, 0, 1, cap)
        let sent: Int = core.net_send(client, buf, 0, 1, cap)
        let recv_status: Int = core.net_recv(client, buf, 0, 1, cap)
        let nb: Int = core.net_set_nonblocking(server, cap)
        let reuse: Int = core.net_set_reuseport(server, cap)
        let nodelay: Int = core.net_set_tcp_nodelay(client, cap)
        let epfd: Int = core.net_epoll_create(cap)
        let add_read: Int = core.net_epoll_ctl_add_read(epfd, server, cap)
        let mod_read: Int = core.net_epoll_ctl_mod_read(epfd, server, cap)
        let mod_rw: Int = core.net_epoll_ctl_mod_read_write(epfd, server, cap)
        let del_read: Int = core.net_epoll_ctl_delete(epfd, server, cap)
        let add_rw: Int = core.net_epoll_ctl_add_read_write(epfd, server, cap)
        let del_rw: Int = core.net_epoll_ctl_delete(epfd, server, cap)
        let wait_one: Int = core.net_epoll_wait_one(epfd, 0, cap)
        var event: []i32 = core.make_i32(2)
        let wait_into: Int = core.net_epoll_wait_one_into(epfd, event, 0, cap)
        let epfd_closed: Int = core.net_close(epfd, cap)
        let client_closed: Int = core.net_close(client, cap)
        let server_closed: Int = core.net_close(server, cap)
        if bind_status == 999 || listen_status == 999 || connect_status == 999 || accepted == 999:
            return 12
        if written == 999 || read_status == 999 || sent == 999 || recv_status == 999:
            return 13
        if nb < 0:
            return 14
        if reuse == 999 || nodelay == 999:
            return 15
        if epfd == 999 || add_read == 999 || mod_read == 999 || mod_rw == 999:
            return 16
        if del_read == 999 || add_rw == 999 || del_rw == 999:
            return 17
        if wait_one == 999 || wait_into == 999 || epfd_closed == 999:
            return 18
        if client_closed == 999 || server_closed == 999:
            return 19
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if err := buildExecutable(deps, srcPath, outPath, opts.target); err != nil {
		return err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return err
	}
	if err := checkRuntimeSmokeELF(data, opts.label, opts.wantClass, opts.wantMachine); err != nil {
		return err
	}
	for _, wantCode := range opts.wantCode {
		if !bytes.Contains(data, wantCode) {
			return fmt.Errorf("%s missing target networking syscall sequence % x", opts.label, wantCode)
		}
	}
	if len(opts.forbidCode) > 0 && bytes.Contains(data, opts.forbidCode) {
		return fmt.Errorf("%s contains forbidden syscall sequence % x", opts.label, opts.forbidCode)
	}
	return nil
}

func checkIslandFreeExecutableSmoke(opts islandFreeExecutableSmokeOptions, deps RuntimeSmokeDeps) error {
	tmpDir, err := os.MkdirTemp("", "tetra-"+opts.stem+"-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	srcPath := filepath.Join(tmpDir, opts.stem+".tetra")
	src := `fun main(): i32 uses alloc, islands, mem {
  var out: i32 = 0
  island(64) as isl {
    var xs: []u16 = core.island_make_u16(isl, 2)
    xs[0] = 40
    xs[1] = 2
    out = xs[0] + xs[1]
  }
  return out
}
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	normalPath := filepath.Join(tmpDir, opts.stem)
	if err := buildExecutableWithOptions(deps, srcPath, normalPath, opts.target, RuntimeBuildOptions{}); err != nil {
		return err
	}
	if err := checkIslandFreeExecutableBytes(normalPath, opts.label, opts, opts.wantCode, opts.forbidCode); err != nil {
		return err
	}

	debugPath := filepath.Join(tmpDir, opts.stem+"_debug")
	if err := buildExecutableWithOptions(deps, srcPath, debugPath, opts.target, RuntimeBuildOptions{IslandsDebug: true}); err != nil {
		return err
	}
	return checkIslandFreeExecutableBytes(debugPath, opts.label+" debug", opts, opts.wantDebugCode, opts.forbidDebugCode)
}

func checkIslandFreeExecutableBytes(path string, label string, opts islandFreeExecutableSmokeOptions, wantCodes [][]byte, forbidCodes [][]byte) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := checkRuntimeSmokeELF(data, label, opts.wantClass, opts.wantMachine); err != nil {
		return err
	}
	for _, wantCode := range wantCodes {
		if !bytes.Contains(data, wantCode) {
			return fmt.Errorf("%s missing target island/free sequence % x", label, wantCode)
		}
	}
	for _, forbidCode := range forbidCodes {
		if len(forbidCode) > 0 && bytes.Contains(data, forbidCode) {
			return fmt.Errorf("%s contains forbidden island/free sequence % x", label, forbidCode)
		}
	}
	return nil
}

func checkRuntimeSmokeELF(data []byte, label string, wantClass byte, wantMachine uint16) error {
	if len(data) < 20 || string(data[:4]) != "\x7fELF" {
		return fmt.Errorf("%s output is not an ELF executable", label)
	}
	if data[4] != wantClass {
		return fmt.Errorf("%s ELF class = %d, want %d", label, data[4], wantClass)
	}
	if machine := uint16(data[18]) | uint16(data[19])<<8; machine != wantMachine {
		return fmt.Errorf("%s ELF machine = %#x, want %#x", label, machine, wantMachine)
	}
	return nil
}
