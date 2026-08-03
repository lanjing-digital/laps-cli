---
name: laps-material-master
description: Manage material records, BOM headers and items, inventory workbook imports, and import history through laps-cli. Use for material or BOM CRUD and previewed material-master imports.
---

# LAPS Material Master

Use `laps-cli materials ...`, `laps-cli boms ...`, and `laps-cli material-import ...`. Read leaf `--help` output for fields.

## Rules

- Use flags for simple objects and `--file JSON` for BOM structures; do not mix business flags with `--file`.
- Use exact IDs and `--yes` for deletion. Do not bypass server reference protection.
- Download the workbook template, then run `material-import preview` before `apply`.
- Material imports update matching codes and do not delete existing rows omitted from the file.
- Apply only after the user confirms the preview counts, warnings, and scope.
