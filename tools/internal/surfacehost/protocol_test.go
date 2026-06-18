package surfacehost

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"
)

func TestProtocolRoundTripRequestAndResponse(t *testing.T) {
	var wire bytes.Buffer
	req := Request{
		Op:        OpOpen,
		RequestID: 7,
		Handle:    3,
		Width:     320,
		Height:    200,
		Stride:    1280,
		Payload:   []byte("Surface Window Counter"),
	}
	if err := WriteRequest(&wire, req); err != nil {
		t.Fatalf("WriteRequest: %v", err)
	}
	got, err := ReadRequest(&wire)
	if err != nil {
		t.Fatalf("ReadRequest: %v", err)
	}
	if got.Op != req.Op || got.RequestID != req.RequestID || got.Handle != req.Handle ||
		got.Width != req.Width || got.Height != req.Height || got.Stride != req.Stride ||
		string(got.Payload) != string(req.Payload) {
		t.Fatalf("request round trip = %#v, want %#v", got, req)
	}

	resp := Response{
		Op:        OpOpen,
		RequestID: 7,
		Status:    0,
		Value0:    41,
		Payload:   []byte("ok"),
	}
	if err := WriteResponse(&wire, resp); err != nil {
		t.Fatalf("WriteResponse: %v", err)
	}
	gotResp, err := ReadResponse(&wire)
	if err != nil {
		t.Fatalf("ReadResponse: %v", err)
	}
	if gotResp.Op != resp.Op || gotResp.RequestID != resp.RequestID ||
		gotResp.Status != resp.Status || gotResp.Value0 != resp.Value0 ||
		string(gotResp.Payload) != string(resp.Payload) {
		t.Fatalf("response round trip = %#v, want %#v", gotResp, resp)
	}
}

func TestReadRequestRejectsBadMagic(t *testing.T) {
	raw := make([]byte, requestHeaderSize)
	raw[0] = 0xff
	_, err := ReadRequest(bytes.NewReader(raw))
	if err == nil {
		t.Fatalf("expected bad magic to fail")
	}
	if !errors.Is(err, ErrBadMagic) {
		t.Fatalf("error = %v, want ErrBadMagic", err)
	}
}

func TestServeConnDispatchesOpenPresentPollAndClose(t *testing.T) {
	backend := &recordingBackend{
		nextHandle: 9,
		events:     []Event{{Kind: 5, X: 48, Y: 96, Button: 1, Width: 320, Height: 200}},
	}
	appToHostReader, appToHostWriter := io.Pipe()
	hostToAppReader, hostToAppWriter := io.Pipe()
	serverRW := pipeReadWriter{PipeReader: appToHostReader, PipeWriter: hostToAppWriter}
	clientRW := pipeReadWriter{PipeReader: hostToAppReader, PipeWriter: appToHostWriter}
	errCh := make(chan error, 1)
	go func() {
		errCh <- ServeConn(context.Background(), serverRW, backend)
	}()

	if err := WriteRequest(
		clientRW,
		Request{Op: OpOpen, RequestID: 1, Width: 320, Height: 200, Payload: []byte("Counter")},
	); err != nil {
		t.Fatalf("write open: %v", err)
	}
	openResp, err := ReadResponse(clientRW)
	if err != nil {
		t.Fatalf("read open response: %v", err)
	}
	if openResp.Status != 0 || openResp.Value0 != 9 {
		t.Fatalf("open response = %#v, want handle 9", openResp)
	}

	if err := WriteRequest(
		clientRW,
		Request{
			Op:        OpPresentRGBA,
			RequestID: 2,
			Handle:    9,
			Width:     2,
			Height:    2,
			Stride:    8,
			Payload:   []byte{1, 2, 3, 4},
		},
	); err != nil {
		t.Fatalf("write present: %v", err)
	}
	if resp, err := ReadResponse(clientRW); err != nil || resp.Status != 0 {
		t.Fatalf("present response = %#v, %v", resp, err)
	}
	if backend.presentedBytes != 4 {
		t.Fatalf("presented bytes = %d, want 4", backend.presentedBytes)
	}

	if err := WriteRequest(
		clientRW,
		Request{Op: OpPollEventInto, RequestID: 3, Handle: 9},
	); err != nil {
		t.Fatalf("write poll: %v", err)
	}
	pollResp, err := ReadResponse(clientRW)
	if err != nil {
		t.Fatalf("read poll response: %v", err)
	}
	if pollResp.Status != 0 || len(pollResp.Payload) != eventPayloadSize {
		t.Fatalf("poll response = %#v, want event payload", pollResp)
	}

	if err := WriteRequest(clientRW, Request{Op: OpClose, RequestID: 4, Handle: 9}); err != nil {
		t.Fatalf("write close: %v", err)
	}
	if resp, err := ReadResponse(clientRW); err != nil || resp.Status != 0 {
		t.Fatalf("close response = %#v, %v", resp, err)
	}
	_ = appToHostWriter.Close()
	_ = hostToAppReader.Close()
	if err := <-errCh; err != nil {
		t.Fatalf("ServeConn: %v", err)
	}
}

