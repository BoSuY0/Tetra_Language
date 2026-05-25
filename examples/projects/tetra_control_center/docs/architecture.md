# Architecture

## Boundary

Tetra owns the application state contract. `src/main.tetra` declares the
Control Center state, six screen names, profile choices, bindings, and commands.
Building the project for `wasm32-web` emits the Tetra UI bundle at
`build/tetra_control_center.ui.json`.

The current Tetra UI surface emits deterministic metadata and a generic preview
shell. It does not yet provide a rich dashboard layout engine or direct Linux
system integration. For that reason this example adds two narrow sidecars:

- `web/app.mjs` renders the real six-screen web UI from the Tetra UI bundle and
  helper snapshot data.
- `backend/tcc_backend.py` reads Linux system state and exposes a small local
  JSON API.

The sidecars are intentionally small and are documented as boundaries rather
than hidden Tetra capabilities.

## Data Flow

1. `./tetra build -target wasm32-web ... examples/projects/tetra_control_center`
   validates `src/main.tetra` and emits the Tetra UI bundle.
2. The helper serves `/`, `/web/*`, `/build/*`, and allowlisted `/api/*`.
3. The web host loads `/build/tetra_control_center.ui.json` and derives the
   navigation/profile contract from the Tetra `screen*` and `profile*` state
   fields.
4. The web host polls `/api/snapshot` for CPU, GPU, memory, battery, sensors,
   power profile, diagnostics, support matrix, logs, and settings.
5. Profile actions call `POST /api/profile` with `{profile, dry_run}`.
6. The helper validates the profile and either records a dry-run plan or, only
   when started with `--allow-writes`, applies allowlisted operations.

## API

- `GET /api/health` - helper status and write mode.
- `GET /api/snapshot` - dashboard, diagnostics, support matrix, logs, settings.
- `GET /api/logs` - recent audit log entries.
- `POST /api/profile` - apply or dry-run `quiet`, `balanced`, `performance`, or
  `custom`.

There is no generic command endpoint and no shell string executor.

## Safe Adapters

- `powerprofilesctl get/list/set` through argv-only subprocess calls.
- CPU governor/EPP discovery from `cpufreq` policy files.
- CPU governor/EPP writes only to discovered policy
  `scaling_governor`/`energy_performance_preference` files, only with allowlisted
  values, and only with `--allow-writes`.
- Sensors from `/sys/class/hwmon`, read-only.
- NVIDIA from `nvidia-smi --query-gpu`, read-only.
- NBFC-Linux status only if `nbfc` or `nbfc-linux` exists and `status -a`
  succeeds.
- TUXEDO/TCC diagnostics from DMI, modules, command presence, and sysfs
  interface discovery.

## Error Handling

Unavailable commands, unreadable sysfs files, missing batteries, failed NVIDIA
queries, and failed NBFC status calls are represented as `unsupported` with a
reason. The UI shows supported and unsupported paths side by side.
