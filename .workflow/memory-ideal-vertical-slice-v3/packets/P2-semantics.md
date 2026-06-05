# P2 Semantics Packet

## Scope

Add focused semantics tests for interface/protocol/static-conformance borrow
boundary behavior using only already-supported syntax and checker behavior.

## Required Coverage

- positive known/static target local borrowed use;
- borrowed owned-return escape through interface/protocol path rejected;
- borrowed global-storage escape through interface/protocol path rejected;
- unknown/dynamic dispatch conservative rejection or no trusted fact path;
- broad noalias through protocol/interface dispatch rejected.

## Acceptance

Accepted only after the focused semantics regex gate passes and any unsupported
runtime protocol/existential behavior is documented as conservative rather than
implemented out of scope.
