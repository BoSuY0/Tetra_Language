# Native UI Runtime Validator

This package validates Linux-x64 native UI runtime smoke reports.

The accepted report schema is `tetra.ui.native-runtime.v1`. It is deliberately
stricter than the native shell sidecar schema: a passing report must show an
executable runtime process, widget hierarchy, event dispatch, state transition,
widget update propagation, lifecycle coverage, and negative cases.
