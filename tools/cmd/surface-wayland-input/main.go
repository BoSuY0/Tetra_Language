package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

type wlGlobal struct {
	Name    uint32
	Iface   string
	Version uint32
}

type wlClient struct {
	conn       *net.UnixConn
	reader     *bufio.Reader
	nextID     uint32
	registryID uint32
	globals    map[string]wlGlobal
}

type runOptions struct {
	ListGlobals     bool
	Pointer         bool
	PointerSeat     bool
	PointerOutput   bool
	PointerRelative bool
	Keyboard        bool
	X               uint32
	Y               uint32
	XExtent         uint32
	YExtent         uint32
	Button          uint32
	Key             uint32
	Delay           time.Duration
	Hold            time.Duration
}

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	fs := flag.NewFlagSet("surface-wayland-input", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	opt := runOptions{}
	delayMS := fs.Int("delay-ms", 80, "delay between press and release events")
	holdMS := fs.Int("hold-ms", 0, "time to keep virtual input devices alive after sending events")
	x := fs.Uint("x", 960, "absolute pointer x")
	y := fs.Uint("y", 540, "absolute pointer y")
	xExtent := fs.Uint("x-extent", 1920, "absolute pointer x extent")
	yExtent := fs.Uint("y-extent", 1080, "absolute pointer y extent")
	button := fs.Uint("button", 0x110, "evdev pointer button code")
	key := fs.Uint("key", 57, "evdev key code")
	fs.BoolVar(&opt.ListGlobals, "list-globals", false, "list compositor globals and exit")
	fs.BoolVar(&opt.Pointer, "pointer", true, "send a virtual pointer click")
	fs.BoolVar(&opt.PointerSeat, "pointer-seat", true, "associate the virtual pointer with wl_seat")
	fs.BoolVar(
		&opt.PointerOutput,
		"pointer-output",
		true,
		"associate the virtual pointer with wl_output when supported",
	)
	fs.BoolVar(
		&opt.PointerRelative,
		"pointer-relative",
		false,
		"send relative virtual-pointer motion instead of motion_absolute",
	)
	fs.BoolVar(&opt.Keyboard, "keyboard", true, "send a virtual keyboard key press")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	opt.Delay = time.Duration(*delayMS) * time.Millisecond
	opt.Hold = time.Duration(*holdMS) * time.Millisecond
	opt.X = uint32(*x)
	opt.Y = uint32(*y)
	opt.XExtent = uint32(*xExtent)
	opt.YExtent = uint32(*yExtent)
	opt.Button = uint32(*button)
	opt.Key = uint32(*key)
	if err := runWaylandInput(opt); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func runWaylandInput(opt runOptions) error {
	client, err := connectWayland()
	if err != nil {
		return err
	}
	defer client.conn.Close()
	if err := client.roundtripRegistry(); err != nil {
		return err
	}
	if opt.ListGlobals {
		for _, global := range sortedGlobals(client.globals) {
			fmt.Printf("%s v%d name=%d\n", global.Iface, global.Version, global.Name)
		}
		return nil
	}
	seat, ok := client.globals["wl_seat"]
	if !ok {
		return fmt.Errorf("Wayland compositor does not expose wl_seat")
	}
	seatID := client.newID()
	if err := client.bind(seat, minVersion(seat.Version, 7), seatID); err != nil {
		return err
	}
	if err := client.roundtrip(); err != nil {
		return err
	}
	outputID := uint32(0)
	if opt.Pointer && opt.PointerOutput {
		if output, ok := client.globals["wl_output"]; ok {
			outputID = client.newID()
			if err := client.bind(output, minVersion(output.Version, 1), outputID); err != nil {
				return err
			}
		}
	}
	if opt.Pointer {
		pointerSeatID := seatID
		if !opt.PointerSeat {
			pointerSeatID = 0
		}
		if err := client.sendPointerClick(pointerSeatID, outputID, opt); err != nil {
			return err
		}
	}
	if opt.Keyboard {
		if err := client.sendKeyboardKey(seatID, opt); err != nil {
			return err
		}
	}
	if err := client.roundtrip(); err != nil {
		return err
	}
	if opt.Hold > 0 {
		time.Sleep(opt.Hold)
	}
	return nil
}

func connectWayland() (*wlClient, error) {
	socketPath, err := waylandSocketPath()
	if err != nil {
		return nil, err
	}
	addr := net.UnixAddr{Name: socketPath, Net: "unix"}
	conn, err := net.DialUnix("unix", nil, &addr)
	if err != nil {
		return nil, fmt.Errorf("connect Wayland compositor at %s: %w", socketPath, err)
	}
	return &wlClient{
		conn:    conn,
		reader:  bufio.NewReader(conn),
		nextID:  2,
		globals: map[string]wlGlobal{},
	}, nil
}

func waylandSocketPath() (string, error) {
	display := strings.TrimSpace(os.Getenv("WAYLAND_DISPLAY"))
	if display == "" {
		return "", fmt.Errorf("WAYLAND_DISPLAY is not set")
	}
	if filepath.IsAbs(display) {
		return display, nil
	}
	runtimeDir := strings.TrimSpace(os.Getenv("XDG_RUNTIME_DIR"))
	if runtimeDir == "" {
		return "", fmt.Errorf("XDG_RUNTIME_DIR is not set")
	}
	return filepath.Join(runtimeDir, display), nil
}

func (c *wlClient) roundtripRegistry() error {
	c.registryID = c.newID()
	callbackID := c.newID()
	if err := c.send(1, 1, u32(c.registryID), nil); err != nil {
		return fmt.Errorf("wl_display.get_registry: %w", err)
	}
	if err := c.send(1, 0, u32(callbackID), nil); err != nil {
		return fmt.Errorf("wl_display.sync: %w", err)
	}
	return c.readUntilCallback(callbackID)
}

func (c *wlClient) roundtrip() error {
	callbackID := c.newID()
	if err := c.send(1, 0, u32(callbackID), nil); err != nil {
		return fmt.Errorf("wl_display.sync: %w", err)
	}
	return c.readUntilCallback(callbackID)
}

func (c *wlClient) readUntilCallback(callbackID uint32) error {
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		_ = c.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		object, opcode, payload, err := c.readMessage()
		if err != nil {
			if isTimeout(err) {
				continue
			}
			return err
		}
		if err := c.handleMessage(object, opcode, payload); err != nil {
			return err
		}
		if object == callbackID && opcode == 0 {
			return nil
		}
	}
	return fmt.Errorf("timed out waiting for Wayland sync callback %d", callbackID)
}

