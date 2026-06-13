# TOON Format

TOON is an opt-in structured output/input format for tested Tetra surfaces.
JSON remains the default and canonical format for existing reports, release
artifacts, manifests, LSP JSON-RPC, HTTP payloads, and Eco package/store
metadata unless a command explicitly documents TOON.

Supported commands in the current scoped implementation:

```sh
tetra targets --format=toon
tetra features --format=toon
tetra formats --format=toon
tetra doctor --format=toon
tetra smoke --list --format=toon
tetra lsp --stdio-smoke examples/flow_hello.tetra --format=toon
tetra run --diagnostics=toon --target not-a-target
tetra test --report=toon path/to/tests.tetra
tetra test --format=toon path/to/tests.tetra
```

Path-based report commands keep JSON by default and can add TOON mirrors:

```sh
tetra smoke --target linux-x64 --run=false --report reports/smoke.json --report-format=both
bash scripts/ci/test-all.sh --quick --report-dir reports/test-all --report-format=both
go run ./tools/cmd/gen-manifest -o reports/manifest.json --format=both
```

Selected Eco report artifacts also support TOON or JSON+TOON mirrors:

```sh
tetra eco verify --lock Tetra.lock --lock-format=both Capsule.t4
tetra eco seed export --out tetra.seed.json --format=both Capsule.t4
tetra eco needmap --lock Tetra.lock -o tetra.needmap.json --format=both
tetra eco trust snapshot --lock Tetra.lock --store .tetra/todex-vault -o trust.snapshot.json --format=both
tetra eco materialize app.tdx -C out --metadata-format both
tetra eco tetrahub mirror --from store-a --to store-b --id tetra://app --version 0.1.0 --target linux-x64 -o mirror.json --format=both
```

The matching validators accept JSON and TOON for these reports:

```sh
go run ./tools/cmd/validate-targets --report targets.toon
go run ./tools/cmd/validate-features --report features.toon
go run ./tools/cmd/validate-formats --report formats.toon
go run ./tools/cmd/validate-doctor --report doctor.toon
go run ./tools/cmd/validate-diagnostic --diagnostic diagnostic.toon
go run ./tools/cmd/validate-test-report --report test-report.toon
go run ./tools/cmd/validate-smoke-list --report smoke-list.toon --format=toon
go run ./tools/cmd/smoke-report-to-checklist --validate-only --report smoke.toon --format=toon
go run ./tools/cmd/validate-lsp-smoke --report lsp-smoke.toon --format=toon
go run ./tools/cmd/validate-manifest --manifest manifest.toon --format=toon
```

TOON input is decoded into the same typed report model used by the JSON
validators. Unknown fields and malformed reports remain validation failures.

HTTP `Accept: text/toon` is supported only on tested Tetra-owned JSON-shaped
runtime endpoints. LSP editor traffic remains JSON-RPC; TOON applies to the
`--stdio-smoke` report, not to `lsp --stdio` frames.

TOON support does not imply token savings, speed improvements, or wholesale
release artifact migration. Those require separate measured evidence before
they can be claimed.
