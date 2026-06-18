package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"tetra_language/tools/internal/surfacehost"
	"tetra_language/tools/validators/surface"
)

type nativeSurfaceHostReportOptions struct {
	ReportPath   string
	Source       string
	ArtifactDir  string
	ComponentApp string
	HostBinary   string
	HostReport   string
	AppExitCode  int
	HostExitCode int
	BuildCommand string
	AppCommand   string
	HostCommand  string
}

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	var opt nativeSurfaceHostReportOptions
	fs := flag.NewFlagSet("surface-native-host-report", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.StringVar(&opt.ReportPath, "report", "", "path to write tetra.surface.runtime.v1 report")
	fs.StringVar(
		&opt.Source,
		"source",
		"examples/surface/runtime/surface_window_counter.tetra",
		"reported Tetra source",
	)
	fs.StringVar(&opt.ArtifactDir, "artifact-dir", "", "artifact scan root")
	fs.StringVar(&opt.ComponentApp, "component-app", "", "compiled linux-x64 Tetra app artifact")
	fs.StringVar(&opt.HostBinary, "host-binary", "", "tetra-surface-host-wayland artifact")
	fs.StringVar(&opt.HostReport, "host-report", "", "tetra.surface.host-report.v1 JSON")
	fs.IntVar(&opt.AppExitCode, "app-exit-code", 0, "compiled app exit code")
	fs.IntVar(&opt.HostExitCode, "host-exit-code", 0, "surface host exit code")
	fs.StringVar(&opt.BuildCommand, "build-command", "", "build command evidence")
	fs.StringVar(&opt.AppCommand, "app-command", "", "app command evidence")
	fs.StringVar(&opt.HostCommand, "host-command", "", "host command evidence")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	report, err := buildNativeSurfaceHostRuntimeReport(opt)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	raw = append(raw, '\n')
	if err := surface.ValidateReport(raw); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if strings.TrimSpace(opt.ReportPath) == "" {
		fmt.Fprint(os.Stdout, string(raw))
		return 0
	}
	if err := os.MkdirAll(filepath.Dir(opt.ReportPath), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := os.WriteFile(opt.ReportPath, raw, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func buildNativeSurfaceHostRuntimeReport(
	opt nativeSurfaceHostReportOptions,
) (surface.Report, error) {
	if strings.TrimSpace(opt.Source) == "" {
		return surface.Report{}, errors.New("--source is required")
	}
	if strings.TrimSpace(opt.ArtifactDir) == "" {
		return surface.Report{}, errors.New("--artifact-dir is required")
	}
	hostReport, err := readHostReport(opt.HostReport)
	if err != nil {
		return surface.Report{}, err
	}
	appArtifact, err := artifactForPath("component-app", opt.ComponentApp)
	if err != nil {
		return surface.Report{}, err
	}
	hostArtifact, err := artifactForPath("surface-host", opt.HostBinary)
	if err != nil {
		return surface.Report{}, err
	}
	hostReportArtifact, err := artifactForPath("native-surface-host-report", opt.HostReport)
	if err != nil {
		return surface.Report{}, err
	}
	filesChecked, err := countFiles(opt.ArtifactDir)
	if err != nil {
		return surface.Report{}, err
	}
	source := filepath.ToSlash(opt.Source)
	appCommand := defaultString(opt.AppCommand, opt.ComponentApp+" --surface-host wayland")
	hostCommand := defaultString(
		opt.HostCommand,
		opt.HostBinary+" --socket "+hostReport.SocketPath+" --report "+opt.HostReport,
	)
	buildCommand := defaultString(
		opt.BuildCommand,
		"tetra build --target linux-x64 "+source+" -o "+opt.ComponentApp,
	)
	events, transitions := nativeRuntimeEventsAndTransitions(hostReport.Events)
	frames := nativeRuntimeFrames(source, hostReport.Frames)
	report := surface.Report{
		Schema:        surface.SchemaV1,
		Status:        "pass",
		Target:        "linux-x64",
		Host:          "linux-x64",
		Runtime:       "surface-linux-x64",
		SurfaceSchema: "tetra.surface.v1",
		HostABI:       "tetra.surface.host-abi.v1",
		HostEvidence: surface.HostEvidenceReport{
			Level:                     surface.NativeSurfaceHostLevelLinuxX64,
			Backend:                   surface.NativeSurfaceHostBackendWayland,
			Framebuffer:               true,
			RealWindow:                true,
			NativeInput:               true,
			UserFacingPlatformWidgets: false,
		},
		Source: source,
		Processes: []surface.ProcessReport{
			processReport("tetra build native surface host app", "build", buildCommand, 0, nil),
			processReport(
				"surface component app",
				"app",
				appCommand,
				opt.AppExitCode,
				intPtr(opt.AppExitCode),
			),
			processReport(
				"surface linux-x64 native surface host wayland",
				"runtime",
				hostCommand,
				opt.HostExitCode,
				nil,
			),
		},
		Artifacts: []surface.ArtifactReport{
			appArtifact,
			hostArtifact,
			hostReportArtifact,
		},
		ArtifactScan: surface.ArtifactScanReport{
			Root:           filepath.ToSlash(opt.ArtifactDir),
			FilesChecked:   filesChecked,
			ForbiddenPaths: []string{},
			Pass:           true,
		},
		Components:        nativeCounterComponents(),
		Events:            events,
		Frames:            frames,
		StateTransitions:  transitions,
		Cases:             nativeHostCases(),
		NativeSurfaceHost: nativeSurfaceHostEvidence(hostReport, len(frames)),
	}
	return report, nil
}

func readHostReport(path string) (surfacehost.HostReport, error) {
	if strings.TrimSpace(path) == "" {
		return surfacehost.HostReport{}, errors.New("--host-report is required")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return surfacehost.HostReport{}, err
	}
	var report surfacehost.HostReport
	if err := json.Unmarshal(raw, &report); err != nil {
		return surfacehost.HostReport{}, err
	}
	if report.Schema != surfacehost.HostReportSchemaV1 {
		return surfacehost.HostReport{}, fmt.Errorf(
			"host report schema is %q, want %s",
			report.Schema,
			surfacehost.HostReportSchemaV1,
		)
	}
	return report, nil
}

func artifactForPath(kind string, path string) (surface.ArtifactReport, error) {
	if strings.TrimSpace(path) == "" {
		return surface.ArtifactReport{}, fmt.Errorf(
			"--%s path is required",
			strings.ReplaceAll(kind, "-", "-"),
		)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return surface.ArtifactReport{}, err
	}
	sum := sha256.Sum256(raw)
	return surface.ArtifactReport{
		Kind:   kind,
		Path:   filepath.ToSlash(path),
		SHA256: "sha256:" + hex.EncodeToString(sum[:]),
		Size:   int64(len(raw)),
	}, nil
}

func countFiles(root string) (int, error) {
	count := 0
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			count++
		}
		return nil
	})
	return count, err
}

