# Tetra v1.0 Release Gate Report

- status: `blocked`
- started_at: `2026-04-26T19:56:27Z`
- ended_at: `2026-04-26T19:56:33Z`
- step_count: `26`
- failed_count: `4`
- report_dir: `/tmp/release-v1_0-gate-20260426-195627`

## Steps

- `go test packages`: `fail` in 1s, exit `1`, command `go test ./compiler/... ./cli/... ./tools/... -count=1` ([logs/01-go-test-packages.log](logs/01-go-test-packages.log))
- `bootstrap tetra binaries`: `pass` in 0s, exit `0`, command `bash scripts/bootstrap.sh` ([logs/02-bootstrap-tetra-binaries.log](logs/02-bootstrap-tetra-binaries.log))
- `version preflight (v1.0 required)`: `fail` in 0s, exit `1`, command `check_v1_version` ([logs/03-version-preflight-v1-0-required.log](logs/03-version-preflight-v1-0-required.log))
- `short alias version parity`: `pass` in 0s, exit `0`, command `check_short_alias_version` ([logs/04-short-alias-version-parity.log](logs/04-short-alias-version-parity.log))
- `full stabilization wrapper`: `fail` in 2s, exit `1`, command `bash scripts/test_all.sh --full --keep-going --report-dir /tmp/release-v1_0-gate-20260426-195627/artifacts/test-all` ([logs/05-full-stabilization-wrapper.log](logs/05-full-stabilization-wrapper.log))
- `flow-only source scan`: `pass` in 0s, exit `0`, command `go run ./tools/cmd/validate-flow-only examples lib __rt compiler/selfhostrt` ([logs/06-flow-only-source-scan.log](logs/06-flow-only-source-scan.log))
- `targets report validation`: `pass` in 0s, exit `0`, command `sh -c ./tetra targets --format=json >"$1" && go run ./tools/cmd/validate-targets --report "$1" sh /tmp/release-v1_0-gate-20260426-195627/artifacts/targets.json` ([logs/07-targets-report-validation.log](logs/07-targets-report-validation.log))
- `doctor report validation`: `pass` in 0s, exit `0`, command `sh -c ./tetra doctor --format=json >"$1" && go run ./tools/cmd/validate-doctor --report "$1" sh /tmp/release-v1_0-gate-20260426-195627/artifacts/doctor.json` ([logs/08-doctor-report-validation.log](logs/08-doctor-report-validation.log))
- `tetra check flow hello`: `pass` in 0s, exit `0`, command `./tetra check examples/flow_hello.tetra` ([logs/09-tetra-check-flow-hello.log](logs/09-tetra-check-flow-hello.log))
- `formatter check`: `pass` in 0s, exit `0`, command `./tetra fmt --check examples lib __rt compiler/selfhostrt` ([logs/10-formatter-check.log](logs/10-formatter-check.log))
- `tetra test examples json`: `pass` in 0s, exit `0`, command `sh -c ./tetra test --report=json examples >"$1" && go run ./tools/cmd/validate-test-report --report "$1" sh /tmp/release-v1_0-gate-20260426-195627/artifacts/tetra-test-report.json` ([logs/11-tetra-test-examples-json.log](logs/11-tetra-test-examples-json.log))
- `docs manifest regenerate+validate`: `pass` in 0s, exit `0`, command `check_docs_manifest` ([logs/12-docs-manifest-regenerate-validate.log](logs/12-docs-manifest-regenerate-validate.log))
- `docs verification and doctests`: `pass` in 1s, exit `0`, command `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json` ([logs/13-docs-verification-and-doctests.log](logs/13-docs-verification-and-doctests.log))
- `tetra doc output validation`: `pass` in 0s, exit `0`, command `sh -c ./tetra doc examples >"$1" && go run ./tools/cmd/validate-api-docs --docs "$1" sh /tmp/release-v1_0-gate-20260426-195627/artifacts/tetra-docs.md` ([logs/14-tetra-doc-output-validation.log](logs/14-tetra-doc-output-validation.log))
- `smoke list validation`: `pass` in 0s, exit `0`, command `sh -c ./tetra smoke --list --format=json >"$1" && go run ./tools/cmd/validate-smoke-list --report "$1" sh /tmp/release-v1_0-gate-20260426-195627/artifacts/smoke-list.json` ([logs/15-smoke-list-validation.log](logs/15-smoke-list-validation.log))
- `native host smoke linux-x64`: `pass` in 0s, exit `0`, command `sh -c ./tetra smoke --target linux-x64 --run=true --report "$1" && go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$1" sh /tmp/release-v1_0-gate-20260426-195627/artifacts/host-smoke.json` ([logs/16-native-host-smoke-linux-x64.log](logs/16-native-host-smoke-linux-x64.log))
- `build-only smoke linux-x64`: `pass` in 0s, exit `0`, command `sh -c ./tetra smoke --target linux-x64 --run=false --report "$1" && go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$1" sh /tmp/release-v1_0-gate-20260426-195627/artifacts/linux-smoke.json` ([logs/17-build-only-smoke-linux-x64.log](logs/17-build-only-smoke-linux-x64.log))
- `build-only smoke macos-x64`: `pass` in 0s, exit `0`, command `sh -c ./tetra smoke --target macos-x64 --run=false --report "$1" && go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$1" sh /tmp/release-v1_0-gate-20260426-195627/artifacts/macos-smoke.json` ([logs/18-build-only-smoke-macos-x64.log](logs/18-build-only-smoke-macos-x64.log))
- `build-only smoke windows-x64`: `pass` in 0s, exit `0`, command `sh -c ./tetra smoke --target windows-x64 --run=false --report "$1" && go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$1" sh /tmp/release-v1_0-gate-20260426-195627/artifacts/windows-smoke.json` ([logs/19-build-only-smoke-windows-x64.log](logs/19-build-only-smoke-windows-x64.log))
- `build-only smoke wasm32-wasi`: `pass` in 0s, exit `0`, command `sh -c ./tetra smoke --target wasm32-wasi --run=false --report "$1" && go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$1" sh /tmp/release-v1_0-gate-20260426-195627/artifacts/wasm32-wasi-smoke.json` ([logs/20-build-only-smoke-wasm32-wasi.log](logs/20-build-only-smoke-wasm32-wasi.log))
- `build-only smoke wasm32-web`: `pass` in 0s, exit `0`, command `sh -c ./tetra smoke --target wasm32-web --run=false --report "$1" && go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$1" sh /tmp/release-v1_0-gate-20260426-195627/artifacts/wasm32-web-smoke.json` ([logs/21-build-only-smoke-wasm32-web.log](logs/21-build-only-smoke-wasm32-web.log))
- `WASI runner smoke`: `pass` in 0s, exit `0`, command `check_wasi_runner_smoke` ([logs/22-wasi-runner-smoke.log](logs/22-wasi-runner-smoke.log))
- `Web UI browser smoke`: `fail` in 2s, exit `1`, command `check_web_ui_smoke` ([logs/23-web-ui-browser-smoke.log](logs/23-web-ui-browser-smoke.log))
- `API diff gate`: `pass` in 0s, exit `0`, command `check_api_diff` ([logs/24-api-diff-gate.log](logs/24-api-diff-gate.log))
- `reproducible build proof`: `pass` in 0s, exit `0`, command `check_repro_build` ([logs/25-reproducible-build-proof.log](logs/25-reproducible-build-proof.log))
- `eco verify command surface`: `pass` in 0s, exit `0`, command `sh -c test -x ./tetra && ./tetra eco verify --help >/dev/null` ([logs/26-eco-verify-command-surface.log](logs/26-eco-verify-command-surface.log))
