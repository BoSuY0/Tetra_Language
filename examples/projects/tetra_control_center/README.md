# Tetra Control Center

Tetra-first laptop control center for the Dream Machines
`V3xxSNP_SNN_SNM` laptop. The app keeps the UI state and screen/profile
contract in Tetra, uses a small web host for the current rich layout gap, and
uses a safe Python helper for Linux system discovery and allowlisted profile
operations.

## Current Implementation Checklist

- [x] Tetra project/workspace under `examples/projects/tetra_control_center`.
- [x] Tetra UI state/view contract for Dashboard, Profiles, Fans/Backends,
  Diagnostics, Logs, and Settings.
- [x] Dashboard API data for CPU, GPU, RAM, battery, sensors, power profile,
  governor/EPP, and backend support status.
- [x] Profiles: Quiet, Balanced, Performance, and Custom.
- [x] Safe adapters for `powerprofilesctl`, CPU governor/EPP sysfs discovery,
  `hwmon`, `nvidia-smi`, NBFC-Linux status, and TUXEDO/TCC diagnostics.
- [x] Helper defaults to read-only/dry-run, validates inputs, has no arbitrary
  shell executor, and writes JSON audit log entries for profile attempts.
- [x] Smoke checks for Tetra UI, diagnostics discovery, API behavior, and safe
  failure modes.
- [ ] Fan/RGB control. This is intentionally not claimed: this laptop exposes
  read-only fan RPM sensors but no validated fan control backend.

## Run

From the Tetra repository root:

```sh
cd /home/tetra/Desktop/Projects/Tetra_Language
./tetra build -target wasm32-web \
  -o examples/projects/tetra_control_center/build/tetra_control_center.wasm \
  examples/projects/tetra_control_center
cd examples/projects/tetra_control_center
python3 -m backend.tcc_backend \
  --host 127.0.0.1 \
  --port 8765 \
  --project-root . \
  --audit-log "$HOME/.local/state/tetra-control-center/audit.jsonl"
```

Open:

```text
http://127.0.0.1:8765/
```

Or from the operator workspace:

```sh
cd /home/tetra/Desktop/Projects/TetraControlCenter
./run.sh
```

The helper starts read-only by default. Profile buttons run as dry-run unless
the UI dry-run toggle is disabled and the helper was started with
`--allow-writes`.

## Smoke

```sh
cd /home/tetra/Desktop/Projects/Tetra_Language
bash examples/projects/tetra_control_center/scripts/smoke.sh
```

The smoke script runs Python unit tests, `./tetra check`, a `wasm32-web` build,
snapshot/API checks, a dry-run profile request, and Chromium DOM checks for all
six screens when `chromium` is installed.

## Files

- `src/main.tetra` - Tetra UI state, screens, profiles, and command contract.
- `backend/tcc_backend.py` - safe read-only discovery API and allowlisted
  profile helper.
- `web/` - current web host shell that renders the Tetra UI contract plus live
  backend snapshot data.
- `tests/` - backend and UI contract tests.
- `docs/architecture.md` - component and data-flow notes.
- `docs/privileged-helper-threat-model.md` - write boundary and threat model.
- `docs/hardware-support-matrix.md` - evidence-backed hardware support status.
- `docs/final-report.md` - implemented features and verification evidence.
