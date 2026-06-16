package compiler_test

import (
	"path/filepath"
	"strings"
	"testing"

	compiler "tetra_language/compiler"
	"tetra_language/compiler/internal/testkit"
)

func TestSurfaceReleaseTextInputExampleLoadsCoreTextModule(t *testing.T) {
	entry := testkit.RepoPath(t, "examples", "surface_release_text_input.tetra")
	world, err := compiler.LoadWorld(entry)
	if err != nil {
		t.Fatalf("LoadWorld(%s): %v", filepath.ToSlash(entry), err)
	}

	for _, module := range []string{"lib.core.surface", "lib.core.draw", "lib.core.text"} {
		if _, ok := world.ByModule[module]; !ok {
			t.Fatalf("surface release text input did not load module %s; modules=%v", module, world.ByModule)
		}
	}

	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld(surface release text input): %v", err)
	}
}

func TestSurfaceReleaseTextInputExampleBuildsLinuxX64(t *testing.T) {
	entry := testkit.RepoPath(t, "examples", "surface_release_text_input.tetra")
	out := filepath.Join(t.TempDir(), "surface-release-text-input")

	if _, err := compiler.BuildFileWithStatsOpt(entry, out, "linux-x64", compiler.BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt(surface release text input): %v", err)
	}
}

func TestSurfaceReleaseCounterExampleLoadsStableWidgetAccessibilityModules(t *testing.T) {
	entry := testkit.RepoPath(t, "examples", "surface_release_counter.tetra")
	world, err := compiler.LoadWorld(entry)
	if err != nil {
		t.Fatalf("LoadWorld(%s): %v", filepath.ToSlash(entry), err)
	}

	for _, module := range []string{
		"lib.core.surface",
		"lib.core.draw",
		"lib.core.component",
		"lib.core.widgets",
		"lib.core.style",
		"lib.core.accessibility",
	} {
		if _, ok := world.ByModule[module]; !ok {
			t.Fatalf("surface release counter did not load module %s; modules=%v", module, world.ByModule)
		}
	}

	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld(surface release counter): %v", err)
	}
}

func TestSurfaceModuleDefinesClipboardAndCompositionABIWrappers(t *testing.T) {
	tmp := t.TempDir()
	testkit.WriteFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

func main() -> Int
uses alloc, mem, surface:
    var win: surface.Surface = surface.open("clipboard-ime", 160, 80)
    var text: []u8 = core.make_u8(3)
    text[0] = 84
    text[1] = 101
    text[2] = 116
    var out: []u8 = core.make_u8(3)
    var slots: []i32 = core.make_i32(4)
    let wrote: Int = surface.clipboard_write_text(win, text)
    let read: Int = surface.clipboard_read_text_into(win, out)
    let copied: Int = surface.poll_composition_into(win, slots)
    let trace: surface.CompositionTrace = surface.poll_composition(win)
    let closed: Int = surface.close(win)
    if wrote == 3 && read == 3 && copied == 4 && trace.start && trace.update && trace.commit && trace.cancel && surface.event_composition_start() == 10 && closed == 0:
        return 0
    return 1
`,
	})

	entry := filepath.Join(tmp, "app", "main.t4")
	if _, err := compiler.BuildFileWithStatsOpt(entry, filepath.Join(t.TempDir(), "surface-clipboard-ime"), "linux-x64", compiler.BuildOptions{
		DependencyRoots: []compiler.ModuleRoot{{Root: testkit.RepoRoot(t)}},
		Jobs:            1,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt(lib.core.surface clipboard/IME consumer): %v", err)
	}
}

func TestSurfaceClipboardRejectsBorrowedTextBoundaryWithoutCopy(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "surface wrapper",
			body: `
    let copied: Int = surface.clipboard_write_text(win, borrowed)
`,
			want: "borrowed value derived from 'xs' cannot be passed to non-borrow parameter 2 of 'lib.core.surface.clipboard_write_text'",
		},
		{
			name: "raw host abi",
			body: `
    let copied: Int = core.surface_clipboard_write_text(win.handle, borrowed)
`,
			want: "borrowed value derived from 'xs' cannot be passed to non-borrow parameter 2 of 'core.surface_clipboard_write_text'",
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			requireSurfaceCheckErrorContains(t, map[string]string{
				"app/main.t4": `module app.main
import lib.core.surface as surface

func main() -> Int
uses alloc, mem, surface:
    var win: surface.Surface = surface.open("clipboard-borrow", 160, 80)
    var xs: []u8 = core.make_u8(4)
    let borrowed: []u8 = xs.window(0, 3).borrow()
` + tc.body + `
    let closed: Int = surface.close(win)
    return copied + closed
`,
			}, tc.want)
		})
	}
}

func TestSurfaceClipboardAcceptsCopiedTextBoundary(t *testing.T) {
	tmp := t.TempDir()
	testkit.WriteFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

func main() -> Int
uses alloc, mem, surface:
    var win: surface.Surface = surface.open("clipboard-copy", 160, 80)
    var xs: []u8 = core.make_u8(4)
    let copied_text: []u8 = xs.window(0, 3).borrow().copy()
    let copied: Int = surface.clipboard_write_text(win, copied_text)
    let closed: Int = surface.close(win)
    return copied + closed
`,
	})

	entry := filepath.Join(tmp, "app", "main.t4")
	world, err := compiler.LoadWorldOpt(entry, compiler.WorldOptions{
		DependencyRoots: []compiler.ModuleRoot{{
			Root: testkit.RepoRoot(t),
		}},
	})
	if err != nil {
		t.Fatalf("LoadWorldOpt: %v", err)
	}
	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld(copied clipboard boundary): %v", err)
	}
}

