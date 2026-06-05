package runtimeabi

import "testing"

func TestRegionAllocatorConfigDefinesAlignedBumpContract(t *testing.T) {
	cfg := RuntimeRegionAllocatorConfig(false)
	if cfg.AlignmentBytes != 16 {
		t.Fatalf("alignment = %d, want 16", cfg.AlignmentBytes)
	}
	if cfg.HeaderBytes != 16 {
		t.Fatalf("header bytes = %d, want 16", cfg.HeaderBytes)
	}
	if cfg.DebugHeaderBytes != 4096 {
		t.Fatalf("debug header bytes = %d, want 4096", cfg.DebugHeaderBytes)
	}
	if cfg.MaxPayloadBytes <= 0 || cfg.MaxPayloadBytes+cfg.HeaderBytes != MaxRegionMapBytes {
		t.Fatalf("max payload/header = %d/%d, want total %d", cfg.MaxPayloadBytes, cfg.HeaderBytes, MaxRegionMapBytes)
	}

	debug := RuntimeRegionAllocatorConfig(true)
	if debug.HeaderBytes != cfg.DebugHeaderBytes {
		t.Fatalf("debug header bytes = %d, want %d", debug.HeaderBytes, cfg.DebugHeaderBytes)
	}
	if debug.MaxPayloadBytes+debug.HeaderBytes != MaxRegionMapBytes {
		t.Fatalf("debug max payload/header = %d/%d, want total %d", debug.MaxPayloadBytes, debug.HeaderBytes, MaxRegionMapBytes)
	}
}

func TestAlignRegionBytes(t *testing.T) {
	tests := []struct {
		in   int64
		want int64
	}{
		{0, 0},
		{1, 16},
		{15, 16},
		{16, 16},
		{17, 32},
		{31, 32},
		{32, 32},
	}
	for _, tt := range tests {
		got, ok := AlignRegionBytes(tt.in)
		if !ok || got != tt.want {
			t.Fatalf("AlignRegionBytes(%d) = %d,%v want %d,true", tt.in, got, ok, tt.want)
		}
	}
	if _, ok := AlignRegionBytes(-1); ok {
		t.Fatalf("AlignRegionBytes(-1) unexpectedly succeeded")
	}
	if _, ok := AlignRegionBytes(int64(MaxRegionMapBytes)); ok {
		t.Fatalf("AlignRegionBytes(MaxRegionMapBytes) unexpectedly succeeded")
	}
}
