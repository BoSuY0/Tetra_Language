# P0 Discovery Packet

## Scope

Map v0/v1/v2 memory correlation patterns and the current interface/protocol
surface before implementation.

## Required Evidence

- Graphify MCP context for memoryfacts/protocol/interface relationships.
- Local reads of v0/v1/v2 correlation/final docs.
- Local reads of `compiler/internal/memoryfacts`, `compiler/internal/memorymodel`,
  report validators, correlation validator, and protocol/static conformance
  semantics tests.
- Exact recommended RED tests for P1/P2/P3.

## Acceptance

Accepted only if findings name concrete files/symbols and identify whether v3
can use existing static-conformance support or must stay conservative.
