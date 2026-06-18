package compiler_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	compiler "tetra_language/compiler"
)

func TestBuildBoundsReportShowsWindowLoopCheckRemoval(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
func sum_window(xs: []i32) -> Int
uses mem:
    var total = 0
    for x in xs.window(1, 2):
        total = total + x
    return total

func get(xs: []i32, i: Int) -> Int
uses mem:
    return xs[i]

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(3)
    xs[0] = 1
    xs[1] = 2
    xs[2] = 3
    return sum_window(xs) + get(xs, 1)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:             1,
		EmitBoundsReport: true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	var bounds struct {
		Totals struct {
			Removed int `json:"removed"`
			Left    int `json:"left"`
		} `json:"totals"`
		Functions []struct {
			Function string `json:"function"`
			Sites    []struct {
				Removed bool   `json:"removed"`
				ProofID string `json:"proof_id"`
				Reason  string `json:"reason"`
			} `json:"sites"`
		} `json:"functions"`
	}
	raw, err := os.ReadFile(outPath + ".bounds.json")
	if err != nil {
		t.Fatalf("read bounds report: %v", err)
	}
	if err := json.Unmarshal(raw, &bounds); err != nil {
		t.Fatalf("parse bounds report: %v", err)
	}
	if bounds.Totals.Removed == 0 || bounds.Totals.Left == 0 {
		t.Fatalf("bounds totals = %+v, want removed window loop check and left external check", bounds.Totals)
	}
	var sawWindowLoopRemoval bool
	for _, fn := range bounds.Functions {
		if fn.Function != "sum_window" {
			continue
		}
		for _, site := range fn.Sites {
			if site.Removed && site.ProofID != "" && site.Reason != "" {
				sawWindowLoopRemoval = true
			}
		}
	}
	if !sawWindowLoopRemoval {
		t.Fatalf("bounds report did not show proof-tagged removal for sum_window: %+v", bounds.Functions)
	}
}

func TestBuildBoundsReportShowsViewChainReason(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
func sum_chain(xs: []i32) -> Int
uses mem:
    let view: []i32 = xs.prefix(4).suffix(1)
    var total = 0
    for x in view:
        total = total + x
    return total

func sum_bad() -> Int:
    let view: String = core.string_window("abc", 4, 0)
    var total = 0
    for ch in view:
        total = total + ch
    return total

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(4)
    xs[0] = 1
    xs[1] = 2
    xs[2] = 3
    xs[3] = 4
    return sum_chain(xs) + sum_bad()
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:             1,
		EmitBoundsReport: true,
		EmitProof:        true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	var bounds struct {
		Functions []struct {
			Function string `json:"function"`
			Sites    []struct {
				Removed bool   `json:"removed"`
				ProofID string `json:"proof_id"`
				Reason  string `json:"reason"`
			} `json:"sites"`
		} `json:"functions"`
	}
	raw, err := os.ReadFile(outPath + ".bounds.json")
	if err != nil {
		t.Fatalf("read bounds report: %v", err)
	}
	if err := json.Unmarshal(raw, &bounds); err != nil {
		t.Fatalf("parse bounds report: %v", err)
	}
	var sawChain bool
	var sawBadChecked bool
	for _, fn := range bounds.Functions {
		for _, site := range fn.Sites {
			if fn.Function == "sum_chain" && site.Removed && site.Reason == "removed_by_view_chain" && site.ProofID != "" {
				sawChain = true
			}
			if fn.Function == "sum_bad" && !site.Removed && site.ProofID == "" {
				sawBadChecked = true
			}
			if fn.Function == "sum_bad" && (site.Removed || site.ProofID != "") {
				t.Fatalf("invalid view chain must not claim removed proof site: %+v", fn.Sites)
			}
		}
	}
	if !sawChain {
		t.Fatalf("bounds report missing removed_by_view_chain for sum_chain: %+v", bounds.Functions)
	}
	if !sawBadChecked {
		t.Fatalf("bounds report missing checked invalid view site for sum_bad: %+v", bounds.Functions)
	}
}

