package runtimeabi

import (
	"fmt"
	"math"
)

type RawPointerBoundsStatus string

const (
	RawPointerBoundsAllocationBase              RawPointerBoundsStatus = "allocation_base_metadata"
	RawPointerBoundsDerivedOffset               RawPointerBoundsStatus = "derived_allocation_offset"
	RawPointerBoundsCheckedExternalUnknown      RawPointerBoundsStatus = "checked_external_unknown"
	RawPointerBoundsRejectedNegativeOffset      RawPointerBoundsStatus = "rejected_negative_offset"
	RawPointerBoundsRejectedUpperBound          RawPointerBoundsStatus = "rejected_upper_bound"
	RawPointerBoundsRejectedAccessWidthOverflow RawPointerBoundsStatus = "rejected_access_width_overflow"
	RawPointerBoundsRejected                    RawPointerBoundsStatus = RawPointerBoundsRejectedUpperBound
)

type RawPointerDiagnosticCode string

const (
	RawPointerDiagnosticNegativePtrAdd       RawPointerDiagnosticCode = "negative_ptr_add_offset"
	RawPointerDiagnosticAllocationUpperBound RawPointerDiagnosticCode = "allocation_upper_bound"
	RawPointerDiagnosticAccessWidth          RawPointerDiagnosticCode = "access_width_exceeds_allocation"
)

type RawSliceBoundsStatus string

const (
	RawSliceBoundsVerifiedAllocationRoot RawSliceBoundsStatus = "verified_allocation_root"
	RawSliceBoundsExternalUnknown        RawSliceBoundsStatus = "external_unknown"
	RawSliceBoundsRejectedNegativeLength RawSliceBoundsStatus = "rejected_negative_length"
	RawSliceBoundsRejectedLengthOverflow RawSliceBoundsStatus = "rejected_length_overflow"
	RawSliceBoundsRejectedInvalidElement RawSliceBoundsStatus = "rejected_invalid_element_width"
)

type RawPointerBoundsABI struct {
	MetadataVersion      string                 `json:"metadata_version"`
	AllocationBaseStatus RawPointerBoundsStatus `json:"allocation_base_status"`
	DerivedOffsetStatus  RawPointerBoundsStatus `json:"derived_offset_status"`
	UnknownPointerStatus RawPointerBoundsStatus `json:"unknown_pointer_status"`
	RejectedNegative     RawPointerBoundsStatus `json:"rejected_negative_offset_status,omitempty"`
	RejectedUpperBound   RawPointerBoundsStatus `json:"rejected_upper_bound_status,omitempty"`
	RejectedAccessWidth  RawPointerBoundsStatus `json:"rejected_access_width_overflow_status,omitempty"`
	RawSliceUnknown      RawSliceBoundsStatus   `json:"raw_slice_unknown"`
	RawSliceVerifiedRoot RawSliceBoundsStatus   `json:"raw_slice_verified_root"`
}

type RawPointerBoundsMetadata struct {
	Status                 RawPointerBoundsStatus `json:"status"`
	BaseID                 string                 `json:"base_id,omitempty"`
	BaseBytes              int64                  `json:"base_bytes,omitempty"`
	OffsetBytes            int64                  `json:"offset_bytes,omitempty"`
	AccessWidthBytes       int64                  `json:"access_width_bytes,omitempty"`
	VerifiedAllocationRoot bool                   `json:"verified_allocation_root"`
	Reason                 string                 `json:"reason,omitempty"`
}

type RawPointerDiagnostic struct {
	Code    RawPointerDiagnosticCode `json:"code"`
	Message string                   `json:"message"`
}

type RawSliceBoundsMetadata struct {
	Status                 RawSliceBoundsStatus   `json:"status"`
	PointerStatus          RawPointerBoundsStatus `json:"pointer_status"`
	BaseID                 string                 `json:"base_id,omitempty"`
	BaseBytes              int64                  `json:"base_bytes,omitempty"`
	LengthBytes            int64                  `json:"length_bytes,omitempty"`
	VerifiedAllocationRoot bool                   `json:"verified_allocation_root"`
	Reason                 string                 `json:"reason,omitempty"`
}

const maxRawSliceByteLength = int64(1<<31 - 1)

