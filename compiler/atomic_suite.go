package compiler

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	ctarget "tetra_language/compiler/target"
)

type AtomicStressCheck struct {
	Name  string
	Error string
}

func RunTargetAtomicStressChecks(targetName string) ([]AtomicStressCheck, error) {
	tgt, err := ctarget.Parse(targetName)
	if err != nil {
		return nil, err
	}
	if tgt.Arch != ctarget.ArchX86 && tgt.Arch != ctarget.ArchX64 {
		return nil, fmt.Errorf("atomic stress suite for target %s requires an x86/x64 native target model", tgt.Triple)
	}
	prefix := atomicSuiteTargetPrefix(tgt)
	return runAtomicStressChecks([]struct {
		name string
		run  func() error
	}{
		{name: prefix + " atomic validation matrix", run: func() error { return checkAtomicValidationMatrix(tgt) }},
		{name: prefix + " atomic object matrix", run: func() error { return checkAtomicObjectMatrix(tgt) }},
		{name: prefix + " pointer atomic object width", run: func() error { return checkAtomicPointerObjectWidth(tgt) }},
		{name: prefix + " atomic concurrency stress oracle", run: func() error { return checkAtomicConcurrencyStressOracle(tgt) }},
		{name: prefix + " atomic diagnostics", run: func() error { return checkAtomicDiagnostics(tgt) }},
	}), nil
}

func runAtomicStressChecks(cases []struct {
	name string
	run  func() error
}) []AtomicStressCheck {
	out := make([]AtomicStressCheck, 0, len(cases))
	for _, tc := range cases {
		check := AtomicStressCheck{Name: tc.name}
		if err := tc.run(); err != nil {
			check.Error = err.Error()
		}
		out = append(out, check)
	}
	return out
}

func atomicSuiteTargetPrefix(tgt ctarget.Target) string {
	if tgt.Arch == ctarget.ArchX86 {
		return "x86"
	}
	if tgt.ABI == ctarget.ABIX32SysV {
		return "x32"
	}
	if tgt.Triple == "windows-x64" || tgt.Triple == "macos-x64" {
		return tgt.Triple
	}
	return "x64"
}

func checkAtomicValidationMatrix(tgt ctarget.Target) error {
	ptrAtomic, err := tgt.AtomicPointerLayout()
	if err != nil {
		return err
	}
	if ptrAtomic.WidthBits != tgt.PointerWidthBits || ptrAtomic.RegisterWidthBits != tgt.RegisterWidthBits || !ptrAtomic.PointerSized {
		return fmt.Errorf("%s pointer atomic layout = %#v, want pointer width %d and register width %d", tgt.Triple, ptrAtomic, tgt.PointerWidthBits, tgt.RegisterWidthBits)
	}
	allOrders := []ctarget.MemoryOrder{
		ctarget.MemoryOrderRelaxed,
		ctarget.MemoryOrderAcquire,
		ctarget.MemoryOrderRelease,
		ctarget.MemoryOrderAcqRel,
		ctarget.MemoryOrderSeqCst,
	}
	ops := []ctarget.AtomicOp{
		ctarget.AtomicLoad,
		ctarget.AtomicStore,
		ctarget.AtomicExchange,
		ctarget.AtomicCompareExchange,
		ctarget.AtomicCompareExchangeWeak,
		ctarget.AtomicFetchAdd,
		ctarget.AtomicFetchSub,
		ctarget.AtomicFetchAnd,
		ctarget.AtomicFetchOr,
		ctarget.AtomicFetchXor,
	}
	widths := tgt.AtomicWidthBits()
	if len(widths) == 0 {
		return fmt.Errorf("%s has no declared atomic widths", tgt.Triple)
	}
	for _, width := range widths {
		layout, err := tgt.AtomicLayout(width)
		if err != nil {
			return fmt.Errorf("%s rejected %d-bit atomic layout: %w", tgt.Triple, width, err)
		}
		for _, op := range ops {
			for _, order := range allOrders {
				err := tgt.ValidateAtomic(op, width, layout.AlignBytes, order)
				wantOK := atomicSuiteOrderAllowed(op, order)
				if wantOK && err != nil {
					return fmt.Errorf("%s rejected atomic %s/%d/%s: %w", tgt.Triple, op, width, order, err)
				}
				if !wantOK && err == nil {
					return fmt.Errorf("%s accepted invalid atomic %s/%d/%s", tgt.Triple, op, width, order)
				}
			}
		}
		if err := tgt.ValidateAtomic(ctarget.AtomicExchange, width, layout.AlignBytes-1, ctarget.MemoryOrderSeqCst); err == nil {
			return fmt.Errorf("%s accepted misaligned %d-bit atomic exchange", tgt.Triple, width)
		}
	}
	for _, order := range allOrders {
		if err := tgt.ValidateAtomic(ctarget.AtomicFence, 0, 0, order); err != nil {
			return fmt.Errorf("%s rejected atomic fence %s: %w", tgt.Triple, order, err)
		}
	}
	if err := tgt.ValidateAtomic(ctarget.AtomicFence, 0, 0, ctarget.MemoryOrderUnknown); err == nil {
		return fmt.Errorf("%s accepted atomic fence with unknown order", tgt.Triple)
	}
	if tgt.MaxAtomicWidthBits < 64 {
		if _, err := tgt.AtomicLayout(64); err == nil {
			return fmt.Errorf("%s accepted unsupported 64-bit atomic layout", tgt.Triple)
		}
	}
	if _, err := tgt.AtomicLayout(128); err == nil {
		return fmt.Errorf("%s accepted unsupported 128-bit atomic layout", tgt.Triple)
	}
	return nil
}

