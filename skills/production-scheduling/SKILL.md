---
name: production-scheduling
description: Query teams and schedule records, lock or batch-apply schedules, preview or apply automatic scheduling, and preview or apply order or schedule moves through laps-cli. Use for 一键排产, category daily output, readiness policy, unstarted-order replan, timeline, SVG, and schedule CRUD.
---

# Production Scheduling

Use `laps-cli teams`, `laps-cli schedules`, `laps-cli auto-schedule`, and `laps-cli move`. Use leaf `--help` for exact flags and keep capacity maintenance in `$laps-capacity`.

## Workflow

1. Resolve exact order, team, and schedule IDs with read commands.
2. Run `auto-schedule preview` or `move ... preview`; JSON is the default, while timeline/SVG are optional presentations.
3. Report dates, target teams, quantities, capacity mode, readiness warnings, and late items.
4. Use `schedules lock --id ... --locked true|false` only after confirming the schedule ID and lock intent. Submit GUI-equivalent draft batches with `schedules apply --file JSON` only after confirmation.
5. Apply only after explicit confirmation. Keep automatic controls at `inherit` unless the user explicitly selects same-product preference or unstarted-order replan. Use exact IDs and require `--yes` for schedule deletion.

The server owns scheduling, splitting, dates, conflicts, capacity, and readiness rules. `category_daily_output` and readiness `ignore|warn|block` are request controls, not calculations to reproduce locally.
