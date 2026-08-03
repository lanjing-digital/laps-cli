---
name: laps-material-readiness
description: Inspect and run material-readiness analysis through laps-cli. Use for readiness status, latest or persisted results, schema discovery, analysis, or importing external readiness results.
---

# LAPS Material Readiness

Use `laps-cli readiness status|latest|get|schema|analyze` and `laps-cli readiness external import`; use leaf `--help` for exact input.

## Workflow

1. Check status/schema when the source contract is uncertain.
2. Use `latest` for current metadata or result and `get --analysis-id` for a persisted analysis.
3. Run analysis with explicit order IDs or a JSON file.
4. Import external results only from a trusted source and report the returned analysis ID.

Do not calculate readiness locally. Preserve server warnings, source, freshness, and per-material shortage details.
