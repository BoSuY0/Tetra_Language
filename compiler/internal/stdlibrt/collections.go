package stdlibrt

import (
	"bytes"
	"errors"
	"fmt"
)

type StorageClass string

const (
	StorageHeap     StorageClass = "heap"
	StorageRegion   StorageClass = "region"
	StorageBorrowed StorageClass = "borrowed"
)

var (
	ErrNegativeCapacity = errors.New("negative collection capacity")
	ErrRegionClosed     = errors.New("region is closed")
	ErrRegionCapacity   = errors.New("region capacity exceeded")
	ErrViewOutOfRange   = errors.New("view out of range")
)

type Region struct {
	id     string
	data   []byte
	offset int
	closed bool
}

func NewRegion(id string, capacity int) *Region {
	if id == "" {
		id = "region"
	}
	if capacity < 0 {
		capacity = 0
	}
	return &Region{id: id, data: make([]byte, capacity)}
}

func (r *Region) ID() string {
	if r == nil {
		return ""
	}
	return r.id
}

func (r *Region) Alloc(size int) ([]byte, error) {
	if size < 0 {
		return nil, ErrNegativeCapacity
	}
	if r == nil {
		return make([]byte, size), nil
	}
	if r.closed {
		return nil, ErrRegionClosed
	}
	if len(r.data)-r.offset < size {
		return nil, ErrRegionCapacity
	}
	start := r.offset
	r.offset += size
	return r.data[start:r.offset], nil
}

func (r *Region) Used() int {
	if r == nil {
		return 0
	}
	return r.offset
}

func (r *Region) Reset() error {
	if r == nil {
		return nil
	}
	if r.closed {
		return ErrRegionClosed
	}
	r.offset = 0
	return nil
}

type BytesView struct {
	Bytes      []byte
	Storage    StorageClass
	RegionID   string
	Provenance string
	Copied     bool
}

type CollectionKind string

const (
	VecCollection           CollectionKind = "Vec"
	StringBuilderCollection CollectionKind = "StringBuilder"
	HashMapCollection       CollectionKind = "HashMap"
	ByteBufferCollection    CollectionKind = "ByteBuffer"
	ArenaBufferCollection   CollectionKind = "ArenaBuffer"
	RingBufferCollection    CollectionKind = "RingBuffer"
)

type CollectionSpec struct {
	Kind     CollectionKind
	Element  string
	Capacity int
	Region   *Region
}

type CollectionPlan struct {
	Kind          CollectionKind
	Element       string
	Capacity      int
	Storage       StorageClass
	RegionID      string
	HiddenHeap    bool
	BytesReserved int
	Provenance    string
}

func PlanCollection(spec CollectionSpec) (CollectionPlan, error) {
	if spec.Capacity < 0 {
		return CollectionPlan{}, ErrNegativeCapacity
	}
	if spec.Kind == "" {
		return CollectionPlan{}, errors.New("collection kind is required")
	}
	storage := StorageHeap
	hiddenHeap := true
	regionID := ""
	if spec.Region != nil {
		storage = StorageRegion
		hiddenHeap = false
		regionID = spec.Region.ID()
	}
	element := spec.Element
	if element == "" {
		element = "unknown"
	}
	return CollectionPlan{
		Kind:          spec.Kind,
		Element:       element,
		Capacity:      spec.Capacity,
		Storage:       storage,
		RegionID:      regionID,
		HiddenHeap:    hiddenHeap,
		BytesReserved: estimatedBytes(spec.Kind, spec.Capacity),
		Provenance:    fmt.Sprintf("%s<%s>:%s:%d", spec.Kind, element, storage, spec.Capacity),
	}, nil
}

func estimatedBytes(kind CollectionKind, capacity int) int {
	if capacity <= 0 {
		return 0
	}
	switch kind {
	case HashMapCollection:
		return capacity * 32
	case StringBuilderCollection, ByteBufferCollection, ArenaBufferCollection, RingBufferCollection:
		return capacity
	default:
		return capacity * 8
	}
}

type StorageReport struct {
	Component      string
	Storage        StorageClass
	RegionID       string
	HiddenHeap     bool
	BytesReserved  int
	BytesUsed      int
	CopyOperations int
	BytesCopied    int
	BorrowedViews  int
	Provenance     string
}

