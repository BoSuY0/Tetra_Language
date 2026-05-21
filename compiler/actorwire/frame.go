package actorwire

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/fnv"
)

const (
	Magic          = 0x52444154
	Version uint16 = 1

	MaxNodeID       = 127
	MaxPayloadSlots = 8

	FrameMagicOffset      = 0
	FrameVersionOffset    = 4
	FrameTypeOffset       = 6
	FrameSourceNodeOffset = 8
	FrameDestNodeOffset   = 10
	FrameSequenceOffset   = 12
	FrameActorIDOffset    = 16
	FrameSlotCountOffset  = 18
	FrameTagOffset        = 20
	FrameStatusOffset     = 24
	FramePayloadOffset    = 28
	FrameSize             = FramePayloadOffset + MaxPayloadSlots*4

	remoteHandleMask = uint32(1 << 31)
	nodeIDMask       = uint32(0x7f)
	actorIDMask      = uint32(0xffff)
)

var (
	ErrBadMagic           = errors.New("actor wire: bad magic")
	ErrUnsupportedVersion = errors.New("actor wire: unsupported version")
	ErrInvalidFrameType   = errors.New("actor wire: invalid frame type")
	ErrInvalidNodeID      = errors.New("actor wire: invalid node id")
	ErrInvalidSlotCount   = errors.New("actor wire: invalid slot count")
	ErrShortFrame         = errors.New("actor wire: short frame")
	ErrLocalActorHandle   = errors.New("actor wire: local actor handle")
)

type FrameType uint16

const (
	FrameHello FrameType = iota + 1
	FrameHelloAck
	FrameSpawnReq
	FrameSpawnAck
	FrameSendI32
	FrameSendMsg
	FrameSendTyped
	FrameNodeDown
	FrameError
)

const (
	StatusOK int32 = iota
	StatusNodeUnavailable
	StatusDuplicateNode
	StatusDecodeError
)

type Frame struct {
	Type         FrameType
	SourceNodeID uint16
	DestNodeID   uint16
	SequenceID   uint32
	ActorID      uint16
	Tag          int32
	Status       int32
	Payload      []int32
}

type RemoteHandle struct {
	NodeID  uint16
	ActorID uint16
}

func EncodeFrame(frame Frame) ([]byte, error) {
	if err := validateFrame(frame); err != nil {
		return nil, err
	}

	data := make([]byte, FrameSize)
	binary.LittleEndian.PutUint32(data[FrameMagicOffset:], Magic)
	binary.LittleEndian.PutUint16(data[FrameVersionOffset:], Version)
	binary.LittleEndian.PutUint16(data[FrameTypeOffset:], uint16(frame.Type))
	binary.LittleEndian.PutUint16(data[FrameSourceNodeOffset:], frame.SourceNodeID)
	binary.LittleEndian.PutUint16(data[FrameDestNodeOffset:], frame.DestNodeID)
	binary.LittleEndian.PutUint32(data[FrameSequenceOffset:], frame.SequenceID)
	binary.LittleEndian.PutUint16(data[FrameActorIDOffset:], frame.ActorID)
	binary.LittleEndian.PutUint16(data[FrameSlotCountOffset:], uint16(len(frame.Payload)))
	binary.LittleEndian.PutUint32(data[FrameTagOffset:], uint32(frame.Tag))
	binary.LittleEndian.PutUint32(data[FrameStatusOffset:], uint32(frame.Status))
	for i, slot := range frame.Payload {
		offset := FramePayloadOffset + i*4
		binary.LittleEndian.PutUint32(data[offset:], uint32(slot))
	}
	return data, nil
}

