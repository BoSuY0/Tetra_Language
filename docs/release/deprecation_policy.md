# Deprecation Policy

Status: current compatibility policy for documented public surfaces.

## Deprecation Policy

A deprecation announces that a public surface remains available for now but
should be replaced. Deprecation is not removal.

Every deprecation needs:

- a replacement path;
- diagnostics or documentation that points to the replacement path;
- release notes naming the affected surface and expected timeline;
- tests or validators proving the old path is still handled intentionally.

## Removal Rules

For `v1.0.x`, removals wait for a later minor or major line unless a security
fix requires a documented exception.

For stable `lib.core.*` APIs, incompatible removals or signature changes wait
for the next major release line. Additive APIs and clarifications may happen in
compatible lines when tests and docs are updated.

Experimental surfaces may change faster only when docs explicitly mark them as
experimental and no stable release contract claims them.

## Release Notes

Release notes for a deprecation must include:

- the deprecated name or command;
- the replacement path;
- the first release where the deprecation appears;
- the earliest release line where removal may be considered;
- migration examples when user code or manifests need edits.

## Compatibility Gate

Patch-line changes must not remove documented public behavior without a
security exception, migration notes, and focused verification evidence.