type ByteBuffer struct {
	data   []byte
	used   int
	report StorageReport
}

func NewByteBuffer(capacity int, region *Region) (*ByteBuffer, error) {
	plan, err := PlanCollection(CollectionSpec{
		Kind:     ByteBufferCollection,
		Element:  "u8",
		Capacity: capacity,
		Region:   region,
	})
	if err != nil {
		return nil, err
	}
	var data []byte
	if region != nil {
		data, err = region.Alloc(capacity)
		if err != nil {
			return nil, err
		}
	} else {
		data = make([]byte, capacity)
	}
	return &ByteBuffer{
		data: data,
		report: StorageReport{
			Component:     string(ByteBufferCollection),
			Storage:       plan.Storage,
			RegionID:      plan.RegionID,
			HiddenHeap:    plan.HiddenHeap,
			BytesReserved: plan.BytesReserved,
			Provenance:    plan.Provenance,
		},
	}, nil
}

func (b *ByteBuffer) Append(src []byte) (int, error) {
	if b == nil || len(b.data)-b.used < len(src) {
		return 0, ErrRegionCapacity
	}
	n := copy(b.data[b.used:], src)
	b.used += n
	b.report.BytesUsed = b.used
	if n > 0 {
		b.report.CopyOperations++
		b.report.BytesCopied += n
	}
	return n, nil
}

func (b *ByteBuffer) View(start int, length int) (BytesView, error) {
	if b == nil || start < 0 || length < 0 || start+length < start || start+length > b.used {
		return BytesView{}, ErrViewOutOfRange
	}
	b.report.BorrowedViews++
	return BytesView{
		Bytes:      b.data[start : start+length],
		Storage:    b.report.Storage,
		RegionID:   b.report.RegionID,
		Provenance: b.report.Provenance,
	}, nil
}

func (b *ByteBuffer) Report() StorageReport {
	if b == nil {
		return StorageReport{Storage: StorageHeap, HiddenHeap: true}
	}
	report := b.report
	report.BytesUsed = b.used
	return report
}

type StringBuilder struct {
	data   []byte
	used   int
	report StorageReport
}

func NewStringBuilder(capacity int, region *Region) (*StringBuilder, error) {
	data, report, err := allocateCollectionBytes(StringBuilderCollection, "u8", capacity, region)
	if err != nil {
		return nil, err
	}
	return &StringBuilder{data: data, report: report}, nil
}

func (b *StringBuilder) Append(src []byte) (int, error) {
	if b == nil || len(b.data)-b.used < len(src) {
		return 0, ErrRegionCapacity
	}
	n := copy(b.data[b.used:], src)
	b.used += n
	b.report.BytesUsed = b.used
	if n > 0 {
		b.report.CopyOperations++
		b.report.BytesCopied += n
	}
	return n, nil
}

func (b *StringBuilder) AppendString(src string) (int, error) {
	if b == nil || len(b.data)-b.used < len(src) {
		return 0, ErrRegionCapacity
	}
	n := copy(b.data[b.used:], src)
	b.used += n
	b.report.BytesUsed = b.used
	if n > 0 {
		b.report.CopyOperations++
		b.report.BytesCopied += n
	}
	return n, nil
}

func (b *StringBuilder) View() (BytesView, error) {
	if b == nil {
		return BytesView{}, ErrViewOutOfRange
	}
	b.report.BorrowedViews++
	return BytesView{
		Bytes:      b.data[:b.used],
		Storage:    b.report.Storage,
		RegionID:   b.report.RegionID,
		Provenance: b.report.Provenance,
	}, nil
}

func (b *StringBuilder) Report() StorageReport {
	if b == nil {
		return StorageReport{
			Component:  string(StringBuilderCollection),
			Storage:    StorageHeap,
			HiddenHeap: true,
		}
	}
	report := b.report
	report.BytesUsed = b.used
	return report
}

type VecBytes struct {
	data   []byte
	used   int
	report StorageReport
}

func NewVecBytes(capacity int, region *Region) (*VecBytes, error) {
	data, report, err := allocateCollectionBytes(VecCollection, "u8", capacity, region)
	if err != nil {
		return nil, err
	}
	return &VecBytes{data: data, report: report}, nil
}

