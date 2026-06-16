package postv04

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleasePostV04LinuxNativeTargetScriptsPassArtifactHashesToValidator(t *testing.T) {
	root := repoRoot(t)
	for _, name := range []string{
		"linux-native-targets-smoke.sh",
		"linux-x86-smoke.sh",
		"linux-x32-smoke.sh",
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(root, "scripts", "release", "post_v0_4", name)
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", name, err)
			}
			text := string(raw)
			for _, want := range []string{
				`go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"`,
				`go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"`,
				`go run ./tools/cmd/validate-linux-native-targets`,
				`--artifact-hashes "$report_dir/artifact-hashes.json"`,
			} {
				if !strings.Contains(text, want) {
					t.Fatalf("%s missing %q", name, want)
				}
			}
			hashIdx := strings.Index(text, `go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"`)
			validatorIdx := strings.Index(text, `go run ./tools/cmd/validate-linux-native-targets`)
			if hashIdx < 0 || validatorIdx < 0 || hashIdx > validatorIdx {
				t.Fatalf("%s must validate artifact hashes before validate-linux-native-targets", name)
			}
		})
	}
}

func TestReleasePostV04LinuxNativeTargetScriptsUsePersistentGoCache(t *testing.T) {
	root := repoRoot(t)
	for _, name := range []string{
		"linux-native-targets-smoke.sh",
		"linux-x86-smoke.sh",
		"linux-x32-smoke.sh",
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(root, "scripts", "release", "post_v0_4", name)
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", name, err)
			}
			text := string(raw)
			for _, want := range []string{
				`: "${GOCACHE:=$repo_root/.cache/go-build-`,
				`export GOCACHE`,
			} {
				if !strings.Contains(text, want) {
					t.Fatalf("%s missing persistent Go cache setup %q", name, want)
				}
			}
			cacheIdx := strings.Index(text, `export GOCACHE`)
			firstGoRunIdx := strings.Index(text, `go run `)
			if cacheIdx < 0 || firstGoRunIdx < 0 || cacheIdx > firstGoRunIdx {
				t.Fatalf("%s must export GOCACHE before the first go run", name)
			}
			for _, forbidden := range []string{
				`GOCACHE=/tmp`,
				`GOCACHE="${TMPDIR`,
				`GOCACHE="$TMPDIR`,
			} {
				if strings.Contains(text, forbidden) {
					t.Fatalf("%s must not use tmpfs Go cache pattern %q", name, forbidden)
				}
			}
		})
	}
}

func TestReleasePostV04LinuxNativeTargetScriptsDefaultToCurrentSourceCLI(t *testing.T) {
	root := repoRoot(t)
	for _, name := range []string{
		"linux-native-targets-smoke.sh",
		"linux-x86-smoke.sh",
		"linux-x32-smoke.sh",
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(root, "scripts", "release", "post_v0_4", name)
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", name, err)
			}
			text := string(raw)
			for _, want := range []string{
				`if [[ -n "${TETRA_CMD:-}" ]]; then`,
				`read -r -a tetra_cmd <<<"$TETRA_CMD"`,
				`tetra_cmd=(go run ./cli/cmd/tetra)`,
			} {
				if !strings.Contains(text, want) {
					t.Fatalf("%s missing source CLI command setup %q", name, want)
				}
			}
			for _, forbidden := range []string{
				`tetra_cmd=(./tetra)`,
				`elif [[ -x ./tetra ]]`,
			} {
				if strings.Contains(text, forbidden) {
					t.Fatalf("%s must not default release evidence to possibly stale local binary via %q", name, forbidden)
				}
			}
		})
	}
}

func TestReleasePostV04LinuxNativeTargetScriptsUseRuntimeRunnerSmoke(t *testing.T) {
	root := repoRoot(t)
	for _, name := range []string{
		"linux-native-targets-smoke.sh",
		"linux-x86-smoke.sh",
		"linux-x32-smoke.sh",
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(root, "scripts", "release", "post_v0_4", name)
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", name, err)
			}
			text := string(raw)
			for _, want := range []string{
				`func runner_worker() -> Int:`,
				`test "runner arithmetic":`,
				`test "runner alloc memory":`,
				`test "runner filesystem":`,
				`test "runner stderr fd":`,
				`test "runner time":`,
				`test "runner network socket":`,
				`test "runner network options":`,
				`test "runner task join":`,
				`core.alloc_bytes(4)`,
				`core.fs_exists("README.md", cap)`,
				`core.net_write(2, buf, 0, 1, cap)`,
				`core.time_now_ms()`,
				`core.net_socket_tcp4(cap)`,
				`core.net_set_nonblocking(fd, cap)`,
				`core.net_set_reuseport(fd, cap)`,
				`core.net_close(fd, cap)`,
				`core.task_spawn_i32("runner_worker")`,
			} {
				if !strings.Contains(text, want) {
					t.Fatalf("%s missing runtime runner smoke source %q", name, want)
				}
			}
		})
	}
}
