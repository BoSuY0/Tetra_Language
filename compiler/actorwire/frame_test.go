package actorwire

import (
	"encoding/binary"
	"errors"
	"reflect"
	"testing"
)

func TestEncodeDecodeRoundTripSendTypedFrame(t *testing.T) {
	frame := Frame{
		Type:         FrameSendTyped,
		SourceNodeID: 1,
		DestNodeID:   2,
		SequenceID:   99,
		ActorID:      7,
		Tag:          42,
		Payload:      []int32{11, -22, 33},
	}

	data, err := EncodeFrame(frame)
	if err != nil {
		t.Fatalf("EncodeFrame returned error: %v", err)
	}
	if len(data) != FrameSize {
		t.Fatalf("encoded frame size = %d, want %d", len(data), FrameSize)
	}

	got, err := DecodeFrame(data)
	if err != nil {
		t.Fatalf("DecodeFrame returned error: %v", err)
	}
	if got.Type != frame.Type ||
		got.SourceNodeID != frame.SourceNodeID ||
		got.DestNodeID != frame.DestNodeID ||
		got.SequenceID != frame.SequenceID ||
		got.ActorID != frame.ActorID ||
		got.Tag != frame.Tag {
		t.Fatalf("decoded header = %+v, want %+v", got, frame)
	}
	if !reflect.DeepEqual(got.Payload, frame.Payload) {
		t.Fatalf("decoded payload = %v, want %v", got.Payload, frame.Payload)
	}
}

func TestDecodeRejectsBadHeaderAndInvalidSlotCount(t *testing.T) {
	frame := Frame{
		Type:         FrameSendI32,
		SourceNodeID: 1,
		DestNodeID:   2,
		SequenceID:   1,
		ActorID:      3,
		Payload:      []int32{5},
	}

	data, err := EncodeFrame(frame)
	if err != nil {
		t.Fatalf("EncodeFrame returned error: %v", err)
	}

	badMagic := append([]byte(nil), data...)
	badMagic[0] ^= 0xff
	if _, err := DecodeFrame(badMagic); !errors.Is(err, ErrBadMagic) {
		t.Fatalf("DecodeFrame bad magic error = %v, want %v", err, ErrBadMagic)
	}

	badVersion := append([]byte(nil), data...)
	binary.LittleEndian.PutUint16(badVersion[FrameVersionOffset:], Version+1)
	if _, err := DecodeFrame(badVersion); !errors.Is(err, ErrUnsupportedVersion) {
		t.Fatalf("DecodeFrame bad version error = %v, want %v", err, ErrUnsupportedVersion)
	}

	badSlotCount := append([]byte(nil), data...)
	binary.LittleEndian.PutUint16(badSlotCount[FrameSlotCountOffset:], MaxPayloadSlots+1)
	if _, err := DecodeFrame(badSlotCount); !errors.Is(err, ErrInvalidSlotCount) {
		t.Fatalf("DecodeFrame bad slot count error = %v, want %v", err, ErrInvalidSlotCount)
	}

	if _, err := DecodeFrame(data[:FrameSize-1]); !errors.Is(err, ErrShortFrame) {
		t.Fatalf("DecodeFrame short frame error = %v, want %v", err, ErrShortFrame)
	}
}

func TestEncodeRejectsFramesOutsideProtocolBounds(t *testing.T) {
	oversized := Frame{
		Type:         FrameSendTyped,
		SourceNodeID: 1,
		DestNodeID:   2,
		ActorID:      3,
		Payload:      []int32{1, 2, 3, 4, 5, 6, 7, 8, 9},
	}
	if _, err := EncodeFrame(oversized); !errors.Is(err, ErrInvalidSlotCount) {
		t.Fatalf("EncodeFrame oversized payload error = %v, want %v", err, ErrInvalidSlotCount)
	}

	invalidNode := Frame{
		Type:         FrameSendI32,
		SourceNodeID: 128,
		DestNodeID:   2,
		ActorID:      3,
		Payload:      []int32{1},
	}
	if _, err := EncodeFrame(invalidNode); !errors.Is(err, ErrInvalidNodeID) {
		t.Fatalf("EncodeFrame invalid node error = %v, want %v", err, ErrInvalidNodeID)
	}
}

func TestRemoteActorHandleEncoding(t *testing.T) {
	handle, err := EncodeRemoteHandle(2, 7)
	if err != nil {
		t.Fatalf("EncodeRemoteHandle returned error: %v", err)
	}
	if !IsRemoteHandle(handle) {
		t.Fatalf("IsRemoteHandle(%d) = false, want true", handle)
	}

	ref, err := DecodeRemoteHandle(handle)
	if err != nil {
		t.Fatalf("DecodeRemoteHandle returned error: %v", err)
	}
	if ref.NodeID != 2 || ref.ActorID != 7 {
		t.Fatalf("decoded remote handle = %+v, want node 2 actor 7", ref)
	}
	if IsRemoteHandle(7) {
		t.Fatalf("local actor handle was detected as remote")
	}

	if _, err := EncodeRemoteHandle(0, 7); !errors.Is(err, ErrInvalidNodeID) {
		t.Fatalf("EncodeRemoteHandle node 0 error = %v, want %v", err, ErrInvalidNodeID)
	}
	if _, err := EncodeRemoteHandle(128, 7); !errors.Is(err, ErrInvalidNodeID) {
		t.Fatalf("EncodeRemoteHandle node 128 error = %v, want %v", err, ErrInvalidNodeID)
	}
	if _, err := DecodeRemoteHandle(7); !errors.Is(err, ErrLocalActorHandle) {
		t.Fatalf("DecodeRemoteHandle local handle error = %v, want %v", err, ErrLocalActorHandle)
	}
}
