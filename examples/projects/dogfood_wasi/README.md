# Dogfood WASI

WASI dogfood project scoped to the currently supported subset: deterministic
entrypoint, WASI stdout through `print`, no filesystem, no network, and no host
preopen assumptions.

Expected run output:

```text
wasi dogfood: ok
```

Expected exit code: `0`.

