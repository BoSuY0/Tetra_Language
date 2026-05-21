# Tutorial Path

Status: recommended reading order for new users of the current `v0.4.0` profile.

This path uses existing release-covered examples and avoids claiming future
`v1.0.0` features as current support. The current support boundary is
`docs/spec/current_supported_surface.md`.

## Path

1. Read `docs/user/status.md` to understand what is current, next-cycle, and
   future.
2. Build the CLI with `bash scripts/dev/bootstrap.sh` from
   `docs/user/getting_started.md`.
3. Run `./tetra check examples/flow_hello.tetra` and
   `./tetra run examples/flow_hello.tetra`.
4. Use `docs/user/cli_cheatsheet.md` for the common command surface.
5. Run the short project path: `check`, `build`, `run`, `test`, then `doc`.
6. Read `docs/user/language_tour.md` before diving into the full specs.
7. Explore `docs/user/standard_library_guide.md` and the matching
   `examples/core_*_smoke.tetra` files.
8. Use `docs/user/examples_index.md` to find release-covered examples by topic.
9. Use `docs/user/troubleshooting.md` when a command fails.

## Project-First Practice

After the single-file hello flow, use the T4 project example:

```sh
./tetra check examples/projects/hello_t4
./tetra build examples/projects/hello_t4
./tetra run examples/projects/hello_t4
./tetra test examples/projects/hello_t4
./tetra doc examples/projects/hello_t4
```

The project root is `examples/projects/hello_t4/Capsule.t4`, and the entry source
is `examples/projects/hello_t4/src/main.t4`.
