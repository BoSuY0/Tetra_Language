//go:build linux

package surfacehost

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

type WaylandBackend struct {
	mu         sync.Mutex
	nextHandle uint32
	windows    map[uint32]*waylandWindow
	clipboard  []byte
}

type waylandWindow struct {
	client       *waylandClient
	compositorID uint32
	shmID        uint32
	wmBaseID     uint32
	seatID       uint32
	surfaceID    uint32
	xdgSurfaceID uint32
	width        int32
	height       int32
	events       []Event
	shmFiles     []*os.File
	shmPaths     []string
}

type waylandGlobal struct {
	Name    uint32
	Version uint32
}

type waylandClient struct {
	conn   *net.UnixConn
	reader *bufio.Reader
	nextID uint32

	registryID    uint32
	compositor    waylandGlobal
	shm           waylandGlobal
	xdgWMBase     waylandGlobal
	seat          waylandGlobal
	seatID        uint32
	seatCaps      uint32
	pointerID     uint32
	pointerX      int32
	pointerY      int32
	keyboardID    uint32
	xdgToplevelID uint32
	configured    bool
	configSerial  uint32
	closed        bool
	events        []Event
	textQueue     []byte
}

func NewWaylandBackend() (Backend, error) {
	return &WaylandBackend{nextHandle: 1, windows: map[uint32]*waylandWindow{}}, nil
}

func (b *WaylandBackend) Open(title string, width int32, height int32) (uint32, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if width <= 0 || height <= 0 {
		return 0, fmt.Errorf("surface dimensions must be positive")
	}
	window, err := openWaylandWindow(title, width, height)
	if err != nil {
		return 0, err
	}
	handle := b.nextHandle
	b.nextHandle++
	b.windows[handle] = window
	return handle, nil
}

func (b *WaylandBackend) Close(handle uint32) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	window, err := b.window(handle)
	if err != nil {
		return err
	}
	window.close()
	delete(b.windows, handle)
	return nil
}

func (b *WaylandBackend) BeginFrame(handle uint32) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	_, err := b.window(handle)
	return err
}

func (b *WaylandBackend) PresentRGBA(
	handle uint32,
	width int32,
	height int32,
	stride int32,
	rgba []byte,
) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	window, err := b.window(handle)
	if err != nil {
		return err
	}
	if width <= 0 || height <= 0 || stride < width*4 {
		return fmt.Errorf("invalid RGBA frame dimensions %dx%d stride %d", width, height, stride)
	}
	if len(rgba) < int(stride*height) {
		return fmt.Errorf("RGBA payload has %d bytes, want at least %d", len(rgba), stride*height)
	}
	return window.present(width, height, stride, rgba[:stride*height])
}

func (b *WaylandBackend) PollEvent(handle uint32) (Event, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	window, err := b.window(handle)
	if err != nil {
		return Event{}, err
	}
	if err := window.pump(1 * time.Millisecond); err != nil {
		return Event{}, err
	}
	if len(window.events) == 0 {
		return Event{}, nil
	}
	event := window.events[0]
	window.events = window.events[1:]
	return event, nil
}

func (b *WaylandBackend) PollEventText(handle uint32) ([]byte, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	window, err := b.window(handle)
	if err != nil {
		return nil, err
	}
	if window.client == nil {
		return nil, nil
	}
	if len(window.client.textQueue) == 0 && window.client.conn != nil {
		if err := window.pump(1 * time.Millisecond); err != nil {
			return nil, err
		}
	}
	return append([]byte(nil), window.client.textQueue...), nil
}

func (b *WaylandBackend) ClipboardWriteText(handle uint32, text []byte) (int32, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, err := b.window(handle); err != nil {
		return 0, err
	}
	b.clipboard = append(b.clipboard[:0], text...)
	return int32(len(text)), nil
}

func (b *WaylandBackend) ClipboardReadText(handle uint32) ([]byte, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, err := b.window(handle); err != nil {
		return nil, err
	}
	return append([]byte(nil), b.clipboard...), nil
}

func (b *WaylandBackend) PollComposition(handle uint32) ([4]int32, error) {
	if _, err := b.window(handle); err != nil {
		return [4]int32{}, err
	}
	return [4]int32{}, nil
}

