package surfacehost

import (
	"context"
	"errors"
	"fmt"
	"io"
)

type Backend interface {
	Open(title string, width int32, height int32) (uint32, error)
	Close(handle uint32) error
	BeginFrame(handle uint32) error
	PresentRGBA(handle uint32, width int32, height int32, stride int32, rgba []byte) error
	PollEvent(handle uint32) (Event, error)
	PollEventText(handle uint32) ([]byte, error)
	ClipboardWriteText(handle uint32, text []byte) (int32, error)
	ClipboardReadText(handle uint32) ([]byte, error)
	PollComposition(handle uint32) ([4]int32, error)
	NowMS() int32
	RequestRedraw(handle uint32) error
}

type ReadWriter interface {
	io.Reader
	io.Writer
}

func ServeConn(ctx context.Context, rw ReadWriter, backend Backend) error {
	var connectionHandle uint32
	responses := make(chan Response, 128)
	writerDone := make(chan error, 1)
	go func() {
		for resp := range responses {
			if err := WriteResponse(rw, resp); err != nil {
				if errors.Is(err, io.ErrClosedPipe) {
					writerDone <- nil
					return
				}
				writerDone <- err
				return
			}
		}
		writerDone <- nil
	}()
	defer close(responses)

	sendResponse := func(resp Response) error {
		select {
		case responses <- resp:
			return nil
		case err := <-writerDone:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-writerDone:
			return err
		default:
		}
		req, err := ReadRequest(rw)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrClosedPipe) {
				return nil
			}
			return err
		}
		if connectionHandle != 0 && req.Op != OpOpen {
			req.Handle = connectionHandle
		}
		resp := handleRequest(req, backend)
		if req.Op == OpOpen && resp.Status == 0 && resp.Value0 > 0 {
			connectionHandle = uint32(resp.Value0)
		}
		if req.Op == OpClose && resp.Status == 0 {
			connectionHandle = 0
		}
		if err := sendResponse(resp); err != nil {
			return err
		}
	}
}

func handleRequest(req Request, backend Backend) Response {
	resp := Response{Op: req.Op, RequestID: req.RequestID}
	fail := func(err error) Response {
		resp.Status = 1
		resp.Payload = []byte(err.Error())
		return resp
	}
	switch req.Op {
	case OpOpen:
		handle, err := backend.Open(string(req.Payload), req.Width, req.Height)
		if err != nil {
			return fail(err)
		}
		resp.Value0 = int32(handle)
	case OpClose:
		if err := backend.Close(req.Handle); err != nil {
			return fail(err)
		}
	case OpBeginFrame:
		if err := backend.BeginFrame(req.Handle); err != nil {
			return fail(err)
		}
	case OpPresentRGBA:
		if err := backend.PresentRGBA(
			req.Handle,
			req.Width,
			req.Height,
			req.Stride,
			req.Payload,
		); err != nil {
			return fail(err)
		}
	case OpPollEventInto:
		event, err := backend.PollEvent(req.Handle)
		if err != nil {
			return fail(err)
		}
		resp.Value0 = event.Kind
		resp.Payload = EncodeEvent(event)
	case OpPollEventTextInto:
		payload, err := backend.PollEventText(req.Handle)
		if err != nil {
			return fail(err)
		}
		resp.Value0 = int32(len(payload))
		if req.Width > 0 {
			resp.Payload = capPayload(payload, req.Width)
			resp.Value0 = int32(len(resp.Payload))
		}
	case OpClipboardWriteText:
		n, err := backend.ClipboardWriteText(req.Handle, req.Payload)
		if err != nil {
			return fail(err)
		}
		resp.Value0 = n
	case OpClipboardReadText:
		payload, err := backend.ClipboardReadText(req.Handle)
		if err != nil {
			return fail(err)
		}
		resp.Payload = capPayload(payload, req.Width)
		resp.Value0 = int32(len(resp.Payload))
	case OpPollCompositionInto:
		slots, err := backend.PollComposition(req.Handle)
		if err != nil {
			return fail(err)
		}
		resp.Payload = EncodeComposition(slots)
	case OpNowMS:
		resp.Value0 = backend.NowMS()
	case OpRequestRedraw:
		if err := backend.RequestRedraw(req.Handle); err != nil {
			return fail(err)
		}
	default:
		return fail(fmt.Errorf("unsupported Surface host op %d", req.Op))
	}
	return resp
}

func capPayload(payload []byte, max int32) []byte {
	if max <= 0 {
		return nil
	}
	if int64(len(payload)) > int64(max) {
		return payload[:max]
	}
	return payload
}
