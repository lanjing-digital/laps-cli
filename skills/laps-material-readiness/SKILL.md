---
name: laps-material-readiness
description: Inspect and run material-readiness analysis through laps-cli. Use for readiness status, latest or persisted results, schema discovery, analysis, or importing external readiness results.
---

# LAPS Material Readiness

Use `laps-cli readiness status|latest|get|schema|analyze` and `laps-cli readiness external import`; use leaf `--help` for exact input.

In WorkBuddy, use the `laps_material_readiness` tool and select the requested analysis operation.

## Workflow

1. Check status/schema when the source contract is uncertain.
2. Use `latest` for current metadata or result and `get --analysis-id` for a persisted analysis.
3. Run analysis with explicit order IDs or a JSON file.
4. Import external results only from a trusted source and report the returned analysis ID.

Do not calculate readiness locally. Preserve server warnings, source, freshness, and per-material shortage details.

## Business communication

- Explain whether orders are material-ready, what is short, and how current the result is; do not surface protocol or raw failure text.
- Make clear whether the result is an existing analysis, a newly calculated analysis, or an imported external result.
- When the analysis cannot run, tell the user which order, material source, or required business data needs attention.
