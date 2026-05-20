package pe

import (
	"strings"
	"testing"
)

func TestWritePE64WindowsX64RejectsMissingImage(t *testing.T) {
	err := WritePE64WindowsX64(t.TempDir()+"/missing.exe", nil)
	if err == nil {
		t.Fatalf("expected missing image error")
	}
	if !strings.Contains(err.Error(), "missing PE image") {
		t.Fatalf("error = %v", err)
	}
}

func TestWritePE64WindowsX64RejectsEntryOffsetOutOfRange(t *testing.T) {
	err := WritePE64WindowsX64(t.TempDir()+"/bad-entry.exe", &PEImage{
		Text:        []byte{0xc3},
		EntryOffset: 2,
		Imports:     []string{"kernel32.ExitProcess"},
	})
	if err == nil {
		t.Fatalf("expected entry offset error")
	}
	if !strings.Contains(err.Error(), "entry offset out of range") {
		t.Fatalf("error = %v", err)
	}
}

func TestWritePE64WindowsX64RejectsMissingImports(t *testing.T) {
	err := WritePE64WindowsX64(t.TempDir()+"/missing-imports.exe", &PEImage{
		Text: []byte{0xc3},
	})
	if err == nil {
		t.Fatalf("expected missing imports error")
	}
	if !strings.Contains(err.Error(), "missing imports") {
		t.Fatalf("error = %v", err)
	}
}

func TestWritePE64WindowsX64RejectsOverflowSizedRelocOffsets(t *testing.T) {
	cases := []struct {
		name string
		img  PEImage
		want string
	}{
		{
			name: "iat",
			img: PEImage{
				Text:      []byte{0x90, 0x90, 0x90, 0x90},
				Imports:   []string{"kernel32.ExitProcess"},
				IATRelocs: []IATReloc{{At: int(^uint(0) >> 1), Name: "kernel32.ExitProcess"}},
			},
			want: "IAT relocation out of range",
		},
		{
			name: "rdata",
			img: PEImage{
				Text:        []byte{0x90, 0x90, 0x90, 0x90},
				RData:       []byte("literal"),
				Imports:     []string{"kernel32.ExitProcess"},
				RDataRelocs: []RDataReloc{{At: int(^uint(0) >> 1)}},
			},
			want: "rdata relocation out of range",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recovered := recover(); recovered != nil {
					t.Fatalf("WritePE64WindowsX64 panicked: %v", recovered)
				}
			}()
			err := WritePE64WindowsX64(t.TempDir()+"/bad-reloc.exe", &tc.img)
			if err == nil {
				t.Fatalf("expected relocation offset error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestWritePE64WindowsX64RejectsRDataRelocTargetOutOfRange(t *testing.T) {
	err := WritePE64WindowsX64(t.TempDir()+"/bad-target.exe", &PEImage{
		Text:        []byte{0x90, 0x90, 0x90, 0x90},
		RData:       []byte("x"),
		Imports:     []string{"kernel32.ExitProcess"},
		RDataRelocs: []RDataReloc{{At: 0, TargetOff: 1}},
	})
	if err == nil {
		t.Fatalf("expected rdata relocation target error")
	}
	if !strings.Contains(err.Error(), "rdata relocation target out of range") {
		t.Fatalf("error = %v", err)
	}
}
