package surfacehost

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	ProtocolName        = "tetra.surface.host-ipc.v1"
	Magic        uint32 = 0x31534854

	requestHeaderSize  = 32
	responseHeaderSize = 36
	eventPayloadSize   = 36
)

var ErrBadMagic = errors.New("bad Tetra Surface host IPC magic")

type Op uint32

const (
	OpOpen Op = iota + 1
	OpClose
	OpBeginFrame
	OpPresentRGBA
	OpPollEventInto
	OpPollEventTextInto
	OpClipboardWriteText
	OpClipboardReadText
	OpPollCompositionInto
	OpNowMS
	OpRequestRedraw
)

type Request struct {
	Op        Op
	RequestID uint32
	Handle    uint32
	Width     int32
	Height    int32
	Stride    int32
	Payload   []byte
}

type Response struct {
	Op        Op
	RequestID uint32
	Status    int32
	Value0    int32
	Value1    int32
	Value2    int32
	Value3    int32
	Payload   []byte
}

type Event struct {
	Kind        int32
	X           int32
	Y           int32
	Button      int32
	Key         int32
	Width       int32
	Height      int32
	TimestampMS int32
	TextLen     int32
}

func WriteRequest(w io.Writer, req Request) error {
	header := make([]byte, requestHeaderSize)
	binary.LittleEndian.PutUint32(header[0:4], Magic)
	binary.LittleEndian.PutUint32(header[4:8], uint32(req.Op))
	binary.LittleEndian.PutUint32(header[8:12], req.RequestID)
	binary.LittleEndian.PutUint32(header[12:16], req.Handle)
	binary.LittleEndian.PutUint32(header[16:20], uint32(req.Width))
	binary.LittleEndian.PutUint32(header[20:24], uint32(req.Height))
	binary.LittleEndian.PutUint32(header[24:28], uint32(req.Stride))
	binary.LittleEndian.PutUint32(header[28:32], uint32(len(req.Payload)))
	if _, err := w.Write(header); err != nil {
		return err
	}
	if len(req.Payload) > 0 {
		_, err := w.Write(req.Payload)
		return err
	}
	return nil
}

func ReadRequest(r io.Reader) (Request, error) {
	header := make([]byte, requestHeaderSize)
	if _, err := io.ReadFull(r, header); err != nil {
		return Request{}, err
	}
	if got := binary.LittleEndian.Uint32(header[0:4]); got != Magic {
		return Request{}, fmt.Errorf("%w: request got 0x%08x", ErrBadMagic, got)
	}
	payloadLen := binary.LittleEndian.Uint32(header[28:32])
	payload := make([]byte, payloadLen)
	if payloadLen > 0 {
		if _, err := io.ReadFull(r, payload); err != nil {
			return Request{}, err
		}
	}
	return Request{
		Op:        Op(binary.LittleEndian.Uint32(header[4:8])),
		RequestID: binary.LittleEndian.Uint32(header[8:12]),
		Handle:    binary.LittleEndian.Uint32(header[12:16]),
		Width:     int32(binary.LittleEndian.Uint32(header[16:20])),
		Height:    int32(binary.LittleEndian.Uint32(header[20:24])),
		Stride:    int32(binary.LittleEndian.Uint32(header[24:28])),
		Payload:   payload,
	}, nil
}

func WriteResponse(w io.Writer, resp Response) error {
	header := make([]byte, responseHeaderSize)
	binary.LittleEndian.PutUint32(header[0:4], Magic)
	binary.LittleEndian.PutUint32(header[4:8], uint32(resp.Op))
	binary.LittleEndian.PutUint32(header[8:12], resp.RequestID)
	binary.LittleEndian.PutUint32(header[12:16], uint32(resp.Status))
	binary.LittleEndian.PutUint32(header[16:20], uint32(resp.Value0))
	binary.LittleEndian.PutUint32(header[20:24], uint32(resp.Value1))
	binary.LittleEndian.PutUint32(header[24:28], uint32(resp.Value2))
	binary.LittleEndian.PutUint32(header[28:32], uint32(resp.Value3))
	binary.LittleEndian.PutUint32(header[32:36], uint32(len(resp.Payload)))
	if _, err := w.Write(header); err != nil {
		return err
	}
	if len(resp.Payload) > 0 {
		_, err := w.Write(resp.Payload)
		return err
	}
	return nil
}

func ReadResponse(r io.Reader) (Response, error) {
	header := make([]byte, responseHeaderSize)
	if _, err := io.ReadFull(r, header); err != nil {
		return Response{}, err
	}
	if got := binary.LittleEndian.Uint32(header[0:4]); got != Magic {
		return Response{}, fmt.Errorf("%w: response got 0x%08x", ErrBadMagic, got)
	}
	payloadLen := binary.LittleEndian.Uint32(header[32:36])
	payload := make([]byte, payloadLen)
	if payloadLen > 0 {
		if _, err := io.ReadFull(r, payload); err != nil {
			return Response{}, err
		}
	}
	return Response{
		Op:        Op(binary.LittleEndian.Uint32(header[4:8])),
		RequestID: binary.LittleEndian.Uint32(header[8:12]),
		Status:    int32(binary.LittleEndian.Uint32(header[12:16])),
		Value0:    int32(binary.LittleEndian.Uint32(header[16:20])),
		Value1:    int32(binary.LittleEndian.Uint32(header[20:24])),
		Value2:    int32(binary.LittleEndian.Uint32(header[24:28])),
		Value3:    int32(binary.LittleEndian.Uint32(header[28:32])),
		Payload:   payload,
	}, nil
}

func EncodeEvent(event Event) []byte {
	payload := make([]byte, eventPayloadSize)
	values := []int32{
		event.Kind,
		event.X,
		event.Y,
		event.Button,
		event.Key,
		event.Width,
		event.Height,
		event.TimestampMS,
		event.TextLen,
	}
	for i, value := range values {
		binary.LittleEndian.PutUint32(payload[i*4:i*4+4], uint32(value))
	}
	return payload
}

func EncodeComposition(slots [4]int32) []byte {
	payload := make([]byte, 16)
	for i, value := range slots {
		binary.LittleEndian.PutUint32(payload[i*4:i*4+4], uint32(value))
	}
	return payload
}
