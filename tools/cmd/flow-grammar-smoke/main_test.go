package main

import (
	"strings"
	"testing"
)

func TestGenerateSmokeSourceCoversSupportedFamilies(t *testing.T) {
	src := generateSmokeSource()
	for _, want := range []string{
		"module generated.flow_grammar_smoke",
		"enum SmokeColor",
		"struct Pair",
		"func id<T>",
		"protocol Drawable",
		"extension Pair",
		"async func worker",
		"state CounterState",
		"view CounterView",
		"test \"generated grammar smoke\"",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("generated source missing %q:\n%s", want, src)
		}
	}
}
