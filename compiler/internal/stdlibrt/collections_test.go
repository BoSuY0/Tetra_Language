package stdlibrt

import (
	"strings"
	"testing"
)

func TestRegionAwareCollectionPlansAvoidHiddenHeapWhenRegionKnown(t *testing.T) {
	region := NewRegion("request-region", 4096)
	kinds := []CollectionKind{
		VecCollection,
		StringBuilderCollection,
		HashMapCollection,
		ByteBufferCollection,
		ArenaBufferCollection,
	}

	for _, kind := range kinds {
		plan, err := PlanCollection(CollectionSpec{
			Kind:     kind,
			Element:  "u8",
			Capacity: 32,
			Region:   region,
		})
		if err != nil {
			t.Fatalf("PlanCollection(%s): %v", kind, err)
		}
		if plan.Storage != StorageRegion {
			t.Fatalf("%s storage = %q, want region", kind, plan.Storage)
		}
		if plan.HiddenHeap {
			t.Fatalf("%s unexpectedly reports hidden heap for known region", kind)
		}
		if plan.RegionID != "request-region" {
			t.Fatalf("%s region id = %q, want request-region", kind, plan.RegionID)
		}
		if plan.Provenance == "" {
			t.Fatalf("%s missing safe-view provenance", kind)
		}
	}
}

func TestRegionAwareCollectionPlansReportHeapFallbackWhenRegionUnknown(t *testing.T) {
	plan, err := PlanCollection(CollectionSpec{
		Kind:     ByteBufferCollection,
		Element:  "u8",
		Capacity: 16,
	})
	if err != nil {
		t.Fatalf("PlanCollection heap fallback: %v", err)
	}
	if plan.Storage != StorageHeap || !plan.HiddenHeap || plan.RegionID != "" {
		t.Fatalf("heap fallback plan = %#v", plan)
	}
}

func TestByteBufferViewsPreserveRegionProvenance(t *testing.T) {
	region := NewRegion("json-region", 64)
	buf, err := NewByteBuffer(16, region)
	if err != nil {
		t.Fatalf("NewByteBuffer: %v", err)
	}
	if _, err := buf.Append([]byte("abcdef")); err != nil {
		t.Fatalf("Append: %v", err)
	}
	view, err := buf.View(1, 3)
	if err != nil {
		t.Fatalf("View: %v", err)
	}
	if string(view.Bytes) != "bcd" {
		t.Fatalf("view bytes = %q, want bcd", view.Bytes)
	}
	if view.Storage != StorageRegion || view.RegionID != "json-region" {
		t.Fatalf("view storage = %#v", view)
	}
	if view.Provenance != buf.Report().Provenance {
		t.Fatalf("view provenance = %q, want %q", view.Provenance, buf.Report().Provenance)
	}
}

func TestRegionResetRestoresRequestLifetimeCapacity(t *testing.T) {
	region := NewRegion("request-region", 16)
	if first, err := region.Alloc(12); err != nil {
		t.Fatalf("first region alloc: %v", err)
	} else if len(first) != 12 {
		t.Fatalf("first region alloc len = %d, want 12", len(first))
	}
	if region.Used() != 12 {
		t.Fatalf("region used before reset = %d, want 12", region.Used())
	}
	if err := region.Reset(); err != nil {
		t.Fatalf("region reset: %v", err)
	}
	if region.Used() != 0 {
		t.Fatalf("region used after reset = %d, want 0", region.Used())
	}
	if second, err := region.Alloc(16); err != nil {
		t.Fatalf("second region alloc after reset: %v", err)
	} else if len(second) != 16 {
		t.Fatalf("second region alloc len = %d, want 16", len(second))
	}
}

