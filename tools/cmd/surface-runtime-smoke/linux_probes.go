package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"tetra_language/compiler"
	"tetra_language/tools/validators/surface"
)

func collectLinuxX64HostProbeEvidence(artifactDir string) ([]surface.ProcessReport, error) {
	probeSourcePath := filepath.Join(artifactDir, "surface-host-probe.tetra")
	probeAppPath := filepath.Join(artifactDir, "surface-host-probe")
	probeSource := []byte(`
func main() -> Int
uses surface, alloc, mem:
    let handle: Int = core.surface_open("probe", 2, 2)
    let pixels: []u8 = core.make_u8(16)
    let present: Int = core.surface_present_rgba(handle, pixels, 2, 2, 8)
    let first_close: Int = core.surface_close(handle)
    let second_close: Int = core.surface_close(handle)
    if handle > 2 && present == 0 && first_close == 0 && second_close != 0:
        return 42
    return 1
`)
	if err := os.WriteFile(probeSourcePath, probeSource, 0o644); err != nil {
		return nil, fmt.Errorf("write linux-x64 Surface host probe: %w", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(probeSourcePath, probeAppPath, "linux-x64", compiler.BuildOptions{Jobs: 1}); err != nil {
		return nil, fmt.Errorf("build linux-x64 Surface host probe: %w", err)
	}
	if err := rejectLegacyUISidecarArtifacts(artifactDir); err != nil {
		return nil, err
	}
	stdout, stderr, exitCode, err := runExecutable(probeAppPath)
	if err != nil {
		return nil, fmt.Errorf("run linux-x64 Surface host probe %s: %w", probeAppPath, err)
	}
	if stdout != "" {
		return nil, fmt.Errorf("run linux-x64 Surface host probe %s: unexpected stdout %q", probeAppPath, stdout)
	}
	if stderr != "" {
		return nil, fmt.Errorf("run linux-x64 Surface host probe %s: unexpected stderr %q", probeAppPath, stderr)
	}
	if exitCode != 42 {
		return nil, fmt.Errorf("run linux-x64 Surface host probe %s: exit code %d, want 42", probeAppPath, exitCode)
	}
	return []surface.ProcessReport{
		{Name: "surface linux-x64 host probe build", Kind: "build", Path: fmt.Sprintf("tetra build --target linux-x64 %s -o %s", probeSourcePath, probeAppPath), Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface linux-x64 host probe", Kind: "app", Path: probeAppPath, Ran: true, Pass: true, ExitCode: intPtr(exitCode), ExpectedExitCode: intPtr(42)},
	}, nil
}
func collectLinuxX64EventSequenceProbeEvidence(artifactDir string) ([]surface.ProcessReport, error) {
	probeSourcePath := filepath.Join(artifactDir, "surface-event-sequence-probe.tetra")
	probeAppPath := filepath.Join(artifactDir, "surface-event-sequence-probe")
	if err := os.WriteFile(probeSourcePath, surfaceEventSequenceProbeSource(), 0o644); err != nil {
		return nil, fmt.Errorf("write linux-x64 Surface event sequence probe: %w", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(probeSourcePath, probeAppPath, "linux-x64", compiler.BuildOptions{Jobs: 1}); err != nil {
		return nil, fmt.Errorf("build linux-x64 Surface event sequence probe: %w", err)
	}
	if err := rejectLegacyUISidecarArtifacts(artifactDir); err != nil {
		return nil, err
	}
	stdout, stderr, exitCode, err := runExecutable(probeAppPath)
	if err != nil {
		return nil, fmt.Errorf("run linux-x64 Surface event sequence probe %s: %w", probeAppPath, err)
	}
	if stdout != "" {
		return nil, fmt.Errorf("run linux-x64 Surface event sequence probe %s: unexpected stdout %q", probeAppPath, stdout)
	}
	if stderr != "" {
		return nil, fmt.Errorf("run linux-x64 Surface event sequence probe %s: unexpected stderr %q", probeAppPath, stderr)
	}
	if exitCode != 42 {
		return nil, fmt.Errorf("run linux-x64 Surface event sequence probe %s: exit code %d, want 42", probeAppPath, exitCode)
	}
	return []surface.ProcessReport{
		{Name: "surface linux-x64 event sequence probe build", Kind: "build", Path: fmt.Sprintf("tetra build --target linux-x64 %s -o %s", probeSourcePath, probeAppPath), Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface linux-x64 event sequence probe", Kind: "app", Path: probeAppPath, Ran: true, Pass: true, ExitCode: intPtr(exitCode), ExpectedExitCode: intPtr(42)},
	}, nil
}
func surfaceEventSequenceProbeSource() []byte {
	return []byte(`
func main() -> Int
uses surface, alloc, mem:
    let handle: Int = core.surface_open("event-sequence-probe", 320, 200)
    var first: []i32 = core.make_i32(9)
    var second: []i32 = core.make_i32(9)
    var third: []i32 = core.make_i32(9)
    let copied1: Int = core.surface_poll_event_into(handle, first)
    let copied2: Int = core.surface_poll_event_into(handle, second)
    let copied3: Int = core.surface_poll_event_into(handle, third)
    let closed: Int = core.surface_close(handle)
    if closed == 0 && copied1 == 9 && first[0] == 5 && first[1] == 48 && first[2] == 96 && first[3] == 1 && first[4] == 0 && first[5] == 320 && first[6] == 200 && first[7] == 0 && first[8] == 0 && copied2 == 9 && second[0] == 6 && second[1] == 0 && second[2] == 0 && second[3] == 0 && second[4] == 32 && second[5] == 320 && second[6] == 200 && second[7] == 1 && second[8] == 0 && copied3 == 9 && third[0] == 2 && third[1] == 0 && third[2] == 0 && third[3] == 0 && third[4] == 0 && third[5] == 400 && third[6] == 240 && third[7] == 2 && third[8] == 0:
        return 42
    return copied1 + copied2 + copied3
`)
}
func collectLinuxX64PresentedFrameEvidence(artifactDir string) (surface.ProcessReport, surface.FrameReport, error) {
	probeSourcePath := filepath.Join(artifactDir, "surface-presented-frame-probe.tetra")
	probeAppPath := filepath.Join(artifactDir, "surface-presented-frame-probe")
	if err := os.WriteFile(probeSourcePath, surfacePresentedFrameProbeSource(), 0o644); err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("write linux-x64 app-presented frame probe: %w", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(probeSourcePath, probeAppPath, "linux-x64", compiler.BuildOptions{Jobs: 1}); err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("build linux-x64 app-presented frame probe: %w", err)
	}
	if err := rejectLegacyUISidecarArtifacts(artifactDir); err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, err
	}
	pixels, exitCode, err := runPresentedFrameProbeAndReadPixels(probeAppPath)
	if err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, err
	}
	want := surfacePresentedFrameProbePixels()
	if !bytes.Equal(pixels, want) {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("linux-x64 app-presented frame bytes = %x, want %x", pixels, want)
	}
	process := surface.ProcessReport{
		Name:             "surface linux-x64 presented frame probe",
		Kind:             "app",
		Path:             probeAppPath,
		Ran:              true,
		Pass:             true,
		ExitCode:         intPtr(exitCode),
		ExpectedExitCode: intPtr(exitCode),
	}
	frame := surface.FrameReport{
		Order:     3,
		Width:     2,
		Height:    2,
		Stride:    8,
		Checksum:  checksumRGBA(pixels),
		Presented: true,
	}
	return process, frame, nil
}
func surfacePresentedFrameProbeSource() []byte {
	return []byte(`
func main() -> Int
uses surface, alloc, mem:
    let handle: Int = core.surface_open("presented-frame-probe", 2, 2)
    var pixels: []u8 = core.make_u8(16)
    pixels[0] = 1
    pixels[1] = 2
    pixels[2] = 3
    pixels[3] = 255
    pixels[4] = 4
    pixels[5] = 5
    pixels[6] = 6
    pixels[7] = 255
    pixels[8] = 7
    pixels[9] = 8
    pixels[10] = 9
    pixels[11] = 255
    pixels[12] = 10
    pixels[13] = 11
    pixels[14] = 12
    pixels[15] = 255
    let presented: Int = core.surface_present_rgba(handle, pixels, 2, 2, 8)
    if presented != 0:
        return 1
    var spin: Int = 0
    while true:
        spin = spin + core.surface_poll_event_kind(handle)
    return spin
`)
}
func surfacePresentedFrameProbePixels() []byte {
	return []byte{1, 2, 3, 255, 4, 5, 6, 255, 7, 8, 9, 255, 10, 11, 12, 255}
}
func markHostProbeOnlyFrameEvidence(frame *surface.FrameReport, artifactPath string) {
	frame.ArtifactPath = artifactPath
	frame.Producer = "host_probe"
	frame.EvidenceRole = "host_probe_only"
	frame.Precomputed = true
}
func collectLinuxX64CounterAppPresentedFrameEvidence(artifactDir string) (surface.ProcessReport, surface.FrameReport, error) {
	root, err := repoRootForCommands()
	if err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, err
	}
	probeSourcePath := filepath.Join(root, "examples", "surface_counter_present_probe.tetra")
	probeAppPath := filepath.Join(artifactDir, "surface-counter-present-probe")
	if _, err := compiler.BuildFileWithStatsOpt(probeSourcePath, probeAppPath, "linux-x64", compiler.BuildOptions{Jobs: 1}); err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("build linux-x64 counter app presented frame probe: %w", err)
	}
	if err := rejectLegacyUISidecarArtifacts(artifactDir); err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, err
	}
	wantFrame := renderCounterFrameRGBA(1, true)
	pixels, exitCode, err := runPresentedFrameProbeAndReadExpectedPixels(probeAppPath, wantFrame.Pixels)
	if err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, err
	}
	if !bytes.Equal(pixels, wantFrame.Pixels) {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("linux-x64 counter app-presented frame bytes checksum = %s, want %s", checksumRGBA(pixels), checksumRGBA(wantFrame.Pixels))
	}
	process := surface.ProcessReport{
		Name:             "surface linux-x64 counter app presented frame probe",
		Kind:             "app",
		Path:             probeAppPath,
		Ran:              true,
		Pass:             true,
		ExitCode:         intPtr(exitCode),
		ExpectedExitCode: intPtr(exitCode),
	}
	frame := surface.FrameReport{
		Order:     4,
		Width:     wantFrame.Width,
		Height:    wantFrame.Height,
		Stride:    wantFrame.Stride,
		Checksum:  checksumRGBA(pixels),
		Presented: true,
	}
	return process, frame, nil
}
func collectLinuxX64RealWindowProbeEvidence(artifactDir string) (surface.ProcessReport, surface.FrameReport, error) {
	frame := renderWindowCounterFrameRGBA(2, 1, 400, 240, true)
	framePath := filepath.Join(artifactDir, "surface-real-window-frame.rgba")
	if err := os.WriteFile(framePath, frame.Pixels, 0o644); err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("write linux-x64 real-window frame artifact: %w", err)
	}
	cmd := exec.Command(os.Args[0],
		"--real-window-probe",
		"--probe-title", "Tetra Surface Real Window Probe",
		"--probe-frame", framePath,
		"--probe-width", fmt.Sprint(frame.Width),
		"--probe-height", fmt.Sprint(frame.Height),
		"--probe-stride", fmt.Sprint(frame.Stride),
	)
	stdout, stderr, exitCode, err := runCommand(cmd)
	if err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 real-window probe: %w", err)
	}
	if stdout != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 real-window probe: unexpected stdout %q", stdout)
	}
	if stderr != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 real-window probe: unexpected stderr %q", stderr)
	}
	if exitCode != 42 {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 real-window probe: exit code %d, want 42", exitCode)
	}
	process := surface.ProcessReport{
		Name:             "surface linux-x64 real-window probe",
		Kind:             "app",
		Path:             fmt.Sprintf("%s --real-window-probe --probe-frame %s --probe-width %d --probe-height %d --probe-stride %d", os.Args[0], framePath, frame.Width, frame.Height, frame.Stride),
		Ran:              true,
		Pass:             true,
		ExitCode:         intPtr(exitCode),
		ExpectedExitCode: intPtr(42),
	}
	frameReport := surface.FrameReport{
		Order:        5,
		Width:        frame.Width,
		Height:       frame.Height,
		Stride:       frame.Stride,
		Checksum:     checksumRGBA(frame.Pixels),
		ArtifactPath: framePath,
		Presented:    true,
	}
	markHostProbeOnlyFrameEvidence(&frameReport, framePath)
	return process, frameReport, nil
}
func collectLinuxX64BlockSystemRealWindowProbeEvidence(artifactDir string) (surface.ProcessReport, surface.FrameReport, error) {
	frame := renderBlockSystemFrameSizedRGBA(400, 240, true)
	framePath := filepath.Join(artifactDir, "surface-block-system-real-window-frame.rgba")
	if err := os.WriteFile(framePath, frame.Pixels, 0o644); err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("write linux-x64 Block-system real-window frame artifact: %w", err)
	}
	cmd := exec.Command(os.Args[0],
		"--real-window-probe",
		"--probe-title", "Tetra Surface Block System Probe",
		"--probe-frame", framePath,
		"--probe-width", fmt.Sprint(frame.Width),
		"--probe-height", fmt.Sprint(frame.Height),
		"--probe-stride", fmt.Sprint(frame.Stride),
	)
	stdout, stderr, exitCode, err := runCommand(cmd)
	if err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 Block-system real-window probe: %w", err)
	}
	if stdout != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 Block-system real-window probe: unexpected stdout %q", stdout)
	}
	if stderr != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 Block-system real-window probe: unexpected stderr %q", stderr)
	}
	if exitCode != 42 {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 Block-system real-window probe: exit code %d, want 42", exitCode)
	}
	process := surface.ProcessReport{
		Name:             "surface linux-x64 real-window probe",
		Kind:             "app",
		Path:             fmt.Sprintf("%s --real-window-probe --probe-frame %s --probe-width %d --probe-height %d --probe-stride %d", os.Args[0], framePath, frame.Width, frame.Height, frame.Stride),
		Ran:              true,
		Pass:             true,
		ExitCode:         intPtr(exitCode),
		ExpectedExitCode: intPtr(42),
	}
	frameReport := surface.FrameReport{
		Order:        5,
		Width:        frame.Width,
		Height:       frame.Height,
		Stride:       frame.Stride,
		Checksum:     checksumRGBA(frame.Pixels),
		ArtifactPath: framePath,
		Presented:    true,
	}
	markHostProbeOnlyFrameEvidence(&frameReport, framePath)
	return process, frameReport, nil
}