func TestSurfaceSafeViewLifetimeRejectsBorrowedTextBoxBuffer(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface
import lib.core.widgets as widgets

func bad_textbox_init(xs: borrow []u8) -> widgets.TextBox
uses alloc, mem:
    let rect: surface.Rect = surface.Rect(x: 0, y: 0, w: 160, h: 48)
    var storage: []u8 = core.make_u8(8)
    var box: widgets.TextBox = widgets.TextBox(rect: rect, focused: false, text_len: 0, caret: 0, buffer: storage)
    let ok: Int = widgets.textbox_init(box, rect, xs.window(0, 2).borrow())
    return box

func main() -> Int:
    return 0
`,
	}, "borrowed value derived from 'xs' cannot be passed to non-borrow parameter 3 of 'lib.core.widgets.textbox_init'")
}

func TestSurfaceSafeViewLifetimeRejectsBorrowedWidgetStateLabel(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main

struct WidgetState:
    label: String

func bad_widget_label(text: borrow String) -> WidgetState:
    return WidgetState(label: text.window(0, 2).borrow())

func main() -> Int:
    return 0
`,
	}, "aggregate 'WidgetState' contains borrowed String field 'label' that cannot escape through owned return")
}

func TestSurfaceSafeViewLifetimeRejectsBorrowedAccessibilityLabel(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.accessibility as accessibility

struct AccessibilityLabelState:
    label: String
    metadata: accessibility.NodeMetadata

func bad_accessibility_label(text: borrow String) -> AccessibilityLabelState:
    let metadata: accessibility.NodeMetadata = accessibility.label_metadata(1, accessibility.value_name(), 1)
    return AccessibilityLabelState(label: text.window(0, 2).borrow(), metadata: metadata)

func main() -> Int:
    return 0
`,
	}, "aggregate 'AccessibilityLabelState' contains borrowed String field 'label' that cannot escape through owned return")
}

func TestSurfaceSafeViewLifetimeAcceptsOwnedCopyState(t *testing.T) {
	tmp := t.TempDir()
	testkit.WriteFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.core.accessibility as accessibility
import lib.core.surface as surface
import lib.core.widgets as widgets

struct AccessibilityLabelState:
    label: String
    metadata: accessibility.NodeMetadata

struct WidgetState:
    label: String
    box: widgets.TextBox
    accessibility_label: AccessibilityLabelState

func good_state(text: borrow String, bytes: borrow []u8) -> WidgetState
uses alloc, mem:
    let rect: surface.Rect = surface.Rect(x: 0, y: 0, w: 160, h: 48)
    let copied_label: String = text.window(0, 2).copy()
    let copied_buffer: []u8 = bytes.window(0, 2).copy()
    let metadata: accessibility.NodeMetadata = accessibility.label_metadata(1, accessibility.value_name(), 1)
    let label_state: AccessibilityLabelState = AccessibilityLabelState(label: copied_label.copy(), metadata: metadata)
    let box: widgets.TextBox = widgets.TextBox(rect: rect, focused: false, text_len: 0, caret: 0, buffer: copied_buffer)
    return WidgetState(label: copied_label, box: box, accessibility_label: label_state)

func main() -> Int:
    return 0
`,
	})

	entry := filepath.Join(tmp, "app", "main.t4")
	world, err := compiler.LoadWorldOpt(entry, compiler.WorldOptions{
		DependencyRoots: []compiler.ModuleRoot{{
			Root: testkit.RepoRoot(t),
		}},
	})
	if err != nil {
		t.Fatalf("LoadWorldOpt: %v", err)
	}
	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld(surface safe-view owned-copy state): %v", err)
	}
}