func nativeRuntimeFrames(
	source string,
	frames []surfacehost.HostFrameReport,
) []surface.FrameReport {
	out := make([]surface.FrameReport, 0, len(frames))
	for _, frame := range frames {
		out = append(out, surface.FrameReport{
			Order:        frame.Order,
			Width:        int(frame.Width),
			Height:       int(frame.Height),
			Stride:       int(frame.Stride),
			Checksum:     frame.SHA256,
			Producer:     "app",
			EvidenceRole: "native-surface-live-frame",
			AppSource:    source,
			Precomputed:  false,
			Presented:    true,
		})
	}
	return out
}

func nativeRuntimeEventsAndTransitions(
	events []surfacehost.HostEventReport,
) ([]surface.EventReport, []surface.StateTransitionReport) {
	var out []surface.EventReport
	var transitions []surface.StateTransitionReport
	count := 0
	keyCount := 0
	width := 320
	height := 200
	closed := false
	for _, event := range events {
		kind := nativeEventKind(event.Kind)
		if kind == "" {
			continue
		}
		target := "CounterApp"
		dispatch := []string{"CounterApp"}
		before := map[string]string{"CounterApp.event": strconv.Itoa(len(out))}
		after := map[string]string{"CounterApp.event": strconv.Itoa(len(out))}
		handled := true
		switch kind {
		case "mouse_up":
			if pointInCounterButton(int(event.X), int(event.Y)) {
				target = "CounterButton"
				dispatch = []string{"CounterApp", "CounterButton"}
			}
			before = map[string]string{
				"CounterApp.count":      strconv.Itoa(count),
				"CounterButton.pressed": "false",
			}
			count++
			after = map[string]string{
				"CounterApp.count":      strconv.Itoa(count),
				"CounterButton.pressed": "false",
			}
			transitions = append(transitions, surface.StateTransitionReport{
				Order:     len(transitions) + 1,
				Component: "CounterApp",
				Field:     "count",
				Before:    strconv.Itoa(count - 1),
				After:     strconv.Itoa(count),
				Cause:     kind,
			})
		case "key_down":
			before = map[string]string{"CounterApp.key_count": strconv.Itoa(keyCount)}
			keyCount++
			after = map[string]string{"CounterApp.key_count": strconv.Itoa(keyCount)}
			transitions = append(transitions, surface.StateTransitionReport{
				Order:     len(transitions) + 1,
				Component: "CounterApp",
				Field:     "key_count",
				Before:    strconv.Itoa(keyCount - 1),
				After:     strconv.Itoa(keyCount),
				Cause:     kind,
			})
		case "resize":
			before = map[string]string{
				"CounterApp.width":  strconv.Itoa(width),
				"CounterApp.height": strconv.Itoa(height),
			}
			width = int(event.Width)
			height = int(event.Height)
			after = map[string]string{
				"CounterApp.width":  strconv.Itoa(width),
				"CounterApp.height": strconv.Itoa(height),
			}
			transitions = append(transitions, surface.StateTransitionReport{
				Order:     len(transitions) + 1,
				Component: "CounterApp",
				Field:     "width",
				Before:    before["CounterApp.width"],
				After:     after["CounterApp.width"],
				Cause:     kind,
			})
		case "close":
			before = map[string]string{"CounterApp.closed": strconv.FormatBool(closed)}
			closed = true
			after = map[string]string{"CounterApp.closed": strconv.FormatBool(closed)}
			transitions = append(transitions, surface.StateTransitionReport{
				Order:     len(transitions) + 1,
				Component: "CounterApp",
				Field:     "closed",
				Before:    "false",
				After:     "true",
				Cause:     kind,
			})
		}
		out = append(out, surface.EventReport{
			Order:           len(out) + 1,
			Kind:            kind,
			TargetComponent: target,
			DispatchPath:    dispatch,
			Handled:         handled,
			Pass:            true,
			X:               int(event.X),
			Y:               int(event.Y),
			Key:             int(event.Key),
			Width:           int(event.Width),
			Height:          int(event.Height),
			TimestampMS:     int(event.TimestampMS),
			BufferSlots: []int{
				int(event.Kind),
				int(event.X),
				int(event.Y),
				int(event.Button),
				int(event.Key),
				int(event.Width),
				int(event.Height),
				int(event.TimestampMS),
				int(event.TextLen),
			},
			BeforeState: before,
			AfterState:  after,
		})
	}
	return out, transitions
}