func TestRegionAwareStdlibHelpersUseRegionReportsAndBorrowedViews(t *testing.T) {
	region := NewRegion("stdlib-region", 4096)

	builder, err := NewStringBuilder(64, region)
	if err != nil {
		t.Fatalf("NewStringBuilder: %v", err)
	}
	if _, err := builder.AppendString("hello"); err != nil {
		t.Fatalf("StringBuilder.AppendString: %v", err)
	}
	if _, err := builder.Append([]byte(" world")); err != nil {
		t.Fatalf("StringBuilder.Append: %v", err)
	}
	builderView, err := builder.View()
	if err != nil {
		t.Fatalf("StringBuilder.View: %v", err)
	}
	if string(builderView.Bytes) != "hello world" || builderView.Storage != StorageRegion ||
		builderView.RegionID != "stdlib-region" {
		t.Fatalf("builder view = %#v", builderView)
	}
	if report := builder.Report(); report.Component != "StringBuilder" || report.HiddenHeap ||
		report.BytesUsed != len("hello world") {
		t.Fatalf("builder report = %#v", report)
	}

	vec, err := NewVecBytes(8, region)
	if err != nil {
		t.Fatalf("NewVecBytes: %v", err)
	}
	if err := vec.Push('a'); err != nil {
		t.Fatalf("VecBytes.Push: %v", err)
	}
	if _, err := vec.AppendBorrowed([]byte("bc")); err != nil {
		t.Fatalf("VecBytes.AppendBorrowed: %v", err)
	}
	vecView, err := vec.View()
	if err != nil {
		t.Fatalf("VecBytes.View: %v", err)
	}
	if string(vecView.Bytes) != "abc" || vecView.Storage != StorageRegion ||
		vecView.RegionID != "stdlib-region" {
		t.Fatalf("vec view = %#v", vecView)
	}
	if report := vec.Report(); report.Component != "Vec" || report.HiddenHeap ||
		report.CopyOperations != 1 ||
		report.BytesCopied != 2 {
		t.Fatalf("vec report = %#v", report)
	}

	hash, err := NewHashMapBytes(HashMapBytesOptions{Slots: 8, BytesCapacity: 128, Region: region})
	if err != nil {
		t.Fatalf("NewHashMapBytes: %v", err)
	}
	if err := hash.Put([]byte("message"), []byte("ok")); err != nil {
		t.Fatalf("HashMapBytes.Put: %v", err)
	}
	hashView, ok := hash.Get([]byte("message"))
	if !ok {
		t.Fatalf("HashMapBytes.Get missing key")
	}
	if string(hashView.Bytes) != "ok" || hashView.Storage != StorageRegion ||
		hashView.RegionID != "stdlib-region" ||
		hashView.Copied {
		t.Fatalf("hash map value view = %#v", hashView)
	}
	if report := hash.Report(); report.Component != "HashMap" || report.HiddenHeap ||
		report.CopyOperations != 2 ||
		report.BytesCopied != len("message")+len("ok") {
		t.Fatalf("hash map report = %#v", report)
	}

	ring, err := NewRingBuffer(8, region)
	if err != nil {
		t.Fatalf("NewRingBuffer: %v", err)
	}
	if _, err := ring.Write([]byte("abcdef")); err != nil {
		t.Fatalf("RingBuffer.Write initial: %v", err)
	}
	borrowed, err := ring.PeekView(3)
	if err != nil {
		t.Fatalf("RingBuffer.PeekView borrowed: %v", err)
	}
	if string(borrowed.Bytes) != "abc" || borrowed.Copied || borrowed.Storage != StorageRegion {
		t.Fatalf("borrowed ring view = %#v", borrowed)
	}
	if err := ring.Consume(5); err != nil {
		t.Fatalf("RingBuffer.Consume: %v", err)
	}
	if _, err := ring.Write([]byte("XYZ")); err != nil {
		t.Fatalf("RingBuffer.Write wrapped: %v", err)
	}
	copied, err := ring.PeekView(4)
	if err != nil {
		t.Fatalf("RingBuffer.PeekView wrapped: %v", err)
	}
	if string(copied.Bytes) != "fXYZ" || !copied.Copied || copied.Storage != StorageRegion ||
		copied.RegionID != "stdlib-region" {
		t.Fatalf("wrapped ring view = %#v", copied)
	}
	if report := ring.Report(); report.Component != "RingBuffer" || report.HiddenHeap ||
		report.CopyOperations != 1 ||
		report.BytesCopied != 4 {
		t.Fatalf("ring report = %#v", report)
	}
}