func TestSurfaceMigrationExamplesCheck(t *testing.T) {
	examples := []string{
		filepath.Join("examples", "surface_migration_ui_web_smoke.tetra"),
		filepath.Join("examples", "surface_migration_ui_native_shell_smoke.tetra"),
		filepath.Join("examples", "surface_migration_dogfood_web_ui.tetra"),
		filepath.Join("examples", "surface_migration_tetra_control_center.tetra"),
	}

	for _, rel := range examples {
		rel := rel
		t.Run(filepath.ToSlash(rel), func(t *testing.T) {
			entry := testkit.RepoPath(t, rel)
			world, err := compiler.LoadWorld(entry)
			if err != nil {
				t.Fatalf("LoadWorld(%s): %v", filepath.ToSlash(entry), err)
			}
			if _, err := compiler.CheckWorld(world); err != nil {
				t.Fatalf("CheckWorld(%s): %v", filepath.ToSlash(entry), err)
			}
		})
	}
}

func TestSurfaceFrameCannotEscapeViaReturn(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

func leak(win: borrow surface.Surface) -> surface.Frame
uses alloc, mem, surface:
    let frame: surface.Frame = surface.begin_frame(win)
    return frame

func main() -> Int:
    return 0
`,
	}, "surface value 'lib.core.surface.Frame' cannot escape via return")
}

func TestSurfaceFramePixelsCannotEscapeViaReturn(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

func leak(win: borrow surface.Surface) -> []u8
uses alloc, mem, surface:
    let frame: surface.Frame = surface.begin_frame(win)
    return frame.pixels

func main() -> Int:
    return 0
`,
	}, "surface frame pixels cannot escape via return")
}

func TestSurfaceFramePixelsAliasCannotEscapeViaReturn(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

func leak(win: borrow surface.Surface) -> []u8
uses alloc, mem, surface:
    let frame: surface.Frame = surface.begin_frame(win)
    var pixels: []u8 = frame.pixels
    return pixels

func main() -> Int:
    return 0
`,
	}, "surface frame pixels cannot escape via return")
}

func TestSurfaceFramePixelsCannotEscapeViaStructConstructorReturn(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

struct PixelBox:
    pixels: []u8

func leak(win: borrow surface.Surface) -> PixelBox
uses alloc, mem, surface:
    let frame: surface.Frame = surface.begin_frame(win)
    return PixelBox(pixels: frame.pixels)

func main() -> Int:
    return 0
`,
	}, "surface frame pixels cannot escape via return")
}

func TestSurfaceFramePixelsCannotEscapeViaInoutAssignment(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

func leak(win: borrow surface.Surface, out: inout []u8) -> Int
uses alloc, mem, surface:
    let frame: surface.Frame = surface.begin_frame(win)
    out = frame.pixels
    return 0

func main() -> Int:
    return 0
`,
	}, "surface frame pixels cannot escape via inout assignment to 'out'")
}

func TestSurfaceEventCannotBeStoredInGlobalState(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

var leaked: surface.Event

func main() -> Int:
    return 0
`,
	}, "surface value 'lib.core.surface.Event' cannot be stored in global 'leaked'")
}

func TestSurfaceEventCannotBeStoredInUserStructField(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

struct EventBox:
    event: surface.Event

func main() -> Int:
    return 0
`,
	}, "surface value 'lib.core.surface.Event' cannot be stored in struct field 'event'")
}

func TestSurfaceDrawContextCannotBeStoredInUserStructField(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.draw as draw

struct ContextBox:
    ctx: draw.DrawContext

func main() -> Int:
    return 0
`,
	}, "surface value 'lib.core.draw.DrawContext' cannot be stored in struct field 'ctx'")
}

func TestSurfaceEventCannotBeStoredInUserEnumPayload(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

enum EventSlot:
    case event(surface.Event)

func main() -> Int:
    return 0
`,
	}, "surface value 'lib.core.surface.Event' cannot be stored in enum payload 'event'")
}

func TestSurfaceDrawContextCannotBeStoredInUserEnumPayload(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.draw as draw

enum ContextSlot:
    case ctx(draw.DrawContext)

func main() -> Int:
    return 0
`,
	}, "surface value 'lib.core.draw.DrawContext' cannot be stored in enum payload 'ctx'")
}