func RuntimeRawPointerBoundsABI() RawPointerBoundsABI {
	return RawPointerBoundsABI{
		MetadataVersion:      "raw-pointer-bounds-v1",
		AllocationBaseStatus: RawPointerBoundsAllocationBase,
		DerivedOffsetStatus:  RawPointerBoundsDerivedOffset,
		UnknownPointerStatus: RawPointerBoundsCheckedExternalUnknown,
		RejectedNegative:     RawPointerBoundsRejectedNegativeOffset,
		RejectedUpperBound:   RawPointerBoundsRejectedUpperBound,
		RejectedAccessWidth:  RawPointerBoundsRejectedAccessWidthOverflow,
		RawSliceUnknown:      RawSliceBoundsExternalUnknown,
		RawSliceVerifiedRoot: RawSliceBoundsVerifiedAllocationRoot,
	}
}

func NewRawAllocationBounds(baseID string, byteSize int64) (RawPointerBoundsMetadata, error) {
	if baseID == "" {
		return RawPointerBoundsMetadata{}, fmt.Errorf("raw pointer bounds: base id is required")
	}
	if byteSize <= 0 {
		return RawPointerBoundsMetadata{}, fmt.Errorf(
			"raw pointer bounds: allocation byte size must be positive",
		)
	}
	return RawPointerBoundsMetadata{
		Status:                 RawPointerBoundsAllocationBase,
		BaseID:                 baseID,
		BaseBytes:              byteSize,
		OffsetBytes:            0,
		VerifiedAllocationRoot: true,
		Reason:                 "core.alloc_bytes allocation root",
	}, nil
}

func UnknownRawPointerBounds(reason string) RawPointerBoundsMetadata {
	if reason == "" {
		reason = "raw pointer source is external or unknown"
	}
	return RawPointerBoundsMetadata{
		Status: RawPointerBoundsCheckedExternalUnknown,
		Reason: reason,
	}
}

func DeriveRawPointerBounds(
	base RawPointerBoundsMetadata,
	offsetBytes int64,
	accessWidthBytes int64,
) (RawPointerBoundsMetadata, *RawPointerDiagnostic) {
	if !base.VerifiedAllocationRoot || base.BaseID == "" || base.BaseBytes <= 0 {
		out := UnknownRawPointerBounds("ptr_add source is not a verified allocation root")
		out.OffsetBytes = offsetBytes
		out.AccessWidthBytes = accessWidthBytes
		return out, nil
	}
	if accessWidthBytes <= 0 {
		accessWidthBytes = 1
	}
	if offsetBytes < 0 {
		return rejectedRawPointerBounds(
				base,
				offsetBytes,
				accessWidthBytes,
				RawPointerBoundsRejectedNegativeOffset,
			), &RawPointerDiagnostic{
				Code:    RawPointerDiagnosticNegativePtrAdd,
				Message: "negative ptr_add offset is impossible for allocation-base metadata",
			}
	}
	if offsetBytes >= base.BaseBytes {
		return rejectedRawPointerBounds(
				base,
				offsetBytes,
				accessWidthBytes,
				RawPointerBoundsRejectedUpperBound,
			), &RawPointerDiagnostic{
				Code:    RawPointerDiagnosticAllocationUpperBound,
				Message: "ptr_add offset reaches allocation upper bound",
			}
	}
	if accessWidthBytes > 0 && offsetBytes > math.MaxInt64-accessWidthBytes {
		return rejectedRawPointerBounds(
				base,
				offsetBytes,
				accessWidthBytes,
				RawPointerBoundsRejectedAccessWidthOverflow,
			), &RawPointerDiagnostic{
				Code:    RawPointerDiagnosticAccessWidth,
				Message: "raw access width overflows pointer offset",
			}
	}
	if offsetBytes+accessWidthBytes > base.BaseBytes {
		return rejectedRawPointerBounds(
				base,
				offsetBytes,
				accessWidthBytes,
				RawPointerBoundsRejectedAccessWidthOverflow,
			), &RawPointerDiagnostic{
				Code:    RawPointerDiagnosticAccessWidth,
				Message: "raw access width exceeds allocation",
			}
	}
	return RawPointerBoundsMetadata{
		Status:                 RawPointerBoundsDerivedOffset,
		BaseID:                 base.BaseID,
		BaseBytes:              base.BaseBytes,
		OffsetBytes:            offsetBytes,
		AccessWidthBytes:       accessWidthBytes,
		VerifiedAllocationRoot: true,
		Reason:                 "ptr_add derived from allocation-base metadata",
	}, nil
}