func TestBuildBoundsAndProofReportsShowWhileRangeReason(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
func sum_while(xs: []i32) -> Int
uses mem:
    var total = 0
    var i = 0
    while i < xs.len:
        total = total + xs[i]
        i = i + 1
    return total

func get(xs: []i32, i: Int) -> Int
uses mem:
    return xs[i]

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(2)
    xs[0] = 1
    xs[1] = 2
    return sum_while(xs) + get(xs, 0)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:             1,
		EmitBoundsReport: true,
		EmitProof:        true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	boundsRaw, err := os.ReadFile(outPath + ".bounds.json")
	if err != nil {
		t.Fatalf("read bounds report: %v", err)
	}
	boundsText := string(boundsRaw)
	for _, want := range []string{
		`"reason": "removed_by_while_range"`,
		`"reason": "left_missing_dominance"`,
		`"proof_id": "proof:while:`,
	} {
		if !strings.Contains(boundsText, want) {
			t.Fatalf("bounds report missing %q:\n%s", want, boundsText)
		}
	}

	proofRaw, err := os.ReadFile(outPath + ".proof.json")
	if err != nil {
		t.Fatalf("read proof report: %v", err)
	}
	proofText := string(proofRaw)
	for _, want := range []string{
		`"reason": "while loop range proof"`,
		`"removed_bounds_check": true`,
		`"guard": "i < xs.len"`,
		`"fact": "i in [0, xs.len);`,
		`derivation: non_negative, less_than_len`,
	} {
		if !strings.Contains(proofText, want) {
			t.Fatalf("proof report missing %q:\n%s", want, proofText)
		}
	}
}

func TestBuildBoundsAndProofReportsShowCanonicalWhileIncrementReasons(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
func sum_commuted(xs: []i32) -> Int
uses mem:
    var total = 0
    var i = 0
    while i < xs.len:
        total = total + xs[i]
        i = 1 + i
    return total

func sum_step(xs: []i32) -> Int
uses mem:
    let step: Int = 1
    var total = 0
    var i = 0
    while i < xs.len:
        total = total + xs[i]
        i = i + step
    return total

func sum_alias(xs: []i32) -> Int
uses mem:
    let start: Int = 0
    let end: Int = xs.len
    var total = 0
    var i = start
    while i < end:
        total = total + xs[i]
        i = i + 1
    return total

func sum_bad(xs: []i32) -> Int
uses mem:
    var total = 0
    var i = 0
    while i < xs.len:
        total = total + xs[i]
        i = i + 2
    return total

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(2)
    xs[0] = 1
    xs[1] = 2
    return sum_commuted(xs) + sum_step(xs) + sum_alias(xs) + sum_bad(xs)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:             1,
		EmitBoundsReport: true,
		EmitProof:        true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	var bounds struct {
		Functions []struct {
			Function string `json:"function"`
			Sites    []struct {
				Removed bool   `json:"removed"`
				ProofID string `json:"proof_id"`
				Reason  string `json:"reason"`
			} `json:"sites"`
		} `json:"functions"`
	}
	raw, err := os.ReadFile(outPath + ".bounds.json")
	if err != nil {
		t.Fatalf("read bounds report: %v", err)
	}
	if err := json.Unmarshal(raw, &bounds); err != nil {
		t.Fatalf("parse bounds report: %v", err)
	}
	findSite := func(function string, removed bool) (reason string, proofID string, ok bool) {
		for _, fn := range bounds.Functions {
			if fn.Function != function {
				continue
			}
			for _, site := range fn.Sites {
				if site.Removed == removed {
					return site.Reason, site.ProofID, true
				}
			}
		}
		return "", "", false
	}
	for _, function := range []string{"sum_commuted", "sum_step", "sum_alias"} {
		if reason, proofID, ok := findSite(function, true); !ok || reason != "removed_by_while_range" || !strings.HasPrefix(proofID, "proof:while:") {
			t.Fatalf("%s site = reason %q proof %q ok=%v, want removed_by_while_range with proof:while", function, reason, proofID, ok)
		}
	}
	if reason, proofID, ok := findSite("sum_bad", false); !ok || proofID != "" || reason == "removed_by_while_range" {
		t.Fatalf("sum_bad site = reason %q proof %q ok=%v, want checked site without while removal", reason, proofID, ok)
	}
}