func atomicSuiteOrderAllowed(op ctarget.AtomicOp, order ctarget.MemoryOrder) bool {
	switch order {
	case ctarget.MemoryOrderRelaxed, ctarget.MemoryOrderAcquire, ctarget.MemoryOrderRelease, ctarget.MemoryOrderAcqRel, ctarget.MemoryOrderSeqCst:
	default:
		return false
	}
	switch op {
	case ctarget.AtomicLoad:
		return order == ctarget.MemoryOrderRelaxed || order == ctarget.MemoryOrderAcquire || order == ctarget.MemoryOrderSeqCst
	case ctarget.AtomicStore:
		return order == ctarget.MemoryOrderRelaxed || order == ctarget.MemoryOrderRelease || order == ctarget.MemoryOrderSeqCst
	case ctarget.AtomicExchange, ctarget.AtomicCompareExchange, ctarget.AtomicCompareExchangeWeak,
		ctarget.AtomicFetchAdd, ctarget.AtomicFetchSub, ctarget.AtomicFetchAnd, ctarget.AtomicFetchOr, ctarget.AtomicFetchXor:
		return true
	default:
		return false
	}
}

func checkAtomicConcurrencyStressOracle(tgt ctarget.Target) error {
	iters, err := atomicStressIterations()
	if err != nil {
		return err
	}
	checks := []struct {
		name string
		run  func() error
	}{
		{name: "contended CAS loop", run: func() error { return checkAtomicContendedCASLoop(tgt, iters) }},
		{name: "release/acquire message passing", run: func() error { return checkAtomicMessagePassing(tgt, iters) }},
		{name: "seq_cst ordering", run: func() error { return checkAtomicSeqCstOrdering(iters) }},
		{name: "ABA stamped pointer", run: func() error { return checkAtomicABAStampedPointer(tgt, iters) }},
		{name: "false sharing counters", run: func() error { return checkAtomicFalseSharingCounters(tgt, iters) }},
		{name: "weak CAS spurious retry", run: func() error { return checkAtomicWeakCASSpuriousRetry(tgt, iters) }},
		{name: "8/16-bit masked CAS loops", run: func() error { return checkAtomicNarrowMaskedCASLoops(iters) }},
	}
	for _, check := range checks {
		if err := check.run(); err != nil {
			return fmt.Errorf("%s %s: %w", tgt.Triple, check.name, err)
		}
	}
	return nil
}

func atomicStressIterations() (int, error) {
	raw := strings.TrimSpace(os.Getenv("TETRA_ATOMIC_STRESS_ITERS"))
	if raw == "" {
		return 128, nil
	}
	iters, err := strconv.Atoi(raw)
	if err != nil || iters <= 0 {
		return 0, fmt.Errorf("TETRA_ATOMIC_STRESS_ITERS must be a positive integer, got %q", raw)
	}
	if iters > 100000 {
		return 0, fmt.Errorf("TETRA_ATOMIC_STRESS_ITERS=%d is too high for the compiler-owned stress oracle; use <= 100000", iters)
	}
	return iters, nil
}

func checkAtomicContendedCASLoop(tgt ctarget.Target, iters int) error {
	const workers = 4
	if tgt.PointerWidthBits == 32 {
		var counter atomic.Uint32
		var wg sync.WaitGroup
		wg.Add(workers)
		for worker := 0; worker < workers; worker++ {
			go func(worker int) {
				defer wg.Done()
				for i := 0; i < iters; i++ {
					for {
						old := counter.Load()
						if counter.CompareAndSwap(old, old+1) {
							break
						}
						atomicStressYield(i, worker)
					}
					atomicStressYield(i, worker+17)
				}
			}(worker)
		}
		wg.Wait()
		want := uint32(workers * iters)
		if got := counter.Load(); got != want {
			return fmt.Errorf("32-bit pointer CAS counter = %d, want %d", got, want)
		}
		return nil
	}
	var counter atomic.Uint64
	var wg sync.WaitGroup
	wg.Add(workers)
	for worker := 0; worker < workers; worker++ {
		go func(worker int) {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				for {
					old := counter.Load()
					if counter.CompareAndSwap(old, old+1) {
						break
					}
					atomicStressYield(i, worker)
				}
				atomicStressYield(i, worker+17)
			}
		}(worker)
	}
	wg.Wait()
	want := uint64(workers * iters)
	if got := counter.Load(); got != want {
		return fmt.Errorf("64-bit pointer CAS counter = %d, want %d", got, want)
	}
	return nil
}