func TestRegionAwareStdlibCoverageCoversP19PlanList(t *testing.T) {
	report, err := RegionAwareStdlibCoverage()
	if err != nil {
		t.Fatalf("RegionAwareStdlibCoverage: %v", err)
	}
	if err := ValidateRegionAwareStdlibCoverage(report); err != nil {
		t.Fatalf("ValidateRegionAwareStdlibCoverage failed: %v", err)
	}
	if report.SchemaVersion != "tetra.stdlib.region_aware.v1" {
		t.Fatalf("schema = %q", report.SchemaVersion)
	}
	if report.FullProductionWebStackClaimed || report.OfficialTechEmpowerResultClaimed ||
		report.ProductionPostgreSQLStackClaimed {
		t.Fatalf("P19.0 must not promote production web/db claims: %#v", report)
	}
	for _, want := range []string{
		"full production web stack is not claimed",
		"official TechEmpower result is not claimed",
		"production PostgreSQL stack is not claimed",
		"generic collection API is not claimed",
	} {
		if !hasStdlibCoverageText(report.NonClaims, want) {
			t.Fatalf("non-claims missing %q: %#v", want, report.NonClaims)
		}
	}

	byID := map[RegionAwareStdlibEvidenceID]RegionAwareStdlibEvidenceRow{}
	for _, row := range report.Rows {
		byID[row.ID] = row
		if row.Evidence == "" || row.Boundary == "" {
			t.Fatalf("row %q missing evidence or boundary: %#v", row.ID, row)
		}
		if row.HiddenHeapInHotPath {
			t.Fatalf("row %q claims hidden heap in hot path: %#v", row.ID, row)
		}
	}
	expected := []RegionAwareStdlibEvidenceID{
		RegionStdlibStringBuilder,
		RegionStdlibVecArray,
		RegionStdlibHashMap,
		RegionStdlibJSONParserBuilder,
		RegionStdlibHTTPParserBuilder,
		RegionStdlibPostgreSQLProtocol,
		RegionStdlibBuffers,
		RegionStdlibRingBuffers,
		RegionStdlibBorrowedViews,
		RegionStdlibCopyReports,
		RegionStdlibNoHiddenHeapReports,
		RegionStdlibProductionBoundaries,
	}
	for _, id := range expected {
		if _, ok := byID[id]; !ok {
			t.Fatalf("missing P19.0 row %q", id)
		}
	}

	requireStdlibCoverageFacts(
		t,
		byID[RegionStdlibStringBuilder],
		"NewStringBuilder",
		"StorageRegion",
		"borrowed BytesView",
	)
	requireStdlibCoverageFacts(
		t,
		byID[RegionStdlibVecArray],
		"NewVecBytes",
		"Vec/Array equivalent",
		"CopyOperations",
	)
	requireStdlibCoverageFacts(
		t,
		byID[RegionStdlibHashMap],
		"NewHashMapBytes",
		"open addressing",
		"BytesCopied",
	)
	requireStdlibCoverageFacts(
		t,
		byID[RegionStdlibJSONParserBuilder],
		"ParseValueView",
		"AppendValue",
		"BorrowedStrings",
		"CopiedStrings",
	)
	requireStdlibCoverageFacts(
		t,
		byID[RegionStdlibHTTPParserBuilder],
		"ParseRequestViewInRegion",
		"AppendResponseWithReport",
		"HeaderViewsBorrowed",
	)
	requireStdlibCoverageFacts(
		t,
		byID[RegionStdlibPostgreSQLProtocol],
		"AppendBindFormat",
		"DecodeDataRowBorrowed",
		"RowDecodeReport",
	)
	requireStdlibCoverageFacts(t, byID[RegionStdlibBuffers], "NewByteBuffer", "StorageReport")
	requireStdlibCoverageFacts(
		t,
		byID[RegionStdlibRingBuffers],
		"NewRingBuffer",
		"wrapped readable window",
		"Copied",
	)
	requireStdlibCoverageFacts(
		t,
		byID[RegionStdlibBorrowedViews],
		"BytesView",
		"StorageBorrowed",
		"StorageRegion",
	)
	requireStdlibCoverageFacts(
		t,
		byID[RegionStdlibCopyReports],
		"copy only when needed",
		"CopyOperations",
		"BytesCopied",
	)
	requireStdlibCoverageFacts(
		t,
		byID[RegionStdlibNoHiddenHeapReports],
		"HiddenHeap=false",
		"HeapAllocations=0",
	)
	requireStdlibCoverageFacts(
		t,
		byID[RegionStdlibProductionBoundaries],
		"no full production web stack",
		"no official TechEmpower",
		"no production PostgreSQL",
	)
}

