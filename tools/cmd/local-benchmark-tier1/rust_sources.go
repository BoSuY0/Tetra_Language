package main

func rustSource(category string) string {
	body := rustBody(category)
	return "use std::hint::black_box;\n" + body
}

func rustBody(category string) string {
	switch category {
	case "slice sum", "bounds-check loops":
		return `fn main() {
    let n = 4096usize;
    let mut xs = vec![0i64; n];
    for i in 0..n { xs[i] = (i % 97) as i64; }
    let mut total = 0i64;
    for _ in 0..128 {
        for i in 0..n { total += xs[(i * 17) % n]; }
    }
    black_box(total);
}
`
	case "function calls":
		return `#[inline(never)]
fn mix(a: i64, b: i64) -> i64 { (a * 3 + b) % 97 }
fn main() {
    let mut total = 0i64;
    for i in 0..400000 { total += mix(i, total); }
    black_box(total);
}
`
	case "recursion":
		return `fn fib(n: i64) -> i64 { if n < 2 { n } else { fib(n - 1) + fib(n - 2) } }
fn main() {
    let mut total = 0i64;
    for _ in 0..80 { total += fib(12); }
    black_box(total);
}
`
	case "matrix multiply":
		return `fn main() {
    let n = 16usize;
    let mut a = vec![0i64; n*n];
    let mut b = vec![0i64; n*n];
    let mut c = vec![0i64; n*n];
    for i in 0..n*n { a[i] = (i % 13) as i64; b[i] = ((i * 7) % 17) as i64; }
    let mut total = 0i64;
    for r in 0..64 {
        for row in 0..n {
            for col in 0..n {
                let mut acc = 0i64;
                for k in 0..n { acc += a[row*n+k] * b[k*n+col]; }
                c[row*n+col] = acc;
            }
        }
        total += c[r % (n*n)];
    }
    black_box(total);
}
`
	case "hash table":
		return `use std::collections::HashMap;
fn main() {
    let n = 1024i64;
    let mut map = HashMap::new();
    for i in 0..n { map.insert(i * 2 + 1, i + 7); }
    let mut total = 0i64;
    for q in 0..n { total += *map.get(&(q * 2 + 1)).unwrap_or(&0); }
    black_box(total);
}
`
	case "allocation", "region/island allocation":
		return `fn main() {
    let mut total = 0i64;
    for r in 0..4096i64 {
        let mut xs = vec![0i64; 64];
        for i in 0..64usize { xs[i] = r + i as i64; }
        total += xs[(r as usize) % 64];
    }
    black_box(total);
}
`
	case "JSON parse/stringify":
		return `fn main() {
    let mut total = 0usize;
    for i in 0..20000 {
        let s = format!("{{\"message\":\"Hello, World!\",\"id\":{}}}", i % 100);
        total += s.len() + usize::from(s.as_bytes()[1] == b'"');
    }
    black_box(total);
}
`
	case "HTTP plaintext/json":
		return `fn main() {
    let mut total = 0usize;
    for _ in 0..20000 {
        let s = "HTTP/1.1 200 OK\r\nServer: Tetra\r\nContent-Length: 13\r\n\r\nHello, World!";
        total += s.len() + usize::from(s.as_bytes()[0] == b'H');
    }
    black_box(total);
}
`
	case "PostgreSQL single/multiple/update":
		return `fn main() {
    let mut frame = [0u8; 64];
    let mut total = 0u64;
    for i in 0..20000u64 {
        frame[0] = b'D';
        frame[4] = 12;
        frame[6] = 2;
        total += frame[0] as u64 + frame[6] as u64 + (i % 17);
    }
    black_box(total);
}
`
	case "actor ping-pong", "parallel map/reduce":
		return `fn main() {
    let mut total = 0i64;
    for i in 0..400000 { total += if i % 2 == 0 { 41 } else { 42 }; }
    black_box(total);
}
`
	case "startup time", "binary size":
		return `fn main() { black_box(42); }
`
	case "compile time":
		return `#[inline(never)] fn f0(x: i64) -> i64 { x + 1 }
#[inline(never)] fn f1(x: i64) -> i64 { f0(x) * 3 }
#[inline(never)] fn f2(x: i64) -> i64 { f1(x) + f0(x) }
fn main() {
    let mut total = 0i64;
    for i in 0..200000 { total += f2(i); }
    black_box(total);
}
`
	default:
		return `fn main() {
    let mut total = 0i64;
    for i in 0..400000 { total += i % 7; }
    black_box(total);
}
`
	}
}
