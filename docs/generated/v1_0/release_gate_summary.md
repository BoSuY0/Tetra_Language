# Tetra v0.1.1 Release Gate Report

- status: `pass`
- started_at: `2026-04-27T10:32:28Z`
- ended_at: `2026-04-27T10:32:36Z`
- step_count: `33`
- failed_count: `0`
- report_dir: `/tmp/tetra-v0_1_1-final-release-gate-20260427`

## Steps

- `bootstrap tetra binaries`: `pass` in 0s, exit `0`, command `bash scripts/bootstrap.sh` ([logs/01-bootstrap-tetra-binaries.log](logs/01-bootstrap-tetra-binaries.log))
- `version preflight (v0.1.1 required)`: `pass` in 0s, exit `0`, command `check_release_version` ([logs/02-version-preflight-v0-1-1-required.log](logs/02-version-preflight-v0-1-1-required.log))
- `short alias version parity`: `pass` in 0s, exit `0`, command `check_short_alias_version` ([logs/03-short-alias-version-parity.log](logs/03-short-alias-version-parity.log))
- `go test packages`: `pass` in 2s, exit `0`, command `go test ./compiler/... ./cli/... ./tools/... -count=1` ([logs/04-go-test-packages.log](logs/04-go-test-packages.log))
- `full stabilization wrapper`: `pass` in 2s, exit `0`, command `bash scripts/test_all.sh --full --keep-going --report-dir /tmp/tetra-v0_1_1-final-release-gate-20260427/artifacts/test-all` ([logs/05-full-stabilization-wrapper.log](logs/05-full-stabilization-wrapper.log))
- `flow-only source scan`: `pass` in 0s, exit `0`, command `go run ./tools/cmd/validate-flow-only examples lib __rt compiler/selfhostrt` ([logs/06-flow-only-source-scan.log](logs/06-flow-only-source-scan.log))
- `targets report validation`: `pass` in 0s, exit `0`, command `sh -c ./tetra targets --format=json >"$1" && go run ./tools/cmd/validate-targets --report "$1" sh /tmp/tetra-v0_1_1-final-release-gate-20260427/artifacts/targets.json` ([logs/07-targets-report-validation.log](logs/07-targets-report-validation.log))
- `doctor report validation`: `pass` in 0s, exit `0`, command `sh -c ./tetra doctor --format=json >"$1" && go run ./tools/cmd/validate-doctor --report "$1" sh /tmp/tetra-v0_1_1-final-release-gate-20260427/artifacts/doctor.json` ([logs/08-doctor-report-validation.log](logs/08-doctor-report-validation.log))
- `tetra check flow hello`: `pass` in 0s, exit `0`, command `./tetra check examples/flow_hello.tetra` ([logs/09-tetra-check-flow-hello.log](logs/09-tetra-check-flow-hello.log))
- `formatter check`: `pass` in 0s, exit `0`, command `./tetra fmt --check examples lib __rt compiler/selfhostrt` ([logs/10-formatter-check.log](logs/10-formatter-check.log))
- `tetra test examples json`: `pass` in 0s, exit `0`, command `sh -c ./tetra test --report=json examples >"$1" && go run ./tools/cmd/validate-test-report --report "$1" sh /tmp/tetra-v0_1_1-final-release-gate-20260427/artifacts/tetra-test-report.json` ([logs/11-tetra-test-examples-json.log](logs/11-tetra-test-examples-json.log))
- `docs manifest regenerate+validate`: `pass` in 0s, exit `0`, command `check_docs_manifest` ([logs/12-docs-manifest-regenerate-validate.log](logs/12-docs-manifest-regenerate-validate.log))
- `docs verification and doctests`: `pass` in 0s, exit `0`, command `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json` ([logs/13-docs-verification-and-doctests.log](logs/13-docs-verification-and-doctests.log))
- `tetra doc output validation`: `pass` in 0s, exit `0`, command `check_tetra_doc_output` ([logs/14-tetra-doc-output-validation.log](logs/14-tetra-doc-output-validation.log))
- `json diagnostic shape`: `pass` in 0s, exit `0`, command `check_json_diagnostic` ([logs/15-json-diagnostic-shape.log](logs/15-json-diagnostic-shape.log))
- `smoke list validation`: `pass` in 0s, exit `0`, command `sh -c ./tetra smoke --list --format=json >"$1" && go run ./tools/cmd/validate-smoke-list --report "$1" --examples-root examples sh /tmp/tetra-v0_1_1-final-release-gate-20260427/artifacts/smoke-list.json` ([logs/16-smoke-list-validation.log](logs/16-smoke-list-validation.log))
- `native host smoke linux-x64`: `pass` in 0s, exit `0`, command `sh -c ./tetra smoke --target linux-x64 --run=true --report "$1" && go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$1" sh /tmp/tetra-v0_1_1-final-release-gate-20260427/artifacts/host-smoke.json` ([logs/17-native-host-smoke-linux-x64.log](logs/17-native-host-smoke-linux-x64.log))
- `build-only smoke linux-x64`: `pass` in 0s, exit `0`, command `sh -c ./tetra smoke --target linux-x64 --run=false --report "$1" && go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$1" sh /tmp/tetra-v0_1_1-final-release-gate-20260427/artifacts/linux-smoke.json` ([logs/18-build-only-smoke-linux-x64.log](logs/18-build-only-smoke-linux-x64.log))
- `build-only smoke macos-x64`: `pass` in 0s, exit `0`, command `sh -c ./tetra smoke --target macos-x64 --run=false --report "$1" && go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$1" sh /tmp/tetra-v0_1_1-final-release-gate-20260427/artifacts/macos-smoke.json` ([logs/19-build-only-smoke-macos-x64.log](logs/19-build-only-smoke-macos-x64.log))
- `build-only smoke windows-x64`: `pass` in 0s, exit `0`, command `sh -c ./tetra smoke --target windows-x64 --run=false --report "$1" && go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$1" sh /tmp/tetra-v0_1_1-final-release-gate-20260427/artifacts/windows-smoke.json` ([logs/20-build-only-smoke-windows-x64.log](logs/20-build-only-smoke-windows-x64.log))
- `build-only smoke wasm32-wasi`: `pass` in 0s, exit `0`, command `sh -c ./tetra smoke --target wasm32-wasi --run=false --report "$1" && go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$1" sh /tmp/tetra-v0_1_1-final-release-gate-20260427/artifacts/wasm32-wasi-smoke.json` ([logs/21-build-only-smoke-wasm32-wasi.log](logs/21-build-only-smoke-wasm32-wasi.log))
- `build-only smoke wasm32-web`: `pass` in 0s, exit `0`, command `sh -c ./tetra smoke --target wasm32-web --run=false --report "$1" && go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$1" sh /tmp/tetra-v0_1_1-final-release-gate-20260427/artifacts/wasm32-web-smoke.json` ([logs/22-build-only-smoke-wasm32-web.log](logs/22-build-only-smoke-wasm32-web.log))
- `WASI runner smoke`: `pass` in 1s, exit `0`, command `check_wasi_runner_smoke` ([logs/23-wasi-runner-smoke.log](logs/23-wasi-runner-smoke.log))
- `Web UI browser smoke`: `pass` in 1s, exit `0`, command `check_web_ui_smoke` ([logs/24-web-ui-browser-smoke.log](logs/24-web-ui-browser-smoke.log))
- `security review signoff`: `pass` in 0s, exit `0`, command `check_security_review_signoff` ([logs/25-security-review-signoff.log](logs/25-security-review-signoff.log))
- `API diff gate`: `pass` in 0s, exit `0`, command `check_api_diff` ([logs/26-api-diff-gate.log](logs/26-api-diff-gate.log))
- `binary size thresholds`: `pass` in 0s, exit `0`, command `check_binary_size_thresholds` ([logs/27-binary-size-thresholds.log](logs/27-binary-size-thresholds.log))
- `reproducible build proof`: `pass` in 0s, exit `0`, command `check_repro_build` ([logs/28-reproducible-build-proof.log](logs/28-reproducible-build-proof.log))
- `eco verify command surface`: `pass` in 0s, exit `0`, command `sh -c test -x ./tetra && ./tetra eco verify --help >/dev/null` ([logs/29-eco-verify-command-surface.log](logs/29-eco-verify-command-surface.log))
- `release state audit`: `pass` in 1s, exit `0`, command `check_release_state` ([logs/30-release-state-audit.log](logs/30-release-state-audit.log))
- `known issues artifact`: `pass` in 0s, exit `0`, command `write_known_issues_artifact` ([logs/31-known-issues-artifact.log](logs/31-known-issues-artifact.log))
- `artifact hash manifest`: `pass` in 0s, exit `0`, command `check_artifact_hash_manifest` ([logs/32-artifact-hash-manifest.log](logs/32-artifact-hash-manifest.log))
- `generated artifact churn check`: `pass` in 0s, exit `0`, command `check_generated_artifact_churn` ([logs/33-generated-artifact-churn-check.log](logs/33-generated-artifact-churn-check.log))
