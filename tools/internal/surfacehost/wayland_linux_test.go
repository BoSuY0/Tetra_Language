//go:build linux

package surfacehost

import (
	"encoding/binary"
	"testing"
)

func TestWaylandPointerMotionQueuesSurfaceEvent(t *testing.T) {
	client := &waylandClient{pointerID: 7}
	payload := make([]byte, 12)
	binary.LittleEndian.PutUint32(payload[0:4], 123)
	binary.LittleEndian.PutUint32(payload[4:8], uint32(48<<8))
	binary.LittleEndian.PutUint32(payload[8:12], uint32(96<<8))

	if err := client.handleCommonEvent(7, 2, payload, 0, 0, 320, 200); err != nil {
		t.Fatalf("handle pointer motion: %v", err)
	}
	if len(client.events) != 1 {
		t.Fatalf("events = %d, want 1", len(client.events))
	}
	event := client.events[0]
	if event.Kind != 3 || event.X != 48 || event.Y != 96 || event.Width != 320 ||
		event.Height != 200 ||
		event.TimestampMS != 123 {
		t.Fatalf("event = %#v, want mouse move at 48,96", event)
	}
}

func TestWaylandPointerButtonQueuesSurfaceEvent(t *testing.T) {
	client := &waylandClient{pointerID: 7, pointerX: 48, pointerY: 96}
	payload := make([]byte, 16)
	binary.LittleEndian.PutUint32(payload[4:8], 124)
	binary.LittleEndian.PutUint32(payload[8:12], 272)
	binary.LittleEndian.PutUint32(payload[12:16], 1)

	if err := client.handleCommonEvent(7, 3, payload, 0, 0, 320, 200); err != nil {
		t.Fatalf("handle pointer button: %v", err)
	}
	if len(client.events) != 1 {
		t.Fatalf("events = %d, want 1", len(client.events))
	}
	event := client.events[0]
	if event.Kind != 4 || event.X != 48 || event.Y != 96 || event.Button != 1 ||
		event.TimestampMS != 124 {
		t.Fatalf("event = %#v, want mouse down button 1 at 48,96", event)
	}
}

func TestWaylandKeyboardKeyQueuesSurfaceEvent(t *testing.T) {
	client := &waylandClient{keyboardID: 8}
	payload := make([]byte, 16)
	binary.LittleEndian.PutUint32(payload[4:8], 125)
	binary.LittleEndian.PutUint32(payload[8:12], 57)
	binary.LittleEndian.PutUint32(payload[12:16], 1)

	if err := client.handleCommonEvent(8, 3, payload, 0, 0, 320, 200); err != nil {
		t.Fatalf("handle keyboard key: %v", err)
	}
	if len(client.events) != 2 {
		t.Fatalf("events = %d, want key + text", len(client.events))
	}
	event := client.events[0]
	if event.Kind != 6 || event.Key != 57 || event.Width != 320 || event.Height != 200 ||
		event.TimestampMS != 125 {
		t.Fatalf("event = %#v, want key down 57", event)
	}
	textEvent := client.events[1]
	if textEvent.Kind != 8 || textEvent.TextLen != 1 || string(client.textQueue) != " " {
		t.Fatalf("text event = %#v textQueue=%q, want one space", textEvent, client.textQueue)
	}
}

func TestWaylandPollEventTextReadsQueuedASCIIBytes(t *testing.T) {
	backend := &WaylandBackend{
		nextHandle: 10,
		windows: map[uint32]*waylandWindow{
			9: {client: &waylandClient{textQueue: []byte("az")}, width: 320, height: 200},
		},
	}

	text, err := backend.PollEventText(9)
	if err != nil {
		t.Fatalf("PollEventText: %v", err)
	}
	if string(text) != "az" {
		t.Fatalf("text = %q, want az", text)
	}
}

func TestWaylandASCIITextForKeyUsesUSFallback(t *testing.T) {
	for _, tc := range []struct {
		key  uint32
		want string
	}{
		{30, "a"},
		{31, "s"},
		{16, "q"},
		{2, "1"},
		{11, "0"},
		{57, " "},
		{28, "\n"},
	} {
		got, ok := waylandASCIITextForKey(tc.key)
		if !ok || string(got) != tc.want {
			t.Fatalf("waylandASCIITextForKey(%d) = %q, %v; want %q", tc.key, got, ok, tc.want)
		}
	}
	if got, ok := waylandASCIITextForKey(1000); ok || got != nil {
		t.Fatalf("waylandASCIITextForKey(1000) = %q, %v; want nil false", got, ok)
	}
}

func TestWaylandRequestRedrawDoesNotQueueSyntheticEvent(t *testing.T) {
	backend := &WaylandBackend{
		nextHandle: 10,
		windows: map[uint32]*waylandWindow{
			9: {width: 320, height: 200},
		},
	}

	if err := backend.RequestRedraw(9); err != nil {
		t.Fatalf("RequestRedraw: %v", err)
	}
	if got := len(backend.windows[9].events); got != 0 {
		t.Fatalf("queued events = %d, want no synthetic redraw event", got)
	}
}
