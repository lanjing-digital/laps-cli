---
name: laps-orders
description: Manage production orders through laps-cli. Use to list, inspect, create, update, delete, export, download an import template, or preview and apply JSON or Excel order imports.
---

# LAPS Orders

Use `laps-cli orders ...`; consult `laps-cli orders --help` and leaf `--help` output for exact flags.

In WorkBuddy, use the `laps_orders` tool. Select an explicit business operation rather than passing command text.

## Rules

- Use exact IDs for get, update, and delete. Deletion requires `--yes`.
- Prefer `--set kebab-case=value` for simple records and `--file JSON` for complex input; never mix the two.
- For imports, run `orders import preview` first. Apply only after the user confirms.
- Import mode defaults to `create`; use `--mode upsert` only when matching by unique `sequenceNo` is intended.
- Use `--file -` for JSON stdin and the template command for Excel input.
- Report preview errors without applying; duplicate sequence numbers must be resolved at the source.

## Business communication

- Describe results as production orders, quantities, delivery dates, and affected records; do not mention CLI, HTTP, JSON, or raw errors.
- Before an import or change, state the intended order scope. Say “仅检查，尚未写入” for a preview and ask for confirmation before saving.
- Explain invalid or duplicate source data in terms of the affected order number and field, with a correction suggestion.
