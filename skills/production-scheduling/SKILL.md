---
name: production-scheduling
description: Query teams and schedules, preview or apply automatic scheduling and moves, and diagnose the published capacity configuration through laps-cli. Use for 一键排产, 按今天排产, 待排产订单预览, category daily output, readiness policy, timeline, SVG, locking, and schedule CRUD.
---

# Production Scheduling

Use `laps-cli orders`, `laps-cli teams`, `laps-cli schedules`, `laps-cli auto-schedule`, and `laps-cli move`. Keep capacity writes in `$laps-capacity`; `capacity resolved` and `capacity list` are read-only diagnostics available to this workflow.

In WorkBuddy, use the `laps_scheduling` tool and its named preview, apply, move, lock, or schedule operation.

## Workflow

1. Resolve pending orders and exact team/schedule IDs with read commands. When the user requests every pending order, omit repeated `--order-id` flags and let the server select the pending scope.
2. In WorkBuddy, obtain every scheduling chart only by calling `auto-preview`, `move-order-preview`, `move-schedule-preview`, or `schedules-list` without specifying a display format or local output location. The connector uses the locally installed LAPS program to generate its standard HTML Gantt chart, with its time axis, and attaches that result to the conversation. If neither a capacity plan nor a date is supplied, omit both `--plan-id` and `--ref-date`: the system uses its local current date as the planning reference. Outside WorkBuddy, run `auto-schedule preview` in the default `json` format.
3. Keep capacity mode, same-product preference, unstarted-order replan, and readiness controls at `inherit` unless the user explicitly overrides them.
4. For WorkBuddy, do not run a second duplicate preview merely to obtain a chart: the first preview already includes the locally generated, horizontally scrollable Gantt chart. Never create, save, or attach an HTML file from order lists, material data, capacity summaries, or any hand-authored layout. A table, workload dashboard, or due-date bar chart is not a substitute for the official Gantt chart. If the conversation cannot show the attached chart, provide the business result and offer the official SVG file as the portable fallback.
5. On success, report dates, target teams, quantity splits, resolved capacity plan/mode, readiness warnings, unscheduled or late items, and tell the user that the attached Gantt chart is the preview. Never recompute dates or capacity locally.
6. If preview reports no published plan or another capacity configuration error, run `capacity resolved`, then `capacity list --resource plans` if needed. Report the exact server result; do not infer plan status from zero `dailyCapacity` or efficiency values in `teams list`.
7. Use `schedules lock --id ... --locked true|false` only after confirming the schedule ID and lock intent. Submit GUI-equivalent draft batches with `schedules apply --file JSON` only after confirmation.
8. Apply only after explicit user confirmation. Use the same `--plan-id` or `--ref-date` from the accepted preview so apply cannot resolve a different plan. Use exact IDs and require `--yes` for deletion.

The server owns scheduling, splitting, dates, conflicts, capacity, and readiness rules. `category_daily_output` and readiness `ignore|warn|block` are request controls, not calculations to reproduce locally.

## Business communication

- Describe schedules as teams, dates, quantities, delays, and unscheduled orders. Present preview results as “试排方案，尚未保存”.
- Before applying, moving, locking, or deleting, state the affected orders/schedule entries and obtain explicit confirmation.
- Explain conflicts or unavailable capacity as planning conditions and next actions, never as CLI or server errors unless the user asks for technical diagnosis.