func TestBuildBoundsReportShowsMutationInvalidationReason(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
func sum_reassign(xs: []i32, ys: []i32) -> Int
uses mem:
    var view: []i32 = xs
    var total = 0
    var i = 0
    while i < view.len:
        view = ys
        total = total + view[i]
        i = i + 1
    return total

func touch(view: inout []i32) -> Int
uses mem:
    return view.len

func sum_inout(view: inout []i32) -> Int
uses mem:
    var total = 0
    var i = 0
    while i < view.len:
        touch(view)
        total = total + view[i]
        i = i + 1
    return total

func sum_callback(view: inout []i32, cb: fn(inout []i32) -> Int uses mem) -> Int
uses mem:
    var total = 0
    var i = 0
    while i < view.len:
        cb(view)
        total = total + view[i]
        i = i + 1
    return total

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(1)
    var ys: []i32 = make_i32(1)
    xs[0] = 1
    ys[0] = 2
    return sum_reassign(xs, ys) + sum_inout(xs) + sum_callback(xs, touch)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:             1,
		EmitBoundsReport: true,
		EmitProof:        true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	var bounds struct {
		Functions []struct {
			Function string `json:"function"`
			Sites    []struct {
				Removed bool   `json:"removed"`
				ProofID string `json:"proof_id"`
				Reason  string `json:"reason"`
			} `json:"sites"`
		} `json:"functions"`
	}
	raw, err := os.ReadFile(outPath + ".bounds.json")
	if err != nil {
		t.Fatalf("read bounds report: %v", err)
	}
	if err := json.Unmarshal(raw, &bounds); err != nil {
		t.Fatalf("parse bounds report: %v", err)
	}

	for _, wantFunction := range []string{"sum_reassign", "sum_inout", "sum_callback"} {
		var sawInvalidated bool
		for _, fn := range bounds.Functions {
			if fn.Function != wantFunction {
				continue
			}
			for _, site := range fn.Sites {
				if site.Removed || site.ProofID != "" {
					t.Fatalf("%s mutation-invalidated site must remain checked without proof: %+v", wantFunction, fn.Sites)
				}
				if site.Reason == "left_proof_invalidated_by_mutation" {
					sawInvalidated = true
				}
			}
		}
		if !sawInvalidated {
			t.Fatalf("bounds report missing left_proof_invalidated_by_mutation for %s: %+v", wantFunction, bounds.Functions)
		}
	}
}

func TestBuildBoundsReportShowsBranchGuardReasons(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
func branch_remove(xs: []i32, i: Int) -> Int
uses mem:
    if i >= 0 && i < xs.len:
        return xs[i]
    return 0

func branch_missing_lower(xs: []i32, i: Int) -> Int
uses mem:
    if i < xs.len:
        return xs[i]
    return 0

func branch_not_dominating(xs: []i32, i: Int) -> Int
uses mem:
    if i >= 0 && i < xs.len:
        var j = i + 0
    return xs[i]

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(1)
    xs[0] = 7
    return branch_remove(xs, 0) + branch_missing_lower(xs, 0) + branch_not_dominating(xs, 0)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:             1,
		EmitBoundsReport: true,
		EmitProof:        true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	var bounds struct {
		Functions []struct {
			Function string `json:"function"`
			Sites    []struct {
				Removed bool   `json:"removed"`
				ProofID string `json:"proof_id"`
				Reason  string `json:"reason"`
			} `json:"sites"`
		} `json:"functions"`
	}
	raw, err := os.ReadFile(outPath + ".bounds.json")
	if err != nil {
		t.Fatalf("read bounds report: %v", err)
	}
	if err := json.Unmarshal(raw, &bounds); err != nil {
		t.Fatalf("parse bounds report: %v", err)
	}
	findSite := func(function string, removed bool) (reason string, proofID string, ok bool) {
		for _, fn := range bounds.Functions {
			if fn.Function != function {
				continue
			}
			for _, site := range fn.Sites {
				if site.Removed == removed {
					return site.Reason, site.ProofID, true
				}
			}
		}
		return "", "", false
	}

	if reason, proofID, ok := findSite("branch_remove", true); !ok || reason != "removed_by_branch_guard" || !strings.HasPrefix(proofID, "proof:if:") {
		t.Fatalf("branch_remove site = reason %q proof %q ok=%v, want removed_by_branch_guard with proof:if", reason, proofID, ok)
	}
	if reason, _, ok := findSite("branch_missing_lower", false); !ok || reason != "left_missing_non_negative_lower_bound" {
		t.Fatalf("branch_missing_lower reason = %q ok=%v, want left_missing_non_negative_lower_bound", reason, ok)
	}
	if reason, _, ok := findSite("branch_not_dominating", false); !ok || reason != "left_guard_not_dominating" {
		t.Fatalf("branch_not_dominating reason = %q ok=%v, want left_guard_not_dominating", reason, ok)
	}
}

