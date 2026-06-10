package actorsrt

import (
	"bytes"
	"crypto/sha256"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

func TestSelfhostActorRuntimeSourcesMatchCanonicalRT(t *testing.T) {
	root := repoRootFromActorsRTTest(t)
	canonicalDir := filepath.Join(root, "__rt")
	selfhostDir := filepath.Join(root, "compiler", "selfhostrt")

	canonical, err := filepath.Glob(filepath.Join(canonicalDir, "actors_*.tetra"))
	if err != nil {
		t.Fatalf("glob canonical actor runtime files: %v", err)
	}
	if len(canonical) == 0 {
		t.Fatalf("no canonical actor runtime files found under %s", canonicalDir)
	}
	sort.Strings(canonical)

	for _, canonicalPath := range canonical {
		name := filepath.Base(canonicalPath)
		selfhostPath := filepath.Join(selfhostDir, name)
		t.Run(name, func(t *testing.T) {
			canonicalRaw, err := os.ReadFile(canonicalPath)
			if err != nil {
				t.Fatalf("read canonical runtime source: %v", err)
			}
			selfhostRaw, err := os.ReadFile(selfhostPath)
			if err != nil {
				t.Fatalf("read selfhost runtime source: %v", err)
			}
			if !bytes.Equal(canonicalRaw, selfhostRaw) {
				canonicalSum := sha256.Sum256(canonicalRaw)
				selfhostSum := sha256.Sum256(selfhostRaw)
				t.Fatalf("selfhost actor runtime source drift for %s: __rt sha256=%x selfhostrt sha256=%x", name, canonicalSum, selfhostSum)
			}
		})
	}

	selfhost, err := filepath.Glob(filepath.Join(selfhostDir, "actors_*.tetra"))
	if err != nil {
		t.Fatalf("glob selfhost actor runtime files: %v", err)
	}
	canonicalNames := map[string]bool{}
	for _, path := range canonical {
		canonicalNames[filepath.Base(path)] = true
	}
	for _, path := range selfhost {
		name := filepath.Base(path)
		if !canonicalNames[name] {
			t.Fatalf("selfhost actor runtime source %s has no canonical __rt peer", name)
		}
	}
}

func repoRootFromActorsRTTest(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "__rt")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, "compiler", "selfhostrt")); err == nil {
				return dir
			}
		}
		if _, err := os.Stat(filepath.Join(dir, "AGENTS.md")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find repo root from %s", file)
		}
		if strings.TrimSpace(parent) == "" {
			t.Fatalf("invalid parent while walking from %s", file)
		}
		dir = parent
	}
}
