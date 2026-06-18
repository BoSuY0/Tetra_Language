package main

import "testing"

func TestSurfaceHostRejectsPositionalUIFileInput(t *testing.T) {
	if code := run([]string{"guest.png"}); code != 2 {
		t.Fatalf("run positional UI file exit = %d, want 2", code)
	}
}