func (v *VecBytes) Push(value byte) error {
	if v == nil || len(v.data)-v.used < 1 {
		return ErrRegionCapacity
	}
	v.data[v.used] = value
	v.used++
	v.report.BytesUsed = v.used
	return nil
}

func (v *VecBytes) AppendBorrowed(src []byte) (int, error) {
	if v == nil || len(v.data)-v.used < len(src) {
		return 0, ErrRegionCapacity
	}
	n := copy(v.data[v.used:], src)
	v.used += n
	v.report.BytesUsed = v.used
	if n > 0 {
		v.report.CopyOperations++
		v.report.BytesCopied += n
	}
	return n, nil
}

func (v *VecBytes) View() (BytesView, error) {
	if v == nil {
		return BytesView{}, ErrViewOutOfRange
	}
	v.report.BorrowedViews++
	return BytesView{
		Bytes:      v.data[:v.used],
		Storage:    v.report.Storage,
		RegionID:   v.report.RegionID,
		Provenance: v.report.Provenance,
	}, nil
}

func (v *VecBytes) Report() StorageReport {
	if v == nil {
		return StorageReport{
			Component:  string(VecCollection),
			Storage:    StorageHeap,
			HiddenHeap: true,
		}
	}
	report := v.report
	report.BytesUsed = v.used
	return report
}

type HashMapBytesOptions struct {
	Slots         int
	BytesCapacity int
	Region        *Region
}

type HashMapBytes struct {
	slots     []byte
	arena     []byte
	arenaUsed int
	count     int
	report    StorageReport
}

const (
	hashMapSlotSize     = 21
	hashMapSlotState    = 0
	hashMapSlotHash     = 1
	hashMapSlotKeyOff   = 5
	hashMapSlotKeyLen   = 9
	hashMapSlotValOff   = 13
	hashMapSlotValLen   = 17
	hashMapSlotOccupied = byte(1)
)

func NewHashMapBytes(opt HashMapBytesOptions) (*HashMapBytes, error) {
	if opt.Slots < 0 || opt.BytesCapacity < 0 {
		return nil, ErrNegativeCapacity
	}
	if opt.Slots == 0 {
		return nil, ErrRegionCapacity
	}
	plan, err := PlanCollection(CollectionSpec{
		Kind:     HashMapCollection,
		Element:  "[]u8",
		Capacity: opt.Slots,
		Region:   opt.Region,
	})
	if err != nil {
		return nil, err
	}
	slotBytes := opt.Slots * hashMapSlotSize
	var slots []byte
	var arena []byte
	if opt.Region != nil {
		slots, err = opt.Region.Alloc(slotBytes)
		if err != nil {
			return nil, err
		}
		arena, err = opt.Region.Alloc(opt.BytesCapacity)
		if err != nil {
			return nil, err
		}
		zeroBytes(slots)
		zeroBytes(arena)
	} else {
		slots = make([]byte, slotBytes)
		arena = make([]byte, opt.BytesCapacity)
	}
	report := StorageReport{
		Component:     string(HashMapCollection),
		Storage:       plan.Storage,
		RegionID:      plan.RegionID,
		HiddenHeap:    plan.HiddenHeap,
		BytesReserved: slotBytes + opt.BytesCapacity,
		Provenance:    plan.Provenance,
	}
	return &HashMapBytes{slots: slots, arena: arena, report: report}, nil
}

func (h *HashMapBytes) Put(key []byte, value []byte) error {
	if h == nil || len(h.slots) == 0 {
		return ErrRegionCapacity
	}
	hash := hashBytes(key)
	slotCount := len(h.slots) / hashMapSlotSize
	start := int(hash % uint32(slotCount))
	for probe := 0; probe < slotCount; probe++ {
		index := (start + probe) % slotCount
		slot := h.slot(index)
		if slot[hashMapSlotState] == 0 {
			keyOff, valueOff, err := h.copyKeyValue(key, value)
			if err != nil {
				return err
			}
			slot[hashMapSlotState] = hashMapSlotOccupied
			writeU32(slot, hashMapSlotHash, hash)
			writeU32(slot, hashMapSlotKeyOff, uint32(keyOff))
			writeU32(slot, hashMapSlotKeyLen, uint32(len(key)))
			writeU32(slot, hashMapSlotValOff, uint32(valueOff))
			writeU32(slot, hashMapSlotValLen, uint32(len(value)))
			h.count++
			h.report.BytesUsed = h.arenaUsed
			return nil
		}
		if readU32(slot, hashMapSlotHash) == hash && bytes.Equal(h.keyForSlot(slot), key) {
			valueOff, err := h.copyValue(value)
			if err != nil {
				return err
			}
			writeU32(slot, hashMapSlotValOff, uint32(valueOff))
			writeU32(slot, hashMapSlotValLen, uint32(len(value)))
			h.report.BytesUsed = h.arenaUsed
			return nil
		}
	}
	return ErrRegionCapacity
}

