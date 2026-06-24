package memoryfacts

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestBuildReportFromGraphDeterministicProjection(t *testing.T) {
	graph := NewGraph("program")
	addReportFact(t, graph, "fact:z", "z", "site:z")
	addReportFact(t, graph, "fact:a", "a", "site:a")

	report := BuildReportFromGraph(graph)
	if err := ValidateReportProjection(graph, report); err != nil {
		t.Fatalf("ValidateReportProjection: %v", err)
	}
	if got, want := report.Rows[0].SourceFactID, FactID("fact:a"); got != want {
		t.Fatalf("first report row source_fact_id = %q, want %q", got, want)
	}
	if got, want := report.Rows[1].SourceFactID, FactID("fact:z"); got != want {
		t.Fatalf("second report row source_fact_id = %q, want %q", got, want)
	}

	reordered := NewGraph("program")
	addReportFact(t, reordered, "fact:a", "a", "site:a")
	addReportFact(t, reordered, "fact:z", "z", "site:z")
	reorderedReport := BuildReportFromGraph(reordered)
	if !reflect.DeepEqual(report, reorderedReport) {
		t.Fatalf("report projection depends on insertion order:\n%#v\n%#v", report, reorderedReport)
	}
}

func TestMemoryReportJSONMutationDoesNotMutateGraphOrSnapshot(t *testing.T) {
	graph := NewGraph("program")
	addReportFact(t, graph, "fact:a", "a", "site:a")
	snapshot, err := graph.Snapshot()
	if err != nil {
		t.Fatal(err)
	}

	raw, err := json.Marshal(BuildReportFromGraph(graph))
	if err != nil {
		t.Fatal(err)
	}
	var report Report
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatal(err)
	}
	report.Rows[0].Claim = "mutated"
	report.Rows[0].SourceFactID = "fact:mutated"

	if fact, ok := graph.Fact("fact:a"); !ok || fact.Claim != ClaimOwned {
		t.Fatalf("graph fact changed after report mutation: %#v ok=%v", fact, ok)
	}
	if fact, ok := snapshot.Fact("fact:a"); !ok || fact.Claim != ClaimOwned {
		t.Fatalf("snapshot fact changed after report mutation: %#v ok=%v", fact, ok)
	}
	if err := ValidateReportProjection(graph, BuildReportFromGraph(graph)); err != nil {
		t.Fatalf("fresh report projection after mutation: %v", err)
	}
}

func addReportFact(t *testing.T, graph *Graph, id FactID, valueID string, siteID string) {
	t.Helper()
	if _, err := graph.AddFact(Fact{
		ID:              id,
		FunctionID:      "main",
		ValueID:         valueID,
		SiteID:          siteID,
		SourceStage:     StagePLIR,
		ProvenanceClass: ProvenanceSafeKnown,
		UnsafeClass:     UnsafeSafe,
		Claim:           ClaimOwned,
	}); err != nil {
		t.Fatal(err)
	}
}
