# Surface Security Permission Model Design

Status: approved by the active `surface-electron-react-beauty-production`
Goal plan and continuation directive. This document narrows
`SURFACE-BEAUTY-P17` into a repo-local validation contract.

## Observed Facts

- `tetra.surface.linux-app-shell.v1` already records the P16 app-shell feature
  ledger for lifecycle, multi-window, clipboard, IME, accessibility, scoped app
  menu, crash recovery, error reporting, and blocked-pass nonclaims for dialogs,
  file picker/dialogs, notifications, tray, and deep links.
- `tetra.surface.browser-surface.v1` already uses canonical browser guards:
  `dom_host_canvas_only`, `no_dom_app_ui_tree`, `no_user_js_app_logic`, and
  `no_node_only_promotion`.
- Release summaries currently require `electron-feature-ledger-v1`, but do not
  require a security/permission model row.
- Release state validation currently requires the Linux app-shell runtime
  report, but does not require app-shell permission evidence.

## Design

Add `tetra.surface.security-permission.v1` as an embedded runtime report section
named `security_permissions`. The report is mandatory for Linux app-shell
release evidence and is validated by both the generic Surface runtime validator
and a dedicated `validate-surface-security-report` CLI.

The model is default-deny. Filesystem, network, shell/open-url, notifications,
dialogs, file picker/dialogs, tray, and deep links remain denied unless target
host evidence and explicit capability rows exist. Clipboard remains allowed only
through the existing host ABI evidence. App-shell feature rows cannot claim a
blocked P16 feature unless the permission model also changes with matching
target evidence, which P17 does not add.

The IPC/process boundary is explicit: the Surface app talks to a host ABI, the
Linux app-shell adapter, and the browser canvas host through schema-checked,
capability-checked messages. User JavaScript app logic, Node integration, and
Electron runtime integration remain false.

Asset safety is explicit for font, image, and icon inputs. Release evidence must
show local-only assets, SHA-256 requirements, size limits, parser/bounds checks,
and network asset fetch denial.

## Validation Strategy

- RED tests require `security_permissions` in app-shell runtime reports.
- RED tests reject permission rows that allow blocked P16 features.
- Release summaries must carry `security_permissions:
  surface-security-permission-v1`.
- `validate-surface-security-report` validates the embedded report directly from
  the Linux app-shell runtime JSON.
- Release scripts run the dedicated validator before artifact hashes and release
  state validation.

## Nonclaims

P17 does not add unrestricted filesystem/network access, native permission
prompts, production notifications/dialogs, remote asset fetch, Electron runtime,
Node integration, user JavaScript app logic, or DOM-authored app UI.
