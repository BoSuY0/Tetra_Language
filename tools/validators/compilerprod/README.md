# tools/validators/compilerprod

Validator package for executable Compiler Production Core evidence.

This boundary owns the `tetra.compiler.production.v1` report contract. A
passing report must show a fresh Linux-x64 CLI compiler build, `v0.4.0` version
identity, native compile/run evidence, TOBJ object emission, interface-only
compilation, WASM target emission, parser/semantic/lowering/backend diagnostics,
compiler cache evidence, smoke-profile compilation coverage, and completion
audit rows.
