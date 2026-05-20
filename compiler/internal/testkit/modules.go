package testkit

import (
	"sort"
	"testing"
)

func AssertModules(t testing.TB, got []string, want []string) {
	t.Helper()
	gotSorted := append([]string(nil), got...)
	wantSorted := append([]string(nil), want...)
	sort.Strings(gotSorted)
	sort.Strings(wantSorted)
	if len(gotSorted) != len(wantSorted) {
		t.Fatalf("module list mismatch: got %v want %v", gotSorted, wantSorted)
	}
	for i := range gotSorted {
		if gotSorted[i] != wantSorted[i] {
			t.Fatalf("module list mismatch: got %v want %v", gotSorted, wantSorted)
		}
	}
}
