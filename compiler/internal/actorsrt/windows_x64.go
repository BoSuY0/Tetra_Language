package actorsrt

import (
	"fmt"
	"sort"

	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/format/tobj"
)

const (
	winImportVirtualAlloc = "kernel32.VirtualAlloc"
)

// BuildWindowsX64 returns a runtime object that provides:
// - __tetra_entry
// - __tetra_actor_spawn / send / recv / self / sender
// - __tetra_actor_send_msg / __tetra_actor_recv_msg
//
// entries[0] must be the program entry symbol (main).
// Actor entry IDs are computed as FNV-1a 32-bit hashes of the string literals used in `core.spawn(...)`.
func BuildWindowsX64(entries []string) (*tobj.Object, error) {
	if len(entries) == 0 || entries[0] == "" {
		return nil, fmt.Errorf("missing entry symbols (need main at index 0)")
	}

	e := &x64.Emitter{}
	funcOffsets := make(map[string]int)
	var callPatches []callPatch
	var leaPatches []leaPatch
	var jmpPatches []callPatch
	var importPatches []importPatch

	emitFunc := func(name string, fn func() error) error {
		if _, exists := funcOffsets[name]; exists {
			return fmt.Errorf("duplicate runtime function '%s'", name)
		}
		funcOffsets[name] = len(e.Buf)
		return fn()
	}

	if err := emitFunc("__tetra_entry", func() error { return emitEntryWindowsX64(e, entries[0], &callPatches, &leaPatches, &importPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_switch_to", func() error { return emitSwitchToWindowsX64(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_yield", func() error { return emitActorYieldWindowsX64(e, &callPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_exit", func() error { return emitActorExitWindowsX64(e, &callPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_trampoline", func() error { return emitActorTrampolineWindowsX64(e, &callPatches) }); err != nil {
		return nil, err
	}

	if err := emitFunc("__tetra_actor_spawn_impl", func() error { return emitSpawnWindowsX64(e, &callPatches, &leaPatches, &importPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_send_impl", func() error { return emitSend(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_send_msg_impl", func() error { return emitSendMsg(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_recv_impl", func() error { return emitRecv(e, &callPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_recv_msg_impl", func() error { return emitRecvMsg(e, &callPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_self_impl", func() error { return emitSelf(e) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_sender_impl", func() error { return emitSender(e) }); err != nil {
		return nil, err
	}

	if err := emitFunc("__tetra_actor_spawn", func() error { return emitActorSpawnWrapperWindowsX64(e, &jmpPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_send", func() error { return emitActorSendWrapperWindowsX64(e, &jmpPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_send_msg", func() error { return emitActorSendMsgWrapperWindowsX64(e, &jmpPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_recv", func() error { return emitActorNoArgWrapperWindowsX64(e, "__tetra_actor_recv_impl", &jmpPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_recv_msg", func() error { return emitActorNoArgWrapperWindowsX64(e, "__tetra_actor_recv_msg_impl", &jmpPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_self", func() error { return emitActorNoArgWrapperWindowsX64(e, "__tetra_actor_self_impl", &jmpPatches) }); err != nil {
		return nil, err
	}
	if err := emitFunc("__tetra_actor_sender", func() error { return emitActorNoArgWrapperWindowsX64(e, "__tetra_actor_sender_impl", &jmpPatches) }); err != nil {
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
	for _, patch := range jmpPatches {
		target, ok := funcOffsets[patch.name]
		if !ok {
			return nil, fmt.Errorf("unknown jmp target '%s'", patch.name)
		}
		if err := x64.PatchRel32(code, patch.at, target); err != nil {
			return nil, err
		}
	}
	for _, patch := range importPatches {
		relocs = append(relocs, tobj.Reloc{Kind: tobj.RelocIATDisp32, At: uint32(patch.at), Name: patch.name, Addend: 0})
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

type importPatch struct {
	at   int
	name string
}
