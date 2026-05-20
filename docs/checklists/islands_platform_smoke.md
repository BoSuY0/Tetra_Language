# Islands Platform Smoke Checklist

Date: 2025-12-30
Target version: linux-x64
Git HEAD:
Compiler version (compilerVersion): v0.6.0

## Prereqs

- Go 1.20+
- `tetra` built: `bash scripts/dev/bootstrap.sh`
- Record versions:
  - `./tetra version` (or `./tetra.exe version` on Windows)

## Windows x64

Build:
- [x] `./tetra build --target windows-x64 -o islands_hello.exe examples/islands_hello.tetra`
- [x] `./tetra build --target windows-x64 -o islands_i32.exe examples/islands_i32.tetra`
- [x] `./tetra build --target windows-x64 -o islands_overflow.exe examples/islands_overflow.tetra`
- [x] `./tetra build --target windows-x64 -o mmio_smoke.exe examples/mmio_smoke.tetra`
- [x] `./tetra build --target windows-x64 -o cap_mem_smoke.exe examples/cap_mem_smoke.tetra`
- [x] `./tetra build --target windows-x64 -o memset_smoke.exe examples/memset_smoke.tetra`

Run:
- [ ] `./islands_hello.exe` (exit code 0)
- [ ] `./islands_i32.exe` (exit code 55)
- [ ] `./islands_overflow.exe` (exit code 1)
- [ ] `./mmio_smoke.exe` (exit code 123)
- [ ] `./cap_mem_smoke.exe` (exit code 77)
- [ ] `./memset_smoke.exe` (exit code 88)

Notes:

## macOS x64

Build:
- [x] `./tetra build --target macos-x64 -o islands_hello examples/islands_hello.tetra`
- [x] `./tetra build --target macos-x64 -o islands_i32 examples/islands_i32.tetra`
- [x] `./tetra build --target macos-x64 -o islands_overflow examples/islands_overflow.tetra`
- [x] `./tetra build --target macos-x64 -o mmio_smoke examples/mmio_smoke.tetra`
- [x] `./tetra build --target macos-x64 -o cap_mem_smoke examples/cap_mem_smoke.tetra`
- [x] `./tetra build --target macos-x64 -o memset_smoke examples/memset_smoke.tetra`

Run:
- [ ] `./islands_hello` (exit code 0)
- [ ] `./islands_i32` (exit code 55)
- [ ] `./islands_overflow` (exit code 1)
- [ ] `./mmio_smoke` (exit code 123)
- [ ] `./cap_mem_smoke` (exit code 77)
- [ ] `./memset_smoke` (exit code 88)

Notes:

## Linux x64 (sanity)

Build:
- [x] `./tetra build --target linux-x64 -o islands_hello examples/islands_hello.tetra`
- [x] `./tetra build --target linux-x64 -o islands_i32 examples/islands_i32.tetra`
- [x] `./tetra build --target linux-x64 -o islands_overflow examples/islands_overflow.tetra`
- [x] `./tetra build --target linux-x64 -o mmio_smoke examples/mmio_smoke.tetra`
- [x] `./tetra build --target linux-x64 -o cap_mem_smoke examples/cap_mem_smoke.tetra`
- [x] `./tetra build --target linux-x64 -o memset_smoke examples/memset_smoke.tetra`

Run:
- [x] `./islands_hello` (exit code 0)
- [x] `./islands_i32` (exit code 55)
- [x] `./islands_overflow` (exit code 1)
- [x] `./mmio_smoke` (exit code 123)
- [x] `./cap_mem_smoke` (exit code 77)
- [x] `./memset_smoke` (exit code 88)

Notes:
