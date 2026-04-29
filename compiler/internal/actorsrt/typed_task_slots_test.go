package actorsrt

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/backend/x64"
)

func TestEmitTaskJoinTypedSlotBounds(t *testing.T) {
	tests := []struct {
		name  string
		slots int
		ok    bool
	}{
		{name: "slot_1_rejected", slots: 1, ok: false},
		{name: "slot_2_allowed", slots: 2, ok: true},
		{name: "slot_4_allowed", slots: 4, ok: true},
		{name: "slot_5_allowed", slots: 5, ok: true},
		{name: "slot_8_allowed", slots: 8, ok: true},
		{name: "slot_9_rejected", slots: 9, ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &x64.Emitter{}
			var patches []callPatch
			err := emitTaskJoinTyped(e, tt.slots, &patches)
			if tt.ok {
				if err != nil {
					t.Fatalf("emitTaskJoinTyped(%d): %v", tt.slots, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error for slot count %d", tt.slots)
			}
			if !strings.Contains(err.Error(), "unsupported typed task join slot count") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestEmitTaskJoinTypedWrapperWindowsX64SlotBounds(t *testing.T) {
	tests := []struct {
		name  string
		slots int
		ok    bool
	}{
		{name: "slot_1_rejected", slots: 1, ok: false},
		{name: "slot_2_allowed", slots: 2, ok: true},
		{name: "slot_4_allowed", slots: 4, ok: true},
		{name: "slot_5_allowed", slots: 5, ok: true},
		{name: "slot_8_allowed", slots: 8, ok: true},
		{name: "slot_9_rejected", slots: 9, ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &x64.Emitter{}
			var patches []callPatch
			err := emitTaskJoinTypedWrapperWindowsX64(e, tt.slots, "__tetra_task_join_typed_impl", &patches)
			if tt.ok {
				if err != nil {
					t.Fatalf("emitTaskJoinTypedWrapperWindowsX64(%d): %v", tt.slots, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error for slot count %d", tt.slots)
			}
			if !strings.Contains(err.Error(), "unsupported typed task join wrapper slots") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}
