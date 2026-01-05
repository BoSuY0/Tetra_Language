package actorsrt

import (
	"fmt"
	"sort"

	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/backend/x64abi"
	"tetra_language/compiler/internal/format/tobj"
)

const (
	schedActorsPtrOff   = 0  // u64
	schedCapacityOff    = 8  // u32
	schedCountOff       = 12 // u32
	schedRspOff         = 16 // u64
	schedCurrentIdxOff  = 24 // u32
	schedMsgBaseOff     = 32 // u64
	schedMsgBumpOff     = 40 // u64
	schedMsgEndOff      = 48 // u64
	schedSize           = 64
	actorSizeShift      = 6 // 64 bytes
	actorSize           = 1 << actorSizeShift
	actorRspOff         = 0  // u64
	actorStatusOff      = 8  // u32
	actorEntryIDOff     = 12 // u32
	actorMailboxHeadOff = 16 // u64
	actorMailboxTailOff = 24 // u64
	actorLastSenderOff  = 32 // u32
	actorExitCodeOff    = 36 // u32

	statusFree    = 0
	statusReady   = 1
	statusBlocked = 2
	statusDone    = 3
)

const (
	stackSize   = 64 * 1024
	msgPoolSize = 64 * 1024
)

// BuildLinuxX64 returns a runtime object that provides:
// - __tetra_entry
// - __tetra_actor_spawn / send / recv / self / sender
//
// entries[0] must be the program entry symbol (main).
// Actor entry IDs are computed as FNV-1a 32-bit hashes of the string literals used in `core.spawn(...)`.
func BuildLinuxX64(entries []string) (*tobj.Object, error) {
	abi := x64abi.LinuxSysV()
	const linuxMapPrivateAnon = 0x22
	return buildSysVUnixX64(entries, abi.SysMmap, linuxMapPrivateAnon)
}

func buildSysVUnixX64(entries []string, sysMmap uint32, mapFlags uint32) (*tobj.Object, error) {
	if len(entries) == 0 || entries[0] == "" {
		return nil, fmt.Errorf("missing entry symbols (need main at index 0)")
	}

	e := &x64.Emitter{}
	funcOffsets := make(map[string]int)
	var callPatches []callPatch
	var leaPatches []leaPatch

	emitFunc := func(name string, fn func() error) error {
		if _, exists := funcOffsets[name]; exists {
			return fmt.Errorf("duplicate runtime function '%s'", name)
		}
		funcOffsets[name] = len(e.Buf)
		return fn()
	}

	if err := emitFunc("__tetra_entry", func() error { return emitEntry(e, entries[0], sysMmap, mapFlags, &callPatches, &leaPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_switch_to", func() error { return emitSwitchTo(e, &callPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_yield", func() error { return emitActorYield(e, &callPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_exit", func() error { return emitActorExit(e, &callPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_trampoline", func() error { return emitActorTrampoline(e, &callPatches) }); err != nil {
		return nil, err
	}

	if err := emitFunc("__tetra_actor_spawn", func() error { return emitSpawn(e, sysMmap, mapFlags, &callPatches, &leaPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_send", func() error { return emitSend(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_recv", func() error { return emitRecv(e, &callPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_self", func() error { return emitSelf(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_sender", func() error { return emitSender(e) }); err != nil {
		return nil, err
	}

	code := e.Buf
	for _, patch := range leaPatches {
		target, ok := funcOffsets[patch.name]
		if !ok {
			return nil, fmt.Errorf("unknown lea target '%s'", patch.name)
		}
		if err := x64.PatchRel32(code, patch.at, target); err != nil {
			return nil, err
		}
	}
	var relocs []tobj.Reloc
	for _, patch := range callPatches {
		target, ok := funcOffsets[patch.name]
		if ok {
			if err := x64.PatchRel32(code, patch.at, target); err != nil {
				return nil, err
			}
			continue
		}
		relocs = append(relocs, tobj.Reloc{Kind: tobj.RelocCallRel32, At: uint32(patch.at), Name: patch.name, Addend: 0})
	}

	names := make([]string, 0, len(funcOffsets))
	for name := range funcOffsets {
		names = append(names, name)
	}
	sort.Strings(names)
	symbols := make([]tobj.Symbol, 0, len(names))
	for _, name := range names {
		symbols = append(symbols, tobj.Symbol{Name: name, Offset: uint32(funcOffsets[name])})
	}

	return &tobj.Object{Code: code, Data: nil, Symbols: symbols, Relocs: relocs}, nil
}

type callPatch struct {
	at   int
	name string
}

type leaPatch struct {
	at   int
	name string
}