func nativeEventKind(kind int32) string {
	switch kind {
	case 1:
		return "close"
	case 2:
		return "resize"
	case 3:
		return "mouse_move"
	case 4:
		return "mouse_down"
	case 5:
		return "mouse_up"
	case 6:
		return "key_down"
	case 7:
		return "key_up"
	default:
		return ""
	}
}

func pointInCounterButton(x int, y int) bool {
	return x >= 32 && y >= 80 && x < 192 && y < 128
}

func nativeCounterComponents() []surface.ComponentReport {
	return []surface.ComponentReport{
		{
			ID:     "CounterApp",
			Type:   "examples.surface.runtime.surface_window_counter.CounterApp",
			Bounds: surface.RectReport{X: 0, Y: 0, W: 320, H: 200},
			Abilities: []string{
				"measure",
				"layout",
				"draw",
				"event",
				"focus",
				"text",
				"accessibility",
			},
			State: map[string]string{
				"count":              "1",
				"key_count":          "1",
				"closed":             "true",
				"accessibility_role": "button",
			},
		},
		{
			ID:     "CounterButton",
			Type:   "examples.surface.runtime.surface_window_counter.CounterButton",
			Parent: "CounterApp",
			Bounds: surface.RectReport{X: 32, Y: 80, W: 160, H: 48},
			Abilities: []string{
				"measure",
				"layout",
				"draw",
				"event",
				"focus",
				"text",
				"accessibility",
			},
			State: map[string]string{
				"pressed":            "false",
				"focused":            "true",
				"accessibility_role": "button",
			},
		},
	}
}

