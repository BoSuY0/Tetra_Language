# Surface Security Validator

`tools/validators/surfacesecurity` validates
`tetra.surface.security-report.v1` evidence for
`surface-security-sandbox-v1`.

The package owns permission, filesystem, network, clipboard, window,
open-URL, notification, asset/font/image, IPC, and supply-chain security
checks. It rejects broad sandbox or permission claims without scoped host
evidence.