func (b *WaylandBackend) NowMS() int32 {
	return int32(time.Now().UnixMilli() & 0x7fffffff)
}

func (b *WaylandBackend) RequestRedraw(handle uint32) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	_, err := b.window(handle)
	if err == nil {
		time.Sleep(16 * time.Millisecond)
	}
	return err
}

func (b *WaylandBackend) window(handle uint32) (*waylandWindow, error) {
	window, ok := b.windows[handle]
	if !ok {
		return nil, fmt.Errorf("unknown Surface handle %d", handle)
	}
	return window, nil
}

func openWaylandWindow(title string, width int32, height int32) (*waylandWindow, error) {
	socketPath, err := waylandSocketPath()
	if err != nil {
		return nil, err
	}
	addr := net.UnixAddr{Name: socketPath, Net: "unix"}
	conn, err := net.DialUnix("unix", nil, &addr)
	if err != nil {
		return nil, fmt.Errorf("connect Wayland compositor at %s: %w", socketPath, err)
	}
	client := &waylandClient{conn: conn, reader: bufio.NewReader(conn), nextID: 2}
	if err := client.roundtripRegistry(); err != nil {
		conn.Close()
		return nil, err
	}
	if client.compositor.Name == 0 || client.shm.Name == 0 || client.xdgWMBase.Name == 0 {
		conn.Close()
		return nil, fmt.Errorf(
			"Wayland compositor missing required globals: wl_compositor=%d wl_shm=%d xdg_wm_base=%d",
			client.compositor.Name,
			client.shm.Name,
			client.xdgWMBase.Name,
		)
	}

	compositorID := client.newID()
	shmID := client.newID()
	wmBaseID := client.newID()
	seatID := uint32(0)
	if err := client.bind(
		client.compositor.Name,
		"wl_compositor",
		minVersion(client.compositor.Version, 4),
		compositorID,
	); err != nil {
		conn.Close()
		return nil, err
	}
	if err := client.bind(
		client.shm.Name,
		"wl_shm",
		minVersion(client.shm.Version, 1),
		shmID,
	); err != nil {
		conn.Close()
		return nil, err
	}
	if err := client.bind(
		client.xdgWMBase.Name,
		"xdg_wm_base",
		minVersion(client.xdgWMBase.Version, 1),
		wmBaseID,
	); err != nil {
		conn.Close()
		return nil, err
	}
	if client.seat.Name != 0 {
		seatID = client.newID()
		client.seatID = seatID
		if err := client.bind(
			client.seat.Name,
			"wl_seat",
			minVersion(client.seat.Version, 1),
			seatID,
		); err != nil {
			conn.Close()
			return nil, err
		}
		if err := client.waitForSeatCapabilities(seatID, wmBaseID, 0, 500*time.Millisecond); err != nil {
			conn.Close()
			return nil, err
		}
		if err := client.ensureSeatDevices(); err != nil {
			conn.Close()
			return nil, err
		}
	}

	surfaceID := client.newID()
	xdgSurfaceID := client.newID()
	toplevelID := client.newID()
	client.xdgToplevelID = toplevelID
	if err := client.send(compositorID, 0, u32(surfaceID), nil); err != nil {
		conn.Close()
		return nil, fmt.Errorf("create wl_surface: %w", err)
	}
	if err := client.send(wmBaseID, 2, concat(u32(xdgSurfaceID), u32(surfaceID)), nil); err != nil {
		conn.Close()
		return nil, fmt.Errorf("create xdg_surface: %w", err)
	}
	if err := client.send(xdgSurfaceID, 1, u32(toplevelID), nil); err != nil {
		conn.Close()
		return nil, fmt.Errorf("create xdg_toplevel: %w", err)
	}
	if err := client.send(toplevelID, 2, wlString(title), nil); err != nil {
		conn.Close()
		return nil, fmt.Errorf("set xdg_toplevel title: %w", err)
	}
	if err := client.send(toplevelID, 3, wlString("tetra-surface-host-wayland"), nil); err != nil {
		conn.Close()
		return nil, fmt.Errorf("set xdg_toplevel app id: %w", err)
	}
	if err := client.send(surfaceID, 6, nil, nil); err != nil {
		conn.Close()
		return nil, fmt.Errorf("initial wl_surface commit: %w", err)
	}
	if err := client.waitForConfigure(xdgSurfaceID, wmBaseID, 2*time.Second); err != nil {
		conn.Close()
		return nil, err
	}
	if err := client.send(xdgSurfaceID, 4, u32(client.configSerial), nil); err != nil {
		conn.Close()
		return nil, fmt.Errorf("ack xdg_surface configure: %w", err)
	}
	return &waylandWindow{
		client:       client,
		compositorID: compositorID,
		shmID:        shmID,
		wmBaseID:     wmBaseID,
		seatID:       seatID,
		surfaceID:    surfaceID,
		xdgSurfaceID: xdgSurfaceID,
		width:        width,
		height:       height,
	}, nil
}

