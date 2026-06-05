package runtimeabi

const (
	RegionAllocatorAlignmentBytes int32 = 16
	RegionHeaderBytes             int32 = 16
	RegionDebugHeaderBytes        int32 = 4096
	MaxRegionMapBytes             int32 = 1<<31 - 1
)

type RegionAllocatorConfig struct {
	AlignmentBytes   int32 `json:"alignment_bytes"`
	HeaderBytes      int32 `json:"header_bytes"`
	DebugHeaderBytes int32 `json:"debug_header_bytes"`
	MaxPayloadBytes  int32 `json:"max_payload_bytes"`
}

func RuntimeRegionAllocatorConfig(debug bool) RegionAllocatorConfig {
	header := RegionHeaderBytes
	if debug {
		header = RegionDebugHeaderBytes
	}
	return RegionAllocatorConfig{
		AlignmentBytes:   RegionAllocatorAlignmentBytes,
		HeaderBytes:      header,
		DebugHeaderBytes: RegionDebugHeaderBytes,
		MaxPayloadBytes:  MaxRegionMapBytes - header,
	}
}

func AlignRegionBytes(bytes int64) (int64, bool) {
	if bytes < 0 || bytes >= int64(MaxRegionMapBytes) {
		return 0, false
	}
	aligned := (bytes + int64(RegionAllocatorAlignmentBytes-1)) & ^int64(RegionAllocatorAlignmentBytes-1)
	if aligned >= int64(MaxRegionMapBytes) {
		return 0, false
	}
	return aligned, true
}
