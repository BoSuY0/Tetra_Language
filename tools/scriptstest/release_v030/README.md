# v0.3.0 Release Gate Tests

This directory owns the runnable v0.3.0 release gate tests for runtime smoke
evidence and security signoff evidence.

The package is intentionally local to this release gate so runtime smoke and
security signoff fixtures can evolve without depending on the broader
`tools/scriptstest` test helper namespace.