func checkAtomicMessagePassing(tgt ctarget.Target, iters int) error {
	if tgt.PointerWidthBits == 32 {
		var data atomic.Uint32
		var flag atomic.Uint32
		for i := 0; i < iters; i++ {
			payload := uint32(0x1000 + i)
			data.Store(0)
			flag.Store(0)
			errCh := make(chan error, 1)
			var wg sync.WaitGroup
			wg.Add(2)
			go func() {
				defer wg.Done()
				data.Store(payload)
				atomicStressYield(i, 31)
				flag.Store(1)
			}()
			go func() {
				defer wg.Done()
				for flag.Load() == 0 {
					atomicStressYield(i, 47)
				}
				if got := data.Load(); got != payload {
					errCh <- fmt.Errorf("32-bit payload = %d, want %d", got, payload)
				}
			}()
			wg.Wait()
			select {
			case err := <-errCh:
				return err
			default:
			}
		}
		return nil
	}
	var data atomic.Uint64
	var flag atomic.Uint64
	for i := 0; i < iters; i++ {
		payload := uint64(0x1_0000_0000) + uint64(i)
		data.Store(0)
		flag.Store(0)
		errCh := make(chan error, 1)
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			data.Store(payload)
			atomicStressYield(i, 31)
			flag.Store(1)
		}()
		go func() {
			defer wg.Done()
			for flag.Load() == 0 {
				atomicStressYield(i, 47)
			}
			if got := data.Load(); got != payload {
				errCh <- fmt.Errorf("64-bit payload = %d, want %d", got, payload)
			}
		}()
		wg.Wait()
		select {
		case err := <-errCh:
			return err
		default:
		}
	}
	return nil
}

func checkAtomicSeqCstOrdering(iters int) error {
	for i := 0; i < iters; i++ {
		var x atomic.Uint32
		var y atomic.Uint32
		var r1 atomic.Uint32
		var r2 atomic.Uint32
		start := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			x.Store(1)
			atomicStressYield(i, 61)
			r1.Store(y.Load())
		}()
		go func() {
			defer wg.Done()
			<-start
			y.Store(1)
			atomicStressYield(i, 73)
			r2.Store(x.Load())
		}()
		close(start)
		wg.Wait()
		if r1.Load() == 0 && r2.Load() == 0 {
			return fmt.Errorf("seq_cst store/load admitted both-zero result at iteration %d", i)
		}
	}
	return nil
}

func checkAtomicABAStampedPointer(tgt ctarget.Target, iters int) error {
	if tgt.PointerWidthBits == 32 {
		var cell atomic.Uint32
		for i := 0; i < iters; i++ {
			a1 := packABA32(0x1001, 1)
			b2 := packABA32(0x2002, 2)
			a3 := packABA32(0x1001, 3)
			c4 := packABA32(0x3003, 4)
			cell.Store(a1)
			if !cell.CompareAndSwap(a1, b2) || !cell.CompareAndSwap(b2, a3) {
				return fmt.Errorf("32-bit ABA setup failed at iteration %d", i)
			}
			if cell.CompareAndSwap(a1, c4) {
				return fmt.Errorf("32-bit stale ABA CAS succeeded at iteration %d", i)
			}
			if !cell.CompareAndSwap(a3, c4) {
				return fmt.Errorf("32-bit fresh ABA CAS failed at iteration %d", i)
			}
			atomicStressYield(i, 89)
		}
		return nil
	}
	var cell atomic.Uint64
	for i := 0; i < iters; i++ {
		a1 := packABA64(0x10000001, 1)
		b2 := packABA64(0x20000002, 2)
		a3 := packABA64(0x10000001, 3)
		c4 := packABA64(0x30000003, 4)
		cell.Store(a1)
		if !cell.CompareAndSwap(a1, b2) || !cell.CompareAndSwap(b2, a3) {
			return fmt.Errorf("64-bit ABA setup failed at iteration %d", i)
		}
		if cell.CompareAndSwap(a1, c4) {
			return fmt.Errorf("64-bit stale ABA CAS succeeded at iteration %d", i)
		}
		if !cell.CompareAndSwap(a3, c4) {
			return fmt.Errorf("64-bit fresh ABA CAS failed at iteration %d", i)
		}
		atomicStressYield(i, 89)
	}
	return nil
}

