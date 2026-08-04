---
name: laps-material-master
description: Manage material records, BOM headers and items, inventory workbook imports, and import history through laps-cli. Use for material or BOM CRUD and previewed material-master imports.
---

# LAPS Material Master

Use `laps-cli materials ...`, `laps-cli boms ...`, and `laps-cli material-import ...`. Read leaf `--help` output for fields.

In WorkBuddy, use the `laps_material_master` tool and its named business operation.

## Rules

- Use flags for simple objects and `--file JSON` for BOM structures; do not mix business flags with `--file`.
- Use exact IDs and `--yes` for deletion. Do not bypass server reference protection.
- Download the workbook template, then run `material-import preview` before `apply`.
- Material imports update matching codes and do not delete existing rows omitted from the file.
- Apply only after the user confirms the preview counts, warnings, and scope.

## Business communication

- Report materials, BOMs, stock records, and affected counts in plain production language; hide implementation details and raw error messages.
- Clearly distinguish a workbook check from a saved material or BOM change, and obtain confirmation before saving.
- For data issues, identify the material code, BOM relation, or workbook column that the user should correct.