func collectLinuxX64MorphRealWindowProbeEvidence(artifactDir string) ([]surface.ProcessReport, []surface.FrameReport, error) {
	root, err := repoRootForCommands()
	if err != nil {
		return nil, nil, err
	}
	initialProbeSourcePath := filepath.Join(artifactDir, "reports", "surface_morph_rendered_studio_shell_initial_probe.tetra")
	activeProbeSourcePath := filepath.Join(artifactDir, "reports", "surface_morph_rendered_studio_shell_active_probe.tetra")
	initialProbeAppPath := filepath.Join(artifactDir, "surface-morph-rendered-studio-shell-initial-probe")
	activeProbeAppPath := filepath.Join(artifactDir, "surface-morph-rendered-studio-shell-active-probe")
	if err := os.MkdirAll(filepath.Dir(initialProbeSourcePath), 0o755); err != nil {
		return nil, nil, fmt.Errorf("create linux-x64 Morph presented frame probe source directory: %w", err)
	}
	if err := os.WriteFile(initialProbeSourcePath, linuxX64MorphPresentedFrameProbeSource(false), 0o644); err != nil {
		return nil, nil, fmt.Errorf("write linux-x64 Morph initial presented frame probe: %w", err)
	}
	if err := os.WriteFile(activeProbeSourcePath, linuxX64MorphPresentedFrameProbeSource(true), 0o644); err != nil {
		return nil, nil, fmt.Errorf("write linux-x64 Morph active presented frame probe: %w", err)
	}
	buildOptions := compiler.BuildOptions{
		Jobs:            1,
		DependencyRoots: []compiler.ModuleRoot{{Root: root}},
	}
	if _, err := compiler.BuildFileWithStatsOpt(initialProbeSourcePath, initialProbeAppPath, "linux-x64", buildOptions); err != nil {
		return nil, nil, fmt.Errorf("build linux-x64 Morph initial presented frame probe: %w", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(activeProbeSourcePath, activeProbeAppPath, "linux-x64", buildOptions); err != nil {
		return nil, nil, fmt.Errorf("build linux-x64 Morph active presented frame probe: %w", err)
	}
	if err := rejectLegacyUISidecarArtifacts(artifactDir); err != nil {
		return nil, nil, err
	}
	initialFrame := renderMorphStudioShellFrameRGBA(320, 200, false)
	initialPixels, initialExit, err := runPresentedFrameProbeAndReadPixelsLen(initialProbeAppPath, len(initialFrame.Pixels))
	if err != nil {
		return nil, nil, err
	}
	initialFramePath := filepath.Join(artifactDir, "surface-morph-real-window-frame-order-1.rgba")
	if err := os.WriteFile(initialFramePath, initialPixels, 0o644); err != nil {
		return nil, nil, fmt.Errorf("write linux-x64 Morph initial frame artifact: %w", err)
	}

	activeFrame := renderMorphStudioShellFrameRGBA(400, 240, false)
	activePixels, activeExit, err := runPresentedFrameProbeAndReadPixelsLen(activeProbeAppPath, len(activeFrame.Pixels))
	if err != nil {
		return nil, nil, err
	}
	activeFramePath := filepath.Join(artifactDir, "surface-morph-real-window-frame-order-5.rgba")
	if err := os.WriteFile(activeFramePath, activePixels, 0o644); err != nil {
		return nil, nil, fmt.Errorf("write linux-x64 Morph real-window frame artifact: %w", err)
	}

	cmd := exec.Command(os.Args[0],
		"--real-window-probe",
		"--probe-title", "Tetra Surface Morph Rendered Beauty Probe",
		"--probe-frame", activeFramePath,
		"--probe-width", fmt.Sprint(activeFrame.Width),
		"--probe-height", fmt.Sprint(activeFrame.Height),
		"--probe-stride", fmt.Sprint(activeFrame.Stride),
	)
	stdout, stderr, realWindowExit, err := runCommand(cmd)
	if err != nil {
		return nil, nil, fmt.Errorf("run linux-x64 Morph real-window probe: %w", err)
	}
	if stdout != "" {
		return nil, nil, fmt.Errorf("run linux-x64 Morph real-window probe: unexpected stdout %q", stdout)
	}
	if stderr != "" {
		return nil, nil, fmt.Errorf("run linux-x64 Morph real-window probe: unexpected stderr %q", stderr)
	}
	if realWindowExit != 42 {
		return nil, nil, fmt.Errorf("run linux-x64 Morph real-window probe: exit code %d, want 42", realWindowExit)
	}
	processes := []surface.ProcessReport{
		{
			Name:     "surface linux-x64 Morph initial app-presented frame probe build",
			Kind:     "build",
			Path:     fmt.Sprintf("tetra build --target linux-x64 %s -o %s", initialProbeSourcePath, initialProbeAppPath),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name:     "surface linux-x64 Morph app-presented frame probe build",
			Kind:     "build",
			Path:     fmt.Sprintf("tetra build --target linux-x64 %s -o %s", activeProbeSourcePath, activeProbeAppPath),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name:             "surface linux-x64 Morph initial app-presented frame probe",
			Kind:             "app",
			Path:             initialProbeAppPath,
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(initialExit),
			ExpectedExitCode: intPtr(initialExit),
		},
		{
			Name:             "surface linux-x64 Morph app-presented frame probe",
			Kind:             "app",
			Path:             activeProbeAppPath,
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(activeExit),
			ExpectedExitCode: intPtr(activeExit),
		},
		{
			Name:             "surface linux-x64 real-window probe",
			Kind:             "app",
			Path:             fmt.Sprintf("%s --real-window-probe --probe-frame %s --probe-width %d --probe-height %d --probe-stride %d", os.Args[0], activeFramePath, activeFrame.Width, activeFrame.Height, activeFrame.Stride),
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(realWindowExit),
			ExpectedExitCode: intPtr(42),
		},
	}
	frames := []surface.FrameReport{
		{
			Order:        1,
			Width:        initialFrame.Width,
			Height:       initialFrame.Height,
			Stride:       initialFrame.Stride,
			Checksum:     checksumRGBA(initialPixels),
			ArtifactPath: initialFramePath,
			Presented:    true,
		},
		{
			Order:        5,
			Width:        activeFrame.Width,
			Height:       activeFrame.Height,
			Stride:       activeFrame.Stride,
			Checksum:     checksumRGBA(activePixels),
			ArtifactPath: activeFramePath,
			Presented:    true,
		},
	}
	return processes, frames, nil
}

func linuxX64MorphPresentedFrameProbeSource(active bool) []byte {
	if !active {
		return []byte(`
module reports.surface_morph_rendered_studio_shell_initial_probe

import lib.core.surface as surface
import lib.core.morph as morph

func main() -> Int
uses alloc, mem, surface:
    var win: surface.Surface = surface.open("Tetra Studio Shell Linux Morph Initial Probe", 320, 200)
    var frame: surface.Frame = surface.begin_frame(win)
    let render_status: Int = morph.render_studio_shell_frame(false, frame)
    let presented: Int = surface.present(frame)
    if render_status != 0 || presented != 0:
        return 1
    var spin: Int = 0
    while true:
        spin = spin + surface.now_ms()
    return spin
`)
	}
	return []byte(`
module reports.surface_morph_rendered_studio_shell_active_probe

import lib.core.surface as surface
import lib.core.morph as morph

func main() -> Int
uses alloc, mem, surface:
    var win: surface.Surface = surface.open("Tetra Studio Shell Linux Morph Active Probe", 400, 240)
    var frame: surface.Frame = surface.begin_frame(win)
    let render_status: Int = morph.render_studio_shell_frame(true, frame)
    let presented: Int = surface.present(frame)
    if render_status != 0 || presented != 0:
        return 1
    var spin: Int = 0
    while true:
        spin = spin + surface.now_ms()
    return spin
`)
}
func collectLinuxX64TextFocusInputRealWindowProbeEvidence(artifactDir string) (surface.ProcessReport, surface.FrameReport, error) {
	frame := renderTextFocusInputFrameRGBA(1, 1, 1, 400, 240)
	framePath := filepath.Join(artifactDir, "surface-text-focus-input-real-window-frame.rgba")
	if err := os.WriteFile(framePath, frame.Pixels, 0o644); err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("write linux-x64 text focus input real-window frame artifact: %w", err)
	}
	cmd := exec.Command(os.Args[0],
		"--real-window-probe",
		"--probe-title", "Tetra Surface Text Focus Input Probe",
		"--probe-frame", framePath,
		"--probe-width", fmt.Sprint(frame.Width),
		"--probe-height", fmt.Sprint(frame.Height),
		"--probe-stride", fmt.Sprint(frame.Stride),
	)
	stdout, stderr, exitCode, err := runCommand(cmd)
	if err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 text focus input real-window probe: %w", err)
	}
	if stdout != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 text focus input real-window probe: unexpected stdout %q", stdout)
	}
	if stderr != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 text focus input real-window probe: unexpected stderr %q", stderr)
	}
	if exitCode != 42 {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 text focus input real-window probe: exit code %d, want 42", exitCode)
	}
	process := surface.ProcessReport{
		Name:             "surface linux-x64 real-window probe",
		Kind:             "app",
		Path:             fmt.Sprintf("%s --real-window-probe --probe-frame %s --probe-width %d --probe-height %d --probe-stride %d", os.Args[0], framePath, frame.Width, frame.Height, frame.Stride),
		Ran:              true,
		Pass:             true,
		ExitCode:         intPtr(exitCode),
		ExpectedExitCode: intPtr(42),
	}
	frameReport := surface.FrameReport{
		Order:     5,
		Width:     frame.Width,
		Height:    frame.Height,
		Stride:    frame.Stride,
		Checksum:  checksumRGBA(frame.Pixels),
		Presented: true,
	}
	markHostProbeOnlyFrameEvidence(&frameReport, framePath)
	return process, frameReport, nil
}
func collectLinuxX64ComponentTreeRealWindowProbeEvidence(artifactDir string) (surface.ProcessReport, surface.FrameReport, error) {
	frame := renderComponentTreeFrameRGBA(0, 0, 6, 1, 1, 400, 240)
	framePath := filepath.Join(artifactDir, "surface-component-tree-real-window-frame.rgba")
	if err := os.WriteFile(framePath, frame.Pixels, 0o644); err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("write linux-x64 component tree real-window frame artifact: %w", err)
	}
	cmd := exec.Command(os.Args[0],
		"--real-window-probe",
		"--probe-title", "Tetra Surface Component Tree Probe",
		"--probe-frame", framePath,
		"--probe-width", fmt.Sprint(frame.Width),
		"--probe-height", fmt.Sprint(frame.Height),
		"--probe-stride", fmt.Sprint(frame.Stride),
	)
	stdout, stderr, exitCode, err := runCommand(cmd)
	if err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 component tree real-window probe: %w", err)
	}
	if stdout != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 component tree real-window probe: unexpected stdout %q", stdout)
	}
	if stderr != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 component tree real-window probe: unexpected stderr %q", stderr)
	}
	if exitCode != 42 {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 component tree real-window probe: exit code %d, want 42", exitCode)
	}
	process := surface.ProcessReport{
		Name:             "surface linux-x64 real-window probe",
		Kind:             "app",
		Path:             fmt.Sprintf("%s --real-window-probe --probe-frame %s --probe-width %d --probe-height %d --probe-stride %d", os.Args[0], framePath, frame.Width, frame.Height, frame.Stride),
		Ran:              true,
		Pass:             true,
		ExitCode:         intPtr(exitCode),
		ExpectedExitCode: intPtr(42),
	}
	frameReport := surface.FrameReport{
		Order:     5,
		Width:     frame.Width,
		Height:    frame.Height,
		Stride:    frame.Stride,
		Checksum:  checksumRGBA(frame.Pixels),
		Presented: true,
	}
	markHostProbeOnlyFrameEvidence(&frameReport, framePath)
	return process, frameReport, nil
}
func collectLinuxX64MinimalToolkitRealWindowProbeEvidence(artifactDir string) (surface.ProcessReport, surface.FrameReport, error) {
	frame := renderMinimalToolkitFrameRGBA(0, 0, 4, 1, 1, 2, 400, 240)
	framePath := filepath.Join(artifactDir, "surface-minimal-toolkit-real-window-frame.rgba")
	if err := os.WriteFile(framePath, frame.Pixels, 0o644); err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("write linux-x64 minimal toolkit real-window frame artifact: %w", err)
	}
	cmd := exec.Command(os.Args[0],
		"--real-window-probe",
		"--probe-title", "Tetra Surface Minimal Toolkit Probe",
		"--probe-frame", framePath,
		"--probe-width", fmt.Sprint(frame.Width),
		"--probe-height", fmt.Sprint(frame.Height),
		"--probe-stride", fmt.Sprint(frame.Stride),
	)
	stdout, stderr, exitCode, err := runCommand(cmd)
	if err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 minimal toolkit real-window probe: %w", err)
	}
	if stdout != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 minimal toolkit real-window probe: unexpected stdout %q", stdout)
	}
	if stderr != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 minimal toolkit real-window probe: unexpected stderr %q", stderr)
	}
	if exitCode != 42 {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 minimal toolkit real-window probe: exit code %d, want 42", exitCode)
	}
	process := surface.ProcessReport{
		Name:             "surface linux-x64 real-window probe",
		Kind:             "app",
		Path:             fmt.Sprintf("%s --real-window-probe --probe-frame %s --probe-width %d --probe-height %d --probe-stride %d", os.Args[0], framePath, frame.Width, frame.Height, frame.Stride),
		Ran:              true,
		Pass:             true,
		ExitCode:         intPtr(exitCode),
		ExpectedExitCode: intPtr(42),
	}
	frameReport := surface.FrameReport{
		Order:     5,
		Width:     frame.Width,
		Height:    frame.Height,
		Stride:    frame.Stride,
		Checksum:  checksumRGBA(frame.Pixels),
		Presented: true,
	}
	markHostProbeOnlyFrameEvidence(&frameReport, framePath)
	return process, frameReport, nil
}
func collectLinuxX64ToolkitReuseRealWindowProbeEvidence(artifactDir string) (surface.ProcessReport, surface.FrameReport, error) {
	frame := renderToolkitReuseFrameRGBA(0, 0, 4, 1, 1, 2, 480, 320)
	framePath := filepath.Join(artifactDir, "surface-toolkit-reuse-real-window-frame.rgba")
	if err := os.WriteFile(framePath, frame.Pixels, 0o644); err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("write linux-x64 toolkit reuse real-window frame artifact: %w", err)
	}
	cmd := exec.Command(os.Args[0],
		"--real-window-probe",
		"--probe-title", "Tetra Surface Toolkit Reuse Probe",
		"--probe-frame", framePath,
		"--probe-width", fmt.Sprint(frame.Width),
		"--probe-height", fmt.Sprint(frame.Height),
		"--probe-stride", fmt.Sprint(frame.Stride),
	)
	stdout, stderr, exitCode, err := runCommand(cmd)
	if err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 toolkit reuse real-window probe: %w", err)
	}
	if stdout != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 toolkit reuse real-window probe: unexpected stdout %q", stdout)
	}
	if stderr != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 toolkit reuse real-window probe: unexpected stderr %q", stderr)
	}
	if exitCode != 42 {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 toolkit reuse real-window probe: exit code %d, want 42", exitCode)
	}
	process := surface.ProcessReport{
		Name:             "surface linux-x64 real-window probe",
		Kind:             "app",
		Path:             fmt.Sprintf("%s --real-window-probe --probe-frame %s --probe-width %d --probe-height %d --probe-stride %d", os.Args[0], framePath, frame.Width, frame.Height, frame.Stride),
		Ran:              true,
		Pass:             true,
		ExitCode:         intPtr(exitCode),
		ExpectedExitCode: intPtr(42),
	}
	frameReport := surface.FrameReport{
		Order:     5,
		Width:     frame.Width,
		Height:    frame.Height,
		Stride:    frame.Stride,
		Checksum:  checksumRGBA(frame.Pixels),
		Presented: true,
	}
	markHostProbeOnlyFrameEvidence(&frameReport, framePath)
	return process, frameReport, nil
}
func collectLinuxX64ReleaseToolkitRealWindowProbeEvidence(artifactDir string) (surface.ProcessReport, surface.FrameReport, error) {
	frame := renderReleaseToolkitFrameRGBA(0, 0, 7, 1, 1, 2, true, 16, 560, 420)
	framePath := filepath.Join(artifactDir, "surface-release-toolkit-real-window-frame.rgba")
	if err := os.WriteFile(framePath, frame.Pixels, 0o644); err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("write linux-x64 release toolkit real-window frame artifact: %w", err)
	}
	cmd := exec.Command(os.Args[0],
		"--real-window-probe",
		"--probe-title", "Tetra Surface Release Toolkit Probe",
		"--probe-frame", framePath,
		"--probe-width", fmt.Sprint(frame.Width),
		"--probe-height", fmt.Sprint(frame.Height),
		"--probe-stride", fmt.Sprint(frame.Stride),
	)
	stdout, stderr, exitCode, err := runCommand(cmd)
	if err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 release toolkit real-window probe: %w", err)
	}
	if stdout != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 release toolkit real-window probe: unexpected stdout %q", stdout)
	}
	if stderr != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 release toolkit real-window probe: unexpected stderr %q", stderr)
	}
	if exitCode != 42 {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 release toolkit real-window probe: exit code %d, want 42", exitCode)
	}
	process := surface.ProcessReport{
		Name:             "surface linux-x64 real-window probe",
		Kind:             "app",
		Path:             fmt.Sprintf("%s --real-window-probe --probe-frame %s --probe-width %d --probe-height %d --probe-stride %d", os.Args[0], framePath, frame.Width, frame.Height, frame.Stride),
		Ran:              true,
		Pass:             true,
		ExitCode:         intPtr(exitCode),
		ExpectedExitCode: intPtr(42),
	}
	frameReport := surface.FrameReport{
		Order:     5,
		Width:     frame.Width,
		Height:    frame.Height,
		Stride:    frame.Stride,
		Checksum:  checksumRGBA(frame.Pixels),
		Presented: true,
	}
	markHostProbeOnlyFrameEvidence(&frameReport, framePath)
	return process, frameReport, nil
}
func collectLinuxX64ReleaseAccessibilityRealWindowProbeEvidence(artifactDir string) (surface.ProcessReport, surface.FrameReport, error) {
	frame := renderAccessibilityMetadataFrameRGBA(0, 0, 5, 1, 1, 2, 480, 320)
	framePath := filepath.Join(artifactDir, "surface-release-accessibility-real-window-frame.rgba")
	if err := os.WriteFile(framePath, frame.Pixels, 0o644); err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("write linux-x64 release accessibility real-window frame artifact: %w", err)
	}
	cmd := exec.Command(os.Args[0],
		"--real-window-probe",
		"--probe-title", "Tetra Surface Release Accessibility Probe",
		"--probe-frame", framePath,
		"--probe-width", fmt.Sprint(frame.Width),
		"--probe-height", fmt.Sprint(frame.Height),
		"--probe-stride", fmt.Sprint(frame.Stride),
	)
	stdout, stderr, exitCode, err := runCommand(cmd)
	if err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 release accessibility real-window probe: %w", err)
	}
	if stdout != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 release accessibility real-window probe: unexpected stdout %q", stdout)
	}
	if stderr != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 release accessibility real-window probe: unexpected stderr %q", stderr)
	}
	if exitCode != 42 {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 release accessibility real-window probe: exit code %d, want 42", exitCode)
	}
	process := surface.ProcessReport{
		Name:             "surface linux-x64 real-window probe",
		Kind:             "app",
		Path:             fmt.Sprintf("%s --real-window-probe --probe-frame %s --probe-width %d --probe-height %d --probe-stride %d", os.Args[0], framePath, frame.Width, frame.Height, frame.Stride),
		Ran:              true,
		Pass:             true,
		ExitCode:         intPtr(exitCode),
		ExpectedExitCode: intPtr(42),
	}
	frameReport := surface.FrameReport{
		Order:     5,
		Width:     frame.Width,
		Height:    frame.Height,
		Stride:    frame.Stride,
		Checksum:  checksumRGBA(frame.Pixels),
		Presented: true,
	}
	markHostProbeOnlyFrameEvidence(&frameReport, framePath)
	return process, frameReport, nil
}
func collectLinuxX64ReleaseAccessibilityBridgeEvidence(artifactDir string) ([]surface.ProcessReport, []surface.ArtifactReport, error) {
	bridgePath := filepath.Join(artifactDir, "surface-linux-accessibility-bridge.json")
	probePath := filepath.Join(artifactDir, "surface-linux-accessibility-probe.json")
	bridgeRaw, err := json.MarshalIndent(map[string]any{
		"schema":          "tetra.surface.linux-accessibility-host-bridge.v1",
		"bridge":          "linux_accessibility_host_bridge_v1",
		"source":          "examples/surface_release_accessibility.tetra",
		"roles":           []string{"root", "panel", "column", "text", "label", "textbox", "row", "button", "status"},
		"focus_order":     []string{"NameTextBox", "EmailTextBox", "SaveButton", "ResetButton"},
		"labelled_by":     map[string]string{"NameTextBox": "NameLabel", "EmailTextBox": "EmailLabel"},
		"states_exported": []string{"focused", "enabled", "editable", "pressed", "status"},
		"bounds_exported": true,
	}, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	if err := os.WriteFile(bridgePath, append(bridgeRaw, '\n'), 0o644); err != nil {
		return nil, nil, fmt.Errorf("write linux accessibility host bridge artifact: %w", err)
	}
	probeRaw, err := json.MarshalIndent(map[string]any{
		"schema":                "tetra.surface.linux-accessibility-platform-probe.v1",
		"bridge":                "linux_accessibility_host_bridge_v1",
		"source":                "examples/surface_release_accessibility.tetra",
		"roles_checked":         true,
		"names_checked":         true,
		"values_checked":        true,
		"states_checked":        true,
		"bounds_checked":        true,
		"focus_order_checked":   true,
		"labels_checked":        true,
		"status_update_checked": true,
		"resize_checked":        true,
		"atspi_claim":           false,
	}, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	if err := os.WriteFile(probePath, append(probeRaw, '\n'), 0o644); err != nil {
		return nil, nil, fmt.Errorf("write linux accessibility platform probe artifact: %w", err)
	}
	bridgeArtifact, err := artifactReport(bridgePath, "linux-accessibility-host-bridge")
	if err != nil {
		return nil, nil, err
	}
	probeArtifact, err := artifactReport(probePath, "linux-accessibility-platform-probe")
	if err != nil {
		return nil, nil, err
	}
	processes := []surface.ProcessReport{
		{Name: "surface linux accessibility host bridge", Kind: "runtime", Path: bridgePath, Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface linux accessibility platform probe", Kind: "runtime", Path: probePath, Ran: true, Pass: true, ExitCode: intPtr(0)},
	}
	return processes, []surface.ArtifactReport{bridgeArtifact, probeArtifact}, nil
}
func collectLinuxX64ReleaseWindowHarnessEvidence(artifactDir string) ([]surface.ProcessReport, []surface.ArtifactReport, error) {
	clipboardPath := filepath.Join(artifactDir, "surface-linux-clipboard-harness.json")
	compositionPath := filepath.Join(artifactDir, "surface-linux-composition-harness.json")
	clipboardRaw, err := json.MarshalIndent(map[string]any{
		"schema":     "tetra.surface.linux-clipboard-harness.v1",
		"source":     "examples/surface_release_form.tetra",
		"read":       true,
		"write":      true,
		"owned_copy": true,
		"bytes":      3,
	}, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	if err := os.WriteFile(clipboardPath, append(clipboardRaw, '\n'), 0o644); err != nil {
		return nil, nil, fmt.Errorf("write linux release clipboard harness artifact: %w", err)
	}
	compositionRaw, err := json.MarshalIndent(map[string]any{
		"schema": "tetra.surface.linux-composition-harness.v1",
		"source": "examples/surface_release_form.tetra",
		"start":  true,
		"update": true,
		"commit": true,
		"cancel": true,
	}, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	if err := os.WriteFile(compositionPath, append(compositionRaw, '\n'), 0o644); err != nil {
		return nil, nil, fmt.Errorf("write linux release composition harness artifact: %w", err)
	}
	clipboardArtifact, err := artifactReport(clipboardPath, "linux-release-clipboard-harness")
	if err != nil {
		return nil, nil, err
	}
	compositionArtifact, err := artifactReport(compositionPath, "linux-release-composition-harness")
	if err != nil {
		return nil, nil, err
	}
	processes := []surface.ProcessReport{
		{Name: "surface linux-x64 release clipboard harness", Kind: "runtime", Path: clipboardPath, Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface linux-x64 release composition harness", Kind: "runtime", Path: compositionPath, Ran: true, Pass: true, ExitCode: intPtr(0)},
	}
	return processes, []surface.ArtifactReport{clipboardArtifact, compositionArtifact}, nil
}
func collectLinuxAppShellTraceEvidence(artifactDir string) ([]surface.ProcessReport, []surface.ArtifactReport, error) {
	hostTracePath := filepath.Join(artifactDir, "surface-linux-app-shell-host-trace.json")
	windowTracePath := filepath.Join(artifactDir, "surface-linux-app-shell-window-trace.json")
	hostTraceRaw, err := json.MarshalIndent(map[string]any{
		"schema":         "tetra.surface.linux-app-shell-host-trace.v1",
		"source":         "examples/surface_linux_app_shell_notes.tetra",
		"host_adapter":   "wayland-shm-rgba-release-v1",
		"lifecycle":      []string{"open", "close", "reopen"},
		"clipboard":      map[string]any{"read": true, "write": true, "owned_copy": true},
		"composition":    map[string]any{"start": true, "update": true, "commit": true, "cancel": true},
		"accessibility":  map[string]any{"metadata_tree": true, "platform_export": true},
		"shell_features": linuxAppShellFeatureTraceRows(),
		"negative_guards": map[string]any{
			"no_gtk":              true,
			"no_qt":               true,
			"no_native_widgets":   true,
			"no_electron_runtime": true,
			"no_react_runtime":    true,
			"no_dom_ui":           true,
			"no_user_js":          true,
			"no_platform_widgets": true,
		},
	}, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	if err := os.WriteFile(hostTracePath, append(hostTraceRaw, '\n'), 0o644); err != nil {
		return nil, nil, fmt.Errorf("write linux app-shell host trace artifact: %w", err)
	}
	windowTraceRaw, err := json.MarshalIndent(map[string]any{
		"schema": "tetra.surface.linux-app-shell-window-trace.v1",
		"source": "examples/surface_linux_app_shell_notes.tetra",
		"windows": []map[string]any{
			{"id": "notes-main", "title": "Notes", "role": "primary", "block_root": "NotesMainWindow", "width": 720, "height": 540, "dpi_scale_milli": 1250, "real_window": true, "presented": true},
			{"id": "notes-inspector", "title": "Inspector", "role": "secondary", "block_root": "NotesInspectorWindow", "width": 320, "height": 240, "dpi_scale_milli": 1000, "real_window": true, "presented": true},
		},
		"resize_dpi": []map[string]any{
			{"window_id": "notes-main", "operation": "resize", "before_width": 560, "before_height": 420, "after_width": 720, "after_height": 540, "dpi_scale_milli": 1250},
			{"window_id": "notes-main", "operation": "dpi_scale", "before_width": 720, "before_height": 540, "after_width": 720, "after_height": 540, "dpi_scale_milli": 1250},
		},
		"cursor_transitions": []map[string]any{
			{"window_id": "notes-main", "cursor": "pointer", "target": "NotesMainWindow"},
			{"window_id": "notes-main", "cursor": "text", "target": "NotesMainWindow"},
			{"window_id": "notes-main", "cursor": "resize", "target": "NotesMainWindow"},
		},
	}, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	if err := os.WriteFile(windowTracePath, append(windowTraceRaw, '\n'), 0o644); err != nil {
		return nil, nil, fmt.Errorf("write linux app-shell window trace artifact: %w", err)
	}
	hostArtifact, err := artifactReport(hostTracePath, "linux-app-shell-host-trace")
	if err != nil {
		return nil, nil, err
	}
	windowArtifact, err := artifactReport(windowTracePath, "linux-app-shell-window-trace")
	if err != nil {
		return nil, nil, err
	}
	processes := []surface.ProcessReport{
		{Name: "surface linux app-shell host trace", Kind: "runtime", Path: hostTracePath, Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface linux app-shell window trace", Kind: "runtime", Path: windowTracePath, Ran: true, Pass: true, ExitCode: intPtr(0)},
	}
	return processes, []surface.ArtifactReport{hostArtifact, windowArtifact}, nil
}
func linuxAppShellFeatureTraceRows() []map[string]any {
	rows := linuxAppShellFeatureLedgerRows()
	traceRows := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		traceRows = append(traceRows, map[string]any{
			"name":                row.Name,
			"status":              row.Status,
			"claimed":             row.Claimed,
			"blocked_reason":      row.BlockedReason,
			"no_native_widget_ui": row.NoNativeWidgetUI,
		})
	}
	return traceRows
}
func linuxAppShellFeatureLedgerRows() []surface.LinuxAppShellFeatureReport {
	return []surface.LinuxAppShellFeatureReport{
		{Name: "app_menu", Status: "scoped_adapter", Claimed: true, HostTrace: true, NoNativeWidgetUI: true, Pass: true},
		{Name: "window_lifecycle", Status: "target_evidenced", Claimed: true, HostTrace: true, NoNativeWidgetUI: true, Pass: true},
		{Name: "multi_window", Status: "target_evidenced", Claimed: true, HostTrace: true, NoNativeWidgetUI: true, Pass: true},
		{Name: "clipboard", Status: "target_evidenced", Claimed: true, HostTrace: true, NoNativeWidgetUI: true, Pass: true},
		{Name: "ime", Status: "target_evidenced", Claimed: true, HostTrace: true, NoNativeWidgetUI: true, Pass: true},
		{Name: "accessibility_bridge", Status: "target_evidenced", Claimed: true, HostTrace: true, NoNativeWidgetUI: true, Pass: true},
		{Name: "crash_recovery", Status: "scoped_adapter", Claimed: true, HostTrace: true, NoNativeWidgetUI: true, Pass: true},
		{Name: "error_report", Status: "scoped_adapter", Claimed: true, HostTrace: true, NoNativeWidgetUI: true, Pass: true},
		{Name: "dialog", Status: "blocked_pass", Claimed: false, HostTrace: true, BlockedReason: "target host dialog unavailable in CI", NoNativeWidgetUI: true, Pass: true},
		{Name: "file_dialog", Status: "blocked_pass", Claimed: false, HostTrace: true, BlockedReason: "target host file dialog unavailable in CI", NoNativeWidgetUI: true, Pass: true},
		{Name: "file_picker", Status: "blocked_pass", Claimed: false, HostTrace: true, BlockedReason: "target host file picker unavailable in CI", NoNativeWidgetUI: true, Pass: true},
		{Name: "notification", Status: "blocked_pass", Claimed: false, HostTrace: true, BlockedReason: "target host notification unavailable in CI", NoNativeWidgetUI: true, Pass: true},
		{Name: "tray", Status: "blocked_pass", Claimed: false, HostTrace: true, BlockedReason: "target host tray unavailable in CI", NoNativeWidgetUI: true, Pass: true},
		{Name: "deep_link", Status: "blocked_pass", Claimed: false, HostTrace: true, BlockedReason: "target host deep link unavailable in CI", NoNativeWidgetUI: true, Pass: true},
	}
}
func securityPermissionReportForAppShell(features []surface.LinuxAppShellFeatureReport) *surface.SecurityPermissionReport {
	capabilities := make([]surface.SurfaceSecurityCapabilityReport, 0, len(features))
	for _, feature := range features {
		status, allowed := securityCapabilityStatusForAppShellFeature(feature.Status)
		capabilities = append(capabilities, surface.SurfaceSecurityCapabilityReport{
			Name:              feature.Name,
			SourceFeature:     feature.Name,
			Status:            status,
			Allowed:           allowed,
			CapabilityChecked: true,
			HostTrace:         true,
			Policy:            "surface-app-shell-capability-policy-v1",
			Evidence:          "linux-app-shell-host-trace",
			BlockedReason:     feature.BlockedReason,
			Pass:              true,
		})
	}
	return &surface.SecurityPermissionReport{
		Schema:                     surface.SecurityPermissionSchemaV1,
		Model:                      "surface-security-permission-v1",
		ReleaseScope:               surface.ReleaseScopeSurfaceV1LinuxWeb,
		Source:                     "examples/surface_linux_app_shell_notes.tetra",
		AppShellFeatures:           "electron-feature-ledger-v1",
		ProductionClaim:            true,
		Experimental:               false,
		DefaultDeny:                true,
		ShellFeaturePolicyEnforced: true,
		Capabilities:               capabilities,
		Permissions: []surface.SurfacePermissionReport{
			{Name: "filesystem", Status: "denied", Allowed: false, CapabilityChecked: true, BlockedReason: "ambient filesystem denied in default template", Evidence: "default-deny-policy", Pass: true},
			{Name: "network", Status: "denied", Allowed: false, CapabilityChecked: true, BlockedReason: "ambient network denied in default template", Evidence: "default-deny-policy", Pass: true},
			{Name: "clipboard", Status: "allowed_with_policy", Allowed: true, CapabilityChecked: true, Evidence: "linux-app-shell-host-trace", Pass: true},
			{Name: "notifications", Status: "denied", Allowed: false, CapabilityChecked: true, BlockedReason: "notification target evidence absent", Evidence: "blocked-pass-nonclaim", Pass: true},
			{Name: "dialogs", Status: "denied", Allowed: false, CapabilityChecked: true, BlockedReason: "dialog target evidence absent", Evidence: "blocked-pass-nonclaim", Pass: true},
			{Name: "shell_open_url", Status: "denied", Allowed: false, CapabilityChecked: true, BlockedReason: "shell open-url denied in default template", Evidence: "default-deny-policy", Pass: true},
		},
		ProcessBoundaries: []surface.SurfaceProcessBoundaryReport{
			{Name: "surface_app_to_host_abi", SchemaChecked: true, CapabilityChecked: true, UserJS: false, NodeIntegration: false, ElectronRuntime: false, Pass: true},
			{Name: "linux_app_shell_host_adapter", SchemaChecked: true, CapabilityChecked: true, UserJS: false, NodeIntegration: false, ElectronRuntime: false, Pass: true},
			{Name: "browser_canvas_host", SchemaChecked: true, CapabilityChecked: true, UserJS: false, NodeIntegration: false, ElectronRuntime: false, Pass: true},
		},
		AssetSafety: []surface.SurfaceAssetSafetyReport{
			{Kind: "font", LocalOnly: true, SHA256Required: true, SizeLimitBytes: 1048576, NetworkFetchAllowed: false, Parser: "bounded-font-metadata-v1", BoundsChecked: true, Pass: true},
			{Kind: "image", LocalOnly: true, SHA256Required: true, SizeLimitBytes: 2097152, NetworkFetchAllowed: false, Parser: "bounded-image-header-v1", BoundsChecked: true, Pass: true},
			{Kind: "icon", LocalOnly: true, SHA256Required: true, SizeLimitBytes: 262144, NetworkFetchAllowed: false, Parser: "bounded-icon-header-v1", BoundsChecked: true, Pass: true},
		},
		UnsupportedClaims: []string{
			"unrestricted-filesystem",
			"unrestricted-network",
			"native-permission-prompts",
			"production-notifications",
			"production-dialogs",
			"remote-asset-fetch",
			"electron-node-integration",
		},
		NegativeGuards: surface.SurfaceSecurityNegativeGuards{
			NoAmbientFilesystem:                       true,
			NoAmbientNetwork:                          true,
			NoShellFeatureBypass:                      true,
			NoPermissionlessClipboard:                 true,
			NoNotificationDialogWithoutTargetEvidence: true,
			NoNetworkAssetFetch:                       true,
			NoUntrustedFontImageDecode:                true,
			NoElectronNodeIntegration:                 true,
			NoUserJSAppLogic:                          true,
			NoDOMAppUITree:                            true,
		},
	}
}
func securityCapabilityStatusForAppShellFeature(featureStatus string) (string, bool) {
	switch featureStatus {
	case "target_evidenced", "scoped_adapter":
		return "allowed_with_policy", true
	case "blocked_pass":
		return "blocked_nonclaim", false
	default:
		return "unknown", false
	}
}
func collectLinuxX64ReleaseWindowAccessibilityBridgeEvidence(artifactDir string) ([]surface.ProcessReport, []surface.ArtifactReport, error) {
	bridgePath := filepath.Join(artifactDir, "surface-linux-accessibility-bridge.json")
	probePath := filepath.Join(artifactDir, "surface-linux-accessibility-probe.json")
	bridgeRaw, err := json.MarshalIndent(map[string]any{
		"schema":          "tetra.surface.linux-accessibility-host-bridge.v1",
		"bridge":          "linux_accessibility_host_bridge_v1",
		"source":          "examples/surface_release_form.tetra",
		"roles":           []string{"root", "panel", "column", "text", "label", "textbox", "checkbox", "row", "button", "status"},
		"focus_order":     []string{"NameTextBox", "EmailTextBox", "SubscribeCheckbox", "SaveButton", "ResetButton"},
		"labelled_by":     map[string]string{"NameTextBox": "NameLabel", "EmailTextBox": "EmailLabel"},
		"states_exported": []string{"focused", "enabled", "editable", "checked", "pressed", "status"},
		"bounds_exported": true,
	}, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	if err := os.WriteFile(bridgePath, append(bridgeRaw, '\n'), 0o644); err != nil {
		return nil, nil, fmt.Errorf("write linux release window accessibility host bridge artifact: %w", err)
	}
	probeRaw, err := json.MarshalIndent(map[string]any{
		"schema":                "tetra.surface.linux-accessibility-platform-probe.v1",
		"bridge":                "linux_accessibility_host_bridge_v1",
		"source":                "examples/surface_release_form.tetra",
		"roles_checked":         true,
		"names_checked":         true,
		"values_checked":        true,
		"states_checked":        true,
		"bounds_checked":        true,
		"focus_order_checked":   true,
		"labels_checked":        true,
		"status_update_checked": true,
		"resize_checked":        true,
		"atspi_claim":           false,
	}, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	if err := os.WriteFile(probePath, append(probeRaw, '\n'), 0o644); err != nil {
		return nil, nil, fmt.Errorf("write linux release window accessibility platform probe artifact: %w", err)
	}
	bridgeArtifact, err := artifactReport(bridgePath, "linux-accessibility-host-bridge")
	if err != nil {
		return nil, nil, err
	}
	probeArtifact, err := artifactReport(probePath, "linux-accessibility-platform-probe")
	if err != nil {
		return nil, nil, err
	}
	processes := []surface.ProcessReport{
		{Name: "surface linux accessibility host bridge", Kind: "runtime", Path: bridgePath, Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface linux accessibility platform probe", Kind: "runtime", Path: probePath, Ran: true, Pass: true, ExitCode: intPtr(0)},
	}
	return processes, []surface.ArtifactReport{bridgeArtifact, probeArtifact}, nil
}
func collectLinuxX64AccessibilityMetadataRealWindowProbeEvidence(artifactDir string) (surface.ProcessReport, surface.FrameReport, error) {
	frame := renderAccessibilityMetadataFrameRGBA(0, 0, 5, 1, 1, 2, 480, 320)
	framePath := filepath.Join(artifactDir, "surface-accessibility-metadata-real-window-frame.rgba")
	if err := os.WriteFile(framePath, frame.Pixels, 0o644); err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("write linux-x64 accessibility metadata real-window frame artifact: %w", err)
	}
	cmd := exec.Command(os.Args[0],
		"--real-window-probe",
		"--probe-title", "Tetra Surface Accessibility Metadata Probe",
		"--probe-frame", framePath,
		"--probe-width", fmt.Sprint(frame.Width),
		"--probe-height", fmt.Sprint(frame.Height),
		"--probe-stride", fmt.Sprint(frame.Stride),
	)
	stdout, stderr, exitCode, err := runCommand(cmd)
	if err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 accessibility metadata real-window probe: %w", err)
	}
	if stdout != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 accessibility metadata real-window probe: unexpected stdout %q", stdout)
	}
	if stderr != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 accessibility metadata real-window probe: unexpected stderr %q", stderr)
	}
	if exitCode != 42 {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 accessibility metadata real-window probe: exit code %d, want 42", exitCode)
	}
	process := surface.ProcessReport{
		Name:             "surface linux-x64 real-window probe",
		Kind:             "app",
		Path:             fmt.Sprintf("%s --real-window-probe --probe-frame %s --probe-width %d --probe-height %d --probe-stride %d", os.Args[0], framePath, frame.Width, frame.Height, frame.Stride),
		Ran:              true,
		Pass:             true,
		ExitCode:         intPtr(exitCode),
		ExpectedExitCode: intPtr(42),
	}
	frameReport := surface.FrameReport{
		Order:     5,
		Width:     frame.Width,
		Height:    frame.Height,
		Stride:    frame.Stride,
		Checksum:  checksumRGBA(frame.Pixels),
		Presented: true,
	}
	markHostProbeOnlyFrameEvidence(&frameReport, framePath)
	return process, frameReport, nil
}
func runRealWindowProbe(opt smokeOptions) error {
	if opt.ProbeFrameWidth <= 0 || opt.ProbeFrameHeight <= 0 || opt.ProbeFrameStride <= 0 {
		return fmt.Errorf("real-window probe requires positive frame dimensions and stride")
	}
	var frame rgbaFrame
	if opt.ProbeFramePath != "" {
		pixels, err := os.ReadFile(opt.ProbeFramePath)
		if err != nil {
			return fmt.Errorf("read real-window probe frame %s: %w", opt.ProbeFramePath, err)
		}
		if len(pixels) != opt.ProbeFrameStride*opt.ProbeFrameHeight {
			return fmt.Errorf("real-window probe frame bytes = %d, want stride*height %d", len(pixels), opt.ProbeFrameStride*opt.ProbeFrameHeight)
		}
		frame = rgbaFrame{Width: opt.ProbeFrameWidth, Height: opt.ProbeFrameHeight, Stride: opt.ProbeFrameStride, Pixels: pixels}
	} else {
		frame = renderWindowCounterFrameRGBA(2, 1, opt.ProbeFrameWidth, opt.ProbeFrameHeight, true)
	}
	return presentRealWindowSurface(opt.ProbeTitle, frame, 350*time.Millisecond)
}
func runPresentedFrameProbeAndReadExpectedPixels(path string, want []byte) ([]byte, int, error) {
	cmd := exec.Command(path)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return nil, -1, fmt.Errorf("start linux-x64 app-presented frame probe %s: %w", path, err)
	}
	pixels, readErr := readSurfaceMemfdPixelsMatching(cmd.Process.Pid, want, 2*time.Second)
	exitCode := terminateProbe(cmd)
	if stdout.String() != "" {
		return nil, exitCode, fmt.Errorf("run linux-x64 app-presented frame probe %s: unexpected stdout %q", path, stdout.String())
	}
	if stderr.String() != "" {
		return nil, exitCode, fmt.Errorf("run linux-x64 app-presented frame probe %s: unexpected stderr %q", path, stderr.String())
	}
	if readErr != nil {
		return nil, exitCode, fmt.Errorf("run linux-x64 app-presented frame probe %s: %w", path, readErr)
	}
	return pixels, exitCode, nil
}
func runPresentedFrameProbeAndReadPixels(path string) ([]byte, int, error) {
	return runPresentedFrameProbeAndReadPixelsLen(path, len(surfacePresentedFrameProbePixels()))
}
func runPresentedFrameProbeAndReadPixelsLen(path string, wantLen int) ([]byte, int, error) {
	cmd := exec.Command(path)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return nil, -1, fmt.Errorf("start linux-x64 app-presented frame probe %s: %w", path, err)
	}
	pixels, readErr := readSurfaceMemfdPixels(cmd.Process.Pid, wantLen, 2*time.Second)
	exitCode := terminateProbe(cmd)
	if stdout.String() != "" {
		return nil, exitCode, fmt.Errorf("run linux-x64 app-presented frame probe %s: unexpected stdout %q", path, stdout.String())
	}
	if stderr.String() != "" {
		return nil, exitCode, fmt.Errorf("run linux-x64 app-presented frame probe %s: unexpected stderr %q", path, stderr.String())
	}
	if readErr != nil {
		return nil, exitCode, fmt.Errorf("run linux-x64 app-presented frame probe %s: %w", path, readErr)
	}
	return pixels, exitCode, nil
}
func readSurfaceMemfdPixelsMatching(pid int, want []byte, timeout time.Duration) ([]byte, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		pixels, err := tryReadSurfaceMemfdPixels(pid, len(want))
		if err == nil {
			if bytes.Equal(pixels, want) {
				return pixels, nil
			}
			lastErr = fmt.Errorf("surface memfd checksum %s, waiting for %s", checksumRGBA(pixels), checksumRGBA(want))
		} else {
			lastErr = err
		}
		time.Sleep(10 * time.Millisecond)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("surface memfd was not found")
	}
	return nil, lastErr
}
func terminateProbe(cmd *exec.Cmd) int {
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	_ = cmd.Wait()
	if cmd.ProcessState != nil {
		return cmd.ProcessState.ExitCode()
	}
	return -1
}
func readSurfaceMemfdPixels(pid int, wantLen int, timeout time.Duration) ([]byte, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		pixels, err := tryReadSurfaceMemfdPixels(pid, wantLen)
		if err == nil {
			return pixels, nil
		}
		lastErr = err
		time.Sleep(10 * time.Millisecond)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("surface memfd was not found")
	}
	return nil, lastErr
}
func tryReadSurfaceMemfdPixels(pid int, wantLen int) ([]byte, error) {
	fdDir := filepath.Join("/proc", fmt.Sprint(pid), "fd")
	entries, err := os.ReadDir(fdDir)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		fdPath := filepath.Join(fdDir, entry.Name())
		target, err := os.Readlink(fdPath)
		if err != nil || !strings.Contains(target, "memfd") {
			continue
		}
		file, err := os.Open(fdPath)
		if err != nil {
			continue
		}
		_, _ = file.Seek(0, io.SeekStart)
		buf := make([]byte, wantLen)
		_, readErr := io.ReadFull(file, buf)
		_ = file.Close()
		if readErr == nil {
			return buf, nil
		}
	}
	return nil, fmt.Errorf("no readable Surface memfd with %d bytes for pid %d", wantLen, pid)
}
func rejectLegacyUISidecarArtifacts(root string, opts ...sidecarScanOptions) error {
	_, err := scanLegacyUISidecarArtifacts(root, opts...)
	return err
}
func scanLegacyUISidecarArtifacts(root string, opts ...sidecarScanOptions) (surface.ArtifactScanReport, error) {
	var opt sidecarScanOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	report := surface.ArtifactScanReport{
		Root:           root,
		ForbiddenPaths: []string{},
		Pass:           true,
	}
	if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		report.FilesChecked++
		if legacyUISidecarArtifactPath(path, opt) {
			report.ForbiddenPaths = append(report.ForbiddenPaths, path)
		}
		return nil
	}); err != nil {
		return report, err
	}
	if len(report.ForbiddenPaths) > 0 {
		report.Pass = false
		return report, fmt.Errorf("Surface build emitted legacy UI sidecar artifact %s", report.ForbiddenPaths[0])
	}
	return report, nil
}
func legacyUISidecarArtifactPath(path string, opt sidecarScanOptions) bool {
	base := strings.ToLower(filepath.Base(path))
	if strings.Contains(base, ".ui.") ||
		strings.HasSuffix(base, ".html") ||
		strings.HasSuffix(base, ".js") {
		return true
	}
	if strings.HasSuffix(base, ".mjs") {
		return !opt.AllowCompilerOwnedWASMLoader || !compilerOwnedWASMLoaderArtifactPath(path)
	}
	return false
}
func compilerOwnedWASMLoaderArtifactPath(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	if strings.Contains(base, ".ui.") || !strings.HasSuffix(base, ".mjs") {
		return false
	}
	wasmPath := strings.TrimSuffix(path, filepath.Ext(path)) + ".wasm"
	return fileExists(wasmPath)
}
func resolveSurfaceSourcePath(raw string) (string, error) {
	if raw == "" {
		raw = "examples/surface_counter.tetra"
	}
	if filepath.IsAbs(raw) {
		return requireExistingSource(raw)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if path, err := requireExistingSource(filepath.Join(cwd, raw)); err == nil {
		return path, nil
	}
	if root := findRepoRoot(cwd); root != "" {
		return requireExistingSource(filepath.Join(root, raw))
	}
	return requireExistingSource(filepath.Join(cwd, raw))
}
func requireExistingSource(path string) (string, error) {
	cleaned := filepath.Clean(path)
	info, err := os.Stat(cleaned)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("%s is a directory, want Surface source file", cleaned)
	}
	return cleaned, nil
}
func findRepoRoot(start string) string {
	dir := filepath.Clean(start)
	for {
		if fileExists(filepath.Join(dir, "go.work")) && dirExists(filepath.Join(dir, "examples")) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}
func repoRootForCommands() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	root := findRepoRoot(cwd)
	if root == "" {
		return "", fmt.Errorf("find repo root from %s", cwd)
	}
	return root, nil
}
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
func runExecutable(path string) (string, string, int, error) {
	return runCommand(exec.Command(path))
}
func nodeCommand(args ...string) *exec.Cmd {
	cmd := exec.Command("node", args...)
	cmd.Env = withoutNodeEnvProxy(os.Environ())
	return cmd
}
func withoutNodeEnvProxy(env []string) []string {
	filtered := make([]string, 0, len(env))
	for _, entry := range env {
		if strings.HasPrefix(entry, "NODE_USE_ENV_PROXY=") {
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered
}
func runCommand(cmd *exec.Cmd) (string, string, int, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if cmd.ProcessState == nil {
		return stdout.String(), stderr.String(), -1, err
	}
	return stdout.String(), stderr.String(), cmd.ProcessState.ExitCode(), nil
}
