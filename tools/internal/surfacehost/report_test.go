package surfacehost

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReportingBackendRecordsFramesAndNativeEvents(t *testing.T) {
	backend := &recordingBackend{
		nextHandle: 3,
		events: []Event{
			{Kind: 4, X: 48, Y: 96, Button: 1},
			{Kind: 6, Key: 57},
			{Kind: 1},
		},
	}
	reporting := NewReportingBackend(backend, "wayland", "/run/user/1000/tetra.sock")
	reporting.RecordAppPID(4242)
	handle, err := reporting.Open("Counter", 2, 2)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := reporting.PresentRGBA(handle, 2, 2, 8, []byte{
		1, 2, 3, 4, 5, 6, 7, 8,
		9, 10, 11, 12, 13, 14, 15, 16,
	}); err != nil {
		t.Fatalf("present: %v", err)
	}
	for i := 0; i < 3; i++ {
		if _, err := reporting.PollEvent(handle); err != nil {
			t.Fatalf("poll %d: %v", i, err)
		}
	}
	if err := reporting.Close(handle); err != nil {
		t.Fatalf("close: %v", err)
	}
	reportPath := filepath.Join(t.TempDir(), "host-report.json")
	if err := reporting.WriteReport(reportPath); err != nil {
		t.Fatalf("write report: %v", err)
	}
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	var report HostReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if report.Schema != HostReportSchemaV1 || report.Protocol != ProtocolName {
		t.Fatalf("report identity = %#v", report)
	}
	if report.OpenCount != 1 || report.CloseCount != 1 || report.PresentedFrameCount != 1 {
		t.Fatalf("report lifecycle counts = %#v", report)
	}
	if report.AppPID != 4242 || report.HostPID <= 0 || report.AppPID == report.HostPID {
		t.Fatalf("report process identity = %#v", report)
	}
	if report.RealPointerEventCount != 1 || report.RealKeyEventCount != 1 ||
		report.RealCloseEventCount != 1 {
		t.Fatalf("report event counts = %#v", report)
	}
	if !strings.HasPrefix(report.LastFrameSHA256, "sha256:") || report.PreRenderedFrameSource {
		t.Fatalf("report frame provenance = %#v", report)
	}
	if len(report.Frames) != 1 {
		t.Fatalf("frames = %#v, want one detailed frame", report.Frames)
	}
	if frame := report.Frames[0]; frame.Order != 1 || frame.Width != 2 || frame.Height != 2 ||
		frame.Stride != 8 || !strings.HasPrefix(frame.SHA256, "sha256:") {
		t.Fatalf("frame detail = %#v", frame)
	}
	if len(report.Events) != 3 {
		t.Fatalf("events = %#v, want three detailed events", report.Events)
	}
	if report.Events[0].Kind != 4 || report.Events[1].Kind != 6 || report.Events[2].Kind != 1 {
		t.Fatalf("event details = %#v", report.Events)
	}
}

func TestReportingBackendSamplesOnlyChangedFrames(t *testing.T) {
	backend := &recordingBackend{nextHandle: 3}
	reporting := NewReportingBackend(backend, "wayland", "/run/user/1000/tetra.sock")
	handle, err := reporting.Open("Counter", 2, 2)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	first := []byte{
		1, 2, 3, 4, 5, 6, 7, 8,
		9, 10, 11, 12, 13, 14, 15, 16,
	}
	second := []byte{
		16, 15, 14, 13, 12, 11, 10, 9,
		8, 7, 6, 5, 4, 3, 2, 1,
	}
	if err := reporting.PresentRGBA(handle, 2, 2, 8, first); err != nil {
		t.Fatalf("present first: %v", err)
	}
	if err := reporting.PresentRGBA(handle, 2, 2, 8, first); err != nil {
		t.Fatalf("present duplicate: %v", err)
	}
	if err := reporting.PresentRGBA(handle, 2, 2, 8, second); err != nil {
		t.Fatalf("present changed: %v", err)
	}
	report := reporting.Snapshot()
	if report.PresentedFrameCount != 3 {
		t.Fatalf("presented frame count = %d, want 3", report.PresentedFrameCount)
	}
	if len(report.Frames) != 2 {
		t.Fatalf("sampled frames = %#v, want first and changed frame", report.Frames)
	}
	if report.Frames[0].Order != 1 || report.Frames[1].Order != 3 {
		t.Fatalf("sampled frame orders = %#v, want orders 1 and 3", report.Frames)
	}
	if report.Frames[0].SHA256 == report.Frames[1].SHA256 {
		t.Fatalf("sampled frame checksums should differ: %#v", report.Frames)
	}
}
