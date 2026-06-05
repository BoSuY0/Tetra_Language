package lower

import (
	"testing"

	"tetra_language/compiler/internal/ir"
)

func TestLowerSurfaceHostBuiltinsCallRuntimeABI(t *testing.T) {
	checked := checkCallableProgram(t, `
func main() -> Int
uses surface, alloc, mem:
    let handle: Int = core.surface_open("demo", 10, 10)
    let event: Int = core.surface_poll_event_kind(handle)
    let event_x: Int = core.surface_poll_event_x(handle)
    let event_y: Int = core.surface_poll_event_y(handle)
    let event_button: Int = core.surface_poll_event_button(handle)
    let event_slots: []i32 = core.make_i32(5)
    let event_copied: Int = core.surface_poll_event_into(handle, event_slots)
    let event_text_len: Int = core.surface_poll_event_text_len(handle)
    let text: []u8 = core.make_u8(4)
    let event_text_copied: Int = core.surface_poll_event_text_into(handle, text)
    let clipboard_write: Int = core.surface_clipboard_write_text(handle, text)
    let clipboard_read: Int = core.surface_clipboard_read_text_into(handle, text)
    let composition_slots: []i32 = core.make_i32(4)
    let composition_copied: Int = core.surface_poll_composition_into(handle, composition_slots)
    let _: Int = core.surface_begin_frame(handle)
    let pixels: []u8 = core.make_u8(4)
    let presented: Int = core.surface_present_rgba(handle, pixels, 1, 1, 4)
    let redraw: Int = core.surface_request_redraw(handle)
    let closed: Int = core.surface_close(handle)
    return handle + event + event_x + event_y + event_button + event_copied + event_text_len + event_text_copied + clipboard_write + clipboard_read + composition_copied + presented + redraw + closed + core.surface_now_ms()
`)

	prog, err := Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	mainFn := findIRFuncByName(t, prog.Funcs, "main")

	for _, tc := range []struct {
		name string
		args int
		rets int
	}{
		{name: "__tetra_surface_open", args: 4, rets: 1},
		{name: "__tetra_surface_poll_event_kind", args: 1, rets: 1},
		{name: "__tetra_surface_poll_event_x", args: 1, rets: 1},
		{name: "__tetra_surface_poll_event_y", args: 1, rets: 1},
		{name: "__tetra_surface_poll_event_button", args: 1, rets: 1},
		{name: "__tetra_surface_poll_event_into", args: 3, rets: 1},
		{name: "__tetra_surface_poll_event_text_len", args: 1, rets: 1},
		{name: "__tetra_surface_poll_event_text_into", args: 3, rets: 1},
		{name: "__tetra_surface_clipboard_write_text", args: 3, rets: 1},
		{name: "__tetra_surface_clipboard_read_text_into", args: 3, rets: 1},
		{name: "__tetra_surface_poll_composition_into", args: 3, rets: 1},
		{name: "__tetra_surface_begin_frame", args: 1, rets: 1},
		{name: "__tetra_surface_present_rgba", args: 6, rets: 1},
		{name: "__tetra_surface_request_redraw", args: 1, rets: 1},
		{name: "__tetra_surface_close", args: 1, rets: 1},
		{name: "__tetra_surface_now_ms", args: 0, rets: 1},
	} {
		if countSurfaceRuntimeCall(mainFn.Instrs, tc.name, tc.args, tc.rets) != 1 {
			t.Fatalf("main missing one %s(%d)->%d call: %#v", tc.name, tc.args, tc.rets, mainFn.Instrs)
		}
	}
}

func countSurfaceRuntimeCall(instrs []ir.IRInstr, name string, args int, rets int) int {
	count := 0
	for _, instr := range instrs {
		if instr.Kind == ir.IRCall && instr.Name == name && instr.ArgSlots == args && instr.RetSlots == rets {
			count++
		}
	}
	return count
}
