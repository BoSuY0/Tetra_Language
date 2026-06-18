# Surface Security And Sandbox Evidence

Status: experimental security and sandbox evidence for the scoped Surface
production track. This does not claim a browser plugin sandbox, Node/Electron
process sandbox parity, remote-code execution support, or arbitrary untrusted
asset decoding.

`tetra.surface.security-report.v1` records the security boundary for Surface
apps that declare host permissions and run through validator-enforced sandbox
rules. The current level is `surface-security-sandbox-v1` under
`surface-v1-scoped-linux-web-security`.

The supported gate is:

```sh
bash scripts/release/surface/security-gate.sh \
  --report-dir reports/surface-prod/P27-security-gate
```

Required evidence:

- permission model is `explicit-deny-by-default`;
- permissions are declared for filesystem, network, clipboard, window,
  open-url, and notifications;
- filesystem access is scoped to app-bundle read-only evidence;
- network/open-url/notifications are denied by default;
- clipboard/window access is host-gated and requires diagnostics/traces;
- asset sandbox is `safe-local-assets-only`;
- font, image, and SVG assets are hash-verified before decoder evidence;
- SVG is limited to sanitized static SVG Tiny evidence;
- typed IPC is `typed-host-abi-only`;
- user JavaScript bridges, raw eval, and remote code execution are rejected;
- package/capsule supply-chain evidence requires capsule verification, package
  hashes, lockfile policy, and no postinstall scripts.

Fake-claim rejection cases:

- network/filesystem/clipboard host calls without permission;
- untrusted SVG/font/image acceptance;
- user JavaScript introduced as a Surface runtime path;
- untyped IPC channel;
- package/capsule evidence without hashes.

This security report composes P15 app-shell permissions, P21 asset pipeline
guards, and P26 package hashes into one release gate. P28 owns the deeper IPC
process/lifecycle model; P27 only forbids unsafe IPC and remote-code claims.
