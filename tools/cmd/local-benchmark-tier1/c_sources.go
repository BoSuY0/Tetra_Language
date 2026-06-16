package main

func cLikeSource(category string) string {
	body := cLikeBody(category)
	return `#include <stdint.h>
#include <stdlib.h>
#include <string.h>
#include <stdio.h>
static volatile int64_t sink;
static int done(int64_t value) { sink = value; return 0; }
` + body
}

func cLikeBody(category string) string {
	switch category {
	case "slice sum", "bounds-check loops":
		return `int main(void) {
  const int n = 4096;
  int *xs = (int*)malloc(sizeof(int) * n);
  int64_t total = 0;
  for (int i = 0; i < n; i++) xs[i] = i % 97;
  for (int r = 0; r < 128; r++) {
    for (int i = 0; i < n; i++) total += xs[(i * 17) % n];
  }
  free(xs);
  return done(total);
}
`
	case "function calls":
		return `static int64_t mix(int64_t a, int64_t b) { return (a * 3 + b) % 97; }
int main(void) {
  int64_t total = 0;
  for (int i = 0; i < 400000; i++) total += mix(i, total);
  return done(total);
}
`
	case "recursion":
		return `static int64_t fib(int n) { if (n < 2) return n; return fib(n - 1) + fib(n - 2); }
int main(void) {
  int64_t total = 0;
  for (int i = 0; i < 80; i++) total += fib(12);
  return done(total);
}
`
	case "matrix multiply":
		return `int main(void) {
  const int n = 16;
  int *a = (int*)malloc(sizeof(int) * n * n);
  int *b = (int*)malloc(sizeof(int) * n * n);
  int *c = (int*)malloc(sizeof(int) * n * n);
  int64_t total = 0;
  for (int i = 0; i < n * n; i++) { a[i] = i % 13; b[i] = (i * 7) % 17; }
  for (int r = 0; r < 64; r++) {
    for (int row = 0; row < n; row++) for (int col = 0; col < n; col++) {
      int acc = 0;
      for (int k = 0; k < n; k++) acc += a[row*n+k] * b[k*n+col];
      c[row*n+col] = acc;
    }
    total += c[r % (n*n)];
  }
  free(a); free(b); free(c);
  return done(total);
}
`
	case "hash table":
		return `int main(void) {
  const int n = 1024;
  int *keys = (int*)malloc(sizeof(int) * n);
  int *values = (int*)malloc(sizeof(int) * n);
  int64_t total = 0;
  for (int i = 0; i < n; i++) { keys[i] = i * 2 + 1; values[i] = i + 7; }
  for (int q = 0; q < n; q++) {
    int key = q * 2 + 1;
    for (int i = 0; i < n; i++) if (keys[i] == key) { total += values[i]; break; }
  }
  free(keys); free(values);
  return done(total);
}
`
	case "allocation", "region/island allocation":
		return `int main(void) {
  int64_t total = 0;
  for (int r = 0; r < 4096; r++) {
    int *xs = (int*)malloc(sizeof(int) * 64);
    for (int i = 0; i < 64; i++) xs[i] = r + i;
    total += xs[r % 64];
    free(xs);
  }
  return done(total);
}
`
	case "JSON parse/stringify":
		return `int main(void) {
  char buf[128];
  int64_t total = 0;
  for (int i = 0; i < 20000; i++) {
    int n = snprintf(buf, sizeof(buf), "{\"message\":\"Hello, World!\",\"id\":%d}", i % 100);
    total += n + (buf[1] == '"' ? 1 : 0);
  }
  return done(total);
}
`
	case "HTTP plaintext/json":
		return `int main(void) {
  char buf[256];
  int64_t total = 0;
  for (int i = 0; i < 20000; i++) {
    int n = snprintf(buf, sizeof(buf), "HTTP/1.1 200 OK\r\nServer: Tetra\r\nContent-Length: 13\r\n\r\nHello, World!");
    total += n + (buf[0] == 'H' ? 1 : 0);
  }
  return done(total);
}
`
	case "PostgreSQL single/multiple/update":
		return `int main(void) {
  unsigned char frame[64];
  int64_t total = 0;
  for (int i = 0; i < 20000; i++) {
    frame[0] = 'D';
    frame[1] = 0; frame[2] = 0; frame[3] = 0; frame[4] = 12;
    frame[5] = 0; frame[6] = 2;
    total += frame[0] + frame[6] + (i % 17);
  }
  return done(total);
}
`
	case "actor ping-pong", "parallel map/reduce":
		return `int main(void) {
  int64_t total = 0;
  for (int i = 0; i < 400000; i++) total += (i % 2 == 0) ? 41 : 42;
  return done(total);
}
`
	case "startup time", "binary size":
		return `int main(void) { return done(42); }
`
	case "compile time":
		return `static int64_t f0(int64_t x) { return x + 1; }
static int64_t f1(int64_t x) { return f0(x) * 3; }
static int64_t f2(int64_t x) { return f1(x) + f0(x); }
int main(void) {
  int64_t total = 0;
  for (int i = 0; i < 200000; i++) total += f2(i);
  return done(total);
}
`
	default:
		return `int main(void) {
  int64_t total = 0;
  for (int i = 0; i < 400000; i++) total += i % 7;
  return done(total);
}
`
	}
}
