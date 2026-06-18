package linkcore

import (
	"fmt"
	"math/rand"
	"testing"

	"tetra_language/compiler/internal/format/tobj"
)

func FuzzLinkX64ObjectsDoesNotPanic(f *testing.F) {
	f.Add(uint64(1), uint8(3))
	f.Add(uint64(2), uint8(6))
	f.Add(uint64(123), uint8(8))

	stub := []byte{0xE8, 0, 0, 0, 0, 0xC3}
	stubCallAt := 1

	f.Fuzz(func(t *testing.T, seed uint64, objCountU8 uint8) {
		objCount := int(objCountU8%10) + 1
		if objCount < 2 {
			objCount = 2
		}

		rng := rand.New(rand.NewSource(int64(seed)))
		mainIdx := rng.Intn(objCount)

		var objects []*tobj.Object
		for i := 0; i < objCount; i++ {
			mod := fmt.Sprintf("m%02d_%08x", i, rng.Uint32())
			if i == mainIdx {
				objects = append(objects, &tobj.Object{
					Target:  "linux-x64",
					Module:  mod,
					Code:    []byte{0xC3},
					Symbols: []tobj.Symbol{{Name: "main", Offset: 0}},
				})
				continue
			}

			// Mix caller objects: some call main, some only have data relocs.
			kind := rng.Intn(2)
			switch kind {
			case 0:
				objects = append(objects, &tobj.Object{
					Target:  "linux-x64",
					Module:  mod,
					Code:    []byte{0xE8, 0, 0, 0, 0, 0xC3},
					Symbols: []tobj.Symbol{{Name: fmt.Sprintf("caller_%s", mod), Offset: 0}},
					Relocs:  []tobj.Reloc{{Kind: tobj.RelocCallRel32, At: 1, Name: "main"}},
				})
			case 1:
				data := []byte{byte(rng.Uint32()), byte(rng.Uint32())}
				objects = append(objects, &tobj.Object{
					Target:  "linux-x64",
					Module:  mod,
					Code:    []byte{0x48, 0x8D, 0x05, 0, 0, 0, 0, 0xC3},
					Data:    data,
					Symbols: []tobj.Symbol{{Name: fmt.Sprintf("sym_%s", mod), Offset: 0}},
					Relocs: []tobj.Reloc{
						{Kind: tobj.RelocDataDisp32, At: 3, Addend: uint32(rng.Intn(len(data)))},
					},
				})
			}
		}
		rng.Shuffle(
			len(objects),
			func(i, j int) { objects[i], objects[j] = objects[j], objects[i] },
		)

		res, err := LinkX64Objects(objects, "main", stub, stubCallAt, 0)
		if err != nil {
			// Errors are fine; the contract is “no panic”.
			return
		}

		// Basic invariants.
		if res.EntryOffset != 0 {
			t.Fatalf("unexpected entry offset: %d", res.EntryOffset)
		}
		if len(res.Text) < len(stub) {
			t.Fatalf("text too small: %d", len(res.Text))
		}
		for _, r := range res.DataRelocs {
			if r.At < 0 || r.At+4 > len(res.Text) {
				t.Fatalf("data reloc out of range: at=%d text=%d", r.At, len(res.Text))
			}
			if r.TargetOff < 0 || r.TargetOff >= len(res.Data) {
				t.Fatalf(
					"data reloc target out of range: off=%d data=%d",
					r.TargetOff,
					len(res.Data),
				)
			}
		}
		for _, r := range res.IATRelocs {
			if r.At < 0 || r.At+4 > len(res.Text) {
				t.Fatalf("iat reloc out of range: at=%d text=%d", r.At, len(res.Text))
			}
			if r.Name == "" {
				t.Fatalf("iat reloc name empty")
			}
		}
	})
}
