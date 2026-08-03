---
name: laps-capacity
description: Manage scheduling capacity through laps-cli. Use for resolved configuration, capacity CRUD, factory profiles, resource or category date capacity calendars, Excel or JSON import previews, validation, draft creation, and publishing.
---

# LAPS Capacity

Use `laps-cli capacity ...`; retrieve exact flags from the matching leaf `--help` output. Use `capacity calendar` for resource or category date overrides and the prefilled category-capacity Excel template.

## Workflow

1. Inspect `capacity resolved` or list/get the target resource.
2. For plan files, download the template and run `capacity import preview`; for date overrides use `capacity calendar category-import preview` before apply.
3. `capacity calendar range` and `category-range` write date overrides directly; confirm resource, date range, and zero-output stop-work intent first.
4. Apply creates a draft only; validate the draft separately. Publish only after explicit user confirmation because it changes automatic scheduling inputs.

Use exact IDs and `--yes` for deletion. Use `capacity calendar history` only when the OAuth user has audit permission. Keep factory profile operations separate from plan import. The server owns capacity calculations and validation.