func (w *waylandWindow) present(width int32, height int32, stride int32, rgba []byte) error {
	shmFile, size, err := writeWaylandSHMFrame(width, height, stride, rgba)
	if err != nil {
		return err
	}
	shmPath := shmFile.Name()
	poolID := w.client.newID()
	bufferID := w.client.newID()
	if err := w.client.send(
		w.shmID,
		0,
		concat(u32(poolID), i32(int32(size))),
		[]int{int(shmFile.Fd())},
	); err != nil {
		_ = shmFile.Close()
		_ = os.Remove(shmPath)
		return fmt.Errorf("create wl_shm_pool: %w", err)
	}
	_ = shmFile.Close()
	_ = os.Remove(shmPath)
	if err := w.client.send(
		poolID,
		0,
		concat(u32(bufferID), i32(0), i32(width), i32(height), i32(stride), u32(0)),
		nil,
	); err != nil {
		return fmt.Errorf("create wl_buffer: %w", err)
	}
	if err := w.client.send(w.surfaceID, 1, concat(u32(bufferID), i32(0), i32(0)), nil); err != nil {
		return fmt.Errorf("attach wl_buffer: %w", err)
	}
	if err := w.client.send(
		w.surfaceID,
		2,
		concat(i32(0), i32(0), i32(width), i32(height)),
		nil,
	); err != nil {
		return fmt.Errorf("damage wl_surface: %w", err)
	}
	if err := w.client.send(w.surfaceID, 6, nil, nil); err != nil {
		return fmt.Errorf("present wl_surface commit: %w", err)
	}
	w.width = width
	w.height = height
	return nil
}

func (w *waylandWindow) pump(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if w.client.closed {
			w.events = append(
				w.events,
				Event{
					Kind:        1,
					Width:       w.width,
					Height:      w.height,
					TimestampMS: int32(time.Now().UnixMilli() & 0x7fffffff),
				},
			)
			w.client.closed = false
			return nil
		}
		if len(w.client.events) > 0 {
			w.events = append(w.events, w.client.events...)
			w.client.events = nil
			return nil
		}
		_ = w.client.conn.SetReadDeadline(time.Now().Add(1 * time.Millisecond))
		if err := w.client.readOne(w.wmBaseID, w.xdgSurfaceID, w.width, w.height); err != nil {
			if isTimeout(err) {
				return nil
			}
			return err
		}
	}
	return nil
}

func (w *waylandWindow) close() {
	if w.client != nil && w.client.conn != nil {
		_ = w.client.conn.Close()
	}
	for _, file := range w.shmFiles {
		_ = file.Close()
	}
	for _, path := range w.shmPaths {
		_ = os.Remove(path)
	}
}

func waylandSocketPath() (string, error) {
	display := os.Getenv("WAYLAND_DISPLAY")
	if strings.TrimSpace(display) == "" {
		return "", fmt.Errorf("WAYLAND_DISPLAY is not set; cannot start Wayland Surface host")
	}
	if filepath.IsAbs(display) {
		return display, nil
	}
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if strings.TrimSpace(runtimeDir) == "" {
		return "", fmt.Errorf(
			"XDG_RUNTIME_DIR is not set; cannot resolve Wayland socket %s",
			display,
		)
	}
	return filepath.Join(runtimeDir, display), nil
}

