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

func TestTypedMessageTagDerivesStableTypeBase(t *testing.T) {
	if got, want := TypedMessageTagBase("RemoteMsg"), int32(0x31b33200); got != want {
		t.Fatalf("TypedMessageTagBase(RemoteMsg) = %#x, want %#x", uint32(got), uint32(want))
	}
	if got, want := TypedMessageTag("RemoteMsg", 7), int32(0x31b33207); got != want {
		t.Fatalf("TypedMessageTag(RemoteMsg, 7) = %#x, want %#x", uint32(got), uint32(want))
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

func TestDecodeForActorRuntimeABIV2RejectsOldWireFrame(t *testing.T) {
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

	if _, err := DecodeFrameForActorRefSlots(data, 2); !errors.Is(err, ErrActorWireABIMismatch) {
		t.Fatalf("DecodeFrameForActorRefSlots old-wire error = %v, want %v", err, ErrActorWireABIMismatch)
	}
	if _, err := DecodeFrameForActorRefSlots(data, 1); err != nil {
		t.Fatalf("DecodeFrameForActorRefSlots legacy slots returned error: %v", err)
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

func TestActorRefV2LocalRoundTripCarriesSlotAndGeneration(t *testing.T) {
	ref, err := NewLocalActorRef(42, 9)
	if err != nil {
		t.Fatalf("NewLocalActorRef returned error: %v", err)
	}
	if ref.IsRemote() {
		t.Fatalf("local actor ref was detected as remote")
	}
	parts, err := DecodeActorRefV2(ref.Raw())
	if err != nil {
		t.Fatalf("DecodeActorRefV2 returned error: %v", err)
	}
	if parts.Kind != ActorRefLocal || parts.Slot != 42 || parts.Generation != 9 ||
		parts.NodeID != 0 || parts.NodeEpoch != 0 {
		t.Fatalf("decoded local actor ref = %+v", parts)
	}
	if err := ValidateActorRefGeneration(ref.Raw(), 9); err != nil {
		t.Fatalf("ValidateActorRefGeneration current generation failed: %v", err)
	}
}

func TestActorRefV2RemoteRoundTripCarriesNodeEpochSlotAndGeneration(t *testing.T) {
	ref, err := NewRemoteActorRef(7, 11, 42, 9)
	if err != nil {
		t.Fatalf("NewRemoteActorRef returned error: %v", err)
	}
	if !ref.IsRemote() {
		t.Fatalf("remote actor ref was detected as local")
	}
	parts, err := DecodeActorRefV2(ref.Raw())
	if err != nil {
		t.Fatalf("DecodeActorRefV2 returned error: %v", err)
	}
	if parts.Kind != ActorRefRemote || parts.NodeID != 7 || parts.NodeEpoch != 11 ||
		parts.Slot != 42 || parts.Generation != 9 {
		t.Fatalf("decoded remote actor ref = %+v", parts)
	}
	if err := ValidateActorRefEpoch(ref.Raw(), 11); err != nil {
		t.Fatalf("ValidateActorRefEpoch current epoch failed: %v", err)
	}
}

func TestActorRefV2RejectsForgedLegacyAndZeroGenerationHandles(t *testing.T) {
	legacy, err := EncodeRemoteHandle(2, 7)
	if err != nil {
		t.Fatalf("EncodeRemoteHandle returned error: %v", err)
	}
	if _, err := DecodeActorRefV2(uint64(uint32(legacy))); !errors.Is(err, ErrUnsupportedActorRefVersion) {
		t.Fatalf("DecodeActorRefV2 legacy handle error = %v, want %v", err, ErrUnsupportedActorRefVersion)
	}
	if _, err := NewLocalActorRef(42, 0); !errors.Is(err, ErrInvalidActorGeneration) {
		t.Fatalf("NewLocalActorRef zero generation error = %v, want %v", err, ErrInvalidActorGeneration)
	}
	if _, err := NewRemoteActorRef(7, 0, 42, 9); !errors.Is(err, ErrInvalidNodeEpoch) {
		t.Fatalf("NewRemoteActorRef zero epoch error = %v, want %v", err, ErrInvalidNodeEpoch)
	}
}

func TestActorRefV2RejectsStaleGenerationAndNodeEpoch(t *testing.T) {
	local, err := NewLocalActorRef(42, 9)
	if err != nil {
		t.Fatalf("NewLocalActorRef returned error: %v", err)
	}
	if err := ValidateActorRefGeneration(local.Raw(), 10); !errors.Is(err, ErrStaleActorGeneration) {
		t.Fatalf("ValidateActorRefGeneration stale error = %v, want %v", err, ErrStaleActorGeneration)
	}

	remote, err := NewRemoteActorRef(7, 11, 42, 9)
	if err != nil {
		t.Fatalf("NewRemoteActorRef returned error: %v", err)
	}
	if err := ValidateActorRefEpoch(remote.Raw(), 12); !errors.Is(err, ErrStaleNodeEpoch) {
		t.Fatalf("ValidateActorRefEpoch stale error = %v, want %v", err, ErrStaleNodeEpoch)
	}
}