func (h *HashMapBytes) Get(key []byte) (BytesView, bool) {
	if h == nil || len(h.slots) == 0 {
		return BytesView{}, false
	}
	hash := hashBytes(key)
	slotCount := len(h.slots) / hashMapSlotSize
	start := int(hash % uint32(slotCount))
	for probe := 0; probe < slotCount; probe++ {
		index := (start + probe) % slotCount
		slot := h.slot(index)
		if slot[hashMapSlotState] == 0 {
			return BytesView{}, false
		}
		if readU32(slot, hashMapSlotHash) == hash && bytes.Equal(h.keyForSlot(slot), key) {
			off := int(readU32(slot, hashMapSlotValOff))
			length := int(readU32(slot, hashMapSlotValLen))
			if off < 0 || length < 0 || off+length > len(h.arena) {
				return BytesView{}, false
			}
			h.report.BorrowedViews++
			return BytesView{
				Bytes:      h.arena[off : off+length],
				Storage:    h.report.Storage,
				RegionID:   h.report.RegionID,
				Provenance: h.report.Provenance,
			}, true
		}
	}
	return BytesView{}, false
}

func (h *HashMapBytes) Report() StorageReport {
	if h == nil {
		return StorageReport{
			Component:  string(HashMapCollection),
			Storage:    StorageHeap,
			HiddenHeap: true,
		}
	}
	report := h.report
	report.BytesUsed = h.arenaUsed
	return report
}

func (h *HashMapBytes) slot(index int) []byte {
	start := index * hashMapSlotSize
	return h.slots[start : start+hashMapSlotSize]
}

func (h *HashMapBytes) keyForSlot(slot []byte) []byte {
	off := int(readU32(slot, hashMapSlotKeyOff))
	length := int(readU32(slot, hashMapSlotKeyLen))
	if off < 0 || length < 0 || off+length > len(h.arena) {
		return nil
	}
	return h.arena[off : off+length]
}

func (h *HashMapBytes) copyKeyValue(key []byte, value []byte) (int, int, error) {
	keyOff, err := h.copyBytes(key)
	if err != nil {
		return 0, 0, err
	}
	valueOff, err := h.copyBytes(value)
	if err != nil {
		return 0, 0, err
	}
	return keyOff, valueOff, nil
}

func (h *HashMapBytes) copyValue(value []byte) (int, error) {
	return h.copyBytes(value)
}

func (h *HashMapBytes) copyBytes(src []byte) (int, error) {
	if len(h.arena)-h.arenaUsed < len(src) {
		return 0, ErrRegionCapacity
	}
	off := h.arenaUsed
	n := copy(h.arena[off:], src)
	h.arenaUsed += n
	if n > 0 {
		h.report.CopyOperations++
		h.report.BytesCopied += n
	}
	return off, nil
}

type RingBuffer struct {
	data     []byte
	snapshot []byte
	head     int
	used     int
	report   StorageReport
}

func NewRingBuffer(capacity int, region *Region) (*RingBuffer, error) {
	if capacity < 0 {
		return nil, ErrNegativeCapacity
	}
	plan, err := PlanCollection(CollectionSpec{
		Kind:     RingBufferCollection,
		Element:  "u8",
		Capacity: capacity,
		Region:   region,
	})
	if err != nil {
		return nil, err
	}
	var data []byte
	var snapshot []byte
	if region != nil {
		data, err = region.Alloc(capacity)
		if err != nil {
			return nil, err
		}
		snapshot, err = region.Alloc(capacity)
		if err != nil {
			return nil, err
		}
	} else {
		data = make([]byte, capacity)
		snapshot = make([]byte, capacity)
	}
	report := StorageReport{
		Component:     string(RingBufferCollection),
		Storage:       plan.Storage,
		RegionID:      plan.RegionID,
		HiddenHeap:    plan.HiddenHeap,
		BytesReserved: capacity * 2,
		Provenance:    plan.Provenance,
	}
	return &RingBuffer{data: data, snapshot: snapshot, report: report}, nil
}