func (c *wlClient) handleMessage(object uint32, opcode uint16, payload []byte) error {
	if object == 1 && opcode == 0 {
		_, code, message, err := parseDisplayError(payload)
		if err != nil {
			return err
		}
		return fmt.Errorf("Wayland display error %d: %s", code, message)
	}
	if c.registryID != 0 && object == c.registryID && opcode == 0 {
		global, err := parseRegistryGlobal(payload)
		if err != nil {
			return err
		}
		c.globals[global.Iface] = global
	}
	return nil
}

func (c *wlClient) bind(global wlGlobal, version uint32, id uint32) error {
	payload := concat(u32(global.Name), wlString(global.Iface), u32(version), u32(id))
	return c.send(c.registryID, 0, payload, nil)
}

func (c *wlClient) sendPointerClick(seatID uint32, outputID uint32, opt runOptions) error {
	manager, ok := c.globals["zwlr_virtual_pointer_manager_v1"]
	if !ok {
		return fmt.Errorf("Wayland compositor does not expose zwlr_virtual_pointer_manager_v1")
	}
	managerID := c.newID()
	pointerID := c.newID()
	if err := c.bind(manager, minVersion(manager.Version, 2), managerID); err != nil {
		return err
	}
	if manager.Version >= 2 && outputID != 0 {
		if err := c.send(
			managerID,
			2,
			concat(u32(seatID), u32(outputID), u32(pointerID)),
			nil,
		); err != nil {
			return fmt.Errorf("create virtual pointer with output: %w", err)
		}
	} else if err := c.send(managerID, 0, concat(u32(seatID), u32(pointerID)), nil); err != nil {
		return fmt.Errorf("create virtual pointer: %w", err)
	}
	now := eventTime()
	if opt.PointerRelative {
		if err := c.sendRelativePointerMotion(
			pointerID,
			now,
			-int32(opt.XExtent)*2,
			-int32(opt.YExtent)*2,
		); err != nil {
			return err
		}
		if err := c.send(pointerID, 4, nil, nil); err != nil {
			return fmt.Errorf("virtual pointer frame after relative origin: %w", err)
		}
		if err := c.sendRelativePointerMotion(pointerID, now+1, int32(opt.X), int32(opt.Y)); err != nil {
			return err
		}
	} else if err := c.send(
		pointerID,
		1,
		concat(u32(now), u32(opt.X), u32(opt.Y), u32(opt.XExtent), u32(opt.YExtent)),
		nil,
	); err != nil {
		return fmt.Errorf("virtual pointer motion_absolute: %w", err)
	}
	if err := c.send(pointerID, 4, nil, nil); err != nil {
		return fmt.Errorf("virtual pointer frame after motion: %w", err)
	}
	if err := c.send(pointerID, 2, concat(u32(now+1), u32(opt.Button), u32(1)), nil); err != nil {
		return fmt.Errorf("virtual pointer button press: %w", err)
	}
	if err := c.send(pointerID, 4, nil, nil); err != nil {
		return fmt.Errorf("virtual pointer frame after press: %w", err)
	}
	time.Sleep(opt.Delay)
	if err := c.send(
		pointerID,
		2,
		concat(u32(eventTime()), u32(opt.Button), u32(0)),
		nil,
	); err != nil {
		return fmt.Errorf("virtual pointer button release: %w", err)
	}
	if err := c.send(pointerID, 4, nil, nil); err != nil {
		return fmt.Errorf("virtual pointer frame after release: %w", err)
	}
	return nil
}