func checkAtomicFalseSharingCounters(tgt ctarget.Target, iters int) error {
	const workers = 2
	if tgt.PointerWidthBits == 32 {
		var counters struct {
			left  atomic.Uint32
			right atomic.Uint32
		}
		var wg sync.WaitGroup
		wg.Add(workers)
		go func() {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				counters.left.Add(1)
				atomicStressYield(i, 101)
			}
		}()
		go func() {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				counters.right.Add(1)
				atomicStressYield(i, 103)
			}
		}()
		wg.Wait()
		if got := counters.left.Load(); got != uint32(iters) {
			return fmt.Errorf("32-bit left false-sharing counter = %d, want %d", got, iters)
		}
		if got := counters.right.Load(); got != uint32(iters) {
			return fmt.Errorf("32-bit right false-sharing counter = %d, want %d", got, iters)
		}
		return nil
	}
	var counters struct {
		left  atomic.Uint64
		right atomic.Uint64
	}
	var wg sync.WaitGroup
	wg.Add(workers)
	go func() {
		defer wg.Done()
		for i := 0; i < iters; i++ {
			counters.left.Add(1)
			atomicStressYield(i, 101)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < iters; i++ {
			counters.right.Add(1)
			atomicStressYield(i, 103)
		}
	}()
	wg.Wait()
	if got := counters.left.Load(); got != uint64(iters) {
		return fmt.Errorf("64-bit left false-sharing counter = %d, want %d", got, iters)
	}
	if got := counters.right.Load(); got != uint64(iters) {
		return fmt.Errorf("64-bit right false-sharing counter = %d, want %d", got, iters)
	}
	return nil
}

func checkAtomicWeakCASSpuriousRetry(tgt ctarget.Target, iters int) error {
	if tgt.PointerWidthBits == 32 {
		var cell atomic.Uint32
		for i := 0; i < iters; i++ {
			old := uint32(i)
			next := old + 1
			cell.Store(old)
			attempts := 0
			for {
				attempts++
				if weakCAS32WithSpuriousFailure(&cell, old, next, attempts) {
					break
				}
				if got := cell.Load(); got != old {
					return fmt.Errorf("32-bit weak CAS changed value after spurious failure: got %d want %d", got, old)
				}
				atomicStressYield(i, attempts)
			}
			if attempts < 2 {
				return fmt.Errorf("32-bit weak CAS retry did not exercise a spurious failure")
			}
			if got := cell.Load(); got != next {
				return fmt.Errorf("32-bit weak CAS result = %d, want %d", got, next)
			}
		}
		return nil
	}
	var cell atomic.Uint64
	for i := 0; i < iters; i++ {
		old := uint64(i)
		next := old + 1
		cell.Store(old)
		attempts := 0
		for {
			attempts++
			if weakCAS64WithSpuriousFailure(&cell, old, next, attempts) {
				break
			}
			if got := cell.Load(); got != old {
				return fmt.Errorf("64-bit weak CAS changed value after spurious failure: got %d want %d", got, old)
			}
			atomicStressYield(i, attempts)
		}
		if attempts < 2 {
			return fmt.Errorf("64-bit weak CAS retry did not exercise a spurious failure")
		}
		if got := cell.Load(); got != next {
			return fmt.Errorf("64-bit weak CAS result = %d, want %d", got, next)
		}
	}
	return nil
}

func checkAtomicNarrowMaskedCASLoops(iters int) error {
	var byteCell atomic.Uint32
	var wordCell atomic.Uint32
	for i := 0; i < iters; i++ {
		byteCell.Store(uint32(i) & 0xff)
		wordCell.Store(uint32(i) & 0xffff)
		oldByte, newByte := atomicMaskedFetchXor(&byteCell, 0x5a, 0xff, i, 113)
		if newByte != ((oldByte ^ 0x5a) & 0xff) {
			return fmt.Errorf("u8 masked xor = %#x from old %#x", newByte, oldByte)
		}
		oldWord, newWord := atomicMaskedFetchAdd(&wordCell, 257, 0xffff, i, 127)
		if newWord != ((oldWord + 257) & 0xffff) {
			return fmt.Errorf("u16 masked add = %#x from old %#x", newWord, oldWord)
		}
	}
	return nil
}

func atomicMaskedFetchXor(cell *atomic.Uint32, operand uint32, mask uint32, iter int, salt int) (uint32, uint32) {
	for {
		old := cell.Load() & mask
		next := (old ^ operand) & mask
		if cell.CompareAndSwap(old, next) {
			return old, next
		}
		atomicStressYield(iter, salt)
	}
}

