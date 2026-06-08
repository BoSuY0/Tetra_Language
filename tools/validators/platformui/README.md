# tools/validators/platformui

Validator package for cross-platform platform UI runtime evidence.

This boundary owns the `tetra.ui.platform-runtime.v1` report contract for
Windows and macOS target-host UI runtime smokes. Passing evidence must come
from a real target runner, not metadata-only, build-only, placeholder, or
runtime-less reports. Release gates pass the expected Tetra version and current
Git commit into the validator so copied CI fan-in reports cannot be stale. The
report must also include `runtime_trace` markers for platform process spawn,
an OS-backed platform window API probe, platform widget tree construction,
platform event dispatch, platform timer/redraw work, window create/show/close,
widget-tree load, layout measure/place, event loop start,
focus/input/select/click dispatch, state update, async command, timer tick,
redraw, and error recovery. Current target-host probes use Win32 `user32.dll`
controls and messages for Windows and an AppKit window/control probe compiled
with `swiftc` for macOS.