func (c *waylandClient) newID() uint32 {
	id := c.nextID
	c.nextID++
	return id
}

func (c *waylandClient) roundtripRegistry() error {
	c.registryID = c.newID()
	callbackID := c.newID()
	if err := c.send(1, 1, u32(c.registryID), nil); err != nil {
		return fmt.Errorf("wl_display.get_registry: %w", err)
	}
	if err := c.send(1, 0, u32(callbackID), nil); err != nil {
		return fmt.Errorf("wl_display.sync: %w", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		_ = c.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		done, err := c.readRegistryRoundtripEvent(callbackID)
		if err != nil {
			if isTimeout(err) {
				continue
			}
			return err
		}
		if done {
			return nil
		}
	}
	return fmt.Errorf("timed out waiting for Wayland registry globals")
}

func (c *waylandClient) bind(name uint32, iface string, version uint32, id uint32) error {
	payload := concat(u32(name), wlString(iface), u32(version), u32(id))
	return c.send(c.registryID, 0, payload, nil)
}

func (c *waylandClient) waitForConfigure(
	xdgSurfaceID uint32,
	wmBaseID uint32,
	timeout time.Duration,
) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		_ = c.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		if err := c.readOne(wmBaseID, xdgSurfaceID, 0, 0); err != nil {
			if isTimeout(err) {
				continue
			}
			return err
		}
		if c.configured {
			return nil
		}
	}
	return fmt.Errorf("timed out waiting for xdg_surface configure from real compositor")
}

func (c *waylandClient) readRegistryRoundtripEvent(callbackID uint32) (bool, error) {
	object, opcode, payload, err := c.readMessage()
	if err != nil {
		return false, err
	}
	if err := c.handleCommonEvent(object, opcode, payload, 0, 0, 0, 0); err != nil {
		return false, err
	}
	if object == c.registryID && opcode == 0 {
		name, iface, version, err := parseRegistryGlobal(payload)
		if err != nil {
			return false, err
		}
		switch iface {
		case "wl_compositor":
			c.compositor = waylandGlobal{Name: name, Version: version}
		case "wl_shm":
			c.shm = waylandGlobal{Name: name, Version: version}
		case "xdg_wm_base":
			c.xdgWMBase = waylandGlobal{Name: name, Version: version}
		case "wl_seat":
			c.seat = waylandGlobal{Name: name, Version: version}
		}
	}
	return object == callbackID && opcode == 0, nil
}

func (c *waylandClient) waitForSeatCapabilities(
	seatID uint32,
	wmBaseID uint32,
	xdgSurfaceID uint32,
	timeout time.Duration,
) error {
	callbackID := c.newID()
	if err := c.send(1, 0, u32(callbackID), nil); err != nil {
		return fmt.Errorf("wl_display.sync after wl_seat bind: %w", err)
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		_ = c.conn.SetReadDeadline(time.Now().Add(25 * time.Millisecond))
		object, opcode, payload, err := c.readMessage()
		if err != nil {
			if isTimeout(err) {
				continue
			}
			return err
		}
		if err := c.handleCommonEvent(object, opcode, payload, wmBaseID, xdgSurfaceID, 0, 0); err != nil {
			return err
		}
		if c.seatCaps != 0 && object == callbackID && opcode == 0 {
			return nil
		}
		if object == callbackID && opcode == 0 {
			return nil
		}
	}
	return nil
}

func (c *waylandClient) readOne(
	wmBaseID uint32,
	xdgSurfaceID uint32,
	width int32,
	height int32,
) error {
	object, opcode, payload, err := c.readMessage()
	if err != nil {
		return err
	}
	return c.handleCommonEvent(object, opcode, payload, wmBaseID, xdgSurfaceID, width, height)
}