func RawSliceBoundsFromParts(
	ptr RawPointerBoundsMetadata,
	length int64,
	elemSize int64,
) RawSliceBoundsMetadata {
	if elemSize <= 0 {
		return RawSliceBoundsMetadata{
			Status:        RawSliceBoundsRejectedInvalidElement,
			PointerStatus: ptr.Status,
			BaseID:        ptr.BaseID,
			BaseBytes:     ptr.BaseBytes,
			Reason:        "raw slice element byte width must be positive",
		}
	}
	lengthBytes, lengthOK := checkedMulInt64(length, elemSize)
	if length < 0 {
		return RawSliceBoundsMetadata{
			Status:        RawSliceBoundsRejectedNegativeLength,
			PointerStatus: ptr.Status,
			BaseID:        ptr.BaseID,
			BaseBytes:     ptr.BaseBytes,
			LengthBytes:   lengthBytes,
			Reason:        "negative raw slice length is rejected before view construction",
		}
	}
	if !lengthOK {
		return RawSliceBoundsMetadata{
			Status:        RawSliceBoundsRejectedLengthOverflow,
			PointerStatus: ptr.Status,
			BaseID:        ptr.BaseID,
			BaseBytes:     ptr.BaseBytes,
			Reason:        "raw slice length byte computation overflows before view construction",
		}
	}
	if lengthBytes > maxRawSliceByteLength {
		return RawSliceBoundsMetadata{
			Status:        RawSliceBoundsRejectedLengthOverflow,
			PointerStatus: ptr.Status,
			BaseID:        ptr.BaseID,
			BaseBytes:     ptr.BaseBytes,
			LengthBytes:   lengthBytes,
			Reason:        "raw slice length byte computation exceeds signed 32-bit runtime view limit",
		}
	}
	endOffset, offsetOK := checkedAddInt64(ptr.OffsetBytes, lengthBytes)
	if lengthOK && offsetOK && rawPointerStatusAllowsSliceVerification(ptr.Status) &&
		ptr.VerifiedAllocationRoot && ptr.BaseID != "" && ptr.BaseBytes > 0 &&
		ptr.OffsetBytes >= 0 && lengthBytes >= 0 && endOffset <= ptr.BaseBytes {
		return RawSliceBoundsMetadata{
			Status:                 RawSliceBoundsVerifiedAllocationRoot,
			PointerStatus:          ptr.Status,
			BaseID:                 ptr.BaseID,
			BaseBytes:              ptr.BaseBytes,
			LengthBytes:            lengthBytes,
			VerifiedAllocationRoot: true,
			Reason:                 "raw slice constructed from verified allocation root",
		}
	}
	return RawSliceBoundsMetadata{
		Status:        RawSliceBoundsExternalUnknown,
		PointerStatus: ptr.Status,
		BaseID:        ptr.BaseID,
		BaseBytes:     ptr.BaseBytes,
		LengthBytes:   lengthBytes,
		Reason: ("raw slice remains external/unknown unless constructed from " +
			"verified allocation root"),
	}
}

func rejectedRawPointerBounds(
	base RawPointerBoundsMetadata,
	offsetBytes int64,
	accessWidthBytes int64,
	status RawPointerBoundsStatus,
) RawPointerBoundsMetadata {
	if status == "" {
		status = RawPointerBoundsRejected
	}
	return RawPointerBoundsMetadata{
		Status:                 status,
		BaseID:                 base.BaseID,
		BaseBytes:              base.BaseBytes,
		OffsetBytes:            offsetBytes,
		AccessWidthBytes:       accessWidthBytes,
		VerifiedAllocationRoot: false,
		Reason:                 "impossible ptr_add for allocation-base metadata",
	}
}

func rawPointerStatusAllowsSliceVerification(status RawPointerBoundsStatus) bool {
	switch status {
	case RawPointerBoundsAllocationBase, RawPointerBoundsDerivedOffset:
		return true
	default:
		return false
	}
}

func checkedAddInt64(left int64, right int64) (int64, bool) {
	if right > 0 && left > math.MaxInt64-right {
		return 0, false
	}
	if right < 0 && left < math.MinInt64-right {
		return 0, false
	}
	return left + right, true
}

func checkedMulInt64(left int64, right int64) (int64, bool) {
	if left == 0 || right == 0 {
		return 0, true
	}
	if left > 0 {
		if right > 0 {
			if left > math.MaxInt64/right {
				return 0, false
			}
		} else if right < math.MinInt64/left {
			return 0, false
		}
	} else {
		if right > 0 {
			if left < math.MinInt64/right {
				return 0, false
			}
		} else if left != 0 && right < math.MaxInt64/left {
			return 0, false
		}
	}
	return left * right, true
}
