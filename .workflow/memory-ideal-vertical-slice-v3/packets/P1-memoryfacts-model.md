# P1 MemoryFacts And MiniMemoryModel Packet

## Scope

Add RED/GREEN coverage and implementation for v3 MemoryFactGraph projections,
validators, report validation, correlation validation, and MiniMemoryModel
cases.

## Required Behaviors

- `interface_value_contains_borrow`
- `protocol_dispatch_borrow_conservative`
- `protocol_dispatch_noalias_conservative`
- `interface_borrow_escape_validator`
- `protocol_dispatch_borrow_validator`
- `protocol_dispatch_alias_conservative_validator`
- unknown/dynamic dispatch emits no trusted borrow facts
- broad noalias through protocol/interface dispatch rejected

## Acceptance

Accepted only after focused memoryfacts, memorymodel, and tools tests pass with
recorded command evidence.