func (c *waylandClient) handleCommonEvent(
	object uint32,
	opcode uint16,
	payload []byte,
	wmBaseID uint32,
	xdgSurfaceID uint32,
	width int32,
	height int32,
) error {
	if object == 1 && opcode == 0 {
		_, code, message, err := parseDisplayError(payload)
		if err != nil {
			return err
		}
		return fmt.Errorf("Wayland display error %d: %s", code, message)
	}
	if wmBaseID != 0 && object == wmBaseID && opcode == 0 {
		if len(payload) < 4 {
			return fmt.Errorf("xdg_wm_base ping missing serial")
		}
		serial := binary.LittleEndian.Uint32(payload[:4])
		return c.send(wmBaseID, 3, u32(serial), nil)
	}
	if xdgSurfaceID != 0 && object == xdgSurfaceID && opcode == 0 {
		if len(payload) < 4 {
			return fmt.Errorf("xdg_surface configure missing serial")
		}
		c.configSerial = binary.LittleEndian.Uint32(payload[:4])
		c.configured = true
	}
	if c.xdgToplevelID != 0 && object == c.xdgToplevelID && opcode == 1 {
		c.closed = true
	}
	if c.seatID != 0 && object == c.seatID && opcode == 0 {
		if len(payload) < 4 {
			return fmt.Errorf("wl_seat capabilities missing mask")
		}
		c.seatCaps = binary.LittleEndian.Uint32(payload[:4])
		if err := c.ensureSeatDevices(); err != nil {
			return err
		}
	}
	if c.pointerID != 0 && object == c.pointerID {
		return c.handlePointerEvent(opcode, payload, width, height)
	}
	if c.keyboardID != 0 && object == c.keyboardID {
		return c.handleKeyboardEvent(opcode, payload, width, height)
	}
	return nil
}

func (c *waylandClient) ensureSeatDevices() error {
	if c.seatID == 0 {
		return nil
	}
	if c.seatCaps&1 != 0 && c.pointerID == 0 {
		c.pointerID = c.newID()
		if err := c.send(c.seatID, 0, u32(c.pointerID), nil); err != nil {
			return fmt.Errorf("wl_seat.get_pointer: %w", err)
		}
	}
	if c.seatCaps&2 != 0 && c.keyboardID == 0 {
		c.keyboardID = c.newID()
		if err := c.send(c.seatID, 1, u32(c.keyboardID), nil); err != nil {
			return fmt.Errorf("wl_seat.get_keyboard: %w", err)
		}
	}
	return nil
}

func (c *waylandClient) handlePointerEvent(
	opcode uint16,
	payload []byte,
	width int32,
	height int32,
) error {
	switch opcode {
	case 0: // enter: serial, surface, surface_x, surface_y
		if len(payload) < 16 {
			return fmt.Errorf("wl_pointer enter payload too short")
		}
		c.pointerX = wlFixedToInt32(binary.LittleEndian.Uint32(payload[8:12]))
		c.pointerY = wlFixedToInt32(binary.LittleEndian.Uint32(payload[12:16]))
		c.events = append(
			c.events,
			Event{
				Kind:        3,
				X:           c.pointerX,
				Y:           c.pointerY,
				Width:       width,
				Height:      height,
				TimestampMS: int32(time.Now().UnixMilli() & 0x7fffffff),
			},
		)
	case 2: // motion: time, surface_x, surface_y
		if len(payload) < 12 {
			return fmt.Errorf("wl_pointer motion payload too short")
		}
		c.pointerX = wlFixedToInt32(binary.LittleEndian.Uint32(payload[4:8]))
		c.pointerY = wlFixedToInt32(binary.LittleEndian.Uint32(payload[8:12]))
		c.events = append(
			c.events,
			Event{
				Kind:        3,
				X:           c.pointerX,
				Y:           c.pointerY,
				Width:       width,
				Height:      height,
				TimestampMS: int32(binary.LittleEndian.Uint32(payload[0:4])),
			},
		)
	case 3: // button: serial, time, button, state
		if len(payload) < 16 {
			return fmt.Errorf("wl_pointer button payload too short")
		}
		state := binary.LittleEndian.Uint32(payload[12:16])
		kind := int32(5)
		if state == 1 {
			kind = 4
		}
		c.events = append(c.events, Event{
			Kind:        kind,
			X:           c.pointerX,
			Y:           c.pointerY,
			Button:      normalizeWaylandButton(binary.LittleEndian.Uint32(payload[8:12])),
			Width:       width,
			Height:      height,
			TimestampMS: int32(binary.LittleEndian.Uint32(payload[4:8])),
		})
	}
	return nil
}

