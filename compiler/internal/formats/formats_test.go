package formats

import "testing"

func TestModuleCandidateRelPathsIncludesActorsStdlibBucket(t *testing.T) {
	candidates := ModuleCandidateRelPaths("lib.core.actors")
	want := "lib/core/actors/actors.tetra"
	for _, candidate := range candidates {
		if candidate == want {
			return
		}
	}
	t.Fatalf("ModuleCandidateRelPaths(lib.core.actors) = %#v, want %q", candidates, want)
}
