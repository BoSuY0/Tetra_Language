# Tetra v0.1.x Test Report

- mode: `full`
- status: `pass`
- started_at: `2026-04-27T10:32:30Z`
- ended_at: `2026-04-27T10:32:32Z`
- step_count: `23`
- failed_count: `0`

## Steps

- `go test all packages`: `pass` in 1s, exit `0`, command `go test ./compiler/... ./cli/... ./tools/...` ([logs/01-go-test-all-packages.log](logs/01-go-test-all-packages.log))
- `repo test script`: `pass` in 0s, exit `0`, command `bash scripts/test.sh` ([logs/02-repo-test-script.log](logs/02-repo-test-script.log))
- `bootstrap`: `pass` in 0s, exit `0`, command `bash scripts/bootstrap.sh` ([logs/03-bootstrap.log](logs/03-bootstrap.log))
- `version prefix`: `pass` in 0s, exit `0`, command `check_version_prefix` ([logs/04-version-prefix.log](logs/04-version-prefix.log))
- `short alias version`: `pass` in 0s, exit `0`, command `check_short_alias_version` ([logs/05-short-alias-version.log](logs/05-short-alias-version.log))
- `formatter check examples lib runtime`: `pass` in 0s, exit `0`, command `./tetra fmt --check examples lib __rt compiler/selfhostrt` ([logs/06-formatter-check-examples-lib-runtime.log](logs/06-formatter-check-examples-lib-runtime.log))
- `flow-only source scan`: `pass` in 0s, exit `0`, command `go run ./tools/cmd/validate-flow-only examples lib __rt compiler/selfhostrt` ([logs/07-flow-only-source-scan.log](logs/07-flow-only-source-scan.log))
- `targets json report`: `pass` in 0s, exit `0`, command `check_targets_report` ([logs/08-targets-json-report.log](logs/08-targets-json-report.log))
- `doctor json report`: `pass` in 0s, exit `0`, command `check_doctor_report` ([logs/09-doctor-json-report.log](logs/09-doctor-json-report.log))
- `tetra check flow hello`: `pass` in 0s, exit `0`, command `./tetra check examples/flow_hello.tetra` ([logs/10-tetra-check-flow-hello.log](logs/10-tetra-check-flow-hello.log))
- `json diagnostic shape`: `pass` in 0s, exit `0`, command `check_json_diagnostic` ([logs/11-json-diagnostic-shape.log](logs/11-json-diagnostic-shape.log))
- `smoke list json report`: `pass` in 0s, exit `0`, command `check_smoke_list` ([logs/12-smoke-list-json-report.log](logs/12-smoke-list-json-report.log))
- `tetra test examples`: `pass` in 0s, exit `0`, command `./tetra test examples` ([logs/13-tetra-test-examples.log](logs/13-tetra-test-examples.log))
- `tetra test json report`: `pass` in 0s, exit `0`, command `check_test_json` ([logs/14-tetra-test-json-report.log](logs/14-tetra-test-json-report.log))
- `host smoke linux-x64`: `pass` in 1s, exit `0`, command `check_host_smoke` ([logs/15-host-smoke-linux-x64.log](logs/15-host-smoke-linux-x64.log))
- `docs manifest diff`: `pass` in 0s, exit `0`, command `check_docs_manifest` ([logs/16-docs-manifest-diff.log](logs/16-docs-manifest-diff.log))
- `docs verification`: `pass` in 0s, exit `0`, command `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json` ([logs/17-docs-verification.log](logs/17-docs-verification.log))
- `lsp stdio smoke`: `pass` in 0s, exit `0`, command `check_lsp_smoke` ([logs/18-lsp-stdio-smoke.log](logs/18-lsp-stdio-smoke.log))
- `lsp json-rpc stdio`: `pass` in 0s, exit `0`, command `check_lsp_stdio` ([logs/19-lsp-json-rpc-stdio.log](logs/19-lsp-json-rpc-stdio.log))
- `tetra doc examples`: `pass` in 0s, exit `0`, command `check_tetra_doc` ([logs/20-tetra-doc-examples.log](logs/20-tetra-doc-examples.log))
- `generated api docs`: `pass` in 0s, exit `0`, command `check_generated_api_docs` ([logs/21-generated-api-docs.log](logs/21-generated-api-docs.log))
- `eco graph bundle vault`: `pass` in 0s, exit `0`, command `check_eco_suite` ([logs/22-eco-graph-bundle-vault.log](logs/22-eco-graph-bundle-vault.log))
- `cross-target build smoke`: `pass` in 0s, exit `0`, command `check_cross_target_smoke` ([logs/23-cross-target-build-smoke.log](logs/23-cross-target-build-smoke.log))