func (c *wlClient) sendRelativePointerMotion(
	pointerID uint32,
	timeMS uint32,
	dx int32,
	dy int32,
) error {
	if err := c.send(pointerID, 0, concat(u32(timeMS), wlFixed(dx), wlFixed(dy)), nil); err != nil {
		return fmt.Errorf("virtual pointer relative motion: %w", err)
	}
	return nil
}

func (c *wlClient) sendKeyboardKey(seatID uint32, opt runOptions) error {
	manager, ok := c.globals["zwp_virtual_keyboard_manager_v1"]
	if !ok {
		return fmt.Errorf("Wayland compositor does not expose zwp_virtual_keyboard_manager_v1")
	}
	managerID := c.newID()
	keyboardID := c.newID()
	if err := c.bind(manager, minVersion(manager.Version, 1), managerID); err != nil {
		return err
	}
	if err := c.send(managerID, 0, concat(u32(seatID), u32(keyboardID)), nil); err != nil {
		return fmt.Errorf("create virtual keyboard: %w", err)
	}
	keymapFile, keymapSize, err := createXKBKeymapFile()
	if err != nil {
		return err
	}
	defer func() {
		name := keymapFile.Name()
		_ = keymapFile.Close()
		_ = os.Remove(name)
	}()
	if err := c.send(
		keyboardID,
		0,
		concat(u32(1), u32(uint32(keymapSize))),
		[]int{int(keymapFile.Fd())},
	); err != nil {
		return fmt.Errorf("virtual keyboard keymap: %w", err)
	}
	if err := c.roundtrip(); err != nil {
		return err
	}
	if err := c.send(keyboardID, 2, concat(u32(0), u32(0), u32(0), u32(0)), nil); err != nil {
		return fmt.Errorf("virtual keyboard modifiers: %w", err)
	}
	now := eventTime()
	if err := c.send(keyboardID, 1, concat(u32(now), u32(opt.Key), u32(1)), nil); err != nil {
		return fmt.Errorf("virtual keyboard key press: %w", err)
	}
	time.Sleep(opt.Delay)
	if err := c.send(keyboardID, 1, concat(u32(eventTime()), u32(opt.Key), u32(0)), nil); err != nil {
		return fmt.Errorf("virtual keyboard key release: %w", err)
	}
	return nil
}