func (c *waylandClient) handleKeyboardEvent(
	opcode uint16,
	payload []byte,
	width int32,
	height int32,
) error {
	switch opcode {
	case 0: // keymap: format, fd(out-of-band), size; key text/IME is out of v1 scope.
		return nil
	case 3: // key: serial, time, key, state
		if len(payload) < 16 {
			return fmt.Errorf("wl_keyboard key payload too short")
		}
		state := binary.LittleEndian.Uint32(payload[12:16])
		key := binary.LittleEndian.Uint32(payload[8:12])
		kind := int32(7)
		if state == 1 {
			kind = 6
		}
		c.events = append(c.events, Event{
			Kind:        kind,
			Key:         int32(key),
			Width:       width,
			Height:      height,
			TimestampMS: int32(binary.LittleEndian.Uint32(payload[4:8])),
		})
		if state == 1 {
			if text, ok := waylandASCIITextForKey(key); ok {
				c.textQueue = append(c.textQueue, text...)
				c.events = append(c.events, Event{
					Kind:        8,
					Width:       width,
					Height:      height,
					TimestampMS: int32(binary.LittleEndian.Uint32(payload[4:8])),
					TextLen:     int32(len(text)),
				})
			}
		}
	}
	return nil
}

func waylandASCIITextForKey(key uint32) ([]byte, bool) {
	switch {
	case key >= 2 && key <= 10:
		return []byte{byte('1' + key - 2)}, true
	case key == 11:
		return []byte{'0'}, true
	case key >= 16 && key <= 25:
		return []byte("qwertyuiop")[key-16 : key-15], true
	case key >= 30 && key <= 38:
		return []byte("asdfghjkl")[key-30 : key-29], true
	case key >= 44 && key <= 50:
		return []byte("zxcvbnm")[key-44 : key-43], true
	}
	switch key {
	case 12:
		return []byte{'-'}, true
	case 13:
		return []byte{'='}, true
	case 26:
		return []byte{'['}, true
	case 27:
		return []byte{']'}, true
	case 28:
		return []byte{'\n'}, true
	case 39:
		return []byte{';'}, true
	case 40:
		return []byte{'\''}, true
	case 41:
		return []byte{'`'}, true
	case 43:
		return []byte{'\\'}, true
	case 51:
		return []byte{','}, true
	case 52:
		return []byte{'.'}, true
	case 53:
		return []byte{'/'}, true
	case 57:
		return []byte{' '}, true
	}
	return nil, false
}

func (c *waylandClient) readMessage() (uint32, uint16, []byte, error) {
	header := make([]byte, 8)
	if _, err := io.ReadFull(c.reader, header); err != nil {
		return 0, 0, nil, err
	}
	object := binary.LittleEndian.Uint32(header[:4])
	sizeOpcode := binary.LittleEndian.Uint32(header[4:8])
	opcode := uint16(sizeOpcode & 0xffff)
	size := int(sizeOpcode >> 16)
	if size < 8 {
		return 0, 0, nil, fmt.Errorf("invalid Wayland message size %d", size)
	}
	payload := make([]byte, size-8)
	if _, err := io.ReadFull(c.reader, payload); err != nil {
		return 0, 0, nil, err
	}
	return object, opcode, payload, nil
}

func (c *waylandClient) send(object uint32, opcode uint16, payload []byte, fds []int) error {
	size := 8 + len(payload)
	header := make([]byte, 8)
	binary.LittleEndian.PutUint32(header[:4], object)
	binary.LittleEndian.PutUint32(header[4:8], uint32(size)<<16|uint32(opcode))
	msg := append(header, payload...)
	var oob []byte
	if len(fds) > 0 {
		oob = syscall.UnixRights(fds...)
	}
	n, _, err := c.conn.WriteMsgUnix(msg, oob, nil)
	if err != nil {
		return err
	}
	if n != len(msg) {
		return fmt.Errorf("short Wayland write: %d of %d bytes", n, len(msg))
	}
	return nil
}