func TestServeConnBindsBackendHandleToConnection(t *testing.T) {
	backend := &recordingBackend{nextHandle: 17}
	appToHostReader, appToHostWriter := io.Pipe()
	hostToAppReader, hostToAppWriter := io.Pipe()
	serverRW := pipeReadWriter{PipeReader: appToHostReader, PipeWriter: hostToAppWriter}
	clientRW := pipeReadWriter{PipeReader: hostToAppReader, PipeWriter: appToHostWriter}
	errCh := make(chan error, 1)
	go func() {
		errCh <- ServeConn(context.Background(), serverRW, backend)
	}()

	if err := WriteRequest(
		clientRW,
		Request{Op: OpOpen, RequestID: 1, Width: 320, Height: 200, Payload: []byte("Counter")},
	); err != nil {
		t.Fatalf("write open: %v", err)
	}
	openResp, err := ReadResponse(clientRW)
	if err != nil {
		t.Fatalf("read open response: %v", err)
	}
	if openResp.Status != 0 || openResp.Value0 != 17 {
		t.Fatalf("open response = %#v, want handle 17", openResp)
	}

	if err := WriteRequest(
		clientRW,
		Request{
			Op:        OpPresentRGBA,
			RequestID: 2,
			Handle:    99,
			Width:     2,
			Height:    2,
			Stride:    8,
			Payload:   []byte{1, 2, 3, 4},
		},
	); err != nil {
		t.Fatalf("write present: %v", err)
	}
	if resp, err := ReadResponse(clientRW); err != nil || resp.Status != 0 {
		t.Fatalf("present response = %#v, %v", resp, err)
	}
	if backend.presentHandles[len(backend.presentHandles)-1] != 17 {
		t.Fatalf(
			"present handle = %d, want connection-bound backend handle 17",
			backend.presentHandles[len(backend.presentHandles)-1],
		)
	}

	if err := WriteRequest(clientRW, Request{Op: OpClose, RequestID: 3, Handle: 99}); err != nil {
		t.Fatalf("write close: %v", err)
	}
	if resp, err := ReadResponse(clientRW); err != nil || resp.Status != 0 {
		t.Fatalf("close response = %#v, %v", resp, err)
	}
	if backend.closeHandles[len(backend.closeHandles)-1] != 17 {
		t.Fatalf(
			"close handle = %d, want connection-bound backend handle 17",
			backend.closeHandles[len(backend.closeHandles)-1],
		)
	}

	_ = appToHostWriter.Close()
	_ = hostToAppReader.Close()
	if err := <-errCh; err != nil {
		t.Fatalf("ServeConn: %v", err)
	}
}

func TestServeConnCapsTextAndClipboardPayloadsByRequestWidth(t *testing.T) {
	backend := &recordingBackend{nextHandle: 23}
	appToHostReader, appToHostWriter := io.Pipe()
	hostToAppReader, hostToAppWriter := io.Pipe()
	serverRW := pipeReadWriter{PipeReader: appToHostReader, PipeWriter: hostToAppWriter}
	clientRW := pipeReadWriter{PipeReader: hostToAppReader, PipeWriter: appToHostWriter}
	errCh := make(chan error, 1)
	go func() {
		errCh <- ServeConn(context.Background(), serverRW, backend)
	}()

	if err := WriteRequest(
		clientRW,
		Request{Op: OpOpen, RequestID: 1, Width: 320, Height: 200, Payload: []byte("Counter")},
	); err != nil {
		t.Fatalf("write open: %v", err)
	}
	if resp, err := ReadResponse(clientRW); err != nil || resp.Status != 0 {
		t.Fatalf("open response = %#v, %v", resp, err)
	}

	if err := WriteRequest(
		clientRW,
		Request{Op: OpPollEventTextInto, RequestID: 2, Handle: 23},
	); err != nil {
		t.Fatalf("write text len request: %v", err)
	}
	textLenResp, err := ReadResponse(clientRW)
	if err != nil {
		t.Fatalf("read text len response: %v", err)
	}
	if textLenResp.Value0 != 2 || len(textLenResp.Payload) != 0 {
		t.Fatalf("text len response = %#v, want value0=2 no payload", textLenResp)
	}

	if err := WriteRequest(
		clientRW,
		Request{Op: OpClipboardReadText, RequestID: 3, Handle: 23, Width: 3},
	); err != nil {
		t.Fatalf("write clipboard read request: %v", err)
	}
	clipboardResp, err := ReadResponse(clientRW)
	if err != nil {
		t.Fatalf("read clipboard response: %v", err)
	}
	if clipboardResp.Value0 != 3 || string(clipboardResp.Payload) != "Tet" {
		t.Fatalf("clipboard response = %#v, want capped Tet", clipboardResp)
	}

	if err := WriteRequest(clientRW, Request{Op: OpClose, RequestID: 4, Handle: 23}); err != nil {
		t.Fatalf("write close: %v", err)
	}
	if resp, err := ReadResponse(clientRW); err != nil || resp.Status != 0 {
		t.Fatalf("close response = %#v, %v", resp, err)
	}
	_ = appToHostWriter.Close()
	_ = hostToAppReader.Close()
	if err := <-errCh; err != nil {
		t.Fatalf("ServeConn: %v", err)
	}
}