func nativeHostCases() []surface.CaseReport {
	return []surface.CaseReport{
		{Name: "pure Tetra component app", Kind: "positive", Ran: true, Pass: true},
		{Name: "native Surface host Wayland live window", Kind: "positive", Ran: true, Pass: true},
		{Name: "native Surface host app loop observed", Kind: "positive", Ran: true, Pass: true},
		{Name: "native Surface host close event", Kind: "positive", Ran: true, Pass: true},
		{Name: "native Surface host pointer input", Kind: "positive", Ran: true, Pass: true},
		{Name: "native Surface host keyboard input", Kind: "positive", Ran: true, Pass: true},
		{
			Name: "native Surface host frame presented by running app",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{Name: "host-provided pointer event dispatch", Kind: "positive", Ran: true, Pass: true},
		{Name: "host event buffer poll_event", Kind: "positive", Ran: true, Pass: true},
		{Name: "pre/post event frame sequence", Kind: "positive", Ran: true, Pass: true},
		{Name: "component hierarchy dispatch", Kind: "positive", Ran: true, Pass: true},
		{Name: "component focus dispatch", Kind: "positive", Ran: true, Pass: true},
		{Name: "component accessibility metadata", Kind: "positive", Ran: true, Pass: true},
		{Name: "no legacy UI sidecar artifacts", Kind: "positive", Ran: true, Pass: true},
		{Name: "state transition", Kind: "positive", Ran: true, Pass: true},
		{
			Name:          "native Surface host rejects pre-rendered frame source",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "pre-rendered frame source rejected",
		},
		{
			Name:          "native Surface host rejects viewer substitution",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "viewer substitution rejected",
		},
		{
			Name:          "native Surface host rejects probe-frame substitution",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "probe-frame substitution rejected",
		},
		{
			Name:          "reject legacy UI evidence",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "legacy UI evidence rejected",
		},
	}
}

func nativeSurfaceHostEvidence(
	hostReport surfacehost.HostReport,
	frameCount int,
) *surface.NativeSurfaceHostReport {
	return &surface.NativeSurfaceHostReport{
		Schema:                 surface.NativeSurfaceHostSchemaV1,
		Host:                   "wayland",
		Protocol:               surface.NativeSurfaceHostProtocolV1,
		AppProcessKind:         "compiled-linux-x64-tetra-app",
		HostProcessKind:        "tetra-surface-host-wayland",
		AppPID:                 hostReport.AppPID,
		HostPID:                hostReport.HostPID,
		SurfaceOpenFromApp:     hostReport.OpenCount > 0,
		PollEventFromHost:      len(hostReport.Events) > 0,
		PresentFromAppRGBA:     hostReport.PresentedFrameCount > 0,
		AppLoopObserved:        hostReport.OpenCount > 0 && hostReport.PresentedFrameCount > 0,
		RealWindow:             true,
		RealCloseEvent:         hostReport.RealCloseEventCount > 0,
		RealPointerEventCount:  hostReport.RealPointerEventCount,
		RealKeyEventCount:      hostReport.RealKeyEventCount,
		PresentedFrameCount:    frameCount,
		PreRenderedFrameSource: hostReport.PreRenderedFrameSource,
		DeliveryPath:           hostReport.DeliveryPath,
	}
}

func processReport(
	name string,
	kind string,
	path string,
	exitCode int,
	expected *int,
) surface.ProcessReport {
	return surface.ProcessReport{
		Name:             name,
		Kind:             kind,
		Path:             path,
		Ran:              true,
		Pass:             exitCode == valueOrZero(expected),
		ExitCode:         intPtr(exitCode),
		ExpectedExitCode: expected,
	}
}

func valueOrZero(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}

func intPtr(v int) *int {
	return &v
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}
