---
name: production-scheduling
description: Query teams and schedules, preview or apply automatic scheduling and moves, and diagnose the published capacity configuration through laps-cli. Use for 一键排产, 按今天排产, 待排产订单预览, category daily output, readiness policy, timeline, SVG, locking, and schedule CRUD.
---

# Production Scheduling

Use `laps-cli orders`, `laps-cli teams`, `laps-cli schedules`, `laps-cli auto-schedule`, and `laps-cli move`. Keep capacity writes in `$laps-capacity`; `capacity resolved` and `capacity list` are read-only diagnostics available to this workflow.

## Workflow

1. Resolve pending orders and exact team/schedule IDs with read commands. When the user requests every pending order, omit repeated `--order-id` flags and let the server select the pending scope.
2. Run `auto-schedule preview` in the default `json` format. If neither a capacity plan nor a date is supplied, omit both `--plan-id` and `--ref-date`: the CLI sends its local current date as `planningReferenceDate`. Do not stop to ask for a plan ID and do not use help output as a substitute for the preview.
3. Keep capacity mode, same-product preference, unstarted-order replan, and readiness controls at `inherit` unless the user explicitly overrides them.
4. When the JSON preview succeeds, immediately run the identical `auto-schedule preview` again with `format: html` and no `--output`. This read-only call attaches the CLI-generated, horizontally scrollable Gantt chart to the conversation. Never hand-author SVG or HTML from the JSON result. If HTML rendering is unavailable, retry with `format: svg` as the portable fallback.
5. On success, report dates, target teams, quantity splits, resolved capacity plan/mode, readiness warnings, unscheduled or late items, and tell the user that the attached Gantt chart is the preview. Never recompute dates or capacity locally.
6. If preview reports no published plan or another capacity configuration error, run `capacity resolved`, then `capacity list --resource plans` if needed. Report the exact server result; do not infer plan status from zero `dailyCapacity` or efficiency values in `teams list`.
7. Use `schedules lock --id ... --locked true|false` only after confirming the schedule ID and lock intent. Submit GUI-equivalent draft batches with `schedules apply --file JSON` only after confirmation.
8. Apply only after explicit user confirmation. Use the same `--plan-id` or `--ref-date` from the accepted preview so apply cannot resolve a different plan. Use exact IDs and require `--yes` for deletion.

The server owns scheduling, splitting, dates, conflicts, capacity, and readiness rules. `category_daily_output` and readiness `ignore|warn|block` are request controls, not calculations to reproduce locally.
