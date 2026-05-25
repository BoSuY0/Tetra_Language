# Privileged Helper Threat Model

## Assets

- CPU power policy files under `/sys/devices/system/cpu/**/cpufreq`.
- `powerprofilesctl` profile state.
- Audit log at the configured `--audit-log` path.
- Hardware diagnostic data read from `/proc` and `/sys`.

## Default Mode

The helper is read-only by default. Without `--allow-writes`, `POST /api/profile`
returns a dry-run plan and does not execute `powerprofilesctl set` or write
sysfs files.

## Allowed Write Operations

When `--allow-writes` is present, the helper may perform only these operations:

- `powerprofilesctl set power-saver|balanced|performance`.
- Write `performance`, `powersave`, `schedutil`, `ondemand`, or `conservative`
  to discovered `scaling_governor` files.
- Write `performance`, `balance_performance`, `balance_power`, `power`, or
  `default` to discovered `energy_performance_preference` files.

The helper never writes EC registers, raw fan registers, RGB registers,
`/sys/kernel/debug`, arbitrary sysfs paths, or files outside discovered CPU
policy directories.

## Input Validation

- Profile names are normalized and must be one of `quiet`, `balanced`,
  `performance`, or `custom`.
- Sysfs write targets must resolve to a discovered cpufreq policy file.
- Sysfs values must match the allowlist for that file type.
- Static file serving rejects path traversal with `Path.relative_to`.
- API routing is explicit; unknown paths return `404`.

## Command Execution

The helper uses `subprocess.run(argv, shell=False)` with fixed argv lists. It
does not accept command strings from the UI or API.

## Audit Log

Every profile attempt writes a JSON line with timestamp, action, profile,
dry-run mode, write mode, decision, planned operations, and result. Rejected
profiles are logged with `decision: deny`.

## Known Non-Goals

- No kernel driver.
- No TUXEDO driver patching.
- No EC/fan/RGB register writes.
- No fan curve writes through raw hwmon PWM.
- No root escalation mechanism. If write mode is needed, the helper must be
  launched by an operator under an appropriate service policy.
