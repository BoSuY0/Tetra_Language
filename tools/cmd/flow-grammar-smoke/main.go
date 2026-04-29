package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	out := flag.String("out", "", "write generated smoke source to this path")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "flow-grammar-smoke does not accept positional arguments")
		os.Exit(2)
	}
	src := generateSmokeSource()
	if *out == "" {
		fmt.Print(src)
		return
	}
	if err := os.WriteFile(*out, []byte(src), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func generateSmokeSource() string {
	return `module generated.flow_grammar_smoke

enum SmokeColor:
    case red
    case green

struct Pair:
    x: Int
    y: Int

protocol Drawable:
    func draw(self: Pair) -> Int

extension Pair:
    func sum(self: Pair) -> Int:
        return self.x + self.y

state CounterState:
    var count: Int = 0

view CounterView(state: CounterState):
    bind value: Int = state.count
    event click -> increment
    command increment:
        state.count = state.count + 1
    accessibility label: String = "Increment"

func id<T>(x: T) -> T:
    return x

func answer() -> Int = 42

async func worker() -> Int:
    return 42

func choose(color: SmokeColor) -> Int:
    match color:
    case SmokeColor.red:
        return 1
    case SmokeColor.green:
        return 2
    case _:
        return 0

func optional(value: Int?) -> Int:
    if let x = value:
        return x
    else:
        return 0

func main() -> Int:
    let pair: Pair = Pair(x: 40, y: 2)
    return pair.x + pair.y + choose(SmokeColor.red) + optional(none) + id(0) + answer() - 42

test "generated grammar smoke":
    expect 40 + 2 == 42
`
}
