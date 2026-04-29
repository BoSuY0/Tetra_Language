package x64

import (
	"encoding/binary"
	"testing"
)

func TestObjectPatchRel32ForwardAndBackward(t *testing.T) {
	code := make([]byte, 16)

	if err := PatchRel32(code, 4, 16); err != nil {
		t.Fatalf("forward patch: %v", err)
	}
	gotForward := int32(binary.LittleEndian.Uint32(code[4:8]))
	if gotForward != 8 {
		t.Fatalf("forward disp = %d, want 8", gotForward)
	}

	if err := PatchRel32(code, 8, 4); err != nil {
		t.Fatalf("backward patch: %v", err)
	}
	gotBackward := int32(binary.LittleEndian.Uint32(code[8:12]))
	if gotBackward != -8 {
		t.Fatalf("backward disp = %d, want -8", gotBackward)
	}
}

func TestObjectPatchRel32RejectsOutOfRangeTargets(t *testing.T) {
	code := make([]byte, 8)
	if err := PatchRel32(code, 0, int(^uint32(0)>>1)+16); err == nil {
		t.Fatalf("expected out-of-range error for large forward target")
	}
	if err := PatchRel32(code, 4, -1<<31-16); err == nil {
		t.Fatalf("expected out-of-range error for large backward target")
	}
}

func TestABIAlignStackSize(t *testing.T) {
	cases := []struct {
		in   int
		want int
	}{
		{in: -1, want: 0},
		{in: 0, want: 0},
		{in: 1, want: 16},
		{in: 8, want: 16},
		{in: 16, want: 16},
		{in: 17, want: 32},
	}
	for _, tc := range cases {
		if got := AlignStackSize(tc.in); got != tc.want {
			t.Fatalf("AlignStackSize(%d)=%d, want %d", tc.in, got, tc.want)
		}
	}
}