func atomicMaskedFetchAdd(cell *atomic.Uint32, operand uint32, mask uint32, iter int, salt int) (uint32, uint32) {
	for {
		old := cell.Load() & mask
		next := (old + operand) & mask
		if cell.CompareAndSwap(old, next) {
			return old, next
		}
		atomicStressYield(iter, salt)
	}
}

func weakCAS32WithSpuriousFailure(cell *atomic.Uint32, old uint32, next uint32, attempt int) bool {
	if attempt%3 == 1 {
		return false
	}
	return cell.CompareAndSwap(old, next)
}

func weakCAS64WithSpuriousFailure(cell *atomic.Uint64, old uint64, next uint64, attempt int) bool {
	if attempt%3 == 1 {
		return false
	}
	return cell.CompareAndSwap(old, next)
}

func packABA32(ptr uint32, stamp uint32) uint32 {
	return ((stamp & 0xffff) << 16) | (ptr & 0xffff)
}

func packABA64(ptr uint64, stamp uint64) uint64 {
	return ((stamp & 0xffffffff) << 32) | (ptr & 0xffffffff)
}

func atomicStressYield(iter int, salt int) {
	x := uint32(iter*1103515245 + salt*12345 + 0x9e3779b9)
	if x&7 == 0 {
		runtime.Gosched()
	}
}

func checkAtomicObjectMatrix(tgt ctarget.Target) error {
	tmpDir, err := os.MkdirTemp("", "tetra-atomic-suite-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	srcPath := filepath.Join(tmpDir, "atomic_matrix.tetra")
	outPath := filepath.Join(tmpDir, "atomic_matrix.tobj")
	source := atomicMatrixSource
	if tgt.Arch == ctarget.ArchX86 {
		source = atomicMatrixSourceX86
	}
	if err := os.WriteFile(srcPath, []byte(source), 0o644); err != nil {
		return err
	}
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, tgt.Triple, BuildOptions{Emit: EmitLibrary}); err != nil {
		return err
	}
	obj, err := ReadObject(outPath)
	if err != nil {
		return err
	}
	if obj.Target != tgt.Triple {
		return fmt.Errorf("target mismatch: got %q want %s", obj.Target, tgt.Triple)
	}
	if !objectHasSymbol(obj, "atomic_matrix") {
		return fmt.Errorf("object missing atomic_matrix symbol: %#v", obj.Symbols)
	}
	required := []struct {
		name  string
		bytes []byte
	}{
		{name: "i64 qword CAS", bytes: []byte{0xF0, 0x4C, 0x0F, 0xB1, 0x07}},
		{name: "i64 qword XADD", bytes: []byte{0xF0, 0x4C, 0x0F, 0xC1, 0x07}},
		{name: "i32 dword CAS", bytes: []byte{0xF0, 0x44, 0x0F, 0xB1, 0x07}},
		{name: "i32 dword XADD", bytes: []byte{0xF0, 0x44, 0x0F, 0xC1, 0x07}},
		{name: "u8 byte exchange", bytes: []byte{0x44, 0x86, 0x07}},
		{name: "seq_cst fence", bytes: []byte{0x0F, 0xAE, 0xF0}},
	}
	if tgt.Arch == ctarget.ArchX86 {
		required = []struct {
			name  string
			bytes []byte
		}{
			{name: "i32 dword CAS", bytes: []byte{0xF0, 0x0F, 0xB1, 0x17}},
			{name: "i32 dword XADD", bytes: []byte{0xF0, 0x0F, 0xC1, 0x0F}},
			{name: "u8 byte exchange", bytes: []byte{0x86, 0x0F}},
			{name: "u16 word exchange", bytes: []byte{0x66, 0x87, 0x0F}},
			{name: "u8 byte fetch-and CAS loop", bytes: []byte{0x20, 0xCA, 0xF0, 0x0F, 0xB0, 0x17}},
			{name: "u8 byte fetch-or CAS loop", bytes: []byte{0x08, 0xCA, 0xF0, 0x0F, 0xB0, 0x17}},
			{name: "u8 byte fetch-xor CAS loop", bytes: []byte{0x30, 0xCA, 0xF0, 0x0F, 0xB0, 0x17}},
			{name: "u16 word fetch-and CAS loop", bytes: []byte{0x66, 0x21, 0xCA, 0x66, 0xF0, 0x0F, 0xB1, 0x17}},
			{name: "u16 word fetch-or CAS loop", bytes: []byte{0x66, 0x09, 0xCA, 0x66, 0xF0, 0x0F, 0xB1, 0x17}},
			{name: "u16 word fetch-xor CAS loop", bytes: []byte{0x66, 0x31, 0xCA, 0x66, 0xF0, 0x0F, 0xB1, 0x17}},
			{name: "seq_cst fence", bytes: []byte{0xF0, 0x83, 0x0C, 0x24, 0x00}},
		}
	}
	for _, want := range required {
		if !bytes.Contains(obj.Code, want.bytes) {
			return fmt.Errorf("missing %s bytes % x in %s atomic object", want.name, want.bytes, tgt.Triple)
		}
	}
	return nil
}