func (r *RingBuffer) Write(src []byte) (int, error) {
	if r == nil || len(r.data)-r.used < len(src) {
		return 0, ErrRegionCapacity
	}
	if len(r.data) == 0 && len(src) > 0 {
		return 0, ErrRegionCapacity
	}
	for _, b := range src {
		tail := (r.head + r.used) % len(r.data)
		r.data[tail] = b
		r.used++
	}
	r.report.BytesUsed = r.used
	return len(src), nil
}

func (r *RingBuffer) PeekView(length int) (BytesView, error) {
	if r == nil || length < 0 || length > r.used {
		return BytesView{}, ErrViewOutOfRange
	}
	if length == 0 {
		return BytesView{
			Bytes:      r.data[:0],
			Storage:    r.report.Storage,
			RegionID:   r.report.RegionID,
			Provenance: r.report.Provenance,
		}, nil
	}
	contiguous := len(r.data) - r.head
	if contiguous > r.used {
		contiguous = r.used
	}
	if length <= contiguous {
		r.report.BorrowedViews++
		return BytesView{
			Bytes:      r.data[r.head : r.head+length],
			Storage:    r.report.Storage,
			RegionID:   r.report.RegionID,
			Provenance: r.report.Provenance,
		}, nil
	}
	if len(r.snapshot) < length {
		return BytesView{}, ErrRegionCapacity
	}
	n := copy(r.snapshot, r.data[r.head:])
	copy(r.snapshot[n:], r.data[:length-n])
	r.report.CopyOperations++
	r.report.BytesCopied += length
	return BytesView{
		Bytes:      r.snapshot[:length],
		Storage:    r.report.Storage,
		RegionID:   r.report.RegionID,
		Provenance: r.report.Provenance,
		Copied:     true,
	}, nil
}

func (r *RingBuffer) Consume(length int) error {
	if r == nil || length < 0 || length > r.used {
		return ErrViewOutOfRange
	}
	if len(r.data) == 0 {
		return nil
	}
	r.head = (r.head + length) % len(r.data)
	r.used -= length
	if r.used == 0 {
		r.head = 0
	}
	r.report.BytesUsed = r.used
	return nil
}

func (r *RingBuffer) Report() StorageReport {
	if r == nil {
		return StorageReport{
			Component:  string(RingBufferCollection),
			Storage:    StorageHeap,
			HiddenHeap: true,
		}
	}
	report := r.report
	report.BytesUsed = r.used
	return report
}

func allocateCollectionBytes(
	kind CollectionKind,
	element string,
	capacity int,
	region *Region,
) ([]byte, StorageReport, error) {
	plan, err := PlanCollection(CollectionSpec{
		Kind:     kind,
		Element:  element,
		Capacity: capacity,
		Region:   region,
	})
	if err != nil {
		return nil, StorageReport{}, err
	}
	var data []byte
	if region != nil {
		data, err = region.Alloc(capacity)
		if err != nil {
			return nil, StorageReport{}, err
		}
	} else {
		data = make([]byte, capacity)
	}
	return data, StorageReport{
		Component:     string(kind),
		Storage:       plan.Storage,
		RegionID:      plan.RegionID,
		HiddenHeap:    plan.HiddenHeap,
		BytesReserved: plan.BytesReserved,
		Provenance:    plan.Provenance,
	}, nil
}

func zeroBytes(data []byte) {
	for i := range data {
		data[i] = 0
	}
}

func hashBytes(data []byte) uint32 {
	hash := uint32(2166136261)
	for _, b := range data {
		hash ^= uint32(b)
		hash *= 16777619
	}
	if hash == 0 {
		return 1
	}
	return hash
}

func readU32(data []byte, off int) uint32 {
	return uint32(data[off]) |
		uint32(data[off+1])<<8 |
		uint32(data[off+2])<<16 |
		uint32(data[off+3])<<24
}

func writeU32(data []byte, off int, value uint32) {
	data[off] = byte(value)
	data[off+1] = byte(value >> 8)
	data[off+2] = byte(value >> 16)
	data[off+3] = byte(value >> 24)
}
