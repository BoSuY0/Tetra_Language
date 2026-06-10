# Surface Keyboard UX Evidence

Status: experimental Block System evidence layer.

`tetra.surface.keyboard-ux.v1` records scoped keyboard-operability evidence for
Block System reports. The required quality level is
`production-keyboard-ux-v1`.

This schema is the P13 boundary for focus, keyboard activation, shortcut
resolution, navigation, and undo/redo behavior. It does not claim broad
accessibility parity, screen-reader parity, native platform widget behavior, DOM
keyboard events, React keyboard handling, or Electron compatibility.

## Required Evidence

Block System reports that carry this schema must include:

- graph-derived focus order that matches `tetra.surface.block-graph.v1`;
- accessible names or label relationships for every keyboard-reachable
  focusable Block;
- Tab and Shift+Tab focus transitions, including wrap evidence;
- overlay focus traps with leak rejection and Escape-close behavior;
- roving focus groups with arrow, Home/End, and wrap behavior;
- keyboard activation bindings for Enter and Space;
- scoped shortcuts including global command palette, editor undo, and editor
  redo bindings;
- diagnosed and rejected shortcut conflicts;
- bounded keyboard-driven undo/redo stacks;
- keyboard-only scripts for command palette, search, settings forms, and editor
  shell surfaces.

## Negative Guards

Validators reject:

- focusable elements without accessible names or label relationships;
- overlays that allow focus to leak outside their focus trap;
- shortcut conflicts that are not diagnosed and rejected;
- pointer-only actions that cannot be reached by keyboard;
- unknown shortcuts that silently dispatch;
- undo/redo commands without a bounded action stack.

## Tetra API Surface

`lib.core.block` exposes compact helpers for this evidence:

- `KeyboardFocusNode` plus `keyboard_focus_node_valid`;
- `KeyboardBinding` plus `keyboard_binding_valid`;
- `KeyboardShortcutConflict` plus `keyboard_shortcut_conflict_valid`;
- `KeyboardUndoRedoStack` plus `keyboard_undo_redo_valid`;
- `KeyboardScript` plus `keyboard_script_valid`;
- `keyboard_focus_trap_valid`;
- `keyboard_roving_group_valid`;
- keyboard key constants such as `keyboard_key_tab`,
  `keyboard_key_ctrl_k`, `keyboard_key_ctrl_z`, and `keyboard_key_ctrl_y`.

These helpers are Block/Morph app-model helpers. They are not React hooks, DOM
event listeners, browser CSS focus behavior, or platform-native widgets.