func checkAtomicPointerObjectWidth(tgt ctarget.Target) error {
	tmpDir, err := os.MkdirTemp("", "tetra-atomic-ptr-width-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	srcPath := filepath.Join(tmpDir, "atomic_ptr_width.tetra")
	outPath := filepath.Join(tmpDir, "atomic_ptr_width.tobj")
	if err := os.WriteFile(srcPath, []byte(atomicPointerWidthSource), 0o644); err != nil {
		return err
	}
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, tgt.Triple, BuildOptions{Emit: EmitLibrary, Jobs: 1}); err != nil {
		return err
	}
	obj, err := ReadObject(outPath)
	if err != nil {
		return err
	}
	if obj.Target != tgt.Triple {
		return fmt.Errorf("target mismatch: got %q want %s", obj.Target, tgt.Triple)
	}
	if !objectHasSymbol(obj, "atomic_ptr_width") {
		return fmt.Errorf("object missing atomic_ptr_width symbol: %#v", obj.Symbols)
	}
	if tgt.Arch == ctarget.ArchX86 {
		return requireAtomicPointerWidthBytes(obj.Code, tgt.Triple,
			[][]byte{
				{0x87, 0x0F},
				{0xF0, 0x0F, 0xB1, 0x17},
				{0xF0, 0x0F, 0xC1, 0x0F},
				{0xF0, 0x0F, 0xB1, 0x1F},
			},
			nil,
		)
	}
	dwordPatterns := [][]byte{
		{0x45, 0x89, 0xC1},
		{0x44, 0x89, 0xC8},
		{0x44, 0x87, 0x07},
		{0xF0, 0x44, 0x0F, 0xB1, 0x07},
		{0xF0, 0x44, 0x0F, 0xC1, 0x07},
		{0xF0, 0x44, 0x0F, 0xB1, 0x17},
	}
	qwordPatterns := [][]byte{
		{0x4D, 0x89, 0xC1},
		{0x4C, 0x89, 0xC8},
		{0x4C, 0x87, 0x07},
		{0xF0, 0x4C, 0x0F, 0xB1, 0x07},
		{0xF0, 0x4C, 0x0F, 0xC1, 0x07},
		{0xF0, 0x4C, 0x0F, 0xB1, 0x17},
	}
	if tgt.PointerWidthBits == 32 {
		return requireAtomicPointerWidthBytes(obj.Code, tgt.Triple, dwordPatterns, qwordPatterns)
	}
	return requireAtomicPointerWidthBytes(obj.Code, tgt.Triple, qwordPatterns, dwordPatterns)
}

func requireAtomicPointerWidthBytes(code []byte, target string, required [][]byte, forbidden [][]byte) error {
	for _, pattern := range required {
		if !bytes.Contains(code, pattern) {
			return fmt.Errorf("%s pointer atomic object missing required width bytes % x", target, pattern)
		}
	}
	for _, pattern := range forbidden {
		if bytes.Contains(code, pattern) {
			return fmt.Errorf("%s pointer atomic object contains forbidden opposite-width bytes % x", target, pattern)
		}
	}
	return nil
}

const atomicPointerWidthSource = `
func atomic_ptr_width() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(64)
        let p2: ptr = p
        let loaded: ptr = core.atomic_load_ptr_acquire(p, mem)
        var ignored_store: ptr = core.atomic_store_ptr_release(p, loaded, mem)
        let exchanged: ptr = core.atomic_exchange_ptr_seq_cst(p, loaded, mem)
        let cas: ptr = core.atomic_compare_exchange_ptr_acq_rel(p, loaded, exchanged, mem)
        let weak: ptr = core.atomic_compare_exchange_weak_ptr_seq_cst(p, cas, exchanged, mem)
        let add: ptr = core.atomic_fetch_add_ptr_relaxed(p, p2, mem)
        let sub: ptr = core.atomic_fetch_sub_ptr_seq_cst(p, p2, mem)
        let anded: ptr = core.atomic_fetch_and_ptr_acquire(p, p2, mem)
        let ored: ptr = core.atomic_fetch_or_ptr_release(p, p2, mem)
        let xored: ptr = core.atomic_fetch_xor_ptr_acq_rel(p, p2, mem)
        var fence_seq_cst: i32 = core.atomic_fence_seq_cst(mem)
        return 0
    return 0
`

