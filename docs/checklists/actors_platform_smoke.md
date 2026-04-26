# Actors Platform Smoke Checklist

Date: 2025-12-30
Target version: linux-x64
Git HEAD:
Compiler version (compilerVersion): v0.6.0

## Prereqs

- Go 1.20+
- `tetra` built: `bash scripts/bootstrap.sh`

## Windows x64

Build:
- [x] `./tetra build --target windows-x64 -o actors_pingpong.exe examples/actors_pingpong.tetra`

Run:
- [ ] `./actors_pingpong.exe` (exit code 0)

Notes:

## macOS x64

Build:
- [x] `./tetra build --target macos-x64 -o actors_pingpong examples/actors_pingpong.tetra`

Run:
- [ ] `./actors_pingpong` (exit code 0)

Notes:

## Linux x64 (sanity)

Build:
- [x] `./tetra build --target linux-x64 -o actors_pingpong examples/actors_pingpong.tetra`

Run:
- [x] `./actors_pingpong` (exit code 0)

Notes:
