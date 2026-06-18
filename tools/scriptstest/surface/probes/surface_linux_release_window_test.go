package surface_probes

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestLinuxRealWindowRequiresWayland(t *testing.T) {
	root := t.TempDir()
	scriptDir := filepath.Join(root, "scripts", "release", "surface")
	if err := os.MkdirAll(scriptDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(
		filepath.Join(
			repoRoot(t),
			"scripts",
			"release",
			"surface",
			"surface-linux-x64-release-window-smoke.sh",
		),
		filepath.Join(scriptDir, "surface-linux-x64-release-window-smoke.sh"),
		0o755,
	); err != nil {
		t.Fatalf("copy linux release-window smoke: %v", err)
	}

	reportRel := filepath.ToSlash(filepath.Join("reports", "linux-window"))
	cmd := exec.Command(
		"bash",
		"scripts/release/surface/surface-linux-x64-release-window-smoke.sh",
		"--report-dir",
		reportRel,
	)
	cmd.Dir = root
	cmd.Env = envWithoutDisplay(root)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected missing display to block linux release-window smoke\n%s", out)
	}
	for _, want := range []string{
		"blocked",
		"WAYLAND_DISPLAY or DISPLAY",
		"Surface linux-x64 release-window blocked report:",
	} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("blocked output missing %q:\n%s", want, out)
		}
	}
	assertOutputAvoidsRawPathUtilityErrors(t, out)

	reportPath := filepath.Join(
		root,
		filepath.FromSlash(reportRel),
		"surface-linux-x64-release-window.json",
	)
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("expected blocked report %s: %v\n%s", reportPath, err, out)
	}
	for _, want := range []string{
		`"status": "blocked"`,
		`"blocked_reason": "missing WAYLAND_DISPLAY or DISPLAY for linux-x64 release-window target host"`,
		`"production_claim": false`,
	} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("blocked report missing %q:\n%s", want, raw)
		}
	}
}

func envWithoutDisplay(root string) []string {
	env := make([]string, 0, len(os.Environ())+2)
	for _, entry := range os.Environ() {
		if strings.HasPrefix(entry, "WAYLAND_DISPLAY=") || strings.HasPrefix(entry, "DISPLAY=") ||
			strings.HasPrefix(entry, "GOCACHE=") {
			continue
		}
		env = append(env, entry)
	}
	env = append(env,
		"GOTELEMETRY=off",
		"GOCACHE="+filepath.Join(root, ".cache", "go-build-surface-p04-scriptstest"),
	)
	return env
}
