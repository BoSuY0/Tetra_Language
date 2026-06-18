package surfacehost

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

const HostReportSchemaV1 = "tetra.surface.host-report.v1"

type HostReport struct {
	Schema                 string            `json:"schema"`
	Host                   string            `json:"host"`
	Protocol               string            `json:"protocol"`
	AppPID                 int               `json:"app_pid,omitempty"`
	HostPID                int               `json:"host_pid"`
	SocketPath             string            `json:"socket_path"`
	OpenCount              int               `json:"open_count"`
	CloseCount             int               `json:"close_count"`
	PresentedFrameCount    int               `json:"presented_frame_count"`
	LastFrameSHA256        string            `json:"last_frame_sha256,omitempty"`
	Frames                 []HostFrameReport `json:"frames,omitempty"`
	RealPointerEventCount  int               `json:"real_pointer_event_count"`
	RealKeyEventCount      int               `json:"real_key_event_count"`
	RealCloseEventCount    int               `json:"real_close_event_count"`
	Events                 []HostEventReport `json:"events,omitempty"`
	PreRenderedFrameSource bool              `json:"pre_rendered_frame_source"`
	DeliveryPath           string            `json:"delivery_path"`
}

type HostFrameReport struct {
	Order  int    `json:"order"`
	Width  int32  `json:"width"`
	Height int32  `json:"height"`
	Stride int32  `json:"stride"`
	SHA256 string `json:"sha256"`
}

type HostEventReport struct {
	Order       int   `json:"order"`
	Kind        int32 `json:"kind"`
	X           int32 `json:"x"`
	Y           int32 `json:"y"`
	Button      int32 `json:"button"`
	Key         int32 `json:"key"`
	Width       int32 `json:"width"`
	Height      int32 `json:"height"`
	TimestampMS int32 `json:"timestamp_ms"`
	TextLen     int32 `json:"text_len"`
}

type ReportingBackend struct {
	backend Backend
	mu      sync.Mutex
	report  HostReport
}

type AppPIDRecorder interface {
	RecordAppPID(pid int)
}

func NewReportingBackend(backend Backend, host string, socketPath string) *ReportingBackend {
	return &ReportingBackend{
		backend: backend,
		report: HostReport{
			Schema:                 HostReportSchemaV1,
			Host:                   host,
			Protocol:               ProtocolName,
			HostPID:                os.Getpid(),
			SocketPath:             socketPath,
			PreRenderedFrameSource: false,
			DeliveryPath:           "compiled-tetra-app-to-wayland-surface",
		},
	}
}

func (b *ReportingBackend) RecordAppPID(pid int) {
	if pid <= 0 {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.report.AppPID = pid
}

func (b *ReportingBackend) Open(title string, width int32, height int32) (uint32, error) {
	handle, err := b.backend.Open(title, width, height)
	if err == nil {
		b.mu.Lock()
		b.report.OpenCount++
		b.mu.Unlock()
	}
	return handle, err
}

func (b *ReportingBackend) Close(handle uint32) error {
	err := b.backend.Close(handle)
	if err == nil {
		b.mu.Lock()
		b.report.CloseCount++
		b.mu.Unlock()
	}
	return err
}

func (b *ReportingBackend) BeginFrame(handle uint32) error {
	return b.backend.BeginFrame(handle)
}

func (b *ReportingBackend) PresentRGBA(
	handle uint32,
	width int32,
	height int32,
	stride int32,
	rgba []byte,
) error {
	err := b.backend.PresentRGBA(handle, width, height, stride, rgba)
	if err == nil {
		sum := sha256.Sum256(rgba)
		digest := "sha256:" + hex.EncodeToString(sum[:])
		b.mu.Lock()
		b.report.PresentedFrameCount++
		b.report.LastFrameSHA256 = digest
		if len(b.report.Frames) == 0 || b.report.Frames[len(b.report.Frames)-1].SHA256 != digest {
			b.report.Frames = append(b.report.Frames, HostFrameReport{
				Order:  b.report.PresentedFrameCount,
				Width:  width,
				Height: height,
				Stride: stride,
				SHA256: digest,
			})
		}
		b.mu.Unlock()
	}
	return err
}

func (b *ReportingBackend) PollEvent(handle uint32) (Event, error) {
	event, err := b.backend.PollEvent(handle)
	if err == nil {
		b.recordEvent(event)
	}
	return event, err
}

func (b *ReportingBackend) PollEventText(handle uint32) ([]byte, error) {
	return b.backend.PollEventText(handle)
}

func (b *ReportingBackend) ClipboardWriteText(handle uint32, text []byte) (int32, error) {
	return b.backend.ClipboardWriteText(handle, text)
}

func (b *ReportingBackend) ClipboardReadText(handle uint32) ([]byte, error) {
	return b.backend.ClipboardReadText(handle)
}

func (b *ReportingBackend) PollComposition(handle uint32) ([4]int32, error) {
	return b.backend.PollComposition(handle)
}

func (b *ReportingBackend) NowMS() int32 {
	return b.backend.NowMS()
}

func (b *ReportingBackend) RequestRedraw(handle uint32) error {
	return b.backend.RequestRedraw(handle)
}

func (b *ReportingBackend) Snapshot() HostReport {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.report
}

func (b *ReportingBackend) WriteReport(path string) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(b.Snapshot(), "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o644)
}

func (b *ReportingBackend) recordEvent(event Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if event.Kind != 0 {
		b.report.Events = append(b.report.Events, HostEventReport{
			Order:       len(b.report.Events) + 1,
			Kind:        event.Kind,
			X:           event.X,
			Y:           event.Y,
			Button:      event.Button,
			Key:         event.Key,
			Width:       event.Width,
			Height:      event.Height,
			TimestampMS: event.TimestampMS,
			TextLen:     event.TextLen,
		})
	}
	switch event.Kind {
	case 1:
		b.report.RealCloseEventCount++
	case 3, 4, 5:
		b.report.RealPointerEventCount++
	case 6, 7:
		b.report.RealKeyEventCount++
	}
}
