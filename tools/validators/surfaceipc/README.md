# Surface IPC Lifecycle Validator

`tools/validators/surfaceipc` validates
`tetra.surface.ipc-lifecycle-report.v1` evidence for
`surface-ipc-lifecycle-v1`.

The package owns process role, channel, command, lifecycle, crash isolation,
and restart evidence for the Electron-like app model without depending on
Electron. It rejects missing channel guards, fake process separation, and broad
desktop IPC overclaims.