func TestSurfaceEventCannotEscapeViaThrow(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

func leak(win: borrow surface.Surface) -> Int throws surface.Event
uses alloc, mem, surface:
    let event: surface.Event = surface.poll_event(win)
    throw event

func main() -> Int:
    return 0
`,
	}, "surface value 'lib.core.surface.Event' cannot escape via throw")
}

func TestSurfaceEventCannotEscapeViaFunctionTypedReturnCapture(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

func pick() -> fn(Int) -> Int:
    let event: surface.Event = surface.Event(kind: 5, x: 0, y: 0, button: 0, key: 0, width: 1, height: 1, timestamp_ms: 0, text_len: 0)
    let cb: fn(Int) -> Int = fn(x: Int) -> Int:
        return event.kind + x
    return cb

func main() -> Int:
    return 0
`,
	}, "surface value 'lib.core.surface.Event' cannot escape via function capture")
}

func TestSurfaceFramePixelsCannotEscapeViaFunctionTypedReturnCapture(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

func pick(win: borrow surface.Surface) -> fn(Int) -> Int
uses alloc, mem, surface:
    let frame: surface.Frame = surface.begin_frame(win)
    let pixels: []u8 = frame.pixels
    let cb: fn(Int) -> Int = fn(x: Int) -> Int:
        return pixels[0] + x
    return cb

func main() -> Int:
    return 0
`,
	}, "surface frame pixels cannot escape via function capture")
}

func TestSurfaceFramePixelsCannotEscapeViaThrow(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

func leak(win: borrow surface.Surface) -> Int throws []u8
uses alloc, mem, surface:
    let frame: surface.Frame = surface.begin_frame(win)
    throw frame.pixels

func main() -> Int:
    return 0
`,
	}, "surface frame pixels cannot escape via throw")
}

func TestSurfaceEventCannotCrossTypedTaskErrorBoundary(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

enum TaskErr:
    case event(surface.Event)

func worker() -> Int throws TaskErr:
    return 42

func caller() -> Int throws TaskErr
uses runtime:
    let task: task.i32 = core.task_spawn_i32_typed<TaskErr>("worker")
    return try core.task_join_i32_typed<TaskErr>(task)

func main() -> Int:
    return 0
`,
	}, "surface value 'lib.core.surface.Event' cannot be stored in enum payload 'event'")
}

func TestSurfaceEventCannotCrossTypedActorMessageBoundary(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

enum SurfaceMsg:
    case event(surface.Event)

func main() -> Int
uses actors:
    let msg: SurfaceMsg = core.recv_typed<SurfaceMsg>()
    return 0
`,
	}, "surface value 'lib.core.surface.Event' cannot be stored in enum payload 'event'")
}

func TestSurfaceHandleCannotCrossTypedTaskErrorBoundary(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

enum TaskErr:
    case window(surface.Surface)

func worker() -> Int throws TaskErr:
    return 42

func caller() -> Int throws TaskErr
uses runtime:
    let task: task.i32 = core.task_spawn_i32_typed<TaskErr>("worker")
    return try core.task_join_i32_typed<TaskErr>(task)

func main() -> Int:
    return 0
`,
	}, "surface value 'lib.core.surface.Surface' cannot cross actor/task boundary")
}

func TestSurfaceHandleCannotCrossTypedActorMessageBoundary(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

enum SurfaceMsg:
    case window(surface.Surface)

func main() -> Int
uses actors:
    let msg: SurfaceMsg = core.recv_typed<SurfaceMsg>()
    return 0
`,
	}, "surface value 'lib.core.surface.Surface' cannot cross actor/task boundary")
}

func TestSurfaceDrawContextCannotEscapeViaReturn(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface
import lib.core.draw as draw

func leak(win: borrow surface.Surface) -> draw.DrawContext
uses alloc, mem, surface:
    var frame: surface.Frame = surface.begin_frame(win)
    var ctx: draw.DrawContext = draw.DrawContext(frame: frame)
    return ctx

func main() -> Int:
    return 0
`,
	}, "surface value 'lib.core.draw.DrawContext' cannot escape via return")
}

func TestSurfaceFrameCannotEscapeViaInoutAssignment(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

func leak(win: borrow surface.Surface, out: inout surface.Frame) -> Int
uses alloc, mem, surface:
    out = surface.begin_frame(win)
    return 0

func main() -> Int:
    return 0
`,
	}, "surface value 'lib.core.surface.Frame' cannot escape via inout assignment to 'out'")
}