func DecodeFrame(data []byte) (Frame, error) {
	if len(data) < FrameSize {
		return Frame{}, fmt.Errorf("%w: got %d bytes, want %d", ErrShortFrame, len(data), FrameSize)
	}
	if got := binary.LittleEndian.Uint32(data[FrameMagicOffset:]); got != Magic {
		return Frame{}, fmt.Errorf("%w: 0x%x", ErrBadMagic, got)
	}
	if got := binary.LittleEndian.Uint16(data[FrameVersionOffset:]); got != Version {
		return Frame{}, fmt.Errorf("%w: %d", ErrUnsupportedVersion, got)
	}

	slotCount := binary.LittleEndian.Uint16(data[FrameSlotCountOffset:])
	if slotCount > MaxPayloadSlots {
		return Frame{}, fmt.Errorf("%w: %d", ErrInvalidSlotCount, slotCount)
	}

	frame := Frame{
		Type:         FrameType(binary.LittleEndian.Uint16(data[FrameTypeOffset:])),
		SourceNodeID: binary.LittleEndian.Uint16(data[FrameSourceNodeOffset:]),
		DestNodeID:   binary.LittleEndian.Uint16(data[FrameDestNodeOffset:]),
		SequenceID:   binary.LittleEndian.Uint32(data[FrameSequenceOffset:]),
		ActorID:      binary.LittleEndian.Uint16(data[FrameActorIDOffset:]),
		Tag:          int32(binary.LittleEndian.Uint32(data[FrameTagOffset:])),
		Status:       int32(binary.LittleEndian.Uint32(data[FrameStatusOffset:])),
		Payload:      make([]int32, slotCount),
	}
	if err := validateFrame(frame); err != nil {
		return Frame{}, err
	}
	for i := range frame.Payload {
		offset := FramePayloadOffset + i*4
		frame.Payload[i] = int32(binary.LittleEndian.Uint32(data[offset:]))
	}
	return frame, nil
}

func TypedMessageTagBase(typeName string) int32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(typeName))
	return int32(h.Sum32() & 0x7FFFFF00)
}

func TypedMessageTag(typeName string, ordinal int32) int32 {
	return TypedMessageTagBase(typeName) + ordinal
}

func EncodeRemoteHandle(nodeID, actorID uint16) (int32, error) {
	if err := validateNodeID(nodeID); err != nil {
		return 0, err
	}
	handle := remoteHandleMask | (uint32(nodeID) << 16) | uint32(actorID)
	return int32(handle), nil
}

func DecodeRemoteHandle(handle int32) (RemoteHandle, error) {
	if !IsRemoteHandle(handle) {
		return RemoteHandle{}, ErrLocalActorHandle
	}
	raw := uint32(handle)
	ref := RemoteHandle{
		NodeID:  uint16((raw >> 16) & nodeIDMask),
		ActorID: uint16(raw & actorIDMask),
	}
	if err := validateNodeID(ref.NodeID); err != nil {
		return RemoteHandle{}, err
	}
	return ref, nil
}

func IsRemoteHandle(handle int32) bool {
	return uint32(handle)&remoteHandleMask != 0
}

func validateFrame(frame Frame) error {
	if !isKnownFrameType(frame.Type) {
		return fmt.Errorf("%w: %d", ErrInvalidFrameType, frame.Type)
	}
	if err := validateNodeID(frame.SourceNodeID); err != nil {
		return err
	}
	if err := validateNodeID(frame.DestNodeID); err != nil {
		return err
	}
	if len(frame.Payload) > MaxPayloadSlots {
		return fmt.Errorf("%w: %d", ErrInvalidSlotCount, len(frame.Payload))
	}
	return nil
}

func isKnownFrameType(frameType FrameType) bool {
	switch frameType {
	case FrameHello,
		FrameHelloAck,
		FrameSpawnReq,
		FrameSpawnAck,
		FrameSendI32,
		FrameSendMsg,
		FrameSendTyped,
		FrameNodeDown,
		FrameError:
		return true
	default:
		return false
	}
}

func validateNodeID(nodeID uint16) error {
	if nodeID == 0 || nodeID > MaxNodeID {
		return fmt.Errorf("%w: %d", ErrInvalidNodeID, nodeID)
	}
	return nil
}
