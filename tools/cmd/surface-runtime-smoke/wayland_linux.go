//go:build linux

package main

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
	"syscall"
	"time"
)

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
	xdgToplevelID uint32
	configured    bool
	configSerial  uint32
	closed        bool
}

func presentRealWindowSurface(
	title string,
	frame rgbaFrame,
	dwell time.Duration,
	holdUntilClose bool,
) error {
	socketPath, err := waylandSocketPath()
	if err != nil {
		return err
	}
	addr := net.UnixAddr{Name: socketPath, Net: "unix"}
	conn, err := net.DialUnix("unix", nil, &addr)
	if err != nil {
		return fmt.Errorf("connect Wayland compositor at %s: %w", socketPath, err)
	}
	defer conn.Close()

	client := &waylandClient{conn: conn, reader: bufio.NewReader(conn), nextID: 2}
	if err := client.roundtripRegistry(); err != nil {
		return err
	}
	if client.compositor.Name == 0 || client.shm.Name == 0 || client.xdgWMBase.Name == 0 {
		return fmt.Errorf(
			"Wayland compositor missing required globals: wl_compositor=%d wl_shm=%d xdg_wm_base=%d",
			client.compositor.Name,
			client.shm.Name,
			client.xdgWMBase.Name,
		)
	}

	compositorID := client.newID()
	shmID := client.newID()
	wmBaseID := client.newID()
	if err := client.bind(
		client.compositor.Name,
		"wl_compositor",
		minVersion(client.compositor.Version, 4),
		compositorID,
	); err != nil {
		return err
	}
	if err := client.bind(
		client.shm.Name,
		"wl_shm",
		minVersion(client.shm.Version, 1),
		shmID,
	); err != nil {
		return err
	}
	if err := client.bind(
		client.xdgWMBase.Name,
		"xdg_wm_base",
		minVersion(client.xdgWMBase.Version, 1),
		wmBaseID,
	); err != nil {
		return err
	}

	surfaceID := client.newID()
	xdgSurfaceID := client.newID()
	toplevelID := client.newID()
	client.xdgToplevelID = toplevelID
	if err := client.send(compositorID, 0, u32(surfaceID), nil); err != nil {
		return fmt.Errorf("create wl_surface: %w", err)
	}
	if err := client.send(wmBaseID, 2, concat(u32(xdgSurfaceID), u32(surfaceID)), nil); err != nil {
		return fmt.Errorf("create xdg_surface: %w", err)
	}
	if err := client.send(xdgSurfaceID, 1, u32(toplevelID), nil); err != nil {
		return fmt.Errorf("create xdg_toplevel: %w", err)
	}
	if err := client.send(toplevelID, 2, wlString(title), nil); err != nil {
		return fmt.Errorf("set xdg_toplevel title: %w", err)
	}
	if err := client.send(toplevelID, 3, wlString("tetra-surface-real-window"), nil); err != nil {
		return fmt.Errorf("set xdg_toplevel app id: %w", err)
	}
	if err := client.send(surfaceID, 6, nil, nil); err != nil {
		return fmt.Errorf("initial wl_surface commit: %w", err)
	}
	if err := client.waitForConfigure(xdgSurfaceID, wmBaseID, 2*time.Second); err != nil {
		return err
	}
	if err := client.send(xdgSurfaceID, 4, u32(client.configSerial), nil); err != nil {
		return fmt.Errorf("ack xdg_surface configure: %w", err)
	}

	shmFile, size, err := writeWaylandSHMFrame(frame)
	if err != nil {
		return err
	}
	defer os.Remove(shmFile.Name())
	defer shmFile.Close()

	poolID := client.newID()
	bufferID := client.newID()
	if err := client.send(
		shmID,
		0,
		concat(u32(poolID), i32(int32(size))),
		[]int{int(shmFile.Fd())},
	); err != nil {
		return fmt.Errorf("create wl_shm_pool: %w", err)
	}
	if err := client.send(
		poolID,
		0,
		concat(u32(bufferID), i32(0), i32(int32(frame.Width)), i32(int32(frame.Height)), i32(int32(frame.Stride)), u32(0)),
		nil,
	); err != nil {
		return fmt.Errorf("create wl_buffer: %w", err)
	}
	if err := client.send(surfaceID, 1, concat(u32(bufferID), i32(0), i32(0)), nil); err != nil {
		return fmt.Errorf("attach wl_buffer: %w", err)
	}
	if err := client.send(
		surfaceID,
		2,
		concat(i32(0), i32(0), i32(int32(frame.Width)), i32(int32(frame.Height))),
		nil,
	); err != nil {
		return fmt.Errorf("damage wl_surface: %w", err)
	}
	if err := client.send(surfaceID, 6, nil, nil); err != nil {
		return fmt.Errorf("present wl_surface commit: %w", err)
	}

	deadline := time.Now().Add(dwell)
	for holdUntilClose || time.Now().Before(deadline) {
		if client.closed {
			return nil
		}
		_ = client.conn.SetReadDeadline(time.Now().Add(25 * time.Millisecond))
		if err := client.readOne(wmBaseID, xdgSurfaceID); err != nil {
			if isTimeout(err) {
				continue
			}
			return err
		}
	}
	return nil
}

func waylandSocketPath() (string, error) {
	display := os.Getenv("WAYLAND_DISPLAY")
	if strings.TrimSpace(display) == "" {
		return "", fmt.Errorf(
			"WAYLAND_DISPLAY is not set; cannot collect linux-x64 real-window Surface evidence",
		)
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
		if err := c.readOne(wmBaseID, xdgSurfaceID); err != nil {
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
	if err := c.handleCommonEvent(object, opcode, payload, 0, 0); err != nil {
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
		}
	}
	return object == callbackID && opcode == 0, nil
}

func (c *waylandClient) readOne(wmBaseID uint32, xdgSurfaceID uint32) error {
	object, opcode, payload, err := c.readMessage()
	if err != nil {
		return err
	}
	return c.handleCommonEvent(object, opcode, payload, wmBaseID, xdgSurfaceID)
}

func (c *waylandClient) handleCommonEvent(
	object uint32,
	opcode uint16,
	payload []byte,
	wmBaseID uint32,
	xdgSurfaceID uint32,
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
	return nil
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

func writeWaylandSHMFrame(frame rgbaFrame) (*os.File, int, error) {
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = os.TempDir()
	}
	file, err := os.CreateTemp(runtimeDir, "tetra-surface-shm-*")
	if err != nil {
		return nil, 0, fmt.Errorf("create Wayland shm file: %w", err)
	}
	size := frame.Stride * frame.Height
	if err := file.Truncate(int64(size)); err != nil {
		file.Close()
		return nil, 0, fmt.Errorf("resize Wayland shm file: %w", err)
	}
	argb := rgbaToWaylandARGB(frame)
	if _, err := file.WriteAt(argb, 0); err != nil {
		file.Close()
		return nil, 0, fmt.Errorf("write Wayland shm pixels: %w", err)
	}
	return file, size, nil
}

func rgbaToWaylandARGB(frame rgbaFrame) []byte {
	out := make([]byte, len(frame.Pixels))
	for i := 0; i+3 < len(frame.Pixels); i += 4 {
		r := frame.Pixels[i]
		g := frame.Pixels[i+1]
		b := frame.Pixels[i+2]
		a := frame.Pixels[i+3]
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