func TestSurfaceDrawContextCannotUseFrameAfterPresent(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface
import lib.core.draw as draw

func main() -> Int
uses alloc, mem, surface:
    let win: surface.Surface = surface.open("demo", 2, 2)
    var frame: surface.Frame = surface.begin_frame(win)
    var ctx: draw.DrawContext = draw.DrawContext(frame: frame)
    let color: surface.Color = surface.Color(r: 0, g: 0, b: 0, a: 255)
    let presented: Int = surface.present(ctx.frame)
    let draw_status: Int = draw.clear(ctx, color)
    let closed: Int = surface.close(win)
    return presented + draw_status + closed
`,
	}, "cannot use consumed value 'ctx.frame'")
}

func TestSurfaceFramePixelsAliasCannotBeUsedAfterPresent(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

func main() -> Int
uses alloc, mem, surface:
    let win: surface.Surface = surface.open("demo", 2, 2)
    var frame: surface.Frame = surface.begin_frame(win)
    var pixels: []u8 = frame.pixels
    let presented: Int = surface.present(frame)
    pixels[0] = 255
    let closed: Int = surface.close(win)
    return presented + closed
`,
	}, "surface frame pixels alias 'pixels' cannot be used after frame 'frame' was presented")
}

func TestSurfaceDrawContextFramePixelsAliasCannotBeUsedAfterPresent(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface
import lib.core.draw as draw

func main() -> Int
uses alloc, mem, surface:
    let win: surface.Surface = surface.open("demo", 2, 2)
    var frame: surface.Frame = surface.begin_frame(win)
    var ctx: draw.DrawContext = draw.DrawContext(frame: frame)
    var pixels: []u8 = ctx.frame.pixels
    let presented: Int = surface.present(ctx.frame)
    pixels[0] = 255
    let closed: Int = surface.close(win)
    return presented + closed
`,
	}, "surface frame pixels alias 'pixels' cannot be used after frame 'ctx.frame' was presented")
}

func TestSurfaceDirectHostPresentMarksFramePixelsPresented(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

func main() -> Int
uses alloc, mem, surface:
    let win: surface.Surface = surface.open("demo", 2, 2)
    var frame: surface.Frame = surface.begin_frame(win)
    var pixels: []u8 = frame.pixels
    let raw_present: Int = core.surface_present_rgba(win.handle, frame.pixels, frame.width, frame.height, frame.stride)
    pixels[0] = 255
    let closed: Int = surface.close(win)
    return raw_present + closed
`,
	}, "surface frame pixels alias 'pixels' cannot be used after frame 'frame' was presented")
}

func TestSurfaceDirectHostPresentChecksFrameSurfaceOwner(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

func main() -> Int
uses alloc, mem, surface:
    let win: surface.Surface = surface.open("demo", 2, 2)
    var frame: surface.Frame = surface.begin_frame(win)
    let closed: Int = surface.close(win)
    let raw_present: Int = core.surface_present_rgba(frame.surface.handle, frame.pixels, frame.width, frame.height, frame.stride)
    return raw_present + closed
`,
	}, "cannot use consumed value 'win'")
}

func TestSurfaceAliasCannotBeClosedAfterOwnerClose(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

func main() -> Int
uses surface:
    let win: surface.Surface = surface.open("demo", 2, 2)
    let alias: surface.Surface = win
    let closed: Int = surface.close(win)
    let double_closed: Int = surface.close(alias)
    return closed + double_closed
`,
	}, "cannot use consumed value 'alias'")
}

func TestSurfaceAliasCannotBeUsedAfterOwnerClose(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

func main() -> Int
uses surface:
    let win: surface.Surface = surface.open("demo", 2, 2)
    let alias: surface.Surface = win
    let closed: Int = surface.close(win)
    let redraw: Int = surface.request_redraw(alias)
    return closed + redraw
`,
	}, "cannot use consumed value 'alias'")
}

func TestSurfaceStructLiteralHandleAliasCannotBeUsedAfterOwnerClose(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

func main() -> Int
uses surface:
    let win: surface.Surface = surface.open("demo", 2, 2)
    let forged: surface.Surface = surface.Surface(handle: win.handle, width: win.width, height: win.height)
    let closed: Int = surface.close(win)
    let redraw: Int = surface.request_redraw(forged)
    return closed + redraw
`,
	}, "cannot use consumed value 'forged'")
}

