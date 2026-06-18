package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/tools/internal/surfacehost"
	"tetra_language/tools/validators/surface"
)

func TestBuildNativeSurfaceHostRuntimeReportFromHostReport(t *testing.T) {
	dir := t.TempDir()
	artifactDir := filepath.Join(dir, "artifacts")
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		t.Fatalf("create artifact dir: %v", err)
	}
	appPath := writeNativeReportFixture(
		t,
		artifactDir,
		"surface-window-counter",
		[]byte("compiled app\n"),
		0o755,
	)
	hostPath := writeNativeReportFixture(
		t,
		artifactDir,
		"tetra-surface-host-wayland",
		[]byte("host binary\n"),
		0o755,
	)
	hostReportPath := filepath.Join(artifactDir, "surface-host-report.json")
	hostReport := surfacehost.HostReport{
		Schema:              surfacehost.HostReportSchemaV1,
		Host:                "wayland",
		Protocol:            surfacehost.ProtocolName,
		AppPID:              4242,
		HostPID:             4243,
		SocketPath:          "/run/user/1000/tetra-surface-host.sock",
		OpenCount:           1,
		CloseCount:          1,
		PresentedFrameCount: 2,
		LastFrameSHA256:     "sha256:" + strings.Repeat("b", 64),
		Frames: []surfacehost.HostFrameReport{
			{
				Order:  1,
				Width:  320,
				Height: 200,
				Stride: 1280,
				SHA256: "sha256:" + strings.Repeat("a", 64),
			},
			{
				Order:  2,
				Width:  320,
				Height: 200,
				Stride: 1280,
				SHA256: "sha256:" + strings.Repeat("b", 64),
			},
		},
		RealPointerEventCount: 1,
		RealKeyEventCount:     1,
		RealCloseEventCount:   1,
		Events: []surfacehost.HostEventReport{
			{Order: 1, Kind: 5, X: 48, Y: 96, Button: 1, Width: 320, Height: 200},
			{Order: 2, Kind: 6, Key: 32, Width: 320, Height: 200, TimestampMS: 1},
			{Order: 3, Kind: 1, Width: 320, Height: 200, TimestampMS: 2},
		},
		PreRenderedFrameSource: false,
		DeliveryPath:           "compiled-tetra-app-to-wayland-surface",
	}
	rawHost, err := json.Marshal(hostReport)
	if err != nil {
		t.Fatalf("marshal host report: %v", err)
	}
	if err := os.WriteFile(hostReportPath, rawHost, 0o644); err != nil {
		t.Fatalf("write host report: %v", err)
	}
	report, err := buildNativeSurfaceHostRuntimeReport(nativeSurfaceHostReportOptions{
		Source:       "examples/surface/runtime/surface_window_counter.tetra",
		ArtifactDir:  artifactDir,
		ComponentApp: appPath,
		HostBinary:   hostPath,
		HostReport:   hostReportPath,
		AppExitCode:  0,
		HostExitCode: 0,
		BuildCommand: "tetra build --target linux-x64 " +
			"examples/surface/runtime/surface_window_counter.tetra -o " + appPath,
		AppCommand: appPath + " --surface-host wayland",
		HostCommand: hostPath +
			" --socket /run/user/1000/tetra-surface-host.sock --report " + hostReportPath,
	})
	if err != nil {
		t.Fatalf("build report: %v", err)
	}
	rawReport, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	if err := surface.ValidateReport(rawReport); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, rawReport)
	}
	if report.NativeSurfaceHost == nil || report.NativeSurfaceHost.AppPID != 4242 {
		t.Fatalf("native_surface_host = %#v", report.NativeSurfaceHost)
	}
	if len(report.Frames) != 2 || report.Frames[0].EvidenceRole != "native-surface-live-frame" {
		t.Fatalf("frames = %#v", report.Frames)
	}
}

func writeNativeReportFixture(
	t *testing.T,
	dir string,
	name string,
	contents []byte,
	perm os.FileMode,
) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, contents, perm); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}
