# P-A Build / Toolchain / Scripts

Status: completed read-only sub-agent audit; live fixes in progress.

Covered F-IDs:

- `F-0001`
- `F-0002..F-0005`
- `F-0744..F-0748`

Accepted findings:

- `F-0001..F-0005`: live checkout had `t.Chdir(root)` in `tools/cmd/validate-v0-4-readiness/main_test.go:969` while all module directives and CI still declare Go 1.20. Local Go 1.26 does not reproduce the compile failure, but the static incompatibility is real for the declared floor.
- `F-0744`: `scripts/release/v1_0/wasi-smoke.sh` uses `node` without a prerequisite guard.
- `F-0745`: `scripts/release/v1_0/web-smoke.sh` starts `python3 -m http.server`; cleanup is not centralized in the EXIT trap.
- `F-0746`: `scripts/release/v1_0/web-smoke.sh` chooses from fixed ports `8711..8715` with a pre-check race.
- `F-0747`: `scripts/release/v1_0/security-review.sh` requires literal evidence commands containing `<path>`.
- `F-0748`: `scripts/release/v1_0/reproducible-build.sh` exposes legacy `native` evidence only for `linux-x64`.

Evidence:

- `go1.20` is not installed locally; local `go version` is `go1.26.3-X:nodwarf5`.
- RED before fix: `rg "\bt\.Chdir\(" --glob '*_test.go' .` found the readiness helper.
- GREEN after fix: no `t.Chdir` usage and `go test ./tools/cmd/validate-v0-4-readiness` passed.

Uncertainties:

- Full Go 1.20 compilation cannot be run locally without installing that toolchain.