func createXKBKeymapFile() (*os.File, int, error) {
	raw, err := exec.Command("xkbcli", "compile-keymap", "--layout", "us").Output()
	if err != nil {
		return nil, 0, fmt.Errorf("compile XKB keymap with xkbcli: %w", err)
	}
	if len(raw) == 0 || raw[len(raw)-1] != 0 {
		raw = append(raw, 0)
	}
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = os.TempDir()
	}
	file, err := os.CreateTemp(runtimeDir, "tetra-surface-keymap-*")
	if err != nil {
		return nil, 0, fmt.Errorf("create keymap temp file: %w", err)
	}
	cleanup := func() {
		name := file.Name()
		_ = file.Close()
		_ = os.Remove(name)
	}
	if _, err := file.Write(raw); err != nil {
		cleanup()
		return nil, 0, fmt.Errorf("write keymap temp file: %w", err)
	}
	if err := file.Truncate(int64(len(raw))); err != nil {
		cleanup()
		return nil, 0, fmt.Errorf("truncate keymap temp file: %w", err)
	}
	if _, err := file.Seek(0, 0); err != nil {
		cleanup()
		return nil, 0, fmt.Errorf("rewind keymap temp file: %w", err)
	}
	return file, len(raw), nil
}

func (c *wlClient) readMessage() (uint32, uint16, []byte, error) {
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
	if len(payload) > 0 {
		if _, err := io.ReadFull(c.reader, payload); err != nil {
			return 0, 0, nil, err
		}
	}
	return object, opcode, payload, nil
}

func (c *wlClient) send(object uint32, opcode uint16, payload []byte, fds []int) error {
	size := 8 + len(payload)
	header := make([]byte, 8)
	binary.LittleEndian.PutUint32(header[:4], object)
	binary.LittleEndian.PutUint32(header[4:8], uint32(size)<<16|uint32(opcode))
	msg := append(header, payload...)
	if len(fds) == 0 {
		n, err := c.conn.Write(msg)
		if err != nil {
			return err
		}
		if n != len(msg) {
			return fmt.Errorf("short Wayland write: %d of %d bytes", n, len(msg))
		}
		return nil
	}
	oob := syscall.UnixRights(fds...)
	n, _, err := c.conn.WriteMsgUnix(msg, oob, nil)
	if err != nil {
		return err
	}
	if n != len(msg) {
		return fmt.Errorf("short Wayland fd write: %d of %d bytes", n, len(msg))
	}
	return nil
}

func (c *wlClient) newID() uint32 {
	id := c.nextID
	c.nextID++
	return id
}

func parseRegistryGlobal(payload []byte) (wlGlobal, error) {
	if len(payload) < 12 {
		return wlGlobal{}, fmt.Errorf("registry global payload too short")
	}
	name := binary.LittleEndian.Uint32(payload[:4])
	iface, off, err := parseWLString(payload, 4)
	if err != nil {
		return wlGlobal{}, err
	}
	if len(payload[off:]) < 4 {
		return wlGlobal{}, fmt.Errorf("registry global missing version")
	}
	version := binary.LittleEndian.Uint32(payload[off : off+4])
	return wlGlobal{Name: name, Iface: iface, Version: version}, nil
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
	var buf bytes.Buffer
	buf.Write(u32(uint32(len(raw))))
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

func wlFixed(value int32) []byte {
	return u32(uint32(value << 8))
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

func eventTime() uint32 {
	return uint32(time.Now().UnixMilli() & 0xffffffff)
}

func sortedGlobals(globals map[string]wlGlobal) []wlGlobal {
	out := make([]wlGlobal, 0, len(globals))
	for _, global := range globals {
		out = append(out, global)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Iface == out[j].Iface {
			return out[i].Name < out[j].Name
		}
		return out[i].Iface < out[j].Iface
	})
	return out
}
