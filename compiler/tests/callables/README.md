# compiler/tests/callables

Function-typed callable behavior tests are split into domain subpackages so no
directory carries more than six active Go test files. Shared build/run helpers
live in `compiler/tests/callables/testkit`; generic fixture helpers remain in
`compiler/internal/testkit`.

Current groups:

- `core/`: direct symbols, local return aliases, parameter-return flow
- `captures/`: captured closures and returned struct/enum payload captures
- `globals/`: global values, mutable global storage, imported globals, enum payloads
- `reassignment/`: callable reassignment and enum payload reassignment
- `cross_module/`: callbacks, returns, multi-targets, and direct storage across modules
- `throwing/`: throwing callable smoke tests
- `unsupported/`: unsupported callable diagnostics