func TestSurfaceFrameCannotBePresentedAfterSurfaceClose(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

func main() -> Int
uses alloc, mem, surface:
    let win: surface.Surface = surface.open("demo", 2, 2)
    let frame: surface.Frame = surface.begin_frame(win)
    let closed: Int = surface.close(win)
    let presented: Int = surface.present(frame)
    return closed + presented
`,
	}, "cannot use consumed value 'win'")
}

func TestSurfaceManualFrameCannotBePresentedAfterSurfaceClose(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

func main() -> Int
uses alloc, mem, surface:
    let win: surface.Surface = surface.open("demo", 2, 2)
    let pixels: []u8 = core.make_u8(16)
    let frame: surface.Frame = surface.Frame(surface: win, width: 2, height: 2, stride: 8, pixels: pixels)
    let closed: Int = surface.close(win)
    let presented: Int = surface.present(frame)
    return closed + presented
`,
	}, "cannot use consumed value 'win'")
}

func TestSurfaceDrawContextFrameCannotBePresentedAfterSurfaceClose(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface
import lib.core.draw as draw

func main() -> Int
uses alloc, mem, surface:
    let win: surface.Surface = surface.open("demo", 2, 2)
    let frame: surface.Frame = surface.begin_frame(win)
    let ctx: draw.DrawContext = draw.DrawContext(frame: frame)
    let closed: Int = surface.close(win)
    let presented: Int = surface.present(ctx.frame)
    return closed + presented
`,
	}, "cannot use consumed value 'win'")
}

func TestSurfaceDrawContextFrameAssignmentTracksNewSurfaceOwner(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface
import lib.core.draw as draw

func main() -> Int
uses alloc, mem, surface:
    let win1: surface.Surface = surface.open("one", 2, 2)
    let frame1: surface.Frame = surface.begin_frame(win1)
    var ctx: draw.DrawContext = draw.DrawContext(frame: frame1)
    let win2: surface.Surface = surface.open("two", 2, 2)
    let frame2: surface.Frame = surface.begin_frame(win2)
    ctx.frame = frame2
    let closed: Int = surface.close(win2)
    let presented: Int = surface.present(ctx.frame)
    let closed1: Int = surface.close(win1)
    return closed + presented + closed1
`,
	}, "cannot use consumed value 'win2'")
}

func TestSurfaceDirectHostCloseConsumesSurfaceHandleOwner(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

func main() -> Int
uses surface:
    let win: surface.Surface = surface.open("demo", 2, 2)
    let raw_close: Int = core.surface_close(win.handle)
    let redraw: Int = surface.request_redraw(win)
    return raw_close + redraw
`,
	}, "cannot use consumed value 'win'")
}

func TestSurfaceDirectHostCloseConsumesSurfaceHandleIntAliasOwner(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

func main() -> Int
uses surface:
    let win: surface.Surface = surface.open("demo", 2, 2)
    let handle: Int = win.handle
    let raw_close: Int = core.surface_close(handle)
    let redraw: Int = surface.request_redraw(win)
    return raw_close + redraw
`,
	}, "cannot use consumed value 'win'")
}

func TestSurfaceDirectHostHandleUseAfterCloseRejected(t *testing.T) {
	requireSurfaceCheckErrorContains(t, map[string]string{
		"app/main.t4": `module app.main
import lib.core.surface as surface

func main() -> Int
uses surface:
    let win: surface.Surface = surface.open("demo", 2, 2)
    let handle: Int = win.handle
    let closed: Int = surface.close(win)
    let redraw: Int = core.surface_request_redraw(handle)
    return closed + redraw
`,
	}, "cannot use consumed value 'win'")
}

func requireSurfaceCheckErrorContains(t *testing.T, files map[string]string, want string) {
	t.Helper()

	tmp := t.TempDir()
	testkit.WriteFiles(t, tmp, files)

	entry := filepath.Join(tmp, "app", "main.t4")
	world, err := compiler.LoadWorldOpt(entry, compiler.WorldOptions{
		DependencyRoots: []compiler.ModuleRoot{{
			Root: testkit.RepoRoot(t),
		}},
	})
	if err != nil {
		t.Fatalf("LoadWorldOpt: %v", err)
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected error containing %q, got: %v", want, err)
	}
}