func TestRegionAwareStdlibCoverageRejectsFakeClaims(t *testing.T) {
	report, err := RegionAwareStdlibCoverage()
	if err != nil {
		t.Fatalf("RegionAwareStdlibCoverage: %v", err)
	}
	if err := ValidateRegionAwareStdlibCoverage(report); err != nil {
		t.Fatalf("baseline report should validate: %v", err)
	}

	fakeWeb := cloneStdlibCoverage(report)
	fakeWeb.FullProductionWebStackClaimed = true
	if err := ValidateRegionAwareStdlibCoverage(fakeWeb); err == nil ||
		!strings.Contains(err.Error(), "full production web stack") {
		t.Fatalf("fake web-stack claim error = %v", err)
	}

	fakeTechEmpower := cloneStdlibCoverage(report)
	fakeTechEmpower.OfficialTechEmpowerResultClaimed = true
	if err := ValidateRegionAwareStdlibCoverage(fakeTechEmpower); err == nil ||
		!strings.Contains(err.Error(), "TechEmpower") {
		t.Fatalf("fake TechEmpower claim error = %v", err)
	}

	fakePostgres := cloneStdlibCoverage(report)
	fakePostgres.ProductionPostgreSQLStackClaimed = true
	if err := ValidateRegionAwareStdlibCoverage(fakePostgres); err == nil ||
		!strings.Contains(err.Error(), "PostgreSQL") {
		t.Fatalf("fake PostgreSQL claim error = %v", err)
	}

	hiddenHeap := cloneStdlibCoverage(report)
	for i := range hiddenHeap.Rows {
		if hiddenHeap.Rows[i].ID == RegionStdlibHashMap {
			hiddenHeap.Rows[i].HiddenHeapInHotPath = true
		}
	}
	if err := ValidateRegionAwareStdlibCoverage(hiddenHeap); err == nil ||
		!strings.Contains(err.Error(), "hidden heap") {
		t.Fatalf("hidden heap claim error = %v", err)
	}

	missingBorrowedViews := cloneStdlibCoverage(report)
	for i := range missingBorrowedViews.Rows {
		if missingBorrowedViews.Rows[i].ID == RegionStdlibBorrowedViews {
			missingBorrowedViews.Rows[i].RequiredFacts = []string{"BytesView only"}
		}
	}
	if err := ValidateRegionAwareStdlibCoverage(missingBorrowedViews); err == nil ||
		!strings.Contains(err.Error(), "borrowed") {
		t.Fatalf("missing borrowed-view facts error = %v", err)
	}

	noNonClaims := cloneStdlibCoverage(report)
	noNonClaims.NonClaims = nil
	if err := ValidateRegionAwareStdlibCoverage(noNonClaims); err == nil ||
		!strings.Contains(err.Error(), "non-claim") {
		t.Fatalf("missing non-claim error = %v", err)
	}
}