func TestServeConnDrainsPipelinedRequestBeforeClientReadsPollResponse(t *testing.T) {
	presentedCh := make(chan struct{}, 1)
	backend := &recordingBackend{
		nextHandle:  21,
		presentedCh: presentedCh,
	}
	appToHostReader, appToHostWriter := io.Pipe()
	hostToAppReader, hostToAppWriter := io.Pipe()
	serverRW := pipeReadWriter{PipeReader: appToHostReader, PipeWriter: hostToAppWriter}
	clientRW := pipeReadWriter{PipeReader: hostToAppReader, PipeWriter: appToHostWriter}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error, 1)
	go func() {
		errCh <- ServeConn(ctx, serverRW, backend)
	}()

	if err := WriteRequest(
		clientRW,
		Request{Op: OpOpen, RequestID: 1, Width: 320, Height: 200, Payload: []byte("Counter")},
	); err != nil {
		t.Fatalf("write open: %v", err)
	}
	if resp, err := ReadResponse(clientRW); err != nil || resp.Status != 0 {
		t.Fatalf("open response = %#v, %v", resp, err)
	}

	if err := WriteRequest(
		clientRW,
		Request{Op: OpPollEventInto, RequestID: 2, Handle: 21},
	); err != nil {
		t.Fatalf("write poll: %v", err)
	}
	presentWriteDone := make(chan error, 1)
	go func() {
		presentWriteDone <- WriteRequest(
			clientRW,
			Request{
				Op:        OpPresentRGBA,
				RequestID: 3,
				Handle:    21,
				Width:     2,
				Height:    2,
				Stride:    8,
				Payload:   []byte{1, 2, 3, 4},
			},
		)
	}()

	select {
	case <-presentedCh:
	case <-time.After(200 * time.Millisecond):
		pollResp, readErr := ReadResponse(clientRW)
		writeErr := <-presentWriteDone
		if readErr == nil {
			_, _ = ReadResponse(clientRW)
		}
		t.Fatalf(
			"ServeConn did not drain pipelined present before client read poll response; "+
				"pollResp=%#v readErr=%v writeErr=%v",
			pollResp,
			readErr,
			writeErr,
		)
	}

	pollResp, err := ReadResponse(clientRW)
	if err != nil {
		t.Fatalf("read poll response: %v", err)
	}
	if pollResp.Status != 0 || len(pollResp.Payload) != eventPayloadSize {
		t.Fatalf("poll response = %#v, want event payload", pollResp)
	}
	if err := <-presentWriteDone; err != nil {
		t.Fatalf("write pipelined present: %v", err)
	}
	if resp, err := ReadResponse(clientRW); err != nil || resp.Status != 0 {
		t.Fatalf("present response = %#v, %v", resp, err)
	}
	_ = appToHostWriter.Close()
	_ = hostToAppReader.Close()
	if err := <-errCh; err != nil {
		t.Fatalf("ServeConn: %v", err)
	}
}

type pipeReadWriter struct {
	*io.PipeReader
	*io.PipeWriter
}

type recordingBackend struct {
	nextHandle     uint32
	events         []Event
	presentedBytes int
	presentHandles []uint32
	closeHandles   []uint32
	presentedCh    chan struct{}
}

func (b *recordingBackend) Open(title string, width int32, height int32) (uint32, error) {
	return b.nextHandle, nil
}

func (b *recordingBackend) Close(handle uint32) error {
	b.closeHandles = append(b.closeHandles, handle)
	return nil
}

func (b *recordingBackend) BeginFrame(handle uint32) error {
	return nil
}

func (b *recordingBackend) PresentRGBA(
	handle uint32,
	width int32,
	height int32,
	stride int32,
	rgba []byte,
) error {
	b.presentHandles = append(b.presentHandles, handle)
	b.presentedBytes += len(rgba)
	if b.presentedCh != nil {
		select {
		case b.presentedCh <- struct{}{}:
		default:
		}
	}
	return nil
}

func (b *recordingBackend) PollEvent(handle uint32) (Event, error) {
	if len(b.events) == 0 {
		return Event{}, nil
	}
	event := b.events[0]
	b.events = b.events[1:]
	return event, nil
}

func (b *recordingBackend) PollEventText(handle uint32) ([]byte, error) {
	return []byte("OK"), nil
}

func (b *recordingBackend) ClipboardWriteText(handle uint32, text []byte) (int32, error) {
	return int32(len(text)), nil
}

func (b *recordingBackend) ClipboardReadText(handle uint32) ([]byte, error) {
	return []byte("Tetra"), nil
}

func (b *recordingBackend) PollComposition(handle uint32) ([4]int32, error) {
	return [4]int32{}, nil
}

func (b *recordingBackend) NowMS() int32 {
	return 123
}

func (b *recordingBackend) RequestRedraw(handle uint32) error {
	return nil
}