func writeWaylandSHMFrame(
	width int32,
	height int32,
	stride int32,
	rgba []byte,
) (*os.File, int, error) {
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = os.TempDir()
	}
	file, err := os.CreateTemp(runtimeDir, "tetra-surface-shm-*")
	if err != nil {
		return nil, 0, fmt.Errorf("create Wayland shm file: %w", err)
	}
	cleanup := func() {
		name := file.Name()
		_ = file.Close()
		_ = os.Remove(name)
	}
	size := int(stride * height)
	if err := file.Truncate(int64(size)); err != nil {
		cleanup()
		return nil, 0, fmt.Errorf("resize Wayland shm file: %w", err)
	}
	argb := rgbaToWaylandARGB(rgba)
	if _, err := file.WriteAt(argb, 0); err != nil {
		cleanup()
		return nil, 0, fmt.Errorf("write Wayland shm pixels: %w", err)
	}
	return file, size, nil
}

func rgbaToWaylandARGB(rgba []byte) []byte {
	out := make([]byte, len(rgba))
	for i := 0; i+3 < len(rgba); i += 4 {
		r := rgba[i]
		g := rgba[i+1]
		b := rgba[i+2]
		a := rgba[i+3]
		out[i] = b
		out[i+1] = g
		out[i+2] = r
		out[i+3] = a
	}
	return out
}

func parseRegistryGlobal(payload []byte) (uint32, string, uint32, error) {
	if len(payload) < 12 {
		return 0, "", 0, fmt.Errorf("registry global payload too short")
	}
	off := 0
	name := binary.LittleEndian.Uint32(payload[off : off+4])
	off += 4
	iface, next, err := parseWLString(payload, off)
	if err != nil {
		return 0, "", 0, err
	}
	off = next
	if len(payload[off:]) < 4 {
		return 0, "", 0, fmt.Errorf("registry global missing version")
	}
	version := binary.LittleEndian.Uint32(payload[off : off+4])
	return name, iface, version, nil
}

func parseDisplayError(payload []byte) (uint32, uint32, string, error) {
	if len(payload) < 12 {
		return 0, 0, "", fmt.Errorf("display error payload too short")
	}
	object := binary.LittleEndian.Uint32(payload[:4])
	code := binary.LittleEndian.Uint32(payload[4:8])
	message, _, err := parseWLString(payload, 8)
	return object, code, message, err
}

func parseWLString(payload []byte, off int) (string, int, error) {
	if len(payload[off:]) < 4 {
		return "", off, fmt.Errorf("Wayland string missing length")
	}
	length := int(binary.LittleEndian.Uint32(payload[off : off+4]))
	off += 4
	if length <= 0 || len(payload[off:]) < length {
		return "", off, fmt.Errorf("Wayland string length %d exceeds payload", length)
	}
	raw := payload[off : off+length]
	off += paddedLength(length)
	return strings.TrimRight(string(raw), "\x00"), off, nil
}

func wlString(value string) []byte {
	raw := append([]byte(value), 0)
	buf := bytes.NewBuffer(u32(uint32(len(raw))))
	buf.Write(raw)
	for buf.Len()%4 != 0 {
		buf.WriteByte(0)
	}
	return buf.Bytes()
}

func concat(parts ...[]byte) []byte {
	var out []byte
	for _, part := range parts {
		out = append(out, part...)
	}
	return out
}

func u32(value uint32) []byte {
	var out [4]byte
	binary.LittleEndian.PutUint32(out[:], value)
	return out[:]
}

func i32(value int32) []byte {
	return u32(uint32(value))
}

func wlFixedToInt32(value uint32) int32 {
	return int32(value) >> 8
}

func normalizeWaylandButton(button uint32) int32 {
	if button >= 272 && button <= 279 {
		return int32(button - 271)
	}
	return int32(button)
}

func paddedLength(length int) int {
	if rem := length % 4; rem != 0 {
		return length + (4 - rem)
	}
	return length
}

func minVersion(a uint32, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}

func isTimeout(err error) bool {
	if err == nil {
		return false
	}
	if ne, ok := err.(net.Error); ok && ne.Timeout() {
		return true
	}
	return os.IsTimeout(err)
}