func TestStableGenericCollectionsCoverageCoversP19PlanList(t *testing.T) {
	report, err := StableGenericCollectionsCoverage()
	if err != nil {
		t.Fatalf("StableGenericCollectionsCoverage: %v", err)
	}
	if err := ValidateStableGenericCollectionsCoverage(report); err != nil {
		t.Fatalf("ValidateStableGenericCollectionsCoverage failed: %v", err)
	}
	if report.SchemaVersion != "tetra.stdlib.generic_collections.v1" {
		t.Fatalf("schema = %q", report.SchemaVersion)
	}
	if report.CPlusPlusRustParityClaimed || report.BroadProductionStdlibClaimed ||
		report.HiddenRuntimeAllocatorClaimed {
		t.Fatalf("P19.1 foundation must not promote broad/parity/allocator claims: %#v", report)
	}
	for _, want := range []string{
		"C++/Rust parity is not claimed",
		"broad production stdlib is not claimed",
		"collection storage allocation remains caller-owned",
		"P19.0 byte-oriented runtime helpers remain evidence helpers",
	} {
		if !hasStdlibCoverageText(report.NonClaims, want) {
			t.Fatalf("non-claims missing %q: %#v", want, report.NonClaims)
		}
	}

	byID := map[StableGenericCollectionsEvidenceID]StableGenericCollectionsEvidenceRow{}
	for _, row := range report.Rows {
		byID[row.ID] = row
		if row.Status == "" || row.Evidence == "" || row.Boundary == "" {
			t.Fatalf("row %q missing status/evidence/boundary: %#v", row.ID, row)
		}
	}
	expected := []StableGenericCollectionsEvidenceID{
		StableGenericCollectionsTetraSourceAPI,
		StableGenericCollectionsValueRepresentation,
		StableGenericCollectionsMonomorphizedOperations,
		StableGenericCollectionsCommonSpecializations,
		StableGenericCollectionsAllocationReports,
		StableGenericCollectionsBenchmarkGate,
	}
	for _, id := range expected {
		if _, ok := byID[id]; !ok {
			t.Fatalf("missing P19.1 row %q", id)
		}
	}

	requireStableGenericCollectionsFacts(
		t,
		byID[StableGenericCollectionsTetraSourceAPI],
		"lib.core.collections.Vec<T>",
		"HashMap<K,V>",
		"caller-owned slices",
	)
	requireStableGenericCollectionsFacts(
		t,
		byID[StableGenericCollectionsValueRepresentation],
		"genericTypeName",
		"mangleGenericName",
		"[]T",
	)
	requireStableGenericCollectionsFacts(
		t,
		byID[StableGenericCollectionsMonomorphizedOperations],
		"vec_from_slice<T>",
		"hash_map_from_slices<K,V>",
		"concrete before lowering",
	)
	requireStableGenericCollectionsFacts(
		t,
		byID[StableGenericCollectionsCommonSpecializations],
		"hash_map_get_i32_i32_or",
		"hash_map_get_u8_i32_or",
	)
	requireStableGenericCollectionsFacts(
		t,
		byID[StableGenericCollectionsAllocationReports],
		"core.make_*",
		"allocation-plan reports",
		"no internal allocation",
	)
	benchmarkGate := byID[StableGenericCollectionsBenchmarkGate]
	if benchmarkGate.Status != StableGenericCollectionsEvidenceOnly {
		t.Fatalf(
			"benchmark gate status = %q, want checked evidence-only artifact",
			benchmarkGate.Status,
		)
	}
	if len(benchmarkGate.MissingFacts) != 0 {
		t.Fatalf("benchmark gate still has missing facts: %#v", benchmarkGate.MissingFacts)
	}
	requireStableGenericCollectionsFacts(
		t,
		benchmarkGate,
		"truth-bench-harness",
		"p19.1_generic_collections",
		"reports/stable-generic-collections-v1/benchmarks/generic-collections-hash-table-manifest.json",
		"reports/stable-generic-collections-v1/benchmarks/generic-collections-hash-table-report.json",
		"Tetra/C++/Rust equivalents",
		"tetra",
		"cpp",
		"rust",
		"allocation/proof/bounds",
		"no parity claim",
	)
}