func TestBuildBoundsReportDoesNotClaimProofForInvalidConstructorLoop(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
func sum_bad() -> Int
uses alloc, mem:
    var total = 0
    for x in make_i32(0 - 1):
        total = total + x
    return total

func main() -> Int
uses alloc, mem:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:             1,
		EmitBoundsReport: true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	var bounds struct {
		Totals struct {
			Removed int `json:"removed"`
			Left    int `json:"left"`
		} `json:"totals"`
		Functions []struct {
			Function string `json:"function"`
			Sites    []struct {
				Removed bool   `json:"removed"`
				ProofID string `json:"proof_id"`
			} `json:"sites"`
		} `json:"functions"`
	}
	raw, err := os.ReadFile(outPath + ".bounds.json")
	if err != nil {
		t.Fatalf("read bounds report: %v", err)
	}
	if err := json.Unmarshal(raw, &bounds); err != nil {
		t.Fatalf("parse bounds report: %v", err)
	}
	for _, fn := range bounds.Functions {
		if fn.Function != "sum_bad" {
			continue
		}
		for _, site := range fn.Sites {
			if site.Removed || site.ProofID != "" {
				t.Fatalf("invalid constructor loop must not claim removed proof site: %+v", fn.Sites)
			}
		}
		return
	}
	t.Fatalf("bounds report missing sum_bad checked site: %+v totals=%+v", bounds.Functions, bounds.Totals)
}

func TestBuildBoundsReportShowsStringWindowLoopCheckRemoval(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
func sum_window(text: String) -> Int
uses mem:
    var total = 0
    for ch in text.window(1, 3):
        total = total + ch
    return total

func get(text: String, i: Int) -> Int
uses mem:
    return text[i]

func main() -> Int
uses mem:
    let text: String = "abcdef"
    return sum_window(text) + get(text, 1)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:             1,
		EmitBoundsReport: true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	var bounds struct {
		Totals struct {
			Removed int `json:"removed"`
			Left    int `json:"left"`
		} `json:"totals"`
		Functions []struct {
			Function string `json:"function"`
			Sites    []struct {
				Removed bool   `json:"removed"`
				ProofID string `json:"proof_id"`
				Reason  string `json:"reason"`
			} `json:"sites"`
		} `json:"functions"`
	}
	raw, err := os.ReadFile(outPath + ".bounds.json")
	if err != nil {
		t.Fatalf("read bounds report: %v", err)
	}
	if err := json.Unmarshal(raw, &bounds); err != nil {
		t.Fatalf("parse bounds report: %v", err)
	}
	if bounds.Totals.Removed == 0 || bounds.Totals.Left == 0 {
		t.Fatalf("bounds totals = %+v, want removed String window loop check and left external check", bounds.Totals)
	}
	var sawStringWindowLoopRemoval bool
	for _, fn := range bounds.Functions {
		if fn.Function != "sum_window" {
			continue
		}
		for _, site := range fn.Sites {
			if site.Removed && site.ProofID != "" && site.Reason != "" {
				sawStringWindowLoopRemoval = true
			}
		}
	}
	if !sawStringWindowLoopRemoval {
		t.Fatalf("bounds report did not show proof-tagged removal for sum_window: %+v", bounds.Functions)
	}
}

func TestBuildBoundsReportDoesNotClaimProofForInvalidStringViewConstructorLoop(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
func sum_bad() -> Int:
    var total = 0
    for ch in core.string_window("abc", 4, 0):
        total = total + ch
    return total

func main() -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:             1,
		EmitBoundsReport: true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	var bounds struct {
		Totals struct {
			Removed int `json:"removed"`
			Left    int `json:"left"`
		} `json:"totals"`
		Functions []struct {
			Function string `json:"function"`
			Sites    []struct {
				Removed bool   `json:"removed"`
				ProofID string `json:"proof_id"`
			} `json:"sites"`
		} `json:"functions"`
	}
	raw, err := os.ReadFile(outPath + ".bounds.json")
	if err != nil {
		t.Fatalf("read bounds report: %v", err)
	}
	if err := json.Unmarshal(raw, &bounds); err != nil {
		t.Fatalf("parse bounds report: %v", err)
	}
	for _, fn := range bounds.Functions {
		if fn.Function != "sum_bad" {
			continue
		}
		for _, site := range fn.Sites {
			if site.Removed || site.ProofID != "" {
				t.Fatalf("invalid String view constructor loop must not claim removed proof site: %+v", fn.Sites)
			}
		}
		return
	}
	t.Fatalf("bounds report missing sum_bad checked site: %+v totals=%+v", bounds.Functions, bounds.Totals)
}
