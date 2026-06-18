# Surface Performance Validator

`tools/validators/surfaceperf` validates `tetra.surface.perf-report.v1`
evidence for `surface-performance-memory-v1`.

The package owns startup, frame, memory, binary-size, cache, power, and
Electron-comparison budget checks for the scoped Surface production boundary.
It rejects unsupported faster-than-Electron, fastest-framework, and zero-memory
overhead claims.
