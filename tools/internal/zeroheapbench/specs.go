package zeroheapbench

import (
	"path/filepath"
	"strings"
)

const (
	Schema = "tetra.local_benchmark_zero_heap.v1"
	Scope  = "tetra.zero_heap_microbenchmarks.v1"
)

var Categories = []string{
	"zero heap fixed local array sum",
	"zero heap read-only local call slice",
	"zero heap small struct copy",
	"zero heap borrowed view sum",
	"zero heap copy eliminated unused",
}

type Spec struct {
	Name             string
	Category         string
	Language         string
	AlgorithmID      string
	InputDescription string
	BuildArgs        []string
	SourceRelPath    string
	BinaryRelPath    string
	Source           string
}

func BuildSpecs(outDir string) []Spec {
	specs := make([]Spec, 0, len(Categories))
	for _, category := range Categories {
		name := Slug(category) + "_tetra"
		specs = append(specs, Spec{
			Name:             name,
			Category:         category,
			Language:         "tetra",
			AlgorithmID:      "zero_heap." + Slug(category),
			InputDescription: "Tetra-only zero-heap compiler guardrail outside the comparable Tier 1 matrix",
			BuildArgs:        []string{"tetra", "build", "--target", "linux-x64", "--explain"},
			SourceRelPath:    filepath.Join(outDir, "artifacts", "src", "zero_heap", WorkloadSlug(category)+".tetra"),
			BinaryRelPath:    filepath.Join(outDir, "artifacts", "bin", name),
			Source:           Source(category),
		})
	}
	return specs
}

func Source(category string) string {
	switch category {
	case "zero heap fixed local array sum":
		return `module zero_heap.fixed_local_array_sum

func main() -> Int
uses alloc, mem:
    var xs: []i32 = core.make_i32(8)
    xs[0] = 1
    xs[1] = 2
    xs[2] = 3
    xs[3] = 4
    xs[4] = 5
    xs[5] = 6
    xs[6] = 7
    xs[7] = 14
    let total: Int = xs[0] + xs[1] + xs[2] + xs[3] + xs[4] + xs[5] + xs[6] + xs[7]
    if total == 42:
        return 0
    return 1
`
	case "zero heap read-only local call slice":
		return `module zero_heap.read_only_local_call_slice

func sum(xs: []i32, n: Int) -> Int
uses mem:
    var i: Int = 0
    var total: Int = 0
    while i < n:
        total = total + xs[i]
        i = i + 1
    return total

func main() -> Int
uses alloc, mem:
    let n: Int = 8
    var xs: []i32 = core.make_i32(n)
    var i: Int = 0
    while i < n:
        xs[i] = i + 1
        i = i + 1
    if sum(xs, n) == 36:
        return 0
    return 1
`
	case "zero heap small struct copy":
		return `module zero_heap.small_struct_copy

struct Pair:
    x: i32
    y: i32

func mix(p: Pair) -> i32:
    return p.x + p.y

func main() -> i32:
    var p: Pair = Pair(x: 20, y: 22)
    let q: Pair = p
    if mix(q) == 42:
        return 0
    return 1
`
	case "zero heap borrowed view sum":
		return `module zero_heap.borrowed_view_sum

func first_four(xs: []i32) -> Int
uses mem:
    return xs[0] + xs[1] + xs[2] + xs[3]

func main() -> Int
uses alloc, mem:
    var xs: []i32 = core.make_i32(4)
    xs[0] = 10
    xs[1] = 11
    xs[2] = 12
    xs[3] = 9
    if first_four(xs) == 42:
        return 0
    return 1
`
	case "zero heap copy eliminated unused":
		return `module zero_heap.copy_eliminated_unused

func main() -> Int
uses alloc, mem:
    var xs: []i32 = core.make_i32(4)
    xs[0] = 40
    xs[1] = 2
    let unused: []i32 = xs
    if xs[0] + xs[1] == 42:
        return 0
    return 1
`
	default:
		return `module zero_heap.default

func main() -> Int:
    return 0
`
	}
}

func Slug(value string) string {
	replacer := strings.NewReplacer("/", "_", "-", "_")
	return strings.Join(strings.Fields(replacer.Replace(strings.ToLower(value))), "_")
}

func WorkloadSlug(category string) string {
	return strings.TrimPrefix(Slug(category), "zero_heap_")
}
