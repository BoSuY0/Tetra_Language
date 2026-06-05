# Packet P2: Memoryfacts Design Review

## Objective

Read-only inspection of existing compiler-owned fact/proof/report paths to identify minimal integration points for `compiler/internal/memoryfacts`.

## Context

Current slice must add Memory Fact Graph v0 without rewriting the compiler. Reports must project compiler-owned facts rather than reconstruct truth.

## Files / Sources

Start with:

- `compiler/internal/plir/`
- `compiler/internal/allocplan/`
- `compiler/internal/validation/`
- `compiler/internal/rangeproof/`
- `compiler/internal/lower/`
- `compiler/reports.go`
- existing `compiler/internal/memoryfacts/` if present

## Ownership

Read-only. Do not edit files.

## Do

- Identify existing fact/proof IDs and report structures that can map to MemoryFactGraph v0.
- Identify minimal adapter points for safe borrowed views, `borrow`, `copy`, `copy_into`, `core.alloc_bytes`, raw unknown pointers, and storage lowering claims.
- Flag any current report path that appears to reconstruct truth outside compiler-owned facts.
- Cite concrete files/lines.

## Do Not

- Do not design a huge replacement architecture.
- Do not edit files.

## Expected Output

Markdown report with integration map, risks, recommended minimal v0 scope, files inspected, commands run, uncertainty.

## Verification

Read-only code inspection and focused `rg` probes only.