const atomicMatrixSource = `
func atomic_matrix() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(64)
        let p2: ptr = p
        let byte: u8 = 1
        let word: u16 = 2
        let old_byte: u8 = core.atomic_exchange_u8_seq_cst(p, byte, mem)
        let old_word: u16 = core.atomic_exchange_u16_seq_cst(p, word, mem)
        let and_byte: u8 = core.atomic_fetch_and_u8_acquire(p, byte, mem)
        let or_byte: u8 = core.atomic_fetch_or_u8_release(p, byte, mem)
        let xor_byte: u8 = core.atomic_fetch_xor_u8_acq_rel(p, byte, mem)
        let and_word: u16 = core.atomic_fetch_and_u16_acquire(p, word, mem)
        let or_word: u16 = core.atomic_fetch_or_u16_release(p, word, mem)
        let xor_word: u16 = core.atomic_fetch_xor_u16_acq_rel(p, word, mem)
        let loaded: i32 = core.atomic_load_i32_acquire(p, mem)
        var ignored_store: i32 = core.atomic_store_i32_release(p, loaded, mem)
        let exchanged: i32 = core.atomic_exchange_i32_seq_cst(p, loaded, mem)
        let cas: i32 = core.atomic_compare_exchange_i32_acq_rel(p, loaded, exchanged, mem)
        let weak: i32 = core.atomic_compare_exchange_weak_i32_seq_cst(p, cas, exchanged, mem)
        let add: i32 = core.atomic_fetch_add_i32_relaxed(p, 3, mem)
        let sub: i32 = core.atomic_fetch_sub_i32_seq_cst(p, 1, mem)
        let anded: i32 = core.atomic_fetch_and_i32_acquire(p, 7, mem)
        let ored: i32 = core.atomic_fetch_or_i32_release(p, 8, mem)
        let xored: i32 = core.atomic_fetch_xor_i32_acq_rel(p, 9, mem)
        let loaded_ptr: ptr = core.atomic_load_ptr_acquire(p, mem)
        var ignored_ptr_store: ptr = core.atomic_store_ptr_release(p, loaded_ptr, mem)
        let exchanged_ptr: ptr = core.atomic_exchange_ptr_seq_cst(p, loaded_ptr, mem)
        let cas_ptr: ptr = core.atomic_compare_exchange_ptr_acq_rel(p, loaded_ptr, exchanged_ptr, mem)
        let add_ptr: ptr = core.atomic_fetch_add_ptr_relaxed(p, p2, mem)
        let loaded64: i64 = core.atomic_load_i64_acquire(p, mem)
        var ignored64_store: i64 = core.atomic_store_i64_release(p, loaded64, mem)
        let exchanged64: i64 = core.atomic_exchange_i64_seq_cst(p, loaded64, mem)
        let cas64: i64 = core.atomic_compare_exchange_i64_acq_rel(p, loaded64, exchanged64, mem)
        let weak64: i64 = core.atomic_compare_exchange_weak_i64_seq_cst(p, cas64, exchanged64, mem)
        let add64: i64 = core.atomic_fetch_add_i64_relaxed(p, loaded64, mem)
        var fence_relaxed: i32 = core.atomic_fence_relaxed(mem)
        var fence_acquire: i32 = core.atomic_fence_acquire(mem)
        var fence_release: i32 = core.atomic_fence_release(mem)
        var fence_acq_rel: i32 = core.atomic_fence_acq_rel(mem)
        var fence_seq_cst: i32 = core.atomic_fence_seq_cst(mem)
        return loaded + exchanged + cas + weak + add + sub + anded + ored + xored
    return 0
`

const atomicMatrixSourceX86 = `
func atomic_matrix() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(64)
        let p2: ptr = p
        let byte: u8 = 1
        let word: u16 = 2
        let old_byte: u8 = core.atomic_exchange_u8_seq_cst(p, byte, mem)
        let old_word: u16 = core.atomic_exchange_u16_seq_cst(p, word, mem)
        let and_byte: u8 = core.atomic_fetch_and_u8_acquire(p, byte, mem)
        let or_byte: u8 = core.atomic_fetch_or_u8_release(p, byte, mem)
        let xor_byte: u8 = core.atomic_fetch_xor_u8_acq_rel(p, byte, mem)
        let and_word: u16 = core.atomic_fetch_and_u16_acquire(p, word, mem)
        let or_word: u16 = core.atomic_fetch_or_u16_release(p, word, mem)
        let xor_word: u16 = core.atomic_fetch_xor_u16_acq_rel(p, word, mem)
        let loaded: i32 = core.atomic_load_i32_acquire(p, mem)
        var ignored_store: i32 = core.atomic_store_i32_release(p, loaded, mem)
        let exchanged: i32 = core.atomic_exchange_i32_seq_cst(p, loaded, mem)
        let cas: i32 = core.atomic_compare_exchange_i32_acq_rel(p, loaded, exchanged, mem)
        let weak: i32 = core.atomic_compare_exchange_weak_i32_seq_cst(p, cas, exchanged, mem)
        let add: i32 = core.atomic_fetch_add_i32_relaxed(p, 3, mem)
        let sub: i32 = core.atomic_fetch_sub_i32_seq_cst(p, 1, mem)
        let anded: i32 = core.atomic_fetch_and_i32_acquire(p, 7, mem)
        let ored: i32 = core.atomic_fetch_or_i32_release(p, 8, mem)
        let xored: i32 = core.atomic_fetch_xor_i32_acq_rel(p, 9, mem)
        let loaded_ptr: ptr = core.atomic_load_ptr_acquire(p, mem)
        var ignored_ptr_store: ptr = core.atomic_store_ptr_release(p, loaded_ptr, mem)
        let exchanged_ptr: ptr = core.atomic_exchange_ptr_seq_cst(p, loaded_ptr, mem)
        let cas_ptr: ptr = core.atomic_compare_exchange_ptr_acq_rel(p, loaded_ptr, exchanged_ptr, mem)
        let add_ptr: ptr = core.atomic_fetch_add_ptr_relaxed(p, p2, mem)
        var fence_relaxed: i32 = core.atomic_fence_relaxed(mem)
        var fence_acquire: i32 = core.atomic_fence_acquire(mem)
        var fence_release: i32 = core.atomic_fence_release(mem)
        var fence_acq_rel: i32 = core.atomic_fence_acq_rel(mem)
        var fence_seq_cst: i32 = core.atomic_fence_seq_cst(mem)
        return loaded + exchanged + cas + weak + add + sub + anded + ored + xored
    return 0
`