func TestStableGenericCollectionsCoverageRejectsFakeClaims(t *testing.T) {
	report, err := StableGenericCollectionsCoverage()
	if err != nil {
		t.Fatalf("StableGenericCollectionsCoverage: %v", err)
	}
	if err := ValidateStableGenericCollectionsCoverage(report); err != nil {
		t.Fatalf("baseline report should validate: %v", err)
	}

	fakeParity := cloneStableGenericCollectionsCoverage(report)
	fakeParity.CPlusPlusRustParityClaimed = true
	if err := ValidateStableGenericCollectionsCoverage(fakeParity); err == nil ||
		!strings.Contains(err.Error(), "C++/Rust parity") {
		t.Fatalf("fake parity claim error = %v", err)
	}

	fakeProduction := cloneStableGenericCollectionsCoverage(report)
	fakeProduction.BroadProductionStdlibClaimed = true
	if err := ValidateStableGenericCollectionsCoverage(fakeProduction); err == nil ||
		!strings.Contains(err.Error(), "broad production stdlib") {
		t.Fatalf("fake production claim error = %v", err)
	}

	hiddenAllocator := cloneStableGenericCollectionsCoverage(report)
	hiddenAllocator.HiddenRuntimeAllocatorClaimed = true
	if err := ValidateStableGenericCollectionsCoverage(hiddenAllocator); err == nil ||
		!strings.Contains(err.Error(), "hidden runtime allocator") {
		t.Fatalf("hidden allocator claim error = %v", err)
	}

	missingAllocationReports := cloneStableGenericCollectionsCoverage(report)
	for i := range missingAllocationReports.Rows {
		if missingAllocationReports.Rows[i].ID == StableGenericCollectionsAllocationReports {
			missingAllocationReports.Rows[i].RequiredFacts = []string{"core.make_* only"}
		}
	}
	if err := ValidateStableGenericCollectionsCoverage(missingAllocationReports); err == nil ||
		!strings.Contains(err.Error(), "allocation-plan reports") {
		t.Fatalf("missing allocation report facts error = %v", err)
	}

	fakeBenchmark := cloneStableGenericCollectionsCoverage(report)
	for i := range fakeBenchmark.Rows {
		if fakeBenchmark.Rows[i].ID == StableGenericCollectionsBenchmarkGate {
			fakeBenchmark.Rows[i].Status = "implemented_benchmark_parity"
		}
	}
	if err := ValidateStableGenericCollectionsCoverage(fakeBenchmark); err == nil ||
		!strings.Contains(err.Error(), "benchmark parity") {
		t.Fatalf("fake benchmark parity error = %v", err)
	}
}

func requireStdlibCoverageFacts(t *testing.T, row RegionAwareStdlibEvidenceRow, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !hasStdlibCoverageText(row.RequiredFacts, want) {
			t.Fatalf("row %q missing fact %q: %#v", row.ID, want, row.RequiredFacts)
		}
	}
}

func hasStdlibCoverageText(items []string, want string) bool {
	for _, item := range items {
		if strings.Contains(item, want) {
			return true
		}
	}
	return false
}

func cloneStdlibCoverage(report RegionAwareStdlibCoverageReport) RegionAwareStdlibCoverageReport {
	clone := report
	clone.Rows = append([]RegionAwareStdlibEvidenceRow(nil), report.Rows...)
	clone.NonClaims = append([]string(nil), report.NonClaims...)
	return clone
}

func requireStableGenericCollectionsFacts(
	t *testing.T,
	row StableGenericCollectionsEvidenceRow,
	wants ...string,
) {
	t.Helper()
	for _, want := range wants {
		if !hasStdlibCoverageText(row.RequiredFacts, want) {
			t.Fatalf("row %q missing fact %q: %#v", row.ID, want, row.RequiredFacts)
		}
	}
}

func cloneStableGenericCollectionsCoverage(
	report StableGenericCollectionsCoverageReport,
) StableGenericCollectionsCoverageReport {
	clone := report
	clone.Rows = append([]StableGenericCollectionsEvidenceRow(nil), report.Rows...)
	clone.NonClaims = append([]string(nil), report.NonClaims...)
	return clone
}
