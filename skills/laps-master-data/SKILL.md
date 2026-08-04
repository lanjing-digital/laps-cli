---
name: laps-master-data
description: Maintain factories, teams, batch factory settings, efficiencies, holiday calendars, holiday dates, and calendar bindings through laps-cli. Use for resource-tree and scheduling master-data operations.
---

# LAPS Master Data

Use `laps-cli resources ...`, `laps-cli efficiencies ...`, `laps-cli calendars ...`, and `laps-cli holidays ...`; rely on leaf `--help` for fields.

In WorkBuddy, use the `laps_master_data` tool and a named factory, team, efficiency, or calendar operation.

## Rules

- Represent resources as one factory-root/team-child tree and use `--file JSON` for tree apply.
- Use `resources batch-settings` for a single transactional update across exact factory IDs; direct flags and `--file JSON` are mutually exclusive.
- Keep virtual-line configuration inside `resources apply --file`; do not create a parallel capacity model.
- Use exact factory, team, or record IDs. Every deletion requires `--yes`.
- Use kebab-case `--set` flags for simple records; do not mix them with `--file`.
- Bind a team's calendar only with confirmed calendar and team IDs.
- Let the server enforce reference protection and preserve audit attribution to the OAuth user.

## Business communication

- Refer to factories, teams, calendars, holidays, and efficiencies in business terms, never tool names or raw technical output.
- Before changing or deleting records, summarize the affected factory/team/calendar and request confirmation.
- If a change is blocked, explain the business dependency that must be handled first and suggest the next action.
