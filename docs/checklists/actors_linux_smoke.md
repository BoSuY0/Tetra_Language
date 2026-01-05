# Actors Linux Smoke Checklist

> **Deprecated:** use `docs/checklists/actors_platform_smoke.md` (this file is kept for historical reference).

Date:
Target version:
Git HEAD:
Compiler version (compilerVersion):

## Prereqs

- Linux x64 host
- `tetra` built: `bash scripts/bootstrap.sh`

## Build

- [ ] `./tetra build --target linux-x64 -o actors_pingpong examples/actors_pingpong.tetra`

## Run

- [ ] `./actors_pingpong` (exit code 0)

## Notes
