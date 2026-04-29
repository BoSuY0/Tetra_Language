package lower

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/ir"
)

func TestLowerTypedTaskWrapperSlotBounds(t *testing.T) {
	tests := []struct {
		name      string
		slotCount int
		ok        bool
	}{
		{name: "slot_1_rejected", slotCount: 1, ok: false},
		{name: "slot_2_allowed", slotCount: 2, ok: true},
		{name: "slot_4_allowed", slotCount: 4, ok: true},
		{name: "slot_5_allowed", slotCount: 5, ok: true},
		{name: "slot_8_allowed", slotCount: 8, ok: true},
		{name: "slot_9_rejected", slotCount: 9, ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapper := typedTaskWrapper{
				Name:              "__tetra_task_typed_test",
				Target:            "worker",
				SlotCount:         tt.slotCount,
				StatusSlot:        tt.slotCount - 1,
				TargetReturnSlots: 1,
			}
			fn, err := lowerTypedTaskWrapper(wrapper)
			if tt.ok {
				if err != nil {
					t.Fatalf("lowerTypedTaskWrapper(%d): %v", tt.slotCount, err)
				}
				if fn.LocalSlots != tt.slotCount+1 {
					t.Fatalf("locals = %d, want %d", fn.LocalSlots, tt.slotCount+1)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error for slot count %d", tt.slotCount)
			}
			if !strings.Contains(err.Error(), "unsupported slot count") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestLowerTypedTaskWrapperStagedThrowingTargetPassThroughStatus(t *testing.T) {
	wrapper := typedTaskWrapper{
		Name:              "__tetra_task_typed_throwing",
		Target:            "worker",
		ErrorType:         "TaskErr",
		TargetThrowsType:  "TaskErr",
		SlotCount:         5,
		StatusSlot:        4,
		TargetReturnSlots: 1,
	}
	fn, err := lowerTypedTaskWrapper(wrapper)
	if err != nil {
		t.Fatalf("lowerTypedTaskWrapper: %v", err)
	}
	if fn.LocalSlots != 0 {
		t.Fatalf("locals = %d, want 0", fn.LocalSlots)
	}
	if len(fn.Instrs) != 2 {
		t.Fatalf("instr count = %d, want 2", len(fn.Instrs))
	}
	if fn.Instrs[0].Kind != ir.IRCall || fn.Instrs[0].Name != "worker" || fn.Instrs[0].RetSlots != 1 {
		t.Fatalf("first instr = %#v, want call worker ret1", fn.Instrs[0])
	}
	if fn.Instrs[1].Kind != ir.IRReturn {
		t.Fatalf("second instr = %#v, want return", fn.Instrs[1])
	}
}