func checkAtomicDiagnostics(tgt ctarget.Target) error {
	tests := []struct {
		name string
		call string
		want string
	}{
		{name: "load release", call: "core.atomic_load_i32_release(p, mem)", want: "atomic load does not support memory order release"},
		{name: "store acquire", call: "core.atomic_store_i32_acquire(p, 1, mem)", want: "atomic store does not support memory order acquire"},
		{name: "unknown order", call: "core.atomic_fetch_add_i32_consume(p, 1, mem)", want: "unsupported atomic memory order 'consume'"},
		{name: "unknown op", call: "core.atomic_nand_i32_relaxed(p, 1, mem)", want: "unsupported atomic operation 'nand'"},
	}
	for _, tt := range tests {
		src := `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(4)
        return ` + tt.call + `
    return 0
`
		prog, err := Parse([]byte(src))
		if err != nil {
			return fmt.Errorf("%s parse: %w", tt.name, err)
		}
		_, err = Check(prog)
		if err == nil {
			return fmt.Errorf("%s accepted invalid atomic builtin %s", tt.name, tt.call)
		}
		if !strings.Contains(err.Error(), tt.want) {
			return fmt.Errorf("%s diagnostic = %q, want %q", tt.name, err.Error(), tt.want)
		}
	}
	if tgt.MaxAtomicWidthBits < 64 {
		if err := checkAtomicUnsupportedWidthDiagnostic(tgt); err != nil {
			return err
		}
	}
	return nil
}

func checkAtomicUnsupportedWidthDiagnostic(tgt ctarget.Target) error {
	tmpDir, err := os.MkdirTemp("", "tetra-atomic-diagnostics-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	srcPath := filepath.Join(tmpDir, "atomic_i64_unsupported.tetra")
	outPath := filepath.Join(tmpDir, "atomic_i64_unsupported.tobj")
	src := `
func atomic_i64_unsupported() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(8)
        let loaded: i64 = core.atomic_load_i64_acquire(p, mem)
        return 0
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	_, err = BuildFileWithStatsOpt(srcPath, outPath, tgt.Triple, BuildOptions{Emit: EmitLibrary, Jobs: 1})
	if err == nil {
		return fmt.Errorf("%s accepted unsupported 64-bit atomic source", tgt.Triple)
	}
	diag := DiagnosticFromError(err)
	if diag.Code != DiagnosticCodeTargetRuntime || diag.Severity != "error" {
		return fmt.Errorf("%s unsupported-width diagnostic identity = %#v", tgt.Triple, diag)
	}
	for _, want := range []string{tgt.Triple, "atomic load", "64-bit", "unsupported atomic width 64 bits"} {
		if !strings.Contains(diag.Message, want) {
			return fmt.Errorf("%s unsupported-width diagnostic = %q, want %q", tgt.Triple, diag.Message, want)
		}
	}
	if _, statErr := os.Stat(outPath); !os.IsNotExist(statErr) {
		return fmt.Errorf("%s unsupported-width rejection wrote object %s, stat error = %v", tgt.Triple, outPath, statErr)
	}
	return nil
}

func objectHasSymbol(obj *Object, name string) bool {
	if obj == nil {
		return false
	}
	for _, sym := range obj.Symbols {
		if strings.EqualFold(sym.Name, name) || sym.Name == name {
			return true
		}
	}
	return false
}
